package risk

import (
	"context"
	"database/sql"
	"log"
	"math"
	"time"

	"qcat/internal/automation/scheduler/shared"
)

// storeAdjustmentPerformance stores an adjustment performance record in the database
func (slpt *StopLossPerformanceTracker) storeAdjustmentPerformance(ctx context.Context, performance AdjustmentPerformance) error {
	query := `
		INSERT INTO stop_loss_performance (
			adjustment_id, position_id, symbol, adjustment_time, old_stop_loss, new_stop_loss,
			price_at_adjustment, adjustment_type, was_triggered, effectiveness_score, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`
	
	_, err := slpt.db.ExecContext(ctx, query,
		performance.AdjustmentID, performance.PositionID, performance.Symbol,
		performance.AdjustmentTime, performance.OldStopLoss, performance.NewStopLoss,
		performance.PriceAtAdjustment, performance.AdjustmentType,
		performance.WasTriggered, performance.EffectivenessScore)
	
	return err
}

// updateAdjustmentPerformance updates an adjustment performance record
func (slpt *StopLossPerformanceTracker) updateAdjustmentPerformance(ctx context.Context, performance AdjustmentPerformance) error {
	query := `
		UPDATE stop_loss_performance 
		SET was_triggered = ?, trigger_time = ?, trigger_price = ?, pnl_at_trigger = ?,
			effectiveness_score = ?, would_old_have_been_better = ?, pnl_difference = ?,
			time_to_trigger = ?, updated_at = CURRENT_TIMESTAMP
		WHERE adjustment_id = ?
	`
	
	var triggerTime interface{}
	var timeToTrigger interface{}
	
	if performance.TriggerTime != nil {
		triggerTime = *performance.TriggerTime
	}
	
	if performance.TimeToTrigger != nil {
		timeToTrigger = performance.TimeToTrigger.Seconds()
	}
	
	_, err := slpt.db.ExecContext(ctx, query,
		performance.WasTriggered, triggerTime, performance.TriggerPrice,
		performance.PnLAtTrigger, performance.EffectivenessScore,
		performance.WouldOldHaveBeenBetter, performance.PnLDifference,
		timeToTrigger, performance.AdjustmentID)
	
	return err
}

// getActiveTrackingRecords gets all active performance tracking records
func (slpt *StopLossPerformanceTracker) getActiveTrackingRecords(ctx context.Context) ([]AdjustmentPerformance, error) {
	query := `
		SELECT 
			adjustment_id, position_id, symbol, adjustment_time, old_stop_loss, new_stop_loss,
			price_at_adjustment, adjustment_type, was_triggered, COALESCE(trigger_time, ''),
			trigger_price, pnl_at_trigger, effectiveness_score, would_old_have_been_better,
			pnl_difference, COALESCE(time_to_trigger, 0)
		FROM stop_loss_performance 
		WHERE was_triggered = FALSE OR trigger_time IS NULL
		ORDER BY adjustment_time DESC
		LIMIT 100
	`
	
	rows, err := slpt.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []AdjustmentPerformance
	for rows.Next() {
		var record AdjustmentPerformance
		var triggerTimeStr string
		var timeToTriggerSeconds float64
		
		err := rows.Scan(
			&record.AdjustmentID, &record.PositionID, &record.Symbol,
			&record.AdjustmentTime, &record.OldStopLoss, &record.NewStopLoss,
			&record.PriceAtAdjustment, &record.AdjustmentType, &record.WasTriggered,
			&triggerTimeStr, &record.TriggerPrice, &record.PnLAtTrigger,
			&record.EffectivenessScore, &record.WouldOldHaveBeenBetter,
			&record.PnLDifference, &timeToTriggerSeconds,
		)
		if err != nil {
			return nil, err
		}
		
		// Parse trigger time if present
		if triggerTimeStr != "" {
			triggerTime, err := time.Parse(time.RFC3339, triggerTimeStr)
			if err == nil {
				record.TriggerTime = &triggerTime
			}
		}
		
		// Parse time to trigger if present
		if timeToTriggerSeconds > 0 {
			duration := time.Duration(timeToTriggerSeconds * float64(time.Second))
			record.TimeToTrigger = &duration
		}
		
		records = append(records, record)
	}

	return records, nil
}

// wasStopLossTriggered determines if a stop loss was triggered based on closure details
func (slpt *StopLossPerformanceTracker) wasStopLossTriggered(closure *PositionClosureDetails, stopLossLevel float64) bool {
	// Check if closure reason indicates stop loss trigger
	if closure.CloseReason == "STOP_LOSS" || closure.CloseReason == "STOP_LOSS_TRIGGERED" {
		return true
	}
	
	// Check if close price is near stop loss level (within 0.1% tolerance)
	tolerance := 0.001 // 0.1%
	priceDiff := math.Abs(closure.ClosePrice - stopLossLevel)
	priceThreshold := stopLossLevel * tolerance
	
	return priceDiff <= priceThreshold
}

// calculateEffectivenessScore calculates the effectiveness score of a stop loss adjustment
func (slpt *StopLossPerformanceTracker) calculateEffectivenessScore(tracking AdjustmentPerformance, closure *PositionClosureDetails, wasTriggered bool) float64 {
	baseScore := 0.5 // Neutral score
	
	if !wasTriggered {
		// Stop loss wasn't triggered - this is generally good
		baseScore = 0.7
		
		// Bonus if position was profitable
		if closure.RealizedPnL > 0 {
			baseScore += 0.2
		}
		
		// Penalty if position had large loss (stop loss should have been tighter)
		if closure.RealizedPnL < -1000 { // Adjust threshold as needed
			lossRatio := math.Abs(closure.RealizedPnL) / tracking.PriceAtAdjustment
			penalty := math.Min(0.3, lossRatio*0.1)
			baseScore -= penalty
		}
	} else {
		// Stop loss was triggered
		baseScore = 0.4 // Slightly negative as it indicates loss
		
		// Check if the trigger prevented larger losses
		wouldHaveBeenWorse := slpt.wouldOldStopLossHaveBeenWorse(tracking, closure)
		if wouldHaveBeenWorse {
			baseScore += 0.3 // Good adjustment that prevented larger loss
		}
		
		// Consider time to trigger (faster trigger might be better in volatile markets)
		if tracking.TimeToTrigger != nil {
			if *tracking.TimeToTrigger < time.Hour {
				baseScore += 0.1 // Quick trigger in volatile conditions
			}
		}
	}
	
	// Ensure score is within [0, 1] range
	if baseScore > 1.0 {
		baseScore = 1.0
	} else if baseScore < 0.0 {
		baseScore = 0.0
	}
	
	return baseScore
}

// wouldOldStopLossHaveBeenBetter determines if the old stop loss would have been better
func (slpt *StopLossPerformanceTracker) wouldOldStopLossHaveBeenBetter(tracking AdjustmentPerformance, closure *PositionClosureDetails) bool {
	// Simulate what would have happened with old stop loss
	oldWouldHaveTriggered := slpt.wouldStopLossHaveTriggered(tracking.OldStopLoss, closure)
	newWasTriggered := slpt.wasStopLossTriggered(closure, tracking.NewStopLoss)
	
	// If both would trigger or neither would trigger, compare which is closer to optimal
	if oldWouldHaveTriggered == newWasTriggered {
		// Compare distances from entry price
		entryPrice := tracking.PriceAtAdjustment // Using price at adjustment as proxy
		oldDistance := math.Abs(tracking.OldStopLoss - entryPrice)
		newDistance := math.Abs(tracking.NewStopLoss - entryPrice)
		
		// In general, closer stop loss is better for risk management
		return oldDistance < newDistance
	}
	
	// If only one would trigger, the one that doesn't trigger is generally better
	// (assuming the position wasn't closed at a worse price)
	if !oldWouldHaveTriggered && newWasTriggered {
		return true // Old stop loss would have been better (wouldn't have triggered)
	}
	
	return false // New stop loss was better
}

// wouldOldStopLossHaveBeenWorse determines if old stop loss would have resulted in worse outcome
func (slpt *StopLossPerformanceTracker) wouldOldStopLossHaveBeenWorse(tracking AdjustmentPerformance, closure *PositionClosureDetails) bool {
	return !slpt.wouldOldStopLossHaveBeenBetter(tracking, closure)
}

// wouldStopLossHaveTriggered checks if a stop loss level would have been triggered
func (slpt *StopLossPerformanceTracker) wouldStopLossHaveTriggered(stopLossLevel float64, closure *PositionClosureDetails) bool {
	// This is a simplified check - in reality, you'd need to check historical price data
	// to see if the stop loss level was hit before the actual closure
	tolerance := 0.001 // 0.1%
	priceDiff := math.Abs(closure.ClosePrice - stopLossLevel)
	priceThreshold := stopLossLevel * tolerance
	
	return priceDiff <= priceThreshold
}

// calculatePnLDifference calculates the PnL difference between old and new stop loss
func (slpt *StopLossPerformanceTracker) calculatePnLDifference(tracking AdjustmentPerformance, closure *PositionClosureDetails) float64 {
	// Estimate what PnL would have been with old stop loss
	oldPnL := slpt.estimatePnLWithStopLoss(tracking.OldStopLoss, tracking, closure)
	actualPnL := closure.RealizedPnL
	
	return actualPnL - oldPnL
}

// estimatePnLWithStopLoss estimates PnL if a different stop loss had been used
func (slpt *StopLossPerformanceTracker) estimatePnLWithStopLoss(stopLossLevel float64, tracking AdjustmentPerformance, closure *PositionClosureDetails) float64 {
	// Simplified estimation - in reality, this would require more complex analysis
	// of price movements and when the stop loss would have been triggered
	
	if slpt.wouldStopLossHaveTriggered(stopLossLevel, closure) {
		// Estimate PnL at stop loss trigger
		priceDiff := stopLossLevel - tracking.PriceAtAdjustment
		return priceDiff // Simplified calculation
	}
	
	// If stop loss wouldn't have triggered, return actual PnL
	return closure.RealizedPnL
}

// calculateCurrentEffectiveness calculates current effectiveness for active positions
func (slpt *StopLossPerformanceTracker) calculateCurrentEffectiveness(tracking AdjustmentPerformance, position *shared.Position) float64 {
	// For active positions, calculate unrealized effectiveness
	baseScore := 0.5
	
	// Check current unrealized PnL
	if position.UnrealizedPnL > 0 {
		baseScore += 0.2 // Position is profitable
	} else if position.UnrealizedPnL < -500 { // Adjust threshold as needed
		// Position has significant unrealized loss
		lossRatio := math.Abs(position.UnrealizedPnL) / position.CurrentPrice
		penalty := math.Min(0.3, lossRatio*0.1)
		baseScore -= penalty
	}
	
	// Check how close current price is to stop loss
	stopLossDistance := math.Abs(position.CurrentPrice - tracking.NewStopLoss)
	priceRatio := stopLossDistance / position.CurrentPrice
	
	if priceRatio < 0.02 { // Very close to stop loss (within 2%)
		baseScore -= 0.1 // Slightly negative as it's risky
	} else if priceRatio > 0.1 { // Far from stop loss (more than 10%)
		baseScore += 0.1 // Positive as there's room for profit
	}
	
	// Ensure score is within [0, 1] range
	if baseScore > 1.0 {
		baseScore = 1.0
	} else if baseScore < 0.0 {
		baseScore = 0.0
	}
	
	return baseScore
}

// calculateAggregateMetrics calculates aggregate performance metrics
func (slpt *StopLossPerformanceTracker) calculateAggregateMetrics(ctx context.Context) error {
	// Get performance statistics from database
	stats, err := slpt.getPerformanceStatistics(ctx)
	if err != nil {
		return err
	}
	
	// Update metrics
	slpt.mu.Lock()
	defer slpt.mu.Unlock()
	
	slpt.metrics["total_adjustments"] = stats.TotalAdjustments
	slpt.metrics["triggered_adjustments"] = stats.TriggeredAdjustments
	slpt.metrics["trigger_rate"] = stats.TriggerRate
	slpt.metrics["average_effectiveness"] = stats.AverageEffectiveness
	slpt.metrics["successful_adjustments"] = stats.SuccessfulAdjustments
	slpt.metrics["success_rate"] = stats.SuccessRate
	slpt.metrics["average_time_to_trigger"] = stats.AverageTimeToTrigger
	slpt.metrics["total_pnl_impact"] = stats.TotalPnLImpact
	slpt.metrics["last_metrics_update"] = time.Now()
	
	log.Printf("Updated aggregate performance metrics: %d total adjustments, %.2f%% trigger rate, %.2f average effectiveness",
		stats.TotalAdjustments, stats.TriggerRate*100, stats.AverageEffectiveness)
	
	return nil
}

// PerformanceStatistics represents aggregate performance statistics
type PerformanceStatistics struct {
	TotalAdjustments      int           `json:"total_adjustments"`
	TriggeredAdjustments  int           `json:"triggered_adjustments"`
	TriggerRate           float64       `json:"trigger_rate"`
	AverageEffectiveness  float64       `json:"average_effectiveness"`
	SuccessfulAdjustments int           `json:"successful_adjustments"`
	SuccessRate           float64       `json:"success_rate"`
	AverageTimeToTrigger  time.Duration `json:"average_time_to_trigger"`
	TotalPnLImpact        float64       `json:"total_pnl_impact"`
}

// getPerformanceStatistics gets aggregate performance statistics from database
func (slpt *StopLossPerformanceTracker) getPerformanceStatistics(ctx context.Context) (*PerformanceStatistics, error) {
	query := `
		SELECT 
			COUNT(*) as total_adjustments,
			SUM(CASE WHEN was_triggered THEN 1 ELSE 0 END) as triggered_adjustments,
			AVG(effectiveness_score) as avg_effectiveness,
			SUM(CASE WHEN effectiveness_score > 0.6 THEN 1 ELSE 0 END) as successful_adjustments,
			AVG(CASE WHEN time_to_trigger > 0 THEN time_to_trigger ELSE NULL END) as avg_time_to_trigger,
			SUM(COALESCE(pnl_difference, 0)) as total_pnl_impact
		FROM stop_loss_performance 
		WHERE adjustment_time >= DATE_SUB(NOW(), INTERVAL 30 DAY)
	`
	
	var stats PerformanceStatistics
	var avgTimeToTriggerSeconds sql.NullFloat64
	
	err := slpt.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalAdjustments,
		&stats.TriggeredAdjustments,
		&stats.AverageEffectiveness,
		&stats.SuccessfulAdjustments,
		&avgTimeToTriggerSeconds,
		&stats.TotalPnLImpact,
	)
	
	if err != nil {
		return nil, err
	}
	
	// Calculate derived metrics
	if stats.TotalAdjustments > 0 {
		stats.TriggerRate = float64(stats.TriggeredAdjustments) / float64(stats.TotalAdjustments)
		stats.SuccessRate = float64(stats.SuccessfulAdjustments) / float64(stats.TotalAdjustments)
	}
	
	if avgTimeToTriggerSeconds.Valid {
		stats.AverageTimeToTrigger = time.Duration(avgTimeToTriggerSeconds.Float64 * float64(time.Second))
	}
	
	return &stats, nil
}

// GetPerformanceReport generates a comprehensive performance report
func (slpt *StopLossPerformanceTracker) GetPerformanceReport(ctx context.Context, days int) (*PerformanceReport, error) {
	slpt.mu.RLock()
	defer slpt.mu.RUnlock()
	
	// Get statistics for the specified period
	stats, err := slpt.getPerformanceStatisticsForPeriod(ctx, days)
	if err != nil {
		return nil, err
	}
	
	// Get top performing adjustments
	topPerforming, err := slpt.getTopPerformingAdjustments(ctx, days, 10)
	if err != nil {
		return nil, err
	}
	
	// Get worst performing adjustments
	worstPerforming, err := slpt.getWorstPerformingAdjustments(ctx, days, 10)
	if err != nil {
		return nil, err
	}
	
	// Get performance by adjustment type
	performanceByType, err := slpt.getPerformanceByAdjustmentType(ctx, days)
	if err != nil {
		return nil, err
	}
	
	report := &PerformanceReport{
		Period:              days,
		Statistics:          *stats,
		TopPerforming:       topPerforming,
		WorstPerforming:     worstPerforming,
		PerformanceByType:   performanceByType,
		GeneratedAt:         time.Now(),
	}
	
	return report, nil
}

// PerformanceReport represents a comprehensive performance report
type PerformanceReport struct {
	Period              int                              `json:"period_days"`
	Statistics          PerformanceStatistics            `json:"statistics"`
	TopPerforming       []AdjustmentPerformance          `json:"top_performing"`
	WorstPerforming     []AdjustmentPerformance          `json:"worst_performing"`
	PerformanceByType   map[string]PerformanceStatistics `json:"performance_by_type"`
	GeneratedAt         time.Time                        `json:"generated_at"`
}

// Additional helper methods for performance reporting would be implemented here...
// (getPerformanceStatisticsForPeriod, getTopPerformingAdjustments, etc.)

// getPerformanceStatisticsForPeriod gets performance statistics for a specific period
func (slpt *StopLossPerformanceTracker) getPerformanceStatisticsForPeriod(ctx context.Context, days int) (*PerformanceStatistics, error) {
	query := `
		SELECT 
			COUNT(*) as total_adjustments,
			SUM(CASE WHEN was_triggered THEN 1 ELSE 0 END) as triggered_adjustments,
			AVG(effectiveness_score) as avg_effectiveness,
			SUM(CASE WHEN effectiveness_score > 0.6 THEN 1 ELSE 0 END) as successful_adjustments,
			AVG(CASE WHEN time_to_trigger > 0 THEN time_to_trigger ELSE NULL END) as avg_time_to_trigger,
			SUM(COALESCE(pnl_difference, 0)) as total_pnl_impact
		FROM stop_loss_performance 
		WHERE adjustment_time >= DATE_SUB(NOW(), INTERVAL ? DAY)
	`
	
	var stats PerformanceStatistics
	var avgTimeToTriggerSeconds sql.NullFloat64
	
	err := slpt.db.QueryRowContext(ctx, query, days).Scan(
		&stats.TotalAdjustments,
		&stats.TriggeredAdjustments,
		&stats.AverageEffectiveness,
		&stats.SuccessfulAdjustments,
		&avgTimeToTriggerSeconds,
		&stats.TotalPnLImpact,
	)
	
	if err != nil {
		return nil, err
	}
	
	// Calculate derived metrics
	if stats.TotalAdjustments > 0 {
		stats.TriggerRate = float64(stats.TriggeredAdjustments) / float64(stats.TotalAdjustments)
		stats.SuccessRate = float64(stats.SuccessfulAdjustments) / float64(stats.TotalAdjustments)
	}
	
	if avgTimeToTriggerSeconds.Valid {
		stats.AverageTimeToTrigger = time.Duration(avgTimeToTriggerSeconds.Float64 * float64(time.Second))
	}
	
	return &stats, nil
}

// getTopPerformingAdjustments gets the top performing adjustments
func (slpt *StopLossPerformanceTracker) getTopPerformingAdjustments(ctx context.Context, days, limit int) ([]AdjustmentPerformance, error) {
	query := `
		SELECT 
			adjustment_id, position_id, symbol, adjustment_time, old_stop_loss, new_stop_loss,
			price_at_adjustment, adjustment_type, was_triggered, effectiveness_score
		FROM stop_loss_performance 
		WHERE adjustment_time >= DATE_SUB(NOW(), INTERVAL ? DAY)
		ORDER BY effectiveness_score DESC
		LIMIT ?
	`
	
	rows, err := slpt.db.QueryContext(ctx, query, days, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var adjustments []AdjustmentPerformance
	for rows.Next() {
		var adj AdjustmentPerformance
		err := rows.Scan(
			&adj.AdjustmentID, &adj.PositionID, &adj.Symbol,
			&adj.AdjustmentTime, &adj.OldStopLoss, &adj.NewStopLoss,
			&adj.PriceAtAdjustment, &adj.AdjustmentType, &adj.WasTriggered,
			&adj.EffectivenessScore,
		)
		if err != nil {
			return nil, err
		}
		adjustments = append(adjustments, adj)
	}

	return adjustments, nil
}

// getWorstPerformingAdjustments gets the worst performing adjustments
func (slpt *StopLossPerformanceTracker) getWorstPerformingAdjustments(ctx context.Context, days, limit int) ([]AdjustmentPerformance, error) {
	query := `
		SELECT 
			adjustment_id, position_id, symbol, adjustment_time, old_stop_loss, new_stop_loss,
			price_at_adjustment, adjustment_type, was_triggered, effectiveness_score
		FROM stop_loss_performance 
		WHERE adjustment_time >= DATE_SUB(NOW(), INTERVAL ? DAY)
		ORDER BY effectiveness_score ASC
		LIMIT ?
	`
	
	rows, err := slpt.db.QueryContext(ctx, query, days, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var adjustments []AdjustmentPerformance
	for rows.Next() {
		var adj AdjustmentPerformance
		err := rows.Scan(
			&adj.AdjustmentID, &adj.PositionID, &adj.Symbol,
			&adj.AdjustmentTime, &adj.OldStopLoss, &adj.NewStopLoss,
			&adj.PriceAtAdjustment, &adj.AdjustmentType, &adj.WasTriggered,
			&adj.EffectivenessScore,
		)
		if err != nil {
			return nil, err
		}
		adjustments = append(adjustments, adj)
	}

	return adjustments, nil
}

// getPerformanceByAdjustmentType gets performance statistics by adjustment type
func (slpt *StopLossPerformanceTracker) getPerformanceByAdjustmentType(ctx context.Context, days int) (map[string]PerformanceStatistics, error) {
	query := `
		SELECT 
			adjustment_type,
			COUNT(*) as total_adjustments,
			SUM(CASE WHEN was_triggered THEN 1 ELSE 0 END) as triggered_adjustments,
			AVG(effectiveness_score) as avg_effectiveness,
			SUM(CASE WHEN effectiveness_score > 0.6 THEN 1 ELSE 0 END) as successful_adjustments
		FROM stop_loss_performance 
		WHERE adjustment_time >= DATE_SUB(NOW(), INTERVAL ? DAY)
		GROUP BY adjustment_type
	`
	
	rows, err := slpt.db.QueryContext(ctx, query, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	performanceByType := make(map[string]PerformanceStatistics)
	
	for rows.Next() {
		var adjustmentType string
		var stats PerformanceStatistics
		
		err := rows.Scan(
			&adjustmentType,
			&stats.TotalAdjustments,
			&stats.TriggeredAdjustments,
			&stats.AverageEffectiveness,
			&stats.SuccessfulAdjustments,
		)
		if err != nil {
			return nil, err
		}
		
		// Calculate derived metrics
		if stats.TotalAdjustments > 0 {
			stats.TriggerRate = float64(stats.TriggeredAdjustments) / float64(stats.TotalAdjustments)
			stats.SuccessRate = float64(stats.SuccessfulAdjustments) / float64(stats.TotalAdjustments)
		}
		
		performanceByType[adjustmentType] = stats
	}

	return performanceByType, nil
}