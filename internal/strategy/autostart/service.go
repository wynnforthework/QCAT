package autostart

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"qcat/internal/strategy/validation"
)

// AutoStartService 策略自动启动服务
type AutoStartService struct {
	db         *sql.DB
	gatekeeper *validation.StrategyGatekeeper
	
	// 配置
	config *AutoStartConfig
	
	// 运行状态
	ctx       context.Context
	cancel    context.CancelFunc
	isRunning bool
	mu        sync.RWMutex
	
	// 统计信息
	stats *AutoStartStats
}

// AutoStartConfig 自动启动配置
type AutoStartConfig struct {
	// 启动配置
	Enabled                bool          `yaml:"enabled"`
	StartupDelay          time.Duration `yaml:"startup_delay"`           // 系统启动后延迟时间
	MaxConcurrentStartups int           `yaml:"max_concurrent_startups"` // 最大并发启动数
	StartupTimeout        time.Duration `yaml:"startup_timeout"`         // 单个策略启动超时
	
	// 重试配置
	MaxRetries    int           `yaml:"max_retries"`
	RetryDelay    time.Duration `yaml:"retry_delay"`
	RetryBackoff  float64       `yaml:"retry_backoff"`
	
	// 验证配置
	RequireValidation     bool `yaml:"require_validation"`      // 是否需要验证
	SkipFailedValidation  bool `yaml:"skip_failed_validation"`  // 跳过验证失败的策略
	
	// 安全配置
	MaxAutoStartPerBatch  int           `yaml:"max_auto_start_per_batch"` // 每批最大自动启动数
	BatchInterval         time.Duration `yaml:"batch_interval"`           // 批次间隔
}

// AutoStartStats 自动启动统计
type AutoStartStats struct {
	TotalAttempts    int       `json:"total_attempts"`
	SuccessfulStarts int       `json:"successful_starts"`
	FailedStarts     int       `json:"failed_starts"`
	SkippedStarts    int       `json:"skipped_starts"`
	LastRunTime      time.Time `json:"last_run_time"`
	LastRunDuration  time.Duration `json:"last_run_duration"`
}

// StrategyStartInfo 策略启动信息
type StrategyStartInfo struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Type            string    `json:"type"`
	Status          string    `json:"status"`
	Enabled         bool      `json:"enabled"`
	AutoStart       bool      `json:"auto_start"`
	StartupPriority int       `json:"startup_priority"`
	LastAutoStart   *time.Time `json:"last_auto_start"`
	IsRunning       bool      `json:"is_running"`
}

// NewAutoStartService 创建自动启动服务
func NewAutoStartService(db *sql.DB, config *AutoStartConfig) *AutoStartService {
	if config == nil {
		config = GetDefaultAutoStartConfig()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &AutoStartService{
		db:         db,
		gatekeeper: validation.NewStrategyGatekeeper(),
		config:     config,
		ctx:        ctx,
		cancel:     cancel,
		stats:      &AutoStartStats{},
	}
}

// Start 启动自动启动服务
func (ass *AutoStartService) Start() error {
	ass.mu.Lock()
	defer ass.mu.Unlock()
	
	if ass.isRunning {
		return fmt.Errorf("auto start service is already running")
	}
	
	if !ass.config.Enabled {
		log.Println("策略自动启动服务已禁用")
		return nil
	}
	
	log.Println("启动策略自动启动服务...")
	
	// 延迟启动，等待系统完全初始化
	go func() {
		time.Sleep(ass.config.StartupDelay)
		if err := ass.performAutoStart(); err != nil {
			log.Printf("自动启动策略失败: %v", err)
		}
	}()
	
	ass.isRunning = true
	log.Println("策略自动启动服务启动完成")
	
	return nil
}

// Stop 停止自动启动服务
func (ass *AutoStartService) Stop() {
	ass.mu.Lock()
	defer ass.mu.Unlock()
	
	if !ass.isRunning {
		return
	}
	
	log.Println("停止策略自动启动服务...")
	ass.cancel()
	ass.isRunning = false
	log.Println("策略自动启动服务已停止")
}

// performAutoStart 执行自动启动
func (ass *AutoStartService) performAutoStart() error {
	startTime := time.Now()
	defer func() {
		ass.stats.LastRunTime = startTime
		ass.stats.LastRunDuration = time.Since(startTime)
	}()
	
	log.Println("开始执行策略自动启动...")
	
	// 获取需要自动启动的策略
	strategies, err := ass.getAutoStartStrategies()
	if err != nil {
		return fmt.Errorf("获取自动启动策略失败: %w", err)
	}
	
	if len(strategies) == 0 {
		log.Println("没有需要自动启动的策略")
		return nil
	}
	
	log.Printf("找到 %d 个需要自动启动的策略", len(strategies))
	
	// 按优先级排序
	sort.Slice(strategies, func(i, j int) bool {
		return strategies[i].StartupPriority < strategies[j].StartupPriority
	})
	
	// 分批启动策略
	return ass.startStrategiesInBatches(strategies)
}

// getAutoStartStrategies 获取需要自动启动的策略
func (ass *AutoStartService) getAutoStartStrategies() ([]*StrategyStartInfo, error) {
	query := `
		SELECT id, name, type, status, enabled, auto_start, 
		       startup_priority, last_auto_start, is_running
		FROM strategies
		WHERE auto_start = true 
		  AND enabled = true 
		  AND is_running = false
		ORDER BY startup_priority ASC, name ASC
	`
	
	rows, err := ass.db.QueryContext(ass.ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询自动启动策略失败: %w", err)
	}
	defer rows.Close()
	
	var strategies []*StrategyStartInfo
	for rows.Next() {
		strategy := &StrategyStartInfo{}
		var lastAutoStart sql.NullTime
		
		err := rows.Scan(
			&strategy.ID,
			&strategy.Name,
			&strategy.Type,
			&strategy.Status,
			&strategy.Enabled,
			&strategy.AutoStart,
			&strategy.StartupPriority,
			&lastAutoStart,
			&strategy.IsRunning,
		)
		if err != nil {
			log.Printf("扫描策略信息失败: %v", err)
			continue
		}
		
		if lastAutoStart.Valid {
			strategy.LastAutoStart = &lastAutoStart.Time
		}
		
		strategies = append(strategies, strategy)
	}
	
	return strategies, nil
}

// startStrategiesInBatches 分批启动策略
func (ass *AutoStartService) startStrategiesInBatches(strategies []*StrategyStartInfo) error {
	batchSize := ass.config.MaxAutoStartPerBatch
	if batchSize <= 0 {
		batchSize = 3 // 默认每批3个
	}
	
	for i := 0; i < len(strategies); i += batchSize {
		end := i + batchSize
		if end > len(strategies) {
			end = len(strategies)
		}
		
		batch := strategies[i:end]
		log.Printf("启动第 %d 批策略 (%d-%d)", (i/batchSize)+1, i+1, end)
		
		// 并发启动当前批次的策略
		if err := ass.startBatch(batch); err != nil {
			log.Printf("批次启动失败: %v", err)
			// 继续下一批次
		}
		
		// 批次间隔
		if i+batchSize < len(strategies) {
			time.Sleep(ass.config.BatchInterval)
		}
	}
	
	log.Printf("策略自动启动完成。成功: %d, 失败: %d, 跳过: %d", 
		ass.stats.SuccessfulStarts, ass.stats.FailedStarts, ass.stats.SkippedStarts)
	
	return nil
}

// startBatch 启动一批策略
func (ass *AutoStartService) startBatch(strategies []*StrategyStartInfo) error {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, ass.config.MaxConcurrentStartups)
	
	for _, strategy := range strategies {
		wg.Add(1)
		go func(s *StrategyStartInfo) {
			defer wg.Done()
			
			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			ass.startSingleStrategy(s)
		}(strategy)
	}
	
	wg.Wait()
	return nil
}

// startSingleStrategy 启动单个策略
func (ass *AutoStartService) startSingleStrategy(strategy *StrategyStartInfo) {
	ass.stats.TotalAttempts++
	
	log.Printf("正在启动策略: %s (%s)", strategy.Name, strategy.ID)
	
	// 创建超时上下文
	ctx, cancel := context.WithTimeout(ass.ctx, ass.config.StartupTimeout)
	defer cancel()
	
	// 验证策略（如果需要）
	if ass.config.RequireValidation {
		if err := ass.validateStrategy(ctx, strategy); err != nil {
			if ass.config.SkipFailedValidation {
				log.Printf("策略 %s 验证失败，跳过启动: %v", strategy.Name, err)
				ass.stats.SkippedStarts++
				return
			} else {
				log.Printf("策略 %s 验证失败，启动失败: %v", strategy.Name, err)
				ass.stats.FailedStarts++
				return
			}
		}
	}
	
	// 启动策略
	if err := ass.startStrategyWithRetry(ctx, strategy); err != nil {
		log.Printf("策略 %s 启动失败: %v", strategy.Name, err)
		ass.stats.FailedStarts++
		return
	}
	
	// 更新最后自动启动时间
	if err := ass.updateLastAutoStartTime(strategy.ID); err != nil {
		log.Printf("更新策略 %s 最后自动启动时间失败: %v", strategy.Name, err)
	}
	
	log.Printf("策略 %s 启动成功", strategy.Name)
	ass.stats.SuccessfulStarts++
}

// validateStrategy 验证策略
func (ass *AutoStartService) validateStrategy(ctx context.Context, strategy *StrategyStartInfo) error {
	// 这里可以添加策略验证逻辑
	// 暂时简单检查策略状态
	if !strategy.Enabled {
		return fmt.Errorf("策略未启用")
	}
	
	if strategy.IsRunning {
		return fmt.Errorf("策略已在运行")
	}
	
	return nil
}

// startStrategyWithRetry 带重试的策略启动
func (ass *AutoStartService) startStrategyWithRetry(ctx context.Context, strategy *StrategyStartInfo) error {
	var lastErr error
	delay := ass.config.RetryDelay
	
	for attempt := 0; attempt <= ass.config.MaxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("重试启动策略 %s (第 %d 次)", strategy.Name, attempt)
			time.Sleep(delay)
			delay = time.Duration(float64(delay) * ass.config.RetryBackoff)
		}
		
		if err := ass.startStrategy(ctx, strategy.ID); err != nil {
			lastErr = err
			continue
		}
		
		return nil
	}
	
	return fmt.Errorf("重试 %d 次后仍然失败: %w", ass.config.MaxRetries, lastErr)
}

// startStrategy 启动策略
func (ass *AutoStartService) startStrategy(ctx context.Context, strategyID string) error {
	// 更新数据库状态
	query := `
		UPDATE strategies
		SET is_running = true, status = 'active', updated_at = $1
		WHERE id = $2 AND enabled = true AND auto_start = true
	`
	
	now := time.Now()
	result, err := ass.db.ExecContext(ctx, query, now, strategyID)
	if err != nil {
		return fmt.Errorf("更新策略状态失败: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("策略不存在或不满足启动条件")
	}
	
	return nil
}

// updateLastAutoStartTime 更新最后自动启动时间
func (ass *AutoStartService) updateLastAutoStartTime(strategyID string) error {
	query := `
		UPDATE strategies
		SET last_auto_start = $1
		WHERE id = $2
	`
	
	_, err := ass.db.ExecContext(ass.ctx, query, time.Now(), strategyID)
	return err
}

// GetStats 获取统计信息
func (ass *AutoStartService) GetStats() *AutoStartStats {
	ass.mu.RLock()
	defer ass.mu.RUnlock()
	
	// 返回副本
	stats := *ass.stats
	return &stats
}

// IsRunning 检查服务是否运行中
func (ass *AutoStartService) IsRunning() bool {
	ass.mu.RLock()
	defer ass.mu.RUnlock()
	return ass.isRunning
}

// GetDefaultAutoStartConfig 获取默认配置
func GetDefaultAutoStartConfig() *AutoStartConfig {
	return &AutoStartConfig{
		Enabled:               true,
		StartupDelay:          30 * time.Second,  // 系统启动30秒后开始
		MaxConcurrentStartups: 3,                 // 最多同时启动3个策略
		StartupTimeout:        2 * time.Minute,   // 单个策略启动超时2分钟
		MaxRetries:            2,                 // 最多重试2次
		RetryDelay:            10 * time.Second,  // 重试延迟10秒
		RetryBackoff:          2.0,               // 重试退避倍数
		RequireValidation:     true,              // 需要验证
		SkipFailedValidation:  true,              // 跳过验证失败的策略
		MaxAutoStartPerBatch:  3,                 // 每批最多3个
		BatchInterval:         5 * time.Second,   // 批次间隔5秒
	}
}
