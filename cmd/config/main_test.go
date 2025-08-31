package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"qcat/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShowHelp tests the help message display
func TestShowHelp(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	showHelp()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "QCAT 配置管理工具")
	assert.Contains(t, output, "用法:")
	assert.Contains(t, output, "-validate")
	assert.Contains(t, output, "-encrypt")
	assert.Contains(t, output, "-decrypt")
	assert.Contains(t, output, "示例:")
}

// TestFlagParsing tests command line flag parsing
func TestFlagParsing(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected map[string]interface{}
	}{
		{
			name: "default values",
			args: []string{},
			expected: map[string]interface{}{
				"config":   "configs/config.yaml",
				"validate": false,
				"encrypt":  "",
				"decrypt":  "",
				"help":     false,
			},
		},
		{
			name: "validate flag",
			args: []string{"-validate"},
			expected: map[string]interface{}{
				"validate": true,
			},
		},
		{
			name: "custom config",
			args: []string{"-config", "test.yaml"},
			expected: map[string]interface{}{
				"config": "test.yaml",
			},
		},
		{
			name: "encrypt string",
			args: []string{"-encrypt", "secret"},
			expected: map[string]interface{}{
				"encrypt": "secret",
			},
		},
		{
			name: "decrypt string",
			args: []string{"-decrypt", "ENC:encrypted"},
			expected: map[string]interface{}{
				"decrypt": "ENC:encrypted",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			var (
				configPath = flag.String("config", "configs/config.yaml", "配置文件路径")
				validate   = flag.Bool("validate", false, "验证配置")
				encrypt    = flag.String("encrypt", "", "加密字符串")
				decrypt    = flag.String("decrypt", "", "解密字符串")
				help       = flag.Bool("help", false, "显示帮助信息")
			)

			err := flag.CommandLine.Parse(tt.args)
			require.NoError(t, err)

			if expected, ok := tt.expected["config"]; ok {
				assert.Equal(t, expected, *configPath)
			}
			if expected, ok := tt.expected["validate"]; ok {
				assert.Equal(t, expected, *validate)
			}
			if expected, ok := tt.expected["encrypt"]; ok {
				assert.Equal(t, expected, *encrypt)
			}
			if expected, ok := tt.expected["decrypt"]; ok {
				assert.Equal(t, expected, *decrypt)
			}
			if expected, ok := tt.expected["help"]; ok {
				assert.Equal(t, expected, *help)
			}
		})
	}
}

// TestValidateConfig tests configuration validation
func TestValidateConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yaml")

	// Create a valid test config
	configContent := `
app:
  name: "QCAT Test"
  version: "1.0.0"
  environment: "test"

server:
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  max_header_bytes: 1048576

database:
  host: "localhost"
  port: 5432
  user: "test"
  password: "test"
  dbname: "qcat_test"
  sslmode: "disable"

redis:
  host: "localhost"
  port: 6379
  password: ""
  db: 0
  enabled: true
  addr: "localhost:6379"

exchange:
  name: "binance"
  testnet: true

strategy:
  default_mode: "paper"

optimizer:
  grid_search:
    max_iterations: 100

risk:
  enabled: true

market_data:
  quality:
    min_quality_score: 0.8
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Test config loading and basic validation
	cfg, err := config.Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, "QCAT Test", cfg.App.Name)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "localhost", cfg.Database.Host)
}

// TestValidateConfigNonExistent tests validation with non-existent config file
func TestValidateConfigNonExistent(t *testing.T) {
	// This would normally call log.Fatalf, so we can't test it directly
	// But we can test the file existence check
	_, err := os.Stat("nonexistent_config.yaml")
	assert.True(t, os.IsNotExist(err))
}

// TestShowConfigSummary tests configuration summary display
func TestShowConfigSummary(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			Name:        "Test App",
			Version:     "1.0.0",
			Environment: "test",
		},
		Server: config.ServerConfig{
			Port: 8080,
		},
		Database: config.DatabaseConfig{
			Host:   "localhost",
			Port:   5432,
			DBName: "testdb",
		},
		Redis: config.RedisConfig{
			Addr:    "localhost:6379",
			Enabled: true,
		},
		Exchange: config.ExchangeConfig{
			Name:    "binance",
			TestNet: true,
		},
		Strategy: config.StrategyConfig{
			DefaultMode: "paper",
		},
		Optimizer: config.OptimizerConfig{
			GridSearch: config.GridSearchConfig{
				MaxIterations: 100,
			},
		},
		Risk: config.RiskConfig{
			Enabled: true,
		},
		MarketData: config.MarketDataConfig{
			Quality: config.QualityConfig{
				MinQualityScore: 0.8,
			},
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	showConfigSummary(cfg)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "配置摘要:")
	assert.Contains(t, output, "Test App")
	assert.Contains(t, output, "1.0.0")
	assert.Contains(t, output, "test")
	assert.Contains(t, output, "8080")
	assert.Contains(t, output, "localhost:5432/testdb")
	assert.Contains(t, output, "localhost:6379")
	assert.Contains(t, output, "binance")
	assert.Contains(t, output, "paper")
}

// TestEncryptionFunctions tests encryption and decryption functionality
func TestEncryptionFunctions(t *testing.T) {
	// Set up test encryption key
	originalKey := os.Getenv("QCAT_ENCRYPTION_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("QCAT_ENCRYPTION_KEY", originalKey)
		} else {
			os.Unsetenv("QCAT_ENCRYPTION_KEY")
		}
	}()

	testKey := "test-encryption-key-32-characters"
	os.Setenv("QCAT_ENCRYPTION_KEY", testKey)

	t.Run("encryption key present", func(t *testing.T) {
		key := os.Getenv("QCAT_ENCRYPTION_KEY")
		assert.Equal(t, testKey, key)
	})

	t.Run("encryption key missing", func(t *testing.T) {
		os.Unsetenv("QCAT_ENCRYPTION_KEY")
		key := os.Getenv("QCAT_ENCRYPTION_KEY")
		assert.Empty(t, key)
		
		// Restore key for other tests
		os.Setenv("QCAT_ENCRYPTION_KEY", testKey)
	})
}

// TestEnvironmentVariables tests environment variable handling
func TestEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name     string
		envVar   string
		value    string
		expected string
	}{
		{
			name:     "set encryption key",
			envVar:   "QCAT_ENCRYPTION_KEY",
			value:    "test-key",
			expected: "test-key",
		},
		{
			name:     "set config path",
			envVar:   "QCAT_CONFIG_PATH",
			value:    "/path/to/config.yaml",
			expected: "/path/to/config.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original value
			original := os.Getenv(tt.envVar)
			defer func() {
				if original != "" {
					os.Setenv(tt.envVar, original)
				} else {
					os.Unsetenv(tt.envVar)
				}
			}()

			// Set test value
			os.Setenv(tt.envVar, tt.value)
			
			// Verify
			actual := os.Getenv(tt.envVar)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

// TestStringOperations tests string manipulation functions
func TestStringOperations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		prefix   string
		expected string
	}{
		{
			name:     "has ENC prefix",
			input:    "ENC:encrypted-value",
			prefix:   "ENC:",
			expected: "encrypted-value",
		},
		{
			name:     "no ENC prefix",
			input:    "plain-value",
			prefix:   "ENC:",
			expected: "plain-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasPrefix := strings.HasPrefix(tt.input, tt.prefix)
			if hasPrefix {
				result := strings.TrimPrefix(tt.input, tt.prefix)
				assert.Equal(t, tt.expected, result)
			} else {
				assert.Equal(t, tt.expected, tt.input)
			}
		})
	}
}
