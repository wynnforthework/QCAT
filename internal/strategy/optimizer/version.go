package optimizer

import (
	"context"
	"fmt"
	"time"

	"qcat/internal/config"
	"qcat/internal/strategy/sdk"
)

// StrategyVersion represents a strategy version
type StrategyVersion struct {
	ID          string
	StrategyID  string
	Version     int
	Status      VersionStatus
	Params      map[string]float64
	Performance *sdk.StrategyMetrics
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// VersionStatus represents the status of a strategy version
type VersionStatus string

const (
	VersionStatusDraft    VersionStatus = "draft"
	VersionStatusApproved VersionStatus = "approved"
	VersionStatusCanary   VersionStatus = "canary"
	VersionStatusActive   VersionStatus = "active"
	VersionStatusDisabled VersionStatus = "disabled"
)

// VersionManager manages strategy versions
type VersionManager struct {
	versions map[string]*StrategyVersion
}

// NewVersionManager creates a new version manager
func NewVersionManager() *VersionManager {
	return &VersionManager{
		versions: make(map[string]*StrategyVersion),
	}
}

// CreateVersion creates a new strategy version
func (m *VersionManager) CreateVersion(ctx context.Context, strategyID string, params map[string]float64) (*StrategyVersion, error) {
	// 获取当前最新版本号
	currentVersion := 0
	for _, v := range m.versions {
		if v.StrategyID == strategyID && v.Version > currentVersion {
			currentVersion = v.Version
		}
	}

	version := &StrategyVersion{
		ID:         generateVersionID(),
		StrategyID: strategyID,
		Version:    currentVersion + 1,
		Status:     VersionStatusDraft,
		Params:     params,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	m.versions[version.ID] = version
	return version, nil
}

// ApproveVersion approves a strategy version
func (m *VersionManager) ApproveVersion(ctx context.Context, versionID string) error {
	version, exists := m.versions[versionID]
	if !exists {
		return fmt.Errorf("version not found: %s", versionID)
	}

	if version.Status != VersionStatusDraft {
		return fmt.Errorf("invalid status for approval: %s", version.Status)
	}

	version.Status = VersionStatusApproved
	version.UpdatedAt = time.Now()
	return nil
}

// EnableCanary enables canary deployment for a version
func (m *VersionManager) EnableCanary(ctx context.Context, versionID string, allocation float64) error {
	version, exists := m.versions[versionID]
	if !exists {
		return fmt.Errorf("version not found: %s", versionID)
	}

	if version.Status != VersionStatusApproved {
		return fmt.Errorf("invalid status for canary: %s", version.Status)
	}

	// 检查资金分配比例
	maxAllocation := 0.2 // Default fallback
	if config := config.GetAlgorithmConfig(); config != nil {
		maxAllocation = config.GetMaxWeight()
	}
	
	if allocation <= 0 || allocation > maxAllocation {
		return fmt.Errorf("invalid canary allocation: %.2f (max: %.2f)", allocation, maxAllocation)
	}

	version.Status = VersionStatusCanary
	version.UpdatedAt = time.Now()
	return nil
}

// PromoteToActive promotes a canary version to active
func (m *VersionManager) PromoteToActive(ctx context.Context, versionID string) error {
	version, exists := m.versions[versionID]
	if !exists {
		return fmt.Errorf("version not found: %s", versionID)
	}

	if version.Status != VersionStatusCanary {
		return fmt.Errorf("invalid status for promotion: %s", version.Status)
	}

	// 检查性能指标
	if !checkPerformanceForPromotion(version.Performance) {
		return fmt.Errorf("performance not meeting promotion criteria")
	}

	// 禁用当前active版本
	for _, v := range m.versions {
		if v.StrategyID == version.StrategyID && v.Status == VersionStatusActive {
			v.Status = VersionStatusDisabled
			v.UpdatedAt = time.Now()
		}
	}

	version.Status = VersionStatusActive
	version.UpdatedAt = time.Now()
	return nil
}

// Helper functions

func generateVersionID() string {
	return fmt.Sprintf("ver_%d", time.Now().UnixNano())
}

func checkPerformanceForPromotion(metrics *sdk.StrategyMetrics) bool {
	if metrics == nil {
		return false
	}

	// 检查关键指标
	if metrics.SharpeRatio < 1.0 {
		return false
	}
	if metrics.MaxDrawdown > 0.2 {
		return false
	}
	if metrics.WinRate < 0.5 {
		return false
	}
	if metrics.ProfitFactor < 1.5 {
		return false
	}

	return true
}
