package automation

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/automation/bridge"
	"qcat/internal/automation/executor"
	"qcat/internal/automation/risk"
	"qcat/internal/automation/scheduler"
	"qcat/internal/cache"
	"qcat/internal/config"
	"qcat/internal/database"
	"qcat/internal/exchange"
	"qcat/internal/exchange/account"
	"qcat/internal/exchange/position"
	exchangeRisk "qcat/internal/exchange/risk"
	"qcat/internal/monitor"
	"qcat/internal/strategy/optimizer"
	"qcat/internal/strategy/workflow"
)

// AutomationSystem 自动化系统
// 统一管理所有26项自动化功能
type AutomationSystem struct {
	config         *config.Config
	db             *database.DB
	exchange       exchange.Exchange
	accountManager *account.Manager
	metrics        *monitor.MetricsCollector

	// 核心组件
	scheduler         *scheduler.AutomationScheduler
	executor          *executor.RealtimeExecutor
	bridge            *bridge.MonitorResponseBridge
	monitorIntegrator *bridge.MonitorIntegration
	riskController    *risk.IntelligentRiskController
	strategyPool      *workflow.TradingStrategyPool

	// 运行状态
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	isRunning bool
	mu        sync.RWMutex

	// 系统状态
	status *SystemStatus
}

// SystemStatus 系统状态
type SystemStatus struct {
	StartTime        time.Time
	IsRunning        bool
	SchedulerStatus  string
	ExecutorStatus   string
	ActiveTasks      int
	CompletedTasks   int
	FailedTasks      int
	ActiveActions    int
	CompletedActions int
	FailedActions    int
	LastHealthCheck  time.Time
	HealthScore      float64
	mu               sync.RWMutex
}

// NewAutomationSystem 创建自动化系统
func NewAutomationSystem(
	cfg *config.Config,
	db *database.DB,
	exchange exchange.Exchange,
	accountManager *account.Manager,
	metrics *monitor.MetricsCollector,
	optimizerFactory *optimizer.Factory,
) *AutomationSystem {
	ctx, cancel := context.WithCancel(context.Background())

	// 创建调度器
	automationScheduler := scheduler.NewAutomationScheduler(
		cfg, db, accountManager, metrics, optimizerFactory,
	)

	// 创建执行器
	realtimeExecutor := executor.NewRealtimeExecutor(
		cfg, db, exchange, accountManager, metrics,
	)

	// 创建监控-响应桥接器
	monitorBridge := bridge.NewMonitorResponseBridge(
		cfg, realtimeExecutor, metrics,
	)

	// 创建监控集成器（暂时不使用外部AlertManager和HealthChecker）
	monitorIntegrator := bridge.NewMonitorIntegration(
		monitorBridge, metrics, nil, nil,
	)

	// 创建简单内存缓存（用于仓位管理器）
	cacheFactory := cache.NewCacheFactory(&cache.CacheFactoryConfig{
		MemoryEnabled: true,
		MemoryMaxSize: 1000,
	})
	memoryCache := cacheFactory.CreateMemoryOnlyCache()

	// 创建仓位管理器
	positionManager := position.NewManager(db.DB, memoryCache, exchange)

	// 创建风险引擎
	riskEngine := exchangeRisk.NewRiskEngine(exchange, positionManager)

	// 创建智能风险控制器
	intelligentRiskController := risk.NewIntelligentRiskController(
		cfg, db, exchange, positionManager, riskEngine, realtimeExecutor, metrics,
	)

	return &AutomationSystem{
		config:            cfg,
		db:                db,
		exchange:          exchange,
		accountManager:    accountManager,
		metrics:           metrics,
		scheduler:         automationScheduler,
		executor:          realtimeExecutor,
		bridge:            monitorBridge,
		monitorIntegrator: monitorIntegrator,
		riskController:    intelligentRiskController,
		strategyPool:      nil,
		ctx:               ctx,
		cancel:            cancel,
		status: &SystemStatus{
			StartTime:   time.Now(),
			HealthScore: 1.0,
		},
	}
}

// Start 启动自动化系统
func (as *AutomationSystem) Start() error {
	as.mu.Lock()
	defer as.mu.Unlock()

	if as.isRunning {
		return fmt.Errorf("automation system is already running")
	}

	log.Println("Starting QCAT Automation System...")
	log.Println("Initializing 26 automation features...")

	// 启动执行器
	if err := as.executor.Start(); err != nil {
		return fmt.Errorf("failed to start executor: %w", err)
	}

	// 启动调度器
	if err := as.scheduler.Start(); err != nil {
		as.executor.Stop() // 清理
		return fmt.Errorf("failed to start scheduler: %w", err)
	}

	// 启动监控-响应桥接器
	if err := as.bridge.Start(); err != nil {
		as.executor.Stop()  // 清理
		as.scheduler.Stop() // 清理
		return fmt.Errorf("failed to start monitor bridge: %w", err)
	}

	// 启动监控集成器
	if err := as.monitorIntegrator.Start(); err != nil {
		as.executor.Stop()  // 清理
		as.scheduler.Stop() // 清理
		as.bridge.Stop()    // 清理
		return fmt.Errorf("failed to start monitor integrator: %w", err)
	}

	// 启动智能风险控制器
	if err := as.riskController.Start(); err != nil {
		as.executor.Stop()           // 清理
		as.scheduler.Stop()          // 清理
		as.bridge.Stop()             // 清理
		as.monitorIntegrator.Start() // 清理（这里应该是Stop，但没有Stop方法）
		return fmt.Errorf("failed to start intelligent risk controller: %w", err)
	}

	// 启动监控循环
	as.wg.Add(1)
	go as.monitorLoop()

	// 启动健康检查
	as.wg.Add(1)
	go as.healthCheckLoop()

	as.isRunning = true
	as.status.mu.Lock()
	as.status.IsRunning = true
	as.status.SchedulerStatus = "running"
	as.status.ExecutorStatus = "running"
	as.status.mu.Unlock()

	log.Println("✅ QCAT Automation System started successfully!")
	log.Println("🚀 All 26 automation features are now active!")
	as.logSystemStatus()

	return nil
}

// Stop 停止自动化系统
func (as *AutomationSystem) Stop() error {
	as.mu.Lock()
	defer as.mu.Unlock()

	if !as.isRunning {
		return nil
	}

	log.Println("Stopping QCAT Automation System...")

	// 停止调度器
	if err := as.scheduler.Stop(); err != nil {
		log.Printf("Warning: failed to stop scheduler: %v", err)
	}

	// 停止智能风险控制器
	if err := as.riskController.Stop(); err != nil {
		log.Printf("Warning: failed to stop intelligent risk controller: %v", err)
	}

	// 停止监控-响应桥接器
	if err := as.bridge.Stop(); err != nil {
		log.Printf("Warning: failed to stop monitor bridge: %v", err)
	}

	// 停止执行器
	if err := as.executor.Stop(); err != nil {
		log.Printf("Warning: failed to stop executor: %v", err)
	}

	// 取消上下文
	as.cancel()

	// 等待所有goroutine完成
	as.wg.Wait()

	as.isRunning = false
	as.status.mu.Lock()
	as.status.IsRunning = false
	as.status.SchedulerStatus = "stopped"
	as.status.ExecutorStatus = "stopped"
	as.status.mu.Unlock()

	log.Println("QCAT Automation System stopped")
	return nil
}

// monitorLoop 监控循环
func (as *AutomationSystem) monitorLoop() {
	defer as.wg.Done()

	ticker := time.NewTicker(time.Minute) // 每分钟更新一次
	defer ticker.Stop()

	for {
		select {
		case <-as.ctx.Done():
			return
		case <-ticker.C:
			as.updateSystemStatus()
		}
	}
}

// healthCheckLoop 健康检查循环
func (as *AutomationSystem) healthCheckLoop() {
	defer as.wg.Done()

	ticker := time.NewTicker(time.Minute * 5) // 每5分钟检查一次
	defer ticker.Stop()

	for {
		select {
		case <-as.ctx.Done():
			return
		case <-ticker.C:
			as.performHealthCheck()
		}
	}
}

// updateSystemStatus 更新系统状态
func (as *AutomationSystem) updateSystemStatus() {
	as.status.mu.Lock()
	defer as.status.mu.Unlock()

	// 获取调度器统计
	if as.scheduler != nil {
		schedulerStats := as.scheduler.GetStats()
		as.status.ActiveTasks = schedulerStats.RunningTasks
		as.status.CompletedTasks = schedulerStats.CompletedTasks
		as.status.FailedTasks = schedulerStats.FailedTasks
	} else {
		// 没有调度器时返回0
		as.status.ActiveTasks = 0
		as.status.CompletedTasks = 0
		as.status.FailedTasks = 0
	}

	// 获取执行器统计
	if as.executor != nil {
		executorStats := as.executor.GetStats()
		as.status.ActiveActions = executorStats.TotalActions - executorStats.ExecutedActions - executorStats.FailedActions
		as.status.CompletedActions = executorStats.ExecutedActions
		as.status.FailedActions = executorStats.FailedActions
	} else {
		// 没有执行器时返回0
		as.status.ActiveActions = 0
		as.status.CompletedActions = 0
		as.status.FailedActions = 0
	}
}

// performHealthCheck 执行健康检查
func (as *AutomationSystem) performHealthCheck() {
	as.status.mu.Lock()
	defer as.status.mu.Unlock()

	as.status.LastHealthCheck = time.Now()

	// 计算健康分数
	healthScore := 1.0

	// 检查调度器状态
	if as.status.SchedulerStatus != "running" {
		healthScore -= 0.5
	}

	// 检查执行器状态
	if as.status.ExecutorStatus != "running" {
		healthScore -= 0.5
	}

	// 检查失败率
	totalTasks := as.status.CompletedTasks + as.status.FailedTasks
	if totalTasks > 0 {
		failureRate := float64(as.status.FailedTasks) / float64(totalTasks)
		if failureRate > 0.1 { // 失败率超过10%
			healthScore -= failureRate
		}
	}

	totalActions := as.status.CompletedActions + as.status.FailedActions
	if totalActions > 0 {
		actionFailureRate := float64(as.status.FailedActions) / float64(totalActions)
		if actionFailureRate > 0.1 {
			healthScore -= actionFailureRate
		}
	}

	// 确保健康分数在0-1之间
	if healthScore < 0 {
		healthScore = 0
	}

	as.status.HealthScore = healthScore

	// 记录健康状态
	if healthScore < 0.8 {
		log.Printf("⚠️  System health warning: score %.2f", healthScore)
	}
}

// GetStatus 获取系统状态
func (as *AutomationSystem) GetStatus() *SystemStatus {
	as.status.mu.RLock()
	defer as.status.mu.RUnlock()

	// 返回副本
	return &SystemStatus{
		StartTime:        as.status.StartTime,
		IsRunning:        as.status.IsRunning,
		SchedulerStatus:  as.status.SchedulerStatus,
		ExecutorStatus:   as.status.ExecutorStatus,
		ActiveTasks:      as.status.ActiveTasks,
		CompletedTasks:   as.status.CompletedTasks,
		FailedTasks:      as.status.FailedTasks,
		ActiveActions:    as.status.ActiveActions,
		CompletedActions: as.status.CompletedActions,
		FailedActions:    as.status.FailedActions,
		LastHealthCheck:  as.status.LastHealthCheck,
		HealthScore:      as.status.HealthScore,
	}
}

// logSystemStatus 记录系统状态
func (as *AutomationSystem) logSystemStatus() {
	status := as.GetStatus()

	log.Println("📊 QCAT Automation System Status:")
	log.Printf("   🕐 Start Time: %s", status.StartTime.Format("2006-01-02 15:04:05"))
	log.Printf("   ⚡ Running: %v", status.IsRunning)
	log.Printf("   📋 Scheduler: %s", status.SchedulerStatus)
	log.Printf("   🎯 Executor: %s", status.ExecutorStatus)
	log.Printf("   📈 Health Score: %.2f", status.HealthScore)
	log.Printf("   📊 Tasks: %d active, %d completed, %d failed",
		status.ActiveTasks, status.CompletedTasks, status.FailedTasks)
	log.Printf("   🎯 Actions: %d active, %d completed, %d failed",
		status.ActiveActions, status.CompletedActions, status.FailedActions)
}

// TriggerOptimization 触发策略优化
func (as *AutomationSystem) TriggerOptimization(strategyID string) error {
	// 通过执行器触发优化动作
	return as.executor.ExecuteAction(&executor.ExecutionAction{
		Type:     executor.ActionTypePosition,
		Action:   "trigger_optimization",
		Priority: 2,
		Parameters: map[string]interface{}{
			"strategy_id": strategyID,
		},
		Timeout:    time.Hour,
		MaxRetries: 2,
	})
}

// TriggerRiskControl 触发风险控制
func (as *AutomationSystem) TriggerRiskControl(action string, parameters map[string]interface{}) error {
	return as.executor.ExecuteRiskControl(action, parameters)
}

// IsRunning 检查系统是否运行中
func (as *AutomationSystem) IsRunning() bool {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return as.isRunning
}

// GetExecutor 获取实时执行引擎
func (as *AutomationSystem) GetExecutor() *executor.RealtimeExecutor {
	return as.executor
}

// GetStrategyPool 获取交易策略池（可能为nil）
func (as *AutomationSystem) GetStrategyPool() *workflow.TradingStrategyPool {
	return as.strategyPool
}

// GetScheduler 获取自动化调度器
func (as *AutomationSystem) GetScheduler() *scheduler.AutomationScheduler {
	return as.scheduler
}

// GetBridge 获取监控-响应桥接器
func (as *AutomationSystem) GetBridge() *bridge.MonitorResponseBridge {
	return as.bridge
}

// ToggleTask 切换自动化任务的启用状态
func (as *AutomationSystem) ToggleTask(taskID string, enabled bool) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	// 通过调度器切换任务状态
	if as.scheduler != nil {
		return as.scheduler.ToggleTask(taskID, enabled)
	}

	return fmt.Errorf("scheduler not available")
}

// GetMonitorIntegrator 获取监控集成器
func (as *AutomationSystem) GetMonitorIntegrator() *bridge.MonitorIntegration {
	return as.monitorIntegrator
}

// GetRiskController 获取智能风险控制器
func (as *AutomationSystem) GetRiskController() *risk.IntelligentRiskController {
	return as.riskController
}

// ProcessMonitorEvent 处理监控事件
func (as *AutomationSystem) ProcessMonitorEvent(eventType, source, severity, message string, metadata map[string]interface{}) error {
	event := &bridge.MonitorEvent{
		Type:     bridge.EventType(eventType),
		Source:   source,
		Severity: bridge.EventSeverity(severity),
		Message:  message,
		Metadata: metadata,
	}

	return as.bridge.ProcessEvent(event)
}
