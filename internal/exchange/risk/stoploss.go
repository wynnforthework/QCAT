package risk

import (
	"context"
	"fmt"
	"sync"
	"time"

	exch "qcat/internal/exchange"
	"qcat/internal/exchange/order"
)

// StopManager manages stop loss and take profit orders
type StopManager struct {
	exchange exch.Exchange
	orderMgr *order.Manager
	stops    map[string]*StopOrder
	mu       sync.RWMutex
}

// StopOrder represents a stop loss or take profit order
type StopOrder struct {
	Symbol        string
	Side          exch.OrderSide
	StopPrice     float64
	Quantity      float64
	Type          StopType
	TrailingDelta float64   // 移动止损回调比例
	HighPrice     float64   // 移动止损最高价
	LowPrice      float64   // 移动止损最低价
	ATR           float64   // ATR值
	TimeLimit     time.Time // 时间限制
	CreatedAt     time.Time
}

// StopType defines the type of stop order
type StopType int

const (
	StopTypeFixed      StopType = iota // 固定止损
	StopTypeTrailing                   // 移动止损
	StopTypeATR                        // ATR止损
	StopTypeTime                       // 时间止损
	StopTypeChandelier                 // Chandelier止损
	StopTypeParabolic                  // Parabolic止损
)

// NewStopManager creates a new stop manager
func NewStopManager(ex exch.Exchange, om *order.Manager) *StopManager {
	return &StopManager{
		exchange: ex,
		orderMgr: om,
		stops:    make(map[string]*StopOrder),
	}
}

// SetStopLoss sets a stop loss order
func (m *StopManager) SetStopLoss(symbol string, stopPrice float64, quantity float64, stopType StopType) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if stopPrice <= 0 {
		return fmt.Errorf("invalid stop price")
	}
	if quantity <= 0 {
		return fmt.Errorf("invalid quantity")
	}

	m.stops[symbol] = &StopOrder{
		Symbol:    symbol,
		Side:      exch.OrderSideSell,
		StopPrice: stopPrice,
		Quantity:  quantity,
		Type:      stopType,
		CreatedAt: time.Now(),
	}

	return nil
}

// SetTakeProfit sets a take profit order
func (m *StopManager) SetTakeProfit(symbol string, profitPrice float64, quantity float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if profitPrice <= 0 {
		return fmt.Errorf("invalid profit price")
	}
	if quantity <= 0 {
		return fmt.Errorf("invalid quantity")
	}

	key := fmt.Sprintf("%s_tp", symbol)
	m.stops[key] = &StopOrder{
		Symbol:    symbol,
		Side:      exch.OrderSideSell,
		StopPrice: profitPrice,
		Quantity:  quantity,
		Type:      StopTypeFixed,
		CreatedAt: time.Now(),
	}

	return nil
}

// SetTrailingStop sets a trailing stop loss
func (m *StopManager) SetTrailingStop(symbol string, trailingDelta float64, quantity float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if trailingDelta <= 0 || trailingDelta >= 1 {
		return fmt.Errorf("invalid trailing delta")
	}
	if quantity <= 0 {
		return fmt.Errorf("invalid quantity")
	}

	m.stops[symbol] = &StopOrder{
		Symbol:        symbol,
		Side:          exch.OrderSideSell,
		Quantity:      quantity,
		Type:          StopTypeTrailing,
		TrailingDelta: trailingDelta,
		CreatedAt:     time.Now(),
	}

	return nil
}

// SetATRStop sets an ATR-based stop loss
func (m *StopManager) SetATRStop(symbol string, multiplier float64, atr float64, quantity float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if multiplier <= 0 {
		return fmt.Errorf("invalid multiplier")
	}
	if atr <= 0 {
		return fmt.Errorf("invalid ATR")
	}
	if quantity <= 0 {
		return fmt.Errorf("invalid quantity")
	}

	pos, err := m.exchange.GetPosition(context.Background(), symbol)
	if err != nil {
		return fmt.Errorf("failed to get position: %w", err)
	}

	stopPrice := pos.EntryPrice - (multiplier * atr)
	if string(pos.Side) == string(exch.PositionSideShort) { // 显式转换为 string 进行比较
		stopPrice = pos.EntryPrice + (multiplier * atr)
	}

	m.stops[symbol] = &StopOrder{
		Symbol:    symbol,
		Side:      exch.OrderSideSell,
		StopPrice: stopPrice,
		Quantity:  quantity,
		Type:      StopTypeATR,
		ATR:       atr,
		CreatedAt: time.Now(),
	}

	return nil
}

// SetTimeStop sets a time-based stop loss
func (m *StopManager) SetTimeStop(symbol string, duration time.Duration, quantity float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if duration <= 0 {
		return fmt.Errorf("invalid duration")
	}
	if quantity <= 0 {
		return fmt.Errorf("invalid quantity")
	}

	m.stops[symbol] = &StopOrder{
		Symbol:    symbol,
		Side:      exch.OrderSideSell,
		Quantity:  quantity,
		Type:      StopTypeTime,
		TimeLimit: time.Now().Add(duration),
		CreatedAt: time.Now(),
	}

	return nil
}

// CheckStops checks if any stops should be triggered
func (m *StopManager) CheckStops(ctx context.Context, symbol string, price float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	stop, exists := m.stops[symbol]
	if !exists {
		return nil
	}

	var triggered bool
	switch stop.Type {
	case StopTypeFixed:
		if stop.Side == exch.OrderSideSell {
			triggered = price <= stop.StopPrice
		} else {
			triggered = price >= stop.StopPrice
		}

	case StopTypeTrailing:
		if stop.HighPrice == 0 {
			stop.HighPrice = price
		} else if price > stop.HighPrice {
			stop.HighPrice = price
			stop.StopPrice = price * (1 - stop.TrailingDelta)
		}
		triggered = price <= stop.StopPrice

	case StopTypeATR:
		pos, err := m.exchange.GetPosition(ctx, symbol)
		if err != nil {
			return fmt.Errorf("failed to get position: %w", err)
		}
		if string(pos.Side) == string(exch.PositionSideLong) { // 显式转换为 string 进行比较
			triggered = price <= stop.StopPrice
		} else {
			triggered = price >= stop.StopPrice
		}

	case StopTypeTime:
		triggered = time.Now().After(stop.TimeLimit)

	case StopTypeChandelier:
		if stop.HighPrice == 0 {
			stop.HighPrice = price
		} else if price > stop.HighPrice {
			stop.HighPrice = price
		}
		atrStop := stop.HighPrice - (3 * stop.ATR)
		triggered = price <= atrStop

	case StopTypeParabolic:
		if stop.LowPrice == 0 {
			stop.LowPrice = price
		} else if price < stop.LowPrice {
			stop.LowPrice = price
		}
		parabolicStop := stop.LowPrice + (stop.ATR * 2)
		triggered = price >= parabolicStop
	}

	if triggered {
		// Create market order to execute stop
		order := &exch.OrderRequest{
			Symbol:   stop.Symbol,
			Side:     string(stop.Side),            // 显式转换为 string
			Type:     string(exch.OrderTypeMarket), // 显式转换为 string
			Quantity: stop.Quantity,
		}

		if _, err := m.exchange.PlaceOrder(ctx, order); err != nil {
			return fmt.Errorf("failed to place stop order: %w", err)
		}

		// Remove the stop order
		delete(m.stops, symbol)
	}

	return nil
}

// GetStop returns a stop order for a symbol
func (m *StopManager) GetStop(symbol string) (*StopOrder, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stop, exists := m.stops[symbol]
	if !exists {
		return nil, fmt.Errorf("stop order not found for symbol: %s", symbol)
	}

	return stop, nil
}

// RemoveStop removes a stop order
func (m *StopManager) RemoveStop(symbol string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.stops[symbol]; !exists {
		return fmt.Errorf("stop order not found for symbol: %s", symbol)
	}

	delete(m.stops, symbol)
	return nil
}

// GetAllStops returns all stop orders
func (m *StopManager) GetAllStops() map[string]*StopOrder {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*StopOrder)
	for symbol, stop := range m.stops {
		result[symbol] = stop
	}

	return result
}
