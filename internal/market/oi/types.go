package oi

import (
	"time"
)

// OpenInterest represents open interest data
type OpenInterest struct {
	Symbol    string    `json:"symbol"`
	Value     float64   `json:"value"`
	Notional  float64   `json:"notional"`
	Timestamp time.Time `json:"timestamp"`
}

// Stats represents open interest statistics
type Stats struct {
	Symbol          string    `json:"symbol"`
	CurrentOI       float64   `json:"current_oi"`
	CurrentNotional float64   `json:"current_notional"`
	Change24h       float64   `json:"change_24h"`
	ChangeP24h      float64   `json:"change_p_24h"`
	High24h         float64   `json:"high_24h"`
	Low24h          float64   `json:"low_24h"`
	Mean24h         float64   `json:"mean_24h"`
	StdDev24h       float64   `json:"std_dev_24h"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// History represents historical open interest data
type History struct {
	Symbol    string    `json:"symbol"`
	Value     float64   `json:"value"`
	Notional  float64   `json:"notional"`
	Timestamp time.Time `json:"timestamp"`
}
