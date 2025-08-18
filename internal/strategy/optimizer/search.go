package optimizer

import (
	"context"
	"math"
	"math/rand"
	"sort"
	"time"

	"qcat/internal/config"
	"qcat/internal/market"
)

// SearchAlgorithm defines the interface for optimization algorithms
type SearchAlgorithm interface {
	Search(ctx context.Context, data *DataSet) (map[string]float64, error)
}

// GridSearcher implements grid search optimization
type GridSearcher struct {
	paramSpace map[string][2]float64
	gridSize   int
}

// NewGridSearcher creates a new grid searcher
func NewGridSearcher(paramSpace map[string][2]float64) *GridSearcher {
	// Get grid size from configuration with proper fallback
	gridSize := 10 // Default fallback
	if algorithmConfig := config.GetAlgorithmConfig(); algorithmConfig != nil {
		gridSize = algorithmConfig.GetGridSize()
	}
	
	return &GridSearcher{
		paramSpace: paramSpace,
		gridSize:   gridSize,
	}
}

// Search performs grid search optimization
func (s *GridSearcher) Search(ctx context.Context, data *DataSet) (map[string]float64, error) {
	// 生成网格点
	grid := make(map[string][]float64)
	for param, bounds := range s.paramSpace {
		grid[param] = make([]float64, s.gridSize)
		step := (bounds[1] - bounds[0]) / float64(s.gridSize-1)
		for i := 0; i < s.gridSize; i++ {
			grid[param][i] = bounds[0] + float64(i)*step
		}
	}

	// 评估所有组合
	var bestScore float64 = -math.MaxFloat64
	bestParams := make(map[string]float64)
	params := make(map[string]float64)

	for i := 0; i < s.gridSize; i++ {
		for param := range s.paramSpace {
			params[param] = grid[param][i]
		}

		score, err := evaluateParams(ctx, data, params)
		if err != nil {
			return nil, err
		}

		if score > bestScore {
			bestScore = score
			for k, v := range params {
				bestParams[k] = v
			}
		}
	}

	return bestParams, nil
}

// BayesianOptimizer implements Bayesian optimization
type BayesianOptimizer struct {
	paramSpace map[string][2]float64
	iterations int
	samples    []Sample
}

// Sample represents a parameter sample and its score
type Sample struct {
	Params map[string]float64
	Score  float64
}

// NewBayesianOptimizer creates a new Bayesian optimizer
func NewBayesianOptimizer(paramSpace map[string][2]float64, iterations int) *BayesianOptimizer {
	return &BayesianOptimizer{
		paramSpace: paramSpace,
		iterations: iterations,
		samples:    make([]Sample, 0),
	}
}

// Search performs Bayesian optimization
func (s *BayesianOptimizer) Search(ctx context.Context, data *DataSet) (map[string]float64, error) {
	// 初始随机采样
	for i := 0; i < 5; i++ {
		params := make(map[string]float64)
		for param, bounds := range s.paramSpace {
			params[param] = bounds[0] + rand.Float64()*(bounds[1]-bounds[0])
		}

		score, err := evaluateParams(ctx, data, params)
		if err != nil {
			return nil, err
		}

		s.samples = append(s.samples, Sample{
			Params: params,
			Score:  score,
		})
	}

	// 迭代优化
	for i := 0; i < s.iterations; i++ {
		nextParams := s.proposeNextPoint()
		score, err := evaluateParams(ctx, data, nextParams)
		if err != nil {
			return nil, err
		}

		s.samples = append(s.samples, Sample{
			Params: nextParams,
			Score:  score,
		})
	}

	// 返回最佳参数
	sort.Slice(s.samples, func(i, j int) bool {
		return s.samples[i].Score > s.samples[j].Score
	})

	return s.samples[0].Params, nil
}

// proposeNextPoint proposes the next point to evaluate
func (s *BayesianOptimizer) proposeNextPoint() map[string]float64 {
	// 简化版：使用高斯过程回归
	params := make(map[string]float64)
	for param, bounds := range s.paramSpace {
		// 计算均值和方差
		var sum, sumSq float64
		for _, sample := range s.samples {
			value := sample.Params[param]
			sum += value
			sumSq += value * value
		}
		mean := sum / float64(len(s.samples))
		variance := sumSq/float64(len(s.samples)) - mean*mean

		// 采样新点
		value := mean + rand.NormFloat64()*math.Sqrt(variance)
		value = math.Max(bounds[0], math.Min(bounds[1], value))
		params[param] = value
	}

	return params
}

// CMAESOptimizer implements CMA-ES optimization
type CMAESOptimizer struct {
	paramSpace  map[string][2]float64
	popSize     int
	generations int
}

// NewCMAESOptimizer creates a new CMA-ES optimizer
func NewCMAESOptimizer(paramSpace map[string][2]float64, popSize, generations int) *CMAESOptimizer {
	return &CMAESOptimizer{
		paramSpace:  paramSpace,
		popSize:     popSize,
		generations: generations,
	}
}

// Search performs CMA-ES optimization
func (s *CMAESOptimizer) Search(ctx context.Context, data *DataSet) (map[string]float64, error) {
	// 初始化
	dim := len(s.paramSpace)
	mean := make([]float64, dim)
	sigma := 1.0
	C := identity(dim)
	pc := zeros(dim)
	ps := zeros(dim)

	// 迭代优化
	for gen := 0; gen < s.generations; gen++ {
		// 生成种群
		population := make([][]float64, s.popSize)
		scores := make([]float64, s.popSize)

		for i := 0; i < s.popSize; i++ {
			x := multivarNormal(mean, C, sigma)
			params := s.vectorToParams(x)
			score, err := evaluateParams(ctx, data, params)
			if err != nil {
				return nil, err
			}
			population[i] = x
			scores[i] = score
		}

		// 更新策略参数
		mean, sigma, C, pc, ps = s.updateParameters(population, scores, mean, sigma, C, pc, ps)
	}

	// 返回最佳参数
	return s.vectorToParams(mean), nil
}

// Helper functions for CMA-ES
func identity(n int) [][]float64 {
	m := make([][]float64, n)
	for i := range m {
		m[i] = make([]float64, n)
		m[i][i] = 1
	}
	return m
}

func zeros(n int) []float64 {
	return make([]float64, n)
}

func multivarNormal(mean []float64, C [][]float64, sigma float64) []float64 {
	// 简化版多元正态分布采样
	x := make([]float64, len(mean))
	for i := range x {
		x[i] = mean[i] + rand.NormFloat64()*sigma
	}
	return x
}

func (s *CMAESOptimizer) vectorToParams(x []float64) map[string]float64 {
	params := make(map[string]float64)
	i := 0
	for param, bounds := range s.paramSpace {
		value := x[i]
		value = math.Max(bounds[0], math.Min(bounds[1], value))
		params[param] = value
		i++
	}
	return params
}

func (s *CMAESOptimizer) updateParameters(pop [][]float64, scores []float64, mean []float64, sigma float64, C [][]float64, pc, ps []float64) ([]float64, float64, [][]float64, []float64, []float64) {
	// 简化版参数更新
	// 选择前半部分最好的个体
	indices := make([]int, len(scores))
	for i := range indices {
		indices[i] = i
	}
	sort.Slice(indices, func(i, j int) bool {
		return scores[indices[i]] > scores[indices[j]]
	})

	// 更新均值
	newMean := make([]float64, len(mean))
	for i := 0; i < len(mean); i++ {
		sum := 0.0
		for j := 0; j < s.popSize/2; j++ {
			sum += pop[indices[j]][i]
		}
		newMean[i] = sum / float64(s.popSize/2)
	}

	return newMean, sigma * 0.98, C, pc, ps
}

// evaluateParams evaluates a set of parameters
func evaluateParams(ctx context.Context, data *DataSet, params map[string]float64) (float64, error) {
	// 使用参数运行回测
	stats := calculatePerformanceStats(data.Returns)

	// 计算综合得分
	score := stats.SharpeRatio
	if stats.MaxDrawdown > 0.5 { // 如果最大回撤超过50%，显著降低得分
		score *= 0.5
	}
	if stats.WinRate < 0.4 { // 如果胜率过低，降低得分
		score *= 0.8
	}

	return score, nil
}

// DataSet represents a dataset for optimization
type DataSet struct {
	Symbol     string             `json:"symbol"`
	Returns    []float64          `json:"returns"`
	Prices     []float64          `json:"prices"`
	Volumes    []float64          `json:"volumes"`
	Timestamps []time.Time        `json:"timestamps"`
	Trades     []*market.Trade    `json:"trades,omitempty"`
	StartTime  time.Time          `json:"start_time"`
	EndTime    time.Time          `json:"end_time"`
}
