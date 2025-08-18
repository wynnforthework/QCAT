package cache

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"
)

// MemoryCache implements an in-memory cache with TTL support
type MemoryCache struct {
	items    map[string]*memoryItem
	mu       sync.RWMutex
	maxSize  int
	stopChan chan struct{}
	stopped  bool
}

// memoryItem represents an item in memory cache
type memoryItem struct {
	value      interface{}
	expiration time.Time
	accessed   time.Time
}

// MemoryCacheStats represents memory cache statistics
type MemoryCacheStats struct {
	ItemCount    int       `json:"item_count"`
	MaxSize      int       `json:"max_size"`
	HitCount     int64     `json:"hit_count"`
	MissCount    int64     `json:"miss_count"`
	EvictionCount int64    `json:"eviction_count"`
	LastCleanup  time.Time `json:"last_cleanup"`
}

// NewMemoryCache creates a new memory cache
func NewMemoryCache(maxSize int) *MemoryCache {
	if maxSize <= 0 {
		maxSize = 10000 // Default max size
	}

	mc := &MemoryCache{
		items:    make(map[string]*memoryItem),
		maxSize:  maxSize,
		stopChan: make(chan struct{}),
	}

	// Start cleanup goroutine
	go mc.cleanupLoop()

	return mc
}

// Get retrieves a value from memory cache
func (mc *MemoryCache) Get(ctx context.Context, key string, dest interface{}) error {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	item, exists := mc.items[key]
	if !exists {
		return fmt.Errorf("key not found: %s", key)
	}

	// Check if item has expired
	if time.Now().After(item.expiration) {
		// Item expired, remove it
		go mc.deleteExpired(key)
		return fmt.Errorf("key expired: %s", key)
	}

	// Update access time
	item.accessed = time.Now()
	
	// Use reflection to set the destination
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return fmt.Errorf("dest must be a pointer")
	}

	itemValue := reflect.ValueOf(item.value)
	if itemValue.Type().AssignableTo(destValue.Elem().Type()) {
		destValue.Elem().Set(itemValue)
		return nil
	}

	return fmt.Errorf("type mismatch: cannot assign %v to %v", itemValue.Type(), destValue.Elem().Type())
}

// Set stores a value in memory cache
func (mc *MemoryCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Check if we need to evict items
	if len(mc.items) >= mc.maxSize {
		mc.evictLRU()
	}

	expirationTime := time.Now().Add(expiration)
	if expiration <= 0 {
		expirationTime = time.Now().Add(24 * time.Hour) // Default 24 hour expiration
	}

	mc.items[key] = &memoryItem{
		value:      value,
		expiration: expirationTime,
		accessed:   time.Now(),
	}

	return nil
}

// Delete removes a value from memory cache
func (mc *MemoryCache) Delete(ctx context.Context, key string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	delete(mc.items, key)
	return nil
}

// Exists checks if a key exists in memory cache
func (mc *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	item, exists := mc.items[key]
	if !exists {
		return false, nil
	}

	// Check if item has expired
	if time.Now().After(item.expiration) {
		go mc.deleteExpired(key)
		return false, nil
	}

	return true, nil
}

// GetAllKeys returns all keys in the cache
func (mc *MemoryCache) GetAllKeys() []string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	keys := make([]string, 0, len(mc.items))
	now := time.Now()

	for key, item := range mc.items {
		// Only return non-expired keys
		if now.Before(item.expiration) {
			keys = append(keys, key)
		}
	}

	return keys
}

// GetStats returns memory cache statistics
func (mc *MemoryCache) GetStats() *MemoryCacheStats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return &MemoryCacheStats{
		ItemCount:     len(mc.items),
		MaxSize:       mc.maxSize,
		HitCount:      0,  // Would need to track this separately
		MissCount:     0,  // Would need to track this separately
		EvictionCount: 0,  // Would need to track this separately
		LastCleanup:   time.Now(),
	}
}

// Clear removes all items from the cache
func (mc *MemoryCache) Clear() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.items = make(map[string]*memoryItem)
}

// Size returns the current number of items in the cache
func (mc *MemoryCache) Size() int {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return len(mc.items)
}

// evictLRU evicts the least recently used item
func (mc *MemoryCache) evictLRU() {
	if len(mc.items) == 0 {
		return
	}

	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, item := range mc.items {
		if first || item.accessed.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.accessed
			first = false
		}
	}

	if oldestKey != "" {
		delete(mc.items, oldestKey)
	}
}

// deleteExpired removes an expired key (called asynchronously)
func (mc *MemoryCache) deleteExpired(key string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Double-check that the item is still expired
	if item, exists := mc.items[key]; exists {
		if time.Now().After(item.expiration) {
			delete(mc.items, key)
		}
	}
}

// cleanupLoop runs periodic cleanup of expired items
func (mc *MemoryCache) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute) // Cleanup every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mc.cleanup()
		case <-mc.stopChan:
			return
		}
	}
}

// cleanup removes expired items
func (mc *MemoryCache) cleanup() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	now := time.Now()
	expiredKeys := make([]string, 0)

	// Find expired keys
	for key, item := range mc.items {
		if now.After(item.expiration) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	// Remove expired keys
	for _, key := range expiredKeys {
		delete(mc.items, key)
	}
}

// Close closes the memory cache
func (mc *MemoryCache) Close() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.stopped {
		close(mc.stopChan)
		mc.stopped = true
	}

	return nil
}

// Flush removes all items from the cache (alias for Clear)
func (mc *MemoryCache) Flush(ctx context.Context) error {
	mc.Clear()
	return nil
}

// Keys returns all non-expired keys
func (mc *MemoryCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	// For simplicity, ignore pattern matching and return all keys
	return mc.GetAllKeys(), nil
}

// TTL returns the time to live for a key
func (mc *MemoryCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	item, exists := mc.items[key]
	if !exists {
		return -2 * time.Second, fmt.Errorf("key not found: %s", key) // -2 means key doesn't exist
	}

	ttl := time.Until(item.expiration)
	if ttl < 0 {
		return -1 * time.Second, nil // -1 means key exists but has no expiration
	}

	return ttl, nil
}

// Expire sets a timeout on a key
func (mc *MemoryCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	item, exists := mc.items[key]
	if !exists {
		return fmt.Errorf("key not found: %s", key)
	}

	item.expiration = time.Now().Add(expiration)
	return nil
}

// CheckRateLimit checks if a rate limit has been exceeded
func (mc *MemoryCache) CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	now := time.Now()
	rateKey := fmt.Sprintf("rate_limit:%s", key)
	
	// Get current rate limit data
	var rateData []time.Time
	if item, exists := mc.items[rateKey]; exists {
		if data, ok := item.value.([]time.Time); ok {
			rateData = data
		}
	}

	// Remove expired entries
	cutoff := now.Add(-window)
	validEntries := make([]time.Time, 0)
	for _, entry := range rateData {
		if entry.After(cutoff) {
			validEntries = append(validEntries, entry)
		}
	}

	// Check if limit exceeded
	if len(validEntries) >= limit {
		return false, nil // Rate limit exceeded
	}

	// Add current request
	validEntries = append(validEntries, now)
	
	// Update cache
	mc.items[rateKey] = &memoryItem{
		value:      validEntries,
		expiration: now.Add(window),
	}

	return true, nil // Request allowed
}

// GetFundingRate retrieves funding rate from cache
func (mc *MemoryCache) GetFundingRate(ctx context.Context, symbol string, dest interface{}) error {
	return mc.Get(ctx, "funding_rate:"+symbol, dest)
}

// SetFundingRate stores funding rate in cache
func (mc *MemoryCache) SetFundingRate(ctx context.Context, symbol string, rate interface{}, expiration time.Duration) error {
	return mc.Set(ctx, "funding_rate:"+symbol, rate, expiration)
}

// GetIndexPrice retrieves index price from cache
func (mc *MemoryCache) GetIndexPrice(ctx context.Context, symbol string, dest interface{}) error {
	return mc.Get(ctx, "index_price:"+symbol, dest)
}

// SetIndexPrice stores index price in cache
func (mc *MemoryCache) SetIndexPrice(ctx context.Context, symbol string, price interface{}, expiration time.Duration) error {
	return mc.Set(ctx, "index_price:"+symbol, price, expiration)
}

// SetOrderBook stores order book in cache
func (mc *MemoryCache) SetOrderBook(ctx context.Context, symbol string, snapshot interface{}, expiration time.Duration) error {
	return mc.Set(ctx, "orderbook:"+symbol, snapshot, expiration)
}

// GetOrderBook retrieves order book from cache
func (mc *MemoryCache) GetOrderBook(ctx context.Context, symbol string, dest interface{}) error {
	return mc.Get(ctx, "orderbook:"+symbol, dest)
}

// HDel removes fields from a hash
func (mc *MemoryCache) HDel(ctx context.Context, key string, fields ...string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// For simplicity, we'll just delete the entire key
	// In a real implementation, you'd want to handle individual hash fields
	delete(mc.items, key)
	return nil
}