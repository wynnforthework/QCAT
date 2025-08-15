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
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Initialize core services
	var db *database.DB
	var redis *cache.RedisCache
	var err error

	// Try to connect to database, but don't fail if unavailable
	db, err = database.NewConnection(&database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
		MaxOpen:  cfg.Database.MaxOpen,
		MaxIdle:  cfg.Database.MaxIdle,
		Timeout:  cfg.Database.Timeout,
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
	if s.config.App.Env == "development" {
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

		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().UTC(),
			"services": gin.H{
				"database": dbHealth,
				"redis":    redisHealth,
			},
		})
	})
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.httpServer = &http.Server{
		Addr:           fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port),
		Handler:        s.router,
		ReadTimeout:    s.config.Server.ReadTimeout,
		WriteTimeout:   s.config.Server.WriteTimeout,
		MaxHeaderBytes: s.config.Server.MaxHeaderBytes,
	}

	log.Printf("Starting API server on %s:%d", s.config.Server.Host, s.config.Server.Port)
	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the server
func (s *Server) Stop(ctx context.Context) error {
	log.Println("Shutting down server...")

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
