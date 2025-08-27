package scheduler

import (
	"context"
	"testing"
	"time"

	"qcat/internal/events"
)

func TestTaskScheduler_Basic(t *testing.T) {
	// 创建事件总线
	eventBus := events.NewEventBus(&events.EventBusConfig{
		BufferSize: 100,
		MaxRetries: 3,
		RetryDelay: time.Second,
	})

	// 事件总线在创建时自动启动
	defer eventBus.Stop()

	// 创建调度器
	config := GetDefaultSchedulerConfig()
	config.EnableEventDriven = false // 简化测试

	scheduler := NewTaskScheduler(config, eventBus)

	// 启动调度器
	if err := scheduler.Start(); err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer scheduler.Stop()

	// 验证调度器状态
	if !scheduler.IsRunning() {
		t.Error("Scheduler should be running")
	}

	// 获取统计信息
	stats := scheduler.GetStats()
	if stats == nil {
		t.Error("Stats should not be nil")
	}

	if stats.TotalTasks != 0 {
		t.Errorf("Expected 0 total tasks, got %d", stats.TotalTasks)
	}
}

func TestTaskScheduler_AddAndExecuteTask(t *testing.T) {
	// 创建调度器
	config := GetDefaultSchedulerConfig()
	config.EnableEventDriven = false
	config.EnableCronTasks = false
	config.EnablePeriodicTasks = false

	scheduler := NewTaskScheduler(config, nil)

	// 启动调度器
	if err := scheduler.Start(); err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer scheduler.Stop()

	// 创建测试任务处理器
	executed := false
	handler := NewSimpleTaskHandler("test-handler", "Test handler", func(ctx context.Context, task *ScheduledTask) error {
		executed = true
		return nil
	})

	// 创建一次性任务
	task := &ScheduledTask{
		Name:     "test-task",
		Type:     TaskTypeOneTime,
		Category: CategorySystem,
		Handler:  handler,
		Delay:    100 * time.Millisecond,
		Enabled:  true,
		Priority: 5,
		Timeout:  5 * time.Second,
	}

	// 添加任务
	if err := scheduler.AddTask(task); err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	// 等待任务执行
	time.Sleep(200 * time.Millisecond)

	// 验证任务已执行
	if !executed {
		t.Error("Task should have been executed")
	}

	// 验证任务状态
	retrievedTask, err := scheduler.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}

	if retrievedTask.Status != TaskStatusCompleted {
		t.Errorf("Expected task status to be completed, got %s", retrievedTask.Status)
	}

	if retrievedTask.RunCount != 1 {
		t.Errorf("Expected run count to be 1, got %d", retrievedTask.RunCount)
	}
}

func TestTaskScheduler_PeriodicTask(t *testing.T) {
	// 创建调度器
	config := GetDefaultSchedulerConfig()
	config.EnableEventDriven = false
	config.EnableCronTasks = false

	scheduler := NewTaskScheduler(config, nil)

	// 启动调度器
	if err := scheduler.Start(); err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer scheduler.Stop()

	// 创建测试任务处理器
	executionCount := 0
	handler := NewSimpleTaskHandler("periodic-handler", "Periodic handler", func(ctx context.Context, task *ScheduledTask) error {
		executionCount++
		return nil
	})

	// 创建周期任务
	task := &ScheduledTask{
		Name:     "periodic-task",
		Type:     TaskTypePeriodic,
		Category: CategorySystem,
		Handler:  handler,
		Interval: 100 * time.Millisecond,
		Delay:    50 * time.Millisecond,
		Enabled:  true,
		Priority: 5,
		Timeout:  5 * time.Second,
	}

	// 添加任务
	if err := scheduler.AddTask(task); err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	// 等待任务执行多次
	time.Sleep(350 * time.Millisecond)

	// 验证任务执行了多次
	if executionCount < 2 {
		t.Errorf("Expected at least 2 executions, got %d", executionCount)
	}

	// 禁用任务
	if err := scheduler.DisableTask(task.ID); err != nil {
		t.Fatalf("Failed to disable task: %v", err)
	}

	// 等待一段时间，确保任务不再执行
	previousCount := executionCount
	time.Sleep(200 * time.Millisecond)

	if executionCount > previousCount {
		t.Error("Task should not execute after being disabled")
	}
}

func TestTaskScheduler_TaskManagement(t *testing.T) {
	// 创建调度器
	config := GetDefaultSchedulerConfig()
	config.EnableEventDriven = false
	config.EnableCronTasks = false
	config.EnablePeriodicTasks = false

	scheduler := NewTaskScheduler(config, nil)

	// 创建测试任务处理器
	handler := NewLogTaskHandler("Test message")

	// 创建任务
	task := &ScheduledTask{
		Name:     "management-test-task",
		Type:     TaskTypeOneTime,
		Category: CategorySystem,
		Handler:  handler,
		Delay:    1 * time.Second,
		Enabled:  false, // 初始禁用
		Priority: 5,
		Timeout:  5 * time.Second,
	}

	// 添加任务
	if err := scheduler.AddTask(task); err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	// 验证任务已添加
	retrievedTask, err := scheduler.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}

	if retrievedTask.Name != task.Name {
		t.Errorf("Expected task name %s, got %s", task.Name, retrievedTask.Name)
	}

	// 获取所有任务
	allTasks := scheduler.GetAllTasks()
	if len(allTasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(allTasks))
	}

	// 启用任务
	if err := scheduler.EnableTask(task.ID); err != nil {
		t.Fatalf("Failed to enable task: %v", err)
	}

	// 验证任务已启用
	retrievedTask, _ = scheduler.GetTask(task.ID)
	if !retrievedTask.Enabled {
		t.Error("Task should be enabled")
	}

	// 禁用任务
	if err := scheduler.DisableTask(task.ID); err != nil {
		t.Fatalf("Failed to disable task: %v", err)
	}

	// 验证任务已禁用
	retrievedTask, _ = scheduler.GetTask(task.ID)
	if retrievedTask.Enabled {
		t.Error("Task should be disabled")
	}

	// 移除任务
	if err := scheduler.RemoveTask(task.ID); err != nil {
		t.Fatalf("Failed to remove task: %v", err)
	}

	// 验证任务已移除
	_, err = scheduler.GetTask(task.ID)
	if err == nil {
		t.Error("Task should have been removed")
	}

	// 验证任务列表为空
	allTasks = scheduler.GetAllTasks()
	if len(allTasks) != 0 {
		t.Errorf("Expected 0 tasks, got %d", len(allTasks))
	}
}

func TestTaskScheduler_TaskHandlers(t *testing.T) {
	// 测试日志处理器
	logHandler := NewLogTaskHandler("Test log message")
	if logHandler.GetName() != "LogTaskHandler" {
		t.Errorf("Expected handler name LogTaskHandler, got %s", logHandler.GetName())
	}

	// 测试健康检查处理器
	healthCheckCalled := false
	healthHandler := NewHealthCheckTaskHandler("test-service", func() error {
		healthCheckCalled = true
		return nil
	})

	task := &ScheduledTask{
		Name: "health-check-test",
		Type: TaskTypeOneTime,
	}

	err := healthHandler.Execute(context.Background(), task)
	if err != nil {
		t.Errorf("Health check handler should not return error: %v", err)
	}

	if !healthCheckCalled {
		t.Error("Health check function should have been called")
	}

	// 测试备份处理器
	backupCalled := false
	backupHandler := NewBackupTaskHandler("database", func(backupType string) error {
		backupCalled = true
		if backupType != "database" {
			t.Errorf("Expected backup type database, got %s", backupType)
		}
		return nil
	})

	err = backupHandler.Execute(context.Background(), task)
	if err != nil {
		t.Errorf("Backup handler should not return error: %v", err)
	}

	if !backupCalled {
		t.Error("Backup function should have been called")
	}
}
