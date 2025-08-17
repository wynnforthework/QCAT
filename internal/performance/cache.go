package performance

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"qcat/internal/logger"
)

// CacheOptimizer 缓存性能优化器
type CacheOptimizer struct {
	logger  logger.Logger
	metrics *CacheMetrics
	mu      sync.RWMutex
}

// CacheMetrics 缓存性能指标
type CacheMetrics struct {
	// 命中率指标
	Hits        int64   `json:"hits"`
	Misses      int64   `json:"misses"`
	HitRatio    float64 `json:"hit_ratio"`
	
	// 操作指标
	Gets        int64   `json:"gets"`
	Sets        int64   `json:"sets"`
	Deletes     int64   `json:"deletes"`
	Evictions   int64   `json:"evictions"`
	
	// 性能指标
	AvgGetTime  time.Duration `json:"avg_get_time"`
	AvgSetTime  time.Duration `json:"avg_set_time"`
	MaxGetTime  time.Duration `json:"max_get_time"`
	MaxSetTime  time.Duration `json:"max_set_time"`
	
	// 内存指标
	MemoryUsed  int64   `json:"memory_used"`
	MemoryLimit int64   `json:"memory_limit"`
	MemoryRatio float64 `json:"memory_ratio"`
	
	// 连接指标
	Connections     int `json:"connections"`
	MaxConnections  int `json:"max_connections"`
	IdleConnections int `json:"idle_connections"`
	
	// 错误指标
	ConnectionErrors int64 `json:"connection_errors"`
	TimeoutErrors   int64 `json:"timeout_errors"`
	OtherErrors     int64 `json:"other_errors"`
	
	// 时间戳
	Timestamp time.Time `json:"timestamp"`
}

// CacheOperation 缓存操作记录
type CacheOperation struct {
	Type      string        `json:"type"`      // GET, SET, DELETE
	Key       string        `json:"key"`
	Duration  time.Duration `json:"duration"`
	Success   bool          `json:"success"`
	Error     string        `json:"error,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

// HotKey 热点Key统计
type HotKey struct {
	Key         string    `json:"key"`
	AccessCount int64     `json:"access_count"`
	LastAccess  time.Time `json:"last_access"`
	AvgDuration time.Duration `json:"avg_duration"`
}

// NewCacheOptimizer 创建缓存优化器
func NewCacheOptimizer(logger logger.Logger) *CacheOptimizer {
	return &CacheOptimizer{
		logger:  logger,
		metrics: &CacheMetrics{},
	}
}

// RecordOperation 记录缓存操作
func (c *CacheOptimizer) RecordOperation(op CacheOperation) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// 更新操作计数
	switch op.Type {
	case "GET":
		atomic.AddInt64(&c.metrics.Gets, 1)
		if op.Success {
			atomic.AddInt64(&c.metrics.Hits, 1)
		} else {
			atomic.AddInt64(&c.metrics.Misses, 1)
		}
		
		// 更新GET性能指标
		if op.Duration > c.metrics.MaxGetTime {
			c.metrics.MaxGetTime = op.Duration
		}
		c.updateAvgGetTime(op.Duration)
		
	case "SET":
		atomic.AddInt64(&c.metrics.Sets, 1)
		
		// 更新SET性能指标
		if op.Duration > c.metrics.MaxSetTime {
			c.metrics.MaxSetTime = op.Duration
		}
		c.updateAvgSetTime(op.Duration)
		
	case "DELETE":
		atomic.AddInt64(&c.metrics.Deletes, 1)
	}
	
	// 更新错误计数
	if !op.Success {
		switch {
		case contains(op.Error, "connection"):
			atomic.AddInt64(&c.metrics.ConnectionErrors, 1)
		case contains(op.Error, "timeout"):
			atomic.AddInt64(&c.metrics.TimeoutErrors, 1)
		default:
			atomic.AddInt64(&c.metrics.OtherErrors, 1)
		}
	}
	
	// 更新命中率
	c.updateHitRatio()
	
	// 检查性能问题
	c.checkPerformanceIssues(op)
}

// updateAvgGetTime 更新平均GET时间
func (c *CacheOptimizer) updateAvgGetTime(duration time.Duration) {
	gets := atomic.LoadInt64(&c.metrics.Gets)
	if gets > 0 {
		// 简化的移动平均计算
		currentAvg := int64(c.metrics.AvgGetTime)
		newAvg := (currentAvg*(gets-1) + int64(duration)) / gets
		c.metrics.AvgGetTime = time.Duration(newAvg)
	}
}

// updateAvgSetTime 更新平均SET时间
func (c *CacheOptimizer) updateAvgSetTime(duration time.Duration) {
	sets := atomic.LoadInt64(&c.metrics.Sets)
	if sets > 0 {
		// 简化的移动平均计算
		currentAvg := int64(c.metrics.AvgSetTime)
		newAvg := (currentAvg*(sets-1) + int64(duration)) / sets
		c.metrics.AvgSetTime = time.Duration(newAvg)
	}
}

// updateHitRatio 更新命中率
func (c *CacheOptimizer) updateHitRatio() {
	hits := atomic.LoadInt64(&c.metrics.Hits)
	misses := atomic.LoadInt64(&c.metrics.Misses)
	total := hits + misses
	
	if total > 0 {
		c.metrics.HitRatio = float64(hits) / float64(total)
	}
}

// checkPerformanceIssues 检查性能问题
func (c *CacheOptimizer) checkPerformanceIssues(op CacheOperation) {
	// 检查操作延迟
	if op.Duration > 100*time.Millisecond {
		c.logger.Warn("Slow cache operation detected",
			"type", op.Type,
			"key", op.Key,
			"duration", op.Duration,
		)
	}
	
	// 检查命中率
	if c.metrics.HitRatio < 0.8 && c.metrics.Gets > 100 {
		c.logger.Warn("Low cache hit ratio detected",
			"hit_ratio", c.metrics.HitRatio,
			"hits", c.metrics.Hits,
			"misses", c.metrics.Misses,
		)
	}
	
	// 检查错误率
	totalOps := c.metrics.Gets + c.metrics.Sets + c.metrics.Deletes
	totalErrors := c.metrics.ConnectionErrors + c.metrics.TimeoutErrors + c.metrics.OtherErrors
	if totalOps > 0 {
		errorRate := float64(totalErrors) / float64(totalOps)
		if errorRate > 0.05 { // 错误率超过5%
			c.logger.Warn("High cache error rate detected",
				"error_rate", errorRate,
				"total_errors", totalErrors,
				"total_operations", totalOps,
			)
		}
	}
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && 
			(s[:len(substr)] == substr || 
			 s[len(s)-len(substr):] == substr ||
			 containsMiddle(s, substr))))
}

// containsMiddle 检查字符串中间是否包含子串
func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GetMetrics 获取缓存性能指标
func (c *CacheOptimizer) GetMetrics() *CacheMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// 创建副本
	metrics := *c.metrics
	metrics.Timestamp = time.Now()
	return &metrics
}

// UpdateMemoryStats 更新内存统计
func (c *CacheOptimizer) UpdateMemoryStats(used, limit int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.metrics.MemoryUsed = used
	c.metrics.MemoryLimit = limit
	
	if limit > 0 {
		c.metrics.MemoryRatio = float64(used) / float64(limit)
		
		// 检查内存使用率
		if c.metrics.MemoryRatio > 0.9 {
			c.logger.Warn("High cache memory usage detected",
				"memory_used_mb", used/(1024*1024),
				"memory_limit_mb", limit/(1024*1024),
				"memory_ratio", c.metrics.MemoryRatio,
			)
		}
	}
}

// UpdateConnectionStats 更新连接统计
func (c *CacheOptimizer) UpdateConnectionStats(current, max, idle int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.metrics.Connections = current
	c.metrics.MaxConnections = max
	c.metrics.IdleConnections = idle
	
	// 检查连接使用率
	if max > 0 {
		connectionRatio := float64(current) / float64(max)
		if connectionRatio > 0.8 {
			c.logger.Warn("High cache connection usage detected",
				"current_connections", current,
				"max_connections", max,
				"connection_ratio", connectionRatio,
			)
		}
	}
}

// HotKeyTracker 热点Key追踪器
type HotKeyTracker struct {
	hotKeys map[string]*HotKey
	mu      sync.RWMutex
	logger  logger.Logger
	maxKeys int
}

// NewHotKeyTracker 创建热点Key追踪器
func NewHotKeyTracker(maxKeys int, logger logger.Logger) *HotKeyTracker {
	return &HotKeyTracker{
		hotKeys: make(map[string]*HotKey),
		logger:  logger,
		maxKeys: maxKeys,
	}
}

// RecordAccess 记录Key访问
func (h *HotKeyTracker) RecordAccess(key string, duration time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	hotKey, exists := h.hotKeys[key]
	if !exists {
		// 检查是否超过最大Key数量
		if len(h.hotKeys) >= h.maxKeys {
			h.evictLeastUsedKey()
		}
		
		hotKey = &HotKey{
			Key:         key,
			AccessCount: 0,
			AvgDuration: duration,
		}
		h.hotKeys[key] = hotKey
	}
	
	// 更新统计
	hotKey.AccessCount++
	hotKey.LastAccess = time.Now()
	
	// 更新平均持续时间
	if hotKey.AccessCount > 1 {
		currentAvg := int64(hotKey.AvgDuration)
		newAvg := (currentAvg*(hotKey.AccessCount-1) + int64(duration)) / hotKey.AccessCount
		hotKey.AvgDuration = time.Duration(newAvg)
	} else {
		hotKey.AvgDuration = duration
	}
	
	// 检查是否为热点Key
	if hotKey.AccessCount > 100 && hotKey.AccessCount%100 == 0 {
		h.logger.Info("Hot key detected",
			"key", key,
			"access_count", hotKey.AccessCount,
			"avg_duration", hotKey.AvgDuration,
		)
	}
}

// evictLeastUsedKey 淘汰最少使用的Key
func (h *HotKeyTracker) evictLeastUsedKey() {
	var leastUsedKey string
	var leastAccessCount int64 = -1
	var oldestAccess time.Time
	
	for key, hotKey := range h.hotKeys {
		if leastAccessCount == -1 || 
		   hotKey.AccessCount < leastAccessCount ||
		   (hotKey.AccessCount == leastAccessCount && hotKey.LastAccess.Before(oldestAccess)) {
			leastUsedKey = key
			leastAccessCount = hotKey.AccessCount
			oldestAccess = hotKey.LastAccess
		}
	}
	
	if leastUsedKey != "" {
		delete(h.hotKeys, leastUsedKey)
	}
}

// GetHotKeys 获取热点Key列表
func (h *HotKeyTracker) GetHotKeys(limit int) []*HotKey {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	// 收集所有热点Key
	var hotKeys []*HotKey
	for _, hotKey := range h.hotKeys {
		hotKeyCopy := *hotKey
		hotKeys = append(hotKeys, &hotKeyCopy)
	}
	
	// 按访问次数排序
	for i := 0; i < len(hotKeys)-1; i++ {
		for j := i + 1; j < len(hotKeys); j++ {
			if hotKeys[i].AccessCount < hotKeys[j].AccessCount {
				hotKeys[i], hotKeys[j] = hotKeys[j], hotKeys[i]
			}
		}
	}
	
	// 返回前N个
	if limit > len(hotKeys) {
		limit = len(hotKeys)
	}
	
	return hotKeys[:limit]
}

// ClearHotKeys 清空热点Key统计
func (h *HotKeyTracker) ClearHotKeys() {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	h.hotKeys = make(map[string]*HotKey)
	h.logger.Info("Hot key statistics cleared")
}

// CacheOptimizationStrategy 缓存优化策略
type CacheOptimizationStrategy struct {
	optimizer   *CacheOptimizer
	hotTracker  *HotKeyTracker
	logger      logger.Logger
}

// NewCacheOptimizationStrategy 创建缓存优化策略
func NewCacheOptimizationStrategy(optimizer *CacheOptimizer, hotTracker *HotKeyTracker, logger logger.Logger) *CacheOptimizationStrategy {
	return &CacheOptimizationStrategy{
		optimizer:  optimizer,
		hotTracker: hotTracker,
		logger:     logger,
	}
}

// AnalyzePerformance 分析缓存性能
func (s *CacheOptimizationStrategy) AnalyzePerformance() map[string]interface{} {
	metrics := s.optimizer.GetMetrics()
	hotKeys := s.hotTracker.GetHotKeys(10)
	
	analysis := map[string]interface{}{
		"hit_ratio":           metrics.HitRatio,
		"avg_get_time_ms":     float64(metrics.AvgGetTime) / float64(time.Millisecond),
		"avg_set_time_ms":     float64(metrics.AvgSetTime) / float64(time.Millisecond),
		"memory_usage_ratio":  metrics.MemoryRatio,
		"connection_usage":    float64(metrics.Connections) / float64(metrics.MaxConnections),
		"error_rate":          s.calculateErrorRate(metrics),
		"hot_keys_count":      len(hotKeys),
		"total_operations":    metrics.Gets + metrics.Sets + metrics.Deletes,
	}
	
	return analysis
}

// calculateErrorRate 计算错误率
func (s *CacheOptimizationStrategy) calculateErrorRate(metrics *CacheMetrics) float64 {
	totalOps := metrics.Gets + metrics.Sets + metrics.Deletes
	totalErrors := metrics.ConnectionErrors + metrics.TimeoutErrors + metrics.OtherErrors
	
	if totalOps > 0 {
		return float64(totalErrors) / float64(totalOps)
	}
	return 0
}

// GenerateOptimizationSuggestions 生成优化建议
func (s *CacheOptimizationStrategy) GenerateOptimizationSuggestions() []string {
	metrics := s.optimizer.GetMetrics()
	var suggestions []string
	
	// 命中率优化建议
	if metrics.HitRatio < 0.8 {
		suggestions = append(suggestions, "缓存命中率较低，建议：")
		suggestions = append(suggestions, "1. 增加缓存容量")
		suggestions = append(suggestions, "2. 优化缓存键设计")
		suggestions = append(suggestions, "3. 调整缓存过期策略")
		suggestions = append(suggestions, "4. 预热热点数据")
	}
	
	// 性能优化建议
	avgGetTimeMs := float64(metrics.AvgGetTime) / float64(time.Millisecond)
	if avgGetTimeMs > 10 {
		suggestions = append(suggestions, "缓存GET操作较慢，建议：")
		suggestions = append(suggestions, "1. 检查网络延迟")
		suggestions = append(suggestions, "2. 优化序列化/反序列化")
		suggestions = append(suggestions, "3. 考虑使用本地缓存")
	}
	
	// 内存优化建议
	if metrics.MemoryRatio > 0.8 {
		suggestions = append(suggestions, "缓存内存使用率较高，建议：")
		suggestions = append(suggestions, "1. 增加缓存内存限制")
		suggestions = append(suggestions, "2. 优化数据结构")
		suggestions = append(suggestions, "3. 实施更积极的淘汰策略")
	}
	
	// 连接优化建议
	if metrics.MaxConnections > 0 {
		connectionRatio := float64(metrics.Connections) / float64(metrics.MaxConnections)
		if connectionRatio > 0.8 {
			suggestions = append(suggestions, "缓存连接使用率较高，建议：")
			suggestions = append(suggestions, "1. 增加最大连接数")
			suggestions = append(suggestions, "2. 优化连接池配置")
			suggestions = append(suggestions, "3. 实施连接复用")
		}
	}
	
	// 错误率优化建议
	errorRate := s.calculateErrorRate(metrics)
	if errorRate > 0.05 {
		suggestions = append(suggestions, "缓存错误率较高，建议：")
		suggestions = append(suggestions, "1. 检查网络连接稳定性")
		suggestions = append(suggestions, "2. 增加重试机制")
		suggestions = append(suggestions, "3. 实施熔断器模式")
	}
	
	// 热点Key优化建议
	hotKeys := s.hotTracker.GetHotKeys(5)
	if len(hotKeys) > 0 {
		suggestions = append(suggestions, fmt.Sprintf("发现%d个热点Key，建议：", len(hotKeys)))
		suggestions = append(suggestions, "1. 对热点Key实施本地缓存")
		suggestions = append(suggestions, "2. 考虑Key分片策略")
		suggestions = append(suggestions, "3. 增加热点Key的TTL")
	}
	
	if len(suggestions) == 0 {
		suggestions = append(suggestions, "缓存性能良好，无需特别优化")
	}
	
	return suggestions
}

// OptimizeCache 执行缓存优化
func (s *CacheOptimizationStrategy) OptimizeCache(ctx context.Context) error {
	s.logger.Info("Starting cache optimization")
	
	// 分析当前性能
	analysis := s.AnalyzePerformance()
	s.logger.Info("Cache performance analysis completed", "analysis", analysis)
	
	// 生成优化建议
	suggestions := s.GenerateOptimizationSuggestions()
	for i, suggestion := range suggestions {
		s.logger.Info("Cache optimization suggestion", "index", i+1, "suggestion", suggestion)
	}
	
	// 执行自动优化（如果适用）
	metrics := s.optimizer.GetMetrics()
	
	// 自动清理过期统计
	if metrics.Gets+metrics.Sets+metrics.Deletes > 1000000 {
		s.logger.Info("Clearing cache statistics due to high operation count")
		// 这里可以添加清理逻辑
	}
	
	s.logger.Info("Cache optimization completed")
	return nil
}