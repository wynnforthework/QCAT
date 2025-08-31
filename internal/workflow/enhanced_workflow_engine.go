package workflow

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// EventType 事件类型
type EventType string

const (
	EventFunctionStarted   EventType = "function_started"
	EventFunctionCompleted EventType = "function_completed"
	EventFunctionFailed    EventType = "function_failed"
	EventConflictDetected  EventType = "conflict_detected"
	EventDependencyMet     EventType = "dependency_met"
	EventWorkflowStarted   EventType = "workflow_started"
	EventWorkflowCompleted EventType = "workflow_completed"
)

// WorkflowEvent 工作流事件
type WorkflowEvent struct {
	ID         string                 `json:"id"`
	Type       EventType              `json:"type"`
	FunctionID int                    `json:"function_id,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Data       map[string]interface{} `json:"data,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

// EventHandler 事件处理器
type EventHandler func(event *WorkflowEvent) error

// InterLockRule 互锁规则
type InterLockRule struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	FunctionIDs   []int  `json:"function_ids"`   // 互锁的功能ID列表
	MaxConcurrent int    `json:"max_concurrent"` // 最大并发数
	Priority      int    `json:"priority"`       // 规则优先级
}

// EnhancedWorkflowEngine 增强的工作流引擎
type EnhancedWorkflowEngine struct {
	*WorkflowEngine

	// 事件驱动组件
	eventBus      chan *WorkflowEvent
	eventHandlers map[EventType][]EventHandler
	eventMu       sync.RWMutex

	// 互锁机制
	interlockRules  map[string]*InterLockRule
	activeFunctions map[int]bool
	functionMutex   sync.RWMutex

	// 负载均衡
	resourcePools map[string]chan struct{} // 资源池

	// 监控统计
	stats   *WorkflowStats
	statsMu sync.RWMutex
}

// WorkflowStats 工作流统计信息
type WorkflowStats struct {
	TotalExecutions      int64         `json:"total_executions"`
	SuccessfulExecutions int64         `json:"successful_executions"`
	FailedExecutions     int64         `json:"failed_executions"`
	AverageExecutionTime time.Duration `json:"average_execution_time"`
	ConflictCount        int64         `json:"conflict_count"`
	InterlockBlocks      int64         `json:"interlock_blocks"`
	LastExecutionTime    time.Time     `json:"last_execution_time"`
}

// NewEnhancedWorkflowEngine 创建增强的工作流引擎
func NewEnhancedWorkflowEngine(maxConcurrency int) *EnhancedWorkflowEngine {
	baseEngine := NewWorkflowEngine(maxConcurrency)

	engine := &EnhancedWorkflowEngine{
		WorkflowEngine:  baseEngine,
		eventBus:        make(chan *WorkflowEvent, 1000),
		eventHandlers:   make(map[EventType][]EventHandler),
		interlockRules:  make(map[string]*InterLockRule),
		activeFunctions: make(map[int]bool),
		resourcePools:   make(map[string]chan struct{}),
		stats:           &WorkflowStats{},
	}

	// 初始化资源池
	engine.initializeResourcePools()

	// 初始化互锁规则
	engine.initializeInterlockRules()

	// 注册默认执行器
	engine.registerDefaultExecutors()

	// 启动事件处理器
	go engine.processEvents()

	return engine
}

// registerDefaultExecutors 注册默认执行器
func (ewe *EnhancedWorkflowEngine) registerDefaultExecutors() {
	executors := CreateDefaultExecutors()
	for id, executor := range executors {
		if err := ewe.RegisterExecutor(id, executor); err != nil {
			log.Printf("注册执行器失败: 功能 %d - %v", id, err)
		}
	}
	log.Printf("已注册 %d 个默认执行器", len(executors))
}

// initializeResourcePools 初始化资源池
func (ewe *EnhancedWorkflowEngine) initializeResourcePools() {
	// CPU密集型任务池
	ewe.resourcePools["cpu_intensive"] = make(chan struct{}, 2)

	// IO密集型任务池
	ewe.resourcePools["io_intensive"] = make(chan struct{}, 4)

	// 网络IO任务池
	ewe.resourcePools["network_io"] = make(chan struct{}, 6)

	// 实时任务池
	ewe.resourcePools["realtime"] = make(chan struct{}, 8)

	// 监控任务池
	ewe.resourcePools["monitoring"] = make(chan struct{}, 10)
}

// initializeInterlockRules 初始化互锁规则
func (ewe *EnhancedWorkflowEngine) initializeInterlockRules() {
	rules := []*InterLockRule{
		{
			ID:            "strategy_optimization_mutex",
			Name:          "策略优化互斥",
			FunctionIDs:   []int{1, 6}, // 策略参数优化 和 周期性策略优化
			MaxConcurrent: 1,
			Priority:      10,
		},
		{
			ID:            "strategy_evolution_mutex",
			Name:          "策略进化互斥",
			FunctionIDs:   []int{24, 25}, // 策略自学习 和 遗传升级
			MaxConcurrent: 1,
			Priority:      9,
		},
		{
			ID:            "strategy_management_mutex",
			Name:          "策略管理互斥",
			FunctionIDs:   []int{7, 8}, // 策略淘汰 和 新策略引入
			MaxConcurrent: 1,
			Priority:      8,
		},
		{
			ID:            "risk_trading_mutex",
			Name:          "风险交易互斥",
			FunctionIDs:   []int{4, 12}, // 智能建仓 和 异常行情应对
			MaxConcurrent: 1,
			Priority:      10,
		},
		{
			ID:            "profit_risk_mutex",
			Name:          "收益风险互斥",
			FunctionIDs:   []int{11, 12}, // 利润最大化 和 异常行情应对
			MaxConcurrent: 1,
			Priority:      10,
		},
		{
			ID:            "cpu_intensive_limit",
			Name:          "CPU密集型任务限制",
			FunctionIDs:   []int{1, 6, 8, 19, 24, 25}, // 所有CPU密集型任务
			MaxConcurrent: 2,
			Priority:      7,
		},
	}

	for _, rule := range rules {
		ewe.interlockRules[rule.ID] = rule
	}
}

// RegisterEventHandler 注册事件处理器
func (ewe *EnhancedWorkflowEngine) RegisterEventHandler(eventType EventType, handler EventHandler) {
	ewe.eventMu.Lock()
	defer ewe.eventMu.Unlock()

	ewe.eventHandlers[eventType] = append(ewe.eventHandlers[eventType], handler)
}

// EmitEvent 发送事件
func (ewe *EnhancedWorkflowEngine) EmitEvent(event *WorkflowEvent) {
	event.ID = fmt.Sprintf("event_%d", time.Now().UnixNano())
	event.Timestamp = time.Now()

	select {
	case ewe.eventBus <- event:
		// 事件发送成功
	default:
		log.Printf("警告: 事件总线已满，丢弃事件: %s", event.Type)
	}
}

// processEvents 处理事件
func (ewe *EnhancedWorkflowEngine) processEvents() {
	for event := range ewe.eventBus {
		ewe.eventMu.RLock()
		handlers := ewe.eventHandlers[event.Type]
		ewe.eventMu.RUnlock()

		for _, handler := range handlers {
			go func(h EventHandler, e *WorkflowEvent) {
				if err := h(e); err != nil {
					log.Printf("事件处理器错误: %v", err)
				}
			}(handler, event)
		}
	}
}

// CheckInterlock 检查互锁规则
func (ewe *EnhancedWorkflowEngine) CheckInterlock(functionID int) error {
	ewe.functionMutex.Lock()
	defer ewe.functionMutex.Unlock()

	for _, rule := range ewe.interlockRules {
		// 检查功能是否在规则中
		inRule := false
		for _, id := range rule.FunctionIDs {
			if id == functionID {
				inRule = true
				break
			}
		}

		if !inRule {
			continue
		}

		// 计算当前活跃的相关功能数量
		activeCount := 0
		for _, id := range rule.FunctionIDs {
			if ewe.activeFunctions[id] {
				activeCount++
			}
		}

		// 检查是否超过最大并发数
		if activeCount >= rule.MaxConcurrent {
			ewe.statsMu.Lock()
			ewe.stats.InterlockBlocks++
			ewe.statsMu.Unlock()

			return fmt.Errorf("互锁规则 %s 阻止功能 %d 执行，当前活跃: %d, 最大并发: %d",
				rule.Name, functionID, activeCount, rule.MaxConcurrent)
		}
	}

	return nil
}

// AcquireResource 获取资源
func (ewe *EnhancedWorkflowEngine) AcquireResource(resourceType string) error {
	pool, exists := ewe.resourcePools[resourceType]
	if !exists {
		return fmt.Errorf("资源池 %s 不存在", resourceType)
	}

	select {
	case pool <- struct{}{}:
		return nil
	default:
		return fmt.Errorf("资源池 %s 已满", resourceType)
	}
}

// ReleaseResource 释放资源
func (ewe *EnhancedWorkflowEngine) ReleaseResource(resourceType string) {
	pool, exists := ewe.resourcePools[resourceType]
	if !exists {
		log.Printf("警告: 资源池 %s 不存在", resourceType)
		return
	}

	select {
	case <-pool:
		// 资源释放成功
	default:
		log.Printf("警告: 资源池 %s 为空，无法释放", resourceType)
	}
}

// ExecuteWithInterlock 带互锁的执行功能
func (ewe *EnhancedWorkflowEngine) ExecuteWithInterlock(ctx context.Context, functionID int) error {
	// 检查互锁规则
	if err := ewe.CheckInterlock(functionID); err != nil {
		ewe.EmitEvent(&WorkflowEvent{
			Type:       EventConflictDetected,
			FunctionID: functionID,
			Error:      err.Error(),
		})
		return err
	}

	// 标记功能为活跃状态
	ewe.functionMutex.Lock()
	ewe.activeFunctions[functionID] = true
	ewe.functionMutex.Unlock()

	// 发送开始事件
	ewe.EmitEvent(&WorkflowEvent{
		Type:       EventFunctionStarted,
		FunctionID: functionID,
	})

	// 执行功能
	startTime := time.Now()
	err := ewe.WorkflowEngine.executeFunction(ctx, functionID)
	duration := time.Since(startTime)

	// 更新统计信息
	ewe.statsMu.Lock()
	ewe.stats.TotalExecutions++
	if err != nil {
		ewe.stats.FailedExecutions++
	} else {
		ewe.stats.SuccessfulExecutions++
	}
	ewe.stats.LastExecutionTime = time.Now()
	// 更新平均执行时间
	if ewe.stats.TotalExecutions > 0 {
		ewe.stats.AverageExecutionTime = time.Duration(
			(int64(ewe.stats.AverageExecutionTime)*(ewe.stats.TotalExecutions-1) + int64(duration)) /
				ewe.stats.TotalExecutions)
	}
	ewe.statsMu.Unlock()

	// 清除活跃状态
	ewe.functionMutex.Lock()
	delete(ewe.activeFunctions, functionID)
	ewe.functionMutex.Unlock()

	// 发送完成或失败事件
	if err != nil {
		ewe.EmitEvent(&WorkflowEvent{
			Type:       EventFunctionFailed,
			FunctionID: functionID,
			Error:      err.Error(),
		})
	} else {
		ewe.EmitEvent(&WorkflowEvent{
			Type:       EventFunctionCompleted,
			FunctionID: functionID,
		})
	}

	return err
}

// GetStats 获取统计信息
func (ewe *EnhancedWorkflowEngine) GetStats() *WorkflowStats {
	ewe.statsMu.RLock()
	defer ewe.statsMu.RUnlock()

	// 返回副本
	return &WorkflowStats{
		TotalExecutions:      ewe.stats.TotalExecutions,
		SuccessfulExecutions: ewe.stats.SuccessfulExecutions,
		FailedExecutions:     ewe.stats.FailedExecutions,
		AverageExecutionTime: ewe.stats.AverageExecutionTime,
		ConflictCount:        ewe.stats.ConflictCount,
		InterlockBlocks:      ewe.stats.InterlockBlocks,
		LastExecutionTime:    ewe.stats.LastExecutionTime,
	}
}

// ExecuteWorkflowWithEnhancements 执行增强工作流
func (ewe *EnhancedWorkflowEngine) ExecuteWorkflowWithEnhancements(ctx context.Context) error {
	// 发送工作流开始事件
	ewe.EmitEvent(&WorkflowEvent{
		Type: EventWorkflowStarted,
		Data: map[string]interface{}{
			"start_time": time.Now(),
		},
	})

	// 获取执行顺序
	executionOrder, err := ewe.dependencyGraph.GetExecutionOrder()
	if err != nil {
		return fmt.Errorf("failed to get execution order: %w", err)
	}

	// 只执行已启用的功能
	enabledFunctions := ewe.dependencyGraph.GetEnabledFunctions()
	enabledSet := make(map[int]bool)
	for _, id := range enabledFunctions {
		enabledSet[id] = true
	}

	// 过滤执行顺序
	var filteredOrder []int
	for _, id := range executionOrder {
		if enabledSet[id] {
			filteredOrder = append(filteredOrder, id)
		}
	}

	log.Printf("🚀 开始执行增强工作流，功能顺序: %v", filteredOrder)

	// 清空之前的结果
	ewe.results = make(map[int]*ExecutionResult)

	// 使用增强执行方法
	err = ewe.executeWithEnhancements(ctx, filteredOrder)

	// 发送工作流完成事件
	eventType := EventWorkflowCompleted
	eventData := map[string]interface{}{
		"end_time": time.Now(),
		"success":  err == nil,
	}
	if err != nil {
		eventData["error"] = err.Error()
	}

	ewe.EmitEvent(&WorkflowEvent{
		Type: eventType,
		Data: eventData,
	})

	return err
}

// executeWithEnhancements 使用增强功能执行
func (ewe *EnhancedWorkflowEngine) executeWithEnhancements(ctx context.Context, order []int) error {
	var wg sync.WaitGroup
	errorChan := make(chan error, len(order))

	// 创建执行阶段
	stages := ewe.createExecutionStages(order)

	for stageIndex, stage := range stages {
		log.Printf("执行阶段 %d: %v", stageIndex+1, stage)

		// 并发执行同一阶段的功能
		for _, functionID := range stage {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				// 获取信号量
				select {
				case ewe.semaphore <- struct{}{}:
					defer func() { <-ewe.semaphore }()
				case <-ctx.Done():
					errorChan <- ctx.Err()
					return
				}

				// 使用增强执行方法
				if err := ewe.ExecuteWithInterlock(ctx, id); err != nil {
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

	log.Println("✅ 增强工作流执行完成")
	return nil
}

// GetActiveFunctions 获取当前活跃的功能
func (ewe *EnhancedWorkflowEngine) GetActiveFunctions() []int {
	ewe.functionMutex.RLock()
	defer ewe.functionMutex.RUnlock()

	var active []int
	for id := range ewe.activeFunctions {
		active = append(active, id)
	}

	return active
}

// GetInterlockStatus 获取互锁状态
func (ewe *EnhancedWorkflowEngine) GetInterlockStatus() map[string]interface{} {
	ewe.functionMutex.RLock()
	defer ewe.functionMutex.RUnlock()

	status := make(map[string]interface{})

	for ruleID, rule := range ewe.interlockRules {
		activeCount := 0
		activeFunctions := []int{}

		for _, id := range rule.FunctionIDs {
			if ewe.activeFunctions[id] {
				activeCount++
				activeFunctions = append(activeFunctions, id)
			}
		}

		status[ruleID] = map[string]interface{}{
			"name":             rule.Name,
			"max_concurrent":   rule.MaxConcurrent,
			"active_count":     activeCount,
			"active_functions": activeFunctions,
			"blocked":          activeCount >= rule.MaxConcurrent,
		}
	}

	return status
}

// Stop 停止增强工作流引擎
func (ewe *EnhancedWorkflowEngine) Stop() {
	ewe.WorkflowEngine.Stop()
	close(ewe.eventBus)
	log.Println("增强工作流引擎已停止")
}
