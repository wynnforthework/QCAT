package risk

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"qcat/internal/automation/scheduler/shared"
)

// PortfolioExposure represents portfolio exposure information
type PortfolioExposure struct {
	TotalExposure    float64                    `json:"total_exposure"`
	AssetExposures   map[string]float64         `json:"asset_exposures"`
	SectorExposures  map[string]float64         `json:"sector_exposures"`
	RegionExposures  map[string]float64         `json:"region_exposures"`
	LeverageRatio    float64                    `json:"leverage_ratio"`
	ConcentrationRisk float64                   `json:"concentration_risk"`
}

// HedgeRequirement represents a hedge requirement
type HedgeRequirement struct {
	TargetAsset     string  `json:"target_asset"`
	ExposureAmount  float64 `json:"exposure_amount"`
	RequiredHedge   float64 `json:"required_hedge"`
	HedgeInstrument string  `json:"hedge_instrument"`
	Priority        int     `json:"priority"`
}

// ExchangeHealthStatus represents exchange health information
type ExchangeHealthStatus struct {
	Exchange     string    `json:"exchange"`
	HealthScore  float64   `json:"health_score"`
	Latency      float64   `json:"latency"`
	ErrorRate    float64   `json:"error_rate"`
	Uptime       float64   `json:"uptime"`
	LastCheck    time.Time `json:"last_check"`
	Issues       []string  `json:"issues"`
}

// Helper methods for ProtectiveMeasureExecutor

// getPositionsForScaling retrieves positions that should be scaled
func (pme *ProtectiveMeasureExecutor) getPositionsForScaling(ctx context.Context, config *PositionScalingConfig) ([]shared.Position, error) {
	// Build query based on priority
	var query string
	var args []interface{}

	switch config.Priority {
	case "HIGH_RISK":
		query = `
			SELECT p.id, p.symbol, p.side, p.size, p.entry_price, p.current_price,
				   p.unrealized_pnl, p.realized_pnl, p.leverage, p.margin_used, p.created_at
			FROM positions p
			JOIN position_risk pr ON p.id = pr.position_id
			WHERE p.status = 'ACTIVE' 
			AND pr.risk_score > 0.7
			AND p.symbol NOT IN (` + pme.buildExclusionList(config.ExcludedSymbols) + `)
			ORDER BY pr.risk_score DESC
		`
	case "LARGE_POSITIONS":
		query = `
			SELECT p.id, p.symbol, p.side, p.size, p.entry_price, p.current_price,
				   p.unrealized_pnl, p.realized_pnl, p.leverage, p.margin_used, p.created_at
			FROM positions p
			WHERE p.status = 'ACTIVE' 
			AND p.size * p.current_price > $1
			AND p.symbol NOT IN (` + pme.buildExclusionList(config.ExcludedSymbols) + `)
			ORDER BY (p.size * p.current_price) DESC
		`
		args = append(args, config.MinPositionSize*10) // 10x minimum as threshold for "large"
	default: // ALL
		query = `
			SELECT p.id, p.symbol, p.side, p.size, p.entry_price, p.current_price,
				   p.unrealized_pnl, p.realized_pnl, p.leverage, p.margin_used, p.created_at
			FROM positions p
			WHERE p.status = 'ACTIVE' 
			AND p.size * p.current_price > $1
			AND p.symbol NOT IN (` + pme.buildExclusionList(config.ExcludedSymbols) + `)
			ORDER BY p.created_at DESC
		`
		args = append(args, config.MinPositionSize)
	}

	rows, err := pme.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []shared.Position
	for rows.Next() {
		var pos shared.Position
		if err := rows.Scan(
			&pos.ID, &pos.Symbol, &pos.Side, &pos.Size, &pos.EntryPrice,
			&pos.CurrentPrice, &pos.UnrealizedPnL, &pos.RealizedPnL,
			&pos.Leverage, &pos.MarginUsed, &pos.Timestamp,
		); err != nil {
			return nil, err
		}
		positions = append(positions, pos)
	}

	return positions, nil
}

// scalePosition scales a single position
func (pme *ProtectiveMeasureExecutor) scalePosition(ctx context.Context, position shared.Position, config *PositionScalingConfig) (*ScaledPositionInfo, error) {
	// Calculate new position size
	reductionRatio := 1.0 - config.ScalingFactor
	if reductionRatio > config.MaxReduction {
		reductionRatio = config.MaxReduction
	}

	newSize := position.Size * (1.0 - reductionRatio)
	reductionAmount := position.Size - newSize

	// Check if new size meets minimum requirements
	positionValue := newSize * position.CurrentPrice
	if positionValue < config.MinPositionSize {
		log.Printf("Position %s would be below minimum size after scaling, skipping", position.ID)
		return nil, nil
	}

	// Execute position scaling based on method
	executionPrice, err := pme.executePositionReduction(ctx, position, reductionAmount, config.ExecutionMethod)
	if err != nil {
		return nil, err
	}

	// Update position in database
	updateQuery := `
		UPDATE positions 
		SET size = $1, updated_at = NOW()
		WHERE id = $2
	`
	
	_, err = pme.db.ExecContext(ctx, updateQuery, newSize, position.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update position size: %v", err)
	}

	// Log the scaling action
	logQuery := `
		INSERT INTO position_scaling_log (position_id, original_size, new_size, reduction_amount, execution_price, scaling_reason, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`
	
	_, err = pme.db.ExecContext(ctx, logQuery, position.ID, position.Size, newSize, reductionAmount, executionPrice, "MARKET_STRESS_SCALING")
	if err != nil {
		log.Printf("Warning: Failed to log position scaling: %v", err)
	}

	return &ScaledPositionInfo{
		PositionID:      position.ID,
		Symbol:          position.Symbol,
		OriginalSize:    position.Size,
		NewSize:         newSize,
		ReductionAmount: reductionAmount,
		ReductionRatio:  reductionRatio,
		ExecutionPrice:  executionPrice,
		Status:          "COMPLETED",
	}, nil
}

// executePositionReduction executes the actual position reduction
func (pme *ProtectiveMeasureExecutor) executePositionReduction(ctx context.Context, position shared.Position, reductionAmount float64, method string) (float64, error) {
	switch method {
	case "IMMEDIATE":
		return pme.executeImmediateReduction(ctx, position, reductionAmount)
	case "GRADUAL":
		return pme.executeGradualReduction(ctx, position, reductionAmount)
	case "TWAP":
		return pme.executeTWAPReduction(ctx, position, reductionAmount)
	default:
		return pme.executeImmediateReduction(ctx, position, reductionAmount)
	}
}

// executeImmediateReduction executes immediate position reduction
func (pme *ProtectiveMeasureExecutor) executeImmediateReduction(ctx context.Context, position shared.Position, reductionAmount float64) (float64, error) {
	// For now, simulate execution by returning current price
	// In real implementation, this would place market orders
	log.Printf("Executing immediate reduction of %.4f for position %s at market price", reductionAmount, position.ID)
	
	// Simulate some slippage for market orders
	slippage := 0.001 // 0.1% slippage
	var executionPrice float64
	
	if position.Side == "LONG" {
		executionPrice = position.CurrentPrice * (1.0 - slippage)
	} else {
		executionPrice = position.CurrentPrice * (1.0 + slippage)
	}
	
	return executionPrice, nil
}

// executeGradualReduction executes gradual position reduction
func (pme *ProtectiveMeasureExecutor) executeGradualReduction(ctx context.Context, position shared.Position, reductionAmount float64) (float64, error) {
	// For gradual reduction, we'd split into smaller orders over time
	log.Printf("Executing gradual reduction of %.4f for position %s", reductionAmount, position.ID)
	
	// Simulate better execution price due to gradual approach
	slippage := 0.0005 // 0.05% slippage
	var executionPrice float64
	
	if position.Side == "LONG" {
		executionPrice = position.CurrentPrice * (1.0 - slippage)
	} else {
		executionPrice = position.CurrentPrice * (1.0 + slippage)
	}
	
	return executionPrice, nil
}

// executeTWAPReduction executes TWAP (Time-Weighted Average Price) reduction
func (pme *ProtectiveMeasureExecutor) executeTWAPReduction(ctx context.Context, position shared.Position, reductionAmount float64) (float64, error) {
	// TWAP execution would spread orders over time
	log.Printf("Executing TWAP reduction of %.4f for position %s", reductionAmount, position.ID)
	
	// Simulate best execution price due to TWAP
	slippage := 0.0002 // 0.02% slippage
	var executionPrice float64
	
	if position.Side == "LONG" {
		executionPrice = position.CurrentPrice * (1.0 - slippage)
	} else {
		executionPrice = position.CurrentPrice * (1.0 + slippage)
	}
	
	return executionPrice, nil
}

// getPortfolioExposure calculates current portfolio exposure
func (pme *ProtectiveMeasureExecutor) getPortfolioExposure(ctx context.Context) (*PortfolioExposure, error) {
	query := `
		SELECT 
			p.symbol,
			SUM(p.size * p.current_price) as exposure,
			AVG(p.leverage) as avg_leverage
		FROM positions p
		WHERE p.status = 'ACTIVE'
		GROUP BY p.symbol
	`
	
	rows, err := pme.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	assetExposures := make(map[string]float64)
	var totalExposure float64
	var totalLeverage float64
	var assetCount int

	for rows.Next() {
		var symbol string
		var exposure, leverage float64
		if err := rows.Scan(&symbol, &exposure, &leverage); err != nil {
			return nil, err
		}
		
		assetExposures[symbol] = exposure
		totalExposure += exposure
		totalLeverage += leverage
		assetCount++
	}

	// Calculate concentration risk (Herfindahl index)
	var concentrationRisk float64
	for _, exposure := range assetExposures {
		if totalExposure > 0 {
			weight := exposure / totalExposure
			concentrationRisk += weight * weight
		}
	}

	var avgLeverage float64
	if assetCount > 0 {
		avgLeverage = totalLeverage / float64(assetCount)
	}

	return &PortfolioExposure{
		TotalExposure:     totalExposure,
		AssetExposures:    assetExposures,
		SectorExposures:   make(map[string]float64), // Would be populated with sector mapping
		RegionExposures:   make(map[string]float64), // Would be populated with region mapping
		LeverageRatio:     avgLeverage,
		ConcentrationRisk: concentrationRisk,
	}, nil
}

// calculateHedgeRequirements calculates hedge requirements based on exposure
func (pme *ProtectiveMeasureExecutor) calculateHedgeRequirements(exposure *PortfolioExposure, config *EmergencyHedgingConfig) []HedgeRequirement {
	var requirements []HedgeRequirement
	
	// Calculate hedge requirements for each asset
	for asset, assetExposure := range exposure.AssetExposures {
		if assetExposure < config.MaxHedgeSize * 0.1 { // Skip small exposures
			continue
		}
		
		requiredHedge := assetExposure * config.HedgeRatio
		if requiredHedge > config.MaxHedgeSize {
			requiredHedge = config.MaxHedgeSize
		}
		
		// Find appropriate hedge instrument
		hedgeInstrument := pme.findHedgeInstrument(asset, config.HedgeInstruments)
		if hedgeInstrument == "" {
			continue
		}
		
		// Calculate priority based on exposure size and correlation
		priority := pme.calculateHedgePriority(asset, assetExposure, exposure.TotalExposure, config.Correlations)
		
		requirements = append(requirements, HedgeRequirement{
			TargetAsset:     asset,
			ExposureAmount:  assetExposure,
			RequiredHedge:   requiredHedge,
			HedgeInstrument: hedgeInstrument,
			Priority:        priority,
		})
	}
	
	return requirements
}

// executeHedge executes a hedge position
func (pme *ProtectiveMeasureExecutor) executeHedge(ctx context.Context, requirement HedgeRequirement, config *EmergencyHedgingConfig) (*HedgePositionInfo, error) {
	log.Printf("Executing hedge for %s: %.2f using %s", requirement.TargetAsset, requirement.RequiredHedge, requirement.HedgeInstrument)
	
	// Calculate hedge parameters
	hedgeRatio := requirement.RequiredHedge / requirement.ExposureAmount
	
	// Get correlation for expected offset calculation
	correlation := config.Correlations[requirement.TargetAsset]
	if correlation == 0 {
		correlation = -0.8 // Default negative correlation for hedge
	}
	
	expectedOffset := requirement.RequiredHedge * math.Abs(correlation)
	
	// Execute hedge (simulated)
	hedgePrice := pme.getHedgeInstrumentPrice(ctx, requirement.HedgeInstrument)
	hedgeSize := requirement.RequiredHedge / hedgePrice
	
	// Record hedge position
	hedgeID := fmt.Sprintf("HEDGE_%s_%d", requirement.TargetAsset, time.Now().Unix())
	
	insertQuery := `
		INSERT INTO hedge_positions (hedge_id, instrument, target_asset, hedge_size, hedge_price, hedge_ratio, expected_offset, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'ACTIVE', NOW())
	`
	
	_, err := pme.db.ExecContext(ctx, insertQuery, hedgeID, requirement.HedgeInstrument, requirement.TargetAsset, hedgeSize, hedgePrice, hedgeRatio, expectedOffset)
	if err != nil {
		return nil, fmt.Errorf("failed to record hedge position: %v", err)
	}
	
	return &HedgePositionInfo{
		HedgeID:         hedgeID,
		Instrument:      requirement.HedgeInstrument,
		HedgeSize:       hedgeSize,
		HedgePrice:      hedgePrice,
		TargetAsset:     requirement.TargetAsset,
		HedgeRatio:      hedgeRatio,
		ExpectedOffset:  expectedOffset,
		Status:          "ACTIVE",
	}, nil
}

// getCurrentFundingRates retrieves current funding rates
func (pme *ProtectiveMeasureExecutor) getCurrentFundingRates(ctx context.Context, pairs []string) (map[string]float64, error) {
	if len(pairs) == 0 {
		// Get all active pairs if none specified
		query := `SELECT DISTINCT symbol FROM positions WHERE status = 'ACTIVE'`
		rows, err := pme.db.QueryContext(ctx, query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		
		for rows.Next() {
			var symbol string
			if err := rows.Scan(&symbol); err != nil {
				continue
			}
			pairs = append(pairs, symbol)
		}
	}
	
	fundingRates := make(map[string]float64)
	
	// Get funding rates from database or exchange API
	for _, pair := range pairs {
		query := `
			SELECT funding_rate 
			FROM funding_rates 
			WHERE symbol = $1 
			ORDER BY timestamp DESC 
			LIMIT 1
		`
		
		var fundingRate float64
		err := pme.db.QueryRowContext(ctx, query, pair).Scan(&fundingRate)
		if err != nil {
			log.Printf("Warning: Could not get funding rate for %s: %v", pair, err)
			continue
		}
		
		fundingRates[pair] = fundingRate
	}
	
	return fundingRates, nil
}

// evaluateFundingRateAdjustment evaluates if position adjustment is needed based on funding rate
func (pme *ProtectiveMeasureExecutor) evaluateFundingRateAdjustment(ctx context.Context, symbol string, fundingRate float64, config *FundingRateConfig) (*FundingRateAdjustment, error) {
	// Check if funding rate exceeds thresholds
	var thresholdType string
	var adjustmentNeeded bool
	
	if fundingRate > config.PositiveThreshold {
		thresholdType = "POSITIVE"
		adjustmentNeeded = true
	} else if fundingRate < config.NegativeThreshold {
		thresholdType = "NEGATIVE"
		adjustmentNeeded = true
	}
	
	if !adjustmentNeeded {
		return nil, nil
	}
	
	// Get current position for this symbol
	query := `
		SELECT id, size, side
		FROM positions 
		WHERE symbol = $1 AND status = 'ACTIVE'
		LIMIT 1
	`
	
	var positionID string
	var originalSize float64
	var side string
	
	err := pme.db.QueryRowContext(ctx, query, symbol).Scan(&positionID, &originalSize, &side)
	if err != nil {
		return nil, nil // No position for this symbol
	}
	
	// Calculate adjustment
	var adjustmentRatio float64
	var rationale string
	
	if thresholdType == "POSITIVE" && side == "LONG" {
		// High positive funding rate hurts long positions
		adjustmentRatio = 1.0 - config.AdjustmentFactor
		rationale = "Reducing long position due to high positive funding rate"
	} else if thresholdType == "NEGATIVE" && side == "SHORT" {
		// High negative funding rate hurts short positions
		adjustmentRatio = 1.0 - config.AdjustmentFactor
		rationale = "Reducing short position due to high negative funding rate"
	} else {
		// Funding rate favors current position, no adjustment needed
		return nil, nil
	}
	
	adjustedSize := originalSize * adjustmentRatio
	
	// Execute the adjustment
	updateQuery := `
		UPDATE positions 
		SET size = $1, updated_at = NOW()
		WHERE id = $2
	`
	
	_, err = pme.db.ExecContext(ctx, updateQuery, adjustedSize, positionID)
	if err != nil {
		return nil, fmt.Errorf("failed to adjust position size: %v", err)
	}
	
	return &FundingRateAdjustment{
		Symbol:          symbol,
		CurrentFunding:  fundingRate,
		ThresholdType:   thresholdType,
		OriginalSize:    originalSize,
		AdjustedSize:    adjustedSize,
		AdjustmentRatio: adjustmentRatio,
		Rationale:       rationale,
	}, nil
}

// getExchangeHealthStatus retrieves exchange health status
func (pme *ProtectiveMeasureExecutor) getExchangeHealthStatus(ctx context.Context) (map[string]*ExchangeHealthStatus, error) {
	query := `
		SELECT 
			exchange,
			health_score,
			latency_ms,
			error_rate,
			uptime_percentage,
			last_check,
			issues
		FROM exchange_health 
		WHERE last_check > NOW() - INTERVAL '5 minutes'
	`
	
	rows, err := pme.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	status := make(map[string]*ExchangeHealthStatus)
	
	for rows.Next() {
		var exchange, issues string
		var healthScore, latency, errorRate, uptime float64
		var lastCheck time.Time
		
		if err := rows.Scan(&exchange, &healthScore, &latency, &errorRate, &uptime, &lastCheck, &issues); err != nil {
			continue
		}
		
		var issueList []string
		if issues != "" {
			// Parse issues (assuming comma-separated)
			// In real implementation, this might be JSON
			issueList = []string{issues}
		}
		
		status[exchange] = &ExchangeHealthStatus{
			Exchange:    exchange,
			HealthScore: healthScore,
			Latency:     latency,
			ErrorRate:   errorRate,
			Uptime:      uptime,
			LastCheck:   lastCheck,
			Issues:      issueList,
		}
	}
	
	return status, nil
}

// activateExchangeProtectiveMeasures activates protective measures for a specific exchange
func (pme *ProtectiveMeasureExecutor) activateExchangeProtectiveMeasures(ctx context.Context, exchange string, status *ExchangeHealthStatus) error {
	log.Printf("Activating protective measures for exchange %s (health: %.2f)", exchange, status.HealthScore)
	
	// Reduce order sizes for this exchange
	updateQuery1 := `
		UPDATE system_settings 
		SET value = CAST((CAST(value AS FLOAT) * 0.5) AS TEXT)
		WHERE key = $1
	`
	
	_, err := pme.db.ExecContext(ctx, updateQuery1, fmt.Sprintf("max_order_size_%s", exchange))
	if err != nil {
		log.Printf("Warning: Failed to reduce order sizes for %s: %v", exchange, err)
	}
	
	// Increase monitoring frequency for this exchange
	updateQuery2 := `
		UPDATE system_settings 
		SET value = '15'
		WHERE key = $1
	`
	
	_, err = pme.db.ExecContext(ctx, updateQuery2, fmt.Sprintf("monitoring_interval_%s", exchange))
	if err != nil {
		log.Printf("Warning: Failed to increase monitoring for %s: %v", exchange, err)
	}
	
	// Log protective measure activation
	logQuery := `
		INSERT INTO protective_measures_log (exchange, measure_type, health_score, reason, created_at)
		VALUES ($1, 'EXCHANGE_DEGRADATION', $2, $3, NOW())
	`
	
	reason := fmt.Sprintf("Health score %.2f, Issues: %v", status.HealthScore, status.Issues)
	_, err = pme.db.ExecContext(ctx, logQuery, exchange, status.HealthScore, reason)
	if err != nil {
		log.Printf("Warning: Failed to log protective measures: %v", err)
	}
	
	return nil
}

// Helper utility methods

// buildExclusionList builds SQL IN clause for excluded symbols
func (pme *ProtectiveMeasureExecutor) buildExclusionList(excludedSymbols []string) string {
	if len(excludedSymbols) == 0 {
		return "''"
	}
	
	result := "'"
	for i, symbol := range excludedSymbols {
		if i > 0 {
			result += "','"
		}
		result += symbol
	}
	result += "'"
	return result
}

// findHedgeInstrument finds appropriate hedge instrument for an asset
func (pme *ProtectiveMeasureExecutor) findHedgeInstrument(asset string, availableInstruments []string) string {
	// Simple mapping logic - in real implementation this would be more sophisticated
	for _, instrument := range availableInstruments {
		if instrument == asset+"_PERP" || instrument == asset+"_FUTURE" {
			return instrument
		}
	}
	
	// Default hedge instruments
	if len(availableInstruments) > 0 {
		return availableInstruments[0]
	}
	
	return ""
}

// calculateHedgePriority calculates hedge priority based on various factors
func (pme *ProtectiveMeasureExecutor) calculateHedgePriority(asset string, exposure, totalExposure float64, correlations map[string]float64) int {
	// Higher exposure = higher priority
	exposureWeight := exposure / totalExposure
	
	// Higher correlation = higher priority for hedging
	correlation := correlations[asset]
	if correlation == 0 {
		correlation = 0.5 // Default
	}
	
	priority := int((exposureWeight + math.Abs(correlation)) * 100)
	
	if priority > 100 {
		priority = 100
	}
	
	return priority
}

// getHedgeInstrumentPrice gets the current price of a hedge instrument
func (pme *ProtectiveMeasureExecutor) getHedgeInstrumentPrice(ctx context.Context, instrument string) float64 {
	query := `
		SELECT price 
		FROM market_data 
		WHERE symbol = $1 
		ORDER BY timestamp DESC 
		LIMIT 1
	`
	
	var price float64
	err := pme.db.QueryRowContext(ctx, query, instrument).Scan(&price)
	if err != nil {
		log.Printf("Warning: Could not get price for %s, using default: %v", instrument, err)
		return 100.0 // Default price
	}
	
	return price
}

// Update metrics methods

// updatePositionScalingMetrics updates position scaling metrics
func (pme *ProtectiveMeasureExecutor) updatePositionScalingMetrics(result *PositionScalingResult) {
	pme.metrics["last_position_scaling"] = result.Timestamp
	pme.metrics["positions_scaled"] = result.TotalPositionsScaled
	pme.metrics["total_reduction"] = result.TotalReduction
	pme.metrics["scaling_execution_time"] = result.ExecutionTime
	pme.metrics["scaling_errors"] = len(result.Errors)
}

// updateEmergencyHedgingMetrics updates emergency hedging metrics
func (pme *ProtectiveMeasureExecutor) updateEmergencyHedgingMetrics(result *EmergencyHedgingResult) {
	pme.metrics["last_emergency_hedging"] = result.Timestamp
	pme.metrics["hedges_activated"] = result.HedgesActivated
	pme.metrics["total_hedge_size"] = result.TotalHedgeSize
	pme.metrics["effective_hedge_ratio"] = result.EffectiveHedgeRatio
	pme.metrics["hedging_execution_time"] = result.ExecutionTime
	pme.metrics["hedging_errors"] = len(result.Errors)
}

// updateFundingRateMetrics updates funding rate metrics
func (pme *ProtectiveMeasureExecutor) updateFundingRateMetrics(result *FundingRateAdjustmentResult) {
	pme.metrics["last_funding_adjustment"] = result.Timestamp
	pme.metrics["positions_adjusted"] = result.AdjustedPositions
	pme.metrics["total_adjustment"] = result.TotalAdjustment
	pme.metrics["funding_execution_time"] = result.ExecutionTime
	pme.metrics["funding_errors"] = len(result.Errors)
}

// updateExchangeIntegrationMetrics updates exchange integration metrics
func (pme *ProtectiveMeasureExecutor) updateExchangeIntegrationMetrics(status map[string]*ExchangeHealthStatus) {
	pme.metrics["last_exchange_check"] = time.Now()
	pme.metrics["monitored_exchanges"] = len(status)
	
	var totalHealth float64
	var unhealthyCount int
	
	for _, exchangeStatus := range status {
		totalHealth += exchangeStatus.HealthScore
		if exchangeStatus.HealthScore < 0.7 {
			unhealthyCount++
		}
	}
	
	if len(status) > 0 {
		pme.metrics["average_exchange_health"] = totalHealth / float64(len(status))
	}
	pme.metrics["unhealthy_exchanges"] = unhealthyCount
}