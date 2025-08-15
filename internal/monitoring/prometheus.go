package monitoring

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	httpRequestsInFlight *prometheus.GaugeVec
	activeConnections   prometheus.Gauge
	apiErrorsTotal      *prometheus.CounterVec
	strategyExecutions  *prometheus.CounterVec
	optimizationTasks   *prometheus.CounterVec
	marketDataUpdates   *prometheus.CounterVec
}

// NewMetrics creates new Prometheus metrics
func NewMetrics() *Metrics {
	m := &Metrics{
		httpRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		httpRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),
		httpRequestsInFlight: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "http_requests_in_flight",
				Help: "Current number of HTTP requests being processed",
			},
			[]string{"method", "endpoint"},
		),
		activeConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "websocket_connections_active",
				Help: "Number of active WebSocket connections",
			},
		),
		apiErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_errors_total",
				Help: "Total number of API errors",
			},
			[]string{"endpoint", "error_type"},
		),
		strategyExecutions: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "strategy_executions_total",
				Help: "Total number of strategy executions",
			},
			[]string{"strategy_id", "status"},
		),
		optimizationTasks: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "optimization_tasks_total",
				Help: "Total number of optimization tasks",
			},
			[]string{"method", "status"},
		),
		marketDataUpdates: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "market_data_updates_total",
				Help: "Total number of market data updates",
			},
			[]string{"symbol", "data_type"},
		),
	}

	// Register metrics
	prometheus.MustRegister(
		m.httpRequestsTotal,
		m.httpRequestDuration,
		m.httpRequestsInFlight,
		m.activeConnections,
		m.apiErrorsTotal,
		m.strategyExecutions,
		m.optimizationTasks,
		m.marketDataUpdates,
	)

	return m
}

// MetricsMiddleware creates a Prometheus metrics middleware
func (m *Metrics) MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		// Track in-flight requests
		m.httpRequestsInFlight.WithLabelValues(c.Request.Method, path).Inc()
		defer m.httpRequestsInFlight.WithLabelValues(c.Request.Method, path).Dec()

		// Process request
		c.Next()

		// Record metrics
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())

		m.httpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		m.httpRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)

		// Track errors
		if c.Writer.Status() >= 400 {
			errorType := "client_error"
			if c.Writer.Status() >= 500 {
				errorType = "server_error"
			}
			m.apiErrorsTotal.WithLabelValues(path, errorType).Inc()
		}
	}
}

// PrometheusHandler returns the Prometheus metrics handler
func PrometheusHandler() http.Handler {
	return promhttp.Handler()
}

// RecordStrategyExecution records a strategy execution
func (m *Metrics) RecordStrategyExecution(strategyID, status string) {
	m.strategyExecutions.WithLabelValues(strategyID, status).Inc()
}

// RecordOptimizationTask records an optimization task
func (m *Metrics) RecordOptimizationTask(method, status string) {
	m.optimizationTasks.WithLabelValues(method, status).Inc()
}

// RecordMarketDataUpdate records a market data update
func (m *Metrics) RecordMarketDataUpdate(symbol, dataType string) {
	m.marketDataUpdates.WithLabelValues(symbol, dataType).Inc()
}

// SetActiveConnections sets the number of active WebSocket connections
func (m *Metrics) SetActiveConnections(count float64) {
	m.activeConnections.Set(count)
}
