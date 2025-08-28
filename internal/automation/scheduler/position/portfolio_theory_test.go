package position

import (
	"math"
	"testing"

	"qcat/internal/automation/scheduler/shared"
	"github.com/stretchr/testify/assert"
)

func TestNewPortfolioTheoryCalculator(t *testing.T) {
	riskFreeRate := 0.02
	calc := NewPortfolioTheoryCalculator(riskFreeRate)
	
	assert.NotNil(t, calc)
	assert.Equal(t, riskFreeRate, calc.riskFreeRate)
}

func TestCalculatePortfolioReturn(t *testing.T) {
	calc := NewPortfolioTheoryCalculator(0.02)
	
	weights := map[string]float64{
		"BTCUSDT": 0.6,
		"ETHUSDT": 0.4,
	}
	
	expectedReturns := map[string]float64{
		"BTCUSDT": 0.10,
		"ETHUSDT": 0.08,
	}
	
	portfolioReturn := calc.CalculatePortfolioReturn(weights, expectedReturns)
	
	// Expected: 0.6 * 0.10 + 0.4 * 0.08 = 0.092
	assert.InDelta(t, 0.092, portfolioReturn, 0.001)
}

func TestCalculatePortfolioRisk(t *testing.T) {
	calc := NewPortfolioTheoryCalculator(0.02)
	
	weights := map[string]float64{
		"BTCUSDT": 0.6,
		"ETHUSDT": 0.4,
	}
	
	covarianceMatrix := map[string]map[string]float64{
		"BTCUSDT": {
			"BTCUSDT": 0.04,
			"ETHUSDT": 0.02,
		},
		"ETHUSDT": {
			"BTCUSDT": 0.02,
			"ETHUSDT": 0.03,
		},
	}
	
	portfolioRisk := calc.CalculatePortfolioRisk(weights, covarianceMatrix)
	
	// Should return a positive risk value
	assert.True(t, portfolioRisk > 0)
	assert.True(t, portfolioRisk < 1) // Reasonable risk level
}

func TestCalculateSharpeRatio(t *testing.T) {
	calc := NewPortfolioTheoryCalculator(0.02)
	
	portfolioReturn := 0.10
	portfolioRisk := 0.15
	
	sharpeRatio := calc.CalculateSharpeRatio(portfolioReturn, portfolioRisk)
	
	// Expected: (0.10 - 0.02) / 0.15 = 0.533...
	assert.InDelta(t, 0.533, sharpeRatio, 0.01)
}

func TestCalculateSharpeRatioZeroRisk(t *testing.T) {
	calc := NewPortfolioTheoryCalculator(0.02)
	
	sharpeRatio := calc.CalculateSharpeRatio(0.10, 0.0)
	
	assert.Equal(t, 0.0, sharpeRatio)
}

func TestCalculateInformationRatio(t *testing.T) {
	calc := NewPortfolioTheoryCalculator(0.02)
	
	portfolioReturn := 0.12
	benchmarkReturn := 0.10
	trackingError := 0.05
	
	infoRatio := calc.CalculateInformationRatio(portfolioReturn, benchmarkReturn, trackingError)
	
	// Expected: (0.12 - 0.10) / 0.05 = 0.4
	assert.InDelta(t, 0.4, infoRatio, 0.001)
}

func TestCalculateBeta(t *testing.T) {
	calc := NewPortfolioTheoryCalculator(0.02)
	
	portfolioReturns := []float64{0.01, 0.02, -0.01, 0.03, 0.00}
	marketReturns := []float64{0.015, 0.01, -0.005, 0.025, 0.005}
	
	beta := calc.CalculateBeta(portfolioReturns, marketReturns)
	
	// Should return a reasonable beta value
	assert.True(t, beta > 0)
	assert.True(t, beta < 5) // Reasonable range
}

func TestCalculateAlpha(t *testing.T) {
	calc := NewPortfolioTheoryCalculator(0.02)
	
	portfolioReturn := 0.12
	marketReturn := 0.10
	beta := 1.2
	
	alpha := calc.CalculateAlpha(portfolioReturn, marketReturn, beta)
	
	// Expected: 0.12 - (0.02 + 1.2 * (0.10 - 0.02)) = 0.12 - 0.116 = 0.004
	assert.InDelta(t, 0.004, alpha, 0.001)
}

func TestCalculateTrackingError(t *testing.T) {
	calc := NewPortfolioTheoryCalculator(0.02)
	
	portfolioReturns := []float64{0.01, 0.02, -0.01, 0.03}
	benchmarkReturns := []float64{0.015, 0.01, -0.005, 0.025}
	
	trackingError := calc.CalculateTrackingError(portfolioReturns, benchmarkReturns)
	
	// Should return a positive tracking error
	assert.True(t, trackingError >= 0)
}

func TestCalculateVaR(t *testing.T) {
	calc := NewPortfolioTheoryCalculator(0.02)
	
	portfolioReturn := 0.10
	portfolioRisk := 0.15
	confidenceLevel := 0.95
	
	var95 := calc.CalculateVaR(portfolioReturn, portfolioRisk, confidenceLevel)
	
	// VaR should be less than expected return
	assert.True(t, var95 < portfolioReturn)
}

func TestCalculateExpectedShortfall(t *testing.T) {
	calc := NewPortfolioTheoryCalculator(0.02)
	
	portfolioReturn := 0.10
	portfolioRisk := 0.15
	confidenceLevel := 0.95
	
	es := calc.CalculateExpectedShortfall(portfolioReturn, portfolioRisk, confidenceLevel)
	
	// Expected Shortfall should be less than VaR
	var95 := calc.CalculateVaR(portfolioReturn, portfolioRisk, confidenceLevel)
	assert.True(t, es < var95)
}

func TestCalculateMaxDrawdown(t *testing.T) {
	calc := NewPortfolioTheoryCalculator(0.02)
	
	// Returns that create a drawdown scenario
	returns := []float64{0.10, -0.05, -0.03, 0.08, 0.02}
	
	maxDrawdown := calc.CalculateMaxDrawdown(returns)
	
	// Should return a positive drawdown value
	assert.True(t, maxDrawdown >= 0)
	assert.True(t, maxDrawdown <= 1) // Should not exceed 100%
}

func TestCalculateSortinoRatio(t *testing.T) {
	calc := NewPortfolioTheoryCalculator(0.02)
	
	portfolioReturn := 0.12
	returns := []float64{0.05, -0.02, 0.08, -0.01, 0.10}
	
	sortinoRatio := calc.CalculateSortinoRatio(portfolioReturn, returns)
	
	// Should return a finite value
	assert.False(t, math.IsNaN(sortinoRatio))
	assert.False(t, math.IsInf(sortinoRatio, 0))
}

func TestCalculateCalmarRatio(t *testing.T) {
	calc := NewPortfolioTheoryCalculator(0.02)
	
	annualizedReturn := 0.15
	maxDrawdown := 0.08
	
	calmarRatio := calc.CalculateCalmarRatio(annualizedReturn, maxDrawdown)
	
	// Expected: 0.15 / 0.08 = 1.875
	assert.InDelta(t, 1.875, calmarRatio, 0.001)
}

func TestOptimizeMinimumVariance(t *testing.T) {
	calc := NewPortfolioTheoryCalculator(0.02)
	
	symbols := []string{"BTCUSDT", "ETHUSDT"}
	covarianceMatrix := map[string]map[string]float64{
		"BTCUSDT": {
			"BTCUSDT": 0.04,
			"ETHUSDT": 0.02,
		},
		"ETHUSDT": {
			"BTCUSDT": 0.02,
			"ETHUSDT": 0.03,
		},
	}
	
	constraints := shared.OptimizationConstraints{
		MaxPositionSize: 0.8,
	}
	
	weights, err := calc.OptimizeMinimumVariance(symbols, covarianceMatrix, constraints)
	
	assert.NoError(t, err)
	assert.Len(t, weights, 2)
	assert.Contains(t, weights, "BTCUSDT")
	assert.Contains(t, weights, "ETHUSDT")
	
	// Weights should be positive and sum approximately to 1
	totalWeight := weights["BTCUSDT"] + weights["ETHUSDT"]
	assert.InDelta(t, 1.0, totalWeight, 0.01)
}

func TestOptimizeMaxSharpe(t *testing.T) {
	calc := NewPortfolioTheoryCalculator(0.02)
	
	symbols := []string{"BTCUSDT", "ETHUSDT"}
	expectedReturns := map[string]float64{
		"BTCUSDT": 0.10,
		"ETHUSDT": 0.08,
	}
	covarianceMatrix := map[string]map[string]float64{
		"BTCUSDT": {
			"BTCUSDT": 0.04,
			"ETHUSDT": 0.02,
		},
		"ETHUSDT": {
			"BTCUSDT": 0.02,
			"ETHUSDT": 0.03,
		},
	}
	
	constraints := shared.OptimizationConstraints{
		MaxPositionSize: 0.8,
	}
	
	weights, err := calc.OptimizeMaxSharpe(symbols, expectedReturns, covarianceMatrix, constraints)
	
	assert.NoError(t, err)
	assert.Len(t, weights, 2)
	
	// Should favor higher return asset (BTCUSDT)
	assert.True(t, weights["BTCUSDT"] > 0)
	assert.True(t, weights["ETHUSDT"] >= 0)
}

func TestCalculateRiskParity(t *testing.T) {
	calc := NewPortfolioTheoryCalculator(0.02)
	
	symbols := []string{"BTCUSDT", "ETHUSDT"}
	covarianceMatrix := map[string]map[string]float64{
		"BTCUSDT": {
			"BTCUSDT": 0.04,
			"ETHUSDT": 0.02,
		},
		"ETHUSDT": {
			"BTCUSDT": 0.02,
			"ETHUSDT": 0.03,
		},
	}
	
	weights, err := calc.CalculateRiskParity(symbols, covarianceMatrix)
	
	assert.NoError(t, err)
	assert.Len(t, weights, 2)
	
	// Weights should sum to 1
	totalWeight := weights["BTCUSDT"] + weights["ETHUSDT"]
	assert.InDelta(t, 1.0, totalWeight, 0.01)
	
	// Lower volatility asset should have higher weight
	btcVol := math.Sqrt(covarianceMatrix["BTCUSDT"]["BTCUSDT"])
	ethVol := math.Sqrt(covarianceMatrix["ETHUSDT"]["ETHUSDT"])
	
	if ethVol < btcVol {
		assert.True(t, weights["ETHUSDT"] > weights["BTCUSDT"])
	}
}

func TestCalculateCorrelationMatrix(t *testing.T) {
	calc := NewPortfolioTheoryCalculator(0.02)
	
	covarianceMatrix := map[string]map[string]float64{
		"BTCUSDT": {
			"BTCUSDT": 0.04,
			"ETHUSDT": 0.02,
		},
		"ETHUSDT": {
			"BTCUSDT": 0.02,
			"ETHUSDT": 0.03,
		},
	}
	
	correlationMatrix := calc.CalculateCorrelationMatrix(covarianceMatrix)
	
	assert.Len(t, correlationMatrix, 2)
	
	// Diagonal elements should be 1 (perfect self-correlation)
	assert.InDelta(t, 1.0, correlationMatrix["BTCUSDT"]["BTCUSDT"], 0.001)
	assert.InDelta(t, 1.0, correlationMatrix["ETHUSDT"]["ETHUSDT"], 0.001)
	
	// Matrix should be symmetric
	assert.Equal(t, correlationMatrix["BTCUSDT"]["ETHUSDT"], correlationMatrix["ETHUSDT"]["BTCUSDT"])
	
	// Correlation values should be between -1 and 1
	corr := correlationMatrix["BTCUSDT"]["ETHUSDT"]
	assert.True(t, corr >= -1.0 && corr <= 1.0)
}

func TestValidateWeights(t *testing.T) {
	calc := NewPortfolioTheoryCalculator(0.02)
	
	constraints := shared.OptimizationConstraints{
		MaxPositionSize: 0.6,
	}
	
	// Valid weights
	validWeights := map[string]float64{
		"BTCUSDT": 0.4,
		"ETHUSDT": 0.6,
	}
	
	err := calc.ValidateWeights(validWeights, constraints)
	assert.NoError(t, err)
	
	// Invalid weights (don't sum to 1)
	invalidWeights1 := map[string]float64{
		"BTCUSDT": 0.3,
		"ETHUSDT": 0.5,
	}
	
	err = calc.ValidateWeights(invalidWeights1, constraints)
	assert.Error(t, err)
	
	// Invalid weights (exceed max position size)
	invalidWeights2 := map[string]float64{
		"BTCUSDT": 0.8,
		"ETHUSDT": 0.2,
	}
	
	err = calc.ValidateWeights(invalidWeights2, constraints)
	assert.Error(t, err)
	
	// Invalid weights (negative)
	invalidWeights3 := map[string]float64{
		"BTCUSDT": -0.2,
		"ETHUSDT": 1.2,
	}
	
	err = calc.ValidateWeights(invalidWeights3, constraints)
	assert.Error(t, err)
}

// Benchmark tests

func BenchmarkCalculatePortfolioReturn(b *testing.B) {
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

func BenchmarkCalculatePortfolioRisk(b *testing.B) {
	calc := NewPortfolioTheoryCalculator(0.02)
	
	weights := map[string]float64{
		"BTCUSDT": 0.25,
		"ETHUSDT": 0.25,
		"ADAUSDT": 0.25,
		"DOTUSDT": 0.25,
	}
	
	// Create a 4x4 covariance matrix
	covarianceMatrix := map[string]map[string]float64{
		"BTCUSDT": {"BTCUSDT": 0.04, "ETHUSDT": 0.02, "ADAUSDT": 0.015, "DOTUSDT": 0.018},
		"ETHUSDT": {"BTCUSDT": 0.02, "ETHUSDT": 0.03, "ADAUSDT": 0.012, "DOTUSDT": 0.015},
		"ADAUSDT": {"BTCUSDT": 0.015, "ETHUSDT": 0.012, "ADAUSDT": 0.05, "DOTUSDT": 0.02},
		"DOTUSDT": {"BTCUSDT": 0.018, "ETHUSDT": 0.015, "ADAUSDT": 0.02, "DOTUSDT": 0.045},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calc.CalculatePortfolioRisk(weights, covarianceMatrix)
	}
}