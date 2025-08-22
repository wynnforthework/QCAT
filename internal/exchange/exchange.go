package exchange

import (
	"context"
	"time"
)

// Exchange defines the interface for interacting with exchanges
type Exchange interface {
	// GetExchangeInfo returns exchange information
	GetExchangeInfo(ctx context.Context) (*ExchangeInfo, error)

	// GetSymbolInfo returns symbol information
	GetSymbolInfo(ctx context.Context, symbol string) (*SymbolInfo, error)

	// GetServerTime returns the exchange server time
	GetServerTime(ctx context.Context) (time.Time, error)

	// GetAccountBalance returns account balances
	GetAccountBalance(ctx context.Context) (map[string]*AccountBalance, error)

	// GetPositions returns all positions
	GetPositions(ctx context.Context) ([]*Position, error)

	// GetPosition returns a specific position
	GetPosition(ctx context.Context, symbol string) (*Position, error)

	// GetLeverage returns the leverage for a symbol
	GetLeverage(ctx context.Context, symbol string) (int, error)

	// SetLeverage sets the leverage for a symbol
	SetLeverage(ctx context.Context, symbol string, leverage int) error

	// SetMarginType sets the margin type for a symbol
	SetMarginType(ctx context.Context, symbol string, marginType MarginType) error

	// PlaceOrder places an order
	PlaceOrder(ctx context.Context, req *OrderRequest) (*OrderResponse, error)

	// CancelOrder cancels an order
	CancelOrder(ctx context.Context, req *OrderCancelRequest) (*OrderResponse, error)

	// CancelAllOrders cancels all orders for a symbol
	CancelAllOrders(ctx context.Context, symbol string) error

	// GetOrder returns order information
	GetOrder(ctx context.Context, symbol, orderID string) (*Order, error)

	// GetOpenOrders returns open orders
	GetOpenOrders(ctx context.Context, symbol string) ([]*Order, error)

	// GetOrderHistory returns order history
	GetOrderHistory(ctx context.Context, symbol string, startTime, endTime time.Time) ([]*Order, error)

	// GetRiskLimits returns risk limits for a symbol
	GetRiskLimits(ctx context.Context, symbol string) (*RiskLimits, error)

	// GetMarginInfo returns margin information for account
	GetMarginInfo(ctx context.Context) (*MarginInfo, error)

	// SetRiskLimits sets risk limits for a symbol
	SetRiskLimits(ctx context.Context, symbol string, limits *RiskLimits) error

	// GetPositionByID returns position by ID
	GetPositionByID(ctx context.Context, positionID string) (*Position, error)

	// GetSymbolPrice returns the current price for a symbol
	GetSymbolPrice(ctx context.Context, symbol string) (float64, error)
}

// BaseExchange provides common functionality for exchanges
type BaseExchange struct {
	config *ExchangeConfig
}

// ExchangeConfig represents exchange configuration
type ExchangeConfig struct {
	Name           string
	APIKey         string
	APISecret      string
	TestNet        bool
	BaseURL        string
	FuturesBaseURL string
	// SuppressCacheWarnings indicates whether to attempt suppressing cache warnings
	// from the underlying exchange library (like banexg). Note that this may not
	// completely eliminate all warnings due to library-internal caching mechanisms.
	SuppressCacheWarnings bool
}

// NewBaseExchange creates a new base exchange
func NewBaseExchange(config *ExchangeConfig) *BaseExchange {
	return &BaseExchange{
		config: config,
	}
}

// Name returns the exchange name
func (e *BaseExchange) Name() string {
	return e.config.Name
}

// Config returns the exchange configuration
func (e *BaseExchange) Config() *ExchangeConfig {
	return e.config
}
