package api

import (
	"net/http"

	"qcat/internal/security"

	"github.com/gin-gonic/gin"
)

// SecurityHandler 安全相关处理器
type SecurityHandler struct {
	kms      *security.KMS
	rbac     *security.RBAC
	workflow *security.ApprovalWorkflow
}

// NewSecurityHandler 创建安全处理器
func NewSecurityHandler(kms *security.KMS, rbac *security.RBAC, workflow *security.ApprovalWorkflow) *SecurityHandler {
	return &SecurityHandler{
		kms:      kms,
		rbac:     rbac,
		workflow: workflow,
	}
}

// ==================== KMS API ====================

// CreateAPIKeyRequest 创建API密钥请求
type CreateAPIKeyRequest struct {
	Name        string   `json:"name" binding:"required"`
	Exchange    string   `json:"exchange" binding:"required"`
	Key         string   `json:"key" binding:"required"`
	Secret      string   `json:"secret" binding:"required"`
	Permissions []string `json:"permissions"`
}

// CreateAPIKey 创建API密钥
func (h *SecurityHandler) CreateAPIKey(c *gin.Context) {
	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	apiKey, err := h.kms.CreateAPIKey(c.Request.Context(), req.Name, req.Exchange, req.Key, req.Secret, req.Permissions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, apiKey)
}

// GetAPIKey 获取API密钥
func (h *SecurityHandler) GetAPIKey(c *gin.Context) {
	id := c.Param("id")

	apiKey, err := h.kms.GetAPIKey(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, apiKey)
}

// ListAPIKeys 列出所有API密钥
func (h *SecurityHandler) ListAPIKeys(c *gin.Context) {
	keys, err := h.kms.ListAPIKeys(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, keys)
}

// UpdateAPIKey 更新API密钥
func (h *SecurityHandler) UpdateAPIKey(c *gin.Context) {
	id := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.kms.UpdateAPIKey(c.Request.Context(), id, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key updated successfully"})
}

// DeleteAPIKey 删除API密钥
func (h *SecurityHandler) DeleteAPIKey(c *gin.Context) {
	id := c.Param("id")

	err := h.kms.DeleteAPIKey(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key deleted successfully"})
}

// RotateKeys 轮换密钥
func (h *SecurityHandler) RotateKeys(c *gin.Context) {
	err := h.kms.RotateKeys(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Keys rotated successfully"})
}

// ==================== RBAC API ====================

// CreateRoleRequest 创建角色请求
type CreateRoleRequest struct {
	ID          string            `json:"id" binding:"required"`
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description"`
	Permissions []string          `json:"permissions"`
	Metadata    map[string]string `json:"metadata"`
}

// CreateRole 创建角色
func (h *SecurityHandler) CreateRole(c *gin.Context) {
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	role := &security.Role{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		Permissions: req.Permissions,
		Metadata:    req.Metadata,
	}

	err := h.rbac.CreateRole(c.Request.Context(), role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, role)
}

// GetRole 获取角色
func (h *SecurityHandler) GetRole(c *gin.Context) {
	id := c.Param("id")

	role, err := h.rbac.GetRole(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, role)
}

// ListRoles 列出所有角色
func (h *SecurityHandler) ListRoles(c *gin.Context) {
	roles, err := h.rbac.ListRoles(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, roles)
}

// UpdateRole 更新角色
func (h *SecurityHandler) UpdateRole(c *gin.Context) {
	id := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.rbac.UpdateRole(c.Request.Context(), id, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role updated successfully"})
}

// DeleteRole 删除角色
func (h *SecurityHandler) DeleteRole(c *gin.Context) {
	id := c.Param("id")

	err := h.rbac.DeleteRole(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role deleted successfully"})
}

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	ID       string   `json:"id" binding:"required"`
	Username string   `json:"username" binding:"required"`
	Email    string   `json:"email" binding:"required,email"`
	Roles    []string `json:"roles"`
	IsActive bool     `json:"is_active"`
}

// CreateUser 创建用户
func (h *SecurityHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := &security.User{
		ID:       req.ID,
		Username: req.Username,
		Email:    req.Email,
		Roles:    req.Roles,
		IsActive: req.IsActive,
	}

	err := h.rbac.CreateUser(c.Request.Context(), user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// GetUser 获取用户
func (h *SecurityHandler) GetUser(c *gin.Context) {
	id := c.Param("id")

	user, err := h.rbac.GetUser(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// ListUsers 列出所有用户
func (h *SecurityHandler) ListUsers(c *gin.Context) {
	users, err := h.rbac.ListUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

// UpdateUser 更新用户
func (h *SecurityHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.rbac.UpdateUser(c.Request.Context(), id, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

// DeleteUser 删除用户
func (h *SecurityHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")

	err := h.rbac.DeleteUser(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// CheckPermissionRequest 检查权限请求
type CheckPermissionRequest struct {
	UserID   string `json:"user_id" binding:"required"`
	Resource string `json:"resource" binding:"required"`
	Action   string `json:"action" binding:"required"`
}

// CheckPermission 检查权限
func (h *SecurityHandler) CheckPermission(c *gin.Context) {
	var req CheckPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hasPermission, err := h.rbac.CheckPermission(c.Request.Context(), req.UserID, req.Resource, req.Action)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"has_permission": hasPermission})
}

// GetUserPermissions 获取用户权限
func (h *SecurityHandler) GetUserPermissions(c *gin.Context) {
	userID := c.Param("id")

	permissions, err := h.rbac.GetUserPermissions(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"permissions": permissions})
}

// AssignRoleRequest 分配角色请求
type AssignRoleRequest struct {
	RoleID string `json:"role_id" binding:"required"`
}

// AssignRole 分配角色
func (h *SecurityHandler) AssignRole(c *gin.Context) {
	userID := c.Param("id")

	var req AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.rbac.AssignRole(c.Request.Context(), userID, req.RoleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role assigned successfully"})
}

// RemoveRole 移除角色
func (h *SecurityHandler) RemoveRole(c *gin.Context) {
	userID := c.Param("id")
	roleID := c.Param("role_id")

	err := h.rbac.RemoveRole(c.Request.Context(), userID, roleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role removed successfully"})
}

// ==================== 审批流程 API ====================

// CreateApprovalRequest 创建审批请求
type CreateApprovalRequest struct {
	Type        security.ApprovalType  `json:"type" binding:"required"`
	Title       string                 `json:"title" binding:"required"`
	Description string                 `json:"description"`
	RequesterID string                 `json:"requester_id" binding:"required"`
	Data        map[string]interface{} `json:"data"`
	Priority    int                    `json:"priority"`
	ExpiresAt   string                 `json:"expires_at"`
}

// CreateApproval 创建审批请求
func (h *SecurityHandler) CreateApproval(c *gin.Context) {
	var req CreateApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	approvalReq := &security.ApprovalRequest{
		Type:        req.Type,
		Title:       req.Title,
		Description: req.Description,
		RequesterID: req.RequesterID,
		Data:        req.Data,
		Priority:    req.Priority,
	}

	err := h.workflow.CreateApprovalRequest(c.Request.Context(), approvalReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, approvalReq)
}

// GetApproval 获取审批请求
func (h *SecurityHandler) GetApproval(c *gin.Context) {
	id := c.Param("id")

	approval, err := h.workflow.GetApprovalRequest(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, approval)
}

// ListApprovals 列出审批请求
func (h *SecurityHandler) ListApprovals(c *gin.Context) {
	userID := c.Query("user_id")
	status := c.Query("status")

	var approvalStatus security.ApprovalStatus
	if status != "" {
		approvalStatus = security.ApprovalStatus(status)
	}

	approvals, err := h.workflow.ListApprovalRequests(c.Request.Context(), userID, approvalStatus)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, approvals)
}

// GetPendingApprovals 获取待审批请求
func (h *SecurityHandler) GetPendingApprovals(c *gin.Context) {
	userID := c.Param("id")

	approvals, err := h.workflow.GetPendingApprovals(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, approvals)
}

// ApproveRequest 审批通过请求
type ApproveRequest struct {
	Comment string `json:"comment"`
}

// Approve 审批通过
func (h *SecurityHandler) Approve(c *gin.Context) {
	requestID := c.Param("id")
	approverID := c.GetString("user_id") // 从JWT中获取

	var req ApproveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.workflow.Approve(c.Request.Context(), requestID, approverID, req.Comment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Approval granted successfully"})
}

// Reject 审批拒绝
func (h *SecurityHandler) Reject(c *gin.Context) {
	requestID := c.Param("id")
	approverID := c.GetString("user_id") // 从JWT中获取

	var req ApproveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.workflow.Reject(c.Request.Context(), requestID, approverID, req.Comment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Approval rejected successfully"})
}

// CancelApproval 取消审批请求
func (h *SecurityHandler) CancelApproval(c *gin.Context) {
	requestID := c.Param("id")
	userID := c.GetString("user_id") // 从JWT中获取

	err := h.workflow.CancelApprovalRequest(c.Request.Context(), requestID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Approval request cancelled successfully"})
}
