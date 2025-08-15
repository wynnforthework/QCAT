package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"qcat/internal/auth"
	"qcat/internal/database"
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

	// TODO: Implement actual user authentication against database
	// For now, we'll use a mock authentication
	if req.Username == "admin" && req.Password == "password" {
		// Generate tokens
		accessToken, err := h.jwtManager.GenerateToken("1", req.Username, "admin")
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Error:   "Failed to generate token",
			})
			return
		}

		refreshToken, err := h.jwtManager.GenerateToken("1", req.Username, "admin")
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Error:   "Failed to generate refresh token",
			})
			return
		}

		c.JSON(http.StatusOK, Response{
			Success: true,
			Data: AuthResponse{
				AccessToken:  accessToken,
				RefreshToken: refreshToken,
				ExpiresAt:    time.Now().Add(24 * time.Hour),
				UserID:       "1",
				Username:     req.Username,
				Role:         "admin",
			},
		})
		return
	}

	c.JSON(http.StatusUnauthorized, Response{
		Success: false,
		Error:   "Invalid credentials",
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

	// TODO: Implement actual user registration against database
	// For now, we'll use a mock registration
	if req.Username == "admin" {
		c.JSON(http.StatusConflict, Response{
			Success: false,
			Error:   "Username already exists",
		})
		return
	}

	// Generate tokens for new user
	accessToken, err := h.jwtManager.GenerateToken("2", req.Username, "user")
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to generate token",
		})
		return
	}

	refreshToken, err := h.jwtManager.GenerateToken("2", req.Username, "user")
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to generate refresh token",
		})
		return
	}

	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data: AuthResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresAt:    time.Now().Add(24 * time.Hour),
			UserID:       "2",
			Username:     req.Username,
			Role:         "user",
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

	// Validate refresh token
	claims, err := h.jwtManager.ValidateToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, Response{
			Success: false,
			Error:   "Invalid refresh token",
		})
		return
	}

	// Generate new access token
	accessToken, err := h.jwtManager.GenerateToken(claims.UserID, claims.Username, claims.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error:   "Failed to generate new token",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: AuthResponse{
			AccessToken:  accessToken,
			RefreshToken: req.RefreshToken, // Keep the same refresh token
			ExpiresAt:    time.Now().Add(24 * time.Hour),
			UserID:       claims.UserID,
			Username:     claims.Username,
			Role:         claims.Role,
		},
	})
}
