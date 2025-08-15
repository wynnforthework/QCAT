package paper

import (
	"fmt"
	"time"

	"qcat/internal/exchange"
)

// Account represents a paper trading account
type Account struct {
	Balances  map[string]float64
	Positions map[string]*Position
	Orders    map[string]*Order
}

// Position represents a paper trading position
type Position struct {
	Symbol        string
	Side          exchange.PositionSide
	Quantity      float64
	EntryPrice    float64
	Leverage      int
	MarginType    exchange.MarginType
	UnrealizedPnL float64
	UpdatedAt     time.Time
}

// Order represents a paper trading order
type Order struct {
	ID            string
	ClientOrderID string
	Symbol        string
	Side          exchange.OrderSide
	Type          exchange.OrderType
	Price         float64
	StopPrice     float64
	Quantity      float64
	FilledQty     float64
	RemainingQty  float64
	Status        exchange.OrderStatus
	ReduceOnly    bool
	PostOnly      bool
	TimeInForce   string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// OrderBook represents a simplified order book for paper trading
type OrderBook struct {
	Symbol    string
	Bids      []Level
	Asks      []Level
	UpdatedAt time.Time
}

// Level represents a price level in the order book
type Level struct {
	Price    float64
	Quantity float64
}

// Trade represents a paper trading execution
type Trade struct {
	ID        string
	OrderID   string
	Symbol    string
	Side      exchange.OrderSide
	Price     float64
	Quantity  float64
	Fee       float64
	FeeCoin   string
	Timestamp time.Time
}

// Error types
type ErrInsufficientBalance struct {
	Asset    string
	Required float64
	Current  float64
}

func (e ErrInsufficientBalance) Error() string {
	return fmt.Sprintf("insufficient balance: required %f %s, current %f", e.Required, e.Asset, e.Current)
}

type ErrInvalidOrder struct {
	Message string
}

func (e ErrInvalidOrder) Error() string {
	return "invalid order: " + e.Message
}

type ErrOrderNotFound struct {
	ID string
}

func (e ErrOrderNotFound) Error() string {
	return "order not found: " + e.ID
}

type ErrPositionNotFound struct {
	Symbol string
}

func (e ErrPositionNotFound) Error() string {
	return "position not found: " + e.Symbol
}

type ErrInvalidPosition struct {
	Message string
}

func (e ErrInvalidPosition) Error() string {
	return "invalid position: " + e.Message
}
