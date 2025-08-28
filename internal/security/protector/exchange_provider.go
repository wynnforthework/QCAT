package protector

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"qcat/internal/exchange"
)

// ExchangeDataProvider 交易所数据提供者接口
type ExchangeDataProvider interface {
	// IsHealthy 检查连接健康状态
	IsHealthy() bool
	
	// GetFundData 获取资金数据
	GetFundData(ctx context.Context) (*ExchangeFundData, error)
	
	// GetPositions 获取持仓数据
	GetPositions(ctx context.Context) ([]*Position, error)
	
	// GetHistoricalReturns 获取历史收益率
	GetHistoricalReturns(ctx context.Context, days int) ([]float64, error)
	
	// GetHistoricalEquity 获取历史净值
	GetHistoricalEquity(ctx context.Context, days int) ([]float64, error)
	
	// GetSymbolPrice 获取交易对价格
	GetSymbolPrice(ctx context.Context, symbol string) (float64, error)
	
	// GetOrderBookDepth 获取订单簿深度
	GetOrderBookDepth(ctx context.Context, symbol string) (*OrderBookDepth, error)
	
	// GetTradingVolume 获取交易量数据
	GetTradingVolume(ctx context.Context, symbol string, period string) (float64, error)
}

// ExchangeFundData 交易所资金数据
type ExchangeFundData struct {
	TotalBalance     float64   `json:"total_balance"`
	AvailableBalance float64   `json:"available_balance"`
	LockedBalance    float64   `json:"locked_balance"`
	DailyPL          float64   `json:"daily_pl"`
	UnrealizedPL     float64   `json:"unrealized_pl"`
	Timestamp        time.Time `json:"timestamp"`
}

// Position 持仓信息结构
type Position struct {
	Symbol            string  `json:"symbol"`
	Side              string  `json:"side"` // LONG, SHORT
	Size              float64 `json:"size"`
	Notional          float64 `json:"notional"`
	EntryPrice        float64 `json:"entry_price"`
	MarkPrice         float64 `json:"mark_price"`
	UnrealizedPnL     float64 `json:"unrealized_pnl"`
	RealizedPnL       float64 `json:"realized_pnl"`
	Leverage          int     `json:"leverage"`
	MarginType        string  `json:"margin_type"`        // ISOLATED, CROSS
	IsolatedMargin    float64 `json:"isolated_margin"`
	MaintenanceMargin float64 `json:"maintenance_margin"`
	LiquidationPrice  float64 `json:"liquidation_price"`
}

// OrderBookDepth 订单簿深度
type OrderBookDepth struct {
	Symbol    string      `json:"symbol"`
	Bids      []PriceSize `json:"bids"`
	Asks      []PriceSize `json:"asks"`
	Timestamp time.Time   `json:"timestamp"`
}

// PriceSize 价格和数量
type PriceSize struct {
	Price float64 `json:"price"`
	Size  float64 `json:"size"`
}

// ExchangeProviderConfig 交易所提供者配置
type ExchangeProviderConfig struct {
	RetryAttempts    int           `json:"retry_attempts"`
	RetryDelay       time.Duration `json:"retry_delay"`
	RequestTimeout   time.Duration `json:"request_timeout"`
	CacheTTL         time.Duration `json:"cache_ttl"`
	RateLimitWindow  time.Duration `json:"rate_limit_window"`
	RateLimitRequests int          `json:"rate_limit_requests"`
}

// DefaultExchangeProvider 默认交易所数据提供者实现
type DefaultExchangeProvider struct {
	exchange exchange.Exchange
	config   *ExchangeProviderConfig
	
	// 缓存
	fundDataCache    *CachedData[*ExchangeFundData]
	positionsCache   *CachedData[[]*Position]
	priceCache       map[string]*CachedData[float64]
	
	// 速率限制
	rateLimiter *RateLimiter
	
	// 健康状态
	isHealthy    bool
	lastHealthCheck time.Time
}

// CachedData 缓存数据结构
type CachedData[T any] struct {
	Data      T
	Timestamp time.Time
	TTL       time.Duration
}

// IsValid 检查缓存是否有效
func (c *CachedData[T]) IsValid() bool {
	return time.Since(c.Timestamp) < c.TTL
}

// RateLimiter 速率限制器
type RateLimiter struct {
	requests    []time.Time
	maxRequests int
	window      time.Duration
}

// NewDefaultExchangeProvider 创建默认交易所数据提供者
func NewDefaultExchangeProvider(ex exchange.Exchange, config *ExchangeProviderConfig) *DefaultExchangeProvider {
	if config == nil {
		config = &ExchangeProviderConfig{
			RetryAttempts:     3,
			RetryDelay:        time.Second,
			RequestTimeout:    30 * time.Second,
			CacheTTL:          30 * time.Second,
			RateLimitWindow:   time.Minute,
			RateLimitRequests: 60,
		}
	}

	return &DefaultExchangeProvider{
		exchange:    ex,
		config:      config,
		priceCache:  make(map[string]*CachedData[float64]),
		rateLimiter: &RateLimiter{
			requests:    make([]time.Time, 0),
			maxRequests: config.RateLimitRequests,
			window:      config.RateLimitWindow,
		},
		isHealthy: true,
	}
}

// IsHealthy 检查连接健康状态
func (p *DefaultExchangeProvider) IsHealthy() bool {
	// 如果最近5分钟内没有检查过健康状态，进行检查
	if time.Since(p.lastHealthCheck) > 5*time.Minute {
		p.checkHealth()
	}
	return p.isHealthy
}

// checkHealth 检查健康状态
func (p *DefaultExchangeProvider) checkHealth() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 尝试获取服务器时间来检查连接
	_, err := p.exchange.GetServerTime(ctx)
	p.isHealthy = (err == nil)
	p.lastHealthCheck = time.Now()

	if !p.isHealthy {
		log.Printf("Exchange health check failed: %v", err)
	}
}

// GetFundData 获取资金数据
func (p *DefaultExchangeProvider) GetFundData(ctx context.Context) (*ExchangeFundData, error) {
	// 检查缓存
	if p.fundDataCache != nil && p.fundDataCache.IsValid() {
		return p.fundDataCache.Data, nil
	}

	// 速率限制检查
	if !p.rateLimiter.Allow() {
		return nil, fmt.Errorf("rate limit exceeded")
	}

	// 重试逻辑
	var fundData *ExchangeFundData
	var err error

	for attempt := 0; attempt < p.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(p.config.RetryDelay * time.Duration(attempt))
		}

		fundData, err = p.fetchFundData(ctx)
		if err == nil {
			break
		}

		log.Printf("Attempt %d to fetch fund data failed: %v", attempt+1, err)
	}

	if err != nil {
		p.isHealthy = false
		return nil, fmt.Errorf("failed to fetch fund data after %d attempts: %w", p.config.RetryAttempts, err)
	}

	// 更新缓存
	p.fundDataCache = &CachedData[*ExchangeFundData]{
		Data:      fundData,
		Timestamp: time.Now(),
		TTL:       p.config.CacheTTL,
	}

	return fundData, nil
}

// fetchFundData 从交易所获取资金数据
func (p *DefaultExchangeProvider) fetchFundData(ctx context.Context) (*ExchangeFundData, error) {
	// 获取账户信息
	account, err := p.exchange.GetAccount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get account info: %w", err)
	}

	// 计算总余额和可用余额
	var totalBalance, availableBalance, lockedBalance float64
	
	for _, balance := range account.Balances {
		if balance.Asset == "USDT" || balance.Asset == "USD" {
			totalBalance += balance.Free + balance.Locked
			availableBalance += balance.Free
			lockedBalance += balance.Locked
		}
	}

	// 获取持仓信息计算未实现盈亏
	positions, err := p.exchange.GetPositions(ctx)
	if err != nil {
		log.Printf("Warning: failed to get positions for PnL calculation: %v", err)
	}

	var unrealizedPL float64
	for _, pos := range positions {
		unrealizedPL += pos.UnrealizedPnL
	}

	// 计算日盈亏（简化实现，实际应该基于历史数据）
	dailyPL := unrealizedPL * 0.1 // 假设当日盈亏是未实现盈亏的10%

	return &ExchangeFundData{
		TotalBalance:     totalBalance,
		AvailableBalance: availableBalance,
		LockedBalance:    lockedBalance,
		DailyPL:          dailyPL,
		UnrealizedPL:     unrealizedPL,
		Timestamp:        time.Now(),
	}, nil
}

// GetPositions 获取持仓数据
func (p *DefaultExchangeProvider) GetPositions(ctx context.Context) ([]*Position, error) {
	// 检查缓存
	if p.positionsCache != nil && p.positionsCache.IsValid() {
		return p.positionsCache.Data, nil
	}

	// 速率限制检查
	if !p.rateLimiter.Allow() {
		return nil, fmt.Errorf("rate limit exceeded")
	}

	// 重试逻辑
	var positions []*Position
	var err error

	for attempt := 0; attempt < p.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(p.config.RetryDelay * time.Duration(attempt))
		}

		positions, err = p.fetchPositions(ctx)
		if err == nil {
			break
		}

		log.Printf("Attempt %d to fetch positions failed: %v", attempt+1, err)
	}

	if err != nil {
		p.isHealthy = false
		return nil, fmt.Errorf("failed to fetch positions after %d attempts: %w", p.config.RetryAttempts, err)
	}

	// 更新缓存
	p.positionsCache = &CachedData[[]*Position]{
		Data:      positions,
		Timestamp: time.Now(),
		TTL:       p.config.CacheTTL,
	}

	return positions, nil
}

// fetchPositions 从交易所获取持仓数据
func (p *DefaultExchangeProvider) fetchPositions(ctx context.Context) ([]*Position, error) {
	exchangePositions, err := p.exchange.GetPositions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get positions: %w", err)
	}

	positions := make([]*Position, 0, len(exchangePositions))
	
	for _, pos := range exchangePositions {
		// 跳过零仓位
		if pos.Size == 0 {
			continue
		}

		// 计算名义价值
		notional := math.Abs(pos.Size * pos.MarkPrice)

		position := &Position{
			Symbol:            pos.Symbol,
			Side:              pos.Side,
			Size:              math.Abs(pos.Size),
			Notional:          notional,
			EntryPrice:        pos.EntryPrice,
			MarkPrice:         pos.MarkPrice,
			UnrealizedPnL:     pos.UnrealizedPnL,
			RealizedPnL:       pos.RealizedPnL,
			Leverage:          pos.Leverage,
			MarginType:        pos.MarginType,
			IsolatedMargin:    pos.IsolatedMargin,
			MaintenanceMargin: pos.MaintenanceMargin,
			LiquidationPrice:  pos.LiquidationPrice,
		}

		positions = append(positions, position)
	}

	return positions, nil
}

// GetHistoricalReturns 获取历史收益率
func (p *DefaultExchangeProvider) GetHistoricalReturns(ctx context.Context, days int) ([]float64, error) {
	// 获取历史净值数据
	equity, err := p.GetHistoricalEquity(ctx, days+1) // 多获取一天用于计算收益率
	if err != nil {
		return nil, fmt.Errorf("failed to get historical equity: %w", err)
	}

	if len(equity) < 2 {
		return nil, fmt.Errorf("insufficient equity data for return calculation")
	}

	// 计算日收益率
	returns := make([]float64, len(equity)-1)
	for i := 1; i < len(equity); i++ {
		if equity[i-1] > 0 {
			returns[i-1] = (equity[i] - equity[i-1]) / equity[i-1]
		}
	}

	return returns, nil
}

// GetHistoricalEquity 获取历史净值
func (p *DefaultExchangeProvider) GetHistoricalEquity(ctx context.Context, days int) ([]float64, error) {
	// 获取历史账户快照
	snapshots, err := p.exchange.GetAccountSnapshots(ctx, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get account snapshots: %w", err)
	}

	if len(snapshots) == 0 {
		return nil, fmt.Errorf("no account snapshots available")
	}

	equity := make([]float64, len(snapshots))
	for i, snapshot := range snapshots {
		equity[i] = snapshot.TotalWalletBalance
	}

	return equity, nil
}

// GetSymbolPrice 获取交易对价格
func (p *DefaultExchangeProvider) GetSymbolPrice(ctx context.Context, symbol string) (float64, error) {
	// 检查缓存
	if cached, exists := p.priceCache[symbol]; exists && cached.IsValid() {
		return cached.Data, nil
	}

	// 速率限制检查
	if !p.rateLimiter.Allow() {
		return 0, fmt.Errorf("rate limit exceeded")
	}

	// 获取价格
	ticker, err := p.exchange.GetTicker(ctx, symbol)
	if err != nil {
		return 0, fmt.Errorf("failed to get ticker for %s: %w", symbol, err)
	}

	price := ticker.LastPrice

	// 更新缓存
	p.priceCache[symbol] = &CachedData[float64]{
		Data:      price,
		Timestamp: time.Now(),
		TTL:       p.config.CacheTTL,
	}

	return price, nil
}

// GetOrderBookDepth 获取订单簿深度
func (p *DefaultExchangeProvider) GetOrderBookDepth(ctx context.Context, symbol string) (*OrderBookDepth, error) {
	// 速率限制检查
	if !p.rateLimiter.Allow() {
		return nil, fmt.Errorf("rate limit exceeded")
	}

	orderBook, err := p.exchange.GetOrderBook(ctx, symbol, 20) // 获取前20档
	if err != nil {
		return nil, fmt.Errorf("failed to get order book for %s: %w", symbol, err)
	}

	depth := &OrderBookDepth{
		Symbol:    symbol,
		Timestamp: time.Now(),
		Bids:      make([]PriceSize, len(orderBook.Bids)),
		Asks:      make([]PriceSize, len(orderBook.Asks)),
	}

	for i, bid := range orderBook.Bids {
		depth.Bids[i] = PriceSize{
			Price: bid.Price,
			Size:  bid.Quantity,
		}
	}

	for i, ask := range orderBook.Asks {
		depth.Asks[i] = PriceSize{
			Price: ask.Price,
			Size:  ask.Quantity,
		}
	}

	return depth, nil
}

// GetTradingVolume 获取交易量数据
func (p *DefaultExchangeProvider) GetTradingVolume(ctx context.Context, symbol string, period string) (float64, error) {
	// 速率限制检查
	if !p.rateLimiter.Allow() {
		return 0, fmt.Errorf("rate limit exceeded")
	}

	// 获取24小时统计数据
	stats, err := p.exchange.Get24HrStats(ctx, symbol)
	if err != nil {
		return 0, fmt.Errorf("failed to get 24hr stats for %s: %w", symbol, err)
	}

	return stats.Volume, nil
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow() bool {
	now := time.Now()
	
	// 清理过期的请求记录
	cutoff := now.Add(-rl.window)
	validRequests := make([]time.Time, 0, len(rl.requests))
	for _, req := range rl.requests {
		if req.After(cutoff) {
			validRequests = append(validRequests, req)
		}
	}
	rl.requests = validRequests

	// 检查是否超过限制
	if len(rl.requests) >= rl.maxRequests {
		return false
	}

	// 记录新请求
	rl.requests = append(rl.requests, now)
	return true
}

// MockExchangeProvider 模拟交易所数据提供者（用于测试）
type MockExchangeProvider struct {
	fundData  *ExchangeFundData
	positions []*Position
	healthy   bool
}

// NewMockExchangeProvider 创建模拟交易所数据提供者
func NewMockExchangeProvider() *MockExchangeProvider {
	return &MockExchangeProvider{
		fundData: &ExchangeFundData{
			TotalBalance:     100000.0,
			AvailableBalance: 80000.0,
			LockedBalance:    20000.0,
			DailyPL:          1500.0,
			UnrealizedPL:     2500.0,
			Timestamp:        time.Now(),
		},
		positions: []*Position{
			{
				Symbol:            "BTCUSDT",
				Side:              "LONG",
				Size:              1.5,
				Notional:          75000.0,
				EntryPrice:        48000.0,
				MarkPrice:         50000.0,
				UnrealizedPnL:     3000.0,
				Leverage:          10,
				MarginType:        "CROSS",
				LiquidationPrice:  45000.0,
			},
		},
		healthy: true,
	}
}

// IsHealthy 检查连接健康状态
func (m *MockExchangeProvider) IsHealthy() bool {
	return m.healthy
}

// GetFundData 获取资金数据
func (m *MockExchangeProvider) GetFundData(ctx context.Context) (*ExchangeFundData, error) {
	if !m.healthy {
		return nil, fmt.Errorf("exchange is not healthy")
	}
	return m.fundData, nil
}

// GetPositions 获取持仓数据
func (m *MockExchangeProvider) GetPositions(ctx context.Context) ([]*Position, error) {
	if !m.healthy {
		return nil, fmt.Errorf("exchange is not healthy")
	}
	return m.positions, nil
}

// GetHistoricalReturns 获取历史收益率
func (m *MockExchangeProvider) GetHistoricalReturns(ctx context.Context, days int) ([]float64, error) {
	if !m.healthy {
		return nil, fmt.Errorf("exchange is not healthy")
	}
	
	// 生成模拟的历史收益率数据
	returns := make([]float64, days)
	for i := 0; i < days; i++ {
		// 简单的随机游走模型
		returns[i] = (float64(i%7) - 3) * 0.01 // -3% to +3%
	}
	return returns, nil
}

// GetHistoricalEquity 获取历史净值
func (m *MockExchangeProvider) GetHistoricalEquity(ctx context.Context, days int) ([]float64, error) {
	if !m.healthy {
		return nil, fmt.Errorf("exchange is not healthy")
	}
	
	// 生成模拟的历史净值数据
	equity := make([]float64, days)
	baseValue := 100000.0
	
	for i := 0; i < days; i++ {
		// 简单的增长模型
		growth := 1.0 + float64(i)*0.001 // 每天0.1%增长
		equity[i] = baseValue * growth
	}
	return equity, nil
}

// GetSymbolPrice 获取交易对价格
func (m *MockExchangeProvider) GetSymbolPrice(ctx context.Context, symbol string) (float64, error) {
	if !m.healthy {
		return 0, fmt.Errorf("exchange is not healthy")
	}
	
	// 返回模拟价格
	prices := map[string]float64{
		"BTCUSDT": 50000.0,
		"ETHUSDT": 3000.0,
		"BNBUSDT": 400.0,
	}
	
	if price, exists := prices[symbol]; exists {
		return price, nil
	}
	
	return 0, fmt.Errorf("symbol %s not found", symbol)
}

// GetOrderBookDepth 获取订单簿深度
func (m *MockExchangeProvider) GetOrderBookDepth(ctx context.Context, symbol string) (*OrderBookDepth, error) {
	if !m.healthy {
		return nil, fmt.Errorf("exchange is not healthy")
	}
	
	price, err := m.GetSymbolPrice(ctx, symbol)
	if err != nil {
		return nil, err
	}
	
	return &OrderBookDepth{
		Symbol: symbol,
		Bids: []PriceSize{
			{Price: price - 10, Size: 1.0},
			{Price: price - 20, Size: 2.0},
		},
		Asks: []PriceSize{
			{Price: price + 10, Size: 1.0},
			{Price: price + 20, Size: 2.0},
		},
		Timestamp: time.Now(),
	}, nil
}

// GetTradingVolume 获取交易量数据
func (m *MockExchangeProvider) GetTradingVolume(ctx context.Context, symbol string, period string) (float64, error) {
	if !m.healthy {
		return 0, fmt.Errorf("exchange is not healthy")
	}
	
	// 返回模拟交易量
	return 1000000.0, nil
}

// SetHealthy 设置健康状态（测试用）
func (m *MockExchangeProvider) SetHealthy(healthy bool) {
	m.healthy = healthy
}

// SetFundData 设置资金数据（测试用）
func (m *MockExchangeProvider) SetFundData(data *ExchangeFundData) {
	m.fundData = data
}

// SetPositions 设置持仓数据（测试用）
func (m *MockExchangeProvider) SetPositions(positions []*Position) {
	m.positions = positions
}