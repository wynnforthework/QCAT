package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"qcat/internal/strategy"
)

// Manager manages strategy state
type Manager struct {
	db        *sql.DB
	states    map[string]*State
	callbacks map[string][]StateCallback
	mu        sync.RWMutex
}

// State represents strategy state
type State struct {
	ID         string                 `json:"id"`
	Strategy   string                 `json:"strategy"`
	Symbol     string                 `json:"symbol"`
	Mode       strategy.Mode          `json:"mode"`
	Status     strategy.State         `json:"status"`
	Params     map[string]interface{} `json:"params"`
	Metadata   map[string]interface{} `json:"metadata"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	StartedAt  time.Time              `json:"started_at"`
	StoppedAt  time.Time              `json:"stopped_at"`
	LastError  string                 `json:"last_error"`
	ErrorCount int                    `json:"error_count"`
}

// StateCallback represents a state change callback function
type StateCallback func(*State)

// NewManager creates a new state manager
func NewManager(db *sql.DB) *Manager {
	return &Manager{
		db:        db,
		states:    make(map[string]*State),
		callbacks: make(map[string][]StateCallback),
	}
}

// CreateState creates a new strategy state
func (m *Manager) CreateState(ctx context.Context, strategy, symbol string, mode strategy.Mode, params map[string]interface{}) (*State, error) {
	state := &State{
		ID:        fmt.Sprintf("%s-%s-%s", strategy, symbol, mode),
		Strategy:  strategy,
		Symbol:    symbol,
		Mode:      mode,
		Status:    "initializing",
		Params:    params,
		Metadata:  make(map[string]interface{}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store state in database
	if err := m.saveState(ctx, state); err != nil {
		return nil, fmt.Errorf("failed to save state: %w", err)
	}

	// Store state in memory
	m.mu.Lock()
	m.states[state.ID] = state
	m.mu.Unlock()

	return state, nil
}

// GetState returns a strategy state by ID
func (m *Manager) GetState(ctx context.Context, id string) (*State, error) {
	m.mu.RLock()
	state, exists := m.states[id]
	m.mu.RUnlock()

	if exists {
		return state, nil
	}

	// Load state from database
	state, err := m.loadState(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	// Store state in memory
	m.mu.Lock()
	m.states[state.ID] = state
	m.mu.Unlock()

	return state, nil
}

// ListStates returns all strategy states
func (m *Manager) ListStates(ctx context.Context) ([]*State, error) {
	// Load states from database
	query := `
		SELECT id, strategy, symbol, mode, status, params, metadata,
			created_at, updated_at, started_at, stopped_at, last_error, error_count
		FROM strategy_states
	`

	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query states: %w", err)
	}
	defer rows.Close()

	var states []*State
	for rows.Next() {
		var state State
		var params, metadata []byte
		var startedAt, stoppedAt sql.NullTime

		if err := rows.Scan(
			&state.ID,
			&state.Strategy,
			&state.Symbol,
			&state.Mode,
			&state.Status,
			&params,
			&metadata,
			&state.CreatedAt,
			&state.UpdatedAt,
			&startedAt,
			&stoppedAt,
			&state.LastError,
			&state.ErrorCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan state: %w", err)
		}

		if err := json.Unmarshal(params, &state.Params); err != nil {
			return nil, fmt.Errorf("failed to unmarshal params: %w", err)
		}

		if err := json.Unmarshal(metadata, &state.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		if startedAt.Valid {
			state.StartedAt = startedAt.Time
		}
		if stoppedAt.Valid {
			state.StoppedAt = stoppedAt.Time
		}

		states = append(states, &state)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating states: %w", err)
	}

	return states, nil
}

// UpdateState updates a strategy state
func (m *Manager) UpdateState(ctx context.Context, state *State) error {
	state.UpdatedAt = time.Now()

	// Store state in database
	if err := m.saveState(ctx, state); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	// Store state in memory
	m.mu.Lock()
	m.states[state.ID] = state
	m.mu.Unlock()

	// Notify callbacks
	m.notifyCallbacks(state)

	return nil
}

// DeleteState deletes a strategy state
func (m *Manager) DeleteState(ctx context.Context, id string) error {
	// Delete state from database
	query := `DELETE FROM strategy_states WHERE id = $1`
	if _, err := m.db.ExecContext(ctx, query, id); err != nil {
		return fmt.Errorf("failed to delete state: %w", err)
	}

	// Delete state from memory
	m.mu.Lock()
	delete(m.states, id)
	delete(m.callbacks, id)
	m.mu.Unlock()

	return nil
}

// AddCallback adds a state change callback
func (m *Manager) AddCallback(id string, callback StateCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callbacks[id] = append(m.callbacks[id], callback)
}

// RemoveCallback removes a state change callback
func (m *Manager) RemoveCallback(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.callbacks, id)
}

// saveState saves a state to the database
func (m *Manager) saveState(ctx context.Context, state *State) error {
	params, err := json.Marshal(state.Params)
	if err != nil {
		return fmt.Errorf("failed to marshal params: %w", err)
	}

	metadata, err := json.Marshal(state.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO strategy_states (
			id, strategy, symbol, mode, status, params, metadata,
			created_at, updated_at, started_at, stopped_at, last_error, error_count
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		) ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			params = EXCLUDED.params,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at,
			started_at = EXCLUDED.started_at,
			stopped_at = EXCLUDED.stopped_at,
			last_error = EXCLUDED.last_error,
			error_count = EXCLUDED.error_count
	`

	_, err = m.db.ExecContext(ctx, query,
		state.ID,
		state.Strategy,
		state.Symbol,
		state.Mode,
		state.Status,
		params,
		metadata,
		state.CreatedAt,
		state.UpdatedAt,
		sql.NullTime{Time: state.StartedAt, Valid: !state.StartedAt.IsZero()},
		sql.NullTime{Time: state.StoppedAt, Valid: !state.StoppedAt.IsZero()},
		state.LastError,
		state.ErrorCount,
	)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	return nil
}

// loadState loads a state from the database
func (m *Manager) loadState(ctx context.Context, id string) (*State, error) {
	query := `
		SELECT id, strategy, symbol, mode, status, params, metadata,
			created_at, updated_at, started_at, stopped_at, last_error, error_count
		FROM strategy_states
		WHERE id = $1
	`

	var state State
	var params, metadata []byte
	var startedAt, stoppedAt sql.NullTime

	if err := m.db.QueryRowContext(ctx, query, id).Scan(
		&state.ID,
		&state.Strategy,
		&state.Symbol,
		&state.Mode,
		&state.Status,
		&params,
		&metadata,
		&state.CreatedAt,
		&state.UpdatedAt,
		&startedAt,
		&stoppedAt,
		&state.LastError,
		&state.ErrorCount,
	); err != nil {
		return nil, fmt.Errorf("failed to scan state: %w", err)
	}

	if err := json.Unmarshal(params, &state.Params); err != nil {
		return nil, fmt.Errorf("failed to unmarshal params: %w", err)
	}

	if err := json.Unmarshal(metadata, &state.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	if startedAt.Valid {
		state.StartedAt = startedAt.Time
	}
	if stoppedAt.Valid {
		state.StoppedAt = stoppedAt.Time
	}

	return &state, nil
}

// notifyCallbacks notifies state change callbacks
func (m *Manager) notifyCallbacks(state *State) {
	if callbacks, exists := m.callbacks[state.ID]; exists {
		for _, callback := range callbacks {
			callback(state)
		}
	}
}
