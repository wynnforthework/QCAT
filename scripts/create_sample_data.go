package main

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	_ "github.com/lib/pq"
)

func main() {
	// 数据库连接配置
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "qcat_user")
	dbPassword := getEnv("DB_PASSWORD", "qcat_password")
	dbName := getEnv("DB_NAME", "qcat")

	// 构建连接字符串
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	// 连接数据库
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Connected to database successfully")

	// 读取SQL文件
	sqlFile := "create_sample_trades.sql"
	if len(os.Args) > 1 {
		sqlFile = os.Args[1]
	}

	// 获取脚本目录
	scriptDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatalf("Failed to get script directory: %v", err)
	}

	sqlPath := filepath.Join(scriptDir, sqlFile)
	sqlContent, err := ioutil.ReadFile(sqlPath)
	if err != nil {
		log.Fatalf("Failed to read SQL file %s: %v", sqlPath, err)
	}

	log.Printf("Executing SQL from file: %s", sqlPath)

	// 执行SQL
	ctx := context.Background()
	_, err = db.ExecContext(ctx, string(sqlContent))
	if err != nil {
		log.Fatalf("Failed to execute SQL: %v", err)
	}

	log.Println("Sample data created successfully!")

	// 查询并显示结果
	showDataStats(db)
}

func showDataStats(db *sql.DB) {
	log.Println("\n=== Data Statistics ===")

	// 查询策略数量
	var strategyCount int
	err := db.QueryRow("SELECT COUNT(*) FROM strategies").Scan(&strategyCount)
	if err != nil {
		log.Printf("Failed to query strategies count: %v", err)
	} else {
		log.Printf("Strategies: %d", strategyCount)
	}

	// 查询交易数量
	var tradeCount int
	err = db.QueryRow("SELECT COUNT(*) FROM trades").Scan(&tradeCount)
	if err != nil {
		log.Printf("Failed to query trades count: %v", err)
	} else {
		log.Printf("Trades: %d", tradeCount)
	}

	// 查询订单数量（如果表存在）
	var orderCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM orders 
		WHERE EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'orders')
	`).Scan(&orderCount)
	if err == nil {
		log.Printf("Orders: %d", orderCount)
	}

	// 显示最近的交易记录
	log.Println("\n=== Recent Trades ===")
	rows, err := db.Query(`
		SELECT t.symbol, t.side, t.size, t.price, s.name, t.created_at
		FROM trades t
		JOIN strategies s ON t.strategy_id = s.id
		ORDER BY t.created_at DESC
		LIMIT 5
	`)
	if err != nil {
		log.Printf("Failed to query recent trades: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var symbol, side, strategyName string
		var size, price float64
		var createdAt string

		err := rows.Scan(&symbol, &side, &size, &price, &strategyName, &createdAt)
		if err != nil {
			log.Printf("Failed to scan trade row: %v", err)
			continue
		}

		log.Printf("%s: %s %.6f %s @ %.2f (%s)", 
			createdAt[:19], side, size, symbol, price, strategyName)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
