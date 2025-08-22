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
	fmt.Println("=== QCAT 历史数据回填示例 ===\n")

	// 1. 加载配置
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Printf("Warning: Failed to load config, using defaults: %v", err)
		// 使用默认配置
		cfg = &config.Config{
			Database: config.DatabaseConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "postgres",
				Password: "password",
				DBName:   "qcat",
				SSLMode:  "disable",
			},
			Exchange: config.ExchangeConfig{
				TestNet: true, // 使用测试网络
			},
		}
	}

	// 2. 连接数据库
	fmt.Println("连接数据库...")
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
		log.Printf("数据库连接失败，将使用模拟数据: %v", err)
		demonstrateWithMockData()
		return
	}
	defer db.Close()
	fmt.Println("✓ 数据库连接成功")

	// 3. 创建Binance客户端
	fmt.Println("创建Binance客户端...")
	binanceClient := binance.NewClient(&exchange.ExchangeConfig{
		APIKey:    cfg.Exchange.APIKey,
		APISecret: cfg.Exchange.APISecret,
		TestNet:   cfg.Exchange.TestNet,
	}, nil)
	fmt.Println("✓ Binance客户端创建成功")

	// 4. 创建K线管理器
	klineManager := kline.NewManagerWithBinance(db.DB, binanceClient)
	fmt.Println("✓ K线管理器创建成功")

	ctx := context.Background()
	symbol := "BTCUSDT"
	interval := kline.Interval1h
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -7) // 最近7天

	fmt.Printf("\n=== 数据完整性检查 ===\n")
	fmt.Printf("交易对: %s\n", symbol)
	fmt.Printf("间隔: %s\n", interval)
	fmt.Printf("时间范围: %s 到 %s\n", 
		startTime.Format("2006-01-02 15:04"), 
		endTime.Format("2006-01-02 15:04"))

	// 5. 检查数据完整性
	report, err := klineManager.CheckDataIntegrity(ctx, symbol, interval, startTime, endTime)
	if err != nil {
		log.Printf("数据完整性检查失败: %v", err)
	} else {
		fmt.Printf("\n数据完整性报告:\n")
		fmt.Printf("  期望数据点: %d\n", report.ExpectedCount)
		fmt.Printf("  实际数据点: %d\n", report.ActualCount)
		fmt.Printf("  完整度: %.2f%%\n", report.Completeness)
		fmt.Printf("  数据间隙: %d个\n", len(report.Gaps))

		if report.HasGaps && len(report.Gaps) > 0 {
			fmt.Printf("  前3个间隙:\n")
			for i, gap := range report.Gaps {
				if i >= 3 {
					break
				}
				fmt.Printf("    %s 到 %s\n", 
					gap.Start.Format("2006-01-02 15:04"), 
					gap.End.Format("2006-01-02 15:04"))
			}
		}
	}

	// 6. 演示智能历史数据获取
	fmt.Printf("\n=== 智能历史数据获取 ===\n")
	fmt.Println("获取历史数据（如果数据库中不完整，将自动从API回填）...")
	
	klines, err := klineManager.GetHistoryWithBackfill(ctx, symbol, interval, startTime, endTime)
	if err != nil {
		log.Printf("获取历史数据失败: %v", err)
	} else {
		fmt.Printf("✓ 成功获取 %d 条K线数据\n", len(klines))
		
		if len(klines) > 0 {
			first := klines[0]
			last := klines[len(klines)-1]
			fmt.Printf("  第一条: %s, 价格: %.2f\n", 
				first.OpenTime.Format("2006-01-02 15:04"), first.Close)
			fmt.Printf("  最后一条: %s, 价格: %.2f\n", 
				last.OpenTime.Format("2006-01-02 15:04"), last.Close)
		}
	}

	// 7. 演示MarketAnalyzer集成
	fmt.Printf("\n=== MarketAnalyzer 集成演示 ===\n")
	analyzer := generator.NewMarketAnalyzer(db, binanceClient, klineManager)
	
	fmt.Println("执行市场分析（使用真实历史数据）...")
	analysis, err := analyzer.AnalyzeMarket(ctx, symbol, 7*24*time.Hour)
	if err != nil {
		log.Printf("市场分析失败: %v", err)
	} else {
		fmt.Printf("✓ 市场分析完成\n")
		fmt.Printf("  波动率: %.4f (%.2f%%)\n", analysis.Volatility, analysis.Volatility*100)
		fmt.Printf("  趋势强度: %.4f\n", analysis.Trend)
		fmt.Printf("  夏普比率: %.4f\n", analysis.SharpeRatio)
		fmt.Printf("  最大回撤: %.4f (%.2f%%)\n", analysis.MaxDrawdown, analysis.MaxDrawdown*100)
		fmt.Printf("  市场状态: %s\n", analysis.MarketRegime)
		fmt.Printf("  分析置信度: %.4f (%.1f%%)\n", analysis.Confidence, analysis.Confidence*100)
	}

	// 8. 演示手动回填
	fmt.Printf("\n=== 手动历史数据回填演示 ===\n")
	fmt.Println("演示回填更长时间范围的数据...")
	
	longStartTime := endTime.AddDate(0, 0, -30) // 30天前
	fmt.Printf("回填时间范围: %s 到 %s\n", 
		longStartTime.Format("2006-01-02"), 
		endTime.Format("2006-01-02"))
	
	err = klineManager.BackfillHistoricalData(ctx, symbol, interval, longStartTime, endTime)
	if err != nil {
		log.Printf("手动回填失败: %v", err)
	} else {
		fmt.Println("✓ 手动回填完成")
		
		// 再次检查数据完整性
		finalReport, err := klineManager.CheckDataIntegrity(ctx, symbol, interval, longStartTime, endTime)
		if err != nil {
			log.Printf("最终数据完整性检查失败: %v", err)
		} else {
			fmt.Printf("最终数据完整性:\n")
			fmt.Printf("  数据点: %d/%d\n", finalReport.ActualCount, finalReport.ExpectedCount)
			fmt.Printf("  完整度: %.2f%%\n", finalReport.Completeness)
		}
	}

	fmt.Printf("\n=== 演示完成 ===\n")
	fmt.Println("现在您可以:")
	fmt.Println("1. 使用命令行工具进行批量回填: go run cmd/backfill/main.go")
	fmt.Println("2. 在您的代码中使用 GetHistoryWithBackfill() 智能获取历史数据")
	fmt.Println("3. 使用 CheckDataIntegrity() 定期检查数据质量")
}

// demonstrateWithMockData 使用模拟数据演示功能
func demonstrateWithMockData() {
	fmt.Printf("\n=== 使用模拟数据演示 ===\n")
	
	// 创建不带依赖项的分析器（将使用模拟数据）
	analyzer := &generator.MarketAnalyzer{}
	
	ctx := context.Background()
	symbol := "BTCUSDT"
	timeRange := 7 * 24 * time.Hour
	
	fmt.Printf("分析交易对: %s\n", symbol)
	fmt.Printf("时间范围: %v\n", timeRange)
	
	analysis, err := analyzer.AnalyzeMarket(ctx, symbol, timeRange)
	if err != nil {
		log.Printf("市场分析失败: %v", err)
		return
	}
	
	fmt.Printf("✓ 市场分析完成（使用模拟数据）\n")
	fmt.Printf("  波动率: %.4f (%.2f%%)\n", analysis.Volatility, analysis.Volatility*100)
	fmt.Printf("  趋势强度: %.4f\n", analysis.Trend)
	fmt.Printf("  市场状态: %s\n", analysis.MarketRegime)
	fmt.Printf("  分析置信度: %.4f (%.1f%%)\n", analysis.Confidence, analysis.Confidence*100)
	
	fmt.Println("\n注意: 这是使用模拟数据的演示。")
	fmt.Println("要使用真实数据，请配置数据库和Binance API。")
}
