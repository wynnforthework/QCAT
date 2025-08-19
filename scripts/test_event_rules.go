package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"qcat/internal/automation/bridge"
	"qcat/internal/automation/executor"
	"qcat/internal/config"
	"qcat/internal/monitor"
)

func main() {
	fmt.Println("🧪 Event Processing Rules Test")
	fmt.Println("===============================")

	// Test 1: Initialize event rules
	fmt.Println("\n📋 Test 1: Initialize Event Rules")
	testEventRulesInitialization()

	// Test 2: Test performance event processing
	fmt.Println("\n📋 Test 2: Performance Event Processing")
	testPerformanceEventProcessing()

	// Test 3: Test system event processing
	fmt.Println("\n📋 Test 3: System Event Processing")
	testSystemEventProcessing()

	// Test 4: Test trading event processing
	fmt.Println("\n📋 Test 4: Trading Event Processing")
	testTradingEventProcessing()

	// Test 5: Test rule validation
	fmt.Println("\n📋 Test 5: Rule Validation")
	testRuleValidation()

	fmt.Println("\n🎉 Event processing rules testing completed!")
}

// testEventRulesInitialization tests the initialization of event rules
func testEventRulesInitialization() {
	// Create mock configuration
	cfg := &config.Config{
		// Add minimal config needed for testing
	}

	// Create mock executor
	executor := &executor.RealtimeExecutor{}

	// Create mock metrics collector
	metrics := &monitor.MetricsCollector{}

	// Create monitor response bridge
	bridge := bridge.NewMonitorResponseBridge(cfg, executor, metrics)

	// Create rule manager
	ruleManager := bridge.NewEventRuleManager(bridge)

	// Initialize enhanced rules
	ruleManager.InitializeEnhancedRules()

	// Get rule statistics
	stats := ruleManager.GetRuleStatistics()
	
	fmt.Printf("✅ Event rules initialized successfully\n")
	fmt.Printf("   Total rules: %v\n", stats["total_rules"])
	fmt.Printf("   Enabled rules: %v\n", stats["enabled_rules"])
	fmt.Printf("   Disabled rules: %v\n", stats["disabled_rules"])
	
	if rulesByType, ok := stats["rules_by_type"].(map[bridge.EventType]int); ok {
		fmt.Printf("   Rules by type:\n")
		for eventType, count := range rulesByType {
			fmt.Printf("     %s: %d\n", eventType, count)
		}
	}

	// Validate rules
	if err := ruleManager.ValidateRules(); err != nil {
		fmt.Printf("❌ Rule validation failed: %v\n", err)
	} else {
		fmt.Printf("✅ All rules validated successfully\n")
	}
}

// testPerformanceEventProcessing tests performance event processing
func testPerformanceEventProcessing() {
	// Create monitor response bridge
	bridge := createTestBridge()

	// Create performance event
	performanceEvent := &bridge.MonitorEvent{
		ID:        "performance_test_event_1",
		Type:      bridge.EventTypePerformance,
		Severity:  bridge.SeverityInfo,
		Message:   "Performance monitoring event",
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"type":        "performance",
			"metric_type": "system_performance",
			"cpu_usage":   0.85,
		},
	}

	// Process the event
	fmt.Printf("Processing performance event: %s\n", performanceEvent.ID)
	err := bridge.ProcessEvent(performanceEvent)
	if err != nil {
		fmt.Printf("❌ Failed to process performance event: %v\n", err)
	} else {
		fmt.Printf("✅ Performance event processed successfully\n")
	}

	// Wait a moment for processing
	time.Sleep(2 * time.Second)

	// Check bridge statistics
	stats := bridge.GetStats()
	fmt.Printf("   Total events processed: %d\n", stats.TotalEvents)
	fmt.Printf("   Processed events: %d\n", stats.ProcessedEvents)
	fmt.Printf("   Failed events: %d\n", stats.FailedEvents)
}

// testSystemEventProcessing tests system event processing
func testSystemEventProcessing() {
	// Create monitor response bridge
	bridge := createTestBridge()

	// Create system event
	systemEvent := &bridge.MonitorEvent{
		ID:        "system_test_event_1",
		Type:      bridge.EventTypeSystem,
		Severity:  bridge.SeverityWarning,
		Message:   "Health check failure detected",
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"event_type":   "health_check_failure",
			"service_name": "optimizer",
			"component":    "system",
		},
	}

	// Process the event
	fmt.Printf("Processing system event: %s\n", systemEvent.ID)
	err := bridge.ProcessEvent(systemEvent)
	if err != nil {
		fmt.Printf("❌ Failed to process system event: %v\n", err)
	} else {
		fmt.Printf("✅ System event processed successfully\n")
	}

	// Create cache failure event
	cacheEvent := &bridge.MonitorEvent{
		ID:        "cache_failure_event_1",
		Type:      bridge.EventTypeSystem,
		Severity:  bridge.SeverityCritical,
		Message:   "Cache failure detected",
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"component": "cache",
			"status":    "failure",
			"error":     "redis_get_failure",
		},
	}

	// Process the cache event
	fmt.Printf("Processing cache failure event: %s\n", cacheEvent.ID)
	err = bridge.ProcessEvent(cacheEvent)
	if err != nil {
		fmt.Printf("❌ Failed to process cache event: %v\n", err)
	} else {
		fmt.Printf("✅ Cache failure event processed successfully\n")
	}

	// Wait a moment for processing
	time.Sleep(2 * time.Second)
}

// testTradingEventProcessing tests trading event processing
func testTradingEventProcessing() {
	// Create monitor response bridge
	bridge := createTestBridge()

	// Create trading event
	tradingEvent := &bridge.MonitorEvent{
		ID:        "trading_test_event_1",
		Type:      bridge.EventTypeTrading,
		Severity:  bridge.SeverityWarning,
		Message:   "Order execution failed",
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"event_type": "order_failed",
			"order_id":   "order_12345",
			"symbol":     "BTCUSDT",
			"error":      "insufficient_balance",
		},
	}

	// Process the event
	fmt.Printf("Processing trading event: %s\n", tradingEvent.ID)
	err := bridge.ProcessEvent(tradingEvent)
	if err != nil {
		fmt.Printf("❌ Failed to process trading event: %v\n", err)
	} else {
		fmt.Printf("✅ Trading event processed successfully\n")
	}

	// Create risk violation event
	riskEvent := &bridge.MonitorEvent{
		ID:        "risk_violation_event_1",
		Type:      bridge.EventTypeRiskViolation,
		Severity:  bridge.SeverityCritical,
		Message:   "Position size limit exceeded",
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"violation_type": "position_size_limit",
			"symbol":         "ETHUSDT",
			"current_size":   1000.0,
			"limit":          800.0,
		},
	}

	// Process the risk event
	fmt.Printf("Processing risk violation event: %s\n", riskEvent.ID)
	err = bridge.ProcessEvent(riskEvent)
	if err != nil {
		fmt.Printf("❌ Failed to process risk event: %v\n", err)
	} else {
		fmt.Printf("✅ Risk violation event processed successfully\n")
	}

	// Wait a moment for processing
	time.Sleep(2 * time.Second)
}

// testRuleValidation tests rule validation functionality
func testRuleValidation() {
	// Create monitor response bridge
	bridge := createTestBridge()

	// Create rule manager
	ruleManager := bridge.NewEventRuleManager(bridge)

	// Test valid rule
	validRule := &bridge.ResponseRule{
		ID:        "test_valid_rule",
		Name:      "Test Valid Rule",
		EventType: bridge.EventTypePerformance,
		Conditions: []bridge.RuleCondition{
			{Field: "type", Operator: "eq", Value: "test"},
		},
		Actions: []bridge.ResponseAction{
			{
				Type:   bridge.ActionTypeSystem,
				Action: "test_action",
				Parameters: map[string]interface{}{
					"test": "value",
				},
				Timeout:    time.Minute,
				MaxRetries: 1,
			},
		},
		Enabled:  true,
		Priority: 5,
		Cooldown: time.Minute,
	}

	// Validate the valid rule
	err := ruleManager.ValidateRule(validRule)
	if err != nil {
		fmt.Printf("❌ Valid rule validation failed: %v\n", err)
	} else {
		fmt.Printf("✅ Valid rule validation passed\n")
	}

	// Test invalid rule (missing action)
	invalidRule := &bridge.ResponseRule{
		ID:        "test_invalid_rule",
		Name:      "Test Invalid Rule",
		EventType: bridge.EventTypePerformance,
		Conditions: []bridge.RuleCondition{
			{Field: "type", Operator: "eq", Value: "test"},
		},
		Actions:  []bridge.ResponseAction{}, // Empty actions
		Enabled:  true,
		Priority: 5,
		Cooldown: time.Minute,
	}

	// Validate the invalid rule
	err = ruleManager.ValidateRule(invalidRule)
	if err != nil {
		fmt.Printf("✅ Invalid rule validation correctly failed: %v\n", err)
	} else {
		fmt.Printf("❌ Invalid rule validation should have failed\n")
	}
}

// createTestBridge creates a test monitor response bridge
func createTestBridge() *bridge.MonitorResponseBridge {
	// Create mock configuration
	cfg := &config.Config{
		// Add minimal config needed for testing
	}

	// Create mock executor
	executor := &executor.RealtimeExecutor{}

	// Create mock metrics collector
	metrics := &monitor.MetricsCollector{}

	// Create and return bridge
	return bridge.NewMonitorResponseBridge(cfg, executor, metrics)
}
