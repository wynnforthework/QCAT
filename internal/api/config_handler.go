package api

import (
	"net/http"

	"qcat/internal/config"

	"github.com/gin-gonic/gin"
)

// ConfigHandler handles configuration-related API requests
type ConfigHandler struct {
	watcher *config.ConfigWatcher
}

// NewConfigHandler creates a new configuration handler
func NewConfigHandler(watcher *config.ConfigWatcher) *ConfigHandler {
	return &ConfigHandler{
		watcher: watcher,
	}
}

// GetAlgorithmConfig returns current algorithm configuration
func (h *ConfigHandler) GetAlgorithmConfig(c *gin.Context) {
	algorithmConfig := config.GetAlgorithmConfig()
	if algorithmConfig == nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Algorithm configuration not loaded",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    algorithmConfig,
	})
}

// UpdateAlgorithmConfig updates algorithm configuration
func (h *ConfigHandler) UpdateAlgorithmConfig(c *gin.Context) {
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	algorithmConfig := config.GetAlgorithmConfig()
	if algorithmConfig == nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Algorithm configuration not loaded",
		})
		return
	}

	// Update configuration
	if err := algorithmConfig.UpdateConfig(updates); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Failed to update configuration: " + err.Error(),
		})
		return
	}

	// Validate updated configuration
	if err := algorithmConfig.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid configuration: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Configuration updated successfully",
		Data:    algorithmConfig,
	})
}

// GetOptimizerConfig returns optimizer configuration
func (h *ConfigHandler) GetOptimizerConfig(c *gin.Context) {
	algorithmConfig := config.GetAlgorithmConfig()
	if algorithmConfig == nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Algorithm configuration not loaded",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    algorithmConfig.Optimizer,
	})
}

// UpdateOptimizerConfig updates optimizer configuration
func (h *ConfigHandler) UpdateOptimizerConfig(c *gin.Context) {
	var optimizerConfig config.OptimizerConfig
	if err := c.ShouldBindJSON(&optimizerConfig); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	algorithmConfig := config.GetAlgorithmConfig()
	if algorithmConfig == nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Algorithm configuration not loaded",
		})
		return
	}

	// Update optimizer configuration
	algorithmConfig.Optimizer = optimizerConfig

	// Validate updated configuration
	if err := algorithmConfig.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid configuration: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Optimizer configuration updated successfully",
		Data:    algorithmConfig.Optimizer,
	})
}

// GetRiskConfig returns risk management configuration
func (h *ConfigHandler) GetRiskConfig(c *gin.Context) {
	algorithmConfig := config.GetAlgorithmConfig()
	if algorithmConfig == nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Algorithm configuration not loaded",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    algorithmConfig.RiskMgmt,
	})
}

// UpdateRiskConfig updates risk management configuration
func (h *ConfigHandler) UpdateRiskConfig(c *gin.Context) {
	var riskConfig config.RiskManagementConfig
	if err := c.ShouldBindJSON(&riskConfig); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	algorithmConfig := config.GetAlgorithmConfig()
	if algorithmConfig == nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Algorithm configuration not loaded",
		})
		return
	}

	// Update risk management configuration
	algorithmConfig.RiskMgmt = riskConfig

	// Validate updated configuration
	if err := algorithmConfig.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Invalid configuration: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Risk management configuration updated successfully",
		Data:    algorithmConfig.RiskMgmt,
	})
}

// GetHotlistConfig returns hotlist configuration
func (h *ConfigHandler) GetHotlistConfig(c *gin.Context) {
	algorithmConfig := config.GetAlgorithmConfig()
	if algorithmConfig == nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Algorithm configuration not loaded",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    algorithmConfig.Hotlist,
	})
}

// UpdateHotlistConfig updates hotlist configuration
func (h *ConfigHandler) UpdateHotlistConfig(c *gin.Context) {
	var hotlistConfig config.HotlistConfig
	if err := c.ShouldBindJSON(&hotlistConfig); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	algorithmConfig := config.GetAlgorithmConfig()
	if algorithmConfig == nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Algorithm configuration not loaded",
		})
		return
	}

	// Update hotlist configuration
	algorithmConfig.Hotlist = hotlistConfig

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Hotlist configuration updated successfully",
		Data:    algorithmConfig.Hotlist,
	})
}

// ReloadConfig forces a configuration reload
func (h *ConfigHandler) ReloadConfig(c *gin.Context) {
	if err := config.ReloadAlgorithmConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to reload configuration: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Configuration reloaded successfully",
	})
}

// GetConfigStatus returns configuration watcher status
func (h *ConfigHandler) GetConfigStatus(c *gin.Context) {
	status := map[string]interface{}{
		"watcher_running": h.watcher.IsRunning(),
		"config_loaded":   config.GetAlgorithmConfig() != nil,
	}

	if algorithmConfig := config.GetAlgorithmConfig(); algorithmConfig != nil {
		// Add configuration validation status
		if err := algorithmConfig.Validate(); err != nil {
			status["validation_error"] = err.Error()
			status["valid"] = false
		} else {
			status["valid"] = true
		}
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    status,
	})
}

// ValidateConfig validates current configuration
func (h *ConfigHandler) ValidateConfig(c *gin.Context) {
	algorithmConfig := config.GetAlgorithmConfig()
	if algorithmConfig == nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Algorithm configuration not loaded",
		})
		return
	}

	if err := algorithmConfig.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   "Configuration validation failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Configuration is valid",
	})
}

// GetConfigDefaults returns default configuration values
func (h *ConfigHandler) GetConfigDefaults(c *gin.Context) {
	defaults := map[string]interface{}{
		"optimizer": map[string]interface{}{
			"grid_search": map[string]interface{}{
				"default_grid_size":       10,
				"max_iterations":          1000,
				"convergence_threshold":   0.001,
			},
			"walk_forward": map[string]interface{}{
				"train_ratio":      0.7,
				"validation_ratio": 0.15,
				"test_ratio":       0.15,
				"min_samples":      100,
				"step_size":        30,
			},
		},
		"elimination": map[string]interface{}{
			"window_size_days":        20,
			"min_trades":              50,
			"performance_threshold":   0.05,
			"correlation_threshold":   0.8,
			"volatility_threshold":    0.3,
		},
		"risk_management": map[string]interface{}{
			"position": map[string]interface{}{
				"max_weight_percent":   20.0,
				"min_weight_percent":   1.0,
				"rebalance_threshold":  5.0,
				"max_leverage":         10,
			},
			"stop_loss": map[string]interface{}{
				"default_atr_multiplier": 2.0,
				"trailing_stop_percent":  1.0,
				"max_stop_loss_percent":  10.0,
				"min_stop_loss_percent":  0.5,
			},
		},
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    defaults,
	})
}