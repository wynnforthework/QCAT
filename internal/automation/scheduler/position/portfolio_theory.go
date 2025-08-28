package position

import (
	"fmt"
	"math"

	"qcat/internal/automation/scheduler/shared"
)

// PortfolioTheoryCalculator provides modern portfolio theory calculations
type PortfolioTheoryCalculator struct {
	riskFreeRate float64
}

// NewPortfolioTheoryCalculator creates a new portfolio theory calculator
func NewPortfolioTheoryCalculator(riskFreeRate float64) *PortfolioTheoryCalculator {
	return &PortfolioTheoryCalculator{
		riskFreeRate: riskFreeRate,
	}
}

// CalculatePortfolioReturn calculates expected portfolio return
func (ptc *PortfolioTheoryCalculator) CalculatePortfolioReturn(
	weights map[string]float64,
	expectedReturns map[string]float64,
) float64 {
	portfolioReturn := 0.0
	for symbol, weight := range weights {
		if expectedReturn, exists := expectedReturns[symbol]; exists {
			portfolioReturn += weight * expectedReturn
		}
	}
	return portfolioReturn
}

// CalculatePortfolioRisk calculates portfolio risk (standard deviation)
func (ptc *PortfolioTheoryCalculator) CalculatePortfolioRisk(
	weights map[string]float64,
	covarianceMatrix map[string]map[string]float64,
) float64 {
	variance := 0.0
	
	for symbol1, weight1 := range weights {
		for symbol2, weight2 := range weights {
			if covariances, exists := covarianceMatrix[symbol1]; exists {
				if covariance, exists := covariances[symbol2]; exists {
					variance += weight1 * weight2 * covariance
				}
			}
		}
	}
	
	return math.Sqrt(math.Max(variance, 0))
}

// CalculateSharpeRatio calculates the Sharpe ratio
func (ptc *PortfolioTheoryCalculator) CalculateSharpeRatio(
	portfolioReturn, portfolioRisk float64,
) float64 {
	if portfolioRisk == 0 {
		return 0
	}
	return (portfolioReturn - ptc.riskFreeRate) / portfolioRisk
}

// CalculateInformationRatio calculates the information ratio
func (ptc *PortfolioTheoryCalculator) CalculateInformationRatio(
	portfolioReturn, benchmarkReturn, trackingError float64,
) float64 {
	if trackingError == 0 {
		return 0
	}
	return (portfolioReturn - benchmarkReturn) / trackingError
}

// CalculateBeta calculates portfolio beta
func (ptc *PortfolioTheoryCalculator) CalculateBeta(
	portfolioReturns, marketReturns []float64,
) float64 {
	if len(portfolioReturns) != len(marketReturns) || len(portfolioReturns) < 2 {
		return 1.0 // Default beta
	}
	
	// Calculate means
	portfolioMean := ptc.calculateMean(portfolioReturns)
	marketMean := ptc.calculateMean(marketReturns)
	
	// Calculate covariance and market variance
	covariance := 0.0
	marketVariance := 0.0
	
	for i := 0; i < len(portfolioReturns); i++ {
		portfolioDev := portfolioReturns[i] - portfolioMean
		marketDev := marketReturns[i] - marketMean
		
		covariance += portfolioDev * marketDev
		marketVariance += marketDev * marketDev
	}
	
	if marketVariance == 0 {
		return 1.0
	}
	
	return covariance / marketVariance
}

// CalculateAlpha calculates Jensen's alpha
func (ptc *PortfolioTheoryCalculator) CalculateAlpha(
	portfolioReturn, marketReturn, beta float64,
) float64 {
	expectedReturn := ptc.riskFreeRate + beta*(marketReturn-ptc.riskFreeRate)
	return portfolioReturn - expectedReturn
}

// CalculateTrackingError calculates tracking error
func (ptc *PortfolioTheoryCalculator) CalculateTrackingError(
	portfolioReturns, benchmarkReturns []float64,
) float64 {
	if len(portfolioReturns) != len(benchmarkReturns) || len(portfolioReturns) < 2 {
		return 0
	}
	
	// Calculate excess returns
	excessReturns := make([]float64, len(portfolioReturns))
	for i := 0; i < len(portfolioReturns); i++ {
		excessReturns[i] = portfolioReturns[i] - benchmarkReturns[i]
	}
	
	return ptc.calculateStandardDeviation(excessReturns)
}

// CalculateVaR calculates Value at Risk using parametric method
func (ptc *PortfolioTheoryCalculator) CalculateVaR(
	portfolioReturn, portfolioRisk float64,
	confidenceLevel float64,
) float64 {
	// Z-score for confidence level (approximation)
	var zScore float64
	switch {
	case confidenceLevel >= 0.99:
		zScore = 2.33
	case confidenceLevel >= 0.95:
		zScore = 1.65
	case confidenceLevel >= 0.90:
		zScore = 1.28
	default:
		zScore = 1.65 // Default to 95%
	}
	
	return portfolioReturn - zScore*portfolioRisk
}

// CalculateExpectedShortfall calculates Expected Shortfall (Conditional VaR)
func (ptc *PortfolioTheoryCalculator) CalculateExpectedShortfall(
	portfolioReturn, portfolioRisk float64,
	confidenceLevel float64,
) float64 {
	// Simplified calculation assuming normal distribution
	var factor float64
	switch {
	case confidenceLevel >= 0.99:
		factor = 2.67
	case confidenceLevel >= 0.95:
		factor = 2.06
	case confidenceLevel >= 0.90:
		factor = 1.75
	default:
		factor = 2.06 // Default to 95%
	}
	
	return portfolioReturn - factor*portfolioRisk
}

// CalculateMaxDrawdown calculates maximum drawdown from return series
func (ptc *PortfolioTheoryCalculator) CalculateMaxDrawdown(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}
	
	// Calculate cumulative returns
	cumulative := make([]float64, len(returns))
	cumulative[0] = 1 + returns[0]
	
	for i := 1; i < len(returns); i++ {
		cumulative[i] = cumulative[i-1] * (1 + returns[i])
	}
	
	// Find maximum drawdown
	maxDrawdown := 0.0
	peak := cumulative[0]
	
	for i := 1; i < len(cumulative); i++ {
		if cumulative[i] > peak {
			peak = cumulative[i]
		}
		
		drawdown := (peak - cumulative[i]) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}
	
	return maxDrawdown
}

// CalculateSortinoRatio calculates Sortino ratio (downside deviation)
func (ptc *PortfolioTheoryCalculator) CalculateSortinoRatio(
	portfolioReturn float64,
	returns []float64,
) float64 {
	if len(returns) == 0 {
		return 0
	}
	
	// Calculate downside deviation
	downsideVariance := 0.0
	count := 0
	
	for _, ret := range returns {
		if ret < ptc.riskFreeRate {
			deviation := ret - ptc.riskFreeRate
			downsideVariance += deviation * deviation
			count++
		}
	}
	
	if count == 0 {
		return math.Inf(1) // No downside risk
	}
	
	downsideDeviation := math.Sqrt(downsideVariance / float64(count))
	if downsideDeviation == 0 {
		return math.Inf(1)
	}
	
	return (portfolioReturn - ptc.riskFreeRate) / downsideDeviation
}

// CalculateCalmarRatio calculates Calmar ratio
func (ptc *PortfolioTheoryCalculator) CalculateCalmarRatio(
	annualizedReturn, maxDrawdown float64,
) float64 {
	if maxDrawdown == 0 {
		return math.Inf(1)
	}
	return annualizedReturn / maxDrawdown
}

// OptimizeMinimumVariance finds minimum variance portfolio weights
func (ptc *PortfolioTheoryCalculator) OptimizeMinimumVariance(
	symbols []string,
	covarianceMatrix map[string]map[string]float64,
	constraints shared.OptimizationConstraints,
) (map[string]float64, error) {
	n := len(symbols)
	if n == 0 {
		return make(map[string]float64), nil
	}
	
	// Simplified minimum variance optimization
	// In practice, would use quadratic programming solver
	
	weights := make(map[string]float64)
	
	// Calculate inverse variance weights
	totalInverseVariance := 0.0
	inverseVariances := make(map[string]float64)
	
	for _, symbol := range symbols {
		variance := covarianceMatrix[symbol][symbol]
		if variance <= 0 {
			variance = 0.01 // Minimum variance
		}
		
		inverseVar := 1.0 / variance
		inverseVariances[symbol] = inverseVar
		totalInverseVariance += inverseVar
	}
	
	// Normalize weights
	for _, symbol := range symbols {
		weight := inverseVariances[symbol] / totalInverseVariance
		
		// Apply constraints
		if weight > constraints.MaxPositionSize {
			weight = constraints.MaxPositionSize
		}
		
		weights[symbol] = weight
	}
	
	// Renormalize if constraints were applied
	totalWeight := 0.0
	for _, weight := range weights {
		totalWeight += weight
	}
	
	if totalWeight > 0 {
		for symbol := range weights {
			weights[symbol] /= totalWeight
		}
	}
	
	return weights, nil
}

// OptimizeMaxSharpe finds maximum Sharpe ratio portfolio weights
func (ptc *PortfolioTheoryCalculator) OptimizeMaxSharpe(
	symbols []string,
	expectedReturns map[string]float64,
	covarianceMatrix map[string]map[string]float64,
	constraints shared.OptimizationConstraints,
) (map[string]float64, error) {
	// Simplified maximum Sharpe ratio optimization
	weights := make(map[string]float64)
	
	// Calculate risk-adjusted scores
	scores := make(map[string]float64)
	totalScore := 0.0
	
	for _, symbol := range symbols {
		expectedReturn := expectedReturns[symbol]
		variance := covarianceMatrix[symbol][symbol]
		
		if variance <= 0 {
			variance = 0.01
		}
		
		// Sharpe-like score
		score := (expectedReturn - ptc.riskFreeRate) / math.Sqrt(variance)
		if score > 0 {
			scores[symbol] = score
			totalScore += score
		}
	}
	
	// Normalize to get weights
	if totalScore > 0 {
		for _, symbol := range symbols {
			if score, exists := scores[symbol]; exists {
				weight := score / totalScore
				
				// Apply constraints
				if weight > constraints.MaxPositionSize {
					weight = constraints.MaxPositionSize
				}
				
				weights[symbol] = weight
			}
		}
	} else {
		// Equal weights if no positive scores
		equalWeight := 1.0 / float64(len(symbols))
		for _, symbol := range symbols {
			weights[symbol] = equalWeight
		}
	}
	
	return weights, nil
}

// CalculateRiskParity calculates risk parity weights
func (ptc *PortfolioTheoryCalculator) CalculateRiskParity(
	symbols []string,
	covarianceMatrix map[string]map[string]float64,
) (map[string]float64, error) {
	weights := make(map[string]float64)
	
	// Simplified risk parity - equal risk contribution
	// In practice, would use iterative optimization
	
	totalInverseVol := 0.0
	inverseVols := make(map[string]float64)
	
	for _, symbol := range symbols {
		variance := covarianceMatrix[symbol][symbol]
		if variance <= 0 {
			variance = 0.01
		}
		
		volatility := math.Sqrt(variance)
		inverseVol := 1.0 / volatility
		inverseVols[symbol] = inverseVol
		totalInverseVol += inverseVol
	}
	
	// Normalize weights
	for _, symbol := range symbols {
		weights[symbol] = inverseVols[symbol] / totalInverseVol
	}
	
	return weights, nil
}

// Helper methods

func (ptc *PortfolioTheoryCalculator) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}

func (ptc *PortfolioTheoryCalculator) calculateStandardDeviation(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	
	mean := ptc.calculateMean(values)
	variance := 0.0
	
	for _, value := range values {
		deviation := value - mean
		variance += deviation * deviation
	}
	
	variance /= float64(len(values) - 1)
	return math.Sqrt(variance)
}

// CalculateCorrelationMatrix calculates correlation matrix from covariance matrix
func (ptc *PortfolioTheoryCalculator) CalculateCorrelationMatrix(
	covarianceMatrix map[string]map[string]float64,
) map[string]map[string]float64 {
	correlationMatrix := make(map[string]map[string]float64)
	
	for symbol1, covariances := range covarianceMatrix {
		correlationMatrix[symbol1] = make(map[string]float64)
		
		vol1 := math.Sqrt(covariances[symbol1])
		if vol1 == 0 {
			vol1 = 1 // Avoid division by zero
		}
		
		for symbol2, covariance := range covariances {
			vol2 := math.Sqrt(covarianceMatrix[symbol2][symbol2])
			if vol2 == 0 {
				vol2 = 1
			}
			
			correlation := covariance / (vol1 * vol2)
			correlationMatrix[symbol1][symbol2] = correlation
		}
	}
	
	return correlationMatrix
}

// ValidateWeights validates that portfolio weights sum to 1 and meet constraints
func (ptc *PortfolioTheoryCalculator) ValidateWeights(
	weights map[string]float64,
	constraints shared.OptimizationConstraints,
) error {
	totalWeight := 0.0
	for symbol, weight := range weights {
		if weight < 0 {
			return fmt.Errorf("negative weight for symbol %s: %f", symbol, weight)
		}
		
		if weight > constraints.MaxPositionSize {
			return fmt.Errorf("weight exceeds maximum for symbol %s: %f > %f", 
				symbol, weight, constraints.MaxPositionSize)
		}
		
		totalWeight += weight
	}
	
	if math.Abs(totalWeight-1.0) > 0.01 {
		return fmt.Errorf("weights do not sum to 1: %f", totalWeight)
	}
	
	return nil
}