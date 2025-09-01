package api

import (
	"database/sql"
	"net/http"
	"time"

	"qcat/internal/strategy/validation"

	"github.com/gin-gonic/gin"
)

// BlacklistHandler 黑名单处理器
type BlacklistHandler struct {
	db         *sql.DB
	gatekeeper *validation.StrategyGatekeeper
}

// NewBlacklistHandler 创建黑名单处理器
func NewBlacklistHandler(db *sql.DB) *BlacklistHandler {
	var gatekeeper *validation.StrategyGatekeeper
	if db != nil {
		gatekeeper = validation.NewStrategyGatekeeperWithDB(db)
	} else {
		// 当数据库不可用时，创建不依赖数据库的守门员
		gatekeeper = validation.NewStrategyGatekeeper()
	}

	return &BlacklistHandler{
		db:         db,
		gatekeeper: gatekeeper,
	}
}

// BlacklistEntry 黑名单条目响应
type BlacklistEntry struct {
	ID         int        `json:"id"`
	StrategyID string     `json:"strategy_id"`
	Reason     string     `json:"reason"`
	BlockedAt  time.Time  `json:"blocked_at"`
	BlockedBy  string     `json:"blocked_by"`
	Permanent  bool       `json:"permanent"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// AddToBlacklistRequest 添加到黑名单请求
type AddToBlacklistRequest struct {
	StrategyID string     `json:"strategy_id" binding:"required"`
	Reason     string     `json:"reason" binding:"required"`
	Permanent  bool       `json:"permanent"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

// ListBlacklist 获取黑名单列表
// @Summary Get blacklist entries
// @Description Get list of all blacklisted strategies with their details
// @Tags Blacklist
// @Accept json
// @Produce json
// @Success 200 {object} Response{data=[]object}
// @Failure 500 {object} Response
// @Router /blacklist/ [get]
func (h *BlacklistHandler) ListBlacklist(c *gin.Context) {
	ctx := c.Request.Context()

	query := `
		SELECT id, strategy_id, reason, blocked_at, blocked_by, permanent, expires_at, created_at, updated_at
		FROM strategy_blacklist
		ORDER BY blocked_at DESC
	`

	rows, err := h.db.QueryContext(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to query blacklist: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	var entries []BlacklistEntry
	for rows.Next() {
		var entry BlacklistEntry
		var expiresAt sql.NullTime

		err := rows.Scan(
			&entry.ID, &entry.StrategyID, &entry.Reason,
			&entry.BlockedAt, &entry.BlockedBy, &entry.Permanent,
			&expiresAt, &entry.CreatedAt, &entry.UpdatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Error:   "Failed to scan blacklist entry: " + err.Error(),
			})
			return
		}

		if expiresAt.Valid {
			entry.ExpiresAt = &expiresAt.Time
		}

		entries = append(entries, entry)
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"entries": entries,
			"total":   len(entries),
		},
	})
}

// AddToBlacklist 添加策略到黑名单
// @Summary Add strategy to blacklist
// @Description Add a strategy to the blacklist with specified reason and duration
// @Tags Blacklist
// @Accept json
// @Produce json
// @Param request body AddToBlacklistRequest true "Blacklist entry details"
// @Success 200 {object} Response{data=object{strategy_id=string,status=string}}
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /blacklist/ [post]
func (h *BlacklistHandler) AddToBlacklist(c *gin.Context) {
	var req AddToBlacklistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid request: " + err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// 使用策略守门员添加到黑名单
	if err := h.gatekeeper.DisableStrategy(ctx, req.StrategyID, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to add to blacklist: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"strategy_id": req.StrategyID,
			"message":     "Strategy added to blacklist successfully",
		},
	})
}

// RemoveFromBlacklist 从黑名单移除策略
// @Summary Remove strategy from blacklist
// @Description Remove a strategy from the blacklist by strategy ID
// @Tags Blacklist
// @Accept json
// @Produce json
// @Param strategy_id path string true "Strategy ID to remove from blacklist"
// @Success 200 {object} Response{data=object{strategy_id=string,status=string}}
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /blacklist/{strategy_id} [delete]
func (h *BlacklistHandler) RemoveFromBlacklist(c *gin.Context) {
	strategyID := c.Param("strategy_id")
	if strategyID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Strategy ID is required",
		})
		return
	}

	ctx := c.Request.Context()

	query := `DELETE FROM strategy_blacklist WHERE strategy_id = $1`
	result, err := h.db.ExecContext(ctx, query, strategyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to remove from blacklist: " + err.Error(),
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "Strategy not found in blacklist",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"strategy_id": strategyID,
			"message":     "Strategy removed from blacklist successfully",
		},
	})
}

// CheckBlacklist 检查策略是否在黑名单中
// @Summary Check if strategy is blacklisted
// @Description Check if a specific strategy is currently in the blacklist
// @Tags Blacklist
// @Accept json
// @Produce json
// @Param strategy_id path string true "Strategy ID to check"
// @Success 200 {object} Response{data=object{strategy_id=string,is_blacklisted=boolean,reason=string,expires_at=string}}
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /blacklist/{strategy_id}/check [get]
func (h *BlacklistHandler) CheckBlacklist(c *gin.Context) {
	strategyID := c.Param("strategy_id")
	if strategyID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Strategy ID is required",
		})
		return
	}

	ctx := c.Request.Context()

	// 如果数据库不可用，使用内存中的黑名单检查
	if h.db == nil {
		// 使用守门员的内存黑名单检查
		isBlacklisted := h.gatekeeper.IsBlacklisted(strategyID)
		c.JSON(http.StatusOK, Response{
			Success: true,
			Data: map[string]interface{}{
				"strategy_id":    strategyID,
				"is_blacklisted": isBlacklisted,
				"source":         "memory",
				"message":        "Database unavailable, using memory cache",
			},
		})
		return
	}

	query := `
		SELECT id, strategy_id, reason, blocked_at, blocked_by, permanent, expires_at
		FROM strategy_blacklist
		WHERE strategy_id = $1
		AND (permanent = true OR (expires_at IS NOT NULL AND expires_at > NOW()))
	`

	var entry BlacklistEntry
	var expiresAt sql.NullTime

	err := h.db.QueryRowContext(ctx, query, strategyID).Scan(
		&entry.ID, &entry.StrategyID, &entry.Reason,
		&entry.BlockedAt, &entry.BlockedBy, &entry.Permanent, &expiresAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, Response{
			Success: true,
			Data: map[string]interface{}{
				"strategy_id":    strategyID,
				"is_blacklisted": false,
			},
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to check blacklist: " + err.Error(),
		})
		return
	}

	if expiresAt.Valid {
		entry.ExpiresAt = &expiresAt.Time
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"strategy_id":    strategyID,
			"is_blacklisted": true,
			"entry":          entry,
		},
	})
}

// ClearExpiredEntries 清理过期的黑名单条目
// @Summary Clear expired blacklist entries
// @Description Remove all expired entries from the blacklist
// @Tags Blacklist
// @Accept json
// @Produce json
// @Success 200 {object} Response{data=object{cleared_count=integer,message=string}}
// @Failure 500 {object} Response
// @Router /blacklist/clear-expired [post]
func (h *BlacklistHandler) ClearExpiredEntries(c *gin.Context) {
	ctx := c.Request.Context()

	// 如果数据库不可用，返回适当的响应
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, Response{
			Success: false,
			Error:   "Database unavailable, cannot clear expired entries",
		})
		return
	}

	query := `
		DELETE FROM strategy_blacklist
		WHERE permanent = false AND expires_at IS NOT NULL AND expires_at <= NOW()
	`

	result, err := h.db.ExecContext(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to clear expired entries: " + err.Error(),
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"cleared_count": rowsAffected,
			"message":       "Expired blacklist entries cleared successfully",
		},
	})
}
