package trade

import (
	"time"
)

// Side represents the trade side
type Side string

const (
	SideBuy  Side = "buy"
	SideSell Side = "sell"
)

// Trade represents a single trade
type Trade struct {
	ID        string    `json:"id"`
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Quantity  float64   `json:"quantity"`
	Side      Side      `json:"side"`
	Timestamp time.Time `json:"timestamp"`
	Fee       float64   `json:"fee"`
	FeeCoin   string    `json:"fee_coin"`
}

// TradeStats represents trade statistics for a symbol
type TradeStats struct {
	Symbol            string    `json:"symbol"`
	LastPrice         float64   `json:"last_price"`
	LastQuantity      float64   `json:"last_quantity"`
	LastTradeTime     time.Time `json:"last_trade_time"`
	Volume24h         float64   `json:"volume_24h"`
	QuoteVolume24h    float64   `json:"quote_volume_24h"`
	PriceChange24h    float64   `json:"price_change_24h"`
	PriceChangeP24h   float64   `json:"price_change_p_24h"`
	HighPrice24h      float64   `json:"high_price_24h"`
	LowPrice24h       float64   `json:"low_price_24h"`
	NumberOfTrades24h int64     `json:"number_of_trades_24h"`
}

// TradeAggregation represents aggregated trade data
type TradeAggregation struct {
	Symbol    string    `json:"symbol"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
	VWAP      float64   `json:"vwap"`
	NumTrades int64     `json:"num_trades"`
}
