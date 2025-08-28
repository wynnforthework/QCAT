# Automation Scheduler Shared Components

This package provides the core infrastructure and shared components for all automation schedulers in the QCAT system. It implements the foundation required for the 18 TODO items in the automation scheduler implementation.

## Overview

The shared components provide:

- **Common data models and interfaces** for all schedulers
- **Error handling and recovery mechanisms** with retry logic and circuit breakers
- **Configuration management** for automation parameters
- **Base test framework and utilities** for comprehensive testing
- **Factory pattern** for creating scheduler instances
- **Metrics collection and monitoring** capabilities

## Components

### 1. Data Models (`models.go`)

Defines common data structures used across all schedulers:

- **Error types**: `AutomationError`, `ErrorSeverity`, `RiskLevel`, `Severity`
- **Market data**: `Position`, `MarketRegime`, `MarketConditions`
- **Risk management**: `PositionRisk`, `RiskParams`, `Anomaly`
- **Performance metrics**: `PerformanceMetrics`, `RiskMetrics`
- **System monitoring**: `ResourceUsage`, `SystemPerformance`, `SecurityThreat`

### 2. Error Handling (`errors.go`)

Comprehensive error handling system:

```go
// Create retry strategy
retryStrategy := NewRetryStrategy(3, time.Second, time.Minute, 2.0)

// Create circuit breaker
circuitBreaker := NewCircuitBreaker(CircuitBreakerConfig{
    FailureThreshold: 5,
    RecoveryTimeout:  time.Minute * 5,
    HalfOpenRequests: 3,
    SuccessThreshold: 2,
})

// Create error handler
errorHandler := NewErrorHandler(retryStrategy, circuitBreaker)

// Handle operations with automatic retry and circuit breaking
err := errorHandler.Handle(ctx, originalError, func() error {
    return performOperation()
})
```

### 3. Configuration Management (`config.go`)

Structured configuration for all automation components:

```go
// Create config manager
configManager := NewConfigManager()

// Load configuration
err := configManager.LoadConfig(configMap)

// Get configuration values
riskEnabled := configManager.Get("risk_management.enabled")
checkInterval := configManager.Get("risk_management.risk_monitoring.check_interval")
```

### 4. Testing Framework (`testing.go`)

Comprehensive testing utilities:

```go
// Create test framework
tf := NewTestFramework(t)

// Create mock database
mockDB := NewMockDatabase()
tf.SetMock("database", mockDB)

// Create mock exchange API
mockAPI := NewMockExchangeAPI()
mockAPI.SetPositions(testPositions)

// Generate test data
tdg := NewTestDataGenerator(12345)
positions := tdg.GeneratePositions(10)
marketData := tdg.GenerateMarketData("BTCUSDT")

// Performance testing
pth := NewPerformanceTestHelper(t)
pth.AssertExecutionTime(func() {
    // Test operation
}, time.Second)

// Concurrency testing
cth := NewConcurrencyTestHelper(t)
cth.TestConcurrentExecution(func() {
    // Concurrent operation
}, 10, 100)
```

### 5. Utility Functions (`utils.go`)

Mathematical and statistical utilities:

```go
// Statistical calculations
mean := CalculateMean(values)
stdDev := CalculateStandardDeviation(values)
correlation := CalculateCorrelation(x, y)
sharpeRatio := CalculateSharpeRatio(returns, riskFreeRate)

// Risk calculations
atr := CalculateATR(highs, lows, closes, 14)
volatility := CalculateRealizedVolatility(prices, 20)
var := CalculateVaR(returns, 0.95)
maxDrawdown := CalculateMaxDrawdown(equityCurve)

// Data processing
normalized := NormalizeValues(values)
outliers := DetectOutliers(values, 1.5)
interpolated := InterpolateLinear(values, missingIndices)
```

### 6. Interfaces (`interfaces.go`)

Defines interfaces for all scheduler types:

- `SchedulerInterface`: Base interface for all schedulers
- `RiskSchedulerInterface`: Risk management schedulers
- `PositionSchedulerInterface`: Position management schedulers
- `DataSchedulerInterface`: Data processing schedulers
- `SystemSchedulerInterface`: System monitoring schedulers
- `LearningSchedulerInterface`: Machine learning schedulers

### 7. Factory and Registry (`factory.go`)

Factory pattern for creating and managing schedulers:

```go
// Create scheduler factory
factory := NewSchedulerFactory(config, metricsCollector, eventPublisher)

// Create base scheduler
baseScheduler := factory.CreateBaseScheduler("RiskScheduler", "risk_management")

// Create scheduler registry
registry := NewSchedulerRegistry()

// Register schedulers
registry.Register("risk_scheduler", riskScheduler)
registry.Register("position_scheduler", positionScheduler)

// Start all schedulers
err := registry.StartAll()
```

## Usage Examples

### Creating a Custom Scheduler

```go
type MyCustomScheduler struct {
    *BaseSchedulerImpl
    // Custom fields
}

func NewMyCustomScheduler(factory *SchedulerFactory) *MyCustomScheduler {
    base := factory.CreateBaseScheduler("MyScheduler", "custom")
    base.SetVersion("1.0.0")
    base.SetDescription("Custom automation scheduler")
    base.SetSupportedTasks([]string{"custom_task_1", "custom_task_2"})
    
    return &MyCustomScheduler{
        BaseSchedulerImpl: base,
    }
}

func (mcs *MyCustomScheduler) executeTask(ctx context.Context, task interface{}) error {
    // Implement custom task execution logic
    return nil
}

func (mcs *MyCustomScheduler) CanExecute(task interface{}) bool {
    // Implement task compatibility check
    return true
}
```

### Error Handling with Context

```go
err := NewAutomationError(
    ErrCodeDatabaseConnection,
    "Failed to connect to database",
    "DatabaseService",
    ErrorSeverityHigh,
    true,
).WithContext("host", "localhost").WithContext("port", 5432)

// Register error callback
errorHandler.RegisterErrorCallback(ErrCodeDatabaseConnection, func(err error) {
    // Handle database connection errors
    log.Printf("Database connection error: %v", err)
})
```

### Configuration Usage

```go
// Risk management configuration
if config.RiskManagement.Enabled {
    checkInterval := config.RiskManagement.RiskMonitoring.CheckInterval
    marginThreshold := config.RiskManagement.RiskMonitoring.Thresholds.MarginRatio
    
    // Use configuration values
}

// Position management configuration
if config.PositionManagement.Enabled {
    optimizationFreq := config.PositionManagement.PositionOptimization.Frequency
    maxLeverage := config.PositionManagement.PositionOptimization.Constraints.MaxLeverage
    
    // Use configuration values
}
```

## Testing

The package includes comprehensive tests covering:

- Unit tests for all utility functions
- Integration tests for error handling
- Performance benchmarks
- Concurrency tests
- Mock implementations for external dependencies

Run tests with:

```bash
go test ./internal/automation/scheduler/shared/...
```

Run benchmarks with:

```bash
go test -bench=. ./internal/automation/scheduler/shared/...
```

## Performance Considerations

- All statistical calculations are optimized for performance
- Circuit breakers prevent cascading failures
- Connection pooling is supported through interfaces
- Memory usage is monitored and controlled
- Concurrent operations are thread-safe

## Security Features

- Input validation and sanitization
- Parameterized database queries (through interfaces)
- Secure error handling (no sensitive data in error messages)
- Audit logging capabilities
- Rate limiting support

## Monitoring and Metrics

The shared components provide built-in monitoring:

- Task execution metrics (count, duration, success rate)
- Error metrics (by type and severity)
- Performance metrics (throughput, latency)
- Health status monitoring
- Resource usage tracking

## Configuration Schema

The configuration supports the following main sections:

- `risk_management`: Risk monitoring and control settings
- `position_management`: Position optimization and allocation settings
- `data_processing`: Data cleaning and analysis settings
- `system_monitoring`: System health and security settings
- `machine_learning`: ML pipeline and AutoML settings
- `common`: Shared settings (error handling, performance, logging)

## Dependencies

The shared components have minimal external dependencies:

- `github.com/stretchr/testify` for testing utilities
- `github.com/lib/pq` for PostgreSQL support (interface only)
- Standard Go libraries for core functionality

## Future Enhancements

Planned improvements include:

- Distributed scheduler coordination
- Advanced ML model management
- Real-time streaming data support
- Enhanced security features
- Performance optimizations
- Additional statistical functions

## Contributing

When adding new shared components:

1. Follow the existing patterns and interfaces
2. Add comprehensive tests
3. Update documentation
4. Ensure thread safety
5. Consider performance implications
6. Add appropriate error handling

## License

This code is part of the QCAT quantitative trading system and follows the project's licensing terms.