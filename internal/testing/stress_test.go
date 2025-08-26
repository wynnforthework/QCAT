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

// StressTestConfig 压力测试配置
type StressTestConfig struct {
	Duration           time.Duration `json:"duration"`             // 测试持续时间
	AccelerationFactor int           `json:"acceleration_factor"`  // 时间加速倍数
	ConcurrentUsers    int           `json:"concurrent_users"`     // 并发用户数
	RequestsPerSecond  int           `json:"requests_per_second"`  // 每秒请求数
	DataGenerationRate int           `json:"data_generation_rate"` // 数据生成频率(Hz)
	WorkflowExecutions int           `json:"workflow_executions"`  // 工作流执行次数
	EnableMonitoring   bool          `json:"enable_monitoring"`    // 启用监控
	EnableDataPersist  bool          `json:"enable_data_persist"`  // 启用数据持久化
	SimulateFailures   bool          `json:"simulate_failures"`    // 模拟失败
	FailureRate        float64       `json:"failure_rate"`         // 失败率 (0-1)
}

// StressTestResult 压力测试结果
type StressTestResult struct {
	StartTime          time.Time     `json:"start_time"`
	EndTime            time.Time     `json:"end_time"`
	Duration           time.Duration `json:"duration"`
	SimulatedDuration  time.Duration `json:"simulated_duration"`
	AccelerationFactor int           `json:"acceleration_factor"`

	// 请求统计
	TotalRequests       int64         `json:"total_requests"`
	SuccessfulRequests  int64         `json:"successful_requests"`
	FailedRequests      int64         `json:"failed_requests"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	MaxResponseTime     time.Duration `json:"max_response_time"`
	MinResponseTime     time.Duration `json:"min_response_time"`

	// 工作流统计
	WorkflowExecutions int64 `json:"workflow_executions"`
	WorkflowSuccesses  int64 `json:"workflow_successes"`
	WorkflowFailures   int64 `json:"workflow_failures"`

	// 数据统计
	DataPointsGenerated int64 `json:"data_points_generated"`
	SignalsGenerated    int64 `json:"signals_generated"`

	// 性能指标
	ThroughputRPS float64 `json:"throughput_rps"`
	ErrorRate     float64 `json:"error_rate"`
	SuccessRate   float64 `json:"success_rate"`

	// 资源使用
	PeakMemoryUsage uint64  `json:"peak_memory_usage"`
	PeakCPUUsage    float64 `json:"peak_cpu_usage"`
	PeakGoroutines  int     `json:"peak_goroutines"`

	// 详细统计
	ResponseTimeDistribution map[string]int64 `json:"response_time_distribution"`
	ErrorDistribution        map[string]int64 `json:"error_distribution"`
}

// StressTestFramework 压力测试框架
type StressTestFramework struct {
	config          *StressTestConfig
	dataGenerator   *DataGenerator
	timeAccelerator *TimeAccelerator
	workflowEngine  *workflow.WorkflowEngine
	poolManager     *concurrent.GoroutinePool

	// 统计计数器
	totalRequests       int64
	successfulRequests  int64
	failedRequests      int64
	workflowExecutions  int64
	workflowSuccesses   int64
	workflowFailures    int64
	dataPointsGenerated int64
	signalsGenerated    int64

	// 响应时间统计
	responseTimes      []time.Duration
	responseTimesMutex sync.RWMutex

	// 错误统计
	errors      map[string]int64
	errorsMutex sync.RWMutex

	// 控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewStressTestFramework 创建压力测试框架
func NewStressTestFramework(config *StressTestConfig) *StressTestFramework {
	ctx, cancel := context.WithCancel(context.Background())

	// 创建数据生成器
	dataGenerator := NewDataGenerator()

	// 创建时间加速器
	timeAccelerator := NewTimeAccelerator(
		time.Now(),
		config.AccelerationFactor,
		time.Second,
	)

	// 创建工作流引擎
	workflowEngine := workflow.NewWorkflowEngine(config.ConcurrentUsers)

	// 创建Goroutine池
	poolManager := concurrent.NewGoroutinePool(&concurrent.PoolConfig{
		MaxWorkers: config.ConcurrentUsers * 2,
		QueueSize:  config.ConcurrentUsers * 10,
	})

	return &StressTestFramework{
		config:          config,
		dataGenerator:   dataGenerator,
		timeAccelerator: timeAccelerator,
		workflowEngine:  workflowEngine,
		poolManager:     poolManager,
		ctx:             ctx,
		cancel:          cancel,
		errors:          make(map[string]int64),
	}
}

// Run 运行压力测试
func (stf *StressTestFramework) Run() (*StressTestResult, error) {
	log.Printf("🚀 开始压力测试，配置: %+v", stf.config)

	startTime := time.Now()

	// 启动组件
	stf.poolManager.Start()

	// 启动数据生成器
	stf.wg.Add(1)
	go stf.runDataGenerator()

	// 启动工作流执行器
	stf.wg.Add(1)
	go stf.runWorkflowExecutor()

	// 启动请求生成器
	stf.wg.Add(1)
	go stf.runRequestGenerator()

	// 启动监控器
	if stf.config.EnableMonitoring {
		stf.wg.Add(1)
		go stf.runMonitor()
	}

	// 等待测试完成
	select {
	case <-time.After(stf.config.Duration):
		log.Println("⏰ 压力测试时间到，开始停止...")
	case <-stf.ctx.Done():
		log.Println("🛑 压力测试被取消")
	}

	// 停止测试
	stf.cancel()
	stf.wg.Wait()
	stf.poolManager.Stop()

	endTime := time.Now()

	// 生成测试结果
	result := stf.generateResult(startTime, endTime)

	log.Printf("✅ 压力测试完成，结果: 成功率=%.2f%%, 吞吐量=%.2f RPS",
		result.SuccessRate*100, result.ThroughputRPS)

	return result, nil
}

// runDataGenerator 运行数据生成器
func (stf *StressTestFramework) runDataGenerator() {
	defer stf.wg.Done()

	ticker := time.NewTicker(time.Second / time.Duration(stf.config.DataGenerationRate))
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 推进时间
			currentTime := stf.timeAccelerator.Advance()

			// 更新市场状态
			stf.dataGenerator.UpdateMarketCondition(currentTime)

			// 为每个交易对生成数据
			symbols := stf.dataGenerator.GetSymbols()
			for _, symbol := range symbols {
				// 生成市场数据
				_ = stf.dataGenerator.GenerateMarketData(symbol, currentTime)
				atomic.AddInt64(&stf.dataPointsGenerated, 1)

				// 生成交易信号
				_ = stf.dataGenerator.GenerateTradingSignal(symbol, currentTime)
				atomic.AddInt64(&stf.signalsGenerated, 1)
			}

		case <-stf.ctx.Done():
			return
		}
	}
}

// runWorkflowExecutor 运行工作流执行器
func (stf *StressTestFramework) runWorkflowExecutor() {
	defer stf.wg.Done()

	executionInterval := stf.config.Duration / time.Duration(stf.config.WorkflowExecutions)
	if executionInterval < time.Second {
		executionInterval = time.Second
	}

	ticker := time.NewTicker(executionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			atomic.AddInt64(&stf.workflowExecutions, 1)

			// 异步执行工作流
			go func() {
				err := stf.workflowEngine.ExecuteWorkflow(stf.ctx)
				if err != nil {
					atomic.AddInt64(&stf.workflowFailures, 1)
					stf.recordError("workflow_execution", err)
				} else {
					atomic.AddInt64(&stf.workflowSuccesses, 1)
				}
			}()

		case <-stf.ctx.Done():
			return
		}
	}
}

// runRequestGenerator 运行请求生成器
func (stf *StressTestFramework) runRequestGenerator() {
	defer stf.wg.Done()

	requestInterval := time.Second / time.Duration(stf.config.RequestsPerSecond)
	ticker := time.NewTicker(requestInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 生成并发请求
			for i := 0; i < stf.config.ConcurrentUsers; i++ {
				go stf.simulateRequest()
			}

		case <-stf.ctx.Done():
			return
		}
	}
}

// simulateRequest 模拟请求
func (stf *StressTestFramework) simulateRequest() {
	startTime := time.Now()
	atomic.AddInt64(&stf.totalRequests, 1)

	// 模拟请求处理
	task := concurrent.NewAutomationTask(
		fmt.Sprintf("stress_test_%d", atomic.LoadInt64(&stf.totalRequests)),
		"压力测试任务",
		5,
		time.Second*5,
		func(ctx context.Context) error {
			// 模拟处理时间
			processingTime := time.Duration(50+stf.dataGenerator.rand.Intn(200)) * time.Millisecond

			select {
			case <-time.After(processingTime):
				// 模拟失败
				if stf.config.SimulateFailures && stf.dataGenerator.rand.Float64() < stf.config.FailureRate {
					return fmt.Errorf("simulated failure")
				}
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	)

	err := stf.poolManager.Submit(task)
	responseTime := time.Since(startTime)

	// 记录响应时间
	stf.recordResponseTime(responseTime)

	if err != nil {
		atomic.AddInt64(&stf.failedRequests, 1)
		stf.recordError("request_submission", err)
	} else {
		atomic.AddInt64(&stf.successfulRequests, 1)
	}
}

// runMonitor 运行监控器
func (stf *StressTestFramework) runMonitor() {
	defer stf.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 打印统计信息
			totalReq := atomic.LoadInt64(&stf.totalRequests)
			successReq := atomic.LoadInt64(&stf.successfulRequests)
			failedReq := atomic.LoadInt64(&stf.failedRequests)

			log.Printf("📊 压力测试进度 - 总请求: %d, 成功: %d, 失败: %d, 成功率: %.2f%%",
				totalReq, successReq, failedReq, float64(successReq)/float64(totalReq)*100)

		case <-stf.ctx.Done():
			return
		}
	}
}

// recordResponseTime 记录响应时间
func (stf *StressTestFramework) recordResponseTime(duration time.Duration) {
	stf.responseTimesMutex.Lock()
	defer stf.responseTimesMutex.Unlock()

	stf.responseTimes = append(stf.responseTimes, duration)

	// 限制记录数量
	if len(stf.responseTimes) > 10000 {
		stf.responseTimes = stf.responseTimes[1000:]
	}
}

// recordError 记录错误
func (stf *StressTestFramework) recordError(errorType string, err error) {
	stf.errorsMutex.Lock()
	defer stf.errorsMutex.Unlock()

	key := fmt.Sprintf("%s: %s", errorType, err.Error())
	stf.errors[key]++
}

// generateResult 生成测试结果
func (stf *StressTestFramework) generateResult(startTime, endTime time.Time) *StressTestResult {
	duration := endTime.Sub(startTime)
	simulatedDuration := stf.timeAccelerator.GetElapsedTime()

	totalReq := atomic.LoadInt64(&stf.totalRequests)
	successReq := atomic.LoadInt64(&stf.successfulRequests)
	failedReq := atomic.LoadInt64(&stf.failedRequests)

	// 计算响应时间统计
	stf.responseTimesMutex.RLock()
	var avgResponseTime, maxResponseTime, minResponseTime time.Duration
	if len(stf.responseTimes) > 0 {
		var total time.Duration
		maxResponseTime = stf.responseTimes[0]
		minResponseTime = stf.responseTimes[0]

		for _, rt := range stf.responseTimes {
			total += rt
			if rt > maxResponseTime {
				maxResponseTime = rt
			}
			if rt < minResponseTime {
				minResponseTime = rt
			}
		}
		avgResponseTime = total / time.Duration(len(stf.responseTimes))
	}
	stf.responseTimesMutex.RUnlock()

	// 计算吞吐量
	throughputRPS := float64(totalReq) / duration.Seconds()

	// 计算成功率和错误率
	var successRate, errorRate float64
	if totalReq > 0 {
		successRate = float64(successReq) / float64(totalReq)
		errorRate = float64(failedReq) / float64(totalReq)
	}

	// 复制错误分布
	stf.errorsMutex.RLock()
	errorDistribution := make(map[string]int64)
	for k, v := range stf.errors {
		errorDistribution[k] = v
	}
	stf.errorsMutex.RUnlock()

	return &StressTestResult{
		StartTime:           startTime,
		EndTime:             endTime,
		Duration:            duration,
		SimulatedDuration:   simulatedDuration,
		AccelerationFactor:  stf.config.AccelerationFactor,
		TotalRequests:       totalReq,
		SuccessfulRequests:  successReq,
		FailedRequests:      failedReq,
		AverageResponseTime: avgResponseTime,
		MaxResponseTime:     maxResponseTime,
		MinResponseTime:     minResponseTime,
		WorkflowExecutions:  atomic.LoadInt64(&stf.workflowExecutions),
		WorkflowSuccesses:   atomic.LoadInt64(&stf.workflowSuccesses),
		WorkflowFailures:    atomic.LoadInt64(&stf.workflowFailures),
		DataPointsGenerated: atomic.LoadInt64(&stf.dataPointsGenerated),
		SignalsGenerated:    atomic.LoadInt64(&stf.signalsGenerated),
		ThroughputRPS:       throughputRPS,
		ErrorRate:           errorRate,
		SuccessRate:         successRate,
		ErrorDistribution:   errorDistribution,
	}
}

// Stop 停止压力测试
func (stf *StressTestFramework) Stop() {
	stf.cancel()
}

// PerformanceMonitor 性能监控器
type PerformanceMonitor struct {
	metrics         map[string]*MetricCollector
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	monitorInterval time.Duration
}

// MetricCollector 指标收集器
type MetricCollector struct {
	Name         string      `json:"name"`
	Values       []float64   `json:"values"`
	Timestamps   []time.Time `json:"timestamps"`
	Min          float64     `json:"min"`
	Max          float64     `json:"max"`
	Average      float64     `json:"average"`
	Current      float64     `json:"current"`
	TotalSamples int64       `json:"total_samples"`
	mu           sync.RWMutex
}

// NewPerformanceMonitor 创建性能监控器
func NewPerformanceMonitor(monitorInterval time.Duration) *PerformanceMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	return &PerformanceMonitor{
		metrics:         make(map[string]*MetricCollector),
		ctx:             ctx,
		cancel:          cancel,
		monitorInterval: monitorInterval,
	}
}

// AddMetric 添加指标
func (pm *PerformanceMonitor) AddMetric(name string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.metrics[name] = &MetricCollector{
		Name:       name,
		Values:     make([]float64, 0, 1000),
		Timestamps: make([]time.Time, 0, 1000),
		Min:        math.MaxFloat64,
		Max:        -math.MaxFloat64,
	}
}

// RecordMetric 记录指标值
func (pm *PerformanceMonitor) RecordMetric(name string, value float64) {
	pm.mu.RLock()
	collector, exists := pm.metrics[name]
	pm.mu.RUnlock()

	if !exists {
		pm.AddMetric(name)
		pm.mu.RLock()
		collector = pm.metrics[name]
		pm.mu.RUnlock()
	}

	collector.mu.Lock()
	defer collector.mu.Unlock()

	// 添加值和时间戳
	collector.Values = append(collector.Values, value)
	collector.Timestamps = append(collector.Timestamps, time.Now())
	collector.Current = value
	collector.TotalSamples++

	// 更新统计信息
	if value < collector.Min {
		collector.Min = value
	}
	if value > collector.Max {
		collector.Max = value
	}

	// 计算平均值
	var sum float64
	for _, v := range collector.Values {
		sum += v
	}
	collector.Average = sum / float64(len(collector.Values))

	// 限制存储的数据点数量
	if len(collector.Values) > 1000 {
		collector.Values = collector.Values[100:]
		collector.Timestamps = collector.Timestamps[100:]
	}
}

// GetMetric 获取指标
func (pm *PerformanceMonitor) GetMetric(name string) *MetricCollector {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if collector, exists := pm.metrics[name]; exists {
		collector.mu.RLock()
		defer collector.mu.RUnlock()

		// 返回副本
		return &MetricCollector{
			Name:         collector.Name,
			Values:       append([]float64(nil), collector.Values...),
			Timestamps:   append([]time.Time(nil), collector.Timestamps...),
			Min:          collector.Min,
			Max:          collector.Max,
			Average:      collector.Average,
			Current:      collector.Current,
			TotalSamples: collector.TotalSamples,
		}
	}

	return nil
}

// GetAllMetrics 获取所有指标
func (pm *PerformanceMonitor) GetAllMetrics() map[string]*MetricCollector {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make(map[string]*MetricCollector)
	for name := range pm.metrics {
		result[name] = pm.GetMetric(name)
	}

	return result
}

// Start 启动监控
func (pm *PerformanceMonitor) Start() {
	go pm.monitorLoop()
}

// Stop 停止监控
func (pm *PerformanceMonitor) Stop() {
	pm.cancel()
}

// monitorLoop 监控循环
func (pm *PerformanceMonitor) monitorLoop() {
	ticker := time.NewTicker(pm.monitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pm.collectSystemMetrics()
		case <-pm.ctx.Done():
			return
		}
	}
}

// collectSystemMetrics 收集系统指标
func (pm *PerformanceMonitor) collectSystemMetrics() {
	// 收集内存使用情况
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	pm.RecordMetric("memory_alloc", float64(memStats.Alloc))
	pm.RecordMetric("memory_sys", float64(memStats.Sys))
	pm.RecordMetric("memory_heap_alloc", float64(memStats.HeapAlloc))
	pm.RecordMetric("memory_heap_sys", float64(memStats.HeapSys))

	// 收集Goroutine数量
	pm.RecordMetric("goroutines", float64(runtime.NumGoroutine()))

	// 收集GC统计
	pm.RecordMetric("gc_cycles", float64(memStats.NumGC))
	pm.RecordMetric("gc_pause_total", float64(memStats.PauseTotalNs))
}
