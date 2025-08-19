package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"qcat/internal/api"
	"qcat/internal/automation"
	"qcat/internal/config"
	"qcat/internal/orchestrator"
	"qcat/internal/strategy/optimizer"
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

	// Initialize and start automation system
	log.Println("🚀 Initializing QCAT Automation System...")
	automationSystem, err := initializeAutomationSystem(cfg, server)
	if err != nil {
		log.Fatalf("Failed to initialize automation system: %v", err)
	}

	// Start automation system
	if err := automationSystem.Start(); err != nil {
		log.Fatalf("Failed to start automation system: %v", err)
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
		// Register automation system
		shutdownManager.RegisterComponent("automation_system", "Automation System", 0, func(ctx context.Context) error {
			return automationSystem.Stop()
		}, 20*time.Second)

		// Register orchestrator
		shutdownManager.RegisterComponent("orchestrator", "Process Orchestrator", 1, func(ctx context.Context) error {
			return orch.Shutdown()
		}, 15*time.Second)

		// Register HTTP server
		shutdownManager.RegisterComponent("http_server", "HTTP API Server", 2, func(ctx context.Context) error {
			return server.Stop(ctx)
		}, 10*time.Second)

		// Register database
		if server.GetDB() != nil {
			shutdownManager.RegisterComponent("database", "Database Connection", 3, func(ctx context.Context) error {
				return server.GetDB().Close()
			}, 5*time.Second)
		}

		// Register Redis
		if server.GetRedis() != nil {
			shutdownManager.RegisterComponent("redis_cache", "Redis Cache", 4, func(ctx context.Context) error {
				return server.GetRedis().Close()
			}, 5*time.Second)
		}

		// Register memory manager
		if server.GetMemoryManager() != nil {
			shutdownManager.RegisterComponent("memory_manager", "Memory Manager", 5, func(ctx context.Context) error {
				server.GetMemoryManager().Stop()
				return nil
			}, 5*time.Second)
		}

		// Register network manager
		if server.GetNetworkManager() != nil {
			shutdownManager.RegisterComponent("network_manager", "Network Reconnect Manager", 6, func(ctx context.Context) error {
				server.GetNetworkManager().Stop()
				return nil
			}, 5*time.Second)
		}

		// Register health checker
		if server.GetHealthChecker() != nil {
			shutdownManager.RegisterComponent("health_checker", "Health Checker", 7, func(ctx context.Context) error {
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

// initializeAutomationSystem 初始化自动化系统
func initializeAutomationSystem(cfg *config.Config, server *api.Server) (*automation.AutomationSystem, error) {
	// 获取必要的组件
	db := server.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}

	// 创建优化器工厂
	optimizerFactory := optimizer.NewFactory()

	// 这里需要获取exchange和accountManager
	// 由于当前架构限制，我们先创建一个简化版本
	// 在实际部署时需要从server获取这些组件

	// 创建自动化系统
	automationSystem := automation.NewAutomationSystem(
		cfg,
		db,
		nil, // exchange - 需要从server获取
		nil, // accountManager - 需要从server获取
		nil, // metrics - 需要从server获取
		optimizerFactory,
	)

	return automationSystem, nil
}
