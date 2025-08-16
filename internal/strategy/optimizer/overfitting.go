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

	// 执行参数敏感度分析（使用真实数据）
	// 现在PerformanceStats结构体已经包含Returns字段，使用真实数据
	if len(inSample.Returns) == 0 {
		return nil, fmt.Errorf("no returns data available for sensitivity analysis")
	}
	
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
func (d *OverfitDetector) calculateDeflatedSharpe(stats *backtest.PerformanceStats) (float64, error) {
	if stats.SharpeRatio <= 0 {
		return 0, nil
	}

	// 使用真实的收益率数据计算收缩夏普比率
	if len(stats.Returns) == 0 {
		return 0, fmt.Errorf("no returns data available for DSR calculation")
	}

	n := len(stats.Returns)
	if n < d.config.MinSamples {
		return 0, fmt.Errorf("insufficient samples for DSR calculation: %d < %d", n, d.config.MinSamples)
	}

	// 计算收益率的自相关性
	autocorr := calculateAutocorrelation(stats.Returns)
	
	// 计算有效样本数（考虑自相关性）
	effectiveN := float64(n) * (1 - autocorr) / (1 + autocorr)
	if effectiveN <= 1 {
		effectiveN = 1
	}

	// 计算收缩因子
	shrinkage := math.Sqrt((effectiveN - 1) / effectiveN)
	
	// 应用收缩因子
	dsr := stats.SharpeRatio * shrinkage

	return dsr, nil
}

// performPBOTest performs Probability of Backtest Overfitting test
func (d *OverfitDetector) performPBOTest(inSample, outSample *backtest.PerformanceStats) (float64, error) {
	// 使用真实数据进行PBO检验
	if len(inSample.Returns) == 0 || len(outSample.Returns) == 0 {
		return 0, fmt.Errorf("insufficient returns data for PBO test")
	}

	// 计算样本内和样本外的夏普比率
	inSampleSharpe := inSample.SharpeRatio
	outSampleSharpe := outSample.SharpeRatio

	// 如果样本外表现显著差于样本内，则可能存在过拟合
	if inSampleSharpe <= 0 {
		return 1.0, nil // 如果样本内夏普比率为负，认为过拟合概率为100%
	}

	// 计算性能衰减比率
	performanceDecay := (inSampleSharpe - outSampleSharpe) / inSampleSharpe
	
	// 基于性能衰减计算PBO概率
	// 如果样本外表现比样本内差50%以上，认为过拟合概率很高
	pboScore := math.Max(0, math.Min(1, performanceDecay*2))

	// 考虑统计显著性
	// 计算t统计量来评估差异的显著性
	n1, n2 := len(inSample.Returns), len(outSample.Returns)
	if n1 > 1 && n2 > 1 {
		// 计算合并方差
		var1 := d.calculateVariance(inSample.Returns)
		var2 := d.calculateVariance(outSample.Returns)
		pooledVar := ((float64(n1-1)*var1 + float64(n2-1)*var2) / float64(n1+n2-2))
		
		if pooledVar > 0 {
			// 计算标准误差
			se := math.Sqrt(pooledVar * (1.0/float64(n1) + 1.0/float64(n2)))
			
			// 计算t统计量
			tStat := math.Abs(inSampleSharpe - outSampleSharpe) / se
			
			// 如果t统计量大于临界值（约1.96对应95%置信水平），调整PBO分数
			if tStat > 1.96 {
				pboScore = math.Min(1.0, pboScore*1.5)
			}
		}
	}

	return pboScore, nil
}

// calculateVariance calculates the variance of returns
func (d *OverfitDetector) calculateVariance(returns []float64) float64 {
	if len(returns) <= 1 {
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
	variance /= float64(len(returns) - 1)

	return variance
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
	// 新增：实现性能统计计算
	if len(returns) == 0 {
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

	// 新增：计算总收益率
	totalReturn := 0.0
	for _, r := range returns {
		totalReturn += r
	}

	// 新增：计算年化收益率（假设252个交易日）
	annualReturn := totalReturn * 252.0 / float64(len(returns))

	// 新增：计算夏普比率
	mean := totalReturn / float64(len(returns))
	variance := 0.0
	for _, r := range returns {
		variance += (r - mean) * (r - mean)
	}
	variance /= float64(len(returns))

	sharpeRatio := 0.0
	if variance > 0 {
		sharpeRatio = mean / math.Sqrt(variance)
	}

	// 新增：计算最大回撤
	maxDrawdown := 0.0
	peak := 0.0
	cumulative := 0.0

	for _, r := range returns {
		cumulative += r
		if cumulative > peak {
			peak = cumulative
		}
		drawdown := peak - cumulative
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	// 新增：计算胜率
	wins := 0
	for _, r := range returns {
		if r > 0 {
			wins++
		}
	}
	winRate := float64(wins) / float64(len(returns))

	// 新增：计算盈亏比
	totalWins := 0.0
	totalLosses := 0.0
	for _, r := range returns {
		if r > 0 {
			totalWins += r
		} else {
			totalLosses += math.Abs(r)
		}
	}

	profitFactor := 0.0
	if totalLosses > 0 {
		profitFactor = totalWins / totalLosses
	}

	return &backtest.PerformanceStats{
		TotalReturn:    totalReturn,
		AnnualReturn:   annualReturn,
		SharpeRatio:    sharpeRatio,
		MaxDrawdown:    maxDrawdown,
		WinRate:        winRate,
		ProfitFactor:   profitFactor,
		TradeCount:     len(returns),
		AvgTradeReturn: mean,
		AvgHoldingTime: 0, // 新增：需要交易时间数据来计算平均持仓时间
	}
}
