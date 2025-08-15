package stability

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// NetworkReconnectManager manages network connection reconnection
type NetworkReconnectManager struct {
	// Prometheus metrics
	reconnectAttempts prometheus.Counter
	reconnectSuccess  prometheus.Counter
	reconnectFailures prometheus.Counter
	connectionUptime  prometheus.Gauge
	lastReconnectTime prometheus.Gauge
	reconnectLatency  prometheus.Histogram

	// Configuration
	config *ReconnectConfig

	// State
	connections map[string]*ConnectionState
	mu          sync.RWMutex

	// Channels
	alertCh chan *ReconnectAlert
	stopCh  chan struct{}
}

// ReconnectConfig represents reconnection configuration
type ReconnectConfig struct {
	// Reconnection strategy
	MaxRetries        int
	InitialDelay      time.Duration
	MaxDelay          time.Duration
	BackoffMultiplier float64
	JitterFactor      float64

	// Connection monitoring
	HealthCheckInterval time.Duration
	ConnectionTimeout   time.Duration
	PingInterval        time.Duration
	PongTimeout         time.Duration

	// Alert thresholds
	MaxConsecutiveFailures int
	AlertThreshold         int
}

// ConnectionState represents the state of a network connection
type ConnectionState struct {
	ID                   string
	URL                  string
	Conn                 *websocket.Conn
	Status               ConnectionStatus
	LastConnected        time.Time
	LastDisconnected     time.Time
	ReconnectAttempts    int
	ConsecutiveFailures  int
	TotalUptime          time.Duration
	LastReconnectLatency time.Duration

	// Callbacks
	OnConnect    func(*websocket.Conn) error
	OnDisconnect func(error)
	OnMessage    func([]byte) error

	// Internal state
	stopCh chan struct{}
	mu     sync.RWMutex
}

// ConnectionStatus represents connection status
type ConnectionStatus string

const (
	StatusDisconnected ConnectionStatus = "disconnected"
	StatusConnecting   ConnectionStatus = "connecting"
	StatusConnected    ConnectionStatus = "connected"
	StatusReconnecting ConnectionStatus = "reconnecting"
	StatusFailed       ConnectionStatus = "failed"
)

// ReconnectAlert represents a reconnection alert
type ReconnectAlert struct {
	Type         ReconnectAlertType
	Message      string
	ConnectionID string
	Attempts     int
	Latency      time.Duration
	Timestamp    time.Time
}

// ReconnectAlertType represents the type of reconnection alert
type ReconnectAlertType string

const (
	AlertTypeReconnectAttempt   ReconnectAlertType = "reconnect_attempt"
	AlertTypeReconnectSuccess   ReconnectAlertType = "reconnect_success"
	AlertTypeReconnectFailure   ReconnectAlertType = "reconnect_failure"
	AlertTypeMaxRetriesExceeded ReconnectAlertType = "max_retries_exceeded"
	AlertTypeConnectionLost     ReconnectAlertType = "connection_lost"
)

// NewNetworkReconnectManager creates a new network reconnect manager
func NewNetworkReconnectManager(config *ReconnectConfig) *NetworkReconnectManager {
	if config == nil {
		config = &ReconnectConfig{
			MaxRetries:             10,
			InitialDelay:           1 * time.Second,
			MaxDelay:               5 * time.Minute,
			BackoffMultiplier:      2.0,
			JitterFactor:           0.1,
			HealthCheckInterval:    30 * time.Second,
			ConnectionTimeout:      30 * time.Second,
			PingInterval:           30 * time.Second,
			PongTimeout:            10 * time.Second,
			MaxConsecutiveFailures: 5,
			AlertThreshold:         3,
		}
	}

	nrm := &NetworkReconnectManager{
		config:      config,
		connections: make(map[string]*ConnectionState),
		alertCh:     make(chan *ReconnectAlert, 100),
		stopCh:      make(chan struct{}),
	}

	// Initialize Prometheus metrics
	nrm.initializeMetrics()

	return nrm
}

// initializeMetrics initializes Prometheus metrics
func (nrm *NetworkReconnectManager) initializeMetrics() {
	nrm.reconnectAttempts = promauto.NewCounter(prometheus.CounterOpts{
		Name: "network_reconnect_attempts_total",
		Help: "Total number of reconnection attempts",
	})

	nrm.reconnectSuccess = promauto.NewCounter(prometheus.CounterOpts{
		Name: "network_reconnect_success_total",
		Help: "Total number of successful reconnections",
	})

	nrm.reconnectFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "network_reconnect_failures_total",
		Help: "Total number of failed reconnections",
	})

	nrm.connectionUptime = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "network_connection_uptime_seconds",
		Help: "Connection uptime in seconds",
	})

	nrm.lastReconnectTime = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "network_last_reconnect_timestamp",
		Help: "Timestamp of last reconnection attempt",
	})

	nrm.reconnectLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "network_reconnect_latency_seconds",
		Help:    "Reconnection latency in seconds",
		Buckets: prometheus.ExponentialBuckets(0.1, 2, 10),
	})
}

// RegisterConnection registers a new connection for monitoring
func (nrm *NetworkReconnectManager) RegisterConnection(id, url string, callbacks *ConnectionCallbacks) {
	nrm.mu.Lock()
	defer nrm.mu.Unlock()

	connState := &ConnectionState{
		ID:           id,
		URL:          url,
		Status:       StatusDisconnected,
		OnConnect:    callbacks.OnConnect,
		OnDisconnect: callbacks.OnDisconnect,
		OnMessage:    callbacks.OnMessage,
		stopCh:       make(chan struct{}),
	}

	nrm.connections[id] = connState

	// Start connection monitoring
	go nrm.monitorConnection(connState)
}

// ConnectionCallbacks represents connection callbacks
type ConnectionCallbacks struct {
	OnConnect    func(*websocket.Conn) error
	OnDisconnect func(error)
	OnMessage    func([]byte) error
}

// Connect establishes a WebSocket connection with automatic reconnection
func (nrm *NetworkReconnectManager) Connect(ctx context.Context, id string) error {
	nrm.mu.RLock()
	connState, exists := nrm.connections[id]
	nrm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("connection %s not registered", id)
	}

	return nrm.connectWithRetry(ctx, connState)
}

// connectWithRetry attempts to connect with exponential backoff
func (nrm *NetworkReconnectManager) connectWithRetry(ctx context.Context, connState *ConnectionState) error {
	connState.mu.Lock()
	connState.Status = StatusConnecting
	connState.mu.Unlock()

	dialer := websocket.Dialer{
		HandshakeTimeout: nrm.config.ConnectionTimeout,
	}

	startTime := time.Now()
	conn, _, err := dialer.DialContext(ctx, connState.URL, nil)
	if err != nil {
		connState.mu.Lock()
		connState.Status = StatusFailed
		connState.ConsecutiveFailures++
		connState.mu.Unlock()

		nrm.reconnectFailures.Inc()
		nrm.sendAlert(&ReconnectAlert{
			Type:         AlertTypeReconnectFailure,
			Message:      fmt.Sprintf("Failed to connect: %v", err),
			ConnectionID: connState.ID,
			Attempts:     connState.ReconnectAttempts,
			Latency:      time.Since(startTime),
			Timestamp:    time.Now(),
		})

		return fmt.Errorf("failed to connect: %w", err)
	}

	// Configure connection
	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	connState.mu.Lock()
	connState.Conn = conn
	connState.Status = StatusConnected
	connState.LastConnected = time.Now()
	connState.ConsecutiveFailures = 0
	connState.LastReconnectLatency = time.Since(startTime)
	connState.mu.Unlock()

	nrm.reconnectSuccess.Inc()
	nrm.reconnectLatency.Observe(time.Since(startTime).Seconds())
	nrm.lastReconnectTime.Set(float64(time.Now().Unix()))

	nrm.sendAlert(&ReconnectAlert{
		Type:         AlertTypeReconnectSuccess,
		Message:      "Connection established successfully",
		ConnectionID: connState.ID,
		Attempts:     connState.ReconnectAttempts,
		Latency:      time.Since(startTime),
		Timestamp:    time.Now(),
	})

	// Call OnConnect callback
	if connState.OnConnect != nil {
		if err := connState.OnConnect(conn); err != nil {
			log.Printf("OnConnect callback failed for %s: %v", connState.ID, err)
		}
	}

	// Start message handling and ping/pong
	go nrm.handleMessages(connState)
	go nrm.keepAlive(connState)

	return nil
}

// handleMessages processes incoming WebSocket messages
func (nrm *NetworkReconnectManager) handleMessages(connState *ConnectionState) {
	for {
		select {
		case <-connState.stopCh:
			return
		default:
			connState.mu.RLock()
			conn := connState.Conn
			connState.mu.RUnlock()

			if conn == nil {
				return
			}

			_, message, err := conn.ReadMessage()
			if err != nil {
				connState.mu.Lock()
				connState.Status = StatusDisconnected
				connState.LastDisconnected = time.Now()
				connState.Conn = nil
				connState.mu.Unlock()

				nrm.sendAlert(&ReconnectAlert{
					Type:         AlertTypeConnectionLost,
					Message:      fmt.Sprintf("Connection lost: %v", err),
					ConnectionID: connState.ID,
					Attempts:     connState.ReconnectAttempts,
					Timestamp:    time.Now(),
				})

				// Call OnDisconnect callback
				if connState.OnDisconnect != nil {
					connState.OnDisconnect(err)
				}

				// Attempt reconnection
				go nrm.attemptReconnect(connState)
				return
			}

			// Call OnMessage callback
			if connState.OnMessage != nil {
				if err := connState.OnMessage(message); err != nil {
					log.Printf("OnMessage callback failed for %s: %v", connState.ID, err)
				}
			}
		}
	}
}

// keepAlive maintains the WebSocket connection with ping/pong
func (nrm *NetworkReconnectManager) keepAlive(connState *ConnectionState) {
	ticker := time.NewTicker(nrm.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-connState.stopCh:
			return
		case <-ticker.C:
			connState.mu.RLock()
			conn := connState.Conn
			connState.mu.RUnlock()

			if conn == nil {
				return
			}

			conn.SetWriteDeadline(time.Now().Add(nrm.config.PongTimeout))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Ping failed for %s: %v", connState.ID, err)
				return
			}
		}
	}
}

// attemptReconnect attempts to reconnect with exponential backoff
func (nrm *NetworkReconnectManager) attemptReconnect(connState *ConnectionState) {
	connState.mu.Lock()
	connState.Status = StatusReconnecting
	connState.ReconnectAttempts++
	attempt := connState.ReconnectAttempts
	connState.mu.Unlock()

	nrm.reconnectAttempts.Inc()
	nrm.sendAlert(&ReconnectAlert{
		Type:         AlertTypeReconnectAttempt,
		Message:      fmt.Sprintf("Attempting reconnection %d", attempt),
		ConnectionID: connState.ID,
		Attempts:     attempt,
		Timestamp:    time.Now(),
	})

	// Calculate delay with exponential backoff and jitter
	delay := nrm.calculateDelay(attempt)

	log.Printf("Reconnecting %s in %v (attempt %d/%d)", connState.ID, delay, attempt, nrm.config.MaxRetries)

	time.Sleep(delay)

	// Check if we should stop
	select {
	case <-connState.stopCh:
		return
	default:
	}

	// Attempt connection
	ctx, cancel := context.WithTimeout(context.Background(), nrm.config.ConnectionTimeout)
	defer cancel()

	if err := nrm.connectWithRetry(ctx, connState); err != nil {
		if attempt >= nrm.config.MaxRetries {
			connState.mu.Lock()
			connState.Status = StatusFailed
			connState.mu.Unlock()

			nrm.sendAlert(&ReconnectAlert{
				Type:         AlertTypeMaxRetriesExceeded,
				Message:      fmt.Sprintf("Max retries exceeded for %s", connState.ID),
				ConnectionID: connState.ID,
				Attempts:     attempt,
				Timestamp:    time.Now(),
			})

			log.Printf("Max retries exceeded for %s", connState.ID)
			return
		}

		// Continue with next attempt
		go nrm.attemptReconnect(connState)
	}
}

// calculateDelay calculates the delay for the next reconnection attempt
func (nrm *NetworkReconnectManager) calculateDelay(attempt int) time.Duration {
	// Exponential backoff
	delay := float64(nrm.config.InitialDelay) * math.Pow(nrm.config.BackoffMultiplier, float64(attempt-1))

	// Cap at max delay
	if delay > float64(nrm.config.MaxDelay) {
		delay = float64(nrm.config.MaxDelay)
	}

	// Add jitter
	jitter := delay * nrm.config.JitterFactor * (0.5 + (float64(time.Now().UnixNano()%1000) / 1000.0))
	delay += jitter

	return time.Duration(delay)
}

// monitorConnection monitors connection health
func (nrm *NetworkReconnectManager) monitorConnection(connState *ConnectionState) {
	ticker := time.NewTicker(nrm.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-connState.stopCh:
			return
		case <-ticker.C:
			connState.mu.RLock()
			status := connState.Status
			lastConnected := connState.LastConnected
			connState.mu.RUnlock()

			// Update uptime metric
			if status == StatusConnected {
				uptime := time.Since(lastConnected).Seconds()
				nrm.connectionUptime.Set(uptime)
			}

			// Check for consecutive failures
			connState.mu.RLock()
			failures := connState.ConsecutiveFailures
			connState.mu.RUnlock()

			if failures >= nrm.config.MaxConsecutiveFailures {
				log.Printf("Connection %s has %d consecutive failures", connState.ID, failures)
			}
		}
	}
}

// sendAlert sends a reconnection alert
func (nrm *NetworkReconnectManager) sendAlert(alert *ReconnectAlert) {
	select {
	case nrm.alertCh <- alert:
		log.Printf("Reconnect alert: %s - %s", alert.Type, alert.Message)
	default:
		log.Printf("Alert channel is full, dropped reconnect alert: %s", alert.Message)
	}
}

// GetAlertChannel returns the alert channel
func (nrm *NetworkReconnectManager) GetAlertChannel() <-chan *ReconnectAlert {
	return nrm.alertCh
}

// GetConnectionStatus returns the status of a connection
func (nrm *NetworkReconnectManager) GetConnectionStatus(id string) *ConnectionState {
	nrm.mu.RLock()
	defer nrm.mu.RUnlock()

	connState, exists := nrm.connections[id]
	if !exists {
		return nil
	}

	connState.mu.RLock()
	defer connState.mu.RUnlock()

	// Create a copy to avoid race conditions
	status := &ConnectionState{
		ID:                   connState.ID,
		URL:                  connState.URL,
		Status:               connState.Status,
		LastConnected:        connState.LastConnected,
		LastDisconnected:     connState.LastDisconnected,
		ReconnectAttempts:    connState.ReconnectAttempts,
		ConsecutiveFailures:  connState.ConsecutiveFailures,
		TotalUptime:          connState.TotalUptime,
		LastReconnectLatency: connState.LastReconnectLatency,
	}

	return status
}

// GetAllConnectionStatus returns the status of all connections
func (nrm *NetworkReconnectManager) GetAllConnectionStatus() map[string]*ConnectionState {
	nrm.mu.RLock()
	defer nrm.mu.RUnlock()

	status := make(map[string]*ConnectionState)
	for id, connState := range nrm.connections {
		status[id] = nrm.GetConnectionStatus(id)
	}

	return status
}

// ForceReconnect forces a reconnection for a specific connection
func (nrm *NetworkReconnectManager) ForceReconnect(id string) error {
	nrm.mu.RLock()
	connState, exists := nrm.connections[id]
	nrm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("connection %s not found", id)
	}

	// Close existing connection
	connState.mu.Lock()
	if connState.Conn != nil {
		connState.Conn.Close()
		connState.Conn = nil
	}
	connState.Status = StatusDisconnected
	connState.mu.Unlock()

	// Attempt reconnection
	ctx, cancel := context.WithTimeout(context.Background(), nrm.config.ConnectionTimeout)
	defer cancel()

	return nrm.connectWithRetry(ctx, connState)
}

// Close closes a specific connection
func (nrm *NetworkReconnectManager) Close(id string) error {
	nrm.mu.RLock()
	connState, exists := nrm.connections[id]
	nrm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("connection %s not found", id)
	}

	// Stop monitoring
	close(connState.stopCh)

	// Close connection
	connState.mu.Lock()
	if connState.Conn != nil {
		connState.Conn.Close()
		connState.Conn = nil
	}
	connState.Status = StatusDisconnected
	connState.mu.Unlock()

	// Remove from manager
	nrm.mu.Lock()
	delete(nrm.connections, id)
	nrm.mu.Unlock()

	return nil
}

// Stop stops the network reconnect manager
func (nrm *NetworkReconnectManager) Stop() {
	close(nrm.stopCh)

	// Close all connections
	nrm.mu.RLock()
	connections := make([]*ConnectionState, 0, len(nrm.connections))
	for _, connState := range nrm.connections {
		connections = append(connections, connState)
	}
	nrm.mu.RUnlock()

	for _, connState := range connections {
		nrm.Close(connState.ID)
	}

	close(nrm.alertCh)
}

// IsHealthy checks if all connections are healthy
func (nrm *NetworkReconnectManager) IsHealthy() bool {
	nrm.mu.RLock()
	defer nrm.mu.RUnlock()

	for _, connState := range nrm.connections {
		connState.mu.RLock()
		status := connState.Status
		failures := connState.ConsecutiveFailures
		connState.mu.RUnlock()

		if status != StatusConnected || failures >= nrm.config.MaxConsecutiveFailures {
			return false
		}
	}

	return true
}
