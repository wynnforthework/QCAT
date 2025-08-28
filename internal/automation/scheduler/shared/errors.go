package shared

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// ErrorCode constants for different types of errors
const (
	ErrCodeDatabaseConnection    = "DB_CONNECTION_ERROR"
	ErrCodeExchangeAPI          = "EXCHANGE_API_ERROR"
	ErrCodeInvalidConfiguration = "INVALID_CONFIG_ERROR"
	ErrCodeInsufficientData     = "INSUFFICIENT_DATA_ERROR"
	ErrCodeRiskThresholdExceeded = "RISK_THRESHOLD_EXCEEDED"
	ErrCodePositionNotFound     = "POSITION_NOT_FOUND"
	ErrCodeInvalidParameters    = "INVALID_PARAMETERS"
	ErrCodeTimeout              = "TIMEOUT_ERROR"
	ErrCodeRateLimitExceeded    = "RATE_LIMIT_EXCEEDED"
	ErrCodeSystemOverload       = "SYSTEM_OVERLOAD"
	ErrCodeCalculationFailed    = "CALCULATION_FAILED"
	ErrCodePartialFailure       = "PARTIAL_FAILURE"
	ErrCodeRiskAssessmentFailed = "RISK_ASSESSMENT_FAILED"
)

// NewAutomationError creates a new AutomationError
func NewAutomationError(code, message, component string, severity ErrorSeverity, retryable bool) *AutomationError {
	return &AutomationError{
		Code:      code,
		Message:   message,
		Severity:  severity,
		Component: component,
		Timestamp: time.Now(),
		Context:   make(map[string]interface{}),
		Retryable: retryable,
	}
}

// WithContext adds context information to the error
func (ae *AutomationError) WithContext(keyValuePairs ...interface{}) *AutomationError {
	for i := 0; i < len(keyValuePairs); i += 2 {
		if i+1 < len(keyValuePairs) {
			if key, ok := keyValuePairs[i].(string); ok {
				ae.Context[key] = keyValuePairs[i+1]
			}
		}
	}
	return ae
}

// RecoveryStrategy defines the interface for error recovery strategies
type RecoveryStrategy interface {
	CanRecover(err error) bool
	Recover(ctx context.Context, err error) error
	GetRecoveryTime() time.Duration
}

// RetryStrategy implements exponential backoff retry strategy
type RetryStrategy struct {
	MaxRetries      int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	RetryableErrors map[string]bool
}

// NewRetryStrategy creates a new retry strategy
func NewRetryStrategy(maxRetries int, initialDelay, maxDelay time.Duration, backoffFactor float64) *RetryStrategy {
	return &RetryStrategy{
		MaxRetries:    maxRetries,
		InitialDelay:  initialDelay,
		MaxDelay:      maxDelay,
		BackoffFactor: backoffFactor,
		RetryableErrors: map[string]bool{
			ErrCodeDatabaseConnection: true,
			ErrCodeExchangeAPI:       true,
			ErrCodeTimeout:           true,
			ErrCodeRateLimitExceeded: true,
			ErrCodeSystemOverload:    true,
		},
	}
}

// CanRecover checks if the error is retryable
func (rs *RetryStrategy) CanRecover(err error) bool {
	if ae, ok := err.(*AutomationError); ok {
		return ae.Retryable && rs.RetryableErrors[ae.Code]
	}
	return false
}

// Recover implements the recovery logic with exponential backoff
func (rs *RetryStrategy) Recover(ctx context.Context, err error) error {
	if !rs.CanRecover(err) {
		return err
	}

	delay := rs.InitialDelay
	for attempt := 0; attempt < rs.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Recovery attempt would be implemented by the caller
			log.Printf("Recovery attempt %d/%d for error: %v", attempt+1, rs.MaxRetries, err)
			
			// Calculate next delay with exponential backoff
			delay = time.Duration(float64(delay) * rs.BackoffFactor)
			if delay > rs.MaxDelay {
				delay = rs.MaxDelay
			}
		}
	}

	return fmt.Errorf("recovery failed after %d attempts: %w", rs.MaxRetries, err)
}

// GetRecoveryTime returns the estimated recovery time
func (rs *RetryStrategy) GetRecoveryTime() time.Duration {
	totalTime := time.Duration(0)
	delay := rs.InitialDelay
	
	for i := 0; i < rs.MaxRetries; i++ {
		totalTime += delay
		delay = time.Duration(float64(delay) * rs.BackoffFactor)
		if delay > rs.MaxDelay {
			delay = rs.MaxDelay
		}
	}
	
	return totalTime
}

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	CircuitBreakerClosed CircuitBreakerState = iota
	CircuitBreakerOpen
	CircuitBreakerHalfOpen
)

// String returns the string representation of CircuitBreakerState
func (cbs CircuitBreakerState) String() string {
	switch cbs {
	case CircuitBreakerClosed:
		return "CLOSED"
	case CircuitBreakerOpen:
		return "OPEN"
	case CircuitBreakerHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreakerConfig holds configuration for circuit breaker
type CircuitBreakerConfig struct {
	FailureThreshold   int           `json:"failure_threshold"`
	RecoveryTimeout    time.Duration `json:"recovery_timeout"`
	HalfOpenRequests   int           `json:"half_open_requests"`
	SuccessThreshold   int           `json:"success_threshold"`
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	config           CircuitBreakerConfig
	state            CircuitBreakerState
	failureCount     int
	successCount     int
	lastFailureTime  time.Time
	halfOpenRequests int
	mu               sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		state:  CircuitBreakerClosed,
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Check if circuit breaker allows execution
	if !cb.canExecute() {
		return NewAutomationError(
			"CIRCUIT_BREAKER_OPEN",
			"Circuit breaker is open",
			"CircuitBreaker",
			ErrorSeverityHigh,
			true,
		)
	}

	// Execute the function
	err := fn()
	
	// Update circuit breaker state based on result
	cb.recordResult(err)
	
	return err
}

// canExecute checks if the circuit breaker allows execution
func (cb *CircuitBreaker) canExecute() bool {
	switch cb.state {
	case CircuitBreakerClosed:
		return true
	case CircuitBreakerOpen:
		// Check if recovery timeout has passed
		if time.Since(cb.lastFailureTime) >= cb.config.RecoveryTimeout {
			cb.state = CircuitBreakerHalfOpen
			cb.halfOpenRequests = 0
			cb.successCount = 0
			return true
		}
		return false
	case CircuitBreakerHalfOpen:
		return cb.halfOpenRequests < cb.config.HalfOpenRequests
	default:
		return false
	}
}

// recordResult records the result of an execution
func (cb *CircuitBreaker) recordResult(err error) {
	switch cb.state {
	case CircuitBreakerClosed:
		if err != nil {
			cb.failureCount++
			cb.lastFailureTime = time.Now()
			if cb.failureCount >= cb.config.FailureThreshold {
				cb.state = CircuitBreakerOpen
				log.Printf("Circuit breaker opened due to %d failures", cb.failureCount)
			}
		} else {
			cb.failureCount = 0
		}
	case CircuitBreakerHalfOpen:
		cb.halfOpenRequests++
		if err != nil {
			cb.state = CircuitBreakerOpen
			cb.lastFailureTime = time.Now()
			log.Printf("Circuit breaker reopened due to failure in half-open state")
		} else {
			cb.successCount++
			if cb.successCount >= cb.config.SuccessThreshold {
				cb.state = CircuitBreakerClosed
				cb.failureCount = 0
				log.Printf("Circuit breaker closed after %d successful requests", cb.successCount)
			}
		}
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetMetrics returns circuit breaker metrics
func (cb *CircuitBreaker) GetMetrics() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	return map[string]interface{}{
		"state":              cb.state.String(),
		"failure_count":      cb.failureCount,
		"success_count":      cb.successCount,
		"half_open_requests": cb.halfOpenRequests,
		"last_failure_time":  cb.lastFailureTime,
	}
}

// ErrorHandler provides centralized error handling
type ErrorHandler struct {
	retryStrategy   *RetryStrategy
	circuitBreaker  *CircuitBreaker
	errorCallbacks  map[string][]func(error)
	mu              sync.RWMutex
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(retryStrategy *RetryStrategy, circuitBreaker *CircuitBreaker) *ErrorHandler {
	return &ErrorHandler{
		retryStrategy:  retryStrategy,
		circuitBreaker: circuitBreaker,
		errorCallbacks: make(map[string][]func(error)),
	}
}

// Handle handles an error with appropriate recovery strategy
func (eh *ErrorHandler) Handle(ctx context.Context, err error, operation func() error) error {
	// Execute callbacks for this error type
	eh.executeCallbacks(err)
	
	// Try circuit breaker execution first
	if eh.circuitBreaker != nil {
		return eh.circuitBreaker.Execute(ctx, func() error {
			// Try retry strategy if available
			if eh.retryStrategy != nil && eh.retryStrategy.CanRecover(err) {
				return eh.executeWithRetry(ctx, operation)
			}
			return operation()
		})
	}
	
	// Fallback to retry strategy only
	if eh.retryStrategy != nil && eh.retryStrategy.CanRecover(err) {
		return eh.executeWithRetry(ctx, operation)
	}
	
	return err
}

// executeWithRetry executes an operation with retry logic
func (eh *ErrorHandler) executeWithRetry(ctx context.Context, operation func() error) error {
	delay := eh.retryStrategy.InitialDelay
	
	for attempt := 0; attempt < eh.retryStrategy.MaxRetries; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}
		
		// Check if error is retryable
		if !eh.retryStrategy.CanRecover(err) {
			return err
		}
		
		// Wait before retry
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Calculate next delay
			delay = time.Duration(float64(delay) * eh.retryStrategy.BackoffFactor)
			if delay > eh.retryStrategy.MaxDelay {
				delay = eh.retryStrategy.MaxDelay
			}
		}
	}
	
	return fmt.Errorf("operation failed after %d retries", eh.retryStrategy.MaxRetries)
}

// RegisterErrorCallback registers a callback for specific error codes
func (eh *ErrorHandler) RegisterErrorCallback(errorCode string, callback func(error)) {
	eh.mu.Lock()
	defer eh.mu.Unlock()
	
	eh.errorCallbacks[errorCode] = append(eh.errorCallbacks[errorCode], callback)
}

// executeCallbacks executes registered callbacks for an error
func (eh *ErrorHandler) executeCallbacks(err error) {
	eh.mu.RLock()
	defer eh.mu.RUnlock()
	
	if ae, ok := err.(*AutomationError); ok {
		if callbacks, exists := eh.errorCallbacks[ae.Code]; exists {
			for _, callback := range callbacks {
				go callback(err) // Execute callbacks asynchronously
			}
		}
	}
}

// GracefulDegradation provides graceful degradation capabilities
type GracefulDegradation struct {
	fallbackStrategies map[string]func(ctx context.Context) error
	mu                 sync.RWMutex
}

// NewGracefulDegradation creates a new graceful degradation handler
func NewGracefulDegradation() *GracefulDegradation {
	return &GracefulDegradation{
		fallbackStrategies: make(map[string]func(ctx context.Context) error),
	}
}

// RegisterFallback registers a fallback strategy for a specific component
func (gd *GracefulDegradation) RegisterFallback(component string, fallback func(ctx context.Context) error) {
	gd.mu.Lock()
	defer gd.mu.Unlock()
	
	gd.fallbackStrategies[component] = fallback
}

// ExecuteWithFallback executes an operation with fallback if it fails
func (gd *GracefulDegradation) ExecuteWithFallback(ctx context.Context, component string, operation func() error) error {
	err := operation()
	if err == nil {
		return nil
	}
	
	gd.mu.RLock()
	fallback, exists := gd.fallbackStrategies[component]
	gd.mu.RUnlock()
	
	if exists {
		log.Printf("Executing fallback strategy for component: %s, error: %v", component, err)
		return fallback(ctx)
	}
	
	return err
}

// DeadLetterQueue handles failed tasks that need manual intervention
type DeadLetterQueue struct {
	failedTasks []FailedTask
	maxSize     int
	mu          sync.RWMutex
}

// FailedTask represents a task that failed and needs manual intervention
type FailedTask struct {
	ID          string                 `json:"id"`
	Component   string                 `json:"component"`
	Operation   string                 `json:"operation"`
	Error       error                  `json:"error"`
	Attempts    int                    `json:"attempts"`
	LastAttempt time.Time              `json:"last_attempt"`
	Context     map[string]interface{} `json:"context"`
}

// NewDeadLetterQueue creates a new dead letter queue
func NewDeadLetterQueue(maxSize int) *DeadLetterQueue {
	return &DeadLetterQueue{
		failedTasks: make([]FailedTask, 0),
		maxSize:     maxSize,
	}
}

// Add adds a failed task to the dead letter queue
func (dlq *DeadLetterQueue) Add(task FailedTask) {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()
	
	// Remove oldest task if queue is full
	if len(dlq.failedTasks) >= dlq.maxSize {
		dlq.failedTasks = dlq.failedTasks[1:]
	}
	
	dlq.failedTasks = append(dlq.failedTasks, task)
	log.Printf("Added failed task to dead letter queue: %s", task.ID)
}

// GetFailedTasks returns all failed tasks
func (dlq *DeadLetterQueue) GetFailedTasks() []FailedTask {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()
	
	// Return a copy to avoid race conditions
	tasks := make([]FailedTask, len(dlq.failedTasks))
	copy(tasks, dlq.failedTasks)
	return tasks
}

// Remove removes a task from the dead letter queue
func (dlq *DeadLetterQueue) Remove(taskID string) bool {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()
	
	for i, task := range dlq.failedTasks {
		if task.ID == taskID {
			dlq.failedTasks = append(dlq.failedTasks[:i], dlq.failedTasks[i+1:]...)
			return true
		}
	}
	return false
}

// Size returns the current size of the dead letter queue
func (dlq *DeadLetterQueue) Size() int {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()
	return len(dlq.failedTasks)
}