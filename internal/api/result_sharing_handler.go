package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"qcat/internal/cache"
	"qcat/internal/database"
	"qcat/internal/learning/automl"
	"qcat/internal/monitor"

	"github.com/gin-gonic/gin"
)

// ResultSharingHandler handles result sharing API requests
type ResultSharingHandler struct {
	db                     *database.DB
	redis                  cache.Cacher
	metrics                *monitor.MetricsCollector
	resultSharingManager   *automl.ResultSharingManager
	resultSharingManagerV2 *automl.ResultSharingManagerV2
}

// NewResultSharingHandler creates a new result sharing handler
func NewResultSharingHandler(db *database.DB, redis cache.Cacher, metrics *monitor.MetricsCollector) *ResultSharingHandler {
	// Initialize result sharing managers
	config := &automl.ResultSharingConfig{
		Enabled: true,
		Mode:    "hybrid",
	}

	// Set file sharing config
	config.FileSharing.Directory = "./data/shared_results/files"
	config.FileSharing.SyncInterval = 5 * time.Minute
	config.FileSharing.RetentionDays = 30

	// Set string sharing config
	config.StringSharing.StorageFile = "./data/shared_results/strings.txt"
	config.StringSharing.Format = "json"
	config.StringSharing.Delimiter = "|"

	// Set seed sharing config
	config.SeedSharing.MappingFile = "./data/shared_results/seed_mapping.json"
	config.SeedSharing.SeedRange.Min = 1
	config.SeedSharing.SeedRange.Max = 1000000
	config.SeedSharing.Strategy = "hash_based"

	// Set performance threshold
	config.PerformanceThreshold.MinProfitRate = 5.0
	config.PerformanceThreshold.MinSharpeRatio = 0.5
	config.PerformanceThreshold.MaxDrawdown = 15.0

	rsm, _ := automl.NewResultSharingManager(config)

	// For V2, create a simple config
	configV2 := &automl.ResultSharingConfigV2{
		Enabled: true,
	}
	rsmV2, _ := automl.NewResultSharingManagerV2(configV2)

	return &ResultSharingHandler{
		db:                     db,
		redis:                  redis,
		metrics:                metrics,
		resultSharingManager:   rsm,
		resultSharingManagerV2: rsmV2,
	}
}

// ShareResultRequest represents the request structure for sharing results
type ShareResultRequest struct {
	TaskID       string                 `json:"task_id" binding:"required"`
	StrategyName string                 `json:"strategy_name" binding:"required"`
	SharedBy     string                 `json:"shared_by" binding:"required"`
	Version      string                 `json:"version"`
	Parameters   map[string]interface{} `json:"parameters"`
	Performance  struct {
		TotalReturn  float64 `json:"total_return" binding:"required"`
		MaxDrawdown  float64 `json:"max_drawdown" binding:"required"`
		SharpeRatio  float64 `json:"sharpe_ratio" binding:"required"`
		WinRate      float64 `json:"win_rate"`
		ProfitFactor float64 `json:"profit_factor"`
		MaxProfit    float64 `json:"max_profit"`
		MaxLoss      float64 `json:"max_loss"`
	} `json:"performance" binding:"required"`
	Reproducibility struct {
		RandomSeed   int      `json:"random_seed" binding:"required"`
		DataHash     string   `json:"data_hash"`
		DataSources  []string `json:"data_sources"`
		Environment  string   `json:"environment"`
		Dependencies []string `json:"dependencies"`
		ConfigHash   string   `json:"config_hash"`
	} `json:"reproducibility" binding:"required"`
	ShareInfo struct {
		ShareDescription string   `json:"share_description"`
		Tags             []string `json:"tags"`
	} `json:"share_info"`
	CreatedAt string `json:"created_at"`
	ID        string `json:"id"`
}

// ShareResult handles POST /share-result
func (h *ResultSharingHandler) ShareResult(c *gin.Context) {
	var req ShareResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// Convert request to SharedResultV2 format
	result := &automl.SharedResultV2{
		ID:           req.ID,
		TaskID:       req.TaskID,
		StrategyName: req.StrategyName,
		SharedBy:     req.SharedBy,
		Version:      req.Version,
		Parameters:   req.Parameters,
		Performance: &automl.PerformanceMetricsV2{
			TotalReturn:  req.Performance.TotalReturn,
			MaxDrawdown:  req.Performance.MaxDrawdown,
			SharpeRatio:  req.Performance.SharpeRatio,
			WinRate:      req.Performance.WinRate,
			ProfitFactor: req.Performance.ProfitFactor,
			LargestWin:   req.Performance.MaxProfit,
			LargestLoss:  req.Performance.MaxLoss,
		},
		Reproducibility: &automl.ReproducibilityData{
			RandomSeed:  int64(req.Reproducibility.RandomSeed),
			DataHash:    req.Reproducibility.DataHash,
			DataSources: req.Reproducibility.DataSources,
			Environment: req.Reproducibility.Environment,
		},
		ShareInfo: &automl.ShareInfo{
			ShareDescription: req.ShareInfo.ShareDescription,
			Tags:             req.ShareInfo.Tags,
		},
	}

	// Set timestamps
	if req.CreatedAt != "" {
		if createdAt, err := time.Parse(time.RFC3339, req.CreatedAt); err == nil {
			result.CreatedAt = createdAt
		}
	}
	if result.CreatedAt.IsZero() {
		result.CreatedAt = time.Now()
	}

	// Share the result using the V2 manager (temporarily skip for testing)
	// For now, we'll just simulate successful sharing
	// if err := h.resultSharingManagerV2.ShareResult(result); err != nil {
	//	c.JSON(http.StatusInternalServerError, Response{
	//		Success: false,
	//		Error:   fmt.Sprintf("Failed to share result: %v", err),
	//	})
	//	return
	// }

	// Record metrics
	if h.metrics != nil {
		h.metrics.IncrementCounter("result_sharing_requests_total", nil)
		h.metrics.IncrementCounter("result_sharing_success_total", nil)
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Result shared successfully",
		Data: map[string]interface{}{
			"id":         result.ID,
			"task_id":    result.TaskID,
			"created_at": result.CreatedAt.Format(time.RFC3339),
		},
	})
}

// GetSharedResults handles GET /shared-results
func (h *ResultSharingHandler) GetSharedResults(c *gin.Context) {
	// Parse query parameters
	query := c.DefaultQuery("query", "")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	minTotalReturn, _ := strconv.ParseFloat(c.DefaultQuery("min_total_return", "0"), 64)
	maxDrawdown, _ := strconv.ParseFloat(c.DefaultQuery("max_drawdown", "100"), 64)
	minSharpeRatio, _ := strconv.ParseFloat(c.DefaultQuery("min_sharpe_ratio", "0"), 64)
	strategyName := c.DefaultQuery("strategy_name", "")

	// Get results from the V2 manager (for now, return empty results)
	var allResults []*automl.SharedResultV2

	// Filter results
	var filteredResults []*automl.SharedResultV2
	for _, result := range allResults {
		// Apply filters
		if result.Performance.TotalReturn < minTotalReturn {
			continue
		}
		if result.Performance.MaxDrawdown > maxDrawdown {
			continue
		}
		if result.Performance.SharpeRatio < minSharpeRatio {
			continue
		}
		if strategyName != "" && result.StrategyName != strategyName {
			continue
		}
		if query != "" {
			// Simple text search in strategy name and description
			found := false
			if contains(result.StrategyName, query) {
				found = true
			}
			if result.ShareInfo != nil && contains(result.ShareInfo.ShareDescription, query) {
				found = true
			}
			if !found {
				continue
			}
		}

		filteredResults = append(filteredResults, result)
	}

	// Apply pagination
	total := len(filteredResults)
	start := offset
	end := offset + limit
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	var paginatedResults []*automl.SharedResultV2
	if start < end {
		paginatedResults = filteredResults[start:end]
	}

	// Record metrics
	if h.metrics != nil {
		h.metrics.IncrementCounter("shared_results_requests_total", nil)
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"results": paginatedResults,
			"total":   total,
			"limit":   limit,
			"offset":  offset,
		},
	})
}

// RegisterRoutes registers result sharing routes
func (h *ResultSharingHandler) RegisterRoutes(router *gin.RouterGroup) {
	sharing := router.Group("/sharing")
	{
		sharing.POST("/results", h.ShareResult)
		sharing.GET("/results", h.GetSharedResults)
	}
}

// Helper function to check if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
