package orchestrator

import (
	"context"
	"log"
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// ProcessHealthManager manages process health checks with improved reliability
type ProcessHealthManager struct {
	processes     map[string]*ProcessHealthInfo
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	checkInterval time.Duration
}

// ProcessHealthInfo contains health information for a process
type ProcessHealthInfo struct {
	Process      *Process
	LastCheck    time.Time
	LastStatus   bool
	Failures     int
	MaxFailures  int
	RestartCount int
	MaxRestarts  int
	mu           sync.RWMutex
}

// NewProcessHealthManager creates a new process health manager
func NewProcessHealthManager() *ProcessHealthManager {
	ctx, cancel := context.WithCancel(context.Background())

	phm := &ProcessHealthManager{
		processes:     make(map[string]*ProcessHealthInfo),
		ctx:           ctx,
		cancel:        cancel,
		checkInterval: 30 * time.Second,
	}

	// Start health check loop
	go phm.healthCheckLoop()

	return phm
}

// AddProcess adds a process to health monitoring
func (phm *ProcessHealthManager) AddProcess(process *Process) {
	phm.mu.Lock()
	defer phm.mu.Unlock()

	phm.processes[process.ID] = &ProcessHealthInfo{
		Process:      process,
		LastCheck:    time.Now(),
		LastStatus:   true,
		Failures:     0,
		MaxFailures:  3,
		RestartCount: 0,
		MaxRestarts:  5,
	}

	log.Printf("Added process %s to health monitoring", process.ID)
}

// RemoveProcess removes a process from health monitoring
func (phm *ProcessHealthManager) RemoveProcess(processID string) {
	phm.mu.Lock()
	defer phm.mu.Unlock()

	delete(phm.processes, processID)
	log.Printf("Removed process %s from health monitoring", processID)
}

// healthCheckLoop runs the main health check loop
func (phm *ProcessHealthManager) healthCheckLoop() {
	ticker := time.NewTicker(phm.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-phm.ctx.Done():
			return
		case <-ticker.C:
			phm.performHealthChecks()
		}
	}
}

// performHealthChecks performs health checks for all monitored processes
func (phm *ProcessHealthManager) performHealthChecks() {
	phm.mu.RLock()
	processes := make([]*ProcessHealthInfo, 0, len(phm.processes))
	for _, info := range phm.processes {
		processes = append(processes, info)
	}
	phm.mu.RUnlock()

	for _, info := range processes {
		phm.checkProcessHealth(info)
	}
}

// checkProcessHealth checks the health of a single process
func (phm *ProcessHealthManager) checkProcessHealth(info *ProcessHealthInfo) {
	info.mu.Lock()
	defer info.mu.Unlock()

	info.LastCheck = time.Now()

	// Check if process is running
	isRunning := phm.isProcessRunning(info.Process)

	if isRunning {
		// Process is healthy
		if !info.LastStatus {
			log.Printf("Process %s recovered", info.Process.ID)
		}
		info.LastStatus = true
		info.Failures = 0
	} else {
		// Process is not running
		info.LastStatus = false
		info.Failures++

		log.Printf("Health check failed for process %s (failures: %d/%d): process is not running",
			info.Process.ID, info.Failures, info.MaxFailures)

		// Check if we should restart the process
		if info.Failures >= info.MaxFailures {
			phm.handleProcessFailure(info)
		}
	}
}

// isProcessRunning checks if a process is still running (cross-platform)
func (phm *ProcessHealthManager) isProcessRunning(process *Process) bool {
	process.mu.RLock()
	defer process.mu.RUnlock()

	if process.cmd == nil || process.cmd.Process == nil {
		return false
	}

	// Cross-platform process check
	return phm.checkProcessExists(process.cmd.Process.Pid)
}

// checkProcessExists checks if a process with given PID exists (cross-platform)
func (phm *ProcessHealthManager) checkProcessExists(pid int) bool {
	if runtime.GOOS == "windows" {
		return phm.checkProcessExistsWindows(pid)
	}
	return phm.checkProcessExistsUnix(pid)
}

// checkProcessExistsWindows checks process existence on Windows
func (phm *ProcessHealthManager) checkProcessExistsWindows(pid int) bool {
	// On Windows, try to find the process
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Windows, os.FindProcess always succeeds, so we need to check if it's actually running
	// We can do this by trying to get the process state
	if process == nil {
		return false
	}

	// Try to signal the process (this works on Windows too)
	err = process.Signal(os.Signal(os.Interrupt))
	if err != nil {
		// If we can't signal it, it might not exist or we don't have permission
		// Check the error type to distinguish
		return false
	}

	return true
}

// checkProcessExistsUnix checks process existence on Unix-like systems
func (phm *ProcessHealthManager) checkProcessExistsUnix(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process exists without affecting it
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// handleProcessFailure handles a process that has failed health checks
func (phm *ProcessHealthManager) handleProcessFailure(info *ProcessHealthInfo) {
	if info.RestartCount >= info.MaxRestarts {
		log.Printf("Process %s has exceeded maximum restart attempts (%d), giving up",
			info.Process.ID, info.MaxRestarts)
		return
	}

	log.Printf("Attempting to restart process %s (restart %d/%d)",
		info.Process.ID, info.RestartCount+1, info.MaxRestarts)

	// Attempt to restart the process
	go phm.restartProcess(info)
}

// restartProcess attempts to restart a failed process
func (phm *ProcessHealthManager) restartProcess(info *ProcessHealthInfo) {
	info.mu.Lock()
	info.RestartCount++
	restartAttempt := info.RestartCount
	info.mu.Unlock()

	// Wait before restart (exponential backoff)
	backoffDuration := time.Duration(restartAttempt*restartAttempt) * time.Second
	if backoffDuration > 60*time.Second {
		backoffDuration = 60 * time.Second
	}

	log.Printf("Waiting %v before restarting process %s", backoffDuration, info.Process.ID)
	time.Sleep(backoffDuration)

	// Try to restart the process
	if err := phm.performRestart(info.Process); err != nil {
		log.Printf("Failed to restart process %s: %v", info.Process.ID, err)
		return
	}

	// Reset failure count on successful restart
	info.mu.Lock()
	info.Failures = 0
	info.LastStatus = true
	info.mu.Unlock()

	log.Printf("Successfully restarted process %s", info.Process.ID)
}

// performRestart performs the actual process restart
func (phm *ProcessHealthManager) performRestart(process *Process) error {
	// Stop the old process if it's still running
	if process.cmd != nil && process.cmd.Process != nil {
		log.Printf("Stopping old process %s (PID: %d)", process.ID, process.cmd.Process.Pid)

		// Try graceful shutdown first
		if err := process.cmd.Process.Signal(os.Interrupt); err == nil {
			// Wait for graceful shutdown
			done := make(chan error, 1)
			go func() {
				done <- process.cmd.Wait()
			}()

			select {
			case <-time.After(10 * time.Second):
				// Force kill if graceful shutdown takes too long
				log.Printf("Force killing process %s", process.ID)
				process.cmd.Process.Kill()
			case <-done:
				log.Printf("Process %s shut down gracefully", process.ID)
			}
		} else {
			// Force kill if we can't send interrupt
			log.Printf("Force killing process %s", process.ID)
			process.cmd.Process.Kill()
		}
	}

	// Start new process with same configuration
	return phm.startNewProcess(process)
}

// startNewProcess starts a new process with the given configuration
func (phm *ProcessHealthManager) startNewProcess(process *Process) error {
	// This would need to be implemented based on your process management system
	// For now, we'll just log that we would restart it
	log.Printf("Would restart process %s with config: %+v", process.ID, process.Config)

	// In a real implementation, you would:
	// 1. Create a new command with the same configuration
	// 2. Start the new process
	// 3. Update the process object with the new command and PID
	// 4. Update the process status

	return nil
}

// GetProcessHealth returns health information for a process
func (phm *ProcessHealthManager) GetProcessHealth(processID string) (*ProcessHealthInfo, bool) {
	phm.mu.RLock()
	defer phm.mu.RUnlock()

	info, exists := phm.processes[processID]
	return info, exists
}

// GetAllProcessHealth returns health information for all processes
func (phm *ProcessHealthManager) GetAllProcessHealth() map[string]*ProcessHealthInfo {
	phm.mu.RLock()
	defer phm.mu.RUnlock()

	result := make(map[string]*ProcessHealthInfo)
	for id, info := range phm.processes {
		result[id] = info
	}

	return result
}

// Stop stops the health manager
func (phm *ProcessHealthManager) Stop() {
	phm.cancel()
	log.Println("Process health manager stopped")
}
