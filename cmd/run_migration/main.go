package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"

	"qcat/internal/config"
	"qcat/internal/database"
)

func main() {
	log.Println("Database Migration Tool - Starting...")

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

	// Read migration file
	migrationSQL, err := ioutil.ReadFile("migrations/fix_position_duplicates.sql")
	if err != nil {
		log.Fatalf("Failed to read migration file: %v", err)
	}

	fmt.Println("=== Running Position Duplicates Fix Migration ===")
	fmt.Println()

	// Execute migration
	ctx := context.Background()
	_, err = db.DB.ExecContext(ctx, string(migrationSQL))
	if err != nil {
		log.Fatalf("Failed to execute migration: %v", err)
	}

	fmt.Println("✅ Migration executed successfully!")
	fmt.Println()

	// Verify the migration worked
	fmt.Println("=== Verification ===")

	// Check constraint
	var constraintExists bool
	err = db.DB.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.table_constraints 
			WHERE constraint_name = 'positions_strategy_symbol_unique' 
			AND table_name = 'positions'
		)
	`).Scan(&constraintExists)
	if err != nil {
		log.Printf("Failed to check constraint: %v", err)
	} else {
		fmt.Printf("Unique constraint exists: %t\n", constraintExists)
	}

	// Check positions count
	var totalPositions int
	err = db.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM positions WHERE status IN ('open', 'active')").Scan(&totalPositions)
	if err != nil {
		log.Printf("Failed to count positions: %v", err)
	} else {
		fmt.Printf("Active positions: %d\n", totalPositions)
	}

	// Check for duplicates
	var duplicateCount int
	err = db.DB.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(cnt - 1), 0)
		FROM (
			SELECT strategy_id, symbol, COUNT(*) as cnt
			FROM positions 
			WHERE status IN ('open', 'active')
			GROUP BY strategy_id, symbol
			HAVING COUNT(*) > 1
		) duplicates
	`).Scan(&duplicateCount)
	if err != nil {
		log.Printf("Failed to check duplicates: %v", err)
	} else {
		fmt.Printf("Remaining duplicates: %d\n", duplicateCount)
	}

	fmt.Println()
	if constraintExists && duplicateCount == 0 {
		fmt.Println("🎉 SUCCESS: Migration completed successfully!")
		fmt.Println("   - Unique constraint is in place")
		fmt.Println("   - No duplicate positions remain")
		fmt.Println("   - Future duplicates will be prevented automatically")
	} else {
		fmt.Println("⚠️  WARNING: Migration may not have completed fully")
		if !constraintExists {
			fmt.Println("   - Unique constraint was not created")
		}
		if duplicateCount > 0 {
			fmt.Printf("   - %d duplicate positions still exist\n", duplicateCount)
		}
	}

	fmt.Println("\nMigration completed!")
}
