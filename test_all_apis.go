package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	fmt.Println("🧪 Testing All Unified Strategy APIs...")
	
	// 等待服务器启动
	time.Sleep(1 * time.Second)
	
	baseURL := "http://localhost:8082"
	
	// 测试所有API端点
	tests := []struct {
		name     string
		endpoint string
		expected []string // 期望的响应字段
	}{
		{
			name:     "策略列表API",
			endpoint: "/api/v1/strategy?view=list&page=1&page_size=10",
			expected: []string{"strategies", "total", "summary"},
		},
		{
			name:     "策略池视图API",
			endpoint: "/api/v1/strategy?view=pool",
			expected: []string{"strategies", "summary"},
		},
		{
			name:     "执行监控视图API",
			endpoint: "/api/v1/strategy?view=execution",
			expected: []string{"strategies", "summary"},
		},
		{
			name:     "性能分析视图API",
			endpoint: "/api/v1/strategy?view=performance",
			expected: []string{"strategies", "summary"},
		},
		{
			name:     "策略详情API",
			endpoint: "/api/v1/strategy/strategy_001",
			expected: []string{"id", "name", "lifecycle", "execution", "performance", "pool"},
		},
		{
			name:     "策略池概览API",
			endpoint: "/api/v1/strategy/pool/overview",
			expected: []string{"distribution", "summary"},
		},
		{
			name:     "执行系统概览API",
			endpoint: "/api/v1/strategy/execution/overview",
			expected: []string{"system", "performance"},
		},
		{
			name:     "实时状态API",
			endpoint: "/api/v1/strategy/execution/realtime",
			expected: []string{"activeStrategies", "systemMetrics"},
		},
		{
			name:     "工作流状态API",
			endpoint: "/api/v1/strategy/workflow/status",
			expected: []string{"system", "evolution"},
		},
	}
	
	successCount := 0
	totalCount := len(tests)
	
	for i, test := range tests {
		fmt.Printf("\n%d. Testing %s...\n", i+1, test.name)
		fmt.Printf("   URL: %s%s\n", baseURL, test.endpoint)
		
		if testAPI(baseURL+test.endpoint, test.expected) {
			fmt.Printf("   ✅ PASSED\n")
			successCount++
		} else {
			fmt.Printf("   ❌ FAILED\n")
		}
	}
	
	fmt.Printf("\n🎉 Test Results:\n")
	fmt.Printf("   Total: %d\n", totalCount)
	fmt.Printf("   Passed: %d ✅\n", successCount)
	fmt.Printf("   Failed: %d ❌\n", totalCount-successCount)
	fmt.Printf("   Success Rate: %.1f%%\n", float64(successCount)/float64(totalCount)*100)
	
	if successCount == totalCount {
		fmt.Printf("\n🚀 All APIs are working perfectly!\n")
		fmt.Printf("   The unified strategy management system is ready to use.\n")
	} else {
		fmt.Printf("\n⚠️  Some APIs need attention.\n")
	}
	
	fmt.Printf("\n📋 Frontend Testing:\n")
	fmt.Printf("   1. Start frontend: cd frontend && npm run dev\n")
	fmt.Printf("   2. Test unified page: http://localhost:3000/strategies-unified\n")
	fmt.Printf("   3. Test redirects: http://localhost:3000/strategies\n")
}

func testAPI(url string, expectedFields []string) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	
	resp, err := client.Get(url)
	if err != nil {
		fmt.Printf("   ❌ Request failed: %v\n", err)
		return false
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		fmt.Printf("   ❌ Status code: %d\n", resp.StatusCode)
		return false
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("   ❌ Read body failed: %v\n", err)
		return false
	}
	
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("   ❌ JSON parse failed: %v\n", err)
		return false
	}
	
	// 检查success字段
	if success, ok := response["success"].(bool); !ok || !success {
		fmt.Printf("   ❌ API returned success=false\n")
		return false
	}
	
	// 检查data字段
	data, ok := response["data"].(map[string]interface{})
	if !ok {
		fmt.Printf("   ❌ No data field in response\n")
		return false
	}
	
	// 检查期望的字段
	for _, field := range expectedFields {
		if _, exists := data[field]; !exists {
			fmt.Printf("   ❌ Missing field: %s\n", field)
			return false
		}
	}
	
	fmt.Printf("   ✅ Status: %d, Fields: %v\n", resp.StatusCode, expectedFields)
	return true
}