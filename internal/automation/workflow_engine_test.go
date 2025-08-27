package automation

import (
	"context"
	"testing"
	"time"

	"qcat/internal/events"
)

func TestAutomationWorkflowEngine_Basic(t *testing.T) {
	// 创建配置
	config := GetDefaultWorkflowEngineConfig()
	config.EnablePoolManager = false // 简化测试
	config.MonitorInterval = 100 * time.Millisecond
	config.HealthCheckInterval = 200 * time.Millisecond

	// 创建工作流引擎
	engine := NewAutomationWorkflowEngine(config)

	// 验证引擎创建
	if engine == nil {
		t.Fatal("Engine should not be nil")
	}

	// 验证初始状态
	if engine.IsRunning() {
		t.Error("Engine should not be running initially")
	}

	// 获取统计信息
	stats := engine.GetStats()
	if stats == nil {
		t.Error("Stats should not be nil")
	}

	if stats.TotalTasks != 0 {
		t.Errorf("Expected 0 total tasks, got %d", stats.TotalTasks)
	}
}

func TestAutomationWorkflowEngine_StartStop(t *testing.T) {
	// 创建配置
	config := GetDefaultWorkflowEngineConfig()
	config.EnablePoolManager = false
	config.MonitorInterval = 100 * time.Millisecond

	// 创建工作流引擎
	engine := NewAutomationWorkflowEngine(config)

	// 启动引擎
	if err := engine.Start(); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}

	// 验证运行状态
	if !engine.IsRunning() {
		t.Error("Engine should be running after start")
	}

	// 等待一段时间让组件初始化
	time.Sleep(200 * time.Millisecond)

	// 停止引擎
	if err := engine.Stop(); err != nil {
		t.Fatalf("Failed to stop engine: %v", err)
	}

	// 验证停止状态
	if engine.IsRunning() {
		t.Error("Engine should not be running after stop")
	}
}

func TestAutomationWorkflowEngine_TaskManagement(t *testing.T) {
	// 创建配置
	config := GetDefaultWorkflowEngineConfig()
	config.EnablePoolManager = false
	config.EnableScheduler = false // 暂时禁用调度器

	// 创建工作流引擎
	engine := NewAutomationWorkflowEngine(config)

	// 启动引擎
	if err := engine.Start(); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}
	defer engine.Stop()

	// 测试工作流执行
	ctx := context.Background()
	err := engine.ExecuteWorkflow(ctx)
	if err != nil {
		t.Logf("Workflow execution returned error (expected): %v", err)
	}

	// 验证活跃任务
	activeTasks := engine.GetActiveTasks()
	t.Logf("Active tasks count: %d", len(activeTasks))
}

func TestAutomationWorkflowEngine_Stats(t *testing.T) {
	// 创建配置
	config := GetDefaultWorkflowEngineConfig()
	config.EnablePoolManager = false
	config.MonitorInterval = 50 * time.Millisecond

	// 创建工作流引擎
	engine := NewAutomationWorkflowEngine(config)

	// 启动引擎
	if err := engine.Start(); err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}
	defer engine.Stop()

	// 等待统计信息更新
	time.Sleep(100 * time.Millisecond)

	// 获取统计信息
	stats := engine.GetStats()
	if stats == nil {
		t.Error("Stats should not be nil")
	}

	// 验证统计信息
	if stats.StartTime.IsZero() {
		t.Error("Start time should be set")
	}

	if stats.Uptime == 0 {
		t.Error("Uptime should be greater than 0")
	}

	// 模拟任务完成
	engine.updateStatsAfterTask(true, 100*time.Millisecond)

	// 获取更新后的统计信息
	updatedStats := engine.GetStats()
	if updatedStats.TotalTasks != 1 {
		t.Errorf("Expected 1 total task, got %d", updatedStats.TotalTasks)
	}

	if updatedStats.CompletedTasks != 1 {
		t.Errorf("Expected 1 completed task, got %d", updatedStats.CompletedTasks)
	}

	// 模拟任务失败
	engine.updateStatsAfterTask(false, 50*time.Millisecond)

	// 获取更新后的统计信息
	finalStats := engine.GetStats()
	if finalStats.TotalTasks != 2 {
		t.Errorf("Expected 2 total tasks, got %d", finalStats.TotalTasks)
	}

	if finalStats.FailedTasks != 1 {
		t.Errorf("Expected 1 failed task, got %d", finalStats.FailedTasks)
	}
}

func TestAutomationWorkflowEngine_ActiveTasks(t *testing.T) {
	// 创建配置
	config := GetDefaultWorkflowEngineConfig()
	config.EnablePoolManager = false

	// 创建工作流引擎
	engine := NewAutomationWorkflowEngine(config)

	// 验证初始状态
	activeTasks := engine.GetActiveTasks()
	if len(activeTasks) != 0 {
		t.Errorf("Expected 0 active tasks, got %d", len(activeTasks))
	}

	// 手动添加活跃任务
	task := &ActiveTask{
		ID:        "test-task-1",
		Name:      "Test Task",
		Type:      "test",
		Status:    "running",
		StartTime: time.Now(),
		Progress:  50.0,
		Context:   map[string]interface{}{"key": "value"},
	}

	engine.tasksMu.Lock()
	engine.activeTasks[task.ID] = task
	engine.tasksMu.Unlock()

	// 验证活跃任务
	activeTasks = engine.GetActiveTasks()
	if len(activeTasks) != 1 {
		t.Errorf("Expected 1 active task, got %d", len(activeTasks))
	}

	if activeTasks[0].ID != task.ID {
		t.Errorf("Expected task ID %s, got %s", task.ID, activeTasks[0].ID)
	}

	if activeTasks[0].Progress != 50.0 {
		t.Errorf("Expected progress 50.0, got %f", activeTasks[0].Progress)
	}
}

func TestAutomationWorkflowEngine_Components(t *testing.T) {
	// 创建配置
	config := GetDefaultWorkflowEngineConfig()

	// 创建工作流引擎
	engine := NewAutomationWorkflowEngine(config)

	// 验证组件
	if engine.GetWorkflowEngine() == nil {
		t.Error("Workflow engine should not be nil")
	}

	if engine.GetTaskScheduler() == nil {
		t.Error("Task scheduler should not be nil")
	}

	if engine.GetEventBus() == nil {
		t.Error("Event bus should not be nil")
	}

	if engine.GetPoolManager() == nil {
		t.Error("Pool manager should not be nil")
	}
}

func TestWorkflowEventHandler(t *testing.T) {
	// 创建工作流引擎
	engine := NewAutomationWorkflowEngine(nil)

	// 创建事件处理器
	handler := &WorkflowEventHandler{engine: engine}

	// 验证处理器属性
	if handler.GetName() != "WorkflowEventHandler" {
		t.Errorf("Expected handler name WorkflowEventHandler, got %s", handler.GetName())
	}

	if handler.GetPriority() != 5 {
		t.Errorf("Expected priority 5, got %d", handler.GetPriority())
	}

	eventTypes := handler.GetEventTypes()
	if len(eventTypes) != 6 {
		t.Errorf("Expected 6 event types, got %d", len(eventTypes))
	}

	// 测试事件处理
	ctx := context.Background()
	event := &events.Event{
		Type:   events.EventWorkflowStarted,
		Source: "test",
		Data: map[string]interface{}{
			"workflow_id": "test-workflow",
		},
	}

	err := handler.Handle(ctx, event)
	if err != nil {
		t.Errorf("Event handling should not return error: %v", err)
	}
}
