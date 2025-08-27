package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Settings 设置结构
type Settings struct {
	Trading TradingSettings `json:"trading"`
	System  SystemSettings  `json:"system"`
}

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

func main() {
	baseURL := "http://localhost:8080/api/v1"

	log.Println("🧪 测试 Dry-Run 设置功能")
	log.Println("=" + fmt.Sprintf("%40s", "="))

	// 1. 获取当前设置
	log.Println("\n1. 获取当前设置...")
	currentSettings, err := getSettings(baseURL)
	if err != nil {
		log.Printf("❌ 获取设置失败: %v", err)
		return
	}
	log.Printf("✅ 当前 Dry-Run 模式: %v", currentSettings.Trading.DryRunMode)
	log.Printf("✅ 当前风险控制: %v", currentSettings.Trading.RiskControl)

	// 2. 启用 Dry-Run 模式
	log.Println("\n2. 启用 Dry-Run 模式...")
	newSettings := *currentSettings
	newSettings.Trading.DryRunMode = true
	newSettings.Trading.RiskControl = true
	newSettings.Trading.MaxPositionRatio = 30
	newSettings.Trading.DefaultStopLoss = 3

	updatedSettings, err := updateSettings(baseURL, &newSettings)
	if err != nil {
		log.Printf("❌ 更新设置失败: %v", err)
		return
	}
	log.Printf("✅ Dry-Run 模式已启用: %v", updatedSettings.Trading.DryRunMode)
	log.Printf("✅ 最大持仓比例: %d%%", updatedSettings.Trading.MaxPositionRatio)
	log.Printf("✅ 默认止损比例: %d%%", updatedSettings.Trading.DefaultStopLoss)

	// 3. 等待一段时间，然后关闭 Dry-Run 模式
	log.Println("\n3. 等待 5 秒后关闭 Dry-Run 模式...")
	time.Sleep(5 * time.Second)

	newSettings.Trading.DryRunMode = false
	updatedSettings, err = updateSettings(baseURL, &newSettings)
	if err != nil {
		log.Printf("❌ 关闭 Dry-Run 模式失败: %v", err)
		return
	}
	log.Printf("✅ Dry-Run 模式已关闭: %v", updatedSettings.Trading.DryRunMode)

	// 4. 测试系统设置
	log.Println("\n4. 测试系统设置...")
	newSettings.System.DebugMode = true
	newSettings.System.LogLevel = "DEBUG"
	newSettings.System.CacheSize = "2GB"

	updatedSettings, err = updateSettings(baseURL, &newSettings)
	if err != nil {
		log.Printf("❌ 更新系统设置失败: %v", err)
		return
	}
	log.Printf("✅ 调试模式: %v", updatedSettings.System.DebugMode)
	log.Printf("✅ 日志级别: %s", updatedSettings.System.LogLevel)
	log.Printf("✅ 缓存大小: %s", updatedSettings.System.CacheSize)

	log.Println("\n🎉 Dry-Run 设置功能测试完成!")
}

// getSettings 获取当前设置
func getSettings(baseURL string) (*Settings, error) {
	resp, err := http.Get(baseURL + "/settings")
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var settings Settings
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	return &settings, nil
}

// updateSettings 更新设置
func updateSettings(baseURL string, settings *Settings) (*Settings, error) {
	jsonData, err := json.Marshal(settings)
	if err != nil {
		return nil, fmt.Errorf("序列化设置失败: %v", err)
	}

	req, err := http.NewRequest("PUT", baseURL+"/settings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var updatedSettings Settings
	if err := json.NewDecoder(resp.Body).Decode(&updatedSettings); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	return &updatedSettings, nil
}