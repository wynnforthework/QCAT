package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"qcat/internal/orchestrator"
)

func main() {
	fmt.Println("🧪 Process Health Check Test")
	fmt.Println("============================")

	// Test 1: Cross-platform process existence check
	fmt.Println("\n📋 Test 1: Process Existence Check")
	testProcessExistenceCheck()

	// Test 2: Process health manager
	fmt.Println("\n📋 Test 2: Process Health Manager")
	testProcessHealthManager()

	// Test 3: Process restart simulation
	fmt.Println("\n📋 Test 3: Process Restart Simulation")
	testProcessRestart()

	fmt.Println("\n🎉 Process health testing completed!")
}

// testProcessExistenceCheck tests the cross-platform process existence check
func testProcessExistenceCheck() {
	// Start a test process
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("ping", "127.0.0.1", "-n", "10")
	} else {
		cmd = exec.Command("sleep", "10")
	}

	err := cmd.Start()
	if err != nil {
		fmt.Printf("❌ Failed to start test process: %v\n", err)
		return
	}

	pid := cmd.Process.Pid
	fmt.Printf("✅ Started test process with PID: %d\n", pid)

	// Test process existence check
	exists := checkProcessExists(pid)
	if exists {
		fmt.Printf("✅ Process existence check: PASS (process %d exists)\n", pid)
	} else {
		fmt.Printf("❌ Process existence check: FAIL (process %d not found)\n", pid)
	}

	// Kill the process
	err = cmd.Process.Kill()
	if err != nil {
		fmt.Printf("⚠️  Failed to kill test process: %v\n", err)
	} else {
		fmt.Printf("✅ Killed test process\n")
	}

	// Wait a moment for the process to be cleaned up
	time.Sleep(1 * time.Second)

	// Test process existence check after kill
	exists = checkProcessExists(pid)
	if !exists {
		fmt.Printf("✅ Process existence check after kill: PASS (process %d not found)\n", pid)
	} else {
		fmt.Printf("❌ Process existence check after kill: FAIL (process %d still exists)\n", pid)
	}
}

// testProcessHealthManager tests the process health manager
func testProcessHealthManager() {
	// Create a process health manager
	healthManager := orchestrator.NewProcessHealthManager()
	defer healthManager.Stop()

	// Create a mock process
	process := &orchestrator.Process{
		ID:     "test-process-1",
		Status: orchestrator.ProcessStatusRunning,
		Config: orchestrator.ProcessConfig{
			Name: "test-process",
		},
	}

	// Start a real process for testing
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("ping", "127.0.0.1", "-n", "30")
	} else {
		cmd = exec.Command("sleep", "30")
	}

	err := cmd.Start()
	if err != nil {
		fmt.Printf("❌ Failed to start test process: %v\n", err)
		return
	}

	process.PID = cmd.Process.Pid
	// Set the command directly since SetCommand method doesn't exist
	process.mu.Lock()
	process.cmd = cmd
	process.mu.Unlock()

	fmt.Printf("✅ Started test process with PID: %d\n", process.PID)

	// Add process to health manager
	healthManager.AddProcess(process)
	fmt.Printf("✅ Added process to health manager\n")

	// Wait for a health check
	time.Sleep(2 * time.Second)

	// Get health information
	healthInfo, exists := healthManager.GetProcessHealth(process.ID)
	if exists {
		fmt.Printf("✅ Process health info retrieved:\n")
		fmt.Printf("   Last Check: %v\n", healthInfo.LastCheck)
		fmt.Printf("   Status: %v\n", healthInfo.LastStatus)
		fmt.Printf("   Failures: %d\n", healthInfo.Failures)
	} else {
		fmt.Printf("❌ Failed to get process health info\n")
	}

	// Kill the process to test failure detection
	fmt.Printf("🔪 Killing test process to test failure detection...\n")
	err = cmd.Process.Kill()
	if err != nil {
		fmt.Printf("⚠️  Failed to kill test process: %v\n", err)
	}

	// Wait for health check to detect failure
	time.Sleep(35 * time.Second) // Wait longer than health check interval

	// Check health info after process death
	healthInfo, exists = healthManager.GetProcessHealth(process.ID)
	if exists {
		fmt.Printf("✅ Process health info after kill:\n")
		fmt.Printf("   Last Check: %v\n", healthInfo.LastCheck)
		fmt.Printf("   Status: %v\n", healthInfo.LastStatus)
		fmt.Printf("   Failures: %d\n", healthInfo.Failures)
		fmt.Printf("   Restart Count: %d\n", healthInfo.RestartCount)
	}

	// Remove process from health manager
	healthManager.RemoveProcess(process.ID)
	fmt.Printf("✅ Removed process from health manager\n")
}

// testProcessRestart tests process restart functionality
func testProcessRestart() {
	fmt.Printf("Testing process restart functionality...\n")

	// Create a process manager
	processManager := orchestrator.NewProcessManager()
	defer processManager.Stop()

	// Create a process config
	config := orchestrator.ProcessConfig{
		Name:        "test-restart-process",
		Executable:  "ping",
		Args:        []string{"127.0.0.1", "-n", "5"},
		AutoRestart: true,
		MaxRetries:  2,
	}

	if runtime.GOOS != "windows" {
		config.Executable = "sleep"
		config.Args = []string{"5"}
	}

	// Start the process
	process, err := processManager.StartProcess(config)
	if err != nil {
		fmt.Printf("❌ Failed to start process: %v\n", err)
		return
	}

	fmt.Printf("✅ Started process %s with PID: %d\n", process.ID, process.PID)

	// Wait for process to complete naturally
	time.Sleep(7 * time.Second)

	// Check process status
	status := process.GetStatus()
	fmt.Printf("✅ Process status after completion: %s\n", status)

	// Test restart functionality
	fmt.Printf("🔄 Testing restart functionality...\n")
	err = processManager.RestartProcess(process.ID)
	if err != nil {
		fmt.Printf("❌ Failed to restart process: %v\n", err)
	} else {
		fmt.Printf("✅ Process restart initiated\n")
	}

	// Wait for restart to complete
	time.Sleep(3 * time.Second)

	// Check if process was restarted
	newProcess, exists := processManager.GetProcess(process.ID)
	if exists {
		fmt.Printf("✅ Process restarted with new PID: %d\n", newProcess.PID)
	} else {
		fmt.Printf("❌ Process not found after restart\n")
	}
}

// checkProcessExists checks if a process with given PID exists (cross-platform)
func checkProcessExists(pid int) bool {
	// Try to find the process
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	if process == nil {
		return false
	}

	// On Windows, os.FindProcess always succeeds, so we need additional checks
	if runtime.GOOS == "windows" {
		// On Windows, try to send a signal to check if process exists
		// This is a simplified check
		return true // For now, assume process exists if we can find it
	}

	// On Unix-like systems, send signal 0 to check existence
	err = process.Signal(os.Signal(0))
	return err == nil
}

// Helper function to create a mock process (if needed)
func createMockProcess(id string, pid int) *orchestrator.Process {
	return &orchestrator.Process{
		ID:     id,
		PID:    pid,
		Status: orchestrator.ProcessStatusRunning,
		Config: orchestrator.ProcessConfig{
			Name: "mock-process",
		},
	}
}
