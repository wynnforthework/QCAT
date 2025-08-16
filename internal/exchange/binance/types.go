package binance

// API endpoints
const (
	// Public endpoints
	MethodExchangeInfo = "/fapi/v1/exchangeInfo"
	MethodTime         = "/fapi/v1/time"
	MethodTickerPrice  = "/fapi/v1/ticker/price"
	MethodTicker24hr   = "/fapi/v1/ticker/24hr"
	MethodDepth        = "/fapi/v1/depth"
	MethodKlines       = "/fapi/v1/klines"
	MethodTrades       = "/fapi/v1/trades"
	
	// Account endpoints
	MethodAccount      = "/fapi/v2/account"
	MethodBalance      = "/fapi/v2/balance"
	MethodPositions    = "/fapi/v2/positionRisk"
	MethodPosition     = "/fapi/v2/positionRisk"
	
	// Trading endpoints
	MethodOrder        = "/fapi/v1/order"
	MethodCancelOrder  = "/fapi/v1/order"
	MethodCancelAll    = "/fapi/v1/allOpenOrders"
	MethodOpenOrders   = "/fapi/v1/openOrders"
	MethodAllOrders    = "/fapi/v1/allOrders"
	
	// Leverage and margin
	MethodLeverage     = "/fapi/v1/leverage"
	MethodMarginType   = "/fapi/v1/marginType"
	
	// Risk management
	MethodPositionMode = "/fapi/v1/positionSide/dual"
)

// BinanceResponse represents a standard Binance API response
type BinanceResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// Error returns the error message
func (r *BinanceResponse) Error() string {
	return r.Msg
}

// ExchangeInfo represents exchange information
type ExchangeInfo struct {
	Timezone   string       `json:"timezone"`
	ServerTime int64        `json:"serverTime"`
	RateLimits []RateLimit  `json:"rateLimits"`
	Symbols    []SymbolInfo `json:"symbols"`
}

// RateLimit represents rate limit information
type RateLimit struct {
	RateLimitType string `json:"rateLimitType"`
	Interval      string `json:"interval"`
	IntervalNum   int    `json:"intervalNum"`
	Limit         int    `json:"limit"`
}

// SymbolInfo represents symbol information
type SymbolInfo struct {
	Symbol                string   `json:"symbol"`
	Status                string   `json:"status"`
	BaseAsset             string   `json:"baseAsset"`
	BaseAssetPrecision    int      `json:"baseAssetPrecision"`
	QuoteAsset            string   `json:"quoteAsset"`
	QuoteAssetPrecision   int      `json:"quoteAssetPrecision"`
	PricePrecision        int      `json:"pricePrecision"`
	QuantityPrecision     int      `json:"quantityPrecision"`
	Filters               []Filter `json:"filters"`
}

// Filter represents a symbol filter
type Filter struct {
	FilterType string `json:"filterType"`
	MinPrice   string `json:"minPrice,omitempty"`
	MaxPrice   string `json:"maxPrice,omitempty"`
	TickSize   string `json:"tickSize,omitempty"`
	MinQty     string `json:"minQty,omitempty"`
	MaxQty     string `json:"maxQty,omitempty"`
	StepSize   string `json:"stepSize,omitempty"`
	MinNotional string `json:"minNotional,omitempty"`
}

// AccountInfo represents account information
type AccountInfo struct {
	FeeTier                     int     `json:"feeTier"`
	CanTrade                    bool    `json:"canTrade"`
	CanDeposit                  bool    `json:"canDeposit"`
	CanWithdraw                 bool    `json:"canWithdraw"`
	UpdateTime                  int64   `json:"updateTime"`
	TotalInitialMargin          string  `json:"totalInitialMargin"`
	TotalMaintMargin            string  `json:"totalMaintMargin"`
	TotalWalletBalance          string  `json:"totalWalletBalance"`
	TotalUnrealizedProfit       string  `json:"totalUnrealizedProfit"`
	TotalMarginBalance          string  `json:"totalMarginBalance"`
	TotalPositionInitialMargin  string  `json:"totalPositionInitialMargin"`
	TotalOpenOrderInitialMargin string  `json:"totalOpenOrderInitialMargin"`
	TotalCrossWalletBalance     string  `json:"totalCrossWalletBalance"`
	TotalCrossUnPnl             string  `json:"totalCrossUnPnl"`
	AvailableBalance            string  `json:"availableBalance"`
	MaxWithdrawAmount           string  `json:"maxWithdrawAmount"`
	Assets                      []Asset `json:"assets"`
	Positions                   []Position `json:"positions"`
}

// Asset represents an account asset
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
	MarginAvailable        bool   `json:"marginAvailable"`
	UpdateTime             int64  `json:"updateTime"`
}

// Position represents a position
type Position struct {
	Symbol                 string `json:"symbol"`
	PositionAmt            string `json:"positionAmt"`
	EntryPrice             string `json:"entryPrice"`
	MarkPrice              string `json:"markPrice"`
	UnrealizedProfit       string `json:"unRealizedProfit"`
	LiquidationPrice       string `json:"liquidationPrice"`
	Leverage               string `json:"leverage"`
	MaxNotionalValue       string `json:"maxNotionalValue"`
	MarginType             string `json:"marginType"`
	IsolatedMargin         string `json:"isolatedMargin"`
	IsAutoAddMargin        string `json:"isAutoAddMargin"`
	PositionSide           string `json:"positionSide"`
	Notional               string `json:"notional"`
	IsolatedWallet         string `json:"isolatedWallet"`
	UpdateTime             int64  `json:"updateTime"`
	Isolated               bool   `json:"isolated"`
	AdlQuantile            int    `json:"adlQuantile"`
}

// Order represents an order
type Order struct {
	OrderID       int64  `json:"orderId"`
	Symbol        string `json:"symbol"`
	Status        string `json:"status"`
	ClientOrderID string `json:"clientOrderId"`
	Price         string `json:"price"`
	AvgPrice      string `json:"avgPrice"`
	OrigQty       string `json:"origQty"`
	ExecutedQty   string `json:"executedQty"`
	CumQty        string `json:"cumQty"`
	CumQuote      string `json:"cumQuote"`
	TimeInForce   string `json:"timeInForce"`
	Type          string `json:"type"`
	ReduceOnly    bool   `json:"reduceOnly"`
	ClosePosition bool   `json:"closePosition"`
	Side          string `json:"side"`
	PositionSide  string `json:"positionSide"`
	StopPrice     string `json:"stopPrice"`
	WorkingType   string `json:"workingType"`
	PriceProtect  bool   `json:"priceProtect"`
	OrigType      string `json:"origType"`
	Time          int64  `json:"time"`
	UpdateTime    int64  `json:"updateTime"`
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
	BidPrice           string `json:"bidPrice"`
	BidQty             string `json:"bidQty"`
	AskPrice           string `json:"askPrice"`
	AskQty             string `json:"askQty"`
	OpenPrice          string `json:"openPrice"`
	HighPrice          string `json:"highPrice"`
	LowPrice           string `json:"lowPrice"`
	Volume             string `json:"volume"`
	QuoteVolume        string `json:"quoteVolume"`
	OpenTime           int64  `json:"openTime"`
	CloseTime          int64  `json:"closeTime"`
	Count              int64  `json:"count"`
}

// Balance represents account balance
type Balance struct {
	AccountAlias       string `json:"accountAlias"`
	Asset              string `json:"asset"`
	Balance            string `json:"balance"`
	CrossWalletBalance string `json:"crossWalletBalance"`
	CrossUnPnl         string `json:"crossUnPnl"`
	AvailableBalance   string `json:"availableBalance"`
	MaxWithdrawAmount  string `json:"maxWithdrawAmount"`
	MarginAvailable    bool   `json:"marginAvailable"`
	UpdateTime         int64  `json:"updateTime"`
}

// Trade represents a trade
type Trade struct {
	ID           int64  `json:"id"`
	Price        string `json:"price"`
	Qty          string `json:"qty"`
	QuoteQty     string `json:"quoteQty"`
	Time         int64  `json:"time"`
	IsBuyerMaker bool   `json:"isBuyerMaker"`
	IsBestMatch  bool   `json:"isBestMatch"`
}

// Kline represents a kline/candlestick
type Kline struct {
	OpenTime                 int64  `json:"openTime"`
	Open                     string `json:"open"`
	High                     string `json:"high"`
	Low                      string `json:"low"`
	Close                    string `json:"close"`
	Volume                   string `json:"volume"`
	CloseTime                int64  `json:"closeTime"`
	QuoteAssetVolume         string `json:"quoteAssetVolume"`
	NumberOfTrades           int64  `json:"numberOfTrades"`
	TakerBuyBaseAssetVolume  string `json:"takerBuyBaseAssetVolume"`
	TakerBuyQuoteAssetVolume string `json:"takerBuyQuoteAssetVolume"`
}

// OrderBook represents an order book
type OrderBook struct {
	LastUpdateID int64      `json:"lastUpdateId"`
	MessageTime  int64      `json:"E"`
	TransactTime int64      `json:"T"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
}

// MarkPrice represents mark price and funding rate
type MarkPrice struct {
	Symbol               string `json:"symbol"`
	MarkPrice            string `json:"markPrice"`
	IndexPrice           string `json:"indexPrice"`
	EstimatedSettlePrice string `json:"estimatedSettlePrice"`
	LastFundingRate      string `json:"lastFundingRate"`
	NextFundingTime      int64  `json:"nextFundingTime"`
	InterestRate         string `json:"interestRate"`
	Time                 int64  `json:"time"`
}

// OpenInterest represents open interest
type OpenInterest struct {
	OpenInterest string `json:"openInterest"`
	Symbol       string `json:"symbol"`
	Time         int64  `json:"time"`
}

// LeverageResponse represents leverage change response
type LeverageResponse struct {
	Leverage         int    `json:"leverage"`
	MaxNotionalValue string `json:"maxNotionalValue"`
	Symbol           string `json:"symbol"`
}

// MarginTypeResponse represents margin type change response
type MarginTypeResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// Error codes
const (
	CodeSuccess                = 200
	CodeInvalidSignature       = -1022
	CodeInsufficientBalance    = -2019
	CodeOrderWouldImmediatelyMatch = -2021
	CodeReduceOnlyReject       = -2022
	CodeUserDataStreamExpired  = -2023
	CodeInvalidListenKey       = -2024
	CodeMoreThan24HrsBetweenStartEndTime = -1127
	CodeInvalidParameter       = -1102
	CodeMandatoryParamEmptyOrMalformed = -1100
	CodeUnknownParam           = -1101
	CodeInvalidSymbol          = -1121
	CodeInvalidPeriod          = -1120
	CodeInvalidTimeInForce     = -1145
	CodeInvalidOrderType       = -1116
	CodeInvalidSide            = -1117
	CodeEmptyNewClOrdID        = -1118
	CodeBadAPIKeyFmt           = -2015
	CodeNoSuchOrder            = -2013
	CodeBadSymbol              = -1121
	CodeBadRecvWindow          = -1021
	CodeBadTimestamp           = -1021
)