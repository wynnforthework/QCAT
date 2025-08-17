package security

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Vault provides secure storage for API keys
type Vault struct {
	storage   VaultStorage
	encryptor *Encryptor
	config    *VaultConfig
	mu        sync.RWMutex
}

// VaultStorage defines the interface for vault storage backends
type VaultStorage interface {
	Store(key string, data []byte) error
	Retrieve(key string) ([]byte, error)
	Delete(key string) error
	List(prefix string) ([]string, error)
	Exists(key string) (bool, error)
}

// VaultConfig represents vault configuration
type VaultConfig struct {
	StorageType    string        `json:"storage_type"`    // "memory", "file", "database"
	StoragePath    string        `json:"storage_path"`    // Path for file storage
	EncryptionKey  string        `json:"encryption_key"`  // Key for encryption
	BackupEnabled  bool          `json:"backup_enabled"`
	BackupInterval time.Duration `json:"backup_interval"`
	MaxBackups     int           `json:"max_backups"`
}

// NewVault creates a new vault
func NewVault(config *VaultConfig) (*Vault, error) {
	if config == nil {
		config = DefaultVaultConfig()
	}

	// Create storage backend
	var storage VaultStorage
	var err error

	switch config.StorageType {
	case "memory":
		storage = NewMemoryVaultStorage()
	case "file":
		storage, err = NewFileVaultStorage(config.StoragePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create file storage: %w", err)
		}
	case "database":
		// TODO: Implement database storage
		return nil, fmt.Errorf("database storage not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", config.StorageType)
	}

	// Create encryptor
	encryptor, err := NewEncryptor(config.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}

	vault := &Vault{
		storage:   storage,
		encryptor: encryptor,
		config:    config,
	}

	// Start backup routine if enabled
	if config.BackupEnabled {
		go vault.startBackupRoutine()
	}

	return vault, nil
}

// StoreKey stores a key in the vault
func (v *Vault) StoreKey(keyID string, keyInfo *KeyInfo) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Serialize key info
	data, err := json.Marshal(keyInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal key info: %w", err)
	}

	// Encrypt data
	encryptedData, err := v.encryptor.Encrypt(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt key data: %w", err)
	}

	// Store in backend
	storageKey := v.getStorageKey(keyID)
	if err := v.storage.Store(storageKey, encryptedData); err != nil {
		return fmt.Errorf("failed to store key: %w", err)
	}

	return nil
}

// GetKey retrieves a key from the vault
func (v *Vault) GetKey(keyID string) (*KeyInfo, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	storageKey := v.getStorageKey(keyID)
	
	// Retrieve from backend
	encryptedData, err := v.storage.Retrieve(storageKey)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve key: %w", err)
	}

	// Decrypt data
	data, err := v.encryptor.Decrypt(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt key data: %w", err)
	}

	// Deserialize key info
	var keyInfo KeyInfo
	if err := json.Unmarshal(data, &keyInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal key info: %w", err)
	}

	return &keyInfo, nil
}

// UpdateKey updates a key in the vault
func (v *Vault) UpdateKey(keyID string, keyInfo *KeyInfo) error {
	return v.StoreKey(keyID, keyInfo)
}

// DeleteKey deletes a key from the vault
func (v *Vault) DeleteKey(keyID string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	storageKey := v.getStorageKey(keyID)
	return v.storage.Delete(storageKey)
}

// FindKeyByHash finds a key by its hash
func (v *Vault) FindKeyByHash(keyHash string) (*KeyInfo, error) {
	keys, err := v.ListKeys(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	for _, keyInfo := range keys {
		if keyInfo.KeyHash == keyHash {
			return keyInfo, nil
		}
	}

	return nil, fmt.Errorf("key not found")
}

// ListKeys lists all keys with optional filtering
func (v *Vault) ListKeys(filter *KeyFilter) ([]*KeyInfo, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// List all storage keys
	storageKeys, err := v.storage.List("key:")
	if err != nil {
		return nil, fmt.Errorf("failed to list storage keys: %w", err)
	}

	var keys []*KeyInfo
	for _, storageKey := range storageKeys {
		// Extract key ID from storage key
		keyID := v.extractKeyID(storageKey)
		
		// Get key info
		keyInfo, err := v.GetKey(keyID)
		if err != nil {
			continue // Skip invalid keys
		}

		// Apply filter
		if filter != nil && !v.matchesFilter(keyInfo, filter) {
			continue
		}

		keys = append(keys, keyInfo)
	}

	return keys, nil
}

// KeyExists checks if a key exists in the vault
func (v *Vault) KeyExists(keyID string) (bool, error) {
	storageKey := v.getStorageKey(keyID)
	return v.storage.Exists(storageKey)
}

// Backup creates a backup of all keys
func (v *Vault) Backup() ([]byte, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	keys, err := v.ListKeys(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list keys for backup: %w", err)
	}

	backup := VaultBackup{
		Timestamp: time.Now(),
		Keys:      keys,
		Version:   "1.0",
	}

	data, err := json.Marshal(backup)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal backup: %w", err)
	}

	// Encrypt backup
	return v.encryptor.Encrypt(data)
}

// Restore restores keys from a backup
func (v *Vault) Restore(backupData []byte) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Decrypt backup
	data, err := v.encryptor.Decrypt(backupData)
	if err != nil {
		return fmt.Errorf("failed to decrypt backup: %w", err)
	}

	// Unmarshal backup
	var backup VaultBackup
	if err := json.Unmarshal(data, &backup); err != nil {
		return fmt.Errorf("failed to unmarshal backup: %w", err)
	}

	// Restore keys
	for _, keyInfo := range backup.Keys {
		if err := v.StoreKey(keyInfo.ID, keyInfo); err != nil {
			return fmt.Errorf("failed to restore key %s: %w", keyInfo.ID, err)
		}
	}

	return nil
}

// startBackupRoutine starts the automatic backup routine
func (v *Vault) startBackupRoutine() {
	ticker := time.NewTicker(v.config.BackupInterval)
	defer ticker.Stop()

	for range ticker.C {
		v.performBackup()
	}
}

// performBackup performs a backup operation
func (v *Vault) performBackup() {
	backupData, err := v.Backup()
	if err != nil {
		// Log error but don't fail
		fmt.Printf("Backup failed: %v\n", err)
		return
	}

	// Store backup (implementation depends on storage backend)
	backupKey := fmt.Sprintf("backup:%d", time.Now().Unix())
	if err := v.storage.Store(backupKey, backupData); err != nil {
		fmt.Printf("Failed to store backup: %v\n", err)
		return
	}

	// Clean up old backups
	v.cleanupOldBackups()
}

// cleanupOldBackups removes old backup files
func (v *Vault) cleanupOldBackups() {
	backupKeys, err := v.storage.List("backup:")
	if err != nil {
		return
	}

	if len(backupKeys) <= v.config.MaxBackups {
		return
	}

	// Remove oldest backups
	excess := len(backupKeys) - v.config.MaxBackups
	for i := 0; i < excess; i++ {
		v.storage.Delete(backupKeys[i])
	}
}

// Helper methods

func (v *Vault) getStorageKey(keyID string) string {
	return fmt.Sprintf("key:%s", keyID)
}

func (v *Vault) extractKeyID(storageKey string) string {
	if len(storageKey) > 4 && storageKey[:4] == "key:" {
		return storageKey[4:]
	}
	return storageKey
}

func (v *Vault) matchesFilter(keyInfo *KeyInfo, filter *KeyFilter) bool {
	if filter.Status != "" && keyInfo.Status != filter.Status {
		return false
	}

	if filter.Name != "" && keyInfo.Name != filter.Name {
		return false
	}

	if filter.Permission != "" {
		hasPermission := false
		for _, perm := range keyInfo.Permissions {
			if perm == filter.Permission {
				hasPermission = true
				break
			}
		}
		if !hasPermission {
			return false
		}
	}

	if !filter.CreatedAfter.IsZero() && keyInfo.CreatedAt.Before(filter.CreatedAfter) {
		return false
	}

	if !filter.CreatedBefore.IsZero() && keyInfo.CreatedAt.After(filter.CreatedBefore) {
		return false
	}

	return true
}

// DefaultVaultConfig returns default vault configuration
func DefaultVaultConfig() *VaultConfig {
	return &VaultConfig{
		StorageType:    "memory",
		StoragePath:    "./vault",
		EncryptionKey:  generateDefaultEncryptionKey(),
		BackupEnabled:  true,
		BackupInterval: 24 * time.Hour,
		MaxBackups:     7,
	}
}

// VaultBackup represents a vault backup
type VaultBackup struct {
	Timestamp time.Time  `json:"timestamp"`
	Keys      []*KeyInfo `json:"keys"`
	Version   string     `json:"version"`
}