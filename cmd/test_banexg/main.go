package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"qcat/internal/config"
	"qcat/internal/exchange"
	"qcat/internal/exchange/binance"
)

func main() {
	// Load configuration
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Check if API credentials are set
	if cfg.Exchange.APIKey == "" || cfg.Exchange.APISecret == "" {
		log.Println("API credentials not set. Please set EXCHANGE_API_KEY and EXCHANGE_API_SECRET environment variables.")
		log.Println("For testing purposes, you can set them like this:")
		log.Println("export EXCHANGE_API_KEY=your_api_key")
		log.Println("export EXCHANGE_API_SECRET=your_api_secret")

		// Try to get from environment directly
		apiKey := os.Getenv("EXCHANGE_API_KEY")
		apiSecret := os.Getenv("EXCHANGE_API_SECRET")

		if apiKey == "" || apiSecret == "" {
			log.Fatal("API credentials are required for testing")
		}

		cfg.Exchange.APIKey = apiKey
		cfg.Exchange.APISecret = apiSecret
	}

	// Create exchange config
	exchangeConfig := &exchange.ExchangeConfig{
		Name:      cfg.Exchange.Name,
		APIKey:    cfg.Exchange.APIKey,
		APISecret: cfg.Exchange.APISecret,
		TestNet:   cfg.Exchange.TestNet,
		BaseURL:   cfg.Exchange.BaseURL,
	}

	// Create rate limiter (simple implementation)
	rateLimiter := exchange.NewSimpleRateLimiter(1200, time.Minute) // 1200 requests per minute

	// Create Binance client
	client := binance.NewClient(exchangeConfig, rateLimiter)

	ctx := context.Background()

	fmt.Println("=== Testing Binance Exchange Integration with banexg SDK ===")
	fmt.Printf("Using TestNet: %v\n", cfg.Exchange.TestNet)
	fmt.Println()

	// Test 1: Get Server Time
	fmt.Println("1. Testing GetServerTime...")
	serverTime, err := client.GetServerTime(ctx)
	if err != nil {
		log.Printf("Failed to get server time: %v", err)
	} else {
		fmt.Printf("Server time: %v\n", serverTime)
	}
	fmt.Println()

	// Test 2: Get Account Balance
	fmt.Println("2. Testing GetAccountBalance...")
	balances, err := client.GetAccountBalance(ctx)
	if err != nil {
		log.Printf("Failed to get account balance: %v", err)
	} else {
		fmt.Printf("Account balances (%d assets):\n", len(balances))
		for asset, balance := range balances {
			if balance.Total > 0 {
				fmt.Printf("  %s: Total=%.8f, Available=%.8f, Locked=%.8f\n",
					asset, balance.Total, balance.Available, balance.Locked)
			}
		}
	}
	fmt.Println()

	// Test 3: Get Positions
	fmt.Println("3. Testing GetPositions...")
	positions, err := client.GetPositions(ctx)
	if err != nil {
		log.Printf("Failed to get positions: %v", err)
	} else {
		fmt.Printf("Positions (%d positions):\n", len(positions))
		for _, pos := range positions {
			if pos.Size > 0 {
				fmt.Printf("  %s: Side=%s, Size=%.8f, Entry=%.4f, Mark=%.4f, PnL=%.4f, Leverage=%dx\n",
					pos.Symbol, pos.Side, pos.Size, pos.EntryPrice, pos.MarkPrice, pos.UnrealizedPnL, pos.Leverage)
			}
		}
	}
	fmt.Println()

	// Test 4: Get Exchange Info
	fmt.Println("4. Testing GetExchangeInfo...")
	exchangeInfo, err := client.GetExchangeInfo(ctx)
	if err != nil {
		log.Printf("Failed to get exchange info: %v", err)
	} else {
		fmt.Printf("Exchange: %s, Server Time: %v, Symbols: %d\n",
			exchangeInfo.Name, exchangeInfo.ServerTime, len(exchangeInfo.Symbols))

		// Show first few symbols
		fmt.Println("Sample symbols:")
		for i, symbol := range exchangeInfo.Symbols {
			if i >= 5 {
				break
			}
			fmt.Printf("  %s (%s/%s) - Status: %s\n",
				symbol.Symbol, symbol.BaseAsset, symbol.QuoteAsset, symbol.Status)
		}
	}
	fmt.Println()

	// Test 5: Get Symbol Price
	fmt.Println("5. Testing GetSymbolPrice for BTCUSDT...")
	price, err := client.GetSymbolPrice(ctx, "BTCUSDT")
	if err != nil {
		log.Printf("Failed to get symbol price: %v", err)
	} else {
		fmt.Printf("BTCUSDT price: %.2f\n", price)
	}
	fmt.Println()

	// Test 6: Test specific position
	fmt.Println("6. Testing GetPosition for BTCUSDT...")
	position, err := client.GetPosition(ctx, "BTCUSDT")
	if err != nil {
		log.Printf("Failed to get position for BTCUSDT: %v", err)
	} else {
		fmt.Printf("BTCUSDT position: Side=%s, Size=%.8f, Entry=%.4f, Mark=%.4f, PnL=%.4f\n",
			position.Side, position.Size, position.EntryPrice, position.MarkPrice, position.UnrealizedPnL)
	}
	fmt.Println()

	// Test 7: Get Leverage
	fmt.Println("7. Testing GetLeverage for BTCUSDT...")
	leverage, err := client.GetLeverage(ctx, "BTCUSDT")
	if err != nil {
		log.Printf("Failed to get leverage for BTCUSDT: %v", err)
	} else {
		fmt.Printf("BTCUSDT leverage: %dx\n", leverage)
	}
	fmt.Println()

	// Test 8: Get Risk Limits
	fmt.Println("8. Testing GetRiskLimits for BTCUSDT...")
	riskLimits, err := client.GetRiskLimits(ctx, "BTCUSDT")
	if err != nil {
		log.Printf("Failed to get risk limits for BTCUSDT: %v", err)
	} else {
		fmt.Printf("BTCUSDT risk limits: MaxLeverage=%d, MaxPosition=%.0f, MaxOrder=%.0f\n",
			riskLimits.MaxLeverage, riskLimits.MaxPositionValue, riskLimits.MaxOrderValue)
	}
	fmt.Println()

	// Test 9: Get Margin Info
	fmt.Println("9. Testing GetMarginInfo...")
	marginInfo, err := client.GetMarginInfo(ctx)
	if err != nil {
		log.Printf("Failed to get margin info: %v", err)
	} else {
		fmt.Printf("Margin info: TotalAsset=%.4f, TotalDebt=%.4f, MarginRatio=%.4f\n",
			marginInfo.TotalAssetValue, marginInfo.TotalDebtValue, marginInfo.MarginRatio)
	}
	fmt.Println()

	// Cleanup
	if err := client.Close(); err != nil {
		log.Printf("Failed to close client: %v", err)
	}

	fmt.Println("=== Testing completed ===")
}
