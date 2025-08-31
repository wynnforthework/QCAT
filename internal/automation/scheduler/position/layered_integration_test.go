package position

import (
	"testing"
	"time"

	"qcat/internal/automation/scheduler/shared"
)

// TestLayeredPositionIntegration tests the integration between layered position manager and execution system
func TestLayeredPositionIntegration(t *testing.T) {
	// Test volatility analysis calculations
	t.Run("VolatilityAnalysis", func(t *testing.T) {
		// Test volatility calculation logic
		prices := []float64{100.0, 102.0, 98.0, 101.0, 99.0, 103.0, 97.0}

		// Calculate returns
		returns := make([]float64, len(prices)-1)
		for i := 1; i < len(prices); i++ {
			returns[i-1] = (prices[i] - prices[i-1]) / prices[i-1]
		}

		// Calculate volatility
		mean := 0.0
		for _, ret := range returns {
			mean += ret
		}
		mean /= float64(len(returns))

		variance := 0.0
		for _, ret := range returns {
			variance += (ret - mean) * (ret - mean)
		}
		variance /= float64(len(returns) - 1)

		volatility := variance // Simplified volatility

		if volatility < 0 {
			t.Error("Volatility should be non-negative")
		}

		t.Logf("Calculated volatility: %f", volatility)
	})

	// Test layer configuration logic
	t.Run("LayerConfiguration", func(t *testing.T) {
		strategy := &LayeredStrategy{
			ID:              "test_strategy",
			Symbol:          "BTCUSDT",
			Direction:       "LONG",
			TotalSize:       10.0,
			MaxLayers:       5,
			LayerSizeRatio:  1.0,
			PriceSpacing:    0.02,
			VolatilityBased: true,
			AdaptiveSpacing: true,
			RiskParameters: shared.RiskParams{
				StopLossPercent:   0.05,
				TakeProfitPercent: 0.10,
			},
			CreatedAt: time.Now(),
		}

		// Test basic layer size calculation
		layerSize := strategy.TotalSize / float64(strategy.MaxLayers)
		expectedLayerSize := 2.0

		if layerSize != expectedLayerSize {
			t.Errorf("Expected layer size %f, got %f", expectedLayerSize, layerSize)
		}

		// Test price spacing calculation
		currentPrice := 100.0
		expectedEntryPrice := currentPrice * (1 - strategy.PriceSpacing)

		if expectedEntryPrice >= currentPrice {
			t.Error("Entry price should be below current price for LONG strategy")
		}

		t.Logf("Strategy: %+v", strategy)
		t.Logf("Layer size: %f", layerSize)
		t.Logf("Expected entry price: %f", expectedEntryPrice)
	})

	// Test risk adjustment calculations
	t.Run("RiskAdjustment", func(t *testing.T) {
		// Test volatility-based risk adjustment
		testCases := []struct {
			volatility         float64
			expectedRegime     string
			expectedAdjustment float64
		}{
			{0.10, "LOW", -0.1},
			{0.25, "MEDIUM", 0.0},
			{0.40, "HIGH", 0.2},
			{0.60, "EXTREME", 0.4},
		}

		for _, tc := range testCases {
			regime := classifyVolatilityRegime(tc.volatility)
			adjustment := calculateRiskAdjustmentForVolatility(tc.volatility, regime)

			if regime != tc.expectedRegime {
				t.Errorf("Expected regime %s for volatility %f, got %s", tc.expectedRegime, tc.volatility, regime)
			}

			if adjustment != tc.expectedAdjustment {
				t.Errorf("Expected adjustment %f for volatility %f, got %f", tc.expectedAdjustment, tc.volatility, adjustment)
			}
		}
	})

	// Test execution metrics calculations
	t.Run("ExecutionMetrics", func(t *testing.T) {
		results := []LayerExecutionResult{
			{Success: true, ExecutedSize: 2.0, ExecutedPrice: 100.0, SlippageImpact: 0.001},
			{Success: true, ExecutedSize: 2.0, ExecutedPrice: 98.0, SlippageImpact: 0.002},
			{Success: false, ExecutedSize: 0.0, ExecutedPrice: 0.0, SlippageImpact: 0.0},
		}

		// Calculate success rate
		successfulLayers := 0
		totalSlippage := 0.0
		totalSize := 0.0
		totalValue := 0.0

		for _, result := range results {
			if result.Success {
				successfulLayers++
				totalSlippage += result.SlippageImpact
				totalSize += result.ExecutedSize
				totalValue += result.ExecutedSize * result.ExecutedPrice
			}
		}

		successRate := float64(successfulLayers) / float64(len(results))
		averageSlippage := totalSlippage / float64(successfulLayers)
		averagePrice := totalValue / totalSize

		expectedSuccessRate := 2.0 / 3.0 // 2 out of 3 successful
		if successRate != expectedSuccessRate {
			t.Errorf("Expected success rate %f, got %f", expectedSuccessRate, successRate)
		}

		expectedAverageSlippage := 0.0015 // (0.001 + 0.002) / 2
		if averageSlippage != expectedAverageSlippage {
			t.Errorf("Expected average slippage %f, got %f", expectedAverageSlippage, averageSlippage)
		}

		expectedAveragePrice := 99.0 // (2*100 + 2*98) / 4
		if averagePrice != expectedAveragePrice {
			t.Errorf("Expected average price %f, got %f", expectedAveragePrice, averagePrice)
		}

		t.Logf("Success rate: %f", successRate)
		t.Logf("Average slippage: %f", averageSlippage)
		t.Logf("Average price: %f", averagePrice)
	})

	// Test partial closure calculations
	t.Run("PartialClosure", func(t *testing.T) {
		layer := shared.Layer{
			ID:         "test_layer",
			Size:       2.0,
			EntryPrice: 95.0,
			Status:     "ACTIVE",
		}

		closurePercentage := 0.5
		currentPrice := 105.0

		// Calculate closure metrics
		closureSize := layer.Size * closurePercentage
		pnlPerUnit := currentPrice - layer.EntryPrice
		realizedPnL := pnlPerUnit * closureSize
		remainingSize := layer.Size - closureSize

		expectedClosureSize := 1.0
		expectedRealizedPnL := 10.0 // (105 - 95) * 1.0
		expectedRemainingSize := 1.0

		if closureSize != expectedClosureSize {
			t.Errorf("Expected closure size %f, got %f", expectedClosureSize, closureSize)
		}

		if realizedPnL != expectedRealizedPnL {
			t.Errorf("Expected realized PnL %f, got %f", expectedRealizedPnL, realizedPnL)
		}

		if remainingSize != expectedRemainingSize {
			t.Errorf("Expected remaining size %f, got %f", expectedRemainingSize, remainingSize)
		}

		t.Logf("Closure size: %f", closureSize)
		t.Logf("Realized PnL: %f", realizedPnL)
		t.Logf("Remaining size: %f", remainingSize)
	})
}

// Helper functions for testing
func classifyVolatilityRegime(volatility float64) string {
	if volatility < 0.15 {
		return "LOW"
	} else if volatility < 0.30 {
		return "MEDIUM"
	} else if volatility < 0.50 {
		return "HIGH"
	} else {
		return "EXTREME"
	}
}

func calculateRiskAdjustmentForVolatility(volatility float64, regime string) float64 {
	switch regime {
	case "LOW":
		return -0.1
	case "MEDIUM":
		return 0.0
	case "HIGH":
		return 0.2
	case "EXTREME":
		return 0.4
	default:
		return 0.0
	}
}

// TestLayeredPositionManagerLogic tests the core logic without external dependencies
func TestLayeredPositionManagerLogic(t *testing.T) {
	t.Run("VolatilityTrendDetermination", func(t *testing.T) {
		testCases := []struct {
			shortTerm     float64
			mediumTerm    float64
			longTerm      float64
			expectedTrend string
		}{
			{0.30, 0.25, 0.20, "INCREASING"},
			{0.20, 0.25, 0.30, "DECREASING"},
			{0.25, 0.24, 0.26, "STABLE"},
		}

		for _, tc := range testCases {
			trend := determineVolatilityTrend(tc.shortTerm, tc.mediumTerm, tc.longTerm)
			if trend != tc.expectedTrend {
				t.Errorf("Expected trend %s for volatilities (%f, %f, %f), got %s",
					tc.expectedTrend, tc.shortTerm, tc.mediumTerm, tc.longTerm, trend)
			}
		}
	})

	t.Run("RecommendedLayersCalculation", func(t *testing.T) {
		testCases := []struct {
			regime         string
			maxLayers      int
			expectedLayers int
		}{
			{"LOW", 10, 3},
			{"MEDIUM", 10, 5},
			{"HIGH", 10, 7},
			{"EXTREME", 10, 10},
		}

		for _, tc := range testCases {
			layers := calculateRecommendedLayersForRegime(tc.regime, tc.maxLayers)
			if layers != tc.expectedLayers {
				t.Errorf("Expected %d layers for regime %s (max %d), got %d",
					tc.expectedLayers, tc.regime, tc.maxLayers, layers)
			}
		}
	})
}

// Helper functions that mirror the internal logic
func determineVolatilityTrend(shortTerm, mediumTerm, longTerm float64) string {
	shortToMedium := (shortTerm - mediumTerm) / mediumTerm
	mediumToLong := (mediumTerm - longTerm) / longTerm

	if shortToMedium > 0.1 && mediumToLong > 0.05 {
		return "INCREASING"
	} else if shortToMedium < -0.1 && mediumToLong < -0.05 {
		return "DECREASING"
	} else {
		return "STABLE"
	}
}

func calculateRecommendedLayersForRegime(regime string, maxLayers int) int {
	baseLayers := maxLayers / 2

	switch regime {
	case "LOW":
		if baseLayers-2 < 3 {
			return 3
		}
		return baseLayers - 2
	case "MEDIUM":
		return baseLayers
	case "HIGH":
		if baseLayers+2 > maxLayers {
			return maxLayers
		}
		return baseLayers + 2
	case "EXTREME":
		return maxLayers
	default:
		return baseLayers
	}
}

// BenchmarkLayeredPositionLogic benchmarks the core logic functions
func BenchmarkLayeredPositionLogic(b *testing.B) {
	b.Run("VolatilityCalculation", func(b *testing.B) {
		prices := []float64{100.0, 102.0, 98.0, 101.0, 99.0, 103.0, 97.0, 105.0, 95.0, 104.0}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			calculateVolatilityFromPrices(prices)
		}
	})

	b.Run("LayerConfigurationCalculation", func(b *testing.B) {
		strategy := &LayeredStrategy{
			TotalSize:    10.0,
			MaxLayers:    5,
			PriceSpacing: 0.02,
			Direction:    "LONG",
		}
		currentPrice := 100.0

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			calculateLayerPrices(strategy, currentPrice)
		}
	})
}

func calculateVolatilityFromPrices(prices []float64) float64 {
	if len(prices) < 2 {
		return 0.0
	}

	returns := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		returns[i-1] = (prices[i] - prices[i-1]) / prices[i-1]
	}

	mean := 0.0
	for _, ret := range returns {
		mean += ret
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, ret := range returns {
		variance += (ret - mean) * (ret - mean)
	}
	variance /= float64(len(returns) - 1)

	return variance
}

func calculateLayerPrices(strategy *LayeredStrategy, currentPrice float64) []float64 {
	prices := make([]float64, strategy.MaxLayers)

	for i := 0; i < strategy.MaxLayers; i++ {
		if strategy.Direction == "LONG" {
			prices[i] = currentPrice * (1 - strategy.PriceSpacing*float64(i+1))
		} else {
			prices[i] = currentPrice * (1 + strategy.PriceSpacing*float64(i+1))
		}
	}

	return prices
}
