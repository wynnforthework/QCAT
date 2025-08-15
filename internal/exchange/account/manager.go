package account

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/cache"
	exch "qcat/internal/exchange"
)

// Manager manages account information
type Manager struct {
	db          *sql.DB
	cache       cache.Cacher
	exchange    exch.Exchange
	balances    map[string]*exch.AccountBalance
	subscribers map[string][]chan *exch.AccountBalance
	mu          sync.RWMutex
}

// NewManager creates a new account manager
func NewManager(db *sql.DB, cache cache.Cacher, exchange exch.Exchange) *Manager {
	m := &Manager{
		db:          db,
		cache:       cache,
		exchange:    exchange,
		balances:    make(map[string]*exch.AccountBalance),
		subscribers: make(map[string][]chan *exch.AccountBalance),
	}

	// Start balance monitor
	go m.monitorBalances()

	return m
}

// Subscribe subscribes to balance updates for an asset
func (m *Manager) Subscribe(asset string) chan *exch.AccountBalance {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan *exch.AccountBalance, 100)
	m.subscribers[asset] = append(m.subscribers[asset], ch)
	return ch
}

// Unsubscribe removes a subscription
func (m *Manager) Unsubscribe(asset string, ch chan *exch.AccountBalance) {
	m.mu.Lock()
	defer m.mu.Unlock()

	subs := m.subscribers[asset]
	for i, sub := range subs {
		if sub == ch {
			m.subscribers[asset] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}
}

// GetBalance returns the current balance for an asset
func (m *Manager) GetBalance(ctx context.Context, asset string) (*exch.AccountBalance, error) {
	// Check cache first
	var balance exch.AccountBalance
	err := m.cache.Get(ctx, fmt.Sprintf("balance:%s", asset), &balance)
	if err == nil {
		return &balance, nil
	}

	// Get from exchange
	balances, err := m.exchange.GetAccountBalance(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	balancePtr, exists := balances[asset]
	if !exists {
		return nil, fmt.Errorf("balance not found for asset: %s", asset)
	}

	// Cache the balance
	if err := m.cache.Set(ctx, fmt.Sprintf("balance:%s", asset), balancePtr, time.Minute); err != nil {
		log.Printf("Failed to cache balance: %v", err)
	}

	// Update local cache and notify subscribers
	m.updateBalance(balancePtr)

	return balancePtr, nil
}

// GetAllBalances returns all account balances
func (m *Manager) GetAllBalances(ctx context.Context) (map[string]*exch.AccountBalance, error) {
	// Get from exchange
	balances, err := m.exchange.GetAccountBalance(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get balances: %w", err)
	}

	// Update cache and notify subscribers
	for _, balance := range balances {
		// Cache the balance
		if err := m.cache.Set(ctx, fmt.Sprintf("balance:%s", balance.Asset), balance, time.Minute); err != nil {
			log.Printf("Failed to cache balance: %v", err)
		}

		// Update local cache
		m.updateBalance(balance)
	}

	return balances, nil
}

// GetBalanceHistory returns historical balance data
func (m *Manager) GetBalanceHistory(ctx context.Context, asset string, startTime, endTime time.Time) ([]*exch.AccountBalance, error) {
	query := `
		SELECT asset, total, available, locked, cross_margin,
			   isolated_margin, unrealized_pnl, realized_pnl, updated_at
		FROM account_balances
		WHERE asset = $1 AND updated_at BETWEEN $2 AND $3
		ORDER BY updated_at DESC
	`

	rows, err := m.db.QueryContext(ctx, query, asset, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query balance history: %w", err)
	}
	defer rows.Close()

	var history []*exch.AccountBalance
	for rows.Next() {
		var balance exch.AccountBalance
		if err := rows.Scan(
			&balance.Asset,
			&balance.Total,
			&balance.Available,
			&balance.Locked,
			&balance.CrossMargin,
			&balance.IsolatedMargin,
			&balance.UnrealizedPnL,
			&balance.RealizedPnL,
			&balance.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan balance: %w", err)
		}
		history = append(history, &balance)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating balances: %w", err)
	}

	return history, nil
}

// storeBalance stores a balance in the database
func (m *Manager) storeBalance(balance *exch.AccountBalance) error {
	query := `
		INSERT INTO account_balances (
			asset, total, available, locked, cross_margin,
			isolated_margin, unrealized_pnl, realized_pnl, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
	`

	_, err := m.db.Exec(query,
		balance.Asset,
		balance.Total,
		balance.Available,
		balance.Locked,
		balance.CrossMargin,
		balance.IsolatedMargin,
		balance.UnrealizedPnL,
		balance.RealizedPnL,
		balance.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to store balance: %w", err)
	}

	return nil
}

// updateBalance updates the local balance cache and notifies subscribers
func (m *Manager) updateBalance(balance *exch.AccountBalance) {
	m.mu.Lock()
	m.balances[balance.Asset] = balance
	m.mu.Unlock()

	// Notify subscribers
	m.notifySubscribers(balance)
}

// notifySubscribers notifies all subscribers of a balance update
func (m *Manager) notifySubscribers(balance *exch.AccountBalance) {
	m.mu.RLock()
	subs := m.subscribers[balance.Asset]
	m.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- balance:
		default:
			// Channel is full, skip
		}
	}
}

// monitorBalances periodically updates account balances
func (m *Manager) monitorBalances() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		balances, err := m.GetAllBalances(ctx)
		if err != nil {
			log.Printf("Failed to update balances: %v", err)
			continue
		}

		// Store balances in database
		for _, balance := range balances {
			if err := m.storeBalance(balance); err != nil {
				log.Printf("Failed to store balance: %v", err)
			}
		}
	}
}
