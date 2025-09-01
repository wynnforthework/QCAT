package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"qcat/internal/orchestrator"
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
// @Summary Get orchestrator status
// @Description Get overall status of the orchestrator and all managed services
// @Tags Orchestrator
// @Accept json
// @Produce json
// @Success 200 {object} object{status=string,timestamp=string,services=object}
// @Failure 500 {object} object{error=string}
// @Router /orchestrator/status [get]
func (h *OrchestratorHandler) handleStatus(c *gin.Context) {
	status := h.orchestrator.GetServiceStatus()

	c.JSON(http.StatusOK, gin.H{
		"status":    "running",
		"timestamp": time.Now(),
		"services":  status,
	})
}

// handleServices returns detailed service information
// @Summary Get services information
// @Description Get detailed information about all managed services
// @Tags Orchestrator
// @Accept json
// @Produce json
// @Success 200 {object} object
// @Failure 500 {object} object{error=string}
// @Router /orchestrator/services [get]
func (h *OrchestratorHandler) handleServices(c *gin.Context) {
	services := h.orchestrator.GetServiceStatus()
	c.JSON(http.StatusOK, services)
}

// handleStartService starts a specific service
// @Summary Start service
// @Description Start a specific managed service
// @Tags Orchestrator
// @Accept json
// @Produce json
// @Param request body object{service_name=string} true "Service to start"
// @Success 200 {object} object{status=string,message=string}
// @Failure 400 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /orchestrator/services/start [post]
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
// @Summary Stop service
// @Description Stop a specific managed service
// @Tags Orchestrator
// @Accept json
// @Produce json
// @Param request body object{service_name=string} true "Service to stop"
// @Success 200 {object} object{status=string,message=string}
// @Failure 400 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /orchestrator/services/stop [post]
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
// @Summary Restart service
// @Description Restart a specific managed service
// @Tags Orchestrator
// @Accept json
// @Produce json
// @Param request body object{service_name=string} true "Service to restart"
// @Success 200 {object} object{status=string,message=string}
// @Failure 400 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /orchestrator/services/restart [post]
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
// @Summary Submit optimization request
// @Description Submit an optimization request to the orchestrator
// @Tags Orchestrator
// @Accept json
// @Produce json
// @Param request body orchestrator.OptimizationRequest true "Optimization parameters"
// @Success 200 {object} object{status=string,task_id=string,message=string}
// @Failure 400 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /orchestrator/optimize [post]
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
// @Summary Get health status
// @Description Get health status of all managed services
// @Tags Orchestrator
// @Accept json
// @Produce json
// @Success 200 {object} object{status=string,services=object,timestamp=string}
// @Failure 500 {object} object{error=string}
// @Router /orchestrator/health [get]
func (h *OrchestratorHandler) handleHealth(c *gin.Context) {
	// Get service status
	services := h.orchestrator.GetServiceStatus()

	// Determine overall health - 更宽松的健康检查逻辑
	overallHealth := "healthy"
	runningServices := 0
	totalServices := len(services)
	criticalServices := 0

	for _, service := range services {
		if service.Status == "running" {
			runningServices++
		} else if service.Status == "failed" || service.Status == "error" {
			criticalServices++
		}
	}

	// 健康状态判断逻辑：
	// - 如果没有服务或所有服务都是stopped状态，认为是健康的（系统空闲）
	// - 如果有失败的服务，认为是降级的
	// - 如果有运行的服务，认为是健康的
	if criticalServices > 0 {
		overallHealth = "degraded"
	} else if totalServices == 0 || runningServices > 0 {
		overallHealth = "healthy"
	} else {
		// 所有服务都是stopped状态，但没有失败，认为是健康的
		overallHealth = "healthy"
	}

	// 总是返回200状态码，除非有严重错误
	statusCode := http.StatusOK

	c.JSON(statusCode, gin.H{
		"status":            overallHealth,
		"timestamp":         time.Now(),
		"services":          services,
		"running_services":  runningServices,
		"total_services":    totalServices,
		"critical_services": criticalServices,
		"version":           "1.0.0",
	})
}
