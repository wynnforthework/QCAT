//go:build tools

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"qcat/internal/auth"
	"qcat/internal/config"
	"qcat/internal/database"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	fmt.Println("=== QCAT JWT认证修复测试 ===")

	// 1. 加载配置
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	fmt.Printf("JWT Secret Key: %s\n", cfg.JWT.SecretKey)
	fmt.Printf("JWT Duration: %v\n", cfg.JWT.Duration)

	// 2. 连接数据库
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

	// 3. 检查用户表是否存在
	ctx := context.Background()
	var tableExists bool
	err = db.QueryRowContext(ctx, "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'users')").Scan(&tableExists)
	if err != nil {
		log.Fatalf("检查用户表失败: %v", err)
	}

	if !tableExists {
		fmt.Println("❌ 用户表不存在，需要运行数据库迁移")
		return
	}

	fmt.Println("✅ 用户表存在")

	// 4. 检查是否有admin用户
	var userCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE username = 'admin'").Scan(&userCount)
	if err != nil {
		log.Fatalf("检查admin用户失败: %v", err)
	}

	if userCount == 0 {
		fmt.Println("⚠️ admin用户不存在，创建admin用户...")
		err = createAdminUser(db, ctx)
		if err != nil {
			log.Fatalf("创建admin用户失败: %v", err)
		}
		fmt.Println("✅ admin用户创建成功")
	} else {
		fmt.Println("✅ admin用户存在")
	}

	// 5. 测试JWT token生成和验证
	jwtManager := auth.NewJWTManager(cfg.JWT.SecretKey, cfg.JWT.Duration)

	// 生成token
	userID := uuid.New().String()
	token, err := jwtManager.GenerateToken(userID, "admin", "admin")
	if err != nil {
		log.Fatalf("生成JWT token失败: %v", err)
	}

	fmt.Printf("✅ JWT token生成成功: %s...\n", token[:50])

	// 验证token
	claims, err := jwtManager.ValidateToken(token)
	if err != nil {
		log.Fatalf("验证JWT token失败: %v", err)
	}

	fmt.Printf("✅ JWT token验证成功: UserID=%s, Username=%s, Role=%s\n", 
		claims.UserID, claims.Username, claims.Role)

	// 6. 测试用户认证
	user, err := db.GetUserByUsername(ctx, "admin")
	if err != nil {
		log.Fatalf("获取admin用户失败: %v", err)
	}

	// 验证密码
	err = database.ValidatePassword("admin123", user.PasswordHash)
	if err != nil {
		fmt.Printf("❌ admin用户密码验证失败: %v\n", err)
		fmt.Println("⚠️ 尝试更新admin用户密码...")
		err = updateAdminPassword(db, ctx)
		if err != nil {
			log.Fatalf("更新admin用户密码失败: %v", err)
		}
		fmt.Println("✅ admin用户密码更新成功")
	} else {
		fmt.Println("✅ admin用户密码验证成功")
	}

	fmt.Println("\n=== JWT认证系统检查完成 ===")
	fmt.Println("如果所有检查都通过，JWT认证应该可以正常工作")
}

func createAdminUser(db *database.DB, ctx context.Context) error {
	// 生成密码哈希
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("生成密码哈希失败: %w", err)
	}

	userID := uuid.New()
	query := `
		INSERT INTO users (id, username, email, password_hash, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (username) DO UPDATE SET
			password_hash = EXCLUDED.password_hash,
			updated_at = EXCLUDED.updated_at
	`

	_, err = db.ExecContext(ctx, query,
		userID.String(), "admin", "admin@qcat.local", string(hashedPassword),
		"admin", "active", time.Now(), time.Now())

	return err
}

func updateAdminPassword(db *database.DB, ctx context.Context) error {
	// 生成新的密码哈希
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("生成密码哈希失败: %w", err)
	}

	query := `
		UPDATE users 
		SET password_hash = $1, updated_at = $2
		WHERE username = 'admin'
	`

	_, err = db.ExecContext(ctx, query, string(hashedPassword), time.Now())
	return err
}
