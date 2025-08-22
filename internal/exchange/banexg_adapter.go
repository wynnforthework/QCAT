package exchange

import (
	"context"
	"fmt"
	"time"

	"github.com/banbox/banexg"
	"github.com/banbox/banexg/bex"
)

// BanexgAdapter adapts banexg.BanExchange to our exchange.Exchange interface
type BanexgAdapter struct {
	exchange banexg.BanExchange
	config   *ExchangeConfig
}

// NewBanexgAdapter creates a new banexg adapter
//
// Note: You may see warnings like "cache private api result is not recommend" from banexg library.
// These warnings are triggered when the library caches results from private API endpoints (like
// /fapi/v1/leverageBracket) that require authentication. The library warns about this because:
// 1. Private API responses contain sensitive account information
// 2. Cached data can become stale quickly due to trading activity
// 3. Security best practices discourage persisting sensitive data to disk
//
// These are informational warnings and can be safely ignored as they don't affect functionality.
// The caching is controlled internally by the library's CacheSecs field in API endpoint definitions.
func NewBanexgAdapter(config *ExchangeConfig) (*BanexgAdapter, error) {
	// Prepare banexg options
	options := map[string]interface{}{
		banexg.OptApiKey:    config.APIKey,
		banexg.OptApiSecret: config.APISecret,
	}

	// Set market type to linear futures (USDT-M)
	options[banexg.OptMarketType] = banexg.MarketLinear

	// Attempt to disable private API caching to avoid security warnings
	// The banexg library warns when caching private API results because they contain
	// sensitive account information that shouldn't be persisted to disk.
	// Note: The library's caching is controlled by CacheSecs field in API endpoint
	// definitions, so these options may not completely eliminate the warnings.
	if config.SuppressCacheWarnings {
		options["enableCache"] = false
		options["cachePrivateApi"] = false
		options["disableCache"] = true
	}

	// Set testnet if configured
	if config.TestNet {
		// Set environment to testnet
		options[banexg.OptEnv] = "test"
		// Disable debug to reduce log noise including cache warnings
		options[banexg.OptDebugApi] = false

		// Try to disable verbose logging that might include cache warnings
		if config.SuppressCacheWarnings {
			options["verbose"] = false
			options["debug"] = false
		}
	}

	// Create banexg exchange instance
	exg, err := bex.New("binance", options)
	if err != nil {
		return nil, fmt.Errorf("failed to create banexg exchange: %w", err)
	}

	return &BanexgAdapter{
		exchange: exg,
		config:   config,
	}, nil
}

// GetExchangeInfo implements exchange.Exchange
func (a *BanexgAdapter) GetExchangeInfo(ctx context.Context) (*ExchangeInfo, error) {
	// Load markets from banexg
	markets, err := a.exchange.LoadMarkets(false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load markets: %w", err)
	}

	// Convert to our format
	exchangeInfo := &ExchangeInfo{
		Name:       a.config.Name,
		ServerTime: time.Now(), // banexg doesn't provide server time directly
		Timezone:   "UTC",
		Symbols:    make([]SymbolInfo, 0, len(markets)),
	}

	for _, market := range markets {
		symbol := SymbolInfo{
			Symbol:     market.Symbol,
			BaseAsset:  market.Base,
			QuoteAsset: market.Quote,
			Status:     "TRADING", // Assume active markets are trading
		}
		exchangeInfo.Symbols = append(exchangeInfo.Symbols, symbol)
	}

	return exchangeInfo, nil
}

// GetSymbolInfo implements exchange.Exchange
func (a *BanexgAdapter) GetSymbolInfo(ctx context.Context, symbol string) (*SymbolInfo, error) {
	market, err := a.exchange.GetMarket(symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get market info: %w", err)
	}

	return &SymbolInfo{
		Symbol:     market.Symbol,
		BaseAsset:  market.Base,
		QuoteAsset: market.Quote,
		Status:     "TRADING",
	}, nil
}

// GetServerTime implements exchange.Exchange
func (a *BanexgAdapter) GetServerTime(ctx context.Context) (time.Time, error) {
	// banexg doesn't provide a direct server time method
	// We can use the current time or try to get it from a ticker
	return time.Now(), nil
}

// GetAccountBalance implements exchange.Exchange
func (a *BanexgAdapter) GetAccountBalance(ctx context.Context) (map[string]*AccountBalance, error) {
	balances, err := a.exchange.FetchBalance(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch balance: %w", err)
	}

	result := make(map[string]*AccountBalance)
	for asset, total := range balances.Total {
		free := balances.Free[asset]
		used := balances.Used[asset]

		result[asset] = &AccountBalance{
			Asset:     asset,
			Available: free,
			Locked:    used,
			Total:     total,
		}
	}

	return result, nil
}

// GetPositions implements exchange.Exchange
func (a *BanexgAdapter) GetPositions(ctx context.Context) ([]*Position, error) {
	positions, err := a.exchange.FetchPositions(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch positions: %w", err)
	}

	result := make([]*Position, 0, len(positions))
	for _, pos := range positions {
		position := &Position{
			Symbol:           pos.Symbol,
			Side:             pos.Side,
			Size:             pos.Contracts,
			EntryPrice:       pos.EntryPrice,
			MarkPrice:        pos.MarkPrice,
			UnrealizedPnL:    pos.UnrealizedPnl,
			Leverage:         pos.Leverage,
			MarginType:       pos.MarginMode,
			LiquidationPrice: pos.LiquidationPrice,
			UpdatedAt:        time.Unix(0, pos.TimeStamp*int64(time.Millisecond)),
		}
		result = append(result, position)
	}

	return result, nil
}

// GetPosition implements exchange.Exchange
func (a *BanexgAdapter) GetPosition(ctx context.Context, symbol string) (*Position, error) {
	positions, err := a.GetPositions(ctx)
	if err != nil {
		return nil, err
	}

	for _, pos := range positions {
		if pos.Symbol == symbol {
			return pos, nil
		}
	}

	return nil, fmt.Errorf("position not found for symbol: %s", symbol)
}

// GetLeverage implements exchange.Exchange
func (a *BanexgAdapter) GetLeverage(ctx context.Context, symbol string) (int, error) {
	// Try to get leverage from position info first
	positions, err := a.GetPositions(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get positions: %w", err)
	}

	// Look for the symbol in positions
	for _, pos := range positions {
		if pos.Symbol == symbol && pos.Size > 0 {
			return int(pos.Leverage), nil
		}
	}

	// If no position found, return default leverage (20x for Binance futures)
	return 20, nil
}

// SetLeverage implements exchange.Exchange
func (a *BanexgAdapter) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	_, err := a.exchange.SetLeverage(float64(leverage), symbol, nil)
	if err != nil {
		return fmt.Errorf("failed to set leverage: %w", err)
	}
	return nil
}

// SetMarginType implements exchange.Exchange
func (a *BanexgAdapter) SetMarginType(ctx context.Context, symbol string, marginType MarginType) error {
	// banexg doesn't have a direct set margin type method
	// This might need to be implemented based on the specific exchange API
	return fmt.Errorf("set margin type not implemented in banexg adapter")
}

// PlaceOrder implements exchange.Exchange
func (a *BanexgAdapter) PlaceOrder(ctx context.Context, req *OrderRequest) (*OrderResponse, error) {
	order, err := a.exchange.CreateOrder(
		req.Symbol,
		req.Type,
		req.Side,
		req.Quantity,
		req.Price,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to place order: %w", err)
	}

	return &OrderResponse{
		OrderID:       order.ID,
		ClientOrderID: order.ClientOrderID,
		Symbol:        order.Symbol,
		Side:          order.Side,
		Type:          order.Type,
		Quantity:      order.Amount,
		Price:         order.Price,
		Status:        order.Status,
		TimeInForce:   order.TimeInForce,
		Time:          time.Unix(0, order.Timestamp*int64(time.Millisecond)),
	}, nil
}

// CancelOrder implements exchange.Exchange
func (a *BanexgAdapter) CancelOrder(ctx context.Context, req *OrderCancelRequest) (*OrderResponse, error) {
	order, err := a.exchange.CancelOrder(req.OrderID, req.Symbol, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel order: %w", err)
	}

	return &OrderResponse{
		OrderID:       order.ID,
		ClientOrderID: order.ClientOrderID,
		Symbol:        order.Symbol,
		Side:          order.Side,
		Type:          order.Type,
		Quantity:      order.Amount,
		Price:         order.Price,
		Status:        order.Status,
		TimeInForce:   order.TimeInForce,
		Time:          time.Unix(0, order.Timestamp*int64(time.Millisecond)),
	}, nil
}

// CancelAllOrders implements exchange.Exchange
func (a *BanexgAdapter) CancelAllOrders(ctx context.Context, symbol string) error {
	// banexg doesn't have a direct cancel all orders method
	// We need to get open orders and cancel them one by one
	orders, err := a.exchange.FetchOpenOrders(symbol, 0, 0, nil)
	if err != nil {
		return fmt.Errorf("failed to fetch open orders: %w", err)
	}

	for _, order := range orders {
		_, err := a.exchange.CancelOrder(order.ID, symbol, nil)
		if err != nil {
			return fmt.Errorf("failed to cancel order %s: %w", order.ID, err)
		}
	}

	return nil
}

// GetOrder implements exchange.Exchange
func (a *BanexgAdapter) GetOrder(ctx context.Context, symbol, orderID string) (*Order, error) {
	order, err := a.exchange.FetchOrder(symbol, orderID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order: %w", err)
	}

	return &Order{
		OrderID:       order.ID,
		ClientOrderID: order.ClientOrderID,
		Symbol:        order.Symbol,
		Side:          order.Side,
		Type:          order.Type,
		Quantity:      order.Amount,
		Price:         order.Price,
		ExecutedQty:   order.Filled,
		Status:        order.Status,
		TimeInForce:   order.TimeInForce,
		Time:          time.Unix(0, order.Timestamp*int64(time.Millisecond)),
		UpdatedTime:   time.Unix(0, order.LastUpdateTimestamp*int64(time.Millisecond)),
	}, nil
}

// GetOpenOrders implements exchange.Exchange
func (a *BanexgAdapter) GetOpenOrders(ctx context.Context, symbol string) ([]*Order, error) {
	orders, err := a.exchange.FetchOpenOrders(symbol, 0, 0, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch open orders: %w", err)
	}

	result := make([]*Order, 0, len(orders))
	for _, order := range orders {
		result = append(result, &Order{
			OrderID:       order.ID,
			ClientOrderID: order.ClientOrderID,
			Symbol:        order.Symbol,
			Side:          order.Side,
			Type:          order.Type,
			Quantity:      order.Amount,
			Price:         order.Price,
			ExecutedQty:   order.Filled,
			Status:        order.Status,
			TimeInForce:   order.TimeInForce,
			Time:          time.Unix(0, order.Timestamp*int64(time.Millisecond)),
			UpdatedTime:   time.Unix(0, order.LastUpdateTimestamp*int64(time.Millisecond)),
		})
	}

	return result, nil
}

// GetOrderHistory implements exchange.Exchange
func (a *BanexgAdapter) GetOrderHistory(ctx context.Context, symbol string, startTime, endTime time.Time) ([]*Order, error) {
	since := startTime.UnixMilli()
	orders, err := a.exchange.FetchOrders(symbol, since, 0, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order history: %w", err)
	}

	result := make([]*Order, 0, len(orders))
	for _, order := range orders {
		orderTime := time.Unix(0, order.Timestamp*int64(time.Millisecond))
		if orderTime.After(endTime) {
			continue
		}

		result = append(result, &Order{
			OrderID:       order.ID,
			ClientOrderID: order.ClientOrderID,
			Symbol:        order.Symbol,
			Side:          order.Side,
			Type:          order.Type,
			Quantity:      order.Amount,
			Price:         order.Price,
			ExecutedQty:   order.Filled,
			Status:        order.Status,
			TimeInForce:   order.TimeInForce,
			Time:          time.Unix(0, order.Timestamp*int64(time.Millisecond)),
			UpdatedTime:   time.Unix(0, order.LastUpdateTimestamp*int64(time.Millisecond)),
		})
	}

	return result, nil
}

// GetRiskLimits implements exchange.Exchange
func (a *BanexgAdapter) GetRiskLimits(ctx context.Context, symbol string) (*RiskLimits, error) {
	// banexg doesn't provide risk limits directly
	// We'll return reasonable defaults for Binance futures
	return &RiskLimits{
		Symbol:           symbol,
		MaxLeverage:      125,     // Binance futures max leverage
		MaxPositionValue: 1000000, // 1M USD max position
		MaxOrderValue:    100000,  // 100K USD max order
		MinOrderValue:    5,       // 5 USD min order
		MaxOrderQty:      10000,   // Max order quantity
		MinOrderQty:      0.001,   // Min order quantity
	}, nil
}

// GetMarginInfo implements exchange.Exchange
func (a *BanexgAdapter) GetMarginInfo(ctx context.Context) (*MarginInfo, error) {
	// Get account balance to calculate margin info
	balances, err := a.GetAccountBalance(ctx)
	if err != nil {
		return nil, err
	}

	// Get positions to calculate used margin
	positions, err := a.GetPositions(ctx)
	if err != nil {
		return nil, err
	}

	var totalWalletBalance float64
	var totalUnrealizedPnL float64
	var totalUsedMargin float64

	// Calculate total wallet balance (USDT for futures)
	if usdtBalance, exists := balances["USDT"]; exists {
		totalWalletBalance = usdtBalance.Total
	}

	// Calculate total unrealized PnL and used margin
	for _, pos := range positions {
		totalUnrealizedPnL += pos.UnrealizedPnL
		// Used margin is approximately position size / leverage
		if pos.Leverage > 0 {
			totalUsedMargin += (pos.Size * pos.MarkPrice) / float64(pos.Leverage)
		}
	}

	marginRatio := 0.0
	if totalWalletBalance > 0 {
		marginRatio = totalUsedMargin / totalWalletBalance
	}

	return &MarginInfo{
		TotalAssetValue:   totalWalletBalance,
		TotalDebtValue:    totalUsedMargin,
		MarginRatio:       marginRatio,
		MaintenanceMargin: 0.1, // 10% maintenance margin
		MarginCallRatio:   0.8, // 80% margin call ratio
		LiquidationRatio:  0.9, // 90% liquidation ratio
		UpdatedAt:         time.Now(),
	}, nil
}

// SetRiskLimits implements exchange.Exchange
func (a *BanexgAdapter) SetRiskLimits(ctx context.Context, symbol string, limits *RiskLimits) error {
	// banexg doesn't provide risk limit setting
	return fmt.Errorf("set risk limits not supported by banexg adapter")
}

// GetPositionByID implements exchange.Exchange
func (a *BanexgAdapter) GetPositionByID(ctx context.Context, positionID string) (*Position, error) {
	// banexg doesn't support position by ID lookup
	// We'll need to get all positions and find by ID
	positions, err := a.GetPositions(ctx)
	if err != nil {
		return nil, err
	}

	for _, pos := range positions {
		// Use symbol as position ID since banexg doesn't provide position IDs
		if pos.Symbol == positionID {
			return pos, nil
		}
	}

	return nil, fmt.Errorf("position not found with ID: %s", positionID)
}

// GetSymbolPrice implements exchange.Exchange
func (a *BanexgAdapter) GetSymbolPrice(ctx context.Context, symbol string) (float64, error) {
	ticker, err := a.exchange.FetchTicker(symbol, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch ticker: %w", err)
	}

	return ticker.Last, nil
}

// Close implements cleanup
func (a *BanexgAdapter) Close() error {
	if a.exchange != nil {
		return a.exchange.Close()
	}
	return nil
}
