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
	"strings"
	"time"

	"qcat/internal/market"
)

// Client represents a Binance API client
type Client struct {
	apiKey     string
	apiSecret  string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Binance API client
func NewClient(apiKey, apiSecret string, testnet bool) *Client {
	baseURL := "https://api.binance.com"
	if testnet {
		baseURL = "https://testnet.binance.vision"
	}

	return &Client{
		apiKey:    apiKey,
		apiSecret: apiSecret,
		baseURL:   baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetKlines fetches historical kline data
func (c *Client) GetKlines(ctx context.Context, symbol, interval string, startTime, endTime time.Time, limit int) ([]*market.Kline, error) {
	params := url.Values{}
	params.Set("symbol", strings.ToUpper(symbol))
	params.Set("interval", interval)
	params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}

	resp, err := c.makeRequest(ctx, "GET", "/api/v3/klines", params, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get klines: %w", err)
	}
	defer resp.Body.Close()

	var rawKlines [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawKlines); err != nil {
		return nil, fmt.Errorf("failed to decode klines response: %w", err)
	}

	klines := make([]*market.Kline, 0, len(rawKlines))
	for _, raw := range rawKlines {
		if len(raw) < 11 {
			continue
		}

		openTime, _ := raw[0].(float64)
		open, _ := strconv.ParseFloat(raw[1].(string), 64)
		high, _ := strconv.ParseFloat(raw[2].(string), 64)
		low, _ := strconv.ParseFloat(raw[3].(string), 64)
		close, _ := strconv.ParseFloat(raw[4].(string), 64)
		volume, _ := strconv.ParseFloat(raw[5].(string), 64)
		closeTime, _ := raw[6].(float64)

		kline := &market.Kline{
			Symbol:    symbol,
			Interval:  interval,
			OpenTime:  time.UnixMilli(int64(openTime)),
			CloseTime: time.UnixMilli(int64(closeTime)),
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

// GetTrades fetches historical trade data
func (c *Client) GetTrades(ctx context.Context, symbol string, limit int) ([]*market.Trade, error) {
	params := url.Values{}
	params.Set("symbol", strings.ToUpper(symbol))
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}

	resp, err := c.makeRequest(ctx, "GET", "/api/v3/trades", params, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get trades: %w", err)
	}
	defer resp.Body.Close()

	var rawTrades []struct {
		ID           int64  `json:"id"`
		Price        string `json:"price"`
		Qty          string `json:"qty"`
		QuoteQty     string `json:"quoteQty"`
		Time         int64  `json:"time"`
		IsBuyerMaker bool   `json:"isBuyerMaker"`
		IsBestMatch  bool   `json:"isBestMatch"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawTrades); err != nil {
		return nil, fmt.Errorf("failed to decode trades response: %w", err)
	}

	trades := make([]*market.Trade, 0, len(rawTrades))
	for _, raw := range rawTrades {
		price, _ := strconv.ParseFloat(raw.Price, 64)
		quantity, _ := strconv.ParseFloat(raw.Qty, 64)

		side := "BUY"
		if raw.IsBuyerMaker {
			side = "SELL"
		}

		trade := &market.Trade{
			ID:        strconv.FormatInt(raw.ID, 10),
			Symbol:    symbol,
			Price:     price,
			Quantity:  quantity,
			Side:      side,
			Timestamp: time.UnixMilli(raw.Time),
		}
		trades = append(trades, trade)
	}

	return trades, nil
}

// GetOrderBook fetches current order book
func (c *Client) GetOrderBook(ctx context.Context, symbol string, limit int) (*market.OrderBook, error) {
	params := url.Values{}
	params.Set("symbol", strings.ToUpper(symbol))
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}

	resp, err := c.makeRequest(ctx, "GET", "/api/v3/depth", params, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get order book: %w", err)
	}
	defer resp.Body.Close()

	var rawOrderBook struct {
		LastUpdateID int64      `json:"lastUpdateId"`
		Bids         [][]string `json:"bids"`
		Asks         [][]string `json:"asks"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawOrderBook); err != nil {
		return nil, fmt.Errorf("failed to decode order book response: %w", err)
	}

	orderBook := &market.OrderBook{
		Symbol:    symbol,
		UpdatedAt: time.Now(),
		Bids:      make([]market.Level, 0, len(rawOrderBook.Bids)),
		Asks:      make([]market.Level, 0, len(rawOrderBook.Asks)),
	}

	for _, bid := range rawOrderBook.Bids {
		if len(bid) >= 2 {
			price, _ := strconv.ParseFloat(bid[0], 64)
			quantity, _ := strconv.ParseFloat(bid[1], 64)
			orderBook.Bids = append(orderBook.Bids, market.Level{
				Price:    price,
				Quantity: quantity,
			})
		}
	}

	for _, ask := range rawOrderBook.Asks {
		if len(ask) >= 2 {
			price, _ := strconv.ParseFloat(ask[0], 64)
			quantity, _ := strconv.ParseFloat(ask[1], 64)
			orderBook.Asks = append(orderBook.Asks, market.Level{
				Price:    price,
				Quantity: quantity,
			})
		}
	}

	return orderBook, nil
}

// GetFundingRate fetches current funding rate for futures
func (c *Client) GetFundingRate(ctx context.Context, symbol string) (*market.FundingRate, error) {
	params := url.Values{}
	params.Set("symbol", strings.ToUpper(symbol))

	// Use futures API endpoint
	baseURL := strings.Replace(c.baseURL, "api.binance.com", "fapi.binance.com", 1)
	
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/fapi/v1/premiumIndex?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var rawFunding struct {
		Symbol               string `json:"symbol"`
		MarkPrice            string `json:"markPrice"`
		IndexPrice           string `json:"indexPrice"`
		EstimatedSettlePrice string `json:"estimatedSettlePrice"`
		LastFundingRate      string `json:"lastFundingRate"`
		NextFundingTime      int64  `json:"nextFundingTime"`
		InterestRate         string `json:"interestRate"`
		Time                 int64  `json:"time"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawFunding); err != nil {
		return nil, fmt.Errorf("failed to decode funding rate response: %w", err)
	}

	rate, _ := strconv.ParseFloat(rawFunding.LastFundingRate, 64)
	interestRate, _ := strconv.ParseFloat(rawFunding.InterestRate, 64)

	fundingRate := &market.FundingRate{
		Symbol:      symbol,
		Rate:        rate,
		NextRate:    interestRate, // Use interest rate as next rate approximation
		NextTime:    time.UnixMilli(rawFunding.NextFundingTime),
		LastUpdated: time.UnixMilli(rawFunding.Time),
	}

	return fundingRate, nil
}

// GetOpenInterest fetches open interest data for futures
func (c *Client) GetOpenInterest(ctx context.Context, symbol string) (*market.OpenInterest, error) {
	params := url.Values{}
	params.Set("symbol", strings.ToUpper(symbol))

	// Use futures API endpoint
	baseURL := strings.Replace(c.baseURL, "api.binance.com", "fapi.binance.com", 1)
	
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/fapi/v1/openInterest?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var rawOI struct {
		OpenInterest string `json:"openInterest"`
		Symbol       string `json:"symbol"`
		Time         int64  `json:"time"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawOI); err != nil {
		return nil, fmt.Errorf("failed to decode open interest response: %w", err)
	}

	value, _ := strconv.ParseFloat(rawOI.OpenInterest, 64)

	openInterest := &market.OpenInterest{
		Symbol:    symbol,
		Value:     value,
		Notional:  value, // Simplified - in real implementation, multiply by contract size and price
		Timestamp: time.UnixMilli(rawOI.Time),
	}

	return openInterest, nil
}

// makeRequest makes an HTTP request to Binance API
func (c *Client) makeRequest(ctx context.Context, method, endpoint string, params url.Values, signed bool) (*http.Response, error) {
	if signed {
		timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
		params.Set("timestamp", timestamp)

		// Create signature
		queryString := params.Encode()
		signature := c.sign(queryString)
		params.Set("signature", signature)
	}

	reqURL := c.baseURL + endpoint
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.apiKey != "" {
		req.Header.Set("X-MBX-APIKEY", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

// sign creates HMAC SHA256 signature
func (c *Client) sign(message string) string {
	h := hmac.New(sha256.New, []byte(c.apiSecret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}