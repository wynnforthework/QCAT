package orchestrator

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"
)

// Orchestrator manages the entire system with process separation
type Orchestrator struct {
	processManager *ProcessManager
	msgQueue       MessageQueue
	services       map[string]*ServiceConfig
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
}

// ServiceConfig defines configuration for a service
type ServiceConfig struct {
	Name        string            `json:"name"`
	Type        ProcessType       `json:"type"`
	Executable  string            `json:"executable"`
	Args        []string          `json:"args"`
	Env         map[string]string `json:"env"`
	Port        int               `json:"port,omitempty"`
	AutoStart   bool              `json:"auto_start"`
	AutoRestart bool              `json:"auto_restart"`
	HealthCheck HealthCheckConfig `json:"health_check"`
}

// NewOrchestrator creates a new orchestrator
func NewOrchestrator() *Orchestrator {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Create message queue
	msgQueue := NewInMemoryMessageQueue(1000)
	
	// Create process manager
	processManager := NewProcessManager(msgQueue)
	
	orchestrator := &Orchestrator{
		processManager: processManager,
		msgQueue:       msgQueue,
		services:       make(map[string]*ServiceConfig),
		mu:             sync.RWMutex{},
		ctx:            ctx,
		cancel:         cancel,
	}
	
	// Setup default services
	orchestrator.setupDefaultServices()
	
	// Setup message handlers
	orchestrator.setupMessageHandlers()
	
	return orchestrator
}

// setupDefaultServices sets up default service configurations
func (o *Orchestrator) setupDefaultServices() {
	// Optimizer service
	o.services["optimizer"] = &ServiceConfig{
		Name:       "optimizer",
		Type:       ProcessTypeOptimizer,
		Executable: "./bin/optimizer",
		Args:       []string{"--port=8081", "--log-level=info"},
		Env: map[string]string{
			"QCAT_SERVICE": "optimizer",
		},
		Port:        8081,
		AutoStart:   true,
		AutoRestart: true,
		HealthCheck: HealthCheckConfig{
			Enabled:          true,
			Interval:         30 * time.Second,
			Timeout:          5 * time.Second,
			FailureThreshold: 3,
			HealthEndpoint:   "http://localhost:8081/health",
		},
	}
	
	// Market data ingestor service
	o.services["ingestor"] = &ServiceConfig{
		Name:       "ingestor",
		Type:       ProcessTypeIngestor,
		Executable: "./bin/ingestor",
		Args:       []string{"--port=8082", "--log-level=info"},
		Env: map[string]string{
			"QCAT_SERVICE": "ingestor",
		},
		Port:        8082,
		AutoStart:   true,
		AutoRestart: true,
		HealthCheck: HealthCheckConfig{
			Enabled:          true,
			Interval:         30 * time.Second,
			Timeout:          5 * time.Second,
			FailureThreshold: 3,
			HealthEndpoint:   "http://localhost:8082/health",
		},
	}
	
	// Trading service
	o.services["trader"] = &ServiceConfig{
		Name:       "trader",
		Type:       ProcessTypeTrader,
		Executable: "./bin/trader",
		Args:       []string{"--port=8083", "--log-level=info"},
		Env: map[string]string{
			"QCAT_SERVICE": "trader",
		},
		Port:        8083,
		AutoStart:   false, // Manual start for safety
		AutoRestart: true,
		HealthCheck: HealthCheckConfig{
			Enabled:          true,
			Interval:         10 * time.Second,
			Timeout:          3 * time.Second,
			FailureThreshold: 2,
			HealthEndpoint:   "http://localhost:8083/health",
		},
	}
}

// setupMessageHandlers sets up message queue handlers
func (o *Orchestrator) setupMessageHandlers() {
	// Handle optimization results
	o.msgQueue.Subscribe("optimization.result", o.handleOptimizationResult)
	
	// Handle process exit notifications
	o.msgQueue.Subscribe("process.exit", o.handleProcessExit)
	
	// Handle trade signals
	o.msgQueue.Subscribe("trade.signal", o.handleTradeSignal)
	
	// Handle market data updates
	o.msgQueue.Subscribe("market.data", o.handleMarketData)
}

// Start starts the orchestrator and all auto-start services
func (o *Orchestrator) Start() error {
	log.Println("Starting QCAT Orchestrator...")
	
	// Start auto-start services
	for name, config := range o.services {
		if config.AutoStart {
			if err := o.StartService(name); err != nil {
				log.Printf("Failed to start service %s: %v", name, err)
				// Continue starting other services
			}
		}
	}
	
	log.Println("QCAT Orchestrator started successfully")
	return nil
}

// StartService starts a specific service
func (o *Orchestrator) StartService(serviceName string) error {
	o.mu.RLock()
	config, exists := o.services[serviceName]
	o.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("service %s not found", serviceName)
	}
	
	// Check if service is already running
	processes := o.processManager.GetProcessesByType(config.Type)
	for _, process := range processes {
		if process.Config.Name == serviceName && process.Status == ProcessStatusRunning {
			return fmt.Errorf("service %s is already running", serviceName)
		}
	}
	
	// Create process config
	processConfig := ProcessConfig{
		Name:        config.Name,
		Executable:  config.Executable,
		Args:        config.Args,
		Env:         config.Env,
		WorkingDir:  ".", // Current directory
		AutoRestart: config.AutoRestart,
		MaxRetries:  3,
		HealthCheck: config.HealthCheck,
	}
	
	// Start the process
	process, err := o.processManager.StartProcess(processConfig)
	if err != nil {
		return fmt.Errorf("failed to start service %s: %w", serviceName, err)
	}
	
	log.Printf("Started service %s with process ID %s (PID: %d)", serviceName, process.ID, process.PID)
	
	// Add health check
	o.processManager.monitor.AddHealthCheck(process)
	
	return nil
}

// StopService stops a specific service
func (o *Orchestrator) StopService(serviceName string) error {
	o.mu.RLock()
	config, exists := o.services[serviceName]
	o.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("service %s not found", serviceName)
	}
	
	// Find running processes for this service
	processes := o.processManager.GetProcessesByType(config.Type)
	for _, process := range processes {
		if process.Config.Name == serviceName && process.Status == ProcessStatusRunning {
			if err := o.processManager.StopProcess(process.ID); err != nil {
				return fmt.Errorf("failed to stop service %s: %w", serviceName, err)
			}
			
			// Remove health check
			o.processManager.monitor.RemoveHealthCheck(process.ID)
			
			log.Printf("Stopped service %s", serviceName)
			return nil
		}
	}
	
	return fmt.Errorf("service %s is not running", serviceName)
}

// RestartService restarts a specific service
func (o *Orchestrator) RestartService(serviceName string) error {
	// Stop the service first
	if err := o.StopService(serviceName); err != nil {
		// If service is not running, that's okay
		log.Printf("Service %s was not running: %v", serviceName, err)
	}
	
	// Wait a moment for cleanup
	time.Sleep(2 * time.Second)
	
	// Start the service
	return o.StartService(serviceName)
}

// GetServiceStatus returns the status of all services
func (o *Orchestrator) GetServiceStatus() map[string]ServiceStatus {
	status := make(map[string]ServiceStatus)
	
	for serviceName, config := range o.services {
		serviceStatus := ServiceStatus{
			Name:   serviceName,
			Type:   string(config.Type),
			Status: "stopped",
		}
		
		// Find running processes for this service
		processes := o.processManager.GetProcessesByType(config.Type)
		for _, process := range processes {
			if process.Config.Name == serviceName {
				serviceStatus.Status = string(process.Status)
				serviceStatus.PID = process.PID
				serviceStatus.StartTime = process.StartTime
				break
			}
		}
		
		status[serviceName] = serviceStatus
	}
	
	return status
}

// RequestOptimization requests an optimization to be performed
func (o *Orchestrator) RequestOptimization(req *OptimizationRequest) error {
	// Ensure optimizer service is running
	if err := o.ensureServiceRunning("optimizer"); err != nil {
		return fmt.Errorf("optimizer service not available: %w", err)
	}
	
	// Publish optimization request
	return o.msgQueue.Publish("optimization.request", req)
}

// ensureServiceRunning ensures a service is running
func (o *Orchestrator) ensureServiceRunning(serviceName string) error {
	status := o.GetServiceStatus()
	serviceStatus, exists := status[serviceName]
	
	if !exists {
		return fmt.Errorf("service %s not configured", serviceName)
	}
	
	if serviceStatus.Status != "running" {
		return o.StartService(serviceName)
	}
	
	return nil
}

// handleOptimizationResult handles optimization results
func (o *Orchestrator) handleOptimizationResult(topic string, message []byte) error {
	// TODO: Process optimization results
	log.Printf("Received optimization result: %s", string(message))
	return nil
}

// handleProcessExit handles process exit notifications
func (o *Orchestrator) handleProcessExit(topic string, message []byte) error {
	// TODO: Handle process exits
	log.Printf("Process exited: %s", string(message))
	return nil
}

// handleTradeSignal handles trade signals
func (o *Orchestrator) handleTradeSignal(topic string, message []byte) error {
	// TODO: Forward trade signals to trading service
	log.Printf("Received trade signal: %s", string(message))
	return nil
}

// handleMarketData handles market data updates
func (o *Orchestrator) handleMarketData(topic string, message []byte) error {
	// TODO: Process market data updates
	log.Printf("Received market data: %s", string(message))
	return nil
}

// Shutdown gracefully shuts down the orchestrator
func (o *Orchestrator) Shutdown() error {
	log.Println("Shutting down QCAT Orchestrator...")
	
	o.cancel()
	
	// Stop all services
	for serviceName := range o.services {
		if err := o.StopService(serviceName); err != nil {
			log.Printf("Error stopping service %s: %v", serviceName, err)
		}
	}
	
	// Shutdown process manager
	if err := o.processManager.Shutdown(); err != nil {
		log.Printf("Error shutting down process manager: %v", err)
	}
	
	// Close message queue
	if err := o.msgQueue.Close(); err != nil {
		log.Printf("Error closing message queue: %v", err)
	}
	
	log.Println("QCAT Orchestrator shutdown complete")
	return nil
}

// ServiceStatus represents the status of a service
type ServiceStatus struct {
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	PID       int       `json:"pid,omitempty"`
	StartTime time.Time `json:"start_time,omitempty"`
}