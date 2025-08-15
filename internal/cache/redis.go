package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config holds Redis configuration
type Config struct {
	Enabled  bool
	Host     string
	Port     int
	Password string
	DB       int
	PoolSize int
}

// Cache represents a Redis cache instance
type Cache struct {
	client *redis.Client
}

// NewCache creates a new Redis cache instance
func NewCache(cfg *Config) (*Cache, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Cache{client: client}, nil
}

// Close closes the Redis connection
func (c *Cache) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// Market Data Cache Keys
const (
	KeyTick    = "md:tick:%s"    // Real-time ticker
	KeyBook    = "md:book:%s"    // Order book
	KeyFunding = "md:funding:%s" // Funding rate
)

// Strategy Cache Keys
const (
	KeySignalQueue = "sig:queue"       // Strategy signal queue
	KeyPendingOrd  = "ord:pending"     // Pending orders
	KeyPosState    = "state:pos:%s:%s" // Position state (strategy:symbol)
)

// Lock Keys
const (
	KeyLock = "lock:%s" // Distributed lock
)

// Rate Limit Keys
const (
	KeyRateLimit = "rate:%s" // Rate limit by exchange
)

// Hot Market Keys
const (
	KeyHotScore = "hot:score:%s" // Hot market score by symbol
)

// SetMarketTick stores real-time market tick data
func (c *Cache) SetMarketTick(ctx context.Context, symbol string, data interface{}, expiration time.Duration) error {
	return c.set(ctx, fmt.Sprintf(KeyTick, symbol), data, expiration)
}

// GetMarketTick retrieves real-time market tick data
func (c *Cache) GetMarketTick(ctx context.Context, symbol string, dest interface{}) error {
	return c.get(ctx, fmt.Sprintf(KeyTick, symbol), dest)
}

// SetOrderBook stores order book data
func (c *Cache) SetOrderBook(ctx context.Context, symbol string, data interface{}, expiration time.Duration) error {
	return c.set(ctx, fmt.Sprintf(KeyBook, symbol), data, expiration)
}

// GetOrderBook retrieves order book data
func (c *Cache) GetOrderBook(ctx context.Context, symbol string, dest interface{}) error {
	return c.get(ctx, fmt.Sprintf(KeyBook, symbol), dest)
}

// SetFundingRate stores funding rate data
func (c *Cache) SetFundingRate(ctx context.Context, symbol string, data interface{}, expiration time.Duration) error {
	return c.set(ctx, fmt.Sprintf(KeyFunding, symbol), data, expiration)
}

// GetFundingRate retrieves funding rate data
func (c *Cache) GetFundingRate(ctx context.Context, symbol string, dest interface{}) error {
	return c.get(ctx, fmt.Sprintf(KeyFunding, symbol), dest)
}

// PushSignal adds a signal to the queue
func (c *Cache) PushSignal(ctx context.Context, signal interface{}) error {
	data, err := json.Marshal(signal)
	if err != nil {
		return fmt.Errorf("failed to marshal signal: %w", err)
	}
	return c.client.LPush(ctx, KeySignalQueue, data).Err()
}

// PopSignal retrieves and removes a signal from the queue
func (c *Cache) PopSignal(ctx context.Context, dest interface{}) error {
	data, err := c.client.RPop(ctx, KeySignalQueue).Bytes()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to pop signal: %w", err)
	}
	return json.Unmarshal(data, dest)
}

// SetPositionState stores position state
func (c *Cache) SetPositionState(ctx context.Context, strategy, symbol string, data interface{}, expiration time.Duration) error {
	return c.set(ctx, fmt.Sprintf(KeyPosState, strategy, symbol), data, expiration)
}

// GetPositionState retrieves position state
func (c *Cache) GetPositionState(ctx context.Context, strategy, symbol string, dest interface{}) error {
	return c.get(ctx, fmt.Sprintf(KeyPosState, strategy, symbol), dest)
}

// AcquireLock attempts to acquire a distributed lock
func (c *Cache) AcquireLock(ctx context.Context, name string, expiration time.Duration) (bool, error) {
	return c.client.SetNX(ctx, fmt.Sprintf(KeyLock, name), "1", expiration).Result()
}

// ReleaseLock releases a distributed lock
func (c *Cache) ReleaseLock(ctx context.Context, name string) error {
	return c.client.Del(ctx, fmt.Sprintf(KeyLock, name)).Err()
}

// CheckRateLimit checks and updates rate limit
func (c *Cache) CheckRateLimit(ctx context.Context, exchange string, limit int, window time.Duration) (bool, error) {
	key := fmt.Sprintf(KeyRateLimit, exchange)
	pipe := c.client.Pipeline()

	now := time.Now().UnixNano()
	windowStart := now - window.Nanoseconds()

	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})
	pipe.ZCard(ctx, key)

	cmds, err := pipe.Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to check rate limit: %w", err)
	}

	count := cmds[2].(*redis.IntCmd).Val()
	return count <= int64(limit), nil
}

// SetHotScore stores hot market score
func (c *Cache) SetHotScore(ctx context.Context, symbol string, score float64, expiration time.Duration) error {
	return c.set(ctx, fmt.Sprintf(KeyHotScore, symbol), score, expiration)
}

// GetHotScore retrieves hot market score
func (c *Cache) GetHotScore(ctx context.Context, symbol string) (float64, error) {
	var score float64
	err := c.get(ctx, fmt.Sprintf(KeyHotScore, symbol), &score)
	return score, err
}

// Helper methods for JSON encoding/decoding
func (c *Cache) set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	return c.client.Set(ctx, key, data, expiration).Err()
}

func (c *Cache) get(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get value: %w", err)
	}
	return json.Unmarshal(data, dest)
}
