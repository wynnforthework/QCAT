package stability

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/cache"
	"qcat/internal/database"
)

// FallbackMode Redis降级模式
type FallbackMode string

const (
	FallbackModeRedis    FallbackMode = "redis"    // Redis模式
	FallbackModeMemory   FallbackMode = "memory"   // 内存模式
	FallbackModeDatabase FallbackMode = "database" // 数据库模式
)

// RedisFallback Redis降级管理器
type RedisFallback struct {
	mu           sync.RWMutex
	mode         FallbackMode
	redisCache   cache.Cache
	memoryCache  *MemoryCache
	db           *database.Database
	healthCheck  *RedisHealthCheck
	fallbackChan chan FallbackMode
}

// MemoryCache 内存缓存
type MemoryCache struct {
	mu    sync.RWMutex
	data  map[string]interface{}
	ttl   map[string]time.Time
	stats *CacheStats
}

// CacheStats 缓存统计
type CacheStats struct {
	Hits   int64
	Misses int64
	Sets   int64
	Dels   int64
}

// RedisHealthCheck Redis健康检查
type RedisHealthCheck struct {
	LastCheck    time.Time
	IsHealthy    bool
	Error        error
	ResponseTime time.Duration
}

// NewRedisFallback 创建Redis降级管理器
func NewRedisFallback(redisCache cache.Cache, db *database.Database) *RedisFallback {
	rf := &RedisFallback{
		redisCache:   redisCache,
		memoryCache:  NewMemoryCache(),
		db:           db,
		fallbackChan: make(chan FallbackMode, 10),
		healthCheck:  &RedisHealthCheck{},
	}

	// 启动健康检查
	go rf.startHealthCheck()

	// 启动模式切换监听
	go rf.modeSwitchListener()

	return rf
}

// NewMemoryCache 创建内存缓存
func NewMemoryCache() *MemoryCache {
	mc := &MemoryCache{
		data:  make(map[string]interface{}),
		ttl:   make(map[string]time.Time),
		stats: &CacheStats{},
	}

	// 启动TTL清理
	go mc.cleanupTTL()

	return mc
}

// Get 获取缓存值
func (rf *RedisFallback) Get(ctx context.Context, key string) (interface{}, error) {
	rf.mu.RLock()
	mode := rf.mode
	rf.mu.RUnlock()

	switch mode {
	case FallbackModeRedis:
		return rf.getFromRedis(ctx, key)
	case FallbackModeMemory:
		return rf.getFromMemory(key)
	case FallbackModeDatabase:
		return rf.getFromDatabase(ctx, key)
	default:
		return rf.getFromRedis(ctx, key)
	}
}

// Set 设置缓存值
func (rf *RedisFallback) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	rf.mu.RLock()
	mode := rf.mode
	rf.mu.RUnlock()

	switch mode {
	case FallbackModeRedis:
		return rf.setToRedis(ctx, key, value, ttl)
	case FallbackModeMemory:
		return rf.setToMemory(key, value, ttl)
	case FallbackModeDatabase:
		return rf.setToDatabase(ctx, key, value, ttl)
	default:
		return rf.setToRedis(ctx, key, value, ttl)
	}
}

// Delete 删除缓存值
func (rf *RedisFallback) Delete(ctx context.Context, key string) error {
	rf.mu.RLock()
	mode := rf.mode
	rf.mu.RUnlock()

	switch mode {
	case FallbackModeRedis:
		return rf.deleteFromRedis(ctx, key)
	case FallbackModeMemory:
		return rf.deleteFromMemory(key)
	case FallbackModeDatabase:
		return rf.deleteFromDatabase(ctx, key)
	default:
		return rf.deleteFromRedis(ctx, key)
	}
}

// getFromRedis 从Redis获取
func (rf *RedisFallback) getFromRedis(ctx context.Context, key string) (interface{}, error) {
	start := time.Now()
	value, err := rf.redisCache.Get(ctx, key)
	rf.healthCheck.ResponseTime = time.Since(start)

	if err != nil {
		log.Printf("Redis get error: %v, switching to memory mode", err)
		rf.switchToMemory()
		return rf.getFromMemory(key)
	}

	return value, nil
}

// setToRedis 设置到Redis
func (rf *RedisFallback) setToRedis(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	start := time.Now()
	err := rf.redisCache.Set(ctx, key, value, ttl)
	rf.healthCheck.ResponseTime = time.Since(start)

	if err != nil {
		log.Printf("Redis set error: %v, switching to memory mode", err)
		rf.switchToMemory()
		return rf.setToMemory(key, value, ttl)
	}

	return nil
}

// deleteFromRedis 从Redis删除
func (rf *RedisFallback) deleteFromRedis(ctx context.Context, key string) error {
	start := time.Now()
	err := rf.redisCache.Delete(ctx, key)
	rf.healthCheck.ResponseTime = time.Since(start)

	if err != nil {
		log.Printf("Redis delete error: %v, switching to memory mode", err)
		rf.switchToMemory()
		return rf.deleteFromMemory(key)
	}

	return nil
}

// getFromMemory 从内存获取
func (rf *RedisFallback) getFromMemory(key string) (interface{}, error) {
	rf.memoryCache.mu.RLock()
	defer rf.memoryCache.mu.RUnlock()

	// 检查TTL
	if ttl, exists := rf.memoryCache.ttl[key]; exists && time.Now().After(ttl) {
		delete(rf.memoryCache.data, key)
		delete(rf.memoryCache.ttl, key)
		rf.memoryCache.stats.Misses++
		return nil, fmt.Errorf("key expired")
	}

	value, exists := rf.memoryCache.data[key]
	if !exists {
		rf.memoryCache.stats.Misses++
		return nil, fmt.Errorf("key not found")
	}

	rf.memoryCache.stats.Hits++
	return value, nil
}

// setToMemory 设置到内存
func (rf *RedisFallback) setToMemory(key string, value interface{}, ttl time.Duration) error {
	rf.memoryCache.mu.Lock()
	defer rf.memoryCache.mu.Unlock()

	rf.memoryCache.data[key] = value
	if ttl > 0 {
		rf.memoryCache.ttl[key] = time.Now().Add(ttl)
	}

	rf.memoryCache.stats.Sets++
	return nil
}

// deleteFromMemory 从内存删除
func (rf *RedisFallback) deleteFromMemory(key string) error {
	rf.memoryCache.mu.Lock()
	defer rf.memoryCache.mu.Unlock()

	delete(rf.memoryCache.data, key)
	delete(rf.memoryCache.ttl, key)
	rf.memoryCache.stats.Dels++
	return nil
}

// getFromDatabase 从数据库获取
func (rf *RedisFallback) getFromDatabase(ctx context.Context, key string) (interface{}, error) {
	// 从market_data表获取缓存数据
	var data struct {
		Data      string    `db:"data"`
		Timestamp time.Time `db:"timestamp"`
	}

	query := `SELECT data, timestamp FROM market_data WHERE symbol = $1 AND data_type = 'cache' ORDER BY timestamp DESC LIMIT 1`
	err := rf.db.QueryRowContext(ctx, query, key).Scan(&data.Data, &data.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("database get error: %w", err)
	}

	// 检查数据是否过期（24小时）
	if time.Since(data.Timestamp) > 24*time.Hour {
		return nil, fmt.Errorf("data expired")
	}

	return data.Data, nil
}

// setToDatabase 设置到数据库
func (rf *RedisFallback) setToDatabase(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// 存储到market_data表
	query := `INSERT INTO market_data (symbol, data_type, data, timestamp) VALUES ($1, $2, $3, $4)`
	_, err := rf.db.ExecContext(ctx, query, key, "cache", value, time.Now())
	if err != nil {
		return fmt.Errorf("database set error: %w", err)
	}

	return nil
}

// deleteFromDatabase 从数据库删除
func (rf *RedisFallback) deleteFromDatabase(ctx context.Context, key string) error {
	// 从market_data表删除缓存数据
	query := `DELETE FROM market_data WHERE symbol = $1 AND data_type = 'cache'`
	_, err := rf.db.ExecContext(ctx, query, key)
	if err != nil {
		return fmt.Errorf("database delete error: %w", err)
	}

	return nil
}

// startHealthCheck 启动健康检查
func (rf *RedisFallback) startHealthCheck() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rf.checkRedisHealth()
		}
	}
}

// checkRedisHealth 检查Redis健康状态
func (rf *RedisFallback) checkRedisHealth() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	err := rf.redisCache.Set(ctx, "health_check", "ping", time.Minute)
	responseTime := time.Since(start)

	rf.mu.Lock()
	rf.healthCheck.LastCheck = time.Now()
	rf.healthCheck.ResponseTime = responseTime

	if err != nil {
		rf.healthCheck.IsHealthy = false
		rf.healthCheck.Error = err

		// 如果当前是Redis模式，切换到内存模式
		if rf.mode == FallbackModeRedis {
			log.Printf("Redis health check failed: %v, switching to memory mode", err)
			rf.switchToMemory()
		}
	} else {
		rf.healthCheck.IsHealthy = true
		rf.healthCheck.Error = nil

		// 如果当前是内存模式且Redis恢复，切换回Redis模式
		if rf.mode == FallbackModeMemory {
			log.Printf("Redis recovered, switching back to Redis mode")
			rf.switchToRedis()
		}
	}
	rf.mu.Unlock()
}

// switchToMemory 切换到内存模式
func (rf *RedisFallback) switchToMemory() {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if rf.mode != FallbackModeMemory {
		rf.mode = FallbackModeMemory
		select {
		case rf.fallbackChan <- FallbackModeMemory:
		default:
		}
		log.Printf("Switched to memory cache mode")
	}
}

// switchToRedis 切换到Redis模式
func (rf *RedisFallback) switchToRedis() {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if rf.mode != FallbackModeRedis {
		rf.mode = FallbackModeRedis
		select {
		case rf.fallbackChan <- FallbackModeRedis:
		default:
		}
		log.Printf("Switched to Redis cache mode")
	}
}

// modeSwitchListener 模式切换监听器
func (rf *RedisFallback) modeSwitchListener() {
	for mode := range rf.fallbackChan {
		log.Printf("Cache mode switched to: %s", mode)
		// 这里可以添加模式切换的通知逻辑
	}
}

// cleanupTTL 清理过期TTL
func (mc *MemoryCache) cleanupTTL() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mc.mu.Lock()
			now := time.Now()
			for key, ttl := range mc.ttl {
				if now.After(ttl) {
					delete(mc.data, key)
					delete(mc.ttl, key)
				}
			}
			mc.mu.Unlock()
		}
	}
}

// GetMode 获取当前模式
func (rf *RedisFallback) GetMode() FallbackMode {
	rf.mu.RLock()
	defer rf.mu.RUnlock()
	return rf.mode
}

// GetHealthCheck 获取健康检查状态
func (rf *RedisFallback) GetHealthCheck() *RedisHealthCheck {
	rf.mu.RLock()
	defer rf.mu.RUnlock()
	return rf.healthCheck
}

// GetMemoryStats 获取内存缓存统计
func (rf *RedisFallback) GetMemoryStats() *CacheStats {
	rf.memoryCache.mu.RLock()
	defer rf.memoryCache.mu.RUnlock()
	return rf.memoryCache.stats
}

// SyncToRedis 同步内存缓存到Redis
func (rf *RedisFallback) SyncToRedis(ctx context.Context) error {
	rf.memoryCache.mu.RLock()
	data := make(map[string]interface{})
	ttl := make(map[string]time.Time)
	for k, v := range rf.memoryCache.data {
		data[k] = v
	}
	for k, v := range rf.memoryCache.ttl {
		ttl[k] = v
	}
	rf.memoryCache.mu.RUnlock()

	for key, value := range data {
		var ttlDuration time.Duration
		if ttlTime, exists := ttl[key]; exists {
			ttlDuration = time.Until(ttlTime)
			if ttlDuration <= 0 {
				continue // 跳过已过期的数据
			}
		}

		if err := rf.redisCache.Set(ctx, key, value, ttlDuration); err != nil {
			log.Printf("Failed to sync key %s to Redis: %v", key, err)
		}
	}

	return nil
}
