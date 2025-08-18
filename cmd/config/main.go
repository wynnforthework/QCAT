package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"qcat/internal/config"
)

func main() {
	var (
		configPath = flag.String("config", "configs/config.yaml", "配置文件路径")
		envPath    = flag.String("env", ".env", "环境变量文件路径")
		validate   = flag.Bool("validate", false, "验证配置")
		generate   = flag.Bool("generate", false, "生成环境变量模板")
		encrypt    = flag.String("encrypt", "", "加密字符串")
		decrypt    = flag.String("decrypt", "", "解密字符串")
		help       = flag.Bool("help", false, "显示帮助信息")
	)
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	// 加密/解密功能
	if *encrypt != "" {
		encryptString(*encrypt)
		return
	}

	if *decrypt != "" {
		decryptString(*decrypt)
		return
	}

	// 生成环境变量模板
	if *generate {
		generateEnvTemplate(*envPath)
		return
	}

	// 验证配置
	if *validate {
		validateConfig(*configPath)
		return
	}

	// 默认显示帮助
	showHelp()
}

func showHelp() {
	fmt.Println("QCAT 配置管理工具")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  qcat-config [选项]")
	fmt.Println()
	fmt.Println("选项:")
	fmt.Println("  -config string")
	fmt.Println("        配置文件路径 (默认: configs/config.yaml)")
	fmt.Println("  -env string")
	fmt.Println("        环境变量文件路径 (默认: .env)")
	fmt.Println("  -validate")
	fmt.Println("        验证配置文件")
	fmt.Println("  -generate")
	fmt.Println("        生成环境变量模板")
	fmt.Println("  -encrypt string")
	fmt.Println("        加密字符串")
	fmt.Println("  -decrypt string")
	fmt.Println("        解密字符串")
	fmt.Println("  -help")
	fmt.Println("        显示帮助信息")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  qcat-config -validate")
	fmt.Println("  qcat-config -generate -env .env.example")
	fmt.Println("  qcat-config -encrypt 'my-secret-password'")
	fmt.Println("  qcat-config -decrypt 'ENC:encrypted-string'")
}

func validateConfig(configPath string) {
	fmt.Printf("正在验证配置文件: %s\n", configPath)

	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("配置文件不存在: %s", configPath)
	}

	// 加载配置
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 创建验证器
	validator := config.NewValidator(cfg)

	// 验证配置
	if err := validator.Validate(); err != nil {
		log.Fatalf("配置验证失败: %v", err)
	}

	fmt.Println("✅ 配置验证通过")
	
	// 显示配置摘要
	showConfigSummary(cfg)
}

func showConfigSummary(cfg *config.Config) {
	fmt.Println("\n配置摘要:")
	fmt.Printf("  应用名称: %s\n", cfg.App.Name)
	fmt.Printf("  版本: %s\n", cfg.App.Version)
	fmt.Printf("  环境: %s\n", cfg.App.Environment)
	fmt.Printf("  服务器端口: %d\n", cfg.Server.Port)
	fmt.Printf("  数据库: %s:%d/%s\n", cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
	fmt.Printf("  Redis: %s (启用: %t)\n", cfg.Redis.Addr, cfg.Redis.Enabled)
	fmt.Printf("  交易所: %s (测试网: %t)\n", cfg.Exchange.Name, cfg.Exchange.TestNet)
	fmt.Printf("  策略模式: %s\n", cfg.Strategy.DefaultMode)
	fmt.Printf("  优化器: %s (启用: %t)\n", cfg.Optimizer.DefaultAlgorithm, cfg.Optimizer.Enabled)
	fmt.Printf("  风险管理: %t\n", cfg.Risk.Enabled)
	fmt.Printf("  市场数据: %t\n", cfg.MarketData.Enabled)
}

func generateEnvTemplate(envPath string) {
	fmt.Printf("正在生成环境变量模板: %s\n", envPath)

	// 确保目录存在
	dir := filepath.Dir(envPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("创建目录失败: %v", err)
	}

	// 生成环境变量模板
	template := `# QCAT 环境变量配置模板
# 复制此文件为 .env 并根据实际情况修改
# 注意：此文件不应提交到版本控制系统

# =============================================================================
# 应用配置
# =============================================================================
QCAT_APP_NAME=QCAT
QCAT_APP_VERSION=2.0.0
QCAT_APP_ENVIRONMENT=development

# =============================================================================
# 服务器配置
# =============================================================================
QCAT_SERVER_PORT=8082
QCAT_SERVER_READ_TIMEOUT=30s
QCAT_SERVER_WRITE_TIMEOUT=30s

# =============================================================================
# 数据库配置
# =============================================================================
QCAT_DATABASE_HOST=localhost
QCAT_DATABASE_PORT=5432
QCAT_DATABASE_USER=postgres
QCAT_DATABASE_PASSWORD=your_secure_database_password
QCAT_DATABASE_NAME=qcat
QCAT_DATABASE_SSL_MODE=disable

# =============================================================================
# Redis配置
# =============================================================================
QCAT_REDIS_ENABLED=true
QCAT_REDIS_ADDR=localhost:6379
QCAT_REDIS_PASSWORD=your_secure_redis_password
QCAT_REDIS_DB=0
QCAT_REDIS_POOL_SIZE=20

# =============================================================================
# 交易所配置
# =============================================================================
QCAT_EXCHANGE_NAME=binance
QCAT_EXCHANGE_API_KEY=your_binance_api_key
QCAT_EXCHANGE_API_SECRET=your_binance_api_secret
QCAT_EXCHANGE_TEST_NET=true
QCAT_EXCHANGE_BASE_URL=https://api.binance.com
QCAT_EXCHANGE_WEBSOCKET_URL=wss://stream.binance.com:9443

# =============================================================================
# JWT配置
# =============================================================================
QCAT_JWT_SECRET_KEY=your_super_secure_jwt_secret_key_2024
QCAT_JWT_DURATION=24h

# =============================================================================
# 安全配置
# =============================================================================
QCAT_SECURITY_KMS_MASTER_KEY=your_kms_master_key
QCAT_SECURITY_ENCRYPTION_MASTER_KEY=your_encryption_master_key
QCAT_SECURITY_ENCRYPTION_KEY=your_encryption_key

# =============================================================================
# 加密密钥（用于环境变量加密）
# =============================================================================
QCAT_ENCRYPTION_KEY=your_encryption_key_for_env_vars

# =============================================================================
# 监控配置
# =============================================================================
QCAT_MONITORING_PROMETHEUS_ENABLED=true
QCAT_MONITORING_PROMETHEUS_PATH=/metrics

# =============================================================================
# 日志配置
# =============================================================================
QCAT_LOGGING_LEVEL=info
QCAT_LOGGING_FORMAT=json
QCAT_LOGGING_OUTPUT=file

# =============================================================================
# 策略配置
# =============================================================================
QCAT_STRATEGY_DEFAULT_MODE=paper
QCAT_STRATEGY_MAX_CONCURRENT_STRATEGIES=10
QCAT_STRATEGY_STRATEGY_TIMEOUT=300s

# =============================================================================
# 优化器配置
# =============================================================================
QCAT_OPTIMIZER_ENABLED=true
QCAT_OPTIMIZER_TIMEOUT=1800s
QCAT_OPTIMIZER_MAX_ITERATIONS=1000

# =============================================================================
# 市场数据配置
# =============================================================================
QCAT_MARKET_DATA_ENABLED=true
QCAT_MARKET_DATA_CACHE_TTL=60s
QCAT_MARKET_DATA_BATCH_SIZE=1000

# =============================================================================
# 风险管理配置
# =============================================================================
QCAT_RISK_ENABLED=true
QCAT_RISK_CHECK_INTERVAL=5s
QCAT_RISK_MAX_POSITION_SIZE=100000
QCAT_RISK_MAX_LEVERAGE=10

# =============================================================================
# 缓存配置
# =============================================================================
QCAT_CACHE_TTL=3600s
QCAT_CACHE_MAX_SIZE=1000
QCAT_CACHE_CLEANUP_INTERVAL=300s

# =============================================================================
# 网络配置
# =============================================================================
QCAT_NETWORK_MAX_RETRIES=10
QCAT_NETWORK_INITIAL_DELAY=1s
QCAT_NETWORK_MAX_DELAY=5m

# =============================================================================
# 健康检查配置
# =============================================================================
QCAT_HEALTH_CHECK_INTERVAL=30s
QCAT_HEALTH_TIMEOUT=10s
QCAT_HEALTH_RETRY_COUNT=3

# =============================================================================
# 优雅关闭配置
# =============================================================================
QCAT_SHUTDOWN_SHUTDOWN_TIMEOUT=30s
QCAT_SHUTDOWN_COMPONENT_TIMEOUT=10s
QCAT_SHUTDOWN_ENABLE_SIGNAL_HANDLING=true

# =============================================================================
# 内存管理配置
# =============================================================================
QCAT_MEMORY_MONITOR_INTERVAL=30s
QCAT_MEMORY_HIGH_WATER_MARK_PERCENT=80.0
QCAT_MEMORY_LOW_WATER_MARK_PERCENT=60.0

# =============================================================================
# 限流配置
# =============================================================================
QCAT_RATE_LIMIT_ENABLED=true
QCAT_RATE_LIMIT_REQUESTS_PER_MINUTE=100
QCAT_RATE_LIMIT_BURST=20

# =============================================================================
# CORS配置
# =============================================================================
# QCAT_CORS_ALLOWED_ORIGINS=["*"]
# QCAT_CORS_ALLOWED_METHODS=["GET", "POST", "PUT", "DELETE", "OPTIONS"]
# QCAT_CORS_ALLOW_CREDENTIALS=true
`

	// 写入文件
	if err := os.WriteFile(envPath, []byte(template), 0644); err != nil {
		log.Fatalf("写入环境变量模板失败: %v", err)
	}

	fmt.Printf("✅ 环境变量模板已生成: %s\n", envPath)
	fmt.Println("请根据实际情况修改配置值")
}

func encryptString(text string) {
	// 获取加密密钥
	encryptionKey := os.Getenv("QCAT_ENCRYPTION_KEY")
	if encryptionKey == "" {
		log.Fatal("请设置 QCAT_ENCRYPTION_KEY 环境变量")
	}

	// 创建环境管理器
	envManager := config.NewEnvManager(encryptionKey, "")

	// 加密字符串
	encrypted, err := envManager.SetEncryptedString("TEMP", text)
	if err != nil {
		log.Fatalf("加密失败: %v", err)
	}

	// 提取加密后的值（去掉前缀）
	encryptedValue := encrypted[4:] // 去掉 "TEMP=" 前缀

	fmt.Printf("原文: %s\n", text)
	fmt.Printf("加密后: ENC:%s\n", encryptedValue)
	fmt.Println("请将加密后的值设置到环境变量中")
}

func decryptString(encryptedText string) {
	// 获取加密密钥
	encryptionKey := os.Getenv("QCAT_ENCRYPTION_KEY")
	if encryptionKey == "" {
		log.Fatal("请设置 QCAT_ENCRYPTION_KEY 环境变量")
	}

	// 创建环境管理器
	envManager := config.NewEnvManager(encryptionKey, "")

	// 解密字符串
	decrypted := envManager.GetEncryptedString("TEMP", encryptedText)

	fmt.Printf("加密文本: %s\n", encryptedText)
	fmt.Printf("解密后: %s\n", decrypted)
}
