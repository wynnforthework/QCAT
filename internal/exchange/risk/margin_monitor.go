package risk

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/cache"
	"qcat/internal/exchange"
)

// MarginMonitor monitors margin ratios and triggers alerts
type MarginMonitor struct {
	exchange   exchange.Exchange
	cache      cache.Cacher
	alertCh    chan *exchange.MarginAlert
	thresholds map[exchange.MarginLevel]float64
	mu         sync.RWMutex
}

// NewMarginMonitor creates a new margin monitor
func NewMarginMonitor(ex exchange.Exchange, cache cache.Cacher) *MarginMonitor {
	m := &MarginMonitor{
		exchange:   ex,
		cache:      cache,
		alertCh:    make(chan *exchange.MarginAlert, 100),
		thresholds: make(map[exchange.MarginLevel]float64),
	}

	// 设置默认阈值
	m.thresholds[exchange.MarginLevelWarning] = 1.5     // 150% - 警告线
	m.thresholds[exchange.MarginLevelDanger] = 1.1      // 110% - 危险线
	m.thresholds[exchange.MarginLevelLiquidation] = 1.0 // 100% - 强平线

	// 启动监控
	go m.monitor()

	return m
}

// SetThreshold sets the margin threshold for a specific level
func (m *MarginMonitor) SetThreshold(level exchange.MarginLevel, ratio float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.thresholds[level] = ratio
}

// GetAlertChannel returns the alert channel
func (m *MarginMonitor) GetAlertChannel() <-chan *exchange.MarginAlert {
	return m.alertCh
}

// CheckMarginRatio checks current margin ratio and returns margin level
func (m *MarginMonitor) CheckMarginRatio(ctx context.Context) (*exchange.MarginInfo, exchange.MarginLevel, error) {
	// 获取保证金信息
	info, err := m.exchange.GetMarginInfo(ctx)
	if err != nil {
		return nil, exchange.MarginLevelSafe, fmt.Errorf("failed to get margin info: %w", err)
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
func (m *MarginMonitor) determineMarginLevel(ratio float64) exchange.MarginLevel {
	m.mu.RLock()
	defer m.mu.RUnlock()

	switch {
	case ratio <= m.thresholds[exchange.MarginLevelLiquidation]:
		return exchange.MarginLevelLiquidation
	case ratio <= m.thresholds[exchange.MarginLevelDanger]:
		return exchange.MarginLevelDanger
	case ratio <= m.thresholds[exchange.MarginLevelWarning]:
		return exchange.MarginLevelWarning
	default:
		return exchange.MarginLevelSafe
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
		if level != exchange.MarginLevelSafe {
			alert := &exchange.MarginAlert{
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
