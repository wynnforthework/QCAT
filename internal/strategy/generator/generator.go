package generator

import (
	"context"
	"fmt"
	"log"
	"math"
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
