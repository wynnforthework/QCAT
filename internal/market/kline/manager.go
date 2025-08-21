package kline

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"
)

// Manager manages kline data for multiple symbols and intervals
type Manager struct {
	db           *sql.DB
	klines       map[string]map[Interval]*Kline
	subscribers  map[string][]chan *Kline
	mu           sync.RWMutex
	batchSize    int
	batchTimeout time.Duration
	batchBuffer  []*Kline
	bufferMu     sync.Mutex
}

// NewManager creates a new kline manager
func NewManager(db *sql.DB) *Manager {
	m := &Manager{
		db:           db,
		klines:       make(map[string]map[Interval]*Kline),
		subscribers:  make(map[string][]chan *Kline),
		batchSize:    100,
		batchTimeout: 5 * time.Second,
		batchBuffer:  make([]*Kline, 0, 100),
	}

	// Start batch processor
	go m.processBatch()

	return m
}

// Subscribe subscribes to kline updates for a symbol and interval
func (m *Manager) Subscribe(symbol string, interval Interval) chan *Kline {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan *Kline, 100)
	key := fmt.Sprintf("%s-%s", symbol, interval)
	m.subscribers[key] = append(m.subscribers[key], ch)
	return ch
}

// Unsubscribe removes a subscription
func (m *Manager) Unsubscribe(symbol string, interval Interval, ch chan *Kline) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s-%s", symbol, interval)
	subs := m.subscribers[key]
	for i, sub := range subs {
		if sub == ch {
			m.subscribers[key] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}
}

// UpdateTrade updates klines with a new trade
func (m *Manager) UpdateTrade(symbol string, price, volume float64, timestamp time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Initialize symbol map if not exists
	if _, exists := m.klines[symbol]; !exists {
		m.klines[symbol] = make(map[Interval]*Kline)
	}

	// Update all intervals
	intervals := []Interval{
		Interval1m, Interval3m, Interval5m, Interval15m, Interval30m,
		Interval1h, Interval2h, Interval4h, Interval6h, Interval8h,
		Interval12h, Interval1d, Interval3d, Interval1w, Interval1M,
	}

	for _, interval := range intervals {
		kline := m.klines[symbol][interval]
		if kline == nil || timestamp.After(kline.CloseTime) {
			// Store completed kline
			if kline != nil && kline.Complete {
				if err := m.storeBatch(kline); err != nil {
					return fmt.Errorf("failed to store kline: %w", err)
				}
			}

			// Create new kline
			openTime := timestamp.Truncate(GetIntervalDuration(interval))
			kline = NewKline(symbol, interval, openTime)
			m.klines[symbol][interval] = kline
		}

		// Update kline
		kline.Update(price, volume, timestamp)

		// Notify subscribers
		key := fmt.Sprintf("%s-%s", symbol, interval)
		for _, ch := range m.subscribers[key] {
			select {
			case ch <- kline:
			default:
				// Channel is full, skip
			}
		}
	}

	return nil
}

// GetKline returns the current kline for a symbol and interval
func (m *Manager) GetKline(symbol string, interval Interval) *Kline {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if symbolKlines, exists := m.klines[symbol]; exists {
		return symbolKlines[interval]
	}
	return nil
}

// LoadHistoricalKlines loads historical klines from the database
func (m *Manager) LoadHistoricalKlines(ctx context.Context, symbol string, interval Interval, start, end time.Time) ([]*Kline, error) {
	query := `
		SELECT symbol, interval, timestamp, open, high, low, close, volume
		FROM market_data
		WHERE symbol = $1 AND interval = $2 AND timestamp BETWEEN $3 AND $4
		ORDER BY timestamp ASC
	`

	rows, err := m.db.QueryContext(ctx, query, symbol, interval, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query historical klines: %w", err)
	}
	defer rows.Close()

	var klines []*Kline
	for rows.Next() {
		var k Kline
		var timestamp time.Time
		if err := rows.Scan(
			&k.Symbol,
			&k.Interval,
			&timestamp,
			&k.Open,
			&k.High,
			&k.Low,
			&k.Close,
			&k.Volume,
		); err != nil {
			return nil, fmt.Errorf("failed to scan kline: %w", err)
		}

		k.OpenTime = timestamp
		k.CloseTime = getCloseTime(timestamp, k.Interval)
		k.Complete = true
		klines = append(klines, &k)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating klines: %w", err)
	}

	return klines, nil
}

// GetHistory returns historical klines for a symbol within a time range
func (m *Manager) GetHistory(ctx context.Context, symbol string, start, end time.Time) ([]*Kline, error) {
	// Use 1-hour interval as default for historical data
	return m.LoadHistoricalKlines(ctx, symbol, Interval1h, start, end)
}

// storeBatch adds a kline to the batch buffer
func (m *Manager) storeBatch(kline *Kline) error {
	m.bufferMu.Lock()
	m.batchBuffer = append(m.batchBuffer, kline)
	m.bufferMu.Unlock()

	return nil
}

// processBatch processes the batch buffer periodically
func (m *Manager) processBatch() {
	ticker := time.NewTicker(m.batchTimeout)
	defer ticker.Stop()

	for range ticker.C {
		m.bufferMu.Lock()
		if len(m.batchBuffer) == 0 {
			m.bufferMu.Unlock()
			continue
		}

		// Copy buffer and reset
		buffer := make([]*Kline, len(m.batchBuffer))
		copy(buffer, m.batchBuffer)
		m.batchBuffer = m.batchBuffer[:0]
		m.bufferMu.Unlock()

		// Store klines in database
		if err := m.storeKlines(buffer); err != nil {
			log.Printf("Error storing klines: %v", err)
		}
	}
}

// storeKlines stores multiple klines in the database
func (m *Manager) storeKlines(klines []*Kline) error {
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO market_data (
			symbol, interval, timestamp, open, high, low, close, volume, complete
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		) ON CONFLICT (symbol, timestamp, interval) DO UPDATE SET
			open = EXCLUDED.open,
			high = EXCLUDED.high,
			low = EXCLUDED.low,
			close = EXCLUDED.close,
			volume = EXCLUDED.volume,
			complete = EXCLUDED.complete
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, k := range klines {
		_, err := stmt.Exec(
			k.Symbol,
			k.Interval,
			k.OpenTime,
			k.Open,
			k.High,
			k.Low,
			k.Close,
			k.Volume,
			k.Complete,
		)
		if err != nil {
			return fmt.Errorf("failed to execute statement: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
