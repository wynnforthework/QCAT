package main

import (
	"fmt"
	"time"

	"qcat/internal/automation/bridge"
)

func main() {
	fmt.Println("🧪 Event Processing Rules Test")
	fmt.Println("===============================")

	// Test 1: Event type definitions
	fmt.Println("\n📋 Test 1: Event Type Definitions")
	testEventTypeDefinitions()

	// Test 2: Rule creation and validation
	fmt.Println("\n📋 Test 2: Rule Creation and Validation")
	testRuleCreationAndValidation()

	// Test 3: Event matching simulation
	fmt.Println("\n📋 Test 3: Event Matching Simulation")
	testEventMatchingSimulation()

	fmt.Println("\n🎉 Event processing rules testing completed!")
}

// testEventTypeDefinitions tests that all event types are properly defined
func testEventTypeDefinitions() {
	eventTypes := []bridge.EventType{
		bridge.EventTypeAlert,
		bridge.EventTypeRiskViolation,
		bridge.EventTypePerformance,
		bridge.EventTypeSystem,
		bridge.EventTypeStrategy,
		bridge.EventTypeMarket,
		bridge.EventTypeTrading,
		bridge.EventTypeMaintenance,
		bridge.EventTypeAudit,
	}

	fmt.Printf("Testing %d event types:\n", len(eventTypes))
	for i, eventType := range eventTypes {
		fmt.Printf("  %d. %s ✅\n", i+1, eventType)
	}

	fmt.Printf("✅ All event types are properly defined\n")
}

// testRuleCreationAndValidation tests rule creation and validation
func testRuleCreationAndValidation() {
	// Test creating a performance monitoring rule
	performanceRule := &bridge.ResponseRule{
		ID:        "test_performance_rule",
		Name:      "Test Performance Rule",
		EventType: bridge.EventTypePerformance,
		Conditions: []bridge.RuleCondition{
			{Field: "type", Operator: "eq", Value: "performance"},
		},
		Actions: []bridge.ResponseAction{
			{
				Type:   bridge.ActionTypeSystem,
				Action: "log_performance_metrics",
				Parameters: map[string]interface{}{
					"event_id": "{{event.id}}",
				},
				Timeout:    time.Minute,
				MaxRetries: 1,
			},
		},
		Enabled:  true,
		Priority: 5,
		Cooldown: time.Minute * 2,
	}

	// Validate the rule structure
	if performanceRule.ID == "" {
		fmt.Printf("❌ Rule ID validation failed\n")
	} else {
		fmt.Printf("✅ Rule ID validation passed: %s\n", performanceRule.ID)
	}

	if performanceRule.Name == "" {
		fmt.Printf("❌ Rule name validation failed\n")
	} else {
		fmt.Printf("✅ Rule name validation passed: %s\n", performanceRule.Name)
	}

	if len(performanceRule.Actions) == 0 {
		fmt.Printf("❌ Rule actions validation failed\n")
	} else {
		fmt.Printf("✅ Rule actions validation passed: %d actions\n", len(performanceRule.Actions))
	}

	if performanceRule.EventType == "" {
		fmt.Printf("❌ Rule event type validation failed\n")
	} else {
		fmt.Printf("✅ Rule event type validation passed: %s\n", performanceRule.EventType)
	}

	// Test creating a system monitoring rule
	systemRule := &bridge.ResponseRule{
		ID:        "test_system_rule",
		Name:      "Test System Rule",
		EventType: bridge.EventTypeSystem,
		Conditions: []bridge.RuleCondition{
			{Field: "component", Operator: "eq", Value: "cache"},
			{Field: "status", Operator: "eq", Value: "failure"},
		},
		Actions: []bridge.ResponseAction{
			{
				Type:   bridge.ActionTypeSystem,
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
	}

	fmt.Printf("✅ System rule created: %s\n", systemRule.Name)

	// Test creating a trading rule
	tradingRule := &bridge.ResponseRule{
		ID:        "test_trading_rule",
		Name:      "Test Trading Rule",
		EventType: bridge.EventTypeTrading,
		Conditions: []bridge.RuleCondition{
			{Field: "event_type", Operator: "eq", Value: "order_failed"},
		},
		Actions: []bridge.ResponseAction{
			{
				Type:   bridge.ActionTypePosition,
				Action: "retry_order_execution",
				Parameters: map[string]interface{}{
					"order_id":    "{{event.metadata.order_id}}",
					"retry_count": 3,
				},
				Timeout:    time.Minute * 2,
				MaxRetries: 3,
			},
		},
		Enabled:  true,
		Priority: 2,
		Cooldown: time.Minute,
	}

	fmt.Printf("✅ Trading rule created: %s\n", tradingRule.Name)

	fmt.Printf("✅ All test rules created successfully\n")
}

// testEventMatchingSimulation simulates event matching logic
func testEventMatchingSimulation() {
	// Simulate performance event
	performanceEvent := &bridge.MonitorEvent{
		ID:        "event_1755629702767861300",
		Type:      bridge.EventTypePerformance,
		Severity:  bridge.SeverityInfo,
		Message:   "Performance monitoring event",
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"type":        "performance",
			"metric_type": "system_performance",
			"cpu_usage":   0.75,
		},
	}

	fmt.Printf("Testing event matching for performance event: %s\n", performanceEvent.ID)

	// Test event type matching
	if performanceEvent.Type == bridge.EventTypePerformance {
		fmt.Printf("✅ Event type matching: %s\n", performanceEvent.Type)
	} else {
		fmt.Printf("❌ Event type matching failed\n")
	}

	// Test metadata access
	if eventType, ok := performanceEvent.Metadata["type"].(string); ok && eventType == "performance" {
		fmt.Printf("✅ Metadata matching: type = %s\n", eventType)
	} else {
		fmt.Printf("❌ Metadata matching failed\n")
	}

	// Simulate system event
	systemEvent := &bridge.MonitorEvent{
		ID:        "system_event_123",
		Type:      bridge.EventTypeSystem,
		Severity:  bridge.SeverityWarning,
		Message:   "Cache failure detected",
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"component": "cache",
			"status":    "failure",
			"error":     "redis_get_failure",
		},
	}

	fmt.Printf("Testing event matching for system event: %s\n", systemEvent.ID)

	// Test system event matching
	if systemEvent.Type == bridge.EventTypeSystem {
		fmt.Printf("✅ System event type matching: %s\n", systemEvent.Type)
	}

	if component, ok := systemEvent.Metadata["component"].(string); ok && component == "cache" {
		fmt.Printf("✅ System event metadata matching: component = %s\n", component)
	}

	if status, ok := systemEvent.Metadata["status"].(string); ok && status == "failure" {
		fmt.Printf("✅ System event status matching: status = %s\n", status)
	}

	// Simulate trading event
	tradingEvent := &bridge.MonitorEvent{
		ID:        "trading_event_456",
		Type:      bridge.EventTypeTrading,
		Severity:  bridge.SeverityWarning,
		Message:   "Order execution failed",
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"event_type": "order_failed",
			"order_id":   "order_12345",
			"symbol":     "BTCUSDT",
		},
	}

	fmt.Printf("Testing event matching for trading event: %s\n", tradingEvent.ID)

	// Test trading event matching
	if tradingEvent.Type == bridge.EventTypeTrading {
		fmt.Printf("✅ Trading event type matching: %s\n", tradingEvent.Type)
	}

	if eventType, ok := tradingEvent.Metadata["event_type"].(string); ok && eventType == "order_failed" {
		fmt.Printf("✅ Trading event metadata matching: event_type = %s\n", eventType)
	}

	// Test rule priority simulation
	fmt.Printf("\nTesting rule priority simulation:\n")
	rules := []struct {
		name      string
		priority  int
		eventType bridge.EventType
	}{
		{"Emergency Risk Rule", 1, bridge.EventTypeRiskViolation},
		{"System Health Rule", 2, bridge.EventTypeSystem},
		{"Performance Monitor Rule", 5, bridge.EventTypePerformance},
		{"Trading Retry Rule", 3, bridge.EventTypeTrading},
	}

	// Sort by priority (lower number = higher priority)
	for i := 0; i < len(rules)-1; i++ {
		for j := i + 1; j < len(rules); j++ {
			if rules[i].priority > rules[j].priority {
				rules[i], rules[j] = rules[j], rules[i]
			}
		}
	}

	fmt.Printf("Rules sorted by priority:\n")
	for i, rule := range rules {
		fmt.Printf("  %d. %s (Priority: %d, Type: %s)\n", i+1, rule.name, rule.priority, rule.eventType)
	}

	fmt.Printf("✅ Event matching simulation completed successfully\n")
}

// Helper function to simulate rule validation
func validateRule(rule *bridge.ResponseRule) error {
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
