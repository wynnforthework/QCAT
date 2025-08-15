package stability

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// HealthChecker manages service health monitoring
type HealthChecker struct {
	// Prometheus metrics
	healthStatus     prometheus.Gauge
	healthLatency    prometheus.Histogram
	healthFailures   prometheus.Counter
	healthRecoveries prometheus.Counter

	// Configuration
	config *HealthConfig

	// State
	checks map[string]*ServiceHealthCheck
	mu     sync.RWMutex

	// Channels
	alertCh chan *HealthAlert
	stopCh  chan struct{}
}

// HealthConfig represents health check configuration
type HealthConfig struct {
	// Check intervals
	CheckInterval time.Duration
	Timeout       time.Duration
	RetryCount    int
	RetryInterval time.Duration

	// Thresholds
	DegradedThreshold  float64
	UnhealthyThreshold float64

	// Alert settings
	AlertThreshold int
	AlertCooldown  time.Duration
}

// ServiceHealthCheck represents a health check
type ServiceHealthCheck struct {
	Name          string
	Description   string
	CheckFunc     HealthCheckFunc
	Status        HealthStatus
	LastCheck     time.Time
	LastError     string
	Latency       time.Duration
	FailCount     int
	RecoveryCount int
	Metadata      map[string]interface{}
}

// HealthCheckFunc represents a health check function
type HealthCheckFunc func(ctx context.Context) (*HealthResult, error)

// HealthResult represents the result of a health check
type HealthResult struct {
	Status   HealthStatus
	Latency  time.Duration
	Message  string
	Metadata map[string]interface{}
}

// HealthStatus represents health status
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// HealthAlert represents a health alert
type HealthAlert struct {
	Type      HealthAlertType
	Message   string
	CheckName string
	Status    HealthStatus
	Latency   time.Duration
	Timestamp time.Time
}

// HealthAlertType represents the type of health alert
type HealthAlertType string

const (
	AlertTypeHealthDegraded  HealthAlertType = "health_degraded"
	AlertTypeHealthUnhealthy HealthAlertType = "health_unhealthy"
	AlertTypeHealthRecovered HealthAlertType = "health_recovered"
	AlertTypeCheckFailed     HealthAlertType = "check_failed"
	AlertTypeCheckTimeout    HealthAlertType = "check_timeout"
)

// NewHealthChecker creates a new health checker
func NewHealthChecker(config *HealthConfig) *HealthChecker {
	if config == nil {
		config = &HealthConfig{
			CheckInterval:      30 * time.Second,
			Timeout:            10 * time.Second,
			RetryCount:         3,
			RetryInterval:      5 * time.Second,
			DegradedThreshold:  0.8,
			UnhealthyThreshold: 0.5,
			AlertThreshold:     3,
			AlertCooldown:      5 * time.Minute,
		}
	}

	hc := &HealthChecker{
		config:  config,
		checks:  make(map[string]*ServiceHealthCheck),
		alertCh: make(chan *HealthAlert, 100),
		stopCh:  make(chan struct{}),
	}

	// Initialize Prometheus metrics
	hc.initializeMetrics()

	return hc
}

// initializeMetrics initializes Prometheus metrics
func (hc *HealthChecker) initializeMetrics() {
	hc.healthStatus = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "health_status",
		Help: "Overall health status (0=unhealthy, 1=degraded, 2=healthy)",
	})

	hc.healthLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "health_check_latency_seconds",
		Help:    "Health check latency in seconds",
		Buckets: prometheus.ExponentialBuckets(0.1, 2, 10),
	})

	hc.healthFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "health_check_failures_total",
		Help: "Total number of health check failures",
	})

	hc.healthRecoveries = promauto.NewCounter(prometheus.CounterOpts{
		Name: "health_check_recoveries_total",
		Help: "Total number of health check recoveries",
	})
}

// RegisterCheck registers a new health check
func (hc *HealthChecker) RegisterCheck(name, description string, checkFunc HealthCheckFunc) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.checks[name] = &ServiceHealthCheck{
		Name:        name,
		Description: description,
		CheckFunc:   checkFunc,
		Status:      HealthStatusUnknown,
		LastCheck:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	log.Printf("Registered health check: %s", name)
}

// Start starts the health checker
func (hc *HealthChecker) Start() {
	log.Println("Starting health checker...")

	// Start health check routine
	go hc.runHealthChecks()

	// Start alert processing
	go hc.processAlerts()
}

// Stop stops the health checker
func (hc *HealthChecker) Stop() {
	log.Println("Stopping health checker...")
	close(hc.stopCh)
	close(hc.alertCh)
}

// runHealthChecks runs periodic health checks
func (hc *HealthChecker) runHealthChecks() {
	ticker := time.NewTicker(hc.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-hc.stopCh:
			return
		case <-ticker.C:
			hc.performHealthChecks()
		}
	}
}

// performHealthChecks performs all registered health checks
func (hc *HealthChecker) performHealthChecks() {
	hc.mu.RLock()
	checks := make([]*ServiceHealthCheck, 0, len(hc.checks))
	for _, check := range hc.checks {
		checks = append(checks, check)
	}
	hc.mu.RUnlock()

	var wg sync.WaitGroup
	for _, check := range checks {
		wg.Add(1)
		go func(c *ServiceHealthCheck) {
			defer wg.Done()
			hc.performCheck(c)
		}(check)
	}

	wg.Wait()

	// Update overall health status
	hc.updateOverallHealth()
}

// performCheck performs a single health check
func (hc *HealthChecker) performCheck(check *ServiceHealthCheck) {
	ctx, cancel := context.WithTimeout(context.Background(), hc.config.Timeout)
	defer cancel()

	startTime := time.Now()
	result, err := check.CheckFunc(ctx)
	latency := time.Since(startTime)

	hc.mu.Lock()
	defer hc.mu.Unlock()

	// Update check status
	oldStatus := check.Status
	check.LastCheck = time.Now()
	check.Latency = latency

	if err != nil {
		check.Status = HealthStatusUnhealthy
		check.LastError = err.Error()
		check.FailCount++
		hc.healthFailures.Inc()

		hc.sendAlert(&HealthAlert{
			Type:      AlertTypeCheckFailed,
			Message:   fmt.Sprintf("Health check failed: %v", err),
			CheckName: check.Name,
			Status:    check.Status,
			Latency:   latency,
			Timestamp: time.Now(),
		})
	} else {
		check.Status = result.Status
		check.LastError = ""
		check.Metadata = result.Metadata

		// Check if status improved
		if oldStatus != HealthStatusHealthy && result.Status == HealthStatusHealthy {
			check.RecoveryCount++
			hc.healthRecoveries.Inc()

			hc.sendAlert(&HealthAlert{
				Type:      AlertTypeHealthRecovered,
				Message:   fmt.Sprintf("Health check recovered: %s", result.Message),
				CheckName: check.Name,
				Status:    check.Status,
				Latency:   latency,
				Timestamp: time.Now(),
			})
		}

		// Check for status degradation
		if oldStatus == HealthStatusHealthy && result.Status != HealthStatusHealthy {
			hc.sendAlert(&HealthAlert{
				Type:      AlertTypeHealthDegraded,
				Message:   fmt.Sprintf("Health check degraded: %s", result.Message),
				CheckName: check.Name,
				Status:    check.Status,
				Latency:   latency,
				Timestamp: time.Now(),
			})
		}
	}

	// Update metrics
	hc.healthLatency.Observe(latency.Seconds())
}

// updateOverallHealth updates the overall health status
func (hc *HealthChecker) updateOverallHealth() {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	healthyCount := 0
	totalCount := len(hc.checks)

	for _, check := range hc.checks {
		if check.Status == HealthStatusHealthy {
			healthyCount++
		}
	}

	healthRatio := float64(healthyCount) / float64(totalCount)
	var overallStatus float64

	if healthRatio >= hc.config.DegradedThreshold {
		overallStatus = 2.0 // Healthy
	} else if healthRatio >= hc.config.UnhealthyThreshold {
		overallStatus = 1.0 // Degraded
	} else {
		overallStatus = 0.0 // Unhealthy
	}

	hc.healthStatus.Set(overallStatus)
}

// sendAlert sends a health alert
func (hc *HealthChecker) sendAlert(alert *HealthAlert) {
	select {
	case hc.alertCh <- alert:
		log.Printf("Health alert: %s - %s", alert.Type, alert.Message)
	default:
		log.Printf("Alert channel is full, dropped health alert: %s", alert.Message)
	}
}

// processAlerts processes health alerts
func (hc *HealthChecker) processAlerts() {
	for {
		select {
		case <-hc.stopCh:
			return
		case alert := <-hc.alertCh:
			hc.handleAlert(alert)
		}
	}
}

// handleAlert handles a health alert
func (hc *HealthChecker) handleAlert(alert *HealthAlert) {
	switch alert.Type {
	case AlertTypeHealthUnhealthy:
		log.Printf("CRITICAL: Service %s is unhealthy", alert.CheckName)
		// TODO: Send critical alert (email, SMS, etc.)
	case AlertTypeHealthDegraded:
		log.Printf("WARNING: Service %s is degraded", alert.CheckName)
		// TODO: Send warning alert
	case AlertTypeHealthRecovered:
		log.Printf("INFO: Service %s has recovered", alert.CheckName)
		// TODO: Send recovery notification
	case AlertTypeCheckFailed:
		log.Printf("ERROR: Health check %s failed", alert.CheckName)
		// TODO: Send error alert
	case AlertTypeCheckTimeout:
		log.Printf("WARNING: Health check %s timed out", alert.CheckName)
		// TODO: Send timeout alert
	}
}

// GetHealthStatus gets the health status of a specific check
func (hc *HealthChecker) GetHealthStatus(name string) *ServiceHealthCheck {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	check, exists := hc.checks[name]
	if !exists {
		return nil
	}

	// Create a copy to avoid race conditions
	checkCopy := &ServiceHealthCheck{
		Name:          check.Name,
		Description:   check.Description,
		Status:        check.Status,
		LastCheck:     check.LastCheck,
		LastError:     check.LastError,
		Latency:       check.Latency,
		FailCount:     check.FailCount,
		RecoveryCount: check.RecoveryCount,
		Metadata:      make(map[string]interface{}),
	}

	for k, v := range check.Metadata {
		checkCopy.Metadata[k] = v
	}

	return checkCopy
}

// GetAllHealthStatus gets the health status of all checks
func (hc *HealthChecker) GetAllHealthStatus() map[string]*ServiceHealthCheck {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	status := make(map[string]*ServiceHealthCheck)
	for name, check := range hc.checks {
		status[name] = hc.GetHealthStatus(name)
	}

	return status
}

// GetOverallHealth gets the overall system health
func (hc *HealthChecker) GetOverallHealth() map[string]interface{} {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	healthyCount := 0
	degradedCount := 0
	unhealthyCount := 0
	unknownCount := 0

	for _, check := range hc.checks {
		switch check.Status {
		case HealthStatusHealthy:
			healthyCount++
		case HealthStatusDegraded:
			degradedCount++
		case HealthStatusUnhealthy:
			unhealthyCount++
		case HealthStatusUnknown:
			unknownCount++
		}
	}

	totalCount := len(hc.checks)
	healthRatio := float64(healthyCount) / float64(totalCount)

	var overallStatus string
	if healthRatio >= hc.config.DegradedThreshold {
		overallStatus = "healthy"
	} else if healthRatio >= hc.config.UnhealthyThreshold {
		overallStatus = "degraded"
	} else {
		overallStatus = "unhealthy"
	}

	return map[string]interface{}{
		"status":       overallStatus,
		"health_ratio": healthRatio,
		"total_checks": totalCount,
		"healthy":      healthyCount,
		"degraded":     degradedCount,
		"unhealthy":    unhealthyCount,
		"unknown":      unknownCount,
		"last_updated": time.Now(),
	}
}

// GetAlertChannel returns the alert channel
func (hc *HealthChecker) GetAlertChannel() <-chan *HealthAlert {
	return hc.alertCh
}

// IsHealthy checks if the system is overall healthy
func (hc *HealthChecker) IsHealthy() bool {
	overall := hc.GetOverallHealth()
	status, ok := overall["status"].(string)
	if !ok {
		return false
	}
	return status == "healthy"
}

// ForceCheck forces a health check for a specific service
func (hc *HealthChecker) ForceCheck(name string) error {
	hc.mu.RLock()
	check, exists := hc.checks[name]
	hc.mu.RUnlock()

	if !exists {
		return fmt.Errorf("health check %s not found", name)
	}

	hc.performCheck(check)
	return nil
}

// ForceCheckAll forces health checks for all services
func (hc *HealthChecker) ForceCheckAll() {
	hc.performHealthChecks()
}
