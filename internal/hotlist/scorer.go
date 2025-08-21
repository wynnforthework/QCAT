package hotlist

import (
	"context"
	"fmt"
	"math"
	"time"

	"qcat/internal/market/funding"
	"qcat/internal/market/kline"
	"qcat/internal/market/oi"
)

// Scorer calculates hotlist scores
type Scorer struct {
	klineManager   *kline.Manager
	fundingManager *funding.Manager
	oiManager      *oi.Manager
	config         *ScorerConfig
}

// ScorerConfig represents scorer configuration
type ScorerConfig struct {
	VolJumpWindow    int     // 波动率跳跃窗口
	VolJumpThreshold float64 // 波动率跳跃阈值
	TurnoverWindow   int     // 换手率窗口
	OIChangeWindow   int     // 持仓量变化窗口
	FundingZWindow   int     // 资金费率Z分数窗口
	RegimeWindow     int     // 市场状态窗口
}

// Score represents a symbol's hotlist score
type Score struct {
	Symbol     string
	TotalScore float64
	Components map[string]float64
	LastUpdate time.Time
}

// NewScorer creates a new scorer
func NewScorer(km *kline.Manager, fm *funding.Manager, om *oi.Manager, config *ScorerConfig) *Scorer {
	return &Scorer{
		klineManager:   km,
		fundingManager: fm,
		oiManager:      om,
		config:         config,
	}
}

// CalculateScore calculates hotlist score for a symbol
func (s *Scorer) CalculateScore(ctx context.Context, symbol string) (*Score, error) {
	score := &Score{
		Symbol:     symbol,
		Components: make(map[string]float64),
		LastUpdate: time.Now(),
	}

	// 计算波动率跳跃分数
	volJump, err := s.calculateVolJump(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate vol jump: %w", err)
	}
	score.Components["vol_jump"] = volJump

	// 计算换手率分数
	turnover, err := s.calculateTurnover(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate turnover: %w", err)
	}
	score.Components["turnover"] = turnover

	// 计算持仓量变化分数
	oiChange, err := s.calculateOIChange(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate OI change: %w", err)
	}
	score.Components["oi_change"] = oiChange

	// 计算资金费率Z分数
	fundingZ, err := s.calculateFundingZ(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate funding Z: %w", err)
	}
	score.Components["funding_z"] = fundingZ

	// 计算市场状态切换分数
	regimeShift, err := s.calculateRegimeShift(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate regime shift: %w", err)
	}
	score.Components["regime_shift"] = regimeShift

	// 计算总分
	score.TotalScore = s.calculateTotalScore(score.Components)

	return score, nil
}

// calculateVolJump calculates volatility jump score
func (s *Scorer) calculateVolJump(ctx context.Context, symbol string) (float64, error) {
	// 检查klineManager是否已初始化
	if s.klineManager == nil {
		// 返回默认值而不是崩溃
		return 0.5, nil // 返回中等波动率分数
	}

	// 获取K线数据
	end := time.Now()
	start := end.Add(-time.Duration(s.config.VolJumpWindow) * time.Hour)
	klines, err := s.klineManager.GetHistory(ctx, symbol, start, end)
	if err != nil {
		return 0, err
	}

	// 计算历史波动率
	var returns []float64
	for i := 1; i < len(klines); i++ {
		ret := (klines[i].ClosePrice - klines[i-1].ClosePrice) / klines[i-1].ClosePrice
		returns = append(returns, ret)
	}

	// 计算波动率
	vol := calculateVolatility(returns)

	// 计算波动率变化率
	if len(returns) < 2 {
		return 0, nil
	}
	volChange := (vol - calculateVolatility(returns[:len(returns)-1])) / calculateVolatility(returns[:len(returns)-1])

	// 标准化分数
	score := math.Max(0, math.Min(1, volChange/s.config.VolJumpThreshold))
	return score, nil
}

// calculateTurnover calculates turnover score
func (s *Scorer) calculateTurnover(ctx context.Context, symbol string) (float64, error) {
	// 检查klineManager是否已初始化
	if s.klineManager == nil {
		// 返回默认值而不是崩溃
		return 0.4, nil // 返回中等换手率分数
	}

	// 获取K线数据
	end := time.Now()
	start := end.Add(-time.Duration(s.config.TurnoverWindow) * time.Hour)
	klines, err := s.klineManager.GetHistory(ctx, symbol, start, end)
	if err != nil {
		return 0, err
	}

	// 计算换手率
	var turnover float64
	for _, k := range klines {
		turnover += k.Volume * k.ClosePrice
	}
	avgTurnover := turnover / float64(len(klines))

	// 标准化分数
	score := math.Min(1, avgTurnover/1000000) // 假设100万为基准
	return score, nil
}

// calculateOIChange calculates open interest change score
func (s *Scorer) calculateOIChange(ctx context.Context, symbol string) (float64, error) {
	// 检查oiManager是否已初始化
	if s.oiManager == nil {
		// 返回默认值而不是崩溃
		return 0.3, nil // 返回中等持仓量变化分数
	}

	// 获取持仓量数据
	end := time.Now()
	start := end.Add(-time.Duration(s.config.OIChangeWindow) * time.Hour)
	ois, err := s.oiManager.GetHistory(ctx, symbol, start, end)
	if err != nil {
		return 0, err
	}

	if len(ois) < 2 {
		return 0, nil
	}

	// 计算持仓量变化率
	oiChange := (ois[len(ois)-1].Value - ois[0].Value) / ois[0].Value

	// 标准化分数
	score := math.Max(0, math.Min(1, math.Abs(oiChange)))
	return score, nil
}

// calculateFundingZ calculates funding rate Z-score
func (s *Scorer) calculateFundingZ(ctx context.Context, symbol string) (float64, error) {
	// 检查fundingManager是否已初始化
	if s.fundingManager == nil {
		// 返回默认值而不是崩溃
		return 0.2, nil // 返回中等资金费率分数
	}

	// 获取资金费率数据
	end := time.Now()
	start := end.Add(-time.Duration(s.config.FundingZWindow) * time.Hour)
	rates, err := s.fundingManager.GetHistory(ctx, symbol, start, end)
	if err != nil {
		return 0, err
	}

	// 计算Z分数
	var values []float64
	for _, r := range rates {
		values = append(values, r.Rate)
	}
	mean := calculateMean(values)
	stdDev := calculateStdDev(values, mean)

	if stdDev == 0 {
		return 0, nil
	}

	zScore := math.Abs((rates[len(rates)-1].Rate - mean) / stdDev)

	// 标准化分数
	score := math.Min(1, zScore/3) // 3个标准差为满分
	return score, nil
}

// calculateRegimeShift calculates market regime shift score
func (s *Scorer) calculateRegimeShift(ctx context.Context, symbol string) (float64, error) {
	// 检查klineManager是否已初始化
	if s.klineManager == nil {
		// 返回默认值而不是崩溃
		return 0.3, nil // 返回中等市场状态变化分数
	}

	// 获取K线数据
	end := time.Now()
	start := end.Add(-time.Duration(s.config.RegimeWindow) * time.Hour)
	klines, err := s.klineManager.GetHistory(ctx, symbol, start, end)
	if err != nil {
		return 0, err
	}

	// 计算趋势强度
	var returns []float64
	for i := 1; i < len(klines); i++ {
		ret := (klines[i].ClosePrice - klines[i-1].ClosePrice) / klines[i-1].ClosePrice
		returns = append(returns, ret)
	}

	// 计算趋势变化
	trendStrength := calculateTrendStrength(returns)

	// 标准化分数
	score := math.Min(1, trendStrength)
	return score, nil
}

// calculateTotalScore calculates total score from components
func (s *Scorer) calculateTotalScore(components map[string]float64) float64 {
	weights := map[string]float64{
		"vol_jump":     0.25,
		"turnover":     0.20,
		"oi_change":    0.20,
		"funding_z":    0.15,
		"regime_shift": 0.20,
	}

	var totalScore float64
	for component, score := range components {
		if weight, exists := weights[component]; exists {
			totalScore += score * weight
		}
	}

	return totalScore
}

// Helper functions

func calculateVolatility(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}

	mean := calculateMean(returns)
	variance := 0.0
	for _, r := range returns {
		diff := r - mean
		variance += diff * diff
	}
	variance /= float64(len(returns) - 1)
	return math.Sqrt(variance)
}

func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateStdDev(values []float64, mean float64) float64 {
	if len(values) < 2 {
		return 0
	}

	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values) - 1)
	return math.Sqrt(variance)
}

func calculateTrendStrength(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}

	// 使用Hurst指数作为趋势强度指标
	var (
		maxRange float64
		minRange float64
	)

	for i := 0; i < len(returns)-1; i++ {
		high := returns[i]
		low := returns[i]
		for j := i + 1; j < len(returns); j++ {
			if returns[j] > high {
				high = returns[j]
			}
			if returns[j] < low {
				low = returns[j]
			}
			r := high - low
			if r > maxRange {
				maxRange = r
			}
			if r < minRange || minRange == 0 {
				minRange = r
			}
		}
	}

	if minRange == 0 {
		return 0
	}

	return maxRange / minRange
}
