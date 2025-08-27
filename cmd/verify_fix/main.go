package main

import (
	"context"
	"fmt"
	"log"

	"qcat/internal/config"
	"qcat/internal/database"
)

func main() {
	log.Println("Position Fix Verification Tool - Starting...")

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

	fmt.Println("=== Position Fix Verification Report ===")
	fmt.Println()

	// 1. Check total positions
	var totalPositions int
	err = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM positions").Scan(&totalPositions)
	if err != nil {
		log.Fatalf("Failed to count total positions: %v", err)
	}
	fmt.Printf("1. Total positions in database: %d\n", totalPositions)

	// 2. Check active positions
	var activePositions int
	err = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM positions WHERE status IN ('open', 'active') AND size != 0").Scan(&activePositions)
	if err != nil {
		log.Fatalf("Failed to count active positions: %v", err)
	}
	fmt.Printf("2. Active positions (non-zero size): %d\n", activePositions)

	// 3. Check for duplicates
	var duplicateGroups int
	var totalDuplicates int
	err = db.DB.QueryRowContext(ctx, `
		SELECT 
			COUNT(*) as groups,
			COALESCE(SUM(cnt - 1), 0) as total_duplicates
		FROM (
			SELECT strategy_id, symbol, COUNT(*) as cnt
			FROM positions 
			WHERE status IN ('open', 'active')
			GROUP BY strategy_id, symbol
			HAVING COUNT(*) > 1
		) duplicates
	`).Scan(&duplicateGroups, &totalDuplicates)
	if err != nil {
		log.Fatalf("Failed to check duplicates: %v", err)
	}
	fmt.Printf("3. Duplicate groups: %d\n", duplicateGroups)
	fmt.Printf("4. Total duplicate records: %d\n", totalDuplicates)

	// 4. Check constraint existence
	var constraintExists bool
	err = db.DB.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.table_constraints 
			WHERE constraint_name = 'positions_strategy_symbol_unique' 
			AND table_name = 'positions'
		)
	`).Scan(&constraintExists)
	if err != nil {
		log.Fatalf("Failed to check constraint: %v", err)
	}
	fmt.Printf("5. Unique constraint exists: %t\n", constraintExists)

	// 5. Show current active positions
	fmt.Println("\n=== Current Active Positions ===")
	rows, err := db.DB.QueryContext(ctx, `
		SELECT 
			COALESCE(strategy_id::text, 'NULL') as strategy_id,
			symbol, 
			side, 
			size, 
			entry_price, 
			status,
			created_at
		FROM positions 
		WHERE status IN ('open', 'active') AND size != 0
		ORDER BY created_at DESC
		LIMIT 20
	`)
	if err != nil {
		log.Printf("Failed to query active positions: %v", err)
	} else {
		defer rows.Close()
		count := 0
		for rows.Next() {
			var strategyID, symbol, side, status string
			var size, entryPrice float64
			var createdAt string
			
			if err := rows.Scan(&strategyID, &symbol, &side, &size, &entryPrice, &status, &createdAt); err != nil {
				log.Printf("Failed to scan position: %v", err)
				continue
			}
			
			count++
			fmt.Printf("  %d. %s %s %.8f @ %.8f [%s] (Strategy: %s, Created: %s)\n", 
				count, symbol, side, size, entryPrice, status, strategyID, createdAt[:19])
		}
		
		if count == 0 {
			fmt.Println("  No active positions found")
		}
	}

	// 6. Check if there are any problematic patterns
	fmt.Println("\n=== Potential Issues Check ===")
	
	// Check for positions with same symbol but different strategies
	var multiStrategySymbols int
	err = db.DB.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT symbol)
		FROM (
			SELECT symbol, COUNT(DISTINCT strategy_id) as strategy_count
			FROM positions 
			WHERE status IN ('open', 'active') AND size != 0
			GROUP BY symbol
			HAVING COUNT(DISTINCT strategy_id) > 1
		) multi_strategy
	`).Scan(&multiStrategySymbols)
	if err != nil {
		log.Printf("Failed to check multi-strategy symbols: %v", err)
	} else {
		fmt.Printf("6. Symbols with multiple strategies: %d\n", multiStrategySymbols)
		if multiStrategySymbols > 0 {
			fmt.Println("   ⚠️  Warning: Some symbols have positions across multiple strategies")
		}
	}

	// 7. Summary and recommendations
	fmt.Println("\n=== Summary ===")
	if totalDuplicates == 0 && constraintExists {
		fmt.Println("✅ SUCCESS: No duplicates found and constraint is in place")
		fmt.Println("✅ The position management system is now protected against duplicates")
	} else {
		if totalDuplicates > 0 {
			fmt.Printf("❌ ISSUE: Still have %d duplicate records\n", totalDuplicates)
			fmt.Println("   Recommendation: Run the cleanup program again")
		}
		if !constraintExists {
			fmt.Println("❌ ISSUE: Unique constraint is missing")
			fmt.Println("   Recommendation: Run the migration script")
		}
	}

	fmt.Println("\n=== Next Steps ===")
	if totalDuplicates > 0 {
		fmt.Println("1. Run cleanup: go run cmd/cleanup_positions/main.go")
	}
	if !constraintExists {
		fmt.Println("2. Run migration: psql -h localhost -U postgres -d qcat -f migrations/fix_position_duplicates.sql")
	}
	if totalDuplicates == 0 && constraintExists {
		fmt.Println("1. ✅ System is ready - no further action needed")
		fmt.Println("2. Monitor logs for any 'Position inserted/updated' messages")
		fmt.Println("3. The system will now prevent duplicate positions automatically")
	}

	fmt.Println("\nVerification completed!")
}
