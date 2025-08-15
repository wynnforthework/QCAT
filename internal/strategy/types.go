package strategy

import (
	"context"
	"time"

	"qcat/internal/exchange"
)

// Mode represents the strategy execution mode
type Mode string

const (
	ModeLive     Mode = "live"
	ModePaper    Mode = "paper"
	ModeBacktest Mode = "backtest"
)

// State represents the strategy state
type State string

const (
	StateInitializing State = "initializing"
	StateRunning      State = "running"
	StatePaused       State = "paused"
	StateStopped      State = "stopped"
	StateError        State = "error"
)

// Signal represents a trading signal
type Signal struct {
	ID        string                 `json:"id"`
	Strategy  string                 `json:"strategy"`
	Symbol    string                 `json:"symbol"`
	Side      exchange.OrderSide     `json:"side"`
	Type      exchange.OrderType     `json:"type"`
	Price     float64                `json:"price"`
	Quantity  float64                `json:"quantity"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// Context represents the strategy execution context
type Context struct {
	Mode      Mode
	Strategy  string
	Symbol    string
	Exchange  exchange.Exchange
	StartTime time.Time
	EndTime   time.Time
	Params    map[string]interface{}
}

// Result represents a strategy execution result
type Result struct {
	Strategy     string                 `json:"strategy"`
	Symbol       string                 `json:"symbol"`
	Mode         Mode                   `json:"mode"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time"`
	InitialValue float64                `json:"initial_value"`
	FinalValue   float64                `json:"final_value"`
	PnL          float64                `json:"pnl"`
	PnLPercent   float64                `json:"pnl_percent"`
	MaxDrawdown  float64                `json:"max_drawdown"`
	SharpeRatio  float64                `json:"sharpe_ratio"`
	NumTrades    int                    `json:"num_trades"`
	WinRate      float64                `json:"win_rate"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// Strategy defines the interface that all strategies must implement
type Strategy interface {
	// Initialize initializes the strategy
	Initialize(ctx context.Context, params map[string]interface{}) error

	// Start starts the strategy execution
	Start(ctx context.Context) error

	// Stop stops the strategy execution
	Stop(ctx context.Context) error

	// OnTick handles market data updates
	OnTick(ctx context.Context, data interface{}) error

	// OnSignal handles trading signals
	OnSignal(ctx context.Context, signal *Signal) error

	// OnOrder handles order updates
	OnOrder(ctx context.Context, order *exchange.Order) error

	// OnPosition handles position updates
	OnPosition(ctx context.Context, position *exchange.Position) error

	// GetState returns the current strategy state
	GetState() State

	// GetResult returns the strategy execution result
	GetResult() *Result
}

// Config represents strategy configuration
type Config struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Mode        Mode                   `json:"mode"`
	Symbol      string                 `json:"symbol"`
	Exchange    string                 `json:"exchange"`
	Params      map[string]interface{} `json:"params"`
}

// BaseStrategy provides common functionality for strategies
type BaseStrategy struct {
	config  *Config
	context *Context
	state   State
	result  *Result
}

// NewBaseStrategy creates a new base strategy
func NewBaseStrategy(config *Config) *BaseStrategy {
	return &BaseStrategy{
		config: config,
		state:  StateInitializing,
		result: &Result{
			Strategy:  config.Name,
			Symbol:    config.Symbol,
			Mode:      config.Mode,
			StartTime: time.Now(),
			Metadata:  make(map[string]interface{}),
		},
	}
}

// GetState returns the current strategy state
func (s *BaseStrategy) GetState() State {
	return s.state
}

// GetResult returns the strategy execution result
func (s *BaseStrategy) GetResult() *Result {
	s.result.EndTime = time.Now()
	return s.result
}

// SetState sets the strategy state
func (s *BaseStrategy) SetState(state State) {
	s.state = state
}

// UpdateResult updates the strategy result
func (s *BaseStrategy) UpdateResult(update func(*Result)) {
	update(s.result)
}

// GetConfig returns the strategy configuration
func (s *BaseStrategy) GetConfig() *Config {
	return s.config
}

// GetContext returns the strategy context
func (s *BaseStrategy) GetContext() *Context {
	return s.context
}

// SetContext sets the strategy context
func (s *BaseStrategy) SetContext(ctx *Context) {
	s.context = ctx
}

// Validate validates the strategy configuration
func (s *BaseStrategy) Validate() error {
	if s.config.Name == "" {
		return ErrInvalidConfig{Field: "name", Message: "name is required"}
	}
	if s.config.Version == "" {
		return ErrInvalidConfig{Field: "version", Message: "version is required"}
	}
	if s.config.Symbol == "" {
		return ErrInvalidConfig{Field: "symbol", Message: "symbol is required"}
	}
	if s.config.Exchange == "" {
		return ErrInvalidConfig{Field: "exchange", Message: "exchange is required"}
	}
	return nil
}

// Error types
type ErrInvalidConfig struct {
	Field   string
	Message string
}

func (e ErrInvalidConfig) Error() string {
	return "invalid config: " + e.Field + " - " + e.Message
}

type ErrStrategyError struct {
	Message string
	Err     error
}

func (e ErrStrategyError) Error() string {
	if e.Err != nil {
		return "strategy error: " + e.Message + ": " + e.Err.Error()
	}
	return "strategy error: " + e.Message
}

func (e ErrStrategyError) Unwrap() error {
	return e.Err
}
