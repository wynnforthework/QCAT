package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// MemoryCache implements an in-memory cache as a fallback when Redis is disabled
type MemoryCache struct {
	data  map[string]cacheItem
	mutex sync.RWMutex
}

type cacheItem struct {
	value      []byte
	expiration time.Time
}

// NewMemoryCache creates a new in-memory cache instance
func NewMemoryCache() *MemoryCache {
	cache := &MemoryCache{
		data: make(map[string]cacheItem),
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// cleanup periodically removes expired items
func (c *MemoryCache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		c.mutex.Lock()
		now := time.Now()
		for key, item := range c.data {
			if !item.expiration.IsZero() && now.After(item.expiration) {
				delete(c.data, key)
			}
		}
		c.mutex.Unlock()
	}
}

// SetMarketTick stores real-time market tick data
func (c *MemoryCache) SetMarketTick(ctx context.Context, symbol string, data interface{}, expiration time.Duration) error {
	return c.set(ctx, fmt.Sprintf(KeyTick, symbol), data, expiration)
}

// GetMarketTick retrieves real-time market tick data
func (c *MemoryCache) GetMarketTick(ctx context.Context, symbol string, dest interface{}) error {
	return c.get(ctx, fmt.Sprintf(KeyTick, symbol), dest)
}

// SetOrderBook stores order book data
func (c *MemoryCache) SetOrderBook(ctx context.Context, symbol string, data interface{}, expiration time.Duration) error {
	return c.set(ctx, fmt.Sprintf(KeyBook, symbol), data, expiration)
}

// GetOrderBook retrieves order book data
func (c *MemoryCache) GetOrderBook(ctx context.Context, symbol string, dest interface{}) error {
	return c.get(ctx, fmt.Sprintf(KeyBook, symbol), dest)
}

// SetFundingRate stores funding rate data
func (c *MemoryCache) SetFundingRate(ctx context.Context, symbol string, data interface{}, expiration time.Duration) error {
	return c.set(ctx, fmt.Sprintf(KeyFunding, symbol), data, expiration)
}

// GetFundingRate retrieves funding rate data
func (c *MemoryCache) GetFundingRate(ctx context.Context, symbol string, dest interface{}) error {
	return c.get(ctx, fmt.Sprintf(KeyFunding, symbol), dest)
}

// PushSignal adds a signal to the queue
func (c *MemoryCache) PushSignal(ctx context.Context, signal interface{}) error {
	data, err := json.Marshal(signal)
	if err != nil {
		return fmt.Errorf("failed to marshal signal: %w", err)
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	item, ok := c.data[KeySignalQueue]
	if !ok {
		item = cacheItem{value: []byte("[]")}
	}

	var signals [][]byte
	if err := json.Unmarshal(item.value, &signals); err != nil {
		return fmt.Errorf("failed to unmarshal signals: %w", err)
	}

	signals = append([][]byte{data}, signals...)
	newData, err := json.Marshal(signals)
	if err != nil {
		return fmt.Errorf("failed to marshal signals: %w", err)
	}

	c.data[KeySignalQueue] = cacheItem{value: newData}
	return nil
}

// PopSignal retrieves and removes a signal from the queue
func (c *MemoryCache) PopSignal(ctx context.Context, dest interface{}) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	item, ok := c.data[KeySignalQueue]
	if !ok {
		return nil
	}

	var signals [][]byte
	if err := json.Unmarshal(item.value, &signals); err != nil {
		return fmt.Errorf("failed to unmarshal signals: %w", err)
	}

	if len(signals) == 0 {
		return nil
	}

	signal := signals[len(signals)-1]
	signals = signals[:len(signals)-1]

	newData, err := json.Marshal(signals)
	if err != nil {
		return fmt.Errorf("failed to marshal signals: %w", err)
	}

	c.data[KeySignalQueue] = cacheItem{value: newData}
	return json.Unmarshal(signal, dest)
}

// SetPositionState stores position state
func (c *MemoryCache) SetPositionState(ctx context.Context, strategy, symbol string, data interface{}, expiration time.Duration) error {
	return c.set(ctx, fmt.Sprintf(KeyPosState, strategy, symbol), data, expiration)
}

// GetPositionState retrieves position state
func (c *MemoryCache) GetPositionState(ctx context.Context, strategy, symbol string, dest interface{}) error {
	return c.get(ctx, fmt.Sprintf(KeyPosState, strategy, symbol), dest)
}

// AcquireLock attempts to acquire a distributed lock
func (c *MemoryCache) AcquireLock(ctx context.Context, name string, expiration time.Duration) (bool, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	key := fmt.Sprintf(KeyLock, name)
	if item, exists := c.data[key]; exists {
		if item.expiration.IsZero() || time.Now().Before(item.expiration) {
			return false, nil
		}
	}

	c.data[key] = cacheItem{
		value:      []byte("1"),
		expiration: time.Now().Add(expiration),
	}
	return true, nil
}

// ReleaseLock releases a distributed lock
func (c *MemoryCache) ReleaseLock(ctx context.Context, name string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.data, fmt.Sprintf(KeyLock, name))
	return nil
}

// CheckRateLimit checks and updates rate limit
func (c *MemoryCache) CheckRateLimit(ctx context.Context, exchange string, limit int, window time.Duration) (bool, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	key := fmt.Sprintf(KeyRateLimit, exchange)
	now := time.Now()
	windowStart := now.Add(-window)

	var timestamps []time.Time
	if item, exists := c.data[key]; exists {
		if err := json.Unmarshal(item.value, &timestamps); err != nil {
			return false, fmt.Errorf("failed to unmarshal timestamps: %w", err)
		}
	}

	// Remove old timestamps
	validTimestamps := timestamps[:0]
	for _, ts := range timestamps {
		if ts.After(windowStart) {
			validTimestamps = append(validTimestamps, ts)
		}
	}

	if len(validTimestamps) >= limit {
		return false, nil
	}

	validTimestamps = append(validTimestamps, now)
	data, err := json.Marshal(validTimestamps)
	if err != nil {
		return false, fmt.Errorf("failed to marshal timestamps: %w", err)
	}

	c.data[key] = cacheItem{value: data}
	return true, nil
}

// SetHotScore stores hot market score
func (c *MemoryCache) SetHotScore(ctx context.Context, symbol string, score float64, expiration time.Duration) error {
	return c.set(ctx, fmt.Sprintf(KeyHotScore, symbol), score, expiration)
}

// GetHotScore retrieves hot market score
func (c *MemoryCache) GetHotScore(ctx context.Context, symbol string) (float64, error) {
	var score float64
	err := c.get(ctx, fmt.Sprintf(KeyHotScore, symbol), &score)
	return score, err
}

// Helper methods
func (c *MemoryCache) set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	var exp time.Time
	if expiration > 0 {
		exp = time.Now().Add(expiration)
	}

	c.data[key] = cacheItem{
		value:      data,
		expiration: exp,
	}
	return nil
}

func (c *MemoryCache) get(ctx context.Context, key string, dest interface{}) error {
	c.mutex.RLock()
	item, exists := c.data[key]
	c.mutex.RUnlock()

	if !exists {
		return nil
	}

	if !item.expiration.IsZero() && time.Now().After(item.expiration) {
		c.mutex.Lock()
		delete(c.data, key)
		c.mutex.Unlock()
		return nil
	}

	return json.Unmarshal(item.value, dest)
}
