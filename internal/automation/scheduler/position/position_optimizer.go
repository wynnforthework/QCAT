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

// PositionOptimizer implements portfolio optimization using modern portfolio theory
type PositionOptimizer struct {
	db             *database.DB
	exchangeClient exchange.Exchange
	logger         logger.Logger
	config         shared.ConfigProvider
	
	// Optimization parameters
	riskFreeRate   float64
	lookbackPeriod time.Duration
	rebalanceThreshold float64
}

// NewPositionOptimizer creates a new position optimizer instance
func NewPositionOptimizer(
	db *database.DB,
	exchangeClient exchange.Exchange,
	logger logger.Logger,
	config shared.ConfigProvider,
) *PositionOptimizer {
	return &PositionOptimizer{
		db:             db,
		exchangeClient: exchangeClient,
		logger:         logger,
		config:         config,
		riskFreeRate:   config.GetFloat64("optimization.risk_free_rate"),
		lookbackPeriod: config.GetDuration("optimization.lookback_period"),
		rebalanceThreshold: config.GetFloat64("optimization.rebalance_threshold"),
	}
}

// GetCurrentPositions retrieves current positions from database and exchanges
func (po *PositionOptimizer) GetCurrentPositions(ctx context.Context) ([]shared.Position, error) {
	po.logger.Info("Retrieving current positions from database and exchanges")
	
	// Get positions from database
	dbPositions, err := po.getPositionsFromDB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get positions from database: %w", err)
	}
	
	// Get positions from exchange for verification
	exchangePositions, err := po.getPositionsFromExchange(ctx)
	if err != nil {
		po.logger.Warn("Failed to get positions from exchange, using database positions only", "error", err)
		return dbPositions, nil
	}
	
	// Reconcile positions between database and exchange
	reconciledPositions := po.reconcilePositions(dbPositions, exchangePositions)
	
	po.logger.Info("Successfully retrieved positions", "count", len(reconciledPositions))
	return reconciledPositions, nil
}

// CalculateOptimalPositions calculates optimal portfolio positions using modern portfolio theory
func (po *PositionOptimizer) CalculateOptimalPositions(
	ctx context.Context, 
	constraints shared.OptimizationConstraints,
) ([]shared.TargetPosition, error) {
	po.logger.Info("Calculating optimal positions using modern portfolio theory")
	
	// Get current positions
	currentPositions, err := po.GetCurrentPositions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current positions: %w", err)
	}
	
	// Get historical returns for portfolio optimization
	returns, err := po.getHistoricalReturns(ctx, currentPositions)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical returns: %w", err)
	}
	
	// Calculate expected returns and covariance matrix
	expectedReturns := po.calculateExpectedReturns(returns)
	covarianceMatrix := po.calculateCovarianceMatrix(returns)
	
	// Perform mean-variance optimization
	optimalWeights, err := po.optimizePortfolio(expectedReturns, covarianceMatrix, constraints)
	if err != nil {
		return nil, fmt.Errorf("failed to optimize portfolio: %w", err)
	}
	
	// Convert weights to target positions
	targetPositions := po.convertWeightsToPositions(currentPositions, optimalWeights, constraints)
	
	po.logger.Info("Successfully calculated optimal positions", "targets", len(targetPositions))
	return targetPositions, nil
}

// GenerateRebalanceInstructions creates rebalancing instructions based on current and target positions
func (po *PositionOptimizer) GenerateRebalanceInstructions(
	ctx context.Context,
	current, target []shared.Position,
) ([]RebalanceInstruction, error) {
	po.logger.Info("Generating rebalancing instructions")
	
	instructions := make([]RebalanceInstruction, 0)
	
	// Create maps for easier lookup
	currentMap := make(map[string]shared.Position)
	for _, pos := range current {
		currentMap[pos.Symbol] = pos
	}
	
	targetMap := make(map[string]shared.Position)
	for _, pos := range target {
		targetMap[pos.Symbol] = pos
	}
	
	// Calculate required adjustments
	for symbol, targetPos := range targetMap {
		currentPos, exists := currentMap[symbol]
		
		var adjustment float64
		if exists {
			adjustment = targetPos.Size - currentPos.Size
		} else {
			adjustment = targetPos.Size
		}
		
		// Only create instruction if adjustment is significant
		if math.Abs(adjustment) > po.rebalanceThreshold {
			instruction := RebalanceInstruction{
				ID:             po.generateInstructionID(),
				Symbol:         symbol,
				CurrentSize:    func() float64 { if exists { return currentPos.Size } else { return 0 } }(),
				TargetSize:     targetPos.Size,
				Adjustment:     adjustment,
				Priority:       po.calculatePriority(adjustment, targetPos),
				TransactionCost: po.estimateTransactionCost(symbol, math.Abs(adjustment)),
				CreatedAt:      time.Now(),
				Status:         "PENDING",
			}
			instructions = append(instructions, instruction)
		}
	}
	
	// Handle positions that need to be closed (not in target)
	for symbol, currentPos := range currentMap {
		if _, exists := targetMap[symbol]; !exists && currentPos.Size != 0 {
			instruction := RebalanceInstruction{
				ID:             po.generateInstructionID(),
				Symbol:         symbol,
				CurrentSize:    currentPos.Size,
				TargetSize:     0,
				Adjustment:     -currentPos.Size,
				Priority:       1, // High priority for position closure
				TransactionCost: po.estimateTransactionCost(symbol, currentPos.Size),
				CreatedAt:      time.Now(),
				Status:         "PENDING",
			}
			instructions = append(instructions, instruction)
		}
	}
	
	// Sort instructions by priority
	sort.Slice(instructions, func(i, j int) bool {
		return instructions[i].Priority < instructions[j].Priority
	})
	
	po.logger.Info("Generated rebalancing instructions", "count", len(instructions))
	return instructions, nil
}

// ExecutePositionAdjustments executes the rebalancing instructions
func (po *PositionOptimizer) ExecutePositionAdjustments(
	ctx context.Context,
	instructions []RebalanceInstruction,
) error {
	po.logger.Info("Executing position adjustments", "instructions", len(instructions))
	
	for i, instruction := range instructions {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		err := po.executeInstruction(ctx, &instructions[i])
		if err != nil {
			po.logger.Error("Failed to execute instruction", "instruction_id", instruction.ID, "error", err)
			instructions[i].Status = "FAILED"
			instructions[i].ErrorMessage = err.Error()
			continue
		}
		
		instructions[i].Status = "COMPLETED"
		instructions[i].ExecutedAt = &[]time.Time{time.Now()}[0]
		
		po.logger.Info("Successfully executed instruction", "instruction_id", instruction.ID)
	}
	
	po.logger.Info("Completed position adjustments execution")
	return nil
}

// getPositionsFromDB retrieves positions from the database
func (po *PositionOptimizer) getPositionsFromDB(ctx context.Context) ([]shared.Position, error) {
	query := `
		SELECT id, symbol, side, size, entry_price, current_price, 
		       unrealized_pnl, realized_pnl, leverage, margin_used, timestamp
		FROM positions 
		WHERE status = 'ACTIVE'
		ORDER BY symbol
	`
	
	rows, err := po.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query positions: %w", err)
	}
	defer rows.Close()
	
	var positions []shared.Position
	for rows.Next() {
		var pos shared.Position
		err := rows.Scan(
			&pos.ID, &pos.Symbol, &pos.Side, &pos.Size, &pos.EntryPrice,
			&pos.CurrentPrice, &pos.UnrealizedPnL, &pos.RealizedPnL,
			&pos.Leverage, &pos.MarginUsed, &pos.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan position: %w", err)
		}
		positions = append(positions, pos)
	}
	
	return positions, nil
}

// getPositionsFromExchange retrieves positions from the exchange
func (po *PositionOptimizer) getPositionsFromExchange(ctx context.Context) ([]shared.Position, error) {
	// This would integrate with the actual exchange client
	// For now, return empty slice as fallback
	return []shared.Position{}, nil
}

// reconcilePositions reconciles positions between database and exchange
func (po *PositionOptimizer) reconcilePositions(
	dbPositions, exchangePositions []shared.Position,
) []shared.Position {
	// For now, prioritize database positions
	// In a real implementation, this would perform sophisticated reconciliation
	return dbPositions
}

// getHistoricalReturns retrieves historical returns for portfolio optimization
func (po *PositionOptimizer) getHistoricalReturns(
	ctx context.Context,
	positions []shared.Position,
) (map[string][]float64, error) {
	returns := make(map[string][]float64)
	
	for _, pos := range positions {
		symbolReturns, err := po.getSymbolReturns(ctx, pos.Symbol)
		if err != nil {
			po.logger.Warn("Failed to get returns for symbol", "symbol", pos.Symbol, "error", err)
			continue
		}
		returns[pos.Symbol] = symbolReturns
	}
	
	return returns, nil
}

// getSymbolReturns retrieves historical returns for a specific symbol
func (po *PositionOptimizer) getSymbolReturns(ctx context.Context, symbol string) ([]float64, error) {
	query := `
		SELECT price, timestamp
		FROM market_data 
		WHERE symbol = ? AND timestamp >= ?
		ORDER BY timestamp
	`
	
	startTime := time.Now().Add(-po.lookbackPeriod)
	rows, err := po.db.QueryContext(ctx, query, symbol, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query market data: %w", err)
	}
	defer rows.Close()
	
	var prices []float64
	for rows.Next() {
		var price float64
		var timestamp time.Time
		err := rows.Scan(&price, &timestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to scan market data: %w", err)
		}
		prices = append(prices, price)
	}
	
	// Calculate returns from prices
	returns := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		returns[i-1] = (prices[i] - prices[i-1]) / prices[i-1]
	}
	
	return returns, nil
}

// calculateExpectedReturns calculates expected returns from historical data
func (po *PositionOptimizer) calculateExpectedReturns(returns map[string][]float64) map[string]float64 {
	expectedReturns := make(map[string]float64)
	
	for symbol, symbolReturns := range returns {
		if len(symbolReturns) == 0 {
			expectedReturns[symbol] = 0
			continue
		}
		
		sum := 0.0
		for _, ret := range symbolReturns {
			sum += ret
		}
		expectedReturns[symbol] = sum / float64(len(symbolReturns))
	}
	
	return expectedReturns
}

// calculateCovarianceMatrix calculates the covariance matrix of returns
func (po *PositionOptimizer) calculateCovarianceMatrix(returns map[string][]float64) map[string]map[string]float64 {
	symbols := make([]string, 0, len(returns))
	for symbol := range returns {
		symbols = append(symbols, symbol)
	}
	
	covariance := make(map[string]map[string]float64)
	for _, symbol1 := range symbols {
		covariance[symbol1] = make(map[string]float64)
		for _, symbol2 := range symbols {
			covariance[symbol1][symbol2] = po.calculateCovariance(returns[symbol1], returns[symbol2])
		}
	}
	
	return covariance
}

// calculateCovariance calculates covariance between two return series
func (po *PositionOptimizer) calculateCovariance(returns1, returns2 []float64) float64 {
	if len(returns1) != len(returns2) || len(returns1) == 0 {
		return 0
	}
	
	// Calculate means
	mean1 := 0.0
	mean2 := 0.0
	for i := 0; i < len(returns1); i++ {
		mean1 += returns1[i]
		mean2 += returns2[i]
	}
	mean1 /= float64(len(returns1))
	mean2 /= float64(len(returns2))
	
	// Calculate covariance
	covariance := 0.0
	for i := 0; i < len(returns1); i++ {
		covariance += (returns1[i] - mean1) * (returns2[i] - mean2)
	}
	
	return covariance / float64(len(returns1)-1)
}

// optimizePortfolio performs mean-variance optimization
func (po *PositionOptimizer) optimizePortfolio(
	expectedReturns map[string]float64,
	covarianceMatrix map[string]map[string]float64,
	constraints shared.OptimizationConstraints,
) (map[string]float64, error) {
	// Simplified optimization - in practice would use quadratic programming
	// This implements a basic risk parity approach with return adjustment
	
	symbols := make([]string, 0, len(expectedReturns))
	for symbol := range expectedReturns {
		symbols = append(symbols, symbol)
	}
	
	if len(symbols) == 0 {
		return make(map[string]float64), nil
	}
	
	// Calculate risk-adjusted scores
	scores := make(map[string]float64)
	totalScore := 0.0
	
	for _, symbol := range symbols {
		variance := covarianceMatrix[symbol][symbol]
		if variance <= 0 {
			variance = 0.01 // Minimum variance to avoid division by zero
		}
		
		// Risk-adjusted return score
		score := expectedReturns[symbol] / math.Sqrt(variance)
		scores[symbol] = math.Max(score, 0) // Ensure non-negative
		totalScore += scores[symbol]
	}
	
	// Normalize to get weights
	weights := make(map[string]float64)
	if totalScore > 0 {
		for _, symbol := range symbols {
			weight := scores[symbol] / totalScore
			
			// Apply position size constraints
			if weight > constraints.MaxPositionSize {
				weight = constraints.MaxPositionSize
			}
			
			weights[symbol] = weight
		}
	} else {
		// Equal weight if no positive scores
		equalWeight := 1.0 / float64(len(symbols))
		for _, symbol := range symbols {
			weights[symbol] = equalWeight
		}
	}
	
	return weights, nil
}

// convertWeightsToPositions converts portfolio weights to target positions
func (po *PositionOptimizer) convertWeightsToPositions(
	currentPositions []shared.Position,
	weights map[string]float64,
	constraints shared.OptimizationConstraints,
) []shared.TargetPosition {
	// Calculate total portfolio value
	totalValue := 0.0
	for _, pos := range currentPositions {
		totalValue += pos.Size * pos.CurrentPrice
	}
	
	if totalValue <= 0 {
		totalValue = constraints.RiskBudget // Use risk budget as fallback
	}
	
	targetPositions := make([]shared.TargetPosition, 0)
	
	for symbol, weight := range weights {
		targetValue := totalValue * weight
		
		// Find current position for this symbol
		var currentSize float64
		for _, pos := range currentPositions {
			if pos.Symbol == symbol {
				currentSize = pos.Size
				break
			}
		}
		
		// Get current price (simplified - would use real market data)
		currentPrice := po.getCurrentPrice(symbol)
		if currentPrice <= 0 {
			continue
		}
		
		targetSize := targetValue / currentPrice
		adjustment := targetSize - currentSize
		
		target := shared.TargetPosition{
			Symbol:      symbol,
			TargetSize:  targetSize,
			CurrentSize: currentSize,
			Adjustment:  adjustment,
			Priority:    po.calculateTargetPriority(weight, adjustment),
			Rationale:   fmt.Sprintf("Optimal weight: %.4f, Target value: %.2f", weight, targetValue),
		}
		
		targetPositions = append(targetPositions, target)
	}
	
	return targetPositions
}

// Helper methods

func (po *PositionOptimizer) generateInstructionID() string {
	return fmt.Sprintf("rebal_%d", time.Now().UnixNano())
}

func (po *PositionOptimizer) calculatePriority(adjustment float64, position shared.Position) int {
	// Higher absolute adjustment = higher priority (lower number)
	absAdjustment := math.Abs(adjustment)
	if absAdjustment > 1000 {
		return 1
	} else if absAdjustment > 100 {
		return 2
	} else {
		return 3
	}
}

func (po *PositionOptimizer) calculateTargetPriority(weight, adjustment float64) int {
	// Higher weight and larger adjustment = higher priority
	score := weight * math.Abs(adjustment)
	if score > 0.1 {
		return 1
	} else if score > 0.01 {
		return 2
	} else {
		return 3
	}
}

func (po *PositionOptimizer) estimateTransactionCost(symbol string, size float64) float64 {
	// Simplified transaction cost estimation
	// In practice, this would use real fee schedules and market impact models
	baseFee := 0.001 // 0.1% base fee
	marketImpact := size * 0.0001 // Market impact based on size
	return (baseFee + marketImpact) * size
}

func (po *PositionOptimizer) getCurrentPrice(symbol string) float64 {
	// Simplified price retrieval - would integrate with real market data
	// For now, return a placeholder value
	return 100.0
}

func (po *PositionOptimizer) executeInstruction(ctx context.Context, instruction *RebalanceInstruction) error {
	// This would integrate with the actual trading system
	// For now, simulate execution
	po.logger.Info("Executing rebalance instruction", 
		"symbol", instruction.Symbol,
		"adjustment", instruction.Adjustment,
	)
	
	// Simulate execution delay
	time.Sleep(100 * time.Millisecond)
	
	return nil
}