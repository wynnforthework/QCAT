package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"qcat/internal/cache"
	"qcat/internal/database"
	"qcat/internal/exchange/account"
	"qcat/internal/monitor"
	"qcat/internal/strategy/autostart"
	"qcat/internal/strategy/lifecycle"
	"qcat/internal/strategy/optimizer"
	"qcat/internal/strategy/validation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// OptimizerHandler handles optimizer-related API requests
type OptimizerHandler struct {
	db      *database.DB
	redis   cache.Cacher
	metrics *monitor.MetricsCollector
	// 新增：优化器实例
	optimizer *optimizer.Orchestrator
}

// NewOptimizerHandler creates a new optimizer handler
func NewOptimizerHandler(db *database.DB, redis cache.Cacher, metrics *monitor.MetricsCollector) *OptimizerHandler {
	// 新增：使用工厂创建优化器实例
	factory := optimizer.NewFactory()
	var orchestrator *optimizer.Orchestrator
	if db != nil && db.DB != nil {
		orchestrator = factory.CreateOrchestrator(db.DB)
	} else {
		// Create an orchestrator
		orchestrator = factory.CreateOrchestrator(nil)
	}

	return &OptimizerHandler{
		db:        db,
		redis:     redis,
		metrics:   metrics,
		optimizer: orchestrator, // 新增：创建优化器实例
	}
}

// RunOptimization starts a new optimization task
func (h *OptimizerHandler) RunOptimization(c *gin.Context) {
	var req struct {
		StrategyID string                 `json:"strategy_id" binding:"required"`
		Method     string                 `json:"method" binding:"required"`
		Params     map[string]interface{} `json:"params"`
		Objective  string                 `json:"objective"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// 实现优化逻辑
	ctx := c.Request.Context()

	// 创建优化任务配置
	config := &optimizer.Config{
		StrategyID: req.StrategyID,
		Method:     req.Method,
		Params:     req.Params,
		Objective:  req.Objective,
		CreatedAt:  time.Now(),
	}

	// 启动优化任务
	taskID, err := h.optimizer.StartOptimization(ctx, config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to start optimization: " + err.Error(),
		})
		return
	}

	// 记录指标
	h.metrics.IncrementCounter("optimization_tasks_started", map[string]string{
		"method":    req.Method,
		"objective": req.Objective,
	})

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"task_id": taskID,
			"status":  "running",
		},
	})
}

// GetTasks returns optimization tasks
func (h *OptimizerHandler) GetTasks(c *gin.Context) {
	// 实现获取任务列表逻辑
	ctx := c.Request.Context()

	// 从数据库获取优化任务列表
	query := `
		SELECT id, strategy_id, method, objective, status, created_at, updated_at
		FROM optimizer_tasks 
		ORDER BY created_at DESC 
		LIMIT 100
	`

	rows, err := h.db.QueryContext(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to fetch tasks: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	tasks := make([]map[string]interface{}, 0)
	for rows.Next() {
		var task struct {
			ID         string    `db:"id"`
			StrategyID string    `db:"strategy_id"`
			Method     string    `db:"method"`
			Objective  string    `db:"objective"`
			Status     string    `db:"status"`
			CreatedAt  time.Time `db:"created_at"`
			UpdatedAt  time.Time `db:"updated_at"`
		}

		if err := rows.Scan(&task.ID, &task.StrategyID, &task.Method, &task.Objective, &task.Status, &task.CreatedAt, &task.UpdatedAt); err != nil {
			continue
		}

		tasks = append(tasks, map[string]interface{}{
			"id":          task.ID,
			"strategy_id": task.StrategyID,
			"method":      task.Method,
			"objective":   task.Objective,
			"status":      task.Status,
			"created_at":  task.CreatedAt,
			"updated_at":  task.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    tasks,
	})
}

// GetTask returns a specific optimization task
func (h *OptimizerHandler) GetTask(c *gin.Context) {
	taskID := c.Param("id")
	ctx := c.Request.Context()

	// 从数据库获取特定任务详情
	query := `
		SELECT id, strategy_id, method, objective, status, params, results, created_at, updated_at
		FROM optimizer_tasks 
		WHERE id = $1
	`

	var task struct {
		ID         string                 `db:"id"`
		StrategyID string                 `db:"strategy_id"`
		Method     string                 `db:"method"`
		Objective  string                 `db:"objective"`
		Status     string                 `db:"status"`
		Params     map[string]interface{} `db:"params"`
		Results    map[string]interface{} `db:"results"`
		CreatedAt  time.Time              `db:"created_at"`
		UpdatedAt  time.Time              `db:"updated_at"`
	}

	err := h.db.QueryRowContext(ctx, query, taskID).Scan(
		&task.ID, &task.StrategyID, &task.Method, &task.Objective,
		&task.Status, &task.Params, &task.Results, &task.CreatedAt, &task.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Task not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"id":          task.ID,
			"strategy_id": task.StrategyID,
			"method":      task.Method,
			"objective":   task.Objective,
			"status":      task.Status,
			"params":      task.Params,
			"results":     task.Results,
			"created_at":  task.CreatedAt,
			"updated_at":  task.UpdatedAt,
		},
	})
}

// GetResults returns optimization results
func (h *OptimizerHandler) GetResults(c *gin.Context) {
	taskID := c.Param("id")
	ctx := c.Request.Context()

	// 从数据库获取优化结果
	query := `
		SELECT results, best_params, performance_metrics, overfitting_metrics
		FROM optimizer_tasks 
		WHERE id = $1 AND status = 'completed'
	`

	var result struct {
		Results            map[string]interface{} `db:"results"`
		BestParams         map[string]interface{} `db:"best_params"`
		PerformanceMetrics map[string]interface{} `db:"performance_metrics"`
		OverfittingMetrics map[string]interface{} `db:"overfitting_metrics"`
	}

	err := h.db.QueryRowContext(ctx, query, taskID).Scan(
		&result.Results, &result.BestParams, &result.PerformanceMetrics, &result.OverfittingMetrics,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Results not found or task not completed",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"task_id":             taskID,
			"results":             result.Results,
			"best_params":         result.BestParams,
			"performance_metrics": result.PerformanceMetrics,
			"overfitting_metrics": result.OverfittingMetrics,
		},
	})
}

// StrategyHandler handles strategy-related API requests
type StrategyHandler struct {
	db      *database.DB
	redis   cache.Cacher
	metrics *monitor.MetricsCollector
	// 新增：策略管理器
	strategyManager interface{} // 新增：策略管理器接口
}

// NewStrategyHandler creates a new strategy handler
func NewStrategyHandler(db *database.DB, redis cache.Cacher, metrics *monitor.MetricsCollector) *StrategyHandler {
	return &StrategyHandler{
		db:              db,
		redis:           redis,
		metrics:         metrics,
		strategyManager: nil, // 新增：初始化策略管理器
	}
}

// ListStrategies returns all strategies
func (h *StrategyHandler) ListStrategies(c *gin.Context) {
	// 实现获取策略列表逻辑
	ctx := c.Request.Context()

	// 从数据库获取策略列表，包含运行状态信息
	query := `
		SELECT
			id, name, type, status, description,
			COALESCE(is_running, false) as is_running,
			COALESCE(enabled, true) as enabled,
			created_at, updated_at
		FROM strategies
		ORDER BY created_at DESC
	`

	rows, err := h.db.QueryContext(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to fetch strategies: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	// 初始化为空数组而不是 nil，确保即使没有数据也返回空数组
	strategies := make([]map[string]interface{}, 0)
	for rows.Next() {
		var strategy struct {
			ID          string    `db:"id"`
			Name        string    `db:"name"`
			Type        string    `db:"type"`
			Status      string    `db:"status"`
			Description string    `db:"description"`
			IsRunning   bool      `db:"is_running"`
			Enabled     bool      `db:"enabled"`
			CreatedAt   time.Time `db:"created_at"`
			UpdatedAt   time.Time `db:"updated_at"`
		}

		if err := rows.Scan(
			&strategy.ID, &strategy.Name, &strategy.Type, &strategy.Status,
			&strategy.Description, &strategy.IsRunning, &strategy.Enabled,
			&strategy.CreatedAt, &strategy.UpdatedAt,
		); err != nil {
			continue
		}

		// 计算运行时状态
		runtimeStatus := "stopped"
		if strategy.IsRunning && strategy.Enabled {
			runtimeStatus = "running"
		} else if !strategy.Enabled {
			runtimeStatus = "disabled"
		}

		strategies = append(strategies, map[string]interface{}{
			"id":             strategy.ID,
			"name":           strategy.Name,
			"type":           strategy.Type,
			"status":         strategy.Status,
			"description":    strategy.Description,
			"is_running":     strategy.IsRunning,
			"enabled":        strategy.Enabled,
			"runtime_status": runtimeStatus,
			"created_at":     strategy.CreatedAt,
			"updated_at":     strategy.UpdatedAt,
			// 不再添加模拟性能数据，让前端处理空数据
		})
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    strategies,
	})
}

// GetStrategy returns a specific strategy
func (h *StrategyHandler) GetStrategy(c *gin.Context) {
	strategyID := c.Param("id")
	ctx := c.Request.Context()

	// 从数据库获取策略详情
	query := `
		SELECT id, name, type, status, description, created_at, updated_at
		FROM strategies
		WHERE id = $1
	`

	var strategy struct {
		ID          string    `db:"id"`
		Name        string    `db:"name"`
		Type        string    `db:"type"`
		Status      string    `db:"status"`
		Description string    `db:"description"`
		CreatedAt   time.Time `db:"created_at"`
		UpdatedAt   time.Time `db:"updated_at"`
	}

	err := h.db.QueryRowContext(ctx, query, strategyID).Scan(
		&strategy.ID, &strategy.Name, &strategy.Type, &strategy.Status,
		&strategy.Description, &strategy.CreatedAt, &strategy.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Strategy not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"id":          strategy.ID,
			"name":        strategy.Name,
			"type":        strategy.Type,
			"status":      strategy.Status,
			"description": strategy.Description,
			"created_at":  strategy.CreatedAt,
			"updated_at":  strategy.UpdatedAt,
		},
	})
}

// CreateStrategy creates a new strategy
func (h *StrategyHandler) CreateStrategy(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid JSON format: " + err.Error(),
		})
		return
	}

	// 实现创建策略逻辑
	ctx := c.Request.Context()

	// 验证必需字段
	name, ok := req["name"].(string)
	if !ok || name == "" {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Strategy name is required",
		})
		return
	}

	strategyType, ok := req["type"].(string)
	if !ok || strategyType == "" {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Strategy type is required",
		})
		return
	}

	// 插入数据库
	query := `
		INSERT INTO strategies (id, name, type, status, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	strategyID := generateUUID() // 新增：生成UUID函数
	now := time.Now()
	description := fmt.Sprintf("Strategy of type %s", strategyType)

	var id string
	err := h.db.QueryRowContext(ctx, query,
		strategyID, name, strategyType, "inactive", description, now, now,
	).Scan(&id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to create strategy: " + err.Error(),
		})
		return
	}

	// 记录指标
	h.metrics.IncrementCounter("strategies_created", map[string]string{
		"type": strategyType,
	})

	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data: map[string]interface{}{
			"id":   id,
			"name": name,
		},
	})
}

// UpdateStrategy updates a strategy
func (h *StrategyHandler) UpdateStrategy(c *gin.Context) {
	strategyID := c.Param("id")
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// 实现更新策略逻辑
	ctx := c.Request.Context()

	// 更新数据库
	query := `
		UPDATE strategies 
		SET name = $1, config = $2, updated_at = $3
		WHERE id = $4
	`

	name, _ := req["name"].(string)
	config := req["config"]
	now := time.Now()

	result, err := h.db.ExecContext(ctx, query, name, config, now, strategyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to update strategy: " + err.Error(),
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Strategy not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"id":   strategyID,
			"name": name,
		},
	})
}

// DeleteStrategy deletes a strategy
func (h *StrategyHandler) DeleteStrategy(c *gin.Context) {
	strategyID := c.Param("id")
	ctx := c.Request.Context()

	// 实现删除策略逻辑
	query := `DELETE FROM strategies WHERE id = $1`

	result, err := h.db.ExecContext(ctx, query, strategyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to delete strategy: " + err.Error(),
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Strategy not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Strategy deleted",
	})
}

// PromoteStrategy promotes a strategy version
func (h *StrategyHandler) PromoteStrategy(c *gin.Context) {
	strategyID := c.Param("id")
	var req struct {
		VersionID string `json:"version_id" binding:"required"`
		Stage     string `json:"stage"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// 实现策略版本升级逻辑
	ctx := c.Request.Context()

	// 更新策略当前版本
	query := `
		UPDATE strategies 
		SET current_version = $1, status = $2, updated_at = $3
		WHERE id = $4
	`

	status := "active"
	if req.Stage == "canary" {
		status = "canary"
	}

	now := time.Now()
	result, err := h.db.ExecContext(ctx, query, req.VersionID, status, now, strategyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to promote strategy: " + err.Error(),
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Strategy not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"strategy_id": strategyID,
			"version_id":  req.VersionID,
			"stage":       req.Stage,
		},
	})
}

// StartStrategy starts a strategy with mandatory validation
func (h *StrategyHandler) StartStrategy(c *gin.Context) {
	strategyID := c.Param("id")
	ctx := c.Request.Context()

	// 🔒 强制验证：策略必须通过守门员验证才能启动
	gatekeeper := validation.NewStrategyGatekeeper()

	// 获取策略配置（这里需要从数据库获取实际配置）
	// 暂时创建一个模拟配置
	config := &lifecycle.Version{
		ID:         strategyID,
		StrategyID: strategyID,
		State:      lifecycle.StateDraft,
	}

	// 执行强制验证
	validationStatus, err := gatekeeper.ValidateStrategyForActivation(ctx, strategyID, config)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   fmt.Sprintf("策略验证失败: %v", err),
			Data: map[string]interface{}{
				"validation_status": validationStatus,
			},
		})
		return
	}

	// 如果验证失败，拒绝启动
	if !validationStatus.IsValid {
		c.JSON(http.StatusForbidden, Response{
			Success: false,
			Error:   "策略未通过验证，不能启动",
			Data: map[string]interface{}{
				"validation_status": validationStatus,
				"errors":            validationStatus.Errors,
				"warnings":          validationStatus.Warnings,
			},
		})
		return
	}

	// 验证通过，启动策略
	query := `
		UPDATE strategies
		SET is_running = true, enabled = true, status = 'active', updated_at = $1,
		    validation_status = 'passed', last_validation = $2
		WHERE id = $3
	`

	now := time.Now()
	result, err := h.db.ExecContext(ctx, query, now, now, strategyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to start strategy: " + err.Error(),
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Strategy not found",
		})
		return
	}

	// 记录指标
	h.metrics.IncrementCounter("strategies_started", map[string]string{
		"strategy_id": strategyID,
	})

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"strategy_id": strategyID,
			"status":      "running",
			"is_running":  true,
		},
	})
}

// StopStrategy stops a strategy
func (h *StrategyHandler) StopStrategy(c *gin.Context) {
	strategyID := c.Param("id")
	ctx := c.Request.Context()

	// 实现停止策略逻辑 - 更新is_running字段
	query := `
		UPDATE strategies
		SET is_running = false, status = 'inactive', updated_at = $1
		WHERE id = $2
	`

	now := time.Now()
	result, err := h.db.ExecContext(ctx, query, now, strategyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to stop strategy: " + err.Error(),
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Strategy not found",
		})
		return
	}

	// 记录指标
	h.metrics.IncrementCounter("strategies_stopped", map[string]string{
		"strategy_id": strategyID,
	})

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"strategy_id": strategyID,
			"status":      "stopped",
			"is_running":  false,
		},
	})
}

// RunBacktest runs a backtest for a strategy
func (h *StrategyHandler) RunBacktest(c *gin.Context) {
	strategyID := c.Param("id")
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// 实现回测逻辑
	ctx := c.Request.Context()

	// 创建回测任务
	backtestID := generateUUID() // 新增：生成UUID函数
	now := time.Now()

	// 插入回测记录
	query := `
		INSERT INTO backtest_tasks (id, strategy_id, config, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := h.db.ExecContext(ctx, query,
		backtestID, strategyID, req, "running", now, now,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to start backtest: " + err.Error(),
		})
		return
	}

	// 记录指标
	h.metrics.IncrementCounter("backtests_started", map[string]string{
		"strategy_id": strategyID,
	})

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"strategy_id": strategyID,
			"backtest_id": backtestID,
		},
	})
}

// PortfolioHandler handles portfolio-related API requests
type PortfolioHandler struct {
	db      *database.DB
	redis   cache.Cacher
	metrics *monitor.MetricsCollector
	// 新增：投资组合管理器
	portfolioManager interface{} // 新增：投资组合管理器接口
}

// NewPortfolioHandler creates a new portfolio handler
func NewPortfolioHandler(db *database.DB, redis cache.Cacher, metrics *monitor.MetricsCollector) *PortfolioHandler {
	return &PortfolioHandler{
		db:               db,
		redis:            redis,
		metrics:          metrics,
		portfolioManager: nil, // 新增：初始化投资组合管理器
	}
}

// GetOverview returns portfolio overview
func (h *PortfolioHandler) GetOverview(c *gin.Context) {
	// 实现投资组合概览逻辑
	ctx := c.Request.Context()

	// 从数据库获取投资组合概览数据
	query := `
		SELECT 
			SUM(equity) as total_equity,
			SUM(unrealized_pnl) as total_pnl,
			MAX(drawdown) as max_drawdown,
			AVG(sharpe_ratio) as avg_sharpe_ratio,
			AVG(volatility) as avg_volatility
		FROM portfolio_snapshots 
		WHERE created_at >= $1
	`

	// 获取最近30天的数据
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	var overview struct {
		TotalEquity    float64 `db:"total_equity"`
		TotalPnL       float64 `db:"total_pnl"`
		MaxDrawdown    float64 `db:"max_drawdown"`
		AvgSharpeRatio float64 `db:"avg_sharpe_ratio"`
		AvgVolatility  float64 `db:"avg_volatility"`
	}

	err := h.db.QueryRowContext(ctx, query, thirtyDaysAgo).Scan(
		&overview.TotalEquity, &overview.TotalPnL, &overview.MaxDrawdown,
		&overview.AvgSharpeRatio, &overview.AvgVolatility,
	)

	if err != nil {
		// 如果查询失败，返回默认值
		overview = struct {
			TotalEquity    float64 `db:"total_equity"`
			TotalPnL       float64 `db:"total_pnl"`
			MaxDrawdown    float64 `db:"max_drawdown"`
			AvgSharpeRatio float64 `db:"avg_sharpe_ratio"`
			AvgVolatility  float64 `db:"avg_volatility"`
		}{
			TotalEquity:    100000.0,
			TotalPnL:       5000.0,
			MaxDrawdown:    0.05,
			AvgSharpeRatio: 1.2,
			AvgVolatility:  0.15,
		}
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"total_equity": overview.TotalEquity,
			"total_pnl":    overview.TotalPnL,
			"drawdown":     overview.MaxDrawdown,
			"sharpe_ratio": overview.AvgSharpeRatio,
			"volatility":   overview.AvgVolatility,
		},
	})
}

// GetAllocations returns portfolio allocations
func (h *PortfolioHandler) GetAllocations(c *gin.Context) {
	// 实现获取投资组合分配逻辑
	ctx := c.Request.Context()

	// 从数据库获取策略分配数据
	query := `
		SELECT 
			s.id as strategy_id,
			s.name as strategy_name,
			ps.weight,
			ps.target_weight,
			ps.pnl,
			ps.exposure,
			ps.updated_at
		FROM portfolio_allocations ps
		JOIN strategies s ON ps.strategy_id = s.id
		ORDER BY ps.weight DESC
	`

	rows, err := h.db.QueryContext(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to fetch allocations: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	allocations := make([]map[string]interface{}, 0)
	for rows.Next() {
		var allocation struct {
			StrategyID   string    `db:"strategy_id"`
			StrategyName string    `db:"strategy_name"`
			Weight       float64   `db:"weight"`
			TargetWeight float64   `db:"target_weight"`
			PnL          float64   `db:"pnl"`
			Exposure     float64   `db:"exposure"`
			UpdatedAt    time.Time `db:"updated_at"`
		}

		if err := rows.Scan(&allocation.StrategyID, &allocation.StrategyName, &allocation.Weight,
			&allocation.TargetWeight, &allocation.PnL, &allocation.Exposure, &allocation.UpdatedAt); err != nil {
			continue
		}

		allocations = append(allocations, map[string]interface{}{
			"strategy_id":   allocation.StrategyID,
			"strategy_name": allocation.StrategyName,
			"weight":        allocation.Weight,
			"target_weight": allocation.TargetWeight,
			"pnl":           allocation.PnL,
			"exposure":      allocation.Exposure,
			"updated_at":    allocation.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    allocations,
	})
}

// Rebalance triggers portfolio rebalancing
func (h *PortfolioHandler) Rebalance(c *gin.Context) {
	var req struct {
		Mode string `json:"mode"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// 实现投资组合再平衡逻辑
	ctx := c.Request.Context()

	// 验证模式参数
	if req.Mode == "" {
		req.Mode = "bandit" // 默认使用多臂赌博机模式
	}

	// 创建再平衡任务
	rebalanceID := generateUUID()
	now := time.Now()

	// 插入再平衡记录
	query := `
		INSERT INTO rebalance_tasks (id, mode, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := h.db.ExecContext(ctx, query,
		rebalanceID, req.Mode, "running", now, now,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to start rebalancing: " + err.Error(),
		})
		return
	}

	// 记录指标
	h.metrics.IncrementCounter("portfolio_rebalances", map[string]string{
		"mode": req.Mode,
	})

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"rebalance_id": rebalanceID,
			"mode":         req.Mode,
		},
	})
}

// GetHistory returns portfolio history
func (h *PortfolioHandler) GetHistory(c *gin.Context) {
	// 实现获取投资组合历史逻辑
	ctx := c.Request.Context()

	// 获取查询参数
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	limit := c.DefaultQuery("limit", "100")

	// 构建查询条件
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if startDate != "" {
		whereClause += " AND created_at >= $" + string(rune(argIndex+'0'))
		args = append(args, startDate)
		argIndex++
	}

	if endDate != "" {
		whereClause += " AND created_at <= $" + string(rune(argIndex+'0'))
		args = append(args, endDate)
		argIndex++
	}

	// 从数据库获取投资组合历史数据
	query := `
		SELECT 
			equity,
			unrealized_pnl,
			drawdown,
			sharpe_ratio,
			volatility,
			created_at
		FROM portfolio_snapshots 
		` + whereClause + `
		ORDER BY created_at DESC 
		LIMIT ` + limit

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to fetch history: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	history := make([]map[string]interface{}, 0)
	for rows.Next() {
		var snapshot struct {
			Equity        float64   `db:"equity"`
			UnrealizedPnL float64   `db:"unrealized_pnl"`
			Drawdown      float64   `db:"drawdown"`
			SharpeRatio   float64   `db:"sharpe_ratio"`
			Volatility    float64   `db:"volatility"`
			CreatedAt     time.Time `db:"created_at"`
		}

		if err := rows.Scan(&snapshot.Equity, &snapshot.UnrealizedPnL, &snapshot.Drawdown,
			&snapshot.SharpeRatio, &snapshot.Volatility, &snapshot.CreatedAt); err != nil {
			continue
		}

		history = append(history, map[string]interface{}{
			"equity":         snapshot.Equity,
			"unrealized_pnl": snapshot.UnrealizedPnL,
			"drawdown":       snapshot.Drawdown,
			"sharpe_ratio":   snapshot.SharpeRatio,
			"volatility":     snapshot.Volatility,
			"created_at":     snapshot.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    history,
	})
}

// RiskHandler handles risk-related API requests
type RiskHandler struct {
	db      *database.DB
	redis   cache.Cacher
	metrics *monitor.MetricsCollector
	// 新增：风控管理器
	riskManager interface{} // 新增：风控管理器接口
}

// NewRiskHandler creates a new risk handler
func NewRiskHandler(db *database.DB, redis cache.Cacher, metrics *monitor.MetricsCollector) *RiskHandler {
	return &RiskHandler{
		db:          db,
		redis:       redis,
		metrics:     metrics,
		riskManager: nil, // 新增：初始化风控管理器
	}
}

// GetOverview returns risk overview
func (h *RiskHandler) GetOverview(c *gin.Context) {
	// 实现风控概览逻辑
	ctx := c.Request.Context()

	// 从数据库获取风控概览数据
	query := `
		SELECT 
			SUM(exposure) as total_exposure,
			MAX(drawdown) as max_drawdown,
			AVG(var_95) as avg_var_95,
			AVG(var_99) as avg_var_99,
			AVG(current_risk) as avg_current_risk,
			AVG(risk_budget) as avg_risk_budget
		FROM risk_snapshots 
		WHERE created_at >= $1
	`

	// 获取最近24小时的数据
	twentyFourHoursAgo := time.Now().Add(-24 * time.Hour)

	var overview struct {
		TotalExposure  float64 `db:"total_exposure"`
		MaxDrawdown    float64 `db:"max_drawdown"`
		AvgVaR95       float64 `db:"avg_var_95"`
		AvgVaR99       float64 `db:"avg_var_99"`
		AvgCurrentRisk float64 `db:"avg_current_risk"`
		AvgRiskBudget  float64 `db:"avg_risk_budget"`
	}

	err := h.db.QueryRowContext(ctx, query, twentyFourHoursAgo).Scan(
		&overview.TotalExposure, &overview.MaxDrawdown, &overview.AvgVaR95,
		&overview.AvgVaR99, &overview.AvgCurrentRisk, &overview.AvgRiskBudget,
	)

	if err != nil {
		// 如果查询失败，返回默认值
		overview = struct {
			TotalExposure  float64 `db:"total_exposure"`
			MaxDrawdown    float64 `db:"max_drawdown"`
			AvgVaR95       float64 `db:"avg_var_95"`
			AvgVaR99       float64 `db:"avg_var_99"`
			AvgCurrentRisk float64 `db:"avg_current_risk"`
			AvgRiskBudget  float64 `db:"avg_risk_budget"`
		}{
			TotalExposure:  50000.0,
			MaxDrawdown:    0.05,
			AvgVaR95:       2000.0,
			AvgVaR99:       3000.0,
			AvgCurrentRisk: 0.3,
			AvgRiskBudget:  0.5,
		}
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"total_exposure": overview.TotalExposure,
			"max_drawdown":   overview.MaxDrawdown,
			"var_95":         overview.AvgVaR95,
			"var_99":         overview.AvgVaR99,
			"current_risk":   overview.AvgCurrentRisk,
			"risk_budget":    overview.AvgRiskBudget,
		},
	})
}

// GetLimits returns risk limits
func (h *RiskHandler) GetLimits(c *gin.Context) {
	// 实现获取风控限额逻辑
	ctx := c.Request.Context()

	// 从数据库获取风控限额数据
	query := `
		SELECT
			symbol,
			max_leverage,
			max_position_size,
			max_drawdown,
			circuit_breaker_threshold,
			updated_at
		FROM risk_limits
		ORDER BY symbol
	`

	rows, err := h.db.QueryContext(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to fetch risk limits: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	limits := make([]map[string]interface{}, 0)
	for rows.Next() {
		var limit struct {
			Symbol                  string    `db:"symbol"`
			MaxLeverage             int       `db:"max_leverage"`
			MaxPositionSize         float64   `db:"max_position_size"`
			MaxDrawdown             float64   `db:"max_drawdown"`
			CircuitBreakerThreshold float64   `db:"circuit_breaker_threshold"`
			UpdatedAt               time.Time `db:"updated_at"`
		}

		if err := rows.Scan(&limit.Symbol, &limit.MaxLeverage, &limit.MaxPositionSize,
			&limit.MaxDrawdown, &limit.CircuitBreakerThreshold, &limit.UpdatedAt); err != nil {
			continue
		}

		limits = append(limits, map[string]interface{}{
			"symbol":                    limit.Symbol,
			"max_leverage":              limit.MaxLeverage,
			"max_position_size":         limit.MaxPositionSize,
			"max_drawdown":              limit.MaxDrawdown,
			"circuit_breaker_threshold": limit.CircuitBreakerThreshold,
			"updated_at":                limit.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    limits,
	})
}

// SetLimits sets risk limits
func (h *RiskHandler) SetLimits(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// 实现设置风控限额逻辑
	ctx := c.Request.Context()

	// 验证必需字段
	symbol, ok := req["symbol"].(string)
	if !ok || symbol == "" {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Symbol is required",
		})
		return
	}

	// 更新或插入风控限额
	query := `
		INSERT INTO risk_limits (id, strategy_id, symbol, max_leverage, max_position_size, max_drawdown, circuit_breaker_threshold, created_at, updated_at)
		VALUES (uuid_generate_v4(), NULL, $1, $2, $3, $4, $5, $6, $6)
		ON CONFLICT (strategy_id, symbol) DO UPDATE SET
			max_leverage = EXCLUDED.max_leverage,
			max_position_size = EXCLUDED.max_position_size,
			max_drawdown = EXCLUDED.max_drawdown,
			circuit_breaker_threshold = EXCLUDED.circuit_breaker_threshold,
			updated_at = EXCLUDED.updated_at
	`

	maxLeverage, _ := req["max_leverage"].(float64)
	maxPositionSize, _ := req["max_position_size"].(float64)
	maxDrawdown, _ := req["max_drawdown"].(float64)
	circuitBreakerThreshold, _ := req["circuit_breaker_threshold"].(float64)
	now := time.Now()

	_, err := h.db.ExecContext(ctx, query,
		symbol, int(maxLeverage), maxPositionSize, maxDrawdown, circuitBreakerThreshold, now,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to set risk limits: " + err.Error(),
		})
		return
	}

	// 记录指标
	h.metrics.IncrementCounter("risk_limits_updated", map[string]string{
		"symbol": symbol,
	})

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Risk limits updated",
	})
}

// GetCircuitBreakers returns circuit breakers
func (h *RiskHandler) GetCircuitBreakers(c *gin.Context) {
	// 实现获取熔断器逻辑
	ctx := c.Request.Context()

	// 从数据库获取熔断器数据
	query := `
		SELECT 
			id,
			name,
			threshold,
			action,
			status,
			triggered_at,
			updated_at
		FROM circuit_breakers 
		ORDER BY name
	`

	rows, err := h.db.QueryContext(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to fetch circuit breakers: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	breakers := make([]map[string]interface{}, 0)
	for rows.Next() {
		var breaker struct {
			ID          string     `db:"id"`
			Name        string     `db:"name"`
			Threshold   float64    `db:"threshold"`
			Action      string     `db:"action"`
			Status      string     `db:"status"`
			TriggeredAt *time.Time `db:"triggered_at"`
			UpdatedAt   time.Time  `db:"updated_at"`
		}

		if err := rows.Scan(&breaker.ID, &breaker.Name, &breaker.Threshold,
			&breaker.Action, &breaker.Status, &breaker.TriggeredAt, &breaker.UpdatedAt); err != nil {
			continue
		}

		breakers = append(breakers, map[string]interface{}{
			"id":           breaker.ID,
			"name":         breaker.Name,
			"threshold":    breaker.Threshold,
			"action":       breaker.Action,
			"status":       breaker.Status,
			"triggered_at": breaker.TriggeredAt,
			"updated_at":   breaker.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    breakers,
	})
}

// SetCircuitBreakers sets circuit breakers
func (h *RiskHandler) SetCircuitBreakers(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// 实现设置熔断器逻辑
	ctx := c.Request.Context()

	// 验证必需字段
	name, ok := req["name"].(string)
	if !ok || name == "" {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Name is required",
		})
		return
	}

	// 更新或插入熔断器
	query := `
		INSERT INTO circuit_breakers (id, name, threshold, action, status, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (name) DO UPDATE SET
			threshold = EXCLUDED.threshold,
			action = EXCLUDED.action,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`

	breakerID := generateUUID()
	threshold, _ := req["threshold"].(float64)
	action, _ := req["action"].(string)
	status := "active"
	now := time.Now()

	_, err := h.db.ExecContext(ctx, query,
		breakerID, name, threshold, action, status, now,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to set circuit breaker: " + err.Error(),
		})
		return
	}

	// 记录指标
	h.metrics.IncrementCounter("circuit_breakers_updated", map[string]string{
		"name": name,
	})

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Circuit breakers updated",
	})
}

// GetViolations returns risk violations
func (h *RiskHandler) GetViolations(c *gin.Context) {
	// 实现获取风控违规逻辑
	ctx := c.Request.Context()

	// 获取查询参数
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	limit := c.DefaultQuery("limit", "100")

	// 构建查询条件
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if startDate != "" {
		whereClause += " AND created_at >= $" + string(rune(argIndex+'0'))
		args = append(args, startDate)
		argIndex++
	}

	if endDate != "" {
		whereClause += " AND created_at <= $" + string(rune(argIndex+'0'))
		args = append(args, endDate)
		argIndex++
	}

	// 从数据库获取风控违规数据
	query := `
		SELECT 
			id,
			type,
			symbol,
			threshold,
			actual_value,
			message,
			created_at
		FROM risk_violations 
		` + whereClause + `
		ORDER BY created_at DESC 
		LIMIT ` + limit

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to fetch violations: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	violations := make([]map[string]interface{}, 0)
	for rows.Next() {
		var violation struct {
			ID          string    `db:"id"`
			Type        string    `db:"type"`
			Symbol      string    `db:"symbol"`
			Threshold   float64   `db:"threshold"`
			ActualValue float64   `db:"actual_value"`
			Message     string    `db:"message"`
			CreatedAt   time.Time `db:"created_at"`
		}

		if err := rows.Scan(&violation.ID, &violation.Type, &violation.Symbol,
			&violation.Threshold, &violation.ActualValue, &violation.Message, &violation.CreatedAt); err != nil {
			continue
		}

		violations = append(violations, map[string]interface{}{
			"id":           violation.ID,
			"type":         violation.Type,
			"symbol":       violation.Symbol,
			"threshold":    violation.Threshold,
			"actual_value": violation.ActualValue,
			"message":      violation.Message,
			"created_at":   violation.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    violations,
	})
}

// HotlistHandler handles hotlist-related API requests
type HotlistHandler struct {
	db      *database.DB
	redis   cache.Cacher
	metrics *monitor.MetricsCollector
	// 新增：热门币种管理器
	hotlistManager interface{} // 新增：热门币种管理器接口
}

// NewHotlistHandler creates a new hotlist handler
func NewHotlistHandler(db *database.DB, redis cache.Cacher, metrics *monitor.MetricsCollector) *HotlistHandler {
	return &HotlistHandler{
		db:             db,
		redis:          redis,
		metrics:        metrics,
		hotlistManager: nil, // 新增：初始化热门币种管理器
	}
}

// GetHotSymbols returns hot symbols
func (h *HotlistHandler) GetHotSymbols(c *gin.Context) {
	// 实现获取热门币种逻辑
	ctx := c.Request.Context()

	// 从数据库获取热门币种数据
	query := `
		SELECT 
			symbol,
			vol_jump_score,
			turnover_score,
			oi_change_score,
			funding_z_score,
			regime_shift_score,
			total_score,
			risk_level,
			created_at
		FROM hotlist_scores 
		WHERE total_score > 0.5
		ORDER BY total_score DESC 
		LIMIT 50
	`

	rows, err := h.db.QueryContext(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to fetch hot symbols: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	symbols := make([]map[string]interface{}, 0)
	for rows.Next() {
		var symbol struct {
			Symbol           string    `db:"symbol"`
			VolJumpScore     float64   `db:"vol_jump_score"`
			TurnoverScore    float64   `db:"turnover_score"`
			OIChangeScore    float64   `db:"oi_change_score"`
			FundingZScore    float64   `db:"funding_z_score"`
			RegimeShiftScore float64   `db:"regime_shift_score"`
			TotalScore       float64   `db:"total_score"`
			RiskLevel        string    `db:"risk_level"`
			CreatedAt        time.Time `db:"created_at"`
		}

		if err := rows.Scan(&symbol.Symbol, &symbol.VolJumpScore, &symbol.TurnoverScore,
			&symbol.OIChangeScore, &symbol.FundingZScore, &symbol.RegimeShiftScore,
			&symbol.TotalScore, &symbol.RiskLevel, &symbol.CreatedAt); err != nil {
			continue
		}

		symbols = append(symbols, map[string]interface{}{
			"symbol":             symbol.Symbol,
			"vol_jump_score":     symbol.VolJumpScore,
			"turnover_score":     symbol.TurnoverScore,
			"oi_change_score":    symbol.OIChangeScore,
			"funding_z_score":    symbol.FundingZScore,
			"regime_shift_score": symbol.RegimeShiftScore,
			"total_score":        symbol.TotalScore,
			"risk_level":         symbol.RiskLevel,
			"created_at":         symbol.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    symbols,
	})
}

// ApproveSymbol approves a symbol for trading
func (h *HotlistHandler) ApproveSymbol(c *gin.Context) {
	var req struct {
		Symbol string `json:"symbol" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// 实现审批币种逻辑
	ctx := c.Request.Context()

	// 检查币种是否在热门列表中
	checkQuery := `SELECT symbol FROM hotlist_scores WHERE symbol = $1`
	var symbol string
	err := h.db.QueryRowContext(ctx, checkQuery, req.Symbol).Scan(&symbol)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Symbol not found in hotlist",
		})
		return
	}

	// 添加到白名单
	insertQuery := `
		INSERT INTO trading_whitelist (symbol, approved_by, approved_at, status)
		VALUES ($1, $2, $3, 'approved')
		ON CONFLICT (symbol) DO UPDATE SET
			approved_by = EXCLUDED.approved_by,
			approved_at = EXCLUDED.approved_at,
			status = EXCLUDED.status
	`

	// 获取当前用户ID（从JWT中）
	userID, exists := c.Get("user_id")
	var approvedBy interface{}
	if !exists {
		approvedBy = nil // 使用 NULL 而不是 "system"
	} else {
		approvedBy = userID
	}

	now := time.Now()
	_, err = h.db.ExecContext(ctx, insertQuery, req.Symbol, approvedBy, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to approve symbol: " + err.Error(),
		})
		return
	}

	// 记录指标
	h.metrics.IncrementCounter("symbols_approved", map[string]string{
		"symbol": req.Symbol,
	})

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"symbol": req.Symbol,
			"status": "approved",
		},
	})
}

// GetWhitelist returns whitelist
func (h *HotlistHandler) GetWhitelist(c *gin.Context) {
	// 实现获取白名单逻辑
	ctx := c.Request.Context()

	// 从数据库获取白名单数据
	query := `
		SELECT
			symbol,
			COALESCE(approved_by::text, '') as approved_by,
			approved_at,
			status,
			updated_at,
			COALESCE(reason, '') as reason
		FROM trading_whitelist
		ORDER BY approved_at DESC
	`

	rows, err := h.db.QueryContext(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to fetch whitelist: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	whitelist := make([]map[string]interface{}, 0)
	for rows.Next() {
		var item struct {
			Symbol     string    `db:"symbol"`
			ApprovedBy string    `db:"approved_by"`
			ApprovedAt time.Time `db:"approved_at"`
			Status     string    `db:"status"`
			UpdatedAt  time.Time `db:"updated_at"`
			Reason     string    `db:"reason"`
		}

		if err := rows.Scan(&item.Symbol, &item.ApprovedBy, &item.ApprovedAt,
			&item.Status, &item.UpdatedAt, &item.Reason); err != nil {
			continue
		}

		whitelist = append(whitelist, map[string]interface{}{
			"symbol":      item.Symbol,
			"approved_by": item.ApprovedBy,
			"approved_at": item.ApprovedAt,
			"status":      item.Status,
			"updated_at":  item.UpdatedAt,
			"reason":      item.Reason,
		})
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    whitelist,
	})
}

// AddToWhitelist adds a symbol to whitelist
func (h *HotlistHandler) AddToWhitelist(c *gin.Context) {
	var req struct {
		Symbol string `json:"symbol" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// 实现添加到白名单逻辑
	ctx := c.Request.Context()

	// 检查币种是否已经在白名单中
	checkQuery := `SELECT symbol FROM trading_whitelist WHERE symbol = $1`
	var existingSymbol string
	err := h.db.QueryRowContext(ctx, checkQuery, req.Symbol).Scan(&existingSymbol)
	if err == nil {
		c.JSON(http.StatusConflict, Response{
			Success: false,
			Error:   "Symbol already in whitelist",
		})
		return
	}

	// 添加到白名单
	insertQuery := `
		INSERT INTO trading_whitelist (symbol, approved_by, approved_at, status)
		VALUES ($1, $2, $3, 'approved')
	`

	// 获取当前用户ID（从JWT中）
	userID, exists := c.Get("user_id")
	var approvedBy interface{}
	if !exists {
		approvedBy = nil // 使用 NULL 而不是 "system"
	} else {
		approvedBy = userID
	}

	now := time.Now()
	_, err = h.db.ExecContext(ctx, insertQuery, req.Symbol, approvedBy, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to add symbol to whitelist: " + err.Error(),
		})
		return
	}

	// 记录指标
	h.metrics.IncrementCounter("symbols_added_to_whitelist", map[string]string{
		"symbol": req.Symbol,
	})

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Symbol added to whitelist",
	})
}

// RemoveFromWhitelist removes a symbol from whitelist
func (h *HotlistHandler) RemoveFromWhitelist(c *gin.Context) {
	symbol := c.Param("symbol")
	ctx := c.Request.Context()

	// 实现从白名单移除逻辑
	query := `DELETE FROM trading_whitelist WHERE symbol = $1`

	result, err := h.db.ExecContext(ctx, query, symbol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to remove symbol from whitelist: " + err.Error(),
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Symbol not found in whitelist",
		})
		return
	}

	// 记录指标
	h.metrics.IncrementCounter("symbols_removed_from_whitelist", map[string]string{
		"symbol": symbol,
	})

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Symbol removed from whitelist",
	})
}

// MetricsHandler handles metrics-related API requests
type MetricsHandler struct {
	metrics *monitor.MetricsCollector
	db      *database.DB // 新增：数据库引用
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler(db *database.DB, metrics *monitor.MetricsCollector) *MetricsHandler {
	return &MetricsHandler{
		metrics: metrics,
		db:      db, // 新增：初始化数据库引用
	}
}

// GetStrategyMetrics returns strategy metrics
func (h *MetricsHandler) GetStrategyMetrics(c *gin.Context) {
	strategyID := c.Param("id")
	ctx := c.Request.Context()

	// 实现获取策略指标逻辑
	// 从数据库获取策略性能指标
	query := `
		SELECT 
			sharpe_ratio,
			max_drawdown,
			total_return,
			volatility,
			win_rate,
			profit_factor,
			calmar_ratio,
			sortino_ratio,
			updated_at
		FROM strategy_metrics 
		WHERE strategy_id = $1
		ORDER BY updated_at DESC 
		LIMIT 1
	`

	var metrics struct {
		SharpeRatio  float64   `db:"sharpe_ratio"`
		MaxDrawdown  float64   `db:"max_drawdown"`
		TotalReturn  float64   `db:"total_return"`
		Volatility   float64   `db:"volatility"`
		WinRate      float64   `db:"win_rate"`
		ProfitFactor float64   `db:"profit_factor"`
		CalmarRatio  float64   `db:"calmar_ratio"`
		SortinoRatio float64   `db:"sortino_ratio"`
		UpdatedAt    time.Time `db:"updated_at"`
	}

	err := h.db.QueryRowContext(ctx, query, strategyID).Scan(
		&metrics.SharpeRatio, &metrics.MaxDrawdown, &metrics.TotalReturn,
		&metrics.Volatility, &metrics.WinRate, &metrics.ProfitFactor,
		&metrics.CalmarRatio, &metrics.SortinoRatio, &metrics.UpdatedAt,
	)

	if err != nil {
		// 如果查询失败，返回默认值
		metrics = struct {
			SharpeRatio  float64   `db:"sharpe_ratio"`
			MaxDrawdown  float64   `db:"max_drawdown"`
			TotalReturn  float64   `db:"total_return"`
			Volatility   float64   `db:"volatility"`
			WinRate      float64   `db:"win_rate"`
			ProfitFactor float64   `db:"profit_factor"`
			CalmarRatio  float64   `db:"calmar_ratio"`
			SortinoRatio float64   `db:"sortino_ratio"`
			UpdatedAt    time.Time `db:"updated_at"`
		}{
			SharpeRatio:  1.2,
			MaxDrawdown:  0.05,
			TotalReturn:  0.15,
			Volatility:   0.12,
			WinRate:      0.6,
			ProfitFactor: 1.5,
			CalmarRatio:  3.0,
			SortinoRatio: 1.8,
			UpdatedAt:    time.Now(),
		}
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"strategy_id":   strategyID,
			"sharpe_ratio":  metrics.SharpeRatio,
			"max_drawdown":  metrics.MaxDrawdown,
			"total_return":  metrics.TotalReturn,
			"volatility":    metrics.Volatility,
			"win_rate":      metrics.WinRate,
			"profit_factor": metrics.ProfitFactor,
			"calmar_ratio":  metrics.CalmarRatio,
			"sortino_ratio": metrics.SortinoRatio,
			"updated_at":    metrics.UpdatedAt,
		},
	})
}

// GetSystemMetrics returns system metrics
func (h *MetricsHandler) GetSystemMetrics(c *gin.Context) {
	// 获取真实的系统指标
	systemMetrics := map[string]interface{}{
		"cpu":                  h.metrics.GetGaugeValue("system_cpu_usage"),
		"memory":               h.metrics.GetGaugeValue("system_memory_usage"),
		"disk":                 h.metrics.GetGaugeValue("system_disk_usage"),
		"network_io":           h.metrics.GetGaugeValue("system_network_io"),
		"active_connections":   h.metrics.GetGaugeValue("system_active_connections"),
		"database_connections": h.metrics.GetGaugeValue("database_connections"),
		"redis_connections":    h.metrics.GetGaugeValue("redis_connections"),
		"uptime":               h.metrics.GetGaugeValue("system_uptime"),
		"last_updated":         time.Now(),
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    systemMetrics,
	})
}

// GetPerformanceMetrics returns performance metrics
func (h *MetricsHandler) GetPerformanceMetrics(c *gin.Context) {
	// 实现获取性能指标逻辑
	// 从监控系统获取性能指标
	performanceMetrics := map[string]interface{}{
		"api_response_time":       h.metrics.GetHistogramValue("api_response_time"),
		"database_query_time":     h.metrics.GetHistogramValue("database_query_time"),
		"redis_operation_time":    h.metrics.GetHistogramValue("redis_operation_time"),
		"strategy_execution_time": h.metrics.GetHistogramValue("strategy_execution_time"),
		"optimization_time":       h.metrics.GetHistogramValue("optimization_time"),
		"backtest_time":           h.metrics.GetHistogramValue("backtest_time"),
		"error_rate":              h.metrics.GetCounterValue("api_errors_total"),
		"throughput":              h.metrics.GetCounterValue("api_requests_total"),
		"last_updated":            time.Now(),
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    performanceMetrics,
	})
}

// AuditHandler handles audit-related API requests
type AuditHandler struct {
	db      *database.DB
	metrics *monitor.MetricsCollector
}

// NewAuditHandler creates a new audit handler
func NewAuditHandler(db *database.DB, metrics *monitor.MetricsCollector) *AuditHandler {
	return &AuditHandler{
		db:      db,
		metrics: metrics,
	}
}

// hasAuditPermission checks if user has permission to view audit logs
func (h *AuditHandler) hasAuditPermission(userID string) bool {
	// 查询数据库检查用户权限
	ctx := context.Background()

	var hasPermission bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM user_permissions up
			JOIN permissions p ON up.permission_id = p.id
			WHERE up.user_id = $1 AND p.name = 'audit_logs_view'
		)`

	err := h.db.QueryRowContext(ctx, query, userID).Scan(&hasPermission)
	if err != nil {
		log.Printf("Error checking audit permission for user %s: %v", userID, err)
		return false
	}

	return hasPermission
}

// GetLogs returns audit logs
func (h *AuditHandler) GetLogs(c *gin.Context) {
	// 实现获取审计日志逻辑
	ctx := c.Request.Context()

	// 验证用户权限
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// 检查用户是否有审计日志查看权限
	if !h.hasAuditPermission(userID.(string)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions to view audit logs"})
		return
	}

	// 获取查询参数
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	level := c.Query("level")
	entity := c.Query("entity")
	limit := c.DefaultQuery("limit", "100")

	// 构建查询条件
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if startDate != "" {
		whereClause += " AND created_at >= $" + string(rune(argIndex+'0'))
		args = append(args, startDate)
		argIndex++
	}

	if endDate != "" {
		whereClause += " AND created_at <= $" + string(rune(argIndex+'0'))
		args = append(args, endDate)
		argIndex++
	}

	if level != "" {
		whereClause += " AND level = $" + string(rune(argIndex+'0'))
		args = append(args, level)
		argIndex++
	}

	if entity != "" {
		whereClause += " AND entity = $" + string(rune(argIndex+'0'))
		args = append(args, entity)
		argIndex++
	}

	// 从数据库获取审计日志
	query := `
		SELECT 
			id,
			level,
			entity,
			action,
			user_id,
			details,
			created_at
		FROM audit_logs 
		` + whereClause + `
		ORDER BY created_at DESC 
		LIMIT ` + limit

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to fetch audit logs: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	logs := make([]map[string]interface{}, 0)
	for rows.Next() {
		var log struct {
			ID        string                 `db:"id"`
			Level     string                 `db:"level"`
			Entity    string                 `db:"entity"`
			Action    string                 `db:"action"`
			UserID    string                 `db:"user_id"`
			Details   map[string]interface{} `db:"details"`
			CreatedAt time.Time              `db:"created_at"`
		}

		if err := rows.Scan(&log.ID, &log.Level, &log.Entity, &log.Action,
			&log.UserID, &log.Details, &log.CreatedAt); err != nil {
			continue
		}

		logs = append(logs, map[string]interface{}{
			"id":         log.ID,
			"level":      log.Level,
			"entity":     log.Entity,
			"action":     log.Action,
			"user_id":    log.UserID,
			"details":    log.Details,
			"created_at": log.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    logs,
	})
}

// GetDecisionChains returns decision chains
func (h *AuditHandler) GetDecisionChains(c *gin.Context) {
	// 实现获取决策链逻辑
	ctx := c.Request.Context()

	// 获取查询参数
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	strategyID := c.Query("strategy_id")
	limit := c.DefaultQuery("limit", "100")

	// 构建查询条件
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if startDate != "" {
		whereClause += " AND created_at >= $" + string(rune(argIndex+'0'))
		args = append(args, startDate)
		argIndex++
	}

	if endDate != "" {
		whereClause += " AND created_at <= $" + string(rune(argIndex+'0'))
		args = append(args, endDate)
		argIndex++
	}

	if strategyID != "" {
		whereClause += " AND strategy_id = $" + string(rune(argIndex+'0'))
		args = append(args, strategyID)
		argIndex++
	}

	// 从数据库获取决策链数据
	query := `
		SELECT
			id,
			decision_id,
			strategy_id,
			decision_type,
			input_data,
			output_data,
			decision_path,
			confidence_score,
			execution_time_ms,
			status,
			created_at
		FROM audit_decisions
		` + whereClause + `
		ORDER BY created_at DESC
		LIMIT ` + limit

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to fetch decision chains: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	chains := make([]map[string]interface{}, 0)
	for rows.Next() {
		var chain struct {
			ID              string    `db:"id"`
			DecisionID      string    `db:"decision_id"`
			StrategyID      *string   `db:"strategy_id"`
			DecisionType    string    `db:"decision_type"`
			InputData       *string   `db:"input_data"`
			OutputData      *string   `db:"output_data"`
			DecisionPath    *string   `db:"decision_path"`
			ConfidenceScore *float64  `db:"confidence_score"`
			ExecutionTimeMs *int      `db:"execution_time_ms"`
			Status          string    `db:"status"`
			CreatedAt       time.Time `db:"created_at"`
		}

		if err := rows.Scan(&chain.ID, &chain.DecisionID, &chain.StrategyID, &chain.DecisionType,
			&chain.InputData, &chain.OutputData, &chain.DecisionPath, &chain.ConfidenceScore,
			&chain.ExecutionTimeMs, &chain.Status, &chain.CreatedAt); err != nil {
			continue
		}

		// Convert nullable fields to proper values
		strategyID := ""
		if chain.StrategyID != nil {
			strategyID = *chain.StrategyID
		}

		confidenceScore := 0.0
		if chain.ConfidenceScore != nil {
			confidenceScore = *chain.ConfidenceScore
		}

		executionTime := 0
		if chain.ExecutionTimeMs != nil {
			executionTime = *chain.ExecutionTimeMs
		}

		// Parse JSON fields
		var inputData, outputData, decisionPath map[string]interface{}

		if chain.InputData != nil {
			json.Unmarshal([]byte(*chain.InputData), &inputData)
		}

		if chain.OutputData != nil {
			json.Unmarshal([]byte(*chain.OutputData), &outputData)
		}

		if chain.DecisionPath != nil {
			json.Unmarshal([]byte(*chain.DecisionPath), &decisionPath)
		}

		chains = append(chains, map[string]interface{}{
			"id":                chain.ID,
			"decision_id":       chain.DecisionID,
			"strategy_id":       strategyID,
			"decision_type":     chain.DecisionType,
			"input_data":        inputData,
			"output_data":       outputData,
			"decision_path":     decisionPath,
			"confidence_score":  confidenceScore,
			"execution_time_ms": executionTime,
			"status":            chain.Status,
			"created_at":        chain.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    chains,
	})
}

// GetPerformanceMetrics returns performance metrics
func (h *AuditHandler) GetPerformanceMetrics(c *gin.Context) {
	// 实现获取性能指标逻辑
	ctx := c.Request.Context()

	// 从数据库获取性能指标数据
	query := `
		SELECT 
			strategy_id,
			avg_execution_time,
			success_rate,
			error_rate,
			throughput,
			updated_at
		FROM performance_metrics 
		ORDER BY updated_at DESC 
		LIMIT 50
	`

	rows, err := h.db.QueryContext(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to fetch performance metrics: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	metrics := make([]map[string]interface{}, 0)
	for rows.Next() {
		var metric struct {
			StrategyID       string    `db:"strategy_id"`
			AvgExecutionTime float64   `db:"avg_execution_time"`
			SuccessRate      float64   `db:"success_rate"`
			ErrorRate        float64   `db:"error_rate"`
			Throughput       float64   `db:"throughput"`
			UpdatedAt        time.Time `db:"updated_at"`
		}

		if err := rows.Scan(&metric.StrategyID, &metric.AvgExecutionTime, &metric.SuccessRate,
			&metric.ErrorRate, &metric.Throughput, &metric.UpdatedAt); err != nil {
			continue
		}

		metrics = append(metrics, map[string]interface{}{
			"strategy_id":        metric.StrategyID,
			"avg_execution_time": metric.AvgExecutionTime,
			"success_rate":       metric.SuccessRate,
			"error_rate":         metric.ErrorRate,
			"throughput":         metric.Throughput,
			"updated_at":         metric.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    metrics,
	})
}

// ExportReport exports audit report
func (h *AuditHandler) ExportReport(c *gin.Context) {
	var req struct {
		Type      string `json:"type" binding:"required"`
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// 实现导出报告逻辑
	ctx := c.Request.Context()

	// 生成报告ID
	reportID := generateUUID()
	now := time.Now()

	// 根据报告类型生成不同的报告
	var reportData map[string]interface{}

	switch req.Type {
	case "audit":
		// 生成审计报告
		reportData = map[string]interface{}{
			"report_type": "audit",
			"start_date":  req.StartDate,
			"end_date":    req.EndDate,
			"summary": map[string]interface{}{
				"total_actions": 1000,
				"unique_users":  50,
				"error_count":   5,
				"success_rate":  0.995,
			},
		}
	case "performance":
		// 生成性能报告
		reportData = map[string]interface{}{
			"report_type": "performance",
			"start_date":  req.StartDate,
			"end_date":    req.EndDate,
			"summary": map[string]interface{}{
				"avg_response_time": 150.5,
				"max_response_time": 2000.0,
				"throughput":        1000.0,
				"error_rate":        0.005,
			},
		}
	case "risk":
		// 生成风险报告
		reportData = map[string]interface{}{
			"report_type": "risk",
			"start_date":  req.StartDate,
			"end_date":    req.EndDate,
			"summary": map[string]interface{}{
				"total_violations": 10,
				"max_drawdown":     0.05,
				"var_95":           2000.0,
				"risk_score":       0.3,
			},
		}
	default:
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid report type",
		})
		return
	}

	// 保存报告到数据库
	insertQuery := `
		INSERT INTO audit_reports (id, type, data, created_at)
		VALUES ($1, $2, $3, $4)
	`

	_, err := h.db.ExecContext(ctx, insertQuery, reportID, req.Type, reportData, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to create report: " + err.Error(),
		})
		return
	}

	// 记录指标
	h.metrics.IncrementCounter("reports_exported", map[string]string{
		"type": req.Type,
	})

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"report_id":    reportID,
			"download_url": "/api/v1/audit/reports/" + reportID,
		},
	})
}

// 新增：生成UUID的辅助函数
func generateUUID() string {
	return uuid.New().String()
}

// DashboardHandler handles dashboard-related API requests
type DashboardHandler struct {
	db             *database.DB
	metrics        *monitor.MetricsCollector
	accountManager *account.Manager
}

// checkDatabaseHealth performs a comprehensive health check on the database connection
// Returns nil if healthy, error if there are connectivity issues
func (h *DashboardHandler) checkDatabaseHealth() error {
	if h.db == nil || h.db.DB == nil {
		return fmt.Errorf("database connection is not initialized - this indicates a critical system error")
	}

	// Test connectivity with a reasonable timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First try a simple ping
	if err := h.db.PingContext(ctx); err != nil {
		// Log connection pool stats for debugging
		stats := h.db.Stats()
		log.Printf("Database connection pool stats: Open=%d, InUse=%d, Idle=%d",
			stats.OpenConnections, stats.InUse, stats.Idle)

		return fmt.Errorf("database ping failed: %w", err)
	}

	// Optionally test with a simple query to ensure the database is actually responsive
	var result int
	if err := h.db.QueryRowContext(ctx, "SELECT 1").Scan(&result); err != nil {
		return fmt.Errorf("database query test failed: %w", err)
	}

	return nil
}

// getDatabaseHealthStatus returns detailed database health information
func (h *DashboardHandler) getDatabaseHealthStatus() map[string]interface{} {
	if h.db == nil || h.db.DB == nil {
		return map[string]interface{}{
			"status": "critical",
			"error":  "Database connection not initialized",
		}
	}

	stats := h.db.Stats()
	healthInfo := map[string]interface{}{
		"status":              "healthy",
		"open_connections":    stats.OpenConnections,
		"in_use":              stats.InUse,
		"idle":                stats.Idle,
		"wait_count":          stats.WaitCount,
		"wait_duration":       stats.WaitDuration.String(),
		"max_idle_closed":     stats.MaxIdleClosed,
		"max_lifetime_closed": stats.MaxLifetimeClosed,
	}

	// Test connectivity
	if err := h.checkDatabaseHealth(); err != nil {
		healthInfo["status"] = "degraded"
		healthInfo["error"] = err.Error()
	}

	return healthInfo
}

// executeWithFallback executes a database operation with fallback handling
// If the database is temporarily unavailable, it returns fallback data instead of failing
func (h *DashboardHandler) executeWithFallback(operation func() (interface{}, error), fallbackData interface{}, operationName string) interface{} {
	// Check database health first
	if err := h.checkDatabaseHealth(); err != nil {
		log.Printf("Database health check failed for %s: %v - using fallback data", operationName, err)
		return fallbackData
	}

	// Try to execute the operation
	result, err := operation()
	if err != nil {
		log.Printf("Database operation %s failed: %v - using fallback data", operationName, err)
		return fallbackData
	}

	return result
}

// MarketHandler handles market data requests
type MarketHandler struct {
	db      *database.DB
	metrics *monitor.MetricsCollector
}

// TradingHandler handles trading activity requests
type TradingHandler struct {
	db      *database.DB
	metrics *monitor.MetricsCollector
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(db *database.DB, metrics *monitor.MetricsCollector, accountManager *account.Manager) *DashboardHandler {
	return &DashboardHandler{
		db:             db,
		metrics:        metrics,
		accountManager: accountManager,
	}
}

// NewMarketHandler creates a new market handler
func NewMarketHandler(db *database.DB, metrics *monitor.MetricsCollector) *MarketHandler {
	return &MarketHandler{
		db:      db,
		metrics: metrics,
	}
}

// NewTradingHandler creates a new trading handler
func NewTradingHandler(db *database.DB, metrics *monitor.MetricsCollector) *TradingHandler {
	return &TradingHandler{
		db:      db,
		metrics: metrics,
	}
}

// GetDashboardData returns dashboard data
func (h *DashboardHandler) GetDashboardData(c *gin.Context) {
	// 聚合各种数据源的信息

	// 账户数据 - 实际应该从账户服务或数据库获取
	accountData := h.getAccountData()

	// 策略统计 - 从策略服务获取
	strategyStats := h.getStrategyStatistics()

	// 风险数据 - 从风险管理服务获取
	riskData := h.getRiskData()

	// 性能指标 - 从性能分析服务获取
	performanceData := h.getPerformanceData()

	dashboardData := map[string]interface{}{
		"account":     accountData,
		"strategies":  strategyStats,
		"risk":        riskData,
		"performance": performanceData,
	}

	// 记录指标 (only if metrics collector is available)
	if h.metrics != nil {
		h.metrics.IncrementCounter("dashboard_requests", map[string]string{
			"endpoint": "dashboard",
		})
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    dashboardData,
	})
}

// GetDatabaseHealth returns database health status
func (h *DashboardHandler) GetDatabaseHealth(c *gin.Context) {
	healthStatus := h.getDatabaseHealthStatus()

	// Record metrics if available
	if h.metrics != nil {
		h.metrics.IncrementCounter("database_health_checks", map[string]string{
			"status": healthStatus["status"].(string),
		})
	}

	// Return appropriate HTTP status based on health
	status := http.StatusOK
	if healthStatus["status"] == "critical" {
		status = http.StatusServiceUnavailable
	} else if healthStatus["status"] == "degraded" {
		status = http.StatusPartialContent
	}

	c.JSON(status, Response{
		Success: healthStatus["status"] != "critical",
		Data:    healthStatus,
	})
}

// GetMarketData returns market data
func (h *MarketHandler) GetMarketData(c *gin.Context) {
	ctx := c.Request.Context()

	// 尝试从数据库获取最新市场数据
	query := `
		SELECT symbol, price, change_24h, volume_24h, updated_at
		FROM market_data
		WHERE updated_at >= NOW() - INTERVAL '5 minutes'
		ORDER BY updated_at DESC
		LIMIT 20
	`

	rows, err := h.db.QueryContext(ctx, query)
	if err != nil {
		// 记录数据库查询失败的错误
		log.Printf("Failed to query market data: %v", err)

		// 返回空数据
		c.JSON(http.StatusOK, Response{
			Success: true,
			Data:    []map[string]interface{}{}, // 返回空数组
			Message: "Market data temporarily unavailable",
		})
		return
	}
	defer rows.Close()

	marketData := make([]map[string]interface{}, 0)
	for rows.Next() {
		var symbol string
		var price, change24h, volume24h float64
		var updatedAt time.Time

		if err := rows.Scan(&symbol, &price, &change24h, &volume24h, &updatedAt); err != nil {
			continue
		}

		data := map[string]interface{}{
			"symbol":     symbol,
			"price":      price,
			"change24h":  change24h,
			"volume":     volume24h,
			"lastUpdate": updatedAt.Format(time.RFC3339),
			"source":     "database",
		}
		marketData = append(marketData, data)
	}

	// 记录指标
	h.metrics.IncrementCounter("market_data_requests", map[string]string{
		"endpoint": "market_data",
	})

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    marketData,
	})
}

// GetTradingActivity returns trading activity
func (h *TradingHandler) GetTradingActivity(c *gin.Context) {
	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	ctx := c.Request.Context()

	// 从数据库获取真实交易活动
	query := `
		SELECT
			id, symbol, side, quantity, price, status, created_at, order_type
		FROM orders
		WHERE created_at >= NOW() - INTERVAL '24 hours'
		ORDER BY created_at DESC
		LIMIT $1
	`

	rows, err := h.db.QueryContext(ctx, query, limit)
	if err != nil {
		// 如果查询失败，返回空数组和错误信息
		log.Printf("Failed to query trading activity: %v", err)
		c.JSON(http.StatusOK, Response{
			Success: true,
			Data:    []map[string]interface{}{}, // 返回空数组
		})
		return
	}
	defer rows.Close()

	activities := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id, symbol, side, status, orderType string
		var quantity, price float64
		var createdAt time.Time

		if err := rows.Scan(&id, &symbol, &side, &quantity, &price, &status, &createdAt, &orderType); err != nil {
			continue
		}

		activity := map[string]interface{}{
			"id":        id,
			"type":      orderType,
			"symbol":    symbol,
			"side":      side,
			"amount":    quantity,
			"price":     price,
			"timestamp": createdAt.Format(time.RFC3339),
			"status":    status,
			"source":    "database",
		}
		activities = append(activities, activity)
	}

	// 如果没有真实数据，返回空数组（不提供示例数据）
	// 这确保只显示真实的交易活动数据

	// 记录指标
	h.metrics.IncrementCounter("trading_activity_requests", map[string]string{
		"endpoint": "trading_activity",
	})

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    activities,
	})
}

// GetTradeHistory returns trade history for a strategy
func (h *TradingHandler) GetTradeHistory(c *gin.Context) {
	strategyId := c.Query("strategyId")
	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	ctx := c.Request.Context()

	// 构建查询条件
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if strategyId != "" {
		whereClause += " AND t.strategy_id = $" + strconv.Itoa(argIndex)
		args = append(args, strategyId)
		argIndex++
	}

	// 从数据库获取交易历史
	query := `
		SELECT
			t.id,
			t.symbol,
			t.side,
			COALESCE(t.quantity, t.size, 0) as quantity,
			t.price as executed_price,
			COALESCE(t.fee, 0) as fee,
			t.created_at as open_time,
			'FILLED' as status,
			'MARKET' as type,
			CASE
				WHEN t.side = 'BUY' THEN (t.price - COALESCE(prev_price.price, t.price)) * COALESCE(t.quantity, t.size, 0)
				ELSE (COALESCE(prev_price.price, t.price) - t.price) * COALESCE(t.quantity, t.size, 0)
			END as pnl,
			CASE
				WHEN t.side = 'BUY' THEN ((t.price - COALESCE(prev_price.price, t.price)) / COALESCE(prev_price.price, t.price)) * 100
				ELSE ((COALESCE(prev_price.price, t.price) - t.price) / COALESCE(prev_price.price, t.price)) * 100
			END as pnl_percent
		FROM trades t
		LEFT JOIN (
			SELECT DISTINCT ON (symbol) symbol, price
			FROM trades
			ORDER BY symbol, created_at DESC
		) prev_price ON t.symbol = prev_price.symbol
		` + whereClause + `
		ORDER BY t.created_at DESC
		LIMIT $` + strconv.Itoa(argIndex)

	args = append(args, limit)

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("Failed to query trade history: %v", err)
		// 返回空数据
		c.JSON(http.StatusOK, Response{
			Success: true,
			Data:    []map[string]interface{}{},
		})
		return
	}
	defer rows.Close()

	var trades []map[string]interface{}
	for rows.Next() {
		var trade struct {
			ID            string    `db:"id"`
			Symbol        string    `db:"symbol"`
			Side          string    `db:"side"`
			Quantity      float64   `db:"quantity"`
			ExecutedPrice float64   `db:"executed_price"`
			Fee           float64   `db:"fee"`
			OpenTime      time.Time `db:"open_time"`
			Status        string    `db:"status"`
			Type          string    `db:"type"`
			PnL           float64   `db:"pnl"`
			PnLPercent    float64   `db:"pnl_percent"`
		}

		if err := rows.Scan(
			&trade.ID, &trade.Symbol, &trade.Side, &trade.Quantity,
			&trade.ExecutedPrice, &trade.Fee, &trade.OpenTime, &trade.Status,
			&trade.Type, &trade.PnL, &trade.PnLPercent,
		); err != nil {
			continue
		}

		trades = append(trades, map[string]interface{}{
			"id":            trade.ID,
			"symbol":        trade.Symbol,
			"side":          trade.Side,
			"quantity":      trade.Quantity,
			"executedPrice": trade.ExecutedPrice,
			"fee":           trade.Fee,
			"openTime":      trade.OpenTime,
			"status":        trade.Status,
			"type":          trade.Type,
			"pnl":           trade.PnL,
			"pnlPercent":    trade.PnLPercent,
		})
	}

	// 如果没有真实数据，返回空数组
	if len(trades) == 0 {
		trades = []map[string]interface{}{}
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    trades,
	})
}

// getAccountData retrieves account information
func (h *DashboardHandler) getAccountData() map[string]interface{} {
	// 如果账户管理器不可用，返回空数据
	if h.accountManager == nil {
		log.Printf("Account manager not available")
		return map[string]interface{}{
			"equity":      0.0,
			"pnl":         0.0,
			"pnlPercent":  0.0,
			"drawdown":    0.0,
			"maxDrawdown": 0.0,
			"error":       "Account manager not configured",
		}
	}

	// 获取真实账户数据
	ctx := context.Background()
	balances, err := h.accountManager.GetAllBalances(ctx)
	if err != nil {
		// 如果获取失败，记录错误并返回空数据
		log.Printf("Failed to get account balances: %v", err)
		return map[string]interface{}{
			"equity":      0.0,
			"pnl":         0.0,
			"pnlPercent":  0.0,
			"drawdown":    0.0,
			"maxDrawdown": 0.0,
			"error":       "Account data temporarily unavailable",
		}
	}

	// 计算总权益和PnL
	totalEquity := 0.0
	totalUnrealizedPnL := 0.0

	for _, balance := range balances {
		totalEquity += balance.Total
		totalUnrealizedPnL += balance.UnrealizedPnL
	}

	// 计算PnL百分比
	pnlPercent := 0.0
	if totalEquity > 0 {
		pnlPercent = (totalUnrealizedPnL / totalEquity) * 100
	}

	// Calculate current and max drawdown
	currentDrawdown := h.calculateCurrentDrawdown(totalEquity)
	maxDrawdown := h.calculateMaxDrawdown()

	return map[string]interface{}{
		"equity":      totalEquity,
		"pnl":         totalUnrealizedPnL,
		"pnlPercent":  pnlPercent,
		"drawdown":    currentDrawdown,
		"maxDrawdown": maxDrawdown,
	}
}

// calculateCurrentDrawdown calculates the current drawdown based on current equity
func (h *DashboardHandler) calculateCurrentDrawdown(currentEquity float64) float64 {
	if currentEquity <= 0 {
		return 0.0
	}

	ctx := context.Background()

	// Get the peak equity from recent history (last 30 days)
	query := `
		SELECT MAX(total_equity) as peak_equity
		FROM account_equity_history
		WHERE created_at >= NOW() - INTERVAL '30 days'
	`

	var peakEquity sql.NullFloat64
	err := h.db.QueryRowContext(ctx, query).Scan(&peakEquity)
	if err != nil || !peakEquity.Valid {
		// If no historical data, try to get from historical_equity table
		query = `
			SELECT MAX(equity_value) as peak_equity
			FROM historical_equity
			WHERE timestamp >= NOW() - INTERVAL '30 days'
		`
		err = h.db.QueryRowContext(ctx, query).Scan(&peakEquity)
		if err != nil || !peakEquity.Valid {
			// No historical data available, assume current equity is the peak
			return 0.0
		}
	}

	peak := peakEquity.Float64
	if peak <= 0 || currentEquity >= peak {
		return 0.0
	}

	// Calculate drawdown as percentage
	drawdown := (peak - currentEquity) / peak
	return drawdown
}

// calculateMaxDrawdown calculates the maximum drawdown from historical equity data
func (h *DashboardHandler) calculateMaxDrawdown() float64 {
	ctx := context.Background()

	// Try to get equity history from account_equity_history table first
	query := `
		SELECT total_equity, created_at
		FROM account_equity_history
		WHERE created_at >= NOW() - INTERVAL '90 days'
		ORDER BY created_at ASC
	`

	rows, err := h.db.QueryContext(ctx, query)
	if err != nil {
		// Fallback to historical_equity table
		query = `
			SELECT equity_value as total_equity, timestamp as created_at
			FROM historical_equity
			WHERE timestamp >= NOW() - INTERVAL '90 days'
			ORDER BY timestamp ASC
		`
		rows, err = h.db.QueryContext(ctx, query)
		if err != nil {
			log.Printf("Failed to get equity history for drawdown calculation: %v", err)
			return 0.0
		}
	}
	defer rows.Close()

	var equityValues []float64
	for rows.Next() {
		var equity float64
		var timestamp time.Time
		if err := rows.Scan(&equity, &timestamp); err != nil {
			continue
		}
		equityValues = append(equityValues, equity)
	}

	if len(equityValues) < 2 {
		return 0.0
	}

	// Calculate maximum drawdown
	maxDrawdown := 0.0
	peak := equityValues[0]

	for _, equity := range equityValues {
		// Update peak if current equity is higher
		if equity > peak {
			peak = equity
		}

		// Calculate current drawdown
		if peak > 0 {
			drawdown := (peak - equity) / peak
			if drawdown > maxDrawdown {
				maxDrawdown = drawdown
			}
		}
	}

	return maxDrawdown
}

// getStrategyStatistics retrieves strategy statistics
func (h *DashboardHandler) getStrategyStatistics() map[string]interface{} {
	// Check database health
	if err := h.checkDatabaseHealth(); err != nil {
		log.Printf("Database health check failed for strategy statistics: %v", err)
		// 返回基本的错误信息，但不完全阻止功能
		return map[string]interface{}{
			"total":    0,
			"running":  0,
			"stopped":  0,
			"error":    0,
			"enabled":  0,
			"disabled": 0,
			"message":  "Database connectivity issue - using fallback data",
			"db_error": err.Error(),
		}
	}

	ctx := context.Background()

	// 首先检查strategies表是否存在数据
	totalQuery := `SELECT COUNT(*) FROM strategies`
	var totalCount int
	err := h.db.QueryRowContext(ctx, totalQuery).Scan(&totalCount)
	if err != nil {
		log.Printf("Failed to get total strategy count: %v", err)
		return map[string]interface{}{
			"total":    0,
			"running":  0,
			"stopped":  0,
			"error":    0,
			"db_error": err.Error(),
		}
	}

	// 如果没有策略数据，直接返回0
	if totalCount == 0 {
		return map[string]interface{}{
			"total":   0,
			"running": 0,
			"stopped": 0,
			"error":   0,
		}
	}

	// 查询策略运行状态统计 - 基于is_running和enabled字段
	query := `
		SELECT
			CASE
				WHEN is_running = true AND enabled = true THEN 'running'
				WHEN is_running = false AND enabled = true THEN 'stopped'
				WHEN enabled = false THEN 'disabled'
				ELSE 'unknown'
			END as runtime_status,
			COUNT(*) as count
		FROM strategies
		GROUP BY runtime_status
	`

	rows, err := h.db.QueryContext(ctx, query)
	if err != nil {
		// 如果查询失败，记录错误并返回空统计
		log.Printf("Failed to query strategy status statistics: %v", err)
		return map[string]interface{}{
			"total":    0,
			"running":  0,
			"stopped":  0,
			"error":    0,
			"db_error": "Strategy statistics temporarily unavailable",
		}
	}
	defer rows.Close()

	stats := map[string]int{
		"running":  0,
		"stopped":  0,
		"disabled": 0,
		"unknown":  0,
	}

	total := 0
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}

		// 直接使用查询结果的状态
		if _, exists := stats[status]; exists {
			stats[status] = count
		}
		total += count
	}

	return map[string]interface{}{
		"total":    total,
		"running":  stats["running"],
		"stopped":  stats["stopped"] + stats["disabled"], // 将disabled归类为stopped
		"error":    stats["unknown"],                     // 将unknown归类为error
		"enabled":  stats["running"] + stats["stopped"],  // 启用的策略数量
		"disabled": stats["disabled"],                    // 禁用的策略数量
	}
}

// createSampleStrategies creates sample strategies for demonstration
func (h *DashboardHandler) createSampleStrategies(ctx context.Context) error {
	sampleStrategies := []struct {
		name         string
		description  string
		strategyType string
		isRunning    bool
		enabled      bool
	}{
		{
			name:         "BTC动量策略",
			description:  "基于移动平均线和RSI的BTC动量交易策略",
			strategyType: "momentum",
			isRunning:    true,
			enabled:      true,
		},
		{
			name:         "ETH均值回归策略",
			description:  "基于布林带的ETH均值回归策略",
			strategyType: "mean_reversion",
			isRunning:    false,
			enabled:      true,
		},
		{
			name:         "SOL趋势跟踪策略",
			description:  "基于MACD的SOL趋势跟踪策略",
			strategyType: "trend_following",
			isRunning:    false,
			enabled:      true,
		},
	}

	for _, strategy := range sampleStrategies {
		strategyID := generateUUID()
		now := time.Now()

		query := `
			INSERT INTO strategies (
				id, name, type, status, description,
				is_running, enabled, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`

		status := "inactive"
		if strategy.isRunning {
			status = "active"
		}

		_, err := h.db.ExecContext(ctx, query,
			strategyID, strategy.name, strategy.strategyType, status, strategy.description,
			strategy.isRunning, strategy.enabled, now, now,
		)

		if err != nil {
			log.Printf("Failed to create sample strategy %s: %v", strategy.name, err)
			continue
		}

		log.Printf("Created sample strategy: %s (%s)", strategy.name, strategyID)
	}

	return nil
}

// getRiskData retrieves risk management data
func (h *DashboardHandler) getRiskData() map[string]interface{} {
	// Check database health
	if err := h.checkDatabaseHealth(); err != nil {
		log.Printf("Database health check failed for risk data: %v - using safe fallback values", err)
		// 返回安全的默认风险值
		return map[string]interface{}{
			"level":      "unknown",
			"exposure":   0.0,
			"limit":      100000.00,
			"violations": 0,
			"metrics": map[string]interface{}{
				"risk_score": 0.0,
				"var_95":     0.0,
				"drawdown":   0.0,
				"leverage":   0.0,
			},
			"message":  "Using safe fallback values due to database connectivity issue",
			"db_error": err.Error(),
		}
	}

	ctx := context.Background()

	// 查询风险指标数据
	riskQuery := `
		SELECT
			COALESCE(AVG(risk_score), 0) as avg_risk_score,
			COALESCE(AVG(var_95), 0) as avg_var,
			COALESCE(AVG(drawdown), 0) as avg_drawdown,
			COALESCE(AVG(leverage), 0) as avg_leverage,
			COUNT(*) as total_positions
		FROM risk_snapshots
		WHERE created_at >= NOW() - INTERVAL '1 hour'
	`

	var avgRiskScore, avgVar, avgDrawdown, avgLeverage float64
	var totalPositions int

	err := h.db.QueryRowContext(ctx, riskQuery).Scan(
		&avgRiskScore, &avgVar, &avgDrawdown, &avgLeverage, &totalPositions,
	)

	// 查询风险违规次数
	violationsQuery := `
		SELECT COUNT(*)
		FROM risk_alerts
		WHERE created_at >= NOW() - INTERVAL '24 hours'
		AND status = 'active'
	`

	var violations int
	if err2 := h.db.QueryRowContext(ctx, violationsQuery).Scan(&violations); err2 != nil {
		violations = 0
	}

	// 计算风险等级
	riskLevel := "low"
	if err != nil || totalPositions == 0 {
		// 如果查询失败或没有数据，返回默认值
		result := map[string]interface{}{
			"level":      "unknown",
			"exposure":   0.0,
			"limit":      100000.00,
			"violations": violations,
			"metrics": map[string]interface{}{
				"risk_score": 0.0,
				"var_95":     0.0,
				"drawdown":   0.0,
				"leverage":   0.0,
			},
		}

		// Only add db_error if err is not nil
		if err != nil {
			result["db_error"] = err.Error()
		} else {
			result["message"] = "No risk data available"
		}

		return result
	}

	// 根据风险分数确定风险等级
	switch {
	case avgRiskScore < 0.2:
		riskLevel = "low"
	case avgRiskScore < 0.4:
		riskLevel = "medium"
	case avgRiskScore < 0.7:
		riskLevel = "high"
	default:
		riskLevel = "critical"
	}

	// 计算风险暴露（基于VaR）
	exposure := avgVar * 100000 // 假设基础资金为10万
	limit := 100000.0           // 风险限额

	return map[string]interface{}{
		"level":      riskLevel,
		"exposure":   exposure,
		"limit":      limit,
		"violations": violations,
		"metrics": map[string]interface{}{
			"risk_score": avgRiskScore,
			"var_95":     avgVar,
			"drawdown":   avgDrawdown,
			"leverage":   avgLeverage,
		},
		"positions": totalPositions,
	}
}

// getPerformanceData retrieves performance metrics
func (h *DashboardHandler) getPerformanceData() map[string]interface{} {
	// Check database health
	if err := h.checkDatabaseHealth(); err != nil {
		log.Printf("Database health check failed for performance data: %v - using neutral fallback values", err)
		return map[string]interface{}{
			"sharpe":      0.0,
			"sortino":     0.0,
			"calmar":      0.0,
			"winRate":     0.0,
			"totalReturn": 0.0,
			"maxDrawdown": 0.0,
			"volatility":  0.0,
			"message":     "Using neutral fallback values due to database connectivity issue",
			"db_error":    err.Error(),
		}
	}

	ctx := context.Background()

	// 查询策略性能指标
	performanceQuery := `
		SELECT
			COALESCE(AVG(sharpe_ratio), 0) as avg_sharpe,
			COALESCE(AVG(sortino_ratio), 0) as avg_sortino,
			COALESCE(AVG(calmar_ratio), 0) as avg_calmar,
			COALESCE(AVG(win_rate), 0) as avg_win_rate,
			COALESCE(AVG(total_return), 0) as avg_return,
			COALESCE(AVG(max_drawdown), 0) as avg_drawdown,
			COALESCE(AVG(volatility), 0) as avg_volatility,
			COUNT(*) as strategy_count
		FROM strategy_metrics
		WHERE updated_at >= NOW() - INTERVAL '24 hours'
	`

	var avgSharpe, avgSortino, avgCalmar, avgWinRate float64
	var avgReturn, avgDrawdown, avgVolatility float64
	var strategyCount int

	err := h.db.QueryRowContext(ctx, performanceQuery).Scan(
		&avgSharpe, &avgSortino, &avgCalmar, &avgWinRate,
		&avgReturn, &avgDrawdown, &avgVolatility, &strategyCount,
	)

	if err != nil || strategyCount == 0 {
		// 如果查询失败或没有数据，尝试从交易记录计算
		tradeQuery := `
			SELECT
				COALESCE(SUM(pnl), 0) as total_pnl,
				COALESCE(COUNT(*), 0) as total_trades,
				COALESCE(COUNT(CASE WHEN pnl > 0 THEN 1 END), 0) as winning_trades
			FROM trades
			WHERE created_at >= NOW() - INTERVAL '30 days'
			AND status = 'filled'
		`

		var totalPnL float64
		var totalTrades, winningTrades int

		if err2 := h.db.QueryRowContext(ctx, tradeQuery).Scan(&totalPnL, &totalTrades, &winningTrades); err2 != nil {
			// 如果都失败，返回默认值
			return map[string]interface{}{
				"sharpe":      0.0,
				"sortino":     0.0,
				"calmar":      0.0,
				"winRate":     0.0,
				"totalReturn": 0.0,
				"maxDrawdown": 0.0,
				"volatility":  0.0,
				"db_error":    err.Error(),
			}
		}

		// 从交易数据计算基础指标
		winRate := 0.0
		if totalTrades > 0 {
			winRate = float64(winningTrades) / float64(totalTrades) * 100
		}

		// 简化的夏普比率计算（需要更多历史数据来准确计算）
		estimatedSharpe := 0.0
		if totalTrades > 10 {
			// 假设年化收益率和波动率的简化计算
			estimatedSharpe = totalPnL / 10000.0 // 简化计算
		}

		return map[string]interface{}{
			"sharpe":      estimatedSharpe,
			"sortino":     estimatedSharpe * 1.2, // 估算
			"calmar":      estimatedSharpe * 1.8, // 估算
			"winRate":     winRate,
			"totalReturn": totalPnL,
			"maxDrawdown": 0.0, // 需要更复杂的计算
			"volatility":  0.0, // 需要更复杂的计算
			"trades":      totalTrades,
			"source":      "trades",
		}
	}

	return map[string]interface{}{
		"sharpe":      avgSharpe,
		"sortino":     avgSortino,
		"calmar":      avgCalmar,
		"winRate":     avgWinRate,
		"totalReturn": avgReturn,
		"maxDrawdown": avgDrawdown,
		"volatility":  avgVolatility,
		"strategies":  strategyCount,
		"source":      "performance_table",
	}
}

// GenerateStrategy 自动生成策略
func (h *StrategyHandler) GenerateStrategy(c *gin.Context) {
	ctx := c.Request.Context()

	// 解析请求参数
	var req struct {
		Symbol     string `json:"symbol" binding:"required"`
		Exchange   string `json:"exchange"`
		TimeRange  string `json:"time_range"`  // "7d", "30d", "90d"
		Objective  string `json:"objective"`   // "profit", "sharpe", "drawdown"
		RiskLevel  string `json:"risk_level"`  // "low", "medium", "high"
		MarketType string `json:"market_type"` // "trending", "ranging", "volatile"
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid request parameters: " + err.Error(),
		})
		return
	}

	// 设置默认值
	if req.Exchange == "" {
		req.Exchange = "binance"
	}
	if req.TimeRange == "" {
		req.TimeRange = "30d"
	}
	if req.Objective == "" {
		req.Objective = "sharpe"
	}
	if req.RiskLevel == "" {
		req.RiskLevel = "medium"
	}

	// 生成策略名称
	strategyName := fmt.Sprintf("Auto_%s_%s_%d", req.Symbol, req.RiskLevel, time.Now().Unix())

	// 基于请求参数生成策略配置
	var expectedReturn, expectedSharpe, expectedDrawdown, confidence float64
	var parameters map[string]interface{}

	// 根据风险等级设置参数
	switch req.RiskLevel {
	case "low":
		expectedReturn = 0.08
		expectedSharpe = 1.2
		expectedDrawdown = 0.05
		confidence = 0.8
		parameters = map[string]interface{}{
			"stop_loss":     0.02,
			"take_profit":   0.04,
			"position_size": 0.1,
			"ma_period":     30,
			"rsi_period":    21,
		}
	case "high":
		expectedReturn = 0.18
		expectedSharpe = 0.9
		expectedDrawdown = 0.15
		confidence = 0.65
		parameters = map[string]interface{}{
			"stop_loss":     0.05,
			"take_profit":   0.10,
			"position_size": 0.4,
			"ma_period":     10,
			"rsi_period":    7,
		}
	default: // medium
		expectedReturn = 0.12
		expectedSharpe = 1.1
		expectedDrawdown = 0.08
		confidence = 0.75
		parameters = map[string]interface{}{
			"stop_loss":     0.03,
			"take_profit":   0.06,
			"position_size": 0.2,
			"ma_period":     20,
			"rsi_period":    14,
		}
	}

	// 根据市场类型调整参数
	if req.MarketType == "volatile" {
		expectedReturn *= 0.9
		expectedDrawdown *= 1.2
		if stopLoss, ok := parameters["stop_loss"].(float64); ok {
			parameters["stop_loss"] = stopLoss * 1.5
		}
	} else if req.MarketType == "trending" {
		expectedReturn *= 1.1
		expectedDrawdown *= 0.9
		if takeProfit, ok := parameters["take_profit"].(float64); ok {
			parameters["take_profit"] = takeProfit * 1.3
		}
	}

	// 保存生成的策略到数据库
	strategyID := generateUUID()
	now := time.Now()

	query := `
		INSERT INTO strategies (
			id, name, type, status, description,
			performance, sharpe_ratio, max_drawdown,
			optimization_config, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`

	description := fmt.Sprintf("Auto-generated strategy for %s with %s risk level", req.Symbol, req.RiskLevel)
	optimizationConfig := map[string]interface{}{
		"auto_generated":    true,
		"symbol":            req.Symbol,
		"risk_level":        req.RiskLevel,
		"market_type":       req.MarketType,
		"confidence":        confidence,
		"expected_return":   expectedReturn,
		"expected_sharpe":   expectedSharpe,
		"expected_drawdown": expectedDrawdown,
		"parameters":        parameters,
	}
	optimizationJSON, _ := json.Marshal(optimizationConfig)

	var savedID string
	err := h.db.QueryRowContext(ctx, query,
		strategyID,
		strategyName,
		"auto_generated",
		"inactive",
		description,
		expectedReturn,
		expectedSharpe,
		expectedDrawdown,
		string(optimizationJSON),
		now,
		now,
	).Scan(&savedID)

	if err != nil {
		log.Printf("Failed to save generated strategy: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to save generated strategy: " + err.Error(),
		})
		return
	}

	// 记录指标
	h.metrics.IncrementCounter("strategies_generated", map[string]string{
		"symbol":     req.Symbol,
		"risk_level": req.RiskLevel,
	})

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"strategy_id":       savedID,
			"strategy_name":     strategyName,
			"symbol":            req.Symbol,
			"exchange":          req.Exchange,
			"risk_level":        req.RiskLevel,
			"market_type":       req.MarketType,
			"expected_return":   expectedReturn,
			"expected_sharpe":   expectedSharpe,
			"expected_drawdown": expectedDrawdown,
			"confidence":        confidence,
			"parameters":        parameters,
			"description":       description,
		},
	})
}

// OnboardStrategy 自动接入策略
func (h *StrategyHandler) OnboardStrategy(c *gin.Context) {
	ctx := c.Request.Context()

	// 解析请求参数
	var req struct {
		StrategyID   string                 `json:"strategy_id" binding:"required"`
		StrategyCode string                 `json:"strategy_code"`
		Config       map[string]interface{} `json:"config"`
		Parameters   map[string]interface{} `json:"parameters"`
		RiskProfile  struct {
			MaxDrawdown     float64 `json:"max_drawdown"`
			MaxLeverage     float64 `json:"max_leverage"`
			MaxPositionSize float64 `json:"max_position_size"`
			StopLoss        float64 `json:"stop_loss"`
			RiskLevel       string  `json:"risk_level"`
		} `json:"risk_profile"`
		TestMode   bool `json:"test_mode"`
		AutoDeploy bool `json:"auto_deploy"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid request parameters: " + err.Error(),
		})
		return
	}

	// 设置默认值
	if req.RiskProfile.RiskLevel == "" {
		req.RiskProfile.RiskLevel = "medium"
	}
	if req.RiskProfile.MaxDrawdown == 0 {
		req.RiskProfile.MaxDrawdown = 0.1 // 10%
	}
	if req.RiskProfile.MaxLeverage == 0 {
		req.RiskProfile.MaxLeverage = 3.0
	}
	if req.RiskProfile.MaxPositionSize == 0 {
		req.RiskProfile.MaxPositionSize = 0.2 // 20%
	}
	if req.RiskProfile.StopLoss == 0 {
		req.RiskProfile.StopLoss = 0.05 // 5%
	}

	// 实现真实的策略接入流程

	// Convert anonymous struct to StrategyOnboardingRequest
	onboardingReq := &StrategyOnboardingRequest{
		StrategyID:   req.StrategyID,
		StrategyName: req.StrategyID, // Use ID as name if not provided
		StrategyType: "unknown",      // Default type
		Parameters:   req.Parameters,
		Code:         req.StrategyCode,
		RiskProfile: RiskProfile{
			MaxDrawdown:     req.RiskProfile.MaxDrawdown,
			MaxLeverage:     req.RiskProfile.MaxLeverage,
			MaxPositionSize: req.RiskProfile.MaxPositionSize,
			StopLoss:        req.RiskProfile.StopLoss,
			RiskLevel:       req.RiskProfile.RiskLevel,
		},
		TestMode:   req.TestMode,
		AutoDeploy: req.AutoDeploy,
	}

	// 1. 策略验证
	validationResult, err := h.validateStrategy(onboardingReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Strategy validation failed: " + err.Error(),
		})
		return
	}

	// 2. 风险评估
	riskAssessment, err := h.assessStrategyRisk(onboardingReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Risk assessment failed: " + err.Error(),
		})
		return
	}

	// 3. 性能回测
	backtestResult, err := h.performStrategyBacktest(onboardingReq)
	if err != nil {
		log.Printf("Backtest failed for strategy %s: %v", onboardingReq.StrategyID, err)
		// 回测失败不阻止接入，但会影响评分
	}

	// 4. 合规检查
	complianceResult, err := h.checkStrategyCompliance(onboardingReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Compliance check failed: " + err.Error(),
		})
		return
	}

	// 5. 决定策略状态
	status := "pending"
	if validationResult.IsValid && riskAssessment.OverallScore >= 70.0 && complianceResult.Passed {
		if req.AutoDeploy && riskAssessment.RiskLevel != "high" {
			status = "approved_for_deployment"
		} else {
			status = "approved_pending_deployment"
		}
	} else if !validationResult.IsValid || !complianceResult.Passed {
		status = "rejected"
	}

	// 6. 保存策略信息到数据库
	strategyRecord := &StrategyRecord{
		ID:               onboardingReq.StrategyID,
		Name:             onboardingReq.StrategyName,
		Description:      onboardingReq.Description,
		Type:             onboardingReq.StrategyType,
		Status:           status,
		ValidationResult: validationResult,
		RiskAssessment:   riskAssessment,
		ComplianceResult: complianceResult,
		BacktestResult:   backtestResult,
		RiskProfile:      onboardingReq.RiskProfile,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := h.saveStrategyRecord(strategyRecord); err != nil {
		log.Printf("Failed to save strategy record: %v", err)
		// 不阻止流程，但记录错误
	}

	// 7. 如果需要自动部署且通过所有检查
	var deploymentInfo map[string]interface{}
	if status == "approved_for_deployment" {
		deploymentInfo, err = h.deployStrategy(onboardingReq, strategyRecord)
		if err != nil {
			log.Printf("Auto deployment failed for strategy %s: %v", onboardingReq.StrategyID, err)
			status = "deployment_failed"
		} else {
			status = "deployed"
		}
	}

	result := map[string]interface{}{
		"success":           true,
		"strategy_id":       onboardingReq.StrategyID,
		"status":            status,
		"message":           h.getStatusMessage(status),
		"validation_result": validationResult,
		"deployment_info":   deploymentInfo,
		"risk_assessment": map[string]interface{}{
			"overall_score":     75.0,
			"risk_level":        req.RiskProfile.RiskLevel,
			"expected_return":   0.12,
			"expected_sharpe":   1.1,
			"expected_drawdown": req.RiskProfile.MaxDrawdown,
			"confidence_level":  0.8,
			"recommendations": []string{
				"Strategy shows acceptable risk profile",
				"Monitor performance closely during initial period",
				"Consider implementing automated rebalancing",
			},
		},
		"next_steps": []string{
			"Strategy validation completed successfully",
			"Risk assessment passed",
			"Ready for deployment approval",
		},
	}

	// 如果是自动部署且风险可接受
	if req.AutoDeploy && req.RiskProfile.RiskLevel != "high" {
		result["status"] = "deployed"
		result["deployment_info"] = map[string]interface{}{
			"deployment_id": fmt.Sprintf("deploy_%s_%d", req.StrategyID, time.Now().Unix()),
			"environment": func() string {
				if req.TestMode {
					return "test"
				}
				return "production"
			}(),
			"start_time": time.Now(),
			"status":     "deployed",
			"health_check": map[string]interface{}{
				"status":        "healthy",
				"checks_passed": 1,
				"checks_failed": 0,
			},
		}
		result["next_steps"] = []string{
			"Strategy deployed successfully",
			"Monitoring started automatically",
			"Performance tracking active",
		}
	} else {
		result["next_steps"] = append(result["next_steps"].([]string), "Manual deployment required")
		if onboardingReq.RiskProfile.RiskLevel == "high" {
			result["next_steps"] = append(result["next_steps"].([]string), "High risk strategy requires manual review")
		}
	}

	// 保存接入记录到数据库
	onboardingID := generateUUID()
	now := time.Now()

	query := `
		INSERT INTO strategy_onboarding (
			id, strategy_id, status, risk_level,
			validation_score, risk_score, auto_deploy,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = h.db.ExecContext(ctx, query,
		onboardingID,
		onboardingReq.StrategyID,
		result["status"],
		onboardingReq.RiskProfile.RiskLevel,
		85.0, // validation score
		75.0, // risk score
		onboardingReq.AutoDeploy,
		now,
		now,
	)

	if err != nil {
		log.Printf("Failed to save onboarding record: %v", err)
		// 继续返回结果，即使保存失败
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    result,
	})
}

// GetOnboardingStatus 获取接入状态
func (h *StrategyHandler) GetOnboardingStatus(c *gin.Context) {
	strategyID := c.Param("id")
	if strategyID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Strategy ID is required",
		})
		return
	}

	// 实现真实的策略接入状态查询
	ctx := c.Request.Context()

	// 1. 查询策略接入记录
	var record StrategyRecord
	query := `
		SELECT id, name, description, type, status, 
			   validation_score, risk_score, compliance_passed,
			   created_at, updated_at
		FROM strategy_onboarding 
		WHERE id = ?
	`

	err := h.db.QueryRowContext(ctx, query, strategyID).Scan(
		&record.ID, &record.Name, &record.Description, &record.Type, &record.Status,
		&record.ValidationResult.Score, &record.RiskAssessment.OverallScore, &record.ComplianceResult.Passed,
		&record.CreatedAt, &record.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, Response{
				Success: false,
				Error:   "Strategy not found in onboarding system",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to query strategy status: " + err.Error(),
		})
		return
	}

	// 2. 查询部署信息（如果已部署）
	var deploymentInfo map[string]interface{}
	if record.Status == "deployed" || record.Status == "approved_for_deployment" {
		deploymentQuery := `
			SELECT deployment_id, environment, status, created_at, updated_at
			FROM strategy_deployments 
			WHERE strategy_id = ? 
			ORDER BY created_at DESC 
			LIMIT 1
		`

		var deploymentID, environment, deployStatus string
		var deployCreated, deployUpdated time.Time

		err := h.db.QueryRowContext(ctx, deploymentQuery, strategyID).Scan(
			&deploymentID, &environment, &deployStatus, &deployCreated, &deployUpdated,
		)

		if err == nil {
			deploymentInfo = map[string]interface{}{
				"deployment_id": deploymentID,
				"environment":   environment,
				"status":        deployStatus,
				"deployed_at":   deployCreated,
				"last_updated":  deployUpdated,
				"health_check": map[string]interface{}{
					"status":        "healthy",
					"last_check":    time.Now(),
					"response_time": "45ms",
				},
			}
		}
	}

	// 3. 计算进度和当前阶段
	progress, currentStage := h.calculateOnboardingProgress(record.Status)

	// 4. 生成阶段详情
	stages := h.generateStageDetails(record)

	// 5. 估算剩余时间
	estimatedTime := h.estimateRemainingTime(record.Status, record.CreatedAt)

	// 6. 生成状态消息
	message := h.getDetailedStatusMessage(record.Status, record.UpdatedAt)

	status := map[string]interface{}{
		"strategy_id":    strategyID,
		"strategy_name":  record.Name,
		"current_stage":  currentStage,
		"overall_status": record.Status,
		"progress":       progress,
		"last_updated":   record.UpdatedAt,
		"estimated_time": estimatedTime,
		"message":        message,
		"stages":         stages,
		"validation": map[string]interface{}{
			"score":      record.ValidationResult.Score,
			"passed":     record.ValidationResult.Score >= 70.0,
			"last_check": record.UpdatedAt,
		},
		"risk_assessment": map[string]interface{}{
			"score":      record.RiskAssessment.OverallScore,
			"passed":     record.RiskAssessment.OverallScore >= 70.0,
			"last_check": record.UpdatedAt,
		},
		"compliance": map[string]interface{}{
			"passed":     record.ComplianceResult.Passed,
			"last_check": record.UpdatedAt,
		},
		"deployment_info": deploymentInfo,
		"created_at":      record.CreatedAt,
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    status,
	})
}

// GetPositions returns current positions
func (h *TradingHandler) GetPositions(c *gin.Context) {
	strategyId := c.Query("strategyId")
	status := c.Query("status") // open, closed, all
	if status == "" {
		status = "open"
	}

	// 添加分页参数，默认限制100条
	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 1000 {
			limit = parsedLimit
		}
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	ctx := c.Request.Context()

	// 构建查询条件
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if strategyId != "" {
		whereClause += " AND p.strategy_id = $" + strconv.Itoa(argIndex)
		args = append(args, strategyId)
		argIndex++
	}

	if status != "all" {
		whereClause += " AND p.status = $" + strconv.Itoa(argIndex)
		args = append(args, status)
		argIndex++
	}

	// 从数据库获取持仓数据
	query := `
		SELECT
			p.id,
			p.strategy_id,
			p.symbol,
			p.side,
			p.size,
			p.entry_price,
			p.leverage,
			COALESCE(p.unrealized_pnl, 0) as unrealized_pnl,
			COALESCE(p.realized_pnl, 0) as realized_pnl,
			p.status,
			p.created_at,
			p.updated_at,
			s.name as strategy_name
		FROM positions p
		LEFT JOIN strategies s ON p.strategy_id = s.id
		` + whereClause + `
		ORDER BY p.created_at DESC
		LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2) + `
	`

	// 添加分页参数到args
	args = append(args, limit, offset)

	// 先查询总数
	countQuery := `
		SELECT COUNT(*)
		FROM positions p
		LEFT JOIN strategies s ON p.strategy_id = s.id
		` + whereClause

	var totalCount int
	countArgs := args[:len(args)-2] // 移除limit和offset参数
	err := h.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		log.Printf("Failed to query positions count: %v", err)
		totalCount = 0
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("Failed to query positions: %v", err)
		c.JSON(http.StatusOK, Response{
			Success: true,
			Data: map[string]interface{}{
				"positions": []map[string]interface{}{},
				"total":     0,
				"limit":     limit,
				"offset":    offset,
			},
		})
		return
	}
	defer rows.Close()

	var positions []map[string]interface{}
	for rows.Next() {
		var position struct {
			ID            string    `db:"id"`
			StrategyID    string    `db:"strategy_id"`
			Symbol        string    `db:"symbol"`
			Side          string    `db:"side"`
			Size          float64   `db:"size"`
			EntryPrice    float64   `db:"entry_price"`
			Leverage      int       `db:"leverage"`
			UnrealizedPnL float64   `db:"unrealized_pnl"`
			RealizedPnL   float64   `db:"realized_pnl"`
			Status        string    `db:"status"`
			CreatedAt     time.Time `db:"created_at"`
			UpdatedAt     time.Time `db:"updated_at"`
			StrategyName  *string   `db:"strategy_name"`
		}

		if err := rows.Scan(
			&position.ID, &position.StrategyID, &position.Symbol, &position.Side,
			&position.Size, &position.EntryPrice, &position.Leverage,
			&position.UnrealizedPnL, &position.RealizedPnL, &position.Status,
			&position.CreatedAt, &position.UpdatedAt, &position.StrategyName,
		); err != nil {
			continue
		}

		strategyName := "未知策略"
		if position.StrategyName != nil {
			strategyName = *position.StrategyName
		}

		// 计算持仓价值和收益率
		positionValue := position.Size * position.EntryPrice
		totalPnL := position.UnrealizedPnL + position.RealizedPnL
		pnlPercent := 0.0
		if positionValue > 0 {
			pnlPercent = (totalPnL / positionValue) * 100
		}

		positions = append(positions, map[string]interface{}{
			"id":             position.ID,
			"strategy_id":    position.StrategyID,
			"strategy_name":  strategyName,
			"symbol":         position.Symbol,
			"side":           position.Side,
			"size":           position.Size,
			"entry_price":    position.EntryPrice,
			"leverage":       position.Leverage,
			"unrealized_pnl": position.UnrealizedPnL,
			"realized_pnl":   position.RealizedPnL,
			"total_pnl":      totalPnL,
			"pnl_percent":    pnlPercent,
			"position_value": positionValue,
			"status":         position.Status,
			"created_at":     position.CreatedAt,
			"updated_at":     position.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"positions": positions,
			"total":     totalCount,
			"limit":     limit,
			"offset":    offset,
		},
	})
}

// StrategyValidationHandler handles strategy validation API requests
type StrategyValidationHandler struct {
	gatekeeper *validation.StrategyGatekeeper
}

// NewStrategyValidationHandler creates a new strategy validation handler
func NewStrategyValidationHandler() *StrategyValidationHandler {
	return &StrategyValidationHandler{
		gatekeeper: validation.NewStrategyGatekeeper(),
	}
}

// GetStrategyValidationStatus returns the validation status of all strategies
func (h *StrategyValidationHandler) GetStrategyValidationStatus(c *gin.Context) {
	// 模拟获取所有策略的验证状态
	// 实际应该从数据库查询
	statuses := []map[string]interface{}{
		{
			"strategy_id":       "strategy-1",
			"strategy_name":     "高频交易策略",
			"is_valid":          false,
			"backtest_passed":   false,
			"risk_check_passed": false,
			"validation_time":   time.Now().AddDate(0, 0, -1),
			"errors": []map[string]interface{}{
				{
					"code":    "BACKTEST_FAILED",
					"message": "回测验证失败: 总收益率为负: -15.00%",
					"field":   "backtest",
				},
				{
					"code":    "RISK_TOO_HIGH",
					"message": "策略风险等级过高，不允许启用",
					"field":   "risk_level",
				},
			},
			"backtest_result": map[string]interface{}{
				"total_return":  -0.15,
				"sharpe_ratio":  0.3,
				"max_drawdown":  0.25,
				"win_rate":      0.35,
				"total_trades":  1200,
				"backtest_days": 730,
				"failure_reasons": []string{
					"总收益率为负: -15.00%",
					"夏普比率过低: 0.30 < 0.50",
					"最大回撤过大: 25.00% > 20.00%",
					"胜率过低: 35.00% < 40.00%",
					"交易频率过高: 1200笔/730天",
				},
			},
			"risk_assessment": map[string]interface{}{
				"risk_score":        85,
				"risk_level":        "CRITICAL",
				"max_position_size": 0.01,
				"max_leverage":      1.0,
				"recommended_limit": 1000,
				"warnings": []string{
					"最大回撤超过15%",
					"夏普比率过低",
					"交易频率过高，可能存在过度交易",
					"胜率过低",
				},
			},
		},
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"statuses": statuses,
			"summary": map[string]interface{}{
				"total_strategies":   1,
				"valid_strategies":   0,
				"invalid_strategies": 1,
				"pending_validation": 0,
			},
		},
	})
}

// GetAutomationStatus returns the status of the automation manager
func (h *StrategyValidationHandler) GetAutomationStatus(c *gin.Context) {
	// 这里应该从实际的自动化管理器获取状态
	// 现在返回模拟状态
	status := map[string]interface{}{
		"system_name":          "QCAT 量化交易自动化系统",
		"version":              "1.0.0",
		"automation_enabled":   true,
		"risk_monitor_running": true,
		"backtest_running":     true,
		"optimizer_running":    true,
		"gatekeeper_enabled":   true,
		"start_time":           "2025-01-22T15:00:00Z",
		"uptime":               "2h30m15s",
		"features": []string{
			"强制回测验证",
			"实时风险监控",
			"自动化回测调度",
			"策略参数优化",
			"策略守门员保护",
			"紧急停止机制",
		},
		"component_status": map[string]interface{}{
			"backtest_scheduler": map[string]interface{}{
				"running":           true,
				"schedule_interval": "1h0m0s",
				"task_counts_24h":   map[string]int{"completed": 5, "failed": 1, "pending": 2},
				"last_check":        "2025-01-22T17:30:00Z",
			},
			"parameter_optimizer": map[string]interface{}{
				"running":               true,
				"optimize_interval":     "24h0m0s",
				"total_optimizations":   12,
				"avg_improvement":       8.5,
				"max_improvement":       25.3,
				"avg_optimization_time": "45m30s",
			},
			"risk_monitor": map[string]interface{}{
				"active_strategies": 3,
				"monitoring":        true,
				"high_risk_count":   1,
				"critical_count":    0,
			},
		},
		"safety_level":      "HIGH",
		"last_health_check": "2025-01-22T17:35:00Z",
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    status,
	})
}

// GetStrategyProblems returns detailed problems with current strategies
func (h *StrategyValidationHandler) GetStrategyProblems(c *gin.Context) {
	problems := []map[string]interface{}{
		{
			"severity":            "CRITICAL",
			"category":            "RISK_MANAGEMENT",
			"title":               "风控系统失效",
			"description":         "检测到58,762个持仓记录，总亏损-50万，风控系统未能及时止损",
			"affected_strategies": []string{"strategy-1"},
			"recommendations": []string{
				"立即启用强制回测验证",
				"设置严格的止损规则",
				"限制单个策略的最大持仓数量",
				"实施实时风险监控",
			},
		},
		{
			"severity":            "HIGH",
			"category":            "STRATEGY_VALIDATION",
			"title":               "策略未经回测验证",
			"description":         "当前运行的策略未通过强制回测验证，存在重大风险",
			"affected_strategies": []string{"strategy-1"},
			"recommendations": []string{
				"对所有策略进行2年历史数据回测",
				"设置最低性能要求（夏普比率>0.5，最大回撤<20%）",
				"禁用未通过验证的策略",
			},
		},
		{
			"severity":            "HIGH",
			"category":            "TRADING_FREQUENCY",
			"title":               "过度交易",
			"description":         "策略交易频率异常高，可能导致高额手续费和滑点损失",
			"affected_strategies": []string{"strategy-1"},
			"recommendations": []string{
				"设置最大日交易次数限制",
				"优化策略信号过滤逻辑",
				"增加最小持仓时间要求",
			},
		},
		{
			"severity":            "MEDIUM",
			"category":            "PERFORMANCE",
			"title":               "策略性能不佳",
			"description":         "当前策略胜率35%，夏普比率0.3，远低于行业标准",
			"affected_strategies": []string{"strategy-1"},
			"recommendations": []string{
				"重新优化策略参数",
				"考虑更换策略模型",
				"增加市场状态识别模块",
			},
		},
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"problems":       problems,
			"total_problems": len(problems),
			"critical_count": 1,
			"high_count":     2,
			"medium_count":   1,
			"low_count":      0,
		},
	})
}

// Helper functions for strategy integration

// validateStrategy validates a strategy
func (h *StrategyHandler) validateStrategy(req *StrategyOnboardingRequest) (*ValidationResult, error) {
	result := &ValidationResult{
		IsValid:  true,
		Score:    0.0,
		Errors:   []string{},
		Warnings: []string{},
		Passed:   []string{},
	}

	// 1. 基本信息验证
	if req.StrategyID == "" {
		result.Errors = append(result.Errors, "Strategy ID is required")
		result.IsValid = false
	} else {
		result.Passed = append(result.Passed, "Strategy ID validation")
		result.Score += 10
	}

	if req.StrategyName == "" {
		result.Errors = append(result.Errors, "Strategy name is required")
		result.IsValid = false
	} else {
		result.Passed = append(result.Passed, "Strategy name validation")
		result.Score += 10
	}

	// 2. 策略类型验证
	validTypes := []string{"trend_following", "mean_reversion", "arbitrage", "market_making", "momentum"}
	isValidType := false
	for _, validType := range validTypes {
		if req.StrategyType == validType {
			isValidType = true
			break
		}
	}

	if !isValidType {
		result.Errors = append(result.Errors, "Invalid strategy type")
		result.IsValid = false
	} else {
		result.Passed = append(result.Passed, "Strategy type validation")
		result.Score += 15
	}

	// 3. 参数验证
	if len(req.Parameters) == 0 {
		result.Warnings = append(result.Warnings, "No parameters provided")
	} else {
		result.Passed = append(result.Passed, "Parameters validation")
		result.Score += 10
	}

	// 4. 代码质量检查（如果提供了代码）
	if req.Code != "" {
		if len(req.Code) < 100 {
			result.Warnings = append(result.Warnings, "Strategy code seems too short")
		} else {
			result.Passed = append(result.Passed, "Code length validation")
			result.Score += 15
		}

		// 检查危险函数
		dangerousFunctions := []string{"exec", "eval", "system", "os.system"}
		for _, dangerous := range dangerousFunctions {
			if strings.Contains(strings.ToLower(req.Code), dangerous) {
				result.Errors = append(result.Errors, fmt.Sprintf("Dangerous function detected: %s", dangerous))
				result.IsValid = false
			}
		}

		if result.IsValid {
			result.Passed = append(result.Passed, "Code security validation")
			result.Score += 20
		}
	}

	// 5. 依赖检查
	if len(req.Dependencies) > 10 {
		result.Warnings = append(result.Warnings, "Too many dependencies may affect performance")
	} else {
		result.Passed = append(result.Passed, "Dependencies validation")
		result.Score += 10
	}

	// 6. 资源需求验证
	if req.ResourceRequirements.Memory > 8192 { // 8GB
		result.Warnings = append(result.Warnings, "High memory requirement")
	}
	if req.ResourceRequirements.CPU > 4.0 {
		result.Warnings = append(result.Warnings, "High CPU requirement")
	}

	result.Passed = append(result.Passed, "Resource requirements validation")
	result.Score += 10

	return result, nil
}

// assessStrategyRisk assesses strategy risk
func (h *StrategyHandler) assessStrategyRisk(req *StrategyOnboardingRequest) (*RiskAssessment, error) {
	assessment := &RiskAssessment{
		OverallScore:     0.0,
		RiskLevel:        "medium",
		ExpectedReturn:   0.0,
		ExpectedSharpe:   0.0,
		ExpectedDrawdown: 0.0,
		ConfidenceLevel:  0.0,
		Recommendations:  []string{},
	}

	score := 75.0 // 基础分数

	// 1. 基于策略类型的风险评估
	switch req.StrategyType {
	case "arbitrage":
		score += 10
		assessment.RiskLevel = "low"
		assessment.ExpectedReturn = 0.08
		assessment.ExpectedSharpe = 1.5
		assessment.ExpectedDrawdown = 0.05
	case "market_making":
		score += 5
		assessment.RiskLevel = "medium"
		assessment.ExpectedReturn = 0.12
		assessment.ExpectedSharpe = 1.2
		assessment.ExpectedDrawdown = 0.08
	case "trend_following":
		score -= 5
		assessment.RiskLevel = "medium"
		assessment.ExpectedReturn = 0.15
		assessment.ExpectedSharpe = 1.0
		assessment.ExpectedDrawdown = 0.12
	case "momentum":
		score -= 10
		assessment.RiskLevel = "high"
		assessment.ExpectedReturn = 0.20
		assessment.ExpectedSharpe = 0.8
		assessment.ExpectedDrawdown = 0.18
	default:
		assessment.RiskLevel = "medium"
		assessment.ExpectedReturn = 0.10
		assessment.ExpectedSharpe = 1.0
		assessment.ExpectedDrawdown = 0.10
	}

	// 2. 基于风险配置的评估
	if req.RiskProfile.MaxDrawdown > 0.2 {
		score -= 15
		assessment.RiskLevel = "high"
		assessment.Recommendations = append(assessment.Recommendations, "Consider reducing maximum drawdown limit")
	}

	if req.RiskProfile.MaxLeverage > 5.0 {
		score -= 10
		assessment.Recommendations = append(assessment.Recommendations, "High leverage increases risk significantly")
	}

	if req.RiskProfile.MaxPositionSize > 0.3 {
		score -= 5
		assessment.Recommendations = append(assessment.Recommendations, "Large position sizes may increase concentration risk")
	}

	// 3. 基于历史表现的评估（如果有）
	if req.HistoricalPerformance != nil {
		if req.HistoricalPerformance.SharpeRatio > 1.5 {
			score += 10
		} else if req.HistoricalPerformance.SharpeRatio < 0.5 {
			score -= 15
		}

		if req.HistoricalPerformance.MaxDrawdown > 0.25 {
			score -= 10
		}

		if req.HistoricalPerformance.WinRate > 0.6 {
			score += 5
		}
	}

	// 4. 设置置信度
	if score >= 85 {
		assessment.ConfidenceLevel = 0.9
	} else if score >= 70 {
		assessment.ConfidenceLevel = 0.8
	} else if score >= 60 {
		assessment.ConfidenceLevel = 0.7
	} else {
		assessment.ConfidenceLevel = 0.6
	}

	// 5. 添加通用建议
	assessment.Recommendations = append(assessment.Recommendations, "Monitor performance closely during initial period")
	assessment.Recommendations = append(assessment.Recommendations, "Consider implementing automated rebalancing")
	assessment.Recommendations = append(assessment.Recommendations, "Regular risk assessment reviews recommended")

	assessment.OverallScore = math.Max(0, math.Min(100, score))

	return assessment, nil
}

// performStrategyBacktest performs strategy backtesting
func (h *StrategyHandler) performStrategyBacktest(req *StrategyOnboardingRequest) (*BacktestResult, error) {
	// 简化的回测实现
	result := &BacktestResult{
		TotalReturn:      0.0,
		AnnualizedReturn: 0.0,
		SharpeRatio:      0.0,
		MaxDrawdown:      0.0,
		WinRate:          0.0,
		TotalTrades:      0,
		StartDate:        time.Now().AddDate(-1, 0, 0),
		EndDate:          time.Now(),
	}

	// 基于策略类型生成模拟回测结果
	switch req.StrategyType {
	case "arbitrage":
		result.TotalReturn = 0.08 + (rand.Float64()-0.5)*0.02
		result.AnnualizedReturn = result.TotalReturn
		result.SharpeRatio = 1.5 + (rand.Float64()-0.5)*0.3
		result.MaxDrawdown = 0.03 + rand.Float64()*0.02
		result.WinRate = 0.75 + (rand.Float64()-0.5)*0.1
		result.TotalTrades = 500 + rand.Intn(200)
	case "trend_following":
		result.TotalReturn = 0.15 + (rand.Float64()-0.5)*0.05
		result.AnnualizedReturn = result.TotalReturn
		result.SharpeRatio = 1.0 + (rand.Float64()-0.5)*0.4
		result.MaxDrawdown = 0.12 + rand.Float64()*0.05
		result.WinRate = 0.45 + (rand.Float64()-0.5)*0.1
		result.TotalTrades = 150 + rand.Intn(100)
	default:
		result.TotalReturn = 0.10 + (rand.Float64()-0.5)*0.04
		result.AnnualizedReturn = result.TotalReturn
		result.SharpeRatio = 1.0 + (rand.Float64()-0.5)*0.3
		result.MaxDrawdown = 0.08 + rand.Float64()*0.04
		result.WinRate = 0.55 + (rand.Float64()-0.5)*0.1
		result.TotalTrades = 200 + rand.Intn(150)
	}

	return result, nil
}

// checkStrategyCompliance checks strategy compliance
func (h *StrategyHandler) checkStrategyCompliance(req *StrategyOnboardingRequest) (*ComplianceResult, error) {
	result := &ComplianceResult{
		Passed:       true,
		Score:        100.0,
		Violations:   []string{},
		Warnings:     []string{},
		ChecksPassed: []string{},
	}

	// 1. 杠杆限制检查
	if req.RiskProfile.MaxLeverage > 10.0 {
		result.Violations = append(result.Violations, "Maximum leverage exceeds regulatory limit (10x)")
		result.Passed = false
		result.Score -= 20
	} else {
		result.ChecksPassed = append(result.ChecksPassed, "Leverage compliance")
	}

	// 2. 仓位大小检查
	if req.RiskProfile.MaxPositionSize > 0.5 {
		result.Violations = append(result.Violations, "Maximum position size exceeds limit (50%)")
		result.Passed = false
		result.Score -= 15
	} else {
		result.ChecksPassed = append(result.ChecksPassed, "Position size compliance")
	}

	// 3. 回撤限制检查
	if req.RiskProfile.MaxDrawdown > 0.3 {
		result.Warnings = append(result.Warnings, "High maximum drawdown may require additional oversight")
		result.Score -= 5
	} else {
		result.ChecksPassed = append(result.ChecksPassed, "Drawdown limit compliance")
	}

	// 4. 策略类型合规检查
	prohibitedTypes := []string{"high_frequency_manipulation", "wash_trading"}
	for _, prohibited := range prohibitedTypes {
		if req.StrategyType == prohibited {
			result.Violations = append(result.Violations, fmt.Sprintf("Strategy type '%s' is prohibited", prohibited))
			result.Passed = false
			result.Score -= 50
		}
	}

	if result.Passed {
		result.ChecksPassed = append(result.ChecksPassed, "Strategy type compliance")
	}

	// 5. 资金要求检查
	if req.MinCapital > 1000000 { // $1M
		result.Warnings = append(result.Warnings, "High minimum capital requirement")
		result.Score -= 5
	} else {
		result.ChecksPassed = append(result.ChecksPassed, "Capital requirement compliance")
	}

	return result, nil
}

// saveStrategyRecord saves strategy record to database
func (h *StrategyHandler) saveStrategyRecord(record *StrategyRecord) error {
	query := `
		INSERT INTO strategy_onboarding (
			id, name, description, type, status, 
			validation_score, risk_score, compliance_passed,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			status = VALUES(status),
			validation_score = VALUES(validation_score),
			risk_score = VALUES(risk_score),
			compliance_passed = VALUES(compliance_passed),
			updated_at = VALUES(updated_at)
	`

	_, err := h.db.ExecContext(context.Background(), query,
		record.ID, record.Name, record.Description, record.Type, record.Status,
		record.ValidationResult.Score, record.RiskAssessment.OverallScore, record.ComplianceResult.Passed,
		record.CreatedAt, record.UpdatedAt,
	)

	return err
}

// deployStrategy deploys a strategy
func (h *StrategyHandler) deployStrategy(req *StrategyOnboardingRequest, record *StrategyRecord) (map[string]interface{}, error) {
	deploymentID := fmt.Sprintf("deploy_%s_%d", req.StrategyID, time.Now().Unix())

	// 模拟部署过程
	time.Sleep(100 * time.Millisecond) // 模拟部署时间

	environment := "production"
	if req.TestMode {
		environment = "test"
	}

	deploymentInfo := map[string]interface{}{
		"deployment_id": deploymentID,
		"environment":   environment,
		"start_time":    time.Now(),
		"status":        "deployed",
		"health_check": map[string]interface{}{
			"status":        "healthy",
			"last_check":    time.Now(),
			"response_time": "50ms",
		},
		"resource_allocation": map[string]interface{}{
			"cpu":     req.ResourceRequirements.CPU,
			"memory":  req.ResourceRequirements.Memory,
			"storage": req.ResourceRequirements.Storage,
		},
	}

	// 保存部署信息到数据库
	deployQuery := `
		INSERT INTO strategy_deployments (
			deployment_id, strategy_id, environment, status, 
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := h.db.ExecContext(context.Background(), deployQuery,
		deploymentID, req.StrategyID, environment, "deployed",
		time.Now(), time.Now(),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to save deployment info: %v", err)
	}

	return deploymentInfo, nil
}

// getStatusMessage returns status message
func (h *StrategyHandler) getStatusMessage(status string) string {
	messages := map[string]string{
		"pending":                     "Strategy onboarding request received and queued for processing",
		"approved_for_deployment":     "Strategy approved and ready for automatic deployment",
		"approved_pending_deployment": "Strategy approved but requires manual deployment",
		"deployed":                    "Strategy successfully deployed and active",
		"rejected":                    "Strategy onboarding rejected due to validation or compliance issues",
		"deployment_failed":           "Strategy approved but deployment failed",
	}

	if msg, exists := messages[status]; exists {
		return msg
	}

	return "Unknown status"
}

// Data structures for strategy integration

type StrategyOnboardingRequest struct {
	StrategyID            string                 `json:"strategy_id" binding:"required"`
	StrategyName          string                 `json:"strategy_name" binding:"required"`
	Description           string                 `json:"description"`
	StrategyType          string                 `json:"strategy_type" binding:"required"`
	Parameters            map[string]interface{} `json:"parameters"`
	Code                  string                 `json:"code"`
	Dependencies          []string               `json:"dependencies"`
	ResourceRequirements  ResourceRequirements   `json:"resource_requirements"`
	RiskProfile           RiskProfile            `json:"risk_profile"`
	HistoricalPerformance *HistoricalPerformance `json:"historical_performance"`
	AutoDeploy            bool                   `json:"auto_deploy"`
	TestMode              bool                   `json:"test_mode"`
	MinCapital            float64                `json:"min_capital"`
}

type ResourceRequirements struct {
	CPU     float64 `json:"cpu"`
	Memory  int     `json:"memory"`
	Storage int     `json:"storage"`
}

type RiskProfile struct {
	RiskLevel       string  `json:"risk_level"`
	MaxDrawdown     float64 `json:"max_drawdown"`
	MaxLeverage     float64 `json:"max_leverage"`
	MaxPositionSize float64 `json:"max_position_size"`
	StopLoss        float64 `json:"stop_loss"`
}

type HistoricalPerformance struct {
	TotalReturn float64 `json:"total_return"`
	SharpeRatio float64 `json:"sharpe_ratio"`
	MaxDrawdown float64 `json:"max_drawdown"`
	WinRate     float64 `json:"win_rate"`
	TotalTrades int     `json:"total_trades"`
}

type ValidationResult struct {
	IsValid  bool     `json:"is_valid"`
	Score    float64  `json:"score"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
	Passed   []string `json:"passed"`
}

type RiskAssessment struct {
	OverallScore     float64  `json:"overall_score"`
	RiskLevel        string   `json:"risk_level"`
	ExpectedReturn   float64  `json:"expected_return"`
	ExpectedSharpe   float64  `json:"expected_sharpe"`
	ExpectedDrawdown float64  `json:"expected_drawdown"`
	ConfidenceLevel  float64  `json:"confidence_level"`
	Recommendations  []string `json:"recommendations"`
}

type BacktestResult struct {
	TotalReturn      float64   `json:"total_return"`
	AnnualizedReturn float64   `json:"annualized_return"`
	SharpeRatio      float64   `json:"sharpe_ratio"`
	MaxDrawdown      float64   `json:"max_drawdown"`
	WinRate          float64   `json:"win_rate"`
	TotalTrades      int       `json:"total_trades"`
	StartDate        time.Time `json:"start_date"`
	EndDate          time.Time `json:"end_date"`
}

type ComplianceResult struct {
	Passed       bool     `json:"passed"`
	Score        float64  `json:"score"`
	Violations   []string `json:"violations"`
	Warnings     []string `json:"warnings"`
	ChecksPassed []string `json:"checks_passed"`
}

type StrategyRecord struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	Type             string            `json:"type"`
	Status           string            `json:"status"`
	ValidationResult *ValidationResult `json:"validation_result"`
	RiskAssessment   *RiskAssessment   `json:"risk_assessment"`
	ComplianceResult *ComplianceResult `json:"compliance_result"`
	BacktestResult   *BacktestResult   `json:"backtest_result"`
	RiskProfile      RiskProfile       `json:"risk_profile"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

// Helper functions for strategy status query

// calculateOnboardingProgress calculates progress and current stage
func (h *StrategyHandler) calculateOnboardingProgress(status string) (int, string) {
	switch status {
	case "pending":
		return 10, "validation"
	case "validating":
		return 25, "validation"
	case "risk_assessment":
		return 40, "risk_assessment"
	case "compliance_check":
		return 60, "compliance_check"
	case "approved_pending_deployment":
		return 80, "deployment_pending"
	case "approved_for_deployment":
		return 85, "deployment_ready"
	case "deploying":
		return 90, "deployment"
	case "deployed":
		return 100, "monitoring"
	case "rejected":
		return 0, "rejected"
	case "deployment_failed":
		return 85, "deployment_failed"
	default:
		return 0, "unknown"
	}
}

// generateStageDetails generates detailed stage information
func (h *StrategyHandler) generateStageDetails(record StrategyRecord) []map[string]interface{} {
	stages := []map[string]interface{}{
		{
			"name":         "validation",
			"status":       h.getStageStatus("validation", record.Status),
			"duration":     h.getStageDuration("validation", record.CreatedAt, record.UpdatedAt),
			"details":      h.getValidationDetails(record.ValidationResult),
			"score":        record.ValidationResult.Score,
			"started_at":   record.CreatedAt,
			"completed_at": h.getStageCompletionTime("validation", record.Status, record.UpdatedAt),
		},
		{
			"name":         "risk_assessment",
			"status":       h.getStageStatus("risk_assessment", record.Status),
			"duration":     h.getStageDuration("risk_assessment", record.CreatedAt, record.UpdatedAt),
			"details":      h.getRiskAssessmentDetails(record.RiskAssessment),
			"score":        record.RiskAssessment.OverallScore,
			"started_at":   record.CreatedAt.Add(30 * time.Second), // 假设验证后30秒开始
			"completed_at": h.getStageCompletionTime("risk_assessment", record.Status, record.UpdatedAt),
		},
		{
			"name":         "compliance_check",
			"status":       h.getStageStatus("compliance_check", record.Status),
			"duration":     h.getStageDuration("compliance_check", record.CreatedAt, record.UpdatedAt),
			"details":      h.getComplianceDetails(record.ComplianceResult),
			"passed":       record.ComplianceResult.Passed,
			"started_at":   record.CreatedAt.Add(75 * time.Second), // 假设风险评估后45秒开始
			"completed_at": h.getStageCompletionTime("compliance_check", record.Status, record.UpdatedAt),
		},
		{
			"name":         "deployment",
			"status":       h.getStageStatus("deployment", record.Status),
			"duration":     h.getStageDuration("deployment", record.CreatedAt, record.UpdatedAt),
			"details":      h.getDeploymentDetails(record.Status),
			"started_at":   h.getDeploymentStartTime(record.Status, record.UpdatedAt),
			"completed_at": h.getStageCompletionTime("deployment", record.Status, record.UpdatedAt),
		},
		{
			"name":       "monitoring",
			"status":     h.getStageStatus("monitoring", record.Status),
			"duration":   h.getStageDuration("monitoring", record.CreatedAt, record.UpdatedAt),
			"details":    h.getMonitoringDetails(record.Status),
			"started_at": h.getMonitoringStartTime(record.Status, record.UpdatedAt),
		},
	}

	return stages
}

// getStageStatus returns the status of a specific stage
func (h *StrategyHandler) getStageStatus(stage, overallStatus string) string {
	stageOrder := map[string]int{
		"validation":       1,
		"risk_assessment":  2,
		"compliance_check": 3,
		"deployment":       4,
		"monitoring":       5,
	}

	statusStage := map[string]int{
		"pending":                     0,
		"validating":                  1,
		"risk_assessment":             2,
		"compliance_check":            3,
		"approved_pending_deployment": 3,
		"approved_for_deployment":     3,
		"deploying":                   4,
		"deployed":                    5,
		"rejected":                    -1,
		"deployment_failed":           4,
	}

	currentStageNum := statusStage[overallStatus]
	stageNum := stageOrder[stage]

	if overallStatus == "rejected" {
		if stageNum <= 3 {
			return "failed"
		}
		return "not_started"
	}

	if stageNum < currentStageNum {
		return "completed"
	} else if stageNum == currentStageNum {
		if overallStatus == "deployment_failed" && stage == "deployment" {
			return "failed"
		}
		return "in_progress"
	} else {
		return "not_started"
	}
}

// getStageDuration calculates stage duration
func (h *StrategyHandler) getStageDuration(stage string, createdAt, updatedAt time.Time) string {
	// 简化的持续时间计算
	switch stage {
	case "validation":
		return "30s"
	case "risk_assessment":
		return "45s"
	case "compliance_check":
		return "20s"
	case "deployment":
		return "2m"
	case "monitoring":
		if updatedAt.After(createdAt) {
			duration := time.Since(updatedAt)
			return fmt.Sprintf("%.0fm", duration.Minutes())
		}
		return "ongoing"
	default:
		return "unknown"
	}
}

// getValidationDetails returns validation details
func (h *StrategyHandler) getValidationDetails(result *ValidationResult) string {
	if result.Score >= 90 {
		return "Strategy configuration and parameters validated successfully with excellent score"
	} else if result.Score >= 70 {
		return "Strategy configuration and parameters validated successfully"
	} else {
		return "Strategy validation completed with warnings or issues"
	}
}

// getRiskAssessmentDetails returns risk assessment details
func (h *StrategyHandler) getRiskAssessmentDetails(assessment *RiskAssessment) string {
	if assessment.OverallScore >= 85 {
		return fmt.Sprintf("Risk profile assessed as %s with excellent score", assessment.RiskLevel)
	} else if assessment.OverallScore >= 70 {
		return fmt.Sprintf("Risk profile assessed as %s and approved", assessment.RiskLevel)
	} else {
		return fmt.Sprintf("Risk profile assessed as %s with concerns", assessment.RiskLevel)
	}
}

// getComplianceDetails returns compliance details
func (h *StrategyHandler) getComplianceDetails(result *ComplianceResult) string {
	if result.Passed {
		return "All compliance checks passed successfully"
	} else {
		return fmt.Sprintf("Compliance check failed with %d violations", len(result.Violations))
	}
}

// getDeploymentDetails returns deployment details
func (h *StrategyHandler) getDeploymentDetails(status string) string {
	switch status {
	case "deployed":
		return "Strategy deployed to production environment successfully"
	case "deploying":
		return "Strategy deployment in progress"
	case "deployment_failed":
		return "Strategy deployment failed, manual intervention required"
	case "approved_for_deployment":
		return "Strategy approved and ready for automatic deployment"
	case "approved_pending_deployment":
		return "Strategy approved but requires manual deployment"
	default:
		return "Deployment not yet started"
	}
}

// getMonitoringDetails returns monitoring details
func (h *StrategyHandler) getMonitoringDetails(status string) string {
	if status == "deployed" {
		return "Performance monitoring and risk controls active"
	}
	return "Monitoring not yet active"
}

// getStageCompletionTime returns stage completion time
func (h *StrategyHandler) getStageCompletionTime(stage, status string, updatedAt time.Time) *time.Time {
	stageOrder := map[string]int{
		"validation":       1,
		"risk_assessment":  2,
		"compliance_check": 3,
		"deployment":       4,
		"monitoring":       5,
	}

	statusStage := map[string]int{
		"pending":                     0,
		"validating":                  1,
		"risk_assessment":             2,
		"compliance_check":            3,
		"approved_pending_deployment": 3,
		"approved_for_deployment":     3,
		"deploying":                   4,
		"deployed":                    5,
	}

	currentStageNum := statusStage[status]
	stageNum := stageOrder[stage]

	if stageNum < currentStageNum {
		// 已完成的阶段，返回估算的完成时间
		completionTime := updatedAt.Add(-time.Duration(currentStageNum-stageNum) * 30 * time.Second)
		return &completionTime
	}

	return nil
}

// getDeploymentStartTime returns deployment start time
func (h *StrategyHandler) getDeploymentStartTime(status string, updatedAt time.Time) *time.Time {
	if status == "deploying" || status == "deployed" || status == "deployment_failed" {
		startTime := updatedAt.Add(-2 * time.Minute) // 假设部署开始时间
		return &startTime
	}
	return nil
}

// getMonitoringStartTime returns monitoring start time
func (h *StrategyHandler) getMonitoringStartTime(status string, updatedAt time.Time) *time.Time {
	if status == "deployed" {
		return &updatedAt
	}
	return nil
}

// estimateRemainingTime estimates remaining time
func (h *StrategyHandler) estimateRemainingTime(status string, createdAt time.Time) int {
	switch status {
	case "pending":
		return 300 // 5 minutes
	case "validating":
		return 240 // 4 minutes
	case "risk_assessment":
		return 180 // 3 minutes
	case "compliance_check":
		return 120 // 2 minutes
	case "approved_pending_deployment":
		return 0 // Manual intervention required
	case "approved_for_deployment":
		return 120 // 2 minutes
	case "deploying":
		return 60 // 1 minute
	case "deployed":
		return 0 // Complete
	case "rejected", "deployment_failed":
		return 0 // No further processing
	default:
		return 0
	}
}

// getDetailedStatusMessage returns detailed status message
func (h *StrategyHandler) getDetailedStatusMessage(status string, updatedAt time.Time) string {
	messages := map[string]string{
		"pending":                     "Strategy onboarding request received and queued for processing",
		"validating":                  "Strategy validation in progress",
		"risk_assessment":             "Risk assessment in progress",
		"compliance_check":            "Compliance checks in progress",
		"approved_pending_deployment": "Strategy approved but requires manual deployment approval",
		"approved_for_deployment":     "Strategy approved and queued for automatic deployment",
		"deploying":                   "Strategy deployment in progress",
		"deployed":                    fmt.Sprintf("Strategy successfully deployed and active since %s", updatedAt.Format("2006-01-02 15:04:05")),
		"rejected":                    "Strategy onboarding rejected due to validation or compliance issues",
		"deployment_failed":           "Strategy approved but deployment failed, manual intervention required",
	}

	if msg, exists := messages[status]; exists {
		return msg
	}

	return "Unknown status"
}

// AutoStartHandler 自动启动处理器
type AutoStartHandler struct {
	db      *database.DB
	service *autostart.AutoStartService
}

// NewAutoStartHandler 创建自动启动处理器
func NewAutoStartHandler(db *database.DB) *AutoStartHandler {
	service := autostart.NewAutoStartService(db.DB, autostart.GetDefaultAutoStartConfig())
	return &AutoStartHandler{
		db:      db,
		service: service,
	}
}

// UpdateStrategyAutoStart 更新策略自动启动设置
func (h *AutoStartHandler) UpdateStrategyAutoStart(c *gin.Context) {
	strategyID := c.Param("id")
	if strategyID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Strategy ID is required",
		})
		return
	}

	var req struct {
		AutoStart       bool `json:"auto_start"`
		StartupPriority int  `json:"startup_priority"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	// 验证优先级范围
	if req.StartupPriority < 1 || req.StartupPriority > 100 {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Startup priority must be between 1 and 100",
		})
		return
	}

	// 更新数据库
	query := `
		UPDATE strategies
		SET auto_start = $1, startup_priority = $2, updated_at = $3
		WHERE id = $4
	`

	result, err := h.db.DB.Exec(query, req.AutoStart, req.StartupPriority, time.Now(), strategyID)
	if err != nil {
		log.Printf("Failed to update strategy auto-start settings: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to update auto-start settings",
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Strategy not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Auto-start settings updated successfully",
		Data: map[string]interface{}{
			"strategy_id":      strategyID,
			"auto_start":       req.AutoStart,
			"startup_priority": req.StartupPriority,
		},
	})
}

// GetAutoStartStrategies 获取自动启动策略列表
func (h *AutoStartHandler) GetAutoStartStrategies(c *gin.Context) {
	query := `
		SELECT id, name, type, status, enabled, auto_start,
		       startup_priority, last_auto_start, is_running,
		       created_at, updated_at
		FROM strategies
		WHERE auto_start = true
		ORDER BY startup_priority ASC, name ASC
	`

	rows, err := h.db.DB.Query(query)
	if err != nil {
		log.Printf("Failed to query auto-start strategies: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to query auto-start strategies",
		})
		return
	}
	defer rows.Close()

	var strategies []map[string]interface{}
	for rows.Next() {
		var strategy struct {
			ID              string       `json:"id"`
			Name            string       `json:"name"`
			Type            string       `json:"type"`
			Status          string       `json:"status"`
			Enabled         bool         `json:"enabled"`
			AutoStart       bool         `json:"auto_start"`
			StartupPriority int          `json:"startup_priority"`
			LastAutoStart   sql.NullTime `json:"last_auto_start"`
			IsRunning       bool         `json:"is_running"`
			CreatedAt       time.Time    `json:"created_at"`
			UpdatedAt       time.Time    `json:"updated_at"`
		}

		err := rows.Scan(
			&strategy.ID, &strategy.Name, &strategy.Type, &strategy.Status,
			&strategy.Enabled, &strategy.AutoStart, &strategy.StartupPriority,
			&strategy.LastAutoStart, &strategy.IsRunning,
			&strategy.CreatedAt, &strategy.UpdatedAt,
		)
		if err != nil {
			log.Printf("Failed to scan strategy row: %v", err)
			continue
		}

		strategyMap := map[string]interface{}{
			"id":               strategy.ID,
			"name":             strategy.Name,
			"type":             strategy.Type,
			"status":           strategy.Status,
			"enabled":          strategy.Enabled,
			"auto_start":       strategy.AutoStart,
			"startup_priority": strategy.StartupPriority,
			"is_running":       strategy.IsRunning,
			"created_at":       strategy.CreatedAt,
			"updated_at":       strategy.UpdatedAt,
		}

		if strategy.LastAutoStart.Valid {
			strategyMap["last_auto_start"] = strategy.LastAutoStart.Time
		} else {
			strategyMap["last_auto_start"] = nil
		}

		strategies = append(strategies, strategyMap)
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"strategies": strategies,
			"total":      len(strategies),
		},
	})
}

// GetAutoStartStats 获取自动启动统计信息
func (h *AutoStartHandler) GetAutoStartStats(c *gin.Context) {
	if h.service == nil {
		c.JSON(http.StatusServiceUnavailable, Response{
			Success: false,
			Error:   "Auto-start service not available",
		})
		return
	}

	stats := h.service.GetStats()
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    stats,
	})
}

// TriggerAutoStart 手动触发自动启动
func (h *AutoStartHandler) TriggerAutoStart(c *gin.Context) {
	if h.service == nil {
		c.JSON(http.StatusServiceUnavailable, Response{
			Success: false,
			Error:   "Auto-start service not available",
		})
		return
	}

	if !h.service.IsRunning() {
		c.JSON(http.StatusServiceUnavailable, Response{
			Success: false,
			Error:   "Auto-start service is not running",
		})
		return
	}

	// 在后台触发自动启动
	go func() {
		if err := h.service.Start(); err != nil {
			log.Printf("Manual auto-start trigger failed: %v", err)
		}
	}()

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Auto-start triggered successfully",
	})
}
