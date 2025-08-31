package factors

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSMAIndicator tests Simple Moving Average indicator
func TestSMAIndicator(t *testing.T) {
	sma := &SMAIndicator{}

	// Test data
	data := []MarketData{
		{Close: 10.0, Timestamp: time.Now()},
		{Close: 12.0, Timestamp: time.Now()},
		{Close: 14.0, Timestamp: time.Now()},
		{Close: 16.0, Timestamp: time.Now()},
		{Close: 18.0, Timestamp: time.Now()},
		{Close: 20.0, Timestamp: time.Now()},
	}

	t.Run("valid calculation", func(t *testing.T) {
		params := map[string]float64{"period": 3}
		result, err := sma.Calculate(data, params)
		require.NoError(t, err)
		require.Len(t, result, len(data))

		// First 2 values should be NaN
		assert.True(t, math.IsNaN(result[0]))
		assert.True(t, math.IsNaN(result[1]))

		// Third value: (10+12+14)/3 = 12
		assert.InDelta(t, 12.0, result[2], 0.001)

		// Fourth value: (12+14+16)/3 = 14
		assert.InDelta(t, 14.0, result[3], 0.001)

		// Fifth value: (14+16+18)/3 = 16
		assert.InDelta(t, 16.0, result[4], 0.001)

		// Sixth value: (16+18+20)/3 = 18
		assert.InDelta(t, 18.0, result[5], 0.001)
	})

	t.Run("insufficient data", func(t *testing.T) {
		shortData := data[:2]
		params := map[string]float64{"period": 3}
		_, err := sma.Calculate(shortData, params)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient data")
	})

	t.Run("default period", func(t *testing.T) {
		// Create data with 25 points for default period of 20
		longData := make([]MarketData, 25)
		for i := range longData {
			longData[i] = MarketData{Close: float64(i + 1), Timestamp: time.Now()}
		}

		params := map[string]float64{} // No period specified
		result, err := sma.Calculate(longData, params)
		require.NoError(t, err)

		// Should use default period of 20
		for i := 0; i < 19; i++ {
			assert.True(t, math.IsNaN(result[i]))
		}
		assert.False(t, math.IsNaN(result[19]))
	})

	t.Run("metadata", func(t *testing.T) {
		assert.Equal(t, "SMA", sma.GetName())
		assert.Equal(t, []string{"period"}, sma.GetRequiredParams())
	})
}

// TestEMAIndicator tests Exponential Moving Average indicator
func TestEMAIndicator(t *testing.T) {
	ema := &EMAIndicator{}

	data := []MarketData{
		{Close: 10.0, Timestamp: time.Now()},
		{Close: 12.0, Timestamp: time.Now()},
		{Close: 14.0, Timestamp: time.Now()},
		{Close: 16.0, Timestamp: time.Now()},
		{Close: 18.0, Timestamp: time.Now()},
	}

	t.Run("valid calculation", func(t *testing.T) {
		params := map[string]float64{"period": 3}
		result, err := ema.Calculate(data, params)
		require.NoError(t, err)
		require.Len(t, result, len(data))

		// First value should be the first close price
		assert.InDelta(t, 10.0, result[0], 0.001)

		// Subsequent values should be calculated using EMA formula
		// EMA = (Close * multiplier) + (previous EMA * (1 - multiplier))
		// multiplier = 2 / (period + 1) = 2 / 4 = 0.5
		expectedEMA1 := (12.0 * 0.5) + (10.0 * 0.5) // = 11.0
		assert.InDelta(t, expectedEMA1, result[1], 0.001)
	})

	t.Run("metadata", func(t *testing.T) {
		assert.Equal(t, "EMA", ema.GetName())
		assert.Equal(t, []string{"period"}, ema.GetRequiredParams())
	})
}

// TestRSIIndicator tests Relative Strength Index indicator
func TestRSIIndicator(t *testing.T) {
	rsi := &RSIIndicator{}

	// Create test data with known price movements
	data := []MarketData{
		{Close: 100.0, Timestamp: time.Now()},
		{Close: 102.0, Timestamp: time.Now()}, // +2
		{Close: 101.0, Timestamp: time.Now()}, // -1
		{Close: 103.0, Timestamp: time.Now()}, // +2
		{Close: 105.0, Timestamp: time.Now()}, // +2
		{Close: 104.0, Timestamp: time.Now()}, // -1
		{Close: 106.0, Timestamp: time.Now()}, // +2
		{Close: 108.0, Timestamp: time.Now()}, // +2
		{Close: 107.0, Timestamp: time.Now()}, // -1
		{Close: 109.0, Timestamp: time.Now()}, // +2
		{Close: 111.0, Timestamp: time.Now()}, // +2
		{Close: 110.0, Timestamp: time.Now()}, // -1
		{Close: 112.0, Timestamp: time.Now()}, // +2
		{Close: 114.0, Timestamp: time.Now()}, // +2
		{Close: 113.0, Timestamp: time.Now()}, // -1
	}

	t.Run("valid calculation", func(t *testing.T) {
		params := map[string]float64{"period": 14}
		result, err := rsi.Calculate(data, params)
		require.NoError(t, err)
		require.Len(t, result, len(data))

		// First value should be NaN
		assert.True(t, math.IsNaN(result[0]))

		// RSI should be between 0 and 100
		for i := 14; i < len(result); i++ {
			if !math.IsNaN(result[i]) {
				assert.GreaterOrEqual(t, result[i], 0.0)
				assert.LessOrEqual(t, result[i], 100.0)
			}
		}
	})

	t.Run("metadata", func(t *testing.T) {
		assert.Equal(t, "RSI", rsi.GetName())
		assert.Equal(t, []string{"period"}, rsi.GetRequiredParams())
	})
}

// TestMACDIndicator tests MACD indicator
func TestMACDIndicator(t *testing.T) {
	macd := &MACDIndicator{}

	// Create sufficient test data
	data := make([]MarketData, 50)
	for i := range data {
		data[i] = MarketData{
			Close:     100.0 + float64(i)*0.5 + math.Sin(float64(i)*0.1)*2,
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
		}
	}

	t.Run("valid calculation", func(t *testing.T) {
		params := map[string]float64{
			"fast_period":   12,
			"slow_period":   26,
			"signal_period": 9,
		}
		result, err := macd.Calculate(data, params)
		require.NoError(t, err)
		require.Len(t, result, len(data))

		// MACD values should be calculated after sufficient data
		// Check that we get non-NaN values after the slow period
		foundValidValue := false
		for i := 26; i < len(result); i++ {
			if !math.IsNaN(result[i]) {
				foundValidValue = true
				break
			}
		}
		assert.True(t, foundValidValue, "Should have valid MACD values after slow period")
	})

	t.Run("metadata", func(t *testing.T) {
		assert.Equal(t, "MACD", macd.GetName())
		expectedParams := []string{"fast_period", "slow_period", "signal_period"}
		assert.Equal(t, expectedParams, macd.GetRequiredParams())
	})
}

// TestBollingerBandsIndicator tests Bollinger Bands indicator
func TestBollingerBandsIndicator(t *testing.T) {
	bb := &BollingerBandsIndicator{}

	data := make([]MarketData, 30)
	for i := range data {
		data[i] = MarketData{
			Close:     100.0 + float64(i)*0.1,
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
		}
	}

	t.Run("valid calculation", func(t *testing.T) {
		params := map[string]float64{
			"period":     20,
			"multiplier": 2.0,
		}
		result, err := bb.Calculate(data, params)
		require.NoError(t, err)
		require.Len(t, result, len(data))

		// Should have valid values after the period
		for i := 20; i < len(result); i++ {
			assert.False(t, math.IsNaN(result[i]), "Should have valid BB values after period")
		}
	})

	t.Run("metadata", func(t *testing.T) {
		assert.Equal(t, "BollingerBands", bb.GetName())
		expectedParams := []string{"period", "multiplier"}
		assert.Equal(t, expectedParams, bb.GetRequiredParams())
	})
}

// TestStochasticIndicator tests Stochastic oscillator
func TestStochasticIndicator(t *testing.T) {
	stoch := &StochasticIndicator{}

	data := make([]MarketData, 20)
	for i := range data {
		base := 100.0 + float64(i)*0.5
		data[i] = MarketData{
			High:      base + 2.0,
			Low:       base - 2.0,
			Close:     base + math.Sin(float64(i)*0.3),
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
		}
	}

	t.Run("valid calculation", func(t *testing.T) {
		params := map[string]float64{
			"k_period": 14,
			"d_period": 3,
		}
		result, err := stoch.Calculate(data, params)
		require.NoError(t, err)
		require.Len(t, result, len(data))

		// Stochastic values should be between 0 and 100
		for i := 14; i < len(result); i++ {
			if !math.IsNaN(result[i]) {
				assert.GreaterOrEqual(t, result[i], 0.0)
				assert.LessOrEqual(t, result[i], 100.0)
			}
		}
	})

	t.Run("metadata", func(t *testing.T) {
		assert.Equal(t, "Stochastic", stoch.GetName())
		expectedParams := []string{"k_period", "d_period"}
		assert.Equal(t, expectedParams, stoch.GetRequiredParams())
	})
}

// TestATRIndicator tests Average True Range indicator
func TestATRIndicator(t *testing.T) {
	atr := &ATRIndicator{}

	data := make([]MarketData, 20)
	for i := range data {
		base := 100.0 + float64(i)*0.1
		data[i] = MarketData{
			High:      base + 1.0,
			Low:       base - 1.0,
			Close:     base,
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
		}
	}

	t.Run("valid calculation", func(t *testing.T) {
		params := map[string]float64{"period": 14}
		result, err := atr.Calculate(data, params)
		require.NoError(t, err)
		require.Len(t, result, len(data))

		// ATR values should be positive
		for i := 14; i < len(result); i++ {
			if !math.IsNaN(result[i]) {
				assert.GreaterOrEqual(t, result[i], 0.0)
			}
		}
	})

	t.Run("metadata", func(t *testing.T) {
		assert.Equal(t, "ATR", atr.GetName())
		assert.Equal(t, []string{"period"}, atr.GetRequiredParams())
	})
}
