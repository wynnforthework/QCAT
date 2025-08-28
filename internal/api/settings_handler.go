package api

import (
	"encoding/json"
	"net/http"
	"log"
)

// TradingSettings 交易设置
type TradingSettings struct {
	DryRunMode        bool    `json:"dryRunMode"`
	RiskControl       bool    `json:"riskControl"`
	MaxPositionRatio  int     `json:"maxPositionRatio"`
	DefaultStopLoss   int     `json:"defaultStopLoss"`
}

// SystemSettings 系统设置
type SystemSettings struct {
	LogLevel   string `json:"logLevel"`
	CacheSize  string `json:"cacheSize"`
	DebugMode  bool   `json:"debugMode"`
}

// Settings 完整设置结构
type Settings struct {
	Trading TradingSettings `json:"trading"`
	System  SystemSettings  `json:"system"`
}

// SettingsHandler 设置处理器
type SettingsHandler struct {
	currentSettings *Settings
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
func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if err := json.NewEncoder(w).Encode(h.currentSettings); err != nil {
		log.Printf("Error encoding settings: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Settings retrieved successfully")
}

// UpdateSettings 更新设置
func (h *SettingsHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	var newSettings Settings
	if err := json.NewDecoder(r.Body).Decode(&newSettings); err != nil {
		log.Printf("Error decoding settings: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// 更新设置
	h.currentSettings = &newSettings

	// 应用 Dry-Run 设置到交易系统
	if err := h.applyDryRunSettings(newSettings.Trading.DryRunMode); err != nil {
		log.Printf("Error applying dry-run settings: %v", err)
		http.Error(w, "Failed to apply settings", http.StatusInternalServerError)
		return
	}

	// 返回更新后的设置
	if err := json.NewEncoder(w).Encode(h.currentSettings); err != nil {
		log.Printf("Error encoding updated settings: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Settings updated successfully - DryRun: %v, RiskControl: %v", 
		newSettings.Trading.DryRunMode, newSettings.Trading.RiskControl)
}

// applyDryRunSettings 应用 Dry-Run 设置到交易系统
func (h *SettingsHandler) applyDryRunSettings(dryRunMode bool) error {
	// TODO 集成到实际的交易系统中
	// 例如更新 PnL 服务的 DryRun 配置
	log.Printf("Applying dry-run mode: %v", dryRunMode)
	
	// TODO: 集成到实际的交易执行器
	// 例如：
	// if h.pnlService != nil {
	//     return h.pnlService.SetDryRun(dryRunMode)
	// }
	
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