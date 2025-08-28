package position

import (
	"testing"
)

func TestFundAllocator_CalculateEfficiencyScore(t *testing.T) {
	// Test the efficiency score calculation logic
	allocator := &FundAllocator{
		riskFreeRate: 0.02,
	}
	
	t.Run("calculate efficiency score", func(t *testing.T) {
		sharpe := 0.8
		calmar := 0.6
		sortino := 0.7
		information := 0.5
		
		score := allocator.calculateEfficiencyScore(sharpe, calmar, sortino, information)
		
		if score < 0 {
			t.Error("Efficiency score should be non-negative")
		}
		
		// Expected score: 0.4*0.8 + 0.3*0.6 + 0.2*0.7 + 0.1*0.5 = 0.32 + 0.18 + 0.14 + 0.05 = 0.69
		expectedScore := 0.69
		if score < expectedScore-0.001 || score > expectedScore+0.001 {
			t.Errorf("Expected efficiency score %f, got %f", expectedScore, score)
		}
	})
}

func TestFundAllocator_CalculateVolatilityAdjustment(t *testing.T) {
	allocator := &FundAllocator{}
	
	t.Run("volatility adjustment calculation", func(t *testing.T) {
		testCases := []struct {
			volatility float64
			expected   float64
		}{
			{0.1, 0.95},  // Low volatility gets higher adjustment
			{0.2, 0.9},   // Medium volatility
			{0.4, 0.8},   // High volatility gets lower adjustment
			{0.8, 0.6},   // Very high volatility
		}
		
		for _, tc := range testCases {
			adjustment := allocator.calculateVolatilityAdjustment(tc.volatility)
			
			if adjustment < 0.1 || adjustment > 1.5 {
				t.Errorf("Adjustment %f out of range [0.1, 1.5] for volatility %f", 
					adjustment, tc.volatility)
			}
			
			if adjustment != tc.expected {
				t.Errorf("Expected adjustment %f for volatility %f, got %f", 
					tc.expected, tc.volatility, adjustment)
			}
		}
	})
}

func TestFundAllocator_CalculateMean(t *testing.T) {
	allocator := &FundAllocator{}
	
	t.Run("calculate mean of values", func(t *testing.T) {
		testCases := []struct {
			values   []float64
			expected float64
		}{
			{[]float64{1.0, 2.0, 3.0}, 2.0},
			{[]float64{0.1, 0.2, 0.3, 0.4}, 0.25},
			{[]float64{}, 0.0},
			{[]float64{5.0}, 5.0},
		}
		
		for _, tc := range testCases {
			result := allocator.calculateMean(tc.values)
			if result != tc.expected {
				t.Errorf("Expected mean %f for values %v, got %f", 
					tc.expected, tc.values, result)
			}
		}
	})
}

func TestFundReallocationExecutor_CalculatePriority(t *testing.T) {
	executor := &FundReallocationExecutor{}
	
	t.Run("calculate priority based on allocation change and cost", func(t *testing.T) {
		testCases := []struct {
			allocationChange float64
			transactionCost  float64
			expectedMin      int
		}{
			{0.1, 0.001, 1},   // Large change, low cost = high priority (low number)
			{0.01, 0.01, 1},   // Small change, high cost = lower priority
			{0.05, 0.005, 1},  // Medium change, medium cost
		}
		
		for _, tc := range testCases {
			priority := executor.calculatePriority(tc.allocationChange, tc.transactionCost)
			
			if priority < tc.expectedMin {
				t.Errorf("Priority %d below minimum %d for change %f, cost %f", 
					priority, tc.expectedMin, tc.allocationChange, tc.transactionCost)
			}
		}
	})
}

func TestFundReallocationExecutor_NormalizeWeights(t *testing.T) {
	executor := &FundReallocationExecutor{}
	
	t.Run("normalize weights to sum to 1", func(t *testing.T) {
		weights := []float64{0.3, 0.5, 0.7} // Sum = 1.5
		
		executor.normalizeWeights(weights)
		
		// Check that weights sum to 1
		sum := 0.0
		for _, w := range weights {
			sum += w
		}
		
		if sum < 0.99 || sum > 1.01 {
			t.Errorf("Expected normalized weights to sum to 1, got %f", sum)
		}
		
		// Check individual weights are correct
		expectedWeights := []float64{0.2, 0.333333, 0.466667}
		for i, expected := range expectedWeights {
			if weights[i] < expected-0.01 || weights[i] > expected+0.01 {
				t.Errorf("Expected weight %f at index %d, got %f", expected, i, weights[i])
			}
		}
	})
	
	t.Run("handle zero sum weights", func(t *testing.T) {
		weights := []float64{0.0, 0.0, 0.0}
		
		executor.normalizeWeights(weights)
		
		// Should remain unchanged when sum is zero
		for i, w := range weights {
			if w != 0.0 {
				t.Errorf("Expected weight to remain 0 at index %d, got %f", i, w)
			}
		}
	})
}

func TestFundReallocationExecutor_CalculateEffectivenessScore(t *testing.T) {
	executor := &FundReallocationExecutor{}
	
	t.Run("calculate effectiveness score", func(t *testing.T) {
		performanceChange := map[string]float64{
			"total_return": 0.02, // 2% improvement
		}
		
		riskChange := map[string]float64{
			"volatility": -0.01, // 1% risk reduction
		}
		
		costAnalysis := map[string]float64{
			"total_cost": 100.0,
		}
		
		score := executor.calculateEffectivenessScore(performanceChange, riskChange, costAnalysis)
		
		if score < 0 || score > 1 {
			t.Errorf("Expected effectiveness score between 0 and 1, got %f", score)
		}
		
		// Score should be positive due to return improvement and risk reduction
		if score <= 0.5 {
			t.Errorf("Expected positive effectiveness score, got %f", score)
		}
	})
}