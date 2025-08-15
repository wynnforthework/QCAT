package signal

import (
	"context"
	"fmt"
	"sync"
	"time"

	"qcat/internal/exchange"
)

// DefaultProcessor implements the default signal processing logic
type DefaultProcessor struct {
	exchange  exchange.Exchange
	validator Validator
	signals   map[string]*Signal
	mu        sync.RWMutex
}

// NewDefaultProcessor creates a new default processor
func NewDefaultProcessor(exchange exchange.Exchange, validator Validator) *DefaultProcessor {
	return &DefaultProcessor{
		exchange:  exchange,
		validator: validator,
		signals:   make(map[string]*Signal),
	}
}

// Process processes a signal
func (p *DefaultProcessor) Process(signal *Signal) error {
	// Store signal
	p.mu.Lock()
	p.signals[signal.ID] = signal
	p.mu.Unlock()

	// Validate signal
	if err := p.validator.Validate(signal); err != nil {
		signal.Status = StatusRejected
		signal.Reason = err.Error()
		signal.UpdatedAt = time.Now()
		return err
	}

	// Check expiration
	if !signal.ExpiresAt.IsZero() && signal.ExpiresAt.Before(time.Now()) {
		signal.Status = StatusExpired
		signal.Reason = "signal expired"
		signal.UpdatedAt = time.Now()
		return &ErrSignalProcessing{Message: "signal expired"}
	}

	// Create order request
	req := &exchange.OrderRequest{
		Symbol:        signal.Symbol,
		Side:          string(signal.Side),
		Type:          string(signal.OrderType),
		Price:         signal.Price,
		StopPrice:     signal.StopPrice,
		Quantity:      signal.Quantity,
		ClientOrderID: signal.ID,
		TimeInForce:   signal.TimeInForce,
		ReduceOnly:    signal.ReduceOnly,
		PostOnly:      signal.PostOnly,
	}

	// Set leverage and margin type
	if err := p.exchange.SetLeverage(context.Background(), signal.Symbol, signal.Leverage); err != nil {
		signal.Status = StatusRejected
		signal.Reason = fmt.Sprintf("failed to set leverage: %v", err)
		signal.UpdatedAt = time.Now()
		return &ErrSignalProcessing{Message: "failed to set leverage", Err: err}
	}

	if err := p.exchange.SetMarginType(context.Background(), signal.Symbol, signal.MarginType); err != nil {
		signal.Status = StatusRejected
		signal.Reason = fmt.Sprintf("failed to set margin type: %v", err)
		signal.UpdatedAt = time.Now()
		return &ErrSignalProcessing{Message: "failed to set margin type", Err: err}
	}

	// Place order
	resp, err := p.exchange.PlaceOrder(context.Background(), req)
	if err != nil {
		signal.Status = StatusRejected
		signal.Reason = fmt.Sprintf("failed to place order: %v", err)
		signal.UpdatedAt = time.Now()
		return &ErrSignalProcessing{Message: "failed to place order", Err: err}
	}

	if !resp.Success {
		signal.Status = StatusRejected
		signal.Reason = fmt.Sprintf("order rejected: %v", resp.Error)
		signal.UpdatedAt = time.Now()
		return &ErrSignalProcessing{Message: "order rejected", Err: fmt.Errorf(resp.Error)}
	}

	// Update signal
	signal.Status = StatusAccepted
	signal.OrderID = resp.Order.ID
	signal.UpdatedAt = time.Now()

	return nil
}

// GetSignal returns a signal by ID
func (p *DefaultProcessor) GetSignal(id string) (*Signal, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	signal, exists := p.signals[id]
	return signal, exists
}

// ListSignals returns all signals
func (p *DefaultProcessor) ListSignals() []*Signal {
	p.mu.RLock()
	defer p.mu.RUnlock()

	signals := make([]*Signal, 0, len(p.signals))
	for _, signal := range p.signals {
		signals = append(signals, signal)
	}
	return signals
}

// OnOrder handles order updates
func (p *DefaultProcessor) OnOrder(order *exchange.Order) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Find signal by client order ID
	signal, exists := p.signals[order.ClientOrderID]
	if !exists {
		return
	}

	// Update signal status
	switch exchange.OrderStatus(order.Status) {
	case exchange.OrderStatusFilled:
		signal.Status = StatusExecuted
	case exchange.OrderStatusCancelled:
		signal.Status = StatusCancelled
	case exchange.OrderStatusRejected:
		signal.Status = StatusRejected
		signal.Reason = "order rejected"
	}

	signal.UpdatedAt = time.Now()
}
