package bridge

import (
	"log"
	"time"

	"qcat/internal/monitor"
)

// MonitorIntegration 监控集成器
// 将现有的监控系统与响应桥接器连接
type MonitorIntegration struct {
	bridge           *MonitorResponseBridge
	metricsCollector *monitor.MetricsCollector
	alertManager     *monitor.AlertManager
	// 注意：暂时不使用HealthChecker，因为接口不匹配
}

// NewMonitorIntegration 创建监控集成器
func NewMonitorIntegration(
	bridge *MonitorResponseBridge,
	metrics *monitor.MetricsCollector,
	alertManager *monitor.AlertManager,
	healthChecker interface{}, // 暂时使用interface{}
) *MonitorIntegration {
	return &MonitorIntegration{
		bridge:           bridge,
		metricsCollector: metrics,
		alertManager:     alertManager,
	}
}

// Start 启动监控集成
func (mi *MonitorIntegration) Start() error {
	log.Println("Starting monitor integration...")

	// 启动指标监控
	if mi.metricsCollector != nil {
		go mi.monitorMetrics()
		log.Println("Started metrics monitoring")
	}

	log.Println("Monitor integration started successfully")
	return nil
}

// convertSeverity 转换严重程度
func (mi *MonitorIntegration) convertSeverity(alertSeverity monitor.AlertSeverity) EventSeverity {
	switch alertSeverity {
	case monitor.AlertSeverityInfo:
		return SeverityInfo
	case monitor.AlertSeverityWarning:
		return SeverityWarning
	case monitor.AlertSeverityCritical:
		return SeverityCritical
	default:
		return SeverityInfo
	}
}

// monitorSystemHealth 监控系统健康状态（简化版）
func (mi *MonitorIntegration) monitorSystemHealth() {
	ticker := time.NewTicker(time.Minute * 5) // 每5分钟检查一次
	defer ticker.Stop()

	for range ticker.C {
		// 简单的系统健康检查
		event := &MonitorEvent{
			Type:     EventTypeSystem,
			Source:   "system_monitor",
			Severity: SeverityInfo,
			Message:  "System health check completed",
			Metadata: map[string]interface{}{
				"timestamp": time.Now(),
			},
		}

		if err := mi.bridge.ProcessEvent(event); err != nil {
			log.Printf("Failed to process system health event: %v", err)
		}
	}
}

// monitorMetrics 监控指标
func (mi *MonitorIntegration) monitorMetrics() {
	ticker := time.NewTicker(time.Minute * 2) // 每2分钟检查一次
	defer ticker.Stop()

	for range ticker.C {
		if mi.metricsCollector == nil {
			continue
		}

		// 简单的指标监控事件
		event := &MonitorEvent{
			Type:     EventTypePerformance,
			Source:   "metrics_collector",
			Severity: SeverityInfo,
			Message:  "Metrics monitoring check completed",
			Metadata: map[string]interface{}{
				"timestamp": time.Now(),
			},
		}

		if err := mi.bridge.ProcessEvent(event); err != nil {
			log.Printf("Failed to process metrics event: %v", err)
		}
	}
}

// CreateRiskViolationEvent 创建风险违规事件
func (mi *MonitorIntegration) CreateRiskViolationEvent(riskType, message string, severity EventSeverity, metadata map[string]interface{}) error {
	event := &MonitorEvent{
		Type:     EventTypeRiskViolation,
		Source:   "risk_monitor",
		Severity: severity,
		Message:  message,
		Metadata: metadata,
	}

	if event.Metadata == nil {
		event.Metadata = make(map[string]interface{})
	}
	event.Metadata["risk_type"] = riskType

	return mi.bridge.ProcessEvent(event)
}

// CreateStrategyEvent 创建策略事件
func (mi *MonitorIntegration) CreateStrategyEvent(strategyID, message string, severity EventSeverity, metadata map[string]interface{}) error {
	event := &MonitorEvent{
		Type:     EventTypeStrategy,
		Source:   "strategy_monitor",
		Severity: severity,
		Message:  message,
		Metadata: metadata,
	}

	if event.Metadata == nil {
		event.Metadata = make(map[string]interface{})
	}
	event.Metadata["strategy_id"] = strategyID

	return mi.bridge.ProcessEvent(event)
}

// CreateMarketEvent 创建市场事件
func (mi *MonitorIntegration) CreateMarketEvent(symbol, message string, severity EventSeverity, metadata map[string]interface{}) error {
	event := &MonitorEvent{
		Type:     EventTypeMarket,
		Source:   "market_monitor",
		Severity: severity,
		Message:  message,
		Metadata: metadata,
	}

	if event.Metadata == nil {
		event.Metadata = make(map[string]interface{})
	}
	event.Metadata["symbol"] = symbol

	return mi.bridge.ProcessEvent(event)
}
