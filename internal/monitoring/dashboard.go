package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// DashboardManager manages the monitoring dashboard
type DashboardManager struct {
	// Prometheus metrics
	dashboardUpdates prometheus.Counter
	dashboardErrors  prometheus.Counter

	// Configuration
	config *DashboardConfig

	// State
	metrics map[string]*DashboardMetric
	mu      sync.RWMutex

	// Channels
	updateCh chan *MetricUpdate
	stopCh   chan struct{}
}

// DashboardConfig represents dashboard configuration
type DashboardConfig struct {
	UpdateInterval  time.Duration
	RetentionPeriod time.Duration
	MaxDataPoints   int
	AlertThresholds map[string]float64
	RefreshInterval time.Duration
	EnableRealTime  bool
}

// DashboardMetric represents a dashboard metric
type DashboardMetric struct {
	Name        string
	Value       float64
	Unit        string
	Type        MetricType
	Status      MetricStatus
	Trend       TrendDirection
	LastUpdated time.Time
	History     []DataPoint
	Thresholds  MetricThresholds
}

// MetricType represents the type of metric
type MetricType string

const (
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeCounter   MetricType = "counter"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
)

// MetricStatus represents the status of a metric
type MetricStatus string

const (
	MetricStatusNormal   MetricStatus = "normal"
	MetricStatusWarning  MetricStatus = "warning"
	MetricStatusCritical MetricStatus = "critical"
	MetricStatusUnknown  MetricStatus = "unknown"
)

// TrendDirection represents the trend direction
type TrendDirection string

const (
	TrendUp   TrendDirection = "up"
	TrendDown TrendDirection = "down"
	TrendFlat TrendDirection = "flat"
)

// DataPoint represents a single data point
type DataPoint struct {
	Timestamp time.Time
	Value     float64
}

// MetricThresholds represents metric thresholds
type MetricThresholds struct {
	Warning  float64
	Critical float64
}

// MetricUpdate represents a metric update
type MetricUpdate struct {
	Name  string
	Value float64
	Unit  string
	Type  MetricType
}

// DashboardData represents the complete dashboard data
type DashboardData struct {
	Timestamp time.Time
	Metrics   map[string]*DashboardMetric
	Summary   *DashboardSummary
}

// DashboardSummary represents dashboard summary
type DashboardSummary struct {
	TotalMetrics    int
	NormalMetrics   int
	WarningMetrics  int
	CriticalMetrics int
	UnknownMetrics  int
	LastUpdate      time.Time
}

// NewDashboardManager creates a new dashboard manager
func NewDashboardManager(config *DashboardConfig) *DashboardManager {
	if config == nil {
		config = &DashboardConfig{
			UpdateInterval:  5 * time.Second,
			RetentionPeriod: 24 * time.Hour,
			MaxDataPoints:   1000,
			RefreshInterval: 1 * time.Second,
			EnableRealTime:  true,
			AlertThresholds: make(map[string]float64),
		}
	}

	dm := &DashboardManager{
		config:   config,
		metrics:  make(map[string]*DashboardMetric),
		updateCh: make(chan *MetricUpdate, 100),
		stopCh:   make(chan struct{}),
	}

	// Initialize Prometheus metrics
	dm.initializeMetrics()

	return dm
}

// initializeMetrics initializes Prometheus metrics
func (dm *DashboardManager) initializeMetrics() {
	dm.dashboardUpdates = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dashboard_updates_total",
		Help: "Total number of dashboard updates",
	})

	dm.dashboardErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dashboard_errors_total",
		Help: "Total number of dashboard errors",
	})
}

// Start starts the dashboard manager
func (dm *DashboardManager) Start() {
	// Start metric update worker
	go dm.metricUpdateWorker()

	// Start cleanup worker
	go dm.cleanupWorker()

	// Initialize default metrics
	dm.initializeDefaultMetrics()
}

// Stop stops the dashboard manager
func (dm *DashboardManager) Stop() {
	close(dm.stopCh)
}

// UpdateMetric updates a metric value
func (dm *DashboardManager) UpdateMetric(name string, value float64, unit string, metricType MetricType) {
	update := &MetricUpdate{
		Name:  name,
		Value: value,
		Unit:  unit,
		Type:  metricType,
	}

	select {
	case dm.updateCh <- update:
		dm.dashboardUpdates.Inc()
	default:
		dm.dashboardErrors.Inc()
	}
}

// GetDashboardData returns the current dashboard data
func (dm *DashboardManager) GetDashboardData() *DashboardData {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	metrics := make(map[string]*DashboardMetric)
	for name, metric := range dm.metrics {
		metrics[name] = metric
	}

	return &DashboardData{
		Timestamp: time.Now(),
		Metrics:   metrics,
		Summary:   dm.calculateSummary(metrics),
	}
}

// GetMetric gets a specific metric
func (dm *DashboardManager) GetMetric(name string) *DashboardMetric {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.metrics[name]
}

// SetThreshold sets a threshold for a metric
func (dm *DashboardManager) SetThreshold(metricName string, warning, critical float64) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if metric, exists := dm.metrics[metricName]; exists {
		metric.Thresholds = MetricThresholds{
			Warning:  warning,
			Critical: critical,
		}
	}
}

// GetMetricsHistory returns the history of a metric
func (dm *DashboardManager) GetMetricsHistory(metricName string, duration time.Duration) []DataPoint {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	metric, exists := dm.metrics[metricName]
	if !exists {
		return nil
	}

	cutoff := time.Now().Add(-duration)
	var history []DataPoint

	for _, point := range metric.History {
		if point.Timestamp.After(cutoff) {
			history = append(history, point)
		}
	}

	return history
}

// metricUpdateWorker processes metric updates
func (dm *DashboardManager) metricUpdateWorker() {
	for {
		select {
		case <-dm.stopCh:
			return
		case update := <-dm.updateCh:
			dm.processMetricUpdate(update)
		}
	}
}

// processMetricUpdate processes a single metric update
func (dm *DashboardManager) processMetricUpdate(update *MetricUpdate) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	metric, exists := dm.metrics[update.Name]
	if !exists {
		metric = &DashboardMetric{
			Name:       update.Name,
			Type:       update.Type,
			Unit:       update.Unit,
			Status:     MetricStatusUnknown,
			Trend:      TrendFlat,
			History:    make([]DataPoint, 0),
			Thresholds: MetricThresholds{},
		}
		dm.metrics[update.Name] = metric
	}

	// Update metric value and calculate trend
	oldValue := metric.Value
	metric.Value = update.Value
	metric.LastUpdated = time.Now()

	// Calculate trend
	if oldValue != 0 {
		if update.Value > oldValue*1.05 {
			metric.Trend = TrendUp
		} else if update.Value < oldValue*0.95 {
			metric.Trend = TrendDown
		} else {
			metric.Trend = TrendFlat
		}
	}

	// Update status based on thresholds
	metric.Status = dm.calculateStatus(metric)

	// Add to history
	dataPoint := DataPoint{
		Timestamp: time.Now(),
		Value:     update.Value,
	}

	metric.History = append(metric.History, dataPoint)

	// Trim history if too long
	if len(metric.History) > dm.config.MaxDataPoints {
		metric.History = metric.History[len(metric.History)-dm.config.MaxDataPoints:]
	}
}

// calculateStatus calculates the status of a metric based on thresholds
func (dm *DashboardManager) calculateStatus(metric *DashboardMetric) MetricStatus {
	if metric.Thresholds.Critical > 0 && metric.Value >= metric.Thresholds.Critical {
		return MetricStatusCritical
	}
	if metric.Thresholds.Warning > 0 && metric.Value >= metric.Thresholds.Warning {
		return MetricStatusWarning
	}
	return MetricStatusNormal
}

// calculateSummary calculates dashboard summary
func (dm *DashboardManager) calculateSummary(metrics map[string]*DashboardMetric) *DashboardSummary {
	summary := &DashboardSummary{
		LastUpdate: time.Now(),
	}

	for _, metric := range metrics {
		summary.TotalMetrics++
		switch metric.Status {
		case MetricStatusNormal:
			summary.NormalMetrics++
		case MetricStatusWarning:
			summary.WarningMetrics++
		case MetricStatusCritical:
			summary.CriticalMetrics++
		default:
			summary.UnknownMetrics++
		}
	}

	return summary
}

// cleanupWorker cleans up old data points
func (dm *DashboardManager) cleanupWorker() {
	ticker := time.NewTicker(dm.config.RetentionPeriod / 10) // Clean up 10 times per retention period
	defer ticker.Stop()

	for {
		select {
		case <-dm.stopCh:
			return
		case <-ticker.C:
			dm.cleanupOldData()
		}
	}
}

// cleanupOldData removes old data points
func (dm *DashboardManager) cleanupOldData() {
	cutoff := time.Now().Add(-dm.config.RetentionPeriod)

	dm.mu.Lock()
	defer dm.mu.Unlock()

	for _, metric := range dm.metrics {
		var newHistory []DataPoint
		for _, point := range metric.History {
			if point.Timestamp.After(cutoff) {
				newHistory = append(newHistory, point)
			}
		}
		metric.History = newHistory
	}
}

// initializeDefaultMetrics initializes default metrics
func (dm *DashboardManager) initializeDefaultMetrics() {
	// System metrics
	dm.UpdateMetric("cpu_usage", 0, "%", MetricTypeGauge)
	dm.UpdateMetric("memory_usage", 0, "%", MetricTypeGauge)
	dm.UpdateMetric("disk_usage", 0, "%", MetricTypeGauge)
	dm.UpdateMetric("network_io", 0, "MB/s", MetricTypeGauge)

	// Application metrics
	dm.UpdateMetric("request_rate", 0, "req/s", MetricTypeGauge)
	dm.UpdateMetric("error_rate", 0, "errors/s", MetricTypeGauge)
	dm.UpdateMetric("response_time", 0, "ms", MetricTypeGauge)
	dm.UpdateMetric("active_connections", 0, "connections", MetricTypeGauge)

	// Database metrics
	dm.UpdateMetric("db_connections", 0, "connections", MetricTypeGauge)
	dm.UpdateMetric("db_query_time", 0, "ms", MetricTypeGauge)
	dm.UpdateMetric("db_errors", 0, "errors/s", MetricTypeGauge)

	// Redis metrics
	dm.UpdateMetric("redis_memory", 0, "MB", MetricTypeGauge)
	dm.UpdateMetric("redis_commands", 0, "commands/s", MetricTypeGauge)
	dm.UpdateMetric("redis_errors", 0, "errors/s", MetricTypeGauge)

	// Set default thresholds
	dm.SetThreshold("cpu_usage", 80, 95)
	dm.SetThreshold("memory_usage", 80, 95)
	dm.SetThreshold("disk_usage", 85, 95)
	dm.SetThreshold("error_rate", 5, 10)
	dm.SetThreshold("response_time", 1000, 5000)
}

// GetDashboardJSON returns dashboard data as JSON
func (dm *DashboardManager) GetDashboardJSON() ([]byte, error) {
	data := dm.GetDashboardData()
	return json.MarshalIndent(data, "", "  ")
}

// GetMetricsJSON returns metrics data as JSON
func (dm *DashboardManager) GetMetricsJSON() ([]byte, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	return json.MarshalIndent(dm.metrics, "", "  ")
}

// GetMetricJSON returns a specific metric as JSON
func (dm *DashboardManager) GetMetricJSON(name string) ([]byte, error) {
	metric := dm.GetMetric(name)
	if metric == nil {
		return nil, fmt.Errorf("metric %s not found", name)
	}

	return json.MarshalIndent(metric, "", "  ")
}

// GetSummaryJSON returns dashboard summary as JSON
func (dm *DashboardManager) GetSummaryJSON() ([]byte, error) {
	data := dm.GetDashboardData()
	return json.MarshalIndent(data.Summary, "", "  ")
}

// StartRealTimeUpdates starts real-time metric updates
func (dm *DashboardManager) StartRealTimeUpdates(ctx context.Context) {
	if !dm.config.EnableRealTime {
		return
	}

	ticker := time.NewTicker(dm.config.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dm.updateRealTimeMetrics()
		}
	}
}

// updateRealTimeMetrics updates real-time metrics
func (dm *DashboardManager) updateRealTimeMetrics() {
	// This is a simplified implementation
	// In a real system, you would collect actual system metrics

	// Simulate CPU usage
	cpuUsage := 50.0 + float64(time.Now().Unix()%30)
	dm.UpdateMetric("cpu_usage", cpuUsage, "%", MetricTypeGauge)

	// Simulate memory usage
	memoryUsage := 60.0 + float64(time.Now().Unix()%20)
	dm.UpdateMetric("memory_usage", memoryUsage, "%", MetricTypeGauge)

	// Simulate request rate
	requestRate := 100.0 + float64(time.Now().Unix()%50)
	dm.UpdateMetric("request_rate", requestRate, "req/s", MetricTypeGauge)

	// Simulate response time
	responseTime := 50.0 + float64(time.Now().Unix()%100)
	dm.UpdateMetric("response_time", responseTime, "ms", MetricTypeGauge)
}
