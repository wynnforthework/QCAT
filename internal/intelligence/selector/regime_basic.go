package selector

import (
	"context"
	"time"
)

// BasicRegimeDetector is a stub that returns an empty regime with no features.
type BasicRegimeDetector struct{}

func (b *BasicRegimeDetector) Detect(ctx context.Context, symbol string) (string, map[string]float64, error) {
	return "", map[string]float64{}, nil
}

// MarketContextFactory creates a minimal market context
func MarketContextFactory(symbol string) *MarketContext {
	return &MarketContext{Symbol: symbol, Timestamp: time.Now(), Features: map[string]float64{}}
}


