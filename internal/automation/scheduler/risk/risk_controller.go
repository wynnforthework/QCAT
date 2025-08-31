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

// RiskController handles risk control trigger mechanisms
type RiskController struct {
	config         *config.Config
	db             *database.DB
	accountManager *account.Manager
	riskMonitor    *RiskMonitor
	errorHandler   *shared.ErrorHandler
	mu             sync.RWMutex
	isRunning      bool
	emergencyMode  bool
	lastAction     time.Time
	actionHistory  []RiskAction
}

// RiskAction represents a risk control action taken
type RiskAction struct {
	ID          string                 `json:"id"`
	Type        RiskActionType         `json:"type"`
	Trigger     string                 `json:"trigger"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Result      ActionResult           `json:"result"`
	ExecutedAt  time.Time              `json:"executed_at"`
	Duration    time.Duration          `json:"duration"`
	Timestamp   time.Time              `json:"timestamp"`
}

// RiskActionType defines types of risk control actions
type RiskActionType int

const (
	ActionTypePositionReduction RiskActionType = iota
	ActionTypeEmergencyStop
	ActionTypeMarginIncrease
	ActionTypeHedgeActivation
	ActionTypeLeverageReduction
	ActionTypeCircuitBreaker
)

// String returns string representation of RiskActionType
func (rat RiskActionType) String() string {
	switch rat {
	case ActionTypePositionReduction:
		return "POSITION_REDUCTION"
	case ActionTypeEmergencyStop:
		return "EMERGENCY_STOP"
	case ActionTypeMarginIncrease:
		return "MARGIN_INCREASE"
	case ActionTypeHedgeActivation:
		return "HEDGE_ACTIVATION"
	case ActionTypeLeverageReduction:
		return "LEVERAGE_REDUCTION"
	case ActionTypeCircuitBreaker:
		return "CIRCUIT_BREAKER"
	default:
		return "UNKNOWN"
	}
}

// ActionResult represents the result of a risk action
type ActionResult struct {
	Success           bool                   `json:"success"`
	Error             string                 `json:"error,omitempty"`
	AffectedPositions []string               `json:"affected_positions"`
	AmountReduced     float64                `json:"amount_reduced"`
	NewRiskLevel      shared.RiskLevel       `json:"new_risk_level"`
	Metrics           map[string]interface{} `json:"metrics"`
}

// NewRiskController creates a new risk controller
func NewRiskController(cfg *config.Config, db *database.DB, accountManager *account.Manager, riskMonitor *RiskMonitor) *RiskController {
	// Initialize error handling
	retryStrategy := shared.NewRetryStrategy(3, time.Second, time.Minute*5, 2.0)
	circuitBreaker := shared.NewCircuitBreaker(shared.CircuitBreakerConfig{
		FailureThreshold: 5,
		RecoveryTimeout:  time.Minute * 5,
		HalfOpenRequests: 3,
		SuccessThreshold: 2,
	})
	errorHandler := shared.NewErrorHandler(retryStrategy, circuitBreaker)

	return &RiskController{
		config:         cfg,
		db:             db,
		accountManager: accountManager,
		riskMonitor:    riskMonitor,
		errorHandler:   errorHandler,
		actionHistory:  make([]RiskAction, 0),
	}
}

// TriggerPositionReduction triggers automatic position reduction when thresholds are exceeded
func (rc *RiskController) TriggerPositionReduction(ctx context.Context, marginStatus *MarginStatus, reductionPercent float64) (*RiskAction, error) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	log.Printf("Triggering position reduction: margin_ratio=%.4f, reduction=%.2f%%",
		marginStatus.MarginRatio, reductionPercent*100)

	startTime := time.Now()
	action := &RiskAction{
		ID:          shared.GenerateID("risk_action"),
		Type:        ActionTypePositionReduction,
		Trigger:     fmt.Sprintf("Margin ratio %.4f exceeds threshold", marginStatus.MarginRatio),
		Description: fmt.Sprintf("Reduce positions by %.2f%% due to high margin usage", reductionPercent*100),
		Parameters: map[string]interface{}{
			"margin_ratio":      marginStatus.MarginRatio,
			"reduction_percent": reductionPercent,
			"trigger_threshold": 0.8, // From config
		},
		ExecutedAt: startTime,
	}

	// Get current positions to reduce
	positions, err := rc.getCurrentPositions(ctx)
	if err != nil {
		action.Result = ActionResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to get positions: %v", err),
		}
		action.Duration = time.Since(startTime)
		rc.recordAction(*action)
		return action, err
	}

	// Calculate positions to reduce (prioritize by risk)
	positionsToReduce, err := rc.selectPositionsForReduction(ctx, positions, reductionPercent)
	if err != nil {
		action.Result = ActionResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to select positions for reduction: %v", err),
		}
		action.Duration = time.Since(startTime)
		rc.recordAction(*action)
		return action, err
	}

	// Execute position reductions
	var affectedPositions []string
	var totalReduced float64
	var errors []string

	for _, reduction := range positionsToReduce {
		err := rc.executePositionReduction(ctx, reduction)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Position %s: %v", reduction.PositionID, err))
			continue
		}

		affectedPositions = append(affectedPositions, reduction.PositionID)
		totalReduced += reduction.ReductionAmount
	}

	// Check new risk level after reduction
	newMarginStatus, err := rc.riskMonitor.CheckMarginRatio(ctx)
	var newRiskLevel shared.RiskLevel
	if err != nil {
		log.Printf("Warning: Could not check new margin status after reduction: %v", err)
		newRiskLevel = shared.RiskLevelMedium // Conservative estimate
	} else {
		newRiskLevel = newMarginStatus.RiskLevel
	}

	// Prepare result
	success := len(errors) == 0
	var errorMsg string
	if len(errors) > 0 {
		errorMsg = fmt.Sprintf("Partial success: %d errors occurred", len(errors))
	}

	action.Result = ActionResult{
		Success:           success,
		Error:             errorMsg,
		AffectedPositions: affectedPositions,
		AmountReduced:     totalReduced,
		NewRiskLevel:      newRiskLevel,
		Metrics: map[string]interface{}{
			"positions_targeted":  len(positionsToReduce),
			"positions_reduced":   len(affectedPositions),
			"total_value_reduced": totalReduced,
			"error_count":         len(errors),
		},
	}
	action.Duration = time.Since(startTime)

	// Record action
	rc.recordAction(*action)
	rc.lastAction = time.Now()

	log.Printf("Position reduction completed: success=%t, positions=%d, amount=%.2f",
		success, len(affectedPositions), totalReduced)

	return action, nil
}

// TriggerEmergencyStop triggers emergency stop functionality for critical risk levels
func (rc *RiskController) TriggerEmergencyStop(ctx context.Context, reason string) (*RiskAction, error) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	log.Printf("EMERGENCY STOP TRIGGERED: %s", reason)

	startTime := time.Now()
	action := &RiskAction{
		ID:          shared.GenerateID("emergency_stop"),
		Type:        ActionTypeEmergencyStop,
		Trigger:     reason,
		Description: "Emergency stop - close all positions immediately",
		Parameters: map[string]interface{}{
			"emergency_reason": reason,
			"stop_time":        startTime,
		},
		ExecutedAt: startTime,
	}

	// Set emergency mode
	rc.emergencyMode = true

	// Get all active positions
	positions, err := rc.getCurrentPositions(ctx)
	if err != nil {
		action.Result = ActionResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to get positions for emergency stop: %v", err),
		}
		action.Duration = time.Since(startTime)
		rc.recordAction(*action)
		return action, err
	}

	// Close all positions immediately
	var affectedPositions []string
	var totalClosed float64
	var errors []string

	for _, position := range positions {
		err := rc.executeEmergencyClose(ctx, position)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Position %s: %v", position.ID, err))
			continue
		}

		affectedPositions = append(affectedPositions, position.ID)
		totalClosed += position.Size * position.CurrentPrice
	}

	// Cancel all pending orders
	cancelErr := rc.cancelAllPendingOrders(ctx)
	if cancelErr != nil {
		errors = append(errors, fmt.Sprintf("Order cancellation: %v", cancelErr))
	}

	// Prepare result
	success := len(errors) == 0
	var errorMsg string
	if len(errors) > 0 {
		errorMsg = fmt.Sprintf("Emergency stop completed with %d errors", len(errors))
	}

	action.Result = ActionResult{
		Success:           success,
		Error:             errorMsg,
		AffectedPositions: affectedPositions,
		AmountReduced:     totalClosed,
		NewRiskLevel:      shared.RiskLevelLow, // Should be low after emergency stop
		Metrics: map[string]interface{}{
			"total_positions":    len(positions),
			"positions_closed":   len(affectedPositions),
			"total_value_closed": totalClosed,
			"error_count":        len(errors),
			"emergency_mode":     true,
		},
	}
	action.Duration = time.Since(startTime)

	// Record action
	rc.recordAction(*action)
	rc.lastAction = time.Now()

	// Send emergency notification
	rc.sendEmergencyNotification(ctx, action)

	log.Printf("Emergency stop completed: success=%t, positions_closed=%d, value=%.2f",
		success, len(affectedPositions), totalClosed)

	return action, nil
}

// TriggerLeverageReduction triggers automatic leverage reduction
func (rc *RiskController) TriggerLeverageReduction(ctx context.Context, targetLeverage float64) (*RiskAction, error) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	log.Printf("Triggering leverage reduction to %.2fx", targetLeverage)

	startTime := time.Now()
	action := &RiskAction{
		ID:          shared.GenerateID("leverage_reduction"),
		Type:        ActionTypeLeverageReduction,
		Trigger:     "High risk conditions detected",
		Description: fmt.Sprintf("Reduce leverage to %.2fx", targetLeverage),
		Parameters: map[string]interface{}{
			"target_leverage": targetLeverage,
		},
		ExecutedAt: startTime,
	}

	// Get positions with high leverage
	positions, err := rc.getHighLeveragePositions(ctx, targetLeverage)
	if err != nil {
		action.Result = ActionResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to get high leverage positions: %v", err),
		}
		action.Duration = time.Since(startTime)
		rc.recordAction(*action)
		return action, err
	}

	// Reduce leverage for each position
	var affectedPositions []string
	var totalReduced float64
	var errors []string

	for _, position := range positions {
		newSize, err := rc.calculateReducedPositionSize(position, targetLeverage)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Position %s calculation: %v", position.ID, err))
			continue
		}

		reduction := PositionReduction{
			PositionID:       position.ID,
			CurrentSize:      position.Size,
			NewSize:          newSize,
			ReductionAmount:  position.Size - newSize,
			ReductionPercent: (position.Size - newSize) / position.Size,
		}

		err = rc.executePositionReduction(ctx, reduction)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Position %s execution: %v", position.ID, err))
			continue
		}

		affectedPositions = append(affectedPositions, position.ID)
		totalReduced += reduction.ReductionAmount * position.CurrentPrice
	}

	// Prepare result
	success := len(errors) == 0
	var errorMsg string
	if len(errors) > 0 {
		errorMsg = fmt.Sprintf("Leverage reduction completed with %d errors", len(errors))
	}

	action.Result = ActionResult{
		Success:           success,
		Error:             errorMsg,
		AffectedPositions: affectedPositions,
		AmountReduced:     totalReduced,
		NewRiskLevel:      shared.RiskLevelMedium, // Should be reduced after leverage reduction
		Metrics: map[string]interface{}{
			"target_leverage":     targetLeverage,
			"positions_targeted":  len(positions),
			"positions_reduced":   len(affectedPositions),
			"total_value_reduced": totalReduced,
			"error_count":         len(errors),
		},
	}
	action.Duration = time.Since(startTime)

	// Record action
	rc.recordAction(*action)
	rc.lastAction = time.Now()

	log.Printf("Leverage reduction completed: success=%t, positions=%d, amount=%.2f",
		success, len(affectedPositions), totalReduced)

	return action, nil
}

// PositionReduction represents a position reduction operation
type PositionReduction struct {
	PositionID       string  `json:"position_id"`
	CurrentSize      float64 `json:"current_size"`
	NewSize          float64 `json:"new_size"`
	ReductionAmount  float64 `json:"reduction_amount"`
	ReductionPercent float64 `json:"reduction_percent"`
}

// Helper methods

// getCurrentPositions gets current active positions
func (rc *RiskController) getCurrentPositions(ctx context.Context) ([]shared.Position, error) {
	query := `
		SELECT 
			id, symbol, side, size, entry_price, current_price,
			unrealized_pnl, realized_pnl, leverage, margin_used, created_at
		FROM positions 
		WHERE status = 'ACTIVE'
		ORDER BY margin_used DESC
	`

	rows, err := rc.db.QueryContext(ctx, query)
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

// selectPositionsForReduction selects positions for reduction based on risk
func (rc *RiskController) selectPositionsForReduction(ctx context.Context, positions []shared.Position, reductionPercent float64) ([]PositionReduction, error) {
	var reductions []PositionReduction

	// Calculate total portfolio value
	totalValue := 0.0
	for _, pos := range positions {
		totalValue += pos.Size * pos.CurrentPrice
	}

	targetReduction := totalValue * reductionPercent
	currentReduction := 0.0

	// Sort positions by risk (highest margin usage ratio first)
	// This is a simplified approach - in practice, you'd use more sophisticated risk metrics
	sortedPositions := make([]shared.Position, len(positions))
	copy(sortedPositions, positions)

	// Sort by position value (smaller positions first for better granularity)
	// This allows us to select multiple smaller positions to reach the target
	for i := 0; i < len(sortedPositions)-1; i++ {
		for j := i + 1; j < len(sortedPositions); j++ {
			posValue1 := sortedPositions[i].Size * sortedPositions[i].CurrentPrice
			posValue2 := sortedPositions[j].Size * sortedPositions[j].CurrentPrice

			if posValue1 > posValue2 {
				sortedPositions[i], sortedPositions[j] = sortedPositions[j], sortedPositions[i]
			}
		}
	}

	for _, pos := range sortedPositions {
		if currentReduction >= targetReduction {
			break
		}

		// Calculate how much to reduce this position
		positionValue := pos.Size * pos.CurrentPrice
		remainingReduction := targetReduction - currentReduction

		var reductionAmount float64
		if positionValue <= remainingReduction {
			// Reduce entire position
			reductionAmount = pos.Size
		} else {
			// Partial reduction
			reductionAmount = pos.Size * (remainingReduction / positionValue)
		}

		if reductionAmount > 0 {
			reduction := PositionReduction{
				PositionID:       pos.ID,
				CurrentSize:      pos.Size,
				NewSize:          pos.Size - reductionAmount,
				ReductionAmount:  reductionAmount,
				ReductionPercent: reductionAmount / pos.Size,
			}
			reductions = append(reductions, reduction)
			currentReduction += reductionAmount * pos.CurrentPrice
		}
	}

	return reductions, nil
}

// executePositionReduction executes a position reduction
func (rc *RiskController) executePositionReduction(ctx context.Context, reduction PositionReduction) error {
	// Update position in database
	query := `
		UPDATE positions 
		SET size = $1, updated_at = NOW()
		WHERE id = $2 AND status = 'ACTIVE'
	`

	result, err := rc.db.ExecContext(ctx, query, reduction.NewSize, reduction.PositionID)
	if err != nil {
		return fmt.Errorf("failed to update position in database: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("position %s not found or already closed", reduction.PositionID)
	}

	// In a real implementation, you would also:
	// 1. Place market orders to reduce the position on the exchange
	// 2. Update margin calculations
	// 3. Send notifications

	log.Printf("Position %s reduced from %.4f to %.4f (%.2f%% reduction)",
		reduction.PositionID, reduction.CurrentSize, reduction.NewSize, reduction.ReductionPercent*100)

	return nil
}

// executeEmergencyClose executes emergency position closure
func (rc *RiskController) executeEmergencyClose(ctx context.Context, position shared.Position) error {
	// Close position in database
	query := `
		UPDATE positions 
		SET status = 'CLOSED', size = 0, closed_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status = 'ACTIVE'
	`

	result, err := rc.db.ExecContext(ctx, query, position.ID)
	if err != nil {
		return fmt.Errorf("failed to close position in database: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("position %s not found or already closed", position.ID)
	}

	// In a real implementation, you would also place market orders on the exchange

	log.Printf("Emergency close: Position %s (%s) closed, size: %.4f",
		position.ID, position.Symbol, position.Size)

	return nil
}

// cancelAllPendingOrders cancels all pending orders
func (rc *RiskController) cancelAllPendingOrders(ctx context.Context) error {
	query := `
		UPDATE orders 
		SET status = 'CANCELLED', updated_at = NOW()
		WHERE status IN ('PENDING', 'PARTIALLY_FILLED')
	`

	result, err := rc.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to cancel orders in database: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	log.Printf("Cancelled %d pending orders", rowsAffected)
	return nil
}

// getHighLeveragePositions gets positions with leverage above threshold
func (rc *RiskController) getHighLeveragePositions(ctx context.Context, maxLeverage float64) ([]shared.Position, error) {
	query := `
		SELECT 
			id, symbol, side, size, entry_price, current_price,
			unrealized_pnl, realized_pnl, leverage, margin_used, created_at
		FROM positions 
		WHERE status = 'ACTIVE' AND leverage > $1
		ORDER BY leverage DESC
	`

	rows, err := rc.db.QueryContext(ctx, query, maxLeverage)
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

// calculateReducedPositionSize calculates new position size for target leverage
func (rc *RiskController) calculateReducedPositionSize(position shared.Position, targetLeverage float64) (float64, error) {
	if targetLeverage <= 0 {
		return 0, fmt.Errorf("target leverage must be positive")
	}

	if position.Leverage <= targetLeverage {
		return position.Size, nil // No reduction needed
	}

	// Calculate new size based on leverage ratio
	leverageRatio := targetLeverage / position.Leverage
	newSize := position.Size * leverageRatio

	return newSize, nil
}

// recordAction records a risk action in the database and memory
func (rc *RiskController) recordAction(action RiskAction) {
	// Add to memory
	rc.actionHistory = append(rc.actionHistory, action)

	// Keep only last 100 actions in memory
	if len(rc.actionHistory) > 100 {
		rc.actionHistory = rc.actionHistory[1:]
	}

	// Record in database
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := rc.recordActionInDatabase(ctx, action)
		if err != nil {
			log.Printf("Failed to record risk action in database: %v", err)
		}
	}()
}

// recordActionInDatabase records action in database
func (rc *RiskController) recordActionInDatabase(ctx context.Context, action RiskAction) error {
	query := `
		INSERT INTO risk_actions (
			id, type, trigger_reason, description, parameters,
			success, error_message, affected_positions, amount_reduced,
			new_risk_level, metrics, executed_at, duration_ms
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	// Convert parameters and metrics to JSON strings (simplified)
	parametersJSON := fmt.Sprintf("%v", action.Parameters)
	metricsJSON := fmt.Sprintf("%v", action.Result.Metrics)
	affectedPositionsJSON := fmt.Sprintf("%v", action.Result.AffectedPositions)

	_, err := rc.db.ExecContext(ctx, query,
		action.ID,
		action.Type.String(),
		action.Trigger,
		action.Description,
		parametersJSON,
		action.Result.Success,
		action.Result.Error,
		affectedPositionsJSON,
		action.Result.AmountReduced,
		action.Result.NewRiskLevel.String(),
		metricsJSON,
		action.ExecutedAt,
		action.Duration.Milliseconds(),
	)

	return err
}

// sendEmergencyNotification sends emergency notification
func (rc *RiskController) sendEmergencyNotification(ctx context.Context, action *RiskAction) {
	// In a real implementation, this would send notifications via:
	// - Email
	// - SMS
	// - Slack/Discord webhooks
	// - Push notifications

	log.Printf("EMERGENCY NOTIFICATION: %s - %s", action.Type.String(), action.Description)
}

// GetActionHistory returns recent risk actions
func (rc *RiskController) GetActionHistory() []RiskAction {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	// Return a copy
	history := make([]RiskAction, len(rc.actionHistory))
	copy(history, rc.actionHistory)
	return history
}

// IsEmergencyMode returns whether the controller is in emergency mode
func (rc *RiskController) IsEmergencyMode() bool {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.emergencyMode
}

// ClearEmergencyMode clears emergency mode (should be called manually after review)
func (rc *RiskController) ClearEmergencyMode() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.emergencyMode = false
	log.Printf("Emergency mode cleared")
}

// Start starts the risk controller
func (rc *RiskController) Start() error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.isRunning = true
	log.Printf("Risk controller started")
	return nil
}

// Stop stops the risk controller
func (rc *RiskController) Stop() error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.isRunning = false
	log.Printf("Risk controller stopped")
	return nil
}
