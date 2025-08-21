package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	// Database connection parameters
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DATABASE_PASSWORD", "")
	dbname := getEnv("DB_NAME", "qcat")

	// Create connection string
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	// Connect to database
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("✅ Connected to database successfully")
	fmt.Println("🔍 Verifying database fixes...")
	fmt.Println()

	// Check 1: Verify risk_thresholds table exists
	fmt.Println("1. Checking risk_thresholds table...")
	if tableExists(db, "risk_thresholds") {
		fmt.Println("   ✅ risk_thresholds table exists")

		// Check if default data exists
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM risk_thresholds WHERE name = 'default'").Scan(&count)
		if err != nil {
			fmt.Printf("   ⚠️  Error checking default data: %v\n", err)
		} else if count > 0 {
			fmt.Println("   ✅ Default risk thresholds data exists")
		} else {
			fmt.Println("   ⚠️  Default risk thresholds data missing")
		}
	} else {
		fmt.Println("   ❌ risk_thresholds table missing")
	}

	// Check 2: Verify hedge_history table and strategy_ids column
	fmt.Println("\n2. Checking hedge_history table...")
	if tableExists(db, "hedge_history") {
		fmt.Println("   ✅ hedge_history table exists")

		// Check if strategy_ids column exists and has proper constraints
		if columnExists(db, "hedge_history", "strategy_ids") {
			fmt.Println("   ✅ strategy_ids column exists")

			// Check column type
			var dataType string
			err := db.QueryRow(`
				SELECT data_type 
				FROM information_schema.columns 
				WHERE table_name = 'hedge_history' AND column_name = 'strategy_ids'
			`).Scan(&dataType)
			if err != nil {
				fmt.Printf("   ⚠️  Error checking column type: %v\n", err)
			} else {
				fmt.Printf("   ✅ strategy_ids column type: %s\n", dataType)
			}
		} else {
			fmt.Println("   ❌ strategy_ids column missing")
		}
	} else {
		fmt.Println("   ❌ hedge_history table missing")
	}

	// Check 3: Verify market_data table and complete column
	fmt.Println("\n3. Checking market_data table...")
	if tableExists(db, "market_data") {
		fmt.Println("   ✅ market_data table exists")

		// Check if complete column exists
		if columnExists(db, "market_data", "complete") {
			fmt.Println("   ✅ complete column exists")

			// Check column type
			var dataType string
			err := db.QueryRow(`
				SELECT data_type 
				FROM information_schema.columns 
				WHERE table_name = 'market_data' AND column_name = 'complete'
			`).Scan(&dataType)
			if err != nil {
				fmt.Printf("   ⚠️  Error checking column type: %v\n", err)
			} else {
				fmt.Printf("   ✅ complete column type: %s\n", dataType)
			}
		} else {
			fmt.Println("   ❌ complete column missing")
		}
	} else {
		fmt.Println("   ❌ market_data table missing")
	}

	// Check 4: Verify optimization_history table
	fmt.Println("\n4. Checking optimization_history table...")
	if tableExists(db, "optimization_history") {
		fmt.Println("   ✅ optimization_history table exists")
	} else {
		fmt.Println("   ❌ optimization_history table missing")
	}

	fmt.Println("\n🎉 Database verification completed!")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func tableExists(db *sql.DB, tableName string) bool {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_name = $1
		)
	`
	err := db.QueryRow(query, tableName).Scan(&exists)
	return err == nil && exists
}

func columnExists(db *sql.DB, tableName, columnName string) bool {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = $1 AND column_name = $2
		)
	`
	err := db.QueryRow(query, tableName, columnName).Scan(&exists)
	return err == nil && exists
}
