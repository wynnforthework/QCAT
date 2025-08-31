package testing

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/exchange"
	"qcat/internal/exchange/binance"
	"qcat/internal/types"
)

// RealMarketDataFetcher 真实市场数据获取器
type RealMarketDataFetcher struct {
	binanceClient   *binance.Client
	exchangeAdapter *exchange.BanexgAdapter
	symbols         []string
	intervals       []string

	// 数据缓存
	klineCache     map[string][]*types.Kline
	tickerCache    map[string]*types.Ticker
	orderBookCache map[string]*types.OrderBook
	tradeCache     map[string][]*types.Trade

	// 缓存锁
	klineMutex     sync.RWMutex
	tickerMutex    sync.RWMutex
	orderBookMutex sync.RWMutex
	tradeMutex     sync.RWMutex

	// 配置
	config *RealDataFetcherConfig

	// 控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// RealDataFetcherConfig 真实数据获取器配置
type RealDataFetcherConfig struct {
	// 交易所配置
	ExchangeConfig *exchange.ExchangeConfig `json:"exchange_config"`

	// 数据获取配置
	Symbols   []string `json:"symbols"`
	Intervals []string `json:"intervals"`

	// 缓存配置
	CacheSize int           `json:"cache_size"`
	CacheTTL  time.Duration `json:"cache_ttl"`

	// 获取频率配置
	KlineUpdateInterval     time.Duration `json:"kline_update_interval"`
	TickerUpdateInterval    time.Duration `json:"ticker_update_interval"`
	OrderBookUpdateInterval time.Duration `json:"orderbook_update_interval"`
	TradeUpdateInterval     time.Duration `json:"trade_update_interval"`

	// 重试配置
	MaxRetries int           `json:"max_retries"`
	RetryDelay time.Duration `json:"retry_delay"`

	// 限流配置
	RateLimit int `json:"rate_limit"` // 每秒请求数

	// 批量获取配置
	BatchSize      int `json:"batch_size"`
	MaxConcurrency int `json:"max_concurrency"`
}

// DefaultRealDataFetcherConfig 默认配置
func DefaultRealDataFetcherConfig() *RealDataFetcherConfig {
	return &RealDataFetcherConfig{
		ExchangeConfig: &exchange.ExchangeConfig{
			Name:      "binance",
			TestNet:   true, // 默认使用测试网
			APIKey:    "",   // 需要配置
			APISecret: "",   // 需要配置
		},
		Symbols: []string{
			"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "XRPUSDT",
			"SOLUSDT", "DOTUSDT", "DOGEUSDT", "AVAXUSDT", "MATICUSDT",
		},
		Intervals: []string{"1m", "5m", "15m", "1h", "4h", "1d"},

		CacheSize: 1000,
		CacheTTL:  5 * time.Minute,

		KlineUpdateInterval:     30 * time.Second,
		TickerUpdateInterval:    5 * time.Second,
		OrderBookUpdateInterval: 2 * time.Second,
		TradeUpdateInterval:     10 * time.Second,

		MaxRetries: 3,
		RetryDelay: time.Second,

		RateLimit:      10, // 每秒10个请求
		BatchSize:      100,
		MaxConcurrency: 5,
	}
}

// NewRealMarketDataFetcher 创建真实市场数据获取器
func NewRealMarketDataFetcher(config *RealDataFetcherConfig) (*RealMarketDataFetcher, error) {
	if config == nil {
		config = DefaultRealDataFetcherConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 创建Binance客户端
	rateLimiter := exchange.NewSimpleRateLimiter(config.RateLimit, time.Minute)
	binanceClient := binance.NewClient(config.ExchangeConfig, rateLimiter)

	// 创建banexg适配器
	exchangeAdapter, err := exchange.NewBanexgAdapter(config.ExchangeConfig)
	if err != nil {
		log.Printf("Warning: Failed to create banexg adapter: %v, using fallback client", err)
	}

	fetcher := &RealMarketDataFetcher{
		binanceClient:   binanceClient,
		exchangeAdapter: exchangeAdapter,
		symbols:         config.Symbols,
		intervals:       config.Intervals,

		klineCache:     make(map[string][]*types.Kline),
		tickerCache:    make(map[string]*types.Ticker),
		orderBookCache: make(map[string]*types.OrderBook),
		tradeCache:     make(map[string][]*types.Trade),

		config: config,
		ctx:    ctx,
		cancel: cancel,
	}

	return fetcher, nil
}

// Start 启动数据获取器
func (f *RealMarketDataFetcher) Start() error {
	log.Println("🚀 启动真实市场数据获取器...")

	// 启动K线数据获取
	f.wg.Add(1)
	go f.fetchKlinesLoop()

	// 启动Ticker数据获取
	f.wg.Add(1)
	go f.fetchTickersLoop()

	// 启动订单簿数据获取
	f.wg.Add(1)
	go f.fetchOrderBooksLoop()

	// 启动交易数据获取
	f.wg.Add(1)
	go f.fetchTradesLoop()

	log.Println("✅ 真实市场数据获取器启动完成")
	return nil
}

// Stop 停止数据获取器
func (f *RealMarketDataFetcher) Stop() {
	log.Println("🛑 停止真实市场数据获取器...")
	f.cancel()
	f.wg.Wait()

	if f.binanceClient != nil {
		f.binanceClient.Close()
	}
	if f.exchangeAdapter != nil {
		f.exchangeAdapter.Close()
	}

	log.Println("✅ 真实市场数据获取器已停止")
}

// fetchKlinesLoop K线数据获取循环
func (f *RealMarketDataFetcher) fetchKlinesLoop() {
	defer f.wg.Done()

	ticker := time.NewTicker(f.config.KlineUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			f.fetchAllKlines()
		case <-f.ctx.Done():
			return
		}
	}
}

// fetchTickersLoop Ticker数据获取循环
func (f *RealMarketDataFetcher) fetchTickersLoop() {
	defer f.wg.Done()

	ticker := time.NewTicker(f.config.TickerUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			f.fetchAllTickers()
		case <-f.ctx.Done():
			return
		}
	}
}

// fetchOrderBooksLoop 订单簿数据获取循环
func (f *RealMarketDataFetcher) fetchOrderBooksLoop() {
	defer f.wg.Done()

	ticker := time.NewTicker(f.config.OrderBookUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			f.fetchAllOrderBooks()
		case <-f.ctx.Done():
			return
		}
	}
}

// fetchTradesLoop 交易数据获取循环
func (f *RealMarketDataFetcher) fetchTradesLoop() {
	defer f.wg.Done()

	ticker := time.NewTicker(f.config.TradeUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			f.fetchAllTrades()
		case <-f.ctx.Done():
			return
		}
	}
}

// fetchAllKlines 获取所有交易对的K线数据
func (f *RealMarketDataFetcher) fetchAllKlines() {
	semaphore := make(chan struct{}, f.config.MaxConcurrency)
	var wg sync.WaitGroup

	for _, symbol := range f.symbols {
		for _, interval := range f.intervals {
			wg.Add(1)
			go func(sym, intv string) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				f.fetchKlineWithRetry(sym, intv)
			}(symbol, interval)
		}
	}

	wg.Wait()
}

// fetchKlineWithRetry 带重试的K线数据获取
func (f *RealMarketDataFetcher) fetchKlineWithRetry(symbol, interval string) {
	var klines []*types.Kline
	var err error

	for attempt := 0; attempt < f.config.MaxRetries; attempt++ {
		// 优先使用banexg适配器
		if f.exchangeAdapter != nil {
			klines, err = f.fetchKlineFromBanexg(symbol, interval)
		} else {
			klines, err = f.fetchKlineFromBinance(symbol, interval)
		}

		if err == nil {
			f.cacheKlines(symbol, interval, klines)
			return
		}

		log.Printf("获取K线数据失败 (尝试 %d/%d): %s %s - %v",
			attempt+1, f.config.MaxRetries, symbol, interval, err)

		if attempt < f.config.MaxRetries-1 {
			time.Sleep(f.config.RetryDelay)
		}
	}

	log.Printf("获取K线数据最终失败: %s %s", symbol, interval)
}

// fetchKlineFromBanexg 从banexg获取K线数据
func (f *RealMarketDataFetcher) fetchKlineFromBanexg(symbol, interval string) ([]*types.Kline, error) {
	// banexg适配器暂时不支持直接获取K线，使用Binance客户端
	return f.fetchKlineFromBinance(symbol, interval)
}

// fetchKlineFromBinance 从Binance获取K线数据
func (f *RealMarketDataFetcher) fetchKlineFromBinance(symbol, interval string) ([]*types.Kline, error) {
	ctx, cancel := context.WithTimeout(f.ctx, 30*time.Second)
	defer cancel()

	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour) // 获取最近24小时数据

	return f.binanceClient.GetKlines(ctx, symbol, interval, startTime, endTime, f.config.BatchSize)
}

// cacheKlines 缓存K线数据
func (f *RealMarketDataFetcher) cacheKlines(symbol, interval string, klines []*types.Kline) {
	f.klineMutex.Lock()
	defer f.klineMutex.Unlock()

	key := fmt.Sprintf("%s_%s", symbol, interval)
	f.klineCache[key] = klines

	// 限制缓存大小
	if len(klines) > f.config.CacheSize {
		f.klineCache[key] = klines[len(klines)-f.config.CacheSize:]
	}
}

// fetchAllTickers 获取所有交易对的Ticker数据
func (f *RealMarketDataFetcher) fetchAllTickers() {
	semaphore := make(chan struct{}, f.config.MaxConcurrency)
	var wg sync.WaitGroup

	for _, symbol := range f.symbols {
		wg.Add(1)
		go func(sym string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			f.fetchTickerWithRetry(sym)
		}(symbol)
	}

	wg.Wait()
}

// fetchTickerWithRetry 带重试的Ticker数据获取
func (f *RealMarketDataFetcher) fetchTickerWithRetry(symbol string) {
	var price float64
	var err error

	for attempt := 0; attempt < f.config.MaxRetries; attempt++ {
		// 优先使用banexg适配器
		if f.exchangeAdapter != nil {
			price, err = f.exchangeAdapter.GetSymbolPrice(f.ctx, symbol)
		} else {
			price, err = f.binanceClient.GetSymbolPrice(f.ctx, symbol)
		}

		if err == nil {
			f.cacheTicker(symbol, price)
			return
		}

		log.Printf("获取Ticker数据失败 (尝试 %d/%d): %s - %v",
			attempt+1, f.config.MaxRetries, symbol, err)

		if attempt < f.config.MaxRetries-1 {
			time.Sleep(f.config.RetryDelay)
		}
	}

	log.Printf("获取Ticker数据最终失败: %s", symbol)
}

// cacheTicker 缓存Ticker数据
func (f *RealMarketDataFetcher) cacheTicker(symbol string, price float64) {
	f.tickerMutex.Lock()
	defer f.tickerMutex.Unlock()

	f.tickerCache[symbol] = &types.Ticker{
		Symbol:    symbol,
		LastPrice: price,
		CloseTime: time.Now(),
	}
}

// fetchAllOrderBooks 获取所有交易对的订单簿数据
func (f *RealMarketDataFetcher) fetchAllOrderBooks() {
	semaphore := make(chan struct{}, f.config.MaxConcurrency)
	var wg sync.WaitGroup

	for _, symbol := range f.symbols {
		wg.Add(1)
		go func(sym string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			f.fetchOrderBookWithRetry(sym)
		}(symbol)
	}

	wg.Wait()
}

// fetchOrderBookWithRetry 带重试的订单簿数据获取
func (f *RealMarketDataFetcher) fetchOrderBookWithRetry(symbol string) {
	var exchangeOrderBook *exchange.OrderBook
	var err error

	for attempt := 0; attempt < f.config.MaxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(f.ctx, 10*time.Second)
		exchangeOrderBook, err = f.binanceClient.GetOrderBook(ctx, symbol, 20) // 获取前20档
		cancel()

		if err == nil {
			// 转换为 types.OrderBook
			orderBook := &types.OrderBook{
				Symbol:    exchangeOrderBook.Symbol,
				UpdatedAt: exchangeOrderBook.Timestamp,
			}

			// 转换 bids
			for _, bid := range exchangeOrderBook.Bids {
				orderBook.Bids = append(orderBook.Bids, types.Level{
					Price:    bid.Price,
					Quantity: bid.Quantity,
				})
			}

			// 转换 asks
			for _, ask := range exchangeOrderBook.Asks {
				orderBook.Asks = append(orderBook.Asks, types.Level{
					Price:    ask.Price,
					Quantity: ask.Quantity,
				})
			}

			f.cacheOrderBook(symbol, orderBook)
			return
		}

		log.Printf("获取订单簿数据失败 (尝试 %d/%d): %s - %v",
			attempt+1, f.config.MaxRetries, symbol, err)

		if attempt < f.config.MaxRetries-1 {
			time.Sleep(f.config.RetryDelay)
		}
	}

	log.Printf("获取订单簿数据最终失败: %s", symbol)
}

// cacheOrderBook 缓存订单簿数据
func (f *RealMarketDataFetcher) cacheOrderBook(symbol string, orderBook *types.OrderBook) {
	f.orderBookMutex.Lock()
	defer f.orderBookMutex.Unlock()

	f.orderBookCache[symbol] = orderBook
}

// fetchAllTrades 获取所有交易对的交易数据
func (f *RealMarketDataFetcher) fetchAllTrades() {
	semaphore := make(chan struct{}, f.config.MaxConcurrency)
	var wg sync.WaitGroup

	for _, symbol := range f.symbols {
		wg.Add(1)
		go func(sym string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			f.fetchTradesWithRetry(sym)
		}(symbol)
	}

	wg.Wait()
}

// fetchTradesWithRetry 带重试的交易数据获取
func (f *RealMarketDataFetcher) fetchTradesWithRetry(symbol string) {
	var trades []*types.Trade
	var err error

	for attempt := 0; attempt < f.config.MaxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(f.ctx, 10*time.Second)
		trades, err = f.binanceClient.GetTrades(ctx, symbol, 100) // 获取最近100笔交易
		cancel()

		if err == nil {
			f.cacheTrades(symbol, trades)
			return
		}

		log.Printf("获取交易数据失败 (尝试 %d/%d): %s - %v",
			attempt+1, f.config.MaxRetries, symbol, err)

		if attempt < f.config.MaxRetries-1 {
			time.Sleep(f.config.RetryDelay)
		}
	}

	log.Printf("获取交易数据最终失败: %s", symbol)
}

// cacheTrades 缓存交易数据
func (f *RealMarketDataFetcher) cacheTrades(symbol string, trades []*types.Trade) {
	f.tradeMutex.Lock()
	defer f.tradeMutex.Unlock()

	f.tradeCache[symbol] = trades

	// 限制缓存大小
	if len(trades) > f.config.CacheSize {
		f.tradeCache[symbol] = trades[len(trades)-f.config.CacheSize:]
	}
}

// GetKlines 获取K线数据
func (f *RealMarketDataFetcher) GetKlines(symbol, interval string) []*types.Kline {
	f.klineMutex.RLock()
	defer f.klineMutex.RUnlock()

	key := fmt.Sprintf("%s_%s", symbol, interval)
	if klines, exists := f.klineCache[key]; exists {
		// 返回副本
		result := make([]*types.Kline, len(klines))
		copy(result, klines)
		return result
	}

	return nil
}

// GetTicker 获取Ticker数据
func (f *RealMarketDataFetcher) GetTicker(symbol string) *types.Ticker {
	f.tickerMutex.RLock()
	defer f.tickerMutex.RUnlock()

	if ticker, exists := f.tickerCache[symbol]; exists {
		// 返回副本
		return &types.Ticker{
			Symbol:    ticker.Symbol,
			LastPrice: ticker.LastPrice,
			CloseTime: ticker.CloseTime,
		}
	}

	return nil
}

// GetOrderBook 获取订单簿数据
func (f *RealMarketDataFetcher) GetOrderBook(symbol string) *types.OrderBook {
	f.orderBookMutex.RLock()
	defer f.orderBookMutex.RUnlock()

	if orderBook, exists := f.orderBookCache[symbol]; exists {
		// 返回副本
		return &types.OrderBook{
			Symbol:    orderBook.Symbol,
			Bids:      append([]types.Level(nil), orderBook.Bids...),
			Asks:      append([]types.Level(nil), orderBook.Asks...),
			UpdatedAt: orderBook.UpdatedAt,
		}
	}

	return nil
}

// GetTrades 获取交易数据
func (f *RealMarketDataFetcher) GetTrades(symbol string) []*types.Trade {
	f.tradeMutex.RLock()
	defer f.tradeMutex.RUnlock()

	if trades, exists := f.tradeCache[symbol]; exists {
		// 返回副本
		result := make([]*types.Trade, len(trades))
		copy(result, trades)
		return result
	}

	return nil
}

// GetAllSymbols 获取所有交易对
func (f *RealMarketDataFetcher) GetAllSymbols() []string {
	return append([]string(nil), f.symbols...)
}

// GetAllIntervals 获取所有时间间隔
func (f *RealMarketDataFetcher) GetAllIntervals() []string {
	return append([]string(nil), f.intervals...)
}

// GetCacheStats 获取缓存统计信息
func (f *RealMarketDataFetcher) GetCacheStats() *CacheStats {
	f.klineMutex.RLock()
	klineCount := len(f.klineCache)
	f.klineMutex.RUnlock()

	f.tickerMutex.RLock()
	tickerCount := len(f.tickerCache)
	f.tickerMutex.RUnlock()

	f.orderBookMutex.RLock()
	orderBookCount := len(f.orderBookCache)
	f.orderBookMutex.RUnlock()

	f.tradeMutex.RLock()
	tradeCount := len(f.tradeCache)
	f.tradeMutex.RUnlock()

	return &CacheStats{
		KlineEntries:     klineCount,
		TickerEntries:    tickerCount,
		OrderBookEntries: orderBookCount,
		TradeEntries:     tradeCount,
		LastUpdated:      time.Now(),
	}
}

// ClearCache 清空所有缓存
func (f *RealMarketDataFetcher) ClearCache() {
	f.klineMutex.Lock()
	f.klineCache = make(map[string][]*types.Kline)
	f.klineMutex.Unlock()

	f.tickerMutex.Lock()
	f.tickerCache = make(map[string]*types.Ticker)
	f.tickerMutex.Unlock()

	f.orderBookMutex.Lock()
	f.orderBookCache = make(map[string]*types.OrderBook)
	f.orderBookMutex.Unlock()

	f.tradeMutex.Lock()
	f.tradeCache = make(map[string][]*types.Trade)
	f.tradeMutex.Unlock()

	log.Println("✅ 真实市场数据缓存已清空")
}

// CacheStats 缓存统计信息
type CacheStats struct {
	KlineEntries     int       `json:"kline_entries"`
	TickerEntries    int       `json:"ticker_entries"`
	OrderBookEntries int       `json:"orderbook_entries"`
	TradeEntries     int       `json:"trade_entries"`
	LastUpdated      time.Time `json:"last_updated"`
}

// BatchFetchKlines 批量获取K线数据
func (f *RealMarketDataFetcher) BatchFetchKlines(symbols []string, interval string, startTime, endTime time.Time) map[string][]*types.Kline {
	result := make(map[string][]*types.Kline)
	semaphore := make(chan struct{}, f.config.MaxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, symbol := range symbols {
		wg.Add(1)
		go func(sym string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			ctx, cancel := context.WithTimeout(f.ctx, 30*time.Second)
			defer cancel()

			klines, err := f.binanceClient.GetKlines(ctx, sym, interval, startTime, endTime, f.config.BatchSize)
			if err != nil {
				log.Printf("批量获取K线数据失败: %s - %v", sym, err)
				return
			}

			mu.Lock()
			result[sym] = klines
			mu.Unlock()
		}(symbol)
	}

	wg.Wait()
	return result
}

// GetHistoricalKlines 获取历史K线数据（不使用缓存）
func (f *RealMarketDataFetcher) GetHistoricalKlines(symbol, interval string, startTime, endTime time.Time, limit int) ([]*types.Kline, error) {
	ctx, cancel := context.WithTimeout(f.ctx, 60*time.Second)
	defer cancel()

	return f.binanceClient.GetKlines(ctx, symbol, interval, startTime, endTime, limit)
}

// IsHealthy 检查数据获取器健康状态
func (f *RealMarketDataFetcher) IsHealthy() bool {
	// 检查是否有最近的数据
	stats := f.GetCacheStats()
	return stats.TickerEntries > 0 && stats.LastUpdated.After(time.Now().Add(-5*time.Minute))
}
