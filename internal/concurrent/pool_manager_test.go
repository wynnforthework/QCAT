package concurrent

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestTask 测试任务
type TestTask struct {
	ID       string
	Priority int
	Duration time.Duration
	Result   chan string
}

func (t *TestTask) Execute(ctx context.Context) error {
	// 模拟任务执行
	time.Sleep(t.Duration)

	select {
	case t.Result <- fmt.Sprintf("Task %s completed", t.ID):
	default:
	}

	return nil
}

func (t *TestTask) GetID() string {
	return t.ID
}

func (t *TestTask) GetPriority() int {
	return t.Priority
}

func (t *TestTask) GetTimeout() time.Duration {
	return 30 * time.Second // 默认30秒超时
}

func TestPoolManager_Basic(t *testing.T) {
	// 创建测试配置
	config := GetDefaultConfig()
	config.DefaultPool.MaxWorkers = 2
	config.DefaultPool.QueueSize = 10
	config.Monitor.Enabled = false // 禁用监控以简化测试

	// 创建池管理器
	pm := NewPoolManager(config)

	// 启动池管理器
	if err := pm.Start(); err != nil {
		t.Fatalf("Failed to start pool manager: %v", err)
	}
	defer pm.Stop()

	// 测试提交任务
	task := &TestTask{
		ID:       "test-1",
		Priority: 5,
		Duration: 100 * time.Millisecond,
		Result:   make(chan string, 1),
	}

	err := pm.SubmitTask(task, "default")
	if err != nil {
		t.Errorf("Failed to submit task: %v", err)
	}

	// 等待任务完成
	select {
	case result := <-task.Result:
		if result != "Task test-1 completed" {
			t.Errorf("Unexpected result: %s", result)
		}
	case <-time.After(5 * time.Second):
		t.Error("Task execution timeout")
	}
}

func TestPoolManager_MultiplePoolsWithPriority(t *testing.T) {
	// 创建测试配置
	config := GetDefaultConfig()
	config.DefaultPool.MaxWorkers = 2
	config.HighPriorityPool.MaxWorkers = 1
	config.LowPriorityPool.MaxWorkers = 1
	config.Monitor.Enabled = false

	// 创建池管理器
	pm := NewPoolManager(config)

	// 启动池管理器
	if err := pm.Start(); err != nil {
		t.Fatalf("Failed to start pool manager: %v", err)
	}
	defer pm.Stop()

	// 创建不同优先级的任务
	tasks := []*TestTask{
		{ID: "high-1", Priority: 9, Duration: 50 * time.Millisecond, Result: make(chan string, 1)},
		{ID: "normal-1", Priority: 5, Duration: 50 * time.Millisecond, Result: make(chan string, 1)},
		{ID: "low-1", Priority: 1, Duration: 50 * time.Millisecond, Result: make(chan string, 1)},
	}

	// 提交任务
	for _, task := range tasks {
		err := pm.SubmitTaskWithPriority(task, task.Priority)
		if err != nil {
			t.Errorf("Failed to submit task %s: %v", task.ID, err)
		}
	}

	// 等待所有任务完成
	var wg sync.WaitGroup
	for _, task := range tasks {
		wg.Add(1)
		go func(testTask *TestTask) {
			defer wg.Done()
			select {
			case result := <-testTask.Result:
				if result != fmt.Sprintf("Task %s completed", testTask.ID) {
					t.Errorf("Unexpected result for task %s: %s", testTask.ID, result)
				}
			case <-time.After(5 * time.Second):
				t.Errorf("Task %s execution timeout", testTask.ID)
			}
		}(task)
	}

	wg.Wait()
}

func TestPoolManager_LoadBalancer(t *testing.T) {
	// 创建测试配置
	config := GetDefaultConfig()
	config.DefaultPool.MaxWorkers = 1
	config.HighPriorityPool.MaxWorkers = 1
	config.LoadBalancer.Enabled = true
	config.LoadBalancer.Strategy = "round_robin"
	config.Monitor.Enabled = false

	// 创建池管理器
	pm := NewPoolManager(config)

	// 启动池管理器
	if err := pm.Start(); err != nil {
		t.Fatalf("Failed to start pool manager: %v", err)
	}
	defer pm.Stop()

	// 创建多个任务
	taskCount := 5
	tasks := make([]*TestTask, taskCount)
	for i := 0; i < taskCount; i++ {
		tasks[i] = &TestTask{
			ID:       fmt.Sprintf("task-%d", i),
			Priority: 5,
			Duration: 50 * time.Millisecond,
			Result:   make(chan string, 1),
		}
	}

	// 提交任务（使用负载均衡器）
	for _, task := range tasks {
		err := pm.SubmitTask(task, "") // 空字符串表示使用负载均衡器
		if err != nil {
			t.Errorf("Failed to submit task %s: %v", task.ID, err)
		}
	}

	// 等待所有任务完成
	var wg sync.WaitGroup
	for _, task := range tasks {
		wg.Add(1)
		go func(testTask *TestTask) {
			defer wg.Done()
			select {
			case result := <-testTask.Result:
				if result != fmt.Sprintf("Task %s completed", testTask.ID) {
					t.Errorf("Unexpected result for task %s: %s", testTask.ID, result)
				}
			case <-time.After(5 * time.Second):
				t.Errorf("Task %s execution timeout", testTask.ID)
			}
		}(task)
	}

	wg.Wait()
}

func TestPoolManager_GetStats(t *testing.T) {
	// 创建测试配置
	config := GetDefaultConfig()
	config.Monitor.Enabled = false

	// 创建池管理器
	pm := NewPoolManager(config)

	// 启动池管理器
	if err := pm.Start(); err != nil {
		t.Fatalf("Failed to start pool manager: %v", err)
	}
	defer pm.Stop()

	// 获取统计信息
	stats := pm.GetStats()

	// 验证统计信息
	if stats == nil {
		t.Error("Stats should not be nil")
	}

	pools, exists := stats["pools"]
	if !exists {
		t.Error("Stats should contain pools information")
	}

	poolsMap, ok := pools.(map[string]interface{})
	if !ok {
		t.Error("Pools should be a map")
	}

	// 检查是否包含预期的池
	expectedPools := []string{"default", "high_priority", "low_priority", "strategy", "data_processing"}
	for _, poolName := range expectedPools {
		if _, exists := poolsMap[poolName]; !exists {
			t.Errorf("Pool %s should exist in stats", poolName)
		}
	}
}

func TestPoolManager_GetPool(t *testing.T) {
	// 创建测试配置
	config := GetDefaultConfig()
	config.Monitor.Enabled = false

	// 创建池管理器
	pm := NewPoolManager(config)

	// 测试获取存在的池
	pool, err := pm.GetPool("default")
	if err != nil {
		t.Errorf("Failed to get default pool: %v", err)
	}
	if pool == nil {
		t.Error("Default pool should not be nil")
	}

	// 测试获取不存在的池
	_, err = pm.GetPool("nonexistent")
	if err == nil {
		t.Error("Should return error for nonexistent pool")
	}
}

func TestPoolManager_StopAndStart(t *testing.T) {
	// 创建测试配置
	config := GetDefaultConfig()
	config.Monitor.Enabled = false

	// 创建池管理器
	pm := NewPoolManager(config)

	// 启动
	if err := pm.Start(); err != nil {
		t.Fatalf("Failed to start pool manager: %v", err)
	}

	// 提交一个任务确保系统正常工作
	task := &TestTask{
		ID:       "test-stop-start",
		Priority: 5,
		Duration: 50 * time.Millisecond,
		Result:   make(chan string, 1),
	}

	err := pm.SubmitTask(task, "default")
	if err != nil {
		t.Errorf("Failed to submit task: %v", err)
	}

	// 等待任务完成
	select {
	case <-task.Result:
		// 任务完成
	case <-time.After(2 * time.Second):
		t.Error("Task execution timeout")
	}

	// 停止
	if err := pm.Stop(); err != nil {
		t.Errorf("Failed to stop pool manager: %v", err)
	}

	// 尝试提交任务到已停止的池管理器（应该失败）
	task2 := &TestTask{
		ID:       "test-after-stop",
		Priority: 5,
		Duration: 50 * time.Millisecond,
		Result:   make(chan string, 1),
	}

	err = pm.SubmitTask(task2, "default")
	if err == nil {
		t.Error("Should not be able to submit task to stopped pool manager")
	}
}
