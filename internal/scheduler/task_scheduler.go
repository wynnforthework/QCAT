package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/events"

	"github.com/robfig/cron/v3"
)

// TaskScheduler 通用任务调度器
type TaskScheduler struct {
	// 定时任务调度器
	cronScheduler *cron.Cron

	// 事件总线
	eventBus *events.EventBus

	// 任务存储
	tasks   map[string]*ScheduledTask
	tasksMu sync.RWMutex

	// 任务处理器
	handlers   map[TaskType]TaskHandler
	handlersMu sync.RWMutex

	// 运行状态
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	isRunning bool
	runningMu sync.RWMutex

	// 配置
	config *SchedulerConfig

	// 统计信息
	stats   *SchedulerStats
	statsMu sync.RWMutex
}

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	MaxConcurrentTasks  int           `yaml:"max_concurrent_tasks"`
	TaskTimeout         time.Duration `yaml:"task_timeout"`
	RetryAttempts       int           `yaml:"retry_attempts"`
	RetryDelay          time.Duration `yaml:"retry_delay"`
	EnableEventDriven   bool          `yaml:"enable_event_driven"`
	EnableCronTasks     bool          `yaml:"enable_cron_tasks"`
	EnablePeriodicTasks bool          `yaml:"enable_periodic_tasks"`
	LogLevel            string        `yaml:"log_level"`
}

// ScheduledTask 调度任务
type ScheduledTask struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Type     TaskType     `json:"type"`
	Category TaskCategory `json:"category"`

	// 调度配置
	Schedule string        `json:"schedule"` // Cron表达式
	Interval time.Duration `json:"interval"` // 周期间隔
	Delay    time.Duration `json:"delay"`    // 延迟执行

	// 执行配置
	Handler    TaskHandler            `json:"-"`
	Context    map[string]interface{} `json:"context"`
	Priority   int                    `json:"priority"`
	Timeout    time.Duration          `json:"timeout"`
	MaxRetries int                    `json:"max_retries"`

	// 状态信息
	Status    TaskStatus `json:"status"`
	Enabled   bool       `json:"enabled"`
	LastRun   time.Time  `json:"last_run"`
	NextRun   time.Time  `json:"next_run"`
	RunCount  int        `json:"run_count"`
	FailCount int        `json:"fail_count"`

	// 事件驱动配置
	TriggerEvents []events.EventType `json:"trigger_events"`
	EventFilter   EventFilter        `json:"-"`

	// 时间戳
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 内部状态
	cronEntryID cron.EntryID       `json:"-"`
	cancelFunc  context.CancelFunc `json:"-"`
	isRunning   bool               `json:"-"`
	runningMu   sync.RWMutex       `json:"-"`
}

// TaskType 任务类型
type TaskType string

const (
	TaskTypeScheduled   TaskType = "scheduled"    // 定时任务
	TaskTypePeriodic    TaskType = "periodic"     // 周期任务
	TaskTypeEventDriven TaskType = "event_driven" // 事件驱动任务
	TaskTypeOneTime     TaskType = "one_time"     // 一次性任务
	TaskTypeConditional TaskType = "conditional"  // 条件任务
)

// TaskCategory 任务分类
type TaskCategory string

const (
	CategoryStrategy     TaskCategory = "strategy"
	CategoryRisk         TaskCategory = "risk"
	CategoryPosition     TaskCategory = "position"
	CategoryData         TaskCategory = "data"
	CategorySystem       TaskCategory = "system"
	CategoryMonitoring   TaskCategory = "monitoring"
	CategoryMaintenance  TaskCategory = "maintenance"
	CategoryOptimization TaskCategory = "optimization"
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
	TaskStatusSkipped   TaskStatus = "skipped"
	TaskStatusDisabled  TaskStatus = "disabled"
)

// TaskHandler 任务处理器接口
type TaskHandler interface {
	Execute(ctx context.Context, task *ScheduledTask) error
	GetName() string
	GetDescription() string
}

// EventFilter 事件过滤器
type EventFilter = events.EventFilter

// SchedulerStats 调度器统计信息
type SchedulerStats struct {
	TotalTasks       int           `json:"total_tasks"`
	RunningTasks     int           `json:"running_tasks"`
	CompletedTasks   int           `json:"completed_tasks"`
	FailedTasks      int           `json:"failed_tasks"`
	ScheduledTasks   int           `json:"scheduled_tasks"`
	EventDrivenTasks int           `json:"event_driven_tasks"`
	PeriodicTasks    int           `json:"periodic_tasks"`
	AverageRunTime   time.Duration `json:"average_run_time"`
	LastRunTime      time.Time     `json:"last_run_time"`
	Uptime           time.Duration `json:"uptime"`
	StartTime        time.Time     `json:"start_time"`
}

// NewTaskScheduler 创建新的任务调度器
func NewTaskScheduler(config *SchedulerConfig, eventBus *events.EventBus) *TaskScheduler {
	if config == nil {
		config = GetDefaultSchedulerConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 创建cron调度器
	cronScheduler := cron.New(cron.WithSeconds())

	ts := &TaskScheduler{
		cronScheduler: cronScheduler,
		eventBus:      eventBus,
		tasks:         make(map[string]*ScheduledTask),
		handlers:      make(map[TaskType]TaskHandler),
		ctx:           ctx,
		cancel:        cancel,
		config:        config,
		stats: &SchedulerStats{
			StartTime: time.Now(),
		},
	}

	// 如果启用事件驱动，订阅事件
	if config.EnableEventDriven && eventBus != nil {
		ts.subscribeToEvents()
	}

	log.Printf("任务调度器已创建，配置: %+v", config)
	return ts
}

// GetDefaultSchedulerConfig 获取默认调度器配置
func GetDefaultSchedulerConfig() *SchedulerConfig {
	return &SchedulerConfig{
		MaxConcurrentTasks:  10,
		TaskTimeout:         30 * time.Minute,
		RetryAttempts:       3,
		RetryDelay:          5 * time.Second,
		EnableEventDriven:   true,
		EnableCronTasks:     true,
		EnablePeriodicTasks: true,
		LogLevel:            "info",
	}
}

// Start 启动调度器
func (ts *TaskScheduler) Start() error {
	ts.runningMu.Lock()
	defer ts.runningMu.Unlock()

	if ts.isRunning {
		return fmt.Errorf("scheduler is already running")
	}

	log.Println("启动任务调度器...")

	// 启动cron调度器
	if ts.config.EnableCronTasks {
		ts.cronScheduler.Start()
		log.Println("Cron调度器已启动")
	}

	// 启动周期任务处理器
	if ts.config.EnablePeriodicTasks {
		ts.wg.Add(1)
		go ts.runPeriodicTaskProcessor()
		log.Println("周期任务处理器已启动")
	}

	// 启动统计更新器
	ts.wg.Add(1)
	go ts.runStatsUpdater()

	ts.isRunning = true
	ts.stats.StartTime = time.Now()

	log.Println("任务调度器启动完成")
	return nil
}

// Stop 停止调度器
func (ts *TaskScheduler) Stop() error {
	ts.runningMu.Lock()
	defer ts.runningMu.Unlock()

	if !ts.isRunning {
		return fmt.Errorf("scheduler is not running")
	}

	log.Println("停止任务调度器...")

	// 取消上下文
	ts.cancel()

	// 停止cron调度器
	if ts.cronScheduler != nil {
		ctx := ts.cronScheduler.Stop()
		<-ctx.Done()
		log.Println("Cron调度器已停止")
	}

	// 等待所有goroutine完成
	ts.wg.Wait()

	ts.isRunning = false

	log.Println("任务调度器停止完成")
	return nil
}

// AddTask 添加任务
func (ts *TaskScheduler) AddTask(task *ScheduledTask) error {
	if task.ID == "" {
		task.ID = fmt.Sprintf("task_%d", time.Now().UnixNano())
	}

	if task.Handler == nil {
		return fmt.Errorf("task handler is required")
	}

	// 设置默认值
	if task.Timeout == 0 {
		task.Timeout = ts.config.TaskTimeout
	}
	if task.MaxRetries == 0 {
		task.MaxRetries = ts.config.RetryAttempts
	}
	if task.Priority == 0 {
		task.Priority = 5 // 默认优先级
	}

	task.Status = TaskStatusPending
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()

	ts.tasksMu.Lock()
	ts.tasks[task.ID] = task
	ts.tasksMu.Unlock()

	// 根据任务类型进行调度
	switch task.Type {
	case TaskTypeScheduled:
		if err := ts.scheduleTask(task); err != nil {
			return fmt.Errorf("failed to schedule task: %w", err)
		}
	case TaskTypePeriodic:
		if err := ts.schedulePeriodicTask(task); err != nil {
			return fmt.Errorf("failed to schedule periodic task: %w", err)
		}
	case TaskTypeEventDriven:
		if err := ts.subscribeTaskToEvents(task); err != nil {
			return fmt.Errorf("failed to subscribe task to events: %w", err)
		}
	case TaskTypeOneTime:
		if err := ts.scheduleOneTimeTask(task); err != nil {
			return fmt.Errorf("failed to schedule one-time task: %w", err)
		}
	}

	ts.updateStats()

	log.Printf("任务已添加: %s (%s) - %s", task.Name, task.ID, task.Type)
	return nil
}

// scheduleTask 调度定时任务
func (ts *TaskScheduler) scheduleTask(task *ScheduledTask) error {
	if task.Schedule == "" {
		return fmt.Errorf("schedule is required for scheduled task")
	}

	entryID, err := ts.cronScheduler.AddFunc(task.Schedule, func() {
		ts.executeTask(task)
	})
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	task.cronEntryID = entryID
	task.NextRun = ts.cronScheduler.Entry(entryID).Next

	log.Printf("定时任务已调度: %s, 下次执行: %v", task.Name, task.NextRun)
	return nil
}

// schedulePeriodicTask 调度周期任务
func (ts *TaskScheduler) schedulePeriodicTask(task *ScheduledTask) error {
	if task.Interval == 0 {
		return fmt.Errorf("interval is required for periodic task")
	}

	// 计算下次执行时间
	task.NextRun = time.Now().Add(task.Delay)
	task.Status = TaskStatusPending // 确保状态为pending

	log.Printf("周期任务已调度: %s, 间隔: %v, 下次执行: %v", task.Name, task.Interval, task.NextRun)
	return nil
}

// scheduleOneTimeTask 调度一次性任务
func (ts *TaskScheduler) scheduleOneTimeTask(task *ScheduledTask) error {
	// 计算执行时间
	executeAt := time.Now().Add(task.Delay)
	task.NextRun = executeAt

	// 创建定时器
	go func() {
		timer := time.NewTimer(task.Delay)
		defer timer.Stop()

		select {
		case <-timer.C:
			ts.executeTask(task)
		case <-ts.ctx.Done():
			return
		}
	}()

	log.Printf("一次性任务已调度: %s, 执行时间: %v", task.Name, executeAt)
	return nil
}

// subscribeTaskToEvents 订阅任务到事件
func (ts *TaskScheduler) subscribeTaskToEvents(task *ScheduledTask) error {
	if ts.eventBus == nil {
		return fmt.Errorf("event bus is not available")
	}

	if len(task.TriggerEvents) == 0 {
		return fmt.Errorf("trigger events are required for event-driven task")
	}

	// 创建事件处理器
	handler := &TaskEventHandler{
		task:      task,
		scheduler: ts,
	}

	// 订阅事件
	_, err := ts.eventBus.Subscribe(task.TriggerEvents, handler, task.EventFilter)
	if err != nil {
		return fmt.Errorf("failed to subscribe to events: %w", err)
	}

	log.Printf("事件驱动任务已订阅: %s, 触发事件: %v", task.Name, task.TriggerEvents)
	return nil
}

// executeTask 执行任务
func (ts *TaskScheduler) executeTask(task *ScheduledTask) {
	// 检查任务是否已在运行
	task.runningMu.Lock()
	if task.isRunning {
		task.runningMu.Unlock()
		log.Printf("任务 %s 已在运行中，跳过此次执行", task.Name)
		return
	}
	task.isRunning = true
	task.runningMu.Unlock()

	defer func() {
		task.runningMu.Lock()
		task.isRunning = false
		task.runningMu.Unlock()
	}()

	// 检查任务是否启用
	if !task.Enabled {
		log.Printf("任务 %s 已禁用，跳过执行", task.Name)
		task.Status = TaskStatusSkipped
		return
	}

	log.Printf("开始执行任务: %s (%s)", task.Name, task.ID)

	// 更新任务状态
	task.Status = TaskStatusRunning
	task.LastRun = time.Now()
	task.RunCount++

	// 创建执行上下文
	ctx, cancel := context.WithTimeout(ts.ctx, task.Timeout)
	defer cancel()

	// 执行任务
	startTime := time.Now()
	err := task.Handler.Execute(ctx, task)
	duration := time.Since(startTime)

	// 更新任务状态
	if err != nil {
		task.Status = TaskStatusFailed
		task.FailCount++
		log.Printf("任务执行失败: %s, 错误: %v, 耗时: %v", task.Name, err, duration)

		// 重试逻辑
		if task.FailCount < task.MaxRetries {
			log.Printf("任务 %s 将在 %v 后重试 (第 %d/%d 次)",
				task.Name, ts.config.RetryDelay, task.FailCount, task.MaxRetries)

			go func() {
				time.Sleep(ts.config.RetryDelay)
				ts.executeTask(task)
			}()
		}
	} else {
		task.Status = TaskStatusCompleted
		log.Printf("任务执行成功: %s, 耗时: %v", task.Name, duration)
	}

	task.UpdatedAt = time.Now()

	// 更新统计信息
	ts.updateStats()

	// 如果是周期任务，计算下次执行时间
	if task.Type == TaskTypePeriodic && task.Enabled {
		task.NextRun = time.Now().Add(task.Interval)
		task.Status = TaskStatusPending
	}
}

// TaskEventHandler 任务事件处理器
type TaskEventHandler struct {
	task      *ScheduledTask
	scheduler *TaskScheduler
}

// Handle 处理事件
func (teh *TaskEventHandler) Handle(ctx context.Context, event *events.Event) error {
	// 应用事件过滤器
	if teh.task.EventFilter != nil && !teh.task.EventFilter(event) {
		return nil
	}

	log.Printf("事件触发任务: %s, 事件类型: %s", teh.task.Name, event.Type)

	// 执行任务
	go teh.scheduler.executeTask(teh.task)

	return nil
}

// GetName 获取处理器名称
func (teh *TaskEventHandler) GetName() string {
	return fmt.Sprintf("TaskEventHandler-%s", teh.task.ID)
}

// GetEventTypes 获取事件类型
func (teh *TaskEventHandler) GetEventTypes() []events.EventType {
	return teh.task.TriggerEvents
}

// GetPriority 获取处理器优先级
func (teh *TaskEventHandler) GetPriority() int {
	return teh.task.Priority
}

// subscribeToEvents 订阅事件
func (ts *TaskScheduler) subscribeToEvents() {
	// 订阅一些通用事件
	if ts.eventBus == nil {
		log.Println("警告: 事件总线未初始化，无法订阅事件")
		return
	}

	// 创建通用事件处理器
	systemEventHandler := &SystemEventHandler{scheduler: ts}

	// 订阅系统相关事件
	systemEvents := []events.EventType{
		events.EventSystemAlert,    // 系统警报
		events.EventSystemError,    // 系统错误
		events.EventSystemRecovery, // 系统恢复
	}

	if _, err := ts.eventBus.Subscribe(systemEvents, systemEventHandler, nil); err != nil {
		log.Printf("订阅系统事件失败: %v", err)
	} else {
		log.Printf("已订阅系统事件: %v", systemEvents)
	}

	// 创建工作流事件处理器
	workflowEventHandler := &WorkflowEventHandler{scheduler: ts}

	// 订阅工作流相关事件
	workflowEvents := []events.EventType{
		events.EventWorkflowStarted,   // 工作流开始
		events.EventWorkflowCompleted, // 工作流完成
		events.EventWorkflowFailed,    // 工作流失败
	}

	if _, err := ts.eventBus.Subscribe(workflowEvents, workflowEventHandler, nil); err != nil {
		log.Printf("订阅工作流事件失败: %v", err)
	} else {
		log.Printf("已订阅工作流事件: %v", workflowEvents)
	}

	// 创建资源事件处理器
	resourceEventHandler := &ResourceEventHandler{scheduler: ts}

	// 订阅资源相关事件
	resourceEvents := []events.EventType{
		events.EventResourceAcquired,  // 资源获取
		events.EventResourceReleased,  // 资源释放
		events.EventResourceExhausted, // 资源耗尽
	}

	if _, err := ts.eventBus.Subscribe(resourceEvents, resourceEventHandler, nil); err != nil {
		log.Printf("订阅资源事件失败: %v", err)
	} else {
		log.Printf("已订阅资源事件: %v", resourceEvents)
	}

	log.Println("事件驱动功能已启用，已订阅通用系统事件")
}

// runPeriodicTaskProcessor 运行周期任务处理器
func (ts *TaskScheduler) runPeriodicTaskProcessor() {
	defer ts.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond) // 每100毫秒检查一次，更频繁
	defer ticker.Stop()

	log.Println("周期任务处理器开始运行")

	for {
		select {
		case <-ts.ctx.Done():
			log.Println("周期任务处理器已停止")
			return
		case <-ticker.C:
			ts.checkPeriodicTasks()
		}
	}
}

// checkPeriodicTasks 检查周期任务
func (ts *TaskScheduler) checkPeriodicTasks() {
	ts.tasksMu.RLock()
	tasks := make([]*ScheduledTask, 0)
	for _, task := range ts.tasks {
		if task.Type == TaskTypePeriodic && task.Enabled {
			tasks = append(tasks, task)
		}
	}
	ts.tasksMu.RUnlock()

	now := time.Now()
	for _, task := range tasks {
		if task.Status == TaskStatusPending && now.After(task.NextRun) {
			go ts.executeTask(task)
		}
	}
}

// runStatsUpdater 运行统计更新器
func (ts *TaskScheduler) runStatsUpdater() {
	defer ts.wg.Done()

	ticker := time.NewTicker(10 * time.Second) // 每10秒更新一次统计
	defer ticker.Stop()

	for {
		select {
		case <-ts.ctx.Done():
			return
		case <-ticker.C:
			ts.updateStats()
		}
	}
}

// updateStats 更新统计信息
func (ts *TaskScheduler) updateStats() {
	ts.statsMu.Lock()
	defer ts.statsMu.Unlock()

	ts.tasksMu.RLock()
	defer ts.tasksMu.RUnlock()

	stats := &SchedulerStats{
		StartTime: ts.stats.StartTime,
		Uptime:    time.Since(ts.stats.StartTime),
	}

	for _, task := range ts.tasks {
		stats.TotalTasks++

		switch task.Status {
		case TaskStatusRunning:
			stats.RunningTasks++
		case TaskStatusCompleted:
			stats.CompletedTasks++
		case TaskStatusFailed:
			stats.FailedTasks++
		}

		switch task.Type {
		case TaskTypeScheduled:
			stats.ScheduledTasks++
		case TaskTypeEventDriven:
			stats.EventDrivenTasks++
		case TaskTypePeriodic:
			stats.PeriodicTasks++
		}

		if !task.LastRun.IsZero() && task.LastRun.After(stats.LastRunTime) {
			stats.LastRunTime = task.LastRun
		}
	}

	ts.stats = stats
}

// GetTask 获取任务
func (ts *TaskScheduler) GetTask(taskID string) (*ScheduledTask, error) {
	ts.tasksMu.RLock()
	defer ts.tasksMu.RUnlock()

	task, exists := ts.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return task, nil
}

// GetAllTasks 获取所有任务
func (ts *TaskScheduler) GetAllTasks() []*ScheduledTask {
	ts.tasksMu.RLock()
	defer ts.tasksMu.RUnlock()

	tasks := make([]*ScheduledTask, 0, len(ts.tasks))
	for _, task := range ts.tasks {
		tasks = append(tasks, task)
	}

	return tasks
}

// EnableTask 启用任务
func (ts *TaskScheduler) EnableTask(taskID string) error {
	ts.tasksMu.Lock()
	defer ts.tasksMu.Unlock()

	task, exists := ts.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	task.Enabled = true
	task.UpdatedAt = time.Now()

	log.Printf("任务已启用: %s", task.Name)
	return nil
}

// DisableTask 禁用任务
func (ts *TaskScheduler) DisableTask(taskID string) error {
	ts.tasksMu.Lock()
	defer ts.tasksMu.Unlock()

	task, exists := ts.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	task.Enabled = false
	task.UpdatedAt = time.Now()

	log.Printf("任务已禁用: %s", task.Name)
	return nil
}

// RemoveTask 移除任务
func (ts *TaskScheduler) RemoveTask(taskID string) error {
	ts.tasksMu.Lock()
	defer ts.tasksMu.Unlock()

	task, exists := ts.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// 如果是定时任务，从cron中移除
	if task.Type == TaskTypeScheduled && task.cronEntryID != 0 {
		ts.cronScheduler.Remove(task.cronEntryID)
	}

	// 取消任务
	if task.cancelFunc != nil {
		task.cancelFunc()
	}

	delete(ts.tasks, taskID)

	log.Printf("任务已移除: %s", task.Name)
	return nil
}

// GetStats 获取统计信息
func (ts *TaskScheduler) GetStats() *SchedulerStats {
	ts.statsMu.RLock()
	defer ts.statsMu.RUnlock()

	// 创建副本
	stats := *ts.stats
	return &stats
}

// IsRunning 检查调度器是否运行中
func (ts *TaskScheduler) IsRunning() bool {
	ts.runningMu.RLock()
	defer ts.runningMu.RUnlock()

	return ts.isRunning
}

// SystemEventHandler 系统事件处理器
type SystemEventHandler struct {
	scheduler *TaskScheduler
}

// Handle 处理系统事件
func (seh *SystemEventHandler) Handle(ctx context.Context, event *events.Event) error {
	log.Printf("处理系统事件: %s, 来源: %s", event.Type, event.Source)

	switch event.Type {
	case events.EventSystemAlert:
		// 系统警报事件 - 可能需要触发监控任务
		return seh.handleSystemAlert(ctx, event)
	case events.EventSystemError:
		// 系统错误事件 - 可能需要触发错误处理任务
		return seh.handleSystemError(ctx, event)
	case events.EventSystemRecovery:
		// 系统恢复事件 - 可能需要重新启动暂停的任务
		return seh.handleSystemRecovery(ctx, event)
	}

	return nil
}

// handleSystemAlert 处理系统警报
func (seh *SystemEventHandler) handleSystemAlert(ctx context.Context, event *events.Event) error {
	log.Printf("处理系统警报: %+v", event.Data)

	// 可以在这里触发相关的监控或诊断任务
	// 例如：触发系统健康检查任务

	return nil
}

// handleSystemError 处理系统错误
func (seh *SystemEventHandler) handleSystemError(ctx context.Context, event *events.Event) error {
	log.Printf("处理系统错误: %+v", event.Data)

	// 可以在这里触发错误恢复任务
	// 例如：暂停某些任务，启动错误诊断任务

	return nil
}

// handleSystemRecovery 处理系统恢复
func (seh *SystemEventHandler) handleSystemRecovery(ctx context.Context, event *events.Event) error {
	log.Printf("处理系统恢复: %+v", event.Data)

	// 可以在这里重新启动之前暂停的任务
	// 例如：恢复被暂停的定时任务

	return nil
}

// GetName 获取处理器名称
func (seh *SystemEventHandler) GetName() string {
	return "SystemEventHandler"
}

// GetEventTypes 获取事件类型
func (seh *SystemEventHandler) GetEventTypes() []events.EventType {
	return []events.EventType{
		events.EventSystemAlert,
		events.EventSystemError,
		events.EventSystemRecovery,
	}
}

// GetPriority 获取处理器优先级
func (seh *SystemEventHandler) GetPriority() int {
	return int(events.PriorityHigh)
}

// WorkflowEventHandler 工作流事件处理器
type WorkflowEventHandler struct {
	scheduler *TaskScheduler
}

// Handle 处理工作流事件
func (weh *WorkflowEventHandler) Handle(ctx context.Context, event *events.Event) error {
	log.Printf("处理工作流事件: %s, 来源: %s", event.Type, event.Source)

	switch event.Type {
	case events.EventWorkflowStarted:
		// 工作流开始事件
		return weh.handleWorkflowStarted(ctx, event)
	case events.EventWorkflowCompleted:
		// 工作流完成事件
		return weh.handleWorkflowCompleted(ctx, event)
	case events.EventWorkflowFailed:
		// 工作流失败事件
		return weh.handleWorkflowFailed(ctx, event)
	}

	return nil
}

// handleWorkflowStarted 处理工作流开始
func (weh *WorkflowEventHandler) handleWorkflowStarted(ctx context.Context, event *events.Event) error {
	log.Printf("工作流开始: %+v", event.Data)

	// 可以在这里记录工作流开始时间，更新统计信息
	weh.scheduler.statsMu.Lock()
	weh.scheduler.stats.RunningTasks++
	weh.scheduler.statsMu.Unlock()

	return nil
}

// handleWorkflowCompleted 处理工作流完成
func (weh *WorkflowEventHandler) handleWorkflowCompleted(ctx context.Context, event *events.Event) error {
	log.Printf("工作流完成: %+v", event.Data)

	// 更新统计信息
	weh.scheduler.statsMu.Lock()
	weh.scheduler.stats.RunningTasks--
	weh.scheduler.stats.CompletedTasks++
	weh.scheduler.statsMu.Unlock()

	return nil
}

// handleWorkflowFailed 处理工作流失败
func (weh *WorkflowEventHandler) handleWorkflowFailed(ctx context.Context, event *events.Event) error {
	log.Printf("工作流失败: %+v", event.Data)

	// 更新统计信息
	weh.scheduler.statsMu.Lock()
	weh.scheduler.stats.RunningTasks--
	weh.scheduler.stats.FailedTasks++
	weh.scheduler.statsMu.Unlock()

	// 可以在这里触发重试逻辑或错误处理任务

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
	}
}

// GetPriority 获取处理器优先级
func (weh *WorkflowEventHandler) GetPriority() int {
	return int(events.PriorityNormal)
}

// ResourceEventHandler 资源事件处理器
type ResourceEventHandler struct {
	scheduler *TaskScheduler
}

// Handle 处理资源事件
func (reh *ResourceEventHandler) Handle(ctx context.Context, event *events.Event) error {
	log.Printf("处理资源事件: %s, 来源: %s", event.Type, event.Source)

	switch event.Type {
	case events.EventResourceAcquired:
		// 资源获取事件
		return reh.handleResourceAcquired(ctx, event)
	case events.EventResourceReleased:
		// 资源释放事件
		return reh.handleResourceReleased(ctx, event)
	case events.EventResourceExhausted:
		// 资源耗尽事件
		return reh.handleResourceExhausted(ctx, event)
	}

	return nil
}

// handleResourceAcquired 处理资源获取
func (reh *ResourceEventHandler) handleResourceAcquired(ctx context.Context, event *events.Event) error {
	log.Printf("资源已获取: %+v", event.Data)

	// 可以在这里记录资源使用情况
	// 例如：更新资源使用统计，检查是否可以启动更多任务

	return nil
}

// handleResourceReleased 处理资源释放
func (reh *ResourceEventHandler) handleResourceReleased(ctx context.Context, event *events.Event) error {
	log.Printf("资源已释放: %+v", event.Data)

	// 可以在这里检查是否有等待资源的任务可以启动
	// 例如：检查任务队列，启动等待中的任务

	return nil
}

// handleResourceExhausted 处理资源耗尽
func (reh *ResourceEventHandler) handleResourceExhausted(ctx context.Context, event *events.Event) error {
	log.Printf("资源耗尽: %+v", event.Data)

	// 可以在这里暂停低优先级任务，或者触发资源清理任务
	// 例如：暂停非关键任务，启动资源清理任务

	return nil
}

// GetName 获取处理器名称
func (reh *ResourceEventHandler) GetName() string {
	return "ResourceEventHandler"
}

// GetEventTypes 获取事件类型
func (reh *ResourceEventHandler) GetEventTypes() []events.EventType {
	return []events.EventType{
		events.EventResourceAcquired,
		events.EventResourceReleased,
		events.EventResourceExhausted,
	}
}

// GetPriority 获取处理器优先级
func (reh *ResourceEventHandler) GetPriority() int {
	return int(events.PriorityNormal)
}
