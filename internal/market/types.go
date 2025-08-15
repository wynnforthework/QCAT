package market

import (
	"time"
)

// MarketType represents the type of market (spot, futures, etc.)
type MarketType string

const (
	MarketTypeSpot    MarketType = "spot"
	MarketTypeFutures MarketType = "futures"
)

// Ticker represents real-time price information
type Ticker struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Volume    float64   `json:"volume"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Timestamp time.Time `json:"timestamp"`
}

// OrderBookLevel represents a price level in the order book
type OrderBookLevel struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
}

// OrderBook represents the full order book
type OrderBook struct {
	Symbol    string           `json:"symbol"`
	Bids      []OrderBookLevel `json:"bids"`
	Asks      []OrderBookLevel `json:"asks"`
	Timestamp time.Time        `json:"timestamp"`
}

// Trade represents a single trade
type Trade struct {
	Symbol    string    `json:"symbol"`
	ID        string    `json:"id"`
	Price     float64   `json:"price"`
	Quantity  float64   `json:"quantity"`
	Side      string    `json:"side"`
	Timestamp time.Time `json:"timestamp"`
}

// Kline represents candlestick data
type Kline struct {
	Symbol    string    `json:"symbol"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
	Interval  string    `json:"interval"`
	Timestamp time.Time `json:"timestamp"`
}

// FundingRate represents funding rate data for perpetual futures
type FundingRate struct {
	Symbol      string    `json:"symbol"`
	Rate        float64   `json:"rate"`
	NextRate    float64   `json:"next_rate"`
	NextTime    time.Time `json:"next_time"`
	LastUpdated time.Time `json:"last_updated"`
}

// OpenInterest represents open interest data
type OpenInterest struct {
	Symbol    string    `json:"symbol"`
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// IndexPrice represents index price data
type IndexPrice struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Timestamp time.Time `json:"timestamp"`
}

// DataQuality represents data quality metrics
type DataQuality struct {
	Symbol           string    `json:"symbol"`
	DataType         string    `json:"data_type"`
	UpdateFrequency  float64   `json:"update_frequency"`
	LastUpdate       time.Time `json:"last_update"`
	MissingDataCount int       `json:"missing_data_count"`
	ErrorCount       int       `json:"error_count"`
}

// Subscription represents a market data subscription
type Subscription struct {
	Symbol     string     `json:"symbol"`
	MarketType MarketType `json:"market_type"`
	Channels   []string   `json:"channels"`
}

// MarketDataHandler is a callback function type for handling market data
type MarketDataHandler func(interface{}) error
