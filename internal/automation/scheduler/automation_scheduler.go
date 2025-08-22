package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/config"
	"qcat/internal/database"
	"qcat/internal/exchange/account"
	"qcat/internal/monitor"
	"qcat/internal/strategy/optimizer"
)

// AutomationScheduler 统一的自动化调度器
// 负责协调和执行所有26项自动化功能
type AutomationScheduler struct {
	config           *config.Config
	db               *database.DB
	accountManager   *account.Manager
	metrics          *monitor.MetricsCollector
	optimizerFactory *optimizer.Factory

	// 调度器组件
	strategyScheduler *StrategyScheduler
	riskScheduler     *RiskScheduler
	positionScheduler *PositionScheduler
	dataScheduler     *DataScheduler
	systemScheduler   *SystemScheduler
	learningScheduler *LearningScheduler

	// 运行状态
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	isRunning bool
	mu        sync.RWMutex

	// 调度任务
	tasks     map[string]*ScheduledTask
	taskQueue chan *ScheduledTask
	workers   []*TaskWorker

	// 统计信息
	stats *SchedulerStats
}

// ScheduledTask 调度任务
type ScheduledTask struct {
	ID         string
	Name       string
	Type       TaskType
	Category   TaskCategory
	Schedule   string // cron表达式
	NextRun    time.Time
	LastRun    time.Time
	Status     TaskStatus
	Enabled    bool // 是否启用
	Priority   int
	Timeout    time.Duration
	RetryCount int
	MaxRetries int
	Config     map[string]interface{}
	Handler    TaskHandler
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// TaskType 任务类型
type TaskType string

const (
	TaskTypeOptimization       TaskType = "optimization"
	TaskTypeRiskMonitoring     TaskType = "risk_monitoring"
	TaskTypeRiskManagement     TaskType = "risk_management"
	TaskTypePositionManagement TaskType = "position_management"
	TaskTypeDataProcessing     TaskType = "data_processing"
	TaskTypeSystemMaintenance  TaskType = "system_maintenance"
	TaskTypeLearning           TaskType = "learning"
	TaskTypeSecurityMonitoring TaskType = "security_monitoring"
)

// TaskCategory 任务分类
type TaskCategory string

const (
	CategoryStrategy TaskCategory = "strategy"
	CategoryRisk     TaskCategory = "risk"
	CategoryPosition TaskCategory = "position"
	CategoryData     TaskCategory = "data"
	CategorySystem   TaskCategory = "system"
	CategoryLearning TaskCategory = "learning"
	CategorySecurity TaskCategory = "security"
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusSkipped   TaskStatus = "skipped"
	TaskStatusStopped   TaskStatus = "stopped"
)

// TaskHandler 任务处理器
type TaskHandler func(ctx context.Context, task *ScheduledTask) error

// SchedulerStats 调度器统计
type SchedulerStats struct {
	TotalTasks     int
	RunningTasks   int
	CompletedTasks int
	FailedTasks    int
	SkippedTasks   int
	AverageRunTime time.Duration
	LastUpdateTime time.Time
	mu             sync.RWMutex
}

// NewAutomationScheduler 创建自动化调度器
func NewAutomationScheduler(
	cfg *config.Config,
	db *database.DB,
	accountManager *account.Manager,
	metrics *monitor.MetricsCollector,
	optimizerFactory *optimizer.Factory,
) *AutomationScheduler {
	ctx, cancel := context.WithCancel(context.Background())

	scheduler := &AutomationScheduler{
		config:           cfg,
		db:               db,
		accountManager:   accountManager,
		metrics:          metrics,
		optimizerFactory: optimizerFactory,
		ctx:              ctx,
		cancel:           cancel,
		tasks:            make(map[string]*ScheduledTask),
		taskQueue:        make(chan *ScheduledTask, 1000),
		workers:          make([]*TaskWorker, 0),
		stats:            &SchedulerStats{},
	}

	// 初始化子调度器
	scheduler.initializeSubSchedulers()

	// 初始化工作线程
	scheduler.initializeWorkers()

	// 注册默认任务
	scheduler.registerDefaultTasks()

	return scheduler
}

// Start 启动调度器
func (as *AutomationScheduler) Start() error {
	as.mu.Lock()
	defer as.mu.Unlock()

	if as.isRunning {
		return fmt.Errorf("scheduler is already running")
	}

	log.Println("Starting automation scheduler...")

	// 启动工作线程
	for _, worker := range as.workers {
		as.wg.Add(1)
		go worker.Start(&as.wg)
	}

	// 启动调度循环
	as.wg.Add(1)
	go as.scheduleLoop()

	// 启动子调度器
	if err := as.startSubSchedulers(); err != nil {
		return fmt.Errorf("failed to start sub-schedulers: %w", err)
	}

	as.isRunning = true
	log.Println("Automation scheduler started successfully")

	return nil
}

// Stop 停止调度器
func (as *AutomationScheduler) Stop() error {
	as.mu.Lock()
	defer as.mu.Unlock()

	if !as.isRunning {
		return nil
	}

	log.Println("Stopping automation scheduler...")

	// 停止子调度器
	as.stopSubSchedulers()

	// 取消上下文
	as.cancel()

	// 等待所有goroutine完成
	as.wg.Wait()

	// 关闭任务队列
	close(as.taskQueue)

	as.isRunning = false
	log.Println("Automation scheduler stopped")

	return nil
}

// scheduleLoop 调度循环
func (as *AutomationScheduler) scheduleLoop() {
	defer as.wg.Done()

	ticker := time.NewTicker(time.Minute) // 每分钟检查一次
	defer ticker.Stop()

	for {
		select {
		case <-as.ctx.Done():
			return
		case <-ticker.C:
			as.checkAndScheduleTasks()
		}
	}
}

// checkAndScheduleTasks 检查并调度任务
func (as *AutomationScheduler) checkAndScheduleTasks() {
	as.mu.RLock()
	tasks := make([]*ScheduledTask, 0, len(as.tasks))
	for _, task := range as.tasks {
		tasks = append(tasks, task)
	}
	as.mu.RUnlock()

	now := time.Now()
	for _, task := range tasks {
		if task.Status == TaskStatusPending && now.After(task.NextRun) {
			// 将任务加入队列
			select {
			case as.taskQueue <- task:
				task.Status = TaskStatusRunning
				log.Printf("Scheduled task: %s (%s)", task.Name, task.ID)
			default:
				log.Printf("Task queue is full, skipping task: %s", task.Name)
				task.Status = TaskStatusSkipped
			}
		}
	}
}

// initializeSubSchedulers 初始化子调度器
func (as *AutomationScheduler) initializeSubSchedulers() {
	as.strategyScheduler = NewStrategyScheduler(as.config, as.db, as.optimizerFactory)
	as.riskScheduler = NewRiskScheduler(as.config, as.db, as.accountManager)
	as.positionScheduler = NewPositionScheduler(as.config, as.db, as.accountManager)
	as.dataScheduler = NewDataScheduler(as.config, as.db)
	as.systemScheduler = NewSystemScheduler(as.config, as.db, as.metrics)
	as.learningScheduler = NewLearningScheduler(as.config, as.db)
}

// initializeWorkers 初始化工作线程
func (as *AutomationScheduler) initializeWorkers() {
	workerCount := 5 // 可配置
	for i := 0; i < workerCount; i++ {
		worker := NewTaskWorker(i, as.taskQueue, as.handleTaskCompletion)
		as.workers = append(as.workers, worker)
	}
}

// handleTaskCompletion 处理任务完成
func (as *AutomationScheduler) handleTaskCompletion(task *ScheduledTask, err error) {
	as.mu.Lock()
	defer as.mu.Unlock()

	if err != nil {
		task.Status = TaskStatusFailed
		task.RetryCount++
		log.Printf("Task failed: %s, error: %v, retry: %d/%d",
			task.Name, err, task.RetryCount, task.MaxRetries)

		// 重试逻辑
		if task.RetryCount < task.MaxRetries {
			task.Status = TaskStatusPending
			task.NextRun = time.Now().Add(time.Minute * time.Duration(task.RetryCount))
		}
	} else {
		task.Status = TaskStatusCompleted
		task.LastRun = time.Now()
		log.Printf("Task completed: %s", task.Name)

		// 计算下次运行时间
		task.NextRun = as.calculateNextRun(task)
		task.Status = TaskStatusPending
	}

	task.UpdatedAt = time.Now()
	as.updateStats()
}

// calculateNextRun 计算下次运行时间
func (as *AutomationScheduler) calculateNextRun(task *ScheduledTask) time.Time {
	// 简化的cron解析，实际应该使用专业的cron库
	switch task.Schedule {
	case "*/5 * * * *": // 每5分钟
		return time.Now().Add(5 * time.Minute)
	case "0 * * * *": // 每小时
		return time.Now().Add(time.Hour)
	case "0 0 * * *": // 每天
		return time.Now().Add(24 * time.Hour)
	case "0 0 * * 0": // 每周
		return time.Now().Add(7 * 24 * time.Hour)
	default:
		return time.Now().Add(time.Hour) // 默认1小时
	}
}

// updateStats 更新统计信息
func (as *AutomationScheduler) updateStats() {
	as.stats.mu.Lock()
	defer as.stats.mu.Unlock()

	as.stats.TotalTasks = len(as.tasks)
	as.stats.RunningTasks = 0
	as.stats.CompletedTasks = 0
	as.stats.FailedTasks = 0
	as.stats.SkippedTasks = 0

	for _, task := range as.tasks {
		switch task.Status {
		case TaskStatusRunning:
			as.stats.RunningTasks++
		case TaskStatusCompleted:
			as.stats.CompletedTasks++
		case TaskStatusFailed:
			as.stats.FailedTasks++
		case TaskStatusSkipped:
			as.stats.SkippedTasks++
		}
	}

	as.stats.LastUpdateTime = time.Now()
}

// GetStats 获取统计信息
func (as *AutomationScheduler) GetStats() *SchedulerStats {
	as.stats.mu.RLock()
	defer as.stats.mu.RUnlock()

	// 返回副本
	return &SchedulerStats{
		TotalTasks:     as.stats.TotalTasks,
		RunningTasks:   as.stats.RunningTasks,
		CompletedTasks: as.stats.CompletedTasks,
		FailedTasks:    as.stats.FailedTasks,
		SkippedTasks:   as.stats.SkippedTasks,
		AverageRunTime: as.stats.AverageRunTime,
		LastUpdateTime: as.stats.LastUpdateTime,
	}
}

// registerDefaultTasks 注册默认任务
func (as *AutomationScheduler) registerDefaultTasks() {
	// 1. 策略参数自动优化
	as.RegisterTask(&ScheduledTask{
		ID:         "strategy_optimization",
		Name:       "策略参数自动优化",
		Type:       TaskTypeOptimization,
		Category:   CategoryStrategy,
		Schedule:   "0 0 * * 0", // 每周日执行
		Priority:   1,
		Timeout:    time.Hour * 2,
		MaxRetries: 3,
		Handler:    as.strategyScheduler.HandleOptimization,
	})

	// 2. 仓位动态优化
	as.RegisterTask(&ScheduledTask{
		ID:         "position_optimization",
		Name:       "仓位动态优化",
		Type:       TaskTypePositionManagement,
		Category:   CategoryPosition,
		Schedule:   "*/15 * * * *", // 每15分钟执行
		Priority:   2,
		Timeout:    time.Minute * 5,
		MaxRetries: 2,
		Handler:    as.positionScheduler.HandleOptimization,
	})

	// 3. 风险监控
	as.RegisterTask(&ScheduledTask{
		ID:         "risk_monitoring",
		Name:       "风险监控",
		Type:       TaskTypeRiskMonitoring,
		Category:   CategoryRisk,
		Schedule:   "*/5 * * * *", // 每5分钟执行
		Priority:   3,
		Timeout:    time.Minute * 2,
		MaxRetries: 1,
		Handler:    as.riskScheduler.HandleMonitoring,
	})

	// 4. 数据清洗
	as.RegisterTask(&ScheduledTask{
		ID:         "data_cleaning",
		Name:       "数据清洗与校正",
		Type:       TaskTypeDataProcessing,
		Category:   CategoryData,
		Schedule:   "0 * * * *", // 每小时执行
		Priority:   4,
		Timeout:    time.Minute * 10,
		MaxRetries: 2,
		Handler:    as.dataScheduler.HandleCleaning,
	})

	// 5. 系统健康检查
	as.RegisterTask(&ScheduledTask{
		ID:         "system_health",
		Name:       "系统健康监控",
		Type:       TaskTypeSystemMaintenance,
		Category:   CategorySystem,
		Schedule:   "*/5 * * * *", // 每5分钟执行
		Priority:   5,
		Timeout:    time.Minute,
		MaxRetries: 1,
		Handler:    as.systemScheduler.HandleHealthCheck,
	})

	// 6. 策略自学习
	as.RegisterTask(&ScheduledTask{
		ID:         "strategy_learning",
		Name:       "策略自学习",
		Type:       TaskTypeLearning,
		Category:   CategoryLearning,
		Schedule:   "0 0 * * *", // 每天执行
		Priority:   6,
		Timeout:    time.Hour,
		MaxRetries: 2,
		Handler:    as.learningScheduler.HandleLearning,
	})

	// 7. 周期性策略优化
	as.RegisterTask(&ScheduledTask{
		ID:         "periodic_strategy_optimization",
		Name:       "周期性策略优化",
		Type:       TaskTypeOptimization,
		Category:   CategoryStrategy,
		Schedule:   "0 2 * * 0", // 每周日凌晨2点执行
		Priority:   7,
		Timeout:    time.Hour * 3,
		MaxRetries: 2,
		Handler:    as.strategyScheduler.HandlePeriodicOptimization,
	})

	// 8. 策略淘汰与限时禁用 (降低频率，更谨慎)
	as.RegisterTask(&ScheduledTask{
		ID:         "strategy_elimination",
		Name:       "策略淘汰与限时禁用",
		Type:       TaskTypeOptimization,
		Category:   CategoryStrategy,
		Schedule:   "0 1 * * 0", // 每周日凌晨1点执行 (从每天改为每周)
		Priority:   8,
		Timeout:    time.Minute * 30,
		MaxRetries: 2,
		Handler:    as.strategyScheduler.HandleElimination,
	})

	// 9. 新策略引入 (增加频率，确保有足够策略)
	as.RegisterTask(&ScheduledTask{
		ID:         "new_strategy_introduction",
		Name:       "新策略引入",
		Type:       TaskTypeOptimization,
		Category:   CategoryStrategy,
		Schedule:   "0 3 * * *", // 每天凌晨3点执行 (从每周改为每天)
		Priority:   9,
		Timeout:    time.Hour,
		MaxRetries: 2,
		Handler:    as.strategyScheduler.HandleNewStrategyIntroduction,
	})

	// 9.1. 最小策略数量检查 (新增任务，更频繁检查)
	as.RegisterTask(&ScheduledTask{
		ID:         "minimum_strategy_check",
		Name:       "最小策略数量检查",
		Type:       TaskTypeOptimization,
		Category:   CategoryStrategy,
		Schedule:   "*/30 * * * *", // 每30分钟检查一次
		Priority:   1,              // 高优先级
		Timeout:    time.Minute * 10,
		MaxRetries: 3,
		Handler:    as.strategyScheduler.HandleMinimumStrategyCheck,
	})

	// 10. 止盈止损线自动调整
	as.RegisterTask(&ScheduledTask{
		ID:         "stop_loss_adjustment",
		Name:       "止盈止损线自动调整",
		Type:       TaskTypeRiskManagement,
		Category:   CategoryRisk,
		Schedule:   "*/30 * * * *", // 每30分钟执行
		Priority:   10,
		Timeout:    time.Minute * 10,
		MaxRetries: 2,
		Handler:    as.riskScheduler.HandleStopLossAdjustment,
	})

	// 11. 热门币种推荐
	as.RegisterTask(&ScheduledTask{
		ID:         "hot_coin_recommendation",
		Name:       "热门币种推荐",
		Type:       TaskTypeDataProcessing,
		Category:   CategoryData,
		Schedule:   "0 */4 * * *", // 每4小时执行
		Priority:   11,
		Timeout:    time.Minute * 20,
		MaxRetries: 2,
		Handler:    as.dataScheduler.HandleHotCoinRecommendation,
	})

	// 12. 利润最大化引擎
	as.RegisterTask(&ScheduledTask{
		ID:         "profit_maximization",
		Name:       "利润最大化引擎",
		Type:       TaskTypeOptimization,
		Category:   CategoryStrategy,
		Schedule:   "*/10 * * * *", // 每10分钟执行
		Priority:   12,
		Timeout:    time.Minute * 15,
		MaxRetries: 2,
		Handler:    as.strategyScheduler.HandleProfitMaximization,
	})

	// 13. 资金分散与转移
	as.RegisterTask(&ScheduledTask{
		ID:         "fund_distribution",
		Name:       "资金分散与转移",
		Type:       TaskTypeRiskManagement,
		Category:   CategoryRisk,
		Schedule:   "0 */6 * * *", // 每6小时执行
		Priority:   13,
		Timeout:    time.Minute * 30,
		MaxRetries: 2,
		Handler:    as.riskScheduler.HandleFundDistribution,
	})

	// 14. 自动化多策略对冲
	as.RegisterTask(&ScheduledTask{
		ID:         "multi_strategy_hedging",
		Name:       "自动化多策略对冲",
		Type:       TaskTypePositionManagement,
		Category:   CategoryPosition,
		Schedule:   "*/20 * * * *", // 每20分钟执行
		Priority:   14,
		Timeout:    time.Minute * 10,
		MaxRetries: 2,
		Handler:    as.positionScheduler.HandleMultiStrategyHedging,
	})

	// 15. 因子库动态更新
	as.RegisterTask(&ScheduledTask{
		ID:         "factor_library_update",
		Name:       "因子库动态更新",
		Type:       TaskTypeDataProcessing,
		Category:   CategoryData,
		Schedule:   "0 0 * * *", // 每天执行
		Priority:   15,
		Timeout:    time.Hour,
		MaxRetries: 2,
		Handler:    as.dataScheduler.HandleFactorLibraryUpdate,
	})

	// 16. 策略自学习AutoML
	as.RegisterTask(&ScheduledTask{
		ID:         "automl_learning",
		Name:       "策略自学习AutoML",
		Type:       TaskTypeLearning,
		Category:   CategoryLearning,
		Schedule:   "0 2 * * *", // 每天凌晨2点执行
		Priority:   16,
		Timeout:    time.Hour * 2,
		MaxRetries: 2,
		Handler:    as.learningScheduler.HandleAutoMLLearning,
	})

	// 17. 遗传淘汰制升级
	as.RegisterTask(&ScheduledTask{
		ID:         "genetic_evolution",
		Name:       "遗传淘汰制升级",
		Type:       TaskTypeLearning,
		Category:   CategoryLearning,
		Schedule:   "0 3 * * 0", // 每周日凌晨3点执行
		Priority:   17,
		Timeout:    time.Hour * 3,
		MaxRetries: 2,
		Handler:    as.learningScheduler.HandleGeneticEvolution,
	})

	// 18. 市场模式识别
	as.RegisterTask(&ScheduledTask{
		ID:         "market_pattern_recognition",
		Name:       "市场模式识别",
		Type:       TaskTypeDataProcessing,
		Category:   CategoryData,
		Schedule:   "*/5 * * * *", // 每5分钟执行
		Priority:   18,
		Timeout:    time.Minute * 5,
		MaxRetries: 2,
		Handler:    as.dataScheduler.HandleMarketPatternRecognition,
	})

	// 19. 异常行情应对
	as.RegisterTask(&ScheduledTask{
		ID:         "abnormal_market_response",
		Name:       "异常行情应对",
		Type:       TaskTypeRiskManagement,
		Category:   CategoryRisk,
		Schedule:   "*/1 * * * *", // 每分钟检查
		Priority:   19,
		Timeout:    time.Minute * 2,
		MaxRetries: 1,
		Handler:    as.riskScheduler.HandleAbnormalMarketResponse,
	})

	// 20. 账户安全监控
	as.RegisterTask(&ScheduledTask{
		ID:         "account_security_monitoring",
		Name:       "账户安全监控",
		Type:       TaskTypeSecurityMonitoring,
		Category:   CategorySecurity,
		Schedule:   "*/10 * * * *", // 每10分钟执行
		Priority:   20,
		Timeout:    time.Minute * 5,
		MaxRetries: 2,
		Handler:    as.systemScheduler.HandleAccountSecurityMonitoring,
	})

	// 21. 资金动态分配
	as.RegisterTask(&ScheduledTask{
		ID:         "dynamic_fund_allocation",
		Name:       "资金动态分配",
		Type:       TaskTypePositionManagement,
		Category:   CategoryPosition,
		Schedule:   "*/15 * * * *", // 每15分钟执行
		Priority:   21,
		Timeout:    time.Minute * 10,
		MaxRetries: 2,
		Handler:    as.positionScheduler.HandleDynamicFundAllocation,
	})

	// 22. 仓位分层机制
	as.RegisterTask(&ScheduledTask{
		ID:         "layered_position_management",
		Name:       "仓位分层机制",
		Type:       TaskTypePositionManagement,
		Category:   CategoryPosition,
		Schedule:   "*/20 * * * *", // 每20分钟执行
		Priority:   22,
		Timeout:    time.Minute * 10,
		MaxRetries: 2,
		Handler:    as.positionScheduler.HandleLayeredPositionManagement,
	})

	// 23. 自动回测与前测
	as.RegisterTask(&ScheduledTask{
		ID:         "auto_backtesting",
		Name:       "自动回测与前测",
		Type:       TaskTypeDataProcessing,
		Category:   CategoryData,
		Schedule:   "0 0 * * *", // 每天执行
		Priority:   23,
		Timeout:    time.Hour * 2,
		MaxRetries: 2,
		Handler:    as.dataScheduler.HandleAutoBacktesting,
	})

	// 24. 多交易所冗余
	as.RegisterTask(&ScheduledTask{
		ID:         "multi_exchange_redundancy",
		Name:       "多交易所冗余",
		Type:       TaskTypeSystemMaintenance,
		Category:   CategorySystem,
		Schedule:   "*/5 * * * *", // 每5分钟检查
		Priority:   24,
		Timeout:    time.Minute * 3,
		MaxRetries: 1,
		Handler:    as.systemScheduler.HandleMultiExchangeRedundancy,
	})

	// 25. 日志与审计追踪
	as.RegisterTask(&ScheduledTask{
		ID:         "audit_logging",
		Name:       "日志与审计追踪",
		Type:       TaskTypeSystemMaintenance,
		Category:   CategorySystem,
		Schedule:   "*/30 * * * *", // 每30分钟执行
		Priority:   25,
		Timeout:    time.Minute * 5,
		MaxRetries: 1,
		Handler:    as.systemScheduler.HandleAuditLogging,
	})

	// 26. 最佳参数应用
	as.RegisterTask(&ScheduledTask{
		ID:         "best_parameter_application",
		Name:       "最佳参数应用",
		Type:       TaskTypeOptimization,
		Category:   CategoryStrategy,
		Schedule:   "0 4 * * *", // 每天凌晨4点执行
		Priority:   26,
		Timeout:    time.Minute * 30,
		MaxRetries: 2,
		Handler:    as.strategyScheduler.HandleBestParameterApplication,
	})

	log.Printf("Successfully registered all 26 automation tasks")
}

// RegisterTask 注册任务
func (as *AutomationScheduler) RegisterTask(task *ScheduledTask) {
	as.mu.Lock()
	defer as.mu.Unlock()

	if task.ID == "" {
		task.ID = fmt.Sprintf("task_%d", time.Now().UnixNano())
	}

	task.Status = TaskStatusPending
	task.Enabled = false // 默认禁用，需要手动启用
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	task.NextRun = time.Now().Add(time.Minute) // 1分钟后开始

	as.tasks[task.ID] = task
	log.Printf("Registered task: %s (%s)", task.Name, task.ID)
}

// startSubSchedulers 启动子调度器
func (as *AutomationScheduler) startSubSchedulers() error {
	if err := as.strategyScheduler.Start(); err != nil {
		return fmt.Errorf("failed to start strategy scheduler: %w", err)
	}
	if err := as.riskScheduler.Start(); err != nil {
		return fmt.Errorf("failed to start risk scheduler: %w", err)
	}
	if err := as.positionScheduler.Start(); err != nil {
		return fmt.Errorf("failed to start position scheduler: %w", err)
	}
	if err := as.dataScheduler.Start(); err != nil {
		return fmt.Errorf("failed to start data scheduler: %w", err)
	}
	if err := as.systemScheduler.Start(); err != nil {
		return fmt.Errorf("failed to start system scheduler: %w", err)
	}
	if err := as.learningScheduler.Start(); err != nil {
		return fmt.Errorf("failed to start learning scheduler: %w", err)
	}
	return nil
}

// stopSubSchedulers 停止子调度器
func (as *AutomationScheduler) stopSubSchedulers() {
	as.strategyScheduler.Stop()
	as.riskScheduler.Stop()
	as.positionScheduler.Stop()
	as.dataScheduler.Stop()
	as.systemScheduler.Stop()
	as.learningScheduler.Stop()
}

// ToggleTask 切换任务的启用状态
func (as *AutomationScheduler) ToggleTask(taskID string, enabled bool) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	task, exists := as.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// 更新任务状态
	task.Enabled = enabled
	task.UpdatedAt = time.Now()

	if enabled {
		// 启用任务
		task.Status = TaskStatusPending
		task.NextRun = time.Now().Add(time.Minute) // 1分钟后开始
		log.Printf("Enabled task: %s (%s)", task.Name, task.ID)
	} else {
		// 禁用任务
		task.Status = TaskStatusStopped
		log.Printf("Disabled task: %s (%s)", task.Name, task.ID)
	}

	as.tasks[taskID] = task
	return nil
}

// GetTask 获取指定ID的任务
func (as *AutomationScheduler) GetTask(taskID string) *ScheduledTask {
	as.mu.RLock()
	defer as.mu.RUnlock()

	if task, exists := as.tasks[taskID]; exists {
		return task
	}
	return nil
}
