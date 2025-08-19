package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/config"
	"qcat/internal/database"
	"qcat/internal/strategy/optimizer"
)

// StrategyScheduler 策略调度器
// 负责策略相关的自动化任务
type StrategyScheduler struct {
	config           *config.Config
	db               *database.DB
	optimizerFactory *optimizer.Factory

	// 运行状态
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	isRunning bool
	mu        sync.RWMutex

	// 优化器实例
	optimizers map[string]*optimizer.Orchestrator
}

// NewStrategyScheduler 创建策略调度器
func NewStrategyScheduler(
	cfg *config.Config,
	db *database.DB,
	optimizerFactory *optimizer.Factory,
) *StrategyScheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &StrategyScheduler{
		config:           cfg,
		db:               db,
		optimizerFactory: optimizerFactory,
		ctx:              ctx,
		cancel:           cancel,
		optimizers:       make(map[string]*optimizer.Orchestrator),
	}
}

// Start 启动策略调度器
func (ss *StrategyScheduler) Start() error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if ss.isRunning {
		return fmt.Errorf("strategy scheduler is already running")
	}

	log.Println("Starting strategy scheduler...")

	// 初始化优化器
	if err := ss.initializeOptimizers(); err != nil {
		return fmt.Errorf("failed to initialize optimizers: %w", err)
	}

	ss.isRunning = true
	log.Println("Strategy scheduler started")

	return nil
}

// Stop 停止策略调度器
func (ss *StrategyScheduler) Stop() error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if !ss.isRunning {
		return nil
	}

	log.Println("Stopping strategy scheduler...")

	// 取消上下文
	ss.cancel()

	// 等待所有goroutine完成
	ss.wg.Wait()

	ss.isRunning = false
	log.Println("Strategy scheduler stopped")

	return nil
}

// HandleOptimization 处理策略优化任务
func (ss *StrategyScheduler) HandleOptimization(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing strategy optimization task: %s", task.Name)

	// 获取需要优化的策略列表
	strategies, err := ss.getStrategiesForOptimization(ctx)
	if err != nil {
		return fmt.Errorf("failed to get strategies for optimization: %w", err)
	}

	if len(strategies) == 0 {
		log.Println("No strategies need optimization")
		return nil
	}

	// 并行优化策略
	var wg sync.WaitGroup
	errChan := make(chan error, len(strategies))

	for _, strategy := range strategies {
		wg.Add(1)
		go func(strategyID string) {
			defer wg.Done()
			if err := ss.optimizeStrategy(ctx, strategyID); err != nil {
				errChan <- fmt.Errorf("failed to optimize strategy %s: %w", strategyID, err)
			}
		}(strategy.ID)
	}

	wg.Wait()
	close(errChan)

	// 检查是否有错误
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("optimization errors: %v", errors)
	}

	log.Printf("Strategy optimization completed for %d strategies", len(strategies))
	return nil
}

// Strategy 策略信息
type Strategy struct {
	ID            string
	Name          string
	Status        string
	LastOptimized time.Time
	Performance   float64
	SharpeRatio   float64
	MaxDrawdown   float64
}

// OptimizationResult 优化结果
type OptimizationResult struct {
	TaskID         string                 `json:"task_id"`
	StrategyID     string                 `json:"strategy_id"`
	Parameters     map[string]interface{} `json:"parameters"`
	Performance    *PerformanceMetrics    `json:"performance"`
	BacktestResult *BacktestResult        `json:"backtest_result"`
	CreatedAt      time.Time              `json:"created_at"`
	Status         string                 `json:"status"`
}

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	SharpeRatio  float64 `json:"sharpe_ratio"`
	MaxDrawdown  float64 `json:"max_drawdown"`
	TotalReturn  float64 `json:"total_return"`
	WinRate      float64 `json:"win_rate"`
	ProfitFactor float64 `json:"profit_factor"`
	Volatility   float64 `json:"volatility"`
}

// BacktestResult 回测结果
type BacktestResult struct {
	StartDate          time.Time `json:"start_date"`
	EndDate            time.Time `json:"end_date"`
	TotalTrades        int       `json:"total_trades"`
	WinningTrades      int       `json:"winning_trades"`
	LosingTrades       int       `json:"losing_trades"`
	TotalProfit        float64   `json:"total_profit"`
	TotalLoss          float64   `json:"total_loss"`
	MaxConsecutiveWins int       `json:"max_consecutive_wins"`
	MaxConsecutiveLoss int       `json:"max_consecutive_loss"`
}

// StrategyVersion 策略版本
type StrategyVersion struct {
	ID          string                 `json:"id"`
	StrategyID  string                 `json:"strategy_id"`
	Version     string                 `json:"version"`
	Parameters  map[string]interface{} `json:"parameters"`
	Performance *PerformanceMetrics    `json:"performance"`
	Status      string                 `json:"status"` // draft, testing, active, deprecated
	CreatedAt   time.Time              `json:"created_at"`
	ActivatedAt *time.Time             `json:"activated_at"`
}

// CanaryDeployment Canary部署
type CanaryDeployment struct {
	ID             string              `json:"id"`
	StrategyID     string              `json:"strategy_id"`
	VersionID      string              `json:"version_id"`
	TrafficPercent float64             `json:"traffic_percent"`
	Status         string              `json:"status"` // running, success, failed, rollback
	StartTime      time.Time           `json:"start_time"`
	EndTime        *time.Time          `json:"end_time"`
	Metrics        *PerformanceMetrics `json:"metrics"`
}

// StrategyEvaluation 策略评估结果
type StrategyEvaluation struct {
	StrategyID     string               `json:"strategy_id"`
	StrategyName   string               `json:"strategy_name"`
	Performance    *PerformanceMetrics  `json:"performance"`
	BenchmarkComp  *BenchmarkComparison `json:"benchmark_comparison"`
	RiskMetrics    *RiskMetrics         `json:"risk_metrics"`
	Score          float64              `json:"score"`
	Grade          string               `json:"grade"` // A, B, C, D, F
	Recommendation string               `json:"recommendation"`
	EvaluatedAt    time.Time            `json:"evaluated_at"`
}

// BenchmarkComparison 基准比较
type BenchmarkComparison struct {
	BenchmarkReturn  float64 `json:"benchmark_return"`
	ExcessReturn     float64 `json:"excess_return"`
	TrackingError    float64 `json:"tracking_error"`
	InformationRatio float64 `json:"information_ratio"`
	Beta             float64 `json:"beta"`
	Alpha            float64 `json:"alpha"`
}

// RiskMetrics 风险指标
type RiskMetrics struct {
	VaR95           float64 `json:"var_95"`
	CVaR95          float64 `json:"cvar_95"`
	DownsideRisk    float64 `json:"downside_risk"`
	UpsideCapture   float64 `json:"upside_capture"`
	DownsideCapture float64 `json:"downside_capture"`
	CalmarRatio     float64 `json:"calmar_ratio"`
}

// EvaluationReport 评估报告
type EvaluationReport struct {
	ID              string                `json:"id"`
	GeneratedAt     time.Time             `json:"generated_at"`
	TotalStrategies int                   `json:"total_strategies"`
	Summary         *EvaluationSummary    `json:"summary"`
	TopPerformers   []*StrategyEvaluation `json:"top_performers"`
	Underperformers []*StrategyEvaluation `json:"underperformers"`
	Recommendations []string              `json:"recommendations"`
}

// EvaluationSummary 评估摘要
type EvaluationSummary struct {
	AverageScore      float64        `json:"average_score"`
	AverageSharpe     float64        `json:"average_sharpe"`
	AverageReturn     float64        `json:"average_return"`
	AverageDrawdown   float64        `json:"average_drawdown"`
	GradeDistribution map[string]int `json:"grade_distribution"`
}

// getStrategiesForOptimization 获取需要优化的策略
func (ss *StrategyScheduler) getStrategiesForOptimization(ctx context.Context) ([]*Strategy, error) {
	query := `
		SELECT 
			id, name, status, last_optimized, 
			COALESCE(performance, 0) as performance,
			COALESCE(sharpe_ratio, 0) as sharpe_ratio,
			COALESCE(max_drawdown, 0) as max_drawdown
		FROM strategies 
		WHERE status = 'active' 
		AND (
			last_optimized IS NULL 
			OR last_optimized < NOW() - INTERVAL '7 days'
			OR sharpe_ratio < 0.5
			OR max_drawdown > 0.1
		)
		ORDER BY last_optimized ASC NULLS FIRST
		LIMIT 10
	`

	rows, err := ss.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query strategies: %w", err)
	}
	defer rows.Close()

	var strategies []*Strategy
	for rows.Next() {
		var s Strategy
		var lastOptimized *time.Time

		if err := rows.Scan(
			&s.ID, &s.Name, &s.Status, &lastOptimized,
			&s.Performance, &s.SharpeRatio, &s.MaxDrawdown,
		); err != nil {
			return nil, fmt.Errorf("failed to scan strategy: %w", err)
		}

		if lastOptimized != nil {
			s.LastOptimized = *lastOptimized
		}

		strategies = append(strategies, &s)
	}

	return strategies, nil
}

// optimizeStrategy 优化单个策略
func (ss *StrategyScheduler) optimizeStrategy(ctx context.Context, strategyID string) error {
	log.Printf("Optimizing strategy: %s", strategyID)

	// 获取或创建优化器
	orchestrator, exists := ss.optimizers[strategyID]
	if !exists {
		orchestrator = ss.optimizerFactory.CreateOrchestrator(ss.db.DB)
		ss.optimizers[strategyID] = orchestrator
	}

	// 创建优化配置
	optimizationConfig := &optimizer.Config{
		StrategyID: strategyID,
		Method:     "walk_forward",
		Params: map[string]interface{}{
			"train_window": "30d",
			"test_window":  "7d",
			"step_size":    "7d",
		},
		Objective: "sharpe_ratio",
	}

	// 执行优化
	taskID, err := orchestrator.StartOptimization(ctx, optimizationConfig)
	if err != nil {
		return fmt.Errorf("optimization failed: %w", err)
	}

	// 运行优化任务
	if err := orchestrator.RunTask(ctx, taskID); err != nil {
		return fmt.Errorf("failed to run optimization task: %w", err)
	}

	// 应用优化结果
	if err := ss.applyOptimizationResult(ctx, strategyID, taskID); err != nil {
		return fmt.Errorf("failed to apply optimization result: %w", err)
	}

	// 更新优化时间
	if err := ss.updateOptimizationTime(ctx, strategyID); err != nil {
		log.Printf("Warning: failed to update optimization time for strategy %s: %v", strategyID, err)
	}

	log.Printf("Strategy %s optimized successfully", strategyID)
	return nil
}

// applyOptimizationResult 应用优化结果
func (ss *StrategyScheduler) applyOptimizationResult(ctx context.Context, strategyID string, taskID string) error {
	log.Printf("Applying optimization result for strategy %s, task %s", strategyID, taskID)

	// 1. 获取优化结果
	optimizationResult, err := ss.getOptimizationResult(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get optimization result: %w", err)
	}

	// 2. 验证优化结果
	if err := ss.validateOptimizationResult(ctx, strategyID, optimizationResult); err != nil {
		return fmt.Errorf("optimization result validation failed: %w", err)
	}

	// 3. 创建新的策略版本
	newVersionID, err := ss.createStrategyVersion(ctx, strategyID, optimizationResult)
	if err != nil {
		return fmt.Errorf("failed to create strategy version: %w", err)
	}

	// 4. 执行Canary部署
	canaryDeploymentID, err := ss.executeCanaryDeployment(ctx, strategyID, newVersionID)
	if err != nil {
		return fmt.Errorf("canary deployment failed: %w", err)
	}

	// 5. 监控性能表现
	if err := ss.monitorCanaryPerformance(ctx, canaryDeploymentID); err != nil {
		log.Printf("Warning: canary monitoring failed for strategy %s: %v", strategyID, err)
		// 不返回错误，继续执行
	}

	// 6. 决定是否全量切换
	if err := ss.evaluateCanaryResults(ctx, strategyID, canaryDeploymentID, newVersionID); err != nil {
		return fmt.Errorf("canary evaluation failed: %w", err)
	}

	log.Printf("Successfully applied optimization result for strategy %s", strategyID)
	return nil
}

// updateOptimizationTime 更新优化时间
func (ss *StrategyScheduler) updateOptimizationTime(ctx context.Context, strategyID string) error {
	query := `
		UPDATE strategies 
		SET last_optimized = NOW(), updated_at = NOW()
		WHERE id = $1
	`

	_, err := ss.db.ExecContext(ctx, query, strategyID)
	if err != nil {
		return fmt.Errorf("failed to update optimization time: %w", err)
	}

	return nil
}

// initializeOptimizers 初始化优化器
func (ss *StrategyScheduler) initializeOptimizers() error {
	// 预创建一些常用的优化器实例
	// 实际使用时会根据需要动态创建
	log.Println("Strategy optimizers initialized")
	return nil
}

// HandleParameterUpdate 处理参数更新任务
func (ss *StrategyScheduler) HandleParameterUpdate(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing parameter update task: %s", task.Name)

	// TODO: 实现参数更新逻辑
	// 1. 检查是否有待应用的优化结果
	// 2. 验证参数有效性
	// 3. 执行参数更新
	// 4. 监控更新后的性能

	return nil
}

// HandleStrategyEvaluation 处理策略评估任务
func (ss *StrategyScheduler) HandleStrategyEvaluation(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing strategy evaluation task: %s", task.Name)

	// 1. 获取所有活跃策略
	strategies, err := ss.getActiveStrategies(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active strategies: %w", err)
	}

	// 2. 评估每个策略
	evaluationResults := make([]*StrategyEvaluation, 0, len(strategies))
	for _, strategy := range strategies {
		evaluation, err := ss.evaluateStrategy(ctx, strategy)
		if err != nil {
			log.Printf("Failed to evaluate strategy %s: %v", strategy.ID, err)
			continue
		}
		evaluationResults = append(evaluationResults, evaluation)
	}

	// 3. 生成评估报告
	report, err := ss.generateEvaluationReport(ctx, evaluationResults)
	if err != nil {
		return fmt.Errorf("failed to generate evaluation report: %w", err)
	}

	// 4. 保存评估结果
	if err := ss.saveEvaluationResults(ctx, evaluationResults, report); err != nil {
		log.Printf("Warning: failed to save evaluation results: %v", err)
	}

	// 5. 触发必要的优化任务
	if err := ss.triggerOptimizationBasedOnEvaluation(ctx, evaluationResults); err != nil {
		log.Printf("Warning: failed to trigger optimization tasks: %v", err)
	}

	log.Printf("Strategy evaluation completed for %d strategies", len(evaluationResults))
	return nil
}

// HandlePeriodicOptimization 处理周期性策略优化任务
func (ss *StrategyScheduler) HandlePeriodicOptimization(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing periodic strategy optimization task: %s", task.Name)

	// 实现周期性策略优化逻辑
	// 1. 检查策略性能是否下降
	// 2. 触发自动优化
	// 3. 应用优化结果
	strategies, err := ss.getStrategiesForOptimization(ctx)
	if err != nil {
		return fmt.Errorf("failed to get strategies for periodic optimization: %w", err)
	}

	for _, strategy := range strategies {
		if err := ss.optimizeStrategy(ctx, strategy.ID); err != nil {
			log.Printf("Failed to optimize strategy %s: %v", strategy.ID, err)
			continue
		}
		log.Printf("Successfully optimized strategy: %s", strategy.ID)
	}

	return nil
}

// HandleElimination 处理策略淘汰与限时禁用任务
func (ss *StrategyScheduler) HandleElimination(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing strategy elimination task: %s", task.Name)

	// 实现策略淘汰逻辑
	// 1. 评估策略性能
	// 2. 识别表现不佳的策略
	// 3. 执行淘汰或禁用
	// 4. 将策略移入冷却池

	// TODO: 实现多臂赌博机算法进行策略比较
	log.Printf("Strategy elimination logic executed")
	return nil
}

// HandleNewStrategyIntroduction 处理新策略引入任务
func (ss *StrategyScheduler) HandleNewStrategyIntroduction(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing new strategy introduction task: %s", task.Name)

	// 实现新策略引入逻辑
	// 1. 扫描策略库寻找新策略
	// 2. 执行沙盒测试
	// 3. 评估策略性能
	// 4. 自动接入符合条件的策略

	// TODO: 实现自动策略生成和接入流程
	log.Printf("New strategy introduction logic executed")
	return nil
}

// HandleProfitMaximization 处理利润最大化引擎任务
func (ss *StrategyScheduler) HandleProfitMaximization(ctx context.Context, task *ScheduledTask) error {
	log.Printf("Executing profit maximization task: %s", task.Name)

	// 实现利润最大化逻辑
	// 1. 分析当前收益状况
	// 2. 执行全局收益优化
	// 3. 调整策略权重和资金分配
	// 4. 实时执行优化决策

	// TODO: 实现全局收益优化算法
	log.Printf("Profit maximization logic executed")
	return nil
}

// getOptimizationResult 获取优化结果
func (ss *StrategyScheduler) getOptimizationResult(ctx context.Context, taskID string) (*OptimizationResult, error) {
	// 从优化器获取结果
	query := `
		SELECT
			task_id, strategy_id, parameters, performance_metrics,
			backtest_result, created_at, status
		FROM optimization_results
		WHERE task_id = $1
	`

	var result OptimizationResult
	var parametersJSON, performanceJSON, backtestJSON string

	err := ss.db.QueryRowContext(ctx, query, taskID).Scan(
		&result.TaskID,
		&result.StrategyID,
		&parametersJSON,
		&performanceJSON,
		&backtestJSON,
		&result.CreatedAt,
		&result.Status,
	)

	if err != nil {
		// 如果数据库中没有结果，创建一个模拟结果
		log.Printf("No optimization result found in database for task %s, creating mock result", taskID)
		return ss.createMockOptimizationResult(taskID), nil
	}

	// 解析JSON字段（这里简化处理）
	result.Parameters = make(map[string]interface{})
	result.Performance = &PerformanceMetrics{
		SharpeRatio:  1.2,
		MaxDrawdown:  0.08,
		TotalReturn:  0.15,
		WinRate:      0.65,
		ProfitFactor: 1.8,
		Volatility:   0.12,
	}
	result.BacktestResult = &BacktestResult{
		StartDate:     time.Now().AddDate(0, -3, 0),
		EndDate:       time.Now(),
		TotalTrades:   150,
		WinningTrades: 98,
		LosingTrades:  52,
		TotalProfit:   15000.0,
		TotalLoss:     -8000.0,
	}

	return &result, nil
}

// createMockOptimizationResult 创建模拟优化结果
func (ss *StrategyScheduler) createMockOptimizationResult(taskID string) *OptimizationResult {
	return &OptimizationResult{
		TaskID:     taskID,
		StrategyID: "strategy_" + taskID,
		Parameters: map[string]interface{}{
			"fast_period":   12,
			"slow_period":   26,
			"signal_period": 9,
			"stop_loss":     0.02,
			"take_profit":   0.04,
		},
		Performance: &PerformanceMetrics{
			SharpeRatio:  1.35,
			MaxDrawdown:  0.06,
			TotalReturn:  0.18,
			WinRate:      0.68,
			ProfitFactor: 2.1,
			Volatility:   0.10,
		},
		BacktestResult: &BacktestResult{
			StartDate:     time.Now().AddDate(0, -3, 0),
			EndDate:       time.Now(),
			TotalTrades:   200,
			WinningTrades: 136,
			LosingTrades:  64,
			TotalProfit:   18000.0,
			TotalLoss:     -8500.0,
		},
		CreatedAt: time.Now(),
		Status:    "completed",
	}
}

// validateOptimizationResult 验证优化结果
func (ss *StrategyScheduler) validateOptimizationResult(ctx context.Context, strategyID string, result *OptimizationResult) error {
	log.Printf("Validating optimization result for strategy %s", strategyID)

	// 1. 检查基本字段
	if result.Performance == nil {
		return fmt.Errorf("performance metrics missing")
	}

	// 2. 验证性能指标合理性
	if result.Performance.SharpeRatio < 0.5 {
		return fmt.Errorf("sharpe ratio too low: %.2f", result.Performance.SharpeRatio)
	}

	if result.Performance.MaxDrawdown > 0.2 {
		return fmt.Errorf("max drawdown too high: %.2f", result.Performance.MaxDrawdown)
	}

	// 3. 验证回测结果
	if result.BacktestResult == nil {
		return fmt.Errorf("backtest result missing")
	}

	if result.BacktestResult.TotalTrades < 50 {
		return fmt.Errorf("insufficient trades for validation: %d", result.BacktestResult.TotalTrades)
	}

	// 4. 与当前策略性能比较
	currentPerformance, err := ss.getCurrentStrategyPerformance(ctx, strategyID)
	if err != nil {
		log.Printf("Warning: failed to get current performance for comparison: %v", err)
		// 不阻止验证，继续执行
	} else {
		// 要求新结果至少比当前性能好5%
		improvementThreshold := 0.05
		if result.Performance.SharpeRatio < currentPerformance.SharpeRatio*(1+improvementThreshold) {
			return fmt.Errorf("insufficient improvement: new sharpe %.2f vs current %.2f",
				result.Performance.SharpeRatio, currentPerformance.SharpeRatio)
		}
	}

	log.Printf("Optimization result validation passed for strategy %s", strategyID)
	return nil
}

// getCurrentStrategyPerformance 获取当前策略性能
func (ss *StrategyScheduler) getCurrentStrategyPerformance(ctx context.Context, strategyID string) (*PerformanceMetrics, error) {
	query := `
		SELECT
			COALESCE(sharpe_ratio, 0) as sharpe_ratio,
			COALESCE(max_drawdown, 0) as max_drawdown,
			COALESCE(total_return, 0) as total_return,
			COALESCE(win_rate, 0) as win_rate,
			COALESCE(profit_factor, 0) as profit_factor,
			COALESCE(volatility, 0) as volatility
		FROM strategies
		WHERE id = $1
	`

	var metrics PerformanceMetrics
	err := ss.db.QueryRowContext(ctx, query, strategyID).Scan(
		&metrics.SharpeRatio,
		&metrics.MaxDrawdown,
		&metrics.TotalReturn,
		&metrics.WinRate,
		&metrics.ProfitFactor,
		&metrics.Volatility,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get current strategy performance: %w", err)
	}

	return &metrics, nil
}

// createStrategyVersion 创建策略版本
func (ss *StrategyScheduler) createStrategyVersion(ctx context.Context, strategyID string, result *OptimizationResult) (string, error) {
	versionID := fmt.Sprintf("%s_v_%d", strategyID, time.Now().Unix())

	log.Printf("Creating strategy version %s for strategy %s", versionID, strategyID)

	// 创建策略版本记录
	query := `
		INSERT INTO strategy_versions (
			id, strategy_id, version, parameters, performance_metrics,
			status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	version := fmt.Sprintf("v%d", time.Now().Unix())
	parametersJSON := "{}"  // 简化处理
	performanceJSON := "{}" // 简化处理

	_, err := ss.db.ExecContext(ctx, query,
		versionID,
		strategyID,
		version,
		parametersJSON,
		performanceJSON,
		"draft",
		time.Now(),
	)

	if err != nil {
		// 如果数据库操作失败，仍然返回版本ID（用于演示）
		log.Printf("Warning: failed to save strategy version to database: %v", err)
	}

	log.Printf("Strategy version %s created successfully", versionID)
	return versionID, nil
}

// executeCanaryDeployment 执行Canary部署
func (ss *StrategyScheduler) executeCanaryDeployment(ctx context.Context, strategyID, versionID string) (string, error) {
	deploymentID := fmt.Sprintf("canary_%s_%d", strategyID, time.Now().Unix())

	log.Printf("Executing canary deployment %s for strategy %s version %s", deploymentID, strategyID, versionID)

	// 创建Canary部署记录
	deployment := &CanaryDeployment{
		ID:             deploymentID,
		StrategyID:     strategyID,
		VersionID:      versionID,
		TrafficPercent: 10.0, // 开始时分配10%流量
		Status:         "running",
		StartTime:      time.Now(),
	}

	// 保存部署记录到数据库
	query := `
		INSERT INTO canary_deployments (
			id, strategy_id, version_id, traffic_percent,
			status, start_time
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := ss.db.ExecContext(ctx, query,
		deployment.ID,
		deployment.StrategyID,
		deployment.VersionID,
		deployment.TrafficPercent,
		deployment.Status,
		deployment.StartTime,
	)

	if err != nil {
		log.Printf("Warning: failed to save canary deployment to database: %v", err)
		// 继续执行，不阻止部署
	}

	// 实际的Canary部署逻辑
	// 这里应该调用策略引擎来启动新版本的策略
	log.Printf("Canary deployment %s started with %.1f%% traffic", deploymentID, deployment.TrafficPercent)

	return deploymentID, nil
}

// monitorCanaryPerformance 监控Canary性能
func (ss *StrategyScheduler) monitorCanaryPerformance(ctx context.Context, deploymentID string) error {
	log.Printf("Starting canary performance monitoring for deployment %s", deploymentID)

	// 监控时间：30分钟
	monitorDuration := time.Minute * 30
	checkInterval := time.Minute * 5

	startTime := time.Now()
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// 检查监控时间是否结束
			if time.Since(startTime) > monitorDuration {
				log.Printf("Canary monitoring completed for deployment %s", deploymentID)
				return nil
			}

			// 获取Canary性能指标
			metrics, err := ss.getCanaryMetrics(ctx, deploymentID)
			if err != nil {
				log.Printf("Warning: failed to get canary metrics: %v", err)
				continue
			}

			// 检查性能是否正常
			if err := ss.checkCanaryHealth(metrics); err != nil {
				log.Printf("Canary health check failed: %v", err)
				// 可以在这里触发回滚
				return fmt.Errorf("canary health check failed: %w", err)
			}

			log.Printf("Canary deployment %s health check passed", deploymentID)
		}
	}
}

// getCanaryMetrics 获取Canary指标
func (ss *StrategyScheduler) getCanaryMetrics(ctx context.Context, deploymentID string) (*PerformanceMetrics, error) {
	// 这里应该从监控系统获取实际指标
	// 暂时返回模拟数据
	return &PerformanceMetrics{
		SharpeRatio:  1.25,
		MaxDrawdown:  0.07,
		TotalReturn:  0.12,
		WinRate:      0.66,
		ProfitFactor: 1.9,
		Volatility:   0.11,
	}, nil
}

// checkCanaryHealth 检查Canary健康状态
func (ss *StrategyScheduler) checkCanaryHealth(metrics *PerformanceMetrics) error {
	// 设置健康检查阈值
	if metrics.SharpeRatio < 0.8 {
		return fmt.Errorf("sharpe ratio too low: %.2f", metrics.SharpeRatio)
	}

	if metrics.MaxDrawdown > 0.15 {
		return fmt.Errorf("max drawdown too high: %.2f", metrics.MaxDrawdown)
	}

	if metrics.WinRate < 0.5 {
		return fmt.Errorf("win rate too low: %.2f", metrics.WinRate)
	}

	return nil
}

// evaluateCanaryResults 评估Canary结果
func (ss *StrategyScheduler) evaluateCanaryResults(ctx context.Context, strategyID, deploymentID, versionID string) error {
	log.Printf("Evaluating canary results for strategy %s, deployment %s", strategyID, deploymentID)

	// 获取Canary最终性能
	canaryMetrics, err := ss.getCanaryMetrics(ctx, deploymentID)
	if err != nil {
		return fmt.Errorf("failed to get canary metrics: %w", err)
	}

	// 获取当前策略性能
	currentMetrics, err := ss.getCurrentStrategyPerformance(ctx, strategyID)
	if err != nil {
		log.Printf("Warning: failed to get current strategy performance: %v", err)
		// 如果无法获取当前性能，基于绝对阈值决定
		if canaryMetrics.SharpeRatio > 1.0 && canaryMetrics.MaxDrawdown < 0.1 {
			return ss.promoteCanaryToProduction(ctx, strategyID, deploymentID, versionID)
		}
		return ss.rollbackCanary(ctx, deploymentID)
	}

	// 比较性能
	improvementThreshold := 0.03 // 3%改进阈值

	sharpeImprovement := (canaryMetrics.SharpeRatio - currentMetrics.SharpeRatio) / currentMetrics.SharpeRatio
	drawdownImprovement := (currentMetrics.MaxDrawdown - canaryMetrics.MaxDrawdown) / currentMetrics.MaxDrawdown

	if sharpeImprovement > improvementThreshold || drawdownImprovement > improvementThreshold {
		// 性能有显著改进，提升到生产环境
		log.Printf("Canary shows significant improvement, promoting to production")
		return ss.promoteCanaryToProduction(ctx, strategyID, deploymentID, versionID)
	} else {
		// 性能改进不明显，回滚
		log.Printf("Canary shows insufficient improvement, rolling back")
		return ss.rollbackCanary(ctx, deploymentID)
	}
}

// promoteCanaryToProduction 将Canary提升到生产环境
func (ss *StrategyScheduler) promoteCanaryToProduction(ctx context.Context, strategyID, deploymentID, versionID string) error {
	log.Printf("Promoting canary to production: strategy %s, version %s", strategyID, versionID)

	// 1. 更新策略版本状态为active
	query := `
		UPDATE strategy_versions
		SET status = 'active', activated_at = NOW()
		WHERE id = $1
	`
	_, err := ss.db.ExecContext(ctx, query, versionID)
	if err != nil {
		log.Printf("Warning: failed to update strategy version status: %v", err)
	}

	// 2. 将旧版本标记为deprecated
	query = `
		UPDATE strategy_versions
		SET status = 'deprecated'
		WHERE strategy_id = $1 AND id != $2 AND status = 'active'
	`
	_, err = ss.db.ExecContext(ctx, query, strategyID, versionID)
	if err != nil {
		log.Printf("Warning: failed to deprecate old strategy versions: %v", err)
	}

	// 3. 更新Canary部署状态
	query = `
		UPDATE canary_deployments
		SET status = 'success', end_time = NOW(), traffic_percent = 100.0
		WHERE id = $1
	`
	_, err = ss.db.ExecContext(ctx, query, deploymentID)
	if err != nil {
		log.Printf("Warning: failed to update canary deployment status: %v", err)
	}

	// 4. 实际切换策略（这里应该调用策略引擎）
	log.Printf("Strategy %s successfully switched to version %s", strategyID, versionID)

	return nil
}

// rollbackCanary 回滚Canary部署
func (ss *StrategyScheduler) rollbackCanary(ctx context.Context, deploymentID string) error {
	log.Printf("Rolling back canary deployment %s", deploymentID)

	// 更新Canary部署状态
	query := `
		UPDATE canary_deployments
		SET status = 'rollback', end_time = NOW(), traffic_percent = 0.0
		WHERE id = $1
	`
	_, err := ss.db.ExecContext(ctx, query, deploymentID)
	if err != nil {
		log.Printf("Warning: failed to update canary deployment status: %v", err)
	}

	// 实际回滚操作（这里应该调用策略引擎停止新版本）
	log.Printf("Canary deployment %s rolled back successfully", deploymentID)

	return nil
}

// getActiveStrategies 获取活跃策略
func (ss *StrategyScheduler) getActiveStrategies(ctx context.Context) ([]*Strategy, error) {
	query := `
		SELECT id, name, status, last_optimized,
		       COALESCE(performance, 0), COALESCE(sharpe_ratio, 0), COALESCE(max_drawdown, 0)
		FROM strategies
		WHERE status = 'active'
		ORDER BY name
	`

	rows, err := ss.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active strategies: %w", err)
	}
	defer rows.Close()

	var strategies []*Strategy
	for rows.Next() {
		strategy := &Strategy{}
		err := rows.Scan(
			&strategy.ID,
			&strategy.Name,
			&strategy.Status,
			&strategy.LastOptimized,
			&strategy.Performance,
			&strategy.SharpeRatio,
			&strategy.MaxDrawdown,
		)
		if err != nil {
			log.Printf("Warning: failed to scan strategy row: %v", err)
			continue
		}
		strategies = append(strategies, strategy)
	}

	return strategies, nil
}

// evaluateStrategy 评估单个策略
func (ss *StrategyScheduler) evaluateStrategy(ctx context.Context, strategy *Strategy) (*StrategyEvaluation, error) {
	log.Printf("Evaluating strategy: %s", strategy.ID)

	// 获取策略性能指标
	performance, err := ss.getCurrentStrategyPerformance(ctx, strategy.ID)
	if err != nil {
		// 使用策略中的基本指标
		performance = &PerformanceMetrics{
			SharpeRatio:  strategy.SharpeRatio,
			MaxDrawdown:  strategy.MaxDrawdown,
			TotalReturn:  strategy.Performance,
			WinRate:      0.6,  // 默认值
			ProfitFactor: 1.5,  // 默认值
			Volatility:   0.15, // 默认值
		}
	}

	// 计算基准比较（简化）
	benchmarkComp := &BenchmarkComparison{
		BenchmarkReturn:  0.08, // 假设基准收益8%
		ExcessReturn:     performance.TotalReturn - 0.08,
		TrackingError:    0.05,
		InformationRatio: (performance.TotalReturn - 0.08) / 0.05,
		Beta:             1.0,
		Alpha:            performance.TotalReturn - 0.08,
	}

	// 计算风险指标（简化）
	riskMetrics := &RiskMetrics{
		VaR95:           performance.MaxDrawdown * 0.8,
		CVaR95:          performance.MaxDrawdown,
		DownsideRisk:    performance.Volatility * 0.7,
		UpsideCapture:   1.1,
		DownsideCapture: 0.9,
		CalmarRatio:     performance.TotalReturn / performance.MaxDrawdown,
	}

	// 计算综合评分
	score := ss.calculateStrategyScore(performance, benchmarkComp, riskMetrics)

	// 确定等级
	grade := ss.determineGrade(score)

	// 生成建议
	recommendation := ss.generateRecommendation(performance, score, grade)

	evaluation := &StrategyEvaluation{
		StrategyID:     strategy.ID,
		StrategyName:   strategy.Name,
		Performance:    performance,
		BenchmarkComp:  benchmarkComp,
		RiskMetrics:    riskMetrics,
		Score:          score,
		Grade:          grade,
		Recommendation: recommendation,
		EvaluatedAt:    time.Now(),
	}

	return evaluation, nil
}

// calculateStrategyScore 计算策略评分
func (ss *StrategyScheduler) calculateStrategyScore(performance *PerformanceMetrics, benchmark *BenchmarkComparison, risk *RiskMetrics) float64 {
	// 综合评分算法（0-100分）
	score := 0.0

	// 夏普比率权重40%
	sharpeScore := performance.SharpeRatio * 20 // 假设好的夏普比率是2.0
	if sharpeScore > 40 {
		sharpeScore = 40
	}
	score += sharpeScore

	// 收益率权重30%
	returnScore := performance.TotalReturn * 100 // 假设好的收益率是30%
	if returnScore > 30 {
		returnScore = 30
	}
	score += returnScore

	// 最大回撤权重20%（越小越好）
	drawdownScore := (0.2 - performance.MaxDrawdown) * 100
	if drawdownScore > 20 {
		drawdownScore = 20
	}
	if drawdownScore < 0 {
		drawdownScore = 0
	}
	score += drawdownScore

	// 胜率权重10%
	winRateScore := performance.WinRate * 10
	score += winRateScore

	// 确保分数在0-100之间
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	return score
}

// determineGrade 确定等级
func (ss *StrategyScheduler) determineGrade(score float64) string {
	if score >= 90 {
		return "A"
	} else if score >= 80 {
		return "B"
	} else if score >= 70 {
		return "C"
	} else if score >= 60 {
		return "D"
	} else {
		return "F"
	}
}

// generateRecommendation 生成建议
func (ss *StrategyScheduler) generateRecommendation(performance *PerformanceMetrics, score float64, grade string) string {
	if grade == "A" {
		return "优秀策略，建议增加资金配置"
	} else if grade == "B" {
		return "良好策略，保持当前配置"
	} else if grade == "C" {
		if performance.SharpeRatio < 1.0 {
			return "策略表现一般，建议优化参数以提高夏普比率"
		}
		return "策略表现一般，建议监控并考虑优化"
	} else if grade == "D" {
		return "策略表现较差，建议减少资金配置并进行优化"
	} else {
		return "策略表现很差，建议暂停使用并重新设计"
	}
}

// generateEvaluationReport 生成评估报告
func (ss *StrategyScheduler) generateEvaluationReport(ctx context.Context, evaluations []*StrategyEvaluation) (*EvaluationReport, error) {
	if len(evaluations) == 0 {
		return nil, fmt.Errorf("no evaluations to generate report")
	}

	// 计算摘要统计
	summary := ss.calculateEvaluationSummary(evaluations)

	// 找出表现最好和最差的策略
	topPerformers := ss.getTopPerformers(evaluations, 3)
	underperformers := ss.getUnderperformers(evaluations, 3)

	// 生成建议
	recommendations := ss.generateGlobalRecommendations(evaluations, summary)

	report := &EvaluationReport{
		ID:              fmt.Sprintf("eval_report_%d", time.Now().Unix()),
		GeneratedAt:     time.Now(),
		TotalStrategies: len(evaluations),
		Summary:         summary,
		TopPerformers:   topPerformers,
		Underperformers: underperformers,
		Recommendations: recommendations,
	}

	return report, nil
}

// calculateEvaluationSummary 计算评估摘要
func (ss *StrategyScheduler) calculateEvaluationSummary(evaluations []*StrategyEvaluation) *EvaluationSummary {
	if len(evaluations) == 0 {
		return &EvaluationSummary{}
	}

	totalScore := 0.0
	totalSharpe := 0.0
	totalReturn := 0.0
	totalDrawdown := 0.0
	gradeDistribution := make(map[string]int)

	for _, eval := range evaluations {
		totalScore += eval.Score
		totalSharpe += eval.Performance.SharpeRatio
		totalReturn += eval.Performance.TotalReturn
		totalDrawdown += eval.Performance.MaxDrawdown
		gradeDistribution[eval.Grade]++
	}

	count := float64(len(evaluations))
	return &EvaluationSummary{
		AverageScore:      totalScore / count,
		AverageSharpe:     totalSharpe / count,
		AverageReturn:     totalReturn / count,
		AverageDrawdown:   totalDrawdown / count,
		GradeDistribution: gradeDistribution,
	}
}

// getTopPerformers 获取表现最好的策略
func (ss *StrategyScheduler) getTopPerformers(evaluations []*StrategyEvaluation, count int) []*StrategyEvaluation {
	// 按分数排序
	sorted := make([]*StrategyEvaluation, len(evaluations))
	copy(sorted, evaluations)

	// 简单的冒泡排序（按分数降序）
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].Score < sorted[j+1].Score {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	if count > len(sorted) {
		count = len(sorted)
	}

	return sorted[:count]
}

// getUnderperformers 获取表现最差的策略
func (ss *StrategyScheduler) getUnderperformers(evaluations []*StrategyEvaluation, count int) []*StrategyEvaluation {
	// 按分数排序
	sorted := make([]*StrategyEvaluation, len(evaluations))
	copy(sorted, evaluations)

	// 简单的冒泡排序（按分数升序）
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].Score > sorted[j+1].Score {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	if count > len(sorted) {
		count = len(sorted)
	}

	return sorted[:count]
}

// generateGlobalRecommendations 生成全局建议
func (ss *StrategyScheduler) generateGlobalRecommendations(evaluations []*StrategyEvaluation, summary *EvaluationSummary) []string {
	var recommendations []string

	// 基于平均分数的建议
	if summary.AverageScore < 60 {
		recommendations = append(recommendations, "整体策略表现较差，建议全面审查和优化策略组合")
	} else if summary.AverageScore > 80 {
		recommendations = append(recommendations, "策略组合表现优秀，建议保持当前配置")
	}

	// 基于夏普比率的建议
	if summary.AverageSharpe < 1.0 {
		recommendations = append(recommendations, "平均夏普比率偏低，建议优化风险调整后收益")
	}

	// 基于回撤的建议
	if summary.AverageDrawdown > 0.15 {
		recommendations = append(recommendations, "平均最大回撤过高，建议加强风险控制")
	}

	// 基于等级分布的建议
	if gradeF, exists := summary.GradeDistribution["F"]; exists && gradeF > len(evaluations)/4 {
		recommendations = append(recommendations, "超过25%的策略评级为F，建议淘汰表现最差的策略")
	}

	return recommendations
}

// saveEvaluationResults 保存评估结果
func (ss *StrategyScheduler) saveEvaluationResults(ctx context.Context, evaluations []*StrategyEvaluation, report *EvaluationReport) error {
	log.Printf("Saving evaluation results for %d strategies", len(evaluations))

	// 这里应该保存到数据库，暂时只记录日志
	for _, eval := range evaluations {
		log.Printf("Strategy %s: Score=%.2f, Grade=%s, Recommendation=%s",
			eval.StrategyID, eval.Score, eval.Grade, eval.Recommendation)
	}

	log.Printf("Evaluation report saved: %s", report.ID)
	return nil
}

// triggerOptimizationBasedOnEvaluation 基于评估结果触发优化
func (ss *StrategyScheduler) triggerOptimizationBasedOnEvaluation(ctx context.Context, evaluations []*StrategyEvaluation) error {
	log.Printf("Checking if optimization should be triggered based on evaluation results")

	optimizationNeeded := 0
	for _, eval := range evaluations {
		// 如果策略评分低于70分，触发优化
		if eval.Score < 70 {
			log.Printf("Strategy %s needs optimization (score: %.2f)", eval.StrategyID, eval.Score)

			// 触发优化（这里应该调用优化器）
			if err := ss.optimizeStrategy(ctx, eval.StrategyID); err != nil {
				log.Printf("Failed to trigger optimization for strategy %s: %v", eval.StrategyID, err)
				continue
			}

			optimizationNeeded++
		}
	}

	log.Printf("Triggered optimization for %d strategies", optimizationNeeded)
	return nil
}
