package main

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents JWT claims
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func main() {
	// Use a default secret key (should match the one in config)
	secretKey := "qcat-secret-key-for-testing-only"
	
	// Create claims for a test admin user
	claims := &Claims{
		UserID:   "test-admin-user-id",
		Username: "admin",
		Role:     "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		fmt.Printf("Error generating token: %v\n", err)
		return
	}

	fmt.Printf("Test JWT Token:\n%s\n", tokenString)
	fmt.Printf("\nUse this token in Authorization header:\nAuthorization: Bearer %s\n", tokenString)
	
	// Also generate a curl command for testing
	fmt.Printf("\nTest curl command:\ncurl -H \"Authorization: Bearer %s\" http://localhost:8082/api/v1/audit/logs?limit=5\n", tokenString)
}
