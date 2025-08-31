package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"qcat/internal/config"
	"qcat/internal/database"
	"qcat/internal/strategy/autostart"

	_ "github.com/lib/pq"
)

func main() {
	log.Println("🧪 测试策略自动启动功能...")

	// 加载配置
	cfg, err := config.Load("configs/config.yaml")
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

	log.Println("✅ 数据库连接成功")

	// 创建测试策略
	if err := createTestStrategies(db.DB); err != nil {
		log.Fatalf("创建测试策略失败: %v", err)
	}

	// 创建自动启动服务
	autoStartService := autostart.NewAutoStartService(db.DB, autostart.GetDefaultAutoStartConfig())

	// 启动自动启动服务
	if err := autoStartService.Start(); err != nil {
		log.Fatalf("启动自动启动服务失败: %v", err)
	}

	log.Println("✅ 自动启动服务已启动")

	// 等待一段时间让自动启动完成
	time.Sleep(45 * time.Second)

	// 获取统计信息
	stats := autoStartService.GetStats()
	log.Printf("📊 自动启动统计:")
	log.Printf("   总尝试次数: %d", stats.TotalAttempts)
	log.Printf("   成功启动: %d", stats.SuccessfulStarts)
	log.Printf("   失败启动: %d", stats.FailedStarts)
	log.Printf("   跳过启动: %d", stats.SkippedStarts)
	log.Printf("   最后运行时间: %v", stats.LastRunTime)
	log.Printf("   最后运行耗时: %v", stats.LastRunDuration)

	// 停止服务
	autoStartService.Stop()
	log.Println("✅ 测试完成")
}

func createTestStrategies(db *sql.DB) error {
	log.Println("📝 创建测试策略...")

	strategies := []struct {
		name            string
		strategyType    string
		autoStart       bool
		startupPriority int
		enabled         bool
	}{
		{"测试策略A", "trend_following", true, 10, true},
		{"测试策略B", "mean_reversion", true, 20, true},
		{"测试策略C", "arbitrage", false, 50, true},
		{"测试策略D", "momentum", true, 30, false}, // 禁用的策略
	}

	for _, strategy := range strategies {
		query := `
			INSERT INTO strategies (
				id, name, type, status, enabled, auto_start, startup_priority,
				created_at, updated_at, is_running
			) VALUES (
				gen_random_uuid(), $1, $2, 'inactive', $3, $4, $5,
				$6, $6, false
			) ON CONFLICT (name) DO UPDATE SET
				type = EXCLUDED.type,
				enabled = EXCLUDED.enabled,
				auto_start = EXCLUDED.auto_start,
				startup_priority = EXCLUDED.startup_priority,
				updated_at = EXCLUDED.updated_at
		`

		now := time.Now()
		_, err := db.Exec(query,
			strategy.name,
			strategy.strategyType,
			strategy.enabled,
			strategy.autoStart,
			strategy.startupPriority,
			now,
		)
		if err != nil {
			return fmt.Errorf("创建策略 %s 失败: %w", strategy.name, err)
		}

		log.Printf("   ✅ 创建策略: %s (自动启动: %v, 优先级: %d)",
			strategy.name, strategy.autoStart, strategy.startupPriority)
	}

	return nil
}
