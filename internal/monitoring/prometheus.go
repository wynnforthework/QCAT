package monitoring

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"qcat/internal/database"
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
	
	// Database connection pool metrics
	dbPoolMaxOpen       prometheus.Gauge
	dbPoolOpen          prometheus.Gauge
	dbPoolInUse         prometheus.Gauge
	dbPoolIdle          prometheus.Gauge
	dbPoolWaitCount     prometheus.Counter
	dbPoolWaitDuration  prometheus.Histogram
	dbPoolMaxIdleClosed prometheus.Counter
	dbPoolMaxLifetimeClosed prometheus.Counter
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
		
		// Database connection pool metrics
		dbPoolMaxOpen: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "db_pool_max_open_connections",
			Help: "Maximum number of open connections to the database",
		}),
		dbPoolOpen: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "db_pool_open_connections",
			Help: "The number of established connections both in use and idle",
		}),
		dbPoolInUse: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "db_pool_in_use_connections",
			Help: "The number of connections currently in use",
		}),
		dbPoolIdle: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "db_pool_idle_connections",
			Help: "The number of idle connections",
		}),
		dbPoolWaitCount: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "db_pool_wait_count_total",
			Help: "The total number of connections waited for",
		}),
		dbPoolWaitDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "db_pool_wait_duration_seconds",
			Help:    "The total time blocked waiting for a new connection",
			Buckets: prometheus.DefBuckets,
		}),
		dbPoolMaxIdleClosed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "db_pool_max_idle_closed_total",
			Help: "The total number of connections closed due to SetMaxIdleConns",
		}),
		dbPoolMaxLifetimeClosed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "db_pool_max_lifetime_closed_total",
			Help: "The total number of connections closed due to SetConnMaxLifetime",
		}),
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
		m.dbPoolMaxOpen,
		m.dbPoolOpen,
		m.dbPoolInUse,
		m.dbPoolIdle,
		m.dbPoolWaitCount,
		m.dbPoolWaitDuration,
		m.dbPoolMaxIdleClosed,
		m.dbPoolMaxLifetimeClosed,
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

// UpdateDatabasePoolMetrics updates database connection pool metrics
func (m *Metrics) UpdateDatabasePoolMetrics(stats *database.PoolStats) {
	m.dbPoolMaxOpen.Set(float64(stats.MaxOpenConnections))
	m.dbPoolOpen.Set(float64(stats.OpenConnections))
	m.dbPoolInUse.Set(float64(stats.InUse))
	m.dbPoolIdle.Set(float64(stats.Idle))
	
	// For counters, we need to track the difference
	// This is a simplified approach - in production you might want to track the previous values
	if stats.WaitCount > 0 {
		m.dbPoolWaitCount.Add(float64(stats.WaitCount))
	}
	if stats.WaitDuration > 0 {
		m.dbPoolWaitDuration.Observe(stats.WaitDuration.Seconds())
	}
	if stats.MaxIdleClosed > 0 {
		m.dbPoolMaxIdleClosed.Add(float64(stats.MaxIdleClosed))
	}
	if stats.MaxLifetimeClosed > 0 {
		m.dbPoolMaxLifetimeClosed.Add(float64(stats.MaxLifetimeClosed))
	}
}
