package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"qcat/internal/config"
	"qcat/internal/database"
)

func main() {
	var (
		configPath   = flag.String("config", "configs/config.yaml", "配置文件路径")
		monitorFlag  = flag.Bool("monitor", false, "启动迁移监控")
		checkFlag    = flag.Bool("check", false, "检查迁移状态")
		recoverFlag  = flag.Bool("recover", false, "尝试自动恢复")
		forceVersion = flag.Int("force", -1, "强制设置迁移版本")
		validateFlag = flag.Bool("validate", false, "验证迁移完整性")
		help         = flag.Bool("help", false, "显示帮助信息")
	)
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 连接数据库
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
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	// 创建迁移器
	migrator, err := database.NewMigrator(db, "internal/database/migrations")
	if err != nil {
		log.Fatalf("创建迁移器失败: %v", err)
	}
	defer migrator.Close()

	// 创建迁移监控器
	monitorConfig := &database.MigrationMonitorConfig{
		CheckInterval:  30 * time.Second,
		MaxRetries:     3,
		RetryDelay:     5 * time.Second,
		AlertThreshold: 2,
		AutoRecovery:   true,
		NotificationFunc: func(message string) {
			log.Printf("📧 ALERT: %s", message)
			// 这里可以集成邮件、Slack等通知
		},
	}

	migrationMonitor := database.NewMigrationMonitor(migrator, monitorConfig)

	// 执行操作
	switch {
	case *monitorFlag:
		startMonitoring(migrationMonitor)
	case *checkFlag:
		checkMigrationStatus(migrationMonitor)
	case *recoverFlag:
		attemptRecovery(migrationMonitor)
	case *forceVersion >= 0:
		forceRecovery(migrationMonitor, *forceVersion)
	case *validateFlag:
		validateMigrations(migrationMonitor)
	default:
		// 默认检查状态
		checkMigrationStatus(migrationMonitor)
	}
}

func showHelp() {
	fmt.Println("QCAT 数据库迁移健康检查工具")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  migration-health [选项]")
	fmt.Println()
	fmt.Println("选项:")
	fmt.Println("  -config string")
	fmt.Println("        配置文件路径 (默认: configs/config.yaml)")
	fmt.Println("  -monitor")
	fmt.Println("        启动迁移监控服务")
	fmt.Println("  -check")
	fmt.Println("        检查当前迁移状态")
	fmt.Println("  -recover")
	fmt.Println("        尝试自动恢复脏状态")
	fmt.Println("  -force int")
	fmt.Println("        强制设置迁移版本")
	fmt.Println("  -validate")
	fmt.Println("        验证迁移完整性")
	fmt.Println("  -help")
	fmt.Println("        显示帮助信息")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  migration-health -check")
	fmt.Println("  migration-health -monitor")
	fmt.Println("  migration-health -recover")
	fmt.Println("  migration-health -force 8")
	fmt.Println("  migration-health -validate")
}

func startMonitoring(monitor *database.MigrationMonitor) {
	log.Println("🔍 启动迁移监控服务...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 处理优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("收到关闭信号，正在停止监控...")
		cancel()
	}()

	// 启动监控
	monitor.Start(ctx)
}

func checkMigrationStatus(monitor *database.MigrationMonitor) {
	log.Println("🔍 检查迁移状态...")

	status, err := monitor.GetStatus()
	if err != nil {
		log.Fatalf("获取迁移状态失败: %v", err)
	}

	fmt.Printf("迁移状态报告:\n")
	fmt.Printf("  当前版本: %d\n", status.CurrentVersion)
	fmt.Printf("  是否脏状态: %v\n", status.IsDirty)
	fmt.Printf("  最后检查时间: %s\n", status.LastChecked.Format("2006-01-02 15:04:05"))
	fmt.Printf("  错误次数: %d\n", status.ErrorCount)

	if status.LastError != nil {
		fmt.Printf("  最后错误: %v\n", status.LastError)
	}

	if status.IsDirty {
		fmt.Printf("\n⚠️  数据库处于脏状态，需要修复！\n")
		fmt.Printf("建议运行: migration-health -recover\n")
		os.Exit(1)
	} else {
		fmt.Printf("\n✅ 迁移状态正常\n")
	}
}

func attemptRecovery(monitor *database.MigrationMonitor) {
	log.Println("🔧 尝试自动恢复...")

	status, err := monitor.GetStatus()
	if err != nil {
		log.Fatalf("获取迁移状态失败: %v", err)
	}

	if !status.IsDirty {
		log.Println("✅ 数据库状态正常，无需恢复")
		return
	}

	// 尝试恢复到当前版本
	if err := monitor.ForceRecovery(int(status.CurrentVersion)); err != nil {
		// 如果失败，尝试恢复到前一个版本
		if status.CurrentVersion > 0 {
			log.Printf("尝试恢复到前一个版本 %d", status.CurrentVersion-1)
			if err := monitor.ForceRecovery(int(status.CurrentVersion - 1)); err != nil {
				log.Fatalf("自动恢复失败: %v", err)
			}
		} else {
			log.Fatalf("自动恢复失败: %v", err)
		}
	}

	log.Println("✅ 自动恢复成功")
}

func forceRecovery(monitor *database.MigrationMonitor, version int) {
	log.Printf("🔧 强制恢复到版本 %d...", version)

	if err := monitor.ForceRecovery(version); err != nil {
		log.Fatalf("强制恢复失败: %v", err)
	}

	log.Printf("✅ 强制恢复到版本 %d 成功", version)
}

func validateMigrations(monitor *database.MigrationMonitor) {
	log.Println("🔍 验证迁移完整性...")

	if err := monitor.ValidateMigrationIntegrity(); err != nil {
		log.Fatalf("迁移完整性验证失败: %v", err)
	}

	log.Println("✅ 迁移完整性验证通过")
}
