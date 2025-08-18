package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// MessageQueue defines the interface for inter-process communication
type MessageQueue interface {
	Publish(topic string, message interface{}) error
	Subscribe(topic string, handler MessageHandler) error
	Unsubscribe(topic string) error
	Close() error
}

// MessageHandler defines the function signature for message handlers
type MessageHandler func(topic string, message []byte) error

// Message represents a message in the queue
type Message struct {
	ID        string      `json:"id"`
	Topic     string      `json:"topic"`
	Payload   interface{} `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
	Retries   int         `json:"retries"`
}

// InMemoryMessageQueue implements MessageQueue using in-memory channels
type InMemoryMessageQueue struct {
	subscribers map[string][]MessageHandler
	messages    chan Message
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// NewInMemoryMessageQueue creates a new in-memory message queue
func NewInMemoryMessageQueue(bufferSize int) *InMemoryMessageQueue {
	ctx, cancel := context.WithCancel(context.Background())
	
	mq := &InMemoryMessageQueue{
		subscribers: make(map[string][]MessageHandler),
		messages:    make(chan Message, bufferSize),
		mu:          sync.RWMutex{},
		ctx:         ctx,
		cancel:      cancel,
	}
	
	// Start message processing
	mq.wg.Add(1)
	go mq.processMessages()
	
	return mq
}

// Publish publishes a message to a topic
func (mq *InMemoryMessageQueue) Publish(topic string, message interface{}) error {
	msg := Message{
		ID:        generateMessageID(),
		Topic:     topic,
		Payload:   message,
		Timestamp: time.Now(),
		Retries:   0,
	}
	
	select {
	case mq.messages <- msg:
		return nil
	case <-mq.ctx.Done():
		return fmt.Errorf("message queue is closed")
	default:
		return fmt.Errorf("message queue is full")
	}
}

// Subscribe subscribes to a topic with a message handler
func (mq *InMemoryMessageQueue) Subscribe(topic string, handler MessageHandler) error {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	
	if mq.subscribers[topic] == nil {
		mq.subscribers[topic] = make([]MessageHandler, 0)
	}
	
	mq.subscribers[topic] = append(mq.subscribers[topic], handler)
	return nil
}

// Unsubscribe removes all handlers for a topic
func (mq *InMemoryMessageQueue) Unsubscribe(topic string) error {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	
	delete(mq.subscribers, topic)
	return nil
}

// processMessages processes messages from the queue
func (mq *InMemoryMessageQueue) processMessages() {
	defer mq.wg.Done()
	
	for {
		select {
		case msg := <-mq.messages:
			mq.handleMessage(msg)
		case <-mq.ctx.Done():
			return
		}
	}
}

// handleMessage handles a single message
func (mq *InMemoryMessageQueue) handleMessage(msg Message) {
	mq.mu.RLock()
	handlers := mq.subscribers[msg.Topic]
	mq.mu.RUnlock()
	
	if len(handlers) == 0 {
		return // No subscribers for this topic
	}
	
	// Serialize message payload
	payload, err := json.Marshal(msg.Payload)
	if err != nil {
		fmt.Printf("Failed to serialize message payload: %v\n", err)
		return
	}
	
	// Call all handlers for this topic
	for _, handler := range handlers {
		go func(h MessageHandler) {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("Message handler panic for topic %s: %v\n", msg.Topic, r)
				}
			}()
			
			if err := h(msg.Topic, payload); err != nil {
				fmt.Printf("Message handler error for topic %s: %v\n", msg.Topic, err)
				// Could implement retry logic here
			}
		}(handler)
	}
}

// Close closes the message queue
func (mq *InMemoryMessageQueue) Close() error {
	mq.cancel()
	mq.wg.Wait()
	close(mq.messages)
	return nil
}

// RedisMessageQueue implements MessageQueue using Redis pub/sub
type RedisMessageQueue struct {
	client      *redis.Client
	pubsub      *redis.PubSub
	subscribers map[string][]MessageHandler
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// NewRedisMessageQueue creates a new Redis-based message queue
func NewRedisMessageQueue(redisAddr string) *RedisMessageQueue {
	// Parse Redis address and create client
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   0, // Use default DB
	})
	
	ctx, cancel := context.WithCancel(context.Background())
	
	rmq := &RedisMessageQueue{
		client:      rdb,
		subscribers: make(map[string][]MessageHandler),
		ctx:         ctx,
		cancel:      cancel,
	}
	
	return rmq
}

// Publish publishes a message using Redis
func (rmq *RedisMessageQueue) Publish(topic string, message interface{}) error {
	// Serialize message to JSON
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	
	// Publish to Redis
	err = rmq.client.Publish(rmq.ctx, topic, data).Err()
	if err != nil {
		return fmt.Errorf("failed to publish message to Redis: %w", err)
	}
	
	return nil
}

// Subscribe subscribes to a topic using Redis
func (rmq *RedisMessageQueue) Subscribe(topic string, handler MessageHandler) error {
	rmq.mu.Lock()
	defer rmq.mu.Unlock()
	
	// Add handler to subscribers list
	rmq.subscribers[topic] = append(rmq.subscribers[topic], handler)
	
	// If this is the first subscriber for this topic, start Redis subscription
	if len(rmq.subscribers[topic]) == 1 {
		if rmq.pubsub == nil {
			rmq.pubsub = rmq.client.PSubscribe(rmq.ctx, topic)
		} else {
			rmq.pubsub.Subscribe(rmq.ctx, topic)
		}
		
		// Start message listener for this topic
		rmq.wg.Add(1)
		go rmq.listenForMessages(topic)
	}
	
	return nil
}

// Unsubscribe unsubscribes from a topic
func (rmq *RedisMessageQueue) Unsubscribe(topic string) error {
	rmq.mu.Lock()
	defer rmq.mu.Unlock()
	
	// Remove all handlers for this topic
	delete(rmq.subscribers, topic)
	
	// Unsubscribe from Redis
	if rmq.pubsub != nil {
		err := rmq.pubsub.Unsubscribe(rmq.ctx, topic)
		if err != nil {
			return fmt.Errorf("failed to unsubscribe from Redis: %w", err)
		}
	}
	
	return nil
}

// Close closes the Redis message queue
func (rmq *RedisMessageQueue) Close() error {
	// Cancel context to stop all operations
	rmq.cancel()
	
	// Wait for all goroutines to finish
	rmq.wg.Wait()
	
	// Close Redis pubsub
	if rmq.pubsub != nil {
		err := rmq.pubsub.Close()
		if err != nil {
			return fmt.Errorf("failed to close Redis pubsub: %w", err)
		}
	}
	
	// Close Redis client
	err := rmq.client.Close()
	if err != nil {
		return fmt.Errorf("failed to close Redis client: %w", err)
	}
	
	return nil
}

// listenForMessages listens for messages on a specific topic
func (rmq *RedisMessageQueue) listenForMessages(topic string) {
	defer rmq.wg.Done()
	
	for {
		select {
		case <-rmq.ctx.Done():
			return
		default:
			if rmq.pubsub == nil {
				return
			}
			
			// Receive message with timeout
			msg, err := rmq.pubsub.ReceiveTimeout(rmq.ctx, time.Second)
			if err != nil {
				// Check if context was cancelled
				if rmq.ctx.Err() != nil {
					return
				}
				// Continue on timeout or other errors
				continue
			}
			
			// Process the message
			switch m := msg.(type) {
			case *redis.Message:
				rmq.handleMessage(m.Channel, []byte(m.Payload))
			case *redis.Subscription:
				// Subscription confirmation, ignore
			}
		}
	}
}

// handleMessage handles incoming Redis messages
func (rmq *RedisMessageQueue) handleMessage(topic string, payload []byte) {
	rmq.mu.RLock()
	handlers := rmq.subscribers[topic]
	rmq.mu.RUnlock()
	
	// Call all handlers for this topic
	for _, handler := range handlers {
		go func(h MessageHandler) {
			if err := h(topic, payload); err != nil {
				// Log error but don't stop processing
				fmt.Printf("Error handling message for topic %s: %v\n", topic, err)
			}
		}(handler)
	}
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return fmt.Sprintf("msg-%d", time.Now().UnixNano())
}

// Common message types for inter-process communication

// OptimizationRequest represents a request to start optimization
type OptimizationRequest struct {
	RequestID      string                 `json:"request_id"`
	StrategyID     string                 `json:"strategy_id"`
	Parameters     map[string]interface{} `json:"parameters"`
	TimeRange      TimeRange              `json:"time_range"`
	Optimization   OptimizationConfig     `json:"optimization"`
	Method         string                 `json:"method"`
	GridSize       int                    `json:"grid_size,omitempty"`
	MaxIterations  int                    `json:"max_iterations,omitempty"`
}

// OptimizationResult represents the result of an optimization
type OptimizationResult struct {
	RequestID      string                 `json:"request_id"`
	StrategyID     string                 `json:"strategy_id"`
	BestParameters map[string]interface{} `json:"best_parameters"`
	Performance    PerformanceMetrics     `json:"performance"`
	Status         string                 `json:"status"`
	Error          string                 `json:"error,omitempty"`
	Duration       time.Duration          `json:"duration,omitempty"`
	Iterations     int                    `json:"iterations,omitempty"`
}

// TradeSignal represents a trading signal
type TradeSignal struct {
	SignalID   string    `json:"signal_id"`
	StrategyID string    `json:"strategy_id"`
	Symbol     string    `json:"symbol"`
	Action     string    `json:"action"` // BUY, SELL, CLOSE
	Quantity   float64   `json:"quantity"`
	Price      float64   `json:"price,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// MarketDataUpdate represents a market data update
type MarketDataUpdate struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Volume    float64   `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`
}

// TimeRange represents a time range for backtesting
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// OptimizationConfig represents optimization configuration
type OptimizationConfig struct {
	Method     string                 `json:"method"`
	Parameters map[string]interface{} `json:"parameters"`
}

// PerformanceMetrics represents performance metrics
type PerformanceMetrics struct {
	TotalReturn  float64 `json:"total_return"`
	SharpeRatio  float64 `json:"sharpe_ratio"`
	MaxDrawdown  float64 `json:"max_drawdown"`
	WinRate      float64 `json:"win_rate"`
	TradeCount   int     `json:"trade_count"`
}