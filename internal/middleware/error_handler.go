package middleware

import (
	"encoding/json"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"qcat/internal/errors"
	"qcat/internal/logger"
)

// ErrorHandler 错误处理中间件
func ErrorHandler() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		var err error
		
		// 处理panic
		if recovered != nil {
			// 记录panic堆栈
			logger.Error("Panic recovered",
				"error", recovered,
				"stack", string(debug.Stack()),
				"path", c.Request.URL.Path,
				"method", c.Request.Method,
			)
			
			// 创建内部服务器错误
			err = errors.NewAppError(
				errors.ErrCodeInternal,
				"Internal server error",
				nil,
			).WithRequestID(getRequestID(c))
		}
		
		// 处理错误响应
		handleError(c, err)
	})
}

// HandleError 处理错误的中间件函数
func HandleError(c *gin.Context) {
	c.Next()
	
	// 检查是否有错误
	if len(c.Errors) > 0 {
		err := c.Errors.Last().Err
		handleError(c, err)
	}
}

// handleError 统一错误处理
func handleError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	
	var appErr *errors.AppError
	
	// 转换为应用错误
	if errors.IsAppError(err) {
		appErr = errors.GetAppError(err)
	} else {
		// 包装标准错误
		appErr = errors.WrapError(err, errors.ErrCodeInternal, "Internal server error")
	}
	
	// 添加请求上下文
	if appErr.RequestID == "" {
		appErr = appErr.WithRequestID(getRequestID(c))
	}
	
	// 添加用户ID（如果存在）
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(string); ok {
			appErr = appErr.WithUserID(uid)
		}
	}
	
	// 记录错误日志
	logError(c, appErr)
	
	// 创建错误响应
	response := errors.NewErrorResponse(appErr, c.Request.URL.Path)
	
	// 设置响应头
	c.Header("Content-Type", "application/json")
	
	// 返回错误响应
	c.JSON(appErr.HTTPStatus(), response)
	c.Abort()
}

// logError 记录错误日志
func logError(c *gin.Context, err *errors.AppError) {
	fields := []interface{}{
		"error_code", err.Code,
		"message", err.Message,
		"severity", err.Severity,
		"request_id", err.RequestID,
		"user_id", err.UserID,
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"user_agent", c.Request.UserAgent(),
		"ip", c.ClientIP(),
	}
	
	// 添加错误详情
	if err.Details != "" {
		fields = append(fields, "details", err.Details)
	}
	
	// 添加上下文信息
	if len(err.Context) > 0 {
		contextJSON, _ := json.Marshal(err.Context)
		fields = append(fields, "context", string(contextJSON))
	}
	
	// 添加原始错误
	if err.Cause != nil {
		fields = append(fields, "cause", err.Cause.Error())
	}
	
	// 根据严重程度选择日志级别
	switch err.Severity {
	case errors.SeverityCritical:
		logger.Error("Critical error occurred", fields...)
	case errors.SeverityHigh:
		logger.Error("High severity error occurred", fields...)
	case errors.SeverityMedium:
		logger.Warn("Medium severity error occurred", fields...)
	default:
		logger.Info("Low severity error occurred", fields...)
	}
}

// getRequestID 获取请求ID
func getRequestID(c *gin.Context) string {
	if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
		return requestID
	}
	if requestID, exists := c.Get("request_id"); exists {
		if rid, ok := requestID.(string); ok {
			return rid
		}
	}
	return ""
}

// ValidationErrorHandler 验证错误处理器
func ValidationErrorHandler(err error) *errors.AppError {
	if err == nil {
		return nil
	}
	
	// 如果已经是应用错误，直接返回
	if appErr := errors.GetAppError(err); appErr != nil {
		return appErr
	}
	
	// 包装为验证错误
	return errors.NewAppError(
		errors.ErrCodeInvalidInput,
		"Validation failed",
		err,
	)
}

// DatabaseErrorHandler 数据库错误处理器
func DatabaseErrorHandler(err error) *errors.AppError {
	if err == nil {
		return nil
	}
	
	errMsg := err.Error()
	
	// 根据错误信息判断错误类型
	switch {
	case containsAny(errMsg, []string{"connection", "connect", "dial"}):
		return errors.NewAppError(
			errors.ErrCodeDBConnection,
			"Database connection error",
			err,
		)
	case containsAny(errMsg, []string{"constraint", "duplicate", "unique"}):
		return errors.NewAppError(
			errors.ErrCodeDBConstraint,
			"Database constraint violation",
			err,
		)
	case containsAny(errMsg, []string{"transaction", "rollback", "commit"}):
		return errors.NewAppError(
			errors.ErrCodeDBTransaction,
			"Database transaction error",
			err,
		)
	default:
		return errors.NewAppError(
			errors.ErrCodeDBQuery,
			"Database query error",
			err,
		)
	}
}

// CacheErrorHandler 缓存错误处理器
func CacheErrorHandler(err error) *errors.AppError {
	if err == nil {
		return nil
	}
	
	errMsg := err.Error()
	
	// 根据错误信息判断错误类型
	switch {
	case containsAny(errMsg, []string{"connection", "connect", "dial", "timeout"}):
		return errors.NewAppError(
			errors.ErrCodeCacheConnection,
			"Cache connection error",
			err,
		)
	case containsAny(errMsg, []string{"not found", "miss", "nil"}):
		return errors.NewAppError(
			errors.ErrCodeCacheMiss,
			"Cache miss",
			err,
		)
	default:
		return errors.NewAppError(
			errors.ErrCodeCacheOperation,
			"Cache operation error",
			err,
		)
	}
}

// ExchangeErrorHandler 交易所错误处理器
func ExchangeErrorHandler(err error) *errors.AppError {
	if err == nil {
		return nil
	}
	
	errMsg := err.Error()
	
	// 根据错误信息判断错误类型
	switch {
	case containsAny(errMsg, []string{"connection", "connect", "timeout", "network"}):
		return errors.NewAppError(
			errors.ErrCodeExchangeConnection,
			"Exchange connection error",
			err,
		)
	case containsAny(errMsg, []string{"insufficient", "balance", "funds"}):
		return errors.NewAppError(
			errors.ErrCodeInsufficientFunds,
			"Insufficient funds",
			err,
		)
	case containsAny(errMsg, []string{"invalid", "order", "parameter"}):
		return errors.NewAppError(
			errors.ErrCodeOrderInvalid,
			"Invalid order parameters",
			err,
		)
	case containsAny(errMsg, []string{"rate", "limit", "throttle"}):
		return errors.NewAppError(
			errors.ErrCodeRateLimit,
			"Rate limit exceeded",
			err,
		)
	default:
		return errors.NewAppError(
			errors.ErrCodeExchangeAPI,
			"Exchange API error",
			err,
		)
	}
}

// containsAny 检查字符串是否包含任意一个子字符串
func containsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}

// RetryableErrorHandler 可重试错误处理器
func RetryableErrorHandler(err error, maxRetries int) (*errors.AppError, bool) {
	if err == nil {
		return nil, false
	}
	
	appErr := errors.GetAppError(err)
	if appErr == nil {
		appErr = errors.WrapError(err, errors.ErrCodeInternal, "Internal error")
	}
	
	// 检查是否可重试
	if appErr.IsRetryable() && maxRetries > 0 {
		return appErr, true
	}
	
	return appErr, false
}

// ErrorMetrics 错误指标收集
type ErrorMetrics struct {
	TotalErrors   int64            `json:"total_errors"`
	ErrorsByCode  map[string]int64 `json:"errors_by_code"`
	ErrorsByPath  map[string]int64 `json:"errors_by_path"`
	CriticalErrors int64           `json:"critical_errors"`
	HighErrors    int64            `json:"high_errors"`
}

var globalErrorMetrics = &ErrorMetrics{
	ErrorsByCode: make(map[string]int64),
	ErrorsByPath: make(map[string]int64),
}

// collectErrorMetrics 收集错误指标
func collectErrorMetrics(c *gin.Context, err *errors.AppError) {
	globalErrorMetrics.TotalErrors++
	globalErrorMetrics.ErrorsByCode[string(err.Code)]++
	globalErrorMetrics.ErrorsByPath[c.Request.URL.Path]++
	
	switch err.Severity {
	case errors.SeverityCritical:
		globalErrorMetrics.CriticalErrors++
	case errors.SeverityHigh:
		globalErrorMetrics.HighErrors++
	}
}

// GetErrorMetrics 获取错误指标
func GetErrorMetrics() *ErrorMetrics {
	return globalErrorMetrics
}