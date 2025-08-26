package workflow

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/your-org/qcat/internal/events"
)

// EventDrivenWorkflowEngine 事件驱动的工作流引擎
type EventDrivenWorkflowEngine struct {
	*EnhancedWorkflowEngine

	// 事件总线
	eventBus *events.EventBus

	// 事件处理器
	handlers []events.EventHandler

	// 订阅ID
	subscriptions []string

	// 运行状态
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewEventDrivenWorkflowEngine 创建事件驱动的工作流引擎
func NewEventDrivenWorkflowEngine(maxConcurrency int) *EventDrivenWorkflowEngine {
	baseEngine := NewEnhancedWorkflowEngine(maxConcurrency)

	// 创建事件总线
	eventBusConfig := &events.EventBusConfig{
		BufferSize:       10000,
		WorkerCount:      5,
		MaxRetries:       3,
		RetryDelay:       time.Second,
		EnableStorage:    true,
		StorageRetention: 24 * time.Hour,
	}

	eventBus := events.NewEventBus(eventBusConfig)

	ctx, cancel := context.WithCancel(context.Background())

	engine := &EventDrivenWorkflowEngine{
		EnhancedWorkflowEngine: baseEngine,
		eventBus:               eventBus,
		handlers:               make([]events.EventHandler, 0),
		subscriptions:          make([]string, 0),
		ctx:                    ctx,
		cancel:                 cancel,
	}

	// 初始化事件处理器
	engine.initializeEventHandlers()

	// 注册事件处理器
	engine.registerEventHandlers()

	return engine
}

// initializeEventHandlers 初始化事件处理器
func (edwe *EventDrivenWorkflowEngine) initializeEventHandlers() {
	// 基础日志处理器
	logHandler := NewLogEventHandler()
	edwe.handlers = append(edwe.handlers, logHandler)

	// 统计处理器
	statsHandler := NewStatsEventHandler()
	edwe.handlers = append(edwe.handlers, statsHandler)
}

// registerEventHandlers 注册事件处理器
func (edwe *EventDrivenWorkflowEngine) registerEventHandlers() {
	for _, handler := range edwe.handlers {
		subscriptionID, err := edwe.eventBus.Subscribe(
			handler.GetEventTypes(),
			handler,
			nil, // 无过滤器
		)

		if err != nil {
			log.Printf("注册事件处理器失败: %s - %v", handler.GetName(), err)
			continue
		}

		edwe.subscriptions = append(edwe.subscriptions, subscriptionID)
		log.Printf("注册事件处理器: %s", handler.GetName())
	}
}

// ExecuteEventDrivenWorkflow 执行事件驱动工作流
func (edwe *EventDrivenWorkflowEngine) ExecuteEventDrivenWorkflow(ctx context.Context) error {
	// 发送工作流开始事件
	startEvent := &events.Event{
		Type:     events.EventWorkflowStarted,
		Source:   "EventDrivenWorkflowEngine",
		Priority: events.PriorityHigh,
		Data: map[string]interface{}{
			"start_time":        time.Now(),
			"total_functions":   len(edwe.dependencyGraph.GetAllFunctions()),
			"enabled_functions": len(edwe.dependencyGraph.GetEnabledFunctions()),
		},
	}

	if err := edwe.eventBus.Publish(startEvent); err != nil {
		return fmt.Errorf("failed to publish workflow start event: %w", err)
	}

	// 获取执行顺序
	executionOrder, err := edwe.dependencyGraph.GetExecutionOrder()
	if err != nil {
		return fmt.Errorf("failed to get execution order: %w", err)
	}

	// 只执行已启用的功能
	enabledFunctions := edwe.dependencyGraph.GetEnabledFunctions()
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

	log.Printf("🚀 开始执行事件驱动工作流，功能顺序: %v", filteredOrder)

	// 清空之前的结果
	edwe.results = make(map[int]*ExecutionResult)

	// 启动事件驱动执行
	err = edwe.executeEventDriven(ctx, filteredOrder)

	// 发送工作流完成事件
	endEvent := &events.Event{
		Type:     events.EventWorkflowCompleted,
		Source:   "EventDrivenWorkflowEngine",
		Priority: events.PriorityHigh,
		Data: map[string]interface{}{
			"end_time": time.Now(),
			"success":  err == nil,
			"results":  edwe.GetExecutionSummary(),
		},
	}

	if err != nil {
		endEvent.Type = events.EventWorkflowFailed
		endEvent.Data["error"] = err.Error()
	}

	edwe.eventBus.Publish(endEvent)

	return err
}

// executeEventDriven 事件驱动执行
func (edwe *EventDrivenWorkflowEngine) executeEventDriven(ctx context.Context, order []int) error {
	// 创建执行状态跟踪
	executionState := &ExecutionState{
		pendingFunctions:   make(map[int]bool),
		completedFunctions: make(map[int]bool),
		failedFunctions:    make(map[int]bool),
		dependencyMap:      make(map[int][]int),
	}

	// 初始化执行状态
	for _, id := range order {
		executionState.pendingFunctions[id] = true

		fn, _ := edwe.dependencyGraph.GetFunctionInfo(id)
		executionState.dependencyMap[id] = fn.Dependencies
	}

	// 启动执行状态监控
	edwe.wg.Add(1)
	go edwe.monitorExecution(ctx, executionState)

	// 启动初始可执行的功能
	for _, id := range order {
		if edwe.canExecuteFunction(id, executionState) {
			go edwe.executeEventDrivenFunction(ctx, id, executionState)
		}
	}

	// 等待所有功能完成或失败
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			if edwe.isExecutionComplete(executionState) {
				log.Println("✅ 事件驱动工作流执行完成")
				return nil
			}
		}
	}
}

// executeEventDrivenFunction 执行单个功能（事件驱动）
func (edwe *EventDrivenWorkflowEngine) executeEventDrivenFunction(ctx context.Context, functionID int, state *ExecutionState) {
	fn, err := edwe.dependencyGraph.GetFunctionInfo(functionID)
	if err != nil {
		log.Printf("获取功能信息失败: %d - %v", functionID, err)
		return
	}

	// 发送功能开始事件
	startEvent := &events.Event{
		Type:     events.EventFunctionStarted,
		Source:   "EventDrivenWorkflowEngine",
		Priority: events.PriorityNormal,
		Data: map[string]interface{}{
			"function_id":   functionID,
			"function_name": fn.Name,
			"start_time":    time.Now(),
		},
	}

	edwe.eventBus.Publish(startEvent)

	// 执行功能
	startTime := time.Now()
	err = edwe.ExecuteWithInterlock(ctx, functionID)
	duration := time.Since(startTime)

	// 更新执行状态
	state.mu.Lock()
	delete(state.pendingFunctions, functionID)
	if err != nil {
		state.failedFunctions[functionID] = true
	} else {
		state.completedFunctions[functionID] = true
	}
	state.mu.Unlock()

	// 发送功能完成或失败事件
	var endEvent *events.Event
	if err != nil {
		endEvent = &events.Event{
			Type:     events.EventFunctionFailed,
			Source:   "EventDrivenWorkflowEngine",
			Priority: events.PriorityHigh,
			Data: map[string]interface{}{
				"function_id":   functionID,
				"function_name": fn.Name,
				"error":         err.Error(),
				"duration":      duration,
				"end_time":      time.Now(),
			},
		}
	} else {
		endEvent = &events.Event{
			Type:     events.EventFunctionCompleted,
			Source:   "EventDrivenWorkflowEngine",
			Priority: events.PriorityNormal,
			Data: map[string]interface{}{
				"function_id":   functionID,
				"function_name": fn.Name,
				"duration":      duration,
				"end_time":      time.Now(),
			},
		}
	}

	edwe.eventBus.Publish(endEvent)

	// 检查并启动新的可执行功能
	edwe.checkAndStartNextFunctions(ctx, state)
}

// canExecuteFunction 检查功能是否可以执行
func (edwe *EventDrivenWorkflowEngine) canExecuteFunction(functionID int, state *ExecutionState) bool {
	state.mu.RLock()
	defer state.mu.RUnlock()

	// 检查是否还在待执行列表中
	if !state.pendingFunctions[functionID] {
		return false
	}

	// 检查依赖是否都已完成
	dependencies := state.dependencyMap[functionID]
	for _, depID := range dependencies {
		if !state.completedFunctions[depID] {
			return false
		}
	}

	return true
}

// checkAndStartNextFunctions 检查并启动下一批可执行的功能
func (edwe *EventDrivenWorkflowEngine) checkAndStartNextFunctions(ctx context.Context, state *ExecutionState) {
	state.mu.RLock()
	pendingFunctions := make([]int, 0)
	for id := range state.pendingFunctions {
		pendingFunctions = append(pendingFunctions, id)
	}
	state.mu.RUnlock()

	for _, id := range pendingFunctions {
		if edwe.canExecuteFunction(id, state) {
			go edwe.executeEventDrivenFunction(ctx, id, state)
		}
	}
}

// isExecutionComplete 检查执行是否完成
func (edwe *EventDrivenWorkflowEngine) isExecutionComplete(state *ExecutionState) bool {
	state.mu.RLock()
	defer state.mu.RUnlock()

	return len(state.pendingFunctions) == 0
}

// monitorExecution 监控执行状态
func (edwe *EventDrivenWorkflowEngine) monitorExecution(ctx context.Context, state *ExecutionState) {
	defer edwe.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			state.mu.RLock()
			pendingCount := len(state.pendingFunctions)
			completedCount := len(state.completedFunctions)
			failedCount := len(state.failedFunctions)
			state.mu.RUnlock()

			log.Printf("执行状态: 待执行=%d, 已完成=%d, 失败=%d",
				pendingCount, completedCount, failedCount)

			if pendingCount == 0 {
				return
			}
		}
	}
}

// ExecutionState 执行状态
type ExecutionState struct {
	pendingFunctions   map[int]bool
	completedFunctions map[int]bool
	failedFunctions    map[int]bool
	dependencyMap      map[int][]int
	mu                 sync.RWMutex
}

// GetEventBus 获取事件总线
func (edwe *EventDrivenWorkflowEngine) GetEventBus() *events.EventBus {
	return edwe.eventBus
}

// Stop 停止事件驱动工作流引擎
func (edwe *EventDrivenWorkflowEngine) Stop() {
	// 取消订阅
	for _, subscriptionID := range edwe.subscriptions {
		edwe.eventBus.Unsubscribe(subscriptionID)
	}

	// 停止基础引擎
	edwe.EnhancedWorkflowEngine.Stop()

	// 停止事件总线
	edwe.eventBus.Stop()

	// 停止监控
	edwe.cancel()
	edwe.wg.Wait()

	log.Println("事件驱动工作流引擎已停止")
}

// LogEventHandler 日志事件处理器
type LogEventHandler struct {
	name       string
	eventTypes []events.EventType
	priority   int
}

// NewLogEventHandler 创建日志事件处理器
func NewLogEventHandler() *LogEventHandler {
	return &LogEventHandler{
		name: "LogEventHandler",
		eventTypes: []events.EventType{
			events.EventWorkflowStarted,
			events.EventWorkflowCompleted,
			events.EventFunctionStarted,
			events.EventFunctionCompleted,
			events.EventFunctionFailed,
		},
		priority: 1,
	}
}

// GetName 获取处理器名称
func (leh *LogEventHandler) GetName() string {
	return leh.name
}

// GetEventTypes 获取处理的事件类型
func (leh *LogEventHandler) GetEventTypes() []events.EventType {
	return leh.eventTypes
}

// GetPriority 获取处理器优先级
func (leh *LogEventHandler) GetPriority() int {
	return leh.priority
}

// Handle 处理事件
func (leh *LogEventHandler) Handle(ctx context.Context, event *events.Event) error {
	switch event.Type {
	case events.EventWorkflowStarted:
		log.Printf("📋 工作流开始: %v", event.Data)
	case events.EventWorkflowCompleted:
		log.Printf("✅ 工作流完成: %v", event.Data)
	case events.EventFunctionStarted:
		functionID := event.Data["function_id"]
		functionName := event.Data["function_name"]
		log.Printf("🔄 功能开始: %d - %s", functionID, functionName)
	case events.EventFunctionCompleted:
		functionID := event.Data["function_id"]
		functionName := event.Data["function_name"]
		duration := event.Data["duration"]
		log.Printf("✅ 功能完成: %d - %s (耗时: %v)", functionID, functionName, duration)
	case events.EventFunctionFailed:
		functionID := event.Data["function_id"]
		functionName := event.Data["function_name"]
		errorMsg := event.Data["error"]
		log.Printf("❌ 功能失败: %d - %s (错误: %v)", functionID, functionName, errorMsg)
	}

	return nil
}

// StatsEventHandler 统计事件处理器
type StatsEventHandler struct {
	name       string
	eventTypes []events.EventType
	priority   int

	// 统计数据
	functionStats map[int]*FunctionStats
	mu            sync.RWMutex
}

// FunctionStats 功能统计
type FunctionStats struct {
	FunctionID      int           `json:"function_id"`
	ExecutionCount  int           `json:"execution_count"`
	SuccessCount    int           `json:"success_count"`
	FailureCount    int           `json:"failure_count"`
	TotalDuration   time.Duration `json:"total_duration"`
	AverageDuration time.Duration `json:"average_duration"`
	LastExecution   time.Time     `json:"last_execution"`
}

// NewStatsEventHandler 创建统计事件处理器
func NewStatsEventHandler() *StatsEventHandler {
	return &StatsEventHandler{
		name: "StatsEventHandler",
		eventTypes: []events.EventType{
			events.EventFunctionCompleted,
			events.EventFunctionFailed,
		},
		priority:      2,
		functionStats: make(map[int]*FunctionStats),
	}
}

// GetName 获取处理器名称
func (seh *StatsEventHandler) GetName() string {
	return seh.name
}

// GetEventTypes 获取处理的事件类型
func (seh *StatsEventHandler) GetEventTypes() []events.EventType {
	return seh.eventTypes
}

// GetPriority 获取处理器优先级
func (seh *StatsEventHandler) GetPriority() int {
	return seh.priority
}

// Handle 处理事件
func (seh *StatsEventHandler) Handle(ctx context.Context, event *events.Event) error {
	functionID, ok := event.Data["function_id"].(int)
	if !ok {
		return fmt.Errorf("invalid function_id in event data")
	}

	seh.mu.Lock()
	defer seh.mu.Unlock()

	stats, exists := seh.functionStats[functionID]
	if !exists {
		stats = &FunctionStats{
			FunctionID: functionID,
		}
		seh.functionStats[functionID] = stats
	}

	stats.ExecutionCount++
	stats.LastExecution = time.Now()

	if duration, ok := event.Data["duration"].(time.Duration); ok {
		stats.TotalDuration += duration
		stats.AverageDuration = stats.TotalDuration / time.Duration(stats.ExecutionCount)
	}

	switch event.Type {
	case events.EventFunctionCompleted:
		stats.SuccessCount++
	case events.EventFunctionFailed:
		stats.FailureCount++
	}

	return nil
}

// GetFunctionStats 获取功能统计
func (seh *StatsEventHandler) GetFunctionStats() map[int]*FunctionStats {
	seh.mu.RLock()
	defer seh.mu.RUnlock()

	result := make(map[int]*FunctionStats)
	for id, stats := range seh.functionStats {
		result[id] = &FunctionStats{
			FunctionID:      stats.FunctionID,
			ExecutionCount:  stats.ExecutionCount,
			SuccessCount:    stats.SuccessCount,
			FailureCount:    stats.FailureCount,
			TotalDuration:   stats.TotalDuration,
			AverageDuration: stats.AverageDuration,
			LastExecution:   stats.LastExecution,
		}
	}

	return result
}
