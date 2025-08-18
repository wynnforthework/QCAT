package risk

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/cache"
	"qcat/internal/config"
	exch "qcat/internal/exchange"
)

// Manager manages risk limits and controls
type Manager struct {
	db          *sql.DB
	cache       cache.Cacher
	exchange    exch.Exchange
	limits      map[string]*exch.RiskLimit
	subscribers map[string][]chan *exch.RiskLimit
	mu          sync.RWMutex
}

// NewManager creates a new risk manager
func NewManager(db *sql.DB, cache cache.Cacher, exchange exch.Exchange) *Manager {
	m := &Manager{
		db:          db,
		cache:       cache,
		exchange:    exchange,
		limits:      make(map[string]*exch.RiskLimit),
		subscribers: make(map[string][]chan *exch.RiskLimit),
	}

	// Start risk monitor
	go m.monitorRiskLimits()

	return m
}

// Subscribe subscribes to risk limit updates for a symbol
func (m *Manager) Subscribe(symbol string) chan *exch.RiskLimit {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan *exch.RiskLimit, 100)
	m.subscribers[symbol] = append(m.subscribers[symbol], ch)
	return ch
}

// Unsubscribe removes a subscription
func (m *Manager) Unsubscribe(symbol string, ch chan *exch.RiskLimit) {
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

// GetRiskLimits returns risk limits for a symbol
func (m *Manager) GetRiskLimits(ctx context.Context, symbol string) ([]*exch.RiskLimit, error) {
	// Check cache first
	var limits []*exch.RiskLimit
	err := m.cache.Get(ctx, fmt.Sprintf("risk_limits:%s", symbol), &limits)
	if err == nil {
		return limits, nil
	}

	// Get from exchange
	riskLimits, err := m.exchange.GetRiskLimits(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get risk limits: %w", err)
	}

	// Convert *RiskLimits to []*RiskLimit
	limits = []*exch.RiskLimit{
		{
			Symbol:           riskLimits.Symbol,
			MaxLeverage:      riskLimits.MaxLeverage,
			MaxPositionValue: riskLimits.MaxPositionValue,
			MaxOrderValue:    riskLimits.MaxOrderValue,
			MinOrderValue:    riskLimits.MinOrderValue,
			MaxOrderQty:      riskLimits.MaxOrderQty,
			MinOrderQty:      riskLimits.MinOrderQty,
		},
	}

	// Cache the limits
	if err := m.cache.Set(ctx, fmt.Sprintf("risk_limits:%s", symbol), limits, time.Hour); err != nil {
		log.Printf("Failed to cache risk limits: %v", err)
	}

	// Update local cache and notify subscribers
	for _, limit := range limits {
		m.updateLimit(limit)
	}

	return limits, nil
}

// SetRiskLimits sets risk limits for a symbol
func (m *Manager) SetRiskLimits(ctx context.Context, symbol string, limits []*exch.RiskLimit) error {
	// 尝试通过交易所接口设置风险限额
	for _, limit := range limits {
		// 转换为交易所接口需要的格式
		exchangeLimits := &exch.RiskLimits{
			Symbol:           limit.Symbol,
			MaxLeverage:      limit.MaxLeverage,
			MaxPositionValue: limit.MaxPositionValue,
			MaxOrderValue:    limit.MaxOrderValue,
			MinOrderValue:    limit.MinOrderValue,
			MaxOrderQty:      limit.MaxOrderQty,
			MinOrderQty:      limit.MinOrderQty,
		}

		if err := m.exchange.SetRiskLimits(ctx, symbol, exchangeLimits); err != nil {
			log.Printf("Failed to set risk limits via exchange API: %v, falling back to local management", err)
		}
	}

	// Store in database as backup
	for _, limit := range limits {
		if err := m.storeLimit(limit); err != nil {
			log.Printf("Failed to store risk limit: %v", err)
		}
	}

	// Update cache
	if err := m.cache.Set(ctx, fmt.Sprintf("risk_limits:%s", symbol), limits, time.Hour); err != nil {
		log.Printf("Failed to cache risk limits: %v", err)
	}

	// Update local cache and notify subscribers
	for _, limit := range limits {
		m.updateLimit(limit)
	}

	return nil
}

// CheckRiskLimits checks if an order would violate risk limits
func (m *Manager) CheckRiskLimits(ctx context.Context, order *exch.OrderRequest) error {
	// Get current position
	position, err := m.exchange.GetPosition(ctx, order.Symbol)
	if err != nil {
		return fmt.Errorf("failed to get position: %w", err)
	}

	// Get risk limits
	limits, err := m.GetRiskLimits(ctx, order.Symbol)
	if err != nil {
		return fmt.Errorf("failed to get risk limits: %w", err)
	}

	// Find applicable limit based on position size
	var limit *exch.RiskLimit
	for _, l := range limits {
		if position == nil || position.Quantity+order.Quantity <= l.MaxPositionValue { // 使用 MaxPositionValue 替代 MaxPositionSize
			limit = l
			break
		}
	}

	if limit == nil {
		return fmt.Errorf("no applicable risk limit found for position size")
	}

	// Check leverage
	if position != nil && position.Leverage > limit.MaxLeverage { // 使用 MaxLeverage 替代 Leverage
		return fmt.Errorf("leverage exceeds limit: %d > %d", position.Leverage, limit.MaxLeverage)
	}

	// Check position size
	if position != nil && position.Quantity+order.Quantity > limit.MaxPositionValue { // 使用 MaxPositionValue 替代 MaxPositionSize
		return fmt.Errorf("position size would exceed limit: %.8f > %.8f",
			position.Quantity+order.Quantity, limit.MaxPositionValue)
	}

	// Check maintenance margin
	if position != nil {
		// 从配置获取维持保证金率
		maintenanceMarginRate := 0.1 // 10% 维持保证金率 (默认值)
		if algorithmConfig := config.GetAlgorithmConfig(); algorithmConfig != nil {
			maintenanceMarginRate = algorithmConfig.RiskMgmt.Margin.MaintenanceMarginRatio
		}
		maintenanceMargin := (position.Quantity + order.Quantity) * position.MarkPrice * maintenanceMarginRate
		if maintenanceMargin > position.UnrealizedPnL {
			return fmt.Errorf("maintenance margin would be insufficient: %.8f > %.8f",
				maintenanceMargin, position.UnrealizedPnL)
		}
	}

	return nil
}

// storeLimit stores a risk limit in the database
func (m *Manager) storeLimit(limit *exch.RiskLimit) error {
	query := `
		INSERT INTO risk_limits (
			symbol, leverage, max_position_size, maintenance_margin,
			initial_margin, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)
	`

	_, err := m.db.Exec(query,
		limit.Symbol,
		limit.MaxLeverage,      // 使用 MaxLeverage 替代 Leverage
		limit.MaxPositionValue, // 使用 MaxPositionValue 替代 MaxPositionSize
		getMaintenanceMarginRate(), // 从配置获取维持保证金率
		getInitialMarginRate(),     // 从配置获取初始保证金率
		time.Now(),             // 更新时间
	)

	if err != nil {
		return fmt.Errorf("failed to store risk limit: %w", err)
	}

	return nil
}

// updateLimit updates the local risk limit cache and notifies subscribers
func (m *Manager) updateLimit(limit *exch.RiskLimit) {
	m.mu.Lock()
	m.limits[limit.Symbol] = limit
	m.mu.Unlock()

	// Notify subscribers
	m.notifySubscribers(limit)
}

// notifySubscribers notifies all subscribers of a risk limit update
func (m *Manager) notifySubscribers(limit *exch.RiskLimit) {
	m.mu.RLock()
	subs := m.subscribers[limit.Symbol]
	m.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- limit:
		default:
			// Channel is full, skip
		}
	}
}

// monitorRiskLimits periodically updates risk limits
func (m *Manager) monitorRiskLimits() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.RLock()
		symbols := make([]string, 0, len(m.limits))
		for symbol := range m.limits {
			symbols = append(symbols, symbol)
		}
		m.mu.RUnlock()

		ctx := context.Background()
		for _, symbol := range symbols {
			limits, err := m.GetRiskLimits(ctx, symbol)
			if err != nil {
				log.Printf("Failed to update risk limits for %s: %v", symbol, err)
				continue
			}

			// Store limits in database
			for _, limit := range limits {
				if err := m.storeLimit(limit); err != nil {
					log.Printf("Failed to store risk limit: %v", err)
				}
			}
		}
	}
}

// getMaintenanceMarginRate returns the configured maintenance margin rate
func getMaintenanceMarginRate() float64 {
	defaultRate := 0.1 // 10% default
	if algorithmConfig := config.GetAlgorithmConfig(); algorithmConfig != nil {
		return algorithmConfig.RiskMgmt.Margin.MaintenanceMarginRatio
	}
	return defaultRate
}

// getInitialMarginRate returns the configured initial margin rate
func getInitialMarginRate() float64 {
	defaultRate := 0.2 // 20% default
	if algorithmConfig := config.GetAlgorithmConfig(); algorithmConfig != nil {
		return algorithmConfig.RiskMgmt.Margin.MaxMarginRatio
	}
	return defaultRate
}
