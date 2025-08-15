package recovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	exch "qcat/internal/exchange"
	"qcat/internal/strategy/order"
	"qcat/internal/strategy/state"
)

// Manager manages strategy recovery
type Manager struct {
	stateManager *state.Manager
	orderManager *order.Manager
	exchange     exch.Exchange
	recoveryLogs map[string][]*RecoveryLog
	mu           sync.RWMutex
}

// RecoveryLog represents a recovery log entry
type RecoveryLog struct {
	ID        string                 `json:"id"`
	Strategy  string                 `json:"strategy"`
	Symbol    string                 `json:"symbol"`
	Type      RecoveryType           `json:"type"`
	Status    RecoveryStatus         `json:"status"`
	Error     string                 `json:"error"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// RecoveryType represents the type of recovery
type RecoveryType string

const (
	RecoveryTypeError    RecoveryType = "error"
	RecoveryTypeTimeout  RecoveryType = "timeout"
	RecoveryTypeShutdown RecoveryType = "shutdown"
	RecoveryTypeRestart  RecoveryType = "restart"
	RecoveryTypeFailover RecoveryType = "failover"
	RecoveryTypeRollback RecoveryType = "rollback"
)

// RecoveryStatus represents the status of recovery
type RecoveryStatus string

const (
	RecoveryStatusPending   RecoveryStatus = "pending"
	RecoveryStatusRunning   RecoveryStatus = "running"
	RecoveryStatusCompleted RecoveryStatus = "completed"
	RecoveryStatusFailed    RecoveryStatus = "failed"
)

// NewManager creates a new recovery manager
func NewManager(stateManager *state.Manager, orderManager *order.Manager, exchange exch.Exchange) *Manager {
	return &Manager{
		stateManager: stateManager,
		orderManager: orderManager,
		exchange:     exchange,
		recoveryLogs: make(map[string][]*RecoveryLog),
	}
}

// HandleError handles a strategy error
func (m *Manager) HandleError(ctx context.Context, strategyID string, err error) error {
	// Get strategy state
	state, getErr := m.stateManager.GetState(ctx, strategyID)
	if getErr != nil {
		return fmt.Errorf("failed to get strategy state: %w", getErr)
	}

	// Create recovery log
	log := &RecoveryLog{
		ID:        fmt.Sprintf("%s-%d", strategyID, time.Now().UnixNano()),
		Strategy:  state.Strategy,
		Symbol:    state.Symbol,
		Type:      RecoveryTypeError,
		Status:    RecoveryStatusPending,
		Error:     err.Error(),
		Metadata:  make(map[string]interface{}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store recovery log
	m.mu.Lock()
	m.recoveryLogs[strategyID] = append(m.recoveryLogs[strategyID], log)
	m.mu.Unlock()

	// Update strategy state
	state.Status = "error"
	state.LastError = err.Error()
	state.ErrorCount++
	if err := m.stateManager.UpdateState(ctx, state); err != nil {
		return fmt.Errorf("failed to update strategy state: %w", err)
	}

	// Start recovery process
	go m.recover(ctx, log)

	return nil
}

// HandleTimeout handles a strategy timeout
func (m *Manager) HandleTimeout(ctx context.Context, strategyID string) error {
	// Get strategy state
	state, err := m.stateManager.GetState(ctx, strategyID)
	if err != nil {
		return fmt.Errorf("failed to get strategy state: %w", err)
	}

	// Create recovery log
	log := &RecoveryLog{
		ID:        fmt.Sprintf("%s-%d", strategyID, time.Now().UnixNano()),
		Strategy:  state.Strategy,
		Symbol:    state.Symbol,
		Type:      RecoveryTypeTimeout,
		Status:    RecoveryStatusPending,
		Error:     "strategy timeout",
		Metadata:  make(map[string]interface{}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store recovery log
	m.mu.Lock()
	m.recoveryLogs[strategyID] = append(m.recoveryLogs[strategyID], log)
	m.mu.Unlock()

	// Update strategy state
	state.Status = "error"
	state.LastError = "strategy timeout"
	state.ErrorCount++
	if err := m.stateManager.UpdateState(ctx, state); err != nil {
		return fmt.Errorf("failed to update strategy state: %w", err)
	}

	// Start recovery process
	go m.recover(ctx, log)

	return nil
}

// HandleShutdown handles a strategy shutdown
func (m *Manager) HandleShutdown(ctx context.Context, strategyID string) error {
	// Get strategy state
	state, err := m.stateManager.GetState(ctx, strategyID)
	if err != nil {
		return fmt.Errorf("failed to get strategy state: %w", err)
	}

	// Create recovery log
	log := &RecoveryLog{
		ID:        fmt.Sprintf("%s-%d", strategyID, time.Now().UnixNano()),
		Strategy:  state.Strategy,
		Symbol:    state.Symbol,
		Type:      RecoveryTypeShutdown,
		Status:    RecoveryStatusPending,
		Metadata:  make(map[string]interface{}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store recovery log
	m.mu.Lock()
	m.recoveryLogs[strategyID] = append(m.recoveryLogs[strategyID], log)
	m.mu.Unlock()

	// Update strategy state
	state.Status = "stopped"
	state.StoppedAt = time.Now()
	if err := m.stateManager.UpdateState(ctx, state); err != nil {
		return fmt.Errorf("failed to update strategy state: %w", err)
	}

	// Start recovery process
	go m.recover(ctx, log)

	return nil
}

// GetRecoveryLogs returns recovery logs for a strategy
func (m *Manager) GetRecoveryLogs(strategyID string) []*RecoveryLog {
	m.mu.RLock()
	defer m.mu.RUnlock()

	logs := m.recoveryLogs[strategyID]
	if logs == nil {
		return []*RecoveryLog{}
	}
	return logs
}

// recover performs the recovery process
func (m *Manager) recover(ctx context.Context, log *RecoveryLog) {
	// Update log status
	log.Status = RecoveryStatusRunning
	log.UpdatedAt = time.Now()

	// Get strategy state
	state, err := m.stateManager.GetState(ctx, log.Strategy)
	if err != nil {
		log.Status = RecoveryStatusFailed
		log.Error = fmt.Sprintf("failed to get strategy state: %v", err)
		log.UpdatedAt = time.Now()
		return
	}

	// Cancel open orders
	orders := m.orderManager.ListOrders()
	for _, order := range orders {
		if order.Signal.Strategy == log.Strategy {
			if err := m.orderManager.CancelOrder(ctx, order.ID); err != nil {
				log.Status = RecoveryStatusFailed
				log.Error = fmt.Sprintf("failed to cancel order %s: %v", order.ID, err)
				log.UpdatedAt = time.Now()
				return
			}
		}
	}

	// Close positions if needed
	if log.Type == RecoveryTypeShutdown {
		position, err := m.exchange.GetPosition(ctx, log.Symbol)
		if err != nil {
			log.Status = RecoveryStatusFailed
			log.Error = fmt.Sprintf("failed to get position: %v", err)
			log.UpdatedAt = time.Now()
			return
		}

		if position != nil && position.Quantity > 0 {
			// Create close order
			req := &exch.OrderRequest{
				Symbol:     log.Symbol,
				Side:       string(exch.OrderSideSell),   // 显式转换为 string
				Type:       string(exch.OrderTypeMarket), // 显式转换为 string
				Quantity:   position.Quantity,
				ReduceOnly: true,
			}

			// Place order
			resp, err := m.exchange.PlaceOrder(ctx, req)
			if err != nil {
				log.Status = RecoveryStatusFailed
				log.Error = fmt.Sprintf("failed to close position: %v", err)
				log.UpdatedAt = time.Now()
				return
			}

			if !resp.Success {
				log.Status = RecoveryStatusFailed
				log.Error = fmt.Sprintf("failed to close position: %v", resp.Error)
				log.UpdatedAt = time.Now()
				return
			}
		}
	}

	// Update strategy state
	if log.Type == RecoveryTypeError || log.Type == RecoveryTypeTimeout {
		state.Status = "running"
		state.LastError = ""
		if err := m.stateManager.UpdateState(ctx, state); err != nil {
			log.Status = RecoveryStatusFailed
			log.Error = fmt.Sprintf("failed to update strategy state: %v", err)
			log.UpdatedAt = time.Now()
			return
		}
	}

	// Update log status
	log.Status = RecoveryStatusCompleted
	log.UpdatedAt = time.Now()
}
