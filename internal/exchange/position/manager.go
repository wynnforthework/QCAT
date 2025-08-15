package position

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/cache"
	"qcat/internal/exchange"
)

// Manager manages position information
type Manager struct {
	db          *sql.DB
	cache       cache.Cacher
	exchange    exchange.Exchange
	positions   map[string]*exchange.Position
	subscribers map[string][]chan *exchange.Position
	mu          sync.RWMutex
}

// NewManager creates a new position manager
func NewManager(db *sql.DB, cache cache.Cacher, exchange exchange.Exchange) *Manager {
	m := &Manager{
		db:          db,
		cache:       cache,
		exchange:    exchange,
		positions:   make(map[string]*exchange.Position),
		subscribers: make(map[string][]chan *exchange.Position),
	}

	// Start position monitor
	go m.monitorPositions()

	return m
}

// Subscribe subscribes to position updates for a symbol
func (m *Manager) Subscribe(symbol string) chan *exchange.Position {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan *exchange.Position, 100)
	m.subscribers[symbol] = append(m.subscribers[symbol], ch)
	return ch
}

// Unsubscribe removes a subscription
func (m *Manager) Unsubscribe(symbol string, ch chan *exchange.Position) {
	m.mu.Lock()
	defer m.mu.Unlock()

	subs := m.subscribers[symbol]
	for i, sub := range subs {
		if sub == ch {
			m.subscribers[symbol] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}
}

// GetPosition returns the current position for a symbol
func (m *Manager) GetPosition(ctx context.Context, symbol string) (*exchange.Position, error) {
	// Check cache first
	var position exchange.Position
	err := m.cache.Get(ctx, fmt.Sprintf("position:%s", symbol), &position)
	if err == nil {
		return &position, nil
	}

	// Get from exchange
	position, err := m.exchange.GetPosition(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get position: %w", err)
	}

	// Cache the position
	if err := m.cache.Set(ctx, fmt.Sprintf("position:%s", symbol), position, time.Minute); err != nil {
		log.Printf("Failed to cache position: %v", err)
	}

	// Update local cache and notify subscribers
	m.updatePosition(&position)

	return &position, nil
}

// GetAllPositions returns all open positions
func (m *Manager) GetAllPositions(ctx context.Context) ([]*exchange.Position, error) {
	// Get from exchange
	positions, err := m.exchange.GetPositions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get positions: %w", err)
	}

	// Update cache and notify subscribers
	for _, position := range positions {
		// Cache the position
		if err := m.cache.Set(ctx, fmt.Sprintf("position:%s", position.Symbol), position, time.Minute); err != nil {
			log.Printf("Failed to cache position: %v", err)
		}

		// Update local cache
		m.updatePosition(position)
	}

	return positions, nil
}

// GetPositionHistory returns historical position data
func (m *Manager) GetPositionHistory(ctx context.Context, symbol string, startTime, endTime time.Time) ([]*exchange.Position, error) {
	query := `
		SELECT symbol, side, quantity, entry_price, mark_price,
			   liq_price, leverage, margin_type, unrealized_pnl,
			   realized_pnl, updated_at
		FROM positions
		WHERE symbol = $1 AND updated_at BETWEEN $2 AND $3
		ORDER BY updated_at DESC
	`

	rows, err := m.db.QueryContext(ctx, query, symbol, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query position history: %w", err)
	}
	defer rows.Close()

	var history []*exchange.Position
	for rows.Next() {
		var position exchange.Position
		if err := rows.Scan(
			&position.Symbol,
			&position.Side,
			&position.Quantity,
			&position.EntryPrice,
			&position.MarkPrice,
			&position.LiqPrice,
			&position.Leverage,
			&position.MarginType,
			&position.UnrealizedPnL,
			&position.RealizedPnL,
			&position.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan position: %w", err)
		}
		history = append(history, &position)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating positions: %w", err)
	}

	return history, nil
}

// SetLeverage sets the leverage for a symbol
func (m *Manager) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	// Set leverage on exchange
	if err := m.exchange.SetLeverage(ctx, symbol, leverage); err != nil {
		return fmt.Errorf("failed to set leverage: %w", err)
	}

	// Get updated position
	position, err := m.GetPosition(ctx, symbol)
	if err != nil {
		return fmt.Errorf("failed to get updated position: %w", err)
	}

	// Store position in database
	if err := m.storePosition(position); err != nil {
		log.Printf("Failed to store position: %v", err)
	}

	return nil
}

// SetMarginType sets the margin type for a symbol
func (m *Manager) SetMarginType(ctx context.Context, symbol string, marginType exchange.MarginType) error {
	// Set margin type on exchange
	if err := m.exchange.SetMarginType(ctx, symbol, marginType); err != nil {
		return fmt.Errorf("failed to set margin type: %w", err)
	}

	// Get updated position
	position, err := m.GetPosition(ctx, symbol)
	if err != nil {
		return fmt.Errorf("failed to get updated position: %w", err)
	}

	// Store position in database
	if err := m.storePosition(position); err != nil {
		log.Printf("Failed to store position: %v", err)
	}

	return nil
}

// storePosition stores a position in the database
func (m *Manager) storePosition(position *exchange.Position) error {
	query := `
		INSERT INTO positions (
			symbol, side, quantity, entry_price, mark_price,
			liq_price, leverage, margin_type, unrealized_pnl,
			realized_pnl, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`

	_, err := m.db.Exec(query,
		position.Symbol,
		position.Side,
		position.Quantity,
		position.EntryPrice,
		position.MarkPrice,
		position.LiqPrice,
		position.Leverage,
		position.MarginType,
		position.UnrealizedPnL,
		position.RealizedPnL,
		position.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to store position: %w", err)
	}

	return nil
}

// updatePosition updates the local position cache and notifies subscribers
func (m *Manager) updatePosition(position *exchange.Position) {
	m.mu.Lock()
	m.positions[position.Symbol] = position
	m.mu.Unlock()

	// Notify subscribers
	m.notifySubscribers(position)
}

// notifySubscribers notifies all subscribers of a position update
func (m *Manager) notifySubscribers(position *exchange.Position) {
	m.mu.RLock()
	subs := m.subscribers[position.Symbol]
	m.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- position:
		default:
			// Channel is full, skip
		}
	}
}

// monitorPositions periodically updates position information
func (m *Manager) monitorPositions() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		positions, err := m.GetAllPositions(ctx)
		if err != nil {
			log.Printf("Failed to update positions: %v", err)
			continue
		}

		// Store positions in database
		for _, position := range positions {
			if err := m.storePosition(position); err != nil {
				log.Printf("Failed to store position: %v", err)
			}
		}
	}
}
