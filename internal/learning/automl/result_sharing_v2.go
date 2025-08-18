package automl

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// SharedResultV2 增强版共享结果
type SharedResultV2 struct {
	// 基本信息
	ID           string    `json:"id"`
	TaskID       string    `json:"task_id"`
	StrategyName string    `json:"strategy_name"`
	Version      string    `json:"version"`
	CreatedAt    time.Time `json:"created_at"`
	SharedBy     string    `json:"shared_by"`

	// 策略参数
	Parameters map[string]interface{} `json:"parameters"`

	// 性能指标
	Performance *PerformanceMetricsV2 `json:"performance"`

	// 可复现性数据
	Reproducibility *ReproducibilityData `json:"reproducibility"`

	// 策略支持信息
	StrategySupport *StrategySupportInfo `json:"strategy_support"`

	// 回测信息
	BacktestInfo *BacktestInfo `json:"backtest_info"`

	// 实盘信息
	LiveTradingInfo *LiveTradingInfo `json:"live_trading_info,omitempty"`

	// 风险评估
	RiskAssessment *RiskAssessment `json:"risk_assessment"`

	// 市场适应性
	MarketAdaptation *MarketAdaptation `json:"market_adaptation"`

	// 分享信息
	ShareInfo *ShareInfo `json:"share_info"`
}

// PerformanceMetricsV2 增强版性能指标
type PerformanceMetricsV2 struct {
	// 基础收益指标
	TotalReturn   float64 `json:"total_return"`   // 总收益率
	AnnualReturn  float64 `json:"annual_return"`  // 年化收益率
	MonthlyReturn float64 `json:"monthly_return"` // 月化收益率
	DailyReturn   float64 `json:"daily_return"`   // 日化收益率

	// 风险指标
	MaxDrawdown  float64 `json:"max_drawdown"`  // 最大回撤
	Volatility   float64 `json:"volatility"`    // 波动率
	SharpeRatio  float64 `json:"sharpe_ratio"`  // 夏普比率
	SortinoRatio float64 `json:"sortino_ratio"` // 索提诺比率
	CalmarRatio  float64 `json:"calmar_ratio"`  // 卡玛比率

	// 交易统计
	TotalTrades  int     `json:"total_trades"`  // 总交易次数
	WinRate      float64 `json:"win_rate"`      // 胜率
	ProfitFactor float64 `json:"profit_factor"` // 盈亏比
	AverageWin   float64 `json:"average_win"`   // 平均盈利
	AverageLoss  float64 `json:"average_loss"`  // 平均亏损
	LargestWin   float64 `json:"largest_win"`   // 最大单笔盈利
	LargestLoss  float64 `json:"largest_loss"`  // 最大单笔亏损

	// 时间分析
	BestMonth         string `json:"best_month"`         // 最佳月份
	WorstMonth        string `json:"worst_month"`        // 最差月份
	ConsecutiveWins   int    `json:"consecutive_wins"`   // 最大连续盈利次数
	ConsecutiveLosses int    `json:"consecutive_losses"` // 最大连续亏损次数
}

// ReproducibilityData 可复现性数据
type ReproducibilityData struct {
	RandomSeed         int64    `json:"random_seed"`         // 随机种子
	DataHash           string   `json:"data_hash"`           // 数据哈希
	CodeVersion        string   `json:"code_version"`        // 代码版本
	Environment        string   `json:"environment"`         // 运行环境
	DataRange          string   `json:"data_range"`          // 数据时间范围
	DataSources        []string `json:"data_sources"`        // 数据源列表
	Preprocessing      string   `json:"preprocessing"`       // 数据预处理方法
	FeatureEngineering string   `json:"feature_engineering"` // 特征工程方法
}

// StrategySupportInfo 策略支持信息
type StrategySupportInfo struct {
	SupportedMarkets    []string `json:"supported_markets"`    // 支持的交易品种
	SupportedTimeframes []string `json:"supported_timeframes"` // 支持的时间框架
	MinCapital          float64  `json:"min_capital"`          // 最小资金要求
	MaxCapital          float64  `json:"max_capital"`          // 最大资金要求
	LeverageSupport     bool     `json:"leverage_support"`     // 是否支持杠杆
	MaxLeverage         int      `json:"max_leverage"`         // 最大杠杆倍数
	ShortSupport        bool     `json:"short_support"`        // 是否支持做空
	HedgeSupport        bool     `json:"hedge_support"`        // 是否支持对冲
}

// BacktestInfo 回测信息
type BacktestInfo struct {
	StartDate        time.Time `json:"start_date"`        // 回测开始时间
	EndDate          time.Time `json:"end_date"`          // 回测结束时间
	Duration         string    `json:"duration"`          // 回测时长
	DataPoints       int       `json:"data_points"`       // 数据点数量
	MarketConditions []string  `json:"market_conditions"` // 市场环境
	Commission       float64   `json:"commission"`        // 手续费率
	Slippage         float64   `json:"slippage"`          // 滑点
	InitialCapital   float64   `json:"initial_capital"`   // 初始资金
	FinalCapital     float64   `json:"final_capital"`     // 最终资金
}

// LiveTradingInfo 实盘信息
type LiveTradingInfo struct {
	StartDate    time.Time `json:"start_date"`    // 实盘开始时间
	EndDate      time.Time `json:"end_date"`      // 实盘结束时间
	Duration     string    `json:"duration"`      // 实盘时长
	TotalTrades  int       `json:"total_trades"`  // 实盘交易次数
	LiveReturn   float64   `json:"live_return"`   // 实盘收益率
	LiveDrawdown float64   `json:"live_drawdown"` // 实盘最大回撤
	LiveSharpe   float64   `json:"live_sharpe"`   // 实盘夏普比率
	LiveWinRate  float64   `json:"live_win_rate"` // 实盘胜率
	Platform     string    `json:"platform"`      // 交易平台
	AccountType  string    `json:"account_type"`  // 账户类型
}

// RiskAssessment 风险评估
type RiskAssessment struct {
	VaR95             float64 `json:"var_95"`             // 95%置信度VaR
	VaR99             float64 `json:"var_99"`             // 99%置信度VaR
	ExpectedShortfall float64 `json:"expected_shortfall"` // 期望损失
	Beta              float64 `json:"beta"`               // Beta系数
	Alpha             float64 `json:"alpha"`              // Alpha系数
	InformationRatio  float64 `json:"information_ratio"`  // 信息比率
	TreynorRatio      float64 `json:"treynor_ratio"`      // 特雷诺比率
	JensenAlpha       float64 `json:"jensen_alpha"`       // 詹森Alpha
	DownsideDeviation float64 `json:"downside_deviation"` // 下行偏差
	UpsideCapture     float64 `json:"upside_capture"`     // 上行捕获率
	DownsideCapture   float64 `json:"downside_capture"`   // 下行捕获率
}

// MarketAdaptation 市场适应性
type MarketAdaptation struct {
	BullMarketReturn     float64 `json:"bull_market_return"`     // 牛市收益率
	BearMarketReturn     float64 `json:"bear_market_return"`     // 熊市收益率
	SidewaysMarketReturn float64 `json:"sideways_market_return"` // 震荡市收益率
	HighVolatilityReturn float64 `json:"high_volatility_return"` // 高波动率收益率
	LowVolatilityReturn  float64 `json:"low_volatility_return"`  // 低波动率收益率
	TrendFollowingScore  float64 `json:"trend_following_score"`  // 趋势跟踪评分
	MeanReversionScore   float64 `json:"mean_reversion_score"`   // 均值回归评分
	MomentumScore        float64 `json:"momentum_score"`         // 动量评分
}

// ShareInfo 分享信息
type ShareInfo struct {
	ShareMethod      string    `json:"share_method"`      // 分享方式
	ShareDate        time.Time `json:"share_date"`        // 分享时间
	SharePlatform    string    `json:"share_platform"`    // 分享平台
	ShareDescription string    `json:"share_description"` // 分享描述
	Tags             []string  `json:"tags"`              // 标签
	Rating           float64   `json:"rating"`            // 评分
	ReviewCount      int       `json:"review_count"`      // 评论数量
	DownloadCount    int       `json:"download_count"`    // 下载次数
	UseCount         int       `json:"use_count"`         // 使用次数
}

// ResultSharingManagerV2 增强版结果共享管理器
type ResultSharingManagerV2 struct {
	config       *ResultSharingConfigV2
	resultsDB    map[string]*SharedResultV2
	mu           sync.RWMutex
	storagePath  string
	lastSyncTime time.Time
}

// ResultSharingConfigV2 增强版配置
type ResultSharingConfigV2 struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	// 文件存储配置
	FileStorage struct {
		Directory     string `json:"directory" yaml:"directory"`
		FileExtension string `json:"file_extension" yaml:"file_extension"`
		MaxFileSize   int64  `json:"max_file_size" yaml:"max_file_size"`
		RetentionDays int    `json:"retention_days" yaml:"retention_days"`
	} `json:"file_storage" yaml:"file_storage"`

	// 性能阈值
	PerformanceThreshold struct {
		MinTotalReturn  float64 `json:"min_total_return" yaml:"min_total_return"`
		MinSharpeRatio  float64 `json:"min_sharpe_ratio" yaml:"min_sharpe_ratio"`
		MaxDrawdown     float64 `json:"max_drawdown" yaml:"max_drawdown"`
		MinWinRate      float64 `json:"min_win_rate" yaml:"min_win_rate"`
		MinProfitFactor float64 `json:"min_profit_factor" yaml:"min_profit_factor"`
	} `json:"performance_threshold" yaml:"performance_threshold"`

	// 评分权重
	ScoringWeights struct {
		TotalReturn     float64 `json:"total_return" yaml:"total_return"`
		SharpeRatio     float64 `json:"sharpe_ratio" yaml:"sharpe_ratio"`
		MaxDrawdown     float64 `json:"max_drawdown" yaml:"max_drawdown"`
		WinRate         float64 `json:"win_rate" yaml:"win_rate"`
		ProfitFactor    float64 `json:"profit_factor" yaml:"profit_factor"`
		LivePerformance float64 `json:"live_performance" yaml:"live_performance"`
		RiskAssessment  float64 `json:"risk_assessment" yaml:"risk_assessment"`
	} `json:"scoring_weights" yaml:"scoring_weights"`
}

// NewResultSharingManagerV2 创建新的结果共享管理器
func NewResultSharingManagerV2(config *ResultSharingConfigV2) (*ResultSharingManagerV2, error) {
	if config == nil {
		config = &ResultSharingConfigV2{
			Enabled: true,
		}
	}

	// 设置默认值
	if config.FileStorage.Directory == "" {
		config.FileStorage.Directory = "./data/shared_results"
	}
	if config.FileStorage.FileExtension == "" {
		config.FileStorage.FileExtension = ".json"
	}
	if config.FileStorage.MaxFileSize == 0 {
		config.FileStorage.MaxFileSize = 10 * 1024 * 1024 // 10MB
	}
	if config.FileStorage.RetentionDays == 0 {
		config.FileStorage.RetentionDays = 365
	}

	// 设置默认阈值
	if config.PerformanceThreshold.MinTotalReturn == 0 {
		config.PerformanceThreshold.MinTotalReturn = 5.0
	}
	if config.PerformanceThreshold.MinSharpeRatio == 0 {
		config.PerformanceThreshold.MinSharpeRatio = 0.5
	}
	if config.PerformanceThreshold.MaxDrawdown == 0 {
		config.PerformanceThreshold.MaxDrawdown = 20.0
	}
	if config.PerformanceThreshold.MinWinRate == 0 {
		config.PerformanceThreshold.MinWinRate = 0.4
	}
	if config.PerformanceThreshold.MinProfitFactor == 0 {
		config.PerformanceThreshold.MinProfitFactor = 1.2
	}

	// 设置默认权重
	if config.ScoringWeights.TotalReturn == 0 {
		config.ScoringWeights.TotalReturn = 0.25
	}
	if config.ScoringWeights.SharpeRatio == 0 {
		config.ScoringWeights.SharpeRatio = 0.20
	}
	if config.ScoringWeights.MaxDrawdown == 0 {
		config.ScoringWeights.MaxDrawdown = 0.15
	}
	if config.ScoringWeights.WinRate == 0 {
		config.ScoringWeights.WinRate = 0.10
	}
	if config.ScoringWeights.ProfitFactor == 0 {
		config.ScoringWeights.ProfitFactor = 0.10
	}
	if config.ScoringWeights.LivePerformance == 0 {
		config.ScoringWeights.LivePerformance = 0.15
	}
	if config.ScoringWeights.RiskAssessment == 0 {
		config.ScoringWeights.RiskAssessment = 0.05
	}

	// 创建目录
	if err := os.MkdirAll(config.FileStorage.Directory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	mgr := &ResultSharingManagerV2{
		config:      config,
		resultsDB:   make(map[string]*SharedResultV2),
		storagePath: config.FileStorage.Directory,
	}

	// 加载现有结果
	if err := mgr.loadExistingResults(); err != nil {
		log.Printf("Warning: failed to load existing results: %v", err)
	}

	return mgr, nil
}

// ShareResult 共享结果
func (mgr *ResultSharingManagerV2) ShareResult(result *SharedResultV2) error {
	if !mgr.config.Enabled {
		return fmt.Errorf("result sharing is disabled")
	}

	// 验证结果
	if err := mgr.validateResult(result); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// 检查性能阈值
	if !mgr.checkPerformanceThreshold(result) {
		return fmt.Errorf("performance below threshold")
	}

	// 生成ID
	if result.ID == "" {
		result.ID = mgr.generateID(result)
	}

	// 设置时间戳
	if result.CreatedAt.IsZero() {
		result.CreatedAt = time.Now()
	}

	// 计算评分
	score := mgr.calculateScore(result)

	// 保存到内存
	mgr.mu.Lock()
	mgr.resultsDB[result.ID] = result
	mgr.mu.Unlock()

	// 保存到文件
	if err := mgr.saveToFile(result); err != nil {
		return fmt.Errorf("failed to save to file: %w", err)
	}

	log.Printf("Result shared successfully: %s (Score: %.2f)", result.ID, score)
	return nil
}

// ExportResult 导出结果为JSON文件
func (mgr *ResultSharingManagerV2) ExportResult(resultID string) ([]byte, error) {
	mgr.mu.RLock()
	result, exists := mgr.resultsDB[resultID]
	mgr.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("result not found: %s", resultID)
	}

	// 添加导出信息
	exportData := map[string]interface{}{
		"export_info": map[string]interface{}{
			"export_time": time.Now(),
			"exported_by": "result_sharing_system",
			"version":     "2.0",
		},
		"result": result,
	}

	return json.MarshalIndent(exportData, "", "  ")
}

// ImportResult 导入结果
func (mgr *ResultSharingManagerV2) ImportResult(data []byte) error {
	var importData map[string]interface{}
	if err := json.Unmarshal(data, &importData); err != nil {
		return fmt.Errorf("invalid JSON format: %w", err)
	}

	// 提取结果数据
	resultData, ok := importData["result"]
	if !ok {
		return fmt.Errorf("no result data found")
	}

	resultBytes, err := json.Marshal(resultData)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	var result SharedResultV2
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		return fmt.Errorf("failed to unmarshal result: %w", err)
	}

	// 共享结果
	return mgr.ShareResult(&result)
}

// GetBestResults 获取最优结果
func (mgr *ResultSharingManagerV2) GetBestResults(limit int) []*SharedResultV2 {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	var results []*SharedResultV2
	for _, result := range mgr.resultsDB {
		results = append(results, result)
	}

	// 按评分排序
	sort.Slice(results, func(i, j int) bool {
		scoreI := mgr.calculateScore(results[i])
		scoreJ := mgr.calculateScore(results[j])
		return scoreI > scoreJ
	})

	if limit > 0 && limit < len(results) {
		return results[:limit]
	}
	return results
}

// SearchResults 搜索结果
func (mgr *ResultSharingManagerV2) SearchResults(query string, filters map[string]interface{}) []*SharedResultV2 {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	var results []*SharedResultV2
	for _, result := range mgr.resultsDB {
		if mgr.matchesQuery(result, query, filters) {
			results = append(results, result)
		}
	}

	return results
}

// 验证结果
func (mgr *ResultSharingManagerV2) validateResult(result *SharedResultV2) error {
	if result.TaskID == "" {
		return fmt.Errorf("task_id is required")
	}
	if result.StrategyName == "" {
		return fmt.Errorf("strategy_name is required")
	}
	if result.Performance == nil {
		return fmt.Errorf("performance data is required")
	}
	if result.Reproducibility == nil {
		return fmt.Errorf("reproducibility data is required")
	}
	return nil
}

// 检查性能阈值
func (mgr *ResultSharingManagerV2) checkPerformanceThreshold(result *SharedResultV2) bool {
	perf := result.Performance
	threshold := mgr.config.PerformanceThreshold

	return perf.TotalReturn >= threshold.MinTotalReturn &&
		perf.SharpeRatio >= threshold.MinSharpeRatio &&
		perf.MaxDrawdown <= threshold.MaxDrawdown &&
		perf.WinRate >= threshold.MinWinRate &&
		perf.ProfitFactor >= threshold.MinProfitFactor
}

// 计算评分
func (mgr *ResultSharingManagerV2) calculateScore(result *SharedResultV2) float64 {
	perf := result.Performance
	weights := mgr.config.ScoringWeights

	score := 0.0

	// 基础性能评分
	score += perf.TotalReturn * weights.TotalReturn
	score += perf.SharpeRatio * weights.SharpeRatio
	score += (1 - perf.MaxDrawdown/100) * weights.MaxDrawdown
	score += perf.WinRate * weights.WinRate
	score += perf.ProfitFactor * weights.ProfitFactor

	// 实盘表现评分
	if result.LiveTradingInfo != nil {
		liveScore := (result.LiveTradingInfo.LiveReturn*0.4 +
			result.LiveTradingInfo.LiveSharpe*0.3 +
			(1-result.LiveTradingInfo.LiveDrawdown/100)*0.2 +
			result.LiveTradingInfo.LiveWinRate*0.1)
		score += liveScore * weights.LivePerformance
	}

	// 风险评估评分
	if result.RiskAssessment != nil {
		riskScore := (result.RiskAssessment.InformationRatio*0.4 +
			result.RiskAssessment.TreynorRatio*0.3 +
			result.RiskAssessment.JensenAlpha*0.3)
		score += riskScore * weights.RiskAssessment
	}

	return score
}

// 生成ID
func (mgr *ResultSharingManagerV2) generateID(result *SharedResultV2) string {
	data := fmt.Sprintf("%s_%s_%d", result.TaskID, result.StrategyName, time.Now().Unix())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8])
}

// 保存到文件
func (mgr *ResultSharingManagerV2) saveToFile(result *SharedResultV2) error {
	filename := fmt.Sprintf("%s_%s%s", result.ID, result.StrategyName, mgr.config.FileStorage.FileExtension)
	filepath := filepath.Join(mgr.storagePath, filename)

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	return ioutil.WriteFile(filepath, data, 0644)
}

// 加载现有结果
func (mgr *ResultSharingManagerV2) loadExistingResults() error {
	files, err := filepath.Glob(filepath.Join(mgr.storagePath, "*"+mgr.config.FileStorage.FileExtension))
	if err != nil {
		return err
	}

	for _, file := range files {
		data, err := ioutil.ReadFile(file)
		if err != nil {
			log.Printf("Warning: failed to read file %s: %v", file, err)
			continue
		}

		var result SharedResultV2
		if err := json.Unmarshal(data, &result); err != nil {
			log.Printf("Warning: failed to unmarshal file %s: %v", file, err)
			continue
		}

		mgr.resultsDB[result.ID] = &result
	}

	return nil
}

// 匹配查询
func (mgr *ResultSharingManagerV2) matchesQuery(result *SharedResultV2, query string, filters map[string]interface{}) bool {
	// 文本搜索
	if query != "" {
		text := strings.ToLower(fmt.Sprintf("%s %s %s",
			result.TaskID, result.StrategyName, result.SharedBy))
		if !strings.Contains(text, strings.ToLower(query)) {
			return false
		}
	}

	// 过滤器
	for key, value := range filters {
		switch key {
		case "min_total_return":
			if result.Performance.TotalReturn < value.(float64) {
				return false
			}
		case "min_sharpe_ratio":
			if result.Performance.SharpeRatio < value.(float64) {
				return false
			}
		case "max_drawdown":
			if result.Performance.MaxDrawdown > value.(float64) {
				return false
			}
		case "strategy_name":
			if result.StrategyName != value.(string) {
				return false
			}
		case "shared_by":
			if result.SharedBy != value.(string) {
				return false
			}
		}
	}

	return true
}
