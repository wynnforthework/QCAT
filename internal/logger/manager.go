package logger

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// LogManager 日志管理器
type LogManager struct {
	config     Config
	loggers    map[string]Logger
	rotators   map[string]*LogRotator
	cleaners   map[string]*LogCleaner
	mu         sync.RWMutex
	stopChan   chan struct{}
	wg         sync.WaitGroup
}

// NewLogManager 创建日志管理器
func NewLogManager(config Config) *LogManager {
	return &LogManager{
		config:   config,
		loggers:  make(map[string]Logger),
		rotators: make(map[string]*LogRotator),
		cleaners: make(map[string]*LogCleaner),
		stopChan: make(chan struct{}),
	}
}

// Start 启动日志管理器
func (lm *LogManager) Start() error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// 启动日志轮转器
	for name, rotator := range lm.rotators {
		lm.wg.Add(1)
		go func(n string, r *LogRotator) {
			defer lm.wg.Done()
			r.Start(lm.stopChan)
		}(name, rotator)
	}

	// 启动日志清理器
	for name, cleaner := range lm.cleaners {
		lm.wg.Add(1)
		go func(n string, c *LogCleaner) {
			defer lm.wg.Done()
			c.Start(lm.stopChan)
		}(name, cleaner)
	}

	return nil
}

// Stop 停止日志管理器
func (lm *LogManager) Stop() error {
	close(lm.stopChan)
	lm.wg.Wait()
	return nil
}

// GetLogger 获取指定名称的日志器
func (lm *LogManager) GetLogger(name string) Logger {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	if logger, exists := lm.loggers[name]; exists {
		return logger
	}

	// 返回默认日志器
	return globalLogger
}

// AddLogger 添加日志器
func (lm *LogManager) AddLogger(name string, logger Logger) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lm.loggers[name] = logger
}

// AddRotator 添加日志轮转器
func (lm *LogManager) AddRotator(name string, rotator *LogRotator) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lm.rotators[name] = rotator
}

// AddCleaner 添加日志清理器
func (lm *LogManager) AddCleaner(name string, cleaner *LogCleaner) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lm.cleaners[name] = cleaner
}

// LogRotator 日志轮转器
type LogRotator struct {
	filename      string
	maxSize       int64
	maxAge        time.Duration
	rotateTime    string
	checkInterval time.Duration
	mu            sync.Mutex
}

// NewLogRotator 创建日志轮转器
func NewLogRotator(filename string, maxSize int64, maxAge time.Duration, rotateTime string) *LogRotator {
	return &LogRotator{
		filename:      filename,
		maxSize:       maxSize,
		maxAge:        maxAge,
		rotateTime:    rotateTime,
		checkInterval: 1 * time.Minute,
	}
}

// Start 启动日志轮转器
func (lr *LogRotator) Start(stopChan <-chan struct{}) {
	ticker := time.NewTicker(lr.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			lr.checkAndRotate()
		case <-stopChan:
			return
		}
	}
}

// checkAndRotate 检查并执行日志轮转
func (lr *LogRotator) checkAndRotate() {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	// 检查文件是否存在
	info, err := os.Stat(lr.filename)
	if err != nil {
		return
	}

	shouldRotate := false

	// 检查文件大小
	if lr.maxSize > 0 && info.Size() >= lr.maxSize {
		shouldRotate = true
	}

	// 检查时间
	if lr.rotateTime != "" {
		now := time.Now()
		rotateTime, err := time.Parse("15:04", lr.rotateTime)
		if err == nil {
			today := time.Date(now.Year(), now.Month(), now.Day(), rotateTime.Hour(), rotateTime.Minute(), 0, 0, now.Location())
			if now.After(today) && info.ModTime().Before(today) {
				shouldRotate = true
			}
		}
	}

	if shouldRotate {
		lr.rotate()
	}
}

// rotate 执行日志轮转
func (lr *LogRotator) rotate() {
	// 生成轮转后的文件名
	timestamp := time.Now().Format("20060102-150405")
	rotatedFilename := fmt.Sprintf("%s.%s", lr.filename, timestamp)

	// 重命名当前日志文件
	if err := os.Rename(lr.filename, rotatedFilename); err != nil {
		fmt.Printf("Failed to rotate log file: %v\n", err)
		return
	}

	// 创建新的日志文件
	file, err := os.Create(lr.filename)
	if err != nil {
		fmt.Printf("Failed to create new log file: %v\n", err)
		return
	}
	file.Close()

	fmt.Printf("Log file rotated: %s -> %s\n", lr.filename, rotatedFilename)
}

// LogCleaner 日志清理器
type LogCleaner struct {
	logDir        string
	maxAge        time.Duration
	maxFiles      int
	checkInterval time.Duration
	pattern       string
	mu            sync.Mutex
}

// NewLogCleaner 创建日志清理器
func NewLogCleaner(logDir string, maxAge time.Duration, maxFiles int, pattern string) *LogCleaner {
	return &LogCleaner{
		logDir:        logDir,
		maxAge:        maxAge,
		maxFiles:      maxFiles,
		checkInterval: 1 * time.Hour,
		pattern:       pattern,
	}
}

// Start 启动日志清理器
func (lc *LogCleaner) Start(stopChan <-chan struct{}) {
	ticker := time.NewTicker(lc.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			lc.cleanup()
		case <-stopChan:
			return
		}
	}
}

// cleanup 执行日志清理
func (lc *LogCleaner) cleanup() {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	// 获取日志文件列表
	files, err := lc.getLogFiles()
	if err != nil {
		fmt.Printf("Failed to get log files: %v\n", err)
		return
	}

	// 按修改时间排序（最新的在前）
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime().After(files[j].ModTime())
	})

	now := time.Now()
	deletedCount := 0

	for i, file := range files {
		shouldDelete := false

		// 检查文件年龄
		if lc.maxAge > 0 && now.Sub(file.ModTime()) > lc.maxAge {
			shouldDelete = true
		}

		// 检查文件数量限制
		if lc.maxFiles > 0 && i >= lc.maxFiles {
			shouldDelete = true
		}

		if shouldDelete {
			filePath := filepath.Join(lc.logDir, file.Name())
			if err := os.Remove(filePath); err != nil {
				fmt.Printf("Failed to delete log file %s: %v\n", filePath, err)
			} else {
				deletedCount++
			}
		}
	}

	if deletedCount > 0 {
		fmt.Printf("Cleaned up %d old log files\n", deletedCount)
	}
}

// getLogFiles 获取日志文件列表
func (lc *LogCleaner) getLogFiles() ([]fs.FileInfo, error) {
	var files []fs.FileInfo

	err := filepath.Walk(lc.logDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// 检查文件名模式
		if lc.pattern != "" {
			matched, err := filepath.Match(lc.pattern, info.Name())
			if err != nil || !matched {
				return nil
			}
		}

		// 排除当前活跃的日志文件
		if !strings.Contains(info.Name(), ".") || strings.HasSuffix(info.Name(), ".log") {
			return nil
		}

		files = append(files, info)
		return nil
	})

	return files, err
}

// LogAnalyzer 日志分析器
type LogAnalyzer struct {
	logDir   string
	patterns map[string]string
	mu       sync.RWMutex
}

// NewLogAnalyzer 创建日志分析器
func NewLogAnalyzer(logDir string) *LogAnalyzer {
	return &LogAnalyzer{
		logDir: logDir,
		patterns: map[string]string{
			"error":   `"level":"error"`,
			"warning": `"level":"warn"`,
			"panic":   `"level":"panic"`,
			"fatal":   `"level":"fatal"`,
		},
	}
}

// AnalyzeLogFile 分析日志文件
func (la *LogAnalyzer) AnalyzeLogFile(filename string) (*LogAnalysisResult, error) {
	la.mu.RLock()
	defer la.mu.RUnlock()

	filePath := filepath.Join(la.logDir, filename)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	result := &LogAnalysisResult{
		Filename:  filename,
		FileSize:  len(content),
		LineCount: strings.Count(string(content), "\n"),
		Patterns:  make(map[string]int),
	}

	// 分析模式匹配
	contentStr := string(content)
	for pattern, regex := range la.patterns {
		count := strings.Count(contentStr, regex)
		result.Patterns[pattern] = count
	}

	return result, nil
}

// LogAnalysisResult 日志分析结果
type LogAnalysisResult struct {
	Filename  string         `json:"filename"`
	FileSize  int            `json:"file_size"`
	LineCount int            `json:"line_count"`
	Patterns  map[string]int `json:"patterns"`
}

// LogMetrics 日志指标
type LogMetrics struct {
	TotalLogs     int64            `json:"total_logs"`
	LogsByLevel   map[string]int64 `json:"logs_by_level"`
	LogsByModule  map[string]int64 `json:"logs_by_module"`
	ErrorRate     float64          `json:"error_rate"`
	LogRate       float64          `json:"log_rate"`
	DiskUsage     int64            `json:"disk_usage"`
	LastRotation  time.Time        `json:"last_rotation"`
	LastCleanup   time.Time        `json:"last_cleanup"`
	mu            sync.RWMutex
}

// NewLogMetrics 创建日志指标
func NewLogMetrics() *LogMetrics {
	return &LogMetrics{
		LogsByLevel:  make(map[string]int64),
		LogsByModule: make(map[string]int64),
	}
}

// IncrementLog 增加日志计数
func (lm *LogMetrics) IncrementLog(level, module string) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lm.TotalLogs++
	lm.LogsByLevel[level]++
	lm.LogsByModule[module]++
}

// UpdateErrorRate 更新错误率
func (lm *LogMetrics) UpdateErrorRate() {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if lm.TotalLogs == 0 {
		lm.ErrorRate = 0
		return
	}

	errorLogs := lm.LogsByLevel["error"] + lm.LogsByLevel["fatal"] + lm.LogsByLevel["panic"]
	lm.ErrorRate = float64(errorLogs) / float64(lm.TotalLogs)
}

// GetMetrics 获取指标快照
func (lm *LogMetrics) GetMetrics() LogMetrics {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	// 创建副本
	metrics := LogMetrics{
		TotalLogs:    lm.TotalLogs,
		LogsByLevel:  make(map[string]int64),
		LogsByModule: make(map[string]int64),
		ErrorRate:    lm.ErrorRate,
		LogRate:      lm.LogRate,
		DiskUsage:    lm.DiskUsage,
		LastRotation: lm.LastRotation,
		LastCleanup:  lm.LastCleanup,
	}

	for k, v := range lm.LogsByLevel {
		metrics.LogsByLevel[k] = v
	}

	for k, v := range lm.LogsByModule {
		metrics.LogsByModule[k] = v
	}

	return metrics
}

// 全局日志管理器
var globalLogManager *LogManager

// InitLogManager 初始化全局日志管理器
func InitLogManager(config Config) error {
	globalLogManager = NewLogManager(config)
	return globalLogManager.Start()
}

// GetLogManager 获取全局日志管理器
func GetLogManager() *LogManager {
	return globalLogManager
}

// StopLogManager 停止全局日志管理器
func StopLogManager() error {
	if globalLogManager != nil {
		return globalLogManager.Stop()
	}
	return nil
}