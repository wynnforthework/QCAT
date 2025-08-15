package monitor

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AlertManager manages system alerts
type AlertManager struct {
	alerts     map[string]*Alert
	rules      map[string]*AlertRule
	channels   map[string]AlertChannel
	mu         sync.RWMutex
}

// Alert represents a system alert
type Alert struct {
	ID          string
	RuleID      string
	Severity    AlertSeverity
	Message     string
	Details     map[string]interface{}
	Status      AlertStatus
	CreatedAt   time.Time
	ResolvedAt  time.Time
	ResolvedBy  string
}

// AlertSeverity represents alert severity level
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertStatus represents alert status
type AlertStatus string

const (
	AlertStatusActive   AlertStatus = "active"
	AlertStatusResolved AlertStatus = "resolved"
)

// AlertRule represents an alert rule
type AlertRule struct {
	ID          string
	Name        string
	Description string
	Severity    AlertSeverity
	Condition   AlertCondition
	Channels    []string
	Enabled     bool
}

// AlertCondition represents alert condition
type AlertCondition struct {
	Metric    string
	Operator  string
	Threshold float64
	Duration  time.Duration
}

// AlertChannel represents an alert notification channel
type AlertChannel interface {
	Send(ctx context.Context, alert *Alert) error
}

// EmailChannel represents email alert channel
type EmailChannel struct {
	SMTPHost     string
	SMTPPort     int
	Username     string
	Password     string
	FromEmail    string
	ToEmails     []string
}

// SlackChannel represents Slack alert channel
type SlackChannel struct {
	WebhookURL string
	Channel    string
}

// NewAlertManager creates a new alert manager
func NewAlertManager() *AlertManager {
	return &AlertManager{
		alerts:   make(map[string]*Alert),
		rules:    make(map[string]*AlertRule),
		channels: make(map[string]AlertChannel),
	}
}

// AddRule adds an alert rule
func (am *AlertManager) AddRule(rule *AlertRule) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if _, exists := am.rules[rule.ID]; exists {
		return fmt.Errorf("rule already exists: %s", rule.ID)
	}

	am.rules[rule.ID] = rule
	return nil
}

// RemoveRule removes an alert rule
func (am *AlertManager) RemoveRule(ruleID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if _, exists := am.rules[ruleID]; !exists {
		return fmt.Errorf("rule not found: %s", ruleID)
	}

	delete(am.rules, ruleID)
	return nil
}

// AddChannel adds an alert channel
func (am *AlertManager) AddChannel(name string, channel AlertChannel) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.channels[name] = channel
}

// RemoveChannel removes an alert channel
func (am *AlertManager) RemoveChannel(name string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	delete(am.channels, name)
}

// CheckRules checks all alert rules
func (am *AlertManager) CheckRules(ctx context.Context, metrics map[string]float64) error {
	am.mu.RLock()
	rules := make([]*AlertRule, 0, len(am.rules))
	for _, rule := range am.rules {
		if rule.Enabled {
			rules = append(rules, rule)
		}
	}
	am.mu.RUnlock()

	for _, rule := range rules {
		if am.evaluateCondition(rule.Condition, metrics) {
			if err := am.triggerAlert(ctx, rule, metrics); err != nil {
				return fmt.Errorf("failed to trigger alert for rule %s: %w", rule.ID, err)
			}
		}
	}

	return nil
}

// evaluateCondition evaluates an alert condition
func (am *AlertManager) evaluateCondition(condition AlertCondition, metrics map[string]float64) bool {
	value, exists := metrics[condition.Metric]
	if !exists {
		return false
	}

	switch condition.Operator {
	case ">":
		return value > condition.Threshold
	case ">=":
		return value >= condition.Threshold
	case "<":
		return value < condition.Threshold
	case "<=":
		return value <= condition.Threshold
	case "==":
		return value == condition.Threshold
	case "!=":
		return value != condition.Threshold
	default:
		return false
	}
}

// triggerAlert triggers an alert
func (am *AlertManager) triggerAlert(ctx context.Context, rule *AlertRule, metrics map[string]float64) error {
	alert := &Alert{
		ID:        fmt.Sprintf("alert_%d", time.Now().UnixNano()),
		RuleID:    rule.ID,
		Severity:  rule.Severity,
		Message:   rule.Description,
		Details:   make(map[string]interface{}),
		Status:    AlertStatusActive,
		CreatedAt: time.Now(),
	}

	// Add metric details
	for metric, value := range metrics {
		alert.Details[metric] = value
	}

	am.mu.Lock()
	am.alerts[alert.ID] = alert
	am.mu.Unlock()

	// Send notifications
	for _, channelName := range rule.Channels {
		if channel, exists := am.channels[channelName]; exists {
			if err := channel.Send(ctx, alert); err != nil {
				return fmt.Errorf("failed to send alert to channel %s: %w", channelName, err)
			}
		}
	}

	return nil
}

// ResolveAlert resolves an alert
func (am *AlertManager) ResolveAlert(alertID, resolvedBy string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert, exists := am.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert not found: %s", alertID)
	}

	if alert.Status == AlertStatusResolved {
		return fmt.Errorf("alert already resolved: %s", alertID)
	}

	alert.Status = AlertStatusResolved
	alert.ResolvedAt = time.Now()
	alert.ResolvedBy = resolvedBy

	return nil
}

// GetAlert gets an alert by ID
func (am *AlertManager) GetAlert(alertID string) (*Alert, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	alert, exists := am.alerts[alertID]
	if !exists {
		return nil, fmt.Errorf("alert not found: %s", alertID)
	}

	return alert, nil
}

// ListAlerts lists all alerts
func (am *AlertManager) ListAlerts(status AlertStatus) []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var alerts []*Alert
	for _, alert := range am.alerts {
		if status == "" || alert.Status == status {
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

// GetRule gets a rule by ID
func (am *AlertManager) GetRule(ruleID string) (*AlertRule, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	rule, exists := am.rules[ruleID]
	if !exists {
		return nil, fmt.Errorf("rule not found: %s", ruleID)
	}

	return rule, nil
}

// ListRules lists all rules
func (am *AlertManager) ListRules() []*AlertRule {
	am.mu.RLock()
	defer am.mu.RUnlock()

	rules := make([]*AlertRule, 0, len(am.rules))
	for _, rule := range am.rules {
		rules = append(rules, rule)
	}

	return rules
}

// CleanupResolvedAlerts removes resolved alerts older than the specified duration
func (am *AlertManager) CleanupResolvedAlerts(age time.Duration) {
	am.mu.Lock()
	defer am.mu.Unlock()

	now := time.Now()
	for id, alert := range am.alerts {
		if alert.Status == AlertStatusResolved && now.Sub(alert.ResolvedAt) > age {
			delete(am.alerts, id)
		}
	}
}

// Send implements AlertChannel interface for EmailChannel
func (ec *EmailChannel) Send(ctx context.Context, alert *Alert) error {
	// TODO: Implement email sending logic
	return nil
}

// Send implements AlertChannel interface for SlackChannel
func (sc *SlackChannel) Send(ctx context.Context, alert *Alert) error {
	// TODO: Implement Slack webhook sending logic
	return nil
}
