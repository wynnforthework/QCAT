package bridge

import (
	"fmt"
	"log"
	"time"
)

// EventRuleManager manages event processing rules
type EventRuleManager struct {
	bridge *MonitorResponseBridge
}

// NewEventRuleManager creates a new event rule manager
func NewEventRuleManager(bridge *MonitorResponseBridge) *EventRuleManager {
	return &EventRuleManager{
		bridge: bridge,
	}
}

// InitializeEnhancedRules initializes enhanced event processing rules
func (erm *EventRuleManager) InitializeEnhancedRules() {
	log.Println("Initializing enhanced event processing rules...")

	// Initialize default rules first
	erm.bridge.initializeDefaultRules()

	// Add performance monitoring rules
	erm.addPerformanceRules()

	// Add system monitoring rules
	erm.addSystemMonitoringRules()

	// Add trading activity rules
	erm.addTradingActivityRules()

	// Add error handling rules
	erm.addErrorHandlingRules()

	// Add maintenance rules
	erm.addMaintenanceRules()

	log.Printf("Enhanced event rules initialized. Total rules: %d", len(erm.bridge.responseRules))
}

// addPerformanceRules adds performance monitoring event rules
func (erm *EventRuleManager) addPerformanceRules() {
	// Performance monitoring rule
	erm.bridge.AddResponseRule(&ResponseRule{
		ID:        "performance_monitoring",
		Name:      "性能监控事件处理",
		EventType: EventTypePerformance,
		Conditions: []RuleCondition{
			{Field: "type", Operator: "eq", Value: "performance"},
		},
		Actions: []ResponseAction{
			{
				Type:   ActionTypeSystem,
				Action: "log_performance_metrics",
				Parameters: map[string]interface{}{
					"event_id": "{{event.id}}",
					"metrics":  "{{event.metadata}}",
				},
				Timeout:    time.Minute,
				MaxRetries: 1,
			},
		},
		Enabled:  true,
		Priority: 5,
		Cooldown: time.Minute * 2,
	})

	// Strategy performance degradation
	erm.bridge.AddResponseRule(&ResponseRule{
		ID:        "strategy_performance_degradation",
		Name:      "策略性能下降处理",
		EventType: EventTypePerformance,
		Conditions: []RuleCondition{
			{Field: "metric_type", Operator: "eq", Value: "strategy_performance"},
			{Field: "performance_score", Operator: "lt", Value: 0.3},
		},
		Actions: []ResponseAction{
			{
				Type:   ActionTypeStrategy,
				Action: "pause_strategy",
				Parameters: map[string]interface{}{
					"strategy_id": "{{event.metadata.strategy_id}}",
					"reason":      "performance_degradation",
				},
				Timeout:    time.Minute * 5,
				MaxRetries: 2,
			},
		},
		Enabled:  true,
		Priority: 2,
		Cooldown: time.Hour,
	})

	// System performance alert
	erm.bridge.AddResponseRule(&ResponseRule{
		ID:        "system_performance_alert",
		Name:      "系统性能告警处理",
		EventType: EventTypePerformance,
		Conditions: []RuleCondition{
			{Field: "metric_type", Operator: "eq", Value: "system_performance"},
			{Field: "cpu_usage", Operator: "gt", Value: 0.8},
		},
		Actions: []ResponseAction{
			{
				Type:   ActionTypeSystem,
				Action: "optimize_system_resources",
				Parameters: map[string]interface{}{
					"target": "cpu_optimization",
				},
				Timeout:    time.Minute * 10,
				MaxRetries: 1,
			},
		},
		Enabled:  true,
		Priority: 3,
		Cooldown: time.Minute * 15,
	})
}

// addSystemMonitoringRules adds system monitoring event rules
func (erm *EventRuleManager) addSystemMonitoringRules() {
	// Health check failure
	erm.bridge.AddResponseRule(&ResponseRule{
		ID:        "health_check_failure",
		Name:      "健康检查失败处理",
		EventType: EventTypeSystem,
		Conditions: []RuleCondition{
			{Field: "event_type", Operator: "eq", Value: "health_check_failure"},
		},
		Actions: []ResponseAction{
			{
				Type:   ActionTypeSystem,
				Action: "restart_failed_service",
				Parameters: map[string]interface{}{
					"service_name": "{{event.metadata.service_name}}",
				},
				Timeout:    time.Minute * 3,
				MaxRetries: 2,
			},
		},
		Enabled:  true,
		Priority: 1,
		Cooldown: time.Minute * 5,
	})

	// Database connection failure
	erm.bridge.AddResponseRule(&ResponseRule{
		ID:        "database_connection_failure",
		Name:      "数据库连接失败处理",
		EventType: EventTypeSystem,
		Conditions: []RuleCondition{
			{Field: "component", Operator: "eq", Value: "database"},
			{Field: "status", Operator: "eq", Value: "connection_failed"},
		},
		Actions: []ResponseAction{
			{
				Type:   ActionTypeSystem,
				Action: "reconnect_database",
				Parameters: map[string]interface{}{
					"max_attempts": 3,
					"backoff":      "exponential",
				},
				Timeout:    time.Minute * 2,
				MaxRetries: 3,
			},
		},
		Enabled:  true,
		Priority: 1,
		Cooldown: time.Minute * 2,
	})

	// Cache failure
	erm.bridge.AddResponseRule(&ResponseRule{
		ID:        "cache_failure_handling",
		Name:      "缓存失败处理",
		EventType: EventTypeSystem,
		Conditions: []RuleCondition{
			{Field: "component", Operator: "eq", Value: "cache"},
			{Field: "status", Operator: "eq", Value: "failure"},
		},
		Actions: []ResponseAction{
			{
				Type:   ActionTypeSystem,
				Action: "enable_cache_fallback",
				Parameters: map[string]interface{}{
					"fallback_mode": "memory",
				},
				Timeout:    time.Minute,
				MaxRetries: 1,
			},
		},
		Enabled:  true,
		Priority: 2,
		Cooldown: time.Minute * 5,
	})
}

// addTradingActivityRules adds trading activity event rules
func (erm *EventRuleManager) addTradingActivityRules() {
	// Order execution failure
	erm.bridge.AddResponseRule(&ResponseRule{
		ID:        "order_execution_failure",
		Name:      "订单执行失败处理",
		EventType: EventTypeTrading,
		Conditions: []RuleCondition{
			{Field: "event_type", Operator: "eq", Value: "order_failed"},
		},
		Actions: []ResponseAction{
			{
				Type:   ActionTypePosition,
				Action: "retry_order_execution",
				Parameters: map[string]interface{}{
					"order_id":     "{{event.metadata.order_id}}",
					"retry_count":  3,
					"retry_delay":  "5s",
				},
				Timeout:    time.Minute * 2,
				MaxRetries: 3,
			},
		},
		Enabled:  true,
		Priority: 2,
		Cooldown: time.Minute,
	})

	// Position size limit exceeded
	erm.bridge.AddResponseRule(&ResponseRule{
		ID:        "position_size_limit_exceeded",
		Name:      "仓位大小限制超出处理",
		EventType: EventTypeRiskViolation,
		Conditions: []RuleCondition{
			{Field: "violation_type", Operator: "eq", Value: "position_size_limit"},
		},
		Actions: []ResponseAction{
			{
				Type:   ActionTypePosition,
				Action: "reduce_position_size",
				Parameters: map[string]interface{}{
					"symbol":          "{{event.metadata.symbol}}",
					"target_ratio":    0.8,
				},
				Timeout:    time.Minute * 5,
				MaxRetries: 2,
			},
		},
		Enabled:  true,
		Priority: 1,
		Cooldown: time.Minute * 10,
	})
}

// addErrorHandlingRules adds error handling event rules
func (erm *EventRuleManager) addErrorHandlingRules() {
	// API rate limit exceeded
	erm.bridge.AddResponseRule(&ResponseRule{
		ID:        "api_rate_limit_exceeded",
		Name:      "API速率限制超出处理",
		EventType: EventTypeSystem,
		Conditions: []RuleCondition{
			{Field: "error_type", Operator: "eq", Value: "rate_limit_exceeded"},
		},
		Actions: []ResponseAction{
			{
				Type:   ActionTypeSystem,
				Action: "enable_rate_limit_backoff",
				Parameters: map[string]interface{}{
					"backoff_duration": "30s",
					"endpoint":         "{{event.metadata.endpoint}}",
				},
				Timeout:    time.Minute,
				MaxRetries: 1,
			},
		},
		Enabled:  true,
		Priority: 3,
		Cooldown: time.Minute * 2,
	})

	// Network connectivity issues
	erm.bridge.AddResponseRule(&ResponseRule{
		ID:        "network_connectivity_issues",
		Name:      "网络连接问题处理",
		EventType: EventTypeSystem,
		Conditions: []RuleCondition{
			{Field: "error_type", Operator: "eq", Value: "network_error"},
		},
		Actions: []ResponseAction{
			{
				Type:   ActionTypeSystem,
				Action: "switch_to_backup_endpoint",
				Parameters: map[string]interface{}{
					"primary_endpoint": "{{event.metadata.endpoint}}",
				},
				Timeout:    time.Minute * 2,
				MaxRetries: 2,
			},
		},
		Enabled:  true,
		Priority: 2,
		Cooldown: time.Minute * 5,
	})
}

// addMaintenanceRules adds maintenance event rules
func (erm *EventRuleManager) addMaintenanceRules() {
	// Scheduled maintenance
	erm.bridge.AddResponseRule(&ResponseRule{
		ID:        "scheduled_maintenance",
		Name:      "计划维护处理",
		EventType: EventTypeMaintenance,
		Conditions: []RuleCondition{
			{Field: "maintenance_type", Operator: "eq", Value: "scheduled"},
		},
		Actions: []ResponseAction{
			{
				Type:   ActionTypeSystem,
				Action: "prepare_for_maintenance",
				Parameters: map[string]interface{}{
					"maintenance_window": "{{event.metadata.window}}",
				},
				Timeout:    time.Minute * 10,
				MaxRetries: 1,
			},
		},
		Enabled:  true,
		Priority: 4,
		Cooldown: time.Hour,
	})

	// Emergency maintenance
	erm.bridge.AddResponseRule(&ResponseRule{
		ID:        "emergency_maintenance",
		Name:      "紧急维护处理",
		EventType: EventTypeMaintenance,
		Conditions: []RuleCondition{
			{Field: "maintenance_type", Operator: "eq", Value: "emergency"},
		},
		Actions: []ResponseAction{
			{
				Type:   ActionTypeRisk,
				Action: "emergency_shutdown",
				Parameters: map[string]interface{}{
					"reason": "emergency_maintenance",
				},
				Timeout:    time.Minute * 5,
				MaxRetries: 1,
			},
		},
		Enabled:  true,
		Priority: 1,
		Cooldown: time.Minute * 30,
	})
}

// ValidateRules validates all configured rules
func (erm *EventRuleManager) ValidateRules() error {
	for _, rule := range erm.bridge.responseRules {
		if err := erm.validateRule(rule); err != nil {
			return fmt.Errorf("rule validation failed for %s: %w", rule.ID, err)
		}
	}
	return nil
}

// validateRule validates a single rule
func (erm *EventRuleManager) validateRule(rule *ResponseRule) error {
	if rule.ID == "" {
		return fmt.Errorf("rule ID is required")
	}
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	if len(rule.Actions) == 0 {
		return fmt.Errorf("at least one action is required")
	}
	for i, action := range rule.Actions {
		if action.Action == "" {
			return fmt.Errorf("action %d: action name is required", i)
		}
		if action.Type == "" {
			return fmt.Errorf("action %d: action type is required", i)
		}
	}
	return nil
}

// GetRuleStatistics returns statistics about rule usage
func (erm *EventRuleManager) GetRuleStatistics() map[string]interface{} {
	stats := make(map[string]interface{})
	
	totalRules := len(erm.bridge.responseRules)
	enabledRules := 0
	rulesByType := make(map[EventType]int)
	
	for _, rule := range erm.bridge.responseRules {
		if rule.Enabled {
			enabledRules++
		}
		rulesByType[rule.EventType]++
	}
	
	stats["total_rules"] = totalRules
	stats["enabled_rules"] = enabledRules
	stats["disabled_rules"] = totalRules - enabledRules
	stats["rules_by_type"] = rulesByType
	
	return stats
}
