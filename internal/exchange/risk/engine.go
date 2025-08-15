package risk

import (
	"context"
	"fmt"
	"sync"
	"time"

	"qcat/internal/exchange"
	"qcat/internal/exchange/position"
)

// RiskEngine manages risk control
type RiskEngine struct {
	exchange    exchange.Exchange
	posManager  *position.Manager
	limits      map[string]*RiskLimits
	monitors    map[string]*RiskMonitor
	subscribers map[string][]chan *RiskAlert
	mu          sync.RWMutex
}

// RiskLimits defines risk control limits
type RiskLimits struct {
	Symbol          string    // 交易对
	MaxPositionSize float64   // 最大仓位大小
	MaxLeverage     int       // 最大杠杆倍数
	MaxDrawdown     float64   // 最大回撤限制
	CircuitBreaker  float64   // 熔断阈值
	StopLoss        float64   // 止损阈值
	TakeProfit      float64   // 止盈阈值
	TrailingStop    float64   // 移动止损阈值
	UpdatedAt       time.Time // 更新时间
}

// RiskMonitor monitors risk metrics
type RiskMonitor struct {
	Symbol       string
	HighPrice    float64   // 最高价
	LowPrice     float64   // 最低价
	EntryPrice   float64   // 开仓价
	CurrentPrice float64   // 当前价
	PnL          float64   // 未实现盈亏
	DrawdownPct  float64   // 回撤百分比
	LastUpdate   time.Time // 最后更新时间
}

// RiskAlert represents a risk alert
type RiskAlert struct {
	Symbol    string
	Type      RiskAlertType
	Message   string
	Threshold float64
	Current   float64
	CreatedAt time.Time
}

// RiskAlertType defines the type of risk alert
type RiskAlertType int

const (
	RiskAlertDrawdown RiskAlertType = iota
	RiskAlertCircuitBreaker
	RiskAlertStopLoss
	RiskAlertTakeProfit
	RiskAlertPositionLimit
	RiskAlertLeverageLimit
)

// NewRiskEngine creates a new risk engine
func NewRiskEngine(ex exchange.Exchange, pm *position.Manager) *RiskEngine {
	return &RiskEngine{
		exchange:    ex,
		posManager:  pm,
		limits:      make(map[string]*RiskLimits),
		monitors:    make(map[string]*RiskMonitor),
		subscribers: make(map[string][]chan *RiskAlert),
	}
}

// SetRiskLimits sets risk limits for a symbol
func (e *RiskEngine) SetRiskLimits(symbol string, limits *RiskLimits) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if limits.MaxPositionSize <= 0 {
		return fmt.Errorf("invalid max position size")
	}
	if limits.MaxLeverage <= 0 {
		return fmt.Errorf("invalid max leverage")
	}
	if limits.MaxDrawdown <= 0 || limits.MaxDrawdown >= 1 {
		return fmt.Errorf("invalid max drawdown")
	}

	limits.UpdatedAt = time.Now()
	e.limits[symbol] = limits

	// Initialize monitor
	e.monitors[symbol] = &RiskMonitor{
		Symbol:     symbol,
		LastUpdate: time.Now(),
	}

	return nil
}

// GetRiskLimits returns risk limits for a symbol
func (e *RiskEngine) GetRiskLimits(symbol string) (*RiskLimits, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	limits, exists := e.limits[symbol]
	if !exists {
		return nil, fmt.Errorf("risk limits not found for symbol: %s", symbol)
	}
	return limits, nil
}

// Subscribe subscribes to risk alerts
func (e *RiskEngine) Subscribe(symbol string) chan *RiskAlert {
	e.mu.Lock()
	defer e.mu.Unlock()

	ch := make(chan *RiskAlert, 100)
	e.subscribers[symbol] = append(e.subscribers[symbol], ch)
	return ch
}

// Unsubscribe removes a subscription
func (e *RiskEngine) Unsubscribe(symbol string, ch chan *RiskAlert) {
	e.mu.Lock()
	defer e.mu.Unlock()

	subs := e.subscribers[symbol]
	for i, sub := range subs {
		if sub == ch {
			e.subscribers[symbol] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}
}

// CheckRiskLimits checks if an order would violate risk limits
func (e *RiskEngine) CheckRiskLimits(ctx context.Context, req *exchange.OrderRequest) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Get risk limits
	limits, exists := e.limits[req.Symbol]
	if !exists {
		return fmt.Errorf("risk limits not found for symbol: %s", req.Symbol)
	}

	// Get current position
	pos, err := e.posManager.GetPosition(ctx, req.Symbol)
	if err != nil && err != position.ErrPositionNotFound {
		return fmt.Errorf("failed to get position: %w", err)
	}

	// Check position size limit
	newSize := req.Quantity
	if pos != nil {
		if req.Side == exchange.OrderSideBuy {
			newSize += pos.Quantity
		} else {
			newSize = pos.Quantity - req.Quantity
		}
	}
	if newSize > limits.MaxPositionSize {
		return fmt.Errorf("position size limit exceeded: %f > %f", newSize, limits.MaxPositionSize)
	}

	// Check leverage limit
	if pos != nil && pos.Leverage > limits.MaxLeverage {
		return fmt.Errorf("leverage limit exceeded: %d > %d", pos.Leverage, limits.MaxLeverage)
	}

	return nil
}

// UpdateRiskMetrics updates risk metrics with new market data
func (e *RiskEngine) UpdateRiskMetrics(symbol string, price float64) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	monitor, exists := e.monitors[symbol]
	if !exists {
		return fmt.Errorf("risk monitor not found for symbol: %s", symbol)
	}

	// Update price metrics
	monitor.CurrentPrice = price
	if price > monitor.HighPrice {
		monitor.HighPrice = price
	}
	if monitor.LowPrice == 0 || price < monitor.LowPrice {
		monitor.LowPrice = price
	}

	// Calculate drawdown
	if monitor.HighPrice > 0 {
		monitor.DrawdownPct = (monitor.HighPrice - price) / monitor.HighPrice
	}

	// Check risk limits
	limits := e.limits[symbol]
	if limits != nil {
		// Check circuit breaker
		if monitor.DrawdownPct >= limits.CircuitBreaker {
			e.notifySubscribers(symbol, &RiskAlert{
				Symbol:    symbol,
				Type:      RiskAlertCircuitBreaker,
				Message:   fmt.Sprintf("Circuit breaker triggered: drawdown %.2f%% >= %.2f%%", monitor.DrawdownPct*100, limits.CircuitBreaker*100),
				Threshold: limits.CircuitBreaker,
				Current:   monitor.DrawdownPct,
				CreatedAt: time.Now(),
			})
		}

		// Check stop loss
		if monitor.EntryPrice > 0 {
			pnlPct := (price - monitor.EntryPrice) / monitor.EntryPrice
			if pnlPct <= -limits.StopLoss {
				e.notifySubscribers(symbol, &RiskAlert{
					Symbol:    symbol,
					Type:      RiskAlertStopLoss,
					Message:   fmt.Sprintf("Stop loss triggered: PnL %.2f%% <= %.2f%%", pnlPct*100, -limits.StopLoss*100),
					Threshold: -limits.StopLoss,
					Current:   pnlPct,
					CreatedAt: time.Now(),
				})
			}
		}
	}

	monitor.LastUpdate = time.Now()
	return nil
}

// notifySubscribers notifies all subscribers of a risk alert
func (e *RiskEngine) notifySubscribers(symbol string, alert *RiskAlert) {
	subs := e.subscribers[symbol]
	for _, ch := range subs {
		select {
		case ch <- alert:
		default:
			// Channel is full, skip
		}
	}
}
