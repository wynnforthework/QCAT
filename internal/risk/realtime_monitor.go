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

// RealtimeRiskMonitor 实时风险监控器
type RealtimeRiskMonitor struct {
	db                *sql.DB
	gatekeeper        *validation.StrategyGatekeeper
	monitorInterval   time.Duration
	emergencyActions  map[string]EmergencyAction
	riskThresholds    *RiskThresholds
	activeStrategies  map[string]*StrategyRiskState
	mu                sync.RWMutex
	stopChan          chan struct{}
	running           bool
}

// RiskThresholds 风险阈值配置
type RiskThresholds struct {
	MaxDailyLoss        float64 `json:"max_daily_loss"`         // 最大日损失 (如 0.05 = 5%)
	MaxTotalPositions   int     `json:"max_total_positions"`    // 最大总持仓数
	MaxPositionValue    float64 `json:"max_position_value"`     // 单个持仓最大价值
	MaxDrawdown         float64 `json:"max_drawdown"`           // 最大回撤
	MaxConsecutiveLoss  int     `json:"max_consecutive_loss"`   // 最大连续亏损次数
	MinAccountBalance   float64 `json:"min_account_balance"`    // 最小账户余额
}

// StrategyRiskState 策略风险状态
type StrategyRiskState struct {
	StrategyID         string    `json:"strategy_id"`
	CurrentPositions   int       `json:"current_positions"`
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
func NewRealtimeRiskMonitor(db *sql.DB) *RealtimeRiskMonitor {
	return &RealtimeRiskMonitor{
		db:              db,
		gatekeeper:      validation.NewStrategyGatekeeper(),
		monitorInterval: 30 * time.Second, // 每30秒检查一次
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
		stopChan:        make(chan struct{}),
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
	// 1. 停止策略
	if err := rm.stopStrategy(ctx, strategyID, reason); err != nil {
		return err
	}

	// 2. 通过守门员禁用策略
	return rm.gatekeeper.DisableStrategy(ctx, strategyID, reason)
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
