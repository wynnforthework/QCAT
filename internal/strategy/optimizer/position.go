package optimizer

import (
	"context"
	"fmt"
	"math"

	"qcat/internal/exchange"
)

// PositionOptimizer optimizes position sizes
type PositionOptimizer struct {
	exchange exchange.Exchange
}

// NewPositionOptimizer creates a new position optimizer
func NewPositionOptimizer(ex exchange.Exchange) *PositionOptimizer {
	return &PositionOptimizer{
		exchange: ex,
	}
}

// PositionConfig represents position optimization configuration
type PositionConfig struct {
	Symbol       string
	RiskBudget   float64
	TargetVol    float64
	RealizedVol  float64
	MaxWeight    float64
	AccountValue float64
}

// OptimizePosition calculates optimal position size
func (o *PositionOptimizer) OptimizePosition(ctx context.Context, cfg *PositionConfig) (*PositionResult, error) {
	// 获取合约信息
	symbolInfo, err := o.exchange.GetSymbolInfo(ctx, cfg.Symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get symbol info: %w", err)
	}

	// 计算理论权重
	weight := cfg.RiskBudget * cfg.TargetVol / cfg.RealizedVol
	if weight > cfg.MaxWeight {
		weight = cfg.MaxWeight
	}

	// 计算目标仓位价值
	targetValue := cfg.AccountValue * weight

	// 计算合约张数
	contractValue := symbolInfo.ContractSize // 使用 ContractSize 字段替代 ContractValue
	if contractValue <= 0 {
		return nil, fmt.Errorf("invalid contract size")
	}

	rawQuantity := targetValue / contractValue

	// 根据最小变动值调整
	stepSize := symbolInfo.QuantityStepSize
	if stepSize <= 0 {
		return nil, fmt.Errorf("invalid step size")
	}

	// 向下取整到最接近的合规张数
	quantity := math.Floor(rawQuantity/stepSize) * stepSize

	// 检查最小和最大限制
	if quantity < symbolInfo.MinQuantity {
		quantity = 0 // 如果不够最小张数，就不开仓
	} else if quantity > symbolInfo.MaxQuantity {
		quantity = symbolInfo.MaxQuantity
	}

	return &PositionResult{
		Symbol:         cfg.Symbol,
		TargetQuantity: quantity,
		ActualWeight:   (quantity * contractValue) / cfg.AccountValue,
		ContractValue:  contractValue,
	}, nil
}

// PositionResult represents position optimization result
type PositionResult struct {
	Symbol         string
	TargetQuantity float64
	ActualWeight   float64
	ContractValue  float64
}

// CalculateVolatilityTarget calculates volatility target
func (o *PositionOptimizer) CalculateVolatilityTarget(returns []float64, lookback int) float64 {
	if len(returns) < lookback {
		return 0
	}

	// 使用过去lookback期的实现波动率作为目标
	var sumSquared float64
	mean := 0.0

	// 计算均值
	for i := len(returns) - lookback; i < len(returns); i++ {
		mean += returns[i]
	}
	mean /= float64(lookback)

	// 计算方差
	for i := len(returns) - lookback; i < len(returns); i++ {
		diff := returns[i] - mean
		sumSquared += diff * diff
	}

	// 年化波动率（假设日度收益率）
	dailyVol := math.Sqrt(sumSquared / float64(lookback-1))
	annualVol := dailyVol * math.Sqrt(252)

	return annualVol
}

// CalculateRiskBudget calculates risk budget based on performance metrics
func (o *PositionOptimizer) CalculateRiskBudget(metrics *PerformanceMetrics) float64 {
	// 使用Sharpe比率和最大回撤来调整风险预算
	if metrics.MaxDrawdown >= 0.5 { // 如果最大回撤超过50%，不分配风险预算
		return 0
	}

	// 基础风险预算
	base := 0.1 // 10%基础分配

	// 根据Sharpe比率调整
	if metrics.SharpeRatio > 0 {
		base *= (1 + metrics.SharpeRatio)
	}

	// 根据最大回撤调整
	drawdownFactor := 1 - metrics.MaxDrawdown
	base *= drawdownFactor

	// 确保风险预算在合理范围内
	if base > 0.5 { // 最大50%风险预算
		base = 0.5
	}

	return base
}

// PerformanceMetrics represents strategy performance metrics
type PerformanceMetrics struct {
	SharpeRatio  float64
	MaxDrawdown  float64
	WinRate      float64
	ProfitFactor float64
}

// ValidatePosition validates position parameters
func (o *PositionOptimizer) ValidatePosition(ctx context.Context, symbol string, quantity float64) error {
	symbolInfo, err := o.exchange.GetSymbolInfo(ctx, symbol)
	if err != nil {
		return fmt.Errorf("failed to get symbol info: %w", err)
	}

	// 检查最小数量
	if quantity < symbolInfo.MinQuantity {
		return fmt.Errorf("quantity %f is less than minimum %f", quantity, symbolInfo.MinQuantity)
	}

	// 检查最大数量
	if quantity > symbolInfo.MaxQuantity {
		return fmt.Errorf("quantity %f is greater than maximum %f", quantity, symbolInfo.MaxQuantity)
	}

	// 检查步长
	if math.Mod(quantity, symbolInfo.QuantityStepSize) != 0 {
		return fmt.Errorf("quantity %f is not multiple of step size %f", quantity, symbolInfo.QuantityStepSize)
	}

	return nil
}
