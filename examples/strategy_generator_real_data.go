package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"qcat/internal/config"
	"qcat/internal/database"
	"qcat/internal/exchange"
	"qcat/internal/strategy/generator"
)

func main() {
	// 1. 加载配置
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. 使用生产环境服务（带真实数据源）
	fmt.Println("=== 使用真实数据源创建策略生成服务 ===")
	
	// 数据库配置
	dbConfig := &database.Config{
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
	}

	// 交易所配置
	exchangeConfig := &exchange.ExchangeConfig{
		APIKey:    cfg.Exchange.APIKey,
		APISecret: cfg.Exchange.APISecret,
		TestNet:   cfg.Exchange.TestNet,
	}

	// 创建生产环境服务
	service, err := generator.CreateProductionService(dbConfig, exchangeConfig)
	if err != nil {
		log.Fatalf("Failed to create production service: %v", err)
	}

	fmt.Println("✓ 策略生成服务创建成功（使用真实数据源）")

	// 3. 生成策略（使用真实市场数据）
	ctx := context.Background()
	
	req := &generator.GenerationRequest{
		Symbol:     "BTCUSDT",
		Exchange:   "binance",
		TimeRange:  7 * 24 * time.Hour, // 7天历史数据
		Objective:  "sharpe",
		RiskLevel:  "medium",
		MarketType: "trending",
	}

	fmt.Printf("正在为 %s 生成策略（基于真实市场数据）...\n", req.Symbol)
	
	result, err := service.GenerateStrategy(ctx, req)
	if err != nil {
		log.Fatalf("Failed to generate strategy: %v", err)
	}

	// 4. 显示结果
	fmt.Println("\n=== 策略生成结果 ===")
	fmt.Printf("策略名称: %s\n", result.Strategy.Name)
	fmt.Printf("策略描述: %s\n", result.Strategy.Description)
	fmt.Printf("预期收益: %.2f%%\n", result.ExpectedReturn*100)
	fmt.Printf("预期夏普比率: %.2f\n", result.ExpectedSharpe)
	fmt.Printf("预期最大回撤: %.2f%%\n", result.ExpectedDrawdown*100)
	fmt.Printf("置信度: %.1f%%\n", result.Confidence*100)

	fmt.Println("\n策略参数:")
	for key, value := range result.Strategy.Params {
		fmt.Printf("  %s: %v\n", key, value)
	}

	// 5. 演示自动生成服务（批量生成）
	fmt.Println("\n=== 自动批量策略生成 ===")
	
	autoService, err := generator.CreateProductionAutoService(dbConfig, exchangeConfig)
	if err != nil {
		log.Fatalf("Failed to create auto generation service: %v", err)
	}

	symbols := []string{"BTCUSDT", "ETHUSDT", "ADAUSDT"}
	maxStrategies := 5

	fmt.Printf("为 %v 自动生成最多 %d 个策略...\n", symbols, maxStrategies)
	
	results, err := autoService.AutoGenerateStrategies(ctx, symbols, maxStrategies)
	if err != nil {
		log.Fatalf("Failed to auto-generate strategies: %v", err)
	}

	fmt.Printf("✓ 成功生成 %d 个策略\n", len(results))
	
	for i, result := range results {
		fmt.Printf("\n策略 %d:\n", i+1)
		fmt.Printf("  名称: %s\n", result.Strategy.Name)
		fmt.Printf("  交易对: %s\n", result.Strategy.Symbol)
		fmt.Printf("  预期收益: %.2f%%\n", result.ExpectedReturn*100)
		fmt.Printf("  置信度: %.1f%%\n", result.Confidence*100)
	}

	fmt.Println("\n=== 演示完成 ===")
	fmt.Println("所有策略都基于真实的市场数据和历史表现数据生成")
}

// demonstrateWithMockData 演示使用模拟数据的情况（不推荐用于生产）
func demonstrateWithMockData() {
	fmt.Println("=== 使用模拟数据演示（不推荐用于生产） ===")
	
	// 使用默认服务（会产生警告）
	service := generator.NewService()
	
	ctx := context.Background()
	req := &generator.GenerationRequest{
		Symbol:     "BTCUSDT",
		Exchange:   "binance",
		TimeRange:  7 * 24 * time.Hour,
		Objective:  "sharpe",
		RiskLevel:  "medium",
		MarketType: "trending",
	}

	result, err := service.GenerateStrategy(ctx, req)
	if err != nil {
		log.Printf("Failed to generate strategy with mock data: %v", err)
		return
	}

	fmt.Printf("策略名称: %s (基于模拟数据)\n", result.Strategy.Name)
	fmt.Printf("预期收益: %.2f%% (可能不准确)\n", result.ExpectedReturn*100)
	fmt.Printf("置信度: %.1f%% (基于模拟数据)\n", result.Confidence*100)
	
	fmt.Println("\n警告：此策略基于模拟数据生成，不适用于生产环境")
}
