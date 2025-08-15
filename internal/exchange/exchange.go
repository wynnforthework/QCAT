package exchange

import (
	"context"
	"time"
)

// Exchange defines the interface that all exchange implementations must satisfy
type Exchange interface {
	// Basic Information
	Name() string
	GetExchangeInfo(ctx context.Context) (*ExchangeInfo, error)
	GetSymbolInfo(ctx context.Context, symbol string) (*SymbolInfo, error)
	GetServerTime(ctx context.Context) (time.Time, error)

	// Account & Balance
	GetAccountBalance(ctx context.Context) (map[string]*AccountBalance, error)
	GetPositions(ctx context.Context) ([]*Position, error)
	GetPosition(ctx context.Context, symbol string) (*Position, error)
	GetLeverage(ctx context.Context, symbol string) (int, error)
	SetLeverage(ctx context.Context, symbol string, leverage int) error
	SetMarginType(ctx context.Context, symbol string, marginType MarginType) error

	// Order Management
	PlaceOrder(ctx context.Context, req *OrderRequest) (*OrderResponse, error)
	CancelOrder(ctx context.Context, req *OrderCancelRequest) (*OrderResponse, error)
	CancelAllOrders(ctx context.Context, symbol string) error
	GetOrder(ctx context.Context, symbol, orderID string) (*Order, error)
	GetOpenOrders(ctx context.Context, symbol string) ([]*Order, error)
	GetOrderHistory(ctx context.Context, symbol string, startTime, endTime time.Time) ([]*Order, error)

	// Risk Management
	GetRiskLimits(ctx context.Context, symbol string) ([]*RiskLimit, error)
	SetRiskLimits(ctx context.Context, symbol string, limits []*RiskLimit) error

	// Lifecycle Management
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	IsRunning() bool
}

// ExchangeOption represents an option for configuring an exchange
type ExchangeOption func(Exchange) error

// ExchangeConfig represents common exchange configuration
type ExchangeConfig struct {
	Name      string `json:"name"`
	APIKey    string `json:"api_key"`
	APISecret string `json:"api_secret"`
	TestNet   bool   `json:"testnet"`
}

// BaseExchange provides common functionality for exchange implementations
type BaseExchange struct {
	config     *ExchangeConfig
	running    bool
	startTime  time.Time
	lastUpdate time.Time
}

// NewBaseExchange creates a new base exchange instance
func NewBaseExchange(config *ExchangeConfig) *BaseExchange {
	return &BaseExchange{
		config: config,
	}
}

// Name returns the exchange name
func (e *BaseExchange) Name() string {
	return e.config.Name
}

// IsRunning returns whether the exchange is running
func (e *BaseExchange) IsRunning() bool {
	return e.running
}

// Start starts the exchange
func (e *BaseExchange) Start(ctx context.Context) error {
	e.running = true
	e.startTime = time.Now()
	return nil
}

// Stop stops the exchange
func (e *BaseExchange) Stop(ctx context.Context) error {
	e.running = false
	return nil
}

// WithAPIKey sets the API key
func WithAPIKey(apiKey string) ExchangeOption {
	return func(e Exchange) error {
		if be, ok := e.(*BaseExchange); ok {
			be.config.APIKey = apiKey
		}
		return nil
	}
}

// WithAPISecret sets the API secret
func WithAPISecret(apiSecret string) ExchangeOption {
	return func(e Exchange) error {
		if be, ok := e.(*BaseExchange); ok {
			be.config.APISecret = apiSecret
		}
		return nil
	}
}

// WithTestNet sets whether to use testnet
func WithTestNet(testnet bool) ExchangeOption {
	return func(e Exchange) error {
		if be, ok := e.(*BaseExchange); ok {
			be.config.TestNet = testnet
		}
		return nil
	}
}
