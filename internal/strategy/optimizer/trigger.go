package optimizer

import (
	"context"
	"fmt"
	"time"
)

// TriggerChecker checks optimization trigger conditions
type TriggerChecker struct {
	config *TriggerConfig
}

// TriggerConfig represents trigger configuration
type TriggerConfig struct {
	MinSharpe     float64       // Sharpe比率阈值
	MaxDrawdown   float64       // 最大回撤阈值
	NoNewHighDays int           // 无新高天数
	ReturnRank    float64       // 收益分位数
	CheckInterval time.Duration // 检查间隔
}

// TriggerResult represents trigger check result
type TriggerResult struct {
	Triggered     bool
	Reason        string
	Metrics       *PerformanceMetrics
	LastCheckTime time.Time
}

// NewTriggerChecker creates a new trigger checker
func NewTriggerChecker(config *TriggerConfig) *TriggerChecker {
	return &TriggerChecker{
		config: config,
	}
}

// CheckTriggers checks if optimization should be triggered
func (c *TriggerChecker) CheckTriggers(ctx context.Context, metrics *PerformanceMetrics, returns []float64) (*TriggerResult, error) {
	result := &TriggerResult{
		Metrics:       metrics,
		LastCheckTime: time.Now(),
	}

	// 检查Sharpe比率
	if metrics.SharpeRatio < c.config.MinSharpe {
		result.Triggered = true
		result.Reason = fmt.Sprintf("Low Sharpe ratio: %.2f < %.2f", metrics.SharpeRatio, c.config.MinSharpe)
		return result, nil
	}

	// 检查最大回撤
	if metrics.MaxDrawdown > c.config.MaxDrawdown {
		result.Triggered = true
		result.Reason = fmt.Sprintf("High drawdown: %.2f%% > %.2f%%", metrics.MaxDrawdown*100, c.config.MaxDrawdown*100)
		return result, nil
	}

	// 检查无新高天数
	highestEquity := 0.0
	daysWithoutNewHigh := 0
	equity := 1.0

	for _, r := range returns {
		equity *= (1 + r)
		if equity > highestEquity {
			highestEquity = equity
			daysWithoutNewHigh = 0
		} else {
			daysWithoutNewHigh++
		}
	}

	if daysWithoutNewHigh >= c.config.NoNewHighDays {
		result.Triggered = true
		result.Reason = fmt.Sprintf("No new high for %d days", daysWithoutNewHigh)
		return result, nil
	}

	// 检查收益分位数
	if len(returns) > 0 {
		rank := calculateReturnRank(returns[len(returns)-1], returns)
		if rank < c.config.ReturnRank {
			result.Triggered = true
			result.Reason = fmt.Sprintf("Low return rank: %.2f < %.2f", rank, c.config.ReturnRank)
			return result, nil
		}
	}

	return result, nil
}

// calculateReturnRank calculates the percentile rank of a return
func calculateReturnRank(value float64, returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	count := 0
	for _, r := range returns {
		if r <= value {
			count++
		}
	}

	return float64(count) / float64(len(returns))
}

// ScheduleTrigger schedules periodic optimization checks
func (c *TriggerChecker) ScheduleTrigger(ctx context.Context) (<-chan time.Time, error) {
	if c.config.CheckInterval <= 0 {
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
