package optimizer

import (
	"context"
	"testing"
	"time"

	"qcat/internal/testutils"
)

func TestWalkForwardOptimization(t *testing.T) {
	suite := testutils.NewTestSuite(t, nil)
	defer suite.TearDown()

	// 创建模拟数据
	mockData := testutils.NewMockData()
	
	// 创建优化器
	optimizer := NewOptimizer(&OptimizerConfig{
		Method:         "wfo",
		Objective:      "sharpe",
		MaxIterations:  100,
		TrainRatio:     0.7,
		ValidationRatio: 0.2,
		TestRatio:      0.1,
	})

	// 准备测试数据
	strategy := &Strategy{
		ID:   "test_strategy",
		Name: "Test Strategy",
		Parameters: map[string]Parameter{
			"ma_short": {
				Name: "ma_short",
				Type: "int",
				Min:  5,
				Max:  30,
				Current: 20,
			},
			"ma_long": {
				Name: "ma_long",
				Type: "int",
				Min:  30,
				Max:  100,
				Current: 50,
			},
		},
	}

	// 生成模拟历史数据
	historicalData := generateMockHistoricalData(1000)

	// 运行优化
	ctx := context.Background()
	result, err := optimizer.Optimize(ctx, strategy, historicalData)
	if err != nil {
		t.Fatalf("Optimization failed: %v", err)
	}

	// 验证结果
	if result == nil {
		t.Fatal("Optimization result is nil")
	}

	if result.BestParams == nil {
		t.Fatal("Best parameters is nil")
	}

	if result.BestScore <= 0 {
		t.Error("Best score should be positive")
	}

	// 验证参数在有效范围内
	if maShort, ok := result.BestParams["ma_short"]; ok {
		if maShort < 5 || maShort > 30 {
			t.Errorf("ma_short out of range: %v", maShort)
		}
	}

	if maLong, ok := result.BestParams["ma_long"]; ok {
		if maLong < 30 || maLong > 100 {
			t.Errorf("ma_long out of range: %v", maLong)
		}
	}

	suite.Logger.Info("Optimization completed",
		"best_score", result.BestScore,
		"iterations", result.Iterations,
		"best_params", result.BestParams,
	)
}

func TestOverfittingDetection(t *testing.T) {
	suite := testutils.NewTestSuite(t, nil)
	defer suite.TearDown()

	// 创建过拟合检测器
	detector := NewOverfittingDetector(&OverfittingConfig{
		MinTrades:           50,
		MaxParameterCount:   10,
		MinSharpeRatio:      1.0,
		MaxDrawdownRatio:    0.2,
		PBOThreshold:        0.5,
		DeflatedSharpeThreshold: 1.5,
	})

	// 创建测试性能数据
	performance := &PerformanceStats{
		TotalReturn:    0.25,
		AnnualReturn:   0.15,
		MaxDrawdown:    0.08,
		SharpeRatio:    2.1,
		WinRate:        0.65,
		TradeCount:     150,
		ProfitFactor:   1.8,
		AvgTradeReturn: 0.002,
		Returns:        generateMockReturns(150),
	}

	// 测试过拟合检测
	isOverfitted, metrics := detector.DetectOverfitting(performance, 5)
	
	suite.Logger.Info("Overfitting detection result",
		"is_overfitted", isOverfitted,
		"metrics", metrics,
	)

	// 验证检测结果
	if metrics == nil {
		t.Fatal("Overfitting metrics is nil")
	}

	if metrics.DeflatedSharpe <= 0 {
		t.Error("Deflated Sharpe should be positive")
	}

	if metrics.PBO < 0 || metrics.PBO > 1 {
		t.Error("PBO should be between 0 and 1")
	}
}

func TestParameterSensitivityAnalysis(t *testing.T) {
	suite := testutils.NewTestSuite(t, nil)
	defer suite.TearDown()

	// 创建敏感度分析器
	analyzer := NewSensitivityAnalyzer(&SensitivityConfig{
		PerturbationRatio: 0.1,
		SampleSize:        50,
		ConfidenceLevel:   0.95,
	})

	// 准备测试数据
	baseParams := map[string]float64{
		"ma_short":    20,
		"ma_long":     50,
		"stop_loss":   0.05,
		"take_profit": 0.1,
	}

	// 模拟性能评估函数
	evaluateFunc := func(params map[string]float64) float64 {
		// 简单的模拟函数，实际应该是策略回测
		maShort := params["ma_short"]
		maLong := params["ma_long"]
		
		if maShort >= maLong {
			return -1.0 // 无效参数
		}
		
		// 模拟夏普比率计算
		ratio := maLong / maShort
		if ratio > 2 && ratio < 4 {
			return 2.0 + (ratio-2)*0.5 // 最优范围
		}
		return 1.0 + ratio*0.1
	}

	// 运行敏感度分析
	result := analyzer.AnalyzeSensitivity(baseParams, evaluateFunc)

	// 验证结果
	if result == nil {
		t.Fatal("Sensitivity analysis result is nil")
	}

	if len(result.Sensitivities) == 0 {
		t.Fatal("No sensitivity data generated")
	}

	// 验证每个参数都有敏感度数据
	for param := range baseParams {
		if _, exists := result.Sensitivities[param]; !exists {
			t.Errorf("Missing sensitivity data for parameter: %s", param)
		}
	}

	suite.Logger.Info("Sensitivity analysis completed",
		"sensitivities", result.Sensitivities,
		"rankings", result.Rankings,
	)
}

func BenchmarkOptimization(b *testing.B) {
	config := &testutils.TestConfig{
		LogLevel: testutils.LogLevel("error"),
	}

	testutils.RunBenchmark(b, "WalkForwardOptimization", config, func(b *testing.B, suite *testutils.BenchmarkSuite) {
		optimizer := NewOptimizer(&OptimizerConfig{
			Method:        "wfo",
			Objective:     "sharpe",
			MaxIterations: 50, // 减少迭代次数以加快基准测试
		})

		strategy := &Strategy{
			ID:   "bench_strategy",
			Name: "Benchmark Strategy",
			Parameters: map[string]Parameter{
				"ma_short": {Type: "int", Min: 5, Max: 30, Current: 20},
				"ma_long":  {Type: "int", Min: 30, Max: 100, Current: 50},
			},
		}

		historicalData := generateMockHistoricalData(500)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx := context.Background()
			_, err := optimizer.Optimize(ctx, strategy, historicalData)
			if err != nil {
				b.Fatalf("Optimization failed: %v", err)
			}
		}
	})
}

// 辅助函数

func generateMockHistoricalData(count int) []MarketData {
	mockData := testutils.NewMockData()
	data := make([]MarketData, count)
	
	basePrice := 45000.0
	for i := 0; i < count; i++ {
		// 模拟价格随机游走
		change := mockData.RandomFloat(-0.05, 0.05)
		basePrice *= (1 + change)
		
		data[i] = MarketData{
			Timestamp: time.Now().Add(-time.Duration(count-i) * time.Minute),
			Open:      basePrice * (1 + mockData.RandomFloat(-0.01, 0.01)),
			High:      basePrice * (1 + mockData.RandomFloat(0, 0.02)),
			Low:       basePrice * (1 + mockData.RandomFloat(-0.02, 0)),
			Close:     basePrice,
			Volume:    mockData.RandomFloat(100, 1000),
		}
	}
	
	return data
}

func generateMockReturns(count int) []float64 {
	mockData := testutils.NewMockData()
	returns := make([]float64, count)
	
	for i := 0; i < count; i++ {
		// 生成正态分布的收益率
		returns[i] = mockData.RandomFloat(-0.05, 0.05)
	}
	
	return returns
}
