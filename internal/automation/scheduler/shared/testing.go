package shared

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestFramework provides utilities for testing automation schedulers
type TestFramework struct {
	t              *testing.T
	mocks          map[string]interface{}
	testData       map[string]interface{}
	cleanupFuncs   []func()
	mu             sync.RWMutex
}

// NewTestFramework creates a new test framework instance
func NewTestFramework(t *testing.T) *TestFramework {
	return &TestFramework{
		t:            t,
		mocks:        make(map[string]interface{}),
		testData:     make(map[string]interface{}),
		cleanupFuncs: make([]func(), 0),
	}
}

// Cleanup runs all registered cleanup functions
func (tf *TestFramework) Cleanup() {
	tf.mu.Lock()
	defer tf.mu.Unlock()
	
	for _, cleanup := range tf.cleanupFuncs {
		cleanup()
	}
	tf.cleanupFuncs = nil
}

// RegisterCleanup registers a cleanup function
func (tf *TestFramework) RegisterCleanup(cleanup func()) {
	tf.mu.Lock()
	defer tf.mu.Unlock()
	
	tf.cleanupFuncs = append(tf.cleanupFuncs, cleanup)
}

// SetMock stores a mock object for later retrieval
func (tf *TestFramework) SetMock(name string, mockObj interface{}) {
	tf.mu.Lock()
	defer tf.mu.Unlock()
	
	tf.mocks[name] = mockObj
}

// GetMock retrieves a stored mock object
func (tf *TestFramework) GetMock(name string) interface{} {
	tf.mu.RLock()
	defer tf.mu.RUnlock()
	
	return tf.mocks[name]
}

// SetTestData stores test data for later retrieval
func (tf *TestFramework) SetTestData(key string, data interface{}) {
	tf.mu.Lock()
	defer tf.mu.Unlock()
	
	tf.testData[key] = data
}

// GetTestData retrieves stored test data
func (tf *TestFramework) GetTestData(key string) interface{} {
	tf.mu.RLock()
	defer tf.mu.RUnlock()
	
	return tf.testData[key]
}

// MockDatabase provides a mock database for testing
type MockDatabase struct {
	mock.Mock
	queries map[string][]map[string]interface{}
	mu      sync.RWMutex
}

// NewMockDatabase creates a new mock database
func NewMockDatabase() *MockDatabase {
	return &MockDatabase{
		queries: make(map[string][]map[string]interface{}),
	}
}

// QueryContext mocks database query execution
func (mdb *MockDatabase) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	mdb.mu.RLock()
	defer mdb.mu.RUnlock()
	
	mockArgs := mdb.Called(ctx, query, args)
	return mockArgs.Get(0).(*sql.Rows), mockArgs.Error(1)
}

// QueryRowContext mocks database single row query
func (mdb *MockDatabase) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	mdb.mu.RLock()
	defer mdb.mu.RUnlock()
	
	mockArgs := mdb.Called(ctx, query, args)
	return mockArgs.Get(0).(*sql.Row)
}

// ExecContext mocks database execution
func (mdb *MockDatabase) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	mdb.mu.RLock()
	defer mdb.mu.RUnlock()
	
	mockArgs := mdb.Called(ctx, query, args)
	return mockArgs.Get(0).(sql.Result), mockArgs.Error(1)
}

// SetQueryResult sets the expected result for a query
func (mdb *MockDatabase) SetQueryResult(query string, results []map[string]interface{}) {
	mdb.mu.Lock()
	defer mdb.mu.Unlock()
	
	mdb.queries[query] = results
}

// MockExchangeAPI provides a mock exchange API for testing
type MockExchangeAPI struct {
	mock.Mock
	positions    []Position
	marketData   map[string]interface{}
	orderHistory []map[string]interface{}
	mu           sync.RWMutex
}

// NewMockExchangeAPI creates a new mock exchange API
func NewMockExchangeAPI() *MockExchangeAPI {
	return &MockExchangeAPI{
		positions:    make([]Position, 0),
		marketData:   make(map[string]interface{}),
		orderHistory: make([]map[string]interface{}, 0),
	}
}

// GetPositions mocks getting positions from exchange
func (mea *MockExchangeAPI) GetPositions(ctx context.Context) ([]Position, error) {
	mea.mu.RLock()
	defer mea.mu.RUnlock()
	
	mockArgs := mea.Called(ctx)
	return mea.positions, mockArgs.Error(0)
}

// GetMarketData mocks getting market data from exchange
func (mea *MockExchangeAPI) GetMarketData(ctx context.Context, symbol string) (map[string]interface{}, error) {
	mea.mu.RLock()
	defer mea.mu.RUnlock()
	
	mockArgs := mea.Called(ctx, symbol)
	if data, exists := mea.marketData[symbol]; exists {
		return data.(map[string]interface{}), mockArgs.Error(0)
	}
	return nil, mockArgs.Error(0)
}

// PlaceOrder mocks placing an order
func (mea *MockExchangeAPI) PlaceOrder(ctx context.Context, order map[string]interface{}) (string, error) {
	mea.mu.Lock()
	defer mea.mu.Unlock()
	
	mockArgs := mea.Called(ctx, order)
	orderID := fmt.Sprintf("order_%d", time.Now().UnixNano())
	
	// Add to order history
	orderWithID := make(map[string]interface{})
	for k, v := range order {
		orderWithID[k] = v
	}
	orderWithID["id"] = orderID
	orderWithID["timestamp"] = time.Now()
	
	mea.orderHistory = append(mea.orderHistory, orderWithID)
	
	return orderID, mockArgs.Error(0)
}

// SetPositions sets mock positions
func (mea *MockExchangeAPI) SetPositions(positions []Position) {
	mea.mu.Lock()
	defer mea.mu.Unlock()
	
	mea.positions = positions
}

// SetMarketData sets mock market data
func (mea *MockExchangeAPI) SetMarketData(symbol string, data map[string]interface{}) {
	mea.mu.Lock()
	defer mea.mu.Unlock()
	
	mea.marketData[symbol] = data
}

// GetOrderHistory returns the order history
func (mea *MockExchangeAPI) GetOrderHistory() []map[string]interface{} {
	mea.mu.RLock()
	defer mea.mu.RUnlock()
	
	history := make([]map[string]interface{}, len(mea.orderHistory))
	copy(history, mea.orderHistory)
	return history
}

// MockConfigProvider provides a mock configuration provider for testing
type MockConfigProvider struct {
	mock.Mock
	config map[string]interface{}
	mu     sync.RWMutex
}

// NewMockConfigProvider creates a new mock config provider
func NewMockConfigProvider() *MockConfigProvider {
	return &MockConfigProvider{
		config: make(map[string]interface{}),
	}
}

// Get mocks getting a configuration value
func (mcp *MockConfigProvider) Get(key string) interface{} {
	mcp.mu.RLock()
	defer mcp.mu.RUnlock()
	
	mockArgs := mcp.Called(key)
	if value, exists := mcp.config[key]; exists {
		return value
	}
	return mockArgs.Get(0)
}

// GetString mocks getting a string configuration value
func (mcp *MockConfigProvider) GetString(key string) string {
	value := mcp.Get(key)
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}

// GetInt mocks getting an integer configuration value
func (mcp *MockConfigProvider) GetInt(key string) int {
	value := mcp.Get(key)
	if i, ok := value.(int); ok {
		return i
	}
	return 0
}

// GetFloat64 mocks getting a float64 configuration value
func (mcp *MockConfigProvider) GetFloat64(key string) float64 {
	value := mcp.Get(key)
	if f, ok := value.(float64); ok {
		return f
	}
	return 0.0
}

// GetBool mocks getting a boolean configuration value
func (mcp *MockConfigProvider) GetBool(key string) bool {
	value := mcp.Get(key)
	if b, ok := value.(bool); ok {
		return b
	}
	return false
}

// GetDuration mocks getting a duration configuration value
func (mcp *MockConfigProvider) GetDuration(key string) time.Duration {
	value := mcp.Get(key)
	if d, ok := value.(time.Duration); ok {
		return d
	}
	return 0
}

// Set mocks setting a configuration value
func (mcp *MockConfigProvider) Set(key string, value interface{}) error {
	mcp.mu.Lock()
	defer mcp.mu.Unlock()
	
	mockArgs := mcp.Called(key, value)
	mcp.config[key] = value
	return mockArgs.Error(0)
}

// Reload mocks reloading configuration
func (mcp *MockConfigProvider) Reload() error {
	mockArgs := mcp.Called()
	return mockArgs.Error(0)
}

// SetConfig sets a configuration value directly
func (mcp *MockConfigProvider) SetConfig(key string, value interface{}) {
	mcp.mu.Lock()
	defer mcp.mu.Unlock()
	
	mcp.config[key] = value
}

// MockMetricsCollector provides a mock metrics collector for testing
type MockMetricsCollector struct {
	mock.Mock
	counters   map[string]int64
	gauges     map[string]float64
	histograms map[string][]float64
	timers     map[string][]time.Duration
	mu         sync.RWMutex
}

// NewMockMetricsCollector creates a new mock metrics collector
func NewMockMetricsCollector() *MockMetricsCollector {
	return &MockMetricsCollector{
		counters:   make(map[string]int64),
		gauges:     make(map[string]float64),
		histograms: make(map[string][]float64),
		timers:     make(map[string][]time.Duration),
	}
}

// Counter mocks incrementing a counter metric
func (mmc *MockMetricsCollector) Counter(name string, tags map[string]string) error {
	mmc.mu.Lock()
	defer mmc.mu.Unlock()
	
	mockArgs := mmc.Called(name, tags)
	mmc.counters[name]++
	return mockArgs.Error(0)
}

// Gauge mocks setting a gauge metric
func (mmc *MockMetricsCollector) Gauge(name string, value float64, tags map[string]string) error {
	mmc.mu.Lock()
	defer mmc.mu.Unlock()
	
	mockArgs := mmc.Called(name, value, tags)
	mmc.gauges[name] = value
	return mockArgs.Error(0)
}

// Histogram mocks recording a histogram metric
func (mmc *MockMetricsCollector) Histogram(name string, value float64, tags map[string]string) error {
	mmc.mu.Lock()
	defer mmc.mu.Unlock()
	
	mockArgs := mmc.Called(name, value, tags)
	mmc.histograms[name] = append(mmc.histograms[name], value)
	return mockArgs.Error(0)
}

// Timer mocks recording a timer metric
func (mmc *MockMetricsCollector) Timer(name string, duration time.Duration, tags map[string]string) error {
	mmc.mu.Lock()
	defer mmc.mu.Unlock()
	
	mockArgs := mmc.Called(name, duration, tags)
	mmc.timers[name] = append(mmc.timers[name], duration)
	return mockArgs.Error(0)
}

// GetCounterValue returns the current counter value
func (mmc *MockMetricsCollector) GetCounterValue(name string) int64 {
	mmc.mu.RLock()
	defer mmc.mu.RUnlock()
	
	return mmc.counters[name]
}

// GetGaugeValue returns the current gauge value
func (mmc *MockMetricsCollector) GetGaugeValue(name string) float64 {
	mmc.mu.RLock()
	defer mmc.mu.RUnlock()
	
	return mmc.gauges[name]
}

// GetHistogramValues returns all histogram values
func (mmc *MockMetricsCollector) GetHistogramValues(name string) []float64 {
	mmc.mu.RLock()
	defer mmc.mu.RUnlock()
	
	values := make([]float64, len(mmc.histograms[name]))
	copy(values, mmc.histograms[name])
	return values
}

// GetTimerValues returns all timer values
func (mmc *MockMetricsCollector) GetTimerValues(name string) []time.Duration {
	mmc.mu.RLock()
	defer mmc.mu.RUnlock()
	
	values := make([]time.Duration, len(mmc.timers[name]))
	copy(values, mmc.timers[name])
	return values
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct {
	seed int64
}

// NewTestDataGenerator creates a new test data generator
func NewTestDataGenerator(seed int64) *TestDataGenerator {
	return &TestDataGenerator{seed: seed}
}

// GeneratePositions generates test positions
func (tdg *TestDataGenerator) GeneratePositions(count int) []Position {
	positions := make([]Position, count)
	
	symbols := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT"}
	sides := []string{"LONG", "SHORT"}
	
	for i := 0; i < count; i++ {
		positions[i] = Position{
			ID:           fmt.Sprintf("pos_%d", i+1),
			Symbol:       symbols[i%len(symbols)],
			Side:         sides[i%len(sides)],
			Size:         float64(1000 + i*100),
			EntryPrice:   float64(50000 + i*1000),
			CurrentPrice: float64(50000 + i*1000 + (i%2)*500 - (i%3)*200),
			Leverage:     float64(2 + i%8),
			MarginUsed:   float64(500 + i*50),
			Timestamp:    time.Now().Add(-time.Duration(i) * time.Hour),
		}
		
		// Calculate PnL
		if positions[i].Side == "LONG" {
			positions[i].UnrealizedPnL = (positions[i].CurrentPrice - positions[i].EntryPrice) * positions[i].Size / positions[i].EntryPrice
		} else {
			positions[i].UnrealizedPnL = (positions[i].EntryPrice - positions[i].CurrentPrice) * positions[i].Size / positions[i].EntryPrice
		}
	}
	
	return positions
}

// GenerateMarketData generates test market data
func (tdg *TestDataGenerator) GenerateMarketData(symbol string) map[string]interface{} {
	basePrice := 50000.0
	if symbol == "ETHUSDT" {
		basePrice = 3000.0
	} else if symbol == "BNBUSDT" {
		basePrice = 300.0
	}
	
	return map[string]interface{}{
		"symbol":           symbol,
		"price":            basePrice * (1 + (float64(tdg.seed%100)-50)/1000),
		"volume_24h":       1000000.0 + float64(tdg.seed%1000000),
		"price_change_24h": float64(tdg.seed%20-10) / 100.0,
		"volatility":       float64(tdg.seed%50) / 1000.0,
		"funding_rate":     float64(tdg.seed%200-100) / 100000.0,
		"timestamp":        time.Now(),
	}
}

// GenerateAnomalies generates test anomalies
func (tdg *TestDataGenerator) GenerateAnomalies(count int) []Anomaly {
	anomalies := make([]Anomaly, count)
	
	types := []AnomalyType{
		AnomalyTypeVolatilitySpike,
		AnomalyTypeLiquidityDrop,
		AnomalyTypeCorrelationBreakdown,
		AnomalyTypePriceSpike,
		AnomalyTypeVolumeAnomaly,
	}
	
	severities := []Severity{SeverityInfo, SeverityWarning, SeverityError, SeverityCritical}
	
	for i := 0; i < count; i++ {
		anomalies[i] = Anomaly{
			ID:          fmt.Sprintf("anomaly_%d", i+1),
			Type:        types[i%len(types)],
			Severity:    severities[i%len(severities)],
			Field:       "price",
			Value:       float64(50000 + i*1000),
			ExpectedMin: float64(45000),
			ExpectedMax: float64(55000),
			Confidence:  0.8 + float64(i%20)/100.0,
			Metadata: map[string]interface{}{
				"detector": "statistical",
				"method":   "z_score",
			},
			Timestamp: time.Now().Add(-time.Duration(i) * time.Minute),
		}
	}
	
	return anomalies
}

// AssertionHelpers provides common assertion helpers for testing
type AssertionHelpers struct {
	t *testing.T
}

// NewAssertionHelpers creates new assertion helpers
func NewAssertionHelpers(t *testing.T) *AssertionHelpers {
	return &AssertionHelpers{t: t}
}

// AssertPositionValid asserts that a position is valid
func (ah *AssertionHelpers) AssertPositionValid(position Position) {
	assert.NotEmpty(ah.t, position.ID, "Position ID should not be empty")
	assert.NotEmpty(ah.t, position.Symbol, "Position symbol should not be empty")
	assert.Contains(ah.t, []string{"LONG", "SHORT"}, position.Side, "Position side should be LONG or SHORT")
	assert.Greater(ah.t, position.Size, 0.0, "Position size should be positive")
	assert.Greater(ah.t, position.EntryPrice, 0.0, "Entry price should be positive")
	assert.Greater(ah.t, position.CurrentPrice, 0.0, "Current price should be positive")
	assert.Greater(ah.t, position.Leverage, 0.0, "Leverage should be positive")
}

// AssertErrorHandled asserts that an error was properly handled
func (ah *AssertionHelpers) AssertErrorHandled(err error, expectedCode string) {
	require.Error(ah.t, err, "Expected an error")
	
	if ae, ok := err.(*AutomationError); ok {
		assert.Equal(ah.t, expectedCode, ae.Code, "Error code should match expected")
		assert.NotEmpty(ah.t, ae.Message, "Error message should not be empty")
		assert.NotZero(ah.t, ae.Timestamp, "Error timestamp should be set")
	} else {
		ah.t.Errorf("Expected AutomationError, got %T", err)
	}
}

// AssertMetricsRecorded asserts that metrics were recorded
func (ah *AssertionHelpers) AssertMetricsRecorded(collector *MockMetricsCollector, metricName string) {
	// Check if any metric was recorded
	counterValue := collector.GetCounterValue(metricName)
	gaugeValue := collector.GetGaugeValue(metricName)
	histogramValues := collector.GetHistogramValues(metricName)
	timerValues := collector.GetTimerValues(metricName)
	
	recorded := counterValue > 0 || gaugeValue != 0 || len(histogramValues) > 0 || len(timerValues) > 0
	assert.True(ah.t, recorded, "Expected metric %s to be recorded", metricName)
}

// PerformanceTestHelper provides utilities for performance testing
type PerformanceTestHelper struct {
	t *testing.T
}

// NewPerformanceTestHelper creates a new performance test helper
func NewPerformanceTestHelper(t *testing.T) *PerformanceTestHelper {
	return &PerformanceTestHelper{t: t}
}

// MeasureExecutionTime measures the execution time of a function
func (pth *PerformanceTestHelper) MeasureExecutionTime(fn func()) time.Duration {
	start := time.Now()
	fn()
	return time.Since(start)
}

// AssertExecutionTime asserts that execution time is within expected bounds
func (pth *PerformanceTestHelper) AssertExecutionTime(fn func(), maxDuration time.Duration) {
	duration := pth.MeasureExecutionTime(fn)
	assert.LessOrEqual(pth.t, duration, maxDuration, 
		"Execution time %v should be less than or equal to %v", duration, maxDuration)
}

// BenchmarkFunction runs a benchmark on a function
func (pth *PerformanceTestHelper) BenchmarkFunction(fn func(), iterations int) (time.Duration, time.Duration, time.Duration) {
	durations := make([]time.Duration, iterations)
	
	for i := 0; i < iterations; i++ {
		durations[i] = pth.MeasureExecutionTime(fn)
	}
	
	// Calculate min, max, and average
	min := durations[0]
	max := durations[0]
	total := time.Duration(0)
	
	for _, d := range durations {
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
		total += d
	}
	
	avg := total / time.Duration(iterations)
	
	log.Printf("Benchmark results - Min: %v, Max: %v, Avg: %v", min, max, avg)
	
	return min, max, avg
}

// ConcurrencyTestHelper provides utilities for concurrency testing
type ConcurrencyTestHelper struct {
	t *testing.T
}

// NewConcurrencyTestHelper creates a new concurrency test helper
func NewConcurrencyTestHelper(t *testing.T) *ConcurrencyTestHelper {
	return &ConcurrencyTestHelper{t: t}
}

// TestConcurrentExecution tests concurrent execution of a function
func (cth *ConcurrencyTestHelper) TestConcurrentExecution(fn func(), goroutines int, iterations int) {
	var wg sync.WaitGroup
	errors := make(chan error, goroutines*iterations)
	
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				func() {
					defer func() {
						if r := recover(); r != nil {
							errors <- fmt.Errorf("panic: %v", r)
						}
					}()
					fn()
				}()
			}
		}()
	}
	
	wg.Wait()
	close(errors)
	
	// Check for errors
	errorCount := 0
	for err := range errors {
		errorCount++
		cth.t.Errorf("Concurrent execution error: %v", err)
	}
	
	assert.Equal(cth.t, 0, errorCount, "No errors should occur during concurrent execution")
}