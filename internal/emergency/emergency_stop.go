package emergency

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/exchange"
	"qcat/internal/exchange/order"
	"qcat/internal/orchestrator"
	"qcat/internal/stability"
	"qcat/internal/strategy/validation"
)

// EmergencyStopManager 紧急停止管理器
type EmergencyStopManager struct {
	db             *sql.DB
	gatekeeper     *validation.StrategyGatekeeper
	processManager *stability.ProcessManager
	orchestrator   *orchestrator.Orchestrator
	orderManager   *order.Manager
	exchange       exchange.Exchange
	mu             sync.RWMutex
	stopped        bool
	stopTime       time.Time
}

// NewEmergencyStopManager 创建紧急停止管理器
func NewEmergencyStopManager(db *sql.DB) *EmergencyStopManager {
	return &EmergencyStopManager{
		db:             db,
		gatekeeper:     validation.NewStrategyGatekeeperWithDB(db),
		processManager: stability.NewProcessManager(),
	}
}

// NewEmergencyStopManagerWithDeps 创建带依赖的紧急停止管理器
func NewEmergencyStopManagerWithDeps(db *sql.DB, processManager *stability.ProcessManager,
	orchestrator *orchestrator.Orchestrator, orderManager *order.Manager, exchange exchange.Exchange) *EmergencyStopManager {
	return &EmergencyStopManager{
		db:             db,
		gatekeeper:     validation.NewStrategyGatekeeperWithDB(db),
		processManager: processManager,
		orchestrator:   orchestrator,
		orderManager:   orderManager,
		exchange:       exchange,
	}
}

// EmergencyStopAllStrategies 紧急停止所有策略
func (esm *EmergencyStopManager) EmergencyStopAllStrategies(ctx context.Context, reason string) error {
	esm.mu.Lock()
	defer esm.mu.Unlock()

	if esm.stopped {
		return fmt.Errorf("emergency stop already activated at %v", esm.stopTime)
	}

	log.Printf("🚨 启动紧急停止所有策略，原因: %s", reason)

	// 1. 获取所有活跃策略
	activeStrategies, err := esm.getAllActiveStrategies(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active strategies: %w", err)
	}

	log.Printf("发现 %d 个活跃策略需要停止", len(activeStrategies))

	// 2. 并发停止所有策略
	var wg sync.WaitGroup
	errorChan := make(chan error, len(activeStrategies))

	for _, strategyID := range activeStrategies {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			if err := esm.stopSingleStrategy(ctx, id, reason); err != nil {
				errorChan <- fmt.Errorf("failed to stop strategy %s: %w", id, err)
			}
		}(strategyID)
	}

	// 等待所有策略停止完成
	wg.Wait()
	close(errorChan)

	// 收集错误
	var errors []error
	for err := range errorChan {
		errors = append(errors, err)
	}

	// 3. 设置全局紧急停止状态
	esm.stopped = true
	esm.stopTime = time.Now()

	// 4. 记录紧急停止事件
	if err := esm.recordEmergencyStopEvent(ctx, reason, len(activeStrategies), len(errors)); err != nil {
		log.Printf("记录紧急停止事件失败: %v", err)
	}

	if len(errors) > 0 {
		log.Printf("⚠️ 紧急停止完成，但有 %d 个策略停止失败", len(errors))
		return fmt.Errorf("emergency stop completed with %d errors: %v", len(errors), errors)
	}

	log.Printf("✅ 紧急停止完成，成功停止 %d 个策略", len(activeStrategies))
	return nil
}

// getAllActiveStrategies 获取所有活跃策略
func (esm *EmergencyStopManager) getAllActiveStrategies(ctx context.Context) ([]string, error) {
	query := `
		SELECT id FROM strategies 
		WHERE status = 'active' OR is_running = true
	`

	rows, err := esm.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var strategies []string
	for rows.Next() {
		var strategyID string
		if err := rows.Scan(&strategyID); err != nil {
			log.Printf("扫描策略ID失败: %v", err)
			continue
		}
		strategies = append(strategies, strategyID)
	}

	return strategies, rows.Err()
}

// stopSingleStrategy 停止单个策略
func (esm *EmergencyStopManager) stopSingleStrategy(ctx context.Context, strategyID string, reason string) error {
	log.Printf("停止策略: %s", strategyID)

	// 1. 通过守门员禁用策略
	if err := esm.gatekeeper.DisableStrategy(ctx, strategyID, fmt.Sprintf("Emergency Stop: %s", reason)); err != nil {
		return fmt.Errorf("failed to disable strategy via gatekeeper: %w", err)
	}

	// 2. 更新数据库状态
	if err := esm.updateStrategyStatus(ctx, strategyID, "emergency_stopped"); err != nil {
		return fmt.Errorf("failed to update strategy status: %w", err)
	}

	// 3. 强制停止相关进程
	if err := esm.forceStopStrategyProcesses(ctx, strategyID); err != nil {
		log.Printf("强制停止策略 %s 进程失败: %v", strategyID, err)
		// 不返回错误，继续执行其他清理操作
	}

	// 4. 取消待处理订单
	if err := esm.cancelPendingOrders(ctx, strategyID); err != nil {
		log.Printf("取消策略 %s 待处理订单失败: %v", strategyID, err)
		// 不返回错误，继续执行其他清理操作
	}

	log.Printf("策略 %s 已成功停止", strategyID)
	return nil
}

// updateStrategyStatus 更新策略状态
func (esm *EmergencyStopManager) updateStrategyStatus(ctx context.Context, strategyID string, status string) error {
	query := `
		UPDATE strategies 
		SET status = $1, is_running = false, updated_at = NOW()
		WHERE id = $2
	`

	_, err := esm.db.ExecContext(ctx, query, status, strategyID)
	return err
}

// forceStopStrategyProcesses 强制停止策略进程
func (esm *EmergencyStopManager) forceStopStrategyProcesses(ctx context.Context, strategyID string) error {
	log.Printf("强制停止策略 %s 的进程", strategyID)

	// 1. 通过进程管理器停止策略进程
	if esm.processManager != nil {
		// 停止策略相关的进程
		if err := esm.processManager.StopProcess(stability.ProcessTypeStrategy); err != nil {
			log.Printf("停止策略进程失败: %v", err)
			// 不返回错误，继续执行其他停止操作
		}
	}

	// 2. 通过编排器停止策略服务
	if esm.orchestrator != nil {
		serviceName := fmt.Sprintf("strategy-%s", strategyID)
		if err := esm.orchestrator.StopService(serviceName); err != nil {
			log.Printf("停止策略服务 %s 失败: %v", serviceName, err)
			// 不返回错误，继续执行其他停止操作
		}
	}

	// 3. 查询并停止策略相关的系统进程
	if err := esm.stopStrategySystemProcesses(ctx, strategyID); err != nil {
		log.Printf("停止策略系统进程失败: %v", err)
		// 不返回错误，继续执行其他停止操作
	}

	log.Printf("策略 %s 的进程停止操作完成", strategyID)
	return nil
}

// cancelPendingOrders 取消待处理订单
func (esm *EmergencyStopManager) cancelPendingOrders(ctx context.Context, strategyID string) error {
	log.Printf("取消策略 %s 的待处理订单", strategyID)

	// 1. 通过订单管理器取消订单
	if esm.orderManager != nil {
		// 获取策略的所有待处理订单
		if err := esm.cancelOrdersByStrategy(ctx, strategyID); err != nil {
			log.Printf("通过订单管理器取消订单失败: %v", err)
		}
	}

	// 2. 直接通过交易所API取消订单
	if esm.exchange != nil {
		if err := esm.cancelOrdersViaExchange(ctx, strategyID); err != nil {
			log.Printf("通过交易所API取消订单失败: %v", err)
		}
	}

	// 3. 从数据库查询并取消订单
	if err := esm.cancelOrdersFromDatabase(ctx, strategyID); err != nil {
		log.Printf("从数据库取消订单失败: %v", err)
	}

	log.Printf("策略 %s 的订单取消操作完成", strategyID)
	return nil
}

// recordEmergencyStopEvent 记录紧急停止事件
func (esm *EmergencyStopManager) recordEmergencyStopEvent(ctx context.Context, reason string, totalStrategies, failedCount int) error {
	query := `
		INSERT INTO emergency_stop_events (
			reason, total_strategies, failed_count, 
			stopped_at, created_at
		) VALUES ($1, $2, $3, $4, $5)
	`

	now := time.Now()
	_, err := esm.db.ExecContext(ctx, query, reason, totalStrategies, failedCount, now, now)
	return err
}

// IsEmergencyStopped 检查是否处于紧急停止状态
func (esm *EmergencyStopManager) IsEmergencyStopped() bool {
	esm.mu.RLock()
	defer esm.mu.RUnlock()
	return esm.stopped
}

// GetEmergencyStopTime 获取紧急停止时间
func (esm *EmergencyStopManager) GetEmergencyStopTime() time.Time {
	esm.mu.RLock()
	defer esm.mu.RUnlock()
	return esm.stopTime
}

// ResetEmergencyStop 重置紧急停止状态（谨慎使用）
func (esm *EmergencyStopManager) ResetEmergencyStop(ctx context.Context, reason string) error {
	esm.mu.Lock()
	defer esm.mu.Unlock()

	if !esm.stopped {
		return fmt.Errorf("emergency stop is not active")
	}

	log.Printf("重置紧急停止状态，原因: %s", reason)

	// 记录重置事件
	query := `
		INSERT INTO emergency_stop_reset_events (
			reason, reset_at, created_at
		) VALUES ($1, $2, $3)
	`

	now := time.Now()
	if _, err := esm.db.ExecContext(ctx, query, reason, now, now); err != nil {
		return fmt.Errorf("failed to record reset event: %w", err)
	}

	esm.stopped = false
	esm.stopTime = time.Time{}

	log.Printf("✅ 紧急停止状态已重置")
	return nil
}

// stopStrategySystemProcesses 停止策略相关的系统进程
func (esm *EmergencyStopManager) stopStrategySystemProcesses(ctx context.Context, strategyID string) error {
	// 查询策略相关的进程信息
	query := `
		SELECT process_id, process_name
		FROM strategy_processes
		WHERE strategy_id = $1 AND status = 'running'
	`

	rows, err := esm.db.QueryContext(ctx, query, strategyID)
	if err != nil {
		return fmt.Errorf("查询策略进程失败: %w", err)
	}
	defer rows.Close()

	var processIDs []string
	var processNames []string

	for rows.Next() {
		var processID, processName string
		if err := rows.Scan(&processID, &processName); err != nil {
			log.Printf("扫描进程信息失败: %v", err)
			continue
		}
		processIDs = append(processIDs, processID)
		processNames = append(processNames, processName)
	}

	// 停止找到的进程
	for i, processID := range processIDs {
		log.Printf("停止进程: %s (%s)", processNames[i], processID)

		// 更新进程状态为停止
		updateQuery := `
			UPDATE strategy_processes
			SET status = 'stopped', stopped_at = NOW()
			WHERE process_id = $1
		`
		if _, err := esm.db.ExecContext(ctx, updateQuery, processID); err != nil {
			log.Printf("更新进程状态失败: %v", err)
		}
	}

	return nil
}

// cancelOrdersByStrategy 通过订单管理器取消策略订单
func (esm *EmergencyStopManager) cancelOrdersByStrategy(ctx context.Context, strategyID string) error {
	// 查询策略的所有待处理订单
	query := `
		SELECT order_id, symbol
		FROM orders
		WHERE strategy_id = $1 AND status IN ('NEW', 'PARTIALLY_FILLED')
	`

	rows, err := esm.db.QueryContext(ctx, query, strategyID)
	if err != nil {
		return fmt.Errorf("查询策略订单失败: %w", err)
	}
	defer rows.Close()

	var orders []struct {
		OrderID string
		Symbol  string
	}

	for rows.Next() {
		var order struct {
			OrderID string
			Symbol  string
		}
		if err := rows.Scan(&order.OrderID, &order.Symbol); err != nil {
			log.Printf("扫描订单信息失败: %v", err)
			continue
		}
		orders = append(orders, order)
	}

	// 取消找到的订单
	for _, order := range orders {
		req := &exchange.OrderCancelRequest{
			Symbol:  order.Symbol,
			OrderID: order.OrderID,
		}

		if _, err := esm.orderManager.CancelOrder(ctx, req); err != nil {
			log.Printf("取消订单 %s 失败: %v", order.OrderID, err)
		} else {
			log.Printf("已取消订单: %s", order.OrderID)
		}
	}

	return nil
}

// cancelOrdersViaExchange 直接通过交易所API取消订单
func (esm *EmergencyStopManager) cancelOrdersViaExchange(ctx context.Context, strategyID string) error {
	// 查询策略使用的交易对
	query := `
		SELECT DISTINCT symbol
		FROM orders
		WHERE strategy_id = $1 AND status IN ('NEW', 'PARTIALLY_FILLED')
	`

	rows, err := esm.db.QueryContext(ctx, query, strategyID)
	if err != nil {
		return fmt.Errorf("查询策略交易对失败: %w", err)
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			log.Printf("扫描交易对失败: %v", err)
			continue
		}
		symbols = append(symbols, symbol)
	}

	// 为每个交易对取消所有订单
	for _, symbol := range symbols {
		if err := esm.exchange.CancelAllOrders(ctx, symbol); err != nil {
			log.Printf("取消交易对 %s 的所有订单失败: %v", symbol, err)
		} else {
			log.Printf("已取消交易对 %s 的所有订单", symbol)
		}
	}

	return nil
}

// cancelOrdersFromDatabase 从数据库取消订单
func (esm *EmergencyStopManager) cancelOrdersFromDatabase(ctx context.Context, strategyID string) error {
	// 更新数据库中的订单状态
	query := `
		UPDATE orders
		SET status = 'CANCELLED', updated_at = NOW()
		WHERE strategy_id = $1 AND status IN ('NEW', 'PARTIALLY_FILLED')
	`

	result, err := esm.db.ExecContext(ctx, query, strategyID)
	if err != nil {
		return fmt.Errorf("更新订单状态失败: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("已从数据库取消 %d 个订单", rowsAffected)

	return nil
}

// GetEmergencyStopStatus 获取紧急停止状态信息
func (esm *EmergencyStopManager) GetEmergencyStopStatus() map[string]interface{} {
	esm.mu.RLock()
	defer esm.mu.RUnlock()

	status := map[string]interface{}{
		"is_stopped": esm.stopped,
	}

	if esm.stopped {
		status["stop_time"] = esm.stopTime
		status["duration"] = time.Since(esm.stopTime).String()
	}

	return status
}
