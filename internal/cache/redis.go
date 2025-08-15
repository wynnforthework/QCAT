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
func (r *RedisCache) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

// Set sets a value in cache with expiration
func (r *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

// Del deletes a key from cache
func (r *RedisCache) Del(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

// Exists checks if a key exists
func (r *RedisCache) Exists(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Exists(ctx, keys...).Result()
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

// ZAdd adds a member to a sorted set
func (r *RedisCache) ZAdd(ctx context.Context, key string, score float64, member string) error {
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

// Close closes the Redis connection
func (r *RedisCache) Close() error {
	return r.client.Close()
}
