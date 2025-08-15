package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Claims represents JWT claims
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// JWTManager handles JWT operations
type JWTManager struct {
	secretKey []byte
	duration  time.Duration
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(secretKey string, duration time.Duration) *JWTManager {
	return &JWTManager{
		secretKey: []byte(secretKey),
		duration:  duration,
	}
}

// GenerateToken generates a new JWT token
func (j *JWTManager) GenerateToken(userID, username, role string) (string, error) {
	claims := &Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

// ValidateToken validates a JWT token
func (j *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// AuthMiddleware creates a JWT authentication middleware
func (j *JWTManager) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Bearer token required"})
			c.Abort()
			return
		}

		claims, err := j.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// RoleMiddleware creates a role-based authorization middleware
func (j *JWTManager) RoleMiddleware(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found"})
			c.Abort()
			return
		}

		role := userRole.(string)
		for _, requiredRole := range requiredRoles {
			if role == requiredRole {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		c.Abort()
	}
}

// GetUserFromContext extracts user information from context
func GetUserFromContext(c *gin.Context) (userID, username, role string, ok bool) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		return "", "", "", false
	}
	userID, ok = userIDVal.(string)
	if !ok {
		return "", "", "", false
	}

	usernameVal, exists := c.Get("username")
	if !exists {
		return "", "", "", false
	}
	username, ok = usernameVal.(string)
	if !ok {
		return "", "", "", false
	}

	roleVal, exists := c.Get("role")
	if !exists {
		return "", "", "", false
	}
	role, ok = roleVal.(string)
	if !ok {
		return "", "", "", false
	}

	return userID, username, role, true
}
