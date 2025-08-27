package interfaces

import (
	"context"
	"time"
)

// StrategyPoolInterface 策略池接口
// 用于打破循环依赖
type StrategyPoolInterface interface {
	// 生命周期管理
	Start() error
	Stop() error

	// 获取活跃策略数量
	GetActiveStrategyCount() int

	// 获取策略状态
	GetStrategyStatus(strategyID string) (string, error)

	// 获取所有活跃策略ID
	GetActiveStrategyIDs() []string

	// 获取启用的策略
	GetEnabledStrategies() []string

	// 检查策略是否存在
	HasStrategy(strategyID string) bool

	// 获取策略统计信息
	GetStrategyStats(strategyID string) (map[string]interface{}, error)
}

// WorkflowEngineInterface 工作流引擎接口
type WorkflowEngineInterface interface {
	// 启动工作流引擎
	Start(ctx context.Context) error

	// 停止工作流引擎
	Stop() error

	// 获取运行状态
	IsRunning() bool

	// 执行工作流
	ExecuteWorkflow(ctx context.Context) error

	// 获取统计信息
	GetStats() map[string]interface{}
}

// TradingWorkflowConfig 交易工作流配置
type TradingWorkflowConfig struct {
	// 最大并发数
	MaxConcurrency int `yaml:"max_concurrency"`

	// 执行间隔
	ExecutionInterval time.Duration `yaml:"execution_interval"`

	// 策略检查间隔
	StrategyCheckInterval time.Duration `yaml:"strategy_check_interval"`

	// 超时设置
	ExecutionTimeout time.Duration `yaml:"execution_timeout"`

	// 重试配置
	MaxRetries int           `yaml:"max_retries"`
	RetryDelay time.Duration `yaml:"retry_delay"`

	// 启用的功能
	EnabledFunctions []int `yaml:"enabled_functions"`
}

// GetDefaultTradingWorkflowConfig 获取默认交易工作流配置
func GetDefaultTradingWorkflowConfig() *TradingWorkflowConfig {
	return &TradingWorkflowConfig{
		MaxConcurrency:        5,
		ExecutionInterval:     30 * time.Second,
		StrategyCheckInterval: 60 * time.Second,
		ExecutionTimeout:      300 * time.Second,
		MaxRetries:            3,
		RetryDelay:            10 * time.Second,
		EnabledFunctions: []int{
			18, 21, 23, // 基础数据层
			26, 12, 13, // 监控分析层
			15, 3, 16, 17, // 仓位管理层
			4, 5, 9, // 交易执行层
			11,     // 收益优化层
			14, 22, // 安全保障层
		},
	}
}

// TradingWorkflowStats 交易工作流统计信息
type TradingWorkflowStats struct {
	// 执行统计
	TotalExecutions      int64         `json:"total_executions"`
	SuccessfulExecutions int64         `json:"successful_executions"`
	FailedExecutions     int64         `json:"failed_executions"`
	AverageExecutionTime time.Duration `json:"average_execution_time"`

	// 策略统计
	ActiveStrategies int `json:"active_strategies"`
	TotalStrategies  int `json:"total_strategies"`

	// 时间统计
	StartTime         time.Time     `json:"start_time"`
	LastExecutionTime time.Time     `json:"last_execution_time"`
	Uptime            time.Duration `json:"uptime"`
}

// SystemConfig 系统配置接口
type SystemConfig interface {
	// 获取交易工作流配置
	GetTradingWorkflowConfig() *TradingWorkflowConfig

	// 获取事件系统配置
	GetEventSystemConfig() map[string]interface{}

	// 获取多策略管理器配置
	GetMultiStrategyManagerConfig() map[string]interface{}

	// 获取进化管理器配置
	GetEvolutionManagerConfig() map[string]interface{}
}

// EventBusInterface 事件总线接口
type EventBusInterface interface {
	// 发布事件
	Publish(topic string, data interface{}) error

	// 订阅事件
	Subscribe(topic string, handler func(interface{})) error

	// 取消订阅
	Unsubscribe(topic string) error

	// 获取统计信息
	GetStats() map[string]interface{}
}
