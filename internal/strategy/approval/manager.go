package approval

import (
	"context"
	"database/sql"
	"fmt"
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
		// TODO: 实现参数变更逻辑
	case TypeRiskLimitChange:
		// TODO: 实现风控限额变更逻辑
	case TypeModeChange:
		// TODO: 实现模式变更逻辑
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
	// TODO: 实现数据库存储逻辑
	return nil
}
