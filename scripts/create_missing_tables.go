package main

import (
	"fmt"
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

	fmt.Println("Creating missing tables...")

	// 创建hotlist_scores表
	createHotlistScores := `
	CREATE TABLE IF NOT EXISTS hotlist_scores (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		symbol VARCHAR(20) NOT NULL,
		vol_jump_score DECIMAL(10,6) DEFAULT 0,
		turnover_score DECIMAL(10,6) DEFAULT 0,
		oi_change_score DECIMAL(10,6) DEFAULT 0,
		funding_z_score DECIMAL(10,6) DEFAULT 0,
		regime_shift_score DECIMAL(10,6) DEFAULT 0,
		total_score DECIMAL(10,6) DEFAULT 0,
		risk_level VARCHAR(20) DEFAULT 'medium',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(symbol)
	);`

	if _, err := db.Exec(createHotlistScores); err != nil {
		log.Printf("Failed to create hotlist_scores table: %v", err)
	} else {
		fmt.Println("✅ hotlist_scores table created")
	}

	// 创建audit_logs表
	createAuditLogs := `
	CREATE TABLE IF NOT EXISTS audit_logs (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		user_id UUID,
		action VARCHAR(100) NOT NULL,
		resource_type VARCHAR(50),
		resource_id VARCHAR(100),
		details JSONB,
		ip_address INET,
		user_agent TEXT,
		timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		success BOOLEAN DEFAULT true
	);`

	if _, err := db.Exec(createAuditLogs); err != nil {
		log.Printf("Failed to create audit_logs table: %v", err)
	} else {
		fmt.Println("✅ audit_logs table created")
	}

	// 创建索引
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_hotlist_scores_total_score ON hotlist_scores(total_score DESC);",
		"CREATE INDEX IF NOT EXISTS idx_hotlist_scores_symbol ON hotlist_scores(symbol);",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp DESC);",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_type ON audit_logs(resource_type);",
	}

	for _, indexSQL := range indexes {
		if _, err := db.Exec(indexSQL); err != nil {
			log.Printf("Failed to create index: %v", err)
		}
	}
	fmt.Println("✅ Indexes created")

	// 插入测试数据
	insertHotlistData := `
	INSERT INTO hotlist_scores (symbol, vol_jump_score, turnover_score, oi_change_score, funding_z_score, regime_shift_score, total_score, risk_level) 
	VALUES 
		('BTCUSDT', 0.85, 0.92, 0.78, 0.65, 0.88, 0.82, 'high'),
		('ETHUSDT', 0.75, 0.88, 0.82, 0.70, 0.85, 0.80, 'high'),
		('ADAUSDT', 0.65, 0.75, 0.68, 0.55, 0.72, 0.67, 'medium'),
		('SOLUSDT', 0.88, 0.85, 0.90, 0.75, 0.82, 0.84, 'high'),
		('DOTUSDT', 0.55, 0.68, 0.62, 0.48, 0.65, 0.60, 'medium')
	ON CONFLICT (symbol) DO UPDATE SET
		vol_jump_score = EXCLUDED.vol_jump_score,
		turnover_score = EXCLUDED.turnover_score,
		oi_change_score = EXCLUDED.oi_change_score,
		funding_z_score = EXCLUDED.funding_z_score,
		regime_shift_score = EXCLUDED.regime_shift_score,
		total_score = EXCLUDED.total_score,
		risk_level = EXCLUDED.risk_level,
		updated_at = CURRENT_TIMESTAMP;`

	if _, err := db.Exec(insertHotlistData); err != nil {
		log.Printf("Failed to insert hotlist data: %v", err)
	} else {
		fmt.Println("✅ Hotlist test data inserted")
	}

	insertAuditData := `
	INSERT INTO audit_logs (action, resource_type, resource_id, details, ip_address, user_agent, success) 
	VALUES 
		('login', 'user', 'admin', '{"login_method": "password"}', '127.0.0.1', 'Mozilla/5.0', true),
		('create_strategy', 'strategy', 'strategy-001', '{"name": "Test Strategy", "type": "momentum"}', '127.0.0.1', 'Mozilla/5.0', true),
		('update_portfolio', 'portfolio', 'portfolio-001', '{"action": "rebalance", "mode": "bandit"}', '127.0.0.1', 'Mozilla/5.0', true),
		('delete_strategy', 'strategy', 'strategy-002', '{"name": "Old Strategy"}', '127.0.0.1', 'Mozilla/5.0', false);`

	if _, err := db.Exec(insertAuditData); err != nil {
		log.Printf("Failed to insert audit data: %v", err)
	} else {
		fmt.Println("✅ Audit test data inserted")
	}

	fmt.Println("🎉 All missing tables and data created successfully!")
}
