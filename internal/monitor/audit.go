package monitor

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AuditLogger manages audit logging
type AuditLogger struct {
	logs    []*AuditLog
	maxLogs int
	mu      sync.RWMutex
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID         string
	Timestamp  time.Time
	UserID     string
	Action     string
	Resource   string
	ResourceID string
	Details    map[string]interface{}
	IPAddress  string
	UserAgent  string
	Result     AuditResult
	Duration   time.Duration
}

// AuditResult represents audit result
type AuditResult string

const (
	AuditResultSuccess AuditResult = "success"
	AuditResultFailure AuditResult = "failure"
)

// DecisionChain represents a decision chain
type DecisionChain struct {
	ID          string
	StrategyID  string
	Symbol      string
	Timestamp   time.Time
	Decisions   []*Decision
	FinalAction string
	Context     map[string]interface{}
}

// Decision represents a decision in the chain
type Decision struct {
	ID         string
	Type       string
	Input      map[string]interface{}
	Output     map[string]interface{}
	Reason     string
	Confidence float64
	Timestamp  time.Time
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(maxLogs int) *AuditLogger {
	return &AuditLogger{
		logs:    make([]*AuditLog, 0),
		maxLogs: maxLogs,
	}
}

// Log logs an audit event
func (al *AuditLogger) Log(ctx context.Context, userID, action, resource, resourceID string, details map[string]interface{}, result AuditResult, duration time.Duration) error {
	log := &AuditLog{
		ID:         fmt.Sprintf("audit_%d", time.Now().UnixNano()),
		Timestamp:  time.Now(),
		UserID:     userID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    details,
		Result:     result,
		Duration:   duration,
	}

	al.mu.Lock()
	defer al.mu.Unlock()

	al.logs = append(al.logs, log)

	// Maintain log size limit
	if len(al.logs) > al.maxLogs {
		al.logs = al.logs[1:]
	}

	return nil
}

// GetLogs gets audit logs with filters
func (al *AuditLogger) GetLogs(userID, action, resource string, startTime, endTime time.Time, limit int) []*AuditLog {
	al.mu.RLock()
	defer al.mu.RUnlock()

	var filteredLogs []*AuditLog
	for _, log := range al.logs {
		// Apply filters
		if userID != "" && log.UserID != userID {
			continue
		}
		if action != "" && log.Action != action {
			continue
		}
		if resource != "" && log.Resource != resource {
			continue
		}
		if !startTime.IsZero() && log.Timestamp.Before(startTime) {
			continue
		}
		if !endTime.IsZero() && log.Timestamp.After(endTime) {
			continue
		}

		filteredLogs = append(filteredLogs, log)
	}

	// Apply limit
	if limit > 0 && len(filteredLogs) > limit {
		filteredLogs = filteredLogs[len(filteredLogs)-limit:]
	}

	return filteredLogs
}

// GetLog gets a specific audit log
func (al *AuditLogger) GetLog(logID string) (*AuditLog, error) {
	al.mu.RLock()
	defer al.mu.RUnlock()

	for _, log := range al.logs {
		if log.ID == logID {
			return log, nil
		}
	}

	return nil, fmt.Errorf("audit log not found: %s", logID)
}

// DecisionTracker tracks decision chains
type DecisionTracker struct {
	chains map[string]*DecisionChain
	mu     sync.RWMutex
}

// NewDecisionTracker creates a new decision tracker
func NewDecisionTracker() *DecisionTracker {
	return &DecisionTracker{
		chains: make(map[string]*DecisionChain),
	}
}

// StartChain starts a new decision chain
func (dt *DecisionTracker) StartChain(strategyID, symbol string, context map[string]interface{}) *DecisionChain {
	chain := &DecisionChain{
		ID:         fmt.Sprintf("chain_%d", time.Now().UnixNano()),
		StrategyID: strategyID,
		Symbol:     symbol,
		Timestamp:  time.Now(),
		Decisions:  make([]*Decision, 0),
		Context:    context,
	}

	dt.mu.Lock()
	dt.chains[chain.ID] = chain
	dt.mu.Unlock()

	return chain
}

// AddDecision adds a decision to a chain
func (dt *DecisionTracker) AddDecision(chainID, decisionType string, input, output map[string]interface{}, reason string, confidence float64) error {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	chain, exists := dt.chains[chainID]
	if !exists {
		return fmt.Errorf("decision chain not found: %s", chainID)
	}

	decision := &Decision{
		ID:         fmt.Sprintf("decision_%d", time.Now().UnixNano()),
		Type:       decisionType,
		Input:      input,
		Output:     output,
		Reason:     reason,
		Confidence: confidence,
		Timestamp:  time.Now(),
	}

	chain.Decisions = append(chain.Decisions, decision)
	return nil
}

// CompleteChain completes a decision chain
func (dt *DecisionTracker) CompleteChain(chainID, finalAction string) error {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	chain, exists := dt.chains[chainID]
	if !exists {
		return fmt.Errorf("decision chain not found: %s", chainID)
	}

	chain.FinalAction = finalAction
	return nil
}

// GetChain gets a decision chain by ID
func (dt *DecisionTracker) GetChain(chainID string) (*DecisionChain, error) {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	chain, exists := dt.chains[chainID]
	if !exists {
		return nil, fmt.Errorf("decision chain not found: %s", chainID)
	}

	return chain, nil
}

// GetChains gets decision chains with filters
func (dt *DecisionTracker) GetChains(strategyID, symbol string, startTime, endTime time.Time, limit int) []*DecisionChain {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	var filteredChains []*DecisionChain
	for _, chain := range dt.chains {
		// Apply filters
		if strategyID != "" && chain.StrategyID != strategyID {
			continue
		}
		if symbol != "" && chain.Symbol != symbol {
			continue
		}
		if !startTime.IsZero() && chain.Timestamp.Before(startTime) {
			continue
		}
		if !endTime.IsZero() && chain.Timestamp.After(endTime) {
			continue
		}

		filteredChains = append(filteredChains, chain)
	}

	// Apply limit
	if limit > 0 && len(filteredChains) > limit {
		filteredChains = filteredChains[len(filteredChains)-limit:]
	}

	return filteredChains
}

// PerformanceMonitor monitors system performance
type PerformanceMonitor struct {
	metrics map[string]*PerformanceMetric
	mu      sync.RWMutex
}

// PerformanceMetric represents a performance metric
type PerformanceMetric struct {
	Name      string
	Value     float64
	Unit      string
	Timestamp time.Time
	Tags      map[string]string
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		metrics: make(map[string]*PerformanceMetric),
	}
}

// RecordMetric records a performance metric
func (pm *PerformanceMonitor) RecordMetric(name string, value float64, unit string, tags map[string]string) {
	metric := &PerformanceMetric{
		Name:      name,
		Value:     value,
		Unit:      unit,
		Timestamp: time.Now(),
		Tags:      tags,
	}

	pm.mu.Lock()
	pm.metrics[name] = metric
	pm.mu.Unlock()
}

// GetMetric gets a performance metric
func (pm *PerformanceMonitor) GetMetric(name string) (*PerformanceMetric, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	metric, exists := pm.metrics[name]
	if !exists {
		return nil, fmt.Errorf("metric not found: %s", name)
	}

	return metric, nil
}

// GetMetrics gets all performance metrics
func (pm *PerformanceMonitor) GetMetrics() map[string]*PerformanceMetric {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	metrics := make(map[string]*PerformanceMetric)
	for name, metric := range pm.metrics {
		metrics[name] = metric
	}

	return metrics
}

// SystemHealthChecker checks system health
type SystemHealthChecker struct {
	checks map[string]*HealthCheck
	mu     sync.RWMutex
}

// HealthCheck represents a health check
type HealthCheck struct {
	Name        string
	Status      HealthStatus
	LastCheck   time.Time
	LastError   string
	Description string
}

// HealthStatus represents health status
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// NewSystemHealthChecker creates a new system health checker
func NewSystemHealthChecker() *SystemHealthChecker {
	return &SystemHealthChecker{
		checks: make(map[string]*HealthCheck),
	}
}

// AddCheck adds a health check
func (shc *SystemHealthChecker) AddCheck(name, description string) {
	shc.mu.Lock()
	defer shc.mu.Unlock()

	shc.checks[name] = &HealthCheck{
		Name:        name,
		Status:      HealthStatusHealthy,
		LastCheck:   time.Now(),
		Description: description,
	}
}

// UpdateCheck updates a health check status
func (shc *SystemHealthChecker) UpdateCheck(name string, status HealthStatus, error string) error {
	shc.mu.Lock()
	defer shc.mu.Unlock()

	check, exists := shc.checks[name]
	if !exists {
		return fmt.Errorf("health check not found: %s", name)
	}

	check.Status = status
	check.LastCheck = time.Now()
	check.LastError = error

	return nil
}

// GetCheck gets a health check
func (shc *SystemHealthChecker) GetCheck(name string) (*HealthCheck, error) {
	shc.mu.RLock()
	defer shc.mu.RUnlock()

	check, exists := shc.checks[name]
	if !exists {
		return nil, fmt.Errorf("health check not found: %s", name)
	}

	return check, nil
}

// GetChecks gets all health checks
func (shc *SystemHealthChecker) GetChecks() map[string]*HealthCheck {
	shc.mu.RLock()
	defer shc.mu.RUnlock()

	checks := make(map[string]*HealthCheck)
	for name, check := range shc.checks {
		checks[name] = check
	}

	return checks
}

// GetOverallStatus gets overall system health status
func (shc *SystemHealthChecker) GetOverallStatus() HealthStatus {
	shc.mu.RLock()
	defer shc.mu.RUnlock()

	hasUnhealthy := false
	hasDegraded := false

	for _, check := range shc.checks {
		switch check.Status {
		case HealthStatusUnhealthy:
			hasUnhealthy = true
		case HealthStatusDegraded:
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		return HealthStatusUnhealthy
	}
	if hasDegraded {
		return HealthStatusDegraded
	}
	return HealthStatusHealthy
}
