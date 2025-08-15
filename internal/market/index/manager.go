package index

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"qcat/internal/cache"
)

// Manager manages index price data collection and processing
type Manager struct {
	db          *sql.DB
	cache       cache.Cacher
	stats       map[string]*Stats
	components  map[string][]Component
	subscribers map[string][]chan *Price
	mu          sync.RWMutex
}

// NewManager creates a new index price manager
func NewManager(db *sql.DB, cache cache.Cacher) *Manager {
	m := &Manager{
		db:          db,
		cache:       cache,
		stats:       make(map[string]*Stats),
		components:  make(map[string][]Component),
		subscribers: make(map[string][]chan *Price),
	}

	// Start stats updater
	go m.updateStats()

	return m
}

// Subscribe subscribes to index price updates for a symbol
func (m *Manager) Subscribe(symbol string) chan *Price {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan *Price, 100)
	m.subscribers[symbol] = append(m.subscribers[symbol], ch)
	return ch
}

// Unsubscribe removes a subscription
func (m *Manager) Unsubscribe(symbol string, ch chan *Price) {
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

// ProcessPrice processes a new index price
func (m *Manager) ProcessPrice(price *Price) error {
	// Store in database
	if err := m.storePrice(price); err != nil {
		return err
	}

	// Update stats
	m.updatePriceStats(price)

	// Notify subscribers
	m.notifySubscribers(price)

	return nil
}

// UpdateComponents updates the components of an index
func (m *Manager) UpdateComponents(symbol string, components []Component) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Store components
	m.components[symbol] = components

	// Calculate and update index price
	price := m.calculateIndexPrice(symbol)
	if price != nil {
		return m.ProcessPrice(price)
	}

	return nil
}

// GetCurrentPrice returns the current index price for a symbol
func (m *Manager) GetCurrentPrice(symbol string) *Price {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if stats, exists := m.stats[symbol]; exists {
		return &Price{
			Symbol:    symbol,
			Price:     stats.CurrentPrice,
			Timestamp: stats.UpdatedAt,
		}
	}
	return nil
}

// GetStats returns index price statistics for a symbol
func (m *Manager) GetStats(symbol string) *Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if stats, exists := m.stats[symbol]; exists {
		return stats
	}
	return nil
}

// GetComponents returns the components of an index
func (m *Manager) GetComponents(symbol string) []Component {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if components, exists := m.components[symbol]; exists {
		return components
	}
	return nil
}

// GetHistory returns historical index price data for a symbol
func (m *Manager) GetHistory(ctx context.Context, symbol string, start, end time.Time) ([]*History, error) {
	query := `
		SELECT symbol, price, timestamp
		FROM index_prices
		WHERE symbol = $1 AND timestamp BETWEEN $2 AND $3
		ORDER BY timestamp DESC
	`

	rows, err := m.db.QueryContext(ctx, query, symbol, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query index price history: %w", err)
	}
	defer rows.Close()

	var history []*History
	for rows.Next() {
		var h History
		if err := rows.Scan(&h.Symbol, &h.Price, &h.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan index price: %w", err)
		}
		history = append(history, &h)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating index prices: %w", err)
	}

	return history, nil
}

// storePrice stores an index price in the database
func (m *Manager) storePrice(price *Price) error {
	query := `
		INSERT INTO index_prices (symbol, price, timestamp)
		VALUES ($1, $2, $3)
	`

	_, err := m.db.Exec(query,
		price.Symbol,
		price.Price,
		price.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to store index price: %w", err)
	}

	return nil
}

// calculateIndexPrice calculates the index price from its components
func (m *Manager) calculateIndexPrice(symbol string) *Price {
	components, exists := m.components[symbol]
	if !exists || len(components) == 0 {
		return nil
	}

	var weightedSum float64
	var totalWeight float64
	var oldestTimestamp time.Time

	for _, comp := range components {
		weightedSum += comp.Price * comp.Weight
		totalWeight += comp.Weight
		if oldestTimestamp.IsZero() || comp.Timestamp.Before(oldestTimestamp) {
			oldestTimestamp = comp.Timestamp
		}
	}

	if totalWeight == 0 {
		return nil
	}

	return &Price{
		Symbol:    symbol,
		Price:     weightedSum / totalWeight,
		Timestamp: oldestTimestamp,
	}
}

// updatePriceStats updates index price statistics
func (m *Manager) updatePriceStats(price *Price) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stats, exists := m.stats[price.Symbol]
	if !exists {
		stats = &Stats{
			Symbol:       price.Symbol,
			CurrentPrice: price.Price,
			UpdatedAt:    price.Timestamp,
		}
		m.stats[price.Symbol] = stats
	} else {
		stats.CurrentPrice = price.Price
		stats.UpdatedAt = price.Timestamp
	}
}

// updateStats periodically updates index price statistics
func (m *Manager) updateStats() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		for symbol := range m.stats {
			if err := m.calculate24hStats(symbol); err != nil {
				log.Printf("Error calculating 24h stats for %s: %v", symbol, err)
			}
		}
		m.mu.Unlock()
	}
}

// calculate24hStats calculates 24-hour statistics for a symbol
func (m *Manager) calculate24hStats(symbol string) error {
	ctx := context.Background()
	end := time.Now()
	start := end.Add(-24 * time.Hour)

	history, err := m.GetHistory(ctx, symbol, start, end)
	if err != nil {
		return err
	}

	if len(history) == 0 {
		return nil
	}

	stats := m.stats[symbol]

	// Calculate mean
	var sum float64
	for _, h := range history {
		sum += h.Price
	}
	mean := sum / float64(len(history))
	stats.Mean24h = mean

	// Calculate standard deviation
	var sumSquares float64
	for _, h := range history {
		diff := h.Price - mean
		sumSquares += diff * diff
	}
	stats.StdDev24h = math.Sqrt(sumSquares / float64(len(history)))

	// Find high and low
	stats.High24h = history[0].Price
	stats.Low24h = history[0].Price
	for _, h := range history {
		if h.Price > stats.High24h {
			stats.High24h = h.Price
		}
		if h.Price < stats.Low24h {
			stats.Low24h = h.Price
		}
	}

	// Calculate 24h change
	if len(history) > 1 {
		oldestPrice := history[len(history)-1].Price
		stats.Change24h = stats.CurrentPrice - oldestPrice
		if oldestPrice > 0 {
			stats.ChangeP24h = (stats.Change24h / oldestPrice) * 100
		}
	}

	return nil
}

// notifySubscribers notifies all subscribers of a new index price
func (m *Manager) notifySubscribers(price *Price) {
	m.mu.RLock()
	subs := m.subscribers[price.Symbol]
	m.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- price:
		default:
			// Channel is full, skip
		}
	}
}
