package automl

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"qcat/internal/config"
)

// ConsistencyManager 训练一致性管理器
type ConsistencyManager struct {
	config *config.Config
	mu     sync.RWMutex

	// 全局随机种子管理
	globalSeed int64
	seeds      map[string]int64 // 任务ID -> 种子映射

	// 模型结果缓存和共享
	modelCache     map[string]*CachedModel
	resultRegistry map[string]*TrainingResult

	// 分布式协调
	nodeID         string
	clusterNodes   map[string]*ClusterNode
	consensusState *ConsensusState

	// 配置
	enableDeterministic bool
	enableModelSharing  bool
	enableConsensus     bool
	cacheTTL            time.Duration
}

// CachedModel 缓存的模型
type CachedModel struct {
	ModelID      string                 `json:"model_id"`
	TaskID       string                 `json:"task_id"`
	Parameters   map[string]interface{} `json:"parameters"`
	DataHash     string                 `json:"data_hash"`
	Result       *TrainingResult        `json:"result"`
	CreatedAt    time.Time              `json:"created_at"`
	LastAccessed time.Time              `json:"last_accessed"`
	AccessCount  int                    `json:"access_count"`
	IsValid      bool                   `json:"is_valid"`
}

// TrainingResult 训练结果
type TrainingResult struct {
	TaskID            string                 `json:"task_id"`
	ModelID           string                 `json:"model_id"`
	Parameters        map[string]interface{} `json:"parameters"`
	DataHash          string                 `json:"data_hash"`
	Performance       map[string]float64     `json:"performance"`
	TrainingMetrics   map[string]float64     `json:"training_metrics"`
	ValidationMetrics map[string]float64     `json:"validation_metrics"`
	TestMetrics       map[string]float64     `json:"test_metrics"`
	TrainingTime      time.Duration          `json:"training_time"`
	ModelSize         int64                  `json:"model_size"`
	CreatedAt         time.Time              `json:"created_at"`
	NodeID            string                 `json:"node_id"`
	ConsensusHash     string                 `json:"consensus_hash"`
}

// ClusterNode 集群节点信息
type ClusterNode struct {
	NodeID     string    `json:"node_id"`
	Address    string    `json:"address"`
	LastSeen   time.Time `json:"last_seen"`
	IsActive   bool      `json:"is_active"`
	ModelCount int       `json:"model_count"`
	LoadFactor float64   `json:"load_factor"`
}

// ConsensusState 共识状态
type ConsensusState struct {
	CurrentTerm   int64                   `json:"current_term"`
	LeaderID      string                  `json:"leader_id"`
	VotedFor      string                  `json:"voted_for"`
	LogIndex      int64                   `json:"log_index"`
	CommitIndex   int64                   `json:"commit_index"`
	AppliedIndex  int64                   `json:"applied_index"`
	LastHeartbeat time.Time               `json:"last_heartbeat"`
	ClusterConfig map[string]*ClusterNode `json:"cluster_config"`
}

// NewConsistencyManager 创建一致性管理器
func NewConsistencyManager(cfg *config.Config) (*ConsistencyManager, error) {
	cm := &ConsistencyManager{
		config:              cfg,
		globalSeed:          time.Now().UnixNano(),
		seeds:               make(map[string]int64),
		modelCache:          make(map[string]*CachedModel),
		resultRegistry:      make(map[string]*TrainingResult),
		nodeID:              generateNodeID(),
		clusterNodes:        make(map[string]*ClusterNode),
		consensusState:      &ConsensusState{},
		enableDeterministic: true,
		enableModelSharing:  true,
		enableConsensus:     true,
		cacheTTL:            24 * time.Hour,
	}

	// 从配置文件读取一致性设置
	if cfg != nil {
		// TODO: 从配置文件读取一致性参数
	}

	// 启动后台任务
	go cm.startBackgroundTasks()

	return cm, nil
}

// GetDeterministicSeed 获取确定性随机种子
func (cm *ConsistencyManager) GetDeterministicSeed(taskID string, dataHash string) int64 {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 生成基于任务ID和数据哈希的确定性种子
	seedKey := fmt.Sprintf("%s_%s", taskID, dataHash)

	if seed, exists := cm.seeds[seedKey]; exists {
		return seed
	}

	// 使用MD5哈希生成种子
	hash := md5.Sum([]byte(seedKey))
	seed := int64(hash[0])<<56 | int64(hash[1])<<48 | int64(hash[2])<<40 | int64(hash[3])<<32 |
		int64(hash[4])<<24 | int64(hash[5])<<16 | int64(hash[6])<<8 | int64(hash[7])

	cm.seeds[seedKey] = seed
	return seed
}

// SetRandomSeed 设置随机种子
func (cm *ConsistencyManager) SetRandomSeed(taskID string, dataHash string) {
	seed := cm.GetDeterministicSeed(taskID, dataHash)
	rand.Seed(seed)
	log.Printf("Set deterministic seed for task %s: %d", taskID, seed)
}

// CheckModelCache 检查模型缓存
func (cm *ConsistencyManager) CheckModelCache(taskID string, parameters map[string]interface{}, dataHash string) (*TrainingResult, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	cacheKey := cm.generateCacheKey(taskID, parameters, dataHash)

	if cached, exists := cm.modelCache[cacheKey]; exists && cached.IsValid {
		// 检查缓存是否过期
		if time.Since(cached.LastAccessed) < cm.cacheTTL {
			// 更新访问统计
			cm.mu.RUnlock()
			cm.mu.Lock()
			cached.LastAccessed = time.Now()
			cached.AccessCount++
			cm.mu.Unlock()
			cm.mu.RLock()

			log.Printf("Cache hit for task %s, returning cached result", taskID)
			return cached.Result, true
		} else {
			// 标记缓存过期
			cached.IsValid = false
		}
	}

	return nil, false
}

// CacheModelResult 缓存模型结果
func (cm *ConsistencyManager) CacheModelResult(taskID string, parameters map[string]interface{}, dataHash string, result *TrainingResult) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cacheKey := cm.generateCacheKey(taskID, parameters, dataHash)

	cached := &CachedModel{
		ModelID:      result.ModelID,
		TaskID:       taskID,
		Parameters:   parameters,
		DataHash:     dataHash,
		Result:       result,
		CreatedAt:    time.Now(),
		LastAccessed: time.Now(),
		AccessCount:  1,
		IsValid:      true,
	}

	cm.modelCache[cacheKey] = cached
	cm.resultRegistry[result.ModelID] = result

	log.Printf("Cached model result for task %s, cache key: %s", taskID, cacheKey)
}

// ShareModelResult 共享模型结果到集群
func (cm *ConsistencyManager) ShareModelResult(result *TrainingResult) error {
	if !cm.enableModelSharing {
		return nil
	}

	// 生成共识哈希
	result.ConsensusHash = cm.generateConsensusHash(result)

	// 广播到其他节点
	for nodeID, node := range cm.clusterNodes {
		if nodeID != cm.nodeID && node.IsActive {
			err := cm.broadcastModelResult(node, result)
			if err != nil {
				log.Printf("Failed to broadcast model result to node %s: %v", nodeID, err)
			}
		}
	}

	return nil
}

// GetSharedModelResult 从集群获取共享的模型结果
func (cm *ConsistencyManager) GetSharedModelResult(taskID string, parameters map[string]interface{}, dataHash string) (*TrainingResult, bool) {
	if !cm.enableModelSharing {
		return nil, false
	}

	// 从其他节点查询
	for nodeID, node := range cm.clusterNodes {
		if nodeID != cm.nodeID && node.IsActive {
			result, err := cm.queryModelResult(node, taskID, parameters, dataHash)
			if err == nil && result != nil {
				// 缓存结果
				cm.CacheModelResult(taskID, parameters, dataHash, result)
				log.Printf("Retrieved shared model result from node %s for task %s", nodeID, taskID)
				return result, true
			}
		}
	}

	return nil, false
}

// ValidateResultConsistency 验证结果一致性
func (cm *ConsistencyManager) ValidateResultConsistency(taskID string, localResult *TrainingResult) (*ConsistencyReport, error) {
	report := &ConsistencyReport{
		TaskID:           taskID,
		LocalResult:      localResult,
		ConsensusResults: make([]*TrainingResult, 0),
		IsConsistent:     true,
		Confidence:       1.0,
		CreatedAt:        time.Now(),
	}

	if !cm.enableConsensus {
		return report, nil
	}

	// 收集其他节点的结果
	for nodeID, node := range cm.clusterNodes {
		if nodeID != cm.nodeID && node.IsActive {
			result, err := cm.queryModelResult(node, taskID, localResult.Parameters, localResult.DataHash)
			if err == nil && result != nil {
				report.ConsensusResults = append(report.ConsensusResults, result)
			}
		}
	}

	// 计算一致性
	if len(report.ConsensusResults) > 0 {
		report.IsConsistent = cm.calculateConsistency(localResult, report.ConsensusResults)
		report.Confidence = cm.calculateConfidence(report.ConsensusResults)
	}

	return report, nil
}

// ConsistencyReport 一致性报告
type ConsistencyReport struct {
	TaskID           string             `json:"task_id"`
	LocalResult      *TrainingResult    `json:"local_result"`
	ConsensusResults []*TrainingResult  `json:"consensus_results"`
	IsConsistent     bool               `json:"is_consistent"`
	Confidence       float64            `json:"confidence"`
	Variance         map[string]float64 `json:"variance"`
	CreatedAt        time.Time          `json:"created_at"`
}

// Helper methods

func (cm *ConsistencyManager) generateCacheKey(taskID string, parameters map[string]interface{}, dataHash string) string {
	paramBytes, _ := json.Marshal(parameters)
	key := fmt.Sprintf("%s_%s_%s", taskID, string(paramBytes), dataHash)
	hash := md5.Sum([]byte(key))
	return hex.EncodeToString(hash[:])
}

func (cm *ConsistencyManager) generateConsensusHash(result *TrainingResult) string {
	data := fmt.Sprintf("%s_%s_%v_%s", result.TaskID, result.ModelID, result.Parameters, result.DataHash)
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (cm *ConsistencyManager) calculateConsistency(local *TrainingResult, others []*TrainingResult) bool {
	if len(others) == 0 {
		return true
	}

	// 检查性能指标的一致性
	tolerance := 0.01 // 1%的容差

	for _, other := range others {
		for metric, localValue := range local.Performance {
			if otherValue, exists := other.Performance[metric]; exists {
				diff := abs(localValue - otherValue)
				if diff > tolerance {
					log.Printf("Inconsistency detected for metric %s: local=%.4f, remote=%.4f, diff=%.4f",
						metric, localValue, otherValue, diff)
					return false
				}
			}
		}
	}

	return true
}

func (cm *ConsistencyManager) calculateConfidence(results []*TrainingResult) float64 {
	if len(results) == 0 {
		return 1.0
	}

	// 基于结果数量和一致性计算置信度
	consistentCount := 0
	for _, result := range results {
		if result.ConsensusHash != "" {
			consistentCount++
		}
	}

	return float64(consistentCount) / float64(len(results))
}

func (cm *ConsistencyManager) broadcastModelResult(node *ClusterNode, result *TrainingResult) error {
	// TODO: 实现实际的网络广播
	log.Printf("Broadcasting model result to node %s", node.NodeID)
	return nil
}

func (cm *ConsistencyManager) queryModelResult(node *ClusterNode, taskID string, parameters map[string]interface{}, dataHash string) (*TrainingResult, error) {
	// TODO: 实现实际的网络查询
	log.Printf("Querying model result from node %s for task %s", node.NodeID, taskID)
	return nil, nil
}

func (cm *ConsistencyManager) startBackgroundTasks() {
	// 定期清理过期缓存
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			cm.cleanupExpiredCache()
		}
	}()

	// 定期同步集群状态
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			cm.syncClusterState()
		}
	}()
}

func (cm *ConsistencyManager) cleanupExpiredCache() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	now := time.Now()
	expiredKeys := make([]string, 0)

	for key, cached := range cm.modelCache {
		if now.Sub(cached.LastAccessed) > cm.cacheTTL {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		delete(cm.modelCache, key)
	}

	if len(expiredKeys) > 0 {
		log.Printf("Cleaned up %d expired cache entries", len(expiredKeys))
	}
}

func (cm *ConsistencyManager) syncClusterState() {
	// TODO: 实现集群状态同步
}

func generateNodeID() string {
	// 生成唯一的节点ID
	timestamp := time.Now().UnixNano()
	random := rand.Int63()
	return fmt.Sprintf("node_%d_%d", timestamp, random)
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
