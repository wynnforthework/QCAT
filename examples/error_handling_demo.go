package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"qcat/internal/errors"
	"qcat/internal/logger"
	"qcat/internal/validation"
)

func main() {
	// 初始化日志系统
	logConfig := logger.Config{
		Level:     logger.LevelDebug,
		Format:    logger.FormatText,
		Output:    "stdout",
		Caller:    true,
		Timestamp: true,
	}
	
	logger.Init(logConfig)
	
	fmt.Println("=== 错误处理和日志系统演示 ===\n")
	
	// 演示1: 基本错误处理
	demonstrateBasicErrorHandling()
	
	// 演示2: 错误包装和上下文
	demonstrateErrorWrappingAndContext()
	
	// 演示3: 参数验证
	demonstrateValidation()
	
	// 演示4: 结构化日志
	demonstrateStructuredLogging()
	
	// 演示5: 性能日志
	demonstratePerformanceLogging()
}

func demonstrateBasicErrorHandling() {
	fmt.Println("1. 基本错误处理演示:")
	
	// 创建不同类型的错误
	errors := []*errors.AppError{
		errors.NewAppError(errors.ErrCodeInvalidInput, "用户输入无效", nil),
		errors.NewAppError(errors.ErrCodeDBConnection, "数据库连接失败", nil),
		errors.NewAppError(errors.ErrCodeRateLimit, "请求频率过高", nil),
		errors.NewAppError(errors.ErrCodeStrategyExecution, "策略执行失败", nil),
	}
	
	for _, err := range errors {
		fmt.Printf("  错误代码: %s\n", err.Code)
		fmt.Printf("  错误消息: %s\n", err.Message)
		fmt.Printf("  严重程度: %s\n", err.Severity)
		fmt.Printf("  HTTP状态码: %d\n", err.HTTPStatus())
		fmt.Printf("  可重试: %t\n", err.IsRetryable())
		fmt.Println()
	}
}

func demonstrateErrorWrappingAndContext() {
	fmt.Println("2. 错误包装和上下文演示:")
	
	// 模拟一个原始错误
	originalErr := fmt.Errorf("connection timeout")
	
	// 包装错误并添加上下文
	appErr := errors.WrapError(originalErr, errors.ErrCodeDBConnection, "数据库操作失败")
	appErr = appErr.WithRequestID("req_12345")
	appErr = appErr.WithUserID("user_67890")
	appErr = appErr.WithContext("operation", "query_strategies")
	appErr = appErr.WithContext("table", "strategies")
	
	fmt.Printf("  包装后的错误: %s\n", appErr.Error())
	fmt.Printf("  请求ID: %s\n", appErr.RequestID)
	fmt.Printf("  用户ID: %s\n", appErr.UserID)
	fmt.Printf("  上下文: %+v\n", appErr.Context)
	fmt.Printf("  原始错误: %s\n", appErr.Unwrap().Error())
	fmt.Println()
}

func demonstrateValidation() {
	fmt.Println("3. 参数验证演示:")
	
	// 演示策略参数验证
	fmt.Println("  策略参数验证:")
	validParams := map[string]interface{}{
		"ma_short":      20,
		"ma_long":       50,
		"stop_loss":     0.05,
		"take_profit":   0.1,
		"leverage":      5,
		"position_size": 1000,
	}
	
	if err := validation.StrategyParamsValidator(validParams); err != nil {
		fmt.Printf("    验证失败: %s\n", err.Error())
	} else {
		fmt.Printf("    验证通过: 参数有效\n")
	}
	
	// 无效参数
	invalidParams := map[string]interface{}{
		"ma_short":      -5,  // 无效：负数
		"ma_long":       300, // 无效：超出范围
		"stop_loss":     1.5, // 无效：超出范围
		"leverage":      150, // 无效：超出范围
	}
	
	if err := validation.StrategyParamsValidator(invalidParams); err != nil {
		fmt.Printf("    验证失败: %s\n", err.Error())
	}
	
	// 演示订单验证
	fmt.Println("\n  订单验证:")
	validOrder := map[string]interface{}{
		"symbol":   "BTCUSDT",
		"side":     "BUY",
		"type":     "LIMIT",
		"quantity": 1.5,
		"price":    45000.0,
	}
	
	if err := validation.OrderValidator(validOrder); err != nil {
		fmt.Printf("    验证失败: %s\n", err.Error())
	} else {
		fmt.Printf("    验证通过: 订单有效\n")
	}
	
	fmt.Println()
}

func demonstrateStructuredLogging() {
	fmt.Println("4. 结构化日志演示:")
	
	// 基本日志
	logger.Info("系统启动", "version", "1.0.0", "port", 8080)
	
	// 带上下文的日志
	ctx := context.WithValue(context.Background(), "request_id", "req_12345")
	contextLogger := logger.WithContext(ctx)
	contextLogger.Info("处理用户请求", "user_id", "user_67890", "action", "create_strategy")
	
	// 错误日志
	err := errors.NewAppError(errors.ErrCodeStrategyExecution, "策略执行失败", nil)
	logger.Error("策略执行出错",
		"error_code", err.Code,
		"message", err.Message,
		"severity", err.Severity,
		"strategy_id", "strategy_123",
	)
	
	// 警告日志
	logger.Warn("系统资源使用率较高",
		"cpu_usage", 85.5,
		"memory_usage", 78.2,
		"disk_usage", 65.0,
	)
	
	// 调试日志
	logger.Debug("缓存操作",
		"operation", "set",
		"key", "market_data_BTCUSDT",
		"ttl", "60s",
	)
	
	fmt.Println()
}

func demonstratePerformanceLogging() {
	fmt.Println("5. 性能日志演示:")
	
	// 创建性能日志记录器
	perfLogger := logger.NewPerformanceLogger(logger.GetGlobalLogger())
	
	// 模拟一些操作并记录性能
	operations := []struct {
		name     string
		duration time.Duration
		fields   map[string]interface{}
	}{
		{
			name:     "数据库查询",
			duration: 150 * time.Millisecond,
			fields: map[string]interface{}{
				"query": "SELECT * FROM strategies",
				"rows":  25,
			},
		},
		{
			name:     "策略优化",
			duration: 2500 * time.Millisecond,
			fields: map[string]interface{}{
				"strategy_id": "strategy_123",
				"iterations": 1000,
			},
		},
		{
			name:     "缓存操作",
			duration: 5 * time.Millisecond,
			fields: map[string]interface{}{
				"operation": "get",
				"hit":       true,
			},
		},
		{
			name:     "API请求",
			duration: 8 * time.Second, // 慢请求，会记录为错误
			fields: map[string]interface{}{
				"endpoint": "/api/v1/strategies",
				"method":   "GET",
			},
		},
	}
	
	for _, op := range operations {
		perfLogger.LogPerformance(op.name, op.duration, op.fields)
	}
	
	fmt.Println()
}

// 演示HTTP请求日志记录
func demonstrateHTTPRequestLogging() {
	fmt.Println("6. HTTP请求日志演示:")
	
	// 模拟HTTP请求信息
	requests := []logger.HTTPRequestInfo{
		{
			Method:     "GET",
			Path:       "/api/v1/strategies",
			StatusCode: 200,
			Latency:    45 * time.Millisecond,
			ClientIP:   "192.168.1.100",
			UserAgent:  "QCAT-Client/1.0",
			BodySize:   1024,
			RequestID:  "req_12345",
			UserID:     "user_67890",
		},
		{
			Method:     "POST",
			Path:       "/api/v1/optimizer/run",
			StatusCode: 500,
			Latency:    2 * time.Second,
			ClientIP:   "192.168.1.101",
			UserAgent:  "curl/7.68.0",
			BodySize:   512,
			RequestID:  "req_12346",
		},
		{
			Method:     "PUT",
			Path:       "/api/v1/strategies/123",
			StatusCode: 400,
			Latency:    25 * time.Millisecond,
			ClientIP:   "192.168.1.102",
			UserAgent:  "PostmanRuntime/7.28.0",
			BodySize:   256,
			RequestID:  "req_12347",
			UserID:     "user_67891",
		},
	}
	
	for _, req := range requests {
		logger.LogHTTPRequest(req)
	}
	
	fmt.Println()
}

func init() {
	// 设置标准库日志输出格式
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}