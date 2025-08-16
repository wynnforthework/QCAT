package monitor

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MetricsCollector collects system metrics
type MetricsCollector struct {
	// 策略相关指标
	strategyPnL       *prometheus.GaugeVec
	strategyDrawdown  *prometheus.GaugeVec
	strategySharpe    *prometheus.GaugeVec
	strategyPositions *prometheus.GaugeVec
	strategyOrders    *prometheus.CounterVec

	// 市场数据指标
	marketDataLatency  *prometheus.HistogramVec
	marketDataGaps     *prometheus.CounterVec
	marketDataOutliers *prometheus.CounterVec

	// 系统性能指标
	systemCPUUsage    prometheus.Gauge
	systemMemoryUsage prometheus.Gauge
	systemDiskUsage   prometheus.Gauge
	systemGoroutines  prometheus.Gauge

	// 交易相关指标
	tradeVolume      *prometheus.CounterVec
	tradeCount       *prometheus.CounterVec
	tradeLatency     *prometheus.HistogramVec
	orderSuccessRate *prometheus.GaugeVec

	// 风控指标
	riskExposure   *prometheus.GaugeVec
	riskLimits     *prometheus.GaugeVec
	riskViolations *prometheus.CounterVec

	// 优化相关指标
	optimizationDuration    *prometheus.HistogramVec
	optimizationSuccess     *prometheus.CounterVec
	optimizationImprovement *prometheus.GaugeVec

	mu sync.RWMutex // 保护并发访问
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	mc := &MetricsCollector{
		// 策略指标
		strategyPnL: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "strategy_pnl",
			Help: "Strategy profit and loss",
		}, []string{"strategy_id", "symbol"}),

		strategyDrawdown: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "strategy_drawdown",
			Help: "Strategy maximum drawdown",
		}, []string{"strategy_id"}),

		strategySharpe: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "strategy_sharpe",
			Help: "Strategy Sharpe ratio",
		}, []string{"strategy_id"}),

		strategyPositions: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "strategy_positions",
			Help: "Strategy current positions",
		}, []string{"strategy_id", "symbol"}),

		strategyOrders: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "strategy_orders_total",
			Help: "Total number of strategy orders",
		}, []string{"strategy_id", "symbol", "side", "status"}),

		// 市场数据指标
		marketDataLatency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "market_data_latency_seconds",
			Help:    "Market data latency in seconds",
			Buckets: prometheus.DefBuckets,
		}, []string{"symbol", "data_type"}),

		marketDataGaps: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "market_data_gaps_total",
			Help: "Total number of market data gaps",
		}, []string{"symbol", "data_type"}),

		marketDataOutliers: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "market_data_outliers_total",
			Help: "Total number of market data outliers",
		}, []string{"symbol", "data_type"}),

		// 系统性能指标
		systemCPUUsage: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "system_cpu_usage_percent",
			Help: "System CPU usage percentage",
		}),

		systemMemoryUsage: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "system_memory_usage_percent",
			Help: "System memory usage percentage",
		}),

		systemDiskUsage: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "system_disk_usage_percent",
			Help: "System disk usage percentage",
		}),

		systemGoroutines: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "system_goroutines",
			Help: "Number of goroutines",
		}),

		// 交易指标
		tradeVolume: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "trade_volume_total",
			Help: "Total trade volume",
		}, []string{"symbol", "side"}),

		tradeCount: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "trade_count_total",
			Help: "Total number of trades",
		}, []string{"symbol", "side"}),

		tradeLatency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "trade_latency_seconds",
			Help:    "Trade execution latency in seconds",
			Buckets: prometheus.DefBuckets,
		}, []string{"symbol", "exchange"}),

		orderSuccessRate: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "order_success_rate",
			Help: "Order success rate",
		}, []string{"symbol", "exchange"}),

		// 风控指标
		riskExposure: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "risk_exposure",
			Help: "Current risk exposure",
		}, []string{"risk_type"}),

		riskLimits: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "risk_limits",
			Help: "Risk limits",
		}, []string{"risk_type"}),

		riskViolations: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "risk_violations_total",
			Help: "Total number of risk violations",
		}, []string{"risk_type"}),

		// 优化指标
		optimizationDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "optimization_duration_seconds",
			Help:    "Optimization duration in seconds",
			Buckets: prometheus.DefBuckets,
		}, []string{"strategy_id", "optimization_type"}),

		optimizationSuccess: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "optimization_success_total",
			Help: "Total number of successful optimizations",
		}, []string{"strategy_id", "optimization_type"}),

		optimizationImprovement: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "optimization_improvement",
			Help: "Optimization improvement percentage",
		}, []string{"strategy_id", "optimization_type"}),
	}

	return mc
}

// UpdateStrategyMetrics updates strategy metrics
func (mc *MetricsCollector) UpdateStrategyMetrics(strategyID, symbol string, pnl, drawdown, sharpe float64) {
	mc.strategyPnL.WithLabelValues(strategyID, symbol).Set(pnl)
	mc.strategyDrawdown.WithLabelValues(strategyID).Set(drawdown)
	mc.strategySharpe.WithLabelValues(strategyID).Set(sharpe)
}

// UpdateStrategyPosition updates strategy position metrics
func (mc *MetricsCollector) UpdateStrategyPosition(strategyID, symbol string, position float64) {
	mc.strategyPositions.WithLabelValues(strategyID, symbol).Set(position)
}

// RecordStrategyOrder records a strategy order
func (mc *MetricsCollector) RecordStrategyOrder(strategyID, symbol, side, status string) {
	mc.strategyOrders.WithLabelValues(strategyID, symbol, side, status).Inc()
}

// RecordMarketDataLatency records market data latency
func (mc *MetricsCollector) RecordMarketDataLatency(symbol, dataType string, latency time.Duration) {
	mc.marketDataLatency.WithLabelValues(symbol, dataType).Observe(latency.Seconds())
}

// RecordMarketDataGap records a market data gap
func (mc *MetricsCollector) RecordMarketDataGap(symbol, dataType string) {
	mc.marketDataGaps.WithLabelValues(symbol, dataType).Inc()
}

// RecordMarketDataOutlier records a market data outlier
func (mc *MetricsCollector) RecordMarketDataOutlier(symbol, dataType string) {
	mc.marketDataOutliers.WithLabelValues(symbol, dataType).Inc()
}

// UpdateSystemMetrics updates system metrics
func (mc *MetricsCollector) UpdateSystemMetrics(cpuUsage, memUsage, diskUsage float64, goroutines int) {
	mc.systemCPUUsage.Set(cpuUsage)
	mc.systemMemoryUsage.Set(memUsage)
	mc.systemDiskUsage.Set(diskUsage)
	mc.systemGoroutines.Set(float64(goroutines))
}

// RecordTrade records a trade
func (mc *MetricsCollector) RecordTrade(symbol, side string, volume float64, latency time.Duration) {
	mc.tradeVolume.WithLabelValues(symbol, side).Add(volume)
	mc.tradeCount.WithLabelValues(symbol, side).Inc()
	mc.tradeLatency.WithLabelValues(symbol, "binance").Observe(latency.Seconds())
}

// UpdateOrderSuccessRate updates order success rate
func (mc *MetricsCollector) UpdateOrderSuccessRate(symbol, exchange string, rate float64) {
	mc.orderSuccessRate.WithLabelValues(symbol, exchange).Set(rate)
}

// UpdateRiskMetrics updates risk metrics
func (mc *MetricsCollector) UpdateRiskMetrics(riskType string, exposure, limit float64) {
	mc.riskExposure.WithLabelValues(riskType).Set(exposure)
	mc.riskLimits.WithLabelValues(riskType).Set(limit)
}

// RecordRiskViolation records a risk violation
func (mc *MetricsCollector) RecordRiskViolation(riskType string) {
	mc.riskViolations.WithLabelValues(riskType).Inc()
}

// RecordOptimization records optimization metrics
func (mc *MetricsCollector) RecordOptimization(strategyID, optimizationType string, duration time.Duration, success bool, improvement float64) {
	mc.optimizationDuration.WithLabelValues(strategyID, optimizationType).Observe(duration.Seconds())

	if success {
		mc.optimizationSuccess.WithLabelValues(strategyID, optimizationType).Inc()
		mc.optimizationImprovement.WithLabelValues(strategyID, optimizationType).Set(improvement)
	}
}
