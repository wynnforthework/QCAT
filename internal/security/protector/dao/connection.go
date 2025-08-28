package dao

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// DatabaseConfig represents database connection configuration
type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpen         int
	MaxIdle         int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// NewDatabaseConnection creates a new database connection
func NewDatabaseConnection(config *DatabaseConfig) (*sqlx.DB, error) {
	// Build connection string
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode)
	
	// Open database connection
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	
	// Set connection pool parameters
	if config.MaxOpen > 0 {
		db.SetMaxOpenConns(config.MaxOpen)
	} else {
		db.SetMaxOpenConns(25) // Default
	}
	
	if config.MaxIdle > 0 {
		db.SetMaxIdleConns(config.MaxIdle)
	} else {
		db.SetMaxIdleConns(5) // Default
	}
	
	if config.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(config.ConnMaxLifetime)
	} else {
		db.SetConnMaxLifetime(1 * time.Hour) // Default
	}
	
	if config.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	} else {
		db.SetConnMaxIdleTime(15 * time.Minute) // Default
	}
	
	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	
	return db, nil
}

// NewDAOManagerFromConfig creates a new DAO manager from database configuration
func NewDAOManagerFromConfig(config *DatabaseConfig) (DAOManager, error) {
	db, err := NewDatabaseConnection(config)
	if err != nil {
		return nil, err
	}
	
	return NewPostgresDAOManager(db), nil
}

// MigrateDatabase runs database migrations
func MigrateDatabase(db *sqlx.DB, migrationSQL string) error {
	_, err := db.Exec(migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to run database migration: %w", err)
	}
	
	return nil
}

// TestDatabaseConnection tests the database connection and basic operations
func TestDatabaseConnection(db *sqlx.DB) error {
	// Test basic connectivity
	if err := db.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}
	
	// Test a simple query
	var result int
	err := db.Get(&result, "SELECT 1")
	if err != nil {
		return fmt.Errorf("test query failed: %w", err)
	}
	
	if result != 1 {
		return fmt.Errorf("unexpected test query result: %d", result)
	}
	
	return nil
}