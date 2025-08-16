package api

import (
	"fmt"
	"net/http"
	"time"

	"qcat/internal/orchestrator"
	"github.com/gin-gonic/gin"
)

// OrchestratorHandler handles orchestrator-related API requests
type OrchestratorHandler struct {
	orchestrator *orchestrator.Orchestrator
}

// NewOrchestratorHandler creates a new orchestrator handler
func NewOrchestratorHandler(orch *orchestrator.Orchestrator) *OrchestratorHandler {
	return &OrchestratorHandler{
		orchestrator: orch,
	}
}

// handleStatus returns the overall orchestrator status
func (h *OrchestratorHandler) handleStatus(c *gin.Context) {
	status := h.orchestrator.GetServiceStatus()
	
	c.JSON(http.StatusOK, gin.H{
		"status":    "running",
		"timestamp": time.Now(),
		"services":  status,
	})
}

// handleServices returns detailed service information
func (h *OrchestratorHandler) handleServices(c *gin.Context) {
	services := h.orchestrator.GetServiceStatus()
	c.JSON(http.StatusOK, services)
}

// handleStartService starts a specific service
func (h *OrchestratorHandler) handleStartService(c *gin.Context) {
	var req struct {
		ServiceName string `json:"service_name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	if req.ServiceName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "service_name is required",
		})
		return
	}

	if err := h.orchestrator.StartService(req.ServiceName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to start service: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Service %s started successfully", req.ServiceName),
	})
}

// handleStopService stops a specific service
func (h *OrchestratorHandler) handleStopService(c *gin.Context) {
	var req struct {
		ServiceName string `json:"service_name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	if req.ServiceName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "service_name is required",
		})
		return
	}

	if err := h.orchestrator.StopService(req.ServiceName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to stop service: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Service %s stopped successfully", req.ServiceName),
	})
}

// handleRestartService restarts a specific service
func (h *OrchestratorHandler) handleRestartService(c *gin.Context) {
	var req struct {
		ServiceName string `json:"service_name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	if req.ServiceName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "service_name is required",
		})
		return
	}

	if err := h.orchestrator.RestartService(req.ServiceName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to restart service: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Service %s restarted successfully", req.ServiceName),
	})
}

// handleOptimize handles optimization requests
func (h *OrchestratorHandler) handleOptimize(c *gin.Context) {
	var req orchestrator.OptimizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// Validate request
	if req.StrategyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "strategy_id is required",
		})
		return
	}

	// Generate request ID if not provided
	if req.RequestID == "" {
		req.RequestID = fmt.Sprintf("opt-%d", time.Now().UnixNano())
	}

	// Submit optimization request
	if err := h.orchestrator.RequestOptimization(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to submit optimization: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"request_id": req.RequestID,
		"message":    "Optimization request submitted successfully",
	})
}

// handleHealth returns health status of all services
func (h *OrchestratorHandler) handleHealth(c *gin.Context) {
	// Get service status
	services := h.orchestrator.GetServiceStatus()
	
	// Determine overall health
	overallHealth := "healthy"
	for _, service := range services {
		if service.Status != "running" {
			overallHealth = "degraded"
			break
		}
	}

	// Set appropriate HTTP status
	statusCode := http.StatusOK
	if overallHealth != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, gin.H{
		"status":     overallHealth,
		"timestamp":  time.Now(),
		"services":   services,
		"version":    "1.0.0", // Could be dynamic
	})
}