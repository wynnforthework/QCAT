package index

import "time"

// Price represents an index price
type Price struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Timestamp time.Time `json:"timestamp"`
}

// Component represents an index component
type Component struct {
	Symbol    string    `json:"symbol"`
	Weight    float64   `json:"weight"`
	Price     float64   `json:"price"`
	Quantity  float64   `json:"quantity"`
	Timestamp time.Time `json:"timestamp"`
}

// Stats represents index price statistics
type Stats struct {
	Symbol       string    `json:"symbol"`
	CurrentPrice float64   `json:"current_price"`
	Mean24h      float64   `json:"mean_24h"`
	StdDev24h    float64   `json:"std_dev_24h"`
	Min24h       float64   `json:"min_24h"`
	Max24h       float64   `json:"max_24h"`
	UpdatedAt    time.Time `json:"updated_at"`
	
	// Additional fields for compatibility
	High24h      float64   `json:"high_24h"`
	Low24h       float64   `json:"low_24h"`
	Change24h    float64   `json:"change_24h"`
	ChangeP24h   float64   `json:"change_p_24h"`
}
