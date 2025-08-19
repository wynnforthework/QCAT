package risk

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"qcat/internal/automation/executor"
	"qcat/internal/config"
	"qcat/internal/database"
	"qcat/internal/exchange"
	"qcat/internal/exchange/position"
	"qcat/internal/exchange/risk"
	"qcat/internal/monitor"
)

// IntelligentRiskController 智能风险控制器
// 实现自适应、动态的风险控制自动化
type IntelligentRiskController struct {
	config     *config.Config
	db         *database.DB
	exchange   exchange.Exchange
	posManager *position.Manager
	riskEngine *risk.RiskEngine
	executor   *executor.RealtimeExecutor
	metrics    *monitor.MetricsCollector

	// 风险控制组件
	dynamicLimits    *DynamicLimitsManager
	volatilityModel  *VolatilityModel
	correlationModel *CorrelationModel
	liquidityModel   *LiquidityModel

	// 运行状态
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	isRunning bool
	mu        sync.RWMutex

	// 风险状态
	currentRiskLevel RiskLevel
	riskMetrics      *RealTimeRiskMetrics
	alertHistory     []*RiskAlert
}

// RiskLevel 风险等级
type RiskLevel int

const (
	RiskLevelLow RiskLevel = iota
	RiskLevelMedium
	RiskLevelHigh
	RiskLevelCritical
	RiskLevelEmergency
)

// RealTimeRiskMetrics 实时风险指标
type RealTimeRiskMetrics struct {
	PortfolioVaR      float64   `json:"portfolio_var"`
	PortfolioCVaR     float64   `json:"portfolio_cvar"`
	MaxDrawdown       float64   `json:"max_drawdown"`
	ConcentrationRisk float64   `json:"concentration_risk"`
	CorrelationRisk   float64   `json:"correlation_risk"`
	LiquidityRisk     float64   `json:"liquidity_risk"`
	LeverageRatio     float64   `json:"leverage_ratio"`
	MarginUtilization float64   `json:"margin_utilization"`
	VolatilityIndex   float64   `json:"volatility_index"`
	StressTestScore   float64   `json:"stress_test_score"`
	LastUpdated       time.Time `json:"last_updated"`
	mu                sync.RWMutex
}

// RiskAlert 风险告警
type RiskAlert struct {
	ID         string             `json:"id"`
	Level      RiskLevel          `json:"level"`
	Type       string             `json:"type"`
	Message    string             `json:"message"`
	Metrics    map[string]float64 `json:"metrics"`
	Actions    []string           `json:"actions"`
	CreatedAt  time.Time          `json:"created_at"`
	ResolvedAt *time.Time         `json:"resolved_at"`
}

// DynamicLimitsManager 动态限额管理器
type DynamicLimitsManager struct {
	baseLimits        map[string]*risk.RiskLimits
	adjustedLimits    map[string]*risk.RiskLimits
	adjustmentFactors *LimitAdjustmentFactors
	mu                sync.RWMutex
}

// LimitAdjustmentFactors 限额调整因子
type LimitAdjustmentFactors struct {
	VolatilityFactor   float64 `json:"volatility_factor"`
	CorrelationFactor  float64 `json:"correlation_factor"`
	LiquidityFactor    float64 `json:"liquidity_factor"`
	MarketRegimeFactor float64 `json:"market_regime_factor"`
	TimeOfDayFactor    float64 `json:"time_of_day_factor"`
}

// VolatilityModel 波动率模型
type VolatilityModel struct {
	historicalVol map[string]float64
	realizedVol   map[string]float64
	impliedVol    map[string]float64
	garchParams   map[string]*GARCHParams
	mu            sync.RWMutex
}

// GARCHParams GARCH模型参数
type GARCHParams struct {
	Alpha float64 `json:"alpha"`
	Beta  float64 `json:"beta"`
	Omega float64 `json:"omega"`
}

// CorrelationModel 相关性模型
type CorrelationModel struct {
	correlationMatrix map[string]map[string]float64
	ewmaDecay         float64
	lastUpdate        time.Time
	mu                sync.RWMutex
}

// LiquidityModel 流动性模型
type LiquidityModel struct {
	liquidityScores map[string]float64
	bidAskSpreads   map[string]float64
	marketDepth     map[string]float64
	impactCosts     map[string]float64
	mu              sync.RWMutex
}

// NewIntelligentRiskController 创建智能风险控制器
func NewIntelligentRiskController(
	cfg *config.Config,
	db *database.DB,
	exchange exchange.Exchange,
	posManager *position.Manager,
	riskEngine *risk.RiskEngine,
	executor *executor.RealtimeExecutor,
	metrics *monitor.MetricsCollector,
) *IntelligentRiskController {
	ctx, cancel := context.WithCancel(context.Background())

	controller := &IntelligentRiskController{
		config:           cfg,
		db:               db,
		exchange:         exchange,
		posManager:       posManager,
		riskEngine:       riskEngine,
		executor:         executor,
		metrics:          metrics,
		ctx:              ctx,
		cancel:           cancel,
		currentRiskLevel: RiskLevelLow,
		riskMetrics:      &RealTimeRiskMetrics{},
		alertHistory:     make([]*RiskAlert, 0),
	}

	// 初始化风险控制组件
	controller.initializeComponents()

	return controller
}

// Start 启动智能风险控制器
func (irc *IntelligentRiskController) Start() error {
	irc.mu.Lock()
	defer irc.mu.Unlock()

	if irc.isRunning {
		return fmt.Errorf("intelligent risk controller is already running")
	}

	log.Println("Starting intelligent risk controller...")

	// 启动风险监控
	irc.wg.Add(1)
	go irc.riskMonitoringLoop()

	// 启动动态限额调整
	irc.wg.Add(1)
	go irc.dynamicLimitsAdjustmentLoop()

	// 启动压力测试
	irc.wg.Add(1)
	go irc.stressTestingLoop()

	// 启动风险报告
	irc.wg.Add(1)
	go irc.riskReportingLoop()

	irc.isRunning = true
	log.Println("Intelligent risk controller started successfully")

	return nil
}

// Stop 停止智能风险控制器
func (irc *IntelligentRiskController) Stop() error {
	irc.mu.Lock()
	defer irc.mu.Unlock()

	if !irc.isRunning {
		return nil
	}

	log.Println("Stopping intelligent risk controller...")

	// 取消上下文
	irc.cancel()

	// 等待所有goroutine完成
	irc.wg.Wait()

	irc.isRunning = false
	log.Println("Intelligent risk controller stopped")

	return nil
}

// initializeComponents 初始化组件
func (irc *IntelligentRiskController) initializeComponents() {
	// 初始化动态限额管理器
	irc.dynamicLimits = &DynamicLimitsManager{
		baseLimits:     make(map[string]*risk.RiskLimits),
		adjustedLimits: make(map[string]*risk.RiskLimits),
		adjustmentFactors: &LimitAdjustmentFactors{
			VolatilityFactor:   1.0,
			CorrelationFactor:  1.0,
			LiquidityFactor:    1.0,
			MarketRegimeFactor: 1.0,
			TimeOfDayFactor:    1.0,
		},
	}

	// 初始化波动率模型
	irc.volatilityModel = &VolatilityModel{
		historicalVol: make(map[string]float64),
		realizedVol:   make(map[string]float64),
		impliedVol:    make(map[string]float64),
		garchParams:   make(map[string]*GARCHParams),
	}

	// 初始化相关性模型
	irc.correlationModel = &CorrelationModel{
		correlationMatrix: make(map[string]map[string]float64),
		ewmaDecay:         0.94, // EWMA衰减因子
		lastUpdate:        time.Now(),
	}

	// 初始化流动性模型
	irc.liquidityModel = &LiquidityModel{
		liquidityScores: make(map[string]float64),
		bidAskSpreads:   make(map[string]float64),
		marketDepth:     make(map[string]float64),
		impactCosts:     make(map[string]float64),
	}

	log.Println("Risk control components initialized")
}

// riskMonitoringLoop 风险监控循环
func (irc *IntelligentRiskController) riskMonitoringLoop() {
	defer irc.wg.Done()

	ticker := time.NewTicker(time.Second * 10) // 每10秒监控一次
	defer ticker.Stop()

	for {
		select {
		case <-irc.ctx.Done():
			return
		case <-ticker.C:
			if err := irc.performRiskAssessment(); err != nil {
				log.Printf("Risk assessment failed: %v", err)
			}
		}
	}
}

// performRiskAssessment 执行风险评估
func (irc *IntelligentRiskController) performRiskAssessment() error {
	ctx := context.Background()

	// 1. 更新实时风险指标
	if err := irc.updateRealTimeMetrics(ctx); err != nil {
		return fmt.Errorf("failed to update real-time metrics: %w", err)
	}

	// 2. 评估风险等级
	newRiskLevel := irc.assessRiskLevel()

	// 3. 检查是否需要触发风险控制动作
	if newRiskLevel != irc.currentRiskLevel {
		log.Printf("Risk level changed from %v to %v", irc.currentRiskLevel, newRiskLevel)
		irc.currentRiskLevel = newRiskLevel

		// 触发相应的风险控制动作
		if err := irc.triggerRiskControlActions(ctx, newRiskLevel); err != nil {
			log.Printf("Failed to trigger risk control actions: %v", err)
		}
	}

	// 4. 检查特定风险阈值
	if err := irc.checkRiskThresholds(ctx); err != nil {
		log.Printf("Risk threshold check failed: %v", err)
	}

	return nil
}

// updateRealTimeMetrics 更新实时风险指标
func (irc *IntelligentRiskController) updateRealTimeMetrics(ctx context.Context) error {
	irc.riskMetrics.mu.Lock()
	defer irc.riskMetrics.mu.Unlock()

	// 检查仓位管理器是否为nil
	if irc.posManager == nil {
		return fmt.Errorf("position manager is not initialized")
	}

	// 获取当前组合信息
	positions, err := irc.posManager.GetAllPositions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get positions: %w", err)
	}

	// 计算组合VaR
	portfolioVaR, err := irc.calculatePortfolioVaR(ctx, positions)
	if err != nil {
		log.Printf("Failed to calculate portfolio VaR: %v", err)
		portfolioVaR = 0.05 // 默认值
	}

	// 计算组合CVaR
	portfolioCVaR := portfolioVaR * 1.3 // 简化计算

	// 计算最大回撤
	maxDrawdown, err := irc.calculateMaxDrawdown(ctx)
	if err != nil {
		log.Printf("Failed to calculate max drawdown: %v", err)
		maxDrawdown = 0.02 // 默认值
	}

	// 计算集中度风险
	concentrationRisk := irc.calculateConcentrationRisk(positions)

	// 计算相关性风险
	correlationRisk := irc.calculateCorrelationRisk(positions)

	// 计算流动性风险
	liquidityRisk := irc.calculateLiquidityRisk(positions)

	// 更新指标
	irc.riskMetrics.PortfolioVaR = portfolioVaR
	irc.riskMetrics.PortfolioCVaR = portfolioCVaR
	irc.riskMetrics.MaxDrawdown = maxDrawdown
	irc.riskMetrics.ConcentrationRisk = concentrationRisk
	irc.riskMetrics.CorrelationRisk = correlationRisk
	irc.riskMetrics.LiquidityRisk = liquidityRisk
	irc.riskMetrics.LastUpdated = time.Now()

	return nil
}

// calculatePortfolioVaR 计算组合VaR
func (irc *IntelligentRiskController) calculatePortfolioVaR(ctx context.Context, positions []*exchange.Position) (float64, error) {
	if len(positions) == 0 {
		return 0.0, nil
	}

	// 简化的VaR计算（实际应该使用历史模拟或蒙特卡洛方法）
	totalValue := 0.0
	totalRisk := 0.0

	for _, pos := range positions {
		positionValue := pos.Size * pos.MarkPrice
		totalValue += math.Abs(positionValue)

		// 获取该资产的波动率
		volatility := irc.getAssetVolatility(pos.Symbol)
		positionRisk := math.Abs(positionValue) * volatility * 1.65 // 95% VaR
		totalRisk += positionRisk * positionRisk                    // 假设独立
	}

	if totalValue == 0 {
		return 0.0, nil
	}

	portfolioVaR := math.Sqrt(totalRisk) / totalValue
	return portfolioVaR, nil
}

// getAssetVolatility 获取资产波动率
func (irc *IntelligentRiskController) getAssetVolatility(symbol string) float64 {
	irc.volatilityModel.mu.RLock()
	defer irc.volatilityModel.mu.RUnlock()

	if vol, exists := irc.volatilityModel.realizedVol[symbol]; exists {
		return vol
	}

	// 默认波动率
	return 0.15
}

// calculateMaxDrawdown 计算最大回撤
func (irc *IntelligentRiskController) calculateMaxDrawdown(ctx context.Context) (float64, error) {
	// 这里应该从账户历史中计算真实的最大回撤
	// 暂时返回模拟值
	return 0.03, nil
}

// calculateConcentrationRisk 计算集中度风险
func (irc *IntelligentRiskController) calculateConcentrationRisk(positions []*exchange.Position) float64 {
	if len(positions) == 0 {
		return 0.0
	}

	totalValue := 0.0
	maxPositionValue := 0.0

	for _, pos := range positions {
		positionValue := math.Abs(pos.Size * pos.MarkPrice)
		totalValue += positionValue
		if positionValue > maxPositionValue {
			maxPositionValue = positionValue
		}
	}

	if totalValue == 0 {
		return 0.0
	}

	// 集中度风险 = 最大仓位占比
	return maxPositionValue / totalValue
}

// calculateCorrelationRisk 计算相关性风险
func (irc *IntelligentRiskController) calculateCorrelationRisk(positions []*exchange.Position) float64 {
	if len(positions) <= 1 {
		return 0.0
	}

	// 简化的相关性风险计算
	// 实际应该基于相关性矩阵计算
	avgCorrelation := 0.3 // 假设平均相关性
	return avgCorrelation * float64(len(positions)) / 10.0
}

// calculateLiquidityRisk 计算流动性风险
func (irc *IntelligentRiskController) calculateLiquidityRisk(positions []*exchange.Position) float64 {
	if len(positions) == 0 {
		return 0.0
	}

	totalRisk := 0.0
	totalValue := 0.0

	for _, pos := range positions {
		positionValue := math.Abs(pos.Size * pos.MarkPrice)
		totalValue += positionValue

		// 获取流动性评分
		liquidityScore := irc.getLiquidityScore(pos.Symbol)
		liquidityRisk := (1.0 - liquidityScore) * positionValue
		totalRisk += liquidityRisk
	}

	if totalValue == 0 {
		return 0.0
	}

	return totalRisk / totalValue
}

// getLiquidityScore 获取流动性评分
func (irc *IntelligentRiskController) getLiquidityScore(symbol string) float64 {
	irc.liquidityModel.mu.RLock()
	defer irc.liquidityModel.mu.RUnlock()

	if score, exists := irc.liquidityModel.liquidityScores[symbol]; exists {
		return score
	}

	// 默认流动性评分
	return 0.8
}

// assessRiskLevel 评估风险等级
func (irc *IntelligentRiskController) assessRiskLevel() RiskLevel {
	irc.riskMetrics.mu.RLock()
	defer irc.riskMetrics.mu.RUnlock()

	// 综合风险评分
	riskScore := 0.0

	// VaR权重30%
	if irc.riskMetrics.PortfolioVaR > 0.05 {
		riskScore += 30.0 * (irc.riskMetrics.PortfolioVaR / 0.05)
	}

	// 最大回撤权重25%
	if irc.riskMetrics.MaxDrawdown > 0.02 {
		riskScore += 25.0 * (irc.riskMetrics.MaxDrawdown / 0.02)
	}

	// 集中度风险权重20%
	if irc.riskMetrics.ConcentrationRisk > 0.3 {
		riskScore += 20.0 * (irc.riskMetrics.ConcentrationRisk / 0.3)
	}

	// 相关性风险权重15%
	if irc.riskMetrics.CorrelationRisk > 0.5 {
		riskScore += 15.0 * (irc.riskMetrics.CorrelationRisk / 0.5)
	}

	// 流动性风险权重10%
	if irc.riskMetrics.LiquidityRisk > 0.2 {
		riskScore += 10.0 * (irc.riskMetrics.LiquidityRisk / 0.2)
	}

	// 根据评分确定风险等级
	if riskScore >= 100 {
		return RiskLevelEmergency
	} else if riskScore >= 80 {
		return RiskLevelCritical
	} else if riskScore >= 60 {
		return RiskLevelHigh
	} else if riskScore >= 30 {
		return RiskLevelMedium
	} else {
		return RiskLevelLow
	}
}

// triggerRiskControlActions 触发风险控制动作
func (irc *IntelligentRiskController) triggerRiskControlActions(ctx context.Context, riskLevel RiskLevel) error {
	log.Printf("Triggering risk control actions for risk level: %v", riskLevel)

	switch riskLevel {
	case RiskLevelEmergency:
		return irc.executeEmergencyActions(ctx)
	case RiskLevelCritical:
		return irc.executeCriticalActions(ctx)
	case RiskLevelHigh:
		return irc.executeHighRiskActions(ctx)
	case RiskLevelMedium:
		return irc.executeMediumRiskActions(ctx)
	case RiskLevelLow:
		return irc.executeLowRiskActions(ctx)
	default:
		return nil
	}
}

// executeEmergencyActions 执行紧急风险控制动作
func (irc *IntelligentRiskController) executeEmergencyActions(ctx context.Context) error {
	log.Println("Executing emergency risk control actions")

	// 1. 紧急停止所有交易
	if err := irc.executor.ExecuteRiskControl("emergency_stop", map[string]interface{}{}); err != nil {
		log.Printf("Failed to execute emergency stop: %v", err)
	}

	// 2. 平掉所有仓位
	if err := irc.executor.ExecuteRiskControl("close_all_positions", map[string]interface{}{}); err != nil {
		log.Printf("Failed to close all positions: %v", err)
	}

	// 3. 发送紧急告警
	alert := &RiskAlert{
		ID:        fmt.Sprintf("emergency_%d", time.Now().Unix()),
		Level:     RiskLevelEmergency,
		Type:      "emergency_risk",
		Message:   "Emergency risk level reached - all trading stopped",
		Metrics:   irc.getRiskMetricsMap(),
		Actions:   []string{"emergency_stop", "close_all_positions"},
		CreatedAt: time.Now(),
	}

	irc.addAlert(alert)

	return nil
}

// executeCriticalActions 执行关键风险控制动作
func (irc *IntelligentRiskController) executeCriticalActions(ctx context.Context) error {
	log.Println("Executing critical risk control actions")

	// 1. 减少高风险仓位
	if err := irc.executor.ExecuteRiskControl("reduce_high_risk_positions", map[string]interface{}{
		"reduction_ratio": 0.5,
	}); err != nil {
		log.Printf("Failed to reduce high risk positions: %v", err)
	}

	// 2. 暂停新开仓
	if err := irc.executor.ExecuteRiskControl("suspend_new_positions", map[string]interface{}{}); err != nil {
		log.Printf("Failed to suspend new positions: %v", err)
	}

	// 3. 发送关键告警
	alert := &RiskAlert{
		ID:        fmt.Sprintf("critical_%d", time.Now().Unix()),
		Level:     RiskLevelCritical,
		Type:      "critical_risk",
		Message:   "Critical risk level reached - reducing positions",
		Metrics:   irc.getRiskMetricsMap(),
		Actions:   []string{"reduce_high_risk_positions", "suspend_new_positions"},
		CreatedAt: time.Now(),
	}

	irc.addAlert(alert)
	return nil
}

// executeHighRiskActions 执行高风险控制动作
func (irc *IntelligentRiskController) executeHighRiskActions(ctx context.Context) error {
	log.Println("Executing high risk control actions")

	// 1. 调整仓位大小
	if err := irc.executor.ExecuteRiskControl("adjust_position_sizes", map[string]interface{}{
		"adjustment_factor": 0.8,
	}); err != nil {
		log.Printf("Failed to adjust position sizes: %v", err)
	}

	// 2. 收紧止损
	if err := irc.executor.ExecuteRiskControl("tighten_stop_loss", map[string]interface{}{
		"tightening_factor": 0.8,
	}); err != nil {
		log.Printf("Failed to tighten stop loss: %v", err)
	}

	alert := &RiskAlert{
		ID:        fmt.Sprintf("high_%d", time.Now().Unix()),
		Level:     RiskLevelHigh,
		Type:      "high_risk",
		Message:   "High risk level - adjusting positions",
		Metrics:   irc.getRiskMetricsMap(),
		Actions:   []string{"adjust_position_sizes", "tighten_stop_loss"},
		CreatedAt: time.Now(),
	}

	irc.addAlert(alert)
	return nil
}

// executeMediumRiskActions 执行中等风险控制动作
func (irc *IntelligentRiskController) executeMediumRiskActions(ctx context.Context) error {
	log.Println("Executing medium risk control actions")

	// 1. 增加监控频率
	if err := irc.executor.ExecuteRiskControl("increase_monitoring", map[string]interface{}{
		"frequency_multiplier": 2.0,
	}); err != nil {
		log.Printf("Failed to increase monitoring: %v", err)
	}

	// 2. 调整风险限额
	if err := irc.adjustRiskLimits(ctx, 0.9); err != nil {
		log.Printf("Failed to adjust risk limits: %v", err)
	}

	return nil
}

// executeLowRiskActions 执行低风险控制动作
func (irc *IntelligentRiskController) executeLowRiskActions(ctx context.Context) error {
	log.Println("Executing low risk control actions")

	// 1. 恢复正常监控频率
	if err := irc.executor.ExecuteRiskControl("normalize_monitoring", map[string]interface{}{}); err != nil {
		log.Printf("Failed to normalize monitoring: %v", err)
	}

	// 2. 恢复正常风险限额
	if err := irc.adjustRiskLimits(ctx, 1.0); err != nil {
		log.Printf("Failed to restore risk limits: %v", err)
	}

	return nil
}

// getRiskMetricsMap 获取风险指标映射
func (irc *IntelligentRiskController) getRiskMetricsMap() map[string]float64 {
	irc.riskMetrics.mu.RLock()
	defer irc.riskMetrics.mu.RUnlock()

	return map[string]float64{
		"portfolio_var":      irc.riskMetrics.PortfolioVaR,
		"portfolio_cvar":     irc.riskMetrics.PortfolioCVaR,
		"max_drawdown":       irc.riskMetrics.MaxDrawdown,
		"concentration_risk": irc.riskMetrics.ConcentrationRisk,
		"correlation_risk":   irc.riskMetrics.CorrelationRisk,
		"liquidity_risk":     irc.riskMetrics.LiquidityRisk,
	}
}

// addAlert 添加告警
func (irc *IntelligentRiskController) addAlert(alert *RiskAlert) {
	irc.mu.Lock()
	defer irc.mu.Unlock()

	irc.alertHistory = append(irc.alertHistory, alert)

	// 保持告警历史在合理范围内
	if len(irc.alertHistory) > 1000 {
		irc.alertHistory = irc.alertHistory[100:]
	}

	log.Printf("Risk alert added: %s - %s", alert.Type, alert.Message)
}

// checkRiskThresholds 检查风险阈值
func (irc *IntelligentRiskController) checkRiskThresholds(ctx context.Context) error {
	irc.riskMetrics.mu.RLock()
	defer irc.riskMetrics.mu.RUnlock()

	// 检查VaR阈值
	if irc.riskMetrics.PortfolioVaR > 0.08 {
		log.Printf("Portfolio VaR threshold exceeded: %.4f > 0.08", irc.riskMetrics.PortfolioVaR)
		// 可以在这里触发特定的风险控制动作
	}

	// 检查集中度风险
	if irc.riskMetrics.ConcentrationRisk > 0.4 {
		log.Printf("Concentration risk threshold exceeded: %.4f > 0.4", irc.riskMetrics.ConcentrationRisk)
	}

	// 检查流动性风险
	if irc.riskMetrics.LiquidityRisk > 0.3 {
		log.Printf("Liquidity risk threshold exceeded: %.4f > 0.3", irc.riskMetrics.LiquidityRisk)
	}

	return nil
}

// adjustRiskLimits 调整风险限额
func (irc *IntelligentRiskController) adjustRiskLimits(ctx context.Context, factor float64) error {
	irc.dynamicLimits.mu.Lock()
	defer irc.dynamicLimits.mu.Unlock()

	// 调整所有风险限额
	for symbol, baseLimit := range irc.dynamicLimits.baseLimits {
		adjustedLimit := &risk.RiskLimits{
			Symbol:          baseLimit.Symbol,
			MaxPositionSize: baseLimit.MaxPositionSize * factor,
			MaxLeverage:     baseLimit.MaxLeverage,
			MaxDrawdown:     baseLimit.MaxDrawdown * factor,
			CircuitBreaker:  baseLimit.CircuitBreaker * factor,
			StopLoss:        baseLimit.StopLoss,
			TakeProfit:      baseLimit.TakeProfit,
			TrailingStop:    baseLimit.TrailingStop,
			UpdatedAt:       time.Now(),
		}

		irc.dynamicLimits.adjustedLimits[symbol] = adjustedLimit
	}

	log.Printf("Risk limits adjusted by factor: %.2f", factor)
	return nil
}

// dynamicLimitsAdjustmentLoop 动态限额调整循环
func (irc *IntelligentRiskController) dynamicLimitsAdjustmentLoop() {
	defer irc.wg.Done()

	ticker := time.NewTicker(time.Minute * 5) // 每5分钟调整一次
	defer ticker.Stop()

	for {
		select {
		case <-irc.ctx.Done():
			return
		case <-ticker.C:
			if err := irc.updateDynamicLimits(); err != nil {
				log.Printf("Failed to update dynamic limits: %v", err)
			}
		}
	}
}

// updateDynamicLimits 更新动态限额
func (irc *IntelligentRiskController) updateDynamicLimits() error {
	// 根据市场条件和风险状态动态调整限额
	adjustmentFactor := irc.calculateLimitAdjustmentFactor()
	return irc.adjustRiskLimits(context.Background(), adjustmentFactor)
}

// calculateLimitAdjustmentFactor 计算限额调整因子
func (irc *IntelligentRiskController) calculateLimitAdjustmentFactor() float64 {
	factor := 1.0

	// 根据当前风险等级调整
	switch irc.currentRiskLevel {
	case RiskLevelEmergency:
		factor = 0.2
	case RiskLevelCritical:
		factor = 0.5
	case RiskLevelHigh:
		factor = 0.7
	case RiskLevelMedium:
		factor = 0.9
	case RiskLevelLow:
		factor = 1.0
	}

	// 根据波动率调整
	avgVolatility := irc.getAverageVolatility()
	if avgVolatility > 0.2 {
		factor *= 0.8
	} else if avgVolatility < 0.1 {
		factor *= 1.1
	}

	// 确保因子在合理范围内
	if factor > 1.2 {
		factor = 1.2
	}
	if factor < 0.1 {
		factor = 0.1
	}

	return factor
}

// getAverageVolatility 获取平均波动率
func (irc *IntelligentRiskController) getAverageVolatility() float64 {
	irc.volatilityModel.mu.RLock()
	defer irc.volatilityModel.mu.RUnlock()

	if len(irc.volatilityModel.realizedVol) == 0 {
		return 0.15 // 默认波动率
	}

	total := 0.0
	count := 0
	for _, vol := range irc.volatilityModel.realizedVol {
		total += vol
		count++
	}

	return total / float64(count)
}

// stressTestingLoop 压力测试循环
func (irc *IntelligentRiskController) stressTestingLoop() {
	defer irc.wg.Done()

	ticker := time.NewTicker(time.Hour) // 每小时进行一次压力测试
	defer ticker.Stop()

	for {
		select {
		case <-irc.ctx.Done():
			return
		case <-ticker.C:
			if err := irc.performStressTest(); err != nil {
				log.Printf("Stress test failed: %v", err)
			}
		}
	}
}

// performStressTest 执行压力测试
func (irc *IntelligentRiskController) performStressTest() error {
	log.Println("Performing portfolio stress test")

	// 检查仓位管理器是否为nil
	if irc.posManager == nil {
		log.Println("Position manager is not initialized, skipping stress test")
		return nil
	}

	// 获取当前仓位
	ctx := context.Background()
	positions, err := irc.posManager.GetAllPositions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get positions for stress test: %w", err)
	}

	// 定义压力测试场景
	scenarios := []StressScenario{
		{Name: "Market Crash", PriceShock: -0.2, VolatilityShock: 2.0},
		{Name: "Flash Crash", PriceShock: -0.1, VolatilityShock: 3.0},
		{Name: "High Volatility", PriceShock: 0.0, VolatilityShock: 2.5},
		{Name: "Liquidity Crisis", PriceShock: -0.05, VolatilityShock: 1.5},
	}

	// 执行每个场景的压力测试
	for _, scenario := range scenarios {
		result := irc.runStressScenario(positions, scenario)
		log.Printf("Stress test %s: Portfolio loss %.2f%%, VaR %.4f",
			scenario.Name, result.PortfolioLoss*100, result.StressVaR)

		// 如果压力测试结果超过阈值，发出警告
		if result.PortfolioLoss > 0.15 { // 15%损失阈值
			alert := &RiskAlert{
				ID:      fmt.Sprintf("stress_%s_%d", scenario.Name, time.Now().Unix()),
				Level:   RiskLevelHigh,
				Type:    "stress_test",
				Message: fmt.Sprintf("Stress test %s shows high risk: %.2f%% loss", scenario.Name, result.PortfolioLoss*100),
				Metrics: map[string]float64{
					"portfolio_loss": result.PortfolioLoss,
					"stress_var":     result.StressVaR,
				},
				Actions:   []string{"review_positions", "consider_hedging"},
				CreatedAt: time.Now(),
			}
			irc.addAlert(alert)
		}
	}

	return nil
}

// StressScenario 压力测试场景
type StressScenario struct {
	Name            string  `json:"name"`
	PriceShock      float64 `json:"price_shock"`      // 价格冲击（百分比）
	VolatilityShock float64 `json:"volatility_shock"` // 波动率冲击（倍数）
}

// StressTestResult 压力测试结果
type StressTestResult struct {
	Scenario      StressScenario `json:"scenario"`
	PortfolioLoss float64        `json:"portfolio_loss"`
	StressVaR     float64        `json:"stress_var"`
	MaxDrawdown   float64        `json:"max_drawdown"`
}

// runStressScenario 运行压力测试场景
func (irc *IntelligentRiskController) runStressScenario(positions []*exchange.Position, scenario StressScenario) *StressTestResult {
	totalValue := 0.0
	totalLoss := 0.0

	for _, pos := range positions {
		positionValue := math.Abs(pos.Size * pos.MarkPrice)
		totalValue += positionValue

		// 计算在压力场景下的损失
		stressPrice := pos.MarkPrice * (1 + scenario.PriceShock)
		positionLoss := math.Abs(pos.Size) * (pos.MarkPrice - stressPrice)
		if pos.Size < 0 { // 空头仓位
			positionLoss = -positionLoss
		}

		if positionLoss > 0 {
			totalLoss += positionLoss
		}
	}

	portfolioLoss := 0.0
	if totalValue > 0 {
		portfolioLoss = totalLoss / totalValue
	}

	// 计算压力VaR
	stressVaR := portfolioLoss * scenario.VolatilityShock

	return &StressTestResult{
		Scenario:      scenario,
		PortfolioLoss: portfolioLoss,
		StressVaR:     stressVaR,
		MaxDrawdown:   portfolioLoss * 1.2, // 简化计算
	}
}

// riskReportingLoop 风险报告循环
func (irc *IntelligentRiskController) riskReportingLoop() {
	defer irc.wg.Done()

	ticker := time.NewTicker(time.Hour * 6) // 每6小时生成一次风险报告
	defer ticker.Stop()

	for {
		select {
		case <-irc.ctx.Done():
			return
		case <-ticker.C:
			if err := irc.generateRiskReport(); err != nil {
				log.Printf("Failed to generate risk report: %v", err)
			}
		}
	}
}

// generateRiskReport 生成风险报告
func (irc *IntelligentRiskController) generateRiskReport() error {
	log.Println("Generating risk report")

	irc.riskMetrics.mu.RLock()
	metrics := *irc.riskMetrics
	irc.riskMetrics.mu.RUnlock()

	// 生成风险报告
	report := fmt.Sprintf(`
=== 智能风险控制报告 ===
生成时间: %s
当前风险等级: %v

风险指标:
- 组合VaR: %.4f
- 组合CVaR: %.4f
- 最大回撤: %.4f
- 集中度风险: %.4f
- 相关性风险: %.4f
- 流动性风险: %.4f

告警历史: %d 条
最近告警: %s

建议:
%s
`,
		time.Now().Format("2006-01-02 15:04:05"),
		irc.currentRiskLevel,
		metrics.PortfolioVaR,
		metrics.PortfolioCVaR,
		metrics.MaxDrawdown,
		metrics.ConcentrationRisk,
		metrics.CorrelationRisk,
		metrics.LiquidityRisk,
		len(irc.alertHistory),
		irc.getLatestAlertSummary(),
		irc.generateRiskRecommendations(),
	)

	log.Println(report)

	// 这里可以将报告保存到数据库或发送给相关人员
	return nil
}

// getLatestAlertSummary 获取最新告警摘要
func (irc *IntelligentRiskController) getLatestAlertSummary() string {
	irc.mu.RLock()
	defer irc.mu.RUnlock()

	if len(irc.alertHistory) == 0 {
		return "无"
	}

	latest := irc.alertHistory[len(irc.alertHistory)-1]
	return fmt.Sprintf("%s (%s)", latest.Message, latest.CreatedAt.Format("15:04:05"))
}

// generateRiskRecommendations 生成风险建议
func (irc *IntelligentRiskController) generateRiskRecommendations() string {
	recommendations := []string{}

	irc.riskMetrics.mu.RLock()
	defer irc.riskMetrics.mu.RUnlock()

	if irc.riskMetrics.PortfolioVaR > 0.05 {
		recommendations = append(recommendations, "- 组合VaR偏高，建议减少高风险仓位")
	}

	if irc.riskMetrics.ConcentrationRisk > 0.3 {
		recommendations = append(recommendations, "- 仓位过于集中，建议分散投资")
	}

	if irc.riskMetrics.LiquidityRisk > 0.2 {
		recommendations = append(recommendations, "- 流动性风险较高，建议增加高流动性资产")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "- 当前风险水平正常，继续监控")
	}

	result := ""
	for _, rec := range recommendations {
		result += rec + "\n"
	}

	return result
}
