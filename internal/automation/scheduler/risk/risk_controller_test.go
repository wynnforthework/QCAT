package risk

import (
	"context"
	"fmt"
	"testing"

	"qcat/internal/automation/scheduler/shared"

	"github.com/stretchr/testify/assert"
)

func TestNewRiskController(t *testing.T) {
	rc := NewTestRiskController()

	assert.NotNil(t, rc)
	assert.NotNil(t, rc.RiskController)
	assert.NotNil(t, rc.config)
	assert.NotNil(t, rc.riskMonitor)
	assert.NotNil(t, rc.errorHandler)
	assert.False(t, rc.emergencyMode)
	assert.Empty(t, rc.actionHistory)
}

func TestRiskController_TriggerPositionReduction(t *testing.T) {
	mockRC := NewTestRiskController()
	ctx := context.Background()

	// Create test margin status
	marginStatus := CreateTestMarginStatus(100000.0, 85000.0)

	// Store original positions for comparison
	originalPositions := make([]shared.Position, len(mockRC.testDB.positions))
	copy(originalPositions, mockRC.testDB.positions)
	
	// Call the mocked version by creating a custom implementation
	action, err := mockRC.triggerPositionReductionMocked(ctx, marginStatus, 0.3)

	assert.NoError(t, err)
	assert.NotNil(t, action)
	assert.Equal(t, ActionTypePositionReduction, action.Type)
	assert.NotEmpty(t, action.ID)
	assert.True(t, action.Result.Success)
	assert.NotEmpty(t, action.Result.AffectedPositions)

	// Check that action was recorded
	history := mockRC.GetActionHistory()
	assert.Len(t, history, 1)
	assert.Equal(t, action.ID, history[0].ID)
}

func TestRiskController_TriggerEmergencyStop(t *testing.T) {
	mockRC := NewTestRiskController()
	ctx := context.Background()
	reason := "Critical margin ratio exceeded"

	action, err := mockRC.triggerEmergencyStopMocked(ctx, reason)

	assert.NoError(t, err)
	assert.NotNil(t, action)
	assert.Equal(t, ActionTypeEmergencyStop, action.Type)
	assert.Equal(t, reason, action.Trigger)
	assert.True(t, mockRC.IsEmergencyMode())
	assert.True(t, action.Result.Success)

	// Check that all positions were closed (size = 0)
	for _, pos := range mockRC.testDB.positions {
		assert.Equal(t, 0.0, pos.Size)
	}

	// Check that action was recorded
	history := mockRC.GetActionHistory()
	assert.Len(t, history, 1)
	assert.Equal(t, action.ID, history[0].ID)
}

func TestRiskController_TriggerLeverageReduction(t *testing.T) {
	mockRC := NewTestRiskController()
	ctx := context.Background()
	targetLeverage := 5.0

	action, err := mockRC.triggerLeverageReductionMocked(ctx, targetLeverage)

	assert.NoError(t, err)
	assert.NotNil(t, action)
	assert.Equal(t, ActionTypeLeverageReduction, action.Type)
	assert.Equal(t, targetLeverage, action.Parameters["target_leverage"])
	assert.True(t, action.Result.Success)
}

func TestRiskController_CalculateReducedPositionSize(t *testing.T) {
	mockRC := NewTestRiskController()

	tests := []struct {
		name           string
		position       shared.Position
		targetLeverage float64
		expectedSize   float64
		expectError    bool
	}{
		{
			name: "Reduce leverage from 10x to 5x",
			position: shared.Position{
				Size:     1.0,
				Leverage: 10.0,
			},
			targetLeverage: 5.0,
			expectedSize:   0.5, // 1.0 * (5.0 / 10.0)
			expectError:    false,
		},
		{
			name: "No reduction needed",
			position: shared.Position{
				Size:     1.0,
				Leverage: 3.0,
			},
			targetLeverage: 5.0,
			expectedSize:   1.0, // No change needed
			expectError:    false,
		},
		{
			name: "Invalid target leverage",
			position: shared.Position{
				Size:     1.0,
				Leverage: 10.0,
			},
			targetLeverage: 0.0,
			expectedSize:   0.0,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newSize, err := mockRC.calculateReducedPositionSize(tt.position, tt.targetLeverage)
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.InDelta(t, tt.expectedSize, newSize, 0.001)
			}
		})
	}
}

func TestRiskController_SelectPositionsForReduction(t *testing.T) {
	mockRC := NewTestRiskController()

	ctx := context.Background()

	positions := []shared.Position{
		{
			ID:           "pos1",
			Size:         1.0,
			CurrentPrice: 50000.0, // $50,000 value
			MarginUsed:   25000.0,
		},
		{
			ID:           "pos2",
			Size:         10.0,
			CurrentPrice: 3000.0, // $30,000 value
			MarginUsed:   15000.0,
		},
		{
			ID:           "pos3",
			Size:         100.0,
			CurrentPrice: 100.0, // $10,000 value
			MarginUsed:   5000.0,
		},
	}

	reductionPercent := 0.3 // 30% reduction

	reductions, err := mockRC.selectPositionsForReduction(ctx, positions, reductionPercent)

	assert.NoError(t, err)
	assert.NotEmpty(t, reductions)

	// Calculate total reduction value
	totalReductionValue := 0.0
	for _, reduction := range reductions {
		totalReductionValue += reduction.ReductionAmount * positions[0].CurrentPrice // Simplified
	}

	// Should be approximately 30% of total portfolio value
	totalPortfolioValue := 50000.0 + 30000.0 + 10000.0 // $90,000
	expectedReduction := totalPortfolioValue * reductionPercent // $27,000

	// Allow some tolerance due to rounding and selection logic
	assert.InDelta(t, expectedReduction, totalReductionValue, expectedReduction*0.2) // 20% tolerance
}

func TestRiskController_EmergencyMode(t *testing.T) {
	mockRC := NewTestRiskController()

	// Initially not in emergency mode
	assert.False(t, mockRC.IsEmergencyMode())

	// Trigger emergency stop to activate emergency mode
	ctx := context.Background()

	_, err := mockRC.triggerEmergencyStopMocked(ctx, "Test emergency")
	assert.NoError(t, err)

	// Should now be in emergency mode
	assert.True(t, mockRC.IsEmergencyMode())

	// Clear emergency mode
	mockRC.ClearEmergencyMode()
	assert.False(t, mockRC.IsEmergencyMode())
}

func TestRiskController_StartStop(t *testing.T) {
	mockRC := NewTestRiskController()

	// Start
	err := mockRC.Start()
	assert.NoError(t, err)

	// Stop
	err = mockRC.Stop()
	assert.NoError(t, err)
}

func TestRiskActionType_String(t *testing.T) {
	tests := []struct {
		actionType RiskActionType
		expected   string
	}{
		{ActionTypePositionReduction, "POSITION_REDUCTION"},
		{ActionTypeEmergencyStop, "EMERGENCY_STOP"},
		{ActionTypeMarginIncrease, "MARGIN_INCREASE"},
		{ActionTypeHedgeActivation, "HEDGE_ACTIVATION"},
		{ActionTypeLeverageReduction, "LEVERAGE_REDUCTION"},
		{ActionTypeCircuitBreaker, "CIRCUIT_BREAKER"},
		{RiskActionType(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.actionType.String())
		})
	}
}



// Benchmark tests
func BenchmarkRiskController_SelectPositionsForReduction(b *testing.B) {
	mockRC := NewTestRiskController()

	// Create test positions
	positions := make([]shared.Position, 100)
	for i := range positions {
		positions[i] = shared.Position{
			ID:           fmt.Sprintf("pos%d", i),
			Size:         float64(i + 1),
			CurrentPrice: 1000.0 + float64(i)*10,
			MarginUsed:   float64(i+1) * 500,
		}
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := mockRC.selectPositionsForReduction(ctx, positions, 0.3)
		if err != nil {
			b.Fatal(err)
		}
	}
}