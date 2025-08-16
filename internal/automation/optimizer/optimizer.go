package optimizer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"qcat/internal/strategy/backtest"
)

// Optimizer manages strategy parameter optimization
type Optimizer struct {
	db          *sql.DB
	tasks       map[string]*Task
	subscribers map[string][]TaskCallback
	mu          sync.RWMutex
}

// Task represents an optimization task
type Task struct {
	ID          string                 `json:"id"`
	Strategy    string                 `json:"strategy"`
	Symbol      string                 `json:"symbol"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     time.Time              `json:"end_time"`
	Status      TaskStatus             `json:"status"`
	Parameters  []Parameter            `json:"parameters"`
	Objective   Objective              `json:"objective"`
	Constraints []Constraint           `json:"constraints"`
	Results     []Result               `json:"results"`
	BestResult  *Result                `json:"best_result"`
	Progress    float64                `json:"progress"`
	Error       string                 `json:"error"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// TaskStatus represents the status of an optimization task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

// Parameter represents a strategy parameter to optimize
type Parameter struct {
	Name    string      `json:"name"`
	Type    string      `json:"type"` // "float", "int", "bool", "string"
	Min     float64     `json:"min,omitempty"`
	Max     float64     `json:"max,omitempty"`
	Step    float64     `json:"step,omitempty"`
	Choices []string    `json:"choices,omitempty"`
	Current interface{} `json:"current"`
	Best    interface{} `json:"best"`
}

// Objective represents the optimization objective
type Objective struct {
	Metric    string  `json:"metric"`    // "pnl", "sharpe", "sortino", "calmar", etc.
	Direction string  `json:"direction"` // "maximize", "minimize"
	Weight    float64 `json:"weight"`
}

// Constraint represents an optimization constraint
type Constraint struct {
	Metric string  `json:"metric"`
	Min    float64 `json:"min,omitempty"`
	Max    float64 `json:"max,omitempty"`
}

// Result represents an optimization result
type Result struct {
	Parameters map[string]interface{} `json:"parameters"`
	Metrics    map[string]float64     `json:"metrics"`
	Score      float64                `json:"score"`
	CreatedAt  time.Time              `json:"created_at"`
}

// TaskCallback represents a task callback function
type TaskCallback func(*Task)

// NewOptimizer creates a new optimizer
func NewOptimizer(db *sql.DB) *Optimizer {
	return &Optimizer{
		db:          db,
		tasks:       make(map[string]*Task),
		subscribers: make(map[string][]TaskCallback),
	}
}

// CreateTask creates a new optimization task
func (o *Optimizer) CreateTask(ctx context.Context, strategy, symbol string, startTime, endTime time.Time, params []Parameter, objective Objective, constraints []Constraint) (*Task, error) {
	task := &Task{
		ID:          fmt.Sprintf("%s-%s-%d", strategy, symbol, time.Now().UnixNano()),
		Strategy:    strategy,
		Symbol:      symbol,
		StartTime:   startTime,
		EndTime:     endTime,
		Status:      TaskStatusPending,
		Parameters:  params,
		Objective:   objective,
		Constraints: constraints,
		Results:     make([]Result, 0),
		Metadata:    make(map[string]interface{}),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Store task in database
	if err := o.saveTask(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to save task: %w", err)
	}

	// Store task in memory
	o.mu.Lock()
	o.tasks[task.ID] = task
	o.mu.Unlock()

	// Start optimization
	go o.optimize(ctx, task)

	return task, nil
}

// GetTask returns a task by ID
func (o *Optimizer) GetTask(ctx context.Context, id string) (*Task, error) {
	o.mu.RLock()
	task, exists := o.tasks[id]
	o.mu.RUnlock()

	if exists {
		return task, nil
	}

	// Load task from database
	task, err := o.loadTask(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load task: %w", err)
	}

	// Store task in memory
	o.mu.Lock()
	o.tasks[task.ID] = task
	o.mu.Unlock()

	return task, nil
}

// ListTasks returns all tasks
func (o *Optimizer) ListTasks(ctx context.Context) ([]*Task, error) {
	// Load tasks from database
	query := `
		SELECT id, strategy, symbol, start_time, end_time, status, parameters,
			objective, constraints, results, best_result, progress, error, metadata,
			created_at, updated_at
		FROM optimizer_tasks
	`

	rows, err := o.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		var task Task
		var params, obj, cons, results, best, meta []byte

		if err := rows.Scan(
			&task.ID,
			&task.Strategy,
			&task.Symbol,
			&task.StartTime,
			&task.EndTime,
			&task.Status,
			&params,
			&obj,
			&cons,
			&results,
			&best,
			&task.Progress,
			&task.Error,
			&meta,
			&task.CreatedAt,
			&task.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		if err := json.Unmarshal(params, &task.Parameters); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}

		if err := json.Unmarshal(obj, &task.Objective); err != nil {
			return nil, fmt.Errorf("failed to unmarshal objective: %w", err)
		}

		if err := json.Unmarshal(cons, &task.Constraints); err != nil {
			return nil, fmt.Errorf("failed to unmarshal constraints: %w", err)
		}

		if err := json.Unmarshal(results, &task.Results); err != nil {
			return nil, fmt.Errorf("failed to unmarshal results: %w", err)
		}

		if best != nil {
			if err := json.Unmarshal(best, &task.BestResult); err != nil {
				return nil, fmt.Errorf("failed to unmarshal best result: %w", err)
			}
		}

		if err := json.Unmarshal(meta, &task.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		tasks = append(tasks, &task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tasks: %w", err)
	}

	return tasks, nil
}

// Subscribe subscribes to task updates
func (o *Optimizer) Subscribe(taskID string, callback TaskCallback) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.subscribers[taskID] = append(o.subscribers[taskID], callback)
}

// Unsubscribe removes a task subscription
func (o *Optimizer) Unsubscribe(taskID string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	delete(o.subscribers, taskID)
}

// saveTask saves a task to the database
func (o *Optimizer) saveTask(ctx context.Context, task *Task) error {
	params, err := json.Marshal(task.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	obj, err := json.Marshal(task.Objective)
	if err != nil {
		return fmt.Errorf("failed to marshal objective: %w", err)
	}

	cons, err := json.Marshal(task.Constraints)
	if err != nil {
		return fmt.Errorf("failed to marshal constraints: %w", err)
	}

	results, err := json.Marshal(task.Results)
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	var best []byte
	if task.BestResult != nil {
		best, err = json.Marshal(task.BestResult)
		if err != nil {
			return fmt.Errorf("failed to marshal best result: %w", err)
		}
	}

	meta, err := json.Marshal(task.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO optimizer_tasks (
			id, strategy, symbol, start_time, end_time, status, parameters,
			objective, constraints, results, best_result, progress, error, metadata,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		) ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			parameters = EXCLUDED.parameters,
			results = EXCLUDED.results,
			best_result = EXCLUDED.best_result,
			progress = EXCLUDED.progress,
			error = EXCLUDED.error,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at
	`

	_, err = o.db.ExecContext(ctx, query,
		task.ID,
		task.Strategy,
		task.Symbol,
		task.StartTime,
		task.EndTime,
		task.Status,
		params,
		obj,
		cons,
		results,
		best,
		task.Progress,
		task.Error,
		meta,
		task.CreatedAt,
		task.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	return nil
}

// loadTask loads a task from the database
func (o *Optimizer) loadTask(ctx context.Context, id string) (*Task, error) {
	query := `
		SELECT id, strategy, symbol, start_time, end_time, status, parameters,
			objective, constraints, results, best_result, progress, error, metadata,
			created_at, updated_at
		FROM optimizer_tasks
		WHERE id = $1
	`

	var task Task
	var params, obj, cons, results, best, meta []byte

	if err := o.db.QueryRowContext(ctx, query, id).Scan(
		&task.ID,
		&task.Strategy,
		&task.Symbol,
		&task.StartTime,
		&task.EndTime,
		&task.Status,
		&params,
		&obj,
		&cons,
		&results,
		&best,
		&task.Progress,
		&task.Error,
		&meta,
		&task.CreatedAt,
		&task.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("failed to scan task: %w", err)
	}

	if err := json.Unmarshal(params, &task.Parameters); err != nil {
		return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
	}

	if err := json.Unmarshal(obj, &task.Objective); err != nil {
		return nil, fmt.Errorf("failed to unmarshal objective: %w", err)
	}

	if err := json.Unmarshal(cons, &task.Constraints); err != nil {
		return nil, fmt.Errorf("failed to unmarshal constraints: %w", err)
	}

	if err := json.Unmarshal(results, &task.Results); err != nil {
		return nil, fmt.Errorf("failed to unmarshal results: %w", err)
	}

	if best != nil {
		if err := json.Unmarshal(best, &task.BestResult); err != nil {
			return nil, fmt.Errorf("failed to unmarshal best result: %w", err)
		}
	}

	if err := json.Unmarshal(meta, &task.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &task, nil
}

// optimize runs the optimization process
func (o *Optimizer) optimize(ctx context.Context, task *Task) {
	// Update task status
	task.Status = TaskStatusRunning
	task.UpdatedAt = time.Now()
	if err := o.saveTask(ctx, task); err != nil {
		task.Status = TaskStatusFailed
		task.Error = fmt.Sprintf("failed to save task: %v", err)
		task.UpdatedAt = time.Now()
		o.saveTask(ctx, task)
		return
	}

	// Create backtest engine
	engine := backtest.NewEngine(nil, nil, nil)

	// Run optimization iterations
	numIterations := 100
	for i := 0; i < numIterations; i++ {
		// Generate parameters
		params := o.generateParameters(task.Parameters)

		// Run backtest
		result := &backtest.Result{}
		result, err := engine.Run(ctx)
		if err != nil {
			task.Status = TaskStatusFailed
			task.Error = fmt.Sprintf("failed to run backtest: %v", err)
			task.UpdatedAt = time.Now()
			o.saveTask(ctx, task)
			return
		}

		// Calculate metrics
		metrics := o.calculateMetrics(result)

		// Check constraints
		if !o.checkConstraints(metrics, task.Constraints) {
			continue
		}

		// Calculate score
		score := o.calculateScore(metrics, task.Objective)

		// Store result
		taskResult := Result{
			Parameters: params,
			Metrics:    metrics,
			Score:      score,
			CreatedAt:  time.Now(),
		}
		task.Results = append(task.Results, taskResult)

		// Update best result
		if task.BestResult == nil || (task.Objective.Direction == "maximize" && score > task.BestResult.Score) ||
			(task.Objective.Direction == "minimize" && score < task.BestResult.Score) {
			task.BestResult = &taskResult
			for i := range task.Parameters {
				task.Parameters[i].Best = params[task.Parameters[i].Name]
			}
		}

		// Update progress
		task.Progress = float64(i+1) / float64(numIterations) * 100
		task.UpdatedAt = time.Now()

		// Save task
		if err := o.saveTask(ctx, task); err != nil {
			task.Status = TaskStatusFailed
			task.Error = fmt.Sprintf("failed to save task: %v", err)
			task.UpdatedAt = time.Now()
			o.saveTask(ctx, task)
			return
		}

		// Notify subscribers
		o.notifySubscribers(task)
	}

	// Update task status
	task.Status = TaskStatusCompleted
	task.UpdatedAt = time.Now()
	o.saveTask(ctx, task)
}

// generateParameters generates random parameters within constraints
func (o *Optimizer) generateParameters(params []Parameter) map[string]interface{} {
	result := make(map[string]interface{})

	for _, param := range params {
		switch param.Type {
		case "float":
			value := param.Min + rand.Float64()*(param.Max-param.Min)
			value = math.Round(value/param.Step) * param.Step
			result[param.Name] = value

		case "int":
			value := param.Min + rand.Float64()*(param.Max-param.Min)
			result[param.Name] = int(math.Round(value/param.Step) * param.Step)

		case "bool":
			result[param.Name] = rand.Float64() < 0.5

		case "string":
			if len(param.Choices) > 0 {
				result[param.Name] = param.Choices[rand.Intn(len(param.Choices))]
			}
		}
	}

	return result
}

// calculateMetrics calculates metrics from backtest results
func (o *Optimizer) calculateMetrics(result *backtest.Result) map[string]float64 {
	metrics := make(map[string]float64)

	// Calculate PnL metrics from PerformanceStats
	if result.PerformanceStats != nil {
		metrics["total_return"] = result.PerformanceStats.TotalReturn
		metrics["annual_return"] = result.PerformanceStats.AnnualReturn
		metrics["pnl"] = result.PerformanceStats.TotalReturn               // 使用TotalReturn作为PnL
		metrics["pnl_percent"] = result.PerformanceStats.TotalReturn * 100 // 转换为百分比
		metrics["max_drawdown"] = result.PerformanceStats.MaxDrawdown
		metrics["sharpe_ratio"] = result.PerformanceStats.SharpeRatio
		metrics["win_rate"] = result.PerformanceStats.WinRate
		metrics["num_trades"] = float64(result.PerformanceStats.TradeCount)
		metrics["profit_factor"] = result.PerformanceStats.ProfitFactor
		metrics["avg_trade_return"] = result.PerformanceStats.AvgTradeReturn
	}

	return metrics
}

// checkConstraints checks if metrics satisfy constraints
func (o *Optimizer) checkConstraints(metrics map[string]float64, constraints []Constraint) bool {
	for _, constraint := range constraints {
		value, exists := metrics[constraint.Metric]
		if !exists {
			continue
		}

		if constraint.Min != 0 && value < constraint.Min {
			return false
		}
		if constraint.Max != 0 && value > constraint.Max {
			return false
		}
	}

	return true
}

// calculateScore calculates the optimization score
func (o *Optimizer) calculateScore(metrics map[string]float64, objective Objective) float64 {
	value, exists := metrics[objective.Metric]
	if !exists {
		return 0
	}

	score := value * objective.Weight
	if objective.Direction == "minimize" {
		score = -score
	}

	return score
}

// notifySubscribers notifies task subscribers
func (o *Optimizer) notifySubscribers(task *Task) {
	o.mu.RLock()
	callbacks := o.subscribers[task.ID]
	o.mu.RUnlock()

	for _, callback := range callbacks {
		callback(task)
	}
}
