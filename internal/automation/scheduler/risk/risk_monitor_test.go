package risk

import (
	"context"
	"fmt"
	"testing"

	"qcat/internal/automation/scheduler/shared"
	"qcat/internal/config"
	"qcat/internal/database"
	"qcat/internal/exchange/account"

	"github.com/stretchr/testify/assert"
)

// createTestRiskMonitor creates a test risk monitor instance
func createTestRiskMonitor() *RiskMonitor {
	// Create minimal test configuration
	cfg := &config.Config{}

	// Create test database
	db := &database.DB{}

	// Create test account manager
	accountManager := &account.Manager{}

	return NewRiskMonitor(cfg, db, accountManager)
}

func TestNewRiskMonitor(t *testing.T) {
	rm := createTestRiskMonitor()

	assert.NotNil(t, rm)
	assert.NotNil(t, rm.config)
	assert.NotNil(t, rm.db)
	assert.NotNil(t, rm.accountManager)
	assert.NotNil(t, rm.configManager)
	assert.NotNil(t, rm.errorHandler)
	assert.NotNil(t, rm.metrics)
	assert.False(t, rm.isRunning)
}

func TestRiskMonitor_CheckMarginRatio(t *testing.T) {
	rm := createTestRiskMonitor()
	ctx := context.Background()

	status, err := rm.CheckMarginRatio(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, 100000.0, status.TotalEquity)
	assert.Equal(t, 50000.0, status.UsedMargin)
	assert.Equal(t, 50000.0, status.AvailableMargin)
	assert.Equal(t, 2.0, status.MarginRatio)
	assert.Equal(t, shared.RiskLevelLow, status.RiskLevel)
	assert.NotEmpty(t, status.Recommendations)
}

func TestRiskMonitor_DetermineMarginRiskLevel(t *testing.T) {
	rm := createTestRiskMonitor()

	tests := []struct {
		name         string
		marginRatio  float64
		expectedRisk shared.RiskLevel
	}{
		{
			name:         "Low risk",
			marginRatio:  0.3,
			expectedRisk: shared.RiskLevelLow,
		},
		{
			name:         "Medium risk",
			marginRatio:  0.6,
			expectedRisk: shared.RiskLevelMedium,
		},
		{
			name:         "High risk",
			marginRatio:  0.8,
			expectedRisk: shared.RiskLevelHigh,
		},
		{
			name:         "Critical risk",
			marginRatio:  0.9,
			expectedRisk: shared.RiskLevelCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			riskLevel := rm.determineMarginRiskLevel(tt.marginRatio)
			assert.Equal(t, tt.expectedRisk, riskLevel)
		})
	}
}

func TestRiskMonitor_GenerateMarginRecommendations(t *testing.T) {
	rm := createTestRiskMonitor()

	tests := []struct {
		name         string
		marginRatio  float64
		riskLevel    shared.RiskLevel
		expectUrgent bool
	}{
		{
			name:         "Critical risk recommendations",
			marginRatio:  0.9,
			riskLevel:    shared.RiskLevelCritical,
			expectUrgent: true,
		},
		{
			name:         "High risk recommendations",
			marginRatio:  0.8,
			riskLevel:    shared.RiskLevelHigh,
			expectUrgent: false,
		},
		{
			name:         "Low risk recommendations",
			marginRatio:  0.3,
			riskLevel:    shared.RiskLevelLow,
			expectUrgent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recommendations := rm.generateMarginRecommendations(tt.marginRatio, tt.riskLevel)
			assert.NotEmpty(t, recommendations)

			if tt.expectUrgent {
				found := false
				for _, rec := range recommendations {
					if len(rec) > 6 && rec[:6] == "URGENT" {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected urgent recommendation for critical risk")
			}
		})
	}
}

func TestRiskMonitor_CalculateConcentrationRisk(t *testing.T) {
	rm := createTestRiskMonitor()

	tests := []struct {
		name      string
		positions []shared.Position
		expected  float64
	}{
		{
			name:      "No positions",
			positions: []shared.Position{},
			expected:  0.0,
		},
		{
			name: "Single position (maximum concentration)",
			positions: []shared.Position{
				{Size: 100, CurrentPrice: 50000},
			},
			expected: 1.0,
		},
		{
			name: "Two equal positions",
			positions: []shared.Position{
				{Size: 100, CurrentPrice: 50000},
				{Size: 100, CurrentPrice: 50000},
			},
			expected: 0.5, // HHI = 0.5^2 + 0.5^2 = 0.5
		},
		{
			name: "Unequal positions",
			positions: []shared.Position{
				{Size: 150, CurrentPrice: 50000}, // 75% of portfolio
				{Size: 50, CurrentPrice: 50000},  // 25% of portfolio
			},
			expected: 0.625, // HHI = 0.75^2 + 0.25^2 = 0.5625 + 0.0625 = 0.625
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			risk := rm.calculateConcentrationRisk(tt.positions)
			assert.InDelta(t, tt.expected, risk, 0.001)
		})
	}
}

func TestRiskMonitor_CalculateSymbolLiquidityRisk(t *testing.T) {
	rm := createTestRiskMonitor()

	tests := []struct {
		name     string
		symbol   string
		expected float64
	}{
		{
			name:     "Major pair - BTCUSDT",
			symbol:   "BTCUSDT",
			expected: 0.1,
		},
		{
			name:     "Major pair - ETHUSDT",
			symbol:   "ETHUSDT",
			expected: 0.1,
		},
		{
			name:     "Minor pair",
			symbol:   "XYZUSDT",
			expected: 0.3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			risk := rm.calculateSymbolLiquidityRisk(tt.symbol)
			assert.Equal(t, tt.expected, risk)
		})
	}
}

func TestRiskMonitor_DetectVolatilitySpike(t *testing.T) {
	rm := createTestRiskMonitor()

	tests := []struct {
		name        string
		marketData  []MarketData
		expectSpike bool
	}{
		{
			name:        "No data",
			marketData:  []MarketData{},
			expectSpike: false,
		},
		{
			name: "Normal volatility",
			marketData: []MarketData{
				{Symbol: "BTCUSDT", Volatility: 0.02},
				{Symbol: "ETHUSDT", Volatility: 0.025},
				{Symbol: "BNBUSDT", Volatility: 0.03},
			},
			expectSpike: false,
		},
		{
			name: "Volatility spike",
			marketData: []MarketData{
				{Symbol: "BTCUSDT", Volatility: 0.02},
				{Symbol: "ETHUSDT", Volatility: 0.025},
				{Symbol: "BNBUSDT", Volatility: 1.0}, // Extreme volatility spike to ensure detection
			},
			expectSpike: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			anomaly := rm.detectVolatilitySpike(tt.marketData)
			if tt.expectSpike {
				assert.NotNil(t, anomaly)
				assert.Equal(t, shared.AnomalyTypeVolatilitySpike, anomaly.AnomalyType)
				assert.NotEmpty(t, anomaly.AffectedSymbols)
			} else {
				assert.Nil(t, anomaly)
			}
		})
	}
}

func TestRiskMonitor_DetectLiquidityDrop(t *testing.T) {
	rm := createTestRiskMonitor()

	tests := []struct {
		name       string
		marketData []MarketData
		expectDrop bool
	}{
		{
			name:       "No data",
			marketData: []MarketData{},
			expectDrop: false,
		},
		{
			name: "Normal liquidity",
			marketData: []MarketData{
				{Symbol: "BTCUSDT", Liquidity: 0.8},
				{Symbol: "ETHUSDT", Liquidity: 0.75},
				{Symbol: "BNBUSDT", Liquidity: 0.7},
			},
			expectDrop: false,
		},
		{
			name: "Liquidity drop",
			marketData: []MarketData{
				{Symbol: "BTCUSDT", Liquidity: 0.8},
				{Symbol: "ETHUSDT", Liquidity: 0.75},
				{Symbol: "BNBUSDT", Liquidity: 0.1}, // Low liquidity
			},
			expectDrop: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			anomaly := rm.detectLiquidityDrop(tt.marketData)
			if tt.expectDrop {
				assert.NotNil(t, anomaly)
				assert.Equal(t, shared.AnomalyTypeLiquidityDrop, anomaly.AnomalyType)
				assert.NotEmpty(t, anomaly.AffectedSymbols)
			} else {
				assert.Nil(t, anomaly)
			}
		})
	}
}

func TestRiskMonitor_StartStop(t *testing.T) {
	rm := createTestRiskMonitor()

	// Initially not running
	assert.False(t, rm.IsRunning())

	// Start
	err := rm.Start()
	assert.NoError(t, err)
	assert.True(t, rm.IsRunning())

	// Stop
	err = rm.Stop()
	assert.NoError(t, err)
	assert.False(t, rm.IsRunning())
}

func TestRiskMonitor_GetMetrics(t *testing.T) {
	rm := createTestRiskMonitor()

	// Initially empty metrics
	metrics := rm.GetMetrics()
	assert.NotNil(t, metrics)
	assert.Empty(t, metrics)

	// Update some metrics
	rm.metrics["test_metric"] = 123.45
	rm.metrics["test_string"] = "test_value"

	// Get metrics should return a copy
	metrics = rm.GetMetrics()
	assert.Equal(t, 123.45, metrics["test_metric"])
	assert.Equal(t, "test_value", metrics["test_string"])

	// Modifying returned metrics should not affect internal metrics
	metrics["new_metric"] = "new_value"
	internalMetrics := rm.GetMetrics()
	_, exists := internalMetrics["new_metric"]
	assert.False(t, exists)
}

// Benchmark tests
func BenchmarkRiskMonitor_CalculateConcentrationRisk(b *testing.B) {
	rm := createTestRiskMonitor()

	// Create test positions
	positions := make([]shared.Position, 100)
	for i := range positions {
		positions[i] = shared.Position{
			Size:         float64(i + 1),
			CurrentPrice: 50000.0,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rm.calculateConcentrationRisk(positions)
	}
}

func BenchmarkRiskMonitor_DetectVolatilitySpike(b *testing.B) {
	rm := createTestRiskMonitor()

	// Create test market data
	marketData := make([]MarketData, 50)
	for i := range marketData {
		marketData[i] = MarketData{
			Symbol:     fmt.Sprintf("SYMBOL%d", i),
			Volatility: 0.02 + float64(i)*0.001,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rm.detectVolatilitySpike(marketData)
	}
}
