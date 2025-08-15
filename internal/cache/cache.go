package cache

import (
	"context"
	"time"
)

// Cacher defines the interface for cache operations
type Cacher interface {
	// Market Data
	SetMarketTick(ctx context.Context, symbol string, data interface{}, expiration time.Duration) error
	GetMarketTick(ctx context.Context, symbol string, dest interface{}) error
	SetOrderBook(ctx context.Context, symbol string, data interface{}, expiration time.Duration) error
	GetOrderBook(ctx context.Context, symbol string, dest interface{}) error
	SetFundingRate(ctx context.Context, symbol string, data interface{}, expiration time.Duration) error
	GetFundingRate(ctx context.Context, symbol string, dest interface{}) error

	// Strategy
	PushSignal(ctx context.Context, signal interface{}) error
	PopSignal(ctx context.Context, dest interface{}) error
	SetPositionState(ctx context.Context, strategy, symbol string, data interface{}, expiration time.Duration) error
	GetPositionState(ctx context.Context, strategy, symbol string, dest interface{}) error

	// Locks
	AcquireLock(ctx context.Context, name string, expiration time.Duration) (bool, error)
	ReleaseLock(ctx context.Context, name string) error

	// Rate Limiting
	CheckRateLimit(ctx context.Context, exchange string, limit int, window time.Duration) (bool, error)

	// Hot Market
	SetHotScore(ctx context.Context, symbol string, score float64, expiration time.Duration) error
	GetHotScore(ctx context.Context, symbol string) (float64, error)
}

// NewCacher creates a new cache instance based on configuration
func NewCacher(cfg *Config) (Cacher, error) {
	if cfg.Enabled {
		return NewCache(cfg)
	}
	return NewMemoryCache(), nil
}
