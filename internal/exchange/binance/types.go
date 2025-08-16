package binance

// API endpoints
const (
	BaseAPIURL     = "https://api.binance.com"
	BaseFuturesURL = "https://fapi.binance.com"
	BaseTestnetURL = "https://testnet.binancefuture.com"
)

// API methods
const (
	MethodPing         = "/fapi/v1/ping"
	MethodTime         = "/fapi/v1/time"
	MethodExchangeInfo = "/fapi/v1/exchangeInfo"
	MethodAccount      = "/fapi/v2/account"
	MethodBalance      = "/fapi/v2/balance"
	MethodPositions    = "/fapi/v2/positionRisk"
	MethodPosition     = "/fapi/v2/positionRisk"
	MethodLeverage     = "/fapi/v1/leverage"
	MethodMarginType   = "/fapi/v1/marginType"
	MethodOrder        = "/fapi/v1/order"
	MethodOpenOrders   = "/fapi/v1/openOrders"
	MethodAllOrders    = "/fapi/v1/allOrders"
	MethodCancelOrder  = "/fapi/v1/order"
	MethodCancelAll    = "/fapi/v1/allOpenOrders"
	MethodTicker24hr   = "/fapi/v1/ticker/24hr"
	MethodTickerPrice  = "/fapi/v1/ticker/price"
)

// BinanceResponse represents a generic Binance API response
type BinanceResponse struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

// ExchangeInfo represents Binance exchange information
type ExchangeInfo struct {
	Timezone        string        `json:"timezone"`
	ServerTime      int64         `json:"serverTime"`
	RateLimits      []RateLimit   `json:"rateLimits"`
	ExchangeFilters []interface{} `json:"exchangeFilters"`
	Symbols         []SymbolInfo  `json:"symbols"`
}

// RateLimit represents a Binance rate limit
type RateLimit struct {
	RateLimitType string `json:"rateLimitType"`
	Interval      string `json:"interval"`
	IntervalNum   int    `json:"intervalNum"`
	Limit         int    `json:"limit"`
}

// SymbolInfo represents Binance symbol information
type SymbolInfo struct {
	Symbol                string   `json:"symbol"`
	Pair                  string   `json:"pair"`
	ContractType          string   `json:"contractType"`
	DeliveryDate          int64    `json:"deliveryDate"`
	OnboardDate           int64    `json:"onboardDate"`
	Status                string   `json:"status"`
	MaintMarginPercent    string   `json:"maintMarginPercent"`
	RequiredMarginPercent string   `json:"requiredMarginPercent"`
	BaseAsset             string   `json:"baseAsset"`
	QuoteAsset            string   `json:"quoteAsset"`
	MarginAsset           string   `json:"marginAsset"`
	PricePrecision        int      `json:"pricePrecision"`
	QuantityPrecision     int      `json:"quantityPrecision"`
	BaseAssetPrecision    int      `json:"baseAssetPrecision"`
	QuotePrecision        int      `json:"quotePrecision"`
	UnderlyingType        string   `json:"underlyingType"`
	UnderlyingSubType     []string `json:"underlyingSubType"`
	SettlePlan            int      `json:"settlePlan"`
	TriggerProtect        string   `json:"triggerProtect"`
	Filters               []Filter `json:"filters"`
	OrderTypes            []string `json:"orderTypes"`
	TimeInForce           []string `json:"timeInForce"`
}

// Filter represents a Binance symbol filter
type Filter struct {
	FilterType     string `json:"filterType"`
	MinPrice       string `json:"minPrice,omitempty"`
	MaxPrice       string `json:"maxPrice,omitempty"`
	TickSize       string `json:"tickSize,omitempty"`
	MinQty         string `json:"minQty,omitempty"`
	MaxQty         string `json:"maxQty,omitempty"`
	StepSize       string `json:"stepSize,omitempty"`
	Limit          int    `json:"limit,omitempty"`
	Notional       string `json:"notional,omitempty"`
	Multiplier     string `json:"multiplier,omitempty"`
	MultiplierUp   string `json:"multiplierUp,omitempty"`
	MultiplierDown string `json:"multiplierDown,omitempty"`
}

// AccountInfo represents Binance account information
type AccountInfo struct {
	FeeTier                     int        `json:"feeTier"`
	CanTrade                    bool       `json:"canTrade"`
	CanDeposit                  bool       `json:"canDeposit"`
	CanWithdraw                 bool       `json:"canWithdraw"`
	UpdateTime                  int64      `json:"updateTime"`
	TotalInitialMargin          string     `json:"totalInitialMargin"`
	TotalMaintMargin            string     `json:"totalMaintMargin"`
	TotalWalletBalance          string     `json:"totalWalletBalance"`
	TotalUnrealizedProfit       string     `json:"totalUnrealizedProfit"`
	TotalMarginBalance          string     `json:"totalMarginBalance"`
	TotalPositionInitialMargin  string     `json:"totalPositionInitialMargin"`
	TotalOpenOrderInitialMargin string     `json:"totalOpenOrderInitialMargin"`
	TotalCrossWalletBalance     string     `json:"totalCrossWalletBalance"`
	TotalCrossUnPnl             string     `json:"totalCrossUnPnl"`
	AvailableBalance            string     `json:"availableBalance"`
	MaxWithdrawAmount           string     `json:"maxWithdrawAmount"`
	Assets                      []Asset    `json:"assets"`
	Positions                   []Position `json:"positions"`
}

// Asset represents a Binance asset
type Asset struct {
	Asset                  string `json:"asset"`
	WalletBalance          string `json:"walletBalance"`
	UnrealizedProfit       string `json:"unrealizedProfit"`
	MarginBalance          string `json:"marginBalance"`
	MaintMargin            string `json:"maintMargin"`
	InitialMargin          string `json:"initialMargin"`
	PositionInitialMargin  string `json:"positionInitialMargin"`
	OpenOrderInitialMargin string `json:"openOrderInitialMargin"`
	CrossWalletBalance     string `json:"crossWalletBalance"`
	CrossUnPnl             string `json:"crossUnPnl"`
	AvailableBalance       string `json:"availableBalance"`
	MaxWithdrawAmount      string `json:"maxWithdrawAmount"`
}

// Position represents a Binance position
type Position struct {
	Symbol                 string `json:"symbol"`
	InitialMargin          string `json:"initialMargin"`
	MaintMargin            string `json:"maintMargin"`
	UnrealizedProfit       string `json:"unrealizedProfit"`
	PositionInitialMargin  string `json:"positionInitialMargin"`
	OpenOrderInitialMargin string `json:"openOrderInitialMargin"`
	Leverage               string `json:"leverage"`
	Isolated               bool   `json:"isolated"`
	EntryPrice             string `json:"entryPrice"`
	MaxNotional            string `json:"maxNotional"`
	PositionSide           string `json:"positionSide"`
	PositionAmt            string `json:"positionAmt"`
	Notional               string `json:"notional"`
	IsolatedWallet         string `json:"isolatedWallet"`
	UpdateTime             int64  `json:"updateTime"`
}

// Order represents a Binance order
type Order struct {
	ClientOrderID string `json:"clientOrderId"`
	CumQty        string `json:"cumQty"`
	CumQuote      string `json:"cumQuote"`
	ExecutedQty   string `json:"executedQty"`
	OrderID       int64  `json:"orderId"`
	AvgPrice      string `json:"avgPrice"`
	OrigQty       string `json:"origQty"`
	Price         string `json:"price"`
	ReduceOnly    bool   `json:"reduceOnly"`
	Side          string `json:"side"`
	PositionSide  string `json:"positionSide"`
	Status        string `json:"status"`
	StopPrice     string `json:"stopPrice"`
	ClosePosition bool   `json:"closePosition"`
	Symbol        string `json:"symbol"`
	TimeInForce   string `json:"timeInForce"`
	Type          string `json:"type"`
	OrigType      string `json:"origType"`
	ActivatePrice string `json:"activatePrice"`
	PriceRate     string `json:"priceRate"`
	UpdateTime    int64  `json:"updateTime"`
	WorkingType   string `json:"workingType"`
	PriceProtect  bool   `json:"priceProtect"`
}

// TickerPrice represents a ticker price
type TickerPrice struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

// Ticker24hr represents 24hr ticker statistics
type Ticker24hr struct {
	Symbol             string `json:"symbol"`
	PriceChange        string `json:"priceChange"`
	PriceChangePercent string `json:"priceChangePercent"`
	WeightedAvgPrice   string `json:"weightedAvgPrice"`
	PrevClosePrice     string `json:"prevClosePrice"`
	LastPrice          string `json:"lastPrice"`
	LastQty            string `json:"lastQty"`
	OpenPrice          string `json:"openPrice"`
	HighPrice          string `json:"highPrice"`
	LowPrice           string `json:"lowPrice"`
	Volume             string `json:"volume"`
	QuoteVolume        string `json:"quoteVolume"`
	OpenTime           int64  `json:"openTime"`
	CloseTime          int64  `json:"closeTime"`
	Count              int64  `json:"count"`
}

// OrderRequest represents a Binance order request
type OrderRequest struct {
	Symbol           string `json:"symbol"`
	Side             string `json:"side"`
	PositionSide     string `json:"positionSide,omitempty"`
	Type             string `json:"type"`
	TimeInForce      string `json:"timeInForce,omitempty"`
	Quantity         string `json:"quantity,omitempty"`
	ReduceOnly       bool   `json:"reduceOnly"`
	Price            string `json:"price,omitempty"`
	NewClientOrderID string `json:"newClientOrderId,omitempty"`
	StopPrice        string `json:"stopPrice,omitempty"`
	ClosePosition    bool   `json:"closePosition,omitempty"`
	ActivationPrice  string `json:"activationPrice,omitempty"`
	CallbackRate     string `json:"callbackRate,omitempty"`
	WorkingType      string `json:"workingType,omitempty"`
	PriceProtect     bool   `json:"priceProtect,omitempty"`
	NewOrderRespType string `json:"newOrderRespType,omitempty"`
	Timestamp        int64  `json:"timestamp"`
}

// LeverageRequest represents a Binance leverage update request
type LeverageRequest struct {
	Symbol    string `json:"symbol"`
	Leverage  int    `json:"leverage"`
	Timestamp int64  `json:"timestamp"`
}

// MarginTypeRequest represents a Binance margin type update request
type MarginTypeRequest struct {
	Symbol     string `json:"symbol"`
	MarginType string `json:"marginType"`
	Timestamp  int64  `json:"timestamp"`
}
