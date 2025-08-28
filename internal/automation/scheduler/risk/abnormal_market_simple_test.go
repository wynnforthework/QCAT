package risk

import (
	"testing"
	"time"

	"qcat/internal/automation/scheduler/shared"

	"github.com/stretchr/testify/assert"
)

func TestCircuitBreakerConfig(t *testing.T) {
	config := CircuitBreakerConfig{
		VolatilityThreshold:    2.0,
		LiquidityThreshold:     0.5,
		CorrelationThreshold:   0.3,
		PriceChangeThreshold:   0.1,
		VolumeChangeThreshold:  3.0,
		ActivationDuration:     time.Minute * 15,
	}

	assert.Equal(t, 2.0, config.VolatilityThreshold)
	assert.Equal(t, 0.5, config.LiquidityThreshold)
	assert.Equal(t, 0.3, config.CorrelationThreshold)
	assert.Equal(t, 0.1, config.PriceChangeThreshold)
	assert.Equal(t, 3.0, config.VolumeChangeThreshold)
	assert.Equal(t, time.Minute*15, config.ActivationDuration)
}

func TestVolatilityAlert(t *testing.T) {
	alert := VolatilityAlert{
		Symbol:          "BTC/USDT",
		CurrentVol:      0.05,
		HistoricalVol:   0.02,
		VolRatio:        2.5,
		Severity:        shared.AlertSeverityHigh,
		DetectedAt:      time.Now(),
		Recommendations: []string{"Reduce position sizes", "Monitor closely"},
	}

	assert.Equal(t, "BTC/USDT", alert.Symbol)
	assert.Equal(t, 0.05, alert.CurrentVol)
	assert.Equal(t, 0.02, alert.HistoricalVol)
	assert.Equal(t, 2.5, alert.VolRatio)
	assert.Equal(t, shared.AlertSeverityHigh, alert.Severity)
	assert.Len(t, alert.Recommendations, 2)
}

func TestLiquidityAlert(t *testing.T) {
	alert := LiquidityAlert{
		Symbol:              "ETH/USDT",
		CurrentLiquidity:    5.0,
		HistoricalLiquidity: 15.0,
		LiquidityRatio:      0.33,
		BidAskSpread:        0.002,
		Severity:            shared.AlertSeverityMedium,
		DetectedAt:          time.Now(),
		Recommendations:     []string{"Use limit orders", "Reduce order sizes"},
	}

	assert.Equal(t, "ETH/USDT", alert.Symbol)
	assert.Equal(t, 5.0, alert.CurrentLiquidity)
	assert.Equal(t, 15.0, alert.HistoricalLiquidity)
	assert.Equal(t, 0.33, alert.LiquidityRatio)
	assert.Equal(t, 0.002, alert.BidAskSpread)
	assert.Equal(t, shared.AlertSeverityMedium, alert.Severity)
	assert.Len(t, alert.Recommendations, 2)
}

func TestCorrelationAlert(t *testing.T) {
	alert := CorrelationAlert{
		AssetPairs:      []string{"BTC/USDT", "ETH/USDT"},
		CurrentCorr:     0.2,
		HistoricalCorr:  0.8,
		CorrChange:      0.6,
		Severity:        shared.AlertSeverityCritical,
		DetectedAt:      time.Now(),
		Recommendations: []string{"Review hedging strategies", "Update correlation matrices"},
	}

	assert.Len(t, alert.AssetPairs, 2)
	assert.Contains(t, alert.AssetPairs, "BTC/USDT")
	assert.Contains(t, alert.AssetPairs, "ETH/USDT")
	assert.Equal(t, 0.2, alert.CurrentCorr)
	assert.Equal(t, 0.8, alert.HistoricalCorr)
	assert.Equal(t, 0.6, alert.CorrChange)
	assert.Equal(t, shared.AlertSeverityCritical, alert.Severity)
	assert.Len(t, alert.Recommendations, 2)
}

func TestPositionScalingConfig(t *testing.T) {
	config := PositionScalingConfig{
		ScalingFactor:   0.7,
		MaxReduction:    0.5,
		MinPositionSize: 100.0,
		ExcludedSymbols: []string{"BTC/USDT", "ETH/USDT"},
		Priority:        "HIGH_RISK",
		ExecutionMethod: "GRADUAL",
	}

	assert.Equal(t, 0.7, config.ScalingFactor)
	assert.Equal(t, 0.5, config.MaxReduction)
	assert.Equal(t, 100.0, config.MinPositionSize)
	assert.Len(t, config.ExcludedSymbols, 2)
	assert.Equal(t, "HIGH_RISK", config.Priority)
	assert.Equal(t, "GRADUAL", config.ExecutionMethod)
}

func TestEmergencyHedgingConfig(t *testing.T) {
	config := EmergencyHedgingConfig{
		HedgeRatio:       0.3,
		HedgeInstruments: []string{"BTC_PERP", "ETH_PERP"},
		MaxHedgeSize:     100000.0,
		HedgeMethod:      "FUTURES",
		Correlations: map[string]float64{
			"BTC/USDT": -0.8,
			"ETH/USDT": -0.7,
		},
	}

	assert.Equal(t, 0.3, config.HedgeRatio)
	assert.Len(t, config.HedgeInstruments, 2)
	assert.Equal(t, 100000.0, config.MaxHedgeSize)
	assert.Equal(t, "FUTURES", config.HedgeMethod)
	assert.Equal(t, -0.8, config.Correlations["BTC/USDT"])
	assert.Equal(t, -0.7, config.Correlations["ETH/USDT"])
}

func TestFundingRateConfig(t *testing.T) {
	config := FundingRateConfig{
		PositiveThreshold: 0.01,
		NegativeThreshold: -0.01,
		AdjustmentFactor:  0.2,
		MonitoringPairs:   []string{"BTC/USDT", "ETH/USDT"},
	}

	assert.Equal(t, 0.01, config.PositiveThreshold)
	assert.Equal(t, -0.01, config.NegativeThreshold)
	assert.Equal(t, 0.2, config.AdjustmentFactor)
	assert.Len(t, config.MonitoringPairs, 2)
}

func TestScaledPositionInfo(t *testing.T) {
	info := ScaledPositionInfo{
		PositionID:      "pos123",
		Symbol:          "BTC/USDT",
		OriginalSize:    1.0,
		NewSize:         0.7,
		ReductionAmount: 0.3,
		ReductionRatio:  0.3,
		ExecutionPrice:  50000.0,
		Status:          "COMPLETED",
	}

	assert.Equal(t, "pos123", info.PositionID)
	assert.Equal(t, "BTC/USDT", info.Symbol)
	assert.Equal(t, 1.0, info.OriginalSize)
	assert.Equal(t, 0.7, info.NewSize)
	assert.Equal(t, 0.3, info.ReductionAmount)
	assert.Equal(t, 0.3, info.ReductionRatio)
	assert.Equal(t, 50000.0, info.ExecutionPrice)
	assert.Equal(t, "COMPLETED", info.Status)
}

func TestHedgePositionInfo(t *testing.T) {
	info := HedgePositionInfo{
		HedgeID:         "hedge123",
		Instrument:      "BTC_PERP",
		HedgeSize:       0.5,
		HedgePrice:      51000.0,
		TargetAsset:     "BTC/USDT",
		HedgeRatio:      0.3,
		ExpectedOffset:  15000.0,
		Status:          "ACTIVE",
	}

	assert.Equal(t, "hedge123", info.HedgeID)
	assert.Equal(t, "BTC_PERP", info.Instrument)
	assert.Equal(t, 0.5, info.HedgeSize)
	assert.Equal(t, 51000.0, info.HedgePrice)
	assert.Equal(t, "BTC/USDT", info.TargetAsset)
	assert.Equal(t, 0.3, info.HedgeRatio)
	assert.Equal(t, 15000.0, info.ExpectedOffset)
	assert.Equal(t, "ACTIVE", info.Status)
}

func TestFundingRateAdjustment(t *testing.T) {
	adjustment := FundingRateAdjustment{
		Symbol:          "BTC/USDT",
		CurrentFunding:  0.015,
		ThresholdType:   "POSITIVE",
		OriginalSize:    1.0,
		AdjustedSize:    0.8,
		AdjustmentRatio: 0.8,
		Rationale:       "High positive funding rate",
	}

	assert.Equal(t, "BTC/USDT", adjustment.Symbol)
	assert.Equal(t, 0.015, adjustment.CurrentFunding)
	assert.Equal(t, "POSITIVE", adjustment.ThresholdType)
	assert.Equal(t, 1.0, adjustment.OriginalSize)
	assert.Equal(t, 0.8, adjustment.AdjustedSize)
	assert.Equal(t, 0.8, adjustment.AdjustmentRatio)
	assert.Contains(t, adjustment.Rationale, "positive funding rate")
}

func TestOrderBook(t *testing.T) {
	orderBook := OrderBook{
		Symbol: "BTC/USDT",
		Bids: []PriceLevel{
			{Price: 50000.0, Quantity: 1.0},
			{Price: 49990.0, Quantity: 0.5},
		},
		Asks: []PriceLevel{
			{Price: 50010.0, Quantity: 1.0},
			{Price: 50020.0, Quantity: 0.5},
		},
		Timestamp: time.Now(),
	}

	assert.Equal(t, "BTC/USDT", orderBook.Symbol)
	assert.Len(t, orderBook.Bids, 2)
	assert.Len(t, orderBook.Asks, 2)
	assert.Equal(t, 50000.0, orderBook.Bids[0].Price)
	assert.Equal(t, 1.0, orderBook.Bids[0].Quantity)
	assert.Equal(t, 50010.0, orderBook.Asks[0].Price)
	assert.Equal(t, 1.0, orderBook.Asks[0].Quantity)
}

func TestPortfolioExposure(t *testing.T) {
	exposure := PortfolioExposure{
		TotalExposure: 100000.0,
		AssetExposures: map[string]float64{
			"BTC/USDT": 60000.0,
			"ETH/USDT": 40000.0,
		},
		SectorExposures: map[string]float64{
			"CRYPTO": 100000.0,
		},
		RegionExposures: map[string]float64{
			"GLOBAL": 100000.0,
		},
		LeverageRatio:     2.5,
		ConcentrationRisk: 0.52, // 0.6^2 + 0.4^2 = 0.36 + 0.16 = 0.52
	}

	assert.Equal(t, 100000.0, exposure.TotalExposure)
	assert.Equal(t, 60000.0, exposure.AssetExposures["BTC/USDT"])
	assert.Equal(t, 40000.0, exposure.AssetExposures["ETH/USDT"])
	assert.Equal(t, 2.5, exposure.LeverageRatio)
	assert.Equal(t, 0.52, exposure.ConcentrationRisk)
}

func TestExchangeHealthStatus(t *testing.T) {
	status := ExchangeHealthStatus{
		Exchange:    "binance",
		HealthScore: 0.95,
		Latency:     50.0,
		ErrorRate:   0.01,
		Uptime:      99.9,
		LastCheck:   time.Now(),
		Issues:      []string{},
	}

	assert.Equal(t, "binance", status.Exchange)
	assert.Equal(t, 0.95, status.HealthScore)
	assert.Equal(t, 50.0, status.Latency)
	assert.Equal(t, 0.01, status.ErrorRate)
	assert.Equal(t, 99.9, status.Uptime)
	assert.Empty(t, status.Issues)
}

// Test helper functions

func TestCalculateReturns(t *testing.T) {
	// Create a mock detector to test the helper method
	detector := &AbnormalMarketDetector{}
	
	prices := []float64{100.0, 102.0, 101.0, 103.0, 105.0}
	returns := detector.calculateReturns(prices)
	
	assert.Len(t, returns, 4) // Should be len(prices) - 1
	
	// Verify first return: ln(102/100) ≈ 0.0198
	assert.InDelta(t, 0.0198, returns[0], 0.001)
	
	// Verify second return: ln(101/102) ≈ -0.0099
	assert.InDelta(t, -0.0099, returns[1], 0.001)
}

func TestDetermineVolatilitySeverity(t *testing.T) {
	detector := &AbnormalMarketDetector{}
	
	// Test different volatility ratios
	assert.Equal(t, shared.AlertSeverityCritical, detector.determineVolatilitySeverity(5.5))
	assert.Equal(t, shared.AlertSeverityHigh, detector.determineVolatilitySeverity(3.5))
	assert.Equal(t, shared.AlertSeverityMedium, detector.determineVolatilitySeverity(2.5))
	assert.Equal(t, shared.AlertSeverityLow, detector.determineVolatilitySeverity(1.5))
}

func TestDetermineLiquiditySeverity(t *testing.T) {
	detector := &AbnormalMarketDetector{}
	
	// Test critical liquidity (very low ratio and high spread)
	assert.Equal(t, shared.AlertSeverityCritical, detector.determineLiquiditySeverity(0.1, 0.06))
	
	// Test high severity (low ratio)
	assert.Equal(t, shared.AlertSeverityHigh, detector.determineLiquiditySeverity(0.3, 0.04))
	
	// Test medium severity
	assert.Equal(t, shared.AlertSeverityMedium, detector.determineLiquiditySeverity(0.5, 0.025))
	
	// Test low severity (good liquidity)
	assert.Equal(t, shared.AlertSeverityLow, detector.determineLiquiditySeverity(0.8, 0.01))
}

func TestDetermineCorrelationSeverity(t *testing.T) {
	detector := &AbnormalMarketDetector{}
	
	// Test critical correlation breakdown
	assert.Equal(t, shared.AlertSeverityCritical, detector.determineCorrelationSeverity(0.8, 0.2, 0.9))
	
	// Test high severity
	assert.Equal(t, shared.AlertSeverityHigh, detector.determineCorrelationSeverity(0.6, 0.3, 0.8))
	
	// Test medium severity
	assert.Equal(t, shared.AlertSeverityMedium, detector.determineCorrelationSeverity(0.4, 0.4, 0.7))
	
	// Test low severity
	assert.Equal(t, shared.AlertSeverityLow, detector.determineCorrelationSeverity(0.2, 0.6, 0.7))
}

func TestGenerateVolatilityRecommendations(t *testing.T) {
	detector := &AbnormalMarketDetector{}
	
	// Test critical recommendations
	recommendations := detector.generateVolatilityRecommendations("BTC/USDT", 5.0, shared.AlertSeverityCritical)
	assert.NotEmpty(t, recommendations)
	assert.Contains(t, recommendations[0], "URGENT")
	assert.Contains(t, recommendations[0], "5.0x normal")
	
	// Test high severity recommendations
	recommendations = detector.generateVolatilityRecommendations("ETH/USDT", 3.0, shared.AlertSeverityHigh)
	assert.NotEmpty(t, recommendations)
	assert.Contains(t, recommendations[0], "High volatility")
	assert.Contains(t, recommendations[0], "3.0x normal")
}

func TestGenerateLiquidityRecommendations(t *testing.T) {
	detector := &AbnormalMarketDetector{}
	
	// Test critical recommendations
	recommendations := detector.generateLiquidityRecommendations("BTC/USDT", 0.2, shared.AlertSeverityCritical)
	assert.NotEmpty(t, recommendations)
	assert.Contains(t, recommendations[0], "CRITICAL")
	assert.Contains(t, recommendations[0], "20.0% of normal")
	
	// Test medium severity recommendations
	recommendations = detector.generateLiquidityRecommendations("ETH/USDT", 0.6, shared.AlertSeverityMedium)
	assert.NotEmpty(t, recommendations)
	assert.Contains(t, recommendations[0], "Reduced liquidity")
	assert.Contains(t, recommendations[0], "60.0% of normal")
}

func TestGenerateCorrelationRecommendations(t *testing.T) {
	detector := &AbnormalMarketDetector{}
	
	pair := []string{"BTC/USDT", "ETH/USDT"}
	
	// Test critical recommendations
	recommendations := detector.generateCorrelationRecommendations(pair, 0.7, shared.AlertSeverityCritical)
	assert.NotEmpty(t, recommendations)
	assert.Contains(t, recommendations[0], "CRITICAL")
	assert.Contains(t, recommendations[0], "BTC/USDT/ETH/USDT")
	assert.Contains(t, recommendations[0], "70.0% change")
	
	// Test medium severity recommendations
	recommendations = detector.generateCorrelationRecommendations(pair, 0.4, shared.AlertSeverityMedium)
	assert.NotEmpty(t, recommendations)
	assert.Contains(t, recommendations[0], "Correlation shift")
	assert.Contains(t, recommendations[0], "40.0% change")
}