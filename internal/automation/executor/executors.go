package executor

import (
	"context"
	"fmt"
	"log"

	"qcat/internal/config"
	"qcat/internal/database"
	"qcat/internal/exchange"
	"qcat/internal/exchange/account"
)

// PositionExecutor 仓位执行器
type PositionExecutor struct {
	config         *config.Config
	db             *database.DB
	exchange       exchange.Exchange
	accountManager *account.Manager
}

// NewPositionExecutor 创建仓位执行器
func NewPositionExecutor(
	cfg *config.Config,
	db *database.DB,
	exchange exchange.Exchange,
	accountManager *account.Manager,
) *PositionExecutor {
	return &PositionExecutor{
		config:         cfg,
		db:             db,
		exchange:       exchange,
		accountManager: accountManager,
	}
}

// HandleAction 处理仓位动作
func (pe *PositionExecutor) HandleAction(ctx context.Context, action *ExecutionAction) error {
	switch action.Action {
	case "adjust_position":
		return pe.adjustPosition(ctx, action)
	case "close_position":
		return pe.closePosition(ctx, action)
	case "reduce_position":
		return pe.reducePosition(ctx, action)
	default:
		return fmt.Errorf("unknown position action: %s", action.Action)
	}
}

// adjustPosition 调整仓位
func (pe *PositionExecutor) adjustPosition(ctx context.Context, action *ExecutionAction) error {
	symbol := action.Symbol
	targetSize, ok := action.Parameters["target_size"].(float64)
	if !ok {
		return fmt.Errorf("invalid target_size parameter")
	}

	log.Printf("Adjusting position for %s to size: %.4f", symbol, targetSize)

	// 1. 获取当前仓位
	currentPosition, err := pe.exchange.GetPosition(ctx, symbol)
	if err != nil {
		return fmt.Errorf("failed to get current position: %w", err)
	}

	currentSize := 0.0
	if currentPosition != nil {
		currentSize = currentPosition.Quantity
	}

	// 2. 计算需要调整的数量
	adjustmentSize := targetSize - currentSize
	if adjustmentSize == 0 {
		log.Printf("Position for %s already at target size", symbol)
		return nil
	}

	// 3. 确定订单方向
	var side string
	if adjustmentSize > 0 {
		side = "BUY"
	} else {
		side = "SELL"
		adjustmentSize = -adjustmentSize // 转为正数
	}

	// 4. 获取当前价格
	price, err := pe.exchange.GetSymbolPrice(ctx, symbol)
	if err != nil {
		return fmt.Errorf("failed to get symbol price: %w", err)
	}

	// 5. 生成并执行订单
	orderReq := &exchange.OrderRequest{
		Symbol:   symbol,
		Side:     side,
		Type:     "MARKET",
		Quantity: adjustmentSize,
		Price:    price,
	}

	orderResp, err := pe.exchange.PlaceOrder(ctx, orderReq)
	if err != nil {
		return fmt.Errorf("failed to place adjustment order: %w", err)
	}

	log.Printf("Position adjustment order placed: %s, OrderID: %s", symbol, orderResp.OrderID)
	return nil
}

// closePosition 平仓
func (pe *PositionExecutor) closePosition(ctx context.Context, action *ExecutionAction) error {
	symbol := action.Symbol
	log.Printf("Closing position for %s", symbol)

	// 1. 获取当前仓位
	currentPosition, err := pe.exchange.GetPosition(ctx, symbol)
	if err != nil {
		return fmt.Errorf("failed to get current position: %w", err)
	}

	if currentPosition == nil || currentPosition.Quantity == 0 {
		log.Printf("No position to close for %s", symbol)
		return nil
	}

	// 2. 确定平仓方向（与当前仓位相反）
	var side string
	closeQuantity := currentPosition.Quantity
	if closeQuantity > 0 {
		side = "SELL"
	} else {
		side = "BUY"
		closeQuantity = -closeQuantity // 转为正数
	}

	// 3. 获取当前价格
	price, err := pe.exchange.GetSymbolPrice(ctx, symbol)
	if err != nil {
		return fmt.Errorf("failed to get symbol price: %w", err)
	}

	// 4. 执行平仓订单
	orderReq := &exchange.OrderRequest{
		Symbol:   symbol,
		Side:     side,
		Type:     "MARKET",
		Quantity: closeQuantity,
		Price:    price,
	}

	orderResp, err := pe.exchange.PlaceOrder(ctx, orderReq)
	if err != nil {
		return fmt.Errorf("failed to place close order: %w", err)
	}

	log.Printf("Position closed: %s, OrderID: %s", symbol, orderResp.OrderID)
	return nil
}

// reducePosition 减仓
func (pe *PositionExecutor) reducePosition(ctx context.Context, action *ExecutionAction) error {
	symbol := action.Symbol
	reductionRatio, ok := action.Parameters["reduction_ratio"].(float64)
	if !ok {
		return fmt.Errorf("invalid reduction_ratio parameter")
	}

	log.Printf("Reducing position for %s by %.2f%%", symbol, reductionRatio*100)

	// 1. 获取当前仓位
	currentPosition, err := pe.exchange.GetPosition(ctx, symbol)
	if err != nil {
		return fmt.Errorf("failed to get current position: %w", err)
	}

	if currentPosition == nil || currentPosition.Quantity == 0 {
		log.Printf("No position to reduce for %s", symbol)
		return nil
	}

	// 2. 计算减仓数量
	currentSize := currentPosition.Quantity
	reductionSize := currentSize * reductionRatio
	if reductionSize == 0 {
		log.Printf("Reduction size is zero for %s", symbol)
		return nil
	}

	// 3. 确定减仓方向（与当前仓位相反）
	var side string
	if currentSize > 0 {
		side = "SELL"
	} else {
		side = "BUY"
		reductionSize = -reductionSize // 转为正数
	}

	// 4. 获取当前价格
	price, err := pe.exchange.GetSymbolPrice(ctx, symbol)
	if err != nil {
		return fmt.Errorf("failed to get symbol price: %w", err)
	}

	// 5. 执行减仓订单
	orderReq := &exchange.OrderRequest{
		Symbol:   symbol,
		Side:     side,
		Type:     "MARKET",
		Quantity: reductionSize,
		Price:    price,
	}

	orderResp, err := pe.exchange.PlaceOrder(ctx, orderReq)
	if err != nil {
		return fmt.Errorf("failed to place reduction order: %w", err)
	}

	log.Printf("Position reduced: %s by %.2f%%, OrderID: %s", symbol, reductionRatio*100, orderResp.OrderID)
	return nil
}

// RiskExecutor 风险执行器
type RiskExecutor struct {
	config         *config.Config
	db             *database.DB
	exchange       exchange.Exchange
	accountManager *account.Manager
}

// NewRiskExecutor 创建风险执行器
func NewRiskExecutor(
	cfg *config.Config,
	db *database.DB,
	exchange exchange.Exchange,
	accountManager *account.Manager,
) *RiskExecutor {
	return &RiskExecutor{
		config:         cfg,
		db:             db,
		exchange:       exchange,
		accountManager: accountManager,
	}
}

// HandleAction 处理风险动作
func (re *RiskExecutor) HandleAction(ctx context.Context, action *ExecutionAction) error {
	switch action.Action {
	case "emergency_stop":
		return re.emergencyStop(ctx, action)
	case "reduce_leverage":
		return re.reduceLeverage(ctx, action)
	case "hedge_position":
		return re.hedgePosition(ctx, action)
	case "circuit_breaker":
		return re.circuitBreaker(ctx, action)
	default:
		return fmt.Errorf("unknown risk action: %s", action.Action)
	}
}

// emergencyStop 紧急停止
func (re *RiskExecutor) emergencyStop(ctx context.Context, action *ExecutionAction) error {
	log.Printf("Executing emergency stop")

	// 1. 取消所有挂单
	openOrders, err := re.exchange.GetOpenOrders(ctx, "")
	if err != nil {
		log.Printf("Failed to get open orders during emergency stop: %v", err)
	} else {
		for _, order := range openOrders {
			cancelReq := &exchange.OrderCancelRequest{
				Symbol:  order.Symbol,
				OrderID: order.OrderID,
			}
			_, err := re.exchange.CancelOrder(ctx, cancelReq)
			if err != nil {
				log.Printf("Failed to cancel order %s: %v", order.OrderID, err)
			} else {
				log.Printf("Cancelled order: %s", order.OrderID)
			}
		}
	}

	// 2. 平掉所有仓位
	positions, err := re.exchange.GetPositions(ctx)
	if err != nil {
		log.Printf("Failed to get positions during emergency stop: %v", err)
	} else {
		for _, position := range positions {
			if position.Quantity == 0 {
				continue
			}

			// 确定平仓方向
			var side string
			closeQuantity := position.Quantity
			if closeQuantity > 0 {
				side = "SELL"
			} else {
				side = "BUY"
				closeQuantity = -closeQuantity
			}

			// 获取当前价格并平仓
			price, err := re.exchange.GetSymbolPrice(ctx, position.Symbol)
			if err != nil {
				log.Printf("Failed to get price for %s during emergency stop: %v", position.Symbol, err)
				continue
			}

			orderReq := &exchange.OrderRequest{
				Symbol:   position.Symbol,
				Side:     side,
				Type:     "MARKET",
				Quantity: closeQuantity,
				Price:    price,
			}

			orderResp, err := re.exchange.PlaceOrder(ctx, orderReq)
			if err != nil {
				log.Printf("Failed to close position %s during emergency stop: %v", position.Symbol, err)
			} else {
				log.Printf("Emergency close position: %s, OrderID: %s", position.Symbol, orderResp.OrderID)
			}
		}
	}

	// 3. 记录紧急停止事件
	log.Printf("Emergency stop completed")
	return nil
}

// reduceLeverage 降低杠杆
func (re *RiskExecutor) reduceLeverage(ctx context.Context, action *ExecutionAction) error {
	symbol := action.Symbol
	targetLeverage, ok := action.Parameters["target_leverage"].(float64)
	if !ok {
		return fmt.Errorf("invalid target_leverage parameter")
	}

	log.Printf("Reducing leverage for %s to %.1fx", symbol, targetLeverage)

	// TODO: 实现降杠杆逻辑
	return nil
}

// hedgePosition 对冲仓位
func (re *RiskExecutor) hedgePosition(ctx context.Context, action *ExecutionAction) error {
	symbol := action.Symbol
	hedgeRatio, ok := action.Parameters["hedge_ratio"].(float64)
	if !ok {
		return fmt.Errorf("invalid hedge_ratio parameter")
	}

	log.Printf("Hedging position for %s with ratio: %.2f", symbol, hedgeRatio)

	// TODO: 实现对冲逻辑
	return nil
}

// circuitBreaker 熔断器
func (re *RiskExecutor) circuitBreaker(ctx context.Context, action *ExecutionAction) error {
	log.Printf("Triggering circuit breaker")

	// TODO: 实现熔断器逻辑
	// 1. 暂停交易
	// 2. 评估风险
	// 3. 决定后续动作

	return nil
}

// OrderExecutor 订单执行器
type OrderExecutor struct {
	config         *config.Config
	db             *database.DB
	exchange       exchange.Exchange
	accountManager *account.Manager
}

// NewOrderExecutor 创建订单执行器
func NewOrderExecutor(
	cfg *config.Config,
	db *database.DB,
	exchange exchange.Exchange,
	accountManager *account.Manager,
) *OrderExecutor {
	return &OrderExecutor{
		config:         cfg,
		db:             db,
		exchange:       exchange,
		accountManager: accountManager,
	}
}

// HandleAction 处理订单动作
func (oe *OrderExecutor) HandleAction(ctx context.Context, action *ExecutionAction) error {
	switch action.Action {
	case "place_order":
		return oe.placeOrder(ctx, action)
	case "cancel_order":
		return oe.cancelOrder(ctx, action)
	case "modify_order":
		return oe.modifyOrder(ctx, action)
	case "stop_loss":
		return oe.placeStopLoss(ctx, action)
	case "take_profit":
		return oe.placeTakeProfit(ctx, action)
	default:
		return fmt.Errorf("unknown order action: %s", action.Action)
	}
}

// placeOrder 下单
func (oe *OrderExecutor) placeOrder(ctx context.Context, action *ExecutionAction) error {
	symbol := action.Symbol
	side, _ := action.Parameters["side"].(string)
	quantity, _ := action.Parameters["quantity"].(float64)
	price, _ := action.Parameters["price"].(float64)
	orderType, _ := action.Parameters["type"].(string)

	// 设置默认值
	if orderType == "" {
		orderType = "MARKET"
	}
	if side == "" {
		return fmt.Errorf("order side is required")
	}
	if quantity <= 0 {
		return fmt.Errorf("order quantity must be positive")
	}

	log.Printf("Placing %s %s order for %s: %.4f @ %.4f", orderType, side, symbol, quantity, price)

	// 1. 验证订单参数
	if symbol == "" {
		return fmt.Errorf("symbol is required")
	}

	// 2. 检查账户余额
	balances, err := oe.exchange.GetAccountBalance(ctx)
	if err != nil {
		log.Printf("Warning: Failed to check account balance: %v", err)
		// 继续执行，让交易所来验证余额
	} else {
		// 简单的余额检查（这里可以添加更复杂的逻辑）
		if len(balances) == 0 {
			log.Printf("Warning: No account balance information available")
		}
	}

	// 3. 构建订单请求
	orderReq := &exchange.OrderRequest{
		Symbol:   symbol,
		Side:     side,
		Type:     orderType,
		Quantity: quantity,
		Price:    price,
	}

	// 如果是市价单，价格设为0
	if orderType == "MARKET" {
		orderReq.Price = 0
	}

	// 4. 调用交易所API
	orderResp, err := oe.exchange.PlaceOrder(ctx, orderReq)
	if err != nil {
		return fmt.Errorf("failed to place order: %w", err)
	}

	if !orderResp.Success {
		return fmt.Errorf("order rejected: %s", orderResp.Error)
	}

	// 5. 记录订单信息
	log.Printf("Order placed successfully: %s, OrderID: %s", symbol, orderResp.OrderID)
	return nil
}

// cancelOrder 撤单
func (oe *OrderExecutor) cancelOrder(ctx context.Context, action *ExecutionAction) error {
	orderID, ok := action.Parameters["order_id"].(string)
	if !ok {
		return fmt.Errorf("invalid order_id parameter")
	}

	log.Printf("Cancelling order: %s", orderID)

	// TODO: 实现撤单逻辑
	return nil
}

// modifyOrder 修改订单
func (oe *OrderExecutor) modifyOrder(ctx context.Context, action *ExecutionAction) error {
	orderID, ok := action.Parameters["order_id"].(string)
	if !ok {
		return fmt.Errorf("invalid order_id parameter")
	}

	log.Printf("Modifying order: %s", orderID)

	// TODO: 实现修改订单逻辑
	return nil
}

// placeStopLoss 设置止损
func (oe *OrderExecutor) placeStopLoss(ctx context.Context, action *ExecutionAction) error {
	symbol := action.Symbol
	stopPrice, ok := action.Parameters["stop_price"].(float64)
	if !ok {
		return fmt.Errorf("invalid stop_price parameter")
	}

	log.Printf("Placing stop loss for %s at price: %.4f", symbol, stopPrice)

	// 1. 获取当前仓位以确定止损方向
	position, err := oe.exchange.GetPosition(ctx, symbol)
	if err != nil {
		return fmt.Errorf("failed to get position: %w", err)
	}

	if position == nil || position.Quantity == 0 {
		return fmt.Errorf("no position found for %s", symbol)
	}

	// 2. 确定止损订单方向（与仓位相反）
	var side string
	if position.Quantity > 0 {
		side = "SELL" // 多头仓位用卖出止损
	} else {
		side = "BUY" // 空头仓位用买入止损
	}

	// 3. 构建止损订单
	orderReq := &exchange.OrderRequest{
		Symbol:    symbol,
		Side:      side,
		Type:      "STOP_MARKET",
		Quantity:  position.Quantity, // 全部仓位
		StopPrice: stopPrice,
	}

	// 如果是空头仓位，数量需要转为正数
	if orderReq.Quantity < 0 {
		orderReq.Quantity = -orderReq.Quantity
	}

	// 4. 执行止损订单
	orderResp, err := oe.exchange.PlaceOrder(ctx, orderReq)
	if err != nil {
		return fmt.Errorf("failed to place stop loss order: %w", err)
	}

	if !orderResp.Success {
		return fmt.Errorf("stop loss order rejected: %s", orderResp.Error)
	}

	log.Printf("Stop loss order placed: %s, OrderID: %s", symbol, orderResp.OrderID)
	return nil
}

// placeTakeProfit 设置止盈
func (oe *OrderExecutor) placeTakeProfit(ctx context.Context, action *ExecutionAction) error {
	symbol := action.Symbol
	profitPrice, ok := action.Parameters["profit_price"].(float64)
	if !ok {
		return fmt.Errorf("invalid profit_price parameter")
	}

	log.Printf("Placing take profit for %s at price: %.4f", symbol, profitPrice)

	// TODO: 实现止盈逻辑
	return nil
}
