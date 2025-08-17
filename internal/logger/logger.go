package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LogLevel 日志级别
type LogLevel string

const (
	LevelTrace LogLevel = "trace"
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
	LevelFatal LogLevel = "fatal"
	LevelPanic LogLevel = "panic"
)

// LogFormat 日志格式
type LogFormat string

const (
	FormatJSON LogFormat = "json"
	FormatText LogFormat = "text"
)

// Config 日志配置
type Config struct {
	Level      LogLevel  `yaml:"level" json:"level"`
	Format     LogFormat `yaml:"format" json:"format"`
	Output     string    `yaml:"output" json:"output"`           // stdout, stderr, file
	Filename   string    `yaml:"filename" json:"filename"`       // 日志文件路径
	MaxSize    int       `yaml:"max_size" json:"max_size"`       // 单个日志文件最大大小(MB)
	MaxAge     int       `yaml:"max_age" json:"max_age"`         // 日志文件保留天数
	MaxBackups int       `yaml:"max_backups" json:"max_backups"` // 最大备份文件数
	Compress   bool      `yaml:"compress" json:"compress"`       // 是否压缩备份文件
	Caller     bool      `yaml:"caller" json:"caller"`           // 是否显示调用者信息
	Timestamp  bool      `yaml:"timestamp" json:"timestamp"`     // 是否显示时间戳
}

// DefaultConfig 默认配置
var DefaultConfig = Config{
	Level:      LevelInfo,
	Format:     FormatJSON,
	Output:     "stdout",
	MaxSize:    100,
	MaxAge:     30,
	MaxBackups: 10,
	Compress:   true,
	Caller:     true,
	Timestamp:  true,
}

// Logger 日志器接口
type Logger interface {
	Trace(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Fatal(msg string, fields ...interface{})
	Panic(msg string, fields ...interface{})
	
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
	WithContext(ctx context.Context) Logger
	
	SetLevel(level LogLevel)
	GetLevel() LogLevel
}

// StructuredLogger 结构化日志器
type StructuredLogger struct {
	logger *logrus.Logger
	entry  *logrus.Entry
	config Config
	mu     sync.RWMutex
}

// NewLogger 创建新的日志器
func NewLogger(config Config) Logger {
	logger := logrus.New()
	
	// 设置日志级别
	level, err := logrus.ParseLevel(string(config.Level))
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)
	
	// 设置日志格式
	if config.Format == FormatJSON {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				filename := filepath.Base(f.File)
				return fmt.Sprintf("%s()", f.Function), fmt.Sprintf("%s:%d", filename, f.Line)
			},
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   config.Timestamp,
			TimestampFormat: time.RFC3339,
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				filename := filepath.Base(f.File)
				return fmt.Sprintf("%s()", f.Function), fmt.Sprintf("%s:%d", filename, f.Line)
			},
		})
	}
	
	// 设置输出
	var output io.Writer
	switch config.Output {
	case "stderr":
		output = os.Stderr
	case "file":
		if config.Filename == "" {
			config.Filename = "logs/app.log"
		}
		
		// 确保日志目录存在
		if err := os.MkdirAll(filepath.Dir(config.Filename), 0755); err != nil {
			fmt.Printf("Failed to create log directory: %v\n", err)
			output = os.Stdout
		} else {
			output = &lumberjack.Logger{
				Filename:   config.Filename,
				MaxSize:    config.MaxSize,
				MaxAge:     config.MaxAge,
				MaxBackups: config.MaxBackups,
				Compress:   config.Compress,
			}
		}
	default:
		output = os.Stdout
	}
	
	logger.SetOutput(output)
	
	// 设置调用者信息
	logger.SetReportCaller(config.Caller)
	
	return &StructuredLogger{
		logger: logger,
		entry:  logrus.NewEntry(logger),
		config: config,
	}
}

// Trace 记录trace级别日志
func (l *StructuredLogger) Trace(msg string, fields ...interface{}) {
	l.logWithFields(logrus.TraceLevel, msg, fields...)
}

// Debug 记录debug级别日志
func (l *StructuredLogger) Debug(msg string, fields ...interface{}) {
	l.logWithFields(logrus.DebugLevel, msg, fields...)
}

// Info 记录info级别日志
func (l *StructuredLogger) Info(msg string, fields ...interface{}) {
	l.logWithFields(logrus.InfoLevel, msg, fields...)
}

// Warn 记录warn级别日志
func (l *StructuredLogger) Warn(msg string, fields ...interface{}) {
	l.logWithFields(logrus.WarnLevel, msg, fields...)
}

// Error 记录error级别日志
func (l *StructuredLogger) Error(msg string, fields ...interface{}) {
	l.logWithFields(logrus.ErrorLevel, msg, fields...)
}

// Fatal 记录fatal级别日志
func (l *StructuredLogger) Fatal(msg string, fields ...interface{}) {
	l.logWithFields(logrus.FatalLevel, msg, fields...)
}

// Panic 记录panic级别日志
func (l *StructuredLogger) Panic(msg string, fields ...interface{}) {
	l.logWithFields(logrus.PanicLevel, msg, fields...)
}

// WithField 添加单个字段
func (l *StructuredLogger) WithField(key string, value interface{}) Logger {
	return &StructuredLogger{
		logger: l.logger,
		entry:  l.entry.WithField(key, value),
		config: l.config,
	}
}

// WithFields 添加多个字段
func (l *StructuredLogger) WithFields(fields map[string]interface{}) Logger {
	return &StructuredLogger{
		logger: l.logger,
		entry:  l.entry.WithFields(fields),
		config: l.config,
	}
}

// WithContext 添加上下文
func (l *StructuredLogger) WithContext(ctx context.Context) Logger {
	entry := l.entry.WithContext(ctx)
	
	// 从上下文中提取常用字段
	if requestID := ctx.Value("request_id"); requestID != nil {
		entry = entry.WithField("request_id", requestID)
	}
	if userID := ctx.Value("user_id"); userID != nil {
		entry = entry.WithField("user_id", userID)
	}
	if traceID := ctx.Value("trace_id"); traceID != nil {
		entry = entry.WithField("trace_id", traceID)
	}
	
	return &StructuredLogger{
		logger: l.logger,
		entry:  entry,
		config: l.config,
	}
}

// SetLevel 设置日志级别
func (l *StructuredLogger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	logrusLevel, err := logrus.ParseLevel(string(level))
	if err != nil {
		return
	}
	
	l.logger.SetLevel(logrusLevel)
	l.config.Level = level
}

// GetLevel 获取日志级别
func (l *StructuredLogger) GetLevel() LogLevel {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	return l.config.Level
}

// logWithFields 记录带字段的日志
func (l *StructuredLogger) logWithFields(level logrus.Level, msg string, fields ...interface{}) {
	entry := l.entry
	
	// 处理字段参数
	if len(fields) > 0 {
		fieldMap := make(map[string]interface{})
		for i := 0; i < len(fields)-1; i += 2 {
			if key, ok := fields[i].(string); ok && i+1 < len(fields) {
				fieldMap[key] = fields[i+1]
			}
		}
		if len(fieldMap) > 0 {
			entry = entry.WithFields(fieldMap)
		}
	}
	
	entry.Log(level, msg)
}

// 全局日志器实例
var globalLogger Logger

// 初始化全局日志器
func init() {
	globalLogger = NewLogger(DefaultConfig)
}

// Init 初始化日志器
func Init(config Config) {
	globalLogger = NewLogger(config)
}

// SetGlobalLogger 设置全局日志器
func SetGlobalLogger(logger Logger) {
	globalLogger = logger
}

// GetGlobalLogger 获取全局日志器
func GetGlobalLogger() Logger {
	return globalLogger
}

// 全局日志函数

// Trace 记录trace级别日志
func Trace(msg string, fields ...interface{}) {
	globalLogger.Trace(msg, fields...)
}

// Debug 记录debug级别日志
func Debug(msg string, fields ...interface{}) {
	globalLogger.Debug(msg, fields...)
}

// Info 记录info级别日志
func Info(msg string, fields ...interface{}) {
	globalLogger.Info(msg, fields...)
}

// Warn 记录warn级别日志
func Warn(msg string, fields ...interface{}) {
	globalLogger.Warn(msg, fields...)
}

// Error 记录error级别日志
func Error(msg string, fields ...interface{}) {
	globalLogger.Error(msg, fields...)
}

// Fatal 记录fatal级别日志
func Fatal(msg string, fields ...interface{}) {
	globalLogger.Fatal(msg, fields...)
}

// Panic 记录panic级别日志
func Panic(msg string, fields ...interface{}) {
	globalLogger.Panic(msg, fields...)
}

// WithField 添加单个字段
func WithField(key string, value interface{}) Logger {
	return globalLogger.WithField(key, value)
}

// WithFields 添加多个字段
func WithFields(fields map[string]interface{}) Logger {
	return globalLogger.WithFields(fields)
}

// WithContext 添加上下文
func WithContext(ctx context.Context) Logger {
	return globalLogger.WithContext(ctx)
}

// SetLevel 设置日志级别
func SetLevel(level LogLevel) {
	globalLogger.SetLevel(level)
}

// GetLevel 获取日志级别
func GetLevel() LogLevel {
	return globalLogger.GetLevel()
}

// HTTPRequestInfo HTTP请求信息
type HTTPRequestInfo struct {
	Method     string
	Path       string
	StatusCode int
	Latency    time.Duration
	ClientIP   string
	UserAgent  string
	BodySize   int64
	RequestID  string
	UserID     string
	Headers    map[string]string
}

// LogHTTPRequest 记录HTTP请求日志
func LogHTTPRequest(info HTTPRequestInfo) {
	fields := map[string]interface{}{
		"method":      info.Method,
		"path":        info.Path,
		"status_code": info.StatusCode,
		"latency":     info.Latency.String(),
		"client_ip":   info.ClientIP,
		"user_agent":  info.UserAgent,
		"body_size":   info.BodySize,
	}
	
	if info.RequestID != "" {
		fields["request_id"] = info.RequestID
	}
	
	if info.UserID != "" {
		fields["user_id"] = info.UserID
	}
	
	// 添加额外的头信息
	for k, v := range info.Headers {
		fields["header_"+strings.ToLower(k)] = v
	}
	
	msg := fmt.Sprintf("%s %s - %d", info.Method, info.Path, info.StatusCode)
	
	// 根据状态码选择日志级别
	if info.StatusCode >= 500 {
		WithFields(fields).Error(msg)
	} else if info.StatusCode >= 400 {
		WithFields(fields).Warn(msg)
	} else {
		WithFields(fields).Info(msg)
	}
}

// RequestLogger 请求日志记录器
type RequestLogger struct {
	logger Logger
}

// NewRequestLogger 创建请求日志记录器
func NewRequestLogger(logger Logger) *RequestLogger {
	return &RequestLogger{logger: logger}
}

// LogRequest 记录请求日志
func (rl *RequestLogger) LogRequest(method, path string, statusCode int, latency time.Duration, fields map[string]interface{}) {
	msg := fmt.Sprintf("%s %s - %d", method, path, statusCode)
	
	logFields := map[string]interface{}{
		"method":      method,
		"path":        path,
		"status_code": statusCode,
		"latency":     latency.String(),
	}
	
	// 合并额外字段
	for k, v := range fields {
		logFields[k] = v
	}
	
	// 根据状态码选择日志级别
	if statusCode >= 500 {
		rl.logger.WithFields(logFields).Error(msg)
	} else if statusCode >= 400 {
		rl.logger.WithFields(logFields).Warn(msg)
	} else {
		rl.logger.WithFields(logFields).Info(msg)
	}
}

// PerformanceLogger 性能日志记录器
type PerformanceLogger struct {
	logger Logger
}

// NewPerformanceLogger 创建性能日志记录器
func NewPerformanceLogger(logger Logger) *PerformanceLogger {
	return &PerformanceLogger{logger: logger}
}

// LogPerformance 记录性能日志
func (pl *PerformanceLogger) LogPerformance(operation string, duration time.Duration, fields map[string]interface{}) {
	logFields := map[string]interface{}{
		"operation": operation,
		"duration":  duration.String(),
		"duration_ms": duration.Milliseconds(),
	}
	
	// 合并额外字段
	for k, v := range fields {
		logFields[k] = v
	}
	
	msg := fmt.Sprintf("Performance: %s took %s", operation, duration.String())
	
	// 根据耗时选择日志级别
	if duration > 5*time.Second {
		pl.logger.WithFields(logFields).Error(msg)
	} else if duration > 1*time.Second {
		pl.logger.WithFields(logFields).Warn(msg)
	} else {
		pl.logger.WithFields(logFields).Info(msg)
	}
}

// AuditLogger 审计日志记录器
type AuditLogger struct {
	logger Logger
}

// NewAuditLogger 创建审计日志记录器
func NewAuditLogger(logger Logger) *AuditLogger {
	return &AuditLogger{logger: logger}
}

// LogAudit 记录审计日志
func (al *AuditLogger) LogAudit(userID, action, resource string, result bool, details map[string]interface{}) {
	logFields := map[string]interface{}{
		"user_id":  userID,
		"action":   action,
		"resource": resource,
		"result":   result,
		"audit":    true, // 标记为审计日志
	}
	
	// 合并详细信息
	for k, v := range details {
		logFields[k] = v
	}
	
	msg := fmt.Sprintf("Audit: %s %s %s", userID, action, resource)
	if result {
		msg += " - SUCCESS"
	} else {
		msg += " - FAILED"
	}
	
	al.logger.WithFields(logFields).Info(msg)
}

// SecurityLogger 安全日志记录器
type SecurityLogger struct {
	logger Logger
}

// NewSecurityLogger 创建安全日志记录器
func NewSecurityLogger(logger Logger) *SecurityLogger {
	return &SecurityLogger{logger: logger}
}

// LogSecurity 记录安全日志
func (sl *SecurityLogger) LogSecurity(event, userID, ip string, severity string, details map[string]interface{}) {
	logFields := map[string]interface{}{
		"event":     event,
		"user_id":   userID,
		"ip":        ip,
		"severity":  severity,
		"security":  true, // 标记为安全日志
	}
	
	// 合并详细信息
	for k, v := range details {
		logFields[k] = v
	}
	
	msg := fmt.Sprintf("Security: %s from %s", event, ip)
	
	// 根据严重程度选择日志级别
	switch strings.ToLower(severity) {
	case "critical", "high":
		sl.logger.WithFields(logFields).Error(msg)
	case "medium":
		sl.logger.WithFields(logFields).Warn(msg)
	default:
		sl.logger.WithFields(logFields).Info(msg)
	}
}