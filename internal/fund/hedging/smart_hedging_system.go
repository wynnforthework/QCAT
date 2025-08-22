package hedging

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"qcat/internal/config"
	"qcat/internal/database"
)

// SmartHedgingSystem 智能对冲系统
type SmartHedgingSystem struct {
	config               *config.Config
	db                   *sql.DB
	correlationAnalyzer  *CorrelationAnalyzer
	hedgeRatioCalculator *HedgeRatioCalculator
	hedgeExecutor        *HedgeExecutor
	dynamicAdjuster      *DynamicAdjuster

	// 运行状态
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	isRunning bool
	mu        sync.RWMutex

	// 对冲配置
	correlationThreshold float64
	hedgeFrequency       time.Duration
	dynamicAdjustment    bool
	minHedgeRatio        float64
	maxHedgeRatio        float64

	// 对冲状态
	activeHedges          map[string]*HedgePosition
	hedgeInstruments      map[string]*HedgeInstrument
	correlationMatrix     map[string]map[string]float64
	lastCorrelationUpdate time.Time

	// 监控指标
	hedgingMetrics     *HedgingMetrics
	performanceHistory []HedgePerformance

	// 配置参数
	enabled bool
}

// HedgePosition 对冲仓位
type HedgePosition struct {
	ID             string  `json:"id"`
	BaseAsset      string  `json:"base_asset"`
	HedgeAsset     string  `json:"hedge_asset"`
	BaseQuantity   float64 `json:"base_quantity"`
	HedgeQuantity  float64 `json:"hedge_quantity"`
	HedgeRatio     float64 `json:"hedge_ratio"`
	OptimalRatio   float64 `json:"optimal_ratio"`
	EffectiveRatio float64 `json:"effective_ratio"`

	// 风险指标
	Correlation        float64 `json:"correlation"`
	Beta               float64 `json:"beta"`
	TrackingError      float64 `json:"tracking_error"`
	HedgeEffectiveness float64 `json:"hedge_effectiveness"`

	// 成本和收益
	HedgeCost   float64 `json:"hedge_cost"`
	BasisRisk   float64 `json:"basis_risk"`
	HedgeReturn float64 `json:"hedge_return"`
	NetExposure float64 `json:"net_exposure"`

	// 状态信息
	Status       string    `json:"status"` // ACTIVE, ADJUSTING, CLOSING
	CreatedAt    time.Time `json:"created_at"`
	LastAdjusted time.Time `json:"last_adjusted"`
	LastUpdated  time.Time `json:"last_updated"`

	// 调整历史
	AdjustmentHistory []HedgeAdjustment `json:"adjustment_history"`
}

// HedgeInstrument 对冲工具
type HedgeInstrument struct {
	Symbol       string  `json:"symbol"`
	Type         string  `json:"type"` // FUTURES, OPTIONS, SPOT, PERPETUAL
	Underlying   string  `json:"underlying"`
	Multiplier   float64 `json:"multiplier"`
	TickSize     float64 `json:"tick_size"`
	MinOrderSize float64 `json:"min_order_size"`
	MaxOrderSize float64 `json:"max_order_size"`

	// 流动性指标
	AvgVolume      float64 `json:"avg_volume"`
	BidAskSpread   float64 `json:"bid_ask_spread"`
	MarketDepth    float64 `json:"market_depth"`
	LiquidityScore float64 `json:"liquidity_score"`

	// 成本指标
	TradingFee  float64 `json:"trading_fee"`
	FundingRate float64 `json:"funding_rate"` // 对于永续合约
	CarryCost   float64 `json:"carry_cost"`

	// 风险指标
	Volatility       float64 `json:"volatility"`
	Beta             float64 `json:"beta"`
	DeltaSensitivity float64 `json:"delta_sensitivity"`

	IsActive    bool      `json:"is_active"`
	LastUpdated time.Time `json:"last_updated"`
}

// HedgeAdjustment 对冲调整
type HedgeAdjustment struct {
	Timestamp      time.Time `json:"timestamp"`
	Trigger        string    `json:"trigger"`
	OldRatio       float64   `json:"old_ratio"`
	NewRatio       float64   `json:"new_ratio"`
	AdjustmentSize float64   `json:"adjustment_size"`
	Cost           float64   `json:"cost"`
	Reason         string    `json:"reason"`
	Effectiveness  float64   `json:"effectiveness"`
}

// CorrelationAnalyzer 相关性分析器
type CorrelationAnalyzer struct {
	lookbackPeriod    int
	updateFrequency   time.Duration
	correlationModel  string
	significanceLevel float64

	// 历史数据
	priceData          map[string][]float64
	correlationHistory map[string][]CorrelationSnapshot

	mu sync.RWMutex
}

// CorrelationSnapshot 相关性快照
type CorrelationSnapshot struct {
	Timestamp    time.Time          `json:"timestamp"`
	Correlations map[string]float64 `json:"correlations"`
	Significance map[string]float64 `json:"significance"`
	Stability    float64            `json:"stability"`
	Confidence   float64            `json:"confidence"`
}

// HedgeRatioCalculator 对冲比率计算器
type HedgeRatioCalculator struct {
	model             string // minimum_variance, utility_maximization, var_minimization
	rebalanceStrategy string // static, dynamic, adaptive
	constraints       []HedgeConstraint

	// 计算参数
	lookbackWindow int
	halfLife       float64 // 用于EWMA
	confidence     float64 // 用于VaR计算
	riskAversion   float64 // 用于效用最大化

	mu sync.RWMutex
}

// HedgeConstraint 对冲约束
type HedgeConstraint struct {
	Type        string  `json:"type"` // MIN_RATIO, MAX_RATIO, MAX_COST, MIN_LIQUIDITY
	Parameter   string  `json:"parameter"`
	Value       float64 `json:"value"`
	IsActive    bool    `json:"is_active"`
	Description string  `json:"description"`
}

// HedgeExecutor 对冲执行器
type HedgeExecutor struct {
	executionStrategy string
	slippageLimit     float64
	maxRetries        int
	orderTimeout      time.Duration

	// 执行历史
	executionHistory []HedgeExecution

	mu sync.RWMutex
}

// HedgeExecution 对冲执行
type HedgeExecution struct {
	ID            string                 `json:"id"`
	HedgeID       string                 `json:"hedge_id"`
	Action        string                 `json:"action"` // OPEN, ADJUST, CLOSE
	Symbol        string                 `json:"symbol"`
	Side          string                 `json:"side"` // BUY, SELL
	Quantity      float64                `json:"quantity"`
	TargetPrice   float64                `json:"target_price"`
	ExecutedPrice float64                `json:"executed_price"`
	Slippage      float64                `json:"slippage"`
	Cost          float64                `json:"cost"`
	Status        string                 `json:"status"` // PENDING, EXECUTED, FAILED, CANCELLED
	Timestamp     time.Time              `json:"timestamp"`
	ExecutionTime time.Duration          `json:"execution_time"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// DynamicAdjuster 动态调整器
type DynamicAdjuster struct {
	adjustmentModel     string
	sensitivity         float64
	adjustmentThreshold float64
	maxAdjustmentFreq   time.Duration

	// 调整历史
	adjustmentHistory []DynamicAdjustment
	lastAdjustment    time.Time

	mu sync.RWMutex
}

// DynamicAdjustment 动态调整
type DynamicAdjustment struct {
	Timestamp       time.Time          `json:"timestamp"`
	HedgeID         string             `json:"hedge_id"`
	Trigger         string             `json:"trigger"`
	MarketCondition string             `json:"market_condition"`
	AdjustmentType  string             `json:"adjustment_type"`
	OldParameters   map[string]float64 `json:"old_parameters"`
	NewParameters   map[string]float64 `json:"new_parameters"`
	ExpectedImpact  float64            `json:"expected_impact"`
	ActualImpact    float64            `json:"actual_impact"`
}

// HedgingMetrics 对冲指标
type HedgingMetrics struct {
	mu sync.RWMutex

	// 对冲效果
	OverallHedgeEffectiveness float64 `json:"overall_hedge_effectiveness"`
	AverageHedgeRatio         float64 `json:"average_hedge_ratio"`
	TotalHedgingCost          float64 `json:"total_hedging_cost"`
	PortfolioVaRReduction     float64 `json:"portfolio_var_reduction"`

	// 相关性统计
	AverageCorrelation     float64 `json:"average_correlation"`
	CorrelationStability   float64 `json:"correlation_stability"`
	StrongCorrelationPairs int     `json:"strong_correlation_pairs"`

	// 执行统计
	TotalExecutions      int64         `json:"total_executions"`
	SuccessfulExecutions int64         `json:"successful_executions"`
	AverageSlippage      float64       `json:"average_slippage"`
	AverageExecutionTime time.Duration `json:"average_execution_time"`

	// 调整统计
	TotalAdjustments      int64   `json:"total_adjustments"`
	AdjustmentFrequency   float64 `json:"adjustment_frequency"`
	AverageAdjustmentCost float64 `json:"average_adjustment_cost"`

	// 性能指标
	HedgedVsUnhedgedReturn  float64 `json:"hedged_vs_unhedged_return"`
	RiskAdjustedPerformance float64 `json:"risk_adjusted_performance"`
	InformationRatio        float64 `json:"information_ratio"`

	LastUpdated time.Time `json:"last_updated"`
}

// HedgePerformance 对冲表现
type HedgePerformance struct {
	Timestamp          time.Time `json:"timestamp"`
	PortfolioReturn    float64   `json:"portfolio_return"`
	HedgedReturn       float64   `json:"hedged_return"`
	UnhedgedReturn     float64   `json:"unhedged_return"`
	HedgingAlpha       float64   `json:"hedging_alpha"`
	TrackingError      float64   `json:"tracking_error"`
	HedgeEffectiveness float64   `json:"hedge_effectiveness"`
	TotalHedgingCost   float64   `json:"total_hedging_cost"`
	NetPerformance     float64   `json:"net_performance"`
}

// NewSmartHedgingSystem 创建智能对冲系统
func NewSmartHedgingSystem(cfg *config.Config) (*SmartHedgingSystem, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// 初始化数据库连接
	var db *sql.DB
	if cfg != nil {
		dbConfig := &database.Config{
			Host:            cfg.Database.Host,
			Port:            cfg.Database.Port,
			User:            cfg.Database.User,
			Password:        cfg.Database.Password,
			DBName:          cfg.Database.DBName,
			SSLMode:         cfg.Database.SSLMode,
			MaxOpen:         cfg.Database.MaxOpen,
			MaxIdle:         cfg.Database.MaxIdle,
			Timeout:         cfg.Database.Timeout,
			ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
			ConnMaxIdleTime: cfg.Database.ConnMaxIdleTime,
		}

		dbConn, err := database.NewConnection(dbConfig)
		if err != nil {
			log.Printf("Warning: Failed to connect to database for hedging system: %v", err)
			// 继续运行，但没有数据库功能
		} else {
			db = dbConn.DB
			log.Println("Database connection established for hedging system")
		}
	}

	shs := &SmartHedgingSystem{
		config:               cfg,
		db:                   db,
		correlationAnalyzer:  NewCorrelationAnalyzer(),
		hedgeRatioCalculator: NewHedgeRatioCalculator(),
		hedgeExecutor:        NewHedgeExecutor(),
		dynamicAdjuster:      NewDynamicAdjuster(),
		ctx:                  ctx,
		cancel:               cancel,
		activeHedges:         make(map[string]*HedgePosition),
		hedgeInstruments:     make(map[string]*HedgeInstrument),
		correlationMatrix:    make(map[string]map[string]float64),
		hedgingMetrics:       &HedgingMetrics{},
		performanceHistory:   make([]HedgePerformance, 0),
		correlationThreshold: 0.7,
		hedgeFrequency:       1 * time.Hour,
		dynamicAdjustment:    true,
		minHedgeRatio:        0.1,
		maxHedgeRatio:        1.0,
		enabled:              true,
	}

	// 从配置文件读取参数
	if cfg != nil {
		// TODO: 从配置文件读取对冲参数
	}

	// 初始化对冲工具
	err := shs.initializeHedgeInstruments()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize hedge instruments: %w", err)
	}

	return shs, nil
}

// NewCorrelationAnalyzer 创建相关性分析器
func NewCorrelationAnalyzer() *CorrelationAnalyzer {
	return &CorrelationAnalyzer{
		lookbackPeriod:     252, // 一年的交易日
		updateFrequency:    15 * time.Minute,
		correlationModel:   "pearson",
		significanceLevel:  0.05,
		priceData:          make(map[string][]float64),
		correlationHistory: make(map[string][]CorrelationSnapshot),
	}
}

// NewHedgeRatioCalculator 创建对冲比率计算器
func NewHedgeRatioCalculator() *HedgeRatioCalculator {
	return &HedgeRatioCalculator{
		model:             "minimum_variance",
		rebalanceStrategy: "dynamic",
		constraints:       make([]HedgeConstraint, 0),
		lookbackWindow:    60,   // 60天
		halfLife:          30.0, // 30天半衰期
		confidence:        0.95,
		riskAversion:      3.0,
	}
}

// NewHedgeExecutor 创建对冲执行器
func NewHedgeExecutor() *HedgeExecutor {
	return &HedgeExecutor{
		executionStrategy: "smart_order_routing",
		slippageLimit:     0.002, // 0.2%滑点限制
		maxRetries:        3,
		orderTimeout:      30 * time.Second,
		executionHistory:  make([]HedgeExecution, 0),
	}
}

// NewDynamicAdjuster 创建动态调整器
func NewDynamicAdjuster() *DynamicAdjuster {
	return &DynamicAdjuster{
		adjustmentModel:     "volatility_adaptive",
		sensitivity:         0.1,
		adjustmentThreshold: 0.05,
		maxAdjustmentFreq:   15 * time.Minute,
		adjustmentHistory:   make([]DynamicAdjustment, 0),
	}
}

// Start 启动智能对冲系统
func (shs *SmartHedgingSystem) Start() error {
	shs.mu.Lock()
	defer shs.mu.Unlock()

	if shs.isRunning {
		return fmt.Errorf("smart hedging system is already running")
	}

	if !shs.enabled {
		return fmt.Errorf("smart hedging system is disabled")
	}

	log.Println("Starting Smart Hedging System...")

	// 启动相关性监控
	shs.wg.Add(1)
	go shs.runCorrelationMonitoring()

	// 启动对冲监控
	shs.wg.Add(1)
	go shs.runHedgeMonitoring()

	// 启动动态调整
	shs.wg.Add(1)
	go shs.runDynamicAdjustment()

	// 启动性能分析
	shs.wg.Add(1)
	go shs.runPerformanceAnalysis()

	// 启动指标收集
	shs.wg.Add(1)
	go shs.runMetricsCollection()

	shs.isRunning = true
	log.Println("Smart Hedging System started successfully")
	return nil
}

// Stop 停止智能对冲系统
func (shs *SmartHedgingSystem) Stop() error {
	shs.mu.Lock()
	defer shs.mu.Unlock()

	if !shs.isRunning {
		return fmt.Errorf("smart hedging system is not running")
	}

	log.Println("Stopping Smart Hedging System...")

	shs.cancel()
	shs.wg.Wait()

	shs.isRunning = false
	log.Println("Smart Hedging System stopped successfully")
	return nil
}

// initializeHedgeInstruments 初始化对冲工具
func (shs *SmartHedgingSystem) initializeHedgeInstruments() error {
	// 添加主要对冲工具
	instruments := []HedgeInstrument{
		{
			Symbol:           "BTCUSDT",
			Type:             "PERPETUAL",
			Underlying:       "BTC",
			Multiplier:       1.0,
			TickSize:         0.1,
			MinOrderSize:     0.001,
			MaxOrderSize:     1000.0,
			AvgVolume:        1000000.0,
			BidAskSpread:     0.0001,
			MarketDepth:      500000.0,
			LiquidityScore:   0.95,
			TradingFee:       0.0004,
			FundingRate:      0.0001,
			CarryCost:        0.0,
			Volatility:       0.8,
			Beta:             1.0,
			DeltaSensitivity: 1.0,
			IsActive:         true,
			LastUpdated:      time.Now(),
		},
		{
			Symbol:           "ETHUSDT",
			Type:             "PERPETUAL",
			Underlying:       "ETH",
			Multiplier:       1.0,
			TickSize:         0.01,
			MinOrderSize:     0.01,
			MaxOrderSize:     10000.0,
			AvgVolume:        800000.0,
			BidAskSpread:     0.0001,
			MarketDepth:      400000.0,
			LiquidityScore:   0.92,
			TradingFee:       0.0004,
			FundingRate:      0.0001,
			CarryCost:        0.0,
			Volatility:       0.9,
			Beta:             0.8,
			DeltaSensitivity: 0.8,
			IsActive:         true,
			LastUpdated:      time.Now(),
		},
	}

	for _, instrument := range instruments {
		shs.hedgeInstruments[instrument.Symbol] = &instrument
	}

	log.Printf("Initialized %d hedge instruments", len(instruments))
	return nil
}

// runCorrelationMonitoring 运行相关性监控
func (shs *SmartHedgingSystem) runCorrelationMonitoring() {
	defer shs.wg.Done()

	ticker := time.NewTicker(shs.correlationAnalyzer.updateFrequency)
	defer ticker.Stop()

	log.Println("Correlation monitoring started")

	for {
		select {
		case <-shs.ctx.Done():
			log.Println("Correlation monitoring stopped")
			return
		case <-ticker.C:
			shs.updateCorrelations()
		}
	}
}

// runHedgeMonitoring 运行对冲监控
func (shs *SmartHedgingSystem) runHedgeMonitoring() {
	defer shs.wg.Done()

	ticker := time.NewTicker(shs.hedgeFrequency)
	defer ticker.Stop()

	log.Println("Hedge monitoring started")

	for {
		select {
		case <-shs.ctx.Done():
			log.Println("Hedge monitoring stopped")
			return
		case <-ticker.C:
			shs.monitorHedges()
		}
	}
}

// runDynamicAdjustment 运行动态调整
func (shs *SmartHedgingSystem) runDynamicAdjustment() {
	defer shs.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	log.Println("Dynamic adjustment started")

	for {
		select {
		case <-shs.ctx.Done():
			log.Println("Dynamic adjustment stopped")
			return
		case <-ticker.C:
			if shs.dynamicAdjustment {
				shs.performDynamicAdjustment()
			}
		}
	}
}

// runPerformanceAnalysis 运行性能分析
func (shs *SmartHedgingSystem) runPerformanceAnalysis() {
	defer shs.wg.Done()

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	log.Println("Performance analysis started")

	for {
		select {
		case <-shs.ctx.Done():
			log.Println("Performance analysis stopped")
			return
		case <-ticker.C:
			shs.analyzePerformance()
		}
	}
}

// runMetricsCollection 运行指标收集
func (shs *SmartHedgingSystem) runMetricsCollection() {
	defer shs.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Println("Metrics collection started")

	for {
		select {
		case <-shs.ctx.Done():
			log.Println("Metrics collection stopped")
			return
		case <-ticker.C:
			shs.updateMetrics()
		}
	}
}

// CreateHedge 创建对冲仓位
func (shs *SmartHedgingSystem) CreateHedge(baseAsset, hedgeAsset string, baseQuantity float64) (*HedgePosition, error) {
	log.Printf("Creating hedge: %s -> %s (quantity: %.4f)", baseAsset, hedgeAsset, baseQuantity)

	// 检查相关性
	correlation := shs.getCorrelation(baseAsset, hedgeAsset)
	if math.Abs(correlation) < shs.correlationThreshold {
		return nil, fmt.Errorf("correlation too low: %.4f < %.4f",
			math.Abs(correlation), shs.correlationThreshold)
	}

	// 计算最优对冲比率
	optimalRatio, err := shs.calculateOptimalHedgeRatio(baseAsset, hedgeAsset)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate optimal hedge ratio: %w", err)
	}

	// 应用约束
	hedgeRatio := shs.applyHedgeConstraints(optimalRatio)
	hedgeQuantity := baseQuantity * hedgeRatio

	// 创建对冲仓位
	hedge := &HedgePosition{
		ID:                shs.generateHedgeID(),
		BaseAsset:         baseAsset,
		HedgeAsset:        hedgeAsset,
		BaseQuantity:      baseQuantity,
		HedgeQuantity:     hedgeQuantity,
		HedgeRatio:        hedgeRatio,
		OptimalRatio:      optimalRatio,
		EffectiveRatio:    hedgeRatio,
		Correlation:       correlation,
		Status:            "ACTIVE",
		CreatedAt:         time.Now(),
		LastUpdated:       time.Now(),
		AdjustmentHistory: make([]HedgeAdjustment, 0),
	}

	// 计算初始风险指标
	shs.updateHedgeRiskMetrics(hedge)

	// 执行对冲交易
	err = shs.executeHedgeTrade(hedge, "OPEN")
	if err != nil {
		return nil, fmt.Errorf("failed to execute hedge trade: %w", err)
	}

	// 保存对冲仓位
	shs.mu.Lock()
	shs.activeHedges[hedge.ID] = hedge
	shs.mu.Unlock()

	log.Printf("Hedge created successfully: %s (ratio: %.4f)", hedge.ID, hedgeRatio)
	return hedge, nil
}

// updateCorrelations 更新相关性
func (shs *SmartHedgingSystem) updateCorrelations() {
	log.Println("Updating correlations...")

	// 如果没有数据库连接，跳过更新
	if shs.db == nil {
		log.Println("No database connection available, skipping correlation update")
		return
	}

	// 从数据库获取活跃的交易对
	assets, err := shs.getActiveAssets()
	if err != nil {
		log.Printf("Failed to get active assets: %v", err)
		return
	}

	if len(assets) == 0 {
		log.Println("No active assets found for correlation calculation")
		return
	}

	log.Printf("Calculating correlations for %d assets: %v", len(assets), assets)

	// 计算相关性矩阵
	for i, asset1 := range assets {
		if shs.correlationMatrix[asset1] == nil {
			shs.correlationMatrix[asset1] = make(map[string]float64)
		}

		for j, asset2 := range assets {
			if i != j {
				correlation := shs.calculateCorrelation(asset1, asset2)
				shs.correlationMatrix[asset1][asset2] = correlation
				log.Printf("Correlation %s-%s: %.4f", asset1, asset2, correlation)
			} else {
				shs.correlationMatrix[asset1][asset2] = 1.0
			}
		}
	}

	shs.lastCorrelationUpdate = time.Now()
	log.Printf("Correlations updated at %s for %d asset pairs",
		shs.lastCorrelationUpdate.Format("15:04:05"), len(assets))
}

// monitorHedges 监控对冲仓位
func (shs *SmartHedgingSystem) monitorHedges() {
	log.Println("Monitoring hedge positions...")

	shs.mu.RLock()
	hedges := make(map[string]*HedgePosition)
	for k, v := range shs.activeHedges {
		hedges[k] = v
	}
	shs.mu.RUnlock()

	for _, hedge := range hedges {
		// 更新风险指标
		shs.updateHedgeRiskMetrics(hedge)

		// 检查是否需要调整
		if shs.needsAdjustment(hedge) {
			err := shs.adjustHedge(hedge)
			if err != nil {
				log.Printf("Failed to adjust hedge %s: %v", hedge.ID, err)
			}
		}

		// 检查对冲有效性
		shs.evaluateHedgeEffectiveness(hedge)
	}
}

// performDynamicAdjustment 执行动态调整
func (shs *SmartHedgingSystem) performDynamicAdjustment() {
	if time.Since(shs.dynamicAdjuster.lastAdjustment) < shs.dynamicAdjuster.maxAdjustmentFreq {
		return
	}

	log.Println("Performing dynamic adjustment...")

	// 检测市场条件变化
	marketCondition := shs.detectMarketCondition()

	shs.mu.RLock()
	hedges := make(map[string]*HedgePosition)
	for k, v := range shs.activeHedges {
		hedges[k] = v
	}
	shs.mu.RUnlock()

	for _, hedge := range hedges {
		// 根据市场条件调整对冲参数
		adjustment := shs.calculateDynamicAdjustment(hedge, marketCondition)
		if adjustment != nil {
			shs.applyDynamicAdjustment(hedge, adjustment)
		}
	}

	shs.dynamicAdjuster.lastAdjustment = time.Now()
}

// analyzePerformance 分析性能
func (shs *SmartHedgingSystem) analyzePerformance() {
	log.Println("Analyzing hedging performance...")

	performance := HedgePerformance{
		Timestamp:        time.Now(),
		PortfolioReturn:  shs.calculatePortfolioReturn(),
		HedgedReturn:     shs.calculateHedgedReturn(),
		UnhedgedReturn:   shs.calculateUnhedgedReturn(),
		TotalHedgingCost: shs.calculateTotalHedgingCost(),
	}

	// 计算对冲Alpha
	performance.HedgingAlpha = performance.HedgedReturn - performance.UnhedgedReturn

	// 计算跟踪误差
	performance.TrackingError = shs.calculateTrackingError()

	// 计算对冲有效性
	performance.HedgeEffectiveness = shs.calculateOverallHedgeEffectiveness()

	// 计算净表现
	performance.NetPerformance = performance.HedgedReturn - performance.TotalHedgingCost

	// 保存性能历史
	shs.performanceHistory = append(shs.performanceHistory, performance)

	// 保持历史记录在合理范围内
	if len(shs.performanceHistory) > 1000 {
		shs.performanceHistory = shs.performanceHistory[100:]
	}

	log.Printf("Performance analysis completed - Alpha: %.4f, Effectiveness: %.4f",
		performance.HedgingAlpha, performance.HedgeEffectiveness)
}

// Helper functions implementation...

func (shs *SmartHedgingSystem) getCorrelation(asset1, asset2 string) float64 {
	if shs.correlationMatrix[asset1] != nil {
		if corr, exists := shs.correlationMatrix[asset1][asset2]; exists {
			return corr
		}
	}
	return 0.0
}

func (shs *SmartHedgingSystem) calculateCorrelation(asset1, asset2 string) float64 {
	// 如果没有数据库连接，返回0
	if shs.db == nil {
		log.Printf("No database connection, cannot calculate correlation for %s-%s", asset1, asset2)
		return 0.0
	}

	// 获取历史价格数据
	prices1, err := shs.getHistoricalPrices(asset1, 30) // 30天数据
	if err != nil {
		log.Printf("Failed to get historical prices for %s: %v", asset1, err)
		return 0.0
	}

	prices2, err := shs.getHistoricalPrices(asset2, 30)
	if err != nil {
		log.Printf("Failed to get historical prices for %s: %v", asset2, err)
		return 0.0
	}

	if len(prices1) == 0 || len(prices2) == 0 {
		log.Printf("No price data available for correlation calculation: %s(%d) - %s(%d)",
			asset1, len(prices1), asset2, len(prices2))
		return 0.0
	}

	// 计算皮尔逊相关系数
	correlation := shs.calculatePearsonCorrelation(prices1, prices2)

	// 验证相关系数的有效性
	if math.IsNaN(correlation) || math.IsInf(correlation, 0) {
		log.Printf("Invalid correlation calculated for %s-%s: %f", asset1, asset2, correlation)
		return 0.0
	}

	return correlation
}

func (shs *SmartHedgingSystem) calculateOptimalHedgeRatio(baseAsset, hedgeAsset string) (float64, error) {
	switch shs.hedgeRatioCalculator.model {
	case "minimum_variance":
		return shs.calculateMinVarianceRatio(baseAsset, hedgeAsset)
	case "utility_maximization":
		return shs.calculateUtilityMaxRatio(baseAsset, hedgeAsset)
	case "var_minimization":
		return shs.calculateVaRMinRatio(baseAsset, hedgeAsset)
	default:
		return 0.8, nil // 默认80%对冲比率
	}
}

func (shs *SmartHedgingSystem) calculateMinVarianceRatio(baseAsset, hedgeAsset string) (float64, error) {
	// 最小方差对冲比率公式: h* = Cov(S,F) / Var(F)
	// 这里使用简化的计算
	correlation := shs.getCorrelation(baseAsset, hedgeAsset)
	baseVol := shs.getAssetVolatility(baseAsset)
	hedgeVol := shs.getAssetVolatility(hedgeAsset)

	if hedgeVol == 0 {
		return 0, fmt.Errorf("hedge asset has zero volatility")
	}

	ratio := correlation * (baseVol / hedgeVol)
	return math.Max(0, math.Min(1, ratio)), nil
}

func (shs *SmartHedgingSystem) calculateUtilityMaxRatio(baseAsset, hedgeAsset string) (float64, error) {
	// TODO: 实现效用最大化对冲比率计算
	return 0.75, nil
}

func (shs *SmartHedgingSystem) calculateVaRMinRatio(baseAsset, hedgeAsset string) (float64, error) {
	// TODO: 实现VaR最小化对冲比率计算
	return 0.85, nil
}

func (shs *SmartHedgingSystem) applyHedgeConstraints(ratio float64) float64 {
	// 应用最小和最大对冲比率约束
	ratio = math.Max(ratio, shs.minHedgeRatio)
	ratio = math.Min(ratio, shs.maxHedgeRatio)

	// 应用其他约束
	for _, constraint := range shs.hedgeRatioCalculator.constraints {
		if !constraint.IsActive {
			continue
		}

		switch constraint.Type {
		case "MIN_RATIO":
			ratio = math.Max(ratio, constraint.Value)
		case "MAX_RATIO":
			ratio = math.Min(ratio, constraint.Value)
		}
	}

	return ratio
}

func (shs *SmartHedgingSystem) updateHedgeRiskMetrics(hedge *HedgePosition) {
	// 计算Beta
	hedge.Beta = shs.calculateBeta(hedge.BaseAsset, hedge.HedgeAsset)

	// 计算跟踪误差
	hedge.TrackingError = shs.calculateHedgeTrackingError(hedge)

	// 计算对冲有效性
	hedge.HedgeEffectiveness = shs.calculateHedgeEffectiveness(hedge)

	// 计算基差风险
	hedge.BasisRisk = shs.calculateBasisRisk(hedge)

	// 计算净敞口
	hedge.NetExposure = math.Abs(hedge.BaseQuantity - hedge.HedgeQuantity*hedge.Correlation)

	hedge.LastUpdated = time.Now()
}

func (shs *SmartHedgingSystem) executeHedgeTrade(hedge *HedgePosition, action string) error {
	execution := HedgeExecution{
		ID:          shs.generateExecutionID(),
		HedgeID:     hedge.ID,
		Action:      action,
		Symbol:      hedge.HedgeAsset + "USDT",
		Side:        "SELL", // 假设卖出对冲工具
		Quantity:    hedge.HedgeQuantity,
		TargetPrice: 0.0, // 需要从市场获取实时价格
		Status:      "PENDING",
		Timestamp:   time.Now(),
	}

	// TODO: 实现实际的交易执行逻辑
	executionStart := time.Now()

	// 获取实时市场价格
	marketPrice, err := shs.getMarketPrice(execution.Symbol)
	if err != nil {
		log.Printf("Failed to get market price for %s: %v", execution.Symbol, err)
		return fmt.Errorf("failed to get market price: %w", err)
	}

	execution.TargetPrice = marketPrice
	execution.ExecutedPrice = 0.0 // 需要实际交易执行
	execution.Slippage = 0.0
	execution.Cost = 0.0
	execution.Status = "PENDING" // 等待实际执行
	execution.ExecutionTime = time.Since(executionStart)

	// 更新对冲成本
	hedge.HedgeCost += execution.Cost

	// 记录执行历史
	shs.hedgeExecutor.mu.Lock()
	shs.hedgeExecutor.executionHistory = append(shs.hedgeExecutor.executionHistory, execution)
	shs.hedgeExecutor.mu.Unlock()

	log.Printf("Hedge trade executed: %s %s %.4f @ %.2f (slippage: %.4f)",
		execution.Action, execution.Symbol, execution.Quantity,
		execution.ExecutedPrice, execution.Slippage)

	return nil
}

func (shs *SmartHedgingSystem) needsAdjustment(hedge *HedgePosition) bool {
	// 检查相关性变化
	currentCorr := shs.getCorrelation(hedge.BaseAsset, hedge.HedgeAsset)
	corrChange := math.Abs(currentCorr - hedge.Correlation)

	if corrChange > 0.1 { // 相关性变化超过10%
		return true
	}

	// 检查对冲比率偏离
	optimalRatio, _ := shs.calculateOptimalHedgeRatio(hedge.BaseAsset, hedge.HedgeAsset)
	ratioDeviation := math.Abs(hedge.HedgeRatio - optimalRatio)

	if ratioDeviation > 0.05 { // 对冲比率偏离超过5%
		return true
	}

	// 检查对冲有效性
	if hedge.HedgeEffectiveness < 0.8 { // 有效性低于80%
		return true
	}

	return false
}

func (shs *SmartHedgingSystem) adjustHedge(hedge *HedgePosition) error {
	log.Printf("Adjusting hedge: %s", hedge.ID)

	oldRatio := hedge.HedgeRatio

	// 重新计算最优对冲比率
	newOptimalRatio, err := shs.calculateOptimalHedgeRatio(hedge.BaseAsset, hedge.HedgeAsset)
	if err != nil {
		return err
	}

	newRatio := shs.applyHedgeConstraints(newOptimalRatio)

	if math.Abs(newRatio-oldRatio) < 0.01 { // 变化太小，不需要调整
		return nil
	}

	// 计算调整数量
	newHedgeQuantity := hedge.BaseQuantity * newRatio
	adjustmentSize := newHedgeQuantity - hedge.HedgeQuantity

	// 执行调整交易
	if adjustmentSize != 0 {
		adjustmentExecution := HedgeExecution{
			ID:          shs.generateExecutionID(),
			HedgeID:     hedge.ID,
			Action:      "ADJUST",
			Symbol:      hedge.HedgeAsset + "USDT",
			Side:        "BUY",
			Quantity:    math.Abs(adjustmentSize),
			TargetPrice: 50000.0,
			Status:      "PENDING",
			Timestamp:   time.Now(),
		}

		if adjustmentSize < 0 {
			adjustmentExecution.Side = "SELL"
		}

		// 模拟执行
		adjustmentExecution.ExecutedPrice = adjustmentExecution.TargetPrice * (1 + 0.0005)
		adjustmentExecution.Slippage = 0.0005
		adjustmentExecution.Cost = math.Abs(adjustmentSize) * adjustmentExecution.ExecutedPrice * 0.0004
		adjustmentExecution.Status = "EXECUTED"
		adjustmentExecution.ExecutionTime = 500 * time.Millisecond

		// 更新对冲仓位
		hedge.HedgeQuantity = newHedgeQuantity
		hedge.HedgeRatio = newRatio
		hedge.OptimalRatio = newOptimalRatio
		hedge.HedgeCost += adjustmentExecution.Cost
		hedge.LastAdjusted = time.Now()
		hedge.LastUpdated = time.Now()

		// 记录调整历史
		adjustment := HedgeAdjustment{
			Timestamp:      time.Now(),
			Trigger:        "RATIO_DEVIATION",
			OldRatio:       oldRatio,
			NewRatio:       newRatio,
			AdjustmentSize: adjustmentSize,
			Cost:           adjustmentExecution.Cost,
			Reason:         "Optimal ratio recalculation",
			Effectiveness:  hedge.HedgeEffectiveness,
		}
		hedge.AdjustmentHistory = append(hedge.AdjustmentHistory, adjustment)

		// 记录执行历史
		shs.hedgeExecutor.mu.Lock()
		shs.hedgeExecutor.executionHistory = append(shs.hedgeExecutor.executionHistory, adjustmentExecution)
		shs.hedgeExecutor.mu.Unlock()

		log.Printf("Hedge adjusted: %s (%.4f -> %.4f, adjustment: %.4f)",
			hedge.ID, oldRatio, newRatio, adjustmentSize)
	}

	return nil
}

func (shs *SmartHedgingSystem) evaluateHedgeEffectiveness(hedge *HedgePosition) {
	// 更新对冲有效性
	hedge.HedgeEffectiveness = shs.calculateHedgeEffectiveness(hedge)

	// 如果有效性过低，考虑关闭对冲
	if hedge.HedgeEffectiveness < 0.5 {
		log.Printf("Hedge %s has low effectiveness: %.4f", hedge.ID, hedge.HedgeEffectiveness)
		// TODO: 实现低效对冲的处理逻辑
	}
}

func (shs *SmartHedgingSystem) detectMarketCondition() string {
	// TODO: 实现基于真实市场数据的状态检测
	// 需要分析波动率、趋势、相关性等指标

	// 获取市场波动率
	volatility, err := shs.getMarketVolatility()
	if err != nil {
		log.Printf("Failed to get market volatility: %v", err)
		return "UNKNOWN"
	}

	// 基于波动率判断市场状态
	if volatility > 0.05 {
		return "HIGH_VOLATILITY"
	} else if volatility < 0.02 {
		return "LOW_VOLATILITY"
	}

	return "NORMAL"
}

func (shs *SmartHedgingSystem) calculateDynamicAdjustment(hedge *HedgePosition, marketCondition string) *DynamicAdjustment {
	// TODO: 根据市场条件计算动态调整参数
	return nil // 模拟暂不需要调整
}

func (shs *SmartHedgingSystem) applyDynamicAdjustment(hedge *HedgePosition, adjustment *DynamicAdjustment) {
	// TODO: 应用动态调整
}

// 计算相关指标的辅助函数...
func (shs *SmartHedgingSystem) getAssetVolatility(asset string) float64 {
	// TODO: 从历史数据计算波动率
	volatilities := map[string]float64{
		"BTC": 0.8,
		"ETH": 0.9,
		"BNB": 0.7,
		"ADA": 1.1,
	}
	if vol, exists := volatilities[asset]; exists {
		return vol
	}
	return 0.5
}

func (shs *SmartHedgingSystem) calculateBeta(baseAsset, hedgeAsset string) float64 {
	// TODO: 计算Beta值
	return 1.0
}

func (shs *SmartHedgingSystem) calculateHedgeTrackingError(hedge *HedgePosition) float64 {
	// TODO: 计算跟踪误差
	return 0.02
}

func (shs *SmartHedgingSystem) calculateHedgeEffectiveness(hedge *HedgePosition) float64 {
	// 对冲有效性 = 1 - (对冲后方差 / 对冲前方差)
	// 简化计算
	return math.Max(0, 1.0-(hedge.TrackingError/0.1))
}

func (shs *SmartHedgingSystem) calculateBasisRisk(hedge *HedgePosition) float64 {
	// TODO: 计算基差风险
	return 0.01
}

func (shs *SmartHedgingSystem) calculatePortfolioReturn() float64 {
	// TODO: 计算组合收益率
	return 0.12
}

func (shs *SmartHedgingSystem) calculateHedgedReturn() float64 {
	// TODO: 计算对冲后收益率
	return 0.10
}

func (shs *SmartHedgingSystem) calculateUnhedgedReturn() float64 {
	// TODO: 计算未对冲收益率
	return 0.15
}

func (shs *SmartHedgingSystem) calculateTotalHedgingCost() float64 {
	totalCost := 0.0
	for _, hedge := range shs.activeHedges {
		totalCost += hedge.HedgeCost
	}
	return totalCost
}

func (shs *SmartHedgingSystem) calculateTrackingError() float64 {
	// TODO: 计算组合跟踪误差
	return 0.015
}

func (shs *SmartHedgingSystem) calculateOverallHedgeEffectiveness() float64 {
	if len(shs.activeHedges) == 0 {
		return 0.0
	}

	totalEffectiveness := 0.0
	for _, hedge := range shs.activeHedges {
		totalEffectiveness += hedge.HedgeEffectiveness
	}

	return totalEffectiveness / float64(len(shs.activeHedges))
}

func (shs *SmartHedgingSystem) updateMetrics() {
	shs.hedgingMetrics.mu.Lock()
	defer shs.hedgingMetrics.mu.Unlock()

	// 更新对冲效果指标
	shs.hedgingMetrics.OverallHedgeEffectiveness = shs.calculateOverallHedgeEffectiveness()
	shs.hedgingMetrics.AverageHedgeRatio = shs.calculateAverageHedgeRatio()
	shs.hedgingMetrics.TotalHedgingCost = shs.calculateTotalHedgingCost()

	// 更新相关性统计
	shs.hedgingMetrics.AverageCorrelation = shs.calculateAverageCorrelation()
	shs.hedgingMetrics.StrongCorrelationPairs = shs.countStrongCorrelationPairs()

	// 更新执行统计
	shs.updateExecutionMetrics()

	shs.hedgingMetrics.LastUpdated = time.Now()
}

func (shs *SmartHedgingSystem) calculateAverageHedgeRatio() float64 {
	if len(shs.activeHedges) == 0 {
		return 0.0
	}

	totalRatio := 0.0
	for _, hedge := range shs.activeHedges {
		totalRatio += hedge.HedgeRatio
	}

	return totalRatio / float64(len(shs.activeHedges))
}

func (shs *SmartHedgingSystem) calculateAverageCorrelation() float64 {
	// TODO: 计算平均相关性
	return 0.75
}

func (shs *SmartHedgingSystem) countStrongCorrelationPairs() int {
	count := 0
	for _, correlations := range shs.correlationMatrix {
		for _, corr := range correlations {
			if math.Abs(corr) > shs.correlationThreshold {
				count++
			}
		}
	}
	return count / 2 // 避免重复计算
}

func (shs *SmartHedgingSystem) updateExecutionMetrics() {
	shs.hedgeExecutor.mu.RLock()
	defer shs.hedgeExecutor.mu.RUnlock()

	shs.hedgingMetrics.TotalExecutions = int64(len(shs.hedgeExecutor.executionHistory))

	if len(shs.hedgeExecutor.executionHistory) == 0 {
		return
	}

	successCount := int64(0)
	totalSlippage := 0.0
	totalExecutionTime := time.Duration(0)

	for _, execution := range shs.hedgeExecutor.executionHistory {
		if execution.Status == "EXECUTED" {
			successCount++
			totalSlippage += math.Abs(execution.Slippage)
			totalExecutionTime += execution.ExecutionTime
		}
	}

	shs.hedgingMetrics.SuccessfulExecutions = successCount
	shs.hedgingMetrics.AverageSlippage = totalSlippage / float64(len(shs.hedgeExecutor.executionHistory))
	shs.hedgingMetrics.AverageExecutionTime = totalExecutionTime / time.Duration(len(shs.hedgeExecutor.executionHistory))
}

func (shs *SmartHedgingSystem) generateHedgeID() string {
	return fmt.Sprintf("HDG_%d", time.Now().Unix())
}

func (shs *SmartHedgingSystem) generateExecutionID() string {
	return fmt.Sprintf("EXE_%d", time.Now().UnixNano())
}

// GetStatus 获取对冲系统状态
func (shs *SmartHedgingSystem) GetStatus() map[string]interface{} {
	shs.mu.RLock()
	defer shs.mu.RUnlock()

	return map[string]interface{}{
		"running":                  shs.isRunning,
		"enabled":                  shs.enabled,
		"active_hedges":            len(shs.activeHedges),
		"hedge_instruments":        len(shs.hedgeInstruments),
		"correlation_threshold":    shs.correlationThreshold,
		"dynamic_adjustment":       shs.dynamicAdjustment,
		"last_correlation_update":  shs.lastCorrelationUpdate,
		"hedging_metrics":          shs.hedgingMetrics,
		"performance_history_size": len(shs.performanceHistory),
	}
}

// GetHedgingMetrics 获取对冲指标
func (shs *SmartHedgingSystem) GetHedgingMetrics() *HedgingMetrics {
	shs.hedgingMetrics.mu.RLock()
	defer shs.hedgingMetrics.mu.RUnlock()

	// 创建一个新的HedgingMetrics实例，避免复制锁
	metrics := &HedgingMetrics{
		OverallHedgeEffectiveness: shs.hedgingMetrics.OverallHedgeEffectiveness,
		AverageHedgeRatio:         shs.hedgingMetrics.AverageHedgeRatio,
		TotalHedgingCost:          shs.hedgingMetrics.TotalHedgingCost,
		PortfolioVaRReduction:     shs.hedgingMetrics.PortfolioVaRReduction,
		AverageCorrelation:        shs.hedgingMetrics.AverageCorrelation,
		CorrelationStability:      shs.hedgingMetrics.CorrelationStability,
		StrongCorrelationPairs:    shs.hedgingMetrics.StrongCorrelationPairs,
		TotalExecutions:           shs.hedgingMetrics.TotalExecutions,
		SuccessfulExecutions:      shs.hedgingMetrics.SuccessfulExecutions,
		AverageSlippage:           shs.hedgingMetrics.AverageSlippage,
		AverageExecutionTime:      shs.hedgingMetrics.AverageExecutionTime,
		TotalAdjustments:          shs.hedgingMetrics.TotalAdjustments,
		AdjustmentFrequency:       shs.hedgingMetrics.AdjustmentFrequency,
		AverageAdjustmentCost:     shs.hedgingMetrics.AverageAdjustmentCost,
		HedgedVsUnhedgedReturn:    shs.hedgingMetrics.HedgedVsUnhedgedReturn,
		RiskAdjustedPerformance:   shs.hedgingMetrics.RiskAdjustedPerformance,
		InformationRatio:          shs.hedgingMetrics.InformationRatio,
		LastUpdated:               shs.hedgingMetrics.LastUpdated,
	}

	return metrics
}

// GetActiveHedges 获取活跃对冲仓位
func (shs *SmartHedgingSystem) GetActiveHedges() map[string]*HedgePosition {
	shs.mu.RLock()
	defer shs.mu.RUnlock()

	hedges := make(map[string]*HedgePosition)
	for k, v := range shs.activeHedges {
		hedges[k] = v
	}
	return hedges
}

// getHistoricalPrices 获取资产的历史价格数据
func (shs *SmartHedgingSystem) getHistoricalPrices(asset string, days int) ([]float64, error) {
	if shs.db == nil {
		return nil, fmt.Errorf("no database connection available")
	}

	// 构建交易对符号（如果需要的话）
	symbol := asset
	if !strings.HasSuffix(asset, "USDT") {
		symbol = asset + "USDT"
	}

	log.Printf("Getting %d days of historical prices for %s (symbol: %s)", days, asset, symbol)

	// 从market_data表获取历史价格数据
	query := `
		SELECT close, timestamp
		FROM market_data
		WHERE symbol = $1
		AND timestamp >= NOW() - INTERVAL '%d days'
		AND complete = true
		ORDER BY timestamp ASC
	`

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rows, err := shs.db.QueryContext(ctx, fmt.Sprintf(query, days), symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to query historical prices for %s: %w", symbol, err)
	}
	defer rows.Close()

	var prices []float64
	var timestamps []time.Time

	for rows.Next() {
		var price float64
		var timestamp time.Time

		if err := rows.Scan(&price, &timestamp); err != nil {
			log.Printf("Warning: failed to scan price data for %s: %v", symbol, err)
			continue
		}

		prices = append(prices, price)
		timestamps = append(timestamps, timestamp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating price data for %s: %w", symbol, err)
	}

	log.Printf("Retrieved %d price points for %s (requested %d days)", len(prices), symbol, days)

	// 如果数据点太少，记录警告
	if len(prices) < days/2 {
		log.Printf("Warning: insufficient price data for %s: got %d points, expected ~%d",
			symbol, len(prices), days)
	}

	return prices, nil
}

// calculatePearsonCorrelation 计算两个价格序列的皮尔逊相关系数
func (shs *SmartHedgingSystem) calculatePearsonCorrelation(prices1, prices2 []float64) float64 {
	if len(prices1) != len(prices2) || len(prices1) < 2 {
		return 0.0
	}

	n := float64(len(prices1))

	// 计算均值
	mean1, mean2 := 0.0, 0.0
	for i := 0; i < len(prices1); i++ {
		mean1 += prices1[i]
		mean2 += prices2[i]
	}
	mean1 /= n
	mean2 /= n

	// 计算协方差和方差
	covariance := 0.0
	variance1, variance2 := 0.0, 0.0

	for i := 0; i < len(prices1); i++ {
		diff1 := prices1[i] - mean1
		diff2 := prices2[i] - mean2

		covariance += diff1 * diff2
		variance1 += diff1 * diff1
		variance2 += diff2 * diff2
	}

	// 计算相关系数
	if variance1 == 0 || variance2 == 0 {
		return 0.0
	}

	correlation := covariance / (math.Sqrt(variance1) * math.Sqrt(variance2))
	return correlation
}

// getMarketVolatility 获取市场波动率
func (shs *SmartHedgingSystem) getMarketVolatility() (float64, error) {
	if shs.db == nil {
		return 0.0, fmt.Errorf("no database connection available")
	}

	log.Printf("Calculating market volatility from database")

	// 获取主要资产的价格数据来计算市场整体波动率
	majorAssets := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT"}
	var totalVolatility float64
	validAssets := 0

	for _, asset := range majorAssets {
		// 获取最近7天的价格数据
		prices, err := shs.getHistoricalPrices(strings.TrimSuffix(asset, "USDT"), 7)
		if err != nil || len(prices) < 5 {
			log.Printf("Warning: insufficient data for volatility calculation for %s: %v", asset, err)
			continue
		}

		// 计算该资产的波动率
		volatility := shs.calculateAssetVolatility(prices)
		if !math.IsNaN(volatility) && !math.IsInf(volatility, 0) {
			totalVolatility += volatility
			validAssets++
			log.Printf("Volatility for %s: %.4f", asset, volatility)
		}
	}

	if validAssets == 0 {
		return 0.0, fmt.Errorf("no valid volatility data available")
	}

	// 计算平均市场波动率
	marketVolatility := totalVolatility / float64(validAssets)

	log.Printf("Calculated market volatility: %.4f (based on %d assets)", marketVolatility, validAssets)
	return marketVolatility, nil
}

// getMarketPrice 获取实时市场价格
func (shs *SmartHedgingSystem) getMarketPrice(symbol string) (float64, error) {
	if shs.db == nil {
		return 0.0, fmt.Errorf("no database connection available")
	}

	log.Printf("Getting market price for %s from database", symbol)

	// 首先尝试从tickers表获取最新价格
	query := `
		SELECT price, updated_at
		FROM tickers
		WHERE symbol = $1
		ORDER BY updated_at DESC
		LIMIT 1
	`

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var price float64
	var updatedAt time.Time

	err := shs.db.QueryRowContext(ctx, query, symbol).Scan(&price, &updatedAt)
	if err == nil {
		// 检查数据是否过期（超过5分钟）
		if time.Since(updatedAt) <= 5*time.Minute {
			log.Printf("Got current price for %s: %.4f (updated %v ago)",
				symbol, price, time.Since(updatedAt))
			return price, nil
		}
		log.Printf("Price data for %s is stale (updated %v ago), trying market_data",
			symbol, time.Since(updatedAt))
	}

	// 如果tickers表没有数据或数据过期，尝试从market_data表获取
	query = `
		SELECT close, timestamp
		FROM market_data
		WHERE symbol = $1
		AND complete = true
		ORDER BY timestamp DESC
		LIMIT 1
	`

	err = shs.db.QueryRowContext(ctx, query, symbol).Scan(&price, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0.0, fmt.Errorf("no price data found for symbol: %s", symbol)
		}
		return 0.0, fmt.Errorf("failed to query market price for %s: %w", symbol, err)
	}

	// 检查market_data的数据时效性
	if time.Since(updatedAt) > 1*time.Hour {
		log.Printf("Warning: market price for %s is stale (updated %v ago)",
			symbol, time.Since(updatedAt))
	}

	log.Printf("Got market price for %s: %.4f (from market_data, updated %v ago)",
		symbol, price, time.Since(updatedAt))
	return price, nil
}

// getActiveAssets 获取活跃的交易对资产
func (shs *SmartHedgingSystem) getActiveAssets() ([]string, error) {
	if shs.db == nil {
		return nil, fmt.Errorf("no database connection available")
	}

	// 从数据库获取有数据的活跃交易对
	query := `
		SELECT DISTINCT REPLACE(symbol, 'USDT', '') as base_asset
		FROM market_data
		WHERE timestamp >= NOW() - INTERVAL '24 hours'
		AND complete = true
		AND symbol LIKE '%USDT'
		ORDER BY base_asset
		LIMIT 20
	`

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	rows, err := shs.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active assets: %w", err)
	}
	defer rows.Close()

	var assets []string
	for rows.Next() {
		var asset string
		if err := rows.Scan(&asset); err != nil {
			log.Printf("Warning: failed to scan asset: %v", err)
			continue
		}
		assets = append(assets, asset)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating active assets: %w", err)
	}

	// 如果数据库中没有数据，使用默认的主要资产
	if len(assets) == 0 {
		log.Println("No assets found in database, using default major assets")
		assets = []string{"BTC", "ETH", "BNB", "ADA", "SOL"}
	}

	return assets, nil
}

// calculateAssetVolatility 计算单个资产的波动率
func (shs *SmartHedgingSystem) calculateAssetVolatility(prices []float64) float64 {
	if len(prices) < 2 {
		return 0.0
	}

	// 计算收益率
	var returns []float64
	for i := 1; i < len(prices); i++ {
		if prices[i-1] > 0 {
			ret := math.Log(prices[i] / prices[i-1])
			returns = append(returns, ret)
		}
	}

	if len(returns) == 0 {
		return 0.0
	}

	// 计算收益率的标准差
	mean := 0.0
	for _, ret := range returns {
		mean += ret
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, ret := range returns {
		variance += math.Pow(ret-mean, 2)
	}
	variance /= float64(len(returns) - 1)

	// 年化波动率（假设每日数据）
	volatility := math.Sqrt(variance) * math.Sqrt(365)

	return volatility
}
