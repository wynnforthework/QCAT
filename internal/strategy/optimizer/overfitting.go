package optimizer

import (
	"context"
	"fmt"
	"math"
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
func (d *OverfitDetector) CheckOverfitting(ctx context.Context, inSample, outSample *PerformanceStats) (*OverfitResult, error) {
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

	// 执行参数敏感度分析
	sensitivity, err := d.analyzeSensitivity(inSample.Returns)
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
func (d *OverfitDetector) calculateDeflatedSharpe(stats *PerformanceStats) (float64, error) {
	if stats.SharpeRatio <= 0 {
		return 0, nil
	}

	// 计算有效自由度
	n := len(stats.Returns)
	if n < d.config.MinSamples {
		return 0, fmt.Errorf("insufficient samples for DSR calculation")
	}

	// 计算自相关系数
	ac := calculateAutocorrelation(stats.Returns)

	// 计算有效样本量
	nEff := float64(n) / (1 + 2*ac)

	// 计算收缩因子
	shrinkage := math.Sqrt((nEff - 1) / nEff)

	// 计算收缩夏普比率
	dsr := stats.SharpeRatio * shrinkage

	return dsr, nil
}

// performPBOTest performs Probability of Backtest Overfitting test
func (d *OverfitDetector) performPBOTest(inSample, outSample *PerformanceStats) (float64, error) {
	if len(inSample.Returns) != len(outSample.Returns) {
		return 0, fmt.Errorf("sample size mismatch")
	}

	// 计算样本内外性能比率
	inSharpe := inSample.SharpeRatio
	outSharpe := outSample.SharpeRatio

	if inSharpe <= 0 {
		return 1, nil // 样本内表现不佳，认为是过拟合
	}

	// 计算PBO得分
	pbo := 1 - (outSharpe / inSharpe)
	if pbo < 0 {
		pbo = 0
	}

	return pbo, nil
}

// analyzeSensitivity performs parameter sensitivity analysis
func (d *OverfitDetector) analyzeSensitivity(returns []float64) (map[string]float64, error) {
	if len(returns) < d.config.MinSamples {
		return nil, fmt.Errorf("insufficient samples for sensitivity analysis")
	}

	// 模拟参数扰动
	sensitivity := make(map[string]float64)
	baseStats := calculatePerformanceStats(returns)

	// 对每个参数进行敏感度分析
	params := []string{"stopLoss", "takeProfit", "entryThreshold", "exitThreshold"}
	for _, param := range params {
		// 计算参数扰动对性能的影响
		variations := []float64{0.8, 0.9, 1.1, 1.2} // 参数变化范围
		impacts := make([]float64, len(variations))

		for i, v := range variations {
			// 模拟参数变化后的性能
			modifiedReturns := simulateParamChange(returns, param, v)
			modifiedStats := calculatePerformanceStats(modifiedReturns)

			// 计算性能变化
			impacts[i] = math.Abs(modifiedStats.SharpeRatio-baseStats.SharpeRatio) / baseStats.SharpeRatio
		}

		// 计算平均敏感度
		sensitivity[param] = calculateMean(impacts)
	}

	return sensitivity, nil
}

// evaluateOverfitting evaluates if the strategy is overfitting
func (d *OverfitDetector) evaluateOverfitting(result *OverfitResult) bool {
	// 检查Deflated Sharpe Ratio
	if result.DeflatedSharpe < 0.5 { // DSR显著低于原始夏普比率
		return true
	}

	// 检查PBO得分
	if result.PBOScore > d.config.PBOThreshold {
		return true
	}

	// 检查参数敏感度
	for _, sensitivity := range result.ParamSensitivity {
		if sensitivity > 0.3 { // 参数敏感度过高
			return true
		}
	}

	return false
}

// Helper functions

func calculateAutocorrelation(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}

	mean := calculateMean(returns)
	var numerator, denominator float64

	for i := 1; i < len(returns); i++ {
		numerator += (returns[i] - mean) * (returns[i-1] - mean)
		denominator += (returns[i] - mean) * (returns[i] - mean)
	}

	if denominator == 0 {
		return 0
	}
	return numerator / denominator
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

func simulateParamChange(returns []float64, param string, variation float64) []float64 {
	// 简化版参数影响模拟
	modified := make([]float64, len(returns))
	copy(modified, returns)

	switch param {
	case "stopLoss":
		// 模拟止损变化的影响
		for i := range modified {
			if modified[i] < 0 {
				modified[i] *= variation
			}
		}
	case "takeProfit":
		// 模拟止盈变化的影响
		for i := range modified {
			if modified[i] > 0 {
				modified[i] *= variation
			}
		}
	case "entryThreshold":
		// 模拟入场阈值变化的影响
		for i := range modified {
			modified[i] *= variation
		}
	case "exitThreshold":
		// 模拟出场阈值变化的影响
		for i := range modified {
			if i > 0 && modified[i]*modified[i-1] < 0 {
				modified[i] *= variation
			}
		}
	}

	return modified
}
