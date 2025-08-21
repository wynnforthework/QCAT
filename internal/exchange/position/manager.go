package position

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/cache"
	exch "qcat/internal/exchange"
)

// Manager manages position information
type Manager struct {
	db          *sql.DB
	cache       cache.Cacher
	exchange    exch.Exchange
	positions   map[string]*exch.Position
	subscribers map[string][]chan *exch.Position
	mu          sync.RWMutex
}

// NewManager creates a new position manager
func NewManager(db *sql.DB, cache cache.Cacher, exchange exch.Exchange) *Manager {
	m := &Manager{
		db:          db,
		cache:       cache,
		exchange:    exchange,
		positions:   make(map[string]*exch.Position),
		subscribers: make(map[string][]chan *exch.Position),
	}

	// Start position monitor only if exchange is available
	if exchange != nil {
		go m.monitorPositions()
	} else {
		log.Printf("Warning: Exchange not available, position monitoring disabled")
	}

	return m
}

// Subscribe subscribes to position updates for a symbol
func (m *Manager) Subscribe(symbol string) chan *exch.Position {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan *exch.Position, 100)
	m.subscribers[symbol] = append(m.subscribers[symbol], ch)
	return ch
}

// Unsubscribe removes a subscription
func (m *Manager) Unsubscribe(symbol string, ch chan *exch.Position) {
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
func (m *Manager) GetPosition(ctx context.Context, symbol string) (*exch.Position, error) {
	// Check cache first
	var position exch.Position
	err := m.cache.Get(ctx, fmt.Sprintf("position:%s", symbol), &position)
	if err == nil {
		return &position, nil
	}

	// Get from exchange
	positionPtr, err := m.exchange.GetPosition(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get position: %w", err)
	}

	// Cache the position
	if err := m.cache.Set(ctx, fmt.Sprintf("position:%s", symbol), positionPtr, time.Minute); err != nil {
		log.Printf("Failed to cache position: %v", err)
	}

	// Update local cache and notify subscribers
	m.updatePosition(positionPtr)

	return positionPtr, nil
}

// GetAllPositions returns all open positions
func (m *Manager) GetAllPositions(ctx context.Context) ([]*exch.Position, error) {
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
func (m *Manager) GetPositionHistory(ctx context.Context, symbol string, startTime, endTime time.Time) ([]*exch.Position, error) {
	query := `
		SELECT symbol, side, size, entry_price, leverage,
			   unrealized_pnl, realized_pnl, status, updated_at
		FROM positions
		WHERE symbol = $1 AND updated_at BETWEEN $2 AND $3
		ORDER BY updated_at DESC
	`

	rows, err := m.db.QueryContext(ctx, query, symbol, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query position history: %w", err)
	}
	defer rows.Close()

	var history []*exch.Position
	for rows.Next() {
		var position exch.Position
		var status string
		if err := rows.Scan(
			&position.Symbol,
			&position.Side,
			&position.Size, // Use Size instead of Quantity
			&position.EntryPrice,
			&position.Leverage,
			&position.UnrealizedPnL,
			&position.RealizedPnL,
			&status,
			&position.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan position: %w", err)
		}

		// Set Quantity field for compatibility (since Position struct has both Size and Quantity)
		position.Quantity = position.Size

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
func (m *Manager) SetMarginType(ctx context.Context, symbol string, marginType exch.MarginType) error {
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
func (m *Manager) storePosition(position *exch.Position) error {
	return m.storePositionWithStrategy(position, "")
}

// storePositionWithStrategy stores a position in the database with a specific strategy ID
func (m *Manager) storePositionWithStrategy(position *exch.Position, strategyID string) error {
	// If no strategy ID provided, try to get or create a default one
	if strategyID == "" {
		defaultStrategyID, err := m.getOrCreateDefaultStrategy()
		if err != nil {
			log.Printf("Warning: failed to get default strategy, using NULL: %v", err)
			// Use NULL for strategy_id if we can't get a default strategy
			return m.storePositionWithNullStrategy(position)
		}
		strategyID = defaultStrategyID
	}

	query := `
		INSERT INTO positions (
			strategy_id, symbol, side, size, entry_price, leverage,
			unrealized_pnl, realized_pnl, status, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
	`

	// Use Size field instead of Quantity, and set a default status
	status := "open"
	if position.Size == 0 {
		status = "closed"
	}

	_, err := m.db.Exec(query,
		strategyID,
		position.Symbol,
		position.Side,
		position.Size, // Use Size instead of Quantity
		position.EntryPrice,
		position.Leverage,
		position.UnrealizedPnL,
		position.RealizedPnL,
		status,
		position.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to store position: %w", err)
	}

	return nil
}

// getOrCreateDefaultStrategy gets or creates a default strategy for positions without strategy association
func (m *Manager) getOrCreateDefaultStrategy() (string, error) {
	// First try to get existing default strategy
	var strategyID string
	query := `SELECT id FROM strategies WHERE name = 'Default Position Strategy' LIMIT 1`
	err := m.db.QueryRow(query).Scan(&strategyID)
	if err == nil {
		return strategyID, nil
	}

	// If not found, create a new default strategy
	createQuery := `
		INSERT INTO strategies (id, name, type, status, description, created_at, updated_at)
		VALUES (uuid_generate_v4(), 'Default Position Strategy', 'manual', 'active',
				'Default strategy for positions without specific strategy association',
				CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`

	err = m.db.QueryRow(createQuery).Scan(&strategyID)
	if err != nil {
		return "", fmt.Errorf("failed to create default strategy: %w", err)
	}

	log.Printf("Created default strategy with ID: %s", strategyID)
	return strategyID, nil
}

// storePositionWithNullStrategy stores position with NULL strategy_id (fallback method)
func (m *Manager) storePositionWithNullStrategy(position *exch.Position) error {
	// First, check if we can modify the schema to allow NULL strategy_id temporarily
	// This is a fallback method when we can't create a default strategy

	// Try to insert with a placeholder strategy_id
	query := `
		INSERT INTO positions (
			strategy_id, symbol, side, size, entry_price, leverage,
			unrealized_pnl, realized_pnl, status, updated_at
		) VALUES (
			(SELECT id FROM strategies LIMIT 1), $1, $2, $3, $4, $5, $6, $7, $8, $9
		)
	`

	status := "open"
	if position.Size == 0 {
		status = "closed"
	}

	_, err := m.db.Exec(query,
		position.Symbol,
		position.Side,
		position.Size,
		position.EntryPrice,
		position.Leverage,
		position.UnrealizedPnL,
		position.RealizedPnL,
		status,
		position.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to store position with fallback strategy: %w", err)
	}

	return nil
}

// updatePosition updates the local position cache and notifies subscribers
func (m *Manager) updatePosition(position *exch.Position) {
	m.mu.Lock()
	m.positions[position.Symbol] = position
	m.mu.Unlock()

	// Notify subscribers
	m.notifySubscribers(position)
}

// notifySubscribers notifies all subscribers of a position update
func (m *Manager) notifySubscribers(position *exch.Position) {
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

	consecutiveErrors := 0
	maxConsecutiveErrors := 5

	for range ticker.C {
		ctx := context.Background()
		positions, err := m.GetAllPositions(ctx)
		if err != nil {
			consecutiveErrors++
			if consecutiveErrors <= maxConsecutiveErrors {
				log.Printf("Failed to update positions: %v", err)
			} else if consecutiveErrors == maxConsecutiveErrors+1 {
				log.Printf("Position monitoring disabled after %d consecutive errors. Last error: %v", maxConsecutiveErrors, err)
			}
			// Stop logging after max errors to prevent spam
			continue
		}

		// Reset error counter on success
		if consecutiveErrors > 0 {
			log.Printf("Position monitoring recovered after %d errors", consecutiveErrors)
			consecutiveErrors = 0
		}

		// Store positions in database
		for _, position := range positions {
			if err := m.storePosition(position); err != nil {
				log.Printf("Failed to store position: %v", err)
			}
		}
	}
}
