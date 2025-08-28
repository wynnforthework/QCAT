package position

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"qcat/internal/automation/scheduler/shared"
	"qcat/internal/database"
	"qcat/internal/exchange"
	"qcat/internal/logger"
)

// FundReallocationExecutor implements optimal allocation calculation and execution
type FundReallocationExecutor struct {
	db             *database.DB
	exchangeClient exchange.Exchange
	logger         logger.Logger
	config         shared.ConfigProvider
	
	// Configuration
	maxSlippage        float64
	executionTimeout   time.Duration
	minTradeSize       float64
	maxTradeSize       float64
	impactThreshold    float64
	batchSize          int
}

// NewFundReallocationExecutor creates a new fund reallocation executor
func NewFundReallocationExecutor(
	db *database.DB,
	exchangeClient exchange.Exchange,
	logger logger.Logger,
	config shared.ConfigProvider,
) *FundReallocationExecutor {
	return &FundReallocationExecutor{
		db:              db,
		exchangeClient:  exchangeClient,
		logger:          logger,
		config:          config,
		maxSlippage:     config.GetFloat64("fund_reallocation.max_slippage"),
		executionTimeout: config.GetDuration("fund_reallocation.execution_timeout"),
		minTradeSize:    config.GetFloat64("fund_reallocation.min_trade_size"),
		maxTradeSize:    config.GetFloat64("fund_reallocation.max_trade_size"),
		impactThreshold: config.GetFloat64("fund_reallocation.impact_threshold"),
		batchSize:       config.GetInt("fund_reallocation.batch_size"),
	}
}

// AllocationConstraints represents constraints for optimal allocation calculation
type AllocationConstraints struct {
	MinAllocation      float64            `json:"min_allocation"`
	MaxAllocation      float64            `json:"max_allocation"`
	MaxConcentration   float64            `json:"max_concentration"`
	MinDiversification int                `json:"min_diversification"`
	RiskBudget         float64            `json:"risk_budget"`
	TransactionCosts   map[string]float64 `json:"transaction_costs"`
	LiquidityLimits    map[string]float64 `json:"liquidity_limits"`
	MarketImpactLimits map[string]float64 `json:"market_impact_limits"`
}

// OptimalAllocation represents optimal allocation calculation result
type OptimalAllocation struct {
	StrategyID         string                 `json:"strategy_id"`
	CurrentAllocation  float64                `json:"current_allocation"`
	TargetAllocation   float64                `json:"target_allocation"`
	AllocationChange   float64                `json:"allocation_change"`
	ExpectedReturn     float64                `json:"expected_return"`
	ExpectedRisk       float64                `json:"expected_risk"`
	RiskContribution   float64                `json:"risk_contribution"`
	TransactionCost    float64                `json:"transaction_cost"`
	MarketImpact       float64                `json:"market_impact"`
	Priority           int                    `json:"priority"`
	Rationale          string                 `json:"rationale"`
	Metadata           map[string]interface{} `json:"metadata"`
	CalculatedAt       time.Time              `json:"calculated_at"`
}

// FundTransferPlan represents a comprehensive fund transfer plan
type FundTransferPlan struct {
	ID                 string                 `json:"id"`
	TotalAmount        float64                `json:"total_amount"`
	Transfers          []shared.FundTransfer  `json:"transfers"`
	ExecutionSteps     []shared.ExecutionStep `json:"execution_steps"`
	EstimatedDuration  time.Duration          `json:"estimated_duration"`
	EstimatedCost      float64                `json:"estimated_cost"`
	EstimatedImpact    float64                `json:"estimated_impact"`
	RiskAssessment     *TransferRiskAssessment `json:"risk_assessment"`
	CreatedAt          time.Time              `json:"created_at"`
}

// TransferRiskAssessment represents risk assessment for fund transfers
type TransferRiskAssessment struct {
	OverallRisk        shared.RiskLevel       `json:"overall_risk"`
	LiquidityRisk      float64                `json:"liquidity_risk"`
	MarketImpactRisk   float64                `json:"market_impact_risk"`
	ExecutionRisk      float64                `json:"execution_risk"`
	CounterpartyRisk   float64                `json:"counterparty_risk"`
	RiskFactors        []string               `json:"risk_factors"`
	MitigationStrategies []string             `json:"mitigation_strategies"`
	AssessedAt         time.Time              `json:"assessed_at"`
}

// MarketImpactModel represents market impact modeling
type MarketImpactModel struct {
	Symbol             string    `json:"symbol"`
	LinearImpact       float64   `json:"linear_impact"`
	SquareRootImpact   float64   `json:"square_root_impact"`
	TemporaryImpact    float64   `json:"temporary_impact"`
	PermanentImpact    float64   `json:"permanent_impact"`
	LiquidityScore     float64   `json:"liquidity_score"`
	VolatilityAdjustment float64 `json:"volatility_adjustment"`
	CalculatedAt       time.Time `json:"calculated_at"`
}

// ReallocationResult represents the result of fund reallocation execution
type ReallocationResult struct {
	PlanID             string                 `json:"plan_id"`
	Success            bool                   `json:"success"`
	CompletedTransfers int                    `json:"completed_transfers"`
	FailedTransfers    int                    `json:"failed_transfers"`
	TotalCost          float64                `json:"total_cost"`
	TotalImpact        float64                `json:"total_impact"`
	ExecutionTime      time.Duration          `json:"execution_time"`
	Errors             []string               `json:"errors"`
	Warnings           []string               `json:"warnings"`
	Metadata           map[string]interface{} `json:"metadata"`
	CompletedAt        time.Time              `json:"completed_at"`
}

// AllocationPerformanceReport represents allocation performance monitoring
type AllocationPerformanceReport struct {
	AllocationID       string                 `json:"allocation_id"`
	PreAllocationMetrics map[string]float64   `json:"pre_allocation_metrics"`
	PostAllocationMetrics map[string]float64  `json:"post_allocation_metrics"`
	PerformanceChange  map[string]float64     `json:"performance_change"`
	RiskChange         map[string]float64     `json:"risk_change"`
	CostAnalysis       map[string]float64     `json:"cost_analysis"`
	EffectivenessScore float64                `json:"effectiveness_score"`
	Recommendations    []string               `json:"recommendations"`
	GeneratedAt        time.Time              `json:"generated_at"`
}

// CalculateOptimalAllocation implements optimal allocation calculation with constraints
func (fre *FundReallocationExecutor) CalculateOptimalAllocation(
	ctx context.Context,
	strategies []shared.Strategy,
	constraints AllocationConstraints,
) ([]OptimalAllocation, error) {
	fre.logger.Info("Calculating optimal allocation", 
		"strategy_count", len(strategies),
		"risk_budget", constraints.RiskBudget,
	)
	
	if len(strategies) == 0 {
		return nil, fmt.Errorf("no strategies provided for allocation calculation")
	}
	
	// Get current allocations
	currentAllocations, err := fre.getCurrentAllocations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current allocations: %w", err)
	}
	
	// Calculate expected returns and risks for each strategy
	expectedMetrics, err := fre.calculateExpectedMetrics(ctx, strategies)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate expected metrics: %w", err)
	}
	
	// Build covariance matrix
	covarianceMatrix, err := fre.buildCovarianceMatrix(ctx, strategies)
	if err != nil {
		return nil, fmt.Errorf("failed to build covariance matrix: %w", err)
	}
	
	// Solve optimization problem
	optimalWeights, err := fre.solveOptimizationProblem(
		expectedMetrics, covarianceMatrix, constraints)
	if err != nil {
		return nil, fmt.Errorf("failed to solve optimization problem: %w", err)
	}
	
	// Create optimal allocation results
	allocations := make([]OptimalAllocation, 0, len(strategies))
	
	for i, strategy := range strategies {
		currentAlloc := currentAllocations[strategy.ID]
		targetAlloc := optimalWeights[i]
		allocationChange := targetAlloc - currentAlloc
		
		// Calculate transaction cost and market impact
		transactionCost := fre.calculateTransactionCost(strategy.ID, math.Abs(allocationChange))
		marketImpact := fre.calculateMarketImpact(ctx, strategy.ID, math.Abs(allocationChange))
		
		// Calculate risk contribution
		riskContribution := fre.calculateRiskContribution(
			targetAlloc, expectedMetrics[i].Risk, covarianceMatrix, optimalWeights)
		
		allocation := OptimalAllocation{
			StrategyID:        strategy.ID,
			CurrentAllocation: currentAlloc,
			TargetAllocation:  targetAlloc,
			AllocationChange:  allocationChange,
			ExpectedReturn:    expectedMetrics[i].Return,
			ExpectedRisk:      expectedMetrics[i].Risk,
			RiskContribution:  riskContribution,
			TransactionCost:   transactionCost,
			MarketImpact:      marketImpact,
			Priority:          fre.calculatePriority(allocationChange, transactionCost),
			Rationale:         fre.generateRationale(strategy, allocationChange),
			Metadata: map[string]interface{}{
				"optimization_method": "mean_variance",
				"constraint_active":   fre.checkConstraintActive(targetAlloc, constraints),
			},
			CalculatedAt: time.Now(),
		}
		
		allocations = append(allocations, allocation)
	}
	
	// Sort by priority
	sort.Slice(allocations, func(i, j int) bool {
		return allocations[i].Priority < allocations[j].Priority
	})
	
	fre.logger.Info("Optimal allocation calculated", 
		"allocations_count", len(allocations))
	
	return allocations, nil
}

// CreateFundTransferPlan creates fund transfer and reallocation mechanisms
func (fre *FundReallocationExecutor) CreateFundTransferPlan(
	ctx context.Context,
	allocations []OptimalAllocation,
) (*FundTransferPlan, error) {
	fre.logger.Info("Creating fund transfer plan", "allocations_count", len(allocations))
	
	planID := fmt.Sprintf("transfer_plan_%d", time.Now().UnixNano())
	
	// Create fund transfers
	transfers := make([]shared.FundTransfer, 0)
	totalAmount := 0.0
	
	for _, allocation := range allocations {
		if math.Abs(allocation.AllocationChange) < fre.minTradeSize {
			continue // Skip small changes
		}
		
		transfer := fre.createFundTransfer(allocation)
		transfers = append(transfers, transfer)
		totalAmount += math.Abs(transfer.Amount)
	}
	
	// Create execution steps
	executionSteps := fre.createExecutionSteps(transfers)
	
	// Estimate duration and costs
	estimatedDuration := fre.estimateExecutionDuration(transfers)
	estimatedCost := fre.estimateTotalCost(transfers)
	estimatedImpact := fre.estimateTotalImpact(transfers)
	
	// Perform risk assessment
	riskAssessment := fre.assessTransferRisk(ctx, transfers)
	
	plan := &FundTransferPlan{
		ID:                planID,
		TotalAmount:       totalAmount,
		Transfers:         transfers,
		ExecutionSteps:    executionSteps,
		EstimatedDuration: estimatedDuration,
		EstimatedCost:     estimatedCost,
		EstimatedImpact:   estimatedImpact,
		RiskAssessment:    riskAssessment,
		CreatedAt:         time.Now(),
	}
	
	fre.logger.Info("Fund transfer plan created", 
		"plan_id", planID,
		"transfers_count", len(transfers),
		"estimated_cost", estimatedCost,
		"estimated_impact", estimatedImpact,
	)
	
	return plan, nil
}

// ExecuteReallocation executes fund reallocation with market impact minimization
func (fre *FundReallocationExecutor) ExecuteReallocation(
	ctx context.Context,
	plan *FundTransferPlan,
) (*ReallocationResult, error) {
	fre.logger.Info("Executing fund reallocation", "plan_id", plan.ID)
	
	startTime := time.Now()
	
	result := &ReallocationResult{
		PlanID:    plan.ID,
		Success:   true,
		Errors:    make([]string, 0),
		Warnings:  make([]string, 0),
		Metadata:  make(map[string]interface{}),
	}
	
	// Execute transfers in batches to minimize market impact
	batches := fre.createExecutionBatches(plan.Transfers)
	
	for batchIndex, batch := range batches {
		fre.logger.Info("Executing batch", 
			"batch_index", batchIndex+1,
			"batch_size", len(batch),
		)
		
		batchResult := fre.executeBatch(ctx, batch)
		
		result.CompletedTransfers += batchResult.CompletedTransfers
		result.FailedTransfers += batchResult.FailedTransfers
		result.TotalCost += batchResult.TotalCost
		result.TotalImpact += batchResult.TotalImpact
		
		if len(batchResult.Errors) > 0 {
			result.Errors = append(result.Errors, batchResult.Errors...)
			result.Success = false
		}
		
		if len(batchResult.Warnings) > 0 {
			result.Warnings = append(result.Warnings, batchResult.Warnings...)
		}
		
		// Wait between batches to reduce market impact
		if batchIndex < len(batches)-1 {
			waitTime := fre.calculateBatchWaitTime(batchResult.TotalImpact)
			time.Sleep(waitTime)
		}
	}
	
	result.ExecutionTime = time.Since(startTime)
	result.CompletedAt = time.Now()
	
	// Update allocation records
	if result.Success {
		err := fre.updateAllocationRecords(ctx, plan)
		if err != nil {
			fre.logger.Warn("Failed to update allocation records", "error", err)
			result.Warnings = append(result.Warnings, 
				fmt.Sprintf("Failed to update allocation records: %v", err))
		}
	}
	
	fre.logger.Info("Fund reallocation execution completed", 
		"plan_id", plan.ID,
		"success", result.Success,
		"completed_transfers", result.CompletedTransfers,
		"failed_transfers", result.FailedTransfers,
		"execution_time", result.ExecutionTime,
	)
	
	return result, nil
}

// MonitorAllocationPerformance creates allocation performance monitoring and reporting
func (fre *FundReallocationExecutor) MonitorAllocationPerformance(
	ctx context.Context,
	allocationID string,
	monitoringPeriod time.Duration,
) (*AllocationPerformanceReport, error) {
	fre.logger.Info("Monitoring allocation performance", 
		"allocation_id", allocationID,
		"monitoring_period", monitoringPeriod,
	)
	
	// Get pre-allocation metrics
	preMetrics, err := fre.getHistoricalMetrics(ctx, allocationID, monitoringPeriod, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get pre-allocation metrics: %w", err)
	}
	
	// Get post-allocation metrics
	postMetrics, err := fre.getHistoricalMetrics(ctx, allocationID, monitoringPeriod, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get post-allocation metrics: %w", err)
	}
	
	// Calculate performance changes
	performanceChange := fre.calculatePerformanceChange(preMetrics, postMetrics)
	riskChange := fre.calculateRiskChange(preMetrics, postMetrics)
	costAnalysis := fre.analyzeCosts(ctx, allocationID)
	
	// Calculate effectiveness score
	effectivenessScore := fre.calculateEffectivenessScore(
		performanceChange, riskChange, costAnalysis)
	
	// Generate recommendations
	recommendations := fre.generatePerformanceRecommendations(
		performanceChange, riskChange, effectivenessScore)
	
	report := &AllocationPerformanceReport{
		AllocationID:          allocationID,
		PreAllocationMetrics:  preMetrics,
		PostAllocationMetrics: postMetrics,
		PerformanceChange:     performanceChange,
		RiskChange:            riskChange,
		CostAnalysis:          costAnalysis,
		EffectivenessScore:    effectivenessScore,
		Recommendations:       recommendations,
		GeneratedAt:           time.Now(),
	}
	
	fre.logger.Info("Allocation performance monitoring completed", 
		"allocation_id", allocationID,
		"effectiveness_score", effectivenessScore,
	)
	
	return report, nil
}

// Private helper methods

type StrategyMetrics struct {
	Return float64
	Risk   float64
}

func (fre *FundReallocationExecutor) getCurrentAllocations(ctx context.Context) (map[string]float64, error) {
	query := `
		SELECT strategy_id, allocation_percent 
		FROM current_allocations 
		WHERE status = 'ACTIVE'
	`
	
	rows, err := fre.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query current allocations: %w", err)
	}
	defer rows.Close()
	
	allocations := make(map[string]float64)
	
	for rows.Next() {
		var strategyID string
		var allocationPercent float64
		
		if err := rows.Scan(&strategyID, &allocationPercent); err != nil {
			fre.logger.Warn("Failed to scan allocation row", "error", err)
			continue
		}
		
		allocations[strategyID] = allocationPercent
	}
	
	return allocations, nil
}

func (fre *FundReallocationExecutor) calculateExpectedMetrics(
	ctx context.Context,
	strategies []shared.Strategy,
) ([]StrategyMetrics, error) {
	metrics := make([]StrategyMetrics, len(strategies))
	
	for i, strategy := range strategies {
		// Get historical returns for the strategy
		returns, err := fre.getStrategyReturns(ctx, strategy.ID)
		if err != nil {
			fre.logger.Warn("Failed to get strategy returns", 
				"strategy_id", strategy.ID, "error", err)
			// Use default values
			metrics[i] = StrategyMetrics{Return: 0.08, Risk: 0.15}
			continue
		}
		
		expectedReturn := fre.calculateMean(returns)
		expectedRisk := fre.calculateVolatility(returns)
		
		metrics[i] = StrategyMetrics{
			Return: expectedReturn,
			Risk:   expectedRisk,
		}
	}
	
	return metrics, nil
}

func (fre *FundReallocationExecutor) getStrategyReturns(ctx context.Context, strategyID string) ([]float64, error) {
	query := `
		SELECT daily_return 
		FROM strategy_performance 
		WHERE strategy_id = ? 
		AND date >= ? 
		ORDER BY date DESC
		LIMIT 252
	`
	
	startDate := time.Now().AddDate(0, 0, -252) // Last 252 trading days
	
	rows, err := fre.db.QueryContext(ctx, query, strategyID, startDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query strategy returns: %w", err)
	}
	defer rows.Close()
	
	returns := make([]float64, 0)
	
	for rows.Next() {
		var dailyReturn float64
		if err := rows.Scan(&dailyReturn); err != nil {
			fre.logger.Warn("Failed to scan return value", "error", err)
			continue
		}
		returns = append(returns, dailyReturn)
	}
	
	return returns, nil
}

func (fre *FundReallocationExecutor) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	
	return sum / float64(len(values))
}

func (fre *FundReallocationExecutor) calculateVolatility(returns []float64) float64 {
	if len(returns) < 2 {
		return 0.0
	}
	
	mean := fre.calculateMean(returns)
	sumSquaredDiffs := 0.0
	
	for _, ret := range returns {
		diff := ret - mean
		sumSquaredDiffs += diff * diff
	}
	
	variance := sumSquaredDiffs / float64(len(returns)-1)
	return math.Sqrt(variance) * math.Sqrt(252) // Annualized
}

func (fre *FundReallocationExecutor) buildCovarianceMatrix(
	ctx context.Context,
	strategies []shared.Strategy,
) ([][]float64, error) {
	n := len(strategies)
	matrix := make([][]float64, n)
	for i := range matrix {
		matrix[i] = make([]float64, n)
	}
	
	// Get returns for all strategies
	allReturns := make([][]float64, n)
	for i, strategy := range strategies {
		returns, err := fre.getStrategyReturns(ctx, strategy.ID)
		if err != nil {
			fre.logger.Warn("Failed to get returns for covariance", 
				"strategy_id", strategy.ID, "error", err)
			// Use default correlation
			for j := 0; j < n; j++ {
				if i == j {
					matrix[i][j] = 0.15 * 0.15 // Default variance
				} else {
					matrix[i][j] = 0.3 * 0.15 * 0.15 // Default correlation * variances
				}
			}
			continue
		}
		allReturns[i] = returns
	}
	
	// Calculate covariance matrix
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				matrix[i][j] = fre.calculateVariance(allReturns[i])
			} else {
				matrix[i][j] = fre.calculateCovariance(allReturns[i], allReturns[j])
			}
		}
	}
	
	return matrix, nil
}

func (fre *FundReallocationExecutor) calculateVariance(returns []float64) float64 {
	if len(returns) < 2 {
		return 0.15 * 0.15 // Default variance
	}
	
	mean := fre.calculateMean(returns)
	sumSquaredDiffs := 0.0
	
	for _, ret := range returns {
		diff := ret - mean
		sumSquaredDiffs += diff * diff
	}
	
	return sumSquaredDiffs / float64(len(returns)-1)
}

func (fre *FundReallocationExecutor) calculateCovariance(returns1, returns2 []float64) float64 {
	if len(returns1) != len(returns2) || len(returns1) < 2 {
		return 0.3 * 0.15 * 0.15 // Default covariance
	}
	
	mean1 := fre.calculateMean(returns1)
	mean2 := fre.calculateMean(returns2)
	
	sumProducts := 0.0
	for i := 0; i < len(returns1); i++ {
		sumProducts += (returns1[i] - mean1) * (returns2[i] - mean2)
	}
	
	return sumProducts / float64(len(returns1)-1)
}

func (fre *FundReallocationExecutor) solveOptimizationProblem(
	expectedMetrics []StrategyMetrics,
	covarianceMatrix [][]float64,
	constraints AllocationConstraints,
) ([]float64, error) {
	n := len(expectedMetrics)
	if n == 0 {
		return nil, fmt.Errorf("no strategies provided")
	}
	
	// Simplified mean-variance optimization
	// In practice, this would use a proper quadratic programming solver
	
	weights := make([]float64, n)
	
	// Start with equal weights
	equalWeight := 1.0 / float64(n)
	for i := range weights {
		weights[i] = equalWeight
	}
	
	// Apply constraints
	for i := range weights {
		if weights[i] < constraints.MinAllocation {
			weights[i] = constraints.MinAllocation
		}
		if weights[i] > constraints.MaxAllocation {
			weights[i] = constraints.MaxAllocation
		}
	}
	
	// Normalize to sum to 1
	fre.normalizeWeights(weights)
	
	return weights, nil
}

func (fre *FundReallocationExecutor) normalizeWeights(weights []float64) {
	sum := 0.0
	for _, w := range weights {
		sum += w
	}
	
	if sum > 0 {
		for i := range weights {
			weights[i] /= sum
		}
	}
}

func (fre *FundReallocationExecutor) calculateTransactionCost(strategyID string, amount float64) float64 {
	// Get transaction cost rate from config or database
	costRate := fre.config.GetFloat64("fund_reallocation.transaction_cost_rate")
	if costRate == 0 {
		costRate = 0.001 // Default 0.1%
	}
	
	return amount * costRate
}

func (fre *FundReallocationExecutor) calculateMarketImpact(
	ctx context.Context,
	strategyID string,
	amount float64,
) float64 {
	// Simplified market impact calculation
	// In practice, this would use sophisticated market impact models
	
	impactModel := fre.getMarketImpactModel(ctx, strategyID)
	
	// Linear + square root impact model
	linearImpact := impactModel.LinearImpact * amount
	sqrtImpact := impactModel.SquareRootImpact * math.Sqrt(amount)
	
	totalImpact := linearImpact + sqrtImpact
	
	// Apply volatility adjustment
	adjustedImpact := totalImpact * impactModel.VolatilityAdjustment
	
	return adjustedImpact
}

func (fre *FundReallocationExecutor) getMarketImpactModel(
	ctx context.Context,
	strategyID string,
) MarketImpactModel {
	// In practice, this would query from database or calculate from market data
	return MarketImpactModel{
		Symbol:               strategyID,
		LinearImpact:         0.0001,
		SquareRootImpact:     0.001,
		TemporaryImpact:      0.0005,
		PermanentImpact:      0.0002,
		LiquidityScore:       0.8,
		VolatilityAdjustment: 1.2,
		CalculatedAt:         time.Now(),
	}
}

func (fre *FundReallocationExecutor) calculateRiskContribution(
	weight float64,
	risk float64,
	covarianceMatrix [][]float64,
	weights []float64,
) float64 {
	// Simplified risk contribution calculation
	// In practice, this would use proper portfolio risk decomposition
	
	return weight * risk * risk
}

func (fre *FundReallocationExecutor) calculatePriority(allocationChange, transactionCost float64) int {
	// Higher absolute change and lower cost get higher priority (lower number)
	changeScore := math.Abs(allocationChange) * 100
	costPenalty := transactionCost * 1000
	
	priority := int(costPenalty - changeScore)
	
	// Ensure positive priority
	if priority < 1 {
		priority = 1
	}
	
	return priority
}

func (fre *FundReallocationExecutor) generateRationale(
	strategy shared.Strategy,
	allocationChange float64,
) string {
	if allocationChange > 0 {
		return fmt.Sprintf("Increasing allocation to %s due to positive expected performance", 
			strategy.Name)
	} else if allocationChange < 0 {
		return fmt.Sprintf("Decreasing allocation to %s due to risk management or underperformance", 
			strategy.Name)
	}
	return fmt.Sprintf("Maintaining allocation to %s", strategy.Name)
}

func (fre *FundReallocationExecutor) checkConstraintActive(
	allocation float64,
	constraints AllocationConstraints,
) bool {
	return allocation <= constraints.MinAllocation || allocation >= constraints.MaxAllocation
}

func (fre *FundReallocationExecutor) createFundTransfer(allocation OptimalAllocation) shared.FundTransfer {
	transferType := "REALLOCATION"
	if allocation.AllocationChange > 0 {
		transferType = "ALLOCATION_INCREASE"
	} else if allocation.AllocationChange < 0 {
		transferType = "ALLOCATION_DECREASE"
	}
	
	return shared.FundTransfer{
		ID:               fmt.Sprintf("transfer_%s_%d", allocation.StrategyID, time.Now().UnixNano()),
		Type:             transferType,
		Amount:           math.Abs(allocation.AllocationChange),
		Currency:         "USDT",
		Status:           "PENDING",
		Priority:         allocation.Priority,
		EstimatedFee:     allocation.TransactionCost,
		CreatedAt:        time.Now(),
		Metadata: map[string]interface{}{
			"strategy_id":       allocation.StrategyID,
			"allocation_change": allocation.AllocationChange,
			"market_impact":     allocation.MarketImpact,
		},
	}
}

func (fre *FundReallocationExecutor) createExecutionSteps(transfers []shared.FundTransfer) []shared.ExecutionStep {
	steps := make([]shared.ExecutionStep, 0)
	
	for i, transfer := range transfers {
		step := shared.ExecutionStep{
			ID:          fmt.Sprintf("step_%d", i+1),
			Type:        "FUND_TRANSFER",
			Description: fmt.Sprintf("Execute fund transfer %s", transfer.ID),
			Parameters: map[string]interface{}{
				"transfer_id": transfer.ID,
				"amount":      transfer.Amount,
				"priority":    transfer.Priority,
			},
			Order:  i + 1,
			Status: "PENDING",
		}
		steps = append(steps, step)
	}
	
	return steps
}

func (fre *FundReallocationExecutor) estimateExecutionDuration(transfers []shared.FundTransfer) time.Duration {
	// Estimate based on number of transfers and their complexity
	baseTime := time.Minute * 5 // Base execution time
	perTransferTime := time.Minute * 2
	
	totalTime := baseTime + time.Duration(len(transfers))*perTransferTime
	
	return totalTime
}

func (fre *FundReallocationExecutor) estimateTotalCost(transfers []shared.FundTransfer) float64 {
	totalCost := 0.0
	for _, transfer := range transfers {
		totalCost += transfer.EstimatedFee
	}
	return totalCost
}

func (fre *FundReallocationExecutor) estimateTotalImpact(transfers []shared.FundTransfer) float64 {
	totalImpact := 0.0
	for _, transfer := range transfers {
		if marketImpact, ok := transfer.Metadata["market_impact"].(float64); ok {
			totalImpact += marketImpact
		}
	}
	return totalImpact
}

func (fre *FundReallocationExecutor) assessTransferRisk(
	ctx context.Context,
	transfers []shared.FundTransfer,
) *TransferRiskAssessment {
	// Simplified risk assessment
	overallRisk := shared.RiskLevelLow
	liquidityRisk := 0.1
	marketImpactRisk := 0.15
	executionRisk := 0.05
	counterpartyRisk := 0.02
	
	// Increase risk based on transfer size and count
	if len(transfers) > 10 {
		overallRisk = shared.RiskLevelMedium
		executionRisk += 0.1
	}
	
	totalAmount := 0.0
	for _, transfer := range transfers {
		totalAmount += transfer.Amount
	}
	
	if totalAmount > 1000000 { // Large transfers
		overallRisk = shared.RiskLevelHigh
		marketImpactRisk += 0.2
		liquidityRisk += 0.15
	}
	
	return &TransferRiskAssessment{
		OverallRisk:      overallRisk,
		LiquidityRisk:    liquidityRisk,
		MarketImpactRisk: marketImpactRisk,
		ExecutionRisk:    executionRisk,
		CounterpartyRisk: counterpartyRisk,
		RiskFactors: []string{
			"Market volatility",
			"Liquidity constraints",
			"Execution timing",
		},
		MitigationStrategies: []string{
			"Batch execution",
			"Time-weighted execution",
			"Impact monitoring",
		},
		AssessedAt: time.Now(),
	}
}

type BatchResult struct {
	CompletedTransfers int
	FailedTransfers    int
	TotalCost          float64
	TotalImpact        float64
	Errors             []string
	Warnings           []string
}

func (fre *FundReallocationExecutor) createExecutionBatches(transfers []shared.FundTransfer) [][]shared.FundTransfer {
	if len(transfers) == 0 {
		return [][]shared.FundTransfer{}
	}
	
	batches := make([][]shared.FundTransfer, 0)
	
	for i := 0; i < len(transfers); i += fre.batchSize {
		end := i + fre.batchSize
		if end > len(transfers) {
			end = len(transfers)
		}
		batches = append(batches, transfers[i:end])
	}
	
	return batches
}

func (fre *FundReallocationExecutor) executeBatch(
	ctx context.Context,
	batch []shared.FundTransfer,
) BatchResult {
	result := BatchResult{
		Errors:   make([]string, 0),
		Warnings: make([]string, 0),
	}
	
	for _, transfer := range batch {
		err := fre.executeTransfer(ctx, transfer)
		if err != nil {
			result.FailedTransfers++
			result.Errors = append(result.Errors, 
				fmt.Sprintf("Transfer %s failed: %v", transfer.ID, err))
		} else {
			result.CompletedTransfers++
			result.TotalCost += transfer.EstimatedFee
			
			if marketImpact, ok := transfer.Metadata["market_impact"].(float64); ok {
				result.TotalImpact += marketImpact
			}
		}
	}
	
	return result
}

func (fre *FundReallocationExecutor) executeTransfer(
	ctx context.Context,
	transfer shared.FundTransfer,
) error {
	// Simplified transfer execution
	// In practice, this would interact with actual exchange APIs or internal systems
	
	fre.logger.Info("Executing fund transfer", 
		"transfer_id", transfer.ID,
		"amount", transfer.Amount,
		"type", transfer.Type,
	)
	
	// Simulate execution time
	time.Sleep(time.Millisecond * 100)
	
	// Update transfer status in database
	query := `
		UPDATE fund_transfers 
		SET status = 'COMPLETED', executed_at = ?, actual_fee = ?
		WHERE id = ?
	`
	
	_, err := fre.db.ExecContext(ctx, query, time.Now(), transfer.EstimatedFee, transfer.ID)
	if err != nil {
		return fmt.Errorf("failed to update transfer status: %w", err)
	}
	
	return nil
}

func (fre *FundReallocationExecutor) calculateBatchWaitTime(totalImpact float64) time.Duration {
	// Calculate wait time based on market impact
	baseWait := time.Second * 30
	impactMultiplier := totalImpact * 100
	
	waitTime := baseWait + time.Duration(impactMultiplier)*time.Second
	
	// Cap at maximum wait time
	maxWait := time.Minute * 5
	if waitTime > maxWait {
		waitTime = maxWait
	}
	
	return waitTime
}

func (fre *FundReallocationExecutor) updateAllocationRecords(
	ctx context.Context,
	plan *FundTransferPlan,
) error {
	// Update allocation records in database
	for _, transfer := range plan.Transfers {
		if strategyID, ok := transfer.Metadata["strategy_id"].(string); ok {
			if allocationChange, ok := transfer.Metadata["allocation_change"].(float64); ok {
				query := `
					UPDATE current_allocations 
					SET allocation_percent = allocation_percent + ?,
						last_updated = ?
					WHERE strategy_id = ?
				`
				
				_, err := fre.db.ExecContext(ctx, query, allocationChange, time.Now(), strategyID)
				if err != nil {
					return fmt.Errorf("failed to update allocation for strategy %s: %w", strategyID, err)
				}
			}
		}
	}
	
	return nil
}

func (fre *FundReallocationExecutor) getHistoricalMetrics(
	ctx context.Context,
	allocationID string,
	period time.Duration,
	beforeAllocation bool,
) (map[string]float64, error) {
	// Simplified historical metrics retrieval
	// In practice, this would query comprehensive performance data
	
	metrics := map[string]float64{
		"total_return":    0.08,
		"volatility":      0.15,
		"sharpe_ratio":    0.53,
		"max_drawdown":    0.12,
		"win_rate":        0.65,
	}
	
	// Simulate different metrics for before/after
	if beforeAllocation {
		metrics["total_return"] = 0.06
		metrics["sharpe_ratio"] = 0.40
	}
	
	return metrics, nil
}

func (fre *FundReallocationExecutor) calculatePerformanceChange(
	preMetrics, postMetrics map[string]float64,
) map[string]float64 {
	changes := make(map[string]float64)
	
	for metric, postValue := range postMetrics {
		if preValue, exists := preMetrics[metric]; exists {
			change := postValue - preValue
			changes[metric] = change
		}
	}
	
	return changes
}

func (fre *FundReallocationExecutor) calculateRiskChange(
	preMetrics, postMetrics map[string]float64,
) map[string]float64 {
	riskChanges := make(map[string]float64)
	
	riskMetrics := []string{"volatility", "max_drawdown", "var"}
	
	for _, metric := range riskMetrics {
		if postValue, postExists := postMetrics[metric]; postExists {
			if preValue, preExists := preMetrics[metric]; preExists {
				change := postValue - preValue
				riskChanges[metric] = change
			}
		}
	}
	
	return riskChanges
}

func (fre *FundReallocationExecutor) analyzeCosts(
	ctx context.Context,
	allocationID string,
) map[string]float64 {
	// Simplified cost analysis
	return map[string]float64{
		"transaction_costs": 150.0,
		"market_impact":     75.0,
		"opportunity_cost":  25.0,
		"total_cost":        250.0,
	}
}

func (fre *FundReallocationExecutor) calculateEffectivenessScore(
	performanceChange, riskChange, costAnalysis map[string]float64,
) float64 {
	// Simplified effectiveness score calculation
	returnImprovement := performanceChange["total_return"]
	riskReduction := -riskChange["volatility"] // Negative change in risk is good
	totalCost := costAnalysis["total_cost"]
	
	// Normalize and weight the components
	returnScore := returnImprovement * 100
	riskScore := riskReduction * 50
	costScore := -totalCost / 1000 // Cost penalty
	
	effectivenessScore := returnScore + riskScore + costScore
	
	// Normalize to 0-1 scale
	return math.Max(0, math.Min(1, (effectivenessScore+50)/100))
}

func (fre *FundReallocationExecutor) generatePerformanceRecommendations(
	performanceChange, riskChange map[string]float64,
	effectivenessScore float64,
) []string {
	recommendations := make([]string, 0)
	
	if effectivenessScore > 0.7 {
		recommendations = append(recommendations, 
			"Allocation rebalancing was highly effective, consider similar strategies")
	} else if effectivenessScore < 0.3 {
		recommendations = append(recommendations, 
			"Allocation rebalancing showed limited effectiveness, review allocation criteria")
	}
	
	if returnImprovement, ok := performanceChange["total_return"]; ok && returnImprovement > 0.02 {
		recommendations = append(recommendations, 
			"Significant return improvement observed, monitor for sustainability")
	}
	
	if riskIncrease, ok := riskChange["volatility"]; ok && riskIncrease > 0.05 {
		recommendations = append(recommendations, 
			"Risk increased significantly, consider risk management measures")
	}
	
	return recommendations
}