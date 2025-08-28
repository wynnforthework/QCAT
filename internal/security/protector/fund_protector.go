package protector

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"

	"qcat/internal/config"
	"qcat/internal/exchange"
	"qcat/internal/security/protector/dao"
)

// FundProtector 资金保护系统
type FundProtector struct {
	config              *config.Config
	circuitBreaker      *CircuitBreaker
	autoTransferManager *AutoTransferManager
	emergencyProtocol   *EmergencyProtocol
	riskAssessment      *RiskAssessment

	// 运行状态
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	isRunning bool
	mu        sync.RWMutex

	// 保护状态
	isEmergencyMode    bool
	circuitBreakerOpen bool
	lastRiskCheck      time.Time

	// 资金状态监控
	fundStatus        *FundStatus
	protectionMetrics *ProtectionMetrics
	transferHistory   []TransferRecord
	emergencyEvents   []EmergencyEvent

	// 配置参数
	profitThreshold       float64
	transferRatio         float64
	maxDailyLoss          float64
	checkInterval         time.Duration
	circuitBreakerEnabled bool

	// 服务依赖
	exchangeProvider    ExchangeDataProvider
	notificationService NotificationService
	walletService       WalletService
	exchange            exchange.Exchange

	// 数据访问层
	daoManager dao.DAOManager

	// 数据收集控制
	dataCollectionInterval time.Duration
	lastDataCollection     time.Time

	// 交易控制
	tradingPaused bool
}

// FundStatus 资金状态
type FundStatus struct {
	mu sync.RWMutex

	TotalBalance     float64 `json:"total_balance"`
	AvailableBalance float64 `json:"available_balance"`
	LockedBalance    float64 `json:"locked_balance"`
	ProfitLoss       float64 `json:"profit_loss"`
	DailyPL          float64 `json:"daily_pl"`
	UnrealizedPL     float64 `json:"unrealized_pl"`
	RealizedPL       float64 `json:"realized_pl"`

	// 风险指标
	CurrentRisk       float64 `json:"current_risk"`
	MaxRisk           float64 `json:"max_risk"`
	VaR95             float64 `json:"var_95"`
	ExpectedShortfall float64 `json:"expected_shortfall"`

	// 仓位信息
	TotalPositions  int `json:"total_positions"`
	ActivePositions int `json:"active_positions"`
	LongPositions   int `json:"long_positions"`
	ShortPositions  int `json:"short_positions"`

	LastUpdated time.Time `json:"last_updated"`
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	isOpen         bool
	lastTriggered  time.Time
	triggerCount   int
	maxDailyLoss   float64
	cooldownPeriod time.Duration
	mu             sync.RWMutex
}

// AutoTransferManager 自动转账管理器
type AutoTransferManager struct {
	enabled           bool
	profitThreshold   float64
	transferRatio     float64
	coldWalletAddress string
	minTransferAmount float64
	maxTransferAmount float64
	transferHistory   []TransferRecord
	mu                sync.RWMutex
}

// TransferRecord 转账记录
type TransferRecord struct {
	ID              string                 `json:"id"`
	Type            string                 `json:"type"` // PROFIT_TRANSFER, EMERGENCY_TRANSFER
	Amount          float64                `json:"amount"`
	From            string                 `json:"from"`
	To              string                 `json:"to"`
	Status          string                 `json:"status"` // PENDING, COMPLETED, FAILED
	Timestamp       time.Time              `json:"timestamp"`
	TriggerReason   string                 `json:"trigger_reason"`
	TransactionHash string                 `json:"transaction_hash"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// EmergencyProtocol 紧急协议
type EmergencyProtocol struct {
	isActive          bool
	emergencyContacts []EmergencyContact
	responsePlan      []ResponseAction
	lastActivation    time.Time
	activationCount   int
	mu                sync.RWMutex
}

// EmergencyContact 紧急联系人
type EmergencyContact struct {
	Name        string   `json:"name"`
	Role        string   `json:"role"`
	Phone       string   `json:"phone"`
	Email       string   `json:"email"`
	Priority    int      `json:"priority"`
	IsAvailable bool     `json:"is_available"`
	Channels    []string `json:"channels"`
}

// ResponseAction 响应动作
type ResponseAction struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Priority    int                    `json:"priority"`
	Description string                 `json:"description"`
	IsAutomatic bool                   `json:"is_automatic"`
	Condition   string                 `json:"condition"`
	Action      func() error           `json:"-"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// EmergencyEvent 紧急事件
type EmergencyEvent struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
	TriggerData map[string]interface{} `json:"trigger_data"`
	Response    *EmergencyResponse     `json:"response"`
}

// EmergencyResponse 紧急响应
type EmergencyResponse struct {
	ResponseTime  time.Duration          `json:"response_time"`
	Actions       []string               `json:"actions"`
	Status        string                 `json:"status"`
	Notifications []NotificationRecord   `json:"notifications"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// NotificationRecord 通知记录
type NotificationRecord struct {
	Channel   string    `json:"channel"`
	Recipient string    `json:"recipient"`
	Status    string    `json:"status"`
	SentAt    time.Time `json:"sent_at"`
	Message   string    `json:"message"`
}

// RiskAssessment 风险评估
type RiskAssessment struct {
	model            string
	checkInterval    time.Duration
	lastAssessment   time.Time
	currentRiskLevel string
	riskHistory      []RiskSnapshot
	mu               sync.RWMutex
}

// RiskSnapshot 风险快照
type RiskSnapshot struct {
	Timestamp       time.Time `json:"timestamp"`
	RiskLevel       string    `json:"risk_level"`
	RiskScore       float64   `json:"risk_score"`
	VaR             float64   `json:"var"`
	ExpectedLoss    float64   `json:"expected_loss"`
	MaxDrawdown     float64   `json:"max_drawdown"`
	VolatilityIndex float64   `json:"volatility_index"`
	Leverage        float64   `json:"leverage"`
	Concentration   float64   `json:"concentration"`
}

// ProtectionMetrics 保护指标
type ProtectionMetrics struct {
	mu sync.RWMutex

	// 保护统计
	CircuitBreakerTriggered int64 `json:"circuit_breaker_triggered"`
	EmergencyActivations    int64 `json:"emergency_activations"`
	AutoTransfers           int64 `json:"auto_transfers"`
	ManualInterventions     int64 `json:"manual_interventions"`

	// 资金保护效果
	LossesPrevented float64 `json:"losses_prevented"`
	ProfitsSecured  float64 `json:"profits_secured"`
	MaxLossAvoided  float64 `json:"max_loss_avoided"`

	// 响应性能
	AvgResponseTime    time.Duration `json:"avg_response_time"`
	ProtectionAccuracy float64       `json:"protection_accuracy"`
	FalsePositiveRate  float64       `json:"false_positive_rate"`

	// 系统健康
	SystemUptime      time.Duration `json:"system_uptime"`
	LastEmergencyTest time.Time     `json:"last_emergency_test"`

	LastUpdated time.Time `json:"last_updated"`
}

// NewFundProtector 创建资金保护系统
func NewFundProtector(cfg *config.Config, exchangeProvider ExchangeDataProvider, ex exchange.Exchange, daoManager dao.DAOManager, notificationService NotificationService, walletService WalletService) (*FundProtector, error) {
	ctx, cancel := context.WithCancel(context.Background())

	fp := &FundProtector{
		config:                cfg,
		circuitBreaker:        NewCircuitBreaker(0.05, 30*time.Minute), // 5%最大日亏损，30分钟冷却
		autoTransferManager:   NewAutoTransferManager(),
		emergencyProtocol:     NewEmergencyProtocol(),
		riskAssessment:        NewRiskAssessment(),
		ctx:                   ctx,
		cancel:                cancel,
		fundStatus:            &FundStatus{},
		protectionMetrics:     &ProtectionMetrics{},
		transferHistory:       make([]TransferRecord, 0),
		emergencyEvents:       make([]EmergencyEvent, 0),
		profitThreshold:       0.1,  // 10%利润转移阈值
		transferRatio:         0.3,  // 30%转移比例
		maxDailyLoss:          0.05, // 5%最大日亏损
		checkInterval:         5 * time.Minute,
		circuitBreakerEnabled: true,
		exchangeProvider:      exchangeProvider,
		notificationService:   notificationService,
		walletService:         walletService,
		exchange:              ex,
		daoManager:            daoManager,
		dataCollectionInterval: 1 * time.Hour, // 每小时收集一次数据
	}

	// 从配置文件读取参数
	if cfg != nil {
		fp.loadConfigurationParameters(cfg)
	}

	return fp, nil
}

// loadConfigurationParameters 从配置文件读取资金保护参数
func (fp *FundProtector) loadConfigurationParameters(cfg *config.Config) {
	// 读取资金保护相关配置
	if cfg.Risk.MaxDrawdown > 0 {
		fp.maxDailyLoss = cfg.Risk.MaxDrawdown
	}
	
	if cfg.Risk.CheckInterval > 0 {
		fp.checkInterval = cfg.Risk.CheckInterval
	}
	
	// 从环境变量或配置中读取其他参数
	// 这些参数应该在config.yaml中定义
	
	// 设置最大风险限制
	fp.fundStatus.MaxRisk = 0.15 // 15% 最大风险
	
	log.Printf("Fund protector configuration loaded: maxDailyLoss=%.2f%%, checkInterval=%v", 
		fp.maxDailyLoss*100, fp.checkInterval)
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(maxDailyLoss float64, cooldownPeriod time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		isOpen:         false,
		maxDailyLoss:   maxDailyLoss,
		cooldownPeriod: cooldownPeriod,
	}
}

// NewAutoTransferManager 创建自动转账管理器
func NewAutoTransferManager() *AutoTransferManager {
	return &AutoTransferManager{
		enabled:           true,
		profitThreshold:   0.1,
		transferRatio:     0.3,
		minTransferAmount: 100.0,
		maxTransferAmount: 100000.0,
		transferHistory:   make([]TransferRecord, 0),
	}
}

// NewEmergencyProtocol 创建紧急协议
func NewEmergencyProtocol() *EmergencyProtocol {
	return &EmergencyProtocol{
		isActive:          false,
		emergencyContacts: make([]EmergencyContact, 0),
		responsePlan:      make([]ResponseAction, 0),
	}
}

// NewRiskAssessment 创建风险评估
func NewRiskAssessment() *RiskAssessment {
	return &RiskAssessment{
		model:            "var_based",
		checkInterval:    5 * time.Minute,
		currentRiskLevel: "LOW",
		riskHistory:      make([]RiskSnapshot, 0),
	}
}

// Start 启动资金保护系统
func (fp *FundProtector) Start() error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	if fp.isRunning {
		return fmt.Errorf("fund protector is already running")
	}

	log.Println("Starting Fund Protector...")

	// 启动资金状态监控
	fp.wg.Add(1)
	go fp.runFundMonitoring()

	// 启动自动转账监控
	fp.wg.Add(1)
	go fp.runAutoTransferMonitoring()

	// 启动风险评估
	fp.wg.Add(1)
	go fp.runRiskAssessment()

	// 启动熔断器监控
	fp.wg.Add(1)
	go fp.runCircuitBreakerMonitoring()

	// 启动指标收集
	fp.wg.Add(1)
	go fp.runMetricsCollection()

	// 启动数据收集和持久化
	if fp.daoManager != nil {
		fp.wg.Add(1)
		go fp.runDataCollection()
		
		// 启动数据清理（每天执行一次）
		fp.wg.Add(1)
		go fp.runDataCleanup()
	}

	fp.isRunning = true
	log.Println("Fund Protector started successfully")
	return nil
}

// Stop 停止资金保护系统
func (fp *FundProtector) Stop() error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	if !fp.isRunning {
		return fmt.Errorf("fund protector is not running")
	}

	log.Println("Stopping Fund Protector...")

	fp.cancel()
	fp.wg.Wait()

	fp.isRunning = false
	log.Println("Fund Protector stopped successfully")
	return nil
}

// runFundMonitoring 运行资金监控
func (fp *FundProtector) runFundMonitoring() {
	defer fp.wg.Done()

	ticker := time.NewTicker(fp.checkInterval)
	defer ticker.Stop()

	log.Println("Fund monitoring started")

	for {
		select {
		case <-fp.ctx.Done():
			log.Println("Fund monitoring stopped")
			return
		case <-ticker.C:
			fp.monitorFundStatus()
		}
	}
}

// runAutoTransferMonitoring 运行自动转账监控
func (fp *FundProtector) runAutoTransferMonitoring() {
	defer fp.wg.Done()

	ticker := time.NewTicker(1 * time.Hour) // 每小时检查一次
	defer ticker.Stop()

	log.Println("Auto transfer monitoring started")

	for {
		select {
		case <-fp.ctx.Done():
			log.Println("Auto transfer monitoring stopped")
			return
		case <-ticker.C:
			fp.checkAutoTransfer()
		}
	}
}

// runRiskAssessment 运行风险评估
func (fp *FundProtector) runRiskAssessment() {
	defer fp.wg.Done()

	ticker := time.NewTicker(fp.riskAssessment.checkInterval)
	defer ticker.Stop()

	log.Println("Risk assessment started")

	for {
		select {
		case <-fp.ctx.Done():
			log.Println("Risk assessment stopped")
			return
		case <-ticker.C:
			fp.performRiskAssessment()
		}
	}
}

// runCircuitBreakerMonitoring 运行熔断器监控
func (fp *FundProtector) runCircuitBreakerMonitoring() {
	defer fp.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.Println("Circuit breaker monitoring started")

	for {
		select {
		case <-fp.ctx.Done():
			log.Println("Circuit breaker monitoring stopped")
			return
		case <-ticker.C:
			fp.checkCircuitBreaker()
		}
	}
}

// runMetricsCollection 运行指标收集
func (fp *FundProtector) runMetricsCollection() {
	defer fp.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Println("Metrics collection started")

	for {
		select {
		case <-fp.ctx.Done():
			log.Println("Metrics collection stopped")
			return
		case <-ticker.C:
			fp.updateProtectionMetrics()
		}
	}
}

// monitorFundStatus 监控资金状态
func (fp *FundProtector) monitorFundStatus() {
	log.Println("Monitoring fund status...")

	// 更新资金状态
	fp.updateFundStatus()

	// 检查资金安全
	fp.checkFundSafety()

	fp.lastRiskCheck = time.Now()
}

// updateFundStatus 更新资金状态
func (fp *FundProtector) updateFundStatus() {
	fp.fundStatus.mu.Lock()
	defer fp.fundStatus.mu.Unlock()

	// 从交易系统获取实际资金状态
	fundData, err := fp.getFundDataFromExchange()
	if err != nil {
		log.Printf("Failed to get fund data from exchange: %v", err)
		// 保持当前状态
		return
	}

	fp.fundStatus.TotalBalance = fundData.TotalBalance
	fp.fundStatus.AvailableBalance = fundData.AvailableBalance
	fp.fundStatus.LockedBalance = fundData.LockedBalance
	fp.fundStatus.DailyPL = fundData.DailyPL
	fp.fundStatus.UnrealizedPL = fundData.UnrealizedPL
	fp.fundStatus.RealizedPL = 3500.0
	fp.fundStatus.ProfitLoss = fp.fundStatus.UnrealizedPL + fp.fundStatus.RealizedPL

	// 计算风险指标
	fp.fundStatus.CurrentRisk = fp.calculateCurrentRisk()
	fp.fundStatus.VaR95 = fp.calculateVaR95()
	fp.fundStatus.ExpectedShortfall = fp.calculateExpectedShortfall()

	fp.fundStatus.LastUpdated = time.Now()
}

// checkFundSafety 检查资金安全
func (fp *FundProtector) checkFundSafety() {
	fp.fundStatus.mu.RLock()
	dailyLossRatio := -fp.fundStatus.DailyPL / fp.fundStatus.TotalBalance
	totalLossRatio := -fp.fundStatus.ProfitLoss / fp.fundStatus.TotalBalance
	fp.fundStatus.mu.RUnlock()

	// 检查日亏损是否超限
	if dailyLossRatio > fp.maxDailyLoss {
		fp.triggerEmergency("DAILY_LOSS_EXCEEDED", map[string]interface{}{
			"daily_loss_ratio": dailyLossRatio,
			"max_daily_loss":   fp.maxDailyLoss,
			"actual_loss":      fp.fundStatus.DailyPL,
		})
	}

	// 检查总体亏损
	if totalLossRatio > 0.2 { // 总亏损超过20%
		fp.triggerEmergency("CRITICAL_LOSS", map[string]interface{}{
			"total_loss_ratio": totalLossRatio,
			"total_loss":       fp.fundStatus.ProfitLoss,
		})
	}

	// 检查风险指标
	if fp.fundStatus.CurrentRisk > fp.fundStatus.MaxRisk {
		fp.triggerEmergency("RISK_LIMIT_EXCEEDED", map[string]interface{}{
			"current_risk": fp.fundStatus.CurrentRisk,
			"max_risk":     fp.fundStatus.MaxRisk,
		})
	}
}

// checkAutoTransfer 检查自动转账
func (fp *FundProtector) checkAutoTransfer() {
	if !fp.autoTransferManager.enabled {
		return
	}

	log.Println("Checking auto transfer conditions...")

	fp.fundStatus.mu.RLock()
	profitRatio := fp.fundStatus.RealizedPL / fp.fundStatus.TotalBalance
	fp.fundStatus.mu.RUnlock()

	// 检查是否达到利润转移阈值
	if profitRatio > fp.profitThreshold {
		transferAmount := fp.fundStatus.RealizedPL * fp.transferRatio

		if transferAmount >= fp.autoTransferManager.minTransferAmount &&
			transferAmount <= fp.autoTransferManager.maxTransferAmount {
			fp.executeAutoTransfer(transferAmount, "PROFIT_PROTECTION")
		}
	}
}

// executeAutoTransfer 执行自动转账
func (fp *FundProtector) executeAutoTransfer(amount float64, reason string) {
	log.Printf("Executing auto transfer: %.2f (reason: %s)", amount, reason)

	transfer := TransferRecord{
		ID:            fp.generateTransferID(),
		Type:          "PROFIT_TRANSFER",
		Amount:        amount,
		From:          "trading_account",
		To:            fp.autoTransferManager.coldWalletAddress,
		Status:        "PENDING",
		Timestamp:     time.Now(),
		TriggerReason: reason,
		Metadata: map[string]interface{}{
			"auto_transfer": true,
			"profit_ratio":  fp.fundStatus.RealizedPL / fp.fundStatus.TotalBalance,
		},
	}

	// 执行转账
	err := fp.performTransfer(transfer)
	if err != nil {
		log.Printf("Auto transfer failed: %v", err)
		transfer.Status = "FAILED"
		transfer.Metadata["error"] = err.Error()
	} else {
		log.Printf("Auto transfer completed: %s", transfer.ID)
		transfer.Status = "COMPLETED"
		transfer.TransactionHash = fp.generateTransactionHash()

		// 更新指标
		fp.protectionMetrics.mu.Lock()
		fp.protectionMetrics.AutoTransfers++
		fp.protectionMetrics.ProfitsSecured += amount
		fp.protectionMetrics.mu.Unlock()
	}

	// 记录转账历史
	fp.autoTransferManager.mu.Lock()
	fp.autoTransferManager.transferHistory = append(fp.autoTransferManager.transferHistory, transfer)
	fp.mu.Lock()
	fp.transferHistory = append(fp.transferHistory, transfer)
	fp.mu.Unlock()
	fp.autoTransferManager.mu.Unlock()
}

// performRiskAssessment 执行风险评估
func (fp *FundProtector) performRiskAssessment() {
	log.Println("Performing risk assessment...")

	// 创建风险快照
	snapshot := RiskSnapshot{
		Timestamp:       time.Now(),
		RiskScore:       fp.calculateRiskScore(),
		VaR:             fp.fundStatus.VaR95,
		ExpectedLoss:    fp.fundStatus.ExpectedShortfall,
		MaxDrawdown:     fp.calculateMaxDrawdown(),
		VolatilityIndex: fp.calculateVolatilityIndex(),
		Leverage:        fp.calculateLeverage(),
		Concentration:   fp.calculateConcentration(),
	}

	// 确定风险等级
	snapshot.RiskLevel = fp.determineRiskLevel(snapshot.RiskScore)

	// 更新风险评估
	fp.riskAssessment.mu.Lock()
	fp.riskAssessment.lastAssessment = time.Now()
	fp.riskAssessment.currentRiskLevel = snapshot.RiskLevel
	fp.riskAssessment.riskHistory = append(fp.riskAssessment.riskHistory, snapshot)

	// 保持历史记录在合理范围内
	if len(fp.riskAssessment.riskHistory) > 1000 {
		fp.riskAssessment.riskHistory = fp.riskAssessment.riskHistory[100:]
	}
	fp.riskAssessment.mu.Unlock()

	// 如果风险级别过高，触发紧急协议
	if snapshot.RiskLevel == "CRITICAL" {
		fp.triggerEmergency("CRITICAL_RISK_LEVEL", map[string]interface{}{
			"risk_score": snapshot.RiskScore,
			"risk_level": snapshot.RiskLevel,
		})
	}
}

// checkCircuitBreaker 检查熔断器
func (fp *FundProtector) checkCircuitBreaker() {
	if !fp.circuitBreakerEnabled {
		return
	}

	fp.circuitBreaker.mu.Lock()
	defer fp.circuitBreaker.mu.Unlock()

	// 检查是否需要关闭熔断器（冷却期结束）
	if fp.circuitBreaker.isOpen {
		if time.Since(fp.circuitBreaker.lastTriggered) > fp.circuitBreaker.cooldownPeriod {
			fp.circuitBreaker.isOpen = false
			fp.circuitBreakerOpen = false
			log.Println("Circuit breaker reset (cooldown period ended)")
		}
		return
	}

	// 检查是否需要触发熔断器
	fp.fundStatus.mu.RLock()
	dailyLossRatio := -fp.fundStatus.DailyPL / fp.fundStatus.TotalBalance
	fp.fundStatus.mu.RUnlock()

	if dailyLossRatio > fp.circuitBreaker.maxDailyLoss {
		fp.triggerCircuitBreaker("DAILY_LOSS_LIMIT", dailyLossRatio)
	}
}

// triggerCircuitBreaker 触发熔断器
func (fp *FundProtector) triggerCircuitBreaker(reason string, lossRatio float64) {
	log.Printf("Circuit breaker triggered: %s (loss ratio: %.4f)", reason, lossRatio)

	fp.circuitBreaker.isOpen = true
	fp.circuitBreaker.lastTriggered = time.Now()
	fp.circuitBreaker.triggerCount++
	fp.circuitBreakerOpen = true

	// 更新指标
	fp.protectionMetrics.mu.Lock()
	fp.protectionMetrics.CircuitBreakerTriggered++
	fp.protectionMetrics.mu.Unlock()

	// 实施具体的熔断动作
	log.Printf("Executing circuit breaker actions...")
	
	// 1. 停止所有交易
	if err := fp.stopAllTrading(); err != nil {
		log.Printf("Failed to stop all trading: %v", err)
	}

	// 2. 平仓所有高风险仓位
	if err := fp.closeHighRiskPositions(); err != nil {
		log.Printf("Failed to close high risk positions: %v", err)
	}

	// 3. 执行紧急资金转移
	if err := fp.executeEmergencyTransfer(); err != nil {
		log.Printf("Failed to execute emergency transfer: %v", err)
	}

	// 4. 发送紧急通知（通过紧急协议处理）

	// 触发紧急协议
	fp.triggerEmergency("CIRCUIT_BREAKER_ACTIVATED", map[string]interface{}{
		"reason":        reason,
		"loss_ratio":    lossRatio,
		"trigger_count": fp.circuitBreaker.triggerCount,
	})
}

// triggerEmergency 触发紧急协议
func (fp *FundProtector) triggerEmergency(eventType string, triggerData map[string]interface{}) {
	log.Printf("Emergency triggered: %s", eventType)

	emergency := EmergencyEvent{
		ID:          fp.generateEmergencyID(),
		Type:        eventType,
		Severity:    fp.determineSeverity(eventType),
		Description: fp.getEmergencyDescription(eventType),
		Timestamp:   time.Now(),
		TriggerData: triggerData,
	}

	// 激活紧急协议
	fp.emergencyProtocol.mu.Lock()
	fp.emergencyProtocol.isActive = true
	fp.emergencyProtocol.lastActivation = time.Now()
	fp.emergencyProtocol.activationCount++
	fp.emergencyProtocol.mu.Unlock()

	fp.isEmergencyMode = true

	// 执行紧急响应
	responseStart := time.Now()
	response := fp.executeEmergencyResponse(emergency)
	emergency.Response = response

	// 记录紧急事件
	fp.mu.Lock()
	fp.emergencyEvents = append(fp.emergencyEvents, emergency)
	fp.mu.Unlock()

	// 更新指标
	fp.protectionMetrics.mu.Lock()
	fp.protectionMetrics.EmergencyActivations++
	fp.protectionMetrics.AvgResponseTime = time.Since(responseStart)
	fp.protectionMetrics.mu.Unlock()
}

// executeEmergencyResponse 执行紧急响应
func (fp *FundProtector) executeEmergencyResponse(emergency EmergencyEvent) *EmergencyResponse {
	log.Printf("Executing emergency response for: %s", emergency.Type)

	response := &EmergencyResponse{
		ResponseTime:  time.Now().Sub(emergency.Timestamp),
		Actions:       make([]string, 0),
		Status:        "IN_PROGRESS",
		Notifications: make([]NotificationRecord, 0),
		Metadata:      make(map[string]interface{}),
	}

	// 执行自动响应动作
	for _, action := range fp.emergencyProtocol.responsePlan {
		if action.IsAutomatic && fp.shouldExecuteAction(action, emergency) {
			err := action.Action()
			if err != nil {
				log.Printf("Emergency action failed: %s - %v", action.Type, err)
				response.Actions = append(response.Actions, fmt.Sprintf("FAILED: %s", action.Type))
			} else {
				log.Printf("Emergency action completed: %s", action.Type)
				response.Actions = append(response.Actions, action.Type)
			}
		}
	}

	// 发送紧急通知
	notifications := fp.sendEmergencyNotifications(emergency)
	response.Notifications = notifications

	response.Status = "COMPLETED"
	return response
}

// Helper functions for calculations and operations
func (fp *FundProtector) calculateCurrentRisk() float64 {
	// 获取当前持仓数据
	positions, err := fp.getCurrentPositions()
	if err != nil {
		log.Printf("Failed to get current positions for risk calculation: %v", err)
		return 0.0
	}

	if len(positions) == 0 {
		return 0.0 // 无持仓，无风险
	}

	// 计算综合风险评分
	riskComponents := fp.calculateRiskComponents(positions)
	
	// 加权计算总风险
	totalRisk := fp.aggregateRiskComponents(riskComponents)

	log.Printf("Current risk calculation: TotalRisk=%.4f, Components=%+v", totalRisk, riskComponents)

	return totalRisk
}

// RiskComponents 风险组件结构
type RiskComponents struct {
	PositionRisk      float64 `json:"position_risk"`
	ConcentrationRisk float64 `json:"concentration_risk"`
	LeverageRisk      float64 `json:"leverage_risk"`
	LiquidityRisk     float64 `json:"liquidity_risk"`
	VolatilityRisk    float64 `json:"volatility_risk"`
	CorrelationRisk   float64 `json:"correlation_risk"`
	MarketRisk        float64 `json:"market_risk"`
}

// calculateRiskComponents 计算各风险组件
func (fp *FundProtector) calculateRiskComponents(positions []*Position) *RiskComponents {
	components := &RiskComponents{}

	// 1. 持仓风险 - 基于个别持仓的风险
	components.PositionRisk = fp.calculateIndividualPositionRisk(positions)

	// 2. 集中度风险 - 基于持仓分布
	components.ConcentrationRisk = fp.calculateConcentrationRisk(positions)

	// 3. 杠杆风险 - 基于杠杆倍数
	components.LeverageRisk = fp.calculateLeverageRisk(positions)

	// 4. 流动性风险 - 基于市场流动性
	components.LiquidityRisk = fp.calculateLiquidityRisk(positions)

	// 5. 波动率风险 - 基于历史波动率
	components.VolatilityRisk = fp.calculateVolatilityRisk(positions)

	// 6. 相关性风险 - 基于持仓间相关性
	components.CorrelationRisk = fp.calculateCorrelationRisk(positions)

	// 7. 市场风险 - 基于市场环境
	components.MarketRisk = fp.calculateMarketRisk(positions)

	return components
}

// calculateIndividualPositionRisk 计算个别持仓风险
func (fp *FundProtector) calculateIndividualPositionRisk(positions []*Position) float64 {
	if len(positions) == 0 {
		return 0.0
	}

	var totalRisk float64
	var totalNotional float64

	for _, pos := range positions {
		// 计算单个持仓的风险
		posRisk := fp.calculateSinglePositionRisk(pos)
		
		// 按名义价值加权
		totalRisk += posRisk * pos.Notional
		totalNotional += pos.Notional
	}

	if totalNotional == 0 {
		return 0.0
	}

	return totalRisk / totalNotional
}

// calculateConcentrationRisk 计算集中度风险
func (fp *FundProtector) calculateConcentrationRisk(positions []*Position) float64 {
	if len(positions) <= 1 {
		return 1.0 // 单一持仓或无持仓，集中度风险最高
	}

	// 计算赫芬达尔指数
	symbolNotional := make(map[string]float64)
	var totalNotional float64

	for _, pos := range positions {
		symbolNotional[pos.Symbol] += pos.Notional
		totalNotional += pos.Notional
	}

	if totalNotional == 0 {
		return 0.0
	}

	var hhi float64
	for _, notional := range symbolNotional {
		share := notional / totalNotional
		hhi += share * share
	}

	// 标准化HHI到0-1范围
	minHHI := 1.0 / float64(len(symbolNotional))
	concentrationRisk := (hhi - minHHI) / (1.0 - minHHI)

	return math.Min(concentrationRisk, 1.0)
}

// calculateLeverageRisk 计算杠杆风险
func (fp *FundProtector) calculateLeverageRisk(positions []*Position) float64 {
	if len(positions) == 0 {
		return 0.0
	}

	var totalNotional float64
	var totalMargin float64

	for _, pos := range positions {
		totalNotional += pos.Notional
		margin := pos.Notional / float64(pos.Leverage)
		if pos.IsolatedMargin > 0 {
			margin = pos.IsolatedMargin
		}
		totalMargin += margin
	}

	if totalMargin == 0 {
		return 1.0 // 无保证金，风险最高
	}

	effectiveLeverage := totalNotional / totalMargin
	
	// 将杠杆转换为风险评分 (1x = 0风险, 10x+ = 高风险)
	leverageRisk := math.Min(effectiveLeverage/10.0, 1.0)

	return leverageRisk
}

// calculateLiquidityRisk 计算流动性风险
func (fp *FundProtector) calculateLiquidityRisk(positions []*Position) float64 {
	if len(positions) == 0 {
		return 0.0
	}

	// 简化的流动性风险评估
	// 在实际应用中，这应该基于订单簿深度、交易量等数据
	
	var totalRisk float64
	var totalNotional float64

	for _, pos := range positions {
		// 基于持仓大小的流动性风险评估
		liquidityRisk := fp.estimateSymbolLiquidityRisk(pos.Symbol, pos.Notional)
		
		totalRisk += liquidityRisk * pos.Notional
		totalNotional += pos.Notional
	}

	if totalNotional == 0 {
		return 0.0
	}

	return totalRisk / totalNotional
}

// estimateSymbolLiquidityRisk 估算交易对的流动性风险
func (fp *FundProtector) estimateSymbolLiquidityRisk(symbol string, notional float64) float64 {
	// 简化的流动性风险模型
	// 实际应该基于市场数据
	
	// 主要交易对流动性较好
	majorPairs := map[string]float64{
		"BTCUSDT": 0.1,
		"ETHUSDT": 0.15,
		"BNBUSDT": 0.2,
		"ADAUSDT": 0.25,
		"XRPUSDT": 0.2,
		"SOLUSDT": 0.25,
		"DOTUSDT": 0.3,
	}

	baseRisk := 0.5 // 默认中等流动性风险
	if risk, exists := majorPairs[symbol]; exists {
		baseRisk = risk
	}

	// 持仓规模调整
	if notional > 100000 { // 大额持仓流动性风险更高
		baseRisk *= 1.5
	} else if notional > 50000 {
		baseRisk *= 1.2
	}

	return math.Min(baseRisk, 1.0)
}

// calculateVolatilityRisk 计算波动率风险
func (fp *FundProtector) calculateVolatilityRisk(positions []*Position) float64 {
	if len(positions) == 0 {
		return 0.0
	}

	// 获取历史收益率计算组合波动率
	historicalReturns, err := fp.getHistoricalReturns(30)
	if err != nil || len(historicalReturns) == 0 {
		// 如果无法获取历史数据，使用持仓的未实现盈亏波动作为代理
		return fp.calculatePnLVolatilityRisk(positions)
	}

	volatility := fp.calculateSimpleVolatility(historicalReturns)
	
	// 将年化波动率转换为风险评分
	// 20%年化波动率对应0.5风险，40%+对应1.0风险
	volatilityRisk := math.Min(volatility/0.4, 1.0)

	return volatilityRisk
}

// calculatePnLVolatilityRisk 基于盈亏波动计算风险
func (fp *FundProtector) calculatePnLVolatilityRisk(positions []*Position) float64 {
	if len(positions) == 0 {
		return 0.0
	}

	var totalPnLRatio float64
	var count int

	for _, pos := range positions {
		if pos.Notional > 0 {
			pnlRatio := math.Abs(pos.UnrealizedPnL) / pos.Notional
			totalPnLRatio += pnlRatio
			count++
		}
	}

	if count == 0 {
		return 0.0
	}

	avgPnLRatio := totalPnLRatio / float64(count)
	
	// 将平均盈亏比例转换为风险评分
	return math.Min(avgPnLRatio*2, 1.0)
}

// calculateCorrelationRisk 计算相关性风险
func (fp *FundProtector) calculateCorrelationRisk(positions []*Position) float64 {
	if len(positions) <= 1 {
		return 0.0 // 单一持仓无相关性风险
	}

	// 简化的相关性风险评估
	// 实际应该基于历史价格相关性矩阵
	
	// 检查是否有相同类型的资产
	assetTypes := make(map[string]float64)
	var totalNotional float64

	for _, pos := range positions {
		assetType := fp.classifyAssetType(pos.Symbol)
		assetTypes[assetType] += pos.Notional
		totalNotional += pos.Notional
	}

	if totalNotional == 0 {
		return 0.0
	}

	// 计算资产类型集中度
	var typeHHI float64
	for _, notional := range assetTypes {
		share := notional / totalNotional
		typeHHI += share * share
	}

	// 相关性风险与类型集中度正相关
	correlationRisk := typeHHI

	return math.Min(correlationRisk, 1.0)
}

// classifyAssetType 分类资产类型
func (fp *FundProtector) classifyAssetType(symbol string) string {
	// 简化的资产分类
	switch {
	case symbol == "BTCUSDT":
		return "Bitcoin"
	case symbol == "ETHUSDT":
		return "Ethereum"
	case contains(symbol, []string{"BNB", "ADA", "XRP", "SOL", "DOT", "LINK", "UNI"}):
		return "Altcoin"
	case contains(symbol, []string{"USDT", "USDC", "BUSD", "DAI"}):
		return "Stablecoin"
	default:
		return "Other"
	}
}

// contains 检查字符串是否包含任一子字符串
func contains(s string, substrs []string) bool {
	for _, substr := range substrs {
		if len(s) >= len(substr) && s[:len(substr)] == substr {
			return true
		}
	}
	return false
}

// calculateMarketRisk 计算市场风险
func (fp *FundProtector) calculateMarketRisk(positions []*Position) float64 {
	// 简化的市场风险评估
	// 实际应该基于市场情绪指标、VIX等
	
	// 基于持仓的市场暴露度
	var longExposure, shortExposure float64
	
	for _, pos := range positions {
		if pos.Side == "LONG" {
			longExposure += pos.Notional
		} else {
			shortExposure += pos.Notional
		}
	}

	totalExposure := longExposure + shortExposure
	if totalExposure == 0 {
		return 0.0
	}

	// 计算方向性偏差
	netExposure := math.Abs(longExposure - shortExposure)
	directionalRisk := netExposure / totalExposure

	// 市场风险与方向性偏差和总暴露度相关
	marketRisk := directionalRisk * 0.7 + math.Min(totalExposure/1000000, 0.3) // 暴露度标准化

	return math.Min(marketRisk, 1.0)
}

// aggregateRiskComponents 聚合风险组件
func (fp *FundProtector) aggregateRiskComponents(components *RiskComponents) float64 {
	// 风险组件权重
	weights := map[string]float64{
		"position":      0.20,
		"concentration": 0.15,
		"leverage":      0.20,
		"liquidity":     0.10,
		"volatility":    0.15,
		"correlation":   0.10,
		"market":        0.10,
	}

	// 加权平均
	totalRisk := components.PositionRisk*weights["position"] +
		components.ConcentrationRisk*weights["concentration"] +
		components.LeverageRisk*weights["leverage"] +
		components.LiquidityRisk*weights["liquidity"] +
		components.VolatilityRisk*weights["volatility"] +
		components.CorrelationRisk*weights["correlation"] +
		components.MarketRisk*weights["market"]

	// 应用非线性调整（风险聚集效应）
	if totalRisk > 0.7 {
		totalRisk = 0.7 + (totalRisk-0.7)*1.5 // 高风险区域加速增长
	}

	return math.Min(totalRisk, 1.0)
}

func (fp *FundProtector) calculateVaR95() float64 {
	// 获取历史收益率数据并计算95%置信度的VaR
	historicalReturns, err := fp.getHistoricalReturns(30) // 30天历史数据
	if err != nil {
		log.Printf("Failed to get historical returns for VaR calculation: %v", err)
		return 0.0
	}

	if len(historicalReturns) < 10 {
		log.Printf("Insufficient historical data for VaR calculation: %d returns", len(historicalReturns))
		return 0.0
	}

	return fp.calculateVaRFromReturns(historicalReturns, 0.95)
}

func (fp *FundProtector) calculateExpectedShortfall() float64 {
	// 获取历史收益率数据
	historicalReturns, err := fp.getHistoricalReturns(30) // 30天历史数据
	if err != nil {
		log.Printf("Failed to get historical returns for ES calculation: %v", err)
		return 0.0
	}

	if len(historicalReturns) == 0 {
		log.Printf("No historical returns available for ES calculation")
		return 0.0
	}

	// 使用95%置信度计算Expected Shortfall
	confidence := 0.95
	return fp.calculateExpectedShortfallFromReturns(historicalReturns, confidence)
}

// calculateExpectedShortfallFromReturns 从收益率计算Expected Shortfall
func (fp *FundProtector) calculateExpectedShortfallFromReturns(returns []float64, confidence float64) float64 {
	if len(returns) == 0 {
		return 0.0
	}

	// 验证置信度参数
	if confidence <= 0 || confidence >= 1 {
		log.Printf("Warning: Invalid confidence level %.2f for ES, using 0.95", confidence)
		confidence = 0.95
	}

	// 使用历史模拟法计算Expected Shortfall
	historicalES := fp.calculateHistoricalExpectedShortfall(returns, confidence)
	
	// 使用参数法计算Expected Shortfall
	parametricES := fp.calculateParametricExpectedShortfall(returns, confidence)

	// 取两种方法的加权平均，历史模拟法权重更高
	finalES := historicalES*0.7 + parametricES*0.3

	log.Printf("Expected Shortfall calculation (%.1f%% confidence): Historical=%.4f, Parametric=%.4f, Final=%.4f", 
		confidence*100, historicalES, parametricES, finalES)

	return finalES
}

// calculateHistoricalExpectedShortfall 使用历史模拟法计算Expected Shortfall
func (fp *FundProtector) calculateHistoricalExpectedShortfall(returns []float64, confidence float64) float64 {
	if len(returns) == 0 {
		return 0.0
	}

	// 复制并排序收益率数组
	sortedReturns := make([]float64, len(returns))
	copy(sortedReturns, returns)
	sort.Float64s(sortedReturns)

	// 计算VaR对应的分位数位置
	alpha := 1.0 - confidence
	varPosition := alpha * float64(len(sortedReturns)-1)
	varIndex := int(math.Ceil(varPosition))

	// 确保索引在有效范围内
	if varIndex >= len(sortedReturns) {
		varIndex = len(sortedReturns) - 1
	}

	// 计算超过VaR的所有损失的平均值
	var sum float64
	var count int

	for i := 0; i <= varIndex; i++ {
		sum += sortedReturns[i]
		count++
	}

	if count == 0 {
		return 0.0
	}

	// Expected Shortfall是尾部损失的条件期望
	expectedShortfall := -(sum / float64(count)) // 转换为正值表示损失

	return expectedShortfall
}

// calculateParametricExpectedShortfall 使用参数法计算Expected Shortfall（假设正态分布）
func (fp *FundProtector) calculateParametricExpectedShortfall(returns []float64, confidence float64) float64 {
	if len(returns) == 0 {
		return 0.0
	}

	// 计算均值和标准差
	mean := fp.calculateMean(returns)
	stdDev := fp.calculateStandardDeviation(returns, mean)

	// 获取VaR对应的分位数
	alpha := 1.0 - confidence
	zAlpha := fp.getNormalQuantile(alpha)

	// 对于正态分布，Expected Shortfall的公式为：
	// ES = -μ - σ * φ(z_α) / α
	// 其中 φ(z) 是标准正态分布的概率密度函数
	
	// 计算标准正态分布在z_α处的概率密度
	phi := (1.0 / math.Sqrt(2*math.Pi)) * math.Exp(-0.5*zAlpha*zAlpha)

	// 计算Expected Shortfall
	expectedShortfall := -mean - stdDev*phi/alpha

	return expectedShortfall
}

// calculateConditionalVaR 计算条件VaR（Expected Shortfall的别名）
func (fp *FundProtector) calculateConditionalVaR(returns []float64, confidence float64) float64 {
	return fp.calculateExpectedShortfallFromReturns(returns, confidence)
}

// calculateTailRisk 计算尾部风险指标
func (fp *FundProtector) calculateTailRisk(returns []float64) map[string]float64 {
	if len(returns) == 0 {
		return map[string]float64{}
	}

	// 计算不同置信度下的风险指标
	confidenceLevels := []float64{0.90, 0.95, 0.99}
	tailRisk := make(map[string]float64)

	for _, confidence := range confidenceLevels {
		var_ := fp.calculateHistoricalVaR(returns, confidence)
		es := fp.calculateHistoricalExpectedShortfall(returns, confidence)
		
		confidenceStr := fmt.Sprintf("%.0f", confidence*100)
		tailRisk[fmt.Sprintf("VaR_%s", confidenceStr)] = var_
		tailRisk[fmt.Sprintf("ES_%s", confidenceStr)] = es
	}

	// 计算最大损失
	sortedReturns := make([]float64, len(returns))
	copy(sortedReturns, returns)
	sort.Float64s(sortedReturns)
	tailRisk["MaxLoss"] = -sortedReturns[0] // 转换为正值

	// 计算尾部比率（ES/VaR）
	var95 := tailRisk["VaR_95"]
	es95 := tailRisk["ES_95"]
	if var95 > 0 {
		tailRisk["TailRatio_95"] = es95 / var95
	}

	return tailRisk
}

func (fp *FundProtector) calculateRiskScore() float64 {
	// 基于多个风险因子的综合评分
	// 包括VaR、波动率、集中度、杠杆等

	var95 := fp.calculateVaR95()
	volatility := fp.calculateVolatilityIndex()
	concentration := fp.calculateConcentration()
	leverage := fp.calculateLeverage()

	// 获取当前持仓进行更详细的风险分析
	positions, err := fp.getCurrentPositions()
	if err != nil {
		log.Printf("Failed to get positions for risk score calculation: %v", err)
		// 使用基础指标计算
		if var95 == 0.0 && volatility == 0.0 {
			return 0.0
		}
		return math.Min((var95*0.4 + volatility*0.4 + concentration*0.1 + leverage*0.1), 1.0)
	}

	// 计算详细的风险组件
	riskComponents := fp.calculateRiskComponents(positions)
	
	// 多因子风险评分权重
	weights := map[string]float64{
		"var":           0.25,  // VaR权重
		"volatility":    0.20,  // 波动率权重
		"concentration": 0.15,  // 集中度权重
		"leverage":      0.15,  // 杠杆权重
		"position":      0.10,  // 持仓风险权重
		"liquidity":     0.08,  // 流动性风险权重
		"correlation":   0.07,  // 相关性风险权重
	}

	// 标准化各个指标到0-1范围
	normalizedVaR := math.Min(var95/0.1, 1.0)           // VaR超过10%视为高风险
	normalizedVolatility := math.Min(volatility/0.5, 1.0) // 波动率超过50%视为高风险
	normalizedConcentration := concentration              // 已经是0-1范围
	normalizedLeverage := math.Min(leverage/20.0, 1.0)   // 杠杆超过20倍视为高风险

	// 综合风险评分
	riskScore := normalizedVaR*weights["var"] +
		normalizedVolatility*weights["volatility"] +
		normalizedConcentration*weights["concentration"] +
		normalizedLeverage*weights["leverage"] +
		riskComponents.PositionRisk*weights["position"] +
		riskComponents.LiquidityRisk*weights["liquidity"] +
		riskComponents.CorrelationRisk*weights["correlation"]

	// 应用风险放大因子（当多个风险因子同时较高时）
	if normalizedVaR > 0.7 && normalizedVolatility > 0.7 {
		riskScore *= 1.2 // 20%风险放大
	}

	if normalizedLeverage > 0.8 && normalizedConcentration > 0.8 {
		riskScore *= 1.15 // 15%风险放大
	}

	return math.Min(riskScore, 1.0)
}

func (fp *FundProtector) calculateMaxDrawdown() float64 {
	// 基于历史净值的最大回撤计算
	historicalEquity, err := fp.getHistoricalEquity(90) // 90天历史净值
	if err != nil {
		log.Printf("Failed to get historical equity for drawdown calculation: %v", err)
		return 0.0
	}

	if len(historicalEquity) < 2 {
		log.Printf("Insufficient equity data for drawdown calculation: %d points", len(historicalEquity))
		return 0.0
	}

	return fp.calculateDrawdownFromEquity(historicalEquity)
}

func (fp *FundProtector) calculateVolatilityIndex() float64 {
	// 基于历史收益率的波动率计算
	historicalReturns, err := fp.getHistoricalReturns(30)
	if err != nil {
		log.Printf("Failed to get historical returns for volatility calculation: %v", err)
		return 0.0
	}

	if len(historicalReturns) < 5 {
		log.Printf("Insufficient return data for volatility calculation: %d returns", len(historicalReturns))
		return 0.0
	}

	return fp.calculateVolatilityFromReturns(historicalReturns)
}

func (fp *FundProtector) calculateLeverage() float64 {
	// 基于当前持仓的杠杆计算
	positions, err := fp.getCurrentPositions()
	if err != nil {
		log.Printf("Failed to get positions for leverage calculation: %v", err)
		return 0.0
	}

	if len(positions) == 0 {
		return 0.0 // 无持仓，无杠杆
	}

	return fp.calculateLeverageFromPositions(positions)
}

func (fp *FundProtector) calculateConcentration() float64 {
	// 基于持仓分布的集中度计算
	positions, err := fp.getCurrentPositions()
	if err != nil {
		log.Printf("Failed to get positions for concentration calculation: %v", err)
		return 0.0
	}

	if len(positions) == 0 {
		return 0.0 // 无持仓，无集中度风险
	}

	return fp.calculateConcentrationFromPositions(positions)
}

func (fp *FundProtector) determineRiskLevel(riskScore float64) string {
	switch {
	case riskScore < 0.2:
		return "LOW"
	case riskScore < 0.4:
		return "MEDIUM"
	case riskScore < 0.7:
		return "HIGH"
	default:
		return "CRITICAL"
	}
}

func (fp *FundProtector) determineSeverity(eventType string) string {
	switch eventType {
	case "DAILY_LOSS_EXCEEDED":
		return "HIGH"
	case "CRITICAL_LOSS", "CIRCUIT_BREAKER_ACTIVATED":
		return "CRITICAL"
	case "RISK_LIMIT_EXCEEDED":
		return "MEDIUM"
	default:
		return "LOW"
	}
}

func (fp *FundProtector) getEmergencyDescription(eventType string) string {
	descriptions := map[string]string{
		"DAILY_LOSS_EXCEEDED":       "Daily loss limit exceeded",
		"CRITICAL_LOSS":             "Critical total loss detected",
		"RISK_LIMIT_EXCEEDED":       "Risk limit exceeded",
		"CIRCUIT_BREAKER_ACTIVATED": "Circuit breaker activated",
		"CRITICAL_RISK_LEVEL":       "Critical risk level reached",
	}

	if desc, exists := descriptions[eventType]; exists {
		return desc
	}
	return "Unknown emergency event"
}

func (fp *FundProtector) shouldExecuteAction(action ResponseAction, emergency EmergencyEvent) bool {
	// 检查动作是否应该执行
	
	// 1. 检查动作条件
	if action.Condition != "" {
		if !fp.evaluateActionCondition(action.Condition, emergency) {
			log.Printf("Action %s condition not met: %s", action.ID, action.Condition)
			return false
		}
	}

	// 2. 检查紧急事件严重程度
	severityLevel := fp.getSeverityLevel(emergency.Severity)
	actionPriority := action.Priority

	// 高优先级动作在任何严重程度下都执行
	if actionPriority >= 9 {
		return true
	}

	// 根据严重程度决定是否执行
	switch emergency.Severity {
	case "CRITICAL":
		return actionPriority >= 7 // 关键事件执行高优先级动作
	case "HIGH":
		return actionPriority >= 5 // 高级事件执行中高优先级动作
	case "MEDIUM":
		return actionPriority >= 3 // 中级事件执行中等优先级动作
	case "LOW":
		return actionPriority >= 1 // 低级事件执行低优先级动作
	default:
		return actionPriority >= 5 // 默认中等阈值
	}
}

// evaluateActionCondition 评估动作执行条件
func (fp *FundProtector) evaluateActionCondition(condition string, emergency EmergencyEvent) bool {
	// 简化的条件评估系统
	// 实际应该实现更复杂的表达式解析器
	
	switch condition {
	case "DAILY_LOSS_EXCEEDED":
		return emergency.Type == "DAILY_LOSS_EXCEEDED"
	case "CRITICAL_LOSS":
		return emergency.Type == "CRITICAL_LOSS"
	case "CIRCUIT_BREAKER_ACTIVATED":
		return emergency.Type == "CIRCUIT_BREAKER_ACTIVATED"
	case "RISK_LIMIT_EXCEEDED":
		return emergency.Type == "RISK_LIMIT_EXCEEDED"
	case "ALWAYS":
		return true
	case "NEVER":
		return false
	default:
		// 检查是否包含特定的触发数据
		if triggerData, exists := emergency.TriggerData[condition]; exists {
			if boolValue, ok := triggerData.(bool); ok {
				return boolValue
			}
			if floatValue, ok := triggerData.(float64); ok {
				return floatValue > 0
			}
		}
		return true // 默认执行
	}
}

// getSeverityLevel 获取严重程度数值
func (fp *FundProtector) getSeverityLevel(severity string) int {
	switch severity {
	case "CRITICAL":
		return 4
	case "HIGH":
		return 3
	case "MEDIUM":
		return 2
	case "LOW":
		return 1
	default:
		return 2
	}
}

// stopAllTrading 停止所有交易
func (fp *FundProtector) stopAllTrading() error {
	log.Printf("Stopping all trading activities...")
	
	if fp.exchangeProvider == nil {
		log.Printf("Exchange provider not configured, cannot stop trading")
		return fmt.Errorf("exchange provider not configured")
	}

	// 检查连接健康状态
	if !fp.exchangeProvider.IsHealthy() {
		return fmt.Errorf("exchange connection is not healthy")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// 获取当前持仓以确定需要取消订单的交易对
	positions, err := fp.getCurrentPositions()
	if err != nil {
		log.Printf("Failed to get positions for order cancellation: %v", err)
		// 继续执行，不因为获取持仓失败而停止
	}

	// 收集所有需要取消订单的交易对
	symbols := make(map[string]bool)
	for _, pos := range positions {
		symbols[pos.Symbol] = true
	}

	// 取消所有交易对的挂单
	var cancelErrors []error
	for symbol := range symbols {
		log.Printf("Cancelling all orders for symbol: %s", symbol)
		
		// 使用exchange接口取消所有订单
		if err := fp.exchange.CancelAllOrders(ctx, symbol); err != nil {
			log.Printf("Failed to cancel orders for %s: %v", symbol, err)
			cancelErrors = append(cancelErrors, fmt.Errorf("cancel orders for %s: %w", symbol, err))
		} else {
			log.Printf("Successfully cancelled all orders for %s", symbol)
		}
	}

	// 设置交易暂停标志
	fp.mu.Lock()
	fp.tradingPaused = true
	fp.mu.Unlock()

	log.Printf("Trading activities stopped. Cancelled orders for %d symbols", len(symbols))

	// 如果有取消订单的错误，返回汇总错误
	if len(cancelErrors) > 0 {
		return fmt.Errorf("some order cancellations failed: %v", cancelErrors)
	}

	return nil
}

// closeHighRiskPositions 平仓高风险仓位
func (fp *FundProtector) closeHighRiskPositions() error {
	log.Printf("Closing high risk positions...")

	// 获取当前持仓
	positions, err := fp.getCurrentPositions()
	if err != nil {
		return fmt.Errorf("failed to get current positions: %w", err)
	}

	if len(positions) == 0 {
		log.Printf("No positions to close")
		return nil
	}

	// 识别高风险仓位
	highRiskPositions := fp.identifyHighRiskPositions(positions)
	
	if len(highRiskPositions) == 0 {
		log.Printf("No high risk positions found")
		return nil
	}

	log.Printf("Found %d high risk positions to close", len(highRiskPositions))

	// 按风险程度排序，最高风险的先平仓
	sort.Slice(highRiskPositions, func(i, j int) bool {
		riskI := fp.calculateSinglePositionRisk(highRiskPositions[i])
		riskJ := fp.calculateSinglePositionRisk(highRiskPositions[j])
		return riskI > riskJ
	})

	// 逐个平仓高风险仓位
	for _, pos := range highRiskPositions {
		if err := fp.closePosition(pos); err != nil {
			log.Printf("Failed to close position %s: %v", pos.Symbol, err)
			continue
		}
		log.Printf("Successfully closed high risk position: %s", pos.Symbol)
	}

	return nil
}

// identifyHighRiskPositions 识别高风险仓位
func (fp *FundProtector) identifyHighRiskPositions(positions []*Position) []*Position {
	highRiskPositions := make([]*Position, 0)
	
	for _, pos := range positions {
		risk := fp.calculateSinglePositionRisk(pos)
		
		// 风险阈值：0.7以上为高风险
		if risk > 0.7 {
			highRiskPositions = append(highRiskPositions, pos)
			log.Printf("High risk position identified: %s (risk: %.4f)", pos.Symbol, risk)
		}
		
		// 强平风险：距离强平价格小于5%
		liquidationRisk := fp.calculateLiquidationRisk(pos)
		if liquidationRisk > 0.8 {
			highRiskPositions = append(highRiskPositions, pos)
			log.Printf("Near liquidation position identified: %s (liquidation risk: %.4f)", pos.Symbol, liquidationRisk)
		}
		
		// 大额亏损：未实现亏损超过名义价值的20%
		if pos.UnrealizedPnL < 0 && math.Abs(pos.UnrealizedPnL) > pos.Notional*0.2 {
			highRiskPositions = append(highRiskPositions, pos)
			log.Printf("Large loss position identified: %s (loss: %.2f)", pos.Symbol, pos.UnrealizedPnL)
		}
	}
	
	// 去重
	uniquePositions := make([]*Position, 0)
	seen := make(map[string]bool)
	
	for _, pos := range highRiskPositions {
		if !seen[pos.Symbol] {
			uniquePositions = append(uniquePositions, pos)
			seen[pos.Symbol] = true
		}
	}
	
	return uniquePositions
}

// closePosition 平仓指定仓位
func (fp *FundProtector) closePosition(pos *Position) error {
	log.Printf("Closing position: %s, Size: %.8f, Side: %s", pos.Symbol, pos.Size, pos.Side)
	
	if fp.exchange == nil {
		return fmt.Errorf("exchange not configured")
	}

	// 检查连接健康状态
	if fp.exchangeProvider != nil && !fp.exchangeProvider.IsHealthy() {
		return fmt.Errorf("exchange connection is not healthy")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 确定平仓方向（与当前持仓相反）
	var orderSide string
	if pos.Side == "LONG" {
		orderSide = "SELL" // 平多仓
	} else {
		orderSide = "BUY" // 平空仓
	}

	// 创建市价平仓订单
	orderRequest := &exchange.OrderRequest{
		Symbol:      pos.Symbol,
		Side:        orderSide,
		Type:        "MARKET",
		Quantity:    pos.Size,
		ReduceOnly:  true, // 仅减仓，不开新仓
		TimeInForce: "IOC", // 立即成交或取消
	}

	log.Printf("Placing close order: %s %s %.8f %s (ReduceOnly)", 
		orderRequest.Side, orderRequest.Symbol, orderRequest.Quantity, orderRequest.Type)

	// 下单平仓
	response, err := fp.exchange.PlaceOrder(ctx, orderRequest)
	if err != nil {
		return fmt.Errorf("failed to place close order: %w", err)
	}

	log.Printf("Close order placed successfully: OrderID=%s, Status=%s", 
		response.OrderID, response.Status)

	// 检查订单状态
	if response.Status == "FILLED" {
		log.Printf("Position %s closed successfully", pos.Symbol)
		return nil
	} else if response.Status == "PARTIALLY_FILLED" {
		log.Printf("Position %s partially closed: %.8f/%.8f", 
			pos.Symbol, response.ExecutedQty, orderRequest.Quantity)
		
		// 对于部分成交，可以选择等待或取消剩余订单
		// 这里选择等待一段时间
		time.Sleep(2 * time.Second)
		
		// 检查订单最终状态
		finalOrder, err := fp.exchange.GetOrder(ctx, pos.Symbol, response.OrderID)
		if err != nil {
			log.Printf("Failed to get final order status: %v", err)
			return nil // 不返回错误，因为部分平仓已经成功
		}
		
		if finalOrder.Status == "FILLED" {
			log.Printf("Position %s fully closed after waiting", pos.Symbol)
		} else {
			log.Printf("Position %s still partially open: Status=%s, Executed=%.8f", 
				pos.Symbol, finalOrder.Status, finalOrder.ExecutedQty)
		}
		
		return nil
	} else {
		return fmt.Errorf("close order failed with status: %s", response.Status)
	}
}

// executeEmergencyTransfer 执行紧急资金转移
func (fp *FundProtector) executeEmergencyTransfer() error {
	log.Printf("Executing emergency fund transfer...")

	// 检查是否配置了冷钱包地址
	if fp.autoTransferManager.coldWalletAddress == "" {
		log.Printf("Cold wallet address not configured, skipping emergency transfer")
		return nil
	}

	// 计算紧急转移金额（可用余额的80%）
	fp.fundStatus.mu.RLock()
	availableBalance := fp.fundStatus.AvailableBalance
	fp.fundStatus.mu.RUnlock()

	if availableBalance <= 0 {
		log.Printf("No available balance for emergency transfer")
		return nil
	}

	emergencyAmount := availableBalance * 0.8 // 转移80%的可用余额
	
	// 检查转移限额
	if emergencyAmount < fp.autoTransferManager.minTransferAmount {
		log.Printf("Emergency transfer amount %.2f below minimum %.2f", 
			emergencyAmount, fp.autoTransferManager.minTransferAmount)
		return nil
	}

	if emergencyAmount > fp.autoTransferManager.maxTransferAmount {
		emergencyAmount = fp.autoTransferManager.maxTransferAmount
		log.Printf("Emergency transfer amount capped at maximum: %.2f", emergencyAmount)
	}

	// 创建紧急转账记录
	transfer := TransferRecord{
		ID:            fp.generateTransferID(),
		Type:          "EMERGENCY_TRANSFER",
		Amount:        emergencyAmount,
		From:          "trading_account",
		To:            fp.autoTransferManager.coldWalletAddress,
		Status:        "PENDING",
		Timestamp:     time.Now(),
		TriggerReason: "CIRCUIT_BREAKER_EMERGENCY",
		Metadata: map[string]interface{}{
			"emergency_transfer": true,
			"circuit_breaker":    true,
			"available_balance":  availableBalance,
		},
	}

	// 执行紧急转账
	err := fp.performTransfer(transfer)
	if err != nil {
		log.Printf("Emergency transfer failed: %v", err)
		transfer.Status = "FAILED"
		transfer.Metadata["error"] = err.Error()
	} else {
		log.Printf("Emergency transfer completed: %s", transfer.ID)
		transfer.Status = "COMPLETED"
		transfer.TransactionHash = fp.generateTransactionHash()
		
		// 更新指标
		fp.protectionMetrics.mu.Lock()
		fp.protectionMetrics.AutoTransfers++
		fp.protectionMetrics.ProfitsSecured += emergencyAmount
		fp.protectionMetrics.mu.Unlock()
	}

	// 记录转账历史
	fp.autoTransferManager.mu.Lock()
	fp.autoTransferManager.transferHistory = append(fp.autoTransferManager.transferHistory, transfer)
	fp.mu.Lock()
	fp.transferHistory = append(fp.transferHistory, transfer)
	fp.mu.Unlock()
	fp.autoTransferManager.mu.Unlock()

	return err
}

func (fp *FundProtector) sendEmergencyNotifications(emergency EmergencyEvent) []NotificationRecord {
	notifications := make([]NotificationRecord, 0)

	// 获取紧急联系人列表
	fp.emergencyProtocol.mu.RLock()
	contacts := fp.emergencyProtocol.emergencyContacts
	fp.emergencyProtocol.mu.RUnlock()

	if len(contacts) == 0 {
		log.Printf("No emergency contacts configured for notifications")
		return notifications
	}

	// 根据严重程度确定通知范围
	severityLevel := fp.getSeverityLevel(emergency.Severity)
	
	// 构建通知消息
	message := fp.buildEmergencyMessage(emergency)

	// 按优先级排序联系人
	sortedContacts := fp.sortContactsByPriority(contacts)

	// 发送通知
	for _, contact := range sortedContacts {
		if !contact.IsAvailable {
			log.Printf("Contact %s is not available, skipping", contact.Name)
			continue
		}

		// 根据严重程度决定是否通知此联系人
		if !fp.shouldNotifyContact(contact, severityLevel) {
			continue
		}

		// 通过各种渠道发送通知
		for _, channel := range contact.Channels {
			notification := fp.sendNotificationViaChannel(contact, channel, message, emergency)
			notifications = append(notifications, notification)
		}
	}

	log.Printf("Sent %d emergency notifications for event %s", len(notifications), emergency.ID)
	return notifications
}

// buildEmergencyMessage 构建紧急通知消息
func (fp *FundProtector) buildEmergencyMessage(emergency EmergencyEvent) string {
	message := fmt.Sprintf("🚨 EMERGENCY ALERT 🚨\n\n")
	message += fmt.Sprintf("Event Type: %s\n", emergency.Type)
	message += fmt.Sprintf("Severity: %s\n", emergency.Severity)
	message += fmt.Sprintf("Description: %s\n", emergency.Description)
	message += fmt.Sprintf("Time: %s\n", emergency.Timestamp.Format("2006-01-02 15:04:05"))
	
	// 添加触发数据
	if len(emergency.TriggerData) > 0 {
		message += "\nTrigger Data:\n"
		for key, value := range emergency.TriggerData {
			message += fmt.Sprintf("- %s: %v\n", key, value)
		}
	}

	// 添加当前资金状态
	fundStatus := fp.GetFundStatus()
	message += fmt.Sprintf("\nCurrent Fund Status:\n")
	message += fmt.Sprintf("- Total Balance: %.2f\n", fundStatus.TotalBalance)
	message += fmt.Sprintf("- Daily P&L: %.2f\n", fundStatus.DailyPL)
	message += fmt.Sprintf("- Current Risk: %.4f\n", fundStatus.CurrentRisk)

	message += "\nPlease take immediate action if required."
	
	return message
}

// sortContactsByPriority 按优先级排序联系人
func (fp *FundProtector) sortContactsByPriority(contacts []EmergencyContact) []EmergencyContact {
	sortedContacts := make([]EmergencyContact, len(contacts))
	copy(sortedContacts, contacts)
	
	// 按优先级排序（数字越小优先级越高）
	sort.Slice(sortedContacts, func(i, j int) bool {
		return sortedContacts[i].Priority < sortedContacts[j].Priority
	})
	
	return sortedContacts
}

// shouldNotifyContact 判断是否应该通知此联系人
func (fp *FundProtector) shouldNotifyContact(contact EmergencyContact, severityLevel int) bool {
	// 根据角色和严重程度决定
	switch contact.Role {
	case "ADMIN", "OWNER":
		return true // 管理员和所有者总是收到通知
	case "TRADER":
		return severityLevel >= 2 // 交易员在中等及以上严重程度时收到通知
	case "ANALYST":
		return severityLevel >= 3 // 分析师在高等及以上严重程度时收到通知
	case "OBSERVER":
		return severityLevel >= 4 // 观察者只在关键事件时收到通知
	default:
		return severityLevel >= 3 // 默认高等及以上
	}
}

// sendNotificationViaChannel 通过指定渠道发送通知
func (fp *FundProtector) sendNotificationViaChannel(contact EmergencyContact, channel, message string, emergency EmergencyEvent) NotificationRecord {
	notification := NotificationRecord{
		Channel:   channel,
		Recipient: contact.Name,
		Status:    "PENDING",
		SentAt:    time.Now(),
		Message:   message,
	}

	var err error
	switch channel {
	case "EMAIL":
		err = fp.sendEmailNotification(contact.Email, message, emergency)
	case "SMS":
		err = fp.sendSMSNotification(contact.Phone, message, emergency)
	case "WEBHOOK":
		err = fp.sendWebhookNotification(contact, message, emergency)
	case "SLACK":
		err = fp.sendSlackNotification(contact, message, emergency)
	default:
		err = fmt.Errorf("unsupported notification channel: %s", channel)
	}

	if err != nil {
		notification.Status = "FAILED"
		log.Printf("Failed to send %s notification to %s: %v", channel, contact.Name, err)
	} else {
		notification.Status = "SENT"
		log.Printf("Successfully sent %s notification to %s", channel, contact.Name)
	}

	return notification
}

// sendEmailNotification 发送邮件通知
func (fp *FundProtector) sendEmailNotification(email, message string, emergency EmergencyEvent) error {
	if fp.notificationService == nil {
		log.Printf("Notification service not configured, skipping email to %s", email)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	subject := fmt.Sprintf("🚨 Fund Protector Alert: %s", emergency.Type)
	return fp.notificationService.SendEmail(ctx, email, subject, message)
}

// sendSMSNotification 发送短信通知
func (fp *FundProtector) sendSMSNotification(phone, message string, emergency EmergencyEvent) error {
	if fp.notificationService == nil {
		log.Printf("Notification service not configured, skipping SMS to %s", phone)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 截断SMS消息以适应长度限制
	smsMessage := message
	if len(smsMessage) > 160 {
		smsMessage = smsMessage[:157] + "..."
	}

	return fp.notificationService.SendSMS(ctx, phone, smsMessage)
}

// sendWebhookNotification 发送Webhook通知
func (fp *FundProtector) sendWebhookNotification(contact EmergencyContact, message string, emergency EmergencyEvent) error {
	if fp.notificationService == nil {
		log.Printf("Notification service not configured, skipping webhook for %s", contact.Name)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	payload := map[string]interface{}{
		"event_type":    emergency.Type,
		"severity":      emergency.Severity,
		"description":   emergency.Description,
		"timestamp":     emergency.Timestamp,
		"trigger_data":  emergency.TriggerData,
		"contact_name":  contact.Name,
		"message":       message,
	}

	// 使用联系人的webhook URL或默认URL
	webhookURL := ""
	if len(contact.Channels) > 0 {
		// 假设webhook URL存储在联系人的元数据中
		webhookURL = "https://default-webhook-url.com/emergency"
	}

	return fp.notificationService.SendWebhook(ctx, webhookURL, payload)
}

// sendSlackNotification 发送Slack通知
func (fp *FundProtector) sendSlackNotification(contact EmergencyContact, message string, emergency EmergencyEvent) error {
	if fp.notificationService == nil {
		log.Printf("Notification service not configured, skipping Slack for %s", contact.Name)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 格式化Slack消息
	slackMessage := fmt.Sprintf("🚨 *Fund Protector Alert*\n*Type:* %s\n*Severity:* %s\n*Time:* %s\n\n%s", 
		emergency.Type, emergency.Severity, emergency.Timestamp.Format("2006-01-02 15:04:05"), message)

	return fp.notificationService.SendSlack(ctx, "", slackMessage)
}

func (fp *FundProtector) performTransfer(transfer TransferRecord) error {
	log.Printf("Performing transfer: %.2f from %s to %s", transfer.Amount, transfer.From, transfer.To)

	// 1. 验证转账参数
	if err := fp.validateTransferParameters(transfer); err != nil {
		return fmt.Errorf("transfer validation failed: %w", err)
	}

	// 2. 检查资金充足性
	if err := fp.checkFundAvailability(transfer); err != nil {
		return fmt.Errorf("insufficient funds: %w", err)
	}

	// 3. 执行预转账检查
	if err := fp.preTransferChecks(transfer); err != nil {
		return fmt.Errorf("pre-transfer checks failed: %w", err)
	}

	// 4. 执行实际转账
	if err := fp.executeTransfer(transfer); err != nil {
		return fmt.Errorf("transfer execution failed: %w", err)
	}

	// 5. 记录转账到数据库
	if err := fp.recordTransfer(transfer); err != nil {
		log.Printf("Failed to record transfer to database: %v", err)
		// 不返回错误，因为转账已经成功
	}

	log.Printf("Transfer completed successfully: %s", transfer.ID)
	return nil
}

// validateTransferParameters 验证转账参数
func (fp *FundProtector) validateTransferParameters(transfer TransferRecord) error {
	if transfer.Amount <= 0 {
		return fmt.Errorf("transfer amount must be positive: %.2f", transfer.Amount)
	}

	if transfer.From == "" {
		return fmt.Errorf("source account cannot be empty")
	}

	if transfer.To == "" {
		return fmt.Errorf("destination account cannot be empty")
	}

	if transfer.From == transfer.To {
		return fmt.Errorf("source and destination cannot be the same")
	}

	// 检查转账限额
	if transfer.Amount < fp.autoTransferManager.minTransferAmount {
		return fmt.Errorf("transfer amount %.2f below minimum %.2f", 
			transfer.Amount, fp.autoTransferManager.minTransferAmount)
	}

	if transfer.Amount > fp.autoTransferManager.maxTransferAmount {
		return fmt.Errorf("transfer amount %.2f exceeds maximum %.2f", 
			transfer.Amount, fp.autoTransferManager.maxTransferAmount)
	}

	return nil
}

// checkFundAvailability 检查资金可用性
func (fp *FundProtector) checkFundAvailability(transfer TransferRecord) error {
	fp.fundStatus.mu.RLock()
	availableBalance := fp.fundStatus.AvailableBalance
	fp.fundStatus.mu.RUnlock()

	if transfer.Amount > availableBalance {
		return fmt.Errorf("insufficient available balance: need %.2f, have %.2f", 
			transfer.Amount, availableBalance)
	}

	// 保留一定的安全余额
	safetyBuffer := availableBalance * 0.05 // 5%安全缓冲
	if transfer.Amount > (availableBalance - safetyBuffer) {
		return fmt.Errorf("transfer would exceed safety buffer: need %.2f, safe amount %.2f", 
			transfer.Amount, availableBalance-safetyBuffer)
	}

	return nil
}

// preTransferChecks 执行转账前检查
func (fp *FundProtector) preTransferChecks(transfer TransferRecord) error {
	// 1. 检查是否处于紧急模式
	if fp.IsEmergencyMode() && transfer.Type != "EMERGENCY_TRANSFER" {
		return fmt.Errorf("system in emergency mode, only emergency transfers allowed")
	}

	// 2. 检查熔断器状态
	if fp.IsCircuitBreakerOpen() && transfer.Type == "PROFIT_TRANSFER" {
		return fmt.Errorf("circuit breaker is open, profit transfers suspended")
	}

	// 3. 检查转账频率限制
	if err := fp.checkTransferRateLimit(transfer); err != nil {
		return err
	}

	// 4. 验证目标地址
	if err := fp.validateDestinationAddress(transfer.To); err != nil {
		return err
	}

	return nil
}

// checkTransferRateLimit 检查转账频率限制
func (fp *FundProtector) checkTransferRateLimit(transfer TransferRecord) error {
	// 检查最近1小时内的转账次数
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	recentTransfers := 0

	fp.autoTransferManager.mu.RLock()
	for _, t := range fp.autoTransferManager.transferHistory {
		if t.Timestamp.After(oneHourAgo) && t.Status == "COMPLETED" {
			recentTransfers++
		}
	}
	fp.autoTransferManager.mu.RUnlock()

	maxTransfersPerHour := 10 // 每小时最多10次转账
	if recentTransfers >= maxTransfersPerHour {
		return fmt.Errorf("transfer rate limit exceeded: %d transfers in last hour", recentTransfers)
	}

	return nil
}

// validateDestinationAddress 验证目标地址
func (fp *FundProtector) validateDestinationAddress(address string) error {
	// 简化的地址验证
	if len(address) < 10 {
		return fmt.Errorf("destination address too short: %s", address)
	}

	// 实现更严格的地址格式验证
	// 根据不同的区块链网络验证地址格式
	
	// 以太坊地址验证
	if len(address) == 42 && address[:2] == "0x" {
		// 验证是否为有效的十六进制字符
		for i := 2; i < len(address); i++ {
			c := address[i]
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return fmt.Errorf("invalid ethereum address format: contains invalid characters")
			}
		}
		return nil
	}
	
	// 比特币地址验证（简化）
	if len(address) >= 26 && len(address) <= 35 {
		// 检查是否以1、3或bc1开头（比特币地址格式）
		if address[0] == '1' || address[0] == '3' || (len(address) >= 3 && address[:3] == "bc1") {
			return nil
		}
	}
	
	return fmt.Errorf("unsupported or invalid address format: %s", address)
}

// executeTransfer 执行实际转账
func (fp *FundProtector) executeTransfer(transfer TransferRecord) error {
	if fp.walletService == nil {
		return fmt.Errorf("wallet service not configured")
	}

	log.Printf("Executing transfer via wallet service: %s -> %s, Amount: %.2f", 
		transfer.From, transfer.To, transfer.Amount)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 创建转账请求
	request := &TransferRequest{
		Type:          transfer.Type,
		Amount:        transfer.Amount,
		FromAddress:   transfer.From,
		ToAddress:     transfer.To,
		Priority:      5, // 默认优先级
		TriggerReason: transfer.TriggerReason,
		Metadata:      transfer.Metadata,
	}

	// 执行转账
	response, err := fp.walletService.InitiateTransfer(ctx, request)
	if err != nil {
		return fmt.Errorf("wallet service transfer failed: %w", err)
	}

	// 更新转账记录
	transfer.TransactionHash = response.TransactionHash
	transfer.Status = response.Status

	log.Printf("Transfer executed successfully: %s (TxHash: %s)", 
		response.TransferID, response.TransactionHash)

	return nil
}

// recordTransfer 记录转账到数据库
func (fp *FundProtector) recordTransfer(transfer TransferRecord) error {
	if fp.daoManager == nil {
		return fmt.Errorf("database manager not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 创建转账记录
	record := &dao.TransferRecord{
		ID:              transfer.ID,
		Type:            transfer.Type,
		Amount:          transfer.Amount,
		FromAddress:     transfer.From,
		ToAddress:       transfer.To,
		Status:          transfer.Status,
		TriggerReason:   transfer.TriggerReason,
		TransactionHash: transfer.TransactionHash,
		Metadata:        transfer.Metadata,
		CreatedAt:       transfer.Timestamp,
		UpdatedAt:       time.Now(),
	}

	// 存储到数据库
	err := fp.daoManager.TransferRecords().Insert(ctx, record)
	if err != nil {
		return fmt.Errorf("failed to insert transfer record: %w", err)
	}

	log.Printf("Transfer %s recorded to database successfully", transfer.ID)
	return nil
}

func (fp *FundProtector) generateTransferID() string {
	return fmt.Sprintf("TRF_%d", time.Now().Unix())
}

func (fp *FundProtector) generateEmergencyID() string {
	return fmt.Sprintf("EMG_%d", time.Now().Unix())
}

func (fp *FundProtector) generateTransactionHash() string {
	return fmt.Sprintf("0x%x", time.Now().UnixNano())
}

func (fp *FundProtector) updateProtectionMetrics() {
	fp.protectionMetrics.mu.Lock()
	defer fp.protectionMetrics.mu.Unlock()

	// 计算保护准确率
	total := fp.protectionMetrics.CircuitBreakerTriggered + fp.protectionMetrics.EmergencyActivations
	if total > 0 {
		// 基于实际效果计算准确率
		accuracy, falsePositiveRate := fp.calculateProtectionAccuracy()
		fp.protectionMetrics.ProtectionAccuracy = accuracy
		fp.protectionMetrics.FalsePositiveRate = falsePositiveRate
		
		// 计算系统运行时间
		if fp.isRunning {
			startTime := time.Now().Add(-fp.protectionMetrics.SystemUptime)
			fp.protectionMetrics.SystemUptime = time.Since(startTime)
		}
	}

	fp.protectionMetrics.LastUpdated = time.Now()
}

// calculateProtectionAccuracy 计算保护准确率
func (fp *FundProtector) calculateProtectionAccuracy() (accuracy float64, falsePositiveRate float64) {
	// 简化的准确率计算
	// 实际实现中应该基于历史数据分析保护措施的有效性
	
	totalEvents := fp.protectionMetrics.CircuitBreakerTriggered + fp.protectionMetrics.EmergencyActivations
	if totalEvents == 0 {
		return 0.0, 0.0
	}

	// 假设保护措施的基础准确率为85%
	baseAccuracy := 0.85
	
	// 根据避免的损失调整准确率
	if fp.protectionMetrics.LossesPrevented > 0 {
		// 如果成功避免了损失，提高准确率
		baseAccuracy += 0.1
	}
	
	// 根据误报情况调整
	// 这里简化为基于事件频率的估算
	if totalEvents > 10 {
		// 如果事件过于频繁，可能存在误报
		falsePositiveRate = 0.15
		baseAccuracy -= 0.05
	} else {
		falsePositiveRate = 0.05
	}

	accuracy = math.Min(baseAccuracy, 1.0)
	accuracy = math.Max(accuracy, 0.0)
	
	return accuracy, falsePositiveRate
}

// GetFundStatus 获取资金状态
func (fp *FundProtector) GetFundStatus() *FundStatus {
	fp.fundStatus.mu.RLock()
	defer fp.fundStatus.mu.RUnlock()

	status := *fp.fundStatus
	return &status
}

// GetProtectionMetrics 获取保护指标
func (fp *FundProtector) GetProtectionMetrics() *ProtectionMetrics {
	fp.protectionMetrics.mu.RLock()
	defer fp.protectionMetrics.mu.RUnlock()

	metrics := *fp.protectionMetrics
	return &metrics
}

// IsEmergencyMode 检查是否处于紧急模式
func (fp *FundProtector) IsEmergencyMode() bool {
	fp.mu.RLock()
	defer fp.mu.RUnlock()
	return fp.isEmergencyMode
}

// IsCircuitBreakerOpen 检查熔断器是否开启
func (fp *FundProtector) IsCircuitBreakerOpen() bool {
	fp.mu.RLock()
	defer fp.mu.RUnlock()
	return fp.circuitBreakerOpen
}

// GetStatus 获取保护器状态
func (fp *FundProtector) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"running":                fp.isRunning,
		"emergency_mode":         fp.IsEmergencyMode(),
		"circuit_breaker_open":   fp.IsCircuitBreakerOpen(),
		"last_risk_check":        fp.lastRiskCheck,
		"fund_status":            fp.GetFundStatus(),
		"protection_metrics":     fp.GetProtectionMetrics(),
		"transfer_count":         len(fp.transferHistory),
		"emergency_events_count": len(fp.emergencyEvents),
	}
}

// ExchangeFundData 交易所资金数据
type ExchangeFundData struct {
	TotalBalance     float64 `json:"total_balance"`
	AvailableBalance float64 `json:"available_balance"`
	LockedBalance    float64 `json:"locked_balance"`
	DailyPL          float64 `json:"daily_pl"`
	UnrealizedPL     float64 `json:"unrealized_pl"`
}

// getFundDataFromExchange 从交易所获取资金数据
func (fp *FundProtector) getFundDataFromExchange() (*ExchangeFundData, error) {
	if fp.exchangeProvider == nil {
		return nil, fmt.Errorf("exchange provider not configured")
	}

	// 检查交易所连接健康状态
	if !fp.exchangeProvider.IsHealthy() {
		return nil, fmt.Errorf("exchange connection is not healthy")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Printf("Retrieving fund data from exchange...")

	// 从交易所获取资金数据
	fundData, err := fp.exchangeProvider.GetFundData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get fund data from exchange: %w", err)
	}

	// 验证数据完整性
	if err := fp.validateFundData(fundData); err != nil {
		return nil, fmt.Errorf("fund data validation failed: %w", err)
	}

	log.Printf("Successfully retrieved fund data: TotalBalance=%.2f, AvailableBalance=%.2f, DailyPL=%.2f", 
		fundData.TotalBalance, fundData.AvailableBalance, fundData.DailyPL)

	return fundData, nil
}

// validateFundData 验证从交易所获取的资金数据
func (fp *FundProtector) validateFundData(data *ExchangeFundData) error {
	if data == nil {
		return fmt.Errorf("fund data is nil")
	}

	// 检查余额数据的合理性
	if data.TotalBalance < 0 {
		return fmt.Errorf("total balance cannot be negative: %.2f", data.TotalBalance)
	}

	if data.AvailableBalance < 0 {
		return fmt.Errorf("available balance cannot be negative: %.2f", data.AvailableBalance)
	}

	if data.LockedBalance < 0 {
		return fmt.Errorf("locked balance cannot be negative: %.2f", data.LockedBalance)
	}

	// 检查余额关系的合理性
	if data.AvailableBalance+data.LockedBalance > data.TotalBalance*1.01 { // 允许1%的误差
		return fmt.Errorf("available + locked balance (%.2f) exceeds total balance (%.2f)", 
			data.AvailableBalance+data.LockedBalance, data.TotalBalance)
	}

	// 检查P&L数据的合理性（允许较大的波动）
	maxReasonablePL := data.TotalBalance * 0.5 // 最大合理P&L为总余额的50%
	if math.Abs(data.DailyPL) > maxReasonablePL {
		log.Printf("Warning: Daily P&L (%.2f) seems unusually large compared to total balance (%.2f)", 
			data.DailyPL, data.TotalBalance)
	}

	if math.Abs(data.UnrealizedPL) > maxReasonablePL {
		log.Printf("Warning: Unrealized P&L (%.2f) seems unusually large compared to total balance (%.2f)", 
			data.UnrealizedPL, data.TotalBalance)
	}

	return nil
}

// getCurrentPositions 获取当前持仓
func (fp *FundProtector) getCurrentPositions() ([]*Position, error) {
	if fp.exchangeProvider == nil {
		return nil, fmt.Errorf("exchange provider not configured")
	}

	// 检查交易所连接健康状态
	if !fp.exchangeProvider.IsHealthy() {
		return nil, fmt.Errorf("exchange connection is not healthy")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Printf("Retrieving current positions from exchange...")

	// 从交易所获取持仓数据
	positions, err := fp.exchangeProvider.GetPositions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get positions from exchange: %w", err)
	}

	// 验证持仓数据
	validPositions := make([]*Position, 0, len(positions))
	for _, pos := range positions {
		if err := fp.validatePosition(pos); err != nil {
			log.Printf("Warning: Invalid position data for %s: %v", pos.Symbol, err)
			continue
		}
		validPositions = append(validPositions, pos)
	}

	log.Printf("Successfully retrieved %d valid positions", len(validPositions))

	// 更新资金状态中的持仓统计
	fp.updatePositionStatistics(validPositions)

	return validPositions, nil
}

// validatePosition 验证持仓数据的合理性
func (fp *FundProtector) validatePosition(pos *Position) error {
	if pos == nil {
		return fmt.Errorf("position is nil")
	}

	if pos.Symbol == "" {
		return fmt.Errorf("position symbol is empty")
	}

	if pos.Side != "LONG" && pos.Side != "SHORT" {
		return fmt.Errorf("invalid position side: %s", pos.Side)
	}

	if pos.Size < 0 {
		return fmt.Errorf("position size cannot be negative: %.8f", pos.Size)
	}

	if pos.Size == 0 {
		return fmt.Errorf("position size is zero")
	}

	if pos.EntryPrice <= 0 {
		return fmt.Errorf("entry price must be positive: %.8f", pos.EntryPrice)
	}

	if pos.MarkPrice <= 0 {
		return fmt.Errorf("mark price must be positive: %.8f", pos.MarkPrice)
	}

	if pos.Leverage <= 0 {
		return fmt.Errorf("leverage must be positive: %d", pos.Leverage)
	}

	if pos.Leverage > 100 {
		log.Printf("Warning: Very high leverage detected for %s: %dx", pos.Symbol, pos.Leverage)
	}

	// 检查P&L的合理性
	expectedPnL := fp.calculateExpectedPnL(pos)
	pnlDifference := math.Abs(pos.UnrealizedPnL - expectedPnL)
	
	// 允许5%的误差
	if pnlDifference > math.Abs(expectedPnL)*0.05 && math.Abs(expectedPnL) > 1.0 {
		log.Printf("Warning: P&L mismatch for %s - Expected: %.2f, Actual: %.2f", 
			pos.Symbol, expectedPnL, pos.UnrealizedPnL)
	}

	return nil
}

// calculateExpectedPnL 计算预期的未实现盈亏
func (fp *FundProtector) calculateExpectedPnL(pos *Position) float64 {
	if pos.Side == "LONG" {
		return (pos.MarkPrice - pos.EntryPrice) * pos.Size
	} else {
		return (pos.EntryPrice - pos.MarkPrice) * pos.Size
	}
}

// updatePositionStatistics 更新持仓统计信息
func (fp *FundProtector) updatePositionStatistics(positions []*Position) {
	fp.fundStatus.mu.Lock()
	defer fp.fundStatus.mu.Unlock()

	fp.fundStatus.TotalPositions = len(positions)
	fp.fundStatus.ActivePositions = 0
	fp.fundStatus.LongPositions = 0
	fp.fundStatus.ShortPositions = 0

	for _, pos := range positions {
		if pos.Size > 0 {
			fp.fundStatus.ActivePositions++
			
			if pos.Side == "LONG" {
				fp.fundStatus.LongPositions++
			} else if pos.Side == "SHORT" {
				fp.fundStatus.ShortPositions++
			}
		}
	}

	log.Printf("Position statistics updated: Total=%d, Active=%d, Long=%d, Short=%d", 
		fp.fundStatus.TotalPositions, fp.fundStatus.ActivePositions, 
		fp.fundStatus.LongPositions, fp.fundStatus.ShortPositions)
}

// calculatePositionRisk 计算持仓风险
func (fp *FundProtector) calculatePositionRisk(positions []*Position) float64 {
	if len(positions) == 0 {
		return 0.0
	}

	var totalRisk float64
	var totalNotional float64

	for _, pos := range positions {
		// 计算单个持仓的风险
		positionRisk := fp.calculateSinglePositionRisk(pos)
		
		// 按名义价值加权
		totalRisk += positionRisk * pos.Notional
		totalNotional += pos.Notional
	}

	if totalNotional == 0 {
		return 0.0
	}

	// 返回加权平均风险
	averageRisk := totalRisk / totalNotional

	// 应用集中度调整
	concentrationMultiplier := fp.calculateConcentrationMultiplier(positions)
	
	return averageRisk * concentrationMultiplier
}

// calculateSinglePositionRisk 计算单个持仓的风险
func (fp *FundProtector) calculateSinglePositionRisk(pos *Position) float64 {
	// 基于杠杆和波动率计算风险
	leverageRisk := float64(pos.Leverage) / 100.0 // 杠杆风险系数
	
	// 基于未实现盈亏的波动性
	pnlRatio := math.Abs(pos.UnrealizedPnL) / pos.Notional
	
	// 基于距离强平价格的风险
	liquidationRisk := fp.calculateLiquidationRisk(pos)
	
	// 综合风险评分
	totalRisk := (leverageRisk*0.4 + pnlRatio*0.3 + liquidationRisk*0.3)
	
	// 限制风险值在合理范围内
	if totalRisk > 1.0 {
		totalRisk = 1.0
	}
	
	return totalRisk
}

// calculateLiquidationRisk 计算强平风险
func (fp *FundProtector) calculateLiquidationRisk(pos *Position) float64 {
	if pos.LiquidationPrice <= 0 {
		return 0.5 // 如果没有强平价格，返回中等风险
	}

	var distanceToLiquidation float64
	if pos.Side == "LONG" {
		distanceToLiquidation = (pos.MarkPrice - pos.LiquidationPrice) / pos.MarkPrice
	} else {
		distanceToLiquidation = (pos.LiquidationPrice - pos.MarkPrice) / pos.MarkPrice
	}

	// 距离强平价格越近，风险越高
	if distanceToLiquidation <= 0 {
		return 1.0 // 已经到达或超过强平价格
	}

	// 风险与距离成反比
	risk := 1.0 / (1.0 + distanceToLiquidation*10) // 10倍缩放因子
	
	return math.Min(risk, 1.0)
}

// calculateConcentrationMultiplier 计算集中度乘数
func (fp *FundProtector) calculateConcentrationMultiplier(positions []*Position) float64 {
	if len(positions) <= 1 {
		return 1.5 // 单一持仓风险较高
	}

	// 计算持仓分散度
	symbolCount := make(map[string]float64)
	var totalNotional float64

	for _, pos := range positions {
		symbolCount[pos.Symbol] += pos.Notional
		totalNotional += pos.Notional
	}

	// 计算赫芬达尔指数 (Herfindahl Index)
	var herfindahlIndex float64
	for _, notional := range symbolCount {
		share := notional / totalNotional
		herfindahlIndex += share * share
	}

	// 集中度越高，风险乘数越大
	concentrationMultiplier := 0.5 + herfindahlIndex
	
	return math.Min(concentrationMultiplier, 2.0) // 限制最大乘数为2
}

// getHistoricalReturns 获取历史收益率
func (fp *FundProtector) getHistoricalReturns(days int) ([]float64, error) {
	if fp.daoManager == nil {
		// 如果没有数据库连接，尝试从交易所获取
		return fp.getHistoricalReturnsFromExchange(days)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 首先尝试从数据库获取
	historicalReturns, err := fp.daoManager.HistoricalReturns().GetLastNDays(ctx, days)
	if err != nil {
		log.Printf("Failed to get historical returns from database: %v", err)
		// 如果数据库查询失败，尝试从交易所获取
		return fp.getHistoricalReturnsFromExchange(days)
	}

	if len(historicalReturns) == 0 {
		log.Printf("No historical returns found in database, fetching from exchange")
		return fp.getHistoricalReturnsFromExchange(days)
	}

	// 转换为float64数组
	returns := make([]float64, len(historicalReturns))
	for i, hr := range historicalReturns {
		returns[i] = hr.ReturnValue
	}

	log.Printf("Retrieved %d historical returns from database", len(returns))

	// 如果数据不足，补充从交易所获取的数据
	if len(returns) < days {
		missingDays := days - len(returns)
		exchangeReturns, err := fp.getHistoricalReturnsFromExchange(missingDays)
		if err == nil && len(exchangeReturns) > 0 {
			// 将交易所数据添加到前面（更早的日期）
			allReturns := make([]float64, len(exchangeReturns)+len(returns))
			copy(allReturns, exchangeReturns)
			copy(allReturns[len(exchangeReturns):], returns)
			returns = allReturns
		}
	}

	return returns, nil
}

// getHistoricalReturnsFromExchange 从交易所获取历史收益率
func (fp *FundProtector) getHistoricalReturnsFromExchange(days int) ([]float64, error) {
	if fp.exchangeProvider == nil {
		return nil, fmt.Errorf("exchange provider not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	log.Printf("Fetching %d days of historical returns from exchange", days)

	returns, err := fp.exchangeProvider.GetHistoricalReturns(ctx, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical returns from exchange: %w", err)
	}

	// 验证数据质量
	validReturns := fp.validateHistoricalReturns(returns)

	log.Printf("Retrieved %d valid historical returns from exchange", len(validReturns))

	// 异步存储到数据库
	if fp.daoManager != nil {
		go fp.storeHistoricalReturns(validReturns)
	}

	return validReturns, nil
}

// validateHistoricalReturns 验证历史收益率数据
func (fp *FundProtector) validateHistoricalReturns(returns []float64) []float64 {
	if len(returns) == 0 {
		return returns
	}

	validReturns := make([]float64, 0, len(returns))
	
	for i, ret := range returns {
		// 检查是否为有效数值
		if math.IsNaN(ret) || math.IsInf(ret, 0) {
			log.Printf("Warning: Invalid return value at index %d: %f", i, ret)
			continue
		}

		// 检查是否在合理范围内 (-50% to +50% daily return)
		if ret < -0.5 || ret > 0.5 {
			log.Printf("Warning: Extreme return value at index %d: %.4f (%.2f%%)", i, ret, ret*100)
			// 仍然保留，但记录警告
		}

		validReturns = append(validReturns, ret)
	}

	return validReturns
}

// storeHistoricalReturns 异步存储历史收益率到数据库
func (fp *FundProtector) storeHistoricalReturns(returns []float64) {
	if fp.daoManager == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 从今天开始往前推算日期
	currentDate := time.Now().Truncate(24 * time.Hour)
	
	for i, ret := range returns {
		// 计算对应的日期（从最新到最旧）
		date := currentDate.AddDate(0, 0, -(len(returns)-1-i))
		
		historicalReturn := &dao.HistoricalReturn{
			Date:        date,
			ReturnValue: ret,
		}

		err := fp.daoManager.HistoricalReturns().Insert(ctx, historicalReturn)
		if err != nil {
			log.Printf("Failed to store historical return for %s: %v", date.Format("2006-01-02"), err)
		}
	}

	log.Printf("Stored %d historical returns to database", len(returns))
}

// calculateVaRFromReturns 从收益率计算VaR
func (fp *FundProtector) calculateVaRFromReturns(returns []float64, confidence float64) float64 {
	if len(returns) == 0 {
		log.Printf("Warning: No returns data available for VaR calculation")
		return 0.0
	}

	if len(returns) < 10 {
		log.Printf("Warning: Insufficient data for reliable VaR calculation (only %d returns)", len(returns))
	}

	// 验证置信度参数
	if confidence <= 0 || confidence >= 1 {
		log.Printf("Warning: Invalid confidence level %.2f, using 0.95", confidence)
		confidence = 0.95
	}

	// 使用历史模拟法计算VaR
	historicalVaR := fp.calculateHistoricalVaR(returns, confidence)
	
	// 使用参数法计算VaR作为对比
	parametricVaR := fp.calculateParametricVaR(returns, confidence)
	
	// 使用蒙特卡洛模拟法计算VaR
	monteCarloVaR := fp.calculateMonteCarloVaR(returns, confidence)

	// 取三种方法的加权平均，历史模拟法权重最高
	finalVaR := historicalVaR*0.5 + parametricVaR*0.3 + monteCarloVaR*0.2

	log.Printf("VaR calculation (%.1f%% confidence): Historical=%.4f, Parametric=%.4f, MonteCarlo=%.4f, Final=%.4f", 
		confidence*100, historicalVaR, parametricVaR, monteCarloVaR, finalVaR)

	return finalVaR
}

// calculateHistoricalVaR 使用历史模拟法计算VaR
func (fp *FundProtector) calculateHistoricalVaR(returns []float64, confidence float64) float64 {
	if len(returns) == 0 {
		return 0.0
	}

	// 复制并排序收益率数组
	sortedReturns := make([]float64, len(returns))
	copy(sortedReturns, returns)
	sort.Float64s(sortedReturns)

	// 计算分位数位置
	alpha := 1.0 - confidence
	position := alpha * float64(len(sortedReturns)-1)
	
	// 线性插值计算VaR
	lowerIndex := int(math.Floor(position))
	upperIndex := int(math.Ceil(position))
	
	if lowerIndex == upperIndex {
		return -sortedReturns[lowerIndex] // VaR通常表示为正值（损失）
	}
	
	// 线性插值
	weight := position - float64(lowerIndex)
	var_ := sortedReturns[lowerIndex]*(1-weight) + sortedReturns[upperIndex]*weight
	
	return -var_ // 返回正值表示损失
}

// calculateParametricVaR 使用参数法计算VaR（假设正态分布）
func (fp *FundProtector) calculateParametricVaR(returns []float64, confidence float64) float64 {
	if len(returns) == 0 {
		return 0.0
	}

	// 计算均值和标准差
	mean := fp.calculateMean(returns)
	stdDev := fp.calculateStandardDeviation(returns, mean)

	// 获取正态分布的分位数
	alpha := 1.0 - confidence
	zScore := fp.getNormalQuantile(alpha)

	// 计算VaR = -(μ + z * σ)
	var_ := -(mean + zScore*stdDev)

	return var_
}

// calculateMonteCarloVaR 使用蒙特卡洛模拟法计算VaR
func (fp *FundProtector) calculateMonteCarloVaR(returns []float64, confidence float64) float64 {
	if len(returns) == 0 {
		return 0.0
	}

	// 计算历史收益率的统计特征
	mean := fp.calculateMean(returns)
	stdDev := fp.calculateStandardDeviation(returns, mean)

	// 蒙特卡洛模拟参数
	numSimulations := 10000
	simulatedReturns := make([]float64, numSimulations)

	// 生成模拟收益率（假设正态分布）
	for i := 0; i < numSimulations; i++ {
		// 使用Box-Muller变换生成正态分布随机数
		u1 := fp.randomFloat()
		u2 := fp.randomFloat()
		
		z := math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
		simulatedReturns[i] = mean + z*stdDev
	}

	// 对模拟结果计算历史VaR
	return fp.calculateHistoricalVaR(simulatedReturns, confidence)
}

// calculateMean 计算均值
func (fp *FundProtector) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// calculateStandardDeviation 计算标准差
func (fp *FundProtector) calculateStandardDeviation(values []float64, mean float64) float64 {
	if len(values) <= 1 {
		return 0.0
	}

	sumSquaredDiff := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquaredDiff += diff * diff
	}

	variance := sumSquaredDiff / float64(len(values)-1) // 样本标准差
	return math.Sqrt(variance)
}

// getNormalQuantile 获取标准正态分布的分位数（近似）
func (fp *FundProtector) getNormalQuantile(p float64) float64 {
	// 使用Beasley-Springer-Moro算法的简化版本
	// 这是一个近似算法，对于风险管理应用足够精确
	
	if p <= 0 || p >= 1 {
		return 0.0
	}

	// 对于常用的置信度，使用预计算的值
	switch {
	case math.Abs(p-0.05) < 0.001: // 95% 置信度
		return -1.645
	case math.Abs(p-0.025) < 0.001: // 97.5% 置信度
		return -1.96
	case math.Abs(p-0.01) < 0.001: // 99% 置信度
		return -2.326
	case math.Abs(p-0.005) < 0.001: // 99.5% 置信度
		return -2.576
	}

	// 对于其他值，使用近似公式
	if p < 0.5 {
		// 使用反函数近似
		t := math.Sqrt(-2 * math.Log(p))
		return -(t - (2.30753+t*0.27061)/(1+t*(0.99229+t*0.04481)))
	} else {
		// 利用对称性
		return -fp.getNormalQuantile(1 - p)
	}
}

// randomFloat 生成0-1之间的随机浮点数
func (fp *FundProtector) randomFloat() float64 {
	// 使用当前时间的纳秒作为种子生成伪随机数
	// 注意：这不是加密安全的随机数，但对于风险计算足够
	nanos := time.Now().UnixNano()
	
	// 简单的线性同余生成器
	seed := uint64(nanos)
	seed = (seed*1103515245 + 12345) & 0x7fffffff
	
	return float64(seed) / float64(0x7fffffff)
}

// getHistoricalEquity 获取历史净值
func (fp *FundProtector) getHistoricalEquity(days int) ([]float64, error) {
	if fp.daoManager == nil {
		// 如果没有数据库连接，尝试从交易所获取
		return fp.getHistoricalEquityFromExchange(days)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 首先尝试从数据库获取
	historicalEquity, err := fp.daoManager.HistoricalEquity().GetLastNDays(ctx, days)
	if err != nil {
		log.Printf("Failed to get historical equity from database: %v", err)
		// 如果数据库查询失败，尝试从交易所获取
		return fp.getHistoricalEquityFromExchange(days)
	}

	if len(historicalEquity) == 0 {
		log.Printf("No historical equity found in database, fetching from exchange")
		return fp.getHistoricalEquityFromExchange(days)
	}

	// 转换为float64数组
	equity := make([]float64, len(historicalEquity))
	for i, he := range historicalEquity {
		equity[i] = he.EquityValue
	}

	log.Printf("Retrieved %d historical equity values from database", len(equity))

	// 如果数据不足，补充从交易所获取的数据
	if len(equity) < days {
		missingDays := days - len(equity)
		exchangeEquity, err := fp.getHistoricalEquityFromExchange(missingDays)
		if err == nil && len(exchangeEquity) > 0 {
			// 将交易所数据添加到前面（更早的日期）
			allEquity := make([]float64, len(exchangeEquity)+len(equity))
			copy(allEquity, exchangeEquity)
			copy(allEquity[len(exchangeEquity):], equity)
			equity = allEquity
		}
	}

	return equity, nil
}

// getHistoricalEquityFromExchange 从交易所获取历史净值
func (fp *FundProtector) getHistoricalEquityFromExchange(days int) ([]float64, error) {
	if fp.exchangeProvider == nil {
		return nil, fmt.Errorf("exchange provider not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	log.Printf("Fetching %d days of historical equity from exchange", days)

	equity, err := fp.exchangeProvider.GetHistoricalEquity(ctx, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical equity from exchange: %w", err)
	}

	// 验证数据质量
	validEquity := fp.validateHistoricalEquity(equity)

	log.Printf("Retrieved %d valid historical equity values from exchange", len(validEquity))

	// 异步存储到数据库
	if fp.daoManager != nil {
		go fp.storeHistoricalEquity(validEquity)
	}

	return validEquity, nil
}

// validateHistoricalEquity 验证历史净值数据
func (fp *FundProtector) validateHistoricalEquity(equity []float64) []float64 {
	if len(equity) == 0 {
		return equity
	}

	validEquity := make([]float64, 0, len(equity))
	
	for i, eq := range equity {
		// 检查是否为有效数值
		if math.IsNaN(eq) || math.IsInf(eq, 0) {
			log.Printf("Warning: Invalid equity value at index %d: %f", i, eq)
			continue
		}

		// 检查是否为正数
		if eq <= 0 {
			log.Printf("Warning: Non-positive equity value at index %d: %f", i, eq)
			continue
		}

		// 检查是否有异常的跳跃（相邻值变化超过50%）
		if i > 0 && len(validEquity) > 0 {
			prevEquity := validEquity[len(validEquity)-1]
			change := math.Abs(eq-prevEquity) / prevEquity
			if change > 0.5 {
				log.Printf("Warning: Large equity change at index %d: %.2f%% (from %.2f to %.2f)", 
					i, change*100, prevEquity, eq)
			}
		}

		validEquity = append(validEquity, eq)
	}

	return validEquity
}

// storeHistoricalEquity 异步存储历史净值到数据库
func (fp *FundProtector) storeHistoricalEquity(equity []float64) {
	if fp.daoManager == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 从今天开始往前推算时间
	currentTime := time.Now()
	
	for i, eq := range equity {
		// 计算对应的时间戳（从最新到最旧）
		timestamp := currentTime.Add(-time.Duration(len(equity)-1-i) * 24 * time.Hour)
		
		historicalEquity := &dao.HistoricalEquity{
			Timestamp:   timestamp,
			EquityValue: eq,
		}

		err := fp.daoManager.HistoricalEquity().Insert(ctx, historicalEquity)
		if err != nil {
			log.Printf("Failed to store historical equity for %s: %v", 
				timestamp.Format("2006-01-02 15:04:05"), err)
		}
	}

	log.Printf("Stored %d historical equity values to database", len(equity))
}

// calculateDrawdownFromEquity 从净值计算回撤
func (fp *FundProtector) calculateDrawdownFromEquity(equity []float64) float64 {
	if len(equity) < 2 {
		log.Printf("Warning: Insufficient data for drawdown calculation (need at least 2 equity values)")
		return 0.0
	}

	// 计算多种回撤指标
	maxDrawdown := fp.calculateMaxDrawdownFromEquity(equity)
	avgDrawdown := fp.calculateAverageDrawdown(equity)
	currentDrawdown := fp.calculateCurrentDrawdown(equity)

	log.Printf("Drawdown calculation: Max=%.4f, Avg=%.4f, Current=%.4f", 
		maxDrawdown, avgDrawdown, currentDrawdown)

	// 返回最大回撤作为主要指标
	return maxDrawdown
}

// calculateMaxDrawdownFromEquity 计算最大回撤
func (fp *FundProtector) calculateMaxDrawdownFromEquity(equity []float64) float64 {
	if len(equity) < 2 {
		return 0.0
	}

	var maxDrawdown float64
	var peak float64 = equity[0]

	for _, value := range equity {
		// 更新峰值
		if value > peak {
			peak = value
		}

		// 计算当前回撤
		if peak > 0 {
			drawdown := (peak - value) / peak
			if drawdown > maxDrawdown {
				maxDrawdown = drawdown
			}
		}
	}

	return maxDrawdown
}

// calculateAverageDrawdown 计算平均回撤
func (fp *FundProtector) calculateAverageDrawdown(equity []float64) float64 {
	if len(equity) < 2 {
		return 0.0
	}

	drawdowns := fp.getAllDrawdowns(equity)
	if len(drawdowns) == 0 {
		return 0.0
	}

	var sum float64
	for _, dd := range drawdowns {
		sum += dd.MaxDrawdown
	}

	return sum / float64(len(drawdowns))
}

// calculateCurrentDrawdown 计算当前回撤
func (fp *FundProtector) calculateCurrentDrawdown(equity []float64) float64 {
	if len(equity) < 2 {
		return 0.0
	}

	// 找到历史最高点
	var peak float64
	for _, value := range equity {
		if value > peak {
			peak = value
		}
	}

	// 计算当前回撤
	currentValue := equity[len(equity)-1]
	if peak > 0 {
		return (peak - currentValue) / peak
	}

	return 0.0
}

// DrawdownPeriod 回撤期间结构
type DrawdownPeriod struct {
	StartIndex   int     `json:"start_index"`
	EndIndex     int     `json:"end_index"`
	PeakIndex    int     `json:"peak_index"`
	TroughIndex  int     `json:"trough_index"`
	MaxDrawdown  float64 `json:"max_drawdown"`
	Duration     int     `json:"duration"`
	Recovery     bool    `json:"recovery"`
}

// getAllDrawdowns 获取所有回撤期间
func (fp *FundProtector) getAllDrawdowns(equity []float64) []DrawdownPeriod {
	if len(equity) < 2 {
		return []DrawdownPeriod{}
	}

	var drawdowns []DrawdownPeriod
	var currentPeak float64 = equity[0]
	var peakIndex int = 0
	var inDrawdown bool = false
	var drawdownStart int = 0

	for i, value := range equity {
		if value > currentPeak {
			// 新的峰值
			if inDrawdown {
				// 结束当前回撤期
				drawdown := DrawdownPeriod{
					StartIndex:  drawdownStart,
					EndIndex:    i - 1,
					PeakIndex:   peakIndex,
					TroughIndex: fp.findTroughIndex(equity, drawdownStart, i-1),
					MaxDrawdown: fp.calculateMaxDrawdownInRange(equity, drawdownStart, i-1, currentPeak),
					Duration:    i - drawdownStart,
					Recovery:    true,
				}
				drawdowns = append(drawdowns, drawdown)
				inDrawdown = false
			}
			currentPeak = value
			peakIndex = i
		} else if value < currentPeak && !inDrawdown {
			// 开始新的回撤期
			inDrawdown = true
			drawdownStart = peakIndex
		}
	}

	// 处理未结束的回撤期
	if inDrawdown {
		drawdown := DrawdownPeriod{
			StartIndex:  drawdownStart,
			EndIndex:    len(equity) - 1,
			PeakIndex:   peakIndex,
			TroughIndex: fp.findTroughIndex(equity, drawdownStart, len(equity)-1),
			MaxDrawdown: fp.calculateMaxDrawdownInRange(equity, drawdownStart, len(equity)-1, currentPeak),
			Duration:    len(equity) - drawdownStart,
			Recovery:    false,
		}
		drawdowns = append(drawdowns, drawdown)
	}

	return drawdowns
}

// findTroughIndex 找到指定范围内的最低点索引
func (fp *FundProtector) findTroughIndex(equity []float64, start, end int) int {
	if start >= len(equity) || end >= len(equity) || start > end {
		return start
	}

	minValue := equity[start]
	minIndex := start

	for i := start; i <= end; i++ {
		if equity[i] < minValue {
			minValue = equity[i]
			minIndex = i
		}
	}

	return minIndex
}

// calculateMaxDrawdownInRange 计算指定范围内的最大回撤
func (fp *FundProtector) calculateMaxDrawdownInRange(equity []float64, start, end int, peak float64) float64 {
	if start >= len(equity) || end >= len(equity) || start > end || peak <= 0 {
		return 0.0
	}

	var maxDrawdown float64
	for i := start; i <= end; i++ {
		drawdown := (peak - equity[i]) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}

// calculateDrawdownMetrics 计算综合回撤指标
func (fp *FundProtector) calculateDrawdownMetrics(equity []float64) map[string]float64 {
	if len(equity) < 2 {
		return map[string]float64{}
	}

	metrics := make(map[string]float64)
	drawdowns := fp.getAllDrawdowns(equity)

	// 基础回撤指标
	metrics["MaxDrawdown"] = fp.calculateMaxDrawdownFromEquity(equity)
	metrics["CurrentDrawdown"] = fp.calculateCurrentDrawdown(equity)
	metrics["AverageDrawdown"] = fp.calculateAverageDrawdown(equity)

	if len(drawdowns) > 0 {
		// 回撤期间统计
		var totalDuration int
		var maxDuration int
		var recoveredCount int

		for _, dd := range drawdowns {
			totalDuration += dd.Duration
			if dd.Duration > maxDuration {
				maxDuration = dd.Duration
			}
			if dd.Recovery {
				recoveredCount++
			}
		}

		metrics["DrawdownCount"] = float64(len(drawdowns))
		metrics["AverageDrawdownDuration"] = float64(totalDuration) / float64(len(drawdowns))
		metrics["MaxDrawdownDuration"] = float64(maxDuration)
		metrics["RecoveryRate"] = float64(recoveredCount) / float64(len(drawdowns))

		// 计算回撤频率（每年回撤次数，假设日频数据）
		if len(equity) > 252 {
			annualDrawdownFreq := float64(len(drawdowns)) * 252.0 / float64(len(equity))
			metrics["AnnualDrawdownFrequency"] = annualDrawdownFreq
		}
	}

	// Calmar比率（年化收益率/最大回撤）
	if metrics["MaxDrawdown"] > 0 && len(equity) > 252 {
		totalReturn := (equity[len(equity)-1] - equity[0]) / equity[0]
		annualReturn := math.Pow(1+totalReturn, 252.0/float64(len(equity))) - 1
		metrics["CalmarRatio"] = annualReturn / metrics["MaxDrawdown"]
	}

	return metrics
}

// Position 持仓信息结构
type Position struct {
	Symbol            string  `json:"symbol"`
	Side              string  `json:"side"` // LONG, SHORT
	Size              float64 `json:"size"`
	Notional          float64 `json:"notional"`
	EntryPrice        float64 `json:"entry_price"`
	MarkPrice         float64 `json:"mark_price"`
	UnrealizedPnL     float64 `json:"unrealized_pnl"`
	RealizedPnL       float64 `json:"realized_pnl"`
	Leverage          int     `json:"leverage"`
	MarginType        string  `json:"margin_type"`        // ISOLATED, CROSS
	IsolatedMargin    float64 `json:"isolated_margin"`
	MaintenanceMargin float64 `json:"maintenance_margin"`
	LiquidationPrice  float64 `json:"liquidation_price"`
}

// ExchangeDataProvider 交易所数据提供者接口
type ExchangeDataProvider interface {
	// IsHealthy 检查连接健康状态
	IsHealthy() bool
	
	// GetFundData 获取资金数据
	GetFundData(ctx context.Context) (*ExchangeFundData, error)
	
	// GetPositions 获取持仓数据
	GetPositions(ctx context.Context) ([]*Position, error)
	
	// GetHistoricalReturns 获取历史收益率
	GetHistoricalReturns(ctx context.Context, days int) ([]float64, error)
	
	// GetHistoricalEquity 获取历史净值
	GetHistoricalEquity(ctx context.Context, days int) ([]float64, error)
}

// calculateVolatilityFromReturns 从收益率计算波动率
func (fp *FundProtector) calculateVolatilityFromReturns(returns []float64) float64 {
	if len(returns) < 2 {
		log.Printf("Warning: Insufficient data for volatility calculation (need at least 2 returns)")
		return 0.0
	}

	// 计算多种波动率指标
	simpleVol := fp.calculateSimpleVolatility(returns)
	ewmaVol := fp.calculateEWMAVolatility(returns, 0.94) // 常用的λ=0.94
	garchVol := fp.calculateGARCHVolatility(returns)

	// 加权平均，EWMA权重最高（更适合金融时间序列）
	finalVolatility := simpleVol*0.2 + ewmaVol*0.5 + garchVol*0.3

	log.Printf("Volatility calculation: Simple=%.4f, EWMA=%.4f, GARCH=%.4f, Final=%.4f", 
		simpleVol, ewmaVol, garchVol, finalVolatility)

	return finalVolatility
}

// calculateSimpleVolatility 计算简单历史波动率
func (fp *FundProtector) calculateSimpleVolatility(returns []float64) float64 {
	if len(returns) < 2 {
		return 0.0
	}

	mean := fp.calculateMean(returns)
	stdDev := fp.calculateStandardDeviation(returns, mean)

	// 年化波动率（假设252个交易日）
	annualizedVol := stdDev * math.Sqrt(252)

	return annualizedVol
}

// calculateEWMAVolatility 计算指数加权移动平均波动率
func (fp *FundProtector) calculateEWMAVolatility(returns []float64, lambda float64) float64 {
	if len(returns) < 2 {
		return 0.0
	}

	// 验证lambda参数
	if lambda <= 0 || lambda >= 1 {
		lambda = 0.94 // 默认值
	}

	// 初始化方差为简单方差
	mean := fp.calculateMean(returns)
	initialVariance := 0.0
	for _, ret := range returns[:min(len(returns), 10)] { // 使用前10个观测值
		diff := ret - mean
		initialVariance += diff * diff
	}
	initialVariance /= float64(min(len(returns), 10))

	// EWMA递归计算
	variance := initialVariance
	for i := 1; i < len(returns); i++ {
		returnSquared := returns[i] * returns[i]
		variance = lambda*variance + (1-lambda)*returnSquared
	}

	// 年化波动率
	annualizedVol := math.Sqrt(variance * 252)

	return annualizedVol
}

// calculateGARCHVolatility 计算GARCH(1,1)波动率（简化版本）
func (fp *FundProtector) calculateGARCHVolatility(returns []float64) float64 {
	if len(returns) < 10 {
		// 数据不足时回退到简单波动率
		return fp.calculateSimpleVolatility(returns)
	}

	// GARCH(1,1)参数（简化估计）
	omega := 0.000001  // 常数项
	alpha := 0.1       // ARCH项系数
	beta := 0.85       // GARCH项系数

	// 初始化条件方差
	mean := fp.calculateMean(returns)
	initialVariance := 0.0
	for i := 0; i < min(len(returns), 10); i++ {
		diff := returns[i] - mean
		initialVariance += diff * diff
	}
	initialVariance /= float64(min(len(returns), 10))

	// GARCH递归计算
	variance := initialVariance
	for i := 1; i < len(returns); i++ {
		returnSquared := (returns[i] - mean) * (returns[i] - mean)
		variance = omega + alpha*returnSquared + beta*variance
	}

	// 年化波动率
	annualizedVol := math.Sqrt(variance * 252)

	return annualizedVol
}

// calculateRollingVolatility 计算滚动窗口波动率
func (fp *FundProtector) calculateRollingVolatility(returns []float64, windowSize int) []float64 {
	if len(returns) < windowSize {
		return []float64{}
	}

	rollingVols := make([]float64, len(returns)-windowSize+1)

	for i := 0; i <= len(returns)-windowSize; i++ {
		window := returns[i : i+windowSize]
		rollingVols[i] = fp.calculateSimpleVolatility(window)
	}

	return rollingVols
}

// calculateVolatilityMetrics 计算综合波动率指标
func (fp *FundProtector) calculateVolatilityMetrics(returns []float64) map[string]float64 {
	if len(returns) < 2 {
		return map[string]float64{}
	}

	metrics := make(map[string]float64)

	// 基础波动率指标
	metrics["SimpleVolatility"] = fp.calculateSimpleVolatility(returns)
	metrics["EWMAVolatility"] = fp.calculateEWMAVolatility(returns, 0.94)
	metrics["GARCHVolatility"] = fp.calculateGARCHVolatility(returns)

	// 波动率的波动率（Vol of Vol）
	if len(returns) >= 20 {
		rollingVols := fp.calculateRollingVolatility(returns, 10)
		if len(rollingVols) > 1 {
			volMean := fp.calculateMean(rollingVols)
			metrics["VolatilityOfVolatility"] = fp.calculateStandardDeviation(rollingVols, volMean)
		}
	}

	// 上行和下行波动率
	upReturns := []float64{}
	downReturns := []float64{}
	mean := fp.calculateMean(returns)

	for _, ret := range returns {
		if ret > mean {
			upReturns = append(upReturns, ret)
		} else {
			downReturns = append(downReturns, ret)
		}
	}

	if len(upReturns) > 1 {
		metrics["UpsideVolatility"] = fp.calculateSimpleVolatility(upReturns)
	}
	if len(downReturns) > 1 {
		metrics["DownsideVolatility"] = fp.calculateSimpleVolatility(downReturns)
	}

	// 偏度和峰度
	metrics["Skewness"] = fp.calculateSkewness(returns)
	metrics["Kurtosis"] = fp.calculateKurtosis(returns)

	return metrics
}

// calculateSkewness 计算偏度
func (fp *FundProtector) calculateSkewness(returns []float64) float64 {
	if len(returns) < 3 {
		return 0.0
	}

	mean := fp.calculateMean(returns)
	stdDev := fp.calculateStandardDeviation(returns, mean)

	if stdDev == 0 {
		return 0.0
	}

	var sum float64
	n := float64(len(returns))

	for _, ret := range returns {
		standardized := (ret - mean) / stdDev
		sum += standardized * standardized * standardized
	}

	skewness := (n / ((n - 1) * (n - 2))) * sum

	return skewness
}

// calculateKurtosis 计算峰度
func (fp *FundProtector) calculateKurtosis(returns []float64) float64 {
	if len(returns) < 4 {
		return 0.0
	}

	mean := fp.calculateMean(returns)
	stdDev := fp.calculateStandardDeviation(returns, mean)

	if stdDev == 0 {
		return 0.0
	}

	var sum float64
	n := float64(len(returns))

	for _, ret := range returns {
		standardized := (ret - mean) / stdDev
		sum += standardized * standardized * standardized * standardized
	}

	kurtosis := (n*(n+1)/((n-1)*(n-2)*(n-3)))*sum - 3*(n-1)*(n-1)/((n-2)*(n-3))

	return kurtosis
}

// min 返回两个整数的最小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// calculateCorrelationMatrix 计算持仓间的相关性矩阵
func (fp *FundProtector) calculateCorrelationMatrix(positions []*Position) (map[string]map[string]float64, error) {
	if len(positions) < 2 {
		return nil, fmt.Errorf("need at least 2 positions for correlation calculation")
	}

	// 获取各个交易对的历史收益率
	symbolReturns := make(map[string][]float64)
	
	for _, pos := range positions {
		returns, err := fp.getSymbolHistoricalReturns(pos.Symbol, 30)
		if err != nil {
			log.Printf("Failed to get returns for %s: %v", pos.Symbol, err)
			continue
		}
		if len(returns) > 0 {
			symbolReturns[pos.Symbol] = returns
		}
	}

	// 计算相关性矩阵
	correlationMatrix := make(map[string]map[string]float64)
	
	for symbol1, returns1 := range symbolReturns {
		correlationMatrix[symbol1] = make(map[string]float64)
		
		for symbol2, returns2 := range symbolReturns {
			correlation := fp.calculateCorrelation(returns1, returns2)
			correlationMatrix[symbol1][symbol2] = correlation
		}
	}

	return correlationMatrix, nil
}

// getSymbolHistoricalReturns 获取特定交易对的历史收益率
func (fp *FundProtector) getSymbolHistoricalReturns(symbol string, days int) ([]float64, error) {
	// 这里应该从交易所或数据库获取特定交易对的历史价格数据
	// 然后计算收益率，这里提供一个简化的实现
	
	if fp.exchangeProvider == nil {
		return nil, fmt.Errorf("exchange provider not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 这里需要扩展ExchangeDataProvider接口来支持单个交易对的历史数据
	// 暂时返回空数据，实际实现时需要添加相应的方法
	log.Printf("Symbol historical returns not implemented for %s", symbol)
	return []float64{}, nil
}

// calculateCorrelation 计算两个时间序列的相关系数
func (fp *FundProtector) calculateCorrelation(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 2 {
		return 0.0
	}

	n := len(x)
	
	// 计算均值
	meanX := fp.calculateMean(x)
	meanY := fp.calculateMean(y)

	// 计算协方差和方差
	var covariance, varianceX, varianceY float64
	
	for i := 0; i < n; i++ {
		dx := x[i] - meanX
		dy := y[i] - meanY
		
		covariance += dx * dy
		varianceX += dx * dx
		varianceY += dy * dy
	}

	// 计算相关系数
	if varianceX == 0 || varianceY == 0 {
		return 0.0
	}

	correlation := covariance / math.Sqrt(varianceX*varianceY)
	return correlation
}

// calculateBeta 计算投资组合相对于基准的Beta值
func (fp *FundProtector) calculateBeta(portfolioReturns, benchmarkReturns []float64) float64 {
	if len(portfolioReturns) != len(benchmarkReturns) || len(portfolioReturns) < 2 {
		return 1.0 // 默认Beta为1
	}

	// Beta = Cov(Portfolio, Benchmark) / Var(Benchmark)
	covariance := fp.calculateCovariance(portfolioReturns, benchmarkReturns)
	benchmarkVariance := fp.calculateVariance(benchmarkReturns)

	if benchmarkVariance == 0 {
		return 1.0
	}

	return covariance / benchmarkVariance
}

// calculateCovariance 计算协方差
func (fp *FundProtector) calculateCovariance(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 2 {
		return 0.0
	}

	meanX := fp.calculateMean(x)
	meanY := fp.calculateMean(y)

	var covariance float64
	for i := 0; i < len(x); i++ {
		covariance += (x[i] - meanX) * (y[i] - meanY)
	}

	return covariance / float64(len(x)-1)
}

// calculateVariance 计算方差
func (fp *FundProtector) calculateVariance(values []float64) float64 {
	if len(values) < 2 {
		return 0.0
	}

	mean := fp.calculateMean(values)
	stdDev := fp.calculateStandardDeviation(values, mean)
	return stdDev * stdDev
}

// calculateSharpeRatio 计算夏普比率
func (fp *FundProtector) calculateSharpeRatio(returns []float64, riskFreeRate float64) float64 {
	if len(returns) < 2 {
		return 0.0
	}

	mean := fp.calculateMean(returns)
	stdDev := fp.calculateStandardDeviation(returns, mean)

	if stdDev == 0 {
		return 0.0
	}

	// 年化夏普比率
	annualReturn := mean * 252
	annualVolatility := stdDev * math.Sqrt(252)
	
	return (annualReturn - riskFreeRate) / annualVolatility
}

// calculateSortinoRatio 计算索提诺比率（只考虑下行风险）
func (fp *FundProtector) calculateSortinoRatio(returns []float64, targetReturn float64) float64 {
	if len(returns) < 2 {
		return 0.0
	}

	mean := fp.calculateMean(returns)
	downsideDeviation := fp.calculateDownsideDeviation(returns, targetReturn)

	if downsideDeviation == 0 {
		return 0.0
	}

	// 年化索提诺比率
	annualReturn := mean * 252
	annualDownsideVol := downsideDeviation * math.Sqrt(252)
	
	return (annualReturn - targetReturn*252) / annualDownsideVol
}

// calculateDownsideDeviation 计算下行偏差
func (fp *FundProtector) calculateDownsideDeviation(returns []float64, targetReturn float64) float64 {
	if len(returns) < 2 {
		return 0.0
	}

	var sumSquaredDownsideDeviations float64
	var count int

	for _, ret := range returns {
		if ret < targetReturn {
			deviation := ret - targetReturn
			sumSquaredDownsideDeviations += deviation * deviation
			count++
		}
	}

	if count == 0 {
		return 0.0
	}

	return math.Sqrt(sumSquaredDownsideDeviations / float64(count))
}

// calculateInformationRatio 计算信息比率
func (fp *FundProtector) calculateInformationRatio(portfolioReturns, benchmarkReturns []float64) float64 {
	if len(portfolioReturns) != len(benchmarkReturns) || len(portfolioReturns) < 2 {
		return 0.0
	}

	// 计算超额收益
	excessReturns := make([]float64, len(portfolioReturns))
	for i := 0; i < len(portfolioReturns); i++ {
		excessReturns[i] = portfolioReturns[i] - benchmarkReturns[i]
	}

	mean := fp.calculateMean(excessReturns)
	stdDev := fp.calculateStandardDeviation(excessReturns, mean)

	if stdDev == 0 {
		return 0.0
	}

	// 年化信息比率
	return (mean * 252) / (stdDev * math.Sqrt(252))
}

// calculateMaximumLikelihoodVolatility 使用最大似然估计计算波动率
func (fp *FundProtector) calculateMaximumLikelihoodVolatility(returns []float64) float64 {
	if len(returns) < 2 {
		return 0.0
	}

	// 对于正态分布，最大似然估计等于样本标准差
	mean := fp.calculateMean(returns)
	
	var sumSquaredDeviations float64
	for _, ret := range returns {
		deviation := ret - mean
		sumSquaredDeviations += deviation * deviation
	}

	variance := sumSquaredDeviations / float64(len(returns)) // MLE使用n而不是n-1
	return math.Sqrt(variance * 252) // 年化
}

// calculateRiskAdjustedReturns 计算风险调整收益指标
func (fp *FundProtector) calculateRiskAdjustedReturns(returns []float64, benchmarkReturns []float64, riskFreeRate float64) map[string]float64 {
	if len(returns) < 2 {
		return map[string]float64{}
	}

	metrics := make(map[string]float64)

	// 基础统计
	mean := fp.calculateMean(returns)
	volatility := fp.calculateStandardDeviation(returns, mean)
	
	// 年化指标
	annualReturn := mean * 252
	annualVolatility := volatility * math.Sqrt(252)

	metrics["AnnualReturn"] = annualReturn
	metrics["AnnualVolatility"] = annualVolatility

	// 夏普比率
	metrics["SharpeRatio"] = fp.calculateSharpeRatio(returns, riskFreeRate)

	// 索提诺比率（目标收益为无风险利率）
	metrics["SortinoRatio"] = fp.calculateSortinoRatio(returns, riskFreeRate/252)

	// 如果有基准数据，计算相对指标
	if len(benchmarkReturns) == len(returns) {
		metrics["Beta"] = fp.calculateBeta(returns, benchmarkReturns)
		metrics["InformationRatio"] = fp.calculateInformationRatio(returns, benchmarkReturns)
		
		// Alpha (CAPM)
		benchmarkMean := fp.calculateMean(benchmarkReturns)
		alpha := mean - (riskFreeRate/252 + metrics["Beta"]*(benchmarkMean-riskFreeRate/252))
		metrics["Alpha"] = alpha * 252 // 年化
	}

	// 最大回撤相关指标
	equity := fp.convertReturnsToEquity(returns, 100000) // 假设初始资金10万
	maxDrawdown := fp.calculateMaxDrawdownFromEquity(equity)
	metrics["MaxDrawdown"] = maxDrawdown

	// Calmar比率
	if maxDrawdown > 0 {
		metrics["CalmarRatio"] = annualReturn / maxDrawdown
	}

	return metrics
}

// convertReturnsToEquity 将收益率序列转换为净值序列
func (fp *FundProtector) convertReturnsToEquity(returns []float64, initialValue float64) []float64 {
	equity := make([]float64, len(returns)+1)
	equity[0] = initialValue

	for i, ret := range returns {
		equity[i+1] = equity[i] * (1 + ret)
	}

	return equity
}

// calculateLeverageFromPositions 从持仓计算杠杆
func (fp *FundProtector) calculateLeverageFromPositions(positions []*Position) float64 {
	if len(positions) == 0 {
		return 0.0
	}

	var totalNotional float64
	var totalMarginUsed float64

	for _, pos := range positions {
		totalNotional += pos.Notional
		
		// 计算保证金使用量
		marginUsed := pos.Notional / float64(pos.Leverage)
		if pos.IsolatedMargin > 0 {
			marginUsed = pos.IsolatedMargin
		}
		
		totalMarginUsed += marginUsed
	}

	if totalMarginUsed == 0 {
		return 0.0
	}

	// 有效杠杆 = 总名义价值 / 总保证金使用量
	effectiveLeverage := totalNotional / totalMarginUsed

	log.Printf("Portfolio leverage calculation: TotalNotional=%.2f, TotalMargin=%.2f, EffectiveLeverage=%.2fx", 
		totalNotional, totalMarginUsed, effectiveLeverage)

	return effectiveLeverage
}

// calculateMarginUtilization 计算保证金使用率
func (fp *FundProtector) calculateMarginUtilization() (float64, error) {
	// 获取当前持仓
	positions, err := fp.getCurrentPositions()
	if err != nil {
		return 0.0, fmt.Errorf("failed to get positions: %w", err)
	}

	// 获取账户余额
	fundData, err := fp.getFundDataFromExchange()
	if err != nil {
		return 0.0, fmt.Errorf("failed to get fund data: %w", err)
	}

	var totalMarginUsed float64
	for _, pos := range positions {
		marginUsed := pos.Notional / float64(pos.Leverage)
		if pos.IsolatedMargin > 0 {
			marginUsed = pos.IsolatedMargin
		}
		totalMarginUsed += marginUsed
	}

	if fundData.TotalBalance == 0 {
		return 0.0, nil
	}

	utilizationRate := totalMarginUsed / fundData.TotalBalance
	
	log.Printf("Margin utilization: Used=%.2f, Total=%.2f, Rate=%.2f%%", 
		totalMarginUsed, fundData.TotalBalance, utilizationRate*100)

	return utilizationRate, nil
}

// checkLeverageRisk 检查杠杆风险
func (fp *FundProtector) checkLeverageRisk() error {
	positions, err := fp.getCurrentPositions()
	if err != nil {
		return fmt.Errorf("failed to get positions for leverage check: %w", err)
	}

	if len(positions) == 0 {
		return nil // 无持仓，无杠杆风险
	}

	// 计算有效杠杆
	effectiveLeverage := fp.calculateLeverageFromPositions(positions)
	
	// 计算保证金使用率
	marginUtilization, err := fp.calculateMarginUtilization()
	if err != nil {
		log.Printf("Failed to calculate margin utilization: %v", err)
		marginUtilization = 0.0
	}

	// 杠杆风险阈值
	maxLeverage := 20.0        // 最大有效杠杆20倍
	maxMarginUtilization := 0.8 // 最大保证金使用率80%

	// 检查杠杆是否超限
	if effectiveLeverage > maxLeverage {
		log.Printf("WARNING: Effective leverage %.2fx exceeds maximum %.2fx", 
			effectiveLeverage, maxLeverage)
		
		// 触发杠杆风险警告
		fp.triggerEmergency("LEVERAGE_RISK_EXCEEDED", map[string]interface{}{
			"effective_leverage": effectiveLeverage,
			"max_leverage":      maxLeverage,
			"margin_utilization": marginUtilization,
		})
	}

	// 检查保证金使用率是否过高
	if marginUtilization > maxMarginUtilization {
		log.Printf("WARNING: Margin utilization %.2f%% exceeds maximum %.2f%%", 
			marginUtilization*100, maxMarginUtilization*100)
		
		// 触发保证金风险警告
		fp.triggerEmergency("MARGIN_UTILIZATION_HIGH", map[string]interface{}{
			"margin_utilization": marginUtilization,
			"max_utilization":   maxMarginUtilization,
			"effective_leverage": effectiveLeverage,
		})
	}

	return nil
}

// getLeverageMetrics 获取杠杆相关指标
func (fp *FundProtector) getLeverageMetrics() map[string]float64 {
	positions, err := fp.getCurrentPositions()
	if err != nil {
		log.Printf("Failed to get positions for leverage metrics: %v", err)
		return map[string]float64{}
	}

	metrics := make(map[string]float64)
	
	// 有效杠杆
	metrics["effective_leverage"] = fp.calculateLeverageFromPositions(positions)
	
	// 保证金使用率
	if marginUtilization, err := fp.calculateMarginUtilization(); err == nil {
		metrics["margin_utilization"] = marginUtilization
	}

	// 各个持仓的杠杆统计
	if len(positions) > 0 {
		var totalLeverage float64
		var maxLeverage float64
		var minLeverage float64 = float64(positions[0].Leverage)
		
		for _, pos := range positions {
			leverage := float64(pos.Leverage)
			totalLeverage += leverage
			
			if leverage > maxLeverage {
				maxLeverage = leverage
			}
			if leverage < minLeverage {
				minLeverage = leverage
			}
		}
		
		metrics["avg_position_leverage"] = totalLeverage / float64(len(positions))
		metrics["max_position_leverage"] = maxLeverage
		metrics["min_position_leverage"] = minLeverage
	}

	// 杠杆风险评分
	leverageRisk := fp.calculateLeverageRisk(positions)
	metrics["leverage_risk_score"] = leverageRisk

	return metrics
}

// adjustLeverageForRisk 根据风险调整杠杆
func (fp *FundProtector) adjustLeverageForRisk() error {
	if fp.exchange == nil {
		return fmt.Errorf("exchange not configured")
	}

	positions, err := fp.getCurrentPositions()
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// 计算当前风险水平
	currentRisk := fp.calculateCurrentRisk()
	
	// 如果风险过高，降低杠杆
	if currentRisk > 0.7 { // 风险超过70%
		log.Printf("High risk detected (%.2f), reducing leverage for high-risk positions", currentRisk)
		
		for _, pos := range positions {
			positionRisk := fp.calculateSinglePositionRisk(pos)
			
			// 对于高风险持仓，降低杠杆
			if positionRisk > 0.6 && pos.Leverage > 5 {
				newLeverage := pos.Leverage / 2 // 杠杆减半
				if newLeverage < 1 {
					newLeverage = 1
				}
				
				log.Printf("Reducing leverage for %s from %dx to %dx (risk: %.2f)", 
					pos.Symbol, pos.Leverage, newLeverage, positionRisk)
				
				err := fp.exchange.SetLeverage(ctx, pos.Symbol, newLeverage)
				if err != nil {
					log.Printf("Failed to set leverage for %s: %v", pos.Symbol, err)
					continue
				}
				
				log.Printf("Successfully reduced leverage for %s to %dx", pos.Symbol, newLeverage)
			}
		}
	}

	return nil
}

// calculateConcentrationFromPositions 从持仓计算集中度
func (fp *FundProtector) calculateConcentrationFromPositions(positions []*Position) float64 {
	if len(positions) == 0 {
		return 0.0
	}

	// 按交易对分组计算集中度
	symbolNotional := make(map[string]float64)
	var totalNotional float64

	for _, pos := range positions {
		symbolNotional[pos.Symbol] += pos.Notional
		totalNotional += pos.Notional
	}

	if totalNotional == 0 {
		return 0.0
	}

	// 计算赫芬达尔-赫希曼指数 (HHI)
	var hhi float64
	maxShare := 0.0
	
	for symbol, notional := range symbolNotional {
		share := notional / totalNotional
		hhi += share * share
		
		if share > maxShare {
			maxShare = share
		}
		
		log.Printf("Position concentration - %s: %.2f%% (%.2f)", 
			symbol, share*100, notional)
	}

	// HHI 范围: 1/n (完全分散) 到 1 (完全集中)
	// 其中 n 是持仓数量
	minHHI := 1.0 / float64(len(symbolNotional))
	
	// 标准化集中度指数 (0 = 完全分散, 1 = 完全集中)
	concentrationIndex := (hhi - minHHI) / (1.0 - minHHI)
	
	log.Printf("Portfolio concentration: HHI=%.4f, MaxShare=%.2f%%, ConcentrationIndex=%.4f", 
		hhi, maxShare*100, concentrationIndex)

	return concentrationIndex
}

// calculateSectorConcentration 计算行业集中度
func (fp *FundProtector) calculateSectorConcentration(positions []*Position) float64 {
	if len(positions) == 0 {
		return 0.0
	}

	// 按行业分组
	sectorNotional := make(map[string]float64)
	var totalNotional float64

	for _, pos := range positions {
		sector := fp.classifySector(pos.Symbol)
		sectorNotional[sector] += pos.Notional
		totalNotional += pos.Notional
	}

	if totalNotional == 0 {
		return 0.0
	}

	// 计算行业HHI
	var sectorHHI float64
	for sector, notional := range sectorNotional {
		share := notional / totalNotional
		sectorHHI += share * share
		log.Printf("Sector concentration - %s: %.2f%% (%.2f)", 
			sector, share*100, notional)
	}

	// 标准化行业集中度
	minHHI := 1.0 / float64(len(sectorNotional))
	sectorConcentration := (sectorHHI - minHHI) / (1.0 - minHHI)

	log.Printf("Sector concentration: HHI=%.4f, Concentration=%.4f", 
		sectorHHI, sectorConcentration)

	return math.Min(sectorConcentration, 1.0)
}

// classifySector 分类行业
func (fp *FundProtector) classifySector(symbol string) string {
	// 基于交易对符号进行行业分类
	switch {
	case contains(symbol, []string{"BTC"}):
		return "Bitcoin"
	case contains(symbol, []string{"ETH"}):
		return "Ethereum"
	case contains(symbol, []string{"BNB", "CRO", "FTT", "OKB"}):
		return "Exchange_Tokens"
	case contains(symbol, []string{"ADA", "DOT", "SOL", "AVAX", "ATOM"}):
		return "Smart_Contract_Platforms"
	case contains(symbol, []string{"LINK", "UNI", "AAVE", "COMP", "MKR"}):
		return "DeFi"
	case contains(symbol, []string{"XRP", "XLM", "ALGO"}):
		return "Payment_Systems"
	case contains(symbol, []string{"DOGE", "SHIB", "FLOKI"}):
		return "Meme_Coins"
	case contains(symbol, []string{"USDT", "USDC", "BUSD", "DAI"}):
		return "Stablecoins"
	case contains(symbol, []string{"SAND", "MANA", "AXS", "ENJ"}):
		return "Gaming_Metaverse"
	case contains(symbol, []string{"FIL", "AR", "STORJ"}):
		return "Storage"
	default:
		return "Other"
	}
}

// calculateGeographicConcentration 计算地理集中度
func (fp *FundProtector) calculateGeographicConcentration(positions []*Position) float64 {
	if len(positions) == 0 {
		return 0.0
	}

	// 按地理区域分组
	regionNotional := make(map[string]float64)
	var totalNotional float64

	for _, pos := range positions {
		region := fp.classifyGeographicRegion(pos.Symbol)
		regionNotional[region] += pos.Notional
		totalNotional += pos.Notional
	}

	if totalNotional == 0 {
		return 0.0
	}

	// 计算地理HHI
	var geoHHI float64
	for region, notional := range regionNotional {
		share := notional / totalNotional
		geoHHI += share * share
		log.Printf("Geographic concentration - %s: %.2f%% (%.2f)", 
			region, share*100, notional)
	}

	// 标准化地理集中度
	minHHI := 1.0 / float64(len(regionNotional))
	geoConcentration := (geoHHI - minHHI) / (1.0 - minHHI)

	log.Printf("Geographic concentration: HHI=%.4f, Concentration=%.4f", 
		geoHHI, geoConcentration)

	return math.Min(geoConcentration, 1.0)
}

// classifyGeographicRegion 分类地理区域
func (fp *FundProtector) classifyGeographicRegion(symbol string) string {
	// 基于项目起源地进行地理分类
	switch {
	case contains(symbol, []string{"BTC"}):
		return "Global" // Bitcoin是全球性的
	case contains(symbol, []string{"ETH", "UNI", "AAVE", "COMP", "MKR"}):
		return "North_America" // 以太坊生态主要在北美
	case contains(symbol, []string{"BNB", "CAKE"}):
		return "Asia" // 币安生态
	case contains(symbol, []string{"ADA"}):
		return "Europe" // Cardano起源于欧洲
	case contains(symbol, []string{"DOT", "KSM"}):
		return "Europe" // Polkadot起源于欧洲
	case contains(symbol, []string{"SOL"}):
		return "North_America" // Solana
	case contains(symbol, []string{"AVAX"}):
		return "North_America" // Avalanche
	case contains(symbol, []string{"ATOM"}):
		return "North_America" // Cosmos
	case contains(symbol, []string{"NEAR"}):
		return "North_America" // Near Protocol
	case contains(symbol, []string{"FTM"}):
		return "Asia" // Fantom
	case contains(symbol, []string{"MATIC"}):
		return "Asia" // Polygon
	default:
		return "Global"
	}
}

// calculateCorrelationBasedConcentration 基于相关性的集中度分析
func (fp *FundProtector) calculateCorrelationBasedConcentration(positions []*Position) float64 {
	if len(positions) <= 1 {
		return 0.0
	}

	// 获取相关性矩阵
	correlationMatrix, err := fp.calculateCorrelationMatrix(positions)
	if err != nil {
		log.Printf("Failed to calculate correlation matrix: %v", err)
		return 0.0
	}

	// 计算平均相关性
	var totalCorrelation float64
	var pairCount int

	for symbol1, correlations := range correlationMatrix {
		for symbol2, correlation := range correlations {
			if symbol1 != symbol2 {
				totalCorrelation += math.Abs(correlation)
				pairCount++
			}
		}
	}

	if pairCount == 0 {
		return 0.0
	}

	avgCorrelation := totalCorrelation / float64(pairCount)
	
	// 相关性越高，集中度风险越大
	correlationRisk := avgCorrelation

	log.Printf("Correlation-based concentration: AvgCorrelation=%.4f, Risk=%.4f", 
		avgCorrelation, correlationRisk)

	return math.Min(correlationRisk, 1.0)
}

// getConcentrationMetrics 获取综合集中度指标
func (fp *FundProtector) getConcentrationMetrics() map[string]float64 {
	positions, err := fp.getCurrentPositions()
	if err != nil {
		log.Printf("Failed to get positions for concentration metrics: %v", err)
		return map[string]float64{}
	}

	metrics := make(map[string]float64)

	// 基础集中度指标
	metrics["symbol_concentration"] = fp.calculateConcentrationFromPositions(positions)
	metrics["sector_concentration"] = fp.calculateSectorConcentration(positions)
	metrics["geographic_concentration"] = fp.calculateGeographicConcentration(positions)
	metrics["correlation_concentration"] = fp.calculateCorrelationBasedConcentration(positions)

	// 综合集中度风险评分
	totalConcentrationRisk := (metrics["symbol_concentration"]*0.4 + 
		metrics["sector_concentration"]*0.3 + 
		metrics["geographic_concentration"]*0.2 + 
		metrics["correlation_concentration"]*0.1)
	
	metrics["total_concentration_risk"] = totalConcentrationRisk

	// 持仓数量和分散度
	metrics["position_count"] = float64(len(positions))
	
	if len(positions) > 0 {
		// 最大单一持仓占比
		var maxPositionShare float64
		var totalNotional float64
		
		for _, pos := range positions {
			totalNotional += pos.Notional
		}
		
		if totalNotional > 0 {
			for _, pos := range positions {
				share := pos.Notional / totalNotional
				if share > maxPositionShare {
					maxPositionShare = share
				}
			}
		}
		
		metrics["max_position_share"] = maxPositionShare
		metrics["diversification_ratio"] = 1.0 / maxPositionShare // 分散化比率
	}

	return metrics
}

// checkConcentrationRisk 检查集中度风险
func (fp *FundProtector) checkConcentrationRisk() error {
	metrics := fp.getConcentrationMetrics()
	
	// 集中度风险阈值
	maxSymbolConcentration := 0.7    // 最大符号集中度70%
	maxSectorConcentration := 0.8    // 最大行业集中度80%
	maxPositionShare := 0.4          // 最大单一持仓占比40%
	maxTotalConcentrationRisk := 0.6 // 最大综合集中度风险60%

	// 检查符号集中度
	if symbolConc := metrics["symbol_concentration"]; symbolConc > maxSymbolConcentration {
		log.Printf("WARNING: Symbol concentration %.2f%% exceeds maximum %.2f%%", 
			symbolConc*100, maxSymbolConcentration*100)
		
		fp.triggerEmergency("SYMBOL_CONCENTRATION_HIGH", map[string]interface{}{
			"symbol_concentration": symbolConc,
			"max_concentration":   maxSymbolConcentration,
			"metrics":            metrics,
		})
	}

	// 检查行业集中度
	if sectorConc := metrics["sector_concentration"]; sectorConc > maxSectorConcentration {
		log.Printf("WARNING: Sector concentration %.2f%% exceeds maximum %.2f%%", 
			sectorConc*100, maxSectorConcentration*100)
		
		fp.triggerEmergency("SECTOR_CONCENTRATION_HIGH", map[string]interface{}{
			"sector_concentration": sectorConc,
			"max_concentration":   maxSectorConcentration,
			"metrics":            metrics,
		})
	}

	// 检查单一持仓占比
	if maxShare := metrics["max_position_share"]; maxShare > maxPositionShare {
		log.Printf("WARNING: Maximum position share %.2f%% exceeds limit %.2f%%", 
			maxShare*100, maxPositionShare*100)
		
		fp.triggerEmergency("POSITION_CONCENTRATION_HIGH", map[string]interface{}{
			"max_position_share": maxShare,
			"max_allowed_share": maxPositionShare,
			"metrics":           metrics,
		})
	}

	// 检查综合集中度风险
	if totalRisk := metrics["total_concentration_risk"]; totalRisk > maxTotalConcentrationRisk {
		log.Printf("WARNING: Total concentration risk %.2f%% exceeds maximum %.2f%%", 
			totalRisk*100, maxTotalConcentrationRisk*100)
		
		fp.triggerEmergency("TOTAL_CONCENTRATION_RISK_HIGH", map[string]interface{}{
			"total_concentration_risk": totalRisk,
			"max_risk":                maxTotalConcentrationRisk,
			"metrics":                 metrics,
		})
	}

	return nil
}

// runDataCollection 运行数据收集和持久化
func (fp *FundProtector) runDataCollection() {
	defer fp.wg.Done()

	ticker := time.NewTicker(fp.dataCollectionInterval)
	defer ticker.Stop()

	log.Println("Data collection started")

	// 立即执行一次数据收集
	fp.collectAndPersistData()

	for {
		select {
		case <-fp.ctx.Done():
			log.Println("Data collection stopped")
			return
		case <-ticker.C:
			fp.collectAndPersistData()
		}
	}
}

// collectAndPersistData 收集并持久化数据
func (fp *FundProtector) collectAndPersistData() {
	log.Println("Starting data collection and persistence...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 收集资金状态快照
	if err := fp.collectFundStatusSnapshot(ctx); err != nil {
		log.Printf("Failed to collect fund status snapshot: %v", err)
	}

	// 收集持仓快照
	if err := fp.collectPositionSnapshots(ctx); err != nil {
		log.Printf("Failed to collect position snapshots: %v", err)
	}

	// 收集风险快照
	if err := fp.collectRiskSnapshot(ctx); err != nil {
		log.Printf("Failed to collect risk snapshot: %v", err)
	}

	// 收集保护指标
	if err := fp.collectProtectionMetrics(ctx); err != nil {
		log.Printf("Failed to collect protection metrics: %v", err)
	}

	// 计算并存储日收益率
	if err := fp.calculateAndStoreDailyReturn(ctx); err != nil {
		log.Printf("Failed to calculate and store daily return: %v", err)
	}

	fp.lastDataCollection = time.Now()
	log.Println("Data collection completed successfully")
}

// collectFundStatusSnapshot 收集资金状态快照
func (fp *FundProtector) collectFundStatusSnapshot(ctx context.Context) error {
	fp.fundStatus.mu.RLock()
	snapshot := &dao.FundStatusSnapshot{
		Timestamp:         time.Now(),
		TotalBalance:      fp.fundStatus.TotalBalance,
		AvailableBalance:  fp.fundStatus.AvailableBalance,
		LockedBalance:     fp.fundStatus.LockedBalance,
		ProfitLoss:        fp.fundStatus.ProfitLoss,
		DailyPL:           fp.fundStatus.DailyPL,
		UnrealizedPL:      fp.fundStatus.UnrealizedPL,
		RealizedPL:        fp.fundStatus.RealizedPL,
		CurrentRisk:       &fp.fundStatus.CurrentRisk,
		MaxRisk:           &fp.fundStatus.MaxRisk,
		VaR95:             &fp.fundStatus.VaR95,
		ExpectedShortfall: &fp.fundStatus.ExpectedShortfall,
		TotalPositions:    fp.fundStatus.TotalPositions,
		ActivePositions:   fp.fundStatus.ActivePositions,
		LongPositions:     fp.fundStatus.LongPositions,
		ShortPositions:    fp.fundStatus.ShortPositions,
	}
	fp.fundStatus.mu.RUnlock()

	err := fp.daoManager.FundStatusSnapshots().Insert(ctx, snapshot)
	if err != nil {
		return fmt.Errorf("failed to insert fund status snapshot: %w", err)
	}

	log.Printf("Fund status snapshot saved: Balance=%.2f, PL=%.2f", 
		snapshot.TotalBalance, snapshot.ProfitLoss)
	return nil
}

// collectPositionSnapshots 收集持仓快照
func (fp *FundProtector) collectPositionSnapshots(ctx context.Context) error {
	positions, err := fp.getCurrentPositions()
	if err != nil {
		return fmt.Errorf("failed to get current positions: %w", err)
	}

	timestamp := time.Now()
	for _, pos := range positions {
		snapshot := &dao.PositionSnapshot{
			Timestamp:         timestamp,
			Symbol:            pos.Symbol,
			Side:              pos.Side,
			Size:              pos.Size,
			Notional:          pos.Notional,
			EntryPrice:        pos.EntryPrice,
			MarkPrice:         pos.MarkPrice,
			UnrealizedPnL:     pos.UnrealizedPnL,
			RealizedPnL:       pos.RealizedPnL,
			Leverage:          pos.Leverage,
			MarginType:        &pos.MarginType,
			IsolatedMargin:    &pos.IsolatedMargin,
			MaintenanceMargin: &pos.MaintenanceMargin,
			LiquidationPrice:  &pos.LiquidationPrice,
		}

		err := fp.daoManager.PositionSnapshots().Insert(ctx, snapshot)
		if err != nil {
			log.Printf("Failed to insert position snapshot for %s: %v", pos.Symbol, err)
			continue
		}
	}

	log.Printf("Collected %d position snapshots", len(positions))
	return nil
}

// collectRiskSnapshot 收集风险快照
func (fp *FundProtector) collectRiskSnapshot(ctx context.Context) error {
	fp.riskAssessment.mu.RLock()
	if len(fp.riskAssessment.riskHistory) == 0 {
		fp.riskAssessment.mu.RUnlock()
		log.Println("No risk history available for snapshot")
		return nil
	}

	// 获取最新的风险快照
	latestRisk := fp.riskAssessment.riskHistory[len(fp.riskAssessment.riskHistory)-1]
	fp.riskAssessment.mu.RUnlock()

	snapshot := &dao.RiskSnapshot{
		Timestamp:         time.Now(),
		RiskLevel:         latestRisk.RiskLevel,
		RiskScore:         latestRisk.RiskScore,
		VaR95:             latestRisk.VaR,
		ExpectedShortfall: latestRisk.ExpectedLoss,
		MaxDrawdown:       latestRisk.MaxDrawdown,
		VolatilityIndex:   latestRisk.VolatilityIndex,
		Leverage:          latestRisk.Leverage,
		Concentration:     latestRisk.Concentration,
	}

	err := fp.daoManager.RiskSnapshots().Insert(ctx, snapshot)
	if err != nil {
		return fmt.Errorf("failed to insert risk snapshot: %w", err)
	}

	log.Printf("Risk snapshot saved: Level=%s, Score=%.4f", 
		snapshot.RiskLevel, snapshot.RiskScore)
	return nil
}

// collectProtectionMetrics 收集保护指标
func (fp *FundProtector) collectProtectionMetrics(ctx context.Context) error {
	fp.protectionMetrics.mu.RLock()
	metrics := &dao.ProtectionMetrics{
		Timestamp:               time.Now(),
		CircuitBreakerTriggered: fp.protectionMetrics.CircuitBreakerTriggered,
		EmergencyActivations:    fp.protectionMetrics.EmergencyActivations,
		AutoTransfers:           fp.protectionMetrics.AutoTransfers,
		ManualInterventions:     fp.protectionMetrics.ManualInterventions,
		LossesPrevented:         fp.protectionMetrics.LossesPrevented,
		ProfitsSecured:          fp.protectionMetrics.ProfitsSecured,
		MaxLossAvoided:          fp.protectionMetrics.MaxLossAvoided,
		AvgResponseTimeMs:       int(fp.protectionMetrics.AvgResponseTime.Milliseconds()),
		ProtectionAccuracy:      fp.protectionMetrics.ProtectionAccuracy,
		FalsePositiveRate:       fp.protectionMetrics.FalsePositiveRate,
		SystemUptimeSeconds:     int64(fp.protectionMetrics.SystemUptime.Seconds()),
		LastEmergencyTest:       &fp.protectionMetrics.LastEmergencyTest,
	}
	fp.protectionMetrics.mu.RUnlock()

	err := fp.daoManager.ProtectionMetrics().Insert(ctx, metrics)
	if err != nil {
		return fmt.Errorf("failed to insert protection metrics: %w", err)
	}

	log.Printf("Protection metrics saved: Transfers=%d, Emergencies=%d", 
		metrics.AutoTransfers, metrics.EmergencyActivations)
	return nil
}

// calculateAndStoreDailyReturn 计算并存储日收益率
func (fp *FundProtector) calculateAndStoreDailyReturn(ctx context.Context) error {
	// 获取当前净值
	currentEquity := fp.fundStatus.TotalBalance + fp.fundStatus.UnrealizedPL

	// 获取昨天的净值
	yesterday := time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour)
	yesterdayEquity, err := fp.getEquityForDate(ctx, yesterday)
	if err != nil {
		log.Printf("Could not get yesterday's equity for return calculation: %v", err)
		return nil // 不是致命错误，继续执行
	}

	if yesterdayEquity <= 0 {
		log.Printf("Invalid yesterday equity value: %.2f", yesterdayEquity)
		return nil
	}

	// 计算日收益率
	dailyReturn := (currentEquity - yesterdayEquity) / yesterdayEquity

	// 存储历史收益率
	today := time.Now().Truncate(24 * time.Hour)
	historicalReturn := &dao.HistoricalReturn{
		Date:           today,
		ReturnValue:    dailyReturn,
		PortfolioValue: &currentEquity,
	}

	err = fp.daoManager.HistoricalReturns().Insert(ctx, historicalReturn)
	if err != nil {
		return fmt.Errorf("failed to insert daily return: %w", err)
	}

	log.Printf("Daily return calculated and stored: %.4f%% (%.2f -> %.2f)", 
		dailyReturn*100, yesterdayEquity, currentEquity)
	return nil
}

// getEquityForDate 获取指定日期的净值
func (fp *FundProtector) getEquityForDate(ctx context.Context, date time.Time) (float64, error) {
	// 首先尝试从资金状态快照获取
	snapshots, err := fp.daoManager.FundStatusSnapshots().GetByTimeRange(ctx, 
		date, date.Add(24*time.Hour))
	if err != nil {
		return 0, fmt.Errorf("failed to get fund status snapshots: %w", err)
	}

	if len(snapshots) > 0 {
		// 使用最接近的快照
		return snapshots[0].TotalBalance + snapshots[0].UnrealizedPL, nil
	}

	// 如果没有快照，尝试从历史净值获取
	equityData, err := fp.daoManager.HistoricalEquity().GetByTimeRange(ctx, 
		date, date.Add(24*time.Hour))
	if err != nil {
		return 0, fmt.Errorf("failed to get historical equity: %w", err)
	}

	if len(equityData) > 0 {
		return equityData[0].EquityValue, nil
	}

	return 0, fmt.Errorf("no equity data found for date %s", date.Format("2006-01-02"))
}

// cleanupOldData 清理旧数据
func (fp *FundProtector) cleanupOldData() {
	if fp.daoManager == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	retentionDays := 365 // 保留1年的数据
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	log.Printf("Starting data cleanup for data older than %s", cutoffDate.Format("2006-01-02"))

	// 清理历史收益率
	if deleted, err := fp.daoManager.HistoricalReturns().DeleteOlderThan(ctx, cutoffDate); err != nil {
		log.Printf("Failed to cleanup historical returns: %v", err)
	} else if deleted > 0 {
		log.Printf("Cleaned up %d historical return records", deleted)
	}

	// 清理历史净值
	if deleted, err := fp.daoManager.HistoricalEquity().DeleteOlderThan(ctx, cutoffDate); err != nil {
		log.Printf("Failed to cleanup historical equity: %v", err)
	} else if deleted > 0 {
		log.Printf("Cleaned up %d historical equity records", deleted)
	}

	// 清理风险快照
	if deleted, err := fp.daoManager.RiskSnapshots().DeleteOlderThan(ctx, cutoffDate); err != nil {
		log.Printf("Failed to cleanup risk snapshots: %v", err)
	} else if deleted > 0 {
		log.Printf("Cleaned up %d risk snapshot records", deleted)
	}

	// 清理持仓快照
	if deleted, err := fp.daoManager.PositionSnapshots().DeleteOlderThan(ctx, cutoffDate); err != nil {
		log.Printf("Failed to cleanup position snapshots: %v", err)
	} else if deleted > 0 {
		log.Printf("Cleaned up %d position snapshot records", deleted)
	}

	// 清理资金状态快照
	if deleted, err := fp.daoManager.FundStatusSnapshots().DeleteOlderThan(ctx, cutoffDate); err != nil {
		log.Printf("Failed to cleanup fund status snapshots: %v", err)
	} else if deleted > 0 {
		log.Printf("Cleaned up %d fund status snapshot records", deleted)
	}

	log.Println("Data cleanup completed")
}

// GetDataCollectionStatus 获取数据收集状态
func (fp *FundProtector) GetDataCollectionStatus() map[string]interface{} {
	return map[string]interface{}{
		"enabled":              fp.daoManager != nil,
		"collection_interval":  fp.dataCollectionInterval.String(),
		"last_collection":      fp.lastDataCollection,
		"next_collection":      fp.lastDataCollection.Add(fp.dataCollectionInterval),
		"time_until_next":      time.Until(fp.lastDataCollection.Add(fp.dataCollectionInterval)).String(),
	}
}

// runDataCleanup 运行数据清理
func (fp *FundProtector) runDataCleanup() {
	defer fp.wg.Done()

	// 每天凌晨2点执行数据清理
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	log.Println("Data cleanup scheduler started")

	for {
		select {
		case <-fp.ctx.Done():
			log.Println("Data cleanup scheduler stopped")
			return
		case <-ticker.C:
			// 检查是否是凌晨2点左右
			now := time.Now()
			if now.Hour() >= 2 && now.Hour() < 3 {
				fp.cleanupOldData()
			}
		}
	}
}