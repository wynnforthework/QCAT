package main

import (
	"fmt"
)

// APIEndpoint represents an API endpoint
type APIEndpoint struct {
	Method       string
	Path         string
	Handler      string
	Category     string
	RequiresAuth bool
	Status       string // working, broken, not_implemented
	Error        string
}

func main() {
	fmt.Println("🔍 QCAT API接口全面分析报告")
	fmt.Println("============================================================")

	// 基于代码分析的完整接口列表
	endpoints := []APIEndpoint{
		// 公共接口
		{Method: "GET", Path: "/health", Handler: "server.getHealthStatus", Category: "公共接口", RequiresAuth: false},
		{Method: "GET", Path: "/swagger/*any", Handler: "ginSwagger.WrapHandler", Category: "文档", RequiresAuth: false},
		{Method: "GET", Path: "/metrics", Handler: "monitoring.PrometheusHandler", Category: "监控", RequiresAuth: false},

		// 认证接口
		{Method: "POST", Path: "/api/v1/auth/login", Handler: "Auth.Login", Category: "认证", RequiresAuth: false},
		{Method: "POST", Path: "/api/v1/auth/register", Handler: "Auth.Register", Category: "认证", RequiresAuth: false},
		{Method: "POST", Path: "/api/v1/auth/refresh", Handler: "Auth.RefreshToken", Category: "认证", RequiresAuth: false},

		// 仪表板
		{Method: "GET", Path: "/api/v1/dashboard", Handler: "Dashboard.GetDashboardData", Category: "仪表板", RequiresAuth: true},

		// 市场数据
		{Method: "GET", Path: "/api/v1/market/data", Handler: "Market.GetMarketData", Category: "市场数据", RequiresAuth: true},

		// 交易活动
		{Method: "GET", Path: "/api/v1/trading/activity", Handler: "Trading.GetTradingActivity", Category: "交易", RequiresAuth: true},

		// 系统指标
		{Method: "GET", Path: "/api/v1/metrics/system", Handler: "Metrics.GetSystemMetrics", Category: "系统指标", RequiresAuth: true},
		{Method: "GET", Path: "/api/v1/metrics/performance", Handler: "Metrics.GetPerformanceMetrics", Category: "系统指标", RequiresAuth: true},
		{Method: "GET", Path: "/api/v1/metrics/strategy/:id", Handler: "Metrics.GetStrategyMetrics", Category: "系统指标", RequiresAuth: true},

		// 策略管理 (9个接口)
		{Method: "GET", Path: "/api/v1/strategy/", Handler: "Strategy.ListStrategies", Category: "策略管理", RequiresAuth: true},
		{Method: "GET", Path: "/api/v1/strategy/:id", Handler: "Strategy.GetStrategy", Category: "策略管理", RequiresAuth: true},
		{Method: "POST", Path: "/api/v1/strategy/", Handler: "Strategy.CreateStrategy", Category: "策略管理", RequiresAuth: true},
		{Method: "PUT", Path: "/api/v1/strategy/:id", Handler: "Strategy.UpdateStrategy", Category: "策略管理", RequiresAuth: true},
		{Method: "DELETE", Path: "/api/v1/strategy/:id", Handler: "Strategy.DeleteStrategy", Category: "策略管理", RequiresAuth: true},
		{Method: "POST", Path: "/api/v1/strategy/:id/promote", Handler: "Strategy.PromoteStrategy", Category: "策略管理", RequiresAuth: true},
		{Method: "POST", Path: "/api/v1/strategy/:id/start", Handler: "Strategy.StartStrategy", Category: "策略管理", RequiresAuth: true},
		{Method: "POST", Path: "/api/v1/strategy/:id/stop", Handler: "Strategy.StopStrategy", Category: "策略管理", RequiresAuth: true},
		{Method: "POST", Path: "/api/v1/strategy/:id/backtest", Handler: "Strategy.RunBacktest", Category: "策略管理", RequiresAuth: true},

		// 优化器 (4个接口)
		{Method: "POST", Path: "/api/v1/optimizer/run", Handler: "Optimizer.RunOptimization", Category: "优化器", RequiresAuth: true},
		{Method: "GET", Path: "/api/v1/optimizer/tasks", Handler: "Optimizer.GetTasks", Category: "优化器", RequiresAuth: true},
		{Method: "GET", Path: "/api/v1/optimizer/tasks/:id", Handler: "Optimizer.GetTask", Category: "优化器", RequiresAuth: true},
		{Method: "GET", Path: "/api/v1/optimizer/results/:id", Handler: "Optimizer.GetResults", Category: "优化器", RequiresAuth: true},

		// 投资组合 (4个接口)
		{Method: "GET", Path: "/api/v1/portfolio/overview", Handler: "Portfolio.GetOverview", Category: "投资组合", RequiresAuth: true},
		{Method: "GET", Path: "/api/v1/portfolio/allocations", Handler: "Portfolio.GetAllocations", Category: "投资组合", RequiresAuth: true},
		{Method: "POST", Path: "/api/v1/portfolio/rebalance", Handler: "Portfolio.Rebalance", Category: "投资组合", RequiresAuth: true},
		{Method: "GET", Path: "/api/v1/portfolio/history", Handler: "Portfolio.GetHistory", Category: "投资组合", RequiresAuth: true},

		// 风险管理 (6个接口)
		{Method: "GET", Path: "/api/v1/risk/overview", Handler: "Risk.GetOverview", Category: "风险管理", RequiresAuth: true},
		{Method: "GET", Path: "/api/v1/risk/limits", Handler: "Risk.GetLimits", Category: "风险管理", RequiresAuth: true},
		{Method: "POST", Path: "/api/v1/risk/limits", Handler: "Risk.SetLimits", Category: "风险管理", RequiresAuth: true},
		{Method: "GET", Path: "/api/v1/risk/circuit-breakers", Handler: "Risk.GetCircuitBreakers", Category: "风险管理", RequiresAuth: true},
		{Method: "POST", Path: "/api/v1/risk/circuit-breakers", Handler: "Risk.SetCircuitBreakers", Category: "风险管理", RequiresAuth: true},
		{Method: "GET", Path: "/api/v1/risk/violations", Handler: "Risk.GetViolations", Category: "风险管理", RequiresAuth: true},

		// 热门列表 (5个接口)
		{Method: "GET", Path: "/api/v1/hotlist/symbols", Handler: "Hotlist.GetHotSymbols", Category: "热门列表", RequiresAuth: true},
		{Method: "POST", Path: "/api/v1/hotlist/approve", Handler: "Hotlist.ApproveSymbol", Category: "热门列表", RequiresAuth: true},
		{Method: "GET", Path: "/api/v1/hotlist/whitelist", Handler: "Hotlist.GetWhitelist", Category: "热门列表", RequiresAuth: true},
		{Method: "POST", Path: "/api/v1/hotlist/whitelist", Handler: "Hotlist.AddToWhitelist", Category: "热门列表", RequiresAuth: true},
		{Method: "DELETE", Path: "/api/v1/hotlist/whitelist/:symbol", Handler: "Hotlist.RemoveFromWhitelist", Category: "热门列表", RequiresAuth: true},

		// 健康检查 (4个接口)
		{Method: "GET", Path: "/api/v1/health/status", Handler: "server.getHealthStatus", Category: "健康检查", RequiresAuth: true},
		{Method: "GET", Path: "/api/v1/health/checks", Handler: "server.getAllHealthChecks", Category: "健康检查", RequiresAuth: true},
		{Method: "GET", Path: "/api/v1/health/checks/:name", Handler: "server.getHealthCheck", Category: "健康检查", RequiresAuth: true},
		{Method: "POST", Path: "/api/v1/health/checks/:name/force", Handler: "server.forceHealthCheck", Category: "健康检查", RequiresAuth: true},

		// 系统管理 (3个接口)
		{Method: "GET", Path: "/api/v1/shutdown/status", Handler: "server.getShutdownStatus", Category: "系统管理", RequiresAuth: true},
		{Method: "POST", Path: "/api/v1/shutdown/graceful", Handler: "server.initiateGracefulShutdown", Category: "系统管理", RequiresAuth: true},
		{Method: "POST", Path: "/api/v1/shutdown/force", Handler: "server.forceShutdown", Category: "系统管理", RequiresAuth: true},

		// 审计 (4个接口)
		{Method: "GET", Path: "/api/v1/audit/logs", Handler: "Audit.GetLogs", Category: "审计", RequiresAuth: true},
		{Method: "GET", Path: "/api/v1/audit/decisions", Handler: "Audit.GetDecisionChains", Category: "审计", RequiresAuth: true},
		{Method: "GET", Path: "/api/v1/audit/performance", Handler: "Audit.GetPerformanceMetrics", Category: "审计", RequiresAuth: true},
		{Method: "POST", Path: "/api/v1/audit/export", Handler: "Audit.ExportReport", Category: "审计", RequiresAuth: true},

		// 缓存管理 (8个接口)
		{Method: "GET", Path: "/api/v1/cache/status", Handler: "Cache.handleCacheStatus", Category: "缓存管理", RequiresAuth: true},
		{Method: "GET", Path: "/api/v1/cache/health", Handler: "Cache.handleCacheHealth", Category: "缓存管理", RequiresAuth: true},
		{Method: "GET", Path: "/api/v1/cache/metrics", Handler: "Cache.handleCacheMetrics", Category: "缓存管理", RequiresAuth: true},
		{Method: "GET", Path: "/api/v1/cache/events", Handler: "Cache.handleCacheEvents", Category: "缓存管理", RequiresAuth: true},
		{Method: "GET", Path: "/api/v1/cache/config", Handler: "Cache.handleCacheConfig", Category: "缓存管理", RequiresAuth: true},
		{Method: "POST", Path: "/api/v1/cache/test", Handler: "Cache.handleTestCache", Category: "缓存管理", RequiresAuth: true},
		{Method: "POST", Path: "/api/v1/cache/fallback/force", Handler: "Cache.handleForceFallback", Category: "缓存管理", RequiresAuth: true},
		{Method: "POST", Path: "/api/v1/cache/counters/reset", Handler: "Cache.handleResetCounters", Category: "缓存管理", RequiresAuth: true},

		// 安全管理 (11个接口)
		{Method: "POST", Path: "/api/v1/security/keys/", Handler: "Security.createAPIKey", Category: "安全管理", RequiresAuth: true},
		{Method: "GET", Path: "/api/v1/security/keys/", Handler: "Security.listAPIKeys", Category: "安全管理", RequiresAuth: true},
		{Method: "GET", Path: "/api/v1/security/keys/:keyId", Handler: "Security.getAPIKey", Category: "安全管理", RequiresAuth: true},
		{Method: "POST", Path: "/api/v1/security/keys/:keyId/rotate", Handler: "Security.rotateAPIKey", Category: "安全管理", RequiresAuth: true},
		{Method: "POST", Path: "/api/v1/security/keys/:keyId/revoke", Handler: "Security.revokeAPIKey", Category: "安全管理", RequiresAuth: true},
		{Method: "GET", Path: "/api/v1/security/keys/:keyId/usage", Handler: "Security.getKeyUsage", Category: "安全管理", RequiresAuth: true},
		{Method: "GET", Path: "/api/v1/security/keys/:keyId/schedule", Handler: "Security.getRotationSchedule", Category: "安全管理", RequiresAuth: true},
		{Method: "GET", Path: "/api/v1/security/audit/logs", Handler: "Security.getAuditLogs", Category: "安全管理", RequiresAuth: true},
		{Method: "GET", Path: "/api/v1/security/audit/logs/:id", Handler: "Security.getAuditLog", Category: "安全管理", RequiresAuth: true},
		{Method: "POST", Path: "/api/v1/security/audit/logs/export", Handler: "Security.exportAuditLogs", Category: "安全管理", RequiresAuth: true},
		{Method: "GET", Path: "/api/v1/security/audit/integrity", Handler: "Security.verifyIntegrity", Category: "安全管理", RequiresAuth: true},

		// 编排器 (7个接口)
		{Method: "GET", Path: "/api/v1/orchestrator/status", Handler: "Orchestrator.handleStatus", Category: "编排器", RequiresAuth: true},
		{Method: "GET", Path: "/api/v1/orchestrator/services", Handler: "Orchestrator.handleServices", Category: "编排器", RequiresAuth: true},
		{Method: "POST", Path: "/api/v1/orchestrator/services/start", Handler: "Orchestrator.handleStartService", Category: "编排器", RequiresAuth: true},
		{Method: "POST", Path: "/api/v1/orchestrator/services/stop", Handler: "Orchestrator.handleStopService", Category: "编排器", RequiresAuth: true},
		{Method: "POST", Path: "/api/v1/orchestrator/services/restart", Handler: "Orchestrator.handleRestartService", Category: "编排器", RequiresAuth: true},
		{Method: "POST", Path: "/api/v1/orchestrator/optimize", Handler: "Orchestrator.handleOptimize", Category: "编排器", RequiresAuth: true},
		{Method: "GET", Path: "/api/v1/orchestrator/health", Handler: "Orchestrator.handleHealth", Category: "编排器", RequiresAuth: true},

		// WebSocket接口 (3个接口)
		{Method: "GET", Path: "/ws/market/:symbol", Handler: "WebSocket.MarketStream", Category: "WebSocket", RequiresAuth: false},
		{Method: "GET", Path: "/ws/strategy/:id", Handler: "WebSocket.StrategyStream", Category: "WebSocket", RequiresAuth: false},
		{Method: "GET", Path: "/ws/alerts", Handler: "WebSocket.AlertsStream", Category: "WebSocket", RequiresAuth: false},
	}

	// 统计分析
	totalEndpoints := len(endpoints)
	categoryCounts := make(map[string]int)
	authRequired := 0
	publicEndpoints := 0

	for _, ep := range endpoints {
		categoryCounts[ep.Category]++
		if ep.RequiresAuth {
			authRequired++
		} else {
			publicEndpoints++
		}
	}

	// 输出统计结果
	fmt.Printf("📊 接口统计概览:\n")
	fmt.Printf("总接口数: %d\n", totalEndpoints)
	fmt.Printf("需要认证: %d (%.1f%%)\n", authRequired, float64(authRequired)/float64(totalEndpoints)*100)
	fmt.Printf("公共接口: %d (%.1f%%)\n", publicEndpoints, float64(publicEndpoints)/float64(totalEndpoints)*100)
	fmt.Println()

	fmt.Println("📋 按分类统计:")
	for category, count := range categoryCounts {
		fmt.Printf("  %-12s: %d个接口\n", category, count)
	}
	fmt.Println()

	fmt.Println("📝 详细接口列表:")
	currentCategory := ""
	for _, ep := range endpoints {
		if ep.Category != currentCategory {
			fmt.Printf("\n🔸 %s:\n", ep.Category)
			currentCategory = ep.Category
		}
		authStatus := "🔒"
		if !ep.RequiresAuth {
			authStatus = "🌐"
		}
		fmt.Printf("  %s %s %-6s %s\n", authStatus, ep.Method, "", ep.Path)
	}

	fmt.Println("\n📍 图例:")
	fmt.Println("  🔒 需要认证")
	fmt.Println("  🌐 公共接口")
}
