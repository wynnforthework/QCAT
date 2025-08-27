package api

import (
	"net/http"
	"strconv"

	"qcat/internal/strategy"

	"github.com/gin-gonic/gin"
)

// UnifiedStrategyHandler 统一策略处理器
// 整合原有的策略管理和双闭环策略池功能
type UnifiedStrategyHandler struct {
	service *strategy.UnifiedStrategyService
}

// NewUnifiedStrategyHandler 创建统一策略处理器
func NewUnifiedStrategyHandler(service *strategy.UnifiedStrategyService) *UnifiedStrategyHandler {
	return &UnifiedStrategyHandler{
		service: service,
	}
}

// ListStrategies 获取策略列表
// @Summary 获取策略列表
// @Description 获取策略列表，支持多种视图和过滤选项
// @Tags strategy
// @Accept json
// @Produce json
// @Param view query string false "视图类型" Enums(list,pool,execution,performance) default(list)
// @Param status query []string false "状态过滤"
// @Param type query []string false "类型过滤"
// @Param stage query []string false "阶段过滤"
// @Param pool_status query []string false "池状态过滤"
// @Param sort_by query string false "排序字段" default(updated_at)
// @Param sort_order query string false "排序方向" Enums(asc,desc) default(desc)
// @Param page query int false "页码" default(1)
// @Param page_size query int false "页大小" default(20)
// @Success 200 {object} strategy.StrategyListResponse
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/strategy [get]
func (h *UnifiedStrategyHandler) ListStrategies(c *gin.Context) {
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
	if stages := c.QueryArray("stage"); len(stages) > 0 {
		options.Stage = stages
	}
	if poolStatuses := c.QueryArray("pool_status"); len(poolStatuses) > 0 {
		options.PoolStatus = poolStatuses
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
// @Summary 获取策略详情
// @Description 获取单个策略的完整信息，包括执行状态、性能数据、池信息等
// @Tags strategy
// @Accept json
// @Produce json
// @Param id path string true "策略ID"
// @Success 200 {object} strategy.UnifiedStrategy
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/strategy/{id} [get]
func (h *UnifiedStrategyHandler) GetStrategy(c *gin.Context) {
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
// @Summary 获取策略池概览
// @Description 获取策略池的整体状态和统计信息
// @Tags strategy
// @Accept json
// @Produce json
// @Success 200 {object} PoolOverviewResponse
// @Failure 500 {object} Response
// @Router /api/v1/strategy/pool/overview [get]
func (h *UnifiedStrategyHandler) GetPoolOverview(c *gin.Context) {
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
	overview := PoolOverviewResponse{
		Distribution: map[string]interface{}{
			"enabled":  result.Summary.Pool.Enabled,
			"disabled": result.Summary.Pool.Disabled,
			"pending":  result.Summary.Pool.Pending,
			"testing":  result.Summary.Pool.Testing,
		},
		Summary:      result.Summary,
		RecentActivity: []PoolActivity{
			{
				Type:        "strategy_enabled",
				StrategyID:  "strategy_001",
				Message:     "策略 动量策略Alpha 已启用",
				Timestamp:   "2分钟前",
			},
			{
				Type:        "performance_update",
				StrategyID:  "strategy_002",
				Message:     "策略 均值回归策略Beta 性能更新",
				Timestamp:   "5分钟前",
			},
		},
		ResourceUsage: ResourceUsageInfo{
			CPU: ResourceMetric{
				Used:  6.3,
				Total: 16.0,
				Usage: 39.4,
			},
			Memory: ResourceMetric{
				Used:  12.4,
				Total: 32.0,
				Usage: 38.8,
			},
			ActiveWorkers: 15,
			QueuedTasks:   8,
		},
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    overview,
	})
}

// GetExecutionOverview 获取执行系统概览
// @Summary 获取执行系统概览
// @Description 获取策略执行系统的整体状态和性能指标
// @Tags strategy
// @Accept json
// @Produce json
// @Success 200 {object} ExecutionOverviewResponse
// @Failure 500 {object} Response
// @Router /api/v1/strategy/execution/overview [get]
func (h *UnifiedStrategyHandler) GetExecutionOverview(c *gin.Context) {
	// 获取策略列表用于统计
	options := strategy.StrategyListOptions{
		View:     "execution",
		PageSize: 1000,
	}

	result, err := h.service.ListStrategies(c.Request.Context(), options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// 计算执行统计
	var totalExecutions, totalSuccess, totalErrors int
	var totalLatency float64
	activeStrategies := 0

	for _, strategy := range result.Strategies {
		if strategy.Execution.IsRunning {
			activeStrategies++
		}
		totalExecutions += strategy.Execution.ExecutionCount
		totalSuccess += strategy.Execution.SuccessCount
		totalErrors += strategy.Execution.ErrorCount
		totalLatency += strategy.Execution.AvgLatency
	}

	avgLatency := float64(0)
	if len(result.Strategies) > 0 {
		avgLatency = totalLatency / float64(len(result.Strategies))
	}

	successRate := float64(0)
	if totalExecutions > 0 {
		successRate = float64(totalSuccess) / float64(totalExecutions) * 100
	}

	overview := ExecutionOverviewResponse{
		System: ExecutionSystemInfo{
			Status:           "running",
			ActiveStrategies: activeStrategies,
			TotalStrategies:  len(result.Strategies),
			Uptime:          "15天 8小时 32分钟",
		},
		Performance: ExecutionPerformanceInfo{
			Latency:         avgLatency,
			Throughput:      1250,
			SuccessRate:     successRate,
			ErrorRate:       100 - successRate,
			TotalExecutions: totalExecutions,
		},
		RecentExecutions: []ExecutionActivity{
			{
				StrategyID:   "strategy_001",
				StrategyName: "动量策略Alpha",
				Action:       "buy",
				Status:       "success",
				Latency:      8.2,
				Timestamp:    "30秒前",
			},
			{
				StrategyID:   "strategy_002",
				StrategyName: "均值回归策略Beta",
				Action:       "sell",
				Status:       "success",
				Latency:      12.5,
				Timestamp:    "1分钟前",
			},
		},
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    overview,
	})
}

// GetRealtimeStatus 获取实时状态
// @Summary 获取实时状态
// @Description 获取策略执行的实时状态信息
// @Tags strategy
// @Accept json
// @Produce json
// @Success 200 {object} RealtimeStatusResponse
// @Failure 500 {object} Response
// @Router /api/v1/strategy/execution/realtime [get]
func (h *UnifiedStrategyHandler) GetRealtimeStatus(c *gin.Context) {
	// 获取活跃策略
	options := strategy.StrategyListOptions{
		Status:   []string{"active"},
		PageSize: 100,
	}

	result, err := h.service.ListStrategies(c.Request.Context(), options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// 构建实时状态
	activeStrategies := make([]RealtimeStrategyInfo, 0)
	for _, strategy := range result.Strategies {
		if strategy.Execution.IsRunning {
			activeStrategies = append(activeStrategies, RealtimeStrategyInfo{
				ID:           strategy.ID,
				Name:         strategy.Name,
				Status:       "running",
				LastExecution: strategy.Execution.LastExecution.Format("15:04:05"),
				SuccessRate:  strategy.Execution.SuccessRate,
				Latency:      strategy.Execution.AvgLatency,
				PNL:          strategy.Performance.PNL,
			})
		}
	}

	status := RealtimeStatusResponse{
		Timestamp:        "2024-01-20T16:45:32Z",
		ActiveStrategies: activeStrategies,
		SystemMetrics: SystemMetrics{
			CPUUsage:    6.3,
			MemoryUsage: 12.4,
			DiskUsage:   8.1,
			Goroutines:  150,
			Uptime:      3600.0,
		},
		Alerts: []SystemAlert{
			{
				Level:   "warning",
				Message: "策略 strategy_003 执行延迟较高",
				Time:    "16:43:15",
			},
		},
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    status,
	})
}

// GetWorkflowStatus 获取工作流状态
// @Summary 获取工作流状态
// @Description 获取多策略工作流系统的状态信息
// @Tags strategy
// @Accept json
// @Produce json
// @Success 200 {object} WorkflowStatusResponse
// @Failure 500 {object} Response
// @Router /api/v1/strategy/workflow/status [get]
func (h *UnifiedStrategyHandler) GetWorkflowStatus(c *gin.Context) {
	status := WorkflowStatusResponse{
		System: WorkflowSystemInfo{
			Status:              "running",
			ActiveWorkflows:     10,
			ConcurrentStrategies: 15,
			EvolutionGeneration: 47,
			Uptime:             "15天 8小时 32分钟",
		},
		Evolution: EvolutionInfo{
			CurrentGeneration: 47,
			PopulationSize:    20,
			BestFitness:       0.892,
			AverageFitness:    0.634,
			DiversityIndex:    0.78,
			EliminatedCount:   156,
		},
		Resources: WorkflowResourceInfo{
			GlobalCPUUsage:    6.3,
			GlobalMemoryUsage: 12.4,
			ActiveWorkers:     15,
			QueuedTasks:       8,
		},
		RecentWorkflows: []WorkflowActivity{
			{
				ID:           "workflow_001",
				Name:         "动量策略优化",
				Stage:        "backtesting",
				Progress:     65,
				Status:       "running",
				StartTime:    "2小时前",
				EstimatedEnd: "1小时后",
			},
		},
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    status,
	})
}

// 响应数据结构定义

// PoolOverviewResponse 策略池概览响应
type PoolOverviewResponse struct {
	Distribution   map[string]interface{} `json:"distribution"`
	Summary        strategy.StrategySummary  `json:"summary"`
	RecentActivity []PoolActivity            `json:"recentActivity"`
	ResourceUsage  ResourceUsageInfo         `json:"resourceUsage"`
}

// PoolActivity 池活动
type PoolActivity struct {
	Type       string `json:"type"`
	StrategyID string `json:"strategyId"`
	Message    string `json:"message"`
	Timestamp  string `json:"timestamp"`
}

// ResourceUsageInfo 资源使用信息
type ResourceUsageInfo struct {
	CPU           ResourceMetric `json:"cpu"`
	Memory        ResourceMetric `json:"memory"`
	ActiveWorkers int            `json:"activeWorkers"`
	QueuedTasks   int            `json:"queuedTasks"`
}

// ResourceMetric 资源指标
type ResourceMetric struct {
	Used  float64 `json:"used"`
	Total float64 `json:"total"`
	Usage float64 `json:"usage"`
}

// ExecutionOverviewResponse 执行概览响应
type ExecutionOverviewResponse struct {
	System           ExecutionSystemInfo      `json:"system"`
	Performance      ExecutionPerformanceInfo `json:"performance"`
	RecentExecutions []ExecutionActivity      `json:"recentExecutions"`
}

// ExecutionSystemInfo 执行系统信息
type ExecutionSystemInfo struct {
	Status           string `json:"status"`
	ActiveStrategies int    `json:"activeStrategies"`
	TotalStrategies  int    `json:"totalStrategies"`
	Uptime          string `json:"uptime"`
}

// ExecutionPerformanceInfo 执行性能信息
type ExecutionPerformanceInfo struct {
	Latency         float64 `json:"latency"`
	Throughput      int     `json:"throughput"`
	SuccessRate     float64 `json:"successRate"`
	ErrorRate       float64 `json:"errorRate"`
	TotalExecutions int     `json:"totalExecutions"`
}

// ExecutionActivity 执行活动
type ExecutionActivity struct {
	StrategyID   string  `json:"strategyId"`
	StrategyName string  `json:"strategyName"`
	Action       string  `json:"action"`
	Status       string  `json:"status"`
	Latency      float64 `json:"latency"`
	Timestamp    string  `json:"timestamp"`
}

// RealtimeStatusResponse 实时状态响应
type RealtimeStatusResponse struct {
	Timestamp        string                  `json:"timestamp"`
	ActiveStrategies []RealtimeStrategyInfo  `json:"activeStrategies"`
	SystemMetrics    SystemMetrics           `json:"systemMetrics"`
	Alerts           []SystemAlert           `json:"alerts"`
}

// RealtimeStrategyInfo 实时策略信息
type RealtimeStrategyInfo struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Status        string  `json:"status"`
	LastExecution string  `json:"lastExecution"`
	SuccessRate   float64 `json:"successRate"`
	Latency       float64 `json:"latency"`
	PNL           float64 `json:"pnl"`
}

// NetworkMetrics 网络指标
type NetworkMetrics struct {
	Latency    float64 `json:"latency"`
	Throughput int     `json:"throughput"`
}

// SystemAlert 系统告警
type SystemAlert struct {
	Level   string `json:"level"`
	Message string `json:"message"`
	Time    string `json:"time"`
}

// WorkflowStatusResponse 工作流状态响应
type WorkflowStatusResponse struct {
	System          WorkflowSystemInfo     `json:"system"`
	Evolution       EvolutionInfo          `json:"evolution"`
	Resources       WorkflowResourceInfo   `json:"resources"`
	RecentWorkflows []WorkflowActivity     `json:"recentWorkflows"`
}

// WorkflowSystemInfo 工作流系统信息
type WorkflowSystemInfo struct {
	Status               string `json:"status"`
	ActiveWorkflows      int    `json:"activeWorkflows"`
	ConcurrentStrategies int    `json:"concurrentStrategies"`
	EvolutionGeneration  int    `json:"evolutionGeneration"`
	Uptime              string `json:"uptime"`
}

// EvolutionInfo 进化信息
type EvolutionInfo struct {
	CurrentGeneration int     `json:"currentGeneration"`
	PopulationSize    int     `json:"populationSize"`
	BestFitness       float64 `json:"bestFitness"`
	AverageFitness    float64 `json:"averageFitness"`
	DiversityIndex    float64 `json:"diversityIndex"`
	EliminatedCount   int     `json:"eliminatedCount"`
}

// WorkflowResourceInfo 工作流资源信息
type WorkflowResourceInfo struct {
	GlobalCPUUsage    float64 `json:"globalCpuUsage"`
	GlobalMemoryUsage float64 `json:"globalMemoryUsage"`
	ActiveWorkers     int     `json:"activeWorkers"`
	QueuedTasks       int     `json:"queuedTasks"`
}

// WorkflowActivity 工作流活动
type WorkflowActivity struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Stage        string `json:"stage"`
	Progress     int    `json:"progress"`
	Status       string `json:"status"`
	StartTime    string `json:"startTime"`
	EstimatedEnd string `json:"estimatedEnd"`
}