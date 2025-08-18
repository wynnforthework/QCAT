package security

import (
	"fmt"
	"sync"
	"time"
)

// KeyMonitor monitors key usage and security events
type KeyMonitor struct {
	config     *MonitorConfig
	events     []KeyEvent
	usageStats map[string]*KeyUsageStats
	alerts     []SecurityAlert
	mu         sync.RWMutex
}

// MonitorConfig represents monitor configuration
type MonitorConfig struct {
	MaxEvents        int           `json:"max_events"`
	AlertThresholds  AlertThresholds `json:"alert_thresholds"`
	RetentionPeriod  time.Duration `json:"retention_period"`
	EnableAlerting   bool          `json:"enable_alerting"`
	AlertChannels    []string      `json:"alert_channels"`
}

// AlertThresholds defines thresholds for security alerts
type AlertThresholds struct {
	MaxFailedValidations int           `json:"max_failed_validations"`
	MaxUsagePerHour      int64         `json:"max_usage_per_hour"`
	MaxUsagePerDay       int64         `json:"max_usage_per_day"`
	SuspiciousPatterns   bool          `json:"suspicious_patterns"`
	UnusualAccessTimes   bool          `json:"unusual_access_times"`
	AlertCooldown        time.Duration `json:"alert_cooldown"`
}

// SecurityAlert represents a security alert
type SecurityAlert struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    AlertSeverity          `json:"severity"`
	KeyID       string                 `json:"key_id"`
	Message     string                 `json:"message"`
	Details     map[string]interface{} `json:"details"`
	Timestamp   time.Time              `json:"timestamp"`
	Acknowledged bool                  `json:"acknowledged"`
	AckedBy     string                 `json:"acked_by,omitempty"`
	AckedAt     time.Time              `json:"acked_at,omitempty"`
}

// AlertSeverity represents the severity of an alert
type AlertSeverity string

const (
	SeverityLow      AlertSeverity = "low"
	SeverityMedium   AlertSeverity = "medium"
	SeverityHigh     AlertSeverity = "high"
	SeverityCritical AlertSeverity = "critical"
)

// NewKeyMonitor creates a new key monitor
func NewKeyMonitor(config *MonitorConfig) *KeyMonitor {
	if config == nil {
		config = DefaultMonitorConfig()
	}

	monitor := &KeyMonitor{
		config:     config,
		events:     make([]KeyEvent, 0),
		usageStats: make(map[string]*KeyUsageStats),
		alerts:     make([]SecurityAlert, 0),
	}

	// Start cleanup routine
	go monitor.startCleanupRoutine()

	return monitor
}

// LogKeyEvent logs a key-related event
func (km *KeyMonitor) LogKeyEvent(event KeyEvent) {
	km.mu.Lock()
	defer km.mu.Unlock()

	// Add event to the beginning of the slice
	km.events = append([]KeyEvent{event}, km.events...)

	// Keep only the most recent events
	if len(km.events) > km.config.MaxEvents {
		km.events = km.events[:km.config.MaxEvents]
	}

	// Update usage statistics
	km.updateUsageStats(event)

	// Check for alerts
	if km.config.EnableAlerting {
		km.checkForAlerts(event)
	}
}

// LogKeyRotation logs a key rotation event
func (km *KeyMonitor) LogKeyRotation(keyID string, timestamp time.Time) {
	event := KeyEvent{
		Type:      "key_rotated",
		KeyID:     keyID,
		Timestamp: timestamp,
		Details: map[string]interface{}{
			"rotation_type": "automatic",
		},
	}

	km.LogKeyEvent(event)
}

// LogError logs an error event
func (km *KeyMonitor) LogError(eventType string, err error) {
	event := KeyEvent{
		Type:      eventType,
		KeyID:     "system",
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"error": err.Error(),
		},
	}

	km.LogKeyEvent(event)
}

// GetKeyUsageStats returns usage statistics for a key
func (km *KeyMonitor) GetKeyUsageStats(keyID string, period time.Duration) (*KeyUsageStats, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	stats, exists := km.usageStats[keyID]
	if !exists {
		return nil, fmt.Errorf("no usage stats found for key: %s", keyID)
	}

	// Calculate period-specific stats
	periodStats := &KeyUsageStats{
		KeyID:       keyID,
		TotalUsage:  stats.TotalUsage,
		LastUsed:    stats.LastUsed,
	}

	// Count usage in the specified period
	cutoff := time.Now().Add(-period)
	var periodUsage int64

	for _, event := range km.events {
		if event.KeyID == keyID && event.Type == "key_validated" && event.Timestamp.After(cutoff) {
			periodUsage++
			
			// Group by day for daily average calculation
			day := event.Timestamp.Truncate(24 * time.Hour)
			// This is a simplified calculation - in production, you'd want more sophisticated tracking
			_ = day
		}
	}

	periodStats.PeriodUsage = periodUsage
	
	// Calculate average daily usage
	days := int(period.Hours() / 24)
	if days > 0 {
		periodStats.AverageDaily = float64(periodUsage) / float64(days)
	}

	return periodStats, nil
}

// GetRecentEvents returns recent events
func (km *KeyMonitor) GetRecentEvents(limit int) []KeyEvent {
	km.mu.RLock()
	defer km.mu.RUnlock()

	if limit <= 0 || limit > len(km.events) {
		limit = len(km.events)
	}

	events := make([]KeyEvent, limit)
	copy(events, km.events[:limit])
	return events
}

// GetAlerts returns security alerts
func (km *KeyMonitor) GetAlerts(acknowledged bool) []SecurityAlert {
	km.mu.RLock()
	defer km.mu.RUnlock()

	var alerts []SecurityAlert
	for _, alert := range km.alerts {
		if alert.Acknowledged == acknowledged {
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

// AcknowledgeAlert acknowledges a security alert
func (km *KeyMonitor) AcknowledgeAlert(alertID, acknowledgedBy string) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	for i, alert := range km.alerts {
		if alert.ID == alertID {
			km.alerts[i].Acknowledged = true
			km.alerts[i].AckedBy = acknowledgedBy
			km.alerts[i].AckedAt = time.Now()
			return nil
		}
	}

	return fmt.Errorf("alert not found: %s", alertID)
}

// updateUsageStats updates usage statistics for a key
func (km *KeyMonitor) updateUsageStats(event KeyEvent) {
	if event.Type != "key_validated" {
		return
	}

	stats, exists := km.usageStats[event.KeyID]
	if !exists {
		stats = &KeyUsageStats{
			KeyID: event.KeyID,
		}
		km.usageStats[event.KeyID] = stats
	}

	stats.TotalUsage++
	stats.LastUsed = event.Timestamp
}

// checkForAlerts checks if an event should trigger an alert
func (km *KeyMonitor) checkForAlerts(event KeyEvent) {
	switch event.Type {
	case "key_validation_failed":
		km.checkFailedValidationAlert(event)
	case "key_validated":
		km.checkUsageAlert(event)
	case "key_expired":
		km.createAlert("key_expired", SeverityMedium, event.KeyID, "API key has expired", event.Details)
	case "key_revoked":
		km.createAlert("key_revoked", SeverityHigh, event.KeyID, "API key has been revoked", event.Details)
	}
}

// checkFailedValidationAlert checks for failed validation alerts
func (km *KeyMonitor) checkFailedValidationAlert(event KeyEvent) {
	// Count recent failed validations
	cutoff := time.Now().Add(-time.Hour)
	failedCount := 0

	for _, e := range km.events {
		if e.Type == "key_validation_failed" && e.Timestamp.After(cutoff) {
			failedCount++
		}
	}

	if failedCount >= km.config.AlertThresholds.MaxFailedValidations {
		km.createAlert(
			"excessive_failed_validations",
			SeverityHigh,
			"system",
			fmt.Sprintf("Excessive failed key validations: %d in the last hour", failedCount),
			map[string]interface{}{"failed_count": failedCount},
		)
	}
}

// checkUsageAlert checks for usage-based alerts
func (km *KeyMonitor) checkUsageAlert(event KeyEvent) {
	stats := km.usageStats[event.KeyID]
	if stats == nil {
		return
	}

	// Check hourly usage
	hourlyUsage := km.countRecentUsage(event.KeyID, time.Hour)
	if hourlyUsage > km.config.AlertThresholds.MaxUsagePerHour {
		km.createAlert(
			"excessive_hourly_usage",
			SeverityMedium,
			event.KeyID,
			fmt.Sprintf("Excessive API key usage: %d requests in the last hour", hourlyUsage),
			map[string]interface{}{"hourly_usage": hourlyUsage},
		)
	}

	// Check daily usage
	dailyUsage := km.countRecentUsage(event.KeyID, 24*time.Hour)
	if dailyUsage > km.config.AlertThresholds.MaxUsagePerDay {
		km.createAlert(
			"excessive_daily_usage",
			SeverityMedium,
			event.KeyID,
			fmt.Sprintf("Excessive API key usage: %d requests in the last day", dailyUsage),
			map[string]interface{}{"daily_usage": dailyUsage},
		)
	}
}

// countRecentUsage counts recent usage for a key
func (km *KeyMonitor) countRecentUsage(keyID string, period time.Duration) int64 {
	cutoff := time.Now().Add(-period)
	var count int64

	for _, event := range km.events {
		if event.KeyID == keyID && event.Type == "key_validated" && event.Timestamp.After(cutoff) {
			count++
		}
	}

	return count
}

// createAlert creates a new security alert
func (km *KeyMonitor) createAlert(alertType string, severity AlertSeverity, keyID, message string, details map[string]interface{}) {
	// Check if we should create this alert (cooldown period)
	if km.isInCooldown(alertType, keyID) {
		return
	}

	alert := SecurityAlert{
		ID:        generateAlertID(),
		Type:      alertType,
		Severity:  severity,
		KeyID:     keyID,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
	}

	km.alerts = append(km.alerts, alert)

	// Send alert through configured channels
	km.sendAlert(alert)
}

// isInCooldown checks if an alert type is in cooldown period
func (km *KeyMonitor) isInCooldown(alertType, keyID string) bool {
	cutoff := time.Now().Add(-km.config.AlertThresholds.AlertCooldown)

	for _, alert := range km.alerts {
		if alert.Type == alertType && alert.KeyID == keyID && alert.Timestamp.After(cutoff) {
			return true
		}
	}

	return false
}

// sendAlert sends an alert through configured channels
func (km *KeyMonitor) sendAlert(alert SecurityAlert) {
	// This is a placeholder - in production, you'd integrate with
	// actual alerting systems like email, Slack, PagerDuty, etc.
	fmt.Printf("SECURITY ALERT [%s]: %s (Key: %s)\n", alert.Severity, alert.Message, alert.KeyID)
}

// startCleanupRoutine starts the cleanup routine for old events and alerts
func (km *KeyMonitor) startCleanupRoutine() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		km.cleanup()
	}
}

// cleanup removes old events and alerts
func (km *KeyMonitor) cleanup() {
	km.mu.Lock()
	defer km.mu.Unlock()

	cutoff := time.Now().Add(-km.config.RetentionPeriod)

	// Clean up old events
	var newEvents []KeyEvent
	for _, event := range km.events {
		if event.Timestamp.After(cutoff) {
			newEvents = append(newEvents, event)
		}
	}
	km.events = newEvents

	// Clean up old alerts (keep acknowledged ones longer)
	var newAlerts []SecurityAlert
	for _, alert := range km.alerts {
		keepAlert := alert.Timestamp.After(cutoff)
		if alert.Acknowledged {
			// Keep acknowledged alerts for longer
			keepAlert = alert.Timestamp.After(cutoff.Add(-24 * time.Hour))
		}

		if keepAlert {
			newAlerts = append(newAlerts, alert)
		}
	}
	km.alerts = newAlerts
}

// DefaultMonitorConfig returns default monitor configuration
func DefaultMonitorConfig() *MonitorConfig {
	return &MonitorConfig{
		MaxEvents:       10000,
		RetentionPeriod: 30 * 24 * time.Hour, // 30 days
		EnableAlerting:  true,
		AlertChannels:   []string{"console"},
		AlertThresholds: AlertThresholds{
			MaxFailedValidations: 10,
			MaxUsagePerHour:      1000,
			MaxUsagePerDay:       10000,
			SuspiciousPatterns:   true,
			UnusualAccessTimes:   true,
			AlertCooldown:        time.Hour,
		},
	}
}

// generateAlertID generates a unique alert ID
func generateAlertID() string {
	return fmt.Sprintf("alert-%d", time.Now().UnixNano())
}