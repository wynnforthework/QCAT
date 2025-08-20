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
	fmt.Println("🔑 API Configuration Validator")
	fmt.Println("==============================")
	fmt.Println("📊 Trading Mode: FUTURES (合约模式)")
	fmt.Println("🌐 API Type: Binance Futures API")
	fmt.Println()

	// Load configuration
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Check if API credentials are configured
	fmt.Println("\n📋 API Configuration Check:")
	if cfg.Exchange.APIKey == "" || cfg.Exchange.APIKey == "your_real_testnet_api_key_here" {
		fmt.Println("❌ API Key: Not configured or using placeholder")
		fmt.Println("   Please set EXCHANGE_API_KEY in your .env file")
		fmt.Println("   Get FUTURES testnet keys from: https://testnet.binancefuture.com/")
		fmt.Println("   ⚠️  Important: Enable 'Futures' permission when creating API key")
		return
	} else {
		fmt.Printf("✅ API Key: Configured (%s...)\n", maskAPIKey(cfg.Exchange.APIKey))
	}

	if cfg.Exchange.APISecret == "" || cfg.Exchange.APISecret == "your_real_testnet_api_secret_here" {
		fmt.Println("❌ API Secret: Not configured or using placeholder")
		fmt.Println("   Please set EXCHANGE_API_SECRET in your .env file")
		fmt.Println("   ⚠️  Important: This must be a FUTURES API key, not spot")
		return
	} else {
		fmt.Printf("✅ API Secret: Configured (%s...)\n", maskAPIKey(cfg.Exchange.APISecret))
	}

	fmt.Printf("✅ Exchange: %s\n", cfg.Exchange.Name)
	fmt.Printf("✅ TestNet: %t\n", cfg.Exchange.TestNet)
	fmt.Printf("✅ Base URL: %s\n", cfg.Exchange.FuturesBaseURL)

	// Test API connection
	fmt.Println("\n🔗 Testing API Connection:")

	exchangeConfig := &exchange.ExchangeConfig{
		Name:      cfg.Exchange.Name,
		APIKey:    cfg.Exchange.APIKey,
		APISecret: cfg.Exchange.APISecret,
		TestNet:   cfg.Exchange.TestNet,
		BaseURL:   cfg.Exchange.FuturesBaseURL,
	}

	// Create rate limiter with proper limits
	rateLimiter := exchange.NewRateLimiter(nil, 100*time.Millisecond)
	rateLimiter.AddLimit("server_time", time.Second, 10)
	rateLimiter.AddLimit("exchange_info", time.Second, 10)
	rateLimiter.AddLimit("account", time.Second, 5)
	rateLimiter.AddLimit("positions", time.Second, 5)
	rateLimiter.AddLimit("get_symbol_price", time.Second, 10)

	// Create client
	client := binance.NewClient(exchangeConfig, rateLimiter)

	// Test server time (doesn't require authentication)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fmt.Print("Testing server connectivity... ")
	serverTime, err := client.GetServerTime(ctx)
	if err != nil {
		fmt.Printf("❌ Failed: %v\n", err)
		fmt.Println("   Check your internet connection and exchange URL")
		return
	}
	fmt.Printf("✅ Success (Server time: %v)\n", serverTime)

	// Test account access (requires valid API key)
	fmt.Print("Testing API key authentication... ")
	_, err = client.GetAccountBalance(ctx)
	if err != nil {
		fmt.Printf("❌ Failed: %v\n", err)
		fmt.Println("   This indicates your API key/secret is invalid or has insufficient permissions")
		fmt.Println("   Please check:")
		fmt.Println("   1. API key and secret are correct")
		fmt.Println("   2. API key has futures trading permissions enabled")
		fmt.Println("   3. IP restrictions (if any) allow your current IP")
		return
	}
	fmt.Println("✅ Success - API credentials are valid!")

	// Test position access
	fmt.Print("Testing position access... ")
	_, err = client.GetPositions(ctx)
	if err != nil {
		fmt.Printf("❌ Failed: %v\n", err)
		fmt.Println("   API key may not have position access permissions")
		return
	}
	fmt.Println("✅ Success - Position access working!")

	fmt.Println("\n🎉 All API configuration tests passed!")
	fmt.Println("Your application should now work without API errors.")
}

// maskAPIKey masks an API key for safe display
func maskAPIKey(key string) string {
	if len(key) < 8 {
		return "***"
	}
	return key[:4] + "..." + key[len(key)-4:]
}
