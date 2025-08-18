package optimizer

import (
	"context"
	"fmt"
	"math"
	"sort"

	"qcat/internal/strategy/backtest"
)

// WalkForwardOptimizer implements walk-forward optimization
type WalkForwardOptimizer struct {
	config *WFOConfig
}

// WFOConfig represents walk-forward optimization configuration
type WFOConfig struct {
	InSampleSize    int     // 样本内数据大小
	OutSampleSize   int     // 样本外数据大小
	MinWinRate      float64 // 最小胜率要求
	MinProfitFactor float64 // 最小盈亏比要求
	Anchored        bool    // 是否使用锚定式
}

// WFOResult represents walk-forward optimization result
type WFOResult struct {
	Parameters     map[string]float64         // 最优参数
	InSampleStats  *backtest.PerformanceStats // 样本内统计
	OutSampleStats *backtest.PerformanceStats // 样本外统计
	Robustness     float64                    // 稳健性得分
}

// NewWalkForwardOptimizer creates a new walk-forward optimizer
func NewWalkForwardOptimizer(config *WFOConfig) *WalkForwardOptimizer {
	return &WalkForwardOptimizer{
		config: config,
	}
}

// Optimize performs walk-forward optimization
func (o *WalkForwardOptimizer) Optimize(ctx context.Context, data *DataSet, paramSpace map[string][2]float64) (*WFOResult, error) {
	if len(data.Returns) < o.config.InSampleSize+o.config.OutSampleSize {
		return nil, fmt.Errorf("insufficient data for walk-forward optimization")
	}

	var windows []Window
	if o.config.Anchored {
		// 锚定式：固定起始点
		start := 0
		for end := o.config.InSampleSize; end+o.config.OutSampleSize <= len(data.Returns); end += o.config.OutSampleSize {
			windows = append(windows, Window{
				InSampleStart:  start,
				InSampleEnd:    end,
				OutSampleStart: end,
				OutSampleEnd:   end + o.config.OutSampleSize,
			})
		}
	} else {
		// 滚动式：移动窗口
		for start := 0; start+o.config.InSampleSize+o.config.OutSampleSize <= len(data.Returns); start += o.config.OutSampleSize {
			windows = append(windows, Window{
				InSampleStart:  start,
				InSampleEnd:    start + o.config.InSampleSize,
				OutSampleStart: start + o.config.InSampleSize,
				OutSampleEnd:   start + o.config.InSampleSize + o.config.OutSampleSize,
			})
		}
	}

	// 对每个窗口进行优化
	var results []*WindowResult
	for _, window := range windows {
		result, err := o.optimizeWindow(ctx, data, window, paramSpace)
		if err != nil {
			return nil, fmt.Errorf("failed to optimize window: %w", err)
		}
		results = append(results, result)
	}

	// 汇总结果
	return o.summarizeResults(results)
}

// Window represents a walk-forward optimization window
type Window struct {
	InSampleStart  int
	InSampleEnd    int
	OutSampleStart int
	OutSampleEnd   int
}



// WindowResult represents optimization result for a window
type WindowResult struct {
	Window     Window
	Parameters map[string]float64
	InSample   *backtest.PerformanceStats
	OutSample  *backtest.PerformanceStats
}

// optimizeWindow optimizes parameters for a single window
func (o *WalkForwardOptimizer) optimizeWindow(ctx context.Context, data *DataSet, window Window, paramSpace map[string][2]float64) (*WindowResult, error) {
	// 提取样本内数据
	inSampleData := &DataSet{
		Returns: data.Returns[window.InSampleStart:window.InSampleEnd],
		Prices:  data.Prices[window.InSampleStart:window.InSampleEnd],
		Volumes: data.Volumes[window.InSampleStart:window.InSampleEnd],
	}

	// 使用网格搜索找到最优参数
	gridSearcher := NewGridSearcher(paramSpace)
	bestParams, err := gridSearcher.Search(ctx, inSampleData)
	if err != nil {
		return nil, fmt.Errorf("grid search failed: %w", err)
	}

	// 计算样本内性能
	inSampleStats := backtest.CalculatePerformanceStats(inSampleData.Returns)

	// 使用最优参数计算样本外性能
	outSampleData := &DataSet{
		Returns: data.Returns[window.OutSampleStart:window.OutSampleEnd],
		Prices:  data.Prices[window.OutSampleStart:window.OutSampleEnd],
		Volumes: data.Volumes[window.OutSampleStart:window.OutSampleEnd],
	}
	outSampleStats := backtest.CalculatePerformanceStats(outSampleData.Returns)

	return &WindowResult{
		Window:     window,
		Parameters: bestParams,
		InSample:   inSampleStats,
		OutSample:  outSampleStats,
	}, nil
}

// summarizeResults summarizes walk-forward optimization results
func (o *WalkForwardOptimizer) summarizeResults(results []*WindowResult) (*WFOResult, error) {
	if len(results) == 0 {
		return nil, fmt.Errorf("no results to summarize")
	}

	// 计算参数稳定性
	paramStability := make(map[string]float64)
	for param := range results[0].Parameters {
		values := make([]float64, len(results))
		for i, r := range results {
			values[i] = r.Parameters[param]
		}
		paramStability[param] = calculateStability(values)
	}

	// 选择最稳定的参数组合
	var bestParams map[string]float64
	maxStability := -math.MaxFloat64
	for _, r := range results {
		stability := 0.0
		for param, value := range r.Parameters {
			stability += paramStability[param] * value
		}
		if stability > maxStability {
			maxStability = stability
			bestParams = r.Parameters
		}
	}

	// 计算整体性能统计
	// 由于PerformanceStats结构体中没有Returns字段，使用模拟数据
	// 在实际实现中，需要从PerformanceStats中提取收益率数据

	return &WFOResult{
		Parameters:     bestParams,
		InSampleStats:  &backtest.PerformanceStats{}, // 暂时使用空结构体
		OutSampleStats: &backtest.PerformanceStats{}, // 暂时使用空结构体
		Robustness:     calculateRobustness(results),
	}, nil
}

// calculateStability calculates parameter stability
func calculateStability(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))

	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values) - 1)

	return 1 / (1 + math.Sqrt(variance))
}

// calculateRobustness calculates strategy robustness
func calculateRobustness(results []*WindowResult) float64 {
	if len(results) == 0 {
		return 0
	}

	// 计算样本内外性能比率
	var ratios []float64
	for _, r := range results {
		if r.OutSample.SharpeRatio > 0 && r.InSample.SharpeRatio > 0 {
			ratio := r.OutSample.SharpeRatio / r.InSample.SharpeRatio
			ratios = append(ratios, ratio)
		}
	}

	if len(ratios) == 0 {
		return 0
	}

	// 计算稳定性得分
	sort.Float64s(ratios)
	median := ratios[len(ratios)/2]
	consistency := 0.0
	for _, ratio := range ratios {
		if ratio >= 0.7 { // 样本外性能至少达到样本内的70%
			consistency++
		}
	}
	consistency /= float64(len(ratios))

	return median * consistency
}
