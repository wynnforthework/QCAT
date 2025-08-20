package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Database connection parameters
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "qcat_user")
	dbPassword := getEnv("DB_PASSWORD", "qcat_password")
	dbName := getEnv("DB_NAME", "qcat")

	// Connect to database
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("✅ Connected to database successfully")

	// Generate fresh password hashes
	adminHash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to generate admin password hash: %v", err)
	}

	testHash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to generate test password hash: %v", err)
	}

	demoHash, err := bcrypt.GenerateFromPassword([]byte("demo123"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to generate demo password hash: %v", err)
	}

	fmt.Printf("Generated password hashes:\n")
	fmt.Printf("Admin (admin123): %s\n", string(adminHash))
	fmt.Printf("Test (admin123): %s\n", string(testHash))
	fmt.Printf("Demo (demo123): %s\n", string(demoHash))

	// Create users table if it doesn't exist
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		username VARCHAR(255) NOT NULL UNIQUE,
		email VARCHAR(255) NOT NULL UNIQUE,
		password_hash VARCHAR(255) NOT NULL,
		role VARCHAR(50) NOT NULL DEFAULT 'user',
		status VARCHAR(20) NOT NULL DEFAULT 'active',
		last_login TIMESTAMP WITH TIME ZONE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS user_sessions (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		refresh_token VARCHAR(500) NOT NULL UNIQUE,
		expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	-- Create indexes
	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
	CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_user_sessions_refresh_token ON user_sessions(refresh_token);
	CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at);
	`

	_, err = db.ExecContext(ctx, createTableSQL)
	if err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}

	fmt.Println("✅ Tables created/verified successfully")

	// Insert/update users with fresh hashes
	upsertSQL := `
	INSERT INTO users (username, email, password_hash, role, status, created_at, updated_at) 
	VALUES ($1, $2, $3, $4, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	ON CONFLICT (username) DO UPDATE SET
		password_hash = EXCLUDED.password_hash,
		updated_at = CURRENT_TIMESTAMP;
	`

	users := []struct {
		username string
		email    string
		hash     string
		role     string
	}{
		{"admin", "admin@qcat.local", string(adminHash), "admin"},
		{"testuser", "test@qcat.local", string(testHash), "user"},
		{"demo", "demo@qcat.local", string(demoHash), "user"},
	}

	for _, user := range users {
		_, err = db.ExecContext(ctx, upsertSQL, user.username, user.email, user.hash, user.role)
		if err != nil {
			log.Printf("❌ Failed to create/update user %s: %v", user.username, err)
		} else {
			fmt.Printf("✅ User %s created/updated successfully\n", user.username)
		}
	}

	fmt.Println("\n🎉 Migration completed successfully!")
	fmt.Println("\nDefault users created:")
	fmt.Println("- Username: admin, Password: admin123, Role: admin")
	fmt.Println("- Username: testuser, Password: admin123, Role: user")
	fmt.Println("- Username: demo, Password: demo123, Role: user")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
