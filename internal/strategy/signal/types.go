package signal

import (
	"time"

	"qcat/internal/exchange"
)

// Type represents the signal type
type Type string

const (
	TypeEntry Type = "entry"
	TypeExit  Type = "exit"
	TypeStop  Type = "stop"
	TypeLimit Type = "limit"
)

// Source represents the signal source
type Source string

const (
	SourceStrategy Source = "strategy"
	SourceManual   Source = "manual"
	SourceSystem   Source = "system"
)

// Status represents the signal status
type Status string

const (
	StatusPending   Status = "pending"
	StatusAccepted  Status = "accepted"
	StatusRejected  Status = "rejected"
	StatusExecuted  Status = "executed"
	StatusCancelled Status = "cancelled"
	StatusExpired   Status = "expired"
)

// Signal represents a trading signal
type Signal struct {
	ID          string                 `json:"id"`
	Strategy    string                 `json:"strategy"`
	Symbol      string                 `json:"symbol"`
	Type        Type                   `json:"type"`
	Source      Source                 `json:"source"`
	Side        exchange.OrderSide     `json:"side"`
	OrderType   exchange.OrderType     `json:"order_type"`
	Price       float64                `json:"price"`
	StopPrice   float64                `json:"stop_price"`
	Quantity    float64                `json:"quantity"`
	Leverage    int                    `json:"leverage"`
	MarginType  exchange.MarginType    `json:"margin_type"`
	ReduceOnly  bool                   `json:"reduce_only"`
	PostOnly    bool                   `json:"post_only"`
	TimeInForce string                 `json:"time_in_force"`
	Status      Status                 `json:"status"`
	Reason      string                 `json:"reason"`
	OrderID     string                 `json:"order_id"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	ExpiresAt   time.Time              `json:"expires_at"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Validator represents a signal validator
type Validator interface {
	// Validate validates a signal
	Validate(signal *Signal) error
}

// Processor represents a signal processor
type Processor interface {
	// Process processes a signal
	Process(signal *Signal) error
}

// Error types
type ErrInvalidSignal struct {
	Field   string
	Message string
}

func (e ErrInvalidSignal) Error() string {
	return "invalid signal: " + e.Field + " - " + e.Message
}

type ErrSignalProcessing struct {
	Message string
	Err     error
}

func (e ErrSignalProcessing) Error() string {
	if e.Err != nil {
		return "signal processing error: " + e.Message + ": " + e.Err.Error()
	}
	return "signal processing error: " + e.Message
}

func (e ErrSignalProcessing) Unwrap() error {
	return e.Err
}

type ErrSignalValidation struct {
	Message string
	Err     error
}

func (e ErrSignalValidation) Error() string {
	if e.Err != nil {
		return "signal validation error: " + e.Message + ": " + e.Err.Error()
	}
	return "signal validation error: " + e.Message
}

func (e ErrSignalValidation) Unwrap() error {
	return e.Err
}
