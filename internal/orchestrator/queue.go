package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TaskQueue manages task execution queue
type TaskQueue struct {
	tasks      []*QueuedTask
	workers    int
	maxRetries int
	retryDelay time.Duration
	mu         sync.RWMutex
	workerPool chan struct{}
}

// QueuedTask represents a task in the queue
type QueuedTask struct {
	ID          string
	Type        TaskType
	Priority    int
	RetryCount  int
	LastError   string
	Status      TaskStatus
	Handler     TaskHandler
	CreatedAt   time.Time
	StartedAt   time.Time
	CompletedAt time.Time
}

// NewTaskQueue creates a new task queue
func NewTaskQueue(workers int, maxRetries int, retryDelay time.Duration) *TaskQueue {
	return &TaskQueue{
		tasks:      make([]*QueuedTask, 0),
		workers:    workers,
		maxRetries: maxRetries,
		retryDelay: retryDelay,
		workerPool: make(chan struct{}, workers),
	}
}

// AddTask adds a task to the queue
func (q *TaskQueue) AddTask(taskType TaskType, priority int, handler TaskHandler) (*QueuedTask, error) {
	task := &QueuedTask{
		ID:        fmt.Sprintf("task_%d", time.Now().UnixNano()),
		Type:      taskType,
		Priority:  priority,
		Status:    TaskStatusPending,
		Handler:   handler,
		CreatedAt: time.Now(),
	}

	q.mu.Lock()
	q.tasks = append(q.tasks, task)
	q.mu.Unlock()

	// 尝试立即执行任务
	go q.processTask(task)

	return task, nil
}

// processTask processes a single task
func (q *TaskQueue) processTask(task *QueuedTask) {
	// 获取worker
	q.workerPool <- struct{}{}
	defer func() {
		<-q.workerPool
	}()

	ctx := context.Background()

	for task.RetryCount <= q.maxRetries {
		task.Status = TaskStatusRunning
		task.StartedAt = time.Now()

		err := task.Handler.Handle(ctx)
		if err == nil {
			// 任务成功完成
			task.Status = TaskStatusCompleted
			task.CompletedAt = time.Now()
			return
		}

		// 任务失败，准备重试
		task.LastError = err.Error()
		task.RetryCount++

		if task.RetryCount > q.maxRetries {
			task.Status = TaskStatusFailed
			return
		}

		// 等待重试延迟
		time.Sleep(q.retryDelay * time.Duration(task.RetryCount))
	}
}

// GetTask gets a task by ID
func (q *TaskQueue) GetTask(taskID string) (*QueuedTask, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for _, task := range q.tasks {
		if task.ID == taskID {
			return task, nil
		}
	}

	return nil, fmt.Errorf("task not found: %s", taskID)
}

// ListTasks lists all tasks
func (q *TaskQueue) ListTasks() []*QueuedTask {
	q.mu.RLock()
	defer q.mu.RUnlock()

	tasks := make([]*QueuedTask, len(q.tasks))
	copy(tasks, q.tasks)
	return tasks
}

// CleanupCompletedTasks removes completed tasks older than the specified duration
func (q *TaskQueue) CleanupCompletedTasks(age time.Duration) {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now()
	var activeTasks []*QueuedTask

	for _, task := range q.tasks {
		if task.Status == TaskStatusCompleted && now.Sub(task.CompletedAt) > age {
			continue
		}
		activeTasks = append(activeTasks, task)
	}

	q.tasks = activeTasks
}

// GetQueueStats gets queue statistics
type QueueStats struct {
	TotalTasks     int
	PendingTasks   int
	RunningTasks   int
	CompletedTasks int
	FailedTasks    int
	AvgWaitTime    time.Duration
	AvgProcessTime time.Duration
}

func (q *TaskQueue) GetStats() *QueueStats {
	q.mu.RLock()
	defer q.mu.RUnlock()

	stats := &QueueStats{}
	var totalWaitTime time.Duration
	var totalProcessTime time.Duration
	var processedTasks int

	for _, task := range q.tasks {
		stats.TotalTasks++

		switch task.Status {
		case TaskStatusPending:
			stats.PendingTasks++
		case TaskStatusRunning:
			stats.RunningTasks++
		case TaskStatusCompleted:
			stats.CompletedTasks++
			if !task.StartedAt.IsZero() {
				totalWaitTime += task.StartedAt.Sub(task.CreatedAt)
				totalProcessTime += task.CompletedAt.Sub(task.StartedAt)
				processedTasks++
			}
		case TaskStatusFailed:
			stats.FailedTasks++
		}
	}

	if processedTasks > 0 {
		stats.AvgWaitTime = totalWaitTime / time.Duration(processedTasks)
		stats.AvgProcessTime = totalProcessTime / time.Duration(processedTasks)
	}

	return stats
}
