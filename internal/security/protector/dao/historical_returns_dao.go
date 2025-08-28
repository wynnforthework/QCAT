package dao

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// postgresHistoricalReturnsDAO implements HistoricalReturnsDAO for PostgreSQL
type postgresHistoricalReturnsDAO struct {
	*baseDAO
}

// Insert inserts a new historical return record
func (dao *postgresHistoricalReturnsDAO) Insert(ctx context.Context, return_ *HistoricalReturn) error {
	query := `
		INSERT INTO historical_returns (
			date, return_value, portfolio_value, benchmark_return, volatility, updated_at
		) VALUES (
			:date, :return_value, :portfolio_value, :benchmark_return, :volatility, NOW()
		) ON CONFLICT (date) DO UPDATE SET
			return_value = EXCLUDED.return_value,
			portfolio_value = EXCLUDED.portfolio_value,
			benchmark_return = EXCLUDED.benchmark_return,
			volatility = EXCLUDED.volatility,
			updated_at = NOW()
		RETURNING id, created_at, updated_at`
	
	rows, err := dao.db.NamedQuery(query, return_)
	if err != nil {
		return fmt.Errorf("failed to insert historical return: %w", err)
	}
	defer rows.Close()
	
	if rows.Next() {
		err = rows.Scan(&return_.ID, &return_.CreatedAt, &return_.UpdatedAt)
		if err != nil {
			return fmt.Errorf("failed to scan inserted historical return: %w", err)
		}
	}
	
	return nil
}

// GetByDateRange retrieves historical returns within a date range
func (dao *postgresHistoricalReturnsDAO) GetByDateRange(ctx context.Context, startDate, endDate time.Time) ([]*HistoricalReturn, error) {
	query := `
		SELECT id, date, return_value, portfolio_value, benchmark_return, volatility, created_at, updated_at
		FROM historical_returns
		WHERE date >= $1 AND date <= $2
		ORDER BY date ASC`
	
	var returns []*HistoricalReturn
	err := dao.db.SelectContext(ctx, &returns, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical returns by date range: %w", err)
	}
	
	return returns, nil
}

// GetLastNDays retrieves the last N days of historical returns
func (dao *postgresHistoricalReturnsDAO) GetLastNDays(ctx context.Context, days int) ([]*HistoricalReturn, error) {
	query := `
		SELECT id, date, return_value, portfolio_value, benchmark_return, volatility, created_at, updated_at
		FROM historical_returns
		WHERE date >= CURRENT_DATE - INTERVAL '%d days'
		ORDER BY date ASC`
	
	var returns []*HistoricalReturn
	err := dao.db.SelectContext(ctx, &returns, fmt.Sprintf(query, days))
	if err != nil {
		return nil, fmt.Errorf("failed to get last %d days of historical returns: %w", days, err)
	}
	
	return returns, nil
}

// GetLatest retrieves the most recent historical return
func (dao *postgresHistoricalReturnsDAO) GetLatest(ctx context.Context) (*HistoricalReturn, error) {
	query := `
		SELECT id, date, return_value, portfolio_value, benchmark_return, volatility, created_at, updated_at
		FROM historical_returns
		ORDER BY date DESC
		LIMIT 1`
	
	var return_ HistoricalReturn
	err := dao.db.GetContext(ctx, &return_, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest historical return: %w", err)
	}
	
	return &return_, nil
}

// DeleteOlderThan deletes historical returns older than the specified date
func (dao *postgresHistoricalReturnsDAO) DeleteOlderThan(ctx context.Context, date time.Time) (int64, error) {
	query := `DELETE FROM historical_returns WHERE date < $1`
	
	result, err := dao.db.ExecContext(ctx, query, date)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old historical returns: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	return rowsAffected, nil
}