package position

import (
	"context"
	"testing"
	"time"

	"qcat/internal/automation/scheduler/shared"
	"qcat/internal/database"
	"qcat/internal/exchange"
	"qcat/internal/logger"
)

// Mock implementations for testing
type mockExchangeClient struct{}

func (m *mockExchangeClient) GetCurrentPrice(ctx context.Context, symbol string) (float64, error) {
	return 100.0, nil
}

func (m *mockExchangeClient) GetHistoricalPrices(ctx context.Context, symbol string, period time.Duration) ([]float64, error) {
	// Return mock historical prices
	prices := []float64{95.0, 98.0, 102.0, 99.0, 101.0, 103.0, 100.0}
	return prices, nil
}

// Add missing methods to satisfy exchange.Exchange interface
func (m *mockExchangeClient) GetExchangeInfo(ctx context.Context) (*exchange.ExchangeInfo, error) {
	return nil, nil
}
func (m *mockExchangeClient) GetSymbolInfo(ctx context.Context, symbol string) (*exchange.SymbolInfo, error) {
	return nil, nil
}
func (m *mockExchangeClient) GetServerTime(ctx context.Context) (time.Time, error) {
	return time.Now(), nil
}
func (m *mockExchangeClient) GetAccountBalance(ctx context.Context) (map[string]*exchange.AccountBalance, error) {
	return nil, nil
}
func (m *mockExchangeClient) GetPositions(ctx context.Context) ([]*exchange.Position, error) {
	return nil, nil
}
func (m *mockExchangeClient) GetPosition(ctx context.Context, symbol string) (*exchange.Position, error) {
	return nil, nil
}
func (m *mockExchangeClient) GetLeverage(ctx context.Context, symbol string) (int, error) {
	return 1, nil
}
func (m *mockExchangeClient) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	return nil
}
func (m *mockExchangeClient) SetMarginType(ctx context.Context, symbol string, marginType exchange.MarginType) error {
	return nil
}
func (m *mockExchangeClient) PlaceOrder(ctx context.Context, req *exchange.OrderRequest) (*exchange.OrderResponse, error) {
	return nil, nil
}
func (m *mockExchangeClient) CancelOrder(ctx context.Context, req *exchange.OrderCancelRequest) (*exchange.OrderResponse, error) {
	return nil, nil
}
func (m *mockExchangeClient) CancelAllOrders(ctx context.Context, symbol string) error { return nil }
func (m *mockExchangeClient) GetOrder(ctx context.Context, symbol, orderID string) (*exchange.Order, error) {
	return nil, nil
}
func (m *mockExchangeClient) GetOpenOrders(ctx context.Context, symbol string) ([]*exchange.Order, error) {
	return nil, nil
}
func (m *mockExchangeClient) GetOrderHistory(ctx context.Context, symbol string, startTime, endTime time.Time) ([]*exchange.Order, error) {
	return nil, nil
}
func (m *mockExchangeClient) GetRiskLimits(ctx context.Context, symbol string) (*exchange.RiskLimits, error) {
	return nil, nil
}
func (m *mockExchangeClient) GetMarginInfo(ctx context.Context) (*exchange.MarginInfo, error) {
	return nil, nil
}
func (m *mockExchangeClient) SetRiskLimits(ctx context.Context, symbol string, limits *exchange.RiskLimits) error {
	return nil
}
func (m *mockExchangeClient) GetPositionByID(ctx context.Context, positionID string) (*exchange.Position, error) {
	return nil, nil
}
func (m *mockExchangeClient) GetSymbolPrice(ctx context.Context, symbol string) (float64, error) {
	return 100.0, nil
}
func (m *mockExchangeClient) GetAccount(ctx context.Context) (*exchange.Account, error) {
	return nil, nil
}
func (m *mockExchangeClient) GetTicker(ctx context.Context, symbol string) (*exchange.Ticker, error) {
	return nil, nil
}
func (m *mockExchangeClient) GetOrderBook(ctx context.Context, symbol string, limit int) (*exchange.OrderBook, error) {
	return nil, nil
}
func (m *mockExchangeClient) GetAccountSnapshots(ctx context.Context, days int) ([]*exchange.AccountSnapshot, error) {
	return nil, nil
}
func (m *mockExchangeClient) Get24HrStats(ctx context.Context, symbol string) (*exchange.Stats24Hr, error) {
	return nil, nil
}

func newMockDB() *database.DB {
	// Return nil for testing - the actual implementation will handle nil checks
	return nil
}

type mockLogger struct{}

func (m *mockLogger) Trace(msg string, fields ...interface{})                {}
func (m *mockLogger) Debug(msg string, fields ...interface{})                {}
func (m *mockLogger) Info(msg string, fields ...interface{})                 {}
func (m *mockLogger) Warn(msg string, fields ...interface{})                 {}
func (m *mockLogger) Error(msg string, fields ...interface{})                {}
func (m *mockLogger) Fatal(msg string, fields ...interface{})                {}
func (m *mockLogger) Panic(msg string, fields ...interface{})                {}
func (m *mockLogger) WithField(key string, value interface{}) logger.Logger  { return m }
func (m *mockLogger) WithFields(fields map[string]interface{}) logger.Logger { return m }
func (m *mockLogger) WithContext(ctx context.Context) logger.Logger          { return m }
func (m *mockLogger) SetLevel(level logger.LogLevel)                         {}
func (m *mockLogger) GetLevel() logger.LogLevel                              { return logger.LevelInfo }

type mockConfig struct {
	values map[string]interface{}
}

func newMockConfig() *mockConfig {
	return &mockConfig{
		values: map[string]interface{}{
			"layered_position.max_layers":              10,
			"layered_position.min_layer_size":          0.1,
			"layered_position.max_layer_size":          10.0,
			"layered_position.volatility_window":       30,
			"layered_position.rebalance_threshold":     0.05,
			"layered_position.risk_adjustment_factor":  0.2,
			"layered_execution.max_concurrent":         5,
			"layered_execution.timeout":                time.Minute * 30,
			"layered_execution.retry_attempts":         3,
			"layered_execution.slippage_tolerance":     0.01,
			"layered_execution.partial_fill_threshold": 0.8,
			"layered_execution.min_closure_size":       0.01,
			"layered_execution.layer_delay":            time.Second * 5,
			"performance.risk_free_rate":               0.02,
		},
	}
}

func (m *mockConfig) Get(key string) interface{} {
	return m.values[key]
}

func (m *mockConfig) GetString(key string) string {
	if v, ok := m.values[key].(string); ok {
		return v
	}
	return ""
}

func (m *mockConfig) GetInt(key string) int {
	if v, ok := m.values[key].(int); ok {
		return v
	}
	return 0
}

func (m *mockConfig) GetFloat64(key string) float64 {
	if v, ok := m.values[key].(float64); ok {
		return v
	}
	return 0.0
}

func (m *mockConfig) GetBool(key string) bool {
	if v, ok := m.values[key].(bool); ok {
		return v
	}
	return false
}

func (m *mockConfig) GetDuration(key string) time.Duration {
	if v, ok := m.values[key].(time.Duration); ok {
		return v
	}
	return 0
}

func (m *mockConfig) Set(key string, value interface{}) error {
	m.values[key] = value
	return nil
}

func (m *mockConfig) Reload() error {
	return nil
}

func TestLayeredPositionManager_AnalyzeMarketVolatility(t *testing.T) {
	// Setup
	db := newMockDB()
	exchangeClient := &mockExchangeClient{}
	logger := &mockLogger{}
	config := newMockConfig()

	lpm := NewLayeredPositionManager(db, exchangeClient, logger, config)

	ctx := context.Background()
	symbol := "BTCUSDT"

	// Test volatility analysis
	analysis, err := lpm.AnalyzeMarketVolatility(ctx, symbol)

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if analysis == nil {
		t.Fatal("Expected analysis result, got nil")
	}

	if analysis.Symbol != symbol {
		t.Errorf("Expected symbol %s, got %s", symbol, analysis.Symbol)
	}

	if analysis.ShortTermVolatility <= 0 {
		t.Error("Expected positive short-term volatility")
	}

	if analysis.MediumTermVolatility <= 0 {
		t.Error("Expected positive medium-term volatility")
	}

	if analysis.LongTermVolatility <= 0 {
		t.Error("Expected positive long-term volatility")
	}

	if analysis.RecommendedLayers <= 0 {
		t.Error("Expected positive number of recommended layers")
	}

	if analysis.OptimalLayerSize <= 0 {
		t.Error("Expected positive optimal layer size")
	}

	if analysis.Confidence < 0 || analysis.Confidence > 1 {
		t.Errorf("Expected confidence between 0 and 1, got %f", analysis.Confidence)
	}

	// Test volatility trend classification
	validTrends := []string{"INCREASING", "DECREASING", "STABLE"}
	found := false
	for _, trend := range validTrends {
		if analysis.VolatilityTrend == trend {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected valid volatility trend, got %s", analysis.VolatilityTrend)
	}

	// Test volatility regime classification
	validRegimes := []string{"LOW", "MEDIUM", "HIGH", "EXTREME"}
	found = false
	for _, regime := range validRegimes {
		if analysis.VolatilityRegime == regime {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected valid volatility regime, got %s", analysis.VolatilityRegime)
	}
}

func TestLayeredPositionManager_CalculateLayerConfiguration(t *testing.T) {
	// Setup
	db := newMockDB()
	exchangeClient := &mockExchangeClient{}
	logger := &mockLogger{}
	config := newMockConfig()

	lpm := NewLayeredPositionManager(db, exchangeClient, logger, config)

	ctx := context.Background()

	// Create test strategy
	strategy := &LayeredStrategy{
		ID:              "test_strategy_1",
		Symbol:          "BTCUSDT",
		Direction:       "LONG",
		TotalSize:       10.0,
		MaxLayers:       5,
		LayerSizeRatio:  1.0,
		PriceSpacing:    0.02,
		VolatilityBased: true,
		AdaptiveSpacing: true,
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

	// Test layer configuration calculation
	config_result, err := lpm.CalculateLayerConfiguration(ctx, strategy)

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if config_result == nil {
		t.Fatal("Expected configuration result, got nil")
	}

	if config_result.Symbol != strategy.Symbol {
		t.Errorf("Expected symbol %s, got %s", strategy.Symbol, config_result.Symbol)
	}

	if len(config_result.Layers) == 0 {
		t.Error("Expected at least one layer")
	}

	if len(config_result.Layers) > strategy.MaxLayers {
		t.Errorf("Expected max %d layers, got %d", strategy.MaxLayers, len(config_result.Layers))
	}

	// Test layer properties
	for i, layer := range config_result.Layers {
		if layer.ID == "" {
			t.Errorf("Layer %d has empty ID", i)
		}

		if layer.Level != i+1 {
			t.Errorf("Layer %d has incorrect level %d", i, layer.Level)
		}

		if layer.Size <= 0 {
			t.Errorf("Layer %d has invalid size %f", i, layer.Size)
		}

		if layer.EntryPrice <= 0 {
			t.Errorf("Layer %d has invalid entry price %f", i, layer.EntryPrice)
		}

		if layer.StopLoss <= 0 {
			t.Errorf("Layer %d has invalid stop loss %f", i, layer.StopLoss)
		}

		if layer.TakeProfit <= 0 {
			t.Errorf("Layer %d has invalid take profit %f", i, layer.TakeProfit)
		}

		if layer.Status != "PENDING" {
			t.Errorf("Layer %d has incorrect initial status %s", i, layer.Status)
		}

		// Test price relationships for LONG strategy
		if strategy.Direction == "LONG" {
			if layer.StopLoss >= layer.EntryPrice {
				t.Errorf("Layer %d stop loss should be below entry price for LONG", i)
			}

			if layer.TakeProfit <= layer.EntryPrice {
				t.Errorf("Layer %d take profit should be above entry price for LONG", i)
			}
		}
	}

	// Test total size consistency
	totalLayerSize := 0.0
	for _, layer := range config_result.Layers {
		totalLayerSize += layer.Size
	}

	// Allow for some variance due to volatility adjustments
	if totalLayerSize < strategy.TotalSize*0.8 || totalLayerSize > strategy.TotalSize*1.2 {
		t.Errorf("Total layer size %f significantly different from strategy total size %f", totalLayerSize, strategy.TotalSize)
	}
}

func TestLayeredPositionManager_DetermineOptimalEntryExitPoints(t *testing.T) {
	// Setup
	db := newMockDB()
	exchangeClient := &mockExchangeClient{}
	logger := &mockLogger{}
	config := newMockConfig()

	lpm := NewLayeredPositionManager(db, exchangeClient, logger, config)

	ctx := context.Background()
	symbol := "BTCUSDT"
	direction := "LONG"

	// Create mock volatility analysis
	volatilityAnalysis := &VolatilityAnalysis{
		Symbol:               symbol,
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

	// Test optimal entry/exit point determination
	entryPoints, exitPoints, err := lpm.DetermineOptimalEntryExitPoints(ctx, symbol, direction, volatilityAnalysis)

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(entryPoints) == 0 {
		t.Error("Expected at least one entry point")
	}

	if len(exitPoints) == 0 {
		t.Error("Expected at least one exit point")
	}

	if len(entryPoints) > volatilityAnalysis.RecommendedLayers {
		t.Errorf("Expected max %d entry points, got %d", volatilityAnalysis.RecommendedLayers, len(entryPoints))
	}

	if len(exitPoints) > volatilityAnalysis.RecommendedLayers {
		t.Errorf("Expected max %d exit points, got %d", volatilityAnalysis.RecommendedLayers, len(exitPoints))
	}

	// Test price ordering for LONG direction
	currentPrice := 100.0 // Mock current price

	for i, entryPrice := range entryPoints {
		if entryPrice >= currentPrice {
			t.Errorf("Entry point %d (%f) should be below current price (%f) for LONG", i, entryPrice, currentPrice)
		}

		// For LONG strategy, entry points should be in descending order (higher to lower)
		// This allows buying at progressively lower prices as the market falls
		if i > 0 && entryPrice >= entryPoints[i-1] {
			t.Errorf("Entry points should be in descending order for LONG, but point %d (%f) >= point %d (%f)", i, entryPrice, i-1, entryPoints[i-1])
		}
	}

	for i, exitPrice := range exitPoints {
		if exitPrice <= currentPrice {
			t.Errorf("Exit point %d (%f) should be above current price (%f) for LONG", i, exitPrice, currentPrice)
		}
	}
}

func TestLayeredPositionManager_AdjustLayerParametersDynamically(t *testing.T) {
	// Setup
	db := newMockDB()
	exchangeClient := &mockExchangeClient{}
	logger := &mockLogger{}
	config := newMockConfig()

	lpm := NewLayeredPositionManager(db, exchangeClient, logger, config)

	ctx := context.Background()

	// Create test layer configuration
	layerConfig := &shared.LayerConfig{
		Symbol:    "BTCUSDT",
		TotalSize: 10.0,
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
			{
				ID:         "layer_2",
				Level:      2,
				Size:       2.0,
				EntryPrice: 90.0,
				StopLoss:   85.5,
				TakeProfit: 99.0,
				Status:     "PENDING",
				CreatedAt:  time.Now(),
			},
		},
		RiskParameters: shared.RiskParams{
			StopLossPercent:   0.05,
			TakeProfitPercent: 0.10,
		},
		ExecutionStrategy: "SEQUENTIAL",
	}

	// Create test market conditions
	marketConditions := &shared.MarketConditions{
		Volatility: 0.4, // High volatility
		Trend:      0.2, // Positive trend
		Liquidity:  0.3, // Low liquidity
		Sentiment:  "BULLISH",
		RegimeType: "VOLATILE",
		Timestamp:  time.Now(),
	}

	// Store original values for comparison
	originalStopLoss1 := layerConfig.Layers[0].StopLoss
	originalTakeProfit1 := layerConfig.Layers[0].TakeProfit
	originalSize1 := layerConfig.Layers[0].Size

	// Test dynamic parameter adjustment
	err := lpm.AdjustLayerParametersDynamically(ctx, layerConfig, marketConditions)

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test that active layer parameters were adjusted
	activeLayer := &layerConfig.Layers[0]

	// Stop loss should be adjusted due to high volatility
	if activeLayer.StopLoss == originalStopLoss1 {
		t.Error("Expected stop loss to be adjusted for high volatility")
	}

	// Take profit should be adjusted due to positive trend
	if activeLayer.TakeProfit == originalTakeProfit1 {
		t.Error("Expected take profit to be adjusted for positive trend")
	}

	// Size should be adjusted due to low liquidity
	if activeLayer.Size == originalSize1 {
		t.Error("Expected size to be adjusted for low liquidity")
	}

	// Size should not go below minimum
	minSize := config.GetFloat64("layered_position.min_layer_size")
	if activeLayer.Size < minSize {
		t.Errorf("Layer size %f should not be below minimum %f", activeLayer.Size, minSize)
	}

	// Test that pending layer was not significantly modified (only active layers should be adjusted)
	pendingLayer := &layerConfig.Layers[1]
	if pendingLayer.Status != "PENDING" {
		t.Error("Pending layer status should remain unchanged")
	}
}

func TestLayeredExecutionSystem_ExecuteLayeredPositions(t *testing.T) {
	// Setup
	db := newMockDB()
	exchangeClient := &mockExchangeClient{}
	logger := &mockLogger{}
	config := newMockConfig()

	les := NewLayeredExecutionSystem(db, exchangeClient, logger, config)

	ctx := context.Background()

	// Create test layer configuration
	layerConfig := &shared.LayerConfig{
		Symbol:    "BTCUSDT",
		TotalSize: 6.0,
		Layers: []shared.Layer{
			{
				ID:         "layer_1",
				Level:      1,
				Size:       2.0,
				EntryPrice: 95.0,
				StopLoss:   90.25,
				TakeProfit: 104.5,
				Status:     "PENDING",
				CreatedAt:  time.Now(),
			},
			{
				ID:         "layer_2",
				Level:      2,
				Size:       2.0,
				EntryPrice: 90.0,
				StopLoss:   85.5,
				TakeProfit: 99.0,
				Status:     "PENDING",
				CreatedAt:  time.Now(),
			},
			{
				ID:         "layer_3",
				Level:      3,
				Size:       2.0,
				EntryPrice: 85.0,
				StopLoss:   80.75,
				TakeProfit: 93.5,
				Status:     "PENDING",
				CreatedAt:  time.Now(),
			},
		},
		RiskParameters: shared.RiskParams{
			MaxLeverage:       2.0,
			MaxPositionSize:   20.0,
			StopLossPercent:   0.05,
			TakeProfitPercent: 0.10,
		},
		ExecutionStrategy: "SEQUENTIAL",
	}

	// Test layered position execution
	execution, err := les.ExecuteLayeredPositions(ctx, layerConfig)

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if execution == nil {
		t.Fatal("Expected execution result, got nil")
	}

	if execution.ID == "" {
		t.Error("Expected execution ID to be set")
	}

	if execution.Symbol != layerConfig.Symbol {
		t.Errorf("Expected symbol %s, got %s", layerConfig.Symbol, execution.Symbol)
	}

	if execution.Status != "PENDING" {
		t.Errorf("Expected initial status PENDING, got %s", execution.Status)
	}

	if execution.Config != layerConfig {
		t.Error("Expected config to match input")
	}

	if execution.StartedAt.IsZero() {
		t.Error("Expected started time to be set")
	}

	// Wait a bit for async execution to start
	time.Sleep(time.Millisecond * 100)

	// Check that execution is registered
	les.mu.RLock()
	_, exists := les.activeExecutions[execution.ID]
	les.mu.RUnlock()

	if !exists {
		t.Error("Expected execution to be registered in active executions")
	}
}

func TestLayeredExecutionSystem_ExecutePartialPositionClosure(t *testing.T) {
	// Setup
	db := newMockDB()
	exchangeClient := &mockExchangeClient{}
	logger := &mockLogger{}
	config := newMockConfig()

	les := NewLayeredExecutionSystem(db, exchangeClient, logger, config)

	ctx := context.Background()

	// Create test execution with active layers
	execution := &LayeredExecution{
		ID:         "test_execution_1",
		StrategyID: "test_strategy_1",
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
				{
					ID:         "layer_2",
					Level:      2,
					Size:       2.0,
					EntryPrice: 90.0,
					StopLoss:   85.5,
					TakeProfit: 99.0,
					Status:     "ACTIVE",
					CreatedAt:  time.Now(),
				},
			},
		},
		TotalExecuted: 4.0,
		AveragePrice:  92.5,
	}

	// Register execution
	les.mu.Lock()
	les.activeExecutions[execution.ID] = execution
	les.mu.Unlock()

	// Create partial closure request
	closureRequest := &PartialClosureRequest{
		ExecutionID:       execution.ID,
		LayerIDs:          []string{"layer_1"},
		ClosureReason:     "PROFIT_TAKING",
		ClosurePercentage: 0.5, // Close 50%
		PriceLimit:        105.0,
		TimeLimit:         time.Minute * 5,
		Metadata: map[string]interface{}{
			"trigger": "manual",
		},
	}

	// Test partial position closure
	result, err := les.ExecutePartialPositionClosure(ctx, closureRequest)

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected closure result, got nil")
	}

	if result.RequestID == "" {
		t.Error("Expected request ID to be set")
	}

	if !result.Success {
		t.Error("Expected closure to be successful")
	}

	if len(result.ClosedLayers) == 0 {
		t.Error("Expected at least one closed layer")
	}

	if result.ClosedSize <= 0 {
		t.Error("Expected positive closed size")
	}

	if result.AverageClosePrice <= 0 {
		t.Error("Expected positive average close price")
	}

	if result.ExecutionTime <= 0 {
		t.Error("Expected positive execution time")
	}

	if result.CompletedAt.IsZero() {
		t.Error("Expected completion time to be set")
	}

	// Test that layer status was updated
	layer1 := &execution.Config.Layers[0]
	if layer1.Status != "PARTIALLY_CLOSED" && layer1.Status != "CLOSED" {
		t.Errorf("Expected layer status to be updated, got %s", layer1.Status)
	}

	// Test that layer size was reduced for partial closure
	expectedRemainingSize := 2.0 * (1 - closureRequest.ClosurePercentage)
	if layer1.Status == "PARTIALLY_CLOSED" && layer1.Size != expectedRemainingSize {
		t.Errorf("Expected remaining size %f, got %f", expectedRemainingSize, layer1.Size)
	}
}

func TestLayeredExecutionSystem_TrackLayeredPositionPerformance(t *testing.T) {
	// Setup
	db := newMockDB()
	exchangeClient := &mockExchangeClient{}
	logger := &mockLogger{}
	config := newMockConfig()

	les := NewLayeredExecutionSystem(db, exchangeClient, logger, config)

	ctx := context.Background()

	// Create test execution with mixed layer statuses
	execution := &LayeredExecution{
		ID:         "test_execution_1",
		StrategyID: "test_strategy_1",
		Symbol:     "BTCUSDT",
		Status:     "EXECUTING",
		StartedAt:  time.Now().Add(-time.Hour), // Started 1 hour ago
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
					CreatedAt:  time.Now().Add(-time.Hour),
				},
				{
					ID:         "layer_2",
					Level:      2,
					Size:       2.0,
					EntryPrice: 90.0,
					StopLoss:   85.5,
					TakeProfit: 99.0,
					Status:     "CLOSED",
					CreatedAt:  time.Now().Add(-time.Hour),
				},
				{
					ID:         "layer_3",
					Level:      3,
					Size:       1.5,
					EntryPrice: 85.0,
					StopLoss:   80.75,
					TakeProfit: 93.5,
					Status:     "PARTIALLY_CLOSED",
					CreatedAt:  time.Now().Add(-time.Hour),
				},
			},
		},
		TotalExecuted: 5.5,
		AveragePrice:  90.0,
		TotalCost:     55.0,
		LayerResults: []LayerExecutionResult{
			{
				LayerID:         "layer_1",
				Success:         true,
				ExecutedSize:    2.0,
				ExecutedPrice:   95.0,
				ExecutionTime:   time.Second * 30,
				TransactionCost: 0.19,
				SlippageImpact:  0.001,
				ExecutedAt:      time.Now().Add(-time.Minute * 50),
			},
			{
				LayerID:         "layer_2",
				Success:         true,
				ExecutedSize:    2.0,
				ExecutedPrice:   90.0,
				ExecutionTime:   time.Second * 25,
				TransactionCost: 0.18,
				SlippageImpact:  0.0005,
				ExecutedAt:      time.Now().Add(-time.Minute * 40),
			},
		},
	}

	// Register execution
	les.mu.Lock()
	les.activeExecutions[execution.ID] = execution
	les.mu.Unlock()

	// Test performance tracking
	metrics, err := les.TrackLayeredPositionPerformance(ctx, execution.ID)

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if metrics == nil {
		t.Fatal("Expected performance metrics, got nil")
	}

	if metrics.StrategyID != execution.StrategyID {
		t.Errorf("Expected strategy ID %s, got %s", execution.StrategyID, metrics.StrategyID)
	}

	if metrics.TotalLayers != len(execution.Config.Layers) {
		t.Errorf("Expected total layers %d, got %d", len(execution.Config.Layers), metrics.TotalLayers)
	}

	expectedActiveLayers := 1 // layer_1 is ACTIVE
	if metrics.ActiveLayers != expectedActiveLayers {
		t.Errorf("Expected active layers %d, got %d", expectedActiveLayers, metrics.ActiveLayers)
	}

	expectedCompletedLayers := 1 // layer_2 is CLOSED
	if metrics.CompletedLayers != expectedCompletedLayers {
		t.Errorf("Expected completed layers %d, got %d", expectedCompletedLayers, metrics.CompletedLayers)
	}

	if metrics.ExecutionEfficiency < 0 || metrics.ExecutionEfficiency > 1 {
		t.Errorf("Expected execution efficiency between 0 and 1, got %f", metrics.ExecutionEfficiency)
	}

	if metrics.CalculatedAt.IsZero() {
		t.Error("Expected calculation time to be set")
	}

	if metrics.Metadata == nil {
		t.Error("Expected metadata to be set")
	}

	// Test metadata content
	if executionDuration, ok := metrics.Metadata["execution_duration"]; ok {
		if duration, ok := executionDuration.(time.Duration); ok {
			if duration <= 0 {
				t.Error("Expected positive execution duration in metadata")
			}
		} else {
			t.Error("Expected execution duration to be time.Duration")
		}
	} else {
		t.Error("Expected execution_duration in metadata")
	}

	if totalExecuted, ok := metrics.Metadata["total_executed"]; ok {
		if executed, ok := totalExecuted.(float64); ok {
			if executed != execution.TotalExecuted {
				t.Errorf("Expected total executed %f in metadata, got %f", execution.TotalExecuted, executed)
			}
		} else {
			t.Error("Expected total_executed to be float64")
		}
	} else {
		t.Error("Expected total_executed in metadata")
	}
}

// Benchmark tests
func BenchmarkLayeredPositionManager_AnalyzeMarketVolatility(b *testing.B) {
	// Setup
	db := newMockDB()
	exchangeClient := &mockExchangeClient{}
	logger := &mockLogger{}
	config := newMockConfig()

	lpm := NewLayeredPositionManager(db, exchangeClient, logger, config)
	ctx := context.Background()
	symbol := "BTCUSDT"

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := lpm.AnalyzeMarketVolatility(ctx, symbol)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}

func BenchmarkLayeredPositionManager_CalculateLayerConfiguration(b *testing.B) {
	// Setup
	db := newMockDB()
	exchangeClient := &mockExchangeClient{}
	logger := &mockLogger{}
	config := newMockConfig()

	lpm := NewLayeredPositionManager(db, exchangeClient, logger, config)
	ctx := context.Background()

	strategy := &LayeredStrategy{
		ID:              "benchmark_strategy",
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

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := lpm.CalculateLayerConfiguration(ctx, strategy)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}
