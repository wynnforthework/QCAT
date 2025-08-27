package workflow

import (
	"context"
	"fmt"
	"log"
	"qcat/internal/workflow/interfaces"
	"sync"
	"time"
)

// TradingWorkflowEngine 交易工作流引擎
// 专注于已启用策略的交易执行（第1-2层和第6-9层）
type TradingWorkflowEngine struct {
	// 原有工作流引擎
	enhancedEngine *EnhancedWorkflowEngine

	// 策略池
	strategyPool interfaces.StrategyPoolInterface

	// 交易执行层功能ID
	tradingFunctions []int

	// 运行状态
	ctx       context.Context
	cancel    context.CancelFunc
	isRunning bool
	runningMu sync.RWMutex
	wg        sync.WaitGroup

	// 配置
	config *interfaces.TradingWorkflowConfig

	// 统计信息
	stats   *TradingWorkflowStats
	statsMu sync.RWMutex
}

// TradingWorkflowStats 交易工作流统计信息
type TradingWorkflowStats struct {
	TotalExecutions      int64
	SuccessfulExecutions int64
	FailedExecutions     int64
	AverageExecutionTime time.Duration
	ActiveStrategies     int64
	LastExecutionTime    time.Time
	LastStrategyCheck    time.Time
}

// NewTradingWorkflowEngine 创建交易工作流引擎
func NewTradingWorkflowEngine(strategyPool interfaces.StrategyPoolInterface, config *interfaces.TradingWorkflowConfig) *TradingWorkflowEngine {
	if config == nil {
		config = interfaces.GetDefaultTradingWorkflowConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 创建增强工作流引擎
	enhancedEngine := NewEnhancedWorkflowEngine(config.MaxConcurrency)

	// 定义交易执行相关的功能ID
	tradingFunctions := []int{
		// 第1层: 基础数据层
		18, 21, 23, // 数据清洗、系统监控

		// 第2层: 监控分析层
		26, 12, 13, // 市场分析、风险监控

		// 第6层: 仓位管理层
		15, 3, 16, 17, // 资金分配、仓位优化

		// 第7层: 交易执行层
		4, 5, 9, // 建仓平仓、止盈止损

		// 第8层: 收益优化层
		11, // 利润最大化引擎

		// 第9层: 安全保障层
		14, 22, // 资金转移、交易所冗余
	}

	return &TradingWorkflowEngine{
		enhancedEngine:   enhancedEngine,
		strategyPool:     strategyPool,
		tradingFunctions: tradingFunctions,
		ctx:              ctx,
		cancel:           cancel,
		config:           config,
		stats: &TradingWorkflowStats{
			LastExecutionTime: time.Now(),
			LastStrategyCheck: time.Now(),
		},
	}
}

// Start 启动交易工作流引擎
func (twe *TradingWorkflowEngine) Start(ctx context.Context) error {
	twe.runningMu.Lock()
	defer twe.runningMu.Unlock()

	if twe.isRunning {
		return fmt.Errorf("trading workflow engine is already running")
	}

	log.Println("启动交易工作流引擎...")

	// 增强工作流引擎在创建时已经启动了事件处理器
	log.Println("增强工作流引擎已就绪")

	// 启动策略池
	if err := twe.strategyPool.Start(); err != nil {
		return fmt.Errorf("failed to start strategy pool: %w", err)
	}

	// 启动执行循环
	twe.wg.Add(2)
	go twe.runExecutionLoop()
	go twe.runStrategyCheckLoop()

	twe.isRunning = true

	log.Println("交易工作流引擎启动完成")
	return nil
}

// Stop 停止交易工作流引擎
func (twe *TradingWorkflowEngine) Stop() error {
	twe.runningMu.Lock()
	defer twe.runningMu.Unlock()

	if !twe.isRunning {
		return nil
	}

	log.Println("停止交易工作流引擎...")

	// 取消上下文
	twe.cancel()

	// 等待循环结束
	twe.wg.Wait()

	// 停止策略池
	if err := twe.strategyPool.Stop(); err != nil {
		log.Printf("Warning: failed to stop strategy pool: %v", err)
	}

	// 停止增强工作流引擎
	twe.enhancedEngine.Stop()
	log.Println("增强工作流引擎已停止")

	twe.isRunning = false

	log.Println("交易工作流引擎已停止")
	return nil
}

// runExecutionLoop 运行执行循环
func (twe *TradingWorkflowEngine) runExecutionLoop() {
	defer twe.wg.Done()

	ticker := time.NewTicker(twe.config.ExecutionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-twe.ctx.Done():
			return
		case <-ticker.C:
			if err := twe.executeTrading(); err != nil {
				log.Printf("Error during trading execution: %v", err)
				twe.updateStats(false, time.Since(time.Now()))
			}
		}
	}
}

// runStrategyCheckLoop 运行策略检查循环
func (twe *TradingWorkflowEngine) runStrategyCheckLoop() {
	defer twe.wg.Done()

	ticker := time.NewTicker(twe.config.StrategyCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-twe.ctx.Done():
			return
		case <-ticker.C:
			twe.checkStrategies()
		}
	}
}

// executeTrading 执行交易
func (twe *TradingWorkflowEngine) executeTrading() error {
	startTime := time.Now()

	// 检查是否有启用的策略
	enabledStrategies := twe.strategyPool.GetEnabledStrategies()
	if len(enabledStrategies) == 0 {
		log.Println("没有启用的策略，跳过交易执行")
		return nil
	}

	log.Printf("开始执行交易工作流，启用策略数: %d", len(enabledStrategies))

	// 只启用交易相关的功能
	twe.enableTradingFunctions()

	// 执行工作流
	if err := twe.enhancedEngine.ExecuteWorkflow(twe.ctx); err != nil {
		twe.updateStats(false, time.Since(startTime))
		return fmt.Errorf("failed to execute trading workflow: %w", err)
	}

	// 更新统计信息
	twe.updateStats(true, time.Since(startTime))

	log.Printf("交易工作流执行完成，耗时: %v", time.Since(startTime))
	return nil
}

// enableTradingFunctions 启用交易功能
func (twe *TradingWorkflowEngine) enableTradingFunctions() {
	// 禁用所有功能
	for i := 1; i <= 26; i++ {
		twe.enhancedEngine.dependencyGraph.DisableFunction(i)
	}

	// 只启用交易相关功能
	for _, functionID := range twe.tradingFunctions {
		twe.enhancedEngine.dependencyGraph.EnableFunction(functionID)
	}

	log.Printf("已启用 %d 个交易相关功能", len(twe.tradingFunctions))
}

// checkStrategies 检查策略状态
func (twe *TradingWorkflowEngine) checkStrategies() {
	enabledStrategies := twe.strategyPool.GetEnabledStrategies()

	twe.statsMu.Lock()
	twe.stats.ActiveStrategies = int64(len(enabledStrategies))
	twe.stats.LastStrategyCheck = time.Now()
	twe.statsMu.Unlock()

	log.Printf("策略状态检查完成，当前启用策略数: %d", len(enabledStrategies))

	// 如果没有启用的策略，可以考虑暂停交易执行
	if len(enabledStrategies) == 0 {
		log.Println("Warning: 没有启用的策略，交易执行将被跳过")
	}
}

// updateStats 更新统计信息
func (twe *TradingWorkflowEngine) updateStats(success bool, duration time.Duration) {
	twe.statsMu.Lock()
	defer twe.statsMu.Unlock()

	twe.stats.TotalExecutions++
	if success {
		twe.stats.SuccessfulExecutions++
	} else {
		twe.stats.FailedExecutions++
	}

	// 更新平均执行时间
	if twe.stats.TotalExecutions > 0 {
		totalTime := time.Duration(int64(twe.stats.AverageExecutionTime) * (twe.stats.TotalExecutions - 1))
		twe.stats.AverageExecutionTime = (totalTime + duration) / time.Duration(twe.stats.TotalExecutions)
	}

	twe.stats.LastExecutionTime = time.Now()
}

// GetStats 获取统计信息（实现接口）
func (twe *TradingWorkflowEngine) GetStats() map[string]interface{} {
	twe.statsMu.RLock()
	defer twe.statsMu.RUnlock()

	return map[string]interface{}{
		"total_executions":       twe.stats.TotalExecutions,
		"successful_executions":  twe.stats.SuccessfulExecutions,
		"failed_executions":      twe.stats.FailedExecutions,
		"average_execution_time": twe.stats.AverageExecutionTime,
		"active_strategies":      twe.stats.ActiveStrategies,
		"last_execution_time":    twe.stats.LastExecutionTime,
		"last_strategy_check":    twe.stats.LastStrategyCheck,
	}
}

// GetStatsStruct 获取结构化统计信息
func (twe *TradingWorkflowEngine) GetStatsStruct() *TradingWorkflowStats {
	twe.statsMu.RLock()
	defer twe.statsMu.RUnlock()

	// 返回副本
	stats := *twe.stats
	return &stats
}

// IsRunning 检查是否正在运行
func (twe *TradingWorkflowEngine) IsRunning() bool {
	twe.runningMu.RLock()
	defer twe.runningMu.RUnlock()
	return twe.isRunning
}

// ExecuteWorkflow 执行工作流（实现接口）
func (twe *TradingWorkflowEngine) ExecuteWorkflow(ctx context.Context) error {
	return twe.executeTrading()
}
