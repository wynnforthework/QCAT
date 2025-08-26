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
	return &BlacklistHandler{
		db:         db,
		gatekeeper: validation.NewStrategyGatekeeperWithDB(db),
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
func (h *BlacklistHandler) ClearExpiredEntries(c *gin.Context) {
	ctx := c.Request.Context()

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
