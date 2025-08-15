package orderbook

import (
	"context"
	"fmt"
	"sync"
	"time"

	"qcat/internal/cache"
)

// Manager manages multiple order books
type Manager struct {
	books     map[string]*OrderBook
	cache     cache.Cacher
	snapshots map[string]time.Time
	mu        sync.RWMutex
}

// NewManager creates a new order book manager
func NewManager(cache cache.Cacher) *Manager {
	return &Manager{
		books:     make(map[string]*OrderBook),
		cache:     cache,
		snapshots: make(map[string]time.Time),
	}
}

// GetOrderBook returns an order book for a symbol
func (m *Manager) GetOrderBook(symbol string) *OrderBook {
	m.mu.Lock()
	defer m.mu.Unlock()

	book, exists := m.books[symbol]
	if !exists {
		book = NewOrderBook(symbol)
		m.books[symbol] = book
	}
	return book
}

// UpdateOrderBook updates an order book with new data
func (m *Manager) UpdateOrderBook(symbol string, bids, asks []Level, timestamp time.Time) error {
	book := m.GetOrderBook(symbol)
	book.Update(bids, asks, timestamp)

	// Cache the snapshot
	if err := m.cacheSnapshot(symbol); err != nil {
		return fmt.Errorf("failed to cache snapshot: %w", err)
	}

	return nil
}

// cacheSnapshot caches the current state of an order book
func (m *Manager) cacheSnapshot(symbol string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	book, exists := m.books[symbol]
	if !exists {
		return fmt.Errorf("order book not found: %s", symbol)
	}

	// Only cache if enough time has passed since last snapshot
	lastSnapshot, exists := m.snapshots[symbol]
	if exists && time.Since(lastSnapshot) < time.Second {
		return nil
	}

	snapshot := book.GetSnapshot(20) // Cache top 20 levels
	if err := m.cache.SetOrderBook(context.Background(), symbol, snapshot, 5*time.Second); err != nil {
		return fmt.Errorf("failed to cache order book: %w", err)
	}

	m.snapshots[symbol] = time.Now()
	return nil
}

// GetMidPrice returns the mid price for a symbol
func (m *Manager) GetMidPrice(symbol string) float64 {
	m.mu.RLock()
	book, exists := m.books[symbol]
	m.mu.RUnlock()

	if !exists {
		return 0
	}
	return book.GetMidPrice()
}

// GetSpread returns the bid-ask spread for a symbol
func (m *Manager) GetSpread(symbol string) float64 {
	m.mu.RLock()
	book, exists := m.books[symbol]
	m.mu.RUnlock()

	if !exists {
		return 0
	}
	return book.GetSpread()
}

// GetVWAP returns the VWAP for a given quantity
func (m *Manager) GetVWAP(symbol string, quantity float64, side string) (float64, bool) {
	m.mu.RLock()
	book, exists := m.books[symbol]
	m.mu.RUnlock()

	if !exists {
		return 0, false
	}

	if side == "buy" {
		return book.Asks.GetVWAP(quantity)
	}
	return book.Bids.GetVWAP(quantity)
}

// GetDepth returns the total depth up to a price
func (m *Manager) GetDepth(symbol string, price float64, side string) float64 {
	m.mu.RLock()
	book, exists := m.books[symbol]
	m.mu.RUnlock()

	if !exists {
		return 0
	}

	if side == "buy" {
		return book.Asks.GetDepth(price)
	}
	return book.Bids.GetDepth(price)
}

// GetSnapshot returns a snapshot of an order book
func (m *Manager) GetSnapshot(symbol string, depth int) map[string]interface{} {
	m.mu.RLock()
	book, exists := m.books[symbol]
	m.mu.RUnlock()

	if !exists {
		return nil
	}
	return book.GetSnapshot(depth)
}
