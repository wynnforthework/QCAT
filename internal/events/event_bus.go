package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// EventType 事件类型
type EventType string

const (
	// 工作流事件
	EventWorkflowStarted   EventType = "workflow.started"
	EventWorkflowCompleted EventType = "workflow.completed"
	EventWorkflowFailed    EventType = "workflow.failed"

	// 功能执行事件
	EventFunctionStarted   EventType = "function.started"
	EventFunctionCompleted EventType = "function.completed"
	EventFunctionFailed    EventType = "function.failed"
	EventFunctionSkipped   EventType = "function.skipped"

	// 依赖事件
	EventDependencyMet    EventType = "dependency.met"
	EventDependencyFailed EventType = "dependency.failed"

	// 冲突事件
	EventConflictDetected EventType = "conflict.detected"
	EventConflictResolved EventType = "conflict.resolved"

	// 资源事件
	EventResourceAcquired  EventType = "resource.acquired"
	EventResourceReleased  EventType = "resource.released"
	EventResourceExhausted EventType = "resource.exhausted"

	// 互锁事件
	EventInterlockGranted  EventType = "interlock.granted"
	EventInterlockReleased EventType = "interlock.released"
	EventInterlockBlocked  EventType = "interlock.blocked"

	// 系统事件
	EventSystemAlert    EventType = "system.alert"
	EventSystemError    EventType = "system.error"
	EventSystemRecovery EventType = "system.recovery"

	// 自定义事件
	EventCustom EventType = "custom"
)

// EventPriority 事件优先级
type EventPriority int

const (
	PriorityLow      EventPriority = 1
	PriorityNormal   EventPriority = 5
	PriorityHigh     EventPriority = 8
	PriorityCritical EventPriority = 10
)

// Event 事件结构
type Event struct {
	ID            string                 `json:"id"`
	Type          EventType              `json:"type"`
	Source        string                 `json:"source"`
	Target        string                 `json:"target,omitempty"`
	Priority      EventPriority          `json:"priority"`
	Timestamp     time.Time              `json:"timestamp"`
	Data          map[string]interface{} `json:"data"`
	Metadata      map[string]string      `json:"metadata"`
	CorrelationID string                 `json:"correlation_id,omitempty"`

	// 内部字段
	processed  bool
	retryCount int
	maxRetries int
}

// EventHandler 事件处理器接口
type EventHandler interface {
	Handle(ctx context.Context, event *Event) error
	GetName() string
	GetEventTypes() []EventType
	GetPriority() int
}

// EventFilter 事件过滤器
type EventFilter func(event *Event) bool

// EventMiddleware 事件中间件
type EventMiddleware func(next EventHandler) EventHandler

// EventSubscription 事件订阅
type EventSubscription struct {
	ID         string       `json:"id"`
	EventTypes []EventType  `json:"event_types"`
	Handler    EventHandler `json:"-"`
	Filter     EventFilter  `json:"-"`
	CreatedAt  time.Time    `json:"created_at"`
	Active     bool         `json:"active"`
}

// EventBus 事件总线
type EventBus struct {
	// 事件通道
	eventChan chan *Event

	// 订阅管理
	subscriptions map[string]*EventSubscription
	typeHandlers  map[EventType][]*EventSubscription

	// 中间件
	middlewares []EventMiddleware

	// 事件存储
	eventStore EventStore

	// 统计信息
	stats *EventStats

	// 同步控制
	mu      sync.RWMutex
	statsMu sync.RWMutex

	// 运行控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// 配置
	config *EventBusConfig
}

// EventBusConfig 事件总线配置
type EventBusConfig struct {
	BufferSize       int           `json:"buffer_size"`
	WorkerCount      int           `json:"worker_count"`
	MaxRetries       int           `json:"max_retries"`
	RetryDelay       time.Duration `json:"retry_delay"`
	EnableStorage    bool          `json:"enable_storage"`
	StorageRetention time.Duration `json:"storage_retention"`
}

// EventStats 事件统计
type EventStats struct {
	TotalEvents     int64         `json:"total_events"`
	ProcessedEvents int64         `json:"processed_events"`
	FailedEvents    int64         `json:"failed_events"`
	RetryEvents     int64         `json:"retry_events"`
	AverageLatency  time.Duration `json:"average_latency"`
	MaxLatency      time.Duration `json:"max_latency"`
	ActiveHandlers  int           `json:"active_handlers"`
	QueueLength     int           `json:"queue_length"`
}

// EventStore 事件存储接口
type EventStore interface {
	Store(event *Event) error
	Retrieve(id string) (*Event, error)
	Query(filter map[string]interface{}) ([]*Event, error)
	Delete(id string) error
	Cleanup(before time.Time) error
}

// NewEventBus 创建事件总线
func NewEventBus(config *EventBusConfig) *EventBus {
	if config == nil {
		config = &EventBusConfig{
			BufferSize:       10000,
			WorkerCount:      10,
			MaxRetries:       3,
			RetryDelay:       time.Second,
			EnableStorage:    true,
			StorageRetention: 7 * 24 * time.Hour,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	bus := &EventBus{
		eventChan:     make(chan *Event, config.BufferSize),
		subscriptions: make(map[string]*EventSubscription),
		typeHandlers:  make(map[EventType][]*EventSubscription),
		middlewares:   make([]EventMiddleware, 0),
		stats:         &EventStats{},
		ctx:           ctx,
		cancel:        cancel,
		config:        config,
	}

	// 初始化事件存储
	if config.EnableStorage {
		bus.eventStore = NewMemoryEventStore()
	}

	// 启动工作协程
	for i := 0; i < config.WorkerCount; i++ {
		bus.wg.Add(1)
		go bus.worker(i)
	}

	// 启动清理协程
	if config.EnableStorage {
		bus.wg.Add(1)
		go bus.cleaner()
	}

	return bus
}

// Subscribe 订阅事件
func (eb *EventBus) Subscribe(eventTypes []EventType, handler EventHandler, filter EventFilter) (string, error) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	subscriptionID := fmt.Sprintf("sub_%d", time.Now().UnixNano())

	subscription := &EventSubscription{
		ID:         subscriptionID,
		EventTypes: eventTypes,
		Handler:    handler,
		Filter:     filter,
		CreatedAt:  time.Now(),
		Active:     true,
	}

	eb.subscriptions[subscriptionID] = subscription

	// 更新类型处理器映射
	for _, eventType := range eventTypes {
		eb.typeHandlers[eventType] = append(eb.typeHandlers[eventType], subscription)
	}

	eb.statsMu.Lock()
	eb.stats.ActiveHandlers++
	eb.statsMu.Unlock()

	log.Printf("订阅事件: %s -> %v", handler.GetName(), eventTypes)

	return subscriptionID, nil
}

// Unsubscribe 取消订阅
func (eb *EventBus) Unsubscribe(subscriptionID string) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	subscription, exists := eb.subscriptions[subscriptionID]
	if !exists {
		return fmt.Errorf("subscription %s not found", subscriptionID)
	}

	// 从类型处理器映射中移除
	for _, eventType := range subscription.EventTypes {
		handlers := eb.typeHandlers[eventType]
		for i, sub := range handlers {
			if sub.ID == subscriptionID {
				eb.typeHandlers[eventType] = append(handlers[:i], handlers[i+1:]...)
				break
			}
		}
	}

	delete(eb.subscriptions, subscriptionID)

	eb.statsMu.Lock()
	eb.stats.ActiveHandlers--
	eb.statsMu.Unlock()

	log.Printf("取消订阅: %s", subscription.Handler.GetName())

	return nil
}

// Publish 发布事件
func (eb *EventBus) Publish(event *Event) error {
	if event.ID == "" {
		event.ID = fmt.Sprintf("event_%d", time.Now().UnixNano())
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	if event.Priority == 0 {
		event.Priority = PriorityNormal
	}

	if event.maxRetries == 0 {
		event.maxRetries = eb.config.MaxRetries
	}

	// 更新统计
	eb.statsMu.Lock()
	eb.stats.TotalEvents++
	eb.stats.QueueLength = len(eb.eventChan)
	eb.statsMu.Unlock()

	// 存储事件
	if eb.eventStore != nil {
		if err := eb.eventStore.Store(event); err != nil {
			log.Printf("存储事件失败: %v", err)
		}
	}

	// 发送到处理队列
	select {
	case eb.eventChan <- event:
		return nil
	default:
		return fmt.Errorf("event bus is full")
	}
}

// PublishSync 同步发布事件
func (eb *EventBus) PublishSync(ctx context.Context, event *Event) error {
	if err := eb.Publish(event); err != nil {
		return err
	}

	// 等待事件处理完成
	// 这里简化实现，实际应该有更复杂的同步机制
	time.Sleep(10 * time.Millisecond)

	return nil
}

// AddMiddleware 添加中间件
func (eb *EventBus) AddMiddleware(middleware EventMiddleware) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.middlewares = append(eb.middlewares, middleware)
}

// worker 工作协程
func (eb *EventBus) worker(id int) {
	defer eb.wg.Done()

	log.Printf("事件总线工作协程 %d 启动", id)

	for {
		select {
		case <-eb.ctx.Done():
			log.Printf("事件总线工作协程 %d 停止", id)
			return
		case event := <-eb.eventChan:
			eb.processEvent(event)
		}
	}
}

// processEvent 处理事件
func (eb *EventBus) processEvent(event *Event) {
	startTime := time.Now()

	eb.mu.RLock()
	handlers := eb.typeHandlers[event.Type]
	eb.mu.RUnlock()

	if len(handlers) == 0 {
		log.Printf("没有找到事件处理器: %s", event.Type)
		return
	}

	// 并发处理所有匹配的处理器
	var wg sync.WaitGroup

	for _, subscription := range handlers {
		if !subscription.Active {
			continue
		}

		// 应用过滤器
		if subscription.Filter != nil && !subscription.Filter(event) {
			continue
		}

		wg.Add(1)
		go func(sub *EventSubscription) {
			defer wg.Done()

			handler := sub.Handler

			// 应用中间件
			for i := len(eb.middlewares) - 1; i >= 0; i-- {
				handler = eb.middlewares[i](handler)
			}

			// 执行处理器
			ctx, cancel := context.WithTimeout(eb.ctx, 30*time.Second)
			defer cancel()

			if err := handler.Handle(ctx, event); err != nil {
				log.Printf("事件处理失败: %s -> %s: %v",
					event.Type, sub.Handler.GetName(), err)

				// 重试逻辑
				if event.retryCount < event.maxRetries {
					event.retryCount++

					eb.statsMu.Lock()
					eb.stats.RetryEvents++
					eb.statsMu.Unlock()

					// 延迟重试
					go func() {
						time.Sleep(eb.config.RetryDelay * time.Duration(event.retryCount))
						eb.eventChan <- event
					}()
				} else {
					eb.statsMu.Lock()
					eb.stats.FailedEvents++
					eb.statsMu.Unlock()
				}
			}
		}(subscription)
	}

	wg.Wait()

	// 更新统计
	latency := time.Since(startTime)
	eb.statsMu.Lock()
	eb.stats.ProcessedEvents++
	if latency > eb.stats.MaxLatency {
		eb.stats.MaxLatency = latency
	}
	// 更新平均延迟
	if eb.stats.ProcessedEvents > 0 {
		eb.stats.AverageLatency = time.Duration(
			(int64(eb.stats.AverageLatency)*(eb.stats.ProcessedEvents-1) + int64(latency)) /
				eb.stats.ProcessedEvents)
	}
	eb.statsMu.Unlock()

	event.processed = true
}

// cleaner 清理协程
func (eb *EventBus) cleaner() {
	defer eb.wg.Done()

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-eb.ctx.Done():
			return
		case <-ticker.C:
			if eb.eventStore != nil {
				cutoff := time.Now().Add(-eb.config.StorageRetention)
				if err := eb.eventStore.Cleanup(cutoff); err != nil {
					log.Printf("清理事件存储失败: %v", err)
				}
			}
		}
	}
}

// GetStats 获取统计信息
func (eb *EventBus) GetStats() *EventStats {
	eb.statsMu.RLock()
	defer eb.statsMu.RUnlock()

	return &EventStats{
		TotalEvents:     eb.stats.TotalEvents,
		ProcessedEvents: eb.stats.ProcessedEvents,
		FailedEvents:    eb.stats.FailedEvents,
		RetryEvents:     eb.stats.RetryEvents,
		AverageLatency:  eb.stats.AverageLatency,
		MaxLatency:      eb.stats.MaxLatency,
		ActiveHandlers:  eb.stats.ActiveHandlers,
		QueueLength:     len(eb.eventChan),
	}
}

// GetSubscriptions 获取订阅列表
func (eb *EventBus) GetSubscriptions() map[string]*EventSubscription {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	subscriptions := make(map[string]*EventSubscription)
	for id, sub := range eb.subscriptions {
		subscriptions[id] = sub
	}

	return subscriptions
}

// Stop 停止事件总线
func (eb *EventBus) Stop() {
	eb.cancel()
	eb.wg.Wait()
	close(eb.eventChan)
	log.Println("事件总线已停止")
}

// MemoryEventStore 内存事件存储
type MemoryEventStore struct {
	events map[string]*Event
	mu     sync.RWMutex
}

// NewMemoryEventStore 创建内存事件存储
func NewMemoryEventStore() *MemoryEventStore {
	return &MemoryEventStore{
		events: make(map[string]*Event),
	}
}

// Store 存储事件
func (mes *MemoryEventStore) Store(event *Event) error {
	mes.mu.Lock()
	defer mes.mu.Unlock()

	// 深拷贝事件
	eventData, err := json.Marshal(event)
	if err != nil {
		return err
	}

	var storedEvent Event
	if err := json.Unmarshal(eventData, &storedEvent); err != nil {
		return err
	}

	mes.events[event.ID] = &storedEvent
	return nil
}

// Retrieve 检索事件
func (mes *MemoryEventStore) Retrieve(id string) (*Event, error) {
	mes.mu.RLock()
	defer mes.mu.RUnlock()

	event, exists := mes.events[id]
	if !exists {
		return nil, fmt.Errorf("event %s not found", id)
	}

	return event, nil
}

// Query 查询事件
func (mes *MemoryEventStore) Query(filter map[string]interface{}) ([]*Event, error) {
	mes.mu.RLock()
	defer mes.mu.RUnlock()

	var results []*Event

	for _, event := range mes.events {
		if mes.matchesFilter(event, filter) {
			results = append(results, event)
		}
	}

	return results, nil
}

// Delete 删除事件
func (mes *MemoryEventStore) Delete(id string) error {
	mes.mu.Lock()
	defer mes.mu.Unlock()

	if _, exists := mes.events[id]; !exists {
		return fmt.Errorf("event %s not found", id)
	}

	delete(mes.events, id)
	return nil
}

// Cleanup 清理过期事件
func (mes *MemoryEventStore) Cleanup(before time.Time) error {
	mes.mu.Lock()
	defer mes.mu.Unlock()

	var toDelete []string

	for id, event := range mes.events {
		if event.Timestamp.Before(before) {
			toDelete = append(toDelete, id)
		}
	}

	for _, id := range toDelete {
		delete(mes.events, id)
	}

	log.Printf("清理了 %d 个过期事件", len(toDelete))
	return nil
}

// matchesFilter 检查事件是否匹配过滤器
func (mes *MemoryEventStore) matchesFilter(event *Event, filter map[string]interface{}) bool {
	for key, value := range filter {
		switch key {
		case "type":
			if event.Type != EventType(value.(string)) {
				return false
			}
		case "source":
			if event.Source != value.(string) {
				return false
			}
		case "priority":
			if event.Priority != EventPriority(value.(int)) {
				return false
			}
		case "after":
			if after, ok := value.(time.Time); ok {
				if event.Timestamp.Before(after) {
					return false
				}
			}
		case "before":
			if before, ok := value.(time.Time); ok {
				if event.Timestamp.After(before) {
					return false
				}
			}
		}
	}

	return true
}
