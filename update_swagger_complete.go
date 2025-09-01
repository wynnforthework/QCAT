package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
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
	// 投资组合管理API
	paths["/portfolio/overview"] = createAPIPath("投资组合", "GET", "获取投资组合概览", "获取投资组合的整体状态和收益情况")
	paths["/portfolio/positions"] = createAPIPath("投资组合", "GET", "获取持仓信息", "获取当前所有持仓的详细信息")
	paths["/portfolio/performance"] = createAPIPath("投资组合", "GET", "获取投资组合性能", "获取投资组合的历史性能数据")

	// 风险管理API
	paths["/risk/overview"] = createAPIPath("风险管理", "GET", "获取风险概览", "获取系统整体风险状况")
	paths["/risk/limits"] = createAPIPath("风险管理", "GET", "获取风险限制", "获取当前设置的风险限制参数")
	paths["/risk/violations"] = createAPIPath("风险管理", "GET", "获取风险违规记录", "获取风险违规的历史记录")

	// 市场数据API
	paths["/market/data"] = createAPIPath("市场数据", "GET", "获取市场数据", "获取实时市场行情数据")

	// 交易活动API
	paths["/trading/activity"] = createAPIPath("交易活动", "GET", "获取交易活动", "获取最近的交易活动记录")
	paths["/trading/history"] = createAPIPath("交易活动", "GET", "获取交易历史", "获取历史交易记录")
	paths["/trading/positions"] = createAPIPath("交易活动", "GET", "获取交易持仓", "获取当前交易持仓信息")

	// 系统监控API
	paths["/metrics/system"] = createAPIPath("系统监控", "GET", "获取系统指标", "获取系统性能和资源使用情况")

	// 审计日志API
	paths["/audit/logs"] = createAPIPath("审计日志", "GET", "获取审计日志", "获取系统操作的审计日志")
	paths["/audit/decisions"] = createAPIPath("审计日志", "GET", "获取决策链", "获取交易决策的完整链路")
	paths["/audit/performance"] = createAPIPath("审计日志", "GET", "获取性能指标", "获取系统性能的审计数据")

	// 缓存管理API
	paths["/cache/status"] = createAPIPath("缓存管理", "GET", "获取缓存状态", "获取缓存系统的运行状态")
	paths["/cache/health"] = createAPIPath("缓存管理", "GET", "缓存健康检查", "检查缓存系统的健康状况")
	paths["/cache/metrics"] = createAPIPath("缓存管理", "GET", "获取缓存指标", "获取缓存的性能指标")

	// 自动化系统API
	paths["/automation/status"] = createAPIPath("自动化系统", "GET", "获取自动化状态", "获取自动化系统的运行状态")
	paths["/automation/health"] = createAPIPath("自动化系统", "GET", "自动化健康检查", "检查自动化系统的健康状况")
	paths["/automation/stats"] = createAPIPath("自动化系统", "GET", "获取执行统计", "获取自动化任务的执行统计")

	// 编排器API
	paths["/orchestrator/status"] = createAPIPath("系统编排", "GET", "获取编排器状态", "获取系统编排器的运行状态")
	paths["/orchestrator/services"] = createAPIPath("系统编排", "GET", "获取服务列表", "获取所有管理的服务状态")
	paths["/orchestrator/health"] = createAPIPath("系统编排", "GET", "编排器健康检查", "检查编排器的健康状况")

	// 热点列表API
	paths["/hotlist/symbols"] = createAPIPath("热点管理", "GET", "获取热点符号", "获取当前热点交易符号列表")
	paths["/hotlist/whitelist"] = createAPIPath("热点管理", "GET", "获取白名单", "获取交易白名单")

	// 黑名单API
	paths["/blacklist"] = createAPIPath("黑名单管理", "GET", "获取黑名单", "获取策略黑名单列表")

	// 并发管理API
	paths["/concurrent/pools"] = createAPIPath("并发管理", "GET", "获取线程池状态", "获取并发线程池的状态信息")
	paths["/concurrent/monitor"] = createAPIPath("并发管理", "GET", "获取监控统计", "获取并发系统的监控数据")

	// 安全管理API
	paths["/security/keys"] = createAPIPath("安全管理", "GET", "获取API密钥", "获取API密钥管理信息")
	paths["/security/audit/logs"] = createAPIPath("安全管理", "GET", "获取安全审计日志", "获取安全相关的审计日志")

	// 设置API
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
