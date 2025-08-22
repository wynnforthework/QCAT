package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

// DB represents the database connection
type DB struct {
	*sql.DB
	config *Config
	stats  *PoolStats
	mu     sync.RWMutex

	// Monitoring callback
	monitorCallback func(*PoolStats)
}

// Config represents database configuration
type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpen         int
	MaxIdle         int
	Timeout         time.Duration
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// PoolStats represents connection pool statistics
type PoolStats struct {
	MaxOpenConnections int
	OpenConnections    int
	InUse              int
	Idle               int
	WaitCount          int64
	WaitDuration       time.Duration
	MaxIdleClosed      int64
	MaxLifetimeClosed  int64
	LastUpdated        time.Time
}

// NewConnection creates a new database connection
func NewConnection(cfg *Config) (*DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set default values if not provided
	if cfg.MaxOpen <= 0 {
		cfg.MaxOpen = 25 // 默认最大连接数
	}
	if cfg.MaxIdle <= 0 {
		cfg.MaxIdle = 5 // 默认空闲连接数
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second // 默认连接超时
	}
	if cfg.ConnMaxLifetime <= 0 {
		cfg.ConnMaxLifetime = 1 * time.Hour // 默认连接最大生命周期
	}
	if cfg.ConnMaxIdleTime <= 0 {
		cfg.ConnMaxIdleTime = 15 * time.Minute // 默认空闲连接最大时间
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpen)
	db.SetMaxIdleConns(cfg.MaxIdle)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Test connection with retry logic
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	var pingErr error
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		pingErr = db.PingContext(ctx)
		if pingErr == nil {
			break
		}

		log.Printf("Database ping attempt %d/%d failed: %v", i+1, maxRetries, pingErr)
		if i < maxRetries-1 {
			time.Sleep(time.Second * time.Duration(i+1)) // 递增延迟
		}
	}

	if pingErr != nil {
		// 提供更详细的错误信息和建议
		return nil, fmt.Errorf("failed to ping database after %d attempts: %w\n"+
			"Suggestions:\n"+
			"1. Check if PostgreSQL is running: sudo systemctl status postgresql\n"+
			"2. Verify database connection settings in config file\n"+
			"3. Check if database '%s' exists\n"+
			"4. Verify user '%s' has proper permissions\n"+
			"5. Check firewall settings for port %d",
			maxRetries, pingErr, cfg.DBName, cfg.User, cfg.Port)
	}

	log.Printf("Database connection established successfully with pool config: max_open=%d, max_idle=%d, max_lifetime=%v, max_idle_time=%v",
		cfg.MaxOpen, cfg.MaxIdle, cfg.ConnMaxLifetime, cfg.ConnMaxIdleTime)

	database := &DB{
		DB:     db,
		config: cfg,
		stats:  &PoolStats{},
	}

	// Start stats monitoring
	go database.monitorPoolStats()

	return database, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}

// HealthCheck performs a health check on the database
func (db *DB) HealthCheck(ctx context.Context) error {
	return db.PingContext(ctx)
}

// GetPoolStats returns current connection pool statistics
func (db *DB) GetPoolStats() *PoolStats {
	db.mu.RLock()
	defer db.mu.RUnlock()

	// Create a copy to avoid race conditions
	stats := *db.stats
	return &stats
}

// GetConfig returns the database configuration
func (db *DB) GetConfig() *Config {
	return db.config
}

// SetMonitorCallback sets a callback function for monitoring
func (db *DB) SetMonitorCallback(callback func(*PoolStats)) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.monitorCallback = callback
}

// monitorPoolStats periodically updates connection pool statistics
func (db *DB) monitorPoolStats() {
	ticker := time.NewTicker(30 * time.Second) // 每30秒更新一次统计信息
	defer ticker.Stop()

	for range ticker.C {
		db.updatePoolStats()
	}
}

// updatePoolStats updates the connection pool statistics
func (db *DB) updatePoolStats() {
	stats := db.DB.Stats()

	db.mu.Lock()
	db.stats.MaxOpenConnections = stats.MaxOpenConnections
	db.stats.OpenConnections = stats.OpenConnections
	db.stats.InUse = stats.InUse
	db.stats.Idle = stats.Idle
	db.stats.WaitCount = stats.WaitCount
	db.stats.WaitDuration = stats.WaitDuration
	db.stats.MaxIdleClosed = stats.MaxIdleClosed
	db.stats.MaxLifetimeClosed = stats.MaxLifetimeClosed
	db.stats.LastUpdated = time.Now()

	// Call monitoring callback if set
	if db.monitorCallback != nil {
		// Create a copy to avoid race conditions
		statsCopy := *db.stats
		db.mu.Unlock()
		db.monitorCallback(&statsCopy)
	} else {
		db.mu.Unlock()
	}

	// Log warnings if pool is under pressure
	if stats.WaitCount > 0 {
		log.Printf("Database connection pool under pressure: wait_count=%d, wait_duration=%v, in_use=%d, idle=%d",
			stats.WaitCount, stats.WaitDuration, stats.InUse, stats.Idle)
	}

	// Log if connections are being closed frequently
	if stats.MaxIdleClosed > 0 || stats.MaxLifetimeClosed > 0 {
		log.Printf("Database connections being closed: max_idle_closed=%d, max_lifetime_closed=%d",
			stats.MaxIdleClosed, stats.MaxLifetimeClosed)
	}
}

// IsHealthy checks if the database connection pool is healthy
func (db *DB) IsHealthy() bool {
	stats := db.GetPoolStats()

	// Check if we're using too many connections
	if stats.InUse > stats.MaxOpenConnections*80/100 {
		return false
	}

	// Check if we have too many wait events
	if stats.WaitCount > 100 {
		return false
	}

	return true
}

// GetHealthStatus returns detailed health status
func (db *DB) GetHealthStatus() map[string]interface{} {
	stats := db.GetPoolStats()

	// 尝试执行简单查询来测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var pingResult bool = true
	if err := db.PingContext(ctx); err != nil {
		pingResult = false
		log.Printf("Database health check ping failed: %v", err)
	}

	return map[string]interface{}{
		"healthy":              db.IsHealthy() && pingResult,
		"ping_successful":      pingResult,
		"max_open_connections": stats.MaxOpenConnections,
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"wait_count":           stats.WaitCount,
		"wait_duration":        stats.WaitDuration.String(),
		"max_idle_closed":      stats.MaxIdleClosed,
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
		"last_updated":         stats.LastUpdated,
		"utilization_percent":  float64(stats.InUse) / float64(stats.MaxOpenConnections) * 100,
	}
}

// RecoverConnection attempts to recover database connection
func (db *DB) RecoverConnection() error {
	log.Println("Attempting to recover database connection...")

	// 关闭现有连接
	if err := db.DB.Close(); err != nil {
		log.Printf("Error closing existing database connection: %v", err)
	}

	// 重新建立连接
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		db.config.Host, db.config.Port, db.config.User, db.config.Password,
		db.config.DBName, db.config.SSLMode)

	newDB, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to reopen database: %w", err)
	}

	// 重新配置连接池
	newDB.SetMaxOpenConns(db.config.MaxOpen)
	newDB.SetMaxIdleConns(db.config.MaxIdle)
	newDB.SetConnMaxLifetime(db.config.ConnMaxLifetime)
	newDB.SetConnMaxIdleTime(db.config.ConnMaxIdleTime)

	// 测试新连接
	ctx, cancel := context.WithTimeout(context.Background(), db.config.Timeout)
	defer cancel()

	if err := newDB.PingContext(ctx); err != nil {
		newDB.Close()
		return fmt.Errorf("failed to ping recovered database: %w", err)
	}

	// 替换连接
	db.DB = newDB
	log.Println("Database connection recovered successfully")

	return nil
}
