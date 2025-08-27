package main

import (
	"database/sql"
	"fmt"
	"log"

	"qcat/internal/config"
	"qcat/internal/database"
)

func main() {
	// Load configuration using the same method as the migration tool
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database using the same method as the migration tool
	dbConfig := &database.Config{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		DBName:          cfg.Database.DBName,
		SSLMode:         cfg.Database.SSLMode,
		MaxOpen:         cfg.Database.MaxOpen,
		MaxIdle:         cfg.Database.MaxIdle,
		Timeout:         cfg.Database.Timeout,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.Database.ConnMaxIdleTime,
	}

	db, err := database.NewConnection(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("✅ Connected to database successfully")
	fmt.Println("🔍 Verifying schema fixes for missing fields...")
	fmt.Println()

	// Check 1: Verify transfer_count field in fund_protection_history table
	fmt.Println("1. Checking fund_protection_history table...")
	if tableExists(db.DB, "fund_protection_history") {
		fmt.Println("   ✅ fund_protection_history table exists")

		// Check transfer_count field
		if columnExists(db.DB, "fund_protection_history", "transfer_count") {
			fmt.Println("   ✅ transfer_count column exists")
		} else {
			fmt.Println("   ❌ transfer_count column missing")
		}

		// Check success_rate field
		if columnExists(db.DB, "fund_protection_history", "success_rate") {
			fmt.Println("   ✅ success_rate column exists")
		} else {
			fmt.Println("   ❌ success_rate column missing")
		}

		// Check expected_risk_reduction field
		if columnExists(db.DB, "fund_protection_history", "expected_risk_reduction") {
			fmt.Println("   ✅ expected_risk_reduction column exists")
		} else {
			fmt.Println("   ❌ expected_risk_reduction column missing")
		}
	} else {
		fmt.Println("   ❌ fund_protection_history table missing")
	}

	// Check 2: Verify test_duration field in strategy_onboarding table
	fmt.Println("\n2. Checking strategy_onboarding table...")
	if tableExists(db.DB, "strategy_onboarding") {
		fmt.Println("   ✅ strategy_onboarding table exists")

		// Check test_duration field
		if columnExists(db.DB, "strategy_onboarding", "test_duration") {
			fmt.Println("   ✅ test_duration column exists")
		} else {
			fmt.Println("   ❌ test_duration column missing")
		}

		// Check deploy_threshold field
		if columnExists(db.DB, "strategy_onboarding", "deploy_threshold") {
			fmt.Println("   ✅ deploy_threshold column exists")
		} else {
			fmt.Println("   ❌ deploy_threshold column missing")
		}

		// Check parameters field
		if columnExists(db.DB, "strategy_onboarding", "parameters") {
			fmt.Println("   ✅ parameters column exists")
		} else {
			fmt.Println("   ❌ parameters column missing")
		}

		// Check progress field
		if columnExists(db.DB, "strategy_onboarding", "progress") {
			fmt.Println("   ✅ progress column exists")
		} else {
			fmt.Println("   ❌ progress column missing")
		}

		// Check current_stage field
		if columnExists(db.DB, "strategy_onboarding", "current_stage") {
			fmt.Println("   ✅ current_stage column exists")
		} else {
			fmt.Println("   ❌ current_stage column missing")
		}
	} else {
		fmt.Println("   ❌ strategy_onboarding table missing")
	}

	// Check 3: Test insert operations to verify the fixes work
	fmt.Println("\n3. Testing insert operations...")

	// Test fund_protection_history insert
	testFundProtectionInsert(db.DB)

	// Test strategy_onboarding insert
	testStrategyOnboardingInsert(db.DB)

	fmt.Println("\n🎉 Schema verification completed!")
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

func testFundProtectionInsert(db *sql.DB) {
	fmt.Println("   Testing fund_protection_history insert...")

	query := `
		INSERT INTO fund_protection_history (
			target_distribution, risk_parameters, transfer_count,
			success_rate, expected_risk_reduction, created_at
		) VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id
	`

	var id string
	err := db.QueryRow(query,
		`{"USDT": 0.5, "BTC": 0.3, "ETH": 0.2}`,
		`{"max_loss": 0.05}`,
		3,
		0.8500,
		0.125000,
	).Scan(&id)

	if err != nil {
		fmt.Printf("   ❌ fund_protection_history insert failed: %v\n", err)
	} else {
		fmt.Printf("   ✅ fund_protection_history insert successful (ID: %s)\n", id)

		// Clean up test data
		_, _ = db.Exec("DELETE FROM fund_protection_history WHERE id = $1", id)
	}
}

func testStrategyOnboardingInsert(db *sql.DB) {
	fmt.Println("   Testing strategy_onboarding insert...")

	query := `
		INSERT INTO strategy_onboarding (
			request_id, symbols, max_strategies, test_duration,
			risk_level, auto_deploy, deploy_threshold, parameters,
			status, progress, current_stage, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`

	var id string
	err := db.QueryRow(query,
		"test-request-123",
		`["BTCUSDT", "ETHUSDT"]`,
		5,
		"7 days",
		"medium",
		false,
		0.0200,
		`{"risk_level": 1}`,
		"queued",
		0.0,
		"等待处理",
		"NOW()",
	).Scan(&id)

	if err != nil {
		fmt.Printf("   ❌ strategy_onboarding insert failed: %v\n", err)
	} else {
		fmt.Printf("   ✅ strategy_onboarding insert successful (ID: %s)\n", id)

		// Clean up test data
		_, _ = db.Exec("DELETE FROM strategy_onboarding WHERE id = $1", id)
	}
}
