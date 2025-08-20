package executor

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/config"
	"qcat/internal/database"
	"qcat/internal/exchange"
	"qcat/internal/exchange/account"
	"qcat/internal/monitor"
)

// RealtimeExecutor 实时执行引擎
// 负责实时执行自动化决策和交易指令
type RealtimeExecutor struct {
	config         *config.Config
	db             *database.DB
	exchange       exchange.Exchange
	accountManager *account.Manager
	metrics        *monitor.MetricsCollector

	// 执行组件
	positionExecutor *PositionExecutor
	riskExecutor     *RiskExecutor
	orderExecutor    *OrderExecutor
	strategyExecutor *StrategyExecutor
	dataExecutor     *DataExecutor
	systemExecutor   *SystemExecutor

	// 运行状态
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	isRunning bool
	mu        sync.RWMutex

	// 执行队列
	actionQueue chan *ExecutionAction
	workers     []*ExecutionWorker

	// 统计信息
	stats *ExecutorStats
}

// ExecutionAction 执行动作
type ExecutionAction struct {
	ID          string
	Type        ActionType
	Priority    int
	Symbol      string
	Action      string
	Parameters  map[string]interface{}
	Timeout     time.Duration
	RetryCount  int
	MaxRetries  int
	CreatedAt   time.Time
	ScheduledAt time.Time
	Handler     ActionHandler
}

// ActionType 动作类型
type ActionType string

const (
	ActionTypePosition  ActionType = "position"
	ActionTypeRisk      ActionType = "risk"
	ActionTypeOrder     ActionType = "order"
	ActionTypeStop      ActionType = "stop"
	ActionTypeHedge     ActionType = "hedge"
	ActionTypeStrategy  ActionType = "strategy"
	ActionTypeData      ActionType = "data"
	ActionTypeSecurity  ActionType = "security"
	ActionTypeSystem    ActionType = "system"
	ActionTypeLearning  ActionType = "learning"
	ActionTypeOptimize  ActionType = "optimize"
	ActionTypeRebalance ActionType = "rebalance"
	ActionTypeTransfer  ActionType = "transfer"
	ActionTypeNotify    ActionType = "notify"
	ActionTypeBacktest  ActionType = "backtest"
)

// ActionHandler 动作处理器
type ActionHandler func(ctx context.Context, action *ExecutionAction) error

// ExecutorStats 执行器统计
type ExecutorStats struct {
	TotalActions      int
	ExecutedActions   int
	FailedActions     int
	SuccessfulActions int
	QueueLength       int
	AverageLatency    time.Duration
	LastExecutionTime time.Time
	mu                sync.RWMutex
}

// NewRealtimeExecutor 创建实时执行引擎
func NewRealtimeExecutor(
	cfg *config.Config,
	db *database.DB,
	exchange exchange.Exchange,
	accountManager *account.Manager,
	metrics *monitor.MetricsCollector,
) *RealtimeExecutor {
	ctx, cancel := context.WithCancel(context.Background())

	executor := &RealtimeExecutor{
		config:         cfg,
		db:             db,
		exchange:       exchange,
		accountManager: accountManager,
		metrics:        metrics,
		ctx:            ctx,
		cancel:         cancel,
		actionQueue:    make(chan *ExecutionAction, 10000),
		workers:        make([]*ExecutionWorker, 0),
		stats:          &ExecutorStats{},
	}

	// 初始化执行组件
	executor.initializeExecutors()

	// 初始化工作线程
	executor.initializeWorkers()

	return executor
}

// Start 启动执行引擎
func (re *RealtimeExecutor) Start() error {
	re.mu.Lock()
	defer re.mu.Unlock()

	if re.isRunning {
		return fmt.Errorf("realtime executor is already running")
	}

	log.Println("Starting realtime executor...")

	// 启动工作线程
	for _, worker := range re.workers {
		re.wg.Add(1)
		go worker.Start(&re.wg)
	}

	// 启动监控循环
	re.wg.Add(1)
	go re.monitorLoop()

	re.isRunning = true
	log.Println("Realtime executor started successfully")

	return nil
}

// Stop 停止执行引擎
func (re *RealtimeExecutor) Stop() error {
	re.mu.Lock()
	defer re.mu.Unlock()

	if !re.isRunning {
		return nil
	}

	log.Println("Stopping realtime executor...")

	// 取消上下文
	re.cancel()

	// 等待所有goroutine完成
	re.wg.Wait()

	// 关闭动作队列
	close(re.actionQueue)

	re.isRunning = false
	log.Println("Realtime executor stopped")

	return nil
}

// ExecuteAction 执行动作
func (re *RealtimeExecutor) ExecuteAction(action *ExecutionAction) error {
	if action.ID == "" {
		action.ID = fmt.Sprintf("action_%d", time.Now().UnixNano())
	}

	action.CreatedAt = time.Now()
	action.ScheduledAt = time.Now()

	// 设置处理器
	if action.Handler == nil {
		handler, err := re.getActionHandler(action.Type)
		if err != nil {
			return fmt.Errorf("failed to get action handler: %w", err)
		}
		action.Handler = handler
	}

	// 加入执行队列
	select {
	case re.actionQueue <- action:
		log.Printf("Action queued: %s (%s)", action.Action, action.ID)
		return nil
	default:
		return fmt.Errorf("action queue is full")
	}
}

// getActionHandler 获取动作处理器
func (re *RealtimeExecutor) getActionHandler(actionType ActionType) (ActionHandler, error) {
	switch actionType {
	case ActionTypePosition:
		return re.positionExecutor.HandleAction, nil
	case ActionTypeRisk:
		return re.riskExecutor.HandleAction, nil
	case ActionTypeOrder:
		return re.orderExecutor.HandleAction, nil
	case ActionTypeStrategy:
		return re.strategyExecutor.HandleAction, nil
	case ActionTypeData:
		return re.dataExecutor.HandleAction, nil
	case ActionTypeSystem:
		return re.systemExecutor.HandleAction, nil
	case ActionTypeStop:
		return re.riskExecutor.HandleAction, nil // 止损由风险执行器处理
	case ActionTypeHedge:
		return re.positionExecutor.HandleAction, nil // 对冲由仓位执行器处理
	case ActionTypeOptimize:
		return re.strategyExecutor.HandleAction, nil // 优化由策略执行器处理
	case ActionTypeRebalance:
		return re.positionExecutor.HandleAction, nil // 再平衡由仓位执行器处理
	case ActionTypeTransfer:
		return re.riskExecutor.HandleAction, nil // 资金转移由风险执行器处理
	case ActionTypeBacktest:
		return re.dataExecutor.HandleAction, nil // 回测由数据执行器处理
	case ActionTypeSecurity:
		return re.systemExecutor.HandleAction, nil // 安全由系统执行器处理
	case ActionTypeLearning:
		return re.strategyExecutor.HandleAction, nil // 学习由策略执行器处理
	case ActionTypeNotify:
		return re.systemExecutor.HandleAction, nil // 通知由系统执行器处理
	default:
		return nil, fmt.Errorf("unknown action type: %s", actionType)
	}
}

// monitorLoop 监控循环
func (re *RealtimeExecutor) monitorLoop() {
	defer re.wg.Done()

	ticker := time.NewTicker(time.Second * 30) // 每30秒更新统计
	defer ticker.Stop()

	for {
		select {
		case <-re.ctx.Done():
			return
		case <-ticker.C:
			re.updateStats()
		}
	}
}

// updateStats 更新统计信息
func (re *RealtimeExecutor) updateStats() {
	re.stats.mu.Lock()
	defer re.stats.mu.Unlock()

	// 更新统计信息
	re.stats.LastExecutionTime = time.Now()

	// 计算队列长度
	queueLength := len(re.actionQueue)
	re.stats.QueueLength = queueLength

	// 记录性能指标
	if re.metrics != nil {
		// 使用现有的系统指标更新方法
		re.metrics.UpdateSystemMetrics(0, 0, 0, queueLength) // 使用goroutines字段记录队列长度

		// 记录执行器相关的指标
		labels := map[string]string{
			"component": "executor",
		}

		// 使用IncrementCounter记录计数指标
		if re.stats.TotalActions > 0 {
			re.metrics.IncrementCounter("executor_actions", labels)
		}

		// 计算成功率并记录
		if re.stats.TotalActions > 0 {
			successRate := float64(re.stats.SuccessfulActions) / float64(re.stats.TotalActions)
			log.Printf("Executor success rate: %.2f%% (%d/%d)",
				successRate*100, re.stats.SuccessfulActions, re.stats.TotalActions)
		}
	}

	// 如果队列过长，记录警告
	if queueLength > 5000 {
		log.Printf("Warning: Execution queue is getting long: %d actions", queueLength)
	}
}

// initializeExecutors 初始化执行器
func (re *RealtimeExecutor) initializeExecutors() {
	re.positionExecutor = NewPositionExecutor(re.config, re.db, re.exchange, re.accountManager)
	re.riskExecutor = NewRiskExecutor(re.config, re.db, re.exchange, re.accountManager)
	re.orderExecutor = NewOrderExecutor(re.config, re.db, re.exchange, re.accountManager)
	re.strategyExecutor = NewStrategyExecutor(re.config, re.db, re.exchange, re.accountManager)
	re.dataExecutor = NewDataExecutor(re.config, re.db, re.exchange, re.accountManager)
	re.systemExecutor = NewSystemExecutor(re.config, re.db, re.exchange, re.accountManager)
}

// initializeWorkers 初始化工作线程
func (re *RealtimeExecutor) initializeWorkers() {
	workerCount := 3 // 可配置
	for i := 0; i < workerCount; i++ {
		worker := NewExecutionWorker(i, re.actionQueue, re.handleActionCompletion)
		re.workers = append(re.workers, worker)
	}
}

// handleActionCompletion 处理动作完成
func (re *RealtimeExecutor) handleActionCompletion(action *ExecutionAction, err error) {
	re.stats.mu.Lock()
	re.stats.TotalActions++

	if err != nil {
		log.Printf("Action failed: %s, error: %v, retry: %d/%d",
			action.Action, err, action.RetryCount, action.MaxRetries)

		// 重试逻辑
		if action.RetryCount < action.MaxRetries {
			action.RetryCount++
			action.ScheduledAt = time.Now().Add(time.Second * time.Duration(action.RetryCount))
			re.stats.mu.Unlock()

			// 重新加入队列
			select {
			case re.actionQueue <- action:
				log.Printf("Action retried: %s", action.Action)
			default:
				log.Printf("Failed to retry action: queue is full")
				// 如果重试失败，标记为失败
				re.stats.mu.Lock()
				re.stats.FailedActions++
				re.stats.mu.Unlock()
			}
		} else {
			re.stats.FailedActions++
			re.stats.mu.Unlock()
			log.Printf("Action permanently failed after %d retries: %s", action.MaxRetries, action.Action)
		}
	} else {
		log.Printf("Action completed successfully: %s", action.Action)
		re.stats.ExecutedActions++
		re.stats.SuccessfulActions++
		re.stats.mu.Unlock()

		// 记录执行延迟
		if re.metrics != nil {
			executionTime := time.Since(action.CreatedAt)
			re.metrics.RecordTrade(action.Symbol, action.Action, 1.0, executionTime)
		}
	}
}

// GetStats 获取统计信息
func (re *RealtimeExecutor) GetStats() *ExecutorStats {
	re.stats.mu.RLock()
	defer re.stats.mu.RUnlock()

	return &ExecutorStats{
		TotalActions:      re.stats.TotalActions,
		ExecutedActions:   re.stats.ExecutedActions,
		FailedActions:     re.stats.FailedActions,
		SuccessfulActions: re.stats.SuccessfulActions,
		QueueLength:       re.stats.QueueLength,
		AverageLatency:    re.stats.AverageLatency,
		LastExecutionTime: re.stats.LastExecutionTime,
	}
}

// 便捷方法

// ExecutePositionAdjustment 执行仓位调整
func (re *RealtimeExecutor) ExecutePositionAdjustment(symbol string, targetSize float64) error {
	action := &ExecutionAction{
		Type:     ActionTypePosition,
		Symbol:   symbol,
		Action:   "adjust_position",
		Priority: 2,
		Parameters: map[string]interface{}{
			"target_size": targetSize,
		},
		Timeout:    time.Minute * 5,
		MaxRetries: 2,
	}

	return re.ExecuteAction(action)
}

// ExecuteStopLoss 执行止损
func (re *RealtimeExecutor) ExecuteStopLoss(symbol string, price float64) error {
	action := &ExecutionAction{
		Type:     ActionTypeStop,
		Symbol:   symbol,
		Action:   "stop_loss",
		Priority: 1, // 高优先级
		Parameters: map[string]interface{}{
			"stop_price": price,
		},
		Timeout:    time.Minute * 2,
		MaxRetries: 3,
	}

	return re.ExecuteAction(action)
}

// ExecuteRiskControl 执行风险控制
func (re *RealtimeExecutor) ExecuteRiskControl(action string, parameters map[string]interface{}) error {
	execAction := &ExecutionAction{
		Type:       ActionTypeRisk,
		Action:     action,
		Priority:   1, // 高优先级
		Parameters: parameters,
		Timeout:    time.Minute * 3,
		MaxRetries: 2,
	}

	return re.ExecuteAction(execAction)
}

// ExecutionWorker 执行工作器
type ExecutionWorker struct {
	id                int
	actionQueue       <-chan *ExecutionAction
	completionHandler func(*ExecutionAction, error)
	isRunning         bool
	mu                sync.RWMutex
}

// NewExecutionWorker 创建执行工作器
func NewExecutionWorker(id int, actionQueue <-chan *ExecutionAction, completionHandler func(*ExecutionAction, error)) *ExecutionWorker {
	return &ExecutionWorker{
		id:                id,
		actionQueue:       actionQueue,
		completionHandler: completionHandler,
	}
}

// Start 启动工作器
func (ew *ExecutionWorker) Start(wg *sync.WaitGroup) {
	defer wg.Done()

	ew.mu.Lock()
	ew.isRunning = true
	ew.mu.Unlock()

	log.Printf("Execution worker %d started", ew.id)

	for action := range ew.actionQueue {
		ew.executeAction(action)
	}

	ew.mu.Lock()
	ew.isRunning = false
	ew.mu.Unlock()

	log.Printf("Execution worker %d stopped", ew.id)
}

// executeAction 执行动作
func (ew *ExecutionWorker) executeAction(action *ExecutionAction) {
	log.Printf("Worker %d executing action: %s", ew.id, action.Action)

	startTime := time.Now()

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), action.Timeout)
	defer cancel()

	// 执行动作
	var err error
	if action.Handler != nil {
		err = action.Handler(ctx, action)
	} else {
		err = fmt.Errorf("action %s has no handler", action.Action)
	}

	duration := time.Since(startTime)

	if err != nil {
		log.Printf("Worker %d action failed: %s, duration: %v, error: %v",
			ew.id, action.Action, duration, err)
	} else {
		log.Printf("Worker %d action completed: %s, duration: %v",
			ew.id, action.Action, duration)
	}

	// 调用完成处理器
	if ew.completionHandler != nil {
		ew.completionHandler(action, err)
	}
}
