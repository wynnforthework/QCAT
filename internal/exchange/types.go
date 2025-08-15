package exchange

import (
	"time"
)

// MarginInfo represents margin account information
type MarginInfo struct {
	TotalAssetValue   float64   // 总资产价值
	TotalDebtValue    float64   // 总负债价值
	MarginRatio       float64   // 保证金率 = 总资产/总负债
	MaintenanceMargin float64   // 维持保证金率
	MarginCallRatio   float64   // 追保线
	LiquidationRatio  float64   // 强平线
	UpdatedAt         time.Time // 更新时间
}

// AccountBalance represents account balance information
type AccountBalance struct {
	Asset          string    // 资产名称
	Total          float64   // 总余额
	Available      float64   // 可用余额
	Locked         float64   // 锁定余额
	CrossMargin    float64   // 全仓保证金
	IsolatedMargin float64   // 逐仓保证金
	UnrealizedPnL  float64   // 未实现盈亏
	RealizedPnL    float64   // 已实现盈亏
	UpdatedAt      time.Time // 更新时间
}

// MarginLevel represents different margin thresholds
type MarginLevel int

const (
	MarginLevelSafe MarginLevel = iota
	MarginLevelWarning
	MarginLevelDanger
	MarginLevelLiquidation
)

// MarginAlert represents a margin alert event
type MarginAlert struct {
	Level     MarginLevel // 警告级别
	Ratio     float64     // 当前保证金率
	Threshold float64     // 触发阈值
	Message   string      // 警告信息
	CreatedAt time.Time   // 创建时间
}

// ExchangeInfo represents exchange information
type ExchangeInfo struct {
	Name       string       // 交易所名称
	Version    string       // API版本
	ServerTime time.Time    // 服务器时间
	Timezone   string       // 时区
	RateLimits []RateLimit  // 速率限制
	Symbols    []SymbolInfo // 交易对信息
	UpdatedAt  time.Time    // 更新时间
}

// SymbolInfo represents symbol information
type SymbolInfo struct {
	Symbol         string    // 交易对
	BaseAsset      string    // 基础资产
	QuoteAsset     string    // 计价资产
	Status         string    // 状态
	MinPrice       float64   // 最小价格
	MaxPrice       float64   // 最大价格
	TickSize       float64   // 价格精度
	MinQty         float64   // 最小数量
	MaxQty         float64   // 最大数量
	StepSize       float64   // 数量精度
	MinNotional    float64   // 最小名义价值
	ContractSize   float64   // 合约大小
	ContractType   string    // 合约类型
	ExpiryDate     time.Time // 到期时间
	StrikePrice    float64   // 行权价
	UnderlyingType string    // 标的类型
	UpdatedAt      time.Time // 更新时间

	// Additional fields for compatibility
	PricePrecision    int     // 价格精度位数
	QuantityPrecision int     // 数量精度位数
	PriceTickSize     float64 // 价格最小变动单位
	MinQuantity       float64 // 最小数量
	MaxQuantity       float64 // 最大数量
	QuantityStepSize  float64 // 数量步长
}

// Position represents a trading position
type Position struct {
	Symbol            string    // 交易对
	Side              string    // 方向 (LONG/SHORT)
	Size              float64   // 持仓大小
	Notional          float64   // 名义价值
	EntryPrice        float64   // 开仓价格
	MarkPrice         float64   // 标记价格
	UnrealizedPnL     float64   // 未实现盈亏
	RealizedPnL       float64   // 已实现盈亏
	Leverage          int       // 杠杆倍数
	MarginType        string    // 保证金类型
	IsolatedMargin    float64   // 逐仓保证金
	MaintenanceMargin float64   // 维持保证金
	LiquidationPrice  float64   // 强平价格
	UpdatedAt         time.Time // 更新时间

	// Additional fields for compatibility
	Quantity float64 // 持仓数量
	LiqPrice float64 // 强平价格别名
}

// MarginType represents margin type
type MarginType string

const (
	MarginTypeIsolated MarginType = "ISOLATED"
	MarginTypeCross    MarginType = "CROSSED"
)

// OrderRequest represents an order request
type OrderRequest struct {
	Symbol         string  // 交易对
	Side           string  // 方向 (BUY/SELL)
	Type           string  // 订单类型 (MARKET/LIMIT/STOP)
	Quantity       float64 // 数量
	Price          float64 // 价格
	StopPrice      float64 // 止损价格
	TimeInForce    string  // 有效期 (GTC/IOC/FOK)
	ReduceOnly     bool    // 仅减仓
	CloseOnTrigger bool    // 触发时平仓
	ClientOrderID  string  // 客户端订单ID

	// Additional fields for compatibility
	PostOnly bool // 仅挂单
}

// OrderResponse represents an order response
type OrderResponse struct {
	OrderID            string    // 订单ID
	ClientOrderID      string    // 客户端订单ID
	Symbol             string    // 交易对
	Status             string    // 状态
	Side               string    // 方向
	Type               string    // 订单类型
	Quantity           float64   // 数量
	Price              float64   // 价格
	ExecutedQty        float64   // 已执行数量
	CumulativeQuoteQty float64   // 累计成交金额
	TimeInForce        string    // 有效期
	Time               time.Time // 时间
	UpdatedTime        time.Time // 更新时间

	// Additional fields for compatibility
	Success bool   // 是否成功
	Error   string // 错误信息
	Order   *Order // 订单信息
}

// OrderCancelRequest represents an order cancel request
type OrderCancelRequest struct {
	Symbol           string // 交易对
	OrderID          string // 订单ID
	ClientOrderID    string // 客户端订单ID
	NewClientOrderID string // 新客户端订单ID
}

// Order represents an order
type Order struct {
	OrderID            string    // 订单ID
	ClientOrderID      string    // 客户端订单ID
	Symbol             string    // 交易对
	Status             string    // 状态
	Side               string    // 方向
	Type               string    // 订单类型
	Quantity           float64   // 数量
	Price              float64   // 价格
	ExecutedQty        float64   // 已执行数量
	CumulativeQuoteQty float64   // 累计成交金额
	TimeInForce        string    // 有效期
	Time               time.Time // 时间
	UpdatedTime        time.Time // 更新时间

	// Additional fields for compatibility
	ID         string // 订单ID别名
	ExchangeID string // 交易所订单ID

	// Additional fields for order management
	FilledQty    float64   // 已成交数量
	RemainingQty float64   // 剩余数量
	AvgPrice     float64   // 平均成交价格
	Fee          float64   // 手续费
	FeeCurrency  string    // 手续费币种
	CreatedAt    time.Time // 创建时间
	UpdatedAt    time.Time // 更新时间
}

// RiskLimit represents a single risk limit
type RiskLimit struct {
	Symbol           string  // 交易对
	MaxLeverage      int     // 最大杠杆
	MaxPositionValue float64 // 最大持仓价值
	MaxOrderValue    float64 // 最大订单价值
	MinOrderValue    float64 // 最小订单价值
	MaxOrderQty      float64 // 最大订单数量
	MinOrderQty      float64 // 最小订单数量
}

// RiskLimits represents risk limits for a symbol
type RiskLimits struct {
	Symbol           string  // 交易对
	MaxLeverage      int     // 最大杠杆
	MaxPositionValue float64 // 最大持仓价值
	MaxOrderValue    float64 // 最大订单价值
	MinOrderValue    float64 // 最小订单价值
	MaxOrderQty      float64 // 最大订单数量
	MinOrderQty      float64 // 最小订单数量
}

// RateLimit represents rate limit information
type RateLimit struct {
	RateLimitType string // 限制类型
	Interval      string // 时间间隔
	IntervalNum   int    // 间隔数量
	Limit         int    // 限制数量
}

// OrderSide represents order side
type OrderSide string

const (
	OrderSideBuy  OrderSide = "BUY"
	OrderSideSell OrderSide = "SELL"
)

// OrderType represents order type
type OrderType string

const (
	OrderTypeMarket OrderType = "MARKET"
	OrderTypeLimit  OrderType = "LIMIT"
	OrderTypeStop   OrderType = "STOP"
)

// OrderStatus represents order status
type OrderStatus string

const (
	OrderStatusNew             OrderStatus = "NEW"
	OrderStatusPending         OrderStatus = "PENDING"
	OrderStatusPartiallyFilled OrderStatus = "PARTIALLY_FILLED"
	OrderStatusFilled          OrderStatus = "FILLED"
	OrderStatusCancelled       OrderStatus = "CANCELLED"
	OrderStatusRejected        OrderStatus = "REJECTED"
)

// PositionSide represents position side
type PositionSide string

const (
	PositionSideLong  PositionSide = "LONG"
	PositionSideShort PositionSide = "SHORT"
)

// Trade represents a trade
type Trade struct {
	ID          string    // 交易ID
	Symbol      string    // 交易对
	Price       float64   // 价格
	Quantity    float64   // 数量
	Side        string    // 方向 (BUY/SELL)
	Time        time.Time // 时间
	IsMaker     bool      // 是否为挂单方
	Fee         float64   // 手续费
	FeeCurrency string    // 手续费币种
}
