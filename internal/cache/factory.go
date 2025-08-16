package cache

import (
	"fmt"
	"time"

	"qcat/internal/database"
)

// CacheFactory creates cache instances with fallback support
type CacheFactory struct {
	config *CacheFactoryConfig
}

// CacheFactoryConfig defines cache factory configuration
type CacheFactoryConfig struct {
	// Redis configuration
	RedisEnabled bool   `json:"redis_enabled"`
	RedisAddr    string `json:"redis_addr"`
	RedisPassword string `json:"redis_password"`
	RedisDB      int    `json:"redis_db"`
	RedisPoolSize int   `json:"redis_pool_size"`

	// Memory cache configuration
	MemoryEnabled    bool `json:"memory_enabled"`
	MemoryMaxSize    int  `json:"memory_max_size"`

	// Database cache configuration
	DatabaseEnabled   bool   `json:"database_enabled"`
	DatabaseTableName string `json:"database_table_name"`

	// Fallback configuration
	FallbackConfig *FallbackConfig `json:"fallback_config"`
}

// NewCacheFactory creates a new cache factory
func NewCacheFactory(config *CacheFactoryConfig) *CacheFactory {
	if config == nil {
		config = DefaultCacheFactoryConfig()
	}

	return &CacheFactory{
		config: config,
	}
}

// DefaultCacheFactoryConfig returns default cache factory configuration
func DefaultCacheFactoryConfig() *CacheFactoryConfig {
	return &CacheFactoryConfig{
		RedisEnabled:      true,
		RedisAddr:         "localhost:6379",
		RedisPassword:     "",
		RedisDB:           0,
		RedisPoolSize:     10,
		MemoryEnabled:     true,
		MemoryMaxSize:     10000,
		DatabaseEnabled:   true,
		DatabaseTableName: "cache_entries",
		FallbackConfig:    DefaultFallbackConfig(),
	}
}

// CreateCache creates a cache with fallback support
func (cf *CacheFactory) CreateCache(db *database.DB) (Cacher, error) {
	var redis Cacher
	var dbCache DatabaseCache
	var err error

	// Create Redis cache if enabled
	if cf.config.RedisEnabled {
		redisConfig := &Config{
			Addr:     cf.config.RedisAddr,
			Password: cf.config.RedisPassword,
			DB:       cf.config.RedisDB,
			PoolSize: cf.config.RedisPoolSize,
		}

		redis, err = NewRedisCache(redisConfig)
		if err != nil {
			// Log warning but continue with fallback
			fmt.Printf("Warning: Failed to create Redis cache: %v\n", err)
			redis = nil
		}
	}

	// Create database cache if enabled and database is available
	if cf.config.DatabaseEnabled && db != nil {
		dbCache, err = NewDatabaseCache(db, cf.config.DatabaseTableName)
		if err != nil {
			// Log warning but continue
			fmt.Printf("Warning: Failed to create database cache: %v\n", err)
			dbCache = nil
		}
	}

	// Create cache manager with fallback support
	cacheManager := NewCacheManager(redis, dbCache, cf.config.FallbackConfig)

	return cacheManager, nil
}

// CreateMemoryOnlyCache creates a memory-only cache (for testing or fallback)
func (cf *CacheFactory) CreateMemoryOnlyCache() Cacher {
	return NewMemoryCache(cf.config.MemoryMaxSize)
}

// CreateRedisOnlyCache creates a Redis-only cache
func (cf *CacheFactory) CreateRedisOnlyCache() (Cacher, error) {
	if !cf.config.RedisEnabled {
		return nil, fmt.Errorf("Redis is not enabled in configuration")
	}

	redisConfig := &Config{
		Addr:     cf.config.RedisAddr,
		Password: cf.config.RedisPassword,
		DB:       cf.config.RedisDB,
		PoolSize: cf.config.RedisPoolSize,
	}

	return NewRedisCache(redisConfig)
}

// CreateDatabaseOnlyCache creates a database-only cache
func (cf *CacheFactory) CreateDatabaseOnlyCache(db *database.DB) (DatabaseCache, error) {
	if !cf.config.DatabaseEnabled {
		return nil, fmt.Errorf("Database cache is not enabled in configuration")
	}

	if db == nil {
		return nil, fmt.Errorf("Database connection is required")
	}

	return NewDatabaseCache(db, cf.config.DatabaseTableName)
}

// ValidateConfig validates the cache factory configuration
func (cf *CacheFactory) ValidateConfig() error {
	if !cf.config.RedisEnabled && !cf.config.MemoryEnabled && !cf.config.DatabaseEnabled {
		return fmt.Errorf("at least one cache type must be enabled")
	}

	if cf.config.RedisEnabled {
		if cf.config.RedisAddr == "" {
			return fmt.Errorf("Redis address is required when Redis is enabled")
		}
		if cf.config.RedisPoolSize <= 0 {
			return fmt.Errorf("Redis pool size must be positive")
		}
	}

	if cf.config.MemoryEnabled {
		if cf.config.MemoryMaxSize <= 0 {
			return fmt.Errorf("Memory cache max size must be positive")
		}
	}

	if cf.config.DatabaseEnabled {
		if cf.config.DatabaseTableName == "" {
			return fmt.Errorf("Database table name is required when database cache is enabled")
		}
	}

	if cf.config.FallbackConfig != nil {
		if cf.config.FallbackConfig.HealthCheckInterval <= 0 {
			return fmt.Errorf("Health check interval must be positive")
		}
		if cf.config.FallbackConfig.FailureThreshold <= 0 {
			return fmt.Errorf("Failure threshold must be positive")
		}
		if cf.config.FallbackConfig.RecoveryThreshold <= 0 {
			return fmt.Errorf("Recovery threshold must be positive")
		}
	}

	return nil
}

// GetSupportedCacheTypes returns the supported cache types
func (cf *CacheFactory) GetSupportedCacheTypes() []string {
	var types []string

	if cf.config.RedisEnabled {
		types = append(types, "redis")
	}
	if cf.config.MemoryEnabled {
		types = append(types, "memory")
	}
	if cf.config.DatabaseEnabled {
		types = append(types, "database")
	}

	return types
}

// CreateCacheWithOptions creates a cache with specific options
func (cf *CacheFactory) CreateCacheWithOptions(db *database.DB, options *CacheOptions) (Cacher, error) {
	if options == nil {
		return cf.CreateCache(db)
	}

	// Override configuration with options
	config := *cf.config // Copy config

	if options.DisableRedis {
		config.RedisEnabled = false
	}
	if options.DisableMemory {
		config.MemoryEnabled = false
	}
	if options.DisableDatabase {
		config.DatabaseEnabled = false
	}
	if options.MemoryMaxSize > 0 {
		config.MemoryMaxSize = options.MemoryMaxSize
	}
	if options.FallbackConfig != nil {
		config.FallbackConfig = options.FallbackConfig
	}

	// Create temporary factory with modified config
	tempFactory := &CacheFactory{config: &config}
	return tempFactory.CreateCache(db)
}

// CacheOptions defines options for cache creation
type CacheOptions struct {
	DisableRedis     bool            `json:"disable_redis"`
	DisableMemory    bool            `json:"disable_memory"`
	DisableDatabase  bool            `json:"disable_database"`
	MemoryMaxSize    int             `json:"memory_max_size"`
	FallbackConfig   *FallbackConfig `json:"fallback_config"`
}

// CacheHealthChecker provides health checking for cache systems
type CacheHealthChecker struct {
	cacheManager *CacheManager
}

// NewCacheHealthChecker creates a new cache health checker
func NewCacheHealthChecker(cacheManager *CacheManager) *CacheHealthChecker {
	return &CacheHealthChecker{
		cacheManager: cacheManager,
	}
}

// CheckHealth performs a comprehensive health check
func (chc *CacheHealthChecker) CheckHealth() *CacheHealthReport {
	if chc.cacheManager == nil {
		return &CacheHealthReport{
			Overall: "unhealthy",
			Error:   "Cache manager is nil",
		}
	}

	stats := chc.cacheManager.GetStats()
	summary := chc.cacheManager.monitor.GetHealthSummary()

	report := &CacheHealthReport{
		Overall:        summary.OverallHealth,
		RedisHealth:    summary.RedisHealth,
		MemoryHealth:   true, // Memory cache is always healthy if it exists
		DatabaseHealth: true, // Assume database is healthy if no errors
		FallbackActive: summary.FallbackActive,
		HitRatio:       summary.HitRatio,
		ErrorRatio:     summary.ErrorRatio,
		Stats:          stats,
		Timestamp:      time.Now(),
	}

	// Determine overall health
	if !summary.RedisHealth && summary.FallbackActive {
		report.Overall = "degraded"
	}
	if summary.ErrorRatio > 0.2 { // More than 20% error rate
		report.Overall = "unhealthy"
	}

	return report
}

// CacheHealthReport represents a cache health report
type CacheHealthReport struct {
	Overall        string      `json:"overall"`
	RedisHealth    bool        `json:"redis_health"`
	MemoryHealth   bool        `json:"memory_health"`
	DatabaseHealth bool        `json:"database_health"`
	FallbackActive bool        `json:"fallback_active"`
	HitRatio       float64     `json:"hit_ratio"`
	ErrorRatio     float64     `json:"error_ratio"`
	Stats          *CacheStats `json:"stats"`
	Error          string      `json:"error,omitempty"`
	Timestamp      time.Time   `json:"timestamp"`
}