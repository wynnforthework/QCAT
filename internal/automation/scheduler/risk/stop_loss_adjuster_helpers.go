package risk

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"qcat/internal/automation/scheduler/shared"
)

// calculateATR calculates Average True Range for a symbol
func (sla *StopLossAdjuster) calculateATR(ctx context.Context, symbol string) (*ATRCalculationResult, error) {
	// Check cache first
	if cached, exists := sla.atrCache[symbol]; exists && len(cached) > 0 {
		// Use cached data if it's recent (within 5 minutes)
		if time.Since(sla.lastCheck) < 5*time.Minute {
			return &ATRCalculationResult{
				Symbol:        symbol,
				Period:        20, // Default period
				CurrentATR:    cached[len(cached)-1],
				ATRValues:     cached,
				ATRPercentile: shared.CalculatePercentile(cached, 50),
				Trend:         sla.calculateATRTrend(cached),
				Timestamp:     time.Now(),
			}, nil
		}
	}

	// Get OHLC data for ATR calculation
	ohlcData, err := sla.getOHLCData(ctx, symbol, 50) // Get 50 periods for calculation
	if err != nil {
		return nil, fmt.Errorf("failed to get OHLC data: %w", err)
	}

	if len(ohlcData.Highs) < 21 { // Need at least 21 periods for 20-period ATR
		return nil, fmt.Errorf("insufficient data for ATR calculation: got %d periods, need at least 21", len(ohlcData.Highs))
	}

	// Calculate ATR using shared utility
	atrValues := shared.CalculateATR(ohlcData.Highs, ohlcData.Lows, ohlcData.Closes, 20)
	if len(atrValues) == 0 {
		return nil, fmt.Errorf("ATR calculation returned no values")
	}

	currentATR := atrValues[len(atrValues)-1]
	atrPercentile := shared.CalculatePercentile(atrValues, 50)
	trend := sla.calculateATRTrend(atrValues)

	// Cache the result
	sla.atrCache[symbol] = atrValues

	result := &ATRCalculationResult{
		Symbol:        symbol,
		Period:        20,
		CurrentATR:    currentATR,
		ATRValues:     atrValues,
		ATRPercentile: atrPercentile,
		Trend:         trend,
		Timestamp:     time.Now(),
	}

	return result, nil
}

// calculateRV calculates Realized Volatility for a symbol
func (sla *StopLossAdjuster) calculateRV(ctx context.Context, symbol string) (*RVCalculationResult, error) {
	// Check cache first
	if cached, exists := sla.rvCache[symbol]; exists && len(cached) > 0 {
		// Use cached data if it's recent (within 5 minutes)
		if time.Since(sla.lastCheck) < 5*time.Minute {
			return &RVCalculationResult{
				Symbol:       symbol,
				Period:       20, // Default period
				CurrentRV:    cached[len(cached)-1],
				RVValues:     cached,
				RVPercentile: shared.CalculatePercentile(cached, 50),
				Trend:        sla.calculateRVTrend(cached),
				Timestamp:    time.Now(),
			}, nil
		}
	}

	// Get price data for RV calculation
	prices, err := sla.getHistoricalPrices(ctx, symbol, 50) // Get 50 periods for calculation
	if err != nil {
		return nil, fmt.Errorf("failed to get price data: %w", err)
	}

	if len(prices) < 21 { // Need at least 21 periods for 20-period RV
		return nil, fmt.Errorf("insufficient data for RV calculation: got %d periods, need at least 21", len(prices))
	}

	// Calculate RV using shared utility
	rvValues := shared.CalculateRealizedVolatility(prices, 20)
	if len(rvValues) == 0 {
		return nil, fmt.Errorf("RV calculation returned no values")
	}

	currentRV := rvValues[len(rvValues)-1]
	rvPercentile := shared.CalculatePercentile(rvValues, 50)
	trend := sla.calculateRVTrend(rvValues)

	// Cache the result
	sla.rvCache[symbol] = rvValues

	result := &RVCalculationResult{
		Symbol:       symbol,
		Period:       20,
		CurrentRV:    currentRV,
		RVValues:     rvValues,
		RVPercentile: rvPercentile,
		Trend:        trend,
		Timestamp:    time.Now(),
	}

	return result, nil
}

// detectMarketRegime detects the current market regime using statistical methods
func (sla *StopLossAdjuster) detectMarketRegime(marketData *MarketDataForRegime) *shared.MarketRegime {
	if len(marketData.Prices) < 20 {
		// Default regime if insufficient data
		return &shared.MarketRegime{
			Type:       "SIDEWAYS",
			Confidence: 0.5,
			Volatility: 0.2,
			Trend:      0.0,
			Momentum:   0.0,
			Timestamp:  time.Now(),
		}
	}

	// Calculate price returns
	returns := make([]float64, len(marketData.Prices)-1)
	for i := 1; i < len(marketData.Prices); i++ {
		returns[i-1] = math.Log(marketData.Prices[i] / marketData.Prices[i-1])
	}

	// Calculate volatility (annualized)
	volatility := shared.CalculateStandardDeviation(returns) * math.Sqrt(252)

	// Calculate trend using linear regression slope
	trend := sla.calculateTrendSlope(marketData.Prices)

	// Calculate momentum using price rate of change
	momentum := sla.calculateMomentum(marketData.Prices, 10)

	// Determine regime type based on volatility and trend
	regimeType := sla.classifyRegime(volatility, trend, momentum)

	// Calculate confidence based on consistency of indicators
	confidence := sla.calculateRegimeConfidence(volatility, trend, momentum, returns)

	return &shared.MarketRegime{
		Type:       regimeType,
		Confidence: confidence,
		Volatility: volatility,
		Trend:      trend,
		Momentum:   momentum,
		Timestamp:  time.Now(),
	}
}

// getCurrentPosition gets the current position for a symbol
func (sla *StopLossAdjuster) getCurrentPosition(ctx context.Context, symbol string) (*shared.Position, error) {
	query := `
		SELECT 
			id, symbol, side, size, entry_price, current_price,
			unrealized_pnl, realized_pnl, leverage, margin_used, created_at
		FROM positions 
		WHERE symbol = ? AND status = 'ACTIVE'
		ORDER BY created_at DESC
		LIMIT 1
	`
	
	row := sla.db.QueryRowContext(ctx, query, symbol)
	
	var pos shared.Position
	err := row.Scan(
		&pos.ID, &pos.Symbol, &pos.Side, &pos.Size, &pos.EntryPrice,
		&pos.CurrentPrice, &pos.UnrealizedPnL, &pos.RealizedPnL,
		&pos.Leverage, &pos.MarginUsed, &pos.Timestamp,
	)
	
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil // No position found
		}
		return nil, err
	}

	return &pos, nil
}

// executeStopLossAdjustment executes a single stop loss adjustment
func (sla *StopLossAdjuster) executeStopLossAdjustment(ctx context.Context, adjustment StopLossAdjustment) error {
	// Update the position's stop loss in the database
	query := `
		UPDATE positions 
		SET stop_loss = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'ACTIVE'
	`
	
	result, err := sla.db.ExecContext(ctx, query, adjustment.NewLevel, adjustment.PositionID)
	if err != nil {
		return fmt.Errorf("failed to update stop loss in database: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no active position found with ID %s", adjustment.PositionID)
	}

	// Log the adjustment
	err = sla.logStopLossAdjustment(ctx, adjustment)
	if err != nil {
		log.Printf("Warning: Failed to log stop loss adjustment: %v", err)
	}

	// Send order to exchange if needed (simplified - would need actual exchange integration)
	err = sla.sendStopLossOrderToExchange(ctx, adjustment)
	if err != nil {
		log.Printf("Warning: Failed to send stop loss order to exchange: %v", err)
		// Don't fail the entire operation for exchange errors
	}

	return nil
}

// getATRMultiplier returns the ATR multiplier based on ATR percentile
func (sla *StopLossAdjuster) getATRMultiplier(atrPercentile float64) float64 {
	// Dynamic multiplier based on ATR percentile
	// Higher percentile (higher ATR) = larger multiplier for wider stops
	baseMultiplier := 2.0
	
	if atrPercentile >= 80 {
		return baseMultiplier * 1.5 // Very high volatility
	} else if atrPercentile >= 60 {
		return baseMultiplier * 1.2 // High volatility
	} else if atrPercentile >= 40 {
		return baseMultiplier * 1.0 // Normal volatility
	} else if atrPercentile >= 20 {
		return baseMultiplier * 0.8 // Low volatility
	} else {
		return baseMultiplier * 0.6 // Very low volatility
	}
}

// getRVMultiplier returns the RV multiplier based on RV percentile
func (sla *StopLossAdjuster) getRVMultiplier(rvPercentile float64) float64 {
	// Dynamic multiplier based on RV percentile
	// Higher percentile (higher volatility) = larger multiplier for wider stops
	baseMultiplier := 2.0
	
	if rvPercentile >= 80 {
		return baseMultiplier * 1.5 // Very high volatility
	} else if rvPercentile >= 60 {
		return baseMultiplier * 1.2 // High volatility
	} else if rvPercentile >= 40 {
		return baseMultiplier * 1.0 // Normal volatility
	} else if rvPercentile >= 20 {
		return baseMultiplier * 0.8 // Low volatility
	} else {
		return baseMultiplier * 0.6 // Very low volatility
	}
}

// calculateTrendAdjustment calculates trend adjustment factor
func (sla *StopLossAdjuster) calculateTrendAdjustment(trend float64) float64 {
	// Trend ranges from -1 to 1
	// Positive trend = tighten stops slightly
	// Negative trend = widen stops slightly
	maxAdjustment := 0.1 // 10% maximum adjustment
	return -trend * maxAdjustment // Negative because we want opposite effect
}

// applyTrendAdjustment applies trend adjustment to stop loss level
func (sla *StopLossAdjuster) applyTrendAdjustment(stopLoss, adjustment float64, side string) float64 {
	if side == "LONG" {
		// For long positions, positive adjustment moves stop loss up (tighter)
		return stopLoss * (1 + adjustment)
	} else {
		// For short positions, positive adjustment moves stop loss down (tighter)
		return stopLoss * (1 - adjustment)
	}
}

// calculateATRTrend calculates the trend of ATR values
func (sla *StopLossAdjuster) calculateATRTrend(atrValues []float64) float64 {
	if len(atrValues) < 5 {
		return 0.0
	}

	// Use last 5 values to calculate trend
	recent := atrValues[len(atrValues)-5:]
	return sla.calculateTrendSlope(recent)
}

// calculateRVTrend calculates the trend of RV values
func (sla *StopLossAdjuster) calculateRVTrend(rvValues []float64) float64 {
	if len(rvValues) < 5 {
		return 0.0
	}

	// Use last 5 values to calculate trend
	recent := rvValues[len(rvValues)-5:]
	return sla.calculateTrendSlope(recent)
}

// calculateTrendSlope calculates the slope of a price/value series
func (sla *StopLossAdjuster) calculateTrendSlope(values []float64) float64 {
	if len(values) < 2 {
		return 0.0
	}

	n := float64(len(values))
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumX2 := 0.0

	for i, y := range values {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// Calculate slope using least squares method
	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		return 0.0
	}

	slope := (n*sumXY - sumX*sumY) / denominator
	
	// Normalize slope to [-1, 1] range
	maxSlope := (values[len(values)-1] - values[0]) / float64(len(values)-1)
	if maxSlope == 0 {
		return 0.0
	}
	
	normalizedSlope := slope / math.Abs(maxSlope)
	if normalizedSlope > 1 {
		normalizedSlope = 1
	} else if normalizedSlope < -1 {
		normalizedSlope = -1
	}

	return normalizedSlope
}

// calculateMomentum calculates price momentum
func (sla *StopLossAdjuster) calculateMomentum(prices []float64, period int) float64 {
	if len(prices) < period+1 {
		return 0.0
	}

	current := prices[len(prices)-1]
	past := prices[len(prices)-1-period]
	
	if past == 0 {
		return 0.0
	}

	momentum := (current - past) / past
	
	// Normalize to [-1, 1] range (assuming max 100% change)
	if momentum > 1 {
		momentum = 1
	} else if momentum < -1 {
		momentum = -1
	}

	return momentum
}

// classifyRegime classifies market regime based on indicators
func (sla *StopLossAdjuster) classifyRegime(volatility, trend, momentum float64) string {
	// Volatility thresholds
	highVolThreshold := 0.35  // 35% annualized

	// Trend thresholds
	trendThreshold := 0.3

	if volatility > highVolThreshold {
		return "VOLATILE"
	} else if math.Abs(trend) > trendThreshold {
		if trend > 0 {
			return "BULL"
		} else {
			return "BEAR"
		}
	} else {
		return "SIDEWAYS"
	}
}

// calculateRegimeConfidence calculates confidence in regime classification
func (sla *StopLossAdjuster) calculateRegimeConfidence(volatility, trend, momentum float64, returns []float64) float64 {
	// Base confidence
	confidence := 0.5

	// Increase confidence based on consistency of indicators
	if math.Abs(trend) > 0.2 && math.Abs(momentum) > 0.1 {
		if (trend > 0 && momentum > 0) || (trend < 0 && momentum < 0) {
			confidence += 0.2 // Trend and momentum agree
		}
	}

	// Increase confidence based on volatility consistency
	recentVol := shared.CalculateStandardDeviation(returns[len(returns)-10:]) * math.Sqrt(252)
	if math.Abs(volatility-recentVol) < 0.05 {
		confidence += 0.1 // Consistent volatility
	}

	// Decrease confidence for very low volatility (harder to classify)
	if volatility < 0.1 {
		confidence -= 0.1
	}

	// Ensure confidence is in [0, 1] range
	if confidence > 1 {
		confidence = 1
	} else if confidence < 0 {
		confidence = 0
	}

	return confidence
}

// Data structures for helper functions

// OHLCData represents OHLC market data
type OHLCData struct {
	Timestamps []time.Time `json:"timestamps"`
	Opens      []float64   `json:"opens"`
	Highs      []float64   `json:"highs"`
	Lows       []float64   `json:"lows"`
	Closes     []float64   `json:"closes"`
	Volumes    []float64   `json:"volumes"`
}

// MarketDataForRegime represents market data for regime analysis
type MarketDataForRegime struct {
	Prices    []float64   `json:"prices"`
	Volumes   []float64   `json:"volumes"`
	Timestamps []time.Time `json:"timestamps"`
}

// getOHLCData retrieves OHLC data for a symbol
func (sla *StopLossAdjuster) getOHLCData(ctx context.Context, symbol string, periods int) (*OHLCData, error) {
	query := `
		SELECT timestamp, open_price, high_price, low_price, close_price, volume
		FROM market_data 
		WHERE symbol = ? 
		ORDER BY timestamp DESC 
		LIMIT ?
	`
	
	rows, err := sla.db.QueryContext(ctx, query, symbol, periods)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data OHLCData
	for rows.Next() {
		var timestamp time.Time
		var open, high, low, close, volume float64
		
		if err := rows.Scan(&timestamp, &open, &high, &low, &close, &volume); err != nil {
			return nil, err
		}
		
		// Prepend to maintain chronological order
		data.Timestamps = append([]time.Time{timestamp}, data.Timestamps...)
		data.Opens = append([]float64{open}, data.Opens...)
		data.Highs = append([]float64{high}, data.Highs...)
		data.Lows = append([]float64{low}, data.Lows...)
		data.Closes = append([]float64{close}, data.Closes...)
		data.Volumes = append([]float64{volume}, data.Volumes...)
	}

	return &data, nil
}

// getHistoricalPrices retrieves historical prices for a symbol
func (sla *StopLossAdjuster) getHistoricalPrices(ctx context.Context, symbol string, periods int) ([]float64, error) {
	query := `
		SELECT close_price
		FROM market_data 
		WHERE symbol = ? 
		ORDER BY timestamp DESC 
		LIMIT ?
	`
	
	rows, err := sla.db.QueryContext(ctx, query, symbol, periods)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prices []float64
	for rows.Next() {
		var price float64
		if err := rows.Scan(&price); err != nil {
			return nil, err
		}
		// Prepend to maintain chronological order
		prices = append([]float64{price}, prices...)
	}

	return prices, nil
}

// getMarketDataForRegimeAnalysis retrieves market data for regime analysis
func (sla *StopLossAdjuster) getMarketDataForRegimeAnalysis(ctx context.Context) (*MarketDataForRegime, error) {
	// Get data for major market symbols (simplified - would use market index in practice)
	query := `
		SELECT close_price, volume, timestamp
		FROM market_data 
		WHERE symbol IN ('BTCUSDT', 'ETHUSDT') 
		ORDER BY timestamp DESC 
		LIMIT 100
	`
	
	rows, err := sla.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data MarketDataForRegime
	for rows.Next() {
		var price, volume float64
		var timestamp time.Time
		
		if err := rows.Scan(&price, &volume, &timestamp); err != nil {
			return nil, err
		}
		
		// Prepend to maintain chronological order
		data.Prices = append([]float64{price}, data.Prices...)
		data.Volumes = append([]float64{volume}, data.Volumes...)
		data.Timestamps = append([]time.Time{timestamp}, data.Timestamps...)
	}

	return &data, nil
}

// logStopLossAdjustment logs a stop loss adjustment
func (sla *StopLossAdjuster) logStopLossAdjustment(ctx context.Context, adjustment StopLossAdjustment) error {
	query := `
		INSERT INTO stop_loss_adjustments 
		(position_id, symbol, old_level, new_level, adjustment_type, reason, priority, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`
	
	_, err := sla.db.ExecContext(ctx, query,
		adjustment.PositionID, adjustment.Symbol, adjustment.OldLevel,
		adjustment.NewLevel, adjustment.AdjustmentType, adjustment.Reason, adjustment.Priority)
	
	return err
}

// sendStopLossOrderToExchange sends stop loss order to exchange (simplified)
func (sla *StopLossAdjuster) sendStopLossOrderToExchange(ctx context.Context, adjustment StopLossAdjustment) error {
	// This would integrate with the actual exchange API
	// For now, just log the action
	log.Printf("Sending stop loss order to exchange: Position=%s, Symbol=%s, Level=%.6f", 
		adjustment.PositionID, adjustment.Symbol, adjustment.NewLevel)
	
	// Simulate exchange API call
	time.Sleep(100 * time.Millisecond)
	
	return nil
}

// updateATRMetrics updates ATR-related metrics
func (sla *StopLossAdjuster) updateATRMetrics(symbol string, atrResult *ATRCalculationResult, stopLoss float64) {
	sla.metrics[fmt.Sprintf("atr_%s", symbol)] = atrResult.CurrentATR
	sla.metrics[fmt.Sprintf("atr_percentile_%s", symbol)] = atrResult.ATRPercentile
	sla.metrics[fmt.Sprintf("atr_trend_%s", symbol)] = atrResult.Trend
	sla.metrics[fmt.Sprintf("atr_stop_loss_%s", symbol)] = stopLoss
	sla.metrics["last_atr_calculation"] = atrResult.Timestamp
}

// updateRVMetrics updates RV-related metrics
func (sla *StopLossAdjuster) updateRVMetrics(symbol string, rvResult *RVCalculationResult, stopLoss float64) {
	sla.metrics[fmt.Sprintf("rv_%s", symbol)] = rvResult.CurrentRV
	sla.metrics[fmt.Sprintf("rv_percentile_%s", symbol)] = rvResult.RVPercentile
	sla.metrics[fmt.Sprintf("rv_trend_%s", symbol)] = rvResult.Trend
	sla.metrics[fmt.Sprintf("rv_stop_loss_%s", symbol)] = stopLoss
	sla.metrics["last_rv_calculation"] = rvResult.Timestamp
}

// updateRegimeMetrics updates regime-related metrics
func (sla *StopLossAdjuster) updateRegimeMetrics(regime *shared.MarketRegime) {
	sla.metrics["market_regime_type"] = regime.Type
	sla.metrics["market_regime_confidence"] = regime.Confidence
	sla.metrics["market_volatility"] = regime.Volatility
	sla.metrics["market_trend"] = regime.Trend
	sla.metrics["market_momentum"] = regime.Momentum
	sla.metrics["last_regime_check"] = regime.Timestamp
}