package automl

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ResultSharingManager 结果共享管理器
type ResultSharingManager struct {
	config         *ResultSharingConfig
	resultsDB      map[string]*SharedResult
	mu             sync.RWMutex
	storagePath    string
	lastSyncTime   time.Time
}

// ResultSharingConfig 结果共享配置
type ResultSharingConfig struct {
	// 启用结果共享
	Enabled bool `json:"enabled" yaml:"enabled"`
	
	// 共享模式: file, string, seed, hybrid
	Mode string `json:"mode" yaml:"mode"`
	
	// 文件共享配置
	FileSharing struct {
		// 共享文件目录
		Directory string `json:"directory" yaml:"directory"`
		// 文件同步间隔
		SyncInterval time.Duration `json:"sync_interval" yaml:"sync_interval"`
		// 文件保留天数
		RetentionDays int `json:"retention_days" yaml:"retention_days"`
	} `json:"file_sharing" yaml:"file_sharing"`
	
	// 字符串共享配置
	StringSharing struct {
		// 共享字符串存储文件
		StorageFile string `json:"storage_file" yaml:"storage_file"`
		// 字符串格式: json, csv, custom
		Format string `json:"format" yaml:"format"`
		// 自定义分隔符
		Delimiter string `json:"delimiter" yaml:"delimiter"`
	} `json:"string_sharing" yaml:"string_sharing"`
	
	// 种子共享配置
	SeedSharing struct {
		// 种子映射文件
		MappingFile string `json:"mapping_file" yaml:"mapping_file"`
		// 种子范围
		SeedRange struct {
			Min int64 `json:"min" yaml:"min"`
			Max int64 `json:"max" yaml:"max"`
		} `json:"seed_range" yaml:"seed_range"`
		// 种子生成策略: random, sequential, hash_based
		Strategy string `json:"strategy" yaml:"strategy"`
	} `json:"seed_sharing" yaml:"seed_sharing"`
	
	// 性能阈值
	PerformanceThreshold struct {
		// 最小收益率阈值
		MinProfitRate float64 `json:"min_profit_rate" yaml:"min_profit_rate"`
		// 最小夏普比率阈值
		MinSharpeRatio float64 `json:"min_sharpe_ratio" yaml:"min_sharpe_ratio"`
		// 最大回撤阈值
		MaxDrawdown float64 `json:"max_drawdown" yaml:"max_drawdown"`
	} `json:"performance_threshold" yaml:"performance_threshold"`
}

// SharedResult 共享结果
type SharedResult struct {
	ID              string                 `json:"id"`
	TaskID          string                 `json:"task_id"`
	StrategyName    string                 `json:"strategy_name"`
	Parameters      map[string]interface{} `json:"parameters"`
	Performance     *PerformanceMetrics    `json:"performance"`
	RandomSeed      int64                  `json:"random_seed"`
	DataHash        string                 `json:"data_hash"`
	ModelData       []byte                 `json:"model_data,omitempty"`
	DiscoveredBy    string                 `json:"discovered_by"`
	DiscoveredAt    time.Time              `json:"discovered_at"`
	SharedAt        time.Time              `json:"shared_at"`
	ShareMethod     string                 `json:"share_method"`
	ShareSignature  string                 `json:"share_signature"`
	AdoptionCount   int                    `json:"adoption_count"`
	IsGlobalBest    bool                   `json:"is_global_best"`
}

// NewResultSharingManager 创建结果共享管理器
func NewResultSharingManager(config *ResultSharingConfig) (*ResultSharingManager, error) {
	rsm := &ResultSharingManager{
		config:      config,
		resultsDB:   make(map[string]*SharedResult),
		storagePath: "./data/shared_results",
	}

	// 创建存储目录
	if err := os.MkdirAll(rsm.storagePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// 初始化共享模式
	if err := rsm.initializeSharingMode(); err != nil {
		return nil, fmt.Errorf("failed to initialize sharing mode: %w", err)
	}

	// 加载现有结果
	if err := rsm.loadExistingResults(); err != nil {
		log.Printf("Warning: failed to load existing results: %v", err)
	}

	return rsm, nil
}

// initializeSharingMode 初始化共享模式
func (rsm *ResultSharingManager) initializeSharingMode() error {
	switch rsm.config.Mode {
	case "file":
		return rsm.initializeFileSharing()
	case "string":
		return rsm.initializeStringSharing()
	case "seed":
		return rsm.initializeSeedSharing()
	case "hybrid":
		return rsm.initializeHybridSharing()
	default:
		return fmt.Errorf("unsupported sharing mode: %s", rsm.config.Mode)
	}
}

// initializeFileSharing 初始化文件共享
func (rsm *ResultSharingManager) initializeFileSharing() error {
	if rsm.config.FileSharing.Directory == "" {
		rsm.config.FileSharing.Directory = filepath.Join(rsm.storagePath, "shared_files")
	}
	
	if err := os.MkdirAll(rsm.config.FileSharing.Directory, 0755); err != nil {
		return fmt.Errorf("failed to create file sharing directory: %w", err)
	}
	
	return nil
}

// initializeStringSharing 初始化字符串共享
func (rsm *ResultSharingManager) initializeStringSharing() error {
	if rsm.config.StringSharing.StorageFile == "" {
		rsm.config.StringSharing.StorageFile = filepath.Join(rsm.storagePath, "shared_strings.txt")
	}
	
	if rsm.config.StringSharing.Format == "" {
		rsm.config.StringSharing.Format = "json"
	}
	
	if rsm.config.StringSharing.Delimiter == "" {
		rsm.config.StringSharing.Delimiter = "|"
	}
	
	return nil
}

// initializeSeedSharing 初始化种子共享
func (rsm *ResultSharingManager) initializeSeedSharing() error {
	if rsm.config.SeedSharing.MappingFile == "" {
		rsm.config.SeedSharing.MappingFile = filepath.Join(rsm.storagePath, "seed_mapping.json")
	}
	
	if rsm.config.SeedSharing.SeedRange.Min == 0 {
		rsm.config.SeedSharing.SeedRange.Min = 1
	}
	
	if rsm.config.SeedSharing.SeedRange.Max == 0 {
		rsm.config.SeedSharing.SeedRange.Max = 1000000
	}
	
	if rsm.config.SeedSharing.Strategy == "" {
		rsm.config.SeedSharing.Strategy = "hash_based"
	}
	
	return nil
}

// initializeHybridSharing 初始化混合共享
func (rsm *ResultSharingManager) initializeHybridSharing() error {
	// 初始化所有共享模式
	if err := rsm.initializeFileSharing(); err != nil {
		return err
	}
	if err := rsm.initializeStringSharing(); err != nil {
		return err
	}
	if err := rsm.initializeSeedSharing(); err != nil {
		return err
	}
	return nil
}

// ShareResult 共享结果
func (rsm *ResultSharingManager) ShareResult(result *SharedResult) error {
	if !rsm.config.Enabled {
		return nil
	}

	// 检查性能阈值
	if !rsm.meetsPerformanceThreshold(result) {
		log.Printf("Result does not meet performance threshold, skipping sharing")
		return nil
	}

	// 生成共享签名
	result.ShareSignature = rsm.generateShareSignature(result)
	result.SharedAt = time.Now()

	// 根据模式共享结果
	switch rsm.config.Mode {
	case "file":
		return rsm.shareResultViaFile(result)
	case "string":
		return rsm.shareResultViaString(result)
	case "seed":
		return rsm.shareResultViaSeed(result)
	case "hybrid":
		return rsm.shareResultViaHybrid(result)
	default:
		return fmt.Errorf("unsupported sharing mode: %s", rsm.config.Mode)
	}
}

// meetsPerformanceThreshold 检查是否满足性能阈值
func (rsm *ResultSharingManager) meetsPerformanceThreshold(result *SharedResult) bool {
	if result.Performance == nil {
		return false
	}

	threshold := rsm.config.PerformanceThreshold

	if result.Performance.ProfitRate < threshold.MinProfitRate {
		return false
	}

	if result.Performance.SharpeRatio < threshold.MinSharpeRatio {
		return false
	}

	if result.Performance.MaxDrawdown > threshold.MaxDrawdown {
		return false
	}

	return true
}

// generateShareSignature 生成共享签名
func (rsm *ResultSharingManager) generateShareSignature(result *SharedResult) string {
	data := fmt.Sprintf("%s:%s:%s:%d:%s",
		result.TaskID,
		result.StrategyName,
		result.DataHash,
		result.RandomSeed,
		result.DiscoveredAt.Format(time.RFC3339),
	)

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// shareResultViaFile 通过文件共享结果
func (rsm *ResultSharingManager) shareResultViaFile(result *SharedResult) error {
	// 生成文件名
	filename := fmt.Sprintf("%s_%s_%s.json",
		result.TaskID,
		result.DiscoveredAt.Format("20060102_150405"),
		result.ShareSignature[:8],
	)
	
	filepath := filepath.Join(rsm.config.FileSharing.Directory, filename)
	
	// 序列化结果
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}
	
	// 写入文件
	if err := ioutil.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write result file: %w", err)
	}
	
	log.Printf("Shared result via file: %s", filepath)
	return nil
}

// shareResultViaString 通过字符串共享结果
func (rsm *ResultSharingManager) shareResultViaString(result *SharedResult) error {
	var shareString string
	
	switch rsm.config.StringSharing.Format {
	case "json":
		data, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("failed to marshal result to JSON: %w", err)
		}
		shareString = string(data)
		
	case "csv":
		shareString = rsm.resultToCSV(result)
		
	case "custom":
		shareString = rsm.resultToCustomFormat(result)
		
	default:
		return fmt.Errorf("unsupported string format: %s", rsm.config.StringSharing.Format)
	}
	
	// 追加到存储文件
	file, err := os.OpenFile(rsm.config.StringSharing.StorageFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open storage file: %w", err)
	}
	defer file.Close()
	
	if _, err := file.WriteString(shareString + "\n"); err != nil {
		return fmt.Errorf("failed to write to storage file: %w", err)
	}
	
	log.Printf("Shared result via string: %s", shareString[:100]+"...")
	return nil
}

// shareResultViaSeed 通过种子共享结果
func (rsm *ResultSharingManager) shareResultViaSeed(result *SharedResult) error {
	// 生成种子映射
	seedMapping := rsm.generateSeedMapping(result)
	
	// 读取现有映射
	existingMapping := make(map[string]interface{})
	if data, err := ioutil.ReadFile(rsm.config.SeedSharing.MappingFile); err == nil {
		json.Unmarshal(data, &existingMapping)
	}
	
	// 添加新映射
	existingMapping[seedMapping.Key] = seedMapping.Value
	
	// 写回文件
	data, err := json.MarshalIndent(existingMapping, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal seed mapping: %w", err)
	}
	
	if err := ioutil.WriteFile(rsm.config.SeedSharing.MappingFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write seed mapping file: %w", err)
	}
	
	log.Printf("Shared result via seed mapping: %s -> %s", seedMapping.Key, seedMapping.Value)
	return nil
}

// shareResultViaHybrid 通过混合方式共享结果
func (rsm *ResultSharingManager) shareResultViaHybrid(result *SharedResult) error {
	// 尝试所有共享方式
	var errors []string
	
	if err := rsm.shareResultViaFile(result); err != nil {
		errors = append(errors, fmt.Sprintf("file: %v", err))
	}
	
	if err := rsm.shareResultViaString(result); err != nil {
		errors = append(errors, fmt.Sprintf("string: %v", err))
	}
	
	if err := rsm.shareResultViaSeed(result); err != nil {
		errors = append(errors, fmt.Sprintf("seed: %v", err))
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("hybrid sharing errors: %s", strings.Join(errors, "; "))
	}
	
	return nil
}

// SeedMapping 种子映射
type SeedMapping struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// generateSeedMapping 生成种子映射
func (rsm *ResultSharingManager) generateSeedMapping(result *SharedResult) *SeedMapping {
	var seed int64
	
	switch rsm.config.SeedSharing.Strategy {
	case "random":
		seed = rand.Int63n(rsm.config.SeedSharing.SeedRange.Max-rsm.config.SeedSharing.SeedRange.Min) + rsm.config.SeedSharing.SeedRange.Min
		
	case "sequential":
		// 使用时间戳作为种子
		seed = time.Now().UnixNano() % (rsm.config.SeedSharing.SeedRange.Max - rsm.config.SeedSharing.SeedRange.Min) + rsm.config.SeedSharing.SeedRange.Min
		
	case "hash_based":
		// 基于任务ID和策略名称生成种子
		hash := sha256.Sum256([]byte(result.TaskID + result.StrategyName))
		seed = int64(hash[0])<<56 | int64(hash[1])<<48 | int64(hash[2])<<40 | int64(hash[3])<<32 |
			   int64(hash[4])<<24 | int64(hash[5])<<16 | int64(hash[6])<<8 | int64(hash[7])
		seed = seed % (rsm.config.SeedSharing.SeedRange.Max - rsm.config.SeedSharing.SeedRange.Min) + rsm.config.SeedSharing.SeedRange.Min
		
	default:
		seed = result.RandomSeed
	}
	
	key := fmt.Sprintf("%s_%s", result.TaskID, result.StrategyName)
	value := fmt.Sprintf("%d:%s:%.2f:%.2f",
		seed,
		result.ShareSignature[:8],
		result.Performance.ProfitRate,
		result.Performance.SharpeRatio,
	)
	
	return &SeedMapping{Key: key, Value: value}
}

// resultToCSV 将结果转换为CSV格式
func (rsm *ResultSharingManager) resultToCSV(result *SharedResult) string {
	return fmt.Sprintf("%s,%s,%s,%.2f,%.2f,%.2f,%.2f,%d,%s",
		result.TaskID,
		result.StrategyName,
		result.DiscoveredAt.Format("2006-01-02 15:04:05"),
		result.Performance.ProfitRate,
		result.Performance.SharpeRatio,
		result.Performance.MaxDrawdown,
		result.Performance.WinRate,
		result.RandomSeed,
		result.ShareSignature[:8],
	)
}

// resultToCustomFormat 将结果转换为自定义格式
func (rsm *ResultSharingManager) resultToCustomFormat(result *SharedResult) string {
	delimiter := rsm.config.StringSharing.Delimiter
	return fmt.Sprintf("%s%s%s%s%.2f%s%.2f%s%.2f%s%.2f%s%d%s%s",
		result.TaskID, delimiter,
		result.StrategyName, delimiter,
		result.Performance.ProfitRate, delimiter,
		result.Performance.SharpeRatio, delimiter,
		result.Performance.MaxDrawdown, delimiter,
		result.Performance.WinRate, delimiter,
		result.RandomSeed, delimiter,
		result.ShareSignature[:8],
	)
}

// LoadSharedResults 加载共享结果
func (rsm *ResultSharingManager) LoadSharedResults() error {
	if !rsm.config.Enabled {
		return nil
	}

	switch rsm.config.Mode {
	case "file":
		return rsm.loadSharedResultsFromFiles()
	case "string":
		return rsm.loadSharedResultsFromStrings()
	case "seed":
		return rsm.loadSharedResultsFromSeeds()
	case "hybrid":
		return rsm.loadSharedResultsFromHybrid()
	default:
		return fmt.Errorf("unsupported sharing mode: %s", rsm.config.Mode)
	}
}

// loadSharedResultsFromFiles 从文件加载共享结果
func (rsm *ResultSharingManager) loadSharedResultsFromFiles() error {
	files, err := filepath.Glob(filepath.Join(rsm.config.FileSharing.Directory, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to glob result files: %w", err)
	}

	for _, file := range files {
		data, err := ioutil.ReadFile(file)
		if err != nil {
			log.Printf("Failed to read file %s: %v", file, err)
			continue
		}

		var result SharedResult
		if err := json.Unmarshal(data, &result); err != nil {
			log.Printf("Failed to unmarshal result from %s: %v", file, err)
			continue
		}

		rsm.mu.Lock()
		rsm.resultsDB[result.ID] = &result
		rsm.mu.Unlock()
	}

	log.Printf("Loaded %d shared results from files", len(files))
	return nil
}

// loadSharedResultsFromStrings 从字符串加载共享结果
func (rsm *ResultSharingManager) loadSharedResultsFromStrings() error {
	data, err := ioutil.ReadFile(rsm.config.StringSharing.StorageFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在是正常的
		}
		return fmt.Errorf("failed to read storage file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var result SharedResult
		switch rsm.config.StringSharing.Format {
		case "json":
			if err := json.Unmarshal([]byte(line), &result); err != nil {
				log.Printf("Failed to unmarshal JSON line: %v", err)
				continue
			}
		case "csv":
			if err := rsm.parseCSVLine(line, &result); err != nil {
				log.Printf("Failed to parse CSV line: %v", err)
				continue
			}
		case "custom":
			if err := rsm.parseCustomLine(line, &result); err != nil {
				log.Printf("Failed to parse custom line: %v", err)
				continue
			}
		}

		rsm.mu.Lock()
		rsm.resultsDB[result.ID] = &result
		rsm.mu.Unlock()
	}

	log.Printf("Loaded %d shared results from strings", len(lines))
	return nil
}

// loadSharedResultsFromSeeds 从种子加载共享结果
func (rsm *ResultSharingManager) loadSharedResultsFromSeeds() error {
	data, err := ioutil.ReadFile(rsm.config.SeedSharing.MappingFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在是正常的
		}
		return fmt.Errorf("failed to read seed mapping file: %w", err)
	}

	var mapping map[string]interface{}
	if err := json.Unmarshal(data, &mapping); err != nil {
		return fmt.Errorf("failed to unmarshal seed mapping: %w", err)
	}

	for key, value := range mapping {
		if valueStr, ok := value.(string); ok {
			if result, err := rsm.parseSeedMapping(key, valueStr); err == nil {
				rsm.mu.Lock()
				rsm.resultsDB[result.ID] = result
				rsm.mu.Unlock()
			}
		}
	}

	log.Printf("Loaded %d shared results from seeds", len(mapping))
	return nil
}

// loadSharedResultsFromHybrid 从混合方式加载共享结果
func (rsm *ResultSharingManager) loadSharedResultsFromHybrid() error {
	// 尝试所有加载方式
	var errors []string

	if err := rsm.loadSharedResultsFromFiles(); err != nil {
		errors = append(errors, fmt.Sprintf("files: %v", err))
	}

	if err := rsm.loadSharedResultsFromStrings(); err != nil {
		errors = append(errors, fmt.Sprintf("strings: %v", err))
	}

	if err := rsm.loadSharedResultsFromSeeds(); err != nil {
		errors = append(errors, fmt.Sprintf("seeds: %v", err))
	}

	if len(errors) > 0 {
		log.Printf("Hybrid loading warnings: %s", strings.Join(errors, "; "))
	}

	return nil
}

// parseCSVLine 解析CSV行
func (rsm *ResultSharingManager) parseCSVLine(line string, result *SharedResult) error {
	parts := strings.Split(line, ",")
	if len(parts) < 9 {
		return fmt.Errorf("invalid CSV format")
	}

	result.TaskID = parts[0]
	result.StrategyName = parts[1]
	
	if t, err := time.Parse("2006-01-02 15:04:05", parts[2]); err == nil {
		result.DiscoveredAt = t
	}

	if val, err := strconv.ParseFloat(parts[3], 64); err == nil {
		if result.Performance == nil {
			result.Performance = &PerformanceMetrics{}
		}
		result.Performance.ProfitRate = val
	}

	if val, err := strconv.ParseFloat(parts[4], 64); err == nil {
		if result.Performance == nil {
			result.Performance = &PerformanceMetrics{}
		}
		result.Performance.SharpeRatio = val
	}

	if val, err := strconv.ParseFloat(parts[5], 64); err == nil {
		if result.Performance == nil {
			result.Performance = &PerformanceMetrics{}
		}
		result.Performance.MaxDrawdown = val
	}

	if val, err := strconv.ParseFloat(parts[6], 64); err == nil {
		if result.Performance == nil {
			result.Performance = &PerformanceMetrics{}
		}
		result.Performance.WinRate = val
	}

	if val, err := strconv.ParseInt(parts[7], 10, 64); err == nil {
		result.RandomSeed = val
	}

	result.ShareSignature = parts[8]
	result.ID = fmt.Sprintf("%s_%s_%s", result.TaskID, result.StrategyName, result.ShareSignature[:8])

	return nil
}

// parseCustomLine 解析自定义格式行
func (rsm *ResultSharingManager) parseCustomLine(line string, result *SharedResult) error {
	delimiter := rsm.config.StringSharing.Delimiter
	parts := strings.Split(line, delimiter)
	if len(parts) < 8 {
		return fmt.Errorf("invalid custom format")
	}

	result.TaskID = parts[0]
	result.StrategyName = parts[1]

	if val, err := strconv.ParseFloat(parts[2], 64); err == nil {
		if result.Performance == nil {
			result.Performance = &PerformanceMetrics{}
		}
		result.Performance.ProfitRate = val
	}

	if val, err := strconv.ParseFloat(parts[3], 64); err == nil {
		if result.Performance == nil {
			result.Performance = &PerformanceMetrics{}
		}
		result.Performance.SharpeRatio = val
	}

	if val, err := strconv.ParseFloat(parts[4], 64); err == nil {
		if result.Performance == nil {
			result.Performance = &PerformanceMetrics{}
		}
		result.Performance.MaxDrawdown = val
	}

	if val, err := strconv.ParseFloat(parts[5], 64); err == nil {
		if result.Performance == nil {
			result.Performance = &PerformanceMetrics{}
		}
		result.Performance.WinRate = val
	}

	if val, err := strconv.ParseInt(parts[6], 10, 64); err == nil {
		result.RandomSeed = val
	}

	result.ShareSignature = parts[7]
	result.ID = fmt.Sprintf("%s_%s_%s", result.TaskID, result.StrategyName, result.ShareSignature[:8])

	return nil
}

// parseSeedMapping 解析种子映射
func (rsm *ResultSharingManager) parseSeedMapping(key, value string) (*SharedResult, error) {
	parts := strings.Split(value, ":")
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid seed mapping format")
	}

	keyParts := strings.Split(key, "_")
	if len(keyParts) < 2 {
		return nil, fmt.Errorf("invalid key format")
	}

	result := &SharedResult{
		TaskID:       keyParts[0],
		StrategyName: keyParts[1],
		Performance:  &PerformanceMetrics{},
	}

	if val, err := strconv.ParseInt(parts[0], 10, 64); err == nil {
		result.RandomSeed = val
	}

	result.ShareSignature = parts[1]

	if val, err := strconv.ParseFloat(parts[2], 64); err == nil {
		result.Performance.ProfitRate = val
	}

	if val, err := strconv.ParseFloat(parts[3], 64); err == nil {
		result.Performance.SharpeRatio = val
	}

	result.ID = fmt.Sprintf("%s_%s_%s", result.TaskID, result.StrategyName, result.ShareSignature[:8])

	return result, nil
}

// GetBestSharedResult 获取最佳共享结果
func (rsm *ResultSharingManager) GetBestSharedResult(taskID, strategyName string) *SharedResult {
	rsm.mu.RLock()
	defer rsm.mu.RUnlock()

	var bestResult *SharedResult
	var bestScore float64

	for _, result := range rsm.resultsDB {
		if result.TaskID == taskID && result.StrategyName == strategyName {
			score := rsm.calculateScore(result)
			if bestResult == nil || score > bestScore {
				bestResult = result
				bestScore = score
			}
		}
	}

	return bestResult
}

// calculateScore 计算结果评分
func (rsm *ResultSharingManager) calculateScore(result *SharedResult) float64 {
	if result.Performance == nil {
		return 0
	}

	// 简单的评分算法，可以根据需要调整
	score := result.Performance.ProfitRate * 0.4 +
		result.Performance.SharpeRatio * 0.3 +
		(1-result.Performance.MaxDrawdown) * 0.2 +
		result.Performance.WinRate * 0.1

	return score
}

// GetAllSharedResults 获取所有共享结果
func (rsm *ResultSharingManager) GetAllSharedResults() []*SharedResult {
	rsm.mu.RLock()
	defer rsm.mu.RUnlock()

	var results []*SharedResult
	for _, result := range rsm.resultsDB {
		results = append(results, result)
	}

	// 按发现时间排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].DiscoveredAt.After(results[j].DiscoveredAt)
	})

	return results
}

// loadExistingResults 加载现有结果
func (rsm *ResultSharingManager) loadExistingResults() error {
	return rsm.LoadSharedResults()
}

// CleanupOldResults 清理旧结果
func (rsm *ResultSharingManager) CleanupOldResults() error {
	if !rsm.config.Enabled {
		return nil
	}

	cutoffTime := time.Now().AddDate(0, 0, -rsm.config.FileSharing.RetentionDays)

	// 清理文件
	if rsm.config.Mode == "file" || rsm.config.Mode == "hybrid" {
		files, err := filepath.Glob(filepath.Join(rsm.config.FileSharing.Directory, "*.json"))
		if err != nil {
			return fmt.Errorf("failed to glob result files: %w", err)
		}

		for _, file := range files {
			info, err := os.Stat(file)
			if err != nil {
				continue
			}

			if info.ModTime().Before(cutoffTime) {
				if err := os.Remove(file); err != nil {
					log.Printf("Failed to remove old file %s: %v", file, err)
				} else {
					log.Printf("Removed old result file: %s", file)
				}
			}
		}
	}

	// 清理内存中的旧结果
	rsm.mu.Lock()
	for id, result := range rsm.resultsDB {
		if result.DiscoveredAt.Before(cutoffTime) {
			delete(rsm.resultsDB, id)
		}
	}
	rsm.mu.Unlock()

	return nil
}
