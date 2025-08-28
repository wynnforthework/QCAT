package shared

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"
)

// SchedulerFactory creates and manages scheduler instances
type SchedulerFactory struct {
	config         *AutomationConfig
	configManager  *ConfigManager
	errorHandler   *ErrorHandler
	deadLetterQueue *DeadLetterQueue
	metricsCollector MetricsCollector
	eventPublisher  EventPublisher
	mu             sync.RWMutex
}

// NewSchedulerFactory creates a new scheduler factory
func NewSchedulerFactory(
	config *AutomationConfig,
	metricsCollector MetricsCollector,
	eventPublisher EventPublisher,
) *SchedulerFactory {
	configManager := NewConfigManager()
	
	// Create retry strategy from config
	retryStrategy := NewRetryStrategy(
		config.Common.ErrorHandling.RetryConfig.MaxRetries,
		config.Common.ErrorHandling.RetryConfig.InitialDelay,
		config.Common.ErrorHandling.RetryConfig.MaxDelay,
		config.Common.ErrorHandling.RetryConfig.BackoffFactor,
	)
	
	// Create circuit breaker from config
	circuitBreaker := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: config.Common.ErrorHandling.CircuitBreakerConfig.FailureThreshold,
		RecoveryTimeout:  config.Common.ErrorHandling.CircuitBreakerConfig.RecoveryTimeout,
		HalfOpenRequests: config.Common.ErrorHandling.CircuitBreakerConfig.HalfOpenRequests,
		SuccessThreshold: config.Common.ErrorHandling.CircuitBreakerConfig.SuccessThreshold,
	})
	
	// Create error handler
	errorHandler := NewErrorHandler(retryStrategy, circuitBreaker)
	
	// Create dead letter queue
	deadLetterQueue := NewDeadLetterQueue(config.Common.ErrorHandling.DeadLetterQueueSize)
	
	return &SchedulerFactory{
		config:          config,
		configManager:   configManager,
		errorHandler:    errorHandler,
		deadLetterQueue: deadLetterQueue,
		metricsCollector: metricsCollector,
		eventPublisher:  eventPublisher,
	}
}

// CreateBaseScheduler creates a base scheduler with common functionality
func (sf *SchedulerFactory) CreateBaseScheduler(name, schedulerType string) *BaseSchedulerImpl {
	return &BaseSchedulerImpl{
		name:             name,
		schedulerType:    schedulerType,
		config:           sf.config,
		configManager:    sf.configManager,
		errorHandler:     sf.errorHandler,
		deadLetterQueue:  sf.deadLetterQueue,
		metricsCollector: sf.metricsCollector,
		eventPublisher:   sf.eventPublisher,
		isRunning:        false,
		startTime:        time.Time{},
		taskStats:        &TaskStatistics{},
		healthStatus:     &HealthStatus{Status: "HEALTHY"},
	}
}

// GetErrorHandler returns the error handler
func (sf *SchedulerFactory) GetErrorHandler() *ErrorHandler {
	return sf.errorHandler
}

// GetDeadLetterQueue returns the dead letter queue
func (sf *SchedulerFactory) GetDeadLetterQueue() *DeadLetterQueue {
	return sf.deadLetterQueue
}

// GetConfig returns the automation config
func (sf *SchedulerFactory) GetConfig() *AutomationConfig {
	sf.mu.RLock()
	defer sf.mu.RUnlock()
	return sf.config
}

// UpdateConfig updates the automation config
func (sf *SchedulerFactory) UpdateConfig(config *AutomationConfig) {
	sf.mu.Lock()
	defer sf.mu.Unlock()
	sf.config = config
}

// BaseSchedulerImpl provides a base implementation of the SchedulerInterface
type BaseSchedulerImpl struct {
	name             string
	schedulerType    string
	version          string
	description      string
	supportedTasks   []string
	config           *AutomationConfig
	configManager    *ConfigManager
	errorHandler     *ErrorHandler
	deadLetterQueue  *DeadLetterQueue
	metricsCollector MetricsCollector
	eventPublisher   EventPublisher
	isRunning        bool
	startTime        time.Time
	taskStats        *TaskStatistics
	healthStatus     *HealthStatus
	mu               sync.RWMutex
}

// TaskStatistics holds task execution statistics
type TaskStatistics struct {
	TasksTotal    int64         `json:"tasks_total"`
	TasksSuccess  int64         `json:"tasks_success"`
	TasksFailed   int64         `json:"tasks_failed"`
	AverageTime   time.Duration `json:"average_time"`
	LastExecution time.Time     `json:"last_execution"`
	mu            sync.RWMutex
}

// Start starts the scheduler
func (bs *BaseSchedulerImpl) Start() error {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	
	if bs.isRunning {
		return fmt.Errorf("scheduler %s is already running", bs.name)
	}
	
	bs.isRunning = true
	bs.startTime = time.Now()
	bs.healthStatus.Status = "HEALTHY"
	bs.healthStatus.LastCheck = time.Now()
	
	// Record metrics
	if bs.metricsCollector != nil {
		bs.metricsCollector.Counter("scheduler_starts", map[string]string{
			"scheduler": bs.name,
			"type":      bs.schedulerType,
		})
	}
	
	// Publish event
	if bs.eventPublisher != nil {
		event := map[string]interface{}{
			"type":      "scheduler_started",
			"scheduler": bs.name,
			"timestamp": time.Now(),
		}
		bs.eventPublisher.Publish(context.Background(), event)
	}
	
	log.Printf("Scheduler %s (%s) started", bs.name, bs.schedulerType)
	return nil
}

// Stop stops the scheduler
func (bs *BaseSchedulerImpl) Stop() error {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	
	if !bs.isRunning {
		return fmt.Errorf("scheduler %s is not running", bs.name)
	}
	
	bs.isRunning = false
	bs.healthStatus.Status = "STOPPED"
	bs.healthStatus.LastCheck = time.Now()
	
	// Record metrics
	if bs.metricsCollector != nil {
		bs.metricsCollector.Counter("scheduler_stops", map[string]string{
			"scheduler": bs.name,
			"type":      bs.schedulerType,
		})
		
		uptime := time.Since(bs.startTime)
		bs.metricsCollector.Timer("scheduler_uptime", uptime, map[string]string{
			"scheduler": bs.name,
			"type":      bs.schedulerType,
		})
	}
	
	// Publish event
	if bs.eventPublisher != nil {
		event := map[string]interface{}{
			"type":      "scheduler_stopped",
			"scheduler": bs.name,
			"timestamp": time.Now(),
			"uptime":    time.Since(bs.startTime).String(),
		}
		bs.eventPublisher.Publish(context.Background(), event)
	}
	
	log.Printf("Scheduler %s (%s) stopped", bs.name, bs.schedulerType)
	return nil
}

// IsRunning returns whether the scheduler is running
func (bs *BaseSchedulerImpl) IsRunning() bool {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	return bs.isRunning
}

// GetStatus returns the scheduler status
func (bs *BaseSchedulerImpl) GetStatus() map[string]interface{} {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	
	uptime := time.Duration(0)
	if bs.isRunning {
		uptime = time.Since(bs.startTime)
	}
	
	return map[string]interface{}{
		"name":           bs.name,
		"type":           bs.schedulerType,
		"version":        bs.version,
		"is_running":     bs.isRunning,
		"uptime":         uptime.String(),
		"start_time":     bs.startTime,
		"health_status":  bs.healthStatus,
		"task_stats":     bs.getTaskStats(),
	}
}

// Execute executes a task with error handling and metrics
func (bs *BaseSchedulerImpl) Execute(ctx context.Context, task interface{}) error {
	startTime := time.Now()
	
	// Update task statistics
	bs.taskStats.mu.Lock()
	bs.taskStats.TasksTotal++
	bs.taskStats.LastExecution = startTime
	bs.taskStats.mu.Unlock()
	
	// Record metrics
	if bs.metricsCollector != nil {
		bs.metricsCollector.Counter("task_executions", map[string]string{
			"scheduler": bs.name,
			"type":      bs.schedulerType,
		})
	}
	
	// Execute with error handling
	err := bs.errorHandler.Handle(ctx, nil, func() error {
		return bs.executeTask(ctx, task)
	})
	
	// Update statistics and metrics
	duration := time.Since(startTime)
	bs.updateTaskStats(err == nil, duration)
	
	if bs.metricsCollector != nil {
		status := "success"
		if err != nil {
			status = "failure"
		}
		
		bs.metricsCollector.Timer("task_duration", duration, map[string]string{
			"scheduler": bs.name,
			"type":      bs.schedulerType,
			"status":    status,
		})
		
		if err != nil {
			bs.metricsCollector.Counter("task_failures", map[string]string{
				"scheduler": bs.name,
				"type":      bs.schedulerType,
			})
		}
	}
	
	// Handle failed tasks
	if err != nil {
		failedTask := FailedTask{
			ID:          GenerateID("failed_task"),
			Component:   bs.name,
			Operation:   "execute_task",
			Error:       err,
			Attempts:    1,
			LastAttempt: time.Now(),
			Context: map[string]interface{}{
				"scheduler_type": bs.schedulerType,
				"task":          task,
			},
		}
		bs.deadLetterQueue.Add(failedTask)
	}
	
	return err
}

// executeTask is a placeholder for actual task execution (to be overridden)
func (bs *BaseSchedulerImpl) executeTask(ctx context.Context, task interface{}) error {
	return fmt.Errorf("executeTask not implemented for scheduler %s", bs.name)
}

// CanExecute checks if the scheduler can execute a task (to be overridden)
func (bs *BaseSchedulerImpl) CanExecute(task interface{}) bool {
	return false
}

// GetExecutorType returns the executor type
func (bs *BaseSchedulerImpl) GetExecutorType() string {
	return bs.schedulerType
}

// GetName returns the scheduler name
func (bs *BaseSchedulerImpl) GetName() string {
	return bs.name
}

// GetVersion returns the scheduler version
func (bs *BaseSchedulerImpl) GetVersion() string {
	if bs.version == "" {
		return "1.0.0"
	}
	return bs.version
}

// GetDescription returns the scheduler description
func (bs *BaseSchedulerImpl) GetDescription() string {
	if bs.description == "" {
		return fmt.Sprintf("%s scheduler for automation tasks", bs.schedulerType)
	}
	return bs.description
}

// GetSupportedTasks returns the list of supported tasks
func (bs *BaseSchedulerImpl) GetSupportedTasks() []string {
	return bs.supportedTasks
}

// GetMetrics returns scheduler metrics
func (bs *BaseSchedulerImpl) GetMetrics() map[string]interface{} {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	
	uptime := time.Duration(0)
	if bs.isRunning {
		uptime = time.Since(bs.startTime)
	}
	
	taskStats := bs.getTaskStats()
	
	return map[string]interface{}{
		"uptime":              uptime.String(),
		"tasks_total":         taskStats.TasksTotal,
		"tasks_success":       taskStats.TasksSuccess,
		"tasks_failed":        taskStats.TasksFailed,
		"success_rate":        bs.calculateSuccessRate(),
		"average_task_time":   taskStats.AverageTime.String(),
		"last_execution":      taskStats.LastExecution,
		"dead_letter_queue_size": bs.deadLetterQueue.Size(),
	}
}

// GetHealth returns the health status
func (bs *BaseSchedulerImpl) GetHealth() HealthStatus {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	
	// Update health status
	bs.healthStatus.LastCheck = time.Now()
	bs.healthStatus.Uptime = time.Since(bs.startTime)
	
	taskStats := bs.getTaskStats()
	bs.healthStatus.TasksTotal = taskStats.TasksTotal
	bs.healthStatus.TasksSuccess = taskStats.TasksSuccess
	bs.healthStatus.TasksFailed = taskStats.TasksFailed
	
	// Determine health status based on success rate
	successRate := bs.calculateSuccessRate()
	if successRate < 0.5 && taskStats.TasksTotal > 10 {
		bs.healthStatus.Status = "UNHEALTHY"
		bs.healthStatus.Errors = []string{"Low success rate"}
	} else if successRate < 0.8 && taskStats.TasksTotal > 10 {
		bs.healthStatus.Status = "DEGRADED"
		bs.healthStatus.Warnings = []string{"Moderate success rate"}
	} else if bs.isRunning {
		bs.healthStatus.Status = "HEALTHY"
		bs.healthStatus.Errors = nil
		bs.healthStatus.Warnings = nil
	}
	
	return *bs.healthStatus
}

// SetVersion sets the scheduler version
func (bs *BaseSchedulerImpl) SetVersion(version string) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.version = version
}

// SetDescription sets the scheduler description
func (bs *BaseSchedulerImpl) SetDescription(description string) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.description = description
}

// SetSupportedTasks sets the list of supported tasks
func (bs *BaseSchedulerImpl) SetSupportedTasks(tasks []string) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.supportedTasks = tasks
}

// updateTaskStats updates task execution statistics
func (bs *BaseSchedulerImpl) updateTaskStats(success bool, duration time.Duration) {
	bs.taskStats.mu.Lock()
	defer bs.taskStats.mu.Unlock()
	
	if success {
		bs.taskStats.TasksSuccess++
	} else {
		bs.taskStats.TasksFailed++
	}
	
	// Update average time (simple moving average)
	if bs.taskStats.AverageTime == 0 {
		bs.taskStats.AverageTime = duration
	} else {
		bs.taskStats.AverageTime = (bs.taskStats.AverageTime + duration) / 2
	}
}

// getTaskStats returns a copy of task statistics
func (bs *BaseSchedulerImpl) getTaskStats() TaskStatistics {
	bs.taskStats.mu.RLock()
	defer bs.taskStats.mu.RUnlock()
	return *bs.taskStats
}

// calculateSuccessRate calculates the task success rate
func (bs *BaseSchedulerImpl) calculateSuccessRate() float64 {
	taskStats := bs.getTaskStats()
	if taskStats.TasksTotal == 0 {
		return 1.0
	}
	return float64(taskStats.TasksSuccess) / float64(taskStats.TasksTotal)
}

// DatabaseProvider interface for database operations
type DatabaseProvider interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// ExchangeProvider interface for exchange operations
type ExchangeProvider interface {
	GetPositions(ctx context.Context) ([]Position, error)
	GetMarketData(ctx context.Context, symbol string) (map[string]interface{}, error)
	PlaceOrder(ctx context.Context, order map[string]interface{}) (string, error)
	GetAccountInfo(ctx context.Context) (map[string]interface{}, error)
}

// SchedulerDependencies holds all dependencies needed by schedulers
type SchedulerDependencies struct {
	Database         DatabaseProvider
	Exchange         ExchangeProvider
	Config           *AutomationConfig
	MetricsCollector MetricsCollector
	EventPublisher   EventPublisher
	Logger           Logger
}

// Logger interface for logging operations
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Fatal(msg string, fields ...interface{})
}

// DefaultLogger provides a default logger implementation
type DefaultLogger struct{}

// Debug logs a debug message
func (dl *DefaultLogger) Debug(msg string, fields ...interface{}) {
	log.Printf("[DEBUG] "+msg, fields...)
}

// Info logs an info message
func (dl *DefaultLogger) Info(msg string, fields ...interface{}) {
	log.Printf("[INFO] "+msg, fields...)
}

// Warn logs a warning message
func (dl *DefaultLogger) Warn(msg string, fields ...interface{}) {
	log.Printf("[WARN] "+msg, fields...)
}

// Error logs an error message
func (dl *DefaultLogger) Error(msg string, fields ...interface{}) {
	log.Printf("[ERROR] "+msg, fields...)
}

// Fatal logs a fatal message and exits
func (dl *DefaultLogger) Fatal(msg string, fields ...interface{}) {
	log.Fatalf("[FATAL] "+msg, fields...)
}

// SchedulerRegistry manages registered schedulers
type SchedulerRegistry struct {
	schedulers map[string]SchedulerInterface
	mu         sync.RWMutex
}

// NewSchedulerRegistry creates a new scheduler registry
func NewSchedulerRegistry() *SchedulerRegistry {
	return &SchedulerRegistry{
		schedulers: make(map[string]SchedulerInterface),
	}
}

// Register registers a scheduler
func (sr *SchedulerRegistry) Register(name string, scheduler SchedulerInterface) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	
	if _, exists := sr.schedulers[name]; exists {
		return fmt.Errorf("scheduler %s is already registered", name)
	}
	
	sr.schedulers[name] = scheduler
	return nil
}

// Unregister unregisters a scheduler
func (sr *SchedulerRegistry) Unregister(name string) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	
	if _, exists := sr.schedulers[name]; !exists {
		return fmt.Errorf("scheduler %s is not registered", name)
	}
	
	delete(sr.schedulers, name)
	return nil
}

// Get retrieves a scheduler by name
func (sr *SchedulerRegistry) Get(name string) (SchedulerInterface, error) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	
	scheduler, exists := sr.schedulers[name]
	if !exists {
		return nil, fmt.Errorf("scheduler %s is not registered", name)
	}
	
	return scheduler, nil
}

// List returns all registered scheduler names
func (sr *SchedulerRegistry) List() []string {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	
	names := make([]string, 0, len(sr.schedulers))
	for name := range sr.schedulers {
		names = append(names, name)
	}
	
	return names
}

// GetAll returns all registered schedulers
func (sr *SchedulerRegistry) GetAll() map[string]SchedulerInterface {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	
	schedulers := make(map[string]SchedulerInterface)
	for name, scheduler := range sr.schedulers {
		schedulers[name] = scheduler
	}
	
	return schedulers
}

// StartAll starts all registered schedulers
func (sr *SchedulerRegistry) StartAll() error {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	
	for name, scheduler := range sr.schedulers {
		if err := scheduler.Start(); err != nil {
			return fmt.Errorf("failed to start scheduler %s: %w", name, err)
		}
	}
	
	return nil
}

// StopAll stops all registered schedulers
func (sr *SchedulerRegistry) StopAll() error {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	
	var errors []string
	for name, scheduler := range sr.schedulers {
		if err := scheduler.Stop(); err != nil {
			errors = append(errors, fmt.Sprintf("failed to stop scheduler %s: %v", name, err))
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("errors stopping schedulers: %v", errors)
	}
	
	return nil
}