package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"qcat/internal/config"
	"qcat/internal/learning/automl"
)

// 模拟多个服务器节点的分布式优化
func main() {
	fmt.Println("=== 分布式优化演示 ===")
	fmt.Println("模拟多台服务器并行优化，寻找最优结果并共享")

	// 加载配置
	cfg, err := config.LoadConfig("configs/distributed_optimization.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 创建一致性管理器
	consistencyMgr, err := automl.NewConsistencyManager(cfg)
	if err != nil {
		log.Fatalf("Failed to create consistency manager: %v", err)
	}

	// 模拟3台服务器
	servers := []string{"Server-A", "Server-B", "Server-C"}
	var wg sync.WaitGroup
	results := make(chan *automl.OptimizationResult, len(servers))

	// 启动多台服务器并行优化
	for i, serverName := range servers {
		wg.Add(1)
		go func(serverID string, serverIndex int) {
			defer wg.Done()
			simulateServerOptimization(serverID, serverIndex, consistencyMgr, results)
		}(serverName, i)
	}

	// 等待所有服务器完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集所有结果
	var allResults []*automl.OptimizationResult
	for result := range results {
		allResults = append(allResults, result)
	}

	// 分析结果
	analyzeResults(allResults)
}

// 模拟单台服务器的优化过程
func simulateServerOptimization(serverID string, serverIndex int, consistencyMgr *automl.ConsistencyManager, results chan<- *automl.OptimizationResult) {
	fmt.Printf("\n[%s] 开始优化...\n", serverID)

	// 创建分布式优化器
	cfg, _ := config.LoadConfig("configs/distributed_optimization.yaml")
	optimizer, err := automl.NewDistributedOptimizer(cfg, consistencyMgr)
	if err != nil {
		log.Printf("[%s] Failed to create optimizer: %v", serverID, err)
		return
	}

	// 模拟多次优化尝试
	taskID := fmt.Sprintf("strategy_optimization_%d", serverIndex)
	strategyName := "MomentumStrategy"
	dataHash := "market_data_2024_01"

	// 进行多次随机探索
	for attempt := 1; attempt <= 5; attempt++ {
		fmt.Printf("[%s] 第 %d 次优化尝试...\n", serverID, attempt)

		// 使用不同的随机种子
		randomSeed := time.Now().UnixNano() + int64(serverIndex*1000) + int64(attempt*100)
		rand.Seed(randomSeed)

		// 模拟优化过程
		result, err := optimizer.StartOptimization(
			context.Background(),
			fmt.Sprintf("%s_attempt_%d", taskID, attempt),
			strategyName,
			dataHash,
		)
		if err != nil {
			log.Printf("[%s] Optimization failed: %v", serverID, err)
			continue
		}

		// 模拟训练时间
		time.Sleep(time.Duration(rand.Intn(200)+100) * time.Millisecond)

		// 生成随机性能结果
		result.Performance.ProfitRate = 5.0 + rand.Float64()*15.0 // 5-20% 收益率
		result.Performance.SharpeRatio = 0.5 + rand.Float64()*2.5 // 0.5-3.0 夏普比率
		result.Performance.MaxDrawdown = rand.Float64() * 8.0     // 0-8% 最大回撤
		result.Performance.WinRate = 0.3 + rand.Float64()*0.5     // 30-80% 胜率

		fmt.Printf("[%s] 第 %d 次尝试结果: 收益率=%.2f%%, 夏普比率=%.2f\n", 
			serverID, attempt, result.Performance.ProfitRate, result.Performance.SharpeRatio)

		// 检查是否为新的全局最优
		if optimizer.IsNewGlobalBest(taskID, result) {
			fmt.Printf("[%s] 🎉 发现新的全局最优结果! 收益率: %.2f%%\n", 
				serverID, result.Performance.ProfitRate)
			
			// 广播给其他服务器
			go optimizer.BroadcastBestResult(result)
		}

		// 发送结果
		results <- result
	}

	fmt.Printf("[%s] 优化完成\n", serverID)
}

// 分析所有结果
func analyzeResults(results []*automl.OptimizationResult) {
	fmt.Println("\n=== 结果分析 ===")
	fmt.Printf("总共收集到 %d 个优化结果\n", len(results))

	if len(results) == 0 {
		return
	}

	// 找到最优结果
	var bestResult *automl.OptimizationResult
	var worstResult *automl.OptimizationResult
	var totalProfitRate float64
	var totalSharpeRatio float64

	for _, result := range results {
		totalProfitRate += result.Performance.ProfitRate
		totalSharpeRatio += result.Performance.SharpeRatio

		if bestResult == nil || result.Performance.ProfitRate > bestResult.Performance.ProfitRate {
			bestResult = result
		}
		if worstResult == nil || result.Performance.ProfitRate < worstResult.Performance.ProfitRate {
			worstResult = result
		}
	}

	// 统计信息
	avgProfitRate := totalProfitRate / float64(len(results))
	avgSharpeRatio := totalSharpeRatio / float64(len(results))

	fmt.Printf("\n📊 统计信息:\n")
	fmt.Printf("  平均收益率: %.2f%%\n", avgProfitRate)
	fmt.Printf("  平均夏普比率: %.2f\n", avgSharpeRatio)
	fmt.Printf("  最高收益率: %.2f%% (来自 %s)\n", bestResult.Performance.ProfitRate, bestResult.DiscoveredBy)
	fmt.Printf("  最低收益率: %.2f%% (来自 %s)\n", worstResult.Performance.ProfitRate, worstResult.DiscoveredBy)
	fmt.Printf("  收益率范围: %.2f%%\n", bestResult.Performance.ProfitRate-worstResult.Performance.ProfitRate)

	// 性能分布
	fmt.Printf("\n📈 性能分布:\n")
	profitRanges := map[string]int{
		"5-10%":   0,
		"10-15%":  0,
		"15-20%":  0,
		"20%+":    0,
	}

	for _, result := range results {
		profit := result.Performance.ProfitRate
		switch {
		case profit < 10:
			profitRanges["5-10%"]++
		case profit < 15:
			profitRanges["10-15%"]++
		case profit < 20:
			profitRanges["15-20%"]++
		default:
			profitRanges["20%+"]++
		}
	}

	for range, count := range profitRanges {
		fmt.Printf("  %s: %d 个结果\n", range, count)
	}

	// 分布式优化效果分析
	fmt.Printf("\n🚀 分布式优化效果:\n")
	if bestResult.Performance.ProfitRate > avgProfitRate*1.2 {
		fmt.Printf("  ✅ 发现显著优于平均水平的优化结果\n")
		fmt.Printf("  📤 最优结果将自动传播到所有服务器\n")
		fmt.Printf("  🎯 所有服务器将采用 %.2f%% 的收益率\n", bestResult.Performance.ProfitRate)
	} else {
		fmt.Printf("  ⚠️  未发现显著优于平均水平的优化结果\n")
		fmt.Printf("  💡 建议增加探索次数或调整优化策略\n")
	}

	// 展示最优结果的详细信息
	if bestResult != nil {
		fmt.Printf("\n🏆 最优结果详情:\n")
		fmt.Printf("  发现者: %s\n", bestResult.DiscoveredBy)
		fmt.Printf("  策略: %s\n", bestResult.StrategyName)
		fmt.Printf("  收益率: %.2f%%\n", bestResult.Performance.ProfitRate)
		fmt.Printf("  夏普比率: %.2f\n", bestResult.Performance.SharpeRatio)
		fmt.Printf("  最大回撤: %.2f%%\n", bestResult.Performance.MaxDrawdown)
		fmt.Printf("  胜率: %.2f%%\n", bestResult.Performance.WinRate*100)
		fmt.Printf("  随机种子: %d\n", bestResult.RandomSeed)
		fmt.Printf("  发现时间: %s\n", bestResult.DiscoveredAt.Format("2006-01-02 15:04:05"))
	}
}

// 扩展方法：检查是否为新的全局最优
func (do *automl.DistributedOptimizer) IsNewGlobalBest(taskID string, result *automl.OptimizationResult) bool {
	return do.isNewGlobalBest(taskID, result)
}

// 扩展方法：广播最优结果
func (do *automl.DistributedOptimizer) BroadcastBestResult(result *automl.OptimizationResult) {
	do.broadcastBestResult(result)
}
