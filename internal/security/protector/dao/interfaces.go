package dao

import (
	"context"
	"time"
)

// HistoricalReturnsDAO defines the interface for historical returns data access
type HistoricalReturnsDAO interface {
	Insert(ctx context.Context, historicalReturn *HistoricalReturn) error
	GetByDateRange(ctx context.Context, startDate, endDate time.Time) ([]*HistoricalReturn, error)
	GetLastNDays(ctx context.Context, days int) ([]*HistoricalReturn, error)
	GetLatest(ctx context.Context) (*HistoricalReturn, error)
	DeleteOlderThan(ctx context.Context, date time.Time) (int64, error)
}

// HistoricalEquityDAO defines the interface for historical equity data access
type HistoricalEquityDAO interface {
	Insert(ctx context.Context, equity *HistoricalEquity) error
	GetByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*HistoricalEquity, error)
	GetLastNDays(ctx context.Context, days int) ([]*HistoricalEquity, error)
	GetLatest(ctx context.Context) (*HistoricalEquity, error)
	DeleteOlderThan(ctx context.Context, timestamp time.Time) (int64, error)
}

// RiskSnapshotsDAO defines the interface for risk snapshots data access
type RiskSnapshotsDAO interface {
	Insert(ctx context.Context, snapshot *RiskSnapshot) error
	GetByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*RiskSnapshot, error)
	GetLastNDays(ctx context.Context, days int) ([]*RiskSnapshot, error)
	GetByRiskLevel(ctx context.Context, riskLevel string, limit int) ([]*RiskSnapshot, error)
	GetLatest(ctx context.Context) (*RiskSnapshot, error)
	DeleteOlderThan(ctx context.Context, timestamp time.Time) (int64, error)
}

// TransferRecordsDAO defines the interface for transfer records data access
type TransferRecordsDAO interface {
	Insert(ctx context.Context, transfer *TransferRecord) error
	Update(ctx context.Context, transfer *TransferRecord) error
	GetByID(ctx context.Context, id string) (*TransferRecord, error)
	GetByStatus(ctx context.Context, status string, limit int) ([]*TransferRecord, error)
	GetByType(ctx context.Context, transferType string, limit int) ([]*TransferRecord, error)
	GetRecent(ctx context.Context, limit int) ([]*TransferRecord, error)
	GetByDateRange(ctx context.Context, startTime, endTime time.Time) ([]*TransferRecord, error)
	UpdateStatus(ctx context.Context, id string, status string) error
	DeleteOlderThan(ctx context.Context, timestamp time.Time) (int64, error)
}

// EmergencyEventsDAO defines the interface for emergency events data access
type EmergencyEventsDAO interface {
	Insert(ctx context.Context, event *EmergencyEvent) error
	Update(ctx context.Context, event *EmergencyEvent) error
	GetByID(ctx context.Context, id string) (*EmergencyEvent, error)
	GetBySeverity(ctx context.Context, severity string, limit int) ([]*EmergencyEvent, error)
	GetByStatus(ctx context.Context, status string, limit int) ([]*EmergencyEvent, error)
	GetActive(ctx context.Context) ([]*EmergencyEvent, error)
	GetRecent(ctx context.Context, limit int) ([]*EmergencyEvent, error)
	UpdateStatus(ctx context.Context, id string, status string) error
	DeleteOlderThan(ctx context.Context, timestamp time.Time) (int64, error)
}

// PositionSnapshotsDAO defines the interface for position snapshots data access
type PositionSnapshotsDAO interface {
	Insert(ctx context.Context, snapshot *PositionSnapshot) error
	GetBySymbol(ctx context.Context, symbol string, limit int) ([]*PositionSnapshot, error)
	GetByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*PositionSnapshot, error)
	GetLatestBySymbol(ctx context.Context, symbol string) (*PositionSnapshot, error)
	GetAllLatest(ctx context.Context) ([]*PositionSnapshot, error)
	DeleteOlderThan(ctx context.Context, timestamp time.Time) (int64, error)
}

// FundStatusSnapshotsDAO defines the interface for fund status snapshots data access
type FundStatusSnapshotsDAO interface {
	Insert(ctx context.Context, snapshot *FundStatusSnapshot) error
	GetByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*FundStatusSnapshot, error)
	GetLastNDays(ctx context.Context, days int) ([]*FundStatusSnapshot, error)
	GetLatest(ctx context.Context) (*FundStatusSnapshot, error)
	DeleteOlderThan(ctx context.Context, timestamp time.Time) (int64, error)
}

// CircuitBreakerEventsDAO defines the interface for circuit breaker events data access
type CircuitBreakerEventsDAO interface {
	Insert(ctx context.Context, event *CircuitBreakerEvent) error
	Update(ctx context.Context, event *CircuitBreakerEvent) error
	GetByStatus(ctx context.Context, status string, limit int) ([]*CircuitBreakerEvent, error)
	GetRecent(ctx context.Context, limit int) ([]*CircuitBreakerEvent, error)
	GetActive(ctx context.Context) ([]*CircuitBreakerEvent, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
}

// ProtectionMetricsDAO defines the interface for protection metrics data access
type ProtectionMetricsDAO interface {
	Insert(ctx context.Context, metrics *ProtectionMetrics) error
	Update(ctx context.Context, metrics *ProtectionMetrics) error
	GetLatest(ctx context.Context) (*ProtectionMetrics, error)
	GetByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*ProtectionMetrics, error)
}

// DAOManager provides access to all DAO interfaces
type DAOManager interface {
	HistoricalReturns() HistoricalReturnsDAO
	HistoricalEquity() HistoricalEquityDAO
	RiskSnapshots() RiskSnapshotsDAO
	TransferRecords() TransferRecordsDAO
	EmergencyEvents() EmergencyEventsDAO
	PositionSnapshots() PositionSnapshotsDAO
	FundStatusSnapshots() FundStatusSnapshotsDAO
	CircuitBreakerEvents() CircuitBreakerEventsDAO
	ProtectionMetrics() ProtectionMetricsDAO

	// Transaction management
	BeginTx(ctx context.Context) (TxManager, error)
	Close() error
}

// TxManager provides transaction management
type TxManager interface {
	Commit() error
	Rollback() error

	// Access to DAOs within transaction
	HistoricalReturns() HistoricalReturnsDAO
	HistoricalEquity() HistoricalEquityDAO
	RiskSnapshots() RiskSnapshotsDAO
	TransferRecords() TransferRecordsDAO
	EmergencyEvents() EmergencyEventsDAO
	PositionSnapshots() PositionSnapshotsDAO
	FundStatusSnapshots() FundStatusSnapshotsDAO
	CircuitBreakerEvents() CircuitBreakerEventsDAO
	ProtectionMetrics() ProtectionMetricsDAO
}
