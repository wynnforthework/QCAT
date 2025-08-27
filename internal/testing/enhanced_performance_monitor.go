package testing

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// EnhancedPerformanceMonitor 增强版性能监控器
type EnhancedPerformanceMonitor struct {
	// 基础配置
	config *PerformanceMonitorConfig

	// 指标收集器
	systemMetrics      *SystemMetricsCollector
	networkMetrics     *NetworkMetricsCollector
	applicationMetrics *ApplicationMetricsCollector
	customMetrics      map[string]*EnhancedMetricCollector
	customMetricsMutex sync.RWMutex

	// 报告生成器
	reportGenerator *PerformanceReportGenerator

	// 告警系统
	alertManager *AlertManager

	// 控制
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	running      bool
	runningMutex sync.RWMutex

	// 统计
	stats      *MonitoringStats
	statsMutex sync.RWMutex
}

// PerformanceMonitorConfig 性能监控配置
type PerformanceMonitorConfig struct {
	// 基础配置
	MonitorInterval time.Duration `json:"monitor_interval"`
	ReportInterval  time.Duration `json:"report_interval"`

	// 系统监控配置
	EnableSystemMetrics  bool `json:"enable_system_metrics"`
	EnableNetworkMetrics bool `json:"enable_network_metrics"`
	EnableAppMetrics     bool `json:"enable_app_metrics"`

	// 数据保留配置
	MaxDataPoints int           `json:"max_data_points"`
	DataRetention time.Duration `json:"data_retention"`

	// 告警配置
	EnableAlerts    bool                      `json:"enable_alerts"`
	AlertThresholds map[string]AlertThreshold `json:"alert_thresholds"`

	// 报告配置
	EnableReports   bool     `json:"enable_reports"`
	ReportFormats   []string `json:"report_formats"` // json, csv, html
	ReportOutputDir string   `json:"report_output_dir"`

	// 网络监控配置
	NetworkInterfaces []string `json:"network_interfaces"`
	PingTargets       []string `json:"ping_targets"`

	// 高级配置
	EnableProfiling   bool          `json:"enable_profiling"`
	ProfilingInterval time.Duration `json:"profiling_interval"`
}

// AlertThreshold 告警阈值
type AlertThreshold struct {
	MetricName    string  `json:"metric_name"`
	WarningLevel  float64 `json:"warning_level"`
	CriticalLevel float64 `json:"critical_level"`
	Enabled       bool    `json:"enabled"`
}

// EnhancedMetricCollector 增强版指标收集器
type EnhancedMetricCollector struct {
	Name        string     `json:"name"`
	Unit        string     `json:"unit"`
	Description string     `json:"description"`
	MetricType  MetricType `json:"metric_type"`

	// 数据存储
	Values     []float64         `json:"values"`
	Timestamps []time.Time       `json:"timestamps"`
	Labels     map[string]string `json:"labels"`

	// 统计信息
	Count  int64   `json:"count"`
	Sum    float64 `json:"sum"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Mean   float64 `json:"mean"`
	StdDev float64 `json:"std_dev"`

	// 百分位数
	P50 float64 `json:"p50"`
	P90 float64 `json:"p90"`
	P95 float64 `json:"p95"`
	P99 float64 `json:"p99"`

	// 趋势分析
	Trend         TrendDirection `json:"trend"`
	TrendStrength float64        `json:"trend_strength"`

	// 控制
	mutex      sync.RWMutex
	lastUpdate time.Time
}

// MetricType 指标类型
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
)

// TrendDirection 趋势方向
type TrendDirection string

const (
	TrendUp      TrendDirection = "up"
	TrendDown    TrendDirection = "down"
	TrendStable  TrendDirection = "stable"
	TrendUnknown TrendDirection = "unknown"
)

// SystemMetricsCollector 系统指标收集器
type SystemMetricsCollector struct {
	cpuUsage       *EnhancedMetricCollector
	memoryUsage    *EnhancedMetricCollector
	diskUsage      *EnhancedMetricCollector
	goroutineCount *EnhancedMetricCollector
	gcStats        *EnhancedMetricCollector

	// 系统信息
	startTime    time.Time
	lastCPUTime  time.Time
	lastCPUUsage float64
}

// NetworkMetricsCollector 网络指标收集器
type NetworkMetricsCollector struct {
	latency     map[string]*EnhancedMetricCollector
	bandwidth   map[string]*EnhancedMetricCollector
	packetLoss  map[string]*EnhancedMetricCollector
	connections *EnhancedMetricCollector

	// 网络接口统计
	interfaces      map[string]*NetworkInterfaceStats
	interfacesMutex sync.RWMutex
}

// NetworkInterfaceStats 网络接口统计
type NetworkInterfaceStats struct {
	Name            string    `json:"name"`
	BytesReceived   uint64    `json:"bytes_received"`
	BytesSent       uint64    `json:"bytes_sent"`
	PacketsReceived uint64    `json:"packets_received"`
	PacketsSent     uint64    `json:"packets_sent"`
	Errors          uint64    `json:"errors"`
	Drops           uint64    `json:"drops"`
	LastUpdate      time.Time `json:"last_update"`
}

// ApplicationMetricsCollector 应用指标收集器
type ApplicationMetricsCollector struct {
	requestRate  *EnhancedMetricCollector
	responseTime *EnhancedMetricCollector
	errorRate    *EnhancedMetricCollector
	throughput   *EnhancedMetricCollector

	// 业务指标
	activeStrategies *EnhancedMetricCollector
	tradingVolume    *EnhancedMetricCollector
	profitLoss       *EnhancedMetricCollector
}

// MonitoringStats 监控统计信息
type MonitoringStats struct {
	StartTime        time.Time     `json:"start_time"`
	Uptime           time.Duration `json:"uptime"`
	TotalMetrics     int           `json:"total_metrics"`
	TotalDataPoints  int64         `json:"total_data_points"`
	ReportsGenerated int64         `json:"reports_generated"`
	AlertsTriggered  int64         `json:"alerts_triggered"`
	LastReportTime   time.Time     `json:"last_report_time"`
	LastAlertTime    time.Time     `json:"last_alert_time"`
}

// DefaultPerformanceMonitorConfig 默认配置
func DefaultPerformanceMonitorConfig() *PerformanceMonitorConfig {
	return &PerformanceMonitorConfig{
		MonitorInterval: time.Second,
		ReportInterval:  5 * time.Minute,

		EnableSystemMetrics:  true,
		EnableNetworkMetrics: true,
		EnableAppMetrics:     true,

		MaxDataPoints: 10000,
		DataRetention: 24 * time.Hour,

		EnableAlerts: true,
		AlertThresholds: map[string]AlertThreshold{
			"cpu_usage": {
				MetricName:    "cpu_usage",
				WarningLevel:  80.0,
				CriticalLevel: 95.0,
				Enabled:       true,
			},
			"memory_usage": {
				MetricName:    "memory_usage",
				WarningLevel:  80.0,
				CriticalLevel: 95.0,
				Enabled:       true,
			},
			"response_time": {
				MetricName:    "response_time",
				WarningLevel:  1000.0, // 1秒
				CriticalLevel: 5000.0, // 5秒
				Enabled:       true,
			},
		},

		EnableReports:   true,
		ReportFormats:   []string{"json", "csv"},
		ReportOutputDir: "./reports",

		NetworkInterfaces: []string{"eth0", "lo"},
		PingTargets:       []string{"8.8.8.8", "1.1.1.1"},

		EnableProfiling:   false,
		ProfilingInterval: 10 * time.Minute,
	}
}

// NewEnhancedPerformanceMonitor 创建增强版性能监控器
func NewEnhancedPerformanceMonitor(config *PerformanceMonitorConfig) (*EnhancedPerformanceMonitor, error) {
	if config == nil {
		config = DefaultPerformanceMonitorConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	monitor := &EnhancedPerformanceMonitor{
		config:        config,
		customMetrics: make(map[string]*EnhancedMetricCollector),
		ctx:           ctx,
		cancel:        cancel,
		running:       false,
		stats: &MonitoringStats{
			StartTime: time.Now(),
		},
	}

	// 初始化系统指标收集器
	if config.EnableSystemMetrics {
		monitor.systemMetrics = NewSystemMetricsCollector()
	}

	// 初始化网络指标收集器
	if config.EnableNetworkMetrics {
		monitor.networkMetrics = NewNetworkMetricsCollector(config.NetworkInterfaces, config.PingTargets)
	}

	// 初始化应用指标收集器
	if config.EnableAppMetrics {
		monitor.applicationMetrics = NewApplicationMetricsCollector()
	}

	// 初始化报告生成器
	if config.EnableReports {
		monitor.reportGenerator = NewPerformanceReportGenerator(config)
	}

	// 初始化告警管理器
	if config.EnableAlerts {
		monitor.alertManager = NewAlertManager(config.AlertThresholds)
	}

	return monitor, nil
}

// Start 启动性能监控器
func (epm *EnhancedPerformanceMonitor) Start() error {
	epm.runningMutex.Lock()
	defer epm.runningMutex.Unlock()

	if epm.running {
		return fmt.Errorf("性能监控器已经在运行")
	}

	epm.running = true
	epm.stats.StartTime = time.Now()

	log.Printf("🚀 启动增强版性能监控器")

	// 启动监控循环
	epm.wg.Add(1)
	go epm.monitoringLoop()

	// 启动报告生成循环
	if epm.config.EnableReports && epm.reportGenerator != nil {
		epm.wg.Add(1)
		go epm.reportingLoop()
	}

	// 启动告警检查循环
	if epm.config.EnableAlerts && epm.alertManager != nil {
		epm.wg.Add(1)
		go epm.alertingLoop()
	}

	return nil
}

// Stop 停止性能监控器
func (epm *EnhancedPerformanceMonitor) Stop() {
	epm.runningMutex.Lock()
	defer epm.runningMutex.Unlock()

	if !epm.running {
		return
	}

	log.Printf("🛑 停止增强版性能监控器...")

	epm.running = false
	epm.cancel()
	epm.wg.Wait()

	// 生成最终报告
	if epm.reportGenerator != nil {
		epm.generateFinalReport()
	}

	log.Printf("✅ 性能监控器已停止")
}

// monitoringLoop 监控循环
func (epm *EnhancedPerformanceMonitor) monitoringLoop() {
	defer epm.wg.Done()

	ticker := time.NewTicker(epm.config.MonitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			epm.collectAllMetrics()
			epm.updateStats()
		case <-epm.ctx.Done():
			return
		}
	}
}

// collectAllMetrics 收集所有指标
func (epm *EnhancedPerformanceMonitor) collectAllMetrics() {
	now := time.Now()

	// 收集系统指标
	if epm.systemMetrics != nil {
		epm.collectSystemMetrics(now)
	}

	// 收集网络指标
	if epm.networkMetrics != nil {
		epm.collectNetworkMetrics(now)
	}

	// 收集应用指标
	if epm.applicationMetrics != nil {
		epm.collectApplicationMetrics(now)
	}

	// 清理过期数据
	epm.cleanupExpiredData(now)
}

// collectSystemMetrics 收集系统指标
func (epm *EnhancedPerformanceMonitor) collectSystemMetrics(timestamp time.Time) {
	// 收集内存使用情况
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	memoryUsagePercent := float64(memStats.Alloc) / float64(memStats.Sys) * 100
	epm.systemMetrics.memoryUsage.RecordValue(memoryUsagePercent, timestamp)

	// 收集Goroutine数量
	goroutineCount := float64(runtime.NumGoroutine())
	epm.systemMetrics.goroutineCount.RecordValue(goroutineCount, timestamp)

	// 收集GC统计
	gcPauseTotal := float64(memStats.PauseTotalNs) / 1e6 // 转换为毫秒
	epm.systemMetrics.gcStats.RecordValue(gcPauseTotal, timestamp)

	// 收集CPU使用率（简化版本）
	cpuUsage := epm.calculateCPUUsage()
	epm.systemMetrics.cpuUsage.RecordValue(cpuUsage, timestamp)
}

// calculateCPUUsage 计算CPU使用率
func (epm *EnhancedPerformanceMonitor) calculateCPUUsage() float64 {
	// 这是一个简化的CPU使用率计算
	// 在实际应用中，可能需要使用更精确的方法
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// 基于GC暂停时间和系统调用来估算CPU使用率
	now := time.Now()
	if epm.systemMetrics.lastCPUTime.IsZero() {
		epm.systemMetrics.lastCPUTime = now
		return 0
	}

	elapsed := now.Sub(epm.systemMetrics.lastCPUTime).Seconds()
	if elapsed == 0 {
		return epm.systemMetrics.lastCPUUsage
	}

	// 简化的CPU使用率计算（基于Goroutine数量和内存分配）
	cpuUsage := math.Min(float64(runtime.NumGoroutine())/100.0*10.0, 100.0)

	epm.systemMetrics.lastCPUTime = now
	epm.systemMetrics.lastCPUUsage = cpuUsage

	return cpuUsage
}

// collectNetworkMetrics 收集网络指标
func (epm *EnhancedPerformanceMonitor) collectNetworkMetrics(timestamp time.Time) {
	// 收集网络延迟
	for _, target := range epm.config.PingTargets {
		latency := epm.measureLatency(target)
		if collector, exists := epm.networkMetrics.latency[target]; exists {
			collector.RecordValue(latency, timestamp)
		}
	}

	// 收集网络连接数
	connectionCount := epm.countNetworkConnections()
	epm.networkMetrics.connections.RecordValue(float64(connectionCount), timestamp)
}

// measureLatency 测量网络延迟
func (epm *EnhancedPerformanceMonitor) measureLatency(target string) float64 {
	start := time.Now()

	conn, err := net.DialTimeout("tcp", target+":80", 5*time.Second)
	if err != nil {
		return -1 // 表示连接失败
	}
	defer conn.Close()

	return float64(time.Since(start).Nanoseconds()) / 1e6 // 转换为毫秒
}

// countNetworkConnections 统计网络连接数
func (epm *EnhancedPerformanceMonitor) countNetworkConnections() int {
	// 这是一个简化的实现
	// 在实际应用中，可能需要读取 /proc/net/tcp 等系统文件
	return runtime.NumGoroutine() / 10 // 简化估算
}

// collectApplicationMetrics 收集应用指标
func (epm *EnhancedPerformanceMonitor) collectApplicationMetrics(timestamp time.Time) {
	// 这些指标通常由应用程序主动报告
	// 这里提供默认值，实际使用时应该由应用程序调用RecordMetric方法

	// 示例：记录当前活跃策略数量
	// epm.applicationMetrics.activeStrategies.RecordValue(activeCount, timestamp)
}

// updateStats 更新统计信息
func (epm *EnhancedPerformanceMonitor) updateStats() {
	epm.statsMutex.Lock()
	defer epm.statsMutex.Unlock()

	epm.stats.Uptime = time.Since(epm.stats.StartTime)
	epm.stats.TotalMetrics = epm.getTotalMetricsCount()
	epm.stats.TotalDataPoints = epm.getTotalDataPointsCount()
}

// getTotalMetricsCount 获取总指标数量
func (epm *EnhancedPerformanceMonitor) getTotalMetricsCount() int {
	count := 0

	if epm.systemMetrics != nil {
		count += 4 // cpu, memory, goroutines, gc
	}

	if epm.networkMetrics != nil {
		count += len(epm.networkMetrics.latency) + 1 // latency per target + connections
	}

	if epm.applicationMetrics != nil {
		count += 4 // request_rate, response_time, error_rate, throughput
	}

	epm.customMetricsMutex.RLock()
	count += len(epm.customMetrics)
	epm.customMetricsMutex.RUnlock()

	return count
}

// getTotalDataPointsCount 获取总数据点数量
func (epm *EnhancedPerformanceMonitor) getTotalDataPointsCount() int64 {
	var total int64

	// 系统指标数据点
	if epm.systemMetrics != nil {
		total += epm.systemMetrics.cpuUsage.Count
		total += epm.systemMetrics.memoryUsage.Count
		total += epm.systemMetrics.goroutineCount.Count
		total += epm.systemMetrics.gcStats.Count
	}

	// 网络指标数据点
	if epm.networkMetrics != nil {
		for _, collector := range epm.networkMetrics.latency {
			total += collector.Count
		}
		total += epm.networkMetrics.connections.Count
	}

	// 应用指标数据点
	if epm.applicationMetrics != nil {
		total += epm.applicationMetrics.requestRate.Count
		total += epm.applicationMetrics.responseTime.Count
		total += epm.applicationMetrics.errorRate.Count
		total += epm.applicationMetrics.throughput.Count
	}

	// 自定义指标数据点
	epm.customMetricsMutex.RLock()
	for _, collector := range epm.customMetrics {
		total += collector.Count
	}
	epm.customMetricsMutex.RUnlock()

	return total
}

// cleanupExpiredData 清理过期数据
func (epm *EnhancedPerformanceMonitor) cleanupExpiredData(now time.Time) {
	cutoff := now.Add(-epm.config.DataRetention)

	// 清理系统指标过期数据
	if epm.systemMetrics != nil {
		epm.cleanupMetricData(epm.systemMetrics.cpuUsage, cutoff)
		epm.cleanupMetricData(epm.systemMetrics.memoryUsage, cutoff)
		epm.cleanupMetricData(epm.systemMetrics.diskUsage, cutoff)
		epm.cleanupMetricData(epm.systemMetrics.goroutineCount, cutoff)
		epm.cleanupMetricData(epm.systemMetrics.gcStats, cutoff)
	}

	// 清理网络指标过期数据
	if epm.networkMetrics != nil {
		for _, collector := range epm.networkMetrics.latency {
			epm.cleanupMetricData(collector, cutoff)
		}
		epm.cleanupMetricData(epm.networkMetrics.connections, cutoff)
	}

	// 清理应用指标过期数据
	if epm.applicationMetrics != nil {
		epm.cleanupMetricData(epm.applicationMetrics.requestRate, cutoff)
		epm.cleanupMetricData(epm.applicationMetrics.responseTime, cutoff)
		epm.cleanupMetricData(epm.applicationMetrics.errorRate, cutoff)
		epm.cleanupMetricData(epm.applicationMetrics.throughput, cutoff)
	}

	// 清理自定义指标过期数据
	epm.customMetricsMutex.RLock()
	for _, collector := range epm.customMetrics {
		epm.cleanupMetricData(collector, cutoff)
	}
	epm.customMetricsMutex.RUnlock()
}

// cleanupMetricData 清理指标数据
func (epm *EnhancedPerformanceMonitor) cleanupMetricData(collector *EnhancedMetricCollector, cutoff time.Time) {
	collector.mutex.Lock()
	defer collector.mutex.Unlock()

	// 找到第一个不过期的数据点
	keepIndex := 0
	for i, timestamp := range collector.Timestamps {
		if timestamp.After(cutoff) {
			keepIndex = i
			break
		}
	}

	// 如果有过期数据，则删除
	if keepIndex > 0 {
		collector.Values = collector.Values[keepIndex:]
		collector.Timestamps = collector.Timestamps[keepIndex:]
	}
}

// reportingLoop 报告生成循环
func (epm *EnhancedPerformanceMonitor) reportingLoop() {
	defer epm.wg.Done()

	ticker := time.NewTicker(epm.config.ReportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			epm.generateAndSaveReport()
		case <-epm.ctx.Done():
			return
		}
	}
}

// generateAndSaveReport 生成并保存报告
func (epm *EnhancedPerformanceMonitor) generateAndSaveReport() {
	if epm.reportGenerator == nil {
		return
	}

	report, err := epm.reportGenerator.GenerateReport(epm)
	if err != nil {
		log.Printf("生成性能报告失败: %v", err)
		return
	}

	// 保存报告到文件
	epm.saveReportToFiles(report)

	// 更新统计
	epm.statsMutex.Lock()
	epm.stats.ReportsGenerated++
	epm.stats.LastReportTime = time.Now()
	epm.statsMutex.Unlock()

	log.Printf("📊 性能报告已生成: %s (健康评分: %.1f)", report.ID, report.Summary.HealthScore)
}

// saveReportToFiles 保存报告到文件
func (epm *EnhancedPerformanceMonitor) saveReportToFiles(report *PerformanceReport) {
	if epm.config.ReportOutputDir == "" {
		return
	}

	// 确保输出目录存在
	err := os.MkdirAll(epm.config.ReportOutputDir, 0755)
	if err != nil {
		log.Printf("创建报告输出目录失败: %v", err)
		return
	}

	timestamp := report.GeneratedAt.Format("20060102_150405")

	// 保存JSON格式报告
	if contains(epm.config.ReportFormats, "json") {
		epm.saveJSONReport(report, timestamp)
	}

	// 保存CSV格式报告
	if contains(epm.config.ReportFormats, "csv") {
		epm.saveCSVReport(report, timestamp)
	}

	// 保存HTML格式报告
	if contains(epm.config.ReportFormats, "html") {
		epm.saveHTMLReport(report, timestamp)
	}
}

// saveJSONReport 保存JSON格式报告
func (epm *EnhancedPerformanceMonitor) saveJSONReport(report *PerformanceReport, timestamp string) {
	filename := filepath.Join(epm.config.ReportOutputDir, fmt.Sprintf("performance_report_%s.json", timestamp))

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Printf("序列化JSON报告失败: %v", err)
		return
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		log.Printf("保存JSON报告失败: %v", err)
		return
	}

	log.Printf("JSON报告已保存: %s", filename)
}

// saveCSVReport 保存CSV格式报告
func (epm *EnhancedPerformanceMonitor) saveCSVReport(report *PerformanceReport, timestamp string) {
	filename := filepath.Join(epm.config.ReportOutputDir, fmt.Sprintf("performance_report_%s.csv", timestamp))

	file, err := os.Create(filename)
	if err != nil {
		log.Printf("创建CSV报告文件失败: %v", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入标题行
	headers := []string{"指标名称", "当前值", "平均值", "最小值", "最大值", "P95", "健康状态", "趋势"}
	writer.Write(headers)

	// 写入系统指标
	if report.SystemMetrics != nil {
		epm.writeMetricToCSV(writer, "CPU使用率", report.SystemMetrics.CPUUsage)
		epm.writeMetricToCSV(writer, "内存使用率", report.SystemMetrics.MemoryUsage)
		epm.writeMetricToCSV(writer, "Goroutine数量", report.SystemMetrics.GoroutineCount)
	}

	// 写入网络指标
	if report.NetworkMetrics != nil {
		for target, metric := range report.NetworkMetrics.Latency {
			epm.writeMetricToCSV(writer, fmt.Sprintf("延迟_%s", target), metric)
		}
	}

	log.Printf("CSV报告已保存: %s", filename)
}

// writeMetricToCSV 写入指标到CSV
func (epm *EnhancedPerformanceMonitor) writeMetricToCSV(writer *csv.Writer, name string, metric *MetricReport) {
	if metric == nil {
		return
	}

	row := []string{
		name,
		fmt.Sprintf("%.2f", metric.Current),
		fmt.Sprintf("%.2f", metric.Average),
		fmt.Sprintf("%.2f", metric.Min),
		fmt.Sprintf("%.2f", metric.Max),
		fmt.Sprintf("%.2f", metric.P95),
		string(metric.HealthStatus),
		string(metric.Trend),
	}
	writer.Write(row)
}

// saveHTMLReport 保存HTML格式报告
func (epm *EnhancedPerformanceMonitor) saveHTMLReport(report *PerformanceReport, timestamp string) {
	filename := filepath.Join(epm.config.ReportOutputDir, fmt.Sprintf("performance_report_%s.html", timestamp))

	epm.reportGenerator.templatesMutex.RLock()
	tmpl, exists := epm.reportGenerator.templates["html"]
	epm.reportGenerator.templatesMutex.RUnlock()

	if !exists {
		log.Printf("HTML模板不存在")
		return
	}

	file, err := os.Create(filename)
	if err != nil {
		log.Printf("创建HTML报告文件失败: %v", err)
		return
	}
	defer file.Close()

	err = tmpl.Execute(file, report)
	if err != nil {
		log.Printf("生成HTML报告失败: %v", err)
		return
	}

	log.Printf("HTML报告已保存: %s", filename)
}

// alertingLoop 告警检查循环
func (epm *EnhancedPerformanceMonitor) alertingLoop() {
	defer epm.wg.Done()

	ticker := time.NewTicker(epm.config.MonitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			epm.checkAllAlerts()
		case <-epm.ctx.Done():
			return
		}
	}
}

// checkAllAlerts 检查所有告警
func (epm *EnhancedPerformanceMonitor) checkAllAlerts() {
	if epm.alertManager == nil {
		return
	}

	// 检查系统指标告警
	if epm.systemMetrics != nil {
		epm.checkMetricAlert("cpu_usage", epm.systemMetrics.cpuUsage)
		epm.checkMetricAlert("memory_usage", epm.systemMetrics.memoryUsage)
		epm.checkMetricAlert("goroutine_count", epm.systemMetrics.goroutineCount)
	}

	// 检查网络指标告警
	if epm.networkMetrics != nil {
		for target, collector := range epm.networkMetrics.latency {
			epm.checkMetricAlert(fmt.Sprintf("latency_%s", target), collector)
		}
	}

	// 检查应用指标告警
	if epm.applicationMetrics != nil {
		epm.checkMetricAlert("response_time", epm.applicationMetrics.responseTime)
		epm.checkMetricAlert("error_rate", epm.applicationMetrics.errorRate)
	}

	// 检查自定义指标告警
	epm.customMetricsMutex.RLock()
	for name, collector := range epm.customMetrics {
		epm.checkMetricAlert(name, collector)
	}
	epm.customMetricsMutex.RUnlock()
}

// checkMetricAlert 检查指标告警
func (epm *EnhancedPerformanceMonitor) checkMetricAlert(metricName string, collector *EnhancedMetricCollector) {
	collector.mutex.RLock()
	if len(collector.Values) == 0 {
		collector.mutex.RUnlock()
		return
	}
	currentValue := collector.Values[len(collector.Values)-1]
	collector.mutex.RUnlock()

	epm.alertManager.CheckMetric(metricName, currentValue)
}

// generateFinalReport 生成最终报告
func (epm *EnhancedPerformanceMonitor) generateFinalReport() {
	if epm.reportGenerator == nil {
		return
	}

	report, err := epm.reportGenerator.GenerateReport(epm)
	if err != nil {
		log.Printf("生成最终性能报告失败: %v", err)
		return
	}

	// 保存最终报告
	epm.saveReportToFiles(report)

	log.Printf("📊 最终性能报告已生成: 健康评分 %.1f/100", report.Summary.HealthScore)
}

// IsRunning 检查是否正在运行
func (epm *EnhancedPerformanceMonitor) IsRunning() bool {
	epm.runningMutex.RLock()
	defer epm.runningMutex.RUnlock()

	return epm.running
}

// GetStats 获取监控统计信息
func (epm *EnhancedPerformanceMonitor) GetStats() *MonitoringStats {
	epm.statsMutex.RLock()
	defer epm.statsMutex.RUnlock()

	// 返回副本
	stats := *epm.stats
	return &stats
}

// RecordCustomMetric 记录自定义指标
func (epm *EnhancedPerformanceMonitor) RecordCustomMetric(name string, value float64) {
	epm.customMetricsMutex.Lock()
	defer epm.customMetricsMutex.Unlock()

	collector, exists := epm.customMetrics[name]
	if !exists {
		collector = &EnhancedMetricCollector{
			Name:        name,
			Unit:        "count",
			Description: fmt.Sprintf("自定义指标: %s", name),
			MetricType:  MetricTypeGauge,
			Min:         math.MaxFloat64,
			Labels:      make(map[string]string),
		}
		epm.customMetrics[name] = collector
	}

	collector.RecordValue(value, time.Now())
}

// contains 辅助函数
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
