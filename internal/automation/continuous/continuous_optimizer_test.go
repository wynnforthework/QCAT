package continuous

import (
	"testing"
)

func TestContinuousOptimizer_Creation(t *testing.T) {
	// 由于NewContinuousOptimizer需要真实的数据库连接，
	// 我们在这里只测试配置的创建和验证

	// 测试默认配置创建
	config := DefaultOptimizationConfig()
	if config == nil {
		t.Fatal("默认配置不应该为空")
	}

	if !config.EnableContinuousMode {
		t.Error("默认应该启用持续模式")
	}

	if config.OptimizationInterval <= 0 {
		t.Error("优化间隔应该大于0")
	}

	if config.StrategyOptimization == nil {
		t.Error("策略优化配置不应该为空")
	}

	// 在实际部署中，这里会创建真实的优化器
	// optimizer, err := NewContinuousOptimizer(cfg, db)
	// 但在单元测试中，我们跳过数据库依赖
}

func TestContinuousOptimizer_DefaultConfig(t *testing.T) {
	config := DefaultOptimizationConfig()

	if config == nil {
		t.Fatal("默认配置不应该为空")
	}

	if !config.EnableContinuousMode {
		t.Error("默认应该启用持续模式")
	}

	if config.OptimizationInterval <= 0 {
		t.Error("优化间隔应该大于0")
	}

	if config.BacktestInterval <= 0 {
		t.Error("回测间隔应该大于0")
	}

	if config.AnalysisInterval <= 0 {
		t.Error("分析间隔应该大于0")
	}

	if config.StrategyOptimization == nil {
		t.Error("策略优化配置不应该为空")
	}

	if config.BacktestOptimization == nil {
		t.Error("回测优化配置不应该为空")
	}

	if config.MarketAnalysis == nil {
		t.Error("市场分析配置不应该为空")
	}

	if config.PerformanceTracking == nil {
		t.Error("性能跟踪配置不应该为空")
	}

	if config.ResourceManagement == nil {
		t.Error("资源管理配置不应该为空")
	}
}

func TestTaskManager_Operations(t *testing.T) {
	config := &ResourceManagementConfig{
		MaxConcurrentTasks: 3,
		MaxCPUUsage:        80.0,
		MaxMemoryUsage:     80.0,
		TaskPrioritization: true,
		ResourceMonitoring: true,
	}

	taskManager, err := NewTaskManager(config)
	if err != nil {
		t.Fatalf("创建任务管理器失败: %v", err)
	}

	if taskManager == nil {
		t.Fatal("任务管理器不应该为空")
	}

	if len(taskManager.workers) != 3 {
		t.Errorf("期望工作者数量 3，实际得到 %d", len(taskManager.workers))
	}

	// 测试资源使用率获取
	resourceUsage := taskManager.GetResourceUsage()
	if resourceUsage < 0 || resourceUsage > 100 {
		t.Errorf("资源使用率应该在0-100之间，实际得到 %.2f", resourceUsage)
	}

	// 测试任务执行检查
	canExecute := taskManager.CanExecuteTask("test_task")
	if !canExecute {
		t.Error("应该能够执行任务")
	}
}

func TestOptimizationConfig_Validation(t *testing.T) {
	config := DefaultOptimizationConfig()

	// 测试策略优化配置
	if config.StrategyOptimization.MaxConcurrentStrategies <= 0 {
		t.Error("最大并发策略数应该大于0")
	}

	if config.StrategyOptimization.MinActiveStrategies <= 0 {
		t.Error("最小活跃策略数应该大于0")
	}

	if len(config.StrategyOptimization.OptimizationMethods) == 0 {
		t.Error("优化方法列表不应该为空")
	}

	// 测试回测优化配置
	if config.BacktestOptimization.MaxConcurrentBacktests <= 0 {
		t.Error("最大并发回测数应该大于0")
	}

	if len(config.BacktestOptimization.ValidationMethods) == 0 {
		t.Error("验证方法列表不应该为空")
	}

	// 测试市场分析配置
	if !config.MarketAnalysis.EnableContinuousAnalysis {
		t.Error("默认应该启用持续分析")
	}

	// 测试性能跟踪配置
	if !config.PerformanceTracking.EnableRealTimeTracking {
		t.Error("默认应该启用实时跟踪")
	}

	if len(config.PerformanceTracking.MetricsCollection) == 0 {
		t.Error("指标收集列表不应该为空")
	}

	// 测试资源管理配置
	if config.ResourceManagement.MaxCPUUsage <= 0 || config.ResourceManagement.MaxCPUUsage > 100 {
		t.Error("最大CPU使用率应该在0-100之间")
	}

	if config.ResourceManagement.MaxMemoryUsage <= 0 || config.ResourceManagement.MaxMemoryUsage > 100 {
		t.Error("最大内存使用率应该在0-100之间")
	}
}
