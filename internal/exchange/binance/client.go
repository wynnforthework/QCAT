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

	"qcat/internal/exchange"
)

// Client represents a Binance API client
type Client struct {
	*exchange.BaseExchange
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
		baseURL:      baseURL,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		rateLimiter:  rateLimiter,
	}
}

// signRequest signs a request with the API secret
func (c *Client) signRequest(method, endpoint string, params url.Values) (*http.Request, error) {
	if params == nil {
		params = url.Values{}
	}

	// Add timestamp
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))

	// Create signature
	mac := hmac.New(sha256.New, []byte(c.Config().APISecret))
	mac.Write([]byte(params.Encode()))
	signature := hex.EncodeToString(mac.Sum(nil))
	params.Set("signature", signature)

	// Create request
	var body io.Reader
	if method == http.MethodPost || method == http.MethodPut || method == http.MethodDelete {
		body = strings.NewReader(params.Encode())
	} else {
		endpoint = fmt.Sprintf("%s?%s", endpoint, params.Encode())
	}

	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", c.baseURL, endpoint), body)
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Set("X-MBX-APIKEY", c.Config().APIKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	return req, nil
}

// doRequest executes an API request with rate limiting and retries
func (c *Client) doRequest(ctx context.Context, method, endpoint string, params url.Values, result interface{}) error {
	// Wait for rate limit
	if err := c.rateLimiter.WaitWithFallback(ctx, "binance", endpoint, 0, 0); err != nil {
		return err
	}

	// Create and sign request
	req, err := c.signRequest(method, endpoint, params)
	if err != nil {
		return err
	}

	// Execute request with retry
	var resp *http.Response
	err = exchange.WithRetry(ctx, func(ctx context.Context) error {
		var err error
		req = req.WithContext(ctx)
		resp, err = c.httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// Check for error response
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			var apiErr BinanceResponse
			if err := json.Unmarshal(body, &apiErr); err != nil {
				return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
			}
			return &exchange.Error{
				Code:    apiErr.Code,
				Message: apiErr.Message,
			}
		}

		// Parse response
		return json.NewDecoder(resp.Body).Decode(result)
	}, nil)

	return err
}

// GetExchangeInfo implements exchange.Exchange
func (c *Client) GetExchangeInfo(ctx context.Context) (*exchange.ExchangeInfo, error) {
	var info ExchangeInfo
	if err := c.doRequest(ctx, http.MethodGet, MethodExchangeInfo, nil, &info); err != nil {
		return nil, err
	}

	// Convert to common format
	result := &exchange.ExchangeInfo{
		Name:       c.Name(),
		ServerTime: time.Unix(0, info.ServerTime*int64(time.Millisecond)),
		RateLimits: make([]exchange.RateLimit, len(info.RateLimits)),
	}

	for i, limit := range info.RateLimits {
		result.RateLimits[i] = exchange.RateLimit{
			RateLimitType: limit.RateLimitType,
			Interval:      limit.Interval,
			IntervalNum:   limit.IntervalNum,
			Limit:         limit.Limit,
		}
	}

	result.Symbols = make([]exchange.SymbolInfo, len(info.Symbols))
	for i, symbol := range info.Symbols {
		result.Symbols[i] = exchange.SymbolInfo{
			Symbol: symbol.Symbol,
		}
	}

	return result, nil
}

// GetSymbolInfo implements exchange.Exchange
func (c *Client) GetSymbolInfo(ctx context.Context, symbol string) (*exchange.SymbolInfo, error) {
	var info ExchangeInfo
	if err := c.doRequest(ctx, http.MethodGet, MethodExchangeInfo, nil, &info); err != nil {
		return nil, err
	}

	// Find symbol info
	for _, s := range info.Symbols {
		if s.Symbol == symbol {
			result := &exchange.SymbolInfo{
				Symbol:            s.Symbol,
				BaseAsset:         s.BaseAsset,
				QuoteAsset:        s.QuoteAsset,
				PricePrecision:    s.PricePrecision,
				QuantityPrecision: s.QuantityPrecision,
			}

			// Parse filters
			for _, f := range s.Filters {
				switch f.FilterType {
				case "PRICE_FILTER":
					result.MinPrice, _ = strconv.ParseFloat(f.MinPrice, 64)
					result.MaxPrice, _ = strconv.ParseFloat(f.MaxPrice, 64)
					result.PriceTickSize, _ = strconv.ParseFloat(f.TickSize, 64)
				case "LOT_SIZE":
					result.MinQuantity, _ = strconv.ParseFloat(f.MinQty, 64)
					result.MaxQuantity, _ = strconv.ParseFloat(f.MaxQty, 64)
					result.QuantityStepSize, _ = strconv.ParseFloat(f.StepSize, 64)
				}
			}

			return result, nil
		}
	}

	return nil, fmt.Errorf("symbol not found: %s", symbol)
}

// GetServerTime implements exchange.Exchange
func (c *Client) GetServerTime(ctx context.Context) (time.Time, error) {
	var result struct {
		ServerTime int64 `json:"serverTime"`
	}
	if err := c.doRequest(ctx, http.MethodGet, MethodTime, nil, &result); err != nil {
		return time.Time{}, err
	}
	return time.Unix(0, result.ServerTime*int64(time.Millisecond)), nil
}

// GetAccountBalance implements exchange.Exchange
func (c *Client) GetAccountBalance(ctx context.Context) (map[string]*exchange.AccountBalance, error) {
	var account AccountInfo
	if err := c.doRequest(ctx, http.MethodGet, MethodAccount, nil, &account); err != nil {
		return nil, err
	}

	result := make(map[string]*exchange.AccountBalance)
	for _, asset := range account.Assets {
		balance := &exchange.AccountBalance{
			Asset:     asset.Asset,
			UpdatedAt: time.Unix(0, account.UpdateTime*int64(time.Millisecond)),
		}

		balance.Total, _ = strconv.ParseFloat(asset.WalletBalance, 64)
		balance.Available, _ = strconv.ParseFloat(asset.AvailableBalance, 64)
		balance.UnrealizedPnL, _ = strconv.ParseFloat(asset.UnrealizedProfit, 64)

		result[asset.Asset] = balance
	}

	return result, nil
}

// GetPositions implements exchange.Exchange
func (c *Client) GetPositions(ctx context.Context) ([]*exchange.Position, error) {
	var positions []Position
	if err := c.doRequest(ctx, http.MethodGet, MethodPositions, nil, &positions); err != nil {
		return nil, err
	}

	result := make([]*exchange.Position, 0, len(positions))
	for _, pos := range positions {
		amount, _ := strconv.ParseFloat(pos.PositionAmt, 64)
		if amount == 0 {
			continue
		}

		position := &exchange.Position{
			Symbol:     pos.Symbol,
			UpdatedAt:  time.Unix(0, pos.UpdateTime*int64(time.Millisecond)),
			MarginType: "CROSSED",
		}

		if pos.Isolated {
			position.MarginType = "ISOLATED"
		}

		position.Quantity, _ = strconv.ParseFloat(pos.PositionAmt, 64)
		position.EntryPrice, _ = strconv.ParseFloat(pos.EntryPrice, 64)
		position.UnrealizedPnL, _ = strconv.ParseFloat(pos.UnrealizedProfit, 64)
		position.Leverage, _ = strconv.Atoi(pos.Leverage)

		if position.Quantity > 0 {
			position.Side = "LONG"
		} else {
			position.Side = "SHORT"
			position.Quantity = -position.Quantity
		}

		result = append(result, position)
	}

	return result, nil
}

// GetPosition implements exchange.Exchange
func (c *Client) GetPosition(ctx context.Context, symbol string) (*exchange.Position, error) {
	params := url.Values{}
	params.Set("symbol", symbol)

	var positions []Position
	if err := c.doRequest(ctx, http.MethodGet, MethodPosition, params, &positions); err != nil {
		return nil, err
	}

	if len(positions) == 0 {
		return nil, fmt.Errorf("position not found: %s", symbol)
	}

	pos := positions[0]
	amount, _ := strconv.ParseFloat(pos.PositionAmt, 64)
	if amount == 0 {
		return nil, nil
	}

	position := &exchange.Position{
		Symbol:     pos.Symbol,
		UpdatedAt:  time.Unix(0, pos.UpdateTime*int64(time.Millisecond)),
		MarginType: string(exchange.MarginTypeCross),
	}

	if pos.Isolated {
		position.MarginType = string(exchange.MarginTypeIsolated)
	}

	position.Quantity = amount
	if amount < 0 {
		position.Side = string(exchange.PositionSideShort)
		position.Quantity = -amount
	} else {
		position.Side = string(exchange.PositionSideLong)
	}

	position.EntryPrice, _ = strconv.ParseFloat(pos.EntryPrice, 64)
	position.UnrealizedPnL, _ = strconv.ParseFloat(pos.UnrealizedProfit, 64)
	position.Leverage, _ = strconv.Atoi(pos.Leverage)

	return position, nil
}

// GetLeverage implements exchange.Exchange
func (c *Client) GetLeverage(ctx context.Context, symbol string) (int, error) {
	position, err := c.GetPosition(ctx, symbol)
	if err != nil {
		return 0, err
	}
	if position == nil {
		return 0, fmt.Errorf("position not found: %s", symbol)
	}
	return position.Leverage, nil
}

// SetLeverage implements exchange.Exchange
func (c *Client) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("leverage", strconv.Itoa(leverage))

	var result struct {
		Leverage int    `json:"leverage"`
		Symbol   string `json:"symbol"`
	}

	return c.doRequest(ctx, http.MethodPost, MethodLeverage, params, &result)
}

// SetMarginType implements exchange.Exchange
func (c *Client) SetMarginType(ctx context.Context, symbol string, marginType exchange.MarginType) error {
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("marginType", string(marginType))

	var result BinanceResponse
	return c.doRequest(ctx, http.MethodPost, MethodMarginType, params, &result)
}

// PlaceOrder implements exchange.Exchange
func (c *Client) PlaceOrder(ctx context.Context, req *exchange.OrderRequest) (*exchange.OrderResponse, error) {
	params := url.Values{}
	params.Set("symbol", req.Symbol)
	params.Set("side", string(req.Side))
	params.Set("type", string(req.Type))
	params.Set("quantity", strconv.FormatFloat(req.Quantity, 'f', -1, 64))

	if req.Price > 0 {
		params.Set("price", strconv.FormatFloat(req.Price, 'f', -1, 64))
	}

	if req.ClientOrderID != "" {
		params.Set("newClientOrderId", req.ClientOrderID)
	}

	if req.TimeInForce != "" {
		params.Set("timeInForce", req.TimeInForce)
	}

	if req.ReduceOnly {
		params.Set("reduceOnly", "true")
	}

	if req.PostOnly {
		params.Set("postOnly", "true")
	}

	var order Order
	if err := c.doRequest(ctx, http.MethodPost, MethodOrder, params, &order); err != nil {
		return &exchange.OrderResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &exchange.OrderResponse{
		Success: true,
		Order:   c.convertOrder(&order),
	}, nil
}

// CancelOrder implements exchange.Exchange
func (c *Client) CancelOrder(ctx context.Context, req *exchange.OrderCancelRequest) (*exchange.OrderResponse, error) {
	params := url.Values{}
	params.Set("symbol", req.Symbol)

	if req.OrderID != "" {
		params.Set("orderId", req.OrderID)
	}

	if req.ClientOrderID != "" {
		params.Set("origClientOrderId", req.ClientOrderID)
	}

	var order Order
	if err := c.doRequest(ctx, http.MethodDelete, MethodCancelOrder, params, &order); err != nil {
		return &exchange.OrderResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &exchange.OrderResponse{
		Success: true,
		Order:   c.convertOrder(&order),
	}, nil
}

// CancelAllOrders implements exchange.Exchange
func (c *Client) CancelAllOrders(ctx context.Context, symbol string) error {
	params := url.Values{}
	params.Set("symbol", symbol)

	var result BinanceResponse
	return c.doRequest(ctx, http.MethodDelete, MethodCancelAll, params, &result)
}

// GetOrder implements exchange.Exchange
func (c *Client) GetOrder(ctx context.Context, symbol, orderID string) (*exchange.Order, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("orderId", orderID)

	var order Order
	if err := c.doRequest(ctx, http.MethodGet, MethodOrder, params, &order); err != nil {
		return nil, err
	}

	return c.convertOrder(&order), nil
}

// GetOpenOrders implements exchange.Exchange
func (c *Client) GetOpenOrders(ctx context.Context, symbol string) ([]*exchange.Order, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}

	var orders []Order
	if err := c.doRequest(ctx, http.MethodGet, MethodOpenOrders, params, &orders); err != nil {
		return nil, err
	}

	result := make([]*exchange.Order, len(orders))
	for i, order := range orders {
		result[i] = c.convertOrder(&order)
	}

	return result, nil
}

// GetOrderHistory implements exchange.Exchange
func (c *Client) GetOrderHistory(ctx context.Context, symbol string, startTime, endTime time.Time) ([]*exchange.Order, error) {
	params := url.Values{}
	params.Set("symbol", symbol)

	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}

	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}

	var orders []Order
	if err := c.doRequest(ctx, http.MethodGet, MethodAllOrders, params, &orders); err != nil {
		return nil, err
	}

	result := make([]*exchange.Order, len(orders))
	for i, order := range orders {
		result[i] = c.convertOrder(&order)
	}

	return result, nil
}

// convertOrder converts a Binance order to the common format
func (c *Client) convertOrder(order *Order) *exchange.Order {
	result := &exchange.Order{
		ID:            strconv.FormatInt(order.OrderID, 10),
		ExchangeID:    strconv.FormatInt(order.OrderID, 10),
		ClientOrderID: order.ClientOrderID,
		Symbol:        order.Symbol,
		Side:          string(exchange.OrderSide(strings.ToLower(order.Side))),
		Type:          string(exchange.OrderType(strings.ToLower(order.Type))),
		Status:        string(exchange.OrderStatus(strings.ToLower(order.Status))),
		UpdatedAt:     time.Unix(0, order.UpdateTime*int64(time.Millisecond)),
	}

	result.Price, _ = strconv.ParseFloat(order.Price, 64)
	result.Quantity, _ = strconv.ParseFloat(order.OrigQty, 64)
	result.FilledQty, _ = strconv.ParseFloat(order.ExecutedQty, 64)
	result.RemainingQty = result.Quantity - result.FilledQty
	result.AvgPrice, _ = strconv.ParseFloat(order.AvgPrice, 64)

	return result
}
