package risk

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"qcat/internal/automation/scheduler/shared"
	"qcat/internal/config"
	"qcat/internal/database"
	"qcat/internal/exchange/account"
)

// StopLossIntegration integrates stop loss functionality with the existing risk scheduler
type StopLossIntegration struct {
	adjuster       *StopLossAdjuster
	executor       *StopLossExecutor
	riskMonitor    *RiskMonitor
	config         *config.Config
	db             *database.DB
	accountManager *account.Manager
	configManager  *shared.ConfigManager
	errorHandler   *shared.ErrorHandler
	mu             sync.RWMutex
	isRunning      bool
	lastUpdate     time.Time
	metrics        map[string]interface{}
	
	// Integration settings
	updateInterval     time.Duration
	riskThresholds     StopLossRiskThresholds
	volatilitySettings VolatilityBasedSettings
}

// StopLossRiskThresholds defines risk-based thresholds for stop loss adjustments
type StopLossRiskThresholds struct {
	LowRiskMultiplier    float64 `json:"low_risk_multiplier"`    // 0.8 - tighter stops for low risk
	MediumRiskMultiplier float64 `json:"medium_risk_multiplier"` // 1.0 - normal stops
	HighRiskMultiplier   float64 `json:"high_risk_multiplier"`   // 1.2 - wider stops for high risk
	CriticalRiskMultiplier float64 `json:"critical_risk_multiplier"` // 1.5 - much wider stops
	
	// Emergency thresholds
	EmergencyMarginRatio float64 `json:"emergency_margin_ratio"` // 0.9 - trigger emergency adjustments
	EmergencyVaRLimit    float64 `json:"emergency_var_limit"`    // 0.05 - 5% VaR limit
}

// VolatilityBasedSettings defines volatility-based adjustment settings
type VolatilityBasedSettings struct {
	LowVolatilityThreshold  float64 `json:"low_volatility_threshold"`  // 0.15 - 15% annualized
	HighVolatilityThreshold float64 `json:"high_volatility_threshold"` // 0.35 - 35% annualized
	
	// Adjustment factors
	LowVolatilityFactor  float64 `json:"low_volatility_factor"`  // 0.8 - tighter stops
	HighVolatilityFactor float64 `json:"high_volatility_factor"` // 1.3 - wider stops
	
	// Regime-based adjustments
	BullMarketFactor    float64 `json:"bull_market_factor"`    // 0.9 - slightly tighter
	BearMarketFactor    float64 `json:"bear_market_factor"`    // 1.1 - slightly wider
	VolatileMarketFactor float64 `json:"volatile_market_factor"` // 1.2 - wider stops
}

// IntegratedStopLossResult represents the result of integrated stop loss processing
type IntegratedStopLossResult struct {
	RiskAssessment      *shared.RiskAssessment    `json:"risk_assessment"`
	MarketRegime        *shared.MarketRegime      `json:"market_regime"`
	AdjustmentsTrigger  []AdjustmentTrigger       `json:"adjustments_triggered"`
	ExecutionResult     *ExecutionResult          `json:"execution_result"`
	PerformanceMetrics  map[string]interface{}    `json:"performance_metrics"`
	ProcessingTime      time.Duration             `json:"processing_time"`
	Timestamp           time.Time                 `json:"timestamp"`
}

// AdjustmentTrigger represents a trigger for stop loss adjustment
type AdjustmentTrigger struct {
	TriggerType   string                 `json:"trigger_type"`   // RISK_BASED, VOLATILITY_BASED, REGIME_BASED
	Severity      shared.Severity        `json:"severity"`
	Description   string                 `json:"description"`
	Positions     []string               `json:"positions"`     // Position IDs affected
	Adjustments   []StopLossAdjustment   `json:"adjustments"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// NewStopLossIntegration creates a new stop loss integration
func NewStopLossIntegration(cfg *config.Config, db *database.DB, accountManager *account.Manager, riskMonitor *RiskMonitor) *StopLossIntegration {
	// Create stop loss components
	adjuster := NewStopLossAdjuster(cfg, db, accountManager)
	executor := NewStopLossExecutor(adjuster, cfg, db, accountManager)
	
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

	// Default settings
	riskThresholds := StopLossRiskThresholds{
		LowRiskMultiplier:      0.8,
		MediumRiskMultiplier:   1.0,
		HighRiskMultiplier:     1.2,
		CriticalRiskMultiplier: 1.5,
		EmergencyMarginRatio:   0.9,
		EmergencyVaRLimit:      0.05,
	}

	volatilitySettings := VolatilityBasedSettings{
		LowVolatilityThreshold:  0.15,
		HighVolatilityThreshold: 0.35,
		LowVolatilityFactor:     0.8,
		HighVolatilityFactor:    1.3,
		BullMarketFactor:        0.9,
		BearMarketFactor:        1.1,
		VolatileMarketFactor:    1.2,
	}

	return &StopLossIntegration{
		adjuster:           adjuster,
		executor:           executor,
		riskMonitor:        riskMonitor,
		config:             cfg,
		db:                 db,
		accountManager:     accountManager,
		configManager:      configManager,
		errorHandler:       errorHandler,
		metrics:            make(map[string]interface{}),
		updateInterval:     5 * time.Minute, // Default 5-minute updates
		riskThresholds:     riskThresholds,
		volatilitySettings: volatilitySettings,
	}
}

// ProcessIntegratedStopLoss processes stop loss adjustments based on risk and market conditions
func (sli *StopLossIntegration) ProcessIntegratedStopLoss(ctx context.Context) (*IntegratedStopLossResult, error) {
	sli.mu.Lock()
	defer sli.mu.Unlock()

	startTime := time.Now()
	log.Printf("Starting integrated stop loss processing")

	result := &IntegratedStopLossResult{
		AdjustmentsTrigger: make([]AdjustmentTrigger, 0),
		Timestamp:          time.Now(),
	}

	// 1. Get current risk assessment
	riskAssessment, err := sli.riskMonitor.MonitorPositionRisk(ctx)
	if err != nil {
		return nil, shared.NewAutomationError(
			shared.ErrCodeRiskAssessmentFailed,
			fmt.Sprintf("Failed to get risk assessment: %v", err),
			"StopLossIntegration",
			shared.ErrorSeverityHigh,
			true,
		).WithContext("operation", "ProcessIntegratedStopLoss")
	}
	result.RiskAssessment = sli.convertToSharedRiskAssessment(riskAssessment)

	// 2. Get market regime
	marketRegime, err := sli.adjuster.MonitorMarketRegime(ctx)
	if err != nil {
		log.Printf("Warning: Failed to get market regime: %v", err)
		// Use default regime
		marketRegime = &shared.MarketRegime{
			Type:       "SIDEWAYS",
			Confidence: 0.5,
			Volatility: 0.2,
			Trend:      0.0,
			Momentum:   0.0,
			Timestamp:  time.Now(),
		}
	}
	result.MarketRegime = marketRegime

	// 3. Analyze triggers for stop loss adjustments
	triggers := sli.analyzeAdjustmentTriggers(ctx, riskAssessment, marketRegime)
	result.AdjustmentsTrigger = triggers

	// 4. Execute adjustments if triggers are found
	if len(triggers) > 0 {
		executionResult, err := sli.executeTriggeredAdjustments(ctx, triggers)
		if err != nil {
			log.Printf("Warning: Failed to execute triggered adjustments: %v", err)
		} else {
			result.ExecutionResult = executionResult
		}
	}

	// 5. Update performance metrics
	performanceMetrics := sli.collectPerformanceMetrics()
	result.PerformanceMetrics = performanceMetrics

	result.ProcessingTime = time.Since(startTime)

	// Update internal metrics
	sli.updateIntegrationMetrics(result)

	log.Printf("Integrated stop loss processing completed in %v: %d triggers, %d adjustments executed",
		result.ProcessingTime, len(triggers), 
		func() int {
			if result.ExecutionResult != nil {
				return result.ExecutionResult.TotalAdjustments
			}
			return 0
		}())

	return result, nil
}

// analyzeAdjustmentTriggers analyzes conditions to determine if stop loss adjustments are needed
func (sli *StopLossIntegration) analyzeAdjustmentTriggers(ctx context.Context, riskAssessment *PositionRiskReport, marketRegime *shared.MarketRegime) []AdjustmentTrigger {
	var triggers []AdjustmentTrigger

	// 1. Risk-based triggers
	riskTriggers := sli.analyzeRiskBasedTriggers(ctx, riskAssessment)
	triggers = append(triggers, riskTriggers...)

	// 2. Volatility-based triggers
	volatilityTriggers := sli.analyzeVolatilityBasedTriggers(ctx, marketRegime)
	triggers = append(triggers, volatilityTriggers...)

	// 3. Regime-based triggers
	regimeTriggers := sli.analyzeRegimeBasedTriggers(ctx, marketRegime)
	triggers = append(triggers, regimeTriggers...)

	// 4. Emergency triggers
	emergencyTriggers := sli.analyzeEmergencyTriggers(ctx, riskAssessment)
	triggers = append(triggers, emergencyTriggers...)

	return triggers
}

// analyzeRiskBasedTriggers analyzes risk-based conditions for stop loss adjustments
func (sli *StopLossIntegration) analyzeRiskBasedTriggers(ctx context.Context, riskAssessment *PositionRiskReport) []AdjustmentTrigger {
	var triggers []AdjustmentTrigger

	// Check overall portfolio risk
	if riskAssessment.TotalRisk > 0.8 { // High risk threshold
		trigger := AdjustmentTrigger{
			TriggerType: "RISK_BASED",
			Severity:    shared.SeverityWarning,
			Description: fmt.Sprintf("High portfolio risk detected: %.2f", riskAssessment.TotalRisk),
			Metadata: map[string]interface{}{
				"total_risk":         riskAssessment.TotalRisk,
				"concentration_risk": riskAssessment.ConcentrationRisk,
				"correlation_risk":   riskAssessment.CorrelationRisk,
			},
		}

		// Generate adjustments for high-risk positions
		adjustments := sli.generateRiskBasedAdjustments(ctx, riskAssessment, shared.RiskLevelHigh)
		trigger.Adjustments = adjustments
		trigger.Positions = sli.extractPositionIDs(adjustments)

		triggers = append(triggers, trigger)
	}

	// Check concentration risk
	if riskAssessment.ConcentrationRisk > 0.6 { // High concentration
		trigger := AdjustmentTrigger{
			TriggerType: "RISK_BASED",
			Severity:    shared.SeverityWarning,
			Description: fmt.Sprintf("High concentration risk detected: %.2f", riskAssessment.ConcentrationRisk),
			Metadata: map[string]interface{}{
				"concentration_risk": riskAssessment.ConcentrationRisk,
				"trigger_reason":     "concentration",
			},
		}

		// Generate tighter stop losses for concentrated positions
		adjustments := sli.generateConcentrationBasedAdjustments(ctx, riskAssessment)
		trigger.Adjustments = adjustments
		trigger.Positions = sli.extractPositionIDs(adjustments)

		triggers = append(triggers, trigger)
	}

	return triggers
}

// analyzeVolatilityBasedTriggers analyzes volatility-based conditions
func (sli *StopLossIntegration) analyzeVolatilityBasedTriggers(ctx context.Context, marketRegime *shared.MarketRegime) []AdjustmentTrigger {
	var triggers []AdjustmentTrigger

	// Check for high volatility
	if marketRegime.Volatility > sli.volatilitySettings.HighVolatilityThreshold {
		trigger := AdjustmentTrigger{
			TriggerType: "VOLATILITY_BASED",
			Severity:    shared.SeverityWarning,
			Description: fmt.Sprintf("High market volatility detected: %.2f", marketRegime.Volatility),
			Metadata: map[string]interface{}{
				"volatility":      marketRegime.Volatility,
				"threshold":       sli.volatilitySettings.HighVolatilityThreshold,
				"adjustment_factor": sli.volatilitySettings.HighVolatilityFactor,
			},
		}

		// Generate wider stop losses for high volatility
		adjustments := sli.generateVolatilityBasedAdjustments(ctx, marketRegime.Volatility, sli.volatilitySettings.HighVolatilityFactor)
		trigger.Adjustments = adjustments
		trigger.Positions = sli.extractPositionIDs(adjustments)

		triggers = append(triggers, trigger)
	}

	// Check for low volatility
	if marketRegime.Volatility < sli.volatilitySettings.LowVolatilityThreshold {
		trigger := AdjustmentTrigger{
			TriggerType: "VOLATILITY_BASED",
			Severity:    shared.SeverityInfo,
			Description: fmt.Sprintf("Low market volatility detected: %.2f", marketRegime.Volatility),
			Metadata: map[string]interface{}{
				"volatility":        marketRegime.Volatility,
				"threshold":         sli.volatilitySettings.LowVolatilityThreshold,
				"adjustment_factor": sli.volatilitySettings.LowVolatilityFactor,
			},
		}

		// Generate tighter stop losses for low volatility
		adjustments := sli.generateVolatilityBasedAdjustments(ctx, marketRegime.Volatility, sli.volatilitySettings.LowVolatilityFactor)
		trigger.Adjustments = adjustments
		trigger.Positions = sli.extractPositionIDs(adjustments)

		triggers = append(triggers, trigger)
	}

	return triggers
}

// analyzeRegimeBasedTriggers analyzes market regime-based conditions
func (sli *StopLossIntegration) analyzeRegimeBasedTriggers(ctx context.Context, marketRegime *shared.MarketRegime) []AdjustmentTrigger {
	var triggers []AdjustmentTrigger

	var adjustmentFactor float64
	var severity shared.Severity
	var description string

	switch marketRegime.Type {
	case "BULL":
		adjustmentFactor = sli.volatilitySettings.BullMarketFactor
		severity = shared.SeverityInfo
		description = "Bull market regime detected - adjusting stop losses"
	case "BEAR":
		adjustmentFactor = sli.volatilitySettings.BearMarketFactor
		severity = shared.SeverityWarning
		description = "Bear market regime detected - adjusting stop losses"
	case "VOLATILE":
		adjustmentFactor = sli.volatilitySettings.VolatileMarketFactor
		severity = shared.SeverityWarning
		description = "Volatile market regime detected - widening stop losses"
	default:
		return triggers // No adjustment needed for sideways market
	}

	trigger := AdjustmentTrigger{
		TriggerType: "REGIME_BASED",
		Severity:    severity,
		Description: description,
		Metadata: map[string]interface{}{
			"regime_type":       marketRegime.Type,
			"regime_confidence": marketRegime.Confidence,
			"adjustment_factor": adjustmentFactor,
		},
	}

	// Generate regime-based adjustments
	adjustments := sli.generateRegimeBasedAdjustments(ctx, marketRegime, adjustmentFactor)
	trigger.Adjustments = adjustments
	trigger.Positions = sli.extractPositionIDs(adjustments)

	if len(adjustments) > 0 {
		triggers = append(triggers, trigger)
	}

	return triggers
}

// analyzeEmergencyTriggers analyzes emergency conditions
func (sli *StopLossIntegration) analyzeEmergencyTriggers(ctx context.Context, riskAssessment *PositionRiskReport) []AdjustmentTrigger {
	var triggers []AdjustmentTrigger

	// Check for emergency margin conditions
	marginStatus, err := sli.riskMonitor.CheckMarginRatio(ctx)
	if err != nil {
		log.Printf("Warning: Failed to check margin ratio for emergency triggers: %v", err)
		return triggers
	}

	if marginStatus.MarginRatio > sli.riskThresholds.EmergencyMarginRatio {
		trigger := AdjustmentTrigger{
			TriggerType: "EMERGENCY",
			Severity:    shared.SeverityCritical,
			Description: fmt.Sprintf("Emergency margin condition: %.2f ratio", marginStatus.MarginRatio),
			Metadata: map[string]interface{}{
				"margin_ratio":     marginStatus.MarginRatio,
				"emergency_threshold": sli.riskThresholds.EmergencyMarginRatio,
				"total_equity":     marginStatus.TotalEquity,
				"used_margin":      marginStatus.UsedMargin,
			},
		}

		// Generate emergency stop loss adjustments (much tighter)
		adjustments := sli.generateEmergencyAdjustments(ctx, marginStatus)
		trigger.Adjustments = adjustments
		trigger.Positions = sli.extractPositionIDs(adjustments)

		triggers = append(triggers, trigger)
	}

	// Check for emergency VaR conditions
	if riskAssessment.VaR > sli.riskThresholds.EmergencyVaRLimit {
		trigger := AdjustmentTrigger{
			TriggerType: "EMERGENCY",
			Severity:    shared.SeverityCritical,
			Description: fmt.Sprintf("Emergency VaR condition: %.4f", riskAssessment.VaR),
			Metadata: map[string]interface{}{
				"portfolio_var":    riskAssessment.VaR,
				"emergency_limit":  sli.riskThresholds.EmergencyVaRLimit,
				"expected_shortfall": riskAssessment.ExpectedShortfall,
			},
		}

		// Generate emergency VaR-based adjustments
		adjustments := sli.generateVaRBasedEmergencyAdjustments(ctx, riskAssessment)
		trigger.Adjustments = adjustments
		trigger.Positions = sli.extractPositionIDs(adjustments)

		triggers = append(triggers, trigger)
	}

	return triggers
}

// Helper methods for generating adjustments would be implemented here...
// (generateRiskBasedAdjustments, generateVolatilityBasedAdjustments, etc.)

// executeTriggeredAdjustments executes adjustments from triggers
func (sli *StopLossIntegration) executeTriggeredAdjustments(ctx context.Context, triggers []AdjustmentTrigger) (*ExecutionResult, error) {
	// Collect all adjustments from triggers
	var allAdjustments []StopLossAdjustment
	for _, trigger := range triggers {
		allAdjustments = append(allAdjustments, trigger.Adjustments...)
	}

	if len(allAdjustments) == 0 {
		return &ExecutionResult{
			TotalAdjustments:      0,
			SuccessfulAdjustments: 0,
			FailedAdjustments:     0,
			ExecutionTime:         0,
			Errors:                []string{},
			AdjustmentDetails:     []AdjustmentDetail{},
			Timestamp:             time.Now(),
		}, nil
	}

	// Execute adjustments
	return sli.executor.ExecuteStopLossAdjustments(ctx)
}

// collectPerformanceMetrics collects performance metrics from all components
func (sli *StopLossIntegration) collectPerformanceMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	// Get adjuster metrics
	adjusterMetrics := sli.adjuster.GetMetrics()
	for k, v := range adjusterMetrics {
		metrics["adjuster_"+k] = v
	}

	// Get executor metrics
	executorMetrics := sli.executor.GetMetrics()
	for k, v := range executorMetrics {
		metrics["executor_"+k] = v
	}

	// Get risk monitor metrics
	riskMetrics := sli.riskMonitor.GetMetrics()
	for k, v := range riskMetrics {
		metrics["risk_"+k] = v
	}

	return metrics
}

// Helper methods

// convertToSharedRiskAssessment converts PositionRiskReport to shared.RiskAssessment
func (sli *StopLossIntegration) convertToSharedRiskAssessment(report *PositionRiskReport) *shared.RiskAssessment {
	// Determine overall risk level
	var overallRisk shared.RiskLevel
	if report.TotalRisk > 0.8 {
		overallRisk = shared.RiskLevelCritical
	} else if report.TotalRisk > 0.6 {
		overallRisk = shared.RiskLevelHigh
	} else if report.TotalRisk > 0.4 {
		overallRisk = shared.RiskLevelMedium
	} else {
		overallRisk = shared.RiskLevelLow
	}

	return &shared.RiskAssessment{
		OverallRisk:       overallRisk,
		PortfolioVaR:      report.VaR,
		MaxDrawdown:       report.MaxDrawdown,
		ConcentrationRisk: report.ConcentrationRisk,
		LiquidityRisk:     report.LiquidityRisk,
		CorrelationRisk:   report.CorrelationRisk,
		Recommendations:   report.Recommendations,
		Metrics: map[string]interface{}{
			"total_risk":         report.TotalRisk,
			"expected_shortfall": report.ExpectedShortfall,
			"position_count":     len(report.Positions),
		},
		Timestamp: report.Timestamp,
	}
}

// extractPositionIDs extracts position IDs from adjustments
func (sli *StopLossIntegration) extractPositionIDs(adjustments []StopLossAdjustment) []string {
	positionIDs := make([]string, len(adjustments))
	for i, adj := range adjustments {
		positionIDs[i] = adj.PositionID
	}
	return positionIDs
}

// updateIntegrationMetrics updates internal integration metrics
func (sli *StopLossIntegration) updateIntegrationMetrics(result *IntegratedStopLossResult) {
	sli.metrics["last_processing_time"] = result.ProcessingTime
	sli.metrics["last_triggers_count"] = len(result.AdjustmentsTrigger)
	sli.metrics["last_market_regime"] = result.MarketRegime.Type
	sli.metrics["last_market_volatility"] = result.MarketRegime.Volatility
	sli.metrics["last_overall_risk"] = result.RiskAssessment.OverallRisk.String()
	sli.metrics["last_portfolio_var"] = result.RiskAssessment.PortfolioVaR
	sli.metrics["last_update"] = result.Timestamp

	if result.ExecutionResult != nil {
		sli.metrics["last_adjustments_executed"] = result.ExecutionResult.TotalAdjustments
		sli.metrics["last_execution_success_rate"] = float64(result.ExecutionResult.SuccessfulAdjustments) / float64(result.ExecutionResult.TotalAdjustments)
	}
}

// GetMetrics returns current integration metrics
func (sli *StopLossIntegration) GetMetrics() map[string]interface{} {
	sli.mu.RLock()
	defer sli.mu.RUnlock()
	
	// Return a copy to prevent external modifications
	metrics := make(map[string]interface{})
	for k, v := range sli.metrics {
		metrics[k] = v
	}
	return metrics
}

// IsRunning returns whether the integration is currently running
func (sli *StopLossIntegration) IsRunning() bool {
	sli.mu.RLock()
	defer sli.mu.RUnlock()
	return sli.isRunning
}

// Start starts the stop loss integration
func (sli *StopLossIntegration) Start() error {
	sli.mu.Lock()
	defer sli.mu.Unlock()
	
	// Start all components
	if err := sli.adjuster.Start(); err != nil {
		return fmt.Errorf("failed to start adjuster: %w", err)
	}
	
	if err := sli.executor.Start(); err != nil {
		return fmt.Errorf("failed to start executor: %w", err)
	}
	
	sli.isRunning = true
	sli.lastUpdate = time.Now()
	log.Printf("Stop loss integration started")
	return nil
}

// Stop stops the stop loss integration
func (sli *StopLossIntegration) Stop() error {
	sli.mu.Lock()
	defer sli.mu.Unlock()
	
	// Stop all components
	if err := sli.adjuster.Stop(); err != nil {
		log.Printf("Warning: Failed to stop adjuster: %v", err)
	}
	
	if err := sli.executor.Stop(); err != nil {
		log.Printf("Warning: Failed to stop executor: %v", err)
	}
	
	sli.isRunning = false
	log.Printf("Stop loss integration stopped")
	return nil
}

// generateRiskBasedAdjustments generates stop loss adjustments based on risk levels
func (sli *StopLossIntegration) generateRiskBasedAdjustments(ctx context.Context, riskAssessment *PositionRiskReport, riskLevel shared.RiskLevel) []StopLossAdjustment {
	var adjustments []StopLossAdjustment
	
	// Get risk multiplier based on risk level
	var multiplier float64
	switch riskLevel {
	case shared.RiskLevelLow:
		multiplier = sli.riskThresholds.LowRiskMultiplier
	case shared.RiskLevelMedium:
		multiplier = sli.riskThresholds.MediumRiskMultiplier
	case shared.RiskLevelHigh:
		multiplier = sli.riskThresholds.HighRiskMultiplier
	case shared.RiskLevelCritical:
		multiplier = sli.riskThresholds.CriticalRiskMultiplier
	default:
		multiplier = sli.riskThresholds.MediumRiskMultiplier
	}

	// Generate adjustments for high-risk positions
	for _, positionRisk := range riskAssessment.Positions {
		if positionRisk.VaR > 0.02 { // Position VaR > 2%
			// Calculate new stop loss level with risk-based multiplier
			currentStopLoss, err := sli.getCurrentStopLoss(ctx, positionRisk.Position.ID)
			if err != nil {
				log.Printf("Warning: Failed to get current stop loss for position %s: %v", positionRisk.Position.ID, err)
				continue
			}

			// Calculate optimal stop loss with risk adjustment
			optimalStopLoss, err := sli.adjuster.CalculateOptimalStopLoss(ctx, positionRisk.Position)
			if err != nil {
				log.Printf("Warning: Failed to calculate optimal stop loss for position %s: %v", positionRisk.Position.ID, err)
				continue
			}

			// Apply risk-based multiplier
			adjustedStopLoss := sli.applyRiskMultiplier(optimalStopLoss, multiplier, positionRisk.Position.Side)

			if sli.shouldAdjustStopLoss(currentStopLoss, adjustedStopLoss, positionRisk.Position.Side) {
				adjustment := StopLossAdjustment{
					PositionID:     positionRisk.Position.ID,
					Symbol:         positionRisk.Position.Symbol,
					OldLevel:       currentStopLoss,
					NewLevel:       adjustedStopLoss,
					AdjustmentType: "RISK_BASED",
					Reason:         fmt.Sprintf("Risk-based adjustment for %s risk level (VaR: %.4f)", riskLevel.String(), positionRisk.VaR),
					Priority:       sli.calculateRiskBasedPriority(positionRisk, riskLevel),
					Timestamp:      time.Now(),
				}
				adjustments = append(adjustments, adjustment)
			}
		}
	}

	return adjustments
}

// generateConcentrationBasedAdjustments generates adjustments for concentrated positions
func (sli *StopLossIntegration) generateConcentrationBasedAdjustments(ctx context.Context, riskAssessment *PositionRiskReport) []StopLossAdjustment {
	var adjustments []StopLossAdjustment

	// Find positions with high concentration risk
	for _, positionRisk := range riskAssessment.Positions {
		if positionRisk.ConcentrationRisk > 0.3 { // Position represents >30% of portfolio
			currentStopLoss, err := sli.getCurrentStopLoss(ctx, positionRisk.Position.ID)
			if err != nil {
				continue
			}

			// Tighter stop loss for concentrated positions
			optimalStopLoss, err := sli.adjuster.CalculateOptimalStopLoss(ctx, positionRisk.Position)
			if err != nil {
				continue
			}

			// Apply concentration penalty (tighter stops)
			concentrationMultiplier := 0.8 // 20% tighter
			adjustedStopLoss := sli.applyRiskMultiplier(optimalStopLoss, concentrationMultiplier, positionRisk.Position.Side)

			if sli.shouldAdjustStopLoss(currentStopLoss, adjustedStopLoss, positionRisk.Position.Side) {
				adjustment := StopLossAdjustment{
					PositionID:     positionRisk.Position.ID,
					Symbol:         positionRisk.Position.Symbol,
					OldLevel:       currentStopLoss,
					NewLevel:       adjustedStopLoss,
					AdjustmentType: "CONCENTRATION_BASED",
					Reason:         fmt.Sprintf("Concentration risk adjustment (%.2f concentration)", positionRisk.ConcentrationRisk),
					Priority:       8, // High priority for concentration risk
					Timestamp:      time.Now(),
				}
				adjustments = append(adjustments, adjustment)
			}
		}
	}

	return adjustments
}

// generateVolatilityBasedAdjustments generates adjustments based on market volatility
func (sli *StopLossIntegration) generateVolatilityBasedAdjustments(ctx context.Context, volatility, adjustmentFactor float64) []StopLossAdjustment {
	var adjustments []StopLossAdjustment

	// Get all active positions
	positions, err := sli.getAllActivePositions(ctx)
	if err != nil {
		log.Printf("Warning: Failed to get active positions for volatility adjustments: %v", err)
		return adjustments
	}

	for _, position := range positions {
		currentStopLoss, err := sli.getCurrentStopLoss(ctx, position.ID)
		if err != nil {
			continue
		}

		// Calculate volatility-adjusted stop loss
		optimalStopLoss, err := sli.adjuster.CalculateOptimalStopLoss(ctx, position)
		if err != nil {
			continue
		}

		// Apply volatility-based adjustment
		adjustedStopLoss := sli.applyRiskMultiplier(optimalStopLoss, adjustmentFactor, position.Side)

		if sli.shouldAdjustStopLoss(currentStopLoss, adjustedStopLoss, position.Side) {
			adjustment := StopLossAdjustment{
				PositionID:     position.ID,
				Symbol:         position.Symbol,
				OldLevel:       currentStopLoss,
				NewLevel:       adjustedStopLoss,
				AdjustmentType: "VOLATILITY_BASED",
				Reason:         fmt.Sprintf("Volatility-based adjustment (%.2f volatility, %.2f factor)", volatility, adjustmentFactor),
				Priority:       5, // Medium priority
				Timestamp:      time.Now(),
			}
			adjustments = append(adjustments, adjustment)
		}
	}

	return adjustments
}

// generateRegimeBasedAdjustments generates adjustments based on market regime
func (sli *StopLossIntegration) generateRegimeBasedAdjustments(ctx context.Context, marketRegime *shared.MarketRegime, adjustmentFactor float64) []StopLossAdjustment {
	var adjustments []StopLossAdjustment

	// Only adjust if regime confidence is high enough
	if marketRegime.Confidence < 0.6 {
		return adjustments
	}

	// Get all active positions
	positions, err := sli.getAllActivePositions(ctx)
	if err != nil {
		log.Printf("Warning: Failed to get active positions for regime adjustments: %v", err)
		return adjustments
	}

	for _, position := range positions {
		currentStopLoss, err := sli.getCurrentStopLoss(ctx, position.ID)
		if err != nil {
			continue
		}

		// Calculate regime-adjusted stop loss
		optimalStopLoss, err := sli.adjuster.CalculateOptimalStopLoss(ctx, position)
		if err != nil {
			continue
		}

		// Apply regime-based adjustment
		adjustedStopLoss := sli.applyRiskMultiplier(optimalStopLoss, adjustmentFactor, position.Side)

		if sli.shouldAdjustStopLoss(currentStopLoss, adjustedStopLoss, position.Side) {
			adjustment := StopLossAdjustment{
				PositionID:     position.ID,
				Symbol:         position.Symbol,
				OldLevel:       currentStopLoss,
				NewLevel:       adjustedStopLoss,
				AdjustmentType: "REGIME_BASED",
				Reason:         fmt.Sprintf("Market regime adjustment (%s regime, %.2f confidence)", marketRegime.Type, marketRegime.Confidence),
				Priority:       4, // Medium-low priority
				Timestamp:      time.Now(),
			}
			adjustments = append(adjustments, adjustment)
		}
	}

	return adjustments
}

// generateEmergencyAdjustments generates emergency stop loss adjustments
func (sli *StopLossIntegration) generateEmergencyAdjustments(ctx context.Context, marginStatus *MarginStatus) []StopLossAdjustment {
	var adjustments []StopLossAdjustment

	// Get all active positions
	positions, err := sli.getAllActivePositions(ctx)
	if err != nil {
		log.Printf("Warning: Failed to get active positions for emergency adjustments: %v", err)
		return adjustments
	}

	// Emergency multiplier - much tighter stops
	emergencyMultiplier := 0.5 // 50% tighter stops

	for _, position := range positions {
		currentStopLoss, err := sli.getCurrentStopLoss(ctx, position.ID)
		if err != nil {
			continue
		}

		// Calculate emergency stop loss (much tighter)
		optimalStopLoss, err := sli.adjuster.CalculateOptimalStopLoss(ctx, position)
		if err != nil {
			continue
		}

		// Apply emergency multiplier
		emergencyStopLoss := sli.applyRiskMultiplier(optimalStopLoss, emergencyMultiplier, position.Side)

		// Always adjust in emergency conditions
		adjustment := StopLossAdjustment{
			PositionID:     position.ID,
			Symbol:         position.Symbol,
			OldLevel:       currentStopLoss,
			NewLevel:       emergencyStopLoss,
			AdjustmentType: "EMERGENCY",
			Reason:         fmt.Sprintf("Emergency margin adjustment (%.2f margin ratio)", marginStatus.MarginRatio),
			Priority:       10, // Highest priority
			Timestamp:      time.Now(),
		}
		adjustments = append(adjustments, adjustment)
	}

	return adjustments
}

// generateVaRBasedEmergencyAdjustments generates emergency adjustments based on VaR
func (sli *StopLossIntegration) generateVaRBasedEmergencyAdjustments(ctx context.Context, riskAssessment *PositionRiskReport) []StopLossAdjustment {
	var adjustments []StopLossAdjustment

	// Sort positions by VaR (highest first)
	sortedPositions := make([]shared.PositionRisk, len(riskAssessment.Positions))
	copy(sortedPositions, riskAssessment.Positions)
	
	// Simple bubble sort by VaR
	for i := 0; i < len(sortedPositions)-1; i++ {
		for j := 0; j < len(sortedPositions)-i-1; j++ {
			if sortedPositions[j].VaR < sortedPositions[j+1].VaR {
				sortedPositions[j], sortedPositions[j+1] = sortedPositions[j+1], sortedPositions[j]
			}
		}
	}

	// Adjust highest VaR positions first
	for i, positionRisk := range sortedPositions {
		if i >= 5 { // Only adjust top 5 highest VaR positions
			break
		}

		currentStopLoss, err := sli.getCurrentStopLoss(ctx, positionRisk.Position.ID)
		if err != nil {
			continue
		}

		// Calculate VaR-based emergency stop loss
		optimalStopLoss, err := sli.adjuster.CalculateOptimalStopLoss(ctx, positionRisk.Position)
		if err != nil {
			continue
		}

		// Apply VaR-based emergency multiplier (tighter for higher VaR)
		varMultiplier := 0.6 - (positionRisk.VaR * 2) // Tighter stops for higher VaR
		if varMultiplier < 0.3 {
			varMultiplier = 0.3 // Minimum 30% of optimal
		}

		emergencyStopLoss := sli.applyRiskMultiplier(optimalStopLoss, varMultiplier, positionRisk.Position.Side)

		adjustment := StopLossAdjustment{
			PositionID:     positionRisk.Position.ID,
			Symbol:         positionRisk.Position.Symbol,
			OldLevel:       currentStopLoss,
			NewLevel:       emergencyStopLoss,
			AdjustmentType: "EMERGENCY_VAR",
			Reason:         fmt.Sprintf("Emergency VaR adjustment (%.4f VaR)", positionRisk.VaR),
			Priority:       10, // Highest priority
			Timestamp:      time.Now(),
		}
		adjustments = append(adjustments, adjustment)
	}

	return adjustments
}

// Helper methods

// getCurrentStopLoss gets current stop loss level for a position
func (sli *StopLossIntegration) getCurrentStopLoss(ctx context.Context, positionID string) (float64, error) {
	query := `SELECT COALESCE(stop_loss, 0) FROM positions WHERE id = ? AND status = 'ACTIVE'`
	
	var stopLoss float64
	err := sli.db.QueryRowContext(ctx, query, positionID).Scan(&stopLoss)
	if err != nil {
		return 0, err
	}

	return stopLoss, nil
}

// getAllActivePositions gets all active positions
func (sli *StopLossIntegration) getAllActivePositions(ctx context.Context) ([]shared.Position, error) {
	query := `
		SELECT 
			id, symbol, side, size, entry_price, current_price,
			unrealized_pnl, realized_pnl, leverage, margin_used, created_at
		FROM positions 
		WHERE status = 'ACTIVE'
		ORDER BY created_at DESC
	`
	
	rows, err := sli.db.QueryContext(ctx, query)
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

// applyRiskMultiplier applies a risk multiplier to a stop loss level
func (sli *StopLossIntegration) applyRiskMultiplier(stopLoss, multiplier float64, side string) float64 {
	if side == "LONG" {
		// For long positions, multiplier > 1 means wider stops (lower level)
		// multiplier < 1 means tighter stops (higher level)
		distance := stopLoss * (1 - multiplier) * 0.1 // 10% base distance
		if multiplier > 1 {
			return stopLoss - distance // Wider stop
		} else {
			return stopLoss + distance // Tighter stop
		}
	} else {
		// For short positions, opposite logic
		distance := stopLoss * (1 - multiplier) * 0.1
		if multiplier > 1 {
			return stopLoss + distance // Wider stop
		} else {
			return stopLoss - distance // Tighter stop
		}
	}
}

// shouldAdjustStopLoss determines if stop loss should be adjusted
func (sli *StopLossIntegration) shouldAdjustStopLoss(current, new float64, side string) bool {
	if current == 0 {
		return true // No stop loss set
	}

	// Calculate percentage difference
	var percentDiff float64
	if side == "LONG" {
		percentDiff = (new - current) / current
	} else {
		percentDiff = (current - new) / current
	}

	// Adjust if difference is more than 3%
	return math.Abs(percentDiff) > 0.03
}

// calculateRiskBasedPriority calculates priority for risk-based adjustments
func (sli *StopLossIntegration) calculateRiskBasedPriority(positionRisk shared.PositionRisk, riskLevel shared.RiskLevel) int {
	basePriority := 5

	// Adjust based on risk level
	switch riskLevel {
	case shared.RiskLevelCritical:
		basePriority = 9
	case shared.RiskLevelHigh:
		basePriority = 7
	case shared.RiskLevelMedium:
		basePriority = 5
	case shared.RiskLevelLow:
		basePriority = 3
	}

	// Adjust based on position VaR
	if positionRisk.VaR > 0.05 {
		basePriority += 2
	} else if positionRisk.VaR > 0.03 {
		basePriority += 1
	}

	// Adjust based on position size
	positionValue := positionRisk.Position.Size * positionRisk.Position.CurrentPrice
	if positionValue > 10000 {
		basePriority += 1
	}

	return basePriority
}

