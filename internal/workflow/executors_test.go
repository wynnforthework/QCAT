package workflow

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"
)

// MockExecutor 模拟执行器（用于测试）
type MockExecutor struct {
	BaseExecutor
	simulateFailure bool
	executionTime   time.Duration
}

// NewMockExecutor 创建模拟执行器
func NewMockExecutor(name string, executionTime time.Duration, simulateFailure bool) *MockExecutor {
	return &MockExecutor{
		BaseExecutor: BaseExecutor{
			name: name,
			resourceRequirements: map[string]interface{}{
				"cpu":    "low",
				"memory": "medium",
				"io":     "low",
			},
		},
		simulateFailure: simulateFailure,
		executionTime:   executionTime,
	}
}

// Execute 执行模拟任务
func (me *MockExecutor) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	functionID := params["function_id"].(int)
	functionName := params["function_name"].(string)

	log.Printf("🔄 [模拟] 开始执行功能 %d: %s", functionID, functionName)

	// 模拟执行时间
	select {
	case <-time.After(me.executionTime):
		// 正常完成
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// 模拟失败
	// 基于功能ID生成确定性的失败模拟
	if me.simulateFailure && (functionID%10 == 0) { // 每10个功能中有1个失败
		return nil, fmt.Errorf("simulated failure for function %d", functionID)
	}

	// 基于功能ID生成确定性的测试数据
	processedItems := 100 + (functionID*50)%900
	successRate := 0.95 + float64(functionID%5)*0.01
	performanceScore := 80.0 + float64(functionID%20)

	result := map[string]interface{}{
		"function_id":    functionID,
		"function_name":  functionName,
		"status":         "completed",
		"execution_time": me.executionTime.String(),
		"timestamp":      time.Now(),
		"test_data": map[string]interface{}{
			"processed_items":   processedItems,
			"success_rate":      successRate,
			"performance_score": performanceScore,
		},
	}

	log.Printf("✅ [模拟] 功能 %d (%s) 执行完成", functionID, functionName)
	return result, nil
}

// TestMockExecutor 测试mock执行器
func TestMockExecutor(t *testing.T) {
	ctx := context.Background()
	
	// 创建mock执行器
	executor := NewMockExecutor("测试执行器", 100*time.Millisecond, false)
	
	// 测试参数
	params := map[string]interface{}{
		"function_id":   1,
		"function_name": "测试功能",
	}
	
	// 执行任务
	result, err := executor.Execute(ctx, params)
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}
	
	// 验证结果
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}
	
	if resultMap["function_id"] != 1 {
		t.Errorf("Expected function_id 1, got %v", resultMap["function_id"])
	}
	
	if resultMap["status"] != "completed" {
		t.Errorf("Expected status completed, got %v", resultMap["status"])
	}
	
	// 测试失败模式
	failureExecutor := NewMockExecutor("失败执行器", 50*time.Millisecond, true)
	failureParams := map[string]interface{}{
		"function_id":   10, // 会触发失败
		"function_name": "失败测试",
	}
	
	_, err = failureExecutor.Execute(ctx, failureParams)
	if err == nil {
		t.Error("Expected error for simulated failure")
	}
	
	// 测试取消
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel() // 立即取消
	
	_, err = executor.Execute(cancelCtx, params)
	if err == nil {
		t.Error("Expected error for cancelled context")
	}
}

// TestMockExecutorResourceRequirements 测试资源需求
func TestMockExecutorResourceRequirements(t *testing.T) {
	executor := NewMockExecutor("资源测试", time.Second, false)
	
	requirements := executor.GetResourceRequirements()
	if requirements["cpu"] != "low" {
		t.Errorf("Expected cpu requirement 'low', got %v", requirements["cpu"])
	}
	
	if requirements["memory"] != "medium" {
		t.Errorf("Expected memory requirement 'medium', got %v", requirements["memory"])
	}
	
	if requirements["io"] != "low" {
		t.Errorf("Expected io requirement 'low', got %v", requirements["io"])
	}
}

// TestMockExecutorName 测试执行器名称
func TestMockExecutorName(t *testing.T) {
	name := "测试执行器名称"
	executor := NewMockExecutor(name, time.Second, false)
	
	if executor.GetName() != name {
		t.Errorf("Expected name '%s', got '%s'", name, executor.GetName())
	}
}
