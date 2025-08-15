package trade

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/cache"
)

// Manager manages trade data collection and processing
type Manager struct {
	db          *sql.DB
	cache       cache.Cacher
	stats       map[string]*TradeStats
	subscribers map[string][]chan *Trade
	mu          sync.RWMutex

	// Batch processing
	batchSize    int
	batchTimeout time.Duration
	batchBuffer  []*Trade
	bufferMu     sync.Mutex
}

// NewManager creates a new trade manager
func NewManager(db *sql.DB, cache cache.Cacher) *Manager {
	m := &Manager{
		db:           db,
		cache:        cache,
		stats:        make(map[string]*TradeStats),
		subscribers:  make(map[string][]chan *Trade),
		batchSize:    100,
		batchTimeout: 5 * time.Second,
		batchBuffer:  make([]*Trade, 0, 100),
	}

	// Start batch processor
	go m.processBatch()

	// Start stats updater
	go m.updateStats()

	return m
}

// Subscribe subscribes to trade updates for a symbol
func (m *Manager) Subscribe(symbol string) chan *Trade {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan *Trade, 100)
	m.subscribers[symbol] = append(m.subscribers[symbol], ch)
	return ch
}

// Unsubscribe removes a subscription
func (m *Manager) Unsubscribe(symbol string, ch chan *Trade) {
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

// ProcessTrade processes a new trade
func (m *Manager) ProcessTrade(trade *Trade) error {
	// Update stats
	m.updateTradeStats(trade)

	// Notify subscribers
	m.notifySubscribers(trade)

	// Add to batch buffer
	return m.addToBatch(trade)
}

// GetStats returns trade statistics for a symbol
func (m *Manager) GetStats(symbol string) *TradeStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if stats, exists := m.stats[symbol]; exists {
		return stats
	}
	return nil
}

// GetTradeHistory returns historical trades for a symbol
func (m *Manager) GetTradeHistory(ctx context.Context, symbol string, limit int) ([]*Trade, error) {
	query := `
		SELECT id, symbol, price, size, side, fee, fee_currency, created_at
		FROM trades
		WHERE symbol = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := m.db.QueryContext(ctx, query, symbol, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query trade history: %w", err)
	}
	defer rows.Close()

	var trades []*Trade
	for rows.Next() {
		var t Trade
		if err := rows.Scan(
			&t.ID,
			&t.Symbol,
			&t.Price,
			&t.Quantity,
			&t.Side,
			&t.Fee,
			&t.FeeCoin,
			&t.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("failed to scan trade: %w", err)
		}
		trades = append(trades, &t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating trades: %w", err)
	}

	return trades, nil
}

// GetAggregatedTrades returns aggregated trade data
func (m *Manager) GetAggregatedTrades(ctx context.Context, symbol string, interval time.Duration, start, end time.Time) ([]*TradeAggregation, error) {
	query := `
		WITH trade_bins AS (
			SELECT
				time_bucket($1, created_at) AS bucket,
				first(price, created_at) as open,
				max(price) as high,
				min(price) as low,
				last(price, created_at) as close,
				sum(size) as volume,
				sum(price * size) / sum(size) as vwap,
				count(*) as num_trades
			FROM trades
			WHERE symbol = $2 AND created_at BETWEEN $3 AND $4
			GROUP BY bucket
			ORDER BY bucket ASC
		)
		SELECT * FROM trade_bins
	`

	rows, err := m.db.QueryContext(ctx, query, interval, symbol, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query aggregated trades: %w", err)
	}
	defer rows.Close()

	var aggregations []*TradeAggregation
	for rows.Next() {
		var agg TradeAggregation
		var bucket time.Time
		if err := rows.Scan(
			&bucket,
			&agg.Open,
			&agg.High,
			&agg.Low,
			&agg.Close,
			&agg.Volume,
			&agg.VWAP,
			&agg.NumTrades,
		); err != nil {
			return nil, fmt.Errorf("failed to scan aggregation: %w", err)
		}
		agg.Symbol = symbol
		agg.StartTime = bucket
		agg.EndTime = bucket.Add(interval)
		aggregations = append(aggregations, &agg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating aggregations: %w", err)
	}

	return aggregations, nil
}

// updateTradeStats updates trade statistics
func (m *Manager) updateTradeStats(trade *Trade) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stats, exists := m.stats[trade.Symbol]
	if !exists {
		stats = &TradeStats{
			Symbol:        trade.Symbol,
			LastTradeTime: trade.Timestamp,
		}
		m.stats[trade.Symbol] = stats
	}

	// Update last trade info
	stats.LastPrice = trade.Price
	stats.LastQuantity = trade.Quantity
	stats.LastTradeTime = trade.Timestamp

	// Update 24h stats if needed
	cutoff := time.Now().Add(-24 * time.Hour)
	if trade.Timestamp.After(cutoff) {
		stats.Volume24h += trade.Quantity
		stats.QuoteVolume24h += trade.Price * trade.Quantity
		stats.NumberOfTrades24h++

		if stats.HighPrice24h == 0 || trade.Price > stats.HighPrice24h {
			stats.HighPrice24h = trade.Price
		}
		if stats.LowPrice24h == 0 || trade.Price < stats.LowPrice24h {
			stats.LowPrice24h = trade.Price
		}
	}
}

// notifySubscribers notifies all subscribers of a new trade
func (m *Manager) notifySubscribers(trade *Trade) {
	m.mu.RLock()
	subs := m.subscribers[trade.Symbol]
	m.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- trade:
		default:
			// Channel is full, skip
		}
	}
}

// addToBatch adds a trade to the batch buffer
func (m *Manager) addToBatch(trade *Trade) error {
	m.bufferMu.Lock()
	m.batchBuffer = append(m.batchBuffer, trade)
	size := len(m.batchBuffer)
	m.bufferMu.Unlock()

	if size >= m.batchSize {
		return m.flushBatch()
	}
	return nil
}

// processBatch processes the batch buffer periodically
func (m *Manager) processBatch() {
	ticker := time.NewTicker(m.batchTimeout)
	defer ticker.Stop()

	for range ticker.C {
		if err := m.flushBatch(); err != nil {
			log.Printf("Error flushing trade batch: %v", err)
		}
	}
}

// flushBatch stores the batch buffer in the database
func (m *Manager) flushBatch() error {
	m.bufferMu.Lock()
	if len(m.batchBuffer) == 0 {
		m.bufferMu.Unlock()
		return nil
	}

	// Copy buffer and reset
	buffer := make([]*Trade, len(m.batchBuffer))
	copy(buffer, m.batchBuffer)
	m.batchBuffer = m.batchBuffer[:0]
	m.bufferMu.Unlock()

	// Store trades in database
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO trades (
			id, symbol, price, size, side, fee, fee_currency, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, trade := range buffer {
		_, err := stmt.Exec(
			trade.ID,
			trade.Symbol,
			trade.Price,
			trade.Quantity,
			trade.Side,
			trade.Fee,
			trade.FeeCoin,
			trade.Timestamp,
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

// updateStats periodically updates 24h statistics
func (m *Manager) updateStats() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		for symbol, stats := range m.stats {
			// Calculate 24h price change
			trades, err := m.GetTradeHistory(context.Background(), symbol, 1)
			if err != nil || len(trades) == 0 {
				continue
			}

			oldTrades, err := m.GetAggregatedTrades(
				context.Background(),
				symbol,
				time.Hour*24,
				time.Now().Add(-24*time.Hour),
				time.Now(),
			)
			if err != nil || len(oldTrades) == 0 {
				continue
			}

			priceChange := stats.LastPrice - oldTrades[0].Open
			stats.PriceChange24h = priceChange
			if oldTrades[0].Open > 0 {
				stats.PriceChangeP24h = (priceChange / oldTrades[0].Open) * 100
			}
		}
		m.mu.Unlock()
	}
}
