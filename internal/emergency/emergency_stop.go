package emergency

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/strategy/validation"
)

// EmergencyStopManager 紧急停止管理器
type EmergencyStopManager struct {
	db         *sql.DB
	gatekeeper *validation.StrategyGatekeeper
	mu         sync.RWMutex
	stopped    bool
	stopTime   time.Time
}

// NewEmergencyStopManager 创建紧急停止管理器
func NewEmergencyStopManager(db *sql.DB) *EmergencyStopManager {
	return &EmergencyStopManager{
		db:         db,
		gatekeeper: validation.NewStrategyGatekeeperWithDB(db),
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
	// TODO: 实现进程停止逻辑
	// 这里需要与进程管理器集成
	log.Printf("强制停止策略 %s 的进程", strategyID)
	return nil
}

// cancelPendingOrders 取消待处理订单
func (esm *EmergencyStopManager) cancelPendingOrders(ctx context.Context, strategyID string) error {
	// TODO: 实现订单取消逻辑
	// 这里需要与交易所API集成
	log.Printf("取消策略 %s 的待处理订单", strategyID)
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
