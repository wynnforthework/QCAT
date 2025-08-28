package position

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPortfolioTheoryCalculations tests the portfolio theory calculations
func TestPortfolioTheoryCalculations(t *testing.T) {
	calc := NewPortfolioTheoryCalculator(0.02)
	
	// Test portfolio return calculation
	weights := map[string]float64{
		"BTCUSDT": 0.6,
		"ETHUSDT": 0.4,
	}
	
	expectedReturns := map[string]float64{
		"BTCUSDT": 0.10,
		"ETHUSDT": 0.08,
	}
	
	portfolioReturn := calc.CalculatePortfolioReturn(weights, expectedReturns)
	assert.InDelta(t, 0.092, portfolioReturn, 0.001)
	
	// Test Sharpe ratio calculation
	sharpeRatio := calc.CalculateSharpeRatio(0.10, 0.15)
	assert.InDelta(t, 0.533, sharpeRatio, 0.01)
}

// TestRebalanceInstructionGeneration tests rebalance instruction generation
func TestRebalanceInstructionGeneration(t *testing.T) {
	// Test basic instruction creation
	instruction := RebalanceInstruction{
		ID:          "test_instruction",
		Symbol:      "BTCUSDT",
		CurrentSize: 1.0,
		TargetSize:  1.5,
		Adjustment:  0.5,
		Priority:    1,
		Status:      "PENDING",
	}
	
	assert.Equal(t, "test_instruction", instruction.ID)
	assert.Equal(t, "BTCUSDT", instruction.Symbol)
	assert.Equal(t, 0.5, instruction.Adjustment)
	assert.Equal(t, "PENDING", instruction.Status)
}

// TestOptimizationMetrics tests optimization performance metrics
func TestOptimizationMetrics(t *testing.T) {
	metrics := OptimizationPerformanceMetrics{
		PreOptimizationValue:  100000.0,
		PostOptimizationValue: 105000.0,
		ReturnImprovement:     0.05,
		PreOptimizationRisk:   0.20,
		PostOptimizationRisk:  0.18,
		RiskReduction:         0.02,
	}
	
	assert.Equal(t, 100000.0, metrics.PreOptimizationValue)
	assert.Equal(t, 105000.0, metrics.PostOptimizationValue)
	assert.Equal(t, 0.05, metrics.ReturnImprovement)
	assert.Equal(t, 0.02, metrics.RiskReduction)
}

// TestTransactionCostModel tests transaction cost modeling
func TestTransactionCostModel(t *testing.T) {
	model := TransactionCostModel{
		Symbol:          "BTCUSDT",
		BaseFee:         0.001,
		MarketImpactRate: 0.0001,
		BidAskSpread:    0.0002,
		LiquidityFactor: 0.5,
		VolatilityFactor: 0.3,
		SizeFactor:      0.1,
	}
	
	assert.Equal(t, "BTCUSDT", model.Symbol)
	assert.Equal(t, 0.001, model.BaseFee)
	assert.Equal(t, 0.0001, model.MarketImpactRate)
}

// TestPerformanceModels tests performance tracking models
func TestPerformanceModels(t *testing.T) {
	realTimeMetrics := RealTimePerformanceMetrics{
		TotalValue:    100000.0,
		UnrealizedPnL: 5000.0,
		RealizedPnL:   2000.0,
		TotalPnL:      7000.0,
		DailyReturn:   0.02,
		PositionCount: 5,
		Leverage:      2.0,
		MarginUsage:   0.6,
	}
	
	assert.Equal(t, 100000.0, realTimeMetrics.TotalValue)
	assert.Equal(t, 7000.0, realTimeMetrics.TotalPnL)
	assert.Equal(t, 5, realTimeMetrics.PositionCount)
}

// TestExecutionResult tests execution result structures
func TestExecutionResult(t *testing.T) {
	result := ExecutionResult{
		ExecutionID:        "exec_123",
		Success:            true,
		CompletedTrades:    5,
		FailedTrades:       0,
		TotalCost:          150.0,
		PreExecutionValue:  100000.0,
		PostExecutionValue: 105000.0,
	}
	
	assert.Equal(t, "exec_123", result.ExecutionID)
	assert.True(t, result.Success)
	assert.Equal(t, 5, result.CompletedTrades)
	assert.Equal(t, 0, result.FailedTrades)
	assert.Equal(t, 150.0, result.TotalCost)
}

// TestOptimizationConstraints tests optimization constraint validation
func TestOptimizationConstraints(t *testing.T) {
	constraints := PortfolioConstraints{
		MaxPositions:      10,
		MinPositionSize:   0.01,
		MaxPositionSize:   0.3,
		MaxCorrelation:    0.8,
		MinDiversification: 0.2,
		TurnoverLimit:     0.5,
		LeverageLimit:     3.0,
	}
	
	assert.Equal(t, 10, constraints.MaxPositions)
	assert.Equal(t, 0.01, constraints.MinPositionSize)
	assert.Equal(t, 0.3, constraints.MaxPositionSize)
	assert.Equal(t, 3.0, constraints.LeverageLimit)
}

// Benchmark test for portfolio calculations
func BenchmarkPortfolioCalculations(b *testing.B) {
	calc := NewPortfolioTheoryCalculator(0.02)
	
	weights := map[string]float64{
		"BTCUSDT": 0.3,
		"ETHUSDT": 0.3,
		"ADAUSDT": 0.2,
		"DOTUSDT": 0.2,
	}
	
	expectedReturns := map[string]float64{
		"BTCUSDT": 0.10,
		"ETHUSDT": 0.08,
		"ADAUSDT": 0.12,
		"DOTUSDT": 0.09,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calc.CalculatePortfolioReturn(weights, expectedReturns)
	}
}