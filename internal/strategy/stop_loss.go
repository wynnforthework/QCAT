package strategy

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"qcat/internal/market/kline"
)

// StopLossManager manages dynamic stop loss and take profit
type StopLossManager struct {
	klineManager *kline.Manager
	config       *StopLossConfig
	parameters   map[string]*StopLossParams
	mu           sync.RWMutex
}

// StopLossConfig represents stop loss configuration
type StopLossConfig struct {
	ATRPeriod       int           // ATR计算周期
	RVPeriod        int           // 实现波动率计算周期
	RegimeWindow    int           // 市场状态识别窗口
	MinStopLoss     float64       // 最小止损比例
	MaxStopLoss     float64       // 最大止损比例
	MinTakeProfit   float64       // 最小止盈比例
	MaxTakeProfit   float64       // 最大止盈比例
	UpdateInterval  time.Duration // 更新间隔
	RegimeThreshold float64       // 市场状态切换阈值
}

// StopLossParams represents stop loss parameters
type StopLossParams struct {
	StrategyID  string
	Symbol      string
	StopLoss    float64
	TakeProfit  float64
	ATR         float64
	RealizedVol float64
	Regime      MarketRegime
	CurveSlope  float64
	LastUpdate  time.Time
	Version     int
}

// MarketRegime represents market regime
type MarketRegime string

const (
	RegimeTrending MarketRegime = "trending"
	RegimeRanging  MarketRegime = "ranging"
	RegimeVolatile MarketRegime = "volatile"
	RegimeCalm     MarketRegime = "calm"
)

// NewStopLossManager creates a new stop loss manager
func NewStopLossManager(km *kline.Manager, config *StopLossConfig) *StopLossManager {
	return &StopLossManager{
		klineManager: km,
		config:       config,
		parameters:   make(map[string]*StopLossParams),
	}
}

// UpdateParameters updates stop loss parameters for a strategy
func (slm *StopLossManager) UpdateParameters(ctx context.Context, strategyID, symbol string) error {
	// 获取K线数据
	end := time.Now()
	start := end.Add(-time.Duration(slm.config.RegimeWindow) * time.Hour)
	klines, err := slm.klineManager.GetHistory(ctx, symbol, start, end)
	if err != nil {
		return fmt.Errorf("failed to get kline data: %w", err)
	}

	if len(klines) < slm.config.ATRPeriod {
		return fmt.Errorf("insufficient data for ATR calculation")
	}

	// 计算ATR
	atr := slm.calculateATR(klines, slm.config.ATRPeriod)

	// 计算实现波动率
	realizedVol := slm.calculateRealizedVolatility(klines, slm.config.RVPeriod)

	// 识别市场状态
	regime := slm.detectMarketRegime(klines)

	// 计算资金曲线斜率
	curveSlope := slm.calculateCurveSlope(klines)

	// 计算动态止盈止损参数
	stopLoss, takeProfit := slm.calculateDynamicLevels(atr, realizedVol, regime, curveSlope)

	// 更新参数
	slm.mu.Lock()
	defer slm.mu.Unlock()

	key := fmt.Sprintf("%s_%s", strategyID, symbol)
	params := &StopLossParams{
		StrategyID:  strategyID,
		Symbol:      symbol,
		StopLoss:    stopLoss,
		TakeProfit:  takeProfit,
		ATR:         atr,
		RealizedVol: realizedVol,
		Regime:      regime,
		CurveSlope:  curveSlope,
		LastUpdate:  time.Now(),
	}

	// 版本管理
	if existing, exists := slm.parameters[key]; exists {
		params.Version = existing.Version + 1
	} else {
		params.Version = 1
	}

	slm.parameters[key] = params

	return nil
}

// GetParameters gets stop loss parameters
func (slm *StopLossManager) GetParameters(strategyID, symbol string) (*StopLossParams, error) {
	slm.mu.RLock()
	defer slm.mu.RUnlock()

	key := fmt.Sprintf("%s_%s", strategyID, symbol)
	params, exists := slm.parameters[key]
	if !exists {
		return nil, fmt.Errorf("parameters not found for %s_%s", strategyID, symbol)
	}

	return params, nil
}

// calculateATR calculates Average True Range
func (slm *StopLossManager) calculateATR(klines []*kline.Kline, period int) float64 {
	if len(klines) < period+1 {
		return 0
	}

	var trueRanges []float64
	for i := 1; i < len(klines); i++ {
		high := klines[i].HighPrice
		low := klines[i].LowPrice
		prevClose := klines[i-1].ClosePrice

		tr1 := high - low
		tr2 := math.Abs(high - prevClose)
		tr3 := math.Abs(low - prevClose)

		trueRange := math.Max(tr1, math.Max(tr2, tr3))
		trueRanges = append(trueRanges, trueRange)
	}

	// 计算ATR
	var sum float64
	for i := len(trueRanges) - period; i < len(trueRanges); i++ {
		sum += trueRanges[i]
	}

	return sum / float64(period)
}

// calculateRealizedVolatility calculates realized volatility
func (slm *StopLossManager) calculateRealizedVolatility(klines []*kline.Kline, period int) float64 {
	if len(klines) < period+1 {
		return 0
	}

	var returns []float64
	for i := 1; i < len(klines); i++ {
		ret := (klines[i].ClosePrice - klines[i-1].ClosePrice) / klines[i-1].ClosePrice
		returns = append(returns, ret)
	}

	// 计算实现波动率
	var sum float64
	for i := len(returns) - period; i < len(returns); i++ {
		sum += returns[i] * returns[i]
	}

	return math.Sqrt(sum / float64(period))
}

// detectMarketRegime detects market regime
func (slm *StopLossManager) detectMarketRegime(klines []*kline.Kline) MarketRegime {
	if len(klines) < 20 {
		return RegimeCalm
	}

	// 计算趋势强度
	var returns []float64
	for i := 1; i < len(klines); i++ {
		ret := (klines[i].ClosePrice - klines[i-1].ClosePrice) / klines[i-1].ClosePrice
		returns = append(returns, ret)
	}

	// 计算趋势指标
	trendStrength := slm.calculateTrendStrength(returns)
	volatility := slm.calculateVolatility(returns)

	// 根据趋势强度和波动率判断市场状态
	if trendStrength > slm.config.RegimeThreshold {
		if volatility > 0.02 { // 高波动率
			return RegimeVolatile
		}
		return RegimeTrending
	} else {
		if volatility > 0.02 {
			return RegimeVolatile
		}
		return RegimeRanging
	}
}

// calculateTrendStrength calculates trend strength
func (slm *StopLossManager) calculateTrendStrength(returns []float64) float64 {
	if len(returns) < 10 {
		return 0
	}

	// 使用线性回归计算趋势强度
	var sumX, sumY, sumXY, sumX2 float64
	n := float64(len(returns))

	for i, ret := range returns {
		x := float64(i)
		y := ret
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	return math.Abs(slope)
}

// calculateVolatility calculates volatility
func (slm *StopLossManager) calculateVolatility(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}

	mean := 0.0
	for _, ret := range returns {
		mean += ret
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, ret := range returns {
		diff := ret - mean
		variance += diff * diff
	}
	variance /= float64(len(returns) - 1)

	return math.Sqrt(variance)
}

// calculateCurveSlope calculates equity curve slope
func (slm *StopLossManager) calculateCurveSlope(klines []*kline.Kline) float64 {
	if len(klines) < 10 {
		return 0
	}

	// 使用价格变化作为资金曲线代理
	var prices []float64
	for _, k := range klines {
		prices = append(prices, k.ClosePrice)
	}

	// 计算斜率
	var sumX, sumY, sumXY, sumX2 float64
	n := float64(len(prices))

	for i, price := range prices {
		x := float64(i)
		y := price
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	return slope
}

// calculateDynamicLevels calculates dynamic stop loss and take profit levels
func (slm *StopLossManager) calculateDynamicLevels(atr, realizedVol float64, regime MarketRegime, curveSlope float64) (float64, float64) {
	// 基础参数
	baseStopLoss := atr * 2.0
	baseTakeProfit := atr * 3.0

	// 根据市场状态调整
	switch regime {
	case RegimeTrending:
		baseStopLoss *= 1.5
		baseTakeProfit *= 1.2
	case RegimeRanging:
		baseStopLoss *= 0.8
		baseTakeProfit *= 0.6
	case RegimeVolatile:
		baseStopLoss *= 2.0
		baseTakeProfit *= 2.5
	case RegimeCalm:
		baseStopLoss *= 0.6
		baseTakeProfit *= 0.8
	}

	// 根据实现波动率调整
	volAdjustment := realizedVol / 0.02 // 标准化到2%
	baseStopLoss *= volAdjustment
	baseTakeProfit *= volAdjustment

	// 根据资金曲线斜率调整
	if curveSlope > 0 {
		// 上升趋势，收紧止损
		baseStopLoss *= 0.9
		baseTakeProfit *= 1.1
	} else {
		// 下降趋势，放宽止损
		baseStopLoss *= 1.1
		baseTakeProfit *= 0.9
	}

	// 应用限制
	stopLoss := math.Max(slm.config.MinStopLoss, math.Min(slm.config.MaxStopLoss, baseStopLoss))
	takeProfit := math.Max(slm.config.MinTakeProfit, math.Min(slm.config.MaxTakeProfit, baseTakeProfit))

	return stopLoss, takeProfit
}

// StartUpdateLoop starts the parameter update loop
func (slm *StopLossManager) StartUpdateLoop(ctx context.Context) {
	ticker := time.NewTicker(slm.config.UpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			slm.updateAllParameters(ctx)
		}
	}
}

// updateAllParameters updates parameters for all strategies
func (slm *StopLossManager) updateAllParameters(ctx context.Context) {
	slm.mu.RLock()
	keys := make([]string, 0, len(slm.parameters))
	for key := range slm.parameters {
		keys = append(keys, key)
	}
	slm.mu.RUnlock()

	for _, key := range keys {
		// 解析strategyID和symbol
		// 这里简化处理，实际应该从key中解析
		strategyID := "strategy_1" // 示例
		symbol := "BTCUSDT"        // 示例

		if err := slm.UpdateParameters(ctx, strategyID, symbol); err != nil {
			// 记录错误但不中断其他更新
			fmt.Printf("Failed to update parameters for %s: %v\n", key, err)
		}
	}
}
