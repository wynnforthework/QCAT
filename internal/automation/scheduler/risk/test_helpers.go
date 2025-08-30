package risk

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"qcat/internal/automation/scheduler/shared"
	"qcat/internal/config"
	"qcat/internal/database"
)

// TestRiskController wraps RiskController with test-specific functionality
type TestRiskController struct {
	*RiskController
	testDB *TestDatabase
}

// TestDatabase provides mock database functionality for testing
type TestDatabase struct {
	positions []shared.Position
	orders    []shared.Order
}

// NewTestRiskController creates a RiskController instance for testing
func NewTestRiskController() *TestRiskController {
	// Create test configuration
	cfg := &config.Config{
		Risk: &config.RiskConfig{
			MaxLeverage:        10.0,
			MaxDrawdown:        0.2,
			EmergencyStopRatio: 0.85,
		},
	}

	// Create test database with sample data
	testDB := &TestDatabase{
		positions: []shared.Position{
			{
				ID:           "pos1",
				Symbol:       "BTCUSDT",
				Side:         "LONG",
				Size:         1.0,
				EntryPrice:   50000.0,
				CurrentPrice: 52000.0,
				Leverage:     10.0,
				MarginUsed:   5000.0,
				UnrealizedPnL: 2000.0,
			},
			{
				ID:           "pos2",
				Symbol:       "ETHUSDT",
				Side:         "SHORT",
				Size:         10.0,
				EntryPrice:   3000.0,
				CurrentPrice: 2900.0,
				Leverage:     5.0,
				MarginUsed:   6000.0,
				UnrealizedPnL: 1000.0,
			},
		},
		orders: []shared.Order{},
	}

	// Create mock database connection
	db := &database.DB{} // This would be properly mocked in real implementation

	// Create test risk monitor
	riskMonitor := &RiskMonitor{
		config: cfg,
		db:     db,
	}

	// Create the actual RiskController
	rc := NewRiskController(cfg, db, nil, riskMonitor)

	return &TestRiskController{
		RiskController: rc,
		testDB:         testDB,
	}
}

// triggerPositionReductionMocked provides a mock implementation for testing
func (trc *TestRiskController) triggerPositionReductionMocked(ctx context.Context, marginStatus *shared.MarginStatus, reductionRatio float64) (*RiskAction, error) {
	// Create a mock risk action
	action := &RiskAction{
		ID:        shared.GenerateID("risk_action"),
		Type:      ActionTypePositionReduction,
		Trigger:   fmt.Sprintf("Margin ratio: %.2f", marginStatus.MarginRatio),
		Timestamp: time.Now(),
		Parameters: map[string]interface{}{
			"reduction_ratio": reductionRatio,
			"margin_status":   marginStatus,
		},
		Result: &RiskActionResult{
			Success:           true,
			AffectedPositions: []string{"pos1", "pos2"},
			ExecutionTime:     time.Millisecond * 100,
			Details:           "Successfully reduced positions by 30%",
		},
	}

	// Simulate position reduction by modifying test positions
	for i := range trc.testDB.positions {
		trc.testDB.positions[i].Size *= (1.0 - reductionRatio)
		trc.testDB.positions[i].MarginUsed *= (1.0 - reductionRatio)
	}

	// Record the action
	trc.actionHistory = append(trc.actionHistory, *action)

	return action, nil
}

// triggerEmergencyStopMocked provides a mock implementation for testing
func (trc *TestRiskController) triggerEmergencyStopMocked(ctx context.Context, reason string) (*RiskAction, error) {
	// Create a mock risk action
	action := &RiskAction{
		ID:        shared.GenerateID("emergency_action"),
		Type:      ActionTypeEmergencyStop,
		Trigger:   reason,
		Timestamp: time.Now(),
		Parameters: map[string]interface{}{
			"reason": reason,
		},
		Result: &RiskActionResult{
			Success:           true,
			AffectedPositions: []string{"pos1", "pos2"},
			ExecutionTime:     time.Millisecond * 50,
			Details:           "Emergency stop executed successfully",
		},
	}

	// Simulate emergency stop by closing all positions
	for i := range trc.testDB.positions {
		trc.testDB.positions[i].Size = 0.0
		trc.testDB.positions[i].MarginUsed = 0.0
	}

	// Set emergency mode
	trc.emergencyMode = true

	// Record the action
	trc.actionHistory = append(trc.actionHistory, *action)

	return action, nil
}

// triggerLeverageReductionMocked provides a mock implementation for testing
func (trc *TestRiskController) triggerLeverageReductionMocked(ctx context.Context, targetLeverage float64) (*RiskAction, error) {
	// Create a mock risk action
	action := &RiskAction{
		ID:        shared.GenerateID("leverage_action"),
		Type:      ActionTypeLeverageReduction,
		Trigger:   fmt.Sprintf("Target leverage: %.1fx", targetLeverage),
		Timestamp: time.Now(),
		Parameters: map[string]interface{}{
			"target_leverage": targetLeverage,
		},
		Result: &RiskActionResult{
			Success:           true,
			AffectedPositions: []string{"pos1", "pos2"},
			ExecutionTime:     time.Millisecond * 75,
			Details:           fmt.Sprintf("Leverage reduced to %.1fx", targetLeverage),
		},
	}

	// Simulate leverage reduction by adjusting positions
	for i := range trc.testDB.positions {
		if trc.testDB.positions[i].Leverage > targetLeverage {
			// Calculate new position size based on target leverage
			newSize, _ := trc.calculateReducedPositionSize(trc.testDB.positions[i], targetLeverage)
			trc.testDB.positions[i].Size = newSize
			trc.testDB.positions[i].Leverage = targetLeverage
			trc.testDB.positions[i].MarginUsed = newSize * trc.testDB.positions[i].CurrentPrice / targetLeverage
		}
	}

	// Record the action
	trc.actionHistory = append(trc.actionHistory, *action)

	return action, nil
}

// GetActionHistory returns the action history for testing
func (trc *TestRiskController) GetActionHistory() []RiskAction {
	return trc.actionHistory
}

// IsEmergencyMode returns the emergency mode status
func (trc *TestRiskController) IsEmergencyMode() bool {
	return trc.emergencyMode
}

// ClearEmergencyMode clears the emergency mode for testing
func (trc *TestRiskController) ClearEmergencyMode() {
	trc.emergencyMode = false
}

// Start starts the risk controller (mock implementation)
func (trc *TestRiskController) Start() error {
	// Mock implementation - just return success
	return nil
}

// Stop stops the risk controller (mock implementation)
func (trc *TestRiskController) Stop() error {
	// Mock implementation - just return success
	return nil
}

// CreateTestMarginStatus creates a test margin status for testing
func CreateTestMarginStatus(totalEquity, usedMargin float64) *shared.MarginStatus {
	return &shared.MarginStatus{
		TotalEquity:    totalEquity,
		UsedMargin:     usedMargin,
		FreeMargin:     totalEquity - usedMargin,
		MarginRatio:    usedMargin / totalEquity,
		MarginLevel:    totalEquity / usedMargin,
		LastUpdated:    time.Now(),
	}
}

// selectPositionsForReduction is a helper method for testing
func (trc *TestRiskController) selectPositionsForReduction(ctx context.Context, positions []shared.Position, reductionRatio float64) ([]shared.Position, error) {
	// Simple selection logic for testing - select positions with highest margin usage
	var selectedPositions []shared.Position
	
	totalReductionNeeded := 0.0
	for _, pos := range positions {
		totalReductionNeeded += pos.MarginUsed * reductionRatio
	}
	
	currentReduction := 0.0
	for _, pos := range positions {
		if currentReduction < totalReductionNeeded {
			selectedPositions = append(selectedPositions, pos)
			currentReduction += pos.MarginUsed
		}
	}
	
	return selectedPositions, nil
}

// calculateReducedPositionSize calculates the new position size for leverage reduction
func (trc *TestRiskController) calculateReducedPositionSize(position shared.Position, targetLeverage float64) (float64, error) {
	if targetLeverage <= 0 {
		return 0, fmt.Errorf("invalid target leverage: %f", targetLeverage)
	}
	
	if position.Leverage <= targetLeverage {
		// No reduction needed
		return position.Size, nil
	}
	
	// Calculate new size based on leverage ratio
	leverageRatio := targetLeverage / position.Leverage
	newSize := position.Size * leverageRatio
	
	return newSize, nil
}
