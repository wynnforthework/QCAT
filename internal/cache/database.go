package cache

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"qcat/internal/database"
)

// DatabaseCacheImpl implements DatabaseCache interface using SQL database
type DatabaseCacheImpl struct {
	db        *database.DB
	tableName string
}

// CacheEntry represents a cache entry in the database
type CacheEntry struct {
	Key        string    `json:"key"`
	Value      string    `json:"value"`
	Expiration time.Time `json:"expiration"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// NewDatabaseCache creates a new database cache implementation
func NewDatabaseCache(db *database.DB, tableName string) (*DatabaseCacheImpl, error) {
	if tableName == "" {
		tableName = "cache_entries"
	}

	dbc := &DatabaseCacheImpl{
		db:        db,
		tableName: tableName,
	}

	// Create table if it doesn't exist
	if err := dbc.createTable(); err != nil {
		return nil, fmt.Errorf("failed to create cache table: %w", err)
	}

	// Start cleanup goroutine
	go dbc.cleanupLoop()

	return dbc, nil
}

// createTable creates the cache table if it doesn't exist
func (dbc *DatabaseCacheImpl) createTable() error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			key VARCHAR(255) PRIMARY KEY,
			value TEXT NOT NULL,
			expiration TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_expiration (expiration)
		)
	`, dbc.tableName)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := dbc.db.ExecContext(ctx, query)
	return err
}

// Get retrieves a value from database cache
func (dbc *DatabaseCacheImpl) Get(ctx context.Context, key string) (interface{}, error) {
	query := fmt.Sprintf(`
		SELECT value FROM %s 
		WHERE key = ? AND expiration > NOW()
	`, dbc.tableName)

	var valueStr string
	err := dbc.db.QueryRowContext(ctx, query, key).Scan(&valueStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("key not found or expired: %s", key)
		}
		return nil, fmt.Errorf("database query error: %w", err)
	}

	// Try to unmarshal as JSON, if it fails, return as string
	var value interface{}
	if err := json.Unmarshal([]byte(valueStr), &value); err != nil {
		// If JSON unmarshal fails, return as string
		return valueStr, nil
	}

	return value, nil
}

// Set stores a value in database cache
func (dbc *DatabaseCacheImpl) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	// Serialize value to JSON
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	expirationTime := time.Now().Add(expiration)
	if expiration <= 0 {
		expirationTime = time.Now().Add(24 * time.Hour) // Default 24 hour expiration
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (key, value, expiration) 
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE 
			value = VALUES(value),
			expiration = VALUES(expiration),
			updated_at = CURRENT_TIMESTAMP
	`, dbc.tableName)

	_, err = dbc.db.ExecContext(ctx, query, key, string(valueBytes), expirationTime)
	if err != nil {
		return fmt.Errorf("database insert/update error: %w", err)
	}

	return nil
}

// Delete removes a value from database cache
func (dbc *DatabaseCacheImpl) Delete(ctx context.Context, key string) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE key = ?`, dbc.tableName)

	_, err := dbc.db.ExecContext(ctx, query, key)
	if err != nil {
		return fmt.Errorf("database delete error: %w", err)
	}

	return nil
}

// Exists checks if a key exists in database cache
func (dbc *DatabaseCacheImpl) Exists(ctx context.Context, key string) (bool, error) {
	query := fmt.Sprintf(`
		SELECT 1 FROM %s 
		WHERE key = ? AND expiration > NOW() 
		LIMIT 1
	`, dbc.tableName)

	var exists int
	err := dbc.db.QueryRowContext(ctx, query, key).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("database query error: %w", err)
	}

	return exists == 1, nil
}

// GetAllKeys returns all non-expired keys from database cache
func (dbc *DatabaseCacheImpl) GetAllKeys(ctx context.Context) ([]string, error) {
	query := fmt.Sprintf(`
		SELECT key FROM %s 
		WHERE expiration > NOW()
		ORDER BY created_at DESC
	`, dbc.tableName)

	rows, err := dbc.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("database query error: %w", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("row scan error: %w", err)
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return keys, nil
}

// GetStats returns database cache statistics
func (dbc *DatabaseCacheImpl) GetStats(ctx context.Context) (*DatabaseCacheStats, error) {
	query := fmt.Sprintf(`
		SELECT 
			COUNT(*) as total_entries,
			COUNT(CASE WHEN expiration > NOW() THEN 1 END) as active_entries,
			COUNT(CASE WHEN expiration <= NOW() THEN 1 END) as expired_entries
		FROM %s
	`, dbc.tableName)

	var stats DatabaseCacheStats
	err := dbc.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalEntries,
		&stats.ActiveEntries,
		&stats.ExpiredEntries,
	)
	if err != nil {
		return nil, fmt.Errorf("database query error: %w", err)
	}

	stats.LastUpdated = time.Now()
	return &stats, nil
}

// Clear removes all entries from database cache
func (dbc *DatabaseCacheImpl) Clear(ctx context.Context) error {
	query := fmt.Sprintf(`DELETE FROM %s`, dbc.tableName)

	_, err := dbc.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("database clear error: %w", err)
	}

	return nil
}

// cleanupLoop runs periodic cleanup of expired entries
func (dbc *DatabaseCacheImpl) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute) // Cleanup every 10 minutes
	defer ticker.Stop()

	for range ticker.C {
		dbc.cleanup()
	}
}

// cleanup removes expired entries from database
func (dbc *DatabaseCacheImpl) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := fmt.Sprintf(`DELETE FROM %s WHERE expiration <= NOW()`, dbc.tableName)

	result, err := dbc.db.ExecContext(ctx, query)
	if err != nil {
		// Log error but don't fail
		fmt.Printf("Database cache cleanup error: %v\n", err)
		return
	}

	if rowsAffected, err := result.RowsAffected(); err == nil && rowsAffected > 0 {
		fmt.Printf("Database cache cleanup: removed %d expired entries\n", rowsAffected)
	}
}

// GetEntry retrieves a complete cache entry
func (dbc *DatabaseCacheImpl) GetEntry(ctx context.Context, key string) (*CacheEntry, error) {
	query := fmt.Sprintf(`
		SELECT key, value, expiration, created_at, updated_at 
		FROM %s 
		WHERE key = ?
	`, dbc.tableName)

	var entry CacheEntry
	err := dbc.db.QueryRowContext(ctx, query, key).Scan(
		&entry.Key,
		&entry.Value,
		&entry.Expiration,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("key not found: %s", key)
		}
		return nil, fmt.Errorf("database query error: %w", err)
	}

	return &entry, nil
}

// SetTTL sets the time to live for a key
func (dbc *DatabaseCacheImpl) SetTTL(ctx context.Context, key string, ttl time.Duration) error {
	expirationTime := time.Now().Add(ttl)

	query := fmt.Sprintf(`
		UPDATE %s 
		SET expiration = ?, updated_at = CURRENT_TIMESTAMP 
		WHERE key = ?
	`, dbc.tableName)

	result, err := dbc.db.ExecContext(ctx, query, expirationTime, key)
	if err != nil {
		return fmt.Errorf("database update error: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("key not found: %s", key)
	}

	return nil
}

// GetTTL returns the time to live for a key
func (dbc *DatabaseCacheImpl) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	query := fmt.Sprintf(`
		SELECT expiration FROM %s 
		WHERE key = ?
	`, dbc.tableName)

	var expiration time.Time
	err := dbc.db.QueryRowContext(ctx, query, key).Scan(&expiration)
	if err != nil {
		if err == sql.ErrNoRows {
			return -2 * time.Second, fmt.Errorf("key not found: %s", key) // -2 means key doesn't exist
		}
		return 0, fmt.Errorf("database query error: %w", err)
	}

	ttl := time.Until(expiration)
	if ttl < 0 {
		return -1 * time.Second, nil // -1 means key exists but has expired
	}

	return ttl, nil
}

// DatabaseCacheStats represents database cache statistics
type DatabaseCacheStats struct {
	TotalEntries   int       `json:"total_entries"`
	ActiveEntries  int       `json:"active_entries"`
	ExpiredEntries int       `json:"expired_entries"`
	LastUpdated    time.Time `json:"last_updated"`
}