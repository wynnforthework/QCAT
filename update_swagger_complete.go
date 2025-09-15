//go:build tools

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
)

func main() {
	fmt.Println("=== 更新完整的QCAT API Swagger文档 ===")

	// 读取现有的swagger.json
	existingData, err := ioutil.ReadFile("docs/swagger.json")
	if err != nil {
		log.Fatalf("读取现有swagger.json失败: %v", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(existingData, &doc); err != nil {
		log.Fatalf("解析现有swagger.json失败: %v", err)
	}

	// 获取paths对象
	paths, ok := doc["paths"].(map[string]interface{})
	if !ok {
		paths = make(map[string]interface{})
		doc["paths"] = paths
	}

	// 添加更多API路径
	addMoreAPIPaths(paths)

	// 更新info
	if info, ok := doc["info"].(map[string]interface{}); ok {
		info["description"] = "Quantitative Contract Automated Trading System - 完整API文档，包含所有已实现的接口"
	}

	// 生成更新后的JSON
	jsonData, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		log.Fatalf("生成JSON失败: %v", err)
	}

	// 写入更新后的swagger.json
	if err := ioutil.WriteFile("docs/swagger.json", jsonData, 0644); err != nil {
		log.Fatalf("写入swagger.json失败: %v", err)
	}

	fmt.Printf("✅ 完整的Swagger文档已更新: docs/swagger.json\n")
	fmt.Printf("📊 包含 %d 个API端点\n", len(paths))
	fmt.Println("🌐 启动API服务后可访问: http://localhost:8082/swagger/index.html")
}

func addMoreAPIPaths(paths map[string]interface{}) {
	// 基于实际路由配置添加API端点

	// 仪表盘API
	paths["/dashboard"] = createAPIPath("仪表盘", "GET", "获取仪表盘数据", "获取系统概览和关键指标")
	paths["/dashboard/db-health"] = createAPIPath("仪表盘", "GET", "获取数据库健康状态", "获取数据库连接和性能状态")

	// 策略管理API (扩展)
	paths["/strategy/:id"] = createAPIPath("策略管理", "GET", "获取策略详情", "根据ID获取特定策略的详细信息")
	paths["/strategy/execution/overview"] = createAPIPath("策略管理", "GET", "获取执行概览", "获取策略执行的整体状态")
	paths["/strategy/execution/realtime"] = createAPIPath("策略管理", "GET", "获取实时状态", "获取策略的实时执行状态")
	paths["/strategy/workflow/status"] = createAPIPath("策略管理", "GET", "获取工作流状态", "获取策略工作流的运行状态")

	// 投资组合管理API
	paths["/portfolio/overview"] = createAPIPath("投资组合", "GET", "获取投资组合概览", "获取投资组合的整体状态和收益情况")
	paths["/portfolio/positions"] = createAPIPath("投资组合", "GET", "获取持仓信息", "获取当前所有持仓的详细信息")
	paths["/portfolio/performance"] = createAPIPath("投资组合", "GET", "获取投资组合性能", "获取投资组合的历史性能数据")

	// 风险管理API
	paths["/risk/overview"] = createAPIPath("风险管理", "GET", "获取风险概览", "获取系统整体风险状况")
	paths["/risk/limits"] = createAPIPath("风险管理", "GET", "获取风险限制", "获取当前设置的风险限制参数")
	paths["/risk/violations"] = createAPIPath("风险管理", "GET", "获取风险违规记录", "获取风险违规的历史记录")

	// 交易活动API (扩展)
	paths["/trading/history"] = createAPIPath("交易活动", "GET", "获取交易历史", "获取历史交易记录")
	paths["/trading/positions"] = createAPIPath("交易活动", "GET", "获取交易持仓", "获取当前交易持仓信息")

	// 自动启动管理API
	paths["/auto-start/strategies"] = createAPIPath("自动启动", "GET", "获取自动启动策略", "获取配置为自动启动的策略列表")
	paths["/auto-start/stats"] = createAPIPath("自动启动", "GET", "获取自动启动统计", "获取自动启动的执行统计信息")
	paths["/auto-start/trigger"] = createAPIPath("自动启动", "POST", "触发自动启动", "手动触发自动启动流程")

	// 黑名单管理API (完整CRUD)
	paths["/blacklist/"] = map[string]interface{}{
		"get":  createMethodDef("黑名单管理", "获取黑名单列表", "获取策略黑名单列表", true),
		"post": createMethodDef("黑名单管理", "添加到黑名单", "将策略添加到黑名单", true),
	}
	paths["/blacklist/:strategy_id"] = map[string]interface{}{
		"get":    createMethodDef("黑名单管理", "检查黑名单状态", "检查特定策略是否在黑名单中", true),
		"delete": createMethodDef("黑名单管理", "从黑名单移除", "从黑名单中移除策略", true),
	}
	paths["/blacklist/clear-expired"] = createAPIPath("黑名单管理", "POST", "清理过期条目", "清理黑名单中的过期条目")

	// 热点列表API
	paths["/hotlist/symbols"] = createAPIPath("热点管理", "GET", "获取热点符号", "获取当前热点交易符号列表")
	paths["/hotlist/whitelist"] = createAPIPath("热点管理", "GET", "获取白名单", "获取交易白名单")

	// 并发管理API
	paths["/concurrent/pools"] = createAPIPath("并发管理", "GET", "获取线程池状态", "获取并发线程池的状态信息")
	paths["/concurrent/monitor"] = createAPIPath("并发管理", "GET", "获取监控统计", "获取并发系统的监控数据")

	// 审计日志API (扩展)
	paths["/audit/decisions"] = createAPIPath("审计日志", "GET", "获取决策链", "获取交易决策的完整链路")
	paths["/audit/performance"] = createAPIPath("审计日志", "GET", "获取性能指标", "获取系统性能的审计数据")
	paths["/audit/export"] = createAPIPath("审计日志", "POST", "导出审计报告", "导出审计日志报告")

	// 缓存管理API (通过RegisterRoutes注册)
	paths["/cache/status"] = createAPIPath("缓存管理", "GET", "获取缓存状态", "获取缓存系统的运行状态")
	paths["/cache/health"] = createAPIPath("缓存管理", "GET", "缓存健康检查", "检查缓存系统的健康状况")
	paths["/cache/metrics"] = createAPIPath("缓存管理", "GET", "获取缓存指标", "获取缓存的性能指标")

	// 安全管理API (通过RegisterRoutes注册)
	paths["/security/keys"] = createAPIPath("安全管理", "GET", "获取API密钥", "获取API密钥管理信息")
	paths["/security/audit/logs"] = createAPIPath("安全管理", "GET", "获取安全审计日志", "获取安全相关的审计日志")

	// 自动化系统API (通过RegisterRoutes注册)
	paths["/automation/status"] = createAPIPath("自动化系统", "GET", "获取自动化状态", "获取自动化系统的运行状态")
	paths["/automation/health"] = createAPIPath("自动化系统", "GET", "自动化健康检查", "检查自动化系统的健康状况")
	paths["/automation/stats"] = createAPIPath("自动化系统", "GET", "获取执行统计", "获取自动化任务的执行统计")

	// 编排器API (扩展)
	paths["/orchestrator/services"] = createAPIPath("系统编排", "GET", "获取服务列表", "获取所有管理的服务状态")
	paths["/orchestrator/services/start"] = createAPIPath("系统编排", "POST", "启动服务", "启动指定的服务")
	paths["/orchestrator/services/stop"] = createAPIPath("系统编排", "POST", "停止服务", "停止指定的服务")
	paths["/orchestrator/services/restart"] = createAPIPath("系统编排", "POST", "重启服务", "重启指定的服务")
	paths["/orchestrator/optimize"] = createAPIPath("系统编排", "POST", "优化系统", "执行系统优化操作")
	paths["/orchestrator/health"] = createAPIPath("系统编排", "GET", "编排器健康检查", "检查编排器的健康状况")

	// 结果分享API
	paths["/share-result"] = createAPIPath("结果分享", "POST", "分享结果", "分享交易或策略结果")
	paths["/shared-results"] = createAPIPath("结果分享", "GET", "获取分享结果", "获取已分享的结果列表")

	// WebSocket API文档
	paths["/ws/market/:symbol"] = createWebSocketPath("WebSocket", "市场数据流", "订阅特定符号的实时市场数据")
	paths["/ws/strategy/:id"] = createWebSocketPath("WebSocket", "策略数据流", "订阅特定策略的实时数据")
	paths["/ws/alerts"] = createWebSocketPath("WebSocket", "告警数据流", "订阅系统告警信息")

	// 设置API (支持GET和PUT)
	paths["/settings"] = map[string]interface{}{
		"get": createMethodDef("系统设置", "获取系统设置", "获取当前系统配置设置", false),
		"put": createMethodDef("系统设置", "更新系统设置", "更新系统配置设置", false),
	}
}

func createAPIPath(tag, method, summary, description string) map[string]interface{} {
	return map[string]interface{}{
		strings.ToLower(method): createMethodDef(tag, summary, description, true),
	}
}

func createMethodDef(tag, summary, description string, requireAuth bool) map[string]interface{} {
	methodDef := map[string]interface{}{
		"tags":        []string{tag},
		"summary":     summary,
		"description": description,
		"produces":    []string{"application/json"},
		"responses": map[string]interface{}{
			"200": map[string]interface{}{
				"description": "成功",
				"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
			},
		},
	}

	if requireAuth {
		methodDef["security"] = []map[string]interface{}{{"BearerAuth": []string{}}}
	}

	return methodDef
}

func createWebSocketPath(tag, summary, description string) map[string]interface{} {
	return map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{tag},
			"summary":     summary,
			"description": description,
			"schemes":     []string{"ws", "wss"},
			"responses": map[string]interface{}{
				"101": map[string]interface{}{
					"description": "WebSocket连接升级成功",
				},
				"400": map[string]interface{}{
					"description": "WebSocket连接失败",
				},
			},
		},
	}
}
