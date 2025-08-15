package orchestrator

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"qcat/internal/strategy/sdk"
)

// RankingManager manages strategy ranking and elimination
type RankingManager struct {
	strategies   map[string]*StrategyScore
	cooldownPool map[string]time.Time
	config       *RankingConfig
	mu           sync.RWMutex
}

// StrategyScore represents strategy performance score
type StrategyScore struct {
	ID            string
	Score         float64
	RiskAdjReturn float64
	Correlation   float64
	Volatility    float64
	LastUpdate    time.Time
}

// RankingConfig represents ranking configuration
type RankingConfig struct {
	EliminationThreshold float64       // 淘汰分数阈值
	CooldownPeriod       time.Duration // 冷却期
	MinStrategies        int           // 最小策略数量
	MaxCorrelation       float64       // 最大相关性
}

// NewRankingManager creates a new ranking manager
func NewRankingManager(config *RankingConfig) *RankingManager {
	return &RankingManager{
		strategies:   make(map[string]*StrategyScore),
		cooldownPool: make(map[string]time.Time),
		config:       config,
	}
}

// Handle implements TaskHandler interface
func (m *RankingManager) Handle(ctx context.Context) error {
	// 更新策略分数
	if err := m.updateScores(ctx); err != nil {
		return fmt.Errorf("failed to update scores: %w", err)
	}

	// 执行淘汰
	if err := m.eliminateStrategies(ctx); err != nil {
		return fmt.Errorf("failed to eliminate strategies: %w", err)
	}

	// 管理冷却池
	m.manageCooldownPool()

	return nil
}

// updateScores updates strategy scores
func (m *RankingManager) updateScores(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, strategy := range m.strategies {
		// 计算风险调整收益
		metrics := getStrategyMetrics(id)
		riskAdjReturn := calculateRiskAdjustedReturn(metrics)

		// 计算相关性惩罚
		correlation := calculateCorrelation(id, m.strategies)
		correlationPenalty := correlation * 0.2 // 20%权重

		// 计算波动率惩罚
		volatility := calculateVolatility(metrics)
		volatilityPenalty := volatility * 0.1 // 10%权重

		// 更新分数
		strategy.RiskAdjReturn = riskAdjReturn
		strategy.Correlation = correlation
		strategy.Volatility = volatility
		strategy.Score = riskAdjReturn - correlationPenalty - volatilityPenalty
		strategy.LastUpdate = time.Now()
	}

	return nil
}

// eliminateStrategies eliminates underperforming strategies
func (m *RankingManager) eliminateStrategies(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 如果策略数量已经是最小值，不执行淘汰
	if len(m.strategies) <= m.config.MinStrategies {
		return nil
	}

	// 按分数排序
	var scores []*StrategyScore
	for _, s := range m.strategies {
		scores = append(scores, s)
	}
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	// 找出需要淘汰的策略
	var toEliminate []string
	for i := len(scores) - 1; i >= 0; i-- {
		strategy := scores[i]
		if strategy.Score < m.config.EliminationThreshold {
			toEliminate = append(toEliminate, strategy.ID)
		}
	}

	// 执行淘汰
	for _, id := range toEliminate {
		if err := disableStrategy(id); err != nil {
			return fmt.Errorf("failed to disable strategy %s: %w", id, err)
		}
		delete(m.strategies, id)
		m.cooldownPool[id] = time.Now()
	}

	return nil
}

// manageCooldownPool manages the cooldown pool
func (m *RankingManager) manageCooldownPool() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for id, cooldownTime := range m.cooldownPool {
		if now.Sub(cooldownTime) > m.config.CooldownPeriod {
			delete(m.cooldownPool, id)
		}
	}
}

// IsInCooldown checks if a strategy is in cooldown
func (m *RankingManager) IsInCooldown(strategyID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cooldownTime, exists := m.cooldownPool[strategyID]
	if !exists {
		return false
	}
	return time.Since(cooldownTime) <= m.config.CooldownPeriod
}

// GetScore gets a strategy's score
func (m *RankingManager) GetScore(strategyID string) (*StrategyScore, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	score, exists := m.strategies[strategyID]
	if !exists {
		return nil, fmt.Errorf("strategy not found: %s", strategyID)
	}
	return score, nil
}

// Helper functions (to be implemented based on actual strategy metrics)
func getStrategyMetrics(strategyID string) *sdk.StrategyMetrics                            { return nil }
func calculateRiskAdjustedReturn(metrics *sdk.StrategyMetrics) float64                     { return 0 }
func calculateCorrelation(strategyID string, strategies map[string]*StrategyScore) float64 { return 0 }
func calculateVolatility(metrics *sdk.StrategyMetrics) float64                             { return 0 }
func disableStrategy(strategyID string) error                                              { return nil }
