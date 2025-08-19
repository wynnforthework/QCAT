package main

import (
	"flag"
	"fmt"
	"log"
	"os"

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
