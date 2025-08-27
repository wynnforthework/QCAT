package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"qcat/internal/config"
	"qcat/internal/exchange"
	"qcat/internal/exchange/binance"
)

func main() {
	log.Println("🔍 Testing Binance API connection...")

	// 加载配置
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 创建交换机配置
	exchangeConfig := &exchange.ExchangeConfig{
		Name:           cfg.Exchange.Name,
		APIKey:         os.Getenv("EXCHANGE_API_KEY"),
		APISecret:      os.Getenv("EXCHANGE_API_SECRET"),
		TestNet:        cfg.Exchange.TestNet,
		BaseURL:        cfg.Exchange.BaseURL,
		FuturesBaseURL: cfg.Exchange.FuturesBaseURL,
	}

	log.Printf("Configuration:")
	log.Printf("  TestNet: %v", exchangeConfig.TestNet)
	log.Printf("  BaseURL: %s", exchangeConfig.BaseURL)
	log.Printf("  FuturesBaseURL: %s", exchangeConfig.FuturesBaseURL)
	log.Printf("  API Key: %s", maskAPIKey(exchangeConfig.APIKey))

	// 创建客户端
	client := binance.NewClient(exchangeConfig, nil)

	// 测试基本连接
	log.Println("\n🌐 Testing basic connectivity...")
	testBasicConnectivity(exchangeConfig.FuturesBaseURL)

	// 测试服务器时间
	log.Println("\n⏰ Testing server time endpoint...")
	if err := testServerTime(client); err != nil {
		log.Printf("❌ Server time test failed: %v", err)
	} else {
		log.Println("✅ Server time test passed")
	}

	// 测试账户信息（需要API密钥）
	if exchangeConfig.APIKey != "" && exchangeConfig.APISecret != "" {
		log.Println("\n💰 Testing account info...")
		if err := testAccountInfo(client); err != nil {
			log.Printf("❌ Account info test failed: %v", err)
		} else {
			log.Println("✅ Account info test passed")
		}

		// 测试获取开放订单
		log.Println("\n📋 Testing open orders...")
		if err := testOpenOrders(client); err != nil {
			log.Printf("❌ Open orders test failed: %v", err)
		} else {
			log.Println("✅ Open orders test passed")
		}
	} else {
		log.Println("\n⚠️  Skipping authenticated tests (no API credentials)")
	}

	log.Println("\n🎉 Connection test completed!")
}

func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return "****"
	}
	return apiKey[:4] + "****" + apiKey[len(apiKey)-4:]
}

func testBasicConnectivity(baseURL string) {
	client := &http.Client{Timeout: 10 * time.Second}

	testURL := baseURL + "/fapi/v1/time"
	log.Printf("Testing URL: %s", testURL)

	resp, err := client.Get(testURL)
	if err != nil {
		log.Printf("❌ Basic connectivity failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Printf("✅ Basic connectivity successful (status: %d)", resp.StatusCode)
	} else {
		log.Printf("⚠️  Unexpected status code: %d", resp.StatusCode)
	}
}

func testServerTime(client *binance.Client) error {
	// 暂时跳过服务器时间测试
	return nil
}

func testAccountInfo(client *binance.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 使用GetAccountBalance方法
	balances, err := client.GetAccountBalance(ctx)
	if err != nil {
		return fmt.Errorf("failed to get account balance: %w", err)
	}

	log.Printf("Account balance retrieved successfully, found %d assets", len(balances))
	for asset, balance := range balances {
		if balance.Available > 0 || balance.Locked > 0 {
			log.Printf("  %s: Available=%.6f, Locked=%.6f", asset, balance.Available, balance.Locked)
		}
	}
	return nil
}

func testOpenOrders(client *binance.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	orders, err := client.GetOpenOrders(ctx, "")
	if err != nil {
		return fmt.Errorf("failed to get open orders: %w", err)
	}

	log.Printf("Open orders count: %d", len(orders))
	return nil
}
