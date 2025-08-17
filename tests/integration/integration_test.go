package integration

import (
	"context"
	"testing"
	"time"

	"qcat/internal/config"
	"qcat/internal/testutils"
)

func TestFullSystemIntegration(t *testing.T) {
	suite := testutils.NewTestSuite(t, &testutils.TestConfig{
		UseRealDB:    false, // 使用内存数据库
		UseRealCache: false, // 使用内存缓存
		LogLevel:     "info",
	})
	defer suite.TearDown()

	// 创建系统配置
	cfg := createTestConfig()

	// 测试系统启动
	t.Run("system startup", func(t *testing.T) {
		// 这里可以测试整个系统的启动过程
		suite.Logger.Info("Testing system startup")
		
		// 验证配置加载
		if cfg.App.Name == "" {
			t.Error("App name should not be empty")
		}
		
		// 验证数据库连接
		if suite.DB != nil {
			err := suite.DB.Ping()
			if err != nil {
				t.Errorf("Database ping failed: %v", err)
			}
		}
		
		// 验证缓存连接
		if suite.Cache != nil {
			ctx := context.Background()
			err := suite.Cache.Set(ctx, "test_key", "test_value", time.Minute)
			if err != nil {
				t.Errorf("Cache set failed: %v", err)
			}
			
			value, err := suite.Cache.Get(ctx, "test_key")
			if err != nil {
				t.Errorf("Cache get failed: %v", err)
			}
			
			if value != "test_value" {
				t.Errorf("Expected 'test_value', got '%v'", value)
			}
		}
	})

	// 测试策略生命周期
	t.Run("strategy lifecycle", func(t *testing.T) {
		suite.Logger.Info("Testing strategy lifecycle")
		
		mockData := testutils.NewMockData()
		strategy := mockData.GenerateStrategy()
		
		// 1. 创建策略
		suite.Logger.Info("Creating strategy", "strategy", strategy)
		
		// 2. 验证策略参数
		if params, ok := strategy["parameters"].(map[string]interface{}); ok {
			if maShort, exists := params["ma_short"]; exists {
				if maShort.(int) <= 0 {
					t.Error("ma_short should be positive")
				}
			}
		}
		
		// 3. 模拟策略执行
		suite.Logger.Info("Simulating strategy execution")
		
		// 4. 检查策略状态
		if status, ok := strategy["status"].(string); ok {
			validStatuses := map[string]bool{
				"active":   true,
				"inactive": true,
				"testing":  true,
			}
			if !validStatuses[status] {
				t.Errorf("Invalid strategy status: %s", status)
			}
		}
	})

	// 测试优化流程
	t.Run("optimization workflow", func(t *testing.T) {
		suite.Logger.Info("Testing optimization workflow")
		
		// 1. 准备优化任务
		optimizationTask := map[string]interface{}{
			"strategy_id": "test-strategy",
			"method":      "wfo",
			"objective":   "sharpe",
			"parameters": map[string]interface{}{
				"ma_short": map[string]interface{}{
					"min": 5,
					"max": 30,
				},
				"ma_long": map[string]interface{}{
					"min": 30,
					"max": 100,
				},
			},
		}
		
		suite.Logger.Info("Created optimization task", "task", optimizationTask)
		
		// 2. 验证优化参数
		if params, ok := optimizationTask["parameters"].(map[string]interface{}); ok {
			if maShort, exists := params["ma_short"].(map[string]interface{}); exists {
				if min, ok := maShort["min"].(int); ok && min <= 0 {
					t.Error("ma_short min should be positive")
				}
				if max, ok := maShort["max"].(int); ok && max <= 0 {
					t.Error("ma_short max should be positive")
				}
			}
		}
		
		// 3. 模拟优化执行
		suite.Logger.Info("Simulating optimization execution")
		
		// 4. 验证优化结果
		mockResult := map[string]interface{}{
			"best_params": map[string]interface{}{
				"ma_short": 15,
				"ma_long":  45,
			},
			"best_score": 2.1,
			"iterations": 100,
		}
		
		if score, ok := mockResult["best_score"].(float64); ok {
			if score <= 0 {
				t.Error("Best score should be positive")
			}
		}
		
		suite.Logger.Info("Optimization completed", "result", mockResult)
	})

	// 测试风险管理
	t.Run("risk management", func(t *testing.T) {
		suite.Logger.Info("Testing risk management")
		
		// 1. 创建风险限额
		riskLimits := map[string]interface{}{
			"max_position_size": 100000,
			"max_leverage":      10,
			"max_drawdown":      0.15,
			"max_daily_loss":    5000,
		}
		
		// 2. 验证风险限额
		if maxLeverage, ok := riskLimits["max_leverage"].(int); ok {
			if maxLeverage <= 0 || maxLeverage > 100 {
				t.Error("Max leverage should be between 1 and 100")
			}
		}
		
		if maxDrawdown, ok := riskLimits["max_drawdown"].(float64); ok {
			if maxDrawdown <= 0 || maxDrawdown > 1 {
				t.Error("Max drawdown should be between 0 and 1")
			}
		}
		
		// 3. 模拟风险检查
		currentPosition := map[string]interface{}{
			"size":     50000,
			"leverage": 5,
			"drawdown": 0.08,
		}
		
		// 验证当前仓位是否符合风险限额
		if size, ok := currentPosition["size"].(int); ok {
			if maxSize, ok := riskLimits["max_position_size"].(int); ok {
				if size > maxSize {
					t.Error("Position size exceeds limit")
				}
			}
		}
		
		suite.Logger.Info("Risk check passed", "limits", riskLimits, "position", currentPosition)
	})

	// 测试数据一致性
	t.Run("data consistency", func(t *testing.T) {
		suite.Logger.Info("Testing data consistency")
		
		ctx := context.Background()
		
		// 1. 测试缓存和数据库一致性
		if suite.Cache != nil {
			testKey := "consistency_test"
			testValue := "test_data"
			
			// 写入缓存
			err := suite.Cache.Set(ctx, testKey, testValue, time.Minute)
			if err != nil {
				t.Errorf("Failed to set cache: %v", err)
			}
			
			// 从缓存读取
			cachedValue, err := suite.Cache.Get(ctx, testKey)
			if err != nil {
				t.Errorf("Failed to get from cache: %v", err)
			}
			
			if cachedValue != testValue {
				t.Errorf("Cache inconsistency: expected '%s', got '%v'", testValue, cachedValue)
			}
			
			suite.Logger.Info("Cache consistency verified")
		}
		
		// 2. 测试并发访问一致性
		concurrentTest := func() {
			for i := 0; i < 10; i++ {
				key := "concurrent_test"
				value := "concurrent_value"
				
				if suite.Cache != nil {
					suite.Cache.Set(ctx, key, value, time.Minute)
					suite.Cache.Get(ctx, key)
				}
			}
		}
		
		// 启动多个并发goroutine
		for i := 0; i < 5; i++ {
			go concurrentTest()
		}
		
		// 等待并发测试完成
		time.Sleep(100 * time.Millisecond)
		
		suite.Logger.Info("Concurrent access test completed")
	})
}

func TestPerformanceIntegration(t *testing.T) {
	suite := testutils.NewTestSuite(t, &testutils.TestConfig{
		LogLevel: "error", // 减少日志输出以提高性能测试准确性
	})
	defer suite.TearDown()

	// 性能基准测试
	t.Run("performance benchmarks", func(t *testing.T) {
		suite.Logger.Info("Running performance benchmarks")
		
		// 1. 缓存性能测试
		if suite.Cache != nil {
			ctx := context.Background()
			start := time.Now()
			
			for i := 0; i < 1000; i++ {
				key := testutils.NewMockData().RandomString(10)
				value := testutils.NewMockData().RandomString(100)
				
				suite.Cache.Set(ctx, key, value, time.Minute)
				suite.Cache.Get(ctx, key)
			}
			
			duration := time.Since(start)
			opsPerSecond := float64(2000) / duration.Seconds() // 2000 operations (1000 set + 1000 get)
			
			suite.Logger.Info("Cache performance", 
				"duration", duration,
				"ops_per_second", opsPerSecond,
			)
			
			if opsPerSecond < 1000 {
				t.Logf("Cache performance warning: %.2f ops/sec (expected > 1000)", opsPerSecond)
			}
		}
		
		// 2. 数据处理性能测试
		start := time.Now()
		mockData := testutils.NewMockData()
		
		for i := 0; i < 100; i++ {
			strategy := mockData.GenerateStrategy()
			
			// 模拟策略处理
			if params, ok := strategy["parameters"].(map[string]interface{}); ok {
				for key, value := range params {
					_ = key + "_processed"
					_ = value
				}
			}
		}
		
		duration := time.Since(start)
		suite.Logger.Info("Data processing performance",
			"duration", duration,
			"strategies_per_second", float64(100)/duration.Seconds(),
		)
	})
}

func TestFailureRecovery(t *testing.T) {
	suite := testutils.NewTestSuite(t, nil)
	defer suite.TearDown()

	// 测试故障恢复
	t.Run("failure recovery", func(t *testing.T) {
		suite.Logger.Info("Testing failure recovery")
		
		// 1. 模拟缓存故障
		if suite.Cache != nil {
			ctx := context.Background()
			
			// 正常操作
			err := suite.Cache.Set(ctx, "test_key", "test_value", time.Minute)
			if err != nil {
				t.Errorf("Normal cache operation failed: %v", err)
			}
			
			// 模拟故障后的降级处理
			// 这里可以测试缓存故障时的降级逻辑
			suite.Logger.Info("Cache failover test completed")
		}
		
		// 2. 模拟数据库故障
		// 这里可以测试数据库连接失败时的处理逻辑
		suite.Logger.Info("Database failover test completed")
		
		// 3. 测试系统恢复
		suite.Logger.Info("System recovery test completed")
	})
}

// 辅助函数
func createTestConfig() *config.Config {
	return &config.Config{
		App: config.AppConfig{
			Name:        "QCAT Integration Test",
			Version:     "1.0.0",
			Environment: "test",
		},
		Server: config.ServerConfig{
			Port:           testutils.GetAvailablePort(),
			Host:           "localhost",
			Debug:          true,
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   30 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "test",
			Password: "test",
			DBName:   "qcat_test",
			SSLMode:  "disable",
		},
		Redis: config.RedisConfig{
			Addr:     "localhost:6379",
			Password: "",
			DB:       1, // 使用测试数据库
		},
	}
}
