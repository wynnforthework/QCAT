package risk

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/strategy/validation"
)

// ProcessManager 进程管理器接口
type ProcessManager interface {
	StopProcess(ctx context.Context, processID string) error
	GetProcessesByStrategy(strategyID string) ([]string, error)
}

// ExchangeAPI 交易所API接口
type ExchangeAPI interface {
	CancelAllOrders(ctx context.Context, strategyID string) error
	GetOpenOrders(ctx context.Context, strategyID string) ([]Order, error)
}

// CacheManager 缓存管理器接口
type CacheManager interface {
	DeleteByPattern(ctx context.Context, pattern string) error
	Clear(ctx context.Context) error
}

// Order 订单结构
type Order struct {
	ID         string  `json:"id"`
	StrategyID string  `json:"strategy_id"`
	Symbol     string  `json:"symbol"`
	Side       string  `json:"side"`
	Quantity   float64 `json:"quantity"`
	Price      float64 `json:"price"`
	Status     string  `json:"status"`
}

// RealtimeRiskMonitor 实时风险监控器
type RealtimeRiskMonitor struct {
	db               *sql.DB
	gatekeeper       *validation.StrategyGatekeeper
	processManager   ProcessManager
	exchangeAPI      ExchangeAPI
	cacheManager     CacheManager
	monitorInterval  time.Duration
	emergencyActions map[string]EmergencyAction
	riskThresholds   *RiskThresholds
	activeStrategies map[string]*StrategyRiskState
	mu               sync.RWMutex
	stopChan         chan struct{}
	running          bool
}

// RiskThresholds 风险阈值配置
type RiskThresholds struct {
	MaxDailyLoss       float64 `json:"max_daily_loss"`       // 最大日损失 (如 0.05 = 5%)
	MaxTotalPositions  int     `json:"max_total_positions"`  // 最大总持仓数
	MaxPositionValue   float64 `json:"max_position_value"`   // 单个持仓最大价值
	MaxDrawdown        float64 `json:"max_drawdown"`         // 最大回撤
	MaxConsecutiveLoss int     `json:"max_consecutive_loss"` // 最大连续亏损次数
	MinAccountBalance  float64 `json:"min_account_balance"`  // 最小账户余额
}

// StrategyRiskState 策略风险状态
type StrategyRiskState struct {
	StrategyID        string    `json:"strategy_id"`
	CurrentPositions  int       `json:"current_positions"`
	DailyPnL          float64   `json:"daily_pnl"`
	TotalPnL          float64   `json:"total_pnl"`
	ConsecutiveLosses int       `json:"consecutive_losses"`
	LastTradeTime     time.Time `json:"last_trade_time"`
	RiskLevel         string    `json:"risk_level"`
	IsBlocked         bool      `json:"is_blocked"`
	BlockReason       string    `json:"block_reason"`
}

// EmergencyAction 紧急行动类型
type EmergencyAction int

const (
	ActionWarning EmergencyAction = iota
	ActionReducePosition
	ActionStopStrategy
	ActionEmergencyStop
)

// NewRealtimeRiskMonitor 创建实时风险监控器
func NewRealtimeRiskMonitor(db *sql.DB, processManager ProcessManager, exchangeAPI ExchangeAPI, cacheManager CacheManager) *RealtimeRiskMonitor {
	return &RealtimeRiskMonitor{
		db:               db,
		gatekeeper:       validation.NewStrategyGatekeeper(),
		processManager:   processManager,
		exchangeAPI:      exchangeAPI,
		cacheManager:     cacheManager,
		monitorInterval:  30 * time.Second, // 每30秒检查一次
		emergencyActions: make(map[string]EmergencyAction),
		riskThresholds: &RiskThresholds{
			MaxDailyLoss:       0.05,  // 5%
			MaxTotalPositions:  1000,  // 最多1000个持仓
			MaxPositionValue:   10000, // 单个持仓最大$10k
			MaxDrawdown:        0.15,  // 15%
			MaxConsecutiveLoss: 5,     // 连续5次亏损
			MinAccountBalance:  1000,  // 最小余额$1k
		},
		activeStrategies: make(map[string]*StrategyRiskState),
		stopChan:         make(chan struct{}),
	}
}

// Start 启动实时监控
func (rm *RealtimeRiskMonitor) Start(ctx context.Context) error {
	rm.mu.Lock()
	if rm.running {
		rm.mu.Unlock()
		return fmt.Errorf("risk monitor is already running")
	}
	rm.running = true
	rm.mu.Unlock()

	log.Printf("🚨 实时风险监控器启动，监控间隔: %v", rm.monitorInterval)

	go rm.monitorLoop(ctx)
	return nil
}

// Stop 停止监控
func (rm *RealtimeRiskMonitor) Stop() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if !rm.running {
		return
	}

	rm.running = false
	close(rm.stopChan)
	log.Printf("实时风险监控器已停止")
}

// monitorLoop 监控循环
func (rm *RealtimeRiskMonitor) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(rm.monitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-rm.stopChan:
			return
		case <-ticker.C:
			if err := rm.performRiskCheck(ctx); err != nil {
				log.Printf("风险检查失败: %v", err)
			}
		}
	}
}

// performRiskCheck 执行风险检查
func (rm *RealtimeRiskMonitor) performRiskCheck(ctx context.Context) error {
	// 1. 获取所有活跃策略的当前状态
	strategies, err := rm.getActiveStrategies(ctx)
	if err != nil {
		return fmt.Errorf("获取活跃策略失败: %w", err)
	}

	// 2. 检查每个策略的风险状态
	for _, strategy := range strategies {
		if err := rm.checkStrategyRisk(ctx, strategy); err != nil {
			log.Printf("策略 %s 风险检查失败: %v", strategy.StrategyID, err)
		}
	}

	// 3. 检查系统整体风险
	if err := rm.checkSystemRisk(ctx); err != nil {
		log.Printf("系统风险检查失败: %v", err)
	}

	return nil
}

// getActiveStrategies 获取活跃策略
func (rm *RealtimeRiskMonitor) getActiveStrategies(ctx context.Context) ([]*StrategyRiskState, error) {
	query := `
		SELECT 
			s.id,
			COUNT(p.id) as position_count,
			COALESCE(SUM(p.unrealized_pnl + p.realized_pnl), 0) as total_pnl,
			COALESCE(MAX(p.updated_at), s.updated_at) as last_activity
		FROM strategies s
		LEFT JOIN positions p ON s.id = p.strategy_id AND p.status = 'open'
		WHERE s.is_running = true AND s.status = 'active'
		GROUP BY s.id, s.updated_at
	`

	rows, err := rm.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var strategies []*StrategyRiskState
	for rows.Next() {
		var strategy StrategyRiskState
		var lastActivity time.Time

		err := rows.Scan(
			&strategy.StrategyID,
			&strategy.CurrentPositions,
			&strategy.TotalPnL,
			&lastActivity,
		)
		if err != nil {
			continue
		}

		strategy.LastTradeTime = lastActivity

		// 计算今日盈亏
		strategy.DailyPnL = rm.calculateDailyPnL(ctx, strategy.StrategyID)

		// 评估风险等级
		strategy.RiskLevel = rm.assessRiskLevel(&strategy)

		strategies = append(strategies, &strategy)
	}

	return strategies, nil
}

// checkStrategyRisk 检查单个策略风险
func (rm *RealtimeRiskMonitor) checkStrategyRisk(ctx context.Context, strategy *StrategyRiskState) error {
	var actions []EmergencyAction

	// 检查持仓数量
	if strategy.CurrentPositions > rm.riskThresholds.MaxTotalPositions {
		actions = append(actions, ActionStopStrategy)
		strategy.BlockReason = fmt.Sprintf("持仓数量过多: %d > %d",
			strategy.CurrentPositions, rm.riskThresholds.MaxTotalPositions)
	}

	// 检查日损失
	if strategy.DailyPnL < -rm.riskThresholds.MaxDailyLoss {
		actions = append(actions, ActionStopStrategy)
		strategy.BlockReason = fmt.Sprintf("日损失超限: %.2f%% > %.2f%%",
			strategy.DailyPnL*100, rm.riskThresholds.MaxDailyLoss*100)
	}

	// 检查总盈亏
	if strategy.TotalPnL < -50000 { // 总亏损超过5万
		actions = append(actions, ActionEmergencyStop)
		strategy.BlockReason = "总亏损过大，紧急停止"
	}

	// 执行紧急行动
	for _, action := range actions {
		if err := rm.executeEmergencyAction(ctx, strategy.StrategyID, action, strategy.BlockReason); err != nil {
			log.Printf("执行紧急行动失败: %v", err)
		}
	}

	// 更新策略状态
	rm.mu.Lock()
	rm.activeStrategies[strategy.StrategyID] = strategy
	rm.mu.Unlock()

	return nil
}

// checkSystemRisk 检查系统整体风险
func (rm *RealtimeRiskMonitor) checkSystemRisk(ctx context.Context) error {
	// 查询系统总体状态
	query := `
		SELECT 
			COUNT(*) as total_positions,
			COALESCE(SUM(unrealized_pnl + realized_pnl), 0) as total_pnl,
			COUNT(DISTINCT strategy_id) as active_strategies
		FROM positions 
		WHERE status = 'open'
	`

	var totalPositions int
	var totalPnL float64
	var activeStrategies int

	err := rm.db.QueryRowContext(ctx, query).Scan(&totalPositions, &totalPnL, &activeStrategies)
	if err != nil {
		return err
	}

	// 系统级风险检查
	if totalPositions > 50000 { // 超过5万个持仓
		log.Printf("🚨 系统风险警告: 总持仓数 %d 超过安全阈值", totalPositions)
		// 可以在这里实施系统级紧急措施
	}

	if totalPnL < -100000 { // 总亏损超过10万
		log.Printf("🚨 系统风险严重: 总亏损 $%.2f 超过安全阈值", totalPnL)
		// 可以在这里停用所有策略
	}

	return nil
}

// calculateDailyPnL 计算日盈亏
func (rm *RealtimeRiskMonitor) calculateDailyPnL(ctx context.Context, strategyID string) float64 {
	today := time.Now().Truncate(24 * time.Hour)

	query := `
		SELECT COALESCE(SUM(unrealized_pnl + realized_pnl), 0)
		FROM positions 
		WHERE strategy_id = $1 AND updated_at >= $2
	`

	var dailyPnL float64
	rm.db.QueryRowContext(ctx, query, strategyID, today).Scan(&dailyPnL)

	return dailyPnL
}

// assessRiskLevel 评估风险等级
func (rm *RealtimeRiskMonitor) assessRiskLevel(strategy *StrategyRiskState) string {
	if strategy.CurrentPositions > 10000 || strategy.DailyPnL < -0.1 {
		return "CRITICAL"
	} else if strategy.CurrentPositions > 5000 || strategy.DailyPnL < -0.05 {
		return "HIGH"
	} else if strategy.CurrentPositions > 1000 || strategy.DailyPnL < -0.02 {
		return "MEDIUM"
	}
	return "LOW"
}

// executeEmergencyAction 执行紧急行动
func (rm *RealtimeRiskMonitor) executeEmergencyAction(ctx context.Context, strategyID string, action EmergencyAction, reason string) error {
	switch action {
	case ActionWarning:
		log.Printf("⚠️  策略 %s 风险警告: %s", strategyID, reason)
	case ActionStopStrategy:
		log.Printf("🛑 紧急停止策略 %s: %s", strategyID, reason)
		return rm.stopStrategy(ctx, strategyID, reason)
	case ActionEmergencyStop:
		log.Printf("🚨 紧急停止策略 %s: %s", strategyID, reason)
		return rm.emergencyStopStrategy(ctx, strategyID, reason)
	}
	return nil
}

// stopStrategy 停止策略
func (rm *RealtimeRiskMonitor) stopStrategy(ctx context.Context, strategyID string, reason string) error {
	query := `
		UPDATE strategies 
		SET is_running = false, status = 'stopped', 
		    stop_reason = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := rm.db.ExecContext(ctx, query, reason, time.Now(), strategyID)
	return err
}

// emergencyStopStrategy 紧急停止策略
func (rm *RealtimeRiskMonitor) emergencyStopStrategy(ctx context.Context, strategyID string, reason string) error {
	log.Printf("🚨 紧急停止策略 %s，原因: %s", strategyID, reason)

	// 1. 停止策略
	if err := rm.stopStrategy(ctx, strategyID, reason); err != nil {
		log.Printf("停止策略失败: %v", err)
		return err
	}

	// 2. 通过守门员禁用策略并加入黑名单
	if err := rm.gatekeeper.DisableStrategy(ctx, strategyID, reason); err != nil {
		log.Printf("禁用策略失败: %v", err)
		return err
	}

	// 3. 强制停止所有相关进程
	if err := rm.forceStopStrategyProcesses(ctx, strategyID); err != nil {
		log.Printf("强制停止策略进程失败: %v", err)
		// 不返回错误，继续执行其他清理操作
	}

	// 4. 清理策略相关资源
	if err := rm.cleanupStrategyResources(ctx, strategyID); err != nil {
		log.Printf("清理策略资源失败: %v", err)
		// 不返回错误，继续执行
	}

	log.Printf("✅ 策略 %s 已被紧急停止并禁用", strategyID)
	return nil
}

// GetRiskStatus 获取风险状态
func (rm *RealtimeRiskMonitor) GetRiskStatus() map[string]*StrategyRiskState {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	result := make(map[string]*StrategyRiskState)
	for k, v := range rm.activeStrategies {
		result[k] = v
	}
	return result
}

// forceStopStrategyProcesses 强制停止策略相关进程
func (rm *RealtimeRiskMonitor) forceStopStrategyProcesses(ctx context.Context, strategyID string) error {
	log.Printf("强制停止策略 %s 的相关进程", strategyID)

	// 实现进程停止逻辑
	if rm.processManager == nil {
		log.Printf("警告: 进程管理器未初始化，无法停止策略 %s 的进程", strategyID)
		return fmt.Errorf("process manager not initialized")
	}

	// 获取策略相关的所有进程
	processIDs, err := rm.processManager.GetProcessesByStrategy(strategyID)
	if err != nil {
		log.Printf("获取策略 %s 的进程列表失败: %v", strategyID, err)
		return fmt.Errorf("failed to get processes for strategy %s: %w", strategyID, err)
	}

	if len(processIDs) == 0 {
		log.Printf("策略 %s 没有运行中的进程", strategyID)
		return nil
	}

	// 逐个停止进程
	var stopErrors []error
	for _, processID := range processIDs {
		log.Printf("停止策略 %s 的进程: %s", strategyID, processID)

		if err := rm.processManager.StopProcess(ctx, processID); err != nil {
			log.Printf("停止进程 %s 失败: %v", processID, err)
			stopErrors = append(stopErrors, fmt.Errorf("failed to stop process %s: %w", processID, err))
		} else {
			log.Printf("成功停止进程: %s", processID)
		}
	}

	// 如果有停止失败的进程，返回错误
	if len(stopErrors) > 0 {
		return fmt.Errorf("failed to stop %d processes: %v", len(stopErrors), stopErrors)
	}

	log.Printf("策略 %s 的所有进程已成功停止 (共 %d 个)", strategyID, len(processIDs))
	return nil
}

// cleanupStrategyResources 清理策略相关资源
func (rm *RealtimeRiskMonitor) cleanupStrategyResources(ctx context.Context, strategyID string) error {
	log.Printf("清理策略 %s 的相关资源", strategyID)

	// 1. 清理内存中的策略状态
	rm.mu.Lock()
	delete(rm.activeStrategies, strategyID)
	rm.mu.Unlock()

	// 2. 取消未完成的订单
	if err := rm.cancelPendingOrders(ctx, strategyID); err != nil {
		log.Printf("取消策略 %s 的待处理订单失败: %v", strategyID, err)
	}

	// 3. 清理缓存数据
	if err := rm.clearStrategyCache(ctx, strategyID); err != nil {
		log.Printf("清理策略 %s 的缓存失败: %v", strategyID, err)
	}

	log.Printf("策略 %s 的资源清理完成", strategyID)
	return nil
}

// cancelPendingOrders 取消待处理订单
func (rm *RealtimeRiskMonitor) cancelPendingOrders(ctx context.Context, strategyID string) error {
	// 实现订单取消逻辑
	log.Printf("取消策略 %s 的待处理订单", strategyID)

	if rm.exchangeAPI == nil {
		log.Printf("警告: 交易所API未初始化，无法取消策略 %s 的订单", strategyID)
		return fmt.Errorf("exchange API not initialized")
	}

	// 首先获取策略的所有开放订单
	openOrders, err := rm.exchangeAPI.GetOpenOrders(ctx, strategyID)
	if err != nil {
		log.Printf("获取策略 %s 的开放订单失败: %v", strategyID, err)
		return fmt.Errorf("failed to get open orders for strategy %s: %w", strategyID, err)
	}

	if len(openOrders) == 0 {
		log.Printf("策略 %s 没有待处理的订单", strategyID)
		return nil
	}

	log.Printf("策略 %s 有 %d 个待处理订单，开始取消", strategyID, len(openOrders))

	// 取消所有开放订单
	if err := rm.exchangeAPI.CancelAllOrders(ctx, strategyID); err != nil {
		log.Printf("批量取消策略 %s 的订单失败: %v", strategyID, err)
		return fmt.Errorf("failed to cancel orders for strategy %s: %w", strategyID, err)
	}

	// 记录取消的订单详情
	for _, order := range openOrders {
		log.Printf("已取消订单: ID=%s, Symbol=%s, Side=%s, Quantity=%.4f, Price=%.4f",
			order.ID, order.Symbol, order.Side, order.Quantity, order.Price)
	}

	log.Printf("策略 %s 的所有待处理订单已成功取消 (共 %d 个)", strategyID, len(openOrders))
	return nil
}

// clearStrategyCache 清理策略缓存
func (rm *RealtimeRiskMonitor) clearStrategyCache(ctx context.Context, strategyID string) error {
	// 实现缓存清理逻辑
	log.Printf("清理策略 %s 的缓存数据", strategyID)

	if rm.cacheManager == nil {
		log.Printf("警告: 缓存管理器未初始化，无法清理策略 %s 的缓存", strategyID)
		return fmt.Errorf("cache manager not initialized")
	}

	// 定义需要清理的缓存键模式
	cachePatterns := []string{
		fmt.Sprintf("strategy:%s:*", strategyID),    // 策略相关的所有缓存
		fmt.Sprintf("positions:%s:*", strategyID),   // 持仓缓存
		fmt.Sprintf("orders:%s:*", strategyID),      // 订单缓存
		fmt.Sprintf("signals:%s:*", strategyID),     // 信号缓存
		fmt.Sprintf("metrics:%s:*", strategyID),     // 指标缓存
		fmt.Sprintf("market_data:%s:*", strategyID), // 市场数据缓存
		fmt.Sprintf("risk_state:%s", strategyID),    // 风险状态缓存
		fmt.Sprintf("performance:%s:*", strategyID), // 性能数据缓存
	}

	var clearErrors []error
	totalCleared := 0

	// 逐个清理缓存模式
	for _, pattern := range cachePatterns {
		log.Printf("清理缓存模式: %s", pattern)

		if err := rm.cacheManager.DeleteByPattern(ctx, pattern); err != nil {
			log.Printf("清理缓存模式 %s 失败: %v", pattern, err)
			clearErrors = append(clearErrors, fmt.Errorf("failed to clear pattern %s: %w", pattern, err))
		} else {
			log.Printf("成功清理缓存模式: %s", pattern)
			totalCleared++
		}
	}

	// 如果有清理失败的模式，记录错误但不返回失败
	if len(clearErrors) > 0 {
		log.Printf("策略 %s 的缓存清理部分失败: %d/%d 个模式清理失败",
			strategyID, len(clearErrors), len(cachePatterns))
		// 不返回错误，因为部分清理失败不应该阻止整个紧急停止流程
	}

	log.Printf("策略 %s 的缓存清理完成: 成功清理 %d/%d 个缓存模式",
		strategyID, totalCleared, len(cachePatterns))
	return nil
}
