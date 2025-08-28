package risk

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/automation/scheduler/shared"
	"qcat/internal/config"
	"qcat/internal/database"
	"qcat/internal/exchange/account"
)

// StopLossExecutor handles the execution of stop loss adjustments and integration with position monitoring
type StopLossExecutor struct {
	adjuster       *StopLossAdjuster
	config         *config.Config
	db             *database.DB
	accountManager *account.Manager
	configManager  *shared.ConfigManager
	errorHandler   *shared.ErrorHandler
	mu             sync.RWMutex
	isRunning      bool
	lastExecution  time.Time
	metrics        map[string]interface{}
	performanceTracker *StopLossPerformanceTracker
}

// StopLossPerformanceTracker tracks the effectiveness of stop loss adjustments
type StopLossPerformanceTracker struct {
	db             *database.DB
	mu             sync.RWMutex
	metrics        map[string]interface{}
	adjustmentHistory []AdjustmentPerformance
}

// AdjustmentPerformance tracks the performance of a stop loss adjustment
type AdjustmentPerformance struct {
	AdjustmentID     string    `json:"adjustment_id"`
	PositionID       string    `json:"position_id"`
	Symbol           string    `json:"symbol"`
	AdjustmentTime   time.Time `json:"adjustment_time"`
	OldStopLoss      float64   `json:"old_stop_loss"`
	NewStopLoss      float64   `json:"new_stop_loss"`
	PriceAtAdjustment float64  `json:"price_at_adjustment"`
	AdjustmentType   string    `json:"adjustment_type"`
	
	// Performance metrics
	WasTriggered     bool      `json:"was_triggered"`
	TriggerTime      *time.Time `json:"trigger_time,omitempty"`
	TriggerPrice     float64   `json:"trigger_price"`
	PnLAtTrigger     float64   `json:"pnl_at_trigger"`
	EffectivenessScore float64 `json:"effectiveness_score"`
	
	// Analysis
	WouldOldHaveBeenBetter bool    `json:"would_old_have_been_better"`
	PnLDifference         float64  `json:"pnl_difference"`
	TimeToTrigger         *time.Duration `json:"time_to_trigger,omitempty"`
}

// ExecutionResult represents the result of stop loss execution
type ExecutionResult struct {
	TotalAdjustments    int                    `json:"total_adjustments"`
	SuccessfulAdjustments int                  `json:"successful_adjustments"`
	FailedAdjustments   int                    `json:"failed_adjustments"`
	ExecutionTime       time.Duration          `json:"execution_time"`
	Errors              []string               `json:"errors"`
	AdjustmentDetails   []AdjustmentDetail     `json:"adjustment_details"`
	Timestamp           time.Time              `json:"timestamp"`
}

// AdjustmentDetail provides details about a specific adjustment
type AdjustmentDetail struct {
	PositionID      string    `json:"position_id"`
	Symbol          string    `json:"symbol"`
	Success         bool      `json:"success"`
	OldLevel        float64   `json:"old_level"`
	NewLevel        float64   `json:"new_level"`
	AdjustmentType  string    `json:"adjustment_type"`
	ExecutionTime   time.Duration `json:"execution_time"`
	Error           string    `json:"error,omitempty"`
}

// PositionMonitoringIntegration handles integration with existing position monitoring
type PositionMonitoringIntegration struct {
	executor       *StopLossExecutor
	monitoringInterval time.Duration
	stopChan       chan struct{}
	mu             sync.RWMutex
	isActive       bool
}

// NewStopLossExecutor creates a new stop loss executor
func NewStopLossExecutor(adjuster *StopLossAdjuster, cfg *config.Config, db *database.DB, accountManager *account.Manager) *StopLossExecutor {
	configManager := shared.NewConfigManager()
	
	// Initialize error handling
	retryStrategy := shared.NewRetryStrategy(3, time.Second, time.Minute*5, 2.0)
	circuitBreaker := shared.NewCircuitBreaker(shared.CircuitBreakerConfig{
		FailureThreshold: 5,
		RecoveryTimeout:  time.Minute * 5,
		HalfOpenRequests: 3,
		SuccessThreshold: 2,
	})
	errorHandler := shared.NewErrorHandler(retryStrategy, circuitBreaker)

	performanceTracker := &StopLossPerformanceTracker{
		db:      db,
		metrics: make(map[string]interface{}),
		adjustmentHistory: make([]AdjustmentPerformance, 0),
	}

	return &StopLossExecutor{
		adjuster:           adjuster,
		config:             cfg,
		db:                 db,
		accountManager:     accountManager,
		configManager:      configManager,
		errorHandler:       errorHandler,
		metrics:            make(map[string]interface{}),
		performanceTracker: performanceTracker,
	}
}

// ExecuteStopLossAdjustments executes stop loss adjustments with performance tracking
func (sle *StopLossExecutor) ExecuteStopLossAdjustments(ctx context.Context) (*ExecutionResult, error) {
	sle.mu.Lock()
	defer sle.mu.Unlock()

	startTime := time.Now()
	log.Printf("Starting stop loss adjustment execution")

	// Generate stop loss adjustments
	adjustments, err := sle.adjuster.GenerateStopLossAdjustments(ctx)
	if err != nil {
		return nil, shared.NewAutomationError(
			shared.ErrCodeCalculationFailed,
			fmt.Sprintf("Failed to generate stop loss adjustments: %v", err),
			"StopLossExecutor",
			shared.ErrorSeverityHigh,
			true,
		).WithContext("operation", "ExecuteStopLossAdjustments")
	}

	if len(adjustments) == 0 {
		log.Printf("No stop loss adjustments needed")
		return &ExecutionResult{
			TotalAdjustments:      0,
			SuccessfulAdjustments: 0,
			FailedAdjustments:     0,
			ExecutionTime:         time.Since(startTime),
			Errors:                []string{},
			AdjustmentDetails:     []AdjustmentDetail{},
			Timestamp:             time.Now(),
		}, nil
	}

	// Execute adjustments with detailed tracking
	result := &ExecutionResult{
		TotalAdjustments:  len(adjustments),
		ExecutionTime:     0,
		Errors:           []string{},
		AdjustmentDetails: make([]AdjustmentDetail, 0, len(adjustments)),
		Timestamp:        time.Now(),
	}

	for _, adjustment := range adjustments {
		detail := sle.executeAdjustmentWithTracking(ctx, adjustment)
		result.AdjustmentDetails = append(result.AdjustmentDetails, detail)
		
		if detail.Success {
			result.SuccessfulAdjustments++
		} else {
			result.FailedAdjustments++
			result.Errors = append(result.Errors, detail.Error)
		}
	}

	result.ExecutionTime = time.Since(startTime)

	// Update metrics
	sle.updateExecutionMetrics(result)

	// Log execution summary
	log.Printf("Stop loss adjustment execution completed: %d total, %d successful, %d failed, took %v",
		result.TotalAdjustments, result.SuccessfulAdjustments, result.FailedAdjustments, result.ExecutionTime)

	return result, nil
}

// executeAdjustmentWithTracking executes a single adjustment with detailed tracking
func (sle *StopLossExecutor) executeAdjustmentWithTracking(ctx context.Context, adjustment StopLossAdjustment) AdjustmentDetail {
	startTime := time.Now()
	
	detail := AdjustmentDetail{
		PositionID:     adjustment.PositionID,
		Symbol:         adjustment.Symbol,
		OldLevel:       adjustment.OldLevel,
		NewLevel:       adjustment.NewLevel,
		AdjustmentType: adjustment.AdjustmentType,
	}

	// Execute the adjustment
	err := sle.adjuster.AdjustStopLossLevels(ctx, []StopLossAdjustment{adjustment})
	detail.ExecutionTime = time.Since(startTime)
	
	if err != nil {
		detail.Success = false
		detail.Error = err.Error()
		log.Printf("Failed to execute stop loss adjustment for position %s: %v", adjustment.PositionID, err)
	} else {
		detail.Success = true
		
		// Start tracking performance for this adjustment
		err = sle.performanceTracker.StartTrackingAdjustment(ctx, adjustment)
		if err != nil {
			log.Printf("Warning: Failed to start performance tracking for adjustment %s: %v", adjustment.PositionID, err)
		}
	}

	return detail
}

// IntegrateWithPositionMonitoring integrates stop loss execution with position monitoring
func (sle *StopLossExecutor) IntegrateWithPositionMonitoring(ctx context.Context, monitoringInterval time.Duration) *PositionMonitoringIntegration {
	integration := &PositionMonitoringIntegration{
		executor:           sle,
		monitoringInterval: monitoringInterval,
		stopChan:          make(chan struct{}),
	}

	return integration
}

// StartMonitoring starts the position monitoring integration
func (pmi *PositionMonitoringIntegration) StartMonitoring(ctx context.Context) error {
	pmi.mu.Lock()
	defer pmi.mu.Unlock()

	if pmi.isActive {
		return fmt.Errorf("position monitoring integration is already active")
	}

	pmi.isActive = true
	
	go pmi.monitoringLoop(ctx)
	
	log.Printf("Position monitoring integration started with interval: %v", pmi.monitoringInterval)
	return nil
}

// StopMonitoring stops the position monitoring integration
func (pmi *PositionMonitoringIntegration) StopMonitoring() error {
	pmi.mu.Lock()
	defer pmi.mu.Unlock()

	if !pmi.isActive {
		return fmt.Errorf("position monitoring integration is not active")
	}

	close(pmi.stopChan)
	pmi.isActive = false
	
	log.Printf("Position monitoring integration stopped")
	return nil
}

// monitoringLoop runs the continuous monitoring loop
func (pmi *PositionMonitoringIntegration) monitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(pmi.monitoringInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Position monitoring integration stopped due to context cancellation")
			return
		case <-pmi.stopChan:
			log.Printf("Position monitoring integration stopped")
			return
		case <-ticker.C:
			pmi.executeMonitoringCycle(ctx)
		}
	}
}

// executeMonitoringCycle executes one monitoring cycle
func (pmi *PositionMonitoringIntegration) executeMonitoringCycle(ctx context.Context) {
	log.Printf("Executing position monitoring cycle")

	// Execute stop loss adjustments
	result, err := pmi.executor.ExecuteStopLossAdjustments(ctx)
	if err != nil {
		log.Printf("Error during stop loss adjustment execution: %v", err)
		return
	}

	// Update performance tracking
	err = pmi.executor.performanceTracker.UpdatePerformanceMetrics(ctx)
	if err != nil {
		log.Printf("Warning: Failed to update performance metrics: %v", err)
	}

	// Log cycle summary
	if result.TotalAdjustments > 0 {
		log.Printf("Monitoring cycle completed: %d adjustments processed", result.TotalAdjustments)
	}
}

// Performance Tracking Methods

// StartTrackingAdjustment starts tracking the performance of an adjustment
func (slpt *StopLossPerformanceTracker) StartTrackingAdjustment(ctx context.Context, adjustment StopLossAdjustment) error {
	slpt.mu.Lock()
	defer slpt.mu.Unlock()

	// Get current price for the position
	currentPrice, err := slpt.getCurrentPrice(ctx, adjustment.Symbol)
	if err != nil {
		return fmt.Errorf("failed to get current price for %s: %w", adjustment.Symbol, err)
	}

	// Create performance tracking record
	adjustmentID := shared.GenerateID("adj")
	performance := AdjustmentPerformance{
		AdjustmentID:      adjustmentID,
		PositionID:        adjustment.PositionID,
		Symbol:            adjustment.Symbol,
		AdjustmentTime:    adjustment.Timestamp,
		OldStopLoss:       adjustment.OldLevel,
		NewStopLoss:       adjustment.NewLevel,
		PriceAtAdjustment: currentPrice,
		AdjustmentType:    adjustment.AdjustmentType,
		WasTriggered:      false,
		EffectivenessScore: 0.0,
	}

	// Store in database
	err = slpt.storeAdjustmentPerformance(ctx, performance)
	if err != nil {
		return fmt.Errorf("failed to store adjustment performance: %w", err)
	}

	// Add to in-memory tracking
	slpt.adjustmentHistory = append(slpt.adjustmentHistory, performance)

	log.Printf("Started tracking performance for adjustment %s (Position: %s, Symbol: %s)", 
		adjustmentID, adjustment.PositionID, adjustment.Symbol)

	return nil
}

// UpdatePerformanceMetrics updates performance metrics for all tracked adjustments
func (slpt *StopLossPerformanceTracker) UpdatePerformanceMetrics(ctx context.Context) error {
	slpt.mu.Lock()
	defer slpt.mu.Unlock()

	log.Printf("Updating stop loss performance metrics")

	// Get all active tracking records
	activeTracking, err := slpt.getActiveTrackingRecords(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active tracking records: %w", err)
	}

	updatedCount := 0
	for _, tracking := range activeTracking {
		updated, err := slpt.updateSinglePerformanceRecord(ctx, tracking)
		if err != nil {
			log.Printf("Warning: Failed to update performance record %s: %v", tracking.AdjustmentID, err)
			continue
		}
		if updated {
			updatedCount++
		}
	}

	// Calculate aggregate metrics
	err = slpt.calculateAggregateMetrics(ctx)
	if err != nil {
		log.Printf("Warning: Failed to calculate aggregate metrics: %v", err)
	}

	log.Printf("Updated %d performance records", updatedCount)
	return nil
}

// updateSinglePerformanceRecord updates a single performance record
func (slpt *StopLossPerformanceTracker) updateSinglePerformanceRecord(ctx context.Context, tracking AdjustmentPerformance) (bool, error) {
	// Check if position still exists and get current status
	position, err := slpt.getPositionStatus(ctx, tracking.PositionID)
	if err != nil {
		return false, err
	}

	if position == nil {
		// Position closed - check if stop loss was triggered
		return slpt.analyzeClosedPosition(ctx, tracking)
	}

	// Position still active - check if stop loss needs updating
	return slpt.analyzeActivePosition(ctx, tracking, position)
}

// analyzeClosedPosition analyzes a closed position to determine stop loss effectiveness
func (slpt *StopLossPerformanceTracker) analyzeClosedPosition(ctx context.Context, tracking AdjustmentPerformance) (bool, error) {
	// Get position closure details
	closureDetails, err := slpt.getPositionClosureDetails(ctx, tracking.PositionID)
	if err != nil {
		return false, err
	}

	// Determine if stop loss was triggered
	wasTriggered := slpt.wasStopLossTriggered(closureDetails, tracking.NewStopLoss)
	
	// Calculate effectiveness score
	effectivenessScore := slpt.calculateEffectivenessScore(tracking, closureDetails, wasTriggered)

	// Update the tracking record
	updatedTracking := tracking
	updatedTracking.WasTriggered = wasTriggered
	updatedTracking.EffectivenessScore = effectivenessScore
	
	if wasTriggered {
		updatedTracking.TriggerTime = &closureDetails.CloseTime
		updatedTracking.TriggerPrice = closureDetails.ClosePrice
		updatedTracking.PnLAtTrigger = closureDetails.RealizedPnL
		
		duration := closureDetails.CloseTime.Sub(tracking.AdjustmentTime)
		updatedTracking.TimeToTrigger = &duration
	}

	// Analyze if old stop loss would have been better
	updatedTracking.WouldOldHaveBeenBetter = slpt.wouldOldStopLossHaveBeenBetter(tracking, closureDetails)
	updatedTracking.PnLDifference = slpt.calculatePnLDifference(tracking, closureDetails)

	// Update in database
	err = slpt.updateAdjustmentPerformance(ctx, updatedTracking)
	if err != nil {
		return false, err
	}

	log.Printf("Analyzed closed position %s: triggered=%v, effectiveness=%.2f", 
		tracking.PositionID, wasTriggered, effectivenessScore)

	return true, nil
}

// analyzeActivePosition analyzes an active position
func (slpt *StopLossPerformanceTracker) analyzeActivePosition(ctx context.Context, tracking AdjustmentPerformance, position *shared.Position) (bool, error) {
	// For active positions, just update current metrics
	// Full analysis will happen when position closes
	
	// Calculate current unrealized effectiveness
	currentEffectiveness := slpt.calculateCurrentEffectiveness(tracking, position)
	
	if currentEffectiveness != tracking.EffectivenessScore {
		updatedTracking := tracking
		updatedTracking.EffectivenessScore = currentEffectiveness
		
		err := slpt.updateAdjustmentPerformance(ctx, updatedTracking)
		if err != nil {
			return false, err
		}
		
		return true, nil
	}

	return false, nil
}

// Helper methods for performance tracking

// getCurrentPrice gets current price for a symbol
func (slpt *StopLossPerformanceTracker) getCurrentPrice(ctx context.Context, symbol string) (float64, error) {
	query := `
		SELECT close_price 
		FROM market_data 
		WHERE symbol = ? 
		ORDER BY timestamp DESC 
		LIMIT 1
	`
	
	var price float64
	err := slpt.db.QueryRowContext(ctx, query, symbol).Scan(&price)
	if err != nil {
		return 0, err
	}

	return price, nil
}

// getPositionStatus gets current position status
func (slpt *StopLossPerformanceTracker) getPositionStatus(ctx context.Context, positionID string) (*shared.Position, error) {
	query := `
		SELECT 
			id, symbol, side, size, entry_price, current_price,
			unrealized_pnl, realized_pnl, leverage, margin_used, created_at
		FROM positions 
		WHERE id = ?
	`
	
	var pos shared.Position
	err := slpt.db.QueryRowContext(ctx, query, positionID).Scan(
		&pos.ID, &pos.Symbol, &pos.Side, &pos.Size, &pos.EntryPrice,
		&pos.CurrentPrice, &pos.UnrealizedPnL, &pos.RealizedPnL,
		&pos.Leverage, &pos.MarginUsed, &pos.Timestamp,
	)
	
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil // Position not found (likely closed)
		}
		return nil, err
	}

	return &pos, nil
}

// Data structures for performance tracking

// PositionClosureDetails represents details about a closed position
type PositionClosureDetails struct {
	PositionID   string    `json:"position_id"`
	CloseTime    time.Time `json:"close_time"`
	ClosePrice   float64   `json:"close_price"`
	CloseReason  string    `json:"close_reason"`
	RealizedPnL  float64   `json:"realized_pnl"`
	FinalSize    float64   `json:"final_size"`
}

// getPositionClosureDetails gets closure details for a position
func (slpt *StopLossPerformanceTracker) getPositionClosureDetails(ctx context.Context, positionID string) (*PositionClosureDetails, error) {
	query := `
		SELECT position_id, close_time, close_price, close_reason, realized_pnl, final_size
		FROM position_closures 
		WHERE position_id = ?
	`
	
	var details PositionClosureDetails
	err := slpt.db.QueryRowContext(ctx, query, positionID).Scan(
		&details.PositionID, &details.CloseTime, &details.ClosePrice,
		&details.CloseReason, &details.RealizedPnL, &details.FinalSize,
	)
	
	if err != nil {
		return nil, err
	}

	return &details, nil
}

// Additional helper methods would be implemented here...
// (wasStopLossTriggered, calculateEffectivenessScore, etc.)

// updateExecutionMetrics updates execution-related metrics
func (sle *StopLossExecutor) updateExecutionMetrics(result *ExecutionResult) {
	sle.metrics["last_execution_time"] = result.Timestamp
	sle.metrics["last_execution_duration"] = result.ExecutionTime
	sle.metrics["last_total_adjustments"] = result.TotalAdjustments
	sle.metrics["last_successful_adjustments"] = result.SuccessfulAdjustments
	sle.metrics["last_failed_adjustments"] = result.FailedAdjustments
	sle.metrics["last_success_rate"] = float64(result.SuccessfulAdjustments) / float64(result.TotalAdjustments)
	
	// Update cumulative metrics
	if totalExecs, exists := sle.metrics["total_executions"]; exists {
		sle.metrics["total_executions"] = totalExecs.(int) + 1
	} else {
		sle.metrics["total_executions"] = 1
	}
}

// GetMetrics returns current executor metrics
func (sle *StopLossExecutor) GetMetrics() map[string]interface{} {
	sle.mu.RLock()
	defer sle.mu.RUnlock()
	
	// Combine executor and performance tracker metrics
	metrics := make(map[string]interface{})
	for k, v := range sle.metrics {
		metrics[k] = v
	}
	
	// Add performance tracker metrics
	sle.performanceTracker.mu.RLock()
	for k, v := range sle.performanceTracker.metrics {
		metrics["performance_"+k] = v
	}
	sle.performanceTracker.mu.RUnlock()
	
	return metrics
}

// IsRunning returns whether the executor is currently running
func (sle *StopLossExecutor) IsRunning() bool {
	sle.mu.RLock()
	defer sle.mu.RUnlock()
	return sle.isRunning
}

// Start starts the stop loss executor
func (sle *StopLossExecutor) Start() error {
	sle.mu.Lock()
	defer sle.mu.Unlock()
	
	sle.isRunning = true
	sle.lastExecution = time.Now()
	log.Printf("Stop loss executor started")
	return nil
}

// Stop stops the stop loss executor
func (sle *StopLossExecutor) Stop() error {
	sle.mu.Lock()
	defer sle.mu.Unlock()
	
	sle.isRunning = false
	log.Printf("Stop loss executor stopped")
	return nil
}