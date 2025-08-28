package dao

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// postgresRiskSnapshotsDAO implements RiskSnapshotsDAO for PostgreSQL
type postgresRiskSnapshotsDAO struct {
	*baseDAO
}

// Insert inserts a new risk snapshot record
func (dao *postgresRiskSnapshotsDAO) Insert(ctx context.Context, snapshot *RiskSnapshot) error {
	query := `
		INSERT INTO risk_snapshots (
			timestamp, risk_level, risk_score, var_95, expected_shortfall,
			max_drawdown, volatility_index, leverage, concentration,
			portfolio_beta, sharpe_ratio, sortino_ratio, calmar_ratio
		) VALUES (
			:timestamp, :risk_level, :risk_score, :var_95, :expected_shortfall,
			:max_drawdown, :volatility_index, :leverage, :concentration,
			:portfolio_beta, :sharpe_ratio, :sortino_ratio, :calmar_ratio
		) RETURNING id, created_at`
	
	rows, err := dao.db.NamedQuery(query, snapshot)
	if err != nil {
		return fmt.Errorf("failed to insert risk snapshot: %w", err)
	}
	defer rows.Close()
	
	if rows.Next() {
		err = rows.Scan(&snapshot.ID, &snapshot.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to scan inserted risk snapshot: %w", err)
		}
	}
	
	return nil
}

// GetByTimeRange retrieves risk snapshots within a time range
func (dao *postgresRiskSnapshotsDAO) GetByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*RiskSnapshot, error) {
	query := `
		SELECT id, timestamp, risk_level, risk_score, var_95, expected_shortfall,
			   max_drawdown, volatility_index, leverage, concentration,
			   portfolio_beta, sharpe_ratio, sortino_ratio, calmar_ratio, created_at
		FROM risk_snapshots
		WHERE timestamp >= $1 AND timestamp <= $2
		ORDER BY timestamp ASC`
	
	var snapshots []*RiskSnapshot
	err := dao.db.SelectContext(ctx, &snapshots, query, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get risk snapshots by time range: %w", err)
	}
	
	return snapshots, nil
}

// GetLastNDays retrieves the last N days of risk snapshots
func (dao *postgresRiskSnapshotsDAO) GetLastNDays(ctx context.Context, days int) ([]*RiskSnapshot, error) {
	query := `
		SELECT id, timestamp, risk_level, risk_score, var_95, expected_shortfall,
			   max_drawdown, volatility_index, leverage, concentration,
			   portfolio_beta, sharpe_ratio, sortino_ratio, calmar_ratio, created_at
		FROM risk_snapshots
		WHERE timestamp >= NOW() - INTERVAL '%d days'
		ORDER BY timestamp ASC`
	
	var snapshots []*RiskSnapshot
	err := dao.db.SelectContext(ctx, &snapshots, fmt.Sprintf(query, days))
	if err != nil {
		return nil, fmt.Errorf("failed to get last %d days of risk snapshots: %w", days, err)
	}
	
	return snapshots, nil
}

// GetByRiskLevel retrieves risk snapshots by risk level
func (dao *postgresRiskSnapshotsDAO) GetByRiskLevel(ctx context.Context, riskLevel string, limit int) ([]*RiskSnapshot, error) {
	query := `
		SELECT id, timestamp, risk_level, risk_score, var_95, expected_shortfall,
			   max_drawdown, volatility_index, leverage, concentration,
			   portfolio_beta, sharpe_ratio, sortino_ratio, calmar_ratio, created_at
		FROM risk_snapshots
		WHERE risk_level = $1
		ORDER BY timestamp DESC
		LIMIT $2`
	
	var snapshots []*RiskSnapshot
	err := dao.db.SelectContext(ctx, &snapshots, query, riskLevel, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get risk snapshots by risk level: %w", err)
	}
	
	return snapshots, nil
}

// GetLatest retrieves the most recent risk snapshot
func (dao *postgresRiskSnapshotsDAO) GetLatest(ctx context.Context) (*RiskSnapshot, error) {
	query := `
		SELECT id, timestamp, risk_level, risk_score, var_95, expected_shortfall,
			   max_drawdown, volatility_index, leverage, concentration,
			   portfolio_beta, sharpe_ratio, sortino_ratio, calmar_ratio, created_at
		FROM risk_snapshots
		ORDER BY timestamp DESC
		LIMIT 1`
	
	var snapshot RiskSnapshot
	err := dao.db.GetContext(ctx, &snapshot, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest risk snapshot: %w", err)
	}
	
	return &snapshot, nil
}

// DeleteOlderThan deletes risk snapshots older than the specified timestamp
func (dao *postgresRiskSnapshotsDAO) DeleteOlderThan(ctx context.Context, timestamp time.Time) (int64, error) {
	query := `DELETE FROM risk_snapshots WHERE timestamp < $1`
	
	result, err := dao.db.ExecContext(ctx, query, timestamp)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old risk snapshots: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	return rowsAffected, nil
}