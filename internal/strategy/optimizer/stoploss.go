package optimizer

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"qcat/internal/exchange"
)

// StopLossOptimizer optimizes stop loss parameters
type StopLossOptimizer struct {
	exchange exchange.Exchange
	stops    map[string]*StopLossConfig
	mu       sync.RWMutex
}

// StopLossConfig represents stop loss configuration
type StopLossConfig struct {
	Symbol          string
	HardStopLoss    float64       // 硬止损比例
	ATRMultiplier   float64       // ATR倍数
	ATRPeriod       int           // ATR周期
	VolStopMultiple float64       // 波动率止损倍数
	TimeLimit       time.Duration // 时间止损
	DrawdownLimit   float64       // 资金曲线回撤限制
	TrailingStop    float64       // 移动止损回调比例
	ParabolicAF     float64       // Parabolic SAR加速因子
	UpdatedAt       time.Time
}

// NewStopLossOptimizer creates a new stop loss optimizer
func NewStopLossOptimizer(ex exchange.Exchange) *StopLossOptimizer {
	return &StopLossOptimizer{
		exchange: ex,
		stops:    make(map[string]*StopLossConfig),
	}
}

// SetStopLoss sets stop loss configuration
func (o *StopLossOptimizer) SetStopLoss(symbol string, config *StopLossConfig) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if config.HardStopLoss <= 0 || config.HardStopLoss >= 1 {
		return fmt.Errorf("invalid hard stop loss: must be between 0 and 1")
	}

	config.UpdatedAt = time.Now()
	o.stops[symbol] = config
	return nil
}

// CheckStopLoss checks if any stop loss should be triggered
func (o *StopLossOptimizer) CheckStopLoss(ctx context.Context, symbol string, price float64, metrics *TradeMetrics) (*StopLossSignal, error) {
	o.mu.RLock()
	config, exists := o.stops[symbol]
	o.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("stop loss config not found for symbol: %s", symbol)
	}

	signal := &StopLossSignal{
		Symbol: symbol,
		Price:  price,
	}

	// 检查硬止损
	if metrics.UnrealizedPnL <= -config.HardStopLoss {
		signal.Type = StopLossTypeHard
		signal.Triggered = true
		return signal, nil
	}

	// 检查ATR止损
	atrStop := price - (config.ATRMultiplier * metrics.ATR)
	if price <= atrStop {
		signal.Type = StopLossTypeATR
		signal.Triggered = true
		return signal, nil
	}

	// 检查波动率止损
	volStop := price - (config.VolStopMultiple * metrics.Volatility)
	if price <= volStop {
		signal.Type = StopLossTypeVolatility
		signal.Triggered = true
		return signal, nil
	}

	// 检查时间止损
	if time.Since(metrics.EntryTime) >= config.TimeLimit {
		signal.Type = StopLossTypeTime
		signal.Triggered = true
		return signal, nil
	}

	// 检查资金曲线回撤
	if metrics.Drawdown >= config.DrawdownLimit {
		signal.Type = StopLossTypeDrawdown
		signal.Triggered = true
		return signal, nil
	}

	// 检查移动止损
	if metrics.HighPrice > 0 {
		trailingStop := metrics.HighPrice * (1 - config.TrailingStop)
		if price <= trailingStop {
			signal.Type = StopLossTypeTrailing
			signal.Triggered = true
			return signal, nil
		}
	}

	// 检查Parabolic SAR
	if metrics.ParabolicSAR > 0 && price <= metrics.ParabolicSAR {
		signal.Type = StopLossTypeParabolic
		signal.Triggered = true
		return signal, nil
	}

	return signal, nil
}

// StopLossSignal represents a stop loss signal
type StopLossSignal struct {
	Symbol    string
	Type      StopLossType
	Price     float64
	Triggered bool
}

// StopLossType represents the type of stop loss
type StopLossType int

const (
	StopLossTypeHard StopLossType = iota
	StopLossTypeATR
	StopLossTypeVolatility
	StopLossTypeTime
	StopLossTypeDrawdown
	StopLossTypeTrailing
	StopLossTypeParabolic
)

// TradeMetrics represents trade metrics for stop loss calculation
type TradeMetrics struct {
	EntryPrice    float64
	HighPrice     float64
	UnrealizedPnL float64
	ATR           float64
	Volatility    float64
	Drawdown      float64
	ParabolicSAR  float64
	EntryTime     time.Time
}

// OptimizeStopLoss optimizes stop loss parameters based on historical data
func (o *StopLossOptimizer) OptimizeStopLoss(returns []float64, metrics *PerformanceMetrics) *StopLossConfig {
	config := &StopLossConfig{
		UpdatedAt: time.Now(),
	}

	// 根据历史表现调整硬止损
	config.HardStopLoss = math.Min(0.1, metrics.MaxDrawdown*1.5) // 不超过10%，且基于历史最大回撤

	// 根据Sharpe比率调整ATR倍数
	if metrics.SharpeRatio > 2 {
		config.ATRMultiplier = 3
	} else if metrics.SharpeRatio > 1 {
		config.ATRMultiplier = 2
	} else {
		config.ATRMultiplier = 1.5
	}

	// 根据胜率调整移动止损
	if metrics.WinRate > 0.6 {
		config.TrailingStop = 0.02 // 2%回调
	} else {
		config.TrailingStop = 0.03 // 3%回调
	}

	// 根据盈亏比调整Parabolic因子
	if metrics.ProfitFactor > 2 {
		config.ParabolicAF = 0.02
	} else {
		config.ParabolicAF = 0.01
	}

	// 设置资金曲线回撤限制
	config.DrawdownLimit = metrics.MaxDrawdown * 1.2 // 允许比历史最大回撤多20%

	// 设置波动率止损倍数
	config.VolStopMultiple = 2.0

	// 设置时间限制
	config.TimeLimit = 7 * 24 * time.Hour // 默认7天

	return config
}

// CalculateParabolicSAR calculates Parabolic SAR value
func (o *StopLossOptimizer) CalculateParabolicSAR(high, low []float64, af, maxAF float64) float64 {
	if len(high) < 2 || len(low) < 2 {
		return 0
	}

	// 简化版Parabolic SAR计算
	trend := high[len(high)-1] > high[len(high)-2] // true表示上涨趋势
	ep := high[len(high)-1]                        // 极值点
	if !trend {
		ep = low[len(low)-1]
	}

	prevSAR := low[len(low)-2] // 使用前一个低点作为SAR起点
	if !trend {
		prevSAR = high[len(high)-2]
	}

	sar := prevSAR + af*(ep-prevSAR)

	// 确保SAR在合理范围内
	if trend {
		sar = math.Min(sar, low[len(low)-2])
		sar = math.Min(sar, low[len(low)-1])
	} else {
		sar = math.Max(sar, high[len(high)-2])
		sar = math.Max(sar, high[len(high)-1])
	}

	return sar
}
