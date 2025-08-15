package exchange

import (
	"time"
)

// MarginInfo represents margin account information
type MarginInfo struct {
	TotalAssetValue   float64   // 总资产价值
	TotalDebtValue    float64   // 总负债价值
	MarginRatio       float64   // 保证金率 = 总资产/总负债
	MaintenanceMargin float64   // 维持保证金率
	MarginCallRatio   float64   // 追保线
	LiquidationRatio  float64   // 强平线
	UpdatedAt         time.Time // 更新时间
}

// AccountBalance represents account balance information
type AccountBalance struct {
	Asset          string    // 资产名称
	Total          float64   // 总余额
	Available      float64   // 可用余额
	Locked         float64   // 锁定余额
	CrossMargin    float64   // 全仓保证金
	IsolatedMargin float64   // 逐仓保证金
	UnrealizedPnL  float64   // 未实现盈亏
	RealizedPnL    float64   // 已实现盈亏
	UpdatedAt      time.Time // 更新时间
}

// MarginLevel represents different margin thresholds
type MarginLevel int

const (
	MarginLevelSafe MarginLevel = iota
	MarginLevelWarning
	MarginLevelDanger
	MarginLevelLiquidation
)

// MarginAlert represents a margin alert event
type MarginAlert struct {
	Level     MarginLevel // 警告级别
	Ratio     float64     // 当前保证金率
	Threshold float64     // 触发阈值
	Message   string      // 警告信息
	CreatedAt time.Time   // 创建时间
}
