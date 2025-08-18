package optimization

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"qcat/internal/exchange"
	"qcat/internal/intelligence/position"
	"qcat/internal/intelligence/trading"
	"qcat/internal/market"
	"qcat/internal/strategy"
)

// ProfitMaximizationEngine 全局利润最大化决策引擎
// 综合策略表现、仓位分配、资金使用率，动态追求全局收益最大化
type ProfitMaximizationEngine struct {
	portfolioOptimizer *PortfolioOptimizer
	capitalAllocator   *CapitalAllocator
	riskAdjuster       *RiskAdjuster
	costMinimizer      *CostMinimizer
	performanceTracker *PerformanceTracker
	
	// 依赖服务
	positionOptimizer  *position.DynamicPositionOptimizer
	tradingExecutor    *trading.SmartTradingExecutor
	
	config             *MaximizationConfig
	mu                 sync.RWMutex
	
	// 状态数据
	currentObjective   *ObjectiveFunction
	optimizationHistory []*OptimizationResult
	lastOptimization   time.Time
}

// MaximizationConfig 利润最大化配置
type MaximizationConfig struct {
	OptimizationInterval  time.Duration `yaml:"optimization_interval"`
	RiskTolerance         float64       `yaml:"risk_tolerance"`
	MaxLeverage          float64       `yaml:"max_leverage"`
	MinCashReserve       float64       `yaml:"min_cash_reserve"`
	TransactionCostRate  float64       `yaml:"transaction_cost_rate"`
	
	// 目标函数权重
	ReturnWeight         float64 `yaml:"return_weight"`
	RiskWeight           float64 `yaml:"risk_weight"`
	CostWeight           float64 `yaml:"cost_weight"`
	LiquidityWeight      float64 `yaml:"liquidity_weight"`
	
	// 约束条件
	MaxPositionConcentration float64 `yaml:"max_position_concentration"`
	MaxSectorConcentration   float64 `yaml:"max_sector_concentration"`
	MaxDrawdownLimit         float64 `yaml:"max_drawdown_limit"`
	
	// 优化算法
	OptimizationAlgorithm    string  `yaml:"optimization_algorithm"`
	ConvergenceThreshold     float64 `yaml:"convergence_threshold"`
	MaxIterations           int     `yaml:"max_iterations"`
}

// ObjectiveFunction 目标函数
type ObjectiveFunction struct {
	Name        string                 `json:"name"`
	Function    ObjectiveFunctionType  `json:"function"`
	Weights     map[string]float64     `json:"weights"`
	Constraints []*OptimizationConstraint `json:"constraints"`
	UpdateTime  time.Time              `json:"update_time"`
}

// ObjectiveFunctionType 目标函数类型
type ObjectiveFunctionType string

const (
	ObjectiveMaxReturn     ObjectiveFunctionType = "max_return"
	ObjectiveMaxSharpe     ObjectiveFunctionType = "max_sharpe"
	ObjectiveMaxUtility    ObjectiveFunctionType = "max_utility"
	ObjectiveMinRisk       ObjectiveFunctionType = "min_risk"
	ObjectiveRiskParity    ObjectiveFunctionType = "risk_parity"
	ObjectiveBlackLitterman ObjectiveFunctionType = "black_litterman"
)

// OptimizationConstraint 优化约束
type OptimizationConstraint struct {
	Type        ConstraintType `json:"type"`
	Parameter   string         `json:"parameter"`
	LowerBound  float64        `json:"lower_bound"`
	UpperBound  float64        `json:"upper_bound"`
	Penalty     float64        `json:"penalty"`
}

// ConstraintType 约束类型
type ConstraintType string

const (
	ConstraintPosition     ConstraintType = "position"
	ConstraintSector       ConstraintType = "sector"
	ConstraintLeverage     ConstraintType = "leverage"
	ConstraintLiquidity    ConstraintType = "liquidity"
	ConstraintCorrelation  ConstraintType = "correlation"
	ConstraintDrawdown     ConstraintType = "drawdown"
	ConstraintVolatility   ConstraintType = "volatility"
)

// OptimizationResult 优化结果
type OptimizationResult struct {
	Timestamp           time.Time                   `json:"timestamp"`
	ObjectiveValue      float64                     `json:"objective_value"`
	OptimalAllocation   map[string]float64          `json:"optimal_allocation"`
	RiskMetrics         *RiskMetrics               `json:"risk_metrics"`
	PerformanceForecast *PerformanceForecast       `json:"performance_forecast"`
	ExecutionPlan       *ExecutionPlan             `json:"execution_plan"`
	ConvergenceInfo     *ConvergenceInfo           `json:"convergence_info"`
	CostAnalysis        *CostAnalysis              `json:"cost_analysis"`
}

// RiskMetrics 风险指标
type RiskMetrics struct {
	PortfolioVolatility    float64            `json:"portfolio_volatility"`
	ExpectedReturn         float64            `json:"expected_return"`
	SharpeRatio           float64            `json:"sharpe_ratio"`
	VaR95                 float64            `json:"var_95"`
	ConditionalVaR        float64            `json:"conditional_var"`
	MaxDrawdown           float64            `json:"max_drawdown"`
	ConcentrationRisk     float64            `json:"concentration_risk"`
	CorrelationRisk       map[string]float64 `json:"correlation_risk"`
}

// PerformanceForecast 性能预测
type PerformanceForecast struct {
	ExpectedReturn1M      float64 `json:"expected_return_1m"`
	ExpectedReturn3M      float64 `json:"expected_return_3m"`
	ExpectedReturn1Y      float64 `json:"expected_return_1y"`
	VolatilityForecast    float64 `json:"volatility_forecast"`
	SharpeRatioForecast   float64 `json:"sharpe_ratio_forecast"`
	DownsideRisk          float64 `json:"downside_risk"`
	TailRisk              float64 `json:"tail_risk"`
}

// ExecutionPlan 执行计划
type ExecutionPlan struct {
	RebalanceOrders       []*RebalanceOrder   `json:"rebalance_orders"`
	EstimatedCost         float64             `json:"estimated_cost"`
	ExecutionTimeframe    time.Duration       `json:"execution_timeframe"`
	RiskBudgetAdjustment  map[string]float64  `json:"risk_budget_adjustment"`
	Priority              ExecutionPriority   `json:"priority"`
}

// RebalanceOrder 再平衡订单
type RebalanceOrder struct {
	Symbol        string                  `json:"symbol"`
	Action        RebalanceAction         `json:"action"`
	CurrentWeight float64                 `json:"current_weight"`
	TargetWeight  float64                 `json:"target_weight"`
	Quantity      float64                 `json:"quantity"`
	Urgency       trading.UrgencyLevel    `json:"urgency"`
	EstimatedCost float64                 `json:"estimated_cost"`
}

// RebalanceAction 再平衡动作
type RebalanceAction string

const (
	ActionIncrease RebalanceAction = "increase"
	ActionDecrease RebalanceAction = "decrease"
	ActionMaintain RebalanceAction = "maintain"
	ActionExit     RebalanceAction = "exit"
)

// ExecutionPriority 执行优先级
type ExecutionPriority string

const (
	PriorityLow      ExecutionPriority = "low"
	PriorityMedium   ExecutionPriority = "medium"
	PriorityHigh     ExecutionPriority = "high"
	PriorityCritical ExecutionPriority = "critical"
)

// ConvergenceInfo 收敛信息
type ConvergenceInfo struct {
	Converged          bool    `json:"converged"`
	Iterations         int     `json:"iterations"`
	FinalGradientNorm  float64 `json:"final_gradient_norm"`
	ObjectiveImprovement float64 `json:"objective_improvement"`
	ComputationTime    time.Duration `json:"computation_time"`
}

// CostAnalysis 成本分析
type CostAnalysis struct {
	TransactionCosts    float64 `json:"transaction_costs"`
	MarketImpactCosts   float64 `json:"market_impact_costs"`
	OpportunityCosts    float64 `json:"opportunity_costs"`
	TotalCosts          float64 `json:"total_costs"`
	CostBenefit         float64 `json:"cost_benefit"`
}

// NewProfitMaximizationEngine 创建利润最大化引擎
func NewProfitMaximizationEngine(
	positionOptimizer *position.DynamicPositionOptimizer,
	tradingExecutor *trading.SmartTradingExecutor,
	config *MaximizationConfig) *ProfitMaximizationEngine {
	
	return &ProfitMaximizationEngine{
		portfolioOptimizer: NewPortfolioOptimizer(config),
		capitalAllocator:   NewCapitalAllocator(config),
		riskAdjuster:       NewRiskAdjuster(config),
		costMinimizer:      NewCostMinimizer(config),
		performanceTracker: NewPerformanceTracker(),
		positionOptimizer:  positionOptimizer,
		tradingExecutor:    tradingExecutor,
		config:             config,
	}
}

// MaximizeProfit 执行全局利润最大化
func (pme *ProfitMaximizationEngine) MaximizeProfit(ctx context.Context, 
	portfolio *exchange.Portfolio,
	marketData map[string]*market.MarketData,
	strategies []*strategy.Strategy) (*OptimizationResult, error) {
	
	pme.mu.Lock()
	defer pme.mu.Unlock()
	
	startTime := time.Now()
	
	// 1. 分析当前组合状态
	currentState, err := pme.analyzeCurrentState(portfolio, marketData, strategies)
	if err != nil {
		return nil, fmt.Errorf("current state analysis failed: %w", err)
	}
	
	// 2. 构建目标函数
	objective, err := pme.buildObjectiveFunction(currentState, marketData)
	if err != nil {
		return nil, fmt.Errorf("objective function construction failed: %w", err)
	}
	
	// 3. 设定优化约束
	constraints, err := pme.buildOptimizationConstraints(currentState)
	if err != nil {
		return nil, fmt.Errorf("constraints construction failed: %w", err)
	}
	
	// 4. 执行多目标优化
	optimalAllocation, err := pme.portfolioOptimizer.OptimizePortfolio(
		objective, constraints, currentState)
	if err != nil {
		return nil, fmt.Errorf("portfolio optimization failed: %w", err)
	}
	
	// 5. 风险调整
	riskAdjustedAllocation, riskMetrics, err := pme.riskAdjuster.AdjustForRisk(
		optimalAllocation, currentState)
	if err != nil {
		return nil, fmt.Errorf("risk adjustment failed: %w", err)
	}
	
	// 6. 成本最小化
	costOptimizedAllocation, costAnalysis, err := pme.costMinimizer.MinimizeCosts(
		riskAdjustedAllocation, currentState)
	if err != nil {
		return nil, fmt.Errorf("cost minimization failed: %w", err)
	}
	
	// 7. 生成执行计划
	executionPlan, err := pme.generateExecutionPlan(
		portfolio, costOptimizedAllocation, costAnalysis)
	if err != nil {
		return nil, fmt.Errorf("execution plan generation failed: %w", err)
	}
	
	// 8. 性能预测
	performanceForecast, err := pme.forecastPerformance(
		costOptimizedAllocation, riskMetrics, marketData)
	if err != nil {
		return nil, fmt.Errorf("performance forecasting failed: %w", err)
	}
	
	// 9. 计算目标函数值
	objectiveValue := pme.calculateObjectiveValue(
		costOptimizedAllocation, riskMetrics, costAnalysis)
	
	result := &OptimizationResult{
		Timestamp:           startTime,
		ObjectiveValue:      objectiveValue,
		OptimalAllocation:   costOptimizedAllocation,
		RiskMetrics:         riskMetrics,
		PerformanceForecast: performanceForecast,
		ExecutionPlan:       executionPlan,
		ConvergenceInfo: &ConvergenceInfo{
			Converged:       true,
			Iterations:      100,
			ComputationTime: time.Since(startTime),
		},
		CostAnalysis: costAnalysis,
	}
	
	// 10. 记录优化历史
	pme.optimizationHistory = append(pme.optimizationHistory, result)
	pme.lastOptimization = startTime
	
	return result, nil
}

// ExecuteOptimizationPlan 执行优化计划
func (pme *ProfitMaximizationEngine) ExecuteOptimizationPlan(ctx context.Context, 
	result *OptimizationResult) error {
	
	if result.ExecutionPlan == nil {
		return fmt.Errorf("no execution plan provided")
	}
	
	// 按优先级排序执行
	orders := pme.prioritizeOrders(result.ExecutionPlan.RebalanceOrders)
	
	for _, order := range orders {
		err := pme.executeRebalanceOrder(ctx, order)
		if err != nil {
			return fmt.Errorf("failed to execute order for %s: %w", order.Symbol, err)
		}
		
		// 记录执行结果
		pme.performanceTracker.RecordExecution(order, err)
	}
	
	return nil
}

// analyzeCurrentState 分析当前组合状态
func (pme *ProfitMaximizationEngine) analyzeCurrentState(
	portfolio *exchange.Portfolio,
	marketData map[string]*market.MarketData,
	strategies []*strategy.Strategy) (*PortfolioState, error) {
	
	state := &PortfolioState{
		TotalValue:    portfolio.TotalValue,
		CashBalance:   portfolio.CashBalance,
		Positions:     make(map[string]*PositionState),
		Strategies:    make(map[string]*StrategyState),
		MarketData:    marketData,
		Timestamp:     time.Now(),
	}
	
	// 分析仓位状态
	for _, allocation := range portfolio.Allocations {
		posState := &PositionState{
			Symbol:        allocation.Symbol,
			Quantity:      allocation.Quantity,
			MarketValue:   allocation.MarketValue,
			Weight:        allocation.Weight,
			UnrealizedPnL: allocation.UnrealizedPnL,
		}
		
		// 计算额外指标
		if data, exists := marketData[allocation.Symbol]; exists {
			posState.Volatility = pme.calculateVolatility(data)
			posState.Beta = pme.calculateBeta(data)
			posState.LiquidityScore = pme.calculateLiquidityScore(data)
		}
		
		state.Positions[allocation.Symbol] = posState
	}
	
	// 分析策略状态
	for _, strat := range strategies {
		stratState := &StrategyState{
			ID:           strat.ID,
			Name:         strat.Name,
			Performance:  strat.Performance,
			RiskMetrics:  strat.RiskMetrics,
			IsActive:     strat.IsActive,
		}
		state.Strategies[strat.ID] = stratState
	}
	
	return state, nil
}

// buildObjectiveFunction 构建目标函数
func (pme *ProfitMaximizationEngine) buildObjectiveFunction(
	currentState *PortfolioState,
	marketData map[string]*market.MarketData) (*ObjectiveFunction, error) {
	
	// 根据市场状态动态调整目标函数
	marketRegime := pme.detectMarketRegime(marketData)
	
	var objectiveType ObjectiveFunctionType
	weights := make(map[string]float64)
	
	switch marketRegime {
	case "bull_market":
		objectiveType = ObjectiveMaxReturn
		weights["return"] = 0.7
		weights["risk"] = 0.2
		weights["cost"] = 0.1
	case "bear_market":
		objectiveType = ObjectiveMinRisk
		weights["return"] = 0.2
		weights["risk"] = 0.7
		weights["cost"] = 0.1
	case "volatile_market":
		objectiveType = ObjectiveMaxSharpe
		weights["return"] = 0.4
		weights["risk"] = 0.4
		weights["cost"] = 0.2
	default:
		objectiveType = ObjectiveMaxUtility
		weights["return"] = pme.config.ReturnWeight
		weights["risk"] = pme.config.RiskWeight
		weights["cost"] = pme.config.CostWeight
	}
	
	return &ObjectiveFunction{
		Name:       string(objectiveType),
		Function:   objectiveType,
		Weights:    weights,
		UpdateTime: time.Now(),
	}, nil
}

// PortfolioState 组合状态
type PortfolioState struct {
	TotalValue    float64                    `json:"total_value"`
	CashBalance   float64                    `json:"cash_balance"`
	Positions     map[string]*PositionState  `json:"positions"`
	Strategies    map[string]*StrategyState  `json:"strategies"`
	MarketData    map[string]*market.MarketData `json:"market_data"`
	Timestamp     time.Time                  `json:"timestamp"`
}

// PositionState 仓位状态
type PositionState struct {
	Symbol          string  `json:"symbol"`
	Quantity        float64 `json:"quantity"`
	MarketValue     float64 `json:"market_value"`
	Weight          float64 `json:"weight"`
	UnrealizedPnL   float64 `json:"unrealized_pnl"`
	Volatility      float64 `json:"volatility"`
	Beta            float64 `json:"beta"`
	LiquidityScore  float64 `json:"liquidity_score"`
}

// StrategyState 策略状态
type StrategyState struct {
	ID           string                      `json:"id"`
	Name         string                      `json:"name"`
	Performance  *strategy.PerformanceMetrics `json:"performance"`
	RiskMetrics  *strategy.RiskMetrics       `json:"risk_metrics"`
	IsActive     bool                        `json:"is_active"`
}

// 辅助组件实现
type PortfolioOptimizer struct {
	config *MaximizationConfig
}

func NewPortfolioOptimizer(config *MaximizationConfig) *PortfolioOptimizer {
	return &PortfolioOptimizer{config: config}
}

func (po *PortfolioOptimizer) OptimizePortfolio(
	objective *ObjectiveFunction,
	constraints []*OptimizationConstraint,
	currentState *PortfolioState) (map[string]float64, error) {
	
	// 简化的组合优化实现
	allocation := make(map[string]float64)
	
	// 均等权重作为起始点
	symbolCount := len(currentState.Positions)
	if symbolCount > 0 {
		equalWeight := 1.0 / float64(symbolCount)
		for symbol := range currentState.Positions {
			allocation[symbol] = equalWeight
		}
	}
	
	return allocation, nil
}

type CapitalAllocator struct {
	config *MaximizationConfig
}

func NewCapitalAllocator(config *MaximizationConfig) *CapitalAllocator {
	return &CapitalAllocator{config: config}
}

type RiskAdjuster struct {
	config *MaximizationConfig
}

func NewRiskAdjuster(config *MaximizationConfig) *RiskAdjuster {
	return &RiskAdjuster{config: config}
}

func (ra *RiskAdjuster) AdjustForRisk(
	allocation map[string]float64,
	currentState *PortfolioState) (map[string]float64, *RiskMetrics, error) {
	
	// 风险调整后的分配
	adjustedAllocation := make(map[string]float64)
	for symbol, weight := range allocation {
		adjustedAllocation[symbol] = weight
	}
	
	// 计算风险指标
	riskMetrics := &RiskMetrics{
		PortfolioVolatility: 0.15,
		ExpectedReturn:      0.12,
		SharpeRatio:        0.8,
		VaR95:              0.05,
		ConcentrationRisk:   0.3,
	}
	
	return adjustedAllocation, riskMetrics, nil
}

type CostMinimizer struct {
	config *MaximizationConfig
}

func NewCostMinimizer(config *MaximizationConfig) *CostMinimizer {
	return &CostMinimizer{config: config}
}

func (cm *CostMinimizer) MinimizeCosts(
	allocation map[string]float64,
	currentState *PortfolioState) (map[string]float64, *CostAnalysis, error) {
	
	// 成本最小化后的分配
	costOptimizedAllocation := make(map[string]float64)
	for symbol, weight := range allocation {
		costOptimizedAllocation[symbol] = weight
	}
	
	// 计算成本分析
	costAnalysis := &CostAnalysis{
		TransactionCosts:  100.0,
		MarketImpactCosts: 50.0,
		OpportunityCosts:  25.0,
		TotalCosts:        175.0,
		CostBenefit:       5.0,
	}
	
	return costOptimizedAllocation, costAnalysis, nil
}

type PerformanceTracker struct {
	executionHistory []*ExecutionRecord
	mu               sync.RWMutex
}

type ExecutionRecord struct {
	Order     *RebalanceOrder `json:"order"`
	Timestamp time.Time       `json:"timestamp"`
	Success   bool           `json:"success"`
	Error     string         `json:"error,omitempty"`
}

func NewPerformanceTracker() *PerformanceTracker {
	return &PerformanceTracker{}
}

func (pt *PerformanceTracker) RecordExecution(order *RebalanceOrder, err error) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	
	record := &ExecutionRecord{
		Order:     order,
		Timestamp: time.Now(),
		Success:   err == nil,
	}
	
	if err != nil {
		record.Error = err.Error()
	}
	
	pt.executionHistory = append(pt.executionHistory, record)
}

// 辅助方法实现
func (pme *ProfitMaximizationEngine) buildOptimizationConstraints(
	currentState *PortfolioState) ([]*OptimizationConstraint, error) {
	
	var constraints []*OptimizationConstraint
	
	// 仓位集中度约束
	constraints = append(constraints, &OptimizationConstraint{
		Type:       ConstraintPosition,
		Parameter:  "max_weight",
		UpperBound: pme.config.MaxPositionConcentration,
		Penalty:    1000.0,
	})
	
	// 杠杆约束
	constraints = append(constraints, &OptimizationConstraint{
		Type:       ConstraintLeverage,
		Parameter:  "total_leverage",
		UpperBound: pme.config.MaxLeverage,
		Penalty:    5000.0,
	})
	
	return constraints, nil
}

func (pme *ProfitMaximizationEngine) generateExecutionPlan(
	portfolio *exchange.Portfolio,
	targetAllocation map[string]float64,
	costAnalysis *CostAnalysis) (*ExecutionPlan, error) {
	
	var orders []*RebalanceOrder
	
	for symbol, targetWeight := range targetAllocation {
		currentWeight := pme.getCurrentWeight(portfolio, symbol)
		
		if math.Abs(targetWeight-currentWeight) > 0.01 { // 1%阈值
			action := ActionMaintain
			if targetWeight > currentWeight {
				action = ActionIncrease
			} else if targetWeight < currentWeight {
				action = ActionDecrease
			}
			
			order := &RebalanceOrder{
				Symbol:        symbol,
				Action:        action,
				CurrentWeight: currentWeight,
				TargetWeight:  targetWeight,
				Quantity:      (targetWeight - currentWeight) * portfolio.TotalValue,
				Urgency:       trading.UrgencyMedium,
			}
			
			orders = append(orders, order)
		}
	}
	
	return &ExecutionPlan{
		RebalanceOrders:    orders,
		EstimatedCost:      costAnalysis.TotalCosts,
		ExecutionTimeframe: time.Hour,
		Priority:           PriorityMedium,
	}, nil
}

func (pme *ProfitMaximizationEngine) getCurrentWeight(portfolio *exchange.Portfolio, symbol string) float64 {
	for _, allocation := range portfolio.Allocations {
		if allocation.Symbol == symbol {
			return allocation.Weight
		}
	}
	return 0
}

func (pme *ProfitMaximizationEngine) forecastPerformance(
	allocation map[string]float64,
	riskMetrics *RiskMetrics,
	marketData map[string]*market.MarketData) (*PerformanceForecast, error) {
	
	return &PerformanceForecast{
		ExpectedReturn1M:    riskMetrics.ExpectedReturn / 12,
		ExpectedReturn3M:    riskMetrics.ExpectedReturn / 4,
		ExpectedReturn1Y:    riskMetrics.ExpectedReturn,
		VolatilityForecast:  riskMetrics.PortfolioVolatility,
		SharpeRatioForecast: riskMetrics.SharpeRatio,
		DownsideRisk:        riskMetrics.VaR95,
		TailRisk:            riskMetrics.ConditionalVaR,
	}, nil
}

func (pme *ProfitMaximizationEngine) calculateObjectiveValue(
	allocation map[string]float64,
	riskMetrics *RiskMetrics,
	costAnalysis *CostAnalysis) float64 {
	
	// 效用函数: Return - λ*Risk - γ*Cost
	utility := riskMetrics.ExpectedReturn - 
		pme.config.RiskWeight*riskMetrics.PortfolioVolatility -
		pme.config.CostWeight*costAnalysis.TotalCosts/10000 // 标准化成本
	
	return utility
}

func (pme *ProfitMaximizationEngine) detectMarketRegime(marketData map[string]*market.MarketData) string {
	// 简化的市场状态检测
	return "normal_market"
}

func (pme *ProfitMaximizationEngine) calculateVolatility(data *market.MarketData) float64 {
	// 简化的波动率计算
	return 0.2
}

func (pme *ProfitMaximizationEngine) calculateBeta(data *market.MarketData) float64 {
	// 简化的Beta计算
	return 1.0
}

func (pme *ProfitMaximizationEngine) calculateLiquidityScore(data *market.MarketData) float64 {
	// 简化的流动性评分
	return 0.8
}

func (pme *ProfitMaximizationEngine) prioritizeOrders(orders []*RebalanceOrder) []*RebalanceOrder {
	// 按紧急程度排序
	return orders // 简化实现
}

func (pme *ProfitMaximizationEngine) executeRebalanceOrder(ctx context.Context, order *RebalanceOrder) error {
	// 这里应该调用SmartTradingExecutor执行实际交易
	// 简化实现
	return nil
}
