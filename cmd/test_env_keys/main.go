package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"qcat/internal/exchange"
	"qcat/internal/exchange/binance"
)

// loadEnvFile loads environment variables from .env file
func loadEnvFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// Remove quotes if present
			if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
				value = value[1 : len(value)-1]
			}
			os.Setenv(key, value)
		}
	}

	return scanner.Err()
}

func main() {
	fmt.Println("Starting banexg SDK test...")

	// Load .env file
	if err := loadEnvFile(".env"); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	} else {
		fmt.Println("✅ .env file loaded successfully")
	}

	// Get API credentials from environment
	apiKey := os.Getenv("EXCHANGE_API_KEY")
	apiSecret := os.Getenv("EXCHANGE_API_SECRET")

	fmt.Printf("API Key from env: %s\n", apiKey)
	fmt.Printf("API Secret from env: %s\n", apiSecret)

	if apiKey == "" || apiSecret == "" {
		log.Fatal("API credentials not found. Please set EXCHANGE_API_KEY and EXCHANGE_API_SECRET in .env file")
	}

	// Create exchange config for testnet
	exchangeConfig := &exchange.ExchangeConfig{
		Name:      "binance",
		APIKey:    apiKey,
		APISecret: apiSecret,
		TestNet:   true, // Use testnet
		BaseURL:   "https://testnet.binancefuture.com",
	}

	// Create rate limiter
	rateLimiter := exchange.NewSimpleRateLimiter(1200, time.Minute)

	// Create Binance client
	client := binance.NewClient(exchangeConfig, rateLimiter)

	ctx := context.Background()

	fmt.Println("=== Testing Binance Testnet with .env API Keys ===")
	fmt.Printf("API Key: %s...%s\n", apiKey[:8], apiKey[len(apiKey)-8:])
	fmt.Printf("Using TestNet: %v\n", exchangeConfig.TestNet)
	fmt.Println()

	// Test 1: Get Server Time
	fmt.Println("1. Testing GetServerTime...")
	serverTime, err := client.GetServerTime(ctx)
	if err != nil {
		log.Printf("❌ Failed to get server time: %v", err)
	} else {
		fmt.Printf("✅ Server time: %v\n", serverTime)
	}
	fmt.Println()

	// Test 2: Get Account Balance
	fmt.Println("2. Testing GetAccountBalance...")
	balances, err := client.GetAccountBalance(ctx)
	if err != nil {
		log.Printf("❌ Failed to get account balance: %v", err)
	} else {
		fmt.Printf("✅ Account balances (%d assets):\n", len(balances))
		for asset, balance := range balances {
			if balance.Total > 0 {
				fmt.Printf("  %s: Total=%.8f, Available=%.8f, Locked=%.8f\n",
					asset, balance.Total, balance.Available, balance.Locked)
			}
		}
		if len(balances) == 0 {
			fmt.Println("  No balances found (this is normal for testnet)")
		}
	}
	fmt.Println()

	// Test 3: Get Positions
	fmt.Println("3. Testing GetPositions...")
	positions, err := client.GetPositions(ctx)
	if err != nil {
		log.Printf("❌ Failed to get positions: %v", err)
	} else {
		fmt.Printf("✅ Positions (%d positions):\n", len(positions))
		for _, pos := range positions {
			if pos.Size > 0 {
				fmt.Printf("  %s: Side=%s, Size=%.8f, Entry=%.4f, Mark=%.4f, PnL=%.4f, Leverage=%dx\n",
					pos.Symbol, pos.Side, pos.Size, pos.EntryPrice, pos.MarkPrice, pos.UnrealizedPnL, pos.Leverage)
			}
		}
		if len(positions) == 0 {
			fmt.Println("  No positions found")
		}
	}
	fmt.Println()

	// Test 4: Get Exchange Info (limited symbols)
	fmt.Println("4. Testing GetExchangeInfo...")
	exchangeInfo, err := client.GetExchangeInfo(ctx)
	if err != nil {
		log.Printf("❌ Failed to get exchange info: %v", err)
	} else {
		fmt.Printf("✅ Exchange: %s, Server Time: %v, Symbols: %d\n",
			exchangeInfo.Name, exchangeInfo.ServerTime, len(exchangeInfo.Symbols))

		// Show first few symbols
		fmt.Println("Sample symbols:")
		count := 0
		for _, symbol := range exchangeInfo.Symbols {
			if count >= 5 {
				break
			}
			if symbol.Status == "TRADING" {
				fmt.Printf("  %s (%s/%s) - Status: %s\n",
					symbol.Symbol, symbol.BaseAsset, symbol.QuoteAsset, symbol.Status)
				count++
			}
		}
	}
	fmt.Println()

	// Test 5: Get Symbol Price
	fmt.Println("5. Testing GetSymbolPrice for BTCUSDT...")
	price, err := client.GetSymbolPrice(ctx, "BTCUSDT")
	if err != nil {
		log.Printf("❌ Failed to get symbol price: %v", err)
	} else {
		fmt.Printf("✅ BTCUSDT price: %.2f\n", price)
	}
	fmt.Println()

	// Test 6: Test specific position
	fmt.Println("6. Testing GetPosition for BTCUSDT...")
	position, err := client.GetPosition(ctx, "BTCUSDT")
	if err != nil {
		log.Printf("❌ Failed to get position for BTCUSDT: %v", err)
	} else {
		fmt.Printf("✅ BTCUSDT position: Side=%s, Size=%.8f, Entry=%.4f, Mark=%.4f, PnL=%.4f\n",
			position.Side, position.Size, position.EntryPrice, position.MarkPrice, position.UnrealizedPnL)
	}
	fmt.Println()

	// Test 7: Get Leverage
	fmt.Println("7. Testing GetLeverage for BTCUSDT...")
	leverage, err := client.GetLeverage(ctx, "BTCUSDT")
	if err != nil {
		log.Printf("❌ Failed to get leverage for BTCUSDT: %v", err)
	} else {
		fmt.Printf("✅ BTCUSDT leverage: %dx\n", leverage)
	}
	fmt.Println()

	// Cleanup
	if err := client.Close(); err != nil {
		log.Printf("Failed to close client: %v", err)
	}

	fmt.Println("=== Testing completed ===")
	fmt.Println()
	fmt.Println("If you see ✅ marks, the banexg SDK integration is working correctly!")
	fmt.Println("If you see ❌ marks, there might be API key issues or network problems.")
}
