package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/events"
)

// FactorUpdate 因子更新数据
type FactorUpdate struct {
	FactorID      string                 `json:"factor_id"`
	FactorName    string                 `json:"factor_name"`
	FactorType    string                 `json:"factor_type"`
	Parameters    map[string]interface{} `json:"parameters"`
	Effectiveness float64                `json:"effectiveness"`
	UpdateTime    time.Time              `json:"update_time"`
	Version       string                 `json:"version"`
	IsActive      bool                   `json:"is_active"`
}

// FactorLibrarySnapshot 因子库快照
type FactorLibrarySnapshot struct {
	Version     string                   `json:"version"`
	UpdateTime  time.Time                `json:"update_time"`
	Factors     map[string]*FactorUpdate `json:"factors"`
	ActiveCount int                      `json:"active_count"`
	TotalCount  int                      `json:"total_count"`
}

// FactorSyncService 因子同步服务
type FactorSyncService struct {
	// 事件系统
	eventBus *events.EventBus

	// 因子库缓存
	factorLibrary map[string]*FactorUpdate
	libraryMu     sync.RWMutex

	// 同步状态
	lastSyncTime   time.Time
	syncVersion    string
	syncInProgress bool
	syncMu         sync.RWMutex

	// 配置
	config *FactorSyncConfig

	// 运行状态
	ctx       context.Context
	cancel    context.CancelFunc
	isRunning bool
	runningMu sync.RWMutex
	wg        sync.WaitGroup

	// 统计信息
	stats   *FactorSyncStats
	statsMu sync.RWMutex
}

// FactorSyncConfig 因子同步配置
type FactorSyncConfig struct {
	// 同步间隔
	SyncInterval time.Duration `yaml:"sync_interval"`

	// 事件监听配置
	EventTimeout time.Duration `yaml:"event_timeout"`
	MaxRetries   int           `yaml:"max_retries"`
	RetryDelay   time.Duration `yaml:"retry_delay"`

	// 缓存配置
	CacheSize int           `yaml:"cache_size"`
	CacheTTL  time.Duration `yaml:"cache_ttl"`

	// 同步策略
	FullSyncInterval time.Duration `yaml:"full_sync_interval"`
	EnableDeltaSync  bool          `yaml:"enable_delta_sync"`
}

// FactorSyncStats 因子同步统计
type FactorSyncStats struct {
	TotalSyncs       int64
	SuccessfulSyncs  int64
	FailedSyncs      int64
	LastSyncTime     time.Time
	LastSyncDuration time.Duration
	FactorsUpdated   int64
	EventsProcessed  int64
	CacheHitRate     float64
}

// NewFactorSyncService 创建因子同步服务
func NewFactorSyncService(eventBus *events.EventBus, config *FactorSyncConfig) *FactorSyncService {
	if config == nil {
		config = GetDefaultFactorSyncConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &FactorSyncService{
		eventBus:      eventBus,
		factorLibrary: make(map[string]*FactorUpdate),
		config:        config,
		ctx:           ctx,
		cancel:        cancel,
		stats: &FactorSyncStats{
			LastSyncTime: time.Now(),
		},
	}
}

// Start 启动因子同步服务
func (fss *FactorSyncService) Start() error {
	fss.runningMu.Lock()
	defer fss.runningMu.Unlock()

	if fss.isRunning {
		return fmt.Errorf("factor sync service is already running")
	}

	log.Println("启动因子同步服务...")

	// 订阅因子更新事件
	if err := fss.subscribeToFactorEvents(); err != nil {
		return fmt.Errorf("failed to subscribe to factor events: %w", err)
	}

	// 执行初始同步
	if err := fss.performFullSync(); err != nil {
		log.Printf("Warning: initial sync failed: %v", err)
	}

	// 启动定期同步
	fss.wg.Add(2)
	go fss.runPeriodicSync()
	go fss.runFullSyncScheduler()

	fss.isRunning = true

	log.Println("因子同步服务启动完成")
	return nil
}

// Stop 停止因子同步服务
func (fss *FactorSyncService) Stop() error {
	fss.runningMu.Lock()
	defer fss.runningMu.Unlock()

	if !fss.isRunning {
		return nil
	}

	log.Println("停止因子同步服务...")

	// 取消上下文
	fss.cancel()

	// 等待协程结束
	fss.wg.Wait()

	fss.isRunning = false

	log.Println("因子同步服务已停止")
	return nil
}

// subscribeToFactorEvents 订阅因子更新事件
func (fss *FactorSyncService) subscribeToFactorEvents() error {
	// 订阅因子库更新事件
	err := fss.eventBus.Subscribe("factor_library_updated", fss.handleFactorLibraryUpdate)
	if err != nil {
		return fmt.Errorf("failed to subscribe to factor_library_updated: %w", err)
	}

	// 订阅单个因子更新事件
	err = fss.eventBus.Subscribe("factor_updated", fss.handleFactorUpdate)
	if err != nil {
		return fmt.Errorf("failed to subscribe to factor_updated: %w", err)
	}

	// 订阅因子删除事件
	err = fss.eventBus.Subscribe("factor_deleted", fss.handleFactorDeleted)
	if err != nil {
		return fmt.Errorf("failed to subscribe to factor_deleted: %w", err)
	}

	log.Println("已订阅因子更新事件")
	return nil
}

// handleFactorLibraryUpdate 处理因子库更新事件
func (fss *FactorSyncService) handleFactorLibraryUpdate(event *events.Event) error {
	log.Printf("收到因子库更新事件: %s", event.Type)

	// 解析事件数据
	var snapshot FactorLibrarySnapshot
	if err := fss.parseEventData(event.Data, &snapshot); err != nil {
		return fmt.Errorf("failed to parse factor library snapshot: %w", err)
	}

	// 更新本地因子库
	fss.libraryMu.Lock()
	defer fss.libraryMu.Unlock()

	// 清空现有因子库
	fss.factorLibrary = make(map[string]*FactorUpdate)

	// 加载新的因子库
	for factorID, factor := range snapshot.Factors {
		fss.factorLibrary[factorID] = factor
	}

	// 更新同步状态
	fss.syncMu.Lock()
	fss.lastSyncTime = snapshot.UpdateTime
	fss.syncVersion = snapshot.Version
	fss.syncMu.Unlock()

	// 更新统计
	fss.statsMu.Lock()
	fss.stats.SuccessfulSyncs++
	fss.stats.LastSyncTime = time.Now()
	fss.stats.FactorsUpdated += int64(len(snapshot.Factors))
	fss.stats.EventsProcessed++
	fss.statsMu.Unlock()

	log.Printf("因子库同步完成，共更新 %d 个因子", len(snapshot.Factors))

	// 通知策略工作流引擎
	return fss.notifyStrategyEngines(&snapshot)
}

// handleFactorUpdate 处理单个因子更新事件
func (fss *FactorSyncService) handleFactorUpdate(event *events.Event) error {
	log.Printf("收到因子更新事件: %s", event.Type)

	// 解析事件数据
	var factorUpdate FactorUpdate
	if err := fss.parseEventData(event.Data, &factorUpdate); err != nil {
		return fmt.Errorf("failed to parse factor update: %w", err)
	}

	// 更新本地因子库
	fss.libraryMu.Lock()
	fss.factorLibrary[factorUpdate.FactorID] = &factorUpdate
	fss.libraryMu.Unlock()

	// 更新统计
	fss.statsMu.Lock()
	fss.stats.FactorsUpdated++
	fss.stats.EventsProcessed++
	fss.statsMu.Unlock()

	log.Printf("因子 %s 更新完成", factorUpdate.FactorID)

	// 通知相关策略工作流
	return fss.notifyFactorUpdate(&factorUpdate)
}

// handleFactorDeleted 处理因子删除事件
func (fss *FactorSyncService) handleFactorDeleted(event *events.Event) error {
	log.Printf("收到因子删除事件: %s", event.Type)

	factorID, ok := event.Data["factor_id"].(string)
	if !ok {
		return fmt.Errorf("invalid factor_id in delete event")
	}

	// 从本地因子库删除
	fss.libraryMu.Lock()
	delete(fss.factorLibrary, factorID)
	fss.libraryMu.Unlock()

	// 更新统计
	fss.statsMu.Lock()
	fss.stats.EventsProcessed++
	fss.statsMu.Unlock()

	log.Printf("因子 %s 已删除", factorID)

	// 通知策略工作流引擎
	return fss.notifyFactorDeleted(factorID)
}

// runPeriodicSync 运行定期同步
func (fss *FactorSyncService) runPeriodicSync() {
	defer fss.wg.Done()

	ticker := time.NewTicker(fss.config.SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-fss.ctx.Done():
			return
		case <-ticker.C:
			if err := fss.performDeltaSync(); err != nil {
				log.Printf("定期同步失败: %v", err)
				fss.statsMu.Lock()
				fss.stats.FailedSyncs++
				fss.statsMu.Unlock()
			}
		}
	}
}

// runFullSyncScheduler 运行全量同步调度器
func (fss *FactorSyncService) runFullSyncScheduler() {
	defer fss.wg.Done()

	ticker := time.NewTicker(fss.config.FullSyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-fss.ctx.Done():
			return
		case <-ticker.C:
			if err := fss.performFullSync(); err != nil {
				log.Printf("全量同步失败: %v", err)
				fss.statsMu.Lock()
				fss.stats.FailedSyncs++
				fss.statsMu.Unlock()
			}
		}
	}
}

// parseEventData 解析事件数据
func (fss *FactorSyncService) parseEventData(data interface{}, target interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	if err := json.Unmarshal(jsonData, target); err != nil {
		return fmt.Errorf("failed to unmarshal event data: %w", err)
	}

	return nil
}

// performFullSync 执行全量同步
func (fss *FactorSyncService) performFullSync() error {
	fss.syncMu.Lock()
	if fss.syncInProgress {
		fss.syncMu.Unlock()
		return fmt.Errorf("sync already in progress")
	}
	fss.syncInProgress = true
	fss.syncMu.Unlock()

	defer func() {
		fss.syncMu.Lock()
		fss.syncInProgress = false
		fss.syncMu.Unlock()
	}()

	startTime := time.Now()
	log.Println("开始执行因子库全量同步...")

	// 发送全量同步请求事件
	syncRequestEvent := &events.Event{
		Type:   events.EventType("factor_library_sync_request"),
		Source: "factor_sync_service",
		Data: map[string]interface{}{
			"sync_type":    "full",
			"request_time": startTime,
		},
		Timestamp: startTime,
	}

	if err := fss.eventBus.Publish(syncRequestEvent); err != nil {
		return fmt.Errorf("failed to publish sync request: %w", err)
	}

	// 更新统计
	fss.statsMu.Lock()
	fss.stats.TotalSyncs++
	fss.stats.LastSyncDuration = time.Since(startTime)
	fss.statsMu.Unlock()

	log.Printf("因子库全量同步请求已发送，耗时: %v", time.Since(startTime))
	return nil
}

// performDeltaSync 执行增量同步
func (fss *FactorSyncService) performDeltaSync() error {
	if !fss.config.EnableDeltaSync {
		return nil
	}

	fss.syncMu.RLock()
	lastSyncTime := fss.lastSyncTime
	syncVersion := fss.syncVersion
	fss.syncMu.RUnlock()

	startTime := time.Now()
	log.Printf("开始执行因子库增量同步，上次同步时间: %v", lastSyncTime)

	// 发送增量同步请求事件
	syncRequestEvent := &events.Event{
		Type:   events.EventType("factor_library_delta_sync_request"),
		Source: "factor_sync_service",
		Data: map[string]interface{}{
			"sync_type":      "delta",
			"last_sync_time": lastSyncTime,
			"sync_version":   syncVersion,
			"request_time":   startTime,
		},
		Timestamp: startTime,
	}

	if err := fss.eventBus.Publish(syncRequestEvent); err != nil {
		return fmt.Errorf("failed to publish delta sync request: %w", err)
	}

	log.Printf("因子库增量同步请求已发送")
	return nil
}

// notifyStrategyEngines 通知策略工作流引擎
func (fss *FactorSyncService) notifyStrategyEngines(snapshot *FactorLibrarySnapshot) error {
	// 发送因子库更新通知事件
	notifyEvent := &events.Event{
		Type:   events.EventType("strategy_factor_library_updated"),
		Source: "factor_sync_service",
		Data: map[string]interface{}{
			"version":      snapshot.Version,
			"update_time":  snapshot.UpdateTime,
			"active_count": snapshot.ActiveCount,
			"total_count":  snapshot.TotalCount,
			"factors":      snapshot.Factors,
		},
		Timestamp: time.Now(),
	}

	if err := fss.eventBus.Publish(notifyEvent); err != nil {
		return fmt.Errorf("failed to notify strategy engines: %w", err)
	}

	log.Printf("已通知策略工作流引擎因子库更新，版本: %s", snapshot.Version)
	return nil
}

// notifyFactorUpdate 通知因子更新
func (fss *FactorSyncService) notifyFactorUpdate(factor *FactorUpdate) error {
	// 发送单个因子更新通知事件
	notifyEvent := &events.Event{
		Type:   events.EventType("strategy_factor_updated"),
		Source: "factor_sync_service",
		Data: map[string]interface{}{
			"factor_id":     factor.FactorID,
			"factor_name":   factor.FactorName,
			"factor_type":   factor.FactorType,
			"parameters":    factor.Parameters,
			"effectiveness": factor.Effectiveness,
			"update_time":   factor.UpdateTime,
			"version":       factor.Version,
			"is_active":     factor.IsActive,
		},
		Timestamp: time.Now(),
	}

	if err := fss.eventBus.Publish(notifyEvent); err != nil {
		return fmt.Errorf("failed to notify factor update: %w", err)
	}

	log.Printf("已通知策略工作流引擎因子更新: %s", factor.FactorID)
	return nil
}

// notifyFactorDeleted 通知因子删除
func (fss *FactorSyncService) notifyFactorDeleted(factorID string) error {
	// 发送因子删除通知事件
	notifyEvent := &events.Event{
		Type:   events.EventType("strategy_factor_deleted"),
		Source: "factor_sync_service",
		Data: map[string]interface{}{
			"factor_id":   factorID,
			"delete_time": time.Now(),
		},
		Timestamp: time.Now(),
	}

	if err := fss.eventBus.Publish(notifyEvent); err != nil {
		return fmt.Errorf("failed to notify factor deletion: %w", err)
	}

	log.Printf("已通知策略工作流引擎因子删除: %s", factorID)
	return nil
}

// GetFactorLibrary 获取因子库快照
func (fss *FactorSyncService) GetFactorLibrary() map[string]*FactorUpdate {
	fss.libraryMu.RLock()
	defer fss.libraryMu.RUnlock()

	// 返回副本
	library := make(map[string]*FactorUpdate)
	for id, factor := range fss.factorLibrary {
		factorCopy := *factor
		library[id] = &factorCopy
	}

	return library
}

// GetFactorByID 根据ID获取因子
func (fss *FactorSyncService) GetFactorByID(factorID string) (*FactorUpdate, bool) {
	fss.libraryMu.RLock()
	defer fss.libraryMu.RUnlock()

	factor, exists := fss.factorLibrary[factorID]
	if !exists {
		return nil, false
	}

	// 返回副本
	factorCopy := *factor
	return &factorCopy, true
}

// GetStats 获取同步统计信息
func (fss *FactorSyncService) GetStats() *FactorSyncStats {
	fss.statsMu.RLock()
	defer fss.statsMu.RUnlock()

	// 返回副本
	stats := *fss.stats
	return &stats
}

// IsRunning 检查服务是否正在运行
func (fss *FactorSyncService) IsRunning() bool {
	fss.runningMu.RLock()
	defer fss.runningMu.RUnlock()
	return fss.isRunning
}

// GetDefaultFactorSyncConfig 获取默认因子同步配置
func GetDefaultFactorSyncConfig() *FactorSyncConfig {
	return &FactorSyncConfig{
		SyncInterval:     30 * time.Second,
		EventTimeout:     10 * time.Second,
		MaxRetries:       3,
		RetryDelay:       5 * time.Second,
		CacheSize:        1000,
		CacheTTL:         1 * time.Hour,
		FullSyncInterval: 1 * time.Hour,
		EnableDeltaSync:  true,
	}
}
