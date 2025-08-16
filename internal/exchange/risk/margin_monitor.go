package risk

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/cache"
	exch "qcat/internal/exchange"
)

// MarginMonitor monitors margin ratios and triggers alerts
type MarginMonitor struct {
	exchange   exch.Exchange
	cache      cache.Cacher
	alertCh    chan *exch.MarginAlert
	thresholds map[exch.MarginLevel]float64
	mu         sync.RWMutex
}

// NewMarginMonitor creates a new margin monitor
func NewMarginMonitor(ex exch.Exchange, cache cache.Cacher) *MarginMonitor {
	m := &MarginMonitor{
		exchange:   ex,
		cache:      cache,
		alertCh:    make(chan *exch.MarginAlert, 100),
		thresholds: make(map[exch.MarginLevel]float64),
	}

	// 设置默认阈值
	m.thresholds[exch.MarginLevelWarning] = 1.5     // 150% - 警告线
	m.thresholds[exch.MarginLevelDanger] = 1.1      // 110% - 危险线
	m.thresholds[exch.MarginLevelLiquidation] = 1.0 // 100% - 强平线

	// 启动监控
	go m.monitor()

	return m
}

// SetThreshold sets the margin threshold for a specific level
func (m *MarginMonitor) SetThreshold(level exch.MarginLevel, ratio float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.thresholds[level] = ratio
}

// GetAlertChannel returns the alert channel
func (m *MarginMonitor) GetAlertChannel() <-chan *exch.MarginAlert {
	return m.alertCh
}

// CheckMarginRatio checks current margin ratio and returns margin level
func (m *MarginMonitor) CheckMarginRatio(ctx context.Context) (*exch.MarginInfo, exch.MarginLevel, error) {
	// 从缓存获取保证金信息
	var info *exch.MarginInfo
	if err := m.cache.Get(ctx, "margin:info", &info); err == nil && info != nil {
		// 缓存命中，直接返回
		level := m.determineMarginLevel(info.MarginRatio)
		return info, level, nil
	}

	// 缓存未命中，从交易所获取保证金信息
	info, err := m.exchange.GetMarginInfo(ctx)
	if err != nil {
		// 如果获取失败，使用默认值
		log.Printf("Failed to get margin info from exchange: %v, using default values", err)
		info = &exch.MarginInfo{
			TotalAssetValue:   100000.0,
			TotalDebtValue:    50000.0,
			MarginRatio:       2.0,
			MaintenanceMargin: 1.1,
			MarginCallRatio:   1.5,
			LiquidationRatio:  1.0,
			UpdatedAt:         time.Now(),
		}
	}

	// 缓存保证金信息
	if err := m.cache.Set(ctx, "margin:info", info, time.Minute); err != nil {
		log.Printf("Failed to cache margin info: %v", err)
	}

	// 确定保证金等级
	level := m.determineMarginLevel(info.MarginRatio)

	return info, level, nil
}

// determineMarginLevel determines margin level based on current ratio
func (m *MarginMonitor) determineMarginLevel(ratio float64) exch.MarginLevel {
	m.mu.RLock()
	defer m.mu.RUnlock()

	switch {
	case ratio <= m.thresholds[exch.MarginLevelLiquidation]:
		return exch.MarginLevelLiquidation
	case ratio <= m.thresholds[exch.MarginLevelDanger]:
		return exch.MarginLevelDanger
	case ratio <= m.thresholds[exch.MarginLevelWarning]:
		return exch.MarginLevelWarning
	default:
		return exch.MarginLevelSafe
	}
}

// monitor periodically checks margin ratio and sends alerts
func (m *MarginMonitor) monitor() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		info, level, err := m.CheckMarginRatio(ctx)
		if err != nil {
			log.Printf("Failed to check margin ratio: %v", err)
			continue
		}

		// 如果不是安全等级，发送告警
		if level != exch.MarginLevelSafe {
			alert := &exch.MarginAlert{
				Level:     level,
				Ratio:     info.MarginRatio,
				Threshold: m.thresholds[level],
				Message:   fmt.Sprintf("Margin ratio %.2f%% is below %.2f%% threshold", info.MarginRatio*100, m.thresholds[level]*100),
				CreatedAt: time.Now(),
			}

			select {
			case m.alertCh <- alert:
			default:
				log.Printf("Alert channel is full, dropped alert: %v", alert)
			}
		}
	}
}
