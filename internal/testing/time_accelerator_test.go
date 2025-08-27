package testing

import (
	"testing"
	"time"
)

// TestTimeListener 测试时间监听器
type TestTimeListener struct {
	timeAdvances        []time.Time
	accelerationChanges []AccelerationChange
	eventTriggers       []string
}

type AccelerationChange struct {
	OldFactor int
	NewFactor int
}

func (tl *TestTimeListener) OnTimeAdvance(currentTime time.Time, elapsed time.Duration) {
	tl.timeAdvances = append(tl.timeAdvances, currentTime)
}

func (tl *TestTimeListener) OnAccelerationChange(oldFactor, newFactor int) {
	tl.accelerationChanges = append(tl.accelerationChanges, AccelerationChange{
		OldFactor: oldFactor,
		NewFactor: newFactor,
	})
}

func (tl *TestTimeListener) OnEventTrigger(event *TimeEvent) {
	tl.eventTriggers = append(tl.eventTriggers, event.Name)
}

func TestEnhancedTimeAccelerator_Creation(t *testing.T) {
	startTime := time.Now()
	config := DefaultTimeAcceleratorConfig()

	accelerator := NewEnhancedTimeAccelerator(startTime, config)

	if accelerator == nil {
		t.Fatal("时间加速器创建失败")
	}

	if accelerator.GetCurrentTime() != startTime {
		t.Errorf("期望开始时间 %v，实际得到 %v", startTime, accelerator.GetCurrentTime())
	}

	if accelerator.GetAccelerationFactor() != config.AccelerationFactor {
		t.Errorf("期望加速倍数 %d，实际得到 %d", config.AccelerationFactor, accelerator.GetAccelerationFactor())
	}

	if accelerator.IsRunning() {
		t.Error("新创建的加速器不应该在运行")
	}
}

func TestEnhancedTimeAccelerator_DefaultConfig(t *testing.T) {
	config := DefaultTimeAcceleratorConfig()

	if config == nil {
		t.Fatal("默认配置为空")
	}

	if config.AccelerationFactor <= 0 {
		t.Error("加速倍数应大于0")
	}

	if config.TickInterval <= 0 {
		t.Error("时间推进间隔应大于0")
	}

	if config.MaxAcceleration <= config.MinAcceleration {
		t.Error("最大加速倍数应大于最小加速倍数")
	}
}

func TestEnhancedTimeAccelerator_StartStop(t *testing.T) {
	startTime := time.Now()
	config := DefaultTimeAcceleratorConfig()
	config.SimulationDuration = 100 * time.Millisecond // 短时间测试

	accelerator := NewEnhancedTimeAccelerator(startTime, config)

	// 测试启动
	err := accelerator.Start()
	if err != nil {
		t.Fatalf("启动失败: %v", err)
	}

	if !accelerator.IsRunning() {
		t.Error("启动后应该在运行状态")
	}

	// 测试重复启动
	err = accelerator.Start()
	if err == nil {
		t.Error("重复启动应该返回错误")
	}

	// 等待一段时间
	time.Sleep(50 * time.Millisecond)

	// 检查时间是否推进
	currentTime := accelerator.GetCurrentTime()
	if !currentTime.After(startTime) {
		t.Error("时间应该已经推进")
	}

	// 测试停止
	accelerator.Stop()

	if accelerator.IsRunning() {
		t.Error("停止后不应该在运行状态")
	}
}

func TestEnhancedTimeAccelerator_AccelerationFactor(t *testing.T) {
	startTime := time.Now()
	config := DefaultTimeAcceleratorConfig()

	accelerator := NewEnhancedTimeAccelerator(startTime, config)

	// 测试设置有效的加速倍数
	err := accelerator.SetAccelerationFactor(200)
	if err != nil {
		t.Errorf("设置有效加速倍数失败: %v", err)
	}

	if accelerator.GetAccelerationFactor() != 200 {
		t.Errorf("期望加速倍数 200，实际得到 %d", accelerator.GetAccelerationFactor())
	}

	// 测试设置无效的加速倍数
	err = accelerator.SetAccelerationFactor(config.MaxAcceleration + 1)
	if err == nil {
		t.Error("设置超出范围的加速倍数应该返回错误")
	}

	err = accelerator.SetAccelerationFactor(config.MinAcceleration - 1)
	if err == nil {
		t.Error("设置低于范围的加速倍数应该返回错误")
	}
}

func TestEnhancedTimeAccelerator_Events(t *testing.T) {
	startTime := time.Now()
	config := DefaultTimeAcceleratorConfig()
	config.AccelerationFactor = 1000 // 高倍速以快速触发事件

	accelerator := NewEnhancedTimeAccelerator(startTime, config)

	// 创建测试事件
	eventTriggered := false
	event := &TimeEvent{
		ID:          "test-event",
		Name:        "测试事件",
		TriggerTime: startTime.Add(10 * time.Millisecond),
		Handler: func(t time.Time) error {
			eventTriggered = true
			return nil
		},
		Enabled: true,
	}

	// 添加事件
	err := accelerator.AddEvent(event)
	if err != nil {
		t.Fatalf("添加事件失败: %v", err)
	}

	// 启动加速器
	err = accelerator.Start()
	if err != nil {
		t.Fatalf("启动失败: %v", err)
	}
	defer accelerator.Stop()

	// 等待事件触发
	time.Sleep(100 * time.Millisecond)

	if !eventTriggered {
		t.Error("事件应该已经被触发")
	}

	if event.TriggerCount == 0 {
		t.Error("事件触发计数应该大于0")
	}

	// 测试移除事件
	removed := accelerator.RemoveEvent("test-event")
	if !removed {
		t.Error("应该能够移除事件")
	}

	removed = accelerator.RemoveEvent("non-existent")
	if removed {
		t.Error("移除不存在的事件应该返回false")
	}
}

func TestEnhancedTimeAccelerator_Listeners(t *testing.T) {
	startTime := time.Now()
	config := DefaultTimeAcceleratorConfig()
	config.AccelerationFactor = 100

	accelerator := NewEnhancedTimeAccelerator(startTime, config)

	// 创建测试监听器
	listener := &TestTimeListener{}

	// 添加监听器
	accelerator.AddListener(listener)

	// 启动加速器
	err := accelerator.Start()
	if err != nil {
		t.Fatalf("启动失败: %v", err)
	}

	// 等待一段时间
	time.Sleep(50 * time.Millisecond)

	// 测试加速倍数变化
	err = accelerator.SetAccelerationFactor(200)
	if err != nil {
		t.Fatalf("设置加速倍数失败: %v", err)
	}

	accelerator.Stop()

	// 检查监听器是否收到通知
	if len(listener.timeAdvances) == 0 {
		t.Error("监听器应该收到时间推进通知")
	}

	if len(listener.accelerationChanges) == 0 {
		t.Error("监听器应该收到加速倍数变化通知")
	} else {
		change := listener.accelerationChanges[0]
		if change.OldFactor != 100 || change.NewFactor != 200 {
			t.Errorf("期望加速倍数变化 100->200，实际得到 %d->%d",
				change.OldFactor, change.NewFactor)
		}
	}

	// 测试移除监听器
	removed := accelerator.RemoveListener(listener)
	if !removed {
		t.Error("应该能够移除监听器")
	}
}

func TestEnhancedTimeAccelerator_Stats(t *testing.T) {
	startTime := time.Now()
	config := DefaultTimeAcceleratorConfig()
	config.AccelerationFactor = 100

	accelerator := NewEnhancedTimeAccelerator(startTime, config)

	// 获取初始统计
	stats := accelerator.GetStats()
	if stats == nil {
		t.Fatal("统计信息不应该为空")
	}

	if stats.AccelerationFactor != 100 {
		t.Errorf("期望加速倍数 100，实际得到 %d", stats.AccelerationFactor)
	}

	// 启动加速器
	err := accelerator.Start()
	if err != nil {
		t.Fatalf("启动失败: %v", err)
	}
	defer accelerator.Stop()

	// 等待一段时间
	time.Sleep(100 * time.Millisecond)

	// 获取更新后的统计
	stats = accelerator.GetStats()

	if stats.TicksProcessed == 0 {
		t.Error("应该已经处理了一些时间推进")
	}

	realElapsed := accelerator.GetRealElapsed()
	if realElapsed == 0 {
		t.Error("实际经过时间应该大于0")
	}

	simulatedElapsed := accelerator.GetSimulatedElapsed()
	if simulatedElapsed == 0 {
		t.Error("模拟经过时间应该大于0")
	}

	effectiveRatio := accelerator.GetEffectiveRatio()
	if effectiveRatio <= 0 {
		t.Error("有效加速比率应该大于0")
	}
}

// 基准测试
func BenchmarkEnhancedTimeAccelerator_Tick(b *testing.B) {
	startTime := time.Now()
	config := DefaultTimeAcceleratorConfig()
	config.EnableStats = false  // 禁用统计以提高性能
	config.EnableEvents = false // 禁用事件以提高性能

	accelerator := NewEnhancedTimeAccelerator(startTime, config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		accelerator.tick()
	}
}

func BenchmarkEnhancedTimeAccelerator_GetCurrentTime(b *testing.B) {
	startTime := time.Now()
	config := DefaultTimeAcceleratorConfig()

	accelerator := NewEnhancedTimeAccelerator(startTime, config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = accelerator.GetCurrentTime()
	}
}
