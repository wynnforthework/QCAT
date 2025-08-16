package backtest

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"qcat/internal/market"
)

// DBDataFeed implements DataFeed using database queries
type DBDataFeed struct {
	db        *sql.DB
	config    *Config
	dataTypes map[string]bool
	buffer    []*MarketData
	current   int
}

// NewDBDataFeed creates a new database data feed
func NewDBDataFeed(db *sql.DB, config *Config) (*DBDataFeed, error) {
	// Validate config
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	// Create data type map
	dataTypes := make(map[string]bool)
	// 新增：从配置中获取数据类型，如果没有则使用默认值
	if len(config.DataTypes) > 0 {
		for _, dataType := range config.DataTypes {
			dataTypes[dataType] = true
		}
	} else {
		// 新增：使用默认数据类型
		dataTypes["kline"] = true
		dataTypes["trade"] = true
		dataTypes["funding"] = true
		dataTypes["oi"] = true
	}

	return &DBDataFeed{
		db:        db,
		config:    config,
		dataTypes: dataTypes,
		buffer:    make([]*MarketData, 0),
		current:   0,
	}, nil
}

// Next returns the next market data point
func (f *DBDataFeed) Next() (*MarketData, error) {
	if !f.HasNext() {
		return nil, fmt.Errorf("no more data")
	}

	data := f.buffer[f.current]
	f.current++
	return data, nil
}

// HasNext returns true if there is more data
func (f *DBDataFeed) HasNext() bool {
	return f.current < len(f.buffer)
}

// Reset resets the data feed to the beginning
func (f *DBDataFeed) Reset() error {
	f.current = 0
	return nil
}

// Close closes the data feed
func (f *DBDataFeed) Close() error {
	f.buffer = nil
	return nil
}

// Load loads market data from the database
func (f *DBDataFeed) Load(ctx context.Context) error {
	// Load data for each symbol and type
	for _, symbol := range f.config.Symbols {
		for dataType := range f.dataTypes {
			if err := f.loadData(ctx, symbol, dataType); err != nil {
				return fmt.Errorf("failed to load %s data for %s: %w", dataType, symbol, err)
			}
		}
	}

	// Sort data by timestamp
	sort.Slice(f.buffer, func(i, j int) bool {
		return f.buffer[i].Timestamp.Before(f.buffer[j].Timestamp)
	})

	return nil
}

// loadData loads market data of a specific type
func (f *DBDataFeed) loadData(ctx context.Context, symbol, dataType string) error {
	switch dataType {
	case "kline":
		return f.loadKlines(ctx, symbol)
	case "trade":
		return f.loadTrades(ctx, symbol)
	case "funding":
		return f.loadFundingRates(ctx, symbol)
	case "oi":
		return f.loadOpenInterest(ctx, symbol)
	default:
		return fmt.Errorf("unsupported data type: %s", dataType)
	}
}

// loadKlines loads historical kline data
func (f *DBDataFeed) loadKlines(ctx context.Context, symbol string) error {
	query := `
		SELECT timestamp, open, high, low, close, volume
		FROM market_data
		WHERE symbol = $1 AND interval = '1m'
			AND timestamp BETWEEN $2 AND $3
		ORDER BY timestamp ASC
	`

	rows, err := f.db.QueryContext(ctx, query, symbol, f.config.StartTime, f.config.EndTime)
	if err != nil {
		return fmt.Errorf("failed to query klines: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var k market.Kline
		if err := rows.Scan(
			&k.OpenTime,
			&k.Open,
			&k.High,
			&k.Low,
			&k.Close,
			&k.Volume,
		); err != nil {
			return fmt.Errorf("failed to scan kline: %w", err)
		}

		k.Symbol = symbol
		k.Interval = "1m"
		k.CloseTime = k.OpenTime.Add(time.Minute)
		k.Complete = true

		f.buffer = append(f.buffer, &MarketData{
			Symbol:    symbol,
			Timestamp: k.OpenTime,
			Type:      "kline",
			Data:      &k,
		})
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating klines: %w", err)
	}

	return nil
}

// loadTrades loads historical trade data
func (f *DBDataFeed) loadTrades(ctx context.Context, symbol string) error {
	query := `
		SELECT id, price, size, side, fee, fee_currency, created_at
		FROM trades
		WHERE symbol = $1 AND created_at BETWEEN $2 AND $3
		ORDER BY created_at ASC
	`

	rows, err := f.db.QueryContext(ctx, query, symbol, f.config.StartTime, f.config.EndTime)
	if err != nil {
		return fmt.Errorf("failed to query trades: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var t market.Trade
		if err := rows.Scan(
			&t.ID,
			&t.Price,
			&t.Quantity,
			&t.Side,
			&t.Fee,
			&t.FeeCoin,
			&t.Timestamp,
		); err != nil {
			return fmt.Errorf("failed to scan trade: %w", err)
		}

		t.Symbol = symbol

		f.buffer = append(f.buffer, &MarketData{
			Symbol:    symbol,
			Timestamp: t.Timestamp,
			Type:      "trade",
			Data:      &t,
		})
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating trades: %w", err)
	}

	return nil
}

// loadFundingRates loads historical funding rate data
func (f *DBDataFeed) loadFundingRates(ctx context.Context, symbol string) error {
	query := `
		SELECT rate, next_rate, next_time, created_at
		FROM funding_rates
		WHERE symbol = $1 AND created_at BETWEEN $2 AND $3
		ORDER BY created_at ASC
	`

	rows, err := f.db.QueryContext(ctx, query, symbol, f.config.StartTime, f.config.EndTime)
	if err != nil {
		return fmt.Errorf("failed to query funding rates: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var fr market.FundingRate
		if err := rows.Scan(
			&fr.Rate,
			&fr.NextRate,
			&fr.NextTime,
			&fr.LastUpdated,
		); err != nil {
			return fmt.Errorf("failed to scan funding rate: %w", err)
		}

		fr.Symbol = symbol

		f.buffer = append(f.buffer, &MarketData{
			Symbol:    symbol,
			Timestamp: fr.LastUpdated,
			Type:      "funding",
			Data:      &fr,
		})
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating funding rates: %w", err)
	}

	return nil
}

// loadOpenInterest loads historical open interest data
func (f *DBDataFeed) loadOpenInterest(ctx context.Context, symbol string) error {
	query := `
		SELECT value, notional, timestamp
		FROM open_interest
		WHERE symbol = $1 AND timestamp BETWEEN $2 AND $3
		ORDER BY timestamp ASC
	`

	rows, err := f.db.QueryContext(ctx, query, symbol, f.config.StartTime, f.config.EndTime)
	if err != nil {
		return fmt.Errorf("failed to query open interest: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var oi market.OpenInterest
		if err := rows.Scan(
			&oi.Value,
			&oi.Notional,
			&oi.Timestamp,
		); err != nil {
			return fmt.Errorf("failed to scan open interest: %w", err)
		}

		oi.Symbol = symbol

		f.buffer = append(f.buffer, &MarketData{
			Symbol:    symbol,
			Timestamp: oi.Timestamp,
			Type:      "oi",
			Data:      &oi,
		})
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating open interest: %w", err)
	}

	return nil
}

// validateConfig validates backtest configuration
func validateConfig(config *Config) error {
	if config.StartTime.IsZero() {
		return &ErrInvalidConfig{Field: "start_time", Message: "start time is required"}
	}
	if config.EndTime.IsZero() {
		return &ErrInvalidConfig{Field: "end_time", Message: "end time is required"}
	}
	if config.EndTime.Before(config.StartTime) {
		return &ErrInvalidConfig{Field: "end_time", Message: "end time must be after start time"}
	}
	if len(config.Symbols) == 0 {
		return &ErrInvalidConfig{Field: "symbols", Message: "at least one symbol is required"}
	}
	if config.Capital <= 0 {
		return &ErrInvalidConfig{Field: "capital", Message: "capital must be positive"}
	}
	if config.Leverage <= 0 {
		return &ErrInvalidConfig{Field: "leverage", Message: "leverage must be positive"}
	}
	if len(config.DataTypes) == 0 {
		return &ErrInvalidConfig{Field: "data_types", Message: "at least one data type is required"}
	}
	return nil
}
