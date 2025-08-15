package risk

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

// Manager manages risk limits and controls
type Manager struct {
	db          *sql.DB
	cache       cache.Cacher
	exchange    exchange.Exchange
	limits      map[string]*exchange.RiskLimit
	subscribers map[string][]chan *exchange.RiskLimit
	mu          sync.RWMutex
}

// NewManager creates a new risk manager
func NewManager(db *sql.DB, cache cache.Cacher, exchange exchange.Exchange) *Manager {
	m := &Manager{
		db:          db,
		cache:       cache,
		exchange:    exchange,
		limits:      make(map[string]*exchange.RiskLimit),
		subscribers: make(map[string][]chan *exchange.RiskLimit),
	}

	// Start risk monitor
	go m.monitorRiskLimits()

	return m
}

// Subscribe subscribes to risk limit updates for a symbol
func (m *Manager) Subscribe(symbol string) chan *exchange.RiskLimit {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan *exchange.RiskLimit, 100)
	m.subscribers[symbol] = append(m.subscribers[symbol], ch)
	return ch
}

// Unsubscribe removes a subscription
func (m *Manager) Unsubscribe(symbol string, ch chan *exchange.RiskLimit) {
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

// GetRiskLimits returns risk limits for a symbol
func (m *Manager) GetRiskLimits(ctx context.Context, symbol string) ([]*exchange.RiskLimit, error) {
	// Check cache first
	var limits []*exchange.RiskLimit
	err := m.cache.Get(ctx, fmt.Sprintf("risk_limits:%s", symbol), &limits)
	if err == nil {
		return limits, nil
	}

	// Get from exchange
	limits, err = m.exchange.GetRiskLimits(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get risk limits: %w", err)
	}

	// Cache the limits
	if err := m.cache.Set(ctx, fmt.Sprintf("risk_limits:%s", symbol), limits, time.Hour); err != nil {
		log.Printf("Failed to cache risk limits: %v", err)
	}

	// Update local cache and notify subscribers
	for _, limit := range limits {
		m.updateLimit(limit)
	}

	return limits, nil
}

// SetRiskLimits sets risk limits for a symbol
func (m *Manager) SetRiskLimits(ctx context.Context, symbol string, limits []*exchange.RiskLimit) error {
	// Set on exchange
	if err := m.exchange.SetRiskLimits(ctx, symbol, limits); err != nil {
		return fmt.Errorf("failed to set risk limits: %w", err)
	}

	// Store in database
	for _, limit := range limits {
		if err := m.storeLimit(limit); err != nil {
			log.Printf("Failed to store risk limit: %v", err)
		}
	}

	// Update cache
	if err := m.cache.Set(ctx, fmt.Sprintf("risk_limits:%s", symbol), limits, time.Hour); err != nil {
		log.Printf("Failed to cache risk limits: %v", err)
	}

	// Update local cache and notify subscribers
	for _, limit := range limits {
		m.updateLimit(limit)
	}

	return nil
}

// CheckRiskLimits checks if an order would violate risk limits
func (m *Manager) CheckRiskLimits(ctx context.Context, order *exchange.OrderRequest) error {
	// Get current position
	position, err := m.exchange.GetPosition(ctx, order.Symbol)
	if err != nil {
		return fmt.Errorf("failed to get position: %w", err)
	}

	// Get risk limits
	limits, err := m.GetRiskLimits(ctx, order.Symbol)
	if err != nil {
		return fmt.Errorf("failed to get risk limits: %w", err)
	}

	// Find applicable limit based on position size
	var limit *exchange.RiskLimit
	for _, l := range limits {
		if position == nil || position.Quantity+order.Quantity <= l.MaxPositionSize {
			limit = l
			break
		}
	}

	if limit == nil {
		return fmt.Errorf("no applicable risk limit found for position size")
	}

	// Check leverage
	if position != nil && position.Leverage > limit.Leverage {
		return fmt.Errorf("leverage exceeds limit: %d > %d", position.Leverage, limit.Leverage)
	}

	// Check position size
	if position != nil && position.Quantity+order.Quantity > limit.MaxPositionSize {
		return fmt.Errorf("position size would exceed limit: %.8f > %.8f",
			position.Quantity+order.Quantity, limit.MaxPositionSize)
	}

	// Check maintenance margin
	if position != nil {
		maintenanceMargin := (position.Quantity + order.Quantity) * position.MarkPrice * limit.MaintenanceMargin
		if maintenanceMargin > position.UnrealizedPnL {
			return fmt.Errorf("maintenance margin would be insufficient: %.8f > %.8f",
				maintenanceMargin, position.UnrealizedPnL)
		}
	}

	return nil
}

// storeLimit stores a risk limit in the database
func (m *Manager) storeLimit(limit *exchange.RiskLimit) error {
	query := `
		INSERT INTO risk_limits (
			symbol, leverage, max_position_size, maintenance_margin,
			initial_margin, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)
	`

	_, err := m.db.Exec(query,
		limit.Symbol,
		limit.Leverage,
		limit.MaxPositionSize,
		limit.MaintenanceMargin,
		limit.InitialMargin,
		limit.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to store risk limit: %w", err)
	}

	return nil
}

// updateLimit updates the local risk limit cache and notifies subscribers
func (m *Manager) updateLimit(limit *exchange.RiskLimit) {
	m.mu.Lock()
	m.limits[limit.Symbol] = limit
	m.mu.Unlock()

	// Notify subscribers
	m.notifySubscribers(limit)
}

// notifySubscribers notifies all subscribers of a risk limit update
func (m *Manager) notifySubscribers(limit *exchange.RiskLimit) {
	m.mu.RLock()
	subs := m.subscribers[limit.Symbol]
	m.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- limit:
		default:
			// Channel is full, skip
		}
	}
}

// monitorRiskLimits periodically updates risk limits
func (m *Manager) monitorRiskLimits() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.RLock()
		symbols := make([]string, 0, len(m.limits))
		for symbol := range m.limits {
			symbols = append(symbols, symbol)
		}
		m.mu.RUnlock()

		ctx := context.Background()
		for _, symbol := range symbols {
			limits, err := m.GetRiskLimits(ctx, symbol)
			if err != nil {
				log.Printf("Failed to update risk limits for %s: %v", symbol, err)
				continue
			}

			// Store limits in database
			for _, limit := range limits {
				if err := m.storeLimit(limit); err != nil {
					log.Printf("Failed to store risk limit: %v", err)
				}
			}
		}
	}
}
