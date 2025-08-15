package security

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Role 角色定义
type Role struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Permissions []string          `json:"permissions"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// User 用户定义
type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Roles     []string  `json:"roles"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Permission 权限定义
type Permission struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
}

// RBAC 基于角色的访问控制系统
type RBAC struct {
	mu          sync.RWMutex
	roles       map[string]*Role
	users       map[string]*User
	permissions map[string]*Permission
}

// NewRBAC 创建RBAC系统
func NewRBAC() *RBAC {
	rbac := &RBAC{
		roles:       make(map[string]*Role),
		users:       make(map[string]*User),
		permissions: make(map[string]*Permission),
	}
	
	// 初始化默认权限
	rbac.initializeDefaultPermissions()
	
	// 初始化默认角色
	rbac.initializeDefaultRoles()
	
	return rbac
}

// 初始化默认权限
func (r *RBAC) initializeDefaultPermissions() {
	defaultPermissions := []*Permission{
		// 策略管理权限
		{ID: "strategy:read", Name: "策略读取", Description: "读取策略信息", Resource: "strategy", Action: "read"},
		{ID: "strategy:write", Name: "策略写入", Description: "创建和修改策略", Resource: "strategy", Action: "write"},
		{ID: "strategy:delete", Name: "策略删除", Description: "删除策略", Resource: "strategy", Action: "delete"},
		{ID: "strategy:execute", Name: "策略执行", Description: "执行策略", Resource: "strategy", Action: "execute"},
		
		// 优化器权限
		{ID: "optimizer:read", Name: "优化器读取", Description: "读取优化结果", Resource: "optimizer", Action: "read"},
		{ID: "optimizer:write", Name: "优化器写入", Description: "创建优化任务", Resource: "optimizer", Action: "write"},
		{ID: "optimizer:execute", Name: "优化器执行", Description: "执行优化", Resource: "optimizer", Action: "execute"},
		
		// 投资组合权限
		{ID: "portfolio:read", Name: "投资组合读取", Description: "读取投资组合信息", Resource: "portfolio", Action: "read"},
		{ID: "portfolio:write", Name: "投资组合写入", Description: "修改投资组合", Resource: "portfolio", Action: "write"},
		{ID: "portfolio:rebalance", Name: "投资组合再平衡", Description: "执行再平衡", Resource: "portfolio", Action: "rebalance"},
		
		// 风控权限
		{ID: "risk:read", Name: "风控读取", Description: "读取风控信息", Resource: "risk", Action: "read"},
		{ID: "risk:write", Name: "风控写入", Description: "修改风控设置", Resource: "risk", Action: "write"},
		{ID: "risk:override", Name: "风控覆盖", Description: "覆盖风控限制", Resource: "risk", Action: "override"},
		
		// 热门币种权限
		{ID: "hotlist:read", Name: "热门币种读取", Description: "读取热门币种列表", Resource: "hotlist", Action: "read"},
		{ID: "hotlist:write", Name: "热门币种写入", Description: "修改热门币种", Resource: "hotlist", Action: "write"},
		{ID: "hotlist:approve", Name: "热门币种审批", Description: "审批热门币种", Resource: "hotlist", Action: "approve"},
		
		// 审计权限
		{ID: "audit:read", Name: "审计读取", Description: "读取审计日志", Resource: "audit", Action: "read"},
		{ID: "audit:export", Name: "审计导出", Description: "导出审计日志", Resource: "audit", Action: "export"},
		
		// 系统管理权限
		{ID: "system:read", Name: "系统读取", Description: "读取系统信息", Resource: "system", Action: "read"},
		{ID: "system:write", Name: "系统写入", Description: "修改系统设置", Resource: "system", Action: "write"},
		{ID: "system:admin", Name: "系统管理", Description: "系统管理员权限", Resource: "system", Action: "admin"},
		
		// API密钥管理权限
		{ID: "apikey:read", Name: "API密钥读取", Description: "读取API密钥信息", Resource: "apikey", Action: "read"},
		{ID: "apikey:write", Name: "API密钥写入", Description: "创建和修改API密钥", Resource: "apikey", Action: "write"},
		{ID: "apikey:delete", Name: "API密钥删除", Description: "删除API密钥", Resource: "apikey", Action: "delete"},
	}
	
	for _, perm := range defaultPermissions {
		r.permissions[perm.ID] = perm
	}
}

// 初始化默认角色
func (r *RBAC) initializeDefaultRoles() {
	defaultRoles := []*Role{
		{
			ID:          "admin",
			Name:        "系统管理员",
			Description: "拥有所有权限的系统管理员",
			Permissions: []string{
				"strategy:read", "strategy:write", "strategy:delete", "strategy:execute",
				"optimizer:read", "optimizer:write", "optimizer:execute",
				"portfolio:read", "portfolio:write", "portfolio:rebalance",
				"risk:read", "risk:write", "risk:override",
				"hotlist:read", "hotlist:write", "hotlist:approve",
				"audit:read", "audit:export",
				"system:read", "system:write", "system:admin",
				"apikey:read", "apikey:write", "apikey:delete",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:          "trader",
			Name:        "交易员",
			Description: "负责策略执行和交易操作",
			Permissions: []string{
				"strategy:read", "strategy:write", "strategy:execute",
				"optimizer:read", "optimizer:write",
				"portfolio:read", "portfolio:rebalance",
				"risk:read",
				"hotlist:read",
				"audit:read",
				"system:read",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:          "analyst",
			Name:        "分析师",
			Description: "负责策略分析和优化",
			Permissions: []string{
				"strategy:read", "strategy:write",
				"optimizer:read", "optimizer:write", "optimizer:execute",
				"portfolio:read",
				"risk:read",
				"hotlist:read",
				"audit:read",
				"system:read",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:          "viewer",
			Name:        "观察者",
			Description: "只读权限，用于监控和查看",
			Permissions: []string{
				"strategy:read",
				"optimizer:read",
				"portfolio:read",
				"risk:read",
				"hotlist:read",
				"audit:read",
				"system:read",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	
	for _, role := range defaultRoles {
		r.roles[role.ID] = role
	}
}

// CreateRole 创建角色
func (r *RBAC) CreateRole(ctx context.Context, role *Role) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.roles[role.ID]; exists {
		return fmt.Errorf("role already exists: %s", role.ID)
	}
	
	role.CreatedAt = time.Now()
	role.UpdatedAt = time.Now()
	r.roles[role.ID] = role
	
	return nil
}

// GetRole 获取角色
func (r *RBAC) GetRole(ctx context.Context, id string) (*Role, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	role, exists := r.roles[id]
	if !exists {
		return nil, fmt.Errorf("role not found: %s", id)
	}
	
	return role, nil
}

// UpdateRole 更新角色
func (r *RBAC) UpdateRole(ctx context.Context, id string, updates map[string]interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	role, exists := r.roles[id]
	if !exists {
		return fmt.Errorf("role not found: %s", id)
	}
	
	if name, ok := updates["name"].(string); ok {
		role.Name = name
	}
	
	if description, ok := updates["description"].(string); ok {
		role.Description = description
	}
	
	if permissions, ok := updates["permissions"].([]string); ok {
		role.Permissions = permissions
	}
	
	if metadata, ok := updates["metadata"].(map[string]string); ok {
		role.Metadata = metadata
	}
	
	role.UpdatedAt = time.Now()
	
	return nil
}

// DeleteRole 删除角色
func (r *RBAC) DeleteRole(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.roles[id]; !exists {
		return fmt.Errorf("role not found: %s", id)
	}
	
	// 检查是否有用户使用此角色
	for _, user := range r.users {
		for _, roleID := range user.Roles {
			if roleID == id {
				return fmt.Errorf("cannot delete role %s: still in use by user %s", id, user.ID)
			}
		}
	}
	
	delete(r.roles, id)
	return nil
}

// ListRoles 列出所有角色
func (r *RBAC) ListRoles(ctx context.Context) ([]*Role, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	roles := make([]*Role, 0, len(r.roles))
	for _, role := range r.roles {
		roles = append(roles, role)
	}
	
	return roles, nil
}

// CreateUser 创建用户
func (r *RBAC) CreateUser(ctx context.Context, user *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.users[user.ID]; exists {
		return fmt.Errorf("user already exists: %s", user.ID)
	}
	
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	r.users[user.ID] = user
	
	return nil
}

// GetUser 获取用户
func (r *RBAC) GetUser(ctx context.Context, id string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	user, exists := r.users[id]
	if !exists {
		return nil, fmt.Errorf("user not found: %s", id)
	}
	
	return user, nil
}

// UpdateUser 更新用户
func (r *RBAC) UpdateUser(ctx context.Context, id string, updates map[string]interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	user, exists := r.users[id]
	if !exists {
		return fmt.Errorf("user not found: %s", id)
	}
	
	if username, ok := updates["username"].(string); ok {
		user.Username = username
	}
	
	if email, ok := updates["email"].(string); ok {
		user.Email = email
	}
	
	if roles, ok := updates["roles"].([]string); ok {
		user.Roles = roles
	}
	
	if isActive, ok := updates["is_active"].(bool); ok {
		user.IsActive = isActive
	}
	
	user.UpdatedAt = time.Now()
	
	return nil
}

// DeleteUser 删除用户
func (r *RBAC) DeleteUser(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.users[id]; !exists {
		return fmt.Errorf("user not found: %s", id)
	}
	
	delete(r.users, id)
	return nil
}

// ListUsers 列出所有用户
func (r *RBAC) ListUsers(ctx context.Context) ([]*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	users := make([]*User, 0, len(r.users))
	for _, user := range r.users {
		users = append(users, user)
	}
	
	return users, nil
}

// CheckPermission 检查用户权限
func (r *RBAC) CheckPermission(ctx context.Context, userID, resource, action string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	user, exists := r.users[userID]
	if !exists {
		return false, fmt.Errorf("user not found: %s", userID)
	}
	
	if !user.IsActive {
		return false, nil
	}
	
	// 检查用户的所有角色权限
	for _, roleID := range user.Roles {
		role, exists := r.roles[roleID]
		if !exists {
			continue
		}
		
		permissionID := fmt.Sprintf("%s:%s", resource, action)
		for _, perm := range role.Permissions {
			if perm == permissionID {
				return true, nil
			}
		}
	}
	
	return false, nil
}

// GetUserPermissions 获取用户所有权限
func (r *RBAC) GetUserPermissions(ctx context.Context, userID string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	user, exists := r.users[userID]
	if !exists {
		return nil, fmt.Errorf("user not found: %s", userID)
	}
	
	if !user.IsActive {
		return []string{}, nil
	}
	
	permissions := make(map[string]bool)
	
	// 收集用户所有角色的权限
	for _, roleID := range user.Roles {
		role, exists := r.roles[roleID]
		if !exists {
			continue
		}
		
		for _, perm := range role.Permissions {
			permissions[perm] = true
		}
	}
	
	// 转换为切片
	result := make([]string, 0, len(permissions))
	for perm := range permissions {
		result = append(result, perm)
	}
	
	return result, nil
}

// AssignRole 为用户分配角色
func (r *RBAC) AssignRole(ctx context.Context, userID, roleID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	user, exists := r.users[userID]
	if !exists {
		return fmt.Errorf("user not found: %s", userID)
	}
	
	if _, exists := r.roles[roleID]; !exists {
		return fmt.Errorf("role not found: %s", roleID)
	}
	
	// 检查角色是否已分配
	for _, existingRole := range user.Roles {
		if existingRole == roleID {
			return fmt.Errorf("role %s already assigned to user %s", roleID, userID)
		}
	}
	
	user.Roles = append(user.Roles, roleID)
	user.UpdatedAt = time.Now()
	
	return nil
}

// RemoveRole 移除用户角色
func (r *RBAC) RemoveRole(ctx context.Context, userID, roleID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	user, exists := r.users[userID]
	if !exists {
		return fmt.Errorf("user not found: %s", userID)
	}
	
	// 查找并移除角色
	for i, existingRole := range user.Roles {
		if existingRole == roleID {
			user.Roles = append(user.Roles[:i], user.Roles[i+1:]...)
			user.UpdatedAt = time.Now()
			return nil
		}
	}
	
	return fmt.Errorf("role %s not assigned to user %s", roleID, userID)
}
