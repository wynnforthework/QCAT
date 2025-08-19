package bridge

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/automation/executor"
	"qcat/internal/config"
	"qcat/internal/monitor"
)

// MonitorResponseBridge 监控-响应桥接器
// 连接监控系统与自动响应机制，实现事件驱动的自动化
type MonitorResponseBridge struct {
	config   *config.Config
	executor *executor.RealtimeExecutor
	metrics  *monitor.MetricsCollector

	// 事件处理
	eventQueue    chan *MonitorEvent
	responseRules map[string]*ResponseRule
	workers       []*ResponseWorker

	// 运行状态
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	isRunning bool
	mu        sync.RWMutex

	// 统计信息
	stats *BridgeStats
}

// MonitorEvent 监控事件
type MonitorEvent struct {
	ID          string                 `json:"id"`
	Type        EventType              `json:"type"`
	Source      string                 `json:"source"`
	Severity    EventSeverity          `json:"severity"`
	Message     string                 `json:"message"`
	Metadata    map[string]interface{} `json:"metadata"`
	Timestamp   time.Time              `json:"timestamp"`
	ProcessedAt time.Time              `json:"processed_at"`
}

// EventType 事件类型
type EventType string

const (
	EventTypeAlert         EventType = "alert"
	EventTypeRiskViolation EventType = "risk_violation"
	EventTypePerformance   EventType = "performance"
	EventTypeSystem        EventType = "system"
	EventTypeStrategy      EventType = "strategy"
	EventTypeMarket        EventType = "market"
	EventTypeTrading       EventType = "trading"
	EventTypeMaintenance   EventType = "maintenance"
	EventTypeAudit         EventType = "audit"
)

// EventSeverity 事件严重程度
type EventSeverity string

const (
	SeverityInfo      EventSeverity = "info"
	SeverityWarning   EventSeverity = "warning"
	SeverityCritical  EventSeverity = "critical"
	SeverityEmergency EventSeverity = "emergency"
)

// ResponseRule 响应规则
type ResponseRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	EventType   EventType              `json:"event_type"`
	Conditions  []RuleCondition        `json:"conditions"`
	Actions     []ResponseAction       `json:"actions"`
	Enabled     bool                   `json:"enabled"`
	Priority    int                    `json:"priority"`
	Cooldown    time.Duration          `json:"cooldown"`
	LastTrigger time.Time              `json:"last_trigger"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// RuleCondition 规则条件
type RuleCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // eq, ne, gt, lt, gte, lte, contains
	Value    interface{} `json:"value"`
}

// ResponseAction 响应动作
type ResponseAction struct {
	Type       ActionType             `json:"type"`
	Action     string                 `json:"action"`
	Parameters map[string]interface{} `json:"parameters"`
	Timeout    time.Duration          `json:"timeout"`
	MaxRetries int                    `json:"max_retries"`
}

// ActionType 动作类型
type ActionType string

const (
	ActionTypePosition ActionType = "position"
	ActionTypeRisk     ActionType = "risk"
	ActionTypeOrder    ActionType = "order"
	ActionTypeSystem   ActionType = "system"
	ActionTypeAlert    ActionType = "alert"
	ActionTypeStrategy ActionType = "strategy"
	ActionTypeTrading  ActionType = "trading"
)

// BridgeStats 桥接器统计
type BridgeStats struct {
	TotalEvents     int64
	ProcessedEvents int64
	FailedEvents    int64
	TriggeredRules  int64
	ExecutedActions int64
	FailedActions   int64
	AverageLatency  time.Duration
	LastEventTime   time.Time
	mu              sync.RWMutex
}

// ResponseWorker 响应工作器
type ResponseWorker struct {
	id        int
	bridge    *MonitorResponseBridge
	eventCh   chan *MonitorEvent
	stopCh    chan struct{}
	isRunning bool
	mu        sync.RWMutex
}

// NewMonitorResponseBridge 创建监控-响应桥接器
func NewMonitorResponseBridge(
	cfg *config.Config,
	executor *executor.RealtimeExecutor,
	metrics *monitor.MetricsCollector,
) *MonitorResponseBridge {
	ctx, cancel := context.WithCancel(context.Background())

	bridge := &MonitorResponseBridge{
		config:        cfg,
		executor:      executor,
		metrics:       metrics,
		eventQueue:    make(chan *MonitorEvent, 10000),
		responseRules: make(map[string]*ResponseRule),
		workers:       make([]*ResponseWorker, 0),
		ctx:           ctx,
		cancel:        cancel,
		stats:         &BridgeStats{},
	}

	// 使用增强的事件规则管理器初始化规则
	ruleManager := NewEventRuleManager(bridge)
	ruleManager.InitializeEnhancedRules()

	// 验证规则
	if err := ruleManager.ValidateRules(); err != nil {
		log.Printf("Warning: Rule validation failed: %v", err)
	}

	// 初始化工作器
	bridge.initializeWorkers()

	return bridge
}

// Start 启动桥接器
func (mrb *MonitorResponseBridge) Start() error {
	mrb.mu.Lock()
	defer mrb.mu.Unlock()

	if mrb.isRunning {
		return fmt.Errorf("monitor response bridge is already running")
	}

	log.Println("Starting monitor response bridge...")

	// 启动工作器
	for _, worker := range mrb.workers {
		mrb.wg.Add(1)
		go worker.Start(&mrb.wg)
	}

	// 启动事件分发器
	mrb.wg.Add(1)
	go mrb.eventDispatcher()

	// 启动统计更新器
	mrb.wg.Add(1)
	go mrb.statsUpdater()

	mrb.isRunning = true
	log.Println("Monitor response bridge started successfully")

	return nil
}

// Stop 停止桥接器
func (mrb *MonitorResponseBridge) Stop() error {
	mrb.mu.Lock()
	defer mrb.mu.Unlock()

	if !mrb.isRunning {
		return nil
	}

	log.Println("Stopping monitor response bridge...")

	// 取消上下文
	mrb.cancel()

	// 停止工作器
	for _, worker := range mrb.workers {
		worker.Stop()
	}

	// 等待所有goroutine完成
	mrb.wg.Wait()

	// 关闭事件队列
	close(mrb.eventQueue)

	mrb.isRunning = false
	log.Println("Monitor response bridge stopped")

	return nil
}

// ProcessEvent 处理监控事件
func (mrb *MonitorResponseBridge) ProcessEvent(event *MonitorEvent) error {
	if event.ID == "" {
		event.ID = fmt.Sprintf("event_%d", time.Now().UnixNano())
	}
	event.Timestamp = time.Now()

	// 更新统计
	mrb.stats.mu.Lock()
	mrb.stats.TotalEvents++
	mrb.stats.LastEventTime = time.Now()
	mrb.stats.mu.Unlock()

	// 加入事件队列
	select {
	case mrb.eventQueue <- event:
		log.Printf("Event queued: %s (%s)", event.Type, event.ID)
		return nil
	default:
		mrb.stats.mu.Lock()
		mrb.stats.FailedEvents++
		mrb.stats.mu.Unlock()
		return fmt.Errorf("event queue is full")
	}
}

// eventDispatcher 事件分发器
func (mrb *MonitorResponseBridge) eventDispatcher() {
	defer mrb.wg.Done()

	for {
		select {
		case <-mrb.ctx.Done():
			return
		case event := <-mrb.eventQueue:
			mrb.dispatchEvent(event)
		}
	}
}

// dispatchEvent 分发事件到工作器
func (mrb *MonitorResponseBridge) dispatchEvent(event *MonitorEvent) {
	// 选择负载最轻的工作器
	selectedWorker := mrb.selectWorker()
	if selectedWorker != nil {
		select {
		case selectedWorker.eventCh <- event:
			log.Printf("Event dispatched to worker %d: %s", selectedWorker.id, event.ID)
		default:
			log.Printf("Worker %d is busy, event dropped: %s", selectedWorker.id, event.ID)
			mrb.stats.mu.Lock()
			mrb.stats.FailedEvents++
			mrb.stats.mu.Unlock()
		}
	}
}

// selectWorker 选择工作器
func (mrb *MonitorResponseBridge) selectWorker() *ResponseWorker {
	// 简单的轮询选择
	if len(mrb.workers) == 0 {
		return nil
	}

	for _, worker := range mrb.workers {
		if len(worker.eventCh) < cap(worker.eventCh)/2 {
			return worker
		}
	}

	// 如果所有工作器都很忙，选择第一个
	return mrb.workers[0]
}

// statsUpdater 统计更新器
func (mrb *MonitorResponseBridge) statsUpdater() {
	defer mrb.wg.Done()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-mrb.ctx.Done():
			return
		case <-ticker.C:
			mrb.updateStats()
		}
	}
}

// updateStats 更新统计信息
func (mrb *MonitorResponseBridge) updateStats() {
	mrb.stats.mu.Lock()
	defer mrb.stats.mu.Unlock()

	// 计算处理率
	if mrb.stats.TotalEvents > 0 {
		successRate := float64(mrb.stats.ProcessedEvents) / float64(mrb.stats.TotalEvents)
		log.Printf("Bridge processing rate: %.2f%% (%d/%d)",
			successRate*100, mrb.stats.ProcessedEvents, mrb.stats.TotalEvents)
	}

	// 记录到metrics
	if mrb.metrics != nil {
		// 使用现有的metrics方法记录统计信息
		labels := map[string]string{
			"component": "monitor_bridge",
		}
		mrb.metrics.IncrementCounter("bridge_events", labels)
	}
}

// initializeDefaultRules 初始化默认响应规则
func (mrb *MonitorResponseBridge) initializeDefaultRules() {
	// 1. 风险违规自动响应
	mrb.AddResponseRule(&ResponseRule{
		ID:        "risk_violation_emergency",
		Name:      "风险违规紧急响应",
		EventType: EventTypeRiskViolation,
		Conditions: []RuleCondition{
			{Field: "severity", Operator: "eq", Value: "emergency"},
		},
		Actions: []ResponseAction{
			{
				Type:       ActionTypeRisk,
				Action:     "emergency_stop",
				Parameters: map[string]interface{}{},
				Timeout:    time.Minute * 2,
				MaxRetries: 1,
			},
		},
		Enabled:  true,
		Priority: 1,
		Cooldown: time.Minute * 5,
	})

	// 2. 策略性能下降自动优化
	mrb.AddResponseRule(&ResponseRule{
		ID:        "strategy_performance_decline",
		Name:      "策略性能下降自动优化",
		EventType: EventTypeStrategy,
		Conditions: []RuleCondition{
			{Field: "metric", Operator: "eq", Value: "sharpe_ratio"},
			{Field: "value", Operator: "lt", Value: 0.5},
		},
		Actions: []ResponseAction{
			{
				Type:   ActionTypePosition,
				Action: "trigger_optimization",
				Parameters: map[string]interface{}{
					"strategy_id": "{{metadata.strategy_id}}",
				},
				Timeout:    time.Hour,
				MaxRetries: 2,
			},
		},
		Enabled:  true,
		Priority: 3,
		Cooldown: time.Hour * 6,
	})

	// 3. 系统资源告警自动处理
	mrb.AddResponseRule(&ResponseRule{
		ID:        "system_resource_alert",
		Name:      "系统资源告警自动处理",
		EventType: EventTypeSystem,
		Conditions: []RuleCondition{
			{Field: "metric", Operator: "eq", Value: "memory_usage"},
			{Field: "value", Operator: "gt", Value: 0.9},
		},
		Actions: []ResponseAction{
			{
				Type:       ActionTypeSystem,
				Action:     "cleanup_memory",
				Parameters: map[string]interface{}{},
				Timeout:    time.Minute * 5,
				MaxRetries: 1,
			},
		},
		Enabled:  true,
		Priority: 2,
		Cooldown: time.Minute * 10,
	})

	// 4. 市场异常自动对冲
	mrb.AddResponseRule(&ResponseRule{
		ID:        "market_anomaly_hedge",
		Name:      "市场异常自动对冲",
		EventType: EventTypeMarket,
		Conditions: []RuleCondition{
			{Field: "volatility", Operator: "gt", Value: 0.05},
			{Field: "severity", Operator: "eq", Value: "critical"},
		},
		Actions: []ResponseAction{
			{
				Type:   ActionTypePosition,
				Action: "hedge_position",
				Parameters: map[string]interface{}{
					"hedge_ratio": 0.5,
				},
				Timeout:    time.Minute * 10,
				MaxRetries: 2,
			},
		},
		Enabled:  true,
		Priority: 2,
		Cooldown: time.Minute * 30,
	})

	log.Printf("Initialized %d default response rules", len(mrb.responseRules))
}

// initializeWorkers 初始化工作器
func (mrb *MonitorResponseBridge) initializeWorkers() {
	workerCount := 3 // 可配置
	for i := 0; i < workerCount; i++ {
		worker := &ResponseWorker{
			id:      i,
			bridge:  mrb,
			eventCh: make(chan *MonitorEvent, 100),
			stopCh:  make(chan struct{}),
		}
		mrb.workers = append(mrb.workers, worker)
	}
}

// AddResponseRule 添加响应规则
func (mrb *MonitorResponseBridge) AddResponseRule(rule *ResponseRule) {
	mrb.mu.Lock()
	defer mrb.mu.Unlock()
	mrb.responseRules[rule.ID] = rule
	log.Printf("Added response rule: %s", rule.Name)
}

// RemoveResponseRule 移除响应规则
func (mrb *MonitorResponseBridge) RemoveResponseRule(ruleID string) {
	mrb.mu.Lock()
	defer mrb.mu.Unlock()
	delete(mrb.responseRules, ruleID)
	log.Printf("Removed response rule: %s", ruleID)
}

// GetStats 获取统计信息
func (mrb *MonitorResponseBridge) GetStats() *BridgeStats {
	mrb.stats.mu.RLock()
	defer mrb.stats.mu.RUnlock()

	return &BridgeStats{
		TotalEvents:     mrb.stats.TotalEvents,
		ProcessedEvents: mrb.stats.ProcessedEvents,
		FailedEvents:    mrb.stats.FailedEvents,
		TriggeredRules:  mrb.stats.TriggeredRules,
		ExecutedActions: mrb.stats.ExecutedActions,
		FailedActions:   mrb.stats.FailedActions,
		AverageLatency:  mrb.stats.AverageLatency,
		LastEventTime:   mrb.stats.LastEventTime,
	}
}
