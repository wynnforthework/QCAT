package position

import (
	"math"
	"qcat/internal/market"
)

// MarketRegimeDetector 市场状态检测器
type MarketRegimeDetector struct {
	lookbackPeriod    int
	volatilityWindow  int
	trendStrength     float64
	volatilityThreshold float64
}

// NewMarketRegimeDetector 创建市场状态检测器
func NewMarketRegimeDetector() *MarketRegimeDetector {
	return &MarketRegimeDetector{
		lookbackPeriod:      50,
		volatilityWindow:    20,
		trendStrength:       0.1,
		volatilityThreshold: 0.02,
	}
}

// DetectRegime 检测市场状态
func (mrd *MarketRegimeDetector) DetectRegime(data *market.MarketData) MarketRegime {
	if len(data.Klines) < mrd.lookbackPeriod {
		return RegimeUncertain
	}
	
	// 1. 计算趋势强度
	trendStrength := mrd.calculateTrendStrength(data.Klines)
	
	// 2. 计算波动率水平
	volatility := mrd.calculateVolatility(data.Klines)
	
	// 3. 计算价格动量
	momentum := mrd.calculateMomentum(data.Klines)
	
	// 4. 计算市场压力指标
	stress := mrd.calculateMarketStress(data.Klines)
	
	// 5. 综合判断市场状态
	return mrd.classifyRegime(trendStrength, volatility, momentum, stress)
}

// calculateTrendStrength 计算趋势强度
func (mrd *MarketRegimeDetector) calculateTrendStrength(klines []*market.Kline) float64 {
	if len(klines) < 20 {
		return 0
	}
	
	// 使用线性回归计算趋势强度
	n := len(klines)
	lastN := klines[n-20:]
	
	var sumX, sumY, sumXY, sumX2 float64
	
	for i, kline := range lastN {
		x := float64(i)
		y := math.Log(kline.Close)
		
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	
	// 计算回归系数
	slope := (20*sumXY - sumX*sumY) / (20*sumX2 - sumX*sumX)
	
	// 计算相关系数R²
	avgX := sumX / 20
	avgY := sumY / 20
	
	var ssRes, ssTot float64
	for i, kline := range lastN {
		x := float64(i)
		y := math.Log(kline.Close)
		
		predicted := slope*x + (avgY - slope*avgX)
		ssRes += (y - predicted) * (y - predicted)
		ssTot += (y - avgY) * (y - avgY)
	}
	
	r2 := 1 - ssRes/ssTot
	
	// 趋势强度 = 斜率的绝对值 * R²
	return math.Abs(slope) * r2
}

// calculateVolatility 计算波动率
func (mrd *MarketRegimeDetector) calculateVolatility(klines []*market.Kline) float64 {
	if len(klines) < mrd.volatilityWindow {
		return 0
	}
	
	n := len(klines)
	recent := klines[n-mrd.volatilityWindow:]
	
	var returns []float64
	for i := 1; i < len(recent); i++ {
		ret := math.Log(recent[i].Close / recent[i-1].Close)
		returns = append(returns, ret)
	}
	
	// 计算标准差
	return mrd.standardDeviation(returns) * math.Sqrt(365) // 年化波动率
}

// calculateMomentum 计算价格动量
func (mrd *MarketRegimeDetector) calculateMomentum(klines []*market.Kline) float64 {
	if len(klines) < 10 {
		return 0
	}
	
	n := len(klines)
	
	// 短期动量 (5日)
	shortMomentum := (klines[n-1].Close - klines[n-5].Close) / klines[n-5].Close
	
	// 中期动量 (10日)
	mediumMomentum := (klines[n-1].Close - klines[n-10].Close) / klines[n-10].Close
	
	// 加权平均
	return 0.6*shortMomentum + 0.4*mediumMomentum
}

// calculateMarketStress 计算市场压力指标
func (mrd *MarketRegimeDetector) calculateMarketStress(klines []*market.Kline) float64 {
	if len(klines) < 20 {
		return 0
	}
	
	n := len(klines)
	recent := klines[n-20:]
	
	// 1. 计算下跌日比例
	downDays := 0
	for i := 1; i < len(recent); i++ {
		if recent[i].Close < recent[i-1].Close {
			downDays++
		}
	}
	downRatio := float64(downDays) / float64(len(recent)-1)
	
	// 2. 计算最大单日跌幅
	maxDrop := 0.0
	for i := 1; i < len(recent); i++ {
		drop := (recent[i-1].Close - recent[i].Close) / recent[i-1].Close
		if drop > maxDrop {
			maxDrop = drop
		}
	}
	
	// 3. 计算连续下跌天数
	consecutiveDown := 0
	maxConsecutiveDown := 0
	for i := 1; i < len(recent); i++ {
		if recent[i].Close < recent[i-1].Close {
			consecutiveDown++
			if consecutiveDown > maxConsecutiveDown {
				maxConsecutiveDown = consecutiveDown
			}
		} else {
			consecutiveDown = 0
		}
	}
	
	// 综合压力指标
	stress := downRatio*0.4 + maxDrop*0.4 + float64(maxConsecutiveDown)/10*0.2
	return math.Min(stress, 1.0)
}

// classifyRegime 分类市场状态
func (mrd *MarketRegimeDetector) classifyRegime(trendStrength, volatility, momentum, stress float64) MarketRegime {
	// 危机状态检测
	if stress > 0.7 || volatility > 0.5 {
		return RegimeCrisis
	}
	
	// 高波动状态
	if volatility > 0.3 {
		return RegimeVolatile
	}
	
	// 趋势状态检测
	if trendStrength > 0.15 {
		if momentum > 0.05 {
			return RegimeBullish
		} else if momentum < -0.05 {
			return RegimeBearish
		} else {
			return RegimeTrending
		}
	}
	
	// 区间震荡状态
	if volatility < 0.1 && math.Abs(momentum) < 0.02 {
		return RegimeRanging
	}
	
	// 平静状态
	if volatility < 0.05 && stress < 0.2 {
		return RegimeCalm
	}
	
	// 默认不确定状态
	return RegimeUncertain
}

// standardDeviation 计算标准差
func (mrd *MarketRegimeDetector) standardDeviation(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	// 计算均值
	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))
	
	// 计算方差
	var variance float64
	for _, v := range values {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(values) - 1)
	
	return math.Sqrt(variance)
}

// VolatilityPredictor 波动率预测器
type VolatilityPredictor struct {
	lookbackPeriod int
	ewmaAlpha      float64
	garchParams    *GARCHParams
}

// GARCHParams GARCH模型参数
type GARCHParams struct {
	Omega float64 // 常数项
	Alpha float64 // ARCH项系数
	Beta  float64 // GARCH项系数
}

// NewVolatilityPredictor 创建波动率预测器
func NewVolatilityPredictor(lookbackPeriod int) *VolatilityPredictor {
	return &VolatilityPredictor{
		lookbackPeriod: lookbackPeriod,
		ewmaAlpha:      0.06, // RiskMetrics推荐值
		garchParams: &GARCHParams{
			Omega: 0.000001,
			Alpha: 0.08,
			Beta:  0.9,
		},
	}
}

// PredictVolatility 预测波动率
func (vp *VolatilityPredictor) PredictVolatility(data *market.MarketData) float64 {
	if len(data.Klines) < vp.lookbackPeriod {
		return vp.calculateSimpleVolatility(data.Klines)
	}
	
	// 使用GARCH模型预测
	returns := vp.calculateReturns(data.Klines)
	
	// EWMA波动率作为初始值
	ewmaVol := vp.calculateEWMAVolatility(returns)
	
	// GARCH预测
	garchVol := vp.predictGARCHVolatility(returns)
	
	// 组合预测（EWMA权重70%，GARCH权重30%）
	combinedVol := 0.7*ewmaVol + 0.3*garchVol
	
	return combinedVol
}

// calculateReturns 计算收益率序列
func (vp *VolatilityPredictor) calculateReturns(klines []*market.Kline) []float64 {
	var returns []float64
	
	for i := 1; i < len(klines); i++ {
		ret := math.Log(klines[i].Close / klines[i-1].Close)
		returns = append(returns, ret)
	}
	
	return returns
}

// calculateSimpleVolatility 计算简单波动率
func (vp *VolatilityPredictor) calculateSimpleVolatility(klines []*market.Kline) float64 {
	if len(klines) < 2 {
		return 0.2 // 默认20%波动率
	}
	
	returns := vp.calculateReturns(klines)
	
	// 计算标准差
	var sum, sumSq float64
	for _, ret := range returns {
		sum += ret
		sumSq += ret * ret
	}
	
	n := float64(len(returns))
	mean := sum / n
	variance := sumSq/n - mean*mean
	
	return math.Sqrt(variance) * math.Sqrt(365) // 年化
}

// calculateEWMAVolatility 计算EWMA波动率
func (vp *VolatilityPredictor) calculateEWMAVolatility(returns []float64) float64 {
	if len(returns) == 0 {
		return 0.2
	}
	
	// 初始化
	ewmaVar := returns[0] * returns[0]
	
	// 递推计算
	for i := 1; i < len(returns); i++ {
		ewmaVar = vp.ewmaAlpha*returns[i]*returns[i] + (1-vp.ewmaAlpha)*ewmaVar
	}
	
	return math.Sqrt(ewmaVar) * math.Sqrt(365) // 年化
}

// predictGARCHVolatility 使用GARCH模型预测波动率
func (vp *VolatilityPredictor) predictGARCHVolatility(returns []float64) float64 {
	if len(returns) < 10 {
		return vp.calculateSimpleVolatility(nil)
	}
	
	// 简化的GARCH(1,1)实现
	n := len(returns)
	
	// 计算无条件方差作为初始值
	var sumSq float64
	for _, ret := range returns {
		sumSq += ret * ret
	}
	unconditionalVar := sumSq / float64(n)
	
	// 初始化条件方差
	condVar := unconditionalVar
	
	// 递推计算最后几期的条件方差
	start := max(0, n-50) // 使用最近50个观测值
	for i := start; i < n; i++ {
		if i > 0 {
			// GARCH(1,1): σ²ₜ = ω + α·ε²ₜ₋₁ + β·σ²ₜ₋₁
			condVar = vp.garchParams.Omega + 
				vp.garchParams.Alpha*returns[i-1]*returns[i-1] + 
				vp.garchParams.Beta*condVar
		}
	}
	
	// 一步预测
	lastReturn := returns[n-1]
	predictedVar := vp.garchParams.Omega + 
		vp.garchParams.Alpha*lastReturn*lastReturn + 
		vp.garchParams.Beta*condVar
	
	return math.Sqrt(predictedVar) * math.Sqrt(365) // 年化
}

// RiskBudgetManager 风险预算管理器
type RiskBudgetManager struct {
	maxRiskBudget float64
	riskModel     *RiskModel
}

// RiskModel 风险模型
type RiskModel struct {
	correlationMatrix map[string]map[string]float64
	betaMap          map[string]float64
}

// NewRiskBudgetManager 创建风险预算管理器
func NewRiskBudgetManager(maxRiskBudget float64) *RiskBudgetManager {
	return &RiskBudgetManager{
		maxRiskBudget: maxRiskBudget,
		riskModel:     NewRiskModel(),
	}
}

// NewRiskModel 创建风险模型
func NewRiskModel() *RiskModel {
	return &RiskModel{
		correlationMatrix: make(map[string]map[string]float64),
		betaMap:          make(map[string]float64),
	}
}

// CalculateRiskContributions 计算风险贡献
func (rbm *RiskBudgetManager) CalculateRiskContributions(
	positions map[string]float64, 
	volatilities map[string]float64) map[string]float64 {
	
	contributions := make(map[string]float64)
	
	// 简化的风险贡献计算
	totalRisk := 0.0
	for symbol, position := range positions {
		if volatility, exists := volatilities[symbol]; exists {
			risk := math.Abs(position) * volatility
			contributions[symbol] = risk
			totalRisk += risk
		}
	}
	
	// 标准化为比例
	if totalRisk > 0 {
		for symbol := range contributions {
			contributions[symbol] /= totalRisk
		}
	}
	
	return contributions
}

// ApplyRiskBudget 应用风险预算约束
func (rbm *RiskBudgetManager) ApplyRiskBudget(
	positions map[string]float64, 
	riskContributions map[string]float64) map[string]float64 {
	
	adjustedPositions := make(map[string]float64)
	
	// 计算总风险贡献
	totalRiskContrib := 0.0
	for _, contrib := range riskContributions {
		totalRiskContrib += contrib
	}
	
	// 如果超过风险预算，按比例缩减
	if totalRiskContrib > rbm.maxRiskBudget {
		scaleFactor := rbm.maxRiskBudget / totalRiskContrib
		for symbol, position := range positions {
			adjustedPositions[symbol] = position * scaleFactor
		}
	} else {
		// 直接复制
		for symbol, position := range positions {
			adjustedPositions[symbol] = position
		}
	}
	
	return adjustedPositions
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
