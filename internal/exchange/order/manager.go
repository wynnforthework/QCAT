package order

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	exch "qcat/internal/exchange"
)

// Manager manages order operations
type Manager struct {
	db          *sql.DB
	exchange    exch.Exchange
	orders      map[string]*exch.Order
	subscribers map[string][]chan *exch.Order
	mu          sync.RWMutex
}

// NewManager creates a new order manager
func NewManager(db *sql.DB, exchange exch.Exchange) *Manager {
	m := &Manager{
		db:          db,
		exchange:    exchange,
		orders:      make(map[string]*exch.Order),
		subscribers: make(map[string][]chan *exch.Order),
	}

	// Start order status monitor
	go m.monitorOrders()

	return m
}

// Subscribe subscribes to order updates for a symbol
func (m *Manager) Subscribe(symbol string) chan *exch.Order {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan *exch.Order, 100)
	m.subscribers[symbol] = append(m.subscribers[symbol], ch)
	return ch
}

// Unsubscribe removes a subscription
func (m *Manager) Unsubscribe(symbol string, ch chan *exch.Order) {
	m.mu.Lock()
	defer m.mu.Unlock()

	subs := m.subscribers[symbol]
	for i, sub := range subs {
		if sub == ch {
			m.subscribers[symbol] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}
}

// PlaceOrder places a new order
func (m *Manager) PlaceOrder(ctx context.Context, req *exch.OrderRequest) (*exch.OrderResponse, error) {
	// Place order on exchange
	resp, err := m.exchange.PlaceOrder(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to place order: %w", err)
	}

	if !resp.Success {
		return resp, nil
	}

	// Store order in database
	if err := m.storeOrder(resp.Order); err != nil {
		log.Printf("Failed to store order: %v", err)
	}

	// Update local cache
	m.updateOrder(resp.Order)

	return resp, nil
}

// CancelOrder cancels an existing order
func (m *Manager) CancelOrder(ctx context.Context, req *exch.OrderCancelRequest) (*exch.OrderResponse, error) {
	// Cancel order on exchange
	resp, err := m.exchange.CancelOrder(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel order: %w", err)
	}

	if !resp.Success {
		return resp, nil
	}

	// Update order in database
	if err := m.storeOrder(resp.Order); err != nil {
		log.Printf("Failed to store order: %v", err)
	}

	// Update local cache
	m.updateOrder(resp.Order)

	return resp, nil
}

// CancelAllOrders cancels all open orders for a symbol
func (m *Manager) CancelAllOrders(ctx context.Context, symbol string) error {
	// Cancel orders on exchange
	if err := m.exchange.CancelAllOrders(ctx, symbol); err != nil {
		return fmt.Errorf("failed to cancel all orders: %w", err)
	}

	// Get updated orders
	orders, err := m.exchange.GetOpenOrders(ctx, symbol)
	if err != nil {
		return fmt.Errorf("failed to get open orders: %w", err)
	}

	// Update orders in database and cache
	for _, order := range orders {
		if err := m.storeOrder(order); err != nil {
			log.Printf("Failed to store order: %v", err)
		}
		m.updateOrder(order)
	}

	return nil
}

// GetOrder returns an order by ID
func (m *Manager) GetOrder(ctx context.Context, symbol, orderID string) (*exch.Order, error) {
	// Check local cache first
	m.mu.RLock()
	if order, exists := m.orders[orderID]; exists {
		m.mu.RUnlock()
		return order, nil
	}
	m.mu.RUnlock()

	// Get order from exchange
	order, err := m.exchange.GetOrder(ctx, symbol, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	// Store order in database
	if err := m.storeOrder(order); err != nil {
		log.Printf("Failed to store order: %v", err)
	}

	// Update local cache
	m.updateOrder(order)

	return order, nil
}

// GetOpenOrders returns all open orders for a symbol
func (m *Manager) GetOpenOrders(ctx context.Context, symbol string) ([]*exch.Order, error) {
	// Get orders from exchange
	orders, err := m.exchange.GetOpenOrders(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get open orders: %w", err)
	}

	// Update orders in database and cache
	for _, order := range orders {
		if err := m.storeOrder(order); err != nil {
			log.Printf("Failed to store order: %v", err)
		}
		m.updateOrder(order)
	}

	return orders, nil
}

// GetOrderHistory returns historical orders for a symbol
func (m *Manager) GetOrderHistory(ctx context.Context, symbol string, startTime, endTime time.Time) ([]*exch.Order, error) {
	// Get orders from exchange
	orders, err := m.exchange.GetOrderHistory(ctx, symbol, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get order history: %w", err)
	}

	// Update orders in database and cache
	for _, order := range orders {
		if err := m.storeOrder(order); err != nil {
			log.Printf("Failed to store order: %v", err)
		}
		m.updateOrder(order)
	}

	return orders, nil
}

// storeOrder stores an order in the database
func (m *Manager) storeOrder(order *exch.Order) error {
	query := `
		INSERT INTO orders (
			id, exchange_order_id, client_order_id, symbol, side,
			type, status, price, quantity, filled_qty,
			remaining_qty, avg_price, fee, fee_currency, created_at,
			updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16
		)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			filled_qty = EXCLUDED.filled_qty,
			remaining_qty = EXCLUDED.remaining_qty,
			avg_price = EXCLUDED.avg_price,
			fee = EXCLUDED.fee,
			updated_at = EXCLUDED.updated_at
	`

	_, err := m.db.Exec(query,
		order.ID,
		order.ExchangeID,
		order.ClientOrderID,
		order.Symbol,
		order.Side,
		order.Type,
		order.Status,
		order.Price,
		order.Quantity,
		order.FilledQty,
		order.RemainingQty,
		order.AvgPrice,
		order.Fee,
		order.FeeCurrency,
		order.CreatedAt,
		order.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to store order: %w", err)
	}

	return nil
}

// updateOrder updates the local order cache and notifies subscribers
func (m *Manager) updateOrder(order *exch.Order) {
	m.mu.Lock()
	m.orders[order.ID] = order
	m.mu.Unlock()

	// Notify subscribers
	m.notifySubscribers(order)
}

// notifySubscribers notifies all subscribers of an order update
func (m *Manager) notifySubscribers(order *exch.Order) {
	m.mu.RLock()
	subs := m.subscribers[order.Symbol]
	m.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- order:
		default:
			// Channel is full, skip
		}
	}
}

// monitorOrders periodically checks order status
func (m *Manager) monitorOrders() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.RLock()
		orders := make(map[string]*exch.Order)
		for id, order := range m.orders {
			if order.Status != string(exch.OrderStatusFilled) &&
				order.Status != string(exch.OrderStatusCancelled) &&
				order.Status != string(exch.OrderStatusRejected) &&
				order.Status != "EXPIRED" {
				orders[id] = order
			}
		}
		m.mu.RUnlock()

		for _, order := range orders {
			ctx := context.Background()
			updated, err := m.GetOrder(ctx, order.Symbol, order.ID)
			if err != nil {
				log.Printf("Failed to get order status: %v", err)
				continue
			}

			if updated.Status != order.Status {
				if err := m.storeOrder(updated); err != nil {
					log.Printf("Failed to store order: %v", err)
				}
				m.updateOrder(updated)
			}
		}
	}
}
