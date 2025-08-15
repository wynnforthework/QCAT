package cache

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache represents Redis cache implementation
type RedisCache struct {
	client *redis.Client
}

// Config represents Redis configuration
type Config struct {
	Addr     string
	Password string
	DB       int
	PoolSize int
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(cfg *Config) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Println("Redis connection established successfully")
	return &RedisCache{client: client}, nil
}

// Get retrieves a value from cache
func (r *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	_, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	// TODO: Implement proper deserialization
	return nil
}

// Set sets a value in cache with expiration
func (r *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

// Del deletes a key from cache
func (r *RedisCache) Del(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

// Incr increments a counter
func (r *RedisCache) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

// HGet retrieves a field from a hash
func (r *RedisCache) HGet(ctx context.Context, key, field string) (string, error) {
	return r.client.HGet(ctx, key, field).Result()
}

// HSet sets a field in a hash
func (r *RedisCache) HSet(ctx context.Context, key string, values ...interface{}) error {
	return r.client.HSet(ctx, key, values...).Err()
}

// HGetAll retrieves all fields from a hash
func (r *RedisCache) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return r.client.HGetAll(ctx, key).Result()
}

// ZAdd adds a member to a sorted set (legacy method)
func (r *RedisCache) ZAddLegacy(ctx context.Context, key string, score float64, member string) error {
	return r.client.ZAdd(ctx, key, redis.Z{Score: score, Member: member}).Err()
}

// ZRange retrieves members from a sorted set
func (r *RedisCache) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return r.client.ZRange(ctx, key, start, stop).Result()
}

// HealthCheck performs a health check on Redis
func (r *RedisCache) HealthCheck(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Delete deletes a key from cache
func (r *RedisCache) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// Exists checks if a key exists
func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := r.client.Exists(ctx, key).Result()
	return result > 0, err
}

// HDel deletes fields from a hash
func (r *RedisCache) HDel(ctx context.Context, key string, fields ...string) error {
	return r.client.HDel(ctx, key, fields...).Err()
}

// LPush pushes values to the left of a list
func (r *RedisCache) LPush(ctx context.Context, key string, values ...interface{}) error {
	return r.client.LPush(ctx, key, values...).Err()
}

// RPush pushes values to the right of a list
func (r *RedisCache) RPush(ctx context.Context, key string, values ...interface{}) error {
	return r.client.RPush(ctx, key, values...).Err()
}

// LPop pops a value from the left of a list
func (r *RedisCache) LPop(ctx context.Context, key string, dest interface{}) error {
	_, err := r.client.LPop(ctx, key).Result()
	// TODO: Implement proper deserialization
	return err
}

// RPop pops a value from the right of a list
func (r *RedisCache) RPop(ctx context.Context, key string, dest interface{}) error {
	_, err := r.client.RPop(ctx, key).Result()
	// TODO: Implement proper deserialization
	return err
}

// LRange retrieves a range of elements from a list
func (r *RedisCache) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return r.client.LRange(ctx, key, start, stop).Result()
}

// SAdd adds members to a set
func (r *RedisCache) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return r.client.SAdd(ctx, key, members...).Err()
}

// SRem removes members from a set
func (r *RedisCache) SRem(ctx context.Context, key string, members ...interface{}) error {
	return r.client.SRem(ctx, key, members...).Err()
}

// SMembers retrieves all members of a set
func (r *RedisCache) SMembers(ctx context.Context, key string) ([]string, error) {
	return r.client.SMembers(ctx, key).Result()
}

// SIsMember checks if a member exists in a set
func (r *RedisCache) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return r.client.SIsMember(ctx, key, member).Result()
}

// ZAdd adds a member to a sorted set
func (r *RedisCache) ZAdd(ctx context.Context, key string, score float64, member interface{}) error {
	return r.client.ZAdd(ctx, key, redis.Z{Score: score, Member: member}).Err()
}

// ZRangeByScore retrieves members from a sorted set by score
func (r *RedisCache) ZRangeByScore(ctx context.Context, key string, min, max string) ([]string, error) {
	return r.client.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: min,
		Max: max,
	}).Result()
}

// ZRem removes members from a sorted set
func (r *RedisCache) ZRem(ctx context.Context, key string, members ...interface{}) error {
	return r.client.ZRem(ctx, key, members...).Err()
}

// Expire sets expiration for a key
func (r *RedisCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return r.client.Expire(ctx, key, expiration).Err()
}

// TTL gets the time to live for a key
func (r *RedisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.client.TTL(ctx, key).Result()
}

// Flush flushes all keys
func (r *RedisCache) Flush(ctx context.Context) error {
	return r.client.FlushAll(ctx).Err()
}

// Close closes the Redis connection
func (r *RedisCache) Close() error {
	return r.client.Close()
}

// SetFundingRate sets funding rate in cache
func (r *RedisCache) SetFundingRate(ctx context.Context, symbol string, rate interface{}, expiration time.Duration) error {
	key := fmt.Sprintf("funding:%s", symbol)
	return r.Set(ctx, key, rate, expiration)
}

// GetFundingRate gets funding rate from cache
func (r *RedisCache) GetFundingRate(ctx context.Context, symbol string, dest interface{}) error {
	key := fmt.Sprintf("funding:%s", symbol)
	return r.Get(ctx, key, dest)
}

// SetIndexPrice sets index price in cache
func (r *RedisCache) SetIndexPrice(ctx context.Context, symbol string, price interface{}, expiration time.Duration) error {
	key := fmt.Sprintf("index:%s", symbol)
	return r.Set(ctx, key, price, expiration)
}

// GetIndexPrice gets index price from cache
func (r *RedisCache) GetIndexPrice(ctx context.Context, symbol string, dest interface{}) error {
	key := fmt.Sprintf("index:%s", symbol)
	return r.Get(ctx, key, dest)
}

// CheckRateLimit checks if a rate limit is exceeded
func (r *RedisCache) CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	// Use Redis sorted set for rate limiting
	now := time.Now().Unix()
	windowStart := now - int64(window.Seconds())
	
	// Remove expired entries
	err := r.client.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart)).Err()
	if err != nil {
		return false, err
	}
	
	// Count current entries
	count, err := r.client.ZCard(ctx, key).Result()
	if err != nil {
		return false, err
	}
	
	// Check if limit exceeded
	if int(count) >= limit {
		return false, nil
	}
	
	// Add current request
	err = r.client.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now}).Err()
	if err != nil {
		return false, err
	}
	
	// Set expiration
	err = r.client.Expire(ctx, key, window).Err()
	if err != nil {
		return false, err
	}
	
	return true, nil
}

// SetOrderBook sets order book snapshot in cache
func (r *RedisCache) SetOrderBook(ctx context.Context, symbol string, snapshot interface{}, expiration time.Duration) error {
	key := fmt.Sprintf("orderbook:%s", symbol)
	return r.Set(ctx, key, snapshot, expiration)
}

// GetOrderBook gets order book snapshot from cache
func (r *RedisCache) GetOrderBook(ctx context.Context, symbol string, dest interface{}) error {
	key := fmt.Sprintf("orderbook:%s", symbol)
	return r.Get(ctx, key, dest)
}
