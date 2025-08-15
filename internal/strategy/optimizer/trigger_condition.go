package optimizer

import (
	"context"
	"fmt"
	"time"

	"qcat/internal/strategy/sdk"
)

// TriggerCondition represents optimization trigger conditions
type TriggerCondition struct {
	MinSharpe     float64       // Sharpe比率最小阈值
	MaxDrawdown   float64       // 最大回撤阈值
	NoNewHighDays int           // 无新高天数阈值
	ReturnRankQ   float64       // 收益分位数阈值
	CheckInterval time.Duration // 检查间隔
}

// TriggerResult represents trigger check result
type TriggerResult struct {
	Triggered     bool
	Reason        string
	Metrics       *sdk.StrategyMetrics
	LastCheckTime time.Time
}

// TriggerChecker checks optimization trigger conditions
type TriggerChecker struct {
	condition *TriggerCondition
	strategy  sdk.Strategy
}

// NewTriggerChecker creates a new trigger checker
func NewTriggerChecker(condition *TriggerCondition, strategy sdk.Strategy) *TriggerChecker {
	return &TriggerChecker{
		condition: condition,
		strategy:  strategy,
	}
}

// CheckTriggers checks if optimization should be triggered
func (c *TriggerChecker) CheckTriggers(ctx context.Context) (*TriggerResult, error) {
	metrics := c.strategy.GetResult().Metrics
	result := &TriggerResult{
		Metrics:       metrics,
		LastCheckTime: time.Now(),
	}

	// 检查Sharpe比率
	if metrics.SharpeRatio < c.condition.MinSharpe {
		result.Triggered = true
		result.Reason = fmt.Sprintf("Low Sharpe ratio: %.2f < %.2f", metrics.SharpeRatio, c.condition.MinSharpe)
		return result, nil
	}

	// 检查最大回撤
	if metrics.MaxDrawdown > c.condition.MaxDrawdown {
		result.Triggered = true
		result.Reason = fmt.Sprintf("High drawdown: %.2f%% > %.2f%%", metrics.MaxDrawdown*100, c.condition.MaxDrawdown*100)
		return result, nil
	}

	// 检查无新高天数
	if metrics.DaysWithoutNewHigh >= c.condition.NoNewHighDays {
		result.Triggered = true
		result.Reason = fmt.Sprintf("No new high for %d days", metrics.DaysWithoutNewHigh)
		return result, nil
	}

	// 检查收益分位数
	if metrics.ReturnRank < c.condition.ReturnRankQ {
		result.Triggered = true
		result.Reason = fmt.Sprintf("Low return rank: %.2f < %.2f", metrics.ReturnRank, c.condition.ReturnRankQ)
		return result, nil
	}

	return result, nil
}

// ScheduleTrigger schedules periodic optimization checks
func (c *TriggerChecker) ScheduleTrigger(ctx context.Context) (<-chan time.Time, error) {
	if c.condition.CheckInterval <= 0 {
		return nil, fmt.Errorf("invalid check interval")
	}

	// 创建定时触发器，每天UTC 00:10触发
	now := time.Now().UTC()
	nextCheck := time.Date(now.Year(), now.Month(), now.Day(), 0, 10, 0, 0, time.UTC)
	if now.After(nextCheck) {
		nextCheck = nextCheck.Add(24 * time.Hour)
	}

	ch := make(chan time.Time)
	go func() {
		timer := time.NewTimer(nextCheck.Sub(now))
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				close(ch)
				return
			case t := <-timer.C:
				ch <- t
				timer.Reset(24 * time.Hour)
			}
		}
	}()

	return ch, nil
}
