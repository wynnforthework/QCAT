package backtest

import (
	"fmt"
	"math"

	"qcat/internal/exchange"
	"qcat/internal/market/funding"
)

// PositionManager manages positions and margin
type PositionManager struct {
	positions    map[string]*exchange.Position
	balance      float64
	marginMode   exchange.MarginType
	leverage     float64
	marginRatios map[string]float64 // symbol -> initial margin ratio
}

// NewPositionManager creates a new position manager
func NewPositionManager(initialCapital float64, marginMode exchange.MarginType, leverage float64) *PositionManager {
	return &PositionManager{
		positions:    make(map[string]*exchange.Position),
		balance:      initialCapital,
		marginMode:   marginMode,
		leverage:     leverage,
		marginRatios: make(map[string]float64),
	}
}

// OpenPosition opens a new position
func (m *PositionManager) OpenPosition(trade *exchange.Trade) error {
	position, exists := m.positions[trade.Symbol]
	if !exists {
		position = &exchange.Position{
			Symbol:     trade.Symbol,
			MarginType: m.marginMode,
			Leverage:   m.leverage,
		}
		m.positions[trade.Symbol] = position
	}

	// 计算所需保证金
	requiredMargin := m.calculateRequiredMargin(trade)
	if m.getAvailableMargin() < requiredMargin {
		return fmt.Errorf("insufficient margin: required %.2f, available %.2f", requiredMargin, m.getAvailableMargin())
	}

	// 更新持仓
	if trade.Side == exchange.OrderSideBuy {
		position.Long += trade.Quantity
		position.EntryPrice = (position.EntryPrice*position.Long + trade.Price*trade.Quantity) / (position.Long + trade.Quantity)
	} else {
		position.Short += trade.Quantity
		position.EntryPrice = (position.EntryPrice*position.Short + trade.Price*trade.Quantity) / (position.Short + trade.Quantity)
	}

	// 扣除手续费
	m.balance -= trade.Fee

	return nil
}

// ClosePosition closes an existing position
func (m *PositionManager) ClosePosition(trade *exchange.Trade) error {
	position, exists := m.positions[trade.Symbol]
	if !exists {
		return fmt.Errorf("position not found: %s", trade.Symbol)
	}

	// 计算平仓数量
	var closeQuantity float64
	if trade.Side == exchange.OrderSideSell {
		closeQuantity = math.Min(position.Long, trade.Quantity)
		position.Long -= closeQuantity
	} else {
		closeQuantity = math.Min(position.Short, trade.Quantity)
		position.Short -= closeQuantity
	}

	// 计算平仓盈亏
	pnl := closeQuantity * (trade.Price - position.EntryPrice)
	if trade.Side == exchange.OrderSideBuy {
		pnl = -pnl
	}

	// 更新账户余额
	m.balance += pnl - trade.Fee

	// 如果完全平仓则删除持仓
	if position.Long == 0 && position.Short == 0 {
		delete(m.positions, trade.Symbol)
	}

	return nil
}

// GetEquity returns the current account equity
func (m *PositionManager) GetEquity() float64 {
	equity := m.balance

	// 加上未实现盈亏
	for _, pos := range m.positions {
		equity += pos.UnrealizedPnL
	}

	return equity
}

// ApplyFundingFee applies funding fee to positions
func (m *PositionManager) ApplyFundingFee(rate *funding.Rate) {
	position, exists := m.positions[rate.Symbol]
	if !exists {
		return
	}

	// 计算资金费用
	var fundingFee float64
	if position.Long > 0 {
		fundingFee = position.Long * position.EntryPrice * rate.Rate
	}
	if position.Short > 0 {
		fundingFee -= position.Short * position.EntryPrice * rate.Rate
	}

	// 更新账户余额
	m.balance -= fundingFee
}

// CheckLiquidation checks if any position should be liquidated
func (m *PositionManager) CheckLiquidation(prices map[string]float64) []string {
	var liquidated []string

	for symbol, pos := range m.positions {
		price, exists := prices[symbol]
		if !exists {
			continue
		}

		// 计算维持保证金率
		maintenanceMargin := m.calculateMaintenanceMargin(pos)
		currentMargin := m.calculateCurrentMargin(pos, price)

		if currentMargin < maintenanceMargin {
			liquidated = append(liquidated, symbol)
			// 强平处理
			m.liquidatePosition(pos, price)
		}
	}

	return liquidated
}

// Helper functions

func (m *PositionManager) calculateRequiredMargin(trade *exchange.Trade) float64 {
	ratio := m.marginRatios[trade.Symbol]
	if ratio == 0 {
		ratio = 0.01 // 默认1%
	}
	return trade.Price * trade.Quantity * ratio
}

func (m *PositionManager) getAvailableMargin() float64 {
	if m.marginMode == exchange.MarginTypeCross {
		return m.GetEquity()
	}
	// 对于逐仓模式，需要考虑每个symbol的独立保证金
	return m.balance
}

func (m *PositionManager) calculateMaintenanceMargin(pos *exchange.Position) float64 {
	// 简化版：维持保证金率为初始保证金率的50%
	ratio := m.marginRatios[pos.Symbol]
	if ratio == 0 {
		ratio = 0.01
	}
	return ratio * 0.5
}

func (m *PositionManager) calculateCurrentMargin(pos *exchange.Position, price float64) float64 {
	totalPositionValue := (pos.Long + pos.Short) * price
	if totalPositionValue == 0 {
		return 0
	}
	return m.GetEquity() / totalPositionValue
}

func (m *PositionManager) liquidatePosition(pos *exchange.Position, price float64) {
	// 计算强平损失
	loss := pos.Long*(price-pos.EntryPrice) - pos.Short*(price-pos.EntryPrice)
	m.balance += loss

	// 删除持仓
	delete(m.positions, pos.Symbol)
}
