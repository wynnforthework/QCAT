package testing

import (
	"fmt"
	"math"
	"sort"
	"sync/atomic"
	"time"
)

// EnhancedMetricCollector 的 RecordValue 方法
func (emc *EnhancedMetricCollector) RecordValue(value float64, timestamp time.Time) {
	emc.mutex.Lock()
	defer emc.mutex.Unlock()

	// 添加数据点
	emc.Values = append(emc.Values, value)
	emc.Timestamps = append(emc.Timestamps, timestamp)
	emc.lastUpdate = timestamp

	// 更新计数
	atomic.AddInt64(&emc.Count, 1)

	// 更新统计信息
	emc.Sum += value
	emc.Mean = emc.Sum / float64(emc.Count)

	if value < emc.Min || emc.Min == 0 {
		emc.Min = value
	}
	if value > emc.Max {
		emc.Max = value
	}

	// 计算标准差
	if emc.Count > 1 {
		emc.calculateStdDev()
	}

	// 计算百分位数
	emc.calculatePercentiles()

	// 分析趋势
	emc.analyzeTrend()

	// 限制数据点数量
	maxPoints := 10000 // 可配置
	if len(emc.Values) > maxPoints {
		excess := len(emc.Values) - maxPoints
		emc.Values = emc.Values[excess:]
		emc.Timestamps = emc.Timestamps[excess:]
	}
}

// calculateStdDev 计算标准差
func (emc *EnhancedMetricCollector) calculateStdDev() {
	if emc.Count < 2 {
		return
	}

	var sumSquares float64
	for _, value := range emc.Values {
		diff := value - emc.Mean
		sumSquares += diff * diff
	}

	variance := sumSquares / float64(emc.Count-1)
	emc.StdDev = math.Sqrt(variance)
}

// calculatePercentiles 计算百分位数
func (emc *EnhancedMetricCollector) calculatePercentiles() {
	if len(emc.Values) == 0 {
		return
	}

	// 创建副本并排序
	values := make([]float64, len(emc.Values))
	copy(values, emc.Values)
	sort.Float64s(values)

	emc.P50 = percentile(values, 0.50)
	emc.P90 = percentile(values, 0.90)
	emc.P95 = percentile(values, 0.95)
	emc.P99 = percentile(values, 0.99)
}

// percentile 计算百分位数
func percentile(sortedValues []float64, p float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}

	index := p * float64(len(sortedValues)-1)
	lower := int(index)
	upper := lower + 1

	if upper >= len(sortedValues) {
		return sortedValues[len(sortedValues)-1]
	}

	weight := index - float64(lower)
	return sortedValues[lower]*(1-weight) + sortedValues[upper]*weight
}

// analyzeTrend 分析趋势
func (emc *EnhancedMetricCollector) analyzeTrend() {
	if len(emc.Values) < 10 {
		emc.Trend = TrendUnknown
		return
	}

	// 使用最近的数据点进行趋势分析
	recentCount := min(len(emc.Values), 50)
	recent := emc.Values[len(emc.Values)-recentCount:]

	// 简单的线性回归来确定趋势
	n := float64(len(recent))
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0

	for i, y := range recent {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	emc.TrendStrength = math.Abs(slope)

	if slope > 0.01 {
		emc.Trend = TrendUp
	} else if slope < -0.01 {
		emc.Trend = TrendDown
	} else {
		emc.Trend = TrendStable
	}
}

// NewSystemMetricsCollector 创建系统指标收集器
func NewSystemMetricsCollector() *SystemMetricsCollector {
	return &SystemMetricsCollector{
		cpuUsage: &EnhancedMetricCollector{
			Name:        "cpu_usage",
			Unit:        "percent",
			Description: "CPU使用率",
			MetricType:  MetricTypeGauge,
			Min:         math.MaxFloat64,
			Labels:      make(map[string]string),
		},
		memoryUsage: &EnhancedMetricCollector{
			Name:        "memory_usage",
			Unit:        "percent",
			Description: "内存使用率",
			MetricType:  MetricTypeGauge,
			Min:         math.MaxFloat64,
			Labels:      make(map[string]string),
		},
		diskUsage: &EnhancedMetricCollector{
			Name:        "disk_usage",
			Unit:        "percent",
			Description: "磁盘使用率",
			MetricType:  MetricTypeGauge,
			Min:         math.MaxFloat64,
			Labels:      make(map[string]string),
		},
		goroutineCount: &EnhancedMetricCollector{
			Name:        "goroutine_count",
			Unit:        "count",
			Description: "Goroutine数量",
			MetricType:  MetricTypeGauge,
			Min:         math.MaxFloat64,
			Labels:      make(map[string]string),
		},
		gcStats: &EnhancedMetricCollector{
			Name:        "gc_pause_total",
			Unit:        "milliseconds",
			Description: "GC暂停总时间",
			MetricType:  MetricTypeCounter,
			Min:         math.MaxFloat64,
			Labels:      make(map[string]string),
		},
		startTime: time.Now(),
	}
}

// NewNetworkMetricsCollector 创建网络指标收集器
func NewNetworkMetricsCollector(interfaces []string, pingTargets []string) *NetworkMetricsCollector {
	collector := &NetworkMetricsCollector{
		latency:    make(map[string]*EnhancedMetricCollector),
		bandwidth:  make(map[string]*EnhancedMetricCollector),
		packetLoss: make(map[string]*EnhancedMetricCollector),
		interfaces: make(map[string]*NetworkInterfaceStats),
		connections: &EnhancedMetricCollector{
			Name:        "network_connections",
			Unit:        "count",
			Description: "网络连接数",
			MetricType:  MetricTypeGauge,
			Min:         math.MaxFloat64,
			Labels:      make(map[string]string),
		},
	}

	// 为每个ping目标创建延迟收集器
	for _, target := range pingTargets {
		collector.latency[target] = &EnhancedMetricCollector{
			Name:        fmt.Sprintf("latency_%s", target),
			Unit:        "milliseconds",
			Description: fmt.Sprintf("到%s的网络延迟", target),
			MetricType:  MetricTypeGauge,
			Min:         math.MaxFloat64,
			Labels:      map[string]string{"target": target},
		}
	}

	// 为每个网络接口创建统计
	for _, iface := range interfaces {
		collector.interfaces[iface] = &NetworkInterfaceStats{
			Name:       iface,
			LastUpdate: time.Now(),
		}
	}

	return collector
}

// NewApplicationMetricsCollector 创建应用指标收集器
func NewApplicationMetricsCollector() *ApplicationMetricsCollector {
	return &ApplicationMetricsCollector{
		requestRate: &EnhancedMetricCollector{
			Name:        "request_rate",
			Unit:        "requests/second",
			Description: "请求速率",
			MetricType:  MetricTypeGauge,
			Min:         math.MaxFloat64,
			Labels:      make(map[string]string),
		},
		responseTime: &EnhancedMetricCollector{
			Name:        "response_time",
			Unit:        "milliseconds",
			Description: "响应时间",
			MetricType:  MetricTypeHistogram,
			Min:         math.MaxFloat64,
			Labels:      make(map[string]string),
		},
		errorRate: &EnhancedMetricCollector{
			Name:        "error_rate",
			Unit:        "percent",
			Description: "错误率",
			MetricType:  MetricTypeGauge,
			Min:         math.MaxFloat64,
			Labels:      make(map[string]string),
		},
		throughput: &EnhancedMetricCollector{
			Name:        "throughput",
			Unit:        "bytes/second",
			Description: "吞吐量",
			MetricType:  MetricTypeGauge,
			Min:         math.MaxFloat64,
			Labels:      make(map[string]string),
		},
		activeStrategies: &EnhancedMetricCollector{
			Name:        "active_strategies",
			Unit:        "count",
			Description: "活跃策略数量",
			MetricType:  MetricTypeGauge,
			Min:         math.MaxFloat64,
			Labels:      make(map[string]string),
		},
		tradingVolume: &EnhancedMetricCollector{
			Name:        "trading_volume",
			Unit:        "USDT",
			Description: "交易量",
			MetricType:  MetricTypeCounter,
			Min:         math.MaxFloat64,
			Labels:      make(map[string]string),
		},
		profitLoss: &EnhancedMetricCollector{
			Name:        "profit_loss",
			Unit:        "USDT",
			Description: "盈亏",
			MetricType:  MetricTypeGauge,
			Min:         math.MaxFloat64,
			Labels:      make(map[string]string),
		},
	}
}

// min 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
