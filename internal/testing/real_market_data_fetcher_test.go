package testing

import (
	"testing"
	"time"
)

func TestRealMarketDataFetcher_Creation(t *testing.T) {
	config := DefaultRealDataFetcherConfig()

	// 使用测试网配置
	config.ExchangeConfig.TestNet = true
	config.Symbols = []string{"BTCUSDT", "ETHUSDT"}
	config.Intervals = []string{"1m", "5m"}

	fetcher, err := NewRealMarketDataFetcher(config)
	if err != nil {
		t.Fatalf("创建真实市场数据获取器失败: %v", err)
	}

	if fetcher == nil {
		t.Fatal("获取器为空")
	}

	if len(fetcher.symbols) != 2 {
		t.Errorf("期望2个交易对，实际得到%d个", len(fetcher.symbols))
	}

	if len(fetcher.intervals) != 2 {
		t.Errorf("期望2个时间间隔，实际得到%d个", len(fetcher.intervals))
	}
}

func TestRealMarketDataFetcher_DefaultConfig(t *testing.T) {
	config := DefaultRealDataFetcherConfig()

	if config == nil {
		t.Fatal("默认配置为空")
	}

	if len(config.Symbols) == 0 {
		t.Error("默认配置应包含交易对")
	}

	if len(config.Intervals) == 0 {
		t.Error("默认配置应包含时间间隔")
	}

	if config.CacheSize <= 0 {
		t.Error("缓存大小应大于0")
	}

	if config.MaxRetries <= 0 {
		t.Error("最大重试次数应大于0")
	}
}

func TestRealMarketDataFetcher_CacheOperations(t *testing.T) {
	config := DefaultRealDataFetcherConfig()
	config.ExchangeConfig.TestNet = true
	config.Symbols = []string{"BTCUSDT"}
	config.Intervals = []string{"1m"}

	fetcher, err := NewRealMarketDataFetcher(config)
	if err != nil {
		t.Fatalf("创建获取器失败: %v", err)
	}

	// 测试初始缓存状态
	stats := fetcher.GetCacheStats()
	if stats.KlineEntries != 0 {
		t.Error("初始K线缓存应为空")
	}
	if stats.TickerEntries != 0 {
		t.Error("初始Ticker缓存应为空")
	}

	// 测试缓存Ticker数据
	fetcher.cacheTicker("BTCUSDT", 45000.0)

	ticker := fetcher.GetTicker("BTCUSDT")
	if ticker == nil {
		t.Error("应该能获取到缓存的Ticker数据")
	}
	if ticker.LastPrice != 45000.0 {
		t.Errorf("期望价格45000.0，实际得到%f", ticker.LastPrice)
	}

	// 测试清空缓存
	fetcher.ClearCache()
	stats = fetcher.GetCacheStats()
	if stats.TickerEntries != 0 {
		t.Error("清空后Ticker缓存应为空")
	}
}

func TestRealMarketDataFetcher_GetMethods(t *testing.T) {
	config := DefaultRealDataFetcherConfig()
	config.Symbols = []string{"BTCUSDT", "ETHUSDT"}
	config.Intervals = []string{"1m", "5m"}

	fetcher, err := NewRealMarketDataFetcher(config)
	if err != nil {
		t.Fatalf("创建获取器失败: %v", err)
	}

	// 测试获取所有交易对
	symbols := fetcher.GetAllSymbols()
	if len(symbols) != 2 {
		t.Errorf("期望2个交易对，实际得到%d个", len(symbols))
	}

	// 测试获取所有时间间隔
	intervals := fetcher.GetAllIntervals()
	if len(intervals) != 2 {
		t.Errorf("期望2个时间间隔，实际得到%d个", len(intervals))
	}

	// 测试获取不存在的数据
	klines := fetcher.GetKlines("BTCUSDT", "1m")
	if klines != nil {
		t.Error("不存在的K线数据应返回nil")
	}

	ticker := fetcher.GetTicker("BTCUSDT")
	if ticker != nil {
		t.Error("不存在的Ticker数据应返回nil")
	}
}

func TestRealMarketDataFetcher_HealthCheck(t *testing.T) {
	config := DefaultRealDataFetcherConfig()
	config.ExchangeConfig.TestNet = true

	fetcher, err := NewRealMarketDataFetcher(config)
	if err != nil {
		t.Fatalf("创建获取器失败: %v", err)
	}

	// 初始状态应该不健康（没有数据）
	if fetcher.IsHealthy() {
		t.Error("初始状态应该不健康")
	}

	// 添加一些测试数据
	fetcher.cacheTicker("BTCUSDT", 45000.0)

	// 现在应该健康
	if !fetcher.IsHealthy() {
		t.Error("有数据后应该健康")
	}
}

// 基准测试
func BenchmarkRealMarketDataFetcher_CacheTicker(b *testing.B) {
	config := DefaultRealDataFetcherConfig()
	config.ExchangeConfig.TestNet = true

	fetcher, err := NewRealMarketDataFetcher(config)
	if err != nil {
		b.Fatalf("创建获取器失败: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fetcher.cacheTicker("BTCUSDT", float64(45000+i))
	}
}

func BenchmarkRealMarketDataFetcher_GetTicker(b *testing.B) {
	config := DefaultRealDataFetcherConfig()
	config.ExchangeConfig.TestNet = true

	fetcher, err := NewRealMarketDataFetcher(config)
	if err != nil {
		b.Fatalf("创建获取器失败: %v", err)
	}

	// 预先缓存数据
	fetcher.cacheTicker("BTCUSDT", 45000.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fetcher.GetTicker("BTCUSDT")
	}
}

// 集成测试（需要网络连接）
func TestRealMarketDataFetcher_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	config := DefaultRealDataFetcherConfig()
	config.ExchangeConfig.TestNet = true
	config.Symbols = []string{"BTCUSDT"}
	config.Intervals = []string{"1m"}
	config.KlineUpdateInterval = 5 * time.Second
	config.TickerUpdateInterval = 2 * time.Second

	fetcher, err := NewRealMarketDataFetcher(config)
	if err != nil {
		t.Fatalf("创建获取器失败: %v", err)
	}

	// 启动获取器
	err = fetcher.Start()
	if err != nil {
		t.Fatalf("启动获取器失败: %v", err)
	}
	defer fetcher.Stop()

	// 等待一段时间让数据获取
	time.Sleep(10 * time.Second)

	// 检查是否获取到数据
	ticker := fetcher.GetTicker("BTCUSDT")
	if ticker == nil {
		t.Error("应该能获取到Ticker数据")
	} else {
		t.Logf("获取到BTCUSDT价格: %f", ticker.LastPrice)
	}

	klines := fetcher.GetKlines("BTCUSDT", "1m")
	if klines == nil || len(klines) == 0 {
		t.Error("应该能获取到K线数据")
	} else {
		t.Logf("获取到%d条K线数据", len(klines))
	}

	// 检查健康状态
	if !fetcher.IsHealthy() {
		t.Error("获取器应该处于健康状态")
	}

	// 检查缓存统计
	stats := fetcher.GetCacheStats()
	t.Logf("缓存统计: Ticker=%d, Kline=%d", stats.TickerEntries, stats.KlineEntries)
}

// 测试历史数据获取
func TestRealMarketDataFetcher_HistoricalData(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过历史数据测试")
	}

	config := DefaultRealDataFetcherConfig()
	config.ExchangeConfig.TestNet = true

	fetcher, err := NewRealMarketDataFetcher(config)
	if err != nil {
		t.Fatalf("创建获取器失败: %v", err)
	}

	// 获取历史数据
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	klines, err := fetcher.GetHistoricalKlines("BTCUSDT", "1h", startTime, endTime, 24)
	if err != nil {
		t.Fatalf("获取历史K线数据失败: %v", err)
	}

	if len(klines) == 0 {
		t.Error("应该能获取到历史K线数据")
	} else {
		t.Logf("获取到%d条历史K线数据", len(klines))
	}
}
