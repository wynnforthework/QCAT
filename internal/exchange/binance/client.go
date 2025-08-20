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
