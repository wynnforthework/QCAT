package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

// DB represents the database connection
type DB struct {
	*sql.DB
}

// Config represents database configuration
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	MaxOpen  int
	MaxIdle  int
	Timeout  time.Duration
}

// NewConnection creates a new database connection
func NewConnection(cfg *Config) (*DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpen)
	db.SetMaxIdleConns(cfg.MaxIdle)
	db.SetConnMaxLifetime(cfg.Timeout)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Database connection established successfully")
	return &DB{db}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}

// HealthCheck performs a health check on the database
func (db *DB) HealthCheck(ctx context.Context) error {
	return db.PingContext(ctx)
}
