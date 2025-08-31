package risk

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"qcat/internal/automation/scheduler/shared"
)

// OrderBook represents order book data
type OrderBook struct {
	Symbol    string       `json:"symbol"`
	Bids      []PriceLevel `json:"bids"`
	Asks      []PriceLevel `json:"asks"`
	Timestamp time.Time    `json:"timestamp"`
}

// PriceLevel represents a price level in the order book
type PriceLevel struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
}

// Helper methods for AbnormalMarketDetector

// getActiveSymbols retrieves active trading symbols from the database
func (amd *AbnormalMarketDetector) getActiveSymbols(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT symbol 
		FROM positions 
		WHERE status = 'ACTIVE' 
		UNION 
		SELECT DISTINCT symbol 
		FROM market_data 
		WHERE timestamp > NOW() - INTERVAL '1 hour'
		ORDER BY symbol
	`

	rows, err := amd.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			return nil, err
		}
		symbols = append(symbols, symbol)
	}

	return symbols, nil
}

// getRecentPrices retrieves recent price data for a symbol
func (amd *AbnormalMarketDetector) getRecentPrices(ctx context.Context, symbol string, periods int) ([]float64, error) {
	query := `
		SELECT price 
		FROM market_data 
		WHERE symbol = $1 
		ORDER BY timestamp DESC 
		LIMIT $2
	`

	rows, err := amd.db.QueryContext(ctx, query, symbol, periods)
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
		prices = append(prices, price)
	}

	// Reverse to get chronological order
	for i, j := 0, len(prices)-1; i < j; i, j = i+1, j-1 {
		prices[i], prices[j] = prices[j], prices[i]
	}

	return prices, nil
}

// getCurrentOrderBook retrieves current order book data for a symbol
func (amd *AbnormalMarketDetector) getCurrentOrderBook(ctx context.Context, symbol string) (*OrderBook, error) {
	// Get bids
	bidsQuery := `
		SELECT price, quantity 
		FROM order_book 
		WHERE symbol = $1 AND side = 'BUY' 
		ORDER BY price DESC 
		LIMIT 20
	`

	bidsRows, err := amd.db.QueryContext(ctx, bidsQuery, symbol)
	if err != nil {
		return nil, err
	}
	defer bidsRows.Close()

	var bids []PriceLevel
	for bidsRows.Next() {
		var level PriceLevel
		if err := bidsRows.Scan(&level.Price, &level.Quantity); err != nil {
			return nil, err
		}
		bids = append(bids, level)
	}

	// Get asks
	asksQuery := `
		SELECT price, quantity 
		FROM order_book 
		WHERE symbol = $1 AND side = 'SELL' 
		ORDER BY price ASC 
		LIMIT 20
	`

	asksRows, err := amd.db.QueryContext(ctx, asksQuery, symbol)
	if err != nil {
		return nil, err
	}
	defer asksRows.Close()

	var asks []PriceLevel
	for asksRows.Next() {
		var level PriceLevel
		if err := asksRows.Scan(&level.Price, &level.Quantity); err != nil {
			return nil, err
		}
		asks = append(asks, level)
	}

	return &OrderBook{
		Symbol:    symbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: time.Now(),
	}, nil
}

// getHistoricalLiquidity retrieves historical liquidity data for a symbol
func (amd *AbnormalMarketDetector) getHistoricalLiquidity(ctx context.Context, symbol string, periods int) ([]float64, error) {
	query := `
		SELECT liquidity_metric 
		FROM market_metrics 
		WHERE symbol = $1 
		ORDER BY timestamp DESC 
		LIMIT $2
	`

	rows, err := amd.db.QueryContext(ctx, query, symbol, periods)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var liquidity []float64
	for rows.Next() {
		var metric float64
		if err := rows.Scan(&metric); err != nil {
			return nil, err
		}
		liquidity = append(liquidity, metric)
	}

	return liquidity, nil
}

// getCorrelationPairs retrieves asset pairs for correlation analysis
func (amd *AbnormalMarketDetector) getCorrelationPairs(ctx context.Context) ([][]string, error) {
	// Get major trading pairs for correlation analysis
	query := `
		SELECT DISTINCT p1.symbol, p2.symbol 
		FROM positions p1 
		CROSS JOIN positions p2 
		WHERE p1.symbol < p2.symbol 
		AND p1.status = 'ACTIVE' 
		AND p2.status = 'ACTIVE'
		LIMIT 10
	`

	rows, err := amd.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pairs [][]string
	for rows.Next() {
		var symbol1, symbol2 string
		if err := rows.Scan(&symbol1, &symbol2); err != nil {
			return nil, err
		}
		pairs = append(pairs, []string{symbol1, symbol2})
	}

	// Add some default major pairs if no positions exist
	if len(pairs) == 0 {
		pairs = [][]string{
			{"BTCUSDT", "ETHUSDT"},
			{"BTCUSDT", "BNBUSDT"},
			{"ETHUSDT", "BNBUSDT"},
		}
	}

	return pairs, nil
}

// calculateLiquidityMetric calculates a liquidity metric from order book data
func (amd *AbnormalMarketDetector) calculateLiquidityMetric(orderBook *OrderBook) float64 {
	if len(orderBook.Bids) == 0 || len(orderBook.Asks) == 0 {
		return 0.0
	}

	// Calculate total volume in top 5 levels on each side
	var bidVolume, askVolume float64

	maxLevels := 5
	if len(orderBook.Bids) < maxLevels {
		maxLevels = len(orderBook.Bids)
	}

	for i := 0; i < maxLevels && i < len(orderBook.Bids); i++ {
		bidVolume += orderBook.Bids[i].Quantity
	}

	maxLevels = 5
	if len(orderBook.Asks) < maxLevels {
		maxLevels = len(orderBook.Asks)
	}

	for i := 0; i < maxLevels && i < len(orderBook.Asks); i++ {
		askVolume += orderBook.Asks[i].Quantity
	}

	// Return average of bid and ask volume as liquidity metric
	return (bidVolume + askVolume) / 2.0
}

// calculateBidAskSpread calculates the bid-ask spread
func (amd *AbnormalMarketDetector) calculateBidAskSpread(orderBook *OrderBook) float64 {
	if len(orderBook.Bids) == 0 || len(orderBook.Asks) == 0 {
		return 0.0
	}

	bestBid := orderBook.Bids[0].Price
	bestAsk := orderBook.Asks[0].Price

	if bestAsk > 0 {
		return (bestAsk - bestBid) / bestAsk
	}

	return 0.0
}

// calculateReturns calculates returns from price series
func (amd *AbnormalMarketDetector) calculateReturns(prices []float64) []float64 {
	if len(prices) < 2 {
		return []float64{}
	}

	returns := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		if prices[i-1] > 0 {
			returns[i-1] = math.Log(prices[i] / prices[i-1])
		}
	}

	return returns
}

// determineVolatilitySeverity determines severity based on volatility ratio
func (amd *AbnormalMarketDetector) determineVolatilitySeverity(volRatio float64) shared.AlertSeverity {
	if volRatio >= 5.0 {
		return shared.AlertSeverityCritical
	} else if volRatio >= 3.0 {
		return shared.AlertSeverityHigh
	} else if volRatio >= 2.0 {
		return shared.AlertSeverityMedium
	}
	return shared.AlertSeverityLow
}

// determineLiquiditySeverity determines severity based on liquidity ratio and spread
func (amd *AbnormalMarketDetector) determineLiquiditySeverity(liquidityRatio, bidAskSpread float64) shared.AlertSeverity {
	// Lower liquidity ratio means worse liquidity
	// Higher bid-ask spread means worse liquidity

	if liquidityRatio <= 0.2 || bidAskSpread >= 0.05 {
		return shared.AlertSeverityCritical
	} else if liquidityRatio <= 0.4 || bidAskSpread >= 0.03 {
		return shared.AlertSeverityHigh
	} else if liquidityRatio <= 0.6 || bidAskSpread >= 0.02 {
		return shared.AlertSeverityMedium
	}
	return shared.AlertSeverityLow
}

// determineCorrelationSeverity determines severity based on correlation change
func (amd *AbnormalMarketDetector) determineCorrelationSeverity(corrChange, currentCorr, historicalCorr float64) shared.AlertSeverity {
	// Large correlation changes indicate breakdown
	if corrChange >= 0.7 {
		return shared.AlertSeverityCritical
	} else if corrChange >= 0.5 {
		return shared.AlertSeverityHigh
	} else if corrChange >= 0.3 {
		return shared.AlertSeverityMedium
	}
	return shared.AlertSeverityLow
}

// generateVolatilityRecommendations generates recommendations for volatility alerts
func (amd *AbnormalMarketDetector) generateVolatilityRecommendations(symbol string, volRatio float64, severity shared.AlertSeverity) []string {
	var recommendations []string

	switch severity {
	case shared.AlertSeverityCritical:
		recommendations = append(recommendations, fmt.Sprintf("URGENT: Extreme volatility spike detected for %s (%.1fx normal)", symbol, volRatio))
		recommendations = append(recommendations, "Consider immediate position reduction or closure")
		recommendations = append(recommendations, "Activate emergency risk controls")
		recommendations = append(recommendations, "Increase monitoring frequency")
	case shared.AlertSeverityHigh:
		recommendations = append(recommendations, fmt.Sprintf("High volatility detected for %s (%.1fx normal)", symbol, volRatio))
		recommendations = append(recommendations, "Reduce position sizes")
		recommendations = append(recommendations, "Tighten stop-loss levels")
		recommendations = append(recommendations, "Monitor closely for further developments")
	case shared.AlertSeverityMedium:
		recommendations = append(recommendations, fmt.Sprintf("Elevated volatility for %s (%.1fx normal)", symbol, volRatio))
		recommendations = append(recommendations, "Review position risk parameters")
		recommendations = append(recommendations, "Consider volatility-based position sizing")
	default:
		recommendations = append(recommendations, "Monitor volatility trends")
	}

	return recommendations
}

// generateLiquidityRecommendations generates recommendations for liquidity alerts
func (amd *AbnormalMarketDetector) generateLiquidityRecommendations(symbol string, liquidityRatio float64, severity shared.AlertSeverity) []string {
	var recommendations []string

	switch severity {
	case shared.AlertSeverityCritical:
		recommendations = append(recommendations, fmt.Sprintf("CRITICAL: Severe liquidity drop for %s (%.1f%% of normal)", symbol, liquidityRatio*100))
		recommendations = append(recommendations, "Avoid large orders - high market impact risk")
		recommendations = append(recommendations, "Consider position closure if liquidity doesn't recover")
		recommendations = append(recommendations, "Use smaller order sizes with longer execution times")
	case shared.AlertSeverityHigh:
		recommendations = append(recommendations, fmt.Sprintf("Low liquidity detected for %s (%.1f%% of normal)", symbol, liquidityRatio*100))
		recommendations = append(recommendations, "Reduce order sizes")
		recommendations = append(recommendations, "Use limit orders instead of market orders")
		recommendations = append(recommendations, "Monitor order book depth closely")
	case shared.AlertSeverityMedium:
		recommendations = append(recommendations, fmt.Sprintf("Reduced liquidity for %s (%.1f%% of normal)", symbol, liquidityRatio*100))
		recommendations = append(recommendations, "Exercise caution with large orders")
		recommendations = append(recommendations, "Consider alternative execution strategies")
	default:
		recommendations = append(recommendations, "Monitor liquidity conditions")
	}

	return recommendations
}

// generateCorrelationRecommendations generates recommendations for correlation alerts
func (amd *AbnormalMarketDetector) generateCorrelationRecommendations(pair []string, corrChange float64, severity shared.AlertSeverity) []string {
	var recommendations []string
	pairStr := fmt.Sprintf("%s/%s", pair[0], pair[1])

	switch severity {
	case shared.AlertSeverityCritical:
		recommendations = append(recommendations, fmt.Sprintf("CRITICAL: Major correlation breakdown for %s (%.1f%% change)", pairStr, corrChange*100))
		recommendations = append(recommendations, "Review hedging strategies immediately")
		recommendations = append(recommendations, "Diversification benefits may be compromised")
		recommendations = append(recommendations, "Consider emergency portfolio rebalancing")
	case shared.AlertSeverityHigh:
		recommendations = append(recommendations, fmt.Sprintf("Significant correlation change for %s (%.1f%% change)", pairStr, corrChange*100))
		recommendations = append(recommendations, "Reassess portfolio risk calculations")
		recommendations = append(recommendations, "Update correlation matrices")
		recommendations = append(recommendations, "Review multi-asset strategies")
	case shared.AlertSeverityMedium:
		recommendations = append(recommendations, fmt.Sprintf("Correlation shift detected for %s (%.1f%% change)", pairStr, corrChange*100))
		recommendations = append(recommendations, "Monitor correlation stability")
		recommendations = append(recommendations, "Consider correlation-based adjustments")
	default:
		recommendations = append(recommendations, "Continue monitoring correlation trends")
	}

	return recommendations
}

// determineCircuitBreakerActions determines actions based on alert severity
func (amd *AbnormalMarketDetector) determineCircuitBreakerActions(severity shared.AlertSeverity) []string {
	var actions []string

	switch severity {
	case shared.AlertSeverityCritical:
		actions = append(actions, "HALT_NEW_POSITIONS")
		actions = append(actions, "REDUCE_POSITION_SIZES")
		actions = append(actions, "ACTIVATE_EMERGENCY_HEDGING")
		actions = append(actions, "INCREASE_MONITORING_FREQUENCY")
		actions = append(actions, "NOTIFY_RISK_MANAGERS")
	case shared.AlertSeverityHigh:
		actions = append(actions, "LIMIT_NEW_POSITIONS")
		actions = append(actions, "TIGHTEN_RISK_PARAMETERS")
		actions = append(actions, "ACTIVATE_PROTECTIVE_MEASURES")
		actions = append(actions, "INCREASE_MONITORING")
	case shared.AlertSeverityMedium:
		actions = append(actions, "REVIEW_RISK_PARAMETERS")
		actions = append(actions, "INCREASE_MONITORING")
		actions = append(actions, "PREPARE_PROTECTIVE_MEASURES")
	default:
		actions = append(actions, "CONTINUE_MONITORING")
	}

	return actions
}

// executeCircuitBreakerAction executes a specific circuit breaker action
func (amd *AbnormalMarketDetector) executeCircuitBreakerAction(ctx context.Context, action string) error {
	log.Printf("Executing circuit breaker action: %s", action)

	switch action {
	case "HALT_NEW_POSITIONS":
		return amd.haltNewPositions(ctx)
	case "REDUCE_POSITION_SIZES":
		return amd.reducePositionSizes(ctx)
	case "ACTIVATE_EMERGENCY_HEDGING":
		return amd.activateEmergencyHedging(ctx)
	case "LIMIT_NEW_POSITIONS":
		return amd.limitNewPositions(ctx)
	case "TIGHTEN_RISK_PARAMETERS":
		return amd.tightenRiskParameters(ctx)
	case "ACTIVATE_PROTECTIVE_MEASURES":
		return amd.activateProtectiveMeasures(ctx)
	case "INCREASE_MONITORING_FREQUENCY":
		return amd.increaseMonitoringFrequency(ctx)
	case "NOTIFY_RISK_MANAGERS":
		return amd.notifyRiskManagers(ctx)
	default:
		log.Printf("Unknown circuit breaker action: %s", action)
		return nil
	}
}

// Circuit breaker action implementations

// haltNewPositions halts all new position creation
func (amd *AbnormalMarketDetector) haltNewPositions(ctx context.Context) error {
	query := `
		UPDATE system_settings 
		SET value = 'true' 
		WHERE key = 'halt_new_positions'
	`

	_, err := amd.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to halt new positions: %v", err)
	}

	log.Printf("New positions halted due to circuit breaker activation")
	return nil
}

// reducePositionSizes reduces existing position sizes
func (amd *AbnormalMarketDetector) reducePositionSizes(ctx context.Context) error {
	// This would integrate with position management system
	// For now, just log the action
	log.Printf("Position size reduction triggered by circuit breaker")

	// Update system setting to trigger position reduction
	query := `
		UPDATE system_settings 
		SET value = 'true' 
		WHERE key = 'reduce_position_sizes'
	`

	_, err := amd.db.ExecContext(ctx, query)
	return err
}

// activateEmergencyHedging activates emergency hedging mechanisms
func (amd *AbnormalMarketDetector) activateEmergencyHedging(ctx context.Context) error {
	log.Printf("Emergency hedging activated by circuit breaker")

	query := `
		UPDATE system_settings 
		SET value = 'true' 
		WHERE key = 'emergency_hedging_active'
	`

	_, err := amd.db.ExecContext(ctx, query)
	return err
}

// limitNewPositions limits new position creation
func (amd *AbnormalMarketDetector) limitNewPositions(ctx context.Context) error {
	query := `
		UPDATE system_settings 
		SET value = '0.5' 
		WHERE key = 'max_position_size_multiplier'
	`

	_, err := amd.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to limit new positions: %v", err)
	}

	log.Printf("New position limits activated")
	return nil
}

// tightenRiskParameters tightens risk management parameters
func (amd *AbnormalMarketDetector) tightenRiskParameters(ctx context.Context) error {
	log.Printf("Risk parameters tightened by circuit breaker")

	// Reduce maximum leverage
	query1 := `
		UPDATE system_settings 
		SET value = CAST((CAST(value AS FLOAT) * 0.7) AS TEXT)
		WHERE key = 'max_leverage'
	`

	// Tighten stop loss
	query2 := `
		UPDATE system_settings 
		SET value = CAST((CAST(value AS FLOAT) * 0.8) AS TEXT)
		WHERE key = 'stop_loss_percentage'
	`

	_, err1 := amd.db.ExecContext(ctx, query1)
	_, err2 := amd.db.ExecContext(ctx, query2)

	if err1 != nil {
		return err1
	}
	return err2
}

// activateProtectiveMeasures activates general protective measures
func (amd *AbnormalMarketDetector) activateProtectiveMeasures(ctx context.Context) error {
	log.Printf("Protective measures activated")

	query := `
		UPDATE system_settings 
		SET value = 'true' 
		WHERE key = 'protective_measures_active'
	`

	_, err := amd.db.ExecContext(ctx, query)
	return err
}

// increaseMonitoringFrequency increases monitoring frequency
func (amd *AbnormalMarketDetector) increaseMonitoringFrequency(ctx context.Context) error {
	log.Printf("Monitoring frequency increased")

	query := `
		UPDATE system_settings 
		SET value = '30' 
		WHERE key = 'monitoring_interval_seconds'
	`

	_, err := amd.db.ExecContext(ctx, query)
	return err
}

// notifyRiskManagers sends notifications to risk managers
func (amd *AbnormalMarketDetector) notifyRiskManagers(ctx context.Context) error {
	log.Printf("Risk managers notified of circuit breaker activation")

	// Insert notification record
	query := `
		INSERT INTO notifications (type, severity, message, created_at)
		VALUES ('CIRCUIT_BREAKER', 'CRITICAL', 'Circuit breaker activated due to abnormal market conditions', NOW())
	`

	_, err := amd.db.ExecContext(ctx, query)
	return err
}

// Update metrics methods

// updateVolatilityMetrics updates volatility-related metrics
func (amd *AbnormalMarketDetector) updateVolatilityMetrics(alert *VolatilityAlert) {
	amd.metrics["last_volatility_alert"] = alert.DetectedAt
	amd.metrics["volatility_alert_symbol"] = alert.Symbol
	amd.metrics["volatility_ratio"] = alert.VolRatio
	amd.metrics["volatility_severity"] = alert.Severity.String()
}

// updateLiquidityMetrics updates liquidity-related metrics
func (amd *AbnormalMarketDetector) updateLiquidityMetrics(alert *LiquidityAlert) {
	amd.metrics["last_liquidity_alert"] = alert.DetectedAt
	amd.metrics["liquidity_alert_symbol"] = alert.Symbol
	amd.metrics["liquidity_ratio"] = alert.LiquidityRatio
	amd.metrics["liquidity_severity"] = alert.Severity.String()
	amd.metrics["bid_ask_spread"] = alert.BidAskSpread
}

// updateCorrelationMetrics updates correlation-related metrics
func (amd *AbnormalMarketDetector) updateCorrelationMetrics(alert *CorrelationAlert) {
	amd.metrics["last_correlation_alert"] = alert.DetectedAt
	amd.metrics["correlation_alert_pairs"] = alert.AssetPairs
	amd.metrics["correlation_change"] = alert.CorrChange
	amd.metrics["correlation_severity"] = alert.Severity.String()
}

// updateCircuitBreakerMetrics updates circuit breaker metrics
func (amd *AbnormalMarketDetector) updateCircuitBreakerMetrics(severity shared.AlertSeverity, actions []string) {
	amd.metrics["last_circuit_breaker"] = time.Now()
	amd.metrics["circuit_breaker_severity"] = severity.String()
	amd.metrics["circuit_breaker_actions"] = actions
	amd.metrics["circuit_breaker_count"] = amd.getCircuitBreakerCount() + 1
}

// getCircuitBreakerCount gets the current circuit breaker activation count
func (amd *AbnormalMarketDetector) getCircuitBreakerCount() int {
	if count, exists := amd.metrics["circuit_breaker_count"]; exists {
		if intCount, ok := count.(int); ok {
			return intCount
		}
	}
	return 0
}
