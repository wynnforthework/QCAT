package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"qcat/internal/analysis/backtesting"
	"qcat/internal/config"
	"qcat/internal/database"
	"qcat/internal/exchange"
	"qcat/internal/exchange/binance"
	"qcat/internal/market/kline"
	"qcat/internal/strategy/backtest"
)

func main() {
	fmt.Println("=== 回测系统自动数据回填演示 ===\n")

	// 1. 设置基础组件
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "password",
			DBName:   "qcat",
			SSLMode:  "disable",
		},
		Exchange: config.ExchangeConfig{
			TestNet: true,
		},
	}

	// 连接数据库
	db, err := database.NewConnection(&database.Config{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		DBName:          cfg.Database.DBName,
		SSLMode:         cfg.Database.SSLMode,
		MaxOpen:         25,
		MaxIdle:         5,
		Timeout:         5 * time.Second,
		ConnMaxLifetime: 1 * time.Hour,
		ConnMaxIdleTime: 15 * time.Minute,
	})
	if err != nil {
		log.Printf("数据库连接失败，使用模拟演示: %v", err)
		demonstrateWithoutDB()
		return
	}
	defer db.Close()

	// 创建Binance客户端
	binanceClient := binance.NewClient(&exchange.ExchangeConfig{
		APIKey:    cfg.Exchange.APIKey,
		APISecret: cfg.Exchange.APISecret,
		TestNet:   cfg.Exchange.TestNet,
	}, nil)

	// 创建K线管理器（带自动回填功能）
	klineManager := kline.NewManagerWithBinance(db.DB, binanceClient)

	ctx := context.Background()

	// 2. 演示自动回测引擎的自动数据获取
	fmt.Println("=== 自动回测引擎演示 ===")
	
	// 创建带自动回填功能的回测引擎
	autoEngine, err := backtesting.NewAutoBacktestingEngineWithKline(cfg, klineManager)
	if err != nil {
		log.Fatalf("创建自动回测引擎失败: %v", err)
	}

	fmt.Println("✓ 自动回测引擎已创建（集成自动数据回填功能）")

	// 创建回测任务
	job := &backtesting.BacktestJob{
		ID:             "test_backtest_001",
		StrategyID:     "ma_crossover",
		StrategyName:   "移动平均交叉策略",
		StartDate:      time.Now().AddDate(0, 0, -30), // 30天前
		EndDate:        time.Now().AddDate(0, 0, -1),  // 昨天
		InitialCapital: 10000.0,
		Parameters: map[string]interface{}{
			"fast_period": 10,
			"slow_period": 20,
		},
		Symbols:   []string{"BTCUSDT"},
		Status:    "PENDING",
		CreatedAt: time.Now(),
	}

	fmt.Printf("创建回测任务:\n")
	fmt.Printf("  策略: %s\n", job.StrategyName)
	fmt.Printf("  交易对: %v\n", job.Symbols)
	fmt.Printf("  时间范围: %s 到 %s\n", 
		job.StartDate.Format("2006-01-02"), 
		job.EndDate.Format("2006-01-02"))
	fmt.Printf("  初始资金: $%.2f\n", job.InitialCapital)

	// 提交回测任务
	err = autoEngine.SubmitBacktest(job)
	if err != nil {
		log.Printf("提交回测任务失败: %v", err)
	} else {
		fmt.Println("✓ 回测任务已提交")
		fmt.Println("  系统会自动:")
		fmt.Println("  1. 检查BTCUSDT的历史数据完整性")
		fmt.Println("  2. 如果数据不完整，自动从Binance API回填")
		fmt.Println("  3. 使用完整的历史数据执行回测")
	}

	// 3. 演示策略回测系统的自动数据加载
	fmt.Println("\n=== 策略回测系统演示 ===")

	// 创建数据加载器
	dataLoader := backtest.NewDataLoader(
		klineManager,    // K线管理器（带自动回填）
		nil,            // 订单簿管理器
		nil,            // 交易管理器
		nil,            // 资金费率管理器
		nil,            // 指数价格管理器
	)

	symbol := "BTCUSDT"
	startTime := time.Now().AddDate(0, 0, -7) // 7天前
	endTime := time.Now().AddDate(0, 0, -1)   // 昨天

	fmt.Printf("加载历史数据:\n")
	fmt.Printf("  交易对: %s\n", symbol)
	fmt.Printf("  时间范围: %s 到 %s\n", 
		startTime.Format("2006-01-02"), 
		endTime.Format("2006-01-02"))

	// 加载历史数据（会自动回填缺失数据）
	historicalData, err := dataLoader.LoadData(ctx, symbol, startTime, endTime)
	if err != nil {
		log.Printf("加载历史数据失败: %v", err)
	} else {
		fmt.Printf("✓ 历史数据加载完成:\n")
		fmt.Printf("  K线数据: %d 条\n", len(historicalData.Klines))
		
		if len(historicalData.Klines) > 0 {
			first := historicalData.Klines[0]
			last := historicalData.Klines[len(historicalData.Klines)-1]
			fmt.Printf("  时间范围: %s 到 %s\n", 
				first.OpenTime.Format("2006-01-02 15:04"), 
				last.OpenTime.Format("2006-01-02 15:04"))
			fmt.Printf("  价格范围: %.2f - %.2f\n", first.Close, last.Close)
		}
	}

	// 4. 演示手动确保数据可用性
	fmt.Println("\n=== 手动确保数据可用性演示 ===")

	// 在执行回测前，手动确保数据可用
	fmt.Println("确保数据可用性...")
	err = klineManager.EnsureDataAvailable(ctx, symbol, kline.Interval1h, startTime, endTime)
	if err != nil {
		log.Printf("确保数据可用失败: %v", err)
	} else {
		fmt.Println("✓ 数据可用性已确保")
	}

	// 5. 演示数据完整性检查
	fmt.Println("\n=== 数据完整性检查 ===")

	report, err := klineManager.CheckDataIntegrity(ctx, symbol, kline.Interval1h, startTime, endTime)
	if err != nil {
		log.Printf("数据完整性检查失败: %v", err)
	} else {
		fmt.Printf("数据完整性报告:\n")
		fmt.Printf("  期望数据点: %d\n", report.ExpectedCount)
		fmt.Printf("  实际数据点: %d\n", report.ActualCount)
		fmt.Printf("  完整度: %.2f%%\n", report.Completeness)
		fmt.Printf("  数据间隙: %d个\n", len(report.Gaps))
		
		if report.HasGaps && len(report.Gaps) > 0 {
			fmt.Printf("  间隙详情:\n")
			for i, gap := range report.Gaps {
				if i >= 3 { // 只显示前3个
					break
				}
				fmt.Printf("    %s 到 %s\n", 
					gap.Start.Format("2006-01-02 15:04"), 
					gap.End.Format("2006-01-02 15:04"))
			}
		}
	}

	// 6. 演示回测配置
	fmt.Println("\n=== 回测配置演示 ===")

	// 配置自动回填参数
	autoBackfillConfig := &kline.AutoBackfillConfig{
		Enabled:                true,
		MinCompletenessPercent: 90.0, // 90%完整度阈值（回测需要更高的数据质量）
		MaxBackfillDays:        180,  // 最多回填180天（回测可能需要更长的历史数据）
		RetryAttempts:          5,    // 更多重试次数
		RetryDelay:             time.Second * 3,
	}
	klineManager.SetAutoBackfillConfig(autoBackfillConfig)

	fmt.Printf("回测专用配置:\n")
	fmt.Printf("  完整度阈值: %.1f%% (更高的数据质量要求)\n", autoBackfillConfig.MinCompletenessPercent)
	fmt.Printf("  最大回填天数: %d天 (支持长期回测)\n", autoBackfillConfig.MaxBackfillDays)
	fmt.Printf("  重试次数: %d (确保数据获取成功)\n", autoBackfillConfig.RetryAttempts)

	fmt.Println("\n=== 演示完成 ===")
	fmt.Println("回测系统自动数据回填功能:")
	fmt.Println("✓ 自动检测历史数据缺失")
	fmt.Println("✓ 从Binance API自动回填数据")
	fmt.Println("✓ 确保回测数据的完整性和准确性")
	fmt.Println("✓ 支持长期历史数据回测")
	fmt.Println("✓ 透明集成，无需修改现有回测代码")
	fmt.Println("✓ 可配置的数据质量要求")
}

func demonstrateWithoutDB() {
	fmt.Println("\n=== 无数据库模式演示 ===")
	fmt.Println("在没有数据库连接的情况下:")
	fmt.Println("1. 回测系统会使用模拟数据")
	fmt.Println("2. 自动回填功能被跳过")
	fmt.Println("3. 仍然可以进行策略逻辑验证")
	
	// 创建不带数据库的回测引擎
	cfg := &config.Config{}
	autoEngine, err := backtesting.NewAutoBacktestingEngine(cfg)
	if err != nil {
		log.Printf("创建回测引擎失败: %v", err)
		return
	}
	
	fmt.Printf("✓ 回测引擎已创建（模拟数据模式）\n")
	fmt.Println("注意: 这是使用模拟数据的演示。")
	fmt.Println("要使用真实数据和自动回填，请配置数据库和Binance API。")
}
