package binance

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"qcat/internal/exchange"
	"qcat/internal/types"
)

// Binance API endpoints
const (
	BaseSpotURL    = "https://api.binance.com"
	BaseFuturesURL = "https://fapi.binance.com"
	BaseTestnetURL = "https://testnet.binancefuture.com"
)

// Client represents a Binance API client
type Client struct {
	*exchange.BaseExchange
	config      *exchange.ExchangeConfig
	baseURL     string
	httpClient  *http.Client
	rateLimiter *exchange.RateLimiter
}

// NewClient creates a new Binance client
func NewClient(config *exchange.ExchangeConfig, rateLimiter *exchange.RateLimiter) *Client {
	baseURL := BaseFuturesURL
	if config.TestNet {
		baseURL = BaseTestnetURL
	}

	return &Client{
		BaseExchange: exchange.NewBaseExchange(config),
		config:       config,
		baseURL:      baseURL,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		rateLimiter:  rateLimiter,
	}
}

// signRequest signs the request with HMAC SHA256
func (c *Client) signRequest(params url.Values) string {
	query := params.Encode()
	h := hmac.New(sha256.New, []byte(c.config.APISecret))
	h.Write([]byte(query))
	return hex.EncodeToString(h.Sum(nil))
}

// makeRequest makes an authenticated request to Binance API
func (c *Client) makeRequest(ctx context.Context, method, endpoint string, params url.Values) ([]byte, error) {
	// Add timestamp
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))

	// Sign the request
	signature := c.signRequest(params)
	params.Set("signature", signature)

	// Build URL
	fullURL := c.baseURL + endpoint + "?" + params.Encode()

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, fullURL, nil)
	if err != nil {
		return nil, err
	}

	// Add headers
	req.Header.Set("X-MBX-APIKEY", c.config.APIKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// GetServerTime implements exchange.Exchange
func (c *Client) GetServerTime(ctx context.Context) (time.Time, error) {
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx, "server_time"); err != nil {
			return time.Time{}, err
		}
	}

	// Simple HTTP request to get server time (no authentication needed)
	resp, err := c.httpClient.Get(c.baseURL + "/fapi/v1/time")
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get server time: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return time.Time{}, fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	var result struct {
		ServerTime int64 `json:"serverTime"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return time.Time{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return time.Unix(0, result.ServerTime*int64(time.Millisecond)), nil
}

// GetAccountBalance implements exchange.Exchange
func (c *Client) GetAccountBalance(ctx context.Context) (map[string]*exchange.AccountBalance, error) {
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx, "account"); err != nil {
			return nil, err
		}
	}

	params := url.Values{}
	body, err := c.makeRequest(ctx, "GET", "/fapi/v2/account", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get account balance: %w", err)
	}

	var result struct {
		Assets []struct {
			Asset            string `json:"asset"`
			WalletBalance    string `json:"walletBalance"`
			AvailableBalance string `json:"availableBalance"`
		} `json:"assets"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	balances := make(map[string]*exchange.AccountBalance)
	for _, asset := range result.Assets {
		balance, _ := strconv.ParseFloat(asset.WalletBalance, 64)
		available, _ := strconv.ParseFloat(asset.AvailableBalance, 64)

		balances[asset.Asset] = &exchange.AccountBalance{
			Asset:     asset.Asset,
			Total:     balance,
			Available: available,
			Locked:    balance - available,
		}
	}

	return balances, nil
}

// GetPositions implements exchange.Exchange
func (c *Client) GetPositions(ctx context.Context) ([]*exchange.Position, error) {
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx, "positions"); err != nil {
			return nil, err
		}
	}

	params := url.Values{}
	body, err := c.makeRequest(ctx, "GET", "/fapi/v2/positionRisk", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get positions: %w", err)
	}

	var result []struct {
		Symbol           string `json:"symbol"`
		PositionAmt      string `json:"positionAmt"`
		EntryPrice       string `json:"entryPrice"`
		MarkPrice        string `json:"markPrice"`
		UnrealizedProfit string `json:"unRealizedProfit"`
		Leverage         string `json:"leverage"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var positions []*exchange.Position
	for _, pos := range result {
		size, _ := strconv.ParseFloat(pos.PositionAmt, 64)
		if size == 0 {
			continue // Skip empty positions
		}

		entryPrice, _ := strconv.ParseFloat(pos.EntryPrice, 64)
		markPrice, _ := strconv.ParseFloat(pos.MarkPrice, 64)
		unrealizedPnL, _ := strconv.ParseFloat(pos.UnrealizedProfit, 64)
		leverage, _ := strconv.ParseFloat(pos.Leverage, 64)

		side := "LONG"
		if size < 0 {
			side = "SHORT"
			size = -size // Make size positive
		}

		position := &exchange.Position{
			Symbol:        pos.Symbol,
			Side:          side,
			Size:          size,
			EntryPrice:    entryPrice,
			MarkPrice:     markPrice,
			UnrealizedPnL: unrealizedPnL,
			Leverage:      int(leverage),
			MarginType:    string(exchange.MarginTypeCross),
			UpdatedAt:     time.Now(),
		}

		positions = append(positions, position)
	}

	return positions, nil
}

// GetPosition implements exchange.Exchange
func (c *Client) GetPosition(ctx context.Context, symbol string) (*exchange.Position, error) {
	positions, err := c.GetPositions(ctx)
	if err != nil {
		return nil, err
	}

	for _, position := range positions {
		if position.Symbol == symbol {
			return position, nil
		}
	}

	return nil, fmt.Errorf("position not found: %s", symbol)
}

// GetPositionByID implements exchange.Exchange
func (c *Client) GetPositionByID(ctx context.Context, positionID string) (*exchange.Position, error) {
	// For Binance, position ID is typically the symbol
	return c.GetPosition(ctx, positionID)
}

// GetExchangeInfo implements exchange.Exchange
func (c *Client) GetExchangeInfo(ctx context.Context) (*exchange.ExchangeInfo, error) {
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx, "exchange_info"); err != nil {
			return nil, err
		}
	}

	// Simple HTTP request to get exchange info (no authentication needed)
	resp, err := c.httpClient.Get(c.baseURL + "/fapi/v1/exchangeInfo")
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	var result struct {
		ServerTime int64  `json:"serverTime"`
		Timezone   string `json:"timezone"`
		Symbols    []struct {
			Symbol     string `json:"symbol"`
			BaseAsset  string `json:"baseAsset"`
			QuoteAsset string `json:"quoteAsset"`
			Status     string `json:"status"`
		} `json:"symbols"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to our exchange info format
	exchangeInfo := &exchange.ExchangeInfo{
		Name:       "binance",
		ServerTime: time.Unix(0, result.ServerTime*int64(time.Millisecond)),
		Timezone:   result.Timezone,
		Symbols:    make([]exchange.SymbolInfo, len(result.Symbols)),
	}

	// Convert symbols
	for i, symbol := range result.Symbols {
		exchangeInfo.Symbols[i] = exchange.SymbolInfo{
			Symbol:     symbol.Symbol,
			BaseAsset:  symbol.BaseAsset,
			QuoteAsset: symbol.QuoteAsset,
			Status:     symbol.Status,
		}
	}

	return exchangeInfo, nil
}

// GetSymbolInfo implements exchange.Exchange
func (c *Client) GetSymbolInfo(ctx context.Context, symbol string) (*exchange.SymbolInfo, error) {
	exchangeInfo, err := c.GetExchangeInfo(ctx)
	if err != nil {
		return nil, err
	}

	for _, s := range exchangeInfo.Symbols {
		if s.Symbol == symbol {
			return &s, nil
		}
	}

	return nil, fmt.Errorf("symbol not found: %s", symbol)
}

// GetSymbolPrice implements exchange.Exchange
func (c *Client) GetSymbolPrice(ctx context.Context, symbol string) (float64, error) {
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx, "get_symbol_price"); err != nil {
			return 0, err
		}
	}

	// Simple HTTP request to get symbol price (no authentication needed)
	resp, err := c.httpClient.Get(c.baseURL + "/fapi/v1/ticker/price?symbol=" + symbol)
	if err != nil {
		return 0, fmt.Errorf("failed to get symbol price: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	var result struct {
		Symbol string `json:"symbol"`
		Price  string `json:"price"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	price, err := strconv.ParseFloat(result.Price, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse price: %w", err)
	}

	return price, nil
}

// GetKlines fetches historical kline data from Binance API
func (c *Client) GetKlines(ctx context.Context, symbol, interval string, startTime, endTime time.Time, limit int) ([]*types.Kline, error) {
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx, "klines"); err != nil {
			return nil, err
		}
	}

	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("interval", interval)
	params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}

	// Use public endpoint for klines (no authentication needed)
	fullURL := c.baseURL + "/fapi/v1/klines?" + params.Encode()
	resp, err := c.httpClient.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get klines: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	var result [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	klines := make([]*types.Kline, 0, len(result))
	for _, item := range result {
		if len(item) < 11 {
			continue
		}

		openTime, _ := item[0].(float64)
		open, _ := strconv.ParseFloat(item[1].(string), 64)
		high, _ := strconv.ParseFloat(item[2].(string), 64)
		low, _ := strconv.ParseFloat(item[3].(string), 64)
		close, _ := strconv.ParseFloat(item[4].(string), 64)
		volume, _ := strconv.ParseFloat(item[5].(string), 64)
		closeTime, _ := item[6].(float64)

		kline := &types.Kline{
			Symbol:    symbol,
			Interval:  interval,
			OpenTime:  time.Unix(0, int64(openTime)*int64(time.Millisecond)),
			CloseTime: time.Unix(0, int64(closeTime)*int64(time.Millisecond)),
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
			Complete:  true,
		}
		klines = append(klines, kline)
	}

	return klines, nil
}

// GetTrades fetches historical trade data from Binance API
func (c *Client) GetTrades(ctx context.Context, symbol string, limit int) ([]*types.Trade, error) {
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx, "trades"); err != nil {
			return nil, err
		}
	}

	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}

	// Use public endpoint for recent trades (no authentication needed)
	fullURL := c.baseURL + "/fapi/v1/aggTrades?" + params.Encode()
	resp, err := c.httpClient.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get trades: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	var result []struct {
		ID           int64  `json:"a"`
		Price        string `json:"p"`
		Quantity     string `json:"q"`
		FirstTradeID int64  `json:"f"`
		LastTradeID  int64  `json:"l"`
		Timestamp    int64  `json:"T"`
		IsBuyerMaker bool   `json:"m"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	trades := make([]*types.Trade, 0, len(result))
	for _, item := range result {
		price, _ := strconv.ParseFloat(item.Price, 64)
		quantity, _ := strconv.ParseFloat(item.Quantity, 64)

		side := "BUY"
		if item.IsBuyerMaker {
			side = "SELL"
		}

		trade := &types.Trade{
			ID:        strconv.FormatInt(item.ID, 10),
			Symbol:    symbol,
			Price:     price,
			Quantity:  quantity,
			Side:      side,
			Timestamp: time.Unix(0, item.Timestamp*int64(time.Millisecond)),
		}
		trades = append(trades, trade)
	}

	return trades, nil
}

// GetOrderBook fetches current order book from Binance API
func (c *Client) GetOrderBook(ctx context.Context, symbol string, limit int) (*types.OrderBook, error) {
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx, "orderbook"); err != nil {
			return nil, err
		}
	}

	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}

	// Use public endpoint for order book (no authentication needed)
	fullURL := c.baseURL + "/fapi/v1/depth?" + params.Encode()
	resp, err := c.httpClient.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get order book: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	var result struct {
		LastUpdateID int64      `json:"lastUpdateId"`
		Bids         [][]string `json:"bids"`
		Asks         [][]string `json:"asks"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	orderBook := &types.OrderBook{
		Symbol:    symbol,
		UpdatedAt: time.Now(),
	}

	// Parse bids
	for _, bid := range result.Bids {
		if len(bid) >= 2 {
			price, _ := strconv.ParseFloat(bid[0], 64)
			quantity, _ := strconv.ParseFloat(bid[1], 64)
			orderBook.Bids = append(orderBook.Bids, types.Level{
				Price:    price,
				Quantity: quantity,
			})
		}
	}

	// Parse asks
	for _, ask := range result.Asks {
		if len(ask) >= 2 {
			price, _ := strconv.ParseFloat(ask[0], 64)
			quantity, _ := strconv.ParseFloat(ask[1], 64)
			orderBook.Asks = append(orderBook.Asks, types.Level{
				Price:    price,
				Quantity: quantity,
			})
		}
	}

	return orderBook, nil
}

// GetFundingRate fetches current funding rate from Binance API
func (c *Client) GetFundingRate(ctx context.Context, symbol string) (*types.FundingRate, error) {
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx, "funding_rate"); err != nil {
			return nil, err
		}
	}

	params := url.Values{}
	params.Set("symbol", symbol)

	// Use public endpoint for funding rate (no authentication needed)
	fullURL := c.baseURL + "/fapi/v1/premiumIndex?" + params.Encode()
	resp, err := c.httpClient.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get funding rate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	var result struct {
		Symbol               string `json:"symbol"`
		MarkPrice            string `json:"markPrice"`
		IndexPrice           string `json:"indexPrice"`
		EstimatedSettlePrice string `json:"estimatedSettlePrice"`
		LastFundingRate      string `json:"lastFundingRate"`
		NextFundingTime      int64  `json:"nextFundingTime"`
		InterestRate         string `json:"interestRate"`
		Time                 int64  `json:"time"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	rate, _ := strconv.ParseFloat(result.LastFundingRate, 64)
	nextRate, _ := strconv.ParseFloat(result.InterestRate, 64)

	fundingRate := &types.FundingRate{
		Symbol:      symbol,
		Rate:        rate,
		NextRate:    nextRate,
		NextTime:    time.Unix(0, result.NextFundingTime*int64(time.Millisecond)),
		LastUpdated: time.Unix(0, result.Time*int64(time.Millisecond)),
	}

	return fundingRate, nil
}

// GetOpenInterest fetches current open interest from Binance API
func (c *Client) GetOpenInterest(ctx context.Context, symbol string) (*types.OpenInterest, error) {
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx, "open_interest"); err != nil {
			return nil, err
		}
	}

	params := url.Values{}
	params.Set("symbol", symbol)

	// Use public endpoint for open interest (no authentication needed)
	fullURL := c.baseURL + "/fapi/v1/openInterest?" + params.Encode()
	resp, err := c.httpClient.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get open interest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	var result struct {
		OpenInterest string `json:"openInterest"`
		Symbol       string `json:"symbol"`
		Time         int64  `json:"time"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	value, _ := strconv.ParseFloat(result.OpenInterest, 64)

	openInterest := &types.OpenInterest{
		Symbol:    symbol,
		Value:     value,
		Notional:  0, // Would need additional API call to get notional value
		Timestamp: time.Unix(0, result.Time*int64(time.Millisecond)),
	}

	return openInterest, nil
}

// GetLeverage gets the current leverage for a symbol
func (c *Client) GetLeverage(ctx context.Context, symbol string) (int, error) {
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx, "get_leverage"); err != nil {
			return 0, err
		}
	}

	params := url.Values{}
	params.Set("symbol", symbol)

	body, err := c.makeRequest(ctx, "GET", "/fapi/v2/positionRisk", params)
	if err != nil {
		return 0, fmt.Errorf("failed to get leverage: %w", err)
	}

	var result []struct {
		Symbol   string `json:"symbol"`
		Leverage string `json:"leverage"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	for _, pos := range result {
		if pos.Symbol == symbol {
			leverage, err := strconv.Atoi(pos.Leverage)
			if err != nil {
				return 0, fmt.Errorf("failed to parse leverage: %w", err)
			}
			return leverage, nil
		}
	}

	return 0, fmt.Errorf("symbol not found: %s", symbol)
}

// SetLeverage sets the leverage for a symbol
func (c *Client) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx, "set_leverage"); err != nil {
			return err
		}
	}

	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("leverage", strconv.Itoa(leverage))

	_, err := c.makeRequest(ctx, "POST", "/fapi/v1/leverage", params)
	if err != nil {
		return fmt.Errorf("failed to set leverage: %w", err)
	}

	return nil
}

// SetMarginType sets the margin type for a symbol
func (c *Client) SetMarginType(ctx context.Context, symbol string, marginType exchange.MarginType) error {
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx, "set_margin_type"); err != nil {
			return err
		}
	}

	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("marginType", string(marginType))

	_, err := c.makeRequest(ctx, "POST", "/fapi/v1/marginType", params)
	if err != nil {
		return fmt.Errorf("failed to set margin type: %w", err)
	}

	return nil
}

// PlaceOrder places a new order
func (c *Client) PlaceOrder(ctx context.Context, req *exchange.OrderRequest) (*exchange.OrderResponse, error) {
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx, "place_order"); err != nil {
			return nil, err
		}
	}

	params := url.Values{}
	params.Set("symbol", req.Symbol)
	params.Set("side", req.Side)
	params.Set("type", req.Type)
	params.Set("quantity", strconv.FormatFloat(req.Quantity, 'f', -1, 64))

	if req.Price > 0 {
		params.Set("price", strconv.FormatFloat(req.Price, 'f', -1, 64))
	}
	if req.StopPrice > 0 {
		params.Set("stopPrice", strconv.FormatFloat(req.StopPrice, 'f', -1, 64))
	}
	if req.TimeInForce != "" {
		params.Set("timeInForce", req.TimeInForce)
	}
	if req.ReduceOnly {
		params.Set("reduceOnly", "true")
	}
	if req.ClientOrderID != "" {
		params.Set("newClientOrderId", req.ClientOrderID)
	}

	body, err := c.makeRequest(ctx, "POST", "/fapi/v1/order", params)
	if err != nil {
		return nil, fmt.Errorf("failed to place order: %w", err)
	}

	var result struct {
		OrderID       int64  `json:"orderId"`
		Symbol        string `json:"symbol"`
		Status        string `json:"status"`
		ClientOrderID string `json:"clientOrderId"`
		Price         string `json:"price"`
		OrigQty       string `json:"origQty"`
		ExecutedQty   string `json:"executedQty"`
		CumQuote      string `json:"cumQuote"`
		TimeInForce   string `json:"timeInForce"`
		Type          string `json:"type"`
		Side          string `json:"side"`
		UpdateTime    int64  `json:"updateTime"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	price, _ := strconv.ParseFloat(result.Price, 64)
	quantity, _ := strconv.ParseFloat(result.OrigQty, 64)
	executedQty, _ := strconv.ParseFloat(result.ExecutedQty, 64)
	cumulativeQuoteQty, _ := strconv.ParseFloat(result.CumQuote, 64)

	return &exchange.OrderResponse{
		OrderID:            strconv.FormatInt(result.OrderID, 10),
		ClientOrderID:      result.ClientOrderID,
		Symbol:             result.Symbol,
		Status:             result.Status,
		Side:               result.Side,
		Type:               result.Type,
		Quantity:           quantity,
		Price:              price,
		ExecutedQty:        executedQty,
		CumulativeQuoteQty: cumulativeQuoteQty,
		TimeInForce:        result.TimeInForce,
		UpdatedTime:        time.Unix(0, result.UpdateTime*int64(time.Millisecond)),
		Success:            true,
	}, nil
}

// CancelOrder cancels an existing order
func (c *Client) CancelOrder(ctx context.Context, req *exchange.OrderCancelRequest) (*exchange.OrderResponse, error) {
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx, "cancel_order"); err != nil {
			return nil, err
		}
	}

	params := url.Values{}
	params.Set("symbol", req.Symbol)

	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}
	if req.ClientOrderID != "" {
		params.Set("origClientOrderId", req.ClientOrderID)
	}

	body, err := c.makeRequest(ctx, "DELETE", "/fapi/v1/order", params)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel order: %w", err)
	}

	var result struct {
		OrderID       int64  `json:"orderId"`
		Symbol        string `json:"symbol"`
		Status        string `json:"status"`
		ClientOrderID string `json:"clientOrderId"`
		Price         string `json:"price"`
		OrigQty       string `json:"origQty"`
		ExecutedQty   string `json:"executedQty"`
		CumQuote      string `json:"cumQuote"`
		TimeInForce   string `json:"timeInForce"`
		Type          string `json:"type"`
		Side          string `json:"side"`
		UpdateTime    int64  `json:"updateTime"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	price, _ := strconv.ParseFloat(result.Price, 64)
	quantity, _ := strconv.ParseFloat(result.OrigQty, 64)
	executedQty, _ := strconv.ParseFloat(result.ExecutedQty, 64)
	cumulativeQuoteQty, _ := strconv.ParseFloat(result.CumQuote, 64)

	return &exchange.OrderResponse{
		OrderID:            strconv.FormatInt(result.OrderID, 10),
		ClientOrderID:      result.ClientOrderID,
		Symbol:             result.Symbol,
		Status:             result.Status,
		Side:               result.Side,
		Type:               result.Type,
		Quantity:           quantity,
		Price:              price,
		ExecutedQty:        executedQty,
		CumulativeQuoteQty: cumulativeQuoteQty,
		TimeInForce:        result.TimeInForce,
		UpdatedTime:        time.Unix(0, result.UpdateTime*int64(time.Millisecond)),
		Success:            true,
	}, nil
}

// CancelAllOrders cancels all open orders for a symbol
func (c *Client) CancelAllOrders(ctx context.Context, symbol string) error {
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx, "cancel_all_orders"); err != nil {
			return err
		}
	}

	params := url.Values{}
	params.Set("symbol", symbol)

	_, err := c.makeRequest(ctx, "DELETE", "/fapi/v1/allOpenOrders", params)
	if err != nil {
		return fmt.Errorf("failed to cancel all orders: %w", err)
	}

	return nil
}

// GetOrder gets a specific order
func (c *Client) GetOrder(ctx context.Context, symbol, orderID string) (*exchange.Order, error) {
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx, "get_order"); err != nil {
			return nil, err
		}
	}

	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("orderId", orderID)

	body, err := c.makeRequest(ctx, "GET", "/fapi/v1/order", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	var result struct {
		OrderID       int64  `json:"orderId"`
		Symbol        string `json:"symbol"`
		Status        string `json:"status"`
		ClientOrderID string `json:"clientOrderId"`
		Price         string `json:"price"`
		AvgPrice      string `json:"avgPrice"`
		OrigQty       string `json:"origQty"`
		ExecutedQty   string `json:"executedQty"`
		CumQuote      string `json:"cumQuote"`
		TimeInForce   string `json:"timeInForce"`
		Type          string `json:"type"`
		Side          string `json:"side"`
		Time          int64  `json:"time"`
		UpdateTime    int64  `json:"updateTime"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	price, _ := strconv.ParseFloat(result.Price, 64)
	avgPrice, _ := strconv.ParseFloat(result.AvgPrice, 64)
	quantity, _ := strconv.ParseFloat(result.OrigQty, 64)
	executedQty, _ := strconv.ParseFloat(result.ExecutedQty, 64)
	cumulativeQuoteQty, _ := strconv.ParseFloat(result.CumQuote, 64)

	return &exchange.Order{
		OrderID:            strconv.FormatInt(result.OrderID, 10),
		ClientOrderID:      result.ClientOrderID,
		Symbol:             result.Symbol,
		Status:             result.Status,
		Side:               result.Side,
		Type:               result.Type,
		Quantity:           quantity,
		Price:              price,
		ExecutedQty:        executedQty,
		CumulativeQuoteQty: cumulativeQuoteQty,
		TimeInForce:        result.TimeInForce,
		Time:               time.Unix(0, result.Time*int64(time.Millisecond)),
		UpdatedTime:        time.Unix(0, result.UpdateTime*int64(time.Millisecond)),
		AvgPrice:           avgPrice,
		FilledQty:          executedQty,
		RemainingQty:       quantity - executedQty,
	}, nil
}

// GetOpenOrders gets all open orders for a symbol
func (c *Client) GetOpenOrders(ctx context.Context, symbol string) ([]*exchange.Order, error) {
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx, "get_open_orders"); err != nil {
			return nil, err
		}
	}

	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}

	body, err := c.makeRequest(ctx, "GET", "/fapi/v1/openOrders", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get open orders: %w", err)
	}

	var result []struct {
		OrderID       int64  `json:"orderId"`
		Symbol        string `json:"symbol"`
		Status        string `json:"status"`
		ClientOrderID string `json:"clientOrderId"`
		Price         string `json:"price"`
		AvgPrice      string `json:"avgPrice"`
		OrigQty       string `json:"origQty"`
		ExecutedQty   string `json:"executedQty"`
		CumQuote      string `json:"cumQuote"`
		TimeInForce   string `json:"timeInForce"`
		Type          string `json:"type"`
		Side          string `json:"side"`
		Time          int64  `json:"time"`
		UpdateTime    int64  `json:"updateTime"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	orders := make([]*exchange.Order, 0, len(result))
	for _, item := range result {
		price, _ := strconv.ParseFloat(item.Price, 64)
		avgPrice, _ := strconv.ParseFloat(item.AvgPrice, 64)
		quantity, _ := strconv.ParseFloat(item.OrigQty, 64)
		executedQty, _ := strconv.ParseFloat(item.ExecutedQty, 64)
		cumulativeQuoteQty, _ := strconv.ParseFloat(item.CumQuote, 64)

		order := &exchange.Order{
			OrderID:            strconv.FormatInt(item.OrderID, 10),
			ClientOrderID:      item.ClientOrderID,
			Symbol:             item.Symbol,
			Status:             item.Status,
			Side:               item.Side,
			Type:               item.Type,
			Quantity:           quantity,
			Price:              price,
			ExecutedQty:        executedQty,
			CumulativeQuoteQty: cumulativeQuoteQty,
			TimeInForce:        item.TimeInForce,
			Time:               time.Unix(0, item.Time*int64(time.Millisecond)),
			UpdatedTime:        time.Unix(0, item.UpdateTime*int64(time.Millisecond)),
			AvgPrice:           avgPrice,
			FilledQty:          executedQty,
			RemainingQty:       quantity - executedQty,
		}
		orders = append(orders, order)
	}

	return orders, nil
}

// GetOrderHistory gets order history for a symbol within a time range
func (c *Client) GetOrderHistory(ctx context.Context, symbol string, startTime, endTime time.Time) ([]*exchange.Order, error) {
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx, "get_order_history"); err != nil {
			return nil, err
		}
	}

	params := url.Values{}
	params.Set("symbol", symbol)
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}

	body, err := c.makeRequest(ctx, "GET", "/fapi/v1/allOrders", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get order history: %w", err)
	}

	var result []struct {
		OrderID       int64  `json:"orderId"`
		Symbol        string `json:"symbol"`
		Status        string `json:"status"`
		ClientOrderID string `json:"clientOrderId"`
		Price         string `json:"price"`
		AvgPrice      string `json:"avgPrice"`
		OrigQty       string `json:"origQty"`
		ExecutedQty   string `json:"executedQty"`
		CumQuote      string `json:"cumQuote"`
		TimeInForce   string `json:"timeInForce"`
		Type          string `json:"type"`
		Side          string `json:"side"`
		Time          int64  `json:"time"`
		UpdateTime    int64  `json:"updateTime"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	orders := make([]*exchange.Order, 0, len(result))
	for _, item := range result {
		price, _ := strconv.ParseFloat(item.Price, 64)
		avgPrice, _ := strconv.ParseFloat(item.AvgPrice, 64)
		quantity, _ := strconv.ParseFloat(item.OrigQty, 64)
		executedQty, _ := strconv.ParseFloat(item.ExecutedQty, 64)
		cumulativeQuoteQty, _ := strconv.ParseFloat(item.CumQuote, 64)

		order := &exchange.Order{
			OrderID:            strconv.FormatInt(item.OrderID, 10),
			ClientOrderID:      item.ClientOrderID,
			Symbol:             item.Symbol,
			Status:             item.Status,
			Side:               item.Side,
			Type:               item.Type,
			Quantity:           quantity,
			Price:              price,
			ExecutedQty:        executedQty,
			CumulativeQuoteQty: cumulativeQuoteQty,
			TimeInForce:        item.TimeInForce,
			Time:               time.Unix(0, item.Time*int64(time.Millisecond)),
			UpdatedTime:        time.Unix(0, item.UpdateTime*int64(time.Millisecond)),
			AvgPrice:           avgPrice,
			FilledQty:          executedQty,
			RemainingQty:       quantity - executedQty,
		}
		orders = append(orders, order)
	}

	return orders, nil
}

// GetMarginInfo gets margin information for the account
func (c *Client) GetMarginInfo(ctx context.Context) (*exchange.MarginInfo, error) {
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx, "get_margin_info"); err != nil {
			return nil, err
		}
	}

	// Get account information which includes margin details
	params := url.Values{}
	body, err := c.makeRequest(ctx, "GET", "/fapi/v2/account", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get account info for margin: %w", err)
	}

	var result struct {
		TotalWalletBalance    string `json:"totalWalletBalance"`
		TotalUnrealizedProfit string `json:"totalUnrealizedProfit"`
		TotalMarginBalance    string `json:"totalMarginBalance"`
		TotalMaintMargin      string `json:"totalMaintMargin"`
		TotalInitialMargin    string `json:"totalInitialMargin"`
		AvailableBalance      string `json:"availableBalance"`
		MaxWithdrawAmount     string `json:"maxWithdrawAmount"`
		Assets                []struct {
			Asset                  string `json:"asset"`
			WalletBalance          string `json:"walletBalance"`
			UnrealizedProfit       string `json:"unrealizedProfit"`
			MarginBalance          string `json:"marginBalance"`
			MaintMargin            string `json:"maintMargin"`
			InitialMargin          string `json:"initialMargin"`
			PositionInitialMargin  string `json:"positionInitialMargin"`
			OpenOrderInitialMargin string `json:"openOrderInitialMargin"`
			CrossWalletBalance     string `json:"crossWalletBalance"`
			CrossUnPnl             string `json:"crossUnPnl"`
			AvailableBalance       string `json:"availableBalance"`
			MaxWithdrawAmount      string `json:"maxWithdrawAmount"`
		} `json:"assets"`
		Positions []struct {
			Symbol                 string `json:"symbol"`
			InitialMargin          string `json:"initialMargin"`
			MaintMargin            string `json:"maintMargin"`
			UnrealizedProfit       string `json:"unrealizedProfit"`
			PositionInitialMargin  string `json:"positionInitialMargin"`
			OpenOrderInitialMargin string `json:"openOrderInitialMargin"`
			Leverage               string `json:"leverage"`
			Isolated               bool   `json:"isolated"`
			EntryPrice             string `json:"entryPrice"`
			MaxNotional            string `json:"maxNotional"`
			BidNotional            string `json:"bidNotional"`
			AskNotional            string `json:"askNotional"`
			PositionSide           string `json:"positionSide"`
			PositionAmt            string `json:"positionAmt"`
			UpdateTime             int64  `json:"updateTime"`
		} `json:"positions"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode margin info response: %w", err)
	}

	// Parse the margin information
	totalWalletBalance, _ := strconv.ParseFloat(result.TotalWalletBalance, 64)
	totalUnrealizedProfit, _ := strconv.ParseFloat(result.TotalUnrealizedProfit, 64)
	totalMarginBalance, _ := strconv.ParseFloat(result.TotalMarginBalance, 64)
	totalMaintMargin, _ := strconv.ParseFloat(result.TotalMaintMargin, 64)
	totalInitialMargin, _ := strconv.ParseFloat(result.TotalInitialMargin, 64)

	// Calculate total asset value (wallet balance + unrealized profit)
	totalAssetValue := totalWalletBalance + totalUnrealizedProfit

	// For futures, debt is essentially the margin requirement
	totalDebtValue := totalInitialMargin

	// Calculate margin ratio (asset value / margin requirement)
	marginRatio := 0.0
	if totalDebtValue > 0 {
		marginRatio = totalAssetValue / totalDebtValue
	}

	// Calculate maintenance margin ratio
	maintenanceMarginRatio := 0.0
	if totalMarginBalance > 0 {
		maintenanceMarginRatio = totalMaintMargin / totalMarginBalance
	}

	return &exchange.MarginInfo{
		TotalAssetValue:   totalAssetValue,
		TotalDebtValue:    totalDebtValue,
		MarginRatio:       marginRatio,
		MaintenanceMargin: maintenanceMarginRatio,
		MarginCallRatio:   1.5, // Standard margin call ratio
		LiquidationRatio:  1.0, // Standard liquidation ratio
		UpdatedAt:         time.Now(),
	}, nil
}

// GetRiskLimits gets risk limits for a symbol
func (c *Client) GetRiskLimits(ctx context.Context, symbol string) (*exchange.RiskLimits, error) {
	// Get exchange info to determine symbol limits
	exchangeInfo, err := c.GetExchangeInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange info for risk limits: %w", err)
	}

	// Find the symbol info
	var symbolInfo *exchange.SymbolInfo
	for _, info := range exchangeInfo.Symbols {
		if info.Symbol == symbol {
			symbolInfo = &info
			break
		}
	}

	if symbolInfo == nil {
		return nil, fmt.Errorf("symbol not found: %s", symbol)
	}

	// For Binance futures, provide reasonable default risk limits
	// These would typically come from the exchange's risk management API
	// but Binance doesn't expose all risk limits via public API
	return &exchange.RiskLimits{
		Symbol:           symbol,
		MaxLeverage:      125,     // Binance futures max leverage
		MaxPositionValue: 1000000, // 1M USD max position
		MaxOrderValue:    100000,  // 100K USD max order
		MinOrderValue:    5,       // 5 USD min order
		MaxOrderQty:      10000,   // Max order quantity
		MinOrderQty:      0.001,   // Min order quantity
	}, nil
}

// SetRiskLimits sets risk limits for a symbol
func (c *Client) SetRiskLimits(ctx context.Context, symbol string, limits *exchange.RiskLimits) error {
	// Binance doesn't provide a direct API to set custom risk limits
	// Risk limits are managed by Binance internally based on account tier and symbol
	// This method would typically be used for internal risk management
	// For now, we'll just validate the limits and return success

	if limits.MaxLeverage <= 0 || limits.MaxLeverage > 125 {
		return fmt.Errorf("invalid max leverage: %d (must be between 1 and 125)", limits.MaxLeverage)
	}

	if limits.MaxPositionValue <= 0 {
		return fmt.Errorf("invalid max position value: %f", limits.MaxPositionValue)
	}

	if limits.MaxOrderValue <= 0 {
		return fmt.Errorf("invalid max order value: %f", limits.MaxOrderValue)
	}

	if limits.MinOrderValue <= 0 {
		return fmt.Errorf("invalid min order value: %f", limits.MinOrderValue)
	}

	if limits.MaxOrderQty <= 0 {
		return fmt.Errorf("invalid max order quantity: %f", limits.MaxOrderQty)
	}

	if limits.MinOrderQty <= 0 {
		return fmt.Errorf("invalid min order quantity: %f", limits.MinOrderQty)
	}

	// Log the risk limits setting (since Binance doesn't support direct API setting)
	// In a real implementation, these would be stored in a local database for enforcement
	fmt.Printf("Risk limits set for %s: MaxLeverage=%d, MaxPositionValue=%.2f, MaxOrderValue=%.2f\n",
		symbol, limits.MaxLeverage, limits.MaxPositionValue, limits.MaxOrderValue)

	return nil
}
