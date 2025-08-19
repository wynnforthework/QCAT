package optimizer

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"

	"qcat/internal/config"
	"qcat/internal/database"
)

// EliminationManager manages strategy elimination
type EliminationManager struct {
	strategies   map[string]*StrategyState
	cooldownPool map[string]time.Time
	banditArms   map[string]*BanditArm
	db           *database.DB
	config       *config.Config
	mu           sync.RWMutex

	// 自动化执行相关
	autoExecutionEnabled bool
	lastEvaluationTime   time.Time
	evaluationInterval   time.Duration
}

// BanditArm represents a multi-armed bandit arm for strategy comparison
type BanditArm struct {
	StrategyID    string
	Pulls         int
	TotalReward   float64
	AverageReward float64
	Confidence    float64
	LastPull      time.Time
}

// EliminationConfig holds elimination configuration
type EliminationConfig struct {
	PerformanceThreshold  float64
	CooldownDuration      time.Duration
	EvaluationInterval    time.Duration
	MinObservationPeriod  time.Duration
	MaxCooldownStrategies int
	BanditExplorationRate float64
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

	// 扩展状态信息
	EliminationCount    int
	LastEliminationTime time.Time
	PerformanceScore    float64
	RiskScore           float64
	DiversityScore      float64
	ObservationPeriod   time.Duration
	CreatedAt           time.Time
	Status              string // active, disabled, eliminated, cooldown
}

// NewEliminationManager creates a new elimination manager
func NewEliminationManager(db *database.DB, cfg *config.Config) *EliminationManager {
	return &EliminationManager{
		strategies:           make(map[string]*StrategyState),
		cooldownPool:         make(map[string]time.Time),
		banditArms:           make(map[string]*BanditArm),
		db:                   db,
		config:               cfg,
		autoExecutionEnabled: true,
		evaluationInterval:   time.Hour * 6, // 默认6小时评估一次
	}
}

// NewEliminationManagerSimple creates a simple elimination manager without database
func NewEliminationManagerSimple() *EliminationManager {
	return &EliminationManager{
		strategies:           make(map[string]*StrategyState),
		cooldownPool:         make(map[string]time.Time),
		banditArms:           make(map[string]*BanditArm),
		autoExecutionEnabled: false,
		evaluationInterval:   time.Hour * 6,
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

	// 计算滚动窗口指标 - 从配置获取窗口大小
	windowSize := 20 // Default fallback
	if algorithmConfig := config.GetAlgorithmConfig(); algorithmConfig != nil {
		windowSize = algorithmConfig.GetWindowSize()
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

// EliminationDecision represents a decision made by the elimination system
type EliminationDecision struct {
	StrategyID  string
	Action      string // "eliminate", "disable", "cooldown", "keep"
	Duration    time.Duration
	Reason      string
	Confidence  float64
	BanditScore float64
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

// ExecuteAutomaticElimination 执行自动淘汰逻辑
func (m *EliminationManager) ExecuteAutomaticElimination(ctx context.Context) error {
	if !m.autoExecutionEnabled {
		return fmt.Errorf("automatic execution is disabled")
	}

	// 检查是否到了评估时间
	if time.Since(m.lastEvaluationTime) < m.evaluationInterval {
		return nil // 还没到评估时间
	}

	log.Printf("Starting automatic strategy elimination evaluation")

	// 1. 评估所有策略
	candidates, err := m.EvaluateStrategies(ctx)
	if err != nil {
		return fmt.Errorf("failed to evaluate strategies: %w", err)
	}

	if len(candidates) == 0 {
		log.Printf("No elimination candidates found")
		m.lastEvaluationTime = time.Now()
		return nil
	}

	// 2. 使用多臂赌博机算法进行最终决策
	eliminationDecisions, err := m.makeBanditBasedDecisions(ctx, candidates)
	if err != nil {
		return fmt.Errorf("failed to make bandit-based decisions: %w", err)
	}

	// 3. 执行淘汰决策
	for _, decision := range eliminationDecisions {
		if err := m.executeEliminationDecision(ctx, decision); err != nil {
			log.Printf("Failed to execute elimination decision for strategy %s: %v",
				decision.StrategyID, err)
			continue
		}
		log.Printf("Successfully executed elimination decision for strategy %s: %s",
			decision.StrategyID, decision.Action)
	}

	// 4. 更新评估时间
	m.lastEvaluationTime = time.Now()

	// 5. 清理过期的冷却池条目
	if err := m.cleanupCooldownPool(); err != nil {
		log.Printf("Warning: failed to cleanup cooldown pool: %v", err)
	}

	log.Printf("Automatic elimination evaluation completed, processed %d decisions",
		len(eliminationDecisions))
	return nil
}

// makeBanditBasedDecisions 使用多臂赌博机算法做出淘汰决策
func (m *EliminationManager) makeBanditBasedDecisions(ctx context.Context, candidates []*EliminationCandidate) ([]*EliminationDecision, error) {
	var decisions []*EliminationDecision

	for _, candidate := range candidates {
		// 获取或创建赌博机臂
		arm, exists := m.banditArms[candidate.StrategyID]
		if !exists {
			arm = &BanditArm{
				StrategyID:    candidate.StrategyID,
				Pulls:         0,
				TotalReward:   0,
				AverageReward: 0,
				Confidence:    1.0,
				LastPull:      time.Now(),
			}
			m.banditArms[candidate.StrategyID] = arm
		}

		// 计算UCB1分数 (Upper Confidence Bound)
		ucbScore := m.calculateUCB1Score(arm, len(m.banditArms))

		// 基于UCB1分数和策略表现做出决策
		decision := m.makeEliminationDecision(candidate, arm, ucbScore)
		decisions = append(decisions, decision)

		// 更新赌博机臂
		m.updateBanditArm(arm, candidate.Score)
	}

	return decisions, nil
}

// calculateUCB1Score 计算UCB1分数
func (m *EliminationManager) calculateUCB1Score(arm *BanditArm, totalArms int) float64 {
	if arm.Pulls == 0 {
		return math.Inf(1) // 未拉过的臂优先级最高
	}

	explorationRate := 2.0 // 探索率
	if m.config != nil {
		// 从配置获取探索率
		explorationRate = 1.4 // 默认值
	}

	confidence := explorationRate * math.Sqrt(math.Log(float64(totalArms))/float64(arm.Pulls))
	return arm.AverageReward + confidence
}

// makeEliminationDecision 基于多臂赌博机结果做出淘汰决策
func (m *EliminationManager) makeEliminationDecision(candidate *EliminationCandidate, arm *BanditArm, ucbScore float64) *EliminationDecision {
	decision := &EliminationDecision{
		StrategyID:  candidate.StrategyID,
		Reason:      candidate.Reason,
		BanditScore: ucbScore,
	}

	// 决策逻辑：基于分数和历史表现
	if candidate.Score < -1.0 && arm.AverageReward < -0.5 {
		// 表现极差，直接淘汰
		decision.Action = "eliminate"
		decision.Duration = 0
		decision.Confidence = 0.9
		decision.Reason = fmt.Sprintf("Poor performance: score=%.3f, avg_reward=%.3f",
			candidate.Score, arm.AverageReward)
	} else if candidate.Score < -0.5 && arm.AverageReward < 0 {
		// 表现较差，长期禁用
		decision.Action = "disable"
		decision.Duration = time.Hour * 24 * 7 // 7天
		decision.Confidence = 0.7
		decision.Reason = fmt.Sprintf("Consistently poor performance: score=%.3f", candidate.Score)
	} else if candidate.Score < 0 {
		// 表现一般，短期冷却
		decision.Action = "cooldown"
		decision.Duration = time.Hour * 24 // 1天
		decision.Confidence = 0.5
		decision.Reason = fmt.Sprintf("Temporary underperformance: score=%.3f", candidate.Score)
	} else {
		// 保持观察
		decision.Action = "keep"
		decision.Duration = 0
		decision.Confidence = 0.3
		decision.Reason = "Monitoring performance"
	}

	return decision
}

// updateBanditArm 更新赌博机臂的统计信息
func (m *EliminationManager) updateBanditArm(arm *BanditArm, reward float64) {
	arm.Pulls++
	arm.TotalReward += reward
	arm.AverageReward = arm.TotalReward / float64(arm.Pulls)
	arm.LastPull = time.Now()

	// 计算置信度（基于拉取次数）
	arm.Confidence = math.Min(1.0, float64(arm.Pulls)/100.0)
}

// executeEliminationDecision 执行淘汰决策
func (m *EliminationManager) executeEliminationDecision(ctx context.Context, decision *EliminationDecision) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.strategies[decision.StrategyID]
	if !exists {
		return fmt.Errorf("strategy not found: %s", decision.StrategyID)
	}

	switch decision.Action {
	case "eliminate":
		return m.eliminateStrategyInternal(state, decision.Reason)
	case "disable":
		return m.disableStrategyInternal(state, decision.Duration, decision.Reason)
	case "cooldown":
		return m.cooldownStrategyInternal(state, decision.Duration, decision.Reason)
	case "keep":
		// 只记录决策，不采取行动
		log.Printf("Strategy %s kept under observation: %s", decision.StrategyID, decision.Reason)
		return nil
	default:
		return fmt.Errorf("unknown elimination action: %s", decision.Action)
	}
}

// eliminateStrategyInternal 内部淘汰策略方法
func (m *EliminationManager) eliminateStrategyInternal(state *StrategyState, reason string) error {
	state.Status = "eliminated"
	state.IsDisabled = true
	state.DisabledReason = reason
	state.EliminationCount++
	state.LastEliminationTime = time.Now()

	// 从活跃策略中移除
	delete(m.strategies, state.ID)

	// 保存到数据库（如果可用）
	if m.db != nil {
		if err := m.saveEliminationRecord(state, "eliminate", reason); err != nil {
			log.Printf("Warning: failed to save elimination record: %v", err)
		}
	}

	log.Printf("Strategy %s eliminated: %s", state.ID, reason)
	return nil
}

// disableStrategyInternal 内部禁用策略方法
func (m *EliminationManager) disableStrategyInternal(state *StrategyState, duration time.Duration, reason string) error {
	state.Status = "disabled"
	state.IsDisabled = true
	state.DisabledReason = reason
	state.CooldownUntil = time.Now().Add(duration)
	m.cooldownPool[state.ID] = state.CooldownUntil

	// 保存到数据库（如果可用）
	if m.db != nil {
		if err := m.saveEliminationRecord(state, "disable", reason); err != nil {
			log.Printf("Warning: failed to save elimination record: %v", err)
		}
	}

	log.Printf("Strategy %s disabled for %v: %s", state.ID, duration, reason)
	return nil
}

// cooldownStrategyInternal 内部冷却策略方法
func (m *EliminationManager) cooldownStrategyInternal(state *StrategyState, duration time.Duration, reason string) error {
	state.Status = "cooldown"
	state.IsDisabled = true
	state.DisabledReason = reason
	state.CooldownUntil = time.Now().Add(duration)
	m.cooldownPool[state.ID] = state.CooldownUntil

	log.Printf("Strategy %s in cooldown for %v: %s", state.ID, duration, reason)
	return nil
}

// saveEliminationRecord 保存淘汰记录到数据库
func (m *EliminationManager) saveEliminationRecord(state *StrategyState, action, reason string) error {
	if m.db == nil {
		return fmt.Errorf("database not available")
	}

	query := `
		INSERT INTO strategy_elimination_records (
			strategy_id, action, reason, elimination_time,
			performance_score, risk_score, diversity_score
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := m.db.ExecContext(context.Background(), query,
		state.ID,
		action,
		reason,
		time.Now(),
		state.PerformanceScore,
		state.RiskScore,
		state.DiversityScore,
	)

	return err
}

// cleanupCooldownPool 清理过期的冷却池条目
func (m *EliminationManager) cleanupCooldownPool() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	var expiredStrategies []string

	// 找出过期的策略
	for strategyID, cooldownUntil := range m.cooldownPool {
		if now.After(cooldownUntil) {
			expiredStrategies = append(expiredStrategies, strategyID)
		}
	}

	// 清理过期条目并重新激活策略
	for _, strategyID := range expiredStrategies {
		delete(m.cooldownPool, strategyID)

		// 重新激活策略
		if state, exists := m.strategies[strategyID]; exists {
			state.Status = "active"
			state.IsDisabled = false
			state.DisabledReason = ""
			state.CooldownUntil = time.Time{}
			log.Printf("Strategy %s reactivated after cooldown", strategyID)
		}
	}

	log.Printf("Cleaned up %d expired cooldown entries", len(expiredStrategies))
	return nil
}

// GetCooldownPoolStatus 获取冷却池状态
func (m *EliminationManager) GetCooldownPoolStatus() map[string]time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]time.Time)
	for id, until := range m.cooldownPool {
		status[id] = until
	}
	return status
}

// GetStrategyStates 获取所有策略状态
func (m *EliminationManager) GetStrategyStates() map[string]*StrategyState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	states := make(map[string]*StrategyState)
	for id, state := range m.strategies {
		// 创建副本以避免并发问题
		stateCopy := *state
		states[id] = &stateCopy
	}
	return states
}

// SetAutoExecution 设置自动执行开关
func (m *EliminationManager) SetAutoExecution(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.autoExecutionEnabled = enabled
	log.Printf("Automatic elimination execution set to: %v", enabled)
}

// SetEvaluationInterval 设置评估间隔
func (m *EliminationManager) SetEvaluationInterval(interval time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.evaluationInterval = interval
	log.Printf("Evaluation interval set to: %v", interval)
}
