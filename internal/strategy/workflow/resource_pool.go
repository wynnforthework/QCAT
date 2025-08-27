package workflow

import (
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/concurrent"
)

// StrategyResourceConfig 策略资源配置
type StrategyResourceConfig struct {
	CPUQuota        float64 `yaml:"cpu_quota"`
	MemoryQuota     int64   `yaml:"memory_quota"`
	BacktestWorkers int     `yaml:"backtest_workers"`
	OptimizeWorkers int     `yaml:"optimize_workers"`
	LearningWorkers int     `yaml:"learning_workers"`
}

// StrategyResourcePool 策略资源池
type StrategyResourcePool struct {
	config *StrategyResourceConfig

	// CPU资源
	cpuQuota float64
	cpuUsed  float64
	cpuMu    sync.RWMutex

	// 内存资源
	memoryQuota int64
	memoryUsed  int64
	memoryMu    sync.RWMutex

	// 工作线程池
	backtestPool *concurrent.GoroutinePool
	optimizePool *concurrent.GoroutinePool
	learningPool *concurrent.GoroutinePool

	// 数据连接池
	dataConnections map[string]*DataConnection
	connectionsMu   sync.RWMutex

	// 运行状态
	isRunning bool
	runningMu sync.RWMutex

	// 统计信息
	stats   *ResourceStats
	statsMu sync.RWMutex
}

// DataConnection 数据连接
type DataConnection struct {
	ID          string
	Type        string
	URL         string
	IsActive    bool
	LastUsed    time.Time
	UsageCount  int64
	ErrorCount  int64
	CreatedAt   time.Time
}

// ResourceStats 资源统计信息
type ResourceStats struct {
	CPUUsage           float64
	MemoryUsage        int64
	ActiveConnections  int
	BacktestTasksTotal int64
	OptimizeTasksTotal int64
	LearningTasksTotal int64
	TasksCompleted     int64
	TasksFailed        int64
	AverageTaskTime    time.Duration
	LastUpdateTime     time.Time
}

// NewStrategyResourcePool 创建策略资源池
func NewStrategyResourcePool(config *StrategyResourceConfig) *StrategyResourcePool {
	if config == nil {
		config = GetDefaultResourceConfig()
	}

	pool := &StrategyResourcePool{
		config:          config,
		cpuQuota:        config.CPUQuota,
		memoryQuota:     config.MemoryQuota,
		dataConnections: make(map[string]*DataConnection),
		stats: &ResourceStats{
			LastUpdateTime: time.Now(),
		},
	}

	return pool
}

// Start 启动资源池
func (srp *StrategyResourcePool) Start() error {
	srp.runningMu.Lock()
	defer srp.runningMu.Unlock()

	if srp.isRunning {
		return fmt.Errorf("resource pool is already running")
	}

	log.Println("启动策略资源池...")

	// 创建工作线程池
	if err := srp.createWorkerPools(); err != nil {
		return fmt.Errorf("failed to create worker pools: %w", err)
	}

	// 初始化数据连接
	if err := srp.initializeDataConnections(); err != nil {
		return fmt.Errorf("failed to initialize data connections: %w", err)
	}

	srp.isRunning = true
	log.Println("策略资源池启动完成")

	return nil
}

// Stop 停止资源池
func (srp *StrategyResourcePool) Stop() error {
	srp.runningMu.Lock()
	defer srp.runningMu.Unlock()

	if !srp.isRunning {
		return nil
	}

	log.Println("停止策略资源池...")

	// 停止工作线程池
	if srp.backtestPool != nil {
		srp.backtestPool.Stop()
	}
	if srp.optimizePool != nil {
		srp.optimizePool.Stop()
	}
	if srp.learningPool != nil {
		srp.learningPool.Stop()
	}

	// 关闭数据连接
	srp.closeDataConnections()

	srp.isRunning = false
	log.Println("策略资源池已停止")

	return nil
}

// createWorkerPools 创建工作线程池
func (srp *StrategyResourcePool) createWorkerPools() error {
	// 回测工作池
	backtestConfig := &concurrent.PoolConfig{
		MinWorkers:      1,
		MaxWorkers:      srp.config.BacktestWorkers,
		IdleTimeout:     5 * time.Minute,
		TaskTimeout:     30 * time.Minute,
		QueueSize:       100,
		EnableMetrics:   true,
		EnableProfiling: false,
	}
	srp.backtestPool = concurrent.NewPool("backtest", backtestConfig)

	// 优化工作池
	optimizeConfig := &concurrent.PoolConfig{
		MinWorkers:      1,
		MaxWorkers:      srp.config.OptimizeWorkers,
		IdleTimeout:     10 * time.Minute,
		TaskTimeout:     60 * time.Minute,
		QueueSize:       50,
		EnableMetrics:   true,
		EnableProfiling: false,
	}
	srp.optimizePool = concurrent.NewPool("optimize", optimizeConfig)

	// 学习工作池
	learningConfig := &concurrent.PoolConfig{
		MinWorkers:      1,
		MaxWorkers:      srp.config.LearningWorkers,
		IdleTimeout:     15 * time.Minute,
		TaskTimeout:     120 * time.Minute,
		QueueSize:       20,
		EnableMetrics:   true,
		EnableProfiling: false,
	}
	srp.learningPool = concurrent.NewPool("learning", learningConfig)

	return nil
}

// initializeDataConnections 初始化数据连接
func (srp *StrategyResourcePool) initializeDataConnections() error {
	// 创建默认数据连接
	connections := []*DataConnection{
		{
			ID:        "market_data_primary",
			Type:      "market_data",
			URL:       "ws://localhost:8080/market",
			IsActive:  true,
			CreatedAt: time.Now(),
		},
		{
			ID:        "historical_data",
			Type:      "historical_data",
			URL:       "http://localhost:8081/historical",
			IsActive:  true,
			CreatedAt: time.Now(),
		},
	}

	srp.connectionsMu.Lock()
	for _, conn := range connections {
		srp.dataConnections[conn.ID] = conn
	}
	srp.connectionsMu.Unlock()

	return nil
}

// closeDataConnections 关闭数据连接
func (srp *StrategyResourcePool) closeDataConnections() {
	srp.connectionsMu.Lock()
	defer srp.connectionsMu.Unlock()

	for _, conn := range srp.dataConnections {
		conn.IsActive = false
	}
}

// AcquireCPU 获取CPU资源
func (srp *StrategyResourcePool) AcquireCPU(amount float64) error {
	srp.cpuMu.Lock()
	defer srp.cpuMu.Unlock()

	if srp.cpuUsed+amount > srp.cpuQuota {
		return fmt.Errorf("insufficient CPU resources: requested %.2f, available %.2f", 
			amount, srp.cpuQuota-srp.cpuUsed)
	}

	srp.cpuUsed += amount
	srp.updateStats()
	return nil
}

// ReleaseCPU 释放CPU资源
func (srp *StrategyResourcePool) ReleaseCPU(amount float64) {
	srp.cpuMu.Lock()
	defer srp.cpuMu.Unlock()

	srp.cpuUsed -= amount
	if srp.cpuUsed < 0 {
		srp.cpuUsed = 0
	}
	srp.updateStats()
}

// AcquireMemory 获取内存资源
func (srp *StrategyResourcePool) AcquireMemory(amount int64) error {
	srp.memoryMu.Lock()
	defer srp.memoryMu.Unlock()

	if srp.memoryUsed+amount > srp.memoryQuota {
		return fmt.Errorf("insufficient memory resources: requested %d, available %d", 
			amount, srp.memoryQuota-srp.memoryUsed)
	}

	srp.memoryUsed += amount
	srp.updateStats()
	return nil
}

// ReleaseMemory 释放内存资源
func (srp *StrategyResourcePool) ReleaseMemory(amount int64) {
	srp.memoryMu.Lock()
	defer srp.memoryMu.Unlock()

	srp.memoryUsed -= amount
	if srp.memoryUsed < 0 {
		srp.memoryUsed = 0
	}
	srp.updateStats()
}

// GetBacktestPool 获取回测工作池
func (srp *StrategyResourcePool) GetBacktestPool() *concurrent.Pool {
	return srp.backtestPool
}

// GetOptimizePool 获取优化工作池
func (srp *StrategyResourcePool) GetOptimizePool() *concurrent.Pool {
	return srp.optimizePool
}

// GetLearningPool 获取学习工作池
func (srp *StrategyResourcePool) GetLearningPool() *concurrent.Pool {
	return srp.learningPool
}

// updateStats 更新统计信息
func (srp *StrategyResourcePool) updateStats() {
	srp.statsMu.Lock()
	defer srp.statsMu.Unlock()

	srp.stats.CPUUsage = srp.cpuUsed
	srp.stats.MemoryUsage = srp.memoryUsed
	srp.stats.ActiveConnections = len(srp.dataConnections)
	srp.stats.LastUpdateTime = time.Now()
}

// GetStats 获取统计信息
func (srp *StrategyResourcePool) GetStats() *ResourceStats {
	srp.statsMu.RLock()
	defer srp.statsMu.RUnlock()

	// 返回副本
	stats := *srp.stats
	return &stats
}

// GetDefaultResourceConfig 获取默认资源配置
func GetDefaultResourceConfig() *StrategyResourceConfig {
	return &StrategyResourceConfig{
		CPUQuota:        2.0,
		MemoryQuota:     4 * 1024 * 1024 * 1024, // 4GB
		BacktestWorkers: 3,
		OptimizeWorkers: 2,
		LearningWorkers: 1,
	}
}
