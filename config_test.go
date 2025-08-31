package main

import (
	"fmt"
	"log"
	"time"
)

// 简化的自动启动配置
type SimpleAutoStartConfig struct {
	Enabled               bool          `yaml:"enabled"`
	StartupDelay          time.Duration `yaml:"startup_delay"`
	MaxConcurrentStartups int           `yaml:"max_concurrent_startups"`
	StartupTimeout        time.Duration `yaml:"startup_timeout"`
	MaxRetries            int           `yaml:"max_retries"`
	RetryDelay            time.Duration `yaml:"retry_delay"`
	RetryBackoff          float64       `yaml:"retry_backoff"`
	RequireValidation     bool          `yaml:"require_validation"`
	SkipFailedValidation  bool          `yaml:"skip_failed_validation"`
	MaxAutoStartPerBatch  int           `yaml:"max_auto_start_per_batch"`
	BatchInterval         time.Duration `yaml:"batch_interval"`
}

// 获取默认配置
func getDefaultConfig() *SimpleAutoStartConfig {
	return &SimpleAutoStartConfig{
		Enabled:               true,
		StartupDelay:          30 * time.Second,
		MaxConcurrentStartups: 3,
		StartupTimeout:        2 * time.Minute,
		MaxRetries:            2,
		RetryDelay:            10 * time.Second,
		RetryBackoff:          2.0,
		RequireValidation:     true,
		SkipFailedValidation:  true,
		MaxAutoStartPerBatch:  3,
		BatchInterval:         5 * time.Second,
	}
}

func main() {
	log.Println("🧪 测试自动启动配置...")

	config := getDefaultConfig()
	
	fmt.Printf("📋 自动启动配置:\n")
	fmt.Printf("   启用状态: %v\n", config.Enabled)
	fmt.Printf("   启动延迟: %v\n", config.StartupDelay)
	fmt.Printf("   最大并发启动数: %d\n", config.MaxConcurrentStartups)
	fmt.Printf("   启动超时: %v\n", config.StartupTimeout)
	fmt.Printf("   最大重试次数: %d\n", config.MaxRetries)
	fmt.Printf("   重试延迟: %v\n", config.RetryDelay)
	fmt.Printf("   重试退避倍数: %.1f\n", config.RetryBackoff)
	fmt.Printf("   需要验证: %v\n", config.RequireValidation)
	fmt.Printf("   跳过验证失败: %v\n", config.SkipFailedValidation)
	fmt.Printf("   每批最大启动数: %d\n", config.MaxAutoStartPerBatch)
	fmt.Printf("   批次间隔: %v\n", config.BatchInterval)

	// 模拟策略启动流程
	fmt.Printf("\n🚀 模拟策略启动流程:\n")
	
	strategies := []struct {
		name     string
		priority int
		enabled  bool
	}{
		{"策略A", 10, true},
		{"策略B", 20, true},
		{"策略C", 30, false},
		{"策略D", 15, true},
	}

	fmt.Printf("   发现 %d 个策略\n", len(strategies))
	
	// 过滤启用的策略
	var enabledStrategies []struct {
		name     string
		priority int
		enabled  bool
	}
	
	for _, s := range strategies {
		if s.enabled {
			enabledStrategies = append(enabledStrategies, s)
		}
	}
	
	fmt.Printf("   其中 %d 个策略已启用\n", len(enabledStrategies))
	
	// 按优先级排序（简单排序）
	for i := 0; i < len(enabledStrategies); i++ {
		for j := i + 1; j < len(enabledStrategies); j++ {
			if enabledStrategies[i].priority > enabledStrategies[j].priority {
				enabledStrategies[i], enabledStrategies[j] = enabledStrategies[j], enabledStrategies[i]
			}
		}
	}
	
	fmt.Printf("   按优先级排序后的启动顺序:\n")
	for i, s := range enabledStrategies {
		fmt.Printf("     %d. %s (优先级: %d)\n", i+1, s.name, s.priority)
	}
	
	// 模拟分批启动
	batchSize := config.MaxAutoStartPerBatch
	fmt.Printf("\n   分批启动 (每批 %d 个):\n", batchSize)
	
	for i := 0; i < len(enabledStrategies); i += batchSize {
		end := i + batchSize
		if end > len(enabledStrategies) {
			end = len(enabledStrategies)
		}
		
		batch := enabledStrategies[i:end]
		fmt.Printf("     批次 %d: ", (i/batchSize)+1)
		for j, s := range batch {
			if j > 0 {
				fmt.Printf(", ")
			}
			fmt.Printf("%s", s.name)
		}
		fmt.Printf("\n")
		
		// 模拟批次间隔
		if i+batchSize < len(enabledStrategies) {
			fmt.Printf("     等待 %v 后启动下一批...\n", config.BatchInterval)
		}
	}

	log.Println("\n✅ 配置测试完成")
}
