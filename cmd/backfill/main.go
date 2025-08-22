package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"qcat/internal/config"
	"qcat/internal/database"
	"qcat/internal/exchange"
	"qcat/internal/exchange/binance"
	"qcat/internal/market/kline"
)

func main() {
	var (
		configPath = flag.String("config", "configs/config.yaml", "配置文件路径")
		symbol     = flag.String("symbol", "BTCUSDT", "交易对符号")
		interval   = flag.String("interval", "1h", "K线间隔")
		startDate  = flag.String("start", "", "开始日期 (YYYY-MM-DD)")
		endDate    = flag.String("end", "", "结束日期 (YYYY-MM-DD)")
		days       = flag.Int("days", 30, "回填天数（从今天往前）")
		check      = flag.Bool("check", false, "只检查数据完整性，不回填")
		symbols    = flag.String("symbols", "", "多个交易对，用逗号分隔")
	)
	flag.Parse()

	// 加载配置
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 连接数据库
	db, err := database.NewConnection(&database.Config{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		DBName:          cfg.Database.DBName,
		SSLMode:         cfg.Database.SSLMode,
		MaxOpen:         cfg.Database.MaxOpen,
		MaxIdle:         cfg.Database.MaxIdle,
		Timeout:         cfg.Database.Timeout,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.Database.ConnMaxIdleTime,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
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

	// 解析时间范围
	var startTime, endTime time.Time
	if *startDate != "" && *endDate != "" {
		startTime, err = time.Parse("2006-01-02", *startDate)
		if err != nil {
			log.Fatalf("Invalid start date format: %v", err)
		}
		endTime, err = time.Parse("2006-01-02", *endDate)
		if err != nil {
			log.Fatalf("Invalid end date format: %v", err)
		}
	} else {
		// 使用days参数
		endTime = time.Now()
		startTime = endTime.AddDate(0, 0, -*days)
	}

	// 解析间隔
	intervalEnum := kline.Interval(*interval)

	// 解析交易对列表
	var symbolList []string
	if *symbols != "" {
		symbolList = strings.Split(*symbols, ",")
		for i, s := range symbolList {
			symbolList[i] = strings.TrimSpace(s)
		}
	} else {
		symbolList = []string{*symbol}
	}

	ctx := context.Background()

	fmt.Printf("历史数据回填工具\n")
	fmt.Printf("================\n")
	fmt.Printf("交易对: %v\n", symbolList)
	fmt.Printf("间隔: %s\n", *interval)
	fmt.Printf("时间范围: %s 到 %s\n", startTime.Format("2006-01-02"), endTime.Format("2006-01-02"))
	fmt.Printf("操作模式: %s\n", map[bool]string{true: "检查", false: "回填"}[*check])
	fmt.Printf("================\n\n")

	for _, sym := range symbolList {
		fmt.Printf("处理交易对: %s\n", sym)

		if *check {
			// 只检查数据完整性
			report, err := klineManager.CheckDataIntegrity(ctx, sym, intervalEnum, startTime, endTime)
			if err != nil {
				log.Printf("Failed to check data integrity for %s: %v", sym, err)
				continue
			}

			fmt.Printf("数据完整性报告 - %s %s:\n", sym, *interval)
			fmt.Printf("  期望数据点: %d\n", report.ExpectedCount)
			fmt.Printf("  实际数据点: %d\n", report.ActualCount)
			fmt.Printf("  完整度: %.2f%%\n", report.Completeness)
			fmt.Printf("  数据间隙: %d个\n", len(report.Gaps))

			if report.HasGaps {
				fmt.Printf("  间隙详情:\n")
				for i, gap := range report.Gaps {
					if i >= 5 { // 只显示前5个间隙
						fmt.Printf("    ... 还有 %d 个间隙\n", len(report.Gaps)-5)
						break
					}
					fmt.Printf("    %s 到 %s\n", 
						gap.Start.Format("2006-01-02 15:04"), 
						gap.End.Format("2006-01-02 15:04"))
				}
			}
		} else {
			// 执行数据回填
			err := klineManager.BackfillHistoricalData(ctx, sym, intervalEnum, startTime, endTime)
			if err != nil {
				log.Printf("Failed to backfill data for %s: %v", sym, err)
				continue
			}

			// 回填后检查数据完整性
			report, err := klineManager.CheckDataIntegrity(ctx, sym, intervalEnum, startTime, endTime)
			if err != nil {
				log.Printf("Failed to check data integrity after backfill for %s: %v", sym, err)
			} else {
				fmt.Printf("回填完成 - %s %s:\n", sym, *interval)
				fmt.Printf("  最终数据点: %d\n", report.ActualCount)
				fmt.Printf("  最终完整度: %.2f%%\n", report.Completeness)
			}
		}

		fmt.Printf("\n")
	}

	fmt.Printf("操作完成！\n")
}
