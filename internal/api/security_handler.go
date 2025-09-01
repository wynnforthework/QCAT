package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"qcat/internal/security"

	"github.com/gin-gonic/gin"
)

// SecurityHandler handles security-related API requests
type SecurityHandler struct {
	keyManager  *security.KeyManager
	auditLogger *security.AuditLogger
}

// NewSecurityHandler creates a new security handler
func NewSecurityHandler(keyManager *security.KeyManager, auditLogger *security.AuditLogger) *SecurityHandler {
	return &SecurityHandler{
		keyManager:  keyManager,
		auditLogger: auditLogger,
	}
}

// RegisterRoutes registers security management routes
func (h *SecurityHandler) RegisterRoutes(router *gin.RouterGroup) {
	security := router.Group("/security")
	{
		// API Key management
		keys := security.Group("/keys")
		{
			keys.POST("/", h.createAPIKey)
			keys.GET("/", h.listAPIKeys)
			keys.GET("/:keyId", h.getAPIKey)
			keys.POST("/:keyId/rotate", h.rotateAPIKey)
			keys.POST("/:keyId/revoke", h.revokeAPIKey)
			keys.GET("/:keyId/usage", h.getKeyUsage)
			keys.GET("/:keyId/schedule", h.getRotationSchedule)
		}

		// Audit logs
		audit := security.Group("/audit")
		{
			audit.GET("/logs", h.getAuditLogs)
			audit.GET("/logs/:id", h.getAuditLog)
			audit.POST("/logs/export", h.exportAuditLogs)
			audit.GET("/integrity", h.verifyIntegrity)
		}

		// Security monitoring
		monitoring := security.Group("/monitoring")
		{
			monitoring.GET("/alerts", h.getSecurityAlerts)
			monitoring.POST("/alerts/:id/acknowledge", h.acknowledgeAlert)
			monitoring.GET("/events", h.getSecurityEvents)
		}
	}
}

// createAPIKey creates a new API key
// @Summary Create API key
// @Description Create a new API key with specified permissions and expiration
// @Tags Security
// @Accept json
// @Produce json
// @Param request body object{name=string,permissions=[]string,expires_at=string} true "API key details"
// @Success 200 {object} object{status=string,key_id=string,api_key=string,message=string}
// @Failure 400 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /security/api-keys [post]
func (h *SecurityHandler) createAPIKey(c *gin.Context) {
	var req struct {
		Name        string                   `json:"name" binding:"required"`
		Permissions []security.KeyPermission `json:"permissions" binding:"required"`
		ExpiresAt   time.Time                `json:"expires_at"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Set default expiration if not provided
	if req.ExpiresAt.IsZero() {
		req.ExpiresAt = time.Now().Add(365 * 24 * time.Hour) // 1 year
	}

	// Create the key
	keyInfo, keyString, err := h.keyManager.GenerateKey(req.Name, req.Permissions, req.ExpiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create API key",
			"details": err.Error(),
		})
		return
	}

	// Log the action
	userID := getUserID(c)
	h.auditLogger.LogUserAction(userID, "create_api_key", keyInfo.ID, map[string]interface{}{
		"key_name":    req.Name,
		"permissions": req.Permissions,
		"expires_at":  req.ExpiresAt,
	}, true, "")

	c.JSON(http.StatusCreated, gin.H{
		"success":  true,
		"key_info": keyInfo,
		"api_key":  keyString, // Only returned once
		"message":  "API key created successfully",
	})
}

// listAPIKeys lists all API keys
// @Summary List API keys
// @Description Get list of all API keys with optional filtering
// @Tags Security
// @Accept json
// @Produce json
// @Param status query string false "Filter by key status"
// @Param name query string false "Filter by key name"
// @Success 200 {object} object{status=string,keys=[]object,count=integer}
// @Failure 500 {object} object{error=string}
// @Router /security/api-keys [get]
func (h *SecurityHandler) listAPIKeys(c *gin.Context) {
	// Parse query parameters for filtering
	filter := &security.KeyFilter{}

	if status := c.Query("status"); status != "" {
		filter.Status = security.KeyStatus(status)
	}
	if name := c.Query("name"); name != "" {
		filter.Name = name
	}
	if permission := c.Query("permission"); permission != "" {
		filter.Permission = permission
	}

	keys, err := h.keyManager.ListKeys(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list API keys",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"keys":    keys,
		"count":   len(keys),
	})
}

// getAPIKey gets a specific API key
// @Summary Get API key details
// @Description Get detailed information about a specific API key
// @Tags Security
// @Accept json
// @Produce json
// @Param keyId path string true "API Key ID"
// @Success 200 {object} object{status=string,key_info=object}
// @Failure 400 {object} object{error=string}
// @Failure 404 {object} object{error=string}
// @Router /security/api-keys/{keyId} [get]
func (h *SecurityHandler) getAPIKey(c *gin.Context) {
	keyID := c.Param("keyId")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Key ID is required",
		})
		return
	}

	keyInfo, err := h.keyManager.GetKey(keyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "API key not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"key_info": keyInfo,
	})
}

// rotateAPIKey rotates an API key
// @Summary Rotate API key
// @Description Rotate an existing API key to generate a new secret
// @Tags Security
// @Accept json
// @Produce json
// @Param keyId path string true "API Key ID"
// @Success 200 {object} object{status=string,new_key=string,message=string}
// @Failure 400 {object} object{error=string}
// @Failure 404 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /security/api-keys/{keyId}/rotate [post]
func (h *SecurityHandler) rotateAPIKey(c *gin.Context) {
	keyID := c.Param("keyId")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Key ID is required",
		})
		return
	}

	newKeyInfo, newKeyString, err := h.keyManager.RotateKey(keyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to rotate API key",
			"details": err.Error(),
		})
		return
	}

	// Log the action
	userID := getUserID(c)
	h.auditLogger.LogUserAction(userID, "rotate_api_key", keyID, map[string]interface{}{
		"rotation_type": "manual",
	}, true, "")

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"key_info":    newKeyInfo,
		"new_api_key": newKeyString, // Only returned once
		"message":     "API key rotated successfully",
	})
}

// revokeAPIKey revokes an API key
// @Summary Revoke API key
// @Description Revoke an existing API key to disable access
// @Tags Security
// @Accept json
// @Produce json
// @Param keyId path string true "API Key ID"
// @Param request body object{reason=string} false "Revocation reason"
// @Success 200 {object} object{status=string,message=string}
// @Failure 400 {object} object{error=string}
// @Failure 404 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /security/api-keys/{keyId}/revoke [post]
func (h *SecurityHandler) revokeAPIKey(c *gin.Context) {
	keyID := c.Param("keyId")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Key ID is required",
		})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req.Reason = "Manual revocation"
	}

	err := h.keyManager.RevokeKey(keyID, req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to revoke API key",
			"details": err.Error(),
		})
		return
	}

	// Log the action
	userID := getUserID(c)
	h.auditLogger.LogUserAction(userID, "revoke_api_key", keyID, map[string]interface{}{
		"reason": req.Reason,
	}, true, "")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "API key revoked successfully",
	})
}

// getKeyUsage gets usage statistics for an API key
// @Summary Get API key usage
// @Description Get usage statistics for a specific API key
// @Tags Security
// @Accept json
// @Produce json
// @Param keyId path string true "API Key ID"
// @Param period query string false "Time period (24h, 7d, 30d)" default(24h)
// @Success 200 {object} object{status=string,usage=object,period=string}
// @Failure 400 {object} object{error=string}
// @Failure 404 {object} object{error=string}
// @Router /security/api-keys/{keyId}/usage [get]
func (h *SecurityHandler) getKeyUsage(c *gin.Context) {
	keyID := c.Param("keyId")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Key ID is required",
		})
		return
	}

	// Parse period parameter
	periodStr := c.DefaultQuery("period", "24h")
	period, err := time.ParseDuration(periodStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid period format",
			"details": err.Error(),
		})
		return
	}

	stats, err := h.keyManager.GetKeyUsageStats(keyID, period)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Usage statistics not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"usage_stats": stats,
		"period":      periodStr,
	})
}

// getRotationSchedule gets rotation schedule for an API key
// @Summary Get rotation schedule
// @Description Get automatic rotation schedule for a specific API key
// @Tags Security
// @Accept json
// @Produce json
// @Param keyId path string true "API Key ID"
// @Success 200 {object} object{status=string,schedule=object}
// @Failure 400 {object} object{error=string}
// @Failure 404 {object} object{error=string}
// @Router /security/api-keys/{keyId}/rotation-schedule [get]
func (h *SecurityHandler) getRotationSchedule(c *gin.Context) {
	keyID := c.Param("keyId")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Key ID is required",
		})
		return
	}

	keyInfo, err := h.keyManager.GetKey(keyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "API key not found",
			"details": err.Error(),
		})
		return
	}

	schedule := h.keyManager.GetRotationSchedule(keyInfo)

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"schedule": schedule,
	})
}

// getAuditLogs gets audit logs with optional filtering
// @Summary Get audit logs
// @Description Get security audit logs with optional filtering
// @Tags Security
// @Accept json
// @Produce json
// @Param user_id query string false "Filter by user ID"
// @Param action query string false "Filter by action type"
// @Param start_time query string false "Start time (RFC3339)"
// @Param end_time query string false "End time (RFC3339)"
// @Param limit query int false "Limit number of results" default(100)
// @Success 200 {object} object{status=string,entries=[]object,count=integer}
// @Failure 500 {object} object{error=string}
// @Router /security/audit-logs [get]
func (h *SecurityHandler) getAuditLogs(c *gin.Context) {
	filter := &security.AuditFilter{}

	// Parse query parameters
	if userID := c.Query("user_id"); userID != "" {
		filter.UserID = userID
	}
	if action := c.Query("action"); action != "" {
		filter.Action = action
	}
	if resource := c.Query("resource"); resource != "" {
		filter.Resource = resource
	}
	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filter.StartTime = t
		}
	}
	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			filter.EndTime = t
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}

	entries, err := h.auditLogger.GetEntries(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get audit logs",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"entries": entries,
		"count":   len(entries),
	})
}

// getAuditLog gets a specific audit log entry
// @Summary Get audit log entry
// @Description Get details of a specific audit log entry
// @Tags Security
// @Accept json
// @Produce json
// @Param id path string true "Audit log entry ID"
// @Success 200 {object} object{status=string,entry=object}
// @Failure 400 {object} object{error=string}
// @Failure 404 {object} object{error=string}
// @Router /security/audit-logs/{id} [get]
func (h *SecurityHandler) getAuditLog(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Audit log ID is required",
		})
		return
	}

	entry, err := h.auditLogger.GetEntry(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Audit log entry not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"entry":   entry,
	})
}

// exportAuditLogs exports audit logs
// @Summary Export audit logs
// @Description Export audit logs in various formats (JSON, CSV, PDF)
// @Tags Security
// @Accept json
// @Produce application/json
// @Produce text/csv
// @Produce application/pdf
// @Param request body object{filter=object,format=string} true "Export parameters"
// @Success 200 {file} file "Exported audit logs file"
// @Failure 400 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /security/audit-logs/export [post]
func (h *SecurityHandler) exportAuditLogs(c *gin.Context) {
	var req struct {
		Filter *security.AuditFilter `json:"filter"`
		Format string                `json:"format"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	if req.Format == "" {
		req.Format = "json"
	}

	data, err := h.auditLogger.ExportLogs(req.Filter, req.Format)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to export audit logs",
			"details": err.Error(),
		})
		return
	}

	// Set appropriate content type
	var contentType string
	switch req.Format {
	case "json":
		contentType = "application/json"
	case "csv":
		contentType = "text/csv"
	default:
		contentType = "application/octet-stream"
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=audit_logs.%s", req.Format))
	c.Data(http.StatusOK, contentType, data)
}

// verifyIntegrity verifies audit log integrity
// @Summary Verify audit log integrity
// @Description Verify the integrity of audit logs using cryptographic checksums
// @Tags Security
// @Accept json
// @Produce json
// @Success 200 {object} object{status=string,report=object}
// @Failure 500 {object} object{error=string,details=string}
// @Router /security/audit-logs/verify [post]
func (h *SecurityHandler) verifyIntegrity(c *gin.Context) {
	report, err := h.auditLogger.VerifyIntegrity()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to verify integrity",
			"details": err.Error(),
		})
		return
	}

	statusCode := http.StatusOK
	if !report.IntegrityValid {
		statusCode = http.StatusConflict
	}

	c.JSON(statusCode, gin.H{
		"success": report.IntegrityValid,
		"report":  report,
	})
}

// getSecurityAlerts gets security alerts
// @Summary Get security alerts
// @Description Get current security alerts and warnings
// @Tags Security
// @Accept json
// @Produce json
// @Param acknowledged query boolean false "Filter by acknowledgment status" default(false)
// @Success 200 {object} object{status=string,alerts=[]object,count=integer}
// @Failure 500 {object} object{error=string}
// @Router /security/alerts [get]
func (h *SecurityHandler) getSecurityAlerts(c *gin.Context) {
	acknowledgedStr := c.DefaultQuery("acknowledged", "false")
	acknowledged := acknowledgedStr == "true"

	alerts := h.keyManager.GetAlerts(acknowledged)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"alerts":  alerts,
		"count":   len(alerts),
	})
}

// acknowledgeAlert acknowledges a security alert
// @Summary Acknowledge security alert
// @Description Acknowledge a specific security alert
// @Tags Security
// @Accept json
// @Produce json
// @Param id path string true "Alert ID"
// @Success 200 {object} object{status=string,message=string}
// @Failure 400 {object} object{error=string}
// @Failure 404 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /security/alerts/{id}/acknowledge [post]
func (h *SecurityHandler) acknowledgeAlert(c *gin.Context) {
	alertID := c.Param("id")
	if alertID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Alert ID is required",
		})
		return
	}

	userID := getUserID(c)
	err := h.keyManager.AcknowledgeAlert(alertID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Failed to acknowledge alert",
			"details": err.Error(),
		})
		return
	}

	// Log the action
	h.auditLogger.LogUserAction(userID, "acknowledge_alert", alertID, map[string]interface{}{
		"alert_id": alertID,
	}, true, "")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Alert acknowledged successfully",
	})
}

// getSecurityEvents gets recent security events
// @Summary Get security events
// @Description Get recent security events and activities
// @Tags Security
// @Accept json
// @Produce json
// @Param limit query int false "Limit number of results" default(100)
// @Success 200 {object} object{status=string,events=[]object,count=integer}
// @Failure 500 {object} object{error=string}
// @Router /security/events [get]
func (h *SecurityHandler) getSecurityEvents(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 100
	}

	events := h.keyManager.GetRecentEvents(limit)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"events":  events,
		"count":   len(events),
	})
}

// Helper function to get user ID from context
func getUserID(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(string); ok {
			return uid
		}
	}
	return "unknown"
}
