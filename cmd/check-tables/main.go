package main

import (
	"fmt"
	"log"

	"qcat/internal/config"
	"qcat/internal/database"
)

func main() {
	log.Println("🔍 Checking database tables...")

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

	// Check if open_interest table exists
	var exists bool
	err = db.DB.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'open_interest'
		)
	`).Scan(&exists)

	if err != nil {
		log.Fatalf("Failed to check table existence: %v", err)
	}

	fmt.Printf("open_interest table exists: %v\n", exists)

	if !exists {
		log.Println("Creating open_interest table...")
		_, err = db.DB.Exec(`
			CREATE TABLE open_interest (
				id BIGSERIAL PRIMARY KEY,
				symbol VARCHAR(20) NOT NULL,
				value DECIMAL(20,8) NOT NULL,
				notional DECIMAL(20,8) NOT NULL,
				timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
				created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
			)
		`)
		if err != nil {
			log.Fatalf("Failed to create open_interest table: %v", err)
		}

		// Create indexes
		_, err = db.DB.Exec(`CREATE INDEX IF NOT EXISTS idx_open_interest_symbol ON open_interest(symbol)`)
		if err != nil {
			log.Printf("Warning: Failed to create symbol index: %v", err)
		}

		_, err = db.DB.Exec(`CREATE INDEX IF NOT EXISTS idx_open_interest_timestamp ON open_interest(timestamp)`)
		if err != nil {
			log.Printf("Warning: Failed to create timestamp index: %v", err)
		}

		_, err = db.DB.Exec(`CREATE INDEX IF NOT EXISTS idx_open_interest_symbol_timestamp ON open_interest(symbol, timestamp)`)
		if err != nil {
			log.Printf("Warning: Failed to create composite index: %v", err)
		}

		log.Println("✅ open_interest table created successfully")
	}

	// Check if optimization_results table exists
	var optTableExists bool
	err = db.DB.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = 'optimization_results'
		)
	`).Scan(&optTableExists)

	if err != nil {
		log.Printf("Warning: Failed to check optimization_results table: %v", err)
	} else {
		fmt.Printf("optimization_results table exists: %v\n", optTableExists)

		if !optTableExists {
			log.Println("Creating optimization_results table...")
			_, err = db.DB.Exec(`
				CREATE TABLE optimization_results (
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
				// Create indexes
				db.DB.Exec(`CREATE INDEX IF NOT EXISTS idx_optimization_results_task_id ON optimization_results(task_id)`)
				db.DB.Exec(`CREATE INDEX IF NOT EXISTS idx_optimization_results_strategy_id ON optimization_results(strategy_id)`)
				db.DB.Exec(`CREATE INDEX IF NOT EXISTS idx_optimization_results_status ON optimization_results(status)`)
				db.DB.Exec(`CREATE INDEX IF NOT EXISTS idx_optimization_results_score ON optimization_results(optimization_score DESC)`)
				log.Println("✅ optimization_results table created successfully")
			}
		} else {
			// Check if optimization_score field exists
			var hasOptScore bool
			err = db.DB.QueryRow(`
				SELECT EXISTS (
					SELECT FROM information_schema.columns
					WHERE table_name = 'optimization_results'
					AND column_name = 'optimization_score'
				)
			`).Scan(&hasOptScore)

			if err != nil {
				log.Printf("Warning: Failed to check optimization_score field: %v", err)
			} else {
				fmt.Printf("optimization_results.optimization_score field exists: %v\n", hasOptScore)

				if !hasOptScore {
					log.Println("Adding optimization_score field...")
					_, err = db.DB.Exec(`ALTER TABLE optimization_results ADD COLUMN optimization_score DECIMAL(10,6) DEFAULT 0`)
					if err != nil {
						log.Printf("Warning: Failed to add optimization_score field: %v", err)
					} else {
						log.Println("✅ optimization_score field added successfully")
					}
				}
			}
		}
	}

	log.Println("✅ Database table check completed")
}
