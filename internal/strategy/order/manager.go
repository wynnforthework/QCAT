package order

import (
	"context"
	"fmt"
	"sync"
	"time"

	"qcat/internal/exchange"
	"qcat/internal/strategy/signal"
)

// Manager manages order flow
type Manager struct {
	exchange  exchange.Exchange
	signals   *signal.DefaultProcessor
	orders    map[string]*Order
	callbacks map[string][]OrderCallback
	mu        sync.RWMutex
}

// Order represents an order with additional metadata
type Order struct {
	*exchange.Order
	Signal     *signal.Signal
	Metadata   map[string]interface{}
	UpdatedAt  time.Time
	Retries    int
	RetryUntil time.Time
}

// OrderCallback represents an order callback function
type OrderCallback func(*Order)

// NewManager creates a new order manager
func NewManager(exchange exchange.Exchange, signals *signal.DefaultProcessor) *Manager {
	return &Manager{
		exchange:  exchange,
		signals:   signals,
		orders:    make(map[string]*Order),
		callbacks: make(map[string][]OrderCallback),
	}
}

// PlaceOrder places an order
func (m *Manager) PlaceOrder(ctx context.Context, signal *signal.Signal) (*Order, error) {
	// Process signal
	if err := m.signals.Process(signal); err != nil {
		return nil, fmt.Errorf("failed to process signal: %w", err)
	}

	// Get order
	order, err := m.exchange.GetOrder(ctx, signal.Symbol, signal.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	// Create order wrapper
	orderWrapper := &Order{
		Order:     order,
		Signal:    signal,
		Metadata:  make(map[string]interface{}),
		UpdatedAt: time.Now(),
	}

	// Store order
	m.mu.Lock()
	m.orders[order.ID] = orderWrapper
	m.mu.Unlock()

	// Notify callbacks
	m.notifyCallbacks(orderWrapper)

	return orderWrapper, nil
}

// CancelOrder cancels an order
func (m *Manager) CancelOrder(ctx context.Context, orderID string) error {
	m.mu.RLock()
	order, exists := m.orders[orderID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("order not found: %s", orderID)
	}

	// Create cancel request
	req := &exchange.OrderCancelRequest{
		Symbol:  order.Symbol,
		OrderID: orderID,
	}

	// Cancel order
	resp, err := m.exchange.CancelOrder(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("order cancellation rejected: %v", resp.Error)
	}

	// Update order
	order.Order = resp.Order
	order.UpdatedAt = time.Now()

	// Notify callbacks
	m.notifyCallbacks(order)

	return nil
}

// GetOrder returns an order by ID
func (m *Manager) GetOrder(orderID string) (*Order, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	order, exists := m.orders[orderID]
	return order, exists
}

// ListOrders returns all orders
func (m *Manager) ListOrders() []*Order {
	m.mu.RLock()
	defer m.mu.RUnlock()

	orders := make([]*Order, 0, len(m.orders))
	for _, order := range m.orders {
		orders = append(orders, order)
	}
	return orders
}

// AddCallback adds an order callback
func (m *Manager) AddCallback(orderID string, callback OrderCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callbacks[orderID] = append(m.callbacks[orderID], callback)
}

// RemoveCallback removes an order callback
func (m *Manager) RemoveCallback(orderID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.callbacks, orderID)
}

// OnOrder handles order updates
func (m *Manager) OnOrder(order *exchange.Order) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find existing order
	orderWrapper, exists := m.orders[order.ID]
	if !exists {
		orderWrapper = &Order{
			Order:     order,
			Metadata:  make(map[string]interface{}),
			UpdatedAt: time.Now(),
		}
		m.orders[order.ID] = orderWrapper
	} else {
		orderWrapper.Order = order
		orderWrapper.UpdatedAt = time.Now()
	}

	// Notify callbacks
	m.notifyCallbacks(orderWrapper)
}

// notifyCallbacks notifies order callbacks
func (m *Manager) notifyCallbacks(order *Order) {
	if callbacks, exists := m.callbacks[order.ID]; exists {
		for _, callback := range callbacks {
			callback(order)
		}
	}
}

// RetryOrder retries a failed order
func (m *Manager) RetryOrder(ctx context.Context, orderID string, maxRetries int, retryDelay time.Duration) error {
	m.mu.Lock()
	order, exists := m.orders[orderID]
	m.mu.Unlock()

	if !exists {
		return fmt.Errorf("order not found: %s", orderID)
	}

	if order.Status != exchange.OrderStatusRejected {
		return fmt.Errorf("order is not in rejected state: %s", orderID)
	}

	if order.Retries >= maxRetries {
		return fmt.Errorf("max retries exceeded for order: %s", orderID)
	}

	if !order.RetryUntil.IsZero() && time.Now().Before(order.RetryUntil) {
		return fmt.Errorf("retry delay not elapsed for order: %s", orderID)
	}

	// Create new signal
	signal := order.Signal
	signal.ID = fmt.Sprintf("%s-retry-%d", signal.ID, order.Retries+1)
	signal.Status = "pending"
	signal.OrderID = ""
	signal.CreatedAt = time.Now()
	signal.UpdatedAt = time.Now()

	// Place order
	newOrder, err := m.PlaceOrder(ctx, signal)
	if err != nil {
		return fmt.Errorf("failed to retry order: %w", err)
	}

	// Update retry info
	newOrder.Retries = order.Retries + 1
	newOrder.RetryUntil = time.Now().Add(retryDelay)

	return nil
}

// CleanupOrders removes completed orders older than the specified duration
func (m *Manager) CleanupOrders(age time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-age)
	for id, order := range m.orders {
		if order.UpdatedAt.Before(cutoff) &&
			(order.Status == exchange.OrderStatusFilled ||
				order.Status == exchange.OrderStatusCancelled ||
				order.Status == exchange.OrderStatusRejected) {
			delete(m.orders, id)
			delete(m.callbacks, id)
		}
	}
}
