package concurrent

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

// GoroutinePoolConfig Goroutine池配置管理器
type GoroutinePoolConfig struct {
	DefaultPool        PoolSettings         `yaml:"default_pool"`
	HighPriorityPool   PoolSettings         `yaml:"high_priority_pool"`
	LowPriorityPool    PoolSettings         `yaml:"low_priority_pool"`
	StrategyPool       PoolSettings         `yaml:"strategy_pool"`
	DataProcessingPool PoolSettings         `yaml:"data_processing_pool"`
	LoadBalancer       LoadBalancerConfig   `yaml:"load_balancer"`
	Monitor            PoolMonitorConfig    `yaml:"monitor"`
	TaskQueue          TaskQueueConfig      `yaml:"task_queue"`
	AutoScaling        AutoScalingConfig    `yaml:"auto_scaling"`
	Performance        PerformanceConfig    `yaml:"performance"`
	Logging            LoggingConfig        `yaml:"logging"`
	FaultTolerance     FaultToleranceConfig `yaml:"fault_tolerance"`
	ResourceLimits     ResourceLimitsConfig `yaml:"resource_limits"`
}

// PoolSettings 池设置
type PoolSettings struct {
	MaxWorkers int  `yaml:"max_workers"`
	QueueSize  int  `yaml:"queue_size"`
	Enabled    bool `yaml:"enabled"`
}

// LoadBalancerConfig 负载均衡配置
type LoadBalancerConfig struct {
	Strategy string `yaml:"strategy"`
	Enabled  bool   `yaml:"enabled"`
}

// PoolMonitorConfig 监控配置
type PoolMonitorConfig struct {
	Enabled         bool                  `yaml:"enabled"`
	Interval        int                   `yaml:"interval"`
	AlertThresholds AlertThresholdsConfig `yaml:"alert_thresholds"`
}

// AlertThresholdsConfig 告警阈值配置
type AlertThresholdsConfig struct {
	MaxCPUUsage       float64 `yaml:"max_cpu_usage"`
	MaxMemoryUsage    float64 `yaml:"max_memory_usage"`
	MaxGoroutineCount int     `yaml:"max_goroutine_count"`
	MaxQueueLength    int     `yaml:"max_queue_length"`
	MinHealthScore    float64 `yaml:"min_health_score"`
}

// TaskQueueConfig 任务队列配置
type TaskQueueConfig struct {
	MaxSize int  `yaml:"max_size"`
	Enabled bool `yaml:"enabled"`
}

// AutoScalingConfig 自动扩缩容配置
type AutoScalingConfig struct {
	Enabled            bool `yaml:"enabled"`
	ScaleUpThreshold   int  `yaml:"scale_up_threshold"`
	ScaleDownThreshold int  `yaml:"scale_down_threshold"`
	MinWorkers         int  `yaml:"min_workers"`
	MaxWorkers         int  `yaml:"max_workers"`
	ScaleInterval      int  `yaml:"scale_interval"`
}

// PerformanceConfig 性能配置
type PerformanceConfig struct {
	WorkerIdleTimeout int `yaml:"worker_idle_timeout"`
	TaskTimeout       int `yaml:"task_timeout"`
	BatchSize         int `yaml:"batch_size"`
	PrefetchCount     int `yaml:"prefetch_count"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Enabled                   bool   `yaml:"enabled"`
	Level                     string `yaml:"level"`
	IncludeTaskDetails        bool   `yaml:"include_task_details"`
	IncludePerformanceMetrics bool   `yaml:"include_performance_metrics"`
}

// FaultToleranceConfig 故障恢复配置
type FaultToleranceConfig struct {
	Enabled        bool                 `yaml:"enabled"`
	MaxRetries     int                  `yaml:"max_retries"`
	RetryDelay     int                  `yaml:"retry_delay"`
	CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker"`
}

// CircuitBreakerConfig 熔断器配置
type CircuitBreakerConfig struct {
	Enabled          bool `yaml:"enabled"`
	FailureThreshold int  `yaml:"failure_threshold"`
	RecoveryTimeout  int  `yaml:"recovery_timeout"`
}

// ResourceLimitsConfig 资源限制配置
type ResourceLimitsConfig struct {
	MaxMemoryPerWorker string  `yaml:"max_memory_per_worker"`
	MaxCPUPerWorker    float64 `yaml:"max_cpu_per_worker"`
	MaxTotalMemory     string  `yaml:"max_total_memory"`
	MaxTotalCPU        float64 `yaml:"max_total_cpu"`
}

// LoadGoroutinePoolConfig 加载Goroutine池配置
func LoadGoroutinePoolConfig(configPath string) (*GoroutinePoolConfig, error) {
	// 如果没有指定配置文件路径，使用默认路径
	if configPath == "" {
		configPath = "configs/goroutine_pool.yaml"
	}

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	// 解析YAML配置
	var config GoroutinePoolConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	// 验证配置
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// validateConfig 验证配置
func validateConfig(config *GoroutinePoolConfig) error {
	// 验证池配置
	if err := validatePoolSettings("default_pool", &config.DefaultPool); err != nil {
		return err
	}
	if err := validatePoolSettings("high_priority_pool", &config.HighPriorityPool); err != nil {
		return err
	}
	if err := validatePoolSettings("low_priority_pool", &config.LowPriorityPool); err != nil {
		return err
	}
	if err := validatePoolSettings("strategy_pool", &config.StrategyPool); err != nil {
		return err
	}
	if err := validatePoolSettings("data_processing_pool", &config.DataProcessingPool); err != nil {
		return err
	}

	// 验证负载均衡策略
	validStrategies := map[string]bool{
		"round_robin":          true,
		"least_connections":    true,
		"weighted_round_robin": true,
	}
	if !validStrategies[config.LoadBalancer.Strategy] {
		return fmt.Errorf("invalid load balancer strategy: %s", config.LoadBalancer.Strategy)
	}

	// 验证监控配置
	if config.Monitor.Interval <= 0 {
		return fmt.Errorf("monitor interval must be positive")
	}

	// 验证任务队列配置
	if config.TaskQueue.MaxSize <= 0 {
		return fmt.Errorf("task queue max size must be positive")
	}

	// 验证自动扩缩容配置
	if config.AutoScaling.MinWorkers <= 0 {
		return fmt.Errorf("auto scaling min workers must be positive")
	}
	if config.AutoScaling.MaxWorkers < config.AutoScaling.MinWorkers {
		return fmt.Errorf("auto scaling max workers must be >= min workers")
	}

	return nil
}

// validatePoolSettings 验证池设置
func validatePoolSettings(poolName string, settings *PoolSettings) error {
	if settings.MaxWorkers <= 0 {
		return fmt.Errorf("%s max_workers must be positive", poolName)
	}
	if settings.QueueSize <= 0 {
		return fmt.Errorf("%s queue_size must be positive", poolName)
	}
	return nil
}

// GetDefaultConfig 获取默认配置
func GetDefaultConfig() *GoroutinePoolConfig {
	return &GoroutinePoolConfig{
		DefaultPool: PoolSettings{
			MaxWorkers: 10,
			QueueSize:  100,
			Enabled:    true,
		},
		HighPriorityPool: PoolSettings{
			MaxWorkers: 5,
			QueueSize:  50,
			Enabled:    true,
		},
		LowPriorityPool: PoolSettings{
			MaxWorkers: 3,
			QueueSize:  200,
			Enabled:    true,
		},
		StrategyPool: PoolSettings{
			MaxWorkers: 8,
			QueueSize:  80,
			Enabled:    true,
		},
		DataProcessingPool: PoolSettings{
			MaxWorkers: 6,
			QueueSize:  120,
			Enabled:    true,
		},
		LoadBalancer: LoadBalancerConfig{
			Strategy: "least_connections",
			Enabled:  true,
		},
		Monitor: PoolMonitorConfig{
			Enabled:  true,
			Interval: 30,
			AlertThresholds: AlertThresholdsConfig{
				MaxCPUUsage:       80.0,
				MaxMemoryUsage:    85.0,
				MaxGoroutineCount: 1000,
				MaxQueueLength:    500,
				MinHealthScore:    60.0,
			},
		},
		TaskQueue: TaskQueueConfig{
			MaxSize: 500,
			Enabled: true,
		},
		AutoScaling: AutoScalingConfig{
			Enabled:            true,
			ScaleUpThreshold:   80,
			ScaleDownThreshold: 20,
			MinWorkers:         1,
			MaxWorkers:         50,
			ScaleInterval:      60,
		},
		Performance: PerformanceConfig{
			WorkerIdleTimeout: 300,
			TaskTimeout:       600,
			BatchSize:         10,
			PrefetchCount:     5,
		},
		Logging: LoggingConfig{
			Enabled:                   true,
			Level:                     "info",
			IncludeTaskDetails:        true,
			IncludePerformanceMetrics: true,
		},
		FaultTolerance: FaultToleranceConfig{
			Enabled:    true,
			MaxRetries: 3,
			RetryDelay: 5,
			CircuitBreaker: CircuitBreakerConfig{
				Enabled:          true,
				FailureThreshold: 5,
				RecoveryTimeout:  30,
			},
		},
		ResourceLimits: ResourceLimitsConfig{
			MaxMemoryPerWorker: "100MB",
			MaxCPUPerWorker:    0.5,
			MaxTotalMemory:     "1GB",
			MaxTotalCPU:        4.0,
		},
	}
}

// ToPoolConfig 转换为PoolConfig
func (ps *PoolSettings) ToPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxWorkers: ps.MaxWorkers,
		QueueSize:  ps.QueueSize,
	}
}

// ToMonitorInterval 转换监控间隔为Duration
func (pmc *PoolMonitorConfig) ToMonitorInterval() time.Duration {
	return time.Duration(pmc.Interval) * time.Second
}

// ToLoadBalanceStrategy 转换负载均衡策略
func (lbc *LoadBalancerConfig) ToLoadBalanceStrategy() LoadBalanceStrategy {
	switch lbc.Strategy {
	case "round_robin":
		return RoundRobin
	case "least_connections":
		return LeastConnections
	case "weighted_round_robin":
		return WeightedRoundRobin
	default:
		return LeastConnections
	}
}
