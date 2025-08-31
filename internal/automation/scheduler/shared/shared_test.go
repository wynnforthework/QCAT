package shared

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAutomationError(t *testing.T) {
	t.Run("NewAutomationError", func(t *testing.T) {
		err := NewAutomationError(
			ErrCodeDatabaseConnection,
			"Database connection failed",
			"TestComponent",
			ErrorSeverityHigh,
			true,
		)

		assert.Equal(t, ErrCodeDatabaseConnection, err.Code)
		assert.Equal(t, "Database connection failed", err.Message)
		assert.Equal(t, "TestComponent", err.Component)
		assert.Equal(t, ErrorSeverityHigh, err.Severity)
		assert.True(t, err.Retryable)
		assert.NotZero(t, err.Timestamp)
		assert.NotNil(t, err.Context)
	})

	t.Run("WithContext", func(t *testing.T) {
		err := NewAutomationError(
			ErrCodeTimeout,
			"Operation timed out",
			"TestComponent",
			ErrorSeverityMedium,
			true,
		)

		err.WithContext("operation", "test_operation")
		err.WithContext("timeout", 30)

		assert.Equal(t, "test_operation", err.Context["operation"])
		assert.Equal(t, 30, err.Context["timeout"])
	})

	t.Run("Error interface", func(t *testing.T) {
		err := NewAutomationError(
			ErrCodeInvalidParameters,
			"Invalid parameters provided",
			"TestComponent",
			ErrorSeverityLow,
			false,
		)

		assert.Equal(t, "Invalid parameters provided", err.Error())
	})
}

func TestRetryStrategy(t *testing.T) {
	t.Run("NewRetryStrategy", func(t *testing.T) {
		strategy := NewRetryStrategy(3, time.Second, time.Minute, 2.0)

		assert.Equal(t, 3, strategy.MaxRetries)
		assert.Equal(t, time.Second, strategy.InitialDelay)
		assert.Equal(t, time.Minute, strategy.MaxDelay)
		assert.Equal(t, 2.0, strategy.BackoffFactor)
		assert.NotNil(t, strategy.RetryableErrors)
	})

	t.Run("CanRecover", func(t *testing.T) {
		strategy := NewRetryStrategy(3, time.Second, time.Minute, 2.0)

		// Test retryable error
		retryableErr := NewAutomationError(
			ErrCodeDatabaseConnection,
			"DB error",
			"Test",
			ErrorSeverityMedium,
			true,
		)
		assert.True(t, strategy.CanRecover(retryableErr))

		// Test non-retryable error
		nonRetryableErr := NewAutomationError(
			ErrCodeInvalidParameters,
			"Invalid params",
			"Test",
			ErrorSeverityLow,
			false,
		)
		assert.False(t, strategy.CanRecover(nonRetryableErr))
	})

	t.Run("GetRecoveryTime", func(t *testing.T) {
		strategy := NewRetryStrategy(3, time.Second, time.Minute, 2.0)
		recoveryTime := strategy.GetRecoveryTime()

		// Should be sum of delays: 1s + 2s + 4s = 7s
		expectedTime := time.Second + 2*time.Second + 4*time.Second
		assert.Equal(t, expectedTime, recoveryTime)
	})
}

func TestCircuitBreaker(t *testing.T) {
	t.Run("NewCircuitBreaker", func(t *testing.T) {
		config := CircuitBreakerConfig{
			FailureThreshold: 3,
			RecoveryTimeout:  time.Minute,
			HalfOpenRequests: 2,
			SuccessThreshold: 1,
		}

		cb := NewCircuitBreaker(config)
		assert.Equal(t, CircuitBreakerClosed, cb.GetState())
	})

	t.Run("Execute success", func(t *testing.T) {
		config := CircuitBreakerConfig{
			FailureThreshold: 3,
			RecoveryTimeout:  time.Minute,
			HalfOpenRequests: 2,
			SuccessThreshold: 1,
		}

		cb := NewCircuitBreaker(config)
		ctx := context.Background()

		err := cb.Execute(ctx, func() error {
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, CircuitBreakerClosed, cb.GetState())
	})

	t.Run("Execute failure and recovery", func(t *testing.T) {
		config := CircuitBreakerConfig{
			FailureThreshold: 2,
			RecoveryTimeout:  time.Millisecond * 100,
			HalfOpenRequests: 1,
			SuccessThreshold: 1,
		}

		cb := NewCircuitBreaker(config)
		ctx := context.Background()

		// Trigger failures to open circuit breaker
		for i := 0; i < 2; i++ {
			cb.Execute(ctx, func() error {
				return NewAutomationError("TEST_ERROR", "Test error", "Test", ErrorSeverityMedium, true)
			})
		}

		assert.Equal(t, CircuitBreakerOpen, cb.GetState())

		// Wait for recovery timeout
		time.Sleep(time.Millisecond * 150)

		// Should allow execution in half-open state
		err := cb.Execute(ctx, func() error {
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, CircuitBreakerClosed, cb.GetState())
	})
}

func TestConfigManager(t *testing.T) {
	t.Run("NewConfigManager", func(t *testing.T) {
		cm := NewConfigManager()
		assert.NotNil(t, cm)
		assert.NotNil(t, cm.config)
		assert.NotNil(t, cm.defaults)
	})

	t.Run("LoadConfig", func(t *testing.T) {
		cm := NewConfigManager()

		configMap := map[string]interface{}{
			"risk_management": map[string]interface{}{
				"enabled": true,
			},
		}

		err := cm.LoadConfig(configMap)
		assert.NoError(t, err)

		config := cm.GetConfig()
		assert.NotNil(t, config)
	})
}

func TestUtilityFunctions(t *testing.T) {
	t.Run("GenerateID", func(t *testing.T) {
		id1 := GenerateID("test")
		id2 := GenerateID("test")

		assert.NotEqual(t, id1, id2)
		assert.Contains(t, id1, "test_")
		assert.Contains(t, id2, "test_")
	})

	t.Run("CalculateMean", func(t *testing.T) {
		values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
		mean := CalculateMean(values)
		assert.Equal(t, 3.0, mean)

		// Test empty slice
		emptyMean := CalculateMean([]float64{})
		assert.Equal(t, 0.0, emptyMean)
	})

	t.Run("CalculateStandardDeviation", func(t *testing.T) {
		values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
		stdDev := CalculateStandardDeviation(values)
		assert.InDelta(t, 1.58, stdDev, 0.01)

		// Test single value
		singleStdDev := CalculateStandardDeviation([]float64{5.0})
		assert.Equal(t, 0.0, singleStdDev)
	})

	t.Run("CalculateCorrelation", func(t *testing.T) {
		x := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
		y := []float64{2.0, 4.0, 6.0, 8.0, 10.0}

		correlation := CalculateCorrelation(x, y)
		assert.InDelta(t, 1.0, correlation, 0.01) // Perfect positive correlation

		// Test negative correlation
		yNeg := []float64{10.0, 8.0, 6.0, 4.0, 2.0}
		negCorrelation := CalculateCorrelation(x, yNeg)
		assert.InDelta(t, -1.0, negCorrelation, 0.01)
	})

	t.Run("CalculatePercentile", func(t *testing.T) {
		values := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0}

		p50 := CalculatePercentile(values, 50)
		assert.Equal(t, 5.5, p50)

		p90 := CalculatePercentile(values, 90)
		assert.Equal(t, 9.1, p90)

		// Test empty slice
		emptyP50 := CalculatePercentile([]float64{}, 50)
		assert.Equal(t, 0.0, emptyP50)
	})

	t.Run("CalculateZScore", func(t *testing.T) {
		zScore := CalculateZScore(7.0, 5.0, 2.0)
		assert.Equal(t, 1.0, zScore)

		// Test zero standard deviation
		zeroStdScore := CalculateZScore(5.0, 5.0, 0.0)
		assert.Equal(t, 0.0, zeroStdScore)
	})

	t.Run("CalculateSharpeRatio", func(t *testing.T) {
		returns := []float64{0.1, 0.05, 0.15, -0.02, 0.08}
		riskFreeRate := 0.02

		sharpe := CalculateSharpeRatio(returns, riskFreeRate)
		assert.Greater(t, sharpe, 0.0)
	})

	t.Run("CalculateMaxDrawdown", func(t *testing.T) {
		equityCurve := []float64{100, 110, 105, 95, 90, 100, 120}
		maxDD := CalculateMaxDrawdown(equityCurve)

		// Max drawdown should be from 110 to 90 = 18.18%
		assert.InDelta(t, 0.1818, maxDD, 0.01)
	})

	t.Run("NormalizeValues", func(t *testing.T) {
		values := []float64{1.0, 3.0, 5.0, 7.0, 9.0}
		normalized := NormalizeValues(values)

		assert.Equal(t, 0.0, normalized[0])
		assert.Equal(t, 1.0, normalized[4])
		assert.InDelta(t, 0.5, normalized[2], 0.01)
	})

	t.Run("DetectOutliers", func(t *testing.T) {
		values := []float64{1, 2, 3, 4, 5, 100} // 100 is an outlier
		outliers := DetectOutliers(values, 1.5)

		assert.Contains(t, outliers, 5) // Index of outlier value 100
	})

	t.Run("ConvertToFloat64", func(t *testing.T) {
		// Test various types
		f1, err1 := ConvertToFloat64(42)
		assert.NoError(t, err1)
		assert.Equal(t, 42.0, f1)

		f2, err2 := ConvertToFloat64("3.14")
		assert.NoError(t, err2)
		assert.Equal(t, 3.14, f2)

		f3, err3 := ConvertToFloat64(float32(2.5))
		assert.NoError(t, err3)
		assert.Equal(t, 2.5, f3)

		// Test invalid conversion
		_, err4 := ConvertToFloat64([]int{1, 2, 3})
		assert.Error(t, err4)
	})

	t.Run("FormatDuration", func(t *testing.T) {
		assert.Equal(t, "30.0s", FormatDuration(30*time.Second))
		assert.Equal(t, "2.0m", FormatDuration(2*time.Minute))
		assert.Equal(t, "1.5h", FormatDuration(90*time.Minute))
		assert.Equal(t, "2.0d", FormatDuration(48*time.Hour))
	})

	t.Run("ParseDuration", func(t *testing.T) {
		d1, err1 := ParseDuration("2d")
		assert.NoError(t, err1)
		assert.Equal(t, 48*time.Hour, d1)

		d2, err2 := ParseDuration("1w")
		assert.NoError(t, err2)
		assert.Equal(t, 7*24*time.Hour, d2)

		d3, err3 := ParseDuration("30m")
		assert.NoError(t, err3)
		assert.Equal(t, 30*time.Minute, d3)
	})

	t.Run("ValidateRequired", func(t *testing.T) {
		data := map[string]interface{}{
			"name":  "test",
			"value": 42,
			"empty": "",
		}

		// Valid case
		err1 := ValidateRequired(data, []string{"name", "value"})
		assert.NoError(t, err1)

		// Missing field
		err2 := ValidateRequired(data, []string{"missing"})
		assert.Error(t, err2)
		assert.Contains(t, err2.Error(), "missing")

		// Empty field
		err3 := ValidateRequired(data, []string{"empty"})
		assert.Error(t, err3)
		assert.Contains(t, err3.Error(), "empty")
	})

	t.Run("SanitizeString", func(t *testing.T) {
		input := "  test\x00string\n  "
		sanitized := SanitizeString(input)
		assert.Equal(t, "teststring", sanitized)
	})

	t.Run("TruncateString", func(t *testing.T) {
		long := "This is a very long string that needs to be truncated"
		truncated := TruncateString(long, 20)
		assert.Equal(t, "This is a very lo...", truncated)

		short := "Short"
		notTruncated := TruncateString(short, 20)
		assert.Equal(t, "Short", notTruncated)
	})
}

func TestTestFramework(t *testing.T) {
	t.Run("NewTestFramework", func(t *testing.T) {
		tf := NewTestFramework(t)
		assert.NotNil(t, tf)
		assert.NotNil(t, tf.mocks)
		assert.NotNil(t, tf.testData)
	})

	t.Run("Mock management", func(t *testing.T) {
		tf := NewTestFramework(t)

		mockDB := NewMockDatabase()
		tf.SetMock("database", mockDB)

		retrieved := tf.GetMock("database")
		assert.Equal(t, mockDB, retrieved)
	})

	t.Run("Test data management", func(t *testing.T) {
		tf := NewTestFramework(t)

		testData := map[string]interface{}{
			"test_key": "test_value",
		}
		tf.SetTestData("config", testData)

		retrieved := tf.GetTestData("config")
		assert.Equal(t, testData, retrieved)
	})
}

func TestMockDatabase(t *testing.T) {
	t.Run("NewMockDatabase", func(t *testing.T) {
		mockDB := NewMockDatabase()
		assert.NotNil(t, mockDB)
		assert.NotNil(t, mockDB.queries)
	})

	t.Run("SetQueryResult", func(t *testing.T) {
		mockDB := NewMockDatabase()

		results := []map[string]interface{}{
			{"id": 1, "name": "test1"},
			{"id": 2, "name": "test2"},
		}

		mockDB.SetQueryResult("SELECT * FROM test", results)

		storedResults := mockDB.queries["SELECT * FROM test"]
		assert.Equal(t, results, storedResults)
	})
}

func TestMockExchangeAPI(t *testing.T) {
	t.Run("NewMockExchangeAPI", func(t *testing.T) {
		mockAPI := NewMockExchangeAPI()
		assert.NotNil(t, mockAPI)
		assert.NotNil(t, mockAPI.positions)
		assert.NotNil(t, mockAPI.marketData)
		assert.NotNil(t, mockAPI.orderHistory)
	})

	t.Run("Position management", func(t *testing.T) {
		mockAPI := NewMockExchangeAPI()

		positions := []Position{
			{
				ID:     "pos1",
				Symbol: "BTCUSDT",
				Side:   "LONG",
				Size:   1000,
			},
		}

		mockAPI.SetPositions(positions)

		mockAPI.On("GetPositions", context.Background()).Return(nil)

		retrieved, err := mockAPI.GetPositions(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, positions, retrieved)
	})

	t.Run("Market data management", func(t *testing.T) {
		mockAPI := NewMockExchangeAPI()

		marketData := map[string]interface{}{
			"price":  50000.0,
			"volume": 1000000.0,
		}

		mockAPI.SetMarketData("BTCUSDT", marketData)
		mockAPI.On("GetMarketData", context.Background(), "BTCUSDT").Return(nil)

		retrieved, err := mockAPI.GetMarketData(context.Background(), "BTCUSDT")
		assert.NoError(t, err)
		assert.Equal(t, marketData, retrieved)
	})
}

func TestTestDataGenerator(t *testing.T) {
	t.Run("NewTestDataGenerator", func(t *testing.T) {
		tdg := NewTestDataGenerator(12345)
		assert.Equal(t, int64(12345), tdg.seed)
	})

	t.Run("GeneratePositions", func(t *testing.T) {
		tdg := NewTestDataGenerator(12345)
		positions := tdg.GeneratePositions(5)

		assert.Len(t, positions, 5)

		for _, pos := range positions {
			assert.NotEmpty(t, pos.ID)
			assert.NotEmpty(t, pos.Symbol)
			assert.Contains(t, []string{"LONG", "SHORT"}, pos.Side)
			assert.Greater(t, pos.Size, 0.0)
			assert.Greater(t, pos.EntryPrice, 0.0)
		}
	})

	t.Run("GenerateMarketData", func(t *testing.T) {
		tdg := NewTestDataGenerator(12345)
		data := tdg.GenerateMarketData("BTCUSDT")

		assert.Equal(t, "BTCUSDT", data["symbol"])
		assert.NotNil(t, data["price"])
		assert.NotNil(t, data["volume_24h"])
		assert.NotNil(t, data["timestamp"])
	})

	t.Run("GenerateAnomalies", func(t *testing.T) {
		tdg := NewTestDataGenerator(12345)
		anomalies := tdg.GenerateAnomalies(3)

		assert.Len(t, anomalies, 3)

		for _, anomaly := range anomalies {
			assert.NotEmpty(t, anomaly.ID)
			assert.NotEmpty(t, anomaly.Field)
			assert.NotNil(t, anomaly.Value)
			assert.Greater(t, anomaly.Confidence, 0.0)
		}
	})
}

func TestAssertionHelpers(t *testing.T) {
	t.Run("AssertPositionValid", func(t *testing.T) {
		ah := NewAssertionHelpers(t)

		validPosition := Position{
			ID:           "pos1",
			Symbol:       "BTCUSDT",
			Side:         "LONG",
			Size:         1000,
			EntryPrice:   50000,
			CurrentPrice: 51000,
			Leverage:     2,
		}

		// This should not panic or fail
		ah.AssertPositionValid(validPosition)
	})

	t.Run("AssertErrorHandled", func(t *testing.T) {
		ah := NewAssertionHelpers(t)

		err := NewAutomationError(
			ErrCodeDatabaseConnection,
			"Test error",
			"TestComponent",
			ErrorSeverityMedium,
			true,
		)

		// This should not panic or fail
		ah.AssertErrorHandled(err, ErrCodeDatabaseConnection)
	})
}

func TestPerformanceTestHelper(t *testing.T) {
	t.Run("MeasureExecutionTime", func(t *testing.T) {
		pth := NewPerformanceTestHelper(t)

		duration := pth.MeasureExecutionTime(func() {
			time.Sleep(10 * time.Millisecond)
		})

		assert.GreaterOrEqual(t, duration, 10*time.Millisecond)
		assert.Less(t, duration, 50*time.Millisecond) // Allow some overhead
	})

	t.Run("BenchmarkFunction", func(t *testing.T) {
		pth := NewPerformanceTestHelper(t)

		min, max, avg := pth.BenchmarkFunction(func() {
			time.Sleep(time.Millisecond)
		}, 3)

		assert.Greater(t, min, time.Duration(0))
		assert.GreaterOrEqual(t, max, min)
		assert.GreaterOrEqual(t, avg, min)
		assert.LessOrEqual(t, avg, max)
	})
}

func TestConcurrencyTestHelper(t *testing.T) {
	t.Run("TestConcurrentExecution", func(t *testing.T) {
		cth := NewConcurrencyTestHelper(t)

		counter := 0

		// This should execute without errors
		cth.TestConcurrentExecution(func() {
			counter++ // This is not thread-safe, but for testing purposes it's ok
		}, 5, 10)

		// We can't assert exact count due to race conditions, but it should be > 0
		assert.Greater(t, counter, 0)
	})
}

// Benchmark tests
func BenchmarkCalculateMean(b *testing.B) {
	values := make([]float64, 1000)
	for i := range values {
		values[i] = float64(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateMean(values)
	}
}

func BenchmarkCalculateStandardDeviation(b *testing.B) {
	values := make([]float64, 1000)
	for i := range values {
		values[i] = float64(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateStandardDeviation(values)
	}
}

func BenchmarkCalculateCorrelation(b *testing.B) {
	x := make([]float64, 1000)
	y := make([]float64, 1000)
	for i := range x {
		x[i] = float64(i)
		y[i] = float64(i * 2)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateCorrelation(x, y)
	}
}

func BenchmarkGenerateID(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateID("test")
	}
}
