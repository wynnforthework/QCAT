package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"
)

func main() {
	fmt.Println("🧪 Process Health Check Test")
	fmt.Println("============================")

	// Test 1: Cross-platform process existence check
	fmt.Println("\n📋 Test 1: Process Existence Check")
	testProcessExistenceCheck()

	// Test 2: Process monitoring simulation
	fmt.Println("\n📋 Test 2: Process Monitoring Simulation")
	testProcessMonitoring()

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

// testProcessMonitoring simulates process monitoring and health checks
func testProcessMonitoring() {
	fmt.Printf("Simulating process monitoring...\n")

	// Start multiple test processes
	processes := make([]*exec.Cmd, 3)
	pids := make([]int, 3)

	for i := 0; i < 3; i++ {
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("ping", "127.0.0.1", "-n", "15")
		} else {
			cmd = exec.Command("sleep", "15")
		}

		err := cmd.Start()
		if err != nil {
			fmt.Printf("❌ Failed to start test process %d: %v\n", i+1, err)
			continue
		}

		processes[i] = cmd
		pids[i] = cmd.Process.Pid
		fmt.Printf("✅ Started test process %d with PID: %d\n", i+1, pids[i])
	}

	// Simulate health checks
	fmt.Printf("\n🔍 Performing health checks...\n")
	for round := 1; round <= 3; round++ {
		fmt.Printf("\nHealth check round %d:\n", round)
		
		for i, pid := range pids {
			if pid == 0 {
				continue
			}

			exists := checkProcessExists(pid)
			if exists {
				fmt.Printf("✅ Process %d (PID: %d): HEALTHY\n", i+1, pid)
			} else {
				fmt.Printf("❌ Process %d (PID: %d): NOT RUNNING\n", i+1, pid)
			}
		}

		// Kill one process in the second round to simulate failure
		if round == 2 && len(processes) > 0 && processes[1] != nil {
			fmt.Printf("\n🔪 Simulating process failure (killing process 2)...\n")
			err := processes[1].Process.Kill()
			if err != nil {
				fmt.Printf("⚠️  Failed to kill process 2: %v\n", err)
			} else {
				fmt.Printf("✅ Killed process 2 to simulate failure\n")
			}
		}

		time.Sleep(2 * time.Second)
	}

	// Clean up remaining processes
	fmt.Printf("\n🧹 Cleaning up remaining processes...\n")
	for i, cmd := range processes {
		if cmd != nil && cmd.Process != nil {
			err := cmd.Process.Kill()
			if err != nil {
				fmt.Printf("⚠️  Failed to kill process %d: %v\n", i+1, err)
			} else {
				fmt.Printf("✅ Cleaned up process %d\n", i+1)
			}
		}
	}

	// Final health check
	fmt.Printf("\n🔍 Final health check after cleanup:\n")
	for i, pid := range pids {
		if pid == 0 {
			continue
		}

		exists := checkProcessExists(pid)
		if !exists {
			fmt.Printf("✅ Process %d (PID: %d): CLEANED UP\n", i+1, pid)
		} else {
			fmt.Printf("⚠️  Process %d (PID: %d): STILL RUNNING\n", i+1, pid)
		}
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
		// This is a simplified check - in production you might want to use Windows APIs
		return checkWindowsProcess(process)
	}

	// On Unix-like systems, send signal 0 to check existence
	err = process.Signal(os.Signal(0))
	return err == nil
}

// checkWindowsProcess checks if a Windows process is running
func checkWindowsProcess(process *os.Process) bool {
	// On Windows, we can try to get the process state
	// For simplicity, we'll assume the process exists if we can find it
	// In production, you might want to use Windows-specific APIs for better accuracy
	
	// Try to send a harmless signal to test if the process is responsive
	// If the process doesn't exist, this should fail
	err := process.Signal(os.Interrupt)
	if err != nil {
		// Process might not exist or we don't have permission
		// For this test, we'll consider it as not existing
		return false
	}
	
	return true
}
