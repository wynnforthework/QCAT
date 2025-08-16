package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"qcat/internal/market"

	"github.com/gorilla/websocket"
)

// WSClient represents a Binance WebSocket client
type WSClient struct {
	*market.WSClient
	baseURL string
}

// NewWSClient creates a new Binance WebSocket client
func NewWSClient(testnet bool) *WSClient {
	baseURL := "wss://stream.binance.com:9443/ws"
	if testnet {
		baseURL = "wss://testnet.binance.vision/ws"
	}

	wsClient := market.NewWSClient(baseURL)
	
	return &WSClient{
		WSClient: wsClient,
		baseURL:  baseURL,
	}
}

// SubscribeOrderBook subscribes to order book updates
func (c *WSClient) SubscribeOrderBook(ctx context.Context, symbol string, speed string) (<-chan *market.OrderBook, error) {
	ch := make(chan *market.OrderBook, 1000)
	
	// Normalize symbol (lowercase for Binance WebSocket)
	symbol = strings.ToLower(symbol)
	
	// Create subscription
	stream := fmt.Sprintf("%s@depth20@%s", symbol, speed)
	sub := market.WSSubscription{
		Symbol:     symbol,
		MarketType: market.MarketTypeSpot,
		Channels:   []string{stream},
	}

	// Create handler
	handler := func(msg interface{}) error {
		data, ok := msg.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid message format")
		}

		orderBook, err := c.parseOrderBookMessage(data)
		if err != nil {
			return fmt.Errorf("failed to parse order book: %w", err)
		}

		select {
		case ch <- orderBook:
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Channel full, skip this update
		}

		return nil
	}

	if err := c.Subscribe(sub, handler); err != nil {
		return nil, fmt.Errorf("failed to subscribe to order book: %w", err)
	}

	return ch, nil
}

// SubscribeTrades subscribes to trade updates
func (c *WSClient) SubscribeTrades(ctx context.Context, symbol string) (<-chan *market.Trade, error) {
	ch := make(chan *market.Trade, 1000)
	
	// Normalize symbol (lowercase for Binance WebSocket)
	symbol = strings.ToLower(symbol)
	
	// Create subscription
	stream := fmt.Sprintf("%s@trade", symbol)
	sub := market.WSSubscription{
		Symbol:     symbol,
		MarketType: market.MarketTypeSpot,
		Channels:   []string{stream},
	}

	// Create handler
	handler := func(msg interface{}) error {
		data, ok := msg.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid message format")
		}

		trade, err := c.parseTradeMessage(data)
		if err != nil {
			return fmt.Errorf("failed to parse trade: %w", err)
		}

		select {
		case ch <- trade:
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Channel full, skip this update
		}

		return nil
	}

	if err := c.Subscribe(sub, handler); err != nil {
		return nil, fmt.Errorf("failed to subscribe to trades: %w", err)
	}

	return ch, nil
}

// SubscribeKlines subscribes to kline updates
func (c *WSClient) SubscribeKlines(ctx context.Context, symbol, interval string) (<-chan *market.Kline, error) {
	ch := make(chan *market.Kline, 1000)
	
	// Normalize symbol (lowercase for Binance WebSocket)
	symbol = strings.ToLower(symbol)
	
	// Create subscription
	stream := fmt.Sprintf("%s@kline_%s", symbol, interval)
	sub := market.WSSubscription{
		Symbol:     symbol,
		MarketType: market.MarketTypeSpot,
		Channels:   []string{stream},
	}

	// Create handler
	handler := func(msg interface{}) error {
		data, ok := msg.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid message format")
		}

		kline, err := c.parseKlineMessage(data)
		if err != nil {
			return fmt.Errorf("failed to parse kline: %w", err)
		}

		select {
		case ch <- kline:
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Channel full, skip this update
		}

		return nil
	}

	if err := c.Subscribe(sub, handler); err != nil {
		return nil, fmt.Errorf("failed to subscribe to klines: %w", err)
	}

	return ch, nil
}

// SubscribeTicker subscribes to ticker updates
func (c *WSClient) SubscribeTicker(ctx context.Context, symbol string) (<-chan *market.Ticker, error) {
	ch := make(chan *market.Ticker, 1000)
	
	// Normalize symbol (lowercase for Binance WebSocket)
	symbol = strings.ToLower(symbol)
	
	// Create subscription
	stream := fmt.Sprintf("%s@ticker", symbol)
	sub := market.WSSubscription{
		Symbol:     symbol,
		MarketType: market.MarketTypeSpot,
		Channels:   []string{stream},
	}

	// Create handler
	handler := func(msg interface{}) error {
		data, ok := msg.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid message format")
		}

		ticker, err := c.parseTickerMessage(data)
		if err != nil {
			return fmt.Errorf("failed to parse ticker: %w", err)
		}

		select {
		case ch <- ticker:
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Channel full, skip this update
		}

		return nil
	}

	if err := c.Subscribe(sub, handler); err != nil {
		return nil, fmt.Errorf("failed to subscribe to ticker: %w", err)
	}

	return ch, nil
}

// parseOrderBookMessage parses Binance order book message
func (c *WSClient) parseOrderBookMessage(data map[string]interface{}) (*market.OrderBook, error) {
	symbol, ok := data["s"].(string)
	if !ok {
		return nil, fmt.Errorf("missing symbol")
	}

	bidsData, ok := data["b"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("missing bids data")
	}

	asksData, ok := data["a"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("missing asks data")
	}

	orderBook := &market.OrderBook{
		Symbol:    strings.ToUpper(symbol),
		UpdatedAt: time.Now(),
		Bids:      make([]market.Level, 0, len(bidsData)),
		Asks:      make([]market.Level, 0, len(asksData)),
	}

	// Parse bids
	for _, bidData := range bidsData {
		bid, ok := bidData.([]interface{})
		if !ok || len(bid) < 2 {
			continue
		}

		priceStr, ok := bid[0].(string)
		if !ok {
			continue
		}
		qtyStr, ok := bid[1].(string)
		if !ok {
			continue
		}

		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			continue
		}
		quantity, err := strconv.ParseFloat(qtyStr, 64)
		if err != nil {
			continue
		}

		orderBook.Bids = append(orderBook.Bids, market.Level{
			Price:    price,
			Quantity: quantity,
		})
	}

	// Parse asks
	for _, askData := range asksData {
		ask, ok := askData.([]interface{})
		if !ok || len(ask) < 2 {
			continue
		}

		priceStr, ok := ask[0].(string)
		if !ok {
			continue
		}
		qtyStr, ok := ask[1].(string)
		if !ok {
			continue
		}

		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			continue
		}
		quantity, err := strconv.ParseFloat(qtyStr, 64)
		if err != nil {
			continue
		}

		orderBook.Asks = append(orderBook.Asks, market.Level{
			Price:    price,
			Quantity: quantity,
		})
	}

	return orderBook, nil
}

// parseTradeMessage parses Binance trade message
func (c *WSClient) parseTradeMessage(data map[string]interface{}) (*market.Trade, error) {
	symbol, ok := data["s"].(string)
	if !ok {
		return nil, fmt.Errorf("missing symbol")
	}

	tradeID, ok := data["t"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing trade ID")
	}

	priceStr, ok := data["p"].(string)
	if !ok {
		return nil, fmt.Errorf("missing price")
	}

	qtyStr, ok := data["q"].(string)
	if !ok {
		return nil, fmt.Errorf("missing quantity")
	}

	timestamp, ok := data["T"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing timestamp")
	}

	isBuyerMaker, ok := data["m"].(bool)
	if !ok {
		return nil, fmt.Errorf("missing buyer maker flag")
	}

	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid price: %w", err)
	}

	quantity, err := strconv.ParseFloat(qtyStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid quantity: %w", err)
	}

	side := "BUY"
	if isBuyerMaker {
		side = "SELL"
	}

	trade := &market.Trade{
		ID:        strconv.FormatInt(int64(tradeID), 10),
		Symbol:    strings.ToUpper(symbol),
		Price:     price,
		Quantity:  quantity,
		Side:      side,
		Timestamp: time.UnixMilli(int64(timestamp)),
	}

	return trade, nil
}

// parseKlineMessage parses Binance kline message
func (c *WSClient) parseKlineMessage(data map[string]interface{}) (*market.Kline, error) {
	klineData, ok := data["k"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing kline data")
	}

	symbol, ok := klineData["s"].(string)
	if !ok {
		return nil, fmt.Errorf("missing symbol")
	}

	interval, ok := klineData["i"].(string)
	if !ok {
		return nil, fmt.Errorf("missing interval")
	}

	openTime, ok := klineData["t"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing open time")
	}

	closeTime, ok := klineData["T"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing close time")
	}

	openStr, ok := klineData["o"].(string)
	if !ok {
		return nil, fmt.Errorf("missing open price")
	}

	highStr, ok := klineData["h"].(string)
	if !ok {
		return nil, fmt.Errorf("missing high price")
	}

	lowStr, ok := klineData["l"].(string)
	if !ok {
		return nil, fmt.Errorf("missing low price")
	}

	closeStr, ok := klineData["c"].(string)
	if !ok {
		return nil, fmt.Errorf("missing close price")
	}

	volumeStr, ok := klineData["v"].(string)
	if !ok {
		return nil, fmt.Errorf("missing volume")
	}

	isClosed, ok := klineData["x"].(bool)
	if !ok {
		return nil, fmt.Errorf("missing closed flag")
	}

	open, err := strconv.ParseFloat(openStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid open price: %w", err)
	}

	high, err := strconv.ParseFloat(highStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid high price: %w", err)
	}

	low, err := strconv.ParseFloat(lowStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid low price: %w", err)
	}

	close, err := strconv.ParseFloat(closeStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid close price: %w", err)
	}

	volume, err := strconv.ParseFloat(volumeStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid volume: %w", err)
	}

	kline := &market.Kline{
		Symbol:    strings.ToUpper(symbol),
		Interval:  interval,
		OpenTime:  time.UnixMilli(int64(openTime)),
		CloseTime: time.UnixMilli(int64(closeTime)),
		Open:      open,
		High:      high,
		Low:       low,
		Close:     close,
		Volume:    volume,
		Complete:  isClosed,
	}

	return kline, nil
}

// parseTickerMessage parses Binance ticker message
func (c *WSClient) parseTickerMessage(data map[string]interface{}) (*market.Ticker, error) {
	symbol, ok := data["s"].(string)
	if !ok {
		return nil, fmt.Errorf("missing symbol")
	}

	priceChangeStr, ok := data["p"].(string)
	if !ok {
		return nil, fmt.Errorf("missing price change")
	}

	priceChangePercentStr, ok := data["P"].(string)
	if !ok {
		return nil, fmt.Errorf("missing price change percent")
	}

	lastPriceStr, ok := data["c"].(string)
	if !ok {
		return nil, fmt.Errorf("missing last price")
	}

	bidPriceStr, ok := data["b"].(string)
	if !ok {
		return nil, fmt.Errorf("missing bid price")
	}

	askPriceStr, ok := data["a"].(string)
	if !ok {
		return nil, fmt.Errorf("missing ask price")
	}

	openPriceStr, ok := data["o"].(string)
	if !ok {
		return nil, fmt.Errorf("missing open price")
	}

	highPriceStr, ok := data["h"].(string)
	if !ok {
		return nil, fmt.Errorf("missing high price")
	}

	lowPriceStr, ok := data["l"].(string)
	if !ok {
		return nil, fmt.Errorf("missing low price")
	}

	volumeStr, ok := data["v"].(string)
	if !ok {
		return nil, fmt.Errorf("missing volume")
	}

	count, ok := data["c"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing count")
	}

	priceChange, _ := strconv.ParseFloat(priceChangeStr, 64)
	priceChangePercent, _ := strconv.ParseFloat(priceChangePercentStr, 64)
	lastPrice, _ := strconv.ParseFloat(lastPriceStr, 64)
	bidPrice, _ := strconv.ParseFloat(bidPriceStr, 64)
	askPrice, _ := strconv.ParseFloat(askPriceStr, 64)
	openPrice, _ := strconv.ParseFloat(openPriceStr, 64)
	highPrice, _ := strconv.ParseFloat(highPriceStr, 64)
	lowPrice, _ := strconv.ParseFloat(lowPriceStr, 64)
	volume, _ := strconv.ParseFloat(volumeStr, 64)

	ticker := &market.Ticker{
		Symbol:             strings.ToUpper(symbol),
		PriceChange:        priceChange,
		PriceChangePercent: priceChangePercent,
		LastPrice:          lastPrice,
		BidPrice:           bidPrice,
		AskPrice:           askPrice,
		OpenPrice:          openPrice,
		HighPrice:          highPrice,
		LowPrice:           lowPrice,
		Volume:             volume,
		Count:              int64(count),
		OpenTime:           time.Now().Add(-24 * time.Hour), // Approximate
		CloseTime:          time.Now(),
	}

	return ticker, nil
}