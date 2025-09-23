package automation

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"qcat/internal/strategy/lifecycle"
	"qcat/internal/strategy/validation"
)

// BacktestScheduler 自动化回测调度器
type BacktestScheduler struct {
	db               *sql.DB
	validator        *validation.MandatoryBacktestValidator
	gatekeeper       *validation.StrategyGatekeeper
	scheduleInterval time.Duration
	running          bool
	stopChan         chan struct{}
}

// BacktestTask 回测任务
type BacktestTask struct {
	StrategyID    string    `json:"strategy_id"`
	TaskType      string    `json:"task_type"` // "mandatory", "periodic", "validation"
	Priority      int       `json:"priority"`  // 1=highest, 5=lowest
	ScheduledTime time.Time `json:"scheduled_time"`
	Status        string    `json:"status"` // "pending", "running", "completed", "failed"
	CreatedAt     time.Time `json:"created_at"`
}

// NewBacktestScheduler 创建回测调度器
func NewBacktestScheduler(db *sql.DB, gatekeeper *validation.StrategyGatekeeper) *BacktestScheduler {
	defaultGatekeeper := gatekeeper
	if defaultGatekeeper == nil {
		defaultGatekeeper = validation.NewStrategyGatekeeper()
	}

	return &BacktestScheduler{
		db:               db,
		validator:        validation.NewMandatoryBacktestValidator(),
		gatekeeper:       defaultGatekeeper,
		scheduleInterval: 1 * time.Hour, // 每小时检查一次
		stopChan:         make(chan struct{}),
	}
}

// Start 启动调度器
func (bs *BacktestScheduler) Start(ctx context.Context) error {
	if bs.running {
		return fmt.Errorf("backtest scheduler is already running")
	}

	bs.running = true
	log.Printf("🔄 自动化回测调度器启动")

	// 立即执行一次检查
	go bs.scheduleLoop(ctx)

	return nil
}

// Stop 停止调度器
func (bs *BacktestScheduler) Stop() {
	if !bs.running {
		return
	}

	bs.running = false
	close(bs.stopChan)
	log.Printf("自动化回测调度器已停止")
}

// scheduleLoop 调度循环
func (bs *BacktestScheduler) scheduleLoop(ctx context.Context) {
	// 启动时立即检查一次
	if err := bs.checkAndScheduleBacktests(ctx); err != nil {
		log.Printf("初始回测检查失败: %v", err)
	}

	ticker := time.NewTicker(bs.scheduleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-bs.stopChan:
			return
		case <-ticker.C:
			if err := bs.checkAndScheduleBacktests(ctx); err != nil {
				log.Printf("回测调度检查失败: %v", err)
			}
		}
	}
}

// checkAndScheduleBacktests 检查并调度回测
func (bs *BacktestScheduler) checkAndScheduleBacktests(ctx context.Context) error {
	log.Printf("🔍 检查需要回测的策略...")

	// 1. 查找所有需要强制回测的策略
	strategies, err := bs.findStrategiesNeedingBacktest(ctx)
	if err != nil {
		return fmt.Errorf("查找需要回测的策略失败: %w", err)
	}

	log.Printf("发现 %d 个策略需要回测", len(strategies))

	// 2. 为每个策略创建回测任务
	for _, strategyID := range strategies {
		if err := bs.scheduleBacktestTask(ctx, strategyID, "mandatory", 1); err != nil {
			log.Printf("为策略 %s 调度回测任务失败: %v", strategyID, err)
		}
	}

	// 3. 执行待处理的回测任务
	if err := bs.processPendingTasks(ctx); err != nil {
		log.Printf("处理待处理任务失败: %v", err)
	}

	return nil
}

// findStrategiesNeedingBacktest 查找需要回测的策略
func (bs *BacktestScheduler) findStrategiesNeedingBacktest(ctx context.Context) ([]string, error) {
	// 修复ORDER BY和SELECT DISTINCT问题：ORDER BY字段必须出现在SELECT DISTINCT列表中
	query := `
		SELECT DISTINCT s.id, s.updated_at
		FROM strategies s
		LEFT JOIN backtest_results br ON s.id = br.strategy_id
		WHERE (
			-- 策略从未回测过
			br.strategy_id IS NULL
			-- 或者策略已启用但没有有效的回测结果
			OR (s.is_running = true AND (br.is_valid = false OR br.is_valid IS NULL))
			-- 或者回测结果过期（超过30天）
			OR (br.created_at < NOW() - INTERVAL '30 days')
			-- 或者策略配置已更新但未重新回测
			OR (s.updated_at > br.created_at)
		)
		AND s.status != 'disabled'
		ORDER BY s.updated_at DESC
	`

	rows, err := bs.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var strategies []string
	for rows.Next() {
		var strategyID string
		var updatedAt time.Time
		if err := rows.Scan(&strategyID, &updatedAt); err != nil {
			continue
		}
		strategies = append(strategies, strategyID)
	}

	return strategies, nil
}

// scheduleBacktestTask 调度回测任务
func (bs *BacktestScheduler) scheduleBacktestTask(ctx context.Context, strategyID, taskType string, priority int) error {
	// 检查是否已有待处理的任务
	existingQuery := `
		SELECT COUNT(*) FROM backtest_tasks 
		WHERE strategy_id = $1 AND status IN ('pending', 'running')
	`

	var count int
	if err := bs.db.QueryRowContext(ctx, existingQuery, strategyID).Scan(&count); err != nil {
		return err
	}

	if count > 0 {
		log.Printf("策略 %s 已有待处理的回测任务，跳过", strategyID)
		return nil
	}

	// 创建新的回测任务
	insertQuery := `
		INSERT INTO backtest_tasks (strategy_id, task_type, priority, scheduled_time, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	now := time.Now()
	_, err := bs.db.ExecContext(ctx, insertQuery,
		strategyID, taskType, priority, now, "pending", now)

	if err != nil {
		return err
	}

	log.Printf("✅ 为策略 %s 创建了 %s 回测任务", strategyID, taskType)
	return nil
}

// processPendingTasks 处理待处理的任务
func (bs *BacktestScheduler) processPendingTasks(ctx context.Context) error {
	// 获取待处理的任务（按优先级排序）
	query := `
		SELECT strategy_id, task_type, priority
		FROM backtest_tasks 
		WHERE status = 'pending' AND scheduled_time <= $1
		ORDER BY priority ASC, created_at ASC
		LIMIT 5
	`

	rows, err := bs.db.QueryContext(ctx, query, time.Now())
	if err != nil {
		return err
	}
	defer rows.Close()

	var tasks []BacktestTask
	for rows.Next() {
		var task BacktestTask
		if err := rows.Scan(&task.StrategyID, &task.TaskType, &task.Priority); err != nil {
			continue
		}
		tasks = append(tasks, task)
	}

	// 处理每个任务
	for _, task := range tasks {
		if err := bs.executeBacktestTask(ctx, &task); err != nil {
			log.Printf("执行回测任务失败 (策略: %s): %v", task.StrategyID, err)
		}
	}

	return nil
}

// executeBacktestTask 执行回测任务
func (bs *BacktestScheduler) executeBacktestTask(ctx context.Context, task *BacktestTask) error {
	log.Printf("🚀 开始执行策略 %s 的回测任务", task.StrategyID)

	// 1. 更新任务状态为运行中
	if err := bs.updateTaskStatus(ctx, task.StrategyID, "running"); err != nil {
		return err
	}

	// 2. 创建策略配置（从数据库获取）
	config := &lifecycle.Version{
		ID:         task.StrategyID,
		StrategyID: task.StrategyID,
		State:      lifecycle.StateDraft,
	}

	// 3. 执行强制回测验证
	if bs.gatekeeper != nil {
		auditResult, auditErr := bs.gatekeeper.RunScenarioAudit(ctx, task.StrategyID, config)
		if auditErr != nil {
			log.Printf("Scenario audit failed for strategy %s: %v", task.StrategyID, auditErr)
			bs.updateTaskStatus(ctx, task.StrategyID, "failed")
			return fmt.Errorf("scenario audit failed: %w", auditErr)
		}
		if auditResult != nil && !auditResult.Passed {
			summary := strings.Join(auditResult.FailureSummary, "; ")
			if summary == "" {
				summary = "scenario checks did not pass"
			}
			log.Printf("Scenario audit rejected strategy %s: %s", task.StrategyID, summary)
			bs.updateTaskStatus(ctx, task.StrategyID, "failed")
			return fmt.Errorf("scenario audit failed: %s", summary)
		}
	}

	result, err := bs.validator.ValidateStrategy(ctx, task.StrategyID, config)
	if err != nil {
		log.Printf("策略 %s 回测失败: %v", task.StrategyID, err)
		bs.updateTaskStatus(ctx, task.StrategyID, "failed")
		return err
	}

	// 4. 保存回测结果到数据库
	if err := bs.saveBacktestResult(ctx, task.StrategyID, result); err != nil {
		log.Printf("保存回测结果失败: %v", err)
	}

	// 5. 更新任务状态
	if result.IsValid {
		log.Printf("✅ 策略 %s 回测通过", task.StrategyID)
		bs.updateTaskStatus(ctx, task.StrategyID, "completed")
	} else {
		log.Printf("❌ 策略 %s 回测未通过: %v", task.StrategyID, result.FailureReasons)
		bs.updateTaskStatus(ctx, task.StrategyID, "failed")

		// 如果策略正在运行但回测失败，停止策略
		if err := bs.stopFailedStrategy(ctx, task.StrategyID, result.FailureReasons); err != nil {
			log.Printf("停止失败策略时出错: %v", err)
		}
	}

	return nil
}

// updateTaskStatus 更新任务状态
func (bs *BacktestScheduler) updateTaskStatus(ctx context.Context, strategyID, status string) error {
	query := `
		UPDATE backtest_tasks 
		SET status = $1, updated_at = $2
		WHERE strategy_id = $3 AND status != 'completed'
	`

	_, err := bs.db.ExecContext(ctx, query, status, time.Now(), strategyID)
	return err
}

// saveBacktestResult 保存回测结果
func (bs *BacktestScheduler) saveBacktestResult(ctx context.Context, strategyID string, result *validation.BacktestResult) error {
	query := `
		INSERT INTO backtest_results (
			strategy_id, total_return, sharpe_ratio, max_drawdown, win_rate,
			total_trades, backtest_days, is_valid, failure_reasons, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (strategy_id) DO UPDATE SET
			total_return = EXCLUDED.total_return,
			sharpe_ratio = EXCLUDED.sharpe_ratio,
			max_drawdown = EXCLUDED.max_drawdown,
			win_rate = EXCLUDED.win_rate,
			total_trades = EXCLUDED.total_trades,
			backtest_days = EXCLUDED.backtest_days,
			is_valid = EXCLUDED.is_valid,
			failure_reasons = EXCLUDED.failure_reasons,
			created_at = EXCLUDED.created_at
	`

	failureReasonsJSON := ""
	if len(result.FailureReasons) > 0 {
		// 简单的字符串连接，实际应该用JSON
		for i, reason := range result.FailureReasons {
			if i > 0 {
				failureReasonsJSON += "; "
			}
			failureReasonsJSON += reason
		}
	}

	_, err := bs.db.ExecContext(ctx, query,
		strategyID, result.TotalReturn, result.SharpeRatio, result.MaxDrawdown,
		result.WinRate, result.TotalTrades, result.BacktestDays, result.IsValid,
		failureReasonsJSON, time.Now())

	return err
}

// stopFailedStrategy 停止失败的策略
func (bs *BacktestScheduler) stopFailedStrategy(ctx context.Context, strategyID string, reasons []string) error {
	// 检查策略是否正在运行
	var isRunning bool
	checkQuery := `SELECT is_running FROM strategies WHERE id = $1`
	if err := bs.db.QueryRowContext(ctx, checkQuery, strategyID).Scan(&isRunning); err != nil {
		return err
	}

	if !isRunning {
		return nil // 策略已经停止
	}

	// 停止策略
	reason := "回测验证失败"
	if len(reasons) > 0 {
		reason = reasons[0]
	}

	stopQuery := `
		UPDATE strategies 
		SET is_running = false, status = 'stopped', stop_reason = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := bs.db.ExecContext(ctx, stopQuery, reason, time.Now(), strategyID)
	if err != nil {
		return err
	}

	log.Printf("🛑 已停止未通过回测的策略: %s (原因: %s)", strategyID, reason)
	return nil
}

// GetSchedulerStatus 获取调度器状态
func (bs *BacktestScheduler) GetSchedulerStatus(ctx context.Context) (map[string]interface{}, error) {
	// 统计任务状态
	statusQuery := `
		SELECT status, COUNT(*) 
		FROM backtest_tasks 
		WHERE created_at > NOW() - INTERVAL '24 hours'
		GROUP BY status
	`

	rows, err := bs.db.QueryContext(ctx, statusQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	statusCounts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}
		statusCounts[status] = count
	}

	return map[string]interface{}{
		"running":           bs.running,
		"schedule_interval": bs.scheduleInterval.String(),
		"task_counts_24h":   statusCounts,
		"last_check":        time.Now(),
	}, nil
}
