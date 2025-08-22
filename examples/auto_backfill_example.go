package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"qcat/internal/config"
	"qcat/internal/database"
	"qcat/internal/exchange"
	"qcat/internal/exchange/binance"
	"qcat/internal/market/kline"
	"qcat/internal/strategy/generator"
)

func main() {
	fmt.Println("=== QCAT 自动回填功能演示 ===\n")

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

	// 创建K线管理器
	klineManager := kline.NewManagerWithBinance(db.DB, binanceClient)

	ctx := context.Background()

	// 2. 演示自动回填配置
	fmt.Println("=== 自动回填配置 ===")
	
	// 自定义自动回填配置
	config := &kline.AutoBackfillConfig{
		Enabled:                true,
		MinCompletenessPercent: 85.0, // 85%完整度阈值
		MaxBackfillDays:        60,   // 最多回填60天
		RetryAttempts:          3,
		RetryDelay:             time.Second * 5,
	}
	klineManager.SetAutoBackfillConfig(config)
	
	fmt.Printf("✓ 自动回填已启用\n")
	fmt.Printf("  完整度阈值: %.1f%%\n", config.MinCompletenessPercent)
	fmt.Printf("  最大回填天数: %d天\n", config.MaxBackfillDays)
	fmt.Printf("  重试次数: %d\n", config.RetryAttempts)

	// 3. 演示智能数据获取
	fmt.Println("\n=== 智能数据获取演示 ===")
	
	symbol := "BTCUSDT"
	interval := kline.Interval1h
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -7) // 最近7天
	
	fmt.Printf("请求数据: %s %s (%s 到 %s)\n", 
		symbol, interval, 
		startTime.Format("2006-01-02"), 
		endTime.Format("2006-01-02"))
	
	// 使用智能获取（自动回填）
	klines, err := klineManager.GetHistoryWithBackfill(ctx, symbol, interval, startTime, endTime)
	if err != nil {
		log.Printf("获取数据失败: %v", err)
	} else {
		fmt.Printf("✓ 获取到 %d 条K线数据\n", len(klines))
	}

	// 4. 演示确保数据可用性
	fmt.Println("\n=== 确保数据可用性演示 ===")
	
	// 在执行任何需要历史数据的操作前，确保数据可用
	err = klineManager.EnsureDataAvailable(ctx, symbol, interval, startTime, endTime)
	if err != nil {
		log.Printf("确保数据可用失败: %v", err)
	} else {
		fmt.Println("✓ 数据可用性已确保")
	}

	// 5. 演示装饰器模式
	fmt.Println("\n=== 装饰器模式演示 ===")
	
	// 使用装饰器执行需要历史数据的操作
	err = klineManager.WithAutoBackfill(ctx, symbol, interval, startTime, endTime, 
		func(klines []*kline.Kline) error {
			if len(klines) == 0 {
				return fmt.Errorf("no data available")
			}
			
			// 计算一些简单的统计信息
			var totalVolume float64
			var minPrice, maxPrice float64 = klines[0].Close, klines[0].Close
			
			for _, k := range klines {
				totalVolume += k.Volume
				if k.Close < minPrice {
					minPrice = k.Close
				}
				if k.Close > maxPrice {
					maxPrice = k.Close
				}
			}
			
			fmt.Printf("✓ 数据分析完成:\n")
			fmt.Printf("  数据点数: %d\n", len(klines))
			fmt.Printf("  总成交量: %.0f\n", totalVolume)
			fmt.Printf("  价格范围: %.2f - %.2f\n", minPrice, maxPrice)
			
			return nil
		})
	
	if err != nil {
		log.Printf("装饰器操作失败: %v", err)
	}

	// 6. 演示自动回填服务
	fmt.Println("\n=== 自动回填服务演示 ===")
	
	// 创建自动回填服务
	autoService := kline.NewAutoBackfillService(klineManager)
	
	// 添加监控的交易对
	autoService.AddWatchedSymbol("BTCUSDT", kline.Interval1h, kline.Interval1d)
	autoService.AddWatchedSymbol("ETHUSDT", kline.Interval1h)
	autoService.AddWatchedSymbol("ADAUSDT", kline.Interval1h)
	
	// 设置检查间隔（演示用，设置为30秒）
	autoService.SetCheckInterval(30 * time.Second)
	
	fmt.Println("✓ 自动回填服务已配置")
	fmt.Printf("  监控交易对: BTCUSDT, ETHUSDT, ADAUSDT\n")
	fmt.Printf("  检查间隔: 30秒\n")
	
	// 启动服务（在后台运行）
	serviceCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	
	go func() {
		if err := autoService.Start(serviceCtx); err != nil {
			log.Printf("自动回填服务错误: %v", err)
		}
	}()
	
	// 等待一段时间让服务运行
	fmt.Println("等待自动回填服务运行...")
	time.Sleep(45 * time.Second)
	
	// 检查服务状态
	status := autoService.GetStatus()
	fmt.Printf("✓ 服务状态: %+v\n", status)
	
	// 获取回填历史
	history := autoService.GetBackfillHistory(5)
	fmt.Printf("✓ 最近 %d 条回填记录:\n", len(history))
	for i, record := range history {
		status := "成功"
		if !record.Success {
			status = fmt.Sprintf("失败: %s", record.Error)
		}
		fmt.Printf("  %d. %s %s - %s (%d条记录, 耗时%v)\n", 
			i+1, record.Symbol, record.Interval, status, 
			record.RecordCount, record.Duration)
	}
	
	// 停止服务
	autoService.Stop()

	// 7. 演示MarketAnalyzer集成
	fmt.Println("\n=== MarketAnalyzer 自动回填集成 ===")
	
	analyzer := generator.NewMarketAnalyzer(db, binanceClient, klineManager)
	
	fmt.Println("执行市场分析（自动回填缺失数据）...")
	analysis, err := analyzer.AnalyzeMarket(ctx, "BTCUSDT", 30*24*time.Hour) // 30天数据
	if err != nil {
		log.Printf("市场分析失败: %v", err)
	} else {
		fmt.Printf("✓ 市场分析完成（30天数据）\n")
		fmt.Printf("  波动率: %.4f\n", analysis.Volatility)
		fmt.Printf("  趋势强度: %.4f\n", analysis.Trend)
		fmt.Printf("  市场状态: %s\n", analysis.MarketRegime)
		fmt.Printf("  分析置信度: %.1f%%\n", analysis.Confidence*100)
	}

	fmt.Println("\n=== 演示完成 ===")
	fmt.Println("自动回填功能特性:")
	fmt.Println("✓ 智能检测数据缺失")
	fmt.Println("✓ 自动从API回填数据")
	fmt.Println("✓ 可配置的回填策略")
	fmt.Println("✓ 后台自动监控服务")
	fmt.Println("✓ 装饰器模式支持")
	fmt.Println("✓ 与MarketAnalyzer无缝集成")
}

func demonstrateWithoutDB() {
	fmt.Println("\n=== 无数据库模式演示 ===")
	fmt.Println("在没有数据库连接的情况下，系统会:")
	fmt.Println("1. 自动使用模拟数据")
	fmt.Println("2. 跳过自动回填功能")
	fmt.Println("3. 仍然提供完整的分析功能")
	
	analyzer := &generator.MarketAnalyzer{}
	ctx := context.Background()
	
	analysis, err := analyzer.AnalyzeMarket(ctx, "BTCUSDT", 7*24*time.Hour)
	if err != nil {
		log.Printf("分析失败: %v", err)
		return
	}
	
	fmt.Printf("✓ 模拟数据分析完成\n")
	fmt.Printf("  波动率: %.4f\n", analysis.Volatility)
	fmt.Printf("  趋势强度: %.4f\n", analysis.Trend)
	fmt.Printf("  市场状态: %s\n", analysis.MarketRegime)
}
