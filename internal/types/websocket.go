package types

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
	delete(c.subscriptions, key)
	delete(c.handlers, key)

	if c.conn != nil {
		msg := map[string]interface{}{
			"method": "UNSUBSCRIBE",
			"params": []string{key},
			"id":     key,
		}
		return c.conn.WriteJSON(msg)
	}

	return nil
}

// Close closes the WebSocket connection
func (c *WSClient) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	close(c.done)

	if c.conn != nil {
		return c.conn.Close()
	}

	return nil
}

// handleMessages handles incoming WebSocket messages
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
				time.Sleep(100 * time.Millisecond)
				continue
			}

			_, message, err := conn.ReadMessage()
			if err != nil {
				c.OnError(fmt.Errorf("failed to read message: %w", err))
				continue
			}

			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err != nil {
				c.OnError(fmt.Errorf("failed to unmarshal message: %w", err))
				continue
			}

			// Handle the message
			c.handleMessage(msg)
		}
	}
}

// handleMessage processes a single message
func (c *WSClient) handleMessage(msg map[string]interface{}) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// Find the appropriate handler based on the message
	for key, handler := range c.handlers {
		if handler != nil {
			if err := handler(msg); err != nil {
				c.OnError(fmt.Errorf("handler error for %s: %w", key, err))
			}
		}
	}
}

// keepAlive sends periodic ping messages
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
					c.OnError(fmt.Errorf("failed to send ping: %w", err))
				}
			}
		}
	}
}
