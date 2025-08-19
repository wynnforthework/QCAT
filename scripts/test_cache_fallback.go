package main

import (
	"context"
	"fmt"
	"time"

	"qcat/internal/cache"
)

// TestCacheEventListener implements the cache event listener interface
type TestCacheEventListener struct {
	name string
}

func (t *TestCacheEventListener) OnFallbackEnabled(from, to cache.CacheFallbackState, reason string) {
	fmt.Printf("🔄 [%s] Cache fallback enabled: %s -> %s (reason: %s)\n", t.name, from, to, reason)
}

func (t *TestCacheEventListener) OnRecoveryStarted(target cache.CacheFallbackState) {
	fmt.Printf("🔧 [%s] Cache recovery started, target: %s\n", t.name, target)
}

func (t *TestCacheEventListener) OnRecoveryCompleted(state cache.CacheFallbackState, success bool) {
	status := "SUCCESS"
	if !success {
		status = "FAILED"
	}
	fmt.Printf("✅ [%s] Cache recovery completed: %s (%s)\n", t.name, state, status)
}

func (t *TestCacheEventListener) OnHealthCheckFailed(cacheType string, err error) {
	fmt.Printf("❌ [%s] Health check failed for %s: %v\n", t.name, cacheType, err)
}

func (t *TestCacheEventListener) OnHealthCheckRecovered(cacheType string) {
	fmt.Printf("💚 [%s] Health check recovered for %s\n", t.name, cacheType)
}

func main() {
	fmt.Println("🧪 Cache Fallback Mechanism Test")
	fmt.Println("=================================")

	// Test 1: Basic fallback functionality
	fmt.Println("\n📋 Test 1: Basic Cache Fallback")
	testBasicFallback()

	// Test 2: Health monitoring
	fmt.Println("\n📋 Test 2: Health Monitoring")
	testHealthMonitoring()

	// Test 3: Recovery mechanism
	fmt.Println("\n📋 Test 3: Recovery Mechanism")
	testRecoveryMechanism()

	fmt.Println("\n🎉 Cache fallback testing completed!")
}

// testBasicFallback tests basic fallback functionality
func testBasicFallback() {
	// Create mock caches
	redisCache := cache.NewMemoryCache(100)    // Simulate Redis with memory cache
	memoryCache := cache.NewMemoryCache(100)   // Memory cache
	databaseCache := cache.NewMemoryCache(100) // Simulate database cache

	// Create fallback manager
	config := cache.DefaultCacheFallbackConfig()
	config.FailureThreshold = 2 // Lower threshold for testing

	fallbackManager := cache.NewCacheFallbackManager(redisCache, memoryCache, databaseCache, config)

	// Add event listener
	listener := &TestCacheEventListener{name: "BasicTest"}
	fallbackManager.AddListener(listener)

	ctx := context.Background()

	// Test normal operation
	fmt.Printf("Testing normal cache operations...\n")

	// Set some test data
	err := fallbackManager.Set(ctx, "test_key_1", "test_value_1", time.Minute)
	if err != nil {
		fmt.Printf("❌ Failed to set test_key_1: %v\n", err)
	} else {
		fmt.Printf("✅ Set test_key_1 successfully\n")
	}

	// Get the data back
	var value string
	err = fallbackManager.Get(ctx, "test_key_1", &value)
	if err != nil {
		fmt.Printf("❌ Failed to get test_key_1: %v\n", err)
	} else {
		fmt.Printf("✅ Got test_key_1: %s\n", value)
	}

	// Check current state
	state := fallbackManager.GetCurrentState()
	fmt.Printf("Current cache state: %s\n", state)

	// Simulate Redis failures by triggering multiple failures
	fmt.Printf("\nSimulating Redis failures...\n")
	fallbackManager.HandleCacheFailure("redis", fmt.Errorf("simulated redis connection error"))
	fallbackManager.HandleCacheFailure("redis", fmt.Errorf("simulated redis timeout"))
	fallbackManager.HandleCacheFailure("redis", fmt.Errorf("simulated redis network error"))

	// Check state after failures
	state = fallbackManager.GetCurrentState()
	fmt.Printf("Cache state after Redis failures: %s\n", state)

	// Test operations in fallback mode
	err = fallbackManager.Set(ctx, "test_key_2", "test_value_2", time.Minute)
	if err != nil {
		fmt.Printf("❌ Failed to set test_key_2 in fallback mode: %v\n", err)
	} else {
		fmt.Printf("✅ Set test_key_2 in fallback mode successfully\n")
	}
}

// testHealthMonitoring tests health monitoring functionality
func testHealthMonitoring() {
	// Create mock caches
	redisCache := cache.NewMemoryCache(100)
	memoryCache := cache.NewMemoryCache(100)
	databaseCache := cache.NewMemoryCache(100)

	// Create fallback manager with shorter health check interval
	config := cache.DefaultCacheFallbackConfig()
	config.HealthCheckInterval = 2 * time.Second
	config.FailureThreshold = 1

	fallbackManager := cache.NewCacheFallbackManager(redisCache, memoryCache, databaseCache, config)

	// Add event listener
	listener := &TestCacheEventListener{name: "HealthTest"}
	fallbackManager.AddListener(listener)

	fmt.Printf("Starting health monitoring test (will run for 10 seconds)...\n")

	// Let health monitoring run for a while
	time.Sleep(3 * time.Second)

	// Get health status
	healthStatus := fallbackManager.GetCurrentHealthStatus()
	fmt.Printf("Health Status:\n")
	for key, value := range healthStatus {
		fmt.Printf("  %s: %v\n", key, value)
	}

	// Simulate some failures
	fmt.Printf("\nSimulating cache failures...\n")
	fallbackManager.HandleCacheFailure("redis", fmt.Errorf("health check failure"))

	time.Sleep(3 * time.Second)

	// Get updated health status
	healthStatus = fallbackManager.GetCurrentHealthStatus()
	fmt.Printf("Updated Health Status:\n")
	for key, value := range healthStatus {
		fmt.Printf("  %s: %v\n", key, value)
	}
}

// testRecoveryMechanism tests the recovery mechanism
func testRecoveryMechanism() {
	// Create mock caches
	redisCache := cache.NewMemoryCache(100)
	memoryCache := cache.NewMemoryCache(100)
	databaseCache := cache.NewMemoryCache(100)

	// Create fallback manager with recovery enabled
	config := cache.DefaultCacheFallbackConfig()
	config.RecoveryCheckInterval = 3 * time.Second
	config.FailureThreshold = 1
	config.RecoveryThreshold = 1
	config.EnableAutoRecovery = true

	fallbackManager := cache.NewCacheFallbackManager(redisCache, memoryCache, databaseCache, config)

	// Add event listener
	listener := &TestCacheEventListener{name: "RecoveryTest"}
	fallbackManager.AddListener(listener)

	fmt.Printf("Testing recovery mechanism...\n")

	// Force fallback to memory
	fmt.Printf("Forcing fallback to memory cache...\n")
	fallbackManager.HandleCacheFailure("redis", fmt.Errorf("forced failure for recovery test"))

	state := fallbackManager.GetCurrentState()
	fmt.Printf("State after forced failure: %s\n", state)

	// Wait for potential recovery
	fmt.Printf("Waiting for recovery attempt (10 seconds)...\n")
	time.Sleep(10 * time.Second)

	// Check final state
	state = fallbackManager.GetCurrentState()
	fmt.Printf("Final state after recovery period: %s\n", state)

	// Get final health status
	healthStatus := fallbackManager.GetCurrentHealthStatus()
	fmt.Printf("Final Health Status:\n")
	for key, value := range healthStatus {
		fmt.Printf("  %s: %v\n", key, value)
	}
}
