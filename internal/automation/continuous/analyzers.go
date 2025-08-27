package continuous

import (
	"context"
	"fmt"
	"log"
	"sync"

	"qcat/internal/config"
	"qcat/internal/database"
)

// MarketAnalyzer 市场分析器
type MarketAnalyzer struct {
	config    *config.Config
	db        *database.DB
	optConfig *MarketAnalysisConfig
	running   bool
	mutex     sync.RWMutex
}

// NewMarketAnalyzer 创建市场分析器
func NewMarketAnalyzer(config *config.Config, db *database.DB, optConfig *MarketAnalysisConfig) (*MarketAnalyzer, error) {
	return &MarketAnalyzer{
		config:    config,
		db:        db,
		optConfig: optConfig,
		running:   false,
	}, nil
}

// Start 启动市场分析器
func (ma *MarketAnalyzer) Start(ctx context.Context) error {
	ma.mutex.Lock()
	defer ma.mutex.Unlock()
	
	if ma.running {
		return fmt.Errorf("市场分析器已经在运行")
	}
	
	ma.running = true
	log.Printf("✅ 市场分析器已启动")
	return nil
}

// Stop 停止市场分析器
func (ma *MarketAnalyzer) Stop() {
	ma.mutex.Lock()
	defer ma.mutex.Unlock()
	
	if !ma.running {
		return
	}
	
	ma.running = false
	log.Printf("✅ 市场分析器已停止")
}

// UpdateConfig 更新配置
func (ma *MarketAnalyzer) UpdateConfig(config *MarketAnalysisConfig) {
	ma.mutex.Lock()
	defer ma.mutex.Unlock()
	
	ma.optConfig = config
}

// PerformanceTracker 性能跟踪器
type PerformanceTracker struct {
	config    *config.Config
	db        *database.DB
	optConfig *PerformanceTrackingConfig
	running   bool
	mutex     sync.RWMutex
}

// NewPerformanceTracker 创建性能跟踪器
func NewPerformanceTracker(config *config.Config, db *database.DB, optConfig *PerformanceTrackingConfig) (*PerformanceTracker, error) {
	return &PerformanceTracker{
		config:    config,
		db:        db,
		optConfig: optConfig,
		running:   false,
	}, nil
}

// Start 启动性能跟踪器
func (pt *PerformanceTracker) Start(ctx context.Context) error {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()
	
	if pt.running {
		return fmt.Errorf("性能跟踪器已经在运行")
	}
	
	pt.running = true
	log.Printf("✅ 性能跟踪器已启动")
	return nil
}

// Stop 停止性能跟踪器
func (pt *PerformanceTracker) Stop() {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()
	
	if !pt.running {
		return
	}
	
	pt.running = false
	log.Printf("✅ 性能跟踪器已停止")
}

// UpdateConfig 更新配置
func (pt *PerformanceTracker) UpdateConfig(config *PerformanceTrackingConfig) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()
	
	pt.optConfig = config
}

// ContinuousScheduler 持续调度器
type ContinuousScheduler struct {
	config  *OptimizationConfig
	running bool
	mutex   sync.RWMutex
}

// NewContinuousScheduler 创建持续调度器
func NewContinuousScheduler(config *OptimizationConfig) (*ContinuousScheduler, error) {
	return &ContinuousScheduler{
		config:  config,
		running: false,
	}, nil
}

// Start 启动调度器
func (cs *ContinuousScheduler) Start(ctx context.Context) error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	
	if cs.running {
		return fmt.Errorf("调度器已经在运行")
	}
	
	cs.running = true
	log.Printf("✅ 持续调度器已启动")
	return nil
}

// Stop 停止调度器
func (cs *ContinuousScheduler) Stop() {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	
	if !cs.running {
		return
	}
	
	cs.running = false
	log.Printf("✅ 持续调度器已停止")
}
