package cache

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestCacheManagerFallback(t *testing.T) {
	// Create a mock Redis cache that will fail
	mockRedis := &MockRedisCache{
		shouldFail: false,
		data:       make(map[string]interface{}),
	}

	// Create cache manager with fallback
	config := DefaultFallbackConfig()
	config.FailureThreshold = 2 // Lower threshold for testing
	config.HealthCheckInterval = 100 * time.Millisecond

	cm := NewCacheManager(mockRedis, nil, config)
	defer cm.Close()

	ctx := context.Background()

	// Test normal operation
	err := cm.Set(ctx, "test_key", "test_value", time.Minute)
	if err != nil {
		t.Fatalf("Expected successful set, got error: %v", err)
	}

	var value string
	err = cm.Get(ctx, "test_key", &value)
	if err != nil {
		t.Fatalf("Expected successful get, got error: %v", err)
	}

	if value != "test_value" {
		t.Errorf("Expected 'test_value', got %v", value)
	}

	// Simulate Redis failures
	mockRedis.shouldFail = true

	// Trigger failures to enable fallback
	for i := 0; i < 3; i++ {
		var dummy string
		cm.Get(ctx, "test_key", &dummy)
	}

	// Check if fallback is enabled
	stats := cm.GetStats()
	if !stats.InFallback {
		t.Error("Expected fallback to be enabled after failures")
	}

	// Test that cache still works in fallback mode
	err = cm.Set(ctx, "fallback_key", "fallback_value", time.Minute)
	if err != nil {
		t.Errorf("Expected successful set in fallback mode, got error: %v", err)
	}

	var fallbackValue string
	err = cm.Get(ctx, "fallback_key", &fallbackValue)
	if err != nil {
		t.Errorf("Expected successful get in fallback mode, got error: %v", err)
	}
	value = fallbackValue

	if value != "fallback_value" {
		t.Errorf("Expected 'fallback_value', got %v", value)
	}
}

func TestMemoryCacheExpiration(t *testing.T) {
	mc := NewMemoryCache(100)
	defer mc.Close()

	ctx := context.Background()

	// Set with short expiration
	err := mc.Set(ctx, "expire_key", "expire_value", 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Should exist immediately
	exists, err := mc.Exists(ctx, "expire_key")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}

	if !exists {
		t.Error("Expected key to exist immediately after set")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should not exist after expiration
	exists, err = mc.Exists(ctx, "expire_key")
	if err != nil {
		t.Fatalf("Exists check after expiration failed: %v", err)
	}

	if exists {
		t.Error("Expected key to not exist after expiration")
	}
}

func TestCacheMonitor(t *testing.T) {
	mockRedis := &MockRedisCache{
		shouldFail: false,
		data:       make(map[string]interface{}),
	}

	config := DefaultFallbackConfig()
	cm := NewCacheManager(mockRedis, nil, config)
	defer cm.Close()

	monitor := cm.monitor

	// Test recording hits and misses
	monitor.RecordHit("redis")
	monitor.RecordHit("memory")
	monitor.RecordMiss("nonexistent_key")

	stats := monitor.GetStats()
	if stats.HitCount != 2 {
		t.Errorf("Expected 2 hits, got %d", stats.HitCount)
	}

	if stats.MissCount != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.MissCount)
	}

	// Test recording failures
	monitor.RecordFailure("redis_get", nil)
	monitor.RecordFailure("redis_set", nil)

	stats = monitor.GetStats()
	if stats.ErrorCount != 2 {
		t.Errorf("Expected 2 errors, got %d", stats.ErrorCount)
	}

	// Test events
	events := monitor.GetRecentEvents(10)
	if len(events) == 0 {
		t.Error("Expected some events to be recorded")
	}
}

func TestCacheFactory(t *testing.T) {
	factory := NewCacheFactory(nil) // Use default config

	// Test memory-only cache creation
	memCache := factory.CreateMemoryOnlyCache()
	if memCache == nil {
		t.Error("Expected memory cache to be created")
	}

	ctx := context.Background()
	err := memCache.Set(ctx, "test", "value", time.Minute)
	if err != nil {
		t.Errorf("Memory cache set failed: %v", err)
	}

	var value string
	err = memCache.Get(ctx, "test", &value)
	if err != nil {
		t.Errorf("Memory cache get failed: %v", err)
	}

	if value != "value" {
		t.Errorf("Expected 'value', got %v", value)
	}
}

// MockRedisCache is a mock implementation for testing
type MockRedisCache struct {
	shouldFail bool
	data       map[string]interface{}
}

func (m *MockRedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	if m.shouldFail {
		return fmt.Errorf("mock Redis failure")
	}

	value, exists := m.data[key]
	if !exists {
		return fmt.Errorf("key not found: %s", key)
	}

	// 使用反射将值赋给dest
	if dest != nil {
		switch d := dest.(type) {
		case *string:
			if str, ok := value.(string); ok {
				*d = str
			}
		case *interface{}:
			*d = value
		}
	}
	return nil
}

func (m *MockRedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if m.shouldFail {
		return fmt.Errorf("mock Redis failure")
	}

	m.data[key] = value
	return nil
}

func (m *MockRedisCache) Delete(ctx context.Context, key string) error {
	if m.shouldFail {
		return fmt.Errorf("mock Redis failure")
	}

	delete(m.data, key)
	return nil
}

func (m *MockRedisCache) Exists(ctx context.Context, key string) (bool, error) {
	if m.shouldFail {
		return false, fmt.Errorf("mock Redis failure")
	}

	_, exists := m.data[key]
	return exists, nil
}

func (m *MockRedisCache) Close() error {
	return nil
}

// Implement other required methods for Cacher interface
func (m *MockRedisCache) HGet(ctx context.Context, key, field string, dest interface{}) error {
	return fmt.Errorf("not implemented")
}

func (m *MockRedisCache) HSet(ctx context.Context, key, field string, value interface{}) error {
	return fmt.Errorf("not implemented")
}

func (m *MockRedisCache) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockRedisCache) HDel(ctx context.Context, key string, fields ...string) error {
	return fmt.Errorf("not implemented")
}

func (m *MockRedisCache) LPush(ctx context.Context, key string, values ...interface{}) error {
	return fmt.Errorf("not implemented")
}

func (m *MockRedisCache) RPush(ctx context.Context, key string, values ...interface{}) error {
	return fmt.Errorf("not implemented")
}

func (m *MockRedisCache) LPop(ctx context.Context, key string, dest interface{}) error {
	return fmt.Errorf("not implemented")
}

func (m *MockRedisCache) RPop(ctx context.Context, key string, dest interface{}) error {
	return fmt.Errorf("not implemented")
}

func (m *MockRedisCache) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockRedisCache) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return fmt.Errorf("not implemented")
}

func (m *MockRedisCache) SRem(ctx context.Context, key string, members ...interface{}) error {
	return fmt.Errorf("not implemented")
}

func (m *MockRedisCache) SMembers(ctx context.Context, key string) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockRedisCache) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

func (m *MockRedisCache) ZAdd(ctx context.Context, key string, score float64, member interface{}) error {
	return fmt.Errorf("not implemented")
}

func (m *MockRedisCache) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockRedisCache) ZRangeByScore(ctx context.Context, key string, min, max string) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockRedisCache) ZRem(ctx context.Context, key string, members ...interface{}) error {
	return fmt.Errorf("not implemented")
}

func (m *MockRedisCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return fmt.Errorf("not implemented")
}

func (m *MockRedisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return 0, fmt.Errorf("not implemented")
}

func (m *MockRedisCache) Flush(ctx context.Context) error {
	m.data = make(map[string]interface{})
	return nil
}

func (m *MockRedisCache) SetFundingRate(ctx context.Context, symbol string, rate interface{}, expiration time.Duration) error {
	return fmt.Errorf("not implemented")
}

func (m *MockRedisCache) GetFundingRate(ctx context.Context, symbol string, dest interface{}) error {
	return fmt.Errorf("not implemented")
}

func (m *MockRedisCache) SetIndexPrice(ctx context.Context, symbol string, price interface{}, expiration time.Duration) error {
	return fmt.Errorf("not implemented")
}

func (m *MockRedisCache) GetIndexPrice(ctx context.Context, symbol string, dest interface{}) error {
	return fmt.Errorf("not implemented")
}

func (m *MockRedisCache) CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	return true, nil
}

func (m *MockRedisCache) SetOrderBook(ctx context.Context, symbol string, snapshot interface{}, expiration time.Duration) error {
	return fmt.Errorf("not implemented")
}

func (m *MockRedisCache) GetOrderBook(ctx context.Context, symbol string, dest interface{}) error {
	return fmt.Errorf("not implemented")
}
