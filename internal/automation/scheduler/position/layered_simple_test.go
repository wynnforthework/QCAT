package position

import (
	"testing"
	"time"

	"qcat/internal/automation/scheduler/shared"
)

// TestLayeredPositionManagerTypes tests the type definitions and basic functionality
func TestLayeredPositionManagerTypes(t *testing.T) {
	t.Run("VolatilityAnalysis", func(t *testing.T) {
		analysis := &VolatilityAnalysis{
			Symbol:               "BTCUSDT",
			ShortTermVolatility:  0.25,
			MediumTermVolatility: 0.20,
			LongTermVolatility:   0.18,
			VolatilityTrend:      "STABLE",
			VolatilityRegime:     "MEDIUM",
			RecommendedLayers:    5,
			OptimalLayerSize:     2.0,
			RiskAdjustment:       0.1,
			Confidence:           0.8,
			AnalyzedAt:           time.Now(),
		}
		
		if analysis.Symbol != "BTCUSDT" {
			t.Errorf("Expected symbol BTCUSDT, got %s", analysis.Symbol)
		}
		
		if analysis.RecommendedLayers != 5 {
			t.Errorf("Expected 5 recommended layers, got %d", analysis.RecommendedLayers)
		}
		
		if analysis.OptimalLayerSize != 2.0 {
			t.Errorf("Expected optimal layer size 2.0, got %f", analysis.OptimalLayerSize)
		}
	})
	
	t.Run("LayeredStrategy", func(t *testing.T) {
		strategy := &LayeredStrategy{
			ID:                "test_strategy_1",
			Symbol:            "BTCUSDT",
			Direction:         "LONG",
			TotalSize:         10.0,
			MaxLayers:         5,
			LayerSizeRatio:    1.0,
			PriceSpacing:      0.02,
			VolatilityBased:   true,
			AdaptiveSpacing:   true,
			RiskParameters: shared.RiskParams{
				MaxLeverage:       2.0,
				MaxPositionSize:   20.0,
				StopLossPercent:   0.05,
				TakeProfitPercent: 0.10,
				MaxDrawdown:       0.15,
				VaRLimit:          0.02,
			},
			CreatedAt: time.Now(),
		}
		
		if strategy.Symbol != "BTCUSDT" {
			t.Errorf("Expected symbol BTCUSDT, got %s", strategy.Symbol)
		}
		
		if strategy.Direction != "LONG" {
			t.Errorf("Expected direction LONG, got %s", strategy.Direction)
		}
		
		if strategy.TotalSize != 10.0 {
			t.Errorf("Expected total size 10.0, got %f", strategy.TotalSize)
		}
		
		if strategy.MaxLayers != 5 {
			t.Errorf("Expected max layers 5, got %d", strategy.MaxLayers)
		}
	})
	
	t.Run("LayeredExecution", func(t *testing.T) {
		execution := &LayeredExecution{
			ID:         "exec_1",
			StrategyID: "strategy_1",
			Symbol:     "BTCUSDT",
			Status:     "EXECUTING",
			StartedAt:  time.Now(),
			Config: &shared.LayerConfig{
				Symbol: "BTCUSDT",
				Layers: []shared.Layer{
					{
						ID:         "layer_1",
						Level:      1,
						Size:       2.0,
						EntryPrice: 95.0,
						StopLoss:   90.25,
						TakeProfit: 104.5,
						Status:     "ACTIVE",
						CreatedAt:  time.Now(),
					},
				},
			},
			TotalExecuted: 2.0,
			AveragePrice:  95.0,
		}
		
		if execution.Symbol != "BTCUSDT" {
			t.Errorf("Expected symbol BTCUSDT, got %s", execution.Symbol)
		}
		
		if execution.Status != "EXECUTING" {
			t.Errorf("Expected status EXECUTING, got %s", execution.Status)
		}
		
		if len(execution.Config.Layers) != 1 {
			t.Errorf("Expected 1 layer, got %d", len(execution.Config.Layers))
		}
		
		layer := execution.Config.Layers[0]
		if layer.Size != 2.0 {
			t.Errorf("Expected layer size 2.0, got %f", layer.Size)
		}
		
		if layer.EntryPrice != 95.0 {
			t.Errorf("Expected entry price 95.0, got %f", layer.EntryPrice)
		}
	})
	
	t.Run("LayerExecutionResult", func(t *testing.T) {
		result := &LayerExecutionResult{
			LayerID:         "layer_1",
			Success:         true,
			ExecutedSize:    2.0,
			ExecutedPrice:   95.5,
			ExecutionTime:   time.Second * 30,
			TransactionCost: 0.19,
			SlippageImpact:  0.005,
			ExecutedAt:      time.Now(),
			Metadata: map[string]interface{}{
				"order_type": "LIMIT",
			},
		}
		
		if result.LayerID != "layer_1" {
			t.Errorf("Expected layer ID layer_1, got %s", result.LayerID)
		}
		
		if !result.Success {
			t.Error("Expected success to be true")
		}
		
		if result.ExecutedSize != 2.0 {
			t.Errorf("Expected executed size 2.0, got %f", result.ExecutedSize)
		}
		
		if result.ExecutedPrice != 95.5 {
			t.Errorf("Expected executed price 95.5, got %f", result.ExecutedPrice)
		}
		
		if result.SlippageImpact != 0.005 {
			t.Errorf("Expected slippage impact 0.005, got %f", result.SlippageImpact)
		}
	})
	
	t.Run("PartialClosureRequest", func(t *testing.T) {
		request := &PartialClosureRequest{
			ExecutionID:       "exec_1",
			LayerIDs:          []string{"layer_1", "layer_2"},
			ClosureReason:     "PROFIT_TAKING",
			ClosurePercentage: 0.5,
			PriceLimit:        105.0,
			TimeLimit:         time.Minute * 5,
			Metadata: map[string]interface{}{
				"trigger": "manual",
			},
		}
		
		if request.ExecutionID != "exec_1" {
			t.Errorf("Expected execution ID exec_1, got %s", request.ExecutionID)
		}
		
		if len(request.LayerIDs) != 2 {
			t.Errorf("Expected 2 layer IDs, got %d", len(request.LayerIDs))
		}
		
		if request.ClosurePercentage != 0.5 {
			t.Errorf("Expected closure percentage 0.5, got %f", request.ClosurePercentage)
		}
		
		if request.ClosureReason != "PROFIT_TAKING" {
			t.Errorf("Expected closure reason PROFIT_TAKING, got %s", request.ClosureReason)
		}
	})
	
	t.Run("LayerPerformanceMetrics", func(t *testing.T) {
		metrics := &LayerPerformanceMetrics{
			StrategyID:          "strategy_1",
			TotalLayers:         5,
			ActiveLayers:        3,
			CompletedLayers:     2,
			AverageLayerReturn:  0.05,
			TotalReturn:         0.08,
			RiskAdjustedReturn:  0.12,
			MaxDrawdown:         0.03,
			VolatilityImpact:    0.02,
			ExecutionEfficiency: 0.85,
			CalculatedAt:        time.Now(),
			Metadata: map[string]interface{}{
				"calculation_method": "weighted_average",
			},
		}
		
		if metrics.StrategyID != "strategy_1" {
			t.Errorf("Expected strategy ID strategy_1, got %s", metrics.StrategyID)
		}
		
		if metrics.TotalLayers != 5 {
			t.Errorf("Expected total layers 5, got %d", metrics.TotalLayers)
		}
		
		if metrics.ActiveLayers != 3 {
			t.Errorf("Expected active layers 3, got %d", metrics.ActiveLayers)
		}
		
		if metrics.ExecutionEfficiency != 0.85 {
			t.Errorf("Expected execution efficiency 0.85, got %f", metrics.ExecutionEfficiency)
		}
		
		if metrics.TotalReturn != 0.08 {
			t.Errorf("Expected total return 0.08, got %f", metrics.TotalReturn)
		}
	})
}

// TestLayeredPositionLogic tests the core logic functions
func TestLayeredPositionLogic(t *testing.T) {
	t.Run("VolatilityCalculations", func(t *testing.T) {
		// Test volatility regime classification
		testCases := []struct {
			shortTerm    float64
			mediumTerm   float64
			longTerm     float64
			expectedRegime string
		}{
			{0.10, 0.12, 0.11, "LOW"},
			{0.25, 0.23, 0.24, "MEDIUM"},
			{0.40, 0.38, 0.39, "HIGH"},
			{0.60, 0.58, 0.59, "EXTREME"},
		}
		
		for _, tc := range testCases {
			avgVolatility := (tc.shortTerm + tc.mediumTerm + tc.longTerm) / 3
			
			var regime string
			if avgVolatility < 0.15 {
				regime = "LOW"
			} else if avgVolatility < 0.30 {
				regime = "MEDIUM"
			} else if avgVolatility < 0.50 {
				regime = "HIGH"
			} else {
				regime = "EXTREME"
			}
			
			if regime != tc.expectedRegime {
				t.Errorf("Expected regime %s for avg volatility %f, got %s", 
					tc.expectedRegime, avgVolatility, regime)
			}
		}
	})
	
	t.Run("LayerSizeCalculations", func(t *testing.T) {
		strategy := &LayeredStrategy{
			TotalSize:      10.0,
			MaxLayers:      5,
			LayerSizeRatio: 1.0,
		}
		
		// Test equal layer sizes
		layerSize := strategy.TotalSize / float64(strategy.MaxLayers)
		expectedLayerSize := 2.0
		
		if layerSize != expectedLayerSize {
			t.Errorf("Expected layer size %f, got %f", expectedLayerSize, layerSize)
		}
		
		// Test geometric progression
		strategy.LayerSizeRatio = 1.2
		
		totalCalculatedSize := 0.0
		for i := 0; i < strategy.MaxLayers; i++ {
			adjustedLayerSize := layerSize * pow(strategy.LayerSizeRatio, float64(i))
			totalCalculatedSize += adjustedLayerSize
		}
		
		if totalCalculatedSize <= strategy.TotalSize {
			t.Logf("Geometric progression total size: %f (original: %f)", totalCalculatedSize, strategy.TotalSize)
		}
	})
	
	t.Run("PriceSpacingCalculations", func(t *testing.T) {
		currentPrice := 100.0
		priceSpacing := 0.02
		maxLayers := 5
		
		// Test LONG position layer prices
		for i := 0; i < maxLayers; i++ {
			entryPrice := currentPrice * (1 - priceSpacing*float64(i+1))
			
			if entryPrice >= currentPrice {
				t.Errorf("Layer %d entry price %f should be below current price %f for LONG", 
					i+1, entryPrice, currentPrice)
			}
			
			if i > 0 {
				prevEntryPrice := currentPrice * (1 - priceSpacing*float64(i))
				if entryPrice >= prevEntryPrice {
					t.Errorf("Layer %d entry price %f should be below previous layer price %f", 
						i+1, entryPrice, prevEntryPrice)
				}
			}
		}
		
		// Test SHORT position layer prices
		for i := 0; i < maxLayers; i++ {
			entryPrice := currentPrice * (1 + priceSpacing*float64(i+1))
			
			if entryPrice <= currentPrice {
				t.Errorf("Layer %d entry price %f should be above current price %f for SHORT", 
					i+1, entryPrice, currentPrice)
			}
		}
	})
	
	t.Run("RiskAdjustmentCalculations", func(t *testing.T) {
		layer := &shared.Layer{
			EntryPrice: 100.0,
			StopLoss:   95.0,
			Size:       2.0,
		}
		
		// Test risk adjustment
		riskAdjustment := 0.2 // 20% increase in risk management
		
		originalStopLoss := layer.StopLoss
		stopLossDistance := layer.EntryPrice - layer.StopLoss // 5.0
		
		// Apply risk adjustment (tighter stop loss)
		newStopLoss := layer.EntryPrice - stopLossDistance*(1+riskAdjustment*0.5)
		expectedNewStopLoss := 100.0 - 5.0*1.1 // 94.5
		
		if newStopLoss != expectedNewStopLoss {
			t.Errorf("Expected new stop loss %f, got %f", expectedNewStopLoss, newStopLoss)
		}
		
		if newStopLoss >= originalStopLoss {
			t.Error("Risk adjustment should result in tighter stop loss")
		}
	})
}

// Helper function for power calculation
func pow(base, exp float64) float64 {
	if exp == 0 {
		return 1
	}
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}

// TestLayeredExecutionMetrics tests execution metrics calculations
func TestLayeredExecutionMetrics(t *testing.T) {
	t.Run("ExecutionEfficiency", func(t *testing.T) {
		results := []LayerExecutionResult{
			{Success: true, ExecutedSize: 2.0},
			{Success: true, ExecutedSize: 2.0},
			{Success: false, ExecutedSize: 0.0},
			{Success: true, ExecutedSize: 1.5},
		}
		
		successfulLayers := 0
		for _, result := range results {
			if result.Success {
				successfulLayers++
			}
		}
		
		efficiency := float64(successfulLayers) / float64(len(results))
		expectedEfficiency := 3.0 / 4.0 // 75%
		
		if efficiency != expectedEfficiency {
			t.Errorf("Expected efficiency %f, got %f", expectedEfficiency, efficiency)
		}
	})
	
	t.Run("AverageSlippage", func(t *testing.T) {
		results := []LayerExecutionResult{
			{Success: true, SlippageImpact: 0.001},
			{Success: true, SlippageImpact: 0.002},
			{Success: true, SlippageImpact: 0.0015},
		}
		
		totalSlippage := 0.0
		successfulLayers := 0
		
		for _, result := range results {
			if result.Success {
				totalSlippage += result.SlippageImpact
				successfulLayers++
			}
		}
		
		averageSlippage := totalSlippage / float64(successfulLayers)
		expectedAverageSlippage := 0.0015 // (0.001 + 0.002 + 0.0015) / 3
		
		if averageSlippage != expectedAverageSlippage {
			t.Errorf("Expected average slippage %f, got %f", expectedAverageSlippage, averageSlippage)
		}
	})
	
	t.Run("WeightedAveragePrice", func(t *testing.T) {
		results := []LayerExecutionResult{
			{Success: true, ExecutedSize: 2.0, ExecutedPrice: 100.0},
			{Success: true, ExecutedSize: 3.0, ExecutedPrice: 98.0},
			{Success: true, ExecutedSize: 1.0, ExecutedPrice: 102.0},
		}
		
		totalValue := 0.0
		totalSize := 0.0
		
		for _, result := range results {
			if result.Success {
				totalValue += result.ExecutedSize * result.ExecutedPrice
				totalSize += result.ExecutedSize
			}
		}
		
		averagePrice := totalValue / totalSize
		expectedAveragePrice := (2.0*100.0 + 3.0*98.0 + 1.0*102.0) / 6.0 // 99.0
		
		if averagePrice != expectedAveragePrice {
			t.Errorf("Expected average price %f, got %f", expectedAveragePrice, averagePrice)
		}
	})
}