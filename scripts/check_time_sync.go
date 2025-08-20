package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"qcat/internal/config"
	"qcat/internal/exchange"
	"qcat/internal/exchange/binance"
)

func main() {
	fmt.Println("⏰ Binance Server Time Synchronization Check")
	fmt.Println("============================================")

	// Load configuration
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create exchange config
	exchangeConfig := &exchange.ExchangeConfig{
		Name:      cfg.Exchange.Name,
		APIKey:    cfg.Exchange.APIKey,
		APISecret: cfg.Exchange.APISecret,
		TestNet:   cfg.Exchange.TestNet,
		BaseURL:   cfg.Exchange.BaseURL,
	}

	// Create rate limiter
	rateLimiter := exchange.NewRateLimiter(nil, 100*time.Millisecond)

	// Create client
	client := binance.NewClient(exchangeConfig, rateLimiter)

	// Perform multiple time checks
	fmt.Println("\n📊 Time Synchronization Analysis:")
	fmt.Println("==================================")

	var totalOffset int64
	var measurements []int64
	numChecks := 5

	for i := 0; i < numChecks; i++ {
		// Record local time before request
		localTimeBefore := time.Now()
		
		// Get server time
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		serverTime, err := client.GetServerTime(ctx)
		cancel()
		
		// Record local time after request
		localTimeAfter := time.Now()
		
		if err != nil {
			fmt.Printf("❌ Check %d failed: %v\n", i+1, err)
			continue
		}

		// Calculate network latency (round trip time / 2)
		networkLatency := localTimeAfter.Sub(localTimeBefore) / 2
		
		// Estimate the time when server responded (accounting for network latency)
		estimatedRequestTime := localTimeBefore.Add(networkLatency)
		
		// Calculate time offset
		offset := serverTime.Sub(estimatedRequestTime)
		offsetMs := offset.Milliseconds()
		
		measurements = append(measurements, offsetMs)
		totalOffset += offsetMs

		fmt.Printf("📋 Check %d:\n", i+1)
		fmt.Printf("   Local Time (before): %s\n", localTimeBefore.Format("2006-01-02 15:04:05.000"))
		fmt.Printf("   Server Time:         %s\n", serverTime.Format("2006-01-02 15:04:05.000"))
		fmt.Printf("   Local Time (after):  %s\n", localTimeAfter.Format("2006-01-02 15:04:05.000"))
		fmt.Printf("   Network Latency:     %v\n", networkLatency)
		fmt.Printf("   Time Offset:         %d ms\n", offsetMs)
		
		if math.Abs(float64(offsetMs)) > 5000 {
			fmt.Printf("   ⚠️  WARNING: Large time offset detected!\n")
		} else if math.Abs(float64(offsetMs)) > 1000 {
			fmt.Printf("   ⚠️  CAUTION: Moderate time offset\n")
		} else {
			fmt.Printf("   ✅ Time offset within acceptable range\n")
		}
		fmt.Println()

		// Wait a bit between checks
		if i < numChecks-1 {
			time.Sleep(1 * time.Second)
		}
	}

	if len(measurements) == 0 {
		fmt.Println("❌ No successful time measurements obtained")
		return
	}

	// Calculate statistics
	avgOffset := totalOffset / int64(len(measurements))
	
	// Calculate standard deviation
	var variance float64
	for _, offset := range measurements {
		diff := float64(offset - avgOffset)
		variance += diff * diff
	}
	variance /= float64(len(measurements))
	stdDev := math.Sqrt(variance)

	// Find min and max
	minOffset := measurements[0]
	maxOffset := measurements[0]
	for _, offset := range measurements {
		if offset < minOffset {
			minOffset = offset
		}
		if offset > maxOffset {
			maxOffset = offset
		}
	}

	fmt.Println("📈 Statistical Analysis:")
	fmt.Println("========================")
	fmt.Printf("Number of measurements: %d\n", len(measurements))
	fmt.Printf("Average offset:         %d ms\n", avgOffset)
	fmt.Printf("Standard deviation:     %.2f ms\n", stdDev)
	fmt.Printf("Minimum offset:         %d ms\n", minOffset)
	fmt.Printf("Maximum offset:         %d ms\n", maxOffset)
	fmt.Printf("Offset range:           %d ms\n", maxOffset-minOffset)

	fmt.Println("\n🔍 Analysis Results:")
	fmt.Println("===================")

	absAvgOffset := math.Abs(float64(avgOffset))
	
	if absAvgOffset <= 1000 {
		fmt.Println("✅ GOOD: Time synchronization is within acceptable range")
		fmt.Println("   Your system time is well synchronized with Binance servers")
	} else if absAvgOffset <= 5000 {
		fmt.Println("⚠️  WARNING: Moderate time offset detected")
		fmt.Println("   This may cause occasional timestamp errors")
		fmt.Println("   Consider synchronizing your system clock")
	} else {
		fmt.Println("❌ CRITICAL: Large time offset detected")
		fmt.Println("   This will likely cause timestamp errors (-1021)")
		fmt.Println("   You MUST synchronize your system clock")
	}

	fmt.Println("\n💡 Recommendations:")
	fmt.Println("===================")
	
	if absAvgOffset > 1000 {
		fmt.Println("1. Synchronize your system clock:")
		fmt.Println("   Windows: Run 'w32tm /resync' as administrator")
		fmt.Println("   Linux/Mac: Run 'sudo ntpdate -s time.nist.gov'")
		fmt.Println()
		fmt.Println("2. Enable automatic time synchronization:")
		fmt.Println("   Windows: Settings > Time & Language > Date & Time > Sync now")
		fmt.Println("   Linux: Install and configure ntp service")
		fmt.Println()
		fmt.Println("3. Check your timezone settings")
		fmt.Println()
	}

	if stdDev > 500 {
		fmt.Println("4. High time variance detected - check network stability")
		fmt.Println()
	}

	fmt.Printf("5. Current system timezone: %s\n", time.Now().Location())
	fmt.Printf("6. Binance expects UTC timestamps\n")
	
	// Show current UTC time
	utcNow := time.Now().UTC()
	fmt.Printf("7. Current UTC time: %s\n", utcNow.Format("2006-01-02 15:04:05.000"))

	fmt.Println("\n🔧 Next Steps:")
	fmt.Println("==============")
	if absAvgOffset <= 1000 {
		fmt.Println("✅ Your time sync is good. The API key issue is likely due to:")
		fmt.Println("   - Invalid API key or secret")
		fmt.Println("   - Missing 'Enable Futures' permission")
		fmt.Println("   - API key not activated")
		fmt.Println("   - IP restrictions (if enabled)")
	} else {
		fmt.Println("1. Fix time synchronization first")
		fmt.Println("2. Then test API connection again")
		fmt.Println("3. Run: go run scripts/validate_api_config.go")
	}
}
