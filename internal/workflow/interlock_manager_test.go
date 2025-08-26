package workflow

import (
	"testing"
	"time"
)

// TestInterlockManager_BasicMutex 测试基本互斥锁功能
func TestInterlockManager_BasicMutex(t *testing.T) {
	im := NewInterlockManager()
	defer im.Stop()
	
	// 添加互斥规则
	rule := &InterlockRule{
		ID:            "test_mutex",
		Name:          "测试互斥锁",
		Type:          InterlockTypeMutex,
		FunctionIDs:   []int{1, 6}, // 策略参数优化 和 周期性策略优化
		MaxConcurrent: 1,
		Priority:      10,
		Timeout:       30 * time.Second,
	}
	
	err := im.AddRule(rule)
	if err != nil {
		t.Fatalf("添加规则失败: %v", err)
	}
	
	// 第一个请求应该立即获得授权
	grant1, err := im.RequestLock(1, "test_mutex", 5*time.Second)
	if err != nil {
		t.Fatalf("第一个请求失败: %v", err)
	}
	if grant1 == nil {
		t.Fatal("第一个请求应该立即获得授权")
	}
	
	// 第二个请求应该被阻塞
	done := make(chan bool)
	var grant2 *InterlockGrant
	var err2 error
	
	go func() {
		grant2, err2 = im.RequestLock(6, "test_mutex", 2*time.Second)
		done <- true
	}()
	
	// 等待一段时间确保第二个请求被阻塞
	time.Sleep(500 * time.Millisecond)
	
	select {
	case <-done:
		t.Fatal("第二个请求不应该立即完成")
	default:
		// 正常，第二个请求被阻塞
	}
	
	// 释放第一个锁
	err = im.ReleaseLock(grant1.ID)
	if err != nil {
		t.Fatalf("释放锁失败: %v", err)
	}
	
	// 等待第二个请求完成
	<-done
	
	if err2 != nil {
		t.Fatalf("第二个请求失败: %v", err2)
	}
	if grant2 == nil {
		t.Fatal("第二个请求应该获得授权")
	}
	
	// 清理
	im.ReleaseLock(grant2.ID)
}

// TestInterlockManager_Semaphore 测试信号量功能
func TestInterlockManager_Semaphore(t *testing.T) {
	im := NewInterlockManager()
	defer im.Stop()
	
	// 添加信号量规则
	rule := &InterlockRule{
		ID:            "test_semaphore",
		Name:          "测试信号量",
		Type:          InterlockTypeSemaphore,
		FunctionIDs:   []int{1, 6, 24, 25}, // CPU密集型任务
		MaxConcurrent: 2,
		Priority:      8,
		Timeout:       30 * time.Second,
	}
	
	err := im.AddRule(rule)
	if err != nil {
		t.Fatalf("添加规则失败: %v", err)
	}
	
	// 前两个请求应该立即获得授权
	grant1, err := im.RequestLock(1, "test_semaphore", 5*time.Second)
	if err != nil || grant1 == nil {
		t.Fatalf("第一个请求失败: %v", err)
	}
	
	grant2, err := im.RequestLock(6, "test_semaphore", 5*time.Second)
	if err != nil || grant2 == nil {
		t.Fatalf("第二个请求失败: %v", err)
	}
	
	// 第三个请求应该被阻塞
	done := make(chan bool)
	var grant3 *InterlockGrant
	var err3 error
	
	go func() {
		grant3, err3 = im.RequestLock(24, "test_semaphore", 2*time.Second)
		done <- true
	}()
	
	// 等待一段时间确保第三个请求被阻塞
	time.Sleep(500 * time.Millisecond)
	
	select {
	case <-done:
		t.Fatal("第三个请求不应该立即完成")
	default:
		// 正常，第三个请求被阻塞
	}
	
	// 释放第一个锁
	err = im.ReleaseLock(grant1.ID)
	if err != nil {
		t.Fatalf("释放锁失败: %v", err)
	}
	
	// 等待第三个请求完成
	<-done
	
	if err3 != nil {
		t.Fatalf("第三个请求失败: %v", err3)
	}
	if grant3 == nil {
		t.Fatal("第三个请求应该获得授权")
	}
	
	// 清理
	im.ReleaseLock(grant2.ID)
	im.ReleaseLock(grant3.ID)
}

// TestInterlockManager_Timeout 测试超时功能
func TestInterlockManager_Timeout(t *testing.T) {
	im := NewInterlockManager()
	defer im.Stop()
	
	// 添加互斥规则
	rule := &InterlockRule{
		ID:            "test_timeout",
		Name:          "测试超时",
		Type:          InterlockTypeMutex,
		FunctionIDs:   []int{4, 12}, // 智能建仓 和 异常行情应对
		MaxConcurrent: 1,
		Priority:      10,
		Timeout:       30 * time.Second,
	}
	
	err := im.AddRule(rule)
	if err != nil {
		t.Fatalf("添加规则失败: %v", err)
	}
	
	// 第一个请求获得授权
	grant1, err := im.RequestLock(4, "test_timeout", 5*time.Second)
	if err != nil || grant1 == nil {
		t.Fatalf("第一个请求失败: %v", err)
	}
	
	// 第二个请求应该超时
	start := time.Now()
	grant2, err := im.RequestLock(12, "test_timeout", 1*time.Second)
	duration := time.Since(start)
	
	if err == nil {
		t.Fatal("第二个请求应该超时失败")
	}
	if grant2 != nil {
		t.Fatal("超时请求不应该获得授权")
	}
	
	// 检查超时时间是否合理
	if duration < 900*time.Millisecond || duration > 1500*time.Millisecond {
		t.Fatalf("超时时间不合理: %v", duration)
	}
	
	// 清理
	im.ReleaseLock(grant1.ID)
}

// TestInterlockManager_Priority 测试优先级功能
func TestInterlockManager_Priority(t *testing.T) {
	im := NewInterlockManager()
	defer im.Stop()
	
	// 添加规则
	rule := &InterlockRule{
		ID:            "test_priority",
		Name:          "测试优先级",
		Type:          InterlockTypeMutex,
		FunctionIDs:   []int{11, 12}, // 利润最大化 和 异常行情应对
		MaxConcurrent: 1,
		Priority:      10,
		Timeout:       30 * time.Second,
	}
	
	err := im.AddRule(rule)
	if err != nil {
		t.Fatalf("添加规则失败: %v", err)
	}
	
	// 第一个请求获得授权
	grant1, err := im.RequestLock(11, "test_priority", 5*time.Second)
	if err != nil || grant1 == nil {
		t.Fatalf("第一个请求失败: %v", err)
	}
	
	// 启动多个等待请求
	done1 := make(chan *InterlockGrant)
	done2 := make(chan *InterlockGrant)
	
	go func() {
		grant, _ := im.RequestLock(12, "test_priority", 5*time.Second)
		done1 <- grant
	}()
	
	go func() {
		grant, _ := im.RequestLock(11, "test_priority", 5*time.Second)
		done2 <- grant
	}()
	
	// 等待请求进入队列
	time.Sleep(100 * time.Millisecond)
	
	// 检查等待队列
	queue := im.GetWaitQueue()
	if len(queue) != 2 {
		t.Fatalf("等待队列长度应该为2，实际为: %d", len(queue))
	}
	
	// 释放第一个锁
	im.ReleaseLock(grant1.ID)
	
	// 等待其中一个请求完成
	var nextGrant *InterlockGrant
	select {
	case nextGrant = <-done1:
	case nextGrant = <-done2:
	case <-time.After(2 * time.Second):
		t.Fatal("等待请求超时")
	}
	
	if nextGrant == nil {
		t.Fatal("下一个请求应该获得授权")
	}
	
	// 清理
	im.ReleaseLock(nextGrant.ID)
}

// TestInterlockManager_Stats 测试统计功能
func TestInterlockManager_Stats(t *testing.T) {
	im := NewInterlockManager()
	defer im.Stop()
	
	// 添加规则
	rule := &InterlockRule{
		ID:            "test_stats",
		Name:          "测试统计",
		Type:          InterlockTypeMutex,
		FunctionIDs:   []int{1, 2},
		MaxConcurrent: 1,
		Priority:      5,
		Timeout:       30 * time.Second,
	}
	
	err := im.AddRule(rule)
	if err != nil {
		t.Fatalf("添加规则失败: %v", err)
	}
	
	// 初始统计应该为0
	stats := im.GetStats()
	if stats.TotalRequests != 0 {
		t.Fatalf("初始总请求数应该为0，实际为: %d", stats.TotalRequests)
	}
	
	// 发送一个请求
	grant, err := im.RequestLock(1, "test_stats", 5*time.Second)
	if err != nil || grant == nil {
		t.Fatalf("请求失败: %v", err)
	}
	
	// 检查统计
	stats = im.GetStats()
	if stats.TotalRequests != 1 {
		t.Fatalf("总请求数应该为1，实际为: %d", stats.TotalRequests)
	}
	if stats.GrantedRequests != 1 {
		t.Fatalf("授权请求数应该为1，实际为: %d", stats.GrantedRequests)
	}
	if stats.ActiveGrants != 1 {
		t.Fatalf("活跃授权数应该为1，实际为: %d", stats.ActiveGrants)
	}
	
	// 释放锁
	im.ReleaseLock(grant.ID)
	
	// 检查统计更新
	stats = im.GetStats()
	if stats.ActiveGrants != 0 {
		t.Fatalf("释放后活跃授权数应该为0，实际为: %d", stats.ActiveGrants)
	}
}

// TestInterlockManager_RuleManagement 测试规则管理
func TestInterlockManager_RuleManagement(t *testing.T) {
	im := NewInterlockManager()
	defer im.Stop()
	
	// 添加规则
	rule := &InterlockRule{
		ID:            "test_rule_mgmt",
		Name:          "测试规则管理",
		Type:          InterlockTypeMutex,
		FunctionIDs:   []int{1, 2},
		MaxConcurrent: 1,
		Priority:      5,
		Timeout:       30 * time.Second,
	}
	
	err := im.AddRule(rule)
	if err != nil {
		t.Fatalf("添加规则失败: %v", err)
	}
	
	// 检查规则是否存在
	rules := im.GetRules()
	if len(rules) != 1 {
		t.Fatalf("规则数量应该为1，实际为: %d", len(rules))
	}
	
	retrievedRule, exists := rules["test_rule_mgmt"]
	if !exists {
		t.Fatal("规则应该存在")
	}
	if retrievedRule.Name != "测试规则管理" {
		t.Fatalf("规则名称不匹配: %s", retrievedRule.Name)
	}
	
	// 更新规则
	updates := map[string]interface{}{
		"name":           "更新后的规则名称",
		"max_concurrent": 2,
		"priority":       8,
	}
	
	err = im.UpdateRule("test_rule_mgmt", updates)
	if err != nil {
		t.Fatalf("更新规则失败: %v", err)
	}
	
	// 检查更新结果
	rules = im.GetRules()
	updatedRule := rules["test_rule_mgmt"]
	if updatedRule.Name != "更新后的规则名称" {
		t.Fatalf("规则名称未更新: %s", updatedRule.Name)
	}
	if updatedRule.MaxConcurrent != 2 {
		t.Fatalf("最大并发数未更新: %d", updatedRule.MaxConcurrent)
	}
	if updatedRule.Priority != 8 {
		t.Fatalf("优先级未更新: %d", updatedRule.Priority)
	}
	
	// 禁用规则
	err = im.DisableRule("test_rule_mgmt")
	if err != nil {
		t.Fatalf("禁用规则失败: %v", err)
	}
	
	rules = im.GetRules()
	disabledRule := rules["test_rule_mgmt"]
	if disabledRule.Status != InterlockStatusBlocked {
		t.Fatalf("规则状态应该为blocked，实际为: %s", disabledRule.Status)
	}
	
	// 启用规则
	err = im.EnableRule("test_rule_mgmt")
	if err != nil {
		t.Fatalf("启用规则失败: %v", err)
	}
	
	rules = im.GetRules()
	enabledRule := rules["test_rule_mgmt"]
	if enabledRule.Status != InterlockStatusActive {
		t.Fatalf("规则状态应该为active，实际为: %s", enabledRule.Status)
	}
	
	// 移除规则
	err = im.RemoveRule("test_rule_mgmt")
	if err != nil {
		t.Fatalf("移除规则失败: %v", err)
	}
	
	rules = im.GetRules()
	if len(rules) != 0 {
		t.Fatalf("移除后规则数量应该为0，实际为: %d", len(rules))
	}
}
