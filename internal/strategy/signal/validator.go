package signal

import (
	"fmt"
	"time"

	"qcat/internal/exchange"
)

// DefaultValidator implements the default signal validation logic
type DefaultValidator struct {
	exchange exchange.Exchange
}

// NewDefaultValidator creates a new default validator
func NewDefaultValidator(exchange exchange.Exchange) *DefaultValidator {
	return &DefaultValidator{
		exchange: exchange,
	}
}

// Validate validates a signal
func (v *DefaultValidator) Validate(signal *Signal) error {
	// Validate required fields
	if signal.Strategy == "" {
		return &ErrInvalidSignal{Field: "strategy", Message: "strategy is required"}
	}
	if signal.Symbol == "" {
		return &ErrInvalidSignal{Field: "symbol", Message: "symbol is required"}
	}
	if signal.Type == "" {
		return &ErrInvalidSignal{Field: "type", Message: "type is required"}
	}
	if signal.Source == "" {
		return &ErrInvalidSignal{Field: "source", Message: "source is required"}
	}
	if signal.Side == "" {
		return &ErrInvalidSignal{Field: "side", Message: "side is required"}
	}
	if signal.OrderType == "" {
		return &ErrInvalidSignal{Field: "order_type", Message: "order type is required"}
	}
	if signal.Quantity <= 0 {
		return &ErrInvalidSignal{Field: "quantity", Message: "quantity must be positive"}
	}

	// Validate price for limit orders
	if signal.OrderType == exchange.OrderTypeLimit && signal.Price <= 0 {
		return &ErrInvalidSignal{Field: "price", Message: "price must be positive for limit orders"}
	}

	// Validate stop price for stop orders
	if signal.OrderType == exchange.OrderTypeStop && signal.StopPrice <= 0 {
		return &ErrInvalidSignal{Field: "stop_price", Message: "stop price must be positive for stop orders"}
	}

	// Validate leverage
	if signal.Leverage < 1 {
		return &ErrInvalidSignal{Field: "leverage", Message: "leverage must be at least 1"}
	}

	// Validate margin type
	if signal.MarginType != exchange.MarginTypeCross && signal.MarginType != exchange.MarginTypeIsolated {
		return &ErrInvalidSignal{Field: "margin_type", Message: "invalid margin type"}
	}

	// Validate expiration time
	if !signal.ExpiresAt.IsZero() && signal.ExpiresAt.Before(time.Now()) {
		return &ErrInvalidSignal{Field: "expires_at", Message: "expiration time must be in the future"}
	}

	// Validate symbol info
	symbolInfo, err := v.exchange.GetSymbolInfo(nil, signal.Symbol)
	if err != nil {
		return &ErrSignalValidation{Message: "failed to get symbol info", Err: err}
	}

	// Validate price precision
	if signal.OrderType == exchange.OrderTypeLimit {
		precision := 1.0 / float64(symbolInfo.PricePrecision)
		if signal.Price != float64(int(signal.Price/precision))*precision {
			return &ErrInvalidSignal{Field: "price", Message: fmt.Sprintf("price must have at most %d decimal places", symbolInfo.PricePrecision)}
		}
	}

	// Validate quantity precision
	precision := 1.0 / float64(symbolInfo.QuantityPrecision)
	if signal.Quantity != float64(int(signal.Quantity/precision))*precision {
		return &ErrInvalidSignal{Field: "quantity", Message: fmt.Sprintf("quantity must have at most %d decimal places", symbolInfo.QuantityPrecision)}
	}

	// Validate risk limits
	riskLimits, err := v.exchange.GetRiskLimits(nil, signal.Symbol)
	if err != nil {
		return &ErrSignalValidation{Message: "failed to get risk limits", Err: err}
	}

	if signal.Leverage > riskLimits.MaxLeverage {
		return &ErrInvalidSignal{Field: "leverage", Message: fmt.Sprintf("leverage cannot exceed %d", riskLimits.MaxLeverage)}
	}

	// Validate position
	position, err := v.exchange.GetPosition(nil, signal.Symbol)
	if err != nil && err.Error() != "position not found" {
		return &ErrSignalValidation{Message: "failed to get position", Err: err}
	}

	if position != nil {
		// Validate reduce only
		if signal.ReduceOnly {
			if signal.Side == exchange.OrderSideBuy && position.Side != exchange.PositionSideShort {
				return &ErrInvalidSignal{Field: "side", Message: "reduce only buy order requires short position"}
			}
			if signal.Side == exchange.OrderSideSell && position.Side != exchange.PositionSideLong {
				return &ErrInvalidSignal{Field: "side", Message: "reduce only sell order requires long position"}
			}
		}

		// Validate exposure
		exposure := position.Quantity * position.EntryPrice
		if (signal.Side == exchange.OrderSideBuy && position.Side == exchange.PositionSideLong) ||
			(signal.Side == exchange.OrderSideSell && position.Side == exchange.PositionSideShort) {
			exposure += signal.Quantity * signal.Price
		}
		if exposure > riskLimits.MaxExposure {
			return &ErrInvalidSignal{Field: "quantity", Message: fmt.Sprintf("total exposure cannot exceed %f", riskLimits.MaxExposure)}
		}
	}

	return nil
}
