package optimizer

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"qcat/internal/config"
)

// EliminationManager manages strategy elimination
type EliminationManager struct {
	strategies   map[string]*StrategyState
	cooldownPool map[string]time.Time
	mu           sync.RWMutex
}

// StrategyState represents strategy state
type StrategyState struct {
	ID             string
	Returns        []float64
	RollingReturns []float64
	RollingVol     []float64
	RollingSharpe  []float64
	Correlations   map[string]float64
	LastEvaluation time.Time
	CooldownUntil  time.Time
	IsDisabled     bool
	DisabledReason string
}

// NewEliminationManager creates a new elimination manager
func NewEliminationManager() *EliminationManager {
	return &EliminationManager{
		strategies:   make(map[string]*StrategyState),
		cooldownPool: make(map[string]time.Time),
	}
}

// UpdateStrategyMetrics updates strategy metrics
func (m *EliminationManager) UpdateStrategyMetrics(id string, returns []float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.strategies[id]
	if !exists {
		state = &StrategyState{
			ID:           id,
			Correlations: make(map[string]float64),
		}
		m.strategies[id] = state
	}

	// 更新收益序列
	state.Returns = returns

	// 计算滚动窗口指标
	windowSize := 20 // Default fallback
	if config := config.GetAlgorithmConfig(); config != nil {
		windowSize = config.GetWindowSize()
	}
	if len(returns) >= windowSize {
		state.RollingReturns = calculateRollingReturns(returns, windowSize)
		state.RollingVol = calculateRollingVolatility(returns, windowSize)
		state.RollingSharpe = calculateRollingSharpe(returns, windowSize)
	}

	// 更新相关性
	for otherId, otherState := range m.strategies {
		if otherId != id {
			correlation := calculateCorrelation(state.Returns, otherState.Returns)
			state.Correlations[otherId] = correlation
			otherState.Correlations[id] = correlation
		}
	}

	state.LastEvaluation = time.Now()
	return nil
}

// EvaluateStrategies evaluates all strategies and returns elimination candidates
func (m *EliminationManager) EvaluateStrategies(ctx context.Context) ([]*EliminationCandidate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var candidates []*EliminationCandidate

	// 计算每个策略的综合得分
	scores := make(map[string]float64)
	for id, state := range m.strategies {
		if state.IsDisabled || time.Now().Before(state.CooldownUntil) {
			continue
		}

		// 计算风险调整后收益
		riskAdjustedReturn := calculateRiskAdjustedReturn(state)

		// 计算相关性惩罚
		correlationPenalty := calculateCorrelationPenalty(state)

		// 计算波动率惩罚
		volatilityPenalty := calculateVolatilityPenalty(state)

		// 综合得分
		scores[id] = riskAdjustedReturn - correlationPenalty - volatilityPenalty
	}

	// 排序策略
	var sortedStrategies []string
	for id := range scores {
		sortedStrategies = append(sortedStrategies, id)
	}
	sort.Slice(sortedStrategies, func(i, j int) bool {
		return scores[sortedStrategies[i]] < scores[sortedStrategies[j]]
	})

	// 选择末尾策略作为淘汰候选
	eliminationRatio := 0.2 // Default fallback
	if config := config.GetAlgorithmConfig(); config != nil {
		eliminationRatio = config.Elimination.PerformanceThreshold
	}
	
	eliminationCount := int(float64(len(sortedStrategies)) * eliminationRatio)
	for i := 0; i < eliminationCount && i < len(sortedStrategies); i++ {
		id := sortedStrategies[i]
		candidates = append(candidates, &EliminationCandidate{
			StrategyID: id,
			Score:      scores[id],
			Reason:     "Low performance score",
		})
	}

	return candidates, nil
}

// DisableStrategy disables a strategy
func (m *EliminationManager) DisableStrategy(id string, duration time.Duration, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.strategies[id]
	if !exists {
		return fmt.Errorf("strategy not found: %s", id)
	}

	state.IsDisabled = true
	state.DisabledReason = reason
	state.CooldownUntil = time.Now().Add(duration)
	m.cooldownPool[id] = state.CooldownUntil

	return nil
}

// EnableStrategy enables a strategy
func (m *EliminationManager) EnableStrategy(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.strategies[id]
	if !exists {
		return fmt.Errorf("strategy not found: %s", id)
	}

	if time.Now().Before(state.CooldownUntil) {
		return fmt.Errorf("strategy is in cooldown until %v", state.CooldownUntil)
	}

	state.IsDisabled = false
	state.DisabledReason = ""
	delete(m.cooldownPool, id)

	return nil
}

// EliminationCandidate represents a strategy elimination candidate
type EliminationCandidate struct {
	StrategyID string
	Score      float64
	Reason     string
}

// Helper functions

func calculateRollingReturns(returns []float64, window int) []float64 {
	if len(returns) < window {
		return nil
	}

	rolling := make([]float64, len(returns)-window+1)
	for i := 0; i <= len(returns)-window; i++ {
		sum := 0.0
		for j := 0; j < window; j++ {
			sum += returns[i+j]
		}
		rolling[i] = sum
	}
	return rolling
}

func calculateRollingVolatility(returns []float64, window int) []float64 {
	if len(returns) < window {
		return nil
	}

	rolling := make([]float64, len(returns)-window+1)
	for i := 0; i <= len(returns)-window; i++ {
		var sum, sumSquared float64
		for j := 0; j < window; j++ {
			sum += returns[i+j]
			sumSquared += returns[i+j] * returns[i+j]
		}
		mean := sum / float64(window)
		variance := sumSquared/float64(window) - mean*mean
		rolling[i] = math.Sqrt(variance)
	}
	return rolling
}

func calculateRollingSharpe(returns []float64, window int) []float64 {
	if len(returns) < window {
		return nil
	}

	rolling := make([]float64, len(returns)-window+1)
	for i := 0; i <= len(returns)-window; i++ {
		var sum float64
		for j := 0; j < window; j++ {
			sum += returns[i+j]
		}
		mean := sum / float64(window)
		vol := calculateVolatility(returns[i : i+window])
		if vol > 0 {
			rolling[i] = mean / vol
		}
	}
	return rolling
}

func calculateCorrelation(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 2 {
		return 0
	}

	var sumX, sumY, sumXY, sumX2, sumY2 float64
	n := float64(len(x))

	for i := 0; i < len(x); i++ {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
		sumY2 += y[i] * y[i]
	}

	numerator := sumXY - (sumX * sumY / n)
	denominator := math.Sqrt((sumX2 - sumX*sumX/n) * (sumY2 - sumY*sumY/n))

	if denominator == 0 {
		return 0
	}
	return numerator / denominator
}

func calculateRiskAdjustedReturn(state *StrategyState) float64 {
	if len(state.RollingSharpe) == 0 {
		return 0
	}
	return state.RollingSharpe[len(state.RollingSharpe)-1]
}

func calculateCorrelationPenalty(state *StrategyState) float64 {
	if len(state.Correlations) == 0 {
		return 0
	}

	sum := 0.0
	for _, corr := range state.Correlations {
		if corr > 0 {
			sum += corr
		}
	}
	return sum / float64(len(state.Correlations))
}

func calculateVolatilityPenalty(state *StrategyState) float64 {
	if len(state.RollingVol) == 0 {
		return 0
	}
	return state.RollingVol[len(state.RollingVol)-1]
}

// calculateVolatility calculates volatility of returns
func calculateVolatility(returns []float64) float64 {
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
	variance /= float64(len(returns) - 1)

	return math.Sqrt(variance)
}
