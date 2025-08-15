package cache

import (
	"context"
	"time"
)

// Cacher defines the interface for cache operations
type Cacher interface {
	// Basic operations
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)

	// Hash operations
	HGet(ctx context.Context, key, field string, dest interface{}) error
	HSet(ctx context.Context, key, field string, value interface{}) error
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	HDel(ctx context.Context, key string, fields ...string) error

	// List operations
	LPush(ctx context.Context, key string, values ...interface{}) error
	RPush(ctx context.Context, key string, values ...interface{}) error
	LPop(ctx context.Context, key string, dest interface{}) error
	RPop(ctx context.Context, key string, dest interface{}) error
	LRange(ctx context.Context, key string, start, stop int64) ([]string, error)

	// Set operations
	SAdd(ctx context.Context, key string, members ...interface{}) error
	SRem(ctx context.Context, key string, members ...interface{}) error
	SMembers(ctx context.Context, key string) ([]string, error)
	SIsMember(ctx context.Context, key string, member interface{}) (bool, error)

	// Sorted set operations
	ZAdd(ctx context.Context, key string, score float64, member interface{}) error
	ZRange(ctx context.Context, key string, start, stop int64) ([]string, error)
	ZRangeByScore(ctx context.Context, key string, min, max string) ([]string, error)
	ZRem(ctx context.Context, key string, members ...interface{}) error

	// Expiration
	Expire(ctx context.Context, key string, expiration time.Duration) error
	TTL(ctx context.Context, key string) (time.Duration, error)

	// Utility
	Flush(ctx context.Context) error
	Close() error

	// Specific methods for funding rates
	SetFundingRate(ctx context.Context, symbol string, rate interface{}, expiration time.Duration) error
	GetFundingRate(ctx context.Context, symbol string, dest interface{}) error

	// Specific methods for index prices
	SetIndexPrice(ctx context.Context, symbol string, price interface{}, expiration time.Duration) error
	GetIndexPrice(ctx context.Context, symbol string, dest interface{}) error

	// Rate limiting
	CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error)

	// Order book operations
	SetOrderBook(ctx context.Context, symbol string, snapshot interface{}, expiration time.Duration) error
	GetOrderBook(ctx context.Context, symbol string, dest interface{}) error
}
