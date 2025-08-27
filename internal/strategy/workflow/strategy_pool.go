package workflow

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/events"
)

// StrategyPoolInterface 策略池接口
type StrategyPoolInterface interface {
	// 获取已启用策略
	GetEnabledStrategies() []*EnabledStrategy
	
	// 策略状态变更通知
	OnStrategyEnabled(strategyID string)
	OnStrategyDisabled(strategyID string)
	
	// 策略性能更新
	UpdateStrategyPerformance(strategyID string, metrics *PerformanceMetrics)
	
	// 检查策略是否启用
	IsStrategyEnabled(strategyID string) bool
	
	// 获取策略信息
	GetStrategyInfo(strategyID string) (*EnabledStrategy, error)
}

// TradingStrategyPool 交易策略池
type TradingStrategyPool struct {
	// 多策略管理器引用
	multiStrategyManager *MultiStrategyManager
	
	// 本地缓存
	enabledStrategies map[string]*EnabledStrategy
	strategiesMu      sync.RWMutex
	
	// 事件系统
	eventBus *events.EventBus
	
	// 更新配置
	updateInterval time.Duration
	lastUpdate     time.Time
	
	// 运行状态
	ctx       context.Context
	cancel    context.CancelFunc
	isRunning bool
	runningMu sync.RWMutex
	wg        sync.WaitGroup
}

// NewTradingStrategyPool 创建交易策略池
func NewTradingStrategyPool(multiStrategyManager *MultiStrategyManager, eventBus *events.EventBus) *TradingStrategyPool {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &TradingStrategyPool{
		multiStrategyManager: multiStrategyManager,
		enabledStrategies:    make(map[string]*EnabledStrategy),
		eventBus:             eventBus,
		updateInterval:       30 * time.Second,
		ctx:                  ctx,
		cancel:               cancel,
	}
}

// Start 启动策略池
func (tsp *TradingStrategyPool) Start() error {
	tsp.runningMu.Lock()
	defer tsp.runningMu.Unlock()
	
	if tsp.isRunning {
		return fmt.Errorf("trading strategy pool is already running")
	}
	
	log.Println("启动交易策略池...")
	
	// 初始同步策略
	if err := tsp.syncStrategies(); err != nil {
		return fmt.Errorf("failed to sync strategies: %w", err)
	}
	
	// 启动定期更新
	tsp.wg.Add(1)
	go tsp.runUpdateLoop()
	
	// 订阅策略事件
	tsp.subscribeToStrategyEvents()
	
	tsp.isRunning = true
	
	log.Printf("交易策略池启动完成，当前启用策略数: %d", len(tsp.enabledStrategies))
	return nil
}

// Stop 停止策略池
func (tsp *TradingStrategyPool) Stop() error {
	tsp.runningMu.Lock()
	defer tsp.runningMu.Unlock()
	
	if !tsp.isRunning {
		return nil
	}
	
	log.Println("停止交易策略池...")
	
	// 取消上下文
	tsp.cancel()
	
	// 等待更新循环结束
	tsp.wg.Wait()
	
	tsp.isRunning = false
	
	log.Println("交易策略池已停止")
	return nil
}

// GetEnabledStrategies 获取已启用策略
func (tsp *TradingStrategyPool) GetEnabledStrategies() []*EnabledStrategy {
	tsp.strategiesMu.RLock()
	defer tsp.strategiesMu.RUnlock()
	
	strategies := make([]*EnabledStrategy, 0, len(tsp.enabledStrategies))
	for _, strategy := range tsp.enabledStrategies {
		if strategy.IsActive && strategy.TradingEnabled {
			strategies = append(strategies, strategy)
		}
	}
	
	return strategies
}

// OnStrategyEnabled 策略启用通知
func (tsp *TradingStrategyPool) OnStrategyEnabled(strategyID string) {
	log.Printf("策略 %s 已启用，更新交易策略池", strategyID)
	
	// 立即同步策略
	if err := tsp.syncStrategies(); err != nil {
		log.Printf("Warning: failed to sync strategies after enable: %v", err)
	}
	
	// 发送事件
	tsp.emitEvent("trading_strategy_enabled", map[string]interface{}{
		"strategy_id": strategyID,
		"timestamp":   time.Now(),
	})
}

// OnStrategyDisabled 策略禁用通知
func (tsp *TradingStrategyPool) OnStrategyDisabled(strategyID string) {
	log.Printf("策略 %s 已禁用，从交易策略池移除", strategyID)
	
	tsp.strategiesMu.Lock()
	delete(tsp.enabledStrategies, strategyID)
	tsp.strategiesMu.Unlock()
	
	// 发送事件
	tsp.emitEvent("trading_strategy_disabled", map[string]interface{}{
		"strategy_id": strategyID,
		"timestamp":   time.Now(),
	})
}

// UpdateStrategyPerformance 更新策略性能
func (tsp *TradingStrategyPool) UpdateStrategyPerformance(strategyID string, metrics *PerformanceMetrics) {
	tsp.strategiesMu.Lock()
	defer tsp.strategiesMu.Unlock()
	
	if strategy, exists := tsp.enabledStrategies[strategyID]; exists {
		strategy.Performance = metrics
		strategy.LastUpdated = time.Now()
		
		log.Printf("更新策略 %s 性能指标: 夏普比率=%.3f", strategyID, metrics.SharpeRatio)
	}
}

// IsStrategyEnabled 检查策略是否启用
func (tsp *TradingStrategyPool) IsStrategyEnabled(strategyID string) bool {
	tsp.strategiesMu.RLock()
	defer tsp.strategiesMu.RUnlock()
	
	strategy, exists := tsp.enabledStrategies[strategyID]
	return exists && strategy.IsActive && strategy.TradingEnabled
}

// GetStrategyInfo 获取策略信息
func (tsp *TradingStrategyPool) GetStrategyInfo(strategyID string) (*EnabledStrategy, error) {
	tsp.strategiesMu.RLock()
	defer tsp.strategiesMu.RUnlock()
	
	strategy, exists := tsp.enabledStrategies[strategyID]
	if !exists {
		return nil, fmt.Errorf("strategy %s not found in enabled strategies", strategyID)
	}
	
	// 返回副本
	strategyCopy := *strategy
	return &strategyCopy, nil
}

// syncStrategies 同步策略
func (tsp *TradingStrategyPool) syncStrategies() error {
	if tsp.multiStrategyManager == nil {
		return fmt.Errorf("multi-strategy manager is not available")
	}
	
	// 从多策略管理器获取启用的策略
	tsp.multiStrategyManager.strategiesMu.RLock()
	enabledStrategies := make(map[string]*EnabledStrategy)
	for id, strategy := range tsp.multiStrategyManager.enabledStrategies {
		// 创建副本
		strategyCopy := *strategy
		enabledStrategies[id] = &strategyCopy
	}
	tsp.multiStrategyManager.strategiesMu.RUnlock()
	
	// 更新本地缓存
	tsp.strategiesMu.Lock()
	tsp.enabledStrategies = enabledStrategies
	tsp.lastUpdate = time.Now()
	tsp.strategiesMu.Unlock()
	
	log.Printf("策略同步完成，当前启用策略数: %d", len(enabledStrategies))
	return nil
}

// runUpdateLoop 运行更新循环
func (tsp *TradingStrategyPool) runUpdateLoop() {
	defer tsp.wg.Done()
	
	ticker := time.NewTicker(tsp.updateInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-tsp.ctx.Done():
			return
		case <-ticker.C:
			if err := tsp.syncStrategies(); err != nil {
				log.Printf("Warning: failed to sync strategies: %v", err)
			}
		}
	}
}

// subscribeToStrategyEvents 订阅策略事件
func (tsp *TradingStrategyPool) subscribeToStrategyEvents() {
	if tsp.eventBus == nil {
		return
	}
	
	// 订阅策略启用事件
	tsp.eventBus.Subscribe("strategy_enabled", func(event *events.Event) error {
		if strategyID, ok := event.Data["strategy_id"].(string); ok {
			tsp.OnStrategyEnabled(strategyID)
		}
		return nil
	})
	
	// 订阅策略禁用事件
	tsp.eventBus.Subscribe("strategy_disabled", func(event *events.Event) error {
		if strategyID, ok := event.Data["strategy_id"].(string); ok {
			tsp.OnStrategyDisabled(strategyID)
		}
		return nil
	})
}

// emitEvent 发送事件
func (tsp *TradingStrategyPool) emitEvent(eventType string, data map[string]interface{}) {
	if tsp.eventBus == nil {
		return
	}
	
	event := &events.Event{
		Type:      events.EventType(eventType),
		Source:    "trading_strategy_pool",
		Data:      data,
		Timestamp: time.Now(),
	}
	
	if err := tsp.eventBus.Publish(event); err != nil {
		log.Printf("Warning: failed to emit event %s: %v", eventType, err)
	}
}

// GetStats 获取策略池统计信息
func (tsp *TradingStrategyPool) GetStats() map[string]interface{} {
	tsp.strategiesMu.RLock()
	defer tsp.strategiesMu.RUnlock()
	
	totalStrategies := len(tsp.enabledStrategies)
	activeStrategies := 0
	tradingEnabledStrategies := 0
	
	for _, strategy := range tsp.enabledStrategies {
		if strategy.IsActive {
			activeStrategies++
		}
		if strategy.TradingEnabled {
			tradingEnabledStrategies++
		}
	}
	
	return map[string]interface{}{
		"total_strategies":           totalStrategies,
		"active_strategies":          activeStrategies,
		"trading_enabled_strategies": tradingEnabledStrategies,
		"last_update":                tsp.lastUpdate,
		"update_interval":            tsp.updateInterval.String(),
	}
}
