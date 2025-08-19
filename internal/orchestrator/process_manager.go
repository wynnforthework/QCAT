package orchestrator

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"
)

// ProcessType defines the type of process
type ProcessType string

const (
	ProcessTypeOptimizer ProcessType = "optimizer"
	ProcessTypeTrader    ProcessType = "trader"
	ProcessTypeMonitor   ProcessType = "monitor"
	ProcessTypeIngestor  ProcessType = "ingestor"
)

// ProcessStatus defines the status of a process
type ProcessStatus string

const (
	ProcessStatusStarting ProcessStatus = "starting"
	ProcessStatusRunning  ProcessStatus = "running"
	ProcessStatusStopping ProcessStatus = "stopping"
	ProcessStatusStopped  ProcessStatus = "stopped"
	ProcessStatusFailed   ProcessStatus = "failed"
)

// Process represents a managed process
type Process struct {
	ID        string        `json:"id"`
	Type      ProcessType   `json:"type"`
	Status    ProcessStatus `json:"status"`
	PID       int           `json:"pid"`
	StartTime time.Time     `json:"start_time"`
	Config    ProcessConfig `json:"config"`
	cmd       *exec.Cmd
	mu        sync.RWMutex
}

// ProcessConfig holds configuration for a process
type ProcessConfig struct {
	Name        string            `json:"name"`
	Type        ProcessType       `json:"type"`
	Executable  string            `json:"executable"`
	Args        []string          `json:"args"`
	Env         map[string]string `json:"env"`
	WorkingDir  string            `json:"working_dir"`
	AutoRestart bool              `json:"auto_restart"`
	MaxRetries  int               `json:"max_retries"`
	HealthCheck HealthCheckConfig `json:"health_check"`
}

// HealthCheckConfig defines health check parameters
type HealthCheckConfig struct {
	Enabled          bool          `json:"enabled"`
	Interval         time.Duration `json:"interval"`
	Timeout          time.Duration `json:"timeout"`
	FailureThreshold int           `json:"failure_threshold"`
	HealthEndpoint   string        `json:"health_endpoint"`
}

// ProcessManager manages multiple processes
type ProcessManager struct {
	processes map[string]*Process
	msgQueue  MessageQueue
	monitor   *ProcessMonitor
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewProcessManager creates a new process manager
func NewProcessManager(msgQueue MessageQueue) *ProcessManager {
	ctx, cancel := context.WithCancel(context.Background())

	pm := &ProcessManager{
		processes: make(map[string]*Process),
		msgQueue:  msgQueue,
		mu:        sync.RWMutex{},
		ctx:       ctx,
		cancel:    cancel,
	}

	pm.monitor = NewProcessMonitor(pm)
	return pm
}

// StartProcess starts a new process with the given configuration
func (pm *ProcessManager) StartProcess(config ProcessConfig) (*Process, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Generate unique process ID
	processID := generateProcessID(config.Type, config.Name)

	// Check if process already exists
	if _, exists := pm.processes[processID]; exists {
		return nil, fmt.Errorf("process %s already exists", processID)
	}

	// Create process instance
	process := &Process{
		ID:        processID,
		Type:      ProcessType(config.Name),
		Status:    ProcessStatusStarting,
		StartTime: time.Now(),
		Config:    config,
	}

	// Prepare command
	cmd := exec.CommandContext(pm.ctx, config.Executable, config.Args...)
	cmd.Dir = config.WorkingDir

	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range config.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process %s: %w", processID, err)
	}

	process.cmd = cmd
	process.PID = cmd.Process.Pid
	process.Status = ProcessStatusRunning

	// Store process
	pm.processes[processID] = process

	// Start monitoring
	go pm.monitorProcess(process)

	return process, nil
}

// StopProcess stops a running process
func (pm *ProcessManager) StopProcess(processID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	process, exists := pm.processes[processID]
	if !exists {
		return fmt.Errorf("process %s not found", processID)
	}

	process.mu.Lock()
	defer process.mu.Unlock()

	if process.Status != ProcessStatusRunning {
		return fmt.Errorf("process %s is not running (status: %s)", processID, process.Status)
	}

	process.Status = ProcessStatusStopping

	// Try graceful shutdown first (Windows doesn't support SIGTERM)
	// On Windows, we'll use Kill() directly with a timeout
	// On Unix systems, we could use SIGTERM, but for simplicity we'll use Kill() everywhere

	// Force kill the process (works on both Windows and Unix)
	if err := process.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to kill process %s: %w", processID, err)
	}

	// Wait for process to exit
	err := process.cmd.Wait()
	if err != nil {
		process.Status = ProcessStatusFailed
	} else {
		process.Status = ProcessStatusStopped
	}

	return err
}

// RestartProcess restarts a process
func (pm *ProcessManager) RestartProcess(processID string) error {
	process, exists := pm.GetProcess(processID)
	if !exists {
		return fmt.Errorf("process %s not found", processID)
	}

	// Stop the process
	if err := pm.StopProcess(processID); err != nil {
		return fmt.Errorf("failed to stop process %s: %w", processID, err)
	}

	// Remove from processes map
	pm.mu.Lock()
	delete(pm.processes, processID)
	pm.mu.Unlock()

	// Start new process with same config
	_, err := pm.StartProcess(process.Config)
	return err
}

// GetProcess returns a process by ID
func (pm *ProcessManager) GetProcess(processID string) (*Process, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	process, exists := pm.processes[processID]
	return process, exists
}

// ListProcesses returns all managed processes
func (pm *ProcessManager) ListProcesses() map[string]*Process {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make(map[string]*Process)
	for id, process := range pm.processes {
		result[id] = process
	}
	return result
}

// GetProcessesByType returns processes of a specific type
func (pm *ProcessManager) GetProcessesByType(processType ProcessType) []*Process {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var result []*Process
	for _, process := range pm.processes {
		if process.Type == processType {
			result = append(result, process)
		}
	}
	return result
}

// monitorProcess monitors a single process
func (pm *ProcessManager) monitorProcess(process *Process) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Process monitor panic for %s: %v\n", process.ID, r)
		}
	}()

	// Wait for process to exit
	err := process.cmd.Wait()

	process.mu.Lock()
	if process.Status == ProcessStatusStopping {
		process.Status = ProcessStatusStopped
	} else {
		process.Status = ProcessStatusFailed
	}
	process.mu.Unlock()

	// Handle auto-restart
	if process.Config.AutoRestart && process.Status == ProcessStatusFailed {
		go pm.handleAutoRestart(process)
	}

	// Notify about process exit
	pm.notifyProcessExit(process, err)
}

// handleAutoRestart handles automatic process restart
func (pm *ProcessManager) handleAutoRestart(process *Process) {
	maxRetries := process.Config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3 // Default max retries
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Wait before retry (exponential backoff)
		backoff := time.Duration(attempt*attempt) * time.Second
		time.Sleep(backoff)

		// Try to restart
		if err := pm.RestartProcess(process.ID); err == nil {
			fmt.Printf("Successfully restarted process %s after %d attempts\n", process.ID, attempt)
			return
		}

		fmt.Printf("Failed to restart process %s (attempt %d/%d)\n", process.ID, attempt, maxRetries)
	}

	fmt.Printf("Giving up on restarting process %s after %d attempts\n", process.ID, maxRetries)
}

// notifyProcessExit notifies about process exit
func (pm *ProcessManager) notifyProcessExit(process *Process, err error) {
	message := ProcessExitMessage{
		ProcessID: process.ID,
		ExitTime:  time.Now(),
		Error:     err,
	}

	if pm.msgQueue != nil {
		pm.msgQueue.Publish("process.exit", message)
	}
}

// Shutdown gracefully shuts down all processes
func (pm *ProcessManager) Shutdown() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Cancel context to stop new processes
	pm.cancel()

	// Stop all running processes
	var wg sync.WaitGroup
	for processID := range pm.processes {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			if err := pm.StopProcess(id); err != nil {
				fmt.Printf("Error stopping process %s: %v\n", id, err)
			}
		}(processID)
	}

	// Wait for all processes to stop with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(60 * time.Second):
		return fmt.Errorf("timeout waiting for processes to stop")
	}
}

// generateProcessID generates a unique process ID
func generateProcessID(processType ProcessType, name string) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%s-%s-%d", processType, name, timestamp)
}

// ProcessExitMessage represents a process exit notification
type ProcessExitMessage struct {
	ProcessID string    `json:"process_id"`
	ExitTime  time.Time `json:"exit_time"`
	Error     error     `json:"error,omitempty"`
}
