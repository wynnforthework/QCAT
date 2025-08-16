package api

import (
	"net/http"
	"time"

	"qcat/internal/auth"
	"qcat/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	jwtManager *auth.JWTManager
	db         *database.DB
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(jwtManager *auth.JWTManager, db *database.DB) *AuthHandler {
	return &AuthHandler{
		jwtManager: jwtManager,
		db:         db,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
}

// RefreshRequest represents a token refresh request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	UserID       string    `json:"user_id"`
	Username     string    `json:"username"`
	Role         string    `json:"role"`
}

// @Summary User login
// @Description Authenticate user and return JWT tokens
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} Response{data=AuthResponse}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// 实现真实的用户认证
	ctx := c.Request.Context()

	// 从数据库获取用户信息
	user, err := h.db.GetUserByUsername(ctx, req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, Response{
			Success: false,
			Error:   "Invalid credentials",
		})
		return
	}

	// 验证密码
	if err := database.ValidatePassword(req.Password, user.PasswordHash); err != nil {
		c.JSON(http.StatusUnauthorized, Response{
			Success: false,
			Error:   "Invalid credentials",
		})
		return
	}

	// 生成访问令牌
	accessToken, err := h.jwtManager.GenerateToken(user.ID.String(), user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to generate access token",
		})
		return
	}

	// 生成刷新令牌
	refreshToken, err := h.jwtManager.GenerateToken(user.ID.String(), user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to generate refresh token",
		})
		return
	}

	// 创建用户会话
	expiresAt := time.Now().Add(7 * 24 * time.Hour) // 7天过期
	_, err = h.db.CreateUserSession(ctx, user.ID, refreshToken, expiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to create user session",
		})
		return
	}

	// 更新最后登录时间
	if err := h.db.UpdateUserLastLogin(ctx, user.ID); err != nil {
		// 记录错误但不影响登录流程
		// log.Printf("Failed to update last login time: %v", err)
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: AuthResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresAt:    expiresAt,
			UserID:       user.ID.String(),
			Username:     user.Username,
			Role:         user.Role,
		},
	})
}

// @Summary User registration
// @Description Register a new user
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration data"
// @Success 201 {object} Response{data=AuthResponse}
// @Failure 400 {object} Response
// @Failure 409 {object} Response
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// 检查用户是否已存在
	exists, err := h.db.CheckUserExists(ctx, req.Username, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to check user existence",
		})
		return
	}

	if exists {
		c.JSON(http.StatusConflict, Response{
			Success: false,
			Error:   "Username or email already exists",
		})
		return
	}

	// 创建新用户
	user, err := h.db.CreateUser(ctx, req.Username, req.Email, req.Password, "user")
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to create user",
		})
		return
	}

	// 生成访问令牌
	accessToken, err := h.jwtManager.GenerateToken(user.ID.String(), user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to generate access token",
		})
		return
	}

	// 生成刷新令牌
	refreshToken, err := h.jwtManager.GenerateToken(user.ID.String(), user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to generate refresh token",
		})
		return
	}

	// 创建用户会话
	expiresAt := time.Now().Add(7 * 24 * time.Hour) // 7天过期
	_, err = h.db.CreateUserSession(ctx, user.ID, refreshToken, expiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to create user session",
		})
		return
	}

	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data: AuthResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresAt:    expiresAt,
			UserID:       user.ID.String(),
			Username:     req.Username,
			Role:         user.Role,
		},
	})
}

// @Summary Refresh token
// @Description Refresh access token using refresh token
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RefreshRequest true "Refresh token"
// @Success 200 {object} Response{data=AuthResponse}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// 验证刷新令牌
	claims, err := h.jwtManager.ValidateToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, Response{
			Success: false,
			Error:   "Invalid refresh token",
		})
		return
	}

	// 检查用户会话是否存在且未过期
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, Response{
			Success: false,
			Error:   "Invalid user ID in token",
		})
		return
	}

	session, err := h.db.GetUserSessionByToken(ctx, req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, Response{
			Success: false,
			Error:   "Session not found or expired",
		})
		return
	}

	// 验证会话是否属于该用户
	if session.UserID != userID {
		c.JSON(http.StatusUnauthorized, Response{
			Success: false,
			Error:   "Invalid session",
		})
		return
	}

	// 获取用户信息
	user, err := h.db.GetUserByID(ctx, userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, Response{
			Success: false,
			Error:   "User not found",
		})
		return
	}

	// 生成新的访问令牌
	accessToken, err := h.jwtManager.GenerateToken(user.ID.String(), user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to generate new access token",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: AuthResponse{
			AccessToken:  accessToken,
			RefreshToken: req.RefreshToken, // 保持相同的刷新令牌
			ExpiresAt:    session.ExpiresAt,
			UserID:       user.ID.String(),
			Username:     user.Username,
			Role:         user.Role,
		},
	})
}

// @Summary User logout
// @Description Logout user and invalidate refresh token
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RefreshRequest true "Refresh token to invalidate"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// 获取用户会话
	session, err := h.db.GetUserSessionByToken(ctx, req.RefreshToken)
	if err != nil {
		// 如果会话不存在，仍然返回成功（幂等性）
		c.JSON(http.StatusOK, Response{
			Success: true,
			Data:    map[string]string{"message": "Logged out successfully"},
		})
		return
	}

	// 删除用户会话
	if err := h.db.DeleteUserSession(ctx, session.ID); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to logout",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    map[string]string{"message": "Logged out successfully"},
	})
}

// @Summary Get current user profile
// @Description Get current user information
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Response{data=database.User}
// @Failure 401 {object} Response
// @Router /auth/profile [get]
func (h *AuthHandler) GetProfile(c *gin.Context) {
	// 从JWT令牌中获取用户ID
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, Response{
			Success: false,
			Error:   "Invalid user ID",
		})
		return
	}

	ctx := c.Request.Context()
	user, err := h.db.GetUserByID(ctx, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Success: false,
			Error:   "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    user,
	})
}
