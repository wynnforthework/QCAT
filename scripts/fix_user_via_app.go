package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"qcat/internal/config"
	"qcat/internal/database"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	fmt.Println("🔧 通过应用程序修复用户登录问题")
	fmt.Println("================================")

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

	fmt.Println("✅ 数据库连接成功")

	ctx := context.Background()

	// 生成密码哈希
	adminHash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("生成admin密码哈希失败: %v", err)
	}

	demoHash, err := bcrypt.GenerateFromPassword([]byte("demo123"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("生成demo密码哈希失败: %v", err)
	}

	fmt.Printf("生成的密码哈希:\n")
	fmt.Printf("Admin (admin123): %s\n", string(adminHash))
	fmt.Printf("Demo (demo123): %s\n", string(demoHash))

	// 创建或更新用户的SQL
	upsertSQL := `
	INSERT INTO users (username, email, password_hash, role, status, created_at, updated_at) 
	VALUES ($1, $2, $3, $4, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	ON CONFLICT (username) DO UPDATE SET
		password_hash = EXCLUDED.password_hash,
		email = EXCLUDED.email,
		role = EXCLUDED.role,
		updated_at = CURRENT_TIMESTAMP;
	`

	// 用户数据
	users := []struct {
		username string
		email    string
		hash     string
		role     string
	}{
		{"admin", "admin@qcat.local", string(adminHash), "admin"},
		{"testuser", "test@qcat.local", string(adminHash), "user"},
		{"demo", "demo@qcat.local", string(demoHash), "user"},
	}

	// 插入/更新用户
	for _, user := range users {
		_, err = db.ExecContext(ctx, upsertSQL, user.username, user.email, user.hash, user.role)
		if err != nil {
			log.Printf("❌ 创建/更新用户 %s 失败: %v", user.username, err)
		} else {
			fmt.Printf("✅ 用户 %s 创建/更新成功\n", user.username)
		}
	}

	// 验证用户是否创建成功
	fmt.Println("\n🔍 验证用户创建结果:")
	rows, err := db.QueryContext(ctx, `
		SELECT username, email, role, status, created_at 
		FROM users 
		WHERE username IN ('admin', 'testuser', 'demo')
		ORDER BY username
	`)
	if err != nil {
		log.Printf("❌ 查询用户失败: %v", err)
	} else {
		defer rows.Close()
		fmt.Printf("%-10s %-20s %-10s %-10s %s\n", "用户名", "邮箱", "角色", "状态", "创建时间")
		fmt.Println(strings.Repeat("-", 80))

		for rows.Next() {
			var username, email, role, status string
			var createdAt time.Time
			if err := rows.Scan(&username, &email, &role, &status, &createdAt); err != nil {
				log.Printf("❌ 扫描行失败: %v", err)
				continue
			}
			fmt.Printf("%-10s %-20s %-10s %-10s %s\n",
				username, email, role, status, createdAt.Format("2006-01-02 15:04:05"))
		}
	}

	// 测试密码验证
	fmt.Println("\n🧪 测试密码验证:")
	testCases := []struct {
		username string
		password string
	}{
		{"admin", "admin123"},
		{"demo", "demo123"},
	}

	for _, tc := range testCases {
		var storedHash string
		err := db.QueryRowContext(ctx, "SELECT password_hash FROM users WHERE username = $1", tc.username).Scan(&storedHash)
		if err != nil {
			fmt.Printf("❌ 获取用户 %s 的密码哈希失败: %v\n", tc.username, err)
			continue
		}

		err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(tc.password))
		if err != nil {
			fmt.Printf("❌ 用户 %s 密码验证失败: %v\n", tc.username, err)
		} else {
			fmt.Printf("✅ 用户 %s 密码验证成功\n", tc.username)
		}
	}

	fmt.Println("\n🎉 用户修复完成！")
	fmt.Println("\n默认用户账户:")
	fmt.Println("- 用户名: admin, 密码: admin123, 角色: admin")
	fmt.Println("- 用户名: testuser, 密码: admin123, 角色: user")
	fmt.Println("- 用户名: demo, 密码: demo123, 角色: user")
	fmt.Println("\n现在可以测试登录:")
	fmt.Println(`curl -X POST http://localhost:8082/api/v1/auth/login \`)
	fmt.Println(`  -H "Content-Type: application/json" \`)
	fmt.Println(`  -d '{"username": "admin", "password": "admin123"}'`)
}
