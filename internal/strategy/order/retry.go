package order

import (
	"context"
	"fmt"
	"sync"
	"time"

	exch "qcat/internal/exchange"
)

// RetryConfig represents order retry configuration
type RetryConfig struct {
	MaxRetries  int
	RetryDelay  time.Duration
	MaxAttempts int
	BackoffBase float64
}

// RetryManager manages order retries
type RetryManager struct {
	manager *Manager
	config  *RetryConfig
	retries map[string]*RetryState
	mu      sync.RWMutex
}

// RetryState represents the state of order retries
type RetryState struct {
	OrderID     string
	Attempts    int
	NextRetry   time.Time
	LastError   error
	RetryReason string
}

// NewRetryManager creates a new retry manager
func NewRetryManager(manager *Manager, config *RetryConfig) *RetryManager {
	if config == nil {
		config = &RetryConfig{
			MaxRetries:  3,
			RetryDelay:  time.Second * 5,
			MaxAttempts: 5,
			BackoffBase: 2.0,
		}
	}

	return &RetryManager{
		manager: manager,
		config:  config,
		retries: make(map[string]*RetryState),
	}
}

// Start starts the retry manager
func (m *RetryManager) Start(ctx context.Context) {
	go m.processRetries(ctx)
}

// AddRetry adds an order for retry
func (m *RetryManager) AddRetry(orderID string, reason string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.retries[orderID]
	if !exists {
		state = &RetryState{
			OrderID:     orderID,
			Attempts:    0,
			NextRetry:   time.Now().Add(m.getRetryDelay(0)),
			LastError:   err,
			RetryReason: reason,
		}
		m.retries[orderID] = state
	} else {
		state.Attempts++
		state.NextRetry = time.Now().Add(m.getRetryDelay(state.Attempts))
		state.LastError = err
		state.RetryReason = reason
	}
}

// RemoveRetry removes an order from retry
func (m *RetryManager) RemoveRetry(orderID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.retries, orderID)
}

// GetRetryState returns the retry state for an order
func (m *RetryManager) GetRetryState(orderID string) (*RetryState, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.retries[orderID]
	return state, exists
}

// processRetries processes order retries
func (m *RetryManager) processRetries(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.retryOrders(ctx)
		}
	}
}

// retryOrders retries failed orders
func (m *RetryManager) retryOrders(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for orderID, state := range m.retries {
		if state.Attempts >= m.config.MaxRetries {
			delete(m.retries, orderID)
			continue
		}

		if now.Before(state.NextRetry) {
			continue
		}

		// Get order
		order, exists := m.manager.GetOrder(orderID)
		if !exists {
			delete(m.retries, orderID)
			continue
		}

		// Check if order needs retry
		if !m.shouldRetry(order) {
			delete(m.retries, orderID)
			continue
		}

		// Retry order
		if err := m.manager.RetryOrder(ctx, orderID, m.config.MaxRetries, m.getRetryDelay(state.Attempts)); err != nil {
			state.LastError = err
			state.Attempts++
			state.NextRetry = now.Add(m.getRetryDelay(state.Attempts))
		} else {
			delete(m.retries, orderID)
		}
	}
}

// shouldRetry checks if an order should be retried
func (m *RetryManager) shouldRetry(order *Order) bool {
	// Don't retry if order is not in a final state
	if string(order.Order.Status) != string(exch.OrderStatusRejected) &&
		string(order.Order.Status) != string(exch.OrderStatusCancelled) {
		// TODO: 待确认 - OrderStatusExpired 不存在，暂时注释掉
		// string(order.Order.Status) != string(exch.OrderStatusExpired) {
		return false
	}

	// Don't retry if order has exceeded max attempts
	if order.Retries >= m.config.MaxAttempts {
		return false
	}

	// Don't retry if retry delay hasn't elapsed
	if !order.RetryUntil.IsZero() && time.Now().Before(order.RetryUntil) {
		return false
	}

	return true
}

// getRetryDelay returns the retry delay for an attempt
func (m *RetryManager) getRetryDelay(attempt int) time.Duration {
	delay := m.config.RetryDelay
	for i := 0; i < attempt; i++ {
		delay = time.Duration(float64(delay) * m.config.BackoffBase)
	}
	return delay
}

// Error types
type ErrRetryExceeded struct {
	OrderID string
	Reason  string
}

func (e ErrRetryExceeded) Error() string {
	return fmt.Sprintf("max retries exceeded for order %s: %s", e.OrderID, e.Reason)
}

type ErrRetryFailed struct {
	OrderID string
	Reason  string
	Err     error
}

func (e ErrRetryFailed) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("failed to retry order %s: %s: %v", e.OrderID, e.Reason, e.Err)
	}
	return fmt.Sprintf("failed to retry order %s: %s", e.OrderID, e.Reason)
}

func (e ErrRetryFailed) Unwrap() error {
	return e.Err
}
