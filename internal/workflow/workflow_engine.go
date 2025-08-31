package workflow

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// ExecutionStatus 执行状态
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusCompleted ExecutionStatus = "completed"
	StatusFailed    ExecutionStatus = "failed"
	StatusSkipped   ExecutionStatus = "skipped"
)

// ExecutionResult 执行结果
type ExecutionResult struct {
	FunctionID    int                    `json:"function_id"`
	Status        ExecutionStatus        `json:"status"`
	StartTime     time.Time              `json:"start_time"`
	EndTime       time.Time              `json:"end_time"`
	Duration      time.Duration          `json:"duration"`
	Error         string                 `json:"error,omitempty"`
	Output        interface{}            `json:"output,omitempty"`
	ResourceUsage map[string]interface{} `json:"resource_usage,omitempty"`
}

// WorkflowEngine 工作流引擎
type WorkflowEngine struct {
	dependencyGraph *DependencyGraph
	executors       map[int]AutomationExecutor
	results         map[int]*ExecutionResult
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	maxConcurrency  int
	semaphore       chan struct{}
}

// AutomationExecutor 自动化执行器接口
type AutomationExecutor interface {
	Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
	GetName() string
	GetResourceRequirements() map[string]interface{}
}

// NewWorkflowEngine 创建工作流引擎
func NewWorkflowEngine(maxConcurrency int) *WorkflowEngine {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkflowEngine{
		dependencyGraph: NewDependencyGraph(),
		executors:       make(map[int]AutomationExecutor),
		results:         make(map[int]*ExecutionResult),
		ctx:             ctx,
		cancel:          cancel,
		maxConcurrency:  maxConcurrency,
		semaphore:       make(chan struct{}, maxConcurrency),
	}
}

// RegisterExecutor 注册执行器
func (we *WorkflowEngine) RegisterExecutor(functionID int, executor AutomationExecutor) error {
	we.mu.Lock()
	defer we.mu.Unlock()

	if _, exists := we.executors[functionID]; exists {
		return fmt.Errorf("executor for function %d already registered", functionID)
	}

	we.executors[functionID] = executor
	log.Printf("注册执行器: 功能 %d - %s", functionID, executor.GetName())
	return nil
}

// ExecuteWorkflow 执行工作流
func (we *WorkflowEngine) ExecuteWorkflow(ctx context.Context) error {
	we.mu.Lock()
	defer we.mu.Unlock()

	log.Println("🚀 开始执行26项自动化功能工作流")

	// 获取执行顺序
	executionOrder, err := we.dependencyGraph.GetExecutionOrder()
	if err != nil {
		return fmt.Errorf("failed to get execution order: %w", err)
	}

	log.Printf("执行顺序: %v", executionOrder)

	// 只执行已启用的功能
	enabledFunctions := we.dependencyGraph.GetEnabledFunctions()
	enabledSet := make(map[int]bool)
	for _, id := range enabledFunctions {
		enabledSet[id] = true
	}

	// 过滤执行顺序，只保留已启用的功能
	var filteredOrder []int
	for _, id := range executionOrder {
		if enabledSet[id] {
			filteredOrder = append(filteredOrder, id)
		}
	}

	log.Printf("已启用功能执行顺序: %v", filteredOrder)

	// 清空之前的结果
	we.results = make(map[int]*ExecutionResult)

	// 执行功能
	return we.executeInOrder(ctx, filteredOrder)
}

// executeInOrder 按顺序执行功能
func (we *WorkflowEngine) executeInOrder(ctx context.Context, order []int) error {
	var wg sync.WaitGroup
	errorChan := make(chan error, len(order))

	// 创建执行阶段
	stages := we.createExecutionStages(order)

	for stageIndex, stage := range stages {
		log.Printf("执行阶段 %d: %v", stageIndex+1, stage)

		// 并发执行同一阶段的功能
		for _, functionID := range stage {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				// 获取信号量
				select {
				case we.semaphore <- struct{}{}:
					defer func() { <-we.semaphore }()
				case <-ctx.Done():
					errorChan <- ctx.Err()
					return
				}

				if err := we.executeFunction(ctx, id); err != nil {
					errorChan <- fmt.Errorf("function %d failed: %w", id, err)
				}
			}(functionID)
		}

		// 等待当前阶段完成
		wg.Wait()

		// 检查是否有错误
		select {
		case err := <-errorChan:
			return err
		default:
		}

		log.Printf("阶段 %d 执行完成", stageIndex+1)
	}

	log.Println("✅ 工作流执行完成")
	return nil
}

// createExecutionStages 创建执行阶段
func (we *WorkflowEngine) createExecutionStages(order []int) [][]int {
	var stages [][]int
	processed := make(map[int]bool)

	for len(processed) < len(order) {
		var currentStage []int

		for _, id := range order {
			if processed[id] {
				continue
			}

			// 检查依赖是否都已完成
			fn, _ := we.dependencyGraph.GetFunctionInfo(id)
			canExecute := true

			for _, depID := range fn.Dependencies {
				if !processed[depID] {
					canExecute = false
					break
				}
			}

			// 检查冲突
			hasConflict := false
			for _, conflictID := range fn.Conflicts {
				for _, stageID := range currentStage {
					if stageID == conflictID {
						hasConflict = true
						break
					}
				}
				if hasConflict {
					break
				}
			}

			if canExecute && !hasConflict {
				currentStage = append(currentStage, id)
				processed[id] = true
			}
		}

		if len(currentStage) > 0 {
			stages = append(stages, currentStage)
		} else {
			// 如果没有可执行的功能，可能存在循环依赖或其他问题
			break
		}
	}

	return stages
}

// executeFunction 执行单个功能
func (we *WorkflowEngine) executeFunction(ctx context.Context, functionID int) error {
	fn, err := we.dependencyGraph.GetFunctionInfo(functionID)
	if err != nil {
		return err
	}

	executor, exists := we.executors[functionID]
	if !exists {
		// 如果没有注册执行器，跳过执行
		log.Printf("⚠️ 功能 %d (%s) 没有注册执行器，跳过执行", functionID, fn.Name)
		we.mu.Lock()
		we.results[functionID] = &ExecutionResult{
			FunctionID: functionID,
			Status:     StatusSkipped,
			StartTime:  time.Now(),
			EndTime:    time.Now(),
			Duration:   0,
		}
		we.mu.Unlock()
		return nil
	}

	log.Printf("🔄 执行功能 %d: %s", functionID, fn.Name)

	result := &ExecutionResult{
		FunctionID: functionID,
		Status:     StatusRunning,
		StartTime:  time.Now(),
	}
	we.mu.Lock()
	we.results[functionID] = result
	we.mu.Unlock()

	// 创建带超时的上下文
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(fn.ExecutionTime)*time.Second*2)
	defer cancel()

	// 执行功能
	output, err := executor.Execute(execCtx, map[string]interface{}{
		"function_id":   functionID,
		"function_name": fn.Name,
		"priority":      fn.Priority,
	})

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	if err != nil {
		result.Status = StatusFailed
		result.Error = err.Error()
		log.Printf("❌ 功能 %d (%s) 执行失败: %v", functionID, fn.Name, err)
		return err
	}

	result.Status = StatusCompleted
	result.Output = output
	log.Printf("✅ 功能 %d (%s) 执行成功，耗时: %v", functionID, fn.Name, result.Duration)

	return nil
}

// GetExecutionResults 获取执行结果
func (we *WorkflowEngine) GetExecutionResults() map[int]*ExecutionResult {
	we.mu.RLock()
	defer we.mu.RUnlock()

	results := make(map[int]*ExecutionResult)
	for id, result := range we.results {
		results[id] = result
	}

	return results
}

// GetExecutionSummary 获取执行摘要
func (we *WorkflowEngine) GetExecutionSummary() map[string]interface{} {
	we.mu.RLock()
	defer we.mu.RUnlock()

	summary := map[string]interface{}{
		"total_functions": len(we.results),
		"completed":       0,
		"failed":          0,
		"skipped":         0,
		"total_duration":  time.Duration(0),
	}

	for _, result := range we.results {
		switch result.Status {
		case StatusCompleted:
			summary["completed"] = summary["completed"].(int) + 1
		case StatusFailed:
			summary["failed"] = summary["failed"].(int) + 1
		case StatusSkipped:
			summary["skipped"] = summary["skipped"].(int) + 1
		}

		summary["total_duration"] = summary["total_duration"].(time.Duration) + result.Duration
	}

	return summary
}

// Stop 停止工作流引擎
func (we *WorkflowEngine) Stop() {
	we.cancel()
	log.Println("工作流引擎已停止")
}

// GetDependencyGraph 获取依赖图
func (we *WorkflowEngine) GetDependencyGraph() *DependencyGraph {
	return we.dependencyGraph
}
