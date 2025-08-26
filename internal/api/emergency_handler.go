package api

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"qcat/internal/emergency"
)

// EmergencyHandler 紧急停止处理器
type EmergencyHandler struct {
	db          *sql.DB
	emergencyMgr *emergency.EmergencyStopManager
}

// NewEmergencyHandler 创建紧急停止处理器
func NewEmergencyHandler(db *sql.DB) *EmergencyHandler {
	return &EmergencyHandler{
		db:          db,
		emergencyMgr: emergency.NewEmergencyStopManager(db),
	}
}

// EmergencyStopRequest 紧急停止请求
type EmergencyStopRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// EmergencyResetRequest 紧急重置请求
type EmergencyResetRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// EmergencyStopAll 紧急停止所有策略
func (h *EmergencyHandler) EmergencyStopAll(c *gin.Context) {
	var req EmergencyStopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid request: " + err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// 执行紧急停止
	if err := h.emergencyMgr.EmergencyStopAllStrategies(ctx, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to execute emergency stop: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"message": "Emergency stop executed successfully",
			"reason":  req.Reason,
			"status":  h.emergencyMgr.GetEmergencyStopStatus(),
		},
	})
}

// GetEmergencyStatus 获取紧急停止状态
func (h *EmergencyHandler) GetEmergencyStatus(c *gin.Context) {
	status := h.emergencyMgr.GetEmergencyStopStatus()

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    status,
	})
}

// ResetEmergencyStop 重置紧急停止状态
func (h *EmergencyHandler) ResetEmergencyStop(c *gin.Context) {
	var req EmergencyResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid request: " + err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// 重置紧急停止状态
	if err := h.emergencyMgr.ResetEmergencyStop(ctx, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to reset emergency stop: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"message": "Emergency stop reset successfully",
			"reason":  req.Reason,
			"status":  h.emergencyMgr.GetEmergencyStopStatus(),
		},
	})
}

// GetEmergencyHistory 获取紧急停止历史记录
func (h *EmergencyHandler) GetEmergencyHistory(c *gin.Context) {
	ctx := c.Request.Context()

	// 获取紧急停止事件历史
	query := `
		SELECT id, reason, total_strategies, failed_count, stopped_at, created_at
		FROM emergency_stop_events
		ORDER BY stopped_at DESC
		LIMIT 50
	`

	rows, err := h.db.QueryContext(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to query emergency stop history: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	var events []map[string]interface{}
	for rows.Next() {
		var event struct {
			ID              int    `db:"id"`
			Reason          string `db:"reason"`
			TotalStrategies int    `db:"total_strategies"`
			FailedCount     int    `db:"failed_count"`
			StoppedAt       string `db:"stopped_at"`
			CreatedAt       string `db:"created_at"`
		}

		err := rows.Scan(
			&event.ID, &event.Reason, &event.TotalStrategies,
			&event.FailedCount, &event.StoppedAt, &event.CreatedAt,
		)
		if err != nil {
			continue
		}

		events = append(events, map[string]interface{}{
			"id":               event.ID,
			"reason":           event.Reason,
			"total_strategies": event.TotalStrategies,
			"failed_count":     event.FailedCount,
			"stopped_at":       event.StoppedAt,
			"created_at":       event.CreatedAt,
		})
	}

	// 获取重置事件历史
	resetQuery := `
		SELECT id, reason, reset_at, created_at
		FROM emergency_stop_reset_events
		ORDER BY reset_at DESC
		LIMIT 20
	`

	resetRows, err := h.db.QueryContext(ctx, resetQuery)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to query reset history: " + err.Error(),
		})
		return
	}
	defer resetRows.Close()

	var resetEvents []map[string]interface{}
	for resetRows.Next() {
		var resetEvent struct {
			ID        int    `db:"id"`
			Reason    string `db:"reason"`
			ResetAt   string `db:"reset_at"`
			CreatedAt string `db:"created_at"`
		}

		err := resetRows.Scan(
			&resetEvent.ID, &resetEvent.Reason,
			&resetEvent.ResetAt, &resetEvent.CreatedAt,
		)
		if err != nil {
			continue
		}

		resetEvents = append(resetEvents, map[string]interface{}{
			"id":         resetEvent.ID,
			"reason":     resetEvent.Reason,
			"reset_at":   resetEvent.ResetAt,
			"created_at": resetEvent.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"emergency_stops": events,
			"resets":          resetEvents,
			"current_status":  h.emergencyMgr.GetEmergencyStopStatus(),
		},
	})
}

// IsEmergencyActive 检查是否处于紧急停止状态（中间件使用）
func (h *EmergencyHandler) IsEmergencyActive() bool {
	return h.emergencyMgr.IsEmergencyStopped()
}
