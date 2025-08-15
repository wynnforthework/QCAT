package stability

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// GracefulShutdownManager manages graceful shutdown process
type GracefulShutdownManager struct {
	// Prometheus metrics
	shutdownDuration   prometheus.Histogram
	shutdownStatus     prometheus.Gauge
	shutdownErrors     prometheus.Counter
	shutdownComponents prometheus.Gauge

	// Configuration
	config *ShutdownConfig

	// State
	components map[string]*ShutdownComponent
	mu         sync.RWMutex

	// Channels
	shutdownCh chan os.Signal
	doneCh     chan struct{}
	stopCh     chan struct{}

	// Status
	isShuttingDown bool
	shutdownStart  time.Time
}

// ShutdownConfig represents shutdown configuration
type ShutdownConfig struct {
	// Timeouts
	ShutdownTimeout  time.Duration
	ComponentTimeout time.Duration
	SignalTimeout    time.Duration

	// Behavior
	EnableSignalHandling bool
	ForceShutdownAfter   time.Duration
	LogShutdownProgress  bool

	// Order
	ShutdownOrder []string
}

// ShutdownComponent represents a component that needs graceful shutdown
type ShutdownComponent struct {
	Name         string
	Description  string
	Priority     int
	ShutdownFunc func(ctx context.Context) error
	Timeout      time.Duration
	Status       ShutdownStatus
	Error        string
	StartTime    time.Time
	EndTime      time.Time
}

// ShutdownStatus represents shutdown status
type ShutdownStatus string

const (
	ShutdownStatusPending   ShutdownStatus = "pending"
	ShutdownStatusRunning   ShutdownStatus = "running"
	ShutdownStatusCompleted ShutdownStatus = "completed"
	ShutdownStatusFailed    ShutdownStatus = "failed"
	ShutdownStatusSkipped   ShutdownStatus = "skipped"
)

// ShutdownResult represents the result of shutdown process
type ShutdownResult struct {
	Success    bool
	Duration   time.Duration
	Components map[string]*ShutdownComponent
	Errors     []string
	StartTime  time.Time
	EndTime    time.Time
}

// NewGracefulShutdownManager creates a new graceful shutdown manager
func NewGracefulShutdownManager(config *ShutdownConfig) *GracefulShutdownManager {
	if config == nil {
		config = &ShutdownConfig{
			ShutdownTimeout:      30 * time.Second,
			ComponentTimeout:     10 * time.Second,
			SignalTimeout:        5 * time.Second,
			EnableSignalHandling: true,
			ForceShutdownAfter:   60 * time.Second,
			LogShutdownProgress:  true,
			ShutdownOrder: []string{
				"websocket_connections",
				"strategy_runners",
				"market_data_streams",
				"order_managers",
				"position_managers",
				"risk_engine",
				"optimizer",
				"health_checker",
				"network_manager",
				"memory_manager",
				"redis_cache",
				"database",
				"http_server",
			},
		}
	}

	gsm := &GracefulShutdownManager{
		config:     config,
		components: make(map[string]*ShutdownComponent),
		shutdownCh: make(chan os.Signal, 1),
		doneCh:     make(chan struct{}),
		stopCh:     make(chan struct{}),
	}

	// Initialize Prometheus metrics
	gsm.initializeMetrics()

	return gsm
}

// initializeMetrics initializes Prometheus metrics
func (gsm *GracefulShutdownManager) initializeMetrics() {
	gsm.shutdownDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "graceful_shutdown_duration_seconds",
		Help:    "Graceful shutdown duration in seconds",
		Buckets: prometheus.ExponentialBuckets(1, 2, 10),
	})

	gsm.shutdownStatus = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "graceful_shutdown_status",
		Help: "Graceful shutdown status (0=not_shutting_down, 1=shutting_down, 2=completed, 3=failed)",
	})

	gsm.shutdownErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "graceful_shutdown_errors_total",
		Help: "Total number of graceful shutdown errors",
	})

	gsm.shutdownComponents = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "graceful_shutdown_components_total",
		Help: "Total number of components registered for shutdown",
	})
}

// RegisterComponent registers a component for graceful shutdown
func (gsm *GracefulShutdownManager) RegisterComponent(name, description string, priority int, shutdownFunc func(ctx context.Context) error, timeout time.Duration) {
	gsm.mu.Lock()
	defer gsm.mu.Unlock()

	if timeout <= 0 {
		timeout = gsm.config.ComponentTimeout
	}

	gsm.components[name] = &ShutdownComponent{
		Name:         name,
		Description:  description,
		Priority:     priority,
		ShutdownFunc: shutdownFunc,
		Timeout:      timeout,
		Status:       ShutdownStatusPending,
	}

	gsm.shutdownComponents.Set(float64(len(gsm.components)))
	log.Printf("Registered shutdown component: %s (priority: %d)", name, priority)
}

// Start starts the graceful shutdown manager
func (gsm *GracefulShutdownManager) Start() {
	if gsm.config.EnableSignalHandling {
		// Setup signal handling
		signal.Notify(gsm.shutdownCh, syscall.SIGINT, syscall.SIGTERM)

		// Start signal handler
		go gsm.handleSignals()
	}

	log.Println("Graceful shutdown manager started")
}

// Stop stops the graceful shutdown manager
func (gsm *GracefulShutdownManager) Stop() {
	log.Println("Stopping graceful shutdown manager...")
	close(gsm.stopCh)
	close(gsm.shutdownCh)
}

// handleSignals handles shutdown signals
func (gsm *GracefulShutdownManager) handleSignals() {
	for {
		select {
		case <-gsm.stopCh:
			return
		case sig := <-gsm.shutdownCh:
			log.Printf("Received signal %v, initiating graceful shutdown...", sig)
			gsm.Shutdown(context.Background())
		}
	}
}

// Shutdown initiates graceful shutdown process
func (gsm *GracefulShutdownManager) Shutdown(ctx context.Context) *ShutdownResult {
	gsm.mu.Lock()
	if gsm.isShuttingDown {
		gsm.mu.Unlock()
		return &ShutdownResult{
			Success: false,
			Errors:  []string{"Shutdown already in progress"},
		}
	}

	gsm.isShuttingDown = true
	gsm.shutdownStart = time.Now()
	gsm.mu.Unlock()

	// Update metrics
	gsm.shutdownStatus.Set(1.0) // Shutting down

	log.Println("Starting graceful shutdown process...")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, gsm.config.ShutdownTimeout)
	defer cancel()

	// Start force shutdown timer
	forceTimer := time.AfterFunc(gsm.config.ForceShutdownAfter, func() {
		log.Printf("Force shutdown after %v timeout", gsm.config.ForceShutdownAfter)
		cancel()
	})
	defer forceTimer.Stop()

	// Execute shutdown
	result := gsm.executeShutdown(shutdownCtx)

	// Update metrics
	if result.Success {
		gsm.shutdownStatus.Set(2.0) // Completed
	} else {
		gsm.shutdownStatus.Set(3.0) // Failed
		gsm.shutdownErrors.Add(float64(len(result.Errors)))
	}

	gsm.shutdownDuration.Observe(result.Duration.Seconds())

	// Signal completion
	close(gsm.doneCh)

	log.Printf("Graceful shutdown completed in %v (success: %t)", result.Duration, result.Success)
	return result
}

// executeShutdown executes the shutdown process
func (gsm *GracefulShutdownManager) executeShutdown(ctx context.Context) *ShutdownResult {
	result := &ShutdownResult{
		Success:    true,
		StartTime:  gsm.shutdownStart,
		Components: make(map[string]*ShutdownComponent),
		Errors:     []string{},
	}

	// Get components in shutdown order
	components := gsm.getComponentsInOrder()

	// Shutdown components
	for _, component := range components {
		if ctx.Err() != nil {
			component.Status = ShutdownStatusSkipped
			component.Error = "Shutdown cancelled"
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", component.Name, component.Error))
			continue
		}

		// Create component context with timeout
		compCtx, cancel := context.WithTimeout(ctx, component.Timeout)

		// Execute component shutdown
		component.Status = ShutdownStatusRunning
		component.StartTime = time.Now()

		if gsm.config.LogShutdownProgress {
			log.Printf("Shutting down component: %s", component.Name)
		}

		if err := component.ShutdownFunc(compCtx); err != nil {
			component.Status = ShutdownStatusFailed
			component.Error = err.Error()
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", component.Name, err))
			result.Success = false
		} else {
			component.Status = ShutdownStatusCompleted
		}

		component.EndTime = time.Now()
		cancel()

		result.Components[component.Name] = component

		if gsm.config.LogShutdownProgress {
			log.Printf("Component %s shutdown %s", component.Name, component.Status)
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result
}

// getComponentsInOrder returns components in shutdown order
func (gsm *GracefulShutdownManager) getComponentsInOrder() []*ShutdownComponent {
	gsm.mu.RLock()
	defer gsm.mu.RUnlock()

	// Create ordered list based on config
	ordered := make([]*ShutdownComponent, 0, len(gsm.components))
	unordered := make([]*ShutdownComponent, 0, len(gsm.components))

	// First, add components in specified order
	added := make(map[string]bool)
	for _, name := range gsm.config.ShutdownOrder {
		if component, exists := gsm.components[name]; exists {
			ordered = append(ordered, component)
			added[name] = true
		}
	}

	// Then, add remaining components by priority
	for _, component := range gsm.components {
		if !added[component.Name] {
			unordered = append(unordered, component)
		}
	}

	// Sort unordered components by priority (higher priority first)
	for i := 0; i < len(unordered); i++ {
		for j := i + 1; j < len(unordered); j++ {
			if unordered[i].Priority < unordered[j].Priority {
				unordered[i], unordered[j] = unordered[j], unordered[i]
			}
		}
	}

	ordered = append(ordered, unordered...)
	return ordered
}

// WaitForShutdown waits for shutdown to complete
func (gsm *GracefulShutdownManager) WaitForShutdown() {
	<-gsm.doneCh
}

// IsShuttingDown checks if shutdown is in progress
func (gsm *GracefulShutdownManager) IsShuttingDown() bool {
	gsm.mu.RLock()
	defer gsm.mu.RUnlock()
	return gsm.isShuttingDown
}

// GetShutdownStatus returns current shutdown status
func (gsm *GracefulShutdownManager) GetShutdownStatus() map[string]interface{} {
	gsm.mu.RLock()
	defer gsm.mu.RUnlock()

	components := make(map[string]interface{})
	for name, component := range gsm.components {
		components[name] = map[string]interface{}{
			"name":        component.Name,
			"description": component.Description,
			"priority":    component.Priority,
			"status":      component.Status,
			"error":       component.Error,
			"start_time":  component.StartTime,
			"end_time":    component.EndTime,
		}
	}

	return map[string]interface{}{
		"is_shutting_down": gsm.isShuttingDown,
		"shutdown_start":   gsm.shutdownStart,
		"components":       components,
		"total_components": len(gsm.components),
	}
}

// ForceShutdown forces immediate shutdown
func (gsm *GracefulShutdownManager) ForceShutdown() {
	log.Println("Force shutdown initiated")
	gsm.Shutdown(context.Background())
}
