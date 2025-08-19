package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"qcat/internal/api"
	"qcat/internal/config"
	"qcat/internal/orchestrator"
)

func main() {
	log.Println("Starting QCAT - Quantitative Contract Automated Trading System")

	// Load configuration
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize orchestrator
	orch := orchestrator.NewOrchestrator()

	// Start orchestrator
	if err := orch.Start(); err != nil {
		log.Fatalf("Failed to start orchestrator: %v", err)
	}

	// Create API server
	server, err := api.NewServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Add orchestrator handler to server
	orchHandler := api.NewOrchestratorHandler(orch)
	server.RegisterOrchestratorHandler(orchHandler)

	// Start graceful shutdown manager
	shutdownManager := server.GetShutdownManager()
	if shutdownManager != nil {
		shutdownManager.Start()
	}

	// Register shutdown components
	if shutdownManager != nil {
		// Register orchestrator
		shutdownManager.RegisterComponent("orchestrator", "Process Orchestrator", 0, func(ctx context.Context) error {
			return orch.Shutdown()
		}, 15*time.Second)

		// Register HTTP server
		shutdownManager.RegisterComponent("http_server", "HTTP API Server", 1, func(ctx context.Context) error {
			return server.Stop(ctx)
		}, 10*time.Second)

		// Register database
		if server.GetDB() != nil {
			shutdownManager.RegisterComponent("database", "Database Connection", 2, func(ctx context.Context) error {
				return server.GetDB().Close()
			}, 5*time.Second)
		}

		// Register Redis
		if server.GetRedis() != nil {
			shutdownManager.RegisterComponent("redis_cache", "Redis Cache", 3, func(ctx context.Context) error {
				return server.GetRedis().Close()
			}, 5*time.Second)
		}

		// Register memory manager
		if server.GetMemoryManager() != nil {
			shutdownManager.RegisterComponent("memory_manager", "Memory Manager", 4, func(ctx context.Context) error {
				server.GetMemoryManager().Stop()
				return nil
			}, 5*time.Second)
		}

		// Register network manager
		if server.GetNetworkManager() != nil {
			shutdownManager.RegisterComponent("network_manager", "Network Reconnect Manager", 5, func(ctx context.Context) error {
				server.GetNetworkManager().Stop()
				return nil
			}, 5*time.Second)
		}

		// Register health checker
		if server.GetHealthChecker() != nil {
			shutdownManager.RegisterComponent("health_checker", "Health Checker", 6, func(ctx context.Context) error {
				server.GetHealthChecker().Stop()
				return nil
			}, 5*time.Second)
		}
	}

	// Start API server in a goroutine
	go func() {
		if err := server.Start(); err != nil {
			log.Printf("API server error: %v", err)
		}
	}()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for termination signal
	sig := <-sigChan
	log.Printf("Received signal %v, shutting down...\n", sig)

	// Use graceful shutdown manager if available
	if shutdownManager != nil {
		log.Println("Using graceful shutdown manager...")
		shutdownManager.WaitForShutdown()
	} else {
		// Fallback to manual shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Stop(ctx); err != nil {
			log.Printf("Error during server shutdown: %v", err)
		}
	}

	log.Println("Server stopped gracefully")
}
