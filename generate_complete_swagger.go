//go:build tools

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

// SwaggerDoc represents the complete Swagger documentation structure
type SwaggerDoc struct {
	Swagger             string                 `json:"swagger"`
	Info                SwaggerInfo            `json:"info"`
	Host                string                 `json:"host"`
	BasePath            string                 `json:"basePath"`
	Schemes             []string               `json:"schemes"`
	Paths               map[string]interface{} `json:"paths"`
	Definitions         map[string]interface{} `json:"definitions"`
	SecurityDefinitions map[string]interface{} `json:"securityDefinitions"`
}

type SwaggerInfo struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Version     string         `json:"version"`
	Contact     SwaggerContact `json:"contact"`
	License     SwaggerLicense `json:"license"`
}

type SwaggerContact struct {
	Name  string `json:"name"`
	URL   string `json:"url"`
	Email string `json:"email"`
}

type SwaggerLicense struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func main() {
	fmt.Println("=== 生成完整的QCAT API Swagger文档 ===")

	// 创建完整的Swagger文档
	doc := SwaggerDoc{
		Swagger:  "2.0",
		Host:     "localhost:8082",
		BasePath: "/api/v1",
		Schemes:  []string{"http", "https"},
		Info: SwaggerInfo{
			Title:       "QCAT API",
			Description: "Quantitative Contract Automated Trading System - 完整API文档",
			Version:     "1.0",
			Contact: SwaggerContact{
				Name:  "QCAT Team",
				URL:   "https://github.com/qcat",
				Email: "support@qcat.local",
			},
			License: SwaggerLicense{
				Name: "Apache 2.0",
				URL:  "http://www.apache.org/licenses/LICENSE-2.0.html",
			},
		},
		SecurityDefinitions: map[string]interface{}{
			"BearerAuth": map[string]interface{}{
				"type":        "apiKey",
				"name":        "Authorization",
				"in":          "header",
				"description": "Type 'Bearer' followed by a space and JWT token.",
			},
		},
		Paths: make(map[string]interface{}),
		Definitions: map[string]interface{}{
			"Response": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"success": map[string]interface{}{"type": "boolean"},
					"data":    map[string]interface{}{"type": "object"},
					"error":   map[string]interface{}{"type": "string"},
					"message": map[string]interface{}{"type": "string"},
				},
			},
		},
	}

	// 添加所有已实现的API路径
	addAuthPaths(doc.Paths)
	addDashboardPaths(doc.Paths)
	addStrategyPaths(doc.Paths)
	addPortfolioPaths(doc.Paths)
	addRiskPaths(doc.Paths)
	addMarketPaths(doc.Paths)
	addTradingPaths(doc.Paths)
	addMetricsPaths(doc.Paths)
	addAuditPaths(doc.Paths)

	addOrchestratorPaths(doc.Paths)
	addHotlistPaths(doc.Paths)
	addBlacklistPaths(doc.Paths)
	addConcurrentPaths(doc.Paths)

	addOptimizerPaths(doc.Paths)
	addAutoStartPaths(doc.Paths)
	addEmergencyPaths(doc.Paths)
	addHealthPaths(doc.Paths)
	addMemoryPaths(doc.Paths)
	addNetworkPaths(doc.Paths)
	addShutdownPaths(doc.Paths)
	addValidationPaths(doc.Paths)
	addWorkflowPaths(doc.Paths)
	addSharedResultsPaths(doc.Paths)
	addSettingsPaths(doc.Paths)
	addWebSocketPaths(doc.Paths)
	addBasicHealthPaths(doc.Paths)

	// 生成JSON文件
	jsonData, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		log.Fatalf("生成JSON失败: %v", err)
	}

	// 确保docs目录存在
	if err := os.MkdirAll("docs", 0755); err != nil {
		log.Fatalf("创建docs目录失败: %v", err)
	}

	// 写入swagger.json
	if err := ioutil.WriteFile("docs/swagger.json", jsonData, 0644); err != nil {
		log.Fatalf("写入swagger.json失败: %v", err)
	}

	fmt.Printf("✅ 完整的Swagger文档已生成: docs/swagger.json\n")
	fmt.Printf("📊 包含 %d 个API端点\n", len(doc.Paths))
	fmt.Println("🌐 启动API服务后可访问: http://localhost:8082/swagger/index.html")
}

// 添加认证相关API
func addAuthPaths(paths map[string]interface{}) {
	paths["/auth/login"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"认证"},
			"summary":     "用户登录",
			"description": "用户登录认证",
			"consumes":    []string{"application/json"},
			"produces":    []string{"application/json"},
			"parameters": []map[string]interface{}{
				{
					"name":        "body",
					"in":          "body",
					"required":    true,
					"description": "登录信息",
					"schema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"username": map[string]interface{}{"type": "string"},
							"password": map[string]interface{}{"type": "string"},
						},
						"required": []string{"username", "password"},
					},
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "登录成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
				"401": map[string]interface{}{
					"description": "认证失败",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/auth/register"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"认证"},
			"summary":     "用户注册",
			"description": "注册新用户",
			"consumes":    []string{"application/json"},
			"produces":    []string{"application/json"},
			"parameters": []map[string]interface{}{
				{
					"name":        "body",
					"in":          "body",
					"required":    true,
					"description": "注册信息",
					"schema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"username": map[string]interface{}{"type": "string"},
							"password": map[string]interface{}{"type": "string"},
							"email":    map[string]interface{}{"type": "string"},
						},
						"required": []string{"username", "password", "email"},
					},
				},
			},
			"responses": map[string]interface{}{
				"201": map[string]interface{}{
					"description": "注册成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
				"400": map[string]interface{}{
					"description": "请求参数错误",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/auth/profile"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"认证"},
			"summary":     "获取用户信息",
			"description": "获取当前登录用户的详细信息",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "用户信息",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
				"401": map[string]interface{}{
					"description": "未授权",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/auth/refresh"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"认证"},
			"summary":     "刷新访问令牌",
			"description": "使用刷新令牌获取新的访问令牌",
			"consumes":    []string{"application/json"},
			"produces":    []string{"application/json"},
			"parameters": []map[string]interface{}{
				{
					"name":        "body",
					"in":          "body",
					"required":    true,
					"description": "刷新令牌",
					"schema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"refresh_token": map[string]interface{}{"type": "string"},
						},
						"required": []string{"refresh_token"},
					},
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "令牌刷新成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
				"401": map[string]interface{}{
					"description": "刷新令牌无效",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}
}

// 添加仪表盘相关API
func addDashboardPaths(paths map[string]interface{}) {
	paths["/dashboard"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"仪表盘"},
			"summary":     "获取仪表盘数据",
			"description": "获取系统概览和关键指标",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "仪表盘数据",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/dashboard/db-health"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"仪表盘"},
			"summary":     "数据库健康检查",
			"description": "检查数据库连接状态和性能",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "数据库健康状态",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}
}

// 添加策略相关API
func addStrategyPaths(paths map[string]interface{}) {
	paths["/strategy"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"策略管理"},
			"summary":     "获取策略列表",
			"description": "获取所有交易策略的列表",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "策略列表",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
		"post": map[string]interface{}{
			"tags":        []string{"策略管理"},
			"summary":     "创建新策略",
			"description": "创建一个新的交易策略",
			"consumes":    []string{"application/json"},
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "body",
					"in":          "body",
					"required":    true,
					"description": "策略信息",
					"schema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"name":        map[string]interface{}{"type": "string"},
							"description": map[string]interface{}{"type": "string"},
							"type":        map[string]interface{}{"type": "string"},
							"config":      map[string]interface{}{"type": "object"},
						},
						"required": []string{"name", "type"},
					},
				},
			},
			"responses": map[string]interface{}{
				"201": map[string]interface{}{
					"description": "策略创建成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/strategy/{id}"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"策略管理"},
			"summary":     "获取策略详情",
			"description": "根据ID获取特定策略的详细信息",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "id",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "策略ID",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "策略详情",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
				"404": map[string]interface{}{
					"description": "策略不存在",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
		"put": map[string]interface{}{
			"tags":        []string{"策略管理"},
			"summary":     "更新策略",
			"description": "更新指定策略的配置",
			"consumes":    []string{"application/json"},
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "id",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "策略ID",
				},
				{
					"name":        "body",
					"in":          "body",
					"required":    true,
					"description": "更新的策略信息",
					"schema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"name":        map[string]interface{}{"type": "string"},
							"description": map[string]interface{}{"type": "string"},
							"config":      map[string]interface{}{"type": "object"},
						},
					},
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "策略更新成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
		"delete": map[string]interface{}{
			"tags":        []string{"策略管理"},
			"summary":     "删除策略",
			"description": "删除指定的策略",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "id",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "策略ID",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "策略删除成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	// 策略池概览
	paths["/strategy/pool/overview"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"策略管理"},
			"summary":     "获取策略池概览",
			"description": "获取策略池的整体状态和统计信息",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "策略池概览",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	// 策略执行概览
	paths["/strategy/execution/overview"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"策略管理"},
			"summary":     "获取策略执行概览",
			"description": "获取策略执行的整体状态",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "策略执行概览",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	// 策略操作
	paths["/strategy/{id}/start"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"策略管理"},
			"summary":     "启动策略",
			"description": "启动指定的交易策略",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "id",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "策略ID",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "策略启动成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/strategy/{id}/stop"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"策略管理"},
			"summary":     "停止策略",
			"description": "停止指定的交易策略",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "id",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "策略ID",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "策略停止成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	// 策略回测
	paths["/strategy/{id}/backtest"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"策略管理"},
			"summary":     "策略回测",
			"description": "对指定策略进行回测",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "id",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "策略ID",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "回测启动成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	// 策略提升
	paths["/strategy/{id}/promote"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"策略管理"},
			"summary":     "策略提升",
			"description": "将策略提升到生产环境",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "id",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "策略ID",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "策略提升成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	// 策略实时执行状态
	paths["/strategy/execution/realtime"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"策略管理"},
			"summary":     "获取实时执行状态",
			"description": "获取策略的实时执行状态",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "实时执行状态",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	// 策略工作流状态
	paths["/strategy/workflow/status"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"策略管理"},
			"summary":     "获取工作流状态",
			"description": "获取策略工作流的运行状态",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "工作流状态",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	// 策略基础操作
	paths["/strategy/:id"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"策略管理"},
			"summary":     "获取策略详情",
			"description": "获取指定策略的详细信息",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "id",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "策略ID",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "策略详情",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/strategy/:id/start"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"策略管理"},
			"summary":     "启动策略",
			"description": "启动指定的交易策略",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "id",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "策略ID",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "策略启动成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/strategy/:id/stop"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"策略管理"},
			"summary":     "停止策略",
			"description": "停止指定的交易策略",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "id",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "策略ID",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "策略停止成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/strategy/:id/backtest"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"策略管理"},
			"summary":     "策略回测",
			"description": "对指定策略进行回测",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "id",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "策略ID",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "回测启动成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/strategy/:id/promote"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"策略管理"},
			"summary":     "策略提升",
			"description": "将策略提升到生产环境",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "id",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "策略ID",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "策略提升成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/strategy/:id/auto-start"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"策略管理"},
			"summary":     "设置策略自动启动",
			"description": "为指定策略设置自动启动",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "id",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "策略ID",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "设置成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}
}

// 添加投资组合相关API
func addPortfolioPaths(paths map[string]interface{}) {
	paths["/portfolio/overview"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"投资组合"},
			"summary":     "获取投资组合概览",
			"description": "获取投资组合的整体状态和统计信息",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "投资组合概览",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/portfolio/allocations"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"投资组合"},
			"summary":     "获取资产配置",
			"description": "获取投资组合的资产配置信息",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "资产配置信息",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/portfolio/performance"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"投资组合"},
			"summary":     "获取投资组合表现",
			"description": "获取投资组合的历史表现数据",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "投资组合表现",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/portfolio/history"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"投资组合"},
			"summary":     "获取投资组合历史",
			"description": "获取投资组合的历史记录",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "投资组合历史",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/portfolio/rebalance"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"投资组合"},
			"summary":     "重新平衡投资组合",
			"description": "执行投资组合重新平衡操作",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "重新平衡成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}
}

// 添加风险管理相关API
func addRiskPaths(paths map[string]interface{}) {
	paths["/risk/overview"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"风险管理"},
			"summary":     "获取风险概览",
			"description": "获取系统风险管理的整体状态",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "风险概览",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/risk/limits"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"风险管理"},
			"summary":     "获取风险限制",
			"description": "获取当前的风险限制设置",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "风险限制",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/risk/violations"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"风险管理"},
			"summary":     "获取风险违规",
			"description": "获取风险违规记录",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "风险违规记录",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/risk/circuit-breakers"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"风险管理"},
			"summary":     "获取熔断器状态",
			"description": "获取系统熔断器的状态",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "熔断器状态",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}
}

// 添加市场数据相关API
func addMarketPaths(paths map[string]interface{}) {
	paths["/market/data"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"市场数据"},
			"summary":     "获取市场数据",
			"description": "获取实时市场数据",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "市场数据",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}
}

// 添加交易相关API
func addTradingPaths(paths map[string]interface{}) {
	paths["/trading/activity"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"交易管理"},
			"summary":     "获取交易活动",
			"description": "获取当前的交易活动信息",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "交易活动",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/trading/history"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"交易管理"},
			"summary":     "获取交易历史",
			"description": "获取历史交易记录",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "交易历史",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/trading/positions"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"交易管理"},
			"summary":     "获取持仓信息",
			"description": "获取当前的持仓信息",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "持仓信息",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}
}

// 添加指标相关API
func addMetricsPaths(paths map[string]interface{}) {
	paths["/metrics/system"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"系统指标"},
			"summary":     "获取系统指标",
			"description": "获取系统性能指标",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "系统指标",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/metrics/performance"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"系统指标"},
			"summary":     "获取性能指标",
			"description": "获取系统性能指标",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "性能指标",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/metrics/strategy/:id"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"系统指标"},
			"summary":     "获取策略指标",
			"description": "获取特定策略的性能指标",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "id",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "策略ID",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "策略指标",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}
}

// 添加审计相关API
func addAuditPaths(paths map[string]interface{}) {
	paths["/audit/logs"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"审计日志"},
			"summary":     "获取审计日志",
			"description": "获取系统审计日志",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "审计日志",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/audit/decisions"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"审计日志"},
			"summary":     "获取决策链",
			"description": "获取交易决策的完整链路",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "决策链",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/audit/performance"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"审计日志"},
			"summary":     "获取性能审计",
			"description": "获取系统性能的审计数据",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "性能审计",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/audit/export"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"审计日志"},
			"summary":     "导出审计报告",
			"description": "导出审计日志报告",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "导出成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}
}

// 添加缓存管理相关API
func addCachePaths(paths map[string]interface{}) {
	paths["/cache/status"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"缓存管理"},
			"summary":     "获取缓存状态",
			"description": "获取缓存系统的运行状态",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "缓存状态",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/cache/health"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"缓存管理"},
			"summary":     "缓存健康检查",
			"description": "检查缓存系统的健康状况",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "缓存健康状态",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/cache/metrics"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"缓存管理"},
			"summary":     "获取缓存指标",
			"description": "获取缓存的性能指标",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "缓存指标",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}
}

// 添加自动化系统相关API
func addAutomationPaths(paths map[string]interface{}) {
	paths["/automation/status"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"自动化系统"},
			"summary":     "获取自动化状态",
			"description": "获取自动化系统的运行状态",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "自动化状态",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/automation/health"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"自动化系统"},
			"summary":     "自动化健康检查",
			"description": "检查自动化系统的健康状况",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "自动化健康状态",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/automation/stats"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"自动化系统"},
			"summary":     "获取执行统计",
			"description": "获取自动化任务的执行统计",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "执行统计",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}
}

// 添加编排器相关API
func addOrchestratorPaths(paths map[string]interface{}) {
	paths["/orchestrator/status"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"系统编排"},
			"summary":     "获取编排器状态",
			"description": "获取系统编排器的运行状态",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "编排器状态",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/orchestrator/services"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"系统编排"},
			"summary":     "获取服务列表",
			"description": "获取系统中所有服务的状态",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "服务列表",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/orchestrator/services/start"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"系统编排"},
			"summary":     "启动服务",
			"description": "启动指定的系统服务",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "服务启动成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/orchestrator/services/stop"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"系统编排"},
			"summary":     "停止服务",
			"description": "停止指定的系统服务",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "服务停止成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/orchestrator/services/restart"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"系统编排"},
			"summary":     "重启服务",
			"description": "重启指定的系统服务",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "服务重启成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/orchestrator/optimize"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"系统编排"},
			"summary":     "优化系统",
			"description": "执行系统优化操作",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "优化成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/orchestrator/health"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"系统编排"},
			"summary":     "编排器健康检查",
			"description": "检查编排器的健康状况",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "编排器健康状态",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}
}

// 添加热点管理相关API
func addHotlistPaths(paths map[string]interface{}) {
	paths["/hotlist/symbols"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"热点管理"},
			"summary":     "获取热点符号",
			"description": "获取当前热点交易符号列表",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "热点符号列表",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/hotlist/whitelist"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"热点管理"},
			"summary":     "获取白名单",
			"description": "获取交易白名单",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "白名单",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/hotlist/whitelist/{symbol}"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"热点管理"},
			"summary":     "添加到白名单",
			"description": "将符号添加到白名单",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "symbol",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "交易符号",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "添加成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
		"delete": map[string]interface{}{
			"tags":        []string{"热点管理"},
			"summary":     "从白名单移除",
			"description": "从白名单中移除符号",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "symbol",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "交易符号",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "移除成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/hotlist/approve"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"热点管理"},
			"summary":     "批准热点符号",
			"description": "批准热点符号进入交易",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "批准成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/hotlist/whitelist/:symbol"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"热点管理"},
			"summary":     "添加到白名单",
			"description": "将符号添加到白名单",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "symbol",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "交易符号",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "添加成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
		"delete": map[string]interface{}{
			"tags":        []string{"热点管理"},
			"summary":     "从白名单移除",
			"description": "从白名单中移除符号",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "symbol",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "交易符号",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "移除成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}
}

// 添加黑名单管理相关API
func addBlacklistPaths(paths map[string]interface{}) {
	paths["/blacklist/"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"黑名单管理"},
			"summary":     "获取黑名单列表",
			"description": "获取当前的黑名单条目",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "黑名单列表",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
		"post": map[string]interface{}{
			"tags":        []string{"黑名单管理"},
			"summary":     "添加黑名单条目",
			"description": "添加新的黑名单条目",
			"consumes":    []string{"application/json"},
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"201": map[string]interface{}{
					"description": "添加成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/blacklist/:strategy_id"] = map[string]interface{}{
		"delete": map[string]interface{}{
			"tags":        []string{"黑名单管理"},
			"summary":     "删除黑名单条目",
			"description": "删除指定的黑名单条目",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "strategy_id",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "策略ID",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "删除成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/blacklist/clear-expired"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"黑名单管理"},
			"summary":     "清理过期条目",
			"description": "清理黑名单中的过期条目",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "清理成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}
}

// 添加并发管理相关API
func addConcurrentPaths(paths map[string]interface{}) {
	paths["/concurrent/pools"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"并发管理"},
			"summary":     "获取线程池状态",
			"description": "获取并发线程池的状态信息",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "线程池状态",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/concurrent/pools/:pool_name"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"并发管理"},
			"summary":     "获取指定线程池状态",
			"description": "获取指定线程池的详细状态",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "pool_name",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "线程池名称",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "线程池状态",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/concurrent/pools/:pool_name/scale"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"并发管理"},
			"summary":     "扩缩容线程池",
			"description": "调整线程池的大小",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "pool_name",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "线程池名称",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "扩缩容成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/concurrent/monitor"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"并发管理"},
			"summary":     "获取监控统计",
			"description": "获取并发系统的监控数据",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "监控统计",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/concurrent/alerts"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"并发管理"},
			"summary":     "获取并发告警",
			"description": "获取并发系统的告警信息",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "并发告警",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/concurrent/load-balancer"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"并发管理"},
			"summary":     "获取负载均衡状态",
			"description": "获取负载均衡器的状态",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "负载均衡状态",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/concurrent/task-queue"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"并发管理"},
			"summary":     "获取任务队列状态",
			"description": "获取任务队列的状态信息",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "任务队列状态",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/concurrent/tasks"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"并发管理"},
			"summary":     "获取任务列表",
			"description": "获取当前运行的任务列表",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "任务列表",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}
}

// 添加安全管理相关API
func addSecurityPaths(paths map[string]interface{}) {
	paths["/security/keys/"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"安全管理"},
			"summary":     "获取API密钥列表",
			"description": "获取API密钥管理信息",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "API密钥列表",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
		"post": map[string]interface{}{
			"tags":        []string{"安全管理"},
			"summary":     "创建API密钥",
			"description": "创建新的API密钥",
			"consumes":    []string{"application/json"},
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"201": map[string]interface{}{
					"description": "密钥创建成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/security/audit/logs"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"安全管理"},
			"summary":     "获取安全审计日志",
			"description": "获取安全相关的审计日志",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "安全审计日志",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/security/monitoring/alerts"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"安全管理"},
			"summary":     "获取安全告警",
			"description": "获取安全监控告警信息",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "安全告警",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/security/monitoring/events"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"安全管理"},
			"summary":     "获取安全事件",
			"description": "获取安全监控事件信息",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "安全事件",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}
}

// 添加优化器相关API
func addOptimizerPaths(paths map[string]interface{}) {
	paths["/optimizer/run"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"策略优化"},
			"summary":     "运行优化任务",
			"description": "启动策略参数优化任务",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "优化任务启动成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/optimizer/tasks"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"策略优化"},
			"summary":     "获取优化任务列表",
			"description": "获取所有优化任务的状态",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "优化任务列表",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/optimizer/tasks/:id"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"策略优化"},
			"summary":     "获取优化任务详情",
			"description": "获取指定优化任务的详细信息",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "id",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "任务ID",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "优化任务详情",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/optimizer/results/:id"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"策略优化"},
			"summary":     "获取优化结果",
			"description": "获取优化任务的结果",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "id",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "任务ID",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "优化结果",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}
}

// 添加自动启动相关API
func addAutoStartPaths(paths map[string]interface{}) {
	paths["/auto-start/strategies"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"自动启动"},
			"summary":     "获取自动启动策略",
			"description": "获取自动启动策略列表",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "自动启动策略",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/auto-start/stats"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"自动启动"},
			"summary":     "获取自动启动统计",
			"description": "获取自动启动的统计信息",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "自动启动统计",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/auto-start/trigger"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"自动启动"},
			"summary":     "触发自动启动",
			"description": "手动触发自动启动流程",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "触发成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/strategy/{id}/auto-start"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"自动启动"},
			"summary":     "设置策略自动启动",
			"description": "为指定策略设置自动启动",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "id",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "策略ID",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "设置成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}
}

// 添加紧急停止相关API
func addEmergencyPaths(paths map[string]interface{}) {
	paths["/emergency/status"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"紧急停止"},
			"summary":     "获取紧急停止状态",
			"description": "获取系统紧急停止的状态",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "紧急停止状态",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/emergency/stop-all"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"紧急停止"},
			"summary":     "紧急停止所有策略",
			"description": "立即停止所有运行中的策略",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "紧急停止成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/emergency/reset"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"紧急停止"},
			"summary":     "重置紧急停止状态",
			"description": "重置系统的紧急停止状态",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "重置成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/emergency/history"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"紧急停止"},
			"summary":     "获取紧急停止历史",
			"description": "获取紧急停止的历史记录",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "紧急停止历史",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}
}

// 添加健康检查相关API
func addHealthPaths(paths map[string]interface{}) {
	paths["/health/status"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"系统稳定性"},
			"summary":     "获取健康状态",
			"description": "获取系统整体健康状态",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "健康状态",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/health/checks"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"系统稳定性"},
			"summary":     "获取健康检查列表",
			"description": "获取所有健康检查项目的状态",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "健康检查列表",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/health/checks/:name"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"系统稳定性"},
			"summary":     "获取指定健康检查",
			"description": "获取指定健康检查项目的详细状态",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "name",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "健康检查名称",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "健康检查详情",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/health/checks/:name/force"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"系统稳定性"},
			"summary":     "强制执行健康检查",
			"description": "强制执行指定的健康检查",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "name",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "健康检查名称",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "检查执行成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}
}

// 添加内存管理相关API
func addMemoryPaths(paths map[string]interface{}) {
	paths["/memory/stats"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"系统稳定性"},
			"summary":     "获取内存统计",
			"description": "获取系统内存使用统计信息",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "内存统计",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/memory/gc"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"系统稳定性"},
			"summary":     "执行垃圾回收",
			"description": "手动触发垃圾回收",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "垃圾回收完成",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}
}

// 添加网络连接相关API
func addNetworkPaths(paths map[string]interface{}) {
	paths["/network/connections"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"系统稳定性"},
			"summary":     "获取网络连接",
			"description": "获取系统网络连接状态",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "网络连接状态",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/network/connections/:id"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"系统稳定性"},
			"summary":     "获取指定连接详情",
			"description": "获取指定网络连接的详细信息",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "id",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "连接ID",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "连接详情",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/network/connections/:id/reconnect"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"系统稳定性"},
			"summary":     "重新连接",
			"description": "重新建立指定的网络连接",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "id",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "连接ID",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "重连成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}
}

// 添加关闭管理相关API
func addShutdownPaths(paths map[string]interface{}) {
	paths["/shutdown/status"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"系统稳定性"},
			"summary":     "获取关闭状态",
			"description": "获取系统关闭状态",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "关闭状态",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/shutdown/graceful"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"系统稳定性"},
			"summary":     "优雅关闭",
			"description": "执行系统优雅关闭",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "关闭启动成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/shutdown/force"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"系统稳定性"},
			"summary":     "强制关闭",
			"description": "执行系统强制关闭",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "强制关闭启动成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}
}

// 添加策略验证相关API
func addValidationPaths(paths map[string]interface{}) {
	paths["/validation/strategies"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"策略验证"},
			"summary":     "获取策略验证状态",
			"description": "获取策略验证的状态信息",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "策略验证状态",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/validation/problems"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"策略验证"},
			"summary":     "获取策略问题",
			"description": "获取策略验证发现的问题",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "策略问题列表",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/validation/automation"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"策略验证"},
			"summary":     "获取自动化状态",
			"description": "获取策略验证自动化的状态",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "自动化状态",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}
}

// 添加工作流相关API
func addWorkflowPaths(paths map[string]interface{}) {
	paths["/workflow/status"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"工作流"},
			"summary":     "获取工作流状态",
			"description": "获取工作流系统的运行状态",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "工作流状态",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/workflow/dependency-graph"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"工作流"},
			"summary":     "获取依赖图",
			"description": "获取工作流的依赖关系图",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "依赖图",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/workflow/results"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"工作流"},
			"summary":     "获取执行结果",
			"description": "获取工作流的执行结果",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "执行结果",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/workflow/validate"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"工作流"},
			"summary":     "工作流验证",
			"description": "验证工作流配置的正确性",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "验证结果",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/workflow/enabled"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"工作流"},
			"summary":     "获取启用的功能",
			"description": "获取工作流中启用的功能列表",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "启用的功能",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/workflow/execute"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"工作流"},
			"summary":     "执行工作流",
			"description": "执行指定的工作流",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "执行成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/workflow/functions/:function_id"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"工作流"},
			"summary":     "获取工作流函数",
			"description": "获取指定工作流函数的信息",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "function_id",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "函数ID",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "函数信息",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/workflow/functions/:function_id/enable"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"工作流"},
			"summary":     "启用工作流函数",
			"description": "启用指定的工作流函数",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "function_id",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "函数ID",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "启用成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/workflow/functions/:function_id/disable"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"工作流"},
			"summary":     "禁用工作流函数",
			"description": "禁用指定的工作流函数",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"parameters": []map[string]interface{}{
				{
					"name":        "function_id",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "函数ID",
				},
			},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "禁用成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}
}

// 添加共享结果相关API
func addSharedResultsPaths(paths map[string]interface{}) {
	paths["/shared-results"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"结果分享"},
			"summary":     "获取分享结果",
			"description": "获取共享的分析结果",
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "分享结果",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/share-result"] = map[string]interface{}{
		"post": map[string]interface{}{
			"tags":        []string{"结果分享"},
			"summary":     "分享结果",
			"description": "分享分析结果",
			"consumes":    []string{"application/json"},
			"produces":    []string{"application/json"},
			"security":    []map[string]interface{}{{"BearerAuth": []string{}}},
			"responses": map[string]interface{}{
				"201": map[string]interface{}{
					"description": "分享成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}
}

// 添加系统设置相关API
func addSettingsPaths(paths map[string]interface{}) {
	paths["/settings"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"系统设置"},
			"summary":     "获取系统设置",
			"description": "获取当前系统配置设置",
			"produces":    []string{"application/json"},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "系统设置",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
		"put": map[string]interface{}{
			"tags":        []string{"系统设置"},
			"summary":     "更新系统设置",
			"description": "更新系统配置设置",
			"consumes":    []string{"application/json"},
			"produces":    []string{"application/json"},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "更新成功",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}
}

// 添加WebSocket相关API
func addWebSocketPaths(paths map[string]interface{}) {
	paths["/ws/market/:symbol"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"WebSocket"},
			"summary":     "市场数据流",
			"description": "订阅特定符号的实时市场数据",
			"produces":    []string{"application/json"},
			"parameters": []map[string]interface{}{
				{
					"name":        "symbol",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "交易符号",
				},
			},
			"responses": map[string]interface{}{
				"101": map[string]interface{}{
					"description": "WebSocket连接建立",
				},
			},
		},
	}

	paths["/ws/strategy/:id"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"WebSocket"},
			"summary":     "策略数据流",
			"description": "订阅特定策略的实时数据",
			"produces":    []string{"application/json"},
			"parameters": []map[string]interface{}{
				{
					"name":        "id",
					"in":          "path",
					"required":    true,
					"type":        "string",
					"description": "策略ID",
				},
			},
			"responses": map[string]interface{}{
				"101": map[string]interface{}{
					"description": "WebSocket连接建立",
				},
			},
		},
	}

	paths["/ws/alerts"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"WebSocket"},
			"summary":     "告警数据流",
			"description": "订阅系统告警信息",
			"produces":    []string{"application/json"},
			"responses": map[string]interface{}{
				"101": map[string]interface{}{
					"description": "WebSocket连接建立",
				},
			},
		},
	}
}

// 添加基础健康检查API
func addBasicHealthPaths(paths map[string]interface{}) {
	paths["/health"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"基础API"},
			"summary":     "基础健康检查",
			"description": "检查系统基础健康状态",
			"produces":    []string{"application/json"},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "健康状态",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}

	paths["/health/detailed"] = map[string]interface{}{
		"get": map[string]interface{}{
			"tags":        []string{"基础API"},
			"summary":     "详细健康检查",
			"description": "检查系统详细健康状态",
			"produces":    []string{"application/json"},
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "详细健康状态",
					"schema":      map[string]interface{}{"$ref": "#/definitions/Response"},
				},
			},
		},
	}
}
