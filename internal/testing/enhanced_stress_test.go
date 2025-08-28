package testing

import (
	"context"
	"fmt"
	"log"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"qcat/internal/concurrent"
	"qcat/internal/workflow"
)

// EnhancedStressTestFramework 增强版压力测试框架
type EnhancedStressTestFramework struct {
	// 基础组件
	config             *EnhancedStressTestConfig
	realDataFetcher    *RealMarketDataFetcher
	timeAccelerator    *EnhancedTimeAccelerator
	workflowEngine     *workflow.WorkflowEngine
	poolManager        *concurrent.GoroutinePool
	performanceMonitor *PerformanceMonitor

	// 测试场景
	scenarios      []*TestScenario
	scenariosMutex sync.RWMutex

	// 统计计数器（原子操作）
	totalRequests      int64
	successfulRequests int64
	failedRequests     int64
	totalLatency       int64 // 总延迟（纳秒）
	maxLatency         int64 // 最大延迟（纳秒）
	minLatency         int64 // 最小延迟（纳秒）

	// 资源使用统计
	peakMemoryUsage uint64
	peakGoroutines  int64
	peakCPUUsage    float64

	// 错误统计
	errorCounts      map[string]*int64
	errorCountsMutex sync.RWMutex

	// 控制
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	running      bool
	runningMutex sync.RWMutex

	// 结果
	result      *EnhancedStressTestResult
	resultMutex sync.RWMutex
}

// EnhancedStressTestConfig 增强版压力测试配置
type EnhancedStressTestConfig struct {
	// 基础配置
	TestName         string        `json:"test_name"`
	Duration         time.Duration `json:"duration"`
	WarmupDuration   time.Duration `json:"warmup_duration"`
	CooldownDuration time.Duration `json:"cooldown_duration"`

	// 负载配置
	InitialConcurrency  int           `json:"initial_concurrency"`
	MaxConcurrency      int           `json:"max_concurrency"`
	ConcurrencyStep     int           `json:"concurrency_step"`
	ConcurrencyInterval time.Duration `json:"concurrency_interval"`

	// 请求配置
	InitialRPS  int           `json:"initial_rps"`
	MaxRPS      int           `json:"max_rps"`
	RPSStep     int           `json:"rps_step"`
	RPSInterval time.Duration `json:"rps_interval"`

	// 时间加速配置
	TimeAcceleration *TimeAcceleratorConfig `json:"time_acceleration"`

	// 数据配置
	UseRealData       bool                   `json:"use_real_data"`
	DataFetcherConfig *RealDataFetcherConfig `json:"data_fetcher_config"`

	// 极端测试配置
	EnableExtremeTests bool          `json:"enable_extreme_tests"`
	MemoryPressure     bool          `json:"memory_pressure"`
	CPUPressure        bool          `json:"cpu_pressure"`
	NetworkLatency     time.Duration `json:"network_latency"`
	ErrorInjection     bool          `json:"error_injection"`
	ErrorRate          float64       `json:"error_rate"`

	// 监控配置
	MonitoringInterval time.Duration `json:"monitoring_interval"`
	EnableProfiling    bool          `json:"enable_profiling"`

	// 停止条件
	MaxErrors      int64         `json:"max_errors"`
	MaxErrorRate   float64       `json:"max_error_rate"`
	MaxMemoryUsage uint64        `json:"max_memory_usage"`
	MaxLatency     time.Duration `json:"max_latency"`
}

// TestScenario 测试场景
type TestScenario struct {
	ID          string                          `json:"id"`
	Name        string                          `json:"name"`
	Description string                          `json:"description"`
	Weight      float64                         `json:"weight"`
	Handler     func(ctx context.Context) error `json:"-"`
	Enabled     bool                            `json:"enabled"`

	// 场景统计
	ExecutionCount int64 `json:"execution_count"`
	SuccessCount   int64 `json:"success_count"`
	ErrorCount     int64 `json:"error_count"`
	TotalLatency   int64 `json:"total_latency"`
	MaxLatency     int64 `json:"max_latency"`
	MinLatency     int64 `json:"min_latency"`
}

// EnhancedStressTestResult 增强版压力测试结果
type EnhancedStressTestResult struct {
	// 基础信息
	TestName  string        `json:"test_name"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`

	// 负载信息
	PeakConcurrency int `json:"peak_concurrency"`
	PeakRPS         int `json:"peak_rps"`

	// 请求统计
	TotalRequests      int64   `json:"total_requests"`
	SuccessfulRequests int64   `json:"successful_requests"`
	FailedRequests     int64   `json:"failed_requests"`
	SuccessRate        float64 `json:"success_rate"`
	ErrorRate          float64 `json:"error_rate"`

	// 性能统计
	AverageLatency time.Duration `json:"average_latency"`
	MaxLatency     time.Duration `json:"max_latency"`
	MinLatency     time.Duration `json:"min_latency"`
	P50Latency     time.Duration `json:"p50_latency"`
	P95Latency     time.Duration `json:"p95_latency"`
	P99Latency     time.Duration `json:"p99_latency"`

	// 吞吐量统计
	AverageRPS      float64 `json:"average_rps"`
	PeakRPSAchieved float64 `json:"peak_rps_achieved"`

	// 资源使用统计
	PeakMemoryUsage uint64  `json:"peak_memory_usage"`
	PeakGoroutines  int64   `json:"peak_goroutines"`
	PeakCPUUsage    float64 `json:"peak_cpu_usage"`

	// 错误统计
	ErrorDistribution map[string]int64 `json:"error_distribution"`

	// 场景统计
	ScenarioResults []*ScenarioResult `json:"scenario_results"`

	// 时间加速统计
	TimeAcceleration *TimeAcceleratorStats `json:"time_acceleration"`

	// 判定结果
	Passed        bool   `json:"passed"`
	FailureReason string `json:"failure_reason"`
}

// ScenarioResult 场景结果
type ScenarioResult struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	ExecutionCount int64         `json:"execution_count"`
	SuccessCount   int64         `json:"success_count"`
	ErrorCount     int64         `json:"error_count"`
	SuccessRate    float64       `json:"success_rate"`
	AverageLatency time.Duration `json:"average_latency"`
	MaxLatency     time.Duration `json:"max_latency"`
	MinLatency     time.Duration `json:"min_latency"`
}

// DefaultEnhancedStressTestConfig 默认配置
func DefaultEnhancedStressTestConfig() *EnhancedStressTestConfig {
	return &EnhancedStressTestConfig{
		TestName:         "Enhanced Stress Test",
		Duration:         5 * time.Minute,
		WarmupDuration:   30 * time.Second,
		CooldownDuration: 30 * time.Second,

		InitialConcurrency:  10,
		MaxConcurrency:      1000,
		ConcurrencyStep:     10,
		ConcurrencyInterval: 10 * time.Second,

		InitialRPS:  100,
		MaxRPS:      10000,
		RPSStep:     100,
		RPSInterval: 10 * time.Second,

		TimeAcceleration: DefaultTimeAcceleratorConfig(),

		UseRealData:       true,
		DataFetcherConfig: DefaultRealDataFetcherConfig(),

		EnableExtremeTests: true,
		MemoryPressure:     false,
		CPUPressure:        false,
		NetworkLatency:     0,
		ErrorInjection:     false,
		ErrorRate:          0.01, // 1% 错误率

		MonitoringInterval: time.Second,
		EnableProfiling:    false,

		MaxErrors:      1000,
		MaxErrorRate:   0.1,                    // 10% 最大错误率
		MaxMemoryUsage: 2 * 1024 * 1024 * 1024, // 2GB
		MaxLatency:     10 * time.Second,
	}
}

// NewEnhancedStressTestFramework 创建增强版压力测试框架
func NewEnhancedStressTestFramework(config *EnhancedStressTestConfig) (*EnhancedStressTestFramework, error) {
	if config == nil {
		config = DefaultEnhancedStressTestConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	framework := &EnhancedStressTestFramework{
		config:      config,
		scenarios:   make([]*TestScenario, 0),
		errorCounts: make(map[string]*int64),
		ctx:         ctx,
		cancel:      cancel,
		running:     false,
		minLatency:  math.MaxInt64,
	}

	// 创建真实数据获取器
	if config.UseRealData {
		realDataFetcher, err := NewRealMarketDataFetcher(config.DataFetcherConfig)
		if err != nil {
			log.Printf("Warning: Failed to create real data fetcher: %v", err)
		} else {
			framework.realDataFetcher = realDataFetcher
		}
	}

	// 创建时间加速器
	if config.TimeAcceleration != nil {
		timeAccelerator := NewEnhancedTimeAccelerator(time.Now(), config.TimeAcceleration)
		framework.timeAccelerator = timeAccelerator
	}

	// 创建工作流引擎
	workflowEngine := workflow.NewWorkflowEngine(config.MaxConcurrency)
	framework.workflowEngine = workflowEngine

	// 创建Goroutine池
	poolManager := concurrent.NewGoroutinePool(&concurrent.PoolConfig{
		MaxWorkers: config.MaxConcurrency * 2,
		QueueSize:  config.MaxConcurrency * 10,
	})
	framework.poolManager = poolManager

	// 创建性能监控器
	performanceMonitor := NewPerformanceMonitor(config.MonitoringInterval)
	framework.performanceMonitor = performanceMonitor

	return framework, nil
}

// AddScenario 添加测试场景
func (estf *EnhancedStressTestFramework) AddScenario(scenario *TestScenario) error {
	if scenario == nil {
		return fmt.Errorf("场景不能为空")
	}

	estf.scenariosMutex.Lock()
	defer estf.scenariosMutex.Unlock()

	// 检查ID是否重复
	for _, s := range estf.scenarios {
		if s.ID == scenario.ID {
			return fmt.Errorf("场景ID已存在: %s", scenario.ID)
		}
	}

	// 初始化统计
	scenario.MinLatency = math.MaxInt64

	estf.scenarios = append(estf.scenarios, scenario)
	log.Printf("添加测试场景: %s - %s", scenario.ID, scenario.Name)

	return nil
}

// RemoveScenario 移除测试场景
func (estf *EnhancedStressTestFramework) RemoveScenario(scenarioID string) bool {
	estf.scenariosMutex.Lock()
	defer estf.scenariosMutex.Unlock()

	for i, scenario := range estf.scenarios {
		if scenario.ID == scenarioID {
			estf.scenarios = append(estf.scenarios[:i], estf.scenarios[i+1:]...)
			log.Printf("移除测试场景: %s", scenarioID)
			return true
		}
	}

	return false
}

// Start 启动压力测试
func (estf *EnhancedStressTestFramework) Start() error {
	estf.runningMutex.Lock()
	defer estf.runningMutex.Unlock()

	if estf.running {
		return fmt.Errorf("压力测试已经在运行")
	}

	estf.running = true

	log.Printf("🚀 启动增强版压力测试: %s", estf.config.TestName)

	// 初始化结果
	estf.resultMutex.Lock()
	estf.result = &EnhancedStressTestResult{
		TestName:          estf.config.TestName,
		StartTime:         time.Now(),
		ErrorDistribution: make(map[string]int64),
		ScenarioResults:   make([]*ScenarioResult, 0),
	}
	estf.resultMutex.Unlock()

	// 启动组件
	if err := estf.startComponents(); err != nil {
		estf.running = false
		return fmt.Errorf("启动组件失败: %v", err)
	}

	// 启动监控
	estf.wg.Add(1)
	go estf.monitoringLoop()

	// 启动测试执行
	estf.wg.Add(1)
	go estf.testExecutionLoop()

	return nil
}

// Stop 停止压力测试
func (estf *EnhancedStressTestFramework) Stop() {
	estf.runningMutex.Lock()
	defer estf.runningMutex.Unlock()

	if !estf.running {
		return
	}

	log.Printf("🛑 停止增强版压力测试...")

	estf.running = false
	estf.cancel()
	estf.wg.Wait()

	// 停止组件
	estf.stopComponents()

	// 生成最终结果
	estf.generateFinalResult()

	log.Printf("✅ 压力测试已停止")
}

// startComponents 启动组件
func (estf *EnhancedStressTestFramework) startComponents() error {
	// 启动真实数据获取器
	if estf.realDataFetcher != nil {
		if err := estf.realDataFetcher.Start(); err != nil {
			log.Printf("Warning: Failed to start real data fetcher: %v", err)
		}
	}

	// 启动时间加速器
	if estf.timeAccelerator != nil {
		if err := estf.timeAccelerator.Start(); err != nil {
			log.Printf("Warning: Failed to start time accelerator: %v", err)
		}
	}

	// 启动性能监控器
	if estf.performanceMonitor != nil {
		estf.performanceMonitor.Start()
	}

	// 启动Goroutine池
	if estf.poolManager != nil {
		estf.poolManager.Start()
	}

	return nil
}

// stopComponents 停止组件
func (estf *EnhancedStressTestFramework) stopComponents() {
	// 停止真实数据获取器
	if estf.realDataFetcher != nil {
		estf.realDataFetcher.Stop()
	}

	// 停止时间加速器
	if estf.timeAccelerator != nil {
		estf.timeAccelerator.Stop()
	}

	// 停止性能监控器
	if estf.performanceMonitor != nil {
		estf.performanceMonitor.Stop()
	}

	// 停止Goroutine池
	if estf.poolManager != nil {
		estf.poolManager.Stop()
	}
}

// testExecutionLoop 测试执行循环
func (estf *EnhancedStressTestFramework) testExecutionLoop() {
	defer estf.wg.Done()

	startTime := time.Now()

	// 预热阶段
	if estf.config.WarmupDuration > 0 {
		log.Printf("📈 开始预热阶段，持续时间: %v", estf.config.WarmupDuration)
		estf.runPhase("warmup", estf.config.WarmupDuration, estf.config.InitialConcurrency, estf.config.InitialRPS)
	}

	// 主测试阶段
	log.Printf("🔥 开始主测试阶段，持续时间: %v", estf.config.Duration)
	estf.runPhase("main", estf.config.Duration, estf.config.MaxConcurrency, estf.config.MaxRPS)

	// 冷却阶段
	if estf.config.CooldownDuration > 0 {
		log.Printf("📉 开始冷却阶段，持续时间: %v", estf.config.CooldownDuration)
		estf.runPhase("cooldown", estf.config.CooldownDuration, estf.config.InitialConcurrency, estf.config.InitialRPS)
	}

	log.Printf("✅ 测试执行完成，总耗时: %v", time.Since(startTime))
	estf.Stop()
}

// runPhase 运行测试阶段
func (estf *EnhancedStressTestFramework) runPhase(phaseName string, duration time.Duration, maxConcurrency, maxRPS int) {
	phaseStart := time.Now()
	phaseEnd := phaseStart.Add(duration)

	currentConcurrency := estf.config.InitialConcurrency
	currentRPS := estf.config.InitialRPS

	// 创建工作协程
	for i := 0; i < maxConcurrency; i++ {
		estf.wg.Add(1)
		go estf.workerLoop(phaseName)
	}

	// 动态调整负载
	ticker := time.NewTicker(estf.config.ConcurrencyInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 调整并发数
			if currentConcurrency < maxConcurrency {
				currentConcurrency = minInt(currentConcurrency+estf.config.ConcurrencyStep, maxConcurrency)
			}

			// 调整RPS
			if currentRPS < maxRPS {
				currentRPS = minInt(currentRPS+estf.config.RPSStep, maxRPS)
			}

			log.Printf("[%s] 当前负载: 并发=%d, RPS=%d", phaseName, currentConcurrency, currentRPS)

		case <-estf.ctx.Done():
			return
		default:
			if time.Now().After(phaseEnd) {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// workerLoop 工作协程循环
func (estf *EnhancedStressTestFramework) workerLoop(phaseName string) {
	defer estf.wg.Done()

	for {
		select {
		case <-estf.ctx.Done():
			return
		default:
			// 执行测试场景
			estf.executeScenario()

			// 控制请求频率
			time.Sleep(time.Millisecond * 10) // 基础延迟
		}
	}
}

// executeScenario 执行测试场景
func (estf *EnhancedStressTestFramework) executeScenario() {
	estf.scenariosMutex.RLock()
	scenarios := estf.scenarios
	estf.scenariosMutex.RUnlock()

	if len(scenarios) == 0 {
		return
	}

	// 根据权重选择场景
	scenario := estf.selectScenarioByWeight(scenarios)
	if scenario == nil || !scenario.Enabled {
		return
	}

	// 记录开始时间
	startTime := time.Now()

	// 执行场景
	var err error
	if scenario.Handler != nil {
		err = scenario.Handler(estf.ctx)
	}

	// 记录结束时间和统计
	latency := time.Since(startTime)
	estf.recordScenarioExecution(scenario, latency, err)
}

// selectScenarioByWeight 根据权重选择场景
func (estf *EnhancedStressTestFramework) selectScenarioByWeight(scenarios []*TestScenario) *TestScenario {
	if len(scenarios) == 0 {
		return nil
	}

	// 简单实现：随机选择（可以后续改进为真正的权重选择）
	return scenarios[time.Now().UnixNano()%int64(len(scenarios))]
}

// recordScenarioExecution 记录场景执行结果
func (estf *EnhancedStressTestFramework) recordScenarioExecution(scenario *TestScenario, latency time.Duration, err error) {
	latencyNanos := latency.Nanoseconds()

	// 更新场景统计
	atomic.AddInt64(&scenario.ExecutionCount, 1)
	atomic.AddInt64(&scenario.TotalLatency, latencyNanos)

	// 更新最大延迟
	for {
		current := atomic.LoadInt64(&scenario.MaxLatency)
		if latencyNanos <= current {
			break
		}
		if atomic.CompareAndSwapInt64(&scenario.MaxLatency, current, latencyNanos) {
			break
		}
	}

	// 更新最小延迟
	for {
		current := atomic.LoadInt64(&scenario.MinLatency)
		if latencyNanos >= current {
			break
		}
		if atomic.CompareAndSwapInt64(&scenario.MinLatency, current, latencyNanos) {
			break
		}
	}

	// 更新全局统计
	atomic.AddInt64(&estf.totalRequests, 1)
	atomic.AddInt64(&estf.totalLatency, latencyNanos)

	// 更新全局最大延迟
	for {
		current := atomic.LoadInt64(&estf.maxLatency)
		if latencyNanos <= current {
			break
		}
		if atomic.CompareAndSwapInt64(&estf.maxLatency, current, latencyNanos) {
			break
		}
	}

	// 更新全局最小延迟
	for {
		current := atomic.LoadInt64(&estf.minLatency)
		if latencyNanos >= current {
			break
		}
		if atomic.CompareAndSwapInt64(&estf.minLatency, current, latencyNanos) {
			break
		}
	}

	if err != nil {
		// 记录错误
		atomic.AddInt64(&estf.failedRequests, 1)
		atomic.AddInt64(&scenario.ErrorCount, 1)
		estf.recordError(err.Error())
	} else {
		// 记录成功
		atomic.AddInt64(&estf.successfulRequests, 1)
		atomic.AddInt64(&scenario.SuccessCount, 1)
	}
}

// recordError 记录错误
func (estf *EnhancedStressTestFramework) recordError(errorMsg string) {
	estf.errorCountsMutex.Lock()
	defer estf.errorCountsMutex.Unlock()

	if counter, exists := estf.errorCounts[errorMsg]; exists {
		atomic.AddInt64(counter, 1)
	} else {
		counter := int64(1)
		estf.errorCounts[errorMsg] = &counter
	}
}

// monitoringLoop 监控循环
func (estf *EnhancedStressTestFramework) monitoringLoop() {
	defer estf.wg.Done()

	ticker := time.NewTicker(estf.config.MonitoringInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			estf.updateMonitoringStats()
			estf.checkStopConditions()
		case <-estf.ctx.Done():
			return
		}
	}
}

// updateMonitoringStats 更新监控统计
func (estf *EnhancedStressTestFramework) updateMonitoringStats() {
	// 获取系统资源使用情况
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// 更新峰值内存使用
	if memStats.Alloc > estf.peakMemoryUsage {
		estf.peakMemoryUsage = memStats.Alloc
	}

	// 更新峰值Goroutine数量
	goroutines := int64(runtime.NumGoroutine())
	if goroutines > estf.peakGoroutines {
		estf.peakGoroutines = goroutines
	}

	// 获取性能监控数据
	if estf.performanceMonitor != nil {
		// TODO 获取CPU使用率等指标，暂时跳过
		// cpuMetric := estf.performanceMonitor.GetMetric("cpu_usage")
		// if cpuMetric != nil && cpuMetric.Current > estf.peakCPUUsage {
		//     estf.peakCPUUsage = cpuMetric.Current
		// }
	}

	// 打印当前状态
	totalReqs := atomic.LoadInt64(&estf.totalRequests)
	successReqs := atomic.LoadInt64(&estf.successfulRequests)
	failedReqs := atomic.LoadInt64(&estf.failedRequests)

	if totalReqs > 0 {
		successRate := float64(successReqs) / float64(totalReqs) * 100
		log.Printf("📊 当前状态: 总请求=%d, 成功=%d, 失败=%d, 成功率=%.2f%%, 内存=%dMB, Goroutines=%d",
			totalReqs, successReqs, failedReqs, successRate, estf.peakMemoryUsage/1024/1024, estf.peakGoroutines)
	}
}

// checkStopConditions 检查停止条件
func (estf *EnhancedStressTestFramework) checkStopConditions() {
	totalReqs := atomic.LoadInt64(&estf.totalRequests)
	failedReqs := atomic.LoadInt64(&estf.failedRequests)

	// 检查最大错误数
	if estf.config.MaxErrors > 0 && failedReqs >= estf.config.MaxErrors {
		log.Printf("⚠️ 达到最大错误数限制 (%d)，停止测试", estf.config.MaxErrors)
		estf.Stop()
		return
	}

	// 检查错误率
	if totalReqs > 0 && estf.config.MaxErrorRate > 0 {
		errorRate := float64(failedReqs) / float64(totalReqs)
		if errorRate >= estf.config.MaxErrorRate {
			log.Printf("⚠️ 达到最大错误率限制 (%.2f%%)，停止测试", estf.config.MaxErrorRate*100)
			estf.Stop()
			return
		}
	}

	// 检查内存使用
	if estf.config.MaxMemoryUsage > 0 && estf.peakMemoryUsage >= estf.config.MaxMemoryUsage {
		log.Printf("⚠️ 达到最大内存使用限制 (%dMB)，停止测试", estf.config.MaxMemoryUsage/1024/1024)
		estf.Stop()
		return
	}

	// 检查最大延迟
	if estf.config.MaxLatency > 0 {
		maxLatencyNanos := atomic.LoadInt64(&estf.maxLatency)
		if time.Duration(maxLatencyNanos) >= estf.config.MaxLatency {
			log.Printf("⚠️ 达到最大延迟限制 (%v)，停止测试", estf.config.MaxLatency)
			estf.Stop()
			return
		}
	}
}

// generateFinalResult 生成最终结果
func (estf *EnhancedStressTestFramework) generateFinalResult() {
	estf.resultMutex.Lock()
	defer estf.resultMutex.Unlock()

	if estf.result == nil {
		return
	}

	// 基础信息
	estf.result.EndTime = time.Now()
	estf.result.Duration = estf.result.EndTime.Sub(estf.result.StartTime)

	// 请求统计
	estf.result.TotalRequests = atomic.LoadInt64(&estf.totalRequests)
	estf.result.SuccessfulRequests = atomic.LoadInt64(&estf.successfulRequests)
	estf.result.FailedRequests = atomic.LoadInt64(&estf.failedRequests)

	if estf.result.TotalRequests > 0 {
		estf.result.SuccessRate = float64(estf.result.SuccessfulRequests) / float64(estf.result.TotalRequests)
		estf.result.ErrorRate = float64(estf.result.FailedRequests) / float64(estf.result.TotalRequests)
		estf.result.AverageRPS = float64(estf.result.TotalRequests) / estf.result.Duration.Seconds()
	}

	// 性能统计
	totalLatencyNanos := atomic.LoadInt64(&estf.totalLatency)
	if estf.result.TotalRequests > 0 {
		estf.result.AverageLatency = time.Duration(totalLatencyNanos / estf.result.TotalRequests)
	}
	estf.result.MaxLatency = time.Duration(atomic.LoadInt64(&estf.maxLatency))
	estf.result.MinLatency = time.Duration(atomic.LoadInt64(&estf.minLatency))

	// 资源使用统计
	estf.result.PeakMemoryUsage = estf.peakMemoryUsage
	estf.result.PeakGoroutines = estf.peakGoroutines
	estf.result.PeakCPUUsage = estf.peakCPUUsage

	// 错误分布
	estf.errorCountsMutex.RLock()
	for errorMsg, counter := range estf.errorCounts {
		estf.result.ErrorDistribution[errorMsg] = atomic.LoadInt64(counter)
	}
	estf.errorCountsMutex.RUnlock()

	// 场景结果
	estf.scenariosMutex.RLock()
	for _, scenario := range estf.scenarios {
		scenarioResult := &ScenarioResult{
			ID:             scenario.ID,
			Name:           scenario.Name,
			ExecutionCount: atomic.LoadInt64(&scenario.ExecutionCount),
			SuccessCount:   atomic.LoadInt64(&scenario.SuccessCount),
			ErrorCount:     atomic.LoadInt64(&scenario.ErrorCount),
			MaxLatency:     time.Duration(atomic.LoadInt64(&scenario.MaxLatency)),
			MinLatency:     time.Duration(atomic.LoadInt64(&scenario.MinLatency)),
		}

		if scenarioResult.ExecutionCount > 0 {
			scenarioResult.SuccessRate = float64(scenarioResult.SuccessCount) / float64(scenarioResult.ExecutionCount)
			totalLatency := atomic.LoadInt64(&scenario.TotalLatency)
			scenarioResult.AverageLatency = time.Duration(totalLatency / scenarioResult.ExecutionCount)
		}

		estf.result.ScenarioResults = append(estf.result.ScenarioResults, scenarioResult)
	}
	estf.scenariosMutex.RUnlock()

	// 时间加速统计
	if estf.timeAccelerator != nil {
		estf.result.TimeAcceleration = estf.timeAccelerator.GetStats()
	}

	// 判定结果
	estf.result.Passed = estf.result.ErrorRate <= estf.config.MaxErrorRate
	if !estf.result.Passed {
		estf.result.FailureReason = fmt.Sprintf("错误率过高: %.2f%%", estf.result.ErrorRate*100)
	}

	// 打印最终结果
	log.Printf("📊 压力测试结果:")
	log.Printf("   测试名称: %s", estf.result.TestName)
	log.Printf("   测试时长: %v", estf.result.Duration)
	log.Printf("   总请求数: %d", estf.result.TotalRequests)
	log.Printf("   成功请求: %d", estf.result.SuccessfulRequests)
	log.Printf("   失败请求: %d", estf.result.FailedRequests)
	log.Printf("   成功率: %.2f%%", estf.result.SuccessRate*100)
	log.Printf("   平均延迟: %v", estf.result.AverageLatency)
	log.Printf("   最大延迟: %v", estf.result.MaxLatency)
	log.Printf("   平均RPS: %.2f", estf.result.AverageRPS)
	log.Printf("   峰值内存: %dMB", estf.result.PeakMemoryUsage/1024/1024)
	log.Printf("   峰值Goroutines: %d", estf.result.PeakGoroutines)
	log.Printf("   测试结果: %s", map[bool]string{true: "✅ 通过", false: "❌ 失败"}[estf.result.Passed])
	if !estf.result.Passed {
		log.Printf("   失败原因: %s", estf.result.FailureReason)
	}
}

// GetResult 获取测试结果
func (estf *EnhancedStressTestFramework) GetResult() *EnhancedStressTestResult {
	estf.resultMutex.RLock()
	defer estf.resultMutex.RUnlock()

	if estf.result == nil {
		return nil
	}

	// 返回副本
	result := *estf.result
	result.ErrorDistribution = make(map[string]int64)
	for k, v := range estf.result.ErrorDistribution {
		result.ErrorDistribution[k] = v
	}

	result.ScenarioResults = make([]*ScenarioResult, len(estf.result.ScenarioResults))
	copy(result.ScenarioResults, estf.result.ScenarioResults)

	return &result
}

// IsRunning 检查是否正在运行
func (estf *EnhancedStressTestFramework) IsRunning() bool {
	estf.runningMutex.RLock()
	defer estf.runningMutex.RUnlock()

	return estf.running
}

// GetConfig 获取配置
func (estf *EnhancedStressTestFramework) GetConfig() *EnhancedStressTestConfig {
	// 返回副本
	config := *estf.config
	return &config
}

// minInt 辅助函数
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
