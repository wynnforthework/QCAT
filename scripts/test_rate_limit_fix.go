package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/config"
	"qcat/internal/exchange"
	"qcat/internal/exchange/binance"
)

func main() {
	fmt.Println("🧪 Rate Limit Fix Test")
	fmt.Println("======================")

	// 加载配置
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 创建交易所配置
	exchangeConfig := &exchange.ExchangeConfig{
		Name:      cfg.Exchange.Name,
		APIKey:    cfg.Exchange.APIKey,
		APISecret: cfg.Exchange.APISecret,
		TestNet:   cfg.Exchange.TestNet,
		BaseURL:   cfg.Exchange.BaseURL,
	}

	// 创建速率限制器
	rateLimiter := exchange.NewRateLimiter(nil, 100*time.Millisecond)

	// 创建Binance客户端
	client := binance.NewClient(exchangeConfig, rateLimiter)

	fmt.Printf("Testing with API Key: %s\n", maskAPIKey(cfg.Exchange.APIKey))
	fmt.Printf("Test Net: %v\n", cfg.Exchange.TestNet)
	fmt.Printf("Base URL: %s\n", cfg.Exchange.BaseURL)

	// 测试1: 单个请求
	fmt.Println("\n📋 Test 1: Single Position Request")
	testSinglePositionRequest(client)

	// 测试2: 多个并发请求
	fmt.Println("\n📋 Test 2: Multiple Concurrent Position Requests")
	testConcurrentPositionRequests(client)

	// 测试3: 速率限制恢复
	fmt.Println("\n📋 Test 3: Rate Limit Recovery")
	testRateLimitRecovery(client)

	// 测试4: 不同端点的速率限制
	fmt.Println("\n📋 Test 4: Different Endpoint Rate Limits")
	testDifferentEndpoints(client)

	fmt.Println("\n🎉 Rate limit testing completed!")
}

// testSinglePositionRequest tests a single position request
func testSinglePositionRequest(client *binance.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()
	position, err := client.GetPosition(ctx, "BTCUSDT")
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("❌ Single position request failed: %v (took %v)\n", err, duration)
		
		// Check if it's a rate limit error
		if isRateLimitError(err) {
			fmt.Println("💡 This is a rate limit error - the fix should handle this")
		}
	} else {
		fmt.Printf("✅ Single position request succeeded (took %v)\n", duration)
		if position != nil {
			fmt.Printf("   Position: %s, Size: %.4f\n", position.Symbol, position.Quantity)
		} else {
			fmt.Printf("   No position found for BTCUSDT\n")
		}
	}
}

// testConcurrentPositionRequests tests multiple concurrent position requests
func testConcurrentPositionRequests(client *binance.Client) {
	symbols := []string{"BTCUSDT", "ETHUSDT", "ADAUSDT", "DOTUSDT", "LINKUSDT"}
	
	var wg sync.WaitGroup
	results := make(chan result, len(symbols))

	start := time.Now()
	
	for _, symbol := range symbols {
		wg.Add(1)
		go func(sym string) {
			defer wg.Done()
			
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			
			reqStart := time.Now()
			position, err := client.GetPosition(ctx, sym)
			reqDuration := time.Since(reqStart)
			
			results <- result{
				Symbol:   sym,
				Position: position,
				Error:    err,
				Duration: reqDuration,
			}
		}(symbol)
	}

	wg.Wait()
	close(results)
	
	totalDuration := time.Since(start)
	
	fmt.Printf("Total time for %d concurrent requests: %v\n", len(symbols), totalDuration)
	
	successCount := 0
	rateLimitCount := 0
	
	for res := range results {
		if res.Error != nil {
			if isRateLimitError(res.Error) {
				rateLimitCount++
				fmt.Printf("⚠️  %s: Rate limited (took %v)\n", res.Symbol, res.Duration)
			} else {
				fmt.Printf("❌ %s: Error - %v (took %v)\n", res.Symbol, res.Error, res.Duration)
			}
		} else {
			successCount++
			fmt.Printf("✅ %s: Success (took %v)\n", res.Symbol, res.Duration)
		}
	}
	
	fmt.Printf("Results: %d success, %d rate limited, %d total\n", 
		successCount, rateLimitCount, len(symbols))
}

// testRateLimitRecovery tests rate limit recovery
func testRateLimitRecovery(client *binance.Client) {
	fmt.Println("Testing rate limit recovery by making rapid requests...")
	
	for i := 0; i < 10; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		
		start := time.Now()
		_, err := client.GetPosition(ctx, "BTCUSDT")
		duration := time.Since(start)
		cancel()
		
		if err != nil {
			if isRateLimitError(err) {
				fmt.Printf("Request %d: Rate limited (took %v)\n", i+1, duration)
			} else {
				fmt.Printf("Request %d: Error - %v (took %v)\n", i+1, err, duration)
			}
		} else {
			fmt.Printf("Request %d: Success (took %v)\n", i+1, duration)
		}
		
		// Small delay between requests
		time.Sleep(100 * time.Millisecond)
	}
}

// testDifferentEndpoints tests rate limits for different endpoints
func testDifferentEndpoints(client *binance.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test server time (should have higher rate limit)
	start := time.Now()
	_, err := client.GetServerTime(ctx)
	duration := time.Since(start)
	
	if err != nil {
		fmt.Printf("❌ Server time request failed: %v (took %v)\n", err, duration)
	} else {
		fmt.Printf("✅ Server time request succeeded (took %v)\n", duration)
	}

	// Test exchange info (should have higher rate limit)
	start = time.Now()
	_, err = client.GetExchangeInfo(ctx)
	duration = time.Since(start)
	
	if err != nil {
		fmt.Printf("❌ Exchange info request failed: %v (took %v)\n", err, duration)
	} else {
		fmt.Printf("✅ Exchange info request succeeded (took %v)\n", duration)
	}
}

// result represents the result of a position request
type result struct {
	Symbol   string
	Position *exchange.Position
	Error    error
	Duration time.Duration
}

// maskAPIKey masks an API key for display
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "***" + key[len(key)-4:]
}

// isRateLimitError checks if an error is related to rate limiting
func isRateLimitError(err error) bool {
	errStr := err.Error()
	return contains(errStr, "rate limit") || 
		   contains(errStr, "-1003") ||
		   contains(errStr, "Too Many Requests") ||
		   contains(errStr, "exceeded")
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOfSubstring(s, substr) >= 0
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
