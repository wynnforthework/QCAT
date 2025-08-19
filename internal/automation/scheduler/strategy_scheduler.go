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
	ID           string
	Name         string
	Status       string
	LastOptimized time.Time
	Performance  float64
	SharpeRatio  float64
	MaxDrawdown  float64
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

	// 执行优化
	result, err := orchestrator.OptimizeStrategy(ctx, strategyID)
	if err != nil {
		return fmt.Errorf("optimization failed: %w", err)
	}

	// 应用优化结果
	if err := ss.applyOptimizationResult(ctx, strategyID, result); err != nil {
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
func (ss *StrategyScheduler) applyOptimizationResult(ctx context.Context, strategyID string, result interface{}) error {
	// TODO: 实现优化结果应用逻辑
	// 1. 验证优化结果
	// 2. 创建新的策略版本
	// 3. 执行Canary部署
	// 4. 监控性能表现
	// 5. 决定是否全量切换

	log.Printf("Applying optimization result for strategy %s", strategyID)
	
	// 这里应该实现具体的应用逻辑
	// 暂时只记录日志
	
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

	// TODO: 实现策略评估逻辑
	// 1. 收集策略性能数据
	// 2. 计算关键指标
	// 3. 与基准比较
	// 4. 生成评估报告
	// 5. 触发必要的优化任务

	return nil
}
