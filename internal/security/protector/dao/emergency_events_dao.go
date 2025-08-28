package dao

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// postgresEmergencyEventsDAO implements EmergencyEventsDAO for PostgreSQL
type postgresEmergencyEventsDAO struct {
	*baseDAO
}

// Insert inserts a new emergency event record
func (dao *postgresEmergencyEventsDAO) Insert(ctx context.Context, event *EmergencyEvent) error {
	query := `
		INSERT INTO emergency_events (
			id, type, severity, description, trigger_data, response_data,
			status, response_time_ms, actions_taken, notifications_sent,
			acknowledged_at, resolved_at, escalated_at
		) VALUES (
			:id, :type, :severity, :description, :trigger_data, :response_data,
			:status, :response_time_ms, :actions_taken, :notifications_sent,
			:acknowledged_at, :resolved_at, :escalated_at
		) RETURNING created_at`
	
	rows, err := dao.db.NamedQuery(query, event)
	if err != nil {
		return fmt.Errorf("failed to insert emergency event: %w", err)
	}
	defer rows.Close()
	
	if rows.Next() {
		err = rows.Scan(&event.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to scan inserted emergency event: %w", err)
		}
	}
	
	return nil
}

// Update updates an existing emergency event record
func (dao *postgresEmergencyEventsDAO) Update(ctx context.Context, event *EmergencyEvent) error {
	query := `
		UPDATE emergency_events SET
			type = :type,
			severity = :severity,
			description = :description,
			trigger_data = :trigger_data,
			response_data = :response_data,
			status = :status,
			response_time_ms = :response_time_ms,
			actions_taken = :actions_taken,
			notifications_sent = :notifications_sent,
			acknowledged_at = :acknowledged_at,
			resolved_at = :resolved_at,
			escalated_at = :escalated_at
		WHERE id = :id`
	
	result, err := dao.db.NamedExecContext(ctx, query, event)
	if err != nil {
		return fmt.Errorf("failed to update emergency event: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("emergency event with id %s not found", event.ID)
	}
	
	return nil
}

// GetByID retrieves an emergency event by ID
func (dao *postgresEmergencyEventsDAO) GetByID(ctx context.Context, id string) (*EmergencyEvent, error) {
	query := `
		SELECT id, type, severity, description, trigger_data, response_data,
			   status, response_time_ms, actions_taken, notifications_sent,
			   created_at, acknowledged_at, resolved_at, escalated_at
		FROM emergency_events
		WHERE id = $1`
	
	var event EmergencyEvent
	err := dao.db.GetContext(ctx, &event, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get emergency event by id: %w", err)
	}
	
	return &event, nil
}

// GetBySeverity retrieves emergency events by severity
func (dao *postgresEmergencyEventsDAO) GetBySeverity(ctx context.Context, severity string, limit int) ([]*EmergencyEvent, error) {
	query := `
		SELECT id, type, severity, description, trigger_data, response_data,
			   status, response_time_ms, actions_taken, notifications_sent,
			   created_at, acknowledged_at, resolved_at, escalated_at
		FROM emergency_events
		WHERE severity = $1
		ORDER BY created_at DESC
		LIMIT $2`
	
	var events []*EmergencyEvent
	err := dao.db.SelectContext(ctx, &events, query, severity, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get emergency events by severity: %w", err)
	}
	
	return events, nil
}

// GetByStatus retrieves emergency events by status
func (dao *postgresEmergencyEventsDAO) GetByStatus(ctx context.Context, status string, limit int) ([]*EmergencyEvent, error) {
	query := `
		SELECT id, type, severity, description, trigger_data, response_data,
			   status, response_time_ms, actions_taken, notifications_sent,
			   created_at, acknowledged_at, resolved_at, escalated_at
		FROM emergency_events
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2`
	
	var events []*EmergencyEvent
	err := dao.db.SelectContext(ctx, &events, query, status, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get emergency events by status: %w", err)
	}
	
	return events, nil
}

// GetActive retrieves all active emergency events
func (dao *postgresEmergencyEventsDAO) GetActive(ctx context.Context) ([]*EmergencyEvent, error) {
	query := `
		SELECT id, type, severity, description, trigger_data, response_data,
			   status, response_time_ms, actions_taken, notifications_sent,
			   created_at, acknowledged_at, resolved_at, escalated_at
		FROM emergency_events
		WHERE status = 'ACTIVE'
		ORDER BY severity DESC, created_at DESC`
	
	var events []*EmergencyEvent
	err := dao.db.SelectContext(ctx, &events, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active emergency events: %w", err)
	}
	
	return events, nil
}

// GetRecent retrieves recent emergency events
func (dao *postgresEmergencyEventsDAO) GetRecent(ctx context.Context, limit int) ([]*EmergencyEvent, error) {
	query := `
		SELECT id, type, severity, description, trigger_data, response_data,
			   status, response_time_ms, actions_taken, notifications_sent,
			   created_at, acknowledged_at, resolved_at, escalated_at
		FROM emergency_events
		ORDER BY created_at DESC
		LIMIT $1`
	
	var events []*EmergencyEvent
	err := dao.db.SelectContext(ctx, &events, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent emergency events: %w", err)
	}
	
	return events, nil
}

// UpdateStatus updates the status of an emergency event
func (dao *postgresEmergencyEventsDAO) UpdateStatus(ctx context.Context, id string, status string) error {
	var setClause string
	var args []interface{}
	
	switch status {
	case "ACKNOWLEDGED":
		setClause = "status = $2, acknowledged_at = NOW()"
		args = []interface{}{id, status}
	case "RESOLVED":
		setClause = "status = $2, resolved_at = NOW()"
		args = []interface{}{id, status}
	case "ESCALATED":
		setClause = "status = $2, escalated_at = NOW()"
		args = []interface{}{id, status}
	default:
		setClause = "status = $2"
		args = []interface{}{id, status}
	}
	
	query := fmt.Sprintf("UPDATE emergency_events SET %s WHERE id = $1", setClause)
	
	result, err := dao.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update emergency event status: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("emergency event with id %s not found", id)
	}
	
	return nil
}

// DeleteOlderThan deletes emergency events older than the specified timestamp
func (dao *postgresEmergencyEventsDAO) DeleteOlderThan(ctx context.Context, timestamp time.Time) (int64, error) {
	query := `DELETE FROM emergency_events WHERE created_at < $1`
	
	result, err := dao.db.ExecContext(ctx, query, timestamp)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old emergency events: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	return rowsAffected, nil
}