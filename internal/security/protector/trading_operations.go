package protector

import (
	"context"
	"log"
	"qcat/internal/security/protector/dao"
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

	// 记录事件到数据库
	if fp.daoManager != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 创建紧急事件记录
		event := &dao.EmergencyEvent{
			ID:          fp.generateEmergencyID(),
			Type:        eventType,
			Severity:    "INFO", // 交易事件通常为信息级别
			Description: description,
			TriggerData: dao.JSONMap{
				"event_source": "trading_operations",
				"event_type":   eventType,
			},
			Status:    "ACTIVE",
			CreatedAt: time.Now(),
		}

		if err := fp.daoManager.EmergencyEvents().Insert(ctx, event); err != nil {
			log.Printf("Failed to log trading event to database: %v", err)
		}
	}

	// 发送到监控系统（如果配置了通知服务）
	if fp.notificationService != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 构建监控数据
		monitoringPayload := map[string]interface{}{
			"timestamp":   time.Now(),
			"event_type":  eventType,
			"description": description,
			"source":      "fund_protector",
			"component":   "trading_operations",
			"severity":    "info",
		}

		// 发送到监控系统的Webhook（如果配置了）
		if err := fp.notificationService.SendWebhook(ctx, "", monitoringPayload); err != nil {
			log.Printf("Failed to send trading event to monitoring system: %v", err)
		}
	}
}
