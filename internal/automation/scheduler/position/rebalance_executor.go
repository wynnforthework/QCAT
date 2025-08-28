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

// RebalanceExecutor handles the execution of portfolio rebalancing operations
type RebalanceExecutor struct {
	db                *database.DB
	exchangeClient    exchange.Exchange
	logger            logger.Logger
	config            shared.ConfigProvider
	costCalculator    *TransactionCostCalculator
	
	// Execution parameters
	maxConcurrentTrades int
	executionTimeout    time.Duration
	retryAttempts       int
	slippageTolerance   float64
	
	// State management
	mu                  sync.RWMutex
	activeExecutions    map[string]*ExecutionContext
	executionHistory    []RebalanceEvent
}

// ExecutionContext represents the context of an ongoing rebalancing execution
type ExecutionContext struct {
	ID               string                 `json:"id"`
	Instructions     []RebalanceInstruction `json:"instructions"`
	Status           string                 `json:"status"`
	StartTime        time.Time              `json:"start_time"`
	EndTime          *time.Time             `json:"end_time,omitempty"`
	Progress         float64                `json:"progress"`
	CompletedTrades  int                    `json:"completed_trades"`
	FailedTrades     int                    `json:"failed_trades"`
	TotalCost        float64                `json:"total_cost"`
	EstimatedCost    float64                `json:"estimated_cost"`
	Errors           []string               `json:"errors"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// NewRebalanceExecutor creates a new rebalance executor
func NewRebalanceExecutor(
	db *database.DB,
	exchangeClient exchange.Exchange,
	logger logger.Logger,
	config shared.ConfigProvider,
	costCalculator *TransactionCostCalculator,
) *RebalanceExecutor {
	return &RebalanceExecutor{
		db:                  db,
		exchangeClient:      exchangeClient,
		logger:              logger,
		config:              config,
		costCalculator:      costCalculator,
		maxConcurrentTrades: config.GetInt("rebalance.max_concurrent_trades"),
		executionTimeout:    config.GetDuration("rebalance.execution_timeout"),
		retryAttempts:       config.GetInt("rebalance.retry_attempts"),
		slippageTolerance:   config.GetFloat64("rebalance.slippage_tolerance"),
		activeExecutions:    make(map[string]*ExecutionContext),
		executionHistory:    make([]RebalanceEvent, 0),
	}
}

// ExecuteRebalancing executes a complete rebalancing operation
func (re *RebalanceExecutor) ExecuteRebalancing(
	ctx context.Context,
	instructions []RebalanceInstruction,
	options *ExecutionOptions,
) (*ExecutionResult, error) {
	re.logger.Info("Starting rebalancing execution", "instructions", len(instructions))
	
	// Create execution context
	executionID := re.generateExecutionID()
	execCtx := &ExecutionContext{
		ID:           executionID,
		Instructions: instructions,
		Status:       "INITIALIZING",
		StartTime:    time.Now(),
		Progress:     0.0,
		Metadata:     make(map[string]interface{}),
	}
	
	// Register execution
	re.mu.Lock()
	re.activeExecutions[executionID] = execCtx
	re.mu.Unlock()
	
	defer func() {
		re.mu.Lock()
		delete(re.activeExecutions, executionID)
		re.mu.Unlock()
	}()
	
	// Pre-execution validation
	if err := re.validateInstructions(ctx, instructions); err != nil {
		execCtx.Status = "FAILED"
		return nil, fmt.Errorf("instruction validation failed: %w", err)
	}
	
	// Estimate costs
	_, totalEstimatedCost, err := re.costCalculator.EstimateBatchTransactionCost(ctx, instructions)
	if err != nil {
		re.logger.Warn("Failed to estimate transaction costs", "error", err)
	} else {
		execCtx.EstimatedCost = totalEstimatedCost
	}
	
	// Optimize execution order
	optimizedInstructions, err := re.costCalculator.OptimizeExecutionOrder(ctx, instructions)
	if err != nil {
		re.logger.Warn("Failed to optimize execution order", "error", err)
		optimizedInstructions = instructions
	}
	
	execCtx.Instructions = optimizedInstructions
	execCtx.Status = "EXECUTING"
	
	// Execute instructions
	result, err := re.executeInstructions(ctx, execCtx, options)
	if err != nil {
		execCtx.Status = "FAILED"
		execCtx.Errors = append(execCtx.Errors, err.Error())
		return result, err
	}
	
	execCtx.Status = "COMPLETED"
	endTime := time.Now()
	execCtx.EndTime = &endTime
	
	// Record execution history
	event := RebalanceEvent{
		ID:                 executionID,
		Type:               "OPTIMIZATION",
		Trigger:            "MANUAL",
		PreRebalanceValue:  result.PreExecutionValue,
		PostRebalanceValue: result.PostExecutionValue,
		TotalCost:          result.TotalCost,
		Instructions:       optimizedInstructions,
		ExecutionTime:      endTime.Sub(execCtx.StartTime),
		Success:            result.Success,
		Timestamp:          execCtx.StartTime,
	}
	
	if !result.Success {
		event.ErrorMessage = result.ErrorMessage
	}
	
	re.mu.Lock()
	re.executionHistory = append(re.executionHistory, event)
	re.mu.Unlock()
	
	// Store execution record in database
	if err := re.storeExecutionRecord(ctx, execCtx, result); err != nil {
		re.logger.Warn("Failed to store execution record", "error", err)
	}
	
	re.logger.Info("Rebalancing execution completed", 
		"execution_id", executionID,
		"success", result.Success,
		"total_cost", result.TotalCost,
	)
	
	return result, nil
}

// ExecuteInstructionBatch executes a batch of instructions with optimal ordering
func (re *RebalanceExecutor) ExecuteInstructionBatch(
	ctx context.Context,
	instructions []RebalanceInstruction,
	batchSize int,
) error {
	re.logger.Info("Executing instruction batch", "count", len(instructions), "batch_size", batchSize)
	
	// Process instructions in batches
	for i := 0; i < len(instructions); i += batchSize {
		end := i + batchSize
		if end > len(instructions) {
			end = len(instructions)
		}
		
		batch := instructions[i:end]
		
		// Execute batch concurrently
		if err := re.executeBatchConcurrently(ctx, batch); err != nil {
			return fmt.Errorf("batch execution failed: %w", err)
		}
		
		// Wait between batches to avoid overwhelming the exchange
		if end < len(instructions) {
			time.Sleep(100 * time.Millisecond)
		}
	}
	
	return nil
}

// GetExecutionStatus returns the current status of an execution
func (re *RebalanceExecutor) GetExecutionStatus(executionID string) (*ExecutionContext, error) {
	re.mu.RLock()
	defer re.mu.RUnlock()
	
	if execCtx, exists := re.activeExecutions[executionID]; exists {
		return execCtx, nil
	}
	
	return nil, fmt.Errorf("execution not found: %s", executionID)
}

// CancelExecution cancels an ongoing execution
func (re *RebalanceExecutor) CancelExecution(executionID string) error {
	re.mu.Lock()
	defer re.mu.Unlock()
	
	if execCtx, exists := re.activeExecutions[executionID]; exists {
		execCtx.Status = "CANCELLED"
		re.logger.Info("Execution cancelled", "execution_id", executionID)
		return nil
	}
	
	return fmt.Errorf("execution not found: %s", executionID)
}

// GetExecutionHistory returns the execution history
func (re *RebalanceExecutor) GetExecutionHistory(limit int) []RebalanceEvent {
	re.mu.RLock()
	defer re.mu.RUnlock()
	
	if limit <= 0 || limit > len(re.executionHistory) {
		limit = len(re.executionHistory)
	}
	
	// Return most recent executions
	start := len(re.executionHistory) - limit
	history := make([]RebalanceEvent, limit)
	copy(history, re.executionHistory[start:])
	
	return history
}

// Private helper methods

func (re *RebalanceExecutor) executeInstructions(
	ctx context.Context,
	execCtx *ExecutionContext,
	options *ExecutionOptions,
) (*ExecutionResult, error) {
	result := &ExecutionResult{
		ExecutionID:       execCtx.ID,
		Success:           true,
		CompletedTrades:   0,
		FailedTrades:      0,
		TotalCost:         0.0,
		ExecutionTime:     time.Duration(0),
		TradeResults:      make([]TradeResult, 0),
	}
	
	// Get pre-execution portfolio value
	preValue, err := re.calculatePortfolioValue(ctx)
	if err != nil {
		re.logger.Warn("Failed to calculate pre-execution portfolio value", "error", err)
	} else {
		result.PreExecutionValue = preValue
	}
	
	// Execute instructions based on strategy
	if options != nil && options.ExecutionStrategy == "CONCURRENT" {
		err = re.executeBatchConcurrently(ctx, execCtx.Instructions)
	} else {
		err = re.executeSequentially(ctx, execCtx.Instructions)
	}
	
	if err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
		return result, err
	}
	
	// Get post-execution portfolio value
	postValue, err := re.calculatePortfolioValue(ctx)
	if err != nil {
		re.logger.Warn("Failed to calculate post-execution portfolio value", "error", err)
	} else {
		result.PostExecutionValue = postValue
	}
	
	// Update result metrics
	result.CompletedTrades = execCtx.CompletedTrades
	result.FailedTrades = execCtx.FailedTrades
	result.TotalCost = execCtx.TotalCost
	result.ExecutionTime = time.Since(execCtx.StartTime)
	
	return result, nil
}

func (re *RebalanceExecutor) executeSequentially(
	ctx context.Context,
	instructions []RebalanceInstruction,
) error {
	for i, instruction := range instructions {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		if err := re.executeInstruction(ctx, &instructions[i]); err != nil {
			re.logger.Error("Failed to execute instruction", 
				"instruction_id", instruction.ID, 
				"error", err,
			)
			continue
		}
		
		// Update progress
		progress := float64(i+1) / float64(len(instructions))
		re.updateExecutionProgress(instruction.ID, progress)
	}
	
	return nil
}

func (re *RebalanceExecutor) executeBatchConcurrently(
	ctx context.Context,
	instructions []RebalanceInstruction,
) error {
	semaphore := make(chan struct{}, re.maxConcurrentTrades)
	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]error, 0)
	
	for i := range instructions {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			if err := re.executeInstruction(ctx, &instructions[idx]); err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
			}
		}(i)
	}
	
	wg.Wait()
	
	if len(errors) > 0 {
		return fmt.Errorf("batch execution had %d errors: %v", len(errors), errors[0])
	}
	
	return nil
}

func (re *RebalanceExecutor) executeInstruction(
	ctx context.Context,
	instruction *RebalanceInstruction,
) error {
	re.logger.Info("Executing rebalance instruction", 
		"instruction_id", instruction.ID,
		"symbol", instruction.Symbol,
		"adjustment", instruction.Adjustment,
	)
	
	instruction.Status = "EXECUTING"
	
	// Determine trade parameters
	side := "BUY"
	if instruction.Adjustment < 0 {
		side = "SELL"
	}
	
	size := math.Abs(instruction.Adjustment)
	
	// Execute trade with retry logic
	var tradeResult *TradeResult
	var err error
	
	for attempt := 0; attempt <= re.retryAttempts; attempt++ {
		tradeResult, err = re.executeTrade(ctx, instruction.Symbol, side, size)
		if err == nil {
			break
		}
		
		if attempt < re.retryAttempts {
			re.logger.Warn("Trade execution failed, retrying", 
				"attempt", attempt+1,
				"error", err,
			)
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
	}
	
	if err != nil {
		instruction.Status = "FAILED"
		instruction.ErrorMessage = err.Error()
		re.updateExecutionStats("", 0, 1)
		return err
	}
	
	// Update instruction with results
	instruction.Status = "COMPLETED"
	now := time.Now()
	instruction.ExecutedAt = &now
	
	// Update execution statistics
	re.updateExecutionStats(instruction.ID, 1, 0)
	
	re.logger.Info("Instruction executed successfully", 
		"instruction_id", instruction.ID,
		"trade_id", tradeResult.TradeID,
		"executed_price", tradeResult.ExecutedPrice,
		"cost", tradeResult.Cost,
	)
	
	return nil
}

func (re *RebalanceExecutor) executeTrade(
	ctx context.Context,
	symbol, side string,
	size float64,
) (*TradeResult, error) {
	// This would integrate with the actual exchange client
	// For now, simulate trade execution
	
	executionPrice := re.getMarketPrice(symbol)
	if executionPrice <= 0 {
		return nil, fmt.Errorf("invalid market price for %s", symbol)
	}
	
	// Simulate slippage
	slippage := re.calculateSlippage(symbol, size)
	if side == "BUY" {
		executionPrice *= (1 + slippage)
	} else {
		executionPrice *= (1 - slippage)
	}
	
	// Check slippage tolerance
	if slippage > re.slippageTolerance {
		return nil, fmt.Errorf("slippage exceeds tolerance: %.4f > %.4f", slippage, re.slippageTolerance)
	}
	
	// Calculate trade cost
	cost := size * executionPrice * 0.001 // 0.1% fee
	
	tradeResult := &TradeResult{
		TradeID:       re.generateTradeID(),
		Symbol:        symbol,
		Side:          side,
		Size:          size,
		ExecutedPrice: executionPrice,
		Cost:          cost,
		Slippage:      slippage,
		ExecutedAt:    time.Now(),
		Success:       true,
	}
	
	return tradeResult, nil
}

func (re *RebalanceExecutor) validateInstructions(
	ctx context.Context,
	instructions []RebalanceInstruction,
) error {
	if len(instructions) == 0 {
		return fmt.Errorf("no instructions provided")
	}
	
	// Validate each instruction
	for _, instruction := range instructions {
		if instruction.Symbol == "" {
			return fmt.Errorf("invalid symbol in instruction %s", instruction.ID)
		}
		
		if instruction.Adjustment == 0 {
			return fmt.Errorf("zero adjustment in instruction %s", instruction.ID)
		}
		
		if math.IsNaN(instruction.Adjustment) || math.IsInf(instruction.Adjustment, 0) {
			return fmt.Errorf("invalid adjustment value in instruction %s", instruction.ID)
		}
	}
	
	return nil
}

func (re *RebalanceExecutor) calculatePortfolioValue(ctx context.Context) (float64, error) {
	query := `
		SELECT SUM(size * current_price) as total_value
		FROM positions 
		WHERE status = 'ACTIVE'
	`
	
	var totalValue float64
	err := re.db.QueryRowContext(ctx, query).Scan(&totalValue)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate portfolio value: %w", err)
	}
	
	return totalValue, nil
}

func (re *RebalanceExecutor) updateExecutionProgress(executionID string, progress float64) {
	re.mu.Lock()
	defer re.mu.Unlock()
	
	if execCtx, exists := re.activeExecutions[executionID]; exists {
		execCtx.Progress = progress
	}
}

func (re *RebalanceExecutor) updateExecutionStats(executionID string, completed, failed int) {
	re.mu.Lock()
	defer re.mu.Unlock()
	
	for _, execCtx := range re.activeExecutions {
		execCtx.CompletedTrades += completed
		execCtx.FailedTrades += failed
	}
}

func (re *RebalanceExecutor) storeExecutionRecord(
	ctx context.Context,
	execCtx *ExecutionContext,
	result *ExecutionResult,
) error {
	query := `
		INSERT INTO rebalance_executions 
		(id, status, start_time, end_time, completed_trades, failed_trades, 
		 total_cost, estimated_cost, success, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err := re.db.ExecContext(ctx, query,
		execCtx.ID,
		execCtx.Status,
		execCtx.StartTime,
		execCtx.EndTime,
		execCtx.CompletedTrades,
		execCtx.FailedTrades,
		execCtx.TotalCost,
		execCtx.EstimatedCost,
		result.Success,
		result.ErrorMessage,
	)
	
	return err
}

// Helper methods

func (re *RebalanceExecutor) generateExecutionID() string {
	return fmt.Sprintf("exec_%d", time.Now().UnixNano())
}

func (re *RebalanceExecutor) generateTradeID() string {
	return fmt.Sprintf("trade_%d", time.Now().UnixNano())
}

func (re *RebalanceExecutor) getMarketPrice(symbol string) float64 {
	// Simplified price retrieval - would integrate with real market data
	return 100.0 // Placeholder
}

func (re *RebalanceExecutor) calculateSlippage(symbol string, size float64) float64 {
	// Simplified slippage calculation
	baseSlippage := 0.001 // 0.1% base slippage
	sizeImpact := size * 0.00001 // Size-based impact
	return baseSlippage + sizeImpact
}

// ExecutionOptions represents options for rebalancing execution
type ExecutionOptions struct {
	ExecutionStrategy string        `json:"execution_strategy"` // SEQUENTIAL, CONCURRENT
	MaxSlippage       float64       `json:"max_slippage"`
	TimeLimit         time.Duration `json:"time_limit"`
	DryRun            bool          `json:"dry_run"`
}

// ExecutionResult represents the result of a rebalancing execution
type ExecutionResult struct {
	ExecutionID        string        `json:"execution_id"`
	Success            bool          `json:"success"`
	ErrorMessage       string        `json:"error_message,omitempty"`
	CompletedTrades    int           `json:"completed_trades"`
	FailedTrades       int           `json:"failed_trades"`
	TotalCost          float64       `json:"total_cost"`
	PreExecutionValue  float64       `json:"pre_execution_value"`
	PostExecutionValue float64       `json:"post_execution_value"`
	ExecutionTime      time.Duration `json:"execution_time"`
	TradeResults       []TradeResult `json:"trade_results"`
}

// TradeResult represents the result of a single trade
type TradeResult struct {
	TradeID       string    `json:"trade_id"`
	Symbol        string    `json:"symbol"`
	Side          string    `json:"side"`
	Size          float64   `json:"size"`
	ExecutedPrice float64   `json:"executed_price"`
	Cost          float64   `json:"cost"`
	Slippage      float64   `json:"slippage"`
	ExecutedAt    time.Time `json:"executed_at"`
	Success       bool      `json:"success"`
	ErrorMessage  string    `json:"error_message,omitempty"`
}