package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"qcat/internal/learning/automl"
)

// 演示结果共享功能
func main() {
	fmt.Println("=== 结果共享系统演示 ===")
	fmt.Println("演示随机种子训练和多种共享方式")
	fmt.Println("时间:", time.Now().Format("2006-01-02 15:04:05"))

	// 创建结果共享配置
	config := &automl.ResultSharingConfig{
		Enabled: true,
		Mode:    "hybrid", // 使用混合模式
		PerformanceThreshold: struct {
			MinProfitRate  float64 `json:"min_profit_rate" yaml:"min_profit_rate"`
			MinSharpeRatio float64 `json:"min_sharpe_ratio" yaml:"min_sharpe_ratio"`
			MaxDrawdown    float64 `json:"max_drawdown" yaml:"max_drawdown"`
		}{
			MinProfitRate:  5.0,
			MinSharpeRatio: 0.5,
			MaxDrawdown:    15.0,
		},
	}

	// 创建结果共享管理器
	resultSharingMgr, err := automl.NewResultSharingManager(config)
	if err != nil {
		log.Fatalf("Failed to create result sharing manager: %v", err)
	}

	// 模拟多台服务器
	servers := []string{"Server-A", "Server-B", "Server-C", "Server-D"}
	var wg sync.WaitGroup
	results := make(chan *automl.SharedResult, len(servers)*3)

	// 启动多台服务器并行训练
	for i, serverName := range servers {
		wg.Add(1)
		go func(serverID string, serverIndex int) {
			defer wg.Done()
			simulateServerTraining(serverID, serverIndex, resultSharingMgr, results)
		}(serverName, i)
	}

	// 等待所有服务器完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集所有结果
	var allResults []*automl.SharedResult
	for result := range results {
		allResults = append(allResults, result)
	}

	// 分析结果
	analyzeSharedResults(allResults, resultSharingMgr)

	// 演示结果共享功能
	demonstrateSharingFeatures(resultSharingMgr)
}

// 模拟单台服务器的训练过程
func simulateServerTraining(serverID string, serverIndex int, 
	resultSharingMgr *automl.ResultSharingManager, results chan<- *automl.SharedResult) {
	
	fmt.Printf("[%s] 开始训练...\n", serverID)

	// 模拟多个训练任务
	tasks := []string{"strategy_ma_cross", "strategy_rsi", "strategy_bollinger"}
	
	for taskIndex, taskName := range tasks {
		// 使用随机种子确保每台服务器的训练都不重复
		randomSeed := time.Now().UnixNano() + int64(len(serverID)*1000) + int64(serverIndex*100) + int64(taskIndex*10)
		rand.Seed(randomSeed)
		
		fmt.Printf("[%s] 训练任务 %s，使用随机种子: %d\n", serverID, taskName, randomSeed)
		
		// 模拟训练时间
		trainingTime := time.Duration(rand.Intn(200)+100) * time.Millisecond
		time.Sleep(trainingTime)
		
		// 生成随机性能结果
		performance := &automl.PerformanceMetrics{
			ProfitRate:  5.0 + rand.Float64()*20.0, // 5-25% 收益率
			SharpeRatio: 0.5 + rand.Float64()*2.5,  // 0.5-3.0 夏普比率
			MaxDrawdown: rand.Float64() * 10.0,     // 0-10% 最大回撤
			WinRate:     0.3 + rand.Float64()*0.5,  // 30-80% 胜率
		}
		
		// 生成随机参数
		parameters := map[string]interface{}{
			"ma_short": 5 + rand.Intn(20),
			"ma_long":  20 + rand.Intn(40),
			"rsi_period": 14 + rand.Intn(10),
			"bollinger_period": 20 + rand.Intn(10),
		}
		
		// 创建共享结果
		sharedResult := &automl.SharedResult{
			ID:           fmt.Sprintf("%s_%s_%d", serverID, taskName, time.Now().Unix()),
			TaskID:       fmt.Sprintf("task_%d", taskIndex),
			StrategyName: taskName,
			Parameters:   parameters,
			Performance:  performance,
			RandomSeed:   randomSeed,
			DataHash:     fmt.Sprintf("data_%s_%s", serverID, taskName),
			DiscoveredBy: serverID,
			DiscoveredAt: time.Now(),
			ShareMethod:  "training",
			AdoptionCount: 0,
			IsGlobalBest: false,
		}
		
		// 共享结果
		if err := resultSharingMgr.ShareResult(sharedResult); err != nil {
			fmt.Printf("[%s] 共享结果失败: %v\n", serverID, err)
		} else {
			fmt.Printf("[%s] 成功共享结果，收益率: %.2f%%，夏普比率: %.2f\n", 
				serverID, performance.ProfitRate, performance.SharpeRatio)
		}
		
		// 发送结果
		results <- sharedResult
	}
	
	fmt.Printf("[%s] 训练完成\n", serverID)
}

// 分析共享结果
func analyzeSharedResults(results []*automl.SharedResult, resultSharingMgr *automl.ResultSharingManager) {
	fmt.Println("\n=== 共享结果分析 ===")
	fmt.Printf("总共收集到 %d 个训练结果\n", len(results))

	if len(results) == 0 {
		return
	}

	// 按策略分组分析
	strategyGroups := make(map[string][]*automl.SharedResult)
	for _, result := range results {
		strategyGroups[result.StrategyName] = append(strategyGroups[result.StrategyName], result)
	}

	// 分析每个策略的结果
	for strategyName, strategyResults := range strategyGroups {
		fmt.Printf("\n📊 策略: %s (%d 个结果)\n", strategyName, len(strategyResults))
		
		// 找到最优结果
		var bestResult *automl.SharedResult
		var worstResult *automl.SharedResult
		var totalProfitRate float64
		var totalSharpeRatio float64
		
		for _, result := range strategyResults {
			totalProfitRate += result.Performance.ProfitRate
			totalSharpeRatio += result.Performance.SharpeRatio
			
			if bestResult == nil || result.Performance.ProfitRate > bestResult.Performance.ProfitRate {
				bestResult = result
			}
			if worstResult == nil || result.Performance.ProfitRate < worstResult.Performance.ProfitRate {
				worstResult = result
			}
		}
		
		avgProfitRate := totalProfitRate / float64(len(strategyResults))
		avgSharpeRatio := totalSharpeRatio / float64(len(strategyResults))
		
		fmt.Printf("  平均收益率: %.2f%%\n", avgProfitRate)
		fmt.Printf("  平均夏普比率: %.2f\n", avgSharpeRatio)
		fmt.Printf("  最高收益率: %.2f%% (来自 %s)\n", bestResult.Performance.ProfitRate, bestResult.DiscoveredBy)
		fmt.Printf("  最低收益率: %.2f%% (来自 %s)\n", worstResult.Performance.ProfitRate, worstResult.DiscoveredBy)
		fmt.Printf("  收益率范围: %.2f%%\n", bestResult.Performance.ProfitRate-worstResult.Performance.ProfitRate)
		
		// 检查是否通过结果共享管理器获取到最优结果
		if sharedBest := resultSharingMgr.GetBestSharedResult(bestResult.TaskID, bestResult.StrategyName); sharedBest != nil {
			fmt.Printf("  ✅ 通过结果共享获取到最优结果: %.2f%%\n", sharedBest.Performance.ProfitRate)
		}
	}
}

// 演示结果共享功能
func demonstrateSharingFeatures(resultSharingMgr *automl.ResultSharingManager) {
	fmt.Println("\n=== 结果共享功能演示 ===")
	
	// 1. 演示获取所有共享结果
	fmt.Println("\n1. 获取所有共享结果:")
	allResults := resultSharingMgr.GetAllSharedResults()
	fmt.Printf("   总共 %d 个共享结果\n", len(allResults))
	
	// 2. 演示获取特定任务的最优结果
	fmt.Println("\n2. 获取特定任务的最优结果:")
	for i := 0; i < 3; i++ {
		taskID := fmt.Sprintf("task_%d", i)
		bestResult := resultSharingMgr.GetBestSharedResult(taskID, "strategy_ma_cross")
		if bestResult != nil {
			fmt.Printf("   任务 %s 的最优结果: 收益率 %.2f%%，来自 %s\n", 
				taskID, bestResult.Performance.ProfitRate, bestResult.DiscoveredBy)
		} else {
			fmt.Printf("   任务 %s 没有找到共享结果\n", taskID)
		}
	}
	
	// 3. 演示手动共享结果
	fmt.Println("\n3. 手动共享结果:")
	manualResult := &automl.SharedResult{
		ID:           "manual_test_001",
		TaskID:       "manual_task",
		StrategyName: "manual_strategy",
		Parameters: map[string]interface{}{
			"param1": 100,
			"param2": 200,
		},
		Performance: &automl.PerformanceMetrics{
			ProfitRate:  18.5,
			SharpeRatio: 2.1,
			MaxDrawdown: 8.2,
			WinRate:     0.65,
		},
		RandomSeed:   time.Now().UnixNano(),
		DataHash:     "manual_data_hash",
		DiscoveredBy: "manual_test",
		DiscoveredAt: time.Now(),
		ShareMethod:  "manual",
		AdoptionCount: 0,
		IsGlobalBest: false,
	}
	
	if err := resultSharingMgr.ShareResult(manualResult); err != nil {
		fmt.Printf("   手动共享失败: %v\n", err)
	} else {
		fmt.Printf("   手动共享成功，收益率: %.2f%%\n", manualResult.Performance.ProfitRate)
	}
	
	// 4. 演示结果评分
	fmt.Println("\n4. 结果评分演示:")
	for i, result := range allResults[:3] { // 只显示前3个结果
		score := resultSharingMgr.calculateScore(result)
		fmt.Printf("   结果 %d: 收益率 %.2f%%，评分 %.2f\n", 
			i+1, result.Performance.ProfitRate, score)
	}
	
	// 5. 演示不同共享模式的效果
	fmt.Println("\n5. 共享模式效果:")
	fmt.Println("   - 文件共享: 结果保存为JSON文件，便于跨服务器传输")
	fmt.Println("   - 字符串共享: 结果编码为字符串，便于复制粘贴")
	fmt.Println("   - 种子共享: 通过随机种子映射，便于重现结果")
	fmt.Println("   - 混合模式: 同时使用多种方式，确保结果不丢失")
}

// 演示跨服务器结果共享场景
func demonstrateCrossServerSharing() {
	fmt.Println("\n=== 跨服务器结果共享场景演示 ===")
	
	// 模拟两台完全不相连的服务器
	fmt.Println("\n场景1: 两台完全不相连的服务器")
	fmt.Println("服务器A和服务器B通过网络隔离，无法直接通信")
	fmt.Println("解决方案:")
	fmt.Println("  1. 服务器A将结果保存到共享文件")
	fmt.Println("  2. 通过U盘、邮件等方式传输文件")
	fmt.Println("  3. 服务器B读取共享文件，获取最优结果")
	
	// 模拟通过配置文件共享
	fmt.Println("\n场景2: 通过配置文件共享")
	fmt.Println("服务器A生成配置文件，包含最优参数和性能指标")
	fmt.Println("服务器B读取配置文件，直接应用最优配置")
	
	// 模拟通过字符串共享
	fmt.Println("\n场景3: 通过字符串共享")
	fmt.Println("服务器A生成结果字符串，通过聊天工具发送")
	fmt.Println("服务器B解析字符串，恢复完整结果")
	
	// 模拟通过种子共享
	fmt.Println("\n场景4: 通过种子共享")
	fmt.Println("服务器A记录最优结果的随机种子")
	fmt.Println("服务器B使用相同种子，重现最优结果")
}

// 演示随机种子的重要性
func demonstrateRandomSeedImportance() {
	fmt.Println("\n=== 随机种子重要性演示 ===")
	
	fmt.Println("\n固定种子 vs 随机种子:")
	fmt.Println("固定种子 (seed=42):")
	fmt.Println("  - 所有服务器得到相同结果")
	fmt.Println("  - 无法探索不同的参数空间")
	fmt.Println("  - 可能错过更好的解")
	
	fmt.Println("\n随机种子:")
	fmt.Println("  - 每台服务器得到不同结果")
	fmt.Println("  - 可以探索更大的参数空间")
	fmt.Println("  - 有机会找到全局最优解")
	
	fmt.Println("\n实际效果:")
	fmt.Println("  固定种子: 所有服务器收益率都是 8.5%")
	fmt.Println("  随机种子: 服务器A=5.2%, 服务器B=8.5%, 服务器C=12.8%")
	fmt.Println("  结果: 通过结果共享，所有服务器都能获得 12.8% 的最优结果")
}
