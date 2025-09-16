package api

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "qcat/internal/automation"
    "qcat/internal/intelligence/selector"
)

// SelectorHandler provides debug endpoints for the strategy selector
type SelectorHandler struct {
    automationSystem *automation.AutomationSystem
}

func NewSelectorHandler(as *automation.AutomationSystem) *SelectorHandler {
    return &SelectorHandler{automationSystem: as}
}

// GetLastDecision returns the last selector decision for a symbol
// @Summary Get last selector decision
// @Description Get last strategy selector decision for a symbol (debug)
// @Tags Selector
// @Accept json
// @Produce json
// @Param symbol query string true "Trading symbol, e.g., BTCUSDT"
// @Success 200 {object} interface{}
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Failure 503 {object} Response
// @Router /selector/decision [get]
func (h *SelectorHandler) GetLastDecision(c *gin.Context) {
    symbol := c.Query("symbol")
    if symbol == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "symbol is required"})
        return
    }
    if h.automationSystem == nil || h.automationSystem.GetExecutor() == nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "executor not available"})
        return
    }
    exec := h.automationSystem.GetExecutor()
    if res, ok := exec.GetSelectorLastDecision(symbol); ok {
        c.JSON(http.StatusOK, res)
        return
    }
    c.JSON(http.StatusNotFound, gin.H{"message": "no decision yet"})
}

// GetStats returns selector stats for a symbol
// @Summary Get selector stats
// @Description Get selector stats for a symbol (debug)
// @Tags Selector
// @Accept json
// @Produce json
// @Param symbol query string true "Trading symbol, e.g., BTCUSDT"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} Response
// @Failure 503 {object} Response
// @Router /selector/stats [get]
func (h *SelectorHandler) GetStats(c *gin.Context) {
    symbol := c.Query("symbol")
    if symbol == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "symbol is required"})
        return
    }
    if h.automationSystem == nil || h.automationSystem.GetExecutor() == nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "executor not available"})
        return
    }
    exec := h.automationSystem.GetExecutor()
    if stats, ok := exec.GetSelectorStats(symbol); ok {
        c.JSON(http.StatusOK, stats)
        return
    }
    c.JSON(http.StatusNotFound, gin.H{"message": "no stats yet"})
}

// UpdatePerformance ingests a simple performance sample for the selector store
// @Summary Push selector performance sample
// @Description Push a performance sample for (symbol, strategyID)
// @Tags Selector
// @Accept json
// @Produce json
// @Param sample body object true "Performance sample"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 503 {object} Response
// @Router /selector/sample [post]
func (h *SelectorHandler) UpdatePerformance(c *gin.Context) {
    type req struct {
        Symbol     string  `json:"symbol"`
        StrategyID string  `json:"strategy_id"`
        PnL        float64 `json:"pnl"`
        Return     float64 `json:"return"`
        Drawdown   float64 `json:"drawdown"`
        Win        bool    `json:"win"`
        Cost       float64 `json:"cost"`
        DurationMs int64   `json:"duration_ms"`
    }
    var r req
    if err := c.ShouldBindJSON(&r); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    if h.automationSystem == nil || h.automationSystem.GetExecutor() == nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "executor not available"})
        return
    }
    sample := selector.PerfSample{
        PnL:       r.PnL,
        Return:    r.Return,
        Drawdown:  r.Drawdown,
        Win:       r.Win,
        Cost:      r.Cost,
        Duration:  time.Duration(r.DurationMs) * time.Millisecond,
        Timestamp: time.Now(),
    }
    ok := h.automationSystem.GetExecutor().UpdateSelectorPerformance(r.Symbol, r.StrategyID, sample)
    if !ok {
        c.JSON(http.StatusBadRequest, gin.H{"error": "update failed"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// SetShadow toggles selector shadow runtime settings
// @Summary Set selector shadow mode
// @Description Enable/disable shadow mode and set symbol whitelist
// @Tags Selector
// @Accept json
// @Produce json
// @Param shadow body object true "Shadow settings"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 503 {object} Response
// @Router /selector/shadow [post]
func (h *SelectorHandler) SetShadow(c *gin.Context) {
    type req struct {
        Enabled bool     `json:"enabled"`
        Symbols []string `json:"symbols"`
    }
    var r req
    if err := c.ShouldBindJSON(&r); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    if h.automationSystem == nil || h.automationSystem.GetExecutor() == nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "executor not available"})
        return
    }
    h.automationSystem.GetExecutor().SetSelectorShadow(r.Enabled, r.Symbols)
    c.JSON(http.StatusOK, gin.H{"status": "ok"})
}


