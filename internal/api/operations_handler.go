package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"qcat/internal/operations"
)

// OperationsHandler exposes operational health endpoints.
type OperationsHandler struct {
	manager *operations.Manager
}

// NewOperationsHandler creates a new operations handler instance.
func NewOperationsHandler(manager *operations.Manager) *OperationsHandler {
	return &OperationsHandler{manager: manager}
}

// RegisterRoutes wires operations routes under the provided group.
func (h *OperationsHandler) RegisterRoutes(router *gin.RouterGroup) {
	if h == nil || router == nil {
		return
	}

	ops := router.Group("/operations")
	{
		ops.GET("/matrix", h.GetMatrix)
		ops.POST("/refresh", h.RefreshChecks)
	}
}

// GetMatrix returns the latest operational snapshot. Optional query refresh=true triggers a synchronous run.
func (h *OperationsHandler) GetMatrix(c *gin.Context) {
	if h == nil || h.manager == nil {
		c.JSON(http.StatusServiceUnavailable, Response{Success: false, Error: "operations manager unavailable"})
		return
	}

	if refresh := c.Query("refresh"); refresh == "true" || refresh == "1" {
		h.manager.RunAllChecksOnce()
	}

	snapshot := h.manager.GetSnapshot()
	c.JSON(http.StatusOK, Response{Success: true, Data: snapshot})
}

// RefreshChecks forces all checks to run once and returns the updated snapshot.
func (h *OperationsHandler) RefreshChecks(c *gin.Context) {
	if h == nil || h.manager == nil {
		c.JSON(http.StatusServiceUnavailable, Response{Success: false, Error: "operations manager unavailable"})
		return
	}

	h.manager.RunAllChecksOnce()
	snapshot := h.manager.GetSnapshot()
	c.JSON(http.StatusOK, Response{Success: true, Message: "checks refreshed", Data: snapshot})
}
