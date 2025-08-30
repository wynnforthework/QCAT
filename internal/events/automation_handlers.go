package events

import (
	"context"
	"fmt"
	"log"
	"time"
)

// AutomationEventHandler 自动化事件处理器基类
type AutomationEventHandler struct {
	name       string
	eventTypes []EventType
	priority   int
}

// GetName 获取处理器名称
func (aeh *AutomationEventHandler) GetName() string {
	return aeh.name
}

// GetEventTypes 获取处理的事件类型
func (aeh *AutomationEventHandler) GetEventTypes() []EventType {
	return aeh.eventTypes
}

// GetPriority 获取处理器优先级
func (aeh *AutomationEventHandler) GetPriority() int {
	return aeh.priority
}

// WorkflowCoordinatorHandler 工作流协调器处理器
type WorkflowCoordinatorHandler struct {
	*AutomationEventHandler
	dependencyTracker  map[int][]int // 功能ID -> 依赖的功能ID列表
	completedFunctions map[int]bool  // 已完成的功能
}

// NewWorkflowCoordinatorHandler 创建工作流协调器处理器
func NewWorkflowCoordinatorHandler() *WorkflowCoordinatorHandler {
	return &WorkflowCoordinatorHandler{
		AutomationEventHandler: &AutomationEventHandler{
			name: "WorkflowCoordinator",
			eventTypes: []EventType{
				EventFunctionCompleted,
				EventFunctionFailed,
				EventDependencyMet,
			},
			priority: 10,
		},
		dependencyTracker:  make(map[int][]int),
		completedFunctions: make(map[int]bool),
	}
}

// Handle 处理事件
func (wch *WorkflowCoordinatorHandler) Handle(ctx context.Context, event *Event) error {
	switch event.Type {
	case EventFunctionCompleted:
		return wch.handleFunctionCompleted(ctx, event)
	case EventFunctionFailed:
		return wch.handleFunctionFailed(ctx, event)
	case EventDependencyMet:
		return wch.handleDependencyMet(ctx, event)
	}

	return nil
}

// handleFunctionCompleted 处理功能完成事件
func (wch *WorkflowCoordinatorHandler) handleFunctionCompleted(ctx context.Context, event *Event) error {
	functionID, ok := event.Data["function_id"].(int)
	if !ok {
		return fmt.Errorf("invalid function_id in event data")
	}

	wch.completedFunctions[functionID] = true

	log.Printf("功能 %d 执行完成，检查依赖功能", functionID)

	// 检查哪些功能的依赖现在满足了
	for depFunctionID, dependencies := range wch.dependencyTracker {
		if wch.completedFunctions[depFunctionID] {
			continue // 已经完成的功能跳过
		}

		allDependenciesMet := true
		for _, depID := range dependencies {
			if !wch.completedFunctions[depID] {
				allDependenciesMet = false
				break
			}
		}

		if allDependenciesMet {
			// 这里应该发布事件，但需要访问事件总线
			log.Printf("功能 %d 的依赖已满足，可以开始执行", depFunctionID)
		}
	}

	return nil
}

// handleFunctionFailed 处理功能失败事件
func (wch *WorkflowCoordinatorHandler) handleFunctionFailed(ctx context.Context, event *Event) error {
	functionID, ok := event.Data["function_id"].(int)
	if !ok {
		return fmt.Errorf("invalid function_id in event data")
	}

	log.Printf("功能 %d 执行失败，检查影响", functionID)

	// 检查哪些功能依赖于失败的功能
	affectedFunctions := make([]int, 0)
	for depFunctionID, dependencies := range wch.dependencyTracker {
		for _, depID := range dependencies {
			if depID == functionID {
				affectedFunctions = append(affectedFunctions, depFunctionID)
				break
			}
		}
	}

	if len(affectedFunctions) > 0 {
		log.Printf("功能 %d 失败影响了功能: %v", functionID, affectedFunctions)

		// 发送依赖失败事件
		for _, affectedID := range affectedFunctions {
			failureEvent := &Event{
				Type:     EventDependencyFailed,
				Source:   "WorkflowCoordinator",
				Priority: PriorityCritical,
				Data: map[string]interface{}{
					"function_id":       affectedID,
					"failed_dependency": functionID,
					"error":             event.Data["error"],
				},
				CorrelationID: event.CorrelationID,
			}

			log.Printf("功能 %d 因依赖 %d 失败而受影响", affectedID, functionID)
			_ = failureEvent // 这里应该发布事件
		}
	}

	return nil
}

// handleDependencyMet 处理依赖满足事件
func (wch *WorkflowCoordinatorHandler) handleDependencyMet(ctx context.Context, event *Event) error {
	functionID, ok := event.Data["function_id"].(int)
	if !ok {
		return fmt.Errorf("invalid function_id in event data")
	}

	log.Printf("功能 %d 的依赖已满足，准备执行", functionID)

	// 触发功能执行
	if err := wch.executeFunctionWorkflow(ctx, functionID); err != nil {
		log.Printf("执行功能 %d 失败: %v", functionID, err)
		
		// 发布功能执行失败事件
		failureEvent := &Event{
			Type:      EventFunctionFailed,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"function_id": functionID,
				"error":       err.Error(),
			},
		}
		
		// 这里应该发布事件到事件总线
		_ = failureEvent
		
		return fmt.Errorf("function execution failed: %w", err)
	}

	// 标记功能为已完成
	wch.completedFunctions[functionID] = true
	
	// 发布功能完成事件
	completionEvent := &Event{
		Type:      EventFunctionCompleted,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"function_id": functionID,
		},
	}
	
	// 这里应该发布事件到事件总线
	_ = completionEvent
	
	log.Printf("功能 %d 执行完成", functionID)
	return nil
}

// executeFunctionWorkflow 执行功能工作流
func (wch *WorkflowCoordinatorHandler) executeFunctionWorkflow(ctx context.Context, functionID int) error {
	log.Printf("开始执行功能 %d 的工作流", functionID)
	
	// 1. 获取功能配置
	functionConfig, err := wch.getFunctionConfig(functionID)
	if err != nil {
		return fmt.Errorf("failed to get function config: %w", err)
	}
	
	// 2. 验证执行条件
	if err := wch.validateExecutionConditions(ctx, functionID, functionConfig); err != nil {
		return fmt.Errorf("execution conditions not met: %w", err)
	}
	
	// 3. 准备执行环境
	executionContext, err := wch.prepareExecutionContext(ctx, functionID, functionConfig)
	if err != nil {
		return fmt.Errorf("failed to prepare execution context: %w", err)
	}
	
	// 4. 执行功能逻辑
	result, err := wch.executeFunction(ctx, functionID, functionConfig, executionContext)
	if err != nil {
		return fmt.Errorf("function execution failed: %w", err)
	}
	
	// 5. 处理执行结果
	if err := wch.processExecutionResult(ctx, functionID, result); err != nil {
		log.Printf("Failed to process execution result for function %d: %v", functionID, err)
		// 不返回错误，因为功能本身执行成功了
	}
	
	log.Printf("功能 %d 工作流执行完成", functionID)
	return nil
}

// getFunctionConfig 获取功能配置
func (wch *WorkflowCoordinatorHandler) getFunctionConfig(functionID int) (*FunctionConfig, error) {
	// 简化实现：返回模拟配置
	// 实际实现应该从配置文件或数据库获取
	config := &FunctionConfig{
		ID:          functionID,
		Name:        fmt.Sprintf("Function_%d", functionID),
		Type:        wch.determineFunctionType(functionID),
		Parameters:  wch.getDefaultParameters(functionID),
		Timeout:     30 * time.Second,
		RetryCount:  3,
		Enabled:     true,
	}
	
	log.Printf("获取功能 %d 配置: %s", functionID, config.Name)
	return config, nil
}

// validateExecutionConditions 验证执行条件
func (wch *WorkflowCoordinatorHandler) validateExecutionConditions(ctx context.Context, functionID int, config *FunctionConfig) error {
	// 1. 检查功能是否启用
	if !config.Enabled {
		return fmt.Errorf("function %d is disabled", functionID)
	}
	
	// 2. 检查资源可用性
	if err := wch.checkResourceAvailability(functionID); err != nil {
		return fmt.Errorf("resource check failed: %w", err)
	}
	
	// 3. 检查前置条件
	if err := wch.checkPreconditions(ctx, functionID); err != nil {
		return fmt.Errorf("preconditions not met: %w", err)
	}
	
	// 4. 检查并发限制
	if err := wch.checkConcurrencyLimits(functionID); err != nil {
		return fmt.Errorf("concurrency limits exceeded: %w", err)
	}
	
	return nil
}

// prepareExecutionContext 准备执行环境
func (wch *WorkflowCoordinatorHandler) prepareExecutionContext(ctx context.Context, functionID int, config *FunctionConfig) (*ExecutionContext, error) {
	executionContext := &ExecutionContext{
		FunctionID:  functionID,
		StartTime:   time.Now(),
		Parameters:  config.Parameters,
		Timeout:     config.Timeout,
		RetryCount:  config.RetryCount,
		Context:     ctx,
	}
	
	// 设置执行环境变量
	executionContext.Environment = map[string]string{
		"FUNCTION_ID":   fmt.Sprintf("%d", functionID),
		"EXECUTION_ID":  fmt.Sprintf("%d_%d", functionID, time.Now().Unix()),
		"TIMEOUT":       config.Timeout.String(),
	}
	
	log.Printf("为功能 %d 准备执行环境", functionID)
	return executionContext, nil
}

// executeFunction 执行功能逻辑
func (wch *WorkflowCoordinatorHandler) executeFunction(ctx context.Context, functionID int, config *FunctionConfig, execCtx *ExecutionContext) (*ExecutionResult, error) {
	log.Printf("执行功能 %d: %s", functionID, config.Name)
	
	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()
	
	// 根据功能类型执行不同的逻辑
	var result *ExecutionResult
	var err error
	
	switch config.Type {
	case "risk_monitoring":
		result, err = wch.executeRiskMonitoring(timeoutCtx, execCtx)
	case "position_adjustment":
		result, err = wch.executePositionAdjustment(timeoutCtx, execCtx)
	case "data_processing":
		result, err = wch.executeDataProcessing(timeoutCtx, execCtx)
	case "alert_notification":
		result, err = wch.executeAlertNotification(timeoutCtx, execCtx)
	case "system_maintenance":
		result, err = wch.executeSystemMaintenance(timeoutCtx, execCtx)
	default:
		result, err = wch.executeGenericFunction(timeoutCtx, execCtx)
	}
	
	if err != nil {
		return nil, fmt.Errorf("function execution failed: %w", err)
	}
	
	result.FunctionID = functionID
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(execCtx.StartTime)
	
	return result, nil
}

// processExecutionResult 处理执行结果
func (wch *WorkflowCoordinatorHandler) processExecutionResult(ctx context.Context, functionID int, result *ExecutionResult) error {
	log.Printf("处理功能 %d 的执行结果", functionID)
	
	// 1. 记录执行历史
	if err := wch.recordExecutionHistory(ctx, result); err != nil {
		log.Printf("Failed to record execution history: %v", err)
	}
	
	// 2. 更新性能指标
	wch.updatePerformanceMetrics(result)
	
	// 3. 检查结果质量
	if err := wch.validateResultQuality(result); err != nil {
		log.Printf("Result quality validation failed: %v", err)
	}
	
	// 4. 触发后续动作
	if err := wch.triggerFollowUpActions(ctx, result); err != nil {
		log.Printf("Failed to trigger follow-up actions: %v", err)
	}
	
	return nil
}

// Supporting structures
type FunctionConfig struct {
	ID         int                    `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
	Timeout    time.Duration          `json:"timeout"`
	RetryCount int                    `json:"retry_count"`
	Enabled    bool                   `json:"enabled"`
}

type ExecutionContext struct {
	FunctionID  int                    `json:"function_id"`
	StartTime   time.Time              `json:"start_time"`
	Parameters  map[string]interface{} `json:"parameters"`
	Timeout     time.Duration          `json:"timeout"`
	RetryCount  int                    `json:"retry_count"`
	Context     context.Context        `json:"-"`
	Environment map[string]string      `json:"environment"`
}

type ExecutionResult struct {
	FunctionID int                    `json:"function_id"`
	Success    bool                   `json:"success"`
	StartTime  time.Time              `json:"start_time"`
	EndTime    time.Time              `json:"end_time"`
	Duration   time.Duration          `json:"duration"`
	Output     map[string]interface{} `json:"output"`
	Error      string                 `json:"error,omitempty"`
	Metrics    map[string]float64     `json:"metrics"`
}

// Helper methods
func (wch *WorkflowCoordinatorHandler) determineFunctionType(functionID int) string {
	// 简化实现：根据功能ID确定类型
	switch functionID % 5 {
	case 0:
		return "risk_monitoring"
	case 1:
		return "position_adjustment"
	case 2:
		return "data_processing"
	case 3:
		return "alert_notification"
	case 4:
		return "system_maintenance"
	default:
		return "generic"
	}
}

func (wch *WorkflowCoordinatorHandler) getDefaultParameters(functionID int) map[string]interface{} {
	return map[string]interface{}{
		"function_id": functionID,
		"timestamp":   time.Now().Unix(),
		"version":     "1.0",
	}
}

func (wch *WorkflowCoordinatorHandler) checkResourceAvailability(functionID int) error {
	// 简化实现：检查系统资源
	// 实际实现应该检查CPU、内存、网络等资源
	log.Printf("检查功能 %d 的资源可用性", functionID)
	return nil
}

func (wch *WorkflowCoordinatorHandler) checkPreconditions(ctx context.Context, functionID int) error {
	// 简化实现：检查前置条件
	log.Printf("检查功能 %d 的前置条件", functionID)
	return nil
}

func (wch *WorkflowCoordinatorHandler) checkConcurrencyLimits(functionID int) error {
	// 简化实现：检查并发限制
	log.Printf("检查功能 %d 的并发限制", functionID)
	return nil
}

// Function execution implementations
func (wch *WorkflowCoordinatorHandler) executeRiskMonitoring(ctx context.Context, execCtx *ExecutionContext) (*ExecutionResult, error) {
	log.Printf("执行风险监控功能")
	
	// 模拟风险监控逻辑
	time.Sleep(100 * time.Millisecond)
	
	return &ExecutionResult{
		Success:   true,
		StartTime: execCtx.StartTime,
		Output: map[string]interface{}{
			"risk_level":    "LOW",
			"alerts_count":  0,
			"checked_items": 15,
		},
		Metrics: map[string]float64{
			"execution_time_ms": 100,
			"items_processed":   15,
		},
	}, nil
}

func (wch *WorkflowCoordinatorHandler) executePositionAdjustment(ctx context.Context, execCtx *ExecutionContext) (*ExecutionResult, error) {
	log.Printf("执行仓位调整功能")
	
	// 模拟仓位调整逻辑
	time.Sleep(200 * time.Millisecond)
	
	return &ExecutionResult{
		Success:   true,
		StartTime: execCtx.StartTime,
		Output: map[string]interface{}{
			"positions_adjusted": 3,
			"total_value":        150000.0,
			"adjustments": []map[string]interface{}{
				{"symbol": "BTCUSDT", "old_size": 1.0, "new_size": 1.2},
				{"symbol": "ETHUSDT", "old_size": 5.0, "new_size": 4.8},
			},
		},
		Metrics: map[string]float64{
			"execution_time_ms":    200,
			"positions_processed":  3,
			"adjustment_accuracy":  0.95,
		},
	}, nil
}

func (wch *WorkflowCoordinatorHandler) executeDataProcessing(ctx context.Context, execCtx *ExecutionContext) (*ExecutionResult, error) {
	log.Printf("执行数据处理功能")
	
	// 模拟数据处理逻辑
	time.Sleep(300 * time.Millisecond)
	
	return &ExecutionResult{
		Success:   true,
		StartTime: execCtx.StartTime,
		Output: map[string]interface{}{
			"records_processed": 1000,
			"data_quality":      0.98,
			"processing_rate":   "3333 records/sec",
		},
		Metrics: map[string]float64{
			"execution_time_ms":  300,
			"records_processed":  1000,
			"data_quality_score": 0.98,
		},
	}, nil
}

func (wch *WorkflowCoordinatorHandler) executeAlertNotification(ctx context.Context, execCtx *ExecutionContext) (*ExecutionResult, error) {
	log.Printf("执行告警通知功能")
	
	// 模拟告警通知逻辑
	time.Sleep(50 * time.Millisecond)
	
	return &ExecutionResult{
		Success:   true,
		StartTime: execCtx.StartTime,
		Output: map[string]interface{}{
			"notifications_sent": 5,
			"channels_used":      []string{"email", "slack", "webhook"},
			"delivery_rate":      1.0,
		},
		Metrics: map[string]float64{
			"execution_time_ms":   50,
			"notifications_sent":  5,
			"delivery_success_rate": 1.0,
		},
	}, nil
}

func (wch *WorkflowCoordinatorHandler) executeSystemMaintenance(ctx context.Context, execCtx *ExecutionContext) (*ExecutionResult, error) {
	log.Printf("执行系统维护功能")
	
	// 模拟系统维护逻辑
	time.Sleep(500 * time.Millisecond)
	
	return &ExecutionResult{
		Success:   true,
		StartTime: execCtx.StartTime,
		Output: map[string]interface{}{
			"tasks_completed": []string{"log_cleanup", "cache_refresh", "health_check"},
			"system_health":   "GOOD",
			"maintenance_duration": "500ms",
		},
		Metrics: map[string]float64{
			"execution_time_ms": 500,
			"tasks_completed":   3,
			"system_health_score": 0.95,
		},
	}, nil
}

func (wch *WorkflowCoordinatorHandler) executeGenericFunction(ctx context.Context, execCtx *ExecutionContext) (*ExecutionResult, error) {
	log.Printf("执行通用功能")
	
	// 模拟通用功能逻辑
	time.Sleep(150 * time.Millisecond)
	
	return &ExecutionResult{
		Success:   true,
		StartTime: execCtx.StartTime,
		Output: map[string]interface{}{
			"status":     "completed",
			"message":    "Generic function executed successfully",
		},
		Metrics: map[string]float64{
			"execution_time_ms": 150,
		},
	}, nil
}

func (wch *WorkflowCoordinatorHandler) recordExecutionHistory(ctx context.Context, result *ExecutionResult) error {
	// 简化实现：记录到日志
	log.Printf("记录功能 %d 执行历史: 成功=%t, 耗时=%v", 
		result.FunctionID, result.Success, result.Duration)
	return nil
}

func (wch *WorkflowCoordinatorHandler) updatePerformanceMetrics(result *ExecutionResult) {
	// 简化实现：更新性能指标
	log.Printf("更新功能 %d 性能指标", result.FunctionID)
}

func (wch *WorkflowCoordinatorHandler) validateResultQuality(result *ExecutionResult) error {
	// 简化实现：验证结果质量
	if !result.Success {
		return fmt.Errorf("execution failed")
	}
	return nil
}

func (wch *WorkflowCoordinatorHandler) triggerFollowUpActions(ctx context.Context, result *ExecutionResult) error {
	// 简化实现：触发后续动作
	log.Printf("触发功能 %d 的后续动作", result.FunctionID)
	return nil
}

// ResourceManagerHandler 资源管理器处理器
type ResourceManagerHandler struct {
	*AutomationEventHandler
	resourceUsage  map[string]int // 资源类型 -> 当前使用量
	resourceLimits map[string]int // 资源类型 -> 限制
}

// NewResourceManagerHandler 创建资源管理器处理器
func NewResourceManagerHandler() *ResourceManagerHandler {
	return &ResourceManagerHandler{
		AutomationEventHandler: &AutomationEventHandler{
			name: "ResourceManager",
			eventTypes: []EventType{
				EventResourceAcquired,
				EventResourceReleased,
				EventFunctionStarted,
				EventFunctionCompleted,
			},
			priority: 8,
		},
		resourceUsage: make(map[string]int),
		resourceLimits: map[string]int{
			"cpu_intensive": 2,
			"io_intensive":  4,
			"network_io":    6,
			"realtime":      8,
			"monitoring":    10,
		},
	}
}

// Handle 处理事件
func (rmh *ResourceManagerHandler) Handle(ctx context.Context, event *Event) error {
	switch event.Type {
	case EventResourceAcquired:
		return rmh.handleResourceAcquired(ctx, event)
	case EventResourceReleased:
		return rmh.handleResourceReleased(ctx, event)
	case EventFunctionStarted:
		return rmh.handleFunctionStarted(ctx, event)
	case EventFunctionCompleted:
		return rmh.handleFunctionCompleted(ctx, event)
	}

	return nil
}

// handleResourceAcquired 处理资源获取事件
func (rmh *ResourceManagerHandler) handleResourceAcquired(ctx context.Context, event *Event) error {
	resourceType, ok := event.Data["resource_type"].(string)
	if !ok {
		return fmt.Errorf("invalid resource_type in event data")
	}

	rmh.resourceUsage[resourceType]++

	log.Printf("资源 %s 被获取，当前使用量: %d/%d",
		resourceType, rmh.resourceUsage[resourceType], rmh.resourceLimits[resourceType])

	// 检查资源是否接近耗尽
	if rmh.resourceUsage[resourceType] >= rmh.resourceLimits[resourceType] {
		exhaustedEvent := &Event{
			Type:     EventResourceExhausted,
			Source:   "ResourceManager",
			Priority: PriorityHigh,
			Data: map[string]interface{}{
				"resource_type": resourceType,
				"usage":         rmh.resourceUsage[resourceType],
				"limit":         rmh.resourceLimits[resourceType],
			},
		}

		log.Printf("资源 %s 已耗尽", resourceType)
		_ = exhaustedEvent // 这里应该发布事件
	}

	return nil
}

// handleResourceReleased 处理资源释放事件
func (rmh *ResourceManagerHandler) handleResourceReleased(ctx context.Context, event *Event) error {
	resourceType, ok := event.Data["resource_type"].(string)
	if !ok {
		return fmt.Errorf("invalid resource_type in event data")
	}

	if rmh.resourceUsage[resourceType] > 0 {
		rmh.resourceUsage[resourceType]--
	}

	log.Printf("资源 %s 被释放，当前使用量: %d/%d",
		resourceType, rmh.resourceUsage[resourceType], rmh.resourceLimits[resourceType])

	return nil
}

// handleFunctionStarted 处理功能开始事件
func (rmh *ResourceManagerHandler) handleFunctionStarted(ctx context.Context, event *Event) error {
	functionID, ok := event.Data["function_id"].(int)
	if !ok {
		return fmt.Errorf("invalid function_id in event data")
	}

	// 根据功能ID确定资源类型
	resourceType := rmh.getResourceTypeForFunction(functionID)

	// 发送资源获取事件
	acquiredEvent := &Event{
		Type:     EventResourceAcquired,
		Source:   "ResourceManager",
		Priority: PriorityNormal,
		Data: map[string]interface{}{
			"resource_type": resourceType,
			"function_id":   functionID,
		},
		CorrelationID: event.CorrelationID,
	}

	return rmh.handleResourceAcquired(ctx, acquiredEvent)
}

// handleFunctionCompleted 处理功能完成事件
func (rmh *ResourceManagerHandler) handleFunctionCompleted(ctx context.Context, event *Event) error {
	functionID, ok := event.Data["function_id"].(int)
	if !ok {
		return fmt.Errorf("invalid function_id in event data")
	}

	// 根据功能ID确定资源类型
	resourceType := rmh.getResourceTypeForFunction(functionID)

	// 发送资源释放事件
	releasedEvent := &Event{
		Type:     EventResourceReleased,
		Source:   "ResourceManager",
		Priority: PriorityNormal,
		Data: map[string]interface{}{
			"resource_type": resourceType,
			"function_id":   functionID,
		},
		CorrelationID: event.CorrelationID,
	}

	return rmh.handleResourceReleased(ctx, releasedEvent)
}

// getResourceTypeForFunction 根据功能ID获取资源类型
func (rmh *ResourceManagerHandler) getResourceTypeForFunction(functionID int) string {
	// 根据功能特性映射资源类型
	switch functionID {
	case 1, 6, 8, 24, 25: // CPU密集型功能
		return "cpu_intensive"
	case 18, 19, 23: // IO密集型功能
		return "io_intensive"
	case 4, 10, 14, 22: // 网络IO功能
		return "network_io"
	case 5, 9, 12: // 实时功能
		return "realtime"
	case 13, 21: // 监控功能
		return "monitoring"
	default:
		return "cpu_intensive" // 默认
	}
}

// ConflictDetectorHandler 冲突检测处理器
type ConflictDetectorHandler struct {
	*AutomationEventHandler
	activeFunctions map[int]bool
	conflictRules   map[int][]int // 功能ID -> 冲突的功能ID列表
}

// NewConflictDetectorHandler 创建冲突检测处理器
func NewConflictDetectorHandler() *ConflictDetectorHandler {
	return &ConflictDetectorHandler{
		AutomationEventHandler: &AutomationEventHandler{
			name: "ConflictDetector",
			eventTypes: []EventType{
				EventFunctionStarted,
				EventFunctionCompleted,
				EventFunctionFailed,
			},
			priority: 9,
		},
		activeFunctions: make(map[int]bool),
		conflictRules: map[int][]int{
			1:  {6},     // 策略参数优化 与 周期性策略优化 冲突
			4:  {12},    // 智能建仓 与 异常行情应对 冲突
			6:  {1},     // 周期性策略优化 与 策略参数优化 冲突
			7:  {8},     // 策略淘汰 与 新策略引入 冲突
			8:  {7},     // 新策略引入 与 策略淘汰 冲突
			11: {12},    // 利润最大化引擎 与 异常行情应对 冲突
			12: {4, 11}, // 异常行情应对 与 智能建仓、利润最大化 冲突
			24: {25},    // 策略自学习 与 遗传升级 冲突
			25: {24},    // 遗传升级 与 策略自学习 冲突
		},
	}
}

// Handle 处理事件
func (cdh *ConflictDetectorHandler) Handle(ctx context.Context, event *Event) error {
	switch event.Type {
	case EventFunctionStarted:
		return cdh.handleFunctionStarted(ctx, event)
	case EventFunctionCompleted, EventFunctionFailed:
		return cdh.handleFunctionEnded(ctx, event)
	}

	return nil
}

// handleFunctionStarted 处理功能开始事件
func (cdh *ConflictDetectorHandler) handleFunctionStarted(ctx context.Context, event *Event) error {
	functionID, ok := event.Data["function_id"].(int)
	if !ok {
		return fmt.Errorf("invalid function_id in event data")
	}

	// 检查冲突
	if conflicts, exists := cdh.conflictRules[functionID]; exists {
		for _, conflictID := range conflicts {
			if cdh.activeFunctions[conflictID] {
				// 发现冲突
				conflictEvent := &Event{
					Type:     EventConflictDetected,
					Source:   "ConflictDetector",
					Priority: PriorityCritical,
					Data: map[string]interface{}{
						"function_id":   functionID,
						"conflict_with": conflictID,
						"message":       fmt.Sprintf("功能 %d 与正在运行的功能 %d 冲突", functionID, conflictID),
					},
					CorrelationID: event.CorrelationID,
				}

				log.Printf("检测到冲突: 功能 %d 与功能 %d", functionID, conflictID)
				_ = conflictEvent // 这里应该发布事件

				return fmt.Errorf("conflict detected between function %d and %d", functionID, conflictID)
			}
		}
	}

	cdh.activeFunctions[functionID] = true
	return nil
}

// handleFunctionEnded 处理功能结束事件
func (cdh *ConflictDetectorHandler) handleFunctionEnded(ctx context.Context, event *Event) error {
	functionID, ok := event.Data["function_id"].(int)
	if !ok {
		return fmt.Errorf("invalid function_id in event data")
	}

	delete(cdh.activeFunctions, functionID)

	// 检查是否有等待的冲突功能可以开始执行
	for waitingID, conflicts := range cdh.conflictRules {
		if cdh.activeFunctions[waitingID] {
			continue // 已经在运行
		}

		for _, conflictID := range conflicts {
			if conflictID == functionID {
				// 冲突解除
				resolvedEvent := &Event{
					Type:     EventConflictResolved,
					Source:   "ConflictDetector",
					Priority: PriorityHigh,
					Data: map[string]interface{}{
						"function_id":       waitingID,
						"resolved_conflict": functionID,
						"message":           fmt.Sprintf("功能 %d 的冲突已解除，可以开始执行", waitingID),
					},
					CorrelationID: event.CorrelationID,
				}

				log.Printf("冲突解除: 功能 %d 可以开始执行", waitingID)
				_ = resolvedEvent // 这里应该发布事件
				break
			}
		}
	}

	return nil
}
