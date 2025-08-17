package errors

import (
	"testing"
)

func BenchmarkNewAppError(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewAppError(ErrCodeInvalidInput, "test error", nil)
	}
}

func BenchmarkAppErrorWithContext(b *testing.B) {
	err := NewAppError(ErrCodeInvalidInput, "test error", nil)
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = err.WithContext("key", "value")
	}
}

func BenchmarkWrapError(b *testing.B) {
	originalErr := NewAppError(ErrCodeInternal, "original", nil)
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = WrapError(originalErr, ErrCodeDBQuery, "wrapped error")
	}
}

func BenchmarkHTTPStatus(b *testing.B) {
	err := NewAppError(ErrCodeInvalidInput, "test error", nil)
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = err.HTTPStatus()
	}
}

func BenchmarkIsRetryable(b *testing.B) {
	err := NewAppError(ErrCodeTimeout, "timeout error", nil)
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = err.IsRetryable()
	}
}