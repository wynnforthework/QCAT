package concurrent

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Task 任务接口
type Task interface {
	Execute(ctx context.Context) error
	GetID() string
	GetPriority() int
	GetTimeout() time.Duration
}

// TaskResult 任务执行结果
type TaskResult struct {
	TaskID    string        `json:"task_id"`
	Success   bool          `json:"success"`
	Error     string        `json:"error,omitempty"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
	WorkerID  int           `json:"worker_id"`
}

// Worker 工作者
type Worker struct {
	ID       int
	pool     *GoroutinePool
	taskChan chan Task
	quit     chan bool
	wg       *sync.WaitGroup
}

// GoroutinePool Goroutine池
type GoroutinePool struct {
	workers    []*Worker
	taskQueue  chan Task
	resultChan chan *TaskResult
	wg         sync.WaitGroup
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc

	// 配置
	maxWorkers int
	queueSize  int

	// 统计信息
	totalTasks     int64
	completedTasks int64
	failedTasks    int64
	activeTasks    int64

	// 监控
	startTime    time.Time
	lastActivity time.Time
}

// PoolConfig 池配置
type PoolConfig struct {
	MinWorkers      int
	MaxWorkers      int
	QueueSize       int
	IdleTimeout     time.Duration
	TaskTimeout     time.Duration
	WorkerTimeout   time.Duration
	EnableMetrics   bool
	EnableProfiling bool
}

// NewGoroutinePool 创建Goroutine池
func NewGoroutinePool(config *PoolConfig) *GoroutinePool {
	if config.MaxWorkers <= 0 {
		config.MaxWorkers = runtime.NumCPU()
	}
	if config.QueueSize <= 0 {
		config.QueueSize = config.MaxWorkers * 10
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &GoroutinePool{
		maxWorkers:   config.MaxWorkers,
		queueSize:    config.QueueSize,
		taskQueue:    make(chan Task, config.QueueSize),
		resultChan:   make(chan *TaskResult, config.QueueSize),
		ctx:          ctx,
		cancel:       cancel,
		startTime:    time.Now(),
		lastActivity: time.Now(),
	}

	// 创建工作者
	pool.workers = make([]*Worker, config.MaxWorkers)
	for i := 0; i < config.MaxWorkers; i++ {
		worker := &Worker{
			ID:       i + 1,
			pool:     pool,
			taskChan: make(chan Task),
			quit:     make(chan bool),
			wg:       &pool.wg,
		}
		pool.workers[i] = worker
	}

	log.Printf("创建Goroutine池: %d个工作者, 队列大小: %d", config.MaxWorkers, config.QueueSize)
	return pool
}

// Start 启动池
func (gp *GoroutinePool) Start() {
	log.Println("启动Goroutine池...")

	// 启动任务分发器
	go gp.dispatcher()

	// 启动工作者
	for _, worker := range gp.workers {
		gp.wg.Add(1)
		go worker.start()
	}

	// 启动结果收集器
	go gp.resultCollector()

	log.Printf("Goroutine池已启动，%d个工作者就绪", len(gp.workers))
}

// Stop 停止池
func (gp *GoroutinePool) Stop() {
	log.Println("停止Goroutine池...")

	// 取消上下文
	gp.cancel()

	// 关闭任务队列
	close(gp.taskQueue)

	// 停止所有工作者
	for _, worker := range gp.workers {
		worker.quit <- true
	}

	// 等待所有工作者完成
	gp.wg.Wait()

	// 关闭结果通道
	close(gp.resultChan)

	log.Println("Goroutine池已停止")
}

// Submit 提交任务
func (gp *GoroutinePool) Submit(task Task) error {
	select {
	case gp.taskQueue <- task:
		atomic.AddInt64(&gp.totalTasks, 1)
		gp.mu.Lock()
		gp.lastActivity = time.Now()
		gp.mu.Unlock()
		return nil
	case <-gp.ctx.Done():
		return fmt.Errorf("pool is shutting down")
	default:
		return fmt.Errorf("task queue is full")
	}
}

// dispatcher 任务分发器
func (gp *GoroutinePool) dispatcher() {
	for {
		select {
		case task := <-gp.taskQueue:
			// 找到空闲的工作者
			go func(t Task) {
				for _, worker := range gp.workers {
					select {
					case worker.taskChan <- t:
						return
					default:
						continue
					}
				}
				// 如果所有工作者都忙，重新放回队列
				select {
				case gp.taskQueue <- t:
				default:
					log.Printf("任务 %s 被丢弃，队列已满", t.GetID())
				}
			}(task)
		case <-gp.ctx.Done():
			return
		}
	}
}

// start 启动工作者
func (w *Worker) start() {
	defer w.wg.Done()

	log.Printf("工作者 %d 已启动", w.ID)

	for {
		select {
		case task := <-w.taskChan:
			w.executeTask(task)
		case <-w.quit:
			log.Printf("工作者 %d 已停止", w.ID)
			return
		case <-w.pool.ctx.Done():
			log.Printf("工作者 %d 收到停止信号", w.ID)
			return
		}
	}
}

// executeTask 执行任务
func (w *Worker) executeTask(task Task) {
	atomic.AddInt64(&w.pool.activeTasks, 1)
	defer atomic.AddInt64(&w.pool.activeTasks, -1)

	startTime := time.Now()

	log.Printf("工作者 %d 开始执行任务 %s", w.ID, task.GetID())

	// 创建带超时的上下文
	timeout := task.GetTimeout()
	if timeout <= 0 {
		timeout = 30 * time.Second // 默认超时
	}

	ctx, cancel := context.WithTimeout(w.pool.ctx, timeout)
	defer cancel()

	// 执行任务
	err := task.Execute(ctx)
	endTime := time.Now()
	duration := endTime.Sub(startTime)

	// 创建结果
	result := &TaskResult{
		TaskID:    task.GetID(),
		Success:   err == nil,
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  duration,
		WorkerID:  w.ID,
	}

	if err != nil {
		result.Error = err.Error()
		atomic.AddInt64(&w.pool.failedTasks, 1)
		log.Printf("工作者 %d 任务 %s 执行失败: %v (耗时: %v)", w.ID, task.GetID(), err, duration)
	} else {
		atomic.AddInt64(&w.pool.completedTasks, 1)
		log.Printf("工作者 %d 任务 %s 执行成功 (耗时: %v)", w.ID, task.GetID(), duration)
	}

	// 发送结果
	select {
	case w.pool.resultChan <- result:
	default:
		log.Printf("结果通道已满，丢弃任务 %s 的结果", task.GetID())
	}
}

// resultCollector 结果收集器
func (gp *GoroutinePool) resultCollector() {
	for result := range gp.resultChan {
		// 这里可以添加结果处理逻辑
		// 比如记录到数据库、发送通知等
		log.Printf("收集到任务结果: %s (成功: %t, 耗时: %v)",
			result.TaskID, result.Success, result.Duration)
	}
}

// GetStats 获取统计信息
func (gp *GoroutinePool) GetStats() map[string]interface{} {
	gp.mu.RLock()
	defer gp.mu.RUnlock()

	return map[string]interface{}{
		"max_workers":     gp.maxWorkers,
		"queue_size":      gp.queueSize,
		"total_tasks":     atomic.LoadInt64(&gp.totalTasks),
		"completed_tasks": atomic.LoadInt64(&gp.completedTasks),
		"failed_tasks":    atomic.LoadInt64(&gp.failedTasks),
		"active_tasks":    atomic.LoadInt64(&gp.activeTasks),
		"queue_length":    len(gp.taskQueue),
		"uptime":          time.Since(gp.startTime).String(),
		"last_activity":   gp.lastActivity,
	}
}

// GetQueueLength 获取队列长度
func (gp *GoroutinePool) GetQueueLength() int {
	return len(gp.taskQueue)
}

// GetActiveTaskCount 获取活跃任务数
func (gp *GoroutinePool) GetActiveTaskCount() int64 {
	return atomic.LoadInt64(&gp.activeTasks)
}

// IsHealthy 检查池是否健康
func (gp *GoroutinePool) IsHealthy() bool {
	gp.mu.RLock()
	defer gp.mu.RUnlock()

	// 检查是否有工作者在运行
	if gp.ctx.Err() != nil {
		return false
	}

	// 检查队列是否过满
	if len(gp.taskQueue) >= gp.queueSize*9/10 {
		return false
	}

	// 检查最近是否有活动
	if time.Since(gp.lastActivity) > 5*time.Minute {
		return false
	}

	return true
}

// AutomationTask 自动化任务实现
type AutomationTask struct {
	id         string
	name       string
	priority   int
	timeout    time.Duration
	executor   func(ctx context.Context) error
	params     map[string]interface{}
	retryCount int
	maxRetries int
}

// NewAutomationTask 创建自动化任务
func NewAutomationTask(id, name string, priority int, timeout time.Duration, executor func(ctx context.Context) error) *AutomationTask {
	return &AutomationTask{
		id:         id,
		name:       name,
		priority:   priority,
		timeout:    timeout,
		executor:   executor,
		params:     make(map[string]interface{}),
		maxRetries: 3,
	}
}

// Execute 执行任务
func (at *AutomationTask) Execute(ctx context.Context) error {
	log.Printf("执行自动化任务: %s (%s)", at.id, at.name)

	if at.executor == nil {
		return fmt.Errorf("task %s has no executor", at.id)
	}

	return at.executor(ctx)
}

// GetID 获取任务ID
func (at *AutomationTask) GetID() string {
	return at.id
}

// GetPriority 获取优先级
func (at *AutomationTask) GetPriority() int {
	return at.priority
}

// GetTimeout 获取超时时间
func (at *AutomationTask) GetTimeout() time.Duration {
	return at.timeout
}

// SetParam 设置参数
func (at *AutomationTask) SetParam(key string, value interface{}) {
	at.params[key] = value
}

// GetParam 获取参数
func (at *AutomationTask) GetParam(key string) interface{} {
	return at.params[key]
}
// NewPool 创建池的别名函数，为了兼容性
func NewPool(name string, config *PoolConfig) *GoroutinePool {
	return NewGoroutinePool(config)
}

// Pool 类型别名，为了兼容性
type Pool = GoroutinePool