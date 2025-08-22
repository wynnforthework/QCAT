package validation

import (
	"context"
	"fmt"
	"log"
	"time"

	"qcat/internal/strategy/lifecycle"
)

// StrategyGatekeeper 策略守门员 - 确保只有通过验证的策略才能启用
type StrategyGatekeeper struct {
	backtestValidator *MandatoryBacktestValidator
	riskValidator     *RiskValidator
	enabled           bool
}

// RiskValidator 风险验证器
type RiskValidator struct {
	maxPositionSize    float64 // 最大单个持仓大小
	maxTotalExposure   float64 // 最大总敞口
	maxDailyLoss       float64 // 最大日损失
	maxConsecutiveLoss int     // 最大连续亏损次数
}

// ValidationStatus 验证状态
type ValidationStatus struct {
	StrategyID       string                 `json:"strategy_id"`
	IsValid          bool                   `json:"is_valid"`
	BacktestPassed   bool                   `json:"backtest_passed"`
	RiskCheckPassed  bool                   `json:"risk_check_passed"`
	ValidationTime   time.Time              `json:"validation_time"`
	BacktestResult   *BacktestResult        `json:"backtest_result,omitempty"`
	RiskAssessment   *RiskAssessment        `json:"risk_assessment,omitempty"`
	Errors           []ValidationError      `json:"errors,omitempty"`
	Warnings         []ValidationError      `json:"warnings,omitempty"`
	NextRevalidation time.Time              `json:"next_revalidation"`
}

// RiskAssessment 风险评估
type RiskAssessment struct {
	RiskScore        float64   `json:"risk_score"`        // 0-100风险评分
	RiskLevel        string    `json:"risk_level"`        // LOW/MEDIUM/HIGH/CRITICAL
	MaxPositionSize  float64   `json:"max_position_size"` // 建议最大持仓
	MaxLeverage      float64   `json:"max_leverage"`      // 建议最大杠杆
	RecommendedLimit float64   `json:"recommended_limit"` // 建议资金限制
	Warnings         []string  `json:"warnings"`
}

// NewStrategyGatekeeper 创建策略守门员
func NewStrategyGatekeeper() *StrategyGatekeeper {
	return &StrategyGatekeeper{
		backtestValidator: NewMandatoryBacktestValidator(),
		riskValidator: &RiskValidator{
			maxPositionSize:    0.1,  // 单个持仓不超过10%
			maxTotalExposure:   0.8,  // 总敞口不超过80%
			maxDailyLoss:       0.05, // 日损失不超过5%
			maxConsecutiveLoss: 5,    // 最多连续5次亏损
		},
		enabled: true,
	}
}

// ValidateStrategyForActivation 验证策略是否可以激活
func (sg *StrategyGatekeeper) ValidateStrategyForActivation(ctx context.Context, strategyID string, config *lifecycle.Version) (*ValidationStatus, error) {
	if !sg.enabled {
		log.Printf("策略守门员已禁用，跳过验证")
		return &ValidationStatus{
			StrategyID:     strategyID,
			IsValid:        true,
			ValidationTime: time.Now(),
		}, nil
	}

	log.Printf("开始验证策略 %s 是否可以激活", strategyID)

	status := &ValidationStatus{
		StrategyID:     strategyID,
		ValidationTime: time.Now(),
		Errors:         make([]ValidationError, 0),
		Warnings:       make([]ValidationError, 0),
	}

	// 1. 强制回测验证
	log.Printf("执行强制回测验证...")
	backtestResult, err := sg.backtestValidator.ValidateStrategy(ctx, strategyID, config)
	if err != nil {
		status.BacktestPassed = false
		status.Errors = append(status.Errors, ValidationError{
			Code:    "BACKTEST_FAILED",
			Message: fmt.Sprintf("回测验证失败: %v", err),
			Field:   "backtest",
		})
		log.Printf("策略 %s 回测验证失败: %v", strategyID, err)
	} else {
		status.BacktestPassed = true
		status.BacktestResult = backtestResult
		log.Printf("策略 %s 回测验证通过", strategyID)
	}

	// 2. 风险评估
	log.Printf("执行风险评估...")
	riskAssessment, err := sg.assessRisk(ctx, strategyID, config, backtestResult)
	if err != nil {
		status.RiskCheckPassed = false
		status.Errors = append(status.Errors, ValidationError{
			Code:    "RISK_ASSESSMENT_FAILED",
			Message: fmt.Sprintf("风险评估失败: %v", err),
			Field:   "risk",
		})
		log.Printf("策略 %s 风险评估失败: %v", strategyID, err)
	} else {
		status.RiskAssessment = riskAssessment
		if riskAssessment.RiskLevel == "CRITICAL" {
			status.RiskCheckPassed = false
			status.Errors = append(status.Errors, ValidationError{
				Code:    "RISK_TOO_HIGH",
				Message: "策略风险等级过高，不允许启用",
				Field:   "risk_level",
			})
		} else {
			status.RiskCheckPassed = true
			if riskAssessment.RiskLevel == "HIGH" {
				status.Warnings = append(status.Warnings, ValidationError{
					Code:    "HIGH_RISK_WARNING",
					Message: "策略风险等级较高，建议谨慎使用",
					Field:   "risk_level",
				})
			}
		}
	}

	// 3. 综合判断
	status.IsValid = status.BacktestPassed && status.RiskCheckPassed

	// 4. 设置下次重新验证时间
	if status.IsValid {
		status.NextRevalidation = time.Now().AddDate(0, 1, 0) // 1个月后重新验证
	} else {
		status.NextRevalidation = time.Now().AddDate(0, 0, 7) // 1周后可重新验证
	}

	if status.IsValid {
		log.Printf("策略 %s 验证通过，可以激活", strategyID)
	} else {
		log.Printf("策略 %s 验证失败，不能激活。错误: %d个，警告: %d个", strategyID, len(status.Errors), len(status.Warnings))
	}

	return status, nil
}

// assessRisk 评估策略风险
func (sg *StrategyGatekeeper) assessRisk(ctx context.Context, strategyID string, config *lifecycle.Version, backtestResult *BacktestResult) (*RiskAssessment, error) {
	assessment := &RiskAssessment{
		Warnings: make([]string, 0),
	}

	var riskScore float64 = 0

	// 基于回测结果评估风险
	if backtestResult != nil {
		// 最大回撤风险
		if backtestResult.MaxDrawdown > 0.15 {
			riskScore += 30
			assessment.Warnings = append(assessment.Warnings, "最大回撤超过15%")
		} else if backtestResult.MaxDrawdown > 0.10 {
			riskScore += 15
		}

		// 夏普比率风险
		if backtestResult.SharpeRatio < 0.5 {
			riskScore += 25
			assessment.Warnings = append(assessment.Warnings, "夏普比率过低")
		} else if backtestResult.SharpeRatio < 1.0 {
			riskScore += 10
		}

		// 交易频率风险
		if backtestResult.TotalTrades > backtestResult.BacktestDays*5 {
			riskScore += 20
			assessment.Warnings = append(assessment.Warnings, "交易频率过高，可能存在过度交易")
		}

		// 胜率风险
		if backtestResult.WinRate < 0.4 {
			riskScore += 15
			assessment.Warnings = append(assessment.Warnings, "胜率过低")
		}
	} else {
		// 没有回测结果，风险很高
		riskScore += 50
		assessment.Warnings = append(assessment.Warnings, "缺少回测验证")
	}

	assessment.RiskScore = riskScore

	// 确定风险等级
	if riskScore >= 80 {
		assessment.RiskLevel = "CRITICAL"
	} else if riskScore >= 60 {
		assessment.RiskLevel = "HIGH"
	} else if riskScore >= 30 {
		assessment.RiskLevel = "MEDIUM"
	} else {
		assessment.RiskLevel = "LOW"
	}

	// 设置建议参数
	switch assessment.RiskLevel {
	case "LOW":
		assessment.MaxPositionSize = 0.1  // 10%
		assessment.MaxLeverage = 5.0
		assessment.RecommendedLimit = 100000 // $100k
	case "MEDIUM":
		assessment.MaxPositionSize = 0.05 // 5%
		assessment.MaxLeverage = 3.0
		assessment.RecommendedLimit = 50000 // $50k
	case "HIGH":
		assessment.MaxPositionSize = 0.02 // 2%
		assessment.MaxLeverage = 2.0
		assessment.RecommendedLimit = 10000 // $10k
	case "CRITICAL":
		assessment.MaxPositionSize = 0.01 // 1%
		assessment.MaxLeverage = 1.0
		assessment.RecommendedLimit = 1000 // $1k
	}

	return assessment, nil
}

// DisableStrategy 禁用策略（紧急情况）
func (sg *StrategyGatekeeper) DisableStrategy(ctx context.Context, strategyID string, reason string) error {
	log.Printf("紧急禁用策略 %s，原因: %s", strategyID, reason)
	
	// 这里应该调用策略管理器来禁用策略
	// 并通知所有相关系统
	
	return nil
}

// GetValidationHistory 获取验证历史
func (sg *StrategyGatekeeper) GetValidationHistory(ctx context.Context, strategyID string) ([]*ValidationStatus, error) {
	// 这里应该从数据库查询验证历史
	return nil, fmt.Errorf("not implemented")
}

// Enable 启用守门员
func (sg *StrategyGatekeeper) Enable() {
	sg.enabled = true
	log.Printf("策略守门员已启用")
}

// Disable 禁用守门员（仅用于紧急情况）
func (sg *StrategyGatekeeper) Disable() {
	sg.enabled = false
	log.Printf("警告: 策略守门员已禁用！所有策略将跳过验证")
}
