package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"qcat/internal/strategy/workflow"
)

func main() {
	log.Println("启动多策略工作流系统演示程序...")

	// 创建系统
	system, err := workflow.NewMultiStrategyWorkflowSystem(nil)
	if err != nil {
		log.Fatalf("Failed to create multi-strategy workflow system: %v", err)
	}

	// 启动系统
	if err := system.Start(); err != nil {
		log.Fatalf("Failed to start system: %v", err)
	}

	// 设置优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动演示任务
	go runDemo(ctx, system)

	// 启动状态监控
	go monitorSystem(ctx, system)

	// 等待关闭信号
	<-sigChan
	log.Println("收到关闭信号，正在停止系统...")

	// 停止系统
	if err := system.Stop(); err != nil {
		log.Printf("Error stopping system: %v", err)
	}

	log.Println("系统已停止")
}

// runDemo 运行演示任务
func runDemo(ctx context.Context, system *workflow.MultiStrategyWorkflowSystem) {
	// 等待系统稳定
	time.Sleep(3 * time.Second)

	log.Println("开始创建演示策略...")

	// 创建不同类型的策略
	strategies := []struct {
		name string
		type_ string
	}{
		{"动量策略A", "momentum"},
		{"均值回归策略B", "mean_reversion"},
		{"趋势跟踪策略C", "trend_following"},
		{"套利策略D", "arbitrage"},
		{"机器学习策略E", "ml_based"},
	}

	createdStrategies := make([]string, 0)

	for i, strategy := range strategies {
		select {
		case <-ctx.Done():
			return
		default:
		}

		log.Printf("创建策略 %d: %s (%s)", i+1, strategy.name, strategy.type_)
		
		strategyID, err := system.CreateAndRunStrategy(strategy.name, strategy.type_)
		if err != nil {
			log.Printf("Failed to create strategy %s: %v", strategy.name, err)
			continue
		}

		createdStrategies = append(createdStrategies, strategyID)
		log.Printf("策略 %s 创建成功，ID: %s", strategy.name, strategyID)

		// 间隔创建，避免资源竞争
		time.Sleep(5 * time.Second)
	}

	log.Printf("演示策略创建完成，共创建 %d 个策略", len(createdStrategies))

	// 让策略运行一段时间
	log.Println("策略运行中，观察系统状态...")
	
	// 定期输出策略状态
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			printStrategyStatus(system)
		}
	}
}

// monitorSystem 监控系统状态
func monitorSystem(ctx context.Context, system *workflow.MultiStrategyWorkflowSystem) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			printSystemStats(system)
		}
	}
}

// printSystemStats 打印系统统计信息
func printSystemStats(system *workflow.MultiStrategyWorkflowSystem) {
	stats := system.GetSystemStats()
	
	fmt.Println("\n" + "="*60)
	fmt.Println("系统状态报告")
	fmt.Println("="*60)
	fmt.Printf("运行时间: %v\n", stats.Uptime)
	fmt.Printf("组件状态: %d/%d 运行中\n", stats.ComponentsRunning, stats.ComponentsTotal)
	fmt.Printf("策略统计:\n")
	fmt.Printf("  - 总策略数: %d\n", stats.TotalStrategies)
	fmt.Printf("  - 活跃策略: %d\n", stats.ActiveStrategies)
	fmt.Printf("  - 启用策略: %d\n", stats.EnabledStrategies)
	fmt.Printf("  - 禁用策略: %d\n", stats.DisabledStrategies)
	fmt.Printf("执行统计:\n")
	fmt.Printf("  - 总执行次数: %d\n", stats.TotalExecutions)
	fmt.Printf("  - 成功执行: %d\n", stats.SuccessfulExecutions)
	fmt.Printf("  - 失败执行: %d\n", stats.FailedExecutions)
	
	if stats.TotalExecutions > 0 {
		successRate := float64(stats.SuccessfulExecutions) / float64(stats.TotalExecutions) * 100
		fmt.Printf("  - 成功率: %.2f%%\n", successRate)
	}
	
	fmt.Printf("最后更新: %v\n", stats.LastUpdateTime.Format("2006-01-02 15:04:05"))
	fmt.Println("="*60)
}

// printStrategyStatus 打印策略状态
func printStrategyStatus(system *workflow.MultiStrategyWorkflowSystem) {
	fmt.Println("\n" + "-"*50)
	fmt.Println("策略状态概览")
	fmt.Println("-"*50)
	
	// TODO 添加更详细的策略状态信息
	// 由于我们的系统设计，需要通过多策略管理器获取详细信息
	
	stats := system.GetSystemStats()
	fmt.Printf("当前时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("活跃策略数: %d\n", stats.ActiveStrategies)
	fmt.Printf("启用策略数: %d\n", stats.EnabledStrategies)
	
	if stats.TotalExecutions > 0 {
		fmt.Printf("最近执行状态: %d 成功, %d 失败\n", 
			stats.SuccessfulExecutions, stats.FailedExecutions)
	}
	
	fmt.Println("-"*50)
}

// 演示配置
func createDemoConfig() *workflow.SystemConfig {
	config := workflow.GetDefaultSystemConfig()
	
	// 调整配置以适合演示
	config.MultiStrategyManager.MaxConcurrentStrategies = 8
	config.MultiStrategyManager.MaxConcurrentJobs = 20
	config.MultiStrategyManager.SchedulingInterval = 15 * time.Second
	config.MultiStrategyManager.EvaluationInterval = 2 * time.Minute
	
	config.StrategyWorkflow.MaxConcurrentJobs = 3
	config.StrategyWorkflow.OnboardingTimeout = 20 * time.Second
	config.StrategyWorkflow.BacktestTimeout = 30 * time.Second
	config.StrategyWorkflow.OptimizationTimeout = 45 * time.Second
	config.StrategyWorkflow.LearningTimeout = 60 * time.Second
	config.StrategyWorkflow.ApplicationTimeout = 15 * time.Second
	
	config.EvolutionManager.EvaluationInterval = 3 * time.Minute
	config.EvolutionManager.PopulationSize = 10
	
	config.TradingWorkflow.ExecutionInterval = 30 * time.Second
	config.TradingWorkflow.StrategyCheckInterval = 15 * time.Second
	
	config.Monitoring.MetricsInterval = 15 * time.Second
	config.Monitoring.HealthCheckInterval = 30 * time.Second
	
	return config
}

// 辅助函数：等待用户输入
func waitForUserInput() {
	fmt.Println("\n按 Enter 键继续...")
	fmt.Scanln()
}

// 演示特定功能
func demonstrateFeatures(ctx context.Context, system *workflow.MultiStrategyWorkflowSystem) {
	log.Println("演示系统特性...")
	
	// 1. 演示策略创建和生命周期
	log.Println("1. 创建策略并观察生命周期...")
	strategyID, err := system.CreateAndRunStrategy("演示策略", "demo")
	if err != nil {
		log.Printf("创建策略失败: %v", err)
		return
	}
	
	log.Printf("策略 %s 已创建，正在执行生命周期...", strategyID)
	time.Sleep(30 * time.Second)
	
	// 2. 演示并发策略执行
	log.Println("2. 创建多个并发策略...")
	for i := 0; i < 3; i++ {
		_, err := system.CreateAndRunStrategy(fmt.Sprintf("并发策略%d", i+1), "concurrent")
		if err != nil {
			log.Printf("创建并发策略 %d 失败: %v", i+1, err)
		} else {
			log.Printf("并发策略 %d 创建成功", i+1)
		}
		time.Sleep(5 * time.Second)
	}
	
	// 3. 观察系统状态
	log.Println("3. 观察系统运行状态...")
	for i := 0; i < 5; i++ {
		printSystemStats(system)
		time.Sleep(30 * time.Second)
	}
	
	log.Println("特性演示完成")
}
