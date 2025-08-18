package management

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"

	"qcat/internal/config"
)

// LayeredPositionManager 分层仓位管理器
type LayeredPositionManager struct {
	config              *config.Config
	positionAllocator   *PositionAllocator
	rebalancer          *Rebalancer
	riskManager         *LayeredRiskManager
	
	// 运行状态
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	isRunning  bool
	mu         sync.RWMutex
	
	// 分层配置
	layers            []PositionLayer
	totalFunds        float64
	rebalanceThreshold float64
	
	// 仓位状态
	currentPositions  map[string]*Position
	layerAllocations  map[int]*LayerAllocation
	
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
	ID            int               `json:"id"`
	Name          string            `json:"name"`
	AllocationPct float64           `json:"allocation_pct"`
	RiskLevel     string            `json:"risk_level"`
	Strategy      string            `json:"strategy"`
	MaxLeverage   float64           `json:"max_leverage"`
	MaxDrawdown   float64           `json:"max_drawdown"`
	AllowedAssets []string          `json:"allowed_assets"`
	Constraints   LayerConstraints  `json:"constraints"`
	Performance   LayerPerformance  `json:"performance"`
}

// LayerConstraints 层级约束
type LayerConstraints struct {
	MaxPositionSize     float64   `json:"max_position_size"`
	MinPositionSize     float64   `json:"min_position_size"`
	MaxAssetConcentration float64 `json:"max_asset_concentration"`
	MaxSectorConcentration float64 `json:"max_sector_concentration"`
	RequiredDiversification int    `json:"required_diversification"`
	AllowedInstruments  []string  `json:"allowed_instruments"`
	ForbiddenAssets     []string  `json:"forbidden_assets"`
}

// LayerPerformance 层级表现
type LayerPerformance struct {
	mu sync.RWMutex
	
	TotalReturn       float64   `json:"total_return"`
	AnnualizedReturn  float64   `json:"annualized_return"`
	Volatility        float64   `json:"volatility"`
	SharpeRatio       float64   `json:"sharpe_ratio"`
	MaxDrawdown       float64   `json:"max_drawdown"`
	CalmarRatio       float64   `json:"calmar_ratio"`
	WinRate           float64   `json:"win_rate"`
	ProfitFactor      float64   `json:"profit_factor"`
	
	LastUpdated       time.Time `json:"last_updated"`
}

// Position 仓位信息
type Position struct {
	Symbol        string    `json:"symbol"`
	LayerID       int       `json:"layer_id"`
	Quantity      float64   `json:"quantity"`
	Price         float64   `json:"price"`
	Value         float64   `json:"value"`
	Weight        float64   `json:"weight"`
	Side          string    `json:"side"`      // LONG, SHORT
	Leverage      float64   `json:"leverage"`
	Margin        float64   `json:"margin"`
	UnrealizedPL  float64   `json:"unrealized_pl"`
	RealizedPL    float64   `json:"realized_pl"`
	OpenTime      time.Time `json:"open_time"`
	LastUpdate    time.Time `json:"last_update"`
	Status        string    `json:"status"`    // ACTIVE, CLOSING, CLOSED
}

// LayerAllocation 层级分配
type LayerAllocation struct {
	LayerID         int                 `json:"layer_id"`
	AllocatedFunds  float64             `json:"allocated_funds"`
	UsedFunds       float64             `json:"used_funds"`
	AvailableFunds  float64             `json:"available_funds"`
	Positions       map[string]*Position `json:"positions"`
	Performance     LayerPerformance    `json:"performance"`
	RiskMetrics     LayerRiskMetrics    `json:"risk_metrics"`
	LastRebalance   time.Time           `json:"last_rebalance"`
	RebalanceNeeded bool                `json:"rebalance_needed"`
}

// LayerRiskMetrics 层级风险指标
type LayerRiskMetrics struct {
	CurrentVaR        float64 `json:"current_var"`
	ExpectedShortfall float64 `json:"expected_shortfall"`
	BetaToMarket      float64 `json:"beta_to_market"`
	CorrelationMatrix map[string]map[string]float64 `json:"correlation_matrix"`
	ConcentrationRisk float64 `json:"concentration_risk"`
	LeverageRatio     float64 `json:"leverage_ratio"`
	LiquidityRisk     float64 `json:"liquidity_risk"`
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
	strategy          string
	threshold         float64
	frequency         time.Duration
	costModel         string
	lastRebalance     time.Time
	rebalanceHistory  []RebalanceEvent
	mu                sync.RWMutex
}

// RebalanceEvent 再平衡事件
type RebalanceEvent struct {
	ID            string                 `json:"id"`
	Timestamp     time.Time              `json:"timestamp"`
	Type          string                 `json:"type"`
	Trigger       string                 `json:"trigger"`
	LayersAffected []int                 `json:"layers_affected"`
	Changes       []PositionChange       `json:"changes"`
	TotalCost     float64                `json:"total_cost"`
	ExpectedBenefit float64              `json:"expected_benefit"`
	ActualBenefit float64                `json:"actual_benefit"`
	Success       bool                   `json:"success"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// PositionChange 仓位变化
type PositionChange struct {
	Symbol      string  `json:"symbol"`
	LayerID     int     `json:"layer_id"`
	OldWeight   float64 `json:"old_weight"`
	NewWeight   float64 `json:"new_weight"`
	ChangeType  string  `json:"change_type"` // ADD, REMOVE, ADJUST
	Quantity    float64 `json:"quantity"`
	Price       float64 `json:"price"`
	Cost        float64 `json:"cost"`
	Reason      string  `json:"reason"`
}

// LayeredRiskManager 分层风险管理器
type LayeredRiskManager struct {
	riskLimits        map[int]RiskLimit
	correlationModel  string
	stressScenarios   []StressScenario
	monitoringRules   []RiskRule
	mu                sync.RWMutex
}

// RiskLimit 风险限制
type RiskLimit struct {
	LayerID           int     `json:"layer_id"`
	MaxVaR            float64 `json:"max_var"`
	MaxDrawdown       float64 `json:"max_drawdown"`
	MaxLeverage       float64 `json:"max_leverage"`
	MaxConcentration  float64 `json:"max_concentration"`
	MaxCorrelation    float64 `json:"max_correlation"`
	MinDiversification int    `json:"min_diversification"`
}

// StressScenario 压力测试场景
type StressScenario struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Parameters  map[string]float64 `json:"parameters"`
	Description string            `json:"description"`
	Severity    string            `json:"severity"`
}

// RiskRule 风险规则
type RiskRule struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Condition   string            `json:"condition"`
	Action      string            `json:"action"`
	Priority    int               `json:"priority"`
	IsEnabled   bool              `json:"is_enabled"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ManagementMetrics 管理指标
type ManagementMetrics struct {
	mu sync.RWMutex
	
	// 分配效率
	AllocationEfficiency float64   `json:"allocation_efficiency"`
	RebalanceFrequency   float64   `json:"rebalance_frequency"`
	AverageRebalanceCost float64   `json:"average_rebalance_cost"`
	
	// 风险管理
	RiskAdjustedReturn   float64   `json:"risk_adjusted_return"`
	TrackingError        float64   `json:"tracking_error"`
	InformationRatio     float64   `json:"information_ratio"`
	
	// 层级表现
	LayerPerformances    map[int]LayerPerformance `json:"layer_performances"`
	BestPerformingLayer  int       `json:"best_performing_layer"`
	WorstPerformingLayer int       `json:"worst_performing_layer"`
	
	// 系统指标
	TotalPositions       int       `json:"total_positions"`
	ActiveLayers         int       `json:"active_layers"`
	LastOptimization     time.Time `json:"last_optimization"`
	
	LastUpdated          time.Time `json:"last_updated"`
}

// AllocationSnapshot 分配快照
type AllocationSnapshot struct {
	Timestamp      time.Time                    `json:"timestamp"`
	TotalFunds     float64                      `json:"total_funds"`
	LayerAllocations map[int]float64            `json:"layer_allocations"`
	Positions      map[string]Position          `json:"positions"`
	RiskMetrics    map[int]LayerRiskMetrics     `json:"risk_metrics"`
	Performance    map[int]LayerPerformance     `json:"performance"`
	MarketConditions map[string]float64         `json:"market_conditions"`
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
		allocationHistory: make([]AllocationSnapshot, 0),
		layerCount:        3,
		layerSizes:        []float64{0.4, 0.35, 0.25},
		rebalanceThreshold: 0.05,
		rebalanceInterval:  24 * time.Hour,
		enabled:           true,
	}
	
	// 从配置文件读取参数
	if cfg != nil {
		// TODO: 从配置文件读取分层管理参数
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
			LayerID:        layer.ID,
			AllocatedFunds: 0.0,
			UsedFunds:      0.0,
			AvailableFunds: 0.0,
			Positions:      make(map[string]*Position),
			LastRebalance:  time.Now(),
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
	// TODO: 实现目标分配计算
	allocations := make(map[int]map[string]float64)
	
	for _, layerID := range layerIDs {
		allocations[layerID] = make(map[string]float64)
		// 模拟目标分配
		allocations[layerID]["BTC"] = 0.4
		allocations[layerID]["ETH"] = 0.3
		allocations[layerID]["others"] = 0.3
	}
	
	return allocations, nil
}

func (lpm *LayeredPositionManager) generateRebalanceChanges(allocations map[int]map[string]float64) ([]PositionChange, error) {
	changes := make([]PositionChange, 0)
	
	// TODO: 实现具体的变化计算逻辑
	for layerID, allocation := range allocations {
		for symbol, weight := range allocation {
			change := PositionChange{
				Symbol:     symbol,
				LayerID:    layerID,
				OldWeight:  0.0, // 从当前仓位获取
				NewWeight:  weight,
				ChangeType: "ADJUST",
				Quantity:   1000.0, // 根据权重变化计算
				Price:      50000.0, // 当前市价
				Cost:       5.0,     // 交易成本
				Reason:     "REBALANCE",
			}
			changes = append(changes, change)
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
	// TODO: 实现预期收益计算
	return 100.0 // 模拟预期收益
}

func (lpm *LayeredPositionManager) calculateActualBenefit(changes []PositionChange) float64 {
	// TODO: 实现实际收益计算
	return 95.0 // 模拟实际收益
}

func (lpm *LayeredPositionManager) executeRebalanceTrades(changes []PositionChange) error {
	// TODO: 实现实际的交易执行
	for _, change := range changes {
		log.Printf("Executing trade: %s %s %.2f @ %.2f", 
			change.ChangeType, change.Symbol, change.Quantity, change.Price)
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
	// TODO: 实现具体的风险指标计算
	return LayerRiskMetrics{
		CurrentVaR:        allocation.AllocatedFunds * 0.05, // 5% VaR
		ExpectedShortfall: allocation.AllocatedFunds * 0.07, // 7% ES
		BetaToMarket:      1.2,                             // 市场Beta
		LeverageRatio:     2.0,                             // 杠杆比率
		ConcentrationRisk: 0.3,                             // 集中度风险
		LiquidityRisk:     0.1,                             // 流动性风险
		CorrelationMatrix: make(map[string]map[string]float64),
	}
}

func (lpm *LayeredPositionManager) calculateLayerPerformance(layerID int, allocation *LayerAllocation) LayerPerformance {
	// TODO: 实现具体的性能计算
	return LayerPerformance{
		TotalReturn:      0.15,  // 15%总收益
		AnnualizedReturn: 0.12,  // 12%年化收益
		Volatility:       0.18,  // 18%波动率
		SharpeRatio:      0.67,  // 夏普比率
		MaxDrawdown:      0.08,  // 8%最大回撤
		CalmarRatio:      1.5,   // Calmar比率
		WinRate:          0.65,  // 65%胜率
		ProfitFactor:     1.8,   // 盈亏比
		LastUpdated:      time.Now(),
	}
}

func (lpm *LayeredPositionManager) triggerRiskAction(layerID int, riskType string, value float64) {
	log.Printf("Risk action triggered for layer %d: %s (value: %.4f)", layerID, riskType, value)
	
	// TODO: 实现具体的风险响应动作
	// 1. 减仓
	// 2. 降杠杆
	// 3. 调整配置
	// 4. 发送告警
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
	// TODO: 实现分配效率计算
	return 0.85 // 模拟85%效率
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
		"running":               lpm.isRunning,
		"enabled":               lpm.enabled,
		"total_funds":           lpm.totalFunds,
		"layer_count":           lpm.layerCount,
		"layer_allocations":     lpm.layerAllocations,
		"management_metrics":    lpm.managementMetrics,
		"rebalance_threshold":   lpm.rebalanceThreshold,
		"last_rebalance":        lpm.rebalancer.lastRebalance,
		"allocation_snapshots":  len(lpm.allocationHistory),
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
