package main

import (
	"fmt"
	"log"
	"time"

	"qcat/internal/strategy/autostart"
)

func main() {
	log.Println("🧪 简单测试自动启动服务...")

	// 创建默认配置
	config := autostart.GetDefaultAutoStartConfig()
	
	// 显示配置信息
	fmt.Printf("📋 自动启动配置:\n")
	fmt.Printf("   启用状态: %v\n", config.Enabled)
	fmt.Printf("   启动延迟: %v\n", config.StartupDelay)
	fmt.Printf("   最大并发启动数: %d\n", config.MaxConcurrentStartups)
	fmt.Printf("   启动超时: %v\n", config.StartupTimeout)
	fmt.Printf("   最大重试次数: %d\n", config.MaxRetries)
	fmt.Printf("   重试延迟: %v\n", config.RetryDelay)
	fmt.Printf("   需要验证: %v\n", config.RequireValidation)
	fmt.Printf("   每批最大启动数: %d\n", config.MaxAutoStartPerBatch)
	fmt.Printf("   批次间隔: %v\n", config.BatchInterval)

	// 创建自动启动服务（不连接数据库）
	service := autostart.NewAutoStartService(nil, config)
	
	// 检查服务状态
	fmt.Printf("\n🔍 服务状态:\n")
	fmt.Printf("   运行中: %v\n", service.IsRunning())
	
	// 获取统计信息
	stats := service.GetStats()
	fmt.Printf("\n📊 统计信息:\n")
	fmt.Printf("   总尝试次数: %d\n", stats.TotalAttempts)
	fmt.Printf("   成功启动: %d\n", stats.SuccessfulStarts)
	fmt.Printf("   失败启动: %d\n", stats.FailedStarts)
	fmt.Printf("   跳过启动: %d\n", stats.SkippedStarts)
	fmt.Printf("   最后运行时间: %v\n", stats.LastRunTime)
	fmt.Printf("   最后运行耗时: %v\n", stats.LastRunDuration)

	log.Println("✅ 简单测试完成")
}
