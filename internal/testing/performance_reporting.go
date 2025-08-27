package testing

import (
	"fmt"
	"html/template"
	"log"
	"strings"
	"sync"
	"time"
)

// PerformanceReportGenerator 性能报告生成器
type PerformanceReportGenerator struct {
	config         *PerformanceMonitorConfig
	templates      map[string]*template.Template
	templatesMutex sync.RWMutex
	reportHistory  []*PerformanceReport
	historyMutex   sync.RWMutex
}

// PerformanceReport 性能报告
type PerformanceReport struct {
	ID              string                    `json:"id"`
	GeneratedAt     time.Time                 `json:"generated_at"`
	Period          ReportPeriod              `json:"period"`
	Summary         *ReportSummary            `json:"summary"`
	SystemMetrics   *SystemMetricsReport      `json:"system_metrics"`
	NetworkMetrics  *NetworkMetricsReport     `json:"network_metrics"`
	AppMetrics      *ApplicationMetricsReport `json:"app_metrics"`
	CustomMetrics   map[string]*MetricReport  `json:"custom_metrics"`
	Alerts          []*AlertEvent             `json:"alerts"`
	Recommendations []string                  `json:"recommendations"`
}

// ReportPeriod 报告周期
type ReportPeriod struct {
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
}

// ReportSummary 报告摘要
type ReportSummary struct {
	TotalMetrics     int      `json:"total_metrics"`
	TotalDataPoints  int64    `json:"total_data_points"`
	HealthScore      float64  `json:"health_score"`
	PerformanceGrade string   `json:"performance_grade"`
	KeyFindings      []string `json:"key_findings"`
}

// SystemMetricsReport 系统指标报告
type SystemMetricsReport struct {
	CPUUsage       *MetricReport `json:"cpu_usage"`
	MemoryUsage    *MetricReport `json:"memory_usage"`
	DiskUsage      *MetricReport `json:"disk_usage"`
	GoroutineCount *MetricReport `json:"goroutine_count"`
	GCStats        *MetricReport `json:"gc_stats"`
}

// NetworkMetricsReport 网络指标报告
type NetworkMetricsReport struct {
	Latency     map[string]*MetricReport `json:"latency"`
	Bandwidth   map[string]*MetricReport `json:"bandwidth"`
	PacketLoss  map[string]*MetricReport `json:"packet_loss"`
	Connections *MetricReport            `json:"connections"`
}

// ApplicationMetricsReport 应用指标报告
type ApplicationMetricsReport struct {
	RequestRate      *MetricReport `json:"request_rate"`
	ResponseTime     *MetricReport `json:"response_time"`
	ErrorRate        *MetricReport `json:"error_rate"`
	Throughput       *MetricReport `json:"throughput"`
	ActiveStrategies *MetricReport `json:"active_strategies"`
	TradingVolume    *MetricReport `json:"trading_volume"`
	ProfitLoss       *MetricReport `json:"profit_loss"`
}

// MetricReport 指标报告
type MetricReport struct {
	Name            string         `json:"name"`
	Unit            string         `json:"unit"`
	Description     string         `json:"description"`
	Current         float64        `json:"current"`
	Average         float64        `json:"average"`
	Min             float64        `json:"min"`
	Max             float64        `json:"max"`
	StdDev          float64        `json:"std_dev"`
	P50             float64        `json:"p50"`
	P90             float64        `json:"p90"`
	P95             float64        `json:"p95"`
	P99             float64        `json:"p99"`
	Trend           TrendDirection `json:"trend"`
	TrendStrength   float64        `json:"trend_strength"`
	DataPoints      int64          `json:"data_points"`
	HealthStatus    HealthStatus   `json:"health_status"`
	Recommendations []string       `json:"recommendations"`
}

// HealthStatus 健康状态
type HealthStatus string

const (
	HealthStatusGood     HealthStatus = "good"
	HealthStatusWarning  HealthStatus = "warning"
	HealthStatusCritical HealthStatus = "critical"
	HealthStatusUnknown  HealthStatus = "unknown"
)

// AlertManager 告警管理器
type AlertManager struct {
	thresholds      map[string]AlertThreshold
	thresholdsMutex sync.RWMutex
	alertHistory    []*AlertEvent
	historyMutex    sync.RWMutex
	handlers        []AlertHandler
	handlersMutex   sync.RWMutex
}

// AlertEvent 告警事件
type AlertEvent struct {
	ID         string     `json:"id"`
	MetricName string     `json:"metric_name"`
	Level      AlertLevel `json:"level"`
	Value      float64    `json:"value"`
	Threshold  float64    `json:"threshold"`
	Message    string     `json:"message"`
	Timestamp  time.Time  `json:"timestamp"`
	Resolved   bool       `json:"resolved"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

// AlertLevel 告警级别
type AlertLevel string

const (
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelCritical AlertLevel = "critical"
)

// AlertHandler 告警处理器接口
type AlertHandler interface {
	HandleAlert(event *AlertEvent) error
}

// NewPerformanceReportGenerator 创建性能报告生成器
func NewPerformanceReportGenerator(config *PerformanceMonitorConfig) *PerformanceReportGenerator {
	generator := &PerformanceReportGenerator{
		config:        config,
		templates:     make(map[string]*template.Template),
		reportHistory: make([]*PerformanceReport, 0),
	}

	// 初始化模板
	generator.initializeTemplates()

	return generator
}

// initializeTemplates 初始化报告模板
func (prg *PerformanceReportGenerator) initializeTemplates() {
	// HTML报告模板
	htmlTemplate := `
<!DOCTYPE html>
<html>
<head>
    <title>性能监控报告</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 20px; border-radius: 5px; }
        .metric { margin: 10px 0; padding: 10px; border: 1px solid #ddd; border-radius: 3px; }
        .good { background-color: #d4edda; }
        .warning { background-color: #fff3cd; }
        .critical { background-color: #f8d7da; }
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <div class="header">
        <h1>性能监控报告</h1>
        <p>生成时间: {{.GeneratedAt.Format "2006-01-02 15:04:05"}}</p>
        <p>报告周期: {{.Period.StartTime.Format "2006-01-02 15:04:05"}} - {{.Period.EndTime.Format "2006-01-02 15:04:05"}}</p>
        <p>健康评分: {{.Summary.HealthScore}}/100 ({{.Summary.PerformanceGrade}})</p>
    </div>
    
    <h2>系统指标</h2>
    {{if .SystemMetrics}}
    <div class="metric {{.SystemMetrics.CPUUsage.HealthStatus}}">
        <h3>CPU使用率</h3>
        <p>当前: {{printf "%.2f" .SystemMetrics.CPUUsage.Current}}%</p>
        <p>平均: {{printf "%.2f" .SystemMetrics.CPUUsage.Average}}%</p>
        <p>最大: {{printf "%.2f" .SystemMetrics.CPUUsage.Max}}%</p>
    </div>
    {{end}}
    
    <h2>告警事件</h2>
    {{if .Alerts}}
    <table>
        <tr><th>时间</th><th>指标</th><th>级别</th><th>消息</th></tr>
        {{range .Alerts}}
        <tr>
            <td>{{.Timestamp.Format "15:04:05"}}</td>
            <td>{{.MetricName}}</td>
            <td>{{.Level}}</td>
            <td>{{.Message}}</td>
        </tr>
        {{end}}
    </table>
    {{else}}
    <p>无告警事件</p>
    {{end}}
</body>
</html>
`

	tmpl, err := template.New("html_report").Parse(htmlTemplate)
	if err != nil {
		log.Printf("Failed to parse HTML template: %v", err)
	} else {
		prg.templates["html"] = tmpl
	}
}

// GenerateReport 生成性能报告
func (prg *PerformanceReportGenerator) GenerateReport(monitor *EnhancedPerformanceMonitor) (*PerformanceReport, error) {
	now := time.Now()

	report := &PerformanceReport{
		ID:          fmt.Sprintf("report_%d", now.Unix()),
		GeneratedAt: now,
		Period: ReportPeriod{
			StartTime: now.Add(-prg.config.ReportInterval),
			EndTime:   now,
			Duration:  prg.config.ReportInterval,
		},
		CustomMetrics:   make(map[string]*MetricReport),
		Alerts:          make([]*AlertEvent, 0),
		Recommendations: make([]string, 0),
	}

	// 生成摘要
	report.Summary = prg.generateSummary(monitor)

	// 生成系统指标报告
	if monitor.systemMetrics != nil {
		report.SystemMetrics = prg.generateSystemMetricsReport(monitor.systemMetrics)
	}

	// 生成网络指标报告
	if monitor.networkMetrics != nil {
		report.NetworkMetrics = prg.generateNetworkMetricsReport(monitor.networkMetrics)
	}

	// 生成应用指标报告
	if monitor.applicationMetrics != nil {
		report.AppMetrics = prg.generateApplicationMetricsReport(monitor.applicationMetrics)
	}

	// 生成自定义指标报告
	monitor.customMetricsMutex.RLock()
	for name, collector := range monitor.customMetrics {
		report.CustomMetrics[name] = prg.generateMetricReport(collector)
	}
	monitor.customMetricsMutex.RUnlock()

	// 生成建议
	report.Recommendations = prg.generateRecommendations(report)

	// 保存到历史记录
	prg.historyMutex.Lock()
	prg.reportHistory = append(prg.reportHistory, report)
	// 限制历史记录数量
	if len(prg.reportHistory) > 100 {
		prg.reportHistory = prg.reportHistory[1:]
	}
	prg.historyMutex.Unlock()

	return report, nil
}

// generateSummary 生成报告摘要
func (prg *PerformanceReportGenerator) generateSummary(monitor *EnhancedPerformanceMonitor) *ReportSummary {
	stats := monitor.stats

	// 计算健康评分
	healthScore := prg.calculateHealthScore(monitor)

	// 确定性能等级
	var grade string
	switch {
	case healthScore >= 90:
		grade = "优秀"
	case healthScore >= 80:
		grade = "良好"
	case healthScore >= 70:
		grade = "一般"
	case healthScore >= 60:
		grade = "较差"
	default:
		grade = "差"
	}

	return &ReportSummary{
		TotalMetrics:     stats.TotalMetrics,
		TotalDataPoints:  stats.TotalDataPoints,
		HealthScore:      healthScore,
		PerformanceGrade: grade,
		KeyFindings:      []string{"系统运行正常", "无严重性能问题"},
	}
}

// calculateHealthScore 计算健康评分
func (prg *PerformanceReportGenerator) calculateHealthScore(monitor *EnhancedPerformanceMonitor) float64 {
	score := 100.0

	// 基于系统指标调整评分
	if monitor.systemMetrics != nil {
		if monitor.systemMetrics.cpuUsage.Mean > 80 {
			score -= 20
		} else if monitor.systemMetrics.cpuUsage.Mean > 60 {
			score -= 10
		}

		if monitor.systemMetrics.memoryUsage.Mean > 80 {
			score -= 20
		} else if monitor.systemMetrics.memoryUsage.Mean > 60 {
			score -= 10
		}
	}

	// 基于告警数量调整评分
	if monitor.alertManager != nil {
		monitor.alertManager.historyMutex.RLock()
		recentAlerts := 0
		cutoff := time.Now().Add(-time.Hour)
		for _, alert := range monitor.alertManager.alertHistory {
			if alert.Timestamp.After(cutoff) {
				recentAlerts++
			}
		}
		monitor.alertManager.historyMutex.RUnlock()

		score -= float64(recentAlerts) * 5 // 每个告警扣5分
	}

	if score < 0 {
		score = 0
	}

	return score
}

// generateSystemMetricsReport 生成系统指标报告
func (prg *PerformanceReportGenerator) generateSystemMetricsReport(systemMetrics *SystemMetricsCollector) *SystemMetricsReport {
	return &SystemMetricsReport{
		CPUUsage:       prg.generateMetricReport(systemMetrics.cpuUsage),
		MemoryUsage:    prg.generateMetricReport(systemMetrics.memoryUsage),
		DiskUsage:      prg.generateMetricReport(systemMetrics.diskUsage),
		GoroutineCount: prg.generateMetricReport(systemMetrics.goroutineCount),
		GCStats:        prg.generateMetricReport(systemMetrics.gcStats),
	}
}

// generateNetworkMetricsReport 生成网络指标报告
func (prg *PerformanceReportGenerator) generateNetworkMetricsReport(networkMetrics *NetworkMetricsCollector) *NetworkMetricsReport {
	report := &NetworkMetricsReport{
		Latency:     make(map[string]*MetricReport),
		Bandwidth:   make(map[string]*MetricReport),
		PacketLoss:  make(map[string]*MetricReport),
		Connections: prg.generateMetricReport(networkMetrics.connections),
	}

	for target, collector := range networkMetrics.latency {
		report.Latency[target] = prg.generateMetricReport(collector)
	}

	return report
}

// generateApplicationMetricsReport 生成应用指标报告
func (prg *PerformanceReportGenerator) generateApplicationMetricsReport(appMetrics *ApplicationMetricsCollector) *ApplicationMetricsReport {
	return &ApplicationMetricsReport{
		RequestRate:      prg.generateMetricReport(appMetrics.requestRate),
		ResponseTime:     prg.generateMetricReport(appMetrics.responseTime),
		ErrorRate:        prg.generateMetricReport(appMetrics.errorRate),
		Throughput:       prg.generateMetricReport(appMetrics.throughput),
		ActiveStrategies: prg.generateMetricReport(appMetrics.activeStrategies),
		TradingVolume:    prg.generateMetricReport(appMetrics.tradingVolume),
		ProfitLoss:       prg.generateMetricReport(appMetrics.profitLoss),
	}
}

// generateMetricReport 生成指标报告
func (prg *PerformanceReportGenerator) generateMetricReport(collector *EnhancedMetricCollector) *MetricReport {
	collector.mutex.RLock()
	defer collector.mutex.RUnlock()

	// 检查是否有数据
	if len(collector.Values) == 0 {
		return &MetricReport{
			Name:            collector.Name,
			Unit:            collector.Unit,
			Description:     collector.Description,
			Current:         0,
			Average:         0,
			Min:             0,
			Max:             0,
			StdDev:          0,
			P50:             0,
			P90:             0,
			P95:             0,
			P99:             0,
			Trend:           TrendUnknown,
			TrendStrength:   0,
			DataPoints:      0,
			HealthStatus:    HealthStatusUnknown,
			Recommendations: []string{"暂无数据"},
		}
	}

	// 确定健康状态
	healthStatus := prg.determineHealthStatus(collector)

	// 生成建议
	recommendations := prg.generateMetricRecommendations(collector)

	return &MetricReport{
		Name:            collector.Name,
		Unit:            collector.Unit,
		Description:     collector.Description,
		Current:         collector.Values[len(collector.Values)-1],
		Average:         collector.Mean,
		Min:             collector.Min,
		Max:             collector.Max,
		StdDev:          collector.StdDev,
		P50:             collector.P50,
		P90:             collector.P90,
		P95:             collector.P95,
		P99:             collector.P99,
		Trend:           collector.Trend,
		TrendStrength:   collector.TrendStrength,
		DataPoints:      collector.Count,
		HealthStatus:    healthStatus,
		Recommendations: recommendations,
	}
}

// determineHealthStatus 确定健康状态
func (prg *PerformanceReportGenerator) determineHealthStatus(collector *EnhancedMetricCollector) HealthStatus {
	if len(collector.Values) == 0 {
		return HealthStatusUnknown
	}

	current := collector.Values[len(collector.Values)-1]

	// 基于指标名称和值确定健康状态
	switch {
	case strings.Contains(collector.Name, "cpu_usage") || strings.Contains(collector.Name, "memory_usage"):
		if current > 90 {
			return HealthStatusCritical
		} else if current > 80 {
			return HealthStatusWarning
		}
		return HealthStatusGood

	case strings.Contains(collector.Name, "response_time"):
		if current > 5000 { // 5秒
			return HealthStatusCritical
		} else if current > 1000 { // 1秒
			return HealthStatusWarning
		}
		return HealthStatusGood

	case strings.Contains(collector.Name, "error_rate"):
		if current > 10 { // 10%
			return HealthStatusCritical
		} else if current > 5 { // 5%
			return HealthStatusWarning
		}
		return HealthStatusGood

	default:
		return HealthStatusGood
	}
}

// generateMetricRecommendations 生成指标建议
func (prg *PerformanceReportGenerator) generateMetricRecommendations(collector *EnhancedMetricCollector) []string {
	recommendations := make([]string, 0)

	if len(collector.Values) == 0 {
		return recommendations
	}

	current := collector.Values[len(collector.Values)-1]

	switch {
	case strings.Contains(collector.Name, "cpu_usage"):
		if current > 90 {
			recommendations = append(recommendations, "CPU使用率过高，建议优化算法或增加计算资源")
		} else if current > 80 {
			recommendations = append(recommendations, "CPU使用率较高，建议监控负载变化")
		}

	case strings.Contains(collector.Name, "memory_usage"):
		if current > 90 {
			recommendations = append(recommendations, "内存使用率过高，建议检查内存泄漏或增加内存")
		} else if current > 80 {
			recommendations = append(recommendations, "内存使用率较高，建议优化内存使用")
		}

	case strings.Contains(collector.Name, "response_time"):
		if current > 5000 {
			recommendations = append(recommendations, "响应时间过长，建议优化查询或增加缓存")
		} else if current > 1000 {
			recommendations = append(recommendations, "响应时间较长，建议关注性能优化")
		}
	}

	// 基于趋势的建议
	if collector.Trend == TrendUp && collector.TrendStrength > 0.1 {
		recommendations = append(recommendations, "指标呈上升趋势，建议持续监控")
	}

	return recommendations
}

// generateRecommendations 生成总体建议
func (prg *PerformanceReportGenerator) generateRecommendations(report *PerformanceReport) []string {
	recommendations := make([]string, 0)

	// 基于健康评分的建议
	if report.Summary.HealthScore < 70 {
		recommendations = append(recommendations, "系统健康评分较低，建议进行全面性能优化")
	}

	// 基于系统指标的建议
	if report.SystemMetrics != nil {
		if report.SystemMetrics.CPUUsage.HealthStatus == HealthStatusCritical {
			recommendations = append(recommendations, "CPU使用率严重过高，需要立即处理")
		}
		if report.SystemMetrics.MemoryUsage.HealthStatus == HealthStatusCritical {
			recommendations = append(recommendations, "内存使用率严重过高，需要立即处理")
		}
	}

	// 基于告警数量的建议
	if len(report.Alerts) > 10 {
		recommendations = append(recommendations, "告警数量较多，建议检查系统配置和阈值设置")
	}

	return recommendations
}

// NewAlertManager 创建告警管理器
func NewAlertManager(thresholds map[string]AlertThreshold) *AlertManager {
	return &AlertManager{
		thresholds:   thresholds,
		alertHistory: make([]*AlertEvent, 0),
		handlers:     make([]AlertHandler, 0),
	}
}

// AddHandler 添加告警处理器
func (am *AlertManager) AddHandler(handler AlertHandler) {
	am.handlersMutex.Lock()
	defer am.handlersMutex.Unlock()

	am.handlers = append(am.handlers, handler)
}

// CheckMetric 检查指标是否触发告警
func (am *AlertManager) CheckMetric(metricName string, value float64) {
	am.thresholdsMutex.RLock()
	threshold, exists := am.thresholds[metricName]
	am.thresholdsMutex.RUnlock()

	if !exists || !threshold.Enabled {
		return
	}

	var alertLevel AlertLevel
	var thresholdValue float64
	var triggered bool

	if value >= threshold.CriticalLevel {
		alertLevel = AlertLevelCritical
		thresholdValue = threshold.CriticalLevel
		triggered = true
	} else if value >= threshold.WarningLevel {
		alertLevel = AlertLevelWarning
		thresholdValue = threshold.WarningLevel
		triggered = true
	}

	if triggered {
		alert := &AlertEvent{
			ID:         fmt.Sprintf("alert_%s_%d", metricName, time.Now().Unix()),
			MetricName: metricName,
			Level:      alertLevel,
			Value:      value,
			Threshold:  thresholdValue,
			Message:    fmt.Sprintf("%s 达到 %s 级别: %.2f (阈值: %.2f)", metricName, alertLevel, value, thresholdValue),
			Timestamp:  time.Now(),
			Resolved:   false,
		}

		am.triggerAlert(alert)
	}
}

// triggerAlert 触发告警
func (am *AlertManager) triggerAlert(alert *AlertEvent) {
	// 添加到历史记录
	am.historyMutex.Lock()
	am.alertHistory = append(am.alertHistory, alert)
	// 限制历史记录数量
	if len(am.alertHistory) > 1000 {
		am.alertHistory = am.alertHistory[1:]
	}
	am.historyMutex.Unlock()

	// 调用处理器
	am.handlersMutex.RLock()
	handlers := make([]AlertHandler, len(am.handlers))
	copy(handlers, am.handlers)
	am.handlersMutex.RUnlock()

	for _, handler := range handlers {
		go func(h AlertHandler) {
			if err := h.HandleAlert(alert); err != nil {
				log.Printf("告警处理失败: %v", err)
			}
		}(handler)
	}

	log.Printf("🚨 告警触发: %s", alert.Message)
}
