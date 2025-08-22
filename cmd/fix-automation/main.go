package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token"`
	Error   string `json:"error"`
}

type AutomationStatus struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Category        string    `json:"category"`
	Status          string    `json:"status"`
	Enabled         bool      `json:"enabled"`
	LastExecution   time.Time `json:"lastExecution"`
	NextExecution   time.Time `json:"nextExecution"`
	SuccessRate     float64   `json:"successRate"`
	AvgExecutionTime float64  `json:"avgExecutionTime"`
	ExecutionCount  int       `json:"executionCount"`
	ErrorCount      int       `json:"errorCount"`
	Description     string    `json:"description"`
}

type AutomationResponse struct {
	Success bool               `json:"success"`
	Data    []AutomationStatus `json:"data"`
	Error   string             `json:"error"`
}

type ToggleRequest struct {
	Enabled bool `json:"enabled"`
}

func main() {
	baseURL := getEnv("API_BASE_URL", "http://localhost:8082")
	username := getEnv("ADMIN_USERNAME", "admin")
	password := getEnv("ADMIN_PASSWORD", "admin123")

	fmt.Println("🔧 开始修复自动化系统状态...")

	// 1. 登录获取token
	fmt.Println("📝 正在登录...")
	token, err := login(baseURL, username, password)
	if err != nil {
		log.Fatal("登录失败:", err)
	}
	fmt.Println("✅ 登录成功")

	// 2. 获取当前自动化状态
	fmt.Println("📊 获取自动化任务状态...")
	automations, err := getAutomationStatus(baseURL, token)
	if err != nil {
		log.Fatal("获取自动化状态失败:", err)
	}

	// 3. 分析当前状态
	fmt.Printf("📋 发现 %d 个自动化任务\n", len(automations))
	
	enabledCount := 0
	runningCount := 0
	errorCount := 0
	stoppedCount := 0

	for _, automation := range automations {
		if automation.Enabled {
			enabledCount++
		}
		switch automation.Status {
		case "running":
			runningCount++
		case "error":
			errorCount++
		case "stopped":
			stoppedCount++
		}
	}

	fmt.Printf("📊 当前状态统计:\n")
	fmt.Printf("   - 已启用: %d/%d\n", enabledCount, len(automations))
	fmt.Printf("   - 运行中: %d\n", runningCount)
	fmt.Printf("   - 错误状态: %d\n", errorCount)
	fmt.Printf("   - 已停止: %d\n", stoppedCount)

	// 4. 修复策略：启用关键的自动化任务
	fmt.Println("\n🛠️  开始修复自动化任务...")
	
	// 定义关键任务（优先启用）
	criticalTasks := map[string]bool{
		"1":  true, // 策略自动优化
		"7":  true, // 风险实时监控
		"11": true, // 熔断机制
		"19": true, // 系统健康检查
		"16": true, // 市场数据采集
	}

	fixedCount := 0
	for _, automation := range automations {
		// 只修复关键任务或成功率较高的任务
		shouldEnable := criticalTasks[automation.ID] || 
			(automation.SuccessRate > 60 && automation.Status != "error")

		if !automation.Enabled && shouldEnable {
			fmt.Printf("🔄 启用任务: %s (%s)\n", automation.Name, automation.ID)
			if err := toggleAutomation(baseURL, token, automation.ID, true); err != nil {
				fmt.Printf("❌ 启用失败: %v\n", err)
			} else {
				fmt.Printf("✅ 启用成功\n")
				fixedCount++
			}
			time.Sleep(500 * time.Millisecond) // 避免请求过快
		}
	}

	// 5. 验证修复结果
	fmt.Println("\n📊 验证修复结果...")
	time.Sleep(2 * time.Second) // 等待状态更新

	updatedAutomations, err := getAutomationStatus(baseURL, token)
	if err != nil {
		log.Printf("获取更新状态失败: %v", err)
	} else {
		newEnabledCount := 0
		for _, automation := range updatedAutomations {
			if automation.Enabled {
				newEnabledCount++
			}
		}
		fmt.Printf("✅ 修复完成! 已启用任务数量: %d -> %d\n", enabledCount, newEnabledCount)
		fmt.Printf("🎯 本次修复了 %d 个关键任务\n", fixedCount)
	}

	fmt.Println("\n💡 修复建议:")
	fmt.Println("   1. 检查网络连接，确保能访问Binance API")
	fmt.Println("   2. 监控系统健康分数，保持在0.8以上")
	fmt.Println("   3. 定期检查自动化任务状态")
	fmt.Println("   4. 对于持续失败的任务，检查配置和权限")
}

func login(baseURL, username, password string) (string, error) {
	loginReq := LoginRequest{
		Username: username,
		Password: password,
	}

	jsonData, err := json.Marshal(loginReq)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(baseURL+"/api/v1/auth/login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var loginResp LoginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return "", err
	}

	if !loginResp.Success {
		return "", fmt.Errorf("login failed: %s", loginResp.Error)
	}

	return loginResp.Token, nil
}

func getAutomationStatus(baseURL, token string) ([]AutomationStatus, error) {
	req, err := http.NewRequest("GET", baseURL+"/api/v1/automation/status", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var automationResp AutomationResponse
	if err := json.Unmarshal(body, &automationResp); err != nil {
		return nil, err
	}

	if !automationResp.Success {
		return nil, fmt.Errorf("API error: %s", automationResp.Error)
	}

	return automationResp.Data, nil
}

func toggleAutomation(baseURL, token, automationID string, enabled bool) error {
	toggleReq := ToggleRequest{Enabled: enabled}
	jsonData, err := json.Marshal(toggleReq)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", baseURL+"/api/v1/automation/"+automationID+"/toggle", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
