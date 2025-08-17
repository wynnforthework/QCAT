package errors

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestNewAppError(t *testing.T) {
	err := NewAppError(ErrCodeInvalidInput, "Test error", nil)
	
	if err.Code != ErrCodeInvalidInput {
		t.Errorf("Expected code %s, got %s", ErrCodeInvalidInput, err.Code)
	}
	
	if err.Message != "Test error" {
		t.Errorf("Expected message 'Test error', got %s", err.Message)
	}
	
	if err.Severity != SeverityLow {
		t.Errorf("Expected severity %s, got %s", SeverityLow, err.Severity)
	}
}

func TestAppErrorHTTPStatus(t *testing.T) {
	tests := []struct {
		code           ErrorCode
		expectedStatus int
	}{
		{ErrCodeNotFound, http.StatusNotFound},
		{ErrCodeUnauthorized, http.StatusUnauthorized},
		{ErrCodeInvalidInput, http.StatusBadRequest},
		{ErrCodeInternal, http.StatusInternalServerError},
		{ErrCodeRateLimit, http.StatusTooManyRequests},
	}
	
	for _, test := range tests {
		err := NewAppError(test.code, "Test", nil)
		status := err.HTTPStatus()
		
		if status != test.expectedStatus {
			t.Errorf("Code %s: expected status %d, got %d", test.code, test.expectedStatus, status)
		}
	}
}

func TestAppErrorWithContext(t *testing.T) {
	err := NewAppError(ErrCodeInternal, "Test error", nil)
	err = err.WithContext("user_id", "123")
	err = err.WithRequestID("req_456")
	err = err.WithUserID("user_789")
	
	if err.Context["user_id"] != "123" {
		t.Errorf("Expected context user_id '123', got %v", err.Context["user_id"])
	}
	
	if err.RequestID != "req_456" {
		t.Errorf("Expected request ID 'req_456', got %s", err.RequestID)
	}
	
	if err.UserID != "user_789" {
		t.Errorf("Expected user ID 'user_789', got %s", err.UserID)
	}
}

func TestAppErrorIsRetryable(t *testing.T) {
	retryableErr := NewAppError(ErrCodeTimeout, "Timeout", nil)
	nonRetryableErr := NewAppError(ErrCodeInvalidInput, "Invalid input", nil)
	
	if !retryableErr.IsRetryable() {
		t.Error("Timeout error should be retryable")
	}
	
	if nonRetryableErr.IsRetryable() {
		t.Error("Invalid input error should not be retryable")
	}
}

func TestWrapError(t *testing.T) {
	originalErr := fmt.Errorf("original error")
	wrappedErr := WrapError(originalErr, ErrCodeDBQuery, "Database error")
	
	if wrappedErr.Code != ErrCodeDBQuery {
		t.Errorf("Expected code %s, got %s", ErrCodeDBQuery, wrappedErr.Code)
	}
	
	if wrappedErr.Message != "Database error" {
		t.Errorf("Expected message 'Database error', got %s", wrappedErr.Message)
	}
	
	if wrappedErr.Cause != originalErr {
		t.Error("Wrapped error should preserve original error")
	}
}

func TestErrorResponse(t *testing.T) {
	err := NewAppError(ErrCodeNotFound, "Resource not found", nil)
	response := NewErrorResponse(err, "/api/v1/test")
	
	if response.Error != err {
		t.Error("Response should contain the error")
	}
	
	if response.Success {
		t.Error("Response success should be false")
	}
	
	if response.Path != "/api/v1/test" {
		t.Errorf("Expected path '/api/v1/test', got %s", response.Path)
	}
	
	if time.Since(response.Timestamp) > time.Second {
		t.Error("Response timestamp should be recent")
	}
}

func TestGetSeverityByCode(t *testing.T) {
	tests := []struct {
		code             ErrorCode
		expectedSeverity ErrorSeverity
	}{
		{ErrCodeInternal, SeverityCritical},
		{ErrCodeDBConnection, SeverityCritical},
		{ErrCodeStrategyExecution, SeverityHigh},
		{ErrCodeCacheOperation, SeverityMedium},
		{ErrCodeInvalidInput, SeverityLow},
	}
	
	for _, test := range tests {
		severity := getSeverityByCode(test.code)
		if severity != test.expectedSeverity {
			t.Errorf("Code %s: expected severity %s, got %s", test.code, test.expectedSeverity, severity)
		}
	}
}

func TestIsAppError(t *testing.T) {
	appErr := NewAppError(ErrCodeInternal, "Test", nil)
	standardErr := fmt.Errorf("standard error")
	
	if !IsAppError(appErr) {
		t.Error("Should recognize AppError")
	}
	
	if IsAppError(standardErr) {
		t.Error("Should not recognize standard error as AppError")
	}
}

func TestGetAppError(t *testing.T) {
	appErr := NewAppError(ErrCodeInternal, "Test", nil)
	standardErr := fmt.Errorf("standard error")
	
	retrieved := GetAppError(appErr)
	if retrieved != appErr {
		t.Error("Should return the same AppError")
	}
	
	retrieved = GetAppError(standardErr)
	if retrieved != nil {
		t.Error("Should return nil for standard error")
	}
}