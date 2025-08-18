package position

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"qcat/internal/config"
	"qcat/internal/exchange"
	"qcat/internal/market"
	"qcat/internal/monitor"
)

// DynamicPositionOptimizer 智能仓位动态优化器
// 集成Kelly公式、Black-Litterman模型和风险预算管理
type DynamicPositionOptimizer struct {
	kellyCalculator       *KellyCalculator
	blackLitterman        *BlackLittermanModel
	riskBudgetManager     *RiskBudgetManager
	marketRegimeDetector  *MarketRegimeDetector
	volatilityPredictor   *VolatilityPredictor
	
	config                *OptimizerConfig
	mu                    sync.RWMutex
	
	// 缓存数据
	positionCache         map[string]*OptimalPosition
	lastOptimization      time.Time
	optimizationInterval  time.Duration
}

// OptimizerConfig 优化器配置
type OptimizerConfig struct {
	MaxPosition           float64   `yaml:"max_position"`           // 最大仓位比例
	MinPosition           float64   `yaml:"min_position"`           // 最小仓位比例
	TargetVolatility      float64   `yaml:"target_volatility"`      // 目标波动率
	RiskBudgetLimit       float64   `yaml:"risk_budget_limit"`      // 风险预算限制
	OptimizationInterval  string    `yaml:"optimization_interval"`  // 优化间隔
	KellyMultiplier       float64   `yaml:"kelly_multiplier"`       // Kelly系数调节器
	VolatilityLookback    int       `yaml:"volatility_lookback"`    // 波动率回看期
	ConfidenceLevel       float64   `yaml:"confidence_level"`       // 置信水平
}

// OptimalPosition 最优仓位结果
type OptimalPosition struct {
	Symbol              string    `json:"symbol"`
	TargetWeight        float64   `json:"target_weight"`
	CurrentWeight       float64   `json:"current_weight"`
	OptimalSize         float64   `json:"optimal_size"`
	RiskContribution    float64   `json:"risk_contribution"`
	ExpectedReturn      float64   `json:"expected_return"`
	KellyFraction       float64   `json:"kelly_fraction"`
	VolatilityForecast  float64   `json:"volatility_forecast"`
	MarketRegime        string    `json:"market_regime"`
	RebalanceSignal     string    `json:"rebalance_signal"`
	OptimizationTime    time.Time `json:"optimization_time"`
}

// MarketRegime 市场状态
type MarketRegime string

const (
	RegimeTrending    MarketRegime = "trending"
	RegimeRanging     MarketRegime = "ranging"
	RegimeVolatile    MarketRegime = "volatile"
	RegimeCalm        MarketRegime = "calm"
	RegimeBullish     MarketRegime = "bullish"
	RegimeBearish     MarketRegime = "bearish"
	RegimeUncertain   MarketRegime = "uncertain"
	RegimeCrisis      MarketRegime = "crisis"
)

// NewDynamicPositionOptimizer 创建动态仓位优化器
func NewDynamicPositionOptimizer(config *OptimizerConfig) *DynamicPositionOptimizer {
	interval, _ := time.ParseDuration(config.OptimizationInterval)
	if interval == 0 {
		interval = 15 * time.Minute // 默认15分钟
	}
	
	return &DynamicPositionOptimizer{
		kellyCalculator:      NewKellyCalculator(config.KellyMultiplier),
		blackLitterman:       NewBlackLittermanModel(),
		riskBudgetManager:    NewRiskBudgetManager(config.RiskBudgetLimit),
		marketRegimeDetector: NewMarketRegimeDetector(),
		volatilityPredictor:  NewVolatilityPredictor(config.VolatilityLookback),
		config:               config,
		positionCache:        make(map[string]*OptimalPosition),
		optimizationInterval: interval,
	}
}

// OptimizePortfolio 优化整个投资组合的仓位
func (dpo *DynamicPositionOptimizer) OptimizePortfolio(ctx context.Context, 
	portfolio *exchange.Portfolio, marketData map[string]*market.MarketData) (*PortfolioOptimization, error) {
	
	dpo.mu.Lock()
	defer dpo.mu.Unlock()
	
	// 检查是否需要重新优化
	if time.Since(dpo.lastOptimization) < dpo.optimizationInterval {
		return dpo.getCachedOptimization(), nil
	}
	
	// 1. 市场状态识别
	regimes := dpo.detectMarketRegimes(marketData)
	
	// 2. 波动率预测
	volatilityForecasts := dpo.predictVolatilities(marketData)
	
	// 3. 期望收益估计 (Black-Litterman)
	expectedReturns := dpo.estimateExpectedReturns(marketData, regimes)
	
	// 4. Kelly最优仓位计算
	kellyPositions := dpo.calculateKellyPositions(expectedReturns, volatilityForecasts)
	
	// 5. 风险预算约束
	riskAdjustedPositions := dpo.applyRiskBudgetConstraints(kellyPositions, volatilityForecasts)
	
	// 6. 市场状态调整
	finalPositions := dpo.adjustForMarketRegime(riskAdjustedPositions, regimes)
	
	// 7. 生成重平衡信号
	rebalanceSignals := dpo.generateRebalanceSignals(portfolio, finalPositions)
	
	result := &PortfolioOptimization{
		OptimalPositions:    finalPositions,
		RebalanceSignals:    rebalanceSignals,
		MarketRegimes:       regimes,
		VolatilityForecasts: volatilityForecasts,
		ExpectedReturns:     expectedReturns,
		OptimizationTime:    time.Now(),
		RiskMetrics:         dpo.calculateRiskMetrics(finalPositions, volatilityForecasts),
	}
	
	// 更新缓存
	dpo.updatePositionCache(finalPositions)
	dpo.lastOptimization = time.Now()
	
	return result, nil
}

// detectMarketRegimes 检测市场状态
func (dpo *DynamicPositionOptimizer) detectMarketRegimes(marketData map[string]*market.MarketData) map[string]MarketRegime {
	regimes := make(map[string]MarketRegime)
	
	for symbol, data := range marketData {
		regime := dpo.marketRegimeDetector.DetectRegime(data)
		regimes[symbol] = regime
	}
	
	return regimes
}

// predictVolatilities 预测波动率
func (dpo *DynamicPositionOptimizer) predictVolatilities(marketData map[string]*market.MarketData) map[string]float64 {
	forecasts := make(map[string]float64)
	
	for symbol, data := range marketData {
		volatility := dpo.volatilityPredictor.PredictVolatility(data)
		forecasts[symbol] = volatility
	}
	
	return forecasts
}

// estimateExpectedReturns 使用Black-Litterman模型估计期望收益
func (dpo *DynamicPositionOptimizer) estimateExpectedReturns(
	marketData map[string]*market.MarketData, 
	regimes map[string]MarketRegime) map[string]float64 {
	
	// 市场均衡收益
	equilibriumReturns := dpo.blackLitterman.CalculateEquilibriumReturns(marketData)
	
	// 基于市场状态的观点调整
	views := dpo.generateMarketViews(regimes, marketData)
	
	// Black-Litterman调整
	adjustedReturns := dpo.blackLitterman.AdjustReturns(equilibriumReturns, views)
	
	return adjustedReturns
}

// calculateKellyPositions 计算Kelly最优仓位
func (dpo *DynamicPositionOptimizer) calculateKellyPositions(
	expectedReturns map[string]float64, 
	volatilities map[string]float64) map[string]float64 {
	
	positions := make(map[string]float64)
	
	for symbol, expectedReturn := range expectedReturns {
		volatility, exists := volatilities[symbol]
		if !exists {
			continue
		}
		
		kellyFraction := dpo.kellyCalculator.CalculateOptimalFraction(expectedReturn, volatility)
		positions[symbol] = kellyFraction
	}
	
	return positions
}

// applyRiskBudgetConstraints 应用风险预算约束
func (dpo *DynamicPositionOptimizer) applyRiskBudgetConstraints(
	positions map[string]float64, 
	volatilities map[string]float64) map[string]float64 {
	
	// 计算风险贡献
	riskContributions := dpo.riskBudgetManager.CalculateRiskContributions(positions, volatilities)
	
	// 应用风险预算约束
	adjustedPositions := dpo.riskBudgetManager.ApplyRiskBudget(positions, riskContributions)
	
	// 应用仓位限制
	for symbol, position := range adjustedPositions {
		if position > dpo.config.MaxPosition {
			adjustedPositions[symbol] = dpo.config.MaxPosition
		} else if position < dpo.config.MinPosition && position > 0 {
			adjustedPositions[symbol] = dpo.config.MinPosition
		} else if position < -dpo.config.MaxPosition {
			adjustedPositions[symbol] = -dpo.config.MaxPosition
		} else if position > -dpo.config.MinPosition && position < 0 {
			adjustedPositions[symbol] = -dpo.config.MinPosition
		}
	}
	
	return adjustedPositions
}

// adjustForMarketRegime 根据市场状态调整仓位
func (dpo *DynamicPositionOptimizer) adjustForMarketRegime(
	positions map[string]float64, 
	regimes map[string]MarketRegime) map[string]*OptimalPosition {
	
	optimalPositions := make(map[string]*OptimalPosition)
	
	for symbol, position := range positions {
		regime := regimes[symbol]
		adjustedPosition := dpo.applyRegimeAdjustment(position, regime)
		
		optimalPositions[symbol] = &OptimalPosition{
			Symbol:         symbol,
			TargetWeight:   adjustedPosition,
			OptimalSize:    adjustedPosition,
			MarketRegime:   string(regime),
			OptimizationTime: time.Now(),
		}
	}
	
	return optimalPositions
}

// applyRegimeAdjustment 应用市场状态调整
func (dpo *DynamicPositionOptimizer) applyRegimeAdjustment(position float64, regime MarketRegime) float64 {
	switch regime {
	case RegimeCrisis:
		return position * 0.3 // 危机时期大幅减仓
	case RegimeVolatile:
		return position * 0.7 // 高波动时期适度减仓
	case RegimeUncertain:
		return position * 0.8 // 不确定时期小幅减仓
	case RegimeTrending:
		return position * 1.2 // 趋势明确时适度加仓
	case RegimeBullish:
		return position * 1.1 // 牛市时小幅加仓
	default:
		return position
	}
}

// generateRebalanceSignals 生成重平衡信号
func (dpo *DynamicPositionOptimizer) generateRebalanceSignals(
	portfolio *exchange.Portfolio, 
	optimalPositions map[string]*OptimalPosition) map[string]string {
	
	signals := make(map[string]string)
	
	for symbol, optimal := range optimalPositions {
		current := dpo.getCurrentWeight(portfolio, symbol)
		optimal.CurrentWeight = current
		
		deviation := math.Abs(optimal.TargetWeight - current)
		
		if deviation > 0.05 { // 5%阈值
			if optimal.TargetWeight > current {
				signals[symbol] = "INCREASE"
			} else {
				signals[symbol] = "DECREASE"
			}
		} else {
			signals[symbol] = "HOLD"
		}
		
		optimal.RebalanceSignal = signals[symbol]
	}
	
	return signals
}

// KellyCalculator Kelly公式计算器
type KellyCalculator struct {
	multiplier float64
}

func NewKellyCalculator(multiplier float64) *KellyCalculator {
	if multiplier <= 0 {
		multiplier = 0.25 // 保守的Kelly系数
	}
	return &KellyCalculator{multiplier: multiplier}
}

func (kc *KellyCalculator) CalculateOptimalFraction(expectedReturn, volatility float64) float64 {
	if volatility <= 0 {
		return 0
	}
	
	// Kelly公式: f* = (μ - r) / σ²
	// 这里r假设为0（无风险利率）
	kellyFraction := expectedReturn / (volatility * volatility)
	
	// 应用保守系数
	return kellyFraction * kc.multiplier
}

// BlackLittermanModel Black-Litterman模型
type BlackLittermanModel struct {
	tau        float64 // 不确定性参数
	riskAversion float64 // 风险厌恶参数
}

func NewBlackLittermanModel() *BlackLittermanModel {
	return &BlackLittermanModel{
		tau:          0.05,
		riskAversion: 3.0,
	}
}

func (bl *BlackLittermanModel) CalculateEquilibriumReturns(marketData map[string]*market.MarketData) map[string]float64 {
	// 简化实现：基于历史收益率计算均衡收益
	returns := make(map[string]float64)
	
	for symbol, data := range marketData {
		if len(data.Klines) > 0 {
			// 计算年化收益率
			annualizedReturn := bl.calculateAnnualizedReturn(data.Klines)
			returns[symbol] = annualizedReturn
		}
	}
	
	return returns
}

func (bl *BlackLittermanModel) calculateAnnualizedReturn(klines []*market.Kline) float64 {
	if len(klines) < 2 {
		return 0
	}
	
	// 简单的对数收益率计算
	start := klines[0].Close
	end := klines[len(klines)-1].Close
	
	if start <= 0 {
		return 0
	}
	
	return math.Log(end/start) * 365 / float64(len(klines))
}

func (bl *BlackLittermanModel) AdjustReturns(equilibriumReturns map[string]float64, views map[string]float64) map[string]float64 {
	// 简化的Black-Litterman调整
	adjustedReturns := make(map[string]float64)
	
	for symbol, equilibrium := range equilibriumReturns {
		view, hasView := views[symbol]
		if hasView {
			// 加权平均调整
			adjusted := (equilibrium + bl.tau*view) / (1 + bl.tau)
			adjustedReturns[symbol] = adjusted
		} else {
			adjustedReturns[symbol] = equilibrium
		}
	}
	
	return adjustedReturns
}

// generateMarketViews 生成市场观点
func (dpo *DynamicPositionOptimizer) generateMarketViews(
	regimes map[string]MarketRegime, 
	marketData map[string]*market.MarketData) map[string]float64 {
	
	views := make(map[string]float64)
	
	for symbol, regime := range regimes {
		switch regime {
		case RegimeBullish:
			views[symbol] = 0.15 // 预期15%年化收益
		case RegimeBearish:
			views[symbol] = -0.10 // 预期-10%年化收益
		case RegimeTrending:
			views[symbol] = 0.08 // 预期8%年化收益
		case RegimeCrisis:
			views[symbol] = -0.20 // 预期-20%年化收益
		default:
			views[symbol] = 0.02 // 默认2%年化收益
		}
	}
	
	return views
}

// PortfolioOptimization 投资组合优化结果
type PortfolioOptimization struct {
	OptimalPositions    map[string]*OptimalPosition `json:"optimal_positions"`
	RebalanceSignals    map[string]string           `json:"rebalance_signals"`
	MarketRegimes       map[string]MarketRegime     `json:"market_regimes"`
	VolatilityForecasts map[string]float64          `json:"volatility_forecasts"`
	ExpectedReturns     map[string]float64          `json:"expected_returns"`
	OptimizationTime    time.Time                   `json:"optimization_time"`
	RiskMetrics         *RiskMetrics                `json:"risk_metrics"`
}

// RiskMetrics 风险指标
type RiskMetrics struct {
	PortfolioVolatility float64 `json:"portfolio_volatility"`
	VaR95               float64 `json:"var_95"`
	MaxDrawdown         float64 `json:"max_drawdown"`
	SharpeRatio         float64 `json:"sharpe_ratio"`
	ConcentrationRisk   float64 `json:"concentration_risk"`
}

// 辅助方法
func (dpo *DynamicPositionOptimizer) getCurrentWeight(portfolio *exchange.Portfolio, symbol string) float64 {
	// 从投资组合获取当前权重
	for _, allocation := range portfolio.Allocations {
		if allocation.Symbol == symbol {
			return allocation.Weight
		}
	}
	return 0
}

func (dpo *DynamicPositionOptimizer) updatePositionCache(positions map[string]*OptimalPosition) {
	for symbol, position := range positions {
		dpo.positionCache[symbol] = position
	}
}

func (dpo *DynamicPositionOptimizer) getCachedOptimization() *PortfolioOptimization {
	return &PortfolioOptimization{
		OptimalPositions: dpo.positionCache,
		OptimizationTime: dpo.lastOptimization,
	}
}

func (dpo *DynamicPositionOptimizer) calculateRiskMetrics(
	positions map[string]*OptimalPosition, 
	volatilities map[string]float64) *RiskMetrics {
	
	// 简化的风险指标计算
	var portfolioVol, concentration float64
	totalWeight := 0.0
	
	for symbol, position := range positions {
		weight := math.Abs(position.TargetWeight)
		totalWeight += weight
		
		if vol, exists := volatilities[symbol]; exists {
			portfolioVol += weight * weight * vol * vol
		}
		
		// 集中度风险 (赫芬达尔指数)
		concentration += weight * weight
	}
	
	portfolioVol = math.Sqrt(portfolioVol)
	
	return &RiskMetrics{
		PortfolioVolatility: portfolioVol,
		ConcentrationRisk:   concentration,
		// VaR和其他指标需要更复杂的计算
	}
}
