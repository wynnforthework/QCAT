package optimizer

import (
	"database/sql"
	"time"
)

// 新增：Factory provides factory methods for creating optimizer components
type Factory struct{}

// 新增：NewFactory creates a new optimizer factory
func NewFactory() *Factory {
	return &Factory{}
}

// 新增：CreateTriggerChecker creates a new trigger checker instance
func (f *Factory) CreateTriggerChecker() *TriggerChecker {
	config := &TriggerConfig{
		MinSharpe:     0.5,
		MaxDrawdown:   0.1,
		NoNewHighDays: 30,
		ReturnRank:    0.2,
		CheckInterval: 1 * time.Hour,
	}
	return NewTriggerChecker(config)
}

// 新增：CreateWalkForwardOptimizer creates a new walk-forward optimizer instance
func (f *Factory) CreateWalkForwardOptimizer() *WalkForwardOptimizer {
	config := &WFOConfig{
		InSampleSize:    252, // 一年交易日
		OutSampleSize:   63,  // 一个季度
		MinWinRate:      0.5,
		MinProfitFactor: 1.2,
		Anchored:        false,
	}
	return NewWalkForwardOptimizer(config)
}

// 新增：CreateOverfitDetector creates a new overfit detector instance
func (f *Factory) CreateOverfitDetector() *OverfitDetector {
	config := &OverfitConfig{
		MinSamples:      100,
		ConfidenceLevel: 0.95,
		PBOThreshold:    0.2,
	}
	return NewOverfitDetector(config)
}

// 新增：CreateOrchestrator creates a new orchestrator with all dependencies
func (f *Factory) CreateOrchestrator(db *sql.DB) *Orchestrator {
	triggerChecker := f.CreateTriggerChecker()
	walkForwardOptimizer := f.CreateWalkForwardOptimizer()
	overfitDetector := f.CreateOverfitDetector()

	return NewOrchestrator(triggerChecker, walkForwardOptimizer, overfitDetector, db)
}
