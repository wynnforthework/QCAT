package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/scrypt"
)

// EnvManager manages environment variable configuration
type EnvManager struct {
	encryptionKey []byte
	prefix        string
}

// NewEnvManager creates a new environment variable manager
func NewEnvManager(encryptionKey string, prefix string) *EnvManager {
	if encryptionKey == "" {
		encryptionKey = os.Getenv("QCAT_ENCRYPTION_KEY")
	}
	if prefix == "" {
		prefix = "QCAT_"
	}

	// Derive encryption key from password
	key, _ := scrypt.Key([]byte(encryptionKey), []byte("qcat-salt"), 32768, 8, 1, 32)

	return &EnvManager{
		encryptionKey: key,
		prefix:        prefix,
	}
}

// GetString gets a string environment variable
func (em *EnvManager) GetString(key string, defaultValue string) string {
	envKey := em.prefix + strings.ToUpper(key)
	value := os.Getenv(envKey)
	if value == "" {
		return defaultValue
	}
	return value
}

// GetInt gets an integer environment variable
func (em *EnvManager) GetInt(key string, defaultValue int) int {
	value := em.GetString(key, "")
	if value == "" {
		return defaultValue
	}
	if intValue, err := strconv.Atoi(value); err == nil {
		return intValue
	}
	return defaultValue
}

// GetBool gets a boolean environment variable
func (em *EnvManager) GetBool(key string, defaultValue bool) bool {
	value := em.GetString(key, "")
	if value == "" {
		return defaultValue
	}
	if boolValue, err := strconv.ParseBool(value); err == nil {
		return boolValue
	}
	return defaultValue
}

// GetDuration gets a duration environment variable
func (em *EnvManager) GetDuration(key string, defaultValue time.Duration) time.Duration {
	value := em.GetString(key, "")
	if value == "" {
		return defaultValue
	}
	if duration, err := time.ParseDuration(value); err == nil {
		return duration
	}
	return defaultValue
}

// GetEncryptedString gets an encrypted string environment variable
func (em *EnvManager) GetEncryptedString(key string, defaultValue string) string {
	value := em.GetString(key, "")
	if value == "" {
		return defaultValue
	}

	// Check if value is encrypted (starts with "ENC:")
	if !strings.HasPrefix(value, "ENC:") {
		return value
	}

	// Decrypt the value
	encryptedValue := strings.TrimPrefix(value, "ENC:")
	decryptedValue, err := em.decrypt(encryptedValue)
	if err != nil {
		fmt.Printf("Warning: Failed to decrypt %s: %v\n", key, err)
		return defaultValue
	}

	return decryptedValue
}

// SetEncryptedString sets an encrypted string environment variable
func (em *EnvManager) SetEncryptedString(key string, value string) error {
	if value == "" {
		return em.SetString(key, "")
	}

	// Encrypt the value
	encryptedValue, err := em.encrypt(value)
	if err != nil {
		return fmt.Errorf("failed to encrypt value: %w", err)
	}

	// Set the encrypted value with ENC: prefix
	return em.SetString(key, "ENC:"+encryptedValue)
}

// SetString sets a string environment variable
func (em *EnvManager) SetString(key string, value string) error {
	envKey := em.prefix + strings.ToUpper(key)
	return os.Setenv(envKey, value)
}

// SetInt sets an integer environment variable
func (em *EnvManager) SetInt(key string, value int) error {
	return em.SetString(key, strconv.Itoa(value))
}

// SetBool sets a boolean environment variable
func (em *EnvManager) SetBool(key string, value bool) error {
	return em.SetString(key, strconv.FormatBool(value))
}

// SetDuration sets a duration environment variable
func (em *EnvManager) SetDuration(key string, value time.Duration) error {
	return em.SetString(key, value.String())
}

// encrypt encrypts a string value
func (em *EnvManager) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(em.encryptionKey)
	if err != nil {
		return "", err
	}

	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], []byte(plaintext))

	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts an encrypted string value
func (em *EnvManager) decrypt(encryptedText string) (string, error) {
	ciphertext, err := base64.URLEncoding.DecodeString(encryptedText)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(em.encryptionKey)
	if err != nil {
		return "", err
	}

	if len(ciphertext) < aes.BlockSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	return string(ciphertext), nil
}

// LoadFromFile loads environment variables from a file
func (em *EnvManager) LoadFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if len(value) >= 2 && (value[0] == '"' && value[len(value)-1] == '"') {
			value = value[1 : len(value)-1]
		}

		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("failed to set environment variable %s: %w", key, err)
		}
	}

	return nil
}

// ExportToFile exports current environment variables to a file
func (em *EnvManager) ExportToFile(filename string, includeEncrypted bool) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get all environment variables with our prefix
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, em.prefix) {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				key := parts[0]
				value := parts[1]

				// Skip encrypted values unless explicitly requested
				if !includeEncrypted && strings.HasPrefix(value, "ENC:") {
					continue
				}

				// Write to file
				if _, err := file.WriteString(fmt.Sprintf("%s=\"%s\"\n", key, value)); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// ValidateRequired checks if all required environment variables are set
func (em *EnvManager) ValidateRequired(required []string) error {
	var missing []string

	for _, key := range required {
		envKey := em.prefix + strings.ToUpper(key)
		if os.Getenv(envKey) == "" {
			missing = append(missing, envKey)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %v", missing)
	}

	return nil
}
