package operations

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (m *Manager) registerDefaultChecks() {
	checks := []Check{
		m.newDoubleLoopCheck(),
		m.newStrategyCheck(),
		m.newTradingCheck(),
		m.newExchangeKeyCheck(),
		m.newSystemCheck(),
	}
	for _, chk := range checks {
		if chk != nil {
			m.registerCheck(chk)
		}
	}
}

func (m *Manager) newDoubleLoopCheck() Check {
	return newFunctionalCheck("double_loop.core", "双闭环运行基线", DomainDoubleLoop, CheckModeAll, time.Minute, func() (Status, string, map[string]interface{}) {
		automation := m.getAutomation()
		if automation == nil {
			return StatusCritical, "automation system not initialized", map[string]interface{}{"component": "automation_system"}
		}

		status := automation.GetStatus()
		if status == nil {
			return StatusCritical, "automation status not available", map[string]interface{}{"component": "automation_status"}
		}

		details := map[string]interface{}{
			"scheduler_status":  status.SchedulerStatus,
			"executor_status":   status.ExecutorStatus,
			"active_tasks":      status.ActiveTasks,
			"completed_tasks":   status.CompletedTasks,
			"failed_tasks":      status.FailedTasks,
			"active_actions":    status.ActiveActions,
			"completed_actions": status.CompletedActions,
			"failed_actions":    status.FailedActions,
			"health_score":      status.HealthScore,
			"last_health_check": status.LastHealthCheck,
		}

		totalTasks := status.CompletedTasks + status.FailedTasks
		var taskFailure float64
		if totalTasks > 0 {
			taskFailure = float64(status.FailedTasks) / float64(totalTasks)
		}
		details["task_failure_rate"] = taskFailure

		totalActions := status.CompletedActions + status.FailedActions
		var actionFailure float64
		if totalActions > 0 {
			actionFailure = float64(status.FailedActions) / float64(totalActions)
		}
		details["action_failure_rate"] = actionFailure

		if !status.IsRunning {
			return StatusCritical, "automation system is not running", details
		}

		if strings.ToLower(status.SchedulerStatus) != "running" || strings.ToLower(status.ExecutorStatus) != "running" {
			return StatusCritical, fmt.Sprintf("scheduler=%s executor=%s", status.SchedulerStatus, status.ExecutorStatus), details
		}

		summary := fmt.Sprintf("tasks ok (%d total) actions ok (%d total)", totalTasks, totalActions)
		result := StatusHealthy

		if taskFailure > 0.20 || actionFailure > 0.20 {
			result = StatusCritical
			summary = fmt.Sprintf("excessive failure rate: tasks %.1f%% actions %.1f%%", taskFailure*100, actionFailure*100)
		} else if taskFailure > 0.05 || actionFailure > 0.05 {
			result = StatusWarning
			summary = fmt.Sprintf("elevated failure rate: tasks %.1f%% actions %.1f%%", taskFailure*100, actionFailure*100)
		}

		return result, summary, details
	})
}

func (m *Manager) newStrategyCheck() Check {
	return newFunctionalCheck("double_loop.strategy", "策略生命周期运行", DomainStrategy, CheckModeAll, 2*time.Minute, func() (Status, string, map[string]interface{}) {
		automation := m.getAutomation()
		if automation == nil {
			return StatusCritical, "automation system not initialized", map[string]interface{}{"component": "automation_system"}
		}

		scheduler := automation.GetScheduler()
		if scheduler == nil {
			return StatusWarning, "automation scheduler not available", map[string]interface{}{"component": "scheduler"}
		}

		stats := scheduler.GetStats()
		if stats == nil {
			return StatusWarning, "scheduler stats unavailable", map[string]interface{}{"component": "scheduler"}
		}

		details := map[string]interface{}{
			"total_tasks":     stats.TotalTasks,
			"running_tasks":   stats.RunningTasks,
			"completed_tasks": stats.CompletedTasks,
			"failed_tasks":    stats.FailedTasks,
			"skipped_tasks":   stats.SkippedTasks,
			"last_update":     stats.LastUpdateTime,
		}

		if stats.TotalTasks == 0 {
			return StatusWarning, "no automation tasks registered", details
		}

		total := stats.CompletedTasks + stats.FailedTasks
		var failRate float64
		if total > 0 {
			failRate = float64(stats.FailedTasks) / float64(total)
		}
		details["failure_rate"] = failRate

		summary := fmt.Sprintf("tasks: %d total, %d running", stats.TotalTasks, stats.RunningTasks)
		status := StatusHealthy

		if failRate > 0.15 {
			status = StatusCritical
			summary = fmt.Sprintf("strategy tasks failing %.1f%%", failRate*100)
		} else if failRate > 0.05 {
			status = StatusWarning
			summary = fmt.Sprintf("strategy task failures elevated %.1f%%", failRate*100)
		}

		return status, summary, details
	})
}

func (m *Manager) newTradingCheck() Check {
	return newFunctionalCheck("double_loop.trading", "交易执行链路", DomainTrading, CheckModeAll, time.Minute*2, func() (Status, string, map[string]interface{}) {
		automation := m.getAutomation()
		if automation == nil {
			return StatusCritical, "automation system not initialized", map[string]interface{}{"component": "automation_system"}
		}

		executor := automation.GetExecutor()
		if executor == nil {
			return StatusCritical, "realtime executor not available", map[string]interface{}{"component": "executor"}
		}

		stats := executor.GetStats()
		if stats == nil {
			return StatusWarning, "executor stats unavailable", map[string]interface{}{"component": "executor"}
		}

		details := map[string]interface{}{
			"total_actions":      stats.TotalActions,
			"executed_actions":   stats.ExecutedActions,
			"failed_actions":     stats.FailedActions,
			"successful_actions": stats.SuccessfulActions,
			"queue_length":       stats.QueueLength,
			"last_execution":     stats.LastExecutionTime,
		}

		if stats.TotalActions == 0 {
			return StatusWarning, "no trading actions processed yet", details
		}

		total := stats.ExecutedActions + stats.FailedActions
		var failRate float64
		if total > 0 {
			failRate = float64(stats.FailedActions) / float64(total)
		}
		details["failure_rate"] = failRate

		status := StatusHealthy
		summary := fmt.Sprintf("queue:%d failure:%.1f%%", stats.QueueLength, failRate*100)

		if stats.QueueLength > 500 || failRate > 0.10 {
			status = StatusCritical
			summary = fmt.Sprintf("trading pipeline unhealthy (queue %d, failure %.1f%%)", stats.QueueLength, failRate*100)
		} else if stats.QueueLength > 200 || failRate > 0.03 {
			status = StatusWarning
			summary = fmt.Sprintf("trading pressure detected (queue %d, failure %.1f%%)", stats.QueueLength, failRate*100)
		}

		return status, summary, details
	})
}

func (m *Manager) newExchangeKeyCheck() Check {
	return newFunctionalCheck("exchange.keys", "交易所 Key 验证", DomainExchangeKeys, CheckModeAll, time.Minute*3, func() (Status, string, map[string]interface{}) {
		cfg := m.cfg
		modeName := m.modeName
		modeCfg := m.modeConfig

		details := map[string]interface{}{
			"mode": modeName,
		}

		if cfg == nil || modeCfg == nil {
			return StatusWarning, "operational mode not configured", details
		}

		details["required_profile"] = modeCfg.RequiredKeyProfile

		profile := cfg.Operational.GetKeyProfile(modeCfg.RequiredKeyProfile)
		if modeCfg.RequiredKeyProfile != "" && profile == nil {
			return StatusCritical, fmt.Sprintf("required key profile %s missing", modeCfg.RequiredKeyProfile), details
		}

		keyPresent := cfg.Exchange.APIKey != "" && cfg.Exchange.APISecret != ""
		details["key_present"] = keyPresent
		details["base_url"] = cfg.Exchange.BaseURL
		details["futures_base_url"] = cfg.Exchange.FuturesBaseURL

		if modeCfg.RequireTrading && !keyPresent {
			return StatusCritical, "exchange key missing for trading mode", details
		}

		status := StatusHealthy
		summary := "exchange credentials validated"

		if profile != nil {
			details["profile_market"] = profile.Market
			details["profile_environment"] = profile.Environment
			if profile.BaseURL != "" && cfg.Exchange.BaseURL != "" && !strings.EqualFold(profile.BaseURL, cfg.Exchange.BaseURL) {
				status = StatusWarning
				summary = fmt.Sprintf("base url mismatch (expected %s)", profile.BaseURL)
				details["expected_base_url"] = profile.BaseURL
			}
			if profile.FuturesBaseURL != "" && cfg.Exchange.FuturesBaseURL != "" && !strings.EqualFold(profile.FuturesBaseURL, cfg.Exchange.FuturesBaseURL) {
				status = combineStatus(status, StatusWarning)
				summary = fmt.Sprintf("futures base url mismatch (expected %s)", profile.FuturesBaseURL)
				details["expected_futures_base_url"] = profile.FuturesBaseURL
			}
		}

		exch := m.getExchange()
		if exch == nil {
			if modeCfg.RequireTrading {
				return StatusCritical, "exchange client not initialized", details
			}
			return StatusWarning, "exchange client unavailable", details
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if _, err := exch.GetServerTime(ctx); err != nil {
			return StatusCritical, fmt.Sprintf("exchange ping failed: %v", err), details
		}

		account, err := exch.GetAccount(ctx)
		if err != nil {
			return StatusCritical, fmt.Sprintf("account fetch failed: %v", err), details
		}

		details["balances"] = len(account.Balances)
		if len(account.Balances) == 0 {
			status = combineStatus(status, StatusWarning)
			summary = "account accessible but no balances returned"
		}

		if modeCfg.RequireTrading {
			if positions, err := exch.GetPositions(ctx); err != nil {
				status = StatusWarning
				summary = fmt.Sprintf("positions fetch failed: %v", err)
				details["positions_error"] = err.Error()
			} else {
				details["positions_checked"] = len(positions)
			}
		}

		return status, summary, details
	})
}

func (m *Manager) newSystemCheck() Check {
	return newFunctionalCheck("system.health", "系统基础健康", DomainSystem, CheckModeAll, time.Minute*2, func() (Status, string, map[string]interface{}) {
		health := m.getHealthChecker()
		if health == nil {
			return StatusWarning, "health checker not initialized", nil
		}

		overall := health.GetOverallHealth()
		statusText, _ := overall["status"].(string)
		var status Status
		switch strings.ToLower(statusText) {
		case "healthy":
			status = StatusHealthy
		case "degraded":
			status = StatusWarning
		case "unhealthy":
			status = StatusCritical
		default:
			status = StatusUnknown
		}

		totalChecks, _ := overall["total_checks"].(int)
		healthyCount, _ := overall["healthy"].(int)
		summary := fmt.Sprintf("overall=%s (%d/%d healthy)", statusText, healthyCount, totalChecks)

		return status, summary, overall
	})
}
