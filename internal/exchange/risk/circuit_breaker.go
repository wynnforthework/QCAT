package risk

import (
	"context"
	"fmt"
	"sync"
	"time"

	"qcat/internal/exchange"
)

// CircuitBreaker implements trading circuit breaker
type CircuitBreaker struct {
	exchange    exchange.Exchange
	thresholds  map[string]*Threshold
	states      map[string]*State
	subscribers map[string][]chan *Event
	mu          sync.RWMutex
}

// Threshold defines circuit breaker thresholds
type Threshold struct {
	Symbol         string
	PriceChange    float64       // 价格变动阈值
	TimeWindow     time.Duration // 时间窗口
	CooldownPeriod time.Duration // 冷却期
	UpdatedAt      time.Time
}

// State represents circuit breaker state
type State struct {
	Symbol      string
	Triggered   bool
	TriggeredAt time.Time
	ResumeAt    time.Time
	BasePrice   float64
	UpdatedAt   time.Time
}

// Event represents a circuit breaker event
type Event struct {
	Symbol      string
	Type        EventType
	Message     string
	Threshold   float64
	Current     float64
	TriggeredAt time.Time
}

// EventType defines the type of circuit breaker event
type EventType int

const (
	EventTypeTriggered EventType = iota
	EventTypeResumed
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(ex exchange.Exchange) *CircuitBreaker {
	return &CircuitBreaker{
		exchange:    ex,
		thresholds:  make(map[string]*Threshold),
		states:      make(map[string]*State),
		subscribers: make(map[string][]chan *Event),
	}
}

// SetThreshold sets circuit breaker threshold
func (cb *CircuitBreaker) SetThreshold(symbol string, threshold *Threshold) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if threshold.PriceChange <= 0 {
		return fmt.Errorf("invalid price change threshold")
	}
	if threshold.TimeWindow <= 0 {
		return fmt.Errorf("invalid time window")
	}
	if threshold.CooldownPeriod <= 0 {
		return fmt.Errorf("invalid cooldown period")
	}

	threshold.UpdatedAt = time.Now()
	cb.thresholds[symbol] = threshold

	// Initialize state
	cb.states[symbol] = &State{
		Symbol:    symbol,
		UpdatedAt: time.Now(),
	}

	return nil
}

// Subscribe subscribes to circuit breaker events
func (cb *CircuitBreaker) Subscribe(symbol string) chan *Event {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	ch := make(chan *Event, 100)
	cb.subscribers[symbol] = append(cb.subscribers[symbol], ch)
	return ch
}

// Unsubscribe removes a subscription
func (cb *CircuitBreaker) Unsubscribe(symbol string, ch chan *Event) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	subs := cb.subscribers[symbol]
	for i, sub := range subs {
		if sub == ch {
			cb.subscribers[symbol] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}
}

// CheckPrice checks if circuit breaker should be triggered
func (cb *CircuitBreaker) CheckPrice(ctx context.Context, symbol string, price float64) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	threshold, exists := cb.thresholds[symbol]
	if !exists {
		return fmt.Errorf("threshold not found for symbol: %s", symbol)
	}

	state := cb.states[symbol]
	if state == nil {
		return fmt.Errorf("state not found for symbol: %s", symbol)
	}

	// Check if in cooldown period
	if state.Triggered && time.Now().Before(state.ResumeAt) {
		return nil
	}

	// Reset if cooldown period has passed
	if state.Triggered && time.Now().After(state.ResumeAt) {
		state.Triggered = false
		state.BasePrice = price
		state.UpdatedAt = time.Now()

		cb.notifySubscribers(symbol, &Event{
			Symbol:    symbol,
			Type:      EventTypeResumed,
			Message:   "Circuit breaker resumed",
			UpdatedAt: time.Now(),
		})
	}

	// Initialize base price if not set
	if state.BasePrice == 0 {
		state.BasePrice = price
		state.UpdatedAt = time.Now()
		return nil
	}

	// Check price change within time window
	if time.Since(state.UpdatedAt) <= threshold.TimeWindow {
		priceChange := (price - state.BasePrice) / state.BasePrice
		if abs(priceChange) >= threshold.PriceChange {
			// Trigger circuit breaker
			state.Triggered = true
			state.TriggeredAt = time.Now()
			state.ResumeAt = time.Now().Add(threshold.CooldownPeriod)

			// Cancel all orders
			if err := cb.exchange.CancelAllOrders(ctx, symbol); err != nil {
				return fmt.Errorf("failed to cancel orders: %w", err)
			}

			// Notify subscribers
			cb.notifySubscribers(symbol, &Event{
				Symbol:      symbol,
				Type:        EventTypeTriggered,
				Message:     fmt.Sprintf("Circuit breaker triggered: price change %.2f%% >= %.2f%%", priceChange*100, threshold.PriceChange*100),
				Threshold:   threshold.PriceChange,
				Current:     priceChange,
				TriggeredAt: state.TriggeredAt,
			})
		}
	} else {
		// Reset base price for new time window
		state.BasePrice = price
		state.UpdatedAt = time.Now()
	}

	return nil
}

// IsTriggered checks if circuit breaker is triggered
func (cb *CircuitBreaker) IsTriggered(symbol string) bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	state, exists := cb.states[symbol]
	if !exists {
		return false
	}
	return state.Triggered
}

// GetState returns circuit breaker state
func (cb *CircuitBreaker) GetState(symbol string) (*State, error) {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	state, exists := cb.states[symbol]
	if !exists {
		return nil, fmt.Errorf("state not found for symbol: %s", symbol)
	}
	return state, nil
}

// notifySubscribers notifies all subscribers of an event
func (cb *CircuitBreaker) notifySubscribers(symbol string, event *Event) {
	subs := cb.subscribers[symbol]
	for _, ch := range subs {
		select {
		case ch <- event:
		default:
			// Channel is full, skip
		}
	}
}

// abs returns the absolute value of x
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
