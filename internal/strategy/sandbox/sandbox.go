package sandbox

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"qcat/internal/exchange"
	. "qcat/internal/exchange"
	. "qcat/internal/strategy"
)

// Sandbox provides an isolated environment for strategy execution
type Sandbox struct {
	strategy   Strategy
	config     *Config
	exchange   exchange.Exchange
	marketData chan interface{}
	signals    chan *Signal
	orders     chan *Order
	positions  chan *Position
	errors     chan error
	done       chan struct{}
	state      State
	mu         sync.RWMutex
}

// NewSandbox creates a new strategy sandbox
func NewSandbox(strategy Strategy, config *Config, exchange exchange.Exchange) *Sandbox {
	return &Sandbox{
		strategy:   strategy,
		config:     config,
		exchange:   exchange,
		marketData: make(chan interface{}, 1000),
		signals:    make(chan *Signal, 100),
		orders:     make(chan *Order, 100),
		positions:  make(chan *Position, 100),
		errors:     make(chan error, 100),
		done:       make(chan struct{}),
		state:      StateInitializing,
	}
}

// GetConfig returns the sandbox configuration
func (s *Sandbox) GetConfig() *Config {
	return s.config
}

// Start starts the strategy sandbox
func (s *Sandbox) Start(ctx context.Context) error {
	// Initialize strategy
	if err := s.strategy.Initialize(ctx, s.config.Params); err != nil {
		return fmt.Errorf("failed to initialize strategy: %w", err)
	}

	// Set up strategy context
	strategyCtx := &Context{
		Mode:      s.config.Mode,
		Strategy:  s.config.Name,
		Symbol:    s.config.Symbol,
		Exchange:  s.exchange,
		StartTime: time.Now(),
		Params:    s.config.Params,
	}

	// Set strategy context
	if bs, ok := s.strategy.(interface{ SetContext(*Context) }); ok {
		bs.SetContext(strategyCtx)
	}

	// Start strategy
	if err := s.strategy.Start(ctx); err != nil {
		return fmt.Errorf("failed to start strategy: %w", err)
	}

	// Start event processing
	go s.processEvents(ctx)

	// Update state
	s.setState(StateRunning)

	return nil
}

// Stop stops the strategy sandbox
func (s *Sandbox) Stop(ctx context.Context) error {
	// Stop strategy
	if err := s.strategy.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop strategy: %w", err)
	}

	// Signal done
	close(s.done)

	// Update state
	s.setState(StateStopped)

	return nil
}

// GetState returns the current sandbox state
func (s *Sandbox) GetState() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// setState sets the sandbox state
func (s *Sandbox) setState(state State) {
	s.mu.Lock()
	s.state = state
	s.mu.Unlock()
}

// OnMarketData handles market data updates
func (s *Sandbox) OnMarketData(data interface{}) {
	select {
	case s.marketData <- data:
	default:
		log.Printf("Market data channel full, dropping update")
	}
}

// OnSignal handles trading signals
func (s *Sandbox) OnSignal(signal *Signal) {
	select {
	case s.signals <- signal:
	default:
		log.Printf("Signal channel full, dropping signal")
	}
}

// OnOrder handles order updates
func (s *Sandbox) OnOrder(order *Order) {
	select {
	case s.orders <- order:
	default:
		log.Printf("Order channel full, dropping update")
	}
}

// OnPosition handles position updates
func (s *Sandbox) OnPosition(position *Position) {
	select {
	case s.positions <- position:
	default:
		log.Printf("Position channel full, dropping update")
	}
}

// OnError handles errors
func (s *Sandbox) OnError(err error) {
	select {
	case s.errors <- err:
	default:
		log.Printf("Error channel full, dropping error: %v", err)
	}
}

// processEvents processes strategy events
func (s *Sandbox) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.done:
			return
		case data := <-s.marketData:
			if err := s.strategy.OnTick(ctx, data); err != nil {
				s.OnError(fmt.Errorf("error processing market data: %w", err))
			}
		case signal := <-s.signals:
			if err := s.strategy.OnSignal(ctx, signal); err != nil {
				s.OnError(fmt.Errorf("error processing signal: %w", err))
			}
		case order := <-s.orders:
			if err := s.strategy.OnOrder(ctx, order); err != nil {
				s.OnError(fmt.Errorf("error processing order: %w", err))
			}
		case position := <-s.positions:
			if err := s.strategy.OnPosition(ctx, position); err != nil {
				s.OnError(fmt.Errorf("error processing position: %w", err))
			}
		case err := <-s.errors:
			log.Printf("Strategy error: %v", err)
			s.setState(StateError)
		}
	}
}

// GetResult returns the strategy execution result
func (s *Sandbox) GetResult() *Result {
	return s.strategy.GetResult()
}

// Validate validates the sandbox configuration
func (s *Sandbox) Validate() error {
	// Validate strategy
	if s.strategy == nil {
		return fmt.Errorf("strategy is required")
	}

	// Validate config
	if s.config == nil {
		return fmt.Errorf("config is required")
	}

	// Validate exchange
	if s.exchange == nil {
		return fmt.Errorf("exchange is required")
	}

	// Validate strategy config
	if bs, ok := s.strategy.(interface{ Validate() error }); ok {
		if err := bs.Validate(); err != nil {
			return fmt.Errorf("invalid strategy config: %w", err)
		}
	}

	return nil
}
