package bridge

import (
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"qcat/internal/automation/executor"
)

// Start 启动响应工作器
func (rw *ResponseWorker) Start(wg *sync.WaitGroup) {
	defer wg.Done()

	rw.mu.Lock()
	rw.isRunning = true
	rw.mu.Unlock()

	log.Printf("Response worker %d started", rw.id)

	for {
		select {
		case <-rw.stopCh:
			rw.mu.Lock()
			rw.isRunning = false
			rw.mu.Unlock()
			log.Printf("Response worker %d stopped", rw.id)
			return
		case event := <-rw.eventCh:
			rw.processEvent(event)
		}
	}
}

// Stop 停止响应工作器
func (rw *ResponseWorker) Stop() {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	if rw.isRunning {
		close(rw.stopCh)
	}
}

// processEvent 处理事件
func (rw *ResponseWorker) processEvent(event *MonitorEvent) {
	startTime := time.Now()
	log.Printf("Worker %d processing event: %s (%s)", rw.id, event.Type, event.ID)

	// 更新统计
	rw.bridge.stats.mu.Lock()
	rw.bridge.stats.ProcessedEvents++
	rw.bridge.stats.mu.Unlock()

	// 查找匹配的响应规则
	matchedRules := rw.findMatchingRules(event)
	if len(matchedRules) == 0 {
		log.Printf("No matching rules for event: %s", event.ID)
		return
	}

	// 按优先级排序并执行
	for _, rule := range matchedRules {
		if rw.shouldTriggerRule(rule) {
			rw.executeRule(event, rule)
		}
	}

	// 记录处理延迟
	duration := time.Since(startTime)
	rw.bridge.stats.mu.Lock()
	rw.bridge.stats.AverageLatency = duration
	rw.bridge.stats.mu.Unlock()

	log.Printf("Worker %d completed event processing: %s (duration: %v)",
		rw.id, event.ID, duration)
}

// findMatchingRules 查找匹配的规则
func (rw *ResponseWorker) findMatchingRules(event *MonitorEvent) []*ResponseRule {
	var matchedRules []*ResponseRule

	rw.bridge.mu.RLock()
	defer rw.bridge.mu.RUnlock()

	for _, rule := range rw.bridge.responseRules {
		if !rule.Enabled {
			continue
		}

		// 检查事件类型
		if rule.EventType != event.Type {
			continue
		}

		// 检查条件
		if rw.evaluateConditions(event, rule.Conditions) {
			matchedRules = append(matchedRules, rule)
		}
	}

	// 按优先级排序（优先级数字越小越高）
	for i := 0; i < len(matchedRules)-1; i++ {
		for j := i + 1; j < len(matchedRules); j++ {
			if matchedRules[i].Priority > matchedRules[j].Priority {
				matchedRules[i], matchedRules[j] = matchedRules[j], matchedRules[i]
			}
		}
	}

	return matchedRules
}

// evaluateConditions 评估条件
func (rw *ResponseWorker) evaluateConditions(event *MonitorEvent, conditions []RuleCondition) bool {
	for _, condition := range conditions {
		if !rw.evaluateCondition(event, condition) {
			return false
		}
	}
	return true
}

// evaluateCondition 评估单个条件
func (rw *ResponseWorker) evaluateCondition(event *MonitorEvent, condition RuleCondition) bool {
	// 获取字段值
	fieldValue := rw.getFieldValue(event, condition.Field)
	if fieldValue == nil {
		return false
	}

	// 执行比较
	return rw.compareValues(fieldValue, condition.Operator, condition.Value)
}

// getFieldValue 获取字段值
func (rw *ResponseWorker) getFieldValue(event *MonitorEvent, field string) interface{} {
	switch field {
	case "type":
		return string(event.Type)
	case "severity":
		return string(event.Severity)
	case "source":
		return event.Source
	case "message":
		return event.Message
	default:
		// 从metadata中获取
		if strings.HasPrefix(field, "metadata.") {
			metadataKey := strings.TrimPrefix(field, "metadata.")
			return event.Metadata[metadataKey]
		}
		// 直接从metadata获取
		return event.Metadata[field]
	}
}

// compareValues 比较值
func (rw *ResponseWorker) compareValues(fieldValue interface{}, operator string, expectedValue interface{}) bool {
	switch operator {
	case "eq":
		return reflect.DeepEqual(fieldValue, expectedValue)
	case "ne":
		return !reflect.DeepEqual(fieldValue, expectedValue)
	case "gt":
		return rw.compareNumbers(fieldValue, expectedValue, ">")
	case "lt":
		return rw.compareNumbers(fieldValue, expectedValue, "<")
	case "gte":
		return rw.compareNumbers(fieldValue, expectedValue, ">=")
	case "lte":
		return rw.compareNumbers(fieldValue, expectedValue, "<=")
	case "contains":
		fieldStr := fmt.Sprintf("%v", fieldValue)
		expectedStr := fmt.Sprintf("%v", expectedValue)
		return strings.Contains(fieldStr, expectedStr)
	default:
		return false
	}
}

// compareNumbers 比较数字
func (rw *ResponseWorker) compareNumbers(fieldValue, expectedValue interface{}, operator string) bool {
	fieldFloat, err1 := rw.toFloat64(fieldValue)
	expectedFloat, err2 := rw.toFloat64(expectedValue)

	if err1 != nil || err2 != nil {
		return false
	}

	switch operator {
	case ">":
		return fieldFloat > expectedFloat
	case "<":
		return fieldFloat < expectedFloat
	case ">=":
		return fieldFloat >= expectedFloat
	case "<=":
		return fieldFloat <= expectedFloat
	default:
		return false
	}
}

// toFloat64 转换为float64
func (rw *ResponseWorker) toFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", value)
	}
}

// shouldTriggerRule 检查是否应该触发规则
func (rw *ResponseWorker) shouldTriggerRule(rule *ResponseRule) bool {
	// 检查冷却期
	if time.Since(rule.LastTrigger) < rule.Cooldown {
		log.Printf("Rule %s is in cooldown period", rule.ID)
		return false
	}

	return true
}

// executeRule 执行规则
func (rw *ResponseWorker) executeRule(event *MonitorEvent, rule *ResponseRule) {
	log.Printf("Executing rule: %s for event: %s", rule.Name, event.ID)

	// 更新触发时间
	rule.LastTrigger = time.Now()

	// 更新统计
	rw.bridge.stats.mu.Lock()
	rw.bridge.stats.TriggeredRules++
	rw.bridge.stats.mu.Unlock()

	// 执行所有动作
	for _, action := range rule.Actions {
		if err := rw.executeAction(event, action); err != nil {
			log.Printf("Failed to execute action %s: %v", action.Action, err)
			rw.bridge.stats.mu.Lock()
			rw.bridge.stats.FailedActions++
			rw.bridge.stats.mu.Unlock()
		} else {
			log.Printf("Successfully executed action: %s", action.Action)
			rw.bridge.stats.mu.Lock()
			rw.bridge.stats.ExecutedActions++
			rw.bridge.stats.mu.Unlock()
		}
	}
}

// executeAction 执行动作
func (rw *ResponseWorker) executeAction(event *MonitorEvent, action ResponseAction) error {
	// 处理参数模板
	parameters := rw.processParameters(event, action.Parameters)

	// 创建执行动作
	execAction := &executor.ExecutionAction{
		Type:       executor.ActionType(action.Type),
		Action:     action.Action,
		Priority:   1, // 响应动作优先级较高
		Parameters: parameters,
		Timeout:    action.Timeout,
		MaxRetries: action.MaxRetries,
	}

	// 通过执行器执行
	return rw.bridge.executor.ExecuteAction(execAction)
}

// processParameters 处理参数模板
func (rw *ResponseWorker) processParameters(event *MonitorEvent, parameters map[string]interface{}) map[string]interface{} {
	processed := make(map[string]interface{})

	for key, value := range parameters {
		if strValue, ok := value.(string); ok {
			// 处理模板变量
			processed[key] = rw.processTemplate(event, strValue)
		} else {
			processed[key] = value
		}
	}

	return processed
}

// processTemplate 处理模板
func (rw *ResponseWorker) processTemplate(event *MonitorEvent, template string) string {
	result := template

	// 替换事件字段
	result = strings.ReplaceAll(result, "{{event.id}}", event.ID)
	result = strings.ReplaceAll(result, "{{event.type}}", string(event.Type))
	result = strings.ReplaceAll(result, "{{event.severity}}", string(event.Severity))
	result = strings.ReplaceAll(result, "{{event.source}}", event.Source)
	result = strings.ReplaceAll(result, "{{event.message}}", event.Message)

	// 替换metadata字段
	for key, value := range event.Metadata {
		placeholder := fmt.Sprintf("{{metadata.%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}

	return result
}
