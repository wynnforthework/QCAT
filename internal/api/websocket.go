package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"qcat/internal/monitoring"
)

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	upgrader websocket.Upgrader
	clients  map[string]*Client
	mu       sync.RWMutex
	metrics  *monitoring.Metrics
}

// Client represents a WebSocket client
type Client struct {
	ID       string
	Type     string
	Conn     *websocket.Conn
	Send     chan []byte
	Handler  *WebSocketHandler
}

// Message represents a WebSocket message
type Message struct {
	Type    string      `json:"type"`
	Data    interface{} `json:"data"`
	Time    time.Time   `json:"time"`
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(upgrader websocket.Upgrader, metrics *monitoring.Metrics) *WebSocketHandler {
	handler := &WebSocketHandler{
		upgrader: upgrader,
		clients:  make(map[string]*Client),
		metrics:  metrics,
	}

	// Start broadcast goroutines
	go handler.broadcastMarketData()
	go handler.broadcastStrategyStatus()
	go handler.broadcastAlerts()

	return handler
}

// MarketStream handles market data WebSocket connections
func (h *WebSocketHandler) MarketStream(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol is required"})
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	clientID := generateClientID()
	client := &Client{
		ID:      clientID,
		Type:    "market",
		Conn:    conn,
		Send:    make(chan []byte, 256),
		Handler: h,
	}

	h.registerClient(client)
	defer h.unregisterClient(client)

	// Start client goroutines
	go client.writePump()
	go client.readPump()

	// Send initial connection message
	msg := Message{
		Type: "connected",
		Data: map[string]interface{}{
			"symbol": symbol,
			"client_id": clientID,
		},
		Time: time.Now(),
	}

	if err := client.Conn.WriteJSON(msg); err != nil {
		log.Printf("Failed to send initial message: %v", err)
		return
	}
}

// StrategyStream handles strategy status WebSocket connections
func (h *WebSocketHandler) StrategyStream(c *gin.Context) {
	strategyID := c.Param("id")
	if strategyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Strategy ID is required"})
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	clientID := generateClientID()
	client := &Client{
		ID:      clientID,
		Type:    "strategy",
		Conn:    conn,
		Send:    make(chan []byte, 256),
		Handler: h,
	}

	h.registerClient(client)
	defer h.unregisterClient(client)

	// Start client goroutines
	go client.writePump()
	go client.readPump()

	// Send initial connection message
	msg := Message{
		Type: "connected",
		Data: map[string]interface{}{
			"strategy_id": strategyID,
			"client_id":   clientID,
		},
		Time: time.Now(),
	}

	if err := client.Conn.WriteJSON(msg); err != nil {
		log.Printf("Failed to send initial message: %v", err)
		return
	}
}

// AlertsStream handles alerts WebSocket connections
func (h *WebSocketHandler) AlertsStream(c *gin.Context) {
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	clientID := generateClientID()
	client := &Client{
		ID:      clientID,
		Type:    "alerts",
		Conn:    conn,
		Send:    make(chan []byte, 256),
		Handler: h,
	}

	h.registerClient(client)
	defer h.unregisterClient(client)

	// Start client goroutines
	go client.writePump()
	go client.readPump()

	// Send initial connection message
	msg := Message{
		Type: "connected",
		Data: map[string]interface{}{
			"client_id": clientID,
		},
		Time: time.Now(),
	}

	if err := client.Conn.WriteJSON(msg); err != nil {
		log.Printf("Failed to send initial message: %v", err)
		return
	}
}

// registerClient registers a new client
func (h *WebSocketHandler) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client.ID] = client
	
	// Update metrics
	h.metrics.SetActiveConnections(float64(len(h.clients)))
}

// unregisterClient unregisters a client
func (h *WebSocketHandler) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, client.ID)
	close(client.Send)
	
	// Update metrics
	h.metrics.SetActiveConnections(float64(len(h.clients)))
}

// broadcastMarketData broadcasts market data to all market clients
func (h *WebSocketHandler) broadcastMarketData() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		h.mu.RLock()
		clients := make([]*Client, 0, len(h.clients))
		for _, client := range h.clients {
			if client.Type == "market" {
				clients = append(clients, client)
			}
		}
		h.mu.RUnlock()

		if len(clients) == 0 {
			continue
		}

		// Mock market data
		msg := Message{
			Type: "market_data",
			Data: map[string]interface{}{
				"symbol":    "BTCUSDT",
				"price":     45000.0 + (time.Now().Unix() % 1000),
				"volume":    1000.0,
				"timestamp": time.Now().Unix(),
			},
			Time: time.Now(),
		}

		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Failed to marshal market data: %v", err)
			continue
		}

		for _, client := range clients {
			select {
			case client.Send <- data:
			default:
				log.Printf("Client %s send buffer full, closing connection", client.ID)
				client.Conn.Close()
			}
		}
	}
}

// broadcastStrategyStatus broadcasts strategy status to all strategy clients
func (h *WebSocketHandler) broadcastStrategyStatus() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		h.mu.RLock()
		clients := make([]*Client, 0, len(h.clients))
		for _, client := range h.clients {
			if client.Type == "strategy" {
				clients = append(clients, client)
			}
		}
		h.mu.RUnlock()

		if len(clients) == 0 {
			continue
		}

		// Mock strategy status
		msg := Message{
			Type: "strategy_status",
			Data: map[string]interface{}{
				"strategy_id": "strategy_001",
				"status":      "running",
				"pnl":         1250.50,
				"positions":   []string{"BTCUSDT", "ETHUSDT"},
				"timestamp":   time.Now().Unix(),
			},
			Time: time.Now(),
		}

		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Failed to marshal strategy status: %v", err)
			continue
		}

		for _, client := range clients {
			select {
			case client.Send <- data:
			default:
				log.Printf("Client %s send buffer full, closing connection", client.ID)
				client.Conn.Close()
			}
		}
	}
}

// broadcastAlerts broadcasts alerts to all alert clients
func (h *WebSocketHandler) broadcastAlerts() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		h.mu.RLock()
		clients := make([]*Client, 0, len(h.clients))
		for _, client := range h.clients {
			if client.Type == "alerts" {
				clients = append(clients, client)
			}
		}
		h.mu.RUnlock()

		if len(clients) == 0 {
			continue
		}

		// Mock alerts
		msg := Message{
			Type: "alert",
			Data: map[string]interface{}{
				"level":     "info",
				"message":   "System running normally",
				"timestamp": time.Now().Unix(),
			},
			Time: time.Now(),
		}

		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Failed to marshal alert: %v", err)
			continue
		}

		for _, client := range clients {
			select {
			case client.Send <- data:
			default:
				log.Printf("Client %s send buffer full, closing connection", client.ID)
				client.Conn.Close()
			}
		}
	}
}

// writePump pumps messages from the send channel to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.Handler.unregisterClient(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle incoming messages if needed
		log.Printf("Received message from client %s: %s", c.ID, string(message))
	}
}

// generateClientID generates a unique client ID
func generateClientID() string {
	return fmt.Sprintf("client_%d", time.Now().UnixNano())
}
