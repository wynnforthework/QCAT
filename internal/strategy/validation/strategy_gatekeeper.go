package validation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"qcat/internal/strategy/lifecycle"
)

// StrategyGatekeeper 策略守门员 - 确保只有通过验证的策略才能启用
type StrategyGatekeeper struct {
	backtestValidator *MandatoryBacktestValidator
	riskValidator     *RiskValidator
	scenarioAuditor   *MarketScenarioAuditor
	enabled           bool
	blacklist         map[string]*BlacklistEntry
	mu                sync.RWMutex
	db                *sql.DB
}

// BlacklistEntry 黑名单条目
type BlacklistEntry struct {
	StrategyID string     `json:"strategy_id"`
	Reason     string     `json:"reason"`
	BlockedAt  time.Time  `json:"blocked_at"`
	BlockedBy  string     `json:"blocked_by"`
	Permanent  bool       `json:"permanent"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

// RiskValidator 风险验证器
type RiskValidator struct {
	maxPositionSize    float64 // 最大单个持仓大小
	maxTotalExposure   float64 // 最大总敞口
	maxDailyLoss       float64 // 最大日损失
	maxConsecutiveLoss int     // 最大连续亏损次数
}

// ValidationStatus 验证状态
type ValidationStatus struct {
	StrategyID          string               `json:"strategy_id"`
	IsValid             bool                 `json:"is_valid"`
	BacktestPassed      bool                 `json:"backtest_passed"`
	RiskCheckPassed     bool                 `json:"risk_check_passed"`
	ScenarioAuditPassed bool                 `json:"scenario_audit_passed"`
	ScenarioAuditResult *ScenarioAuditResult `json:"scenario_audit_result,omitempty"`
	ValidationTime      time.Time            `json:"validation_time"`
	BacktestResult      *BacktestResult      `json:"backtest_result,omitempty"`
	RiskAssessment      *RiskAssessment      `json:"risk_assessment,omitempty"`
	Errors              []ValidationError    `json:"errors,omitempty"`
	Warnings            []ValidationError    `json:"warnings,omitempty"`
	NextRevalidation    time.Time            `json:"next_revalidation"`
}

// RiskAssessment 风险评估
type RiskAssessment struct {
	RiskScore        float64  `json:"risk_score"`        // 0-100风险评分
	RiskLevel        string   `json:"risk_level"`        // LOW/MEDIUM/HIGH/CRITICAL
	MaxPositionSize  float64  `json:"max_position_size"` // 建议最大持仓
	MaxLeverage      float64  `json:"max_leverage"`      // 建议最大杠杆
	RecommendedLimit float64  `json:"recommended_limit"` // 建议资金限制
	Warnings         []string `json:"warnings"`
}

// NewStrategyGatekeeper 创建策略守门员
func NewStrategyGatekeeper() *StrategyGatekeeper {
	return &StrategyGatekeeper{
		backtestValidator: NewMandatoryBacktestValidator(),
		riskValidator: &RiskValidator{
			maxPositionSize:    0.1,  // 单个持仓不超过10%
			maxTotalExposure:   0.8,  // 总敞口不超过80%
			maxDailyLoss:       0.05, // 日损失不超过5%
			maxConsecutiveLoss: 5,    // 最多连续5次亏损
		},
		scenarioAuditor: NewMarketScenarioAuditor(),
		enabled:         true,
		blacklist:       make(map[string]*BlacklistEntry),
	}
}

// NewStrategyGatekeeperWithDB 创建带数据库连接的策略守门员
func NewStrategyGatekeeperWithDB(db *sql.DB) *StrategyGatekeeper {
	sg := NewStrategyGatekeeper()
	sg.db = db

	// 从数据库加载黑名单
	if err := sg.loadBlacklistFromDB(context.Background()); err != nil {
		log.Printf("从数据库加载黑名单失败: %v", err)
	}

	return sg
}

// ValidateStrategyForActivation 验证策略是否可以激活
func (sg *StrategyGatekeeper) ValidateStrategyForActivation(ctx context.Context, strategyID string, config *lifecycle.Version) (*ValidationStatus, error) {
	if !sg.enabled {
		log.Printf("策略守门员已禁用，跳过验证")
		return &ValidationStatus{
			StrategyID:     strategyID,
			IsValid:        true,
			ValidationTime: time.Now(),
		}, nil
	}

	log.Printf("开始验证策略 %s 是否可以激活", strategyID)

	// 首先检查黑名单
	if sg.isBlacklisted(strategyID) {
		log.Printf("策略 %s 在黑名单中，拒绝激活", strategyID)
		return &ValidationStatus{
			StrategyID:     strategyID,
			IsValid:        false,
			ValidationTime: time.Now(),
			Errors: []ValidationError{{
				Code:    "STRATEGY_BLACKLISTED",
				Message: "策略已被加入黑名单，禁止启动",
				Field:   "strategy_id",
			}},
		}, nil
	}

	status := &ValidationStatus{
		StrategyID:          strategyID,
		ValidationTime:      time.Now(),
		Errors:              make([]ValidationError, 0),
		Warnings:            make([]ValidationError, 0),
		ScenarioAuditPassed: true,
	}

	// 1. 强制回测验证
	log.Printf("执行强制回测验证...")
	backtestResult, err := sg.backtestValidator.ValidateStrategy(ctx, strategyID, config)
	if err != nil {
		status.BacktestPassed = false
		status.Errors = append(status.Errors, ValidationError{
			Code:    "BACKTEST_FAILED",
			Message: fmt.Sprintf("回测验证失败: %v", err),
			Field:   "backtest",
		})
		log.Printf("策略 %s 回测验证失败: %v", strategyID, err)
	} else {
		status.BacktestPassed = true
		status.BacktestResult = backtestResult
		log.Printf("策略 %s 回测验证通过", strategyID)
	}

	// 2. 风险评估
	log.Printf("执行风险评估...")
	riskAssessment, err := sg.assessRisk(ctx, strategyID, config, backtestResult)
	if err != nil {
		status.RiskCheckPassed = false
		status.Errors = append(status.Errors, ValidationError{
			Code:    "RISK_ASSESSMENT_FAILED",
			Message: fmt.Sprintf("风险评估失败: %v", err),
			Field:   "risk",
		})
		log.Printf("策略 %s 风险评估失败: %v", strategyID, err)
	} else {
		status.RiskAssessment = riskAssessment
		if riskAssessment.RiskLevel == "CRITICAL" {
			status.RiskCheckPassed = false
			status.Errors = append(status.Errors, ValidationError{
				Code:    "RISK_TOO_HIGH",
				Message: "策略风险等级过高，不允许启用",
				Field:   "risk_level",
			})
		} else {
			status.RiskCheckPassed = true
			if riskAssessment.RiskLevel == "HIGH" {
				status.Warnings = append(status.Warnings, ValidationError{
					Code:    "HIGH_RISK_WARNING",
					Message: "策略风险等级较高，建议谨慎使用",
					Field:   "risk_level",
				})
			}
		}
	}

	// 3. Market scenario audit
	if sg.scenarioAuditor != nil {
		log.Printf("Executing market scenario audit...")
		auditResult, auditErr := sg.scenarioAuditor.Evaluate(ctx, strategyID, config)
		if auditErr != nil {
			status.ScenarioAuditPassed = false
			status.Errors = append(status.Errors, ValidationError{
				Code:    "SCENARIO_AUDIT_ERROR",
				Message: fmt.Sprintf("scenario audit failed: %v", auditErr),
				Field:   "scenario_audit",
			})
			log.Printf("Strategy %s scenario audit failed: %v", strategyID, auditErr)
		} else {
			status.ScenarioAuditResult = auditResult
			status.ScenarioAuditPassed = auditResult.Passed
			if !auditResult.Passed {
				for _, item := range auditResult.ScenarioResults {
					if item.Passed {
						continue
					}
					message := item.Name
					if item.Summary != "" {
						message = item.Summary
					}
					status.Errors = append(status.Errors, ValidationError{
						Code:    "SCENARIO_CHECK_FAILED",
						Message: fmt.Sprintf("%s scenario failed: %s", item.Name, message),
						Field:   string(item.Scenario),
					})
					if len(item.Suggestions) > 0 {
						status.Warnings = append(status.Warnings, ValidationError{
							Code:    "SCENARIO_IMPROVEMENT",
							Message: strings.Join(item.Suggestions, "; "),
							Field:   string(item.Scenario),
						})
					}
				}
			}
		}
	} else {
		log.Printf("Scenario auditor not configured, skipping market scenario checks")
	}
	// 4. Comprehensive evaluation
	status.IsValid = status.BacktestPassed && status.RiskCheckPassed && status.ScenarioAuditPassed

	// 5. 设置下次重新验证时间
	if status.IsValid {
		status.NextRevalidation = time.Now().AddDate(0, 1, 0) // 1个月后重新验证
	} else {
		status.NextRevalidation = time.Now().AddDate(0, 0, 7) // 1周后可重新验证
	}

	if status.IsValid {
		log.Printf("策略 %s 验证通过，可以激活", strategyID)
	} else {
		log.Printf("策略 %s 验证失败，不能激活。错误: %d个，警告: %d个", strategyID, len(status.Errors), len(status.Warnings))
	}

	return status, nil
}

// RunScenarioAudit executes the market scenario audit independently.
func (sg *StrategyGatekeeper) RunScenarioAudit(ctx context.Context, strategyID string, config *lifecycle.Version) (*ScenarioAuditResult, error) {
	if sg.scenarioAuditor == nil {
		return nil, fmt.Errorf("scenario auditor not configured")
	}
	return sg.scenarioAuditor.Evaluate(ctx, strategyID, config)
}

// assessRisk 评估策略风险
func (sg *StrategyGatekeeper) assessRisk(ctx context.Context, strategyID string, config *lifecycle.Version, backtestResult *BacktestResult) (*RiskAssessment, error) {
	assessment := &RiskAssessment{
		Warnings: make([]string, 0),
	}

	var riskScore float64 = 0

	// 基于回测结果评估风险
	if backtestResult != nil {
		// 最大回撤风险
		if backtestResult.MaxDrawdown > 0.15 {
			riskScore += 30
			assessment.Warnings = append(assessment.Warnings, "最大回撤超过15%")
		} else if backtestResult.MaxDrawdown > 0.10 {
			riskScore += 15
		}

		// 夏普比率风险
		if backtestResult.SharpeRatio < 0.5 {
			riskScore += 25
			assessment.Warnings = append(assessment.Warnings, "夏普比率过低")
		} else if backtestResult.SharpeRatio < 1.0 {
			riskScore += 10
		}

		// 交易频率风险
		if backtestResult.TotalTrades > backtestResult.BacktestDays*5 {
			riskScore += 20
			assessment.Warnings = append(assessment.Warnings, "交易频率过高，可能存在过度交易")
		}

		// 胜率风险
		if backtestResult.WinRate < 0.4 {
			riskScore += 15
			assessment.Warnings = append(assessment.Warnings, "胜率过低")
		}
	} else {
		// 没有回测结果，风险很高
		riskScore += 50
		assessment.Warnings = append(assessment.Warnings, "缺少回测验证")
	}

	assessment.RiskScore = riskScore

	// 确定风险等级
	if riskScore >= 80 {
		assessment.RiskLevel = "CRITICAL"
	} else if riskScore >= 60 {
		assessment.RiskLevel = "HIGH"
	} else if riskScore >= 30 {
		assessment.RiskLevel = "MEDIUM"
	} else {
		assessment.RiskLevel = "LOW"
	}

	// 设置建议参数
	switch assessment.RiskLevel {
	case "LOW":
		assessment.MaxPositionSize = 0.1 // 10%
		assessment.MaxLeverage = 5.0
		assessment.RecommendedLimit = 100000 // $100k
	case "MEDIUM":
		assessment.MaxPositionSize = 0.05 // 5%
		assessment.MaxLeverage = 3.0
		assessment.RecommendedLimit = 50000 // $50k
	case "HIGH":
		assessment.MaxPositionSize = 0.02 // 2%
		assessment.MaxLeverage = 2.0
		assessment.RecommendedLimit = 10000 // $10k
	case "CRITICAL":
		assessment.MaxPositionSize = 0.01 // 1%
		assessment.MaxLeverage = 1.0
		assessment.RecommendedLimit = 1000 // $1k
	}

	return assessment, nil
}

// DisableStrategy 禁用策略（紧急情况）
func (sg *StrategyGatekeeper) DisableStrategy(ctx context.Context, strategyID string, reason string) error {
	log.Printf("紧急禁用策略 %s，原因: %s", strategyID, reason)

	// 添加到黑名单
	if err := sg.addToBlacklist(ctx, strategyID, reason); err != nil {
		log.Printf("添加策略 %s 到黑名单失败: %v", strategyID, err)
		return err
	}

	// 强制停止策略
	if err := sg.forceStopStrategy(ctx, strategyID, reason); err != nil {
		log.Printf("强制停止策略 %s 失败: %v", strategyID, err)
		return err
	}

	log.Printf("策略 %s 已被紧急禁用并加入黑名单", strategyID)
	return nil
}

// GetValidationHistory 获取验证历史
func (sg *StrategyGatekeeper) GetValidationHistory(ctx context.Context, strategyID string) ([]*ValidationStatus, error) {
	// 这里应该从数据库查询验证历史
	return nil, fmt.Errorf("not implemented")
}

// Enable 启用守门员
func (sg *StrategyGatekeeper) Enable() {
	sg.enabled = true
	log.Printf("策略守门员已启用")
}

// Disable 禁用守门员（仅用于紧急情况）
func (sg *StrategyGatekeeper) Disable() {
	sg.enabled = false
	log.Printf("警告: 策略守门员已禁用！所有策略将跳过验证")
}

// AddToBlacklist 添加策略到黑名单 (exported)
func (sg *StrategyGatekeeper) AddToBlacklist(strategyID string, reason string) error {
	return sg.addToBlacklist(context.Background(), strategyID, reason)
}

// RemoveFromBlacklist 从黑名单移除策略 (exported)
func (sg *StrategyGatekeeper) RemoveFromBlacklist(strategyID string) error {
	sg.mu.Lock()
	defer sg.mu.Unlock()

	if sg.blacklist == nil {
		return nil
	}

	if _, exists := sg.blacklist[strategyID]; exists {
		delete(sg.blacklist, strategyID)
		log.Printf("策略 %s 已从黑名单移除", strategyID)

		// 如果有数据库连接，从数据库删除
		if sg.db != nil {
			if err := sg.removeBlacklistEntryFromDB(context.Background(), strategyID); err != nil {
				log.Printf("从数据库删除黑名单条目失败: %v", err)
				// 不返回错误，允许内存操作继续
			}
		}
	}

	return nil
}

// addToBlacklist 添加策略到黑名单
func (sg *StrategyGatekeeper) addToBlacklist(ctx context.Context, strategyID string, reason string) error {
	sg.mu.Lock()
	defer sg.mu.Unlock()

	entry := &BlacklistEntry{
		StrategyID: strategyID,
		Reason:     reason,
		BlockedAt:  time.Now(),
		BlockedBy:  "risk_control",
		Permanent:  true, // 风控停止的策略永久禁用
	}

	if sg.blacklist == nil {
		sg.blacklist = make(map[string]*BlacklistEntry)
	}
	sg.blacklist[strategyID] = entry

	// 如果有数据库连接，保存到数据库
	if sg.db != nil {
		if err := sg.saveBlacklistEntry(ctx, entry); err != nil {
			log.Printf("保存黑名单条目到数据库失败: %v", err)
			// 不返回错误，允许内存操作继续
		}
	}

	log.Printf("策略 %s 已添加到黑名单，原因: %s", strategyID, reason)
	return nil
}

// forceStopStrategy 强制停止策略
func (sg *StrategyGatekeeper) forceStopStrategy(ctx context.Context, strategyID string, reason string) error {
	// 这里应该调用策略管理器来强制停止策略
	// 暂时只记录日志，实际实现需要依赖策略管理器
	log.Printf("强制停止策略 %s，原因: %s", strategyID, reason)

	// 实现实际的策略停止逻辑

	// 1. 更新策略状态为停止
	if err := sg.updateStrategyStatus(ctx, strategyID, "force_stopped", reason); err != nil {
		return fmt.Errorf("failed to update strategy status: %w", err)
	}

	// 2. 取消策略的所有活跃订单
	if err := sg.cancelStrategyOrders(ctx, strategyID); err != nil {
		log.Printf("Warning: Failed to cancel orders for strategy %s: %v", strategyID, err)
	}

	// 3. 清理策略资源
	if err := sg.cleanupStrategyResources(ctx, strategyID); err != nil {
		log.Printf("Warning: Failed to cleanup resources for strategy %s: %v", strategyID, err)
	}

	// 4. 发送停止通知
	if err := sg.sendStrategyStopNotification(ctx, strategyID, reason); err != nil {
		log.Printf("Warning: Failed to send stop notification for strategy %s: %v", strategyID, err)
	}

	// 5. 记录停止事件
	if err := sg.recordStrategyStopEvent(ctx, strategyID, reason); err != nil {
		log.Printf("Warning: Failed to record stop event for strategy %s: %v", strategyID, err)
	}

	log.Printf("Strategy %s has been force stopped successfully", strategyID)
	return nil
}

// IsBlacklisted 检查策略是否在黑名单中 (exported)
func (sg *StrategyGatekeeper) IsBlacklisted(strategyID string) bool {
	return sg.isBlacklisted(strategyID)
}

// isBlacklisted 检查策略是否在黑名单中
func (sg *StrategyGatekeeper) isBlacklisted(strategyID string) bool {
	sg.mu.RLock()
	defer sg.mu.RUnlock()

	entry, exists := sg.blacklist[strategyID]
	if !exists {
		return false
	}

	// 检查是否过期
	if !entry.Permanent && entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
		// 已过期，从黑名单移除
		delete(sg.blacklist, strategyID)
		return false
	}

	return true
}

// saveBlacklistEntry 保存黑名单条目到数据库
func (sg *StrategyGatekeeper) saveBlacklistEntry(ctx context.Context, entry *BlacklistEntry) error {
	if sg.db == nil {
		return fmt.Errorf("database connection not available")
	}

	query := `
		INSERT INTO strategy_blacklist (strategy_id, reason, blocked_at, blocked_by, permanent, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (strategy_id) DO UPDATE SET
			reason = EXCLUDED.reason,
			blocked_at = EXCLUDED.blocked_at,
			blocked_by = EXCLUDED.blocked_by,
			permanent = EXCLUDED.permanent,
			expires_at = EXCLUDED.expires_at
	`

	_, err := sg.db.ExecContext(ctx, query,
		entry.StrategyID, entry.Reason, entry.BlockedAt,
		entry.BlockedBy, entry.Permanent, entry.ExpiresAt)

	return err
}

// loadBlacklistFromDB 从数据库加载黑名单
func (sg *StrategyGatekeeper) loadBlacklistFromDB(ctx context.Context) error {
	if sg.db == nil {
		return fmt.Errorf("database connection not available")
	}

	query := `
		SELECT strategy_id, reason, blocked_at, blocked_by, permanent, expires_at
		FROM strategy_blacklist
		WHERE permanent = true OR (expires_at IS NOT NULL AND expires_at > NOW())
	`

	rows, err := sg.db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	sg.mu.Lock()
	defer sg.mu.Unlock()

	if sg.blacklist == nil {
		sg.blacklist = make(map[string]*BlacklistEntry)
	}

	for rows.Next() {
		entry := &BlacklistEntry{}
		var expiresAt sql.NullTime

		err := rows.Scan(
			&entry.StrategyID, &entry.Reason, &entry.BlockedAt,
			&entry.BlockedBy, &entry.Permanent, &expiresAt,
		)
		if err != nil {
			log.Printf("扫描黑名单条目失败: %v", err)
			continue
		}

		if expiresAt.Valid {
			entry.ExpiresAt = &expiresAt.Time
		}

		sg.blacklist[entry.StrategyID] = entry
	}

	log.Printf("从数据库加载了 %d 个黑名单条目", len(sg.blacklist))
	return rows.Err()
}

// removeBlacklistEntryFromDB 从数据库删除黑名单条目
func (sg *StrategyGatekeeper) removeBlacklistEntryFromDB(ctx context.Context, strategyID string) error {
	if sg.db == nil {
		return fmt.Errorf("database connection not available")
	}

	query := `DELETE FROM strategy_blacklist WHERE strategy_id = $1`
	_, err := sg.db.ExecContext(ctx, query, strategyID)
	return err
}

// updateStrategyStatus 更新策略状态
func (sg *StrategyGatekeeper) updateStrategyStatus(ctx context.Context, strategyID, status, reason string) error {
	if sg.db == nil {
		return fmt.Errorf("database not available")
	}

	query := `
		UPDATE strategies 
		SET status = ?, stop_reason = ?, updated_at = ?, force_stopped_at = ?
		WHERE id = ?
	`

	_, err := sg.db.ExecContext(ctx, query, status, reason, time.Now(), time.Now(), strategyID)
	if err != nil {
		return fmt.Errorf("failed to update strategy status: %w", err)
	}

	return nil
}

// cancelStrategyOrders 取消策略的所有活跃订单
func (sg *StrategyGatekeeper) cancelStrategyOrders(ctx context.Context, strategyID string) error {
	if sg.db == nil {
		return fmt.Errorf("database not available")
	}

	// 查询策略的活跃订单
	query := `
		SELECT id, exchange_order_id, symbol, exchange 
		FROM orders 
		WHERE strategy_id = ? AND status IN ('pending', 'partial_filled')
	`

	rows, err := sg.db.QueryContext(ctx, query, strategyID)
	if err != nil {
		return fmt.Errorf("failed to query active orders: %w", err)
	}
	defer rows.Close()

	var cancelledCount int
	for rows.Next() {
		var orderID, exchangeOrderID, symbol, exchange string
		if err := rows.Scan(&orderID, &exchangeOrderID, &symbol, &exchange); err != nil {
			log.Printf("Failed to scan order: %v", err)
			continue
		}

		// 取消交易所订单
		if err := sg.cancelExchangeOrder(ctx, exchangeOrderID, symbol, exchange); err != nil {
			log.Printf("Failed to cancel exchange order %s: %v", exchangeOrderID, err)
			continue
		}

		// 更新本地订单状态
		if err := sg.updateOrderStatus(ctx, orderID, "cancelled", "force_stopped"); err != nil {
			log.Printf("Failed to update order status %s: %v", orderID, err)
		}

		cancelledCount++
	}

	log.Printf("Cancelled %d orders for strategy %s", cancelledCount, strategyID)
	return nil
}

// cancelExchangeOrder 取消交易所订单
func (sg *StrategyGatekeeper) cancelExchangeOrder(ctx context.Context, exchangeOrderID, symbol, exchange string) error {
	// 这里应该调用相应交易所的API来取消订单
	// 由于没有具体的交易所客户端，我们模拟取消过程
	log.Printf("Cancelling exchange order %s on %s for %s", exchangeOrderID, exchange, symbol)

	// 模拟API调用延迟
	time.Sleep(100 * time.Millisecond)

	// 在实际实现中，这里应该：
	// 1. 根据exchange类型选择相应的客户端
	// 2. 调用取消订单API
	// 3. 处理API响应和错误

	return nil
}

// updateOrderStatus 更新订单状态
func (sg *StrategyGatekeeper) updateOrderStatus(ctx context.Context, orderID, status, reason string) error {
	if sg.db == nil {
		return fmt.Errorf("database not available")
	}

	query := `
		UPDATE orders 
		SET status = ?, cancel_reason = ?, updated_at = ?
		WHERE id = ?
	`

	_, err := sg.db.ExecContext(ctx, query, status, reason, time.Now(), orderID)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	return nil
}

// cleanupStrategyResources 清理策略资源
func (sg *StrategyGatekeeper) cleanupStrategyResources(ctx context.Context, strategyID string) error {
	// 1. 清理内存中的策略实例
	if err := sg.removeStrategyFromMemory(strategyID); err != nil {
		log.Printf("Failed to remove strategy from memory: %v", err)
	}

	// 2. 清理缓存数据
	if err := sg.clearStrategyCache(ctx, strategyID); err != nil {
		log.Printf("Failed to clear strategy cache: %v", err)
	}

	// 3. 停止相关的定时任务
	if err := sg.stopStrategyScheduledTasks(ctx, strategyID); err != nil {
		log.Printf("Failed to stop scheduled tasks: %v", err)
	}

	return nil
}

// removeStrategyFromMemory 从内存中移除策略实例
func (sg *StrategyGatekeeper) removeStrategyFromMemory(strategyID string) error {
	// 这里应该调用策略管理器来移除内存中的策略实例
	log.Printf("Removing strategy %s from memory", strategyID)
	return nil
}

// clearStrategyCache 清理策略缓存
func (sg *StrategyGatekeeper) clearStrategyCache(ctx context.Context, strategyID string) error {
	// 这里应该清理Redis或其他缓存中的策略相关数据
	log.Printf("Clearing cache for strategy %s", strategyID)
	return nil
}

// stopStrategyScheduledTasks 停止策略的定时任务
func (sg *StrategyGatekeeper) stopStrategyScheduledTasks(ctx context.Context, strategyID string) error {
	// 这里应该停止与策略相关的所有定时任务
	log.Printf("Stopping scheduled tasks for strategy %s", strategyID)
	return nil
}

// sendStrategyStopNotification 发送策略停止通知
func (sg *StrategyGatekeeper) sendStrategyStopNotification(ctx context.Context, strategyID, reason string) error {
	notification := map[string]interface{}{
		"type":        "strategy_force_stopped",
		"strategy_id": strategyID,
		"reason":      reason,
		"timestamp":   time.Now(),
		"severity":    "high",
	}

	// 这里应该发送到通知系统
	log.Printf("Strategy stop notification: %+v", notification)
	return nil
}

// recordStrategyStopEvent 记录策略停止事件
func (sg *StrategyGatekeeper) recordStrategyStopEvent(ctx context.Context, strategyID, reason string) error {
	if sg.db == nil {
		return fmt.Errorf("database not available")
	}

	query := `
		INSERT INTO strategy_events 
		(strategy_id, event_type, event_data, created_at)
		VALUES (?, ?, ?, ?)
	`

	eventData := map[string]interface{}{
		"reason":    reason,
		"timestamp": time.Now(),
		"action":    "force_stop",
	}

	eventDataJSON, _ := json.Marshal(eventData)

	_, err := sg.db.ExecContext(ctx, query, strategyID, "force_stopped", string(eventDataJSON), time.Now())
	if err != nil {
		return fmt.Errorf("failed to record strategy stop event: %w", err)
	}

	return nil
}
