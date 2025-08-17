package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// KeyManager manages API keys with rotation and monitoring
type KeyManager struct {
	vault     *Vault
	rotator   *KeyRotator
	monitor   *KeyMonitor
	encryptor *Encryptor
	mu        sync.RWMutex
}

// KeyInfo represents information about an API key
type KeyInfo struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	KeyHash     string    `json:"key_hash"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	LastUsed    time.Time `json:"last_used"`
	UsageCount  int64     `json:"usage_count"`
	Status      KeyStatus `json:"status"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// KeyStatus represents the status of an API key
type KeyStatus string

const (
	KeyStatusActive    KeyStatus = "active"
	KeyStatusInactive  KeyStatus = "inactive"
	KeyStatusExpired   KeyStatus = "expired"
	KeyStatusRevoked   KeyStatus = "revoked"
	KeyStatusRotating  KeyStatus = "rotating"
)

// KeyPermission represents different permission levels
type KeyPermission string

const (
	PermissionRead      KeyPermission = "read"
	PermissionWrite     KeyPermission = "write"
	PermissionTrade     KeyPermission = "trade"
	PermissionAdmin     KeyPermission = "admin"
	PermissionWithdraw  KeyPermission = "withdraw"
)

// NewKeyManager creates a new key manager
func NewKeyManager(config *KeyManagerConfig) (*KeyManager, error) {
	if config == nil {
		config = DefaultKeyManagerConfig()
	}

	vault, err := NewVault(config.VaultConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault: %w", err)
	}

	encryptor, err := NewEncryptor(config.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}

	monitor := NewKeyMonitor(config.MonitorConfig)
	rotator := NewKeyRotator(config.RotationConfig)

	km := &KeyManager{
		vault:     vault,
		rotator:   rotator,
		monitor:   monitor,
		encryptor: encryptor,
	}

	// Start automatic rotation if enabled
	if config.RotationConfig.AutoRotate {
		go km.startAutoRotation()
	}

	return km, nil
}

// GenerateKey generates a new API key
func (km *KeyManager) GenerateKey(name string, permissions []KeyPermission, expiresAt time.Time) (*KeyInfo, string, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	// Generate random key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, "", fmt.Errorf("failed to generate random key: %w", err)
	}

	keyString := base64.URLEncoding.EncodeToString(keyBytes)
	keyHash := km.hashKey(keyString)

	// Create key info
	keyInfo := &KeyInfo{
		ID:          generateKeyID(),
		Name:        name,
		KeyHash:     keyHash,
		Permissions: permissionsToStrings(permissions),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
		Status:      KeyStatusActive,
		Metadata:    make(map[string]interface{}),
	}

	// Store in vault
	if err := km.vault.StoreKey(keyInfo.ID, keyInfo); err != nil {
		return nil, "", fmt.Errorf("failed to store key: %w", err)
	}

	// Log key creation
	km.monitor.LogKeyEvent(KeyEvent{
		Type:      "key_created",
		KeyID:     keyInfo.ID,
		Timestamp: time.Now(),
		Details:   map[string]interface{}{"name": name, "permissions": permissions},
	})

	return keyInfo, keyString, nil
}

// ValidateKey validates an API key and returns key info
func (km *KeyManager) ValidateKey(keyString string) (*KeyInfo, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	keyHash := km.hashKey(keyString)

	// Find key by hash
	keyInfo, err := km.vault.FindKeyByHash(keyHash)
	if err != nil {
		km.monitor.LogKeyEvent(KeyEvent{
			Type:      "key_validation_failed",
			KeyID:     "unknown",
			Timestamp: time.Now(),
			Details:   map[string]interface{}{"error": err.Error()},
		})
		return nil, fmt.Errorf("key validation failed: %w", err)
	}

	// Check if key is active
	if keyInfo.Status != KeyStatusActive {
		km.monitor.LogKeyEvent(KeyEvent{
			Type:      "key_inactive",
			KeyID:     keyInfo.ID,
			Timestamp: time.Now(),
			Details:   map[string]interface{}{"status": keyInfo.Status},
		})
		return nil, fmt.Errorf("key is not active: %s", keyInfo.Status)
	}

	// Check if key is expired
	if time.Now().After(keyInfo.ExpiresAt) {
		// Mark as expired
		keyInfo.Status = KeyStatusExpired
		km.vault.UpdateKey(keyInfo.ID, keyInfo)
		
		km.monitor.LogKeyEvent(KeyEvent{
			Type:      "key_expired",
			KeyID:     keyInfo.ID,
			Timestamp: time.Now(),
		})
		return nil, fmt.Errorf("key has expired")
	}

	// Update usage statistics
	keyInfo.LastUsed = time.Now()
	keyInfo.UsageCount++
	km.vault.UpdateKey(keyInfo.ID, keyInfo)

	// Log successful validation
	km.monitor.LogKeyEvent(KeyEvent{
		Type:      "key_validated",
		KeyID:     keyInfo.ID,
		Timestamp: time.Now(),
	})

	return keyInfo, nil
}

// RotateKey rotates an API key
func (km *KeyManager) RotateKey(keyID string) (*KeyInfo, string, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	// Get existing key
	oldKeyInfo, err := km.vault.GetKey(keyID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get key: %w", err)
	}

	// Mark as rotating
	oldKeyInfo.Status = KeyStatusRotating
	km.vault.UpdateKey(keyID, oldKeyInfo)

	// Generate new key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, "", fmt.Errorf("failed to generate new key: %w", err)
	}

	newKeyString := base64.URLEncoding.EncodeToString(keyBytes)
	newKeyHash := km.hashKey(newKeyString)

	// Update key info
	newKeyInfo := *oldKeyInfo
	newKeyInfo.KeyHash = newKeyHash
	newKeyInfo.UpdatedAt = time.Now()
	newKeyInfo.Status = KeyStatusActive
	newKeyInfo.UsageCount = 0

	// Store updated key
	if err := km.vault.UpdateKey(keyID, &newKeyInfo); err != nil {
		return nil, "", fmt.Errorf("failed to update key: %w", err)
	}

	// Log rotation
	km.monitor.LogKeyRotation(keyID, time.Now())

	return &newKeyInfo, newKeyString, nil
}

// RevokeKey revokes an API key
func (km *KeyManager) RevokeKey(keyID string, reason string) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	keyInfo, err := km.vault.GetKey(keyID)
	if err != nil {
		return fmt.Errorf("failed to get key: %w", err)
	}

	keyInfo.Status = KeyStatusRevoked
	keyInfo.UpdatedAt = time.Now()
	keyInfo.Metadata["revocation_reason"] = reason
	keyInfo.Metadata["revoked_at"] = time.Now()

	if err := km.vault.UpdateKey(keyID, keyInfo); err != nil {
		return fmt.Errorf("failed to update key: %w", err)
	}

	// Log revocation
	km.monitor.LogKeyEvent(KeyEvent{
		Type:      "key_revoked",
		KeyID:     keyID,
		Timestamp: time.Now(),
		Details:   map[string]interface{}{"reason": reason},
	})

	return nil
}

// ListKeys returns all keys with optional filtering
func (km *KeyManager) ListKeys(filter *KeyFilter) ([]*KeyInfo, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	return km.vault.ListKeys(filter)
}

// GetKeyUsageStats returns usage statistics for a key
func (km *KeyManager) GetKeyUsageStats(keyID string, period time.Duration) (*KeyUsageStats, error) {
	return km.monitor.GetKeyUsageStats(keyID, period)
}

// CheckPermission checks if a key has a specific permission
func (km *KeyManager) CheckPermission(keyInfo *KeyInfo, permission KeyPermission) bool {
	for _, perm := range keyInfo.Permissions {
		if perm == string(permission) || perm == string(PermissionAdmin) {
			return true
		}
	}
	return false
}

// startAutoRotation starts automatic key rotation
func (km *KeyManager) startAutoRotation() {
	ticker := time.NewTicker(km.rotator.config.CheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		km.performAutoRotation()
	}
}

// performAutoRotation performs automatic rotation for eligible keys
func (km *KeyManager) performAutoRotation() {
	keys, err := km.vault.ListKeys(&KeyFilter{Status: KeyStatusActive})
	if err != nil {
		km.monitor.LogError("auto_rotation_list_failed", err)
		return
	}

	for _, keyInfo := range keys {
		if km.shouldRotateKey(keyInfo) {
			_, _, err := km.RotateKey(keyInfo.ID)
			if err != nil {
				km.monitor.LogError("auto_rotation_failed", err)
			}
		}
	}
}

// shouldRotateKey determines if a key should be rotated
func (km *KeyManager) shouldRotateKey(keyInfo *KeyInfo) bool {
	// Check if key is old enough for rotation
	age := time.Since(keyInfo.UpdatedAt)
	if age < km.rotator.config.MinRotationInterval {
		return false
	}

	// Check if key is approaching expiration
	timeToExpiry := time.Until(keyInfo.ExpiresAt)
	if timeToExpiry < km.rotator.config.RotateBeforeExpiry {
		return true
	}

	// Check if key has been used too much
	if keyInfo.UsageCount > km.rotator.config.MaxUsageBeforeRotation {
		return true
	}

	// Check if key is older than max age
	if age > km.rotator.config.MaxKeyAge {
		return true
	}

	return false
}

// hashKey creates a hash of the key for storage
func (km *KeyManager) hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// Helper functions

func generateKeyID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func permissionsToStrings(permissions []KeyPermission) []string {
	result := make([]string, len(permissions))
	for i, perm := range permissions {
		result[i] = string(perm)
	}
	return result
}

// KeyManagerConfig represents key manager configuration
type KeyManagerConfig struct {
	VaultConfig     *VaultConfig     `json:"vault_config"`
	RotationConfig  *RotationConfig  `json:"rotation_config"`
	MonitorConfig   *MonitorConfig   `json:"monitor_config"`
	EncryptionKey   string           `json:"encryption_key"`
}

// DefaultKeyManagerConfig returns default configuration
func DefaultKeyManagerConfig() *KeyManagerConfig {
	return &KeyManagerConfig{
		VaultConfig:    DefaultVaultConfig(),
		RotationConfig: DefaultRotationConfig(),
		MonitorConfig:  DefaultMonitorConfig(),
		EncryptionKey:  generateDefaultEncryptionKey(),
	}
}

func generateDefaultEncryptionKey() string {
	key := make([]byte, 32)
	rand.Read(key)
	return base64.StdEncoding.EncodeToString(key)
}

// KeyFilter represents filtering options for listing keys
type KeyFilter struct {
	Status      KeyStatus `json:"status,omitempty"`
	Name        string    `json:"name,omitempty"`
	Permission  string    `json:"permission,omitempty"`
	CreatedAfter time.Time `json:"created_after,omitempty"`
	CreatedBefore time.Time `json:"created_before,omitempty"`
}

// KeyEvent represents a key-related event
type KeyEvent struct {
	Type      string                 `json:"type"`
	KeyID     string                 `json:"key_id"`
	Timestamp time.Time              `json:"timestamp"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// KeyUsageStats represents usage statistics for a key
type KeyUsageStats struct {
	KeyID        string    `json:"key_id"`
	TotalUsage   int64     `json:"total_usage"`
	PeriodUsage  int64     `json:"period_usage"`
	LastUsed     time.Time `json:"last_used"`
	AverageDaily float64   `json:"average_daily"`
	PeakDaily    int64     `json:"peak_daily"`
}