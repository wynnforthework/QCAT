package exchange

import (
	"time"
)

// OrderSide represents the side of an order (buy/sell)
type OrderSide string

const (
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"
)

// OrderType represents the type of an order
type OrderType string

const (
	OrderTypeMarket OrderType = "market"
	OrderTypeLimit  OrderType = "limit"
	OrderTypePost   OrderType = "post_only"
	OrderTypeIOC    OrderType = "ioc"
	OrderTypeFOK    OrderType = "fok"
)

// OrderStatus represents the status of an order
type OrderStatus string

const (
	OrderStatusNew      OrderStatus = "new"
	OrderStatusPartial  OrderStatus = "partial"
	OrderStatusFilled   OrderStatus = "filled"
	OrderStatusCanceled OrderStatus = "canceled"
	OrderStatusRejected OrderStatus = "rejected"
	OrderStatusExpired  OrderStatus = "expired"
)

// PositionSide represents the side of a position
type PositionSide string

const (
	PositionSideLong  PositionSide = "long"
	PositionSideShort PositionSide = "short"
)

// MarginType represents the margin type of a position
type MarginType string

const (
	MarginTypeCross    MarginType = "cross"
	MarginTypeIsolated MarginType = "isolated"
)

// Order represents a trading order
type Order struct {
	ID            string      `json:"id"`
	ExchangeID    string      `json:"exchange_id"`
	Symbol        string      `json:"symbol"`
	Side          OrderSide   `json:"side"`
	Type          OrderType   `json:"type"`
	Status        OrderStatus `json:"status"`
	Price         float64     `json:"price"`
	Quantity      float64     `json:"quantity"`
	FilledQty     float64     `json:"filled_qty"`
	RemainingQty  float64     `json:"remaining_qty"`
	AvgPrice      float64     `json:"avg_price"`
	Fee           float64     `json:"fee"`
	FeeCurrency   string      `json:"fee_currency"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
	ClientOrderID string      `json:"client_order_id"`
}

// Position represents a trading position
type Position struct {
	Symbol        string       `json:"symbol"`
	Side          PositionSide `json:"side"`
	Quantity      float64      `json:"quantity"`
	EntryPrice    float64      `json:"entry_price"`
	MarkPrice     float64      `json:"mark_price"`
	LiqPrice      float64      `json:"liq_price"`
	Leverage      int          `json:"leverage"`
	MarginType    MarginType   `json:"margin_type"`
	UnrealizedPnL float64      `json:"unrealized_pnl"`
	RealizedPnL   float64      `json:"realized_pnl"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

// AccountBalance represents account balance information
type AccountBalance struct {
	Asset          string    `json:"asset"`
	Total          float64   `json:"total"`
	Available      float64   `json:"available"`
	Locked         float64   `json:"locked"`
	CrossMargin    float64   `json:"cross_margin"`
	IsolatedMargin float64   `json:"isolated_margin"`
	UnrealizedPnL  float64   `json:"unrealized_pnl"`
	RealizedPnL    float64   `json:"realized_pnl"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// RiskLimit represents risk limit settings
type RiskLimit struct {
	Symbol            string    `json:"symbol"`
	Leverage          int       `json:"leverage"`
	MaxPositionSize   float64   `json:"max_position_size"`
	MaintenanceMargin float64   `json:"maintenance_margin"`
	InitialMargin     float64   `json:"initial_margin"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// OrderRequest represents an order placement request
type OrderRequest struct {
	Symbol        string    `json:"symbol"`
	Side          OrderSide `json:"side"`
	Type          OrderType `json:"type"`
	Price         float64   `json:"price"`
	Quantity      float64   `json:"quantity"`
	ClientOrderID string    `json:"client_order_id"`
	ReduceOnly    bool      `json:"reduce_only"`
	PostOnly      bool      `json:"post_only"`
	TimeInForce   string    `json:"time_in_force"`
}

// OrderCancelRequest represents an order cancellation request
type OrderCancelRequest struct {
	Symbol        string `json:"symbol"`
	OrderID       string `json:"order_id"`
	ClientOrderID string `json:"client_order_id"`
}

// OrderResponse represents an order operation response
type OrderResponse struct {
	Order   *Order `json:"order"`
	Success bool   `json:"success"`
	Error   *Error `json:"error"`
}

// Error represents an exchange error
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// ExchangeInfo represents exchange information
type ExchangeInfo struct {
	Name            string    `json:"name"`
	Symbols         []string  `json:"symbols"`
	RateLimits      []int     `json:"rate_limits"`
	ServerTime      time.Time `json:"server_time"`
	ExchangeFilters []string  `json:"exchange_filters"`
}

// SymbolInfo represents symbol information
type SymbolInfo struct {
	Symbol            string  `json:"symbol"`
	BaseAsset         string  `json:"base_asset"`
	QuoteAsset        string  `json:"quote_asset"`
	PricePrecision    int     `json:"price_precision"`
	QuantityPrecision int     `json:"quantity_precision"`
	MinPrice          float64 `json:"min_price"`
	MaxPrice          float64 `json:"max_price"`
	MinQuantity       float64 `json:"min_quantity"`
	MaxQuantity       float64 `json:"max_quantity"`
	MinNotional       float64 `json:"min_notional"`
	MaxLeverage       int     `json:"max_leverage"`
	ContractSize      float64 `json:"contract_size"`
	MaintenanceMargin float64 `json:"maintenance_margin"`
	RequiredMargin    float64 `json:"required_margin"`
	PriceTickSize     float64 `json:"price_tick_size"`
	QuantityStepSize  float64 `json:"quantity_step_size"`
}
