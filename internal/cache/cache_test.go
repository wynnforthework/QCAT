package cache

import (
	"context"
	"testing"
	"time"

	"qcat/internal/testutils"
)

func TestMemoryCache(t *testing.T) {
	suite := testutils.NewTestSuite(t, nil)
	defer suite.TearDown()

	cache := NewMemoryCache(100)
	ctx := context.Background()

	// 测试基本操作
	t.Run("basic operations", func(t *testing.T) {
		// Set
		err := cache.Set(ctx, "key1", "value1", time.Minute)
		if err != nil {
			t.Errorf("Set failed: %v", err)
		}

		// Get
		value, err := cache.Get(ctx, "key1")
		if err != nil {
			t.Errorf("Get failed: %v", err)
		}
		if value != "value1" {
			t.Errorf("Expected 'value1', got '%v'", value)
		}

		// Exists
		exists, err := cache.Exists(ctx, "key1")
		if err != nil {
			t.Errorf("Exists failed: %v", err)
		}
		if !exists {
			t.Error("Key should exist")
		}

		// Delete
		err = cache.Delete(ctx, "key1")
		if err != nil {
			t.Errorf("Delete failed: %v", err)
		}

		// Get after delete
		_, err = cache.Get(ctx, "key1")
		if err == nil {
			t.Error("Expected error for non-existent key")
		}
	})

	// 测试过期
	t.Run("expiration", func(t *testing.T) {
		err := cache.Set(ctx, "expire_key", "expire_value", 100*time.Millisecond)
		if err != nil {
			t.Errorf("Set failed: %v", err)
		}

		// 立即获取应该成功
		value, err := cache.Get(ctx, "expire_key")
		if err != nil {
			t.Errorf("Get failed: %v", err)
		}
		if value != "expire_value" {
			t.Errorf("Expected 'expire_value', got '%v'", value)
		}

		// 等待过期
		time.Sleep(150 * time.Millisecond)

		// 过期后获取应该失败
		_, err = cache.Get(ctx, "expire_key")
		if err == nil {
			t.Error("Expected error for expired key")
		}
	})

	// 测试容量限制
	t.Run("capacity limit", func(t *testing.T) {
		smallCache := NewMemoryCache(2)

		// 添加3个项目，应该只保留最新的2个
		smallCache.Set(ctx, "k1", "v1", time.Minute)
		smallCache.Set(ctx, "k2", "v2", time.Minute)
		smallCache.Set(ctx, "k3", "v3", time.Minute)

		// k1应该被淘汰
		_, err := smallCache.Get(ctx, "k1")
		if err == nil {
			t.Error("k1 should have been evicted")
		}

		// k2和k3应该存在
		_, err = smallCache.Get(ctx, "k2")
		if err != nil {
			t.Error("k2 should exist")
		}

		_, err = smallCache.Get(ctx, "k3")
		if err != nil {
			t.Error("k3 should exist")
		}
	})
}

func TestCacheAdapter(t *testing.T) {
	suite := testutils.NewTestSuite(t, nil)
	defer suite.TearDown()

	// 创建缓存管理器
	factory := NewCacheFactory(&CacheFactoryConfig{
		RedisEnabled:  false,
		MemoryEnabled: true,
		MemoryMaxSize: 100,
	})

	manager := factory.CreateMemoryOnlyCache()
	adapter := NewCacheAdapter(manager.(*CacheManager))

	ctx := context.Background()

	// 测试适配器功能
	err := adapter.Set(ctx, "test_key", "test_value", time.Minute)
	if err != nil {
		t.Errorf("Set failed: %v", err)
	}

	value, err := adapter.Get(ctx, "test_key")
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}

	if value != "test_value" {
		t.Errorf("Expected 'test_value', got '%v'", value)
	}
}

func TestCacheFallback(t *testing.T) {
	suite := testutils.NewTestSuite(t, nil)
	defer suite.TearDown()

	// 创建带降级的缓存管理器
	factory := NewCacheFactory(&CacheFactoryConfig{
		RedisEnabled:    true,
		RedisAddr:       "invalid:6379", // 故意使用无效地址
		MemoryEnabled:   true,
		MemoryMaxSize:   100,
		DatabaseEnabled: false,
		FallbackConfig:  DefaultFallbackConfig(),
	})

	manager, err := factory.CreateCache(nil)
	if err != nil {
		t.Fatalf("Failed to create cache manager: %v", err)
	}

	ctx := context.Background()

	// 测试降级功能
	err = manager.Set(ctx, "fallback_key", "fallback_value", time.Minute)
	if err != nil {
		t.Errorf("Set with fallback failed: %v", err)
	}

	value, err := manager.Get(ctx, "fallback_key")
	if err != nil {
		t.Errorf("Get with fallback failed: %v", err)
	}

	if value != "fallback_value" {
		t.Errorf("Expected 'fallback_value', got '%v'", value)
	}
}

func BenchmarkMemoryCache(b *testing.B) {
	config := &testutils.TestConfig{
		UseRealCache: false,
		LogLevel:     testutils.LogLevel("error"),
	}

	testutils.RunBenchmark(b, "MemoryCache_Set_Get", config, func(b *testing.B, suite *testutils.BenchmarkSuite) {
		cache := NewMemoryCache(10000)
		ctx := context.Background()
		mockData := testutils.NewMockData()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				key := mockData.RandomString(10)
				value := mockData.RandomString(100)

				// Set
				cache.Set(ctx, key, value, time.Minute)

				// Get
				cache.Get(ctx, key)
			}
		})
	})
}
