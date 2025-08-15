package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	App        AppConfig        `yaml:"app"`
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	Redis      RedisConfig      `yaml:"redis"`
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
}

// AppConfig 应用配置
type AppConfig struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Environment string `yaml:"environment"`
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
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	SecretKey string        `yaml:"secret_key"`
	Duration  time.Duration `yaml:"duration"`
}

// MonitoringConfig 监控配置
type MonitoringConfig struct {
	PrometheusEnabled bool   `yaml:"prometheus_enabled"`
	PrometheusPath    string `yaml:"prometheus_path"`
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
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
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

// Load 加载配置文件
func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
