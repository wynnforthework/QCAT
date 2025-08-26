package concurrent

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"
)

// ConcurrencyMonitor 并发监控器
type ConcurrencyMonitor struct {
	pools        []*GoroutinePool
	loadBalancer *LoadBalancer
	taskQueue    *TaskQueue
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc

	// 监控配置
	monitorInterval time.Duration
	alertThresholds *AlertThresholds

	// 统计信息
	stats     *MonitorStats
	alerts    []Alert
	maxAlerts int

	// 回调函数
	onAlert       func(alert Alert)
	onHealthCheck func(health HealthStatus)
}

// AlertThresholds 告警阈值
type AlertThresholds struct {
	MaxCPUUsage        float64       // CPU使用率阈值
	MaxMemoryUsage     float64       // 内存使用率阈值
	MaxQueueLength     int           // 队列长度阈值
	MaxActiveTaskRatio float64       // 活跃任务比例阈值
	MaxResponseTime    time.Duration // 响应时间阈值
	MinSuccessRate     float64       // 最小成功率阈值
}

// MonitorStats 监控统计
type MonitorStats struct {
	StartTime           time.Time     `json:"start_time"`
	LastUpdateTime      time.Time     `json:"last_update_time"`
	TotalTasks          int64         `json:"total_tasks"`
	CompletedTasks      int64         `json:"completed_tasks"`
	FailedTasks         int64         `json:"failed_tasks"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	CPUUsage            float64       `json:"cpu_usage"`
	MemoryUsage         float64       `json:"memory_usage"`
	GoroutineCount      int           `json:"goroutine_count"`
	HealthScore         float64       `json:"health_score"`
}

// Alert 告警信息
type Alert struct {
	ID         string                 `json:"id"`
	Type       AlertType              `json:"type"`
	Level      AlertLevel             `json:"level"`
	Message    string                 `json:"message"`
	Timestamp  time.Time              `json:"timestamp"`
	Data       map[string]interface{} `json:"data"`
	Resolved   bool                   `json:"resolved"`
	ResolvedAt *time.Time             `json:"resolved_at,omitempty"`
}

// AlertType 告警类型
type AlertType string

const (
	AlertTypeCPU      AlertType = "cpu"
	AlertTypeMemory   AlertType = "memory"
	AlertTypeQueue    AlertType = "queue"
	AlertTypeTask     AlertType = "task"
	AlertTypeResponse AlertType = "response"
	AlertTypeHealth   AlertType = "health"
)

// AlertLevel 告警级别
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelCritical AlertLevel = "critical"
)

// HealthStatus 健康状态
type HealthStatus struct {
	Overall    string                 `json:"overall"`
	Score      float64                `json:"score"`
	Components map[string]interface{} `json:"components"`
	Timestamp  time.Time              `json:"timestamp"`
}

// NewConcurrencyMonitor 创建并发监控器
func NewConcurrencyMonitor(config *MonitorConfig) *ConcurrencyMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	if config.MonitorInterval <= 0 {
		config.MonitorInterval = 30 * time.Second
	}

	if config.AlertThresholds == nil {
		config.AlertThresholds = &AlertThresholds{
			MaxCPUUsage:        80.0,
			MaxMemoryUsage:     85.0,
			MaxQueueLength:     1000,
			MaxActiveTaskRatio: 0.9,
			MaxResponseTime:    30 * time.Second,
			MinSuccessRate:     0.95,
		}
	}

	return &ConcurrencyMonitor{
		pools:           make([]*GoroutinePool, 0),
		ctx:             ctx,
		cancel:          cancel,
		monitorInterval: config.MonitorInterval,
		alertThresholds: config.AlertThresholds,
		stats: &MonitorStats{
			StartTime: time.Now(),
		},
		alerts:    make([]Alert, 0),
		maxAlerts: 100,
	}
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	MonitorInterval time.Duration
	AlertThresholds *AlertThresholds
}

// AddPool 添加池到监控
func (cm *ConcurrencyMonitor) AddPool(pool *GoroutinePool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.pools = append(cm.pools, pool)
	log.Printf("添加Goroutine池到监控器，当前监控池数量: %d", len(cm.pools))
}

// SetLoadBalancer 设置负载均衡器
func (cm *ConcurrencyMonitor) SetLoadBalancer(lb *LoadBalancer) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.loadBalancer = lb
}

// SetTaskQueue 设置任务队列
func (cm *ConcurrencyMonitor) SetTaskQueue(tq *TaskQueue) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.taskQueue = tq
}

// Start 启动监控
func (cm *ConcurrencyMonitor) Start() {
	log.Println("启动并发监控器...")

	go cm.monitorLoop()

	log.Println("并发监控器已启动")
}

// Stop 停止监控
func (cm *ConcurrencyMonitor) Stop() {
	log.Println("停止并发监控器...")

	cm.cancel()

	log.Println("并发监控器已停止")
}

// monitorLoop 监控循环
func (cm *ConcurrencyMonitor) monitorLoop() {
	ticker := time.NewTicker(cm.monitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cm.collectMetrics()
			cm.checkAlerts()
			cm.performHealthCheck()
		case <-cm.ctx.Done():
			return
		}
	}
}

// collectMetrics 收集指标
func (cm *ConcurrencyMonitor) collectMetrics() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 更新系统指标
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	cm.stats.LastUpdateTime = time.Now()
	cm.stats.GoroutineCount = runtime.NumGoroutine()
	cm.stats.MemoryUsage = float64(memStats.Alloc) / float64(memStats.Sys) * 100

	// 收集池统计信息
	var totalTasks, completedTasks, failedTasks int64
	for _, pool := range cm.pools {
		stats := pool.GetStats()
		totalTasks += stats["total_tasks"].(int64)
		completedTasks += stats["completed_tasks"].(int64)
		failedTasks += stats["failed_tasks"].(int64)
	}

	cm.stats.TotalTasks = totalTasks
	cm.stats.CompletedTasks = completedTasks
	cm.stats.FailedTasks = failedTasks

	// 计算健康分数
	cm.calculateHealthScore()
}

// calculateHealthScore 计算健康分数
func (cm *ConcurrencyMonitor) calculateHealthScore() {
	score := 100.0

	// CPU使用率影响
	if cm.stats.CPUUsage > cm.alertThresholds.MaxCPUUsage {
		score -= (cm.stats.CPUUsage - cm.alertThresholds.MaxCPUUsage) * 0.5
	}

	// 内存使用率影响
	if cm.stats.MemoryUsage > cm.alertThresholds.MaxMemoryUsage {
		score -= (cm.stats.MemoryUsage - cm.alertThresholds.MaxMemoryUsage) * 0.3
	}

	// 任务成功率影响
	if cm.stats.TotalTasks > 0 {
		successRate := float64(cm.stats.CompletedTasks) / float64(cm.stats.TotalTasks)
		if successRate < cm.alertThresholds.MinSuccessRate {
			score -= (cm.alertThresholds.MinSuccessRate - successRate) * 100
		}
	}

	// 确保分数在0-100范围内
	if score < 0 {
		score = 0
	}

	cm.stats.HealthScore = score
}

// checkAlerts 检查告警
func (cm *ConcurrencyMonitor) checkAlerts() {
	// 检查CPU使用率
	if cm.stats.CPUUsage > cm.alertThresholds.MaxCPUUsage {
		cm.createAlert(AlertTypeCPU, AlertLevelWarning,
			fmt.Sprintf("CPU使用率过高: %.2f%%", cm.stats.CPUUsage),
			map[string]interface{}{"cpu_usage": cm.stats.CPUUsage})
	}

	// 检查内存使用率
	if cm.stats.MemoryUsage > cm.alertThresholds.MaxMemoryUsage {
		cm.createAlert(AlertTypeMemory, AlertLevelWarning,
			fmt.Sprintf("内存使用率过高: %.2f%%", cm.stats.MemoryUsage),
			map[string]interface{}{"memory_usage": cm.stats.MemoryUsage})
	}

	// 检查队列长度
	if cm.taskQueue != nil {
		queueSize := cm.taskQueue.Size()
		if queueSize > cm.alertThresholds.MaxQueueLength {
			cm.createAlert(AlertTypeQueue, AlertLevelCritical,
				fmt.Sprintf("任务队列过长: %d", queueSize),
				map[string]interface{}{"queue_size": queueSize})
		}
	}

	// 检查健康分数
	if cm.stats.HealthScore < 70 {
		level := AlertLevelWarning
		if cm.stats.HealthScore < 50 {
			level = AlertLevelCritical
		}
		cm.createAlert(AlertTypeHealth, level,
			fmt.Sprintf("系统健康分数过低: %.2f", cm.stats.HealthScore),
			map[string]interface{}{"health_score": cm.stats.HealthScore})
	}
}

// createAlert 创建告警
func (cm *ConcurrencyMonitor) createAlert(alertType AlertType, level AlertLevel, message string, data map[string]interface{}) {
	alert := Alert{
		ID:        fmt.Sprintf("%s_%d", alertType, time.Now().Unix()),
		Type:      alertType,
		Level:     level,
		Message:   message,
		Timestamp: time.Now(),
		Data:      data,
		Resolved:  false,
	}

	// 添加到告警列表
	cm.alerts = append(cm.alerts, alert)

	// 限制告警数量
	if len(cm.alerts) > cm.maxAlerts {
		cm.alerts = cm.alerts[len(cm.alerts)-cm.maxAlerts:]
	}

	// 调用回调函数
	if cm.onAlert != nil {
		go cm.onAlert(alert)
	}

	log.Printf("🚨 [%s] %s: %s", level, alertType, message)
}

// performHealthCheck 执行健康检查
func (cm *ConcurrencyMonitor) performHealthCheck() {
	health := HealthStatus{
		Score:     cm.stats.HealthScore,
		Timestamp: time.Now(),
		Components: map[string]interface{}{
			"cpu_usage":       cm.stats.CPUUsage,
			"memory_usage":    cm.stats.MemoryUsage,
			"goroutine_count": cm.stats.GoroutineCount,
			"total_tasks":     cm.stats.TotalTasks,
			"completed_tasks": cm.stats.CompletedTasks,
			"failed_tasks":    cm.stats.FailedTasks,
		},
	}

	// 确定整体健康状态
	if health.Score >= 80 {
		health.Overall = "healthy"
	} else if health.Score >= 60 {
		health.Overall = "warning"
	} else {
		health.Overall = "critical"
	}

	// 调用回调函数
	if cm.onHealthCheck != nil {
		go cm.onHealthCheck(health)
	}
}

// GetStats 获取监控统计信息
func (cm *ConcurrencyMonitor) GetStats() *MonitorStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// 返回副本
	statsCopy := *cm.stats
	return &statsCopy
}

// GetAlerts 获取告警列表
func (cm *ConcurrencyMonitor) GetAlerts() []Alert {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// 返回副本
	alertsCopy := make([]Alert, len(cm.alerts))
	copy(alertsCopy, cm.alerts)
	return alertsCopy
}

// SetOnAlert 设置告警回调
func (cm *ConcurrencyMonitor) SetOnAlert(callback func(alert Alert)) {
	cm.onAlert = callback
}

// SetOnHealthCheck 设置健康检查回调
func (cm *ConcurrencyMonitor) SetOnHealthCheck(callback func(health HealthStatus)) {
	cm.onHealthCheck = callback
}
