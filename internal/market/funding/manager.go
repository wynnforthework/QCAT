package funding

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

// Manager manages funding rate data collection and processing
type Manager struct {
	db          *sql.DB
	cache       cache.Cacher
	stats       map[string]*Stats
	subscribers map[string][]chan *Rate
	mu          sync.RWMutex
}

// NewManager creates a new funding rate manager
func NewManager(db *sql.DB, cache cache.Cacher) *Manager {
	m := &Manager{
		db:          db,
		cache:       cache,
		stats:       make(map[string]*Stats),
		subscribers: make(map[string][]chan *Rate),
	}

	// Start stats updater
	go m.updateStats()

	return m
}

// Subscribe subscribes to funding rate updates for a symbol
func (m *Manager) Subscribe(symbol string) chan *Rate {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan *Rate, 100)
	m.subscribers[symbol] = append(m.subscribers[symbol], ch)
	return ch
}

// Unsubscribe removes a subscription
func (m *Manager) Unsubscribe(symbol string, ch chan *Rate) {
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

// ProcessRate processes a new funding rate
func (m *Manager) ProcessRate(rate *Rate) error {
	// Store in database
	if err := m.storeRate(rate); err != nil {
		return err
	}

	// Cache the rate
	if err := m.cache.SetFundingRate(context.Background(), rate.Symbol, rate, 8*time.Hour); err != nil {
		return fmt.Errorf("failed to cache funding rate: %w", err)
	}

	// Update stats
	m.updateRateStats(rate)

	// Notify subscribers
	m.notifySubscribers(rate)

	return nil
}

// GetCurrentRate returns the current funding rate for a symbol
func (m *Manager) GetCurrentRate(symbol string) (*Rate, error) {
	var rate Rate
	err := m.cache.GetFundingRate(context.Background(), symbol, &rate)
	if err != nil {
		return nil, fmt.Errorf("failed to get funding rate from cache: %w", err)
	}
	return &rate, nil
}

// GetStats returns funding rate statistics for a symbol
func (m *Manager) GetStats(symbol string) *Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if stats, exists := m.stats[symbol]; exists {
		return stats
	}
	return nil
}

// GetHistory returns historical funding rates for a symbol
func (m *Manager) GetHistory(ctx context.Context, symbol string, start, end time.Time) ([]*Rate, error) {
	query := `
		SELECT symbol, rate, next_rate, next_time, last_updated
		FROM funding_rates
		WHERE symbol = $1 AND last_updated BETWEEN $2 AND $3
		ORDER BY last_updated DESC
	`

	rows, err := m.db.QueryContext(ctx, query, symbol, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query funding rate history: %w", err)
	}
	defer rows.Close()

	var history []*Rate
	for rows.Next() {
		var h Rate
		if err := rows.Scan(&h.Symbol, &h.Rate, &h.NextRate, &h.NextTime, &h.LastUpdated); err != nil {
			return nil, fmt.Errorf("failed to scan funding rate: %w", err)
		}
		history = append(history, &h)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating funding rates: %w", err)
	}

	return history, nil
}

// storeRate stores a funding rate in the database
func (m *Manager) storeRate(rate *Rate) error {
	query := `
		INSERT INTO funding_rates (symbol, rate, next_rate, next_time, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := m.db.Exec(query,
		rate.Symbol,
		rate.Rate,
		rate.NextRate,
		rate.NextTime,
		rate.LastUpdated,
	)
	if err != nil {
		return fmt.Errorf("failed to store funding rate: %w", err)
	}

	return nil
}

// updateRateStats updates funding rate statistics
func (m *Manager) updateRateStats(rate *Rate) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stats, exists := m.stats[rate.Symbol]
	if !exists {
		stats = &Stats{
			Symbol:          rate.Symbol,
			CurrentRate:     rate.Rate,
			PredictedRate:   rate.NextRate,
			NextFundingTime: rate.NextTime,
			UpdatedAt:       rate.LastUpdated,
		}
		m.stats[rate.Symbol] = stats
	} else {
		stats.CurrentRate = rate.Rate
		stats.PredictedRate = rate.NextRate
		stats.NextFundingTime = rate.NextTime
		stats.UpdatedAt = rate.LastUpdated
	}
}

// updateStats periodically updates funding rate statistics
func (m *Manager) updateStats() {
	ticker := time.NewTicker(time.Hour)
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
		sum += h.Rate
	}
	mean := sum / float64(len(history))
	stats.Mean24h = mean

	// Calculate standard deviation
	var sumSquares float64
	for _, h := range history {
		diff := h.Rate - mean
		sumSquares += diff * diff
	}
	stats.StdDev24h = math.Sqrt(sumSquares / float64(len(history)))

	// Find min and max
	stats.Min24h = history[0].Rate
	stats.Max24h = history[0].Rate
	for _, h := range history {
		if h.Rate < stats.Min24h {
			stats.Min24h = h.Rate
		}
		if h.Rate > stats.Max24h {
			stats.Max24h = h.Rate
		}
	}

	// Calculate annualized rate (assuming 8-hour funding intervals)
	stats.AnnualizedRate = stats.Mean24h * 3 * 365 * 100 // Convert to percentage

	return nil
}

// notifySubscribers notifies all subscribers of a new funding rate
func (m *Manager) notifySubscribers(rate *Rate) {
	m.mu.RLock()
	subs := m.subscribers[rate.Symbol]
	m.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- rate:
		default:
			// Channel is full, skip
		}
	}
}
