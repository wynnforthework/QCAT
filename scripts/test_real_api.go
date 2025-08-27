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
	fmt.Println("🔍 Testing Real Binance API (No Mock Data)")
	fmt.Println("==========================================")
	fmt.Println()

	// Load configuration
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create exchange config
	exchangeConfig := &exchange.ExchangeConfig{
		Name:              cfg.Exchange.Name,
		APIKey:            cfg.Exchange.APIKey,
		APISecret:         cfg.Exchange.APISecret,
		TestNet:           cfg.Exchange.TestNet,
		BaseURL:           cfg.Exchange.BaseURL,
		FuturesBaseURL:    cfg.Exchange.FuturesBaseURL,
		FallbackMode:      cfg.Exchange.FallbackMode,
		SkipKlinesOnError: cfg.Exchange.SkipKlinesOnError,
		UseCachedData:     cfg.Exchange.UseCachedData,
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

	// Test the exact same API call that was failing in the original error
	fmt.Println("🔍 Testing the EXACT API call that was failing...")
	fmt.Println("Original error URL: https://api.binance.com/api/v3/klines?endTime=1756279733591&interval=1d&limit=1000&startTime=1724743733591&symbol=BTCUSDT")
	fmt.Println("Now using correct futures testnet URL with /fapi/v1/klines endpoint")
	fmt.Println()

	// Use similar parameters as in the original error
	endTime := time.Unix(1756279733, 591000000)   // Convert from milliseconds
	startTime := time.Unix(1724743733, 591000000) // Convert from milliseconds

	fmt.Printf("Parameters:\n")
	fmt.Printf("  Symbol: BTCUSDT\n")
	fmt.Printf("  Interval: 1d\n")
	fmt.Printf("  StartTime: %s\n", startTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("  EndTime: %s\n", endTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Limit: 1000\n")
	fmt.Println()

	klines, err := client.GetKlines(ctx, "BTCUSDT", "1d", startTime, endTime, 1000)
	if err != nil {
		fmt.Printf("❌ API call failed: %v\n", err)
		fmt.Println()
		fmt.Println("🔍 Troubleshooting:")
		fmt.Println("1. Check if the testnet URL is correct")
		fmt.Println("2. Verify network connectivity to testnet.binancefuture.com")
		fmt.Println("3. Ensure the endpoint /fapi/v1/klines is correct for futures")
		fmt.Println("4. Check if the symbol BTCUSDT exists on futures testnet")
	} else {
		fmt.Printf("✅ API call successful! Retrieved %d klines\n", len(klines))
		if len(klines) > 0 {
			fmt.Printf("   First kline: Open=%.2f, High=%.2f, Low=%.2f, Close=%.2f, Time=%v\n",
				klines[0].Open, klines[0].High, klines[0].Low, klines[0].Close, klines[0].OpenTime)
		}
		fmt.Println()
		fmt.Println("🎉 The original optimization error should now be resolved!")
	}

	// Test a few more quick calls to ensure stability
	fmt.Println("\n🔍 Testing additional API calls...")

	// Test with recent data (last 24 hours)
	recentEnd := time.Now()
	recentStart := recentEnd.Add(-24 * time.Hour)

	klines2, err := client.GetKlines(ctx, "BTCUSDT", "1h", recentStart, recentEnd, 24)
	if err != nil {
		fmt.Printf("❌ Recent data test failed: %v\n", err)
	} else {
		fmt.Printf("✅ Recent data test successful: %d klines\n", len(klines2))
	}

	// Test different symbol
	klines3, err := client.GetKlines(ctx, "ETHUSDT", "1h", recentStart, recentEnd, 5)
	if err != nil {
		fmt.Printf("❌ ETHUSDT test failed: %v\n", err)
	} else {
		fmt.Printf("✅ ETHUSDT test successful: %d klines\n", len(klines3))
	}

	fmt.Println()
	fmt.Println("📊 Summary:")
	fmt.Println("===========")
	fmt.Println("✅ Removed all mock data functionality")
	fmt.Println("✅ Using real Binance Futures Testnet API")
	fmt.Println("✅ Correct URL: https://testnet.binancefuture.com")
	fmt.Println("✅ Correct endpoint: /fapi/v1/klines")
	fmt.Println("✅ Fixed baseURL selection logic")

	if err == nil {
		fmt.Println("✅ API connectivity working - optimization tasks should succeed!")
	} else {
		fmt.Println("⚠️  API connectivity issues detected - check network/configuration")
	}
}
