package main

import (
	"flag"
	"fmt"
	"log"

	"qcat/internal/config"
	"qcat/internal/database"
)

func main() {
	var (
		configPath = flag.String("config", "configs/config.yaml", "配置文件路径")
		up         = flag.Bool("up", false, "运行数据库迁移")
		down       = flag.Bool("down", false, "回滚数据库迁移")
		version    = flag.Bool("version", false, "显示当前迁移版本")
		force      = flag.Int("force", -1, "强制设置迁移版本（用于修复脏状态）")
		drop       = flag.Bool("drop", false, "删除所有数据库表")
		help       = flag.Bool("help", false, "显示帮助信息")
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

	// 执行操作
	switch {
	case *up:
		runMigrations(migrator)
	case *down:
		rollbackMigrations(migrator)
	case *version:
		showVersion(migrator)
	case *force >= 0:
		forceMigrationVersion(migrator, *force)
	case *drop:
		dropDatabase(migrator)
	default:
		// 默认运行迁移
		runMigrations(migrator)
	}
}

func showHelp() {
	fmt.Println("QCAT 数据库迁移工具")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  migrate [选项]")
	fmt.Println()
	fmt.Println("选项:")
	fmt.Println("  -config string")
	fmt.Println("        配置文件路径 (默认: configs/config.yaml)")
	fmt.Println("  -up")
	fmt.Println("        运行数据库迁移")
	fmt.Println("  -down")
	fmt.Println("        回滚数据库迁移")
	fmt.Println("  -version")
	fmt.Println("        显示当前迁移版本")
	fmt.Println("  -force int")
	fmt.Println("        强制设置迁移版本（用于修复脏状态）")
	fmt.Println("  -drop")
	fmt.Println("        删除所有数据库表（危险操作）")
	fmt.Println("  -help")
	fmt.Println("        显示帮助信息")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  migrate -up")
	fmt.Println("  migrate -down")
	fmt.Println("  migrate -version")
	fmt.Println("  migrate -force 7    # 修复脏状态，强制设置为版本7")
	fmt.Println("  migrate -drop       # 删除所有表")
	fmt.Println("  migrate -config configs/production.yaml -up")
}

func runMigrations(migrator *database.Migrator) {
	log.Println("开始运行数据库迁移...")

	if err := migrator.Up(); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	log.Println("✅ 数据库迁移完成")
}

func rollbackMigrations(migrator *database.Migrator) {
	log.Println("开始回滚数据库迁移...")

	if err := migrator.Down(); err != nil {
		log.Fatalf("数据库回滚失败: %v", err)
	}

	log.Println("✅ 数据库回滚完成")
}

func showVersion(migrator *database.Migrator) {
	version, err := migrator.Version()
	if err != nil {
		log.Fatalf("获取迁移版本失败: %v", err)
	}

	fmt.Printf("当前迁移版本: %d\n", version)
}

func forceMigrationVersion(migrator *database.Migrator, version int) {
	log.Printf("强制设置迁移版本为: %d", version)

	if err := migrator.Force(version); err != nil {
		log.Fatalf("强制设置迁移版本失败: %v", err)
	}

	log.Println("✅ 迁移版本强制设置完成")
}

func dropDatabase(migrator *database.Migrator) {
	log.Println("警告: 即将删除所有数据库表!")
	log.Println("开始删除数据库表...")

	if err := migrator.Drop(); err != nil {
		log.Fatalf("删除数据库表失败: %v", err)
	}

	log.Println("✅ 数据库表删除完成")
}
