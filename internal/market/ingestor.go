package market

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"
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

// Ingestor manages market data collection
type Ingestor struct {
	db       *sql.DB
	wsClient *WSClient
	mu       sync.RWMutex // 保护并发访问

	// 新增：延迟监控
	latencyHistory []time.Duration
	lastUpdate     time.Time

	// 新增：数据质量监控
	dataGaps []time.Time
	outliers []interface{}

	// 新增：订阅管理
	subscriptions map[string]*channelSubscription
}

// NewIngestor creates a new market data ingestor
func NewIngestor(db *sql.DB) *Ingestor {
	// 新增：创建WebSocket客户端
	wsClient := NewWSClient("wss://stream.binance.com:9443/ws")

	return &Ingestor{
		db:             db,
		wsClient:       wsClient,
		subscriptions:  make(map[string]*channelSubscription),
		latencyHistory: make([]time.Duration, 0, 100),
		dataGaps:       make([]time.Time, 0, 50),
		outliers:       make([]interface{}, 0, 50),
	}
}

// SubscribeOrderBook subscribes to order book updates
func (i *Ingestor) SubscribeOrderBook(ctx context.Context, symbol string) (<-chan *OrderBook, error) {
	ch := make(chan *OrderBook, 1000)

	// 新增：创建可取消的上下文
	ctx, cancel := context.WithCancel(ctx)
	sub := &channelSubscription{ch: ch, cancel: cancel}

	// 新增：保存订阅
	i.mu.Lock()
	i.subscriptions[symbol+"_orderbook"] = sub
	i.mu.Unlock()

	// 新增：创建WebSocket订阅
	wsSub := WSSubscription{
		Symbol:     symbol,
		MarketType: MarketTypeSpot,
		Channels:   []string{fmt.Sprintf("%s@depth20@100ms", symbol)},
	}

	// 新增：设置消息处理器
	handler := func(msg interface{}) error {
		if orderBook, ok := msg.(*OrderBook); ok {
			select {
			case ch <- orderBook:
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
		return nil, fmt.Errorf("failed to subscribe to order book: %w", err)
	}

	return ch, nil
}

// SubscribeTrades subscribes to trade updates
func (i *Ingestor) SubscribeTrades(ctx context.Context, symbol string) (<-chan *Trade, error) {
	ch := make(chan *Trade, 1000)

	// 新增：创建可取消的上下文
	ctx, cancel := context.WithCancel(ctx)
	sub := &channelSubscription{ch: ch, cancel: cancel}

	// 新增：保存订阅
	i.mu.Lock()
	i.subscriptions[symbol+"_trades"] = sub
	i.mu.Unlock()

	// 新增：创建WebSocket订阅
	wsSub := WSSubscription{
		Symbol:     symbol,
		MarketType: MarketTypeSpot,
		Channels:   []string{fmt.Sprintf("%s@trade", symbol)},
	}

	// 新增：设置消息处理器
	handler := func(msg interface{}) error {
		if trade, ok := msg.(*Trade); ok {
			select {
			case ch <- trade:
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
		return nil, fmt.Errorf("failed to subscribe to trades: %w", err)
	}

	return ch, nil
}

// SubscribeKlines subscribes to kline updates
func (i *Ingestor) SubscribeKlines(ctx context.Context, symbol, interval string) (<-chan *Kline, error) {
	ch := make(chan *Kline, 1000)

	// 新增：创建可取消的上下文
	ctx, cancel := context.WithCancel(ctx)
	sub := &channelSubscription{ch: ch, cancel: cancel}

	// 新增：保存订阅
	i.mu.Lock()
	i.subscriptions[symbol+"_klines"] = sub
	i.mu.Unlock()

	// 新增：创建WebSocket订阅
	wsSub := WSSubscription{
		Symbol:     symbol,
		MarketType: MarketTypeSpot,
		Channels:   []string{fmt.Sprintf("%s@kline_%s", symbol, interval)},
	}

	// 新增：设置消息处理器
	handler := func(msg interface{}) error {
		if kline, ok := msg.(*Kline); ok {
			select {
			case ch <- kline:
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
		return nil, fmt.Errorf("failed to subscribe to klines: %w", err)
	}

	return ch, nil
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
