package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"qcat/internal/types"
)

// Storage handles market data persistence
type Storage struct {
	db *sql.DB
}

// NewStorage creates a new storage instance
func NewStorage(db *sql.DB) *Storage {
	return &Storage{db: db}
}

// SaveKline saves kline data to database
func (s *Storage) SaveKline(ctx context.Context, kline *types.Kline) error {
	query := `
		INSERT INTO market_data (symbol, interval, timestamp, open, high, low, close, volume, complete, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (symbol, interval, timestamp) 
		DO UPDATE SET 
			open = EXCLUDED.open,
			high = EXCLUDED.high,
			low = EXCLUDED.low,
			close = EXCLUDED.close,
			volume = EXCLUDED.volume,
			complete = EXCLUDED.complete,
			updated_at = NOW()
	`

	_, err := s.db.ExecContext(ctx, query,
		kline.Symbol,
		kline.Interval,
		kline.OpenTime,
		kline.Open,
		kline.High,
		kline.Low,
		kline.Close,
		kline.Volume,
		kline.Complete,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to save kline: %w", err)
	}

	return nil
}

// SaveTrade saves trade data to database
func (s *Storage) SaveTrade(ctx context.Context, trade *types.Trade) error {
	query := `
		INSERT INTO trades (id, symbol, price, size, side, fee, fee_currency, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO NOTHING
	`

	_, err := s.db.ExecContext(ctx, query,
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
		return fmt.Errorf("failed to save trade: %w", err)
	}

	return nil
}

// SaveOrderBook saves order book data to database
func (s *Storage) SaveOrderBook(ctx context.Context, orderBook *types.OrderBook) error {
	bidsJSON, err := json.Marshal(orderBook.Bids)
	if err != nil {
		return fmt.Errorf("failed to marshal bids: %w", err)
	}

	asksJSON, err := json.Marshal(orderBook.Asks)
	if err != nil {
		return fmt.Errorf("failed to marshal asks: %w", err)
	}

	query := `
		INSERT INTO order_books (symbol, bids, asks, updated_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (symbol) 
		DO UPDATE SET 
			bids = EXCLUDED.bids,
			asks = EXCLUDED.asks,
			updated_at = EXCLUDED.updated_at
	`

	_, err = s.db.ExecContext(ctx, query,
		orderBook.Symbol,
		bidsJSON,
		asksJSON,
		orderBook.UpdatedAt,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to save order book: %w", err)
	}

	return nil
}

// SaveFundingRate saves funding rate data to database
func (s *Storage) SaveFundingRate(ctx context.Context, fundingRate *types.FundingRate) error {
	query := `
		INSERT INTO funding_rates (symbol, rate, next_rate, next_time, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := s.db.ExecContext(ctx, query,
		fundingRate.Symbol,
		fundingRate.Rate,
		fundingRate.NextRate,
		fundingRate.NextTime,
		fundingRate.LastUpdated,
	)

	if err != nil {
		return fmt.Errorf("failed to save funding rate: %w", err)
	}

	return nil
}

// SaveOpenInterest saves open interest data to database
func (s *Storage) SaveOpenInterest(ctx context.Context, openInterest *types.OpenInterest) error {
	query := `
		INSERT INTO open_interest (symbol, value, notional, timestamp, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := s.db.ExecContext(ctx, query,
		openInterest.Symbol,
		openInterest.Value,
		openInterest.Notional,
		openInterest.Timestamp,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to save open interest: %w", err)
	}

	return nil
}

// SaveTicker saves ticker data to database
func (s *Storage) SaveTicker(ctx context.Context, ticker *types.Ticker) error {
	query := `
		INSERT INTO tickers (
			symbol, price_change, price_change_percent, weighted_avg_price,
			prev_close_price, last_price, last_qty, bid_price, bid_qty,
			ask_price, ask_qty, open_price, high_price, low_price,
			volume, quote_volume, open_time, close_time, count, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20
		)
		ON CONFLICT (symbol) 
		DO UPDATE SET 
			price_change = EXCLUDED.price_change,
			price_change_percent = EXCLUDED.price_change_percent,
			weighted_avg_price = EXCLUDED.weighted_avg_price,
			prev_close_price = EXCLUDED.prev_close_price,
			last_price = EXCLUDED.last_price,
			last_qty = EXCLUDED.last_qty,
			bid_price = EXCLUDED.bid_price,
			bid_qty = EXCLUDED.bid_qty,
			ask_price = EXCLUDED.ask_price,
			ask_qty = EXCLUDED.ask_qty,
			open_price = EXCLUDED.open_price,
			high_price = EXCLUDED.high_price,
			low_price = EXCLUDED.low_price,
			volume = EXCLUDED.volume,
			quote_volume = EXCLUDED.quote_volume,
			open_time = EXCLUDED.open_time,
			close_time = EXCLUDED.close_time,
			count = EXCLUDED.count,
			updated_at = NOW()
	`

	_, err := s.db.ExecContext(ctx, query,
		ticker.Symbol,
		ticker.PriceChange,
		ticker.PriceChangePercent,
		ticker.WeightedAvgPrice,
		ticker.PrevClosePrice,
		ticker.LastPrice,
		ticker.LastQty,
		ticker.BidPrice,
		ticker.BidQty,
		ticker.AskPrice,
		ticker.AskQty,
		ticker.OpenPrice,
		ticker.HighPrice,
		ticker.LowPrice,
		ticker.Volume,
		ticker.QuoteVolume,
		ticker.OpenTime,
		ticker.CloseTime,
		ticker.Count,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to save ticker: %w", err)
	}

	return nil
}

// GetLatestKlines retrieves the latest klines for a symbol
func (s *Storage) GetLatestKlines(ctx context.Context, symbol, interval string, limit int) ([]*types.Kline, error) {
	query := `
		SELECT symbol, interval, timestamp, open, high, low, close, volume, complete
		FROM market_data
		WHERE symbol = $1 AND interval = $2
		ORDER BY timestamp DESC
		LIMIT $3
	`

	rows, err := s.db.QueryContext(ctx, query, symbol, interval, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query klines: %w", err)
	}
	defer rows.Close()

	var klines []*types.Kline
	for rows.Next() {
		var k types.Kline
		if err := rows.Scan(
			&k.Symbol,
			&k.Interval,
			&k.OpenTime,
			&k.Open,
			&k.High,
			&k.Low,
			&k.Close,
			&k.Volume,
			&k.Complete,
		); err != nil {
			return nil, fmt.Errorf("failed to scan kline: %w", err)
		}
		klines = append(klines, &k)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating klines: %w", err)
	}

	return klines, nil
}

// GetLatestTrades retrieves the latest trades for a symbol
func (s *Storage) GetLatestTrades(ctx context.Context, symbol string, limit int) ([]*types.Trade, error) {
	query := `
		SELECT id, symbol, price, size, side, fee, fee_currency, created_at
		FROM trades
		WHERE symbol = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, symbol, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query trades: %w", err)
	}
	defer rows.Close()

	var trades []*types.Trade
	for rows.Next() {
		var t types.Trade
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

// GetLatestOrderBook retrieves the latest order book for a symbol
func (s *Storage) GetLatestOrderBook(ctx context.Context, symbol string) (*types.OrderBook, error) {
	query := `
		SELECT symbol, bids, asks, updated_at
		FROM order_books
		WHERE symbol = $1
		ORDER BY updated_at DESC
		LIMIT 1
	`

	var orderBook types.OrderBook
	var bidsJSON, asksJSON []byte

	err := s.db.QueryRowContext(ctx, query, symbol).Scan(
		&orderBook.Symbol,
		&bidsJSON,
		&asksJSON,
		&orderBook.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return &types.OrderBook{
				Symbol:    symbol,
				Bids:      []types.Level{},
				Asks:      []types.Level{},
				UpdatedAt: time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("failed to query order book: %w", err)
	}

	if err := json.Unmarshal(bidsJSON, &orderBook.Bids); err != nil {
		return nil, fmt.Errorf("failed to parse bids: %w", err)
	}

	if err := json.Unmarshal(asksJSON, &orderBook.Asks); err != nil {
		return nil, fmt.Errorf("failed to parse asks: %w", err)
	}

	return &orderBook, nil
}

// GetLatestTicker retrieves the latest ticker for a symbol
func (s *Storage) GetLatestTicker(ctx context.Context, symbol string) (*types.Ticker, error) {
	query := `
		SELECT 
			symbol, price_change, price_change_percent, weighted_avg_price,
			prev_close_price, last_price, last_qty, bid_price, bid_qty,
			ask_price, ask_qty, open_price, high_price, low_price,
			volume, quote_volume, open_time, close_time, count
		FROM tickers
		WHERE symbol = $1
		ORDER BY updated_at DESC
		LIMIT 1
	`

	var ticker types.Ticker
	err := s.db.QueryRowContext(ctx, query, symbol).Scan(
		&ticker.Symbol,
		&ticker.PriceChange,
		&ticker.PriceChangePercent,
		&ticker.WeightedAvgPrice,
		&ticker.PrevClosePrice,
		&ticker.LastPrice,
		&ticker.LastQty,
		&ticker.BidPrice,
		&ticker.BidQty,
		&ticker.AskPrice,
		&ticker.AskQty,
		&ticker.OpenPrice,
		&ticker.HighPrice,
		&ticker.LowPrice,
		&ticker.Volume,
		&ticker.QuoteVolume,
		&ticker.OpenTime,
		&ticker.CloseTime,
		&ticker.Count,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("ticker not found for symbol %s", symbol)
		}
		return nil, fmt.Errorf("failed to query ticker: %w", err)
	}

	return &ticker, nil
}

// CleanupOldData removes old market data to manage storage
func (s *Storage) CleanupOldData(ctx context.Context, retentionDays int) error {
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	// Clean up old trades
	_, err := s.db.ExecContext(ctx, "DELETE FROM trades WHERE created_at < $1", cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to cleanup old trades: %w", err)
	}

	// Clean up old klines (keep only daily and above for long-term storage)
	_, err = s.db.ExecContext(ctx, 
		"DELETE FROM market_data WHERE created_at < $1 AND interval NOT IN ('1d', '1w', '1M')", 
		cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to cleanup old klines: %w", err)
	}

	// Clean up old funding rates
	_, err = s.db.ExecContext(ctx, "DELETE FROM funding_rates WHERE created_at < $1", cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to cleanup old funding rates: %w", err)
	}

	// Clean up old open interest
	_, err = s.db.ExecContext(ctx, "DELETE FROM open_interest WHERE created_at < $1", cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to cleanup old open interest: %w", err)
	}

	return nil
}