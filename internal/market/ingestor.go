package market

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/exchange/binance"
	"qcat/internal/market/quality"
	"qcat/internal/market/storage"
)

// Subscription represents a market data subscription
type Subscription interface {
	Close()
}

// channelSubscription implements Subscription
// 用于管理订阅的取消函数
type channelSubscription struct {
	ch     interface{}
	cancel context.CancelFunc
}

func (s *channelSubscription) Close() {
	s.cancel()
}

// Ingestor manages market data collection with real Binance integration
type Ingestor struct {
	db            *sql.DB
	binanceClient *binance.Client
	binanceWS     *binance.WSClient
	storage       *storage.Storage
	qualityMonitor *quality.Monitor
	mu            sync.RWMutex

	// Configuration
	testnet bool
	apiKey  string
	apiSecret string

	// Subscription management
	subscriptions map[string]*channelSubscription
	
	// Data processing workers
	workers       int
	workerPool    chan struct{}
	
	// Metrics and monitoring
	stats         map[string]*IngestorStats
	lastUpdate    time.Time
}

// IngestorStats represents ingestion statistics
type IngestorStats struct {
	Symbol          string    `json:"symbol"`
	DataType        string    `json:"data_type"`
	MessagesTotal   int64     `json:"messages_total"`
	MessagesValid   int64     `json:"messages_valid"`
	MessagesInvalid int64     `json:"messages_invalid"`
	LastMessage     time.Time `json:"last_message"`
	AvgLatency      time.Duration `json:"avg_latency"`
}

// NewIngestor creates a new market data ingestor with real Binance integration
func NewIngestor(db *sql.DB, apiKey, apiSecret string, testnet bool) *Ingestor {
	// Create Binance clients
	binanceClient := binance.NewClient(apiKey, apiSecret, testnet)
	binanceWS := binance.NewWSClient(testnet)
	
	// Create storage and quality monitor
	storageManager := storage.NewStorage(db)
	qualityMonitor := quality.NewMonitor()
	
	ingestor := &Ingestor{
		db:             db,
		binanceClient:  binanceClient,
		binanceWS:      binanceWS,
		storage:        storageManager,
		qualityMonitor: qualityMonitor,
		testnet:        testnet,
		apiKey:         apiKey,
		apiSecret:      apiSecret,
		subscriptions:  make(map[string]*channelSubscription),
		workers:        10, // Number of worker goroutines
		workerPool:     make(chan struct{}, 10),
		stats:          make(map[string]*IngestorStats),
	}
	
	// Set up quality monitor callback
	qualityMonitor.SetIssueCallback(func(issue quality.QualityIssue) {
		log.Printf("Data quality issue: %s - %s", issue.IssueType, issue.Description)
	})
	
	// Initialize worker pool
	for i := 0; i < ingestor.workers; i++ {
		ingestor.workerPool <- struct{}{}
	}
	
	return ingestor
}

// SubscribeOrderBook subscribes to order book updates using real Binance WebSocket
func (i *Ingestor) SubscribeOrderBook(ctx context.Context, symbol string) (<-chan *OrderBook, error) {
	// Connect to Binance WebSocket if not already connected
	if err := i.binanceWS.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to Binance WebSocket: %w", err)
	}

	// Subscribe to order book updates with 100ms speed
	orderBookCh, err := i.binanceWS.SubscribeOrderBook(ctx, symbol, "100ms")
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to order book: %w", err)
	}

	// Create output channel
	outputCh := make(chan *OrderBook, 1000)

	// Create cancellable context
	ctx, cancel := context.WithCancel(ctx)
	sub := &channelSubscription{ch: outputCh, cancel: cancel}

	// Save subscription
	i.mu.Lock()
	i.subscriptions[symbol+"_orderbook"] = sub
	i.updateStats(symbol, "orderbook")
	i.mu.Unlock()

	// Start processing goroutine
	go func() {
		defer close(outputCh)
		defer cancel()

		for {
			select {
			case orderBook, ok := <-orderBookCh:
				if !ok {
					log.Printf("Order book channel closed for %s", symbol)
					return
				}

				// Quality check
				if err := i.qualityMonitor.CheckOrderBook(orderBook); err != nil {
					log.Printf("Order book quality check failed for %s: %v", symbol, err)
					i.updateStatsError(symbol, "orderbook")
					continue
				}

				// Store in database (async)
				go func(ob *OrderBook) {
					<-i.workerPool // Acquire worker
					defer func() { i.workerPool <- struct{}{} }() // Release worker

					if err := i.storage.SaveOrderBook(context.Background(), ob); err != nil {
						log.Printf("Failed to save order book for %s: %v", ob.Symbol, err)
					}
				}(orderBook)

				// Send to output channel
				select {
				case outputCh <- orderBook:
					i.updateStatsSuccess(symbol, "orderbook")
				case <-ctx.Done():
					return
				default:
					// Channel full, skip this update
					log.Printf("Order book channel full for %s, skipping update", symbol)
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	return outputCh, nil
}

// SubscribeTrades subscribes to trade updates using real Binance WebSocket
func (i *Ingestor) SubscribeTrades(ctx context.Context, symbol string) (<-chan *Trade, error) {
	// Connect to Binance WebSocket if not already connected
	if err := i.binanceWS.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to Binance WebSocket: %w", err)
	}

	// Subscribe to trade updates
	tradeCh, err := i.binanceWS.SubscribeTrades(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to trades: %w", err)
	}

	// Create output channel
	outputCh := make(chan *Trade, 1000)

	// Create cancellable context
	ctx, cancel := context.WithCancel(ctx)
	sub := &channelSubscription{ch: outputCh, cancel: cancel}

	// Save subscription
	i.mu.Lock()
	i.subscriptions[symbol+"_trades"] = sub
	i.updateStats(symbol, "trades")
	i.mu.Unlock()

	// Start processing goroutine
	go func() {
		defer close(outputCh)
		defer cancel()

		for {
			select {
			case trade, ok := <-tradeCh:
				if !ok {
					log.Printf("Trade channel closed for %s", symbol)
					return
				}

				// Quality check
				if err := i.qualityMonitor.CheckTrade(trade); err != nil {
					log.Printf("Trade quality check failed for %s: %v", symbol, err)
					i.updateStatsError(symbol, "trades")
					continue
				}

				// Store in database (async)
				go func(t *Trade) {
					<-i.workerPool // Acquire worker
					defer func() { i.workerPool <- struct{}{} }() // Release worker

					if err := i.storage.SaveTrade(context.Background(), t); err != nil {
						log.Printf("Failed to save trade for %s: %v", t.Symbol, err)
					}
				}(trade)

				// Send to output channel
				select {
				case outputCh <- trade:
					i.updateStatsSuccess(symbol, "trades")
				case <-ctx.Done():
					return
				default:
					// Channel full, skip this update
					log.Printf("Trade channel full for %s, skipping update", symbol)
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	return outputCh, nil
}

// SubscribeKlines subscribes to kline updates using real Binance WebSocket
func (i *Ingestor) SubscribeKlines(ctx context.Context, symbol, interval string) (<-chan *Kline, error) {
	// Connect to Binance WebSocket if not already connected
	if err := i.binanceWS.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to Binance WebSocket: %w", err)
	}

	// Subscribe to kline updates
	klineCh, err := i.binanceWS.SubscribeKlines(ctx, symbol, interval)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to klines: %w", err)
	}

	// Create output channel
	outputCh := make(chan *Kline, 1000)

	// Create cancellable context
	ctx, cancel := context.WithCancel(ctx)
	sub := &channelSubscription{ch: outputCh, cancel: cancel}

	// Save subscription
	i.mu.Lock()
	i.subscriptions[symbol+"_klines_"+interval] = sub
	i.updateStats(symbol, "klines_"+interval)
	i.mu.Unlock()

	// Start processing goroutine
	go func() {
		defer close(outputCh)
		defer cancel()

		for {
			select {
			case kline, ok := <-klineCh:
				if !ok {
					log.Printf("Kline channel closed for %s %s", symbol, interval)
					return
				}

				// Quality check
				if err := i.qualityMonitor.CheckKline(kline); err != nil {
					log.Printf("Kline quality check failed for %s %s: %v", symbol, interval, err)
					i.updateStatsError(symbol, "klines_"+interval)
					continue
				}

				// Store in database (async)
				go func(k *Kline) {
					<-i.workerPool // Acquire worker
					defer func() { i.workerPool <- struct{}{} }() // Release worker

					if err := i.storage.SaveKline(context.Background(), k); err != nil {
						log.Printf("Failed to save kline for %s %s: %v", k.Symbol, k.Interval, err)
					}
				}(kline)

				// Send to output channel
				select {
				case outputCh <- kline:
					i.updateStatsSuccess(symbol, "klines_"+interval)
				case <-ctx.Done():
					return
				default:
					// Channel full, skip this update
					log.Printf("Kline channel full for %s %s, skipping update", symbol, interval)
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	return outputCh, nil
}

// SubscribeFundingRates subscribes to funding rate updates
func (i *Ingestor) SubscribeFundingRates(ctx context.Context, symbol string) (<-chan *FundingRate, error) {
	ch := make(chan *FundingRate, 1000)

	// 新增：创建可取消的上下文
	ctx, cancel := context.WithCancel(ctx)
	sub := &channelSubscription{ch: ch, cancel: cancel}

	// 新增：保存订阅
	i.mu.Lock()
	i.subscriptions[symbol+"_funding"] = sub
	i.mu.Unlock()

	// 新增：创建WebSocket订阅
	wsSub := WSSubscription{
		Symbol:     symbol,
		MarketType: MarketTypeFutures,
		Channels:   []string{fmt.Sprintf("%s@markPrice@1s", symbol)},
	}

	// 新增：设置消息处理器
	handler := func(msg interface{}) error {
		if funding, ok := msg.(*FundingRate); ok {
			select {
			case ch <- funding:
				// 新增：更新延迟监控
				i.updateLatency()
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	}

	// 新增：连接到WebSocket并订阅
	if err := i.wsClient.Connect(ctx); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to connect WebSocket: %w", err)
	}

	if err := i.wsClient.Subscribe(wsSub, handler); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to subscribe to funding rates: %w", err)
	}

	return ch, nil
}

// 新增：更新延迟监控
func (i *Ingestor) updateLatency() {
	i.mu.Lock()
	defer i.mu.Unlock()

	now := time.Now()
	if !i.lastUpdate.IsZero() {
		latency := now.Sub(i.lastUpdate)
		i.latencyHistory = append(i.latencyHistory, latency)

		// 保持历史记录在合理范围内
		if len(i.latencyHistory) > 100 {
			i.latencyHistory = i.latencyHistory[1:]
		}
	}
	i.lastUpdate = now
}

// GetDataLatency returns the current data latency
func (i *Ingestor) GetDataLatency() time.Duration {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if len(i.latencyHistory) == 0 {
		return 100 * time.Millisecond // 默认值
	}

	// 新增：计算平均延迟
	var total time.Duration
	for _, latency := range i.latencyHistory {
		total += latency
	}
	return total / time.Duration(len(i.latencyHistory))
}

// 新增：检测数据间隙
func (i *Ingestor) detectDataGaps() {
	i.mu.Lock()
	defer i.mu.Unlock()

	now := time.Now()
	if !i.lastUpdate.IsZero() {
		gap := now.Sub(i.lastUpdate)
		// 如果间隙超过5秒，认为是数据间隙
		if gap > 5*time.Second {
			i.dataGaps = append(i.dataGaps, now)

			// 保持间隙记录在合理范围内
			if len(i.dataGaps) > 50 {
				i.dataGaps = i.dataGaps[1:]
			}
		}
	}
}

// GetDataGaps returns data gaps
func (i *Ingestor) GetDataGaps() []time.Time {
	i.mu.RLock()
	defer i.mu.RUnlock()

	// 新增：检测当前数据间隙
	i.detectDataGaps()

	// 返回间隙记录的副本
	gaps := make([]time.Time, len(i.dataGaps))
	copy(gaps, i.dataGaps)
	return gaps
}

// 新增：检测异常值
func (i *Ingestor) detectOutliers(data interface{}) {
	i.mu.Lock()
	defer i.mu.Unlock()

	// 新增：简单的异常值检测逻辑
	switch v := data.(type) {
	case *Trade:
		// 检测价格异常
		if v.Price <= 0 || v.Quantity <= 0 {
			i.outliers = append(i.outliers, v)
		}
	case *Kline:
		// 检测K线数据异常
		if v.High < v.Low || v.Open < 0 || v.Close < 0 {
			i.outliers = append(i.outliers, v)
		}
	case *OrderBook:
		// 检测订单簿异常
		if len(v.Bids) == 0 && len(v.Asks) == 0 {
			i.outliers = append(i.outliers, v)
		}
	}

	// 保持异常值记录在合理范围内
	if len(i.outliers) > 50 {
		i.outliers = i.outliers[1:]
	}
}

// GetOutliers returns data outliers
func (i *Ingestor) GetOutliers() []interface{} {
	i.mu.RLock()
	defer i.mu.RUnlock()

	// 返回异常值记录的副本
	outliers := make([]interface{}, len(i.outliers))
	copy(outliers, i.outliers)
	return outliers
}

// GetTradeHistory returns historical trades
func (i *Ingestor) GetTradeHistory(ctx context.Context, symbol string, start, end time.Time) ([]*Trade, error) {
	query := `
		SELECT id, symbol, price, size, side, fee, fee_currency, created_at
		FROM trades
		WHERE symbol = $1 AND created_at BETWEEN $2 AND $3
		ORDER BY created_at ASC
	`

	rows, err := i.db.QueryContext(ctx, query, symbol, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query trades: %w", err)
	}
	defer rows.Close()

	var trades []*Trade
	for rows.Next() {
		var t Trade
		if err := rows.Scan(
			&t.ID,
			&t.Symbol,
			&t.Price,
			&t.Quantity,
			&t.Side,
			&t.Fee,
			&t.FeeCoin,
			&t.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("failed to scan trade: %w", err)
		}
		trades = append(trades, &t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating trades: %w", err)
	}

	return trades, nil
}

// GetKlineHistory returns historical klines
func (i *Ingestor) GetKlineHistory(ctx context.Context, symbol, interval string, start, end time.Time) ([]*Kline, error) {
	query := `
		SELECT symbol, interval, timestamp, open, high, low, close, volume
		FROM market_data
		WHERE symbol = $1 AND interval = $2 AND timestamp BETWEEN $3 AND $4
		ORDER BY timestamp ASC
	`

	rows, err := i.db.QueryContext(ctx, query, symbol, interval, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query klines: %w", err)
	}
	defer rows.Close()

	var klines []*Kline
	for rows.Next() {
		var k Kline
		if err := rows.Scan(
			&k.Symbol,
			&k.Interval,
			&k.OpenTime,
			&k.Open,
			&k.High,
			&k.Low,
			&k.Close,
			&k.Volume,
		); err != nil {
			return nil, fmt.Errorf("failed to scan kline: %w", err)
		}
		k.CloseTime = k.OpenTime.Add(time.Minute)
		k.Complete = true
		klines = append(klines, &k)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating klines: %w", err)
	}

	return klines, nil
}

// GetFundingRates returns historical funding rates
func (i *Ingestor) GetFundingRates(ctx context.Context, symbol string, start, end time.Time) ([]*FundingRate, error) {
	query := `
		SELECT symbol, rate, next_rate, next_time, created_at
		FROM funding_rates
		WHERE symbol = $1 AND created_at BETWEEN $2 AND $3
		ORDER BY created_at ASC
	`

	rows, err := i.db.QueryContext(ctx, query, symbol, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query funding rates: %w", err)
	}
	defer rows.Close()

	var rates []*FundingRate
	for rows.Next() {
		var r FundingRate
		if err := rows.Scan(
			&r.Symbol,
			&r.Rate,
			&r.NextRate,
			&r.NextTime,
			&r.LastUpdated,
		); err != nil {
			return nil, fmt.Errorf("failed to scan funding rate: %w", err)
		}
		rates = append(rates, &r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating funding rates: %w", err)
	}

	return rates, nil
}

// GetOpenInterest returns historical open interest
func (i *Ingestor) GetOpenInterest(ctx context.Context, symbol string, start, end time.Time) ([]*OpenInterest, error) {
	query := `
		SELECT symbol, value, notional, timestamp
		FROM open_interest
		WHERE symbol = $1 AND timestamp BETWEEN $2 AND $3
		ORDER BY timestamp ASC
	`

	rows, err := i.db.QueryContext(ctx, query, symbol, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query open interest: %w", err)
	}
	defer rows.Close()

	var oi []*OpenInterest
	for rows.Next() {
		var o OpenInterest
		if err := rows.Scan(
			&o.Symbol,
			&o.Value,
			&o.Notional,
			&o.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("failed to scan open interest: %w", err)
		}
		oi = append(oi, &o)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating open interest: %w", err)
	}

	return oi, nil
}

// GetOrderBook returns the current order book
func (i *Ingestor) GetOrderBook(ctx context.Context, symbol string) (*OrderBook, error) {
	// 新增：从数据库获取最新的订单簿数据
	query := `
		SELECT symbol, bids, asks, updated_at
		FROM order_books
		WHERE symbol = $1
		ORDER BY updated_at DESC
		LIMIT 1
	`

	var orderBook OrderBook
	var bidsJSON, asksJSON []byte

	err := i.db.QueryRowContext(ctx, query, symbol).Scan(
		&orderBook.Symbol,
		&bidsJSON,
		&asksJSON,
		&orderBook.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// 新增：如果没有数据，返回空的订单簿
			return &OrderBook{
				Symbol:    symbol,
				Bids:      []Level{},
				Asks:      []Level{},
				UpdatedAt: time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("failed to query order book: %w", err)
	}

	// 新增：解析JSON数据
	if err := json.Unmarshal(bidsJSON, &orderBook.Bids); err != nil {
		return nil, fmt.Errorf("failed to parse bids: %w", err)
	}

	if err := json.Unmarshal(asksJSON, &orderBook.Asks); err != nil {
		return nil, fmt.Errorf("failed to parse asks: %w", err)
	}

	return &orderBook, nil
}
// updateStats initializes or updates statistics for a symbol and data type
func (i *Ingestor) updateStats(symbol, dataType string) {
	key := fmt.Sprintf("%s:%s", symbol, dataType)
	if _, exists := i.stats[key]; !exists {
		i.stats[key] = &IngestorStats{
			Symbol:   symbol,
			DataType: dataType,
		}
	}
}

// updateStatsSuccess updates statistics for successful message processing
func (i *Ingestor) updateStatsSuccess(symbol, dataType string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	
	key := fmt.Sprintf("%s:%s", symbol, dataType)
	if stats, exists := i.stats[key]; exists {
		stats.MessagesTotal++
		stats.MessagesValid++
		stats.LastMessage = time.Now()
	}
}

// updateStatsError updates statistics for failed message processing
func (i *Ingestor) updateStatsError(symbol, dataType string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	
	key := fmt.Sprintf("%s:%s", symbol, dataType)
	if stats, exists := i.stats[key]; exists {
		stats.MessagesTotal++
		stats.MessagesInvalid++
		stats.LastMessage = time.Now()
	}
}

// GetHistoricalKlines fetches historical kline data from Binance API
func (i *Ingestor) GetHistoricalKlines(ctx context.Context, symbol, interval string, startTime, endTime time.Time, limit int) ([]*Kline, error) {
	return i.binanceClient.GetKlines(ctx, symbol, interval, startTime, endTime, limit)
}

// GetHistoricalTrades fetches historical trade data from Binance API
func (i *Ingestor) GetHistoricalTrades(ctx context.Context, symbol string, limit int) ([]*Trade, error) {
	return i.binanceClient.GetTrades(ctx, symbol, limit)
}

// GetCurrentOrderBook fetches current order book from Binance API
func (i *Ingestor) GetCurrentOrderBook(ctx context.Context, symbol string, limit int) (*OrderBook, error) {
	return i.binanceClient.GetOrderBook(ctx, symbol, limit)
}

// GetCurrentFundingRate fetches current funding rate from Binance API
func (i *Ingestor) GetCurrentFundingRate(ctx context.Context, symbol string) (*FundingRate, error) {
	return i.binanceClient.GetFundingRate(ctx, symbol)
}

// GetCurrentOpenInterest fetches current open interest from Binance API
func (i *Ingestor) GetCurrentOpenInterest(ctx context.Context, symbol string) (*OpenInterest, error) {
	return i.binanceClient.GetOpenInterest(ctx, symbol)
}

// GetStats returns ingestion statistics
func (i *Ingestor) GetStats() map[string]*IngestorStats {
	i.mu.RLock()
	defer i.mu.RUnlock()
	
	result := make(map[string]*IngestorStats)
	for k, v := range i.stats {
		// Create a copy
		stats := *v
		result[k] = &stats
	}
	
	return result
}

// GetQualityMetrics returns data quality metrics
func (i *Ingestor) GetQualityMetrics() map[string]*quality.QualityMetrics {
	return i.qualityMonitor.GetMetrics()
}

// GetQualityIssues returns recent data quality issues
func (i *Ingestor) GetQualityIssues(limit int) []quality.QualityIssue {
	return i.qualityMonitor.GetIssues(limit)
}

// GetOverallQualityScore returns overall data quality score
func (i *Ingestor) GetOverallQualityScore() float64 {
	return i.qualityMonitor.GetQualityScore()
}

// Close closes all subscriptions and connections
func (i *Ingestor) Close() error {
	i.mu.Lock()
	defer i.mu.Unlock()
	
	// Close all subscriptions
	for _, sub := range i.subscriptions {
		sub.Close()
	}
	
	// Close WebSocket connection
	if err := i.binanceWS.Close(); err != nil {
		log.Printf("Error closing Binance WebSocket: %v", err)
	}
	
	return nil
}

// StartDataCollection starts collecting data for specified symbols
func (i *Ingestor) StartDataCollection(ctx context.Context, symbols []string, intervals []string) error {
	log.Printf("Starting data collection for %d symbols", len(symbols))
	
	for _, symbol := range symbols {
		// Subscribe to order book updates
		if _, err := i.SubscribeOrderBook(ctx, symbol); err != nil {
			log.Printf("Failed to subscribe to order book for %s: %v", symbol, err)
		}
		
		// Subscribe to trade updates
		if _, err := i.SubscribeTrades(ctx, symbol); err != nil {
			log.Printf("Failed to subscribe to trades for %s: %v", symbol, err)
		}
		
		// Subscribe to kline updates for each interval
		for _, interval := range intervals {
			if _, err := i.SubscribeKlines(ctx, symbol, interval); err != nil {
				log.Printf("Failed to subscribe to klines for %s %s: %v", symbol, interval, err)
			}
		}
		
		// Fetch and store initial historical data
		go i.fetchInitialData(ctx, symbol, intervals)
	}
	
	log.Printf("Data collection started successfully")
	return nil
}

// fetchInitialData fetches initial historical data for a symbol
func (i *Ingestor) fetchInitialData(ctx context.Context, symbol string, intervals []string) {
	log.Printf("Fetching initial data for %s", symbol)
	
	// Fetch recent trades
	trades, err := i.GetHistoricalTrades(ctx, symbol, 1000)
	if err != nil {
		log.Printf("Failed to fetch historical trades for %s: %v", symbol, err)
	} else {
		for _, trade := range trades {
			if err := i.storage.SaveTrade(ctx, trade); err != nil {
				log.Printf("Failed to save historical trade for %s: %v", symbol, err)
			}
		}
		log.Printf("Saved %d historical trades for %s", len(trades), symbol)
	}
	
	// Fetch recent klines for each interval
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour) // Last 24 hours
	
	for _, interval := range intervals {
		klines, err := i.GetHistoricalKlines(ctx, symbol, interval, startTime, endTime, 1000)
		if err != nil {
			log.Printf("Failed to fetch historical klines for %s %s: %v", symbol, interval, err)
			continue
		}
		
		for _, kline := range klines {
			if err := i.storage.SaveKline(ctx, kline); err != nil {
				log.Printf("Failed to save historical kline for %s %s: %v", symbol, interval, err)
			}
		}
		log.Printf("Saved %d historical klines for %s %s", len(klines), symbol, interval)
	}
	
	// Fetch current order book
	orderBook, err := i.GetCurrentOrderBook(ctx, symbol, 20)
	if err != nil {
		log.Printf("Failed to fetch current order book for %s: %v", symbol, err)
	} else {
		if err := i.storage.SaveOrderBook(ctx, orderBook); err != nil {
			log.Printf("Failed to save current order book for %s: %v", symbol, err)
		} else {
			log.Printf("Saved current order book for %s", symbol)
		}
	}
}

// PerformDataCleanup performs periodic data cleanup
func (i *Ingestor) PerformDataCleanup(ctx context.Context, retentionDays int) error {
	log.Printf("Starting data cleanup with %d days retention", retentionDays)
	
	if err := i.storage.CleanupOldData(ctx, retentionDays); err != nil {
		return fmt.Errorf("failed to cleanup old data: %w", err)
	}
	
	log.Printf("Data cleanup completed successfully")
	return nil
}