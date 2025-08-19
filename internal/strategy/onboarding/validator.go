package onboarding

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Validator 策略验证器
type Validator struct {
	rules []ValidationRule
}

// NewValidator 创建新的验证器
func NewValidator() *Validator {
	validator := &Validator{
		rules: make([]ValidationRule, 0),
	}
	
	// 加载默认验证规则
	validator.loadDefaultRules()
	
	return validator
}

// ValidationRule 验证规则接口
type ValidationRule interface {
	Name() string
	Validate(ctx context.Context, req *OnboardingRequest) *ValidationError
}

// ValidationResult 验证结果
type ValidationResult struct {
	IsValid   bool               `json:"is_valid"`
	Score     float64            `json:"score"`     // 0-100
	Errors    []*ValidationError `json:"errors"`
	Warnings  []*ValidationError `json:"warnings"`
	Passed    []string           `json:"passed"`
	Duration  time.Duration      `json:"duration"`
}

// ValidationError 验证错误
type ValidationError struct {
	Rule        string `json:"rule"`
	Severity    string `json:"severity"` // "error", "warning", "info"
	Message     string `json:"message"`
	Field       string `json:"field,omitempty"`
	Suggestion  string `json:"suggestion,omitempty"`
}

// ValidateStrategy 验证策略
func (v *Validator) ValidateStrategy(ctx context.Context, req *OnboardingRequest) (*ValidationResult, error) {
	startTime := time.Now()
	
	result := &ValidationResult{
		IsValid:  true,
		Score:    100.0,
		Errors:   make([]*ValidationError, 0),
		Warnings: make([]*ValidationError, 0),
		Passed:   make([]string, 0),
	}

	// 执行所有验证规则
	for _, rule := range v.rules {
		if err := rule.Validate(ctx, req); err != nil {
			if err.Severity == "error" {
				result.Errors = append(result.Errors, err)
				result.IsValid = false
				result.Score -= 20.0 // 每个错误扣20分
			} else if err.Severity == "warning" {
				result.Warnings = append(result.Warnings, err)
				result.Score -= 5.0 // 每个警告扣5分
			}
		} else {
			result.Passed = append(result.Passed, rule.Name())
		}
	}

	// 确保分数不低于0
	if result.Score < 0 {
		result.Score = 0
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// loadDefaultRules 加载默认验证规则
func (v *Validator) loadDefaultRules() {
	v.rules = []ValidationRule{
		&ConfigValidationRule{},
		&ParameterValidationRule{},
		&RiskValidationRule{},
		&CodeQualityRule{},
		&PerformanceRule{},
		&SecurityRule{},
	}
}

// ConfigValidationRule 配置验证规则
type ConfigValidationRule struct{}

func (r *ConfigValidationRule) Name() string {
	return "config_validation"
}

func (r *ConfigValidationRule) Validate(ctx context.Context, req *OnboardingRequest) *ValidationError {
	if req.Config == nil {
		return &ValidationError{
			Rule:       r.Name(),
			Severity:   "error",
			Message:    "Strategy configuration is required",
			Field:      "config",
			Suggestion: "Provide a valid strategy configuration",
		}
	}

	if req.Config.Name == "" {
		return &ValidationError{
			Rule:       r.Name(),
			Severity:   "error",
			Message:    "Strategy name is required",
			Field:      "config.name",
			Suggestion: "Provide a descriptive strategy name",
		}
	}

	if req.Config.Symbol == "" {
		return &ValidationError{
			Rule:       r.Name(),
			Severity:   "error",
			Message:    "Trading symbol is required",
			Field:      "config.symbol",
			Suggestion: "Specify the trading symbol (e.g., BTCUSDT)",
		}
	}

	// 验证交易对格式
	symbolPattern := regexp.MustCompile(`^[A-Z]{3,10}USDT?$`)
	if !symbolPattern.MatchString(req.Config.Symbol) {
		return &ValidationError{
			Rule:       r.Name(),
			Severity:   "warning",
			Message:    "Trading symbol format may be invalid",
			Field:      "config.symbol",
			Suggestion: "Use standard format like BTCUSDT",
		}
	}

	return nil
}

// ParameterValidationRule 参数验证规则
type ParameterValidationRule struct{}

func (r *ParameterValidationRule) Name() string {
	return "parameter_validation"
}

func (r *ParameterValidationRule) Validate(ctx context.Context, req *OnboardingRequest) *ValidationError {
	if req.Parameters == nil || len(req.Parameters) == 0 {
		return &ValidationError{
			Rule:       r.Name(),
			Severity:   "warning",
			Message:    "No strategy parameters provided",
			Field:      "parameters",
			Suggestion: "Consider adding strategy parameters for better control",
		}
	}

	// 验证关键参数
	if stopLoss, exists := req.Parameters["stop_loss"]; exists {
		if sl, ok := stopLoss.(float64); ok {
			if sl <= 0 || sl > 0.5 {
				return &ValidationError{
					Rule:       r.Name(),
					Severity:   "error",
					Message:    "Stop loss parameter is out of valid range (0-0.5)",
					Field:      "parameters.stop_loss",
					Suggestion: "Set stop loss between 0.01 (1%) and 0.5 (50%)",
				}
			}
		}
	}

	if positionSize, exists := req.Parameters["position_size"]; exists {
		if ps, ok := positionSize.(float64); ok {
			if ps <= 0 || ps > 1.0 {
				return &ValidationError{
					Rule:       r.Name(),
					Severity:   "error",
					Message:    "Position size parameter is out of valid range (0-1.0)",
					Field:      "parameters.position_size",
					Suggestion: "Set position size between 0.01 (1%) and 1.0 (100%)",
				}
			}
		}
	}

	return nil
}

// RiskValidationRule 风险验证规则
type RiskValidationRule struct{}

func (r *RiskValidationRule) Name() string {
	return "risk_validation"
}

func (r *RiskValidationRule) Validate(ctx context.Context, req *OnboardingRequest) *ValidationError {
	if req.RiskProfile == nil {
		return &ValidationError{
			Rule:       r.Name(),
			Severity:   "warning",
			Message:    "No risk profile provided, using default settings",
			Field:      "risk_profile",
			Suggestion: "Provide a risk profile for better risk management",
		}
	}

	profile := req.RiskProfile

	// 验证最大回撤
	if profile.MaxDrawdown <= 0 || profile.MaxDrawdown > 0.5 {
		return &ValidationError{
			Rule:       r.Name(),
			Severity:   "error",
			Message:    "Max drawdown is out of valid range (0-0.5)",
			Field:      "risk_profile.max_drawdown",
			Suggestion: "Set max drawdown between 0.05 (5%) and 0.5 (50%)",
		}
	}

	// 验证最大杠杆
	if profile.MaxLeverage <= 0 || profile.MaxLeverage > 100 {
		return &ValidationError{
			Rule:       r.Name(),
			Severity:   "error",
			Message:    "Max leverage is out of valid range (1-100)",
			Field:      "risk_profile.max_leverage",
			Suggestion: "Set max leverage between 1 and 100",
		}
	}

	// 验证风险等级
	validRiskLevels := []string{"low", "medium", "high"}
	isValidRiskLevel := false
	for _, level := range validRiskLevels {
		if profile.RiskLevel == level {
			isValidRiskLevel = true
			break
		}
	}

	if !isValidRiskLevel {
		return &ValidationError{
			Rule:       r.Name(),
			Severity:   "error",
			Message:    "Invalid risk level",
			Field:      "risk_profile.risk_level",
			Suggestion: "Use one of: low, medium, high",
		}
	}

	return nil
}

// CodeQualityRule 代码质量验证规则
type CodeQualityRule struct{}

func (r *CodeQualityRule) Name() string {
	return "code_quality"
}

func (r *CodeQualityRule) Validate(ctx context.Context, req *OnboardingRequest) *ValidationError {
	if req.StrategyCode == "" {
		return &ValidationError{
			Rule:       r.Name(),
			Severity:   "warning",
			Message:    "No strategy code provided",
			Field:      "strategy_code",
			Suggestion: "Provide strategy code for better validation",
		}
	}

	// 简单的代码质量检查
	code := strings.ToLower(req.StrategyCode)
	
	// 检查是否包含危险操作
	dangerousPatterns := []string{
		"os.system",
		"exec(",
		"eval(",
		"import os",
		"subprocess",
		"__import__",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(code, pattern) {
			return &ValidationError{
				Rule:       r.Name(),
				Severity:   "error",
				Message:    fmt.Sprintf("Potentially dangerous code pattern detected: %s", pattern),
				Field:      "strategy_code",
				Suggestion: "Remove dangerous system calls and imports",
			}
		}
	}

	// 检查代码长度
	if len(req.StrategyCode) > 100000 { // 100KB
		return &ValidationError{
			Rule:       r.Name(),
			Severity:   "warning",
			Message:    "Strategy code is very large",
			Field:      "strategy_code",
			Suggestion: "Consider breaking down into smaller modules",
		}
	}

	return nil
}

// PerformanceRule 性能验证规则
type PerformanceRule struct{}

func (r *PerformanceRule) Name() string {
	return "performance_validation"
}

func (r *PerformanceRule) Validate(ctx context.Context, req *OnboardingRequest) *ValidationError {
	// 这里可以添加性能相关的验证
	// 例如：检查策略的预期性能指标
	
	// 模拟性能检查
	if req.Config != nil && req.Config.Params != nil {
		if expectedReturn, exists := req.Config.Params["expected_return"]; exists {
			if er, ok := expectedReturn.(float64); ok {
				if er < -0.5 || er > 5.0 { // -50% to 500%
					return &ValidationError{
						Rule:       r.Name(),
						Severity:   "warning",
						Message:    "Expected return is outside reasonable range",
						Field:      "config.params.expected_return",
						Suggestion: "Review expected return calculations",
					}
				}
			}
		}
	}

	return nil
}

// SecurityRule 安全验证规则
type SecurityRule struct{}

func (r *SecurityRule) Name() string {
	return "security_validation"
}

func (r *SecurityRule) Validate(ctx context.Context, req *OnboardingRequest) *ValidationError {
	// 检查策略ID格式
	if req.StrategyID == "" {
		return &ValidationError{
			Rule:       r.Name(),
			Severity:   "error",
			Message:    "Strategy ID is required",
			Field:      "strategy_id",
			Suggestion: "Provide a unique strategy identifier",
		}
	}

	// 检查ID格式（简单验证）
	idPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !idPattern.MatchString(req.StrategyID) {
		return &ValidationError{
			Rule:       r.Name(),
			Severity:   "error",
			Message:    "Strategy ID contains invalid characters",
			Field:      "strategy_id",
			Suggestion: "Use only letters, numbers, underscores, and hyphens",
		}
	}

	return nil
}
