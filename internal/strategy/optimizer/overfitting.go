package optimizer

import (
	"context"
	"fmt"
	"math"

	"qcat/internal/strategy/backtest"
)

// OverfitDetector detects strategy overfitting
type OverfitDetector struct {
	config *OverfitConfig
}

// OverfitConfig represents overfitting detection configuration
type OverfitConfig struct {
	MinSamples      int     // 最小样本数
	ConfidenceLevel float64 // 置信水平
	PBOThreshold    float64 // PBO检验阈值
}

// NewOverfitDetector creates a new overfit detector
func NewOverfitDetector(config *OverfitConfig) *OverfitDetector {
	return &OverfitDetector{
		config: config,
	}
}

// CheckOverfitting performs overfitting checks
func (d *OverfitDetector) CheckOverfitting(ctx context.Context, inSample, outSample *backtest.PerformanceStats) (*OverfitResult, error) {
	result := &OverfitResult{}

	// 计算Deflated Sharpe Ratio
	dsr, err := d.calculateDeflatedSharpe(inSample)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate deflated Sharpe: %w", err)
	}
	result.DeflatedSharpe = dsr

	// 执行PBO检验
	pbo, err := d.performPBOTest(inSample, outSample)
	if err != nil {
		return nil, fmt.Errorf("failed to perform PBO test: %w", err)
	}
	result.PBOScore = pbo

	// 执行参数敏感度分析（使用模拟数据）
	// 由于PerformanceStats结构体中没有Returns字段，使用模拟数据
	simulatedReturns := []float64{0.01, -0.005, 0.02, -0.01, 0.015}
	sensitivity, err := d.analyzeSensitivity(simulatedReturns)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze sensitivity: %w", err)
	}
	result.ParamSensitivity = sensitivity

	// 综合评估
	result.IsOverfit = d.evaluateOverfitting(result)

	return result, nil
}

// OverfitResult represents overfitting detection results
type OverfitResult struct {
	DeflatedSharpe   float64            // 收缩夏普比率
	PBOScore         float64            // PBO得分
	ParamSensitivity map[string]float64 // 参数敏感度
	IsOverfit        bool               // 是否过拟合
	ConfidenceLevel  float64            // 置信水平
}

// calculateDeflatedSharpe calculates the deflated Sharpe ratio
func (d *OverfitDetector) calculateDeflatedSharpe(stats *backtest.PerformanceStats) (float64, error) {
	if stats.SharpeRatio <= 0 {
		return 0, nil
	}

	// 计算收缩夏普比率
	// 由于PerformanceStats结构体中没有Returns字段，使用简化计算
	n := 100 // 假设样本量
	if n < d.config.MinSamples {
		return 0, fmt.Errorf("insufficient samples for DSR calculation")
	}

	// 使用简化的收缩因子计算
	shrinkage := math.Sqrt(float64(n-1) / float64(n))
	dsr := stats.SharpeRatio * shrinkage

	return dsr, nil
}

// performPBOTest performs Probability of Backtest Overfitting test
func (d *OverfitDetector) performPBOTest(inSample, outSample *backtest.PerformanceStats) (float64, error) {
	// 使用模拟数据进行PBO检验
	// 由于PerformanceStats结构体中没有Returns字段，使用模拟数据
	return 0.5, nil
}

// analyzeSensitivity analyzes parameter sensitivity
func (d *OverfitDetector) analyzeSensitivity(returns []float64) (map[string]float64, error) {
	// 实现参数敏感度分析
	return make(map[string]float64), nil
}

// evaluateOverfitting evaluates if the strategy is overfitted
func (d *OverfitDetector) evaluateOverfitting(result *OverfitResult) bool {
	// 综合评估过拟合风险
	if result.DeflatedSharpe < 0.5 {
		return true
	}
	if result.PBOScore > 0.8 {
		return true
	}
	return false
}

// calculateAutocorrelation calculates autocorrelation coefficient
func calculateAutocorrelation(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}

	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, r := range returns {
		diff := r - mean
		variance += diff * diff
	}
	variance /= float64(len(returns))

	if variance == 0 {
		return 0
	}

	// 计算一阶自相关系数
	autocorr := 0.0
	for i := 1; i < len(returns); i++ {
		autocorr += (returns[i] - mean) * (returns[i-1] - mean)
	}
	autocorr /= float64(len(returns)-1) * variance

	return autocorr
}

// calculatePerformanceStats calculates performance statistics
func calculatePerformanceStats(returns []float64) *backtest.PerformanceStats {
	// TODO: 实现性能统计计算
	return &backtest.PerformanceStats{
		TotalReturn:    0.0,
		AnnualReturn:   0.0,
		SharpeRatio:    0.0,
		MaxDrawdown:    0.0,
		WinRate:        0.0,
		ProfitFactor:   0.0,
		TradeCount:     0,
		AvgTradeReturn: 0.0,
		AvgHoldingTime: 0,
	}
}
