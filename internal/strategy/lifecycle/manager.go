package lifecycle

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"qcat/internal/strategy/sdk"
)

// StrategyState represents strategy lifecycle state
type State int

const (
	StateDraft State = iota
	StateInReview
	StatePaperTrading
	StateShadowTrading
	StateCanaryTrading
	StateLiveTrading
	StatePaused
	StateDisabled
)

// StrategyVersion represents a strategy version
type Version struct {
	ID          string
	StrategyID  string
	Version     string
	Config      *sdk.StrategyConfig
	State       State
	Performance *Performance
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Performance represents strategy performance metrics
type Performance struct {
	Sharpe       float64
	SortinoRatio float64
	MaxDrawdown  float64
	WinRate      float64
	ProfitFactor float64
	TotalPnL     float64
	TotalTrades  int
}

// Manager manages strategy lifecycle
type Manager struct {
	db        *sql.DB
	versions  map[string]*Version
	approvals map[string]bool
	mu        sync.RWMutex
}

// NewManager creates a new lifecycle manager
func NewManager(db *sql.DB) *Manager {
	return &Manager{
		db:        db,
		versions:  make(map[string]*Version),
		approvals: make(map[string]bool),
	}
}

// CreateVersion creates a new strategy version
func (m *Manager) CreateVersion(ctx context.Context, strategyID string, config *sdk.StrategyConfig) (*Version, error) {
	version := &Version{
		ID:         fmt.Sprintf("%s-v%d", strategyID, time.Now().Unix()),
		StrategyID: strategyID,
		Version:    fmt.Sprintf("v%d", time.Now().Unix()),
		Config:     config,
		State:      StateDraft,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// 保存到数据库
	if err := m.saveVersion(ctx, version); err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.versions[version.ID] = version
	m.mu.Unlock()

	return version, nil
}

// SubmitForReview submits a strategy version for review
func (m *Manager) SubmitForReview(ctx context.Context, versionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	version, exists := m.versions[versionID]
	if !exists {
		return fmt.Errorf("version not found: %s", versionID)
	}

	if version.State != StateDraft {
		return fmt.Errorf("invalid state transition: %v -> %v", version.State, StateInReview)
	}

	version.State = StateInReview
	version.UpdatedAt = time.Now()

	return m.saveVersion(ctx, version)
}

// ApproveVersion approves a strategy version
func (m *Manager) ApproveVersion(ctx context.Context, versionID string, approverID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	version, exists := m.versions[versionID]
	if !exists {
		return fmt.Errorf("version not found: %s", versionID)
	}

	if version.State != StateInReview {
		return fmt.Errorf("invalid state transition: %v -> %v", version.State, StatePaperTrading)
	}

	// 记录审批
	m.approvals[versionID] = true

	// 更新状态
	version.State = StatePaperTrading
	version.UpdatedAt = time.Now()

	return m.saveVersion(ctx, version)
}

// PromoteToShadow promotes a strategy to shadow trading
func (m *Manager) PromoteToShadow(ctx context.Context, versionID string) error {
	return m.promoteStrategy(ctx, versionID, StateShadowTrading)
}

// PromoteToCanary promotes a strategy to canary trading
func (m *Manager) PromoteToCanary(ctx context.Context, versionID string) error {
	return m.promoteStrategy(ctx, versionID, StateCanaryTrading)
}

// PromoteToLive promotes a strategy to live trading
func (m *Manager) PromoteToLive(ctx context.Context, versionID string) error {
	return m.promoteStrategy(ctx, versionID, StateLiveTrading)
}

// UpdatePerformance updates strategy performance metrics
func (m *Manager) UpdatePerformance(ctx context.Context, versionID string, perf *Performance) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	version, exists := m.versions[versionID]
	if !exists {
		return fmt.Errorf("version not found: %s", versionID)
	}

	version.Performance = perf
	version.UpdatedAt = time.Now()

	return m.saveVersion(ctx, version)
}

// DisableStrategy disables a strategy
func (m *Manager) DisableStrategy(ctx context.Context, versionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	version, exists := m.versions[versionID]
	if !exists {
		return fmt.Errorf("version not found: %s", versionID)
	}

	version.State = StateDisabled
	version.UpdatedAt = time.Now()

	return m.saveVersion(ctx, version)
}

// promoteStrategy promotes a strategy to the next state
func (m *Manager) promoteStrategy(ctx context.Context, versionID string, targetState State) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	version, exists := m.versions[versionID]
	if !exists {
		return fmt.Errorf("version not found: %s", versionID)
	}

	// 检查状态转换是否有效
	if !isValidStateTransition(version.State, targetState) {
		return fmt.Errorf("invalid state transition: %v -> %v", version.State, targetState)
	}

	// 更新状态
	version.State = targetState
	version.UpdatedAt = time.Now()

	return m.saveVersion(ctx, version)
}

// saveVersion saves version to database
func (m *Manager) saveVersion(ctx context.Context, version *Version) error {
	// 新增：实现数据库存储逻辑
	query := `
		INSERT INTO strategy_versions (
			id, strategy_id, version, config, state, performance, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			config = $4, state = $5, performance = $6, updated_at = $8
	`

	// 新增：序列化配置和性能数据
	configJSON, err := json.Marshal(version.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	performanceJSON, err := json.Marshal(version.Performance)
	if err != nil {
		return fmt.Errorf("failed to marshal performance: %w", err)
	}

	_, err = m.db.ExecContext(ctx, query,
		version.ID, version.StrategyID, version.Version, configJSON,
		version.State, performanceJSON, version.CreatedAt, version.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save strategy version: %w", err)
	}

	return nil
}

// isValidStateTransition checks if state transition is valid
func isValidStateTransition(from, to State) bool {
	validTransitions := map[State][]State{
		StateDraft:         {StateInReview},
		StateInReview:      {StatePaperTrading, StateDraft},
		StatePaperTrading:  {StateShadowTrading, StateDisabled},
		StateShadowTrading: {StateCanaryTrading, StateDisabled},
		StateCanaryTrading: {StateLiveTrading, StateDisabled},
		StateLiveTrading:   {StatePaused, StateDisabled},
		StatePaused:        {StateLiveTrading, StateDisabled},
	}

	transitions, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, validTo := range transitions {
		if to == validTo {
			return true
		}
	}

	return false
}
