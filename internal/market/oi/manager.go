package oi

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

// Manager manages open interest data collection and processing
type Manager struct {
	db          *sql.DB
	cache       cache.Cacher
	stats       map[string]*Stats
	subscribers map[string][]chan *OpenInterest
	mu          sync.RWMutex
}

// NewManager creates a new open interest manager
func NewManager(db *sql.DB, cache cache.Cacher) *Manager {
	m := &Manager{
		db:          db,
		cache:       cache,
		stats:       make(map[string]*Stats),
		subscribers: make(map[string][]chan *OpenInterest),
	}

	// Start stats updater
	go m.updateStats()

	return m
}

// Subscribe subscribes to open interest updates for a symbol
func (m *Manager) Subscribe(symbol string) chan *OpenInterest {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan *OpenInterest, 100)
	m.subscribers[symbol] = append(m.subscribers[symbol], ch)
	return ch
}

// Unsubscribe removes a subscription
func (m *Manager) Unsubscribe(symbol string, ch chan *OpenInterest) {
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

// ProcessOI processes new open interest data
func (m *Manager) ProcessOI(oi *OpenInterest) error {
	// Store in database
	if err := m.storeOI(oi); err != nil {
		return err
	}

	// Update stats
	m.updateOIStats(oi)

	// Notify subscribers
	m.notifySubscribers(oi)

	return nil
}

// GetCurrentOI returns the current open interest for a symbol
func (m *Manager) GetCurrentOI(symbol string) *OpenInterest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if stats, exists := m.stats[symbol]; exists {
		return &OpenInterest{
			Symbol:    symbol,
			Value:     stats.CurrentOI,
			Notional:  stats.CurrentNotional,
			Timestamp: stats.UpdatedAt,
		}
	}
	return nil
}

// GetStats returns open interest statistics for a symbol
func (m *Manager) GetStats(symbol string) *Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if stats, exists := m.stats[symbol]; exists {
		return stats
	}
	return nil
}

// GetHistory returns historical open interest data for a symbol
func (m *Manager) GetHistory(ctx context.Context, symbol string, start, end time.Time) ([]*History, error) {
	query := `
		SELECT symbol, value, notional, timestamp
		FROM open_interest
		WHERE symbol = $1 AND timestamp BETWEEN $2 AND $3
		ORDER BY timestamp DESC
	`

	rows, err := m.db.QueryContext(ctx, query, symbol, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query open interest history: %w", err)
	}
	defer rows.Close()

	var history []*History
	for rows.Next() {
		var h History
		if err := rows.Scan(&h.Symbol, &h.Value, &h.Notional, &h.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan open interest: %w", err)
		}
		history = append(history, &h)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating open interest: %w", err)
	}

	return history, nil
}

// storeOI stores open interest data in the database
func (m *Manager) storeOI(oi *OpenInterest) error {
	query := `
		INSERT INTO open_interest (symbol, value, notional, timestamp)
		VALUES ($1, $2, $3, $4)
	`

	_, err := m.db.Exec(query,
		oi.Symbol,
		oi.Value,
		oi.Notional,
		oi.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to store open interest: %w", err)
	}

	return nil
}

// updateOIStats updates open interest statistics
func (m *Manager) updateOIStats(oi *OpenInterest) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stats, exists := m.stats[oi.Symbol]
	if !exists {
		stats = &Stats{
			Symbol:          oi.Symbol,
			CurrentOI:       oi.Value,
			CurrentNotional: oi.Notional,
			UpdatedAt:       oi.Timestamp,
		}
		m.stats[oi.Symbol] = stats
	} else {
		stats.CurrentOI = oi.Value
		stats.CurrentNotional = oi.Notional
		stats.UpdatedAt = oi.Timestamp
	}
}

// updateStats periodically updates open interest statistics
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
		sum += h.Value
	}
	mean := sum / float64(len(history))
	stats.Mean24h = mean

	// Calculate standard deviation
	var sumSquares float64
	for _, h := range history {
		diff := h.Value - mean
		sumSquares += diff * diff
	}
	stats.StdDev24h = math.Sqrt(sumSquares / float64(len(history)))

	// Find high and low
	stats.High24h = history[0].Value
	stats.Low24h = history[0].Value
	for _, h := range history {
		if h.Value > stats.High24h {
			stats.High24h = h.Value
		}
		if h.Value < stats.Low24h {
			stats.Low24h = h.Value
		}
	}

	// Calculate 24h change
	if len(history) > 1 {
		oldestOI := history[len(history)-1].Value
		stats.Change24h = stats.CurrentOI - oldestOI
		if oldestOI > 0 {
			stats.ChangeP24h = (stats.Change24h / oldestOI) * 100
		}
	}

	return nil
}

// notifySubscribers notifies all subscribers of new open interest data
func (m *Manager) notifySubscribers(oi *OpenInterest) {
	m.mu.RLock()
	subs := m.subscribers[oi.Symbol]
	m.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- oi:
		default:
			// Channel is full, skip
		}
	}
}
