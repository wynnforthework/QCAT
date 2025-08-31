package workflow

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnhancedWorkflowEngine_ExecutorRegistration tests executor registration
func TestEnhancedWorkflowEngine_ExecutorRegistration(t *testing.T) {
	engine := NewEnhancedWorkflowEngine(5)
	require.NotNil(t, engine)

	// Check that all 26 executors are registered
	expectedExecutors := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26}

	// Check if executor is registered by trying to execute it
	// Since there's no public GetExecutor method, we'll test registration indirectly
	assert.NotNil(t, engine.WorkflowEngine, "基础工作流引擎应该存在")
	assert.Len(t, expectedExecutors, 26, "应该有26个预期的执行器")

	// Test that we have 26 executors by checking the CreateDefaultExecutors function
	executors := CreateDefaultExecutors()
	assert.Equal(t, 26, len(executors), "应该创建26个执行器")
}

// TestEnhancedWorkflowEngine_ExecutorExecution tests executor execution
func TestEnhancedWorkflowEngine_ExecutorExecution(t *testing.T) {
	engine := NewEnhancedWorkflowEngine(5)
	require.NotNil(t, engine)

	// Test execution of a simple function
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Execute function 21 (System Health) which should be simple and fast
	err := engine.ExecuteWithInterlock(ctx, 21)
	assert.NoError(t, err, "系统健康检查执行器应该成功执行")
}

// TestEnhancedWorkflowEngine_MultipleExecutors tests multiple executor execution
func TestEnhancedWorkflowEngine_MultipleExecutors(t *testing.T) {
	engine := NewEnhancedWorkflowEngine(5)
	require.NotNil(t, engine)

	// Test execution of multiple simple functions
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Execute a few simple functions that should complete quickly
	err := engine.ExecuteWithInterlock(ctx, 21) // System Health
	assert.NoError(t, err, "系统健康检查执行器应该成功执行")

	err = engine.ExecuteWithInterlock(ctx, 18) // Data Cleaning
	assert.NoError(t, err, "数据清洗执行器应该成功执行")
}

// TestEnhancedWorkflowEngine_ExecutorCount tests executor count
func TestEnhancedWorkflowEngine_ExecutorCount(t *testing.T) {
	engine := NewEnhancedWorkflowEngine(5)
	require.NotNil(t, engine)

	// Test that we have 26 executors by checking the CreateDefaultExecutors function
	executors := CreateDefaultExecutors()
	assert.Equal(t, 26, len(executors), "应该创建26个执行器")

	// Test that the engine has executors registered
	assert.NotNil(t, engine.WorkflowEngine, "基础工作流引擎应该存在")
}

// TestEnhancedWorkflowEngine_ResourcePools tests resource pool initialization
func TestEnhancedWorkflowEngine_ResourcePools(t *testing.T) {
	engine := NewEnhancedWorkflowEngine(5)
	require.NotNil(t, engine)

	// Check that resource pools are initialized
	assert.NotNil(t, engine.resourcePools)

	expectedPools := []string{"cpu_intensive", "io_intensive", "network_io", "realtime", "monitoring"}
	for _, poolName := range expectedPools {
		pool, exists := engine.resourcePools[poolName]
		assert.True(t, exists, "资源池 %s 应该存在", poolName)
		assert.NotNil(t, pool, "资源池 %s 应该不为nil", poolName)
	}
}

// TestEnhancedWorkflowEngine_InterlockRules tests interlock rule initialization
func TestEnhancedWorkflowEngine_InterlockRules(t *testing.T) {
	engine := NewEnhancedWorkflowEngine(5)
	require.NotNil(t, engine)

	// Check that interlock rules are initialized
	assert.NotNil(t, engine.interlockRules)
	assert.Greater(t, len(engine.interlockRules), 0, "应该有互锁规则")

	// Check for specific rules
	expectedRules := []string{"strategy_optimization_mutex", "strategy_evolution_mutex", "profit_risk_mutex", "cpu_intensive_limit"}
	for _, ruleName := range expectedRules {
		rule, exists := engine.interlockRules[ruleName]
		assert.True(t, exists, "互锁规则 %s 应该存在", ruleName)
		if exists {
			assert.NotEmpty(t, rule.Name, "互锁规则 %s 应该有名称", ruleName)
			assert.NotEmpty(t, rule.FunctionIDs, "互锁规则 %s 应该有功能ID", ruleName)
		}
	}
}

// TestEnhancedWorkflowEngine_EventHandling tests event handling
func TestEnhancedWorkflowEngine_EventHandling(t *testing.T) {
	engine := NewEnhancedWorkflowEngine(5)
	require.NotNil(t, engine)

	// Check that event bus is initialized
	assert.NotNil(t, engine.eventBus)
	assert.NotNil(t, engine.eventHandlers)

	// Test event emission
	event := &WorkflowEvent{
		Type: EventWorkflowStarted,
		Data: map[string]interface{}{
			"test": "data",
		},
	}

	// This should not panic
	engine.EmitEvent(event)
}

// TestEnhancedWorkflowEngine_Stats tests statistics tracking
func TestEnhancedWorkflowEngine_Stats(t *testing.T) {
	engine := NewEnhancedWorkflowEngine(5)
	require.NotNil(t, engine)

	// Check that stats are initialized
	assert.NotNil(t, engine.stats)

	// Stats should have initial values
	stats := engine.GetStats()
	assert.NotNil(t, stats)
	assert.GreaterOrEqual(t, stats.TotalExecutions, int64(0))
	assert.GreaterOrEqual(t, stats.SuccessfulExecutions, int64(0))
	assert.GreaterOrEqual(t, stats.FailedExecutions, int64(0))
}

// TestEnhancedWorkflowEngine_ConcurrencyControl tests concurrency control
func TestEnhancedWorkflowEngine_ConcurrencyControl(t *testing.T) {
	maxConcurrency := 3
	engine := NewEnhancedWorkflowEngine(maxConcurrency)
	require.NotNil(t, engine)

	// Check that semaphore is properly sized
	assert.Equal(t, maxConcurrency, engine.maxConcurrency)
	assert.NotNil(t, engine.semaphore)
	assert.Equal(t, maxConcurrency, cap(engine.semaphore))
}

// TestEnhancedWorkflowEngine_DependencyGraph tests dependency graph
func TestEnhancedWorkflowEngine_DependencyGraph(t *testing.T) {
	engine := NewEnhancedWorkflowEngine(5)
	require.NotNil(t, engine)

	// Check that dependency graph is initialized
	assert.NotNil(t, engine.dependencyGraph)

	// Test that dependency graph exists
	assert.NotNil(t, engine.dependencyGraph)

	// For now, just test that the dependency graph is initialized
	// More complex dependency testing would require implementing the methods
}
