package workflow

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiStrategyWorkflowSystem_StartStop(t *testing.T) {
	// 创建系统
	system, err := NewMultiStrategyWorkflowSystem(nil)
	require.NoError(t, err)
	require.NotNil(t, system)

	// 测试启动
	err = system.Start()
	require.NoError(t, err)
	assert.True(t, system.IsRunning())

	// 等待系统稳定
	time.Sleep(2 * time.Second)

	// 检查组件状态
	stats := system.GetSystemStats()
	assert.Equal(t, 4, stats.ComponentsRunning)
	assert.Equal(t, 4, stats.ComponentsTotal)

	// 测试停止
	err = system.Stop()
	require.NoError(t, err)
	assert.False(t, system.IsRunning())
}

func TestMultiStrategyWorkflowSystem_CreateStrategy(t *testing.T) {
	// 创建并启动系统
	system, err := NewMultiStrategyWorkflowSystem(nil)
	require.NoError(t, err)

	err = system.Start()
	require.NoError(t, err)
	defer system.Stop()

	// 等待系统稳定
	time.Sleep(1 * time.Second)

	// 创建策略
	strategyID, err := system.CreateAndRunStrategy("TestStrategy", "momentum")
	require.NoError(t, err)
	assert.NotEmpty(t, strategyID)

	// 等待策略创建完成
	time.Sleep(2 * time.Second)

	// 检查策略是否创建成功
	stats := system.GetSystemStats()
	assert.Greater(t, stats.TotalStrategies, int64(0))
}

func TestStrategyWorkflowEngine_Lifecycle(t *testing.T) {
	// 创建策略工作流引擎
	config := GetDefaultWorkflowConfig()
	config.OnboardingTimeout = 10 * time.Second
	config.BacktestTimeout = 15 * time.Second
	config.OptimizationTimeout = 20 * time.Second
	config.LearningTimeout = 25 * time.Second
	config.ApplicationTimeout = 10 * time.Second

	engine := NewStrategyWorkflowEngine("test_strategy", "TestStrategy", config)
	require.NotNil(t, engine)

	// 启动引擎
	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	// 执行生命周期
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- engine.ExecuteLifecycle()
	}()

	// 等待执行完成或超时
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-ctx.Done():
		t.Fatal("Lifecycle execution timed out")
	}

	// 检查统计信息
	stats := engine.GetStats()
	assert.Greater(t, stats.TotalJobs, int64(0))
	assert.Greater(t, stats.CompletedJobs, int64(0))
}

func TestMultiStrategyManager_ConcurrentStrategies(t *testing.T) {
	// 创建多策略管理器
	config := GetDefaultMultiStrategyConfig()
	config.MaxConcurrentStrategies = 3
	
	manager := NewMultiStrategyManager(config)
	require.NotNil(t, manager)

	// 启动管理器
	err := manager.Start()
	require.NoError(t, err)
	defer manager.Stop()

	// 创建多个策略
	strategyIDs := make([]string, 0)
	for i := 0; i < 3; i++ {
		strategyID := time.Now().Format("20060102150405") + string(rune('A'+i))
		engine, err := manager.CreateStrategyWorkflow(
			strategyID,
			"TestStrategy"+string(rune('A'+i)),
			"test",
		)
		require.NoError(t, err)
		require.NotNil(t, engine)
		strategyIDs = append(strategyIDs, strategyID)
	}

	// 等待策略创建完成
	time.Sleep(2 * time.Second)

	// 检查策略数量
	manager.enginesMu.RLock()
	assert.Equal(t, 3, len(manager.strategyEngines))
	manager.enginesMu.RUnlock()

	// 测试超出限制
	_, err = manager.CreateStrategyWorkflow("overflow", "OverflowStrategy", "test")
	assert.Error(t, err)

	// 清理策略
	for _, strategyID := range strategyIDs {
		err := manager.RemoveStrategyWorkflow(strategyID)
		assert.NoError(t, err)
	}
}

func TestEvolutionManager_Evolution(t *testing.T) {
	// 创建进化管理器
	config := GetDefaultEvolutionConfig()
	config.EvaluationInterval = 5 * time.Second
	config.PopulationSize = 5

	manager := NewEvolutionManager(config, nil)
	require.NotNil(t, manager)

	// 启动管理器
	err := manager.Start()
	require.NoError(t, err)
	defer manager.Stop()

	// 等待至少一次进化
	time.Sleep(10 * time.Second)

	// 检查种群
	manager.populationMu.RLock()
	populationSize := len(manager.population)
	manager.populationMu.RUnlock()

	assert.Equal(t, config.PopulationSize, populationSize)

	// 检查是否有进化历史
	assert.Greater(t, len(manager.generationHistory), 0)
}

func TestTradingStrategyPool_Integration(t *testing.T) {
	// 创建多策略管理器
	multiManager := NewMultiStrategyManager(nil)
	require.NotNil(t, multiManager)

	err := multiManager.Start()
	require.NoError(t, err)
	defer multiManager.Stop()

	// 创建策略池
	pool := NewTradingStrategyPool(multiManager, nil)
	require.NotNil(t, pool)

	err = pool.Start()
	require.NoError(t, err)
	defer pool.Stop()

	// 创建一个策略
	engine, err := multiManager.CreateStrategyWorkflow("test_strategy", "TestStrategy", "test")
	require.NoError(t, err)
	require.NotNil(t, engine)

	// 模拟策略启用
	multiManager.enableStrategy("test_strategy")

	// 等待同步
	time.Sleep(2 * time.Second)

	// 检查策略池
	enabledStrategies := pool.GetEnabledStrategyObjects()
	assert.Greater(t, len(enabledStrategies), 0)

	// 检查策略是否启用
	assert.True(t, pool.IsStrategyEnabled("test_strategy"))

	// 获取策略信息
	strategyInfo, err := pool.GetStrategyInfo("test_strategy")
	require.NoError(t, err)
	assert.Equal(t, "test_strategy", strategyInfo.ID)
}

func TestSystemStats_Updates(t *testing.T) {
	// 创建系统
	system, err := NewMultiStrategyWorkflowSystem(nil)
	require.NoError(t, err)

	err = system.Start()
	require.NoError(t, err)
	defer system.Stop()

	// 等待统计更新
	time.Sleep(3 * time.Second)

	// 获取统计信息
	stats := system.GetSystemStats()
	assert.NotZero(t, stats.StartTime)
	assert.Greater(t, stats.Uptime, time.Duration(0))
	assert.Equal(t, 4, stats.ComponentsRunning)
	assert.Equal(t, 4, stats.ComponentsTotal)
	assert.NotZero(t, stats.LastUpdateTime)
}

func TestResourceAllocation(t *testing.T) {
	// 创建多策略管理器
	config := GetDefaultMultiStrategyConfig()
	config.GlobalCPUQuota = 10.0
	config.GlobalMemoryQuota = 10 * 1024 * 1024 * 1024 // 10GB
	config.MaxConcurrentStrategies = 5

	manager := NewMultiStrategyManager(config)
	require.NotNil(t, manager)

	err := manager.Start()
	require.NoError(t, err)
	defer manager.Stop()

	// 创建策略并检查资源分配
	strategyID := "resource_test_strategy"
	engine, err := manager.CreateStrategyWorkflow(strategyID, "ResourceTestStrategy", "test")
	require.NoError(t, err)
	require.NotNil(t, engine)

	// 检查资源分配
	manager.globalResourceManager.mu.RLock()
	allocation, exists := manager.globalResourceManager.allocations[strategyID]
	manager.globalResourceManager.mu.RUnlock()

	assert.True(t, exists)
	assert.Greater(t, allocation.CPU, 0.0)
	assert.Greater(t, allocation.Memory, int64(0))

	// 移除策略并检查资源释放
	err = manager.RemoveStrategyWorkflow(strategyID)
	require.NoError(t, err)

	manager.globalResourceManager.mu.RLock()
	_, exists = manager.globalResourceManager.allocations[strategyID]
	manager.globalResourceManager.mu.RUnlock()

	assert.False(t, exists)
}

// 基准测试
func BenchmarkStrategyCreation(b *testing.B) {
	system, err := NewMultiStrategyWorkflowSystem(nil)
	require.NoError(b, err)

	err = system.Start()
	require.NoError(b, err)
	defer system.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		strategyID, err := system.CreateAndRunStrategy("BenchStrategy", "test")
		require.NoError(b, err)
		require.NotEmpty(b, strategyID)
	}
}

func BenchmarkEvolutionCycle(b *testing.B) {
	config := GetDefaultEvolutionConfig()
	config.PopulationSize = 10
	
	manager := NewEvolutionManager(config, nil)
	require.NoError(b, manager.Start())
	defer manager.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := manager.performEvolution()
		require.NoError(b, err)
	}
}
