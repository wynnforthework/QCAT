package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
		Env     string `yaml:"env"`
	} `yaml:"app"`

	Server struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"server"`

	Database struct {
		Driver          string `yaml:"driver"`
		Host            string `yaml:"host"`
		Port            int    `yaml:"port"`
		Name            string `yaml:"name"`
		User            string `yaml:"user"`
		Password        string `yaml:"password"`
		SSLMode         string `yaml:"sslmode"`
		MaxOpenConns    int    `yaml:"max_open_conns"`
		MaxIdleConns    int    `yaml:"max_idle_conns"`
		ConnMaxLifetime string `yaml:"conn_max_lifetime"`
	} `yaml:"database"`

	Redis struct {
		Enabled  bool   `yaml:"enabled"`
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
		PoolSize int    `yaml:"pool_size"`
	} `yaml:"redis"`

	Exchange struct {
		Binance struct {
			APIKey    string `yaml:"api_key"`
			APISecret string `yaml:"api_secret"`
			TestNet   bool   `yaml:"testnet"`
			RateLimit int    `yaml:"rate_limit"`
		} `yaml:"binance"`
	} `yaml:"exchange"`

	Risk struct {
		MaxLeverage             float64 `yaml:"max_leverage"`
		MaxPositionSize         float64 `yaml:"max_position_size"`
		MaxDrawdown             float64 `yaml:"max_drawdown"`
		CircuitBreakerThreshold float64 `yaml:"circuit_breaker_threshold"`
	} `yaml:"risk"`

	Logging struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
		Output string `yaml:"output"`
	} `yaml:"logging"`
}

// Load reads the config file and returns a Config struct
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	return &cfg, nil
}
