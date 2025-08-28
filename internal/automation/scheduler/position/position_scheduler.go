package position

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"qcat/internal/automation/scheduler/shared"
	"qcat/internal/database"
	"qcat/internal/exchange"
	"qcat/internal/logger"
)

// PositionScheduler implements the main position optimization scheduler
type PositionScheduler struct {
	// Core components
	optimizer         *PositionOptimizer
	executor          *RebalanceExecutor
	performanceTracker *PerformanceTracker
	costCalculator    *TransactionCostCalculator
	fundAllocator     *FundAllocator
	reallocationExecutor *FundReallocationExecutor
	
	// Layered position management components
	layeredPositionManager *LayeredPositionManager
	layeredExecutionSystem *LayeredExecutionSystem
	
	// Dependencies
	db             *database.DB
	exchangeClient exchange.Exchange
	logger         logger.Logger
	config         shared.ConfigProvider
	
	// Scheduler state
	mu              sync.RWMutex
	isRunning       bool
	lastOptimization time.Time
	optimizationCount int64
	
	// Configuration
	optimizationInterval time.Duration
	rebalanceThreshold   float64
	maxPositions         int
	riskBudget          float64
}

// NewPositionScheduler creates a new position scheduler
func NewPositionScheduler(
	db *database.DB,
	exchangeClient exchange.Exchange,
	logger logger.Logger,
	config shared.ConfigProvider,
) *PositionScheduler {
	costCalculator := NewTransactionCostCalculator(db, logger, config)
	optimizer := NewPositionOptimizer(db, exchangeClient, logger, config)
	executor := NewRebalanceExecutor(db, exchangeClient, logger, config, costCalculator)
	performanceTracker := NewPerformanceTracker(db, logger, config)
	fundAllocator := NewFundAllocator(db, exchangeClient, logger, config)
	reallocationExecutor := NewFundReallocationExecutor(db, exchangeClient, logger, config)
	
	// Initialize layered position management components
	layeredPositionManager := NewLayeredPositionManager(db, exchangeClient, logger, config)
	layeredExecutionSystem := NewLayeredExecutionSystem(db, exchangeClient, logger, config)
	
	return &PositionScheduler{
		optimizer:              optimizer,
		executor:               executor,
		performanceTracker:     performanceTracker,
		costCalculator:         costCalculator,
		fundAllocator:          fundAllocator,
		reallocationExecutor:   reallocationExecutor,
		layeredPositionManager: layeredPositionManager,
		layeredExecutionSystem: layeredExecutionSystem,
		db:                     db,
		exchangeClient:         exchangeClient,
		logger:                 logger,
		config:                 config,
		optimizationInterval:   config.GetDuration("position.optimization_interval"),
		rebalanceThreshold:     config.GetFloat64("position.rebalance_threshold"),
		maxPositions:           config.GetInt("position.max_positions"),
		riskBudget:            config.GetFloat64("position.risk_budget"),
	}
}

// Start starts the position scheduler
func (ps *PositionScheduler) Start() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	
	if ps.isRunning {
		return fmt.Errorf("position scheduler is already running")
	}
	
	ps.isRunning = true
	ps.logger.Info("Starting position scheduler")
	
	// Start optimization loop in background
	go ps.optimizationLoop()
	
	return nil
}

// Stop stops the position scheduler
func (ps *PositionScheduler) Stop() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	
	if !ps.isRunning {
		return fmt.Errorf("position scheduler is not running")
	}
	
	ps.isRunning = false
	ps.logger.Info("Stopping position scheduler")
	
	return nil
}

// IsRunning returns whether the scheduler is running
func (ps *PositionScheduler) IsRunning() bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.isRunning
}

// GetStatus returns the current status of the scheduler
func (ps *PositionScheduler) GetStatus() map[string]interface{} {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	
	return map[string]interface{}{
		"running":             ps.isRunning,
		"last_optimization":   ps.lastOptimization,
		"optimization_count":  ps.optimizationCount,
		"optimization_interval": ps.optimizationInterval,
		"rebalance_threshold": ps.rebalanceThreshold,
		"max_positions":       ps.maxPositions,
		"risk_budget":         ps.riskBudget,
	}
}

// OptimizePositions performs portfolio optimization
func (ps *PositionScheduler) OptimizePositions(ctx context.Context) (*shared.OptimizationResult, error) {
	ps.logger.Info("Starting position optimization")
	
	startTime := time.Now()
	
	// Get current positions
	currentPositions, err := ps.optimizer.GetCurrentPositions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current positions: %w", err)
	}
	
	// Define optimization constraints
	constraints := shared.OptimizationConstraints{
		MaxPositionSize:    ps.config.GetFloat64("position.max_position_size"),
		MaxLeverage:        ps.config.GetFloat64("position.max_leverage"),
		MinDiversification: ps.config.GetFloat64("position.min_diversification"),
		TransactionCosts:   ps.getTransactionCostMap(),
		RiskBudget:         ps.riskBudget,
	}
	
	// Calculate optimal positions
	targetPositions, err := ps.optimizer.CalculateOptimalPositions(ctx, constraints)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate optimal positions: %w", err)
	}
	
	// Convert to Position slice for rebalance instructions
	currentPosSlice := make([]shared.Position, len(currentPositions))
	copy(currentPosSlice, currentPositions)
	
	targetPosSlice := make([]shared.Position, len(targetPositions))
	for i, target := range targetPositions {
		targetPosSlice[i] = shared.Position{
			Symbol: target.Symbol,
			Size:   target.TargetSize,
		}
	}
	
	// Generate rebalancing instructions
	instructions, err := ps.optimizer.GenerateRebalanceInstructions(ctx, currentPosSlice, targetPosSlice)
	if err != nil {
		return nil, fmt.Errorf("failed to generate rebalance instructions: %w", err)
	}
	
	// Calculate expected metrics
	expectedReturn, expectedRisk, expectedSharpe := ps.calculateExpectedMetrics(targetPositions, constraints)
	
	// Create optimization result
	result := &shared.OptimizationResult{
		TargetPositions:   targetPositions,
		ExpectedReturn:    expectedReturn,
		ExpectedRisk:      expectedRisk,
		SharpeRatio:       expectedSharpe,
		Constraints:       constraints,
		OptimizationTime:  time.Since(startTime),
		Metadata: map[string]interface{}{
			"current_positions_count": len(currentPositions),
			"target_positions_count":  len(targetPositions),
			"rebalance_instructions":  len(instructions),
		},
	}
	
	// Update scheduler state
	ps.mu.Lock()
	ps.lastOptimization = time.Now()
	ps.optimizationCount++
	ps.mu.Unlock()
	
	ps.logger.Info("Position optimization completed",
		"optimization_time", result.OptimizationTime,
		"expected_return", expectedReturn,
		"expected_risk", expectedRisk,
		"expected_sharpe", expectedSharpe,
	)
	
	return result, nil
}

// RebalancePortfolio executes portfolio rebalancing
func (ps *PositionScheduler) RebalancePortfolio(
	ctx context.Context,
	target *shared.AllocationTarget,
) error {
	ps.logger.Info("Starting portfolio rebalancing")
	
	// Get current positions
	currentPositions, err := ps.optimizer.GetCurrentPositions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current positions: %w", err)
	}
	
	// Convert allocation target to target positions
	targetPositions := ps.convertAllocationToPositions(target, currentPositions)
	
	// Generate rebalancing instructions
	instructions, err := ps.optimizer.GenerateRebalanceInstructions(ctx, currentPositions, targetPositions)
	if err != nil {
		return fmt.Errorf("failed to generate rebalance instructions: %w", err)
	}
	
	// Check if rebalancing is needed
	if !ps.isRebalancingNeeded(instructions) {
		ps.logger.Info("No significant rebalancing needed")
		return nil
	}
	
	// Execute rebalancing
	options := &ExecutionOptions{
		ExecutionStrategy: ps.config.GetString("position.execution_strategy"),
		MaxSlippage:       ps.config.GetFloat64("position.max_slippage"),
		TimeLimit:         ps.config.GetDuration("position.execution_timeout"),
		DryRun:            ps.config.GetBool("position.dry_run"),
	}
	
	result, err := ps.executor.ExecuteRebalancing(ctx, instructions, options)
	if err != nil {
		return fmt.Errorf("failed to execute rebalancing: %w", err)
	}
	
	// Track performance
	if result.Success {
		optimizationID := fmt.Sprintf("rebalance_%d", time.Now().UnixNano())
		err = ps.performanceTracker.TrackOptimizationPerformance(
			ctx, optimizationID, currentPositions, targetPositions)
		if err != nil {
			ps.logger.Warn("Failed to track optimization performance", "error", err)
		}
	}
	
	ps.logger.Info("Portfolio rebalancing completed",
		"success", result.Success,
		"completed_trades", result.CompletedTrades,
		"failed_trades", result.FailedTrades,
		"total_cost", result.TotalCost,
	)
	
	return nil
}

// GetName returns the scheduler name
func (ps *PositionScheduler) GetName() string {
	return "PositionScheduler"
}

// GetVersion returns the scheduler version
func (ps *PositionScheduler) GetVersion() string {
	return "1.0.0"
}

// GetDescription returns the scheduler description
func (ps *PositionScheduler) GetDescription() string {
	return "Automated portfolio position optimization and rebalancing scheduler"
}

// GetSupportedTasks returns the list of supported tasks
func (ps *PositionScheduler) GetSupportedTasks() []string {
	return []string{
		"optimize_positions",
		"rebalance_portfolio",
		"calculate_optimal_allocation",
		"generate_rebalance_instructions",
		"execute_trades",
		"track_performance",
	}
}

// GetMetrics returns scheduler metrics
func (ps *PositionScheduler) GetMetrics() map[string]interface{} {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	
	return map[string]interface{}{
		"optimization_count":    ps.optimizationCount,
		"last_optimization":     ps.lastOptimization,
		"optimization_interval": ps.optimizationInterval,
		"is_running":           ps.isRunning,
	}
}

// GetHealth returns scheduler health status
func (ps *PositionScheduler) GetHealth() shared.HealthStatus {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	
	status := "HEALTHY"
	errors := make([]string, 0)
	warnings := make([]string, 0)
	
	// Check if scheduler is running
	if !ps.isRunning {
		status = "UNHEALTHY"
		errors = append(errors, "Scheduler is not running")
	}
	
	// Check last optimization time
	if time.Since(ps.lastOptimization) > ps.optimizationInterval*2 {
		status = "DEGRADED"
		warnings = append(warnings, "Last optimization was too long ago")
	}
	
	return shared.HealthStatus{
		Status:       status,
		LastCheck:    time.Now(),
		Errors:       errors,
		Warnings:     warnings,
		Metrics:      ps.GetMetrics(),
		Uptime:       time.Since(ps.lastOptimization),
		TasksTotal:   ps.optimizationCount,
		TasksSuccess: ps.optimizationCount, // Simplified
		TasksFailed:  0,                    // Simplified
	}
}

// Execute executes a task
func (ps *PositionScheduler) Execute(ctx context.Context, task interface{}) error {
	taskMap, ok := task.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid task format")
	}
	
	taskType, ok := taskMap["type"].(string)
	if !ok {
		return fmt.Errorf("missing task type")
	}
	
	switch taskType {
	case "optimize_positions":
		_, err := ps.OptimizePositions(ctx)
		return err
	case "rebalance_portfolio":
		target, ok := taskMap["target"].(*shared.AllocationTarget)
		if !ok {
			return fmt.Errorf("invalid allocation target")
		}
		return ps.RebalancePortfolio(ctx, target)
	default:
		return fmt.Errorf("unsupported task type: %s", taskType)
	}
}

// CanExecute checks if a task can be executed
func (ps *PositionScheduler) CanExecute(task interface{}) bool {
	taskMap, ok := task.(map[string]interface{})
	if !ok {
		return false
	}
	
	taskType, ok := taskMap["type"].(string)
	if !ok {
		return false
	}
	
	supportedTasks := ps.GetSupportedTasks()
	for _, supported := range supportedTasks {
		if supported == taskType {
			return true
		}
	}
	
	return false
}

// GetExecutorType returns the executor type
func (ps *PositionScheduler) GetExecutorType() string {
	return "PositionOptimizationExecutor"
}

// AllocateFunds allocates funds across strategies using intelligent fund allocation
func (ps *PositionScheduler) AllocateFunds(ctx context.Context, strategies []shared.Strategy) (*shared.AllocationPlan, error) {
	ps.logger.Info("Allocating funds across strategies", "strategy_count", len(strategies))
	
	// Perform fund efficiency analysis
	efficiencyReport, err := ps.fundAllocator.AnalyzeFundEfficiency(ctx)
	if err != nil {
		ps.logger.Warn("Failed to analyze fund efficiency, using fallback allocation", "error", err)
		return ps.fallbackAllocation(strategies)
	}
	
	// Calculate risk parity allocation
	riskParityAllocations, err := ps.fundAllocator.CalculateRiskParityAllocation(ctx, strategies)
	if err != nil {
		ps.logger.Warn("Failed to calculate risk parity allocation, using efficiency-based allocation", "error", err)
		return ps.efficiencyBasedAllocation(strategies, efficiencyReport)
	}
	
	// Create allocation plan based on risk parity with efficiency adjustments
	allocations := make(map[string]float64)
	transfers := make([]shared.FundTransfer, 0)
	
	for _, riskAllocation := range riskParityAllocations {
		// Find efficiency score for this strategy
		efficiencyScore := ps.getEfficiencyScore(riskAllocation.StrategyID, efficiencyReport)
		
		// Adjust allocation based on efficiency
		adjustedAllocation := riskAllocation.OptimalAllocation * (0.5 + efficiencyScore*0.5)
		allocations[riskAllocation.StrategyID] = adjustedAllocation
		
		// Create fund transfer
		transferAmount := adjustedAllocation * ps.riskBudget
		transfer := shared.FundTransfer{
			ID:       fmt.Sprintf("transfer_%s_%d", riskAllocation.StrategyID, time.Now().UnixNano()),
			Type:     "STRATEGY_ALLOCATION",
			Amount:   transferAmount,
			Currency: "USDT",
			Status:   "PENDING",
			Priority: 1,
			CreatedAt: time.Now(),
			Metadata: map[string]interface{}{
				"risk_contribution": riskAllocation.RiskContribution,
				"efficiency_score":  efficiencyScore,
				"allocation_method": "risk_parity_efficiency_adjusted",
			},
		}
		transfers = append(transfers, transfer)
	}
	
	// Normalize allocations to sum to 1
	ps.normalizeAllocations(allocations)
	
	plan := &shared.AllocationPlan{
		ID:                fmt.Sprintf("allocation_%d", time.Now().UnixNano()),
		TargetAllocations: allocations,
		RequiredTransfers: transfers,
		ExpectedImpact: map[string]interface{}{
			"efficiency_report": efficiencyReport,
			"risk_parity_allocations": riskParityAllocations,
		},
		CreatedAt: time.Now(),
	}
	
	ps.logger.Info("Fund allocation plan created", 
		"plan_id", plan.ID,
		"strategies_count", len(allocations),
		"total_transfers", len(transfers),
	)
	
	return plan, nil
}

// ExecuteAllocation executes a fund allocation plan with optimal execution
func (ps *PositionScheduler) ExecuteAllocation(ctx context.Context, plan *shared.AllocationPlan) error {
	ps.logger.Info("Executing allocation plan", "plan_id", plan.ID)
	
	// Convert allocation plan to optimal allocations
	optimalAllocations := ps.convertPlanToOptimalAllocations(plan)
	
	// Create fund transfer plan with market impact minimization
	transferPlan, err := ps.reallocationExecutor.CreateFundTransferPlan(ctx, optimalAllocations)
	if err != nil {
		return fmt.Errorf("failed to create fund transfer plan: %w", err)
	}
	
	// Execute reallocation with monitoring
	result, err := ps.reallocationExecutor.ExecuteReallocation(ctx, transferPlan)
	if err != nil {
		return fmt.Errorf("failed to execute reallocation: %w", err)
	}
	
	// Log execution results
	ps.logger.Info("Allocation execution completed", 
		"plan_id", plan.ID,
		"success", result.Success,
		"completed_transfers", result.CompletedTransfers,
		"failed_transfers", result.FailedTransfers,
		"total_cost", result.TotalCost,
		"execution_time", result.ExecutionTime,
	)
	
	if !result.Success {
		return fmt.Errorf("allocation execution failed: %d transfers failed", result.FailedTransfers)
	}
	
	// Monitor allocation performance after execution
	go ps.monitorAllocationPerformance(ctx, plan.ID)
	
	return nil
}

// ManageLayeredPositions manages layered position strategies
func (ps *PositionScheduler) ManageLayeredPositions(ctx context.Context, config *shared.LayerConfig) error {
	ps.logger.Info("Managing layered positions", "symbol", config.Symbol)
	
	// Analyze market volatility for the symbol
	volatilityAnalysis, err := ps.layeredPositionManager.AnalyzeMarketVolatility(ctx, config.Symbol)
	if err != nil {
		ps.logger.Warn("Failed to analyze market volatility, proceeding with basic management", "error", err)
	} else {
		ps.logger.Info("Market volatility analysis completed",
			"symbol", config.Symbol,
			"volatility_regime", volatilityAnalysis.VolatilityRegime,
			"recommended_layers", volatilityAnalysis.RecommendedLayers,
		)
	}
	
	// Execute layered positions using the execution system
	execution, err := ps.layeredExecutionSystem.ExecuteLayeredPositions(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to execute layered positions: %w", err)
	}
	
	ps.logger.Info("Layered position execution started",
		"execution_id", execution.ID,
		"symbol", config.Symbol,
		"total_layers", len(config.Layers),
	)
	
	// Monitor execution progress
	go ps.monitorLayeredExecution(ctx, execution.ID)
	
	return nil
}

// AdjustLayers adjusts layer parameters based on market conditions
func (ps *PositionScheduler) AdjustLayers(ctx context.Context, marketConditions *shared.MarketConditions) error {
	ps.logger.Info("Adjusting layers based on market conditions", 
		"volatility", marketConditions.Volatility,
		"trend", marketConditions.Trend,
	)
	
	// Get all active layered executions
	ps.layeredExecutionSystem.mu.RLock()
	activeExecutions := make([]*LayeredExecution, 0, len(ps.layeredExecutionSystem.activeExecutions))
	for _, execution := range ps.layeredExecutionSystem.activeExecutions {
		if execution.Status == "EXECUTING" {
			activeExecutions = append(activeExecutions, execution)
		}
	}
	ps.layeredExecutionSystem.mu.RUnlock()
	
	// Adjust parameters for each active execution
	for _, execution := range activeExecutions {
		err := ps.layeredPositionManager.AdjustLayerParametersDynamically(ctx, execution.Config, marketConditions)
		if err != nil {
			ps.logger.Warn("Failed to adjust layer parameters", 
				"execution_id", execution.ID,
				"symbol", execution.Symbol,
				"error", err,
			)
			continue
		}
		
		ps.logger.Info("Layer parameters adjusted",
			"execution_id", execution.ID,
			"symbol", execution.Symbol,
			"volatility", marketConditions.Volatility,
			"trend", marketConditions.Trend,
		)
	}
	
	return nil
}

// CalculateHedgeRatios calculates hedge ratios for multi-strategy hedging
func (ps *PositionScheduler) CalculateHedgeRatios(ctx context.Context) ([]shared.HedgeRatio, error) {
	ps.logger.Info("Calculating hedge ratios")
	
	// Simplified hedge ratio calculation
	ratios := []shared.HedgeRatio{
		{
			StrategyID:        "strategy1",
			HedgeStrategyID:   "hedge1",
			Ratio:             0.5,
			Confidence:        0.8,
			EffectivenesScore: 0.75,
			LastUpdated:       time.Now(),
		},
		{
			StrategyID:        "strategy2",
			HedgeStrategyID:   "hedge2",
			Ratio:             0.3,
			Confidence:        0.7,
			EffectivenesScore: 0.65,
			LastUpdated:       time.Now(),
		},
	}
	
	return ratios, nil
}

// ExecuteHedging executes hedging operations
func (ps *PositionScheduler) ExecuteHedging(ctx context.Context, ratios []shared.HedgeRatio) error {
	ps.logger.Info("Executing hedging operations", "hedge_count", len(ratios))
	
	for _, ratio := range ratios {
		ps.logger.Info("Executing hedge", 
			"strategy_id", ratio.StrategyID,
			"hedge_strategy_id", ratio.HedgeStrategyID,
			"ratio", ratio.Ratio,
		)
		
		// In a real implementation, this would execute actual hedge trades
	}
	
	return nil
}

// Private methods

func (ps *PositionScheduler) optimizationLoop() {
	ticker := time.NewTicker(ps.optimizationInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if !ps.IsRunning() {
				return
			}
			
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
			
			_, err := ps.OptimizePositions(ctx)
			if err != nil {
				ps.logger.Error("Optimization loop failed", "error", err)
			}
			
			cancel()
			
		default:
			if !ps.IsRunning() {
				return
			}
			time.Sleep(time.Second)
		}
	}
}

func (ps *PositionScheduler) getTransactionCostMap() map[string]float64 {
	// Simplified transaction cost mapping
	return map[string]float64{
		"BTCUSDT": 0.001,
		"ETHUSDT": 0.001,
		"ADAUSDT": 0.001,
		"DOTUSDT": 0.001,
	}
}

func (ps *PositionScheduler) calculateExpectedMetrics(
	targetPositions []shared.TargetPosition,
	constraints shared.OptimizationConstraints,
) (float64, float64, float64) {
	// Simplified expected metrics calculation
	expectedReturn := 0.08  // 8% expected return
	expectedRisk := 0.15    // 15% expected risk
	riskFreeRate := ps.config.GetFloat64("optimization.risk_free_rate")
	expectedSharpe := (expectedReturn - riskFreeRate) / expectedRisk
	
	return expectedReturn, expectedRisk, expectedSharpe
}

func (ps *PositionScheduler) convertAllocationToPositions(
	target *shared.AllocationTarget,
	currentPositions []shared.Position,
) []shared.Position {
	targetPositions := make([]shared.Position, 0)
	
	for symbol, allocation := range target.Allocations {
		targetSize := target.TotalCapital * allocation
		
		// Find current price for the symbol
		currentPrice := ps.getCurrentPrice(symbol)
		if currentPrice > 0 {
			targetQuantity := targetSize / currentPrice
			
			targetPos := shared.Position{
				Symbol: symbol,
				Size:   targetQuantity,
			}
			targetPositions = append(targetPositions, targetPos)
		}
	}
	
	return targetPositions
}

func (ps *PositionScheduler) isRebalancingNeeded(instructions []RebalanceInstruction) bool {
	totalAdjustment := 0.0
	for _, instruction := range instructions {
		totalAdjustment += math.Abs(instruction.Adjustment)
	}
	
	return totalAdjustment > ps.rebalanceThreshold
}

func (ps *PositionScheduler) getCurrentPrice(symbol string) float64 {
	// Simplified price retrieval - would integrate with real market data
	return 100.0 // Placeholder
}

// Helper methods for fund allocation

func (ps *PositionScheduler) fallbackAllocation(strategies []shared.Strategy) (*shared.AllocationPlan, error) {
	// Equal weight allocation as fallback
	totalCapital := ps.riskBudget
	allocationPerStrategy := 1.0 / float64(len(strategies))
	
	allocations := make(map[string]float64)
	transfers := make([]shared.FundTransfer, 0)
	
	for _, strategy := range strategies {
		allocations[strategy.ID] = allocationPerStrategy
		
		transfer := shared.FundTransfer{
			ID:       fmt.Sprintf("fallback_transfer_%s_%d", strategy.ID, time.Now().UnixNano()),
			Type:     "STRATEGY_ALLOCATION",
			Amount:   allocationPerStrategy * totalCapital,
			Currency: "USDT",
			Status:   "PENDING",
			Priority: 1,
			CreatedAt: time.Now(),
		}
		transfers = append(transfers, transfer)
	}
	
	plan := &shared.AllocationPlan{
		ID:                fmt.Sprintf("fallback_allocation_%d", time.Now().UnixNano()),
		TargetAllocations: allocations,
		RequiredTransfers: transfers,
		CreatedAt:         time.Now(),
	}
	
	return plan, nil
}

func (ps *PositionScheduler) efficiencyBasedAllocation(
	strategies []shared.Strategy, 
	efficiencyReport *EfficiencyReport,
) (*shared.AllocationPlan, error) {
	// Allocate based on efficiency scores
	totalEfficiency := 0.0
	efficiencyMap := make(map[string]float64)
	
	for _, result := range efficiencyReport.AnalysisResults {
		efficiencyMap[result.StrategyID] = result.EfficiencyScore
		totalEfficiency += result.EfficiencyScore
	}
	
	allocations := make(map[string]float64)
	transfers := make([]shared.FundTransfer, 0)
	
	for _, strategy := range strategies {
		efficiency := efficiencyMap[strategy.ID]
		allocation := efficiency / totalEfficiency
		
		allocations[strategy.ID] = allocation
		
		transfer := shared.FundTransfer{
			ID:       fmt.Sprintf("efficiency_transfer_%s_%d", strategy.ID, time.Now().UnixNano()),
			Type:     "STRATEGY_ALLOCATION",
			Amount:   allocation * ps.riskBudget,
			Currency: "USDT",
			Status:   "PENDING",
			Priority: 1,
			CreatedAt: time.Now(),
		}
		transfers = append(transfers, transfer)
	}
	
	plan := &shared.AllocationPlan{
		ID:                fmt.Sprintf("efficiency_allocation_%d", time.Now().UnixNano()),
		TargetAllocations: allocations,
		RequiredTransfers: transfers,
		CreatedAt:         time.Now(),
	}
	
	return plan, nil
}

func (ps *PositionScheduler) getEfficiencyScore(strategyID string, report *EfficiencyReport) float64 {
	for _, result := range report.AnalysisResults {
		if result.StrategyID == strategyID {
			return result.EfficiencyScore
		}
	}
	return 0.5 // Default efficiency score
}

func (ps *PositionScheduler) normalizeAllocations(allocations map[string]float64) {
	total := 0.0
	for _, allocation := range allocations {
		total += allocation
	}
	
	if total > 0 {
		for strategyID := range allocations {
			allocations[strategyID] /= total
		}
	}
}

func (ps *PositionScheduler) convertPlanToOptimalAllocations(plan *shared.AllocationPlan) []OptimalAllocation {
	allocations := make([]OptimalAllocation, 0, len(plan.TargetAllocations))
	
	for strategyID, targetAllocation := range plan.TargetAllocations {
		// Get current allocation (simplified)
		currentAllocation := 0.0 // Would query from database
		
		allocation := OptimalAllocation{
			StrategyID:        strategyID,
			CurrentAllocation: currentAllocation,
			TargetAllocation:  targetAllocation,
			AllocationChange:  targetAllocation - currentAllocation,
			TransactionCost:   math.Abs(targetAllocation-currentAllocation) * 0.001, // 0.1% cost
			MarketImpact:      math.Abs(targetAllocation-currentAllocation) * 0.0005, // 0.05% impact
			Priority:          1,
			Rationale:         fmt.Sprintf("Allocation adjustment for strategy %s", strategyID),
			CalculatedAt:      time.Now(),
		}
		
		allocations = append(allocations, allocation)
	}
	
	return allocations
}

func (ps *PositionScheduler) monitorAllocationPerformance(ctx context.Context, allocationID string) {
	// Monitor allocation performance in background
	monitoringPeriod := ps.config.GetDuration("fund_allocation.monitoring_period")
	if monitoringPeriod == 0 {
		monitoringPeriod = time.Hour * 24 * 30 // Default 30 days
	}
	
	// Wait for some time to allow allocation to take effect
	time.Sleep(time.Hour * 24) // Wait 1 day
	
	report, err := ps.reallocationExecutor.MonitorAllocationPerformance(
		ctx, allocationID, monitoringPeriod)
	if err != nil {
		ps.logger.Warn("Failed to monitor allocation performance", 
			"allocation_id", allocationID, "error", err)
		return
	}
	
	ps.logger.Info("Allocation performance monitoring completed", 
		"allocation_id", allocationID,
		"effectiveness_score", report.EffectivenessScore,
		"recommendations_count", len(report.Recommendations),
	)
	
	// Log recommendations
	for _, recommendation := range report.Recommendations {
		ps.logger.Info("Allocation recommendation", 
			"allocation_id", allocationID,
			"recommendation", recommendation,
		)
	}
}

// monitorLayeredExecution monitors the progress of a layered execution
func (ps *PositionScheduler) monitorLayeredExecution(ctx context.Context, executionID string) {
	ticker := time.NewTicker(time.Minute * 5) // Check every 5 minutes
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Check if execution still exists and is active
			ps.layeredExecutionSystem.mu.RLock()
			execution, exists := ps.layeredExecutionSystem.activeExecutions[executionID]
			ps.layeredExecutionSystem.mu.RUnlock()
			
			if !exists {
				ps.logger.Info("Layered execution monitoring completed - execution no longer active", "execution_id", executionID)
				return
			}
			
			if execution.Status == "COMPLETED" || execution.Status == "FAILED" || execution.Status == "CANCELLED" {
				ps.logger.Info("Layered execution monitoring completed", 
					"execution_id", executionID,
					"final_status", execution.Status,
					"total_executed", execution.TotalExecuted,
					"average_price", execution.AveragePrice,
				)
				return
			}
			
			// Track performance metrics
			metrics, err := ps.layeredExecutionSystem.TrackLayeredPositionPerformance(ctx, executionID)
			if err != nil {
				ps.logger.Warn("Failed to track layered position performance", 
					"execution_id", executionID, "error", err)
				continue
			}
			
			ps.logger.Debug("Layered execution progress",
				"execution_id", executionID,
				"status", execution.Status,
				"active_layers", metrics.ActiveLayers,
				"completed_layers", metrics.CompletedLayers,
				"total_return", metrics.TotalReturn,
				"execution_efficiency", metrics.ExecutionEfficiency,
			)
			
			// Check for performance issues
			if metrics.ExecutionEfficiency < 0.7 {
				ps.logger.Warn("Low execution efficiency detected", 
					"execution_id", executionID,
					"efficiency", metrics.ExecutionEfficiency,
				)
			}
			
			if metrics.MaxDrawdown > 0.1 {
				ps.logger.Warn("High drawdown detected in layered execution", 
					"execution_id", executionID,
					"max_drawdown", metrics.MaxDrawdown,
				)
			}
		}
	}
}

// ExecutePartialClosure executes partial closure of layered positions
func (ps *PositionScheduler) ExecutePartialClosure(ctx context.Context, request *PartialClosureRequest) (*PartialClosureResult, error) {
	ps.logger.Info("Executing partial closure", 
		"execution_id", request.ExecutionID,
		"closure_percentage", request.ClosurePercentage,
		"reason", request.ClosureReason,
	)
	
	result, err := ps.layeredExecutionSystem.ExecutePartialPositionClosure(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to execute partial closure: %w", err)
	}
	
	ps.logger.Info("Partial closure completed",
		"execution_id", request.ExecutionID,
		"closed_layers", len(result.ClosedLayers),
		"closed_size", result.ClosedSize,
		"realized_pnl", result.RealizedPnL,
	)
	
	return result, nil
}

// CreateLayeredStrategy creates a new layered position strategy
func (ps *PositionScheduler) CreateLayeredStrategy(ctx context.Context, symbol, direction string, totalSize float64, params map[string]interface{}) (*LayeredStrategy, error) {
	ps.logger.Info("Creating layered strategy", 
		"symbol", symbol,
		"direction", direction,
		"total_size", totalSize,
	)
	
	// Extract parameters with defaults
	maxLayers := ps.config.GetInt("layered_position.default_max_layers")
	if val, ok := params["max_layers"].(int); ok {
		maxLayers = val
	}
	
	layerSizeRatio := 1.0
	if val, ok := params["layer_size_ratio"].(float64); ok {
		layerSizeRatio = val
	}
	
	priceSpacing := ps.config.GetFloat64("layered_position.default_price_spacing")
	if val, ok := params["price_spacing"].(float64); ok {
		priceSpacing = val
	}
	
	volatilityBased := true
	if val, ok := params["volatility_based"].(bool); ok {
		volatilityBased = val
	}
	
	adaptiveSpacing := true
	if val, ok := params["adaptive_spacing"].(bool); ok {
		adaptiveSpacing = val
	}
	
	strategy := &LayeredStrategy{
		ID:                fmt.Sprintf("layered_strategy_%s_%d", symbol, time.Now().UnixNano()),
		Symbol:            symbol,
		Direction:         direction,
		TotalSize:         totalSize,
		MaxLayers:         maxLayers,
		LayerSizeRatio:    layerSizeRatio,
		PriceSpacing:      priceSpacing,
		VolatilityBased:   volatilityBased,
		AdaptiveSpacing:   adaptiveSpacing,
		RiskParameters: shared.RiskParams{
			MaxLeverage:       ps.config.GetFloat64("position.max_leverage"),
			MaxPositionSize:   ps.config.GetFloat64("position.max_position_size"),
			StopLossPercent:   ps.config.GetFloat64("layered_position.default_stop_loss"),
			TakeProfitPercent: ps.config.GetFloat64("layered_position.default_take_profit"),
			MaxDrawdown:       ps.config.GetFloat64("position.max_drawdown"),
			VaRLimit:          ps.config.GetFloat64("position.var_limit"),
		},
		Metadata: params,
		CreatedAt: time.Now(),
	}
	
	ps.logger.Info("Layered strategy created", 
		"strategy_id", strategy.ID,
		"symbol", symbol,
		"max_layers", maxLayers,
		"volatility_based", volatilityBased,
	)
	
	return strategy, nil
}

// ExecuteLayeredStrategy executes a complete layered position strategy
func (ps *PositionScheduler) ExecuteLayeredStrategy(ctx context.Context, strategy *LayeredStrategy) (*LayeredExecution, error) {
	ps.logger.Info("Executing layered strategy", 
		"strategy_id", strategy.ID,
		"symbol", strategy.Symbol,
	)
	
	// Calculate layer configuration based on current market conditions
	layerConfig, err := ps.layeredPositionManager.CalculateLayerConfiguration(ctx, strategy)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate layer configuration: %w", err)
	}
	
	// Execute the layered positions
	execution, err := ps.layeredExecutionSystem.ExecuteLayeredPositions(ctx, layerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to execute layered positions: %w", err)
	}
	
	// Start monitoring
	go ps.monitorLayeredExecution(ctx, execution.ID)
	
	ps.logger.Info("Layered strategy execution started", 
		"strategy_id", strategy.ID,
		"execution_id", execution.ID,
		"layers_count", len(layerConfig.Layers),
	)
	
	return execution, nil
}

// GetLayeredExecutionStatus returns the status of a layered execution
func (ps *PositionScheduler) GetLayeredExecutionStatus(executionID string) (*LayeredExecution, error) {
	ps.layeredExecutionSystem.mu.RLock()
	defer ps.layeredExecutionSystem.mu.RUnlock()
	
	execution, exists := ps.layeredExecutionSystem.activeExecutions[executionID]
	if !exists {
		return nil, fmt.Errorf("execution not found: %s", executionID)
	}
	
	return execution, nil
}

// ListActiveLayeredExecutions returns all active layered executions
func (ps *PositionScheduler) ListActiveLayeredExecutions() []*LayeredExecution {
	ps.layeredExecutionSystem.mu.RLock()
	defer ps.layeredExecutionSystem.mu.RUnlock()
	
	executions := make([]*LayeredExecution, 0, len(ps.layeredExecutionSystem.activeExecutions))
	for _, execution := range ps.layeredExecutionSystem.activeExecutions {
		executions = append(executions, execution)
	}
	
	return executions
}

// Ensure PositionScheduler implements required interfaces
var _ shared.SchedulerInterface = (*PositionScheduler)(nil)
var _ shared.PositionSchedulerInterface = (*PositionScheduler)(nil)
var _ shared.BaseScheduler = (*PositionScheduler)(nil)
var _ shared.TaskExecutor = (*PositionScheduler)(nil)