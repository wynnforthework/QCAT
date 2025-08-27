package workflow

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/events"
)

// StrategyStage 策略生命周期阶段
type StrategyStage int

const (
	StageCreated StrategyStage = iota
	StageOnboarding
	StageBacktesting
	StageOptimizing
	StageLearning
	StageApplying
	StageEnabled
	StageDisabled
	StageArchived
)

func (s StrategyStage) String() string {
	stages := []string{
		"Created", "Onboarding", "Backtesting", "Optimizing",
		"Learning", "Applying", "Enabled", "Disabled", "Archived",
	}
	if int(s) < len(stages) {
		return stages[s]
	}
	return "Unknown"
}

// StrategyJob 策略任务
type StrategyJob struct {
	ID        string
	Type      JobType
	Stage     StrategyStage
	Status    JobStatus
	StartTime time.Time
	EndTime   time.Time
	Result    interface{}
	Error     error
	Progress  float64
	Metadata  map[string]interface{}
}

// JobType 任务类型
type JobType int

const (
	JobOnboarding JobType = iota
	JobBacktest
	JobOptimization
	JobLearning
	JobApplication
	JobEvaluation
)

// JobStatus 任务状态
type JobStatus int

const (
	JobPending JobStatus = iota
	JobRunning
	JobCompleted
	JobFailed
	JobCancelled
)

// StrategyWorkflowEngine 策略工作流引擎
type StrategyWorkflowEngine struct {
	// 策略标识
	StrategyID   string
	StrategyName string
	StrategyType string

	// 核心组件
	lifecycleManager *LifecycleManager
	resourcePool     *StrategyResourcePool
	eventBus         *events.EventBus

	// 状态管理
	currentStage StrategyStage
	activeJobs   map[string]*StrategyJob
	jobHistory   []*StrategyJob
	stageMu      sync.RWMutex
	jobsMu       sync.RWMutex

	// 并发控制
	concurrencyLimit int
	jobSemaphore     chan struct{}

	// 配置
	config *StrategyWorkflowConfig

	// 运行状态
	ctx       context.Context
	cancel    context.CancelFunc
	isRunning bool
	runningMu sync.RWMutex

	// 统计信息
	stats   *WorkflowStats
	statsMu sync.RWMutex
}

// StrategyWorkflowConfig 策略工作流配置
type StrategyWorkflowConfig struct {
	// 并发配置
	MaxConcurrentJobs int `yaml:"max_concurrent_jobs"`

	// 超时配置
	OnboardingTimeout   time.Duration `yaml:"onboarding_timeout"`
	BacktestTimeout     time.Duration `yaml:"backtest_timeout"`
	OptimizationTimeout time.Duration `yaml:"optimization_timeout"`
	LearningTimeout     time.Duration `yaml:"learning_timeout"`
	ApplicationTimeout  time.Duration `yaml:"application_timeout"`

	// 重试配置
	MaxRetries   int           `yaml:"max_retries"`
	RetryDelay   time.Duration `yaml:"retry_delay"`
	RetryBackoff float64       `yaml:"retry_backoff"`

	// 资源配置
	CPUQuota    float64 `yaml:"cpu_quota"`
	MemoryQuota int64   `yaml:"memory_quota"`

	// 性能阈值
	PerformanceThreshold float64 `yaml:"performance_threshold"`
	MinBacktestPeriod    int     `yaml:"min_backtest_period"`
}

// WorkflowStats 工作流统计信息
type WorkflowStats struct {
	TotalJobs      int64
	CompletedJobs  int64
	FailedJobs     int64
	ActiveJobs     int64
	AverageJobTime time.Duration
	CurrentStage   StrategyStage
	StageStartTime time.Time
	TotalStageTime map[StrategyStage]time.Duration
	LastUpdateTime time.Time
}

// NewStrategyWorkflowEngine 创建策略工作流引擎
func NewStrategyWorkflowEngine(strategyID, strategyName string, config *StrategyWorkflowConfig) *StrategyWorkflowEngine {
	if config == nil {
		config = GetDefaultWorkflowConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 创建事件总线
	eventBus := events.NewEventBus(&events.EventBusConfig{
		BufferSize: 1000,
		MaxRetries: 3,
		RetryDelay: time.Second,
	})

	// 创建资源池
	resourcePool := NewStrategyResourcePool(&StrategyResourceConfig{
		CPUQuota:        config.CPUQuota,
		MemoryQuota:     config.MemoryQuota,
		BacktestWorkers: 3,
		OptimizeWorkers: 2,
		LearningWorkers: 1,
	})

	engine := &StrategyWorkflowEngine{
		StrategyID:       strategyID,
		StrategyName:     strategyName,
		lifecycleManager: NewLifecycleManager(strategyID),
		resourcePool:     resourcePool,
		eventBus:         eventBus,
		currentStage:     StageCreated,
		activeJobs:       make(map[string]*StrategyJob),
		jobHistory:       make([]*StrategyJob, 0),
		concurrencyLimit: config.MaxConcurrentJobs,
		jobSemaphore:     make(chan struct{}, config.MaxConcurrentJobs),
		config:           config,
		ctx:              ctx,
		cancel:           cancel,
		stats: &WorkflowStats{
			TotalStageTime: make(map[StrategyStage]time.Duration),
			CurrentStage:   StageCreated,
			StageStartTime: time.Now(),
		},
	}

	return engine
}

// Start 启动策略工作流引擎
func (swe *StrategyWorkflowEngine) Start() error {
	swe.runningMu.Lock()
	defer swe.runningMu.Unlock()

	if swe.isRunning {
		return fmt.Errorf("strategy workflow engine %s is already running", swe.StrategyID)
	}

	log.Printf("启动策略工作流引擎: %s (%s)", swe.StrategyID, swe.StrategyName)

	// 启动资源池
	if err := swe.resourcePool.Start(); err != nil {
		return fmt.Errorf("failed to start resource pool: %w", err)
	}

	// 启动生命周期管理器
	if err := swe.lifecycleManager.Start(); err != nil {
		return fmt.Errorf("failed to start lifecycle manager: %w", err)
	}

	// 订阅因子更新事件
	if err := swe.subscribeToFactorEvents(); err != nil {
		log.Printf("Warning: failed to subscribe to factor events: %v", err)
	}

	swe.isRunning = true

	// 发送启动事件
	swe.emitEvent("workflow_started", map[string]interface{}{
		"strategy_id":   swe.StrategyID,
		"strategy_name": swe.StrategyName,
		"stage":         swe.currentStage.String(),
	})

	log.Printf("策略工作流引擎 %s 启动完成", swe.StrategyID)
	return nil
}

// Stop 停止策略工作流引擎
func (swe *StrategyWorkflowEngine) Stop() error {
	swe.runningMu.Lock()
	defer swe.runningMu.Unlock()

	if !swe.isRunning {
		return nil
	}

	log.Printf("停止策略工作流引擎: %s", swe.StrategyID)

	// 取消所有活跃任务
	swe.cancel()

	// 等待所有任务完成
	swe.waitForActiveJobs()

	// 停止资源池
	if err := swe.resourcePool.Stop(); err != nil {
		log.Printf("Warning: failed to stop resource pool: %v", err)
	}

	// 停止生命周期管理器
	if err := swe.lifecycleManager.Stop(); err != nil {
		log.Printf("Warning: failed to stop lifecycle manager: %v", err)
	}

	swe.isRunning = false

	// 发送停止事件
	swe.emitEvent("workflow_stopped", map[string]interface{}{
		"strategy_id": swe.StrategyID,
		"stage":       swe.currentStage.String(),
	})

	log.Printf("策略工作流引擎 %s 已停止", swe.StrategyID)
	return nil
}

// ExecuteLifecycle 执行完整生命周期
func (swe *StrategyWorkflowEngine) ExecuteLifecycle() error {
	if !swe.isRunning {
		return fmt.Errorf("workflow engine is not running")
	}

	log.Printf("开始执行策略 %s 的完整生命周期", swe.StrategyID)

	// 执行生命周期阶段
	stages := []StrategyStage{
		StageOnboarding,
		StageBacktesting,
		StageOptimizing,
		StageLearning,
		StageApplying,
	}

	for _, stage := range stages {
		if err := swe.executeStage(stage); err != nil {
			log.Printf("策略 %s 在阶段 %s 执行失败: %v", swe.StrategyID, stage.String(), err)
			return err
		}

		// 检查是否应该继续
		if !swe.shouldContinueToNextStage(stage) {
			log.Printf("策略 %s 在阶段 %s 后停止执行", swe.StrategyID, stage.String())
			break
		}
	}

	log.Printf("策略 %s 生命周期执行完成", swe.StrategyID)
	return nil
}

// executeStage 执行指定阶段
func (swe *StrategyWorkflowEngine) executeStage(stage StrategyStage) error {
	log.Printf("策略 %s 开始执行阶段: %s", swe.StrategyID, stage.String())

	// 更新当前阶段
	swe.transitionToStage(stage)

	// 根据阶段类型执行相应任务
	switch stage {
	case StageOnboarding:
		return swe.executeOnboardingStage()
	case StageBacktesting:
		return swe.executeBacktestStage()
	case StageOptimizing:
		return swe.executeOptimizationStage()
	case StageLearning:
		return swe.executeLearningStage()
	case StageApplying:
		return swe.executeApplicationStage()
	default:
		return fmt.Errorf("unsupported stage: %s", stage.String())
	}
}

// executeOnboardingStage 执行策略引入阶段
func (swe *StrategyWorkflowEngine) executeOnboardingStage() error {
	job := swe.createJob(JobOnboarding, StageOnboarding)

	return swe.executeJob(job, func(ctx context.Context) (interface{}, error) {
		log.Printf("执行策略引入: %s", swe.StrategyID)

		// 模拟策略引入过程
		time.Sleep(5 * time.Second)

		return map[string]interface{}{
			"strategy_code": "generated_strategy_code",
			"initial_params": map[string]float64{
				"param1": 0.1,
				"param2": 0.2,
			},
		}, nil
	})
}

// executeBacktestStage 执行回测阶段
func (swe *StrategyWorkflowEngine) executeBacktestStage() error {
	// 并发执行多个回测任务
	backtestJobs := []string{"backtest_1", "backtest_2", "backtest_3"}

	var wg sync.WaitGroup
	results := make(chan *StrategyJob, len(backtestJobs))
	errors := make(chan error, len(backtestJobs))

	for _, backtestID := range backtestJobs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()

			job := swe.createJob(JobBacktest, StageBacktesting)
			job.ID = fmt.Sprintf("%s_%s", job.ID, id)

			err := swe.executeJob(job, func(ctx context.Context) (interface{}, error) {
				log.Printf("执行回测任务: %s", id)

				// 模拟回测过程
				time.Sleep(10 * time.Second)

				return map[string]interface{}{
					"sharpe_ratio": 1.2 + (float64(len(id)) * 0.1),
					"max_drawdown": 0.15,
					"total_return": 0.25,
				}, nil
			})

			if err != nil {
				errors <- err
			} else {
				results <- job
			}
		}(backtestID)
	}

	wg.Wait()
	close(results)
	close(errors)

	// 检查错误
	select {
	case err := <-errors:
		return err
	default:
	}

	// 收集结果
	var backtestResults []*StrategyJob
	for job := range results {
		backtestResults = append(backtestResults, job)
	}

	log.Printf("回测阶段完成，共完成 %d 个回测任务", len(backtestResults))
	return nil
}

// executeOptimizationStage 执行参数优化阶段
func (swe *StrategyWorkflowEngine) executeOptimizationStage() error {
	// 并发执行多个优化任务
	optimizationJobs := []string{"bayesian_opt", "genetic_opt"}

	var wg sync.WaitGroup
	results := make(chan *StrategyJob, len(optimizationJobs))
	errors := make(chan error, len(optimizationJobs))

	for _, optID := range optimizationJobs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()

			job := swe.createJob(JobOptimization, StageOptimizing)
			job.ID = fmt.Sprintf("%s_%s", job.ID, id)

			err := swe.executeJob(job, func(ctx context.Context) (interface{}, error) {
				log.Printf("执行优化任务: %s", id)

				// 模拟优化过程
				time.Sleep(15 * time.Second)

				return map[string]interface{}{
					"optimal_params": map[string]float64{
						"param1": 0.15,
						"param2": 0.25,
					},
					"optimization_score": 0.85,
				}, nil
			})

			if err != nil {
				errors <- err
			} else {
				results <- job
			}
		}(optID)
	}

	wg.Wait()
	close(results)
	close(errors)

	// 检查错误
	select {
	case err := <-errors:
		return err
	default:
	}

	log.Printf("参数优化阶段完成")
	return nil
}

// executeLearningStage 执行自学习阶段
func (swe *StrategyWorkflowEngine) executeLearningStage() error {
	job := swe.createJob(JobLearning, StageLearning)

	return swe.executeJob(job, func(ctx context.Context) (interface{}, error) {
		log.Printf("执行自学习任务: %s", swe.StrategyID)

		// 模拟自学习过程
		time.Sleep(20 * time.Second)

		return map[string]interface{}{
			"learned_patterns": []string{"pattern1", "pattern2"},
			"model_accuracy":   0.92,
			"feature_importance": map[string]float64{
				"feature1": 0.3,
				"feature2": 0.7,
			},
		}, nil
	})
}

// executeApplicationStage 执行参数应用阶段
func (swe *StrategyWorkflowEngine) executeApplicationStage() error {
	job := swe.createJob(JobApplication, StageApplying)

	return swe.executeJob(job, func(ctx context.Context) (interface{}, error) {
		log.Printf("执行参数应用: %s", swe.StrategyID)

		// 模拟参数应用过程
		time.Sleep(8 * time.Second)

		return map[string]interface{}{
			"applied_params": map[string]float64{
				"param1": 0.15,
				"param2": 0.25,
			},
			"application_status": "success",
		}, nil
	})
}

// transitionToStage 转换到指定阶段
func (swe *StrategyWorkflowEngine) transitionToStage(stage StrategyStage) {
	swe.stageMu.Lock()
	defer swe.stageMu.Unlock()

	// 记录阶段时间
	now := time.Now()
	if swe.currentStage != StageCreated {
		duration := now.Sub(swe.stats.StageStartTime)
		swe.statsMu.Lock()
		swe.stats.TotalStageTime[swe.currentStage] += duration
		swe.statsMu.Unlock()
	}

	// 更新当前阶段
	oldStage := swe.currentStage
	swe.currentStage = stage
	swe.stats.CurrentStage = stage
	swe.stats.StageStartTime = now

	log.Printf("策略 %s 阶段转换: %s -> %s", swe.StrategyID, oldStage.String(), stage.String())

	// 发送阶段转换事件
	swe.emitEvent("stage_transition", map[string]interface{}{
		"strategy_id": swe.StrategyID,
		"old_stage":   oldStage.String(),
		"new_stage":   stage.String(),
		"timestamp":   now,
	})
}

// createJob 创建任务
func (swe *StrategyWorkflowEngine) createJob(jobType JobType, stage StrategyStage) *StrategyJob {
	job := &StrategyJob{
		ID:        fmt.Sprintf("job_%s_%d", swe.StrategyID, time.Now().UnixNano()),
		Type:      jobType,
		Stage:     stage,
		Status:    JobPending,
		StartTime: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// 添加到活跃任务
	swe.jobsMu.Lock()
	swe.activeJobs[job.ID] = job
	swe.jobsMu.Unlock()

	// 更新统计
	swe.statsMu.Lock()
	swe.stats.TotalJobs++
	swe.stats.ActiveJobs++
	swe.statsMu.Unlock()

	return job
}

// executeJob 执行任务
func (swe *StrategyWorkflowEngine) executeJob(job *StrategyJob, executor func(context.Context) (interface{}, error)) error {
	// 获取信号量
	select {
	case swe.jobSemaphore <- struct{}{}:
		defer func() { <-swe.jobSemaphore }()
	case <-swe.ctx.Done():
		return swe.ctx.Err()
	}

	// 更新任务状态
	job.Status = JobRunning
	job.StartTime = time.Now()

	// 发送任务开始事件
	swe.emitEvent("job_started", map[string]interface{}{
		"strategy_id": swe.StrategyID,
		"job_id":      job.ID,
		"job_type":    job.Type,
		"stage":       job.Stage.String(),
	})

	// 执行任务
	result, err := executor(swe.ctx)
	job.EndTime = time.Now()

	if err != nil {
		job.Status = JobFailed
		job.Error = err

		// 更新统计
		swe.statsMu.Lock()
		swe.stats.FailedJobs++
		swe.stats.ActiveJobs--
		swe.statsMu.Unlock()

		// 发送任务失败事件
		swe.emitEvent("job_failed", map[string]interface{}{
			"strategy_id": swe.StrategyID,
			"job_id":      job.ID,
			"error":       err.Error(),
		})

		return err
	}

	job.Status = JobCompleted
	job.Result = result

	// 更新统计
	duration := job.EndTime.Sub(job.StartTime)
	swe.statsMu.Lock()
	swe.stats.CompletedJobs++
	swe.stats.ActiveJobs--
	if swe.stats.CompletedJobs > 0 {
		swe.stats.AverageJobTime = time.Duration(
			(int64(swe.stats.AverageJobTime)*int64(swe.stats.CompletedJobs-1) + int64(duration)) / int64(swe.stats.CompletedJobs),
		)
	}
	swe.statsMu.Unlock()

	// 移动到历史记录
	swe.jobsMu.Lock()
	delete(swe.activeJobs, job.ID)
	swe.jobHistory = append(swe.jobHistory, job)
	swe.jobsMu.Unlock()

	// 发送任务完成事件
	swe.emitEvent("job_completed", map[string]interface{}{
		"strategy_id": swe.StrategyID,
		"job_id":      job.ID,
		"duration":    duration.String(),
		"result":      result,
	})

	return nil
}

// GetDefaultWorkflowConfig 获取默认工作流配置
func GetDefaultWorkflowConfig() *StrategyWorkflowConfig {
	return &StrategyWorkflowConfig{
		MaxConcurrentJobs:    5,
		OnboardingTimeout:    30 * time.Minute,
		BacktestTimeout:      60 * time.Minute,
		OptimizationTimeout:  120 * time.Minute,
		LearningTimeout:      240 * time.Minute,
		ApplicationTimeout:   30 * time.Minute,
		MaxRetries:           3,
		RetryDelay:           30 * time.Second,
		RetryBackoff:         2.0,
		CPUQuota:             2.0,
		MemoryQuota:          4 * 1024 * 1024 * 1024, // 4GB
		PerformanceThreshold: 0.1,
		MinBacktestPeriod:    30,
	}
}

// emitEvent 发送事件
func (swe *StrategyWorkflowEngine) emitEvent(eventType string, data map[string]interface{}) {
	event := &events.Event{
		Type:      events.EventType(eventType),
		Source:    fmt.Sprintf("strategy_workflow_%s", swe.StrategyID),
		Data:      data,
		Timestamp: time.Now(),
	}

	if err := swe.eventBus.Publish(event); err != nil {
		log.Printf("Warning: failed to emit event %s: %v", eventType, err)
	}
}

// shouldContinueToNextStage 判断是否应该继续到下一阶段
func (swe *StrategyWorkflowEngine) shouldContinueToNextStage(currentStage StrategyStage) bool {
	// 根据当前阶段的执行结果决定是否继续
	switch currentStage {
	case StageOnboarding:
		// 策略引入成功后继续
		return true
	case StageBacktesting:
		// 检查回测结果是否满足要求
		return swe.evaluateBacktestResults()
	case StageOptimizing:
		// 检查优化结果是否满足要求
		return swe.evaluateOptimizationResults()
	case StageLearning:
		// 检查学习结果是否满足要求
		return swe.evaluateLearningResults()
	case StageApplying:
		// 参数应用后进入评估阶段
		return swe.evaluateApplicationResults()
	default:
		return false
	}
}

// evaluateBacktestResults 评估回测结果
func (swe *StrategyWorkflowEngine) evaluateBacktestResults() bool {
	// 简化的评估逻辑
	// 实际应用中应该检查夏普比率、最大回撤等指标
	return true
}

// evaluateOptimizationResults 评估优化结果
func (swe *StrategyWorkflowEngine) evaluateOptimizationResults() bool {
	// 简化的评估逻辑
	// 实际应用中应该检查优化收敛性、参数稳定性等
	return true
}

// evaluateLearningResults 评估学习结果
func (swe *StrategyWorkflowEngine) evaluateLearningResults() bool {
	// 简化的评估逻辑
	// 实际应用中应该检查模型准确率、泛化能力等
	return true
}

// evaluateApplicationResults 评估应用结果
func (swe *StrategyWorkflowEngine) evaluateApplicationResults() bool {
	// 简化的评估逻辑
	// 实际应用中应该检查参数应用是否成功
	return true
}

// waitForActiveJobs 等待所有活跃任务完成
func (swe *StrategyWorkflowEngine) waitForActiveJobs() {
	for {
		swe.jobsMu.RLock()
		activeCount := len(swe.activeJobs)
		swe.jobsMu.RUnlock()

		if activeCount == 0 {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// GetStats 获取统计信息
func (swe *StrategyWorkflowEngine) GetStats() *WorkflowStats {
	swe.statsMu.RLock()
	defer swe.statsMu.RUnlock()

	// 创建副本
	stats := *swe.stats
	stats.TotalStageTime = make(map[StrategyStage]time.Duration)
	for stage, duration := range swe.stats.TotalStageTime {
		stats.TotalStageTime[stage] = duration
	}

	return &stats
}

// GetCurrentStage 获取当前阶段
func (swe *StrategyWorkflowEngine) GetCurrentStage() StrategyStage {
	swe.stageMu.RLock()
	defer swe.stageMu.RUnlock()
	return swe.currentStage
}

// GetActiveJobs 获取活跃任务
func (swe *StrategyWorkflowEngine) GetActiveJobs() []*StrategyJob {
	swe.jobsMu.RLock()
	defer swe.jobsMu.RUnlock()

	jobs := make([]*StrategyJob, 0, len(swe.activeJobs))
	for _, job := range swe.activeJobs {
		jobs = append(jobs, job)
	}

	return jobs
}

// GetJobHistory 获取任务历史
func (swe *StrategyWorkflowEngine) GetJobHistory() []*StrategyJob {
	swe.jobsMu.RLock()
	defer swe.jobsMu.RUnlock()

	// 返回副本
	history := make([]*StrategyJob, len(swe.jobHistory))
	copy(history, swe.jobHistory)

	return history
}

// FactorEventHandler 因子事件处理器
type FactorEventHandler struct {
	strategyEngine *StrategyWorkflowEngine
	eventTypes     []events.EventType
	name           string
}

// Handle 处理事件
func (feh *FactorEventHandler) Handle(ctx context.Context, event *events.Event) error {
	switch event.Type {
	case "strategy_factor_library_updated":
		return feh.strategyEngine.handleFactorLibraryUpdate(event)
	case "strategy_factor_updated":
		return feh.strategyEngine.handleFactorUpdate(event)
	case "strategy_factor_deleted":
		return feh.strategyEngine.handleFactorDeleted(event)
	default:
		return nil
	}
}

// GetName 获取处理器名称
func (feh *FactorEventHandler) GetName() string {
	return feh.name
}

// GetEventTypes 获取事件类型
func (feh *FactorEventHandler) GetEventTypes() []events.EventType {
	return feh.eventTypes
}

// GetPriority 获取优先级
func (feh *FactorEventHandler) GetPriority() int {
	return 100 // 中等优先级
}

// subscribeToFactorEvents 订阅因子更新事件
func (swe *StrategyWorkflowEngine) subscribeToFactorEvents() error {
	// 创建因子事件处理器
	handler := &FactorEventHandler{
		strategyEngine: swe,
		eventTypes: []events.EventType{
			"strategy_factor_library_updated",
			"strategy_factor_updated",
			"strategy_factor_deleted",
		},
		name: fmt.Sprintf("factor_handler_%s", swe.StrategyID),
	}

	// 订阅事件
	_, err := swe.eventBus.Subscribe(handler.eventTypes, handler, nil)
	if err != nil {
		return fmt.Errorf("failed to subscribe to factor events: %w", err)
	}

	log.Printf("策略 %s 已订阅因子更新事件", swe.StrategyID)
	return nil
}

// handleFactorLibraryUpdate 处理因子库更新事件
func (swe *StrategyWorkflowEngine) handleFactorLibraryUpdate(event *events.Event) error {
	log.Printf("策略 %s 收到因子库更新事件", swe.StrategyID)

	// 检查当前阶段是否需要因子信息
	currentStage := swe.GetCurrentStage()
	if currentStage == StageBacktesting || currentStage == StageOptimizing || currentStage == StageLearning {
		log.Printf("策略 %s 在阶段 %s，因子库更新可能影响当前任务", swe.StrategyID, currentStage.String())

		// 可以选择重新启动当前阶段或标记需要更新
		// 这里简化处理，只记录日志
		swe.emitEvent("factor_library_updated_received", map[string]interface{}{
			"strategy_id": swe.StrategyID,
			"stage":       currentStage.String(),
			"version":     event.Data["version"],
		})
	}

	return nil
}

// handleFactorUpdate 处理单个因子更新事件
func (swe *StrategyWorkflowEngine) handleFactorUpdate(event *events.Event) error {
	factorID, ok := event.Data["factor_id"].(string)
	if !ok {
		return fmt.Errorf("invalid factor_id in update event")
	}

	log.Printf("策略 %s 收到因子 %s 更新事件", swe.StrategyID, factorID)

	// 发送策略级因子更新事件
	swe.emitEvent("factor_updated_received", map[string]interface{}{
		"strategy_id": swe.StrategyID,
		"factor_id":   factorID,
		"stage":       swe.GetCurrentStage().String(),
	})

	return nil
}

// handleFactorDeleted 处理因子删除事件
func (swe *StrategyWorkflowEngine) handleFactorDeleted(event *events.Event) error {
	factorID, ok := event.Data["factor_id"].(string)
	if !ok {
		return fmt.Errorf("invalid factor_id in delete event")
	}

	log.Printf("策略 %s 收到因子 %s 删除事件", swe.StrategyID, factorID)

	// 发送策略级因子删除事件
	swe.emitEvent("factor_deleted_received", map[string]interface{}{
		"strategy_id": swe.StrategyID,
		"factor_id":   factorID,
		"stage":       swe.GetCurrentStage().String(),
	})

	return nil
}

// IsRunning 检查是否正在运行
func (swe *StrategyWorkflowEngine) IsRunning() bool {
	swe.runningMu.RLock()
	defer swe.runningMu.RUnlock()
	return swe.isRunning
}
