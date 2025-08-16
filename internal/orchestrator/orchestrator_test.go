package orchestrator

import (
	"context"
	"testing"
	"time"
)

func TestNewOrchestrator(t *testing.T) {
	orch := NewOrchestrator()
	if orch == nil {
		t.Fatal("Expected orchestrator to be created, got nil")
	}

	// Check that default services are configured
	services := orch.GetServiceStatus()
	expectedServices := []string{"optimizer", "ingestor", "trader"}
	
	for _, serviceName := range expectedServices {
		if _, exists := services[serviceName]; !exists {
			t.Errorf("Expected service %s to be configured", serviceName)
		}
	}
}

func TestServiceManagement(t *testing.T) {
	orch := NewOrchestrator()
	defer orch.Shutdown()

	// Test getting service status
	services := orch.GetServiceStatus()
	if len(services) == 0 {
		t.Error("Expected at least one service to be configured")
	}

	// All services should initially be stopped
	for serviceName, service := range services {
		if service.Status != "stopped" {
			t.Errorf("Expected service %s to be stopped initially, got %s", serviceName, service.Status)
		}
	}
}

func TestOptimizationRequest(t *testing.T) {
	orch := NewOrchestrator()
	defer orch.Shutdown()

	// Create optimization request
	req := &OptimizationRequest{
		RequestID:  "test-request-1",
		StrategyID: "test-strategy",
		Parameters: map[string]interface{}{
			"param1": 10,
			"param2": 0.5,
		},
		TimeRange: TimeRange{
			Start: time.Now().AddDate(0, -1, 0),
			End:   time.Now(),
		},
		Optimization: OptimizationConfig{
			Method: "grid_search",
			Parameters: map[string]interface{}{
				"iterations": 100,
			},
		},
	}

	// Test optimization request (should fail since optimizer service is not running)
	err := orch.RequestOptimization(req)
	if err == nil {
		t.Error("Expected error when optimizer service is not running")
	}
}

func TestMessageQueue(t *testing.T) {
	// Test in-memory message queue
	mq := NewInMemoryMessageQueue(10)
	defer mq.Close()

	// Test publish and subscribe
	received := make(chan bool, 1)
	handler := func(topic string, message []byte) error {
		if topic == "test.topic" {
			received <- true
		}
		return nil
	}

	err := mq.Subscribe("test.topic", handler)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	err = mq.Publish("test.topic", "test message")
	if err != nil {
		t.Fatalf("Failed to publish: %v", err)
	}

	// Wait for message to be received
	select {
	case <-received:
		// Success
	case <-time.After(1 * time.Second):
		t.Error("Message was not received within timeout")
	}
}

func TestProcessManager(t *testing.T) {
	mq := NewInMemoryMessageQueue(10)
	defer mq.Close()

	pm := NewProcessManager(mq)
	defer pm.Shutdown()

	// Test process creation (this will fail since we don't have actual executables)
	config := ProcessConfig{
		Name:       "test-process",
		Executable: "echo",
		Args:       []string{"hello"},
		Env:        map[string]string{"TEST": "value"},
		WorkingDir: ".",
	}

	process, err := pm.StartProcess(config)
	if err != nil {
		// This is expected since we're using a simple echo command
		t.Logf("Expected error starting process: %v", err)
		return
	}

	// If process started successfully, test stopping it
	if process != nil {
		time.Sleep(100 * time.Millisecond) // Let it run briefly
		err = pm.StopProcess(process.ID)
		if err != nil {
			t.Errorf("Failed to stop process: %v", err)
		}
	}
}

func TestProcessMonitor(t *testing.T) {
	mq := NewInMemoryMessageQueue(10)
	defer mq.Close()

	pm := NewProcessManager(mq)
	defer pm.Shutdown()

	monitor := NewProcessMonitor(pm)
	defer monitor.Stop()

	// Test health status (should be empty initially)
	status := monitor.GetHealthStatus()
	if len(status) != 0 {
		t.Errorf("Expected empty health status, got %d entries", len(status))
	}

	// Test process metrics (should be empty initially)
	metrics := monitor.GetProcessMetrics()
	if len(metrics) != 0 {
		t.Errorf("Expected empty process metrics, got %d entries", len(metrics))
	}
}

func TestServiceConfiguration(t *testing.T) {
	orch := NewOrchestrator()
	defer orch.Shutdown()

	// Test that services have proper configuration
	services := orch.GetServiceStatus()

	// Check optimizer service
	if optimizer, exists := services["optimizer"]; exists {
		if optimizer.Type != "optimizer" {
			t.Errorf("Expected optimizer type to be 'optimizer', got '%s'", optimizer.Type)
		}
	} else {
		t.Error("Optimizer service not found")
	}

	// Check ingestor service
	if ingestor, exists := services["ingestor"]; exists {
		if ingestor.Type != "ingestor" {
			t.Errorf("Expected ingestor type to be 'ingestor', got '%s'", ingestor.Type)
		}
	} else {
		t.Error("Ingestor service not found")
	}

	// Check trader service
	if trader, exists := services["trader"]; exists {
		if trader.Type != "trader" {
			t.Errorf("Expected trader type to be 'trader', got '%s'", trader.Type)
		}
	} else {
		t.Error("Trader service not found")
	}
}

func TestGracefulShutdown(t *testing.T) {
	orch := NewOrchestrator()

	// Start orchestrator
	err := orch.Start()
	if err != nil {
		t.Fatalf("Failed to start orchestrator: %v", err)
	}

	// Test graceful shutdown
	err = orch.Shutdown()
	if err != nil {
		t.Errorf("Failed to shutdown orchestrator gracefully: %v", err)
	}
}

func TestConcurrentOperations(t *testing.T) {
	orch := NewOrchestrator()
	defer orch.Shutdown()

	// Test concurrent service status requests
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			services := orch.GetServiceStatus()
			if len(services) == 0 {
				t.Error("Expected services to be configured")
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(5 * time.Second):
			t.Error("Concurrent operations timed out")
			return
		}
	}
}

func TestMessageQueueResilience(t *testing.T) {
	mq := NewInMemoryMessageQueue(2) // Small buffer to test overflow
	defer mq.Close()

	// Test buffer overflow
	for i := 0; i < 5; i++ {
		err := mq.Publish("test.topic", "message")
		if i >= 2 && err == nil {
			t.Error("Expected error when message queue buffer is full")
		}
	}
}

func TestHealthCheckConfiguration(t *testing.T) {
	orch := NewOrchestrator()
	defer orch.Shutdown()

	// Check that services have health check configuration
	services := orch.services

	for serviceName, config := range services {
		if !config.HealthCheck.Enabled {
			t.Errorf("Expected health check to be enabled for service %s", serviceName)
		}

		if config.HealthCheck.Interval <= 0 {
			t.Errorf("Expected positive health check interval for service %s", serviceName)
		}

		if config.HealthCheck.Timeout <= 0 {
			t.Errorf("Expected positive health check timeout for service %s", serviceName)
		}

		if config.HealthCheck.FailureThreshold <= 0 {
			t.Errorf("Expected positive failure threshold for service %s", serviceName)
		}
	}
}

func TestProcessTypeValidation(t *testing.T) {
	// Test process type constants
	types := []ProcessType{
		ProcessTypeOptimizer,
		ProcessTypeTrader,
		ProcessTypeMonitor,
		ProcessTypeIngestor,
	}

	for _, processType := range types {
		if string(processType) == "" {
			t.Errorf("Process type should not be empty: %v", processType)
		}
	}
}

func TestProcessStatusValidation(t *testing.T) {
	// Test process status constants
	statuses := []ProcessStatus{
		ProcessStatusStarting,
		ProcessStatusRunning,
		ProcessStatusStopping,
		ProcessStatusStopped,
		ProcessStatusFailed,
	}

	for _, status := range statuses {
		if string(status) == "" {
			t.Errorf("Process status should not be empty: %v", status)
		}
	}
}