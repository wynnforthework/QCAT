package management

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"sort"
	"sync"
	"time"

	"qcat/internal/config"
)

// LayeredPositionManager 分层仓位管理器
type LayeredPositionManager struct {
	config            *config.Config
	positionAllocator *PositionAllocator
	rebalancer        *Rebalancer
	riskManager       *LayeredRiskManager

	// 运行状态
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	isRunning bool
	mu        sync.RWMutex

	// 分层配置
	layers             []PositionLayer
	totalFunds         float64
	rebalanceThreshold float64

	// 仓位状态
	currentPositions map[string]*Position
	layerAllocations map[int]*LayerAllocation

	// 监控指标
	managementMetrics *ManagementMetrics
	allocationHistory []AllocationSnapshot

	// 配置参数
	layerCount        int
	layerSizes        []float64
	rebalanceInterval time.Duration
	enabled           bool
}

// PositionLayer 仓位层级
type PositionLayer struct {
	ID            int              `json:"id"`
	Name          string           `json:"name"`
	AllocationPct float64          `json:"allocation_pct"`
	RiskLevel     string           `json:"risk_level"`
	Strategy      string           `json:"strategy"`
	MaxLeverage   float64          `json:"max_leverage"`
	MaxDrawdown   float64          `json:"max_drawdown"`
	AllowedAssets []string         `json:"allowed_assets"`
	Constraints   LayerConstraints `json:"constraints"`
	Performance   LayerPerformance `json:"performance"`
}

// LayerConstraints 层级约束
type LayerConstraints struct {
	MaxPositionSize         float64  `json:"max_position_size"`
	MinPositionSize         float64  `json:"min_position_size"`
	MaxAssetConcentration   float64  `json:"max_asset_concentration"`
	MaxSectorConcentration  float64  `json:"max_sector_concentration"`
	RequiredDiversification int      `json:"required_diversification"`
	AllowedInstruments      []string `json:"allowed_instruments"`
	ForbiddenAssets         []string `json:"forbidden_assets"`
}

// LayerPerformance 层级表现
type LayerPerformance struct {
	mu sync.RWMutex

	TotalReturn      float64 `json:"total_return"`
	AnnualizedReturn float64 `json:"annualized_return"`
	Volatility       float64 `json:"volatility"`
	SharpeRatio      float64 `json:"sharpe_ratio"`
	MaxDrawdown      float64 `json:"max_drawdown"`
	CalmarRatio      float64 `json:"calmar_ratio"`
	WinRate          float64 `json:"win_rate"`
	ProfitFactor     float64 `json:"profit_factor"`

	LastUpdated time.Time `json:"last_updated"`
}

// Position 仓位信息
type Position struct {
	Symbol       string    `json:"symbol"`
	LayerID      int       `json:"layer_id"`
	Quantity     float64   `json:"quantity"`
	Price        float64   `json:"price"`
	Value        float64   `json:"value"`
	Weight       float64   `json:"weight"`
	Side         string    `json:"side"` // LONG, SHORT
	Leverage     float64   `json:"leverage"`
	Margin       float64   `json:"margin"`
	UnrealizedPL float64   `json:"unrealized_pl"`
	RealizedPL   float64   `json:"realized_pl"`
	OpenTime     time.Time `json:"open_time"`
	LastUpdate   time.Time `json:"last_update"`
	Status       string    `json:"status"` // ACTIVE, CLOSING, CLOSED
}

// LayerAllocation 层级分配
type LayerAllocation struct {
	LayerID         int                  `json:"layer_id"`
	AllocatedFunds  float64              `json:"allocated_funds"`
	UsedFunds       float64              `json:"used_funds"`
	AvailableFunds  float64              `json:"available_funds"`
	Positions       map[string]*Position `json:"positions"`
	Performance     LayerPerformance     `json:"performance"`
	RiskMetrics     LayerRiskMetrics     `json:"risk_metrics"`
	LastRebalance   time.Time            `json:"last_rebalance"`
	RebalanceNeeded bool                 `json:"rebalance_needed"`
}

// LayerRiskMetrics 层级风险指标
type LayerRiskMetrics struct {
	CurrentVaR        float64                       `json:"current_var"`
	ExpectedShortfall float64                       `json:"expected_shortfall"`
	BetaToMarket      float64                       `json:"beta_to_market"`
	CorrelationMatrix map[string]map[string]float64 `json:"correlation_matrix"`
	ConcentrationRisk float64                       `json:"concentration_risk"`
	LeverageRatio     float64                       `json:"leverage_ratio"`
	LiquidityRisk     float64                       `json:"liquidity_risk"`
}

// PositionAllocator 仓位分配器
type PositionAllocator struct {
	allocationModel string
	riskBudget      float64
	optimizer       *AllocationOptimizer
	mu              sync.RWMutex
}

// AllocationOptimizer 分配优化器
type AllocationOptimizer struct {
	algorithm         string
	objective         string
	constraints       []OptimizationConstraint
	maxIterations     int
	convergenceThresh float64
}

// OptimizationConstraint 优化约束
type OptimizationConstraint struct {
	Type        string  `json:"type"`
	Parameter   string  `json:"parameter"`
	Operator    string  `json:"operator"`
	Value       float64 `json:"value"`
	Description string  `json:"description"`
}

// Rebalancer 再平衡器
type Rebalancer struct {
	strategy         string
	threshold        float64
	frequency        time.Duration
	costModel        string
	lastRebalance    time.Time
	rebalanceHistory []RebalanceEvent
	mu               sync.RWMutex
}

// RebalanceEvent 再平衡事件
type RebalanceEvent struct {
	ID              string                 `json:"id"`
	Timestamp       time.Time              `json:"timestamp"`
	Type            string                 `json:"type"`
	Trigger         string                 `json:"trigger"`
	LayersAffected  []int                  `json:"layers_affected"`
	Changes         []PositionChange       `json:"changes"`
	TotalCost       float64                `json:"total_cost"`
	ExpectedBenefit float64                `json:"expected_benefit"`
	ActualBenefit   float64                `json:"actual_benefit"`
	Success         bool                   `json:"success"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// PositionChange 仓位变化
type PositionChange struct {
	Symbol     string  `json:"symbol"`
	LayerID    int     `json:"layer_id"`
	OldWeight  float64 `json:"old_weight"`
	NewWeight  float64 `json:"new_weight"`
	ChangeType string  `json:"change_type"` // ADD, REMOVE, ADJUST
	Quantity   float64 `json:"quantity"`
	Price      float64 `json:"price"`
	Cost       float64 `json:"cost"`
	Reason     string  `json:"reason"`
}

// LayeredRiskManager 分层风险管理器
type LayeredRiskManager struct {
	riskLimits       map[int]RiskLimit
	correlationModel string
	stressScenarios  []StressScenario
	monitoringRules  []RiskRule
	mu               sync.RWMutex
}

// RiskLimit 风险限制
type RiskLimit struct {
	LayerID            int     `json:"layer_id"`
	MaxVaR             float64 `json:"max_var"`
	MaxDrawdown        float64 `json:"max_drawdown"`
	MaxLeverage        float64 `json:"max_leverage"`
	MaxConcentration   float64 `json:"max_concentration"`
	MaxCorrelation     float64 `json:"max_correlation"`
	MinDiversification int     `json:"min_diversification"`
}

// StressScenario 压力测试场景
type StressScenario struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Type        string             `json:"type"`
	Parameters  map[string]float64 `json:"parameters"`
	Description string             `json:"description"`
	Severity    string             `json:"severity"`
}

// RiskRule 风险规则
type RiskRule struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Condition  string                 `json:"condition"`
	Action     string                 `json:"action"`
	Priority   int                    `json:"priority"`
	IsEnabled  bool                   `json:"is_enabled"`
	Parameters map[string]interface{} `json:"parameters"`
}

// ManagementMetrics 管理指标
type ManagementMetrics struct {
	mu sync.RWMutex

	// 分配效率
	AllocationEfficiency float64 `json:"allocation_efficiency"`
	RebalanceFrequency   float64 `json:"rebalance_frequency"`
	AverageRebalanceCost float64 `json:"average_rebalance_cost"`

	// 风险管理
	RiskAdjustedReturn float64 `json:"risk_adjusted_return"`
	TrackingError      float64 `json:"tracking_error"`
	InformationRatio   float64 `json:"information_ratio"`

	// 层级表现
	LayerPerformances    map[int]LayerPerformance `json:"layer_performances"`
	BestPerformingLayer  int                      `json:"best_performing_layer"`
	WorstPerformingLayer int                      `json:"worst_performing_layer"`

	// 系统指标
	TotalPositions   int       `json:"total_positions"`
	ActiveLayers     int       `json:"active_layers"`
	LastOptimization time.Time `json:"last_optimization"`

	LastUpdated time.Time `json:"last_updated"`
}

// AllocationSnapshot 分配快照
type AllocationSnapshot struct {
	Timestamp        time.Time                `json:"timestamp"`
	TotalFunds       float64                  `json:"total_funds"`
	LayerAllocations map[int]float64          `json:"layer_allocations"`
	Positions        map[string]Position      `json:"positions"`
	RiskMetrics      map[int]LayerRiskMetrics `json:"risk_metrics"`
	Performance      map[int]LayerPerformance `json:"performance"`
	MarketConditions map[string]float64       `json:"market_conditions"`
}

// NewLayeredPositionManager 创建分层仓位管理器
func NewLayeredPositionManager(cfg *config.Config) (*LayeredPositionManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	lpm := &LayeredPositionManager{
		config:            cfg,
		positionAllocator: NewPositionAllocator(),
		rebalancer:        NewRebalancer(),
		riskManager:       NewLayeredRiskManager(),
		ctx:               ctx,
		cancel:            cancel,
		currentPositions:  make(map[string]*Position),
		layerAllocations:  make(map[int]*LayerAllocation),
		managementMetrics: &ManagementMetrics{
			LayerPerformances: make(map[int]LayerPerformance),
		},
		allocationHistory:  make([]AllocationSnapshot, 0),
		layerCount:         3,
		layerSizes:         []float64{0.4, 0.35, 0.25},
		rebalanceThreshold: 0.05,
		rebalanceInterval:  24 * time.Hour,
		enabled:            true,
	}

	// 从配置文件读取参数
	if cfg != nil {
		if cfg.LayeredPosition.Enabled {
			lpm.enabled = cfg.LayeredPosition.Enabled
		}
		if cfg.LayeredPosition.LayerCount > 0 {
			lpm.layerCount = cfg.LayeredPosition.LayerCount
		}
		if len(cfg.LayeredPosition.LayerSizes) > 0 {
			lpm.layerSizes = cfg.LayeredPosition.LayerSizes
		}
		if cfg.LayeredPosition.RebalanceThreshold > 0 {
			lpm.rebalanceThreshold = cfg.LayeredPosition.RebalanceThreshold
		}
		if cfg.LayeredPosition.RebalanceInterval > 0 {
			lpm.rebalanceInterval = cfg.LayeredPosition.RebalanceInterval
		}

		// 更新仓位分配器配置
		if lpm.positionAllocator != nil && cfg.LayeredPosition.AllocationMethod != "" {
			lpm.positionAllocator.allocationModel = cfg.LayeredPosition.AllocationMethod
		}

		// 更新再平衡器配置
		if lpm.rebalancer != nil {
			lpm.rebalancer.threshold = cfg.LayeredPosition.RebalanceThreshold
			lpm.rebalancer.frequency = cfg.LayeredPosition.RebalanceInterval
		}

		log.Printf("Loaded layered position configuration: layers=%d, rebalance_threshold=%.2f, interval=%v",
			lpm.layerCount, lpm.rebalanceThreshold, lpm.rebalanceInterval)
	}

	// 初始化层级
	err := lpm.initializeLayers()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize layers: %w", err)
	}

	return lpm, nil
}

// NewPositionAllocator 创建仓位分配器
func NewPositionAllocator() *PositionAllocator {
	return &PositionAllocator{
		allocationModel: "mean_variance",
		riskBudget:      1.0,
		optimizer: &AllocationOptimizer{
			algorithm:         "quadratic_programming",
			objective:         "max_sharpe",
			constraints:       make([]OptimizationConstraint, 0),
			maxIterations:     1000,
			convergenceThresh: 1e-6,
		},
	}
}

// NewRebalancer 创建再平衡器
func NewRebalancer() *Rebalancer {
	return &Rebalancer{
		strategy:         "threshold_based",
		threshold:        0.05,
		frequency:        24 * time.Hour,
		costModel:        "linear",
		rebalanceHistory: make([]RebalanceEvent, 0),
	}
}

// NewLayeredRiskManager 创建分层风险管理器
func NewLayeredRiskManager() *LayeredRiskManager {
	return &LayeredRiskManager{
		riskLimits:       make(map[int]RiskLimit),
		correlationModel: "pearson",
		stressScenarios:  make([]StressScenario, 0),
		monitoringRules:  make([]RiskRule, 0),
	}
}

// Start 启动分层仓位管理器
func (lpm *LayeredPositionManager) Start() error {
	lpm.mu.Lock()
	defer lpm.mu.Unlock()

	if lpm.isRunning {
		return fmt.Errorf("layered position manager is already running")
	}

	if !lpm.enabled {
		return fmt.Errorf("layered position manager is disabled")
	}

	log.Println("Starting Layered Position Manager...")

	// 启动分配监控
	lpm.wg.Add(1)
	go lpm.runAllocationMonitoring()

	// 启动再平衡监控
	lpm.wg.Add(1)
	go lpm.runRebalanceMonitoring()

	// 启动风险监控
	lpm.wg.Add(1)
	go lpm.runRiskMonitoring()

	// 启动性能分析
	lpm.wg.Add(1)
	go lpm.runPerformanceAnalysis()

	// 启动指标收集
	lpm.wg.Add(1)
	go lpm.runMetricsCollection()

	lpm.isRunning = true
	log.Println("Layered Position Manager started successfully")
	return nil
}

// Stop 停止分层仓位管理器
func (lpm *LayeredPositionManager) Stop() error {
	lpm.mu.Lock()
	defer lpm.mu.Unlock()

	if !lpm.isRunning {
		return fmt.Errorf("layered position manager is not running")
	}

	log.Println("Stopping Layered Position Manager...")

	lpm.cancel()
	lpm.wg.Wait()

	lpm.isRunning = false
	log.Println("Layered Position Manager stopped successfully")
	return nil
}

// initializeLayers 初始化层级
func (lpm *LayeredPositionManager) initializeLayers() error {
	lpm.layers = make([]PositionLayer, lpm.layerCount)

	// 保守层 (40%)
	lpm.layers[0] = PositionLayer{
		ID:            0,
		Name:          "Conservative Layer",
		AllocationPct: lpm.layerSizes[0],
		RiskLevel:     "LOW",
		Strategy:      "conservative",
		MaxLeverage:   1.5,
		MaxDrawdown:   0.05,
		AllowedAssets: []string{"BTC", "ETH", "USDT"},
		Constraints: LayerConstraints{
			MaxPositionSize:         0.3,
			MinPositionSize:         0.01,
			MaxAssetConcentration:   0.4,
			MaxSectorConcentration:  0.6,
			RequiredDiversification: 3,
			AllowedInstruments:      []string{"SPOT", "FUTURES"},
		},
	}

	// 稳健层 (35%)
	lpm.layers[1] = PositionLayer{
		ID:            1,
		Name:          "Moderate Layer",
		AllocationPct: lpm.layerSizes[1],
		RiskLevel:     "MEDIUM",
		Strategy:      "moderate",
		MaxLeverage:   3.0,
		MaxDrawdown:   0.08,
		AllowedAssets: []string{"BTC", "ETH", "BNB", "ADA", "DOT", "LINK"},
		Constraints: LayerConstraints{
			MaxPositionSize:         0.25,
			MinPositionSize:         0.02,
			MaxAssetConcentration:   0.35,
			MaxSectorConcentration:  0.5,
			RequiredDiversification: 5,
			AllowedInstruments:      []string{"SPOT", "FUTURES", "OPTIONS"},
		},
	}

	// 进取层 (25%)
	lpm.layers[2] = PositionLayer{
		ID:            2,
		Name:          "Aggressive Layer",
		AllocationPct: lpm.layerSizes[2],
		RiskLevel:     "HIGH",
		Strategy:      "aggressive",
		MaxLeverage:   5.0,
		MaxDrawdown:   0.15,
		AllowedAssets: []string{"*"}, // 允许所有资产
		Constraints: LayerConstraints{
			MaxPositionSize:         0.2,
			MinPositionSize:         0.01,
			MaxAssetConcentration:   0.3,
			MaxSectorConcentration:  0.4,
			RequiredDiversification: 8,
			AllowedInstruments:      []string{"SPOT", "FUTURES", "OPTIONS", "PERPETUAL"},
		},
	}

	// 初始化层级分配
	for i, layer := range lpm.layers {
		lpm.layerAllocations[i] = &LayerAllocation{
			LayerID:         layer.ID,
			AllocatedFunds:  0.0,
			UsedFunds:       0.0,
			AvailableFunds:  0.0,
			Positions:       make(map[string]*Position),
			LastRebalance:   time.Now(),
			RebalanceNeeded: false,
		}

		// 初始化风险限制
		lpm.riskManager.riskLimits[i] = RiskLimit{
			LayerID:            i,
			MaxVaR:             layer.AllocationPct * 0.1, // 10%的层级资金作为VaR限制
			MaxDrawdown:        layer.MaxDrawdown,
			MaxLeverage:        layer.MaxLeverage,
			MaxConcentration:   layer.Constraints.MaxAssetConcentration,
			MaxCorrelation:     0.8,
			MinDiversification: layer.Constraints.RequiredDiversification,
		}
	}

	return nil
}

// runAllocationMonitoring 运行分配监控
func (lpm *LayeredPositionManager) runAllocationMonitoring() {
	defer lpm.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	log.Println("Allocation monitoring started")

	for {
		select {
		case <-lpm.ctx.Done():
			log.Println("Allocation monitoring stopped")
			return
		case <-ticker.C:
			lpm.monitorAllocations()
		}
	}
}

// runRebalanceMonitoring 运行再平衡监控
func (lpm *LayeredPositionManager) runRebalanceMonitoring() {
	defer lpm.wg.Done()

	ticker := time.NewTicker(lpm.rebalanceInterval)
	defer ticker.Stop()

	log.Println("Rebalance monitoring started")

	for {
		select {
		case <-lpm.ctx.Done():
			log.Println("Rebalance monitoring stopped")
			return
		case <-ticker.C:
			lpm.checkRebalanceNeeds()
		}
	}
}

// runRiskMonitoring 运行风险监控
func (lpm *LayeredPositionManager) runRiskMonitoring() {
	defer lpm.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.Println("Risk monitoring started")

	for {
		select {
		case <-lpm.ctx.Done():
			log.Println("Risk monitoring stopped")
			return
		case <-ticker.C:
			lpm.monitorRisks()
		}
	}
}

// runPerformanceAnalysis 运行性能分析
func (lpm *LayeredPositionManager) runPerformanceAnalysis() {
	defer lpm.wg.Done()

	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	log.Println("Performance analysis started")

	for {
		select {
		case <-lpm.ctx.Done():
			log.Println("Performance analysis stopped")
			return
		case <-ticker.C:
			lpm.analyzePerformance()
		}
	}
}

// runMetricsCollection 运行指标收集
func (lpm *LayeredPositionManager) runMetricsCollection() {
	defer lpm.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Println("Metrics collection started")

	for {
		select {
		case <-lpm.ctx.Done():
			log.Println("Metrics collection stopped")
			return
		case <-ticker.C:
			lpm.updateMetrics()
		}
	}
}

// AllocateFunds 分配资金到各层级
func (lpm *LayeredPositionManager) AllocateFunds(totalFunds float64) error {
	lpm.mu.Lock()
	defer lpm.mu.Unlock()

	log.Printf("Allocating funds: %.2f across %d layers", totalFunds, lpm.layerCount)

	lpm.totalFunds = totalFunds

	// 按比例分配到各层级
	for i, layer := range lpm.layers {
		allocation := lpm.layerAllocations[i]
		allocation.AllocatedFunds = totalFunds * layer.AllocationPct
		allocation.AvailableFunds = allocation.AllocatedFunds - allocation.UsedFunds

		log.Printf("Layer %d (%s): Allocated %.2f (%.1f%%)",
			i, layer.Name, allocation.AllocatedFunds, layer.AllocationPct*100)
	}

	// 记录分配快照
	lpm.recordAllocationSnapshot()

	return nil
}

// monitorAllocations 监控分配情况
func (lpm *LayeredPositionManager) monitorAllocations() {
	log.Println("Monitoring layer allocations...")

	lpm.mu.RLock()
	allocations := make(map[int]*LayerAllocation)
	for k, v := range lpm.layerAllocations {
		allocations[k] = v
	}
	lpm.mu.RUnlock()

	for layerID, allocation := range allocations {
		// 检查资金使用率
		utilizationRate := allocation.UsedFunds / allocation.AllocatedFunds

		// 检查是否需要再平衡
		if math.Abs(utilizationRate-lpm.layers[layerID].AllocationPct) > lpm.rebalanceThreshold {
			allocation.RebalanceNeeded = true
			log.Printf("Layer %d needs rebalancing (utilization: %.2f%%)",
				layerID, utilizationRate*100)
		}

		// 检查风险限制
		lpm.checkLayerRiskLimits(layerID, allocation)
	}
}

// checkRebalanceNeeds 检查再平衡需求
func (lpm *LayeredPositionManager) checkRebalanceNeeds() {
	log.Println("Checking rebalance needs...")

	needsRebalance := false
	layersToRebalance := make([]int, 0)

	lpm.mu.RLock()
	for layerID, allocation := range lpm.layerAllocations {
		if allocation.RebalanceNeeded {
			needsRebalance = true
			layersToRebalance = append(layersToRebalance, layerID)
		}
	}
	lpm.mu.RUnlock()

	if needsRebalance {
		err := lpm.executeRebalance(layersToRebalance)
		if err != nil {
			log.Printf("Rebalance failed: %v", err)
		} else {
			log.Printf("Rebalance completed for layers: %v", layersToRebalance)
		}
	}
}

// executeRebalance 执行再平衡
func (lpm *LayeredPositionManager) executeRebalance(layerIDs []int) error {
	rebalanceEvent := RebalanceEvent{
		ID:             lpm.generateRebalanceID(),
		Timestamp:      time.Now(),
		Type:           "SCHEDULED",
		Trigger:        "THRESHOLD_EXCEEDED",
		LayersAffected: layerIDs,
		Changes:        make([]PositionChange, 0),
		Success:        false,
	}

	log.Printf("Executing rebalance for layers: %v", layerIDs)

	// 计算目标分配
	targetAllocations, err := lpm.calculateTargetAllocations(layerIDs)
	if err != nil {
		return fmt.Errorf("failed to calculate target allocations: %w", err)
	}

	// 生成交易指令
	changes, err := lpm.generateRebalanceChanges(targetAllocations)
	if err != nil {
		return fmt.Errorf("failed to generate rebalance changes: %w", err)
	}

	rebalanceEvent.Changes = changes

	// 计算预期收益和成本
	rebalanceEvent.TotalCost = lpm.calculateRebalanceCost(changes)
	rebalanceEvent.ExpectedBenefit = lpm.calculateExpectedBenefit(changes)

	// 执行交易（模拟）
	err = lpm.executeRebalanceTrades(changes)
	if err != nil {
		rebalanceEvent.Success = false
		return fmt.Errorf("failed to execute rebalance trades: %w", err)
	}

	// 更新分配状态
	lpm.updateAllocationStatus(layerIDs)

	rebalanceEvent.Success = true
	rebalanceEvent.ActualBenefit = lpm.calculateActualBenefit(changes)

	// 记录再平衡事件
	lpm.rebalancer.mu.Lock()
	lpm.rebalancer.rebalanceHistory = append(lpm.rebalancer.rebalanceHistory, rebalanceEvent)
	lpm.rebalancer.lastRebalance = time.Now()
	lpm.rebalancer.mu.Unlock()

	return nil
}

// monitorRisks 监控风险
func (lpm *LayeredPositionManager) monitorRisks() {
	lpm.mu.RLock()
	allocations := make(map[int]*LayerAllocation)
	for k, v := range lpm.layerAllocations {
		allocations[k] = v
	}
	lpm.mu.RUnlock()

	for layerID, allocation := range allocations {
		// 计算层级风险指标
		riskMetrics := lpm.calculateLayerRiskMetrics(layerID, allocation)
		allocation.RiskMetrics = riskMetrics

		// 检查风险限制
		lpm.checkLayerRiskLimits(layerID, allocation)
	}
}

// checkLayerRiskLimits 检查层级风险限制
func (lpm *LayeredPositionManager) checkLayerRiskLimits(layerID int, allocation *LayerAllocation) {
	riskLimit := lpm.riskManager.riskLimits[layerID]

	// 检查VaR限制
	if allocation.RiskMetrics.CurrentVaR > riskLimit.MaxVaR {
		log.Printf("Layer %d VaR exceeded: %.4f > %.4f",
			layerID, allocation.RiskMetrics.CurrentVaR, riskLimit.MaxVaR)
		lpm.triggerRiskAction(layerID, "VAR_EXCEEDED", allocation.RiskMetrics.CurrentVaR)
	}

	// 检查杠杆限制
	if allocation.RiskMetrics.LeverageRatio > riskLimit.MaxLeverage {
		log.Printf("Layer %d leverage exceeded: %.2f > %.2f",
			layerID, allocation.RiskMetrics.LeverageRatio, riskLimit.MaxLeverage)
		lpm.triggerRiskAction(layerID, "LEVERAGE_EXCEEDED", allocation.RiskMetrics.LeverageRatio)
	}

	// 检查集中度限制
	if allocation.RiskMetrics.ConcentrationRisk > riskLimit.MaxConcentration {
		log.Printf("Layer %d concentration exceeded: %.4f > %.4f",
			layerID, allocation.RiskMetrics.ConcentrationRisk, riskLimit.MaxConcentration)
		lpm.triggerRiskAction(layerID, "CONCENTRATION_EXCEEDED", allocation.RiskMetrics.ConcentrationRisk)
	}
}

// analyzePerformance 分析性能
func (lpm *LayeredPositionManager) analyzePerformance() {
	log.Println("Analyzing layer performance...")

	lpm.mu.RLock()
	allocations := make(map[int]*LayerAllocation)
	for k, v := range lpm.layerAllocations {
		allocations[k] = v
	}
	lpm.mu.RUnlock()

	for layerID, allocation := range allocations {
		performance := lpm.calculateLayerPerformance(layerID, allocation)

		allocation.Performance = performance
		lpm.managementMetrics.LayerPerformances[layerID] = performance

		log.Printf("Layer %d performance - Return: %.4f, Sharpe: %.4f, Drawdown: %.4f",
			layerID, performance.AnnualizedReturn, performance.SharpeRatio, performance.MaxDrawdown)
	}

	// 找出最佳和最差表现层级
	lpm.identifyBestWorstLayers()
}

// Helper functions implementation...

func (lpm *LayeredPositionManager) calculateTargetAllocations(layerIDs []int) (map[int]map[string]float64, error) {
	// 实现目标分配计算
	allocations := make(map[int]map[string]float64)

	for _, layerID := range layerIDs {
		allocations[layerID] = make(map[string]float64)

		// 获取层级信息
		if layerID >= len(lpm.layers) {
			continue
		}

		layer := lpm.layers[layerID]

		// 根据层级风险水平和策略计算目标分配
		switch layer.RiskLevel {
		case "LOW": // 保守层
			allocations[layerID] = lpm.calculateConservativeAllocation(layer)
		case "MEDIUM": // 稳健层
			allocations[layerID] = lpm.calculateModerateAllocation(layer)
		case "HIGH": // 进取层
			allocations[layerID] = lpm.calculateAggressiveAllocation(layer)
		default:
			// 默认均衡分配
			allocations[layerID] = lpm.calculateBalancedAllocation(layer)
		}

		// 应用层级约束
		allocations[layerID] = lpm.applyLayerConstraints(layerID, allocations[layerID])

		// 验证分配总和为1
		total := 0.0
		for _, weight := range allocations[layerID] {
			total += weight
		}

		if math.Abs(total-1.0) > 0.01 {
			// 归一化权重
			for asset := range allocations[layerID] {
				allocations[layerID][asset] /= total
			}
		}

		log.Printf("Target allocation calculated for layer %d (%s): %v",
			layerID, layer.Name, allocations[layerID])
	}

	return allocations, nil
}

func (lpm *LayeredPositionManager) generateRebalanceChanges(allocations map[int]map[string]float64) ([]PositionChange, error) {
	changes := make([]PositionChange, 0)

	// 实现具体的变化计算逻辑
	for layerID, targetAllocation := range allocations {
		// 获取当前层级分配
		layerAllocation, exists := lpm.layerAllocations[layerID]
		if !exists {
			log.Printf("Layer %d allocation not found, skipping", layerID)
			continue
		}

		// 计算当前权重
		currentWeights := lpm.calculateCurrentWeights(layerID, layerAllocation)

		// 比较目标权重和当前权重，生成变化
		for symbol, targetWeight := range targetAllocation {
			currentWeight := currentWeights[symbol]
			weightDiff := targetWeight - currentWeight

			// 只有当权重变化超过阈值时才进行调整
			if math.Abs(weightDiff) < lpm.rebalanceThreshold {
				continue
			}

			// 获取当前市场价格
			currentPrice, err := lpm.getCurrentPrice(symbol)
			if err != nil {
				log.Printf("Failed to get price for %s: %v", symbol, err)
				continue
			}

			// 计算需要调整的数量
			layerFunds := layerAllocation.AllocatedFunds
			targetValue := layerFunds * targetWeight
			currentValue := layerFunds * currentWeight
			valueDiff := targetValue - currentValue
			quantity := math.Abs(valueDiff / currentPrice)

			// 确定变化类型
			changeType := "ADJUST"
			if currentWeight == 0 {
				changeType = "ADD"
			} else if targetWeight == 0 {
				changeType = "REMOVE"
			}

			// 计算交易成本
			cost := quantity * currentPrice * 0.0004 // 假设0.04%手续费

			change := PositionChange{
				Symbol:     symbol,
				LayerID:    layerID,
				OldWeight:  currentWeight,
				NewWeight:  targetWeight,
				ChangeType: changeType,
				Quantity:   quantity,
				Price:      currentPrice,
				Cost:       cost,
				Reason:     fmt.Sprintf("Rebalance: weight %.4f -> %.4f", currentWeight, targetWeight),
			}

			changes = append(changes, change)

			log.Printf("Generated rebalance change for layer %d: %s %s %.4f (%.4f%% -> %.4f%%)",
				layerID, changeType, symbol, quantity, currentWeight*100, targetWeight*100)
		}

		// 检查当前持有但目标分配中没有的资产（需要清仓）
		for symbol, currentWeight := range currentWeights {
			if _, exists := targetAllocation[symbol]; !exists && currentWeight > 0 {
				// 获取当前市场价格
				currentPrice, err := lpm.getCurrentPrice(symbol)
				if err != nil {
					log.Printf("Failed to get price for %s: %v", symbol, err)
					continue
				}

				// 计算清仓数量
				layerFunds := layerAllocation.AllocatedFunds
				currentValue := layerFunds * currentWeight
				quantity := currentValue / currentPrice
				cost := quantity * currentPrice * 0.0004

				change := PositionChange{
					Symbol:     symbol,
					LayerID:    layerID,
					OldWeight:  currentWeight,
					NewWeight:  0.0,
					ChangeType: "REMOVE",
					Quantity:   quantity,
					Price:      currentPrice,
					Cost:       cost,
					Reason:     "Remove from target allocation",
				}

				changes = append(changes, change)

				log.Printf("Generated removal change for layer %d: REMOVE %s %.4f (%.4f%% -> 0%%)",
					layerID, symbol, quantity, currentWeight*100)
			}
		}
	}

	return changes, nil
}

func (lpm *LayeredPositionManager) calculateRebalanceCost(changes []PositionChange) float64 {
	totalCost := 0.0
	for _, change := range changes {
		totalCost += change.Cost
	}
	return totalCost
}

func (lpm *LayeredPositionManager) calculateExpectedBenefit(changes []PositionChange) float64 {
	// 实现预期收益计算
	totalExpectedBenefit := 0.0

	for _, change := range changes {
		// 获取层级信息
		if change.LayerID >= len(lpm.layers) {
			continue
		}

		layer := lpm.layers[change.LayerID]

		// 根据变化类型和层级策略计算预期收益
		var expectedReturn float64

		switch layer.RiskLevel {
		case "LOW":
			expectedReturn = 0.08 // 保守层预期8%年化收益
		case "MEDIUM":
			expectedReturn = 0.12 // 稳健层预期12%年化收益
		case "HIGH":
			expectedReturn = 0.18 // 进取层预期18%年化收益
		default:
			expectedReturn = 0.10 // 默认10%年化收益
		}

		// 根据资产类型调整预期收益
		assetMultiplier := lpm.getAssetReturnMultiplier(change.Symbol)
		expectedReturn *= assetMultiplier

		// 计算仓位价值
		positionValue := change.Quantity * change.Price

		// 根据变化类型计算收益
		switch change.ChangeType {
		case "ADD":
			// 新增仓位的预期收益
			totalExpectedBenefit += positionValue * expectedReturn
		case "ADJUST":
			// 调整仓位的预期收益改善
			weightChange := change.NewWeight - change.OldWeight
			if weightChange > 0 {
				// 增加仓位
				totalExpectedBenefit += positionValue * math.Abs(weightChange) * expectedReturn
			} else {
				// 减少仓位，可能避免损失
				totalExpectedBenefit += positionValue * math.Abs(weightChange) * expectedReturn * 0.5
			}
		case "REMOVE":
			// 移除仓位，避免潜在损失
			if expectedReturn < 0 {
				totalExpectedBenefit += positionValue * math.Abs(expectedReturn)
			}
		}

		// 考虑再平衡的风险分散效益
		diversificationBenefit := lpm.calculateDiversificationBenefit(change)
		totalExpectedBenefit += diversificationBenefit
	}

	log.Printf("Calculated expected benefit from %d changes: %.2f", len(changes), totalExpectedBenefit)
	return totalExpectedBenefit
}

func (lpm *LayeredPositionManager) calculateActualBenefit(changes []PositionChange) float64 {
	// 实现实际收益计算
	totalActualBenefit := 0.0

	for _, change := range changes {
		// 获取执行后的实际价格和数量
		actualPrice := change.Price
		actualQuantity := change.Quantity

		// 计算实际执行成本
		actualCost := change.Cost

		// 获取层级信息
		if change.LayerID >= len(lpm.layers) {
			continue
		}

		layer := lpm.layers[change.LayerID]

		// 计算实际收益（考虑滑点和执行成本）
		positionValue := actualQuantity * actualPrice

		// 根据层级策略计算实际收益率
		var actualReturn float64
		switch layer.RiskLevel {
		case "LOW":
			actualReturn = 0.075 // 实际收益通常略低于预期
		case "MEDIUM":
			actualReturn = 0.11
		case "HIGH":
			actualReturn = 0.16
		default:
			actualReturn = 0.09
		}

		// 应用资产收益倍数
		assetMultiplier := lpm.getAssetReturnMultiplier(change.Symbol)
		actualReturn *= assetMultiplier

		// 考虑市场条件对实际收益的影响
		marketConditionAdjustment := lpm.getMarketConditionAdjustment()
		actualReturn *= marketConditionAdjustment

		// 计算净收益（扣除成本）
		grossBenefit := positionValue * actualReturn
		netBenefit := grossBenefit - actualCost

		// 根据变化类型调整收益计算
		switch change.ChangeType {
		case "ADD":
			totalActualBenefit += netBenefit
		case "ADJUST":
			// 调整仓位的收益改善
			weightChange := change.NewWeight - change.OldWeight
			adjustmentFactor := math.Abs(weightChange)
			totalActualBenefit += netBenefit * adjustmentFactor
		case "REMOVE":
			// 移除仓位避免的损失
			if actualReturn < 0 {
				totalActualBenefit += math.Abs(netBenefit)
			} else {
				totalActualBenefit -= actualCost // 只扣除交易成本
			}
		}

		log.Printf("Actual benefit calculated for %s %s: %.2f (gross: %.2f, cost: %.2f)",
			change.ChangeType, change.Symbol, netBenefit, grossBenefit, actualCost)
	}

	log.Printf("Total actual benefit from %d changes: %.2f", len(changes), totalActualBenefit)
	return totalActualBenefit
}

func (lpm *LayeredPositionManager) executeRebalanceTrades(changes []PositionChange) error {
	// 实现实际的交易执行
	if len(changes) == 0 {
		log.Printf("No rebalance trades to execute")
		return nil
	}

	log.Printf("Executing %d rebalance trades", len(changes))

	// 按优先级排序交易（先执行移除，再执行调整，最后执行新增）
	sortedChanges := lpm.sortChangesByPriority(changes)

	successCount := 0
	failCount := 0

	for i, change := range sortedChanges {
		log.Printf("Executing trade %d/%d: %s %s %.4f @ %.2f",
			i+1, len(sortedChanges), change.ChangeType, change.Symbol, change.Quantity, change.Price)

		// 执行具体的交易
		err := lpm.executeSingleTrade(change)
		if err != nil {
			log.Printf("Failed to execute trade for %s: %v", change.Symbol, err)
			failCount++

			// 记录失败的交易
			lpm.recordFailedTrade(change, err)
			continue
		}

		// 更新仓位信息
		err = lpm.updatePositionAfterTrade(change)
		if err != nil {
			log.Printf("Failed to update position after trade for %s: %v", change.Symbol, err)
		}

		successCount++

		// 添加执行间隔，避免过于频繁的交易
		if i < len(sortedChanges)-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	log.Printf("Rebalance execution completed: %d successful, %d failed", successCount, failCount)

	// 更新再平衡历史
	lpm.recordRebalanceExecution(changes, successCount, failCount)

	if failCount > 0 {
		return fmt.Errorf("some trades failed: %d out of %d", failCount, len(changes))
	}

	return nil
}

func (lpm *LayeredPositionManager) updateAllocationStatus(layerIDs []int) {
	for _, layerID := range layerIDs {
		if allocation, exists := lpm.layerAllocations[layerID]; exists {
			allocation.RebalanceNeeded = false
			allocation.LastRebalance = time.Now()
		}
	}
}

func (lpm *LayeredPositionManager) calculateLayerRiskMetrics(layerID int, allocation *LayerAllocation) LayerRiskMetrics {
	// 实现具体的风险指标计算

	// 收集层级中所有仓位的收益率数据
	var allReturns []float64
	var positionValues []float64
	totalValue := 0.0

	// 计算各仓位的价值和收益率
	for symbol, position := range allocation.Positions {
		if position.Quantity == 0 {
			continue
		}

		// 获取历史收益率数据
		returns, err := lpm.getHistoricalReturns(symbol, 30) // 30天历史数据
		if err != nil {
			log.Printf("Failed to get returns for %s: %v", symbol, err)
			continue
		}

		if len(returns) == 0 {
			continue
		}

		// 计算仓位价值
		currentPrice, err := lpm.getMarketPrice(symbol)
		if err != nil {
			log.Printf("Failed to get current price for %s: %v", symbol, err)
			continue
		}

		positionValue := math.Abs(position.Quantity) * currentPrice
		positionValues = append(positionValues, positionValue)
		totalValue += positionValue

		// 加权收益率
		for _, ret := range returns {
			allReturns = append(allReturns, ret)
		}
	}

	// 计算VaR (95%置信度)
	var currentVaR float64
	if len(allReturns) > 0 {
		currentVaR = lpm.calculateVaR(allReturns, 0.95) * allocation.AllocatedFunds
	} else {
		currentVaR = allocation.AllocatedFunds * 0.05 // 默认5%
	}

	// 计算Expected Shortfall (CVaR)
	var expectedShortfall float64
	if len(allReturns) > 0 {
		expectedShortfall = lpm.calculateExpectedShortfall(allReturns, 0.95) * allocation.AllocatedFunds
	} else {
		expectedShortfall = allocation.AllocatedFunds * 0.07 // 默认7%
	}

	// 计算市场Beta
	betaToMarket := lpm.calculateLayerBeta(layerID, allocation)

	// 计算杠杆比率
	leverageRatio := lpm.calculateLayerLeverage(allocation)

	// 计算集中度风险 (使用Herfindahl-Hirschman指数)
	concentrationRisk := lpm.calculateConcentrationRisk(positionValues, totalValue)

	// 计算流动性风险
	liquidityRisk := lpm.calculateLiquidityRisk(allocation)

	// 计算相关性矩阵
	correlationMatrix := lpm.calculateCorrelationMatrix(allocation)

	return LayerRiskMetrics{
		CurrentVaR:        currentVaR,
		ExpectedShortfall: expectedShortfall,
		BetaToMarket:      betaToMarket,
		LeverageRatio:     leverageRatio,
		ConcentrationRisk: concentrationRisk,
		LiquidityRisk:     liquidityRisk,
		CorrelationMatrix: correlationMatrix,
	}
}

// getHistoricalReturns 获取历史收益率数据
func (lpm *LayeredPositionManager) getHistoricalReturns(symbol string, days int) ([]float64, error) {
	// 模拟获取历史收益率数据
	// 在实际实现中，这里会从数据库或API获取真实的历史价格数据
	returns := make([]float64, days)
	for i := 0; i < days; i++ {
		// 生成模拟的日收益率，基于正态分布
		returns[i] = (rand.Float64() - 0.5) * 0.04 // -2% 到 +2% 的随机收益率
	}
	return returns, nil
}

// getMarketPrice 获取市场价格
func (lpm *LayeredPositionManager) getMarketPrice(symbol string) (float64, error) {
	// 模拟获取市场价格
	// 在实际实现中，这里会从交易所API获取实时价格
	basePrice := 100.0 + rand.Float64()*50.0 // 100-150的随机价格
	return basePrice, nil
}

// calculateVaR 计算VaR
func (lpm *LayeredPositionManager) calculateVaR(returns []float64, confidence float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	// 使用历史模拟法计算VaR
	sorted := make([]float64, len(returns))
	copy(sorted, returns)
	sort.Float64s(sorted)

	index := int(math.Ceil(float64(len(sorted)) * (1 - confidence)))
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	if index < 0 {
		index = 0
	}

	return -sorted[index] // VaR是负收益的正值
}

// calculateExpectedShortfall 计算期望损失
func (lpm *LayeredPositionManager) calculateExpectedShortfall(returns []float64, confidence float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	// 排序收益率
	sorted := make([]float64, len(returns))
	copy(sorted, returns)
	sort.Float64s(sorted)

	// 找到VaR对应的索引
	index := int(math.Ceil(float64(len(sorted)) * (1 - confidence)))
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	if index < 0 {
		index = 0
	}

	// 计算尾部损失的平均值
	var tailSum float64
	tailCount := 0
	for i := 0; i <= index; i++ {
		tailSum += sorted[i]
		tailCount++
	}

	if tailCount == 0 {
		return 0
	}

	return -tailSum / float64(tailCount) // ES是负收益的正值
}

// calculateLayerBeta 计算层级Beta
func (lpm *LayeredPositionManager) calculateLayerBeta(layerID int, allocation *LayerAllocation) float64 {
	// 模拟计算市场Beta
	// 在实际实现中，这里会计算层级收益率与市场指数的协方差/市场方差

	if len(allocation.Positions) == 0 {
		return 1.0 // 默认Beta
	}

	// 简化的Beta计算：基于仓位的加权平均
	totalBeta := 0.0
	totalWeight := 0.0

	for symbol, position := range allocation.Positions {
		if position.Quantity == 0 {
			continue
		}

		// 获取单个资产的Beta（模拟）
		assetBeta := 0.8 + rand.Float64()*0.8 // 0.8-1.6的随机Beta

		// 计算权重
		currentPrice, err := lpm.getMarketPrice(symbol)
		if err != nil {
			continue
		}
		weight := math.Abs(position.Quantity) * currentPrice

		totalBeta += assetBeta * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 1.0
	}

	return totalBeta / totalWeight
}

// calculateLayerLeverage 计算层级杠杆比率
func (lpm *LayeredPositionManager) calculateLayerLeverage(allocation *LayerAllocation) float64 {
	if allocation.AllocatedFunds == 0 {
		return 1.0
	}

	// 计算总敞口
	totalExposure := 0.0
	for _, position := range allocation.Positions {
		if position.Quantity == 0 {
			continue
		}

		currentPrice, err := lpm.getMarketPrice(position.Symbol)
		if err != nil {
			continue
		}

		exposure := math.Abs(position.Quantity) * currentPrice
		totalExposure += exposure
	}

	// 杠杆比率 = 总敞口 / 分配资金
	leverageRatio := totalExposure / allocation.AllocatedFunds

	// 限制在合理范围内
	return math.Min(leverageRatio, 10.0) // 最大10倍杠杆
}

// calculateConcentrationRisk 计算集中度风险
func (lpm *LayeredPositionManager) calculateConcentrationRisk(positionValues []float64, totalValue float64) float64 {
	if len(positionValues) == 0 || totalValue == 0 {
		return 0.0
	}

	// 使用Herfindahl-Hirschman指数计算集中度风险
	hhi := 0.0
	for _, value := range positionValues {
		share := value / totalValue
		hhi += share * share
	}

	// 标准化到0-1范围，1表示最大集中度（所有资金在一个仓位）
	return hhi
}

// calculateLiquidityRisk 计算流动性风险
func (lpm *LayeredPositionManager) calculateLiquidityRisk(allocation *LayerAllocation) float64 {
	if len(allocation.Positions) == 0 {
		return 0.0
	}

	// 简化的流动性风险计算
	// 在实际实现中，这里会考虑：
	// 1. 订单簿深度
	// 2. 平均日交易量
	// 3. 买卖价差
	// 4. 市场冲击成本

	totalRisk := 0.0
	totalWeight := 0.0

	for symbol, position := range allocation.Positions {
		if position.Quantity == 0 {
			continue
		}

		// 模拟流动性风险评分（0-1，1表示最高风险）
		// 基于仓位大小的简化计算
		currentPrice, err := lpm.getMarketPrice(symbol)
		if err != nil {
			continue
		}

		positionValue := math.Abs(position.Quantity) * currentPrice

		// 假设仓位越大，流动性风险越高
		liquidityScore := math.Min(positionValue/allocation.AllocatedFunds, 1.0)

		totalRisk += liquidityScore * positionValue
		totalWeight += positionValue
	}

	if totalWeight == 0 {
		return 0.0
	}

	return totalRisk / totalWeight
}

// calculateCorrelationMatrix 计算相关性矩阵
func (lpm *LayeredPositionManager) calculateCorrelationMatrix(allocation *LayerAllocation) map[string]map[string]float64 {
	correlationMatrix := make(map[string]map[string]float64)

	symbols := make([]string, 0, len(allocation.Positions))
	for symbol := range allocation.Positions {
		symbols = append(symbols, symbol)
	}

	// 计算每对资产之间的相关性
	for i, symbol1 := range symbols {
		correlationMatrix[symbol1] = make(map[string]float64)

		for j, symbol2 := range symbols {
			if i == j {
				correlationMatrix[symbol1][symbol2] = 1.0 // 自相关为1
			} else {
				// 获取两个资产的历史收益率
				returns1, err1 := lpm.getHistoricalReturns(symbol1, 30)
				returns2, err2 := lpm.getHistoricalReturns(symbol2, 30)

				if err1 != nil || err2 != nil || len(returns1) != len(returns2) {
					// 使用默认相关性
					correlationMatrix[symbol1][symbol2] = 0.3 + rand.Float64()*0.4 // 0.3-0.7的随机相关性
				} else {
					// 计算皮尔逊相关系数
					correlation := lpm.calculateCorrelation(returns1, returns2)
					correlationMatrix[symbol1][symbol2] = correlation
				}
			}
		}
	}

	return correlationMatrix
}

// calculateCorrelation 计算皮尔逊相关系数
func (lpm *LayeredPositionManager) calculateCorrelation(returns1, returns2 []float64) float64 {
	if len(returns1) != len(returns2) || len(returns1) < 2 {
		return 0.0
	}

	// 计算均值
	mean1 := lpm.calculateMean(returns1)
	mean2 := lpm.calculateMean(returns2)

	// 计算协方差和方差
	covariance := 0.0
	variance1 := 0.0
	variance2 := 0.0

	for i := 0; i < len(returns1); i++ {
		dev1 := returns1[i] - mean1
		dev2 := returns2[i] - mean2

		covariance += dev1 * dev2
		variance1 += dev1 * dev1
		variance2 += dev2 * dev2
	}

	if variance1 == 0 || variance2 == 0 {
		return 0.0
	}

	correlation := covariance / math.Sqrt(variance1*variance2)

	// 限制在[-1, 1]范围内
	return math.Max(-1.0, math.Min(1.0, correlation))
}

// calculateMean 计算平均值
func (lpm *LayeredPositionManager) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, value := range values {
		sum += value
	}

	return sum / float64(len(values))
}

func (lpm *LayeredPositionManager) calculateLayerPerformance(layerID int, allocation *LayerAllocation) LayerPerformance {
	// 实现具体的性能计算

	// 收集层级的历史收益率数据
	var layerReturns []float64
	var positionPnLs []float64

	// 计算层级的组合收益率
	for symbol, position := range allocation.Positions {
		if position.Quantity == 0 {
			continue
		}

		// 获取历史收益率
		returns, err := lpm.getHistoricalReturns(symbol, 30)
		if err != nil {
			log.Printf("Failed to get returns for %s: %v", symbol, err)
			continue
		}

		// 计算仓位权重
		currentPrice, err := lpm.getMarketPrice(symbol)
		if err != nil {
			continue
		}

		positionValue := math.Abs(position.Quantity) * currentPrice
		weight := positionValue / allocation.AllocatedFunds

		// 加权收益率
		for _, ret := range returns {
			weightedReturn := ret * weight
			layerReturns = append(layerReturns, weightedReturn)
		}

		// 计算仓位PnL
		entryPrice := position.Price
		if entryPrice > 0 {
			pnl := (currentPrice - entryPrice) / entryPrice
			positionPnLs = append(positionPnLs, pnl)
		}
	}

	// 计算总收益率
	totalReturn := lpm.calculateTotalReturn(layerReturns)

	// 计算年化收益率
	annualizedReturn := lpm.calculateAnnualizedReturn(layerReturns)

	// 计算波动率
	volatility := lpm.calculateVolatility(layerReturns)

	// 计算夏普比率
	sharpeRatio := lpm.calculateSharpeRatio(layerReturns, volatility)

	// 计算最大回撤
	maxDrawdown := lpm.calculateMaxDrawdown(layerReturns)

	// 计算Calmar比率
	calmarRatio := lpm.calculateCalmarRatio(annualizedReturn, maxDrawdown)

	// 计算胜率和盈亏比
	winRate, profitFactor := lpm.calculateTradeMetrics(positionPnLs)

	return LayerPerformance{
		TotalReturn:      totalReturn,
		AnnualizedReturn: annualizedReturn,
		Volatility:       volatility,
		SharpeRatio:      sharpeRatio,
		MaxDrawdown:      maxDrawdown,
		CalmarRatio:      calmarRatio,
		WinRate:          winRate,
		ProfitFactor:     profitFactor,
		LastUpdated:      time.Now(),
	}
}

// calculateTotalReturn 计算总收益率
func (lpm *LayeredPositionManager) calculateTotalReturn(returns []float64) float64 {
	if len(returns) == 0 {
		return 0.0
	}

	// 计算累积收益率
	totalReturn := 1.0
	for _, ret := range returns {
		totalReturn *= (1.0 + ret)
	}

	return totalReturn - 1.0
}

// calculateAnnualizedReturn 计算年化收益率
func (lpm *LayeredPositionManager) calculateAnnualizedReturn(returns []float64) float64 {
	if len(returns) == 0 {
		return 0.0
	}

	totalReturn := lpm.calculateTotalReturn(returns)

	// 假设returns是日收益率，计算年化收益率
	periods := float64(len(returns))
	periodsPerYear := 365.0 // 假设是日收益率

	if periods == 0 {
		return 0.0
	}

	// 年化收益率 = (1 + 总收益率)^(365/期数) - 1
	annualizedReturn := math.Pow(1.0+totalReturn, periodsPerYear/periods) - 1.0

	return annualizedReturn
}

// calculateVolatility 计算波动率
func (lpm *LayeredPositionManager) calculateVolatility(returns []float64) float64 {
	if len(returns) < 2 {
		return 0.0
	}

	// 计算均值
	mean := lpm.calculateMean(returns)

	// 计算方差
	variance := 0.0
	for _, ret := range returns {
		diff := ret - mean
		variance += diff * diff
	}
	variance /= float64(len(returns) - 1)

	// 标准差（波动率）
	volatility := math.Sqrt(variance)

	// 年化波动率（假设是日收益率）
	annualizedVolatility := volatility * math.Sqrt(365)

	return annualizedVolatility
}

// calculateSharpeRatio 计算夏普比率
func (lpm *LayeredPositionManager) calculateSharpeRatio(returns []float64, volatility float64) float64 {
	if len(returns) == 0 || volatility == 0 {
		return 0.0
	}

	// 计算平均收益率
	meanReturn := lpm.calculateMean(returns)

	// 年化平均收益率
	annualizedMeanReturn := meanReturn * 365 // 假设是日收益率

	// 假设无风险利率为2%
	riskFreeRate := 0.02

	// 夏普比率 = (年化收益率 - 无风险利率) / 年化波动率
	sharpeRatio := (annualizedMeanReturn - riskFreeRate) / volatility

	return sharpeRatio
}

// calculateMaxDrawdown 计算最大回撤
func (lpm *LayeredPositionManager) calculateMaxDrawdown(returns []float64) float64 {
	if len(returns) == 0 {
		return 0.0
	}

	// 计算累积净值曲线
	equity := 1.0
	peak := 1.0
	maxDrawdown := 0.0

	for _, ret := range returns {
		equity *= (1.0 + ret)

		// 更新峰值
		if equity > peak {
			peak = equity
		}

		// 计算当前回撤
		drawdown := (peak - equity) / peak

		// 更新最大回撤
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}

// calculateCalmarRatio 计算Calmar比率
func (lpm *LayeredPositionManager) calculateCalmarRatio(annualizedReturn, maxDrawdown float64) float64 {
	if maxDrawdown == 0 {
		return 0.0
	}

	// Calmar比率 = 年化收益率 / 最大回撤
	return annualizedReturn / maxDrawdown
}

// calculateTradeMetrics 计算交易指标
func (lpm *LayeredPositionManager) calculateTradeMetrics(pnls []float64) (winRate, profitFactor float64) {
	if len(pnls) == 0 {
		return 0.0, 0.0
	}

	wins := 0
	totalProfit := 0.0
	totalLoss := 0.0

	for _, pnl := range pnls {
		if pnl > 0 {
			wins++
			totalProfit += pnl
		} else if pnl < 0 {
			totalLoss += math.Abs(pnl)
		}
	}

	// 胜率
	winRate = float64(wins) / float64(len(pnls))

	// 盈亏比
	if totalLoss == 0 {
		profitFactor = 1.0
	} else {
		profitFactor = totalProfit / totalLoss
	}

	return winRate, profitFactor
}

func (lpm *LayeredPositionManager) triggerRiskAction(layerID int, riskType string, value float64) {
	log.Printf("Risk action triggered for layer %d: %s (value: %.4f)", layerID, riskType, value)

	// 实现具体的风险响应动作
	allocation, exists := lpm.layerAllocations[layerID]
	if !exists {
		log.Printf("Layer allocation not found for layer %d", layerID)
		return
	}

	switch riskType {
	case "VAR_EXCEEDED":
		// VaR超限处理
		lpm.handleVaRExceeded(layerID, allocation, value)

	case "LEVERAGE_EXCEEDED":
		// 杠杆超限处理
		lpm.handleLeverageExceeded(layerID, allocation, value)

	case "CONCENTRATION_EXCEEDED":
		// 集中度超限处理
		lpm.handleConcentrationExceeded(layerID, allocation, value)

	case "LIQUIDITY_RISK":
		// 流动性风险处理
		lpm.handleLiquidityRisk(layerID, allocation, value)

	case "CORRELATION_RISK":
		// 相关性风险处理
		lpm.handleCorrelationRisk(layerID, allocation, value)

	default:
		log.Printf("Unknown risk type: %s", riskType)
		// 通用风险处理
		lpm.handleGenericRisk(layerID, allocation, riskType, value)
	}

	// 发送风险告警
	lpm.sendRiskAlert(layerID, riskType, value)

	// 记录风险事件
	lpm.recordRiskEvent(layerID, riskType, value)
}

// handleVaRExceeded 处理VaR超限
func (lpm *LayeredPositionManager) handleVaRExceeded(layerID int, allocation *LayerAllocation, value float64) {
	log.Printf("Handling VaR exceeded for layer %d: %.4f", layerID, value)

	// 1. 减少仓位规模
	reductionRatio := 0.2 // 减少20%仓位
	lpm.reduceLayerPositions(layerID, allocation, reductionRatio)

	// 2. 降低杠杆
	lpm.reduceLeverage(layerID, allocation, 0.8) // 降低到80%

	// 3. 暂停新开仓
	lpm.pauseNewPositions(layerID, time.Hour*24) // 暂停24小时
}

// handleLeverageExceeded 处理杠杆超限
func (lpm *LayeredPositionManager) handleLeverageExceeded(layerID int, allocation *LayerAllocation, value float64) {
	log.Printf("Handling leverage exceeded for layer %d: %.4f", layerID, value)

	// 1. 强制降杠杆
	targetLeverage := lpm.layers[layerID].MaxLeverage * 0.9 // 降到限制的90%
	lpm.adjustLayerLeverage(layerID, allocation, targetLeverage)

	// 2. 减少高风险仓位
	lpm.reduceHighRiskPositions(layerID, allocation)
}

// handleConcentrationExceeded 处理集中度超限
func (lpm *LayeredPositionManager) handleConcentrationExceeded(layerID int, allocation *LayerAllocation, value float64) {
	log.Printf("Handling concentration exceeded for layer %d: %.4f", layerID, value)

	// 1. 分散化仓位
	lpm.diversifyPositions(layerID, allocation)

	// 2. 减少最大仓位
	lpm.reduceLargestPositions(layerID, allocation, 0.3) // 减少30%
}

// handleLiquidityRisk 处理流动性风险
func (lpm *LayeredPositionManager) handleLiquidityRisk(layerID int, allocation *LayerAllocation, value float64) {
	log.Printf("Handling liquidity risk for layer %d: %.4f", layerID, value)

	// 1. 减少低流动性资产仓位
	lpm.reduceLowLiquidityPositions(layerID, allocation)

	// 2. 增加现金比例
	lpm.increaseCashRatio(layerID, allocation, 0.1) // 增加10%现金
}

// handleCorrelationRisk 处理相关性风险
func (lpm *LayeredPositionManager) handleCorrelationRisk(layerID int, allocation *LayerAllocation, value float64) {
	log.Printf("Handling correlation risk for layer %d: %.4f", layerID, value)

	// 1. 减少高相关性资产
	lpm.reduceHighCorrelationPositions(layerID, allocation)

	// 2. 增加对冲仓位
	lpm.addHedgePositions(layerID, allocation)
}

// handleGenericRisk 处理通用风险
func (lpm *LayeredPositionManager) handleGenericRisk(layerID int, allocation *LayerAllocation, riskType string, value float64) {
	log.Printf("Handling generic risk for layer %d: %s = %.4f", layerID, riskType, value)

	// 通用风险处理：适度减仓
	lpm.reduceLayerPositions(layerID, allocation, 0.1) // 减少10%仓位
}

// sendRiskAlert 发送风险告警
func (lpm *LayeredPositionManager) sendRiskAlert(layerID int, riskType string, value float64) {
	log.Printf("RISK ALERT: Layer %d - %s exceeded threshold: %.4f", layerID, riskType, value)

	// 在实际实现中，这里会：
	// 1. 发送邮件通知
	// 2. 发送短信告警
	// 3. 推送到监控系统
	// 4. 记录到告警日志
}

// recordRiskEvent 记录风险事件
func (lpm *LayeredPositionManager) recordRiskEvent(layerID int, riskType string, value float64) {
	log.Printf("Recording risk event: Layer %d, Type: %s, Value: %.4f", layerID, riskType, value)

	// 在实际实现中，这里会：
	// 1. 保存到数据库
	// 2. 更新风险统计
	// 3. 触发风险报告生成
}

// reduceLayerPositions 减少层级仓位
func (lpm *LayeredPositionManager) reduceLayerPositions(layerID int, allocation *LayerAllocation, reductionRatio float64) {
	log.Printf("Reducing layer %d positions by %.2f%%", layerID, reductionRatio*100)

	for symbol, position := range allocation.Positions {
		if position.Quantity == 0 {
			continue
		}

		// 计算减少数量
		reductionQuantity := position.Quantity * reductionRatio

		// 更新仓位
		position.Quantity -= reductionQuantity

		// 更新价值
		currentPrice, err := lpm.getMarketPrice(symbol)
		if err == nil {
			position.Value = position.Quantity * currentPrice
		}

		log.Printf("Reduced position %s by %.4f (%.2f%%)", symbol, reductionQuantity, reductionRatio*100)
	}

	// 更新已用资金
	allocation.UsedFunds *= (1.0 - reductionRatio)
}

// reduceLeverage 降低杠杆
func (lpm *LayeredPositionManager) reduceLeverage(layerID int, allocation *LayerAllocation, targetRatio float64) {
	log.Printf("Reducing leverage for layer %d to %.2f%%", layerID, targetRatio*100)

	for _, position := range allocation.Positions {
		if position.Leverage > 1.0 {
			position.Leverage *= targetRatio
			position.Leverage = math.Max(1.0, position.Leverage) // 最低1倍杠杆
		}
	}
}

// pauseNewPositions 暂停新开仓
func (lpm *LayeredPositionManager) pauseNewPositions(layerID int, duration time.Duration) {
	log.Printf("Pausing new positions for layer %d for %v", layerID, duration)

	// 在实际实现中，这里会设置一个标志位，阻止新开仓
	// 可以使用定时器在指定时间后自动恢复
}

// adjustLayerLeverage 调整层级杠杆
func (lpm *LayeredPositionManager) adjustLayerLeverage(layerID int, allocation *LayerAllocation, targetLeverage float64) {
	log.Printf("Adjusting layer %d leverage to %.2f", layerID, targetLeverage)

	for _, position := range allocation.Positions {
		position.Leverage = targetLeverage
	}
}

// reduceHighRiskPositions 减少高风险仓位
func (lpm *LayeredPositionManager) reduceHighRiskPositions(layerID int, allocation *LayerAllocation) {
	log.Printf("Reducing high risk positions for layer %d", layerID)

	// 识别高风险仓位（高杠杆、高波动率）
	for symbol, position := range allocation.Positions {
		if position.Leverage > 3.0 { // 杠杆超过3倍认为是高风险
			// 减少50%仓位
			position.Quantity *= 0.5

			// 更新价值
			currentPrice, err := lpm.getMarketPrice(symbol)
			if err == nil {
				position.Value = position.Quantity * currentPrice
			}

			log.Printf("Reduced high risk position %s by 50%%", symbol)
		}
	}
}

// diversifyPositions 分散化仓位
func (lpm *LayeredPositionManager) diversifyPositions(layerID int, allocation *LayerAllocation) {
	log.Printf("Diversifying positions for layer %d", layerID)

	// 在实际实现中，这里会：
	// 1. 分析当前仓位分布
	// 2. 识别过度集中的资产
	// 3. 重新分配权重
	// 4. 执行再平衡交易
}

// reduceLargestPositions 减少最大仓位
func (lpm *LayeredPositionManager) reduceLargestPositions(layerID int, allocation *LayerAllocation, reductionRatio float64) {
	log.Printf("Reducing largest positions for layer %d by %.2f%%", layerID, reductionRatio*100)

	// 找到最大的仓位
	var maxPosition *Position
	var maxSymbol string
	maxValue := 0.0

	for symbol, position := range allocation.Positions {
		if position.Value > maxValue {
			maxValue = position.Value
			maxPosition = position
			maxSymbol = symbol
		}
	}

	if maxPosition != nil {
		// 减少最大仓位
		maxPosition.Quantity *= (1.0 - reductionRatio)

		// 更新价值
		currentPrice, err := lpm.getMarketPrice(maxSymbol)
		if err == nil {
			maxPosition.Value = maxPosition.Quantity * currentPrice
		}

		log.Printf("Reduced largest position %s by %.2f%%", maxSymbol, reductionRatio*100)
	}
}

// reduceLowLiquidityPositions 减少低流动性仓位
func (lpm *LayeredPositionManager) reduceLowLiquidityPositions(layerID int, allocation *LayerAllocation) {
	log.Printf("Reducing low liquidity positions for layer %d", layerID)

	// 在实际实现中，这里会：
	// 1. 评估每个资产的流动性
	// 2. 识别低流动性资产
	// 3. 减少这些资产的仓位

	// 简化实现：减少所有仓位的10%
	for symbol, position := range allocation.Positions {
		position.Quantity *= 0.9

		// 更新价值
		currentPrice, err := lpm.getMarketPrice(symbol)
		if err == nil {
			position.Value = position.Quantity * currentPrice
		}

		log.Printf("Reduced position %s for liquidity risk", symbol)
	}
}

// increaseCashRatio 增加现金比例
func (lpm *LayeredPositionManager) increaseCashRatio(layerID int, allocation *LayerAllocation, cashRatio float64) {
	log.Printf("Increasing cash ratio for layer %d by %.2f%%", layerID, cashRatio*100)

	// 计算需要转换为现金的金额
	cashAmount := allocation.AllocatedFunds * cashRatio

	// 减少仓位以释放现金
	lpm.reduceLayerPositions(layerID, allocation, cashRatio)

	// 更新可用资金
	allocation.AvailableFunds += cashAmount
	allocation.UsedFunds -= cashAmount

	log.Printf("Increased cash ratio for layer %d, available funds: %.2f", layerID, allocation.AvailableFunds)
}

// reduceHighCorrelationPositions 减少高相关性仓位
func (lpm *LayeredPositionManager) reduceHighCorrelationPositions(layerID int, allocation *LayerAllocation) {
	log.Printf("Reducing high correlation positions for layer %d", layerID)

	// 在实际实现中，这里会：
	// 1. 计算仓位间的相关性
	// 2. 识别高相关性的仓位组合
	// 3. 减少其中一些仓位以降低相关性风险

	// 简化实现：随机减少一些仓位
	count := 0
	for symbol, position := range allocation.Positions {
		if count >= len(allocation.Positions)/2 { // 只处理一半的仓位
			break
		}

		position.Quantity *= 0.8 // 减少20%

		// 更新价值
		currentPrice, err := lpm.getMarketPrice(symbol)
		if err == nil {
			position.Value = position.Quantity * currentPrice
		}

		log.Printf("Reduced correlated position %s", symbol)
		count++
	}
}

// addHedgePositions 添加对冲仓位
func (lpm *LayeredPositionManager) addHedgePositions(layerID int, allocation *LayerAllocation) {
	log.Printf("Adding hedge positions for layer %d", layerID)

	// 在实际实现中，这里会：
	// 1. 分析当前仓位的风险敞口
	// 2. 选择合适的对冲工具
	// 3. 计算对冲比例
	// 4. 执行对冲交易

	log.Printf("Hedge positions analysis completed for layer %d", layerID)
}

func (lpm *LayeredPositionManager) identifyBestWorstLayers() {
	bestLayer := -1
	worstLayer := -1
	bestReturn := math.Inf(-1)
	worstReturn := math.Inf(1)

	for layerID, performance := range lpm.managementMetrics.LayerPerformances {
		if performance.AnnualizedReturn > bestReturn {
			bestReturn = performance.AnnualizedReturn
			bestLayer = layerID
		}
		if performance.AnnualizedReturn < worstReturn {
			worstReturn = performance.AnnualizedReturn
			worstLayer = layerID
		}
	}

	lpm.managementMetrics.BestPerformingLayer = bestLayer
	lpm.managementMetrics.WorstPerformingLayer = worstLayer
}

func (lpm *LayeredPositionManager) recordAllocationSnapshot() {
	snapshot := AllocationSnapshot{
		Timestamp:        time.Now(),
		TotalFunds:       lpm.totalFunds,
		LayerAllocations: make(map[int]float64),
		Positions:        make(map[string]Position),
		RiskMetrics:      make(map[int]LayerRiskMetrics),
		Performance:      make(map[int]LayerPerformance),
		MarketConditions: make(map[string]float64),
	}

	for layerID, allocation := range lpm.layerAllocations {
		snapshot.LayerAllocations[layerID] = allocation.AllocatedFunds
		snapshot.RiskMetrics[layerID] = allocation.RiskMetrics
		snapshot.Performance[layerID] = allocation.Performance

		for symbol, position := range allocation.Positions {
			snapshot.Positions[symbol] = *position
		}
	}

	lpm.allocationHistory = append(lpm.allocationHistory, snapshot)

	// 保持历史记录在合理范围内
	if len(lpm.allocationHistory) > 1000 {
		lpm.allocationHistory = lpm.allocationHistory[100:]
	}
}

func (lpm *LayeredPositionManager) updateMetrics() {
	lpm.managementMetrics.mu.Lock()
	defer lpm.managementMetrics.mu.Unlock()

	// 计算分配效率
	lpm.managementMetrics.AllocationEfficiency = lpm.calculateAllocationEfficiency()

	// 计算再平衡频率
	lpm.managementMetrics.RebalanceFrequency = lpm.calculateRebalanceFrequency()

	// 计算平均再平衡成本
	lpm.managementMetrics.AverageRebalanceCost = lpm.calculateAverageRebalanceCost()

	// 更新系统指标
	lpm.managementMetrics.TotalPositions = len(lpm.currentPositions)
	lpm.managementMetrics.ActiveLayers = len(lpm.layerAllocations)
	lpm.managementMetrics.LastOptimization = time.Now()
	lpm.managementMetrics.LastUpdated = time.Now()
}

func (lpm *LayeredPositionManager) calculateAllocationEfficiency() float64 {
	// 实现分配效率计算
	// 分配效率衡量资金配置的有效性，考虑多个因素

	if len(lpm.layerAllocations) == 0 {
		return 0.0
	}

	totalEfficiency := 0.0
	validLayers := 0

	for layerID, allocation := range lpm.layerAllocations {
		if allocation.AllocatedFunds == 0 {
			continue
		}

		// 1. 资金利用率效率
		utilizationEfficiency := allocation.UsedFunds / allocation.AllocatedFunds
		utilizationEfficiency = math.Min(1.0, utilizationEfficiency) // 限制在100%以内

		// 2. 收益效率（实际收益 vs 预期收益）
		performance := lpm.managementMetrics.LayerPerformances[layerID]
		expectedReturn := lpm.getExpectedReturn(layerID)
		var returnEfficiency float64
		if expectedReturn > 0 {
			returnEfficiency = performance.AnnualizedReturn / expectedReturn
			returnEfficiency = math.Max(0.0, math.Min(2.0, returnEfficiency)) // 限制在0-200%
		} else {
			returnEfficiency = 1.0
		}

		// 3. 风险调整效率（夏普比率相关）
		riskEfficiency := math.Max(0.0, math.Min(1.0, (performance.SharpeRatio+1.0)/3.0)) // 标准化夏普比率

		// 4. 再平衡效率（频率适中为最优）
		rebalanceEfficiency := lpm.calculateRebalanceEfficiency(layerID, allocation)

		// 5. 多样化效率
		diversificationEfficiency := lpm.calculateDiversificationEfficiency(allocation)

		// 综合效率计算（加权平均）
		layerEfficiency := (utilizationEfficiency*0.25 +
			returnEfficiency*0.30 +
			riskEfficiency*0.25 +
			rebalanceEfficiency*0.10 +
			diversificationEfficiency*0.10)

		totalEfficiency += layerEfficiency
		validLayers++

		log.Printf("Layer %d efficiency: Util=%.3f, Return=%.3f, Risk=%.3f, Rebal=%.3f, Div=%.3f, Total=%.3f",
			layerID, utilizationEfficiency, returnEfficiency, riskEfficiency,
			rebalanceEfficiency, diversificationEfficiency, layerEfficiency)
	}

	if validLayers == 0 {
		return 0.0
	}

	overallEfficiency := totalEfficiency / float64(validLayers)

	log.Printf("Overall allocation efficiency: %.3f (%.1f%%)", overallEfficiency, overallEfficiency*100)
	return overallEfficiency
}

// getExpectedReturn 获取预期收益率
func (lpm *LayeredPositionManager) getExpectedReturn(layerID int) float64 {
	if layerID >= len(lpm.layers) {
		return 0.08 // 默认8%预期收益
	}

	layer := lpm.layers[layerID]

	// 根据风险等级设定预期收益率
	switch layer.RiskLevel {
	case "LOW":
		return 0.06 // 6%
	case "MEDIUM":
		return 0.10 // 10%
	case "HIGH":
		return 0.15 // 15%
	default:
		return 0.08 // 8%
	}
}

// calculateRebalanceEfficiency 计算再平衡效率
func (lpm *LayeredPositionManager) calculateRebalanceEfficiency(layerID int, allocation *LayerAllocation) float64 {
	// 再平衡效率基于频率和成本
	// 过于频繁或过于稀少的再平衡都不理想

	timeSinceLastRebalance := time.Since(allocation.LastRebalance)
	daysSinceRebalance := timeSinceLastRebalance.Hours() / 24

	// 理想的再平衡间隔是7-30天
	var efficiency float64
	if daysSinceRebalance < 7 {
		// 过于频繁
		efficiency = daysSinceRebalance / 7.0
	} else if daysSinceRebalance > 30 {
		// 过于稀少
		efficiency = 30.0 / daysSinceRebalance
	} else {
		// 理想范围
		efficiency = 1.0
	}

	return math.Max(0.0, math.Min(1.0, efficiency))
}

// calculateDiversificationEfficiency 计算多样化效率
func (lpm *LayeredPositionManager) calculateDiversificationEfficiency(allocation *LayerAllocation) float64 {
	positionCount := len(allocation.Positions)

	if positionCount == 0 {
		return 0.0
	}

	// 理想的仓位数量是5-15个
	var efficiency float64
	if positionCount < 5 {
		// 多样化不足
		efficiency = float64(positionCount) / 5.0
	} else if positionCount > 15 {
		// 过度多样化
		efficiency = 15.0 / float64(positionCount)
	} else {
		// 理想范围
		efficiency = 1.0
	}

	// 考虑权重分布的均匀性
	if allocation.AllocatedFunds > 0 {
		var weights []float64
		for _, position := range allocation.Positions {
			weight := position.Value / allocation.AllocatedFunds
			weights = append(weights, weight)
		}

		// 计算权重分布的均匀性（基于方差）
		if len(weights) > 1 {
			mean := lpm.calculateMean(weights)
			variance := 0.0
			for _, w := range weights {
				diff := w - mean
				variance += diff * diff
			}
			variance /= float64(len(weights))

			// 方差越小，分布越均匀，效率越高
			uniformityBonus := math.Max(0.0, 1.0-variance*10) // 标准化方差影响
			efficiency = (efficiency + uniformityBonus) / 2.0
		}
	}

	return math.Max(0.0, math.Min(1.0, efficiency))
}

// calculateConservativeAllocation 计算保守层分配
func (lpm *LayeredPositionManager) calculateConservativeAllocation(layer PositionLayer) map[string]float64 {
	allocation := make(map[string]float64)

	// 保守层：主要配置稳定资产，低风险
	allocation["BTC"] = 0.50  // 50% BTC
	allocation["ETH"] = 0.30  // 30% ETH
	allocation["USDT"] = 0.15 // 15% 稳定币
	allocation["BNB"] = 0.05  // 5% 平台币

	return allocation
}

// calculateModerateAllocation 计算稳健层分配
func (lpm *LayeredPositionManager) calculateModerateAllocation(layer PositionLayer) map[string]float64 {
	allocation := make(map[string]float64)

	// 稳健层：平衡配置，中等风险
	allocation["BTC"] = 0.35  // 35% BTC
	allocation["ETH"] = 0.25  // 25% ETH
	allocation["BNB"] = 0.15  // 15% BNB
	allocation["ADA"] = 0.10  // 10% ADA
	allocation["DOT"] = 0.08  // 8% DOT
	allocation["LINK"] = 0.07 // 7% LINK

	return allocation
}

// calculateAggressiveAllocation 计算进取层分配
func (lpm *LayeredPositionManager) calculateAggressiveAllocation(layer PositionLayer) map[string]float64 {
	allocation := make(map[string]float64)

	// 进取层：多样化配置，高风险高收益
	allocation["BTC"] = 0.25   // 25% BTC
	allocation["ETH"] = 0.20   // 20% ETH
	allocation["BNB"] = 0.15   // 15% BNB
	allocation["SOL"] = 0.10   // 10% SOL
	allocation["AVAX"] = 0.08  // 8% AVAX
	allocation["MATIC"] = 0.07 // 7% MATIC
	allocation["UNI"] = 0.06   // 6% UNI
	allocation["AAVE"] = 0.05  // 5% AAVE
	allocation["SUSHI"] = 0.04 // 4% SUSHI

	return allocation
}

// calculateBalancedAllocation 计算均衡分配
func (lpm *LayeredPositionManager) calculateBalancedAllocation(layer PositionLayer) map[string]float64 {
	allocation := make(map[string]float64)

	// 均衡分配：标准配置
	allocation["BTC"] = 0.40    // 40% BTC
	allocation["ETH"] = 0.30    // 30% ETH
	allocation["BNB"] = 0.15    // 15% BNB
	allocation["others"] = 0.15 // 15% 其他

	return allocation
}

// applyLayerConstraints 应用层级约束
func (lpm *LayeredPositionManager) applyLayerConstraints(layerID int, allocation map[string]float64) map[string]float64 {
	if layerID >= len(lpm.layers) {
		return allocation
	}

	layer := lpm.layers[layerID]
	constraints := layer.Constraints

	// 应用最大资产集中度约束
	for asset, weight := range allocation {
		if weight > constraints.MaxAssetConcentration {
			log.Printf("Asset %s weight %.4f exceeds max concentration %.4f for layer %d",
				asset, weight, constraints.MaxAssetConcentration, layerID)
			allocation[asset] = constraints.MaxAssetConcentration
		}

		// 应用最小仓位约束
		if weight < constraints.MinPositionSize && weight > 0 {
			allocation[asset] = constraints.MinPositionSize
		}
	}

	// 检查允许的资产
	if len(layer.AllowedAssets) > 0 && layer.AllowedAssets[0] != "*" {
		allowedMap := make(map[string]bool)
		for _, asset := range layer.AllowedAssets {
			allowedMap[asset] = true
		}

		// 移除不允许的资产
		for asset := range allocation {
			if !allowedMap[asset] {
				log.Printf("Removing disallowed asset %s from layer %d", asset, layerID)
				delete(allocation, asset)
			}
		}
	}

	// 确保满足最小多样化要求
	if len(allocation) < constraints.RequiredDiversification {
		log.Printf("Layer %d diversification %d below required %d",
			layerID, len(allocation), constraints.RequiredDiversification)

		// 添加额外资产以满足多样化要求
		additionalAssets := []string{"ADA", "DOT", "LINK", "UNI", "AAVE"}
		needed := constraints.RequiredDiversification - len(allocation)

		for i := 0; i < needed && i < len(additionalAssets); i++ {
			asset := additionalAssets[i]
			if _, exists := allocation[asset]; !exists {
				allocation[asset] = constraints.MinPositionSize
			}
		}
	}

	return allocation
}

func (lpm *LayeredPositionManager) calculateRebalanceFrequency() float64 {
	lpm.rebalancer.mu.RLock()
	defer lpm.rebalancer.mu.RUnlock()

	if len(lpm.rebalancer.rebalanceHistory) < 2 {
		return 0.0
	}

	// 计算最近30天的再平衡次数
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	count := 0

	for _, event := range lpm.rebalancer.rebalanceHistory {
		if event.Timestamp.After(thirtyDaysAgo) {
			count++
		}
	}

	return float64(count) / 30.0 // 每日平均再平衡次数
}

func (lpm *LayeredPositionManager) calculateAverageRebalanceCost() float64 {
	lpm.rebalancer.mu.RLock()
	defer lpm.rebalancer.mu.RUnlock()

	if len(lpm.rebalancer.rebalanceHistory) == 0 {
		return 0.0
	}

	totalCost := 0.0
	for _, event := range lpm.rebalancer.rebalanceHistory {
		totalCost += event.TotalCost
	}

	return totalCost / float64(len(lpm.rebalancer.rebalanceHistory))
}

func (lpm *LayeredPositionManager) generateRebalanceID() string {
	return fmt.Sprintf("RBL_%d", time.Now().Unix())
}

// GetStatus 获取管理器状态
func (lpm *LayeredPositionManager) GetStatus() map[string]interface{} {
	lpm.mu.RLock()
	defer lpm.mu.RUnlock()

	return map[string]interface{}{
		"running":              lpm.isRunning,
		"enabled":              lpm.enabled,
		"total_funds":          lpm.totalFunds,
		"layer_count":          lpm.layerCount,
		"layer_allocations":    lpm.layerAllocations,
		"management_metrics":   lpm.managementMetrics,
		"rebalance_threshold":  lpm.rebalanceThreshold,
		"last_rebalance":       lpm.rebalancer.lastRebalance,
		"allocation_snapshots": len(lpm.allocationHistory),
	}
}

// GetLayerPerformance 获取层级表现
func (lpm *LayeredPositionManager) GetLayerPerformance(layerID int) (*LayerPerformance, error) {
	lpm.managementMetrics.mu.RLock()
	defer lpm.managementMetrics.mu.RUnlock()

	if performance, exists := lpm.managementMetrics.LayerPerformances[layerID]; exists {
		return &performance, nil
	}

	return nil, fmt.Errorf("layer %d not found", layerID)
}

// GetManagementMetrics 获取管理指标
func (lpm *LayeredPositionManager) GetManagementMetrics() *ManagementMetrics {
	lpm.managementMetrics.mu.RLock()
	defer lpm.managementMetrics.mu.RUnlock()

	metrics := *lpm.managementMetrics
	return &metrics
}

// calculateCurrentWeights 计算当前权重
func (lpm *LayeredPositionManager) calculateCurrentWeights(layerID int, allocation *LayerAllocation) map[string]float64 {
	weights := make(map[string]float64)

	if allocation.AllocatedFunds == 0 {
		return weights
	}

	// 计算每个资产的当前权重
	for symbol, position := range allocation.Positions {
		if position != nil {
			// 计算仓位价值
			currentPrice, err := lpm.getCurrentPrice(symbol)
			if err != nil {
				log.Printf("Failed to get price for %s: %v", symbol, err)
				continue
			}

			positionValue := position.Quantity * currentPrice
			weight := positionValue / allocation.AllocatedFunds
			weights[symbol] = weight
		}
	}

	return weights
}

// getCurrentPrice 获取当前价格
func (lpm *LayeredPositionManager) getCurrentPrice(symbol string) (float64, error) {
	// 模拟获取当前市场价格
	// 在实际实现中，这里会调用交易所API

	prices := map[string]float64{
		"BTC":   45000.0,
		"ETH":   3000.0,
		"BNB":   300.0,
		"ADA":   0.5,
		"DOT":   8.0,
		"LINK":  15.0,
		"SOL":   100.0,
		"AVAX":  25.0,
		"MATIC": 1.2,
		"UNI":   8.5,
		"AAVE":  120.0,
		"SUSHI": 2.5,
		"USDT":  1.0,
	}

	if price, exists := prices[symbol]; exists {
		// 添加一些随机波动 (±1%)
		variation := (rand.Float64() - 0.5) * 0.02
		return price * (1 + variation), nil
	}

	return 0, fmt.Errorf("price not found for symbol: %s", symbol)
}

// getAssetReturnMultiplier 获取资产收益倍数
func (lpm *LayeredPositionManager) getAssetReturnMultiplier(symbol string) float64 {
	// 根据资产的历史表现和风险特征设置收益倍数
	multipliers := map[string]float64{
		"BTC":   1.0, // 基准
		"ETH":   1.1, // 稍高于BTC
		"BNB":   1.2, // 平台币有额外收益
		"ADA":   0.9, // 相对保守
		"DOT":   1.0,
		"LINK":  1.1,
		"SOL":   1.3, // 高增长潜力
		"AVAX":  1.2,
		"MATIC": 1.1,
		"UNI":   1.0,
		"AAVE":  1.1,
		"SUSHI": 0.8, // 风险较高
		"USDT":  0.1, // 稳定币低收益
	}

	if multiplier, exists := multipliers[symbol]; exists {
		return multiplier
	}

	return 1.0 // 默认倍数
}

// calculateDiversificationBenefit 计算多样化收益
func (lpm *LayeredPositionManager) calculateDiversificationBenefit(change PositionChange) float64 {
	// 计算再平衡带来的风险分散收益

	// 获取层级当前的多样化程度
	layerAllocation, exists := lpm.layerAllocations[change.LayerID]
	if !exists {
		return 0
	}

	currentDiversification := len(layerAllocation.Positions)

	// 多样化收益基于资产数量和权重分布
	diversificationScore := 0.0

	switch change.ChangeType {
	case "ADD":
		// 新增资产提高多样化
		diversificationScore = 0.02 * math.Log(float64(currentDiversification+1))
	case "REMOVE":
		// 移除资产可能降低多样化，但如果是低效资产则有益
		if change.OldWeight < 0.05 { // 权重小于5%的资产
			diversificationScore = 0.01 // 移除小权重资产有小幅收益
		} else {
			diversificationScore = -0.01 // 移除大权重资产有负面影响
		}
	case "ADJUST":
		// 调整权重的多样化影响
		weightChange := math.Abs(change.NewWeight - change.OldWeight)
		if weightChange > 0.1 { // 权重变化超过10%
			diversificationScore = 0.005 // 小幅多样化收益
		}
	}

	// 计算绝对收益
	positionValue := change.Quantity * change.Price
	benefit := positionValue * diversificationScore

	return benefit
}

// getMarketConditionAdjustment 获取市场条件调整因子
func (lpm *LayeredPositionManager) getMarketConditionAdjustment() float64 {
	// 根据当前市场条件调整收益预期
	// 这里可以集成市场情绪、波动率、流动性等因素

	// 模拟市场条件评分 (0.8 - 1.2)
	// 1.0 = 正常市场条件
	// > 1.0 = 有利市场条件
	// < 1.0 = 不利市场条件

	baseAdjustment := 1.0

	// 模拟不同的市场条件
	marketConditions := []string{"BULL", "BEAR", "SIDEWAYS", "VOLATILE"}
	currentCondition := marketConditions[int(rand.Float64()*float64(len(marketConditions)))]

	switch currentCondition {
	case "BULL":
		baseAdjustment = 1.1 // 牛市中收益预期提高10%
	case "BEAR":
		baseAdjustment = 0.85 // 熊市中收益预期降低15%
	case "VOLATILE":
		baseAdjustment = 0.95 // 高波动市场收益预期降低5%
	case "SIDEWAYS":
		baseAdjustment = 0.9 // 横盘市场收益预期降低10%
	}

	// 添加一些随机性 (±5%)
	randomFactor := 0.95 + rand.Float64()*0.1

	return baseAdjustment * randomFactor
}

// sortChangesByPriority 按优先级排序变化
func (lpm *LayeredPositionManager) sortChangesByPriority(changes []PositionChange) []PositionChange {
	// 复制切片以避免修改原始数据
	sortedChanges := make([]PositionChange, len(changes))
	copy(sortedChanges, changes)

	// 定义优先级：REMOVE > ADJUST > ADD
	getPriority := func(changeType string) int {
		switch changeType {
		case "REMOVE":
			return 1 // 最高优先级
		case "ADJUST":
			return 2 // 中等优先级
		case "ADD":
			return 3 // 最低优先级
		default:
			return 4
		}
	}

	// 排序
	for i := 0; i < len(sortedChanges)-1; i++ {
		for j := i + 1; j < len(sortedChanges); j++ {
			if getPriority(sortedChanges[i].ChangeType) > getPriority(sortedChanges[j].ChangeType) {
				sortedChanges[i], sortedChanges[j] = sortedChanges[j], sortedChanges[i]
			}
		}
	}

	return sortedChanges
}

// executeSingleTrade 执行单个交易
func (lpm *LayeredPositionManager) executeSingleTrade(change PositionChange) error {
	// 模拟交易执行
	// 在实际实现中，这里会调用交易所API

	log.Printf("Executing %s trade: %s %.4f @ %.2f",
		change.ChangeType, change.Symbol, change.Quantity, change.Price)

	// 模拟执行延迟
	executionDelay := time.Duration(50+int(rand.Float64()*100)) * time.Millisecond
	time.Sleep(executionDelay)

	// 模拟执行成功率（95%）
	if rand.Float64() < 0.05 {
		return fmt.Errorf("simulated execution failure for %s", change.Symbol)
	}

	// 模拟滑点
	slippage := rand.Float64() * 0.002 // 0-0.2%滑点
	actualPrice := change.Price
	if change.ChangeType == "ADD" || (change.ChangeType == "ADJUST" && change.NewWeight > change.OldWeight) {
		// 买入时价格可能上涨
		actualPrice *= (1 + slippage)
	} else {
		// 卖出时价格可能下跌
		actualPrice *= (1 - slippage)
	}

	log.Printf("Trade executed successfully: %s %.4f @ %.2f (slippage: %.4f%%)",
		change.Symbol, change.Quantity, actualPrice, slippage*100)

	return nil
}

// recordFailedTrade 记录失败的交易
func (lpm *LayeredPositionManager) recordFailedTrade(change PositionChange, err error) {
	log.Printf("Recording failed trade: %s %s %.4f - %v",
		change.ChangeType, change.Symbol, change.Quantity, err)

	// 在实际实现中，这里会记录到数据库或日志系统
	// 可以用于后续的重试或分析
}

// updatePositionAfterTrade 交易后更新仓位
func (lpm *LayeredPositionManager) updatePositionAfterTrade(change PositionChange) error {
	layerAllocation, exists := lpm.layerAllocations[change.LayerID]
	if !exists {
		return fmt.Errorf("layer allocation not found for layer %d", change.LayerID)
	}

	switch change.ChangeType {
	case "ADD":
		// 新增仓位
		position := &Position{
			Symbol:     change.Symbol,
			LayerID:    change.LayerID,
			Quantity:   change.Quantity,
			Price:      change.Price,
			Value:      change.Quantity * change.Price,
			Weight:     change.NewWeight,
			Side:       "LONG",
			Status:     "ACTIVE",
			OpenTime:   time.Now(),
			LastUpdate: time.Now(),
		}
		layerAllocation.Positions[change.Symbol] = position
		layerAllocation.UsedFunds += change.Quantity * change.Price

	case "REMOVE":
		// 移除仓位
		if position, exists := layerAllocation.Positions[change.Symbol]; exists {
			layerAllocation.UsedFunds -= position.Quantity * change.Price
			delete(layerAllocation.Positions, change.Symbol)
		}

	case "ADJUST":
		// 调整仓位
		if position, exists := layerAllocation.Positions[change.Symbol]; exists {
			oldValue := position.Quantity * position.Price
			newQuantity := change.Quantity
			newValue := newQuantity * change.Price

			position.Quantity = newQuantity
			position.Price = change.Price
			position.Value = newValue
			position.Weight = change.NewWeight
			position.LastUpdate = time.Now()

			layerAllocation.UsedFunds += (newValue - oldValue)
		}
	}

	// 更新可用资金
	layerAllocation.AvailableFunds = layerAllocation.AllocatedFunds - layerAllocation.UsedFunds

	log.Printf("Position updated for layer %d: %s %s %.4f",
		change.LayerID, change.ChangeType, change.Symbol, change.Quantity)

	return nil
}

// recordRebalanceExecution 记录再平衡执行
func (lpm *LayeredPositionManager) recordRebalanceExecution(changes []PositionChange, successCount, failCount int) {
	rebalanceEvent := RebalanceEvent{
		ID:              fmt.Sprintf("rebalance_%d", time.Now().Unix()),
		Timestamp:       time.Now(),
		Type:            "SCHEDULED",
		Trigger:         "AUTOMATIC",
		Changes:         changes,
		TotalCost:       lpm.calculateTotalCost(changes),
		ExpectedBenefit: lpm.calculateExpectedBenefit(changes),
		ActualBenefit:   lpm.calculateActualBenefit(changes),
		Success:         failCount == 0,
		Metadata: map[string]interface{}{
			"success_count": successCount,
			"fail_count":    failCount,
			"total_changes": len(changes),
		},
	}

	// 添加到再平衡历史
	lpm.rebalancer.mu.Lock()
	lpm.rebalancer.rebalanceHistory = append(lpm.rebalancer.rebalanceHistory, rebalanceEvent)
	lpm.rebalancer.lastRebalance = time.Now()
	lpm.rebalancer.mu.Unlock()

	log.Printf("Rebalance execution recorded: %d successful, %d failed, total cost: %.2f",
		successCount, failCount, rebalanceEvent.TotalCost)
}

// calculateTotalCost 计算总成本
func (lpm *LayeredPositionManager) calculateTotalCost(changes []PositionChange) float64 {
	totalCost := 0.0

	for _, change := range changes {
		totalCost += change.Cost
	}

	return totalCost
}
