package main

import (
	"fmt"
	"log"

	"qcat/internal/config"
	"qcat/internal/database"
)

func main() {
	log.Println("🔧 Fixing database tables...")

	// Load configuration
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	db, err := database.NewConnection(&database.Config{
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
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create optimization_results table
	log.Println("Creating optimization_results table...")
	_, err = db.DB.Exec(`
		CREATE TABLE IF NOT EXISTS optimization_results (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			task_id VARCHAR(100) NOT NULL UNIQUE,
			strategy_id UUID,
			parameters JSONB DEFAULT '{}',
			performance_metrics JSONB DEFAULT '{}',
			backtest_result JSONB DEFAULT '{}',
			optimization_score DECIMAL(10,6) DEFAULT 0,
			improvement_score DECIMAL(10,6) DEFAULT 0,
			objective_value DECIMAL(10,6) DEFAULT 0,
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			completed_at TIMESTAMP WITH TIME ZONE,
			error_message TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Printf("Warning: Failed to create optimization_results table: %v", err)
	} else {
		log.Println("✅ optimization_results table created")
	}

	// Create indexes for optimization_results
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_optimization_results_task_id ON optimization_results(task_id)`,
		`CREATE INDEX IF NOT EXISTS idx_optimization_results_strategy_id ON optimization_results(strategy_id)`,
		`CREATE INDEX IF NOT EXISTS idx_optimization_results_status ON optimization_results(status)`,
		`CREATE INDEX IF NOT EXISTS idx_optimization_results_score ON optimization_results(optimization_score DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_optimization_results_created_at ON optimization_results(created_at DESC)`,
	}

	for _, indexSQL := range indexes {
		_, err = db.DB.Exec(indexSQL)
		if err != nil {
			log.Printf("Warning: Failed to create index: %v", err)
		}
	}
	log.Println("✅ optimization_results indexes created")

	// Create open_interest table (if not already exists)
	log.Println("Creating open_interest table...")
	_, err = db.DB.Exec(`
		CREATE TABLE IF NOT EXISTS open_interest (
			id BIGSERIAL PRIMARY KEY,
			symbol VARCHAR(20) NOT NULL,
			value DECIMAL(20,8) NOT NULL,
			notional DECIMAL(20,8) NOT NULL,
			timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Printf("Warning: Failed to create open_interest table: %v", err)
	} else {
		log.Println("✅ open_interest table created")
	}

	// Create indexes for open_interest
	oiIndexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_open_interest_symbol ON open_interest(symbol)`,
		`CREATE INDEX IF NOT EXISTS idx_open_interest_timestamp ON open_interest(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_open_interest_symbol_timestamp ON open_interest(symbol, timestamp)`,
	}

	for _, indexSQL := range oiIndexes {
		_, err = db.DB.Exec(indexSQL)
		if err != nil {
			log.Printf("Warning: Failed to create open_interest index: %v", err)
		}
	}
	log.Println("✅ open_interest indexes created")

	// Verify tables exist
	var optExists, oiExists bool
	
	err = db.DB.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'optimization_results'
		)
	`).Scan(&optExists)
	if err != nil {
		log.Printf("Error checking optimization_results: %v", err)
	}

	err = db.DB.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'open_interest'
		)
	`).Scan(&oiExists)
	if err != nil {
		log.Printf("Error checking open_interest: %v", err)
	}

	fmt.Printf("\n📊 Table Status:\n")
	fmt.Printf("  optimization_results: %v\n", optExists)
	fmt.Printf("  open_interest: %v\n", oiExists)

	if optExists && oiExists {
		log.Println("\n🎉 All required tables are now available!")
	} else {
		log.Println("\n⚠️  Some tables are still missing")
	}
}
