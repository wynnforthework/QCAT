package emergency

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func TestEmergencyStopManager_EmergencyStopAllStrategies(t *testing.T) {
	// 跳过集成测试，除非设置了环境变量
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 创建测试数据库连接
	db, err := sql.Open("postgres", "postgres://postgres:123@localhost:5432/qcat_test?sslmode=disable")
	if err != nil {
		t.Skipf("无法连接到测试数据库: %v", err)
	}
	defer db.Close()

	// 测试数据库连接
	if err := db.Ping(); err != nil {
		t.Skipf("数据库连接失败: %v", err)
	}

	// 创建紧急停止管理器
	esm := NewEmergencyStopManager(db)

	// 测试紧急停止功能
	ctx := context.Background()
	reason := "测试紧急停止"

	err = esm.EmergencyStopAllStrategies(ctx, reason)
	if err != nil {
		t.Errorf("紧急停止失败: %v", err)
	}

	// 验证紧急停止状态
	status := esm.GetEmergencyStopStatus()
	if !status["is_stopped"].(bool) {
		t.Error("紧急停止状态应该为true")
	}

	// 测试重置功能
	time.Sleep(1 * time.Second) // 等待一秒确保时间差异
	err = esm.ResetEmergencyStop(ctx, "测试重置")
	if err != nil {
		t.Errorf("重置紧急停止失败: %v", err)
	}

	// 验证重置后状态
	status = esm.GetEmergencyStopStatus()
	if status["is_stopped"].(bool) {
		t.Error("重置后紧急停止状态应该为false")
	}
}

func TestEmergencyStopManager_GetEmergencyStopStatus(t *testing.T) {
	// 创建一个简单的紧急停止管理器，不依赖数据库
	esm := &EmergencyStopManager{}

	// 测试初始状态
	status := esm.GetEmergencyStopStatus()
	if status["is_stopped"].(bool) {
		t.Error("初始状态应该为false")
	}

	// 手动设置停止状态
	esm.mu.Lock()
	esm.stopped = true
	esm.stopTime = time.Now()
	esm.mu.Unlock()

	// 验证状态
	status = esm.GetEmergencyStopStatus()
	if !status["is_stopped"].(bool) {
		t.Error("设置后状态应该为true")
	}

	if _, exists := status["stop_time"]; !exists {
		t.Error("应该包含停止时间")
	}

	if _, exists := status["duration"]; !exists {
		t.Error("应该包含持续时间")
	}
}

func TestEmergencyStopManager_IsEmergencyStopped(t *testing.T) {
	esm := &EmergencyStopManager{}

	// 测试初始状态
	if esm.IsEmergencyStopped() {
		t.Error("初始状态应该为false")
	}

	// 设置停止状态
	esm.mu.Lock()
	esm.stopped = true
	esm.mu.Unlock()

	// 验证状态
	if !esm.IsEmergencyStopped() {
		t.Error("设置后状态应该为true")
	}
}
