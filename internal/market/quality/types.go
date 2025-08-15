package quality

import (
	"time"
)

// DataType represents the type of market data
type DataType string

const (
	DataTypeTicker    DataType = "ticker"
	DataTypeOrderBook DataType = "orderbook"
	DataTypeTrade     DataType = "trade"
	DataTypeKline     DataType = "kline"
	DataTypeFunding   DataType = "funding"
	DataTypeOI        DataType = "open_interest"
	DataTypeIndex     DataType = "index"
)

// Metric represents a data quality metric
type Metric struct {
	Symbol           string    `json:"symbol"`
	DataType         DataType  `json:"data_type"`
	UpdateFrequency  float64   `json:"update_frequency"`
	LastUpdate       time.Time `json:"last_update"`
	MissingDataCount int       `json:"missing_data_count"`
	ErrorCount       int       `json:"error_count"`
	Latency          float64   `json:"latency"`
	Staleness        float64   `json:"staleness"`
	Completeness     float64   `json:"completeness"`
	Accuracy         float64   `json:"accuracy"`
}

// Alert represents a data quality alert
type Alert struct {
	Symbol      string    `json:"symbol"`
	DataType    DataType  `json:"data_type"`
	Level       string    `json:"level"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
	MetricValue float64   `json:"metric_value"`
	Threshold   float64   `json:"threshold"`
}

// Threshold represents alert thresholds for data quality metrics
type Threshold struct {
	MaxUpdateInterval time.Duration `json:"max_update_interval"`
	MaxMissingData    int           `json:"max_missing_data"`
	MaxErrors         int           `json:"max_errors"`
	MaxLatency        float64       `json:"max_latency"`
	MaxStaleness      float64       `json:"max_staleness"`
	MinCompleteness   float64       `json:"min_completeness"`
	MinAccuracy       float64       `json:"min_accuracy"`
}
