package automation

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/automation/scheduler"
	"qcat/internal/concurrent"
	"qcat/internal/events"
	"qcat/internal/workflow"
)

// AutomationWorkflowEngine 自动化工作流引擎
// 统一管理和执行所有自动化任务
type AutomationWorkflowEngine struct {
	// 核心组件
	workflowEngine      *workflow.EnhancedWorkflowEngine
	automationScheduler *scheduler.AutomationScheduler
	poolManager         *concurrent.PoolManager
	eventBus            *events.EventBus

	// 配置
	config *WorkflowEngineConfig

	// 运行状态
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	isRunning bool
	runningMu sync.RWMutex

	// 统计信息
	stats   *WorkflowEngineStats
	statsMu sync.RWMutex

	// 任务管理
	activeTasks map[string]*ActiveTask
	tasksMu     sync.RWMutex
}

// WorkflowEngineConfig 工作流引擎配置
type WorkflowEngineConfig struct {
	MaxConcurrency      int           `yaml:"max_concurrency"`
	TaskTimeout         time.Duration `yaml:"task_timeout"`
	RetryAttempts       int           `yaml:"retry_attempts"`
	RetryDelay          time.Duration `yaml:"retry_delay"`
	EnableScheduler     bool          `yaml:"enable_scheduler"`
	EnableEventDriven   bool          `yaml:"enable_event_driven"`
	EnablePoolManager   bool          `yaml:"enable_pool_manager"`
	MonitorInterval     time.Duration `yaml:"monitor_interval"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`
	LogLevel            string        `yaml:"log_level"`
}

// WorkflowEngineStats 工作流引擎统计信息
type WorkflowEngineStats struct {
	TotalTasks           int           `json:"total_tasks"`
	CompletedTasks       int           `json:"completed_tasks"`
	FailedTasks          int           `json:"failed_tasks"`
	RunningTasks         int           `json:"running_tasks"`
	AverageExecutionTime time.Duration `json:"average_execution_time"`
	TotalExecutionTime   time.Duration `json:"total_execution_time"`
	StartTime            time.Time     `json:"start_time"`
	Uptime               time.Duration `json:"uptime"`
	LastTaskTime         time.Time     `json:"last_task_time"`
	TasksPerMinute       float64       `json:"tasks_per_minute"`
}

// ActiveTask 活跃任务
type ActiveTask struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Status    string                 `json:"status"`
	StartTime time.Time              `json:"start_time"`
	Progress  float64                `json:"progress"`
	Context   map[string]interface{} `json:"context"`
	Error     string                 `json:"error,omitempty"`
}

// NewAutomationWorkflowEngine 创建自动化工作流引擎
func NewAutomationWorkflowEngine(config *WorkflowEngineConfig) *AutomationWorkflowEngine {
	if config == nil {
		config = GetDefaultWorkflowEngineConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 创建事件总线
	eventBus := events.NewEventBus(&events.EventBusConfig{
		BufferSize: 1000,
		MaxRetries: 3,
		RetryDelay: time.Second,
	})

	// 创建工作流引擎
	workflowEngine := workflow.NewEnhancedWorkflowEngine(config.MaxConcurrency)

	// 创建池管理器
	var poolManager *concurrent.PoolManager
	if config.EnablePoolManager {
		poolConfig := concurrent.GetDefaultConfig()
		poolManager = concurrent.NewPoolManager(poolConfig)
	}

	// 创建自动化调度器（如果启用）
	var automationScheduler *scheduler.AutomationScheduler
	if config.EnableScheduler {
		// 创建一个基本的调度器配置
		// 注意：这里使用 nil 数据库，因为在测试环境中可能没有数据库
		automationScheduler = scheduler.NewAutomationSchedulerWithoutDB()
	}

	awe := &AutomationWorkflowEngine{
		workflowEngine:      workflowEngine,
		automationScheduler: automationScheduler,
		poolManager:         poolManager,
		eventBus:            eventBus,
		config:              config,
		ctx:                 ctx,
		cancel:              cancel,
		activeTasks:         make(map[string]*ActiveTask),
		stats: &WorkflowEngineStats{
			StartTime: time.Now(),
		},
	}

	// 初始化组件
	awe.initializeComponents()

	log.Printf("自动化工作流引擎已创建，配置: %+v", config)
	return awe
}

// GetDefaultWorkflowEngineConfig 获取默认配置
func GetDefaultWorkflowEngineConfig() *WorkflowEngineConfig {
	return &WorkflowEngineConfig{
		MaxConcurrency:      10,
		TaskTimeout:         30 * time.Minute,
		RetryAttempts:       3,
		RetryDelay:          5 * time.Second,
		EnableScheduler:     true,
		EnableEventDriven:   true,
		EnablePoolManager:   true,
		MonitorInterval:     30 * time.Second,
		HealthCheckInterval: 60 * time.Second,
		LogLevel:            "info",
	}
}

// initializeComponents 初始化组件
func (awe *AutomationWorkflowEngine) initializeComponents() {
	// 注册事件处理器
	awe.registerEventHandlers()

	// 初始化默认任务
	awe.initializeDefaultTasks()

	log.Println("工作流引擎组件初始化完成")
}

// registerEventHandlers 注册事件处理器
func (awe *AutomationWorkflowEngine) registerEventHandlers() {
	if !awe.config.EnableEventDriven {
		return
	}

	// 注册工作流事件处理器
	workflowHandler := &WorkflowEventHandler{engine: awe}
	awe.eventBus.Subscribe([]events.EventType{
		events.EventWorkflowStarted,
		events.EventWorkflowCompleted,
		events.EventWorkflowFailed,
		events.EventFunctionStarted,
		events.EventFunctionCompleted,
		events.EventFunctionFailed,
	}, workflowHandler, nil)

	log.Println("事件处理器注册完成")
}

// initializeDefaultTasks 初始化默认任务
func (awe *AutomationWorkflowEngine) initializeDefaultTasks() {
	log.Println("初始化默认任务完成")
}

// Start 启动工作流引擎
func (awe *AutomationWorkflowEngine) Start() error {
	awe.runningMu.Lock()
	defer awe.runningMu.Unlock()

	if awe.isRunning {
		return fmt.Errorf("workflow engine is already running")
	}

	log.Println("启动自动化工作流引擎...")

	// 事件总线在创建时自动启动

	// 启动池管理器
	if awe.poolManager != nil {
		if err := awe.poolManager.Start(); err != nil {
			return fmt.Errorf("failed to start pool manager: %w", err)
		}
		log.Println("池管理器已启动")
	}

	// 启动监控器
	awe.wg.Add(1)
	go awe.runMonitor()

	awe.isRunning = true
	awe.stats.StartTime = time.Now()

	log.Println("自动化工作流引擎启动完成")
	return nil
}

// Stop 停止工作流引擎
func (awe *AutomationWorkflowEngine) Stop() error {
	awe.runningMu.Lock()
	defer awe.runningMu.Unlock()

	if !awe.isRunning {
		return fmt.Errorf("workflow engine is not running")
	}

	log.Println("停止自动化工作流引擎...")

	// 取消上下文
	awe.cancel()

	// 停止池管理器
	if awe.poolManager != nil {
		if err := awe.poolManager.Stop(); err != nil {
			log.Printf("停止池管理器失败: %v", err)
		}
		log.Println("池管理器已停止")
	}

	// 任务调度器功能暂未实现

	// 停止事件总线
	awe.eventBus.Stop()
	log.Println("事件总线已停止")

	// 等待所有goroutine完成
	awe.wg.Wait()

	awe.isRunning = false

	log.Println("自动化工作流引擎停止完成")
	return nil
}

// ExecuteWorkflow 执行工作流
func (awe *AutomationWorkflowEngine) ExecuteWorkflow(ctx context.Context) error {
	log.Println("开始执行自动化工作流")

	// 创建活跃任务
	taskID := fmt.Sprintf("workflow_%d", time.Now().UnixNano())
	task := &ActiveTask{
		ID:        taskID,
		Name:      "自动化工作流执行",
		Type:      "workflow",
		Status:    "running",
		StartTime: time.Now(),
		Progress:  0.0,
		Context:   make(map[string]interface{}),
	}

	awe.tasksMu.Lock()
	awe.activeTasks[taskID] = task
	awe.tasksMu.Unlock()

	defer func() {
		awe.tasksMu.Lock()
		delete(awe.activeTasks, taskID)
		awe.tasksMu.Unlock()
	}()

	// 执行工作流
	err := awe.workflowEngine.ExecuteWorkflowWithEnhancements(ctx)

	// 更新任务状态
	awe.tasksMu.Lock()
	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
	} else {
		task.Status = "completed"
		task.Progress = 100.0
	}
	awe.tasksMu.Unlock()

	// 更新统计信息
	awe.updateStatsAfterTask(err == nil, time.Since(task.StartTime))

	if err != nil {
		log.Printf("工作流执行失败: %v", err)
		return err
	}

	log.Println("工作流执行完成")
	return nil
}

// executeWorkflow 执行工作流的任务处理器方法
func (awe *AutomationWorkflowEngine) executeWorkflow(ctx context.Context, task *scheduler.ScheduledTask) error {
	return awe.ExecuteWorkflow(ctx)
}

// performHealthCheck 执行健康检查
func (awe *AutomationWorkflowEngine) performHealthCheck() error {
	log.Println("执行系统健康检查")

	// 检查各个组件的健康状态
	var errors []string

	// 任务调度器检查暂未实现

	// 检查池管理器
	if awe.poolManager != nil {
		poolStats := awe.poolManager.GetStats()
		if poolStats == nil {
			errors = append(errors, "池管理器状态异常")
		}
	}

	// 检查工作流引擎
	if awe.workflowEngine == nil {
		errors = append(errors, "工作流引擎未初始化")
	}

	if len(errors) > 0 {
		return fmt.Errorf("健康检查失败: %v", errors)
	}

	log.Println("系统健康检查通过")
	return nil
}

// updateStats 更新统计信息的任务处理器方法
func (awe *AutomationWorkflowEngine) updateStats(ctx context.Context, task *scheduler.ScheduledTask) error {
	awe.updateStatsInternal()
	return nil
}

// updateStatsInternal 内部统计信息更新
func (awe *AutomationWorkflowEngine) updateStatsInternal() {
	awe.statsMu.Lock()
	defer awe.statsMu.Unlock()

	now := time.Now()
	awe.stats.Uptime = now.Sub(awe.stats.StartTime)

	// 计算任务执行速率
	if awe.stats.Uptime.Minutes() > 0 {
		awe.stats.TasksPerMinute = float64(awe.stats.CompletedTasks) / awe.stats.Uptime.Minutes()
	}

	// 计算平均执行时间
	if awe.stats.CompletedTasks > 0 {
		awe.stats.AverageExecutionTime = awe.stats.TotalExecutionTime / time.Duration(awe.stats.CompletedTasks)
	}

	// 更新运行中的任务数量
	awe.tasksMu.RLock()
	awe.stats.RunningTasks = len(awe.activeTasks)
	awe.tasksMu.RUnlock()
}

// updateStatsAfterTask 任务完成后更新统计信息
func (awe *AutomationWorkflowEngine) updateStatsAfterTask(success bool, duration time.Duration) {
	awe.statsMu.Lock()
	defer awe.statsMu.Unlock()

	awe.stats.TotalTasks++
	awe.stats.TotalExecutionTime += duration
	awe.stats.LastTaskTime = time.Now()

	if success {
		awe.stats.CompletedTasks++
	} else {
		awe.stats.FailedTasks++
	}
}

// runMonitor 运行监控器
func (awe *AutomationWorkflowEngine) runMonitor() {
	defer awe.wg.Done()

	ticker := time.NewTicker(awe.config.MonitorInterval)
	defer ticker.Stop()

	log.Println("工作流引擎监控器开始运行")

	for {
		select {
		case <-awe.ctx.Done():
			log.Println("工作流引擎监控器已停止")
			return
		case <-ticker.C:
			awe.performMonitoring()
		}
	}
}

// performMonitoring 执行监控
func (awe *AutomationWorkflowEngine) performMonitoring() {
	// 更新统计信息
	awe.updateStatsInternal()

	// 检查系统状态
	awe.checkSystemHealth()

	// 清理过期任务
	awe.cleanupExpiredTasks()
}

// checkSystemHealth 检查系统健康状态
func (awe *AutomationWorkflowEngine) checkSystemHealth() {
	// 检查活跃任务是否超时
	awe.tasksMu.RLock()
	now := time.Now()
	for _, task := range awe.activeTasks {
		if task.Status == "running" && now.Sub(task.StartTime) > awe.config.TaskTimeout {
			log.Printf("警告: 任务 %s 执行超时 (%v)", task.Name, now.Sub(task.StartTime))
		}
	}
	awe.tasksMu.RUnlock()

	// 检查统计信息异常
	awe.statsMu.RLock()
	if awe.stats.FailedTasks > 0 && awe.stats.CompletedTasks > 0 {
		failureRate := float64(awe.stats.FailedTasks) / float64(awe.stats.TotalTasks) * 100
		if failureRate > 20.0 { // 失败率超过20%
			log.Printf("警告: 任务失败率过高 (%.1f%%)", failureRate)
		}
	}
	awe.statsMu.RUnlock()
}

// cleanupExpiredTasks 清理过期任务
func (awe *AutomationWorkflowEngine) cleanupExpiredTasks() {
	awe.tasksMu.Lock()
	defer awe.tasksMu.Unlock()

	now := time.Now()
	expiredTasks := make([]string, 0)

	for id, task := range awe.activeTasks {
		// 清理超过1小时的已完成或失败任务
		if (task.Status == "completed" || task.Status == "failed") &&
			now.Sub(task.StartTime) > time.Hour {
			expiredTasks = append(expiredTasks, id)
		}
	}

	for _, id := range expiredTasks {
		delete(awe.activeTasks, id)
	}

	if len(expiredTasks) > 0 {
		log.Printf("清理了 %d 个过期任务", len(expiredTasks))
	}
}

// GetStats 获取统计信息
func (awe *AutomationWorkflowEngine) GetStats() *WorkflowEngineStats {
	awe.statsMu.RLock()
	defer awe.statsMu.RUnlock()

	// 创建副本
	stats := *awe.stats
	return &stats
}

// GetActiveTasks 获取活跃任务
func (awe *AutomationWorkflowEngine) GetActiveTasks() []*ActiveTask {
	awe.tasksMu.RLock()
	defer awe.tasksMu.RUnlock()

	tasks := make([]*ActiveTask, 0, len(awe.activeTasks))
	for _, task := range awe.activeTasks {
		// 创建副本
		taskCopy := *task
		tasks = append(tasks, &taskCopy)
	}

	return tasks
}

// IsRunning 检查引擎是否运行中
func (awe *AutomationWorkflowEngine) IsRunning() bool {
	awe.runningMu.RLock()
	defer awe.runningMu.RUnlock()

	return awe.isRunning
}

// 任务调度相关方法暂未实现

// GetWorkflowEngine 获取工作流引擎
func (awe *AutomationWorkflowEngine) GetWorkflowEngine() *workflow.EnhancedWorkflowEngine {
	return awe.workflowEngine
}

// GetTaskScheduler 获取任务调度器
func (awe *AutomationWorkflowEngine) GetTaskScheduler() interface{} {
	return awe.automationScheduler
}

// WorkflowEventHandler 工作流事件处理器
type WorkflowEventHandler struct {
	engine *AutomationWorkflowEngine
}

// Handle 处理事件
func (weh *WorkflowEventHandler) Handle(ctx context.Context, event *events.Event) error {
	switch event.Type {
	case events.EventWorkflowStarted:
		log.Printf("工作流开始: %v", event.Data)
	case events.EventWorkflowCompleted:
		log.Printf("工作流完成: %v", event.Data)
	case events.EventWorkflowFailed:
		log.Printf("工作流失败: %v", event.Data)
	case events.EventFunctionStarted:
		functionID := event.Data["function_id"]
		functionName := event.Data["function_name"]
		log.Printf("功能开始: %v - %v", functionID, functionName)
	case events.EventFunctionCompleted:
		functionID := event.Data["function_id"]
		functionName := event.Data["function_name"]
		duration := event.Data["duration"]
		log.Printf("功能完成: %v - %v (耗时: %v)", functionID, functionName, duration)
	case events.EventFunctionFailed:
		functionID := event.Data["function_id"]
		functionName := event.Data["function_name"]
		errorMsg := event.Data["error"]
		log.Printf("功能失败: %v - %v (错误: %v)", functionID, functionName, errorMsg)
	}

	return nil
}

// GetName 获取处理器名称
func (weh *WorkflowEventHandler) GetName() string {
	return "WorkflowEventHandler"
}

// GetEventTypes 获取事件类型
func (weh *WorkflowEventHandler) GetEventTypes() []events.EventType {
	return []events.EventType{
		events.EventWorkflowStarted,
		events.EventWorkflowCompleted,
		events.EventWorkflowFailed,
		events.EventFunctionStarted,
		events.EventFunctionCompleted,
		events.EventFunctionFailed,
	}
}

// GetPriority 获取处理器优先级
func (weh *WorkflowEventHandler) GetPriority() int {
	return 5
}

// GetPoolManager 获取池管理器
func (awe *AutomationWorkflowEngine) GetPoolManager() *concurrent.PoolManager {
	return awe.poolManager
}

// GetEventBus 获取事件总线
func (awe *AutomationWorkflowEngine) GetEventBus() *events.EventBus {
	return awe.eventBus
}
