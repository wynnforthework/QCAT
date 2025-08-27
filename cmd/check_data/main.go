package main

import (
	"context"
	"fmt"
	"log"

	"qcat/internal/config"
	"qcat/internal/database"
)

func main() {
	log.Println("Checking database data...")

	// Load configuration
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	dbConfig := &database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
	}

	db, err := database.NewConnection(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Check trades table
	var tradesCount int
	err = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM trades").Scan(&tradesCount)
	if err != nil {
		log.Printf("Failed to count trades: %v", err)
	} else {
		fmt.Printf("Total trades in database: %d\n", tradesCount)
	}

	// Check positions table
	var positionsCount int
	err = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM positions").Scan(&positionsCount)
	if err != nil {
		log.Printf("Failed to count positions: %v", err)
	} else {
		fmt.Printf("Total positions in database: %d\n", positionsCount)
	}

	// Check open positions
	var openPositionsCount int
	err = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM positions WHERE status IN ('open', 'active') AND size != 0").Scan(&openPositionsCount)
	if err != nil {
		log.Printf("Failed to count open positions: %v", err)
	} else {
		fmt.Printf("Open positions in database: %d\n", openPositionsCount)
	}

	// Check orders table
	var ordersCount int
	err = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM orders").Scan(&ordersCount)
	if err != nil {
		log.Printf("Failed to count orders: %v", err)
	} else {
		fmt.Printf("Total orders in database: %d\n", ordersCount)
	}

	// Show sample trades data
	fmt.Println("\n=== Sample Trades Data ===")
	rows, err := db.DB.QueryContext(ctx, `
		SELECT symbol, side, size, price, created_at 
		FROM trades 
		ORDER BY created_at DESC 
		LIMIT 10
	`)
	if err != nil {
		log.Printf("Failed to query sample trades: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var symbol, side string
			var size, price float64
			var createdAt string
			if err := rows.Scan(&symbol, &side, &size, &price, &createdAt); err != nil {
				log.Printf("Failed to scan trade: %v", err)
				continue
			}
			fmt.Printf("Trade: %s %s %.8f @ %.8f (%s)\n", symbol, side, size, price, createdAt)
		}
	}

	// Show sample positions data
	fmt.Println("\n=== Sample Positions Data ===")
	rows, err = db.DB.QueryContext(ctx, `
		SELECT symbol, side, size, entry_price, status, created_at 
		FROM positions 
		ORDER BY created_at DESC 
		LIMIT 10
	`)
	if err != nil {
		log.Printf("Failed to query sample positions: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var symbol, side, status string
			var size, entryPrice float64
			var createdAt string
			if err := rows.Scan(&symbol, &side, &size, &entryPrice, &status, &createdAt); err != nil {
				log.Printf("Failed to scan position: %v", err)
				continue
			}
			fmt.Printf("Position: %s %s %.8f @ %.8f [%s] (%s)\n", symbol, side, size, entryPrice, status, createdAt)
		}
	}

	// Check if there are any market_data entries (might be causing confusion)
	var marketDataCount int
	err = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM market_data").Scan(&marketDataCount)
	if err != nil {
		log.Printf("Failed to count market_data: %v", err)
	} else {
		fmt.Printf("\nMarket data entries: %d\n", marketDataCount)
	}

	fmt.Println("\n=== Summary ===")
	fmt.Printf("- Trades (历史交易记录): %d\n", tradesCount)
	fmt.Printf("- Positions (持仓记录): %d\n", positionsCount)
	fmt.Printf("- Open Positions (当前持仓): %d\n", openPositionsCount)
	fmt.Printf("- Orders (订单记录): %d\n", ordersCount)
	fmt.Printf("- Market Data (市场数据): %d\n", marketDataCount)

	if tradesCount > 50000 {
		fmt.Println("\n⚠️  WARNING: 发现大量交易记录！")
		fmt.Println("前端显示的10万条数据可能来自trades表（历史交易记录）")
		fmt.Println("而实际当前持仓只有", openPositionsCount, "个")
	}
}
