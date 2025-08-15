package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"qcat/internal/config"
)

// Server represents the API server
type Server struct {
	config     *config.Config
	router     *gin.Engine
	httpServer *http.Server
	upgrader   websocket.Upgrader
	handlers   *Handlers
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
}

// NewServer creates a new API server
func NewServer(cfg *config.Config) *Server {
	// Set Gin mode
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	
	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())
	router.Use(rateLimitMiddleware())

	server := &Server{
		config: cfg,
		router: router,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
		},
	}

	// Initialize handlers
	server.handlers = &Handlers{
		Optimizer: NewOptimizerHandler(),
		Strategy:  NewStrategyHandler(),
		Portfolio: NewPortfolioHandler(),
		Risk:      NewRiskHandler(),
		Hotlist:   NewHotlistHandler(),
		Metrics:   NewMetricsHandler(),
		Audit:     NewAuditHandler(),
		WebSocket: NewWebSocketHandler(server.upgrader),
	}

	// Setup routes
	server.setupRoutes()

	return server
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// API v1 group
	v1 := s.router.Group("/api/v1")
	{
		// Optimizer routes
		optimizer := v1.Group("/optimizer")
		{
			optimizer.POST("/run", s.handlers.Optimizer.RunOptimization)
			optimizer.GET("/tasks", s.handlers.Optimizer.GetTasks)
			optimizer.GET("/tasks/:id", s.handlers.Optimizer.GetTask)
			optimizer.GET("/results/:id", s.handlers.Optimizer.GetResults)
		}

		// Strategy routes
		strategy := v1.Group("/strategy")
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
		portfolio := v1.Group("/portfolio")
		{
			portfolio.GET("/overview", s.handlers.Portfolio.GetOverview)
			portfolio.GET("/allocations", s.handlers.Portfolio.GetAllocations)
			portfolio.POST("/rebalance", s.handlers.Portfolio.Rebalance)
			portfolio.GET("/history", s.handlers.Portfolio.GetHistory)
		}

		// Risk routes
		risk := v1.Group("/risk")
		{
			risk.GET("/overview", s.handlers.Risk.GetOverview)
			risk.GET("/limits", s.handlers.Risk.GetLimits)
			risk.POST("/limits", s.handlers.Risk.SetLimits)
			risk.GET("/circuit-breakers", s.handlers.Risk.GetCircuitBreakers)
			risk.POST("/circuit-breakers", s.handlers.Risk.SetCircuitBreakers)
			risk.GET("/violations", s.handlers.Risk.GetViolations)
		}

		// Hotlist routes
		hotlist := v1.Group("/hotlist")
		{
			hotlist.GET("/symbols", s.handlers.Hotlist.GetHotSymbols)
			hotlist.POST("/approve", s.handlers.Hotlist.ApproveSymbol)
			hotlist.GET("/whitelist", s.handlers.Hotlist.GetWhitelist)
			hotlist.POST("/whitelist", s.handlers.Hotlist.AddToWhitelist)
			hotlist.DELETE("/whitelist/:symbol", s.handlers.Hotlist.RemoveFromWhitelist)
		}

		// Metrics routes
		metrics := v1.Group("/metrics")
		{
			metrics.GET("/strategy/:id", s.handlers.Metrics.GetStrategyMetrics)
			metrics.GET("/system", s.handlers.Metrics.GetSystemMetrics)
			metrics.GET("/performance", s.handlers.Metrics.GetPerformanceMetrics)
		}

		// Audit routes
		audit := v1.Group("/audit")
		{
			audit.GET("/logs", s.handlers.Audit.GetLogs)
			audit.GET("/decisions", s.handlers.Audit.GetDecisionChains)
			audit.GET("/performance", s.handlers.Audit.GetPerformanceMetrics)
			audit.POST("/export", s.handlers.Audit.ExportReport)
		}
	}

	// WebSocket routes
	ws := s.router.Group("/ws")
	{
		ws.GET("/market/:symbol", s.handlers.WebSocket.MarketStream)
		ws.GET("/strategy/:id", s.handlers.WebSocket.StrategyStream)
		ws.GET("/alerts", s.handlers.WebSocket.AlertsStream)
	}

	// Health check
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().UTC(),
		})
	})
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	log.Printf("Starting API server on %s", addr)
	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the server
func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// corsMiddleware adds CORS headers
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		c.Next()
	}
}

// rateLimitMiddleware adds rate limiting
func rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement rate limiting
		c.Next()
	}
}
