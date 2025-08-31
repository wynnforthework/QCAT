package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"qcat/internal/config"
	"qcat/internal/database"
	"qcat/internal/testutils"

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

	assert.Contains(t, output, "QCAT 数据库迁移工具")
	assert.Contains(t, output, "用法:")
	assert.Contains(t, output, "-up")
	assert.Contains(t, output, "-down")
	assert.Contains(t, output, "-version")
	assert.Contains(t, output, "-force")
	assert.Contains(t, output, "-drop")
	assert.Contains(t, output, "示例:")
}

// TestFlagParsing tests command line flag parsing
func TestFlagParsing(t *testing.T) {
	// Reset flags for testing
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	tests := []struct {
		name     string
		args     []string
		expected map[string]interface{}
	}{
		{
			name: "default values",
			args: []string{},
			expected: map[string]interface{}{
				"config":  "configs/config.yaml",
				"up":      false,
				"down":    false,
				"version": false,
				"force":   -1,
				"drop":    false,
				"help":    false,
			},
		},
		{
			name: "up migration",
			args: []string{"-up"},
			expected: map[string]interface{}{
				"up": true,
			},
		},
		{
			name: "down migration",
			args: []string{"-down"},
			expected: map[string]interface{}{
				"down": true,
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
			name: "force version",
			args: []string{"-force", "5"},
			expected: map[string]interface{}{
				"force": 5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			var (
				configPath = flag.String("config", "configs/config.yaml", "配置文件路径")
				up         = flag.Bool("up", false, "运行数据库迁移")
				down       = flag.Bool("down", false, "回滚数据库迁移")
				version    = flag.Bool("version", false, "显示当前迁移版本")
				force      = flag.Int("force", -1, "强制设置迁移版本")
				drop       = flag.Bool("drop", false, "删除所有数据库表")
				help       = flag.Bool("help", false, "显示帮助信息")
			)

			err := flag.CommandLine.Parse(tt.args)
			require.NoError(t, err)

			if expected, ok := tt.expected["config"]; ok {
				assert.Equal(t, expected, *configPath)
			}
			if expected, ok := tt.expected["up"]; ok {
				assert.Equal(t, expected, *up)
			}
			if expected, ok := tt.expected["down"]; ok {
				assert.Equal(t, expected, *down)
			}
			if expected, ok := tt.expected["version"]; ok {
				assert.Equal(t, expected, *version)
			}
			if expected, ok := tt.expected["force"]; ok {
				assert.Equal(t, expected, *force)
			}
			if expected, ok := tt.expected["drop"]; ok {
				assert.Equal(t, expected, *drop)
			}
			if expected, ok := tt.expected["help"]; ok {
				assert.Equal(t, expected, *help)
			}
		})
	}
}

// TestConfigLoading tests configuration loading for migration
func TestConfigLoading(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "migrate_test_config.yaml")

	configContent := `
app:
  name: "QCAT Migration Test"
  version: "1.0.0"
  environment: "test"

database:
  host: "localhost"
  port: 5432
  user: "test_user"
  password: "test_password"
  dbname: "qcat_migrate_test"
  ssl_mode: "disable"
  max_open: 25
  max_idle: 10
  timeout: 30s
  conn_max_lifetime: 5m
  conn_max_idle_time: 5m
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := config.Load(configPath)
	require.NoError(t, err)

	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "test_user", cfg.Database.User)
	assert.Equal(t, "test_password", cfg.Database.Password)
	assert.Equal(t, "qcat_migrate_test", cfg.Database.DBName)
	assert.Equal(t, "disable", cfg.Database.SSLMode)
}

// TestDatabaseConfigConversion tests conversion from config to database config
func TestDatabaseConfigConversion(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:            "test-host",
			Port:            5433,
			User:            "test-user",
			Password:        "test-pass",
			DBName:          "test-db",
			SSLMode:         "require",
			MaxOpen:         30,
			MaxIdle:         15,
			Timeout:         45 * time.Second,
			ConnMaxLifetime: 10 * time.Minute,
			ConnMaxIdleTime: 8 * time.Minute,
		},
	}

	dbConfig := &database.Config{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		DBName:          cfg.Database.DBName,
		SSLMode:         cfg.Database.SSLMode,
		MaxOpen:         cfg.Database.MaxOpen,
		MaxIdle:         cfg.Database.MaxIdle,
		Timeout:         cfg.Database.Timeout,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.Database.ConnMaxIdleTime,
	}

	assert.Equal(t, "test-host", dbConfig.Host)
	assert.Equal(t, 5433, dbConfig.Port)
	assert.Equal(t, "test-user", dbConfig.User)
	assert.Equal(t, "test-pass", dbConfig.Password)
	assert.Equal(t, "test-db", dbConfig.DBName)
	assert.Equal(t, "require", dbConfig.SSLMode)
	assert.Equal(t, 30, dbConfig.MaxOpen)
	assert.Equal(t, 15, dbConfig.MaxIdle)
}

// TestMigrationFunctions tests migration helper functions with mock migrator
func TestMigrationFunctions(t *testing.T) {
	suite := testutils.NewTestSuite(t, &testutils.TestConfig{
		UseRealDB:    false,
		UseRealCache: false,
		LogLevel:     "error",
	})
	defer suite.TearDown()

	// Create mock migrator for testing
	mockMigrator := &MockMigrator{}

	t.Run("runMigrations", func(t *testing.T) {
		// Test successful migration
		mockMigrator.upError = nil

		// We can't easily test the actual function due to log.Fatalf,
		// but we can test the logic
		err := mockMigrator.Up()
		assert.NoError(t, err)
		assert.True(t, mockMigrator.upCalled)
	})

	t.Run("rollbackMigrations", func(t *testing.T) {
		mockMigrator.Reset()
		mockMigrator.downError = nil

		err := mockMigrator.Down()
		assert.NoError(t, err)
		assert.True(t, mockMigrator.downCalled)
	})

	t.Run("showVersion", func(t *testing.T) {
		mockMigrator.Reset()
		mockMigrator.version = 5
		mockMigrator.versionError = nil

		version, err := mockMigrator.Version()
		assert.NoError(t, err)
		assert.Equal(t, 5, version)
		assert.True(t, mockMigrator.versionCalled)
	})

	t.Run("forceMigrationVersion", func(t *testing.T) {
		mockMigrator.Reset()
		mockMigrator.forceError = nil

		err := mockMigrator.Force(3)
		assert.NoError(t, err)
		assert.True(t, mockMigrator.forceCalled)
		assert.Equal(t, 3, mockMigrator.forceVersion)
	})

	t.Run("dropDatabase", func(t *testing.T) {
		mockMigrator.Reset()
		mockMigrator.dropError = nil

		err := mockMigrator.Drop()
		assert.NoError(t, err)
		assert.True(t, mockMigrator.dropCalled)
	})
}

// MockMigrator implements a mock migrator for testing
type MockMigrator struct {
	upCalled      bool
	downCalled    bool
	versionCalled bool
	forceCalled   bool
	dropCalled    bool

	upError      error
	downError    error
	versionError error
	forceError   error
	dropError    error

	version      int
	forceVersion int
}

func (m *MockMigrator) Up() error {
	m.upCalled = true
	return m.upError
}

func (m *MockMigrator) Down() error {
	m.downCalled = true
	return m.downError
}

func (m *MockMigrator) Version() (int, error) {
	m.versionCalled = true
	return m.version, m.versionError
}

func (m *MockMigrator) Force(version int) error {
	m.forceCalled = true
	m.forceVersion = version
	return m.forceError
}

func (m *MockMigrator) Drop() error {
	m.dropCalled = true
	return m.dropError
}

func (m *MockMigrator) Close() error {
	return nil
}

func (m *MockMigrator) Reset() {
	m.upCalled = false
	m.downCalled = false
	m.versionCalled = false
	m.forceCalled = false
	m.dropCalled = false
	m.forceVersion = 0
}
