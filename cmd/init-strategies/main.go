package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	// 数据库连接参数
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbName := getEnv("DB_NAME", "qcat")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "")

	// 构建连接字符串
	connStr := fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbName)

	if dbPassword != "" {
		connStr += " password=" + dbPassword
	}

	// 连接数据库
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	fmt.Println("🔧 开始修复策略数据...")

	// 1. 添加缺失的字段
	fmt.Println("📊 添加缺失的字段...")
	alterQueries := []string{
		"ALTER TABLE strategies ADD COLUMN IF NOT EXISTS is_running BOOLEAN DEFAULT false",
		"ALTER TABLE strategies ADD COLUMN IF NOT EXISTS enabled BOOLEAN DEFAULT true",
		"ALTER TABLE strategies ADD COLUMN IF NOT EXISTS parameters JSONB DEFAULT '{}'",
	}

	for _, query := range alterQueries {
		if _, err := db.Exec(query); err != nil {
			log.Printf("Warning: %v", err)
		}
	}

	// 2. 清理现有测试数据
	fmt.Println("🧹 清理现有测试数据...")
	_, err = db.Exec(`DELETE FROM strategies WHERE name IN (
		'BTC动量策略', 'ETH均值回归策略', 'SOL趋势跟踪策略', 
		'ADA网格交易策略', 'MATIC波段交易策略'
	)`)
	if err != nil {
		log.Printf("Warning: Failed to clean existing data: %v", err)
	}

	// 3. 插入测试策略数据
	fmt.Println("📝 插入测试策略数据...")
	strategies := []struct {
		name         string
		description  string
		strategyType string
		status       string
		isRunning    bool
		enabled      bool
		parameters   string
	}{
		{"BTC动量策略", "基于比特币价格动量的交易策略，使用移动平均线和RSI指标", "momentum", "active", true, true, `{"symbol": "BTCUSDT", "timeframe": "1h", "ma_short": 10, "ma_long": 30}`},
		{"ETH均值回归策略", "以太坊均值回归策略，利用价格偏离均值时的回归特性", "mean_reversion", "active", false, true, `{"symbol": "ETHUSDT", "timeframe": "4h", "lookback": 20}`},
		{"SOL趋势跟踪策略", "Solana趋势跟踪策略，使用布林带和MACD指标", "trend_following", "inactive", false, false, `{"symbol": "SOLUSDT", "timeframe": "1h", "bb_period": 20}`},
		{"ADA网格交易策略", "Cardano网格交易策略，在震荡市场中获取收益", "grid_trading", "active", false, true, `{"symbol": "ADAUSDT", "timeframe": "15m", "grid_levels": 10}`},
		{"MATIC波段交易策略", "Polygon波段交易策略，捕捉中期价格波动", "swing_trading", "active", true, true, `{"symbol": "MATICUSDT", "timeframe": "1d", "swing_period": 14}`},
	}

	insertQuery := `
		INSERT INTO strategies (id, name, description, type, status, is_running, enabled, parameters, created_at, updated_at) 
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	now := time.Now()
	for _, strategy := range strategies {
		_, err := db.Exec(insertQuery,
			strategy.name, strategy.description, strategy.strategyType, strategy.status,
			strategy.isRunning, strategy.enabled, strategy.parameters, now, now)
		if err != nil {
			log.Printf("Failed to insert strategy %s: %v", strategy.name, err)
		} else {
			fmt.Printf("✅ 插入策略: %s\n", strategy.name)
		}
	}

	// 4. 验证数据
	fmt.Println("\n📊 验证数据插入结果:")
	rows, err := db.Query(`
		SELECT 
			name, 
			type, 
			status, 
			is_running, 
			enabled,
			CASE 
				WHEN is_running AND enabled THEN 'running'
				WHEN enabled THEN 'stopped'
				ELSE 'disabled'
			END as runtime_status
		FROM strategies 
		WHERE name IN (
			'BTC动量策略', 'ETH均值回归策略', 'SOL趋势跟踪策略', 
			'ADA网格交易策略', 'MATIC波段交易策略'
		)
		ORDER BY name
	`)
	if err != nil {
		log.Fatal("Failed to verify data:", err)
	}
	defer rows.Close()

	fmt.Printf("%-20s %-15s %-10s %-10s %-8s %-15s\n", "Name", "Type", "Status", "Running", "Enabled", "Runtime Status")
	fmt.Println(strings.Repeat("-", 90))

	for rows.Next() {
		var name, strategyType, status, runtimeStatus string
		var isRunning, enabled bool
		if err := rows.Scan(&name, &strategyType, &status, &isRunning, &enabled, &runtimeStatus); err != nil {
			log.Printf("Failed to scan row: %v", err)
			continue
		}
		fmt.Printf("%-20s %-15s %-10s %-10t %-8t %-15s\n",
			name, strategyType, status, isRunning, enabled, runtimeStatus)
	}

	fmt.Println("\n✅ 策略数据修复完成！")
	fmt.Println("🎯 现在可以测试分享结果页面的策略选择功能了")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
