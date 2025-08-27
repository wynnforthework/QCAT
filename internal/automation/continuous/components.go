package continuous

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"qcat/internal/config"
	"qcat/internal/database"
)

// OptimizationTask 优化任务
type OptimizationTask struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Priority    int                    `json:"priority"`
	MaxDuration time.Duration          `json:"max_duration"`
	Handler     func(context.Context) error `json:"-"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Status      string                 `json:"status"`
	Error       string                 `json:"error,omitempty"`
}

// TaskManager 任务管理器
type TaskManager struct {
	config          *ResourceManagementConfig
	tasks           chan *OptimizationTask
	activeTasks     map[string]*OptimizationTask
	activeTasksMutex sync.RWMutex
	workers         []*TaskWorker
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	running         bool
	runningMutex    sync.RWMutex
}

// TaskWorker 任务工作者
type TaskWorker struct {
	id          int
	taskManager *TaskManager
}

// NewTaskManager 创建任务管理器
func NewTaskManager(config *ResourceManagementConfig) (*TaskManager, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	tm := &TaskManager{
		config:      config,
		tasks:       make(chan *OptimizationTask, 100),
		activeTasks: make(map[string]*OptimizationTask),
		ctx:         ctx,
		cancel:      cancel,
		running:     false,
	}
	
	// 创建工作者
	for i := 0; i < config.MaxConcurrentTasks; i++ {
		worker := &TaskWorker{
			id:          i,
			taskManager: tm,
		}
		tm.workers = append(tm.workers, worker)
	}
	
	return tm, nil
}

// Start 启动任务管理器
func (tm *TaskManager) Start(ctx context.Context) error {
	tm.runningMutex.Lock()
	defer tm.runningMutex.Unlock()
	
	if tm.running {
		return fmt.Errorf("任务管理器已经在运行")
	}
	
	tm.running = true
	
	// 启动工作者
	for _, worker := range tm.workers {
		tm.wg.Add(1)
		go worker.run()
	}
	
	log.Printf("✅ 任务管理器已启动，工作者数量: %d", len(tm.workers))
	return nil
}

// Stop 停止任务管理器
func (tm *TaskManager) Stop() {
	tm.runningMutex.Lock()
	defer tm.runningMutex.Unlock()
	
	if !tm.running {
		return
	}
	
	tm.running = false
	tm.cancel()
	close(tm.tasks)
	tm.wg.Wait()
	
	log.Printf("✅ 任务管理器已停止")
}

// SubmitTask 提交任务
func (tm *TaskManager) SubmitTask(task *OptimizationTask) error {
	if task == nil {
		return fmt.Errorf("任务不能为空")
	}
	
	task.ID = fmt.Sprintf("task_%d_%d", time.Now().Unix(), len(tm.activeTasks))
	task.CreatedAt = time.Now()
	task.Status = "queued"
	
	select {
	case tm.tasks <- task:
		return nil
	default:
		return fmt.Errorf("任务队列已满")
	}
}

// CanExecuteTask 检查是否可以执行任务
func (tm *TaskManager) CanExecuteTask(taskType string) bool {
	// 检查资源使用情况
	resourceUsage := tm.GetResourceUsage()
	if resourceUsage > tm.config.MaxCPUUsage {
		return false
	}
	
	// 检查活跃任务数量
	tm.activeTasksMutex.RLock()
	activeCount := len(tm.activeTasks)
	tm.activeTasksMutex.RUnlock()
	
	return activeCount < tm.config.MaxConcurrentTasks
}

// GetResourceUsage 获取资源使用率
func (tm *TaskManager) GetResourceUsage() float64 {
	// 简化的资源使用率计算
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	// 基于内存使用和Goroutine数量估算CPU使用率
	goroutineCount := float64(runtime.NumGoroutine())
	memoryUsage := float64(memStats.Alloc) / float64(memStats.Sys) * 100
	
	// 简化的资源使用率计算
	resourceUsage := (goroutineCount/100.0*20.0 + memoryUsage) / 2.0
	if resourceUsage > 100.0 {
		resourceUsage = 100.0
	}
	
	return resourceUsage
}

// PauseLowPriorityTasks 暂停低优先级任务
func (tm *TaskManager) PauseLowPriorityTasks() {
	tm.activeTasksMutex.Lock()
	defer tm.activeTasksMutex.Unlock()
	
	for _, task := range tm.activeTasks {
		if task.Priority < 5 { // 优先级小于5的任务被认为是低优先级
			task.Status = "paused"
			log.Printf("⏸️ 暂停低优先级任务: %s", task.ID)
		}
	}
}

// run 工作者运行循环
func (tw *TaskWorker) run() {
	defer tw.taskManager.wg.Done()
	
	for {
		select {
		case task, ok := <-tw.taskManager.tasks:
			if !ok {
				return
			}
			tw.executeTask(task)
			
		case <-tw.taskManager.ctx.Done():
			return
		}
	}
}

// executeTask 执行任务
func (tw *TaskWorker) executeTask(task *OptimizationTask) {
	log.Printf("🔄 工作者 %d 开始执行任务: %s", tw.id, task.Type)
	
	// 添加到活跃任务
	tw.taskManager.activeTasksMutex.Lock()
	tw.taskManager.activeTasks[task.ID] = task
	tw.taskManager.activeTasksMutex.Unlock()
	
	// 更新任务状态
	now := time.Now()
	task.StartedAt = &now
	task.Status = "running"
	
	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(tw.taskManager.ctx, task.MaxDuration)
	defer cancel()
	
	// 执行任务
	err := task.Handler(ctx)
	
	// 更新任务状态
	completedAt := time.Now()
	task.CompletedAt = &completedAt
	
	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
		log.Printf("❌ 任务执行失败: %s, 错误: %v", task.Type, err)
	} else {
		task.Status = "completed"
		log.Printf("✅ 任务执行成功: %s", task.Type)
	}
	
	// 从活跃任务中移除
	tw.taskManager.activeTasksMutex.Lock()
	delete(tw.taskManager.activeTasks, task.ID)
	tw.taskManager.activeTasksMutex.Unlock()
}

// StrategyOptimizer 策略优化器
type StrategyOptimizer struct {
	config   *config.Config
	db       *database.DB
	optConfig *StrategyOptimizationConfig
	running  bool
	mutex    sync.RWMutex
}

// NewStrategyOptimizer 创建策略优化器
func NewStrategyOptimizer(config *config.Config, db *database.DB, optConfig *StrategyOptimizationConfig) (*StrategyOptimizer, error) {
	return &StrategyOptimizer{
		config:    config,
		db:        db,
		optConfig: optConfig,
		running:   false,
	}, nil
}

// Start 启动策略优化器
func (so *StrategyOptimizer) Start(ctx context.Context) error {
	so.mutex.Lock()
	defer so.mutex.Unlock()
	
	if so.running {
		return fmt.Errorf("策略优化器已经在运行")
	}
	
	so.running = true
	log.Printf("✅ 策略优化器已启动")
	return nil
}

// Stop 停止策略优化器
func (so *StrategyOptimizer) Stop() {
	so.mutex.Lock()
	defer so.mutex.Unlock()
	
	if !so.running {
		return
	}
	
	so.running = false
	log.Printf("✅ 策略优化器已停止")
}

// UpdateConfig 更新配置
func (so *StrategyOptimizer) UpdateConfig(config *StrategyOptimizationConfig) {
	so.mutex.Lock()
	defer so.mutex.Unlock()
	
	so.optConfig = config
}

// BacktestOptimizer 回测优化器
type BacktestOptimizer struct {
	config    *config.Config
	db        *database.DB
	optConfig *BacktestOptimizationConfig
	running   bool
	mutex     sync.RWMutex
}

// NewBacktestOptimizer 创建回测优化器
func NewBacktestOptimizer(config *config.Config, db *database.DB, optConfig *BacktestOptimizationConfig) (*BacktestOptimizer, error) {
	return &BacktestOptimizer{
		config:    config,
		db:        db,
		optConfig: optConfig,
		running:   false,
	}, nil
}

// Start 启动回测优化器
func (bo *BacktestOptimizer) Start(ctx context.Context) error {
	bo.mutex.Lock()
	defer bo.mutex.Unlock()
	
	if bo.running {
		return fmt.Errorf("回测优化器已经在运行")
	}
	
	bo.running = true
	log.Printf("✅ 回测优化器已启动")
	return nil
}

// Stop 停止回测优化器
func (bo *BacktestOptimizer) Stop() {
	bo.mutex.Lock()
	defer bo.mutex.Unlock()
	
	if !bo.running {
		return
	}
	
	bo.running = false
	log.Printf("✅ 回测优化器已停止")
}

// UpdateConfig 更新配置
func (bo *BacktestOptimizer) UpdateConfig(config *BacktestOptimizationConfig) {
	bo.mutex.Lock()
	defer bo.mutex.Unlock()
	
	bo.optConfig = config
}
