package errors

import (
	"fmt"
	"net/http"
	"time"
)

// ErrorCode 定义错误代码类型
type ErrorCode string

// 错误代码常量
const (
	// 通用错误 (1000-1999)
	ErrCodeInternal       ErrorCode = "INTERNAL_ERROR"
	ErrCodeInvalidInput   ErrorCode = "INVALID_INPUT"
	ErrCodeNotFound       ErrorCode = "NOT_FOUND"
	ErrCodeUnauthorized   ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden      ErrorCode = "FORBIDDEN"
	ErrCodeConflict       ErrorCode = "CONFLICT"
	ErrCodeTimeout        ErrorCode = "TIMEOUT"
	ErrCodeRateLimit      ErrorCode = "RATE_LIMIT"

	// 数据库错误 (2000-2999)
	ErrCodeDBConnection   ErrorCode = "DB_CONNECTION_ERROR"
	ErrCodeDBQuery        ErrorCode = "DB_QUERY_ERROR"
	ErrCodeDBTransaction  ErrorCode = "DB_TRANSACTION_ERROR"
	ErrCodeDBConstraint   ErrorCode = "DB_CONSTRAINT_ERROR"

	// 缓存错误 (3000-3999)
	ErrCodeCacheConnection ErrorCode = "CACHE_CONNECTION_ERROR"
	ErrCodeCacheOperation  ErrorCode = "CACHE_OPERATION_ERROR"
	ErrCodeCacheMiss       ErrorCode = "CACHE_MISS"

	// 策略错误 (4000-4999)
	ErrCodeStrategyNotFound    ErrorCode = "STRATEGY_NOT_FOUND"
	ErrCodeStrategyInvalid     ErrorCode = "STRATEGY_INVALID"
	ErrCodeStrategyExecution   ErrorCode = "STRATEGY_EXECUTION_ERROR"
	ErrCodeParameterInvalid    ErrorCode = "PARAMETER_INVALID"
	ErrCodeOptimizationFailed  ErrorCode = "OPTIMIZATION_FAILED"

	// 交易错误 (5000-5999)
	ErrCodeOrderInvalid       ErrorCode = "ORDER_INVALID"
	ErrCodeOrderExecution     ErrorCode = "ORDER_EXECUTION_ERROR"
	ErrCodeInsufficientFunds  ErrorCode = "INSUFFICIENT_FUNDS"
	ErrCodePositionNotFound   ErrorCode = "POSITION_NOT_FOUND"
	ErrCodeExchangeConnection ErrorCode = "EXCHANGE_CONNECTION_ERROR"
	ErrCodeExchangeAPI        ErrorCode = "EXCHANGE_API_ERROR"

	// 风控错误 (6000-6999)
	ErrCodeRiskLimitExceeded  ErrorCode = "RISK_LIMIT_EXCEEDED"
	ErrCodeRiskValidation     ErrorCode = "RISK_VALIDATION_ERROR"
	ErrCodeCircuitBreaker     ErrorCode = "CIRCUIT_BREAKER_TRIGGERED"
	ErrCodeMarginInsufficient ErrorCode = "MARGIN_INSUFFICIENT"

	// 市场数据错误 (7000-7999)
	ErrCodeMarketDataUnavailable ErrorCode = "MARKET_DATA_UNAVAILABLE"
	ErrCodeMarketDataInvalid     ErrorCode = "MARKET_DATA_INVALID"
	ErrCodeMarketDataTimeout     ErrorCode = "MARKET_DATA_TIMEOUT"
)

// ErrorSeverity 定义错误严重程度
type ErrorSeverity string

const (
	SeverityLow      ErrorSeverity = "low"
	SeverityMedium   ErrorSeverity = "medium"
	SeverityHigh     ErrorSeverity = "high"
	SeverityCritical ErrorSeverity = "critical"
)

// AppError 应用错误结构
type AppError struct {
	Code      ErrorCode     `json:"code"`
	Message   string        `json:"message"`
	Details   string        `json:"details,omitempty"`
	Severity  ErrorSeverity `json:"severity"`
	Timestamp time.Time     `json:"timestamp"`
	RequestID string        `json:"request_id,omitempty"`
	UserID    string        `json:"user_id,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Cause     error         `json:"-"`
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap 返回原始错误
func (e *AppError) Unwrap() error {
	return e.Cause
}

// HTTPStatus 返回对应的HTTP状态码
func (e *AppError) HTTPStatus() int {
	switch e.Code {
	case ErrCodeNotFound, ErrCodeStrategyNotFound, ErrCodePositionNotFound:
		return http.StatusNotFound
	case ErrCodeUnauthorized:
		return http.StatusUnauthorized
	case ErrCodeForbidden:
		return http.StatusForbidden
	case ErrCodeInvalidInput, ErrCodeStrategyInvalid, ErrCodeParameterInvalid, ErrCodeOrderInvalid:
		return http.StatusBadRequest
	case ErrCodeConflict:
		return http.StatusConflict
	case ErrCodeTimeout, ErrCodeMarketDataTimeout:
		return http.StatusRequestTimeout
	case ErrCodeRateLimit:
		return http.StatusTooManyRequests
	case ErrCodeInsufficientFunds, ErrCodeMarginInsufficient:
		return http.StatusPaymentRequired
	case ErrCodeRiskLimitExceeded, ErrCodeCircuitBreaker:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

// NewAppError 创建新的应用错误
func NewAppError(code ErrorCode, message string, cause error) *AppError {
	severity := getSeverityByCode(code)
	return &AppError{
		Code:      code,
		Message:   message,
		Severity:  severity,
		Timestamp: time.Now(),
		Cause:     cause,
		Context:   make(map[string]interface{}),
	}
}

// NewAppErrorWithDetails 创建带详细信息的应用错误
func NewAppErrorWithDetails(code ErrorCode, message, details string, cause error) *AppError {
	err := NewAppError(code, message, cause)
	err.Details = details
	return err
}

// WithContext 添加上下文信息
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithRequestID 添加请求ID
func (e *AppError) WithRequestID(requestID string) *AppError {
	e.RequestID = requestID
	return e
}

// WithUserID 添加用户ID
func (e *AppError) WithUserID(userID string) *AppError {
	e.UserID = userID
	return e
}

// getSeverityByCode 根据错误代码确定严重程度
func getSeverityByCode(code ErrorCode) ErrorSeverity {
	switch code {
	case ErrCodeInternal, ErrCodeDBConnection, ErrCodeExchangeConnection:
		return SeverityCritical
	case ErrCodeDBQuery, ErrCodeDBTransaction, ErrCodeStrategyExecution, 
		 ErrCodeOrderExecution, ErrCodeRiskLimitExceeded, ErrCodeCircuitBreaker:
		return SeverityHigh
	case ErrCodeCacheConnection, ErrCodeCacheOperation, ErrCodeOptimizationFailed,
		 ErrCodeMarketDataUnavailable, ErrCodeRiskValidation:
		return SeverityMedium
	default:
		return SeverityLow
	}
}

// IsRetryable 判断错误是否可重试
func (e *AppError) IsRetryable() bool {
	switch e.Code {
	case ErrCodeTimeout, ErrCodeDBConnection, ErrCodeCacheConnection,
		 ErrCodeExchangeConnection, ErrCodeMarketDataTimeout:
		return true
	default:
		return false
	}
}

// ErrorResponse API错误响应结构
type ErrorResponse struct {
	Error     *AppError `json:"error"`
	Success   bool      `json:"success"`
	Timestamp time.Time `json:"timestamp"`
	Path      string    `json:"path,omitempty"`
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(err *AppError, path string) *ErrorResponse {
	return &ErrorResponse{
		Error:     err,
		Success:   false,
		Timestamp: time.Now(),
		Path:      path,
	}
}

// 预定义的常用错误
var (
	ErrInternalServer = NewAppError(ErrCodeInternal, "Internal server error", nil)
	ErrInvalidInput   = NewAppError(ErrCodeInvalidInput, "Invalid input parameters", nil)
	ErrNotFound       = NewAppError(ErrCodeNotFound, "Resource not found", nil)
	ErrUnauthorized   = NewAppError(ErrCodeUnauthorized, "Unauthorized access", nil)
	ErrForbidden      = NewAppError(ErrCodeForbidden, "Access forbidden", nil)
	ErrTimeout        = NewAppError(ErrCodeTimeout, "Request timeout", nil)
	ErrRateLimit      = NewAppError(ErrCodeRateLimit, "Rate limit exceeded", nil)
)

// WrapError 包装标准错误为应用错误
func WrapError(err error, code ErrorCode, message string) *AppError {
	if err == nil {
		return nil
	}
	
	// 如果已经是AppError，直接返回
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	
	return NewAppError(code, message, err)
}

// IsAppError 检查是否为应用错误
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// GetAppError 获取应用错误
func GetAppError(err error) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	return nil
}