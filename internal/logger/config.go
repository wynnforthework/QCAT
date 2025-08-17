package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

// LoggerConfig 完整的日志配置结构
type LoggerConfig struct {
	Logger       Config                    `yaml:"logger"`
	Environments map[string]EnvironmentConfig `yaml:"environments"`
	Modules      map[string]ModuleConfig   `yaml:"modules"`
	Rotation     RotationConfig            `yaml:"rotation"`
	Filters      FiltersConfig             `yaml:"filters"`
	Sampling     SamplingConfig            `yaml:"sampling"`
	Buffering    BufferingConfig           `yaml:"buffering"`
	Monitoring   MonitoringConfig          `yaml:"monitoring"`
	Query        QueryConfig               `yaml:"query"`
}

// EnvironmentConfig 环境特定配置
type EnvironmentConfig struct {
	Logger Config `yaml:"logger"`
}

// ModuleConfig 模块特定配置
type ModuleConfig struct {
	Level        LogLevel `yaml:"level"`
	SeparateFile bool     `yaml:"separate_file"`
	Filename     string   `yaml:"filename"`
}

// RotationConfig 日志轮转配置
type RotationConfig struct {
	Strategy            string `yaml:"strategy"`
	SizeThreshold       int    `yaml:"size_threshold"`
	TimeInterval        string `yaml:"time_interval"`
	RotationTime        string `yaml:"rotation_time"`
	RotateImmediately   bool   `yaml:"rotate_immediately"`
}

// FiltersConfig 日志过滤配置
type FiltersConfig struct {
	SensitiveFields     []string            `yaml:"sensitive_fields"`
	ExcludePaths        []string            `yaml:"exclude_paths"`
	ExcludeUserAgents   []string            `yaml:"exclude_user_agents"`
	MinLevelByPath      map[string]string   `yaml:"min_level_by_path"`
}

// SamplingConfig 日志采样配置
type SamplingConfig struct {
	Enabled       bool        `yaml:"enabled"`
	Rate          float64     `yaml:"rate"`
	Strategy      string      `yaml:"strategy"`
	ExcludeLevels []LogLevel  `yaml:"exclude_levels"`
}

// BufferingConfig 日志缓冲配置
type BufferingConfig struct {
	Enabled           bool        `yaml:"enabled"`
	BufferSize        int         `yaml:"buffer_size"`
	FlushInterval     int         `yaml:"flush_interval"`
	ForceFlushLevels  []LogLevel  `yaml:"force_flush_levels"`
}

// MonitoringConfig 日志监控配置
type MonitoringConfig struct {
	Enabled bool                    `yaml:"enabled"`
	Metrics []string                `yaml:"metrics"`
	Alerts  MonitoringAlertsConfig  `yaml:"alerts"`
}

// MonitoringAlertsConfig 监控告警配置
type MonitoringAlertsConfig struct {
	ErrorRateThreshold   float64 `yaml:"error_rate_threshold"`
	LogRateThreshold     int     `yaml:"log_rate_threshold"`
	DiskUsageThreshold   float64 `yaml:"disk_usage_threshold"`
}

// QueryConfig 日志查询配置
type QueryConfig struct {
	Enabled          bool     `yaml:"enabled"`
	Endpoint         string   `yaml:"endpoint"`
	MaxResults       int      `yaml:"max_results"`
	Timeout          int      `yaml:"timeout"`
	SearchableFields []string `yaml:"searchable_fields"`
	Indexes          []IndexConfig `yaml:"indexes"`
}

// IndexConfig 索引配置
type IndexConfig struct {
	Field string `yaml:"field"`
	Type  string `yaml:"type"`
}

// LoadConfig 从文件加载日志配置
func LoadConfig(configPath string) (*LoggerConfig, error) {
	// 检查文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 解析YAML配置
	var config LoggerConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 验证配置
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// 应用默认值
	applyDefaults(&config)

	return &config, nil
}

// LoadConfigForEnvironment 为特定环境加载配置
func LoadConfigForEnvironment(configPath, environment string) (*Config, error) {
	loggerConfig, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	// 获取基础配置
	config := loggerConfig.Logger

	// 应用环境特定配置
	if envConfig, exists := loggerConfig.Environments[environment]; exists {
		mergeConfigs(&config, &envConfig.Logger)
	}

	return &config, nil
}

// GetModuleConfig 获取模块特定配置
func GetModuleConfig(loggerConfig *LoggerConfig, moduleName string) *ModuleConfig {
	if moduleConfig, exists := loggerConfig.Modules[moduleName]; exists {
		return &moduleConfig
	}
	return nil
}

// validateConfig 验证配置的有效性
func validateConfig(config *LoggerConfig) error {
	// 验证日志级别
	validLevels := map[LogLevel]bool{
		LevelTrace: true,
		LevelDebug: true,
		LevelInfo:  true,
		LevelWarn:  true,
		LevelError: true,
		LevelFatal: true,
		LevelPanic: true,
	}

	if !validLevels[config.Logger.Level] {
		return fmt.Errorf("invalid log level: %s", config.Logger.Level)
	}

	// 验证日志格式
	validFormats := map[LogFormat]bool{
		FormatJSON: true,
		FormatText: true,
	}

	if !validFormats[config.Logger.Format] {
		return fmt.Errorf("invalid log format: %s", config.Logger.Format)
	}

	// 验证输出目标
	validOutputs := map[string]bool{
		"stdout": true,
		"stderr": true,
		"file":   true,
	}

	if !validOutputs[config.Logger.Output] {
		return fmt.Errorf("invalid output target: %s", config.Logger.Output)
	}

	// 如果输出到文件，检查文件名
	if config.Logger.Output == "file" && config.Logger.Filename == "" {
		return fmt.Errorf("filename is required when output is 'file'")
	}

	// 验证轮转配置
	if config.Rotation.Strategy != "" {
		validStrategies := map[string]bool{
			"size": true,
			"time": true,
			"both": true,
		}
		if !validStrategies[config.Rotation.Strategy] {
			return fmt.Errorf("invalid rotation strategy: %s", config.Rotation.Strategy)
		}
	}

	// 验证采样配置
	if config.Sampling.Enabled {
		if config.Sampling.Rate < 0 || config.Sampling.Rate > 1 {
			return fmt.Errorf("sampling rate must be between 0 and 1")
		}
	}

	return nil
}

// applyDefaults 应用默认配置值
func applyDefaults(config *LoggerConfig) {
	// 应用基础配置默认值
	if config.Logger.Level == "" {
		config.Logger.Level = LevelInfo
	}
	if config.Logger.Format == "" {
		config.Logger.Format = FormatJSON
	}
	if config.Logger.Output == "" {
		config.Logger.Output = "stdout"
	}
	if config.Logger.MaxSize == 0 {
		config.Logger.MaxSize = 100
	}
	if config.Logger.MaxAge == 0 {
		config.Logger.MaxAge = 30
	}
	if config.Logger.MaxBackups == 0 {
		config.Logger.MaxBackups = 10
	}

	// 应用轮转配置默认值
	if config.Rotation.Strategy == "" {
		config.Rotation.Strategy = "both"
	}
	if config.Rotation.SizeThreshold == 0 {
		config.Rotation.SizeThreshold = 100
	}
	if config.Rotation.TimeInterval == "" {
		config.Rotation.TimeInterval = "daily"
	}
	if config.Rotation.RotationTime == "" {
		config.Rotation.RotationTime = "00:00"
	}

	// 应用缓冲配置默认值
	if config.Buffering.BufferSize == 0 {
		config.Buffering.BufferSize = 1000
	}
	if config.Buffering.FlushInterval == 0 {
		config.Buffering.FlushInterval = 5
	}

	// 应用查询配置默认值
	if config.Query.Endpoint == "" {
		config.Query.Endpoint = "/api/v1/logs"
	}
	if config.Query.MaxResults == 0 {
		config.Query.MaxResults = 1000
	}
	if config.Query.Timeout == 0 {
		config.Query.Timeout = 30
	}
}

// mergeConfigs 合并配置
func mergeConfigs(base *Config, override *Config) {
	if override.Level != "" {
		base.Level = override.Level
	}
	if override.Format != "" {
		base.Format = override.Format
	}
	if override.Output != "" {
		base.Output = override.Output
	}
	if override.Filename != "" {
		base.Filename = override.Filename
	}
	if override.MaxSize != 0 {
		base.MaxSize = override.MaxSize
	}
	if override.MaxAge != 0 {
		base.MaxAge = override.MaxAge
	}
	if override.MaxBackups != 0 {
		base.MaxBackups = override.MaxBackups
	}
	// 布尔值需要特殊处理，因为false是零值
	base.Compress = override.Compress
	base.Caller = override.Caller
	base.Timestamp = override.Timestamp
}

// CreateLoggerFromConfig 从配置创建日志器
func CreateLoggerFromConfig(config *Config) Logger {
	return NewLogger(*config)
}

// CreateModuleLogger 为特定模块创建日志器
func CreateModuleLogger(loggerConfig *LoggerConfig, moduleName string) Logger {
	// 获取基础配置
	config := loggerConfig.Logger

	// 应用模块特定配置
	if moduleConfig := GetModuleConfig(loggerConfig, moduleName); moduleConfig != nil {
		if moduleConfig.Level != "" {
			config.Level = moduleConfig.Level
		}
		if moduleConfig.SeparateFile && moduleConfig.Filename != "" {
			config.Output = "file"
			config.Filename = moduleConfig.Filename
		}
	}

	logger := NewLogger(config)
	return logger.WithField("module", moduleName)
}

// GetConfigPath 获取配置文件路径
func GetConfigPath() string {
	// 优先使用环境变量
	if path := os.Getenv("QCAT_LOG_CONFIG"); path != "" {
		return path
	}

	// 检查常见位置
	possiblePaths := []string{
		"configs/logger.yaml",
		"config/logger.yaml",
		"/etc/qcat/logger.yaml",
		filepath.Join(os.Getenv("HOME"), ".qcat", "logger.yaml"),
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// 返回默认路径
	return "configs/logger.yaml"
}

// InitFromConfig 从配置文件初始化全局日志器
func InitFromConfig(configPath string) error {
	config, err := LoadConfigForEnvironment(configPath, getEnvironment())
	if err != nil {
		return fmt.Errorf("failed to load logger config: %w", err)
	}

	Init(*config)
	return nil
}

// InitFromConfigWithEnvironment 从配置文件为特定环境初始化全局日志器
func InitFromConfigWithEnvironment(configPath, environment string) error {
	config, err := LoadConfigForEnvironment(configPath, environment)
	if err != nil {
		return fmt.Errorf("failed to load logger config: %w", err)
	}

	Init(*config)
	return nil
}

// getEnvironment 获取当前环境
func getEnvironment() string {
	env := os.Getenv("QCAT_ENV")
	if env == "" {
		env = os.Getenv("GO_ENV")
	}
	if env == "" {
		env = "development"
	}
	return strings.ToLower(env)
}

// SaveConfig 保存配置到文件
func SaveConfig(config *LoggerConfig, configPath string) error {
	// 确保目录存在
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// 序列化配置
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ReloadConfig 重新加载配置
func ReloadConfig(configPath string) error {
	config, err := LoadConfigForEnvironment(configPath, getEnvironment())
	if err != nil {
		return err
	}

	// 更新全局日志器配置
	globalLogger.SetLevel(config.Level)
	return nil
}