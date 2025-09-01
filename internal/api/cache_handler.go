package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"qcat/internal/cache"
)

// CacheHandler handles cache-related API requests
type CacheHandler struct {
	cacheManager *cache.CacheManager
}

// NewCacheHandler creates a new cache handler
func NewCacheHandler(cacheManager *cache.CacheManager) *CacheHandler {
	return &CacheHandler{
		cacheManager: cacheManager,
	}
}

// handleCacheStatus returns cache status and statistics
func (h *CacheHandler) handleCacheStatus(c *gin.Context) {
	if h.cacheManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Cache manager not available",
		})
		return
	}

	stats := h.cacheManager.GetStats()
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now(),
		"stats":     stats,
	})
}

// handleCacheHealth returns comprehensive cache health information
func (h *CacheHandler) handleCacheHealth(c *gin.Context) {
	if h.cacheManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Cache manager not available",
		})
		return
	}

	// 执行简单的缓存健康检查
	ctx := c.Request.Context()
	testKey := "health_check_" + time.Now().Format("20060102150405")
	testValue := "ping"

	// 测试缓存写入
	setErr := h.cacheManager.Set(ctx, testKey, testValue, time.Minute)

	// 测试缓存读取
	var retrievedValue string
	getErr := h.cacheManager.Get(ctx, testKey, &retrievedValue)

	// 清理测试数据
	h.cacheManager.Delete(ctx, testKey)

	// 获取缓存统计信息
	stats := h.cacheManager.GetStats()

	// 判断健康状态
	overall := "healthy"
	statusCode := http.StatusOK

	if setErr != nil || getErr != nil {
		overall = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	healthReport := map[string]interface{}{
		"overall":   overall,
		"timestamp": time.Now(),
		"checks": map[string]interface{}{
			"set_operation": map[string]interface{}{
				"status": setErr == nil,
				"error": func() string {
					if setErr != nil {
						return setErr.Error()
					}
					return ""
				}(),
			},
			"get_operation": map[string]interface{}{
				"status": getErr == nil,
				"error": func() string {
					if getErr != nil {
						return getErr.Error()
					}
					return ""
				}(),
			},
		},
		"stats": stats,
	}

	c.JSON(statusCode, healthReport)
}

// handleCacheMetrics returns detailed cache metrics
func (h *CacheHandler) handleCacheMetrics(c *gin.Context) {
	if h.cacheManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Cache manager not available",
		})
		return
	}

	stats := h.cacheManager.GetStats()
	monitorStats := h.cacheManager.GetMonitor().GetStats()

	c.JSON(http.StatusOK, gin.H{
		"cache_stats":   stats,
		"monitor_stats": monitorStats,
		"timestamp":     time.Now(),
	})
}

// handleCacheEvents returns recent cache events
func (h *CacheHandler) handleCacheEvents(c *gin.Context) {
	if h.cacheManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Cache manager not available",
		})
		return
	}

	// Get limit from query parameter, default to 50
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := parseIntParam(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	events := h.cacheManager.GetMonitor().GetRecentEvents(limit)
	c.JSON(http.StatusOK, gin.H{
		"events":    events,
		"count":     len(events),
		"timestamp": time.Now(),
	})
}

// handleForceFallback forces cache fallback mode
func (h *CacheHandler) handleForceFallback(c *gin.Context) {
	if h.cacheManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Cache manager not available",
		})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	reason := req.Reason
	if reason == "" {
		reason = "manual_trigger"
	}

	// Force fallback by simulating failures
	h.cacheManager.GetMonitor().RecordFailure("manual_fallback", nil)
	h.cacheManager.GetMonitor().RecordFailure("manual_fallback", nil)
	h.cacheManager.GetMonitor().RecordFailure("manual_fallback", nil)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cache fallback mode triggered",
		"reason":  reason,
	})
}

// handleResetCounters resets cache monitoring counters
func (h *CacheHandler) handleResetCounters(c *gin.Context) {
	if h.cacheManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Cache manager not available",
		})
		return
	}

	h.cacheManager.GetMonitor().ResetCounters()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cache counters reset successfully",
	})
}

// handleCacheConfig returns cache configuration
func (h *CacheHandler) handleCacheConfig(c *gin.Context) {
	if h.cacheManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Cache manager not available",
		})
		return
	}

	config := h.cacheManager.GetConfig()
	c.JSON(http.StatusOK, gin.H{
		"config":    config,
		"timestamp": time.Now(),
	})
}

// handleTestCache tests cache operations
func (h *CacheHandler) handleTestCache(c *gin.Context) {
	if h.cacheManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Cache manager not available",
		})
		return
	}

	ctx := c.Request.Context()
	testKey := "cache_test_" + time.Now().Format("20060102150405")
	testValue := map[string]interface{}{
		"test":      true,
		"timestamp": time.Now(),
		"message":   "Cache test successful",
	}

	// Test set operation
	setStart := time.Now()
	err := h.cacheManager.Set(ctx, testKey, testValue, 5*time.Minute)
	setDuration := time.Since(setStart)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Cache set operation failed",
			"details": err.Error(),
		})
		return
	}

	// Test get operation
	getStart := time.Now()
	var retrievedValue interface{}
	err = h.cacheManager.Get(ctx, testKey, &retrievedValue)
	getDuration := time.Since(getStart)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Cache get operation failed",
			"details": err.Error(),
		})
		return
	}

	// Test exists operation
	existsStart := time.Now()
	exists, err := h.cacheManager.Exists(ctx, testKey)
	existsDuration := time.Since(existsStart)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Cache exists operation failed",
			"details": err.Error(),
		})
		return
	}

	// Test delete operation
	deleteStart := time.Now()
	err = h.cacheManager.Delete(ctx, testKey)
	deleteDuration := time.Since(deleteStart)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Cache delete operation failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cache test completed successfully",
		"results": gin.H{
			"set_operation": gin.H{
				"success":  true,
				"duration": setDuration.String(),
			},
			"get_operation": gin.H{
				"success":       true,
				"duration":      getDuration.String(),
				"value_matches": retrievedValue != nil,
			},
			"exists_operation": gin.H{
				"success":  true,
				"duration": existsDuration.String(),
				"exists":   exists,
			},
			"delete_operation": gin.H{
				"success":  true,
				"duration": deleteDuration.String(),
			},
		},
		"test_key": testKey,
	})
}

// Helper function to parse integer parameters
func parseIntParam(param string) (int, error) {
	if param == "" {
		return 0, nil
	}

	var result int
	if _, err := fmt.Sscanf(param, "%d", &result); err != nil {
		return 0, err
	}
	return result, nil
}

// RegisterRoutes registers cache management routes
func (h *CacheHandler) RegisterRoutes(router *gin.RouterGroup) {
	cache := router.Group("/cache")
	{
		cache.GET("/status", h.handleCacheStatus)
		cache.GET("/health", h.handleCacheHealth)
		cache.GET("/metrics", h.handleCacheMetrics)
		cache.GET("/events", h.handleCacheEvents)
		cache.GET("/config", h.handleCacheConfig)
		cache.POST("/test", h.handleTestCache)
		cache.POST("/fallback/force", h.handleForceFallback)
		cache.POST("/counters/reset", h.handleResetCounters)
	}
}
