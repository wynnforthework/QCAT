package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// 模拟优化结果
type OptimizationResult struct {
	ServerID      string
	Attempt       int
	ProfitRate    float64
	SharpeRatio   float64
	MaxDrawdown   float64
	WinRate       float64
	RandomSeed    int64
	IsGlobalBest  bool
	DiscoveredAt  time.Time
}

// 模拟分布式优化器
type SimpleDistributedOptimizer struct {
	globalBest    *OptimizationResult
	mu            sync.RWMutex
	results       []*OptimizationResult
}

func NewSimpleDistributedOptimizer() *SimpleDistributedOptimizer {
	return &SimpleDistributedOptimizer{
		results: make([]*OptimizationResult, 0),
	}
}

// 开始优化
func (opt *SimpleDistributedOptimizer) StartOptimization(serverID string, attempt int) *OptimizationResult {
	// 使用随机种子
	randomSeed := time.Now().UnixNano() + int64(len(serverID)*1000) + int64(attempt*100)
	rand.Seed(randomSeed)
	
	// 模拟训练时间
	time.Sleep(time.Duration(rand.Intn(200)+100) * time.Millisecond)
	
	// 生成随机性能结果
	result := &OptimizationResult{
		ServerID:     serverID,
		Attempt:      attempt,
		ProfitRate:   5.0 + rand.Float64()*15.0, // 5-20% 收益率
		SharpeRatio:  0.5 + rand.Float64()*2.5,  // 0.5-3.0 夏普比率
		MaxDrawdown:  rand.Float64() * 8.0,      // 0-8% 最大回撤
		WinRate:      0.3 + rand.Float64()*0.5,  // 30-80% 胜率
		RandomSeed:   randomSeed,
		DiscoveredAt: time.Now(),
	}
	
	// 检查是否为新的全局最优
	opt.mu.Lock()
	if opt.globalBest == nil || result.ProfitRate > opt.globalBest.ProfitRate {
		result.IsGlobalBest = true
		opt.globalBest = result
		fmt.Printf("[%s] 🎉 发现新的全局最优结果! 收益率: %.2f%%\n", 
			serverID, result.ProfitRate)
	}
	opt.results = append(opt.results, result)
	opt.mu.Unlock()
	
	return result
}

// 获取全局最优结果
func (opt *SimpleDistributedOptimizer) GetGlobalBest() *OptimizationResult {
	opt.mu.RLock()
	defer opt.mu.RUnlock()
	return opt.globalBest
}

// 获取所有结果
func (opt *SimpleDistributedOptimizer) GetAllResults() []*OptimizationResult {
	opt.mu.RLock()
	defer opt.mu.RUnlock()
	
	results := make([]*OptimizationResult, len(opt.results))
	copy(results, opt.results)
	return results
}

// 模拟单台服务器的优化过程
func simulateServerOptimization(serverID string, optimizer *SimpleDistributedOptimizer, results chan<- *OptimizationResult) {
	fmt.Printf("\n[%s] 开始优化...\n", serverID)
	
	// 进行多次随机探索
	for attempt := 1; attempt <= 5; attempt++ {
		fmt.Printf("[%s] 第 %d 次优化尝试...\n", serverID, attempt)
		
		// 执行优化
		result := optimizer.StartOptimization(serverID, attempt)
		
		fmt.Printf("[%s] 第 %d 次尝试结果: 收益率=%.2f%%, 夏普比率=%.2f\n", 
			serverID, attempt, result.ProfitRate, result.SharpeRatio)
		
		// 发送结果
		results <- result
	}
	
	fmt.Printf("[%s] 优化完成\n", serverID)
}

// 分析所有结果
func analyzeResults(results []*OptimizationResult) {
	fmt.Println("\n=== 结果分析 ===")
	fmt.Printf("总共收集到 %d 个优化结果\n", len(results))
	
	if len(results) == 0 {
		return
	}
	
	// 找到最优和最差结果
	var bestResult *OptimizationResult
	var worstResult *OptimizationResult
	var totalProfitRate float64
	var totalSharpeRatio float64
	
	for _, result := range results {
		totalProfitRate += result.ProfitRate
		totalSharpeRatio += result.SharpeRatio
		
		if bestResult == nil || result.ProfitRate > bestResult.ProfitRate {
			bestResult = result
		}
		if worstResult == nil || result.ProfitRate < worstResult.ProfitRate {
			worstResult = result
		}
	}
	
	// 统计信息
	avgProfitRate := totalProfitRate / float64(len(results))
	avgSharpeRatio := totalSharpeRatio / float64(len(results))
	
	fmt.Printf("\n📊 统计信息:\n")
	fmt.Printf("  平均收益率: %.2f%%\n", avgProfitRate)
	fmt.Printf("  平均夏普比率: %.2f\n", avgSharpeRatio)
	fmt.Printf("  最高收益率: %.2f%% (来自 %s)\n", bestResult.ProfitRate, bestResult.ServerID)
	fmt.Printf("  最低收益率: %.2f%% (来自 %s)\n", worstResult.ProfitRate, worstResult.ServerID)
	fmt.Printf("  收益率范围: %.2f%%\n", bestResult.ProfitRate-worstResult.ProfitRate)
	
	// 性能分布
	fmt.Printf("\n📈 性能分布:\n")
	profitRanges := map[string]int{
		"5-10%":   0,
		"10-15%":  0,
		"15-20%":  0,
		"20%+":    0,
	}
	
	for _, result := range results {
		profit := result.ProfitRate
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
	
	for profitRange, count := range profitRanges {
		fmt.Printf("  %s: %d 个结果\n", profitRange, count)
	}
	
	// 分布式优化效果分析
	fmt.Printf("\n🚀 分布式优化效果:\n")
	if bestResult.ProfitRate > avgProfitRate*1.2 {
		fmt.Printf("  ✅ 发现显著优于平均水平的优化结果\n")
		fmt.Printf("  📤 最优结果将自动传播到所有服务器\n")
		fmt.Printf("  🎯 所有服务器将采用 %.2f%% 的收益率\n", bestResult.ProfitRate)
		fmt.Printf("  📈 相比平均水平提升: %.2f%%\n", bestResult.ProfitRate-avgProfitRate)
	} else {
		fmt.Printf("  ⚠️  未发现显著优于平均水平的优化结果\n")
		fmt.Printf("  💡 建议增加探索次数或调整优化策略\n")
	}
	
	// 展示最优结果的详细信息
	if bestResult != nil {
		fmt.Printf("\n🏆 最优结果详情:\n")
		fmt.Printf("  发现者: %s\n", bestResult.ServerID)
		fmt.Printf("  尝试次数: %d\n", bestResult.Attempt)
		fmt.Printf("  收益率: %.2f%%\n", bestResult.ProfitRate)
		fmt.Printf("  夏普比率: %.2f\n", bestResult.SharpeRatio)
		fmt.Printf("  最大回撤: %.2f%%\n", bestResult.MaxDrawdown)
		fmt.Printf("  胜率: %.2f%%\n", bestResult.WinRate*100)
		fmt.Printf("  随机种子: %d\n", bestResult.RandomSeed)
		fmt.Printf("  发现时间: %s\n", bestResult.DiscoveredAt.Format("15:04:05"))
	}
	
	// 展示每台服务器的表现
	fmt.Printf("\n📋 各服务器表现:\n")
	serverStats := make(map[string][]*OptimizationResult)
	for _, result := range results {
		serverStats[result.ServerID] = append(serverStats[result.ServerID], result)
	}
	
	for serverID, serverResults := range serverStats {
		var serverTotalProfit float64
		var serverBestProfit float64
		for _, result := range serverResults {
			serverTotalProfit += result.ProfitRate
			if result.ProfitRate > serverBestProfit {
				serverBestProfit = result.ProfitRate
			}
		}
		serverAvgProfit := serverTotalProfit / float64(len(serverResults))
		fmt.Printf("  %s: 平均=%.2f%%, 最佳=%.2f%%, 尝试=%d次\n", 
			serverID, serverAvgProfit, serverBestProfit, len(serverResults))
	}
}

func main() {
	fmt.Println("=== 分布式优化演示 ===")
	fmt.Println("模拟多台服务器并行优化，寻找最优结果并共享")
	fmt.Println("时间:", time.Now().Format("2006-01-02 15:04:05"))
	
	// 创建分布式优化器
	optimizer := NewSimpleDistributedOptimizer()
	
	// 模拟3台服务器
	servers := []string{"Server-A", "Server-B", "Server-C"}
	var wg sync.WaitGroup
	results := make(chan *OptimizationResult, len(servers)*5) // 每台服务器5次尝试
	
	// 启动多台服务器并行优化
	for _, serverName := range servers {
		wg.Add(1)
		go func(serverID string) {
			defer wg.Done()
			simulateServerOptimization(serverID, optimizer, results)
		}(serverName)
	}
	
	// 等待所有服务器完成
	go func() {
		wg.Wait()
		close(results)
	}()
	
	// 收集所有结果
	var allResults []*OptimizationResult
	for result := range results {
		allResults = append(allResults, result)
	}
	
	// 分析结果
	analyzeResults(allResults)
	
	// 展示全局最优结果
	globalBest := optimizer.GetGlobalBest()
	if globalBest != nil {
		fmt.Printf("\n🎯 全局最优结果总结:\n")
		fmt.Printf("  服务器 %s 在第 %d 次尝试中发现最优结果\n", 
			globalBest.ServerID, globalBest.Attempt)
		fmt.Printf("  收益率: %.2f%%\n", globalBest.ProfitRate)
		fmt.Printf("  这个结果将自动传播到所有其他服务器\n")
		fmt.Printf("  所有服务器都将采用这个最优配置\n")
	}
	
	fmt.Println("\n=== 演示完成 ===")
}
