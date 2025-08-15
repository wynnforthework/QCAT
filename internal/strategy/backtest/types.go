package backtest

import (
	"time"

	"qcat/internal/exchange"
)

// BacktestConfig represents backtest configuration
type BacktestConfig struct {
	StartTime time.Time
	EndTime   time.Time
	Symbols   []string
	Capital   float64
	Leverage  int
	Fees      struct {
		Maker float64
		Taker float64
	}
	DataTypes []string // "kline", "trade", "funding", "oi", etc.
}

// BacktestResult represents backtest results
type BacktestResult struct {
	StartTime     time.Time
	EndTime       time.Time
	InitialValue  float64
	FinalValue    float64
	PnL           float64
	PnLPercent    float64
	MaxDrawdown   float64
	SharpeRatio   float64
	NumTrades     int
	WinRate       float64
	Trades        []*Trade
	Positions     []*Position
	EquityCurve   []EquityPoint
	DrawdownCurve []DrawdownPoint
}

// Trade represents a backtest trade
type Trade struct {
	ID            string
	Symbol        string
	Side          exchange.OrderSide
	Type          exchange.OrderType
	Price         float64
	Quantity      float64
	Fee           float64
	PnL           float64
	PnLPercent    float64
	EntryTime     time.Time
	ExitTime      time.Time
	HoldingPeriod time.Duration
}

// Position represents a backtest position
type Position struct {
	Symbol        string
	Side          exchange.PositionSide
	EntryPrice    float64
	ExitPrice     float64
	Quantity      float64
	Leverage      int
	MarginType    exchange.MarginType
	UnrealizedPnL float64
	RealizedPnL   float64
	OpenTime      time.Time
	CloseTime     time.Time
}

// EquityPoint represents a point in the equity curve
type EquityPoint struct {
	Timestamp time.Time
	Equity    float64
	PnL       float64
}

// DrawdownPoint represents a point in the drawdown curve
type DrawdownPoint struct {
	Timestamp time.Time
	Drawdown  float64
	Duration  time.Duration
}

// MarketData represents historical market data
type MarketData struct {
	Symbol    string
	Timestamp time.Time
	Type      string // "kline", "trade", "funding", "oi", etc.
	Data      interface{}
}

// DataFeed represents a market data feed
type DataFeed interface {
	// Next returns the next market data point
	Next() (*MarketData, error)

	// HasNext returns true if there is more data
	HasNext() bool

	// Reset resets the data feed to the beginning
	Reset() error

	// Close closes the data feed
	Close() error
}

// Error types
type ErrInvalidConfig struct {
	Field   string
	Message string
}

func (e ErrInvalidConfig) Error() string {
	return "invalid config: " + e.Field + " - " + e.Message
}

type ErrDataFeed struct {
	Message string
	Err     error
}

func (e ErrDataFeed) Error() string {
	if e.Err != nil {
		return "data feed error: " + e.Message + ": " + e.Err.Error()
	}
	return "data feed error: " + e.Message
}

func (e ErrDataFeed) Unwrap() error {
	return e.Err
}

type ErrBacktest struct {
	Message string
	Err     error
}

func (e ErrBacktest) Error() string {
	if e.Err != nil {
		return "backtest error: " + e.Message + ": " + e.Err.Error()
	}
	return "backtest error: " + e.Message
}

func (e ErrBacktest) Unwrap() error {
	return e.Err
}
