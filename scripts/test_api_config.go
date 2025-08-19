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
	// 加载配置
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Printf("Loaded configuration:\n")
	fmt.Printf("Exchange Name: %s\n", cfg.Exchange.Name)
	fmt.Printf("API Key: %s\n", maskAPIKey(cfg.Exchange.APIKey))
	fmt.Printf("API Secret: %s\n", maskAPIKey(cfg.Exchange.APISecret))
	fmt.Printf("Test Net: %v\n", cfg.Exchange.TestNet)
	fmt.Printf("Base URL: %s\n", cfg.Exchange.BaseURL)

	// 检查API密钥是否为占位符
	if cfg.Exchange.APIKey == "your_api_key" || cfg.Exchange.APIKey == "" {
		fmt.Println("❌ API Key is not configured properly (still using placeholder)")
		fmt.Println("Please set EXCHANGE_API_KEY environment variable with your actual Binance API key")
		return
	}

	if cfg.Exchange.APISecret == "your_api_secret" || cfg.Exchange.APISecret == "" {
		fmt.Println("❌ API Secret is not configured properly (still using placeholder)")
		fmt.Println("Please set EXCHANGE_API_SECRET environment variable with your actual Binance API secret")
		return
	}

	// 创建交易所配置
	exchangeConfig := &exchange.ExchangeConfig{
		Name:      cfg.Exchange.Name,
		APIKey:    cfg.Exchange.APIKey,
		APISecret: cfg.Exchange.APISecret,
		TestNet:   cfg.Exchange.TestNet,
		BaseURL:   cfg.Exchange.BaseURL,
	}

	// 创建速率限制器（使用内存缓存）
	rateLimiter := exchange.NewRateLimiter(nil, 100*time.Millisecond)

	// 创建Binance客户端
	client := binance.NewClient(exchangeConfig, rateLimiter)

	fmt.Println("\n🔍 Testing API connection...")

	// 测试连接 - 获取服务器时间
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverTime, err := client.GetServerTime(ctx)
	if err != nil {
		fmt.Printf("❌ Failed to get server time: %v\n", err)

		// 检查是否是API密钥格式错误
		if isAPIKeyFormatError(err) {
			fmt.Println("\n💡 This appears to be an API key format error (-2014)")
			fmt.Println("Possible solutions:")
			fmt.Println("1. Check that your API key is correctly formatted")
			fmt.Println("2. Ensure there are no extra spaces or characters")
			fmt.Println("3. Verify the API key is active on Binance")
			fmt.Println("4. Check that the API key has the required permissions")
		}
		return
	}

	fmt.Printf("✅ Server time: %v\n", serverTime)

	// 测试获取交易所信息
	fmt.Println("\n🔍 Testing exchange info...")
	exchangeInfo, err := client.GetExchangeInfo(ctx)
	if err != nil {
		fmt.Printf("❌ Failed to get exchange info: %v\n", err)
		return
	}

	fmt.Printf("✅ Exchange: %s\n", exchangeInfo.Name)
	fmt.Printf("✅ Server time: %v\n", exchangeInfo.ServerTime)
	fmt.Printf("✅ Rate limits: %d\n", len(exchangeInfo.RateLimits))

	// 测试获取账户信息（需要签名）
	fmt.Println("\n🔍 Testing account balance (signed request)...")
	balance, err := client.GetAccountBalance(ctx)
	if err != nil {
		fmt.Printf("❌ Failed to get account balance: %v\n", err)

		// 检查是否是API密钥格式错误
		if isAPIKeyFormatError(err) {
			fmt.Println("\n💡 This appears to be an API key format error (-2014)")
			fmt.Println("This error occurs when making signed requests")
			fmt.Println("Possible solutions:")
			fmt.Println("1. Check that your API secret is correctly formatted")
			fmt.Println("2. Ensure the API key has 'Enable Futures' permission")
			fmt.Println("3. Verify the timestamp is within acceptable range")
			fmt.Println("4. Check the signature algorithm implementation")
		}
		return
	}

	fmt.Printf("✅ Account balance retrieved successfully\n")
	fmt.Printf("✅ Total balance entries: %d\n", len(balance))

	fmt.Println("\n🎉 API configuration test completed successfully!")
	fmt.Println("Your Binance API credentials are working correctly.")
}

// maskAPIKey masks an API key for display
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "***" + key[len(key)-4:]
}

// isAPIKeyFormatError checks if the error is related to API key format
func isAPIKeyFormatError(err error) bool {
	errStr := err.Error()
	return contains(errStr, "-2014") ||
		contains(errStr, "API-key format invalid") ||
		contains(errStr, "Invalid API-key") ||
		contains(errStr, "signature") ||
		contains(errStr, "authentication")
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					indexOfSubstring(s, substr) >= 0)))
}

// indexOfSubstring finds the index of a substring
func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
