package dao

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// postgresFundStatusSnapshotsDAO implements FundStatusSnapshotsDAO for PostgreSQL
type postgresFundStatusSnapshotsDAO struct {
	*baseDAO
}

// Insert inserts a new fund status snapshot record
func (dao *postgresFundStatusSnapshotsDAO) Insert(ctx context.Context, snapshot *FundStatusSnapshot) error {
	query := `
		INSERT INTO fund_status_snapshots (
			timestamp, total_balance, available_balance, locked_balance,
			profit_loss, daily_pl, unrealized_pl, realized_pl,
			current_risk, max_risk, var_95, expected_shortfall,
			total_positions, active_positions, long_positions, short_positions
		) VALUES (
			:timestamp, :total_balance, :available_balance, :locked_balance,
			:profit_loss, :daily_pl, :unrealized_pl, :realized_pl,
			:current_risk, :max_risk, :var_95, :expected_shortfall,
			:total_positions, :active_positions, :long_positions, :short_positions
		) RETURNING id, created_at`
	
	rows, err := dao.db.NamedQuery(query, snapshot)
	if err != nil {
		return fmt.Errorf("failed to insert fund status snapshot: %w", err)
	}
	defer rows.Close()
	
	if rows.Next() {
		err = rows.Scan(&snapshot.ID, &snapshot.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to scan inserted fund status snapshot: %w", err)
		}
	}
	
	return nil
}

// GetByTimeRange retrieves fund status snapshots within a time range
func (dao *postgresFundStatusSnapshotsDAO) GetByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*FundStatusSnapshot, error) {
	query := `
		SELECT id, timestamp, total_balance, available_balance, locked_balance,
			   profit_loss, daily_pl, unrealized_pl, realized_pl,
			   current_risk, max_risk, var_95, expected_shortfall,
			   total_positions, active_positions, long_positions, short_positions, created_at
		FROM fund_status_snapshots
		WHERE timestamp >= $1 AND timestamp <= $2
		ORDER BY timestamp ASC`
	
	var snapshots []*FundStatusSnapshot
	err := dao.db.SelectContext(ctx, &snapshots, query, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get fund status snapshots by time range: %w", err)
	}
	
	return snapshots, nil
}

// GetLastNDays retrieves the last N days of fund status snapshots
func (dao *postgresFundStatusSnapshotsDAO) GetLastNDays(ctx context.Context, days int) ([]*FundStatusSnapshot, error) {
	query := `
		SELECT id, timestamp, total_balance, available_balance, locked_balance,
			   profit_loss, daily_pl, unrealized_pl, realized_pl,
			   current_risk, max_risk, var_95, expected_shortfall,
			   total_positions, active_positions, long_positions, short_positions, created_at
		FROM fund_status_snapshots
		WHERE timestamp >= NOW() - INTERVAL '%d days'
		ORDER BY timestamp ASC`
	
	var snapshots []*FundStatusSnapshot
	err := dao.db.SelectContext(ctx, &snapshots, fmt.Sprintf(query, days))
	if err != nil {
		return nil, fmt.Errorf("failed to get last %d days of fund status snapshots: %w", days, err)
	}
	
	return snapshots, nil
}

// GetLatest retrieves the most recent fund status snapshot
func (dao *postgresFundStatusSnapshotsDAO) GetLatest(ctx context.Context) (*FundStatusSnapshot, error) {
	query := `
		SELECT id, timestamp, total_balance, available_balance, locked_balance,
			   profit_loss, daily_pl, unrealized_pl, realized_pl,
			   current_risk, max_risk, var_95, expected_shortfall,
			   total_positions, active_positions, long_positions, short_positions, created_at
		FROM fund_status_snapshots
		ORDER BY timestamp DESC
		LIMIT 1`
	
	var snapshot FundStatusSnapshot
	err := dao.db.GetContext(ctx, &snapshot, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest fund status snapshot: %w", err)
	}
	
	return &snapshot, nil
}

// DeleteOlderThan deletes fund status snapshots older than the specified timestamp
func (dao *postgresFundStatusSnapshotsDAO) DeleteOlderThan(ctx context.Context, timestamp time.Time) (int64, error) {
	query := `DELETE FROM fund_status_snapshots WHERE timestamp < $1`
	
	result, err := dao.db.ExecContext(ctx, query, timestamp)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old fund status snapshots: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	return rowsAffected, nil
}