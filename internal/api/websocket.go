package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	upgrader websocket.Upgrader
	clients  map[string]*Client
	mu       sync.RWMutex
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
func NewWebSocketHandler(upgrader websocket.Upgrader) *WebSocketHandler {
	handler := &WebSocketHandler{
		upgrader: upgrader,
		clients:  make(map[string]*Client),
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
	log.Printf("Client %s connected (%s)", client.ID, client.Type)
}

// unregisterClient unregisters a client
func (h *WebSocketHandler) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	if _, ok := h.clients[client.ID]; ok {
		delete(h.clients, client.ID)
		close(client.Send)
		log.Printf("Client %s disconnected", client.ID)
	}
}

// broadcastMarketData broadcasts market data to all market clients
func (h *WebSocketHandler) broadcastMarketData() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// TODO: Get real market data from market ingestor
		marketData := map[string]interface{}{
			"symbol": "BTCUSDT",
			"price":  50000.0 + (time.Now().Unix() % 1000),
			"volume": 1000.0,
			"change": 0.02,
		}

		msg := Message{
			Type: "market_data",
			Data: marketData,
			Time: time.Now(),
		}

		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Failed to marshal market data: %v", err)
			continue
		}

		h.broadcastToType("market", data)
	}
}

// broadcastStrategyStatus broadcasts strategy status to all strategy clients
func (h *WebSocketHandler) broadcastStrategyStatus() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// TODO: Get real strategy status from strategy runner
		strategyStatus := map[string]interface{}{
			"strategy_id": "strategy_1",
			"status":      "running",
			"pnl":         1000.0,
			"positions":   []map[string]interface{}{},
		}

		msg := Message{
			Type: "strategy_status",
			Data: strategyStatus,
			Time: time.Now(),
		}

		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Failed to marshal strategy status: %v", err)
			continue
		}

		h.broadcastToType("strategy", data)
	}
}

// broadcastAlerts broadcasts alerts to all alert clients
func (h *WebSocketHandler) broadcastAlerts() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// TODO: Get real alerts from alert manager
		alert := map[string]interface{}{
			"id":       "alert_1",
			"severity": "warning",
			"message":  "Strategy performance below threshold",
			"time":     time.Now(),
		}

		msg := Message{
			Type: "alert",
			Data: alert,
			Time: time.Now(),
		}

		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Failed to marshal alert: %v", err)
			continue
		}

		h.broadcastToType("alerts", data)
	}
}

// broadcastToType broadcasts message to clients of specific type
func (h *WebSocketHandler) broadcastToType(clientType string, data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		if client.Type == clientType {
			select {
			case client.Send <- data:
			default:
				// Channel is full, close connection
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
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
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

		// Handle incoming message
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			continue
		}

		// Echo back for now
		response := Message{
			Type: "echo",
			Data: msg.Data,
			Time: time.Now(),
		}

		responseData, err := json.Marshal(response)
		if err != nil {
			log.Printf("Failed to marshal response: %v", err)
			continue
		}

		select {
		case c.Send <- responseData:
		default:
			// Channel is full, close connection
			return
		}
	}
}

// generateClientID generates a unique client ID
func generateClientID() string {
	return "client_" + time.Now().Format("20060102150405") + "_" + time.Now().Format("000000000")
}
