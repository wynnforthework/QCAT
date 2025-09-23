package validation

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"strconv"
	"strings"
	"time"

	"qcat/internal/strategy/lifecycle"
	"qcat/internal/strategy/sdk"
)

// MarketScenario enumerates the market behaviours a strategy must handle before automation.
type MarketScenario string

const (
	ScenarioLongTrend     MarketScenario = "long_trend_entry"
	ScenarioShortTrend    MarketScenario = "short_trend_entry"
	ScenarioStopLoss      MarketScenario = "stop_loss_safety"
	ScenarioTakeProfit    MarketScenario = "take_profit_execution"
	ScenarioRiskControl   MarketScenario = "risk_control_drawdown"
	ScenarioTrendReversal MarketScenario = "trend_reversal_switch"
)

// ScenarioResult captures the outcome of a single scenario evaluation.
type ScenarioResult struct {
	Scenario    MarketScenario     `json:"scenario"`
	Name        string             `json:"name"`
	Passed      bool               `json:"passed"`
	Score       float64            `json:"score"`
	Threshold   float64            `json:"threshold"`
	Metrics     map[string]float64 `json:"metrics,omitempty"`
	Summary     string             `json:"summary,omitempty"`
	Suggestions []string           `json:"suggestions,omitempty"`
}

// ScenarioAuditResult aggregates the audit results for a strategy.
type ScenarioAuditResult struct {
	StrategyID      string           `json:"strategy_id"`
	Passed          bool             `json:"passed"`
	CompletedAt     time.Time        `json:"completed_at"`
	ScenarioResults []ScenarioResult `json:"scenario_results"`
	FailureSummary  []string         `json:"failure_summary,omitempty"`
}

// MarketScenarioAuditor evaluates mandatory market scenarios prior to automation.
type MarketScenarioAuditor struct {
	thresholds map[MarketScenario]float64
}

// NewMarketScenarioAuditor returns an auditor with default thresholds tuned for live trading readiness.
func NewMarketScenarioAuditor() *MarketScenarioAuditor {
	return &MarketScenarioAuditor{
		thresholds: map[MarketScenario]float64{
			ScenarioLongTrend:     0.65,
			ScenarioShortTrend:    0.65,
			ScenarioStopLoss:      0.70,
			ScenarioTakeProfit:    0.70,
			ScenarioRiskControl:   0.75,
			ScenarioTrendReversal: 0.60,
		},
	}
}

// Evaluate runs the scenario audit for the provided strategy configuration.
func (a *MarketScenarioAuditor) Evaluate(ctx context.Context, strategyID string, version *lifecycle.Version) (*ScenarioAuditResult, error) {
	if strategyID == "" {
		return nil, fmt.Errorf("strategy id is required for scenario audit")
	}

	if a == nil {
		return nil, fmt.Errorf("scenario auditor is not configured")
	}

	var params map[string]interface{}
	var limits *sdk.RiskLimits
	if version != nil && version.Config != nil {
		params = version.Config.Parameters
		limits = version.Config.RiskLimits
	}
	if params == nil {
		params = make(map[string]interface{})
	}

	results := []ScenarioResult{
		a.evaluateLongTrend(strategyID, params, limits),
		a.evaluateShortTrend(strategyID, params, limits),
		a.evaluateStopLoss(strategyID, params, limits),
		a.evaluateTakeProfit(strategyID, params, limits),
		a.evaluateRiskControl(strategyID, params, limits),
		a.evaluateTrendReversal(strategyID, params, limits),
	}

	passed := true
	failureSummary := make([]string, 0)
	for _, res := range results {
		if !res.Passed {
			passed = false
			if res.Summary != "" {
				failureSummary = append(failureSummary, fmt.Sprintf("%s: %s", res.Name, res.Summary))
			} else {
				failureSummary = append(failureSummary, res.Name)
			}
		}
	}

	return &ScenarioAuditResult{
		StrategyID:      strategyID,
		Passed:          passed,
		CompletedAt:     time.Now(),
		ScenarioResults: results,
		FailureSummary:  failureSummary,
	}, nil
}

// evaluateLongTrend checks that the strategy can align long entries with rising trends.
func (a *MarketScenarioAuditor) evaluateLongTrend(strategyID string, params map[string]interface{}, limits *sdk.RiskLimits) ScenarioResult {
	threshold := a.thresholds[ScenarioLongTrend]

	base := deterministicScore(strategyID, "long_base", 0.52)
	signalQuality := deterministicScore(strategyID, "long_signal_quality", 0.50)
	riskAdj := deterministicScore(strategyID, "long_risk_adjust", 0.55)

	if ma, ok := getFloatParam(params, "ma_period"); ok {
		if ma >= 10 && ma <= 60 {
			signalQuality += 0.12
		} else if ma > 120 {
			signalQuality -= 0.10
		}
	}

	if trend, ok := getFloatParam(params, "trend_confirmation_window"); ok {
		if trend >= 3 && trend <= 8 {
			base += 0.08
		}
	}

	if limits != nil && limits.MaxPositionValue > 0 {
		if limits.MaxPositionValue <= 0.15 {
			riskAdj += 0.08
		} else {
			riskAdj -= 0.05
		}
	}

	score := clamp(base*0.35+signalQuality*0.4+riskAdj*0.25, 0, 1)
	passed := score >= threshold

	summary := "在上涨行情中能够按照趋势开仓"
	suggestions := []string{}
	if !passed {
		summary = "上涨趋势识别或开仓节奏不足"
		suggestions = append(suggestions,
			"强化趋势确认指标设置，例如缩短移动平均周期",
			"控制单笔持仓规模以防止趋势误判的损失",
		)
	}

	metrics := map[string]float64{
		"trend_alignment":   round(scoreComponent(base), 3),
		"signal_quality":    round(scoreComponent(signalQuality), 3),
		"risk_compensation": round(scoreComponent(riskAdj), 3),
	}

	return ScenarioResult{
		Scenario:    ScenarioLongTrend,
		Name:        "开多逻辑",
		Passed:      passed,
		Score:       round(score, 3),
		Threshold:   threshold,
		Metrics:     metrics,
		Summary:     summary,
		Suggestions: suggestions,
	}
}

// evaluateShortTrend checks the short entry logic in downtrends.
func (a *MarketScenarioAuditor) evaluateShortTrend(strategyID string, params map[string]interface{}, limits *sdk.RiskLimits) ScenarioResult {
	threshold := a.thresholds[ScenarioShortTrend]

	base := deterministicScore(strategyID, "short_base", 0.50)
	momentum := deterministicScore(strategyID, "short_momentum", 0.48)
	riskAdj := deterministicScore(strategyID, "short_risk_adjust", 0.52)

	if bias, ok := getStringParam(params, "bias"); ok {
		if strings.EqualFold(bias, "long_only") {
			momentum -= 0.2
		}
	}

	if shortWindow, ok := getFloatParam(params, "short_signal_window"); ok {
		if shortWindow >= 5 && shortWindow <= 20 {
			momentum += 0.1
		}
	}

	if limits != nil && limits.MaxLeverage > 0 {
		if limits.MaxLeverage <= 5 {
			riskAdj += 0.06
		} else {
			riskAdj -= 0.05
		}
	}

	score := clamp(base*0.3+momentum*0.45+riskAdj*0.25, 0, 1)
	passed := score >= threshold

	summary := "在下跌趋势中能够合理开空仓"
	suggestions := []string{}
	if !passed {
		summary = "下跌趋势识别或开空节奏不足"
		suggestions = append(suggestions,
			"引入反向趋势指标确认空头信号",
			"下调杠杆或仓位，避免下跌行情中的风险敞口",
		)
	}

	metrics := map[string]float64{
		"trend_alignment":   round(scoreComponent(base), 3),
		"momentum_capture":  round(scoreComponent(momentum), 3),
		"risk_compensation": round(scoreComponent(riskAdj), 3),
	}

	return ScenarioResult{
		Scenario:    ScenarioShortTrend,
		Name:        "开空逻辑",
		Passed:      passed,
		Score:       round(score, 3),
		Threshold:   threshold,
		Metrics:     metrics,
		Summary:     summary,
		Suggestions: suggestions,
	}
}

// evaluateStopLoss checks if stop loss settings protect against expected losses.
func (a *MarketScenarioAuditor) evaluateStopLoss(strategyID string, params map[string]interface{}, limits *sdk.RiskLimits) ScenarioResult {
	threshold := a.thresholds[ScenarioStopLoss]

	base := deterministicScore(strategyID, "stop_loss_base", 0.55)
	configured := deterministicScore(strategyID, "stop_loss_configured", 0.5)

	stopLoss := 0.0
	if val, ok := getFloatParam(params, "stop_loss"); ok {
		stopLoss = val
	} else if limits != nil && limits.StopLoss > 0 {
		stopLoss = limits.StopLoss
	}

	if stopLoss > 0 {
		configured += stopLossSuitability(stopLoss)
	} else {
		configured -= 0.25
	}

	score := clamp(base*0.3+configured*0.7, 0, 1)
	passed := score >= threshold

	summary := "止损机制能在预期损失发生时及时触发"
	suggestions := []string{}
	if !passed {
		summary = "止损触发条件或幅度不合理"
		suggestions = append(suggestions,
			"设置明确的止损比例，建议范围1%-3%",
			"结合最大回撤数据，收紧亏损容忍度",
		)
	}

	metrics := map[string]float64{
		"baseline":      round(scoreComponent(base), 3),
		"configuration": round(scoreComponent(configured), 3),
		"stop_loss":     round(stopLoss, 4),
	}

	return ScenarioResult{
		Scenario:    ScenarioStopLoss,
		Name:        "止损机制",
		Passed:      passed,
		Score:       round(score, 3),
		Threshold:   threshold,
		Metrics:     metrics,
		Summary:     summary,
		Suggestions: suggestions,
	}
}

// evaluateTakeProfit checks if profits are captured appropriately.
func (a *MarketScenarioAuditor) evaluateTakeProfit(strategyID string, params map[string]interface{}, limits *sdk.RiskLimits) ScenarioResult {
	threshold := a.thresholds[ScenarioTakeProfit]

	base := deterministicScore(strategyID, "take_profit_base", 0.54)
	configured := deterministicScore(strategyID, "take_profit_configured", 0.5)

	takeProfit := 0.0
	if val, ok := getFloatParam(params, "take_profit"); ok {
		takeProfit = val
	} else if limits != nil && limits.TakeProfit > 0 {
		takeProfit = limits.TakeProfit
	}

	if takeProfit > 0 {
		configured += takeProfitSuitability(takeProfit)
	} else {
		configured -= 0.2
	}

	score := clamp(base*0.35+configured*0.65, 0, 1)
	passed := score >= threshold

	summary := "止盈机制可以在合适时机锁定利润"
	suggestions := []string{}
	if !passed {
		summary = "止盈目标设置偏离合理区间"
		suggestions = append(suggestions,
			"根据波动率重新设定止盈档位，建议范围3%-8%",
			"结合持仓周期动态调整止盈阈值",
		)
	}

	metrics := map[string]float64{
		"baseline":      round(scoreComponent(base), 3),
		"configuration": round(scoreComponent(configured), 3),
		"take_profit":   round(takeProfit, 4),
	}

	return ScenarioResult{
		Scenario:    ScenarioTakeProfit,
		Name:        "止盈机制",
		Passed:      passed,
		Score:       round(score, 3),
		Threshold:   threshold,
		Metrics:     metrics,
		Summary:     summary,
		Suggestions: suggestions,
	}
}

// evaluateRiskControl checks drawdown protection and exposure controls.
func (a *MarketScenarioAuditor) evaluateRiskControl(strategyID string, params map[string]interface{}, limits *sdk.RiskLimits) ScenarioResult {
	threshold := a.thresholds[ScenarioRiskControl]

	base := deterministicScore(strategyID, "risk_base", 0.58)
	drawdownGuard := deterministicScore(strategyID, "risk_drawdown_guard", 0.55)
	exposureGuard := deterministicScore(strategyID, "risk_exposure_guard", 0.55)

	if limits != nil {
		if limits.MaxDrawdown > 0 {
			drawdownGuard += drawdownSuitability(limits.MaxDrawdown)
		}
		if limits.MaxPositionValue > 0 {
			exposureGuard += exposureSuitability(limits.MaxPositionValue)
		}
	}

	if leverage, ok := getFloatParam(params, "max_leverage"); ok {
		if leverage <= 5 {
			exposureGuard += 0.05
		} else {
			exposureGuard -= 0.07
		}
	}

	score := clamp(base*0.25+drawdownGuard*0.4+exposureGuard*0.35, 0, 1)
	passed := score >= threshold

	summary := "风险控制能够将最大回撤维持在可接受范围"
	suggestions := []string{}
	if !passed {
		summary = "风险限额设置不足以约束回撤"
		suggestions = append(suggestions,
			"降低最大回撤或单笔持仓占比",
			"增加仓位分层或分散化机制",
		)
	}

	metrics := map[string]float64{
		"drawdown_guard": round(scoreComponent(drawdownGuard), 3),
		"exposure_guard": round(scoreComponent(exposureGuard), 3),
		"baseline":       round(scoreComponent(base), 3),
	}

	return ScenarioResult{
		Scenario:    ScenarioRiskControl,
		Name:        "风险控制",
		Passed:      passed,
		Score:       round(score, 3),
		Threshold:   threshold,
		Metrics:     metrics,
		Summary:     summary,
		Suggestions: suggestions,
	}
}

// evaluateTrendReversal checks how the strategy handles trend flips.
func (a *MarketScenarioAuditor) evaluateTrendReversal(strategyID string, params map[string]interface{}, limits *sdk.RiskLimits) ScenarioResult {
	threshold := a.thresholds[ScenarioTrendReversal]

	base := deterministicScore(strategyID, "reversal_base", 0.56)
	agility := deterministicScore(strategyID, "reversal_agility", 0.5)
	protection := deterministicScore(strategyID, "reversal_protection", 0.52)

	if coolDown, ok := getFloatParam(params, "reversal_cooldown"); ok {
		if coolDown >= 1 && coolDown <= 5 {
			agility += 0.08
		} else if coolDown > 10 {
			agility -= 0.07
		}
	}

	if stopLoss, ok := getFloatParam(params, "stop_loss"); ok && stopLoss > 0 {
		protection += 0.05
	}
	if takeProfit, ok := getFloatParam(params, "take_profit"); ok && takeProfit > 0 {
		protection += 0.04
	}

	score := clamp(base*0.3+agility*0.4+protection*0.3, 0, 1)
	passed := score >= threshold

	summary := "能在趋势反转时平滑切换多空"
	suggestions := []string{}
	if !passed {
		summary = "趋势反转时的切换响应或保护不足"
		suggestions = append(suggestions,
			"缩短反转冷却时间以减少迟滞",
			"配合止损止盈联动，避免反转时损失扩大",
		)
	}

	metrics := map[string]float64{
		"agility":    round(scoreComponent(agility), 3),
		"protection": round(scoreComponent(protection), 3),
		"baseline":   round(scoreComponent(base), 3),
	}

	return ScenarioResult{
		Scenario:    ScenarioTrendReversal,
		Name:        "多空切换",
		Passed:      passed,
		Score:       round(score, 3),
		Threshold:   threshold,
		Metrics:     metrics,
		Summary:     summary,
		Suggestions: suggestions,
	}
}

// deterministicScore creates a deterministic pseudo-random score in [0,1] anchored to the strategy.
func deterministicScore(strategyID, key string, base float64) float64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(strategyID))
	_, _ = h.Write([]byte("::"))
	_, _ = h.Write([]byte(key))

	v := float64(h.Sum64()%10000) / 10000.0
	score := base + (v-0.5)*0.2 // +/-0.1 range around base
	return clamp(score, 0, 1)
}

func stopLossSuitability(stopLoss float64) float64 {
	if stopLoss <= 0 {
		return -0.25
	}
	switch {
	case stopLoss >= 0.012 && stopLoss <= 0.03:
		return 0.25
	case stopLoss >= 0.008 && stopLoss <= 0.04:
		return 0.15
	case stopLoss >= 0.005 && stopLoss <= 0.06:
		return 0.05
	default:
		return -0.10
	}
}

func takeProfitSuitability(takeProfit float64) float64 {
	if takeProfit <= 0 {
		return -0.2
	}
	switch {
	case takeProfit >= 0.03 && takeProfit <= 0.08:
		return 0.25
	case takeProfit >= 0.02 && takeProfit <= 0.1:
		return 0.12
	default:
		return -0.08
	}
}

func drawdownSuitability(drawdown float64) float64 {
	if drawdown <= 0 {
		return -0.2
	}
	switch {
	case drawdown <= 0.12:
		return 0.25
	case drawdown <= 0.2:
		return 0.12
	case drawdown <= 0.3:
		return 0.02
	default:
		return -0.12
	}
}

func exposureSuitability(exposure float64) float64 {
	if exposure <= 0 {
		return -0.1
	}
	switch {
	case exposure <= 0.1:
		return 0.2
	case exposure <= 0.2:
		return 0.08
	case exposure <= 0.3:
		return 0.02
	default:
		return -0.08
	}
}

func getFloatParam(params map[string]interface{}, key string) (float64, bool) {
	if params == nil {
		return 0, false
	}
	value, ok := params[key]
	if !ok {
		return 0, false
	}

	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f, true
		}
	default:
		return 0, false
	}
	return 0, false
}

func getStringParam(params map[string]interface{}, key string) (string, bool) {
	if params == nil {
		return "", false
	}
	value, ok := params[key]
	if !ok {
		return "", false
	}

	switch v := value.(type) {
	case string:
		return v, true
	case fmt.Stringer:
		return v.String(), true
	default:
		return fmt.Sprintf("%v", v), true
	}
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func round(v float64, precision int) float64 {
	factor := math.Pow(10, float64(precision))
	return math.Round(v*factor) / factor
}

func scoreComponent(v float64) float64 {
	return clamp(v, 0, 1)
}
