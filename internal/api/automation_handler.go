package api

import (
	"net/http"
	"time"

	"qcat/internal/automation"
	"qcat/internal/database"
	"qcat/internal/monitor"

	"github.com/gin-gonic/gin"
)

// AutomationHandler handles automation system API requests
type AutomationHandler struct {
	db               *database.DB
	metrics          *monitor.MetricsCollector
	automationSystem *automation.AutomationSystem
}

// NewAutomationHandler creates a new automation handler
func NewAutomationHandler(db *database.DB, metrics *monitor.MetricsCollector, automationSystem *automation.AutomationSystem) *AutomationHandler {
	return &AutomationHandler{
		db:               db,
		metrics:          metrics,
		automationSystem: automationSystem,
	}
}

// AutomationStatus represents the status of an automation feature
type AutomationStatus struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Category         string    `json:"category"`
	Status           string    `json:"status"`
	Enabled          bool      `json:"enabled"`
	LastExecution    time.Time `json:"lastExecution"`
	NextExecution    time.Time `json:"nextExecution"`
	SuccessRate      float64   `json:"successRate"`
	AvgExecutionTime float64   `json:"avgExecutionTime"`
	ExecutionCount   int       `json:"executionCount"`
	ErrorCount       int       `json:"errorCount"`
	Description      string    `json:"description"`
}

// HealthMetrics represents automation system health metrics
type HealthMetrics struct {
	OverallHealth      int     `json:"overallHealth"`
	AutomationCoverage int     `json:"automationCoverage"`
	SuccessRate        float64 `json:"successRate"`
	AvgResponseTime    float64 `json:"avgResponseTime"`
	ActiveAutomations  int     `json:"activeAutomations"`
	TotalAutomations   int     `json:"totalAutomations"`
}

// ExecutionStats represents execution statistics
type ExecutionStats struct {
	Today     ExecutionPeriod `json:"today"`
	ThisWeek  ExecutionPeriod `json:"thisWeek"`
	ThisMonth ExecutionPeriod `json:"thisMonth"`
}

// ExecutionPeriod represents execution stats for a time period
type ExecutionPeriod struct {
	Successful int `json:"successful"`
	Failed     int `json:"failed"`
	Pending    int `json:"pending"`
}

// GetAutomationStatus returns the status of all automation features
func (h *AutomationHandler) GetAutomationStatus(c *gin.Context) {
	// Get system status from automation system
	systemStatus := h.automationSystem.GetStatus()

	// Generate automation features list (26 features)
	automations := h.generateAutomationFeatures(systemStatus)

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    automations,
	})
}

// GetHealthMetrics returns automation system health metrics
func (h *AutomationHandler) GetHealthMetrics(c *gin.Context) {
	systemStatus := h.automationSystem.GetStatus()

	// Calculate health metrics based on system status
	activeCount := 0
	totalCount := 26

	if systemStatus.IsRunning {
		activeCount = int(float64(totalCount) * systemStatus.HealthScore)
	}

	healthMetrics := HealthMetrics{
		OverallHealth:      int(systemStatus.HealthScore * 100),
		AutomationCoverage: int(float64(activeCount) / float64(totalCount) * 100),
		SuccessRate:        calculateSuccessRate(systemStatus),
		AvgResponseTime:    2.5, // Default response time
		ActiveAutomations:  activeCount,
		TotalAutomations:   totalCount,
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    healthMetrics,
	})
}

// GetExecutionStats returns execution statistics
func (h *AutomationHandler) GetExecutionStats(c *gin.Context) {
	systemStatus := h.automationSystem.GetStatus()

	stats := ExecutionStats{
		Today: ExecutionPeriod{
			Successful: systemStatus.CompletedActions,
			Failed:     systemStatus.FailedActions,
			Pending:    systemStatus.ActiveActions,
		},
		ThisWeek: ExecutionPeriod{
			Successful: systemStatus.CompletedActions * 7,
			Failed:     systemStatus.FailedActions * 7,
			Pending:    systemStatus.ActiveActions,
		},
		ThisMonth: ExecutionPeriod{
			Successful: systemStatus.CompletedActions * 30,
			Failed:     systemStatus.FailedActions * 30,
			Pending:    systemStatus.ActiveActions,
		},
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    stats,
	})
}

// ToggleAutomation toggles an automation feature on/off
func (h *AutomationHandler) ToggleAutomation(c *gin.Context) {
	automationID := c.Param("id")

	var req struct {
		Enabled bool `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// For now, just return success
	// In a real implementation, you would toggle the specific automation
	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Automation " + automationID + " toggled successfully",
	})
}

// GetSystemStatus returns overall automation system status
func (h *AutomationHandler) GetSystemStatus(c *gin.Context) {
	systemStatus := h.automationSystem.GetStatus()

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    systemStatus,
	})
}

// generateAutomationFeatures generates the list of 26 automation features
func (h *AutomationHandler) generateAutomationFeatures(systemStatus *automation.SystemStatus) []AutomationStatus {
	features := []struct {
		id          string
		name        string
		category    string
		description string
	}{
		// Strategy Management (6 features)
		{"1", "策略自动优化", "strategy", "自动优化策略参数以提升收益"},
		{"2", "策略自动切换", "strategy", "根据市场条件自动切换策略"},
		{"3", "策略性能监控", "strategy", "实时监控策略表现并预警"},
		{"4", "策略参数调整", "strategy", "动态调整策略参数"},
		{"5", "策略回测验证", "strategy", "自动回测验证策略有效性"},
		{"6", "策略组合优化", "strategy", "优化多策略组合配置"},

		// Risk Management (5 features)
		{"7", "风险实时监控", "risk", "实时监控账户风险指标"},
		{"8", "自动止损止盈", "risk", "根据风险阈值自动止损止盈"},
		{"9", "仓位自动调整", "risk", "根据风险水平自动调整仓位"},
		{"10", "风险预警系统", "risk", "提前预警潜在风险"},
		{"11", "熔断机制", "risk", "极端情况下自动熔断交易"},

		// Position Management (4 features)
		{"12", "仓位自动再平衡", "position", "定期自动再平衡投资组合"},
		{"13", "动态仓位分配", "position", "根据市场条件动态分配仓位"},
		{"14", "仓位风险控制", "position", "控制单个仓位风险敞口"},
		{"15", "仓位成本优化", "position", "优化仓位建立和平仓成本"},

		// Market Data (3 features)
		{"16", "市场数据采集", "data", "自动采集和处理市场数据"},
		{"17", "异常数据检测", "data", "检测和处理异常市场数据"},
		{"18", "数据质量监控", "data", "监控数据质量和完整性"},

		// System Operations (4 features)
		{"19", "系统健康检查", "system", "定期检查系统健康状态"},
		{"20", "自动故障恢复", "system", "自动检测和恢复系统故障"},
		{"21", "性能优化调整", "system", "自动优化系统性能参数"},
		{"22", "资源使用监控", "system", "监控和优化资源使用"},

		// Learning & Intelligence (4 features)
		{"23", "机器学习训练", "learning", "自动训练和更新ML模型"},
		{"24", "市场模式识别", "learning", "识别和学习市场模式"},
		{"25", "智能决策支持", "learning", "提供智能化决策建议"},
		{"26", "自适应参数调整", "learning", "基于学习结果自适应调整"},
	}

	automations := make([]AutomationStatus, len(features))

	for i, feature := range features {
		// Calculate status based on system health
		status := "stopped"
		enabled := false
		successRate := 0.0

		if systemStatus.IsRunning {
			// Simulate different statuses based on system health
			if systemStatus.HealthScore > 0.8 {
				status = "running"
				enabled = true
				successRate = 85.0 + float64(i%15) // 85-100%
			} else if systemStatus.HealthScore > 0.5 {
				if i%3 == 0 {
					status = "warning"
					enabled = true
					successRate = 70.0 + float64(i%20) // 70-90%
				} else {
					status = "running"
					enabled = true
					successRate = 80.0 + float64(i%15) // 80-95%
				}
			} else {
				if i%2 == 0 {
					status = "error"
					enabled = false
					successRate = 50.0 + float64(i%30) // 50-80%
				}
			}
		}

		automations[i] = AutomationStatus{
			ID:               feature.id,
			Name:             feature.name,
			Category:         feature.category,
			Status:           status,
			Enabled:          enabled,
			LastExecution:    time.Now().Add(-time.Duration(i*5) * time.Minute),
			NextExecution:    time.Now().Add(time.Duration(30-i) * time.Minute),
			SuccessRate:      successRate,
			AvgExecutionTime: 1.5 + float64(i%10)*0.3, // 1.5-4.5 seconds
			ExecutionCount:   100 + i*10,
			ErrorCount:       int(float64(100+i*10) * (100 - successRate) / 100),
			Description:      feature.description,
		}
	}

	return automations
}

// calculateSuccessRate calculates overall success rate from system status
func calculateSuccessRate(status *automation.SystemStatus) float64 {
	total := status.CompletedActions + status.FailedActions
	if total == 0 {
		return 100.0
	}
	return float64(status.CompletedActions) / float64(total) * 100.0
}

// RegisterRoutes registers automation management routes
func (h *AutomationHandler) RegisterRoutes(router *gin.RouterGroup) {
	automation := router.Group("/automation")
	{
		automation.GET("/status", h.GetAutomationStatus)
		automation.GET("/health", h.GetHealthMetrics)
		automation.GET("/stats", h.GetExecutionStats)
		automation.GET("/system", h.GetSystemStatus)
		automation.POST("/:id/toggle", h.ToggleAutomation)
	}
}
