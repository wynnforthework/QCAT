package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// TaskType represents the type of scheduled task
type TaskType string

const (
	TaskTypeMarketHealth  TaskType = "market_health"
	TaskTypeStrategyScore TaskType = "strategy_score"
	TaskTypeHotlistScan   TaskType = "hotlist_scan"
	TaskTypeDailyOptimize TaskType = "daily_optimize"
)

// Task represents a scheduled task
type Task struct {
	ID          string
	Type        TaskType
	Schedule    string
	LastRunTime time.Time
	NextRunTime time.Time
	Status      TaskStatus
	Error       string
}

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

// Scheduler manages task scheduling
type Scheduler struct {
	cron     *cron.Cron
	tasks    map[string]*Task
	handlers map[TaskType]TaskHandler
	mu       sync.RWMutex
}

// TaskHandler defines the interface for task handlers
type TaskHandler interface {
	Handle(ctx context.Context) error
}

// NewScheduler creates a new scheduler
func NewScheduler() *Scheduler {
	c := cron.New(cron.WithSeconds())
	return &Scheduler{
		cron:     c,
		tasks:    make(map[string]*Task),
		handlers: make(map[TaskType]TaskHandler),
	}
}

// RegisterHandler registers a handler for a task type
func (s *Scheduler) RegisterHandler(taskType TaskType, handler TaskHandler) {
	s.handlers[taskType] = handler
}

// AddTask adds a new task to the scheduler
func (s *Scheduler) AddTask(taskType TaskType, schedule string) error {
	handler, exists := s.handlers[taskType]
	if !exists {
		return fmt.Errorf("no handler registered for task type: %s", taskType)
	}

	task := &Task{
		ID:       fmt.Sprintf("%s_%d", taskType, time.Now().UnixNano()),
		Type:     taskType,
		Schedule: schedule,
		Status:   TaskStatusPending,
	}

	// 添加到cron
	_, err := s.cron.AddFunc(schedule, func() {
		ctx := context.Background()
		s.runTask(ctx, task, handler)
	})
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	s.mu.Lock()
	s.tasks[task.ID] = task
	s.mu.Unlock()

	return nil
}

// Start starts the scheduler
func (s *Scheduler) Start() {
	s.cron.Start()
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.cron.Stop()
}

// runTask executes a task
func (s *Scheduler) runTask(ctx context.Context, task *Task, handler TaskHandler) {
	s.mu.Lock()
	task.Status = TaskStatusRunning
	task.LastRunTime = time.Now()
	s.mu.Unlock()

	err := handler.Handle(ctx)

	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil {
		task.Status = TaskStatusFailed
		task.Error = err.Error()
	} else {
		task.Status = TaskStatusCompleted
		task.Error = ""
	}
}

// GetTask gets a task by ID
func (s *Scheduler) GetTask(taskID string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	return task, nil
}

// ListTasks lists all tasks
func (s *Scheduler) ListTasks() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}
