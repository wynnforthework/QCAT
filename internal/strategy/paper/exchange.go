package paper

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/exchange"
	"qcat/internal/market"
)

// Exchange implements exchange.Exchange for paper trading
type Exchange struct {
	account   *Account
	market    *market.Ingestor
	orderBook map[string]*OrderBook
	mu        sync.RWMutex
}

// NewExchange creates a new paper trading exchange
func NewExchange(market *market.Ingestor, initialBalance map[string]float64) *Exchange {
	return &Exchange{
		account: &Account{
			Balances:  initialBalance,
			Positions: make(map[string]*Position),
			Orders:    make(map[string]*Order),
		},
		market:    market,
		orderBook: make(map[string]*OrderBook),
	}
}

// GetExchangeInfo implements exchange.Exchange
func (e *Exchange) GetExchangeInfo(ctx context.Context) (*exchange.ExchangeInfo, error) {
	return &exchange.ExchangeInfo{
		Name:       "paper",
		ServerTime: time.Now(),
		RateLimits: []exchange.RateLimit{},
		Symbols:    []exchange.SymbolInfo{},
	}, nil
}

// GetSymbolInfo implements exchange.Exchange
func (e *Exchange) GetSymbolInfo(ctx context.Context, symbol string) (*exchange.SymbolInfo, error) {
	return &exchange.SymbolInfo{
		Symbol:            symbol,
		BaseAsset:         "",
		QuoteAsset:        "",
		PricePrecision:    8,
		QuantityPrecision: 8,
	}, nil
}

// GetServerTime implements exchange.Exchange
func (e *Exchange) GetServerTime(ctx context.Context) (time.Time, error) {
	return time.Now(), nil
}

// GetAccountBalance implements exchange.Exchange
func (e *Exchange) GetAccountBalance(ctx context.Context) (map[string]*exchange.AccountBalance, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	balances := make(map[string]*exchange.AccountBalance)
	for asset, amount := range e.account.Balances {
		balances[asset] = &exchange.AccountBalance{
			Asset:     asset,
			Total:     amount,
			Available: amount,
			UpdatedAt: time.Now(),
		}
	}
	return balances, nil
}

// GetPositions implements exchange.Exchange
func (e *Exchange) GetPositions(ctx context.Context) ([]*exchange.Position, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	positions := make([]*exchange.Position, 0, len(e.account.Positions))
	for _, pos := range e.account.Positions {
		positions = append(positions, &exchange.Position{
			Symbol:        pos.Symbol,
			Side:          string(pos.Side),
			Quantity:      pos.Quantity,
			EntryPrice:    pos.EntryPrice,
			Leverage:      pos.Leverage,
			MarginType:    string(pos.MarginType),
			UnrealizedPnL: pos.UnrealizedPnL,
			UpdatedAt:     pos.UpdatedAt,
		})
	}
	return positions, nil
}

// GetPosition implements exchange.Exchange
func (e *Exchange) GetPosition(ctx context.Context, symbol string) (*exchange.Position, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	pos, exists := e.account.Positions[symbol]
	if !exists {
		return nil, &ErrPositionNotFound{Symbol: symbol}
	}

	return &exchange.Position{
		Symbol:        pos.Symbol,
		Side:          string(pos.Side),
		Quantity:      pos.Quantity,
		EntryPrice:    pos.EntryPrice,
		Leverage:      pos.Leverage,
		MarginType:    string(pos.MarginType),
		UnrealizedPnL: pos.UnrealizedPnL,
		UpdatedAt:     pos.UpdatedAt,
	}, nil
}

// GetLeverage implements exchange.Exchange
func (e *Exchange) GetLeverage(ctx context.Context, symbol string) (int, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	pos, exists := e.account.Positions[symbol]
	if !exists {
		return 0, &ErrPositionNotFound{Symbol: symbol}
	}
	return pos.Leverage, nil
}

// SetLeverage implements exchange.Exchange
func (e *Exchange) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	pos, exists := e.account.Positions[symbol]
	if !exists {
		pos = &Position{
			Symbol:   symbol,
			Leverage: leverage,
		}
		e.account.Positions[symbol] = pos
	} else {
		pos.Leverage = leverage
	}
	return nil
}

// SetMarginType implements exchange.Exchange
func (e *Exchange) SetMarginType(ctx context.Context, symbol string, marginType exchange.MarginType) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	pos, exists := e.account.Positions[symbol]
	if !exists {
		pos = &Position{
			Symbol:     symbol,
			MarginType: marginType,
		}
		e.account.Positions[symbol] = pos
	} else {
		pos.MarginType = marginType
	}
	return nil
}

// PlaceOrder implements exchange.Exchange
func (e *Exchange) PlaceOrder(ctx context.Context, req *exchange.OrderRequest) (*exchange.OrderResponse, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Validate order
	if err := e.validateOrder(req); err != nil {
		return &exchange.OrderResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Create order
	order := &Order{
		ID:            fmt.Sprintf("paper-%d", time.Now().UnixNano()),
		ClientOrderID: req.ClientOrderID,
		Symbol:        req.Symbol,
		Side:          exchange.OrderSide(req.Side),
		Type:          exchange.OrderType(req.Type),
		Price:         req.Price,
		StopPrice:     req.StopPrice,
		Quantity:      req.Quantity,
		RemainingQty:  req.Quantity,
		Status:        exchange.OrderStatusNew,
		ReduceOnly:    req.ReduceOnly,
		PostOnly:      req.PostOnly,
		TimeInForce:   req.TimeInForce,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Store order
	e.account.Orders[order.ID] = order

	// Try to match order
	if err := e.matchOrder(order); err != nil {
		return &exchange.OrderResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &exchange.OrderResponse{
		Success: true,
		Order: &exchange.Order{
			ID:            order.ID,
			ClientOrderID: order.ClientOrderID,
			Symbol:        order.Symbol,
			Side:          string(order.Side),
			Type:          string(order.Type),
			Price:         order.Price,
			Quantity:      order.Quantity,
			FilledQty:     order.FilledQty,
			RemainingQty:  order.RemainingQty,
			Status:        string(order.Status),
			UpdatedAt:     order.UpdatedAt,
		},
	}, nil
}

// CancelOrder implements exchange.Exchange
func (e *Exchange) CancelOrder(ctx context.Context, req *exchange.OrderCancelRequest) (*exchange.OrderResponse, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	var order *Order
	if req.OrderID != "" {
		order = e.account.Orders[req.OrderID]
	} else if req.ClientOrderID != "" {
		for _, o := range e.account.Orders {
			if o.ClientOrderID == req.ClientOrderID {
				order = o
				break
			}
		}
	}

	if order == nil {
		return &exchange.OrderResponse{
			Success: false,
			Error:   "order not found",
		}, nil
	}

	if order.Status != exchange.OrderStatusNew && order.Status != exchange.OrderStatusPartiallyFilled {
		return &exchange.OrderResponse{
			Success: false,
			Error:   "order cannot be cancelled",
		}, nil
	}

	order.Status = exchange.OrderStatusCancelled
	order.UpdatedAt = time.Now()

	return &exchange.OrderResponse{
		Success: true,
		Order: &exchange.Order{
			ID:            order.ID,
			ClientOrderID: order.ClientOrderID,
			Symbol:        order.Symbol,
			Side:          string(order.Side),
			Type:          string(order.Type),
			Price:         order.Price,
			Quantity:      order.Quantity,
			FilledQty:     order.FilledQty,
			RemainingQty:  order.RemainingQty,
			Status:        string(order.Status),
			UpdatedAt:     order.UpdatedAt,
		},
	}, nil
}

// CancelAllOrders implements exchange.Exchange
func (e *Exchange) CancelAllOrders(ctx context.Context, symbol string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, order := range e.account.Orders {
		if order.Symbol == symbol && (order.Status == exchange.OrderStatusNew || order.Status == exchange.OrderStatusPartiallyFilled) {
			order.Status = exchange.OrderStatusCancelled
			order.UpdatedAt = time.Now()
		}
	}
	return nil
}

// GetOrder implements exchange.Exchange
func (e *Exchange) GetOrder(ctx context.Context, symbol, orderID string) (*exchange.Order, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	order, exists := e.account.Orders[orderID]
	if !exists {
		return nil, &ErrOrderNotFound{ID: orderID}
	}

	return &exchange.Order{
		ID:            order.ID,
		ClientOrderID: order.ClientOrderID,
		Symbol:        order.Symbol,
		Side:          string(order.Side),
		Type:          string(order.Type),
		Price:         order.Price,
		Quantity:      order.Quantity,
		FilledQty:     order.FilledQty,
		RemainingQty:  order.RemainingQty,
		Status:        string(order.Status),
		UpdatedAt:     order.UpdatedAt,
	}, nil
}

// GetOpenOrders implements exchange.Exchange
func (e *Exchange) GetOpenOrders(ctx context.Context, symbol string) ([]*exchange.Order, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	orders := make([]*exchange.Order, 0)
	for _, order := range e.account.Orders {
		if (symbol == "" || order.Symbol == symbol) && (order.Status == exchange.OrderStatusNew || order.Status == exchange.OrderStatusPartiallyFilled) {
			orders = append(orders, &exchange.Order{
				ID:            order.ID,
				ClientOrderID: order.ClientOrderID,
				Symbol:        order.Symbol,
				Side:          string(order.Side),
				Type:          string(order.Type),
				Price:         order.Price,
				Quantity:      order.Quantity,
				FilledQty:     order.FilledQty,
				RemainingQty:  order.RemainingQty,
				Status:        string(order.Status),
				UpdatedAt:     order.UpdatedAt,
			})
		}
	}
	return orders, nil
}

// GetRiskLimits implements exchange.Exchange
func (e *Exchange) GetRiskLimits(ctx context.Context, symbol string) (*exchange.RiskLimits, error) {
	return &exchange.RiskLimits{
		Symbol:           symbol,
		MaxLeverage:      100,
		MaxPositionValue: 1000000,
		MaxOrderValue:    100000,
		MinOrderValue:    10,
		MaxOrderQty:      1000,
		MinOrderQty:      0.001,
	}, nil
}

// GetMarginInfo implements exchange.Exchange
func (e *Exchange) GetMarginInfo(ctx context.Context) (*exchange.MarginInfo, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	totalBalance := 0.0
	for _, balance := range e.account.Balances {
		totalBalance += balance
	}

	return &exchange.MarginInfo{
		TotalAssetValue:   totalBalance,
		TotalDebtValue:    0.0,
		MarginRatio:       0.0,
		MaintenanceMargin: 1.1,
		MarginCallRatio:   1.5,
		LiquidationRatio:  1.0,
		UpdatedAt:         time.Now(),
	}, nil
}

// SetRiskLimits implements exchange.Exchange
func (e *Exchange) SetRiskLimits(ctx context.Context, symbol string, limits *exchange.RiskLimits) error {
	// Paper trading doesn't enforce risk limits, just log the action
	log.Printf("Paper trading: Set risk limits for %s: %+v", symbol, limits)
	return nil
}

// GetPositionByID implements exchange.Exchange
func (e *Exchange) GetPositionByID(ctx context.Context, positionID string) (*exchange.Position, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// In paper trading, we use symbol as position ID
	pos, exists := e.account.Positions[positionID]
	if !exists {
		return nil, &ErrPositionNotFound{Symbol: positionID}
	}

	return &exchange.Position{
		Symbol:        pos.Symbol,
		Side:          string(pos.Side),
		Quantity:      pos.Quantity,
		EntryPrice:    pos.EntryPrice,
		Leverage:      pos.Leverage,
		MarginType:    string(pos.MarginType),
		UnrealizedPnL: pos.UnrealizedPnL,
		UpdatedAt:     pos.UpdatedAt,
	}, nil
}

// GetSymbolPrice implements exchange.Exchange
func (e *Exchange) GetSymbolPrice(ctx context.Context, symbol string) (float64, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	book := e.orderBook[symbol]
	if book == nil {
		return 0, fmt.Errorf("no market data available for symbol: %s", symbol)
	}

	// Return mid price if both bid and ask are available
	if len(book.Bids) > 0 && len(book.Asks) > 0 {
		return (book.Bids[0].Price + book.Asks[0].Price) / 2, nil
	}

	// Return bid price if only bid is available
	if len(book.Bids) > 0 {
		return book.Bids[0].Price, nil
	}

	// Return ask price if only ask is available
	if len(book.Asks) > 0 {
		return book.Asks[0].Price, nil
	}

	return 0, fmt.Errorf("no price data available for symbol: %s", symbol)
}

// GetOrderHistory implements exchange.Exchange
func (e *Exchange) GetOrderHistory(ctx context.Context, symbol string, startTime, endTime time.Time) ([]*exchange.Order, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	orders := make([]*exchange.Order, 0)
	for _, order := range e.account.Orders {
		if (symbol == "" || order.Symbol == symbol) &&
			(startTime.IsZero() || order.CreatedAt.After(startTime)) &&
			(endTime.IsZero() || order.CreatedAt.Before(endTime)) {
			orders = append(orders, &exchange.Order{
				ID:            order.ID,
				ClientOrderID: order.ClientOrderID,
				Symbol:        order.Symbol,
				Side:          string(order.Side),
				Type:          string(order.Type),
				Price:         order.Price,
				Quantity:      order.Quantity,
				FilledQty:     order.FilledQty,
				RemainingQty:  order.RemainingQty,
				Status:        string(order.Status),
				UpdatedAt:     order.UpdatedAt,
			})
		}
	}
	return orders, nil
}

// validateOrder validates an order request
func (e *Exchange) validateOrder(req *exchange.OrderRequest) error {
	if req.Symbol == "" {
		return &ErrInvalidOrder{Message: "symbol is required"}
	}
	if req.Quantity <= 0 {
		return &ErrInvalidOrder{Message: "quantity must be positive"}
	}
	if exchange.OrderType(req.Type) == exchange.OrderTypeLimit && req.Price <= 0 {
		return &ErrInvalidOrder{Message: "price must be positive for limit orders"}
	}
	return nil
}

// matchOrder attempts to match and execute an order
func (e *Exchange) matchOrder(order *Order) error {
	// For market orders, use the current market price
	if order.Type == exchange.OrderTypeMarket {
		book := e.orderBook[order.Symbol]
		if book == nil {
			return &ErrInvalidOrder{Message: "no market data available"}
		}

		var price float64
		if order.Side == exchange.OrderSideBuy {
			if len(book.Asks) == 0 {
				return &ErrInvalidOrder{Message: "no ask price available"}
			}
			price = book.Asks[0].Price
		} else {
			if len(book.Bids) == 0 {
				return &ErrInvalidOrder{Message: "no bid price available"}
			}
			price = book.Bids[0].Price
		}

		order.Price = price
	}

	// Execute the order
	if err := e.executeOrder(order); err != nil {
		return err
	}

	return nil
}

// executeOrder executes an order
func (e *Exchange) executeOrder(order *Order) error {
	// Calculate required margin
	margin := order.Quantity * order.Price
	if pos, exists := e.account.Positions[order.Symbol]; exists && pos.Leverage > 0 {
		margin /= float64(pos.Leverage)
	}

	// Check balance
	if balance, exists := e.account.Balances["USDT"]; !exists || balance < margin {
		return &ErrInsufficientBalance{
			Asset:    "USDT",
			Required: margin,
			Current:  balance,
		}
	}

	// Update position
	pos, exists := e.account.Positions[order.Symbol]
	if !exists {
		pos = &Position{
			Symbol:     order.Symbol,
			Side:       exchange.PositionSideLong,
			UpdatedAt:  time.Now(),
			MarginType: exchange.MarginTypeCross,
		}
		e.account.Positions[order.Symbol] = pos
	}

	// Update position
	if order.Side == exchange.OrderSideBuy {
		pos.Quantity += order.Quantity
		pos.EntryPrice = (pos.EntryPrice*pos.Quantity + order.Price*order.Quantity) / (pos.Quantity + order.Quantity)
	} else {
		pos.Quantity -= order.Quantity
		if pos.Quantity < 0 {
			pos.Side = exchange.PositionSideShort
			pos.Quantity = -pos.Quantity
			pos.EntryPrice = order.Price
		}
	}

	// Update order status
	order.Status = exchange.OrderStatusFilled
	order.FilledQty = order.Quantity
	order.RemainingQty = 0
	order.UpdatedAt = time.Now()

	return nil
}

// UpdateOrderBook updates the order book
func (e *Exchange) UpdateOrderBook(symbol string, bids, asks []Level) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.orderBook[symbol] = &OrderBook{
		Symbol:    symbol,
		Bids:      bids,
		Asks:      asks,
		UpdatedAt: time.Now(),
	}
}
