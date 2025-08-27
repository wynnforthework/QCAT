package testing

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEnhancedPerformanceMonitor_Creation(t *testing.T) {
	config := DefaultPerformanceMonitorConfig()
	config.EnableReports = false // 禁用报告以简化测试
	config.EnableAlerts = false  // 禁用告警以简化测试
	
	monitor, err := NewEnhancedPerformanceMonitor(config)
	if err != nil {
		t.Fatalf("创建增强版性能监控器失败: %v", err)
	}
	
	if monitor == nil {
		t.Fatal("监控器不应该为空")
	}
	
	if monitor.IsRunning() {
		t.Error("新创建的监控器不应该在运行")
	}
	
	if monitor.config.MonitorInterval != config.MonitorInterval {
		t.Errorf("期望监控间隔 %v，实际得到 %v", config.MonitorInterval, monitor.config.MonitorInterval)
	}
}

func TestEnhancedPerformanceMonitor_DefaultConfig(t *testing.T) {
	config := DefaultPerformanceMonitorConfig()
	
	if config == nil {
		t.Fatal("默认配置不应该为空")
	}
	
	if config.MonitorInterval <= 0 {
		t.Error("监控间隔应该大于0")
	}
	
	if config.ReportInterval <= 0 {
		t.Error("报告间隔应该大于0")
	}
	
	if config.MaxDataPoints <= 0 {
		t.Error("最大数据点数应该大于0")
	}
	
	if config.DataRetention <= 0 {
		t.Error("数据保留时间应该大于0")
	}
	
	if !config.EnableSystemMetrics {
		t.Error("默认应该启用系统指标监控")
	}
}

func TestEnhancedPerformanceMonitor_StartStop(t *testing.T) {
	config := DefaultPerformanceMonitorConfig()
	config.MonitorInterval = 100 * time.Millisecond
	config.EnableReports = false
	config.EnableAlerts = false
	
	monitor, err := NewEnhancedPerformanceMonitor(config)
	if err != nil {
		t.Fatalf("创建监控器失败: %v", err)
	}
	
	// 测试启动
	err = monitor.Start()
	if err != nil {
		t.Fatalf("启动监控器失败: %v", err)
	}
	
	if !monitor.IsRunning() {
		t.Error("启动后应该在运行状态")
	}
	
	// 测试重复启动
	err = monitor.Start()
	if err == nil {
		t.Error("重复启动应该返回错误")
	}
	
	// 等待一段时间让监控器收集数据
	time.Sleep(300 * time.Millisecond)
	
	// 检查是否收集到了数据
	stats := monitor.stats
	if stats.TotalMetrics == 0 {
		t.Error("应该收集到一些指标")
	}
	
	// 测试停止
	monitor.Stop()
	
	if monitor.IsRunning() {
		t.Error("停止后不应该在运行状态")
	}
}

func TestEnhancedPerformanceMonitor_SystemMetrics(t *testing.T) {
	config := DefaultPerformanceMonitorConfig()
	config.MonitorInterval = 50 * time.Millisecond
	config.EnableSystemMetrics = true
	config.EnableNetworkMetrics = false
	config.EnableAppMetrics = false
	config.EnableReports = false
	config.EnableAlerts = false
	
	monitor, err := NewEnhancedPerformanceMonitor(config)
	if err != nil {
		t.Fatalf("创建监控器失败: %v", err)
	}
	
	if monitor.systemMetrics == nil {
		t.Fatal("系统指标收集器不应该为空")
	}
	
	// 启动监控器
	err = monitor.Start()
	if err != nil {
		t.Fatalf("启动监控器失败: %v", err)
	}
	defer monitor.Stop()
	
	// 等待收集数据
	time.Sleep(200 * time.Millisecond)
	
	// 检查CPU使用率指标
	cpuCollector := monitor.systemMetrics.cpuUsage
	cpuCollector.mutex.RLock()
	cpuDataPoints := len(cpuCollector.Values)
	cpuCollector.mutex.RUnlock()
	
	if cpuDataPoints == 0 {
		t.Error("应该收集到CPU使用率数据")
	}
	
	// 检查内存使用率指标
	memCollector := monitor.systemMetrics.memoryUsage
	memCollector.mutex.RLock()
	memDataPoints := len(memCollector.Values)
	memCollector.mutex.RUnlock()
	
	if memDataPoints == 0 {
		t.Error("应该收集到内存使用率数据")
	}
	
	// 检查Goroutine数量指标
	goroutineCollector := monitor.systemMetrics.goroutineCount
	goroutineCollector.mutex.RLock()
	goroutineDataPoints := len(goroutineCollector.Values)
	goroutineCollector.mutex.RUnlock()
	
	if goroutineDataPoints == 0 {
		t.Error("应该收集到Goroutine数量数据")
	}
	
	t.Logf("收集到数据点: CPU=%d, 内存=%d, Goroutine=%d", 
		cpuDataPoints, memDataPoints, goroutineDataPoints)
}

func TestEnhancedPerformanceMonitor_MetricCollector(t *testing.T) {
	collector := &EnhancedMetricCollector{
		Name:        "test_metric",
		Unit:        "percent",
		Description: "测试指标",
		MetricType:  MetricTypeGauge,
		Min:         1000000, // 设置一个大值作为初始最小值
		Labels:      make(map[string]string),
	}
	
	// 记录一些测试数据
	testValues := []float64{10.5, 20.3, 15.7, 25.1, 18.9}
	now := time.Now()
	
	for i, value := range testValues {
		collector.RecordValue(value, now.Add(time.Duration(i)*time.Second))
	}
	
	// 检查统计信息
	if collector.Count != int64(len(testValues)) {
		t.Errorf("期望数据点数量 %d，实际得到 %d", len(testValues), collector.Count)
	}
	
	if collector.Min > 10.5 {
		t.Errorf("期望最小值 <= 10.5，实际得到 %f", collector.Min)
	}
	
	if collector.Max < 25.1 {
		t.Errorf("期望最大值 >= 25.1，实际得到 %f", collector.Max)
	}
	
	expectedMean := (10.5 + 20.3 + 15.7 + 25.1 + 18.9) / 5
	if collector.Mean < expectedMean-0.1 || collector.Mean > expectedMean+0.1 {
		t.Errorf("期望平均值约为 %.2f，实际得到 %.2f", expectedMean, collector.Mean)
	}
	
	// 检查百分位数
	if collector.P50 == 0 {
		t.Error("P50百分位数不应该为0")
	}
	
	if collector.P95 == 0 {
		t.Error("P95百分位数不应该为0")
	}
	
	t.Logf("指标统计: 平均=%.2f, 最小=%.2f, 最大=%.2f, P50=%.2f, P95=%.2f", 
		collector.Mean, collector.Min, collector.Max, collector.P50, collector.P95)
}

func TestEnhancedPerformanceMonitor_ReportGeneration(t *testing.T) {
	// 创建临时目录用于测试报告输出
	tempDir, err := os.MkdirTemp("", "performance_test_")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	config := DefaultPerformanceMonitorConfig()
	config.MonitorInterval = 50 * time.Millisecond
	config.ReportInterval = 200 * time.Millisecond
	config.EnableReports = true
	config.ReportFormats = []string{"json", "csv"}
	config.ReportOutputDir = tempDir
	config.EnableAlerts = false
	
	monitor, err := NewEnhancedPerformanceMonitor(config)
	if err != nil {
		t.Fatalf("创建监控器失败: %v", err)
	}
	
	if monitor.reportGenerator == nil {
		t.Fatal("报告生成器不应该为空")
	}
	
	// 启动监控器
	err = monitor.Start()
	if err != nil {
		t.Fatalf("启动监控器失败: %v", err)
	}
	
	// 等待生成报告
	time.Sleep(500 * time.Millisecond)
	
	monitor.Stop()
	
	// 检查是否生成了报告文件
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("读取报告目录失败: %v", err)
	}
	
	jsonFound := false
	csvFound := false
	
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			jsonFound = true
		}
		if filepath.Ext(file.Name()) == ".csv" {
			csvFound = true
		}
	}
	
	if !jsonFound {
		t.Error("应该生成JSON格式的报告文件")
	}
	
	if !csvFound {
		t.Error("应该生成CSV格式的报告文件")
	}
	
	t.Logf("生成了 %d 个报告文件", len(files))
}

func TestEnhancedPerformanceMonitor_AlertManager(t *testing.T) {
	thresholds := map[string]AlertThreshold{
		"test_metric": {
			MetricName:    "test_metric",
			WarningLevel:  80.0,
			CriticalLevel: 95.0,
			Enabled:       true,
		},
	}
	
	alertManager := NewAlertManager(thresholds)
	if alertManager == nil {
		t.Fatal("告警管理器不应该为空")
	}
	
	// 测试正常值（不触发告警）
	alertManager.CheckMetric("test_metric", 50.0)
	
	if len(alertManager.alertHistory) != 0 {
		t.Error("正常值不应该触发告警")
	}
	
	// 测试警告级别
	alertManager.CheckMetric("test_metric", 85.0)
	
	if len(alertManager.alertHistory) != 1 {
		t.Error("应该触发一个警告级别告警")
	}
	
	if alertManager.alertHistory[0].Level != AlertLevelWarning {
		t.Error("应该是警告级别告警")
	}
	
	// 测试严重级别
	alertManager.CheckMetric("test_metric", 98.0)
	
	if len(alertManager.alertHistory) != 2 {
		t.Error("应该触发两个告警")
	}
	
	if alertManager.alertHistory[1].Level != AlertLevelCritical {
		t.Error("第二个告警应该是严重级别")
	}
	
	t.Logf("触发了 %d 个告警", len(alertManager.alertHistory))
}

// 基准测试
func BenchmarkEnhancedMetricCollector_RecordValue(b *testing.B) {
	collector := &EnhancedMetricCollector{
		Name:   "benchmark_metric",
		Unit:   "count",
		Min:    1000000,
		Labels: make(map[string]string),
	}
	
	now := time.Now()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.RecordValue(float64(i), now)
	}
}
