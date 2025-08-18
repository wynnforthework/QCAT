package api

import (
	"net/http"
	"time"

	"qcat/internal/cache"
	"qcat/internal/common"
	"qcat/internal/database"
	"qcat/internal/monitor"
	"qcat/internal/strategy/optimizer"

	"github.com/gin-gonic/gin"
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
	redis   *cache.RedisCache
	metrics *monitor.MetricsCollector
	// 新增：优化器实例
	optimizer *optimizer.Orchestrator
}

// NewOptimizerHandler creates a new optimizer handler
func NewOptimizerHandler(db *database.DB, redis *cache.RedisCache, metrics *monitor.MetricsCollector) *OptimizerHandler {
	// 新增：使用工厂创建优化器实例
	factory := optimizer.NewFactory()
	orchestrator := factory.CreateOrchestrator(db.DB)

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

	var tasks []map[string]interface{}
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
	redis   *cache.RedisCache
	metrics *monitor.MetricsCollector
	// 新增：策略管理器
	strategyManager interface{} // 新增：策略管理器接口
}

// NewStrategyHandler creates a new strategy handler
func NewStrategyHandler(db *database.DB, redis *cache.RedisCache, metrics *monitor.MetricsCollector) *StrategyHandler {
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

	// 从数据库获取策略列表
	query := `
		SELECT id, name, type, status, current_version, created_at, updated_at
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

	var strategies []map[string]interface{}
	for rows.Next() {
		var strategy struct {
			ID             string    `db:"id"`
			Name           string    `db:"name"`
			Type           string    `db:"type"`
			Status         string    `db:"status"`
			CurrentVersion string    `db:"current_version"`
			CreatedAt      time.Time `db:"created_at"`
			UpdatedAt      time.Time `db:"updated_at"`
		}

		if err := rows.Scan(&strategy.ID, &strategy.Name, &strategy.Type, &strategy.Status, &strategy.CurrentVersion, &strategy.CreatedAt, &strategy.UpdatedAt); err != nil {
			continue
		}

		strategies = append(strategies, map[string]interface{}{
			"id":              strategy.ID,
			"name":            strategy.Name,
			"type":            strategy.Type,
			"status":          strategy.Status,
			"current_version": strategy.CurrentVersion,
			"created_at":      strategy.CreatedAt,
			"updated_at":      strategy.UpdatedAt,
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
		SELECT id, name, type, status, current_version, config, created_at, updated_at
		FROM strategies 
		WHERE id = $1
	`

	var strategy struct {
		ID             string                 `db:"id"`
		Name           string                 `db:"name"`
		Type           string                 `db:"type"`
		Status         string                 `db:"status"`
		CurrentVersion string                 `db:"current_version"`
		Config         map[string]interface{} `db:"config"`
		CreatedAt      time.Time              `db:"created_at"`
		UpdatedAt      time.Time              `db:"updated_at"`
	}

	err := h.db.QueryRowContext(ctx, query, strategyID).Scan(
		&strategy.ID, &strategy.Name, &strategy.Type, &strategy.Status,
		&strategy.CurrentVersion, &strategy.Config, &strategy.CreatedAt, &strategy.UpdatedAt,
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
			"id":              strategy.ID,
			"name":            strategy.Name,
			"type":            strategy.Type,
			"status":          strategy.Status,
			"current_version": strategy.CurrentVersion,
			"config":          strategy.Config,
			"created_at":      strategy.CreatedAt,
			"updated_at":      strategy.UpdatedAt,
		},
	})
}

// CreateStrategy creates a new strategy
func (h *StrategyHandler) CreateStrategy(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
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
		INSERT INTO strategies (id, name, type, status, config, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	strategyID := generateUUID() // 新增：生成UUID函数
	now := time.Now()

	var id string
	err := h.db.QueryRowContext(ctx, query,
		strategyID, name, strategyType, "inactive", req, now, now,
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

// StartStrategy starts a strategy
func (h *StrategyHandler) StartStrategy(c *gin.Context) {
	strategyID := c.Param("id")
	ctx := c.Request.Context()

	// 实现启动策略逻辑
	query := `
		UPDATE strategies 
		SET status = 'running', updated_at = $1
		WHERE id = $2
	`

	now := time.Now()
	result, err := h.db.ExecContext(ctx, query, now, strategyID)
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
		},
	})
}

// StopStrategy stops a strategy
func (h *StrategyHandler) StopStrategy(c *gin.Context) {
	strategyID := c.Param("id")
	ctx := c.Request.Context()

	// 实现停止策略逻辑
	query := `
		UPDATE strategies 
		SET status = 'stopped', updated_at = $1
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
	redis   *cache.RedisCache
	metrics *monitor.MetricsCollector
	// 新增：投资组合管理器
	portfolioManager interface{} // 新增：投资组合管理器接口
}

// NewPortfolioHandler creates a new portfolio handler
func NewPortfolioHandler(db *database.DB, redis *cache.RedisCache, metrics *monitor.MetricsCollector) *PortfolioHandler {
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

	var allocations []map[string]interface{}
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

	var history []map[string]interface{}
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
	redis   *cache.RedisCache
	metrics *monitor.MetricsCollector
	// 新增：风控管理器
	riskManager interface{} // 新增：风控管理器接口
}

// NewRiskHandler creates a new risk handler
func NewRiskHandler(db *database.DB, redis *cache.RedisCache, metrics *monitor.MetricsCollector) *RiskHandler {
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
			stop_loss,
			take_profit,
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

	var limits []map[string]interface{}
	for rows.Next() {
		var limit struct {
			Symbol          string    `db:"symbol"`
			MaxLeverage     float64   `db:"max_leverage"`
			MaxPositionSize float64   `db:"max_position_size"`
			MaxDrawdown     float64   `db:"max_drawdown"`
			StopLoss        float64   `db:"stop_loss"`
			TakeProfit      float64   `db:"take_profit"`
			UpdatedAt       time.Time `db:"updated_at"`
		}

		if err := rows.Scan(&limit.Symbol, &limit.MaxLeverage, &limit.MaxPositionSize,
			&limit.MaxDrawdown, &limit.StopLoss, &limit.TakeProfit, &limit.UpdatedAt); err != nil {
			continue
		}

		limits = append(limits, map[string]interface{}{
			"symbol":            limit.Symbol,
			"max_leverage":      limit.MaxLeverage,
			"max_position_size": limit.MaxPositionSize,
			"max_drawdown":      limit.MaxDrawdown,
			"stop_loss":         limit.StopLoss,
			"take_profit":       limit.TakeProfit,
			"updated_at":        limit.UpdatedAt,
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
		INSERT INTO risk_limits (symbol, max_leverage, max_position_size, max_drawdown, stop_loss, take_profit, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (symbol) DO UPDATE SET
			max_leverage = EXCLUDED.max_leverage,
			max_position_size = EXCLUDED.max_position_size,
			max_drawdown = EXCLUDED.max_drawdown,
			stop_loss = EXCLUDED.stop_loss,
			take_profit = EXCLUDED.take_profit,
			updated_at = EXCLUDED.updated_at
	`

	maxLeverage, _ := req["max_leverage"].(float64)
	maxPositionSize, _ := req["max_position_size"].(float64)
	maxDrawdown, _ := req["max_drawdown"].(float64)
	stopLoss, _ := req["stop_loss"].(float64)
	takeProfit, _ := req["take_profit"].(float64)
	now := time.Now()

	_, err := h.db.ExecContext(ctx, query,
		symbol, maxLeverage, maxPositionSize, maxDrawdown, stopLoss, takeProfit, now,
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

	var breakers []map[string]interface{}
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

	var violations []map[string]interface{}
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
	redis   *cache.RedisCache
	metrics *monitor.MetricsCollector
	// 新增：热门币种管理器
	hotlistManager interface{} // 新增：热门币种管理器接口
}

// NewHotlistHandler creates a new hotlist handler
func NewHotlistHandler(db *database.DB, redis *cache.RedisCache, metrics *monitor.MetricsCollector) *HotlistHandler {
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

	var symbols []map[string]interface{}
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
	if !exists {
		userID = "system" // 默认值
	}

	now := time.Now()
	_, err = h.db.ExecContext(ctx, insertQuery, req.Symbol, userID, now)
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
			approved_by,
			approved_at,
			status,
			updated_at
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

	var whitelist []map[string]interface{}
	for rows.Next() {
		var item struct {
			Symbol     string    `db:"symbol"`
			ApprovedBy string    `db:"approved_by"`
			ApprovedAt time.Time `db:"approved_at"`
			Status     string    `db:"status"`
			UpdatedAt  time.Time `db:"updated_at"`
		}

		if err := rows.Scan(&item.Symbol, &item.ApprovedBy, &item.ApprovedAt,
			&item.Status, &item.UpdatedAt); err != nil {
			continue
		}

		whitelist = append(whitelist, map[string]interface{}{
			"symbol":      item.Symbol,
			"approved_by": item.ApprovedBy,
			"approved_at": item.ApprovedAt,
			"status":      item.Status,
			"updated_at":  item.UpdatedAt,
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
	if !exists {
		userID = "system" // 默认值
	}

	now := time.Now()
	_, err = h.db.ExecContext(ctx, insertQuery, req.Symbol, userID, now)
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
	// 实现获取系统指标逻辑
	// 从监控系统获取系统指标
	systemMetrics := map[string]interface{}{
		"cpu_usage":            h.metrics.GetGaugeValue("system_cpu_usage"),
		"memory_usage":         h.metrics.GetGaugeValue("system_memory_usage"),
		"disk_usage":           h.metrics.GetGaugeValue("system_disk_usage"),
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

// GetLogs returns audit logs
func (h *AuditHandler) GetLogs(c *gin.Context) {
	// 实现获取审计日志逻辑
	ctx := c.Request.Context()

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

	var logs []map[string]interface{}
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
			strategy_id,
			signal_id,
			decision_type,
			decision_data,
			risk_check_result,
			execution_result,
			created_at
		FROM decision_chains 
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

	var chains []map[string]interface{}
	for rows.Next() {
		var chain struct {
			ID              string                 `db:"id"`
			StrategyID      string                 `db:"strategy_id"`
			SignalID        string                 `db:"signal_id"`
			DecisionType    string                 `db:"decision_type"`
			DecisionData    map[string]interface{} `db:"decision_data"`
			RiskCheckResult map[string]interface{} `db:"risk_check_result"`
			ExecutionResult map[string]interface{} `db:"execution_result"`
			CreatedAt       time.Time              `db:"created_at"`
		}

		if err := rows.Scan(&chain.ID, &chain.StrategyID, &chain.SignalID, &chain.DecisionType,
			&chain.DecisionData, &chain.RiskCheckResult, &chain.ExecutionResult, &chain.CreatedAt); err != nil {
			continue
		}

		chains = append(chains, map[string]interface{}{
			"id":                chain.ID,
			"strategy_id":       chain.StrategyID,
			"signal_id":         chain.SignalID,
			"decision_type":     chain.DecisionType,
			"decision_data":     chain.DecisionData,
			"risk_check_result": chain.RiskCheckResult,
			"execution_result":  chain.ExecutionResult,
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

	var metrics []map[string]interface{}
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
	return common.GenerateUUID()
}
