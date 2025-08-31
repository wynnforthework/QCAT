package risk

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"qcat/internal/automation/scheduler/shared"
	"qcat/internal/config"
	exch "qcat/internal/exchange"
)

// TestDB is a test implementation of database.DB
type TestDB struct {
	positions []shared.Position
	orders    []shared.Order
}

// NewTestDB creates a new test database
func NewTestDB() *TestDB {
	return &TestDB{
		positions: []shared.Position{
			{
				ID:           "pos1",
				Symbol:       "BTCUSDT",
				Side:         "LONG",
				Size:         1.0,
				EntryPrice:   50000.0,
				CurrentPrice: 50000.0,
				Leverage:     10.0,
				MarginUsed:   5000.0,
				Timestamp:    time.Now(),
			},
			{
				ID:           "pos2",
				Symbol:       "ETHUSDT",
				Side:         "LONG",
				Size:         10.0,
				EntryPrice:   3000.0,
				CurrentPrice: 3000.0,
				Leverage:     5.0,
				MarginUsed:   6000.0,
				Timestamp:    time.Now(),
			},
		},
		orders: make([]shared.Order, 0),
	}
}

// QueryContext mock implementation
func (tdb *TestDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	// For testing, we'll simulate returning positions
	// In a real implementation, this would parse the query and return appropriate mock data
	return nil, nil // This will be handled by the calling code
}

// ExecContext mock implementation
func (tdb *TestDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	// Simulate successful database operations
	return &TestResult{rowsAffected: 1}, nil
}

// TestResult implements sql.Result for testing
type TestResult struct {
	rowsAffected int64
}

func (tr *TestResult) LastInsertId() (int64, error) {
	return 0, nil
}

func (tr *TestResult) RowsAffected() (int64, error) {
	return tr.rowsAffected, nil
}

// TestAccountManager is a test implementation of account.Manager
type TestAccountManager struct {
	balances map[string]*exch.AccountBalance
}

// NewTestAccountManager creates a new test account manager
func NewTestAccountManager() *TestAccountManager {
	return &TestAccountManager{
		balances: make(map[string]*exch.AccountBalance),
	}
}

// GetAllBalances returns test balances
func (tam *TestAccountManager) GetAllBalances(ctx context.Context) (map[string]*exch.AccountBalance, error) {
	if len(tam.balances) == 0 {
		// Return default test balances
		return map[string]*exch.AccountBalance{
			"USDT": {
				Asset:     "USDT",
				Total:     100000.0,
				Available: 30000.0,
				Locked:    70000.0,
				UpdatedAt: time.Now(),
			},
		}, nil
	}
	return tam.balances, nil
}

// SetBalances sets test balances
func (tam *TestAccountManager) SetBalances(balances map[string]*exch.AccountBalance) {
	tam.balances = balances
}

// GetBalance returns balance for a specific asset
func (tam *TestAccountManager) GetBalance(ctx context.Context, asset string) (*exch.AccountBalance, error) {
	if balance, exists := tam.balances[asset]; exists {
		return balance, nil
	}
	return &exch.AccountBalance{
		Asset:     asset,
		Total:     0,
		Available: 0,
		Locked:    0,
		UpdatedAt: time.Now(),
	}, nil
}

// Helper functions for creating test instances

// NewTestRiskMonitor creates a RiskMonitor for testing
func NewTestRiskMonitor() *RiskMonitor {
	rm := &RiskMonitor{
		config:        &config.Config{},
		db:            nil, // We'll mock database operations
		configManager: shared.NewConfigManager(),
		errorHandler:  shared.NewErrorHandler(nil, nil),
		metrics:       make(map[string]interface{}),
		isRunning:     false,
	}

	return rm
}

// MockRiskController creates a RiskController for testing with mocked database operations
type MockRiskController struct {
	*RiskController
	testDB *TestDB
}

// Override getCurrentPositions for testing
func (mrc *MockRiskController) getCurrentPositions(ctx context.Context) ([]shared.Position, error) {
	return mrc.testDB.positions, nil
}

// Override executePositionReduction for testing
func (mrc *MockRiskController) executePositionReduction(ctx context.Context, reduction PositionReduction) error {
	// Find and update the position in test data
	for i, pos := range mrc.testDB.positions {
		if pos.ID == reduction.PositionID {
			mrc.testDB.positions[i].Size = reduction.NewSize
			break
		}
	}
	return nil
}

// Override executeEmergencyClose for testing
func (mrc *MockRiskController) executeEmergencyClose(ctx context.Context, position shared.Position) error {
	// Remove position from test data
	for i, pos := range mrc.testDB.positions {
		if pos.ID == position.ID {
			mrc.testDB.positions[i].Size = 0
			break
		}
	}
	return nil
}

// Override cancelAllPendingOrders for testing
func (mrc *MockRiskController) cancelAllPendingOrders(ctx context.Context) error {
	// Clear all orders in test data
	mrc.testDB.orders = make([]shared.Order, 0)
	return nil
}

// Override getHighLeveragePositions for testing
func (mrc *MockRiskController) getHighLeveragePositions(ctx context.Context, maxLeverage float64) ([]shared.Position, error) {
	var highLeveragePositions []shared.Position
	for _, pos := range mrc.testDB.positions {
		if pos.Leverage > maxLeverage {
			highLeveragePositions = append(highLeveragePositions, pos)
		}
	}
	return highLeveragePositions, nil
}

// Override recordActionInDatabase for testing
func (mrc *MockRiskController) recordActionInDatabase(ctx context.Context, action RiskAction) error {
	// In testing, we just return success
	return nil
}

// Override recordAction for testing to avoid database operations
func (mrc *MockRiskController) recordAction(action RiskAction) {
	// Add to memory only
	mrc.actionHistory = append(mrc.actionHistory, action)

	// Keep only last 100 actions in memory
	if len(mrc.actionHistory) > 100 {
		mrc.actionHistory = mrc.actionHistory[1:]
	}

	// Don't try to record in database for testing
}

// triggerPositionReductionMocked is a mocked version of TriggerPositionReduction for testing
func (mrc *MockRiskController) triggerPositionReductionMocked(ctx context.Context, marginStatus *MarginStatus, reductionPercent float64) (*RiskAction, error) {
	mrc.mu.Lock()
	defer mrc.mu.Unlock()

	startTime := time.Now()
	action := &RiskAction{
		ID:          shared.GenerateID("risk_action"),
		Type:        ActionTypePositionReduction,
		Trigger:     fmt.Sprintf("Margin ratio %.4f exceeds threshold", marginStatus.MarginRatio),
		Description: fmt.Sprintf("Reduce positions by %.2f%% due to high margin usage", reductionPercent*100),
		Parameters: map[string]interface{}{
			"margin_ratio":      marginStatus.MarginRatio,
			"reduction_percent": reductionPercent,
			"trigger_threshold": 0.8,
		},
		ExecutedAt: startTime,
	}

	// Get positions from mock data
	positions := mrc.testDB.positions

	// Calculate positions to reduce
	positionsToReduce, err := mrc.selectPositionsForReduction(ctx, positions, reductionPercent)
	if err != nil {
		action.Result = ActionResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to select positions for reduction: %v", err),
		}
		action.Duration = time.Since(startTime)
		mrc.recordAction(*action)
		return action, err
	}

	// Execute position reductions using mock
	var affectedPositions []string
	var totalReduced float64

	for _, reduction := range positionsToReduce {
		err := mrc.executePositionReduction(ctx, reduction)
		if err != nil {
			continue
		}

		affectedPositions = append(affectedPositions, reduction.PositionID)
		totalReduced += reduction.ReductionAmount
	}

	// Prepare successful result
	action.Result = ActionResult{
		Success:           true,
		AffectedPositions: affectedPositions,
		AmountReduced:     totalReduced,
		NewRiskLevel:      shared.RiskLevelMedium,
		Metrics: map[string]interface{}{
			"positions_targeted":  len(positionsToReduce),
			"positions_reduced":   len(affectedPositions),
			"total_value_reduced": totalReduced,
		},
	}
	action.Duration = time.Since(startTime)

	// Record action
	mrc.recordAction(*action)
	mrc.lastAction = time.Now()

	return action, nil
}

// triggerEmergencyStopMocked is a mocked version of TriggerEmergencyStop for testing
func (mrc *MockRiskController) triggerEmergencyStopMocked(ctx context.Context, reason string) (*RiskAction, error) {
	mrc.mu.Lock()
	defer mrc.mu.Unlock()

	startTime := time.Now()
	action := &RiskAction{
		ID:          shared.GenerateID("emergency_stop"),
		Type:        ActionTypeEmergencyStop,
		Trigger:     reason,
		Description: "Emergency stop - close all positions immediately",
		Parameters: map[string]interface{}{
			"emergency_reason": reason,
			"stop_time":        startTime,
		},
		ExecutedAt: startTime,
	}

	// Set emergency mode
	mrc.emergencyMode = true

	// Get all active positions from mock data
	positions := mrc.testDB.positions

	// Close all positions using mock
	var affectedPositions []string
	var totalClosed float64

	for _, position := range positions {
		err := mrc.executeEmergencyClose(ctx, position)
		if err != nil {
			continue
		}

		affectedPositions = append(affectedPositions, position.ID)
		totalClosed += position.Size * position.CurrentPrice
	}

	// Cancel all pending orders using mock
	_ = mrc.cancelAllPendingOrders(ctx)

	// Prepare successful result
	action.Result = ActionResult{
		Success:           true,
		AffectedPositions: affectedPositions,
		AmountReduced:     totalClosed,
		NewRiskLevel:      shared.RiskLevelLow,
		Metrics: map[string]interface{}{
			"total_positions":    len(positions),
			"positions_closed":   len(affectedPositions),
			"total_value_closed": totalClosed,
			"emergency_mode":     true,
		},
	}
	action.Duration = time.Since(startTime)

	// Record action
	mrc.recordAction(*action)
	mrc.lastAction = time.Now()

	return action, nil
}

// triggerLeverageReductionMocked is a mocked version of TriggerLeverageReduction for testing
func (mrc *MockRiskController) triggerLeverageReductionMocked(ctx context.Context, targetLeverage float64) (*RiskAction, error) {
	mrc.mu.Lock()
	defer mrc.mu.Unlock()

	startTime := time.Now()
	action := &RiskAction{
		ID:          shared.GenerateID("leverage_reduction"),
		Type:        ActionTypeLeverageReduction,
		Trigger:     "High risk conditions detected",
		Description: fmt.Sprintf("Reduce leverage to %.2fx", targetLeverage),
		Parameters: map[string]interface{}{
			"target_leverage": targetLeverage,
		},
		ExecutedAt: startTime,
	}

	// Get positions with high leverage from mock data
	positions, err := mrc.getHighLeveragePositions(ctx, targetLeverage)
	if err != nil {
		action.Result = ActionResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to get high leverage positions: %v", err),
		}
		action.Duration = time.Since(startTime)
		mrc.recordAction(*action)
		return action, err
	}

	// Reduce leverage for each position using mock
	var affectedPositions []string
	var totalReduced float64

	for _, position := range positions {
		newSize, err := mrc.calculateReducedPositionSize(position, targetLeverage)
		if err != nil {
			continue
		}

		reduction := PositionReduction{
			PositionID:       position.ID,
			CurrentSize:      position.Size,
			NewSize:          newSize,
			ReductionAmount:  position.Size - newSize,
			ReductionPercent: (position.Size - newSize) / position.Size,
		}

		err = mrc.executePositionReduction(ctx, reduction)
		if err != nil {
			continue
		}

		affectedPositions = append(affectedPositions, position.ID)
		totalReduced += reduction.ReductionAmount * position.CurrentPrice
	}

	// Prepare successful result
	action.Result = ActionResult{
		Success:           true,
		AffectedPositions: affectedPositions,
		AmountReduced:     totalReduced,
		NewRiskLevel:      shared.RiskLevelMedium,
		Metrics: map[string]interface{}{
			"target_leverage":     targetLeverage,
			"positions_targeted":  len(positions),
			"positions_reduced":   len(affectedPositions),
			"total_value_reduced": totalReduced,
		},
	}
	action.Duration = time.Since(startTime)

	// Record action
	mrc.recordAction(*action)
	mrc.lastAction = time.Now()

	return action, nil
}

// CreateTestPosition creates a test position
func CreateTestPosition(id, symbol string, size, price float64) shared.Position {
	return shared.Position{
		ID:           id,
		Symbol:       symbol,
		Side:         "LONG",
		Size:         size,
		EntryPrice:   price,
		CurrentPrice: price,
		Leverage:     1.0,
		MarginUsed:   size * price * 0.1, // 10% margin
		Timestamp:    time.Now(),
	}
}
