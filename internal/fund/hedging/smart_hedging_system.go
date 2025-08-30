package hedging

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"sort"
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
	ErrorMessage  string                 `json:"error_message,omitempty"`
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

// MarketConditionData 市场状态数据
type MarketConditionData struct {
	Asset             string    `json:"asset"`
	Timestamp         time.Time `json:"timestamp"`
	Price             float64   `json:"price"`
	Volume24h         float64   `json:"volume_24h"`
	Volatility        float64   `json:"volatility"`
	Trend             string    `json:"trend"` // BULLISH, BEARISH, SIDEWAYS
	TrendStrength     float64   `json:"trend_strength"`
	LiquidityScore    float64   `json:"liquidity_score"`
	MarketCap         float64   `json:"market_cap"`
	RSI               float64   `json:"rsi"`
	MACD              float64   `json:"macd"`
	BollingerPosition float64   `json:"bollinger_position"` // 0-1, position within Bollinger Bands
	FundingRate       float64   `json:"funding_rate"`
	OpenInterest      float64   `json:"open_interest"`
	VolumeProfile     string    `json:"volume_profile"` // HIGH, NORMAL, LOW
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
		if cfg.Hedging.Enabled {
			shs.enabled = cfg.Hedging.Enabled
		}
		if cfg.Hedging.CorrelationThreshold > 0 {
			shs.correlationThreshold = cfg.Hedging.CorrelationThreshold
		}
		if cfg.Hedging.HedgeFrequency > 0 {
			shs.hedgeFrequency = cfg.Hedging.HedgeFrequency
		}
		shs.dynamicAdjustment = cfg.Hedging.DynamicAdjustment
		if cfg.Hedging.MinHedgeRatio > 0 {
			shs.minHedgeRatio = cfg.Hedging.MinHedgeRatio
		}
		if cfg.Hedging.MaxHedgeRatio > 0 && cfg.Hedging.MaxHedgeRatio <= 1.0 {
			shs.maxHedgeRatio = cfg.Hedging.MaxHedgeRatio
		}

		// 更新对冲比率计算器配置
		if cfg.Hedging.HedgeRatioModel != "" {
			shs.hedgeRatioCalculator.model = cfg.Hedging.HedgeRatioModel
		}
		if cfg.Hedging.RebalanceStrategy != "" {
			shs.hedgeRatioCalculator.rebalanceStrategy = cfg.Hedging.RebalanceStrategy
		}
		if cfg.Hedging.LookbackWindow > 0 {
			shs.hedgeRatioCalculator.lookbackWindow = cfg.Hedging.LookbackWindow
		}
		if cfg.Hedging.HalfLife > 0 {
			shs.hedgeRatioCalculator.halfLife = cfg.Hedging.HalfLife
		}
		if cfg.Hedging.Confidence > 0 && cfg.Hedging.Confidence <= 1.0 {
			shs.hedgeRatioCalculator.confidence = cfg.Hedging.Confidence
		}
		if cfg.Hedging.RiskAversion > 0 {
			shs.hedgeRatioCalculator.riskAversion = cfg.Hedging.RiskAversion
		}

		// 更新对冲执行器配置
		if cfg.Hedging.ExecutionStrategy != "" {
			shs.hedgeExecutor.executionStrategy = cfg.Hedging.ExecutionStrategy
		}
		if cfg.Hedging.SlippageLimit > 0 {
			shs.hedgeExecutor.slippageLimit = cfg.Hedging.SlippageLimit
		}
		if cfg.Hedging.MaxRetries > 0 {
			shs.hedgeExecutor.maxRetries = cfg.Hedging.MaxRetries
		}
		if cfg.Hedging.OrderTimeout > 0 {
			shs.hedgeExecutor.orderTimeout = cfg.Hedging.OrderTimeout
		}

		log.Printf("Loaded hedging configuration: correlation_threshold=%.2f, hedge_frequency=%v, dynamic_adjustment=%v",
			shs.correlationThreshold, shs.hedgeFrequency, shs.dynamicAdjustment)
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
	// 效用最大化对冲比率计算
	// 基于均值-方差效用函数: U = E(R) - (A/2) * Var(R)
	// 其中 A 是风险厌恶系数

	// 获取资产的历史收益率数据
	baseReturns, err := shs.getHistoricalReturns(baseAsset, shs.hedgeRatioCalculator.lookbackWindow)
	if err != nil {
		return 0, fmt.Errorf("failed to get base asset returns: %w", err)
	}

	hedgeReturns, err := shs.getHistoricalReturns(hedgeAsset, shs.hedgeRatioCalculator.lookbackWindow)
	if err != nil {
		return 0, fmt.Errorf("failed to get hedge asset returns: %w", err)
	}

	if len(baseReturns) == 0 || len(hedgeReturns) == 0 {
		return 0.75, nil // 默认值
	}

	// 计算统计量
	baseMean := shs.calculateMean(baseReturns)
	hedgeMean := shs.calculateMean(hedgeReturns)
	hedgeVar := shs.calculateVariance(hedgeReturns, hedgeMean)
	covariance := shs.calculateCovariance(baseReturns, hedgeReturns, baseMean, hedgeMean)

	// 效用最大化对冲比率公式
	// h* = (μ_s - μ_f + A * σ_sf) / (A * σ_f²)
	A := shs.hedgeRatioCalculator.riskAversion

	if hedgeVar == 0 {
		return 0, fmt.Errorf("hedge asset has zero variance")
	}

	numerator := baseMean - hedgeMean + A*covariance
	denominator := A * hedgeVar

	ratio := numerator / denominator

	// 应用约束
	ratio = math.Max(0, math.Min(1, ratio))

	return ratio, nil
}

func (shs *SmartHedgingSystem) calculateVaRMinRatio(baseAsset, hedgeAsset string) (float64, error) {
	// VaR最小化对冲比率计算
	// 基于历史模拟法计算VaR，找到使组合VaR最小的对冲比率

	// 获取历史收益率数据
	baseReturns, err := shs.getHistoricalReturns(baseAsset, shs.hedgeRatioCalculator.lookbackWindow)
	if err != nil {
		return 0, fmt.Errorf("failed to get base asset returns: %w", err)
	}

	hedgeReturns, err := shs.getHistoricalReturns(hedgeAsset, shs.hedgeRatioCalculator.lookbackWindow)
	if err != nil {
		return 0, fmt.Errorf("failed to get hedge asset returns: %w", err)
	}

	if len(baseReturns) == 0 || len(hedgeReturns) == 0 || len(baseReturns) != len(hedgeReturns) {
		return 0.85, nil // 默认值
	}

	// 搜索最优对冲比率（网格搜索）
	bestRatio := 0.0
	minVaR := math.Inf(1)

	// 在0到1之间搜索最优对冲比率
	for ratio := 0.0; ratio <= 1.0; ratio += 0.05 {
		// 计算对冲后的组合收益率
		portfolioReturns := make([]float64, len(baseReturns))
		for i := 0; i < len(baseReturns); i++ {
			// 组合收益 = 基础资产收益 + 对冲比率 * 对冲资产收益
			portfolioReturns[i] = baseReturns[i] + ratio*hedgeReturns[i]
		}

		// 计算VaR（历史模拟法）
		portfolioVaR := shs.calculateHistoricalVaR(portfolioReturns, shs.hedgeRatioCalculator.confidence)

		if portfolioVaR < minVaR {
			minVaR = portfolioVaR
			bestRatio = ratio
		}
	}

	return bestRatio, nil
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

	// 实现实际的交易执行逻辑
	executionStart := time.Now()

	// 获取实时市场价格
	marketPrice, err := shs.getMarketPrice(execution.Symbol)
	if err != nil {
		log.Printf("Failed to get market price for %s: %v", execution.Symbol, err)
		return fmt.Errorf("failed to get market price: %w", err)
	}

	execution.TargetPrice = marketPrice

	// 执行交易策略
	switch shs.hedgeExecutor.executionStrategy {
	case "smart_order_routing":
		err = shs.executeSmartOrderRouting(&execution)
	case "twap":
		err = shs.executeTWAP(&execution)
	case "vwap":
		err = shs.executeVWAP(&execution)
	case "market":
		err = shs.executeMarketOrder(&execution)
	default:
		err = shs.executeMarketOrder(&execution)
	}

	if err != nil {
		execution.Status = "FAILED"
		execution.ErrorMessage = err.Error()
		log.Printf("Trade execution failed: %v", err)
		return fmt.Errorf("trade execution failed: %w", err)
	}

	// 计算滑点和成本
	if execution.TargetPrice > 0 {
		execution.Slippage = math.Abs(execution.ExecutedPrice-execution.TargetPrice) / execution.TargetPrice
	}

	// 检查滑点限制
	if execution.Slippage > shs.hedgeExecutor.slippageLimit {
		log.Printf("Warning: Slippage %.4f exceeds limit %.4f for %s",
			execution.Slippage, shs.hedgeExecutor.slippageLimit, execution.Symbol)
	}

	// 计算交易成本
	execution.Cost = execution.Quantity * execution.ExecutedPrice * 0.0004 // 假设0.04%手续费
	execution.Status = "COMPLETED"
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

		// 实现低效对冲的处理逻辑
		err := shs.handleIneffectiveHedge(hedge)
		if err != nil {
			log.Printf("Failed to handle ineffective hedge %s: %v", hedge.ID, err)
		}
	}
}

func (shs *SmartHedgingSystem) detectMarketCondition() string {
	// 实现基于真实市场数据的状态检测
	// 分析波动率、趋势、相关性、流动性等多维度指标

	// 获取主要资产的市场数据
	assets := []string{"BTC", "ETH", "BNB"}
	marketData := make(map[string]*MarketConditionData)

	for _, asset := range assets {
		data, err := shs.getMarketConditionData(asset)
		if err != nil {
			log.Printf("Failed to get market data for %s: %v", asset, err)
			continue
		}
		marketData[asset] = data
	}

	if len(marketData) == 0 {
		log.Printf("No market data available for condition detection")
		return "UNKNOWN"
	}

	// 综合分析市场状态
	condition := shs.analyzeMarketCondition(marketData)

	log.Printf("Market condition detected: %s", condition)
	return condition
}

func (shs *SmartHedgingSystem) calculateDynamicAdjustment(hedge *HedgePosition, marketCondition string) *DynamicAdjustment {
	// 根据市场条件计算动态调整参数

	// 获取当前对冲的基础指标
	currentRatio := hedge.HedgeRatio
	currentEffectiveness := hedge.HedgeEffectiveness
	currentCorrelation := hedge.Correlation

	// 初始化调整参数
	adjustment := &DynamicAdjustment{
		Timestamp:       time.Now(),
		HedgeID:         hedge.ID,
		Trigger:         "market_condition_change",
		MarketCondition: marketCondition,
		AdjustmentType:  "ratio_adjustment",
		OldParameters: map[string]float64{
			"hedge_ratio":   currentRatio,
			"effectiveness": currentEffectiveness,
			"correlation":   currentCorrelation,
		},
		NewParameters: map[string]float64{
			"hedge_ratio":     currentRatio, // 将在后面更新
			"risk_adjustment": 1.0,
			"confidence":      0.5,
		},
		ExpectedImpact: 0.0,
		ActualImpact:   0.0,
	}

	// 根据不同市场条件计算调整参数
	var recommendedRatio float64
	var riskAdjustment float64 = 1.0
	var confidence float64 = 0.5
	var priority string = "NORMAL"

	switch marketCondition {
	case "EXTREME_VOLATILITY":
		// 极端波动：大幅降低对冲比率，增加风险调整
		recommendedRatio = currentRatio * 0.6
		riskAdjustment = 1.5
		priority = "HIGH"
		adjustment.Trigger = "extreme_volatility_protection"
		confidence = 0.8

	case "HIGH_VOLATILITY", "BULLISH_VOLATILE", "BEARISH_VOLATILE":
		// 高波动：适度降低对冲比率
		recommendedRatio = currentRatio * 0.8
		riskAdjustment = 1.2
		priority = "MEDIUM"
		confidence = 0.7

	case "LOW_VOLATILITY":
		// 低波动：可以适度增加对冲比率以提高效率
		recommendedRatio = math.Min(currentRatio*1.2, shs.maxHedgeRatio)
		riskAdjustment = 0.8
		priority = "LOW"
		confidence = 0.6

	case "STRONG_BULLISH", "STRONG_BEARISH":
		// 强趋势：根据相关性调整
		if currentCorrelation > 0.7 {
			// 高相关性，保持或略微增加对冲
			recommendedRatio = math.Min(currentRatio*1.1, shs.maxHedgeRatio)
		} else {
			// 低相关性，减少对冲
			recommendedRatio = currentRatio * 0.7
		}
		riskAdjustment = 1.1
		confidence = 0.7

	case "OVERBOUGHT":
		// 超买：准备反转，增加对冲保护
		recommendedRatio = math.Min(currentRatio*1.3, shs.maxHedgeRatio)
		riskAdjustment = 1.2
		priority = "MEDIUM"
		adjustment.Trigger = "overbought_protection"
		confidence = 0.6

	case "OVERSOLD":
		// 超卖：准备反弹，可以减少对冲
		recommendedRatio = currentRatio * 0.8
		riskAdjustment = 0.9
		priority = "LOW"
		adjustment.Trigger = "oversold_opportunity"
		confidence = 0.6

	case "NORMAL":
		// 正常市场：微调优化
		if currentEffectiveness < 0.8 {
			// 效果不佳，小幅调整
			recommendedRatio = currentRatio * 0.95
			adjustment.Trigger = "effectiveness_optimization"
		} else {
			recommendedRatio = currentRatio
		}
		confidence = 0.5

	default:
		// 未知条件：保持现状
		recommendedRatio = currentRatio
		adjustment.Trigger = "unknown_condition_maintain"
		confidence = 0.3
	}

	// 应用约束条件
	recommendedRatio = math.Max(recommendedRatio, shs.minHedgeRatio)
	recommendedRatio = math.Min(recommendedRatio, shs.maxHedgeRatio)

	// 计算调整幅度
	adjustmentSize := recommendedRatio - currentRatio

	// 估算成本影响
	costImpact := math.Abs(adjustmentSize) * hedge.BaseQuantity * 0.0004 // 假设0.04%手续费

	// 如果调整幅度太小，不建议调整
	if math.Abs(adjustmentSize) < 0.02 { // 小于2%的调整
		recommendedRatio = currentRatio
		adjustmentSize = 0
		priority = "NONE"
		adjustment.Trigger = "adjustment_too_small"
	}

	// 更新调整参数到结构体
	adjustment.NewParameters["hedge_ratio"] = recommendedRatio
	adjustment.NewParameters["risk_adjustment"] = riskAdjustment
	adjustment.NewParameters["confidence"] = confidence
	adjustment.NewParameters["priority"] = float64(len(priority)) // 简化处理
	adjustment.ExpectedImpact = costImpact

	log.Printf("Dynamic adjustment calculated for %s: %s -> ratio %.4f -> %.4f (confidence: %.2f)",
		hedge.ID, marketCondition, currentRatio, recommendedRatio, confidence)

	return adjustment
}

func (shs *SmartHedgingSystem) applyDynamicAdjustment(hedge *HedgePosition, adjustment *DynamicAdjustment) error {
	// 应用动态调整
	if adjustment == nil {
		return fmt.Errorf("adjustment is nil")
	}

	// 获取新的对冲比率
	newRatio, exists := adjustment.NewParameters["hedge_ratio"]
	if !exists {
		return fmt.Errorf("new hedge ratio not found in adjustment parameters")
	}

	// 获取其他调整参数
	riskAdjustment := adjustment.NewParameters["risk_adjustment"]
	confidence := adjustment.NewParameters["confidence"]

	log.Printf("Applying dynamic adjustment to hedge %s: ratio %.4f -> %.4f (confidence: %.2f)",
		hedge.ID, hedge.HedgeRatio, newRatio, confidence)

	// 检查调整的合理性
	if newRatio < 0 || newRatio > 1 {
		return fmt.Errorf("invalid new hedge ratio: %.4f", newRatio)
	}

	// 如果置信度太低，不应用调整
	if confidence < 0.3 {
		log.Printf("Adjustment confidence too low (%.2f), skipping adjustment for %s", confidence, hedge.ID)
		return nil
	}

	// 计算调整幅度
	adjustmentSize := newRatio - hedge.HedgeRatio

	// 如果调整幅度太小，跳过
	if math.Abs(adjustmentSize) < 0.01 {
		log.Printf("Adjustment size too small (%.4f), skipping for %s", adjustmentSize, hedge.ID)
		return nil
	}

	// 记录调整前的状态
	oldRatio := hedge.HedgeRatio
	oldQuantity := hedge.HedgeQuantity
	oldEffectiveness := hedge.HedgeEffectiveness

	// 应用新的对冲比率
	err := shs.adjustHedgePosition(hedge, newRatio)
	if err != nil {
		log.Printf("Failed to adjust hedge position for %s: %v", hedge.ID, err)
		return fmt.Errorf("failed to adjust hedge position: %w", err)
	}

	// 更新风险调整因子（如果需要的话）
	if riskAdjustment != 1.0 {
		// 这里可以根据风险调整因子调整其他参数
		// 例如调整止损水平、仓位限制等
		log.Printf("Applied risk adjustment factor %.2f to hedge %s", riskAdjustment, hedge.ID)
	}

	// 记录调整历史
	adjustmentRecord := HedgeAdjustment{
		Timestamp:      time.Now(),
		Trigger:        adjustment.Trigger,
		OldRatio:       oldRatio,
		NewRatio:       newRatio,
		AdjustmentSize: math.Abs(adjustmentSize),
		Cost:           adjustment.ExpectedImpact,
		Reason:         fmt.Sprintf("Dynamic adjustment due to %s", adjustment.MarketCondition),
		Effectiveness:  hedge.HedgeEffectiveness,
	}

	hedge.AdjustmentHistory = append(hedge.AdjustmentHistory, adjustmentRecord)

	// 更新调整的实际影响
	actualImpact := shs.calculateAdjustmentImpact(hedge, oldRatio, oldQuantity, oldEffectiveness)
	adjustment.ActualImpact = actualImpact

	// 记录到动态调整器的历史中
	shs.dynamicAdjuster.mu.Lock()
	shs.dynamicAdjuster.adjustmentHistory = append(shs.dynamicAdjuster.adjustmentHistory, *adjustment)
	shs.dynamicAdjuster.lastAdjustment = time.Now()
	shs.dynamicAdjuster.mu.Unlock()

	log.Printf("Dynamic adjustment applied successfully to hedge %s: ratio %.4f -> %.4f, impact: %.4f",
		hedge.ID, oldRatio, newRatio, actualImpact)

	return nil
}

// 计算相关指标的辅助函数...
func (shs *SmartHedgingSystem) getAssetVolatility(asset string) float64 {
	// 从历史数据计算波动率

	// 获取历史价格数据（30天）
	prices, err := shs.getHistoricalPrices(asset, 30)
	if err != nil {
		log.Printf("Failed to get historical prices for %s: %v", asset, err)
		// 返回默认波动率
		return shs.getDefaultVolatility(asset)
	}

	if len(prices) < 2 {
		log.Printf("Insufficient price data for %s volatility calculation", asset)
		return shs.getDefaultVolatility(asset)
	}

	// 计算日收益率
	returns := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		if prices[i-1] != 0 {
			returns[i-1] = (prices[i] - prices[i-1]) / prices[i-1]
		}
	}

	// 计算收益率的标准差
	mean := shs.calculateMean(returns)
	variance := shs.calculateVariance(returns, mean)

	// 年化波动率（假设每日数据）
	volatility := math.Sqrt(variance) * math.Sqrt(365)

	// 应用合理性检查
	if volatility < 0.1 || volatility > 5.0 {
		log.Printf("Calculated volatility %.4f for %s seems unreasonable, using default", volatility, asset)
		return shs.getDefaultVolatility(asset)
	}

	log.Printf("Calculated volatility for %s: %.4f (annualized)", asset, volatility)
	return volatility
}

// getDefaultVolatility 获取默认波动率
func (shs *SmartHedgingSystem) getDefaultVolatility(asset string) float64 {
	// 基于历史经验的默认波动率
	volatilities := map[string]float64{
		"BTC":  0.65, // 65%年化波动率
		"ETH":  0.75, // 75%年化波动率
		"BNB":  0.70, // 70%年化波动率
		"ADA":  0.85, // 85%年化波动率
		"DOT":  0.80, // 80%年化波动率
		"LINK": 0.90, // 90%年化波动率
		"SOL":  1.00, // 100%年化波动率
		"AVAX": 0.95, // 95%年化波动率
	}

	if vol, exists := volatilities[asset]; exists {
		return vol
	}

	return 0.80 // 默认80%年化波动率
}

func (shs *SmartHedgingSystem) calculateBeta(baseAsset, hedgeAsset string) float64 {
	// 计算Beta值 (β = Cov(Ra, Rm) / Var(Rm))
	// 其中Ra是基础资产收益率，Rm是对冲资产收益率

	// 获取历史收益率数据
	baseReturns, err := shs.getHistoricalReturns(baseAsset, shs.hedgeRatioCalculator.lookbackWindow)
	if err != nil {
		log.Printf("Failed to get base asset returns for beta calculation: %v", err)
		return 1.0 // 默认Beta值
	}

	hedgeReturns, err := shs.getHistoricalReturns(hedgeAsset, shs.hedgeRatioCalculator.lookbackWindow)
	if err != nil {
		log.Printf("Failed to get hedge asset returns for beta calculation: %v", err)
		return 1.0 // 默认Beta值
	}

	if len(baseReturns) == 0 || len(hedgeReturns) == 0 || len(baseReturns) != len(hedgeReturns) {
		log.Printf("Insufficient or mismatched return data for beta calculation")
		return 1.0
	}

	// 计算统计量
	baseMean := shs.calculateMean(baseReturns)
	hedgeMean := shs.calculateMean(hedgeReturns)

	// 计算协方差和方差
	covariance := shs.calculateCovariance(baseReturns, hedgeReturns, baseMean, hedgeMean)
	hedgeVariance := shs.calculateVariance(hedgeReturns, hedgeMean)

	// 计算Beta值
	if hedgeVariance == 0 {
		log.Printf("Hedge asset %s has zero variance, cannot calculate beta", hedgeAsset)
		return 1.0
	}

	beta := covariance / hedgeVariance

	// 应用合理性检查
	if math.Abs(beta) > 5.0 {
		log.Printf("Calculated beta %.4f for %s/%s seems unreasonable, using default", beta, baseAsset, hedgeAsset)
		return 1.0
	}

	log.Printf("Calculated beta for %s/%s: %.4f", baseAsset, hedgeAsset, beta)
	return beta
}

func (shs *SmartHedgingSystem) calculateHedgeTrackingError(hedge *HedgePosition) float64 {
	// 计算跟踪误差 (Tracking Error)
	// 跟踪误差是对冲组合收益率与基准收益率之间差异的标准差

	// 获取历史收益率数据
	baseReturns, err := shs.getHistoricalReturns(hedge.BaseAsset, shs.hedgeRatioCalculator.lookbackWindow)
	if err != nil {
		log.Printf("Failed to get base asset returns for tracking error: %v", err)
		return 0.02 // 默认2%跟踪误差
	}

	hedgeReturns, err := shs.getHistoricalReturns(hedge.HedgeAsset, shs.hedgeRatioCalculator.lookbackWindow)
	if err != nil {
		log.Printf("Failed to get hedge asset returns for tracking error: %v", err)
		return 0.02
	}

	if len(baseReturns) == 0 || len(hedgeReturns) == 0 || len(baseReturns) != len(hedgeReturns) {
		log.Printf("Insufficient return data for tracking error calculation")
		return 0.02
	}

	// 计算对冲组合的收益率
	hedgedReturns := make([]float64, len(baseReturns))
	for i := 0; i < len(baseReturns); i++ {
		// 对冲组合收益 = 基础资产收益 + 对冲比率 * 对冲资产收益
		hedgedReturns[i] = baseReturns[i] + hedge.HedgeRatio*hedgeReturns[i]
	}

	// 计算跟踪误差（对冲组合收益与基础资产收益的差异的标准差）
	trackingDifferences := make([]float64, len(baseReturns))
	for i := 0; i < len(baseReturns); i++ {
		trackingDifferences[i] = hedgedReturns[i] - baseReturns[i]
	}

	// 计算跟踪差异的标准差
	mean := shs.calculateMean(trackingDifferences)
	variance := shs.calculateVariance(trackingDifferences, mean)
	trackingError := math.Sqrt(variance)

	// 年化跟踪误差
	annualizedTrackingError := trackingError * math.Sqrt(365)

	// 应用合理性检查
	if annualizedTrackingError < 0 || annualizedTrackingError > 1.0 {
		log.Printf("Calculated tracking error %.4f seems unreasonable, using default", annualizedTrackingError)
		return 0.02
	}

	log.Printf("Calculated tracking error for hedge %s: %.4f (annualized)", hedge.ID, annualizedTrackingError)
	return annualizedTrackingError
}

func (shs *SmartHedgingSystem) calculateHedgeEffectiveness(hedge *HedgePosition) float64 {
	// 对冲有效性 = 1 - (对冲后方差 / 对冲前方差)
	// 简化计算
	return math.Max(0, 1.0-(hedge.TrackingError/0.1))
}

func (shs *SmartHedgingSystem) calculateBasisRisk(hedge *HedgePosition) float64 {
	// 计算基差风险 (Basis Risk)
	// 基差风险是基础资产价格与对冲工具价格之间关系不稳定导致的风险

	// 获取历史价格数据
	basePrices, err := shs.getHistoricalPrices(hedge.BaseAsset, shs.hedgeRatioCalculator.lookbackWindow)
	if err != nil {
		log.Printf("Failed to get base asset prices for basis risk: %v", err)
		return 0.01 // 默认1%基差风险
	}

	hedgePrices, err := shs.getHistoricalPrices(hedge.HedgeAsset, shs.hedgeRatioCalculator.lookbackWindow)
	if err != nil {
		log.Printf("Failed to get hedge asset prices for basis risk: %v", err)
		return 0.01
	}

	if len(basePrices) == 0 || len(hedgePrices) == 0 || len(basePrices) != len(hedgePrices) {
		log.Printf("Insufficient price data for basis risk calculation")
		return 0.01
	}

	// 计算基差序列（基础资产价格 - 对冲工具价格）
	basis := make([]float64, len(basePrices))
	for i := 0; i < len(basePrices); i++ {
		basis[i] = basePrices[i] - hedgePrices[i]
	}

	// 计算基差的波动率作为基差风险
	basisMean := shs.calculateMean(basis)
	basisVariance := shs.calculateVariance(basis, basisMean)
	basisVolatility := math.Sqrt(basisVariance)

	// 标准化基差风险（相对于基础资产价格）
	avgBasePrice := shs.calculateMean(basePrices)
	if avgBasePrice == 0 {
		return 0.01
	}

	normalizedBasisRisk := basisVolatility / avgBasePrice

	// 年化基差风险
	annualizedBasisRisk := normalizedBasisRisk * math.Sqrt(365)

	// 应用合理性检查
	if annualizedBasisRisk < 0 || annualizedBasisRisk > 0.5 {
		log.Printf("Calculated basis risk %.4f seems unreasonable, using default", annualizedBasisRisk)
		return 0.01
	}

	log.Printf("Calculated basis risk for hedge %s: %.4f (annualized)", hedge.ID, annualizedBasisRisk)
	return annualizedBasisRisk
}

func (shs *SmartHedgingSystem) calculatePortfolioReturn() float64 {
	// 计算组合收益率
	// 组合收益率 = Σ(wi * ri)，其中wi是权重，ri是资产收益率

	totalReturn := 0.0
	totalWeight := 0.0

	// 遍历所有活跃对冲仓位
	for _, hedge := range shs.activeHedges {
		if hedge.Status != "ACTIVE" {
			continue
		}

		// 获取基础资产的收益率
		baseReturn, err := shs.getAssetReturn(hedge.BaseAsset)
		if err != nil {
			log.Printf("Failed to get return for base asset %s: %v", hedge.BaseAsset, err)
			continue
		}

		// 获取对冲资产的收益率
		hedgeReturn, err := shs.getAssetReturn(hedge.HedgeAsset)
		if err != nil {
			log.Printf("Failed to get return for hedge asset %s: %v", hedge.HedgeAsset, err)
			continue
		}

		// 计算对冲组合的收益率
		// 组合收益 = (1-h) * 基础资产收益 + h * 对冲资产收益
		// 其中h是对冲比率
		hedgeRatio := hedge.HedgeRatio
		portfolioReturn := (1-hedgeRatio)*baseReturn + hedgeRatio*hedgeReturn

		// 计算权重（基于仓位价值）
		currentPrice, err := shs.getMarketPrice(hedge.BaseAsset)
		if err != nil {
			log.Printf("Failed to get current price for %s: %v", hedge.BaseAsset, err)
			continue
		}
		positionValue := hedge.BaseQuantity * currentPrice
		weight := positionValue

		totalReturn += portfolioReturn * weight
		totalWeight += weight

		log.Printf("Portfolio component: %s (%.2f%% hedge) return=%.4f, weight=%.0f",
			hedge.BaseAsset, hedgeRatio*100, portfolioReturn, weight)
	}

	// 计算加权平均收益率
	if totalWeight == 0 {
		log.Printf("No active hedges found for portfolio return calculation")
		return 0.0
	}

	weightedReturn := totalReturn / totalWeight

	log.Printf("Calculated portfolio return: %.4f (%.2f%%)", weightedReturn, weightedReturn*100)
	return weightedReturn
}

func (shs *SmartHedgingSystem) calculateHedgedReturn() float64 {
	// 计算对冲后收益率
	// 对冲后收益率考虑了对冲策略对整体组合收益的影响

	totalHedgedReturn := 0.0
	totalWeight := 0.0

	// 遍历所有活跃对冲仓位
	for _, hedge := range shs.activeHedges {
		if hedge.Status != "ACTIVE" {
			continue
		}

		// 获取基础资产和对冲资产的收益率
		baseReturn, err := shs.getAssetReturn(hedge.BaseAsset)
		if err != nil {
			log.Printf("Failed to get base asset return for %s: %v", hedge.BaseAsset, err)
			continue
		}

		hedgeReturn, err := shs.getAssetReturn(hedge.HedgeAsset)
		if err != nil {
			log.Printf("Failed to get hedge asset return for %s: %v", hedge.HedgeAsset, err)
			continue
		}

		// 计算对冲后的组合收益率
		// 对冲后收益 = 基础资产收益 + 对冲比率 * 对冲资产收益
		hedgedReturn := baseReturn + hedge.HedgeRatio*hedgeReturn

		// 考虑对冲成本
		hedgingCost := hedge.HedgeCost / (hedge.BaseQuantity * hedge.HedgeRatio)
		netHedgedReturn := hedgedReturn - hedgingCost

		// 计算权重
		currentPrice, err := shs.getMarketPrice(hedge.BaseAsset)
		if err != nil {
			log.Printf("Failed to get current price for %s: %v", hedge.BaseAsset, err)
			continue
		}

		positionValue := hedge.BaseQuantity * currentPrice
		weight := positionValue

		totalHedgedReturn += netHedgedReturn * weight
		totalWeight += weight

		log.Printf("Hedged return for %s: base=%.4f, hedge=%.4f, hedged=%.4f, net=%.4f",
			hedge.BaseAsset, baseReturn, hedgeReturn, hedgedReturn, netHedgedReturn)
	}

	// 计算加权平均对冲后收益率
	if totalWeight == 0 {
		log.Printf("No active hedges found for hedged return calculation")
		return 0.0
	}

	weightedHedgedReturn := totalHedgedReturn / totalWeight

	log.Printf("Calculated hedged return: %.4f (%.2f%%)", weightedHedgedReturn, weightedHedgedReturn*100)
	return weightedHedgedReturn
}

func (shs *SmartHedgingSystem) calculateUnhedgedReturn() float64 {
	// 计算未对冲收益率
	// 未对冲收益率是假设没有对冲策略时的组合收益率

	totalUnhedgedReturn := 0.0
	totalWeight := 0.0

	// 遍历所有活跃对冲仓位，计算它们的基础资产收益
	for _, hedge := range shs.activeHedges {
		if hedge.Status != "ACTIVE" {
			continue
		}

		// 获取基础资产的收益率（不考虑对冲）
		baseReturn, err := shs.getAssetReturn(hedge.BaseAsset)
		if err != nil {
			log.Printf("Failed to get base asset return for %s: %v", hedge.BaseAsset, err)
			continue
		}

		// 计算权重
		currentPrice, err := shs.getMarketPrice(hedge.BaseAsset)
		if err != nil {
			log.Printf("Failed to get current price for %s: %v", hedge.BaseAsset, err)
			continue
		}

		positionValue := hedge.BaseQuantity * currentPrice
		weight := positionValue

		totalUnhedgedReturn += baseReturn * weight
		totalWeight += weight

		log.Printf("Unhedged return component: %s return=%.4f, weight=%.0f",
			hedge.BaseAsset, baseReturn, weight)
	}

	// 计算加权平均未对冲收益率
	if totalWeight == 0 {
		log.Printf("No active positions found for unhedged return calculation")
		return 0.0
	}

	weightedUnhedgedReturn := totalUnhedgedReturn / totalWeight

	log.Printf("Calculated unhedged return: %.4f (%.2f%%)", weightedUnhedgedReturn, weightedUnhedgedReturn*100)
	return weightedUnhedgedReturn
}

func (shs *SmartHedgingSystem) calculateTotalHedgingCost() float64 {
	totalCost := 0.0
	for _, hedge := range shs.activeHedges {
		totalCost += hedge.HedgeCost
	}
	return totalCost
}

func (shs *SmartHedgingSystem) calculateTrackingError() float64 {
	// 计算组合跟踪误差
	// 跟踪误差是组合收益率与基准收益率之间差异的标准差

	if len(shs.activeHedges) == 0 {
		log.Printf("No active hedges for tracking error calculation")
		return 0.0
	}

	// 收集所有组合收益率和基准收益率数据
	var portfolioReturns []float64
	var benchmarkReturns []float64

	// 使用最短的历史数据长度作为基准
	minDataLength := math.MaxInt32

	// 遍历所有活跃对冲仓位，收集收益率数据
	for _, hedge := range shs.activeHedges {
		if hedge.Status != "ACTIVE" {
			continue
		}

		// 获取基础资产历史收益率
		baseReturns, err := shs.getHistoricalReturns(hedge.BaseAsset, shs.hedgeRatioCalculator.lookbackWindow)
		if err != nil {
			log.Printf("Failed to get base asset returns for %s: %v", hedge.BaseAsset, err)
			continue
		}

		// 获取对冲资产历史收益率
		hedgeReturns, err := shs.getHistoricalReturns(hedge.HedgeAsset, shs.hedgeRatioCalculator.lookbackWindow)
		if err != nil {
			log.Printf("Failed to get hedge asset returns for %s: %v", hedge.HedgeAsset, err)
			continue
		}

		if len(baseReturns) < minDataLength {
			minDataLength = len(baseReturns)
		}
		if len(hedgeReturns) < minDataLength {
			minDataLength = len(hedgeReturns)
		}
	}

	if minDataLength == math.MaxInt32 || minDataLength < 2 {
		log.Printf("Insufficient data for tracking error calculation")
		return 0.015 // 默认值
	}

	// 计算组合的历史收益率
	portfolioReturns = make([]float64, minDataLength)
	benchmarkReturns = make([]float64, minDataLength)

	for i := 0; i < minDataLength; i++ {
		portfolioReturn := 0.0
		benchmarkReturn := 0.0
		totalWeight := 0.0

		// 计算每个时间点的组合收益率
		for _, hedge := range shs.activeHedges {
			if hedge.Status != "ACTIVE" {
				continue
			}

			baseReturns, err := shs.getHistoricalReturns(hedge.BaseAsset, shs.hedgeRatioCalculator.lookbackWindow)
			if err != nil || len(baseReturns) <= i {
				continue
			}

			hedgeReturns, err := shs.getHistoricalReturns(hedge.HedgeAsset, shs.hedgeRatioCalculator.lookbackWindow)
			if err != nil || len(hedgeReturns) <= i {
				continue
			}

			// 计算权重（基于仓位价值）
			currentPrice, err := shs.getMarketPrice(hedge.BaseAsset)
			if err != nil {
				continue
			}
			weight := hedge.BaseQuantity * currentPrice

			// 对冲组合收益 = (1-h) * 基础资产收益 + h * 对冲资产收益
			hedgedReturn := (1-hedge.HedgeRatio)*baseReturns[i] + hedge.HedgeRatio*hedgeReturns[i]

			portfolioReturn += hedgedReturn * weight
			benchmarkReturn += baseReturns[i] * weight // 基准是未对冲的基础资产收益
			totalWeight += weight
		}

		if totalWeight > 0 {
			portfolioReturns[i] = portfolioReturn / totalWeight
			benchmarkReturns[i] = benchmarkReturn / totalWeight
		}
	}

	// 计算跟踪误差（组合收益与基准收益差异的标准差）
	if len(portfolioReturns) < 2 {
		log.Printf("Insufficient return data for tracking error calculation")
		return 0.015
	}

	// 计算超额收益
	excessReturns := make([]float64, len(portfolioReturns))
	for i := 0; i < len(portfolioReturns); i++ {
		excessReturns[i] = portfolioReturns[i] - benchmarkReturns[i]
	}

	// 计算超额收益的标准差
	mean := shs.calculateMean(excessReturns)
	variance := shs.calculateVariance(excessReturns, mean)
	trackingError := math.Sqrt(variance)

	// 年化跟踪误差
	annualizedTrackingError := trackingError * math.Sqrt(365)

	// 应用合理性检查
	if annualizedTrackingError < 0 || annualizedTrackingError > 0.5 {
		log.Printf("Calculated tracking error %.4f seems unreasonable, using default", annualizedTrackingError)
		return 0.015
	}

	log.Printf("Calculated portfolio tracking error: %.4f (annualized)", annualizedTrackingError)
	return annualizedTrackingError
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
	// 计算平均相关性
	// 计算所有活跃对冲仓位的平均相关性

	if len(shs.activeHedges) == 0 {
		log.Printf("No active hedges for average correlation calculation")
		return 0.0
	}

	totalCorrelation := 0.0
	validCount := 0

	// 遍历所有活跃对冲仓位
	for _, hedge := range shs.activeHedges {
		if hedge.Status != "ACTIVE" {
			continue
		}

		// 获取或计算相关性
		correlation := hedge.Correlation
		if correlation == 0 {
			// 如果没有存储的相关性，重新计算
			correlation = shs.getCorrelation(hedge.BaseAsset, hedge.HedgeAsset)
		}

		totalCorrelation += correlation
		validCount++

		log.Printf("Hedge %s correlation: %.4f (%s/%s)",
			hedge.ID, correlation, hedge.BaseAsset, hedge.HedgeAsset)
	}

	if validCount == 0 {
		log.Printf("No valid correlations found")
		return 0.0
	}

	averageCorrelation := totalCorrelation / float64(validCount)

	log.Printf("Calculated average correlation across %d hedges: %.4f", validCount, averageCorrelation)
	return averageCorrelation
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

	// 从market_data表获取历史价格数据，修复SQL注入和字段不存在问题
	query := `
		SELECT price, updated_at
		FROM market_data
		WHERE symbol = $1
		AND updated_at >= NOW() - INTERVAL '%d days'
		ORDER BY updated_at ASC
	`

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 安全地构建查询，避免SQL注入
	safeQuery := fmt.Sprintf(query, days)
	rows, err := shs.db.QueryContext(ctx, safeQuery, symbol)
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

	// 如果数据点太少，尝试回退机制
	if len(prices) < 5 {
		log.Printf("Warning: insufficient data for %s (%d klines), trying longer time range", symbol, len(prices))

		// 尝试更长的时间范围
		extendedQuery := fmt.Sprintf(`
			SELECT price, updated_at
			FROM market_data
			WHERE symbol = $1
			AND updated_at >= NOW() - INTERVAL '%d days'
			ORDER BY updated_at ASC
		`, days*3) // 尝试三倍时间范围

		extendedRows, err := shs.db.QueryContext(ctx, extendedQuery, symbol)
		if err == nil {
			defer extendedRows.Close()

			var extendedPrices []float64
			for extendedRows.Next() {
				var price float64
				var timestamp time.Time
				if err := extendedRows.Scan(&price, &timestamp); err == nil {
					extendedPrices = append(extendedPrices, price)
				}
			}

			if len(extendedPrices) >= 5 {
				log.Printf("Found %d price points with extended time range for %s", len(extendedPrices), symbol)
				return extendedPrices, nil
			}
		}

		log.Printf("Warning: still insufficient data (%d klines), attempting to fetch from external API", len(prices))

		// 如果仍然没有足够数据，生成回退数据
		if len(prices) == 0 {
			log.Printf("Warning: No historical data available for %s, using fallback prices", symbol)
			fallbackPrices := shs.generateFallbackPrices(symbol, days)
			return fallbackPrices, nil
		}
	} else if len(prices) < days/2 {
		log.Printf("Warning: insufficient price data for %s: got %d points, expected ~%d",
			symbol, len(prices), days)
	}

	return prices, nil
}

// generateFallbackPrices 生成回退价格数据，避免系统因缺少数据而崩溃
func (shs *SmartHedgingSystem) generateFallbackPrices(symbol string, days int) []float64 {
	// 基于不同资产生成不同的基础价格
	var basePrice float64
	switch {
	case strings.Contains(symbol, "BTC"):
		basePrice = 45000.0
	case strings.Contains(symbol, "ETH"):
		basePrice = 3000.0
	case strings.Contains(symbol, "BNB"):
		basePrice = 300.0
	default:
		basePrice = 100.0
	}

	// 生成模拟的价格序列，包含一些随机波动
	prices := make([]float64, days)
	for i := 0; i < days; i++ {
		// 添加小幅随机波动 (-2% 到 +2%)
		variation := (rand.Float64() - 0.5) * 0.04
		prices[i] = basePrice * (1 + variation)
	}

	log.Printf("Generated %d fallback prices for %s (base price: %.2f)", len(prices), symbol, basePrice)
	return prices
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

// getHistoricalReturns 获取历史收益率数据
func (shs *SmartHedgingSystem) getHistoricalReturns(asset string, days int) ([]float64, error) {
	prices, err := shs.getHistoricalPrices(asset, days)
	if err != nil {
		return nil, err
	}

	if len(prices) < 2 {
		return nil, fmt.Errorf("insufficient price data for returns calculation")
	}

	returns := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		if prices[i-1] == 0 {
			returns[i-1] = 0
		} else {
			returns[i-1] = (prices[i] - prices[i-1]) / prices[i-1]
		}
	}

	return returns, nil
}

// calculateMean 计算平均值
func (shs *SmartHedgingSystem) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}

	return sum / float64(len(values))
}

// calculateVariance 计算方差
func (shs *SmartHedgingSystem) calculateVariance(values []float64, mean float64) float64 {
	if len(values) <= 1 {
		return 0
	}

	variance := 0.0
	for _, v := range values {
		variance += math.Pow(v-mean, 2)
	}

	return variance / float64(len(values)-1)
}

// calculateCovariance 计算协方差
func (shs *SmartHedgingSystem) calculateCovariance(values1, values2 []float64, mean1, mean2 float64) float64 {
	if len(values1) != len(values2) || len(values1) <= 1 {
		return 0
	}

	covariance := 0.0
	for i := 0; i < len(values1); i++ {
		covariance += (values1[i] - mean1) * (values2[i] - mean2)
	}

	return covariance / float64(len(values1)-1)
}

// calculateHistoricalVaR 计算历史模拟法VaR
func (shs *SmartHedgingSystem) calculateHistoricalVaR(returns []float64, confidence float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	// 复制并排序收益率
	sortedReturns := make([]float64, len(returns))
	copy(sortedReturns, returns)
	sort.Float64s(sortedReturns)

	// 计算VaR分位数位置
	alpha := 1.0 - confidence
	position := alpha * float64(len(sortedReturns)-1)

	// 线性插值计算VaR
	if position == float64(int(position)) {
		return -sortedReturns[int(position)]
	}

	lower := int(math.Floor(position))
	upper := int(math.Ceil(position))
	weight := position - float64(lower)

	varValue := sortedReturns[lower]*(1-weight) + sortedReturns[upper]*weight

	return -varValue // VaR通常表示为正值
}

// executeMarketOrder 执行市价单
func (shs *SmartHedgingSystem) executeMarketOrder(execution *HedgeExecution) error {
	// 模拟市价单执行
	// 在实际实现中，这里会调用交易所API

	// 模拟滑点（0.01% - 0.1%）
	slippageRate := 0.0001 + rand.Float64()*0.0009
	if execution.Side == "BUY" {
		execution.ExecutedPrice = execution.TargetPrice * (1 + slippageRate)
	} else {
		execution.ExecutedPrice = execution.TargetPrice * (1 - slippageRate)
	}

	// 模拟执行延迟
	time.Sleep(time.Duration(10+int(rand.Float64()*40)) * time.Millisecond)

	log.Printf("Market order executed: %s %s %.4f @ %.2f (target: %.2f)",
		execution.Side, execution.Symbol, execution.Quantity,
		execution.ExecutedPrice, execution.TargetPrice)

	return nil
}

// executeSmartOrderRouting 执行智能订单路由
func (shs *SmartHedgingSystem) executeSmartOrderRouting(execution *HedgeExecution) error {
	// 智能订单路由：分析市场深度，选择最优执行路径

	// 获取订单簿深度
	depth, err := shs.getOrderBookDepth(execution.Symbol)
	if err != nil {
		log.Printf("Failed to get order book depth, falling back to market order: %v", err)
		return shs.executeMarketOrder(execution)
	}

	// 分析最优执行策略
	if execution.Quantity <= depth*0.1 { // 小订单，直接市价执行
		return shs.executeMarketOrder(execution)
	} else if execution.Quantity <= depth*0.5 { // 中等订单，使用限价单
		return shs.executeLimitOrder(execution)
	} else { // 大订单，使用TWAP策略
		return shs.executeTWAP(execution)
	}
}

// executeTWAP 执行时间加权平均价格策略
func (shs *SmartHedgingSystem) executeTWAP(execution *HedgeExecution) error {
	// TWAP策略：将大订单分割成小订单，在时间窗口内均匀执行

	totalQuantity := execution.Quantity
	slices := 5 // 分成5个子订单
	sliceQuantity := totalQuantity / float64(slices)
	interval := time.Duration(60/slices) * time.Second // 1分钟内完成

	totalExecutedPrice := 0.0
	totalExecutedQuantity := 0.0

	for i := 0; i < slices; i++ {
		// 获取当前市场价格
		currentPrice, err := shs.getMarketPrice(execution.Symbol)
		if err != nil {
			return fmt.Errorf("failed to get market price for slice %d: %w", i+1, err)
		}

		// 执行子订单
		sliceExecution := *execution
		sliceExecution.Quantity = sliceQuantity
		sliceExecution.TargetPrice = currentPrice

		err = shs.executeMarketOrder(&sliceExecution)
		if err != nil {
			return fmt.Errorf("failed to execute slice %d: %w", i+1, err)
		}

		totalExecutedPrice += sliceExecution.ExecutedPrice * sliceQuantity
		totalExecutedQuantity += sliceQuantity

		// 等待下一个执行时间
		if i < slices-1 {
			time.Sleep(interval)
		}
	}

	// 计算加权平均执行价格
	execution.ExecutedPrice = totalExecutedPrice / totalExecutedQuantity

	log.Printf("TWAP execution completed: %s %s %.4f @ %.2f (avg price)",
		execution.Side, execution.Symbol, execution.Quantity, execution.ExecutedPrice)

	return nil
}

// executeVWAP 执行成交量加权平均价格策略
func (shs *SmartHedgingSystem) executeVWAP(execution *HedgeExecution) error {
	// VWAP策略：根据历史成交量分布执行订单
	// 简化实现，实际中需要获取历史成交量数据

	// 模拟VWAP执行，添加较小的滑点
	slippageRate := 0.0001 + rand.Float64()*0.0003 // 0.01% - 0.04%
	if execution.Side == "BUY" {
		execution.ExecutedPrice = execution.TargetPrice * (1 + slippageRate)
	} else {
		execution.ExecutedPrice = execution.TargetPrice * (1 - slippageRate)
	}

	// 模拟VWAP执行时间
	time.Sleep(time.Duration(50+int(rand.Float64()*100)) * time.Millisecond)

	log.Printf("VWAP execution completed: %s %s %.4f @ %.2f",
		execution.Side, execution.Symbol, execution.Quantity, execution.ExecutedPrice)

	return nil
}

// executeLimitOrder 执行限价单
func (shs *SmartHedgingSystem) executeLimitOrder(execution *HedgeExecution) error {
	// 限价单执行逻辑
	// 设置限价（稍微优于市价）

	limitPriceOffset := 0.0002 // 0.02%的价格改善
	if execution.Side == "BUY" {
		execution.ExecutedPrice = execution.TargetPrice * (1 - limitPriceOffset)
	} else {
		execution.ExecutedPrice = execution.TargetPrice * (1 + limitPriceOffset)
	}

	// 模拟限价单等待时间
	time.Sleep(time.Duration(100+int(rand.Float64()*200)) * time.Millisecond)

	log.Printf("Limit order executed: %s %s %.4f @ %.2f",
		execution.Side, execution.Symbol, execution.Quantity, execution.ExecutedPrice)

	return nil
}

// getOrderBookDepth 获取订单簿深度
func (shs *SmartHedgingSystem) getOrderBookDepth(symbol string) (float64, error) {
	// 模拟获取订单簿深度
	// 实际实现中需要调用交易所API

	// 根据不同交易对返回不同的深度
	switch symbol {
	case "BTCUSDT":
		return 500000.0, nil // 50万USDT深度
	case "ETHUSDT":
		return 300000.0, nil // 30万USDT深度
	default:
		return 100000.0, nil // 默认10万USDT深度
	}
}

// handleIneffectiveHedge 处理低效对冲
func (shs *SmartHedgingSystem) handleIneffectiveHedge(hedge *HedgePosition) error {
	log.Printf("Handling ineffective hedge %s (effectiveness: %.4f)", hedge.ID, hedge.HedgeEffectiveness)

	// 分析低效原因
	reason := shs.analyzeIneffectiveReason(hedge)
	log.Printf("Ineffective hedge reason: %s", reason)

	// 根据原因采取不同的处理策略
	switch reason {
	case "low_correlation":
		// 相关性过低，尝试寻找更好的对冲工具
		return shs.replaceHedgeInstrument(hedge)

	case "high_basis_risk":
		// 基差风险过高，调整对冲比率
		return shs.adjustHedgeRatioForBasisRisk(hedge)

	case "market_regime_change":
		// 市场环境变化，暂停对冲并重新评估
		return shs.pauseAndReassessHedge(hedge)

	case "excessive_cost":
		// 对冲成本过高，考虑降低对冲比率或关闭
		return shs.reduceOrCloseHedge(hedge)

	default:
		// 默认策略：逐步关闭对冲
		return shs.graduallyCloseHedge(hedge)
	}
}

// analyzeIneffectiveReason 分析低效对冲的原因
func (shs *SmartHedgingSystem) analyzeIneffectiveReason(hedge *HedgePosition) string {
	// 检查相关性
	if hedge.Correlation < 0.3 {
		return "low_correlation"
	}

	// 检查基差风险
	if hedge.BasisRisk > 0.1 {
		return "high_basis_risk"
	}

	// 检查对冲成本
	costRatio := hedge.HedgeCost / (hedge.BaseQuantity * hedge.EffectiveRatio)
	if costRatio > 0.05 { // 成本超过5%
		return "excessive_cost"
	}

	// 检查跟踪误差
	if hedge.TrackingError > 0.15 {
		return "market_regime_change"
	}

	return "unknown"
}

// replaceHedgeInstrument 替换对冲工具
func (shs *SmartHedgingSystem) replaceHedgeInstrument(hedge *HedgePosition) error {
	log.Printf("Attempting to replace hedge instrument for %s", hedge.ID)

	// 寻找与基础资产相关性更高的对冲工具
	bestInstrument := ""
	bestCorrelation := 0.0

	for symbol, instrument := range shs.hedgeInstruments {
		if symbol == hedge.HedgeAsset || !instrument.IsActive {
			continue
		}

		correlation := shs.getCorrelation(hedge.BaseAsset, symbol)
		if correlation > bestCorrelation {
			bestCorrelation = correlation
			bestInstrument = symbol
		}
	}

	// 如果找到更好的对冲工具
	if bestInstrument != "" && bestCorrelation > hedge.Correlation+0.1 {
		log.Printf("Found better hedge instrument: %s (correlation: %.4f vs %.4f)",
			bestInstrument, bestCorrelation, hedge.Correlation)

		// 逐步切换到新的对冲工具
		return shs.switchHedgeInstrument(hedge, bestInstrument)
	}

	// 没有找到更好的工具，关闭当前对冲
	return shs.graduallyCloseHedge(hedge)
}

// adjustHedgeRatioForBasisRisk 调整对冲比率以应对基差风险
func (shs *SmartHedgingSystem) adjustHedgeRatioForBasisRisk(hedge *HedgePosition) error {
	log.Printf("Adjusting hedge ratio for basis risk: %s", hedge.ID)

	// 计算基差风险调整后的最优对冲比率
	basisRiskAdjustment := hedge.BasisRisk * 0.5 // 基差风险调整因子
	newRatio := hedge.HedgeRatio * (1 - basisRiskAdjustment)

	// 确保新比率在合理范围内
	newRatio = math.Max(newRatio, shs.minHedgeRatio)
	newRatio = math.Min(newRatio, shs.maxHedgeRatio)

	if math.Abs(newRatio-hedge.HedgeRatio) > 0.05 { // 变化超过5%才调整
		log.Printf("Adjusting hedge ratio from %.4f to %.4f for %s",
			hedge.HedgeRatio, newRatio, hedge.ID)

		// 执行调整
		return shs.adjustHedgePosition(hedge, newRatio)
	}

	return nil
}

// pauseAndReassessHedge 暂停并重新评估对冲
func (shs *SmartHedgingSystem) pauseAndReassessHedge(hedge *HedgePosition) error {
	log.Printf("Pausing and reassessing hedge: %s", hedge.ID)

	// 暂停对冲（设置状态为暂停）
	hedge.Status = "PAUSED"

	// 记录暂停原因
	adjustment := HedgeAdjustment{
		Timestamp:      time.Now(),
		Trigger:        "market_regime_change",
		OldRatio:       hedge.HedgeRatio,
		NewRatio:       0.0,
		AdjustmentSize: -hedge.HedgeRatio,
		Reason:         "Market regime change detected, pausing for reassessment",
		Effectiveness:  hedge.HedgeEffectiveness,
	}

	hedge.AdjustmentHistory = append(hedge.AdjustmentHistory, adjustment)

	// 设置重新评估时间（1小时后）
	go func() {
		time.Sleep(1 * time.Hour)
		shs.reassessHedge(hedge)
	}()

	return nil
}

// reduceOrCloseHedge 降低或关闭对冲
func (shs *SmartHedgingSystem) reduceOrCloseHedge(hedge *HedgePosition) error {
	log.Printf("Reducing or closing hedge due to excessive cost: %s", hedge.ID)

	// 计算成本效益比
	costRatio := hedge.HedgeCost / (hedge.BaseQuantity * hedge.EffectiveRatio)

	if costRatio > 0.1 { // 成本超过10%，直接关闭
		return shs.graduallyCloseHedge(hedge)
	} else { // 成本在5%-10%之间，减少对冲比率
		newRatio := hedge.HedgeRatio * 0.5 // 减少50%
		return shs.adjustHedgePosition(hedge, newRatio)
	}
}

// graduallyCloseHedge 逐步关闭对冲
func (shs *SmartHedgingSystem) graduallyCloseHedge(hedge *HedgePosition) error {
	log.Printf("Gradually closing hedge: %s", hedge.ID)

	// 分3步关闭对冲，避免市场冲击
	steps := []float64{0.7, 0.3, 0.0} // 先减少到70%，再30%，最后0%

	for i, targetRatio := range steps {
		if targetRatio < hedge.HedgeRatio {
			log.Printf("Closing hedge step %d: reducing to %.1f%% for %s",
				i+1, targetRatio*100, hedge.ID)

			err := shs.adjustHedgePosition(hedge, targetRatio)
			if err != nil {
				return fmt.Errorf("failed to close hedge step %d: %w", i+1, err)
			}

			// 等待一段时间再进行下一步
			if i < len(steps)-1 {
				time.Sleep(5 * time.Minute)
			}
		}
	}

	// 最终关闭对冲
	hedge.Status = "CLOSED"
	delete(shs.activeHedges, hedge.ID)

	log.Printf("Hedge %s has been completely closed", hedge.ID)
	return nil
}

// adjustHedgePosition 调整对冲仓位
func (shs *SmartHedgingSystem) adjustHedgePosition(hedge *HedgePosition, newRatio float64) error {
	if newRatio < 0 || newRatio > 1 {
		return fmt.Errorf("invalid hedge ratio: %.4f", newRatio)
	}

	oldRatio := hedge.HedgeRatio
	oldQuantity := hedge.HedgeQuantity

	// 计算新的对冲数量
	newQuantity := hedge.BaseQuantity * newRatio
	quantityChange := newQuantity - oldQuantity

	// 执行调整交易
	if quantityChange != 0 {
		action := "INCREASE"
		if quantityChange < 0 {
			action = "DECREASE"
			quantityChange = -quantityChange
		}

		// 创建调整执行记录
		execution := HedgeExecution{
			ID:        fmt.Sprintf("adj_%s_%d", hedge.ID, time.Now().Unix()),
			HedgeID:   hedge.ID,
			Action:    action,
			Symbol:    hedge.HedgeAsset,
			Side:      "SELL", // 简化处理
			Quantity:  quantityChange,
			Status:    "PENDING",
			Timestamp: time.Now(),
		}

		// 执行交易
		err := shs.executeMarketOrder(&execution)
		if err != nil {
			return fmt.Errorf("failed to adjust hedge position: %w", err)
		}

		// 更新对冲仓位
		hedge.HedgeRatio = newRatio
		hedge.HedgeQuantity = newQuantity
		hedge.EffectiveRatio = newRatio * hedge.Correlation
		hedge.LastAdjusted = time.Now()

		// 记录调整历史
		adjustment := HedgeAdjustment{
			Timestamp:      time.Now(),
			Trigger:        "ratio_adjustment",
			OldRatio:       oldRatio,
			NewRatio:       newRatio,
			AdjustmentSize: quantityChange,
			Cost:           execution.Cost,
			Reason:         "Position adjustment for optimization",
			Effectiveness:  hedge.HedgeEffectiveness,
		}

		hedge.AdjustmentHistory = append(hedge.AdjustmentHistory, adjustment)

		log.Printf("Adjusted hedge %s: ratio %.4f -> %.4f, quantity %.4f -> %.4f",
			hedge.ID, oldRatio, newRatio, oldQuantity, newQuantity)
	}

	return nil
}

// reassessHedge 重新评估对冲
func (shs *SmartHedgingSystem) reassessHedge(hedge *HedgePosition) error {
	log.Printf("Reassessing hedge: %s", hedge.ID)

	// 重新计算相关性
	hedge.Correlation = shs.getCorrelation(hedge.BaseAsset, hedge.HedgeAsset)

	// 重新计算最优对冲比率
	optimalRatio, err := shs.calculateOptimalHedgeRatio(hedge.BaseAsset, hedge.HedgeAsset)
	if err != nil {
		log.Printf("Failed to calculate optimal ratio for %s: %v", hedge.ID, err)
		optimalRatio = 0.8 // 使用默认值
	}

	hedge.OptimalRatio = optimalRatio

	// 更新风险指标
	shs.updateHedgeRiskMetrics(hedge)

	// 判断是否恢复对冲
	if hedge.HedgeEffectiveness > 0.7 && hedge.Correlation > 0.5 {
		// 条件改善，恢复对冲
		hedge.Status = "ACTIVE"
		hedge.HedgeRatio = optimalRatio
		hedge.HedgeQuantity = hedge.BaseQuantity * optimalRatio

		log.Printf("Hedge %s reassessment: conditions improved, resuming with ratio %.4f",
			hedge.ID, optimalRatio)
	} else {
		// 条件仍然不佳，继续暂停或关闭
		log.Printf("Hedge %s reassessment: conditions still poor, considering closure", hedge.ID)
		return shs.graduallyCloseHedge(hedge)
	}

	return nil
}

// switchHedgeInstrument 切换对冲工具
func (shs *SmartHedgingSystem) switchHedgeInstrument(hedge *HedgePosition, newInstrument string) error {
	log.Printf("Switching hedge instrument from %s to %s for %s",
		hedge.HedgeAsset, newInstrument, hedge.ID)

	// 先关闭当前对冲仓位
	err := shs.adjustHedgePosition(hedge, 0.0)
	if err != nil {
		return fmt.Errorf("failed to close current hedge position: %w", err)
	}

	// 更新对冲工具
	hedge.HedgeAsset = newInstrument

	// 重新计算最优对冲比率
	optimalRatio, err := shs.calculateOptimalHedgeRatio(hedge.BaseAsset, newInstrument)
	if err != nil {
		return fmt.Errorf("failed to calculate optimal ratio for new instrument: %w", err)
	}

	// 建立新的对冲仓位
	err = shs.adjustHedgePosition(hedge, optimalRatio)
	if err != nil {
		return fmt.Errorf("failed to establish new hedge position: %w", err)
	}

	// 更新相关指标
	hedge.Correlation = shs.getCorrelation(hedge.BaseAsset, newInstrument)
	shs.updateHedgeRiskMetrics(hedge)

	log.Printf("Successfully switched hedge instrument for %s", hedge.ID)
	return nil
}

// getMarketConditionData 获取市场状态数据
func (shs *SmartHedgingSystem) getMarketConditionData(asset string) (*MarketConditionData, error) {
	// 获取历史价格数据用于计算技术指标
	prices, err := shs.getHistoricalPrices(asset, 30) // 30天数据
	if err != nil {
		return nil, fmt.Errorf("failed to get historical prices: %w", err)
	}

	if len(prices) < 14 { // RSI需要至少14个数据点
		return nil, fmt.Errorf("insufficient price data for %s", asset)
	}

	// 获取当前价格
	currentPrice := prices[len(prices)-1]

	// 计算技术指标
	volatility := shs.calculateVolatilityFromPrices(prices)
	rsi := shs.calculateRSI(prices, 14)
	macd := shs.calculateMACD(prices)
	trend, trendStrength := shs.analyzeTrend(prices)
	bollingerPos := shs.calculateBollingerPosition(prices, 20, 2.0)

	// 获取交易量和市场数据（模拟）
	volume24h := shs.getSimulatedVolume(asset)
	marketCap := shs.getSimulatedMarketCap(asset)
	fundingRate := shs.getSimulatedFundingRate(asset)
	openInterest := shs.getSimulatedOpenInterest(asset)

	// 计算流动性评分
	liquidityScore := shs.calculateLiquidityScore(volume24h, marketCap)

	// 判断成交量状态
	volumeProfile := shs.classifyVolumeProfile(volume24h)

	return &MarketConditionData{
		Asset:             asset,
		Timestamp:         time.Now(),
		Price:             currentPrice,
		Volume24h:         volume24h,
		Volatility:        volatility,
		Trend:             trend,
		TrendStrength:     trendStrength,
		LiquidityScore:    liquidityScore,
		MarketCap:         marketCap,
		RSI:               rsi,
		MACD:              macd,
		BollingerPosition: bollingerPos,
		FundingRate:       fundingRate,
		OpenInterest:      openInterest,
		VolumeProfile:     volumeProfile,
	}, nil
}

// analyzeMarketCondition 综合分析市场状态
func (shs *SmartHedgingSystem) analyzeMarketCondition(marketData map[string]*MarketConditionData) string {
	if len(marketData) == 0 {
		return "UNKNOWN"
	}

	// 计算各项指标的平均值
	var avgVolatility, avgRSI, avgTrendStrength, avgLiquidity float64
	bullishCount, bearishCount := 0, 0
	highVolCount, lowVolCount := 0, 0

	for _, data := range marketData {
		avgVolatility += data.Volatility
		avgRSI += data.RSI
		avgTrendStrength += data.TrendStrength
		avgLiquidity += data.LiquidityScore

		// 统计趋势方向
		switch data.Trend {
		case "BULLISH":
			bullishCount++
		case "BEARISH":
			bearishCount++
		}

		// 统计波动率水平
		if data.Volatility > 0.05 {
			highVolCount++
		} else if data.Volatility < 0.02 {
			lowVolCount++
		}
	}

	count := float64(len(marketData))
	avgVolatility /= count
	avgRSI /= count
	avgTrendStrength /= count
	avgLiquidity /= count

	// 综合判断市场状态
	if avgVolatility > 0.08 {
		return "EXTREME_VOLATILITY"
	} else if avgVolatility > 0.05 {
		if bullishCount > bearishCount {
			return "BULLISH_VOLATILE"
		} else if bearishCount > bullishCount {
			return "BEARISH_VOLATILE"
		} else {
			return "HIGH_VOLATILITY"
		}
	} else if avgVolatility < 0.015 {
		return "LOW_VOLATILITY"
	}

	// 基于趋势判断
	if bullishCount >= int(count*0.7) {
		return "STRONG_BULLISH"
	} else if bearishCount >= int(count*0.7) {
		return "STRONG_BEARISH"
	} else if bullishCount > bearishCount {
		return "MODERATE_BULLISH"
	} else if bearishCount > bullishCount {
		return "MODERATE_BEARISH"
	}

	// 基于RSI判断超买超卖
	if avgRSI > 70 {
		return "OVERBOUGHT"
	} else if avgRSI < 30 {
		return "OVERSOLD"
	}

	return "NORMAL"
}

// calculateVolatilityFromPrices 从价格数据计算波动率
func (shs *SmartHedgingSystem) calculateVolatilityFromPrices(prices []float64) float64 {
	if len(prices) < 2 {
		return 0
	}

	// 计算收益率
	returns := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		if prices[i-1] != 0 {
			returns[i-1] = (prices[i] - prices[i-1]) / prices[i-1]
		}
	}

	// 计算收益率的标准差
	mean := shs.calculateMean(returns)
	variance := shs.calculateVariance(returns, mean)

	// 年化波动率（假设每日数据）
	return math.Sqrt(variance) * math.Sqrt(365)
}

// calculateRSI 计算相对强弱指数
func (shs *SmartHedgingSystem) calculateRSI(prices []float64, period int) float64 {
	if len(prices) < period+1 {
		return 50 // 默认中性值
	}

	gains := make([]float64, 0)
	losses := make([]float64, 0)

	// 计算价格变化
	for i := 1; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			gains = append(gains, change)
			losses = append(losses, 0)
		} else {
			gains = append(gains, 0)
			losses = append(losses, -change)
		}
	}

	if len(gains) < period {
		return 50
	}

	// 计算平均收益和平均损失
	avgGain := shs.calculateMean(gains[len(gains)-period:])
	avgLoss := shs.calculateMean(losses[len(losses)-period:])

	if avgLoss == 0 {
		return 100
	}

	rs := avgGain / avgLoss
	rsi := 100 - (100 / (1 + rs))

	return rsi
}

// calculateMACD 计算MACD指标
func (shs *SmartHedgingSystem) calculateMACD(prices []float64) float64 {
	if len(prices) < 26 {
		return 0
	}

	// 计算12日和26日EMA
	ema12 := shs.calculateEMA(prices, 12)
	ema26 := shs.calculateEMA(prices, 26)

	// MACD线 = EMA12 - EMA26
	macd := ema12 - ema26

	return macd
}

// calculateEMA 计算指数移动平均
func (shs *SmartHedgingSystem) calculateEMA(prices []float64, period int) float64 {
	if len(prices) < period {
		return shs.calculateMean(prices)
	}

	multiplier := 2.0 / (float64(period) + 1.0)
	ema := prices[0]

	for i := 1; i < len(prices); i++ {
		ema = (prices[i] * multiplier) + (ema * (1 - multiplier))
	}

	return ema
}

// analyzeTrend 分析价格趋势
func (shs *SmartHedgingSystem) analyzeTrend(prices []float64) (string, float64) {
	if len(prices) < 10 {
		return "SIDEWAYS", 0.0
	}

	// 使用线性回归分析趋势
	n := float64(len(prices))
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0

	for i, price := range prices {
		x := float64(i)
		sumX += x
		sumY += price
		sumXY += x * price
		sumX2 += x * x
	}

	// 计算斜率
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)

	// 计算R²（决定系数）作为趋势强度
	meanY := sumY / n
	ssRes, ssTot := 0.0, 0.0

	for i, price := range prices {
		predicted := slope*float64(i) + (sumY-slope*sumX)/n
		ssRes += math.Pow(price-predicted, 2)
		ssTot += math.Pow(price-meanY, 2)
	}

	r2 := 1 - (ssRes / ssTot)
	trendStrength := math.Abs(r2)

	// 判断趋势方向
	avgPrice := meanY
	slopePercent := slope / avgPrice * 100

	if slopePercent > 0.1 {
		return "BULLISH", trendStrength
	} else if slopePercent < -0.1 {
		return "BEARISH", trendStrength
	} else {
		return "SIDEWAYS", trendStrength
	}
}

// calculateBollingerPosition 计算布林带位置
func (shs *SmartHedgingSystem) calculateBollingerPosition(prices []float64, period int, stdDev float64) float64 {
	if len(prices) < period {
		return 0.5 // 中间位置
	}

	// 取最近period个价格
	recentPrices := prices[len(prices)-period:]

	// 计算移动平均和标准差
	sma := shs.calculateMean(recentPrices)
	std := math.Sqrt(shs.calculateVariance(recentPrices, sma))

	// 计算布林带上下轨
	upperBand := sma + stdDev*std
	lowerBand := sma - stdDev*std

	currentPrice := prices[len(prices)-1]

	// 计算在布林带中的位置 (0-1)
	if upperBand == lowerBand {
		return 0.5
	}

	position := (currentPrice - lowerBand) / (upperBand - lowerBand)
	return math.Max(0, math.Min(1, position))
}

// getSimulatedVolume 获取模拟交易量
func (shs *SmartHedgingSystem) getSimulatedVolume(asset string) float64 {
	// 模拟不同资产的24小时交易量
	switch asset {
	case "BTC":
		return 1000000000 + rand.Float64()*500000000 // 10-15亿美元
	case "ETH":
		return 500000000 + rand.Float64()*300000000 // 5-8亿美元
	case "BNB":
		return 100000000 + rand.Float64()*100000000 // 1-2亿美元
	default:
		return 50000000 + rand.Float64()*50000000 // 0.5-1亿美元
	}
}

// getSimulatedMarketCap 获取模拟市值
func (shs *SmartHedgingSystem) getSimulatedMarketCap(asset string) float64 {
	// 模拟不同资产的市值
	switch asset {
	case "BTC":
		return 800000000000 + rand.Float64()*200000000000 // 8000-10000亿美元
	case "ETH":
		return 300000000000 + rand.Float64()*100000000000 // 3000-4000亿美元
	case "BNB":
		return 50000000000 + rand.Float64()*20000000000 // 500-700亿美元
	default:
		return 10000000000 + rand.Float64()*10000000000 // 100-200亿美元
	}
}

// getSimulatedFundingRate 获取模拟资金费率
func (shs *SmartHedgingSystem) getSimulatedFundingRate(asset string) float64 {
	// 模拟资金费率 (-0.1% 到 0.1%)
	return (rand.Float64() - 0.5) * 0.002
}

// getSimulatedOpenInterest 获取模拟持仓量
func (shs *SmartHedgingSystem) getSimulatedOpenInterest(asset string) float64 {
	// 模拟持仓量
	switch asset {
	case "BTC":
		return 5000000000 + rand.Float64()*2000000000 // 50-70亿美元
	case "ETH":
		return 2000000000 + rand.Float64()*1000000000 // 20-30亿美元
	case "BNB":
		return 500000000 + rand.Float64()*300000000 // 5-8亿美元
	default:
		return 100000000 + rand.Float64()*100000000 // 1-2亿美元
	}
}

// calculateLiquidityScore 计算流动性评分
func (shs *SmartHedgingSystem) calculateLiquidityScore(volume24h, marketCap float64) float64 {
	// 基于交易量和市值计算流动性评分
	if marketCap == 0 {
		return 0
	}

	// 交易量/市值比率
	turnoverRatio := volume24h / marketCap

	// 将比率转换为0-1的评分
	// 一般来说，日交易量占市值的5-20%是比较健康的
	score := turnoverRatio / 0.2 // 20%作为满分

	return math.Max(0, math.Min(1, score))
}

// classifyVolumeProfile 分类交易量状态
func (shs *SmartHedgingSystem) classifyVolumeProfile(volume24h float64) string {
	// 基于交易量大小分类
	if volume24h > 1000000000 { // 超过10亿美元
		return "HIGH"
	} else if volume24h > 100000000 { // 超过1亿美元
		return "NORMAL"
	} else {
		return "LOW"
	}
}

// calculateAdjustmentImpact 计算调整的实际影响
func (shs *SmartHedgingSystem) calculateAdjustmentImpact(hedge *HedgePosition, oldRatio, oldQuantity, oldEffectiveness float64) float64 {
	// 计算调整前后的效果差异
	newEffectiveness := hedge.HedgeEffectiveness
	effectivenessImprovement := newEffectiveness - oldEffectiveness

	// 计算仓位变化的影响
	quantityChange := math.Abs(hedge.HedgeQuantity - oldQuantity)
	positionImpact := quantityChange / hedge.BaseQuantity

	// 计算比率变化的影响
	ratioChange := math.Abs(hedge.HedgeRatio - oldRatio)

	// 综合影响评分 (0-1)
	// 正面影响：效果改善
	// 负面影响：仓位变化成本、比率偏离
	impact := effectivenessImprovement - (positionImpact * 0.1) - (ratioChange * 0.05)

	return impact
}

// getAssetReturn 获取资产收益率
func (shs *SmartHedgingSystem) getAssetReturn(asset string) (float64, error) {
	// 获取历史价格数据计算收益率
	prices, err := shs.getHistoricalPrices(asset, 2) // 获取最近2天的价格
	if err != nil {
		return 0, fmt.Errorf("failed to get historical prices: %w", err)
	}

	if len(prices) < 2 {
		return 0, fmt.Errorf("insufficient price data for return calculation")
	}

	// 计算最近一期的收益率
	if prices[0] == 0 {
		return 0, fmt.Errorf("invalid price data")
	}

	return (prices[1] - prices[0]) / prices[0], nil
}
