package market

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WSClient represents a WebSocket client
type WSClient struct {
	url            string
	conn           *websocket.Conn
	subscriptions  map[string]WSSubscription
	handlers       map[string]MarketDataHandler
	reconnectDelay time.Duration
	maxRetries     int
	mutex          sync.RWMutex
	done           chan struct{}

	// Error handling
	OnError   func(error)
	OnConnect func()
}

// NewWSClient creates a new WebSocket client
func NewWSClient(url string) *WSClient {
	return &WSClient{
		url:            url,
		subscriptions:  make(map[string]WSSubscription),
		handlers:       make(map[string]MarketDataHandler),
		reconnectDelay: 5 * time.Second,
		maxRetries:     3,
		done:           make(chan struct{}),
		OnError: func(err error) {
			log.Printf("WebSocket error: %v", err)
		},
		OnConnect: func() {
			log.Println("WebSocket connected")
		},
	}
}

// Connect establishes a WebSocket connection
func (c *WSClient) Connect(ctx context.Context) error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 45 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, c.url, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	c.mutex.Lock()
	c.conn = conn
	c.mutex.Unlock()

	c.OnConnect()

	// Start message handling
	go c.handleMessages()

	// Start ping/pong
	go c.keepAlive(ctx)

	return nil
}

// Subscribe adds a new subscription
func (c *WSClient) Subscribe(sub WSSubscription, handler MarketDataHandler) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	key := fmt.Sprintf("%s-%s", sub.Symbol, sub.MarketType)
	c.subscriptions[key] = sub
	c.handlers[key] = handler

	if c.conn != nil {
		msg := map[string]interface{}{
			"method": "SUBSCRIBE",
			"params": sub.Channels,
			"id":     key,
		}
		return c.conn.WriteJSON(msg)
	}

	return nil
}

// Unsubscribe removes a subscription
func (c *WSClient) Unsubscribe(symbol string, marketType MarketType) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	key := fmt.Sprintf("%s-%s", symbol, marketType)
	sub, exists := c.subscriptions[key]
	if !exists {
		return nil
	}

	if c.conn != nil {
		msg := map[string]interface{}{
			"method": "UNSUBSCRIBE",
			"params": sub.Channels,
			"id":     key,
		}
		if err := c.conn.WriteJSON(msg); err != nil {
			return err
		}
	}

	delete(c.subscriptions, key)
	delete(c.handlers, key)
	return nil
}

// Close closes the WebSocket connection
func (c *WSClient) Close() error {
	close(c.done)

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// handleMessages processes incoming WebSocket messages
func (c *WSClient) handleMessages() {
	for {
		select {
		case <-c.done:
			return
		default:
			c.mutex.RLock()
			conn := c.conn
			c.mutex.RUnlock()

			if conn == nil {
				time.Sleep(c.reconnectDelay)
				continue
			}

			_, message, err := conn.ReadMessage()
			if err != nil {
				c.OnError(fmt.Errorf("error reading message: %w", err))
				c.reconnect()
				continue
			}

			var data map[string]interface{}
			if err := json.Unmarshal(message, &data); err != nil {
				c.OnError(fmt.Errorf("error parsing message: %w", err))
				continue
			}

			// Handle the message based on its type
			if err := c.processMessage(data); err != nil {
				c.OnError(fmt.Errorf("error processing message: %w", err))
			}
		}
	}
}

// processMessage processes different types of market data messages
func (c *WSClient) processMessage(data map[string]interface{}) error {
	// Extract message type and symbol
	msgType, ok := data["e"].(string)
	if !ok {
		return fmt.Errorf("missing message type")
	}

	symbol, ok := data["s"].(string)
	if !ok {
		return fmt.Errorf("missing symbol")
	}

	// Find the appropriate handler
	c.mutex.RLock()
	handler, exists := c.handlers[symbol]
	c.mutex.RUnlock()

	if !exists {
		return nil // No handler registered for this symbol
	}

	var msg interface{}
	switch msgType {
	case "ticker":
		msg = &Ticker{}
	case "depth":
		msg = &OrderBook{}
	case "trade":
		msg = &Trade{}
	case "kline":
		msg = &Kline{}
	case "funding":
		msg = &FundingRate{}
	case "oi":
		msg = &OpenInterest{}
	default:
		return fmt.Errorf("unknown message type: %s", msgType)
	}

	// Parse the message
	msgBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error re-marshaling message: %w", err)
	}

	if err := json.Unmarshal(msgBytes, msg); err != nil {
		return fmt.Errorf("error parsing %s message: %w", msgType, err)
	}

	// Handle the message
	return handler(msg)
}

// keepAlive maintains the WebSocket connection
func (c *WSClient) keepAlive(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.done:
			return
		case <-ticker.C:
			c.mutex.RLock()
			conn := c.conn
			c.mutex.RUnlock()

			if conn != nil {
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					c.OnError(fmt.Errorf("ping error: %w", err))
					c.reconnect()
				}
			}
		}
	}
}

// reconnect attempts to reconnect the WebSocket
func (c *WSClient) reconnect() {
	c.mutex.Lock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.mutex.Unlock()

	for i := 0; i < c.maxRetries; i++ {
		ctx := context.Background()
		if err := c.Connect(ctx); err != nil {
			c.OnError(fmt.Errorf("reconnection attempt %d failed: %w", i+1, err))
			time.Sleep(c.reconnectDelay)
			continue
		}

		// Resubscribe to all channels
		c.mutex.RLock()
		subs := make([]WSSubscription, 0, len(c.subscriptions))
		for _, sub := range c.subscriptions {
			subs = append(subs, sub)
		}
		c.mutex.RUnlock()

		for _, sub := range subs {
			msg := map[string]interface{}{
				"method": "SUBSCRIBE",
				"params": sub.Channels,
				"id":     fmt.Sprintf("%s-%s", sub.Symbol, sub.MarketType),
			}
			if err := c.conn.WriteJSON(msg); err != nil {
				c.OnError(fmt.Errorf("resubscription failed: %w", err))
				continue
			}
		}

		return
	}

	c.OnError(fmt.Errorf("failed to reconnect after %d attempts", c.maxRetries))
}
