package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"qcat/internal/config"
	"qcat/internal/database"

	_ "github.com/lib/pq"
)

func main() {
	// 加载配置
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 连接数据库
	dbConfig := &database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
		MaxOpen:  cfg.Database.MaxOpen,
		MaxIdle:  cfg.Database.MaxIdle,
		Timeout:  cfg.Database.Timeout,
	}

	db, err := database.NewConnection(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	fmt.Println("Connected to database successfully")

	// Read migration file
	migrationFile := "internal/database/migrations/000012_fix_missing_tables.up.sql"
	content, err := ioutil.ReadFile(migrationFile)
	if err != nil {
		log.Fatalf("Failed to read migration file: %v", err)
	}

	// Execute migration
	fmt.Println("Executing migration...")
	_, err = db.Exec(string(content))
	if err != nil {
		log.Fatalf("Failed to execute migration: %v", err)
	}

	fmt.Println("✅ Migration executed successfully!")

	// Verify tables were created
	tables := []string{
		"exchange_balances",
		"strategy_positions",
		"elimination_reports",
		"strategy_performance",
		"onboarding_reports",
	}

	fmt.Println("Verifying tables...")
	for _, table := range tables {
		var exists bool
		query := `SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = $1
		)`
		err := db.QueryRow(query, table).Scan(&exists)
		if err != nil {
			log.Printf("Failed to check table %s: %v", table, err)
			continue
		}

		if exists {
			fmt.Printf("✅ Table %s exists\n", table)
		} else {
			fmt.Printf("❌ Table %s does not exist\n", table)
		}
	}

	fmt.Println("🎉 Database migration completed!")
}
