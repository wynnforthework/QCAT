package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	testing "qcat/internal/testing"
)

// StressTestHandler 压力测试处理器
type StressTestHandler struct {
	db                *sql.DB
	activeTests       map[string]*testing.StressTestFramework
	testResults       map[string]*testing.StressTestResult
	performanceMonitor *testing.PerformanceMonitor
}

// NewStressTestHandler 创建压力测试处理器
func NewStressTestHandler(db *sql.DB) *StressTestHandler {
	monitor := testing.NewPerformanceMonitor(5 * time.Second)
	monitor.Start()
	
	return &StressTestHandler{
		db:                 db,
		activeTests:        make(map[string]*testing.StressTestFramework),
		testResults:        make(map[string]*testing.StressTestResult),
		performanceMonitor: monitor,
	}
}

// StartStressTest 启动压力测试
func (h *StressTestHandler) StartStressTest(c *gin.Context) {
	var req struct {
		TestID              string  `json:"test_id" binding:"required"`
		Duration            int     `json:"duration"`             // 秒
		AccelerationFactor  int     `json:"acceleration_factor"`  // 时间加速倍数
		ConcurrentUsers     int     `json:"concurrent_users"`     // 并发用户数
		RequestsPerSecond   int     `json:"requests_per_second"`  // 每秒请求数
		DataGenerationRate  int     `json:"data_generation_rate"` // 数据生成频率
		WorkflowExecutions  int     `json:"workflow_executions"`  // 工作流执行次数
		EnableMonitoring    bool    `json:"enable_monitoring"`    // 启用监控
		EnableDataPersist   bool    `json:"enable_data_persist"`  // 启用数据持久化
		SimulateFailures    bool    `json:"simulate_failures"`    // 模拟失败
		FailureRate         float64 `json:"failure_rate"`         // 失败率
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid request: " + err.Error(),
		})
		return
	}
	
	// 检查测试是否已存在
	if _, exists := h.activeTests[req.TestID]; exists {
		c.JSON(http.StatusConflict, Response{
			Success: false,
			Error:   "Test with ID " + req.TestID + " is already running",
		})
		return
	}
	
	// 设置默认值
	if req.Duration == 0 {
		req.Duration = 300 // 5分钟
	}
	if req.AccelerationFactor == 0 {
		req.AccelerationFactor = 100 // 100倍加速
	}
	if req.ConcurrentUsers == 0 {
		req.ConcurrentUsers = 10
	}
	if req.RequestsPerSecond == 0 {
		req.RequestsPerSecond = 100
	}
	if req.DataGenerationRate == 0 {
		req.DataGenerationRate = 10
	}
	if req.WorkflowExecutions == 0 {
		req.WorkflowExecutions = 5
	}
	
	// 创建测试配置
	config := &testing.StressTestConfig{
		Duration:            time.Duration(req.Duration) * time.Second,
		AccelerationFactor:  req.AccelerationFactor,
		ConcurrentUsers:     req.ConcurrentUsers,
		RequestsPerSecond:   req.RequestsPerSecond,
		DataGenerationRate:  req.DataGenerationRate,
		WorkflowExecutions:  req.WorkflowExecutions,
		EnableMonitoring:    req.EnableMonitoring,
		EnableDataPersist:   req.EnableDataPersist,
		SimulateFailures:    req.SimulateFailures,
		FailureRate:         req.FailureRate,
	}
	
	// 创建压力测试框架
	framework := testing.NewStressTestFramework(config)
	h.activeTests[req.TestID] = framework
	
	// 异步运行测试
	go func() {
		result, err := framework.Run()
		if err != nil {
			// 记录错误
			// TODO: 可以添加错误日志记录
		}
		
		// 保存结果
		if result != nil {
			h.testResults[req.TestID] = result
		}
		
		// 清理活跃测试
		delete(h.activeTests, req.TestID)
	}()
	
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"test_id": req.TestID,
			"message": "Stress test started successfully",
			"config":  config,
		},
	})
}

// StopStressTest 停止压力测试
func (h *StressTestHandler) StopStressTest(c *gin.Context) {
	testID := c.Param("test_id")
	
	framework, exists := h.activeTests[testID]
	if !exists {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Test not found or not running: " + testID,
		})
		return
	}
	
	// 停止测试
	framework.Stop()
	
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"test_id": testID,
			"message": "Stress test stopped successfully",
		},
	})
}

// GetStressTestStatus 获取压力测试状态
func (h *StressTestHandler) GetStressTestStatus(c *gin.Context) {
	testID := c.Param("test_id")
	
	// 检查是否是活跃测试
	if framework, exists := h.activeTests[testID]; exists {
		c.JSON(http.StatusOK, Response{
			Success: true,
			Data: map[string]interface{}{
				"test_id": testID,
				"status":  "running",
				"config":  framework,
			},
		})
		return
	}
	
	// 检查是否有测试结果
	if result, exists := h.testResults[testID]; exists {
		c.JSON(http.StatusOK, Response{
			Success: true,
			Data: map[string]interface{}{
				"test_id": testID,
				"status":  "completed",
				"result":  result,
			},
		})
		return
	}
	
	c.JSON(http.StatusNotFound, Response{
		Success: false,
		Error:   "Test not found: " + testID,
	})
}

// ListStressTests 列出所有压力测试
func (h *StressTestHandler) ListStressTests(c *gin.Context) {
	activeTests := make([]map[string]interface{}, 0)
	for testID := range h.activeTests {
		activeTests = append(activeTests, map[string]interface{}{
			"test_id": testID,
			"status":  "running",
		})
	}
	
	completedTests := make([]map[string]interface{}, 0)
	for testID, result := range h.testResults {
		completedTests = append(completedTests, map[string]interface{}{
			"test_id":    testID,
			"status":     "completed",
			"start_time": result.StartTime,
			"end_time":   result.EndTime,
			"duration":   result.Duration.String(),
		})
	}
	
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"active_tests":    activeTests,
			"completed_tests": completedTests,
			"total_active":    len(activeTests),
			"total_completed": len(completedTests),
		},
	})
}

// GetStressTestResult 获取压力测试结果
func (h *StressTestHandler) GetStressTestResult(c *gin.Context) {
	testID := c.Param("test_id")
	
	result, exists := h.testResults[testID]
	if !exists {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Test result not found: " + testID,
		})
		return
	}
	
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    result,
	})
}

// GetPerformanceMetrics 获取性能指标
func (h *StressTestHandler) GetPerformanceMetrics(c *gin.Context) {
	metricName := c.Query("metric")
	
	if metricName != "" {
		// 获取特定指标
		metric := h.performanceMonitor.GetMetric(metricName)
		if metric == nil {
			c.JSON(http.StatusNotFound, Response{
				Success: false,
				Error:   "Metric not found: " + metricName,
			})
			return
		}
		
		c.JSON(http.StatusOK, Response{
			Success: true,
			Data:    metric,
		})
		return
	}
	
	// 获取所有指标
	metrics := h.performanceMonitor.GetAllMetrics()
	
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"metrics":      metrics,
			"total_count":  len(metrics),
			"collected_at": time.Now(),
		},
	})
}

// CreatePresetTest 创建预设测试
func (h *StressTestHandler) CreatePresetTest(c *gin.Context) {
	presetType := c.Param("preset_type")
	testID := c.Query("test_id")
	
	if testID == "" {
		testID = "preset_" + presetType + "_" + strconv.FormatInt(time.Now().Unix(), 10)
	}
	
	var config *testing.StressTestConfig
	
	switch presetType {
	case "light":
		config = &testing.StressTestConfig{
			Duration:           5 * time.Minute,
			AccelerationFactor: 50,
			ConcurrentUsers:    5,
			RequestsPerSecond:  50,
			DataGenerationRate: 5,
			WorkflowExecutions: 3,
			EnableMonitoring:   true,
			SimulateFailures:   false,
		}
	case "medium":
		config = &testing.StressTestConfig{
			Duration:           10 * time.Minute,
			AccelerationFactor: 100,
			ConcurrentUsers:    10,
			RequestsPerSecond:  100,
			DataGenerationRate: 10,
			WorkflowExecutions: 5,
			EnableMonitoring:   true,
			SimulateFailures:   true,
			FailureRate:        0.05,
		}
	case "heavy":
		config = &testing.StressTestConfig{
			Duration:           15 * time.Minute,
			AccelerationFactor: 200,
			ConcurrentUsers:    20,
			RequestsPerSecond:  200,
			DataGenerationRate: 20,
			WorkflowExecutions: 10,
			EnableMonitoring:   true,
			SimulateFailures:   true,
			FailureRate:        0.1,
		}
	case "extreme":
		config = &testing.StressTestConfig{
			Duration:           30 * time.Minute,
			AccelerationFactor: 500,
			ConcurrentUsers:    50,
			RequestsPerSecond:  500,
			DataGenerationRate: 50,
			WorkflowExecutions: 20,
			EnableMonitoring:   true,
			SimulateFailures:   true,
			FailureRate:        0.15,
		}
	default:
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Unknown preset type: " + presetType + ". Available: light, medium, heavy, extreme",
		})
		return
	}
	
	// 检查测试是否已存在
	if _, exists := h.activeTests[testID]; exists {
		c.JSON(http.StatusConflict, Response{
			Success: false,
			Error:   "Test with ID " + testID + " is already running",
		})
		return
	}
	
	// 创建压力测试框架
	framework := testing.NewStressTestFramework(config)
	h.activeTests[testID] = framework
	
	// 异步运行测试
	go func() {
		result, err := framework.Run()
		if err != nil {
			// 记录错误
		}
		
		// 保存结果
		if result != nil {
			h.testResults[testID] = result
		}
		
		// 清理活跃测试
		delete(h.activeTests, testID)
	}()
	
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"test_id":     testID,
			"preset_type": presetType,
			"message":     "Preset stress test started successfully",
			"config":      config,
		},
	})
}
