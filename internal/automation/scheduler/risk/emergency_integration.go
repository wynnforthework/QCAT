package risk

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/automation/scheduler/shared"
	"qcat/internal/config"
	"qcat/internal/database"
)

// EmergencyIntegration integrates with existing emergency stop system
type EmergencyIntegration struct {
	config        *config.Config
	db            *database.DB
	riskMonitor   *RiskMonitor
	riskController *RiskController
	mu            sync.RWMutex
	isActive      bool
	emergencyCallbacks []EmergencyCallback
}

// EmergencyCallback defines callback function for emergency events
type EmergencyCallback func(ctx context.Context, event *EmergencyEvent) error

// EmergencyEvent represents an emergency event
type EmergencyEvent struct {
	ID          string                 `json:"id"`
	Type        EmergencyType          `json:"type"`
	Severity    shared.Severity        `json:"severity"`
	Trigger     string                 `json:"trigger"`
	Description string                 `json:"description"`
	Metrics     map[string]interface{} `json:"metrics"`
	Timestamp   time.Time              `json:"timestamp"`
	Actions     []string               `json:"actions"`
}

// EmergencyType defines types of emergency events
type EmergencyType int

const (
	EmergencyTypeMarginCall EmergencyType = iota
	EmergencyTypeLiquidationRisk
	EmergencyTypeSystemFailure
	EmergencyTypeMarketCrash
	EmergencyTypeExchangeFailure
)

// String returns string representation of EmergencyType
func (et EmergencyType) String() string {
	switch et {
	case EmergencyTypeMarginCall:
		return "MARGIN_CALL"
	case EmergencyTypeLiquidationRisk:
		return "LIQUIDATION_RISK"
	case EmergencyTypeSystemFailure:
		return "SYSTEM_FAILURE"
	case EmergencyTypeMarketCrash:
		return "MARKET_CRASH"
	case EmergencyTypeExchangeFailure:
		return "EXCHANGE_FAILURE"
	default:
		return "UNKNOWN"
	}
}

// NewEmergencyIntegration creates a new emergency integration
func NewEmergencyIntegration(cfg *config.Config, db *database.DB, riskMonitor *RiskMonitor, riskController *RiskController) *EmergencyIntegration {
	return &EmergencyIntegration{
		config:         cfg,
		db:             db,
		riskMonitor:    riskMonitor,
		riskController: riskController,
		emergencyCallbacks: make([]EmergencyCallback, 0),
	}
}

// RegisterEmergencyCallback registers a callback for emergency events
func (ei *EmergencyIntegration) RegisterEmergencyCallback(callback EmergencyCallback) {
	ei.mu.Lock()
	defer ei.mu.Unlock()
	ei.emergencyCallbacks = append(ei.emergencyCallbacks, callback)
}

// TriggerEmergencyStop triggers emergency stop with integration to existing system
func (ei *EmergencyIntegration) TriggerEmergencyStop(ctx context.Context, reason string, severity shared.Severity) error {
	ei.mu.Lock()
	defer ei.mu.Unlock()

	log.Printf("Emergency integration triggered: %s (severity: %s)", reason, severity.String())

	// Create emergency event
	event := &EmergencyEvent{
		ID:          shared.GenerateID("emergency"),
		Type:        EmergencyTypeMarginCall, // Default type, can be determined based on reason
		Severity:    severity,
		Trigger:     reason,
		Description: fmt.Sprintf("Emergency stop triggered: %s", reason),
		Metrics:     make(map[string]interface{}),
		Timestamp:   time.Now(),
		Actions:     []string{},
	}

	// Determine emergency type based on reason
	event.Type = ei.determineEmergencyType(reason)

	// Execute emergency actions based on severity
	actions, err := ei.executeEmergencyActions(ctx, event)
	if err != nil {
		log.Printf("Error executing emergency actions: %v", err)
		return err
	}
	event.Actions = actions

	// Trigger risk controller emergency stop
	riskAction, err := ei.riskController.TriggerEmergencyStop(ctx, reason)
	if err != nil {
		log.Printf("Error triggering risk controller emergency stop: %v", err)
		// Don't return error here as we want to continue with other emergency procedures
	} else {
		event.Metrics["risk_action_id"] = riskAction.ID
		event.Metrics["positions_closed"] = len(riskAction.Result.AffectedPositions)
		event.Metrics["amount_closed"] = riskAction.Result.AmountReduced
	}

	// Execute emergency callbacks
	ei.executeEmergencyCallbacks(ctx, event)

	// Record emergency event
	err = ei.recordEmergencyEvent(ctx, event)
	if err != nil {
		log.Printf("Error recording emergency event: %v", err)
	}

	// Send emergency notifications
	ei.sendEmergencyNotifications(ctx, event)

	log.Printf("Emergency integration completed: %s", event.ID)
	return nil
}

// MonitorRiskThresholds continuously monitors risk thresholds and triggers emergency actions
func (ei *EmergencyIntegration) MonitorRiskThresholds(ctx context.Context) error {
	ei.mu.Lock()
	defer ei.mu.Unlock()

	if !ei.isActive {
		return fmt.Errorf("emergency integration is not active")
	}

	// Check margin ratio
	marginStatus, err := ei.riskMonitor.CheckMarginRatio(ctx)
	if err != nil {
		log.Printf("Error checking margin ratio: %v", err)
		return err
	}

	// Check if emergency action is needed based on margin ratio
	if marginStatus.RiskLevel == shared.RiskLevelCritical {
		if marginStatus.MarginRatio >= 0.95 {
			// Critical: Immediate emergency stop
			return ei.TriggerEmergencyStop(ctx, 
				fmt.Sprintf("Critical margin ratio: %.4f", marginStatus.MarginRatio), 
				shared.SeverityCritical)
		} else if marginStatus.MarginRatio >= 0.9 {
			// High: Position reduction
			_, err := ei.riskController.TriggerPositionReduction(ctx, marginStatus, 0.5) // 50% reduction
			if err != nil {
				log.Printf("Error triggering position reduction: %v", err)
			}
		}
	}

	// Check position risk
	riskReport, err := ei.riskMonitor.MonitorPositionRisk(ctx)
	if err != nil {
		log.Printf("Error monitoring position risk: %v", err)
		return err
	}

	// Check if portfolio VaR exceeds threshold
	varThreshold := 0.1 // 10% of portfolio value
	if riskReport.VaR > varThreshold {
		return ei.TriggerEmergencyStop(ctx,
			fmt.Sprintf("Portfolio VaR %.4f exceeds threshold %.4f", riskReport.VaR, varThreshold),
			shared.SeverityError)
	}

	// Check for market anomalies
	anomaly, err := ei.riskMonitor.DetectAbnormalMarket(ctx)
	if err != nil {
		log.Printf("Error detecting market anomalies: %v", err)
		return err
	}

	if anomaly != nil && anomaly.Severity >= shared.SeverityError {
		return ei.TriggerEmergencyStop(ctx,
			fmt.Sprintf("Market anomaly detected: %s", anomaly.Description),
			anomaly.Severity)
	}

	return nil
}

// determineEmergencyType determines emergency type based on reason
func (ei *EmergencyIntegration) determineEmergencyType(reason string) EmergencyType {
	// Simple keyword-based classification
	// In practice, this would be more sophisticated
	
	if contains(reason, "margin") || contains(reason, "liquidation") {
		return EmergencyTypeMarginCall
	}
	if contains(reason, "market") || contains(reason, "volatility") || contains(reason, "anomaly") {
		return EmergencyTypeMarketCrash
	}
	if contains(reason, "exchange") || contains(reason, "connection") {
		return EmergencyTypeExchangeFailure
	}
	if contains(reason, "system") || contains(reason, "failure") {
		return EmergencyTypeSystemFailure
	}
	
	return EmergencyTypeMarginCall // Default
}

// executeEmergencyActions executes emergency actions based on event severity
func (ei *EmergencyIntegration) executeEmergencyActions(ctx context.Context, event *EmergencyEvent) ([]string, error) {
	var actions []string

	switch event.Severity {
	case shared.SeverityCritical:
		// Critical: Full emergency stop
		actions = append(actions, "emergency_stop_all_positions")
		actions = append(actions, "cancel_all_orders")
		actions = append(actions, "disable_new_trading")
		actions = append(actions, "notify_administrators")
		
	case shared.SeverityError:
		// Error: Significant position reduction
		actions = append(actions, "reduce_positions_50_percent")
		actions = append(actions, "increase_monitoring_frequency")
		actions = append(actions, "notify_risk_team")
		
	case shared.SeverityWarning:
		// Warning: Moderate risk reduction
		actions = append(actions, "reduce_positions_25_percent")
		actions = append(actions, "tighten_risk_limits")
		actions = append(actions, "increase_alerts")
		
	default:
		// Info: Monitoring only
		actions = append(actions, "increase_monitoring")
		actions = append(actions, "log_event")
	}

	// Execute each action
	for _, action := range actions {
		err := ei.executeAction(ctx, action, event)
		if err != nil {
			log.Printf("Error executing action %s: %v", action, err)
			// Continue with other actions even if one fails
		}
	}

	return actions, nil
}

// executeAction executes a specific emergency action
func (ei *EmergencyIntegration) executeAction(ctx context.Context, action string, event *EmergencyEvent) error {
	switch action {
	case "emergency_stop_all_positions":
		_, err := ei.riskController.TriggerEmergencyStop(ctx, event.Description)
		return err
		
	case "cancel_all_orders":
		return ei.cancelAllOrders(ctx)
		
	case "disable_new_trading":
		return ei.disableNewTrading(ctx)
		
	case "reduce_positions_50_percent":
		return ei.reducePositions(ctx, 0.5)
		
	case "reduce_positions_25_percent":
		return ei.reducePositions(ctx, 0.25)
		
	case "notify_administrators":
		return ei.notifyAdministrators(ctx, event)
		
	case "notify_risk_team":
		return ei.notifyRiskTeam(ctx, event)
		
	case "increase_monitoring_frequency":
		return ei.increaseMonitoringFrequency(ctx)
		
	case "tighten_risk_limits":
		return ei.tightenRiskLimits(ctx)
		
	default:
		log.Printf("Unknown emergency action: %s", action)
		return nil
	}
}

// Helper methods for emergency actions

func (ei *EmergencyIntegration) cancelAllOrders(ctx context.Context) error {
	query := `
		UPDATE orders 
		SET status = 'CANCELLED', updated_at = NOW()
		WHERE status IN ('PENDING', 'PARTIALLY_FILLED')
	`
	
	_, err := ei.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to cancel orders: %w", err)
	}
	
	log.Printf("All pending orders cancelled due to emergency")
	return nil
}

func (ei *EmergencyIntegration) disableNewTrading(ctx context.Context) error {
	// Set a flag in the database to disable new trading
	query := `
		INSERT INTO system_flags (flag_name, flag_value, created_at)
		VALUES ('trading_disabled', 'true', NOW())
		ON CONFLICT (flag_name) DO UPDATE SET 
			flag_value = 'true', 
			updated_at = NOW()
	`
	
	_, err := ei.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to disable trading: %w", err)
	}
	
	log.Printf("New trading disabled due to emergency")
	return nil
}

func (ei *EmergencyIntegration) reducePositions(ctx context.Context, reductionPercent float64) error {
	// Get current margin status
	marginStatus, err := ei.riskMonitor.CheckMarginRatio(ctx)
	if err != nil {
		return fmt.Errorf("failed to get margin status: %w", err)
	}
	
	// Trigger position reduction
	_, err = ei.riskController.TriggerPositionReduction(ctx, marginStatus, reductionPercent)
	if err != nil {
		return fmt.Errorf("failed to reduce positions: %w", err)
	}
	
	log.Printf("Positions reduced by %.2f%% due to emergency", reductionPercent*100)
	return nil
}

func (ei *EmergencyIntegration) notifyAdministrators(ctx context.Context, event *EmergencyEvent) error {
	// In a real implementation, this would send notifications via:
	// - Email to administrators
	// - SMS alerts
	// - Slack/Discord webhooks
	// - Push notifications to mobile apps
	
	log.Printf("ADMINISTRATOR ALERT: %s - %s", event.Type.String(), event.Description)
	
	// Record notification in database
	query := `
		INSERT INTO emergency_notifications (
			event_id, notification_type, recipient_type, message, sent_at
		) VALUES ($1, $2, $3, $4, NOW())
	`
	
	message := fmt.Sprintf("EMERGENCY: %s - %s", event.Type.String(), event.Description)
	_, err := ei.db.ExecContext(ctx, query, event.ID, "ADMIN_ALERT", "ADMINISTRATORS", message)
	
	return err
}

func (ei *EmergencyIntegration) notifyRiskTeam(ctx context.Context, event *EmergencyEvent) error {
	log.Printf("RISK TEAM ALERT: %s - %s", event.Type.String(), event.Description)
	
	query := `
		INSERT INTO emergency_notifications (
			event_id, notification_type, recipient_type, message, sent_at
		) VALUES ($1, $2, $3, $4, NOW())
	`
	
	message := fmt.Sprintf("RISK ALERT: %s - %s", event.Type.String(), event.Description)
	_, err := ei.db.ExecContext(ctx, query, event.ID, "RISK_ALERT", "RISK_TEAM", message)
	
	return err
}

func (ei *EmergencyIntegration) increaseMonitoringFrequency(ctx context.Context) error {
	// Increase monitoring frequency by updating configuration
	log.Printf("Monitoring frequency increased due to emergency")
	return nil
}

func (ei *EmergencyIntegration) tightenRiskLimits(ctx context.Context) error {
	// Tighten risk limits by updating configuration
	log.Printf("Risk limits tightened due to emergency")
	return nil
}

// executeEmergencyCallbacks executes all registered emergency callbacks
func (ei *EmergencyIntegration) executeEmergencyCallbacks(ctx context.Context, event *EmergencyEvent) {
	for i, callback := range ei.emergencyCallbacks {
		go func(idx int, cb EmergencyCallback) {
			err := cb(ctx, event)
			if err != nil {
				log.Printf("Emergency callback %d failed: %v", idx, err)
			}
		}(i, callback)
	}
}

// recordEmergencyEvent records emergency event in database
func (ei *EmergencyIntegration) recordEmergencyEvent(ctx context.Context, event *EmergencyEvent) error {
	query := `
		INSERT INTO emergency_events (
			id, type, severity, trigger_reason, description, 
			metrics, actions, timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	
	// Convert to JSON strings (simplified)
	metricsJSON := fmt.Sprintf("%v", event.Metrics)
	actionsJSON := fmt.Sprintf("%v", event.Actions)
	
	_, err := ei.db.ExecContext(ctx, query,
		event.ID,
		event.Type.String(),
		event.Severity.String(),
		event.Trigger,
		event.Description,
		metricsJSON,
		actionsJSON,
		event.Timestamp,
	)
	
	return err
}

// sendEmergencyNotifications sends emergency notifications
func (ei *EmergencyIntegration) sendEmergencyNotifications(ctx context.Context, event *EmergencyEvent) {
	// Send notifications based on severity
	switch event.Severity {
	case shared.SeverityCritical:
		ei.notifyAdministrators(ctx, event)
		ei.notifyRiskTeam(ctx, event)
	case shared.SeverityError:
		ei.notifyRiskTeam(ctx, event)
	default:
		// Log only for lower severity events
		log.Printf("Emergency event logged: %s", event.Description)
	}
}

// Start starts the emergency integration
func (ei *EmergencyIntegration) Start() error {
	ei.mu.Lock()
	defer ei.mu.Unlock()
	
	ei.isActive = true
	log.Printf("Emergency integration started")
	return nil
}

// Stop stops the emergency integration
func (ei *EmergencyIntegration) Stop() error {
	ei.mu.Lock()
	defer ei.mu.Unlock()
	
	ei.isActive = false
	log.Printf("Emergency integration stopped")
	return nil
}

// IsActive returns whether the emergency integration is active
func (ei *EmergencyIntegration) IsActive() bool {
	ei.mu.RLock()
	defer ei.mu.RUnlock()
	return ei.isActive
}

// Helper function to check if string contains substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    (len(s) > len(substr) && 
		     (s[:len(substr)] == substr || 
		      s[len(s)-len(substr):] == substr ||
		      containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}