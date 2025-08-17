package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

// Encryptor provides encryption and decryption functionality
type Encryptor struct {
	gcm cipher.AEAD
}

// NewEncryptor creates a new encryptor with the given key
func NewEncryptor(key string) (*Encryptor, error) {
	// Derive a 32-byte key from the input
	keyBytes := sha256.Sum256([]byte(key))
	
	// Create AES cipher
	block, err := aes.NewCipher(keyBytes[:])
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}
	
	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	
	return &Encryptor{gcm: gcm}, nil
}

// Encrypt encrypts the given data
func (e *Encryptor) Encrypt(data []byte) ([]byte, error) {
	// Generate a random nonce
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	
	// Encrypt the data
	ciphertext := e.gcm.Seal(nonce, nonce, data, nil)
	
	return ciphertext, nil
}

// Decrypt decrypts the given data
func (e *Encryptor) Decrypt(data []byte) ([]byte, error) {
	nonceSize := e.gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	
	// Extract nonce and ciphertext
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	
	// Decrypt the data
	plaintext, err := e.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}
	
	return plaintext, nil
}

// EncryptString encrypts a string and returns base64 encoded result
func (e *Encryptor) EncryptString(text string) (string, error) {
	encrypted, err := e.Encrypt([]byte(text))
	if err != nil {
		return "", err
	}
	
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// DecryptString decrypts a base64 encoded string
func (e *Encryptor) DecryptString(encodedText string) (string, error) {
	encrypted, err := base64.StdEncoding.DecodeString(encodedText)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}
	
	decrypted, err := e.Decrypt(encrypted)
	if err != nil {
		return "", err
	}
	
	return string(decrypted), nil
}

// GenerateKey generates a new random encryption key
func GenerateKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("failed to generate key: %w", err)
	}
	
	return base64.StdEncoding.EncodeToString(key), nil
}

// HashPassword creates a secure hash of a password
func HashPassword(password string) (string, error) {
	// Use SHA-256 for password hashing (in production, use bcrypt or similar)
	hash := sha256.Sum256([]byte(password))
	return base64.StdEncoding.EncodeToString(hash[:]), nil
}

// VerifyPassword verifies a password against its hash
func VerifyPassword(password, hash string) bool {
	passwordHash, err := HashPassword(password)
	if err != nil {
		return false
	}
	
	return passwordHash == hash
}