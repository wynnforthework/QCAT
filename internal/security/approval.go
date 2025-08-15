package security

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ApprovalType 审批类型
type ApprovalType string

const (
	ApprovalTypeStrategy    ApprovalType = "strategy"    // 策略审批
	ApprovalTypeRiskLimit   ApprovalType = "risk_limit"  // 风控限额审批
	ApprovalTypeHotlist     ApprovalType = "hotlist"     // 热门币种审批
	ApprovalTypeAPIKey      ApprovalType = "api_key"     // API密钥审批
	ApprovalTypeSystem      ApprovalType = "system"      // 系统设置审批
)

// ApprovalStatus 审批状态
type ApprovalStatus string

const (
	ApprovalStatusPending   ApprovalStatus = "pending"   // 待审批
	ApprovalStatusApproved  ApprovalStatus = "approved"  // 已批准
	ApprovalStatusRejected  ApprovalStatus = "rejected"  // 已拒绝
	ApprovalStatusExpired   ApprovalStatus = "expired"   // 已过期
)

// ApprovalRequest 审批请求
type ApprovalRequest struct {
	ID          string                 `json:"id"`
	Type        ApprovalType           `json:"type"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	RequesterID string                 `json:"requester_id"`
	Data        map[string]interface{} `json:"data"`
	Status      ApprovalStatus         `json:"status"`
	Approvers   []string               `json:"approvers"`
	Approvals   []*Approval            `json:"approvals"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	ExpiresAt   time.Time              `json:"expires_at"`
	Priority    int                    `json:"priority"` // 优先级：1-低，2-中，3-高，4-紧急
}

// Approval 审批记录
type Approval struct {
	ID        string         `json:"id"`
	RequestID string         `json:"request_id"`
	ApproverID string        `json:"approver_id"`
	Status    ApprovalStatus `json:"status"`
	Comment   string         `json:"comment"`
	CreatedAt time.Time      `json:"created_at"`
}

// ApprovalWorkflow 审批工作流
type ApprovalWorkflow struct {
	mu      sync.RWMutex
	requests map[string]*ApprovalRequest
	rbac    *RBAC
}

// NewApprovalWorkflow 创建审批工作流
func NewApprovalWorkflow(rbac *RBAC) *ApprovalWorkflow {
	return &ApprovalWorkflow{
		requests: make(map[string]*ApprovalRequest),
		rbac:     rbac,
	}
}

// CreateApprovalRequest 创建审批请求
func (w *ApprovalWorkflow) CreateApprovalRequest(ctx context.Context, req *ApprovalRequest) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	// 生成唯一ID
	req.ID = generateApprovalID()
	req.Status = ApprovalStatusPending
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()
	
	// 设置默认过期时间（7天）
	if req.ExpiresAt.IsZero() {
		req.ExpiresAt = time.Now().AddDate(0, 0, 7)
	}
	
	// 根据审批类型确定审批人
	approvers, err := w.determineApprovers(ctx, req.Type, req.RequesterID)
	if err != nil {
		return fmt.Errorf("failed to determine approvers: %w", err)
	}
	req.Approvers = approvers
	
	w.requests[req.ID] = req
	return nil
}

// GetApprovalRequest 获取审批请求
func (w *ApprovalWorkflow) GetApprovalRequest(ctx context.Context, id string) (*ApprovalRequest, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	
	req, exists := w.requests[id]
	if !exists {
		return nil, fmt.Errorf("approval request not found: %s", id)
	}
	
	// 检查是否过期
	if time.Now().After(req.ExpiresAt) && req.Status == ApprovalStatusPending {
		req.Status = ApprovalStatusExpired
	}
	
	return req, nil
}

// Approve 审批通过
func (w *ApprovalWorkflow) Approve(ctx context.Context, requestID, approverID, comment string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	req, exists := w.requests[requestID]
	if !exists {
		return fmt.Errorf("approval request not found: %s", requestID)
	}
	
	// 检查审批人是否有权限
	if !w.isApprover(req, approverID) {
		return fmt.Errorf("user %s is not an approver for request %s", approverID, requestID)
	}
	
	// 检查是否已经审批过
	if w.hasApproved(req, approverID) {
		return fmt.Errorf("user %s has already approved request %s", approverID, requestID)
	}
	
	// 检查是否过期
	if time.Now().After(req.ExpiresAt) {
		req.Status = ApprovalStatusExpired
		return fmt.Errorf("approval request %s has expired", requestID)
	}
	
	// 创建审批记录
	approval := &Approval{
		ID:         generateApprovalID(),
		RequestID:  requestID,
		ApproverID: approverID,
		Status:     ApprovalStatusApproved,
		Comment:    comment,
		CreatedAt:  time.Now(),
	}
	
	req.Approvals = append(req.Approvals, approval)
	req.UpdatedAt = time.Now()
	
	// 检查是否所有审批人都已审批
	if w.allApproversApproved(req) {
		req.Status = ApprovalStatusApproved
	}
	
	return nil
}

// Reject 审批拒绝
func (w *ApprovalWorkflow) Reject(ctx context.Context, requestID, approverID, comment string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	req, exists := w.requests[requestID]
	if !exists {
		return fmt.Errorf("approval request not found: %s", requestID)
	}
	
	// 检查审批人是否有权限
	if !w.isApprover(req, approverID) {
		return fmt.Errorf("user %s is not an approver for request %s", approverID, requestID)
	}
	
	// 检查是否已经审批过
	if w.hasApproved(req, approverID) {
		return fmt.Errorf("user %s has already approved request %s", approverID, requestID)
	}
	
	// 检查是否过期
	if time.Now().After(req.ExpiresAt) {
		req.Status = ApprovalStatusExpired
		return fmt.Errorf("approval request %s has expired", requestID)
	}
	
	// 创建审批记录
	approval := &Approval{
		ID:         generateApprovalID(),
		RequestID:  requestID,
		ApproverID: approverID,
		Status:     ApprovalStatusRejected,
		Comment:    comment,
		CreatedAt:  time.Now(),
	}
	
	req.Approvals = append(req.Approvals, approval)
	req.Status = ApprovalStatusRejected
	req.UpdatedAt = time.Now()
	
	return nil
}

// ListApprovalRequests 列出审批请求
func (w *ApprovalWorkflow) ListApprovalRequests(ctx context.Context, userID string, status ApprovalStatus) ([]*ApprovalRequest, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	
	var requests []*ApprovalRequest
	
	for _, req := range w.requests {
		// 检查是否过期
		if time.Now().After(req.ExpiresAt) && req.Status == ApprovalStatusPending {
			req.Status = ApprovalStatusExpired
		}
		
		// 过滤状态
		if status != "" && req.Status != status {
			continue
		}
		
		// 过滤用户相关的请求（申请人或审批人）
		if req.RequesterID == userID || w.isApprover(req, userID) {
			requests = append(requests, req)
		}
	}
	
	return requests, nil
}

// GetPendingApprovals 获取待审批的请求
func (w *ApprovalWorkflow) GetPendingApprovals(ctx context.Context, userID string) ([]*ApprovalRequest, error) {
	return w.ListApprovalRequests(ctx, userID, ApprovalStatusPending)
}

// CancelApprovalRequest 取消审批请求
func (w *ApprovalWorkflow) CancelApprovalRequest(ctx context.Context, requestID, userID string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	req, exists := w.requests[requestID]
	if !exists {
		return fmt.Errorf("approval request not found: %s", requestID)
	}
	
	// 只有申请人可以取消
	if req.RequesterID != userID {
		return fmt.Errorf("only the requester can cancel the approval request")
	}
	
	// 只有待审批状态可以取消
	if req.Status != ApprovalStatusPending {
		return fmt.Errorf("cannot cancel approval request with status: %s", req.Status)
	}
	
	delete(w.requests, requestID)
	return nil
}

// 确定审批人
func (w *ApprovalWorkflow) determineApprovers(ctx context.Context, approvalType ApprovalType, requesterID string) ([]string, error) {
	var approvers []string
	
	switch approvalType {
	case ApprovalTypeStrategy:
		// 策略审批：需要交易员和风控人员
		approvers = w.findUsersWithPermission(ctx, "strategy:execute")
		approvers = append(approvers, w.findUsersWithPermission(ctx, "risk:write")...)
		
	case ApprovalTypeRiskLimit:
		// 风控限额审批：需要风控人员和系统管理员
		approvers = w.findUsersWithPermission(ctx, "risk:write")
		approvers = append(approvers, w.findUsersWithPermission(ctx, "system:admin")...)
		
	case ApprovalTypeHotlist:
		// 热门币种审批：需要分析师和风控人员
		approvers = w.findUsersWithPermission(ctx, "hotlist:approve")
		approvers = append(approvers, w.findUsersWithPermission(ctx, "risk:write")...)
		
	case ApprovalTypeAPIKey:
		// API密钥审批：需要系统管理员
		approvers = w.findUsersWithPermission(ctx, "apikey:write")
		
	case ApprovalTypeSystem:
		// 系统设置审批：需要系统管理员
		approvers = w.findUsersWithPermission(ctx, "system:admin")
		
	default:
		return nil, fmt.Errorf("unknown approval type: %s", approvalType)
	}
	
	// 移除申请人自己
	approvers = w.removeUser(approvers, requesterID)
	
	// 确保至少有两个审批人（4-eyes原则）
	if len(approvers) < 2 {
		// 如果没有足够的审批人，添加系统管理员
		adminUsers := w.findUsersWithPermission(ctx, "system:admin")
		for _, admin := range adminUsers {
			if admin != requesterID && !w.containsUser(approvers, admin) {
				approvers = append(approvers, admin)
			}
		}
	}
	
	// 如果仍然没有足够的审批人，返回错误
	if len(approvers) < 2 {
		return nil, fmt.Errorf("insufficient approvers for approval type: %s", approvalType)
	}
	
	return approvers, nil
}

// 查找具有特定权限的用户
func (w *ApprovalWorkflow) findUsersWithPermission(ctx context.Context, permission string) []string {
	var users []string
	
	allUsers, err := w.rbac.ListUsers(ctx)
	if err != nil {
		return users
	}
	
	for _, user := range allUsers {
		if !user.IsActive {
			continue
		}
		
		hasPermission, err := w.rbac.CheckPermission(ctx, user.ID, permission, "")
		if err == nil && hasPermission {
			users = append(users, user.ID)
		}
	}
	
	return users
}

// 检查用户是否是审批人
func (w *ApprovalWorkflow) isApprover(req *ApprovalRequest, userID string) bool {
	return w.containsUser(req.Approvers, userID)
}

// 检查用户是否已经审批
func (w *ApprovalWorkflow) hasApproved(req *ApprovalRequest, userID string) bool {
	for _, approval := range req.Approvals {
		if approval.ApproverID == userID {
			return true
		}
	}
	return false
}

// 检查是否所有审批人都已审批
func (w *ApprovalWorkflow) allApproversApproved(req *ApprovalRequest) bool {
	approvedCount := 0
	for _, approval := range req.Approvals {
		if approval.Status == ApprovalStatusApproved {
			approvedCount++
		}
	}
	return approvedCount >= len(req.Approvers)
}

// 从用户列表中移除指定用户
func (w *ApprovalWorkflow) removeUser(users []string, userID string) []string {
	var result []string
	for _, user := range users {
		if user != userID {
			result = append(result, user)
		}
	}
	return result
}

// 检查用户列表中是否包含指定用户
func (w *ApprovalWorkflow) containsUser(users []string, userID string) bool {
	for _, user := range users {
		if user == userID {
			return true
		}
	}
	return false
}

// 生成审批ID
func generateApprovalID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("approval_%x", bytes)
}
