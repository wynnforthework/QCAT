package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	App        AppConfig        `yaml:"app"`
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	Redis      RedisConfig      `yaml:"redis"`
	JWT        JWTConfig        `yaml:"jwt"`
	Monitoring MonitoringConfig `yaml:"monitoring"`
	CORS       CORSConfig       `yaml:"cors"`
	RateLimit  RateLimitConfig  `yaml:"rate_limit"`
	Logging    LoggingConfig    `yaml:"logging"`
}

// AppConfig represents application configuration
type AppConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	Env     string `yaml:"env"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Port           int           `yaml:"port"`
	Host           string        `yaml:"host"`
	ReadTimeout    time.Duration `yaml:"read_timeout"`
	WriteTimeout   time.Duration `yaml:"write_timeout"`
	MaxHeaderBytes int           `yaml:"max_header_bytes"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Host     string        `yaml:"host"`
	Port     int           `yaml:"port"`
	User     string        `yaml:"user"`
	Password string        `yaml:"password"`
	DBName   string        `yaml:"dbname"`
	SSLMode  string        `yaml:"sslmode"`
	MaxOpen  int           `yaml:"max_open"`
	MaxIdle  int           `yaml:"max_idle"`
	Timeout  time.Duration `yaml:"timeout"`
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

// JWTConfig represents JWT configuration
type JWTConfig struct {
	SecretKey string        `yaml:"secret_key"`
	Duration  time.Duration `yaml:"duration"`
}

// MonitoringConfig represents monitoring configuration
type MonitoringConfig struct {
	PrometheusEnabled bool   `yaml:"prometheus_enabled"`
	PrometheusPath    string `yaml:"prometheus_path"`
}

// CORSConfig represents CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string `yaml:"allowed_origins"`
	AllowedMethods   []string `yaml:"allowed_methods"`
	AllowedHeaders   []string `yaml:"allowed_headers"`
	AllowCredentials bool     `yaml:"allow_credentials"`
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	Enabled           bool `yaml:"enabled"`
	RequestsPerMinute int  `yaml:"requests_per_minute"`
	Burst             int  `yaml:"burst"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

// Load loads configuration from a YAML file
func Load(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}
