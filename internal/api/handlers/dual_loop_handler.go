package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"qcat/internal/strategy/workflow"
)

// DualLoopHandler 双闭环API处理器
type DualLoopHandler struct {
	system *workflow.MultiStrategyWorkflowSystem
}

// NewDualLoopHandler 创建双闭环处理器
func NewDualLoopHandler(system *workflow.MultiStrategyWorkflowSystem) *DualLoopHandler {
	return &DualLoopHandler{
		system: system,
	}
}

// DualLoopOverviewResponse 双闭环概览响应
type DualLoopOverviewResponse struct {
	TradingSystem struct {
		Status           string  `json:"status"`
		ActiveStrategies int     `json:"activeStrategies"`
		ExecutionLatency float64 `json:"executionLatency"`
		Throughput       int     `json:"throughput"`
		Uptime           string  `json:"uptime"`
		SuccessRate      float64 `json:"successRate"`
	} `json:"tradingSystem"`
	StrategySystem struct {
		Status               string  `json:"status"`
		ActiveWorkflows      int     `json:"activeWorkflows"`
		ConcurrentStrategies int     `json:"concurrentStrategies"`
		EvolutionGeneration  int     `json:"evolutionGeneration"`
		Uptime               string  `json:"uptime"`
		CompletionRate       float64 `json:"completionRate"`
	} `json:"strategySystem"`
	StrategyPool struct {
		TotalStrategies    int    `json:"totalStrategies"`
		EnabledStrategies  int    `json:"enabledStrategies"`
		DisabledStrategies int    `json:"disabledStrategies"`
		PendingStrategies  int    `json:"pendingStrategies"`
		LastSync           string `json:"lastSync"`
		SyncStatus         string `json:"syncStatus"`
	} `json:"strategyPool"`
}

// GetDualLoopOverview 获取双闭环系统概览
func (h *DualLoopHandler) GetDualLoopOverview(w http.ResponseWriter, r *http.Request) {
	if !h.system.IsRunning() {
		http.Error(w, "System is not running", http.StatusServiceUnavailable)
		return
	}

	// 获取系统统计信息
	stats := h.system.GetSystemStats()

	// 构建响应数据
	response := DualLoopOverviewResponse{}

	// 交易系统状态（模拟数据，实际应该从交易系统获取）
	response.TradingSystem.Status = "running"
	response.TradingSystem.ActiveStrategies = int(stats.EnabledStrategies)
	response.TradingSystem.ExecutionLatency = 8.5
	response.TradingSystem.Throughput = 1250
	response.TradingSystem.Uptime = formatDuration(stats.Uptime)
	response.TradingSystem.SuccessRate = 98.7

	// 策略系统状态
	response.StrategySystem.Status = "running"
	response.StrategySystem.ActiveWorkflows = int(stats.ActiveStrategies)
	response.StrategySystem.ConcurrentStrategies = 10
	response.StrategySystem.EvolutionGeneration = 47
	response.StrategySystem.Uptime = formatDuration(stats.Uptime)
	if stats.TotalExecutions > 0 {
		response.StrategySystem.CompletionRate = float64(stats.SuccessfulExecutions) / float64(stats.TotalExecutions) * 100
	}

	// 策略池状态
	response.StrategyPool.TotalStrategies = int(stats.TotalStrategies)
	response.StrategyPool.EnabledStrategies = int(stats.EnabledStrategies)
	response.StrategyPool.DisabledStrategies = int(stats.DisabledStrategies)
	response.StrategyPool.PendingStrategies = int(stats.TotalStrategies - stats.EnabledStrategies - stats.DisabledStrategies)
	response.StrategyPool.LastSync = "2分钟前"
	response.StrategyPool.SyncStatus = "success"

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// StrategyWorkflowResponse 策略工作流响应
type StrategyWorkflowResponse struct {
	Workflows []struct {
		ID                  string `json:"id"`
		Name                string `json:"name"`
		CurrentStage        string `json:"currentStage"`
		Progress            int    `json:"progress"`
		StartTime           string `json:"startTime"`
		EstimatedCompletion string `json:"estimatedCompletion"`
		ResourceUsage       struct {
			CPU    float64 `json:"cpu"`
			Memory float64 `json:"memory"`
		} `json:"resourceUsage"`
		Status string `json:"status"`
	} `json:"workflows"`
	Evolution struct {
		CurrentGeneration int     `json:"currentGeneration"`
		PopulationSize    int     `json:"populationSize"`
		BestFitness       float64 `json:"bestFitness"`
		AverageFitness    float64 `json:"averageFitness"`
		DiversityIndex    float64 `json:"diversityIndex"`
		EliminatedCount   int     `json:"eliminatedCount"`
	} `json:"evolution"`
	Resources struct {
		GlobalCPUUsage    float64 `json:"globalCPUUsage"`
		GlobalMemoryUsage float64 `json:"globalMemoryUsage"`
		ActiveWorkers     int     `json:"activeWorkers"`
		QueuedTasks       int     `json:"queuedTasks"`
	} `json:"resources"`
}

// GetStrategyWorkflows 获取策略工作流状态
func (h *DualLoopHandler) GetStrategyWorkflows(w http.ResponseWriter, r *http.Request) {
	if !h.system.IsRunning() {
		http.Error(w, "System is not running", http.StatusServiceUnavailable)
		return
	}

	// 模拟数据（实际应该从多策略管理器获取）
	response := StrategyWorkflowResponse{}

	// 模拟工作流数据
	response.Workflows = []struct {
		ID                  string `json:"id"`
		Name                string `json:"name"`
		CurrentStage        string `json:"currentStage"`
		Progress            int    `json:"progress"`
		StartTime           string `json:"startTime"`
		EstimatedCompletion string `json:"estimatedCompletion"`
		ResourceUsage       struct {
			CPU    float64 `json:"cpu"`
			Memory float64 `json:"memory"`
		} `json:"resourceUsage"`
		Status string `json:"status"`
	}{
		{
			ID:                  "strategy_001",
			Name:                "动量策略Alpha",
			CurrentStage:        "backtesting",
			Progress:            65,
			StartTime:           "2小时前",
			EstimatedCompletion: "1小时后",
			ResourceUsage: struct {
				CPU    float64 `json:"cpu"`
				Memory float64 `json:"memory"`
			}{CPU: 1.2, Memory: 2.1},
			Status: "running",
		},
		// 可以添加更多工作流...
	}

	// 进化统计
	response.Evolution.CurrentGeneration = 47
	response.Evolution.PopulationSize = 20
	response.Evolution.BestFitness = 0.892
	response.Evolution.AverageFitness = 0.634
	response.Evolution.DiversityIndex = 0.78
	response.Evolution.EliminatedCount = 156

	// 资源使用
	response.Resources.GlobalCPUUsage = 6.3
	response.Resources.GlobalMemoryUsage = 12.4
	response.Resources.ActiveWorkers = 15
	response.Resources.QueuedTasks = 8

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// TradingExecutionResponse 交易执行响应
type TradingExecutionResponse struct {
	Execution struct {
		Latency          float64 `json:"latency"`
		Throughput       int     `json:"throughput"`
		SuccessRate      float64 `json:"successRate"`
		ErrorRate        float64 `json:"errorRate"`
		TotalExecutions  int     `json:"totalExecutions"`
		AvgExecutionTime float64 `json:"avgExecutionTime"`
	} `json:"execution"`
	Strategies []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		IsActive    bool   `json:"isActive"`
		Performance struct {
			PNL         float64 `json:"pnl"`
			SharpeRatio float64 `json:"sharpeRatio"`
			MaxDrawdown float64 `json:"maxDrawdown"`
			WinRate     float64 `json:"winRate"`
		} `json:"performance"`
		LastExecution  string `json:"lastExecution"`
		ExecutionCount int    `json:"executionCount"`
		Status         string `json:"status"`
	} `json:"strategies"`
	Risk struct {
		TotalExposure float64 `json:"totalExposure"`
		RiskLevel     string  `json:"riskLevel"`
		Violations    int     `json:"violations"`
		MaxDrawdown   float64 `json:"maxDrawdown"`
		Var95         float64 `json:"var95"`
	} `json:"risk"`
}

// GetTradingExecution 获取交易执行状态
func (h *DualLoopHandler) GetTradingExecution(w http.ResponseWriter, r *http.Request) {
	// 模拟交易执行数据
	response := TradingExecutionResponse{}

	// 执行指标
	response.Execution.Latency = 8.5
	response.Execution.Throughput = 1250
	response.Execution.SuccessRate = 98.7
	response.Execution.ErrorRate = 1.3
	response.Execution.TotalExecutions = 45678
	response.Execution.AvgExecutionTime = 12.3

	// 策略执行状态（模拟数据）
	response.Strategies = []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		IsActive    bool   `json:"isActive"`
		Performance struct {
			PNL         float64 `json:"pnl"`
			SharpeRatio float64 `json:"sharpeRatio"`
			MaxDrawdown float64 `json:"maxDrawdown"`
			WinRate     float64 `json:"winRate"`
		} `json:"performance"`
		LastExecution  string `json:"lastExecution"`
		ExecutionCount int    `json:"executionCount"`
		Status         string `json:"status"`
	}{
		{
			ID:       "strategy_001",
			Name:     "动量策略Alpha",
			IsActive: true,
			Performance: struct {
				PNL         float64 `json:"pnl"`
				SharpeRatio float64 `json:"sharpeRatio"`
				MaxDrawdown float64 `json:"maxDrawdown"`
				WinRate     float64 `json:"winRate"`
			}{
				PNL:         15420.50,
				SharpeRatio: 2.34,
				MaxDrawdown: 0.08,
				WinRate:     0.67,
			},
			LastExecution:  "2分钟前",
			ExecutionCount: 1234,
			Status:         "active",
		},
	}

	// 风险指标
	response.Risk.TotalExposure = 125000
	response.Risk.RiskLevel = "medium"
	response.Risk.Violations = 2
	response.Risk.MaxDrawdown = 0.15
	response.Risk.Var95 = 8500

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// StrategyPoolResponse 策略池响应
type StrategyPoolResponse struct {
	Distribution struct {
		Enabled  int `json:"enabled"`
		Disabled int `json:"disabled"`
		Pending  int `json:"pending"`
		Testing  int `json:"testing"`
	} `json:"distribution"`
	Strategies []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Type        string `json:"type"`
		Status      string `json:"status"`
		Performance struct {
			Score       float64 `json:"score"`
			SharpeRatio float64 `json:"sharpeRatio"`
			MaxDrawdown float64 `json:"maxDrawdown"`
			TotalReturn float64 `json:"totalReturn"`
		} `json:"performance"`
		Trend          string `json:"trend"`
		LastUpdated    string `json:"lastUpdated"`
		CreatedAt      string `json:"createdAt"`
		ExecutionCount int    `json:"executionCount"`
	} `json:"strategies"`
	Sync struct {
		LastSync   string `json:"lastSync"`
		SyncStatus string `json:"syncStatus"`
		Conflicts  int    `json:"conflicts"`
	} `json:"sync"`
}

// GetStrategyPool 获取策略池状态
func (h *DualLoopHandler) GetStrategyPool(w http.ResponseWriter, r *http.Request) {
	// 模拟策略池数据
	response := StrategyPoolResponse{}

	// 策略分布
	response.Distribution.Enabled = 12
	response.Distribution.Disabled = 134
	response.Distribution.Pending = 10
	response.Distribution.Testing = 8

	// 同步状态
	response.Sync.LastSync = "2分钟前"
	response.Sync.SyncStatus = "success"
	response.Sync.Conflicts = 0

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// formatDuration 格式化时间间隔
func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%d天 %d小时 %d分钟", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%d小时 %d分钟", hours, minutes)
	} else {
		return fmt.Sprintf("%d分钟", minutes)
	}
}
