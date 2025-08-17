package security

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// SecurityMiddleware provides security-related middleware functions
type SecurityMiddleware struct {
	keyManager  *KeyManager
	auditLogger *AuditLogger
}

// NewSecurityMiddleware creates a new security middleware
func NewSecurityMiddleware(keyManager *KeyManager, auditLogger *AuditLogger) *SecurityMiddleware {
	return &SecurityMiddleware{
		keyManager:  keyManager,
		auditLogger: auditLogger,
	}
}

// APIKeyAuth middleware validates API keys
func (sm *SecurityMiddleware) APIKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract API key from header
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			// Try Authorization header with Bearer token
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				apiKey = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if apiKey == "" {
			sm.logAPIAccess(c, "", http.StatusUnauthorized, time.Now())
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "API key is required",
			})
			c.Abort()
			return
		}

		// Validate API key
		startTime := time.Now()
		keyInfo, err := sm.keyManager.ValidateKey(apiKey)
		if err != nil {
			sm.logAPIAccess(c, "", http.StatusUnauthorized, startTime)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid API key",
			})
			c.Abort()
			return
		}

		// Store key info in context
		c.Set("api_key_info", keyInfo)
		c.Set("user_id", keyInfo.ID)

		// Log successful API access
		sm.logAPIAccess(c, keyInfo.ID, http.StatusOK, startTime)

		c.Next()
	}
}

// RequirePermission middleware checks if the API key has required permission
func (sm *SecurityMiddleware) RequirePermission(permission KeyPermission) gin.HandlerFunc {
	return func(c *gin.Context) {
		keyInfoInterface, exists := c.Get("api_key_info")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			c.Abort()
			return
		}

		keyInfo, ok := keyInfoInterface.(*KeyInfo)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Invalid key info",
			})
			c.Abort()
			return
		}

		// Check permission
		if !sm.keyManager.CheckPermission(keyInfo, permission) {
			sm.auditLogger.LogUserAction(keyInfo.ID, "permission_denied", string(permission), map[string]interface{}{
				"required_permission": permission,
				"user_permissions":    keyInfo.Permissions,
				"endpoint":           c.Request.URL.Path,
				"method":             c.Request.Method,
			}, false, "Insufficient permissions")

			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions",
				"required_permission": permission,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitByKey middleware implements rate limiting per API key
func (sm *SecurityMiddleware) RateLimitByKey(requestsPerMinute int) gin.HandlerFunc {
	return func(c *gin.Context) {
		keyInfoInterface, exists := c.Get("api_key_info")
		if !exists {
			c.Next()
			return
		}

		keyInfo, ok := keyInfoInterface.(*KeyInfo)
		if !ok {
			c.Next()
			return
		}

		// Simple rate limiting implementation
		// In production, you'd want to use Redis or similar for distributed rate limiting
		if sm.isRateLimited(keyInfo.ID, requestsPerMinute) {
			sm.auditLogger.LogUserAction(keyInfo.ID, "rate_limit_exceeded", c.Request.URL.Path, map[string]interface{}{
				"requests_per_minute": requestsPerMinute,
				"endpoint":           c.Request.URL.Path,
				"method":             c.Request.Method,
			}, false, "Rate limit exceeded")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
				"limit": requestsPerMinute,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// AuditLog middleware logs all API requests
func (sm *SecurityMiddleware) AuditLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Process request
		c.Next()

		// Log after request is processed
		duration := time.Since(startTime)
		userID := sm.getUserID(c)
		
		sm.auditLogger.LogAPIAccess(
			userID,
			c.Request.Method,
			c.Request.URL.Path,
			c.ClientIP(),
			c.GetHeader("User-Agent"),
			c.Writer.Status(),
			duration,
		)
	}
}

// SecurityHeaders middleware adds security headers
func (sm *SecurityMiddleware) SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		c.Next()
	}
}

// IPWhitelist middleware restricts access to whitelisted IP addresses
func (sm *SecurityMiddleware) IPWhitelist(allowedIPs []string) gin.HandlerFunc {
	allowedIPMap := make(map[string]bool)
	for _, ip := range allowedIPs {
		allowedIPMap[ip] = true
	}

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		
		if !allowedIPMap[clientIP] {
			userID := sm.getUserID(c)
			sm.auditLogger.LogSecurityEvent("ip_blocked", "Access denied from non-whitelisted IP", map[string]interface{}{
				"client_ip":     clientIP,
				"user_id":       userID,
				"endpoint":      c.Request.URL.Path,
				"method":        c.Request.Method,
				"allowed_ips":   allowedIPs,
			})

			c.JSON(http.StatusForbidden, gin.H{
				"error": "Access denied from this IP address",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// DetectSuspiciousActivity middleware detects and logs suspicious activity
func (sm *SecurityMiddleware) DetectSuspiciousActivity() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := sm.getUserID(c)
		clientIP := c.ClientIP()
		userAgent := c.GetHeader("User-Agent")

		// Check for suspicious patterns
		suspicious := false
		reasons := make([]string, 0)

		// Check for unusual user agent
		if userAgent == "" || strings.Contains(strings.ToLower(userAgent), "bot") {
			suspicious = true
			reasons = append(reasons, "suspicious_user_agent")
		}

		// Check for unusual request patterns
		if c.Request.Method == "POST" && c.Request.ContentLength > 10*1024*1024 { // 10MB
			suspicious = true
			reasons = append(reasons, "large_request_body")
		}

		// Check for SQL injection patterns in query parameters
		for key, values := range c.Request.URL.Query() {
			for _, value := range values {
				if sm.containsSQLInjectionPattern(value) {
					suspicious = true
					reasons = append(reasons, "sql_injection_attempt")
					break
				}
			}
		}

		if suspicious {
			sm.auditLogger.LogSecurityEvent("suspicious_activity", "Suspicious activity detected", map[string]interface{}{
				"user_id":     userID,
				"client_ip":   clientIP,
				"user_agent":  userAgent,
				"endpoint":    c.Request.URL.Path,
				"method":      c.Request.Method,
				"reasons":     reasons,
			})

			// Optionally block the request
			if len(reasons) > 1 { // Multiple suspicious indicators
				c.JSON(http.StatusForbidden, gin.H{
					"error": "Suspicious activity detected",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// Helper methods

func (sm *SecurityMiddleware) logAPIAccess(c *gin.Context, userID string, statusCode int, startTime time.Time) {
	duration := time.Since(startTime)
	
	if userID == "" {
		userID = "anonymous"
	}

	sm.auditLogger.LogAPIAccess(
		userID,
		c.Request.Method,
		c.Request.URL.Path,
		c.ClientIP(),
		c.GetHeader("User-Agent"),
		statusCode,
		duration,
	)
}

func (sm *SecurityMiddleware) getUserID(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(string); ok {
			return uid
		}
	}
	return "anonymous"
}

func (sm *SecurityMiddleware) isRateLimited(keyID string, requestsPerMinute int) bool {
	// This is a simplified implementation
	// In production, you'd use Redis or similar for distributed rate limiting
	// For now, we'll just return false (no rate limiting)
	return false
}

func (sm *SecurityMiddleware) containsSQLInjectionPattern(value string) bool {
	lowerValue := strings.ToLower(value)
	patterns := []string{
		"union select",
		"drop table",
		"delete from",
		"insert into",
		"update set",
		"' or '1'='1",
		"' or 1=1",
		"'; drop",
		"'; delete",
		"'; insert",
		"'; update",
	}

	for _, pattern := range patterns {
		if strings.Contains(lowerValue, pattern) {
			return true
		}
	}

	return false
}