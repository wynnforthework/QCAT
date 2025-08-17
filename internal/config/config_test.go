package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"qcat/internal/testutils"
)

func TestLoadConfig(t *testing.T) {
	suite := testutils.NewTestSuite(t, nil)
	defer suite.TearDown()

	// 创建测试配置文件
	configContent := `
app:
  name: "QCAT Test"
  version: "1.0.0"
  environment: "test"

server:
  port: 8080
  host: "localhost"
  debug: true

database:
  host: "localhost"
  port: 5432
  user: "test"
  password: "test"
  dbname: "qcat_test"
  sslmode: "disable"

redis:
  addr: "localhost:6379"
  password: ""
  db: 0
`

	configPath := suite.CreateTempFile("config.yaml", configContent)

	// 测试加载配置
	config, err := LoadConfig(configPath)
	suite.Logger.Info("Loading config", "path", configPath)
	
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 验证配置值
	if config.App.Name != "QCAT Test" {
		t.Errorf("Expected app name 'QCAT Test', got '%s'", config.App.Name)
	}

	if config.Server.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", config.Server.Port)
	}

	if config.Database.Host != "localhost" {
		t.Errorf("Expected database host 'localhost', got '%s'", config.Database.Host)
	}
}

func TestLoadConfigWithEnvironmentOverride(t *testing.T) {
	suite := testutils.NewTestSuite(t, nil)
	defer suite.TearDown()

	// 设置环境变量
	testutils.SetEnv(t, "QCAT_SERVER_PORT", "9090")
	testutils.SetEnv(t, "QCAT_DATABASE_HOST", "db.example.com")

	configContent := `
server:
  port: 8080
  host: "localhost"

database:
  host: "localhost"
  port: 5432
`

	configPath := suite.CreateTempFile("config.yaml", configContent)

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 验证环境变量覆盖
	if config.Server.Port != 9090 {
		t.Errorf("Expected port 9090 (from env), got %d", config.Server.Port)
	}

	if config.Database.Host != "db.example.com" {
		t.Errorf("Expected database host 'db.example.com' (from env), got '%s'", config.Database.Host)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name: "valid config",
			config: &Config{
				App: AppConfig{
					Name:        "QCAT",
					Version:     "1.0.0",
					Environment: "production",
				},
				Server: ServerConfig{
					Port: 8080,
					Host: "localhost",
				},
			},
			expectError: false,
		},
		{
			name: "invalid port",
			config: &Config{
				Server: ServerConfig{
					Port: -1,
					Host: "localhost",
				},
			},
			expectError: true,
		},
		{
			name: "empty app name",
			config: &Config{
				App: AppConfig{
					Name:    "",
					Version: "1.0.0",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if tt.expectError && err == nil {
				t.Error("Expected validation error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no validation error, got: %v", err)
			}
		})
	}
}

func TestConfigWatcher(t *testing.T) {
	suite := testutils.NewTestSuite(t, nil)
	defer suite.TearDown()

	configContent := `
server:
  port: 8080
`

	configPath := suite.CreateTempFile("config.yaml", configContent)

	// 创建配置监听器
	watcher, err := NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("Failed to create config watcher: %v", err)
	}
	defer watcher.Stop()

	// 设置变更回调
	changed := false
	watcher.OnChange(func(config *Config) {
		changed = true
	})

	// 启动监听
	err = watcher.Start()
	if err != nil {
		t.Fatalf("Failed to start config watcher: %v", err)
	}

	// 修改配置文件
	newContent := `
server:
  port: 9090
`
	err = os.WriteFile(configPath, []byte(newContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// 等待变更检测
	testutils.Eventually(t, func() bool {
		return changed
	}, 5*time.Second, "config change should be detected")
}
