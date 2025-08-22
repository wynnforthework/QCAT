package kline

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// AutoBackfillService 自动回填服务
type AutoBackfillService struct {
	manager           *Manager
	watchedSymbols    map[string][]Interval // 监控的交易对和间隔
	checkInterval     time.Duration         // 检查间隔
	running           bool
	stopCh            chan struct{}
	mu                sync.RWMutex
	lastCheckTime     time.Time
	backfillHistory   []BackfillRecord
	maxHistoryRecords int
}

// BackfillRecord 回填记录
type BackfillRecord struct {
	Timestamp   time.Time `json:"timestamp"`
	Symbol      string    `json:"symbol"`
	Interval    Interval  `json:"interval"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Success     bool      `json:"success"`
	RecordCount int       `json:"record_count"`
	Error       string    `json:"error,omitempty"`
	Duration    time.Duration `json:"duration"`
}

// NewAutoBackfillService 创建自动回填服务
func NewAutoBackfillService(manager *Manager) *AutoBackfillService {
	return &AutoBackfillService{
		manager:           manager,
		watchedSymbols:    make(map[string][]Interval),
		checkInterval:     time.Hour, // 默认每小时检查一次
		stopCh:            make(chan struct{}),
		maxHistoryRecords: 1000,
	}
}

// AddWatchedSymbol 添加监控的交易对
func (s *AutoBackfillService) AddWatchedSymbol(symbol string, intervals ...Interval) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if len(intervals) == 0 {
		intervals = []Interval{Interval1h} // 默认1小时间隔
	}
	
	s.watchedSymbols[symbol] = intervals
	log.Printf("Added watched symbol: %s with intervals: %v", symbol, intervals)
}

// RemoveWatchedSymbol 移除监控的交易对
func (s *AutoBackfillService) RemoveWatchedSymbol(symbol string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	delete(s.watchedSymbols, symbol)
	log.Printf("Removed watched symbol: %s", symbol)
}

// SetCheckInterval 设置检查间隔
func (s *AutoBackfillService) SetCheckInterval(interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkInterval = interval
}

// Start 启动自动回填服务
func (s *AutoBackfillService) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("auto backfill service is already running")
	}
	s.running = true
	s.mu.Unlock()
	
	log.Println("Starting auto backfill service...")
	
	ticker := time.NewTicker(s.checkInterval)
	defer ticker.Stop()
	
	// 立即执行一次检查
	s.performCheck(ctx)
	
	for {
		select {
		case <-ctx.Done():
			log.Println("Auto backfill service stopped due to context cancellation")
			return ctx.Err()
		case <-s.stopCh:
			log.Println("Auto backfill service stopped")
			return nil
		case <-ticker.C:
			s.performCheck(ctx)
		}
	}
}

// Stop 停止自动回填服务
func (s *AutoBackfillService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.running {
		return
	}
	
	s.running = false
	close(s.stopCh)
	log.Println("Auto backfill service stop requested")
}

// performCheck 执行数据完整性检查和回填
func (s *AutoBackfillService) performCheck(ctx context.Context) {
	s.mu.Lock()
	watchedSymbols := make(map[string][]Interval)
	for symbol, intervals := range s.watchedSymbols {
		watchedSymbols[symbol] = intervals
	}
	s.lastCheckTime = time.Now()
	s.mu.Unlock()
	
	if len(watchedSymbols) == 0 {
		return
	}
	
	log.Printf("Performing auto backfill check for %d symbols...", len(watchedSymbols))
	
	config := s.manager.GetAutoBackfillConfig()
	if !config.Enabled {
		log.Println("Auto backfill is disabled, skipping check")
		return
	}
	
	now := time.Now()
	// 检查最近7天的数据
	checkStart := now.AddDate(0, 0, -7)
	
	for symbol, intervals := range watchedSymbols {
		for _, interval := range intervals {
			s.checkAndBackfillSymbol(ctx, symbol, interval, checkStart, now)
		}
	}
	
	log.Println("Auto backfill check completed")
}

// checkAndBackfillSymbol 检查并回填单个交易对的数据
func (s *AutoBackfillService) checkAndBackfillSymbol(ctx context.Context, symbol string, interval Interval, start, end time.Time) {
	startTime := time.Now()
	
	record := BackfillRecord{
		Timestamp: startTime,
		Symbol:    symbol,
		Interval:  interval,
		StartTime: start,
		EndTime:   end,
	}
	
	// 检查数据完整性
	report, err := s.manager.CheckDataIntegrity(ctx, symbol, interval, start, end)
	if err != nil {
		record.Success = false
		record.Error = fmt.Sprintf("integrity check failed: %v", err)
		s.addBackfillRecord(record)
		return
	}
	
	config := s.manager.GetAutoBackfillConfig()
	
	// 如果数据完整度足够，不需要回填
	if report.Completeness >= config.MinCompletenessPercent {
		record.Success = true
		record.RecordCount = report.ActualCount
		record.Duration = time.Since(startTime)
		s.addBackfillRecord(record)
		return
	}
	
	log.Printf("Auto backfill needed for %s %s (%.1f%% complete)", 
		symbol, interval, report.Completeness)
	
	// 执行回填
	err = s.manager.BackfillHistoricalData(ctx, symbol, interval, start, end)
	if err != nil {
		record.Success = false
		record.Error = fmt.Sprintf("backfill failed: %v", err)
		log.Printf("Auto backfill failed for %s %s: %v", symbol, interval, err)
	} else {
		// 重新检查数据完整性
		newReport, err := s.manager.CheckDataIntegrity(ctx, symbol, interval, start, end)
		if err != nil {
			record.Success = false
			record.Error = fmt.Sprintf("post-backfill check failed: %v", err)
		} else {
			record.Success = true
			record.RecordCount = newReport.ActualCount
			log.Printf("Auto backfill completed for %s %s: %.1f%% -> %.1f%% (%d records)", 
				symbol, interval, report.Completeness, newReport.Completeness, newReport.ActualCount)
		}
	}
	
	record.Duration = time.Since(startTime)
	s.addBackfillRecord(record)
}

// addBackfillRecord 添加回填记录
func (s *AutoBackfillService) addBackfillRecord(record BackfillRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.backfillHistory = append(s.backfillHistory, record)
	
	// 限制历史记录数量
	if len(s.backfillHistory) > s.maxHistoryRecords {
		s.backfillHistory = s.backfillHistory[len(s.backfillHistory)-s.maxHistoryRecords:]
	}
}

// GetBackfillHistory 获取回填历史记录
func (s *AutoBackfillService) GetBackfillHistory(limit int) []BackfillRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if limit <= 0 || limit > len(s.backfillHistory) {
		limit = len(s.backfillHistory)
	}
	
	// 返回最近的记录
	start := len(s.backfillHistory) - limit
	result := make([]BackfillRecord, limit)
	copy(result, s.backfillHistory[start:])
	
	return result
}

// GetStatus 获取服务状态
func (s *AutoBackfillService) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return map[string]interface{}{
		"running":           s.running,
		"watched_symbols":   len(s.watchedSymbols),
		"check_interval":    s.checkInterval.String(),
		"last_check_time":   s.lastCheckTime,
		"history_records":   len(s.backfillHistory),
		"symbols":           s.watchedSymbols,
	}
}
