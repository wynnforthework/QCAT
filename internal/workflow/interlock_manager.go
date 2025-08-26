package workflow

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// InterlockType 互锁类型
type InterlockType string

const (
	InterlockTypeMutex      InterlockType = "mutex"       // 互斥锁
	InterlockTypeSemaphore  InterlockType = "semaphore"   // 信号量
	InterlockTypeTimeWindow InterlockType = "time_window" // 时间窗口
	InterlockTypeResource   InterlockType = "resource"    // 资源锁
)

// InterlockStatus 互锁状态
type InterlockStatus string

const (
	InterlockStatusActive   InterlockStatus = "active"   // 激活
	InterlockStatusBlocked  InterlockStatus = "blocked"  // 阻塞
	InterlockStatusWaiting  InterlockStatus = "waiting"  // 等待
	InterlockStatusReleased InterlockStatus = "released" // 释放
)

// InterlockRule 互锁规则
type InterlockRule struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Type          InterlockType `json:"type"`
	FunctionIDs   []int         `json:"function_ids"`
	MaxConcurrent int           `json:"max_concurrent"`
	Priority      int           `json:"priority"`
	Timeout       time.Duration `json:"timeout"`

	// 时间窗口相关
	TimeWindows []TimeWindow `json:"time_windows,omitempty"`

	// 资源相关
	ResourceType  string `json:"resource_type,omitempty"`
	ResourceLimit int    `json:"resource_limit,omitempty"`

	// 状态
	Status    InterlockStatus `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// TimeWindow 时间窗口
type TimeWindow struct {
	Start    string `json:"start"`    // HH:MM 格式
	End      string `json:"end"`      // HH:MM 格式
	Timezone string `json:"timezone"` // 时区
	Days     []int  `json:"days"`     // 星期几 (0=Sunday, 1=Monday, ...)
}

// InterlockRequest 互锁请求
type InterlockRequest struct {
	ID          string          `json:"id"`
	FunctionID  int             `json:"function_id"`
	RuleID      string          `json:"rule_id"`
	RequestedAt time.Time       `json:"requested_at"`
	Timeout     time.Duration   `json:"timeout"`
	Priority    int             `json:"priority"`
	Context     context.Context `json:"-"`
}

// InterlockGrant 互锁授权
type InterlockGrant struct {
	ID         string    `json:"id"`
	RequestID  string    `json:"request_id"`
	FunctionID int       `json:"function_id"`
	RuleID     string    `json:"rule_id"`
	GrantedAt  time.Time `json:"granted_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// InterlockManager 互锁管理器
type InterlockManager struct {
	rules     map[string]*InterlockRule
	grants    map[string]*InterlockGrant
	requests  map[string]*InterlockRequest
	waitQueue []*InterlockRequest

	// 同步控制
	mu sync.RWMutex

	// 统计信息
	stats   *InterlockStats
	statsMu sync.RWMutex

	// 事件通知
	eventChan chan *InterlockEvent

	// 运行控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// InterlockStats 互锁统计信息
type InterlockStats struct {
	TotalRequests   int64         `json:"total_requests"`
	GrantedRequests int64         `json:"granted_requests"`
	BlockedRequests int64         `json:"blocked_requests"`
	TimeoutRequests int64         `json:"timeout_requests"`
	AverageWaitTime time.Duration `json:"average_wait_time"`
	MaxWaitTime     time.Duration `json:"max_wait_time"`
	ActiveGrants    int           `json:"active_grants"`
	QueueLength     int           `json:"queue_length"`
}

// InterlockEvent 互锁事件
type InterlockEvent struct {
	Type      string      `json:"type"`
	RuleID    string      `json:"rule_id"`
	RequestID string      `json:"request_id"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// NewInterlockManager 创建互锁管理器
func NewInterlockManager() *InterlockManager {
	ctx, cancel := context.WithCancel(context.Background())

	im := &InterlockManager{
		rules:     make(map[string]*InterlockRule),
		grants:    make(map[string]*InterlockGrant),
		requests:  make(map[string]*InterlockRequest),
		waitQueue: make([]*InterlockRequest, 0),
		stats:     &InterlockStats{},
		eventChan: make(chan *InterlockEvent, 1000),
		ctx:       ctx,
		cancel:    cancel,
	}

	// 启动后台处理
	im.wg.Add(1)
	go im.processRequests()

	im.wg.Add(1)
	go im.cleanupExpiredGrants()

	return im
}

// AddRule 添加互锁规则
func (im *InterlockManager) AddRule(rule *InterlockRule) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	if rule.ID == "" {
		return fmt.Errorf("rule ID cannot be empty")
	}

	if _, exists := im.rules[rule.ID]; exists {
		return fmt.Errorf("rule %s already exists", rule.ID)
	}

	rule.Status = InterlockStatusActive
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	im.rules[rule.ID] = rule

	log.Printf("添加互锁规则: %s (%s)", rule.Name, rule.ID)

	// 发送事件
	im.emitEvent("rule_added", rule.ID, "", rule)

	return nil
}

// RemoveRule 移除互锁规则
func (im *InterlockManager) RemoveRule(ruleID string) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	rule, exists := im.rules[ruleID]
	if !exists {
		return fmt.Errorf("rule %s not found", ruleID)
	}

	// 检查是否有活跃的授权
	activeGrants := 0
	for _, grant := range im.grants {
		if grant.RuleID == ruleID {
			activeGrants++
		}
	}

	if activeGrants > 0 {
		return fmt.Errorf("cannot remove rule %s: %d active grants", ruleID, activeGrants)
	}

	delete(im.rules, ruleID)

	log.Printf("移除互锁规则: %s (%s)", rule.Name, ruleID)

	// 发送事件
	im.emitEvent("rule_removed", ruleID, "", rule)

	return nil
}

// RequestLock 请求互锁
func (im *InterlockManager) RequestLock(functionID int, ruleID string, timeout time.Duration) (*InterlockGrant, error) {
	im.mu.Lock()

	rule, exists := im.rules[ruleID]
	if !exists {
		im.mu.Unlock()
		return nil, fmt.Errorf("rule %s not found", ruleID)
	}

	// 检查功能是否在规则中
	functionInRule := false
	for _, id := range rule.FunctionIDs {
		if id == functionID {
			functionInRule = true
			break
		}
	}

	if !functionInRule {
		im.mu.Unlock()
		return nil, fmt.Errorf("function %d not in rule %s", functionID, ruleID)
	}

	// 创建请求
	request := &InterlockRequest{
		ID:          fmt.Sprintf("req_%d_%s_%d", functionID, ruleID, time.Now().UnixNano()),
		FunctionID:  functionID,
		RuleID:      ruleID,
		RequestedAt: time.Now(),
		Timeout:     timeout,
		Priority:    rule.Priority,
	}

	im.requests[request.ID] = request

	// 更新统计
	im.statsMu.Lock()
	im.stats.TotalRequests++
	im.statsMu.Unlock()

	im.mu.Unlock()

	// 尝试立即获取锁
	if grant := im.tryGrantLock(request); grant != nil {
		return grant, nil
	}

	// 加入等待队列
	im.mu.Lock()
	im.waitQueue = append(im.waitQueue, request)
	im.statsMu.Lock()
	im.stats.QueueLength = len(im.waitQueue)
	im.statsMu.Unlock()
	im.mu.Unlock()

	// 发送事件
	im.emitEvent("request_queued", ruleID, request.ID, request)

	// 等待授权或超时
	return im.waitForGrant(request)
}

// tryGrantLock 尝试授权锁
func (im *InterlockManager) tryGrantLock(request *InterlockRequest) *InterlockGrant {
	im.mu.Lock()
	defer im.mu.Unlock()

	rule := im.rules[request.RuleID]

	// 检查时间窗口
	if !im.isInTimeWindow(rule) {
		return nil
	}

	// 检查并发限制
	activeCount := im.getActiveGrantCount(request.RuleID)
	if activeCount >= rule.MaxConcurrent {
		return nil
	}

	// 创建授权
	grant := &InterlockGrant{
		ID:         fmt.Sprintf("grant_%s_%d", request.ID, time.Now().UnixNano()),
		RequestID:  request.ID,
		FunctionID: request.FunctionID,
		RuleID:     request.RuleID,
		GrantedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(rule.Timeout),
	}

	im.grants[grant.ID] = grant

	// 更新统计
	im.statsMu.Lock()
	im.stats.GrantedRequests++
	im.stats.ActiveGrants++
	im.statsMu.Unlock()

	// 发送事件
	im.emitEvent("lock_granted", request.RuleID, request.ID, grant)

	log.Printf("授权互锁: 功能 %d, 规则 %s, 授权 %s",
		request.FunctionID, request.RuleID, grant.ID)

	return grant
}

// ReleaseLock 释放互锁
func (im *InterlockManager) ReleaseLock(grantID string) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	grant, exists := im.grants[grantID]
	if !exists {
		return fmt.Errorf("grant %s not found", grantID)
	}

	delete(im.grants, grantID)
	delete(im.requests, grant.RequestID)

	// 更新统计
	im.statsMu.Lock()
	im.stats.ActiveGrants--
	im.statsMu.Unlock()

	// 发送事件
	im.emitEvent("lock_released", grant.RuleID, grant.RequestID, grant)

	log.Printf("释放互锁: 功能 %d, 规则 %s, 授权 %s",
		grant.FunctionID, grant.RuleID, grantID)

	// 处理等待队列
	go im.processWaitQueue()

	return nil
}

// isInTimeWindow 检查是否在时间窗口内
func (im *InterlockManager) isInTimeWindow(rule *InterlockRule) bool {
	if len(rule.TimeWindows) == 0 {
		return true // 没有时间窗口限制
	}

	now := time.Now()

	for _, window := range rule.TimeWindows {
		// 简化实现，实际应该考虑时区
		if im.isTimeInWindow(now, window) {
			return true
		}
	}

	return false
}

// isTimeInWindow 检查时间是否在窗口内
func (im *InterlockManager) isTimeInWindow(t time.Time, window TimeWindow) bool {
	// 简化实现
	hour := t.Hour()
	minute := t.Minute()
	currentTime := hour*60 + minute

	// 解析开始和结束时间
	// 这里简化处理，实际应该更严格地解析时间格式
	return true // 简化返回true
}

// getActiveGrantCount 获取活跃授权数量
func (im *InterlockManager) getActiveGrantCount(ruleID string) int {
	count := 0
	for _, grant := range im.grants {
		if grant.RuleID == ruleID && time.Now().Before(grant.ExpiresAt) {
			count++
		}
	}
	return count
}

// waitForGrant 等待授权
func (im *InterlockManager) waitForGrant(request *InterlockRequest) (*InterlockGrant, error) {
	timeout := time.After(request.Timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			// 超时，从队列中移除请求
			im.removeFromWaitQueue(request.ID)

			im.statsMu.Lock()
			im.stats.TimeoutRequests++
			im.statsMu.Unlock()

			im.emitEvent("request_timeout", request.RuleID, request.ID, request)

			return nil, fmt.Errorf("request timeout after %v", request.Timeout)

		case <-ticker.C:
			// 定期检查是否可以授权
			if grant := im.tryGrantLock(request); grant != nil {
				im.removeFromWaitQueue(request.ID)
				return grant, nil
			}
		}
	}
}

// removeFromWaitQueue 从等待队列中移除请求
func (im *InterlockManager) removeFromWaitQueue(requestID string) {
	im.mu.Lock()
	defer im.mu.Unlock()

	for i, req := range im.waitQueue {
		if req.ID == requestID {
			im.waitQueue = append(im.waitQueue[:i], im.waitQueue[i+1:]...)
			break
		}
	}

	im.statsMu.Lock()
	im.stats.QueueLength = len(im.waitQueue)
	im.statsMu.Unlock()
}

// processRequests 处理请求队列
func (im *InterlockManager) processRequests() {
	defer im.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-im.ctx.Done():
			return
		case <-ticker.C:
			im.processWaitQueue()
		}
	}
}

// processWaitQueue 处理等待队列
func (im *InterlockManager) processWaitQueue() {
	im.mu.Lock()
	defer im.mu.Unlock()

	// 按优先级排序等待队列
	// 这里简化实现，实际应该使用优先级队列

	for i := 0; i < len(im.waitQueue); i++ {
		request := im.waitQueue[i]

		if grant := im.tryGrantLock(request); grant != nil {
			// 从队列中移除
			im.waitQueue = append(im.waitQueue[:i], im.waitQueue[i+1:]...)
			i-- // 调整索引
		}
	}

	im.statsMu.Lock()
	im.stats.QueueLength = len(im.waitQueue)
	im.statsMu.Unlock()
}

// cleanupExpiredGrants 清理过期授权
func (im *InterlockManager) cleanupExpiredGrants() {
	defer im.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-im.ctx.Done():
			return
		case <-ticker.C:
			im.mu.Lock()
			now := time.Now()

			for grantID, grant := range im.grants {
				if now.After(grant.ExpiresAt) {
					delete(im.grants, grantID)
					delete(im.requests, grant.RequestID)

					im.statsMu.Lock()
					im.stats.ActiveGrants--
					im.statsMu.Unlock()

					im.emitEvent("grant_expired", grant.RuleID, grant.RequestID, grant)

					log.Printf("清理过期授权: %s", grantID)
				}
			}

			im.mu.Unlock()
		}
	}
}

// emitEvent 发送事件
func (im *InterlockManager) emitEvent(eventType, ruleID, requestID string, data interface{}) {
	event := &InterlockEvent{
		Type:      eventType,
		RuleID:    ruleID,
		RequestID: requestID,
		Data:      data,
		Timestamp: time.Now(),
	}

	select {
	case im.eventChan <- event:
		// 事件发送成功
	default:
		log.Printf("警告: 互锁事件通道已满，丢弃事件: %s", eventType)
	}
}

// GetStats 获取统计信息
func (im *InterlockManager) GetStats() *InterlockStats {
	im.statsMu.RLock()
	defer im.statsMu.RUnlock()

	// 返回副本
	return &InterlockStats{
		TotalRequests:   im.stats.TotalRequests,
		GrantedRequests: im.stats.GrantedRequests,
		BlockedRequests: im.stats.BlockedRequests,
		TimeoutRequests: im.stats.TimeoutRequests,
		AverageWaitTime: im.stats.AverageWaitTime,
		MaxWaitTime:     im.stats.MaxWaitTime,
		ActiveGrants:    im.stats.ActiveGrants,
		QueueLength:     im.stats.QueueLength,
	}
}

// GetRules 获取所有规则
func (im *InterlockManager) GetRules() map[string]*InterlockRule {
	im.mu.RLock()
	defer im.mu.RUnlock()

	rules := make(map[string]*InterlockRule)
	for id, rule := range im.rules {
		rules[id] = rule
	}

	return rules
}

// GetActiveGrants 获取活跃授权
func (im *InterlockManager) GetActiveGrants() map[string]*InterlockGrant {
	im.mu.RLock()
	defer im.mu.RUnlock()

	grants := make(map[string]*InterlockGrant)
	now := time.Now()

	for id, grant := range im.grants {
		if now.Before(grant.ExpiresAt) {
			grants[id] = grant
		}
	}

	return grants
}

// GetWaitQueue 获取等待队列
func (im *InterlockManager) GetWaitQueue() []*InterlockRequest {
	im.mu.RLock()
	defer im.mu.RUnlock()

	queue := make([]*InterlockRequest, len(im.waitQueue))
	copy(queue, im.waitQueue)

	return queue
}

// UpdateRule 更新互锁规则
func (im *InterlockManager) UpdateRule(ruleID string, updates map[string]interface{}) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	rule, exists := im.rules[ruleID]
	if !exists {
		return fmt.Errorf("rule %s not found", ruleID)
	}

	// 更新规则字段
	if name, ok := updates["name"].(string); ok {
		rule.Name = name
	}
	if maxConcurrent, ok := updates["max_concurrent"].(int); ok {
		rule.MaxConcurrent = maxConcurrent
	}
	if priority, ok := updates["priority"].(int); ok {
		rule.Priority = priority
	}
	if timeout, ok := updates["timeout"].(time.Duration); ok {
		rule.Timeout = timeout
	}

	rule.UpdatedAt = time.Now()

	// 发送事件
	im.emitEvent("rule_updated", ruleID, "", rule)

	log.Printf("更新互锁规则: %s (%s)", rule.Name, ruleID)

	return nil
}

// EnableRule 启用规则
func (im *InterlockManager) EnableRule(ruleID string) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	rule, exists := im.rules[ruleID]
	if !exists {
		return fmt.Errorf("rule %s not found", ruleID)
	}

	rule.Status = InterlockStatusActive
	rule.UpdatedAt = time.Now()

	im.emitEvent("rule_enabled", ruleID, "", rule)

	log.Printf("启用互锁规则: %s (%s)", rule.Name, ruleID)

	return nil
}

// DisableRule 禁用规则
func (im *InterlockManager) DisableRule(ruleID string) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	rule, exists := im.rules[ruleID]
	if !exists {
		return fmt.Errorf("rule %s not found", ruleID)
	}

	rule.Status = InterlockStatusBlocked
	rule.UpdatedAt = time.Now()

	im.emitEvent("rule_disabled", ruleID, "", rule)

	log.Printf("禁用互锁规则: %s (%s)", rule.Name, ruleID)

	return nil
}

// GetEventChannel 获取事件通道
func (im *InterlockManager) GetEventChannel() <-chan *InterlockEvent {
	return im.eventChan
}

// Stop 停止互锁管理器
func (im *InterlockManager) Stop() {
	im.cancel()
	im.wg.Wait()
	close(im.eventChan)
	log.Println("互锁管理器已停止")
}
