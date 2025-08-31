package main

import (
	"log"
	"time"

	"qcat/internal/automation/continuous"
)

func main() {
	log.Println("🎯 持续运行优化系统演示")
	log.Println("========================================")

	// 创建自定义优化配置
	config := continuous.DefaultOptimizationConfig()

	// 调整配置以便演示
	config.OptimizationInterval = 3 * time.Second // 3秒优化间隔
	config.BacktestInterval = 5 * time.Second     // 5秒回测间隔
	config.AnalysisInterval = 2 * time.Second     // 2秒分析间隔

	// 策略优化配置
	config.StrategyOptimization.EnableAutoIntroduction = true
	config.StrategyOptimization.MaxConcurrentStrategies = 20
	config.StrategyOptimization.MinActiveStrategies = 5
	config.StrategyOptimization.ParameterOptimization = true

	// 回测优化配置
	config.BacktestOptimization.EnableContinuousBacktest = true
	config.BacktestOptimization.MaxConcurrentBacktests = 3
	config.BacktestOptimization.AutoValidation = true

	// 市场分析配置
	config.MarketAnalysis.EnableContinuousAnalysis = true
	config.MarketAnalysis.PatternRecognition = true
	config.MarketAnalysis.TrendAnalysis = true
	config.MarketAnalysis.AnomalyDetection = true

	// 性能跟踪配置
	config.PerformanceTracking.EnableRealTimeTracking = true
	config.PerformanceTracking.PerformanceAlerts = true
	config.PerformanceTracking.AutoReporting = true

	// 资源管理配置
	config.ResourceManagement.MaxCPUUsage = 70.0
	config.ResourceManagement.MaxMemoryUsage = 80.0
	config.ResourceManagement.MaxConcurrentTasks = 5
	config.ResourceManagement.TaskPrioritization = true
	config.ResourceManagement.ResourceMonitoring = true

	log.Println("📊 优化配置:")
	log.Printf("   - 持续模式: %v", config.EnableContinuousMode)
	log.Printf("   - 优化间隔: %v", config.OptimizationInterval)
	log.Printf("   - 回测间隔: %v", config.BacktestInterval)
	log.Printf("   - 分析间隔: %v", config.AnalysisInterval)
	log.Printf("   - 最大并发任务: %d", config.ResourceManagement.MaxConcurrentTasks)
	log.Printf("   - CPU使用限制: %.1f%%", config.ResourceManagement.MaxCPUUsage)

	// 演示任务管理器功能
	log.Println("\n🔧 演示任务管理器功能:")
	taskManager, err := continuous.NewTaskManager(config.ResourceManagement)
	if err != nil {
		log.Fatalf("创建任务管理器失败: %v", err)
	}

	log.Printf("✅ 任务管理器创建成功，工作者数量: %d", config.ResourceManagement.MaxConcurrentTasks)

	// 检查资源使用情况
	resourceUsage := taskManager.GetResourceUsage()
	log.Printf("📊 当前系统资源使用率: %.2f%%", resourceUsage)

	// 检查是否可以执行任务
	canExecute := taskManager.CanExecuteTask("optimization")
	log.Printf("🔍 是否可以执行优化任务: %v", canExecute)

	canExecuteBacktest := taskManager.CanExecuteTask("backtest")
	log.Printf("🔍 是否可以执行回测任务: %v", canExecuteBacktest)

	canExecuteAnalysis := taskManager.CanExecuteTask("analysis")
	log.Printf("🔍 是否可以执行分析任务: %v", canExecuteAnalysis)

	// 演示配置验证
	log.Println("\n📋 配置验证:")

	// 策略优化配置验证
	if config.StrategyOptimization.MaxConcurrentStrategies > 0 {
		log.Printf("✅ 最大并发策略数: %d", config.StrategyOptimization.MaxConcurrentStrategies)
	}

	if config.StrategyOptimization.MinActiveStrategies > 0 {
		log.Printf("✅ 最小活跃策略数: %d", config.StrategyOptimization.MinActiveStrategies)
	}

	if len(config.StrategyOptimization.OptimizationMethods) > 0 {
		log.Printf("✅ 优化方法: %v", config.StrategyOptimization.OptimizationMethods)
	}

	// 回测优化配置验证
	if config.BacktestOptimization.MaxConcurrentBacktests > 0 {
		log.Printf("✅ 最大并发回测数: %d", config.BacktestOptimization.MaxConcurrentBacktests)
	}

	if len(config.BacktestOptimization.ValidationMethods) > 0 {
		log.Printf("✅ 验证方法: %v", config.BacktestOptimization.ValidationMethods)
	}

	// 市场分析配置验证
	analysisFeatures := []string{}
	if config.MarketAnalysis.PatternRecognition {
		analysisFeatures = append(analysisFeatures, "模式识别")
	}
	if config.MarketAnalysis.TrendAnalysis {
		analysisFeatures = append(analysisFeatures, "趋势分析")
	}
	if config.MarketAnalysis.VolatilityAnalysis {
		analysisFeatures = append(analysisFeatures, "波动率分析")
	}
	if config.MarketAnalysis.AnomalyDetection {
		analysisFeatures = append(analysisFeatures, "异常检测")
	}
	log.Printf("✅ 市场分析功能: %v", analysisFeatures)

	// 性能跟踪配置验证
	if len(config.PerformanceTracking.MetricsCollection) > 0 {
		log.Printf("✅ 性能指标收集: %v", config.PerformanceTracking.MetricsCollection)
	}

	// 演示统计信息结构
	log.Println("\n📊 统计信息结构演示:")
	stats := &continuous.OptimizationStats{
		StartTime:               time.Now(),
		Uptime:                  0,
		StrategiesOptimized:     0,
		StrategiesIntroduced:    0,
		ParameterOptimizations:  0,
		BacktestsCompleted:      0,
		ValidationsPassed:       0,
		ValidationsFailed:       0,
		AnalysisRuns:            0,
		PatternsDetected:        0,
		AnomaliesDetected:       0,
		AverageOptimizationTime: 0,
		AverageBacktestTime:     0,
		SystemResourceUsage:     resourceUsage,
		LastUpdated:             time.Now(),
	}

	log.Printf("📈 初始统计信息:")
	log.Printf("   - 启动时间: %v", stats.StartTime.Format("2006-01-02 15:04:05"))
	log.Printf("   - 运行时间: %v", stats.Uptime)
	log.Printf("   - 优化策略数: %d", stats.StrategiesOptimized)
	log.Printf("   - 引入策略数: %d", stats.StrategiesIntroduced)
	log.Printf("   - 完成回测数: %d", stats.BacktestsCompleted)
	log.Printf("   - 分析运行数: %d", stats.AnalysisRuns)
	log.Printf("   - 检测模式数: %d", stats.PatternsDetected)
	log.Printf("   - 检测异常数: %d", stats.AnomaliesDetected)
	log.Printf("   - 系统资源使用: %.2f%%", stats.SystemResourceUsage)

	// 模拟运行一段时间后的统计更新
	log.Println("\n⏰ 模拟运行5秒后的统计更新...")
	time.Sleep(5 * time.Second)

	// 更新统计信息
	stats.Uptime = time.Since(stats.StartTime)
	stats.StrategiesOptimized = 3
	stats.StrategiesIntroduced = 1
	stats.ParameterOptimizations = 8
	stats.BacktestsCompleted = 2
	stats.ValidationsPassed = 2
	stats.AnalysisRuns = 5
	stats.PatternsDetected = 12
	stats.AnomaliesDetected = 1
	stats.AverageOptimizationTime = 45 * time.Second
	stats.AverageBacktestTime = 2 * time.Minute
	stats.SystemResourceUsage = taskManager.GetResourceUsage()
	stats.LastUpdated = time.Now()

	log.Printf("📈 更新后统计信息:")
	log.Printf("   - 运行时间: %v", stats.Uptime)
	log.Printf("   - 优化策略数: %d", stats.StrategiesOptimized)
	log.Printf("   - 引入策略数: %d", stats.StrategiesIntroduced)
	log.Printf("   - 参数优化数: %d", stats.ParameterOptimizations)
	log.Printf("   - 完成回测数: %d", stats.BacktestsCompleted)
	log.Printf("   - 通过验证数: %d", stats.ValidationsPassed)
	log.Printf("   - 分析运行数: %d", stats.AnalysisRuns)
	log.Printf("   - 检测模式数: %d", stats.PatternsDetected)
	log.Printf("   - 检测异常数: %d", stats.AnomaliesDetected)
	log.Printf("   - 平均优化时间: %v", stats.AverageOptimizationTime)
	log.Printf("   - 平均回测时间: %v", stats.AverageBacktestTime)
	log.Printf("   - 系统资源使用: %.2f%%", stats.SystemResourceUsage)

	// 演示配置更新
	log.Println("\n🔄 演示配置动态更新:")
	originalInterval := config.OptimizationInterval
	config.OptimizationInterval = 10 * time.Second
	log.Printf("✅ 优化间隔从 %v 更新为 %v", originalInterval, config.OptimizationInterval)

	originalMaxTasks := config.ResourceManagement.MaxConcurrentTasks
	config.ResourceManagement.MaxConcurrentTasks = 8
	log.Printf("✅ 最大并发任务从 %d 更新为 %d", originalMaxTasks, config.ResourceManagement.MaxConcurrentTasks)

	// 演示资源监控
	log.Println("\n🔍 资源监控演示:")
	for i := 0; i < 3; i++ {
		currentUsage := taskManager.GetResourceUsage()
		log.Printf("📊 第%d次检查 - 系统资源使用率: %.2f%%", i+1, currentUsage)

		if currentUsage > config.ResourceManagement.MaxCPUUsage {
			log.Printf("⚠️ 资源使用率超过阈值 %.1f%%，建议暂停低优先级任务", config.ResourceManagement.MaxCPUUsage)
		} else {
			log.Printf("✅ 资源使用率正常，可以继续执行任务")
		}

		time.Sleep(time.Second)
	}

	log.Println("\n🎉 持续运行优化系统演示完成!")
	log.Println("========================================")
	log.Println("💡 主要特性:")
	log.Println("   ✅ 持续策略优化和参数调整")
	log.Println("   ✅ 自动回测验证和性能评估")
	log.Println("   ✅ 实时市场分析和模式识别")
	log.Println("   ✅ 智能资源管理和任务调度")
	log.Println("   ✅ 性能跟踪和自动报告")
	log.Println("   ✅ 配置动态更新和系统监控")
	log.Println("")
	log.Println("🚀 这个系统确保量化交易策略在非交易时间也能持续优化和改进！")
}
