package testing

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestEnhancedStressTestFramework_Creation(t *testing.T) {
	config := DefaultEnhancedStressTestConfig()
	config.UseRealData = false // 禁用真实数据以避免网络依赖
	
	framework, err := NewEnhancedStressTestFramework(config)
	if err != nil {
		t.Fatalf("创建增强版压力测试框架失败: %v", err)
	}
	
	if framework == nil {
		t.Fatal("框架不应该为空")
	}
	
	if framework.IsRunning() {
		t.Error("新创建的框架不应该在运行")
	}
	
	if framework.GetConfig().TestName != config.TestName {
		t.Errorf("期望测试名称 %s，实际得到 %s", config.TestName, framework.GetConfig().TestName)
	}
}

func TestEnhancedStressTestFramework_DefaultConfig(t *testing.T) {
	config := DefaultEnhancedStressTestConfig()
	
	if config == nil {
		t.Fatal("默认配置不应该为空")
	}
	
	if config.Duration <= 0 {
		t.Error("测试持续时间应该大于0")
	}
	
	if config.MaxConcurrency <= 0 {
		t.Error("最大并发数应该大于0")
	}
	
	if config.MaxRPS <= 0 {
		t.Error("最大RPS应该大于0")
	}
	
	if config.MonitoringInterval <= 0 {
		t.Error("监控间隔应该大于0")
	}
}

func TestEnhancedStressTestFramework_Scenarios(t *testing.T) {
	config := DefaultEnhancedStressTestConfig()
	config.UseRealData = false
	
	framework, err := NewEnhancedStressTestFramework(config)
	if err != nil {
		t.Fatalf("创建框架失败: %v", err)
	}
	
	// 测试添加场景
	scenario1 := &TestScenario{
		ID:          "test-scenario-1",
		Name:        "测试场景1",
		Description: "第一个测试场景",
		Weight:      1.0,
		Handler: func(ctx context.Context) error {
			time.Sleep(time.Millisecond)
			return nil
		},
		Enabled: true,
	}
	
	err = framework.AddScenario(scenario1)
	if err != nil {
		t.Errorf("添加场景失败: %v", err)
	}
	
	// 测试添加重复ID的场景
	scenario2 := &TestScenario{
		ID:      "test-scenario-1", // 重复ID
		Name:    "测试场景2",
		Enabled: true,
	}
	
	err = framework.AddScenario(scenario2)
	if err == nil {
		t.Error("添加重复ID的场景应该返回错误")
	}
	
	// 测试添加空场景
	err = framework.AddScenario(nil)
	if err == nil {
		t.Error("添加空场景应该返回错误")
	}
	
	// 测试移除场景
	removed := framework.RemoveScenario("test-scenario-1")
	if !removed {
		t.Error("应该能够移除存在的场景")
	}
	
	removed = framework.RemoveScenario("non-existent")
	if removed {
		t.Error("移除不存在的场景应该返回false")
	}
}

func TestEnhancedStressTestFramework_ShortRun(t *testing.T) {
	config := DefaultEnhancedStressTestConfig()
	config.UseRealData = false
	config.Duration = 100 * time.Millisecond // 短时间测试
	config.WarmupDuration = 0
	config.CooldownDuration = 0
	config.InitialConcurrency = 2
	config.MaxConcurrency = 5
	config.InitialRPS = 10
	config.MaxRPS = 20
	config.MonitoringInterval = 50 * time.Millisecond
	
	framework, err := NewEnhancedStressTestFramework(config)
	if err != nil {
		t.Fatalf("创建框架失败: %v", err)
	}
	
	// 添加测试场景
	successCount := int64(0)
	errorCount := int64(0)
	
	scenario := &TestScenario{
		ID:          "success-scenario",
		Name:        "成功场景",
		Description: "总是成功的场景",
		Weight:      1.0,
		Handler: func(ctx context.Context) error {
			successCount++
			time.Sleep(time.Millisecond)
			return nil
		},
		Enabled: true,
	}
	
	err = framework.AddScenario(scenario)
	if err != nil {
		t.Fatalf("添加场景失败: %v", err)
	}
	
	// 添加错误场景
	errorScenario := &TestScenario{
		ID:          "error-scenario",
		Name:        "错误场景",
		Description: "总是失败的场景",
		Weight:      0.1,
		Handler: func(ctx context.Context) error {
			errorCount++
			return errors.New("测试错误")
		},
		Enabled: true,
	}
	
	err = framework.AddScenario(errorScenario)
	if err != nil {
		t.Fatalf("添加错误场景失败: %v", err)
	}
	
	// 启动测试
	err = framework.Start()
	if err != nil {
		t.Fatalf("启动测试失败: %v", err)
	}
	
	if !framework.IsRunning() {
		t.Error("启动后应该在运行状态")
	}
	
	// 等待测试完成
	time.Sleep(200 * time.Millisecond)
	
	// 检查结果
	result := framework.GetResult()
	if result == nil {
		t.Fatal("应该有测试结果")
	}
	
	if result.TotalRequests == 0 {
		t.Error("应该有请求被执行")
	}
	
	if result.Duration <= 0 {
		t.Error("测试持续时间应该大于0")
	}
	
	if result.SuccessfulRequests == 0 {
		t.Error("应该有成功的请求")
	}
	
	if len(result.ScenarioResults) != 2 {
		t.Errorf("期望2个场景结果，实际得到%d个", len(result.ScenarioResults))
	}
	
	// 检查场景结果
	for _, scenarioResult := range result.ScenarioResults {
		if scenarioResult.ExecutionCount == 0 {
			t.Errorf("场景 %s 应该有执行记录", scenarioResult.Name)
		}
	}
	
	t.Logf("测试结果: 总请求=%d, 成功=%d, 失败=%d, 成功率=%.2f%%, 平均延迟=%v",
		result.TotalRequests, result.SuccessfulRequests, result.FailedRequests,
		result.SuccessRate*100, result.AverageLatency)
}

func TestEnhancedStressTestFramework_StopConditions(t *testing.T) {
	config := DefaultEnhancedStressTestConfig()
	config.UseRealData = false
	config.Duration = 5 * time.Second // 较长时间，但会被停止条件中断
	config.WarmupDuration = 0
	config.CooldownDuration = 0
	config.MaxErrors = 5 // 最大5个错误
	config.MaxErrorRate = 0.5 // 最大50%错误率
	config.MonitoringInterval = 10 * time.Millisecond
	
	framework, err := NewEnhancedStressTestFramework(config)
	if err != nil {
		t.Fatalf("创建框架失败: %v", err)
	}
	
	// 添加总是失败的场景
	scenario := &TestScenario{
		ID:          "always-fail",
		Name:        "总是失败",
		Description: "用于测试停止条件",
		Weight:      1.0,
		Handler: func(ctx context.Context) error {
			return errors.New("故意失败")
		},
		Enabled: true,
	}
	
	err = framework.AddScenario(scenario)
	if err != nil {
		t.Fatalf("添加场景失败: %v", err)
	}
	
	// 启动测试
	startTime := time.Now()
	err = framework.Start()
	if err != nil {
		t.Fatalf("启动测试失败: %v", err)
	}
	
	// 等待测试完成（应该被停止条件中断）
	time.Sleep(1 * time.Second)
	
	duration := time.Since(startTime)
	
	// 检查是否提前停止
	if duration >= 4*time.Second {
		t.Error("测试应该被停止条件提前中断")
	}
	
	// 检查结果
	result := framework.GetResult()
	if result == nil {
		t.Fatal("应该有测试结果")
	}
	
	if result.FailedRequests < config.MaxErrors {
		t.Errorf("失败请求数应该达到最大错误数限制 %d，实际为 %d", 
			config.MaxErrors, result.FailedRequests)
	}
	
	t.Logf("测试提前停止: 持续时间=%v, 失败请求=%d, 错误率=%.2f%%",
		duration, result.FailedRequests, result.ErrorRate*100)
}

// 基准测试
func BenchmarkEnhancedStressTestFramework_ScenarioExecution(b *testing.B) {
	config := DefaultEnhancedStressTestConfig()
	config.UseRealData = false
	
	framework, err := NewEnhancedStressTestFramework(config)
	if err != nil {
		b.Fatalf("创建框架失败: %v", err)
	}
	
	scenario := &TestScenario{
		ID:      "benchmark-scenario",
		Name:    "基准测试场景",
		Weight:  1.0,
		Handler: func(ctx context.Context) error {
			return nil
		},
		Enabled: true,
	}
	
	err = framework.AddScenario(scenario)
	if err != nil {
		b.Fatalf("添加场景失败: %v", err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		framework.executeScenario()
	}
}
