package shared

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

// GenerateID generates a unique identifier
func GenerateID(prefix string) string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("%s_%s_%d", prefix, hex.EncodeToString(bytes), time.Now().UnixNano())
}

// CalculatePercentile calculates the percentile of a value in a slice
func CalculatePercentile(values []float64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)
	
	index := percentile * float64(len(sorted)-1) / 100.0
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))
	
	if lower == upper {
		return sorted[lower]
	}
	
	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

// CalculateZScore calculates the z-score of a value
func CalculateZScore(value, mean, stdDev float64) float64 {
	if stdDev == 0 {
		return 0
	}
	return (value - mean) / stdDev
}

// CalculateMean calculates the mean of a slice of float64 values
func CalculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// CalculateStandardDeviation calculates the standard deviation of a slice of float64 values
func CalculateStandardDeviation(values []float64) float64 {
	if len(values) <= 1 {
		return 0
	}
	
	mean := CalculateMean(values)
	sumSquaredDiffs := 0.0
	
	for _, v := range values {
		diff := v - mean
		sumSquaredDiffs += diff * diff
	}
	
	variance := sumSquaredDiffs / float64(len(values)-1)
	return math.Sqrt(variance)
}

// CalculateCorrelation calculates the Pearson correlation coefficient between two slices
func CalculateCorrelation(x, y []float64) float64 {
	if len(x) != len(y) || len(x) == 0 {
		return 0
	}
	
	meanX := CalculateMean(x)
	meanY := CalculateMean(y)
	
	numerator := 0.0
	sumSquaredX := 0.0
	sumSquaredY := 0.0
	
	for i := 0; i < len(x); i++ {
		diffX := x[i] - meanX
		diffY := y[i] - meanY
		
		numerator += diffX * diffY
		sumSquaredX += diffX * diffX
		sumSquaredY += diffY * diffY
	}
	
	denominator := math.Sqrt(sumSquaredX * sumSquaredY)
	if denominator == 0 {
		return 0
	}
	
	return numerator / denominator
}

// CalculateATR calculates the Average True Range
func CalculateATR(highs, lows, closes []float64, period int) []float64 {
	if len(highs) != len(lows) || len(lows) != len(closes) || len(closes) < period+1 {
		return nil
	}
	
	trueRanges := make([]float64, len(closes)-1)
	
	// Calculate True Range for each period
	for i := 1; i < len(closes); i++ {
		tr1 := highs[i] - lows[i]
		tr2 := math.Abs(highs[i] - closes[i-1])
		tr3 := math.Abs(lows[i] - closes[i-1])
		
		trueRanges[i-1] = math.Max(tr1, math.Max(tr2, tr3))
	}
	
	// Calculate ATR using simple moving average
	atr := make([]float64, len(trueRanges)-period+1)
	for i := period - 1; i < len(trueRanges); i++ {
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += trueRanges[j]
		}
		atr[i-period+1] = sum / float64(period)
	}
	
	return atr
}

// CalculateRealizedVolatility calculates realized volatility from price returns
func CalculateRealizedVolatility(prices []float64, period int) []float64 {
	if len(prices) < period+1 {
		return nil
	}
	
	// Calculate returns
	returns := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		returns[i-1] = math.Log(prices[i] / prices[i-1])
	}
	
	// Calculate rolling volatility
	volatilities := make([]float64, len(returns)-period+1)
	for i := period - 1; i < len(returns); i++ {
		windowReturns := returns[i-period+1 : i+1]
		volatilities[i-period+1] = CalculateStandardDeviation(windowReturns) * math.Sqrt(252) // Annualized
	}
	
	return volatilities
}

// CalculateSharpeRatio calculates the Sharpe ratio
func CalculateSharpeRatio(returns []float64, riskFreeRate float64) float64 {
	if len(returns) == 0 {
		return 0
	}
	
	meanReturn := CalculateMean(returns)
	stdDev := CalculateStandardDeviation(returns)
	
	if stdDev == 0 {
		return 0
	}
	
	return (meanReturn - riskFreeRate) / stdDev
}

// CalculateMaxDrawdown calculates the maximum drawdown from equity curve
func CalculateMaxDrawdown(equityCurve []float64) float64 {
	if len(equityCurve) == 0 {
		return 0
	}
	
	maxDrawdown := 0.0
	peak := equityCurve[0]
	
	for _, equity := range equityCurve {
		if equity > peak {
			peak = equity
		}
		
		drawdown := (peak - equity) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}
	
	return maxDrawdown
}

// CalculateVaR calculates Value at Risk using historical simulation
func CalculateVaR(returns []float64, confidenceLevel float64) float64 {
	if len(returns) == 0 {
		return 0
	}
	
	sorted := make([]float64, len(returns))
	copy(sorted, returns)
	sort.Float64s(sorted)
	
	index := int(math.Ceil(float64(len(sorted)) * (1 - confidenceLevel)))
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	
	return -sorted[index] // VaR is typically expressed as a positive number
}

// CalculateExpectedShortfall calculates Expected Shortfall (Conditional VaR)
func CalculateExpectedShortfall(returns []float64, confidenceLevel float64) float64 {
	if len(returns) == 0 {
		return 0
	}
	
	varValue := CalculateVaR(returns, confidenceLevel)
	
	// Calculate average of returns worse than VaR
	sum := 0.0
	count := 0
	
	for _, ret := range returns {
		if -ret >= varValue {
			sum += ret
			count++
		}
	}
	
	if count == 0 {
		return varValue
	}
	
	return -sum / float64(count)
}

// NormalizeValues normalizes a slice of values to [0, 1] range
func NormalizeValues(values []float64) []float64 {
	if len(values) == 0 {
		return values
	}
	
	min := values[0]
	max := values[0]
	
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	
	if max == min {
		normalized := make([]float64, len(values))
		for i := range normalized {
			normalized[i] = 0.5 // If all values are the same, set to middle
		}
		return normalized
	}
	
	normalized := make([]float64, len(values))
	for i, v := range values {
		normalized[i] = (v - min) / (max - min)
	}
	
	return normalized
}

// StandardizeValues standardizes values to have mean 0 and std dev 1
func StandardizeValues(values []float64) []float64 {
	if len(values) == 0 {
		return values
	}
	
	mean := CalculateMean(values)
	stdDev := CalculateStandardDeviation(values)
	
	if stdDev == 0 {
		standardized := make([]float64, len(values))
		return standardized // All zeros if no variation
	}
	
	standardized := make([]float64, len(values))
	for i, v := range values {
		standardized[i] = (v - mean) / stdDev
	}
	
	return standardized
}

// DetectOutliers detects outliers using the IQR method
func DetectOutliers(values []float64, multiplier float64) []int {
	if len(values) < 4 {
		return nil
	}
	
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)
	
	q1 := CalculatePercentile(sorted, 25)
	q3 := CalculatePercentile(sorted, 75)
	iqr := q3 - q1
	
	lowerBound := q1 - multiplier*iqr
	upperBound := q3 + multiplier*iqr
	
	var outliers []int
	for i, v := range values {
		if v < lowerBound || v > upperBound {
			outliers = append(outliers, i)
		}
	}
	
	return outliers
}

// InterpolateLinear performs linear interpolation for missing values
func InterpolateLinear(values []float64, missingIndices []int) []float64 {
	if len(missingIndices) == 0 {
		return values
	}
	
	result := make([]float64, len(values))
	copy(result, values)
	
	for _, idx := range missingIndices {
		if idx <= 0 || idx >= len(values)-1 {
			continue // Can't interpolate at boundaries
		}
		
		// Find nearest non-missing values
		leftIdx := idx - 1
		rightIdx := idx + 1
		
		for leftIdx >= 0 && contains(missingIndices, leftIdx) {
			leftIdx--
		}
		for rightIdx < len(values) && contains(missingIndices, rightIdx) {
			rightIdx++
		}
		
		if leftIdx >= 0 && rightIdx < len(values) {
			// Linear interpolation
			leftVal := values[leftIdx]
			rightVal := values[rightIdx]
			weight := float64(idx-leftIdx) / float64(rightIdx-leftIdx)
			result[idx] = leftVal + weight*(rightVal-leftVal)
		}
	}
	
	return result
}

// contains checks if a slice contains a specific integer
func contains(slice []int, item int) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// RoundToDecimalPlaces rounds a float64 to specified decimal places
func RoundToDecimalPlaces(value float64, places int) float64 {
	multiplier := math.Pow(10, float64(places))
	return math.Round(value*multiplier) / multiplier
}

// FormatDuration formats a duration in a human-readable way
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	} else {
		return fmt.Sprintf("%.1fd", d.Hours()/24)
	}
}

// ParseDuration parses a duration string with support for various units
func ParseDuration(s string) (time.Duration, error) {
	// Handle common cases that time.ParseDuration doesn't support
	s = strings.ToLower(strings.TrimSpace(s))
	
	if strings.HasSuffix(s, "d") || strings.HasSuffix(s, "day") || strings.HasSuffix(s, "days") {
		numStr := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(s, "days"), "day"), "d")
		num, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return 0, err
		}
		return time.Duration(num * 24 * float64(time.Hour)), nil
	}
	
	if strings.HasSuffix(s, "w") || strings.HasSuffix(s, "week") || strings.HasSuffix(s, "weeks") {
		numStr := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(s, "weeks"), "week"), "w")
		num, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return 0, err
		}
		return time.Duration(num * 7 * 24 * float64(time.Hour)), nil
	}
	
	return time.ParseDuration(s)
}

// ConvertToFloat64 safely converts various types to float64
func ConvertToFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", value)
	}
}

// ConvertToInt converts various types to int
func ConvertToInt(value interface{}) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case float32:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("cannot convert %T to int", value)
	}
}

// ConvertToBool converts various types to bool
func ConvertToBool(value interface{}) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		return strconv.ParseBool(v)
	case int:
		return v != 0, nil
	case float64:
		return v != 0, nil
	default:
		return false, fmt.Errorf("cannot convert %T to bool", value)
	}
}

// DeepCopy creates a deep copy of a struct using reflection
func DeepCopy(src, dst interface{}) error {
	srcVal := reflect.ValueOf(src)
	dstVal := reflect.ValueOf(dst)
	
	if srcVal.Kind() == reflect.Ptr {
		srcVal = srcVal.Elem()
	}
	if dstVal.Kind() != reflect.Ptr {
		return fmt.Errorf("destination must be a pointer")
	}
	dstVal = dstVal.Elem()
	
	if srcVal.Type() != dstVal.Type() {
		return fmt.Errorf("source and destination types must match")
	}
	
	return deepCopyValue(srcVal, dstVal)
}

// deepCopyValue recursively copies values
func deepCopyValue(src, dst reflect.Value) error {
	switch src.Kind() {
	case reflect.Struct:
		for i := 0; i < src.NumField(); i++ {
			if err := deepCopyValue(src.Field(i), dst.Field(i)); err != nil {
				return err
			}
		}
	case reflect.Slice:
		if src.IsNil() {
			return nil
		}
		dst.Set(reflect.MakeSlice(src.Type(), src.Len(), src.Cap()))
		for i := 0; i < src.Len(); i++ {
			if err := deepCopyValue(src.Index(i), dst.Index(i)); err != nil {
				return err
			}
		}
	case reflect.Map:
		if src.IsNil() {
			return nil
		}
		dst.Set(reflect.MakeMap(src.Type()))
		for _, key := range src.MapKeys() {
			newKey := reflect.New(key.Type()).Elem()
			if err := deepCopyValue(key, newKey); err != nil {
				return err
			}
			newVal := reflect.New(src.MapIndex(key).Type()).Elem()
			if err := deepCopyValue(src.MapIndex(key), newVal); err != nil {
				return err
			}
			dst.SetMapIndex(newKey, newVal)
		}
	case reflect.Ptr:
		if src.IsNil() {
			return nil
		}
		dst.Set(reflect.New(src.Type().Elem()))
		return deepCopyValue(src.Elem(), dst.Elem())
	default:
		dst.Set(src)
	}
	return nil
}

// RetryWithBackoff executes a function with exponential backoff retry
func RetryWithBackoff(ctx context.Context, fn func() error, maxRetries int, initialDelay time.Duration, maxDelay time.Duration, backoffFactor float64) error {
	var lastErr error
	delay := initialDelay
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// Calculate next delay
				delay = time.Duration(float64(delay) * backoffFactor)
				if delay > maxDelay {
					delay = maxDelay
				}
			}
		}
		
		if err := fn(); err != nil {
			lastErr = err
			continue
		}
		
		return nil // Success
	}
	
	return fmt.Errorf("operation failed after %d attempts: %w", maxRetries+1, lastErr)
}

// TimeoutContext creates a context with timeout
func TimeoutContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, timeout)
}

// MergeStringMaps merges multiple string maps, with later maps overriding earlier ones
func MergeStringMaps(maps ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// MergeInterfaceMaps merges multiple interface{} maps
func MergeInterfaceMaps(maps ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// ValidateRequired checks if required fields are present and non-empty
func ValidateRequired(data map[string]interface{}, requiredFields []string) error {
	for _, field := range requiredFields {
		value, exists := data[field]
		if !exists {
			return fmt.Errorf("required field '%s' is missing", field)
		}
		
		// Check if value is empty
		if value == nil {
			return fmt.Errorf("required field '%s' is nil", field)
		}
		
		if str, ok := value.(string); ok && strings.TrimSpace(str) == "" {
			return fmt.Errorf("required field '%s' is empty", field)
		}
	}
	return nil
}

// SanitizeString removes potentially dangerous characters from strings
func SanitizeString(s string) string {
	// Remove null bytes and control characters
	result := strings.Map(func(r rune) rune {
		if r == 0 || (r < 32 && r != '\t' && r != '\n' && r != '\r') {
			return -1
		}
		return r
	}, s)
	
	return strings.TrimSpace(result)
}

// TruncateString truncates a string to a maximum length
func TruncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	
	if maxLength <= 3 {
		return s[:maxLength]
	}
	
	return s[:maxLength-3] + "..."
}

// GenerateRandomFloat generates a random float between 0 and 1
func GenerateRandomFloat() float64 {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	
	// Convert bytes to uint64 and normalize to [0, 1)
	var value uint64
	for i, b := range bytes {
		value |= uint64(b) << (8 * i)
	}
	
	return float64(value) / float64(^uint64(0))
}