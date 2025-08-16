package cache

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// CacheManager manages cache operations with fallback mechanisms
type CacheManager struct {
	redis      Cacher
	memory     *MemoryCache
	database   DatabaseCache
	fallback   bool
	monitor    *CacheMonitor
	mu         sync.RWMutex
	config     *FallbackConfig
}

// FallbackConfig defines fallback configuration
type FallbackConfig struct {
	EnableFallback       bool          `json:"enable_fallback"`
	HealthCheckInterval  time.Duration `json:"health_check_interval"`
	FailureThreshold     int           `json:"failure_threshold"`
	RecoveryThreshold    int           `json:"recovery_threshold"`
	FallbackTimeout      time.Duration `json:"fallback_timeout"`
	SyncInterval         time.Duration `json:"sync_interval"`
	MaxMemoryCacheSize   int           `json:"max_memory_cache_size"`
	EnableDataSync       bool          `json:"enable_data_sync"`
	LogFallbackEvents    bool          `json:"log_fallback_events"`
}

// DatabaseCache defines interface for database cache operations
type DatabaseCache interface {
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

// NewCacheManager creates a new cache manager with fallback support
func NewCacheManager(redis Cacher, database DatabaseCache, config *FallbackConfig) *CacheManager {
	if config == nil {
		config = DefaultFallbackConfig()
	}

	memory := NewMemoryCache(config.MaxMemoryCacheSize)
	
	cm := &CacheManager{
		redis:    redis,
		memory:   memory,
		database: database,
		fallback: false,
		config:   config,
	}

	// Initialize monitor
	cm.monitor = NewCacheMonitor(cm, config)
	
	// Start health monitoring if enabled
	if config.EnableFallback {
		go cm.startHealthMonitoring()
	}

	// Start data sync if enabled
	if config.EnableDataSync {
		go cm.startDataSync()
	}

	return cm
}

// DefaultFallbackConfig returns default fallback configuration
func DefaultFallbackConfig() *FallbackConfig {
	return &FallbackConfig{
		EnableFallback:       true,
		HealthCheckInterval:  30 * time.Second,
		FailureThreshold:     3,
		RecoveryThreshold:    2,
		FallbackTimeout:      5 * time.Second,
		SyncInterval:         60 * time.Second,
		MaxMemoryCacheSize:   10000,
		EnableDataSync:       true,
		LogFallbackEvents:    true,
	}
}

// Get retrieves a value from cache with fallback
func (cm *CacheManager) Get(ctx context.Context, key string) (interface{}, error) {
	cm.mu.RLock()
	inFallback := cm.fallback
	cm.mu.RUnlock()

	// Try Redis first if not in fallback mode
	if !inFallback && cm.redis != nil {
		value, err := cm.redis.Get(ctx, key)
		if err == nil {
			// Cache hit in Redis, also store in memory for future fallback
			cm.memory.Set(ctx, key, value, time.Hour)
			return value, nil
		}
		
		// Redis error, check if we should enable fallback
		cm.monitor.RecordFailure("redis_get", err)
		if cm.shouldEnableFallback() {
			cm.enableFallback("redis_get_failure")
		}
	}

	// Try memory cache
	if value, err := cm.memory.Get(ctx, key); err == nil {
		cm.monitor.RecordHit("memory")
		return value, nil
	}

	// Try database as last resort
	if cm.database != nil {
		value, err := cm.database.Get(ctx, key)
		if err == nil {
			// Store in memory cache for future requests
			cm.memory.Set(ctx, key, value, time.Hour)
			cm.monitor.RecordHit("database")
			return value, nil
		}
		cm.monitor.RecordFailure("database_get", err)
	}

	cm.monitor.RecordMiss(key)
	return nil, fmt.Errorf("cache miss: key %s not found in any cache layer", key)
}

// Set stores a value in cache with fallback
func (cm *CacheManager) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	var redisErr error
	
	cm.mu.RLock()
	inFallback := cm.fallback
	cm.mu.RUnlock()

	// Try Redis first if not in fallback mode
	if !inFallback && cm.redis != nil {
		redisErr = cm.redis.Set(ctx, key, value, expiration)
		if redisErr == nil {
			// Also store in memory cache
			cm.memory.Set(ctx, key, value, expiration)
			cm.monitor.RecordSuccess("redis_set")
			return nil
		}
		
		// Redis error, check if we should enable fallback
		cm.monitor.RecordFailure("redis_set", redisErr)
		if cm.shouldEnableFallback() {
			cm.enableFallback("redis_set_failure")
		}
	}

	// Store in memory cache
	memErr := cm.memory.Set(ctx, key, value, expiration)
	if memErr == nil {
		cm.monitor.RecordSuccess("memory_set")
		
		// Try to store in database if available
		if cm.database != nil {
			if dbErr := cm.database.Set(ctx, key, value, expiration); dbErr != nil {
				cm.monitor.RecordFailure("database_set", dbErr)
			}
		}
		
		return nil
	}

	// All cache layers failed
	cm.monitor.RecordFailure("all_set", fmt.Errorf("redis: %v, memory: %v", redisErr, memErr))
	return fmt.Errorf("failed to set cache key %s: redis: %v, memory: %v", key, redisErr, memErr)
}

// Delete removes a value from cache with fallback
func (cm *CacheManager) Delete(ctx context.Context, key string) error {
	var errors []error

	cm.mu.RLock()
	inFallback := cm.fallback
	cm.mu.RUnlock()

	// Try Redis first if not in fallback mode
	if !inFallback && cm.redis != nil {
		if err := cm.redis.Delete(ctx, key); err != nil {
			errors = append(errors, fmt.Errorf("redis: %w", err))
			cm.monitor.RecordFailure("redis_delete", err)
		} else {
			cm.monitor.RecordSuccess("redis_delete")
		}
	}

	// Delete from memory cache
	if err := cm.memory.Delete(ctx, key); err != nil {
		errors = append(errors, fmt.Errorf("memory: %w", err))
		cm.monitor.RecordFailure("memory_delete", err)
	} else {
		cm.monitor.RecordSuccess("memory_delete")
	}

	// Delete from database if available
	if cm.database != nil {
		if err := cm.database.Delete(ctx, key); err != nil {
			errors = append(errors, fmt.Errorf("database: %w", err))
			cm.monitor.RecordFailure("database_delete", err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cache delete errors: %v", errors)
	}

	return nil
}

// Exists checks if a key exists in cache with fallback
func (cm *CacheManager) Exists(ctx context.Context, key string) (bool, error) {
	cm.mu.RLock()
	inFallback := cm.fallback
	cm.mu.RUnlock()

	// Try Redis first if not in fallback mode
	if !inFallback && cm.redis != nil {
		exists, err := cm.redis.Exists(ctx, key)
		if err == nil {
			return exists, nil
		}
		cm.monitor.RecordFailure("redis_exists", err)
	}

	// Try memory cache
	exists, err := cm.memory.Exists(ctx, key)
	if err == nil {
		return exists, nil
	}

	// Try database as last resort
	if cm.database != nil {
		return cm.database.Exists(ctx, key)
	}

	return false, fmt.Errorf("unable to check key existence: %s", key)
}

// enableFallback enables fallback mode
func (cm *CacheManager) enableFallback(reason string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.fallback {
		cm.fallback = true
		cm.monitor.RecordFallbackEvent("enabled", reason)
		
		if cm.config.LogFallbackEvents {
			log.Printf("Cache fallback enabled: %s", reason)
		}
	}
}

// disableFallback disables fallback mode
func (cm *CacheManager) disableFallback(reason string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.fallback {
		cm.fallback = false
		cm.monitor.RecordFallbackEvent("disabled", reason)
		
		if cm.config.LogFallbackEvents {
			log.Printf("Cache fallback disabled: %s", reason)
		}
	}
}

// shouldEnableFallback checks if fallback should be enabled
func (cm *CacheManager) shouldEnableFallback() bool {
	return cm.monitor.GetFailureCount() >= cm.config.FailureThreshold
}

// shouldDisableFallback checks if fallback should be disabled
func (cm *CacheManager) shouldDisableFallback() bool {
	return cm.monitor.GetSuccessCount() >= cm.config.RecoveryThreshold
}

// startHealthMonitoring starts health monitoring goroutine
func (cm *CacheManager) startHealthMonitoring() {
	ticker := time.NewTicker(cm.config.HealthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		cm.performHealthCheck()
	}
}

// performHealthCheck performs health check on Redis
func (cm *CacheManager) performHealthCheck() {
	if cm.redis == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), cm.config.FallbackTimeout)
	defer cancel()

	// Try a simple operation to check Redis health
	testKey := "health_check_" + fmt.Sprintf("%d", time.Now().UnixNano())
	err := cm.redis.Set(ctx, testKey, "ok", time.Second)
	
	cm.mu.RLock()
	inFallback := cm.fallback
	cm.mu.RUnlock()

	if err != nil {
		cm.monitor.RecordFailure("health_check", err)
		if !inFallback && cm.shouldEnableFallback() {
			cm.enableFallback("health_check_failure")
		}
	} else {
		cm.monitor.RecordSuccess("health_check")
		// Clean up test key
		cm.redis.Delete(ctx, testKey)
		
		if inFallback && cm.shouldDisableFallback() {
			cm.disableFallback("health_check_recovery")
		}
	}
}

// startDataSync starts data synchronization between cache layers
func (cm *CacheManager) startDataSync() {
	ticker := time.NewTicker(cm.config.SyncInterval)
	defer ticker.Stop()

	for range ticker.C {
		cm.performDataSync()
	}
}

// performDataSync synchronizes data between cache layers
func (cm *CacheManager) performDataSync() {
	// This is a placeholder for data synchronization logic
	// In a real implementation, you might want to:
	// 1. Sync hot keys from memory to Redis when Redis recovers
	// 2. Sync data from database to memory cache
	// 3. Handle data consistency between layers
	
	cm.mu.RLock()
	inFallback := cm.fallback
	cm.mu.RUnlock()

	if !inFallback && cm.redis != nil {
		// Redis is available, sync memory cache to Redis for backup
		cm.syncMemoryToRedis()
	}
}

// syncMemoryToRedis syncs memory cache to Redis
func (cm *CacheManager) syncMemoryToRedis() {
	// Get all keys from memory cache and sync to Redis
	// This is a simplified implementation
	keys := cm.memory.GetAllKeys()
	
	ctx, cancel := context.WithTimeout(context.Background(), cm.config.FallbackTimeout)
	defer cancel()

	for _, key := range keys {
		if value, err := cm.memory.Get(ctx, key); err == nil {
			// Try to set in Redis, but don't fail if it doesn't work
			cm.redis.Set(ctx, key, value, time.Hour)
		}
	}
}

// GetStats returns cache statistics
func (cm *CacheManager) GetStats() *CacheStats {
	cm.mu.RLock()
	inFallback := cm.fallback
	cm.mu.RUnlock()

	return &CacheStats{
		InFallback:    inFallback,
		RedisHealth:   cm.monitor.GetRedisHealth(),
		MemoryStats:   cm.memory.GetStats(),
		MonitorStats:  cm.monitor.GetStats(),
	}
}

// Close closes the cache manager and all its resources
func (cm *CacheManager) Close() error {
	var errors []error

	// Close Redis connection
	if cm.redis != nil {
		if err := cm.redis.Close(); err != nil {
			errors = append(errors, fmt.Errorf("redis close: %w", err))
		}
	}

	// Close memory cache
	if cm.memory != nil {
		if err := cm.memory.Close(); err != nil {
			errors = append(errors, fmt.Errorf("memory close: %w", err))
		}
	}

	// Stop monitor
	if cm.monitor != nil {
		cm.monitor.Stop()
	}

	if len(errors) > 0 {
		return fmt.Errorf("cache manager close errors: %v", errors)
	}

	return nil
}

// CacheStats represents cache statistics
type CacheStats struct {
	InFallback   bool                   `json:"in_fallback"`
	RedisHealth  bool                   `json:"redis_health"`
	MemoryStats  *MemoryCacheStats      `json:"memory_stats"`
	MonitorStats *CacheMonitorStats     `json:"monitor_stats"`
}