package testutils

import (
	"fmt"
	"runtime"
	"testing"
	"time"
)

// BenchmarkSuite 性能测试套件
type BenchmarkSuite struct {
	B       *testing.B
	Suite   *TestSuite
	Metrics *BenchmarkMetrics
}

// BenchmarkMetrics 性能测试指标
type BenchmarkMetrics struct {
	StartTime     time.Time
	EndTime       time.Time
	Duration      time.Duration
	Operations    int64
	BytesPerOp    int64
	AllocsPerOp   int64
	MemAllocated  uint64
	MemSys        uint64
	GCRuns        uint32
	CustomMetrics map[string]interface{}
}

// NewBenchmarkSuite 创建性能测试套件
func NewBenchmarkSuite(b *testing.B, config *TestConfig) *BenchmarkSuite {
	suite := NewTestSuite(&testing.T{}, config)
	
	return &BenchmarkSuite{
		B:     b,
		Suite: suite,
		Metrics: &BenchmarkMetrics{
			CustomMetrics: make(map[string]interface{}),
		},
	}
}

// StartBenchmark 开始性能测试
func (bs *BenchmarkSuite) StartBenchmark() {
	runtime.GC() // 强制垃圾回收
	
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	
	bs.Metrics.StartTime = time.Now()
	bs.Metrics.MemAllocated = m1.Alloc
	bs.Metrics.MemSys = m1.Sys
	bs.Metrics.GCRuns = m1.NumGC
}

// EndBenchmark 结束性能测试
func (bs *BenchmarkSuite) EndBenchmark() {
	bs.Metrics.EndTime = time.Now()
	bs.Metrics.Duration = bs.Metrics.EndTime.Sub(bs.Metrics.StartTime)
	
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	
	bs.Metrics.AllocsPerOp = int64(m2.Mallocs - bs.Metrics.MemAllocated)
	bs.Metrics.BytesPerOp = int64(m2.TotalAlloc - bs.Metrics.MemAllocated)
	bs.Metrics.GCRuns = m2.NumGC - bs.Metrics.GCRuns
}

// RecordCustomMetric 记录自定义指标
func (bs *BenchmarkSuite) RecordCustomMetric(name string, value interface{}) {
	bs.Metrics.CustomMetrics[name] = value
}

// ReportMetrics 报告性能指标
func (bs *BenchmarkSuite) ReportMetrics() {
	bs.B.ReportMetric(float64(bs.Metrics.BytesPerOp), "B/op")
	bs.B.ReportMetric(float64(bs.Metrics.AllocsPerOp), "allocs/op")
	bs.B.ReportMetric(float64(bs.Metrics.Duration.Nanoseconds()), "ns/op")
	
	for name, value := range bs.Metrics.CustomMetrics {
		if v, ok := value.(float64); ok {
			bs.B.ReportMetric(v, name)
		}
	}
}

// BenchmarkFunction 性能测试函数类型
type BenchmarkFunction func(b *testing.B, suite *BenchmarkSuite)

// RunBenchmark 运行性能测试
func RunBenchmark(b *testing.B, name string, config *TestConfig, fn BenchmarkFunction) {
	b.Run(name, func(b *testing.B) {
		suite := NewBenchmarkSuite(b, config)
		defer suite.Suite.TearDown()
		
		b.ResetTimer()
		suite.StartBenchmark()
		
		fn(b, suite)
		
		suite.EndBenchmark()
		suite.ReportMetrics()
	})
}

// LoadTestConfig 负载测试配置
type LoadTestConfig struct {
	Concurrency int           // 并发数
	Duration    time.Duration // 测试持续时间
	RampUp      time.Duration // 预热时间
	RampDown    time.Duration // 冷却时间
	QPS         int           // 每秒请求数
}

// LoadTestResult 负载测试结果
type LoadTestResult struct {
	TotalRequests    int64
	SuccessRequests  int64
	FailedRequests   int64
	AverageLatency   time.Duration
	MinLatency       time.Duration
	MaxLatency       time.Duration
	P50Latency       time.Duration
	P95Latency       time.Duration
	P99Latency       time.Duration
	ThroughputQPS    float64
	ErrorRate        float64
	StartTime        time.Time
	EndTime          time.Time
	Duration         time.Duration
}

// LoadTestRunner 负载测试运行器
type LoadTestRunner struct {
	Config  *LoadTestConfig
	Results *LoadTestResult
	Suite   *TestSuite
}

// NewLoadTestRunner 创建负载测试运行器
func NewLoadTestRunner(config *LoadTestConfig, suite *TestSuite) *LoadTestRunner {
	return &LoadTestRunner{
		Config: config,
		Results: &LoadTestResult{
			MinLatency: time.Hour, // 初始化为很大的值
		},
		Suite: suite,
	}
}

// RunLoadTest 运行负载测试
func (ltr *LoadTestRunner) RunLoadTest(testFunc func() error) *LoadTestResult {
	ltr.Results.StartTime = time.Now()
	
	// 这里可以实现负载测试逻辑
	// 简化实现，实际项目中可以使用更复杂的负载测试框架
	
	for i := 0; i < ltr.Config.Concurrency; i++ {
		go func() {
			for time.Since(ltr.Results.StartTime) < ltr.Config.Duration {
				start := time.Now()
				err := testFunc()
				latency := time.Since(start)
				
				ltr.Results.TotalRequests++
				if err != nil {
					ltr.Results.FailedRequests++
				} else {
					ltr.Results.SuccessRequests++
				}
				
				// 更新延迟统计
				if latency < ltr.Results.MinLatency {
					ltr.Results.MinLatency = latency
				}
				if latency > ltr.Results.MaxLatency {
					ltr.Results.MaxLatency = latency
				}
				
				// 控制QPS
				if ltr.Config.QPS > 0 {
					time.Sleep(time.Second / time.Duration(ltr.Config.QPS))
				}
			}
		}()
	}
	
	// 等待测试完成
	time.Sleep(ltr.Config.Duration)
	
	ltr.Results.EndTime = time.Now()
	ltr.Results.Duration = ltr.Results.EndTime.Sub(ltr.Results.StartTime)
	
	// 计算统计数据
	if ltr.Results.TotalRequests > 0 {
		ltr.Results.ErrorRate = float64(ltr.Results.FailedRequests) / float64(ltr.Results.TotalRequests)
		ltr.Results.ThroughputQPS = float64(ltr.Results.TotalRequests) / ltr.Results.Duration.Seconds()
	}
	
	return ltr.Results
}

// PrintLoadTestResults 打印负载测试结果
func (ltr *LoadTestRunner) PrintLoadTestResults() {
	fmt.Printf("=== Load Test Results ===\n")
	fmt.Printf("Duration: %v\n", ltr.Results.Duration)
	fmt.Printf("Total Requests: %d\n", ltr.Results.TotalRequests)
	fmt.Printf("Success Requests: %d\n", ltr.Results.SuccessRequests)
	fmt.Printf("Failed Requests: %d\n", ltr.Results.FailedRequests)
	fmt.Printf("Error Rate: %.2f%%\n", ltr.Results.ErrorRate*100)
	fmt.Printf("Throughput: %.2f QPS\n", ltr.Results.ThroughputQPS)
	fmt.Printf("Min Latency: %v\n", ltr.Results.MinLatency)
	fmt.Printf("Max Latency: %v\n", ltr.Results.MaxLatency)
	fmt.Printf("Average Latency: %v\n", ltr.Results.AverageLatency)
}
