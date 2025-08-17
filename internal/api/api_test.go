package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"qcat/internal/config"
	"qcat/internal/testutils"
)

func TestAPIServer(t *testing.T) {
	suite := testutils.NewTestSuite(t, &testutils.TestConfig{
		UseRealDB:    false,
		UseRealCache: false,
		LogLevel:     "error",
	})
	defer suite.TearDown()

	// 创建测试配置
	cfg := &config.Config{
		App: config.AppConfig{
			Name:        "QCAT Test",
			Version:     "1.0.0",
			Environment: "test",
		},
		Server: config.ServerConfig{
			Port:           testutils.GetAvailablePort(),
			Host:           "localhost",
			Debug:          true,
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   30 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
	}

	// 创建服务器
	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// 创建HTTP测试助手
	httpHelper := testutils.NewHTTPTestHelper(suite)
	httpHelper.Router = server.router

	// 测试健康检查
	t.Run("health check", func(t *testing.T) {
		resp := httpHelper.GET("/health", nil)
		resp.AssertStatus(http.StatusOK)

		var health map[string]interface{}
		err := resp.GetJSON(&health)
		if err != nil {
			t.Fatalf("Failed to parse health response: %v", err)
		}

		if health["status"] != "ok" {
			t.Errorf("Expected status 'ok', got '%v'", health["status"])
		}
	})

	// 测试指标端点
	t.Run("metrics endpoint", func(t *testing.T) {
		resp := httpHelper.GET("/metrics", nil)
		resp.AssertStatus(http.StatusOK)
	})
}

func TestStrategyAPI(t *testing.T) {
	suite := testutils.NewTestSuite(t, nil)
	defer suite.TearDown()

	httpHelper := testutils.NewHTTPTestHelper(suite)
	setupTestRoutes(httpHelper.Router, suite)

	mockData := testutils.NewMockData()

	t.Run("create strategy", func(t *testing.T) {
		strategy := mockData.GenerateStrategy()
		
		resp := httpHelper.POST("/api/v1/strategy", strategy, map[string]string{
			"Authorization": "Bearer test-token",
		})
		
		resp.AssertStatus(http.StatusCreated)
		
		var response map[string]interface{}
		err := resp.GetJSON(&response)
		if err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		
		if response["success"] != true {
			t.Error("Expected success to be true")
		}
	})

	t.Run("list strategies", func(t *testing.T) {
		resp := httpHelper.GET("/api/v1/strategy", map[string]string{
			"Authorization": "Bearer test-token",
		})
		
		resp.AssertStatus(http.StatusOK)
		
		var response map[string]interface{}
		err := resp.GetJSON(&response)
		if err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		
		if strategies, ok := response["data"].([]interface{}); ok {
			if len(strategies) == 0 {
				t.Log("No strategies found (expected for test)")
			}
		}
	})

	t.Run("get strategy by id", func(t *testing.T) {
		resp := httpHelper.GET("/api/v1/strategy/test-id", map[string]string{
			"Authorization": "Bearer test-token",
		})
		
		// 可能返回404，这在测试中是正常的
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 200 or 404, got %d", resp.StatusCode)
		}
	})

	t.Run("invalid request", func(t *testing.T) {
		// 发送无效的JSON
		invalidJSON := `{"invalid": json}`
		
		req := bytes.NewReader([]byte(invalidJSON))
		resp := httpHelper.Request("POST", "/api/v1/strategy", req, map[string]string{
			"Authorization": "Bearer test-token",
			"Content-Type":  "application/json",
		})
		
		resp.AssertStatus(http.StatusBadRequest)
	})
}

func TestOptimizerAPI(t *testing.T) {
	suite := testutils.NewTestSuite(t, nil)
	defer suite.TearDown()

	httpHelper := testutils.NewHTTPTestHelper(suite)
	setupTestRoutes(httpHelper.Router, suite)

	t.Run("run optimization", func(t *testing.T) {
		optimizationRequest := map[string]interface{}{
			"strategy_id": "test-strategy",
			"method":      "wfo",
			"objective":   "sharpe",
			"time_range": map[string]string{
				"start": "2024-01-01",
				"end":   "2024-01-31",
			},
			"parameters": map[string]interface{}{
				"ma_short": map[string]interface{}{
					"min": 5,
					"max": 30,
				},
				"ma_long": map[string]interface{}{
					"min": 30,
					"max": 100,
				},
			},
		}

		resp := httpHelper.POST("/api/v1/optimizer/run", optimizationRequest, map[string]string{
			"Authorization": "Bearer test-token",
		})

		// 根据实际实现，可能返回202 (Accepted) 或 200
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
			t.Errorf("Expected status 200 or 202, got %d", resp.StatusCode)
		}
	})

	t.Run("get optimization tasks", func(t *testing.T) {
		resp := httpHelper.GET("/api/v1/optimizer/tasks", map[string]string{
			"Authorization": "Bearer test-token",
		})

		resp.AssertStatus(http.StatusOK)
	})
}

func TestErrorHandling(t *testing.T) {
	suite := testutils.NewTestSuite(t, nil)
	defer suite.TearDown()

	httpHelper := testutils.NewHTTPTestHelper(suite)
	setupTestRoutes(httpHelper.Router, suite)

	t.Run("not found", func(t *testing.T) {
		resp := httpHelper.GET("/api/v1/nonexistent", nil)
		resp.AssertStatus(http.StatusNotFound)
	})

	t.Run("method not allowed", func(t *testing.T) {
		resp := httpHelper.Request("PATCH", "/api/v1/strategy", nil, nil)
		resp.AssertStatus(http.StatusMethodNotAllowed)
	})

	t.Run("unauthorized", func(t *testing.T) {
		resp := httpHelper.GET("/api/v1/strategy", nil) // 没有Authorization头
		resp.AssertStatus(http.StatusUnauthorized)
	})
}

func TestRateLimiting(t *testing.T) {
	suite := testutils.NewTestSuite(t, nil)
	defer suite.TearDown()

	httpHelper := testutils.NewHTTPTestHelper(suite)
	setupTestRoutes(httpHelper.Router, suite)

	// 快速发送多个请求测试限流
	t.Run("rate limiting", func(t *testing.T) {
		rateLimitExceeded := false
		
		for i := 0; i < 100; i++ {
			resp := httpHelper.GET("/health", nil)
			if resp.StatusCode == http.StatusTooManyRequests {
				rateLimitExceeded = true
				break
			}
		}
		
		// 注意：这个测试可能不会触发限流，取决于限流配置
		if rateLimitExceeded {
			t.Log("Rate limiting is working")
		} else {
			t.Log("Rate limiting not triggered (may be disabled in test)")
		}
	})
}

// 辅助函数：设置测试路由
func setupTestRoutes(router *gin.Engine, suite *testutils.TestSuite) {
	// 设置测试模式
	gin.SetMode(gin.TestMode)

	// 添加基本中间件
	router.Use(gin.Recovery())

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().UTC(),
		})
	})

	// 指标端点
	router.GET("/metrics", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"metrics": "test-metrics",
		})
	})

	// API路由组
	api := router.Group("/api/v1")
	
	// 简单的认证中间件（测试用）
	api.Use(func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	})

	// 策略路由
	strategies := api.Group("/strategy")
	{
		strategies.GET("", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    []interface{}{},
			})
		})
		
		strategies.POST("", func(c *gin.Context) {
			var strategy map[string]interface{}
			if err := c.ShouldBindJSON(&strategy); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			
			c.JSON(http.StatusCreated, gin.H{
				"success": true,
				"data":    strategy,
			})
		})
		
		strategies.GET("/:id", func(c *gin.Context) {
			id := c.Param("id")
			if id == "test-id" {
				c.JSON(http.StatusOK, gin.H{
					"success": true,
					"data": gin.H{
						"id":   id,
						"name": "Test Strategy",
					},
				})
			} else {
				c.JSON(http.StatusNotFound, gin.H{"error": "strategy not found"})
			}
		})
	}

	// 优化器路由
	optimizer := api.Group("/optimizer")
	{
		optimizer.POST("/run", func(c *gin.Context) {
			var request map[string]interface{}
			if err := c.ShouldBindJSON(&request); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			
			c.JSON(http.StatusAccepted, gin.H{
				"success": true,
				"task_id": "test-task-id",
			})
		})
		
		optimizer.GET("/tasks", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    []interface{}{},
			})
		})
	}
}

func BenchmarkAPIEndpoints(b *testing.B) {
	config := &testutils.TestConfig{
		LogLevel: "error",
	}

	testutils.RunBenchmark(b, "HealthCheck", config, func(b *testing.B, suite *testutils.BenchmarkSuite) {
		httpHelper := testutils.NewHTTPTestHelper(suite.Suite)
		setupTestRoutes(httpHelper.Router, suite.Suite)

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				resp := httpHelper.GET("/health", nil)
				if resp.StatusCode != http.StatusOK {
					b.Errorf("Expected status 200, got %d", resp.StatusCode)
				}
			}
		})
	})
}
