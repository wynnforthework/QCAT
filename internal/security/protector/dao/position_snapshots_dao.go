package dao

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// postgresPositionSnapshotsDAO implements PositionSnapshotsDAO for PostgreSQL
type postgresPositionSnapshotsDAO struct {
	*baseDAO
}

// Insert inserts a new position snapshot record
func (dao *postgresPositionSnapshotsDAO) Insert(ctx context.Context, snapshot *PositionSnapshot) error {
	query := `
		INSERT INTO position_snapshots (
			timestamp, symbol, side, size, notional, entry_price, mark_price,
			unrealized_pnl, realized_pnl, leverage, margin_type, isolated_margin,
			maintenance_margin, liquidation_price
		) VALUES (
			:timestamp, :symbol, :side, :size, :notional, :entry_price, :mark_price,
			:unrealized_pnl, :realized_pnl, :leverage, :margin_type, :isolated_margin,
			:maintenance_margin, :liquidation_price
		) RETURNING id, created_at`
	
	rows, err := dao.db.NamedQuery(query, snapshot)
	if err != nil {
		return fmt.Errorf("failed to insert position snapshot: %w", err)
	}
	defer rows.Close()
	
	if rows.Next() {
		err = rows.Scan(&snapshot.ID, &snapshot.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to scan inserted position snapshot: %w", err)
		}
	}
	
	return nil
}

// GetBySymbol retrieves position snapshots for a specific symbol
func (dao *postgresPositionSnapshotsDAO) GetBySymbol(ctx context.Context, symbol string, limit int) ([]*PositionSnapshot, error) {
	query := `
		SELECT id, timestamp, symbol, side, size, notional, entry_price, mark_price,
			   unrealized_pnl, realized_pnl, leverage, margin_type, isolated_margin,
			   maintenance_margin, liquidation_price, created_at
		FROM position_snapshots
		WHERE symbol = $1
		ORDER BY timestamp DESC
		LIMIT $2`
	
	var snapshots []*PositionSnapshot
	err := dao.db.SelectContext(ctx, &snapshots, query, symbol, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get position snapshots by symbol: %w", err)
	}
	
	return snapshots, nil
}

// GetByTimeRange retrieves position snapshots within a time range
func (dao *postgresPositionSnapshotsDAO) GetByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*PositionSnapshot, error) {
	query := `
		SELECT id, timestamp, symbol, side, size, notional, entry_price, mark_price,
			   unrealized_pnl, realized_pnl, leverage, margin_type, isolated_margin,
			   maintenance_margin, liquidation_price, created_at
		FROM position_snapshots
		WHERE timestamp >= $1 AND timestamp <= $2
		ORDER BY timestamp ASC`
	
	var snapshots []*PositionSnapshot
	err := dao.db.SelectContext(ctx, &snapshots, query, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get position snapshots by time range: %w", err)
	}
	
	return snapshots, nil
}

// GetLatestBySymbol retrieves the latest position snapshot for a specific symbol
func (dao *postgresPositionSnapshotsDAO) GetLatestBySymbol(ctx context.Context, symbol string) (*PositionSnapshot, error) {
	query := `
		SELECT id, timestamp, symbol, side, size, notional, entry_price, mark_price,
			   unrealized_pnl, realized_pnl, leverage, margin_type, isolated_margin,
			   maintenance_margin, liquidation_price, created_at
		FROM position_snapshots
		WHERE symbol = $1
		ORDER BY timestamp DESC
		LIMIT 1`
	
	var snapshot PositionSnapshot
	err := dao.db.GetContext(ctx, &snapshot, query, symbol)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest position snapshot by symbol: %w", err)
	}
	
	return &snapshot, nil
}

// GetAllLatest retrieves the latest position snapshot for each symbol
func (dao *postgresPositionSnapshotsDAO) GetAllLatest(ctx context.Context) ([]*PositionSnapshot, error) {
	query := `
		SELECT DISTINCT ON (symbol) 
			   id, timestamp, symbol, side, size, notional, entry_price, mark_price,
			   unrealized_pnl, realized_pnl, leverage, margin_type, isolated_margin,
			   maintenance_margin, liquidation_price, created_at
		FROM position_snapshots
		ORDER BY symbol, timestamp DESC`
	
	var snapshots []*PositionSnapshot
	err := dao.db.SelectContext(ctx, &snapshots, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all latest position snapshots: %w", err)
	}
	
	return snapshots, nil
}

// DeleteOlderThan deletes position snapshots older than the specified timestamp
func (dao *postgresPositionSnapshotsDAO) DeleteOlderThan(ctx context.Context, timestamp time.Time) (int64, error) {
	query := `DELETE FROM position_snapshots WHERE timestamp < $1`
	
	result, err := dao.db.ExecContext(ctx, query, timestamp)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old position snapshots: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	return rowsAffected, nil
}