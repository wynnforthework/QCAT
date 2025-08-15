package hotlist

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Detector detects hot symbols
type Detector struct {
	scorer    *Scorer
	whitelist map[string]bool
	approvals map[string]*Approval
	config    *DetectorConfig
	mu        sync.RWMutex
}

// DetectorConfig represents detector configuration
type DetectorConfig struct {
	MinScore        float64       // 最小热度分数
	TopN            int           // 返回前N个热门币种
	ApprovalTimeout time.Duration // 审批超时时间
}

// Approval represents a symbol approval status
type Approval struct {
	Symbol     string
	Score      float64
	Status     ApprovalStatus
	ApprovedBy string
	ApprovedAt time.Time
	ExpiresAt  time.Time
}

// ApprovalStatus represents approval status
type ApprovalStatus string

const (
	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusRejected ApprovalStatus = "rejected"
)

// NewDetector creates a new detector
func NewDetector(scorer *Scorer, config *DetectorConfig) *Detector {
	return &Detector{
		scorer:    scorer,
		whitelist: make(map[string]bool),
		approvals: make(map[string]*Approval),
		config:    config,
	}
}

// DetectHotSymbols detects hot symbols
func (d *Detector) DetectHotSymbols(ctx context.Context, symbols []string) ([]*Score, error) {
	var scores []*Score

	// 计算所有币种的分数
	for _, symbol := range symbols {
		score, err := d.scorer.CalculateScore(ctx, symbol)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate score for %s: %w", symbol, err)
		}

		// 只保留达到最小分数的币种
		if score.TotalScore >= d.config.MinScore {
			scores = append(scores, score)
		}
	}

	// 按分数排序
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].TotalScore > scores[j].TotalScore
	})

	// 返回前N个
	if len(scores) > d.config.TopN {
		scores = scores[:d.config.TopN]
	}

	return scores, nil
}

// AddToWhitelist adds a symbol to whitelist
func (d *Detector) AddToWhitelist(symbol string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.whitelist[symbol] = true
}

// RemoveFromWhitelist removes a symbol from whitelist
func (d *Detector) RemoveFromWhitelist(symbol string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.whitelist, symbol)
}

// IsWhitelisted checks if a symbol is whitelisted
func (d *Detector) IsWhitelisted(symbol string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.whitelist[symbol]
}

// RequestApproval requests approval for a symbol
func (d *Detector) RequestApproval(symbol string, score float64) (*Approval, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 检查是否已经有审批
	if approval, exists := d.approvals[symbol]; exists {
		if approval.Status == ApprovalStatusPending && time.Now().Before(approval.ExpiresAt) {
			return approval, nil
		}
	}

	// 创建新的审批请求
	approval := &Approval{
		Symbol:    symbol,
		Score:     score,
		Status:    ApprovalStatusPending,
		ExpiresAt: time.Now().Add(d.config.ApprovalTimeout),
	}

	d.approvals[symbol] = approval
	return approval, nil
}

// ApproveSymbol approves a symbol
func (d *Detector) ApproveSymbol(symbol string, approver string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	approval, exists := d.approvals[symbol]
	if !exists {
		return fmt.Errorf("no approval request found for symbol: %s", symbol)
	}

	if approval.Status != ApprovalStatusPending {
		return fmt.Errorf("approval already processed: %s", symbol)
	}

	if time.Now().After(approval.ExpiresAt) {
		return fmt.Errorf("approval request expired: %s", symbol)
	}

	approval.Status = ApprovalStatusApproved
	approval.ApprovedBy = approver
	approval.ApprovedAt = time.Now()

	// 添加到白名单
	d.whitelist[symbol] = true

	return nil
}

// RejectSymbol rejects a symbol
func (d *Detector) RejectSymbol(symbol string, approver string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	approval, exists := d.approvals[symbol]
	if !exists {
		return fmt.Errorf("no approval request found for symbol: %s", symbol)
	}

	if approval.Status != ApprovalStatusPending {
		return fmt.Errorf("approval already processed: %s", symbol)
	}

	if time.Now().After(approval.ExpiresAt) {
		return fmt.Errorf("approval request expired: %s", symbol)
	}

	approval.Status = ApprovalStatusRejected
	approval.ApprovedBy = approver
	approval.ApprovedAt = time.Now()

	return nil
}

// GetApproval gets approval status for a symbol
func (d *Detector) GetApproval(symbol string) (*Approval, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	approval, exists := d.approvals[symbol]
	if !exists {
		return nil, fmt.Errorf("no approval found for symbol: %s", symbol)
	}

	return approval, nil
}

// CleanupExpiredApprovals cleans up expired approvals
func (d *Detector) CleanupExpiredApprovals() {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	for symbol, approval := range d.approvals {
		if approval.Status == ApprovalStatusPending && now.After(approval.ExpiresAt) {
			delete(d.approvals, symbol)
		}
	}
}
