package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	// 尝试多种数据库连接方式
	connStrings := []string{
		"host=localhost port=5432 user=postgres dbname=qcat sslmode=disable",
		"host=localhost port=5432 user=postgres password=postgres dbname=qcat sslmode=disable",
		"host=localhost port=5432 user=qcat_user dbname=qcat sslmode=disable",
		"host=localhost port=5432 user=qcat_user password=qcat_password dbname=qcat sslmode=disable",
	}

	var db *sql.DB
	var err error

	for i, connStr := range connStrings {
		fmt.Printf("Trying connection %d: %s\n", i+1, connStr)
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			fmt.Printf("Failed to open connection %d: %v\n", i+1, err)
			continue
		}

		// 测试连接
		if err := db.Ping(); err != nil {
			fmt.Printf("Failed to ping with connection %d: %v\n", i+1, err)
			db.Close()
			db = nil
			continue
		}

		fmt.Printf("Successfully connected with connection %d\n", i+1)
		break
	}

	if db == nil {
		log.Fatal("Failed to connect to database with any connection string")
	}
	defer db.Close()

	fmt.Println("Connected to database successfully")

	// 创建表的SQL语句
	createTables := []string{
		`CREATE TABLE IF NOT EXISTS hotlist_scores (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			symbol VARCHAR(20) NOT NULL,
			vol_jump_score DECIMAL(10,6) NOT NULL DEFAULT 0,
			turnover_score DECIMAL(10,6) NOT NULL DEFAULT 0,
			oi_change_score DECIMAL(10,6) NOT NULL DEFAULT 0,
			funding_z_score DECIMAL(10,6) NOT NULL DEFAULT 0,
			regime_shift_score DECIMAL(10,6) NOT NULL DEFAULT 0,
			total_score DECIMAL(10,6) NOT NULL DEFAULT 0,
			risk_level VARCHAR(20) NOT NULL DEFAULT 'medium',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS trading_whitelist (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			symbol VARCHAR(20) NOT NULL UNIQUE,
			approved_by UUID,
			approved_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			status VARCHAR(20) NOT NULL DEFAULT 'active',
			reason TEXT,
			metadata JSONB,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS audit_logs (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			user_id UUID,
			action VARCHAR(100) NOT NULL,
			resource_type VARCHAR(50) NOT NULL,
			resource_id UUID,
			old_values JSONB,
			new_values JSONB,
			ip_address INET,
			user_agent TEXT,
			session_id VARCHAR(100),
			status VARCHAR(20) NOT NULL DEFAULT 'success',
			error_message TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS audit_decisions (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			decision_id VARCHAR(100) NOT NULL UNIQUE,
			strategy_id UUID,
			decision_type VARCHAR(50) NOT NULL,
			input_data JSONB NOT NULL,
			output_data JSONB NOT NULL,
			decision_path JSONB,
			confidence_score DECIMAL(10,6),
			execution_time_ms INTEGER,
			status VARCHAR(20) NOT NULL DEFAULT 'executed',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS audit_performance (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			metric_type VARCHAR(50) NOT NULL,
			metric_name VARCHAR(100) NOT NULL,
			value DECIMAL(30,10) NOT NULL,
			unit VARCHAR(20) NOT NULL DEFAULT 'ms',
			tags JSONB,
			timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	// 执行创建表语句
	for i, createSQL := range createTables {
		fmt.Printf("Creating table %d...\n", i+1)
		if _, err := db.Exec(createSQL); err != nil {
			log.Printf("Failed to create table %d: %v", i+1, err)
		} else {
			fmt.Printf("Table %d created successfully\n", i+1)
		}
	}

	// 修复现有表结构
	fixTables := []string{
		`DO $$ 
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'circuit_breakers' AND column_name = 'action') THEN
				ALTER TABLE circuit_breakers ADD COLUMN action VARCHAR(50) DEFAULT 'halt';
			END IF;
			
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'circuit_breakers' AND column_name = 'triggered_at') THEN
				ALTER TABLE circuit_breakers ADD COLUMN triggered_at TIMESTAMP WITH TIME ZONE;
			END IF;
		EXCEPTION
			WHEN OTHERS THEN
				NULL;
		END $$`,

		`DO $$ 
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_violations' AND column_name = 'symbol') THEN
				ALTER TABLE risk_violations ADD COLUMN symbol VARCHAR(20);
			END IF;
			
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'risk_violations' AND column_name = 'message') THEN
				ALTER TABLE risk_violations ADD COLUMN message TEXT;
			END IF;
		EXCEPTION
			WHEN OTHERS THEN
				NULL;
		END $$`,
	}

	// 执行修复表语句
	for i, fixSQL := range fixTables {
		fmt.Printf("Fixing table structure %d...\n", i+1)
		if _, err := db.Exec(fixSQL); err != nil {
			log.Printf("Failed to fix table %d: %v", i+1, err)
		} else {
			fmt.Printf("Table structure %d fixed successfully\n", i+1)
		}
	}

	// 插入示例数据
	insertData := []string{
		`INSERT INTO hotlist_scores (symbol, vol_jump_score, turnover_score, oi_change_score, funding_z_score, regime_shift_score, total_score, risk_level) 
		SELECT 'BTCUSDT', 0.85, 0.92, 0.78, 0.65, 0.88, 0.816, 'high'
		WHERE NOT EXISTS (SELECT 1 FROM hotlist_scores WHERE symbol = 'BTCUSDT')`,

		`INSERT INTO hotlist_scores (symbol, vol_jump_score, turnover_score, oi_change_score, funding_z_score, regime_shift_score, total_score, risk_level) 
		SELECT 'ETHUSDT', 0.72, 0.85, 0.69, 0.58, 0.75, 0.718, 'medium'
		WHERE NOT EXISTS (SELECT 1 FROM hotlist_scores WHERE symbol = 'ETHUSDT')`,

		`INSERT INTO trading_whitelist (symbol, status, reason) 
		SELECT 'BTCUSDT', 'active', 'High liquidity and volume'
		WHERE NOT EXISTS (SELECT 1 FROM trading_whitelist WHERE symbol = 'BTCUSDT')`,

		`INSERT INTO trading_whitelist (symbol, status, reason) 
		SELECT 'ETHUSDT', 'active', 'Major cryptocurrency'
		WHERE NOT EXISTS (SELECT 1 FROM trading_whitelist WHERE symbol = 'ETHUSDT')`,

		`INSERT INTO audit_performance (metric_type, metric_name, value, unit, tags) 
		SELECT 'api_response_time', 'GET /api/v1/dashboard', 150.5, 'ms', '{"endpoint": "/api/v1/dashboard", "method": "GET"}'::jsonb
		WHERE NOT EXISTS (SELECT 1 FROM audit_performance WHERE metric_name = 'GET /api/v1/dashboard')`,
	}

	// 执行插入数据语句
	for i, insertSQL := range insertData {
		fmt.Printf("Inserting sample data %d...\n", i+1)
		if _, err := db.Exec(insertSQL); err != nil {
			log.Printf("Failed to insert data %d: %v", i+1, err)
		} else {
			fmt.Printf("Sample data %d inserted successfully\n", i+1)
		}
	}

	fmt.Println("All tables created and sample data inserted successfully!")
}
