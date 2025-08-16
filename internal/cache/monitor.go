package cache

import (
	"sync"
	"sync/atomic"
	"time"
)

// CacheMonitor monitors cache operations and health
type CacheMonitor struct {
	cacheManager   *CacheManager
	config         *FallbackConfig
	
	// Counters
	hitCount       int64
	missCount      int64
	errorCount     int64
	successCount   int64
	failureCount   int64
	
	// Health tracking
	redisHealth    bool
	lastHealthCheck time.Time
	
	// Event tracking
	events         []CacheEvent
	eventsMu       sync.RWMutex
	
	// Statistics
	stats          *CacheMonitorStats
	statsMu        sync.RWMutex
	
	// Control
	stopChan       chan struct{}
	stopped        bool
	mu             sync.RWMutex
}

// CacheEvent represents a cache event
type CacheEvent struct {
	Type      string    `json:"type"`
	Operation string    `json:"operation"`
	Key       string    `json:"key,omitempty"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Duration  time.Duration `json:"duration,omitempty"`
}

// CacheMonitorStats represents cache monitor statistics
type CacheMonitorStats struct {
	HitCount       int64     `json:"hit_count"`
	MissCount      int64     `json:"miss_count"`
	ErrorCount     int64     `json:"error_count"`
	SuccessCount   int64     `json:"success_count"`
	FailureCount   int64     `json:"failure_count"`
	HitRatio       float64   `json:"hit_ratio"`
	ErrorRatio     float64   `json:"error_ratio"`
	LastUpdated    time.Time `json:"last_updated"`
	RedisHealth    bool      `json:"redis_health"`
	FallbackEvents int       `json:"fallback_events"`
}

// NewCacheMonitor creates a new cache monitor
func NewCacheMonitor(cacheManager *CacheManager, config *FallbackConfig) *CacheMonitor {
	monitor := &CacheMonitor{
		cacheManager: cacheManager,
		config:       config,
		redisHealth:  true,
		events:       make([]CacheEvent, 0, 1000), // Keep last 1000 events
		stats:        &CacheMonitorStats{},
		stopChan:     make(chan struct{}),
	}
	
	// Start statistics update goroutine
	go monitor.updateStatsLoop()
	
	return monitor
}

// RecordHit records a cache hit
func (cm *CacheMonitor) RecordHit(layer string) {
	atomic.AddInt64(&cm.hitCount, 1)
	
	cm.recordEvent(CacheEvent{
		Type:      "hit",
		Operation: layer,
		Timestamp: time.Now(),
	})
}

// RecordMiss records a cache miss
func (cm *CacheMonitor) RecordMiss(key string) {
	atomic.AddInt64(&cm.missCount, 1)
	
	cm.recordEvent(CacheEvent{
		Type:      "miss",
		Operation: "get",
		Key:       key,
		Timestamp: time.Now(),
	})
}

// RecordSuccess records a successful operation
func (cm *CacheMonitor) RecordSuccess(operation string) {
	atomic.AddInt64(&cm.successCount, 1)
	
	// Reset failure count on success for health check operations
	if operation == "health_check" {
		atomic.StoreInt64(&cm.failureCount, 0)
		cm.redisHealth = true
	}
	
	cm.recordEvent(CacheEvent{
		Type:      "success",
		Operation: operation,
		Timestamp: time.Now(),
	})
}

// RecordFailure records a failed operation
func (cm *CacheMonitor) RecordFailure(operation string, err error) {
	atomic.AddInt64(&cm.errorCount, 1)
	atomic.AddInt64(&cm.failureCount, 1)
	
	// Update Redis health status
	if operation == "health_check" || operation == "redis_get" || operation == "redis_set" {
		cm.redisHealth = false
	}
	
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}
	
	cm.recordEvent(CacheEvent{
		Type:      "failure",
		Operation: operation,
		Error:     errorMsg,
		Timestamp: time.Now(),
	})
}

// RecordFallbackEvent records a fallback event
func (cm *CacheMonitor) RecordFallbackEvent(eventType, reason string) {
	cm.recordEvent(CacheEvent{
		Type:      "fallback_" + eventType,
		Operation: "fallback",
		Error:     reason,
		Timestamp: time.Now(),
	})
}

// recordEvent records an event in the event log
func (cm *CacheMonitor) recordEvent(event CacheEvent) {
	cm.eventsMu.Lock()
	defer cm.eventsMu.Unlock()
	
	// Add event to the beginning of the slice
	cm.events = append([]CacheEvent{event}, cm.events...)
	
	// Keep only the last 1000 events
	if len(cm.events) > 1000 {
		cm.events = cm.events[:1000]
	}
}

// GetFailureCount returns the current failure count
func (cm *CacheMonitor) GetFailureCount() int {
	return int(atomic.LoadInt64(&cm.failureCount))
}

// GetSuccessCount returns the current success count
func (cm *CacheMonitor) GetSuccessCount() int {
	return int(atomic.LoadInt64(&cm.successCount))
}

// GetRedisHealth returns the Redis health status
func (cm *CacheMonitor) GetRedisHealth() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.redisHealth
}

// GetStats returns current cache monitor statistics
func (cm *CacheMonitor) GetStats() *CacheMonitorStats {
	cm.statsMu.RLock()
	defer cm.statsMu.RUnlock()
	
	// Create a copy to avoid race conditions
	stats := *cm.stats
	return &stats
}

// GetRecentEvents returns recent cache events
func (cm *CacheMonitor) GetRecentEvents(limit int) []CacheEvent {
	cm.eventsMu.RLock()
	defer cm.eventsMu.RUnlock()
	
	if limit <= 0 || limit > len(cm.events) {
		limit = len(cm.events)
	}
	
	// Return a copy of the events
	events := make([]CacheEvent, limit)
	copy(events, cm.events[:limit])
	return events
}

// updateStatsLoop updates statistics periodically
func (cm *CacheMonitor) updateStatsLoop() {
	ticker := time.NewTicker(10 * time.Second) // Update stats every 10 seconds
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			cm.updateStats()
		case <-cm.stopChan:
			return
		}
	}
}

// updateStats updates the statistics
func (cm *CacheMonitor) updateStats() {
	cm.statsMu.Lock()
	defer cm.statsMu.Unlock()
	
	hitCount := atomic.LoadInt64(&cm.hitCount)
	missCount := atomic.LoadInt64(&cm.missCount)
	errorCount := atomic.LoadInt64(&cm.errorCount)
	successCount := atomic.LoadInt64(&cm.successCount)
	failureCount := atomic.LoadInt64(&cm.failureCount)
	
	totalRequests := hitCount + missCount
	totalOperations := successCount + errorCount
	
	// Calculate ratios
	var hitRatio, errorRatio float64
	if totalRequests > 0 {
		hitRatio = float64(hitCount) / float64(totalRequests)
	}
	if totalOperations > 0 {
		errorRatio = float64(errorCount) / float64(totalOperations)
	}
	
	// Count fallback events
	fallbackEvents := cm.countFallbackEvents()
	
	cm.stats = &CacheMonitorStats{
		HitCount:       hitCount,
		MissCount:      missCount,
		ErrorCount:     errorCount,
		SuccessCount:   successCount,
		FailureCount:   failureCount,
		HitRatio:       hitRatio,
		ErrorRatio:     errorRatio,
		LastUpdated:    time.Now(),
		RedisHealth:    cm.GetRedisHealth(),
		FallbackEvents: fallbackEvents,
	}
}

// countFallbackEvents counts fallback events in recent events
func (cm *CacheMonitor) countFallbackEvents() int {
	cm.eventsMu.RLock()
	defer cm.eventsMu.RUnlock()
	
	count := 0
	cutoff := time.Now().Add(-time.Hour) // Count events in the last hour
	
	for _, event := range cm.events {
		if event.Timestamp.Before(cutoff) {
			break // Events are ordered by time, so we can break here
		}
		
		if event.Type == "fallback_enabled" || event.Type == "fallback_disabled" {
			count++
		}
	}
	
	return count
}

// ResetCounters resets all counters
func (cm *CacheMonitor) ResetCounters() {
	atomic.StoreInt64(&cm.hitCount, 0)
	atomic.StoreInt64(&cm.missCount, 0)
	atomic.StoreInt64(&cm.errorCount, 0)
	atomic.StoreInt64(&cm.successCount, 0)
	atomic.StoreInt64(&cm.failureCount, 0)
	
	cm.eventsMu.Lock()
	cm.events = cm.events[:0] // Clear events
	cm.eventsMu.Unlock()
	
	cm.updateStats()
}

// Stop stops the cache monitor
func (cm *CacheMonitor) Stop() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	if !cm.stopped {
		close(cm.stopChan)
		cm.stopped = true
	}
}

// GetHealthSummary returns a health summary
func (cm *CacheMonitor) GetHealthSummary() *HealthSummary {
	stats := cm.GetStats()
	
	// Determine overall health
	overallHealth := "healthy"
	if !stats.RedisHealth {
		overallHealth = "degraded"
	}
	if stats.ErrorRatio > 0.1 { // More than 10% error rate
		overallHealth = "unhealthy"
	}
	
	// Get recent critical events
	recentEvents := cm.GetRecentEvents(10)
	criticalEvents := make([]CacheEvent, 0)
	for _, event := range recentEvents {
		if event.Type == "failure" || event.Type == "fallback_enabled" {
			criticalEvents = append(criticalEvents, event)
		}
	}
	
	return &HealthSummary{
		OverallHealth:   overallHealth,
		RedisHealth:     stats.RedisHealth,
		HitRatio:        stats.HitRatio,
		ErrorRatio:      stats.ErrorRatio,
		FallbackActive:  cm.cacheManager != nil && cm.cacheManager.fallback,
		CriticalEvents:  criticalEvents,
		LastUpdated:     time.Now(),
	}
}

// HealthSummary represents a cache health summary
type HealthSummary struct {
	OverallHealth  string       `json:"overall_health"`
	RedisHealth    bool         `json:"redis_health"`
	HitRatio       float64      `json:"hit_ratio"`
	ErrorRatio     float64      `json:"error_ratio"`
	FallbackActive bool         `json:"fallback_active"`
	CriticalEvents []CacheEvent `json:"critical_events"`
	LastUpdated    time.Time    `json:"last_updated"`
}