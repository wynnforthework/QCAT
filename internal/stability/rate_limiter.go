package stability

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiterType 限流器类型
type RateLimiterType string

const (
	RateLimiterTypeAPI      RateLimiterType = "api"      // API限流
	RateLimiterTypeStrategy RateLimiterType = "strategy" // 策略限流
	RateLimiterTypeUser     RateLimiterType = "user"     // 用户限流
)

// RateLimiterConfig 限流器配置
type RateLimiterConfig struct {
	Type           RateLimiterType
	RequestsPerSec float64
	Burst          int
	Window         time.Duration
	MaxRetries     int
	RetryDelay     time.Duration
}

// RateLimiter 限流器
type RateLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
	configs  map[RateLimiterType]*RateLimiterConfig
	stats    map[string]*RateLimitStats
}

// RateLimitStats 限流统计
type RateLimitStats struct {
	Allowed    int64
	Limited    int64
	Retries    int64
	LastReset  time.Time
	WindowSize time.Duration
}

// NewRateLimiter 创建限流器
func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		configs:  make(map[RateLimiterType]*RateLimiterConfig),
		stats:    make(map[string]*RateLimitStats),
	}

	// 设置默认配置
	rl.setDefaultConfigs()

	return rl
}

// setDefaultConfigs 设置默认配置
func (rl *RateLimiter) setDefaultConfigs() {
	// API限流配置
	rl.configs[RateLimiterTypeAPI] = &RateLimiterConfig{
		Type:           RateLimiterTypeAPI,
		RequestsPerSec: 10.0, // 每秒10个请求
		Burst:          20,   // 突发20个请求
		Window:         time.Minute,
		MaxRetries:     3,
		RetryDelay:     time.Second,
	}

	// 策略限流配置
	rl.configs[RateLimiterTypeStrategy] = &RateLimiterConfig{
		Type:           RateLimiterTypeStrategy,
		RequestsPerSec: 5.0, // 每秒5个请求
		Burst:          10,  // 突发10个请求
		Window:         time.Minute,
		MaxRetries:     2,
		RetryDelay:     2 * time.Second,
	}

	// 用户限流配置
	rl.configs[RateLimiterTypeUser] = &RateLimiterConfig{
		Type:           RateLimiterTypeUser,
		RequestsPerSec: 2.0, // 每秒2个请求
		Burst:          5,   // 突发5个请求
		Window:         time.Minute,
		MaxRetries:     1,
		RetryDelay:     5 * time.Second,
	}
}

// SetConfig 设置限流配置
func (rl *RateLimiter) SetConfig(limiterType RateLimiterType, config *RateLimiterConfig) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.configs[limiterType] = config
}

// GetLimiter 获取限流器
func (rl *RateLimiter) GetLimiter(key string, limiterType RateLimiterType) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiterKey := fmt.Sprintf("%s:%s", limiterType, key)

	if limiter, exists := rl.limiters[limiterKey]; exists {
		return limiter
	}

	// 创建新的限流器
	config := rl.configs[limiterType]
	if config == nil {
		config = rl.configs[RateLimiterTypeAPI] // 使用默认配置
	}

	limiter := rate.NewLimiter(rate.Limit(config.RequestsPerSec), config.Burst)
	rl.limiters[limiterKey] = limiter

	// 初始化统计
	rl.stats[limiterKey] = &RateLimitStats{
		LastReset:  time.Now(),
		WindowSize: config.Window,
	}

	return limiter
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow(key string, limiterType RateLimiterType) bool {
	limiter := rl.GetLimiter(key, limiterType)
	allowed := limiter.Allow()

	rl.updateStats(key, limiterType, allowed)
	return allowed
}

// Wait 等待直到允许请求
func (rl *RateLimiter) Wait(ctx context.Context, key string, limiterType RateLimiterType) error {
	limiter := rl.GetLimiter(key, limiterType)

	// 设置超时
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err := limiter.Wait(ctx)
	if err != nil {
		log.Printf("Rate limiter wait error for %s:%s: %v", limiterType, key, err)
		return err
	}

	rl.updateStats(key, limiterType, true)
	return nil
}

// WaitWithRetry 带重试的等待
func (rl *RateLimiter) WaitWithRetry(ctx context.Context, key string, limiterType RateLimiterType) error {
	config := rl.configs[limiterType]
	if config == nil {
		config = rl.configs[RateLimiterTypeAPI]
	}

	for i := 0; i <= config.MaxRetries; i++ {
		if err := rl.Wait(ctx, key, limiterType); err == nil {
			return nil
		}

		if i < config.MaxRetries {
			log.Printf("Rate limit exceeded for %s:%s, retrying in %v (attempt %d/%d)",
				limiterType, key, config.RetryDelay, i+1, config.MaxRetries+1)
			time.Sleep(config.RetryDelay)
		}
	}

	return fmt.Errorf("rate limit exceeded after %d retries", config.MaxRetries)
}

// updateStats 更新统计信息
func (rl *RateLimiter) updateStats(key string, limiterType RateLimiterType, allowed bool) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiterKey := fmt.Sprintf("%s:%s", limiterType, key)
	stats := rl.stats[limiterKey]

	if stats == nil {
		stats = &RateLimitStats{
			LastReset:  time.Now(),
			WindowSize: rl.configs[limiterType].Window,
		}
		rl.stats[limiterKey] = stats
	}

	// 检查是否需要重置统计
	if time.Since(stats.LastReset) >= stats.WindowSize {
		stats.Allowed = 0
		stats.Limited = 0
		stats.Retries = 0
		stats.LastReset = time.Now()
	}

	if allowed {
		stats.Allowed++
	} else {
		stats.Limited++
	}
}

// GetStats 获取统计信息
func (rl *RateLimiter) GetStats(key string, limiterType RateLimiterType) *RateLimitStats {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	limiterKey := fmt.Sprintf("%s:%s", limiterType, key)
	return rl.stats[limiterKey]
}

// GetAllStats 获取所有统计信息
func (rl *RateLimiter) GetAllStats() map[string]*RateLimitStats {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	result := make(map[string]*RateLimitStats)
	for k, v := range rl.stats {
		result[k] = v
	}
	return result
}

// ResetStats 重置统计信息
func (rl *RateLimiter) ResetStats(key string, limiterType RateLimiterType) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiterKey := fmt.Sprintf("%s:%s", limiterType, key)
	if stats := rl.stats[limiterKey]; stats != nil {
		stats.Allowed = 0
		stats.Limited = 0
		stats.Retries = 0
		stats.LastReset = time.Now()
	}
}

// ResetAllStats 重置所有统计信息
func (rl *RateLimiter) ResetAllStats() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	for _, stats := range rl.stats {
		stats.Allowed = 0
		stats.Limited = 0
		stats.Retries = 0
		stats.LastReset = time.Now()
	}
}

// ExchangeRateLimiter 交易所专用限流器
type ExchangeRateLimiter struct {
	*RateLimiter
	exchangeLimits map[string]*ExchangeLimit
}

// ExchangeLimit 交易所限制
type ExchangeLimit struct {
	Exchange       string
	RequestsPerSec float64
	Burst          int
	WeightLimit    int
	WeightWindow   time.Duration
	CurrentWeight  int
	LastReset      time.Time
}

// NewExchangeRateLimiter 创建交易所限流器
func NewExchangeRateLimiter() *ExchangeRateLimiter {
	return &ExchangeRateLimiter{
		RateLimiter:    NewRateLimiter(),
		exchangeLimits: make(map[string]*ExchangeLimit),
	}
}

// SetExchangeLimit 设置交易所限制
func (erl *ExchangeRateLimiter) SetExchangeLimit(exchange string, limit *ExchangeLimit) {
	erl.mu.Lock()
	defer erl.mu.Unlock()

	erl.exchangeLimits[exchange] = limit
}

// CheckWeightLimit 检查权重限制
func (erl *ExchangeRateLimiter) CheckWeightLimit(exchange string, weight int) bool {
	erl.mu.Lock()
	defer erl.mu.Unlock()

	limit := erl.exchangeLimits[exchange]
	if limit == nil {
		return true // 没有限制
	}

	// 检查是否需要重置权重
	if time.Since(limit.LastReset) >= limit.WeightWindow {
		limit.CurrentWeight = 0
		limit.LastReset = time.Now()
	}

	// 检查权重是否超限
	if limit.CurrentWeight+weight > limit.WeightLimit {
		return false
	}

	limit.CurrentWeight += weight
	return true
}

// WaitForWeight 等待权重可用
func (erl *ExchangeRateLimiter) WaitForWeight(ctx context.Context, exchange string, weight int) error {
	for {
		if erl.CheckWeightLimit(exchange, weight) {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			continue
		}
	}
}

// StrategyRateLimiter 策略专用限流器
type StrategyRateLimiter struct {
	*RateLimiter
	strategyLimits map[string]*StrategyLimit
}

// StrategyLimit 策略限制
type StrategyLimit struct {
	StrategyID      string
	MaxOrdersPerMin int
	MaxTradesPerMin int
	CurrentOrders   int
	CurrentTrades   int
	LastReset       time.Time
}

// NewStrategyRateLimiter 创建策略限流器
func NewStrategyRateLimiter() *StrategyRateLimiter {
	return &StrategyRateLimiter{
		RateLimiter:    NewRateLimiter(),
		strategyLimits: make(map[string]*StrategyLimit),
	}
}

// SetStrategyLimit 设置策略限制
func (srl *StrategyRateLimiter) SetStrategyLimit(strategyID string, limit *StrategyLimit) {
	srl.mu.Lock()
	defer srl.mu.Unlock()

	srl.strategyLimits[strategyID] = limit
}

// CheckOrderLimit 检查订单限制
func (srl *StrategyRateLimiter) CheckOrderLimit(strategyID string) bool {
	srl.mu.Lock()
	defer srl.mu.Unlock()

	limit := srl.strategyLimits[strategyID]
	if limit == nil {
		return true // 没有限制
	}

	// 检查是否需要重置
	if time.Since(limit.LastReset) >= time.Minute {
		limit.CurrentOrders = 0
		limit.CurrentTrades = 0
		limit.LastReset = time.Now()
	}

	return limit.CurrentOrders < limit.MaxOrdersPerMin
}

// IncrementOrder 增加订单计数
func (srl *StrategyRateLimiter) IncrementOrder(strategyID string) {
	srl.mu.Lock()
	defer srl.mu.Unlock()

	limit := srl.strategyLimits[strategyID]
	if limit != nil {
		limit.CurrentOrders++
	}
}

// CheckTradeLimit 检查交易限制
func (srl *StrategyRateLimiter) CheckTradeLimit(strategyID string) bool {
	srl.mu.Lock()
	defer srl.mu.Unlock()

	limit := srl.strategyLimits[strategyID]
	if limit == nil {
		return true // 没有限制
	}

	// 检查是否需要重置
	if time.Since(limit.LastReset) >= time.Minute {
		limit.CurrentOrders = 0
		limit.CurrentTrades = 0
		limit.LastReset = time.Now()
	}

	return limit.CurrentTrades < limit.MaxTradesPerMin
}

// IncrementTrade 增加交易计数
func (srl *StrategyRateLimiter) IncrementTrade(strategyID string) {
	srl.mu.Lock()
	defer srl.mu.Unlock()

	limit := srl.strategyLimits[strategyID]
	if limit != nil {
		limit.CurrentTrades++
	}
}
