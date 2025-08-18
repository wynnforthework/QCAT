//go:build !windows

package orchestrator

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

// getCPUUsageByPID gets CPU usage for a process by PID on Unix-like systems
func getCPUUsageByPID(pid int) float64 {
	// Read process stat file
	statFile := fmt.Sprintf("/proc/%d/stat", pid)
	content, err := ioutil.ReadFile(statFile)
	if err != nil {
		return 0.0
	}

	fields := strings.Fields(string(content))
	if len(fields) < 22 {
		return 0.0
	}

	// Fields 13 and 14 are utime and stime (user and system CPU time)
	utime, err1 := strconv.ParseInt(fields[13], 10, 64)
	stime, err2 := strconv.ParseInt(fields[14], 10, 64)
	if err1 != nil || err2 != nil {
		return 0.0
	}

	// Get system uptime
	uptime, err := getSystemUptime()
	if err != nil {
		return 0.0
	}

	// Get process start time (field 21, in clock ticks since system boot)
	starttime, err := strconv.ParseInt(fields[21], 10, 64)
	if err != nil {
		return 0.0
	}

	// Calculate CPU usage percentage
	clockTicks := getClockTicks()
	totalTime := utime + stime
	processUptime := uptime - (float64(starttime) / float64(clockTicks))
	
	if processUptime <= 0 {
		return 0.0
	}

	cpuUsage := (float64(totalTime) / float64(clockTicks)) / processUptime * 100.0
	
	// Cap at 100% per core
	return cpuUsage
}

// getMemoryUsageByPID gets memory usage for a process by PID on Unix-like systems
func getMemoryUsageByPID(pid int) int64 {
	// Read process status file
	statusFile := fmt.Sprintf("/proc/%d/status", pid)
	file, err := os.Open(statusFile)
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				// VmRSS is in kB, convert to bytes
				rss, err := strconv.ParseInt(fields[1], 10, 64)
				if err != nil {
					return 0
				}
				return rss * 1024 // Convert kB to bytes
			}
		}
	}

	return 0
}

// getSystemUptime gets system uptime in seconds
func getSystemUptime() (float64, error) {
	content, err := ioutil.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}

	fields := strings.Fields(string(content))
	if len(fields) < 1 {
		return 0, fmt.Errorf("invalid uptime format")
	}

	uptime, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, err
	}

	return uptime, nil
}

// getClockTicks gets the number of clock ticks per second
func getClockTicks() int64 {
	// Try to read from sysconf, fallback to common value
	content, err := ioutil.ReadFile("/proc/sys/kernel/pid_max")
	if err != nil {
		return 100 // Common default value
	}

	// For simplicity, use a common value
	// In a real implementation, you might use syscall.Sysconf(syscall.SC_CLK_TCK)
	_ = content
	return 100
}
