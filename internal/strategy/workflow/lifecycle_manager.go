package workflow

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// LifecycleState 生命周期状态
type LifecycleState int

const (
	LifecycleCreated LifecycleState = iota
	LifecycleInitializing
	LifecycleRunning
	LifecyclePaused
	LifecycleStopped
	LifecycleCompleted
	LifecycleFailed
	LifecycleArchived
)

func (ls LifecycleState) String() string {
	states := []string{
		"Created", "Initializing", "Running", "Paused",
		"Stopped", "Completed", "Failed", "Archived",
	}
	if int(ls) < len(states) {
		return states[ls]
	}
	return "Unknown"
}

// StateTransition 状态转换记录
type StateTransition struct {
	FromState   LifecycleState
	ToState     LifecycleState
	Timestamp   time.Time
	Reason      string
	Metadata    map[string]interface{}
}

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	// 回测指标
	SharpeRatio     float64 `json:"sharpe_ratio"`
	SortinoRatio    float64 `json:"sortino_ratio"`
	MaxDrawdown     float64 `json:"max_drawdown"`
	TotalReturn     float64 `json:"total_return"`
	WinRate         float64 `json:"win_rate"`
	ProfitFactor    float64 `json:"profit_factor"`

	// 优化指标
	OptimizationScore   float64            `json:"optimization_score"`
	OptimalParams       map[string]float64 `json:"optimal_params"`
	ParameterStability  float64            `json:"parameter_stability"`

	// 学习指标
	ModelAccuracy       float64            `json:"model_accuracy"`
	FeatureImportance   map[string]float64 `json:"feature_importance"`
	LearningProgress    float64            `json:"learning_progress"`

	// 应用指标
	ApplicationSuccess  bool               `json:"application_success"`
	AppliedParams       map[string]float64 `json:"applied_params"`
	ValidationScore     float64            `json:"validation_score"`

	// 时间指标
	LastUpdated         time.Time          `json:"last_updated"`
	UpdateCount         int64              `json:"update_count"`
}

// LifecycleConfig 生命周期配置
type LifecycleConfig struct {
	// 阶段超时配置
	StageTimeouts map[StrategyStage]time.Duration `yaml:"stage_timeouts"`

	// 性能阈值
	MinSharpeRatio      float64 `yaml:"min_sharpe_ratio"`
	MaxDrawdownLimit    float64 `yaml:"max_drawdown_limit"`
	MinOptimizationScore float64 `yaml:"min_optimization_score"`
	MinModelAccuracy    float64 `yaml:"min_model_accuracy"`

	// 重试配置
	MaxRetries          int           `yaml:"max_retries"`
	RetryDelay          time.Duration `yaml:"retry_delay"`

	// 自动推进配置
	AutoAdvance         bool          `yaml:"auto_advance"`
	AdvanceDelay        time.Duration `yaml:"advance_delay"`
}

// LifecycleManager 生命周期管理器
type LifecycleManager struct {
	strategyID string
	config     *LifecycleConfig

	// 状态管理
	currentState    LifecycleState
	stateHistory    []StateTransition
	stateMu         sync.RWMutex

	// 性能管理
	performanceMetrics *PerformanceMetrics
	metricsMu          sync.RWMutex

	// 运行状态
	isRunning bool
	runningMu sync.RWMutex

	// 上下文
	ctx    context.Context
	cancel context.CancelFunc
}

// NewLifecycleManager 创建生命周期管理器
func NewLifecycleManager(strategyID string) *LifecycleManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &LifecycleManager{
		strategyID:         strategyID,
		config:             GetDefaultLifecycleConfig(),
		currentState:       LifecycleCreated,
		stateHistory:       make([]StateTransition, 0),
		performanceMetrics: &PerformanceMetrics{
			OptimalParams:     make(map[string]float64),
			FeatureImportance: make(map[string]float64),
			AppliedParams:     make(map[string]float64),
			LastUpdated:       time.Now(),
		},
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start 启动生命周期管理器
func (lm *LifecycleManager) Start() error {
	lm.runningMu.Lock()
	defer lm.runningMu.Unlock()

	if lm.isRunning {
		return fmt.Errorf("lifecycle manager for strategy %s is already running", lm.strategyID)
	}

	log.Printf("启动策略 %s 的生命周期管理器", lm.strategyID)

	// 转换到初始化状态
	if err := lm.TransitionTo(LifecycleInitializing, "Manager started"); err != nil {
		return fmt.Errorf("failed to transition to initializing state: %w", err)
	}

	lm.isRunning = true
	log.Printf("策略 %s 的生命周期管理器启动完成", lm.strategyID)

	return nil
}

// Stop 停止生命周期管理器
func (lm *LifecycleManager) Stop() error {
	lm.runningMu.Lock()
	defer lm.runningMu.Unlock()

	if !lm.isRunning {
		return nil
	}

	log.Printf("停止策略 %s 的生命周期管理器", lm.strategyID)

	// 取消上下文
	lm.cancel()

	// 转换到停止状态
	if err := lm.TransitionTo(LifecycleStopped, "Manager stopped"); err != nil {
		log.Printf("Warning: failed to transition to stopped state: %v", err)
	}

	lm.isRunning = false
	log.Printf("策略 %s 的生命周期管理器已停止", lm.strategyID)

	return nil
}

// TransitionTo 转换到指定状态
func (lm *LifecycleManager) TransitionTo(newState LifecycleState, reason string) error {
	lm.stateMu.Lock()
	defer lm.stateMu.Unlock()

	// 检查状态转换是否有效
	if !lm.isValidTransition(lm.currentState, newState) {
		return fmt.Errorf("invalid state transition: %s -> %s", 
			lm.currentState.String(), newState.String())
	}

	// 记录状态转换
	transition := StateTransition{
		FromState: lm.currentState,
		ToState:   newState,
		Timestamp: time.Now(),
		Reason:    reason,
		Metadata:  make(map[string]interface{}),
	}

	lm.stateHistory = append(lm.stateHistory, transition)
	oldState := lm.currentState
	lm.currentState = newState

	log.Printf("策略 %s 状态转换: %s -> %s (原因: %s)", 
		lm.strategyID, oldState.String(), newState.String(), reason)

	return nil
}

// isValidTransition 检查状态转换是否有效
func (lm *LifecycleManager) isValidTransition(from, to LifecycleState) bool {
	// 定义有效的状态转换
	validTransitions := map[LifecycleState][]LifecycleState{
		LifecycleCreated: {LifecycleInitializing, LifecycleFailed},
		LifecycleInitializing: {LifecycleRunning, LifecycleFailed, LifecycleStopped},
		LifecycleRunning: {LifecyclePaused, LifecycleCompleted, LifecycleFailed, LifecycleStopped},
		LifecyclePaused: {LifecycleRunning, LifecycleStopped, LifecycleFailed},
		LifecycleStopped: {LifecycleRunning, LifecycleArchived},
		LifecycleCompleted: {LifecycleArchived, LifecycleRunning},
		LifecycleFailed: {LifecycleRunning, LifecycleArchived},
		LifecycleArchived: {},
	}

	allowedStates, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, allowedState := range allowedStates {
		if allowedState == to {
			return true
		}
	}

	return false
}

// UpdatePerformanceMetrics 更新性能指标
func (lm *LifecycleManager) UpdatePerformanceMetrics(metrics *PerformanceMetrics) {
	lm.metricsMu.Lock()
	defer lm.metricsMu.Unlock()

	// 更新指标
	if metrics.SharpeRatio != 0 {
		lm.performanceMetrics.SharpeRatio = metrics.SharpeRatio
	}
	if metrics.SortinoRatio != 0 {
		lm.performanceMetrics.SortinoRatio = metrics.SortinoRatio
	}
	if metrics.MaxDrawdown != 0 {
		lm.performanceMetrics.MaxDrawdown = metrics.MaxDrawdown
	}
	if metrics.TotalReturn != 0 {
		lm.performanceMetrics.TotalReturn = metrics.TotalReturn
	}
	if metrics.OptimizationScore != 0 {
		lm.performanceMetrics.OptimizationScore = metrics.OptimizationScore
	}
	if metrics.ModelAccuracy != 0 {
		lm.performanceMetrics.ModelAccuracy = metrics.ModelAccuracy
	}

	// 更新参数映射
	if len(metrics.OptimalParams) > 0 {
		for k, v := range metrics.OptimalParams {
			lm.performanceMetrics.OptimalParams[k] = v
		}
	}
	if len(metrics.FeatureImportance) > 0 {
		for k, v := range metrics.FeatureImportance {
			lm.performanceMetrics.FeatureImportance[k] = v
		}
	}
	if len(metrics.AppliedParams) > 0 {
		for k, v := range metrics.AppliedParams {
			lm.performanceMetrics.AppliedParams[k] = v
		}
	}

	lm.performanceMetrics.LastUpdated = time.Now()
	lm.performanceMetrics.UpdateCount++

	log.Printf("策略 %s 性能指标已更新", lm.strategyID)
}

// GetCurrentState 获取当前状态
func (lm *LifecycleManager) GetCurrentState() LifecycleState {
	lm.stateMu.RLock()
	defer lm.stateMu.RUnlock()
	return lm.currentState
}

// GetPerformanceMetrics 获取性能指标
func (lm *LifecycleManager) GetPerformanceMetrics() *PerformanceMetrics {
	lm.metricsMu.RLock()
	defer lm.metricsMu.RUnlock()

	// 返回副本
	metrics := *lm.performanceMetrics
	metrics.OptimalParams = make(map[string]float64)
	metrics.FeatureImportance = make(map[string]float64)
	metrics.AppliedParams = make(map[string]float64)

	for k, v := range lm.performanceMetrics.OptimalParams {
		metrics.OptimalParams[k] = v
	}
	for k, v := range lm.performanceMetrics.FeatureImportance {
		metrics.FeatureImportance[k] = v
	}
	for k, v := range lm.performanceMetrics.AppliedParams {
		metrics.AppliedParams[k] = v
	}

	return &metrics
}

// GetStateHistory 获取状态历史
func (lm *LifecycleManager) GetStateHistory() []StateTransition {
	lm.stateMu.RLock()
	defer lm.stateMu.RUnlock()

	// 返回副本
	history := make([]StateTransition, len(lm.stateHistory))
	copy(history, lm.stateHistory)

	return history
}

// GetDefaultLifecycleConfig 获取默认生命周期配置
func GetDefaultLifecycleConfig() *LifecycleConfig {
	return &LifecycleConfig{
		StageTimeouts: map[StrategyStage]time.Duration{
			StageOnboarding:  30 * time.Minute,
			StageBacktesting: 60 * time.Minute,
			StageOptimizing:  120 * time.Minute,
			StageLearning:    240 * time.Minute,
			StageApplying:    30 * time.Minute,
		},
		MinSharpeRatio:       0.5,
		MaxDrawdownLimit:     0.3,
		MinOptimizationScore: 0.7,
		MinModelAccuracy:     0.8,
		MaxRetries:           3,
		RetryDelay:           30 * time.Second,
		AutoAdvance:          true,
		AdvanceDelay:         5 * time.Second,
	}
}
