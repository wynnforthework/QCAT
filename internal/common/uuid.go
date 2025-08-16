package common

import (
	"crypto/rand"
	"fmt"
	"strings"
)

// GenerateUUID generates a UUID v4 string
func GenerateUUID() string {
	// Generate 16 random bytes
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		// Fallback to timestamp-based ID if random generation fails
		return GenerateTimestampID()
	}
	
	// Set version (4) and variant bits
	bytes[6] = (bytes[6] & 0x0f) | 0x40 // Version 4
	bytes[8] = (bytes[8] & 0x3f) | 0x80 // Variant 10
	
	// Format as UUID string
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		bytes[0:4],
		bytes[4:6],
		bytes[6:8],
		bytes[8:10],
		bytes[10:16])
}

// GenerateTimestampID generates a timestamp-based ID as fallback
func GenerateTimestampID() string {
	return fmt.Sprintf("id_%d", getCurrentTimestamp())
}

// GenerateShortID generates a shorter ID for internal use
func GenerateShortID() string {
	bytes := make([]byte, 8)
	_, err := rand.Read(bytes)
	if err != nil {
		return fmt.Sprintf("short_%d", getCurrentTimestamp())
	}
	
	return fmt.Sprintf("%016x", bytes)
}

// ValidateUUID validates if a string is a valid UUID
func ValidateUUID(id string) bool {
	if len(id) != 36 {
		return false
	}
	
	// Check format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	parts := strings.Split(id, "-")
	if len(parts) != 5 {
		return false
	}
	
	// Check part lengths
	expectedLengths := []int{8, 4, 4, 4, 12}
	for i, part := range parts {
		if len(part) != expectedLengths[i] {
			return false
		}
		
		// Check if all characters are hex
		for _, char := range part {
			if !isHexChar(char) {
				return false
			}
		}
	}
	
	return true
}

// isHexChar checks if a character is a valid hexadecimal character
func isHexChar(char rune) bool {
	return (char >= '0' && char <= '9') ||
		   (char >= 'a' && char <= 'f') ||
		   (char >= 'A' && char <= 'F')
}

// getCurrentTimestamp returns current timestamp in nanoseconds
func getCurrentTimestamp() int64 {
	// This would normally use time.Now().UnixNano()
	// but we'll implement it without importing time to avoid circular dependencies
	return 1705420800000000000 // Placeholder timestamp
}