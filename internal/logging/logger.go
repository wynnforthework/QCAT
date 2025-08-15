package logging

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

// Logger represents a structured logger
type Logger struct {
	logger *logrus.Logger
	fields logrus.Fields
	mu     sync.RWMutex
}

// LogConfig represents logging configuration
type LogConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	Output     string `yaml:"output"`
	MaxSize    int    `yaml:"max_size"` // MB
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"` // days
	Compress   bool   `yaml:"compress"`
	LogDir     string `yaml:"log_dir"`
}

// NewLogger creates a new structured logger
func NewLogger(config *LogConfig) (*Logger, error) {
	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}
	logger.SetLevel(level)

	// Set log format
	switch strings.ToLower(config.Format) {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
			},
		})
	case "text":
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339Nano,
		})
	default:
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		})
	}

	// Set output
	if err := setLogOutput(logger, config); err != nil {
		return nil, err
	}

	return &Logger{
		logger: logger,
		fields: make(logrus.Fields),
	}, nil
}

// setLogOutput sets the log output based on configuration
func setLogOutput(logger *logrus.Logger, config *LogConfig) error {
	switch strings.ToLower(config.Output) {
	case "stdout":
		logger.SetOutput(os.Stdout)
	case "stderr":
		logger.SetOutput(os.Stderr)
	case "file":
		if config.LogDir == "" {
			config.LogDir = "logs"
		}

		// Create log directory if it doesn't exist
		if err := os.MkdirAll(config.LogDir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}

		// Set up log rotation
		logFile := filepath.Join(config.LogDir, "qcat.log")
		writer := &lumberjack.Logger{
			Filename:   logFile,
			MaxSize:    config.MaxSize, // MB
			MaxBackups: config.MaxBackups,
			MaxAge:     config.MaxAge, // days
			Compress:   config.Compress,
		}

		// Use multi-writer to log to both file and stdout in development
		if config.Level == "debug" {
			logger.SetOutput(io.MultiWriter(writer, os.Stdout))
		} else {
			logger.SetOutput(writer)
		}
	default:
		logger.SetOutput(os.Stdout)
	}

	return nil
}

// WithField adds a field to the logger
func (l *Logger) WithField(key string, value interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newFields := make(logrus.Fields, len(l.fields)+1)
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value

	return &Logger{
		logger: l.logger,
		fields: newFields,
	}
}

// WithFields adds multiple fields to the logger
func (l *Logger) WithFields(fields logrus.Fields) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newFields := make(logrus.Fields, len(l.fields)+len(fields))
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &Logger{
		logger: l.logger,
		fields: newFields,
	}
}

// WithContext adds context information to the logger
func (l *Logger) WithContext(ctx context.Context) *Logger {
	if ctx == nil {
		return l
	}

	fields := make(logrus.Fields)

	// Add request ID if available
	if requestID, ok := ctx.Value("request_id").(string); ok {
		fields["request_id"] = requestID
	}

	// Add user ID if available
	if userID, ok := ctx.Value("user_id").(string); ok {
		fields["user_id"] = userID
	}

	// Add trace ID if available
	if traceID, ok := ctx.Value("trace_id").(string); ok {
		fields["trace_id"] = traceID
	}

	return l.WithFields(fields)
}

// WithError adds error information to the logger
func (l *Logger) WithError(err error) *Logger {
	if err == nil {
		return l
	}

	fields := logrus.Fields{
		"error": err.Error(),
	}

	// Add stack trace for errors
	if _, file, line, ok := runtime.Caller(1); ok {
		fields["file"] = file
		fields["line"] = line
	}

	return l.WithFields(fields)
}

// Log methods
func (l *Logger) Debug(args ...interface{}) {
	l.logger.WithFields(l.fields).Debug(args...)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.logger.WithFields(l.fields).Debugf(format, args...)
}

func (l *Logger) Info(args ...interface{}) {
	l.logger.WithFields(l.fields).Info(args...)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.logger.WithFields(l.fields).Infof(format, args...)
}

func (l *Logger) Warn(args ...interface{}) {
	l.logger.WithFields(l.fields).Warn(args...)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.logger.WithFields(l.fields).Warnf(format, args...)
}

func (l *Logger) Error(args ...interface{}) {
	l.logger.WithFields(l.fields).Error(args...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.logger.WithFields(l.fields).Errorf(format, args...)
}

func (l *Logger) Fatal(args ...interface{}) {
	l.logger.WithFields(l.fields).Fatal(args...)
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.logger.WithFields(l.fields).Fatalf(format, args...)
}

// LogWithLevel logs with a specific level
func (l *Logger) LogWithLevel(level logrus.Level, args ...interface{}) {
	l.logger.WithFields(l.fields).Log(level, args...)
}

// LogWithLevelf logs with a specific level and format
func (l *Logger) LogWithLevelf(level logrus.Level, format string, args ...interface{}) {
	l.logger.WithFields(l.fields).Logf(level, format, args...)
}

// GetLogger returns the underlying logrus logger
func (l *Logger) GetLogger() *logrus.Logger {
	return l.logger
}

// SetLevel sets the log level
func (l *Logger) SetLevel(level logrus.Level) {
	l.logger.SetLevel(level)
}

// SetFormatter sets the log formatter
func (l *Logger) SetFormatter(formatter logrus.Formatter) {
	l.logger.SetFormatter(formatter)
}

// SetOutput sets the log output
func (l *Logger) SetOutput(output io.Writer) {
	l.logger.SetOutput(output)
}

// Global logger instance
var globalLogger *Logger
var globalLoggerMu sync.RWMutex

// SetGlobalLogger sets the global logger instance
func SetGlobalLogger(logger *Logger) {
	globalLoggerMu.Lock()
	defer globalLoggerMu.Unlock()
	globalLogger = logger
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *Logger {
	globalLoggerMu.RLock()
	defer globalLoggerMu.RUnlock()
	return globalLogger
}

// Global logging functions
func Debug(args ...interface{}) {
	if globalLogger != nil {
		globalLogger.Debug(args...)
	}
}

func Debugf(format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.Debugf(format, args...)
	}
}

func Info(args ...interface{}) {
	if globalLogger != nil {
		globalLogger.Info(args...)
	}
}

func Infof(format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.Infof(format, args...)
	}
}

func Warn(args ...interface{}) {
	if globalLogger != nil {
		globalLogger.Warn(args...)
	}
}

func Warnf(format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.Warnf(format, args...)
	}
}

func Error(args ...interface{}) {
	if globalLogger != nil {
		globalLogger.Error(args...)
	}
}

func Errorf(format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.Errorf(format, args...)
	}
}

func Fatal(args ...interface{}) {
	if globalLogger != nil {
		globalLogger.Fatal(args...)
	}
}

func Fatalf(format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.Fatalf(format, args...)
	}
}

// WithField adds a field to the global logger
func WithField(key string, value interface{}) *Logger {
	if globalLogger != nil {
		return globalLogger.WithField(key, value)
	}
	return &Logger{}
}

// WithFields adds multiple fields to the global logger
func WithFields(fields logrus.Fields) *Logger {
	if globalLogger != nil {
		return globalLogger.WithFields(fields)
	}
	return &Logger{}
}

// WithContext adds context to the global logger
func WithContext(ctx context.Context) *Logger {
	if globalLogger != nil {
		return globalLogger.WithContext(ctx)
	}
	return &Logger{}
}

// WithError adds error to the global logger
func WithError(err error) *Logger {
	if globalLogger != nil {
		return globalLogger.WithError(err)
	}
	return &Logger{}
}
