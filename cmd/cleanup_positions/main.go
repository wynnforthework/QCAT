package main

import (
	"context"
	"fmt"
	"log"

	"qcat/internal/config"
	"qcat/internal/database"
)

func main() {
	log.Println("Position Cleanup Tool - Starting...")

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

	// Check current situation
	var totalPositions int
	err = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM positions").Scan(&totalPositions)
	if err != nil {
		log.Fatalf("Failed to count positions: %v", err)
	}

	fmt.Printf("Current total positions in database: %d\n", totalPositions)

	// Show unique positions
	fmt.Println("\n=== Unique Positions Analysis ===")
	rows, err := db.DB.QueryContext(ctx, `
		SELECT 
			symbol, 
			side, 
			size, 
			entry_price, 
			COUNT(*) as duplicate_count,
			MIN(created_at) as first_created,
			MAX(created_at) as last_created
		FROM positions 
		WHERE status IN ('open', 'active') 
		GROUP BY symbol, side, size, entry_price
		ORDER BY duplicate_count DESC
	`)
	if err != nil {
		log.Fatalf("Failed to analyze positions: %v", err)
	}
	defer rows.Close()

	uniquePositions := 0
	totalDuplicates := 0
	
	for rows.Next() {
		var symbol, side string
		var size, entryPrice float64
		var duplicateCount int
		var firstCreated, lastCreated string
		
		if err := rows.Scan(&symbol, &side, &size, &entryPrice, &duplicateCount, &firstCreated, &lastCreated); err != nil {
			log.Printf("Failed to scan position: %v", err)
			continue
		}
		
		uniquePositions++
		totalDuplicates += duplicateCount - 1 // -1 because we keep one
		
		fmt.Printf("Position: %s %s %.8f @ %.8f - Duplicates: %d (First: %s, Last: %s)\n", 
			symbol, side, size, entryPrice, duplicateCount, firstCreated[:19], lastCreated[:19])
	}

	fmt.Printf("\nSummary:\n")
	fmt.Printf("- Unique positions: %d\n", uniquePositions)
	fmt.Printf("- Total duplicates to remove: %d\n", totalDuplicates)
	fmt.Printf("- Will keep: %d positions\n", uniquePositions)

	// Confirm cleanup
	fmt.Print("\nDo you want to clean up duplicate positions? This will keep only the LATEST record for each unique position. (yes/no): ")
	var confirmation string
	fmt.Scanln(&confirmation)
	
	if confirmation != "yes" {
		log.Println("Cleanup cancelled by user")
		return
	}

	// Perform cleanup - keep only the latest record for each unique position
	log.Println("Starting cleanup...")
	
	cleanupQuery := `
		DELETE FROM positions 
		WHERE id NOT IN (
			SELECT DISTINCT ON (symbol, side, size, entry_price) id
			FROM positions 
			WHERE status IN ('open', 'active')
			ORDER BY symbol, side, size, entry_price, created_at DESC
		)
		AND status IN ('open', 'active')
	`
	
	result, err := db.DB.ExecContext(ctx, cleanupQuery)
	if err != nil {
		log.Fatalf("Failed to cleanup positions: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Could not get rows affected: %v", err)
	} else {
		fmt.Printf("Successfully removed %d duplicate positions\n", rowsAffected)
	}

	// Verify cleanup
	var remainingPositions int
	err = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM positions WHERE status IN ('open', 'active')").Scan(&remainingPositions)
	if err != nil {
		log.Printf("Failed to count remaining positions: %v", err)
	} else {
		fmt.Printf("Remaining open positions: %d\n", remainingPositions)
	}

	// Show final positions
	fmt.Println("\n=== Final Positions ===")
	rows, err = db.DB.QueryContext(ctx, `
		SELECT symbol, side, size, entry_price, created_at 
		FROM positions 
		WHERE status IN ('open', 'active')
		ORDER BY created_at DESC
	`)
	if err != nil {
		log.Printf("Failed to query final positions: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var symbol, side string
			var size, entryPrice float64
			var createdAt string
			if err := rows.Scan(&symbol, &side, &size, &entryPrice, &createdAt); err != nil {
				log.Printf("Failed to scan position: %v", err)
				continue
			}
			fmt.Printf("Position: %s %s %.8f @ %.8f (%s)\n", symbol, side, size, entryPrice, createdAt[:19])
		}
	}

	fmt.Println("\n✅ Position cleanup completed!")
	fmt.Println("Now you can run the emergency close tool to close the remaining positions.")
}
