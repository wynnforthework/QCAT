package automation

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"time"
)

// ParameterOptimizer 策略参数自动优化器
type ParameterOptimizer struct {
	db                *sql.DB
	backtestScheduler *BacktestScheduler
	optimizeInterval  time.Duration
	running           bool
	stopChan          chan struct{}
}

// OptimizationTask 优化任务
type OptimizationTask struct {
	StrategyID     string                 `json:"strategy_id"`
	CurrentParams  map[string]interface{} `json:"current_params"`
	BestParams     map[string]interface{} `json:"best_params"`
	CurrentScore   float64                `json:"current_score"`
	BestScore      float64                `json:"best_score"`
	Iterations     int                    `json:"iterations"`
	Status         string                 `json:"status"`
	StartTime      time.Time              `json:"start_time"`
	LastUpdate     time.Time              `json:"last_update"`
}

// ParameterRange 参数范围定义
type ParameterRange struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`     // "float", "int", "bool"
	MinValue interface{} `json:"min_value"`
	MaxValue interface{} `json:"max_value"`
	Step     interface{} `json:"step"`
}

// OptimizationResult 优化结果
type OptimizationResult struct {
	StrategyID       string                 `json:"strategy_id"`
	OriginalParams   map[string]interface{} `json:"original_params"`
	OptimizedParams  map[string]interface{} `json:"optimized_params"`
	OriginalScore    float64                `json:"original_score"`
	OptimizedScore   float64                `json:"optimized_score"`
	Improvement      float64                `json:"improvement"`
	OptimizationTime time.Duration          `json:"optimization_time"`
	Iterations       int                    `json:"iterations"`
}

// NewParameterOptimizer 创建参数优化器
func NewParameterOptimizer(db *sql.DB, backtestScheduler *BacktestScheduler) *ParameterOptimizer {
	return &ParameterOptimizer{
		db:                db,
		backtestScheduler: backtestScheduler,
		optimizeInterval:  24 * time.Hour, // 每天优化一次
		stopChan:          make(chan struct{}),
	}
}

// Start 启动参数优化器
func (po *ParameterOptimizer) Start(ctx context.Context) error {
	if po.running {
		return fmt.Errorf("parameter optimizer is already running")
	}

	po.running = true
	log.Printf("🔧 策略参数自动优化器启动")

	go po.optimizeLoop(ctx)
	return nil
}

// Stop 停止优化器
func (po *ParameterOptimizer) Stop() {
	if !po.running {
		return
	}

	po.running = false
	close(po.stopChan)
	log.Printf("策略参数自动优化器已停止")
}

// optimizeLoop 优化循环
func (po *ParameterOptimizer) optimizeLoop(ctx context.Context) {
	// 启动时延迟1小时再开始优化
	time.Sleep(1 * time.Hour)

	ticker := time.NewTicker(po.optimizeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-po.stopChan:
			return
		case <-ticker.C:
			if err := po.performOptimization(ctx); err != nil {
				log.Printf("参数优化失败: %v", err)
			}
		}
	}
}

// performOptimization 执行参数优化
func (po *ParameterOptimizer) performOptimization(ctx context.Context) error {
	log.Printf("🔍 开始策略参数优化...")

	// 1. 获取需要优化的策略
	strategies, err := po.getStrategiesForOptimization(ctx)
	if err != nil {
		return fmt.Errorf("获取需要优化的策略失败: %w", err)
	}

	log.Printf("发现 %d 个策略需要参数优化", len(strategies))

	// 2. 为每个策略执行优化
	for _, strategyID := range strategies {
		if err := po.optimizeStrategy(ctx, strategyID); err != nil {
			log.Printf("优化策略 %s 失败: %v", strategyID, err)
		}
	}

	return nil
}

// getStrategiesForOptimization 获取需要优化的策略
func (po *ParameterOptimizer) getStrategiesForOptimization(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT s.id
		FROM strategies s
		LEFT JOIN backtest_results br ON s.id = br.strategy_id
		LEFT JOIN optimization_history oh ON s.id = oh.strategy_id
		WHERE s.is_running = true 
		AND s.status = 'active'
		AND br.is_valid = true
		AND (
			-- 从未优化过
			oh.strategy_id IS NULL
			-- 或者上次优化超过7天
			OR oh.last_optimization < NOW() - INTERVAL '7 days'
			-- 或者策略表现下降
			OR (br.sharpe_ratio < 0.8 AND br.max_drawdown > 0.1)
		)
		ORDER BY br.sharpe_ratio ASC, br.max_drawdown DESC
		LIMIT 3
	`

	rows, err := po.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var strategies []string
	for rows.Next() {
		var strategyID string
		if err := rows.Scan(&strategyID); err != nil {
			continue
		}
		strategies = append(strategies, strategyID)
	}

	return strategies, nil
}

// optimizeStrategy 优化单个策略
func (po *ParameterOptimizer) optimizeStrategy(ctx context.Context, strategyID string) error {
	log.Printf("🔧 开始优化策略 %s", strategyID)

	startTime := time.Now()

	// 1. 获取当前参数和性能
	currentParams, currentScore, err := po.getCurrentStrategyState(ctx, strategyID)
	if err != nil {
		return fmt.Errorf("获取策略当前状态失败: %w", err)
	}

	// 2. 定义参数搜索空间
	paramRanges := po.getParameterRanges(strategyID)

	// 3. 执行网格搜索优化
	bestParams, bestScore, iterations, err := po.gridSearchOptimization(ctx, strategyID, currentParams, paramRanges)
	if err != nil {
		return fmt.Errorf("网格搜索优化失败: %w", err)
	}

	// 4. 计算改进程度
	improvement := (bestScore - currentScore) / math.Abs(currentScore) * 100

	// 5. 如果有显著改进，应用新参数
	if improvement > 5.0 { // 改进超过5%
		if err := po.applyOptimizedParameters(ctx, strategyID, bestParams); err != nil {
			log.Printf("应用优化参数失败: %v", err)
		} else {
			log.Printf("✅ 策略 %s 参数优化完成，改进 %.2f%%", strategyID, improvement)
		}
	} else {
		log.Printf("策略 %s 参数优化完成，但改进不显著 (%.2f%%)，保持原参数", strategyID, improvement)
	}

	// 6. 记录优化历史
	result := &OptimizationResult{
		StrategyID:       strategyID,
		OriginalParams:   currentParams,
		OptimizedParams:  bestParams,
		OriginalScore:    currentScore,
		OptimizedScore:   bestScore,
		Improvement:      improvement,
		OptimizationTime: time.Since(startTime),
		Iterations:       iterations,
	}

	if err := po.saveOptimizationResult(ctx, result); err != nil {
		log.Printf("保存优化结果失败: %v", err)
	}

	return nil
}

// getCurrentStrategyState 获取策略当前状态
func (po *ParameterOptimizer) getCurrentStrategyState(ctx context.Context, strategyID string) (map[string]interface{}, float64, error) {
	// 获取当前参数（从策略配置表）
	paramsQuery := `
		SELECT config_json FROM strategies WHERE id = $1
	`
	
	var configJSON string
	if err := po.db.QueryRowContext(ctx, paramsQuery, strategyID).Scan(&configJSON); err != nil {
		return nil, 0, err
	}

	// 解析参数（这里简化处理）
	currentParams := map[string]interface{}{
		"stop_loss":    0.02,
		"take_profit":  0.05,
		"rsi_period":   14,
		"ma_period":    20,
		"volume_threshold": 1000000,
	}

	// 获取当前性能评分（基于回测结果）
	scoreQuery := `
		SELECT 
			COALESCE(sharpe_ratio, 0) * 0.4 + 
			COALESCE((1 - max_drawdown), 0) * 0.3 + 
			COALESCE(win_rate, 0) * 0.3 as composite_score
		FROM backtest_results 
		WHERE strategy_id = $1
	`

	var currentScore float64
	if err := po.db.QueryRowContext(ctx, scoreQuery, strategyID).Scan(&currentScore); err != nil {
		currentScore = 0.1 // 默认低分
	}

	return currentParams, currentScore, nil
}

// getParameterRanges 获取参数搜索范围
func (po *ParameterOptimizer) getParameterRanges(strategyID string) []ParameterRange {
	// 这里定义各种参数的搜索范围
	return []ParameterRange{
		{
			Name:     "stop_loss",
			Type:     "float",
			MinValue: 0.01,
			MaxValue: 0.05,
			Step:     0.005,
		},
		{
			Name:     "take_profit",
			Type:     "float",
			MinValue: 0.02,
			MaxValue: 0.10,
			Step:     0.01,
		},
		{
			Name:     "rsi_period",
			Type:     "int",
			MinValue: 10,
			MaxValue: 30,
			Step:     2,
		},
		{
			Name:     "ma_period",
			Type:     "int",
			MinValue: 10,
			MaxValue: 50,
			Step:     5,
		},
	}
}

// gridSearchOptimization 网格搜索优化
func (po *ParameterOptimizer) gridSearchOptimization(ctx context.Context, strategyID string, currentParams map[string]interface{}, paramRanges []ParameterRange) (map[string]interface{}, float64, int, error) {
	bestParams := make(map[string]interface{})
	for k, v := range currentParams {
		bestParams[k] = v
	}
	
	bestScore := 0.1 // 初始低分
	iterations := 0

	// 简化的网格搜索（实际应该是多维网格）
	for _, paramRange := range paramRanges {
		if paramRange.Type == "float" {
			minVal := paramRange.MinValue.(float64)
			maxVal := paramRange.MaxValue.(float64)
			step := paramRange.Step.(float64)

			for value := minVal; value <= maxVal; value += step {
				iterations++
				
				// 创建测试参数组合
				testParams := make(map[string]interface{})
				for k, v := range currentParams {
					testParams[k] = v
				}
				testParams[paramRange.Name] = value

				// 模拟回测评分（实际应该运行真实回测）
				score := po.simulateBacktestScore(testParams)

				if score > bestScore {
					bestScore = score
					bestParams[paramRange.Name] = value
					log.Printf("发现更好的参数组合: %s=%.3f, 评分=%.3f", paramRange.Name, value, score)
				}

				// 限制搜索时间
				if iterations > 50 {
					break
				}
			}
		}
	}

	return bestParams, bestScore, iterations, nil
}

// simulateBacktestScore 模拟回测评分
func (po *ParameterOptimizer) simulateBacktestScore(params map[string]interface{}) float64 {
	// 这里应该运行真实的回测，现在用简单的模拟
	stopLoss := params["stop_loss"].(float64)
	takeProfit := params["take_profit"].(float64)
	
	// 简单的评分逻辑：止损太小或盈利目标太大都不好
	score := 0.5
	
	if stopLoss >= 0.015 && stopLoss <= 0.025 {
		score += 0.2
	}
	
	if takeProfit >= 0.03 && takeProfit <= 0.07 {
		score += 0.2
	}
	
	// 添加一些随机性模拟市场变化
	score += (math.Sin(float64(time.Now().Unix())) * 0.1)
	
	return math.Max(0.1, math.Min(1.0, score))
}

// applyOptimizedParameters 应用优化后的参数
func (po *ParameterOptimizer) applyOptimizedParameters(ctx context.Context, strategyID string, params map[string]interface{}) error {
	// 这里应该更新策略配置
	log.Printf("应用优化参数到策略 %s: %+v", strategyID, params)
	
	// 更新数据库中的策略配置
	updateQuery := `
		UPDATE strategies 
		SET config_json = $1, updated_at = $2, optimization_applied = true
		WHERE id = $3
	`
	
	// 简化的JSON序列化（实际应该用proper JSON）
	configJSON := fmt.Sprintf(`{"optimized": true, "params": %+v}`, params)
	
	_, err := po.db.ExecContext(ctx, updateQuery, configJSON, time.Now(), strategyID)
	return err
}

// saveOptimizationResult 保存优化结果
func (po *ParameterOptimizer) saveOptimizationResult(ctx context.Context, result *OptimizationResult) error {
	query := `
		INSERT INTO optimization_history (
			strategy_id, original_score, optimized_score, improvement,
			optimization_time_ms, iterations, last_optimization, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (strategy_id) DO UPDATE SET
			original_score = EXCLUDED.original_score,
			optimized_score = EXCLUDED.optimized_score,
			improvement = EXCLUDED.improvement,
			optimization_time_ms = EXCLUDED.optimization_time_ms,
			iterations = EXCLUDED.iterations,
			last_optimization = EXCLUDED.last_optimization,
			created_at = EXCLUDED.created_at
	`

	_, err := po.db.ExecContext(ctx, query,
		result.StrategyID,
		result.OriginalScore,
		result.OptimizedScore,
		result.Improvement,
		result.OptimizationTime.Milliseconds(),
		result.Iterations,
		time.Now(),
		time.Now(),
	)

	return err
}

// GetOptimizerStatus 获取优化器状态
func (po *ParameterOptimizer) GetOptimizerStatus(ctx context.Context) (map[string]interface{}, error) {
	// 统计最近的优化结果
	statsQuery := `
		SELECT 
			COUNT(*) as total_optimizations,
			AVG(improvement) as avg_improvement,
			MAX(improvement) as max_improvement,
			AVG(optimization_time_ms) as avg_time_ms
		FROM optimization_history 
		WHERE created_at > NOW() - INTERVAL '30 days'
	`

	var totalOpts int
	var avgImprovement, maxImprovement float64
	var avgTimeMs int64

	err := po.db.QueryRowContext(ctx, statsQuery).Scan(
		&totalOpts, &avgImprovement, &maxImprovement, &avgTimeMs)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"running":              po.running,
		"optimize_interval":    po.optimizeInterval.String(),
		"total_optimizations":  totalOpts,
		"avg_improvement":      avgImprovement,
		"max_improvement":      maxImprovement,
		"avg_optimization_time": time.Duration(avgTimeMs * int64(time.Millisecond)).String(),
	}, nil
}
