package workflow

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/concurrent"
	"qcat/internal/events"
)

// EnabledStrategy 已启用策略
type EnabledStrategy struct {
	ID             string
	Name           string
	Type           string
	Version        string
	EnabledAt      time.Time
	Performance    *PerformanceMetrics
	LastUpdated    time.Time
	IsActive       bool
	TradingEnabled bool
}

// DisabledStrategy 已禁用策略
type DisabledStrategy struct {
	ID          string
	Name        string
	Type        string
	Version     string
	DisabledAt  time.Time
	Reason      string
	Performance *PerformanceMetrics
}

// MultiStrategyManagerConfig 多策略管理器配置
type MultiStrategyManagerConfig struct {
	// 并发配置
	MaxConcurrentStrategies int `yaml:"max_concurrent_strategies"`
	MaxConcurrentJobs       int `yaml:"max_concurrent_jobs"`

	// 资源配置
	GlobalCPUQuota    float64 `yaml:"global_cpu_quota"`
	GlobalMemoryQuota int64   `yaml:"global_memory_quota"`

	// 策略池配置
	MinEnabledStrategies int `yaml:"min_enabled_strategies"`
	MaxEnabledStrategies int `yaml:"max_enabled_strategies"`

	// 性能阈值
	PerformanceThreshold float64 `yaml:"performance_threshold"`
	DisableThreshold     float64 `yaml:"disable_threshold"`
	EnableThreshold      float64 `yaml:"enable_threshold"`

	// 调度配置
	SchedulingInterval time.Duration `yaml:"scheduling_interval"`
	EvaluationInterval time.Duration `yaml:"evaluation_interval"`
	CleanupInterval    time.Duration `yaml:"cleanup_interval"`
}

// MultiStrategyManager 多策略并发管理器
type MultiStrategyManager struct {
	config *MultiStrategyManagerConfig

	// 策略工作流实例
	strategyEngines map[string]*StrategyWorkflowEngine
	enginesMu       sync.RWMutex

	// 全局资源管理
	globalResourceManager *GlobalResourceManager
	globalScheduler       *GlobalScheduler

	// 策略池管理
	enabledStrategies  map[string]*EnabledStrategy
	disabledStrategies map[string]*DisabledStrategy
	strategiesMu       sync.RWMutex

	// 事件系统
	eventBus *events.EventBus

	// 运行状态
	ctx       context.Context
	cancel    context.CancelFunc
	isRunning bool
	runningMu sync.RWMutex
	wg        sync.WaitGroup

	// 统计信息
	stats   *MultiStrategyStats
	statsMu sync.RWMutex
}

// MultiStrategyStats 多策略统计信息
type MultiStrategyStats struct {
	TotalStrategies     int64
	ActiveStrategies    int64
	EnabledStrategies   int64
	DisabledStrategies  int64
	TotalJobs           int64
	CompletedJobs       int64
	FailedJobs          int64
	AverageJobTime      time.Duration
	ResourceUtilization *ResourceUtilization
	LastUpdateTime      time.Time
}

// ResourceUtilization 资源利用率
type ResourceUtilization struct {
	CPUUsage      float64
	MemoryUsage   int64
	CPUPercent    float64
	MemoryPercent float64
}

// GlobalResourceManager 全局资源管理器
type GlobalResourceManager struct {
	totalCPU    float64
	totalMemory int64
	usedCPU     float64
	usedMemory  int64
	mu          sync.RWMutex

	// 资源分配记录
	allocations map[string]*ResourceAllocation
}

// ResourceAllocation 资源分配记录
type ResourceAllocation struct {
	StrategyID  string
	CPU         float64
	Memory      int64
	AllocatedAt time.Time
}

// GlobalScheduler 全局调度器
type GlobalScheduler struct {
	taskQueue  chan *ScheduledTask
	workerPool *concurrent.Pool
	isRunning  bool
	runningMu  sync.RWMutex
}

// ScheduledTask 调度任务
type ScheduledTask struct {
	ID         string
	StrategyID string
	Type       TaskType
	Priority   int
	CreatedAt  time.Time
	Executor   func(context.Context) error
}

// TaskType 任务类型
type TaskType int

const (
	TaskStrategyEvaluation TaskType = iota
	TaskStrategyCleanup
	TaskResourceRebalance
	TaskPerformanceUpdate
)

// NewMultiStrategyManager 创建多策略管理器
func NewMultiStrategyManager(config *MultiStrategyManagerConfig) *MultiStrategyManager {
	if config == nil {
		config = GetDefaultMultiStrategyConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 创建事件总线
	eventBus := events.NewEventBus(&events.EventBusConfig{
		BufferSize: 2000,
		MaxRetries: 3,
		RetryDelay: time.Second,
	})

	// 创建全局资源管理器
	globalResourceManager := &GlobalResourceManager{
		totalCPU:    config.GlobalCPUQuota,
		totalMemory: config.GlobalMemoryQuota,
		allocations: make(map[string]*ResourceAllocation),
	}

	// 创建全局调度器
	globalScheduler := &GlobalScheduler{
		taskQueue: make(chan *ScheduledTask, 1000),
		workerPool: concurrent.NewPool("global_scheduler", &concurrent.PoolConfig{
			MinWorkers:      2,
			MaxWorkers:      10,
			IdleTimeout:     5 * time.Minute,
			TaskTimeout:     30 * time.Minute,
			QueueSize:       500,
			EnableMetrics:   true,
			EnableProfiling: false,
		}),
	}

	manager := &MultiStrategyManager{
		config:                config,
		strategyEngines:       make(map[string]*StrategyWorkflowEngine),
		globalResourceManager: globalResourceManager,
		globalScheduler:       globalScheduler,
		enabledStrategies:     make(map[string]*EnabledStrategy),
		disabledStrategies:    make(map[string]*DisabledStrategy),
		eventBus:              eventBus,
		ctx:                   ctx,
		cancel:                cancel,
		stats: &MultiStrategyStats{
			ResourceUtilization: &ResourceUtilization{},
			LastUpdateTime:      time.Now(),
		},
	}

	return manager
}

// Start 启动多策略管理器
func (msm *MultiStrategyManager) Start() error {
	msm.runningMu.Lock()
	defer msm.runningMu.Unlock()

	if msm.isRunning {
		return fmt.Errorf("multi-strategy manager is already running")
	}

	log.Println("启动多策略并发管理器...")

	// 启动全局调度器
	if err := msm.startGlobalScheduler(); err != nil {
		return fmt.Errorf("failed to start global scheduler: %w", err)
	}

	// 启动后台任务
	msm.wg.Add(3)
	go msm.runSchedulingLoop()
	go msm.runEvaluationLoop()
	go msm.runCleanupLoop()

	msm.isRunning = true

	// 发送启动事件
	msm.emitEvent("multi_strategy_manager_started", map[string]interface{}{
		"max_concurrent_strategies": msm.config.MaxConcurrentStrategies,
		"max_concurrent_jobs":       msm.config.MaxConcurrentJobs,
	})

	log.Println("多策略并发管理器启动完成")
	return nil
}

// Stop 停止多策略管理器
func (msm *MultiStrategyManager) Stop() error {
	msm.runningMu.Lock()
	defer msm.runningMu.Unlock()

	if !msm.isRunning {
		return nil
	}

	log.Println("停止多策略并发管理器...")

	// 取消上下文
	msm.cancel()

	// 停止所有策略引擎
	msm.stopAllStrategyEngines()

	// 停止全局调度器
	msm.stopGlobalScheduler()

	// 等待后台任务完成
	msm.wg.Wait()

	msm.isRunning = false

	// 发送停止事件
	msm.emitEvent("multi_strategy_manager_stopped", map[string]interface{}{
		"total_strategies": len(msm.strategyEngines),
	})

	log.Println("多策略并发管理器已停止")
	return nil
}

// CreateStrategyWorkflow 创建策略工作流
func (msm *MultiStrategyManager) CreateStrategyWorkflow(strategyID, strategyName, strategyType string) (*StrategyWorkflowEngine, error) {
	msm.enginesMu.Lock()
	defer msm.enginesMu.Unlock()

	// 检查策略是否已存在
	if _, exists := msm.strategyEngines[strategyID]; exists {
		return nil, fmt.Errorf("strategy workflow %s already exists", strategyID)
	}

	// 检查并发限制
	if len(msm.strategyEngines) >= msm.config.MaxConcurrentStrategies {
		return nil, fmt.Errorf("maximum concurrent strategies limit reached: %d",
			msm.config.MaxConcurrentStrategies)
	}

	// 分配资源
	if err := msm.allocateResourcesForStrategy(strategyID); err != nil {
		return nil, fmt.Errorf("failed to allocate resources: %w", err)
	}

	// 创建策略工作流引擎
	config := GetDefaultWorkflowConfig()
	engine := NewStrategyWorkflowEngine(strategyID, strategyName, config)
	engine.StrategyType = strategyType

	// 启动引擎
	if err := engine.Start(); err != nil {
		msm.releaseResourcesForStrategy(strategyID)
		return nil, fmt.Errorf("failed to start strategy workflow engine: %w", err)
	}

	// 添加到管理器
	msm.strategyEngines[strategyID] = engine

	// 更新统计
	msm.updateStats()

	// 发送事件
	msm.emitEvent("strategy_workflow_created", map[string]interface{}{
		"strategy_id":   strategyID,
		"strategy_name": strategyName,
		"strategy_type": strategyType,
	})

	log.Printf("策略工作流 %s (%s) 创建成功", strategyID, strategyName)
	return engine, nil
}

// RemoveStrategyWorkflow 移除策略工作流
func (msm *MultiStrategyManager) RemoveStrategyWorkflow(strategyID string) error {
	msm.enginesMu.Lock()
	defer msm.enginesMu.Unlock()

	engine, exists := msm.strategyEngines[strategyID]
	if !exists {
		return fmt.Errorf("strategy workflow %s not found", strategyID)
	}

	// 停止引擎
	if err := engine.Stop(); err != nil {
		log.Printf("Warning: failed to stop strategy workflow engine %s: %v", strategyID, err)
	}

	// 释放资源
	msm.releaseResourcesForStrategy(strategyID)

	// 从管理器中移除
	delete(msm.strategyEngines, strategyID)

	// 更新统计
	msm.updateStats()

	// 发送事件
	msm.emitEvent("strategy_workflow_removed", map[string]interface{}{
		"strategy_id": strategyID,
	})

	log.Printf("策略工作流 %s 已移除", strategyID)
	return nil
}

// GetDefaultMultiStrategyConfig 获取默认多策略配置
func GetDefaultMultiStrategyConfig() *MultiStrategyManagerConfig {
	return &MultiStrategyManagerConfig{
		MaxConcurrentStrategies: 10,
		MaxConcurrentJobs:       50,
		GlobalCPUQuota:          20.0,
		GlobalMemoryQuota:       40 * 1024 * 1024 * 1024, // 40GB
		MinEnabledStrategies:    3,
		MaxEnabledStrategies:    15,
		PerformanceThreshold:    0.1,
		DisableThreshold:        -0.2,
		EnableThreshold:         0.15,
		SchedulingInterval:      30 * time.Second,
		EvaluationInterval:      5 * time.Minute,
		CleanupInterval:         30 * time.Minute,
	}
}

// startGlobalScheduler 启动全局调度器
func (msm *MultiStrategyManager) startGlobalScheduler() error {
	msm.globalScheduler.runningMu.Lock()
	defer msm.globalScheduler.runningMu.Unlock()

	if msm.globalScheduler.isRunning {
		return nil
	}

	// 启动工作池
	msm.globalScheduler.workerPool.Start()

	// 启动任务处理循环
	go msm.runTaskProcessingLoop()

	msm.globalScheduler.isRunning = true
	log.Println("全局调度器已启动")

	return nil
}

// stopGlobalScheduler 停止全局调度器
func (msm *MultiStrategyManager) stopGlobalScheduler() {
	msm.globalScheduler.runningMu.Lock()
	defer msm.globalScheduler.runningMu.Unlock()

	if !msm.globalScheduler.isRunning {
		return
	}

	// 关闭任务队列
	close(msm.globalScheduler.taskQueue)

	// 停止工作池
	msm.globalScheduler.workerPool.Stop()

	msm.globalScheduler.isRunning = false
	log.Println("全局调度器已停止")
}

// runTaskProcessingLoop 运行任务处理循环
func (msm *MultiStrategyManager) runTaskProcessingLoop() {
	for task := range msm.globalScheduler.taskQueue {
		// 创建自动化任务
		automationTask := concurrent.NewAutomationTask(
			task.ID,
			fmt.Sprintf("ScheduledTask_%s", task.Type),
			task.Priority,
			30*time.Minute, // 默认超时
			task.Executor,
		)

		// 提交任务到工作池
		if err := msm.globalScheduler.workerPool.Submit(automationTask); err != nil {
			log.Printf("Warning: failed to submit task %s: %v", task.ID, err)
		}
	}
}

// runSchedulingLoop 运行调度循环
func (msm *MultiStrategyManager) runSchedulingLoop() {
	defer msm.wg.Done()

	ticker := time.NewTicker(msm.config.SchedulingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-msm.ctx.Done():
			return
		case <-ticker.C:
			msm.performScheduling()
		}
	}
}

// runEvaluationLoop 运行评估循环
func (msm *MultiStrategyManager) runEvaluationLoop() {
	defer msm.wg.Done()

	ticker := time.NewTicker(msm.config.EvaluationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-msm.ctx.Done():
			return
		case <-ticker.C:
			msm.performEvaluation()
		}
	}
}

// runCleanupLoop 运行清理循环
func (msm *MultiStrategyManager) runCleanupLoop() {
	defer msm.wg.Done()

	ticker := time.NewTicker(msm.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-msm.ctx.Done():
			return
		case <-ticker.C:
			msm.performCleanup()
		}
	}
}

// performScheduling 执行调度
func (msm *MultiStrategyManager) performScheduling() {
	// 检查资源使用情况
	msm.rebalanceResources()

	// 检查策略工作流状态
	msm.checkStrategyWorkflowHealth()

	// 更新统计信息
	msm.updateStats()
}

// performEvaluation 执行评估
func (msm *MultiStrategyManager) performEvaluation() {
	msm.enginesMu.RLock()
	engines := make([]*StrategyWorkflowEngine, 0, len(msm.strategyEngines))
	for _, engine := range msm.strategyEngines {
		engines = append(engines, engine)
	}
	msm.enginesMu.RUnlock()

	// 评估每个策略的性能
	for _, engine := range engines {
		msm.evaluateStrategyPerformance(engine)
	}

	// 执行策略启用/禁用决策
	msm.makeStrategyDecisions()
}

// performCleanup 执行清理
func (msm *MultiStrategyManager) performCleanup() {
	// 清理已完成的策略工作流
	msm.cleanupCompletedWorkflows()

	// 清理过期的禁用策略
	msm.cleanupExpiredDisabledStrategies()

	// 清理资源分配记录
	msm.cleanupResourceAllocations()
}

// allocateResourcesForStrategy 为策略分配资源
func (msm *MultiStrategyManager) allocateResourcesForStrategy(strategyID string) error {
	msm.globalResourceManager.mu.Lock()
	defer msm.globalResourceManager.mu.Unlock()

	// 计算每个策略的资源配额
	cpuPerStrategy := msm.globalResourceManager.totalCPU / float64(msm.config.MaxConcurrentStrategies)
	memoryPerStrategy := msm.globalResourceManager.totalMemory / int64(msm.config.MaxConcurrentStrategies)

	// 检查资源是否足够
	if msm.globalResourceManager.usedCPU+cpuPerStrategy > msm.globalResourceManager.totalCPU {
		return fmt.Errorf("insufficient CPU resources")
	}
	if msm.globalResourceManager.usedMemory+memoryPerStrategy > msm.globalResourceManager.totalMemory {
		return fmt.Errorf("insufficient memory resources")
	}

	// 分配资源
	allocation := &ResourceAllocation{
		StrategyID:  strategyID,
		CPU:         cpuPerStrategy,
		Memory:      memoryPerStrategy,
		AllocatedAt: time.Now(),
	}

	msm.globalResourceManager.allocations[strategyID] = allocation
	msm.globalResourceManager.usedCPU += cpuPerStrategy
	msm.globalResourceManager.usedMemory += memoryPerStrategy

	log.Printf("为策略 %s 分配资源: CPU=%.2f, Memory=%d", strategyID, cpuPerStrategy, memoryPerStrategy)
	return nil
}

// releaseResourcesForStrategy 释放策略资源
func (msm *MultiStrategyManager) releaseResourcesForStrategy(strategyID string) {
	msm.globalResourceManager.mu.Lock()
	defer msm.globalResourceManager.mu.Unlock()

	allocation, exists := msm.globalResourceManager.allocations[strategyID]
	if !exists {
		return
	}

	// 释放资源
	msm.globalResourceManager.usedCPU -= allocation.CPU
	msm.globalResourceManager.usedMemory -= allocation.Memory

	// 删除分配记录
	delete(msm.globalResourceManager.allocations, strategyID)

	log.Printf("释放策略 %s 的资源: CPU=%.2f, Memory=%d", strategyID, allocation.CPU, allocation.Memory)
}

// rebalanceResources 重新平衡资源
func (msm *MultiStrategyManager) rebalanceResources() {
	msm.globalResourceManager.mu.RLock()
	cpuUsagePercent := (msm.globalResourceManager.usedCPU / msm.globalResourceManager.totalCPU) * 100
	memoryUsagePercent := float64(msm.globalResourceManager.usedMemory) / float64(msm.globalResourceManager.totalMemory) * 100
	msm.globalResourceManager.mu.RUnlock()

	// 更新资源利用率统计
	msm.statsMu.Lock()
	msm.stats.ResourceUtilization.CPUPercent = cpuUsagePercent
	msm.stats.ResourceUtilization.MemoryPercent = memoryUsagePercent
	msm.stats.ResourceUtilization.CPUUsage = msm.globalResourceManager.usedCPU
	msm.stats.ResourceUtilization.MemoryUsage = msm.globalResourceManager.usedMemory
	msm.statsMu.Unlock()

	// 如果资源使用率过高，考虑暂停一些低优先级策略
	if cpuUsagePercent > 90 || memoryUsagePercent > 90 {
		log.Printf("Warning: High resource usage - CPU: %.1f%%, Memory: %.1f%%",
			cpuUsagePercent, memoryUsagePercent)
		msm.pauseLowPriorityStrategies()
	}
}

// checkStrategyWorkflowHealth 检查策略工作流健康状态
func (msm *MultiStrategyManager) checkStrategyWorkflowHealth() {
	msm.enginesMu.RLock()
	defer msm.enginesMu.RUnlock()

	for strategyID, engine := range msm.strategyEngines {
		if !engine.IsRunning() {
			log.Printf("Warning: Strategy workflow %s is not running", strategyID)

			// 尝试重启
			if err := engine.Start(); err != nil {
				log.Printf("Error: Failed to restart strategy workflow %s: %v", strategyID, err)
			}
		}

		// 检查是否有长时间运行的任务
		activeJobs := engine.GetActiveJobs()
		for _, job := range activeJobs {
			if time.Since(job.StartTime) > 2*time.Hour {
				log.Printf("Warning: Job %s in strategy %s has been running for %v",
					job.ID, strategyID, time.Since(job.StartTime))
			}
		}
	}
}

// updateStats 更新统计信息
func (msm *MultiStrategyManager) updateStats() {
	msm.statsMu.Lock()
	defer msm.statsMu.Unlock()

	msm.enginesMu.RLock()
	totalStrategies := int64(len(msm.strategyEngines))
	activeStrategies := int64(0)
	totalJobs := int64(0)
	completedJobs := int64(0)
	failedJobs := int64(0)

	for _, engine := range msm.strategyEngines {
		if engine.IsRunning() {
			activeStrategies++
		}

		stats := engine.GetStats()
		totalJobs += stats.TotalJobs
		completedJobs += stats.CompletedJobs
		failedJobs += stats.FailedJobs
	}
	msm.enginesMu.RUnlock()

	msm.strategiesMu.RLock()
	enabledStrategies := int64(len(msm.enabledStrategies))
	disabledStrategies := int64(len(msm.disabledStrategies))
	msm.strategiesMu.RUnlock()

	// 更新统计信息
	msm.stats.TotalStrategies = totalStrategies
	msm.stats.ActiveStrategies = activeStrategies
	msm.stats.EnabledStrategies = enabledStrategies
	msm.stats.DisabledStrategies = disabledStrategies
	msm.stats.TotalJobs = totalJobs
	msm.stats.CompletedJobs = completedJobs
	msm.stats.FailedJobs = failedJobs
	msm.stats.LastUpdateTime = time.Now()
}

// evaluateStrategyPerformance 评估策略性能
func (msm *MultiStrategyManager) evaluateStrategyPerformance(engine *StrategyWorkflowEngine) {
	stats := engine.GetStats()

	// 计算成功率
	var successRate float64
	if stats.TotalJobs > 0 {
		successRate = float64(stats.CompletedJobs) / float64(stats.TotalJobs)
	}

	// 评估策略是否应该启用或禁用
	strategyID := engine.StrategyID

	if successRate < 0.5 && stats.TotalJobs > 10 {
		// 成功率过低，考虑禁用
		msm.disableStrategy(strategyID, "Low success rate")
	} else if successRate > 0.8 && stats.TotalJobs > 5 {
		// 成功率高，考虑启用
		msm.enableStrategy(strategyID)
	}

	log.Printf("策略 %s 性能评估: 成功率=%.2f, 总任务=%d, 平均时间=%v",
		strategyID, successRate, stats.TotalJobs, stats.AverageJobTime)
}

// makeStrategyDecisions 做出策略决策
func (msm *MultiStrategyManager) makeStrategyDecisions() {
	msm.strategiesMu.RLock()
	enabledCount := len(msm.enabledStrategies)
	msm.strategiesMu.RUnlock()

	// 确保最少启用策略数量
	if enabledCount < msm.config.MinEnabledStrategies {
		msm.enableBestPerformingStrategies(msm.config.MinEnabledStrategies - enabledCount)
	}

	// 确保不超过最大启用策略数量
	if enabledCount > msm.config.MaxEnabledStrategies {
		msm.disableWorstPerformingStrategies(enabledCount - msm.config.MaxEnabledStrategies)
	}
}

// cleanupCompletedWorkflows 清理已完成的工作流
func (msm *MultiStrategyManager) cleanupCompletedWorkflows() {
	msm.enginesMu.Lock()
	defer msm.enginesMu.Unlock()

	for strategyID, engine := range msm.strategyEngines {
		currentStage := engine.GetCurrentStage()

		// 如果策略已完成或失败，考虑清理
		if currentStage == StageEnabled || currentStage == StageDisabled || currentStage == StageArchived {
			// 检查是否长时间未活动
			stats := engine.GetStats()
			if time.Since(stats.LastUpdateTime) > 24*time.Hour {
				log.Printf("清理长时间未活动的策略工作流: %s", strategyID)

				if err := msm.RemoveStrategyWorkflow(strategyID); err != nil {
					log.Printf("Warning: failed to remove strategy workflow %s: %v", strategyID, err)
				}
			}
		}
	}
}

// cleanupExpiredDisabledStrategies 清理过期的禁用策略
func (msm *MultiStrategyManager) cleanupExpiredDisabledStrategies() {
	msm.strategiesMu.Lock()
	defer msm.strategiesMu.Unlock()

	for strategyID, strategy := range msm.disabledStrategies {
		// 如果禁用超过7天，考虑归档
		if time.Since(strategy.DisabledAt) > 7*24*time.Hour {
			log.Printf("归档长期禁用的策略: %s", strategyID)
			delete(msm.disabledStrategies, strategyID)
		}
	}
}

// cleanupResourceAllocations 清理资源分配记录
func (msm *MultiStrategyManager) cleanupResourceAllocations() {
	msm.globalResourceManager.mu.Lock()
	defer msm.globalResourceManager.mu.Unlock()

	// 清理不存在的策略的资源分配
	msm.enginesMu.RLock()
	for strategyID := range msm.globalResourceManager.allocations {
		if _, exists := msm.strategyEngines[strategyID]; !exists {
			delete(msm.globalResourceManager.allocations, strategyID)
			log.Printf("清理不存在策略的资源分配: %s", strategyID)
		}
	}
	msm.enginesMu.RUnlock()
}

// pauseLowPriorityStrategies 暂停低优先级策略
func (msm *MultiStrategyManager) pauseLowPriorityStrategies() {
	// 简化实现：暂停最近创建的策略
	msm.enginesMu.RLock()
	defer msm.enginesMu.RUnlock()

	count := 0
	for strategyID, engine := range msm.strategyEngines {
		if count >= 2 { // 最多暂停2个策略
			break
		}

		if engine.IsRunning() {
			log.Printf("暂停低优先级策略: %s", strategyID)
			if err := engine.Stop(); err != nil {
				log.Printf("Warning: failed to pause strategy %s: %v", strategyID, err)
			}
			count++
		}
	}
}

// enableStrategy 启用策略
func (msm *MultiStrategyManager) enableStrategy(strategyID string) {
	msm.strategiesMu.Lock()
	defer msm.strategiesMu.Unlock()

	// 检查是否已经启用
	if _, exists := msm.enabledStrategies[strategyID]; exists {
		return
	}

	// 从禁用列表移除
	var disabledStrategy *DisabledStrategy
	if ds, exists := msm.disabledStrategies[strategyID]; exists {
		disabledStrategy = ds
		delete(msm.disabledStrategies, strategyID)
	}

	// 添加到启用列表
	enabledStrategy := &EnabledStrategy{
		ID:             strategyID,
		Name:           fmt.Sprintf("Strategy_%s", strategyID),
		Type:           "auto_generated",
		Version:        "1.0",
		EnabledAt:      time.Now(),
		LastUpdated:    time.Now(),
		IsActive:       true,
		TradingEnabled: true,
	}

	if disabledStrategy != nil {
		enabledStrategy.Name = disabledStrategy.Name
		enabledStrategy.Type = disabledStrategy.Type
		enabledStrategy.Version = disabledStrategy.Version
		enabledStrategy.Performance = disabledStrategy.Performance
	}

	msm.enabledStrategies[strategyID] = enabledStrategy

	// 发送事件
	msm.emitEvent("strategy_enabled", map[string]interface{}{
		"strategy_id": strategyID,
		"enabled_at":  enabledStrategy.EnabledAt,
	})

	log.Printf("策略 %s 已启用", strategyID)
}

// disableStrategy 禁用策略
func (msm *MultiStrategyManager) disableStrategy(strategyID, reason string) {
	msm.strategiesMu.Lock()
	defer msm.strategiesMu.Unlock()

	// 检查是否已经禁用
	if _, exists := msm.disabledStrategies[strategyID]; exists {
		return
	}

	// 从启用列表移除
	var enabledStrategy *EnabledStrategy
	if es, exists := msm.enabledStrategies[strategyID]; exists {
		enabledStrategy = es
		delete(msm.enabledStrategies, strategyID)
	}

	// 添加到禁用列表
	disabledStrategy := &DisabledStrategy{
		ID:         strategyID,
		Name:       fmt.Sprintf("Strategy_%s", strategyID),
		Type:       "auto_generated",
		Version:    "1.0",
		DisabledAt: time.Now(),
		Reason:     reason,
	}

	if enabledStrategy != nil {
		disabledStrategy.Name = enabledStrategy.Name
		disabledStrategy.Type = enabledStrategy.Type
		disabledStrategy.Version = enabledStrategy.Version
		disabledStrategy.Performance = enabledStrategy.Performance
	}

	msm.disabledStrategies[strategyID] = disabledStrategy

	// 发送事件
	msm.emitEvent("strategy_disabled", map[string]interface{}{
		"strategy_id": strategyID,
		"reason":      reason,
		"disabled_at": disabledStrategy.DisabledAt,
	})

	log.Printf("策略 %s 已禁用，原因: %s", strategyID, reason)
}

// enableBestPerformingStrategies 启用表现最好的策略
func (msm *MultiStrategyManager) enableBestPerformingStrategies(count int) {
	msm.strategiesMu.RLock()
	defer msm.strategiesMu.RUnlock()

	// 简化实现：从禁用策略中随机选择
	enabled := 0
	for strategyID := range msm.disabledStrategies {
		if enabled >= count {
			break
		}

		msm.enableStrategy(strategyID)
		enabled++
	}

	log.Printf("启用了 %d 个表现最好的策略", enabled)
}

// disableWorstPerformingStrategies 禁用表现最差的策略
func (msm *MultiStrategyManager) disableWorstPerformingStrategies(count int) {
	msm.strategiesMu.RLock()
	defer msm.strategiesMu.RUnlock()

	// 简化实现：从启用策略中随机选择
	disabled := 0
	for strategyID := range msm.enabledStrategies {
		if disabled >= count {
			break
		}

		msm.disableStrategy(strategyID, "Performance optimization")
		disabled++
	}

	log.Printf("禁用了 %d 个表现最差的策略", disabled)
}

// stopAllStrategyEngines 停止所有策略引擎
func (msm *MultiStrategyManager) stopAllStrategyEngines() {
	msm.enginesMu.RLock()
	engines := make([]*StrategyWorkflowEngine, 0, len(msm.strategyEngines))
	for _, engine := range msm.strategyEngines {
		engines = append(engines, engine)
	}
	msm.enginesMu.RUnlock()

	// 并发停止所有引擎
	var wg sync.WaitGroup
	for _, engine := range engines {
		wg.Add(1)
		go func(e *StrategyWorkflowEngine) {
			defer wg.Done()
			if err := e.Stop(); err != nil {
				log.Printf("Warning: failed to stop strategy engine %s: %v", e.StrategyID, err)
			}
		}(engine)
	}

	wg.Wait()
	log.Printf("所有策略引擎已停止")
}

// emitEvent 发送事件
func (msm *MultiStrategyManager) emitEvent(eventType string, data map[string]interface{}) {
	event := &events.Event{
		Type:      events.EventType(eventType),
		Source:    "multi_strategy_manager",
		Data:      data,
		Timestamp: time.Now(),
	}

	if err := msm.eventBus.Publish(event); err != nil {
		log.Printf("Warning: failed to emit event %s: %v", eventType, err)
	}
}
