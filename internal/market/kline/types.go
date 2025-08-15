package kline

import (
	"time"
)

// Interval represents a candlestick interval
type Interval string

const (
	Interval1m  Interval = "1m"
	Interval3m  Interval = "3m"
	Interval5m  Interval = "5m"
	Interval15m Interval = "15m"
	Interval30m Interval = "30m"
	Interval1h  Interval = "1h"
	Interval2h  Interval = "2h"
	Interval4h  Interval = "4h"
	Interval6h  Interval = "6h"
	Interval8h  Interval = "8h"
	Interval12h Interval = "12h"
	Interval1d  Interval = "1d"
	Interval3d  Interval = "3d"
	Interval1w  Interval = "1w"
	Interval1M  Interval = "1M"
)

// Kline represents a candlestick
type Kline struct {
	Symbol    string    `json:"symbol"`
	Interval  Interval  `json:"interval"`
	OpenTime  time.Time `json:"open_time"`
	CloseTime time.Time `json:"close_time"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
	Complete  bool      `json:"complete"`
	
	// Aliases for backward compatibility
	ClosePrice float64 `json:"close_price"`
	HighPrice  float64 `json:"high_price"`
	LowPrice   float64 `json:"low_price"`
}

// NewKline creates a new kline
func NewKline(symbol string, interval Interval, openTime time.Time) *Kline {
	return &Kline{
		Symbol:    symbol,
		Interval:  interval,
		OpenTime:  openTime,
		CloseTime: getCloseTime(openTime, interval),
		Complete:  false,
	}
}

// Update updates the kline with a new trade
func (k *Kline) Update(price, volume float64, timestamp time.Time) {
	if k.Open == 0 {
		k.Open = price
	}
	if k.High == 0 || price > k.High {
		k.High = price
		k.HighPrice = price
	}
	if k.Low == 0 || price < k.Low {
		k.Low = price
		k.LowPrice = price
	}
	k.Close = price
	k.ClosePrice = price
	k.Volume += volume

	if timestamp.After(k.CloseTime) {
		k.Complete = true
	}
}

// getCloseTime calculates the close time based on the interval
func getCloseTime(openTime time.Time, interval Interval) time.Time {
	switch interval {
	case Interval1m:
		return openTime.Add(time.Minute)
	case Interval3m:
		return openTime.Add(3 * time.Minute)
	case Interval5m:
		return openTime.Add(5 * time.Minute)
	case Interval15m:
		return openTime.Add(15 * time.Minute)
	case Interval30m:
		return openTime.Add(30 * time.Minute)
	case Interval1h:
		return openTime.Add(time.Hour)
	case Interval2h:
		return openTime.Add(2 * time.Hour)
	case Interval4h:
		return openTime.Add(4 * time.Hour)
	case Interval6h:
		return openTime.Add(6 * time.Hour)
	case Interval8h:
		return openTime.Add(8 * time.Hour)
	case Interval12h:
		return openTime.Add(12 * time.Hour)
	case Interval1d:
		return openTime.Add(24 * time.Hour)
	case Interval3d:
		return openTime.Add(72 * time.Hour)
	case Interval1w:
		return openTime.Add(168 * time.Hour)
	case Interval1M:
		return openTime.AddDate(0, 1, 0)
	default:
		return openTime.Add(time.Minute)
	}
}

// GetIntervalDuration returns the duration of an interval
func GetIntervalDuration(interval Interval) time.Duration {
	switch interval {
	case Interval1m:
		return time.Minute
	case Interval3m:
		return 3 * time.Minute
	case Interval5m:
		return 5 * time.Minute
	case Interval15m:
		return 15 * time.Minute
	case Interval30m:
		return 30 * time.Minute
	case Interval1h:
		return time.Hour
	case Interval2h:
		return 2 * time.Hour
	case Interval4h:
		return 4 * time.Hour
	case Interval6h:
		return 6 * time.Hour
	case Interval8h:
		return 8 * time.Hour
	case Interval12h:
		return 12 * time.Hour
	case Interval1d:
		return 24 * time.Hour
	case Interval3d:
		return 72 * time.Hour
	case Interval1w:
		return 168 * time.Hour
	case Interval1M:
		return 30 * 24 * time.Hour // Approximate
	default:
		return time.Minute
	}
}
