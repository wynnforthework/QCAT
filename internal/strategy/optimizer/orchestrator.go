package optimizer

import (
	"context"
	"fmt"
	"time"
)

// OptimizerTask represents an optimization task
type OptimizerTask struct {
	ID         string
	StrategyID string
	Trigger    string
	Status     TaskStatus
	Params     map[string]interface{}
	BestParams map[string]float64
	Confidence float64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// TaskStatus represents the status of an optimization task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

// Orchestrator manages optimization tasks
type Orchestrator struct {
	checker    *TriggerChecker
	optimizer  *WalkForwardOptimizer
	overfitDet *OverfitDetector
	tasks      map[string]*OptimizerTask
}

// NewOrchestrator creates a new orchestrator
func NewOrchestrator(checker *TriggerChecker, optimizer *WalkForwardOptimizer, detector *OverfitDetector) *Orchestrator {
	return &Orchestrator{
		checker:    checker,
		optimizer:  optimizer,
		overfitDet: detector,
		tasks:      make(map[string]*OptimizerTask),
	}
}

// CreateTask creates a new optimization task
func (o *Orchestrator) CreateTask(ctx context.Context, strategyID string, trigger string) (*OptimizerTask, error) {
	task := &OptimizerTask{
		ID:         generateTaskID(),
		StrategyID: strategyID,
		Trigger:    trigger,
		Status:     TaskStatusPending,
		Params:     make(map[string]interface{}),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	o.tasks[task.ID] = task
	return task, nil
}

// RunTask executes an optimization task
func (o *Orchestrator) RunTask(ctx context.Context, taskID string) error {
	task, exists := o.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// 更新任务状态
	task.Status = TaskStatusRunning
	task.UpdatedAt = time.Now()

	// 执行WFO优化
	// 创建模拟数据用于优化
	data := &DataSet{
		Returns: []float64{0.01, -0.005, 0.02, -0.01, 0.015},
		Prices:  []float64{100, 99.5, 101.5, 100.5, 102},
	}
	
	paramSpace := map[string][2]float64{
		"param1": {0.1, 0.5},
		"param2": {10, 50},
	}
	
	result, err := o.optimizer.Optimize(ctx, data, paramSpace)
	if err != nil {
		task.Status = TaskStatusFailed
		task.UpdatedAt = time.Now()
		return fmt.Errorf("optimization failed: %w", err)
	}

	// 过拟合检测
	overfitResult, err := o.overfitDet.CheckOverfitting(ctx, result.InSampleStats, result.OutSampleStats)
	if err != nil {
		task.Status = TaskStatusFailed
		task.UpdatedAt = time.Now()
		return fmt.Errorf("overfitting check failed: %w", err)
	}

	// 更新最佳参数
	task.BestParams = result.Parameters
	task.Confidence = calculateConfidence(overfitResult)
	task.Status = TaskStatusCompleted
	task.UpdatedAt = time.Now()

	return nil
}

// GetTask retrieves a task by ID
func (o *Orchestrator) GetTask(taskID string) (*OptimizerTask, error) {
	task, exists := o.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	return task, nil
}

// ListTasks lists all tasks
func (o *Orchestrator) ListTasks() []*OptimizerTask {
	tasks := make([]*OptimizerTask, 0, len(o.tasks))
	for _, task := range o.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// Helper functions

func generateTaskID() string {
	return fmt.Sprintf("opt_%d", time.Now().UnixNano())
}

func calculateConfidence(result *OverfitResult) float64 {
	// 基于多个指标计算置信度
	confidence := 1.0

	// Deflated Sharpe影响
	if result.DeflatedSharpe < 0.5 {
		confidence *= 0.8
	}

	// PBO得分影响
	if result.PBOScore > 0.2 {
		confidence *= 0.9
	}

	// 参数敏感度影响
	for _, sensitivity := range result.ParamSensitivity {
		if sensitivity > 0.3 {
			confidence *= 0.95
		}
	}

	return confidence
}
