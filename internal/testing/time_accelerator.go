package testing

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// EnhancedTimeAccelerator 增强版时间加速器
type EnhancedTimeAccelerator struct {
	// 基础配置
	startTime          time.Time
	currentTime        time.Time
	accelerationFactor int
	realStartTime      time.Time

	// 高级配置
	config *TimeAcceleratorConfig

	// 时间事件管理
	events      []*TimeEvent
	eventsMutex sync.RWMutex

	// 时间监听器
	listeners      []TimeListener
	listenersMutex sync.RWMutex

	// 控制
	ctx          context.Context
	cancel       context.CancelFunc
	running      bool
	runningMutex sync.RWMutex

	// 统计
	stats      *TimeAcceleratorStats
	statsMutex sync.RWMutex
}

// TimeAcceleratorConfig 时间加速器配置
type TimeAcceleratorConfig struct {
	// 基础配置
	AccelerationFactor int           `json:"acceleration_factor"` // 加速倍数
	TickInterval       time.Duration `json:"tick_interval"`       // 时间推进间隔

	// 高级配置
	MaxAcceleration int  `json:"max_acceleration"` // 最大加速倍数
	MinAcceleration int  `json:"min_acceleration"` // 最小加速倍数
	AutoAdjust      bool `json:"auto_adjust"`      // 自动调整加速倍数

	// 时间范围配置
	SimulationDuration time.Duration `json:"simulation_duration"` // 模拟总时长
	MaxRealDuration    time.Duration `json:"max_real_duration"`   // 最大实际运行时长

	// 事件配置
	EnableEvents    bool `json:"enable_events"`     // 启用时间事件
	EventBufferSize int  `json:"event_buffer_size"` // 事件缓冲区大小

	// 监控配置
	EnableStats   bool          `json:"enable_stats"`   // 启用统计
	StatsInterval time.Duration `json:"stats_interval"` // 统计更新间隔
}

// TimeEvent 时间事件
type TimeEvent struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	TriggerTime  time.Time             `json:"trigger_time"`
	Handler      func(time.Time) error `json:"-"`
	Recurring    bool                  `json:"recurring"`
	Interval     time.Duration         `json:"interval"`
	Enabled      bool                  `json:"enabled"`
	LastTrigger  time.Time             `json:"last_trigger"`
	TriggerCount int64                 `json:"trigger_count"`
}

// TimeListener 时间监听器接口
type TimeListener interface {
	OnTimeAdvance(currentTime time.Time, elapsed time.Duration)
	OnAccelerationChange(oldFactor, newFactor int)
	OnEventTrigger(event *TimeEvent)
}

// TimeAcceleratorStats 时间加速器统计
type TimeAcceleratorStats struct {
	StartTime           time.Time     `json:"start_time"`
	CurrentTime         time.Time     `json:"current_time"`
	RealElapsed         time.Duration `json:"real_elapsed"`
	SimulatedElapsed    time.Duration `json:"simulated_elapsed"`
	AccelerationFactor  int           `json:"acceleration_factor"`
	EffectiveRatio      float64       `json:"effective_ratio"`
	EventsTriggered     int64         `json:"events_triggered"`
	ListenersCount      int           `json:"listeners_count"`
	TicksProcessed      int64         `json:"ticks_processed"`
	AverageTickDuration time.Duration `json:"average_tick_duration"`
	LastUpdate          time.Time     `json:"last_update"`
}

// DefaultTimeAcceleratorConfig 默认配置
func DefaultTimeAcceleratorConfig() *TimeAcceleratorConfig {
	return &TimeAcceleratorConfig{
		AccelerationFactor: 100, // 100倍加速
		TickInterval:       10 * time.Millisecond,
		MaxAcceleration:    10000, // 最大10000倍加速
		MinAcceleration:    1,     // 最小1倍（正常速度）
		AutoAdjust:         false,
		SimulationDuration: 24 * time.Hour,   // 模拟24小时
		MaxRealDuration:    10 * time.Minute, // 实际运行最多10分钟
		EnableEvents:       true,
		EventBufferSize:    1000,
		EnableStats:        true,
		StatsInterval:      time.Second,
	}
}

// NewEnhancedTimeAccelerator 创建增强版时间加速器
func NewEnhancedTimeAccelerator(startTime time.Time, config *TimeAcceleratorConfig) *EnhancedTimeAccelerator {
	if config == nil {
		config = DefaultTimeAcceleratorConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	accelerator := &EnhancedTimeAccelerator{
		startTime:          startTime,
		currentTime:        startTime,
		accelerationFactor: config.AccelerationFactor,
		realStartTime:      time.Now(),
		config:             config,
		events:             make([]*TimeEvent, 0, config.EventBufferSize),
		listeners:          make([]TimeListener, 0),
		ctx:                ctx,
		cancel:             cancel,
		running:            false,
		stats: &TimeAcceleratorStats{
			StartTime:          startTime,
			CurrentTime:        startTime,
			AccelerationFactor: config.AccelerationFactor,
			LastUpdate:         time.Now(),
		},
	}

	return accelerator
}

// Start 启动时间加速器
func (eta *EnhancedTimeAccelerator) Start() error {
	eta.runningMutex.Lock()
	defer eta.runningMutex.Unlock()

	if eta.running {
		return fmt.Errorf("时间加速器已经在运行")
	}

	eta.running = true
	eta.realStartTime = time.Now()

	log.Printf("🚀 启动时间加速器，加速倍数: %dx", eta.accelerationFactor)

	// 启动主循环
	go eta.runLoop()

	// 启动统计更新
	if eta.config.EnableStats {
		go eta.statsLoop()
	}

	return nil
}

// Stop 停止时间加速器
func (eta *EnhancedTimeAccelerator) Stop() {
	eta.runningMutex.Lock()
	defer eta.runningMutex.Unlock()

	if !eta.running {
		return
	}

	eta.running = false
	eta.cancel()

	log.Printf("🛑 停止时间加速器，模拟时长: %v，实际时长: %v",
		eta.GetSimulatedElapsed(), eta.GetRealElapsed())
}

// runLoop 主运行循环
func (eta *EnhancedTimeAccelerator) runLoop() {
	ticker := time.NewTicker(eta.config.TickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			eta.tick()
		case <-eta.ctx.Done():
			return
		}
	}
}

// tick 时间推进
func (eta *EnhancedTimeAccelerator) tick() {
	tickStart := time.Now()

	// 计算时间推进量
	realElapsed := time.Since(eta.realStartTime)
	targetSimulatedElapsed := time.Duration(int64(realElapsed) * int64(eta.accelerationFactor))
	currentSimulatedElapsed := eta.currentTime.Sub(eta.startTime)

	// 推进时间
	if targetSimulatedElapsed > currentSimulatedElapsed {
		advance := targetSimulatedElapsed - currentSimulatedElapsed
		eta.currentTime = eta.currentTime.Add(advance)

		// 通知监听器
		eta.notifyListeners(advance)

		// 处理时间事件
		if eta.config.EnableEvents {
			eta.processEvents()
		}

		// 更新统计
		eta.updateStats(time.Since(tickStart))

		// 检查是否达到停止条件
		eta.checkStopConditions()
	}
}

// processEvents 处理时间事件
func (eta *EnhancedTimeAccelerator) processEvents() {
	eta.eventsMutex.RLock()
	defer eta.eventsMutex.RUnlock()

	for _, event := range eta.events {
		if !event.Enabled {
			continue
		}

		shouldTrigger := false

		if event.Recurring {
			// 周期性事件
			if event.LastTrigger.IsZero() {
				shouldTrigger = !eta.currentTime.Before(event.TriggerTime)
			} else {
				shouldTrigger = eta.currentTime.Sub(event.LastTrigger) >= event.Interval
			}
		} else {
			// 一次性事件
			shouldTrigger = !eta.currentTime.Before(event.TriggerTime) && event.TriggerCount == 0
		}

		if shouldTrigger {
			eta.triggerEvent(event)
		}
	}
}

// triggerEvent 触发事件
func (eta *EnhancedTimeAccelerator) triggerEvent(event *TimeEvent) {
	if event.Handler != nil {
		go func() {
			err := event.Handler(eta.currentTime)
			if err != nil {
				log.Printf("时间事件处理失败 [%s]: %v", event.Name, err)
			}
		}()
	}

	event.LastTrigger = eta.currentTime
	event.TriggerCount++

	// 通知监听器
	eta.listenersMutex.RLock()
	for _, listener := range eta.listeners {
		go listener.OnEventTrigger(event)
	}
	eta.listenersMutex.RUnlock()

	// 更新统计
	eta.statsMutex.Lock()
	eta.stats.EventsTriggered++
	eta.statsMutex.Unlock()
}

// notifyListeners 通知监听器
func (eta *EnhancedTimeAccelerator) notifyListeners(elapsed time.Duration) {
	eta.listenersMutex.RLock()
	defer eta.listenersMutex.RUnlock()

	for _, listener := range eta.listeners {
		go listener.OnTimeAdvance(eta.currentTime, elapsed)
	}
}

// checkStopConditions 检查停止条件
func (eta *EnhancedTimeAccelerator) checkStopConditions() {
	// 检查模拟时长
	if eta.config.SimulationDuration > 0 {
		if eta.GetSimulatedElapsed() >= eta.config.SimulationDuration {
			log.Printf("达到模拟时长限制，停止时间加速器")
			eta.Stop()
			return
		}
	}

	// 检查实际运行时长
	if eta.config.MaxRealDuration > 0 {
		if eta.GetRealElapsed() >= eta.config.MaxRealDuration {
			log.Printf("达到实际运行时长限制，停止时间加速器")
			eta.Stop()
			return
		}
	}
}

// statsLoop 统计更新循环
func (eta *EnhancedTimeAccelerator) statsLoop() {
	ticker := time.NewTicker(eta.config.StatsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			eta.updateStatsSnapshot()
		case <-eta.ctx.Done():
			return
		}
	}
}

// updateStats 更新统计信息
func (eta *EnhancedTimeAccelerator) updateStats(tickDuration time.Duration) {
	eta.statsMutex.Lock()
	defer eta.statsMutex.Unlock()

	eta.stats.TicksProcessed++

	// 计算平均tick处理时间
	if eta.stats.TicksProcessed == 1 {
		eta.stats.AverageTickDuration = tickDuration
	} else {
		// 使用指数移动平均
		alpha := 0.1
		eta.stats.AverageTickDuration = time.Duration(
			float64(eta.stats.AverageTickDuration)*(1-alpha) + float64(tickDuration)*alpha,
		)
	}
}

// updateStatsSnapshot 更新统计快照
func (eta *EnhancedTimeAccelerator) updateStatsSnapshot() {
	eta.statsMutex.Lock()
	defer eta.statsMutex.Unlock()

	eta.stats.CurrentTime = eta.currentTime
	eta.stats.RealElapsed = eta.GetRealElapsed()
	eta.stats.SimulatedElapsed = eta.GetSimulatedElapsed()
	eta.stats.AccelerationFactor = eta.accelerationFactor
	eta.stats.EffectiveRatio = eta.GetEffectiveRatio()
	eta.stats.ListenersCount = len(eta.listeners)
	eta.stats.LastUpdate = time.Now()
}

// GetCurrentTime 获取当前模拟时间
func (eta *EnhancedTimeAccelerator) GetCurrentTime() time.Time {
	return eta.currentTime
}

// GetRealElapsed 获取实际经过时间
func (eta *EnhancedTimeAccelerator) GetRealElapsed() time.Duration {
	return time.Since(eta.realStartTime)
}

// GetSimulatedElapsed 获取模拟经过时间
func (eta *EnhancedTimeAccelerator) GetSimulatedElapsed() time.Duration {
	return eta.currentTime.Sub(eta.startTime)
}

// GetEffectiveRatio 获取有效加速比率
func (eta *EnhancedTimeAccelerator) GetEffectiveRatio() float64 {
	realElapsed := eta.GetRealElapsed()
	if realElapsed == 0 {
		return 0
	}
	simulatedElapsed := eta.GetSimulatedElapsed()
	return float64(simulatedElapsed) / float64(realElapsed)
}

// GetAccelerationFactor 获取加速倍数
func (eta *EnhancedTimeAccelerator) GetAccelerationFactor() int {
	return eta.accelerationFactor
}

// SetAccelerationFactor 设置加速倍数
func (eta *EnhancedTimeAccelerator) SetAccelerationFactor(factor int) error {
	if factor < eta.config.MinAcceleration || factor > eta.config.MaxAcceleration {
		return fmt.Errorf("加速倍数超出范围 [%d, %d]", eta.config.MinAcceleration, eta.config.MaxAcceleration)
	}

	oldFactor := eta.accelerationFactor
	eta.accelerationFactor = factor
	eta.realStartTime = time.Now() // 重置实际开始时间

	log.Printf("调整加速倍数: %dx -> %dx", oldFactor, factor)

	// 通知监听器
	eta.listenersMutex.RLock()
	for _, listener := range eta.listeners {
		go listener.OnAccelerationChange(oldFactor, factor)
	}
	eta.listenersMutex.RUnlock()

	return nil
}

// AddEvent 添加时间事件
func (eta *EnhancedTimeAccelerator) AddEvent(event *TimeEvent) error {
	if event == nil {
		return fmt.Errorf("事件不能为空")
	}

	eta.eventsMutex.Lock()
	defer eta.eventsMutex.Unlock()

	// 检查缓冲区大小
	if len(eta.events) >= eta.config.EventBufferSize {
		return fmt.Errorf("事件缓冲区已满")
	}

	eta.events = append(eta.events, event)
	log.Printf("添加时间事件: %s，触发时间: %v", event.Name, event.TriggerTime)

	return nil
}

// RemoveEvent 移除时间事件
func (eta *EnhancedTimeAccelerator) RemoveEvent(eventID string) bool {
	eta.eventsMutex.Lock()
	defer eta.eventsMutex.Unlock()

	for i, event := range eta.events {
		if event.ID == eventID {
			eta.events = append(eta.events[:i], eta.events[i+1:]...)
			log.Printf("移除时间事件: %s", event.Name)
			return true
		}
	}

	return false
}

// AddListener 添加时间监听器
func (eta *EnhancedTimeAccelerator) AddListener(listener TimeListener) {
	eta.listenersMutex.Lock()
	defer eta.listenersMutex.Unlock()

	eta.listeners = append(eta.listeners, listener)
}

// RemoveListener 移除时间监听器
func (eta *EnhancedTimeAccelerator) RemoveListener(listener TimeListener) bool {
	eta.listenersMutex.Lock()
	defer eta.listenersMutex.Unlock()

	for i, l := range eta.listeners {
		if l == listener {
			eta.listeners = append(eta.listeners[:i], eta.listeners[i+1:]...)
			return true
		}
	}

	return false
}

// GetStats 获取统计信息
func (eta *EnhancedTimeAccelerator) GetStats() *TimeAcceleratorStats {
	eta.statsMutex.RLock()
	defer eta.statsMutex.RUnlock()

	// 返回副本
	stats := *eta.stats
	return &stats
}

// IsRunning 检查是否正在运行
func (eta *EnhancedTimeAccelerator) IsRunning() bool {
	eta.runningMutex.RLock()
	defer eta.runningMutex.RUnlock()

	return eta.running
}

// GetConfig 获取配置
func (eta *EnhancedTimeAccelerator) GetConfig() *TimeAcceleratorConfig {
	// 返回副本
	config := *eta.config
	return &config
}
