package dao

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// postgresHistoricalEquityDAO implements HistoricalEquityDAO for PostgreSQL
type postgresHistoricalEquityDAO struct {
	*baseDAO
}

// Insert inserts a new historical equity record
func (dao *postgresHistoricalEquityDAO) Insert(ctx context.Context, equity *HistoricalEquity) error {
	query := `
		INSERT INTO historical_equity (
			timestamp, equity_value, available_balance, locked_balance, 
			unrealized_pnl, realized_pnl, total_positions, active_positions
		) VALUES (
			:timestamp, :equity_value, :available_balance, :locked_balance,
			:unrealized_pnl, :realized_pnl, :total_positions, :active_positions
		) ON CONFLICT (timestamp) DO UPDATE SET
			equity_value = EXCLUDED.equity_value,
			available_balance = EXCLUDED.available_balance,
			locked_balance = EXCLUDED.locked_balance,
			unrealized_pnl = EXCLUDED.unrealized_pnl,
			realized_pnl = EXCLUDED.realized_pnl,
			total_positions = EXCLUDED.total_positions,
			active_positions = EXCLUDED.active_positions
		RETURNING id, created_at`
	
	rows, err := dao.db.NamedQuery(query, equity)
	if err != nil {
		return fmt.Errorf("failed to insert historical equity: %w", err)
	}
	defer rows.Close()
	
	if rows.Next() {
		err = rows.Scan(&equity.ID, &equity.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to scan inserted historical equity: %w", err)
		}
	}
	
	return nil
}

// GetByTimeRange retrieves historical equity within a time range
func (dao *postgresHistoricalEquityDAO) GetByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*HistoricalEquity, error) {
	query := `
		SELECT id, timestamp, equity_value, available_balance, locked_balance,
			   unrealized_pnl, realized_pnl, total_positions, active_positions, created_at
		FROM historical_equity
		WHERE timestamp >= $1 AND timestamp <= $2
		ORDER BY timestamp ASC`
	
	var equities []*HistoricalEquity
	err := dao.db.SelectContext(ctx, &equities, query, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical equity by time range: %w", err)
	}
	
	return equities, nil
}

// GetLastNDays retrieves the last N days of historical equity
func (dao *postgresHistoricalEquityDAO) GetLastNDays(ctx context.Context, days int) ([]*HistoricalEquity, error) {
	query := `
		SELECT id, timestamp, equity_value, available_balance, locked_balance,
			   unrealized_pnl, realized_pnl, total_positions, active_positions, created_at
		FROM historical_equity
		WHERE timestamp >= NOW() - INTERVAL '%d days'
		ORDER BY timestamp ASC`
	
	var equities []*HistoricalEquity
	err := dao.db.SelectContext(ctx, &equities, fmt.Sprintf(query, days))
	if err != nil {
		return nil, fmt.Errorf("failed to get last %d days of historical equity: %w", days, err)
	}
	
	return equities, nil
}

// GetLatest retrieves the most recent historical equity
func (dao *postgresHistoricalEquityDAO) GetLatest(ctx context.Context) (*HistoricalEquity, error) {
	query := `
		SELECT id, timestamp, equity_value, available_balance, locked_balance,
			   unrealized_pnl, realized_pnl, total_positions, active_positions, created_at
		FROM historical_equity
		ORDER BY timestamp DESC
		LIMIT 1`
	
	var equity HistoricalEquity
	err := dao.db.GetContext(ctx, &equity, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest historical equity: %w", err)
	}
	
	return &equity, nil
}

// DeleteOlderThan deletes historical equity older than the specified timestamp
func (dao *postgresHistoricalEquityDAO) DeleteOlderThan(ctx context.Context, timestamp time.Time) (int64, error) {
	query := `DELETE FROM historical_equity WHERE timestamp < $1`
	
	result, err := dao.db.ExecContext(ctx, query, timestamp)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old historical equity: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	return rowsAffected, nil
}