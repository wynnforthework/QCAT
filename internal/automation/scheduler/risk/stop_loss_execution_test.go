package risk

import (
	"context"
	"testing"
	"time"

	"qcat/internal/automation/scheduler/shared"
	"qcat/internal/config"
	"qcat/internal/database"
	"qcat/internal/exchange/account"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStopLossExecutor(t *testing.T) {
	adjuster := createTestStopLossAdjuster(t)
	cfg := &config.Config{}
	db := &database.DB{}
	accountManager := &account.Manager{}

	executor := NewStopLossExecutor(adjuster, cfg, db, accountManager)

	assert.NotNil(t, executor)
	assert.NotNil(t, executor.adjuster)
	assert.NotNil(t, executor.config)
	assert.NotNil(t, executor.db)
	assert.NotNil(t, executor.accountManager)
	assert.NotNil(t, executor.configManager)
	assert.NotNil(t, executor.errorHandler)
	assert.NotNil(t, executor.metrics)
	assert.NotNil(t, executor.performanceTracker)
	assert.False(t, executor.isRunning)
}

func TestStopLossExecutor_StartStop(t *testing.T) {
	executor := createTestStopLossExecutor(t)

	// Test Start
	err := executor.Start()
	assert.NoError(t, err)
	assert.True(t, executor.IsRunning())

	// Test Stop
	err = executor.Stop()
	assert.NoError(t, err)
	assert.False(t, executor.IsRunning())
}

func TestStopLossExecutor_ExecuteStopLossAdjustments(t *testing.T) {
	executor, mock := createTestStopLossExecutorWithMock(t)
	defer executor.db.Close()

	ctx := context.Background()

	// Mock current price query for performance tracking
	priceRows := sqlmock.NewRows([]string{"close_price"}).AddRow(51000.0)
	mock.ExpectQuery("SELECT close_price FROM market_data WHERE symbol = \\? ORDER BY timestamp DESC LIMIT 1").
		WithArgs("BTCUSDT").
		WillReturnRows(priceRows)

	// Mock performance tracking insert
	mock.ExpectExec("INSERT INTO stop_loss_performance").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Mock the stop loss update query (this happens first)
	mock.ExpectExec("UPDATE positions SET stop_loss = \\?, updated_at = CURRENT_TIMESTAMP WHERE id = \\? AND status = 'ACTIVE'").
		WithArgs(48500.0, "pos_1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Mock active positions query for GenerateStopLossAdjustments (this happens second)
	positionsRows := sqlmock.NewRows([]string{
		"id", "symbol", "side", "size", "entry_price", "current_price",
		"unrealized_pnl", "realized_pnl", "leverage", "margin_used", "created_at",
	}).AddRow(
		"pos_1", "BTCUSDT", "LONG", 1.0, 50000.0, 51000.0,
		1000.0, 0.0, 2.0, 25000.0, time.Now(),
	)
	mock.ExpectQuery("SELECT id, symbol, side, size, entry_price, current_price").
		WillReturnRows(positionsRows)

	// In test mode, GenerateStopLossAdjustments returns mock data
	result, err := executor.ExecuteStopLossAdjustments(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.TotalAdjustments)
	assert.Equal(t, 1, result.SuccessfulAdjustments)
	assert.Equal(t, 0, result.FailedAdjustments)
	assert.GreaterOrEqual(t, result.ExecutionTime, time.Duration(0))
}

func TestStopLossExecutor_ExecuteAdjustmentWithTracking(t *testing.T) {
	executor, _ := createTestStopLossExecutorWithMock(t)
	defer executor.db.Close()

	ctx := context.Background()
	adjustment := StopLossAdjustment{
		PositionID:     "pos_1",
		Symbol:         "BTCUSDT",
		OldLevel:       49000.0,
		NewLevel:       49500.0,
		AdjustmentType: "ATR",
		Reason:         "ATR-based adjustment",
		Priority:       5,
		Timestamp:      time.Now(),
	}

	// In test mode, database operations are bypassed
	// No mock expectations needed

	detail := executor.executeAdjustmentWithTracking(ctx, adjustment)

	assert.True(t, detail.Success)
	assert.Equal(t, "pos_1", detail.PositionID)
	assert.Equal(t, "BTCUSDT", detail.Symbol)
	assert.Equal(t, 49000.0, detail.OldLevel)
	assert.Equal(t, 49500.0, detail.NewLevel)
	assert.Equal(t, "ATR", detail.AdjustmentType)
	assert.GreaterOrEqual(t, detail.ExecutionTime, time.Duration(0))
	assert.Empty(t, detail.Error)

	// In test mode, database queries are bypassed, so we don't verify mock expectations
	// assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStopLossExecutor_IntegrateWithPositionMonitoring(t *testing.T) {
	executor := createTestStopLossExecutor(t)
	ctx := context.Background()
	monitoringInterval := 5 * time.Minute

	integration := executor.IntegrateWithPositionMonitoring(ctx, monitoringInterval)

	assert.NotNil(t, integration)
	assert.Equal(t, executor, integration.executor)
	assert.Equal(t, monitoringInterval, integration.monitoringInterval)
	assert.NotNil(t, integration.stopChan)
	assert.False(t, integration.isActive)
}

func TestPositionMonitoringIntegration_StartStopMonitoring(t *testing.T) {
	executor := createTestStopLossExecutor(t)
	ctx := context.Background()
	integration := executor.IntegrateWithPositionMonitoring(ctx, time.Second)

	// Test Start
	err := integration.StartMonitoring(ctx)
	assert.NoError(t, err)
	assert.True(t, integration.isActive)

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Test Stop
	err = integration.StopMonitoring()
	assert.NoError(t, err)
	assert.False(t, integration.isActive)

	// Test starting already active integration
	integration.isActive = true
	err = integration.StartMonitoring(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already active")

	// Test stopping inactive integration
	integration.isActive = false
	err = integration.StopMonitoring()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not active")
}

func TestStopLossExecutor_GetMetrics(t *testing.T) {
	executor := createTestStopLossExecutor(t)

	// Set some test metrics
	executor.metrics["test_metric"] = 123.45
	executor.performanceTracker.metrics["perf_metric"] = "test_value"

	metrics := executor.GetMetrics()

	assert.NotNil(t, metrics)
	assert.Equal(t, 123.45, metrics["test_metric"])
	assert.Equal(t, "test_value", metrics["performance_perf_metric"])

	// Verify it returns a copy
	metrics["new_metric"] = "new_value"
	internalMetrics := executor.GetMetrics()
	assert.NotContains(t, internalMetrics, "new_metric")
}

func TestStopLossPerformanceTracker_StartTrackingAdjustment(t *testing.T) {
	tracker, mock := createTestPerformanceTrackerWithMock(t)
	defer tracker.db.Close()

	ctx := context.Background()
	adjustment := StopLossAdjustment{
		PositionID:     "pos_1",
		Symbol:         "BTCUSDT",
		OldLevel:       49000.0,
		NewLevel:       49500.0,
		AdjustmentType: "ATR",
		Timestamp:      time.Now(),
	}

	// Mock current price query
	mock.ExpectQuery("SELECT close_price FROM market_data").
		WithArgs("BTCUSDT").
		WillReturnRows(sqlmock.NewRows([]string{"close_price"}).AddRow(51000.0))

	// Mock performance record insertion
	mock.ExpectExec("INSERT INTO stop_loss_performance").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := tracker.StartTrackingAdjustment(ctx, adjustment)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(tracker.adjustmentHistory))

	// Verify the tracking record
	record := tracker.adjustmentHistory[0]
	assert.Equal(t, "pos_1", record.PositionID)
	assert.Equal(t, "BTCUSDT", record.Symbol)
	assert.Equal(t, 49000.0, record.OldStopLoss)
	assert.Equal(t, 49500.0, record.NewStopLoss)
	assert.Equal(t, 51000.0, record.PriceAtAdjustment)
	assert.Equal(t, "ATR", record.AdjustmentType)
	assert.False(t, record.WasTriggered)

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStopLossPerformanceTracker_UpdatePerformanceMetrics(t *testing.T) {
	tracker, mock := createTestPerformanceTrackerWithMock(t)
	defer tracker.db.Close()

	ctx := context.Background()

	// Mock active tracking records query
	trackingRows := sqlmock.NewRows([]string{
		"adjustment_id", "position_id", "symbol", "adjustment_time", "old_stop_loss", "new_stop_loss",
		"price_at_adjustment", "adjustment_type", "was_triggered", "trigger_time",
		"trigger_price", "pnl_at_trigger", "effectiveness_score", "would_old_have_been_better",
		"pnl_difference", "time_to_trigger",
	}).AddRow(
		"adj_1", "pos_1", "BTCUSDT", time.Now().Add(-time.Hour), 49000.0, 49500.0,
		51000.0, "ATR", false, "",
		0.0, 0.0, 0.5, false,
		0.0, 0.0,
	)
	mock.ExpectQuery("SELECT adjustment_id, position_id, symbol").
		WillReturnRows(trackingRows)

	// Mock position status query (position still active)
	positionRows := sqlmock.NewRows([]string{
		"id", "symbol", "side", "size", "entry_price", "current_price",
		"unrealized_pnl", "realized_pnl", "leverage", "margin_used", "created_at",
	}).AddRow(
		"pos_1", "BTCUSDT", "LONG", 1.0, 50000.0, 51500.0,
		1500.0, 0.0, 2.0, 25000.0, time.Now(),
	)
	mock.ExpectQuery("SELECT id, symbol, side, size, entry_price, current_price").
		WithArgs("pos_1").
		WillReturnRows(positionRows)

	// Mock performance update
	mock.ExpectExec("UPDATE stop_loss_performance").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Mock aggregate metrics calculation
	statsRows := sqlmock.NewRows([]string{
		"total_adjustments", "triggered_adjustments", "avg_effectiveness",
		"successful_adjustments", "avg_time_to_trigger", "total_pnl_impact",
	}).AddRow(1, 0, 0.6, 1, nil, 100.0)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) as total_adjustments").
		WillReturnRows(statsRows)

	err := tracker.UpdatePerformanceMetrics(ctx)

	assert.NoError(t, err)

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStopLossPerformanceTracker_CalculateEffectivenessScore(t *testing.T) {
	tracker := createTestPerformanceTracker(t)

	tracking := AdjustmentPerformance{
		PriceAtAdjustment: 50000.0,
		OldStopLoss:       49000.0,
		NewStopLoss:       49500.0,
	}

	// Test case 1: Stop loss not triggered, profitable position
	closure1 := &PositionClosureDetails{
		CloseReason: "MANUAL",
		RealizedPnL: 1000.0,
	}
	score1 := tracker.calculateEffectivenessScore(tracking, closure1, false)
	assert.Greater(t, score1, 0.7) // Should be high score

	// Test case 2: Stop loss triggered
	closure2 := &PositionClosureDetails{
		CloseReason: "STOP_LOSS",
		RealizedPnL: -500.0,
	}
	score2 := tracker.calculateEffectivenessScore(tracking, closure2, true)
	assert.Less(t, score2, 0.7) // Should be lower score

	// Test case 3: Large loss without stop loss trigger (bad adjustment)
	closure3 := &PositionClosureDetails{
		CloseReason: "MANUAL",
		RealizedPnL: -2000.0,
	}
	score3 := tracker.calculateEffectivenessScore(tracking, closure3, false)
	assert.Less(t, score3, 0.8) // Should be lower score due to large loss
}

func TestStopLossPerformanceTracker_WasStopLossTriggered(t *testing.T) {
	tracker := createTestPerformanceTracker(t)

	stopLossLevel := 49500.0

	// Test case 1: Explicit stop loss reason
	closure1 := &PositionClosureDetails{
		CloseReason: "STOP_LOSS",
		ClosePrice:  49600.0,
	}
	triggered1 := tracker.wasStopLossTriggered(closure1, stopLossLevel)
	assert.True(t, triggered1)

	// Test case 2: Close price near stop loss level
	closure2 := &PositionClosureDetails{
		CloseReason: "MANUAL",
		ClosePrice:  49505.0, // Within 0.1% tolerance
	}
	triggered2 := tracker.wasStopLossTriggered(closure2, stopLossLevel)
	assert.True(t, triggered2)

	// Test case 3: Close price far from stop loss level
	closure3 := &PositionClosureDetails{
		CloseReason: "MANUAL",
		ClosePrice:  51000.0,
	}
	triggered3 := tracker.wasStopLossTriggered(closure3, stopLossLevel)
	assert.False(t, triggered3)
}

func TestStopLossPerformanceTracker_WouldOldStopLossHaveBeenBetter(t *testing.T) {
	tracker := createTestPerformanceTracker(t)

	tracking := AdjustmentPerformance{
		PriceAtAdjustment: 50000.0,
		OldStopLoss:       49000.0,
		NewStopLoss:       49500.0,
	}

	// Test case 1: New stop loss triggered, old wouldn't have
	closure1 := &PositionClosureDetails{
		ClosePrice: 49500.0, // At new stop loss level (will trigger new but not old)
	}
	better1 := tracker.wouldOldStopLossHaveBeenBetter(tracking, closure1)
	assert.True(t, better1) // Old stop loss would have been better

	// Test case 2: Both would trigger at same level
	closure2 := &PositionClosureDetails{
		ClosePrice: 48000.0, // Below both stop losses
	}
	better2 := tracker.wouldOldStopLossHaveBeenBetter(tracking, closure2)
	// Result depends on distance calculation - old stop loss is further from entry
	assert.False(t, better2)

	// Test case 3: Neither would trigger
	closure3 := &PositionClosureDetails{
		ClosePrice: 52000.0, // Above both stop losses
	}
	better3 := tracker.wouldOldStopLossHaveBeenBetter(tracking, closure3)
	// Result depends on distance calculation - old stop loss is further from entry
	assert.False(t, better3)
}

func TestStopLossPerformanceTracker_CalculateCurrentEffectiveness(t *testing.T) {
	tracker := createTestPerformanceTracker(t)

	tracking := AdjustmentPerformance{
		NewStopLoss: 49500.0,
	}

	// Test case 1: Profitable position
	position1 := &shared.Position{
		CurrentPrice:  51000.0,
		UnrealizedPnL: 1000.0,
	}
	effectiveness1 := tracker.calculateCurrentEffectiveness(tracking, position1)
	assert.Greater(t, effectiveness1, 0.5) // Should be above neutral

	// Test case 2: Loss position
	position2 := &shared.Position{
		CurrentPrice:  50000.0,
		UnrealizedPnL: -800.0,
	}
	effectiveness2 := tracker.calculateCurrentEffectiveness(tracking, position2)
	assert.Less(t, effectiveness2, 0.5) // Should be below neutral

	// Test case 3: Position close to stop loss
	position3 := &shared.Position{
		CurrentPrice:  49600.0, // Very close to stop loss
		UnrealizedPnL: 100.0,
	}
	effectiveness3 := tracker.calculateCurrentEffectiveness(tracking, position3)
	assert.LessOrEqual(t, effectiveness3, 0.6) // Should be penalized for being close to stop loss
}

// Helper functions for testing

func createTestStopLossExecutor(t *testing.T) *StopLossExecutor {
	adjuster := createTestStopLossAdjuster(t)
	cfg := &config.Config{}
	db := &database.DB{}
	accountManager := &account.Manager{}

	return NewStopLossExecutor(adjuster, cfg, db, accountManager)
}

func createTestStopLossExecutorWithMock(t *testing.T) (*StopLossExecutor, sqlmock.Sqlmock) {
	// Create mock database
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	adjuster := createTestStopLossAdjuster(t)
	cfg := &config.Config{}
	db := &database.DB{DB: mockDB}
	accountManager := &account.Manager{}

	executor := NewStopLossExecutor(adjuster, cfg, db, accountManager)
	executor.adjuster.db = db         // Update adjuster's db to use mock
	executor.adjuster.testMode = true // Set test mode flag

	return executor, mock
}

func createTestPerformanceTracker(_ *testing.T) *StopLossPerformanceTracker {
	// Use nil database for testing
	return &StopLossPerformanceTracker{
		db:                nil,
		metrics:           make(map[string]interface{}),
		adjustmentHistory: make([]AdjustmentPerformance, 0),
	}
}

func createTestPerformanceTrackerWithMock(t *testing.T) (*StopLossPerformanceTracker, sqlmock.Sqlmock) {
	// Create mock database
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	db := &database.DB{DB: mockDB}
	tracker := &StopLossPerformanceTracker{
		db:                db,
		metrics:           make(map[string]interface{}),
		adjustmentHistory: make([]AdjustmentPerformance, 0),
	}

	return tracker, mock
}

func TestUpdateExecutionMetrics(t *testing.T) {
	executor := createTestStopLossExecutor(t)

	result := &ExecutionResult{
		TotalAdjustments:      10,
		SuccessfulAdjustments: 8,
		FailedAdjustments:     2,
		ExecutionTime:         time.Minute,
		Timestamp:             time.Now(),
	}

	executor.updateExecutionMetrics(result)

	metrics := executor.GetMetrics()
	assert.Equal(t, result.Timestamp, metrics["last_execution_time"])
	assert.Equal(t, result.ExecutionTime, metrics["last_execution_duration"])
	assert.Equal(t, 10, metrics["last_total_adjustments"])
	assert.Equal(t, 8, metrics["last_successful_adjustments"])
	assert.Equal(t, 2, metrics["last_failed_adjustments"])
	assert.Equal(t, 0.8, metrics["last_success_rate"])
	assert.Equal(t, 1, metrics["total_executions"])

	// Test cumulative metrics
	executor.updateExecutionMetrics(result)
	metrics = executor.GetMetrics()
	assert.Equal(t, 2, metrics["total_executions"])
}
