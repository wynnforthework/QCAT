package guardian

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"math"
	"net"
	"sync"
	"time"

	"qcat/internal/config"
)

// AccountGuardian 账户安全监控守护者
type AccountGuardian struct {
	config           *config.Config
	behaviorAnalyzer *BehaviorAnalyzer
	threatDetector   *ThreatDetector
	responseHandler  *ResponseHandler

	// 运行状态
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	isRunning bool
	mu        sync.RWMutex

	// 监控数据
	userSessions    map[string]*UserSession
	threatEvents    []ThreatEvent
	anomalyEvents   []AnomalyEvent
	securityMetrics *SecurityMetrics

	// 配置参数
	baselineDays      int
	anomalyThreshold  float64
	alertThreshold    float64
	autoFreezeEnabled bool
}

// UserSession 用户会话信息
type UserSession struct {
	UserID        string           `json:"user_id"`
	SessionID     string           `json:"session_id"`
	IPAddress     string           `json:"ip_address"`
	UserAgent     string           `json:"user_agent"`
	LoginTime     time.Time        `json:"login_time"`
	LastActivity  time.Time        `json:"last_activity"`
	Location      *GeoLocation     `json:"location"`
	DeviceInfo    *DeviceInfo      `json:"device_info"`
	Activities    []ActivityRecord `json:"activities"`
	RiskScore     float64          `json:"risk_score"`
	IsWhitelisted bool             `json:"is_whitelisted"`
}

// GeoLocation 地理位置信息
type GeoLocation struct {
	Country   string  `json:"country"`
	City      string  `json:"city"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	ISP       string  `json:"isp"`
}

// DeviceInfo 设备信息
type DeviceInfo struct {
	DeviceID    string `json:"device_id"`
	Platform    string `json:"platform"`
	Browser     string `json:"browser"`
	OS          string `json:"os"`
	Fingerprint string `json:"fingerprint"`
	IsTrusted   bool   `json:"is_trusted"`
}

// ActivityRecord 活动记录
type ActivityRecord struct {
	Type      string                 `json:"type"`
	Action    string                 `json:"action"`
	Resource  string                 `json:"resource"`
	Timestamp time.Time              `json:"timestamp"`
	IPAddress string                 `json:"ip_address"`
	Result    string                 `json:"result"`
	Metadata  map[string]interface{} `json:"metadata"`
	RiskScore float64                `json:"risk_score"`
}

// ThreatEvent 威胁事件
type ThreatEvent struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	UserID      string                 `json:"user_id"`
	IPAddress   string                 `json:"ip_address"`
	Timestamp   time.Time              `json:"timestamp"`
	Indicators  []ThreatIndicator      `json:"indicators"`
	Response    *ThreatResponse        `json:"response"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ThreatIndicator 威胁指标
type ThreatIndicator struct {
	Type        string  `json:"type"`
	Value       string  `json:"value"`
	Confidence  float64 `json:"confidence"`
	Description string  `json:"description"`
}

// ThreatResponse 威胁响应
type ThreatResponse struct {
	Action      string    `json:"action"`
	Timestamp   time.Time `json:"timestamp"`
	Automated   bool      `json:"automated"`
	Description string    `json:"description"`
}

// AnomalyEvent 异常事件
type AnomalyEvent struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Score       float64                `json:"score"`
	Description string                 `json:"description"`
	UserID      string                 `json:"user_id"`
	Timestamp   time.Time              `json:"timestamp"`
	Baseline    *BaselineData          `json:"baseline"`
	Current     *CurrentData           `json:"current"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// BaselineData 基线数据
type BaselineData struct {
	Mean         float64   `json:"mean"`
	StdDev       float64   `json:"std_dev"`
	SampleCount  int       `json:"sample_count"`
	CalculatedAt time.Time `json:"calculated_at"`
}

// CurrentData 当前数据
type CurrentData struct {
	Value      float64   `json:"value"`
	ZScore     float64   `json:"z_score"`
	Percentile float64   `json:"percentile"`
	ObservedAt time.Time `json:"observed_at"`
}

// SecurityMetrics 安全指标
type SecurityMetrics struct {
	mu sync.RWMutex

	// 检测统计
	ThreatsDetected   int64 `json:"threats_detected"`
	AnomaliesDetected int64 `json:"anomalies_detected"`
	FalsePositives    int64 `json:"false_positives"`
	TruePositives     int64 `json:"true_positives"`

	// 响应统计
	AutomatedResponses  int64 `json:"automated_responses"`
	ManualInterventions int64 `json:"manual_interventions"`
	AccountsFrozen      int64 `json:"accounts_frozen"`

	// 性能指标
	DetectionLatency time.Duration `json:"detection_latency"`
	ResponseLatency  time.Duration `json:"response_latency"`
	SystemUptime     time.Duration `json:"system_uptime"`

	// 准确率
	DetectionAccuracy float64 `json:"detection_accuracy"`
	PrecisionRate     float64 `json:"precision_rate"`
	RecallRate        float64 `json:"recall_rate"`

	LastUpdated time.Time `json:"last_updated"`
}

// BehaviorAnalyzer 行为分析器
type BehaviorAnalyzer struct {
	userBaselines map[string]*UserBaseline
	mu            sync.RWMutex
}

// UserBaseline 用户基线
type UserBaseline struct {
	UserID           string           `json:"user_id"`
	LoginFrequency   *StatisticalData `json:"login_frequency"`
	SessionDuration  *StatisticalData `json:"session_duration"`
	ActiveHours      map[int]float64  `json:"active_hours"`
	IPAddresses      map[string]int   `json:"ip_addresses"`
	Locations        map[string]int   `json:"locations"`
	TradingVolume    *StatisticalData `json:"trading_volume"`
	TradingFrequency *StatisticalData `json:"trading_frequency"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

// StatisticalData 统计数据
type StatisticalData struct {
	Mean      float64   `json:"mean"`
	StdDev    float64   `json:"std_dev"`
	Min       float64   `json:"min"`
	Max       float64   `json:"max"`
	Count     int64     `json:"count"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ThreatDetector 威胁检测器
type ThreatDetector struct {
	ipWhitelist   map[string]bool
	blacklist     map[string]bool
	suspiciousIPs map[string]*IPRiskData
	mu            sync.RWMutex
}

// IPRiskData IP风险数据
type IPRiskData struct {
	IPAddress      string       `json:"ip_address"`
	RiskScore      float64      `json:"risk_score"`
	FailedAttempts int          `json:"failed_attempts"`
	LastSeen       time.Time    `json:"last_seen"`
	GeoLocation    *GeoLocation `json:"geo_location"`
	IsProxy        bool         `json:"is_proxy"`
	IsTor          bool         `json:"is_tor"`
	IsVPN          bool         `json:"is_vpn"`
}

// ResponseHandler 响应处理器
type ResponseHandler struct {
	escalationRules      []EscalationRule
	notificationChannels map[string]NotificationChannel
	mu                   sync.RWMutex
}

// EscalationRule 升级规则
type EscalationRule struct {
	ThreatType           string        `json:"threat_type"`
	Severity             string        `json:"severity"`
	AutoResponse         string        `json:"auto_response"`
	EscalationDelay      time.Duration `json:"escalation_delay"`
	NotificationChannels []string      `json:"notification_channels"`
}

// NotificationChannel 通知渠道
type NotificationChannel struct {
	Type      string            `json:"type"`
	Endpoint  string            `json:"endpoint"`
	Config    map[string]string `json:"config"`
	IsEnabled bool              `json:"is_enabled"`
}

// NewAccountGuardian 创建账户安全监控守护者
func NewAccountGuardian(cfg *config.Config) (*AccountGuardian, error) {
	ctx, cancel := context.WithCancel(context.Background())

	ag := &AccountGuardian{
		config:            cfg,
		behaviorAnalyzer:  NewBehaviorAnalyzer(),
		threatDetector:    NewThreatDetector(),
		responseHandler:   NewResponseHandler(),
		ctx:               ctx,
		cancel:            cancel,
		userSessions:      make(map[string]*UserSession),
		threatEvents:      make([]ThreatEvent, 0),
		anomalyEvents:     make([]AnomalyEvent, 0),
		securityMetrics:   &SecurityMetrics{},
		baselineDays:      30,
		anomalyThreshold:  3.0,
		alertThreshold:    2.0,
		autoFreezeEnabled: true,
	}

	// 从配置文件读取参数
	if cfg != nil {
		// 从配置文件读取安全参数
		if cfg.Security.Audit.RetentionDays > 0 {
			ag.baselineDays = cfg.Security.Audit.RetentionDays
		}

		// 从监控配置读取阈值参数
		if cfg.Monitoring.Alerts.ErrorRatePercent > 0 {
			ag.anomalyThreshold = float64(cfg.Monitoring.Alerts.ErrorRatePercent) / 10.0 // 转换为合适的异常阈值
		}

		if cfg.Monitoring.Alerts.HighLatencyMs > 0 {
			ag.alertThreshold = float64(cfg.Monitoring.Alerts.HighLatencyMs) / 1000.0 // 转换为秒作为警报阈值
		}

		// 从安全配置读取自动冻结设置
		if len(cfg.Security.Network.AllowedOrigins) > 0 {
			ag.autoFreezeEnabled = true // 如果配置了网络安全，启用自动冻结
		}

		// 设置IP白名单
		if len(cfg.RateLimit.WhitelistIPs) > 0 {
			for _, ip := range cfg.RateLimit.WhitelistIPs {
				ag.threatDetector.ipWhitelist[ip] = true
			}
		}

		log.Printf("从配置文件加载安全参数: baseline_days=%d, anomaly_threshold=%.2f, alert_threshold=%.2f, auto_freeze=%v",
			ag.baselineDays, ag.anomalyThreshold, ag.alertThreshold, ag.autoFreezeEnabled)
	}

	return ag, nil
}

// NewBehaviorAnalyzer 创建行为分析器
func NewBehaviorAnalyzer() *BehaviorAnalyzer {
	return &BehaviorAnalyzer{
		userBaselines: make(map[string]*UserBaseline),
	}
}

// NewThreatDetector 创建威胁检测器
func NewThreatDetector() *ThreatDetector {
	return &ThreatDetector{
		ipWhitelist:   make(map[string]bool),
		blacklist:     make(map[string]bool),
		suspiciousIPs: make(map[string]*IPRiskData),
	}
}

// NewResponseHandler 创建响应处理器
func NewResponseHandler() *ResponseHandler {
	return &ResponseHandler{
		escalationRules:      make([]EscalationRule, 0),
		notificationChannels: make(map[string]NotificationChannel),
	}
}

// Start 启动账户安全监控
func (ag *AccountGuardian) Start() error {
	ag.mu.Lock()
	defer ag.mu.Unlock()

	if ag.isRunning {
		return fmt.Errorf("account guardian is already running")
	}

	log.Println("Starting Account Guardian...")

	// 启动行为分析
	ag.wg.Add(1)
	go ag.runBehaviorAnalysis()

	// 启动威胁检测
	ag.wg.Add(1)
	go ag.runThreatDetection()

	// 启动响应处理
	ag.wg.Add(1)
	go ag.runResponseHandling()

	// 启动指标收集
	ag.wg.Add(1)
	go ag.runMetricsCollection()

	ag.isRunning = true
	log.Println("Account Guardian started successfully")
	return nil
}

// Stop 停止账户安全监控
func (ag *AccountGuardian) Stop() error {
	ag.mu.Lock()
	defer ag.mu.Unlock()

	if !ag.isRunning {
		return fmt.Errorf("account guardian is not running")
	}

	log.Println("Stopping Account Guardian...")

	ag.cancel()
	ag.wg.Wait()

	ag.isRunning = false
	log.Println("Account Guardian stopped successfully")
	return nil
}

// runBehaviorAnalysis 运行行为分析
func (ag *AccountGuardian) runBehaviorAnalysis() {
	defer ag.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.Println("Behavior analysis started")

	for {
		select {
		case <-ag.ctx.Done():
			log.Println("Behavior analysis stopped")
			return
		case <-ticker.C:
			ag.analyzeBehavior()
		}
	}
}

// runThreatDetection 运行威胁检测
func (ag *AccountGuardian) runThreatDetection() {
	defer ag.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Println("Threat detection started")

	for {
		select {
		case <-ag.ctx.Done():
			log.Println("Threat detection stopped")
			return
		case <-ticker.C:
			ag.detectThreats()
		}
	}
}

// runResponseHandling 运行响应处理
func (ag *AccountGuardian) runResponseHandling() {
	defer ag.wg.Done()

	log.Println("Response handling started")

	for {
		select {
		case <-ag.ctx.Done():
			log.Println("Response handling stopped")
			return
		default:
			ag.handlePendingResponses()
			time.Sleep(5 * time.Second)
		}
	}
}

// runMetricsCollection 运行指标收集
func (ag *AccountGuardian) runMetricsCollection() {
	defer ag.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Println("Metrics collection started")

	for {
		select {
		case <-ag.ctx.Done():
			log.Println("Metrics collection stopped")
			return
		case <-ticker.C:
			ag.updateSecurityMetrics()
		}
	}
}

// RecordUserActivity 记录用户活动
func (ag *AccountGuardian) RecordUserActivity(userID, sessionID, ipAddress, userAgent string, activity ActivityRecord) error {
	ag.mu.Lock()
	defer ag.mu.Unlock()

	// 获取或创建用户会话
	session, exists := ag.userSessions[sessionID]
	if !exists {
		session = &UserSession{
			UserID:        userID,
			SessionID:     sessionID,
			IPAddress:     ipAddress,
			UserAgent:     userAgent,
			LoginTime:     time.Now(),
			LastActivity:  time.Now(),
			Activities:    make([]ActivityRecord, 0),
			RiskScore:     0.0,
			IsWhitelisted: ag.isIPWhitelisted(ipAddress),
		}

		// 获取地理位置和设备信息
		session.Location = ag.getGeoLocation(ipAddress)
		session.DeviceInfo = ag.getDeviceInfo(userAgent)

		ag.userSessions[sessionID] = session
	}

	// 更新活动时间
	session.LastActivity = time.Now()

	// 添加活动记录
	activity.Timestamp = time.Now()
	activity.IPAddress = ipAddress
	activity.RiskScore = ag.calculateActivityRisk(activity, session)
	session.Activities = append(session.Activities, activity)

	// 更新会话风险分数
	session.RiskScore = ag.calculateSessionRisk(session)

	// 检查是否需要触发异常检测
	if session.RiskScore > ag.alertThreshold {
		ag.triggerAnomalyDetection(session, activity)
	}

	return nil
}

// analyzeBehavior 分析用户行为
func (ag *AccountGuardian) analyzeBehavior() {
	log.Println("Analyzing user behavior patterns...")

	ag.mu.RLock()
	sessions := make(map[string]*UserSession)
	for k, v := range ag.userSessions {
		sessions[k] = v
	}
	ag.mu.RUnlock()

	for _, session := range sessions {
		// 分析单个用户行为
		ag.analyzeUserBehavior(session)
	}
}

// analyzeUserBehavior 分析单个用户行为
func (ag *AccountGuardian) analyzeUserBehavior(session *UserSession) {
	// 获取用户基线
	baseline := ag.behaviorAnalyzer.getUserBaseline(session.UserID)
	if baseline == nil {
		// 创建新的基线
		baseline = ag.behaviorAnalyzer.createUserBaseline(session.UserID)
	}

	// 检测异常
	anomalies := ag.detectBehaviorAnomalies(session, baseline)

	// 处理检测到的异常
	for _, anomaly := range anomalies {
		ag.handleAnomalyEvent(anomaly)
	}

	// 更新用户基线
	ag.behaviorAnalyzer.updateUserBaseline(session, baseline)
}

// detectThreats 检测威胁
func (ag *AccountGuardian) detectThreats() {
	log.Println("Detecting security threats...")

	ag.mu.RLock()
	sessions := make(map[string]*UserSession)
	for k, v := range ag.userSessions {
		sessions[k] = v
	}
	ag.mu.RUnlock()

	for _, session := range sessions {
		threats := ag.detectSessionThreats(session)
		for _, threat := range threats {
			ag.handleThreatEvent(threat)
		}
	}
}

// detectSessionThreats 检测会话威胁
func (ag *AccountGuardian) detectSessionThreats(session *UserSession) []ThreatEvent {
	threats := make([]ThreatEvent, 0)

	// 检测可疑IP
	if ipThreat := ag.checkSuspiciousIP(session); ipThreat != nil {
		threats = append(threats, *ipThreat)
	}

	// 检测异常登录
	if loginThreat := ag.checkAbnormalLogin(session); loginThreat != nil {
		threats = append(threats, *loginThreat)
	}

	// 检测异常交易模式
	if tradingThreat := ag.checkAbnormalTrading(session); tradingThreat != nil {
		threats = append(threats, *tradingThreat)
	}

	// 检测设备指纹异常
	if deviceThreat := ag.checkDeviceAnomaly(session); deviceThreat != nil {
		threats = append(threats, *deviceThreat)
	}

	return threats
}

// Helper functions for threat detection
func (ag *AccountGuardian) checkSuspiciousIP(session *UserSession) *ThreatEvent {
	ag.threatDetector.mu.RLock()
	defer ag.threatDetector.mu.RUnlock()

	// 检查黑名单
	if ag.threatDetector.blacklist[session.IPAddress] {
		return &ThreatEvent{
			ID:          ag.generateEventID(),
			Type:        "BLACKLISTED_IP",
			Severity:    "HIGH",
			Description: fmt.Sprintf("Login from blacklisted IP: %s", session.IPAddress),
			UserID:      session.UserID,
			IPAddress:   session.IPAddress,
			Timestamp:   time.Now(),
			Indicators: []ThreatIndicator{
				{
					Type:        "IP_BLACKLIST",
					Value:       session.IPAddress,
					Confidence:  1.0,
					Description: "IP address is in security blacklist",
				},
			},
		}
	}

	// 检查可疑IP
	if ipRisk, exists := ag.threatDetector.suspiciousIPs[session.IPAddress]; exists {
		if ipRisk.RiskScore > 0.8 {
			return &ThreatEvent{
				ID:          ag.generateEventID(),
				Type:        "SUSPICIOUS_IP",
				Severity:    "MEDIUM",
				Description: fmt.Sprintf("Login from suspicious IP: %s (risk score: %.2f)", session.IPAddress, ipRisk.RiskScore),
				UserID:      session.UserID,
				IPAddress:   session.IPAddress,
				Timestamp:   time.Now(),
				Indicators: []ThreatIndicator{
					{
						Type:        "IP_RISK_SCORE",
						Value:       fmt.Sprintf("%.2f", ipRisk.RiskScore),
						Confidence:  ipRisk.RiskScore,
						Description: "IP address has elevated risk score",
					},
				},
			}
		}
	}

	return nil
}

// 其他辅助函数的实现...
func (ag *AccountGuardian) checkAbnormalLogin(session *UserSession) *ThreatEvent {
	// 实现异常登录检测逻辑

	// 1. 检查登录时间异常（非正常工作时间）
	hour := session.LoginTime.Hour()
	if hour < 6 || hour > 22 {
		return &ThreatEvent{
			ID:          fmt.Sprintf("abnormal_login_time_%s_%d", session.UserID, time.Now().Unix()),
			Type:        "ABNORMAL_LOGIN_TIME",
			Severity:    "MEDIUM",
			Description: fmt.Sprintf("User %s logged in at unusual hour: %d", session.UserID, hour),
			UserID:      session.UserID,
			IPAddress:   session.IPAddress,
			Timestamp:   time.Now(),
			Metadata: map[string]interface{}{
				"login_hour": hour,
				"session_id": session.SessionID,
			},
		}
	}

	// 2. 检查地理位置异常
	if session.Location != nil {
		// 检查是否来自高风险国家
		highRiskCountries := []string{"CN", "RU", "KP", "IR"}
		for _, country := range highRiskCountries {
			if session.Location.Country == country {
				return &ThreatEvent{
					ID:          fmt.Sprintf("high_risk_location_%s_%d", session.UserID, time.Now().Unix()),
					Type:        "HIGH_RISK_LOCATION",
					Severity:    "HIGH",
					Description: fmt.Sprintf("User %s logged in from high-risk country: %s", session.UserID, country),
					UserID:      session.UserID,
					IPAddress:   session.IPAddress,
					Timestamp:   time.Now(),
					Metadata: map[string]interface{}{
						"country":    session.Location.Country,
						"city":       session.Location.City,
						"session_id": session.SessionID,
					},
				}
			}
		}
	}

	// 3. 检查IP地址异常（代理、Tor等）
	ag.threatDetector.mu.RLock()
	if ipRisk, exists := ag.threatDetector.suspiciousIPs[session.IPAddress]; exists {
		ag.threatDetector.mu.RUnlock()

		if ipRisk.IsProxy || ipRisk.IsTor || ipRisk.IsVPN {
			return &ThreatEvent{
				ID:          fmt.Sprintf("suspicious_ip_%s_%d", session.UserID, time.Now().Unix()),
				Type:        "SUSPICIOUS_IP",
				Severity:    "HIGH",
				Description: fmt.Sprintf("User %s logged in from suspicious IP: %s", session.UserID, session.IPAddress),
				UserID:      session.UserID,
				IPAddress:   session.IPAddress,
				Timestamp:   time.Now(),
				Metadata: map[string]interface{}{
					"is_proxy":   ipRisk.IsProxy,
					"is_tor":     ipRisk.IsTor,
					"is_vpn":     ipRisk.IsVPN,
					"risk_score": ipRisk.RiskScore,
					"session_id": session.SessionID,
				},
			}
		}
	} else {
		ag.threatDetector.mu.RUnlock()
	}

	return nil
}

func (ag *AccountGuardian) checkAbnormalTrading(session *UserSession) *ThreatEvent {
	// 实现异常交易检测逻辑

	if len(session.Activities) == 0 {
		return nil
	}

	// 统计交易活动
	var tradingActivities []ActivityRecord
	var totalVolume float64
	var totalTrades int

	for _, activity := range session.Activities {
		if activity.Type == "TRADE" || activity.Type == "ORDER" {
			tradingActivities = append(tradingActivities, activity)
			if volume, ok := activity.Metadata["volume"].(float64); ok {
				totalVolume += volume
			}
			totalTrades++
		}
	}

	if len(tradingActivities) == 0 {
		return nil
	}

	// 1. 检查异常大额交易
	for _, activity := range tradingActivities {
		if volume, ok := activity.Metadata["volume"].(float64); ok {
			if volume > 100000 { // 超过10万的大额交易
				return &ThreatEvent{
					ID:          fmt.Sprintf("large_volume_trade_%s_%d", session.UserID, time.Now().Unix()),
					Type:        "LARGE_VOLUME_TRADE",
					Severity:    "HIGH",
					Description: fmt.Sprintf("User %s executed large volume trade: %.2f", session.UserID, volume),
					UserID:      session.UserID,
					IPAddress:   session.IPAddress,
					Timestamp:   time.Now(),
					Metadata: map[string]interface{}{
						"volume":     volume,
						"activity":   activity.Type,
						"session_id": session.SessionID,
					},
				}
			}
		}
	}

	// 2. 检查异常高频交易
	sessionDuration := time.Since(session.LoginTime)
	if sessionDuration > 0 {
		tradesPerHour := float64(totalTrades) / sessionDuration.Hours()
		if tradesPerHour > 100 { // 每小时超过100笔交易
			return &ThreatEvent{
				ID:          fmt.Sprintf("high_frequency_trading_%s_%d", session.UserID, time.Now().Unix()),
				Type:        "HIGH_FREQUENCY_TRADING",
				Severity:    "MEDIUM",
				Description: fmt.Sprintf("User %s executed high frequency trading: %.1f trades/hour", session.UserID, tradesPerHour),
				UserID:      session.UserID,
				IPAddress:   session.IPAddress,
				Timestamp:   time.Now(),
				Metadata: map[string]interface{}{
					"trades_per_hour":  tradesPerHour,
					"total_trades":     totalTrades,
					"session_duration": sessionDuration.String(),
					"session_id":       session.SessionID,
				},
			}
		}
	}

	// 3. 检查异常交易时间（非市场时间）
	now := time.Now()
	hour := now.Hour()
	weekday := now.Weekday()

	// 检查是否在非交易时间进行交易（假设交易时间为工作日9-17点）
	if weekday == time.Saturday || weekday == time.Sunday || hour < 9 || hour > 17 {
		if len(tradingActivities) > 0 {
			return &ThreatEvent{
				ID:          fmt.Sprintf("off_hours_trading_%s_%d", session.UserID, time.Now().Unix()),
				Type:        "OFF_HOURS_TRADING",
				Severity:    "MEDIUM",
				Description: fmt.Sprintf("User %s trading during off-hours: %s", session.UserID, now.Format("2006-01-02 15:04:05")),
				UserID:      session.UserID,
				IPAddress:   session.IPAddress,
				Timestamp:   time.Now(),
				Metadata: map[string]interface{}{
					"trading_time": now.Format("2006-01-02 15:04:05"),
					"weekday":      weekday.String(),
					"hour":         hour,
					"trades_count": len(tradingActivities),
					"session_id":   session.SessionID,
				},
			}
		}
	}

	return nil
}

func (ag *AccountGuardian) checkDeviceAnomaly(session *UserSession) *ThreatEvent {
	// 实现设备异常检测逻辑

	if session.DeviceInfo == nil {
		return nil
	}

	// 1. 检查不受信任的设备
	if !session.DeviceInfo.IsTrusted {
		return &ThreatEvent{
			ID:          fmt.Sprintf("untrusted_device_%s_%d", session.UserID, time.Now().Unix()),
			Type:        "UNTRUSTED_DEVICE",
			Severity:    "MEDIUM",
			Description: fmt.Sprintf("User %s logged in from untrusted device: %s", session.UserID, session.DeviceInfo.DeviceID),
			UserID:      session.UserID,
			IPAddress:   session.IPAddress,
			Timestamp:   time.Now(),
			Metadata: map[string]interface{}{
				"device_id":   session.DeviceInfo.DeviceID,
				"platform":    session.DeviceInfo.Platform,
				"browser":     session.DeviceInfo.Browser,
				"os":          session.DeviceInfo.OS,
				"fingerprint": session.DeviceInfo.Fingerprint,
				"session_id":  session.SessionID,
			},
		}
	}

	// 2. 检查可疑的设备指纹
	suspiciousFingerprints := []string{
		"automated_browser", "headless_chrome", "selenium", "phantomjs",
	}

	for _, suspicious := range suspiciousFingerprints {
		if session.DeviceInfo.Fingerprint == suspicious {
			return &ThreatEvent{
				ID:          fmt.Sprintf("suspicious_device_%s_%d", session.UserID, time.Now().Unix()),
				Type:        "SUSPICIOUS_DEVICE",
				Severity:    "HIGH",
				Description: fmt.Sprintf("User %s using suspicious device fingerprint: %s", session.UserID, suspicious),
				UserID:      session.UserID,
				IPAddress:   session.IPAddress,
				Timestamp:   time.Now(),
				Metadata: map[string]interface{}{
					"device_id":   session.DeviceInfo.DeviceID,
					"fingerprint": session.DeviceInfo.Fingerprint,
					"platform":    session.DeviceInfo.Platform,
					"browser":     session.DeviceInfo.Browser,
					"session_id":  session.SessionID,
				},
			}
		}
	}

	// 3. 检查异常的操作系统或浏览器组合
	if session.DeviceInfo.OS == "Linux" && session.DeviceInfo.Browser == "Unknown" {
		return &ThreatEvent{
			ID:       fmt.Sprintf("unusual_device_combo_%s_%d", session.UserID, time.Now().Unix()),
			Type:     "UNUSUAL_DEVICE_COMBINATION",
			Severity: "MEDIUM",
			Description: fmt.Sprintf("User %s using unusual OS/Browser combination: %s/%s",
				session.UserID, session.DeviceInfo.OS, session.DeviceInfo.Browser),
			UserID:    session.UserID,
			IPAddress: session.IPAddress,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"os":         session.DeviceInfo.OS,
				"browser":    session.DeviceInfo.Browser,
				"platform":   session.DeviceInfo.Platform,
				"device_id":  session.DeviceInfo.DeviceID,
				"session_id": session.SessionID,
			},
		}
	}

	return nil
}

func (ag *AccountGuardian) isIPWhitelisted(ipAddress string) bool {
	ag.threatDetector.mu.RLock()
	defer ag.threatDetector.mu.RUnlock()
	return ag.threatDetector.ipWhitelist[ipAddress]
}

func (ag *AccountGuardian) getGeoLocation(ipAddress string) *GeoLocation {
	// 实现IP地理位置查询
	ip := net.ParseIP(ipAddress)
	if ip.IsLoopback() || ip.IsPrivate() {
		return &GeoLocation{
			Country:   "LOCAL",
			City:      "Local",
			Latitude:  0.0,
			Longitude: 0.0,
			ISP:       "Local Network",
		}
	}

	// 简化的地理位置检测逻辑
	// 实际生产环境中应该调用 MaxMind GeoIP2 或其他地理位置API

	// 基于IP地址前缀的简单地理位置映射
	ipStr := ip.String()

	// 中国IP段示例
	if ipStr[:3] == "114" || ipStr[:3] == "202" || ipStr[:3] == "218" {
		return &GeoLocation{
			Country:   "CN",
			City:      "Beijing",
			Latitude:  39.9042,
			Longitude: 116.4074,
			ISP:       "China Telecom",
		}
	}

	// 俄罗斯IP段示例
	if ipStr[:3] == "195" || ipStr[:3] == "178" {
		return &GeoLocation{
			Country:   "RU",
			City:      "Moscow",
			Latitude:  55.7558,
			Longitude: 37.6176,
			ISP:       "Rostelecom",
		}
	}

	// 美国IP段示例
	if ipStr[:2] == "8." || ipStr[:3] == "173" || ipStr[:3] == "104" {
		return &GeoLocation{
			Country:   "US",
			City:      "New York",
			Latitude:  40.7128,
			Longitude: -74.0060,
			ISP:       "Verizon",
		}
	}

	// 欧盟IP段示例
	if ipStr[:3] == "185" || ipStr[:3] == "188" {
		return &GeoLocation{
			Country:   "DE",
			City:      "Frankfurt",
			Latitude:  50.1109,
			Longitude: 8.6821,
			ISP:       "Deutsche Telekom",
		}
	}

	// 默认位置（未知）
	return &GeoLocation{
		Country:   "UNKNOWN",
		City:      "Unknown",
		Latitude:  0.0,
		Longitude: 0.0,
		ISP:       "Unknown ISP",
	}
}

func (ag *AccountGuardian) getDeviceInfo(userAgent string) *DeviceInfo {
	// 实现设备信息解析
	hash := sha256.Sum256([]byte(userAgent))
	fingerprint := fmt.Sprintf("%x", hash)

	// 简化的User-Agent解析逻辑
	// 实际生产环境中应该使用专业的User-Agent解析库

	var platform, browser, os string
	isTrusted := false

	// 检测操作系统
	if contains(userAgent, "Windows") {
		os = "Windows"
		platform = "Desktop"
	} else if contains(userAgent, "Macintosh") || contains(userAgent, "Mac OS") {
		os = "macOS"
		platform = "Desktop"
	} else if contains(userAgent, "Linux") {
		os = "Linux"
		platform = "Desktop"
	} else if contains(userAgent, "Android") {
		os = "Android"
		platform = "Mobile"
	} else if contains(userAgent, "iPhone") || contains(userAgent, "iPad") {
		os = "iOS"
		platform = "Mobile"
	} else {
		os = "Unknown"
		platform = "Unknown"
	}

	// 检测浏览器
	if contains(userAgent, "Chrome") && !contains(userAgent, "Edg") {
		browser = "Chrome"
		isTrusted = true
	} else if contains(userAgent, "Firefox") {
		browser = "Firefox"
		isTrusted = true
	} else if contains(userAgent, "Safari") && !contains(userAgent, "Chrome") {
		browser = "Safari"
		isTrusted = true
	} else if contains(userAgent, "Edg") {
		browser = "Edge"
		isTrusted = true
	} else if contains(userAgent, "Opera") {
		browser = "Opera"
		isTrusted = true
	} else if contains(userAgent, "curl") || contains(userAgent, "wget") {
		browser = "Command Line Tool"
		isTrusted = false
	} else if contains(userAgent, "bot") || contains(userAgent, "crawler") {
		browser = "Bot/Crawler"
		isTrusted = false
	} else {
		browser = "Unknown"
		isTrusted = false
	}

	// 检测可疑特征
	if contains(userAgent, "HeadlessChrome") || contains(userAgent, "PhantomJS") ||
		contains(userAgent, "Selenium") || contains(userAgent, "automated") {
		fingerprint = "automated_browser"
		isTrusted = false
	}

	return &DeviceInfo{
		DeviceID:    fingerprint[:16],
		Platform:    platform,
		Browser:     browser,
		OS:          os,
		Fingerprint: fingerprint,
		IsTrusted:   isTrusted,
	}
}

// contains 检查字符串是否包含子字符串（不区分大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsMiddle(s, substr))))
}

// containsMiddle 检查字符串中间是否包含子字符串
func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (ag *AccountGuardian) calculateActivityRisk(activity ActivityRecord, session *UserSession) float64 {
	riskScore := 0.0

	// 基于活动类型计算风险
	switch activity.Type {
	case "LOGIN":
		riskScore += 0.1
	case "TRADE":
		riskScore += 0.3
	case "WITHDRAW":
		riskScore += 0.5
	case "CONFIG_CHANGE":
		riskScore += 0.4
	}

	// 基于IP地址计算风险
	if !session.IsWhitelisted {
		riskScore += 0.2
	}

	// 基于时间计算风险（非正常时间）
	hour := activity.Timestamp.Hour()
	if hour < 6 || hour > 22 {
		riskScore += 0.1
	}

	return math.Min(riskScore, 1.0)
}

func (ag *AccountGuardian) calculateSessionRisk(session *UserSession) float64 {
	if len(session.Activities) == 0 {
		return 0.0
	}

	totalRisk := 0.0
	for _, activity := range session.Activities {
		totalRisk += activity.RiskScore
	}

	avgRisk := totalRisk / float64(len(session.Activities))

	// 考虑会话持续时间
	duration := time.Since(session.LoginTime)
	if duration > 12*time.Hour {
		avgRisk += 0.1
	}

	return math.Min(avgRisk, 1.0)
}

func (ag *AccountGuardian) triggerAnomalyDetection(session *UserSession, activity ActivityRecord) {
	anomaly := AnomalyEvent{
		ID:          ag.generateEventID(),
		Type:        "HIGH_RISK_ACTIVITY",
		Score:       session.RiskScore,
		Description: fmt.Sprintf("High risk activity detected for user %s", session.UserID),
		UserID:      session.UserID,
		Timestamp:   time.Now(),
		Metadata: map[string]interface{}{
			"session_id":    session.SessionID,
			"ip_address":    session.IPAddress,
			"activity_type": activity.Type,
			"risk_score":    session.RiskScore,
		},
	}

	ag.handleAnomalyEvent(anomaly)
}

func (ag *AccountGuardian) detectBehaviorAnomalies(session *UserSession, baseline *UserBaseline) []AnomalyEvent {
	anomalies := make([]AnomalyEvent, 0)

	// 实现具体的行为异常检测逻辑

	// 1. 检测登录时间异常
	if baseline.LoginFrequency != nil {
		currentHour := float64(session.LoginTime.Hour())
		if baseline.LoginFrequency.Count > 0 {
			zScore := (currentHour - baseline.LoginFrequency.Mean) / baseline.LoginFrequency.StdDev
			if math.Abs(zScore) > ag.anomalyThreshold {
				anomalies = append(anomalies, AnomalyEvent{
					ID:          fmt.Sprintf("login_time_anomaly_%s_%d", session.UserID, time.Now().Unix()),
					Type:        "LOGIN_TIME_ANOMALY",
					Score:       math.Abs(zScore),
					Description: fmt.Sprintf("Unusual login time for user %s: %d:00", session.UserID, int(currentHour)),
					UserID:      session.UserID,
					Timestamp:   time.Now(),
					Baseline: &BaselineData{
						Mean:         baseline.LoginFrequency.Mean,
						StdDev:       baseline.LoginFrequency.StdDev,
						SampleCount:  int(baseline.LoginFrequency.Count),
						CalculatedAt: baseline.UpdatedAt,
					},
					Current: &CurrentData{
						Value:      currentHour,
						ZScore:     zScore,
						Percentile: 0.0, // 简化实现
						ObservedAt: time.Now(),
					},
				})
			}
		}
	}

	// 2. 检测地理位置异常
	if session.Location != nil && len(baseline.Locations) > 0 {
		locationKey := session.Location.Country + ":" + session.Location.City
		if _, exists := baseline.Locations[locationKey]; !exists {
			anomalies = append(anomalies, AnomalyEvent{
				ID:          fmt.Sprintf("location_anomaly_%s_%d", session.UserID, time.Now().Unix()),
				Type:        "LOCATION_ANOMALY",
				Score:       3.0, // 新位置给予固定高分
				Description: fmt.Sprintf("New location for user %s: %s, %s", session.UserID, session.Location.City, session.Location.Country),
				UserID:      session.UserID,
				Timestamp:   time.Now(),
				Metadata: map[string]interface{}{
					"new_location":    locationKey,
					"known_locations": len(baseline.Locations),
				},
			})
		}
	}

	// 3. 检测交易模式异常
	tradingCount := 0
	for _, activity := range session.Activities {
		if activity.Type == "TRADE" || activity.Type == "ORDER" {
			tradingCount++
		}
	}

	if baseline.TradingFrequency != nil && baseline.TradingFrequency.Count > 0 {
		sessionDuration := time.Since(session.LoginTime).Hours()
		if sessionDuration > 0 {
			currentTradingRate := float64(tradingCount) / sessionDuration
			zScore := (currentTradingRate - baseline.TradingFrequency.Mean) / baseline.TradingFrequency.StdDev
			if math.Abs(zScore) > ag.anomalyThreshold {
				anomalies = append(anomalies, AnomalyEvent{
					ID:          fmt.Sprintf("trading_pattern_anomaly_%s_%d", session.UserID, time.Now().Unix()),
					Type:        "TRADING_PATTERN_ANOMALY",
					Score:       math.Abs(zScore),
					Description: fmt.Sprintf("Unusual trading frequency for user %s: %.2f trades/hour", session.UserID, currentTradingRate),
					UserID:      session.UserID,
					Timestamp:   time.Now(),
					Baseline: &BaselineData{
						Mean:         baseline.TradingFrequency.Mean,
						StdDev:       baseline.TradingFrequency.StdDev,
						SampleCount:  int(baseline.TradingFrequency.Count),
						CalculatedAt: baseline.UpdatedAt,
					},
					Current: &CurrentData{
						Value:      currentTradingRate,
						ZScore:     zScore,
						Percentile: 0.0,
						ObservedAt: time.Now(),
					},
				})
			}
		}
	}

	return anomalies
}

func (ag *AccountGuardian) handleAnomalyEvent(anomaly AnomalyEvent) {
	ag.mu.Lock()
	ag.anomalyEvents = append(ag.anomalyEvents, anomaly)
	ag.mu.Unlock()

	log.Printf("Anomaly detected: %s (score: %.2f)", anomaly.Description, anomaly.Score)

	// 根据异常分数决定响应级别
	if anomaly.Score > ag.anomalyThreshold {
		ag.escalateAnomaly(anomaly)
	}
}

func (ag *AccountGuardian) handleThreatEvent(threat ThreatEvent) {
	ag.mu.Lock()
	ag.threatEvents = append(ag.threatEvents, threat)
	ag.mu.Unlock()

	log.Printf("Threat detected: %s (%s severity)", threat.Description, threat.Severity)

	// 自动响应
	if ag.autoFreezeEnabled && threat.Severity == "HIGH" {
		ag.autoFreezeAccount(threat.UserID, threat.ID)
	}
}

func (ag *AccountGuardian) escalateAnomaly(anomaly AnomalyEvent) {
	log.Printf("Escalating anomaly: %s", anomaly.ID)

	// 实现异常升级逻辑

	// 1. 发送告警通知
	log.Printf("发送异常告警: 用户=%s, 类型=%s, 分数=%.2f", anomaly.UserID, anomaly.Type, anomaly.Score)

	// 2. 触发安全响应
	if anomaly.Score >= ag.alertThreshold {
		if ag.autoFreezeEnabled && anomaly.Score >= 4.0 {
			// 高风险异常自动冻结账户
			ag.autoFreezeAccount(anomaly.UserID, anomaly.ID)
		}

		// 更新安全指标
		ag.securityMetrics.mu.Lock()
		ag.securityMetrics.AnomaliesDetected++
		ag.securityMetrics.LastUpdated = time.Now()
		ag.securityMetrics.mu.Unlock()
	}

	// 3. 记录升级事件
	log.Printf("异常升级完成: %s", anomaly.ID)
}

func (ag *AccountGuardian) autoFreezeAccount(userID, eventID string) {
	log.Printf("Auto-freezing account: %s due to event: %s", userID, eventID)

	// 实现账户冻结逻辑

	// 1. 冻结账户
	log.Printf("执行账户冻结: 用户=%s, 事件=%s", userID, eventID)
	// 实际实现中这里会调用账户管理服务的冻结API

	// 2. 发送通知
	log.Printf("发送冻结通知: 用户=%s", userID)
	// 实际实现中这里会发送邮件、短信等通知

	// 3. 记录冻结事件
	freezeEvent := ThreatEvent{
		ID:          fmt.Sprintf("account_freeze_%s_%d", userID, time.Now().Unix()),
		Type:        "ACCOUNT_FREEZE",
		Severity:    "CRITICAL",
		Description: fmt.Sprintf("Account %s automatically frozen due to security event %s", userID, eventID),
		UserID:      userID,
		Timestamp:   time.Now(),
		Metadata: map[string]interface{}{
			"trigger_event": eventID,
			"freeze_reason": "automatic_security_response",
		},
	}

	ag.mu.Lock()
	ag.threatEvents = append(ag.threatEvents, freezeEvent)
	ag.mu.Unlock()

	ag.securityMetrics.mu.Lock()
	ag.securityMetrics.AccountsFrozen++
	ag.securityMetrics.AutomatedResponses++
	ag.securityMetrics.mu.Unlock()
}

func (ag *AccountGuardian) handlePendingResponses() {
	// TODO: 实现待处理响应的处理逻辑
}

func (ag *AccountGuardian) updateSecurityMetrics() {
	ag.securityMetrics.mu.Lock()
	defer ag.securityMetrics.mu.Unlock()

	// 计算检测准确率
	if ag.securityMetrics.TruePositives+ag.securityMetrics.FalsePositives > 0 {
		ag.securityMetrics.PrecisionRate = float64(ag.securityMetrics.TruePositives) /
			float64(ag.securityMetrics.TruePositives+ag.securityMetrics.FalsePositives)
	}

	// 更新最后更新时间
	ag.securityMetrics.LastUpdated = time.Now()
}

func (ag *AccountGuardian) generateEventID() string {
	return fmt.Sprintf("EVT_%d_%d", time.Now().Unix(), time.Now().Nanosecond())
}

// getUserBaseline 获取用户基线
func (ba *BehaviorAnalyzer) getUserBaseline(userID string) *UserBaseline {
	ba.mu.RLock()
	defer ba.mu.RUnlock()
	return ba.userBaselines[userID]
}

// createUserBaseline 创建用户基线
func (ba *BehaviorAnalyzer) createUserBaseline(userID string) *UserBaseline {
	baseline := &UserBaseline{
		UserID:           userID,
		LoginFrequency:   &StatisticalData{},
		SessionDuration:  &StatisticalData{},
		ActiveHours:      make(map[int]float64),
		IPAddresses:      make(map[string]int),
		Locations:        make(map[string]int),
		TradingVolume:    &StatisticalData{},
		TradingFrequency: &StatisticalData{},
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	ba.mu.Lock()
	ba.userBaselines[userID] = baseline
	ba.mu.Unlock()

	return baseline
}

// updateUserBaseline 更新用户基线
func (ba *BehaviorAnalyzer) updateUserBaseline(session *UserSession, baseline *UserBaseline) {
	// 实现基线更新逻辑

	// 更新登录频率基线
	if baseline.LoginFrequency == nil {
		baseline.LoginFrequency = &StatisticalData{}
	}

	loginHour := float64(session.LoginTime.Hour())
	baseline.LoginFrequency.Count++

	// 使用增量更新计算均值和标准差
	if baseline.LoginFrequency.Count == 1 {
		baseline.LoginFrequency.Mean = loginHour
		baseline.LoginFrequency.StdDev = 0.0
		baseline.LoginFrequency.Min = loginHour
		baseline.LoginFrequency.Max = loginHour
	} else {
		// 更新均值
		oldMean := baseline.LoginFrequency.Mean
		baseline.LoginFrequency.Mean = oldMean + (loginHour-oldMean)/float64(baseline.LoginFrequency.Count)

		// 更新标准差（使用 Welford 算法）
		baseline.LoginFrequency.StdDev = math.Sqrt(
			(baseline.LoginFrequency.StdDev*baseline.LoginFrequency.StdDev*float64(baseline.LoginFrequency.Count-1) +
				(loginHour-oldMean)*(loginHour-baseline.LoginFrequency.Mean)) / float64(baseline.LoginFrequency.Count))

		// 更新最小值和最大值
		if loginHour < baseline.LoginFrequency.Min {
			baseline.LoginFrequency.Min = loginHour
		}
		if loginHour > baseline.LoginFrequency.Max {
			baseline.LoginFrequency.Max = loginHour
		}
	}

	// 更新地理位置基线
	if session.Location != nil {
		if baseline.Locations == nil {
			baseline.Locations = make(map[string]int)
		}
		locationKey := session.Location.Country + ":" + session.Location.City
		baseline.Locations[locationKey]++
	}

	// 更新IP地址基线
	if baseline.IPAddresses == nil {
		baseline.IPAddresses = make(map[string]int)
	}
	baseline.IPAddresses[session.IPAddress]++

	// 更新交易频率基线
	tradingCount := 0
	for _, activity := range session.Activities {
		if activity.Type == "TRADE" || activity.Type == "ORDER" {
			tradingCount++
		}
	}

	if tradingCount > 0 {
		if baseline.TradingFrequency == nil {
			baseline.TradingFrequency = &StatisticalData{}
		}

		sessionDuration := time.Since(session.LoginTime).Hours()
		if sessionDuration > 0 {
			tradingRate := float64(tradingCount) / sessionDuration
			baseline.TradingFrequency.Count++

			if baseline.TradingFrequency.Count == 1 {
				baseline.TradingFrequency.Mean = tradingRate
				baseline.TradingFrequency.StdDev = 0.0
				baseline.TradingFrequency.Min = tradingRate
				baseline.TradingFrequency.Max = tradingRate
			} else {
				oldMean := baseline.TradingFrequency.Mean
				baseline.TradingFrequency.Mean = oldMean + (tradingRate-oldMean)/float64(baseline.TradingFrequency.Count)

				baseline.TradingFrequency.StdDev = math.Sqrt(
					(baseline.TradingFrequency.StdDev*baseline.TradingFrequency.StdDev*float64(baseline.TradingFrequency.Count-1) +
						(tradingRate-oldMean)*(tradingRate-baseline.TradingFrequency.Mean)) / float64(baseline.TradingFrequency.Count))

				if tradingRate < baseline.TradingFrequency.Min {
					baseline.TradingFrequency.Min = tradingRate
				}
				if tradingRate > baseline.TradingFrequency.Max {
					baseline.TradingFrequency.Max = tradingRate
				}
			}
		}
	}

	baseline.UpdatedAt = time.Now()
}

// GetSecurityMetrics 获取安全指标
func (ag *AccountGuardian) GetSecurityMetrics() *SecurityMetrics {
	ag.securityMetrics.mu.RLock()
	defer ag.securityMetrics.mu.RUnlock()

	// 创建新的 SecurityMetrics 实例，避免复制锁
	metrics := &SecurityMetrics{
		ThreatsDetected:     ag.securityMetrics.ThreatsDetected,
		AnomaliesDetected:   ag.securityMetrics.AnomaliesDetected,
		FalsePositives:      ag.securityMetrics.FalsePositives,
		TruePositives:       ag.securityMetrics.TruePositives,
		AutomatedResponses:  ag.securityMetrics.AutomatedResponses,
		ManualInterventions: ag.securityMetrics.ManualInterventions,
		AccountsFrozen:      ag.securityMetrics.AccountsFrozen,
		DetectionLatency:    ag.securityMetrics.DetectionLatency,
		ResponseLatency:     ag.securityMetrics.ResponseLatency,
		SystemUptime:        ag.securityMetrics.SystemUptime,
		DetectionAccuracy:   ag.securityMetrics.DetectionAccuracy,
		PrecisionRate:       ag.securityMetrics.PrecisionRate,
		RecallRate:          ag.securityMetrics.RecallRate,
		LastUpdated:         ag.securityMetrics.LastUpdated,
	}
	return metrics
}

// GetStatus 获取守护者状态
func (ag *AccountGuardian) GetStatus() map[string]interface{} {
	ag.mu.RLock()
	defer ag.mu.RUnlock()

	return map[string]interface{}{
		"running":          ag.isRunning,
		"active_sessions":  len(ag.userSessions),
		"threat_events":    len(ag.threatEvents),
		"anomaly_events":   len(ag.anomalyEvents),
		"security_metrics": ag.GetSecurityMetrics(),
	}
}
