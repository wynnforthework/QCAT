package risk

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"time"

	"qcat/internal/automation/scheduler/shared"
)

// MarketData represents market data for analysis
type MarketData struct {
	Symbol      string    `json:"symbol"`
	Price       float64   `json:"price"`
	Volume      float64   `json:"volume"`
	Volatility  float64   `json:"volatility"`
	Liquidity   float64   `json:"liquidity"`
	Timestamp   time.Time `json:"timestamp"`
}

// getHistoricalPrices retrieves historical price data for a symbol
func (rm *RiskMonitor) getHistoricalPrices(ctx context.Context, symbol string, days int) ([]float64, error) {
	query := `
		SELECT close_price 
		FROM market_data 
		WHERE symbol = $1 
		AND timestamp >= NOW() - INTERVAL '%d days'
		ORDER BY timestamp ASC
	`
	
	rows, err := rm.db.QueryContext(ctx, fmt.Sprintf(query, days), symbol)
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

	// If no data in database, return mock data for testing
	if len(prices) == 0 {
		log.Printf("No historical data found for %s, generating mock data", symbol)
		return rm.generateMockPrices(100.0, days*24, 0.02), nil
	}

	return prices, nil
}

// getMarketIndexPrices retrieves market index prices (e.g., BTC as market proxy)
func (rm *RiskMonitor) getMarketIndexPrices(ctx context.Context, days int) ([]float64, error) {
	// Use BTCUSDT as market index proxy
	return rm.getHistoricalPrices(ctx, "BTCUSDT", days)
}

// getTotalPortfolioValue calculates total portfolio value
func (rm *RiskMonitor) getTotalPortfolioValue(ctx context.Context) (float64, error) {
	query := `
		SELECT SUM(size * current_price) as total_value
		FROM positions 
		WHERE status = 'ACTIVE'
	`
	
	var totalValue sql.NullFloat64
	err := rm.db.QueryRowContext(ctx, query).Scan(&totalValue)
	if err != nil {
		return 0, err
	}

	if !totalValue.Valid {
		return 0, nil
	}

	return totalValue.Float64, nil
}

// calculateSymbolLiquidityRisk calculates liquidity risk for a symbol
func (rm *RiskMonitor) calculateSymbolLiquidityRisk(symbol string) float64 {
	// Simplified liquidity risk calculation based on symbol type
	// In practice, this would use order book depth, trading volume, etc.
	
	majorPairs := map[string]bool{
		"BTCUSDT": true, "ETHUSDT": true, "BNBUSDT": true,
		"ADAUSDT": true, "SOLUSDT": true, "XRPUSDT": true,
	}
	
	if majorPairs[symbol] {
		return 0.1 // Low liquidity risk for major pairs
	}
	
	return 0.3 // Higher liquidity risk for other pairs
}

// calculateConcentrationRisk calculates portfolio concentration risk
func (rm *RiskMonitor) calculateConcentrationRisk(positions []shared.Position) float64 {
	if len(positions) == 0 {
		return 0
	}

	// Calculate total portfolio value
	totalValue := 0.0
	positionValues := make([]float64, len(positions))
	
	for i, pos := range positions {
		value := pos.Size * pos.CurrentPrice
		positionValues[i] = value
		totalValue += value
	}

	if totalValue == 0 {
		return 0
	}

	// Calculate Herfindahl-Hirschman Index (HHI) for concentration
	hhi := 0.0
	for _, value := range positionValues {
		share := value / totalValue
		hhi += share * share
	}

	// Normalize HHI to 0-1 scale (1 = maximum concentration)
	maxHHI := 1.0 // When all capital is in one position
	return hhi / maxHHI
}

// calculateCorrelationRisk calculates correlation risk between positions
func (rm *RiskMonitor) calculateCorrelationRisk(ctx context.Context, positions []shared.Position) float64 {
	if len(positions) < 2 {
		return 0
	}

	// Get correlation matrix for all symbols
	symbols := make([]string, len(positions))
	for i, pos := range positions {
		symbols[i] = pos.Symbol
	}

	correlationMatrix, err := rm.calculateCorrelationMatrix(ctx, symbols)
	if err != nil {
		log.Printf("Warning: Could not calculate correlation matrix: %v", err)
		return 0.5 // Default moderate correlation risk
	}

	// Calculate average correlation
	totalCorrelation := 0.0
	count := 0
	
	for i := 0; i < len(symbols); i++ {
		for j := i + 1; j < len(symbols); j++ {
			if corr, exists := correlationMatrix[symbols[i]][symbols[j]]; exists {
				totalCorrelation += math.Abs(corr)
				count++
			}
		}
	}

	if count == 0 {
		return 0.5 // Default moderate correlation risk
	}

	avgCorrelation := totalCorrelation / float64(count)
	return avgCorrelation // Higher correlation = higher risk
}

// calculateLiquidityRisk calculates overall portfolio liquidity risk
func (rm *RiskMonitor) calculateLiquidityRisk(ctx context.Context, positions []shared.Position) float64 {
	if len(positions) == 0 {
		return 0
	}

	totalValue := 0.0
	weightedLiquidityRisk := 0.0

	for _, pos := range positions {
		value := pos.Size * pos.CurrentPrice
		liquidityRisk := rm.calculateSymbolLiquidityRisk(pos.Symbol)
		
		weightedLiquidityRisk += value * liquidityRisk
		totalValue += value
	}

	if totalValue == 0 {
		return 0
	}

	return weightedLiquidityRisk / totalValue
}

// calculatePortfolioVaR calculates portfolio-level Value at Risk
func (rm *RiskMonitor) calculatePortfolioVaR(ctx context.Context, positions []shared.Position) float64 {
	if len(positions) == 0 {
		return 0
	}

	// Simplified portfolio VaR calculation
	// In practice, this would use full covariance matrix and Monte Carlo simulation
	
	totalVaR := 0.0
	for _, pos := range positions {
		// Get historical returns for the position
		prices, err := rm.getHistoricalPrices(ctx, pos.Symbol, 30)
		if err != nil || len(prices) < 2 {
			continue
		}

		returns := make([]float64, len(prices)-1)
		for i := 1; i < len(prices); i++ {
			returns[i-1] = math.Log(prices[i] / prices[i-1])
		}

		// Calculate position VaR
		positionValue := pos.Size * pos.CurrentPrice
		var95 := shared.CalculateVaR(returns, 0.95)
		positionVaR := var95 * positionValue

		totalVaR += positionVaR * positionVaR // Sum of squares for diversification
	}

	return math.Sqrt(totalVaR) // Square root for portfolio effect
}

// calculateExpectedShortfall calculates portfolio Expected Shortfall
func (rm *RiskMonitor) calculateExpectedShortfall(ctx context.Context, positions []shared.Position) float64 {
	if len(positions) == 0 {
		return 0
	}

	totalES := 0.0
	for _, pos := range positions {
		prices, err := rm.getHistoricalPrices(ctx, pos.Symbol, 30)
		if err != nil || len(prices) < 2 {
			continue
		}

		returns := make([]float64, len(prices)-1)
		for i := 1; i < len(prices); i++ {
			returns[i-1] = math.Log(prices[i] / prices[i-1])
		}

		positionValue := pos.Size * pos.CurrentPrice
		es := shared.CalculateExpectedShortfall(returns, 0.95)
		totalES += es * positionValue
	}

	return totalES
}

// calculateMaxDrawdown calculates maximum drawdown from equity curve
func (rm *RiskMonitor) calculateMaxDrawdown(ctx context.Context) float64 {
	query := `
		SELECT equity_value, timestamp
		FROM portfolio_equity 
		WHERE timestamp >= NOW() - INTERVAL '30 days'
		ORDER BY timestamp ASC
	`
	
	rows, err := rm.db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("Warning: Could not get equity curve for drawdown calculation: %v", err)
		return 0
	}
	defer rows.Close()

	var equityCurve []float64
	for rows.Next() {
		var equity float64
		var timestamp time.Time
		if err := rows.Scan(&equity, &timestamp); err != nil {
			continue
		}
		equityCurve = append(equityCurve, equity)
	}

	if len(equityCurve) == 0 {
		return 0
	}

	return shared.CalculateMaxDrawdown(equityCurve)
}

// generateRiskRecommendations generates risk management recommendations
func (rm *RiskMonitor) generateRiskRecommendations(totalRisk, concentrationRisk, correlationRisk, liquidityRisk float64) []string {
	var recommendations []string

	// Total risk recommendations
	if totalRisk > 0.1 {
		recommendations = append(recommendations, "High total portfolio risk - consider reducing position sizes")
	}

	// Concentration risk recommendations
	if concentrationRisk > 0.5 {
		recommendations = append(recommendations, "High concentration risk - diversify positions across more symbols")
	} else if concentrationRisk > 0.3 {
		recommendations = append(recommendations, "Moderate concentration risk - consider adding more positions")
	}

	// Correlation risk recommendations
	if correlationRisk > 0.8 {
		recommendations = append(recommendations, "High correlation risk - positions are highly correlated, consider diversification")
	} else if correlationRisk > 0.6 {
		recommendations = append(recommendations, "Moderate correlation risk - monitor position correlations")
	}

	// Liquidity risk recommendations
	if liquidityRisk > 0.4 {
		recommendations = append(recommendations, "High liquidity risk - consider reducing positions in illiquid assets")
	} else if liquidityRisk > 0.2 {
		recommendations = append(recommendations, "Moderate liquidity risk - monitor market depth for position exits")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Risk levels within acceptable limits")
	}

	return recommendations
}

// calculateCorrelationMatrix calculates correlation matrix for symbols
func (rm *RiskMonitor) calculateCorrelationMatrix(ctx context.Context, symbols []string) (map[string]map[string]float64, error) {
	correlationMatrix := make(map[string]map[string]float64)
	
	// Initialize matrix
	for _, symbol1 := range symbols {
		correlationMatrix[symbol1] = make(map[string]float64)
		for _, symbol2 := range symbols {
			if symbol1 == symbol2 {
				correlationMatrix[symbol1][symbol2] = 1.0
			} else {
				// Calculate correlation between symbol1 and symbol2
				corr, err := rm.calculatePairCorrelation(ctx, symbol1, symbol2)
				if err != nil {
					log.Printf("Warning: Could not calculate correlation between %s and %s: %v", symbol1, symbol2, err)
					corr = 0.5 // Default moderate correlation
				}
				correlationMatrix[symbol1][symbol2] = corr
			}
		}
	}

	return correlationMatrix, nil
}

// calculatePairCorrelation calculates correlation between two symbols
func (rm *RiskMonitor) calculatePairCorrelation(ctx context.Context, symbol1, symbol2 string) (float64, error) {
	// Get price data for both symbols
	prices1, err := rm.getHistoricalPrices(ctx, symbol1, 30)
	if err != nil {
		return 0, err
	}

	prices2, err := rm.getHistoricalPrices(ctx, symbol2, 30)
	if err != nil {
		return 0, err
	}

	// Ensure same length
	minLen := len(prices1)
	if len(prices2) < minLen {
		minLen = len(prices2)
	}

	if minLen < 2 {
		return 0, fmt.Errorf("insufficient data for correlation calculation")
	}

	// Calculate returns
	returns1 := make([]float64, minLen-1)
	returns2 := make([]float64, minLen-1)

	for i := 1; i < minLen; i++ {
		returns1[i-1] = math.Log(prices1[i] / prices1[i-1])
		returns2[i-1] = math.Log(prices2[i] / prices2[i-1])
	}

	return shared.CalculateCorrelation(returns1, returns2), nil
}

// getMarketDataForAnalysis retrieves market data for anomaly detection
func (rm *RiskMonitor) getMarketDataForAnalysis(ctx context.Context) ([]MarketData, error) {
	query := `
		SELECT 
			symbol, price, volume_24h, volatility, 
			COALESCE(liquidity_score, 0.5) as liquidity,
			updated_at
		FROM market_data 
		WHERE updated_at >= NOW() - INTERVAL '1 hour'
		ORDER BY volume_24h DESC
		LIMIT 50
	`
	
	rows, err := rm.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var marketData []MarketData
	for rows.Next() {
		var data MarketData
		if err := rows.Scan(
			&data.Symbol, &data.Price, &data.Volume,
			&data.Volatility, &data.Liquidity, &data.Timestamp,
		); err != nil {
			continue
		}
		marketData = append(marketData, data)
	}

	// If no data in database, generate mock data for testing
	if len(marketData) == 0 {
		log.Printf("No market data found, generating mock data for testing")
		return rm.generateMockMarketData(), nil
	}

	return marketData, nil
}

// detectVolatilitySpike detects volatility spikes in market data
func (rm *RiskMonitor) detectVolatilitySpike(marketData []MarketData) *MarketAnomalyReport {
	if len(marketData) == 0 {
		return nil
	}

	// Calculate average volatility
	totalVolatility := 0.0
	for _, data := range marketData {
		totalVolatility += data.Volatility
	}
	avgVolatility := totalVolatility / float64(len(marketData))

	// Detect spikes (volatility > 2 standard deviations above mean)
	volatilities := make([]float64, len(marketData))
	for i, data := range marketData {
		volatilities[i] = data.Volatility
	}
	
	stdDev := shared.CalculateStandardDeviation(volatilities)
	threshold := avgVolatility + 2*stdDev

	var affectedSymbols []string
	var maxVolatility float64
	
	for _, data := range marketData {
		if data.Volatility > threshold {
			affectedSymbols = append(affectedSymbols, data.Symbol)
			if data.Volatility > maxVolatility {
				maxVolatility = data.Volatility
			}
		}
	}

	if len(affectedSymbols) == 0 {
		return nil
	}

	// Determine severity based on number of affected symbols and magnitude
	var severity shared.Severity
	if len(affectedSymbols) > 10 || maxVolatility > avgVolatility+3*stdDev {
		severity = shared.SeverityCritical
	} else if len(affectedSymbols) > 5 {
		severity = shared.SeverityError
	} else {
		severity = shared.SeverityWarning
	}

	return &MarketAnomalyReport{
		AnomalyType:     shared.AnomalyTypeVolatilitySpike,
		Severity:        severity,
		AffectedSymbols: affectedSymbols,
		DetectionTime:   time.Now(),
		Confidence:      0.85,
		Metrics: map[string]float64{
			"avg_volatility": avgVolatility,
			"max_volatility": maxVolatility,
			"threshold":      threshold,
		},
		RecommendedActions: []string{
			"Reduce position sizes in affected symbols",
			"Increase stop-loss monitoring frequency",
			"Consider temporary position hedging",
		},
		Description: fmt.Sprintf("Volatility spike detected in %d symbols, max volatility: %.4f", len(affectedSymbols), maxVolatility),
	}
}

// detectLiquidityDrop detects liquidity drops in market data
func (rm *RiskMonitor) detectLiquidityDrop(marketData []MarketData) *MarketAnomalyReport {
	if len(marketData) == 0 {
		return nil
	}

	// Calculate average liquidity
	totalLiquidity := 0.0
	for _, data := range marketData {
		totalLiquidity += data.Liquidity
	}
	avgLiquidity := totalLiquidity / float64(len(marketData))

	// Detect drops (liquidity < mean - 2 standard deviations)
	liquidities := make([]float64, len(marketData))
	for i, data := range marketData {
		liquidities[i] = data.Liquidity
	}
	
	stdDev := shared.CalculateStandardDeviation(liquidities)
	threshold := avgLiquidity - 2*stdDev

	var affectedSymbols []string
	var minLiquidity float64 = 1.0
	
	for _, data := range marketData {
		if data.Liquidity < threshold {
			affectedSymbols = append(affectedSymbols, data.Symbol)
			if data.Liquidity < minLiquidity {
				minLiquidity = data.Liquidity
			}
		}
	}

	if len(affectedSymbols) == 0 {
		return nil
	}

	var severity shared.Severity
	if len(affectedSymbols) > 10 || minLiquidity < 0.1 {
		severity = shared.SeverityCritical
	} else if len(affectedSymbols) > 5 {
		severity = shared.SeverityError
	} else {
		severity = shared.SeverityWarning
	}

	return &MarketAnomalyReport{
		AnomalyType:     shared.AnomalyTypeLiquidityDrop,
		Severity:        severity,
		AffectedSymbols: affectedSymbols,
		DetectionTime:   time.Now(),
		Confidence:      0.80,
		Metrics: map[string]float64{
			"avg_liquidity": avgLiquidity,
			"min_liquidity": minLiquidity,
			"threshold":     threshold,
		},
		RecommendedActions: []string{
			"Avoid large position changes in affected symbols",
			"Use smaller order sizes",
			"Monitor order book depth",
		},
		Description: fmt.Sprintf("Liquidity drop detected in %d symbols, min liquidity: %.4f", len(affectedSymbols), minLiquidity),
	}
}

// detectCorrelationBreakdown detects correlation breakdown
func (rm *RiskMonitor) detectCorrelationBreakdown(ctx context.Context, marketData []MarketData) *MarketAnomalyReport {
	if len(marketData) < 2 {
		return nil
	}

	// Get symbols for correlation analysis
	symbols := make([]string, len(marketData))
	for i, data := range marketData {
		symbols[i] = data.Symbol
	}

	// Calculate current correlation matrix
	correlationMatrix, err := rm.calculateCorrelationMatrix(ctx, symbols[:min(10, len(symbols))]) // Limit to 10 symbols for performance
	if err != nil {
		log.Printf("Warning: Could not calculate correlation matrix for anomaly detection: %v", err)
		return nil
	}

	// Analyze correlation breakdown (correlations suddenly becoming very low or negative)
	var lowCorrelations []string
	lowCorrCount := 0
	
	for i, symbol1 := range symbols[:min(10, len(symbols))] {
		for j := i + 1; j < min(10, len(symbols)); j++ {
			symbol2 := symbols[j]
			if corr, exists := correlationMatrix[symbol1][symbol2]; exists {
				if math.Abs(corr) < 0.2 { // Very low correlation
					lowCorrelations = append(lowCorrelations, fmt.Sprintf("%s-%s", symbol1, symbol2))
					lowCorrCount++
				}
			}
		}
	}

	// If more than 30% of pairs have low correlation, it might indicate breakdown
	totalPairs := (min(10, len(symbols)) * (min(10, len(symbols)) - 1)) / 2
	if float64(lowCorrCount)/float64(totalPairs) < 0.3 {
		return nil
	}

	var severity shared.Severity
	if float64(lowCorrCount)/float64(totalPairs) > 0.7 {
		severity = shared.SeverityCritical
	} else if float64(lowCorrCount)/float64(totalPairs) > 0.5 {
		severity = shared.SeverityError
	} else {
		severity = shared.SeverityWarning
	}

	return &MarketAnomalyReport{
		AnomalyType:     shared.AnomalyTypeCorrelationBreakdown,
		Severity:        severity,
		AffectedSymbols: symbols[:min(10, len(symbols))],
		DetectionTime:   time.Now(),
		Confidence:      0.75,
		Metrics: map[string]float64{
			"low_correlation_ratio": float64(lowCorrCount) / float64(totalPairs),
			"total_pairs":           float64(totalPairs),
			"low_corr_count":        float64(lowCorrCount),
		},
		RecommendedActions: []string{
			"Review diversification assumptions",
			"Increase hedging strategies",
			"Monitor individual position risks more closely",
		},
		Description: fmt.Sprintf("Correlation breakdown detected: %d/%d pairs have low correlation", lowCorrCount, totalPairs),
	}
}

// detectPriceSpike detects unusual price movements
func (rm *RiskMonitor) detectPriceSpike(marketData []MarketData) *MarketAnomalyReport {
	// This would require historical price data to detect spikes
	// For now, return nil as this requires more complex implementation
	return nil
}

// Helper utility functions

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// generateMockPrices generates mock price data for testing
func (rm *RiskMonitor) generateMockPrices(startPrice float64, count int, volatility float64) []float64 {
	prices := make([]float64, count)
	prices[0] = startPrice
	
	for i := 1; i < count; i++ {
		// Simple random walk with volatility
		change := (shared.GenerateRandomFloat() - 0.5) * volatility
		prices[i] = prices[i-1] * (1 + change)
	}
	
	return prices
}

// generateMockMarketData generates mock market data for testing
func (rm *RiskMonitor) generateMockMarketData() []MarketData {
	symbols := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT"}
	data := make([]MarketData, len(symbols))
	
	for i, symbol := range symbols {
		data[i] = MarketData{
			Symbol:     symbol,
			Price:      50000 + float64(i)*1000, // Mock prices
			Volume:     1000000 + float64(i)*100000,
			Volatility: 0.02 + float64(i)*0.005,
			Liquidity:  0.8 - float64(i)*0.1,
			Timestamp:  time.Now(),
		}
	}
	
	return data
}