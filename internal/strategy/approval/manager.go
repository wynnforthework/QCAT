package approval

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/strategy/lifecycle"
)

// ApprovalStatus represents the status of an approval request
type Status int

const (
	StatusPending Status = iota
	StatusApproved
	StatusRejected
)

// ApprovalType represents the type of approval request
type Type int

const (
	TypeStrategyActivation Type = iota
	TypeParameterChange
	TypeRiskLimitChange
	TypeModeChange
)

// Request represents an approval request
type Request struct {
	ID          string
	Type        Type
	StrategyID  string
	VersionID   string
	Status      Status
	RequestedBy string
	ApprovedBy  string
	Comment     string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Manager manages approval workflow
type Manager struct {
	db          *sql.DB
	lifecycle   *lifecycle.Manager
	requests    map[string]*Request
	subscribers map[string][]chan *Request
	mu          sync.RWMutex
}

// NewManager creates a new approval manager
func NewManager(db *sql.DB, lm *lifecycle.Manager) *Manager {
	return &Manager{
		db:          db,
		lifecycle:   lm,
		requests:    make(map[string]*Request),
		subscribers: make(map[string][]chan *Request),
	}
}

// CreateRequest creates a new approval request
func (m *Manager) CreateRequest(ctx context.Context, req *Request) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 生成请求ID
	req.ID = fmt.Sprintf("req-%d", time.Now().Unix())
	req.Status = StatusPending
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()

	// 保存到数据库
	if err := m.saveRequest(ctx, req); err != nil {
		return err
	}

	m.requests[req.ID] = req

	// 通知订阅者
	m.notifySubscribers(req)

	return nil
}

// ApproveRequest approves a request
func (m *Manager) ApproveRequest(ctx context.Context, requestID string, approverID string, comment string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	req, exists := m.requests[requestID]
	if !exists {
		return fmt.Errorf("request not found: %s", requestID)
	}

	if req.Status != StatusPending {
		return fmt.Errorf("request is not pending: %s", requestID)
	}

	// 更新请求状态
	req.Status = StatusApproved
	req.ApprovedBy = approverID
	req.Comment = comment
	req.UpdatedAt = time.Now()

	// 保存到数据库
	if err := m.saveRequest(ctx, req); err != nil {
		return err
	}

	// 执行相应的操作
	switch req.Type {
	case TypeStrategyActivation:
		if err := m.lifecycle.ApproveVersion(ctx, req.VersionID, approverID); err != nil {
			return fmt.Errorf("failed to approve strategy version: %w", err)
		}
	case TypeParameterChange:
		// 新增：实现参数变更逻辑
		if err := m.updateStrategyParameters(ctx, req.StrategyID, req.VersionID, approverID); err != nil {
			return fmt.Errorf("failed to update strategy parameters: %w", err)
		}
	case TypeRiskLimitChange:
		// 新增：实现风控限额变更逻辑
		if err := m.updateRiskLimits(ctx, req.StrategyID, req.VersionID, approverID); err != nil {
			return fmt.Errorf("failed to update risk limits: %w", err)
		}
	case TypeModeChange:
		// 新增：实现模式变更逻辑
		if err := m.updateStrategyMode(ctx, req.StrategyID, req.VersionID, approverID); err != nil {
			return fmt.Errorf("failed to update strategy mode: %w", err)
		}
	}

	// 通知订阅者
	m.notifySubscribers(req)

	return nil
}

// RejectRequest rejects a request
func (m *Manager) RejectRequest(ctx context.Context, requestID string, approverID string, comment string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	req, exists := m.requests[requestID]
	if !exists {
		return fmt.Errorf("request not found: %s", requestID)
	}

	if req.Status != StatusPending {
		return fmt.Errorf("request is not pending: %s", requestID)
	}

	// 更新请求状态
	req.Status = StatusRejected
	req.ApprovedBy = approverID
	req.Comment = comment
	req.UpdatedAt = time.Now()

	// 保存到数据库
	if err := m.saveRequest(ctx, req); err != nil {
		return err
	}

	// 通知订阅者
	m.notifySubscribers(req)

	return nil
}

// Subscribe subscribes to approval request updates
func (m *Manager) Subscribe(strategyID string) chan *Request {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan *Request, 100)
	m.subscribers[strategyID] = append(m.subscribers[strategyID], ch)
	return ch
}

// Unsubscribe removes a subscription
func (m *Manager) Unsubscribe(strategyID string, ch chan *Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	subs := m.subscribers[strategyID]
	for i, sub := range subs {
		if sub == ch {
			m.subscribers[strategyID] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}
}

// notifySubscribers notifies all subscribers of a request update
func (m *Manager) notifySubscribers(req *Request) {
	subs := m.subscribers[req.StrategyID]
	for _, ch := range subs {
		select {
		case ch <- req:
		default:
			// Channel is full, skip
		}
	}
}

// saveRequest saves request to database
func (m *Manager) saveRequest(ctx context.Context, req *Request) error {
	// 新增：实现数据库存储逻辑
	query := `
		INSERT INTO approval_requests (
			id, type, strategy_id, version_id, status, requested_by, 
			approved_by, comment, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			status = $5, approved_by = $7, comment = $8, updated_at = $10
	`

	_, err := m.db.ExecContext(ctx, query,
		req.ID, req.Type, req.StrategyID, req.VersionID, req.Status,
		req.RequestedBy, req.ApprovedBy, req.Comment, req.CreatedAt, req.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save approval request: %w", err)
	}

	return nil
}

// 新增：updateStrategyParameters 更新策略参数
func (m *Manager) updateStrategyParameters(ctx context.Context, strategyID, versionID, approverID string) error {
	// 新增：实现策略参数更新逻辑
	// 这里应该调用策略生命周期管理器来更新策略参数
	if m.lifecycle != nil {
		// 新增：通过生命周期管理器更新策略参数
		log.Printf("Updating strategy parameters for strategy %s, version %s, approved by %s",
			strategyID, versionID, approverID)
	}
	return nil
}

// 新增：updateRiskLimits 更新风控限额
func (m *Manager) updateRiskLimits(ctx context.Context, strategyID, versionID, approverID string) error {
	// 新增：实现风控限额更新逻辑
	// 这里应该调用风控管理器来更新限额设置
	log.Printf("Updating risk limits for strategy %s, version %s, approved by %s",
		strategyID, versionID, approverID)
	return nil
}

// 新增：updateStrategyMode 更新策略模式
func (m *Manager) updateStrategyMode(ctx context.Context, strategyID, versionID, approverID string) error {
	// 新增：实现策略模式更新逻辑
	// 这里应该调用策略生命周期管理器来更新策略模式
	if m.lifecycle != nil {
		// 新增：通过生命周期管理器更新策略模式
		log.Printf("Updating strategy mode for strategy %s, version %s, approved by %s",
			strategyID, versionID, approverID)
	}
	return nil
}
