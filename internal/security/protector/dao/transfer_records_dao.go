package dao

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// postgresTransferRecordsDAO implements TransferRecordsDAO for PostgreSQL
type postgresTransferRecordsDAO struct {
	*baseDAO
}

// Insert inserts a new transfer record
func (dao *postgresTransferRecordsDAO) Insert(ctx context.Context, transfer *TransferRecord) error {
	query := `
		INSERT INTO transfer_records (
			id, type, amount, currency, from_address, to_address, status,
			trigger_reason, transaction_hash, estimated_fee, actual_fee,
			confirmations, required_confirmations, priority, metadata,
			executed_at, completed_at, updated_at
		) VALUES (
			:id, :type, :amount, :currency, :from_address, :to_address, :status,
			:trigger_reason, :transaction_hash, :estimated_fee, :actual_fee,
			:confirmations, :required_confirmations, :priority, :metadata,
			:executed_at, :completed_at, NOW()
		) RETURNING created_at, updated_at`
	
	rows, err := dao.db.NamedQuery(query, transfer)
	if err != nil {
		return fmt.Errorf("failed to insert transfer record: %w", err)
	}
	defer rows.Close()
	
	if rows.Next() {
		err = rows.Scan(&transfer.CreatedAt, &transfer.UpdatedAt)
		if err != nil {
			return fmt.Errorf("failed to scan inserted transfer record: %w", err)
		}
	}
	
	return nil
}

// Update updates an existing transfer record
func (dao *postgresTransferRecordsDAO) Update(ctx context.Context, transfer *TransferRecord) error {
	query := `
		UPDATE transfer_records SET
			type = :type,
			amount = :amount,
			currency = :currency,
			from_address = :from_address,
			to_address = :to_address,
			status = :status,
			trigger_reason = :trigger_reason,
			transaction_hash = :transaction_hash,
			estimated_fee = :estimated_fee,
			actual_fee = :actual_fee,
			confirmations = :confirmations,
			required_confirmations = :required_confirmations,
			priority = :priority,
			metadata = :metadata,
			executed_at = :executed_at,
			completed_at = :completed_at,
			updated_at = NOW()
		WHERE id = :id`
	
	result, err := dao.db.NamedExecContext(ctx, query, transfer)
	if err != nil {
		return fmt.Errorf("failed to update transfer record: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("transfer record with id %s not found", transfer.ID)
	}
	
	return nil
}

// GetByID retrieves a transfer record by ID
func (dao *postgresTransferRecordsDAO) GetByID(ctx context.Context, id string) (*TransferRecord, error) {
	query := `
		SELECT id, type, amount, currency, from_address, to_address, status,
			   trigger_reason, transaction_hash, estimated_fee, actual_fee,
			   confirmations, required_confirmations, priority, metadata,
			   created_at, updated_at, executed_at, completed_at
		FROM transfer_records
		WHERE id = $1`
	
	var transfer TransferRecord
	err := dao.db.GetContext(ctx, &transfer, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get transfer record by id: %w", err)
	}
	
	return &transfer, nil
}

// GetByStatus retrieves transfer records by status
func (dao *postgresTransferRecordsDAO) GetByStatus(ctx context.Context, status string, limit int) ([]*TransferRecord, error) {
	query := `
		SELECT id, type, amount, currency, from_address, to_address, status,
			   trigger_reason, transaction_hash, estimated_fee, actual_fee,
			   confirmations, required_confirmations, priority, metadata,
			   created_at, updated_at, executed_at, completed_at
		FROM transfer_records
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2`
	
	var transfers []*TransferRecord
	err := dao.db.SelectContext(ctx, &transfers, query, status, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get transfer records by status: %w", err)
	}
	
	return transfers, nil
}

// GetByType retrieves transfer records by type
func (dao *postgresTransferRecordsDAO) GetByType(ctx context.Context, transferType string, limit int) ([]*TransferRecord, error) {
	query := `
		SELECT id, type, amount, currency, from_address, to_address, status,
			   trigger_reason, transaction_hash, estimated_fee, actual_fee,
			   confirmations, required_confirmations, priority, metadata,
			   created_at, updated_at, executed_at, completed_at
		FROM transfer_records
		WHERE type = $1
		ORDER BY created_at DESC
		LIMIT $2`
	
	var transfers []*TransferRecord
	err := dao.db.SelectContext(ctx, &transfers, query, transferType, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get transfer records by type: %w", err)
	}
	
	return transfers, nil
}

// GetRecent retrieves recent transfer records
func (dao *postgresTransferRecordsDAO) GetRecent(ctx context.Context, limit int) ([]*TransferRecord, error) {
	query := `
		SELECT id, type, amount, currency, from_address, to_address, status,
			   trigger_reason, transaction_hash, estimated_fee, actual_fee,
			   confirmations, required_confirmations, priority, metadata,
			   created_at, updated_at, executed_at, completed_at
		FROM transfer_records
		ORDER BY created_at DESC
		LIMIT $1`
	
	var transfers []*TransferRecord
	err := dao.db.SelectContext(ctx, &transfers, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent transfer records: %w", err)
	}
	
	return transfers, nil
}

// GetByDateRange retrieves transfer records within a date range
func (dao *postgresTransferRecordsDAO) GetByDateRange(ctx context.Context, startTime, endTime time.Time) ([]*TransferRecord, error) {
	query := `
		SELECT id, type, amount, currency, from_address, to_address, status,
			   trigger_reason, transaction_hash, estimated_fee, actual_fee,
			   confirmations, required_confirmations, priority, metadata,
			   created_at, updated_at, executed_at, completed_at
		FROM transfer_records
		WHERE created_at >= $1 AND created_at <= $2
		ORDER BY created_at DESC`
	
	var transfers []*TransferRecord
	err := dao.db.SelectContext(ctx, &transfers, query, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get transfer records by date range: %w", err)
	}
	
	return transfers, nil
}

// UpdateStatus updates the status of a transfer record
func (dao *postgresTransferRecordsDAO) UpdateStatus(ctx context.Context, id string, status string) error {
	query := `
		UPDATE transfer_records 
		SET status = $2, updated_at = NOW()
		WHERE id = $1`
	
	result, err := dao.db.ExecContext(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("failed to update transfer status: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("transfer record with id %s not found", id)
	}
	
	return nil
}

// DeleteOlderThan deletes transfer records older than the specified timestamp
func (dao *postgresTransferRecordsDAO) DeleteOlderThan(ctx context.Context, timestamp time.Time) (int64, error) {
	query := `DELETE FROM transfer_records WHERE created_at < $1`
	
	result, err := dao.db.ExecContext(ctx, query, timestamp)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old transfer records: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	return rowsAffected, nil
}