package live

import (
	"context"
	"fmt"
	"sync"

	exch "qcat/internal/exchange"
	"qcat/internal/exchange/order"
	"qcat/internal/exchange/position"
	"qcat/internal/exchange/risk"
	"qcat/internal/market"
	"qcat/internal/strategy"
	"qcat/internal/strategy/sandbox"
)

// Runner manages real-time strategy execution
type Runner struct {
	sandbox    *sandbox.Sandbox
	market     *market.Ingestor
	order      *order.Manager
	position   *position.Manager
	risk       *risk.Manager
	marketSubs []interface{} // 市场数据订阅列表
	mu sync.RWMutex
}

// NewRunner creates a new real-time strategy runner
func NewRunner(sandbox *sandbox.Sandbox, market *market.Ingestor, order *order.Manager, position *position.Manager, risk *risk.Manager) *Runner {
	return &Runner{
		sandbox:  sandbox,
		market:   market,
		order:    order,
		position: position,
		risk:     risk,
	}
}

// Start starts real-time strategy execution
func (r *Runner) Start(ctx context.Context) error {
	// Validate sandbox
	if err := r.sandbox.Validate(); err != nil {
		return fmt.Errorf("invalid sandbox: %w", err)
	}

	// Subscribe to market data
	if err := r.subscribeMarketData(ctx); err != nil {
		return fmt.Errorf("failed to subscribe to market data: %w", err)
	}

	// Subscribe to order updates
	config := r.sandbox.GetConfig()
	symbol := r.getSymbolFromConfig(config)
	orderCh := r.order.Subscribe(symbol)
	go r.handleOrders(ctx, orderCh)

	// Subscribe to position updates
	positionCh := r.position.Subscribe(symbol)
	go r.handlePositions(ctx, positionCh)

	// Start sandbox
	if err := r.sandbox.Start(ctx); err != nil {
		return fmt.Errorf("failed to start sandbox: %w", err)
	}

	return nil
}

// Stop stops real-time strategy execution
func (r *Runner) Stop(ctx context.Context) error {
	// Unsubscribe from market data
	r.unsubscribeMarketData()

	// Unsubscribe from order updates
	config := r.sandbox.GetConfig()
	symbol := r.getSymbolFromConfig(config)
	r.order.Unsubscribe(symbol, nil)

	// Unsubscribe from position updates
	r.position.Unsubscribe(symbol, nil)

	// Stop sandbox
	if err := r.sandbox.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop sandbox: %w", err)
	}

	return nil
}

// subscribeMarketData subscribes to required market data
func (r *Runner) subscribeMarketData(ctx context.Context) error {
	config := r.sandbox.GetConfig()
	symbol := r.getSymbolFromConfig(config)

	// Subscribe to order book updates
	bookCh, err := r.market.SubscribeOrderBook(ctx, symbol)
	if err != nil {
		return fmt.Errorf("failed to subscribe to order book: %w", err)
	}
	go r.handleOrderBook(ctx, bookCh)
	r.marketSubs = append(r.marketSubs, bookCh)

	// Subscribe to trades
	tradeCh, err := r.market.SubscribeTrades(ctx, symbol)
	if err != nil {
		return fmt.Errorf("failed to subscribe to trades: %w", err)
	}
	go r.handleTrades(ctx, tradeCh)
	r.marketSubs = append(r.marketSubs, tradeCh)

	// Subscribe to klines
	klineCh, err := r.market.SubscribeKlines(ctx, symbol, "1m")
	if err != nil {
		return fmt.Errorf("failed to subscribe to klines: %w", err)
	}
	go r.handleKlines(ctx, klineCh)
	r.marketSubs = append(r.marketSubs, klineCh)

	// Subscribe to funding rates
	fundingCh, err := r.market.SubscribeFundingRates(ctx, symbol)
	if err != nil {
		return fmt.Errorf("failed to subscribe to funding rates: %w", err)
	}
	go r.handleFundingRates(ctx, fundingCh)
	r.marketSubs = append(r.marketSubs, fundingCh)

	return nil
}

// getSymbolFromConfig extracts symbol from configuration
func (r *Runner) getSymbolFromConfig(config map[string]interface{}) string {
	if s, ok := config["symbol"].(string); ok && s != "" {
		return s
	}
	return "BTCUSDT" // 默认交易对
}

// unsubscribeMarketData unsubscribes from all market data
func (r *Runner) unsubscribeMarketData() {
	// 清空订阅列表，market.Ingestor 返回的是 channel，无需显式取消订阅
	r.marketSubs = nil
}

// handleOrderBook handles order book updates
func (r *Runner) handleOrderBook(ctx context.Context, ch <-chan *market.OrderBook) {
	for {
		select {
		case <-ctx.Done():
			return
		case book := <-ch:
			r.sandbox.OnMarketData(book)
		}
	}
}

// handleTrades handles trade updates
func (r *Runner) handleTrades(ctx context.Context, ch <-chan *market.Trade) {
	for {
		select {
		case <-ctx.Done():
			return
		case trade := <-ch:
			r.sandbox.OnMarketData(trade)
		}
	}
}

// handleKlines handles kline updates
func (r *Runner) handleKlines(ctx context.Context, ch <-chan *market.Kline) {
	for {
		select {
		case <-ctx.Done():
			return
		case kline := <-ch:
			r.sandbox.OnMarketData(kline)
		}
	}
}

// handleFundingRates handles funding rate updates
func (r *Runner) handleFundingRates(ctx context.Context, ch <-chan *market.FundingRate) {
	for {
		select {
		case <-ctx.Done():
			return
		case rate := <-ch:
			r.sandbox.OnMarketData(rate)
		}
	}
}

// handleOrders handles order updates
func (r *Runner) handleOrders(ctx context.Context, ch <-chan *exch.Order) {
	for {
		select {
		case <-ctx.Done():
			return
		case order := <-ch:
			r.sandbox.OnOrder(order)
		}
	}
}

// handlePositions handles position updates
func (r *Runner) handlePositions(ctx context.Context, ch <-chan *exch.Position) {
	for {
		select {
		case <-ctx.Done():
			return
		case position := <-ch:
			r.sandbox.OnPosition(position)
		}
	}
}

// OnSignal handles strategy signals
func (r *Runner) OnSignal(signal *strategy.Signal) error {
	// Check risk limits
	if err := r.checkRiskLimits(signal); err != nil {
		return fmt.Errorf("risk check failed: %w", err)
	}

	// Create order request
	req := &exch.OrderRequest{
		Symbol:        signal.Symbol,
		Side:          string(signal.Side), // 显式转换为 string
		Type:          string(signal.Type), // 显式转换为 string
		Price:         signal.Price,
		Quantity:      signal.Quantity,
		ClientOrderID: signal.ID,
	}

	// Place order
	resp, err := r.order.PlaceOrder(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to place order: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("order rejected: %v", resp.Error)
	}

	return nil
}

// checkRiskLimits checks if a signal would violate risk limits
func (r *Runner) checkRiskLimits(signal *strategy.Signal) error {
	// Create order request for risk check
	req := &exch.OrderRequest{
		Symbol:   signal.Symbol,
		Side:     string(signal.Side), // 显式转换为 string
		Type:     string(signal.Type), // 显式转换为 string
		Price:    signal.Price,
		Quantity: signal.Quantity,
	}

	// Check risk limits
	if err := r.risk.CheckRiskLimits(context.Background(), req); err != nil {
		return fmt.Errorf("risk limit violation: %w", err)
	}

	return nil
}

// GetState returns the current strategy state
func (r *Runner) GetState() string {
	return r.sandbox.GetState()
}

// GetResult returns the strategy execution result
func (r *Runner) GetResult() *strategy.Result {
	return r.sandbox.GetResult()
}
