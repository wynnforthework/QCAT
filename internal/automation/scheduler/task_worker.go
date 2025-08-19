package scheduler

import (
	"context"
	"log"
	"sync"
	"time"
)

// TaskWorker 任务工作器
type TaskWorker struct {
	id              int
	taskQueue       <-chan *ScheduledTask
	completionHandler func(*ScheduledTask, error)
	isRunning       bool
	mu              sync.RWMutex
}

// NewTaskWorker 创建任务工作器
func NewTaskWorker(id int, taskQueue <-chan *ScheduledTask, completionHandler func(*ScheduledTask, error)) *TaskWorker {
	return &TaskWorker{
		id:              id,
		taskQueue:       taskQueue,
		completionHandler: completionHandler,
	}
}

// Start 启动工作器
func (tw *TaskWorker) Start(wg *sync.WaitGroup) {
	defer wg.Done()

	tw.mu.Lock()
	tw.isRunning = true
	tw.mu.Unlock()

	log.Printf("Task worker %d started", tw.id)

	for task := range tw.taskQueue {
		tw.executeTask(task)
	}

	tw.mu.Lock()
	tw.isRunning = false
	tw.mu.Unlock()

	log.Printf("Task worker %d stopped", tw.id)
}

// executeTask 执行任务
func (tw *TaskWorker) executeTask(task *ScheduledTask) {
	log.Printf("Worker %d executing task: %s", tw.id, task.Name)

	startTime := time.Now()
	
	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), task.Timeout)
	defer cancel()

	// 执行任务
	var err error
	if task.Handler != nil {
		err = task.Handler(ctx, task)
	} else {
		log.Printf("Task %s has no handler", task.Name)
	}

	duration := time.Since(startTime)
	
	if err != nil {
		log.Printf("Worker %d task failed: %s, duration: %v, error: %v", 
			tw.id, task.Name, duration, err)
	} else {
		log.Printf("Worker %d task completed: %s, duration: %v", 
			tw.id, task.Name, duration)
	}

	// 调用完成处理器
	if tw.completionHandler != nil {
		tw.completionHandler(task, err)
	}
}

// IsRunning 检查工作器是否运行中
func (tw *TaskWorker) IsRunning() bool {
	tw.mu.RLock()
	defer tw.mu.RUnlock()
	return tw.isRunning
}
