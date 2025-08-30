package dao

import (
	"context"
	"fmt"
)

// postgresCircuitBreakerEventsDAO implements CircuitBreakerEventsDAO for PostgreSQL
type postgresCircuitBreakerEventsDAO struct {
	*baseDAO
}

// Insert inserts a new circuit breaker event record
func (dao *postgresCircuitBreakerEventsDAO) Insert(ctx context.Context, event *CircuitBreakerEvent) error {
	query := `
		INSERT INTO circuit_breaker_events (
			trigger_reason, loss_ratio, trigger_count, cooldown_period_minutes,
			triggered_at, reset_at, status, metadata
		) VALUES (
			:trigger_reason, :loss_ratio, :trigger_count, :cooldown_period_minutes,
			:triggered_at, :reset_at, :status, :metadata
		) RETURNING id, created_at`

	rows, err := dao.db.NamedQuery(query, event)
	if err != nil {
		return fmt.Errorf("failed to insert circuit breaker event: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&event.ID, &event.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to scan inserted circuit breaker event: %w", err)
		}
	}

	return nil
}

// Update updates an existing circuit breaker event record
func (dao *postgresCircuitBreakerEventsDAO) Update(ctx context.Context, event *CircuitBreakerEvent) error {
	query := `
		UPDATE circuit_breaker_events SET
			trigger_reason = :trigger_reason,
			loss_ratio = :loss_ratio,
			trigger_count = :trigger_count,
			cooldown_period_minutes = :cooldown_period_minutes,
			triggered_at = :triggered_at,
			reset_at = :reset_at,
			status = :status,
			metadata = :metadata
		WHERE id = :id`

	result, err := dao.db.NamedExecContext(ctx, query, event)
	if err != nil {
		return fmt.Errorf("failed to update circuit breaker event: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("circuit breaker event with id %d not found", event.ID)
	}

	return nil
}

// GetByStatus retrieves circuit breaker events by status
func (dao *postgresCircuitBreakerEventsDAO) GetByStatus(ctx context.Context, status string, limit int) ([]*CircuitBreakerEvent, error) {
	query := `
		SELECT id, trigger_reason, loss_ratio, trigger_count, cooldown_period_minutes,
			   triggered_at, reset_at, status, metadata, created_at
		FROM circuit_breaker_events
		WHERE status = $1
		ORDER BY triggered_at DESC
		LIMIT $2`

	var events []*CircuitBreakerEvent
	err := dao.db.SelectContext(ctx, &events, query, status, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get circuit breaker events by status: %w", err)
	}

	return events, nil
}

// GetRecent retrieves recent circuit breaker events
func (dao *postgresCircuitBreakerEventsDAO) GetRecent(ctx context.Context, limit int) ([]*CircuitBreakerEvent, error) {
	query := `
		SELECT id, trigger_reason, loss_ratio, trigger_count, cooldown_period_minutes,
			   triggered_at, reset_at, status, metadata, created_at
		FROM circuit_breaker_events
		ORDER BY triggered_at DESC
		LIMIT $1`

	var events []*CircuitBreakerEvent
	err := dao.db.SelectContext(ctx, &events, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent circuit breaker events: %w", err)
	}

	return events, nil
}

// GetActive retrieves all active circuit breaker events
func (dao *postgresCircuitBreakerEventsDAO) GetActive(ctx context.Context) ([]*CircuitBreakerEvent, error) {
	query := `
		SELECT id, trigger_reason, loss_ratio, trigger_count, cooldown_period_minutes,
			   triggered_at, reset_at, status, metadata, created_at
		FROM circuit_breaker_events
		WHERE status = 'ACTIVE'
		ORDER BY triggered_at DESC`

	var events []*CircuitBreakerEvent
	err := dao.db.SelectContext(ctx, &events, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active circuit breaker events: %w", err)
	}

	return events, nil
}

// UpdateStatus updates the status of a circuit breaker event
func (dao *postgresCircuitBreakerEventsDAO) UpdateStatus(ctx context.Context, id int64, status string) error {
	var query string
	var args []interface{}

	if status == "RESET" {
		query = `UPDATE circuit_breaker_events SET status = $2, reset_at = NOW() WHERE id = $1`
		args = []interface{}{id, status}
	} else {
		query = `UPDATE circuit_breaker_events SET status = $2 WHERE id = $1`
		args = []interface{}{id, status}
	}

	result, err := dao.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update circuit breaker event status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("circuit breaker event with id %d not found", id)
	}

	return nil
}
