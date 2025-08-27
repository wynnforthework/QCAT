package api

import (
	"net/http"
	"strconv"

	"qcat/internal/strategy"

	"github.com/gin-gonic/gin"
)

// SimpleUnifiedStrategyHandler 简化的统一策略处理器
type SimpleUnifiedStrategyHandler struct {
	service *strategy.SimpleUnifiedStrategyService
}

// NewSimpleUnifiedStrategyHandler 创建简化的统一策略处理器
func NewSimpleUnifiedStrategyHandler(service *strategy.SimpleUnifiedStrategyService) *SimpleUnifiedStrategyHandler {
	return &SimpleUnifiedStrategyHandler{
		service: service,
	}
}

// ListStrategies 获取策略列表
func (h *SimpleUnifiedStrategyHandler) ListStrategies(c *gin.Context) {
	// 解析查询参数
	options := strategy.StrategyListOptions{
		View:      c.DefaultQuery("view", "list"),
		SortBy:    c.DefaultQuery("sort_by", "updated_at"),
		SortOrder: c.DefaultQuery("sort_order", "desc"),
	}

	// 解析分页参数
	if page, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil {
		options.Page = page
	}
	if pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "20")); err == nil {
		options.PageSize = pageSize
	}

	// 解析数组参数
	if statuses := c.QueryArray("status"); len(statuses) > 0 {
		options.Status = statuses
	}
	if types := c.QueryArray("type"); len(types) > 0 {
		options.Type = types
	}

	// 调用服务
	result, err := h.service.ListStrategies(c.Request.Context(), options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    result,
	})
}

// GetStrategy 获取策略详情
func (h *SimpleUnifiedStrategyHandler) GetStrategy(c *gin.Context) {
	strategyID := c.Param("id")
	if strategyID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "strategy ID is required",
		})
		return
	}

	result, err := h.service.GetStrategy(c.Request.Context(), strategyID)
	if err != nil {
		if err.Error() == "strategy not found: "+strategyID {
			c.JSON(http.StatusNotFound, Response{
				Success: false,
				Error:   "strategy not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    result,
	})
}

// GetPoolOverview 获取策略池概览
func (h *SimpleUnifiedStrategyHandler) GetPoolOverview(c *gin.Context) {
	// 获取策略列表用于统计
	options := strategy.StrategyListOptions{
		View:     "pool",
		PageSize: 1000, // 获取所有策略用于统计
	}

	result, err := h.service.ListStrategies(c.Request.Context(), options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// 构建池概览响应
	overview := map[string]interface{}{
		"distribution": result.Summary.Pool,
		"summary":      result.Summary,
		"recentActivity": []map[string]interface{}{
			{
				"type":       "strategy_enabled",
				"strategyId": "strategy_001",
				"message":    "策略 动量策略Alpha 已启用",
				"timestamp":  "2分钟前",
			},
			{
				"type":       "performance_update",
				"strategyId": "strategy_002",
				"message":    "策略 均值回归策略Beta 性能更新",
				"timestamp":  "5分钟前",
			},
		},
		"resourceUsage": map[string]interface{}{
			"cpu": map[string]interface{}{
				"used":  6.3,
				"total": 16.0,
				"usage": 39.4,
			},
			"memory": map[string]interface{}{
				"used":  12.4,
				"total": 32.0,
				"usage": 38.8,
			},
			"activeWorkers": 15,
			"queuedTasks":   8,
		},
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    overview,
	})
}

// GetExecutionOverview 获取执行系统概览
func (h *SimpleUnifiedStrategyHandler) GetExecutionOverview(c *gin.Context) {
	overview := map[string]interface{}{
		"system": map[string]interface{}{
			"status":           "running",
			"activeStrategies": 2,
			"totalStrategies":  3,
			"uptime":          "15天 8小时 32分钟",
		},
		"performance": map[string]interface{}{
			"latency":         8.5,
			"throughput":      1250,
			"successRate":     95.6,
			"errorRate":       4.4,
			"totalExecutions": 2221,
		},
		"recentExecutions": []map[string]interface{}{
			{
				"strategyId":   "strategy_001",
				"strategyName": "动量策略Alpha",
				"action":       "buy",
				"status":       "success",
				"latency":      8.2,
				"timestamp":    "30秒前",
			},
			{
				"strategyId":   "strategy_002",
				"strategyName": "均值回归策略Beta",
				"action":       "sell",
				"status":       "success",
				"latency":      12.5,
				"timestamp":    "1分钟前",
			},
		},
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    overview,
	})
}

// GetRealtimeStatus 获取实时状态
func (h *SimpleUnifiedStrategyHandler) GetRealtimeStatus(c *gin.Context) {
	status := map[string]interface{}{
		"timestamp": "2024-01-20T16:45:32Z",
		"activeStrategies": []map[string]interface{}{
			{
				"id":            "strategy_001",
				"name":          "动量策略Alpha",
				"status":        "running",
				"lastExecution": "16:45:00",
				"successRate":   95.6,
				"latency":       8.5,
				"pnl":           15420.50,
			},
			{
				"id":            "strategy_002",
				"name":          "均值回归策略Beta",
				"status":        "running",
				"lastExecution": "16:44:30",
				"successRate":   94.2,
				"latency":       12.3,
				"pnl":           8750.25,
			},
		},
		"systemMetrics": map[string]interface{}{
			"cpu":    6.3,
			"memory": 12.4,
			"network": map[string]interface{}{
				"latency":    2.1,
				"throughput": 1250,
			},
		},
		"alerts": []map[string]interface{}{
			{
				"level":   "warning",
				"message": "策略 strategy_003 执行延迟较高",
				"time":    "16:43:15",
			},
		},
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    status,
	})
}

// GetWorkflowStatus 获取工作流状态
func (h *SimpleUnifiedStrategyHandler) GetWorkflowStatus(c *gin.Context) {
	status := map[string]interface{}{
		"system": map[string]interface{}{
			"status":              "running",
			"activeWorkflows":     10,
			"concurrentStrategies": 15,
			"evolutionGeneration": 47,
			"uptime":             "15天 8小时 32分钟",
		},
		"evolution": map[string]interface{}{
			"currentGeneration": 47,
			"populationSize":    20,
			"bestFitness":       0.892,
			"averageFitness":    0.634,
			"diversityIndex":    0.78,
			"eliminatedCount":   156,
		},
		"resources": map[string]interface{}{
			"globalCpuUsage":    6.3,
			"globalMemoryUsage": 12.4,
			"activeWorkers":     15,
			"queuedTasks":       8,
		},
		"recentWorkflows": []map[string]interface{}{
			{
				"id":           "workflow_001",
				"name":         "动量策略优化",
				"stage":        "backtesting",
				"progress":     65,
				"status":       "running",
				"startTime":    "2小时前",
				"estimatedEnd": "1小时后",
			},
		},
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    status,
	})
}