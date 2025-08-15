package backtest

import (
	"fmt"
	"math"

	exch "qcat/internal/exchange"
	"qcat/internal/market/funding"
)

// PositionManager manages positions and margin
type PositionManager struct {
	positions    map[string]*exch.Position
	balance      float64
	marginMode   exch.MarginType
	leverage     float64
	marginRatios map[string]float64 // symbol -> initial margin ratio
}

// NewPositionManager creates a new position manager
func NewPositionManager(initialCapital float64, marginMode exch.MarginType, leverage float64) *PositionManager {
	return &PositionManager{
		positions:    make(map[string]*exch.Position),
		balance:      initialCapital,
		marginMode:   marginMode,
		leverage:     leverage,
		marginRatios: make(map[string]float64),
	}
}

// OpenPosition opens a new position
func (m *PositionManager) OpenPosition(trade *exch.Trade) error {
	position, exists := m.positions[trade.Symbol]
	if !exists {
		position = &exch.Position{
			Symbol:     trade.Symbol,
			MarginType: string(m.marginMode), // 显式转换为 string
			Leverage:   int(m.leverage),      // 显式转换为 int
		}
		m.positions[trade.Symbol] = position
	}

	// 计算所需保证金
	requiredMargin := m.calculateRequiredMargin(trade)
	if m.getAvailableMargin() < requiredMargin {
		return fmt.Errorf("insufficient margin: required %.2f, available %.2f", requiredMargin, m.getAvailableMargin())
	}

	// 更新持仓
	if string(trade.Side) == string(exch.OrderSideBuy) { // 显式转换为 string 进行比较
		// TODO: 待确认 - Position 结构体中没有 Long 字段，暂时使用 Size 字段
		position.Size += trade.Quantity
		position.EntryPrice = (position.EntryPrice*position.Size + trade.Price*trade.Quantity) / (position.Size + trade.Quantity)
	} else {
		// TODO: 待确认 - Position 结构体中没有 Short 字段，暂时使用 Size 字段
		position.Size += trade.Quantity
		position.EntryPrice = (position.EntryPrice*position.Size + trade.Price*trade.Quantity) / (position.Size + trade.Quantity)
	}

	// 扣除手续费
	m.balance -= trade.Fee

	return nil
}

// ClosePosition closes an existing position
func (m *PositionManager) ClosePosition(trade *exch.Trade) error {
	position, exists := m.positions[trade.Symbol]
	if !exists {
		return fmt.Errorf("position not found: %s", trade.Symbol)
	}

	// 计算平仓数量
	var closeQuantity float64
	if string(trade.Side) == string(exch.OrderSideSell) { // 显式转换为 string 进行比较
		// TODO: 待确认 - Position 结构体中没有 Long 字段，暂时使用 Size 字段
		closeQuantity = math.Min(position.Size, trade.Quantity)
		position.Size -= closeQuantity
	} else {
		// TODO: 待确认 - Position 结构体中没有 Short 字段，暂时使用 Size 字段
		closeQuantity = math.Min(position.Size, trade.Quantity)
		position.Size -= closeQuantity
	}

	// 计算平仓盈亏
	pnl := closeQuantity * (trade.Price - position.EntryPrice)
	if string(trade.Side) == string(exch.OrderSideBuy) { // 显式转换为 string 进行比较
		pnl = -pnl
	}

	// 更新账户余额
	m.balance += pnl - trade.Fee

	// 如果完全平仓则删除持仓
	if position.Size == 0 {
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
	// TODO: 待确认 - Position 结构体中没有 Long 和 Short 字段，暂时使用 Size 字段
	if position.Size > 0 {
		fundingFee = position.Size * position.EntryPrice * rate.Rate
	}

	// 更新账户余额
	m.balance -= fundingFee
}

// GetPosition returns a position for a symbol
func (m *PositionManager) GetPosition(symbol string) (*exch.Position, bool) {
	position, exists := m.positions[symbol]
	return position, exists
}

// GetAllPositions returns all positions
func (m *PositionManager) GetAllPositions() map[string]*exch.Position {
	return m.positions
}

// GetBalance returns the current balance
func (m *PositionManager) GetBalance() float64 {
	return m.balance
}

// SetBalance sets the balance
func (m *PositionManager) SetBalance(balance float64) {
	m.balance = balance
}

// calculateRequiredMargin calculates required margin for a trade
func (m *PositionManager) calculateRequiredMargin(trade *exch.Trade) float64 {
	// 简单计算：交易价值 / 杠杆倍数
	tradeValue := trade.Price * trade.Quantity
	return tradeValue / m.leverage
}

// getAvailableMargin returns available margin
func (m *PositionManager) getAvailableMargin() float64 {
	// 简单计算：余额 - 已用保证金
	usedMargin := 0.0
	for _, pos := range m.positions {
		usedMargin += pos.Size * pos.EntryPrice / m.leverage
	}
	return m.balance - usedMargin
}

// UpdatePosition updates position with current market price
func (m *PositionManager) UpdatePosition(symbol string, price float64) {
	position, exists := m.positions[symbol]
	if !exists {
		return
	}

	// 更新标记价格
	position.MarkPrice = price

	// 计算未实现盈亏
	// TODO: 待确认 - Position 结构体中没有 Long 和 Short 字段，暂时使用 Size 字段
	if position.Size > 0 {
		position.UnrealizedPnL = position.Size * (price - position.EntryPrice)
	}
}

// GetMarginRatio returns margin ratio for a symbol
func (m *PositionManager) GetMarginRatio(symbol string) float64 {
	position, exists := m.positions[symbol]
	if !exists {
		return 0
	}

	// 计算保证金率：权益 / 保证金
	equity := m.GetEquity()
	margin := position.Size * position.EntryPrice / m.leverage
	if margin == 0 {
		return 0
	}

	return equity / margin
}

// SetMarginRatio sets margin ratio for a symbol
func (m *PositionManager) SetMarginRatio(symbol string, ratio float64) {
	m.marginRatios[symbol] = ratio
}
