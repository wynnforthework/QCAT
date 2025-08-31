package position

import (
	"context"
	"fmt"
	"math"
	"time"

	"qcat/internal/automation/scheduler/shared"
	"qcat/internal/database"
	"qcat/internal/logger"
)

// TransactionCostCalculator provides transaction cost modeling and estimation
type TransactionCostCalculator struct {
	db     *database.DB
	logger logger.Logger
	config shared.ConfigProvider

	// Cost model parameters
	baseFeeRate      float64
	marketImpactRate float64
	liquidityFactor  float64
	volatilityFactor float64
}

// NewTransactionCostCalculator creates a new transaction cost calculator
func NewTransactionCostCalculator(
	db *database.DB,
	logger logger.Logger,
	config shared.ConfigProvider,
) *TransactionCostCalculator {
	return &TransactionCostCalculator{
		db:               db,
		logger:           logger,
		config:           config,
		baseFeeRate:      config.GetFloat64("transaction_cost.base_fee_rate"),
		marketImpactRate: config.GetFloat64("transaction_cost.market_impact_rate"),
		liquidityFactor:  config.GetFloat64("transaction_cost.liquidity_factor"),
		volatilityFactor: config.GetFloat64("transaction_cost.volatility_factor"),
	}
}

// EstimateTransactionCost estimates the total transaction cost for a trade
func (tcc *TransactionCostCalculator) EstimateTransactionCost(
	ctx context.Context,
	symbol string,
	size float64,
	side string,
) (float64, error) {
	tcc.logger.Info("Estimating transaction cost",
		"symbol", symbol,
		"size", size,
		"side", side,
	)

	// Get market data for cost calculation
	marketData, err := tcc.getMarketData(ctx, symbol)
	if err != nil {
		return 0, fmt.Errorf("failed to get market data: %w", err)
	}

	// Calculate different cost components
	baseFee := tcc.calculateBaseFee(size, marketData.Price)
	marketImpact := tcc.calculateMarketImpact(size, marketData, side)
	bidAskSpread := tcc.calculateBidAskSpreadCost(size, marketData)
	liquidityCost := tcc.calculateLiquidityCost(size, marketData)
	volatilityCost := tcc.calculateVolatilityCost(size, marketData)

	totalCost := baseFee + marketImpact + bidAskSpread + liquidityCost + volatilityCost

	tcc.logger.Info("Transaction cost breakdown",
		"symbol", symbol,
		"base_fee", baseFee,
		"market_impact", marketImpact,
		"bid_ask_spread", bidAskSpread,
		"liquidity_cost", liquidityCost,
		"volatility_cost", volatilityCost,
		"total_cost", totalCost,
	)

	return totalCost, nil
}

// EstimateBatchTransactionCost estimates costs for multiple transactions
func (tcc *TransactionCostCalculator) EstimateBatchTransactionCost(
	ctx context.Context,
	instructions []RebalanceInstruction,
) (map[string]float64, float64, error) {
	tcc.logger.Info("Estimating batch transaction costs", "count", len(instructions))

	costs := make(map[string]float64)
	totalCost := 0.0

	for _, instruction := range instructions {
		side := "BUY"
		if instruction.Adjustment < 0 {
			side = "SELL"
		}

		cost, err := tcc.EstimateTransactionCost(
			ctx,
			instruction.Symbol,
			math.Abs(instruction.Adjustment),
			side,
		)
		if err != nil {
			tcc.logger.Warn("Failed to estimate cost for instruction",
				"instruction_id", instruction.ID,
				"error", err,
			)
			continue
		}

		costs[instruction.ID] = cost
		totalCost += cost
	}

	// Apply batch discount if applicable
	batchDiscount := tcc.calculateBatchDiscount(len(instructions), totalCost)
	totalCost -= batchDiscount

	tcc.logger.Info("Batch transaction cost estimation complete",
		"total_cost", totalCost,
		"batch_discount", batchDiscount,
	)

	return costs, totalCost, nil
}

// OptimizeExecutionOrder optimizes the order of trade execution to minimize costs
func (tcc *TransactionCostCalculator) OptimizeExecutionOrder(
	ctx context.Context,
	instructions []RebalanceInstruction,
) ([]RebalanceInstruction, error) {
	tcc.logger.Info("Optimizing execution order for cost minimization")

	// Calculate cost impact for each instruction
	costImpacts := make(map[string]float64)

	for _, instruction := range instructions {
		impact, err := tcc.calculateExecutionImpact(ctx, instruction)
		if err != nil {
			tcc.logger.Warn("Failed to calculate execution impact",
				"instruction_id", instruction.ID,
				"error", err,
			)
			continue
		}
		costImpacts[instruction.ID] = impact
	}

	// Sort instructions by cost impact and urgency
	optimizedInstructions := make([]RebalanceInstruction, len(instructions))
	copy(optimizedInstructions, instructions)

	// Sort by priority first, then by cost impact
	for i := 0; i < len(optimizedInstructions)-1; i++ {
		for j := i + 1; j < len(optimizedInstructions); j++ {
			instr1 := optimizedInstructions[i]
			instr2 := optimizedInstructions[j]

			// Higher priority (lower number) comes first
			if instr1.Priority > instr2.Priority {
				optimizedInstructions[i], optimizedInstructions[j] = instr2, instr1
				continue
			}

			// Same priority, sort by cost impact (lower impact first)
			if instr1.Priority == instr2.Priority {
				impact1 := costImpacts[instr1.ID]
				impact2 := costImpacts[instr2.ID]
				if impact1 > impact2 {
					optimizedInstructions[i], optimizedInstructions[j] = instr2, instr1
				}
			}
		}
	}

	tcc.logger.Info("Execution order optimization complete")
	return optimizedInstructions, nil
}

// CalculateOptimalTradeSize calculates optimal trade size to minimize market impact
func (tcc *TransactionCostCalculator) CalculateOptimalTradeSize(
	ctx context.Context,
	symbol string,
	targetSize float64,
	maxSlices int,
) ([]float64, error) {
	tcc.logger.Info("Calculating optimal trade size slicing",
		"symbol", symbol,
		"target_size", targetSize,
		"max_slices", maxSlices,
	)

	marketData, err := tcc.getMarketData(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get market data: %w", err)
	}

	// Calculate optimal slice size based on market impact model
	optimalSliceSize := tcc.calculateOptimalSliceSize(targetSize, marketData)

	// Determine number of slices
	numSlices := int(math.Ceil(targetSize / optimalSliceSize))
	if numSlices > maxSlices {
		numSlices = maxSlices
		optimalSliceSize = targetSize / float64(numSlices)
	}

	// Create slice sizes
	slices := make([]float64, numSlices)
	remaining := targetSize

	for i := 0; i < numSlices-1; i++ {
		slices[i] = optimalSliceSize
		remaining -= optimalSliceSize
	}
	slices[numSlices-1] = remaining // Last slice gets the remainder

	tcc.logger.Info("Optimal trade slicing calculated",
		"num_slices", numSlices,
		"slice_size", optimalSliceSize,
	)

	return slices, nil
}

// Private helper methods

func (tcc *TransactionCostCalculator) getMarketData(ctx context.Context, symbol string) (*MarketDataSnapshot, error) {
	query := `
		SELECT price, volume, bid_price, ask_price, volatility, liquidity_score
		FROM market_data_snapshot 
		WHERE symbol = ? 
		ORDER BY timestamp DESC 
		LIMIT 1
	`

	var snapshot MarketDataSnapshot
	err := tcc.db.QueryRowContext(ctx, query, symbol).Scan(
		&snapshot.Price,
		&snapshot.Volume,
		&snapshot.BidPrice,
		&snapshot.AskPrice,
		&snapshot.Volatility,
		&snapshot.LiquidityScore,
	)

	if err != nil {
		// Fallback to basic market data if snapshot not available
		return tcc.getBasicMarketData(ctx, symbol)
	}

	snapshot.Symbol = symbol
	snapshot.Timestamp = time.Now()

	return &snapshot, nil
}

func (tcc *TransactionCostCalculator) getBasicMarketData(ctx context.Context, symbol string) (*MarketDataSnapshot, error) {
	query := `
		SELECT price, volume
		FROM market_data 
		WHERE symbol = ? 
		ORDER BY timestamp DESC 
		LIMIT 1
	`

	var snapshot MarketDataSnapshot
	err := tcc.db.QueryRowContext(ctx, query, symbol).Scan(
		&snapshot.Price,
		&snapshot.Volume,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get market data: %w", err)
	}

	// Estimate missing fields
	snapshot.Symbol = symbol
	snapshot.BidPrice = snapshot.Price * 0.999 // Estimate 0.1% spread
	snapshot.AskPrice = snapshot.Price * 1.001
	snapshot.Volatility = 0.02    // Default 2% volatility
	snapshot.LiquidityScore = 0.5 // Default medium liquidity
	snapshot.Timestamp = time.Now()

	return &snapshot, nil
}

func (tcc *TransactionCostCalculator) calculateBaseFee(size, price float64) float64 {
	return size * price * tcc.baseFeeRate
}

func (tcc *TransactionCostCalculator) calculateMarketImpact(
	size float64,
	marketData *MarketDataSnapshot,
	side string,
) float64 {
	// Market impact model: impact = k * (size / volume)^0.5
	volumeRatio := size / math.Max(marketData.Volume, 1)
	impact := tcc.marketImpactRate * math.Sqrt(volumeRatio)

	// Adjust for side (selling typically has higher impact)
	if side == "SELL" {
		impact *= 1.2
	}

	return impact * size * marketData.Price
}

func (tcc *TransactionCostCalculator) calculateBidAskSpreadCost(
	size float64,
	marketData *MarketDataSnapshot,
) float64 {
	spread := marketData.AskPrice - marketData.BidPrice
	spreadCost := spread * size * 0.5 // Assume crossing half the spread
	return spreadCost
}

func (tcc *TransactionCostCalculator) calculateLiquidityCost(
	size float64,
	marketData *MarketDataSnapshot,
) float64 {
	// Higher cost for lower liquidity
	liquidityPenalty := (1.0 - marketData.LiquidityScore) * tcc.liquidityFactor
	return liquidityPenalty * size * marketData.Price
}

func (tcc *TransactionCostCalculator) calculateVolatilityCost(
	size float64,
	marketData *MarketDataSnapshot,
) float64 {
	// Higher cost during high volatility periods
	volatilityCost := marketData.Volatility * tcc.volatilityFactor
	return volatilityCost * size * marketData.Price
}

func (tcc *TransactionCostCalculator) calculateBatchDiscount(
	numTrades int,
	totalCost float64,
) float64 {
	// Volume discount for batch trades
	if numTrades < 5 {
		return 0
	}

	discountRate := math.Min(0.1, float64(numTrades-5)*0.01) // Up to 10% discount
	return totalCost * discountRate
}

func (tcc *TransactionCostCalculator) calculateExecutionImpact(
	ctx context.Context,
	instruction RebalanceInstruction,
) (float64, error) {
	marketData, err := tcc.getMarketData(ctx, instruction.Symbol)
	if err != nil {
		return 0, err
	}

	// Calculate urgency factor based on priority
	urgencyFactor := 1.0
	if instruction.Priority > 0 {
		urgencyFactor = 1.0 / float64(instruction.Priority)
	}

	// Calculate size factor
	sizeFactor := 0.0
	if marketData.Volume > 0 {
		sizeFactor = math.Abs(instruction.Adjustment) / marketData.Volume
	}

	// Calculate timing factor (higher impact during market hours)
	timingFactor := tcc.calculateTimingFactor()

	impact := urgencyFactor * sizeFactor * timingFactor
	return impact, nil
}

func (tcc *TransactionCostCalculator) calculateOptimalSliceSize(
	targetSize float64,
	marketData *MarketDataSnapshot,
) float64 {
	// Optimal slice size based on square root law
	// Minimize: fixed_cost * num_slices + market_impact * sqrt(slice_size)

	if marketData.Price == 0 || targetSize == 0 {
		return targetSize * 0.1 // Default to 10% of target size
	}

	fixedCost := marketData.Price * tcc.baseFeeRate
	impactCoeff := tcc.marketImpactRate * marketData.Price

	// Check for division by zero
	if impactCoeff == 0 {
		return targetSize * 0.1 // Default to 10% of target size
	}

	// Optimal slice size = (2 * fixed_cost / impact_coeff)^(2/3) * target_size^(1/3)
	optimalSize := math.Pow(2*fixedCost/impactCoeff, 2.0/3.0) * math.Pow(targetSize, 1.0/3.0)

	// Ensure reasonable bounds
	minSize := targetSize * 0.01 // At least 1% of target
	maxSize := targetSize * 0.5  // At most 50% of target

	return math.Max(minSize, math.Min(maxSize, optimalSize))
}

func (tcc *TransactionCostCalculator) calculateTimingFactor() float64 {
	now := time.Now()
	hour := now.Hour()

	// Higher impact during market open/close hours
	if (hour >= 9 && hour <= 10) || (hour >= 15 && hour <= 16) {
		return 1.5
	}

	// Lower impact during off-hours
	if hour < 6 || hour > 20 {
		return 0.8
	}

	return 1.0 // Normal hours
}

// MarketDataSnapshot represents a snapshot of market data for cost calculation
type MarketDataSnapshot struct {
	Symbol         string    `json:"symbol"`
	Price          float64   `json:"price"`
	Volume         float64   `json:"volume"`
	BidPrice       float64   `json:"bid_price"`
	AskPrice       float64   `json:"ask_price"`
	Volatility     float64   `json:"volatility"`
	LiquidityScore float64   `json:"liquidity_score"`
	Timestamp      time.Time `json:"timestamp"`
}
