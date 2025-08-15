package stability

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

// PoolConfig 连接池配置
type PoolConfig struct {
	MaxOpenConns        int           // 最大打开连接数
	MaxIdleConns        int           // 最大空闲连接数
	ConnMaxLifetime     time.Duration // 连接最大生命周期
	ConnMaxIdleTime     time.Duration // 连接最大空闲时间
	HealthCheckInterval time.Duration // 健康检查间隔
	MaxRetries          int           // 最大重试次数
	RetryDelay          time.Duration // 重试延迟
}

// ConnectionPool 连接池管理器
type ConnectionPool struct {
	mu     sync.RWMutex
	db     *sql.DB
	config *PoolConfig
	stats  *PoolStats
	health *PoolHealth
	ctx    context.Context
	cancel context.CancelFunc
}

// PoolStats 连接池统计
type PoolStats struct {
	OpenConnections   int
	InUse             int
	Idle              int
	WaitCount         int64
	WaitDuration      time.Duration
	MaxIdleClosed     int64
	MaxLifetimeClosed int64
	LastUpdate        time.Time
}

// PoolHealth 连接池健康状态
type PoolHealth struct {
	IsHealthy           bool
	LastCheck           time.Time
	Error               error
	ResponseTime        time.Duration
	FailedChecks        int
	ConsecutiveFailures int
}

// NewConnectionPool 创建连接池管理器
func NewConnectionPool(dsn string, config *PoolConfig) (*ConnectionPool, error) {
	if config == nil {
		config = &PoolConfig{
			MaxOpenConns:        25,
			MaxIdleConns:        10,
			ConnMaxLifetime:     5 * time.Minute,
			ConnMaxIdleTime:     3 * time.Minute,
			HealthCheckInterval: 30 * time.Second,
			MaxRetries:          3,
			RetryDelay:          time.Second,
		}
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 配置连接池
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	ctx, cancel := context.WithCancel(context.Background())

	cp := &ConnectionPool{
		db:     db,
		config: config,
		stats:  &PoolStats{},
		health: &PoolHealth{},
		ctx:    ctx,
		cancel: cancel,
	}

	// 启动健康检查
	go cp.startHealthCheck()

	// 启动统计收集
	go cp.collectStats()

	return cp, nil
}

// GetDB 获取数据库连接
func (cp *ConnectionPool) GetDB() *sql.DB {
	return cp.db
}

// Ping 检查连接池健康状态
func (cp *ConnectionPool) Ping(ctx context.Context) error {
	start := time.Now()
	err := cp.db.PingContext(ctx)
	responseTime := time.Since(start)

	cp.mu.Lock()
	cp.health.LastCheck = time.Now()
	cp.health.ResponseTime = responseTime

	if err != nil {
		cp.health.IsHealthy = false
		cp.health.Error = err
		cp.health.FailedChecks++
		cp.health.ConsecutiveFailures++
	} else {
		cp.health.IsHealthy = true
		cp.health.Error = nil
		cp.health.ConsecutiveFailures = 0
	}
	cp.mu.Unlock()

	return err
}

// ExecWithRetry 带重试的执行
func (cp *ConnectionPool) ExecWithRetry(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	var result sql.Result
	var err error

	for i := 0; i <= cp.config.MaxRetries; i++ {
		result, err = cp.db.ExecContext(ctx, query, args...)
		if err == nil {
			return result, nil
		}

		// 检查是否是连接相关错误
		if cp.isConnectionError(err) && i < cp.config.MaxRetries {
			log.Printf("Connection error on attempt %d/%d: %v, retrying in %v",
				i+1, cp.config.MaxRetries+1, err, cp.config.RetryDelay)
			time.Sleep(cp.config.RetryDelay)
			continue
		}

		break
	}

	return result, err
}

// QueryWithRetry 带重试的查询
func (cp *ConnectionPool) QueryWithRetry(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	var rows *sql.Rows
	var err error

	for i := 0; i <= cp.config.MaxRetries; i++ {
		rows, err = cp.db.QueryContext(ctx, query, args...)
		if err == nil {
			return rows, nil
		}

		// 检查是否是连接相关错误
		if cp.isConnectionError(err) && i < cp.config.MaxRetries {
			log.Printf("Connection error on attempt %d/%d: %v, retrying in %v",
				i+1, cp.config.MaxRetries+1, err, cp.config.RetryDelay)
			time.Sleep(cp.config.RetryDelay)
			continue
		}

		break
	}

	return rows, err
}

// QueryRowWithRetry 带重试的单行查询
func (cp *ConnectionPool) QueryRowWithRetry(ctx context.Context, query string, args ...interface{}) *sql.Row {
	var row *sql.Row
	var err error

	for i := 0; i <= cp.config.MaxRetries; i++ {
		row = cp.db.QueryRowContext(ctx, query, args...)

		// 尝试扫描一个虚拟值来检查错误
		var dummy interface{}
		err = row.Scan(&dummy)
		if err == nil || err == sql.ErrNoRows {
			return row
		}

		// 检查是否是连接相关错误
		if cp.isConnectionError(err) && i < cp.config.MaxRetries {
			log.Printf("Connection error on attempt %d/%d: %v, retrying in %v",
				i+1, cp.config.MaxRetries+1, err, cp.config.RetryDelay)
			time.Sleep(cp.config.RetryDelay)
			continue
		}

		break
	}

	return row
}

// BeginTxWithRetry 带重试的事务开始
func (cp *ConnectionPool) BeginTxWithRetry(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	var tx *sql.Tx
	var err error

	for i := 0; i <= cp.config.MaxRetries; i++ {
		tx, err = cp.db.BeginTx(ctx, opts)
		if err == nil {
			return tx, nil
		}

		// 检查是否是连接相关错误
		if cp.isConnectionError(err) && i < cp.config.MaxRetries {
			log.Printf("Connection error on attempt %d/%d: %v, retrying in %v",
				i+1, cp.config.MaxRetries+1, err, cp.config.RetryDelay)
			time.Sleep(cp.config.RetryDelay)
			continue
		}

		break
	}

	return tx, err
}

// isConnectionError 检查是否是连接相关错误
func (cp *ConnectionPool) isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	// 检查常见的连接错误
	errStr := err.Error()
	connectionErrors := []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"connection lost",
		"server closed the connection",
		"driver: bad connection",
		"pq: connection to server was lost",
		"pq: could not connect to server",
	}

	for _, connErr := range connectionErrors {
		if contains(errStr, connErr) {
			return true
		}
	}

	return false
}

// contains 检查字符串是否包含子字符串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr))))
}

// containsSubstring 检查字符串中间是否包含子字符串
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// startHealthCheck 启动健康检查
func (cp *ConnectionPool) startHealthCheck() {
	ticker := time.NewTicker(cp.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cp.ctx.Done():
			return
		case <-ticker.C:
			cp.checkHealth()
		}
	}
}

// checkHealth 检查健康状态
func (cp *ConnectionPool) checkHealth() {
	ctx, cancel := context.WithTimeout(cp.ctx, 5*time.Second)
	defer cancel()

	if err := cp.Ping(ctx); err != nil {
		log.Printf("Database health check failed: %v", err)

		// 如果连续失败次数过多，可能需要重启连接池
		cp.mu.RLock()
		consecutiveFailures := cp.health.ConsecutiveFailures
		cp.mu.RUnlock()

		if consecutiveFailures >= 5 {
			log.Printf("Too many consecutive failures (%d), considering pool restart", consecutiveFailures)
			// 这里可以添加重启连接池的逻辑
		}
	}
}

// collectStats 收集统计信息
func (cp *ConnectionPool) collectStats() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cp.ctx.Done():
			return
		case <-ticker.C:
			cp.updateStats()
		}
	}
}

// updateStats 更新统计信息
func (cp *ConnectionPool) updateStats() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.stats.OpenConnections = cp.db.Stats().OpenConnections
	cp.stats.InUse = cp.db.Stats().InUse
	cp.stats.Idle = cp.db.Stats().Idle
	cp.stats.WaitCount = cp.db.Stats().WaitCount
	cp.stats.WaitDuration = cp.db.Stats().WaitDuration
	cp.stats.MaxIdleClosed = cp.db.Stats().MaxIdleClosed
	cp.stats.MaxLifetimeClosed = cp.db.Stats().MaxLifetimeClosed
	cp.stats.LastUpdate = time.Now()

	// 检查连接池状态
	if cp.stats.OpenConnections >= cp.config.MaxOpenConns {
		log.Printf("Warning: Connection pool at maximum capacity (%d/%d)",
			cp.stats.OpenConnections, cp.config.MaxOpenConns)
	}

	if cp.stats.WaitCount > 0 {
		log.Printf("Warning: Connection pool has wait count: %d, wait duration: %v",
			cp.stats.WaitCount, cp.stats.WaitDuration)
	}
}

// GetStats 获取统计信息
func (cp *ConnectionPool) GetStats() *PoolStats {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	stats := *cp.stats
	return &stats
}

// GetHealth 获取健康状态
func (cp *ConnectionPool) GetHealth() *PoolHealth {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	health := *cp.health
	return &health
}

// Close 关闭连接池
func (cp *ConnectionPool) Close() error {
	cp.cancel()
	return cp.db.Close()
}

// SetMaxOpenConns 设置最大打开连接数
func (cp *ConnectionPool) SetMaxOpenConns(maxOpenConns int) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.config.MaxOpenConns = maxOpenConns
	cp.db.SetMaxOpenConns(maxOpenConns)
}

// SetMaxIdleConns 设置最大空闲连接数
func (cp *ConnectionPool) SetMaxIdleConns(maxIdleConns int) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.config.MaxIdleConns = maxIdleConns
	cp.db.SetMaxIdleConns(maxIdleConns)
}

// SetConnMaxLifetime 设置连接最大生命周期
func (cp *ConnectionPool) SetConnMaxLifetime(connMaxLifetime time.Duration) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.config.ConnMaxLifetime = connMaxLifetime
	cp.db.SetConnMaxLifetime(connMaxLifetime)
}

// SetConnMaxIdleTime 设置连接最大空闲时间
func (cp *ConnectionPool) SetConnMaxIdleTime(connMaxIdleTime time.Duration) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.config.ConnMaxIdleTime = connMaxIdleTime
	cp.db.SetConnMaxIdleTime(connMaxIdleTime)
}

// GetConfig 获取配置
func (cp *ConnectionPool) GetConfig() *PoolConfig {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	config := *cp.config
	return &config
}
