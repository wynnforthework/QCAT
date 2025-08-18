package automl

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

// AdaptiveOptimizer 自适应优化器
type AdaptiveOptimizer struct {
	// 基础配置
	config *AdaptiveConfig
	
	// 历史性能记录
	performanceHistory []*PerformanceRecord
	mu                 sync.RWMutex
	
	// 当前优化策略
	currentStrategy *OptimizationStrategy
	
	// 算法注册表
	algorithmRegistry *OptimizationAlgorithmRegistry
	
	// 自适应调整器
	parameterAdjuster *ParameterAdjuster
	algorithmSelector *AlgorithmSelector
}

// AdaptiveConfig 自适应配置
type AdaptiveConfig struct {
	// 性能监控
	PerformanceWindow    int           `yaml:"performance_window"`     // 性能评估窗口大小
	MinPerformanceCount  int           `yaml:"min_performance_count"`  // 最小性能记录数
	PerformanceThreshold float64       `yaml:"performance_threshold"`  // 性能改进阈值
	EvaluationInterval   time.Duration `yaml:"evaluation_interval"`    // 评估间隔
	
	// 策略调整
	StrategyAdjustmentEnabled bool    `yaml:"strategy_adjustment_enabled"` // 是否启用策略调整
	AdjustmentThreshold       float64 `yaml:"adjustment_threshold"`        // 调整阈值
	MaxAdjustmentAttempts     int     `yaml:"max_adjustment_attempts"`     // 最大调整尝试次数
	
	// 算法选择
	AlgorithmSelectionEnabled bool     `yaml:"algorithm_selection_enabled"` // 是否启用算法选择
	AlgorithmWeights          []string `yaml:"algorithm_weights"`           // 算法权重配置
	SelectionStrategy         string   `yaml:"selection_strategy"`          // 选择策略
	
	// 参数调整
	ParameterAdjustmentEnabled bool    `yaml:"parameter_adjustment_enabled"` // 是否启用参数调整
	LearningRate               float64 `yaml:"learning_rate"`                // 学习率
	ExplorationRate            float64 `yaml:"exploration_rate"`             // 探索率
	ConvergenceThreshold       float64 `yaml:"convergence_threshold"`        // 收敛阈值
}

// PerformanceRecord 性能记录
type PerformanceRecord struct {
	Timestamp     time.Time
	Algorithm     string
	Strategy      string
	Parameters    map[string]float64
	Performance   *PerformanceMetrics
	ExecutionTime time.Duration
	Success       bool
	Error         string
}

// OptimizationStrategy 优化策略
type OptimizationStrategy struct {
	Algorithm     string                 `json:"algorithm"`
	Parameters    map[string]float64     `json:"parameters"`
	Weights       map[string]float64     `json:"weights"`
	Metadata      map[string]interface{} `json:"metadata"`
	LastUpdated   time.Time              `json:"last_updated"`
	SuccessCount  int                    `json:"success_count"`
	FailureCount  int                    `json:"failure_count"`
	AverageScore  float64                `json:"average_score"`
}

// ParameterAdjuster 参数调整器
type ParameterAdjuster struct {
	config *AdaptiveConfig
	mu     sync.RWMutex
	
	// 参数历史
	parameterHistory []*ParameterAdjustment
	adjustmentCount  int
}

// ParameterAdjustment 参数调整记录
type ParameterAdjustment struct {
	Timestamp    time.Time
	OldParams    map[string]float64
	NewParams    map[string]float64
	Reason       string
	Performance  float64
	Improvement  float64
	Success      bool
}

// AlgorithmSelector 算法选择器
type AlgorithmSelector struct {
	config *AdaptiveConfig
	mu     sync.RWMutex
	
	// 算法性能统计
	algorithmStats map[string]*AlgorithmStats
	selectionCount int
}

// AlgorithmStats 算法统计
type AlgorithmStats struct {
	Name           string
	UsageCount     int
	SuccessCount   int
	AverageScore   float64
	BestScore      float64
	LastUsed       time.Time
	AverageTime    time.Duration
	ConvergenceRate float64
}

// NewAdaptiveOptimizer 创建自适应优化器
func NewAdaptiveOptimizer(config *AdaptiveConfig) *AdaptiveOptimizer {
	if config == nil {
		config = &AdaptiveConfig{
			PerformanceWindow:          20,
			MinPerformanceCount:        5,
			PerformanceThreshold:       0.05,
			EvaluationInterval:         time.Minute * 5,
			StrategyAdjustmentEnabled:  true,
			AdjustmentThreshold:        0.1,
			MaxAdjustmentAttempts:      10,
			AlgorithmSelectionEnabled:  true,
			SelectionStrategy:          "weighted",
			ParameterAdjustmentEnabled: true,
			LearningRate:               0.1,
			ExplorationRate:            0.2,
			ConvergenceThreshold:       0.01,
		}
	}
	
	return &AdaptiveOptimizer{
		config:             config,
		performanceHistory: make([]*PerformanceRecord, 0),
		algorithmRegistry:  NewOptimizationAlgorithmRegistry(),
		parameterAdjuster:  NewParameterAdjuster(config),
		algorithmSelector:  NewAlgorithmSelector(config),
		currentStrategy:    &OptimizationStrategy{
			Algorithm:   "genetic",
			Parameters:  make(map[string]float64),
			Weights:     make(map[string]float64),
			Metadata:    make(map[string]interface{}),
			LastUpdated: time.Now(),
		},
	}
}

// Optimize 执行自适应优化
func (ao *AdaptiveOptimizer) Optimize(ctx context.Context, strategyName string, dataHash string, seed int64) (*OptimizationResult, error) {
	// 1. 评估当前策略性能
	ao.evaluateCurrentStrategy()
	
	// 2. 选择最优算法
	if ao.config.AlgorithmSelectionEnabled {
		selectedAlgorithm := ao.algorithmSelector.SelectAlgorithm(ao.performanceHistory)
		if selectedAlgorithm != "" && selectedAlgorithm != ao.currentStrategy.Algorithm {
			ao.currentStrategy.Algorithm = selectedAlgorithm
			ao.currentStrategy.LastUpdated = time.Now()
			fmt.Printf("Switched to algorithm: %s\n", selectedAlgorithm)
		}
	}
	
	// 3. 调整优化参数
	if ao.config.ParameterAdjustmentEnabled {
		adjustedParams := ao.parameterAdjuster.AdjustParameters(ao.currentStrategy, ao.performanceHistory)
		if adjustedParams != nil {
			ao.currentStrategy.Parameters = adjustedParams
			ao.currentStrategy.LastUpdated = time.Now()
			fmt.Printf("Adjusted parameters: %v\n", adjustedParams)
		}
	}
	
	// 4. 执行优化
	startTime := time.Now()
	result, err := ao.executeOptimization(ctx, strategyName, dataHash, seed)
	executionTime := time.Since(startTime)
	
	// 5. 记录性能
	ao.recordPerformance(result, executionTime, err)
	
	// 6. 更新策略
	ao.updateStrategy(result, err)
	
	return result, err
}

// evaluateCurrentStrategy 评估当前策略
func (ao *AdaptiveOptimizer) evaluateCurrentStrategy() {
	ao.mu.RLock()
	defer ao.mu.RUnlock()
	
	if len(ao.performanceHistory) < ao.config.MinPerformanceCount {
		return
	}
	
	// 计算最近性能
	recentPerformance := ao.calculateRecentPerformance()
	
	// 更新策略统计
	ao.currentStrategy.AverageScore = recentPerformance.AverageScore
	ao.currentStrategy.SuccessCount = recentPerformance.SuccessCount
	ao.currentStrategy.FailureCount = recentPerformance.FailureCount
}

// calculateRecentPerformance 计算最近性能
func (ao *AdaptiveOptimizer) calculateRecentPerformance() *PerformanceSummary {
	window := ao.config.PerformanceWindow
	if len(ao.performanceHistory) < window {
		window = len(ao.performanceHistory)
	}
	
	recent := ao.performanceHistory[len(ao.performanceHistory)-window:]
	
	var totalScore float64
	successCount := 0
	failureCount := 0
	
	for _, record := range recent {
		if record.Success {
			totalScore += record.Performance.ProfitRate
			successCount++
		} else {
			failureCount++
		}
	}
	
	averageScore := 0.0
	if successCount > 0 {
		averageScore = totalScore / float64(successCount)
	}
	
	return &PerformanceSummary{
		AverageScore: averageScore,
		SuccessCount: successCount,
		FailureCount: failureCount,
		TotalCount:   len(recent),
	}
}

// executeOptimization 执行优化
func (ao *AdaptiveOptimizer) executeOptimization(ctx context.Context, strategyName string, dataHash string, seed int64) (*OptimizationResult, error) {
	algorithm, exists := ao.algorithmRegistry.Get(ao.currentStrategy.Algorithm)
	if !exists {
		return nil, fmt.Errorf("algorithm not found: %s", ao.currentStrategy.Algorithm)
	}
	
	// 应用当前策略的参数
	ao.applyStrategyParameters(algorithm)
	
	// 执行优化
	return algorithm.Optimize(ctx, strategyName, dataHash, seed)
}

// applyStrategyParameters 应用策略参数
func (ao *AdaptiveOptimizer) applyStrategyParameters(algorithm AdvancedOptimizer) {
	// 根据算法类型应用不同的参数
	switch algo := algorithm.(type) {
	case *GeneticAlgorithm:
		if populationSize, ok := ao.currentStrategy.Parameters["population_size"]; ok {
			algo.PopulationSize = int(populationSize)
		}
		if generations, ok := ao.currentStrategy.Parameters["generations"]; ok {
			algo.Generations = int(generations)
		}
		if mutationRate, ok := ao.currentStrategy.Parameters["mutation_rate"]; ok {
			algo.MutationRate = mutationRate
		}
		if crossoverRate, ok := ao.currentStrategy.Parameters["crossover_rate"]; ok {
			algo.CrossoverRate = crossoverRate
		}
		
	case *ParticleSwarmOptimization:
		if particleCount, ok := ao.currentStrategy.Parameters["particle_count"]; ok {
			algo.ParticleCount = int(particleCount)
		}
		if iterations, ok := ao.currentStrategy.Parameters["iterations"]; ok {
			algo.Iterations = int(iterations)
		}
		if inertiaWeight, ok := ao.currentStrategy.Parameters["inertia_weight"]; ok {
			algo.InertiaWeight = inertiaWeight
		}
		if cognitiveWeight, ok := ao.currentStrategy.Parameters["cognitive_weight"]; ok {
			algo.CognitiveWeight = cognitiveWeight
		}
		if socialWeight, ok := ao.currentStrategy.Parameters["social_weight"]; ok {
			algo.SocialWeight = socialWeight
		}
		
	case *BayesianOptimization:
		if maxIterations, ok := ao.currentStrategy.Parameters["max_iterations"]; ok {
			algo.MaxIterations = int(maxIterations)
		}
		if acquisitionFunc, ok := ao.currentStrategy.Parameters["acquisition_func"]; ok {
			algo.AcquisitionFunc = fmt.Sprintf("%v", acquisitionFunc)
		}
	}
}

// recordPerformance 记录性能
func (ao *AdaptiveOptimizer) recordPerformance(result *OptimizationResult, executionTime time.Duration, err error) {
	ao.mu.Lock()
	defer ao.mu.Unlock()
	
	record := &PerformanceRecord{
		Timestamp:     time.Now(),
		Algorithm:     ao.currentStrategy.Algorithm,
		Strategy:      "adaptive",
		Parameters:    ao.currentStrategy.Parameters,
		ExecutionTime: executionTime,
		Success:       err == nil && result != nil,
	}
	
	if result != nil {
		record.Performance = result.Performance
	}
	
	if err != nil {
		record.Error = err.Error()
	}
	
	ao.performanceHistory = append(ao.performanceHistory, record)
	
	// 限制历史记录数量
	if len(ao.performanceHistory) > ao.config.PerformanceWindow*2 {
		ao.performanceHistory = ao.performanceHistory[len(ao.performanceHistory)-ao.config.PerformanceWindow:]
	}
}

// updateStrategy 更新策略
func (ao *AdaptiveOptimizer) updateStrategy(result *OptimizationResult, err error) {
	ao.mu.Lock()
	defer ao.mu.Unlock()
	
	if result != nil {
		// 更新算法统计
		ao.algorithmSelector.UpdateAlgorithmStats(ao.currentStrategy.Algorithm, result, err)
		
		// 更新参数调整器
		ao.parameterAdjuster.UpdateAdjustmentHistory(result, ao.currentStrategy.Parameters)
	}
}

// GetPerformanceHistory 获取性能历史
func (ao *AdaptiveOptimizer) GetPerformanceHistory() []*PerformanceRecord {
	ao.mu.RLock()
	defer ao.mu.RUnlock()
	
	history := make([]*PerformanceRecord, len(ao.performanceHistory))
	copy(history, ao.performanceHistory)
	return history
}

// GetCurrentStrategy 获取当前策略
func (ao *AdaptiveOptimizer) GetCurrentStrategy() *OptimizationStrategy {
	ao.mu.RLock()
	defer ao.mu.RUnlock()
	
	return ao.currentStrategy
}

// GetAlgorithmStats 获取算法统计
func (ao *AdaptiveOptimizer) GetAlgorithmStats() map[string]*AlgorithmStats {
	return ao.algorithmSelector.GetAlgorithmStats()
}

// PerformanceSummary 性能摘要
type PerformanceSummary struct {
	AverageScore float64
	SuccessCount int
	FailureCount int
	TotalCount   int
}

// NewParameterAdjuster 创建参数调整器
func NewParameterAdjuster(config *AdaptiveConfig) *ParameterAdjuster {
	return &ParameterAdjuster{
		config:            config,
		parameterHistory:  make([]*ParameterAdjustment, 0),
		adjustmentCount:   0,
	}
}

// AdjustParameters 调整参数
func (pa *ParameterAdjuster) AdjustParameters(strategy *OptimizationStrategy, history []*PerformanceRecord) map[string]float64 {
	pa.mu.Lock()
	defer pa.mu.Unlock()
	
	if len(history) < pa.config.MinPerformanceCount {
		return nil
	}
	
	// 分析最近性能趋势
	trend := pa.analyzePerformanceTrend(history)
	
	// 如果性能下降，调整参数
	if trend.Direction == "declining" && trend.Magnitude > pa.config.AdjustmentThreshold {
		newParams := pa.generateNewParameters(strategy.Parameters, trend)
		pa.recordAdjustment(strategy.Parameters, newParams, "performance_decline", 0, 0, false)
		return newParams
	}
	
	// 如果性能停滞，增加探索
	if trend.Direction == "stagnant" && pa.adjustmentCount < pa.config.MaxAdjustmentAttempts {
		newParams := pa.increaseExploration(strategy.Parameters)
		pa.recordAdjustment(strategy.Parameters, newParams, "stagnant_performance", 0, 0, false)
		return newParams
	}
	
	return nil
}

// analyzePerformanceTrend 分析性能趋势
func (pa *ParameterAdjuster) analyzePerformanceTrend(history []*PerformanceRecord) *PerformanceTrend {
	if len(history) < 3 {
		return &PerformanceTrend{Direction: "insufficient_data"}
	}
	
	// 计算最近几期的平均性能
	recent := history[len(history)-3:]
	var recentScores []float64
	
	for _, record := range recent {
		if record.Success {
			recentScores = append(recentScores, record.Performance.ProfitRate)
		}
	}
	
	if len(recentScores) < 2 {
		return &PerformanceTrend{Direction: "insufficient_data"}
	}
	
	// 计算趋势
	firstHalf := recentScores[:len(recentScores)/2]
	secondHalf := recentScores[len(recentScores)/2:]
	
	firstAvg := pa.calculateAverage(firstHalf)
	secondAvg := pa.calculateAverage(secondHalf)
	
	change := secondAvg - firstAvg
	magnitude := math.Abs(change)
	
	var direction string
	if change > pa.config.ConvergenceThreshold {
		direction = "improving"
	} else if change < -pa.config.ConvergenceThreshold {
		direction = "declining"
	} else {
		direction = "stagnant"
	}
	
	return &PerformanceTrend{
		Direction: direction,
		Magnitude: magnitude,
		Change:    change,
	}
}

// generateNewParameters 生成新参数
func (pa *ParameterAdjuster) generateNewParameters(currentParams map[string]float64, trend *PerformanceTrend) map[string]float64 {
	newParams := make(map[string]float64)
	
	for key, value := range currentParams {
		// 根据趋势调整参数
		adjustment := pa.calculateParameterAdjustment(key, value, trend)
		newParams[key] = value + adjustment
	}
	
	return newParams
}

// calculateParameterAdjustment 计算参数调整
func (pa *ParameterAdjuster) calculateParameterAdjustment(paramName string, currentValue float64, trend *PerformanceTrend) float64 {
	// 基于参数类型和历史调整记录计算调整量
	adjustment := 0.0
	
	switch paramName {
	case "learning_rate":
		if trend.Direction == "declining" {
			adjustment = -pa.config.LearningRate * currentValue
		} else if trend.Direction == "stagnant" {
			adjustment = pa.config.LearningRate * currentValue
		}
	case "mutation_rate":
		if trend.Direction == "stagnant" {
			adjustment = pa.config.ExplorationRate * 0.1
		}
	case "population_size":
		if trend.Direction == "stagnant" {
			adjustment = 10
		}
	case "iterations":
		if trend.Direction == "stagnant" {
			adjustment = 20
		}
	}
	
	return adjustment
}

// increaseExploration 增加探索
func (pa *ParameterAdjuster) increaseExploration(currentParams map[string]float64) map[string]float64 {
	newParams := make(map[string]float64)
	
	for key, value := range currentParams {
		switch key {
		case "mutation_rate":
			newParams[key] = math.Min(0.5, value+pa.config.ExplorationRate*0.1)
		case "exploration_rate":
			newParams[key] = math.Min(0.5, value+pa.config.ExplorationRate*0.1)
		case "population_size":
			newParams[key] = value + 20
		case "particle_count":
			newParams[key] = value + 10
		default:
			newParams[key] = value
		}
	}
	
	return newParams
}

// recordAdjustment 记录调整
func (pa *ParameterAdjuster) recordAdjustment(oldParams, newParams map[string]float64, reason string, performance, improvement float64, success bool) {
	adjustment := &ParameterAdjustment{
		Timestamp:   time.Now(),
		OldParams:   oldParams,
		NewParams:   newParams,
		Reason:      reason,
		Performance: performance,
		Improvement: improvement,
		Success:     success,
	}
	
	pa.parameterHistory = append(pa.parameterHistory, adjustment)
	pa.adjustmentCount++
	
	// 限制历史记录数量
	if len(pa.parameterHistory) > 50 {
		pa.parameterHistory = pa.parameterHistory[1:]
	}
}

// UpdateAdjustmentHistory 更新调整历史
func (pa *ParameterAdjuster) UpdateAdjustmentHistory(result *OptimizationResult, params map[string]float64) {
	if len(pa.parameterHistory) == 0 {
		return
	}
	
	lastAdjustment := pa.parameterHistory[len(pa.parameterHistory)-1]
	lastAdjustment.Performance = result.Performance.ProfitRate
	
	// 计算改进
	if len(pa.parameterHistory) > 1 {
		previousAdjustment := pa.parameterHistory[len(pa.parameterHistory)-2]
		lastAdjustment.Improvement = lastAdjustment.Performance - previousAdjustment.Performance
		lastAdjustment.Success = lastAdjustment.Improvement > 0
	}
}

// calculateAverage 计算平均值
func (pa *ParameterAdjuster) calculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// PerformanceTrend 性能趋势
type PerformanceTrend struct {
	Direction string  // "improving", "declining", "stagnant", "insufficient_data"
	Magnitude float64 // 变化幅度
	Change    float64 // 具体变化值
}

// NewAlgorithmSelector 创建算法选择器
func NewAlgorithmSelector(config *AdaptiveConfig) *AlgorithmSelector {
	return &AlgorithmSelector{
		config:         config,
		algorithmStats: make(map[string]*AlgorithmStats),
		selectionCount: 0,
	}
}

// SelectAlgorithm 选择算法
func (as *AlgorithmSelector) SelectAlgorithm(history []*PerformanceRecord) string {
	as.mu.Lock()
	defer as.mu.Unlock()
	
	// 更新算法统计
	as.updateAlgorithmStats(history)
	
	// 根据选择策略选择算法
	switch as.config.SelectionStrategy {
	case "best_performance":
		return as.selectBestPerformance()
	case "weighted":
		return as.selectWeighted()
	case "exploration":
		return as.selectExploration()
	default:
		return as.selectBestPerformance()
	}
}

// selectBestPerformance 选择最佳性能算法
func (as *AlgorithmSelector) selectBestPerformance() string {
	var bestAlgorithm string
	var bestScore float64
	
	for name, stats := range as.algorithmStats {
		if stats.UsageCount >= 3 && stats.AverageScore > bestScore {
			bestScore = stats.AverageScore
			bestAlgorithm = name
		}
	}
	
	return bestAlgorithm
}

// selectWeighted 加权选择
func (as *AlgorithmSelector) selectWeighted() string {
	// 简化的加权选择：考虑性能和探索
	totalWeight := 0.0
	algorithmScores := make(map[string]float64)
	
	for name, stats := range as.algorithmStats {
		if stats.UsageCount >= 2 {
			// 性能权重
			performanceWeight := stats.AverageScore * 0.7
			// 探索权重（使用次数少的算法获得更高权重）
			explorationWeight := (1.0 / float64(stats.UsageCount)) * 0.3
			
			score := performanceWeight + explorationWeight
			algorithmScores[name] = score
			totalWeight += score
		}
	}
	
	if totalWeight == 0 {
		return "genetic" // 默认算法
	}
	
	// 随机选择（基于权重）
	randValue := rand.Float64() * totalWeight
	currentWeight := 0.0
	
	for name, score := range algorithmScores {
		currentWeight += score
		if randValue <= currentWeight {
			return name
		}
	}
	
	return "genetic"
}

// selectExploration 探索选择
func (as *AlgorithmSelector) selectExploration() string {
	// 选择使用次数最少的算法
	var leastUsedAlgorithm string
	minUsage := math.MaxInt32
	
	for name, stats := range as.algorithmStats {
		if stats.UsageCount < minUsage {
			minUsage = stats.UsageCount
			leastUsedAlgorithm = name
		}
	}
	
	return leastUsedAlgorithm
}

// updateAlgorithmStats 更新算法统计
func (as *AlgorithmSelector) updateAlgorithmStats(history []*PerformanceRecord) {
	// 重置统计
	as.algorithmStats = make(map[string]*AlgorithmStats)
	
	// 统计每个算法的性能
	for _, record := range history {
		stats, exists := as.algorithmStats[record.Algorithm]
		if !exists {
			stats = &AlgorithmStats{
				Name: record.Algorithm,
			}
			as.algorithmStats[record.Algorithm] = stats
		}
		
		stats.UsageCount++
		stats.LastUsed = record.Timestamp
		stats.AverageTime += record.ExecutionTime
		
		if record.Success {
			stats.SuccessCount++
			score := record.Performance.ProfitRate
			stats.AverageScore = (stats.AverageScore*float64(stats.SuccessCount-1) + score) / float64(stats.SuccessCount)
			
			if score > stats.BestScore {
				stats.BestScore = score
			}
		}
	}
	
	// 计算平均时间和收敛率
	for _, stats := range as.algorithmStats {
		if stats.UsageCount > 0 {
			stats.AverageTime = stats.AverageTime / time.Duration(stats.UsageCount)
			stats.ConvergenceRate = float64(stats.SuccessCount) / float64(stats.UsageCount)
		}
	}
}

// UpdateAlgorithmStats 更新算法统计
func (as *AlgorithmSelector) UpdateAlgorithmStats(algorithmName string, result *OptimizationResult, err error) {
	as.mu.Lock()
	defer as.mu.Unlock()
	
	stats, exists := as.algorithmStats[algorithmName]
	if !exists {
		stats = &AlgorithmStats{
			Name: algorithmName,
		}
		as.algorithmStats[algorithmName] = stats
	}
	
	stats.UsageCount++
	stats.LastUsed = time.Now()
	
	if err == nil && result != nil {
		stats.SuccessCount++
		score := result.Performance.ProfitRate
		stats.AverageScore = (stats.AverageScore*float64(stats.SuccessCount-1) + score) / float64(stats.SuccessCount)
		
		if score > stats.BestScore {
			stats.BestScore = score
		}
	}
	
	stats.ConvergenceRate = float64(stats.SuccessCount) / float64(stats.UsageCount)
}

// GetAlgorithmStats 获取算法统计
func (as *AlgorithmSelector) GetAlgorithmStats() map[string]*AlgorithmStats {
	as.mu.RLock()
	defer as.mu.RUnlock()
	
	stats := make(map[string]*AlgorithmStats)
	for k, v := range as.algorithmStats {
		stats[k] = v
	}
	return stats
}
