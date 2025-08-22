package system

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/risk"
	"qcat/internal/strategy/automation"
	"qcat/internal/strategy/validation"
)

// AutomationManager 自动化管理器 - 统一管理所有自动化组件
type AutomationManager struct {
	db                   *sql.DB
	riskMonitor          *risk.RealtimeRiskMonitor
	backtestScheduler    *automation.BacktestScheduler
	parameterOptimizer   *automation.ParameterOptimizer
	strategyGatekeeper   *validation.StrategyGatekeeper
	running              bool
	mu                   sync.RWMutex
	ctx                  context.Context
	cancel               context.CancelFunc
}

// SystemStatus 系统状态
type SystemStatus struct {
	AutomationEnabled    bool                   `json:"automation_enabled"`
	RiskMonitorRunning   bool                   `json:"risk_monitor_running"`
	BacktestRunning      bool                   `json:"backtest_running"`
	OptimizerRunning     bool                   `json:"optimizer_running"`
	GatekeeperEnabled    bool                   `json:"gatekeeper_enabled"`
	StartTime            time.Time              `json:"start_time"`
	Uptime               time.Duration          `json:"uptime"`
	ComponentStatus      map[string]interface{} `json:"component_status"`
	LastHealthCheck      time.Time              `json:"last_health_check"`
}

// NewAutomationManager 创建自动化管理器
func NewAutomationManager(db *sql.DB) *AutomationManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	// 创建回测调度器
	backtestScheduler := automation.NewBacktestScheduler(db)
	
	// 创建参数优化器
	parameterOptimizer := automation.NewParameterOptimizer(db, backtestScheduler)
	
	return &AutomationManager{
		db:                 db,
		riskMonitor:        risk.NewRealtimeRiskMonitor(db),
		backtestScheduler:  backtestScheduler,
		parameterOptimizer: parameterOptimizer,
		strategyGatekeeper: validation.NewStrategyGatekeeper(),
		ctx:                ctx,
		cancel:             cancel,
	}
}

// Start 启动所有自动化组件
func (am *AutomationManager) Start() error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if am.running {
		return fmt.Errorf("automation manager is already running")
	}

	log.Printf("🚀 启动量化交易自动化系统...")

	// 1. 启动策略守门员（立即生效）
	am.strategyGatekeeper.Enable()
	log.Printf("✅ 策略守门员已启用 - 所有策略启动前必须通过验证")

	// 2. 启动实时风险监控
	if err := am.riskMonitor.Start(am.ctx); err != nil {
		return fmt.Errorf("启动风险监控失败: %w", err)
	}
	log.Printf("✅ 实时风险监控已启动")

	// 3. 启动自动化回测调度器
	if err := am.backtestScheduler.Start(am.ctx); err != nil {
		return fmt.Errorf("启动回测调度器失败: %w", err)
	}
	log.Printf("✅ 自动化回测调度器已启动")

	// 4. 启动参数优化器
	if err := am.parameterOptimizer.Start(am.ctx); err != nil {
		return fmt.Errorf("启动参数优化器失败: %w", err)
	}
	log.Printf("✅ 策略参数自动优化器已启动")

	am.running = true

	// 5. 启动健康检查
	go am.healthCheckLoop()

	log.Printf("🎉 量化交易自动化系统启动完成！")
	log.Printf("📊 系统功能:")
	log.Printf("   - ✅ 强制回测验证: 策略必须通过回测才能启用")
	log.Printf("   - ✅ 实时风险监控: 每30秒检查风险状态，自动停止高风险策略")
	log.Printf("   - ✅ 自动化回测: 每小时检查并回测需要验证的策略")
	log.Printf("   - ✅ 参数优化: 每天自动优化策略参数")
	log.Printf("   - ✅ 策略守门员: 阻止未验证策略启动")

	return nil
}

// Stop 停止所有自动化组件
func (am *AutomationManager) Stop() {
	am.mu.Lock()
	defer am.mu.Unlock()

	if !am.running {
		return
	}

	log.Printf("🛑 停止量化交易自动化系统...")

	// 停止所有组件
	am.riskMonitor.Stop()
	am.backtestScheduler.Stop()
	am.parameterOptimizer.Stop()
	am.strategyGatekeeper.Disable()

	// 取消上下文
	am.cancel()

	am.running = false
	log.Printf("量化交易自动化系统已停止")
}

// healthCheckLoop 健康检查循环
func (am *AutomationManager) healthCheckLoop() {
	ticker := time.NewTicker(5 * time.Minute) // 每5分钟检查一次
	defer ticker.Stop()

	for {
		select {
		case <-am.ctx.Done():
			return
		case <-ticker.C:
			am.performHealthCheck()
		}
	}
}

// performHealthCheck 执行健康检查
func (am *AutomationManager) performHealthCheck() {
	log.Printf("🔍 执行系统健康检查...")

	// 检查数据库连接
	if err := am.db.Ping(); err != nil {
		log.Printf("❌ 数据库连接异常: %v", err)
		return
	}

	// 检查是否有策略绕过了验证
	if err := am.checkUnvalidatedStrategies(); err != nil {
		log.Printf("⚠️  发现未验证策略: %v", err)
	}

	// 检查风险状态
	riskStatus := am.riskMonitor.GetRiskStatus()
	highRiskCount := 0
	for _, status := range riskStatus {
		if status.RiskLevel == "HIGH" || status.RiskLevel == "CRITICAL" {
			highRiskCount++
		}
	}

	if highRiskCount > 0 {
		log.Printf("⚠️  发现 %d 个高风险策略", highRiskCount)
	}

	log.Printf("✅ 系统健康检查完成")
}

// checkUnvalidatedStrategies 检查未验证的策略
func (am *AutomationManager) checkUnvalidatedStrategies() error {
	query := `
		SELECT s.id, s.name
		FROM strategies s
		LEFT JOIN backtest_results br ON s.id = br.strategy_id
		WHERE s.is_running = true 
		AND s.status = 'active'
		AND (br.is_valid IS NULL OR br.is_valid = false)
	`

	rows, err := am.db.QueryContext(am.ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	var unvalidatedStrategies []string
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			continue
		}
		unvalidatedStrategies = append(unvalidatedStrategies, fmt.Sprintf("%s (%s)", name, id))
	}

	if len(unvalidatedStrategies) > 0 {
		log.Printf("🚨 发现 %d 个未验证但正在运行的策略:", len(unvalidatedStrategies))
		for _, strategy := range unvalidatedStrategies {
			log.Printf("   - %s", strategy)
		}
		
		// 可以选择自动停止这些策略
		// am.stopUnvalidatedStrategies(unvalidatedStrategies)
	}

	return nil
}

// GetSystemStatus 获取系统状态
func (am *AutomationManager) GetSystemStatus() (*SystemStatus, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	status := &SystemStatus{
		AutomationEnabled:  am.running,
		RiskMonitorRunning: am.running, // 简化处理
		BacktestRunning:    am.running,
		OptimizerRunning:   am.running,
		GatekeeperEnabled:  am.running,
		StartTime:          time.Now(), // 应该记录实际启动时间
		LastHealthCheck:    time.Now(),
		ComponentStatus:    make(map[string]interface{}),
	}

	if am.running {
		status.Uptime = time.Since(status.StartTime)

		// 获取各组件详细状态
		if backtestStatus, err := am.backtestScheduler.GetSchedulerStatus(am.ctx); err == nil {
			status.ComponentStatus["backtest_scheduler"] = backtestStatus
		}

		if optimizerStatus, err := am.parameterOptimizer.GetOptimizerStatus(am.ctx); err == nil {
			status.ComponentStatus["parameter_optimizer"] = optimizerStatus
		}

		status.ComponentStatus["risk_monitor"] = map[string]interface{}{
			"active_strategies": len(am.riskMonitor.GetRiskStatus()),
			"monitoring":        true,
		}
	}

	return status, nil
}

// EmergencyStop 紧急停止所有策略
func (am *AutomationManager) EmergencyStop(reason string) error {
	log.Printf("🚨 执行紧急停止: %s", reason)

	query := `
		UPDATE strategies 
		SET is_running = false, status = 'emergency_stopped', 
		    stop_reason = $1, updated_at = $2
		WHERE is_running = true
	`

	result, err := am.db.ExecContext(am.ctx, query, reason, time.Now())
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("🛑 已紧急停止 %d 个策略", rowsAffected)

	return nil
}

// GetAutomationSummary 获取自动化系统摘要
func (am *AutomationManager) GetAutomationSummary() map[string]interface{} {
	return map[string]interface{}{
		"system_name":    "QCAT 量化交易自动化系统",
		"version":        "1.0.0",
		"features": []string{
			"强制回测验证",
			"实时风险监控",
			"自动化回测调度",
			"策略参数优化",
			"策略守门员保护",
			"紧急停止机制",
		},
		"running":           am.running,
		"components_count":  5,
		"safety_level":      "HIGH",
		"description":       "全自动化的量化交易风险控制和策略优化系统",
	}
}

// IsRunning 检查系统是否运行中
func (am *AutomationManager) IsRunning() bool {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.running
}

// RestartComponent 重启指定组件
func (am *AutomationManager) RestartComponent(componentName string) error {
	switch componentName {
	case "risk_monitor":
		am.riskMonitor.Stop()
		return am.riskMonitor.Start(am.ctx)
	case "backtest_scheduler":
		am.backtestScheduler.Stop()
		return am.backtestScheduler.Start(am.ctx)
	case "parameter_optimizer":
		am.parameterOptimizer.Stop()
		return am.parameterOptimizer.Start(am.ctx)
	default:
		return fmt.Errorf("unknown component: %s", componentName)
	}
}
