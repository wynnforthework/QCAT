package dao

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// PostgresDAOManager implements DAOManager using PostgreSQL
type PostgresDAOManager struct {
	db *sqlx.DB

	// DAO instances
	historicalReturns    HistoricalReturnsDAO
	historicalEquity     HistoricalEquityDAO
	riskSnapshots        RiskSnapshotsDAO
	transferRecords      TransferRecordsDAO
	emergencyEvents      EmergencyEventsDAO
	positionSnapshots    PositionSnapshotsDAO
	fundStatusSnapshots  FundStatusSnapshotsDAO
	circuitBreakerEvents CircuitBreakerEventsDAO
	protectionMetrics    ProtectionMetricsDAO
}

// NewPostgresDAOManager creates a new PostgreSQL DAO manager
func NewPostgresDAOManager(db *sqlx.DB) *PostgresDAOManager {
	manager := &PostgresDAOManager{db: db}

	// Initialize DAO instances
	manager.historicalReturns = &postgresHistoricalReturnsDAO{baseDAO: newBaseDAO(db)}
	manager.historicalEquity = &postgresHistoricalEquityDAO{baseDAO: newBaseDAO(db)}
	manager.riskSnapshots = &postgresRiskSnapshotsDAO{baseDAO: newBaseDAO(db)}
	manager.transferRecords = &postgresTransferRecordsDAO{baseDAO: newBaseDAO(db)}
	manager.emergencyEvents = &postgresEmergencyEventsDAO{baseDAO: newBaseDAO(db)}
	manager.positionSnapshots = &postgresPositionSnapshotsDAO{baseDAO: newBaseDAO(db)}
	manager.fundStatusSnapshots = &postgresFundStatusSnapshotsDAO{baseDAO: newBaseDAO(db)}
	manager.circuitBreakerEvents = &postgresCircuitBreakerEventsDAO{baseDAO: newBaseDAO(db)}
	manager.protectionMetrics = &postgresProtectionMetricsDAO{baseDAO: newBaseDAO(db)}

	return manager
}

// HistoricalReturns returns the HistoricalReturnsDAO
func (m *PostgresDAOManager) HistoricalReturns() HistoricalReturnsDAO {
	return m.historicalReturns
}

// HistoricalEquity returns the HistoricalEquityDAO
func (m *PostgresDAOManager) HistoricalEquity() HistoricalEquityDAO {
	return m.historicalEquity
}

// RiskSnapshots returns the RiskSnapshotsDAO
func (m *PostgresDAOManager) RiskSnapshots() RiskSnapshotsDAO {
	return m.riskSnapshots
}

// TransferRecords returns the TransferRecordsDAO
func (m *PostgresDAOManager) TransferRecords() TransferRecordsDAO {
	return m.transferRecords
}

// EmergencyEvents returns the EmergencyEventsDAO
func (m *PostgresDAOManager) EmergencyEvents() EmergencyEventsDAO {
	return m.emergencyEvents
}

// PositionSnapshots returns the PositionSnapshotsDAO
func (m *PostgresDAOManager) PositionSnapshots() PositionSnapshotsDAO {
	return m.positionSnapshots
}

// FundStatusSnapshots returns the FundStatusSnapshotsDAO
func (m *PostgresDAOManager) FundStatusSnapshots() FundStatusSnapshotsDAO {
	return m.fundStatusSnapshots
}

// CircuitBreakerEvents returns the CircuitBreakerEventsDAO
func (m *PostgresDAOManager) CircuitBreakerEvents() CircuitBreakerEventsDAO {
	return m.circuitBreakerEvents
}

// ProtectionMetrics returns the ProtectionMetricsDAO
func (m *PostgresDAOManager) ProtectionMetrics() ProtectionMetricsDAO {
	return m.protectionMetrics
}

// BeginTx starts a new transaction
func (m *PostgresDAOManager) BeginTx(ctx context.Context) (TxManager, error) {
	tx, err := m.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &postgresTxManager{
		tx:                   tx,
		historicalReturns:    &postgresHistoricalReturnsDAO{baseDAO: newBaseDAO(tx)},
		historicalEquity:     &postgresHistoricalEquityDAO{baseDAO: newBaseDAO(tx)},
		riskSnapshots:        &postgresRiskSnapshotsDAO{baseDAO: newBaseDAO(tx)},
		transferRecords:      &postgresTransferRecordsDAO{baseDAO: newBaseDAO(tx)},
		emergencyEvents:      &postgresEmergencyEventsDAO{baseDAO: newBaseDAO(tx)},
		positionSnapshots:    &postgresPositionSnapshotsDAO{baseDAO: newBaseDAO(tx)},
		fundStatusSnapshots:  &postgresFundStatusSnapshotsDAO{baseDAO: newBaseDAO(tx)},
		circuitBreakerEvents: &postgresCircuitBreakerEventsDAO{baseDAO: newBaseDAO(tx)},
		protectionMetrics:    &postgresProtectionMetricsDAO{baseDAO: newBaseDAO(tx)},
	}, nil
}

// Close closes the database connection
func (m *PostgresDAOManager) Close() error {
	return m.db.Close()
}

// postgresTxManager implements TxManager for PostgreSQL transactions
type postgresTxManager struct {
	tx *sqlx.Tx

	// DAO instances for transaction
	historicalReturns    HistoricalReturnsDAO
	historicalEquity     HistoricalEquityDAO
	riskSnapshots        RiskSnapshotsDAO
	transferRecords      TransferRecordsDAO
	emergencyEvents      EmergencyEventsDAO
	positionSnapshots    PositionSnapshotsDAO
	fundStatusSnapshots  FundStatusSnapshotsDAO
	circuitBreakerEvents CircuitBreakerEventsDAO
	protectionMetrics    ProtectionMetricsDAO
}

// Commit commits the transaction
func (tx *postgresTxManager) Commit() error {
	return tx.tx.Commit()
}

// Rollback rolls back the transaction
func (tx *postgresTxManager) Rollback() error {
	return tx.tx.Rollback()
}

// HistoricalReturns returns the HistoricalReturnsDAO for this transaction
func (tx *postgresTxManager) HistoricalReturns() HistoricalReturnsDAO {
	return tx.historicalReturns
}

// HistoricalEquity returns the HistoricalEquityDAO for this transaction
func (tx *postgresTxManager) HistoricalEquity() HistoricalEquityDAO {
	return tx.historicalEquity
}

// RiskSnapshots returns the RiskSnapshotsDAO for this transaction
func (tx *postgresTxManager) RiskSnapshots() RiskSnapshotsDAO {
	return tx.riskSnapshots
}

// TransferRecords returns the TransferRecordsDAO for this transaction
func (tx *postgresTxManager) TransferRecords() TransferRecordsDAO {
	return tx.transferRecords
}

// EmergencyEvents returns the EmergencyEventsDAO for this transaction
func (tx *postgresTxManager) EmergencyEvents() EmergencyEventsDAO {
	return tx.emergencyEvents
}

// PositionSnapshots returns the PositionSnapshotsDAO for this transaction
func (tx *postgresTxManager) PositionSnapshots() PositionSnapshotsDAO {
	return tx.positionSnapshots
}

// FundStatusSnapshots returns the FundStatusSnapshotsDAO for this transaction
func (tx *postgresTxManager) FundStatusSnapshots() FundStatusSnapshotsDAO {
	return tx.fundStatusSnapshots
}

// CircuitBreakerEvents returns the CircuitBreakerEventsDAO for this transaction
func (tx *postgresTxManager) CircuitBreakerEvents() CircuitBreakerEventsDAO {
	return tx.circuitBreakerEvents
}

// ProtectionMetrics returns the ProtectionMetricsDAO for this transaction
func (tx *postgresTxManager) ProtectionMetrics() ProtectionMetricsDAO {
	return tx.protectionMetrics
}

// Queryer interface for both *sqlx.DB and *sqlx.Tx
type Queryer interface {
	QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error)
	QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error)
	NamedQuery(query string, arg interface{}) (*sqlx.Rows, error)
}

// baseDAO provides common functionality for all DAOs
type baseDAO struct {
	db Queryer
}

// newBaseDAO creates a new base DAO with either a DB or Tx
func newBaseDAO(dbOrTx interface{}) *baseDAO {
	switch v := dbOrTx.(type) {
	case *sqlx.DB:
		return &baseDAO{db: v}
	case *sqlx.Tx:
		return &baseDAO{db: v}
	default:
		panic(fmt.Sprintf("unsupported database type: %T", dbOrTx))
	}
}

// Helper function to handle nullable time values
func nullableTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return *t
}

// Helper function to handle nullable float values
func nullableFloat64(f *float64) interface{} {
	if f == nil {
		return nil
	}
	return *f
}

// Helper function to handle nullable string values
func nullableString(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}

// Helper function to handle nullable int values
func nullableInt(i *int) interface{} {
	if i == nil {
		return nil
	}
	return *i
}
