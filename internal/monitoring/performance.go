package monitoring

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PerformanceMonitor monitors system performance metrics
type PerformanceMonitor struct {
	// Prometheus metrics
	requestDuration    prometheus.Histogram
	requestRate        prometheus.Counter
	errorRate          prometheus.Counter
	throughput         prometheus.Gauge
	concurrentRequests prometheus.Gauge
	responseTimeP95    prometheus.Gauge
	responseTimeP99    prometheus.Gauge

	// Performance baselines
	baselines map[string]*PerformanceBaseline
	mu        sync.RWMutex

	// Configuration
	config *PerformanceConfig
}

// PerformanceConfig represents performance monitoring configuration
type PerformanceConfig struct {
	BaselineWindow      time.Duration
	AlertThreshold      float64
	P95Threshold        time.Duration
	P99Threshold        time.Duration
	ErrorRateThreshold  float64
	ThroughputThreshold float64
}

// PerformanceBaseline represents a performance baseline
type PerformanceBaseline struct {
	MetricName     string
	AverageValue   float64
	P95Value       float64
	P99Value       float64
	MinValue       float64
	MaxValue       float64
	SampleCount    int64
	LastUpdated    time.Time
	AlertThreshold float64
}

// PerformanceMetrics represents current performance metrics
type PerformanceMetrics struct {
	RequestDuration    time.Duration
	RequestRate        float64
	ErrorRate          float64
	Throughput         float64
	ConcurrentRequests int64
	ResponseTimeP95    time.Duration
	ResponseTimeP99    time.Duration
	Timestamp          time.Time
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(config *PerformanceConfig) *PerformanceMonitor {
	if config == nil {
		config = &PerformanceConfig{
			BaselineWindow:      24 * time.Hour,
			AlertThreshold:      2.0, // 2x baseline
			P95Threshold:        1 * time.Second,
			P99Threshold:        2 * time.Second,
			ErrorRateThreshold:  0.05, // 5%
			ThroughputThreshold: 1000, // requests per second
		}
	}

	pm := &PerformanceMonitor{
		config:    config,
		baselines: make(map[string]*PerformanceBaseline),
	}

	// Initialize Prometheus metrics
	pm.initializeMetrics()

	return pm
}

// initializeMetrics initializes Prometheus metrics
func (pm *PerformanceMonitor) initializeMetrics() {
	pm.requestDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "request_duration_seconds",
		Help:    "Request duration in seconds",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 15),
	})

	pm.requestRate = promauto.NewCounter(prometheus.CounterOpts{
		Name: "request_total",
		Help: "Total number of requests",
	})

	pm.errorRate = promauto.NewCounter(prometheus.CounterOpts{
		Name: "error_total",
		Help: "Total number of errors",
	})

	pm.throughput = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "throughput_requests_per_second",
		Help: "Current throughput in requests per second",
	})

	pm.concurrentRequests = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "concurrent_requests",
		Help: "Number of concurrent requests",
	})

	pm.responseTimeP95 = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "response_time_p95_seconds",
		Help: "95th percentile response time in seconds",
	})

	pm.responseTimeP99 = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "response_time_p99_seconds",
		Help: "99th percentile response time in seconds",
	})
}

// RecordRequest records a request with its duration
func (pm *PerformanceMonitor) RecordRequest(duration time.Duration, success bool) {
	pm.requestDuration.Observe(duration.Seconds())
	pm.requestRate.Inc()

	if !success {
		pm.errorRate.Inc()
	}
}

// RecordConcurrentRequests records the number of concurrent requests
func (pm *PerformanceMonitor) RecordConcurrentRequests(count int64) {
	pm.concurrentRequests.Set(float64(count))
}

// UpdateThroughput updates the current throughput
func (pm *PerformanceMonitor) UpdateThroughput(throughput float64) {
	pm.throughput.Set(throughput)
}

// UpdateResponseTimePercentiles updates response time percentiles
func (pm *PerformanceMonitor) UpdateResponseTimePercentiles(p95, p99 time.Duration) {
	pm.responseTimeP95.Set(p95.Seconds())
	pm.responseTimeP99.Set(p99.Seconds())
}

// CreateBaseline creates a performance baseline for a metric
func (pm *PerformanceMonitor) CreateBaseline(metricName string, values []float64) error {
	if len(values) == 0 {
		return fmt.Errorf("no values provided for baseline")
	}

	// Calculate statistics
	var sum float64
	min := values[0]
	max := values[0]

	for _, value := range values {
		sum += value
		if value < min {
			min = value
		}
		if value > max {
			max = value
		}
	}

	average := sum / float64(len(values))

	// Calculate percentiles (simplified)
	sorted := make([]float64, len(values))
	copy(sorted, values)
	// Note: In a real implementation, you would sort the values and calculate proper percentiles

	p95 := sorted[len(sorted)*95/100]
	p99 := sorted[len(sorted)*99/100]

	baseline := &PerformanceBaseline{
		MetricName:     metricName,
		AverageValue:   average,
		P95Value:       p95,
		P99Value:       p99,
		MinValue:       min,
		MaxValue:       max,
		SampleCount:    int64(len(values)),
		LastUpdated:    time.Now(),
		AlertThreshold: pm.config.AlertThreshold,
	}

	pm.mu.Lock()
	pm.baselines[metricName] = baseline
	pm.mu.Unlock()

	return nil
}

// GetBaseline gets a performance baseline for a metric
func (pm *PerformanceMonitor) GetBaseline(metricName string) *PerformanceBaseline {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.baselines[metricName]
}

// CheckPerformanceAlert checks if current performance exceeds baseline thresholds
func (pm *PerformanceMonitor) CheckPerformanceAlert(metricName string, currentValue float64) (bool, string) {
	baseline := pm.GetBaseline(metricName)
	if baseline == nil {
		return false, "No baseline available"
	}

	threshold := baseline.AverageValue * baseline.AlertThreshold
	if currentValue > threshold {
		return true, fmt.Sprintf("Performance alert: %s current value %.2f exceeds threshold %.2f (baseline: %.2f)",
			metricName, currentValue, threshold, baseline.AverageValue)
	}

	return false, ""
}

// GetPerformanceMetrics returns current performance metrics
func (pm *PerformanceMonitor) GetPerformanceMetrics() *PerformanceMetrics {
	// This is a simplified implementation
	// In a real system, you would calculate these from actual measurements
	return &PerformanceMetrics{
		RequestDuration:    100 * time.Millisecond,
		RequestRate:        100.0,
		ErrorRate:          0.01,
		Throughput:         1000.0,
		ConcurrentRequests: 50,
		ResponseTimeP95:    200 * time.Millisecond,
		ResponseTimeP99:    500 * time.Millisecond,
		Timestamp:          time.Now(),
	}
}

// MonitorFunction monitors the performance of a function
func (pm *PerformanceMonitor) MonitorFunction(ctx context.Context, functionName string, fn func() error) error {
	start := time.Now()
	pm.RecordConcurrentRequests(1)
	defer pm.RecordConcurrentRequests(-1)

	err := fn()
	duration := time.Since(start)

	pm.RecordRequest(duration, err == nil)

	// Check for performance alerts
	baseline := pm.GetBaseline(functionName)
	if baseline != nil {
		if alert, message := pm.CheckPerformanceAlert(functionName, duration.Seconds()); alert {
			// Log performance alert
			fmt.Printf("Performance Alert: %s\n", message)
		}
	}

	return err
}

// MonitorFunctionWithResult monitors the performance of a function with result
func (pm *PerformanceMonitor) MonitorFunctionWithResult(ctx context.Context, functionName string, fn func() (interface{}, error)) (interface{}, error) {
	start := time.Now()
	pm.RecordConcurrentRequests(1)
	defer pm.RecordConcurrentRequests(-1)

	result, err := fn()
	duration := time.Since(start)

	pm.RecordRequest(duration, err == nil)

	// Check for performance alerts
	baseline := pm.GetBaseline(functionName)
	if baseline != nil {
		if alert, message := pm.CheckPerformanceAlert(functionName, duration.Seconds()); alert {
			// Log performance alert
			fmt.Printf("Performance Alert: %s\n", message)
		}
	}

	return result, err
}

// StartBaselineCollection starts collecting baseline data
func (pm *PerformanceMonitor) StartBaselineCollection(ctx context.Context) {
	ticker := time.NewTicker(pm.config.BaselineWindow / 24) // Collect 24 samples per window
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pm.collectBaselineData()
		}
	}
}

// collectBaselineData collects baseline data for all metrics
func (pm *PerformanceMonitor) collectBaselineData() {
	// This is a simplified implementation
	// In a real system, you would collect actual metrics data
	metrics := pm.GetPerformanceMetrics()

	// Update baselines with current metrics
	pm.updateBaseline("request_duration", metrics.RequestDuration.Seconds())
	pm.updateBaseline("throughput", metrics.Throughput)
	pm.updateBaseline("error_rate", metrics.ErrorRate)
}

// updateBaseline updates a baseline with a new value
func (pm *PerformanceMonitor) updateBaseline(metricName string, value float64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	baseline, exists := pm.baselines[metricName]
	if !exists {
		baseline = &PerformanceBaseline{
			MetricName:     metricName,
			AlertThreshold: pm.config.AlertThreshold,
		}
		pm.baselines[metricName] = baseline
	}

	// Update baseline statistics (simplified moving average)
	baseline.SampleCount++
	alpha := 1.0 / float64(baseline.SampleCount)
	baseline.AverageValue = baseline.AverageValue*(1-alpha) + value*alpha

	if value < baseline.MinValue {
		baseline.MinValue = value
	}
	if value > baseline.MaxValue {
		baseline.MaxValue = value
	}

	baseline.LastUpdated = time.Now()
}

// GetBaselineSummary returns a summary of all baselines
func (pm *PerformanceMonitor) GetBaselineSummary() map[string]*PerformanceBaseline {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	summary := make(map[string]*PerformanceBaseline)
	for name, baseline := range pm.baselines {
		summary[name] = baseline
	}

	return summary
}

// ResetBaselines resets all performance baselines
func (pm *PerformanceMonitor) ResetBaselines() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.baselines = make(map[string]*PerformanceBaseline)
}
