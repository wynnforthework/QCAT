package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"qcat/internal/auth"
	"qcat/internal/cache"
	"qcat/internal/config"
	"qcat/internal/database"
	"qcat/internal/monitoring"
	"qcat/internal/stability"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Server represents the API server
type Server struct {
	config     *config.Config
	router     *gin.Engine
	httpServer *http.Server
	upgrader   websocket.Upgrader
	handlers   *Handlers

	// Core services
	db         *database.DB
	redis      *cache.RedisCache
	jwtManager *auth.JWTManager
	metrics    *monitoring.Metrics
	memory     *stability.MemoryManager
	network    *stability.NetworkReconnectManager
	health     *stability.HealthChecker
	shutdown   *stability.GracefulShutdownManager
}

// Handlers contains all API handlers
type Handlers struct {
	Optimizer *OptimizerHandler
	Strategy  *StrategyHandler
	Portfolio *PortfolioHandler
	Risk      *RiskHandler
	Hotlist   *HotlistHandler
	Metrics   *MetricsHandler
	Audit     *AuditHandler
	WebSocket *WebSocketHandler
	Auth      *AuthHandler
}

// NewServer creates a new API server
func NewServer(cfg *config.Config) (*Server, error) {
	// Set Gin mode
	if cfg.App.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Initialize core services
	var db *database.DB
	var redis *cache.RedisCache
	var err error

	// Try to connect to database, but don't fail if unavailable
	db, err = database.NewConnection(&database.Config{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		DBName:          cfg.Database.DBName,
		SSLMode:         cfg.Database.SSLMode,
		MaxOpen:         cfg.Database.MaxOpen,
		MaxIdle:         cfg.Database.MaxIdle,
		Timeout:         cfg.Database.Timeout,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.Database.ConnMaxIdleTime,
	})
	if err != nil {
		log.Printf("Warning: Failed to connect to database: %v", err)
		log.Printf("Server will start without database support")
		db = nil
	}

	// Try to connect to Redis, but don't fail if unavailable
	redis, err = cache.NewRedisCache(&cache.Config{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: cfg.Redis.PoolSize,
	})
	if err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v", err)
		log.Printf("Server will start without Redis support")
		redis = nil
	}

	jwtManager := auth.NewJWTManager(cfg.JWT.SecretKey, cfg.JWT.Duration)
	metrics := monitoring.NewMetrics()

	// Initialize memory manager with configuration
	memoryConfig := &stability.MemoryConfig{
		MonitorInterval:      cfg.Memory.MonitorInterval,
		HighWaterMarkPercent: cfg.Memory.HighWaterMarkPercent,
		LowWaterMarkPercent:  cfg.Memory.LowWaterMarkPercent,
		AlertThreshold:       cfg.Memory.AlertThreshold,
		EnableAutoGC:         cfg.Memory.EnableAutoGC,
		GCInterval:           cfg.Memory.GCInterval,
		ForceGCThreshold:     cfg.Memory.ForceGCThreshold,
		MaxMemoryMB:          cfg.Memory.MaxMemoryMB,
		MaxHeapMB:            cfg.Memory.MaxHeapMB,
	}
	memory := stability.NewMemoryManager(memoryConfig)

	// Initialize network reconnect manager with configuration
	networkConfig := &stability.ReconnectConfig{
		MaxRetries:             cfg.Network.MaxRetries,
		InitialDelay:           cfg.Network.InitialDelay,
		MaxDelay:               cfg.Network.MaxDelay,
		BackoffMultiplier:      cfg.Network.BackoffMultiplier,
		JitterFactor:           cfg.Network.JitterFactor,
		HealthCheckInterval:    cfg.Network.HealthCheckInterval,
		ConnectionTimeout:      cfg.Network.ConnectionTimeout,
		PingInterval:           cfg.Network.PingInterval,
		PongTimeout:            cfg.Network.PongTimeout,
		MaxConsecutiveFailures: cfg.Network.MaxConsecutiveFailures,
		AlertThreshold:         cfg.Network.AlertThreshold,
	}
	network := stability.NewNetworkReconnectManager(networkConfig)

	// Initialize health checker with configuration
	healthConfig := &stability.HealthConfig{
		CheckInterval:      cfg.Health.CheckInterval,
		Timeout:            cfg.Health.Timeout,
		RetryCount:         cfg.Health.RetryCount,
		RetryInterval:      cfg.Health.RetryInterval,
		DegradedThreshold:  cfg.Health.DegradedThreshold,
		UnhealthyThreshold: cfg.Health.UnhealthyThreshold,
		AlertThreshold:     cfg.Health.AlertThreshold,
		AlertCooldown:      cfg.Health.AlertCooldown,
	}
	health := stability.NewHealthChecker(healthConfig)

	// Initialize graceful shutdown manager with configuration
	shutdownConfig := &stability.ShutdownConfig{
		ShutdownTimeout:      cfg.Shutdown.ShutdownTimeout,
		ComponentTimeout:     cfg.Shutdown.ComponentTimeout,
		SignalTimeout:        cfg.Shutdown.SignalTimeout,
		EnableSignalHandling: cfg.Shutdown.EnableSignalHandling,
		ForceShutdownAfter:   cfg.Shutdown.ForceShutdownAfter,
		LogShutdownProgress:  cfg.Shutdown.LogShutdownProgress,
		ShutdownOrder:        cfg.Shutdown.ShutdownOrder,
	}
	shutdown := stability.NewGracefulShutdownManager(shutdownConfig)

	server := &Server{
		config: cfg,
		router: router,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
		},
		db:         db,
		redis:      redis,
		jwtManager: jwtManager,
		metrics:    metrics,
		memory:     memory,
		network:    network,
		health:     health,
		shutdown:   shutdown,
	}

	// Set up database connection pool monitoring
	if db != nil {
		db.SetMonitorCallback(metrics.UpdateDatabasePoolMetrics)
	}

	// Initialize handlers with dependencies
	server.handlers = &Handlers{
		Optimizer: NewOptimizerHandler(db, redis, metrics),
		Strategy:  NewStrategyHandler(db, redis, metrics),
		Portfolio: NewPortfolioHandler(db, redis, metrics),
		Risk:      NewRiskHandler(db, redis, metrics),
		Hotlist:   NewHotlistHandler(db, redis, metrics),
		Metrics:   NewMetricsHandler(metrics),
		Audit:     NewAuditHandler(db, metrics),
		WebSocket: NewWebSocketHandler(server.upgrader, metrics),
		Auth:      NewAuthHandler(jwtManager, db),
	}

	// Setup routes
	server.setupRoutes()

	return server, nil
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Middleware
	s.router.Use(gin.Logger())
	s.router.Use(gin.Recovery())
	s.router.Use(corsMiddleware(s.config.CORS))
	s.router.Use(rateLimitMiddleware(s.config.RateLimit))
	s.router.Use(s.metrics.MetricsMiddleware())

	// Swagger documentation
	if s.config.App.Environment == "development" {
		s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// Prometheus metrics
	if s.config.Monitoring.PrometheusEnabled {
		s.router.GET(s.config.Monitoring.PrometheusPath, gin.WrapH(monitoring.PrometheusHandler()))
	}

	// API v1 group
	v1 := s.router.Group("/api/v1")
	{
		// Public routes (no authentication required)
		auth := v1.Group("/auth")
		{
			auth.POST("/login", s.handlers.Auth.Login)
			auth.POST("/register", s.handlers.Auth.Register)
			auth.POST("/refresh", s.handlers.Auth.RefreshToken)
		}

		// Protected routes (authentication required)
		protected := v1.Group("")
		protected.Use(s.jwtManager.AuthMiddleware())
		{
			// Optimizer routes
			optimizer := protected.Group("/optimizer")
			{
				optimizer.POST("/run", s.handlers.Optimizer.RunOptimization)
				optimizer.GET("/tasks", s.handlers.Optimizer.GetTasks)
				optimizer.GET("/tasks/:id", s.handlers.Optimizer.GetTask)
				optimizer.GET("/results/:id", s.handlers.Optimizer.GetResults)
			}

			// Strategy routes
			strategy := protected.Group("/strategy")
			{
				strategy.GET("/", s.handlers.Strategy.ListStrategies)
				strategy.GET("/:id", s.handlers.Strategy.GetStrategy)
				strategy.POST("/", s.handlers.Strategy.CreateStrategy)
				strategy.PUT("/:id", s.handlers.Strategy.UpdateStrategy)
				strategy.DELETE("/:id", s.handlers.Strategy.DeleteStrategy)
				strategy.POST("/:id/promote", s.handlers.Strategy.PromoteStrategy)
				strategy.POST("/:id/start", s.handlers.Strategy.StartStrategy)
				strategy.POST("/:id/stop", s.handlers.Strategy.StopStrategy)
				strategy.POST("/:id/backtest", s.handlers.Strategy.RunBacktest)
			}

			// Portfolio routes
			portfolio := protected.Group("/portfolio")
			{
				portfolio.GET("/overview", s.handlers.Portfolio.GetOverview)
				portfolio.GET("/allocations", s.handlers.Portfolio.GetAllocations)
				portfolio.POST("/rebalance", s.handlers.Portfolio.Rebalance)
				portfolio.GET("/history", s.handlers.Portfolio.GetHistory)
			}

			// Risk routes
			risk := protected.Group("/risk")
			{
				risk.GET("/overview", s.handlers.Risk.GetOverview)
				risk.GET("/limits", s.handlers.Risk.GetLimits)
				risk.POST("/limits", s.handlers.Risk.SetLimits)
				risk.GET("/circuit-breakers", s.handlers.Risk.GetCircuitBreakers)
				risk.POST("/circuit-breakers", s.handlers.Risk.SetCircuitBreakers)
				risk.GET("/violations", s.handlers.Risk.GetViolations)
			}

			// Hotlist routes
			hotlist := protected.Group("/hotlist")
			{
				hotlist.GET("/symbols", s.handlers.Hotlist.GetHotSymbols)
				hotlist.POST("/approve", s.handlers.Hotlist.ApproveSymbol)
				hotlist.GET("/whitelist", s.handlers.Hotlist.GetWhitelist)
				hotlist.POST("/whitelist", s.handlers.Hotlist.AddToWhitelist)
				hotlist.DELETE("/whitelist/:symbol", s.handlers.Hotlist.RemoveFromWhitelist)
			}

			// Metrics routes
			metrics := protected.Group("/metrics")
			{
				metrics.GET("/strategy/:id", s.handlers.Metrics.GetStrategyMetrics)
				metrics.GET("/system", s.handlers.Metrics.GetSystemMetrics)
				metrics.GET("/performance", s.handlers.Metrics.GetPerformanceMetrics)
			}

			// Memory management routes
			memory := protected.Group("/memory")
			{
				memory.GET("/stats", s.getMemoryStats)
				memory.POST("/gc", s.forceGC)
			}

			// Network management routes
			network := protected.Group("/network")
			{
				network.GET("/connections", s.getNetworkConnections)
				network.GET("/connections/:id", s.getNetworkConnection)
				network.POST("/connections/:id/reconnect", s.forceNetworkReconnect)
			}

			// Health management routes
			health := protected.Group("/health")
			{
				health.GET("/status", s.getHealthStatus)
				health.GET("/checks", s.getAllHealthChecks)
				health.GET("/checks/:name", s.getHealthCheck)
				health.POST("/checks/:name/force", s.forceHealthCheck)
			}

			// Shutdown management routes
			shutdown := protected.Group("/shutdown")
			{
				shutdown.GET("/status", s.getShutdownStatus)
				shutdown.POST("/graceful", s.initiateGracefulShutdown)
				shutdown.POST("/force", s.forceShutdown)
			}

			// Audit routes
			audit := protected.Group("/audit")
			{
				audit.GET("/logs", s.handlers.Audit.GetLogs)
				audit.GET("/decisions", s.handlers.Audit.GetDecisionChains)
				audit.GET("/performance", s.handlers.Audit.GetPerformanceMetrics)
				audit.POST("/export", s.handlers.Audit.ExportReport)
			}
		}
	}

	// WebSocket routes (authentication handled in WebSocket handler)
	ws := s.router.Group("/ws")
	{
		ws.GET("/market/:symbol", s.handlers.WebSocket.MarketStream)
		ws.GET("/strategy/:id", s.handlers.WebSocket.StrategyStream)
		ws.GET("/alerts", s.handlers.WebSocket.AlertsStream)
	}

	// Health check
	s.router.GET("/health", func(c *gin.Context) {
		// Check database health
		dbHealth := "ok"
		if s.db != nil {
			if err := s.db.HealthCheck(c.Request.Context()); err != nil {
				dbHealth = "error"
			}
		} else {
			dbHealth = "unavailable"
		}

		// Check Redis health
		redisHealth := "ok"
		if s.redis != nil {
			if err := s.redis.HealthCheck(c.Request.Context()); err != nil {
				redisHealth = "error"
			}
		} else {
			redisHealth = "unavailable"
		}

		// Check memory health
		memoryHealth := "ok"
		if s.memory != nil && !s.memory.IsHealthy() {
			memoryHealth = "warning"
		}

		// Check network health
		networkHealth := "ok"
		if s.network != nil && !s.network.IsHealthy() {
			networkHealth = "warning"
		}

		// Check health checker status
		healthCheckerStatus := "ok"
		if s.health != nil && !s.health.IsHealthy() {
			healthCheckerStatus = "warning"
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().UTC(),
			"services": gin.H{
				"database": dbHealth,
				"redis":    redisHealth,
				"memory":   memoryHealth,
				"network":  networkHealth,
				"health":   healthCheckerStatus,
			},
		})
	})
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.httpServer = &http.Server{
		Addr:           fmt.Sprintf(":%d", s.config.Server.Port),
		Handler:        s.router,
		ReadTimeout:    s.config.Server.ReadTimeout,
		WriteTimeout:   s.config.Server.WriteTimeout,
		MaxHeaderBytes: s.config.Server.MaxHeaderBytes,
	}

	log.Printf("Starting API server on port %d", s.config.Server.Port)
	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the server
func (s *Server) Stop(ctx context.Context) error {
	log.Println("Shutting down server...")

	// Stop memory manager
	if s.memory != nil {
		s.memory.Stop()
	}

	// Stop network reconnect manager
	if s.network != nil {
		s.network.Stop()
	}

	// Stop health checker
	if s.health != nil {
		s.health.Stop()
	}

	// Stop graceful shutdown manager
	if s.shutdown != nil {
		s.shutdown.Stop()
	}

	// Close database connection
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}

	// Close Redis connection
	if s.redis != nil {
		if err := s.redis.Close(); err != nil {
			log.Printf("Error closing Redis: %v", err)
		}
	}

	// Shutdown HTTP server
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Println("Server stopped gracefully")
	return nil
}

// corsMiddleware adds CORS headers
func corsMiddleware(corsConfig config.CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Allow all origins for now, in production you should check against allowed origins
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// rateLimitMiddleware adds rate limiting
func rateLimitMiddleware(rateLimitConfig config.RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement rate limiting
		c.Next()
	}
}

// getMemoryStats returns current memory statistics
func (s *Server) getMemoryStats(c *gin.Context) {
	if s.memory == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Memory manager not available",
		})
		return
	}

	stats := s.memory.GetMemoryStats()
	c.JSON(http.StatusOK, gin.H{
		"memory": stats,
	})
}

// forceGC forces a garbage collection
func (s *Server) forceGC(c *gin.Context) {
	if s.memory == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Memory manager not available",
		})
		return
	}

	s.memory.ForceGC()
	c.JSON(http.StatusOK, gin.H{
		"message": "Garbage collection completed",
	})
}

// getNetworkConnections returns all network connection status
func (s *Server) getNetworkConnections(c *gin.Context) {
	if s.network == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Network manager not available",
		})
		return
	}

	connections := s.network.GetAllConnectionStatus()
	c.JSON(http.StatusOK, gin.H{
		"connections": connections,
	})
}

// getNetworkConnection returns status of a specific network connection
func (s *Server) getNetworkConnection(c *gin.Context) {
	if s.network == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Network manager not available",
		})
		return
	}

	connectionID := c.Param("id")
	connection := s.network.GetConnectionStatus(connectionID)
	if connection == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Connection not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"connection": connection,
	})
}

// forceNetworkReconnect forces reconnection of a specific network connection
func (s *Server) forceNetworkReconnect(c *gin.Context) {
	if s.network == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Network manager not available",
		})
		return
	}

	connectionID := c.Param("id")
	if err := s.network.ForceReconnect(connectionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Reconnection initiated",
	})
}

// getHealthStatus returns overall health status
func (s *Server) getHealthStatus(c *gin.Context) {
	if s.health == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Health checker not available",
		})
		return
	}

	status := s.health.GetOverallHealth()
	c.JSON(http.StatusOK, gin.H{
		"health": status,
	})
}

// getAllHealthChecks returns all health check statuses
func (s *Server) getAllHealthChecks(c *gin.Context) {
	if s.health == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Health checker not available",
		})
		return
	}

	checks := s.health.GetAllHealthStatus()
	c.JSON(http.StatusOK, gin.H{
		"checks": checks,
	})
}

// getHealthCheck returns status of a specific health check
func (s *Server) getHealthCheck(c *gin.Context) {
	if s.health == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Health checker not available",
		})
		return
	}

	checkName := c.Param("name")
	check := s.health.GetHealthStatus(checkName)
	if check == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Health check not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"check": check,
	})
}

// forceHealthCheck forces a health check for a specific service
func (s *Server) forceHealthCheck(c *gin.Context) {
	if s.health == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Health checker not available",
		})
		return
	}

	checkName := c.Param("name")
	if err := s.health.ForceCheck(checkName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Health check initiated",
	})
}

// getShutdownStatus returns current shutdown status
func (s *Server) getShutdownStatus(c *gin.Context) {
	if s.shutdown == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Shutdown manager not available",
		})
		return
	}

	status := s.shutdown.GetShutdownStatus()
	c.JSON(http.StatusOK, gin.H{
		"shutdown": status,
	})
}

// initiateGracefulShutdown initiates graceful shutdown process
func (s *Server) initiateGracefulShutdown(c *gin.Context) {
	if s.shutdown == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Shutdown manager not available",
		})
		return
	}

	result := s.shutdown.Shutdown(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{
		"message": "Graceful shutdown initiated",
		"result":  result,
	})
}

// forceShutdown forces immediate shutdown
func (s *Server) forceShutdown(c *gin.Context) {
	if s.shutdown == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Shutdown manager not available",
		})
		return
	}

	s.shutdown.ForceShutdown()
	c.JSON(http.StatusOK, gin.H{
		"message": "Force shutdown initiated",
	})
}

// GetShutdownManager returns the shutdown manager
func (s *Server) GetShutdownManager() *stability.GracefulShutdownManager {
	return s.shutdown
}

// GetDB returns the database connection
func (s *Server) GetDB() *database.DB {
	return s.db
}

// GetRedis returns the Redis cache
func (s *Server) GetRedis() *cache.RedisCache {
	return s.redis
}

// GetMemoryManager returns the memory manager
func (s *Server) GetMemoryManager() *stability.MemoryManager {
	return s.memory
}

// GetNetworkManager returns the network manager
func (s *Server) GetNetworkManager() *stability.NetworkReconnectManager {
	return s.network
}

// GetHealthChecker returns the health checker
func (s *Server) GetHealthChecker() *stability.HealthChecker {
	return s.health
}
