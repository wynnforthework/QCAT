package orchestrator

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"syscall"
	"time"
)

// ProcessMonitor monitors the health of managed processes
type ProcessMonitor struct {
	manager     *ProcessManager
	healthChecks map[string]*HealthChecker
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// HealthChecker performs health checks for a specific process
type HealthChecker struct {
	process       *Process
	config        HealthCheckConfig
	failures      int
	lastCheck     time.Time
	lastStatus    bool
	httpClient    *http.Client
	mu            sync.RWMutex
}

// NewProcessMonitor creates a new process monitor
func NewProcessMonitor(manager *ProcessManager) *ProcessMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	
	monitor := &ProcessMonitor{
		manager:      manager,
		healthChecks: make(map[string]*HealthChecker),
		mu:           sync.RWMutex{},
		ctx:          ctx,
		cancel:       cancel,
	}
	
	// Start monitoring loop
	monitor.wg.Add(1)
	go monitor.monitoringLoop()
	
	return monitor
}

// AddHealthCheck adds a health check for a process
func (pm *ProcessMonitor) AddHealthCheck(process *Process) {
	if !process.Config.HealthCheck.Enabled {
		return
	}
	
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	checker := &HealthChecker{
		process: process,
		config:  process.Config.HealthCheck,
		httpClient: &http.Client{
			Timeout: process.Config.HealthCheck.Timeout,
		},
	}
	
	pm.healthChecks[process.ID] = checker
}

// RemoveHealthCheck removes a health check for a process
func (pm *ProcessMonitor) RemoveHealthCheck(processID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	delete(pm.healthChecks, processID)
}

// monitoringLoop runs the main monitoring loop
func (pm *ProcessMonitor) monitoringLoop() {
	defer pm.wg.Done()
	
	ticker := time.NewTicker(5 * time.Second) // Check every 5 seconds
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			pm.performHealthChecks()
		case <-pm.ctx.Done():
			return
		}
	}
}

// performHealthChecks performs health checks for all monitored processes
func (pm *ProcessMonitor) performHealthChecks() {
	pm.mu.RLock()
	checkers := make([]*HealthChecker, 0, len(pm.healthChecks))
	for _, checker := range pm.healthChecks {
		checkers = append(checkers, checker)
	}
	pm.mu.RUnlock()
	
	// Perform health checks concurrently
	var wg sync.WaitGroup
	for _, checker := range checkers {
		wg.Add(1)
		go func(c *HealthChecker) {
			defer wg.Done()
			pm.performHealthCheck(c)
		}(checker)
	}
	wg.Wait()
}

// performHealthCheck performs a health check for a single process
func (pm *ProcessMonitor) performHealthCheck(checker *HealthChecker) {
	checker.mu.Lock()
	defer checker.mu.Unlock()
	
	// Skip if not enough time has passed since last check
	if time.Since(checker.lastCheck) < checker.config.Interval {
		return
	}
	
	checker.lastCheck = time.Now()
	
	// Check if process is still running
	if !pm.isProcessRunning(checker.process) {
		checker.lastStatus = false
		checker.failures++
		pm.handleHealthCheckFailure(checker, fmt.Errorf("process is not running"))
		return
	}
	
	// Perform HTTP health check if configured
	if checker.config.HealthEndpoint != "" {
		if err := pm.performHTTPHealthCheck(checker); err != nil {
			checker.lastStatus = false
			checker.failures++
			pm.handleHealthCheckFailure(checker, err)
			return
		}
	}
	
	// Health check passed
	checker.lastStatus = true
	checker.failures = 0
}

// isProcessRunning checks if a process is still running
func (pm *ProcessMonitor) isProcessRunning(process *Process) bool {
	process.mu.RLock()
	defer process.mu.RUnlock()
	
	if process.cmd == nil || process.cmd.Process == nil {
		return false
	}
	
	// Check if process is still alive
	err := process.cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

// performHTTPHealthCheck performs an HTTP health check
func (pm *ProcessMonitor) performHTTPHealthCheck(checker *HealthChecker) error {
	ctx, cancel := context.WithTimeout(pm.ctx, checker.config.Timeout)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", checker.config.HealthEndpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}
	
	resp, err := checker.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}
	
	return nil
}

// handleHealthCheckFailure handles a health check failure
func (pm *ProcessMonitor) handleHealthCheckFailure(checker *HealthChecker, err error) {
	fmt.Printf("Health check failed for process %s: %v (failures: %d/%d)\n",
		checker.process.ID, err, checker.failures, checker.config.FailureThreshold)
	
	// Check if failure threshold is reached
	if checker.failures >= checker.config.FailureThreshold {
		fmt.Printf("Process %s exceeded failure threshold, attempting restart\n", checker.process.ID)
		
		// Attempt to restart the process
		go func() {
			if err := pm.manager.RestartProcess(checker.process.ID); err != nil {
				fmt.Printf("Failed to restart process %s: %v\n", checker.process.ID, err)
			} else {
				fmt.Printf("Successfully restarted process %s\n", checker.process.ID)
				// Reset failure count after successful restart
				checker.mu.Lock()
				checker.failures = 0
				checker.mu.Unlock()
			}
		}()
	}
}

// GetHealthStatus returns the health status of all monitored processes
func (pm *ProcessMonitor) GetHealthStatus() map[string]HealthStatus {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	status := make(map[string]HealthStatus)
	for processID, checker := range pm.healthChecks {
		checker.mu.RLock()
		status[processID] = HealthStatus{
			ProcessID:   processID,
			Healthy:     checker.lastStatus,
			Failures:    checker.failures,
			LastCheck:   checker.lastCheck,
			LastError:   "", // Could store last error if needed
		}
		checker.mu.RUnlock()
	}
	
	return status
}

// Stop stops the process monitor
func (pm *ProcessMonitor) Stop() {
	pm.cancel()
	pm.wg.Wait()
}

// HealthStatus represents the health status of a process
type HealthStatus struct {
	ProcessID string    `json:"process_id"`
	Healthy   bool      `json:"healthy"`
	Failures  int       `json:"failures"`
	LastCheck time.Time `json:"last_check"`
	LastError string    `json:"last_error,omitempty"`
}

// ProcessMetrics represents metrics for a process
type ProcessMetrics struct {
	ProcessID   string        `json:"process_id"`
	CPUUsage    float64       `json:"cpu_usage"`
	MemoryUsage int64         `json:"memory_usage"`
	Uptime      time.Duration `json:"uptime"`
	Status      ProcessStatus `json:"status"`
}

// GetProcessMetrics returns metrics for all processes
func (pm *ProcessMonitor) GetProcessMetrics() map[string]ProcessMetrics {
	processes := pm.manager.ListProcesses()
	metrics := make(map[string]ProcessMetrics)
	
	for processID, process := range processes {
		process.mu.RLock()
		uptime := time.Since(process.StartTime)
		status := process.Status
		process.mu.RUnlock()
		
		metrics[processID] = ProcessMetrics{
			ProcessID:   processID,
			CPUUsage:    pm.getCPUUsage(process),
			MemoryUsage: pm.getMemoryUsage(process),
			Uptime:      uptime,
			Status:      status,
		}
	}
	
	return metrics
}

// getCPUUsage gets CPU usage for a process
func (pm *ProcessMonitor) getCPUUsage(process *Process) float64 {
	// Get system information for the process
	if process.Handle == nil {
		return 0.0
	}
	
	pid := process.Handle.Pid
	return pm.getCPUUsageByPID(pid)
}

// getMemoryUsage gets memory usage for a process in bytes
func (pm *ProcessMonitor) getMemoryUsage(process *Process) int64 {
	// Get system information for the process
	if process.Handle == nil {
		return 0
	}
	
	pid := process.Handle.Pid
	return pm.getMemoryUsageByPID(pid)
}

// getCPUUsageByPID gets CPU usage for a process by PID
func (pm *ProcessMonitor) getCPUUsageByPID(pid int) float64 {
	return getCPUUsageByPID(pid)
}

// getMemoryUsageByPID gets memory usage for a process by PID
func (pm *ProcessMonitor) getMemoryUsageByPID(pid int) int64 {
	return getMemoryUsageByPID(pid)
}