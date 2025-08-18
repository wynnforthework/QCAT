package types

import "time"

// OrderBook represents a market order book
type OrderBook struct {
	Symbol    string    `json:"symbol"`
	Bids      []Level   `json:"bids"`
	Asks      []Level   `json:"asks"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Level represents a price level in the order book
type Level struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
}

// Trade represents a single executed trade
type Trade struct {
	ID        string    `json:"id"`
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Quantity  float64   `json:"quantity"`
	Side      string    `json:"side"` // "BUY" or "SELL"
	Fee       float64   `json:"fee"`
	FeeCoin   string    `json:"fee_coin"`
	Timestamp time.Time `json:"timestamp"`
}

// Kline represents a candlestick data point
type Kline struct {
	Symbol    string    `json:"symbol"`
	Interval  string    `json:"interval"`
	OpenTime  time.Time `json:"open_time"`
	CloseTime time.Time `json:"close_time"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
	Complete  bool      `json:"complete"`
}

// FundingRate represents the funding rate for a perpetual contract
type FundingRate struct {
	Symbol      string    `json:"symbol"`
	Rate        float64   `json:"rate"`
	NextRate    float64   `json:"next_rate"`
	NextTime    time.Time `json:"next_time"`
	LastUpdated time.Time `json:"last_updated"`
}

// OpenInterest represents the total open interest for a symbol
type OpenInterest struct {
	Symbol    string    `json:"symbol"`
	Value     float64   `json:"value"`    // OI in contracts
	Notional  float64   `json:"notional"` // OI in USD/quote currency
	Timestamp time.Time `json:"timestamp"`
}

// IndexPrice represents the index price for a symbol
type IndexPrice struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Timestamp time.Time `json:"timestamp"`
}

// MarketType represents the type of market
type MarketType string

const (
	MarketTypeSpot    MarketType = "SPOT"
	MarketTypeFutures MarketType = "FUTURES"
	MarketTypeOptions MarketType = "OPTIONS"
)

// WSSubscription represents a WebSocket market data subscription
type WSSubscription struct {
	Symbol     string                 `json:"symbol"`
	MarketType MarketType             `json:"market_type"`
	Channels   []string               `json:"channels"`
	Parameters map[string]interface{} `json:"parameters"`
}

// Ticker represents a ticker update
type Ticker struct {
	Symbol             string    `json:"symbol"`
	PriceChange        float64   `json:"price_change"`
	PriceChangePercent float64   `json:"price_change_percent"`
	WeightedAvgPrice   float64   `json:"weighted_avg_price"`
	PrevClosePrice     float64   `json:"prev_close_price"`
	LastPrice          float64   `json:"last_price"`
	LastQty            float64   `json:"last_qty"`
	BidPrice           float64   `json:"bid_price"`
	BidQty             float64   `json:"bid_qty"`
	AskPrice           float64   `json:"ask_price"`
	AskQty             float64   `json:"ask_qty"`
	OpenPrice          float64   `json:"open_price"`
	HighPrice          float64   `json:"high_price"`
	LowPrice           float64   `json:"low_price"`
	Volume             float64   `json:"volume"`
	QuoteVolume        float64   `json:"quote_volume"`
	OpenTime           time.Time `json:"open_time"`
	CloseTime          time.Time `json:"close_time"`
	Count              int64     `json:"count"`
}

// MarketDataHandler represents a handler for market data messages
type MarketDataHandler func(interface{}) error
