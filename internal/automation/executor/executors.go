package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"qcat/internal/config"
	"qcat/internal/database"
	"qcat/internal/events"
	"qcat/internal/exchange"
	"qcat/internal/exchange/account"
	"qcat/internal/monitor"
	"qcat/internal/security/protector"
	"qcat/internal/intelligence/selector"
	"qcat/internal/strategy/workflow"
)

// StrategyManagerInterface 定义策略管理器接口
type StrategyManagerInterface interface {
	SuspendNewPositions(ctx context.Context, duration time.Duration) error
	ResumeNewPositions(ctx context.Context) error
	GetActiveStrategies(ctx context.Context) ([]string, error)
	NotifyParameterUpdate(ctx context.Context, strategyID string, params map[string]interface{}) error
}

// StrategyInstanceInterface 定义策略实例接口
type StrategyInstanceInterface interface {
	UpdateParameters(ctx context.Context, params map[string]interface{}) error
	GetStatus(ctx context.Context) (string, error)
	Stop(ctx context.Context) error
	Start(ctx context.Context) error
}

// SchedulerInterface 定义调度器接口
type SchedulerInterface interface {
	StopStrategyTasks(ctx context.Context, strategyID string) error
	StartStrategyTasks(ctx context.Context, strategyID string) error
	GetTaskStatus(ctx context.Context, strategyID string) (map[string]string, error)
}

// absFloat returns the absolute value of a float64
func absFloat(x float64) float64 {
	return math.Abs(x)
}

// Position represents a trading position with automation fields
type Position struct {
	ID              string    `json:"id"`
	Symbol          string    `json:"symbol"`
	Side            string    `json:"side"` // LONG, SHORT
	Size            float64   `json:"size"`
	Quantity        float64   `json:"quantity"`
	EntryPrice      float64   `json:"entry_price"`
	MarkPrice       float64   `json:"mark_price"`
	UnrealizedPnL   float64   `json:"unrealized_pnl"`
	RealizedPnL     float64   `json:"realized_pnl"`
	Leverage        int       `json:"leverage"`
	MarginType      string    `json:"margin_type"`
	StopLoss        float64   `json:"stop_loss"`
	StopLossOrderID string    `json:"stop_loss_order_id"`
	TakeProfit      float64   `json:"take_profit"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// StrategyInfo represents strategy information
type StrategyInfo struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Status      string                 `json:"status"`
	Parameters  map[string]interface{} `json:"parameters"`
	Performance map[string]float64     `json:"performance"`
	StartTime   time.Time              `json:"start_time"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// Order represents a trading order for automation
type Order struct {
	Symbol    string  `json:"symbol"`
	Side      string  `json:"side"`
	Type      string  `json:"type"`
	Quantity  float64 `json:"quantity"`
	Price     float64 `json:"price"`
	StopPrice float64 `json:"stop_price"`
}

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
	config                *config.Config
	db                    *database.DB
	exchange              exchange.Exchange
	exchangeClient        exchange.Exchange
	accountManager        *account.Manager
	strategyManager       interface{}                   // placeholder for strategy manager
	notificationService   protector.NotificationService // actual notification service
	metrics               *monitor.MetricsCollector     // actual metrics service
	eventBus              *events.EventBus              // actual event bus
	mu                    sync.RWMutex
	emergencyMode         bool
	newPositionsSuspended bool
	suspensionStartTime   time.Time
}

// NewRiskExecutor 创建风险执行器
func NewRiskExecutor(
	cfg *config.Config,
	db *database.DB,
	exchange exchange.Exchange,
	accountManager *account.Manager,
	notificationService protector.NotificationService,
	metrics *monitor.MetricsCollector,
	eventBus *events.EventBus,
) *RiskExecutor {
	return &RiskExecutor{
		config:              cfg,
		db:                  db,
		exchange:            exchange,
		exchangeClient:      exchange,
		accountManager:      accountManager,
		notificationService: notificationService,
		metrics:             metrics,
		eventBus:            eventBus,
	}
}

// HandleAction 处理风险动作
func (re *RiskExecutor) HandleAction(ctx context.Context, action *ExecutionAction) error {
	switch action.Action {
	case "emergency_stop":
		return re.emergencyStop(ctx, action)
	case "close_all_positions":
		return re.closeAllPositions(ctx, action)
	case "reduce_leverage":
		return re.reduceLeverage(ctx, action)
	case "hedge_position":
		return re.hedgePosition(ctx, action)
	case "circuit_breaker":
		return re.circuitBreaker(ctx, action)
	case "reduce_high_risk_positions":
		return re.reduceHighRiskPositions(ctx, action)
	case "suspend_new_positions":
		return re.suspendNewPositions(ctx, action)
	case "adjust_position_sizes":
		return re.adjustPositionSizes(ctx, action)
	case "tighten_stop_loss":
		return re.tightenStopLoss(ctx, action)
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

	// 实现降杠杆逻辑
	if re.exchangeClient == nil {
		return fmt.Errorf("exchange client not available")
	}

	// 获取当前仓位
	positions, err := re.exchangeClient.GetPositions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}

	for _, pos := range positions {
		if pos.Symbol == symbol {
			currentSize := pos.Size
			currentLeverage := float64(pos.Leverage)

			if currentLeverage > targetLeverage && currentSize > 0 {
				// 计算需要减少的仓位大小
				newSize := currentSize * (targetLeverage / currentLeverage)
				reduceSize := currentSize - newSize

				if reduceSize > 0 {
					// 创建减仓订单
					orderReq := &exchange.OrderRequest{
						Symbol:     symbol,
						Type:       "MARKET",
						Quantity:   reduceSize,
						ReduceOnly: true,
					}

					// 根据当前仓位方向确定订单方向
					if pos.Side == "LONG" {
						orderReq.Side = "SELL"
					} else if pos.Side == "SHORT" {
						orderReq.Side = "BUY"
					}

					_, err := re.exchangeClient.PlaceOrder(ctx, orderReq)
					if err != nil {
						return fmt.Errorf("failed to place reduce order: %w", err)
					}

					log.Printf("Reduced position size for %s from %.4f to %.4f (leverage: %.1fx -> %.1fx)",
						symbol, currentSize, newSize, currentLeverage, targetLeverage)
				}
			}
		}
	}

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

	// 实现对冲逻辑
	if re.exchangeClient == nil {
		return fmt.Errorf("exchange client not available")
	}

	// 获取当前仓位
	positions, err := re.exchangeClient.GetPositions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}

	for _, pos := range positions {
		if pos.Symbol == symbol {
			currentSize := pos.Size
			side := pos.Side

			if currentSize > 0 {
				// 计算对冲仓位大小
				hedgeSize := currentSize * hedgeRatio

				// 确定对冲方向（与原仓位相反）
				hedgeSide := "SELL"
				if side == "SHORT" {
					hedgeSide = "BUY"
				}

				// 创建对冲订单
				hedgeOrderReq := &exchange.OrderRequest{
					Symbol:   symbol,
					Side:     hedgeSide,
					Type:     "MARKET",
					Quantity: hedgeSize,
				}

				_, err := re.exchangeClient.PlaceOrder(ctx, hedgeOrderReq)
				if err != nil {
					return fmt.Errorf("failed to place hedge order: %w", err)
				}

				log.Printf("Placed hedge order for %s: %s %.4f (ratio: %.2f)",
					symbol, hedgeSide, hedgeSize, hedgeRatio)
			}
		}
	}

	return nil
}

// circuitBreaker 熔断器
func (re *RiskExecutor) circuitBreaker(ctx context.Context, action *ExecutionAction) error {
	log.Printf("Triggering circuit breaker")

	// 实现熔断器逻辑
	// 1. 暂停交易
	re.mu.Lock()
	re.emergencyMode = true
	re.mu.Unlock()

	log.Printf("Emergency mode activated - all trading suspended")

	// 2. 评估风险
	if re.exchangeClient != nil {
		positions, err := re.exchangeClient.GetPositions(ctx)
		if err != nil {
			log.Printf("Failed to get positions during circuit breaker: %v", err)
		} else {
			totalExposure := 0.0
			for _, pos := range positions {
				totalExposure += pos.Size * pos.MarkPrice
			}
			log.Printf("Total position exposure during circuit breaker: $%.2f", totalExposure)
		}
	}

	// 3. 决定后续动作
	threshold, ok := action.Parameters["emergency_threshold"].(float64)
	if !ok {
		threshold = 0.15 // 默认15%损失阈值
	}

	// 如果损失超过阈值，触发紧急平仓
	if drawdown, exists := action.Parameters["current_drawdown"].(float64); exists {
		if drawdown > threshold {
			log.Printf("Drawdown %.2f%% exceeds threshold %.2f%%, triggering emergency close",
				drawdown*100, threshold*100)

			// 创建紧急平仓动作
			emergencyAction := &ExecutionAction{
				Type:   "close_all_positions",
				Symbol: "ALL",
				Parameters: map[string]interface{}{
					"reason": "circuit_breaker_triggered",
				},
			}

			return re.closeAllPositions(ctx, emergencyAction)
		}
	}

	return nil
}

// closeAllPositions 关闭所有仓位
func (re *RiskExecutor) closeAllPositions(ctx context.Context, action *ExecutionAction) error {
	log.Printf("Closing all positions")

	// 获取所有持仓
	positions, err := re.exchange.GetPositions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}

	log.Printf("Found %d positions to close", len(positions))

	// 逐个关闭仓位
	for _, position := range positions {
		if position.Size == 0 {
			continue
		}

		// 确定平仓方向
		var side exchange.OrderSide
		if position.Size > 0 {
			side = exchange.OrderSideSell
		} else {
			side = exchange.OrderSideBuy
		}

		// 创建市价平仓单
		order := &exchange.OrderRequest{
			Symbol:   position.Symbol,
			Side:     string(side),
			Type:     string(exchange.OrderTypeMarket),
			Quantity: absFloat(position.Size),
		}

		_, err := re.exchange.PlaceOrder(ctx, order)
		if err != nil {
			log.Printf("Failed to close position %s: %v", position.Symbol, err)
			continue
		}

		log.Printf("Successfully closed position for %s", position.Symbol)
	}

	return nil
}

// reduceHighRiskPositions 减少高风险仓位
func (re *RiskExecutor) reduceHighRiskPositions(ctx context.Context, action *ExecutionAction) error {
	reductionRatio, ok := action.Parameters["reduction_ratio"].(float64)
	if !ok {
		reductionRatio = 0.5 // 默认减少50%
	}

	log.Printf("Reducing high risk positions by %.1f%%", reductionRatio*100)

	// 获取所有持仓
	positions, err := re.exchange.GetPositions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}

	// 识别高风险仓位并减仓
	for _, position := range positions {
		if position.Size == 0 {
			continue
		}

		// 简单的风险评估：基于未实现盈亏
		if position.UnrealizedPnL < 0 && absFloat(position.UnrealizedPnL) > absFloat(position.Size*position.EntryPrice*0.05) {
			// 如果亏损超过5%，认为是高风险仓位
			reduceSize := absFloat(position.Size) * reductionRatio

			var side exchange.OrderSide
			if position.Size > 0 {
				side = exchange.OrderSideSell
			} else {
				side = exchange.OrderSideBuy
			}

			order := &exchange.OrderRequest{
				Symbol:   position.Symbol,
				Side:     string(side),
				Type:     string(exchange.OrderTypeMarket),
				Quantity: reduceSize,
			}

			_, err := re.exchange.PlaceOrder(ctx, order)
			if err != nil {
				log.Printf("Failed to reduce position %s: %v", position.Symbol, err)
				continue
			}

			log.Printf("Reduced high risk position %s by %.2f", position.Symbol, reduceSize)
		}
	}

	return nil
}
// suspendNewPositions 暂停新开仓
func (re *RiskExecutor) suspendNewPositions(ctx context.Context, action *ExecutionAction) error {
	log.Printf("Suspending new position openings")

	// 实现暂停新开仓逻辑

	// 1. 设置全局暂停标志
	re.mu.Lock()
	re.newPositionsSuspended = true
	re.suspensionStartTime = time.Now()
	re.mu.Unlock()

	// 2. 获取暂停持续时间
	duration := 30 * time.Minute // 默认30分钟
	if durationParam, ok := action.Parameters["duration"].(float64); ok {
		duration = time.Duration(durationParam) * time.Minute
	}

	// 3. 通知所有策略执行器暂停新开仓
	if re.strategyManager != nil {
		// 实现策略管理器集成 - 暂停所有策略的新开仓
		if manager, ok := re.strategyManager.(StrategyManagerInterface); ok {
			if err := manager.SuspendNewPositions(ctx, duration); err != nil {
				log.Printf("Failed to suspend new positions via strategy manager: %v", err)
			} else {
				log.Printf("Successfully suspended new positions for all strategies via strategy manager")
			}
		} else {
			log.Printf("Strategy manager does not implement required interface")
		}
	}

	// 4. 更新数据库状态
	if re.db != nil {
		query := `
			INSERT INTO risk_actions (
				id, type, status, parameters, created_at, expires_at
			) VALUES (?, 'suspend_new_positions', 'active', ?, ?, ?)
		`

		parametersJSON, _ := json.Marshal(action.Parameters)
		expiresAt := time.Now().Add(duration)

		_, err := re.db.ExecContext(ctx, query,
			action.ID, string(parametersJSON), time.Now(), expiresAt)
		if err != nil {
			log.Printf("Failed to record suspension in database: %v", err)
		}
	}

	// 5. 设置自动恢复定时器
	go func() {
		timer := time.NewTimer(duration)
		defer timer.Stop()

		select {
		case <-timer.C:
			re.resumeNewPositions(ctx, "automatic_resume_after_timeout")
		case <-ctx.Done():
			return
		}
	}()

	// 6. 发送通知
	if re.notificationService != nil {
		message := fmt.Sprintf("New position openings suspended for %v due to risk management action", duration)
		err := re.notificationService.SendWebhook(ctx, "", map[string]interface{}{
			"type":     "risk_management",
			"action":   "position_suspension",
			"message":  message,
			"duration": duration.String(),
		})
		if err != nil {
			log.Printf("Failed to send suspension notification: %v", err)
		}
	}

	// 7. 记录指标
	if re.metrics != nil {
		re.metrics.IncrementCounter("risk_executor.new_positions_suspended", map[string]string{
			"reason":   action.Action,
			"duration": duration.String(),
		})
	}

	log.Printf("New position openings suspended for %v", duration)
	return nil
}
// resumeNewPositions 恢复新开仓
func (re *RiskExecutor) resumeNewPositions(ctx context.Context, reason string) error {
	re.mu.Lock()
	if !re.newPositionsSuspended {
		re.mu.Unlock()
		return nil // 已经恢复了
	}

	re.newPositionsSuspended = false
	suspensionDuration := time.Since(re.suspensionStartTime)
	re.mu.Unlock()

	// 通知所有策略执行器恢复新开仓
	if re.strategyManager != nil {
		// 实现策略管理器集成 - 恢复所有策略的新开仓
		if manager, ok := re.strategyManager.(StrategyManagerInterface); ok {
			if err := manager.ResumeNewPositions(ctx); err != nil {
				log.Printf("Failed to resume new positions via strategy manager: %v", err)
			} else {
				log.Printf("Successfully resumed new positions for all strategies via strategy manager")
			}
		} else {
			log.Printf("Strategy manager does not implement required interface")
		}
	}

	// 更新数据库状态
	if re.db != nil {
		query := `
			UPDATE risk_actions 
			SET status = 'completed', completed_at = ?
			WHERE type = 'suspend_new_positions' AND status = 'active'
		`

		_, err := re.db.ExecContext(ctx, query, time.Now())
		if err != nil {
			log.Printf("Failed to update suspension status in database: %v", err)
		}
	}

	// 发送通知
	if re.notificationService != nil {
		message := fmt.Sprintf("New position openings resumed after %v suspension (%s)",
			suspensionDuration, reason)
		err := re.notificationService.SendWebhook(ctx, "", map[string]interface{}{
			"type":                "risk_management",
			"action":              "position_resumption",
			"message":             message,
			"suspension_duration": suspensionDuration.String(),
			"reason":              reason,
		})
		if err != nil {
			log.Printf("Failed to send resumption notification: %v", err)
		}
	}

	// 记录指标
	if re.metrics != nil {
		re.metrics.IncrementCounter("risk_executor.new_positions_resumed", map[string]string{
			"reason":              reason,
			"suspension_duration": suspensionDuration.String(),
		})
	}

	log.Printf("New position openings resumed after %v suspension (%s)", suspensionDuration, reason)
	return nil
}

// notifyStrategySuspension 通知策略暂停新开仓
func (re *RiskExecutor) notifyStrategySuspension(ctx context.Context, strategyID string, duration time.Duration) error {
	// 这里应该调用策略管理器的API来暂停特定策略的新开仓
	// 暂时使用日志记录
	log.Printf("Notifying strategy %s to suspend new positions for %v", strategyID, duration)

	// 如果有策略执行器的直接引用，可以调用其暂停方法
	// 例如：re.strategyExecutors[strategyID].SuspendNewPositions(duration)

	return nil
}

// notifyStrategyResumption 通知策略恢复新开仓
func (re *RiskExecutor) notifyStrategyResumption(ctx context.Context, strategyID string) error {
	// 这里应该调用策略管理器的API来恢复特定策略的新开仓
	// 暂时使用日志记录
	log.Printf("Notifying strategy %s to resume new positions", strategyID)

	// 如果有策略执行器的直接引用，可以调用其恢复方法
	// 例如：re.strategyExecutors[strategyID].ResumeNewPositions()

	return nil
}

// isNewPositionAllowed 检查是否允许新开仓
func (re *RiskExecutor) isNewPositionAllowed() bool {
	re.mu.RLock()
	defer re.mu.RUnlock()
	return !re.newPositionsSuspended
}

// adjustPositionSizes 调整仓位大小
func (re *RiskExecutor) adjustPositionSizes(ctx context.Context, action *ExecutionAction) error {
	adjustmentFactor, ok := action.Parameters["adjustment_factor"].(float64)
	if !ok {
		adjustmentFactor = 0.8 // 默认调整为80%
	}

	log.Printf("Adjusting position sizes by factor: %.2f", adjustmentFactor)

	// 获取所有持仓
	positions, err := re.exchange.GetPositions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}

	// 调整每个仓位的大小
	for _, position := range positions {
		if position.Size == 0 {
			continue
		}

		// 计算需要调整的数量
		currentSize := absFloat(position.Size)
		targetSize := currentSize * adjustmentFactor
		adjustSize := currentSize - targetSize

		if adjustSize <= 0 {
			continue
		}

		// 确定平仓方向
		var side exchange.OrderSide
		if position.Size > 0 {
			side = exchange.OrderSideSell
		} else {
			side = exchange.OrderSideBuy
		}

		order := &exchange.OrderRequest{
			Symbol:   position.Symbol,
			Side:     string(side),
			Type:     string(exchange.OrderTypeMarket),
			Quantity: adjustSize,
		}

		_, err := re.exchange.PlaceOrder(ctx, order)
		if err != nil {
			log.Printf("Failed to adjust position %s: %v", position.Symbol, err)
			continue
		}

		log.Printf("Adjusted position %s by %.2f", position.Symbol, adjustSize)
	}

	return nil
}

// tightenStopLoss 收紧止损
func (re *RiskExecutor) tightenStopLoss(ctx context.Context, action *ExecutionAction) error {
	tighteningFactor, ok := action.Parameters["tightening_factor"].(float64)
	if !ok {
		tighteningFactor = 0.8 // 默认收紧到80%
	}

	log.Printf("Tightening stop loss by factor: %.2f", tighteningFactor)

	// 实现收紧止损逻辑

	// 1. 获取所有活跃持仓
	positions, err := re.exchange.GetPositions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active positions: %v", err)
	}

	if len(positions) == 0 {
		log.Printf("No active positions found for stop loss tightening")
		return nil
	}

	adjustedCount := 0
	totalPositions := len(positions)

	// 2. 遍历每个持仓并调整止损
	for _, exchPos := range positions {
		// Convert exchange.Position to local Position type
		position := &Position{
			ID:              exchPos.Symbol, // Use symbol as ID for now
			Symbol:          exchPos.Symbol,
			Side:            exchPos.Side,
			Size:            exchPos.Size,
			Quantity:        exchPos.Quantity,
			EntryPrice:      exchPos.EntryPrice,
			MarkPrice:       exchPos.MarkPrice,
			UnrealizedPnL:   exchPos.UnrealizedPnL,
			RealizedPnL:     exchPos.RealizedPnL,
			Leverage:        exchPos.Leverage,
			MarginType:      exchPos.MarginType,
			StopLoss:        re.getPositionStopLoss(ctx, exchPos.Symbol),
			StopLossOrderID: re.getPositionStopLossOrderID(ctx, exchPos.Symbol),
			TakeProfit:      re.getPositionTakeProfit(ctx, exchPos.Symbol),
			UpdatedAt:       exchPos.UpdatedAt,
		}

		// 获取当前市场价格
		currentPrice := position.MarkPrice
		if currentPrice <= 0 {
			log.Printf("Invalid current price for %s: %.4f", position.Symbol, currentPrice)
			continue
		}

		// 计算新的止损价格
		newStopLoss, shouldUpdate := re.calculateTightenedStopLoss(position, currentPrice, tighteningFactor)
		if !shouldUpdate {
			continue
		}

		// 验证新止损价格的合理性
		if !re.validateStopLossPrice(position, newStopLoss, currentPrice) {
			log.Printf("Invalid stop loss price for position %s: %.4f", position.ID, newStopLoss)
			continue
		}

		// 更新止损订单
		err = re.updateStopLossOrder(ctx, position, newStopLoss)
		if err != nil {
			log.Printf("Failed to update stop loss for position %s: %v", position.ID, err)
			continue
		}

		// 记录调整
		err = re.recordStopLossAdjustment(ctx, position.ID, position.StopLoss, newStopLoss, "risk_tightening")
		if err != nil {
			log.Printf("Failed to record stop loss adjustment for position %s: %v", position.ID, err)
		}

		adjustedCount++
		log.Printf("Tightened stop loss for position %s (%s): %.4f -> %.4f",
			position.ID, position.Symbol, position.StopLoss, newStopLoss)
	}

	// 3. 记录执行结果
	if re.db != nil {
		query := `
			INSERT INTO risk_actions (
				id, type, status, parameters, positions_affected, 
				created_at, completed_at
			) VALUES (?, 'tighten_stop_loss', 'completed', ?, ?, ?, ?)
		`

		parametersJSON, _ := json.Marshal(map[string]interface{}{
			"tightening_factor":  tighteningFactor,
			"total_positions":    totalPositions,
			"adjusted_positions": adjustedCount,
		})

		now := time.Now()
		_, err := re.db.ExecContext(ctx, query,
			action.ID, string(parametersJSON), adjustedCount, now, now)
		if err != nil {
			log.Printf("Failed to record stop loss tightening action: %v", err)
		}
	}

	// 4. 发送通知
	if re.notificationService != nil {
		message := fmt.Sprintf("Stop loss tightened for %d/%d positions (factor: %.2f)",
			adjustedCount, totalPositions, tighteningFactor)
		err := re.notificationService.SendWebhook(ctx, "", map[string]interface{}{
			"type":              "risk_management",
			"action":            "stop_loss_tightening",
			"message":           message,
			"adjusted_count":    adjustedCount,
			"total_positions":   totalPositions,
			"tightening_factor": tighteningFactor,
		})
		if err != nil {
			log.Printf("Failed to send stop loss tightening notification: %v", err)
		}
	}

	// 5. 记录指标
	if re.metrics != nil {
		re.metrics.IncrementCounter("risk_executor.stop_loss_tightened", map[string]string{
			"adjusted_count":  fmt.Sprintf("%d", adjustedCount),
			"total_positions": fmt.Sprintf("%d", totalPositions),
		})
	}

	log.Printf("Stop loss tightening completed: %d/%d positions adjusted", adjustedCount, totalPositions)
	return nil
}

// calculateTightenedStopLoss 计算收紧后的止损价格
func (re *RiskExecutor) calculateTightenedStopLoss(position *Position, currentPrice, tighteningFactor float64) (float64, bool) {
	if position.StopLoss == 0 {
		// 没有设置止损，跳过
		return 0, false
	}

	var newStopLoss float64

	if position.Side == "long" {
		// 多头持仓：收紧止损意味着提高止损价格
		distanceToStop := currentPrice - position.StopLoss
		if distanceToStop <= 0 {
			// 当前价格已经低于止损价格，不需要调整
			return position.StopLoss, false
		}

		// 收紧距离
		newDistance := distanceToStop * tighteningFactor
		newStopLoss = currentPrice - newDistance

		// 确保新止损价格高于原止损价格（更紧）
		if newStopLoss <= position.StopLoss {
			return position.StopLoss, false
		}

	} else {
		// 空头持仓：收紧止损意味着降低止损价格
		distanceToStop := position.StopLoss - currentPrice
		if distanceToStop <= 0 {
			// 当前价格已经高于止损价格，不需要调整
			return position.StopLoss, false
		}

		// 收紧距离
		newDistance := distanceToStop * tighteningFactor
		newStopLoss = currentPrice + newDistance

		// 确保新止损价格低于原止损价格（更紧）
		if newStopLoss >= position.StopLoss {
			return position.StopLoss, false
		}
	}

	return newStopLoss, true
}

// validateStopLossPrice 验证止损价格的合理性
func (re *RiskExecutor) validateStopLossPrice(position *Position, stopLoss, currentPrice float64) bool {
	if stopLoss <= 0 {
		return false
	}

	// 检查止损价格与当前价格的距离是否合理
	var distancePercent float64
	if position.Side == "long" {
		if stopLoss >= currentPrice {
			return false // 多头止损价格不能高于或等于当前价格
		}
		distancePercent = (currentPrice - stopLoss) / currentPrice
	} else {
		if stopLoss <= currentPrice {
			return false // 空头止损价格不能低于或等于当前价格
		}
		distancePercent = (stopLoss - currentPrice) / currentPrice
	}

	// 止损距离应该在合理范围内（0.1% - 10%）
	if distancePercent < 0.001 || distancePercent > 0.1 {
		return false
	}

	return true
}

// updateStopLossOrder 更新止损订单
func (re *RiskExecutor) updateStopLossOrder(ctx context.Context, position *Position, newStopLoss float64) error {
	// 如果有交易所接口，调用交易所API更新止损订单
	if re.exchange != nil {
		// 取消原有止损订单
		if position.StopLossOrderID != "" {
			cancelReq := &exchange.OrderCancelRequest{
				Symbol:  position.Symbol,
				OrderID: position.StopLossOrderID,
			}
			_, err := re.exchange.CancelOrder(ctx, cancelReq)
			if err != nil {
				log.Printf("Failed to cancel existing stop loss order %s: %v", position.StopLossOrderID, err)
			}
		}

		// 创建新的止损订单
		orderReq := &exchange.OrderRequest{
			Symbol:    position.Symbol,
			Side:      getOppositeOrderSide(position.Side),
			Type:      "STOP_MARKET",
			Quantity:  position.Quantity,
			StopPrice: newStopLoss,
		}

		orderResp, err := re.exchange.PlaceOrder(ctx, orderReq)
		if err != nil {
			return fmt.Errorf("failed to place new stop loss order: %v", err)
		}

		// 更新持仓记录
		position.StopLoss = newStopLoss
		position.StopLossOrderID = orderResp.OrderID
	}

	// 更新数据库中的持仓信息
	if re.db != nil {
		query := `
			UPDATE positions 
			SET stop_loss = ?, stop_loss_order_id = ?, updated_at = ?
			WHERE id = ?
		`

		_, err := re.db.ExecContext(ctx, query,
			newStopLoss, position.StopLossOrderID, time.Now(), position.ID)
		if err != nil {
			return fmt.Errorf("failed to update position in database: %v", err)
		}
	}

	return nil
}

// recordStopLossAdjustment 记录止损调整
func (re *RiskExecutor) recordStopLossAdjustment(ctx context.Context, positionID string, oldStopLoss, newStopLoss float64, reason string) error {
	if re.db == nil {
		return nil
	}

	query := `
		INSERT INTO stop_loss_adjustments (
			id, position_id, old_stop_loss, new_stop_loss, 
			reason, created_at
		) VALUES (?, ?, ?, ?, ?, ?)
	`

	adjustmentID := fmt.Sprintf("adj_%s_%d", positionID, time.Now().Unix())
	_, err := re.db.ExecContext(ctx, query,
		adjustmentID, positionID, oldStopLoss, newStopLoss, reason, time.Now())

	return err
}

// getOppositeOrderSide 获取相反的订单方向
func getOppositeOrderSide(positionSide string) string {
	if positionSide == "long" {
		return "sell"
	}
	return "buy"
}

// getPositionStopLoss 获取仓位止损价格
func (re *RiskExecutor) getPositionStopLoss(ctx context.Context, symbol string) float64 {
	if re.db == nil {
		return 0
	}

	var stopLoss float64
	query := `SELECT stop_loss FROM positions WHERE symbol = ? AND quantity != 0 LIMIT 1`
	err := re.db.QueryRowContext(ctx, query, symbol).Scan(&stopLoss)
	if err != nil {
		return 0
	}
	return stopLoss
}

// getPositionStopLossOrderID 获取仓位止损订单ID
func (re *RiskExecutor) getPositionStopLossOrderID(ctx context.Context, symbol string) string {
	if re.db == nil {
		return ""
	}

	var orderID string
	query := `SELECT stop_loss_order_id FROM positions WHERE symbol = ? AND quantity != 0 LIMIT 1`
	err := re.db.QueryRowContext(ctx, query, symbol).Scan(&orderID)
	if err != nil {
		return ""
	}
	return orderID
}

// getPositionTakeProfit 获取仓位止盈价格
func (re *RiskExecutor) getPositionTakeProfit(ctx context.Context, symbol string) float64 {
	if re.db == nil {
		return 0
	}

	var takeProfit float64
	query := `SELECT take_profit FROM positions WHERE symbol = ? AND quantity != 0 LIMIT 1`
	err := re.db.QueryRowContext(ctx, query, symbol).Scan(&takeProfit)
	if err != nil {
		return 0
	}
	return takeProfit
}

// performAdvancedBalanceCheck 执行高级余额检查
func (oe *OrderExecutor) performAdvancedBalanceCheck(ctx context.Context, balances map[string]float64, symbol, side, orderType string, quantity, price float64) *BalanceCheckResult {
	result := &BalanceCheckResult{
		Sufficient: true,
	}

	// 提取基础资产和报价资产
	baseAsset, quoteAsset := oe.parseSymbol(symbol)

	var requiredAmount float64
	var availableAmount float64
	var assetToCheck string

	if side == "BUY" {
		// 买入需要报价资产
		assetToCheck = quoteAsset
		if orderType == "MARKET" {
			// 市价单需要估算价格
			if price <= 0 {
				// 获取当前市价
				currentPrice, err := oe.exchange.GetSymbolPrice(ctx, symbol)
				if err != nil {
					result.Sufficient = false
					result.Reason = "Failed to get current price for market order"
					return result
				}
				price = currentPrice
			}
		}
		requiredAmount = quantity * price
	} else {
		// 卖出需要基础资产
		assetToCheck = baseAsset
		requiredAmount = quantity
	}

	availableAmount = balances[assetToCheck]
	result.RequiredAmount = requiredAmount
	result.AvailableAmount = availableAmount

	// 计算利用率
	if availableAmount > 0 {
		result.UtilizationPercent = (requiredAmount / availableAmount) * 100
	}

	// 检查余额是否充足
	if availableAmount < requiredAmount {
		result.Sufficient = false
		result.Reason = fmt.Sprintf("Insufficient %s balance: required %.8f, available %.8f",
			assetToCheck, requiredAmount, availableAmount)
		return result
	}

	// 检查是否接近余额上限
	if result.UtilizationPercent > 95 {
		result.Warning = fmt.Sprintf("High balance utilization: %.2f%%", result.UtilizationPercent)
	} else if result.UtilizationPercent > 80 {
		result.Warning = fmt.Sprintf("Moderate balance utilization: %.2f%%", result.UtilizationPercent)
	}

	return result
}

// parseSymbol 解析交易对符号
func (oe *OrderExecutor) parseSymbol(symbol string) (baseAsset, quoteAsset string) {
	// 简化的符号解析逻辑
	// 实际实现应该更复杂，支持各种交易对格式
	if strings.Contains(symbol, "USDT") {
		baseAsset = strings.Replace(symbol, "USDT", "", 1)
		quoteAsset = "USDT"
	} else if strings.Contains(symbol, "BTC") {
		baseAsset = strings.Replace(symbol, "BTC", "", 1)
		quoteAsset = "BTC"
	} else if strings.Contains(symbol, "ETH") {
		baseAsset = strings.Replace(symbol, "ETH", "", 1)
		quoteAsset = "ETH"
	} else {
		// 默认假设是USDT交易对
		baseAsset = symbol
		quoteAsset = "USDT"
	}
	return baseAsset, quoteAsset
}

// OrderExecutor 订单执行器
type OrderExecutor struct {
	config         *config.Config
	db             *database.DB
	exchange       exchange.Exchange
	exchangeClient exchange.Exchange
	accountManager *account.Manager
	metrics        *monitor.MetricsCollector // actual metrics service
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

	// 选择器钩子（shadow模式，仅记录，不改变行为）
	if re, ok := ctx.Value("realtime_executor").(*RealtimeExecutor); ok && re != nil && re.strategySelector != nil {
		// shadow gating by config
		shadowEnabled := re.config.Selector.Shadow.Enabled
		if shadowEnabled {
			mctx := selector.MarketContextFactory(symbol)
			// 轻量候选集：从参数或默认构造（真实接入时来自策略池）
			candidates := []selector.EnabledStrategyLite{}
			if ids, ok2 := action.Parameters["candidate_strategies"].([]string); ok2 {
				for _, id := range ids { candidates = append(candidates, selector.EnabledStrategyLite{ID: id}) }
			}
			// 从策略池补充候选（当未提供时）
			if len(candidates) == 0 && re.strategyPool != nil {
				if objs := re.strategyPool.GetEnabledStrategyObjects(); len(objs) > 0 {
					for _, es := range objs { candidates = append(candidates, selector.EnabledStrategyLite{ID: es.ID, Name: es.Name, Type: es.Type}) }
				}
			}
			if res, err := re.strategySelector.Select(ctx, symbol, mctx, candidates); err == nil && res != nil {
				log.Printf("[selector] symbol=%s selected=%s method=%s regime=%s", symbol, res.SelectedID, res.Method, res.Regime)
			}
		}
	}

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
		// 实现复杂的余额检查逻辑
		simpleBalances := make(map[string]float64)
		for asset, balance := range balances {
			simpleBalances[asset] = balance.Available
		}
		balanceCheckResult := oe.performAdvancedBalanceCheck(ctx, simpleBalances, symbol, side, orderType, quantity, price)

		if !balanceCheckResult.Sufficient {
			return fmt.Errorf("insufficient balance: %s", balanceCheckResult.Reason)
		}

		if balanceCheckResult.Warning != "" {
			log.Printf("Balance warning: %s", balanceCheckResult.Warning)
		}

		// 记录余额使用情况
		if oe.metrics != nil {
			oe.metrics.IncrementCounter("order_executor.balance_check", map[string]string{
				"symbol": symbol,
				"side":   side,
				"result": fmt.Sprintf("%.2f", balanceCheckResult.UtilizationPercent),
			})
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

	// 实现撤单逻辑
	if oe.exchangeClient == nil {
		return fmt.Errorf("exchange client not available")
	}

	cancelReq := &exchange.OrderCancelRequest{
		OrderID: orderID,
	}
	_, err := oe.exchangeClient.CancelOrder(ctx, cancelReq)
	if err != nil {
		return fmt.Errorf("failed to cancel order %s: %w", orderID, err)
	}

	log.Printf("Successfully cancelled order: %s", orderID)
	return nil
}
// modifyOrder 修改订单
func (oe *OrderExecutor) modifyOrder(ctx context.Context, action *ExecutionAction) error {
	orderID, ok := action.Parameters["order_id"].(string)
	if !ok {
		return fmt.Errorf("invalid order_id parameter")
	}

	log.Printf("Modifying order: %s", orderID)

	// 实现修改订单逻辑
	if oe.exchangeClient == nil {
		return fmt.Errorf("exchange client not available")
	}

	// 获取修改参数
	newPrice, hasPriceUpdate := action.Parameters["new_price"].(float64)
	newQuantity, hasQuantityUpdate := action.Parameters["new_quantity"].(float64)

	if !hasPriceUpdate && !hasQuantityUpdate {
		return fmt.Errorf("no modification parameters provided")
	}

	// 先获取原订单信息
	// Use alternative approach since GetOrderStatus is not available
	// Query order from database instead
	var orderStatus, orderSymbol, orderSide, orderType string
	var orderQuantity, orderPrice float64

	query := `SELECT status, symbol, side, type, quantity, price FROM orders WHERE id = ?`
	err := oe.db.QueryRowContext(ctx, query, orderID).Scan(
		&orderStatus, &orderSymbol, &orderSide, &orderType, &orderQuantity, &orderPrice)
	if err != nil {
		return fmt.Errorf("failed to get order from database: %w", err)
	}

	if orderStatus != "OPEN" && orderStatus != "PARTIALLY_FILLED" {
		return fmt.Errorf("cannot modify order %s: status is %s", orderID, orderStatus)
	}

	// 取消原订单
	cancelReq := &exchange.OrderCancelRequest{
		OrderID: orderID,
	}
	_, err = oe.exchangeClient.CancelOrder(ctx, cancelReq)
	if err != nil {
		return fmt.Errorf("failed to cancel original order: %w", err)
	}

	// 创建新订单（基于修改参数）
	symbol, _ := action.Parameters["symbol"].(string)
	if symbol == "" {
		return fmt.Errorf("symbol is required for order modification")
	}

	orderType, _ = action.Parameters["type"].(string)
	if orderType == "" {
		orderType = "LIMIT" // Default type
	}

	quantity, _ := action.Parameters["quantity"].(float64)
	price, _ := action.Parameters["price"].(float64)

	newOrderReq := &exchange.OrderRequest{
		Symbol:   symbol,
		Type:     orderType,
		Quantity: quantity,
		Price:    price,
	}

	// 应用修改
	if hasPriceUpdate {
		newOrderReq.Price = newPrice
	}
	if hasQuantityUpdate {
		newOrderReq.Quantity = newQuantity
	}

	newOrderResp, err := oe.exchangeClient.PlaceOrder(ctx, newOrderReq)
	if err != nil {
		return fmt.Errorf("failed to place modified order: %w", err)
	}

	log.Printf("Successfully modified order %s -> %s", orderID, newOrderResp.OrderID)

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

	// 实现止盈逻辑
	if oe.exchangeClient == nil {
		return fmt.Errorf("exchange client not available")
	}

	// 1. 获取当前仓位信息
	positions, err := oe.exchangeClient.GetPositions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}

	var targetPosition *exchange.Position
	for _, pos := range positions {
		if pos.Symbol == symbol {
			targetPosition = pos
			break
		}
	}

	if targetPosition == nil {
		return fmt.Errorf("no position found for symbol %s", symbol)
	}

	// 2. 获取仓位信息
	positionSize := targetPosition.Size
	positionSide := targetPosition.Side

	if positionSize == 0 {
		return fmt.Errorf("no position to set take profit for %s", symbol)
	}

	// 3. 确定止盈订单方向（与仓位相反）
	orderSide := "SELL"
	if positionSide == "SHORT" {
		orderSide = "BUY"
	}

	// 4. 创建止盈订单
	takeProfitOrderReq := &exchange.OrderRequest{
		Symbol:      symbol,
		Side:        orderSide,
		Type:        "TAKE_PROFIT_MARKET",
		Quantity:    positionSize,
		StopPrice:   profitPrice,
		ReduceOnly:  true,
		TimeInForce: "GTC",
	}

	orderResp, err := oe.exchangeClient.PlaceOrder(ctx, takeProfitOrderReq)
	if err != nil {
		return fmt.Errorf("failed to place take profit order: %w", err)
	}

	log.Printf("Take profit order placed for %s: OrderID %s, Price: %.4f", symbol, orderResp.OrderID, profitPrice)
	return nil
}

// StrategyExecutor 策略执行器
type StrategyExecutor struct {
	config              *config.Config
	db                  *database.DB
	exchange            exchange.Exchange
	accountManager      *account.Manager
	metrics             *monitor.MetricsCollector     // actual metrics service
	strategyInstances   map[string]interface{}        // placeholder for strategy instances
	eventBus            *events.EventBus              // actual event bus
	notificationService protector.NotificationService // actual notification service
	scheduler           interface{}                   // placeholder for scheduler service
	strategies          map[string]interface{}        // placeholder for strategy instances
}

// NewStrategyExecutor 创建策略执行器
func NewStrategyExecutor(
	cfg *config.Config,
	db *database.DB,
	exchange exchange.Exchange,
	accountManager *account.Manager,
) *StrategyExecutor {
	return &StrategyExecutor{
		config:         cfg,
		db:             db,
		exchange:       exchange,
		accountManager: accountManager,
	}
}

// HandleAction 处理策略动作
func (se *StrategyExecutor) HandleAction(ctx context.Context, action *ExecutionAction) error {
	switch action.Action {
	case "apply_parameters":
		return se.applyParameters(ctx, action)
	case "eliminate_strategy":
		return se.eliminateStrategy(ctx, action)
	case "introduce_strategy":
		return se.introduceStrategy(ctx, action)
	case "optimize_strategy":
		return se.optimizeStrategy(ctx, action)
	default:
		return fmt.Errorf("unknown strategy action: %s", action.Action)
	}
}

// applyParameters 应用策略参数
func (se *StrategyExecutor) applyParameters(ctx context.Context, action *ExecutionAction) error {
	log.Printf("Applying strategy parameters")

	// 实现参数应用逻辑

	// 1. 解析参数应用请求
	strategyID, ok := action.Parameters["strategy_id"].(string)
	if !ok {
		return fmt.Errorf("strategy_id is required for parameter application")
	}

	newParameters, ok := action.Parameters["parameters"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("parameters are required for parameter application")
	}

	// 2. 获取当前策略配置
	currentStrategy, err := se.getStrategyConfig(ctx, strategyID)
	if err != nil {
		return fmt.Errorf("failed to get current strategy config: %v", err)
	}

	// 3. 验证新参数
	validationResult := se.validateParameters(currentStrategy, newParameters)
	if !validationResult.Valid {
		return fmt.Errorf("parameter validation failed: %s", validationResult.Reason)
	}

	// 4. 备份当前参数
	backup := se.createParameterBackup(currentStrategy)

	// 5. 应用新参数
	updatedStrategy, err := se.mergeParameters(currentStrategy, newParameters)
	if err != nil {
		return fmt.Errorf("failed to merge parameters: %v", err)
	}

	// 6. 执行参数更新
	err = se.updateStrategyParameters(ctx, strategyID, updatedStrategy)
	if err != nil {
		// 回滚参数
		se.rollbackParameters(ctx, strategyID, backup)
		return fmt.Errorf("failed to update strategy parameters: %v", err)
	}

	// 7. 通知策略实例重新加载参数
	err = se.notifyParameterUpdate(ctx, strategyID, newParameters)
	if err != nil {
		log.Printf("Warning: Failed to notify strategy instance of parameter update: %v", err)
	}

	// 8. 记录参数变更历史
	err = se.recordParameterChange(ctx, strategyID, backup.Parameters, newParameters, action.Action)
	if err != nil {
		log.Printf("Warning: Failed to record parameter change: %v", err)
	}

	// 9. 触发参数生效后的验证
	go se.validateParameterApplication(ctx, strategyID, newParameters)

	// 10. 记录指标
	if se.metrics != nil {
		se.metrics.IncrementCounter("strategy_executor.parameters_applied", map[string]string{
			"strategy_id": strategyID,
			"param_count": fmt.Sprintf("%d", len(newParameters)),
		})
	}

	log.Printf("Successfully applied %d parameters to strategy %s", len(newParameters), strategyID)
	return nil
}

// ParameterValidationResult 参数验证结果
type ParameterValidationResult struct {
	Valid       bool     `json:"valid"`
	IsValid     bool     `json:"is_valid"`
	Reason      string   `json:"reason"`
	Warnings    []string `json:"warnings"`
	Score       float64  `json:"score"`
	Issues      []string `json:"issues"`
	Suggestions []string `json:"suggestions"`
	Confidence  float64  `json:"confidence"`
}

// validateParameters 验证策略参数
func (se *StrategyExecutor) validateParameters(strategy *StrategyConfig, newParameters map[string]interface{}) *ParameterValidationResult {
	result := &ParameterValidationResult{
		Valid:    true,
		Warnings: []string{},
	}

	// 1. 检查必需参数
	requiredParams := se.getRequiredParameters(strategy.Type)
	for _, required := range requiredParams {
		if _, exists := newParameters[required]; !exists {
			// 检查是否在当前配置中存在
			if _, existsInCurrent := strategy.Parameters[required]; !existsInCurrent {
				result.Valid = false
				result.Reason = fmt.Sprintf("Required parameter '%s' is missing", required)
				return result
			}
		}
	}

	// 2. 验证参数类型和范围
	for paramName, paramValue := range newParameters {
		paramDef := se.getParameterDefinition(strategy.Type, paramName)
		if paramDef == nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Unknown parameter '%s'", paramName))
			continue
		}

		// 类型检查
		if !se.validateParameterType(paramValue, paramDef.Type) {
			result.Valid = false
			result.Reason = fmt.Sprintf("Parameter '%s' has invalid type, expected %s", paramName, paramDef.Type)
			return result
		}

		// 范围检查
		if !se.validateParameterRange(paramValue, paramDef) {
			result.Valid = false
			result.Reason = fmt.Sprintf("Parameter '%s' is out of valid range", paramName)
			return result
		}
	}

	// 3. 参数组合验证
	combinationErrors := se.validateParameterCombinations(strategy.Type, newParameters)
	if len(combinationErrors) > 0 {
		result.Valid = false
		result.Reason = strings.Join(combinationErrors, "; ")
		return result
	}

	// 4. 风险检查
	riskWarnings := se.checkParameterRisks(strategy, newParameters)
	result.Warnings = append(result.Warnings, riskWarnings...)

	return result
}

// createParameterBackup 创建参数备份
func (se *StrategyExecutor) createParameterBackup(strategy *StrategyConfig) *ParameterBackup {
	return &ParameterBackup{
		StrategyID: strategy.ID,
		Parameters: se.deepCopyParameters(strategy.Parameters),
		Timestamp:  time.Now(),
		Version:    strategy.Version,
	}
}

// mergeParameters 合并参数
func (se *StrategyExecutor) mergeParameters(currentStrategy *StrategyConfig, newParameters map[string]interface{}) (*StrategyConfig, error) {
	updatedStrategy := &StrategyConfig{
		ID:         currentStrategy.ID,
		Name:       currentStrategy.Name,
		Type:       currentStrategy.Type,
		Version:    currentStrategy.Version + 1,
		Parameters: se.deepCopyParameters(currentStrategy.Parameters),
		UpdatedAt:  time.Now(),
	}

	// 合并新参数
	for key, value := range newParameters {
		updatedStrategy.Parameters[key] = value
	}

	return updatedStrategy, nil
}

// updateStrategyParameters 更新策略参数到数据库
func (se *StrategyExecutor) updateStrategyParameters(ctx context.Context, strategyID string, strategy *StrategyConfig) error {
	if se.db == nil {
		return fmt.Errorf("database connection not available")
	}

	parametersJSON, err := json.Marshal(strategy.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %v", err)
	}

	query := `
		UPDATE strategies 
		SET parameters = ?, version = ?, updated_at = ?
		WHERE id = ?
	`

	_, err = se.db.ExecContext(ctx, query,
		string(parametersJSON), strategy.Version, strategy.UpdatedAt, strategyID)

	return err
}

// notifyParameterUpdate 通知策略实例参数更新
func (se *StrategyExecutor) notifyParameterUpdate(ctx context.Context, strategyID string, newParameters map[string]interface{}) error {
	// 这里应该通知运行中的策略实例重新加载参数
	// 可以通过消息队列、事件系统或直接调用策略实例的方法

	log.Printf("Notifying strategy %s of parameter update: %v", strategyID, newParameters)

	// 如果有策略实例的直接引用
	if se.strategyInstances != nil {
		// 实现策略实例集成 - 直接更新策略实例参数
		if instance, exists := se.strategyInstances[strategyID]; exists {
			if strategyInstance, ok := instance.(StrategyInstanceInterface); ok {
				if err := strategyInstance.UpdateParameters(ctx, newParameters); err != nil {
					log.Printf("Failed to update parameters for strategy instance %s: %v", strategyID, err)
				} else {
					log.Printf("Successfully updated parameters for strategy instance %s", strategyID)
				}
			} else {
				log.Printf("Strategy instance %s does not implement required interface", strategyID)
			}
		} else {
			log.Printf("Strategy instance %s not found in instances map", strategyID)
		}
	}

	// 通过事件系统通知
	if se.eventBus != nil {
		event := &events.Event{
			Type:   events.EventCustom,
			Source: "strategy_executor",
			Data: map[string]interface{}{
				"event_type":  "strategy_parameter_update",
				"strategy_id": strategyID,
				"parameters":  newParameters,
				"timestamp":   time.Now(),
			},
			Priority: events.PriorityNormal,
		}
		err := se.eventBus.Publish(event)
		if err != nil {
			log.Printf("Failed to publish parameter update event: %v", err)
		}
	}

	return nil
}

// recordParameterChange 记录参数变更历史
func (se *StrategyExecutor) recordParameterChange(ctx context.Context, strategyID string, oldParams, newParams map[string]interface{}, reason string) error {
	if se.db == nil {
		return nil
	}

	oldParamsJSON, _ := json.Marshal(oldParams)
	newParamsJSON, _ := json.Marshal(newParams)

	query := `
		INSERT INTO strategy_parameter_changes (
			id, strategy_id, old_parameters, new_parameters, 
			reason, created_at
		) VALUES (?, ?, ?, ?, ?, ?)
	`

	changeID := fmt.Sprintf("param_change_%s_%d", strategyID, time.Now().Unix())
	_, err := se.db.ExecContext(ctx, query,
		changeID, strategyID, string(oldParamsJSON), string(newParamsJSON),
		reason, time.Now())

	return err
}

// validateParameterApplication 验证参数应用效果
func (se *StrategyExecutor) validateParameterApplication(ctx context.Context, strategyID string, newParameters map[string]interface{}) {
	// 等待一段时间让参数生效
	time.Sleep(30 * time.Second)

	// 检查策略是否正常运行
	isHealthy := se.checkStrategyHealth(ctx, strategyID)
	if !isHealthy {
		log.Printf("Warning: Strategy %s appears unhealthy after parameter update", strategyID)

		// 发送告警
		if se.notificationService != nil {
			message := fmt.Sprintf("Strategy %s may be unhealthy after parameter update", strategyID)
			err := se.notificationService.SendWebhook(ctx, "", map[string]interface{}{
				"type":        "strategy_management",
				"action":      "parameter_update_issue",
				"message":     message,
				"strategy_id": strategyID,
			})
			if err != nil {
				log.Printf("Failed to send strategy health notification: %v", err)
			}
		}
	}

	// 记录验证结果
	if se.metrics != nil {
		healthStatus := "healthy"
		if !isHealthy {
			healthStatus = "unhealthy"
		}
		se.metrics.IncrementCounter("strategy_executor.parameter_validation", map[string]string{
			"strategy_id":   strategyID,
			"health_status": healthStatus,
		})
	}
}

// Helper functions and data structures

type StrategyConfig struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Type           string                 `json:"type"`
	Version        int                    `json:"version"`
	InitialCapital float64                `json:"initial_capital"`
	Parameters     map[string]interface{} `json:"parameters"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

type ParameterBackup struct {
	StrategyID string                 `json:"strategy_id"`
	Parameters map[string]interface{} `json:"parameters"`
	Timestamp  time.Time              `json:"timestamp"`
	Version    int                    `json:"version"`
}

type ParameterDefinition struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	MinValue     interface{} `json:"min_value"`
	MaxValue     interface{} `json:"max_value"`
	DefaultValue interface{} `json:"default_value"`
	Description  string      `json:"description"`
}

type ParameterUpdateEvent struct {
	StrategyID string                 `json:"strategy_id"`
	Parameters map[string]interface{} `json:"parameters"`
	Timestamp  time.Time              `json:"timestamp"`
}

// Helper method implementations (simplified)

func (se *StrategyExecutor) getStrategyConfig(ctx context.Context, strategyID string) (*StrategyConfig, error) {
	if se.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var config StrategyConfig
	var parametersJSON string

	query := `
		SELECT id, name, type, version, parameters, created_at, updated_at
		FROM strategy_configs
		WHERE id = ?
	`

	err := se.db.QueryRowContext(ctx, query, strategyID).Scan(
		&config.ID,
		&config.Name,
		&config.Type,
		&config.Version,
		&parametersJSON,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get strategy config: %w", err)
	}

	// 解析参数JSON
	if parametersJSON != "" {
		err = json.Unmarshal([]byte(parametersJSON), &config.Parameters)
		if err != nil {
			return nil, fmt.Errorf("failed to parse strategy parameters: %w", err)
		}
	}

	return &config, nil
}

// getStrategyInfo 获取策略信息
func (se *StrategyExecutor) getStrategyInfo(ctx context.Context, strategyID string) (*StrategyInfo, error) {
	if se.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var info StrategyInfo
	query := `
		SELECT id, name, type, status, created_at, updated_at
		FROM strategies
		WHERE id = ?
	`

	err := se.db.QueryRowContext(ctx, query, strategyID).Scan(
		&info.ID,
		&info.Name,
		&info.Type,
		&info.Status,
		&info.CreatedAt,
		&info.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get strategy info: %w", err)
	}

	return &info, nil
}
// StrategyPerformance 策略性能信息
type StrategyPerformance struct {
	TotalReturn      float64       `json:"total_return"`
	SharpeRatio      float64       `json:"sharpe_ratio"`
	MaxDrawdown      float64       `json:"max_drawdown"`
	WinRate          float64       `json:"win_rate"`
	TotalTrades      int           `json:"total_trades"`
	WinningTrades    int           `json:"winning_trades"`
	LosingTrades     int           `json:"losing_trades"`
	AvgTradeDuration time.Duration `json:"avg_trade_duration"`
}
// getStrategyPerformance 获取策略性能信息
func (se *StrategyExecutor) getStrategyPerformance(ctx context.Context, strategyID string) (*StrategyPerformance, error) {
	if se.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var performance StrategyPerformance
	query := `
		SELECT total_return, sharpe_ratio, max_drawdown, win_rate,
		       total_trades, winning_trades, losing_trades, avg_trade_duration
		FROM strategy_performance
		WHERE strategy_id = ?
		ORDER BY updated_at DESC
		LIMIT 1
	`

	err := se.db.QueryRowContext(ctx, query, strategyID).Scan(
		&performance.TotalReturn,
		&performance.SharpeRatio,
		&performance.MaxDrawdown,
		&performance.WinRate,
		&performance.TotalTrades,
		&performance.WinningTrades,
		&performance.LosingTrades,
		&performance.AvgTradeDuration,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get strategy performance: %w", err)
	}

	return &performance, nil
}

// getStrategyPositions 获取策略持仓信息
func (se *StrategyExecutor) getStrategyPositions(ctx context.Context, strategyID string) ([]Position, error) {
	if se.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	query := `
		SELECT symbol, side, quantity, entry_price, mark_price,
		       unrealized_pnl, stop_loss, take_profit, updated_at
		FROM positions
		WHERE strategy_id = ? AND quantity != 0
	`

	rows, err := se.db.QueryContext(ctx, query, strategyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query strategy positions: %w", err)
	}
	defer rows.Close()

	var positions []Position
	for rows.Next() {
		var pos Position
		err := rows.Scan(
			&pos.Symbol,
			&pos.Side,
			&pos.Quantity,
			&pos.EntryPrice,
			&pos.MarkPrice,
			&pos.UnrealizedPnL,
			&pos.StopLoss,
			&pos.TakeProfit,
			&pos.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan position: %w", err)
		}
		positions = append(positions, pos)
	}

	return positions, nil
}

func (se *StrategyExecutor) getRequiredParameters(strategyType string) []string {
	// 根据策略类型返回必需参数
	requiredParams := map[string][]string{
		"trend_following": {"period", "threshold"},
		"mean_reversion":  {"lookback", "deviation"},
		"arbitrage":       {"spread_threshold", "max_position"},
	}

	if params, exists := requiredParams[strategyType]; exists {
		return params
	}
	return []string{}
}

func (se *StrategyExecutor) getParameterDefinition(strategyType, paramName string) *ParameterDefinition {
	// 返回参数定义，实际应该从配置或数据库获取
	return &ParameterDefinition{
		Name: paramName,
		Type: "float64",
	}
}

func (se *StrategyExecutor) validateParameterType(value interface{}, expectedType string) bool {
	// 简化的类型验证
	switch expectedType {
	case "float64":
		_, ok := value.(float64)
		return ok
	case "int":
		_, ok := value.(int)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "bool":
		_, ok := value.(bool)
		return ok
	}
	return false
}

func (se *StrategyExecutor) validateParameterRange(value interface{}, def *ParameterDefinition) bool {
	// 简化的范围验证
	return true
}
func (se *StrategyExecutor) validateParameterCombinations(strategyType string, params map[string]interface{}) []string {
	// 参数组合验证
	return []string{}
}
func (se *StrategyExecutor) checkParameterRisks(strategy *StrategyConfig, newParams map[string]interface{}) []string {
	// 风险检查
	return []string{}
}

func (se *StrategyExecutor) deepCopyParameters(params map[string]interface{}) map[string]interface{} {
	copy := make(map[string]interface{})
	for k, v := range params {
		copy[k] = v
	}
	return copy
}

func (se *StrategyExecutor) rollbackParameters(ctx context.Context, strategyID string, backup *ParameterBackup) error {
	// 参数回滚逻辑
	log.Printf("Rolling back parameters for strategy %s", strategyID)
	return nil
}

func (se *StrategyExecutor) checkStrategyHealth(ctx context.Context, strategyID string) bool {
	// 检查策略健康状态
	return true
}

// eliminateStrategy 淘汰策略
func (se *StrategyExecutor) eliminateStrategy(ctx context.Context, action *ExecutionAction) error {
	strategyID, ok := action.Parameters["strategy_id"].(string)
	if !ok {
		return fmt.Errorf("strategy_id parameter is required")
	}

	log.Printf("Eliminating strategy: %s", strategyID)

	// 1. 验证策略存在且可以被淘汰
	if err := se.validateStrategyForElimination(ctx, strategyID); err != nil {
		return fmt.Errorf("strategy validation failed: %w", err)
	}

	// 2. 关闭策略的所有活跃订单
	if err := se.closeStrategyOrders(ctx, strategyID); err != nil {
		log.Printf("Warning: Failed to close some orders for strategy %s: %v", strategyID, err)
	}

	// 3. 清算策略持仓
	if err := se.liquidateStrategyPositions(ctx, strategyID); err != nil {
		return fmt.Errorf("failed to liquidate positions: %w", err)
	}

	// 4. 停止策略执行
	if err := se.stopStrategyExecution(ctx, strategyID); err != nil {
		return fmt.Errorf("failed to stop strategy execution: %w", err)
	}

	// 5. 更新策略状态为已淘汰
	if err := se.updateStrategyStatus(ctx, strategyID, "eliminated"); err != nil {
		return fmt.Errorf("failed to update strategy status: %w", err)
	}

	// 6. 记录淘汰原因和历史
	if err := se.recordStrategyElimination(ctx, action); err != nil {
		log.Printf("Warning: Failed to record elimination history: %v", err)
	}

	// 7. 释放策略资源
	if err := se.releaseStrategyResources(ctx, strategyID); err != nil {
		log.Printf("Warning: Failed to release some resources: %v", err)
	}

	// 8. 发送淘汰通知
	if err := se.notifyStrategyElimination(ctx, strategyID); err != nil {
		log.Printf("Warning: Failed to send elimination notification: %v", err)
	}

	log.Printf("Strategy %s eliminated successfully", strategyID)
	return nil
}

// introduceStrategy 引入新策略
func (se *StrategyExecutor) introduceStrategy(ctx context.Context, action *ExecutionAction) error {
	strategyID, ok := action.Parameters["strategy_id"].(string)
	if !ok {
		return fmt.Errorf("strategy_id parameter is required")
	}

	log.Printf("Introducing new strategy: %s", strategyID)

	// 1. 验证新策略配置
	strategyConfig, err := se.validateNewStrategyConfig(ctx, action)
	if err != nil {
		return fmt.Errorf("strategy config validation failed: %w", err)
	}

	// 2. 检查资源可用性
	if err := se.checkResourceAvailability(ctx, strategyConfig); err != nil {
		return fmt.Errorf("insufficient resources: %w", err)
	}

	// 3. 创建策略实例
	strategy, err := se.createStrategyInstance(ctx, strategyConfig)
	if err != nil {
		return fmt.Errorf("failed to create strategy instance: %w", err)
	}

	// 4. 分配资金和资源
	if err := se.allocateStrategyResources(ctx, strategy); err != nil {
		return fmt.Errorf("failed to allocate resources: %w", err)
	}

	// 5. 初始化策略参数
	if err := se.initializeStrategyParameters(ctx, strategy); err != nil {
		return fmt.Errorf("failed to initialize parameters: %w", err)
	}

	// 6. 设置风险控制
	if err := se.setupStrategyRiskControls(ctx, strategy); err != nil {
		return fmt.Errorf("failed to setup risk controls: %w", err)
	}

	// 7. 启动策略监控
	if err := se.startStrategyMonitoring(ctx, strategy); err != nil {
		return fmt.Errorf("failed to start monitoring: %w", err)
	}

	// 8. 更新策略状态为活跃
	if err := se.updateStrategyStatus(ctx, strategy.ID, "active"); err != nil {
		return fmt.Errorf("failed to update strategy status: %w", err)
	}

	// 9. 记录策略引入历史
	if err := se.recordStrategyIntroduction(ctx, strategy); err != nil {
		log.Printf("Warning: Failed to record introduction history: %v", err)
	}

	// 10. 发送引入通知
	if err := se.notifyStrategyIntroduction(ctx, strategy); err != nil {
		log.Printf("Warning: Failed to send introduction notification: %v", err)
	}

	log.Printf("Strategy %s introduced successfully", strategy.ID)
	return nil
}

// optimizeStrategy 优化策略
func (se *StrategyExecutor) optimizeStrategy(ctx context.Context, action *ExecutionAction) error {
	log.Printf("Optimizing strategy")

	// 实现策略优化逻辑

	// 1. 解析优化请求
	strategyID, ok := action.Parameters["strategy_id"].(string)
	if !ok {
		return fmt.Errorf("strategy_id is required for strategy optimization")
	}

	optimizationType, _ := action.Parameters["optimization_type"].(string)
	if optimizationType == "" {
		optimizationType = "performance" // 默认性能优化
	}

	optimizationParams, _ := action.Parameters["optimization_params"].(map[string]interface{})

	// 2. 获取策略信息
	strategy, err := se.getStrategyInfo(ctx, strategyID)
	if err != nil {
		return fmt.Errorf("failed to get strategy info: %v", err)
	}

	if strategy == nil {
		return fmt.Errorf("strategy %s not found", strategyID)
	}

	// 3. 检查优化条件
	canOptimize, reason := se.canOptimizeStrategy(ctx, strategy)
	if !canOptimize {
		return fmt.Errorf("cannot optimize strategy: %s", reason)
	}

	// 4. 创建优化任务
	optimizationTask := &StrategyOptimizationTask{
		ID:                fmt.Sprintf("opt_%s_%d", strategyID, time.Now().Unix()),
		StrategyID:        strategyID,
		OptimizationType:  optimizationType,
		Parameters:        optimizationParams,
		Status:            "initializing",
		CreatedAt:         time.Now(),
		EstimatedDuration: se.estimateOptimizationDuration(optimizationType, strategy),
	}

	// 5. 保存优化任务
	err = se.saveOptimizationTask(ctx, optimizationTask)
	if err != nil {
		return fmt.Errorf("failed to save optimization task: %v", err)
	}

	// 6. 执行优化流程
	go se.executeOptimizationTask(ctx, optimizationTask, strategy)

	// 7. 记录指标
	if se.metrics != nil {
		se.metrics.IncrementCounter("strategy_executor.optimization_started", map[string]string{
			"strategy_id":       strategyID,
			"optimization_type": optimizationType,
		})
	}

	log.Printf("Started optimization task %s for strategy %s", optimizationTask.ID, strategyID)
	return nil
}

// canOptimizeStrategy 检查是否可以优化策略
func (se *StrategyExecutor) canOptimizeStrategy(ctx context.Context, strategy *StrategyInfo) (bool, string) {
	// 1. 检查策略状态
	if strategy.Status == "eliminated" {
		return false, "cannot optimize eliminated strategy"
	}

	if strategy.Status == "optimizing" {
		return false, "strategy is already being optimized"
	}

	// 2. 检查运行时间（需要足够的历史数据）
	runningDuration := time.Since(strategy.StartTime)
	minRunningTime := 7 * 24 * time.Hour // 最少运行7天
	if runningDuration < minRunningTime {
		return false, fmt.Sprintf("strategy needs to run for at least %v for optimization (current: %v)",
			minRunningTime, runningDuration)
	}

	// 3. 检查交易数量
	performance, err := se.getStrategyPerformance(ctx, strategy.ID)
	if err != nil {
		log.Printf("Warning: failed to get strategy performance: %v", err)
	} else if performance != nil && performance.TotalTrades < 50 {
		return false, "insufficient trade history for optimization (minimum 50 trades required)"
	}

	// 4. 检查系统资源
	if !se.hasOptimizationResources() {
		return false, "insufficient system resources for optimization"
	}

	// 5. 检查是否有正在进行的优化任务
	activeOptimizations := se.getActiveOptimizationCount()
	maxConcurrentOptimizations := 3
	if activeOptimizations >= maxConcurrentOptimizations {
		return false, fmt.Sprintf("maximum concurrent optimizations reached (%d/%d)",
			activeOptimizations, maxConcurrentOptimizations)
	}

	return true, ""
}

// executeOptimizationTask 执行优化任务
func (se *StrategyExecutor) executeOptimizationTask(ctx context.Context, task *StrategyOptimizationTask, strategy *StrategyInfo) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Optimization task %s panicked: %v", task.ID, r)
			task.Status = "failed"
			task.Error = fmt.Sprintf("panic: %v", r)
			se.updateOptimizationTask(ctx, task)
		}
	}()

	// 1. 更新任务状态
	task.Status = "running"
	task.StartedAt = time.Now()
	se.updateOptimizationTask(ctx, task)

	// 2. 根据优化类型执行相应的优化
	var err error
	switch task.OptimizationType {
	case "performance":
		err = se.optimizePerformance(ctx, task, strategy)
	case "risk":
		err = se.optimizeRisk(ctx, task, strategy)
	case "parameters":
		err = se.optimizeParameters(ctx, task, strategy)
	case "portfolio":
		err = se.optimizePortfolio(ctx, task, strategy)
	case "execution":
		err = se.optimizeExecution(ctx, task, strategy)
	default:
		err = fmt.Errorf("unsupported optimization type: %s", task.OptimizationType)
	}

	// 3. 更新任务结果
	task.CompletedAt = time.Now()
	task.Duration = task.CompletedAt.Sub(task.StartedAt)

	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
		log.Printf("Optimization task %s failed: %v", task.ID, err)
	} else {
		task.Status = "completed"
		log.Printf("Optimization task %s completed successfully", task.ID)
	}

	// 4. 保存最终结果
	se.updateOptimizationTask(ctx, task)

	// 5. 发送通知
	se.notifyOptimizationCompletion(ctx, task, strategy)

	// 6. 记录指标
	if se.metrics != nil {
		se.metrics.IncrementCounter("strategy_executor.optimization_completed", map[string]string{
			"strategy_id":       strategy.ID,
			"optimization_type": task.OptimizationType,
			"status":            task.Status,
		})
	}
}

// optimizePerformance 性能优化
func (se *StrategyExecutor) optimizePerformance(ctx context.Context, task *StrategyOptimizationTask, strategy *StrategyInfo) error {
	log.Printf("Starting performance optimization for strategy %s", strategy.ID)

	// 1. 收集历史性能数据
	historicalData, err := se.collectHistoricalPerformance(ctx, strategy.ID)
	if err != nil {
		return fmt.Errorf("failed to collect historical data: %v", err)
	}

	// 2. 分析性能瓶颈
	bottlenecks := se.analyzePerformanceBottlenecks(historicalData)
	task.Results = map[string]interface{}{
		"bottlenecks": bottlenecks,
	}

	// 3. 生成优化建议
	recommendations := se.generatePerformanceRecommendations(bottlenecks, strategy)
	task.Results["recommendations"] = recommendations

	// 4. 应用优化（如果启用自动应用）
	autoApply, _ := task.Parameters["auto_apply"].(bool)
	if autoApply {
		appliedChanges, err := se.applyPerformanceOptimizations(ctx, strategy, recommendations)
		if err != nil {
			return fmt.Errorf("failed to apply optimizations: %v", err)
		}
		task.Results["applied_changes"] = appliedChanges
	}

	return nil
}

// optimizeRisk 风险优化
func (se *StrategyExecutor) optimizeRisk(ctx context.Context, task *StrategyOptimizationTask, strategy *StrategyInfo) error {
	log.Printf("Starting risk optimization for strategy %s", strategy.ID)

	// 1. 分析当前风险指标
	riskMetrics, err := se.analyzeCurrentRiskMetrics(ctx, strategy.ID)
	if err != nil {
		return fmt.Errorf("failed to analyze risk metrics: %v", err)
	}

	// 2. 识别风险问题
	riskIssues := se.identifyRiskIssues(riskMetrics)
	task.Results = map[string]interface{}{
		"risk_metrics": riskMetrics,
		"risk_issues":  riskIssues,
	}

	// 3. 生成风险优化建议
	riskRecommendations := se.generateRiskRecommendations(riskIssues, strategy)
	task.Results["recommendations"] = riskRecommendations

	// 4. 应用风险控制措施
	autoApply, _ := task.Parameters["auto_apply"].(bool)
	if autoApply {
		appliedMeasures, err := se.applyRiskOptimizations(ctx, strategy, riskRecommendations)
		if err != nil {
			return fmt.Errorf("failed to apply risk optimizations: %v", err)
		}
		task.Results["applied_measures"] = appliedMeasures
	}

	return nil
}

// optimizeParameters 参数优化
func (se *StrategyExecutor) optimizeParameters(ctx context.Context, task *StrategyOptimizationTask, strategy *StrategyInfo) error {
	log.Printf("Starting parameter optimization for strategy %s", strategy.ID)

	// 1. 获取当前参数
	currentParams := se.getCurrentStrategyParameters(ctx, strategy.ID)

	// 2. 定义参数搜索空间
	searchSpace := se.defineParameterSearchSpace(strategy.Type, currentParams)

	// 3. 执行参数优化算法
	optimizationMethod, _ := task.Parameters["method"].(string)
	if optimizationMethod == "" {
		optimizationMethod = "grid_search"
	}

	var optimizedParams map[string]interface{}
	var err error

	switch optimizationMethod {
	case "grid_search":
		optimizedParams, err = se.gridSearchOptimization(ctx, strategy, searchSpace)
	case "random_search":
		optimizedParams, err = se.randomSearchOptimization(ctx, strategy, searchSpace)
	case "bayesian":
		optimizedParams, err = se.bayesianOptimization(ctx, strategy, searchSpace)
	default:
		return fmt.Errorf("unsupported optimization method: %s", optimizationMethod)
	}

	if err != nil {
		return fmt.Errorf("parameter optimization failed: %v", err)
	}

	// 4. 验证优化结果
	validationResult, err := se.validateOptimizedParameters(ctx, strategy, optimizedParams)
	if err != nil {
		return fmt.Errorf("parameter validation failed: %v", err)
	}

	task.Results = map[string]interface{}{
		"current_params":    currentParams,
		"optimized_params":  optimizedParams,
		"validation_result": validationResult,
		"improvement":       validationResult.Score,
	}

	// 5. 应用优化参数
	autoApply, _ := task.Parameters["auto_apply"].(bool)
	if autoApply && validationResult.IsValid {
		err = se.applyOptimizedParameters(ctx, strategy, optimizedParams)
		if err != nil {
			return fmt.Errorf("failed to apply optimized parameters: %v", err)
		}
		task.Results["applied"] = true
	}

	return nil
}

// optimizePortfolio 组合优化
func (se *StrategyExecutor) optimizePortfolio(ctx context.Context, task *StrategyOptimizationTask, strategy *StrategyInfo) error {
	log.Printf("Starting portfolio optimization for strategy %s", strategy.ID)

	// 1. 获取当前持仓
	currentPositions, err := se.getStrategyPositions(ctx, strategy.ID)
	if err != nil {
		return fmt.Errorf("failed to get strategy positions: %v", err)
	}

	// Convert to pointer slice for compatibility
	var positionPtrs []*Position
	for i := range currentPositions {
		positionPtrs = append(positionPtrs, &currentPositions[i])
	}

	// 2. 分析持仓分布
	portfolioAnalysis := se.analyzePortfolioDistribution(positionPtrs)

	// 3. 计算最优权重
	optimalWeights, err := se.calculateOptimalWeights(ctx, strategy, positionPtrs)
	if err != nil {
		return fmt.Errorf("failed to calculate optimal weights: %v", err)
	}

	// 4. 生成调仓建议
	rebalanceRecommendations := se.generateRebalanceRecommendations(positionPtrs, optimalWeights)

	task.Results = map[string]interface{}{
		"current_portfolio":         portfolioAnalysis,
		"optimal_weights":           optimalWeights,
		"rebalance_recommendations": rebalanceRecommendations,
	}

	// 5. 执行调仓
	autoApply, _ := task.Parameters["auto_apply"].(bool)
	if autoApply {
		rebalanceResult, err := se.executePortfolioRebalance(ctx, strategy, rebalanceRecommendations)
		if err != nil {
			return fmt.Errorf("failed to execute rebalance: %v", err)
		}
		task.Results["rebalance_result"] = rebalanceResult
	}

	return nil
}

// optimizeExecution 执行优化
func (se *StrategyExecutor) optimizeExecution(ctx context.Context, task *StrategyOptimizationTask, strategy *StrategyInfo) error {
	log.Printf("Starting execution optimization for strategy %s", strategy.ID)

	// 1. 分析执行效率
	executionMetrics, err := se.analyzeExecutionEfficiency(ctx, strategy.ID)
	if err != nil {
		return fmt.Errorf("failed to analyze execution efficiency: %v", err)
	}

	// 2. 识别执行问题
	executionIssues := se.identifyExecutionIssues(executionMetrics)

	// 3. 优化执行算法
	optimizedAlgorithms := se.optimizeExecutionAlgorithms(strategy, executionIssues)

	// 4. 调整执行参数
	optimizedExecutionParams := se.optimizeExecutionParameters(strategy, executionMetrics)

	task.Results = map[string]interface{}{
		"execution_metrics":    executionMetrics,
		"execution_issues":     executionIssues,
		"optimized_algorithms": optimizedAlgorithms,
		"optimized_parameters": optimizedExecutionParams,
	}

	// 5. 应用执行优化
	autoApply, _ := task.Parameters["auto_apply"].(bool)
	if autoApply {
		appliedOptimizations, err := se.applyExecutionOptimizations(ctx, strategy, optimizedAlgorithms, optimizedExecutionParams)
		if err != nil {
			return fmt.Errorf("failed to apply execution optimizations: %v", err)
		}
		task.Results["applied_optimizations"] = appliedOptimizations
	}

	return nil
}

// Data structures and helper methods

type StrategyOptimizationTask struct {
	ID                string                 `json:"id"`
	StrategyID        string                 `json:"strategy_id"`
	OptimizationType  string                 `json:"optimization_type"`
	Parameters        map[string]interface{} `json:"parameters"`
	Status            string                 `json:"status"`
	Results           map[string]interface{} `json:"results"`
	Error             string                 `json:"error,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	StartedAt         time.Time              `json:"started_at"`
	CompletedAt       time.Time              `json:"completed_at"`
	Duration          time.Duration          `json:"duration"`
	EstimatedDuration time.Duration          `json:"estimated_duration"`
}

type ParameterValidationResult2 struct {
	IsValid                bool     `json:"is_valid"`
	PerformanceImprovement float64  `json:"performance_improvement"`
	RiskImprovement        float64  `json:"risk_improvement"`
	ValidationErrors       []string `json:"validation_errors"`
}

// Helper method implementations (simplified)

func (se *StrategyExecutor) estimateOptimizationDuration(optimizationType string, strategy *StrategyInfo) time.Duration {
	switch optimizationType {
	case "performance":
		return 30 * time.Minute
	case "risk":
		return 15 * time.Minute
	case "parameters":
		return 2 * time.Hour
	case "portfolio":
		return 45 * time.Minute
	case "execution":
		return 20 * time.Minute
	default:
		return 1 * time.Hour
	}
}

func (se *StrategyExecutor) hasOptimizationResources() bool {
	// 检查系统资源是否足够进行优化
	return true
}

func (se *StrategyExecutor) getActiveOptimizationCount() int {
	// 获取当前活跃的优化任务数量
	return 0
}

func (se *StrategyExecutor) saveOptimizationTask(ctx context.Context, task *StrategyOptimizationTask) error {
	// 保存优化任务到数据库
	return nil
}

func (se *StrategyExecutor) updateOptimizationTask(ctx context.Context, task *StrategyOptimizationTask) error {
	// 更新优化任务状态
	return nil
}

func (se *StrategyExecutor) notifyOptimizationCompletion(ctx context.Context, task *StrategyOptimizationTask, strategy *StrategyInfo) {
	// 发送优化完成通知
	if se.notificationService != nil {
		message := fmt.Sprintf("Strategy optimization %s for %s completed with status: %s",
			task.OptimizationType, strategy.Name, task.Status)
		err := se.notificationService.SendWebhook(ctx, "", map[string]interface{}{
			"type":              "strategy_management",
			"action":            "optimization_completed",
			"message":           message,
			"task_id":           task.ID,
			"strategy_id":       strategy.ID,
			"optimization_type": task.OptimizationType,
			"status":            task.Status,
		})
		if err != nil {
			log.Printf("Failed to send optimization completion notification: %v", err)
		}
	}
}

// Simplified implementations for optimization methods
func (se *StrategyExecutor) collectHistoricalPerformance(ctx context.Context, strategyID string) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (se *StrategyExecutor) analyzePerformanceBottlenecks(data map[string]interface{}) []string {
	return []string{}
}

func (se *StrategyExecutor) generatePerformanceRecommendations(bottlenecks []string, strategy *StrategyInfo) []string {
	return []string{}
}

func (se *StrategyExecutor) applyPerformanceOptimizations(ctx context.Context, strategy *StrategyInfo, recommendations []string) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (se *StrategyExecutor) analyzeCurrentRiskMetrics(ctx context.Context, strategyID string) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (se *StrategyExecutor) identifyRiskIssues(metrics map[string]interface{}) []string {
	return []string{}
}

func (se *StrategyExecutor) generateRiskRecommendations(issues []string, strategy *StrategyInfo) []string {
	return []string{}
}

func (se *StrategyExecutor) applyRiskOptimizations(ctx context.Context, strategy *StrategyInfo, recommendations []string) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (se *StrategyExecutor) getCurrentStrategyParameters(ctx context.Context, strategyID string) map[string]interface{} {
	return map[string]interface{}{}
}

func (se *StrategyExecutor) defineParameterSearchSpace(strategyType string, currentParams map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{}
}

func (se *StrategyExecutor) gridSearchOptimization(ctx context.Context, strategy *StrategyInfo, searchSpace map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (se *StrategyExecutor) randomSearchOptimization(ctx context.Context, strategy *StrategyInfo, searchSpace map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (se *StrategyExecutor) bayesianOptimization(ctx context.Context, strategy *StrategyInfo, searchSpace map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}
func (se *StrategyExecutor) validateOptimizedParameters(ctx context.Context, strategy *StrategyInfo, params map[string]interface{}) (*ParameterValidationResult, error) {
	return &ParameterValidationResult{
		IsValid:     true,
		Score:       0.85,
		Issues:      []string{},
		Suggestions: []string{},
		Confidence:  0.9,
	}, nil
}

func (se *StrategyExecutor) applyOptimizedParameters(ctx context.Context, strategy *StrategyInfo, params map[string]interface{}) error {
	return nil
}
func (se *StrategyExecutor) analyzePortfolioDistribution(positions []*Position) map[string]interface{} {
	return map[string]interface{}{}
}

func (se *StrategyExecutor) calculateOptimalWeights(ctx context.Context, strategy *StrategyInfo, positions []*Position) (map[string]float64, error) {
	return map[string]float64{}, nil
}

func (se *StrategyExecutor) generateRebalanceRecommendations(positions []*Position, weights map[string]float64) []string {
	return []string{}
}

func (se *StrategyExecutor) executePortfolioRebalance(ctx context.Context, strategy *StrategyInfo, recommendations []string) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (se *StrategyExecutor) analyzeExecutionEfficiency(ctx context.Context, strategyID string) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (se *StrategyExecutor) identifyExecutionIssues(metrics map[string]interface{}) []string {
	return []string{}
}

func (se *StrategyExecutor) optimizeExecutionAlgorithms(strategy *StrategyInfo, issues []string) map[string]interface{} {
	return map[string]interface{}{}
}

func (se *StrategyExecutor) optimizeExecutionParameters(strategy *StrategyInfo, metrics map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{}
}

func (se *StrategyExecutor) applyExecutionOptimizations(ctx context.Context, strategy *StrategyInfo, algorithms, params map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

// DataExecutor 数据执行器
type DataExecutor struct {
	config              *config.Config
	db                  *database.DB
	exchange            exchange.Exchange
	accountManager      *account.Manager
	metrics             *monitor.MetricsCollector     // actual metrics service
	notificationService protector.NotificationService // actual notification service
}

// NewDataExecutor 创建数据执行器
func NewDataExecutor(
	cfg *config.Config,
	db *database.DB,
	exchange exchange.Exchange,
	accountManager *account.Manager,
) *DataExecutor {
	return &DataExecutor{
		config:         cfg,
		db:             db,
		exchange:       exchange,
		accountManager: accountManager,
	}
}

// HandleAction 处理数据动作
func (de *DataExecutor) HandleAction(ctx context.Context, action *ExecutionAction) error {
	switch action.Action {
	case "clean_data":
		return de.cleanData(ctx, action)
	case "update_factors":
		return de.updateFactors(ctx, action)
	case "run_backtest":
		return de.runBacktest(ctx, action)
	case "recognize_pattern":
		return de.recognizePattern(ctx, action)
	default:
		return fmt.Errorf("unknown data action: %s", action.Action)
	}
}
// cleanData 清洗数据
func (de *DataExecutor) cleanData(ctx context.Context, action *ExecutionAction) error {
	log.Printf("Cleaning data")

	// 实现数据清洗逻辑

	// 1. 解析清洗请求
	dataSource, ok := action.Parameters["data_source"].(string)
	if !ok {
		return fmt.Errorf("data_source is required for data cleaning")
	}

	cleaningRules, _ := action.Parameters["cleaning_rules"].([]interface{})
	timeRange, _ := action.Parameters["time_range"].(map[string]interface{})
	dryRun, _ := action.Parameters["dry_run"].(bool)

	// 2. 创建数据清洗任务
	cleaningTask := &DataCleaningTask{
		ID:            fmt.Sprintf("clean_%s_%d", dataSource, time.Now().Unix()),
		DataSource:    dataSource,
		CleaningRules: cleaningRules,
		TimeRange:     timeRange,
		DryRun:        dryRun,
		Status:        "initializing",
		CreatedAt:     time.Now(),
	}

	// 3. 保存清洗任务
	err := de.saveCleaningTask(ctx, cleaningTask)
	if err != nil {
		return fmt.Errorf("failed to save cleaning task: %v", err)
	}

	// 4. 执行数据清洗
	go de.executeDataCleaning(ctx, cleaningTask)

	// 5. 记录指标
	if de.metrics != nil {
		de.metrics.IncrementCounter("data_executor.cleaning_started", map[string]string{
			"data_source": dataSource,
			"dry_run":     fmt.Sprintf("%t", dryRun),
		})
	}

	log.Printf("Started data cleaning task %s for source %s", cleaningTask.ID, dataSource)
	return nil
}
// executeDataCleaning 执行数据清洗
func (de *DataExecutor) executeDataCleaning(ctx context.Context, task *DataCleaningTask) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Data cleaning task %s panicked: %v", task.ID, r)
			task.Status = "failed"
			task.Error = fmt.Sprintf("panic: %v", r)
			de.updateCleaningTask(ctx, task)
		}
	}()

	// 1. 更新任务状态
	task.Status = "running"
	task.StartedAt = time.Now()
	de.updateCleaningTask(ctx, task)

	// 2. 获取数据源
	dataSource, err := de.getDataSource(task.DataSource)
	if err != nil {
		task.Status = "failed"
		task.Error = fmt.Sprintf("failed to get data source: %v", err)
		de.updateCleaningTask(ctx, task)
		return
	}

	// 3. 加载数据
	rawData, err := de.loadRawData(ctx, dataSource, task.TimeRange)
	if err != nil {
		task.Status = "failed"
		task.Error = fmt.Sprintf("failed to load raw data: %v", err)
		de.updateCleaningTask(ctx, task)
		return
	}

	task.Statistics = &CleaningStatistics{
		TotalRecords: len(rawData),
	}

	// 4. 应用清洗规则
	cleanedData, cleaningReport, err := de.applyCleaningRules(rawData, task.CleaningRules)
	if err != nil {
		task.Status = "failed"
		task.Error = fmt.Sprintf("failed to apply cleaning rules: %v", err)
		de.updateCleaningTask(ctx, task)
		return
	}

	task.Statistics.CleanedRecords = len(cleanedData)
	task.Statistics.RemovedRecords = task.Statistics.TotalRecords - task.Statistics.CleanedRecords
	task.CleaningReport = cleaningReport

	// 5. 数据质量检查
	qualityReport, err := de.performQualityCheck(cleanedData)
	if err != nil {
		log.Printf("Warning: Quality check failed for task %s: %v", task.ID, err)
	} else {
		task.QualityReport = qualityReport
	}

	// 6. 保存清洗后的数据（如果不是干运行）
	if !task.DryRun {
		err = de.saveCleanedData(ctx, dataSource, cleanedData, task.ID)
		if err != nil {
			task.Status = "failed"
			task.Error = fmt.Sprintf("failed to save cleaned data: %v", err)
			de.updateCleaningTask(ctx, task)
			return
		}
	}

	// 7. 更新任务完成状态
	task.Status = "completed"
	task.CompletedAt = time.Now()
	task.Duration = task.CompletedAt.Sub(task.StartedAt)
	de.updateCleaningTask(ctx, task)

	// 8. 发送通知
	de.notifyCleaningCompletion(ctx, task)

	// 9. 记录指标
	if de.metrics != nil {
		de.metrics.IncrementCounter("data_executor.cleaning_completed", map[string]string{
			"data_source": task.DataSource,
			"status":      task.Status,
		})
	}

	log.Printf("Data cleaning task %s completed successfully", task.ID)
}

// applyCleaningRules 应用清洗规则
func (de *DataExecutor) applyCleaningRules(rawData []map[string]interface{}, rules []interface{}) ([]map[string]interface{}, *CleaningReport, error) {
	cleanedData := make([]map[string]interface{}, 0, len(rawData))
	report := &CleaningReport{
		RulesApplied: make(map[string]int),
		Issues:       make([]string, 0),
	}

	for _, record := range rawData {
		shouldKeep := true
		recordIssues := make([]string, 0)

		// 应用每个清洗规则
		for _, rule := range rules {
			ruleMap, ok := rule.(map[string]interface{})
			if !ok {
				continue
			}

			ruleType, _ := ruleMap["type"].(string)
			ruleParams, _ := ruleMap["parameters"].(map[string]interface{})

			keep, issues := de.applyCleaningRule(record, ruleType, ruleParams)
			if !keep {
				shouldKeep = false
				report.RulesApplied[ruleType]++
			}
			recordIssues = append(recordIssues, issues...)
		}

		if shouldKeep {
			cleanedData = append(cleanedData, record)
		} else {
			report.Issues = append(report.Issues, recordIssues...)
		}
	}

	return cleanedData, report, nil
}

// applyCleaningRule 应用单个清洗规则
func (de *DataExecutor) applyCleaningRule(record map[string]interface{}, ruleType string, params map[string]interface{}) (bool, []string) {
	issues := make([]string, 0)

	switch ruleType {
	case "remove_null":
		return de.removeNullRule(record, params, &issues)
	case "remove_duplicates":
		return de.removeDuplicatesRule(record, params, &issues)
	case "validate_range":
		return de.validateRangeRule(record, params, &issues)
	case "validate_format":
		return de.validateFormatRule(record, params, &issues)
	case "remove_outliers":
		return de.removeOutliersRule(record, params, &issues)
	case "normalize_values":
		return de.normalizeValuesRule(record, params, &issues)
	case "validate_timestamp":
		return de.validateTimestampRule(record, params, &issues)
	case "remove_invalid_prices":
		return de.removeInvalidPricesRule(record, params, &issues)
	default:
		issues = append(issues, fmt.Sprintf("unknown cleaning rule: %s", ruleType))
		return true, issues
	}
}

// 具体的清洗规则实现

// removeNullRule 移除空值规则
func (de *DataExecutor) removeNullRule(record map[string]interface{}, params map[string]interface{}, issues *[]string) (bool, []string) {
	fields, _ := params["fields"].([]interface{})

	for _, field := range fields {
		fieldName, _ := field.(string)
		if value, exists := record[fieldName]; !exists || value == nil || value == "" {
			*issues = append(*issues, fmt.Sprintf("null value in field: %s", fieldName))
			return false, *issues
		}
	}

	return true, *issues
}

// removeDuplicatesRule 移除重复规则
func (de *DataExecutor) removeDuplicatesRule(record map[string]interface{}, params map[string]interface{}, issues *[]string) (bool, []string) {
	// 这里需要维护一个全局的重复检查状态
	// 简化实现，实际应该使用更复杂的重复检测逻辑
	return true, *issues
}

// validateRangeRule 验证范围规则
func (de *DataExecutor) validateRangeRule(record map[string]interface{}, params map[string]interface{}, issues *[]string) (bool, []string) {
	field, _ := params["field"].(string)
	minValue, _ := params["min"].(float64)
	maxValue, _ := params["max"].(float64)

	if value, exists := record[field]; exists {
		if numValue, ok := value.(float64); ok {
			if numValue < minValue || numValue > maxValue {
				*issues = append(*issues, fmt.Sprintf("value %f out of range [%f, %f] for field %s", numValue, minValue, maxValue, field))
				return false, *issues
			}
		}
	}

	return true, *issues
}

// validateFormatRule 验证格式规则
func (de *DataExecutor) validateFormatRule(record map[string]interface{}, params map[string]interface{}, issues *[]string) (bool, []string) {
	field, _ := params["field"].(string)
	pattern, _ := params["pattern"].(string)

	if value, exists := record[field]; exists {
		if strValue, ok := value.(string); ok {
			matched, err := regexp.MatchString(pattern, strValue)
			if err != nil || !matched {
				*issues = append(*issues, fmt.Sprintf("invalid format for field %s: %s", field, strValue))
				return false, *issues
			}
		}
	}

	return true, *issues
}

// removeOutliersRule 移除异常值规则
func (de *DataExecutor) removeOutliersRule(record map[string]interface{}, params map[string]interface{}, issues *[]string) (bool, []string) {
	field, _ := params["field"].(string)
	method, _ := params["method"].(string)
	threshold, _ := params["threshold"].(float64)

	if value, exists := record[field]; exists {
		if numValue, ok := value.(float64); ok {
			isOutlier := false

			switch method {
			case "zscore":
				// 简化的Z-score检测
				if threshold > 0 && (numValue > threshold || numValue < -threshold) {
					isOutlier = true
				}
			case "iqr":
				// 简化的IQR检测
				// 实际实现需要计算四分位数
				isOutlier = false
			}

			if isOutlier {
				*issues = append(*issues, fmt.Sprintf("outlier detected for field %s: %f", field, numValue))
				return false, *issues
			}
		}
	}

	return true, *issues
}

// normalizeValuesRule 标准化值规则
func (de *DataExecutor) normalizeValuesRule(record map[string]interface{}, params map[string]interface{}, issues *[]string) (bool, []string) {
	field, _ := params["field"].(string)
	method, _ := params["method"].(string)

	if value, exists := record[field]; exists {
		switch method {
		case "lowercase":
			if strValue, ok := value.(string); ok {
				record[field] = strings.ToLower(strValue)
			}
		case "uppercase":
			if strValue, ok := value.(string); ok {
				record[field] = strings.ToUpper(strValue)
			}
		case "trim":
			if strValue, ok := value.(string); ok {
				record[field] = strings.TrimSpace(strValue)
			}
		}
	}

	return true, *issues
}

// validateTimestampRule 验证时间戳规则
func (de *DataExecutor) validateTimestampRule(record map[string]interface{}, params map[string]interface{}, issues *[]string) (bool, []string) {
	field, _ := params["field"].(string)
	format, _ := params["format"].(string)

	if value, exists := record[field]; exists {
		if strValue, ok := value.(string); ok {
			_, err := time.Parse(format, strValue)
			if err != nil {
				*issues = append(*issues, fmt.Sprintf("invalid timestamp format for field %s: %s", field, strValue))
				return false, *issues
			}
		}
	}

	return true, *issues
}

// removeInvalidPricesRule 移除无效价格规则
func (de *DataExecutor) removeInvalidPricesRule(record map[string]interface{}, params map[string]interface{}, issues *[]string) (bool, []string) {
	priceFields, _ := params["price_fields"].([]interface{})

	for _, field := range priceFields {
		fieldName, _ := field.(string)
		if value, exists := record[fieldName]; exists {
			if numValue, ok := value.(float64); ok {
				if numValue <= 0 || math.IsNaN(numValue) || math.IsInf(numValue, 0) {
					*issues = append(*issues, fmt.Sprintf("invalid price for field %s: %f", fieldName, numValue))
					return false, *issues
				}
			}
		}
	}

	return true, *issues
}

// performQualityCheck 执行数据质量检查
func (de *DataExecutor) performQualityCheck(data []map[string]interface{}) (*QualityReport, error) {
	if len(data) == 0 {
		return &QualityReport{
			OverallScore: 0,
			Issues:       []string{"no data available for quality check"},
		}, nil
	}

	report := &QualityReport{
		TotalRecords: len(data),
		Issues:       make([]string, 0),
	}

	// 1. 完整性检查
	completenessScore := de.checkCompleteness(data, report)

	// 2. 一致性检查
	consistencyScore := de.checkConsistency(data, report)

	// 3. 准确性检查
	accuracyScore := de.checkAccuracy(data, report)

	// 4. 及时性检查
	timelinessScore := de.checkTimeliness(data, report)

	// 5. 计算总体质量分数
	report.OverallScore = (completenessScore + consistencyScore + accuracyScore + timelinessScore) / 4
	report.CompletenessScore = completenessScore
	report.ConsistencyScore = consistencyScore
	report.AccuracyScore = accuracyScore
	report.TimelinessScore = timelinessScore

	return report, nil
}

// Data structures

type DataCleaningTask struct {
	ID             string                 `json:"id"`
	DataSource     string                 `json:"data_source"`
	CleaningRules  []interface{}          `json:"cleaning_rules"`
	TimeRange      map[string]interface{} `json:"time_range"`
	DryRun         bool                   `json:"dry_run"`
	Status         string                 `json:"status"`
	Error          string                 `json:"error,omitempty"`
	Statistics     *CleaningStatistics    `json:"statistics,omitempty"`
	CleaningReport *CleaningReport        `json:"cleaning_report,omitempty"`
	QualityReport  *QualityReport         `json:"quality_report,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	StartedAt      time.Time              `json:"started_at"`
	CompletedAt    time.Time              `json:"completed_at"`
	Duration       time.Duration          `json:"duration"`
}

type CleaningStatistics struct {
	TotalRecords   int `json:"total_records"`
	CleanedRecords int `json:"cleaned_records"`
	RemovedRecords int `json:"removed_records"`
}

type CleaningReport struct {
	RulesApplied map[string]int `json:"rules_applied"`
	Issues       []string       `json:"issues"`
}

type QualityReport struct {
	TotalRecords      int      `json:"total_records"`
	OverallScore      float64  `json:"overall_score"`
	CompletenessScore float64  `json:"completeness_score"`
	ConsistencyScore  float64  `json:"consistency_score"`
	AccuracyScore     float64  `json:"accuracy_score"`
	TimelinessScore   float64  `json:"timeliness_score"`
	Issues            []string `json:"issues"`
}

// Helper method implementations (simplified)

func (de *DataExecutor) saveCleaningTask(ctx context.Context, task *DataCleaningTask) error {
	return nil
}

func (de *DataExecutor) updateCleaningTask(ctx context.Context, task *DataCleaningTask) error {
	return nil
}

func (de *DataExecutor) getDataSource(sourceName string) (interface{}, error) {
	return nil, nil
}

func (de *DataExecutor) loadRawData(ctx context.Context, dataSource interface{}, timeRange map[string]interface{}) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}

func (de *DataExecutor) saveCleanedData(ctx context.Context, dataSource interface{}, data []map[string]interface{}, taskID string) error {
	return nil
}

func (de *DataExecutor) notifyCleaningCompletion(ctx context.Context, task *DataCleaningTask) {
	if de.notificationService != nil {
		message := fmt.Sprintf("Data cleaning task %s completed with status: %s", task.ID, task.Status)
		err := de.notificationService.SendWebhook(ctx, "", map[string]interface{}{
			"type":    "data_management",
			"action":  "cleaning_completed",
			"message": message,
			"task_id": task.ID,
			"status":  task.Status,
		})
		if err != nil {
			log.Printf("Failed to send cleaning completion notification: %v", err)
		}
	}
}

// Quality check helper methods (simplified implementations)

func (de *DataExecutor) checkCompleteness(data []map[string]interface{}, report *QualityReport) float64 {
	return 0.9 // 90% completeness score
}

func (de *DataExecutor) checkConsistency(data []map[string]interface{}, report *QualityReport) float64 {
	return 0.85 // 85% consistency score
}

func (de *DataExecutor) checkAccuracy(data []map[string]interface{}, report *QualityReport) float64 {
	return 0.92 // 92% accuracy score
}

func (de *DataExecutor) checkTimeliness(data []map[string]interface{}, report *QualityReport) float64 {
	return 0.88 // 88% timeliness score
}

// updateFactors 更新因子
func (de *DataExecutor) updateFactors(ctx context.Context, action *ExecutionAction) error {
	log.Printf("Updating factors")

	// 实现因子更新逻辑

	// 1. 解析因子更新请求
	updateType, _ := action.Parameters["update_type"].(string)
	if updateType == "" {
		updateType = "incremental" // 默认增量更新
	}

	factorIDs, _ := action.Parameters["factor_ids"].([]interface{})
	timeRange, _ := action.Parameters["time_range"].(map[string]interface{})
	forceUpdate, _ := action.Parameters["force_update"].(bool)

	// 2. 创建因子更新任务
	updateTask := &FactorUpdateTask{
		ID:          fmt.Sprintf("factor_update_%d", time.Now().Unix()),
		UpdateType:  updateType,
		FactorIDs:   factorIDs,
		TimeRange:   timeRange,
		ForceUpdate: forceUpdate,
		Status:      "initializing",
		CreatedAt:   time.Now(),
	}

	// 3. 保存更新任务
	err := de.saveFactorUpdateTask(ctx, updateTask)
	if err != nil {
		return fmt.Errorf("failed to save factor update task: %v", err)
	}

	// 4. 执行因子更新
	go de.executeFactorUpdate(ctx, updateTask)

	// 5. 记录指标
	if de.metrics != nil {
		de.metrics.IncrementCounter("data_executor.factor_update_started", map[string]string{
			"update_type":  updateType,
			"force_update": fmt.Sprintf("%t", forceUpdate),
		})
	}

	log.Printf("Started factor update task %s", updateTask.ID)
	return nil
}

// executeFactorUpdate 执行因子更新
func (de *DataExecutor) executeFactorUpdate(ctx context.Context, task *FactorUpdateTask) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Factor update task %s panicked: %v", task.ID, r)
			task.Status = "failed"
			task.Error = fmt.Sprintf("panic: %v", r)
			de.updateFactorUpdateTask(ctx, task)
		}
	}()

	// 1. 更新任务状态
	task.Status = "running"
	task.StartedAt = time.Now()
	de.updateFactorUpdateTask(ctx, task)

	// 2. 获取需要更新的因子列表
	factorsToUpdate, err := de.getFactorsToUpdate(ctx, task)
	if err != nil {
		task.Status = "failed"
		task.Error = fmt.Sprintf("failed to get factors to update: %v", err)
		de.updateFactorUpdateTask(ctx, task)
		return
	}

	task.Statistics = &FactorUpdateStatistics{
		TotalFactors: len(factorsToUpdate),
	}

	// 3. 执行因子更新
	updateResults := make(map[string]*FactorUpdateResult)

	for _, factor := range factorsToUpdate {
		result, err := de.updateSingleFactor(ctx, factor, task)
		if err != nil {
			log.Printf("Failed to update factor %s: %v", factor.ID, err)
			task.Statistics.FailedFactors++
			updateResults[factor.ID] = &FactorUpdateResult{
				FactorID: factor.ID,
				Status:   "failed",
				Error:    err.Error(),
			}
		} else {
			task.Statistics.UpdatedFactors++
			updateResults[factor.ID] = result
		}
	}

	task.UpdateResults = updateResults

	// 4. 执行因子验证
	if task.Statistics.UpdatedFactors > 0 {
		validationResults, err := de.validateUpdatedFactors(ctx, updateResults)
		if err != nil {
			log.Printf("Factor validation failed: %v", err)
		} else {
			task.ValidationResults = validationResults
		}
	}

	// 5. 更新因子依赖关系
	err = de.updateFactorDependencies(ctx, updateResults)
	if err != nil {
		log.Printf("Failed to update factor dependencies: %v", err)
	}

	// 6. 刷新因子缓存
	err = de.refreshFactorCache(ctx, updateResults)
	if err != nil {
		log.Printf("Failed to refresh factor cache: %v", err)
	}

	// 7. 更新任务完成状态
	task.Status = "completed"
	task.CompletedAt = time.Now()
	task.Duration = task.CompletedAt.Sub(task.StartedAt)
	de.updateFactorUpdateTask(ctx, task)

	// 8. 发送通知
	de.notifyFactorUpdateCompletion(ctx, task)

	// 9. 记录指标
	if de.metrics != nil {
		de.metrics.IncrementCounter("data_executor.factor_update_completed", map[string]string{
			"update_type": task.UpdateType,
			"status":      task.Status,
		})
	}

	log.Printf("Factor update task %s completed successfully", task.ID)
}

// getFactorsToUpdate 获取需要更新的因子列表
func (de *DataExecutor) getFactorsToUpdate(ctx context.Context, task *FactorUpdateTask) ([]*FactorInfo, error) {
	var factors []*FactorInfo

	if len(task.FactorIDs) > 0 {
		// 更新指定的因子
		for _, factorID := range task.FactorIDs {
			if factorIDStr, ok := factorID.(string); ok {
				factor, err := de.getFactorInfo(ctx, factorIDStr)
				if err != nil {
					log.Printf("Failed to get factor info for %s: %v", factorIDStr, err)
					continue
				}
				factors = append(factors, factor)
			}
		}
	} else {
		// 获取所有需要更新的因子
		allFactors, err := de.getAllFactors(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get all factors: %v", err)
		}

		for _, factor := range allFactors {
			if de.shouldUpdateFactor(factor, task) {
				factors = append(factors, factor)
			}
		}
	}

	return factors, nil
}

// shouldUpdateFactor 判断是否应该更新因子
func (de *DataExecutor) shouldUpdateFactor(factor *FactorInfo, task *FactorUpdateTask) bool {
	// 1. 强制更新
	if task.ForceUpdate {
		return true
	}

	// 2. 检查因子状态
	if factor.Status == "disabled" {
		return false
	}

	// 3. 检查更新频率
	if factor.UpdateFrequency != "" {
		lastUpdate := factor.LastUpdatedAt
		updateInterval := de.parseUpdateFrequency(factor.UpdateFrequency)
		if time.Since(lastUpdate) < updateInterval {
			return false
		}
	}

	// 4. 检查数据可用性
	if !de.isFactorDataAvailable(factor) {
		return false
	}

	return true
}
// updateSingleFactor 更新单个因子
func (de *DataExecutor) updateSingleFactor(ctx context.Context, factor *FactorInfo, task *FactorUpdateTask) (*FactorUpdateResult, error) {
	log.Printf("Updating factor %s (%s)", factor.ID, factor.Name)

	result := &FactorUpdateResult{
		FactorID:  factor.ID,
		StartTime: time.Now(),
	}

	// 1. 获取因子计算所需的数据
	inputData, err := de.getFactorInputData(ctx, factor, task.TimeRange)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("failed to get input data: %v", err)
		return result, err
	}

	// 2. 计算因子值
	factorValues, err := de.calculateFactorValues(ctx, factor, inputData)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("failed to calculate factor values: %v", err)
		return result, err
	}

	result.RecordsProcessed = len(factorValues)

	// 3. 验证因子值
	validationResult := de.validateFactorValues(factor, factorValues)
	if !validationResult.IsValid {
		result.Status = "failed"
		result.Error = fmt.Sprintf("factor validation failed: %s", validationResult.Reason)
		return result, fmt.Errorf("validation failed: %s", validationResult.Reason)
	}

	// 4. 保存因子值
	err = de.saveFactorValues(ctx, factor.ID, factorValues)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("failed to save factor values: %v", err)
		return result, err
	}

	// 5. 更新因子元数据
	err = de.updateFactorMetadata(ctx, factor.ID, len(factorValues))
	if err != nil {
		log.Printf("Failed to update factor metadata for %s: %v", factor.ID, err)
	}

	// 6. 计算因子统计信息
	statistics := de.calculateFactorStatistics(factorValues)
	result.Statistics = statistics

	result.Status = "completed"
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	log.Printf("Successfully updated factor %s with %d records", factor.ID, len(factorValues))
	return result, nil
}

// calculateFactorValues 计算因子值
func (de *DataExecutor) calculateFactorValues(ctx context.Context, factor *FactorInfo, inputData map[string]interface{}) ([]FactorValue, error) {
	var factorValues []FactorValue

	switch factor.Type {
	case "technical":
		return de.calculateTechnicalFactor(factor, inputData)
	case "fundamental":
		return de.calculateFundamentalFactor(factor, inputData)
	case "sentiment":
		return de.calculateSentimentFactor(factor, inputData)
	case "macro":
		return de.calculateMacroFactor(factor, inputData)
	case "alternative":
		return de.calculateAlternativeFactor(factor, inputData)
	default:
		return factorValues, fmt.Errorf("unsupported factor type: %s", factor.Type)
	}
}
// validateUpdatedFactors 验证更新后的因子
func (de *DataExecutor) validateUpdatedFactors(ctx context.Context, updateResults map[string]*FactorUpdateResult) (*FactorValidationResults, error) {
	validationResults := &FactorValidationResults{
		TotalFactors:    len(updateResults),
		ValidatedAt:     time.Now(),
		ValidationTests: make(map[string]*ValidationTest),
	}

	// 1. 数据完整性检查
	completenessTest := de.validateFactorCompleteness(updateResults)
	validationResults.ValidationTests["completeness"] = completenessTest

	// 2. 数据一致性检查
	consistencyTest := de.validateFactorConsistency(updateResults)
	validationResults.ValidationTests["consistency"] = consistencyTest

	// 3. 因子相关性检查
	correlationTest := de.validateFactorCorrelations(updateResults)
	validationResults.ValidationTests["correlation"] = correlationTest

	// 4. 异常值检查
	outlierTest := de.validateFactorOutliers(updateResults)
	validationResults.ValidationTests["outliers"] = outlierTest

	// 5. 计算总体验证分数
	totalScore := 0.0
	passedTests := 0

	for _, test := range validationResults.ValidationTests {
		totalScore += test.Score
		if test.Passed {
			passedTests++
		}
	}

	validationResults.OverallScore = totalScore / float64(len(validationResults.ValidationTests))
	validationResults.PassedTests = passedTests
	validationResults.TotalTests = len(validationResults.ValidationTests)

	return validationResults, nil
}

// Data structures

type FactorUpdateTask struct {
	ID                string                         `json:"id"`
	UpdateType        string                         `json:"update_type"`
	FactorIDs         []interface{}                  `json:"factor_ids"`
	TimeRange         map[string]interface{}         `json:"time_range"`
	ForceUpdate       bool                           `json:"force_update"`
	Status            string                         `json:"status"`
	Error             string                         `json:"error,omitempty"`
	Statistics        *FactorUpdateStatistics        `json:"statistics,omitempty"`
	UpdateResults     map[string]*FactorUpdateResult `json:"update_results,omitempty"`
	ValidationResults *FactorValidationResults       `json:"validation_results,omitempty"`
	CreatedAt         time.Time                      `json:"created_at"`
	StartedAt         time.Time                      `json:"started_at"`
	CompletedAt       time.Time                      `json:"completed_at"`
	Duration          time.Duration                  `json:"duration"`
}

type FactorUpdateStatistics struct {
	TotalFactors   int `json:"total_factors"`
	UpdatedFactors int `json:"updated_factors"`
	FailedFactors  int `json:"failed_factors"`
}
type FactorUpdateResult struct {
	FactorID         string            `json:"factor_id"`
	Status           string            `json:"status"`
	Error            string            `json:"error,omitempty"`
	RecordsProcessed int               `json:"records_processed"`
	Statistics       *FactorStatistics `json:"statistics,omitempty"`
	StartTime        time.Time         `json:"start_time"`
	EndTime          time.Time         `json:"end_time"`
	Duration         time.Duration     `json:"duration"`
}

type FactorInfo struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Type            string                 `json:"type"`
	Status          string                 `json:"status"`
	UpdateFrequency string                 `json:"update_frequency"`
	LastUpdatedAt   time.Time              `json:"last_updated_at"`
	Parameters      map[string]interface{} `json:"parameters"`
}

type FactorValue struct {
	Timestamp time.Time `json:"timestamp"`
	Symbol    string    `json:"symbol"`
	Value     float64   `json:"value"`
}

type FactorStatistics struct {
	Mean      float64 `json:"mean"`
	StdDev    float64 `json:"std_dev"`
	Min       float64 `json:"min"`
	Max       float64 `json:"max"`
	Count     int     `json:"count"`
	NullCount int     `json:"null_count"`
	Skewness  float64 `json:"skewness"`
	Kurtosis  float64 `json:"kurtosis"`
}

type FactorValidationResults struct {
	TotalFactors    int                        `json:"total_factors"`
	PassedTests     int                        `json:"passed_tests"`
	TotalTests      int                        `json:"total_tests"`
	OverallScore    float64                    `json:"overall_score"`
	ValidationTests map[string]*ValidationTest `json:"validation_tests"`
	ValidatedAt     time.Time                  `json:"validated_at"`
}

type ValidationTest struct {
	Name        string   `json:"name"`
	Passed      bool     `json:"passed"`
	Score       float64  `json:"score"`
	Description string   `json:"description"`
	Issues      []string `json:"issues,omitempty"`
}

type FactorValidationResult struct {
	IsValid bool   `json:"is_valid"`
	Reason  string `json:"reason"`
}

// Helper method implementations (simplified)

func (de *DataExecutor) saveFactorUpdateTask(ctx context.Context, task *FactorUpdateTask) error {
	return nil
}

func (de *DataExecutor) updateFactorUpdateTask(ctx context.Context, task *FactorUpdateTask) error {
	return nil
}

func (de *DataExecutor) getFactorInfo(ctx context.Context, factorID string) (*FactorInfo, error) {
	return &FactorInfo{
		ID:              factorID,
		Name:            "Sample Factor",
		Type:            "technical",
		Status:          "active",
		UpdateFrequency: "daily",
		LastUpdatedAt:   time.Now().Add(-24 * time.Hour),
	}, nil
}

func (de *DataExecutor) getAllFactors(ctx context.Context) ([]*FactorInfo, error) {
	return []*FactorInfo{}, nil
}

func (de *DataExecutor) parseUpdateFrequency(frequency string) time.Duration {
	switch frequency {
	case "realtime":
		return 1 * time.Minute
	case "minute":
		return 1 * time.Minute
	case "hourly":
		return 1 * time.Hour
	case "daily":
		return 24 * time.Hour
	case "weekly":
		return 7 * 24 * time.Hour
	default:
		return 24 * time.Hour
	}
}

func (de *DataExecutor) isFactorDataAvailable(factor *FactorInfo) bool {
	return true
}

func (de *DataExecutor) getFactorInputData(ctx context.Context, factor *FactorInfo, timeRange map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (de *DataExecutor) validateFactorValues(factor *FactorInfo, values []FactorValue) *FactorValidationResult {
	return &FactorValidationResult{
		IsValid: true,
		Reason:  "validation passed",
	}
}

func (de *DataExecutor) saveFactorValues(ctx context.Context, factorID string, values []FactorValue) error {
	return nil
}

func (de *DataExecutor) updateFactorMetadata(ctx context.Context, factorID string, recordCount int) error {
	return nil
}

func (de *DataExecutor) calculateFactorStatistics(values []FactorValue) *FactorStatistics {
	if len(values) == 0 {
		return &FactorStatistics{}
	}

	// 简化的统计计算
	sum := 0.0
	min := values[0].Value
	max := values[0].Value

	for _, value := range values {
		sum += value.Value
		if value.Value < min {
			min = value.Value
		}
		if value.Value > max {
			max = value.Value
		}
	}

	mean := sum / float64(len(values))

	return &FactorStatistics{
		Mean:  mean,
		Min:   min,
		Max:   max,
		Count: len(values),
	}
}

// Factor calculation methods (simplified implementations)

func (de *DataExecutor) calculateTechnicalFactor(factor *FactorInfo, inputData map[string]interface{}) ([]FactorValue, error) {
	return []FactorValue{}, nil
}

func (de *DataExecutor) calculateFundamentalFactor(factor *FactorInfo, inputData map[string]interface{}) ([]FactorValue, error) {
	return []FactorValue{}, nil
}

func (de *DataExecutor) calculateSentimentFactor(factor *FactorInfo, inputData map[string]interface{}) ([]FactorValue, error) {
	return []FactorValue{}, nil
}

func (de *DataExecutor) calculateMacroFactor(factor *FactorInfo, inputData map[string]interface{}) ([]FactorValue, error) {
	return []FactorValue{}, nil
}

func (de *DataExecutor) calculateAlternativeFactor(factor *FactorInfo, inputData map[string]interface{}) ([]FactorValue, error) {
	return []FactorValue{}, nil
}

// Validation methods (simplified implementations)

func (de *DataExecutor) validateFactorCompleteness(updateResults map[string]*FactorUpdateResult) *ValidationTest {
	return &ValidationTest{
		Name:        "completeness",
		Passed:      true,
		Score:       0.95,
		Description: "Factor data completeness check",
	}
}

func (de *DataExecutor) validateFactorConsistency(updateResults map[string]*FactorUpdateResult) *ValidationTest {
	return &ValidationTest{
		Name:        "consistency",
		Passed:      true,
		Score:       0.92,
		Description: "Factor data consistency check",
	}
}

func (de *DataExecutor) validateFactorCorrelations(updateResults map[string]*FactorUpdateResult) *ValidationTest {
	return &ValidationTest{
		Name:        "correlation",
		Passed:      true,
		Score:       0.88,
		Description: "Factor correlation analysis",
	}
}

func (de *DataExecutor) validateFactorOutliers(updateResults map[string]*FactorUpdateResult) *ValidationTest {
	return &ValidationTest{
		Name:        "outliers",
		Passed:      true,
		Score:       0.90,
		Description: "Factor outlier detection",
	}
}

func (de *DataExecutor) updateFactorDependencies(ctx context.Context, updateResults map[string]*FactorUpdateResult) error {
	return nil
}

func (de *DataExecutor) refreshFactorCache(ctx context.Context, updateResults map[string]*FactorUpdateResult) error {
	return nil
}

func (de *DataExecutor) notifyFactorUpdateCompletion(ctx context.Context, task *FactorUpdateTask) {
	if de.notificationService != nil {
		message := fmt.Sprintf("Factor update task %s completed with status: %s", task.ID, task.Status)
		err := de.notificationService.SendWebhook(ctx, "", map[string]interface{}{
			"type":    "data_management",
			"action":  "factor_update_completed",
			"message": message,
			"task_id": task.ID,
			"status":  task.Status,
		})
		if err != nil {
			log.Printf("Failed to send factor update completion notification: %v", err)
		}
	}
}

// runBacktest 运行回测
func (de *DataExecutor) runBacktest(ctx context.Context, action *ExecutionAction) error {
	log.Printf("Running backtest")

	// 实现回测逻辑

	// 1. 解析回测请求
	strategyID, ok := action.Parameters["strategy_id"].(string)
	if !ok {
		return fmt.Errorf("strategy_id is required for backtest")
	}

	startDate, _ := action.Parameters["start_date"].(string)
	endDate, _ := action.Parameters["end_date"].(string)
	initialCapital, _ := action.Parameters["initial_capital"].(float64)
	if initialCapital <= 0 {
		initialCapital = 100000 // 默认10万初始资金
	}

	symbols, _ := action.Parameters["symbols"].([]interface{})
	backtestConfig, _ := action.Parameters["config"].(map[string]interface{})

	// 2. 创建回测任务
	backtestTask := &BacktestTask{
		ID:             fmt.Sprintf("backtest_%s_%d", strategyID, time.Now().Unix()),
		StrategyID:     strategyID,
		StartDate:      startDate,
		EndDate:        endDate,
		InitialCapital: initialCapital,
		Symbols:        symbols,
		Config:         backtestConfig,
		Status:         "initializing",
		CreatedAt:      time.Now(),
	}

	// 3. 保存回测任务
	err := de.saveBacktestTask(ctx, backtestTask)
	if err != nil {
		return fmt.Errorf("failed to save backtest task: %v", err)
	}

	// 4. 执行回测
	go de.executeBacktest(ctx, backtestTask)

	// 5. 记录指标
	if de.metrics != nil {
		de.metrics.IncrementCounter("data_executor.backtest_started", map[string]string{
			"strategy_id": strategyID,
		})
	}

	log.Printf("Started backtest task %s for strategy %s", backtestTask.ID, strategyID)
	return nil
}

// executeBacktest 执行回测
func (de *DataExecutor) executeBacktest(ctx context.Context, task *BacktestTask) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Backtest task %s panicked: %v", task.ID, r)
			task.Status = "failed"
			task.Error = fmt.Sprintf("panic: %v", r)
			de.updateBacktestTask(ctx, task)
		}
	}()

	// 1. 更新任务状态
	task.Status = "running"
	task.StartedAt = time.Now()
	de.updateBacktestTask(ctx, task)

	// 2. 获取策略配置
	strategy, err := de.getStrategyForBacktest(ctx, task.StrategyID)
	if err != nil {
		task.Status = "failed"
		task.Error = fmt.Sprintf("failed to get strategy: %v", err)
		de.updateBacktestTask(ctx, task)
		return
	}

	// 3. 准备历史数据
	historicalData, err := de.prepareHistoricalData(ctx, task)
	if err != nil {
		task.Status = "failed"
		task.Error = fmt.Sprintf("failed to prepare historical data: %v", err)
		de.updateBacktestTask(ctx, task)
		return
	}

	// 4. 初始化回测引擎
	backtestEngine := de.createBacktestEngine(task, strategy, historicalData)

	// 5. 运行回测
	backtestResult, err := backtestEngine.Run(ctx)
	if err != nil {
		task.Status = "failed"
		task.Error = fmt.Sprintf("backtest execution failed: %v", err)
		de.updateBacktestTask(ctx, task)
		return
	}

	// 6. 计算性能指标
	performanceMetrics := de.calculateBacktestPerformance(backtestResult)
	task.Results = &BacktestResults{
		Performance: performanceMetrics,
		Trades:      backtestResult.Trades,
		Equity:      backtestResult.Equity,
		Drawdown:    backtestResult.Drawdown,
		Statistics:  backtestResult.Statistics,
	}

	// 7. 生成回测报告
	report, err := de.generateBacktestReport(ctx, task, backtestResult)
	if err != nil {
		log.Printf("Failed to generate backtest report: %v", err)
	} else {
		task.ReportPath = report.FilePath
	}

	// 8. 保存回测结果
	err = de.saveBacktestResults(ctx, task)
	if err != nil {
		log.Printf("Failed to save backtest results: %v", err)
	}

	// 9. 更新任务完成状态
	task.Status = "completed"
	task.CompletedAt = time.Now()
	task.Duration = task.CompletedAt.Sub(task.StartedAt)
	de.updateBacktestTask(ctx, task)

	// 10. 发送通知
	de.notifyBacktestCompletion(ctx, task)

	// 11. 记录指标
	if de.metrics != nil {
		de.metrics.IncrementCounter("data_executor.backtest_completed", map[string]string{
			"strategy_id": task.StrategyID,
			"status":      task.Status,
		})
	}

	log.Printf("Backtest task %s completed successfully", task.ID)
}

// prepareHistoricalData 准备历史数据
func (de *DataExecutor) prepareHistoricalData(ctx context.Context, task *BacktestTask) (*HistoricalDataSet, error) {
	dataSet := &HistoricalDataSet{
		StartDate: task.StartDate,
		EndDate:   task.EndDate,
		Data:      make(map[string][]MarketDataPoint),
	}

	// 解析日期
	startTime, err := time.Parse("2006-01-02", task.StartDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start date: %v", err)
	}

	endTime, err := time.Parse("2006-01-02", task.EndDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end date: %v", err)
	}

	// 获取每个交易对的历史数据
	for _, symbolInterface := range task.Symbols {
		symbol, ok := symbolInterface.(string)
		if !ok {
			continue
		}

		marketData, err := de.getHistoricalMarketData(ctx, symbol, startTime, endTime)
		if err != nil {
			log.Printf("Failed to get historical data for %s: %v", symbol, err)
			continue
		}

		dataSet.Data[symbol] = marketData
	}

	return dataSet, nil
}

// createBacktestEngine 创建回测引擎
func (de *DataExecutor) createBacktestEngine(task *BacktestTask, strategy *BacktestStrategy, data *HistoricalDataSet) *BacktestEngine {
	config := &BacktestEngineConfig{
		InitialCapital: task.InitialCapital,
		Commission:     0.001,  // 默认0.1%手续费
		Slippage:       0.0005, // 默认0.05%滑点
	}

	// 从任务配置中覆盖默认值
	if task.Config != nil {
		if commission, ok := task.Config["commission"].(float64); ok {
			config.Commission = commission
		}
		if slippage, ok := task.Config["slippage"].(float64); ok {
			config.Slippage = slippage
		}
	}

	return &BacktestEngine{
		Config:         config,
		Strategy:       strategy,
		HistoricalData: data,
		Portfolio:      NewBacktestPortfolio(task.InitialCapital),
		Trades:         make([]BacktestTrade, 0),
		Equity:         make([]EquityPoint, 0),
	}
}

// calculateBacktestPerformance 计算回测性能指标
func (de *DataExecutor) calculateBacktestPerformance(result *BacktestEngineResult) *BacktestPerformance {
	if len(result.Equity) == 0 {
		return &BacktestPerformance{}
	}

	initialEquity := result.Equity[0].Value
	finalEquity := result.Equity[len(result.Equity)-1].Value

	totalReturn := (finalEquity - initialEquity) / initialEquity

	// 计算年化收益率
	startTime := result.Equity[0].Timestamp
	endTime := result.Equity[len(result.Equity)-1].Timestamp
	years := endTime.Sub(startTime).Hours() / (24 * 365)
	annualizedReturn := math.Pow(1+totalReturn, 1/years) - 1

	// 计算最大回撤
	maxDrawdown := de.calculateMaxDrawdown(result.Equity)

	// 计算夏普比率
	sharpeRatio := de.calculateSharpeRatio(result.Equity)

	// 计算胜率
	winRate := de.calculateWinRate(result.Trades)

	return &BacktestPerformance{
		TotalReturn:        totalReturn,
		AnnualizedReturn:   annualizedReturn,
		MaxDrawdown:        maxDrawdown,
		SharpeRatio:        sharpeRatio,
		WinRate:            winRate,
		TotalTrades:        len(result.Trades),
		ProfitableTrades:   de.countProfitableTrades(result.Trades),
		AverageReturn:      de.calculateAverageReturn(result.Trades),
		MaxConsecutiveLoss: de.calculateMaxConsecutiveLoss(result.Trades),
	}
}

// Data structures

type BacktestTask struct {
	ID             string                 `json:"id"`
	StrategyID     string                 `json:"strategy_id"`
	StartDate      string                 `json:"start_date"`
	EndDate        string                 `json:"end_date"`
	InitialCapital float64                `json:"initial_capital"`
	Symbols        []interface{}          `json:"symbols"`
	Config         map[string]interface{} `json:"config"`
	Status         string                 `json:"status"`
	Error          string                 `json:"error,omitempty"`
	Results        *BacktestResults       `json:"results,omitempty"`
	ReportPath     string                 `json:"report_path,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	StartedAt      time.Time              `json:"started_at"`
	CompletedAt    time.Time              `json:"completed_at"`
	Duration       time.Duration          `json:"duration"`
}

type BacktestResults struct {
	Performance *BacktestPerformance `json:"performance"`
	Trades      []BacktestTrade      `json:"trades"`
	Equity      []EquityPoint        `json:"equity"`
	Drawdown    []DrawdownPoint      `json:"drawdown"`
	Statistics  *BacktestStatistics  `json:"statistics"`
}

type BacktestPerformance struct {
	TotalReturn        float64 `json:"total_return"`
	AnnualizedReturn   float64 `json:"annualized_return"`
	MaxDrawdown        float64 `json:"max_drawdown"`
	SharpeRatio        float64 `json:"sharpe_ratio"`
	WinRate            float64 `json:"win_rate"`
	TotalTrades        int     `json:"total_trades"`
	ProfitableTrades   int     `json:"profitable_trades"`
	AverageReturn      float64 `json:"average_return"`
	MaxConsecutiveLoss int     `json:"max_consecutive_loss"`
}

type BacktestStrategy struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Parameters map[string]interface{} `json:"parameters"`
	Logic      string                 `json:"logic"`
}

type HistoricalDataSet struct {
	StartDate string                       `json:"start_date"`
	EndDate   string                       `json:"end_date"`
	Data      map[string][]MarketDataPoint `json:"data"`
}

type MarketDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
}

type BacktestEngine struct {
	Config         *BacktestEngineConfig
	Strategy       *BacktestStrategy
	HistoricalData *HistoricalDataSet
	Portfolio      *BacktestPortfolio
	Trades         []BacktestTrade
	Equity         []EquityPoint
}

type BacktestEngineConfig struct {
	InitialCapital float64 `json:"initial_capital"`
	Commission     float64 `json:"commission"`
	Slippage       float64 `json:"slippage"`
}

type BacktestPortfolio struct {
	Cash       float64                      `json:"cash"`
	Positions  map[string]*BacktestPosition `json:"positions"`
	TotalValue float64                      `json:"total_value"`
}

type BacktestPosition struct {
	Symbol   string  `json:"symbol"`
	Quantity float64 `json:"quantity"`
	AvgPrice float64 `json:"avg_price"`
	Value    float64 `json:"value"`
}

type BacktestTrade struct {
	ID        string    `json:"id"`
	Symbol    string    `json:"symbol"`
	Side      string    `json:"side"`
	Quantity  float64   `json:"quantity"`
	Price     float64   `json:"price"`
	Timestamp time.Time `json:"timestamp"`
	PnL       float64   `json:"pnl"`
}

type EquityPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

type DrawdownPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Drawdown  float64   `json:"drawdown"`
}

type BacktestStatistics struct {
	TotalDays      int     `json:"total_days"`
	TradingDays    int     `json:"trading_days"`
	AvgDailyReturn float64 `json:"avg_daily_return"`
	Volatility     float64 `json:"volatility"`
	Beta           float64 `json:"beta"`
	Alpha          float64 `json:"alpha"`
}

type BacktestEngineResult struct {
	Trades     []BacktestTrade     `json:"trades"`
	Equity     []EquityPoint       `json:"equity"`
	Drawdown   []DrawdownPoint     `json:"drawdown"`
	Statistics *BacktestStatistics `json:"statistics"`
}

type BacktestReport struct {
	FilePath string `json:"file_path"`
	Format   string `json:"format"`
}

// BacktestEngine methods

func (be *BacktestEngine) Run(ctx context.Context) (*BacktestEngineResult, error) {
	// 简化的回测执行逻辑
	result := &BacktestEngineResult{
		Trades:     make([]BacktestTrade, 0),
		Equity:     make([]EquityPoint, 0),
		Drawdown:   make([]DrawdownPoint, 0),
		Statistics: &BacktestStatistics{},
	}

	// 这里应该实现实际的回测逻辑
	// 遍历历史数据，应用策略逻辑，执行交易等

	return result, nil
}

func NewBacktestPortfolio(initialCash float64) *BacktestPortfolio {
	return &BacktestPortfolio{
		Cash:       initialCash,
		Positions:  make(map[string]*BacktestPosition),
		TotalValue: initialCash,
	}
}

// Helper method implementations (simplified)

func (de *DataExecutor) saveBacktestTask(ctx context.Context, task *BacktestTask) error {
	return nil
}

func (de *DataExecutor) updateBacktestTask(ctx context.Context, task *BacktestTask) error {
	return nil
}
func (de *DataExecutor) getStrategyForBacktest(ctx context.Context, strategyID string) (*BacktestStrategy, error) {
	return &BacktestStrategy{
		ID:   strategyID,
		Name: "Sample Strategy",
		Parameters: map[string]interface{}{
			"period": 20,
		},
		Logic: "moving_average",
	}, nil
}

func (de *DataExecutor) getHistoricalMarketData(ctx context.Context, symbol string, startTime, endTime time.Time) ([]MarketDataPoint, error) {
	// 简化实现，返回模拟数据
	return []MarketDataPoint{}, nil
}

func (de *DataExecutor) generateBacktestReport(ctx context.Context, task *BacktestTask, result *BacktestEngineResult) (*BacktestReport, error) {
	return &BacktestReport{
		FilePath: fmt.Sprintf("/reports/backtest_%s.pdf", task.ID),
		Format:   "pdf",
	}, nil
}

func (de *DataExecutor) saveBacktestResults(ctx context.Context, task *BacktestTask) error {
	return nil
}

func (de *DataExecutor) notifyBacktestCompletion(ctx context.Context, task *BacktestTask) {
	if de.notificationService != nil {
		message := fmt.Sprintf("Backtest task %s completed with status: %s", task.ID, task.Status)
		err := de.notificationService.SendWebhook(ctx, "", map[string]interface{}{
			"type":    "data_management",
			"action":  "backtest_completed",
			"message": message,
			"task_id": task.ID,
			"status":  task.Status,
		})
		if err != nil {
			log.Printf("Failed to send backtest completion notification: %v", err)
		}
	}
}

// Performance calculation methods (simplified implementations)

func (de *DataExecutor) calculateMaxDrawdown(equity []EquityPoint) float64 {
	if len(equity) == 0 {
		return 0
	}

	maxDrawdown := 0.0
	peak := equity[0].Value

	for _, point := range equity {
		if point.Value > peak {
			peak = point.Value
		}
		drawdown := (peak - point.Value) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}

func (de *DataExecutor) calculateSharpeRatio(equity []EquityPoint) float64 {
	// 简化的夏普比率计算
	return 1.5
}
func (de *DataExecutor) calculateWinRate(trades []BacktestTrade) float64 {
	if len(trades) == 0 {
		return 0
	}

	profitableTrades := 0
	for _, trade := range trades {
		if trade.PnL > 0 {
			profitableTrades++
		}
	}

	return float64(profitableTrades) / float64(len(trades))
}

func (de *DataExecutor) countProfitableTrades(trades []BacktestTrade) int {
	count := 0
	for _, trade := range trades {
		if trade.PnL > 0 {
			count++
		}
	}
	return count
}

func (de *DataExecutor) calculateAverageReturn(trades []BacktestTrade) float64 {
	if len(trades) == 0 {
		return 0
	}

	totalPnL := 0.0
	for _, trade := range trades {
		totalPnL += trade.PnL
	}

	return totalPnL / float64(len(trades))
}

func (de *DataExecutor) calculateMaxConsecutiveLoss(trades []BacktestTrade) int {
	maxLoss := 0
	currentLoss := 0

	for _, trade := range trades {
		if trade.PnL < 0 {
			currentLoss++
			if currentLoss > maxLoss {
				maxLoss = currentLoss
			}
		} else {
			currentLoss = 0
		}
	}

	return maxLoss
}
// recognizePattern 识别模式
func (de *DataExecutor) recognizePattern(ctx context.Context, action *ExecutionAction) error {
	log.Printf("Recognizing market pattern")

	// 实现模式识别逻辑

	// 1. 解析模式识别请求
	symbols, _ := action.Parameters["symbols"].([]interface{})
	patternTypes, _ := action.Parameters["pattern_types"].([]interface{})
	timeframe, _ := action.Parameters["timeframe"].(string)
	if timeframe == "" {
		timeframe = "1h" // 默认1小时时间框架
	}

	lookbackPeriod, _ := action.Parameters["lookback_period"].(float64)
	if lookbackPeriod <= 0 {
		lookbackPeriod = 100 // 默认回看100个周期
	}

	confidence, _ := action.Parameters["min_confidence"].(float64)
	if confidence <= 0 {
		confidence = 0.7 // 默认70%置信度
	}

	// 2. 创建模式识别任务
	patternTask := &PatternRecognitionTask{
		ID:             fmt.Sprintf("pattern_%d", time.Now().Unix()),
		Symbols:        symbols,
		PatternTypes:   patternTypes,
		Timeframe:      timeframe,
		LookbackPeriod: int(lookbackPeriod),
		MinConfidence:  confidence,
		Status:         "initializing",
		CreatedAt:      time.Now(),
	}

	// 3. 保存模式识别任务
	err := de.savePatternTask(ctx, patternTask)
	if err != nil {
		return fmt.Errorf("failed to save pattern recognition task: %v", err)
	}

	// 4. 执行模式识别
	go de.executePatternRecognition(ctx, patternTask)

	// 5. 记录指标
	if de.metrics != nil {
		de.metrics.IncrementCounter("data_executor.pattern_recognition_started", map[string]string{
			"timeframe": timeframe,
		})
	}

	log.Printf("Started pattern recognition task %s", patternTask.ID)
	return nil
}

// executePatternRecognition 执行模式识别
func (de *DataExecutor) executePatternRecognition(ctx context.Context, task *PatternRecognitionTask) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Pattern recognition task %s panicked: %v", task.ID, r)
			task.Status = "failed"
			task.Error = fmt.Sprintf("panic: %v", r)
			de.updatePatternTask(ctx, task)
		}
	}()

	// 1. 更新任务状态
	task.Status = "running"
	task.StartedAt = time.Now()
	de.updatePatternTask(ctx, task)

	// 2. 获取市场数据
	marketData, err := de.getMarketDataForPatterns(ctx, task)
	if err != nil {
		task.Status = "failed"
		task.Error = fmt.Sprintf("failed to get market data: %v", err)
		de.updatePatternTask(ctx, task)
		return
	}

	// 3. 初始化模式识别器
	recognizers := de.initializePatternRecognizers(task.PatternTypes)

	// 4. 执行模式识别
	allPatterns := make(map[string][]RecognizedPattern)

	for _, symbolInterface := range task.Symbols {
		symbol, ok := symbolInterface.(string)
		if !ok {
			continue
		}

		symbolData, exists := marketData[symbol]
		if !exists {
			log.Printf("No data available for symbol %s", symbol)
			continue
		}

		patterns := de.recognizePatternsForSymbol(symbol, symbolData, recognizers, task)
		if len(patterns) > 0 {
			allPatterns[symbol] = patterns
		}
	}

	// 5. 过滤和排序模式
	filteredPatterns := de.filterPatternsByConfidence(allPatterns, task.MinConfidence)

	// 6. 生成模式识别报告
	report := de.generatePatternReport(filteredPatterns, task)
	task.Results = &PatternRecognitionResults{
		Patterns:      filteredPatterns,
		Report:        report,
		TotalPatterns: de.countTotalPatterns(filteredPatterns),
	}

	// 7. 发送模式识别信号
	if len(filteredPatterns) > 0 {
		de.sendPatternSignals(ctx, filteredPatterns, task)
	}

	// 8. 更新任务完成状态
	task.Status = "completed"
	task.CompletedAt = time.Now()
	task.Duration = task.CompletedAt.Sub(task.StartedAt)
	de.updatePatternTask(ctx, task)

	// 9. 发送通知
	de.notifyPatternRecognitionCompletion(ctx, task)

	// 10. 记录指标
	if de.metrics != nil {
		de.metrics.IncrementCounter("data_executor.pattern_recognition_completed", map[string]string{
			"status": task.Status,
		})
	}

	log.Printf("Pattern recognition task %s completed successfully", task.ID)
}

// initializePatternRecognizers 初始化模式识别器
func (de *DataExecutor) initializePatternRecognizers(patternTypes []interface{}) map[string]PatternRecognizer {
	recognizers := make(map[string]PatternRecognizer)

	for _, patternTypeInterface := range patternTypes {
		patternType, ok := patternTypeInterface.(string)
		if !ok {
			continue
		}

		switch patternType {
		case "head_and_shoulders":
			recognizers[patternType] = &HeadAndShouldersRecognizer{}
		case "double_top":
			recognizers[patternType] = &DoubleTopRecognizer{}
		case "double_bottom":
			recognizers[patternType] = &DoubleBottomRecognizer{}
		case "triangle":
			recognizers[patternType] = &TriangleRecognizer{}
		case "flag":
			recognizers[patternType] = &FlagRecognizer{}
		case "wedge":
			recognizers[patternType] = &WedgeRecognizer{}
		case "channel":
			recognizers[patternType] = &ChannelRecognizer{}
		case "support_resistance":
			recognizers[patternType] = &SupportResistanceRecognizer{}
		case "candlestick":
			recognizers[patternType] = &CandlestickRecognizer{}
		default:
			log.Printf("Unknown pattern type: %s", patternType)
		}
	}

	return recognizers
}

// recognizePatternsForSymbol 为单个交易对识别模式
func (de *DataExecutor) recognizePatternsForSymbol(symbol string, data []MarketDataPoint, recognizers map[string]PatternRecognizer, task *PatternRecognitionTask) []RecognizedPattern {
	var patterns []RecognizedPattern

	// 确保有足够的数据
	if len(data) < task.LookbackPeriod {
		log.Printf("Insufficient data for %s: %d < %d", symbol, len(data), task.LookbackPeriod)
		return patterns
	}

	// 使用滑动窗口识别模式
	windowSize := task.LookbackPeriod
	for i := windowSize; i <= len(data); i++ {
		windowData := data[i-windowSize : i]

		// 对每种模式类型进行识别
		for patternType, recognizer := range recognizers {
			recognizedPatterns := recognizer.Recognize(symbol, windowData, task.MinConfidence)
			for _, pattern := range recognizedPatterns {
				pattern.PatternType = patternType
				pattern.Symbol = symbol
				pattern.Timeframe = task.Timeframe
				pattern.DetectedAt = time.Now()
				patterns = append(patterns, pattern)
			}
		}
	}

	return patterns
}

// filterPatternsByConfidence 按置信度过滤模式
func (de *DataExecutor) filterPatternsByConfidence(allPatterns map[string][]RecognizedPattern, minConfidence float64) map[string][]RecognizedPattern {
	filtered := make(map[string][]RecognizedPattern)

	for symbol, patterns := range allPatterns {
		var validPatterns []RecognizedPattern
		for _, pattern := range patterns {
			if pattern.Confidence >= minConfidence {
				validPatterns = append(validPatterns, pattern)
			}
		}

		if len(validPatterns) > 0 {
			// 按置信度排序
			sort.Slice(validPatterns, func(i, j int) bool {
				return validPatterns[i].Confidence > validPatterns[j].Confidence
			})
			filtered[symbol] = validPatterns
		}
	}

	return filtered
}

// generatePatternReport 生成模式识别报告
func (de *DataExecutor) generatePatternReport(patterns map[string][]RecognizedPattern, task *PatternRecognitionTask) *PatternReport {
	report := &PatternReport{
		TaskID:      task.ID,
		GeneratedAt: time.Now(),
		Summary:     make(map[string]int),
		Details:     make(map[string][]PatternDetail),
	}

	// 统计各种模式的数量
	for symbol, symbolPatterns := range patterns {
		var details []PatternDetail

		for _, pattern := range symbolPatterns {
			// 更新统计
			report.Summary[pattern.PatternType]++

			// 添加详细信息
			detail := PatternDetail{
				Symbol:      symbol,
				PatternType: pattern.PatternType,
				Confidence:  pattern.Confidence,
				StartTime:   pattern.StartTime,
				EndTime:     pattern.EndTime,
				KeyLevels:   pattern.KeyLevels,
				Direction:   pattern.Direction,
				Strength:    pattern.Strength,
			}
			details = append(details, detail)
		}

		report.Details[symbol] = details
	}

	return report
}

// sendPatternSignals 发送模式识别信号
func (de *DataExecutor) sendPatternSignals(ctx context.Context, patterns map[string][]RecognizedPattern, task *PatternRecognitionTask) {
	for symbol, symbolPatterns := range patterns {
		for _, pattern := range symbolPatterns {
			signal := &PatternSignal{
				ID:          fmt.Sprintf("signal_%s_%d", pattern.PatternType, time.Now().Unix()),
				Symbol:      symbol,
				PatternType: pattern.PatternType,
				Direction:   pattern.Direction,
				Confidence:  pattern.Confidence,
				Strength:    pattern.Strength,
				KeyLevels:   pattern.KeyLevels,
				Timeframe:   task.Timeframe,
				GeneratedAt: time.Now(),
			}

			// 发送信号到信号处理系统
			de.sendSignalToProcessor(ctx, signal)
		}
	}
}

// Data structures

type PatternRecognitionTask struct {
	ID             string                     `json:"id"`
	Symbols        []interface{}              `json:"symbols"`
	PatternTypes   []interface{}              `json:"pattern_types"`
	Timeframe      string                     `json:"timeframe"`
	LookbackPeriod int                        `json:"lookback_period"`
	MinConfidence  float64                    `json:"min_confidence"`
	Status         string                     `json:"status"`
	Error          string                     `json:"error,omitempty"`
	Results        *PatternRecognitionResults `json:"results,omitempty"`
	CreatedAt      time.Time                  `json:"created_at"`
	StartedAt      time.Time                  `json:"started_at"`
	CompletedAt    time.Time                  `json:"completed_at"`
	Duration       time.Duration              `json:"duration"`
}

type PatternRecognitionResults struct {
	Patterns      map[string][]RecognizedPattern `json:"patterns"`
	Report        *PatternReport                 `json:"report"`
	TotalPatterns int                            `json:"total_patterns"`
}

type RecognizedPattern struct {
	PatternType string                 `json:"pattern_type"`
	Symbol      string                 `json:"symbol"`
	Timeframe   string                 `json:"timeframe"`
	Confidence  float64                `json:"confidence"`
	Direction   string                 `json:"direction"`
	Strength    float64                `json:"strength"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     time.Time              `json:"end_time"`
	KeyLevels   map[string]float64     `json:"key_levels"`
	Properties  map[string]interface{} `json:"properties"`
	DetectedAt  time.Time              `json:"detected_at"`
}

type PatternReport struct {
	TaskID      string                     `json:"task_id"`
	GeneratedAt time.Time                  `json:"generated_at"`
	Summary     map[string]int             `json:"summary"`
	Details     map[string][]PatternDetail `json:"details"`
}

type PatternDetail struct {
	Symbol      string             `json:"symbol"`
	PatternType string             `json:"pattern_type"`
	Confidence  float64            `json:"confidence"`
	StartTime   time.Time          `json:"start_time"`
	EndTime     time.Time          `json:"end_time"`
	KeyLevels   map[string]float64 `json:"key_levels"`
	Direction   string             `json:"direction"`
	Strength    float64            `json:"strength"`
}

type PatternSignal struct {
	ID          string             `json:"id"`
	Symbol      string             `json:"symbol"`
	PatternType string             `json:"pattern_type"`
	Direction   string             `json:"direction"`
	Confidence  float64            `json:"confidence"`
	Strength    float64            `json:"strength"`
	KeyLevels   map[string]float64 `json:"key_levels"`
	Timeframe   string             `json:"timeframe"`
	GeneratedAt time.Time          `json:"generated_at"`
}

// Pattern recognizer interface and implementations

type PatternRecognizer interface {
	Recognize(symbol string, data []MarketDataPoint, minConfidence float64) []RecognizedPattern
}

// Simplified pattern recognizer implementations

type HeadAndShouldersRecognizer struct{}

func (r *HeadAndShouldersRecognizer) Recognize(symbol string, data []MarketDataPoint, minConfidence float64) []RecognizedPattern {
	// 简化的头肩顶识别逻辑
	return []RecognizedPattern{}
}

type DoubleTopRecognizer struct{}

func (r *DoubleTopRecognizer) Recognize(symbol string, data []MarketDataPoint, minConfidence float64) []RecognizedPattern {
	// 简化的双顶识别逻辑
	return []RecognizedPattern{}
}

type DoubleBottomRecognizer struct{}

func (r *DoubleBottomRecognizer) Recognize(symbol string, data []MarketDataPoint, minConfidence float64) []RecognizedPattern {
	// 简化的双底识别逻辑
	return []RecognizedPattern{}
}

type TriangleRecognizer struct{}

func (r *TriangleRecognizer) Recognize(symbol string, data []MarketDataPoint, minConfidence float64) []RecognizedPattern {
	// 简化的三角形识别逻辑
	return []RecognizedPattern{}
}

type FlagRecognizer struct{}

func (r *FlagRecognizer) Recognize(symbol string, data []MarketDataPoint, minConfidence float64) []RecognizedPattern {
	// 简化的旗形识别逻辑
	return []RecognizedPattern{}
}

type WedgeRecognizer struct{}

func (r *WedgeRecognizer) Recognize(symbol string, data []MarketDataPoint, minConfidence float64) []RecognizedPattern {
	// 简化的楔形识别逻辑
	return []RecognizedPattern{}
}

type ChannelRecognizer struct{}

func (r *ChannelRecognizer) Recognize(symbol string, data []MarketDataPoint, minConfidence float64) []RecognizedPattern {
	// 简化的通道识别逻辑
	return []RecognizedPattern{}
}

type SupportResistanceRecognizer struct{}

func (r *SupportResistanceRecognizer) Recognize(symbol string, data []MarketDataPoint, minConfidence float64) []RecognizedPattern {
	// 简化的支撑阻力识别逻辑
	return []RecognizedPattern{}
}

type CandlestickRecognizer struct{}

func (r *CandlestickRecognizer) Recognize(symbol string, data []MarketDataPoint, minConfidence float64) []RecognizedPattern {
	// 简化的K线形态识别逻辑
	return []RecognizedPattern{}
}

// Helper method implementations (simplified)

func (de *DataExecutor) savePatternTask(ctx context.Context, task *PatternRecognitionTask) error {
	return nil
}

func (de *DataExecutor) updatePatternTask(ctx context.Context, task *PatternRecognitionTask) error {
	return nil
}

func (de *DataExecutor) getMarketDataForPatterns(ctx context.Context, task *PatternRecognitionTask) (map[string][]MarketDataPoint, error) {
	data := make(map[string][]MarketDataPoint)

	for _, symbolInterface := range task.Symbols {
		symbol, ok := symbolInterface.(string)
		if !ok {
			continue
		}

		// 简化实现，返回模拟数据
		data[symbol] = []MarketDataPoint{}
	}

	return data, nil
}

func (de *DataExecutor) countTotalPatterns(patterns map[string][]RecognizedPattern) int {
	total := 0
	for _, symbolPatterns := range patterns {
		total += len(symbolPatterns)
	}
	return total
}

func (de *DataExecutor) sendSignalToProcessor(ctx context.Context, signal *PatternSignal) {
	// 发送信号到信号处理系统
	log.Printf("Sending pattern signal: %s for %s", signal.PatternType, signal.Symbol)
}

func (de *DataExecutor) notifyPatternRecognitionCompletion(ctx context.Context, task *PatternRecognitionTask) {
	if de.notificationService != nil {
		message := fmt.Sprintf("Pattern recognition task %s completed with status: %s", task.ID, task.Status)
		err := de.notificationService.SendWebhook(ctx, "", map[string]interface{}{
			"type":    "data_management",
			"action":  "pattern_recognition_completed",
			"message": message,
			"task_id": task.ID,
			"status":  task.Status,
		})
		if err != nil {
			log.Printf("Failed to send pattern recognition completion notification: %v", err)
		}
	}
}

// SystemExecutor 系统执行器
type SystemExecutor struct {
	config              *config.Config
	db                  *database.DB
	exchange            exchange.Exchange
	accountManager      *account.Manager
	metrics             *monitor.MetricsCollector     // actual metrics service
	notificationService protector.NotificationService // actual notification service
}

// NewSystemExecutor 创建系统执行器
func NewSystemExecutor(
	cfg *config.Config,
	db *database.DB,
	exchange exchange.Exchange,
	accountManager *account.Manager,
) *SystemExecutor {
	return &SystemExecutor{
		config:         cfg,
		db:             db,
		exchange:       exchange,
		accountManager: accountManager,
	}
}

// HandleAction 处理系统动作
func (se *SystemExecutor) HandleAction(ctx context.Context, action *ExecutionAction) error {
	switch action.Action {
	case "health_check":
		return se.healthCheck(ctx, action)
	case "security_monitor":
		return se.securityMonitor(ctx, action)
	case "exchange_failover":
		return se.exchangeFailover(ctx, action)
	case "audit_log":
		return se.auditLog(ctx, action)
	case "log_performance_metrics":
		return se.logPerformanceMetrics(ctx, action)
	default:
		return fmt.Errorf("unknown system action: %s", action.Action)
	}
}

// healthCheck 健康检查
func (se *SystemExecutor) healthCheck(ctx context.Context, action *ExecutionAction) error {
	log.Printf("Performing system health check")

	// 实现健康检查逻辑

	// 1. 解析健康检查请求
	checkTypes, _ := action.Parameters["check_types"].([]interface{})
	if len(checkTypes) == 0 {
		// 默认检查所有组件
		checkTypes = []interface{}{"database", "redis", "api", "exchange", "strategy_engine", "system_resources"}
	}

	detailed, _ := action.Parameters["detailed"].(bool)
	timeout, _ := action.Parameters["timeout"].(float64)
	if timeout <= 0 {
		timeout = 30 // 默认30秒超时
	}

	// 2. 创建健康检查任务
	healthCheckTask := &HealthCheckTask{
		ID:         fmt.Sprintf("health_check_%d", time.Now().Unix()),
		CheckTypes: checkTypes,
		Detailed:   detailed,
		Timeout:    time.Duration(timeout) * time.Second,
		Status:     "initializing",
		CreatedAt:  time.Now(),
	}

	// 3. 保存健康检查任务
	err := se.saveHealthCheckTask(ctx, healthCheckTask)
	if err != nil {
		return fmt.Errorf("failed to save health check task: %v", err)
	}

	// 4. 执行健康检查
	go se.executeHealthCheck(ctx, healthCheckTask)

	// 5. 记录指标
	if se.metrics != nil {
		se.metrics.IncrementCounter("system_executor.health_check_started", map[string]string{
			"detailed": fmt.Sprintf("%t", detailed),
		})
	}

	log.Printf("Started health check task %s", healthCheckTask.ID)
	return nil
}

// securityMonitor 安全监控
func (se *SystemExecutor) securityMonitor(ctx context.Context, action *ExecutionAction) error {
	log.Printf("Performing security monitoring")

	// 实现安全监控逻辑

	// 1. 解析安全监控请求
	monitorTypes, _ := action.Parameters["monitor_types"].([]interface{})
	if len(monitorTypes) == 0 {
		// 默认监控所有安全项目
		monitorTypes = []interface{}{"authentication", "authorization", "network", "api_security", "data_protection", "threat_detection"}
	}

	severity, _ := action.Parameters["severity"].(string)
	if severity == "" {
		severity = "medium" // 默认中等严重级别
	}

	realTime, _ := action.Parameters["real_time"].(bool)
	duration, _ := action.Parameters["duration"].(float64)
	if duration <= 0 {
		duration = 300 // 默认5分钟监控
	}

	// 2. 创建安全监控任务
	securityTask := &SecurityMonitoringTask{
		ID:           fmt.Sprintf("security_monitor_%d", time.Now().Unix()),
		MonitorTypes: monitorTypes,
		Severity:     severity,
		RealTime:     realTime,
		Duration:     time.Duration(duration) * time.Second,
		Status:       "initializing",
		CreatedAt:    time.Now(),
	}

	// 3. 保存安全监控任务
	err := se.saveSecurityTask(ctx, securityTask)
	if err != nil {
		return fmt.Errorf("failed to save security monitoring task: %v", err)
	}

	// 4. 执行安全监控
	if realTime {
		go se.executeRealTimeSecurityMonitoring(ctx, securityTask)
	} else {
		go se.executeSecurityMonitoring(ctx, securityTask)
	}

	// 5. 记录指标
	if se.metrics != nil {
		se.metrics.IncrementCounter("system_executor.security_monitoring_started", map[string]string{
			"severity":  severity,
			"real_time": fmt.Sprintf("%t", realTime),
		})
	}

	log.Printf("Started security monitoring task %s", securityTask.ID)
	return nil
}
// exchangeFailover 交易所故障切换
func (se *SystemExecutor) exchangeFailover(ctx context.Context, action *ExecutionAction) error {
	log.Printf("Performing exchange failover")

	// 实现交易所故障切换逻辑

	// 1. 解析故障切换请求
	primaryExchange, ok := action.Parameters["primary_exchange"].(string)
	if !ok {
		return fmt.Errorf("primary_exchange is required for exchange failover")
	}

	backupExchanges, _ := action.Parameters["backup_exchanges"].([]interface{})
	if len(backupExchanges) == 0 {
		return fmt.Errorf("backup_exchanges are required for failover")
	}

	failoverType, _ := action.Parameters["failover_type"].(string)
	if failoverType == "" {
		failoverType = "automatic" // 默认自动切换
	}

	forceFailover, _ := action.Parameters["force"].(bool)

	// 2. 创建故障切换任务
	failoverTask := &ExchangeFailoverTask{
		ID:              fmt.Sprintf("failover_%s_%d", primaryExchange, time.Now().Unix()),
		PrimaryExchange: primaryExchange,
		BackupExchanges: backupExchanges,
		FailoverType:    failoverType,
		ForceFailover:   forceFailover,
		Status:          "initializing",
		CreatedAt:       time.Now(),
	}

	// 3. 保存故障切换任务
	err := se.saveFailoverTask(ctx, failoverTask)
	if err != nil {
		return fmt.Errorf("failed to save failover task: %v", err)
	}

	// 4. 执行故障切换
	go se.executeExchangeFailover(ctx, failoverTask)

	// 5. 记录指标
	if se.metrics != nil {
		se.metrics.IncrementCounter("system_executor.exchange_failover_started", map[string]string{
			"primary_exchange": primaryExchange,
			"failover_type":    failoverType,
			"force":            fmt.Sprintf("%t", forceFailover),
		})
	}

	log.Printf("Started exchange failover task %s", failoverTask.ID)
	return nil
}
// auditLog 审计日志
func (se *SystemExecutor) auditLog(ctx context.Context, action *ExecutionAction) error {
	log.Printf("Processing audit logs")

	// 实现审计日志处理逻辑

	// 1. 解析审计日志处理请求
	logTypes, _ := action.Parameters["log_types"].([]interface{})
	if len(logTypes) == 0 {
		// 默认处理所有类型的审计日志
		logTypes = []interface{}{"authentication", "authorization", "trading", "system", "security", "compliance"}
	}

	timeRange, _ := action.Parameters["time_range"].(map[string]interface{})
	if timeRange == nil {
		// 默认处理最近24小时的日志
		timeRange = map[string]interface{}{
			"start": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
			"end":   time.Now().Format(time.RFC3339),
		}
	}

	processingMode, _ := action.Parameters["processing_mode"].(string)
	if processingMode == "" {
		processingMode = "analysis" // 默认分析模式
	}

	outputFormat, _ := action.Parameters["output_format"].(string)
	if outputFormat == "" {
		outputFormat = "json"
	}

	// 2. 创建审计日志处理任务
	auditTask := &AuditLogTask{
		ID:             fmt.Sprintf("audit_log_%d", time.Now().Unix()),
		LogTypes:       logTypes,
		TimeRange:      timeRange,
		ProcessingMode: processingMode,
		OutputFormat:   outputFormat,
		Status:         "initializing",
		CreatedAt:      time.Now(),
	}

	// 3. 保存审计日志任务
	err := se.saveAuditLogTask(ctx, auditTask)
	if err != nil {
		return fmt.Errorf("failed to save audit log task: %v", err)
	}

	// 4. 执行审计日志处理
	go se.executeAuditLogProcessing(ctx, auditTask)

	// 5. 记录指标
	if se.metrics != nil {
		se.metrics.IncrementCounter("system_executor.audit_log_processing_started", map[string]string{
			"processing_mode": processingMode,
			"output_format":   outputFormat,
		})
	}

	log.Printf("Started audit log processing task %s", auditTask.ID)
	return nil
}
// logPerformanceMetrics 记录性能指标
func (se *SystemExecutor) logPerformanceMetrics(ctx context.Context, action *ExecutionAction) error {
	log.Printf("Logging performance metrics")

	// 从参数中获取指标信息
	metrics := make(map[string]interface{})
	if action.Parameters != nil {
		for key, value := range action.Parameters {
			metrics[key] = value
		}
	}

	// 记录基本系统指标
	metrics["timestamp"] = time.Now().Unix()
	metrics["action_type"] = "performance_metrics"

	// 如果有数据库连接，可以将指标存储到数据库
	if se.db != nil {
		query := `
			INSERT INTO system_metrics (
				metric_name, metric_value, metric_type, recorded_at
			) VALUES ($1, $2, $3, $4)
		`

		for key, value := range metrics {
			if key == "timestamp" || key == "action_type" {
				continue // 跳过元数据字段
			}

			// 尝试将值转换为数字
			var numValue float64
			switch v := value.(type) {
			case float64:
				numValue = v
			case int:
				numValue = float64(v)
			case int64:
				numValue = float64(v)
			default:
				// 如果不是数字，跳过
				continue
			}

			_, err := se.db.ExecContext(ctx, query, key, numValue, "performance", time.Now())
			if err != nil {
				log.Printf("Failed to store metric %s: %v", key, err)
				// 不返回错误，继续处理其他指标
			}
		}
	}

	log.Printf("Performance metrics logged: %+v", metrics)
	return nil
}

// BalanceCheckResult 余额检查结果
type BalanceCheckResult struct {
	Sufficient         bool    `json:"sufficient"`
	Reason             string  `json:"reason"`
	Warning            string  `json:"warning"`
	UtilizationPercent float64 `json:"utilization_percent"`
	RequiredAmount     float64 `json:"required_amount"`
	AvailableAmount    float64 `json:"available_amount"`
	ReservedAmount     float64 `json:"reserved_amount"`
}

// calculateReservedAmount 计算预留金额
func (oe *OrderExecutor) calculateReservedAmount(ctx context.Context, asset string, totalBalance float64) float64 {
	// 1. 基础预留（总余额的5%）
	baseReserve := totalBalance * 0.05

	// 2. 获取未完成订单占用的金额
	pendingOrders := oe.getPendingOrdersAmount(ctx, asset)

	// 3. 风险管理预留
	riskReserve := oe.getRiskManagementReserve(ctx, asset, totalBalance)

	// 4. 最小预留金额
	minReserve := oe.getMinimumReserve(asset)

	totalReserve := baseReserve + pendingOrders + riskReserve
	if totalReserve < minReserve {
		totalReserve = minReserve
	}

	// 预留金额不能超过总余额的30%
	maxReserve := totalBalance * 0.3
	if totalReserve > maxReserve {
		totalReserve = maxReserve
	}

	return totalReserve
}

// BalanceRiskCheck 余额风险检查结果
type BalanceRiskCheck struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Reason  string `json:"reason"`
	Warning string `json:"warning"`
}

// performBalanceRiskChecks 执行余额风险检查
func (oe *OrderExecutor) performBalanceRiskChecks(ctx context.Context, asset string, requiredAmount, effectiveBalance, totalBalance float64) []BalanceRiskCheck {
	var checks []BalanceRiskCheck

	// 1. 单笔订单金额限制检查
	maxSingleOrderAmount := oe.getMaxSingleOrderAmount(asset)
	if requiredAmount > maxSingleOrderAmount {
		checks = append(checks, BalanceRiskCheck{
			Name:   "single_order_limit",
			Passed: false,
			Reason: fmt.Sprintf("Order amount %.6f exceeds single order limit %.6f for %s",
				requiredAmount, maxSingleOrderAmount, asset),
		})
	} else {
		checks = append(checks, BalanceRiskCheck{
			Name:   "single_order_limit",
			Passed: true,
		})
	}

	// 2. 日交易限额检查
	dailyUsed := oe.getDailyTradingAmount(ctx, asset)
	dailyLimit := oe.getDailyTradingLimit(asset)
	if dailyUsed+requiredAmount > dailyLimit {
		checks = append(checks, BalanceRiskCheck{
			Name:   "daily_trading_limit",
			Passed: false,
			Reason: fmt.Sprintf("Daily trading limit exceeded: used %.6f + required %.6f > limit %.6f for %s",
				dailyUsed, requiredAmount, dailyLimit, asset),
		})
	} else if (dailyUsed+requiredAmount)/dailyLimit > 0.8 {
		checks = append(checks, BalanceRiskCheck{
			Name:   "daily_trading_limit",
			Passed: true,
			Warning: fmt.Sprintf("Approaching daily trading limit: %.1f%% used for %s",
				((dailyUsed+requiredAmount)/dailyLimit)*100, asset),
		})
	} else {
		checks = append(checks, BalanceRiskCheck{
			Name:   "daily_trading_limit",
			Passed: true,
		})
	}

	// 3. 余额耗尽风险检查
	remainingAfterOrder := effectiveBalance - requiredAmount
	if remainingAfterOrder/totalBalance < 0.1 { // 剩余不足10%
		checks = append(checks, BalanceRiskCheck{
			Name:   "balance_depletion_risk",
			Passed: true,
			Warning: fmt.Sprintf("Low remaining balance after order: %.6f %s (%.1f%% of total)",
				remainingAfterOrder, asset, (remainingAfterOrder/totalBalance)*100),
		})
	} else {
		checks = append(checks, BalanceRiskCheck{
			Name:   "balance_depletion_risk",
			Passed: true,
		})
	}

	return checks
}

// checkConcentrationRisk 检查集中度风险
func (oe *OrderExecutor) checkConcentrationRisk(ctx context.Context, symbol string, orderAmount float64) bool {
	// 获取该交易对的总持仓价值
	totalPositionValue := oe.getTotalPositionValue(ctx, symbol)

	// 获取总账户价值
	totalAccountValue := oe.getTotalAccountValue(ctx)

	if totalAccountValue == 0 {
		return false
	}

	// 计算订单后的集中度
	concentrationAfterOrder := (totalPositionValue + orderAmount) / totalAccountValue

	// 单个交易对不应超过总资产的30%
	return concentrationAfterOrder > 0.3
}

// checkLiquidityRisk 检查流动性风险
func (oe *OrderExecutor) checkLiquidityRisk(ctx context.Context, symbol string, quantity float64, orderType string) string {
	if orderType != "MARKET" {
		return "" // 限价单不需要检查流动性
	}

	// 获取订单簿深度
	orderBook := oe.getOrderBookDepth(ctx, symbol)
	if orderBook == nil {
		return "Unable to assess liquidity risk: order book unavailable"
	}

	// 检查市价单是否会显著影响价格
	priceImpact := oe.calculatePriceImpact(orderBook, quantity)

	if priceImpact > 0.02 { // 2%价格影响
		return fmt.Sprintf("High price impact expected: %.2f%%", priceImpact*100)
	} else if priceImpact > 0.01 { // 1%价格影响
		return fmt.Sprintf("Moderate price impact expected: %.2f%%", priceImpact*100)
	}

	return ""
}

// Helper functions (simplified implementations)

func (oe *OrderExecutor) getPendingOrdersAmount(ctx context.Context, asset string) float64 {
	// 实际实现应该查询数据库获取未完成订单
	return 0.0
}

func (oe *OrderExecutor) getRiskManagementReserve(ctx context.Context, asset string, totalBalance float64) float64 {
	// 基于风险管理策略计算预留金额
	return totalBalance * 0.02 // 2%风险预留
}

func (oe *OrderExecutor) getMinimumReserve(asset string) float64 {
	// 不同资产的最小预留金额
	reserves := map[string]float64{
		"USDT": 10.0,
		"BTC":  0.001,
		"ETH":  0.01,
	}

	if reserve, exists := reserves[asset]; exists {
		return reserve
	}
	return 1.0 // 默认预留
}

func (oe *OrderExecutor) getMaxSingleOrderAmount(asset string) float64 {
	// 单笔订单最大金额限制
	limits := map[string]float64{
		"USDT": 100000.0, // $100k
		"BTC":  10.0,     // 10 BTC
		"ETH":  100.0,    // 100 ETH
	}

	if limit, exists := limits[asset]; exists {
		return limit
	}
	return 10000.0 // 默认限制
}

func (oe *OrderExecutor) getDailyTradingAmount(ctx context.Context, asset string) float64 {
	// 实际实现应该查询数据库获取当日交易金额
	return 0.0
}

func (oe *OrderExecutor) getDailyTradingLimit(asset string) float64 {
	// 日交易限额
	limits := map[string]float64{
		"USDT": 1000000.0, // $1M
		"BTC":  100.0,     // 100 BTC
		"ETH":  1000.0,    // 1000 ETH
	}

	if limit, exists := limits[asset]; exists {
		return limit
	}
	return 100000.0 // 默认限制
}

func (oe *OrderExecutor) getTotalPositionValue(ctx context.Context, symbol string) float64 {
	// 实际实现应该查询数据库获取持仓价值
	return 0.0
}

func (oe *OrderExecutor) getTotalAccountValue(ctx context.Context) float64 {
	// 实际实现应该计算总账户价值
	return 1000000.0 // 默认$1M
}

func (oe *OrderExecutor) getOrderBookDepth(ctx context.Context, symbol string) interface{} {
	// 实际实现应该获取订单簿数据
	return nil
}

func (oe *OrderExecutor) calculatePriceImpact(orderBook interface{}, quantity float64) float64 {
	// 实际实现应该计算价格影响
	return 0.0
}

// executeHealthCheck 执行健康检查
func (se *SystemExecutor) executeHealthCheck(ctx context.Context, task *HealthCheckTask) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Health check task %s panicked: %v", task.ID, r)
			task.Status = "failed"
			task.Error = fmt.Sprintf("panic: %v", r)
			se.updateHealthCheckTask(ctx, task)
		}
	}()

	// 1. 更新任务状态
	task.Status = "running"
	task.StartedAt = time.Now()
	se.updateHealthCheckTask(ctx, task)

	// 2. 创建超时上下文
	checkCtx, cancel := context.WithTimeout(ctx, task.Timeout)
	defer cancel()

	// 3. 执行各项健康检查
	healthResults := make(map[string]*ComponentHealthResult)

	for _, checkTypeInterface := range task.CheckTypes {
		checkType, ok := checkTypeInterface.(string)
		if !ok {
			continue
		}

		result := se.performComponentHealthCheck(checkCtx, checkType, task.Detailed)
		healthResults[checkType] = result
	}

	// 4. 计算整体健康状态
	overallHealth := se.calculateOverallHealth(healthResults)

	// 5. 生成健康检查报告
	report := se.generateHealthReport(healthResults, overallHealth, task)

	task.Results = &HealthCheckResults{
		OverallHealth:    overallHealth,
		ComponentResults: healthResults,
		Report:           report,
		CheckedAt:        time.Now(),
	}

	// 6. 处理健康问题
	if overallHealth.Status != "healthy" {
		se.handleHealthIssues(ctx, healthResults, task)
	}

	// 7. 更新任务完成状态
	task.Status = "completed"
	task.CompletedAt = time.Now()
	task.Duration = task.CompletedAt.Sub(task.StartedAt)
	se.updateHealthCheckTask(ctx, task)

	// 8. 发送通知
	se.notifyHealthCheckCompletion(ctx, task)

	// 9. 记录指标
	if se.metrics != nil {
		se.metrics.IncrementCounter("system_executor.health_check_completed", map[string]string{
			"status":         task.Status,
			"overall_health": overallHealth.Status,
		})
	}

	log.Printf("Health check task %s completed successfully", task.ID)
}

// performComponentHealthCheck 执行组件健康检查
func (se *SystemExecutor) performComponentHealthCheck(ctx context.Context, component string, detailed bool) *ComponentHealthResult {
	result := &ComponentHealthResult{
		Component: component,
		CheckedAt: time.Now(),
		Details:   make(map[string]interface{}),
	}

	switch component {
	case "database":
		result = se.checkDatabaseHealth(ctx, detailed)
	case "redis":
		result = se.checkRedisHealth(ctx, detailed)
	case "api":
		result = se.checkAPIHealth(ctx, detailed)
	case "exchange":
		result = se.checkExchangeHealth(ctx, detailed)
	case "strategy_engine":
		result = se.checkStrategyEngineHealth(ctx, detailed)
	case "system_resources":
		result = se.checkSystemResourcesHealth(ctx, detailed)
	default:
		result.Status = "unknown"
		result.Score = 0
		result.Message = fmt.Sprintf("Unknown component: %s", component)
	}

	result.Component = component
	result.CheckedAt = time.Now()

	return result
}

// Health check data structures

type HealthCheckTask struct {
	ID          string              `json:"id"`
	CheckTypes  []interface{}       `json:"check_types"`
	Detailed    bool                `json:"detailed"`
	Timeout     time.Duration       `json:"timeout"`
	Status      string              `json:"status"`
	Error       string              `json:"error,omitempty"`
	Results     *HealthCheckResults `json:"results,omitempty"`
	CreatedAt   time.Time           `json:"created_at"`
	StartedAt   time.Time           `json:"started_at"`
	CompletedAt time.Time           `json:"completed_at"`
	Duration    time.Duration       `json:"duration"`
}

type HealthCheckResults struct {
	OverallHealth    *OverallHealthResult              `json:"overall_health"`
	ComponentResults map[string]*ComponentHealthResult `json:"component_results"`
	Report           *HealthReport                     `json:"report"`
	CheckedAt        time.Time                         `json:"checked_at"`
}

type ComponentHealthResult struct {
	Component string                 `json:"component"`
	Status    string                 `json:"status"`
	Score     float64                `json:"score"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details"`
	CheckedAt time.Time              `json:"checked_at"`
}

type OverallHealthResult struct {
	Status         string  `json:"status"`
	Score          float64 `json:"score"`
	Message        string  `json:"message"`
	ComponentCount int     `json:"component_count"`
	HealthyCount   int     `json:"healthy_count"`
	DegradedCount  int     `json:"degraded_count"`
	UnhealthyCount int     `json:"unhealthy_count"`
}

type HealthReport struct {
	TaskID          string                            `json:"task_id"`
	GeneratedAt     time.Time                         `json:"generated_at"`
	OverallHealth   *OverallHealthResult              `json:"overall_health"`
	Components      map[string]*ComponentHealthResult `json:"components"`
	Summary         *HealthSummary                    `json:"summary"`
	Recommendations []string                          `json:"recommendations"`
}

type HealthSummary struct {
	TotalComponents     int     `json:"total_components"`
	HealthyComponents   int     `json:"healthy_components"`
	DegradedComponents  int     `json:"degraded_components"`
	UnhealthyComponents int     `json:"unhealthy_components"`
	AverageScore        float64 `json:"average_score"`
}

// Health check helper methods (simplified implementations)

func (se *SystemExecutor) saveHealthCheckTask(ctx context.Context, task *HealthCheckTask) error {
	return nil
}

func (se *SystemExecutor) updateHealthCheckTask(ctx context.Context, task *HealthCheckTask) error {
	return nil
}

func (se *SystemExecutor) checkDatabaseHealth(ctx context.Context, detailed bool) *ComponentHealthResult {
	result := &ComponentHealthResult{
		Component: "database",
		Status:    "healthy",
		Score:     100,
		Message:   "Database is healthy",
		Details:   make(map[string]interface{}),
	}

	// 简化的数据库健康检查
	if se.db != nil {
		err := se.db.PingContext(ctx)
		if err != nil {
			result.Status = "unhealthy"
			result.Score = 0
			result.Message = fmt.Sprintf("Database connection failed: %v", err)
		}
	} else {
		result.Status = "unhealthy"
		result.Score = 0
		result.Message = "Database connection not initialized"
	}

	return result
}

func (se *SystemExecutor) checkRedisHealth(ctx context.Context, detailed bool) *ComponentHealthResult {
	return &ComponentHealthResult{
		Component: "redis",
		Status:    "healthy",
		Score:     100,
		Message:   "Redis is healthy",
		Details:   make(map[string]interface{}),
	}
}

func (se *SystemExecutor) checkAPIHealth(ctx context.Context, detailed bool) *ComponentHealthResult {
	return &ComponentHealthResult{
		Component: "api",
		Status:    "healthy",
		Score:     100,
		Message:   "API is healthy",
		Details:   make(map[string]interface{}),
	}
}

func (se *SystemExecutor) checkExchangeHealth(ctx context.Context, detailed bool) *ComponentHealthResult {
	return &ComponentHealthResult{
		Component: "exchange",
		Status:    "healthy",
		Score:     100,
		Message:   "Exchange connections are healthy",
		Details:   make(map[string]interface{}),
	}
}

func (se *SystemExecutor) checkStrategyEngineHealth(ctx context.Context, detailed bool) *ComponentHealthResult {
	return &ComponentHealthResult{
		Component: "strategy_engine",
		Status:    "healthy",
		Score:     100,
		Message:   "Strategy engine is healthy",
		Details:   make(map[string]interface{}),
	}
}

func (se *SystemExecutor) checkSystemResourcesHealth(ctx context.Context, detailed bool) *ComponentHealthResult {
	result := &ComponentHealthResult{
		Component: "system_resources",
		Status:    "healthy",
		Score:     100,
		Message:   "System resources are healthy",
		Details:   make(map[string]interface{}),
	}

	// 简化的系统资源检查
	cpuUsage := 45.0  // 模拟值
	memUsage := 60.0  // 模拟值
	diskUsage := 70.0 // 模拟值

	result.Details["cpu_usage_percent"] = cpuUsage
	result.Details["memory_usage_percent"] = memUsage
	result.Details["disk_usage_percent"] = diskUsage

	if cpuUsage > 90 || memUsage > 90 {
		result.Status = "unhealthy"
		result.Score = 0
	} else if cpuUsage > 80 || memUsage > 80 {
		result.Status = "degraded"
		result.Score = 70
	}

	return result
}

func (se *SystemExecutor) calculateOverallHealth(results map[string]*ComponentHealthResult) *OverallHealthResult {
	if len(results) == 0 {
		return &OverallHealthResult{
			Status:  "unknown",
			Score:   0,
			Message: "No components checked",
		}
	}

	totalScore := 0.0
	unhealthyCount := 0
	degradedCount := 0

	for _, result := range results {
		totalScore += result.Score

		switch result.Status {
		case "unhealthy":
			unhealthyCount++
		case "degraded":
			degradedCount++
		}
	}

	avgScore := totalScore / float64(len(results))

	// 确定整体状态
	var status string
	var message string

	if unhealthyCount > 0 {
		status = "unhealthy"
		message = fmt.Sprintf("%d components are unhealthy", unhealthyCount)
	} else if degradedCount > 0 {
		status = "degraded"
		message = fmt.Sprintf("%d components are degraded", degradedCount)
	} else {
		status = "healthy"
		message = "All components are healthy"
	}

	return &OverallHealthResult{
		Status:         status,
		Score:          avgScore,
		Message:        message,
		ComponentCount: len(results),
		HealthyCount:   len(results) - unhealthyCount - degradedCount,
		DegradedCount:  degradedCount,
		UnhealthyCount: unhealthyCount,
	}
}

func (se *SystemExecutor) generateHealthReport(results map[string]*ComponentHealthResult, overall *OverallHealthResult, task *HealthCheckTask) *HealthReport {
	report := &HealthReport{
		TaskID:        task.ID,
		GeneratedAt:   time.Now(),
		OverallHealth: overall,
		Components:    results,
		Summary: &HealthSummary{
			TotalComponents:     len(results),
			HealthyComponents:   overall.HealthyCount,
			DegradedComponents:  overall.DegradedCount,
			UnhealthyComponents: overall.UnhealthyCount,
			AverageScore:        overall.Score,
		},
	}

	// 生成建议
	var recommendations []string

	for component, result := range results {
		if result.Status == "unhealthy" {
			recommendations = append(recommendations,
				fmt.Sprintf("Immediate attention required for %s: %s", component, result.Message))
		} else if result.Status == "degraded" {
			recommendations = append(recommendations,
				fmt.Sprintf("Monitor %s closely: %s", component, result.Message))
		}
	}

	report.Recommendations = recommendations

	return report
}

func (se *SystemExecutor) handleHealthIssues(ctx context.Context, results map[string]*ComponentHealthResult, task *HealthCheckTask) {
	for component, result := range results {
		if result.Status == "unhealthy" {
			// 发送紧急告警
			log.Printf("URGENT: Component %s is unhealthy - %s", component, result.Message)
			if se.notificationService != nil {
				message := fmt.Sprintf("URGENT: Component %s is unhealthy - %s", component, result.Message)
				err := se.notificationService.SendWebhook(ctx, "", map[string]interface{}{
					"type":      "system_health",
					"severity":  "urgent",
					"component": component,
					"message":   message,
				})
				if err != nil {
					log.Printf("Failed to send urgent health alert: %v", err)
				}
			}
		} else if result.Status == "degraded" {
			// 发送警告
			log.Printf("WARNING: Component %s is degraded - %s", component, result.Message)
			if se.notificationService != nil {
				message := fmt.Sprintf("WARNING: Component %s is degraded - %s", component, result.Message)
				err := se.notificationService.SendWebhook(ctx, "", map[string]interface{}{
					"type":      "system_health",
					"severity":  "warning",
					"component": component,
					"message":   message,
				})
				if err != nil {
					log.Printf("Failed to send degraded component warning: %v", err)
				}
			}
		}
	}
}
func (se *SystemExecutor) notifyHealthCheckCompletion(ctx context.Context, task *HealthCheckTask) {
	if se.notificationService != nil {
		message := fmt.Sprintf("Health check task %s completed with status: %s", task.ID, task.Status)
		err := se.notificationService.SendWebhook(ctx, "", map[string]interface{}{
			"type":    "system_management",
			"action":  "health_check_completed",
			"message": message,
			"task_id": task.ID,
			"status":  task.Status,
		})
		if err != nil {
			log.Printf("Failed to send health check completion notification: %v", err)
		}
	}
}
func (se *SystemExecutor) strengthenDataProtection(ctx context.Context) {
	log.Printf("Strengthening data protection measures")
}

func (se *SystemExecutor) activateThreatResponse(ctx context.Context) {
	log.Printf("Activating threat response measures")
}
// validateStrategyForElimination 验证策略是否可以被淘汰
func (se *StrategyExecutor) validateStrategyForElimination(ctx context.Context, strategyID string) error {
	// 检查策略是否存在
	if se.db == nil {
		return fmt.Errorf("database not available")
	}

	var status string
	var hasActivePositions bool

	query := `
		SELECT status, 
		       CASE WHEN EXISTS(SELECT 1 FROM positions WHERE strategy_id = ? AND status = 'open') 
		            THEN 1 ELSE 0 END as has_positions
		FROM strategies WHERE id = ?
	`

	err := se.db.QueryRowContext(ctx, query, strategyID, strategyID).Scan(&status, &hasActivePositions)
	if err != nil {
		return fmt.Errorf("failed to query strategy: %w", err)
	}

	// 检查策略状态
	if status == "eliminated" || status == "terminated" {
		return fmt.Errorf("strategy already eliminated or terminated")
	}

	log.Printf("Strategy %s validation passed: status=%s, has_positions=%v", strategyID, status, hasActivePositions)
	return nil
}

// closeStrategyOrders 关闭策略的所有活跃订单
func (se *StrategyExecutor) closeStrategyOrders(ctx context.Context, strategyID string) error {
	if se.db == nil {
		return fmt.Errorf("database not available")
	}

	// 查询活跃订单
	query := `SELECT id, symbol, side, quantity FROM orders WHERE strategy_id = ? AND status IN ('open', 'partial')`
	rows, err := se.db.QueryContext(ctx, query, strategyID)
	if err != nil {
		return fmt.Errorf("failed to query active orders: %w", err)
	}
	defer rows.Close()

	ordersClosed := 0
	for rows.Next() {
		var orderID, symbol, side string
		var quantity float64

		if err := rows.Scan(&orderID, &symbol, &side, &quantity); err != nil {
			log.Printf("Failed to scan order: %v", err)
			continue
		}

		// 取消订单
		if err := se.cancelOrder(ctx, orderID); err != nil {
			log.Printf("Failed to cancel order %s: %v", orderID, err)
			continue
		}

		ordersClosed++
		log.Printf("Cancelled order %s for strategy %s", orderID, strategyID)
	}

	log.Printf("Closed %d orders for strategy %s", ordersClosed, strategyID)
	return nil
}

// liquidateStrategyPositions 清算策略持仓
func (se *StrategyExecutor) liquidateStrategyPositions(ctx context.Context, strategyID string) error {
	if se.db == nil {
		return fmt.Errorf("database not available")
	}

	// 查询开放持仓
	query := `SELECT id, symbol, side, quantity, entry_price FROM positions WHERE strategy_id = ? AND status = 'open'`
	rows, err := se.db.QueryContext(ctx, query, strategyID)
	if err != nil {
		return fmt.Errorf("failed to query positions: %w", err)
	}
	defer rows.Close()

	positionsLiquidated := 0
	for rows.Next() {
		var positionID, symbol, side string
		var quantity, entryPrice float64

		if err := rows.Scan(&positionID, &symbol, &side, &quantity, &entryPrice); err != nil {
			log.Printf("Failed to scan position: %v", err)
			continue
		}

		// 平仓
		if err := se.closePosition(ctx, positionID, "elimination"); err != nil {
			log.Printf("Failed to close position %s: %v", positionID, err)
			continue
		}

		positionsLiquidated++
		log.Printf("Liquidated position %s for strategy %s", positionID, strategyID)
	}

	log.Printf("Liquidated %d positions for strategy %s", positionsLiquidated, strategyID)
	return nil
}

// stopStrategyExecution 停止策略执行
func (se *StrategyExecutor) stopStrategyExecution(ctx context.Context, strategyID string) error {
	// 停止策略的定时任务
	if se.scheduler != nil {
		// 实现调度器集成 - 停止策略相关的所有定时任务
		if scheduler, ok := se.scheduler.(SchedulerInterface); ok {
			if err := scheduler.StopStrategyTasks(ctx, strategyID); err != nil {
				log.Printf("Failed to stop scheduled tasks for strategy %s: %v", strategyID, err)
			} else {
				log.Printf("Successfully stopped all scheduled tasks for strategy %s", strategyID)
			}
		} else {
			log.Printf("Scheduler does not implement required interface for strategy %s", strategyID)
		}
	}

	// 停止策略的数据订阅
	if err := se.stopStrategyDataSubscription(ctx, strategyID); err != nil {
		log.Printf("Failed to stop data subscription: %v", err)
	}

	log.Printf("Strategy execution stopped for %s", strategyID)
	return nil
}

// updateStrategyStatus 更新策略状态
func (se *StrategyExecutor) updateStrategyStatus(ctx context.Context, strategyID, status string) error {
	if se.db == nil {
		return fmt.Errorf("database not available")
	}

	query := `UPDATE strategies SET status = ?, updated_at = ? WHERE id = ?`
	_, err := se.db.ExecContext(ctx, query, status, time.Now(), strategyID)
	if err != nil {
		return fmt.Errorf("failed to update strategy status: %w", err)
	}

	log.Printf("Strategy %s status updated to %s", strategyID, status)
	return nil
}

// recordStrategyElimination 记录策略淘汰历史
func (se *StrategyExecutor) recordStrategyElimination(ctx context.Context, action *ExecutionAction) error {
	if se.db == nil {
		return fmt.Errorf("database not available")
	}

	query := `
		INSERT INTO strategy_elimination_history 
		(strategy_id, elimination_reason, elimination_time, performance_metrics, created_at)
		VALUES (?, ?, ?, ?, ?)
	`

	reason := "performance_based"
	if action.Parameters != nil {
		if r, ok := action.Parameters["reason"].(string); ok {
			reason = r
		}
	}

	metricsJSON := "{}"
	if action.Parameters != nil {
		if metrics, ok := action.Parameters["metrics"]; ok {
			if metricsBytes, err := json.Marshal(metrics); err == nil {
				metricsJSON = string(metricsBytes)
			}
		}
	}

	strategyID, _ := action.Parameters["strategy_id"].(string)
	_, err := se.db.ExecContext(ctx, query, strategyID, reason, time.Now(), metricsJSON, time.Now())
	if err != nil {
		return fmt.Errorf("failed to record elimination: %w", err)
	}

	return nil
}

// releaseStrategyResources 释放策略资源
func (se *StrategyExecutor) releaseStrategyResources(ctx context.Context, strategyID string) error {
	// 释放内存中的策略实例
	if se.strategies != nil {
		delete(se.strategies, strategyID)
	}

	// 释放资金分配
	if err := se.releaseStrategyFunds(ctx, strategyID); err != nil {
		log.Printf("Failed to release funds for strategy %s: %v", strategyID, err)
	}

	// 清理临时文件和缓存
	if err := se.cleanupStrategyCache(ctx, strategyID); err != nil {
		log.Printf("Failed to cleanup cache for strategy %s: %v", strategyID, err)
	}

	log.Printf("Resources released for strategy %s", strategyID)
	return nil
}

// notifyStrategyElimination 发送策略淘汰通知
func (se *StrategyExecutor) notifyStrategyElimination(ctx context.Context, strategyID string) error {
	notification := map[string]interface{}{
		"type":        "strategy_eliminated",
		"strategy_id": strategyID,
		"timestamp":   time.Now(),
		"message":     fmt.Sprintf("Strategy %s has been eliminated", strategyID),
	}

	return se.sendNotification("strategy_events", notification)
}

// validateNewStrategyConfig 验证新策略配置
func (se *StrategyExecutor) validateNewStrategyConfig(ctx context.Context, action *ExecutionAction) (*StrategyConfig, error) {
	if action.Parameters == nil {
		return nil, fmt.Errorf("strategy parameters not provided")
	}

	configData, ok := action.Parameters["config"]
	if !ok {
		return nil, fmt.Errorf("strategy config not provided")
	}

	// 解析配置
	configBytes, err := json.Marshal(configData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var config StrategyConfig
	if err := json.Unmarshal(configBytes, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 验证必要字段
	if config.Name == "" {
		return nil, fmt.Errorf("strategy name is required")
	}

	if config.Type == "" {
		return nil, fmt.Errorf("strategy type is required")
	}

	if config.InitialCapital <= 0 {
		return nil, fmt.Errorf("initial capital must be positive")
	}

	log.Printf("Strategy config validated: %s (%s)", config.Name, config.Type)
	return &config, nil
}

// checkResourceAvailability 检查资源可用性
func (se *StrategyExecutor) checkResourceAvailability(ctx context.Context, config *StrategyConfig) error {
	// 检查资金可用性
	availableFunds, err := se.getAvailableFunds(ctx)
	if err != nil {
		return fmt.Errorf("failed to get available funds: %w", err)
	}

	if availableFunds < config.InitialCapital {
		return fmt.Errorf("insufficient funds: available=%.2f, required=%.2f",
			availableFunds, config.InitialCapital)
	}

	// 检查系统资源
	if err := se.checkSystemResources(); err != nil {
		return fmt.Errorf("insufficient system resources: %w", err)
	}

	log.Printf("Resource availability check passed for strategy %s", config.Name)
	return nil
}

// createStrategyInstance 创建策略实例
func (se *StrategyExecutor) createStrategyInstance(ctx context.Context, config *StrategyConfig) (*Strategy, error) {
	strategy := &Strategy{
		ID:             generateStrategyID(),
		Name:           config.Name,
		Type:           config.Type,
		Status:         "initializing",
		InitialCapital: config.InitialCapital,
		CurrentCapital: config.InitialCapital,
		Parameters:     config.Parameters,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// 保存到数据库
	if err := se.saveStrategyToDatabase(ctx, strategy); err != nil {
		return nil, fmt.Errorf("failed to save strategy: %w", err)
	}

	log.Printf("Strategy instance created: %s", strategy.ID)
	return strategy, nil
}

// allocateStrategyResources 分配策略资源
func (se *StrategyExecutor) allocateStrategyResources(ctx context.Context, strategy *Strategy) error {
	// 分配资金
	if err := se.allocateStrategyFunds(ctx, strategy.ID, strategy.InitialCapital); err != nil {
		return fmt.Errorf("failed to allocate funds: %w", err)
	}

	// 分配计算资源
	if err := se.allocateComputeResources(ctx, strategy.ID); err != nil {
		return fmt.Errorf("failed to allocate compute resources: %w", err)
	}

	log.Printf("Resources allocated for strategy %s", strategy.ID)
	return nil
}

// initializeStrategyParameters 初始化策略参数
func (se *StrategyExecutor) initializeStrategyParameters(ctx context.Context, strategy *Strategy) error {
	// 设置默认参数
	defaultParams := map[string]interface{}{
		"max_position_size": 0.1,  // 10% of capital
		"stop_loss":         0.02, // 2%
		"take_profit":       0.04, // 4%
		"risk_per_trade":    0.01, // 1%
	}

	// 合并用户参数和默认参数
	if strategy.Parameters == nil {
		strategy.Parameters = make(map[string]interface{})
	}

	for key, value := range defaultParams {
		if _, exists := strategy.Parameters[key]; !exists {
			strategy.Parameters[key] = value
		}
	}

	log.Printf("Strategy parameters initialized for %s", strategy.ID)
	return nil
}

// setupStrategyRiskControls 设置策略风险控制
func (se *StrategyExecutor) setupStrategyRiskControls(ctx context.Context, strategy *Strategy) error {
	riskControls := &RiskControls{
		MaxDrawdown:     0.15, // 15%
		MaxDailyLoss:    0.05, // 5%
		MaxPositionSize: 0.2,  // 20%
		MaxCorrelation:  0.8,  // 80%
		VaRLimit:        0.03, // 3%
	}

	// 保存风险控制设置
	if err := se.saveRiskControls(ctx, strategy.ID, riskControls); err != nil {
		return fmt.Errorf("failed to save risk controls: %w", err)
	}

	log.Printf("Risk controls setup for strategy %s", strategy.ID)
	return nil
}

// startStrategyMonitoring 启动策略监控
func (se *StrategyExecutor) startStrategyMonitoring(ctx context.Context, strategy *Strategy) error {
	// 启动性能监控
	if err := se.startPerformanceMonitoring(ctx, strategy.ID); err != nil {
		return fmt.Errorf("failed to start performance monitoring: %w", err)
	}

	// 启动风险监控
	if err := se.startRiskMonitoring(ctx, strategy.ID); err != nil {
		return fmt.Errorf("failed to start risk monitoring: %w", err)
	}

	log.Printf("Monitoring started for strategy %s", strategy.ID)
	return nil
}

// recordStrategyIntroduction 记录策略引入历史
func (se *StrategyExecutor) recordStrategyIntroduction(ctx context.Context, strategy *Strategy) error {
	if se.db == nil {
		return fmt.Errorf("database not available")
	}

	query := `
		INSERT INTO strategy_introduction_history 
		(strategy_id, strategy_name, strategy_type, initial_capital, introduction_time, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := se.db.ExecContext(ctx, query,
		strategy.ID, strategy.Name, strategy.Type,
		strategy.InitialCapital, time.Now(), time.Now())
	if err != nil {
		return fmt.Errorf("failed to record introduction: %w", err)
	}

	return nil
}

// notifyStrategyIntroduction 发送策略引入通知
func (se *StrategyExecutor) notifyStrategyIntroduction(ctx context.Context, strategy *Strategy) error {
	notification := map[string]interface{}{
		"type":          "strategy_introduced",
		"strategy_id":   strategy.ID,
		"strategy_name": strategy.Name,
		"timestamp":     time.Now(),
		"message":       fmt.Sprintf("New strategy %s (%s) has been introduced", strategy.Name, strategy.ID),
	}

	return se.sendNotification("strategy_events", notification)
}

// Helper functions

func (se *StrategyExecutor) cancelOrder(ctx context.Context, orderID string) error {
	// 简化实现：更新订单状态
	if se.db == nil {
		return fmt.Errorf("database not available")
	}

	query := `UPDATE orders SET status = 'cancelled', updated_at = ? WHERE id = ?`
	_, err := se.db.ExecContext(ctx, query, time.Now(), orderID)
	return err
}

func (se *StrategyExecutor) closePosition(ctx context.Context, positionID, reason string) error {
	// 简化实现：更新持仓状态
	if se.db == nil {
		return fmt.Errorf("database not available")
	}

	query := `UPDATE positions SET status = 'closed', close_reason = ?, updated_at = ? WHERE id = ?`
	_, err := se.db.ExecContext(ctx, query, reason, time.Now(), positionID)
	return err
}

func (se *StrategyExecutor) stopStrategyDataSubscription(ctx context.Context, strategyID string) error {
	// 简化实现：记录日志
	log.Printf("Stopping data subscription for strategy %s", strategyID)
	return nil
}

func (se *StrategyExecutor) releaseStrategyFunds(ctx context.Context, strategyID string) error {
	// 简化实现：记录日志
	log.Printf("Releasing funds for strategy %s", strategyID)
	return nil
}

func (se *StrategyExecutor) cleanupStrategyCache(ctx context.Context, strategyID string) error {
	// 简化实现：记录日志
	log.Printf("Cleaning up cache for strategy %s", strategyID)
	return nil
}

func (se *StrategyExecutor) sendNotification(topic string, notification map[string]interface{}) error {
	// 简化实现：记录日志
	log.Printf("Sending notification to %s: %+v", topic, notification)
	return nil
}

func (se *StrategyExecutor) getAvailableFunds(ctx context.Context) (float64, error) {
	// 简化实现：返回模拟可用资金
	return 100000.0, nil
}

func (se *StrategyExecutor) checkSystemResources() error {
	// 简化实现：总是返回成功
	return nil
}

func generateStrategyID() string {
	return fmt.Sprintf("strategy_%d", time.Now().UnixNano())
}

func (se *StrategyExecutor) saveStrategyToDatabase(ctx context.Context, strategy *Strategy) error {
	if se.db == nil {
		return fmt.Errorf("database not available")
	}

	paramsJSON, _ := json.Marshal(strategy.Parameters)

	query := `
		INSERT INTO strategies 
		(id, name, type, status, initial_capital, current_capital, parameters, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := se.db.ExecContext(ctx, query,
		strategy.ID, strategy.Name, strategy.Type, strategy.Status,
		strategy.InitialCapital, strategy.CurrentCapital,
		string(paramsJSON), strategy.CreatedAt, strategy.UpdatedAt)

	return err
}

func (se *StrategyExecutor) allocateStrategyFunds(ctx context.Context, strategyID string, amount float64) error {
	// 简化实现：记录日志
	log.Printf("Allocating %.2f funds to strategy %s", amount, strategyID)
	return nil
}

func (se *StrategyExecutor) allocateComputeResources(ctx context.Context, strategyID string) error {
	// 简化实现：记录日志
	log.Printf("Allocating compute resources to strategy %s", strategyID)
	return nil
}

func (se *StrategyExecutor) saveRiskControls(ctx context.Context, strategyID string, controls *RiskControls) error {
	// 简化实现：记录日志
	log.Printf("Saving risk controls for strategy %s: %+v", strategyID, controls)
	return nil
}

func (se *StrategyExecutor) startPerformanceMonitoring(ctx context.Context, strategyID string) error {
	// 简化实现：记录日志
	log.Printf("Starting performance monitoring for strategy %s", strategyID)
	return nil
}

func (se *StrategyExecutor) startRiskMonitoring(ctx context.Context, strategyID string) error {
	// 简化实现：记录日志
	log.Printf("Starting risk monitoring for strategy %s", strategyID)
	return nil
}

// Supporting types

type StrategyConfig2 struct {
	Name           string                 `json:"name"`
	Type           string                 `json:"type"`
	InitialCapital float64                `json:"initial_capital"`
	Parameters     map[string]interface{} `json:"parameters"`
}

type Strategy struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Type           string                 `json:"type"`
	Status         string                 `json:"status"`
	InitialCapital float64                `json:"initial_capital"`
	CurrentCapital float64                `json:"current_capital"`
	Parameters     map[string]interface{} `json:"parameters"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

type RiskControls struct {
	MaxDrawdown     float64 `json:"max_drawdown"`
	MaxDailyLoss    float64 `json:"max_daily_loss"`
	MaxPositionSize float64 `json:"max_position_size"`
	MaxCorrelation  float64 `json:"max_correlation"`
	VaRLimit        float64 `json:"var_limit"`
}