package validation

import (
	"context"
	"fmt"
	"log"
	"time"

	"qcat/internal/strategy/backtest"
	"qcat/internal/strategy/lifecycle"
)

// MandatoryBacktestValidator 强制回测验证器
type MandatoryBacktestValidator struct {
	backtestEngine *backtest.Engine
	minBacktestDays int
	minSharpeRatio  float64
	maxDrawdown     float64
	minWinRate      float64
}

// BacktestRequirement 回测要求
type BacktestRequirement struct {
	MinBacktestDays int     `json:"min_backtest_days"` // 最少回测天数
	MinSharpeRatio  float64 `json:"min_sharpe_ratio"`  // 最小夏普比率
	MaxDrawdown     float64 `json:"max_drawdown"`      // 最大回撤限制
	MinWinRate      float64 `json:"min_win_rate"`      // 最小胜率
	MinTotalReturn  float64 `json:"min_total_return"`  // 最小总收益率
}

// BacktestResult 回测结果
type BacktestResult struct {
	TotalReturn    float64 `json:"total_return"`
	SharpeRatio    float64 `json:"sharpe_ratio"`
	MaxDrawdown    float64 `json:"max_drawdown"`
	WinRate        float64 `json:"win_rate"`
	TotalTrades    int     `json:"total_trades"`
	ProfitFactor   float64 `json:"profit_factor"`
	BacktestDays   int     `json:"backtest_days"`
	StartDate      time.Time `json:"start_date"`
	EndDate        time.Time `json:"end_date"`
	IsValid        bool    `json:"is_valid"`
	FailureReasons []string `json:"failure_reasons"`
}

// ValidationError 验证错误
type ValidationError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field"`
}

// NewMandatoryBacktestValidator 创建强制回测验证器
func NewMandatoryBacktestValidator() *MandatoryBacktestValidator {
	return &MandatoryBacktestValidator{
		minBacktestDays: 365, // 至少1年历史数据
		minSharpeRatio:  0.5, // 最小夏普比率0.5
		maxDrawdown:     0.2, // 最大回撤20%
		minWinRate:      0.4, // 最小胜率40%
	}
}

// ValidateStrategy 验证策略必须通过回测
func (v *MandatoryBacktestValidator) ValidateStrategy(ctx context.Context, strategyID string, config *lifecycle.Version) (*BacktestResult, error) {
	log.Printf("开始强制回测验证策略: %s", strategyID)

	// 1. 检查策略是否已有有效的回测结果
	existingResult, err := v.getExistingBacktestResult(ctx, strategyID)
	if err == nil && existingResult.IsValid {
		log.Printf("策略 %s 已有有效回测结果，跳过重复回测", strategyID)
		return existingResult, nil
	}

	// 2. 执行强制回测
	result, err := v.runMandatoryBacktest(ctx, strategyID, config)
	if err != nil {
		return nil, fmt.Errorf("强制回测失败: %w", err)
	}

	// 3. 验证回测结果是否满足要求
	if err := v.validateBacktestResult(result); err != nil {
		result.IsValid = false
		result.FailureReasons = append(result.FailureReasons, err.Error())
		return result, fmt.Errorf("策略未通过回测验证: %w", err)
	}

	result.IsValid = true
	log.Printf("策略 %s 通过强制回测验证", strategyID)

	// 4. 保存回测结果
	if err := v.saveBacktestResult(ctx, strategyID, result); err != nil {
		log.Printf("保存回测结果失败: %v", err)
	}

	return result, nil
}

// runMandatoryBacktest 运行强制回测
func (v *MandatoryBacktestValidator) runMandatoryBacktest(ctx context.Context, strategyID string, config *lifecycle.Version) (*BacktestResult, error) {
	// 设置回测时间范围（最近2年数据）
	endDate := time.Now()
	startDate := endDate.AddDate(-2, 0, 0) // 2年前

	log.Printf("策略 %s 回测时间范围: %s 到 %s", strategyID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	// 模拟回测执行（实际应该调用真实的回测引擎）
	// 这里为了演示，我们生成一些模拟数据
	result := &BacktestResult{
		StartDate:    startDate,
		EndDate:      endDate,
		BacktestDays: int(endDate.Sub(startDate).Hours() / 24),
	}

	// 模拟回测计算（实际应该基于真实历史数据）
	// 注意：这里应该替换为真实的回测逻辑
	result.TotalReturn = -0.15    // -15% (故意设置为负数以演示验证失败)
	result.SharpeRatio = 0.3      // 0.3 (低于要求)
	result.MaxDrawdown = 0.25     // 25% (超过限制)
	result.WinRate = 0.35         // 35% (低于要求)
	result.TotalTrades = 1200     // 大量交易
	result.ProfitFactor = 0.8     // 盈亏比小于1

	return result, nil
}

// validateBacktestResult 验证回测结果
func (v *MandatoryBacktestValidator) validateBacktestResult(result *BacktestResult) error {
	var errors []string

	// 检查回测天数
	if result.BacktestDays < v.minBacktestDays {
		errors = append(errors, fmt.Sprintf("回测天数不足: %d天 < %d天", result.BacktestDays, v.minBacktestDays))
	}

	// 检查总收益率
	if result.TotalReturn < 0 {
		errors = append(errors, fmt.Sprintf("总收益率为负: %.2f%%", result.TotalReturn*100))
	}

	// 检查夏普比率
	if result.SharpeRatio < v.minSharpeRatio {
		errors = append(errors, fmt.Sprintf("夏普比率过低: %.2f < %.2f", result.SharpeRatio, v.minSharpeRatio))
	}

	// 检查最大回撤
	if result.MaxDrawdown > v.maxDrawdown {
		errors = append(errors, fmt.Sprintf("最大回撤过大: %.2f%% > %.2f%%", result.MaxDrawdown*100, v.maxDrawdown*100))
	}

	// 检查胜率
	if result.WinRate < v.minWinRate {
		errors = append(errors, fmt.Sprintf("胜率过低: %.2f%% < %.2f%%", result.WinRate*100, v.minWinRate*100))
	}

	// 检查交易频率（防止过度交易）
	if result.TotalTrades > result.BacktestDays*10 { // 平均每天超过10笔交易
		errors = append(errors, fmt.Sprintf("交易频率过高: %d笔/%d天", result.TotalTrades, result.BacktestDays))
	}

	if len(errors) > 0 {
		result.FailureReasons = errors
		return fmt.Errorf("回测验证失败: %v", errors)
	}

	return nil
}

// getExistingBacktestResult 获取现有回测结果
func (v *MandatoryBacktestValidator) getExistingBacktestResult(ctx context.Context, strategyID string) (*BacktestResult, error) {
	// 这里应该从数据库查询现有的回测结果
	// 暂时返回错误表示没有现有结果
	return nil, fmt.Errorf("no existing backtest result")
}

// saveBacktestResult 保存回测结果
func (v *MandatoryBacktestValidator) saveBacktestResult(ctx context.Context, strategyID string, result *BacktestResult) error {
	// 这里应该将回测结果保存到数据库
	log.Printf("保存策略 %s 的回测结果: 收益率=%.2f%%, 夏普比率=%.2f, 最大回撤=%.2f%%", 
		strategyID, result.TotalReturn*100, result.SharpeRatio, result.MaxDrawdown*100)
	return nil
}

// GetDefaultRequirements 获取默认回测要求
func GetDefaultRequirements() *BacktestRequirement {
	return &BacktestRequirement{
		MinBacktestDays: 365,  // 至少1年
		MinSharpeRatio:  0.5,  // 夏普比率至少0.5
		MaxDrawdown:     0.2,  // 最大回撤不超过20%
		MinWinRate:      0.4,  // 胜率至少40%
		MinTotalReturn:  0.05, // 总收益率至少5%
	}
}
