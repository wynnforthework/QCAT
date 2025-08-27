package workflow

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/events"
	"qcat/internal/workflow"
)

// MultiStrategyWorkflowSystem 多策略工作流系统
type MultiStrategyWorkflowSystem struct {
	// 核心组件
	multiStrategyManager  *MultiStrategyManager
	evolutionManager      *EvolutionManager
	tradingWorkflowEngine *workflow.TradingWorkflowEngine
	strategyPool          *TradingStrategyPool
	factorSyncService     *FactorSyncService

	// 事件系统
	eventBus *events.EventBus

	// 配置
	config *SystemConfig

	// 运行状态
	ctx       context.Context
	cancel    context.CancelFunc
	isRunning bool
	runningMu sync.RWMutex
	wg        sync.WaitGroup

	// 统计信息
	stats   *SystemStats
	statsMu sync.RWMutex
}

// SystemConfig 系统配置
type SystemConfig struct {
	MultiStrategyManager *MultiStrategyManagerConfig     `yaml:"multi_strategy_manager"`
	StrategyWorkflow     *StrategyWorkflowConfig         `yaml:"strategy_workflow"`
	EvolutionManager     *EvolutionConfig                `yaml:"evolution_manager"`
	TradingWorkflow      *workflow.TradingWorkflowConfig `yaml:"trading_workflow"`
	EventSystem          *events.EventBusConfig          `yaml:"event_system"`
	Monitoring           *MonitoringConfig               `yaml:"monitoring"`
}

// MonitoringConfig 监控配置
type MonitoringConfig struct {
	MetricsInterval     time.Duration `yaml:"metrics_interval"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`
	HealthCheckTimeout  time.Duration `yaml:"health_check_timeout"`
	LogLevel            string        `yaml:"log_level"`
	LogFormat           string        `yaml:"log_format"`
	LogFile             string        `yaml:"log_file"`
	EnableProfiling     bool          `yaml:"enable_profiling"`
	ProfilingPort       int           `yaml:"profiling_port"`
}

// SystemStats 系统统计信息
type SystemStats struct {
	// 系统运行时间
	StartTime time.Time
	Uptime    time.Duration

	// 组件状态
	ComponentsRunning int
	ComponentsTotal   int

	// 策略统计
	TotalStrategies    int64
	ActiveStrategies   int64
	EnabledStrategies  int64
	DisabledStrategies int64

	// 执行统计
	TotalExecutions      int64
	SuccessfulExecutions int64
	FailedExecutions     int64

	// 资源统计
	CPUUsage    float64
	MemoryUsage int64

	// 最后更新时间
	LastUpdateTime time.Time
}

// NewMultiStrategyWorkflowSystem 创建多策略工作流系统
func NewMultiStrategyWorkflowSystem(config *SystemConfig) (*MultiStrategyWorkflowSystem, error) {
	if config == nil {
		config = GetDefaultSystemConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 创建事件总线
	eventBus := events.NewEventBus(config.EventSystem)

	// 创建多策略管理器
	multiStrategyManager := NewMultiStrategyManager(config.MultiStrategyManager)

	// 创建进化管理器
	evolutionManager := NewEvolutionManager(config.EvolutionManager, eventBus)

	// 创建策略池
	strategyPool := NewTradingStrategyPool(multiStrategyManager, eventBus)

	// 创建因子同步服务
	factorSyncService := NewFactorSyncService(eventBus, nil)

	// 创建交易工作流引擎
	tradingWorkflowEngine := workflow.NewTradingWorkflowEngine(strategyPool, config.TradingWorkflow)

	system := &MultiStrategyWorkflowSystem{
		multiStrategyManager:  multiStrategyManager,
		evolutionManager:      evolutionManager,
		tradingWorkflowEngine: tradingWorkflowEngine,
		strategyPool:          strategyPool,
		factorSyncService:     factorSyncService,
		eventBus:              eventBus,
		config:                config,
		ctx:                   ctx,
		cancel:                cancel,
		stats: &SystemStats{
			StartTime:       time.Now(),
			ComponentsTotal: 5, // 五个主要组件
			LastUpdateTime:  time.Now(),
		},
	}

	return system, nil
}

// Start 启动系统
func (msws *MultiStrategyWorkflowSystem) Start() error {
	msws.runningMu.Lock()
	defer msws.runningMu.Unlock()

	if msws.isRunning {
		return fmt.Errorf("multi-strategy workflow system is already running")
	}

	log.Println("启动多策略工作流系统...")

	// 事件总线已在创建时启动，无需额外启动
	msws.stats.ComponentsRunning++

	// 启动多策略管理器
	if err := msws.multiStrategyManager.Start(); err != nil {
		return fmt.Errorf("failed to start multi-strategy manager: %w", err)
	}
	msws.stats.ComponentsRunning++

	// 启动进化管理器
	if err := msws.evolutionManager.Start(); err != nil {
		return fmt.Errorf("failed to start evolution manager: %w", err)
	}
	msws.stats.ComponentsRunning++

	// 启动因子同步服务
	if err := msws.factorSyncService.Start(); err != nil {
		return fmt.Errorf("failed to start factor sync service: %w", err)
	}
	msws.stats.ComponentsRunning++

	// 启动交易工作流引擎
	if err := msws.tradingWorkflowEngine.Start(); err != nil {
		return fmt.Errorf("failed to start trading workflow engine: %w", err)
	}
	msws.stats.ComponentsRunning++

	// 启动监控循环
	msws.wg.Add(2)
	go msws.runMonitoringLoop()
	go msws.runHealthCheckLoop()

	msws.isRunning = true
	msws.stats.StartTime = time.Now()

	log.Println("多策略工作流系统启动完成")

	// 发送系统启动事件
	msws.emitSystemEvent("system_started", map[string]interface{}{
		"components_running": msws.stats.ComponentsRunning,
		"start_time":         msws.stats.StartTime,
	})

	return nil
}

// Stop 停止系统
func (msws *MultiStrategyWorkflowSystem) Stop() error {
	msws.runningMu.Lock()
	defer msws.runningMu.Unlock()

	if !msws.isRunning {
		return nil
	}

	log.Println("停止多策略工作流系统...")

	// 取消上下文
	msws.cancel()

	// 等待监控循环结束
	msws.wg.Wait()

	// 停止各个组件
	components := []struct {
		name string
		stop func() error
	}{
		{"交易工作流引擎", msws.tradingWorkflowEngine.Stop},
		{"因子同步服务", msws.factorSyncService.Stop},
		{"进化管理器", msws.evolutionManager.Stop},
		{"多策略管理器", msws.multiStrategyManager.Stop},
	}

	for _, component := range components {
		if err := component.stop(); err != nil {
			log.Printf("Warning: failed to stop %s: %v", component.name, err)
		} else {
			msws.stats.ComponentsRunning--
		}
	}

	// 停止事件总线
	msws.eventBus.Stop()

	msws.isRunning = false

	log.Println("多策略工作流系统已停止")

	// 发送系统停止事件
	msws.emitSystemEvent("system_stopped", map[string]interface{}{
		"uptime": time.Since(msws.stats.StartTime).String(),
	})

	return nil
}

// CreateAndRunStrategy 创建并运行策略
func (msws *MultiStrategyWorkflowSystem) CreateAndRunStrategy(strategyName, strategyType string) (string, error) {
	if !msws.isRunning {
		return "", fmt.Errorf("system is not running")
	}

	// 创建策略工作流
	engine, err := msws.multiStrategyManager.CreateStrategyWorkflow(
		fmt.Sprintf("strategy_%d", time.Now().UnixNano()),
		strategyName,
		strategyType,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create strategy workflow: %w", err)
	}

	// 异步执行策略生命周期
	go func() {
		if err := engine.ExecuteLifecycle(); err != nil {
			log.Printf("Error executing strategy lifecycle for %s: %v", engine.StrategyID, err)
		}
	}()

	log.Printf("策略 %s (%s) 创建并启动成功", engine.StrategyID, strategyName)
	return engine.StrategyID, nil
}

// runMonitoringLoop 运行监控循环
func (msws *MultiStrategyWorkflowSystem) runMonitoringLoop() {
	defer msws.wg.Done()

	ticker := time.NewTicker(msws.config.Monitoring.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-msws.ctx.Done():
			return
		case <-ticker.C:
			msws.updateSystemStats()
		}
	}
}

// runHealthCheckLoop 运行健康检查循环
func (msws *MultiStrategyWorkflowSystem) runHealthCheckLoop() {
	defer msws.wg.Done()

	ticker := time.NewTicker(msws.config.Monitoring.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-msws.ctx.Done():
			return
		case <-ticker.C:
			msws.performHealthCheck()
		}
	}
}

// updateSystemStats 更新系统统计信息
func (msws *MultiStrategyWorkflowSystem) updateSystemStats() {
	msws.statsMu.Lock()
	defer msws.statsMu.Unlock()

	// 更新运行时间
	msws.stats.Uptime = time.Since(msws.stats.StartTime)

	// 获取多策略管理器统计
	if msws.multiStrategyManager != nil {
		multiStats := msws.multiStrategyManager.stats
		msws.stats.TotalStrategies = multiStats.TotalStrategies
		msws.stats.ActiveStrategies = multiStats.ActiveStrategies
		msws.stats.EnabledStrategies = multiStats.EnabledStrategies
		msws.stats.DisabledStrategies = multiStats.DisabledStrategies
	}

	// 获取交易工作流统计
	if msws.tradingWorkflowEngine != nil {
		tradingStats := msws.tradingWorkflowEngine.GetStats()
		msws.stats.TotalExecutions = tradingStats.TotalExecutions
		msws.stats.SuccessfulExecutions = tradingStats.SuccessfulExecutions
		msws.stats.FailedExecutions = tradingStats.FailedExecutions
	}

	msws.stats.LastUpdateTime = time.Now()
}

// performHealthCheck 执行健康检查
func (msws *MultiStrategyWorkflowSystem) performHealthCheck() {
	healthStatus := make(map[string]bool)

	// 检查各组件健康状态
	healthStatus["multi_strategy_manager"] = msws.multiStrategyManager != nil && msws.multiStrategyManager.isRunning
	healthStatus["evolution_manager"] = msws.evolutionManager != nil && msws.evolutionManager.isRunning
	healthStatus["trading_workflow_engine"] = msws.tradingWorkflowEngine != nil && msws.tradingWorkflowEngine.IsRunning()
	healthStatus["event_bus"] = msws.eventBus != nil

	// 统计健康组件数量
	healthyComponents := 0
	for _, healthy := range healthStatus {
		if healthy {
			healthyComponents++
		}
	}

	// 发送健康检查事件
	msws.emitSystemEvent("health_check", map[string]interface{}{
		"healthy_components": healthyComponents,
		"total_components":   len(healthStatus),
		"component_status":   healthStatus,
		"timestamp":          time.Now(),
	})

	if healthyComponents < len(healthStatus) {
		log.Printf("Warning: %d/%d components are healthy", healthyComponents, len(healthStatus))
	}
}

// emitSystemEvent 发送系统事件
func (msws *MultiStrategyWorkflowSystem) emitSystemEvent(eventType string, data map[string]interface{}) {
	if msws.eventBus == nil {
		return
	}

	event := &events.Event{
		Type:      events.EventType(eventType),
		Source:    "multi_strategy_workflow_system",
		Data:      data,
		Timestamp: time.Now(),
	}

	if err := msws.eventBus.Publish(event); err != nil {
		log.Printf("Warning: failed to emit system event %s: %v", eventType, err)
	}
}

// GetSystemStats 获取系统统计信息
func (msws *MultiStrategyWorkflowSystem) GetSystemStats() *SystemStats {
	msws.statsMu.RLock()
	defer msws.statsMu.RUnlock()

	// 返回副本
	stats := *msws.stats
	return &stats
}

// IsRunning 检查系统是否正在运行
func (msws *MultiStrategyWorkflowSystem) IsRunning() bool {
	msws.runningMu.RLock()
	defer msws.runningMu.RUnlock()
	return msws.isRunning
}

// GetDefaultSystemConfig 获取默认系统配置
func GetDefaultSystemConfig() *SystemConfig {
	return &SystemConfig{
		MultiStrategyManager: GetDefaultMultiStrategyConfig(),
		StrategyWorkflow:     GetDefaultWorkflowConfig(),
		EvolutionManager:     GetDefaultEvolutionConfig(),
		TradingWorkflow:      workflow.GetDefaultTradingWorkflowConfig(),
		EventSystem: &events.EventBusConfig{
			BufferSize: 2000,
			MaxRetries: 3,
			RetryDelay: time.Second,
		},
		Monitoring: &MonitoringConfig{
			MetricsInterval:     30 * time.Second,
			HealthCheckInterval: 1 * time.Minute,
			HealthCheckTimeout:  10 * time.Second,
			LogLevel:            "info",
			LogFormat:           "json",
			LogFile:             "logs/multi_strategy_workflow.log",
			EnableProfiling:     false,
			ProfilingPort:       6060,
		},
	}
}
