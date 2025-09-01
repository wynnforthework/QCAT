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
	addCachePaths(doc.Paths)
	addAutomationPaths(doc.Paths)
	addOrchestratorPaths(doc.Paths)
	addHotlistPaths(doc.Paths)
	addBlacklistPaths(doc.Paths)
	addConcurrentPaths(doc.Paths)
	addSecurityPaths(doc.Paths)

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
}
