package generator

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"qcat/internal/strategy"
	"qcat/internal/strategy/templates"
)

// Generator 自动策略生成器
type Generator struct {
	templates map[string]*templates.Template
	analyzer  *MarketAnalyzer
}

// NewGenerator 创建新的策略生成器
func NewGenerator() *Generator {
	return &Generator{
		templates: make(map[string]*templates.Template),
		analyzer:  NewMarketAnalyzer(),
	}
}

// GenerateStrategy 基于市场数据和历史表现自动生成策略
func (g *Generator) GenerateStrategy(ctx context.Context, req *GenerationRequest) (*strategy.Config, error) {
	log.Printf("Generating strategy for symbol: %s", req.Symbol)

	// 1. 分析市场数据
	analysis, err := g.analyzer.AnalyzeMarket(ctx, req.Symbol, req.TimeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze market: %w", err)
	}

	// 2. 选择最适合的策略模板
	template, err := g.selectBestTemplate(analysis)
	if err != nil {
		return nil, fmt.Errorf("failed to select template: %w", err)
	}

	// 3. 生成优化的参数
	params, err := g.generateOptimalParameters(analysis, template)
	if err != nil {
		return nil, fmt.Errorf("failed to generate parameters: %w", err)
	}

	// 4. 创建策略配置
	config := &strategy.Config{
		Name:        fmt.Sprintf("Auto_%s_%s_%d", template.Name, req.Symbol, time.Now().Unix()),
		Version:     "1.0.0",
		Description: fmt.Sprintf("Auto-generated %s strategy for %s", template.Name, req.Symbol),
		Mode:        strategy.ModeLive,
		Symbol:      req.Symbol,
		Exchange:    req.Exchange,
		Params:      params,
	}

	log.Printf("Generated strategy: %s with template: %s", config.Name, template.Name)
	return config, nil
}

// selectBestTemplate 根据市场分析选择最佳策略模板
func (g *Generator) selectBestTemplate(analysis *MarketAnalysis) (*templates.Template, error) {
	// 根据市场特征选择策略类型
	var templateName string

	switch {
	case analysis.Volatility > 0.05: // 高波动市场
		if analysis.Trend > 0.3 {
			templateName = "momentum_breakout"
		} else {
			templateName = "mean_reversion"
		}
	case analysis.Volatility < 0.02: // 低波动市场
		templateName = "grid_trading"
	case analysis.Trend > 0.5: // 强趋势市场
		templateName = "trend_following"
	case analysis.Trend < -0.5: // 强下跌趋势
		templateName = "short_trend"
	default: // 震荡市场
		templateName = "range_trading"
	}

	template, exists := g.templates[templateName]
	if !exists {
		// 如果模板不存在，使用默认模板
		template = templates.GetDefaultTemplate()
	}

	return template, nil
}

// generateOptimalParameters 生成优化的策略参数
func (g *Generator) generateOptimalParameters(analysis *MarketAnalysis, template *templates.Template) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	// 基于市场分析调整参数
	for paramName, paramConfig := range template.Parameters {
		switch paramName {
		case "stop_loss":
			// 根据波动率调整止损
			stopLoss := math.Min(0.05, analysis.Volatility*2)
			params[paramName] = stopLoss

		case "take_profit":
			// 根据波动率调整止盈
			takeProfit := math.Max(0.02, analysis.Volatility*1.5)
			params[paramName] = takeProfit

		case "position_size":
			// 根据波动率调整仓位大小
			positionSize := math.Max(0.1, 1.0/analysis.Volatility*0.01)
			positionSize = math.Min(positionSize, 0.5) // 最大50%
			params[paramName] = positionSize

		case "entry_threshold":
			// 根据趋势强度调整入场阈值
			threshold := math.Abs(analysis.Trend) * 0.5
			params[paramName] = threshold

		case "exit_threshold":
			// 根据趋势强度调整出场阈值
			threshold := math.Abs(analysis.Trend) * 0.3
			params[paramName] = threshold

		case "lookback_period":
			// 根据市场周期调整回看周期
			if analysis.MarketCycle > 0 {
				params[paramName] = int(analysis.MarketCycle * 0.5)
			} else {
				params[paramName] = paramConfig.Default
			}

		case "rsi_period":
			// RSI周期根据波动率调整
			period := 14
			if analysis.Volatility > 0.03 {
				period = 10 // 高波动用短周期
			} else if analysis.Volatility < 0.01 {
				period = 21 // 低波动用长周期
			}
			params[paramName] = period

		case "ma_period":
			// 移动平均周期根据趋势调整
			period := 20
			if math.Abs(analysis.Trend) > 0.5 {
				period = 10 // 强趋势用短周期
			} else if math.Abs(analysis.Trend) < 0.2 {
				period = 50 // 弱趋势用长周期
			}
			params[paramName] = period

		default:
			// 使用模板默认值
			params[paramName] = paramConfig.Default
		}
	}

	// 添加风险管理参数
	params["max_drawdown"] = 0.1    // 最大回撤10%
	params["max_daily_loss"] = 0.05 // 最大日损失5%
	params["leverage"] = g.calculateOptimalLeverage(analysis)

	return params, nil
}

// calculateOptimalLeverage 计算最优杠杆
func (g *Generator) calculateOptimalLeverage(analysis *MarketAnalysis) float64 {
	// 基于凯利公式和风险调整的杠杆计算
	baseleverage := 1.0

	// 根据波动率调整杠杆
	volatilityAdjustment := 1.0 / (1.0 + analysis.Volatility*10)

	// 根据夏普比率调整杠杆
	sharpeAdjustment := 1.0
	if analysis.SharpeRatio > 0 {
		sharpeAdjustment = math.Min(2.0, 1.0+analysis.SharpeRatio*0.5)
	}

	leverage := baseleverage * volatilityAdjustment * sharpeAdjustment

	// 限制杠杆范围
	leverage = math.Max(1.0, math.Min(leverage, 5.0))

	return leverage
}

// RegisterTemplate 注册策略模板
func (g *Generator) RegisterTemplate(name string, template *templates.Template) {
	g.templates[name] = template
}

// LoadDefaultTemplates 加载默认策略模板
func (g *Generator) LoadDefaultTemplates() {
	// 趋势跟踪策略
	g.RegisterTemplate("trend_following", templates.NewTrendFollowingTemplate())

	// 均值回归策略
	g.RegisterTemplate("mean_reversion", templates.NewMeanReversionTemplate())

	// 网格交易策略
	g.RegisterTemplate("grid_trading", templates.NewGridTradingTemplate())

	// 动量突破策略
	g.RegisterTemplate("momentum_breakout", templates.NewMomentumBreakoutTemplate())

	// 区间交易策略
	g.RegisterTemplate("range_trading", templates.NewRangeTradingTemplate())

	// 空头趋势策略
	g.RegisterTemplate("short_trend", templates.NewShortTrendTemplate())
}

// GenerationRequest 策略生成请求
type GenerationRequest struct {
	Symbol     string        `json:"symbol"`
	Exchange   string        `json:"exchange"`
	TimeRange  time.Duration `json:"time_range"`
	Objective  string        `json:"objective"`   // "profit", "sharpe", "drawdown"
	RiskLevel  string        `json:"risk_level"`  // "low", "medium", "high"
	MarketType string        `json:"market_type"` // "trending", "ranging", "volatile"
}

// GenerationResult 策略生成结果
type GenerationResult struct {
	Strategy         *strategy.Config `json:"strategy"`
	ExpectedReturn   float64          `json:"expected_return"`
	ExpectedSharpe   float64          `json:"expected_sharpe"`
	ExpectedDrawdown float64          `json:"expected_drawdown"`
	Confidence       float64          `json:"confidence"`
	BacktestResults  interface{}      `json:"backtest_results,omitempty"`
}

// Service 策略生成服务
type Service struct {
	generator *Generator
}

// NewService 创建策略生成服务
func NewService() *Service {
	generator := NewGenerator()
	generator.LoadDefaultTemplates()

	return &Service{
		generator: generator,
	}
}

// GenerateStrategy 生成策略
func (s *Service) GenerateStrategy(ctx context.Context, req *GenerationRequest) (*GenerationResult, error) {
	// 生成策略配置
	config, err := s.generator.GenerateStrategy(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate strategy: %w", err)
	}

	// 估算策略表现（基于历史数据和模型预测）
	expectedReturn, expectedSharpe, expectedDrawdown := s.estimatePerformance(config, req)

	// 计算置信度
	confidence := s.calculateConfidence(config, req)

	result := &GenerationResult{
		Strategy:         config,
		ExpectedReturn:   expectedReturn,
		ExpectedSharpe:   expectedSharpe,
		ExpectedDrawdown: expectedDrawdown,
		Confidence:       confidence,
	}

	return result, nil
}

// estimatePerformance 估算策略表现
func (s *Service) estimatePerformance(config *strategy.Config, req *GenerationRequest) (float64, float64, float64) {
	// 基于策略类型和参数估算表现
	// 这里使用简化的模型，实际应该基于历史回测数据

	baseReturn := 0.1    // 基础年化收益10%
	baseSharpe := 1.0    // 基础夏普比率1.0
	baseDrawdown := 0.08 // 基础最大回撤8%

	// 根据风险等级调整
	switch req.RiskLevel {
	case "low":
		baseReturn *= 0.7
		baseSharpe *= 1.2
		baseDrawdown *= 0.6
	case "high":
		baseReturn *= 1.5
		baseSharpe *= 0.8
		baseDrawdown *= 1.4
	}

	// 根据市场类型调整
	switch req.MarketType {
	case "trending":
		baseReturn *= 1.2
		baseDrawdown *= 0.9
	case "volatile":
		baseReturn *= 0.9
		baseDrawdown *= 1.3
	}

	return baseReturn, baseSharpe, baseDrawdown
}

// calculateConfidence 计算置信度
func (s *Service) calculateConfidence(config *strategy.Config, req *GenerationRequest) float64 {
	confidence := 0.7 // 基础置信度

	// 根据数据质量调整置信度
	if req.TimeRange >= 30*24*time.Hour { // 30天以上数据
		confidence += 0.1
	}

	// 根据市场流动性调整
	if req.Symbol == "BTCUSDT" || req.Symbol == "ETHUSDT" {
		confidence += 0.1 // 主流币种置信度更高
	}

	// 限制置信度范围
	if confidence > 0.95 {
		confidence = 0.95
	}
	if confidence < 0.3 {
		confidence = 0.3
	}

	return confidence
}

// AutoGenerationService 自动策略生成服务
type AutoGenerationService struct {
	generator       *Generator
	marketAnalyzer  *MarketAnalyzer
	performanceDB   map[string]*StrategyPerformance // 策略性能数据库
	generationRules []*GenerationRule               // 生成规则
}

// StrategyPerformance 策略性能记录
type StrategyPerformance struct {
	StrategyType    string
	Symbol          string
	TimeRange       time.Duration
	Returns         []float64
	SharpeRatio     float64
	MaxDrawdown     float64
	WinRate         float64
	TotalTrades     int
	LastUpdated     time.Time
	MarketCondition string
}

// GenerationRule 策略生成规则
type GenerationRule struct {
	Name            string
	Condition       func(*MarketAnalysis) bool
	StrategyType    string
	ParameterRanges map[string]ParameterRange
	Priority        int
	Enabled         bool
}

// ParameterRange 参数范围
type ParameterRange struct {
	Min     float64
	Max     float64
	Step    float64
	Default float64
}

// NewAutoGenerationService 创建自动生成服务
func NewAutoGenerationService() *AutoGenerationService {
	service := &AutoGenerationService{
		generator:       NewGenerator(),
		marketAnalyzer:  NewMarketAnalyzer(),
		performanceDB:   make(map[string]*StrategyPerformance),
		generationRules: make([]*GenerationRule, 0),
	}

	service.generator.LoadDefaultTemplates()
	service.loadDefaultGenerationRules()

	return service
}

// loadDefaultGenerationRules 加载默认生成规则
func (s *AutoGenerationService) loadDefaultGenerationRules() {
	// 高波动市场规则
	s.generationRules = append(s.generationRules, &GenerationRule{
		Name: "high_volatility_momentum",
		Condition: func(analysis *MarketAnalysis) bool {
			return analysis.Volatility > 0.05 && analysis.Trend > 0.3
		},
		StrategyType: "momentum_breakout",
		ParameterRanges: map[string]ParameterRange{
			"stop_loss":   {Min: 0.02, Max: 0.08, Step: 0.01, Default: 0.04},
			"take_profit": {Min: 0.04, Max: 0.12, Step: 0.01, Default: 0.08},
			"rsi_period":  {Min: 8, Max: 16, Step: 2, Default: 12},
			"ma_period":   {Min: 5, Max: 15, Step: 2, Default: 10},
		},
		Priority: 8,
		Enabled:  true,
	})

	// 低波动网格交易规则
	s.generationRules = append(s.generationRules, &GenerationRule{
		Name: "low_volatility_grid",
		Condition: func(analysis *MarketAnalysis) bool {
			return analysis.Volatility < 0.02 && math.Abs(analysis.Trend) < 0.2
		},
		StrategyType: "grid_trading",
		ParameterRanges: map[string]ParameterRange{
			"grid_spacing":  {Min: 0.005, Max: 0.02, Step: 0.001, Default: 0.01},
			"grid_levels":   {Min: 5, Max: 20, Step: 1, Default: 10},
			"position_size": {Min: 0.05, Max: 0.2, Step: 0.01, Default: 0.1},
		},
		Priority: 7,
		Enabled:  true,
	})

	// 趋势跟踪规则
	s.generationRules = append(s.generationRules, &GenerationRule{
		Name: "strong_trend_following",
		Condition: func(analysis *MarketAnalysis) bool {
			return math.Abs(analysis.Trend) > 0.5
		},
		StrategyType: "trend_following",
		ParameterRanges: map[string]ParameterRange{
			"fast_ma":     {Min: 5, Max: 15, Step: 1, Default: 10},
			"slow_ma":     {Min: 20, Max: 50, Step: 5, Default: 30},
			"stop_loss":   {Min: 0.03, Max: 0.1, Step: 0.01, Default: 0.05},
			"take_profit": {Min: 0.06, Max: 0.2, Step: 0.01, Default: 0.1},
		},
		Priority: 9,
		Enabled:  true,
	})

	// 均值回归规则
	s.generationRules = append(s.generationRules, &GenerationRule{
		Name: "mean_reversion_ranging",
		Condition: func(analysis *MarketAnalysis) bool {
			return analysis.Volatility > 0.02 && math.Abs(analysis.Trend) < 0.3
		},
		StrategyType: "mean_reversion",
		ParameterRanges: map[string]ParameterRange{
			"rsi_oversold":   {Min: 20, Max: 35, Step: 1, Default: 30},
			"rsi_overbought": {Min: 65, Max: 80, Step: 1, Default: 70},
			"bb_period":      {Min: 15, Max: 25, Step: 1, Default: 20},
			"bb_std":         {Min: 1.5, Max: 2.5, Step: 0.1, Default: 2.0},
		},
		Priority: 6,
		Enabled:  true,
	})
}

// AutoGenerateStrategies 自动生成策略
func (s *AutoGenerationService) AutoGenerateStrategies(ctx context.Context, symbols []string, maxStrategies int) ([]*GenerationResult, error) {
	log.Printf("Starting auto-generation for %d symbols, max strategies: %d", len(symbols), maxStrategies)

	var results []*GenerationResult

	for _, symbol := range symbols {
		// 分析市场数据
		analysis, err := s.marketAnalyzer.AnalyzeMarket(ctx, symbol, 30*24*time.Hour)
		if err != nil {
			log.Printf("Failed to analyze market for %s: %v", symbol, err)
			continue
		}

		// 找到匹配的生成规则
		matchingRules := s.findMatchingRules(analysis)
		if len(matchingRules) == 0 {
			log.Printf("No matching rules found for %s", symbol)
			continue
		}

		// 按优先级排序
		sort.Slice(matchingRules, func(i, j int) bool {
			return matchingRules[i].Priority > matchingRules[j].Priority
		})

		// 为每个匹配的规则生成策略
		for _, rule := range matchingRules {
			if len(results) >= maxStrategies {
				break
			}

			result, err := s.generateStrategyFromRule(ctx, symbol, analysis, rule)
			if err != nil {
				log.Printf("Failed to generate strategy from rule %s for %s: %v", rule.Name, symbol, err)
				continue
			}

			results = append(results, result)
		}

		if len(results) >= maxStrategies {
			break
		}
	}

	log.Printf("Auto-generated %d strategies", len(results))
	return results, nil
}

// findMatchingRules 找到匹配的生成规则
func (s *AutoGenerationService) findMatchingRules(analysis *MarketAnalysis) []*GenerationRule {
	var matchingRules []*GenerationRule

	for _, rule := range s.generationRules {
		if rule.Enabled && rule.Condition(analysis) {
			matchingRules = append(matchingRules, rule)
		}
	}

	return matchingRules
}

// generateStrategyFromRule 从规则生成策略
func (s *AutoGenerationService) generateStrategyFromRule(ctx context.Context, symbol string, analysis *MarketAnalysis, rule *GenerationRule) (*GenerationResult, error) {
	// 创建生成请求
	req := &GenerationRequest{
		Symbol:     symbol,
		Exchange:   "binance",
		TimeRange:  30 * 24 * time.Hour,
		Objective:  "sharpe",
		RiskLevel:  s.determineRiskLevel(analysis),
		MarketType: s.determineMarketType(analysis),
	}

	// 生成基础策略
	config, err := s.generator.GenerateStrategy(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate base strategy: %w", err)
	}

	// 使用规则优化参数
	optimizedParams := s.optimizeParametersWithRule(config.Params, rule, analysis)
	config.Params = optimizedParams

	// 添加规则信息到策略名称
	config.Name = fmt.Sprintf("Auto_%s_%s_%d", rule.Name, symbol, time.Now().Unix())
	config.Description = fmt.Sprintf("Auto-generated %s strategy for %s based on rule: %s",
		rule.StrategyType, symbol, rule.Name)

	// 估算性能
	expectedReturn, expectedSharpe, expectedDrawdown := s.estimatePerformanceWithHistory(config, req, rule)

	// 计算置信度
	confidence := s.calculateAdvancedConfidence(config, req, analysis, rule)

	result := &GenerationResult{
		Strategy:         config,
		ExpectedReturn:   expectedReturn,
		ExpectedSharpe:   expectedSharpe,
		ExpectedDrawdown: expectedDrawdown,
		Confidence:       confidence,
	}

	return result, nil
}

// determineRiskLevel 确定风险等级
func (s *AutoGenerationService) determineRiskLevel(analysis *MarketAnalysis) string {
	if analysis.Volatility > 0.08 || analysis.MaxDrawdown > 0.2 {
		return "high"
	} else if analysis.Volatility > 0.04 || analysis.MaxDrawdown > 0.1 {
		return "medium"
	}
	return "low"
}

// determineMarketType 确定市场类型
func (s *AutoGenerationService) determineMarketType(analysis *MarketAnalysis) string {
	if math.Abs(analysis.Trend) > 0.5 {
		return "trending"
	} else if analysis.Volatility > 0.05 {
		return "volatile"
	}
	return "ranging"
}

// optimizeParametersWithRule 使用规则优化参数
func (s *AutoGenerationService) optimizeParametersWithRule(baseParams map[string]interface{}, rule *GenerationRule, analysis *MarketAnalysis) map[string]interface{} {
	optimizedParams := make(map[string]interface{})

	// 复制基础参数
	for k, v := range baseParams {
		optimizedParams[k] = v
	}

	// 应用规则参数范围
	for paramName, paramRange := range rule.ParameterRanges {
		// 基于市场分析调整参数
		value := s.calculateOptimalParameterValue(paramName, paramRange, analysis)
		optimizedParams[paramName] = value
	}

	return optimizedParams
}

// calculateOptimalParameterValue 计算最优参数值
func (s *AutoGenerationService) calculateOptimalParameterValue(paramName string, paramRange ParameterRange, analysis *MarketAnalysis) float64 {
	// 基于市场分析在参数范围内选择最优值
	switch paramName {
	case "stop_loss":
		// 高波动市场使用更大的止损
		factor := 1.0 + analysis.Volatility*5
		value := paramRange.Default * factor
		return math.Max(paramRange.Min, math.Min(paramRange.Max, value))

	case "take_profit":
		// 趋势市场使用更大的止盈
		factor := 1.0 + math.Abs(analysis.Trend)*2
		value := paramRange.Default * factor
		return math.Max(paramRange.Min, math.Min(paramRange.Max, value))

	case "rsi_period":
		// 高波动市场使用更短的RSI周期
		if analysis.Volatility > 0.05 {
			return paramRange.Min
		} else if analysis.Volatility < 0.02 {
			return paramRange.Max
		}
		return paramRange.Default

	case "ma_period":
		// 强趋势市场使用更短的MA周期
		if math.Abs(analysis.Trend) > 0.5 {
			return paramRange.Min
		} else if math.Abs(analysis.Trend) < 0.2 {
			return paramRange.Max
		}
		return paramRange.Default

	case "grid_spacing":
		// 低波动市场使用更小的网格间距
		factor := 1.0 / (1.0 + analysis.Volatility*10)
		value := paramRange.Default * factor
		return math.Max(paramRange.Min, math.Min(paramRange.Max, value))

	default:
		return paramRange.Default
	}
}

// estimatePerformanceWithHistory 基于历史数据估算性能
func (s *AutoGenerationService) estimatePerformanceWithHistory(config *strategy.Config, req *GenerationRequest, rule *GenerationRule) (float64, float64, float64) {
	// 查找历史性能数据
	historyKey := fmt.Sprintf("%s_%s", rule.StrategyType, req.Symbol)
	if performance, exists := s.performanceDB[historyKey]; exists {
		// 基于历史数据调整预期
		baseReturn := performance.SharpeRatio * 0.1 // 假设无风险利率为0，简化计算
		baseSharpe := performance.SharpeRatio
		baseDrawdown := performance.MaxDrawdown

		// 根据市场条件调整
		marketAdjustment := s.calculateMarketAdjustment(req)

		return baseReturn * marketAdjustment,
			baseSharpe * marketAdjustment,
			baseDrawdown / marketAdjustment
	}

	// 如果没有历史数据，使用基础估算
	return s.estimatePerformanceBaseline(config, req, rule)
}

// estimatePerformanceBaseline 基础性能估算
func (s *AutoGenerationService) estimatePerformanceBaseline(config *strategy.Config, req *GenerationRequest, rule *GenerationRule) (float64, float64, float64) {
	// 基于策略类型的基础性能
	var baseReturn, baseSharpe, baseDrawdown float64

	switch rule.StrategyType {
	case "momentum_breakout":
		baseReturn = 0.15   // 15%年化收益
		baseSharpe = 1.2    // 夏普比率1.2
		baseDrawdown = 0.12 // 12%最大回撤
	case "mean_reversion":
		baseReturn = 0.12
		baseSharpe = 1.5
		baseDrawdown = 0.08
	case "grid_trading":
		baseReturn = 0.08
		baseSharpe = 2.0
		baseDrawdown = 0.05
	case "trend_following":
		baseReturn = 0.18
		baseSharpe = 1.0
		baseDrawdown = 0.15
	default:
		baseReturn = 0.10
		baseSharpe = 1.0
		baseDrawdown = 0.10
	}

	// 根据风险等级调整
	switch req.RiskLevel {
	case "low":
		baseReturn *= 0.7
		baseSharpe *= 1.3
		baseDrawdown *= 0.6
	case "high":
		baseReturn *= 1.4
		baseSharpe *= 0.8
		baseDrawdown *= 1.5
	}

	// 根据市场类型调整
	marketAdjustment := s.calculateMarketAdjustment(req)

	return baseReturn * marketAdjustment,
		baseSharpe * marketAdjustment,
		baseDrawdown / marketAdjustment
}

// calculateMarketAdjustment 计算市场调整因子
func (s *AutoGenerationService) calculateMarketAdjustment(req *GenerationRequest) float64 {
	adjustment := 1.0

	switch req.MarketType {
	case "trending":
		adjustment = 1.2 // 趋势市场表现更好
	case "volatile":
		adjustment = 0.9 // 波动市场表现稍差
	case "ranging":
		adjustment = 1.0 // 震荡市场正常表现
	}

	// 根据币种调整
	if req.Symbol == "BTCUSDT" || req.Symbol == "ETHUSDT" {
		adjustment *= 1.1 // 主流币种表现更好
	}

	return adjustment
}

// calculateAdvancedConfidence 计算高级置信度
func (s *AutoGenerationService) calculateAdvancedConfidence(config *strategy.Config, req *GenerationRequest, analysis *MarketAnalysis, rule *GenerationRule) float64 {
	confidence := 0.6 // 基础置信度

	// 基于历史数据调整置信度
	historyKey := fmt.Sprintf("%s_%s", rule.StrategyType, req.Symbol)
	if performance, exists := s.performanceDB[historyKey]; exists {
		// 有历史数据，提高置信度
		confidence += 0.2

		// 基于历史表现调整
		if performance.SharpeRatio > 1.5 {
			confidence += 0.1
		}
		if performance.WinRate > 0.6 {
			confidence += 0.1
		}
	}

	// 基于市场分析质量调整
	if req.TimeRange >= 30*24*time.Hour {
		confidence += 0.1 // 数据充足
	}

	// 基于规则优先级调整
	if rule.Priority >= 8 {
		confidence += 0.1 // 高优先级规则
	}

	// 基于市场条件匹配度调整
	if analysis.Volatility > 0.02 && analysis.Volatility < 0.08 {
		confidence += 0.05 // 适中波动率
	}

	// 限制置信度范围
	if confidence > 0.95 {
		confidence = 0.95
	}
	if confidence < 0.3 {
		confidence = 0.3
	}

	return confidence
}

// UpdatePerformanceHistory 更新策略性能历史
func (s *AutoGenerationService) UpdatePerformanceHistory(strategyType, symbol string, performance *StrategyPerformance) {
	historyKey := fmt.Sprintf("%s_%s", strategyType, symbol)
	s.performanceDB[historyKey] = performance
	log.Printf("Updated performance history for %s: Sharpe=%.2f, Drawdown=%.2f",
		historyKey, performance.SharpeRatio, performance.MaxDrawdown)
}

// GetGenerationRules 获取生成规则
func (s *AutoGenerationService) GetGenerationRules() []*GenerationRule {
	return s.generationRules
}

// AddGenerationRule 添加生成规则
func (s *AutoGenerationService) AddGenerationRule(rule *GenerationRule) {
	s.generationRules = append(s.generationRules, rule)
	log.Printf("Added generation rule: %s", rule.Name)
}

// EnableRule 启用规则
func (s *AutoGenerationService) EnableRule(ruleName string) error {
	for _, rule := range s.generationRules {
		if rule.Name == ruleName {
			rule.Enabled = true
			log.Printf("Enabled generation rule: %s", ruleName)
			return nil
		}
	}
	return fmt.Errorf("rule not found: %s", ruleName)
}

// DisableRule 禁用规则
func (s *AutoGenerationService) DisableRule(ruleName string) error {
	for _, rule := range s.generationRules {
		if rule.Name == ruleName {
			rule.Enabled = false
			log.Printf("Disabled generation rule: %s", ruleName)
			return nil
		}
	}
	return fmt.Errorf("rule not found: %s", ruleName)
}
