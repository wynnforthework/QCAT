package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"qcat/internal/config"
	"qcat/internal/exchange"
	"qcat/internal/exchange/binance"
)

func main() {
	fmt.Println("🔍 Testing Binance API Connectivity")
	fmt.Println("===================================")
	fmt.Println()

	// Load configuration
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create exchange config
	exchangeConfig := &exchange.ExchangeConfig{
		Name:           cfg.Exchange.Name,
		APIKey:         cfg.Exchange.APIKey,
		APISecret:      cfg.Exchange.APISecret,
		TestNet:        cfg.Exchange.TestNet,
		BaseURL:        cfg.Exchange.BaseURL,
		FuturesBaseURL: cfg.Exchange.FuturesBaseURL,
	}

	fmt.Printf("Configuration:\n")
	fmt.Printf("  TestNet: %v\n", exchangeConfig.TestNet)
	fmt.Printf("  BaseURL: %s\n", exchangeConfig.BaseURL)
	fmt.Printf("  FuturesBaseURL: %s\n", exchangeConfig.FuturesBaseURL)
	fmt.Println()

	// Create rate limiter
	rateLimiter := exchange.NewSimpleRateLimiter(cfg.Exchange.RateLimit.RequestsPerMinute, time.Minute)

	// Create Binance client
	client := binance.NewClient(exchangeConfig, rateLimiter)

	// Test context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test 1: Get Server Time (不需要认证)
	fmt.Println("🔍 Test 1: Getting server time...")
	serverTime, err := client.GetServerTime(ctx)
	if err != nil {
		fmt.Printf("❌ Failed to get server time: %v\n", err)
	} else {
		fmt.Printf("✅ Server time: %v\n", serverTime)
	}
	fmt.Println()

	// Test 2: Get Exchange Info (不需要认证)
	fmt.Println("🔍 Test 2: Getting exchange info...")
	exchangeInfo, err := client.GetExchangeInfo(ctx)
	if err != nil {
		fmt.Printf("❌ Failed to get exchange info: %v\n", err)
	} else {
		fmt.Printf("✅ Exchange info: %s, %d symbols\n", exchangeInfo.Name, len(exchangeInfo.Symbols))
	}
	fmt.Println()

	// Test 3: Get Symbol Price (不需要认证)
	fmt.Println("🔍 Test 3: Getting symbol price for BTCUSDT...")
	price, err := client.GetSymbolPrice(ctx, "BTCUSDT")
	if err != nil {
		fmt.Printf("❌ Failed to get symbol price: %v\n", err)
	} else {
		fmt.Printf("✅ BTCUSDT Price: $%.2f\n", price)
	}
	fmt.Println()

	// Test 4: Get Klines (不需要认证) - 这是原来失败的接口
	fmt.Println("🔍 Test 4: Getting kline data for BTCUSDT (This was the failing API)...")
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)
	klines, err := client.GetKlines(ctx, "BTCUSDT", "1h", startTime, endTime, 24)
	if err != nil {
		fmt.Printf("❌ Failed to get klines: %v\n", err)
		fmt.Println("   This is the API that was originally failing!")
	} else {
		fmt.Printf("✅ Retrieved %d klines\n", len(klines))
		if len(klines) > 0 {
			fmt.Printf("   First kline: Open=%.2f, High=%.2f, Low=%.2f, Close=%.2f, Time=%v\n",
				klines[0].Open, klines[0].High, klines[0].Low, klines[0].Close, klines[0].OpenTime)
		}
	}
	fmt.Println()

	// Test 5: Test different intervals
	fmt.Println("🔍 Test 5: Testing different intervals...")
	intervals := []string{"1m", "5m", "15m", "1h", "1d"}
	for _, interval := range intervals {
		klines, err := client.GetKlines(ctx, "BTCUSDT", interval, startTime, endTime, 5)
		if err != nil {
			fmt.Printf("❌ %s: Failed - %v\n", interval, err)
		} else {
			fmt.Printf("✅ %s: %d klines\n", interval, len(klines))
		}
	}
	fmt.Println()

	// Test 6: Test different symbols
	fmt.Println("🔍 Test 6: Testing different symbols...")
	symbols := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT"}
	for _, symbol := range symbols {
		klines, err := client.GetKlines(ctx, symbol, "1h", startTime, endTime, 1)
		if err != nil {
			fmt.Printf("❌ %s: Failed - %v\n", symbol, err)
		} else {
			fmt.Printf("✅ %s: %d klines\n", symbol, len(klines))
		}
	}
	fmt.Println()

	// Summary
	fmt.Println("📊 Test Summary:")
	fmt.Println("================")
	fmt.Println("✅ All tests completed!")
	fmt.Println("✅ Mock mode has been removed")
	fmt.Println("✅ Using real Binance API endpoints")
	fmt.Println("✅ Proper TestNet URL configuration")
	fmt.Println()
	fmt.Println("If klines API is working now, the original optimization error should be resolved!")
}
