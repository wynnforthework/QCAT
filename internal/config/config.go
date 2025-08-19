package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	App        AppConfig        `yaml:"app"`
	Ports      PortsConfig      `yaml:"ports"`
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	Redis      RedisConfig      `yaml:"redis"`
	Exchange   ExchangeConfig   `yaml:"exchange"`
	JWT        JWTConfig        `yaml:"jwt"`
	Monitoring MonitoringConfig `yaml:"monitoring"`
	CORS       CORSConfig       `yaml:"cors"`
	RateLimit  RateLimitConfig  `yaml:"rate_limit"`
	Security   SecurityConfig   `yaml:"security"`
	Logging    LoggingConfig    `yaml:"logging"`
	Memory     MemoryConfig     `yaml:"memory"`
	Network    NetworkConfig    `yaml:"network"`
	Health     HealthConfig     `yaml:"health"`
	Shutdown   ShutdownConfig   `yaml:"shutdown"`
	Strategy   StrategyConfig   `yaml:"strategy"`
	Optimizer  OptimizerConfig  `yaml:"optimizer"`
	MarketData MarketDataConfig `yaml:"market_data"`
	Order      OrderConfig      `yaml:"order"`
	Risk       RiskConfig       `yaml:"risk"`
	Cache      CacheConfig      `yaml:"cache"`
}

// AppConfig 应用配置
type AppConfig struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Environment string `yaml:"environment"`
}

// PortsConfig 端口配置
type PortsConfig struct {
	QcatAPI       int `yaml:"qcat_api"`       // QCAT主应用API服务
	QcatOptimizer int `yaml:"qcat_optimizer"` // QCAT优化器服务
	Postgres      int `yaml:"postgres"`       // PostgreSQL数据库
	Redis         int `yaml:"redis"`          // Redis缓存
	Prometheus    int `yaml:"prometheus"`     // Prometheus监控
	Grafana       int `yaml:"grafana"`        // Grafana监控面板
	AlertManager  int `yaml:"alertmanager"`   // AlertManager告警
	NginxHTTP     int `yaml:"nginx_http"`     // Nginx HTTP
	NginxHTTPS    int `yaml:"nginx_https"`    // Nginx HTTPS
	FrontendDev   int `yaml:"frontend_dev"`   // 前端开发服务器
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port           int           `yaml:"port"`
	ReadTimeout    time.Duration `yaml:"read_timeout"`
	WriteTimeout   time.Duration `yaml:"write_timeout"`
	MaxHeaderBytes int           `yaml:"max_header_bytes"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	DBName          string        `yaml:"dbname"`
	SSLMode         string        `yaml:"ssl_mode"`
	MaxOpen         int           `yaml:"max_open"`
	MaxIdle         int           `yaml:"max_idle"`
	Timeout         time.Duration `yaml:"timeout"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Enabled      bool          `yaml:"enabled"`
	Addr         string        `yaml:"addr"`
	Password     string        `yaml:"password"`
	DB           int           `yaml:"db"`
	PoolSize     int           `yaml:"pool_size"`
	MinIdleConns int           `yaml:"min_idle_conns"`
	MaxRetries   int           `yaml:"max_retries"`
	DialTimeout  time.Duration `yaml:"dial_timeout"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

// ExchangeConfig 交易所配置
type ExchangeConfig struct {
	Name                string            `yaml:"name"`
	APIKey              string            `yaml:"api_key"`
	APISecret           string            `yaml:"api_secret"`
	TestNet             bool              `yaml:"test_net"`
	BaseURL             string            `yaml:"base_url"`
	WebsocketURL        string            `yaml:"websocket_url"`
	FuturesBaseURL      string            `yaml:"futures_base_url"`
	FuturesWebsocketURL string            `yaml:"futures_websocket_url"`
	RateLimit           ExchangeRateLimit `yaml:"rate_limit"`
	Timeout             time.Duration     `yaml:"timeout"`
	RetryAttempts       int               `yaml:"retry_attempts"`
	RetryDelay          time.Duration     `yaml:"retry_delay"`
}

// ExchangeRateLimit 交易所限流配置
type ExchangeRateLimit struct {
	Enabled           bool `yaml:"enabled"`
	RequestsPerMinute int  `yaml:"requests_per_minute"`
	Burst             int  `yaml:"burst"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	SecretKey string        `yaml:"secret_key"`
	Duration  time.Duration `yaml:"duration"`
}

// MonitoringConfig 监控配置
type MonitoringConfig struct {
	PrometheusEnabled bool              `yaml:"prometheus_enabled"`
	PrometheusPath    string            `yaml:"prometheus_path"`
	Alerts            AlertsConfig      `yaml:"alerts"`
	HealthCheck       HealthCheckConfig `yaml:"health_check"`
	Metrics           MetricsMonConfig  `yaml:"metrics"`
}

// AlertsConfig represents alerts configuration
type AlertsConfig struct {
	HighLatencyMs      int     `yaml:"high_latency_ms"`
	ErrorRatePercent   float64 `yaml:"error_rate_percent"`
	MemoryUsagePercent float64 `yaml:"memory_usage_percent"`
	CPUUsagePercent    float64 `yaml:"cpu_usage_percent"`
}

// HealthCheckConfig represents health check configuration
type HealthCheckConfig struct {
	IntervalSeconds int `yaml:"interval_seconds"`
	TimeoutSeconds  int `yaml:"timeout_seconds"`
	RetryCount      int `yaml:"retry_count"`
}

// MetricsMonConfig represents metrics monitoring configuration
type MetricsMonConfig struct {
	CollectionIntervalSeconds int      `yaml:"collection_interval_seconds"`
	RetentionHours            int      `yaml:"retention_hours"`
	AggregationIntervals      []string `yaml:"aggregation_intervals"`
}

// CORSConfig CORS配置
type CORSConfig struct {
	AllowedOrigins   []string `yaml:"allowed_origins"`
	AllowedMethods   []string `yaml:"allowed_methods"`
	AllowedHeaders   []string `yaml:"allowed_headers"`
	AllowCredentials bool     `yaml:"allow_credentials"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled           bool `yaml:"enabled"`
	RequestsPerMinute int  `yaml:"requests_per_minute"`
	Burst             int  `yaml:"burst"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	KMS        KMSConfig             `yaml:"kms"`
	Encryption EncryptionConfig      `yaml:"encryption"`
	TLS        TLSConfig             `yaml:"tls"`
	Network    NetworkSecurityConfig `yaml:"network"`
	Audit      AuditConfig           `yaml:"audit"`
	Approval   ApprovalConfig        `yaml:"approval"`
}

// KMSConfig 密钥管理服务配置
type KMSConfig struct {
	MasterKey     string        `yaml:"master_key"`
	KeyRotation   time.Duration `yaml:"key_rotation"`
	EncryptionKey string        `yaml:"encryption_key"`
}

// EncryptionConfig 加密配置
type EncryptionConfig struct {
	Algorithm      string        `yaml:"algorithm"`
	KeySize        int           `yaml:"key_size"`
	KeyRotation    time.Duration `yaml:"key_rotation"`
	MasterKey      string        `yaml:"master_key"`
	PublicKeyPath  string        `yaml:"public_key_path"`
	PrivateKeyPath string        `yaml:"private_key_path"`
}

// TLSConfig TLS配置
type TLSConfig struct {
	Enabled      bool     `yaml:"enabled"`
	CertPath     string   `yaml:"cert_path"`
	KeyPath      string   `yaml:"key_path"`
	MinVersion   uint16   `yaml:"min_version"`
	MaxVersion   uint16   `yaml:"max_version"`
	CipherSuites []uint16 `yaml:"cipher_suites"`
}

// NetworkSecurityConfig 网络安全配置
type NetworkSecurityConfig struct {
	RateLimitEnabled  bool          `yaml:"rate_limit_enabled"`
	RateLimitRequests int           `yaml:"rate_limit_requests"`
	RateLimitWindow   time.Duration `yaml:"rate_limit_window"`
	MaxConnections    int           `yaml:"max_connections"`
	ConnectionTimeout time.Duration `yaml:"connection_timeout"`
	IdleTimeout       time.Duration `yaml:"idle_timeout"`
	ReadTimeout       time.Duration `yaml:"read_timeout"`
	WriteTimeout      time.Duration `yaml:"write_timeout"`
	AllowedOrigins    []string      `yaml:"allowed_origins"`
	AllowedMethods    []string      `yaml:"allowed_methods"`
	AllowedHeaders    []string      `yaml:"allowed_headers"`
	ExposedHeaders    []string      `yaml:"exposed_headers"`
	AllowCredentials  bool          `yaml:"allow_credentials"`
	MaxAge            time.Duration `yaml:"max_age"`
}

// AuditConfig 审计配置
type AuditConfig struct {
	Enabled           bool   `yaml:"enabled"`
	LogLevel          string `yaml:"log_level"`
	RetentionDays     int    `yaml:"retention_days"`
	EncryptionEnabled bool   `yaml:"encryption_enabled"`
}

// ApprovalConfig 审批配置
type ApprovalConfig struct {
	Enabled          bool          `yaml:"enabled"`
	DefaultExpiry    time.Duration `yaml:"default_expiry"`
	MinApprovers     int           `yaml:"min_approvers"`
	AutoApproveRoles []string      `yaml:"auto_approve_roles"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level        string            `yaml:"level"`
	Format       string            `yaml:"format"`
	Output       string            `yaml:"output"`
	Levels       map[string]string `yaml:"levels"`
	OutputConfig OutputConfig      `yaml:"output_config"`
	MaxSize      int               `yaml:"max_size"`
	MaxBackups   int               `yaml:"max_backups"`
	MaxAge       int               `yaml:"max_age"`
	Compress     bool              `yaml:"compress"`
	LogDir       string            `yaml:"log_dir"`
}

// OutputConfig represents log output configuration
type OutputConfig struct {
	ConsoleEnabled bool   `yaml:"console_enabled"`
	FileEnabled    bool   `yaml:"file_enabled"`
	FilePath       string `yaml:"file_path"`
	MaxFileSizeMB  int    `yaml:"max_file_size_mb"`
	MaxBackupFiles int    `yaml:"max_backup_files"`
}

// MemoryConfig 内存管理配置
type MemoryConfig struct {
	MonitorInterval      time.Duration `yaml:"monitor_interval"`
	HighWaterMarkPercent float64       `yaml:"high_water_mark_percent"`
	LowWaterMarkPercent  float64       `yaml:"low_water_mark_percent"`
	AlertThreshold       float64       `yaml:"alert_threshold"`
	EnableAutoGC         bool          `yaml:"enable_auto_gc"`
	GCInterval           time.Duration `yaml:"gc_interval"`
	ForceGCThreshold     float64       `yaml:"force_gc_threshold"`
	MaxMemoryMB          uint64        `yaml:"max_memory_mb"`
	MaxHeapMB            uint64        `yaml:"max_heap_mb"`
}

// NetworkConfig 网络重连配置
type NetworkConfig struct {
	MaxRetries             int           `yaml:"max_retries"`
	InitialDelay           time.Duration `yaml:"initial_delay"`
	MaxDelay               time.Duration `yaml:"max_delay"`
	BackoffMultiplier      float64       `yaml:"backoff_multiplier"`
	JitterFactor           float64       `yaml:"jitter_factor"`
	HealthCheckInterval    time.Duration `yaml:"health_check_interval"`
	ConnectionTimeout      time.Duration `yaml:"connection_timeout"`
	PingInterval           time.Duration `yaml:"ping_interval"`
	PongTimeout            time.Duration `yaml:"pong_timeout"`
	MaxConsecutiveFailures int           `yaml:"max_consecutive_failures"`
	AlertThreshold         int           `yaml:"alert_threshold"`
}

// HealthConfig 健康检查配置
type HealthConfig struct {
	CheckInterval      time.Duration `yaml:"check_interval"`
	Timeout            time.Duration `yaml:"timeout"`
	RetryCount         int           `yaml:"retry_count"`
	RetryInterval      time.Duration `yaml:"retry_interval"`
	DegradedThreshold  float64       `yaml:"degraded_threshold"`
	UnhealthyThreshold float64       `yaml:"unhealthy_threshold"`
	AlertThreshold     int           `yaml:"alert_threshold"`
	AlertCooldown      time.Duration `yaml:"alert_cooldown"`
}

// ShutdownConfig 优雅关闭配置
type ShutdownConfig struct {
	ShutdownTimeout      time.Duration `yaml:"shutdown_timeout"`
	ComponentTimeout     time.Duration `yaml:"component_timeout"`
	SignalTimeout        time.Duration `yaml:"signal_timeout"`
	EnableSignalHandling bool          `yaml:"enable_signal_handling"`
	ForceShutdownAfter   time.Duration `yaml:"force_shutdown_after"`
	LogShutdownProgress  bool          `yaml:"log_shutdown_progress"`
	ShutdownOrder        []string      `yaml:"shutdown_order"`
}

// StrategyConfig 策略配置
type StrategyConfig struct {
	DefaultMode             string         `yaml:"default_mode"`
	MaxConcurrentStrategies int            `yaml:"max_concurrent_strategies"`
	StrategyTimeout         time.Duration  `yaml:"strategy_timeout"`
	MemoryLimitMB           int            `yaml:"memory_limit_mb"`
	SandboxEnabled          bool           `yaml:"sandbox_enabled"`
	Backtest                BacktestConfig `yaml:"backtest"`
}

// BacktestConfig 回测配置
type BacktestConfig struct {
	Enabled           bool          `yaml:"enabled"`
	Timeout           time.Duration `yaml:"timeout"`
	MaxConcurrency    int           `yaml:"max_concurrency"`
	DataRetentionDays int           `yaml:"data_retention_days"`
}

// OrderConfig 订单管理配置
type OrderConfig struct {
	Timeout           time.Duration `yaml:"timeout"`
	MaxPendingOrders  int           `yaml:"max_pending_orders"`
	RetryAttempts     int           `yaml:"retry_attempts"`
	RetryDelay        time.Duration `yaml:"retry_delay"`
	AutoCancelTimeout time.Duration `yaml:"auto_cancel_timeout"`
}

// RiskConfig 风险管理配置
type RiskConfig struct {
	Enabled                 bool                     `yaml:"enabled"`
	CheckInterval           time.Duration            `yaml:"check_interval"`
	MarginCallThreshold     float64                  `yaml:"margin_call_threshold"`
	LiquidationThreshold    float64                  `yaml:"liquidation_threshold"`
	MaxPositionSize         float64                  `yaml:"max_position_size"`
	MaxLeverage             int                      `yaml:"max_leverage"`
	MaxDrawdown             float64                  `yaml:"max_drawdown"`
	CircuitBreakerThreshold float64                  `yaml:"circuit_breaker_threshold"`
	PositionMonitoring      PositionMonitoringConfig `yaml:"position_monitoring"`
}

// PositionMonitoringConfig 仓位监控配置
type PositionMonitoringConfig struct {
	Enabled        bool          `yaml:"enabled"`
	Interval       time.Duration `yaml:"interval"`
	AlertThreshold float64       `yaml:"alert_threshold"`
}

// CacheConfig 缓存配置
type CacheConfig struct {
	TTL                time.Duration `yaml:"ttl"`
	MaxSize            int           `yaml:"max_size"`
	CleanupInterval    time.Duration `yaml:"cleanup_interval"`
	CompressionEnabled bool          `yaml:"compression_enabled"`
	EncryptionEnabled  bool          `yaml:"encryption_enabled"`
}

// Load 加载配置文件
func Load(configPath string) (*Config, error) {
	// Initialize environment manager
	envManager := NewEnvManager("", "QCAT_")

	// Load environment variables from .env file if it exists
	if _, err := os.Stat(".env"); err == nil {
		if err := envManager.LoadFromFile(".env"); err != nil {
			return nil, fmt.Errorf("failed to load .env file: %w", err)
		}
	}

	// Load YAML configuration
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Override with environment variables
	config.overrideWithEnv(envManager)

	return &config, nil
}

// overrideWithEnv overrides configuration with environment variables
func (c *Config) overrideWithEnv(env *EnvManager) {
	// App configuration
	if env.GetString("APP_NAME", "") != "" {
		c.App.Name = env.GetString("APP_NAME", c.App.Name)
	}
	if env.GetString("APP_VERSION", "") != "" {
		c.App.Version = env.GetString("APP_VERSION", c.App.Version)
	}
	if env.GetString("APP_ENVIRONMENT", "") != "" {
		c.App.Environment = env.GetString("APP_ENVIRONMENT", c.App.Environment)
	}

	// Ports configuration
	if env.GetInt("PORTS_QCAT_API", 0) != 0 {
		c.Ports.QcatAPI = env.GetInt("PORTS_QCAT_API", c.Ports.QcatAPI)
	}
	if env.GetInt("PORTS_QCAT_OPTIMIZER", 0) != 0 {
		c.Ports.QcatOptimizer = env.GetInt("PORTS_QCAT_OPTIMIZER", c.Ports.QcatOptimizer)
	}
	if env.GetInt("PORTS_POSTGRES", 0) != 0 {
		c.Ports.Postgres = env.GetInt("PORTS_POSTGRES", c.Ports.Postgres)
	}
	if env.GetInt("PORTS_REDIS", 0) != 0 {
		c.Ports.Redis = env.GetInt("PORTS_REDIS", c.Ports.Redis)
	}
	if env.GetInt("PORTS_PROMETHEUS", 0) != 0 {
		c.Ports.Prometheus = env.GetInt("PORTS_PROMETHEUS", c.Ports.Prometheus)
	}
	if env.GetInt("PORTS_GRAFANA", 0) != 0 {
		c.Ports.Grafana = env.GetInt("PORTS_GRAFANA", c.Ports.Grafana)
	}
	if env.GetInt("PORTS_ALERTMANAGER", 0) != 0 {
		c.Ports.AlertManager = env.GetInt("PORTS_ALERTMANAGER", c.Ports.AlertManager)
	}
	if env.GetInt("PORTS_NGINX_HTTP", 0) != 0 {
		c.Ports.NginxHTTP = env.GetInt("PORTS_NGINX_HTTP", c.Ports.NginxHTTP)
	}
	if env.GetInt("PORTS_NGINX_HTTPS", 0) != 0 {
		c.Ports.NginxHTTPS = env.GetInt("PORTS_NGINX_HTTPS", c.Ports.NginxHTTPS)
	}
	if env.GetInt("PORTS_FRONTEND_DEV", 0) != 0 {
		c.Ports.FrontendDev = env.GetInt("PORTS_FRONTEND_DEV", c.Ports.FrontendDev)
	}

	// Server configuration
	if env.GetInt("SERVER_PORT", 0) != 0 {
		c.Server.Port = env.GetInt("SERVER_PORT", c.Server.Port)
	}
	// 如果没有设置SERVER_PORT但设置了PORTS_QCAT_API，则使用PORTS_QCAT_API
	if c.Server.Port == 0 && c.Ports.QcatAPI != 0 {
		c.Server.Port = c.Ports.QcatAPI
	}
	if env.GetDuration("SERVER_READ_TIMEOUT", 0) != 0 {
		c.Server.ReadTimeout = env.GetDuration("SERVER_READ_TIMEOUT", c.Server.ReadTimeout)
	}
	if env.GetDuration("SERVER_WRITE_TIMEOUT", 0) != 0 {
		c.Server.WriteTimeout = env.GetDuration("SERVER_WRITE_TIMEOUT", c.Server.WriteTimeout)
	}

	// Database configuration
	if env.GetString("DATABASE_HOST", "") != "" {
		c.Database.Host = env.GetString("DATABASE_HOST", c.Database.Host)
	}
	if env.GetInt("DATABASE_PORT", 0) != 0 {
		c.Database.Port = env.GetInt("DATABASE_PORT", c.Database.Port)
	}
	if env.GetString("DATABASE_USER", "") != "" {
		c.Database.User = env.GetString("DATABASE_USER", c.Database.User)
	}
	if env.GetString("DATABASE_PASSWORD", "") != "" {
		c.Database.Password = env.GetEncryptedString("DATABASE_PASSWORD", c.Database.Password)
	}
	if env.GetString("DATABASE_NAME", "") != "" {
		c.Database.DBName = env.GetString("DATABASE_NAME", c.Database.DBName)
	}
	if env.GetString("DATABASE_SSL_MODE", "") != "" {
		c.Database.SSLMode = env.GetString("DATABASE_SSL_MODE", c.Database.SSLMode)
	}

	// Redis configuration
	if env.GetBool("REDIS_ENABLED", c.Redis.Enabled) != c.Redis.Enabled {
		c.Redis.Enabled = env.GetBool("REDIS_ENABLED", c.Redis.Enabled)
	}
	if env.GetString("REDIS_ADDR", "") != "" {
		c.Redis.Addr = env.GetString("REDIS_ADDR", c.Redis.Addr)
	}
	if env.GetString("REDIS_PASSWORD", "") != "" {
		c.Redis.Password = env.GetEncryptedString("REDIS_PASSWORD", c.Redis.Password)
	}
	if env.GetInt("REDIS_DB", -1) != -1 {
		c.Redis.DB = env.GetInt("REDIS_DB", c.Redis.DB)
	}
	if env.GetInt("REDIS_POOL_SIZE", 0) != 0 {
		c.Redis.PoolSize = env.GetInt("REDIS_POOL_SIZE", c.Redis.PoolSize)
	}

	// Exchange configuration
	if env.GetString("EXCHANGE_NAME", "") != "" {
		c.Exchange.Name = env.GetString("EXCHANGE_NAME", c.Exchange.Name)
	}
	if env.GetString("EXCHANGE_API_KEY", "") != "" {
		c.Exchange.APIKey = env.GetEncryptedString("EXCHANGE_API_KEY", c.Exchange.APIKey)
	}
	if env.GetString("EXCHANGE_API_SECRET", "") != "" {
		c.Exchange.APISecret = env.GetEncryptedString("EXCHANGE_API_SECRET", c.Exchange.APISecret)
	}
	if env.GetBool("EXCHANGE_TEST_NET", c.Exchange.TestNet) != c.Exchange.TestNet {
		c.Exchange.TestNet = env.GetBool("EXCHANGE_TEST_NET", c.Exchange.TestNet)
	}
	if env.GetString("EXCHANGE_BASE_URL", "") != "" {
		c.Exchange.BaseURL = env.GetString("EXCHANGE_BASE_URL", c.Exchange.BaseURL)
	}
	if env.GetString("EXCHANGE_WEBSOCKET_URL", "") != "" {
		c.Exchange.WebsocketURL = env.GetString("EXCHANGE_WEBSOCKET_URL", c.Exchange.WebsocketURL)
	}

	// JWT configuration
	if env.GetString("JWT_SECRET_KEY", "") != "" {
		c.JWT.SecretKey = env.GetEncryptedString("JWT_SECRET_KEY", c.JWT.SecretKey)
	}
	if env.GetDuration("JWT_DURATION", 0) != 0 {
		c.JWT.Duration = env.GetDuration("JWT_DURATION", c.JWT.Duration)
	}

	// Security configuration
	if env.GetString("SECURITY_KMS_MASTER_KEY", "") != "" {
		c.Security.KMS.MasterKey = env.GetEncryptedString("SECURITY_KMS_MASTER_KEY", c.Security.KMS.MasterKey)
	}
	if env.GetString("SECURITY_ENCRYPTION_MASTER_KEY", "") != "" {
		c.Security.Encryption.MasterKey = env.GetEncryptedString("SECURITY_ENCRYPTION_MASTER_KEY", c.Security.Encryption.MasterKey)
	}
}
