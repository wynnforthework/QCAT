package backtest

import (
	"math"
	"time"

	"qcat/internal/exchange"
	"qcat/internal/market/orderbook"
)

// OrderManager manages order matching and execution
type OrderManager struct {
	orders        map[string]*exchange.Order
	slippageModel SlippageModel
	feeModel      FeeModel
	latencyModel  LatencyModel
}

// NewOrderManager creates a new order manager
func NewOrderManager(sm SlippageModel, fm FeeModel, lm LatencyModel) *OrderManager {
	return &OrderManager{
		orders:        make(map[string]*exchange.Order),
		slippageModel: sm,
		feeModel:      fm,
		latencyModel:  lm,
	}
}

// PlaceOrder places a new order
func (m *OrderManager) PlaceOrder(order *exchange.Order) {
	// 应用延迟模型
	latency := m.latencyModel.GetLatency()
	order.CreatedAt = order.CreatedAt.Add(latency)

	m.orders[order.ID] = order
}

// CancelOrder cancels an existing order
func (m *OrderManager) CancelOrder(orderID string) bool {
	if order, exists := m.orders[orderID]; exists {
		order.Status = string(exchange.OrderStatusCancelled)
		delete(m.orders, orderID)
		return true
	}
	return false
}

// Match matches orders against the orderbook
func (m *OrderManager) Match(ob *orderbook.Depth) []*exchange.Trade {
	var trades []*exchange.Trade

	// 处理限价单
	for _, order := range m.orders {
		if exchange.OrderType(order.Type) != exchange.OrderTypeLimit {
			continue
		}

		switch exchange.OrderSide(order.Side) {
		case exchange.OrderSideBuy:
			// 检查卖单簿
			for _, ask := range ob.Asks {
				if ask.Price > order.Price {
					break
				}
				if trade := m.matchOrder(order, ask.Price, math.Min(order.Quantity, ask.Quantity)); trade != nil {
					trades = append(trades, trade)
				}
			}
		case exchange.OrderSideSell:
			// 检查买单簿
			for _, bid := range ob.Bids {
				if bid.Price < order.Price {
					break
				}
				if trade := m.matchOrder(order, bid.Price, math.Min(order.Quantity, bid.Quantity)); trade != nil {
					trades = append(trades, trade)
				}
			}
		}
	}

	// 处理市价单
	for _, order := range m.orders {
		if exchange.OrderType(order.Type) != exchange.OrderTypeMarket {
			continue
		}

		switch exchange.OrderSide(order.Side) {
		case exchange.OrderSideBuy:
			// 使用最优卖价
			if len(ob.Asks) > 0 {
				if trade := m.matchOrder(order, ob.Asks[0].Price, order.Quantity); trade != nil {
					trades = append(trades, trade)
				}
			}
		case exchange.OrderSideSell:
			// 使用最优买价
			if len(ob.Bids) > 0 {
				if trade := m.matchOrder(order, ob.Bids[0].Price, order.Quantity); trade != nil {
					trades = append(trades, trade)
				}
			}
		}
	}

	return trades
}

// matchOrder matches a single order and creates a trade
func (m *OrderManager) matchOrder(order *exchange.Order, price, quantity float64) *exchange.Trade {
	// 应用滑点模型
	executionPrice := price + m.slippageModel.CalculateSlippage(price, quantity, exchange.OrderSide(order.Side))

	// 计算手续费
	fee := m.feeModel.CalculateFee(executionPrice, quantity)

	trade := &exchange.Trade{
		ID:       generateTradeID(),
		Symbol:   order.Symbol,
		Side:     order.Side,
		Price:    executionPrice,
		Quantity: quantity,
		Fee:      fee,
		Time:     time.Now(),
	}

	// 更新订单状态
	order.FilledQty += quantity
	if order.FilledQty >= order.Quantity {
		order.Status = string(exchange.OrderStatusFilled)
		delete(m.orders, order.ID)
	} else {
		order.Status = string(exchange.OrderStatusPartiallyFilled)
	}

	return trade
}

// generateTradeID generates a unique trade ID
func generateTradeID() string {
	return time.Now().Format("20060102150405.000000")
}

// DefaultSlippageModel implements a basic slippage model
type DefaultSlippageModel struct {
	basisPoints float64 // 基点，例如10表示0.1%
}

func NewDefaultSlippageModel(basisPoints float64) *DefaultSlippageModel {
	return &DefaultSlippageModel{basisPoints: basisPoints}
}

func (m *DefaultSlippageModel) CalculateSlippage(price, quantity float64, side exchange.OrderSide) float64 {
	slippage := price * m.basisPoints / 10000
	if side == exchange.OrderSideBuy {
		return slippage
	}
	return -slippage
}

// DefaultFeeModel implements a basic fee model
type DefaultFeeModel struct {
	makerFeeRate float64
	takerFeeRate float64
}

func NewDefaultFeeModel(makerFeeRate, takerFeeRate float64) *DefaultFeeModel {
	return &DefaultFeeModel{
		makerFeeRate: makerFeeRate,
		takerFeeRate: takerFeeRate,
	}
}

func (m *DefaultFeeModel) CalculateFee(price, quantity float64) float64 {
	return price * quantity * m.takerFeeRate
}

// DefaultLatencyModel implements a basic latency model
type DefaultLatencyModel struct {
	meanLatency time.Duration
	stdDev      time.Duration
}

func NewDefaultLatencyModel(meanLatency, stdDev time.Duration) *DefaultLatencyModel {
	return &DefaultLatencyModel{
		meanLatency: meanLatency,
		stdDev:      stdDev,
	}
}

func (m *DefaultLatencyModel) GetLatency() time.Duration {
	// 简化版：仅返回平均延迟
	return m.meanLatency
}
