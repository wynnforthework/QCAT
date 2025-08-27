package workflow

import (
	"qcat/internal/workflow/interfaces"
)

// WorkflowEngineFactory 工作流引擎工厂
type WorkflowEngineFactory struct{}

// NewWorkflowEngineFactory 创建工作流引擎工厂
func NewWorkflowEngineFactory() *WorkflowEngineFactory {
	return &WorkflowEngineFactory{}
}

// CreateTradingWorkflowEngine 创建交易工作流引擎
func (f *WorkflowEngineFactory) CreateTradingWorkflowEngine(
	strategyPool interfaces.StrategyPoolInterface, 
	config *interfaces.TradingWorkflowConfig,
) interfaces.WorkflowEngineInterface {
	return NewTradingWorkflowEngine(strategyPool, config)
}

// CreateEnhancedWorkflowEngine 创建增强工作流引擎
func (f *WorkflowEngineFactory) CreateEnhancedWorkflowEngine(maxConcurrency int) *EnhancedWorkflowEngine {
	return NewEnhancedWorkflowEngine(maxConcurrency)
}

// CreateWorkflowEngine 创建基础工作流引擎
func (f *WorkflowEngineFactory) CreateWorkflowEngine(maxConcurrency int) *WorkflowEngine {
	return NewWorkflowEngine(maxConcurrency)
}

// 全局工厂实例
var GlobalWorkflowEngineFactory = NewWorkflowEngineFactory()

// CreateTradingWorkflowEngineGlobal 全局函数创建交易工作流引擎
func CreateTradingWorkflowEngineGlobal(
	strategyPool interfaces.StrategyPoolInterface, 
	config *interfaces.TradingWorkflowConfig,
) interfaces.WorkflowEngineInterface {
	return GlobalWorkflowEngineFactory.CreateTradingWorkflowEngine(strategyPool, config)
}
