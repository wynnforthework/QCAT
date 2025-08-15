package funding

import "time"

// Rate represents a funding rate
type Rate struct {
	Symbol      string    `json:"symbol"`
	Rate        float64   `json:"rate"`
	NextRate    float64   `json:"next_rate"`
	NextTime    time.Time `json:"next_time"`
	LastUpdated time.Time `json:"last_updated"`
}

// Stats represents funding rate statistics
type Stats struct {
	Symbol          string    `json:"symbol"`
	CurrentRate     float64   `json:"current_rate"`
	PredictedRate   float64   `json:"predicted_rate"`
	NextFundingTime time.Time `json:"next_funding_time"`
	Mean24h         float64   `json:"mean_24h"`
	StdDev24h       float64   `json:"std_dev_24h"`
	Min24h          float64   `json:"min_24h"`
	Max24h          float64   `json:"max_24h"`
	AnnualizedRate  float64   `json:"annualized_rate"`
	UpdatedAt       time.Time `json:"updated_at"`
}
