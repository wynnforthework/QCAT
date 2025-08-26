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
	dependencyTracker map[int][]int // 功能ID -> 依赖的功能ID列表
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
		dependencyTracker: make(map[int][]int),
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
			// 发送依赖满足事件
			dependencyEvent := &Event{
				Type:   EventDependencyMet,
				Source: "WorkflowCoordinator",
				Priority: PriorityHigh,
				Data: map[string]interface{}{
					"function_id": depFunctionID,
					"dependencies": dependencies,
				},
				CorrelationID: event.CorrelationID,
			}
			
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
				Type:   EventDependencyFailed,
				Source: "WorkflowCoordinator",
				Priority: PriorityCritical,
				Data: map[string]interface{}{
					"function_id": affectedID,
					"failed_dependency": functionID,
					"error": event.Data["error"],
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
	
	// 这里可以触发功能执行
	// 实际实现中应该调用工作流引擎的执行方法
	
	return nil
}

// ResourceManagerHandler 资源管理器处理器
type ResourceManagerHandler struct {
	*AutomationEventHandler
	resourceUsage map[string]int // 资源类型 -> 当前使用量
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
			Type:   EventResourceExhausted,
			Source: "ResourceManager",
			Priority: PriorityHigh,
			Data: map[string]interface{}{
				"resource_type": resourceType,
				"usage": rmh.resourceUsage[resourceType],
				"limit": rmh.resourceLimits[resourceType],
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
		Type:   EventResourceAcquired,
		Source: "ResourceManager",
		Priority: PriorityNormal,
		Data: map[string]interface{}{
			"resource_type": resourceType,
			"function_id": functionID,
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
		Type:   EventResourceReleased,
		Source: "ResourceManager",
		Priority: PriorityNormal,
		Data: map[string]interface{}{
			"resource_type": resourceType,
			"function_id": functionID,
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
					Type:   EventConflictDetected,
					Source: "ConflictDetector",
					Priority: PriorityCritical,
					Data: map[string]interface{}{
						"function_id": functionID,
						"conflict_with": conflictID,
						"message": fmt.Sprintf("功能 %d 与正在运行的功能 %d 冲突", functionID, conflictID),
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
					Type:   EventConflictResolved,
					Source: "ConflictDetector",
					Priority: PriorityHigh,
					Data: map[string]interface{}{
						"function_id": waitingID,
						"resolved_conflict": functionID,
						"message": fmt.Sprintf("功能 %d 的冲突已解除，可以开始执行", waitingID),
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
