package onboarding

import (
	"context"
	"math"
	"time"
)

// RiskAssessor 风险评估器
type RiskAssessor struct {
	models []RiskModel
}

// NewRiskAssessor 创建新的风险评估器
func NewRiskAssessor() *RiskAssessor {
	return &RiskAssessor{
		models: []RiskModel{
			&DrawdownRiskModel{},
			&VolatilityRiskModel{},
			&LeverageRiskModel{},
			&ConcentrationRiskModel{},
			&LiquidityRiskModel{},
		},
	}
}

// RiskModel 风险模型接口
type RiskModel interface {
	Name() string
	Assess(ctx context.Context, req *OnboardingRequest, validation *ValidationResult) *RiskScore
}

// RiskAssessment 风险评估结果
type RiskAssessment struct {
	OverallScore     float64                `json:"overall_score"`     // 0-100
	RiskLevel        string                 `json:"risk_level"`        // "low", "medium", "high", "unacceptable"
	Scores           map[string]*RiskScore  `json:"scores"`
	Recommendations  []string               `json:"recommendations"`
	Warnings         []string               `json:"warnings"`
	ExpectedReturn   float64                `json:"expected_return"`
	ExpectedSharpe   float64                `json:"expected_sharpe"`
	ExpectedDrawdown float64                `json:"expected_drawdown"`
	ConfidenceLevel  float64                `json:"confidence_level"`
	Duration         time.Duration          `json:"duration"`
}

// RiskScore 风险评分
type RiskScore struct {
	ModelName   string  `json:"model_name"`
	Score       float64 `json:"score"`       // 0-100 (100 = lowest risk)
	Weight      float64 `json:"weight"`      // 权重
	Description string  `json:"description"`
	Details     string  `json:"details"`
}

// AssessRisk 评估策略风险
func (ra *RiskAssessor) AssessRisk(ctx context.Context, req *OnboardingRequest, validation *ValidationResult) (*RiskAssessment, error) {
	startTime := time.Now()

	assessment := &RiskAssessment{
		Scores:          make(map[string]*RiskScore),
		Recommendations: make([]string, 0),
		Warnings:        make([]string, 0),
	}

	var totalScore float64
	var totalWeight float64

	// 执行所有风险模型
	for _, model := range ra.models {
		score := model.Assess(ctx, req, validation)
		if score != nil {
			assessment.Scores[model.Name()] = score
			totalScore += score.Score * score.Weight
			totalWeight += score.Weight
		}
	}

	// 计算总体风险评分
	if totalWeight > 0 {
		assessment.OverallScore = totalScore / totalWeight
	} else {
		assessment.OverallScore = 50.0 // 默认中等风险
	}

	// 确定风险等级
	assessment.RiskLevel = ra.determineRiskLevel(assessment.OverallScore)

	// 生成预期表现指标
	ra.generateExpectedMetrics(assessment, req)

	// 生成建议和警告
	ra.generateRecommendations(assessment, req)

	assessment.Duration = time.Since(startTime)
	return assessment, nil
}

// determineRiskLevel 确定风险等级
func (ra *RiskAssessor) determineRiskLevel(score float64) string {
	switch {
	case score >= 80:
		return "low"
	case score >= 60:
		return "medium"
	case score >= 40:
		return "high"
	default:
		return "unacceptable"
	}
}

// generateExpectedMetrics 生成预期表现指标
func (ra *RiskAssessor) generateExpectedMetrics(assessment *RiskAssessment, req *OnboardingRequest) {
	// 基于风险评分和策略参数估算表现
	baseReturn := 0.1 // 基础年化收益10%
	baseSharpe := 1.0
	baseDrawdown := 0.08

	// 根据风险等级调整
	switch assessment.RiskLevel {
	case "low":
		assessment.ExpectedReturn = baseReturn * 0.8
		assessment.ExpectedSharpe = baseSharpe * 1.2
		assessment.ExpectedDrawdown = baseDrawdown * 0.6
		assessment.ConfidenceLevel = 0.85
	case "medium":
		assessment.ExpectedReturn = baseReturn
		assessment.ExpectedSharpe = baseSharpe
		assessment.ExpectedDrawdown = baseDrawdown
		assessment.ConfidenceLevel = 0.75
	case "high":
		assessment.ExpectedReturn = baseReturn * 1.3
		assessment.ExpectedSharpe = baseSharpe * 0.8
		assessment.ExpectedDrawdown = baseDrawdown * 1.5
		assessment.ConfidenceLevel = 0.6
	default:
		assessment.ExpectedReturn = baseReturn * 0.5
		assessment.ExpectedSharpe = baseSharpe * 0.5
		assessment.ExpectedDrawdown = baseDrawdown * 2.0
		assessment.ConfidenceLevel = 0.3
	}

	// 根据策略参数微调
	if req.Parameters != nil {
		if leverage, exists := req.Parameters["leverage"]; exists {
			if lev, ok := leverage.(float64); ok && lev > 1 {
				assessment.ExpectedReturn *= lev * 0.8
				assessment.ExpectedDrawdown *= math.Sqrt(lev)
				assessment.ExpectedSharpe /= math.Sqrt(lev)
			}
		}
	}
}

// generateRecommendations 生成建议和警告
func (ra *RiskAssessor) generateRecommendations(assessment *RiskAssessment, req *OnboardingRequest) {
	switch assessment.RiskLevel {
	case "low":
		assessment.Recommendations = append(assessment.Recommendations,
			"Strategy shows low risk profile, suitable for conservative portfolios",
			"Consider increasing position size for better returns",
			"Monitor performance regularly for optimization opportunities")

	case "medium":
		assessment.Recommendations = append(assessment.Recommendations,
			"Strategy has moderate risk, implement standard risk controls",
			"Set up automated stop-loss and take-profit levels",
			"Monitor drawdown closely and adjust position sizing")

	case "high":
		assessment.Recommendations = append(assessment.Recommendations,
			"High-risk strategy requires careful monitoring",
			"Implement strict position sizing limits",
			"Consider reducing leverage or increasing stop-loss levels",
			"Start with paper trading to validate performance")
		assessment.Warnings = append(assessment.Warnings,
			"High risk strategy may result in significant losses",
			"Manual approval required for deployment")

	case "unacceptable":
		assessment.Warnings = append(assessment.Warnings,
			"Strategy risk level is too high for automated deployment",
			"Requires significant modifications before approval",
			"Consider fundamental strategy redesign")
	}

	// 基于具体风险评分添加建议
	if drawdownScore, exists := assessment.Scores["drawdown_risk"]; exists && drawdownScore.Score < 60 {
		assessment.Warnings = append(assessment.Warnings, "High drawdown risk detected")
		assessment.Recommendations = append(assessment.Recommendations, "Implement tighter stop-loss controls")
	}

	if leverageScore, exists := assessment.Scores["leverage_risk"]; exists && leverageScore.Score < 70 {
		assessment.Warnings = append(assessment.Warnings, "High leverage risk detected")
		assessment.Recommendations = append(assessment.Recommendations, "Consider reducing leverage")
	}
}

// DrawdownRiskModel 回撤风险模型
type DrawdownRiskModel struct{}

func (m *DrawdownRiskModel) Name() string {
	return "drawdown_risk"
}

func (m *DrawdownRiskModel) Assess(ctx context.Context, req *OnboardingRequest, validation *ValidationResult) *RiskScore {
	score := 80.0 // 默认较低风险
	
	if req.RiskProfile != nil {
		maxDrawdown := req.RiskProfile.MaxDrawdown
		
		// 根据最大回撤调整评分
		switch {
		case maxDrawdown <= 0.05: // 5%以下
			score = 95.0
		case maxDrawdown <= 0.1: // 10%以下
			score = 85.0
		case maxDrawdown <= 0.2: // 20%以下
			score = 70.0
		case maxDrawdown <= 0.3: // 30%以下
			score = 50.0
		default:
			score = 20.0
		}
	}

	return &RiskScore{
		ModelName:   m.Name(),
		Score:       score,
		Weight:      0.3, // 30%权重
		Description: "Evaluates maximum drawdown risk",
		Details:     "Lower drawdown indicates better risk control",
	}
}

// VolatilityRiskModel 波动率风险模型
type VolatilityRiskModel struct{}

func (m *VolatilityRiskModel) Name() string {
	return "volatility_risk"
}

func (m *VolatilityRiskModel) Assess(ctx context.Context, req *OnboardingRequest, validation *ValidationResult) *RiskScore {
	score := 75.0 // 默认评分

	// 基于交易对评估波动率风险
	if req.Config != nil {
		symbol := req.Config.Symbol
		switch {
		case symbol == "BTCUSDT" || symbol == "ETHUSDT":
			score = 80.0 // 主流币种风险较低
		case len(symbol) <= 7: // 可能是主流币种
			score = 70.0
		default:
			score = 60.0 // 小币种风险较高
		}
	}

	return &RiskScore{
		ModelName:   m.Name(),
		Score:       score,
		Weight:      0.2, // 20%权重
		Description: "Evaluates market volatility risk",
		Details:     "Based on trading pair characteristics",
	}
}

// LeverageRiskModel 杠杆风险模型
type LeverageRiskModel struct{}

func (m *LeverageRiskModel) Name() string {
	return "leverage_risk"
}

func (m *LeverageRiskModel) Assess(ctx context.Context, req *OnboardingRequest, validation *ValidationResult) *RiskScore {
	score := 90.0 // 默认低杠杆风险

	if req.RiskProfile != nil {
		leverage := req.RiskProfile.MaxLeverage
		
		switch {
		case leverage <= 2:
			score = 95.0
		case leverage <= 5:
			score = 85.0
		case leverage <= 10:
			score = 70.0
		case leverage <= 20:
			score = 50.0
		default:
			score = 20.0
		}
	}

	return &RiskScore{
		ModelName:   m.Name(),
		Score:       score,
		Weight:      0.25, // 25%权重
		Description: "Evaluates leverage risk",
		Details:     "Higher leverage increases risk exponentially",
	}
}

// ConcentrationRiskModel 集中度风险模型
type ConcentrationRiskModel struct{}

func (m *ConcentrationRiskModel) Name() string {
	return "concentration_risk"
}

func (m *ConcentrationRiskModel) Assess(ctx context.Context, req *OnboardingRequest, validation *ValidationResult) *RiskScore {
	score := 80.0 // 默认评分

	if req.RiskProfile != nil {
		positionSize := req.RiskProfile.MaxPositionSize
		
		switch {
		case positionSize <= 0.1: // 10%以下
			score = 90.0
		case positionSize <= 0.2: // 20%以下
			score = 80.0
		case positionSize <= 0.5: // 50%以下
			score = 60.0
		default:
			score = 30.0
		}
	}

	return &RiskScore{
		ModelName:   m.Name(),
		Score:       score,
		Weight:      0.15, // 15%权重
		Description: "Evaluates position concentration risk",
		Details:     "Large positions increase portfolio risk",
	}
}

// LiquidityRiskModel 流动性风险模型
type LiquidityRiskModel struct{}

func (m *LiquidityRiskModel) Name() string {
	return "liquidity_risk"
}

func (m *LiquidityRiskModel) Assess(ctx context.Context, req *OnboardingRequest, validation *ValidationResult) *RiskScore {
	score := 75.0 // 默认评分

	if req.Config != nil {
		symbol := req.Config.Symbol
		// 主流交易对流动性风险较低
		if symbol == "BTCUSDT" || symbol == "ETHUSDT" || symbol == "BNBUSDT" {
			score = 90.0
		} else if len(symbol) <= 8 {
			score = 75.0
		} else {
			score = 60.0
		}
	}

	return &RiskScore{
		ModelName:   m.Name(),
		Score:       score,
		Weight:      0.1, // 10%权重
		Description: "Evaluates market liquidity risk",
		Details:     "Low liquidity increases execution risk",
	}
}
