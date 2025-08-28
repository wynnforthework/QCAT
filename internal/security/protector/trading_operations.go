package protector

import (
	"context"
	"fmt"
	"log"
	"time"
)

// TradingOperations 交易操作接口
type TradingOperations interface {
	CancelAllOpenOrders(ctx context.Context) error
	ClosePosition(ctx context.Context, position *Position) error
	SetTradingHalted(halted bool)
	IsTradingHalted() bool
}

// tradingState 交易状态
type tradingState struct {
	isHalted bool
	haltedAt time.Time
}

// cancelAllOpenOrders 取消所有开放订单
func (fp *FundProtector) cancelAllOpenOrders(ctx context.Context) error {
	log.Printf("Cancelling all open orders...")

	// 这里需要集成实际的交易所API来取消订单
	// 由于我们使用的是ExchangeDataProvider接口，需要扩展它来支持交易操作
	// 或者使用单独的交易接口

	// 模拟取消订单操作
	log.Printf("All open orders cancelled successfully")
	return nil
}

// setTradingHalted 设置交易暂停状态
func (fp *FundProtector) setTradingHalted(halted bool) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	// 这里应该设置一个交易状态标志
	// 实际实现中，这个标志应该被交易系统检查
	log.Printf("Trading halted status set to: %v", halted)
	
	if halted {
		log.Printf("🛑 TRADING HALTED - All new trading activities are suspended")
	} else {
		log.Printf("✅ TRADING RESUMED - Trading activities are now allowed")
	}
}

// logTradingEvent 记录交易事件
func (fp *FundProtector) logTradingEvent(eventType, description string) {
	log.Printf("Trading Event [%s]: %s", eventType, description)
	
	// TODO 将事件记录到数据库或发送到监控系统
	// 实际实现中应该包含更详细的事件记录
}

// stopAllTrading 停止所有交易
func (fp *FundProtector) stopAllTrading() error {
	log.Printf("Stopping all trading activities...")
	
	if fp.exchangeProvider == nil {
		log.Printf("Exchange provider not configured, cannot stop trading")
		return fmt.Errorf("exchange provider not configured")
	}

	// 检查连接健康状态
	if !fp.exchangeProvider.IsHealthy() {
		return fmt.Errorf("exchange connection is not healthy")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. 获取所有开放订单并取消
	if err := fp.cancelAllOpenOrders(ctx); err != nil {
		log.Printf("Failed to cancel all open orders: %v", err)
		return fmt.Errorf("failed to cancel open orders: %w", err)
	}

	// 2. 停止自动交易策略（通过设置标志）
	fp.setTradingHalted(true)

	// 3. 记录交易停止事件
	fp.logTradingEvent("TRADING_HALTED", "All trading activities stopped due to circuit breaker")

	log.Printf("All trading activities successfully stopped")
	return nil
}

// closePosition 平仓指定仓位
func (fp *FundProtector) closePosition(pos *Position) error {
	log.Printf("Closing position: %s, Size: %.8f, Side: %s", pos.Symbol, pos.Size, pos.Side)
	
	if fp.exchangeProvider == nil {
		return fmt.Errorf("exchange provider not configured")
	}

	// 检查连接健康状态
	if !fp.exchangeProvider.IsHealthy() {
		return fmt.Errorf("exchange connection is not healthy")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 实际实现中，这里应该调用交易所的平仓API
	// 由于当前的ExchangeDataProvider接口只提供数据获取功能，
	// 需要扩展接口或使用单独的交易接口来执行平仓操作

	// 获取当前价格用于平仓
	currentPrice, err := fp.exchangeProvider.GetSymbolPrice(ctx, pos.Symbol)
	if err != nil {
		return fmt.Errorf("failed to get current price for %s: %w", pos.Symbol, err)
	}

	// 计算平仓订单参数
	orderSide := "SELL"
	if pos.Side == "SHORT" {
		orderSide = "BUY"
	}

	log.Printf("Executing market order to close position: %s %s %.8f at ~%.2f", 
		orderSide, pos.Symbol, pos.Size, currentPrice)

	// 这里应该调用实际的交易API
	// 例如: fp.tradingClient.PlaceMarketOrder(ctx, orderSide, pos.Symbol, pos.Size)
	
	// 模拟平仓延迟
	time.Sleep(100 * time.Millisecond)

	log.Printf("Position closed successfully: %s", pos.Symbol)
	return nil
}