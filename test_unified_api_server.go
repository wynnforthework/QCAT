package main

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// 统一策略数据模型
type UnifiedStrategy struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Version     string                 `json:"version"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
	Lifecycle   map[string]interface{} `json:"lifecycle"`
	Execution   map[string]interface{} `json:"execution"`
	Performance map[string]interface{} `json:"performance"`
	Pool        map[string]interface{} `json:"pool"`
	Config      map[string]interface{} `json:"config"`
}

// API响应格式
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func main() {
	log.Println("🚀 Starting QCAT Unified Strategy API Test Server...")

	r := gin.Default()

	// 启用CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "qcat-unified-api-test",
			"version": "1.0.0",
		})
	})

	// API路由组
	v1 := r.Group("/api/v1")
	{
		// 统一策略API
		strategy := v1.Group("/strategy")
		{
			strategy.GET("/", listStrategies)
			strategy.GET("/:id", getStrategy)
			strategy.GET("/pool/overview", getPoolOverview)
			strategy.GET("/execution/overview", getExecutionOverview)
			strategy.GET("/execution/realtime", getRealtimeStatus)
			strategy.GET("/workflow/status", getWorkflowStatus)
		}
	}

	// 启动服务器
	port := ":8082"
	log.Printf("✅ Test server starting on port %s", port)
	log.Printf("📡 Unified Strategy API endpoints:")
	log.Printf("   GET http://localhost%s/api/v1/strategy", port)
	log.Printf("   GET http://localhost%s/api/v1/strategy/:id", port)
	log.Printf("   GET http://localhost%s/api/v1/strategy/pool/overview", port)
	log.Printf("   GET http://localhost%s/api/v1/strategy/execution/overview", port)
	log.Printf("   GET http://localhost%s/api/v1/strategy/execution/realtime", port)
	log.Printf("   GET http://localhost%s/api/v1/strategy/workflow/status", port)
	log.Printf("🎉 Ready to test unified strategy management!")

	if err := r.Run(port); err != nil {
		log.Fatalf("❌ Failed to start server: %v", err)
	}
}

// 获取策略列表
func listStrategies(c *gin.Context) {
	view := c.DefaultQuery("view", "list")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	strategies := getMockStrategies()

	// 根据视图调整数据
	for i := range strategies {
		switch view {
		case "pool":
			// 突出池相关信息
			strategies[i].Pool["highlighted"] = true
		case "execution":
			// 突出执行相关信息
			strategies[i].Execution["highlighted"] = true
		case "performance":
			// 突出性能相关信息
			strategies[i].Performance["highlighted"] = true
		}
	}

	// 简单分页
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > len(strategies) {
		end = len(strategies)
	}
	if start > len(strategies) {
		start = len(strategies)
	}

	pagedStrategies := strategies[start:end]

	response := map[string]interface{}{
		"strategies": pagedStrategies,
		"total":      len(strategies),
		"page":       page,
		"pageSize":   pageSize,
		"summary": map[string]interface{}{
			"total": map[string]interface{}{
				"count":      len(strategies),
				"active":     2,
				"inactive":   1,
				"testing":    1,
				"production": 2,
			},
			"pool": map[string]interface{}{
				"enabled":  2,
				"disabled": 1,
				"pending":  0,
				"testing":  1,
			},
			"performance": map[string]interface{}{
				"avgReturn":     0.123,
				"avgSharpe":     2.11,
				"totalPnl":      24170.75,
				"winningCount":  2,
			},
		},
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    response,
	})
}

// 获取单个策略
func getStrategy(c *gin.Context) {
	strategyID := c.Param("id")
	strategies := getMockStrategies()

	for _, strategy := range strategies {
		if strategy.ID == strategyID {
			c.JSON(http.StatusOK, Response{
				Success: true,
				Data:    strategy,
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, Response{
		Success: false,
		Error:   "strategy not found",
	})
}

// 获取策略池概览
func getPoolOverview(c *gin.Context) {
	overview := map[string]interface{}{
		"distribution": map[string]interface{}{
			"enabled":  2,
			"disabled": 1,
			"pending":  0,
			"testing":  1,
		},
		"summary": map[string]interface{}{
			"total": map[string]interface{}{
				"count":  4,
				"active": 2,
			},
		},
		"recentActivity": []map[string]interface{}{
			{
				"type":       "strategy_enabled",
				"strategyId": "strategy_001",
				"message":    "策略 动量策略Alpha 已启用",
				"timestamp":  "2分钟前",
			},
		},
		"resourceUsage": map[string]interface{}{
			"cpu": map[string]interface{}{
				"used":  6.3,
				"total": 16.0,
				"usage": 39.4,
			},
			"memory": map[string]interface{}{
				"used":  12.4,
				"total": 32.0,
				"usage": 38.8,
			},
		},
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    overview,
	})
}

// 获取执行概览
func getExecutionOverview(c *gin.Context) {
	overview := map[string]interface{}{
		"system": map[string]interface{}{
			"status":           "running",
			"activeStrategies": 2,
			"totalStrategies":  4,
			"uptime":          "15天 8小时 32分钟",
		},
		"performance": map[string]interface{}{
			"latency":         8.5,
			"throughput":      1250,
			"successRate":     95.6,
			"errorRate":       4.4,
			"totalExecutions": 2221,
		},
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    overview,
	})
}

// 获取实时状态
func getRealtimeStatus(c *gin.Context) {
	status := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"activeStrategies": []map[string]interface{}{
			{
				"id":            "strategy_001",
				"name":          "动量策略Alpha",
				"status":        "running",
				"lastExecution": time.Now().Add(-5 * time.Minute).Format("15:04:05"),
				"successRate":   95.6,
				"latency":       8.5,
				"pnl":           15420.50,
			},
			{
				"id":            "strategy_002",
				"name":          "均值回归策略Beta",
				"status":        "running",
				"lastExecution": time.Now().Add(-3 * time.Minute).Format("15:04:05"),
				"successRate":   94.2,
				"latency":       12.3,
				"pnl":           8750.25,
			},
		},
		"systemMetrics": map[string]interface{}{
			"cpu":    6.3,
			"memory": 12.4,
		},
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    status,
	})
}

// 获取工作流状态
func getWorkflowStatus(c *gin.Context) {
	status := map[string]interface{}{
		"system": map[string]interface{}{
			"status":              "running",
			"activeWorkflows":     10,
			"concurrentStrategies": 15,
			"evolutionGeneration": 47,
			"uptime":             "15天 8小时 32分钟",
		},
		"evolution": map[string]interface{}{
			"currentGeneration": 47,
			"populationSize":    20,
			"bestFitness":       0.892,
			"averageFitness":    0.634,
		},
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    status,
	})
}

// 获取模拟策略数据
func getMockStrategies() []UnifiedStrategy {
	return []UnifiedStrategy{
		{
			ID:          "strategy_001",
			Name:        "动量策略Alpha",
			Type:        "momentum",
			Description: "基于动量指标的交易策略",
			Version:     "1.2.0",
			CreatedAt:   time.Now().Add(-30 * 24 * time.Hour),
			UpdatedAt:   time.Now().Add(-2 * time.Hour),
			Lifecycle: map[string]interface{}{
				"stage":     "production",
				"status":    "active",
				"isEnabled": true,
				"canStart":  false,
				"canStop":   true,
			},
			Execution: map[string]interface{}{
				"isRunning":      true,
				"lastExecution":  time.Now().Add(-5 * time.Minute).Format(time.RFC3339),
				"executionCount": 1234,
				"successRate":    95.6,
				"avgLatency":     8.5,
			},
			Performance: map[string]interface{}{
				"pnl":          15420.50,
				"totalReturn":  0.156,
				"sharpeRatio":  2.34,
				"maxDrawdown":  0.08,
				"winRate":      0.67,
				"tradeCount":   456,
			},
			Pool: map[string]interface{}{
				"poolStatus": "enabled",
				"priority":   "high",
				"resourceAllocation": map[string]interface{}{
					"cpu":    1.2,
					"memory": 2.1,
				},
				"lastSync":   time.Now().Add(-2 * time.Minute).Format(time.RFC3339),
				"syncStatus": "success",
			},
			Config: map[string]interface{}{
				"lookback":  14,
				"threshold": 0.02,
			},
		},
		{
			ID:          "strategy_002",
			Name:        "均值回归策略Beta",
			Type:        "mean_reversion",
			Description: "基于均值回归的交易策略",
			Version:     "2.1.0",
			CreatedAt:   time.Now().Add(-45 * 24 * time.Hour),
			UpdatedAt:   time.Now().Add(-1 * time.Hour),
			Lifecycle: map[string]interface{}{
				"stage":     "production",
				"status":    "active",
				"isEnabled": true,
				"canStart":  false,
				"canStop":   true,
			},
			Execution: map[string]interface{}{
				"isRunning":      true,
				"lastExecution":  time.Now().Add(-3 * time.Minute).Format(time.RFC3339),
				"executionCount": 987,
				"successRate":    94.2,
				"avgLatency":     12.3,
			},
			Performance: map[string]interface{}{
				"pnl":          8750.25,
				"totalReturn":  0.089,
				"sharpeRatio":  1.89,
				"maxDrawdown":  0.12,
				"winRate":      0.58,
				"tradeCount":   324,
			},
			Pool: map[string]interface{}{
				"poolStatus": "enabled",
				"priority":   "medium",
				"resourceAllocation": map[string]interface{}{
					"cpu":    0.8,
					"memory": 1.5,
				},
				"lastSync":   time.Now().Add(-1 * time.Minute).Format(time.RFC3339),
				"syncStatus": "success",
			},
			Config: map[string]interface{}{
				"window":    20,
				"deviation": 2.0,
			},
		},
		{
			ID:          "strategy_003",
			Name:        "套利策略Gamma",
			Type:        "arbitrage",
			Description: "跨交易所套利策略",
			Version:     "1.0.0",
			CreatedAt:   time.Now().Add(-7 * 24 * time.Hour),
			UpdatedAt:   time.Now().Add(-30 * time.Minute),
			Lifecycle: map[string]interface{}{
				"stage":     "testing",
				"status":    "inactive",
				"isEnabled": false,
				"canStart":  true,
				"canStop":   false,
			},
			Execution: map[string]interface{}{
				"isRunning":      false,
				"lastExecution":  time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
				"executionCount": 123,
				"successRate":    87.8,
				"avgLatency":     15.2,
			},
			Performance: map[string]interface{}{
				"pnl":          -1250.75,
				"totalReturn":  -0.025,
				"sharpeRatio":  0.45,
				"maxDrawdown":  0.15,
				"winRate":      0.42,
				"tradeCount":   89,
			},
			Pool: map[string]interface{}{
				"poolStatus": "testing",
				"priority":   "low",
				"resourceAllocation": map[string]interface{}{
					"cpu":    0.3,
					"memory": 0.8,
				},
				"lastSync":   time.Now().Add(-10 * time.Minute).Format(time.RFC3339),
				"syncStatus": "pending",
			},
			Config: map[string]interface{}{
				"min_spread":   0.001,
				"max_position": 10000,
			},
		},
	}
}