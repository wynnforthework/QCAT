package cache

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestMemoryCache(t *testing.T) {
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
		var value string
		err = cache.Get(ctx, "key1", &value)
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
		var deletedValue string
		err = cache.Get(ctx, "key1", &deletedValue)
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
		var value string
		err = cache.Get(ctx, "expire_key", &value)
		if err != nil {
			t.Errorf("Get failed: %v", err)
		}
		if value != "expire_value" {
			t.Errorf("Expected 'expire_value', got '%v'", value)
		}

		// 等待过期
		time.Sleep(150 * time.Millisecond)

		// 过期后获取应该失败
		var expiredValue string
		err = cache.Get(ctx, "expire_key", &expiredValue)
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
		var v1 string
		err := smallCache.Get(ctx, "k1", &v1)
		if err == nil {
			t.Error("k1 should have been evicted")
		}

		// k2和k3应该存在
		var v2 string
		err = smallCache.Get(ctx, "k2", &v2)
		if err != nil {
			t.Error("k2 should exist")
		}

		var v3 string
		err = smallCache.Get(ctx, "k3", &v3)
		if err != nil {
			t.Error("k3 should exist")
		}
	})
}

func TestCacheAdapter(t *testing.T) {
	// 创建缓存管理器 - 使用CreateCache而不是CreateMemoryOnlyCache
	factory := NewCacheFactory(&CacheFactoryConfig{
		RedisEnabled:  false,
		MemoryEnabled: true,
		MemoryMaxSize: 100,
	})

	manager, err := factory.CreateCache(nil)
	if err != nil {
		t.Fatalf("Failed to create cache manager: %v", err)
	}

	// 类型断言为*CacheManager
	cacheManager, ok := manager.(*CacheManager)
	if !ok {
		t.Fatalf("Expected *CacheManager, got %T", manager)
	}

	adapter := NewCacheAdapter(cacheManager)

	ctx := context.Background()

	// 测试适配器功能
	err = adapter.Set(ctx, "test_key", "test_value", time.Minute)
	if err != nil {
		t.Errorf("Set failed: %v", err)
	}

	var value string
	err = adapter.Get(ctx, "test_key", &value)
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}

	if value != "test_value" {
		t.Errorf("Expected 'test_value', got '%v'", value)
	}
}

func TestCacheFallback(t *testing.T) {
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

	var value string
	err = manager.Get(ctx, "fallback_key", &value)
	if err != nil {
		t.Errorf("Get with fallback failed: %v", err)
	}

	if value != "fallback_value" {
		t.Errorf("Expected 'fallback_value', got '%v'", value)
	}
}

func BenchmarkMemoryCache(b *testing.B) {
	cache := NewMemoryCache(10000)
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("test_key_%d", i)
			value := fmt.Sprintf("test_value_%d", i)

			// Set
			cache.Set(ctx, key, value, time.Minute)

			// Get
			var result string
			cache.Get(ctx, key, &result)
			i++
		}
	})
}
