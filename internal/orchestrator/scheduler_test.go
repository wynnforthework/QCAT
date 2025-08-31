package orchestrator

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewScheduler tests scheduler creation
func TestNewScheduler(t *testing.T) {
	scheduler := NewScheduler()
	require.NotNil(t, scheduler)
	assert.NotNil(t, scheduler.cron)
	assert.NotNil(t, scheduler.tasks)
	assert.NotNil(t, scheduler.handlers)
}

// TestSchedulerLifecycle tests start and stop functionality
func TestSchedulerLifecycle(t *testing.T) {
	scheduler := NewScheduler()
	require.NotNil(t, scheduler)

	// Test start
	err := scheduler.Start()
	assert.NoError(t, err)

	// Give some time for cron to start
	time.Sleep(10 * time.Millisecond)

	// Test stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = scheduler.Stop(ctx)
	assert.NoError(t, err)
}

// TestRegisterHandler tests task handler registration
func TestRegisterHandler(t *testing.T) {
	scheduler := NewScheduler()
	require.NotNil(t, scheduler)

	// Create mock handler
	mockHandler := &MockTaskHandler{
		name: "test-handler",
	}

	// Test register handler
	scheduler.RegisterHandler(TaskTypeMarketHealth, mockHandler)

	// Verify handler was registered
	scheduler.mu.RLock()
	handler, exists := scheduler.handlers[TaskTypeMarketHealth]
	scheduler.mu.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, mockHandler, handler)
}

// TestAddTask tests adding scheduled tasks
func TestAddTask(t *testing.T) {
	scheduler := NewScheduler()
	require.NotNil(t, scheduler)

	// Register a handler first
	mockHandler := &MockTaskHandler{
		name: "test-handler",
	}
	scheduler.RegisterHandler(TaskTypeMarketHealth, mockHandler)

	// Test add task
	task := &Task{
		ID:       "test-task-1",
		Type:     TaskTypeMarketHealth,
		Schedule: "0 */5 * * * *", // Every 5 minutes
		Status:   TaskStatusPending,
	}

	err := scheduler.AddTask(task)
	assert.NoError(t, err)

	// Verify task was added
	scheduler.mu.RLock()
	addedTask, exists := scheduler.tasks["test-task-1"]
	scheduler.mu.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, task.ID, addedTask.ID)
	assert.Equal(t, task.Type, addedTask.Type)
	assert.Equal(t, task.Schedule, addedTask.Schedule)
}

// TestRemoveTask tests removing scheduled tasks
func TestRemoveTask(t *testing.T) {
	scheduler := NewScheduler()
	require.NotNil(t, scheduler)

	// Add a task first
	task := &Task{
		ID:       "test-task-2",
		Type:     TaskTypeStrategyScore,
		Schedule: "0 0 * * * *", // Every hour
		Status:   TaskStatusPending,
	}

	scheduler.mu.Lock()
	scheduler.tasks[task.ID] = task
	scheduler.mu.Unlock()

	// Test remove task
	err := scheduler.RemoveTask("test-task-2")
	assert.NoError(t, err)

	// Verify task was removed
	scheduler.mu.RLock()
	_, exists := scheduler.tasks["test-task-2"]
	scheduler.mu.RUnlock()

	assert.False(t, exists)
}

// TestGetTask tests retrieving tasks
func TestGetTask(t *testing.T) {
	scheduler := NewScheduler()
	require.NotNil(t, scheduler)

	// Add a task
	task := &Task{
		ID:       "test-task-3",
		Type:     TaskTypeHotlistScan,
		Schedule: "0 0 9 * * *", // Daily at 9 AM
		Status:   TaskStatusPending,
	}

	scheduler.mu.Lock()
	scheduler.tasks[task.ID] = task
	scheduler.mu.Unlock()

	// Test get task
	retrievedTask := scheduler.GetTask("test-task-3")
	require.NotNil(t, retrievedTask)
	assert.Equal(t, task.ID, retrievedTask.ID)
	assert.Equal(t, task.Type, retrievedTask.Type)
	assert.Equal(t, task.Schedule, retrievedTask.Schedule)

	// Test get non-existent task
	nonExistentTask := scheduler.GetTask("non-existent")
	assert.Nil(t, nonExistentTask)
}

// TestListTasks tests listing all tasks
func TestListTasks(t *testing.T) {
	scheduler := NewScheduler()
	require.NotNil(t, scheduler)

	// Add multiple tasks
	tasks := []*Task{
		{
			ID:       "task-1",
			Type:     TaskTypeMarketHealth,
			Schedule: "0 */5 * * * *",
			Status:   TaskStatusPending,
		},
		{
			ID:       "task-2",
			Type:     TaskTypeStrategyScore,
			Schedule: "0 0 * * * *",
			Status:   TaskStatusRunning,
		},
		{
			ID:       "task-3",
			Type:     TaskTypeDailyOptimize,
			Schedule: "0 0 2 * * *",
			Status:   TaskStatusCompleted,
		},
	}

	scheduler.mu.Lock()
	for _, task := range tasks {
		scheduler.tasks[task.ID] = task
	}
	scheduler.mu.Unlock()

	// Test list tasks
	allTasks := scheduler.ListTasks()
	assert.Len(t, allTasks, 3)

	// Verify all tasks are present
	taskIDs := make(map[string]bool)
	for _, task := range allTasks {
		taskIDs[task.ID] = true
	}

	assert.True(t, taskIDs["task-1"])
	assert.True(t, taskIDs["task-2"])
	assert.True(t, taskIDs["task-3"])
}

// TestTaskExecution tests task execution
func TestTaskExecution(t *testing.T) {
	scheduler := NewScheduler()
	require.NotNil(t, scheduler)

	// Create mock handler
	mockHandler := &MockTaskHandler{
		name: "test-handler",
	}
	scheduler.RegisterHandler(TaskTypeMarketHealth, mockHandler)

	// Start scheduler
	err := scheduler.Start()
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		scheduler.Stop(ctx)
	}()

	// Add a task that runs immediately (for testing)
	task := &Task{
		ID:       "immediate-task",
		Type:     TaskTypeMarketHealth,
		Schedule: "@every 1s", // Run every second for testing
		Status:   TaskStatusPending,
	}

	err = scheduler.AddTask(task)
	assert.NoError(t, err)

	// Wait for task to execute
	time.Sleep(2 * time.Second)

	// Verify handler was called
	assert.True(t, mockHandler.executeCalled)
	assert.Equal(t, task, mockHandler.lastTask)
}

// TestTaskStatus tests task status management
func TestTaskStatus(t *testing.T) {
	scheduler := NewScheduler()
	require.NotNil(t, scheduler)

	task := &Task{
		ID:       "status-test-task",
		Type:     TaskTypeStrategyScore,
		Schedule: "0 0 * * * *",
		Status:   TaskStatusPending,
	}

	scheduler.mu.Lock()
	scheduler.tasks[task.ID] = task
	scheduler.mu.Unlock()

	// Test update task status
	scheduler.updateTaskStatus("status-test-task", TaskStatusRunning, "")

	// Verify status was updated
	scheduler.mu.RLock()
	updatedTask := scheduler.tasks["status-test-task"]
	scheduler.mu.RUnlock()

	assert.Equal(t, TaskStatusRunning, updatedTask.Status)
	assert.True(t, !updatedTask.LastRunTime.IsZero())
}

// TestConcurrentAccess tests concurrent access to scheduler
func TestConcurrentAccess(t *testing.T) {
	scheduler := NewScheduler()
	require.NotNil(t, scheduler)

	var wg sync.WaitGroup
	numGoroutines := 10

	// Test concurrent task additions
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			task := &Task{
				ID:       fmt.Sprintf("concurrent-task-%d", id),
				Type:     TaskTypeMarketHealth,
				Schedule: "0 0 * * * *",
				Status:   TaskStatusPending,
			}
			scheduler.AddTask(task)
		}(i)
	}

	wg.Wait()

	// Verify all tasks were added
	allTasks := scheduler.ListTasks()
	assert.Len(t, allTasks, numGoroutines)
}

// MockTaskHandler implements TaskHandler interface for testing
type MockTaskHandler struct {
	name          string
	executeCalled bool
	lastTask      *Task
	executeError  error
	mu            sync.Mutex
}

func (m *MockTaskHandler) Execute(ctx context.Context, task *Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.executeCalled = true
	m.lastTask = task
	return m.executeError
}

func (m *MockTaskHandler) GetName() string {
	return m.name
}

func (m *MockTaskHandler) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.executeCalled = false
	m.lastTask = nil
	m.executeError = nil
}
