package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"qcat/internal/orchestrator"
	"qcat/internal/strategy/optimizer"
)

// OptimizerService represents the standalone optimizer service
type OptimizerService struct {
	server     *http.Server
	msgQueue   orchestrator.MessageQueue
	optimizer  *optimizer.Orchestrator
	ctx        context.Context
	cancel     context.CancelFunc
}

// Config holds the optimizer service configuration
type Config struct {
	Port         int    `json:"port"`
	MessageQueue string `json:"message_queue"`
	RedisAddr    string `json:"redis_addr,omitempty"`
	LogLevel     string `json:"log_level"`
}

func main() {
	var (
		configFile = flag.String("config", "configs/optimizer.json", "Configuration file path")
		port       = flag.Int("port", 8081, "HTTP server port")
		logLevel   = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	)
	flag.Parse()

	// Load configuration
	config, err := loadConfig(*configFile, *port, *logLevel)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create optimizer service
	service, err := NewOptimizerService(config)
	if err != nil {
		log.Fatalf("Failed to create optimizer service: %v", err)
	}

	// Start the service
	if err := service.Start(); err != nil {
		log.Fatalf("Failed to start optimizer service: %v", err)
	}

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down optimizer service...")
	if err := service.Shutdown(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}

// NewOptimizerService creates a new optimizer service
func NewOptimizerService(config *Config) (*OptimizerService, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create message queue
	var msgQueue orchestrator.MessageQueue
	switch config.MessageQueue {
	case "redis":
		msgQueue = orchestrator.NewRedisMessageQueue(config.RedisAddr)
	default:
		msgQueue = orchestrator.NewInMemoryMessageQueue(1000)
	}

	// Create optimizer
	optimizerInstance := optimizer.NewOrchestrator()

	// Create HTTP server
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Port),
		Handler: mux,
	}

	service := &OptimizerService{
		server:    server,
		msgQueue:  msgQueue,
		optimizer: optimizerInstance,
		ctx:       ctx,
		cancel:    cancel,
	}

	// Setup HTTP routes
	service.setupRoutes(mux)

	// Setup message queue handlers
	service.setupMessageHandlers()

	return service, nil
}

// Start starts the optimizer service
func (s *OptimizerService) Start() error {
	log.Printf("Starting optimizer service on port %s", s.server.Addr)

	// Start HTTP server
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	return nil
}

// Shutdown gracefully shuts down the optimizer service
func (s *OptimizerService) Shutdown() error {
	s.cancel()

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	// Close message queue
	if err := s.msgQueue.Close(); err != nil {
		return fmt.Errorf("failed to close message queue: %w", err)
	}

	return nil
}

// setupRoutes sets up HTTP routes
func (s *OptimizerService) setupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/optimize", s.optimizeHandler)
	mux.HandleFunc("/status", s.statusHandler)
	mux.HandleFunc("/metrics", s.metricsHandler)
}

// setupMessageHandlers sets up message queue handlers
func (s *OptimizerService) setupMessageHandlers() {
	// Subscribe to optimization requests
	s.msgQueue.Subscribe("optimization.request", s.handleOptimizationRequest)
}

// healthHandler handles health check requests
func (s *OptimizerService) healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"service":   "optimizer",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// optimizeHandler handles direct optimization requests
func (s *OptimizerService) optimizeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req orchestrator.OptimizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Process optimization request
	result := s.processOptimizationRequest(&req)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// statusHandler handles status requests
func (s *OptimizerService) statusHandler(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"service":   "optimizer",
		"status":    "running",
		"uptime":    time.Since(time.Now()), // This would be actual uptime
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// metricsHandler handles metrics requests
func (s *OptimizerService) metricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics := map[string]interface{}{
		"service":              "optimizer",
		"optimizations_total":  0, // Would track actual metrics
		"optimizations_active": 0,
		"memory_usage":         0,
		"cpu_usage":           0,
		"timestamp":           time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// handleOptimizationRequest handles optimization requests from message queue
func (s *OptimizerService) handleOptimizationRequest(topic string, message []byte) error {
	var req orchestrator.OptimizationRequest
	if err := json.Unmarshal(message, &req); err != nil {
		return fmt.Errorf("failed to unmarshal optimization request: %w", err)
	}

	log.Printf("Received optimization request: %s", req.RequestID)

	// Process the request
	result := s.processOptimizationRequest(&req)

	// Publish result
	return s.msgQueue.Publish("optimization.result", result)
}

// processOptimizationRequest processes an optimization request
func (s *OptimizerService) processOptimizationRequest(req *orchestrator.OptimizationRequest) *orchestrator.OptimizationResult {
	log.Printf("Processing optimization request %s for strategy %s", req.RequestID, req.StrategyID)

	result := &orchestrator.OptimizationResult{
		RequestID:  req.RequestID,
		StrategyID: req.StrategyID,
		Status:     "completed",
	}

	// TODO: Implement actual optimization logic
	// For now, return mock results
	result.BestParameters = map[string]interface{}{
		"param1": 10.5,
		"param2": 0.02,
		"param3": 100,
	}

	result.Performance = orchestrator.PerformanceMetrics{
		TotalReturn: 0.15,
		SharpeRatio: 1.2,
		MaxDrawdown: -0.08,
		WinRate:     0.65,
		TradeCount:  150,
	}

	log.Printf("Completed optimization request %s", req.RequestID)
	return result
}

// loadConfig loads configuration from file or uses defaults
func loadConfig(configFile string, port int, logLevel string) (*Config, error) {
	config := &Config{
		Port:         port,
		MessageQueue: "memory",
		LogLevel:     logLevel,
	}

	// Try to load from file
	if _, err := os.Stat(configFile); err == nil {
		file, err := os.Open(configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to open config file: %w", err)
		}
		defer file.Close()

		if err := json.NewDecoder(file).Decode(config); err != nil {
			return nil, fmt.Errorf("failed to decode config file: %w", err)
		}
	}

	return config, nil
}