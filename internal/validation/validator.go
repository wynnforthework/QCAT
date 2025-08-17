package validation

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"qcat/internal/errors"
)

// Validator 验证器接口
type Validator interface {
	Validate(value interface{}) error
}

// ValidationRule 验证规则
type ValidationRule struct {
	Field     string
	Value     interface{}
	Rules     []string
	Message   string
	Optional  bool
}

// FieldValidator 字段验证器
type FieldValidator struct {
	rules map[string][]ValidationRule
}

// NewFieldValidator 创建字段验证器
func NewFieldValidator() *FieldValidator {
	return &FieldValidator{
		rules: make(map[string][]ValidationRule),
	}
}

// AddRule 添加验证规则
func (v *FieldValidator) AddRule(field string, value interface{}, rules []string, message string, optional bool) {
	rule := ValidationRule{
		Field:    field,
		Value:    value,
		Rules:    rules,
		Message:  message,
		Optional: optional,
	}
	v.rules[field] = append(v.rules[field], rule)
}

// Validate 执行验证
func (v *FieldValidator) Validate() error {
	var validationErrors []string

	for field, rules := range v.rules {
		for _, rule := range rules {
			if err := v.validateRule(rule); err != nil {
				if rule.Message != "" {
					validationErrors = append(validationErrors, rule.Message)
				} else {
					validationErrors = append(validationErrors, fmt.Sprintf("Field '%s': %s", field, err.Error()))
				}
			}
		}
	}

	if len(validationErrors) > 0 {
		return errors.NewAppErrorWithDetails(
			errors.ErrCodeInvalidInput,
			"Validation failed",
			strings.Join(validationErrors, "; "),
			nil,
		)
	}

	return nil
}

// validateRule 验证单个规则
func (v *FieldValidator) validateRule(rule ValidationRule) error {
	// 如果字段是可选的且值为空，跳过验证
	if rule.Optional && isEmpty(rule.Value) {
		return nil
	}

	for _, r := range rule.Rules {
		if err := v.applyRule(rule.Value, r); err != nil {
			return err
		}
	}

	return nil
}

// applyRule 应用验证规则
func (v *FieldValidator) applyRule(value interface{}, rule string) error {
	parts := strings.Split(rule, ":")
	ruleName := parts[0]
	var ruleParam string
	if len(parts) > 1 {
		ruleParam = parts[1]
	}

	switch ruleName {
	case "required":
		return v.validateRequired(value)
	case "min":
		return v.validateMin(value, ruleParam)
	case "max":
		return v.validateMax(value, ruleParam)
	case "minlen":
		return v.validateMinLength(value, ruleParam)
	case "maxlen":
		return v.validateMaxLength(value, ruleParam)
	case "email":
		return v.validateEmail(value)
	case "url":
		return v.validateURL(value)
	case "numeric":
		return v.validateNumeric(value)
	case "alpha":
		return v.validateAlpha(value)
	case "alphanumeric":
		return v.validateAlphanumeric(value)
	case "regex":
		return v.validateRegex(value, ruleParam)
	case "in":
		return v.validateIn(value, ruleParam)
	case "date":
		return v.validateDate(value)
	case "datetime":
		return v.validateDateTime(value)
	case "positive":
		return v.validatePositive(value)
	case "negative":
		return v.validateNegative(value)
	case "range":
		return v.validateRange(value, ruleParam)
	default:
		return fmt.Errorf("unknown validation rule: %s", ruleName)
	}
}

// validateRequired 验证必填
func (v *FieldValidator) validateRequired(value interface{}) error {
	if isEmpty(value) {
		return fmt.Errorf("field is required")
	}
	return nil
}

// validateMin 验证最小值
func (v *FieldValidator) validateMin(value interface{}, param string) error {
	minVal, err := strconv.ParseFloat(param, 64)
	if err != nil {
		return fmt.Errorf("invalid min parameter: %s", param)
	}

	val, err := toFloat64(value)
	if err != nil {
		return fmt.Errorf("value is not numeric")
	}

	if val < minVal {
		return fmt.Errorf("value must be at least %g", minVal)
	}
	return nil
}

// validateMax 验证最大值
func (v *FieldValidator) validateMax(value interface{}, param string) error {
	maxVal, err := strconv.ParseFloat(param, 64)
	if err != nil {
		return fmt.Errorf("invalid max parameter: %s", param)
	}

	val, err := toFloat64(value)
	if err != nil {
		return fmt.Errorf("value is not numeric")
	}

	if val > maxVal {
		return fmt.Errorf("value must be at most %g", maxVal)
	}
	return nil
}

// validateMinLength 验证最小长度
func (v *FieldValidator) validateMinLength(value interface{}, param string) error {
	minLen, err := strconv.Atoi(param)
	if err != nil {
		return fmt.Errorf("invalid minlen parameter: %s", param)
	}

	str := toString(value)
	if len(str) < minLen {
		return fmt.Errorf("length must be at least %d characters", minLen)
	}
	return nil
}

// validateMaxLength 验证最大长度
func (v *FieldValidator) validateMaxLength(value interface{}, param string) error {
	maxLen, err := strconv.Atoi(param)
	if err != nil {
		return fmt.Errorf("invalid maxlen parameter: %s", param)
	}

	str := toString(value)
	if len(str) > maxLen {
		return fmt.Errorf("length must be at most %d characters", maxLen)
	}
	return nil
}

// validateEmail 验证邮箱格式
func (v *FieldValidator) validateEmail(value interface{}) error {
	str := toString(value)
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(str) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

// validateURL 验证URL格式
func (v *FieldValidator) validateURL(value interface{}) error {
	str := toString(value)
	urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	if !urlRegex.MatchString(str) {
		return fmt.Errorf("invalid URL format")
	}
	return nil
}

// validateNumeric 验证数字
func (v *FieldValidator) validateNumeric(value interface{}) error {
	_, err := toFloat64(value)
	if err != nil {
		return fmt.Errorf("value must be numeric")
	}
	return nil
}

// validateAlpha 验证字母
func (v *FieldValidator) validateAlpha(value interface{}) error {
	str := toString(value)
	alphaRegex := regexp.MustCompile(`^[a-zA-Z]+$`)
	if !alphaRegex.MatchString(str) {
		return fmt.Errorf("value must contain only letters")
	}
	return nil
}

// validateAlphanumeric 验证字母数字
func (v *FieldValidator) validateAlphanumeric(value interface{}) error {
	str := toString(value)
	alphanumericRegex := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	if !alphanumericRegex.MatchString(str) {
		return fmt.Errorf("value must contain only letters and numbers")
	}
	return nil
}

// validateRegex 验证正则表达式
func (v *FieldValidator) validateRegex(value interface{}, pattern string) error {
	str := toString(value)
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %s", pattern)
	}
	if !regex.MatchString(str) {
		return fmt.Errorf("value does not match pattern: %s", pattern)
	}
	return nil
}

// validateIn 验证值在指定范围内
func (v *FieldValidator) validateIn(value interface{}, param string) error {
	str := toString(value)
	values := strings.Split(param, ",")
	for _, val := range values {
		if strings.TrimSpace(val) == str {
			return nil
		}
	}
	return fmt.Errorf("value must be one of: %s", param)
}

// validateDate 验证日期格式
func (v *FieldValidator) validateDate(value interface{}) error {
	str := toString(value)
	_, err := time.Parse("2006-01-02", str)
	if err != nil {
		return fmt.Errorf("invalid date format, expected YYYY-MM-DD")
	}
	return nil
}

// validateDateTime 验证日期时间格式
func (v *FieldValidator) validateDateTime(value interface{}) error {
	str := toString(value)
	_, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return fmt.Errorf("invalid datetime format, expected RFC3339")
	}
	return nil
}

// validatePositive 验证正数
func (v *FieldValidator) validatePositive(value interface{}) error {
	val, err := toFloat64(value)
	if err != nil {
		return fmt.Errorf("value is not numeric")
	}
	if val <= 0 {
		return fmt.Errorf("value must be positive")
	}
	return nil
}

// validateNegative 验证负数
func (v *FieldValidator) validateNegative(value interface{}) error {
	val, err := toFloat64(value)
	if err != nil {
		return fmt.Errorf("value is not numeric")
	}
	if val >= 0 {
		return fmt.Errorf("value must be negative")
	}
	return nil
}

// validateRange 验证范围
func (v *FieldValidator) validateRange(value interface{}, param string) error {
	parts := strings.Split(param, ",")
	if len(parts) != 2 {
		return fmt.Errorf("invalid range parameter: %s", param)
	}

	minVal, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return fmt.Errorf("invalid range min value: %s", parts[0])
	}

	maxVal, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return fmt.Errorf("invalid range max value: %s", parts[1])
	}

	val, err := toFloat64(value)
	if err != nil {
		return fmt.Errorf("value is not numeric")
	}

	if val < minVal || val > maxVal {
		return fmt.Errorf("value must be between %g and %g", minVal, maxVal)
	}
	return nil
}

// 辅助函数

// isEmpty 检查值是否为空
func isEmpty(value interface{}) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Slice, reflect.Map, reflect.Array:
		return v.Len() == 0
	case reflect.Ptr:
		return v.IsNil()
	default:
		return false
	}
}

// toString 转换为字符串
func toString(value interface{}) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%v", value)
}

// toFloat64 转换为float64
func toFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", value)
	}
}

// ValidateStruct 验证结构体
func ValidateStruct(s interface{}) error {
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct, got %T", s)
	}

	validator := NewFieldValidator()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// 获取验证标签
		tag := field.Tag.Get("validate")
		if tag == "" {
			continue
		}

		// 解析验证规则
		rules := strings.Split(tag, "|")
		optional := false

		// 检查是否为可选字段
		for j, rule := range rules {
			if rule == "optional" {
				optional = true
				// 移除optional规则
				rules = append(rules[:j], rules[j+1:]...)
				break
			}
		}

		// 添加验证规则
		validator.AddRule(
			field.Name,
			fieldValue.Interface(),
			rules,
			"",
			optional,
		)
	}

	return validator.Validate()
}

// 预定义的验证器

// StrategyParamsValidator 策略参数验证器
func StrategyParamsValidator(params map[string]interface{}) error {
	validator := NewFieldValidator()

	for key, value := range params {
		switch key {
		case "ma_short", "ma_long":
			validator.AddRule(key, value, []string{"required", "numeric", "positive", "min:1", "max:200"}, "", false)
		case "stop_loss", "take_profit":
			validator.AddRule(key, value, []string{"required", "numeric", "positive", "min:0.001", "max:1"}, "", false)
		case "leverage":
			validator.AddRule(key, value, []string{"required", "numeric", "positive", "min:1", "max:100"}, "", false)
		case "position_size":
			validator.AddRule(key, value, []string{"required", "numeric", "positive", "min:0.001"}, "", false)
		}
	}

	return validator.Validate()
}

// OrderValidator 订单验证器
func OrderValidator(order map[string]interface{}) error {
	validator := NewFieldValidator()

	// 必填字段
	validator.AddRule("symbol", order["symbol"], []string{"required", "alphanumeric"}, "", false)
	validator.AddRule("side", order["side"], []string{"required", "in:BUY,SELL"}, "", false)
	validator.AddRule("type", order["type"], []string{"required", "in:MARKET,LIMIT,STOP"}, "", false)
	validator.AddRule("quantity", order["quantity"], []string{"required", "numeric", "positive"}, "", false)

	// 条件字段
	if order["type"] == "LIMIT" || order["type"] == "STOP" {
		validator.AddRule("price", order["price"], []string{"required", "numeric", "positive"}, "", false)
	}

	return validator.Validate()
}

// RiskLimitsValidator 风险限额验证器
func RiskLimitsValidator(limits map[string]interface{}) error {
	validator := NewFieldValidator()

	validator.AddRule("max_position_size", limits["max_position_size"], []string{"required", "numeric", "positive"}, "", false)
	validator.AddRule("max_leverage", limits["max_leverage"], []string{"required", "numeric", "positive", "min:1", "max:100"}, "", false)
	validator.AddRule("max_drawdown", limits["max_drawdown"], []string{"required", "numeric", "positive", "min:0.01", "max:1"}, "", false)
	validator.AddRule("max_daily_loss", limits["max_daily_loss"], []string{"required", "numeric", "positive"}, "", false)

	return validator.Validate()
}