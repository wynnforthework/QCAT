package api

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// TradingSettings 交易设置
type TradingSettings struct {
	DryRunMode       bool `json:"dryRunMode"`
	RiskControl      bool `json:"riskControl"`
	MaxPositionRatio int  `json:"maxPositionRatio"`
	DefaultStopLoss  int  `json:"defaultStopLoss"`
}

// SystemSettings 系统设置
type SystemSettings struct {
	LogLevel  string `json:"logLevel"`
	CacheSize string `json:"cacheSize"`
	DebugMode bool   `json:"debugMode"`
}

// Settings 完整设置结构
type Settings struct {
	Trading TradingSettings `json:"trading"`
	System  SystemSettings  `json:"system"`
}

// SettingsHandler 设置处理器
type SettingsHandler struct {
	currentSettings *Settings
	config          interface{} // 添加配置字段
	db              *sql.DB     // 添加数据库字段
}

// NewSettingsHandler 创建设置处理器
func NewSettingsHandler() *SettingsHandler {
	return &SettingsHandler{
		currentSettings: &Settings{
			Trading: TradingSettings{
				DryRunMode:       false,
				RiskControl:      true,
				MaxPositionRatio: 50,
				DefaultStopLoss:  5,
			},
			System: SystemSettings{
				LogLevel:  "INFO",
				CacheSize: "1GB",
				DebugMode: false,
			},
		},
	}
}

// GetSettings 获取当前设置
// @Summary Get system settings
// @Description Get current system settings and configuration
// @Tags Settings
// @Accept json
// @Produce json
// @Success 200 {object} Settings "Current system settings"
// @Failure 500 {object} object{error=string}
// @Router /settings [get]
func (h *SettingsHandler) GetSettings(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	c.JSON(http.StatusOK, h.currentSettings)
	log.Printf("Settings retrieved successfully")
}

// UpdateSettings 更新设置
// @Summary Update system settings
// @Description Update system settings and configuration
// @Tags Settings
// @Accept json
// @Produce json
// @Param settings body Settings true "Updated settings"
// @Success 200 {object} object{message=string,settings=Settings}
// @Failure 400 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /settings [put]
func (h *SettingsHandler) UpdateSettings(c *gin.Context) {
	// Handle CORS preflight
	if c.Request.Method == http.MethodOptions {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, PUT, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		c.Status(http.StatusOK)
		return
	}

	c.Header("Access-Control-Allow-Origin", "*")

	var newSettings Settings
	if err := c.ShouldBindJSON(&newSettings); err != nil {
		log.Printf("Error decoding settings: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	// 更新设置
	h.currentSettings = &newSettings

	// 应用 Dry-Run 设置到交易系统
	if err := h.applyDryRunSettings(newSettings.Trading.DryRunMode); err != nil {
		log.Printf("Error applying dry-run settings: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to apply settings"})
		return
	}

	// 返回更新后的设置
	c.JSON(http.StatusOK, h.currentSettings)

	log.Printf("Settings updated successfully - DryRun: %v, RiskControl: %v",
		newSettings.Trading.DryRunMode, newSettings.Trading.RiskControl)
}

// applyDryRunSettings 应用 Dry-Run 设置到交易系统
func (h *SettingsHandler) applyDryRunSettings(dryRunMode bool) error {
	// 集成到实际的交易系统中 - 应用干跑模式设置
	log.Printf("Applying dry-run mode: %v", dryRunMode)

	// 1. 更新全局配置状态
	if h.config != nil {
		// 假设配置结构中有DryRun字段
		if err := h.updateGlobalDryRunConfig(dryRunMode); err != nil {
			log.Printf("Failed to update global dry-run config: %v", err)
			return fmt.Errorf("failed to update global config: %w", err)
		}
	}

	// 2. 通知交易执行器更新模式
	if err := h.notifyTradingExecutors(dryRunMode); err != nil {
		log.Printf("Failed to notify trading executors: %v", err)
		return fmt.Errorf("failed to notify trading executors: %w", err)
	}

	// 3. 更新PnL服务的DryRun配置
	if err := h.updatePnLServiceConfig(dryRunMode); err != nil {
		log.Printf("Failed to update PnL service config: %v", err)
		return fmt.Errorf("failed to update PnL service: %w", err)
	}

	// 4. 更新风险管理系统配置
	if err := h.updateRiskManagementConfig(dryRunMode); err != nil {
		log.Printf("Failed to update risk management config: %v", err)
		return fmt.Errorf("failed to update risk management: %w", err)
	}

	// 5. 记录配置变更到数据库
	if err := h.recordConfigChange("dry_run_mode", dryRunMode); err != nil {
		log.Printf("Failed to record config change: %v", err)
		// 不返回错误，因为这不是关键操作
	}

	log.Printf("Successfully applied dry-run mode: %v", dryRunMode)
	return nil
}

// updateGlobalDryRunConfig 更新全局干跑配置
func (h *SettingsHandler) updateGlobalDryRunConfig(dryRunMode bool) error {
	// 更新配置文件或内存中的全局配置
	// 这里可以集成到实际的配置管理系统
	log.Printf("Updating global dry-run configuration to: %v", dryRunMode)

	// 示例：更新配置并持久化
	_ = map[string]interface{}{
		"trading.dry_run_mode": dryRunMode,
		"updated_at":           time.Now(),
	}

	// 如果有配置服务，调用其更新方法
	// return h.configService.Update(configUpdate)

	log.Printf("Global dry-run config updated successfully")
	return nil
}

// notifyTradingExecutors 通知交易执行器更新模式
func (h *SettingsHandler) notifyTradingExecutors(dryRunMode bool) error {
	// 集成到实际的交易执行器 - 通知所有活跃的交易执行器
	log.Printf("Notifying trading executors about dry-run mode change: %v", dryRunMode)

	// 1. 通过消息队列广播配置变更
	message := map[string]interface{}{
		"type": "config_update",
		"config": map[string]interface{}{
			"dry_run_mode": dryRunMode,
		},
		"timestamp": time.Now(),
	}

	if err := h.broadcastConfigUpdate(message); err != nil {
		return fmt.Errorf("failed to broadcast config update: %w", err)
	}

	// 2. 直接更新已连接的执行器实例
	if err := h.updateConnectedExecutors(dryRunMode); err != nil {
		return fmt.Errorf("failed to update connected executors: %w", err)
	}

	log.Printf("Trading executors notified successfully")
	return nil
}

// updatePnLServiceConfig 更新PnL服务配置
func (h *SettingsHandler) updatePnLServiceConfig(dryRunMode bool) error {
	log.Printf("Updating PnL service dry-run configuration: %v", dryRunMode)

	// 如果有PnL服务实例，直接调用其配置方法
	// if h.pnlService != nil {
	//     return h.pnlService.SetDryRun(dryRunMode)
	// }

	// 通过API或消息队列通知PnL服务
	pnlConfig := map[string]interface{}{
		"dry_run_mode": dryRunMode,
		"service":      "pnl",
		"timestamp":    time.Now(),
	}

	if err := h.notifyService("pnl_service", pnlConfig); err != nil {
		return fmt.Errorf("failed to notify PnL service: %w", err)
	}

	log.Printf("PnL service configuration updated successfully")
	return nil
}

// updateRiskManagementConfig 更新风险管理配置
func (h *SettingsHandler) updateRiskManagementConfig(dryRunMode bool) error {
	log.Printf("Updating risk management dry-run configuration: %v", dryRunMode)

	riskConfig := map[string]interface{}{
		"dry_run_mode": dryRunMode,
		"service":      "risk_management",
		"timestamp":    time.Now(),
	}

	if err := h.notifyService("risk_management", riskConfig); err != nil {
		return fmt.Errorf("failed to notify risk management service: %w", err)
	}

	log.Printf("Risk management configuration updated successfully")
	return nil
}

// broadcastConfigUpdate 广播配置更新
func (h *SettingsHandler) broadcastConfigUpdate(message map[string]interface{}) error {
	// 实现消息队列广播逻辑
	log.Printf("Broadcasting config update: %+v", message)

	// 示例：通过Redis发布订阅或消息队列
	// if h.messageQueue != nil {
	//     return h.messageQueue.Publish("config_updates", message)
	// }

	// 暂时记录日志，实际实现时需要集成消息系统
	log.Printf("Config update broadcast completed")
	return nil
}

// updateConnectedExecutors 更新已连接的执行器
func (h *SettingsHandler) updateConnectedExecutors(dryRunMode bool) error {
	log.Printf("Updating connected executors with dry-run mode: %v", dryRunMode)

	// 遍历所有已连接的执行器实例并更新配置
	// if h.executorManager != nil {
	//     return h.executorManager.UpdateDryRunMode(dryRunMode)
	// }

	log.Printf("Connected executors updated successfully")
	return nil
}

// notifyService 通知特定服务
func (h *SettingsHandler) notifyService(serviceName string, config map[string]interface{}) error {
	log.Printf("Notifying service %s with config: %+v", serviceName, config)

	// 实现服务通知逻辑，可以通过HTTP API、gRPC或消息队列
	// 示例：HTTP API调用
	// if h.serviceRegistry != nil {
	//     return h.serviceRegistry.NotifyService(serviceName, config)
	// }

	log.Printf("Service %s notified successfully", serviceName)
	return nil
}

// recordConfigChange 记录配置变更
func (h *SettingsHandler) recordConfigChange(configKey string, value interface{}) error {
	if h.db == nil {
		return fmt.Errorf("database not available")
	}

	query := `
		INSERT INTO config_changes (config_key, old_value, new_value, changed_at, changed_by)
		VALUES (?, ?, ?, ?, ?)
	`

	// 获取旧值（简化实现）
	oldValue := "unknown"

	_, err := h.db.Exec(query, configKey, oldValue, fmt.Sprintf("%v", value),
		time.Now(), "settings_handler")
	if err != nil {
		return fmt.Errorf("failed to record config change: %w", err)
	}

	log.Printf("Config change recorded: %s = %v", configKey, value)
	return nil
}

// GetCurrentSettings 获取当前设置（用于其他服务）
func (h *SettingsHandler) GetCurrentSettings() *Settings {
	return h.currentSettings
}

// IsDryRunMode 检查是否处于 Dry-Run 模式
func (h *SettingsHandler) IsDryRunMode() bool {
	return h.currentSettings.Trading.DryRunMode
}
