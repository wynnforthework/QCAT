package validation

import (
	"testing"
)

func TestFieldValidator(t *testing.T) {
	validator := NewFieldValidator()
	
	// 测试必填字段
	validator.AddRule("name", "", []string{"required"}, "", false)
	err := validator.Validate()
	if err == nil {
		t.Error("Expected validation error for empty required field")
	}
	
	// 测试有效值
	validator = NewFieldValidator()
	validator.AddRule("name", "test", []string{"required"}, "", false)
	err = validator.Validate()
	if err != nil {
		t.Errorf("Unexpected validation error: %v", err)
	}
}

func TestNumericValidation(t *testing.T) {
	validator := NewFieldValidator()
	
	// 测试数字验证
	validator.AddRule("age", "25", []string{"numeric", "min:18", "max:100"}, "", false)
	err := validator.Validate()
	if err != nil {
		t.Errorf("Unexpected validation error: %v", err)
	}
	
	// 测试超出范围
	validator = NewFieldValidator()
	validator.AddRule("age", "150", []string{"numeric", "min:18", "max:100"}, "", false)
	err = validator.Validate()
	if err == nil {
		t.Error("Expected validation error for value out of range")
	}
}

func TestStringValidation(t *testing.T) {
	validator := NewFieldValidator()
	
	// 测试长度验证
	validator.AddRule("username", "test", []string{"minlen:3", "maxlen:20"}, "", false)
	err := validator.Validate()
	if err != nil {
		t.Errorf("Unexpected validation error: %v", err)
	}
	
	// 测试长度不足
	validator = NewFieldValidator()
	validator.AddRule("username", "ab", []string{"minlen:3"}, "", false)
	err = validator.Validate()
	if err == nil {
		t.Error("Expected validation error for string too short")
	}
}

func TestEmailValidation(t *testing.T) {
	validator := NewFieldValidator()
	
	// 测试有效邮箱
	validator.AddRule("email", "test@example.com", []string{"email"}, "", false)
	err := validator.Validate()
	if err != nil {
		t.Errorf("Unexpected validation error: %v", err)
	}
	
	// 测试无效邮箱
	validator = NewFieldValidator()
	validator.AddRule("email", "invalid-email", []string{"email"}, "", false)
	err = validator.Validate()
	if err == nil {
		t.Error("Expected validation error for invalid email")
	}
}

func TestOptionalFields(t *testing.T) {
	validator := NewFieldValidator()
	
	// 测试可选字段为空
	validator.AddRule("optional_field", "", []string{"email"}, "", true)
	err := validator.Validate()
	if err != nil {
		t.Errorf("Unexpected validation error for optional empty field: %v", err)
	}
	
	// 测试可选字段有值但无效
	validator = NewFieldValidator()
	validator.AddRule("optional_field", "invalid-email", []string{"email"}, "", true)
	err = validator.Validate()
	if err == nil {
		t.Error("Expected validation error for optional field with invalid value")
	}
}

func TestInValidation(t *testing.T) {
	validator := NewFieldValidator()
	
	// 测试有效值
	validator.AddRule("status", "active", []string{"in:active,inactive,pending"}, "", false)
	err := validator.Validate()
	if err != nil {
		t.Errorf("Unexpected validation error: %v", err)
	}
	
	// 测试无效值
	validator = NewFieldValidator()
	validator.AddRule("status", "invalid", []string{"in:active,inactive,pending"}, "", false)
	err = validator.Validate()
	if err == nil {
		t.Error("Expected validation error for value not in allowed list")
	}
}

func TestPositiveNegativeValidation(t *testing.T) {
	validator := NewFieldValidator()
	
	// 测试正数
	validator.AddRule("amount", 100, []string{"positive"}, "", false)
	err := validator.Validate()
	if err != nil {
		t.Errorf("Unexpected validation error: %v", err)
	}
	
	// 测试负数验证失败
	validator = NewFieldValidator()
	validator.AddRule("amount", -100, []string{"positive"}, "", false)
	err = validator.Validate()
	if err == nil {
		t.Error("Expected validation error for negative value when positive required")
	}
	
	// 测试负数
	validator = NewFieldValidator()
	validator.AddRule("loss", -50, []string{"negative"}, "", false)
	err = validator.Validate()
	if err != nil {
		t.Errorf("Unexpected validation error: %v", err)
	}
}

func TestRangeValidation(t *testing.T) {
	validator := NewFieldValidator()
	
	// 测试范围内的值
	validator.AddRule("score", 85, []string{"range:0,100"}, "", false)
	err := validator.Validate()
	if err != nil {
		t.Errorf("Unexpected validation error: %v", err)
	}
	
	// 测试超出范围
	validator = NewFieldValidator()
	validator.AddRule("score", 150, []string{"range:0,100"}, "", false)
	err = validator.Validate()
	if err == nil {
		t.Error("Expected validation error for value out of range")
	}
}

func TestStrategyParamsValidator(t *testing.T) {
	// 测试有效参数
	params := map[string]interface{}{
		"ma_short":      20,
		"ma_long":       50,
		"stop_loss":     0.05,
		"take_profit":   0.1,
		"leverage":      5,
		"position_size": 1000,
	}
	
	err := StrategyParamsValidator(params)
	if err != nil {
		t.Errorf("Unexpected validation error: %v", err)
	}
	
	// 测试无效参数
	params["leverage"] = 150 // 超出最大值
	err = StrategyParamsValidator(params)
	if err == nil {
		t.Error("Expected validation error for invalid leverage")
	}
}

func TestOrderValidator(t *testing.T) {
	// 测试有效订单
	order := map[string]interface{}{
		"symbol":   "BTCUSDT",
		"side":     "BUY",
		"type":     "MARKET",
		"quantity": 1.5,
	}
	
	err := OrderValidator(order)
	if err != nil {
		t.Errorf("Unexpected validation error: %v", err)
	}
	
	// 测试限价单需要价格
	order["type"] = "LIMIT"
	err = OrderValidator(order)
	if err == nil {
		t.Error("Expected validation error for LIMIT order without price")
	}
	
	// 添加价格
	order["price"] = 45000.0
	err = OrderValidator(order)
	if err != nil {
		t.Errorf("Unexpected validation error: %v", err)
	}
}

func TestRiskLimitsValidator(t *testing.T) {
	// 测试有效风险限额
	limits := map[string]interface{}{
		"max_position_size": 100000,
		"max_leverage":      10,
		"max_drawdown":      0.15,
		"max_daily_loss":    5000,
	}
	
	err := RiskLimitsValidator(limits)
	if err != nil {
		t.Errorf("Unexpected validation error: %v", err)
	}
	
	// 测试无效杠杆
	limits["max_leverage"] = 150 // 超出最大值
	err = RiskLimitsValidator(limits)
	if err == nil {
		t.Error("Expected validation error for invalid leverage")
	}
}

// 测试结构体验证
type TestStruct struct {
	Name     string  `validate:"required|minlen:2|maxlen:50"`
	Email    string  `validate:"required|email"`
	Age      int     `validate:"required|min:18|max:100"`
	Score    float64 `validate:"optional|range:0,100"`
	Status   string  `validate:"required|in:active,inactive"`
}

func TestValidateStruct(t *testing.T) {
	// 测试有效结构体
	valid := TestStruct{
		Name:   "John Doe",
		Email:  "john@example.com",
		Age:    25,
		Score:  85.5,
		Status: "active",
	}
	
	err := ValidateStruct(valid)
	if err != nil {
		t.Errorf("Unexpected validation error: %v", err)
	}
	
	// 测试无效结构体
	invalid := TestStruct{
		Name:   "J", // 太短
		Email:  "invalid-email",
		Age:    15, // 太小
		Score:  150, // 超出范围
		Status: "unknown", // 不在允许列表中
	}
	
	err = ValidateStruct(invalid)
	if err == nil {
		t.Error("Expected validation error for invalid struct")
	}
}