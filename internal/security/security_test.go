package security

import (
	"testing"
	"time"
)

func TestKeyManager(t *testing.T) {
	// Create key manager with default config
	config := DefaultKeyManagerConfig()
	km, err := NewKeyManager(config)
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	// Test key generation
	permissions := []KeyPermission{PermissionRead, PermissionWrite}
	expiresAt := time.Now().Add(24 * time.Hour)
	
	keyInfo, keyString, err := km.GenerateKey("test-key", permissions, expiresAt)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	if keyInfo.Name != "test-key" {
		t.Errorf("Expected key name 'test-key', got '%s'", keyInfo.Name)
	}

	if len(keyInfo.Permissions) != 2 {
		t.Errorf("Expected 2 permissions, got %d", len(keyInfo.Permissions))
	}

	if keyString == "" {
		t.Error("Expected non-empty key string")
	}

	// Test key validation
	validatedKeyInfo, err := km.ValidateKey(keyString)
	if err != nil {
		t.Fatalf("Failed to validate key: %v", err)
	}

	if validatedKeyInfo.ID != keyInfo.ID {
		t.Errorf("Expected key ID '%s', got '%s'", keyInfo.ID, validatedKeyInfo.ID)
	}

	// Test key rotation
	newKeyInfo, newKeyString, err := km.RotateKey(keyInfo.ID)
	if err != nil {
		t.Fatalf("Failed to rotate key: %v", err)
	}

	if newKeyString == keyString {
		t.Error("Expected new key string to be different from original")
	}

	if newKeyInfo.KeyHash == keyInfo.KeyHash {
		t.Error("Expected new key hash to be different from original")
	}

	// Test key revocation
	err = km.RevokeKey(keyInfo.ID, "test revocation")
	if err != nil {
		t.Fatalf("Failed to revoke key: %v", err)
	}

	// Validate revoked key should fail
	_, err = km.ValidateKey(newKeyString)
	if err == nil {
		t.Error("Expected validation of revoked key to fail")
	}
}

func TestVault(t *testing.T) {
	// Create vault with memory storage
	config := DefaultVaultConfig()
	vault, err := NewVault(config)
	if err != nil {
		t.Fatalf("Failed to create vault: %v", err)
	}

	// Test key storage and retrieval
	keyInfo := &KeyInfo{
		ID:          "test-key-id",
		Name:        "test-key",
		KeyHash:     "test-hash",
		Permissions: []string{"read", "write"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		Status:      KeyStatusActive,
		Metadata:    make(map[string]interface{}),
	}

	// Store key
	err = vault.StoreKey(keyInfo.ID, keyInfo)
	if err != nil {
		t.Fatalf("Failed to store key: %v", err)
	}

	// Retrieve key
	retrievedKey, err := vault.GetKey(keyInfo.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve key: %v", err)
	}

	if retrievedKey.ID != keyInfo.ID {
		t.Errorf("Expected key ID '%s', got '%s'", keyInfo.ID, retrievedKey.ID)
	}

	if retrievedKey.Name != keyInfo.Name {
		t.Errorf("Expected key name '%s', got '%s'", keyInfo.Name, retrievedKey.Name)
	}

	// Test key existence
	exists, err := vault.KeyExists(keyInfo.ID)
	if err != nil {
		t.Fatalf("Failed to check key existence: %v", err)
	}

	if !exists {
		t.Error("Expected key to exist")
	}

	// Test key listing
	keys, err := vault.ListKeys(nil)
	if err != nil {
		t.Fatalf("Failed to list keys: %v", err)
	}

	if len(keys) != 1 {
		t.Errorf("Expected 1 key, got %d", len(keys))
	}

	// Test key deletion
	err = vault.DeleteKey(keyInfo.ID)
	if err != nil {
		t.Fatalf("Failed to delete key: %v", err)
	}

	exists, err = vault.KeyExists(keyInfo.ID)
	if err != nil {
		t.Fatalf("Failed to check key existence after deletion: %v", err)
	}

	if exists {
		t.Error("Expected key to not exist after deletion")
	}
}

func TestEncryptor(t *testing.T) {
	// Create encryptor
	encryptor, err := NewEncryptor("test-encryption-key")
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	// Test encryption and decryption
	plaintext := "This is a test message"
	
	encrypted, err := encryptor.Encrypt([]byte(plaintext))
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	if len(encrypted) == 0 {
		t.Error("Expected non-empty encrypted data")
	}

	decrypted, err := encryptor.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	if string(decrypted) != plaintext {
		t.Errorf("Expected decrypted text '%s', got '%s'", plaintext, string(decrypted))
	}

	// Test string encryption
	encryptedString, err := encryptor.EncryptString(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt string: %v", err)
	}

	decryptedString, err := encryptor.DecryptString(encryptedString)
	if err != nil {
		t.Fatalf("Failed to decrypt string: %v", err)
	}

	if decryptedString != plaintext {
		t.Errorf("Expected decrypted string '%s', got '%s'", plaintext, decryptedString)
	}
}

func TestAuditLogger(t *testing.T) {
	// Create audit logger
	config := DefaultAuditConfig()
	config.StorageType = "memory"
	
	logger, err := NewAuditLogger(config)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}

	// Test logging
	entry := &AuditEntry{
		UserID:   "test-user",
		Action:   "test-action",
		Resource: "test-resource",
		Success:  true,
		Details: map[string]interface{}{
			"test": "data",
		},
	}

	err = logger.Log(entry)
	if err != nil {
		t.Fatalf("Failed to log entry: %v", err)
	}

	// Test retrieval
	entries, err := logger.GetEntries(nil)
	if err != nil {
		t.Fatalf("Failed to get entries: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}

	if entries[0].UserID != "test-user" {
		t.Errorf("Expected user ID 'test-user', got '%s'", entries[0].UserID)
	}

	// Test filtering
	filter := &AuditFilter{
		UserID: "test-user",
		Action: "test-action",
	}

	filteredEntries, err := logger.GetEntries(filter)
	if err != nil {
		t.Fatalf("Failed to get filtered entries: %v", err)
	}

	if len(filteredEntries) != 1 {
		t.Errorf("Expected 1 filtered entry, got %d", len(filteredEntries))
	}

	// Test integrity verification
	if config.EnableIntegrityCheck {
		report, err := logger.VerifyIntegrity()
		if err != nil {
			t.Fatalf("Failed to verify integrity: %v", err)
		}

		if !report.IntegrityValid {
			t.Error("Expected integrity to be valid")
		}

		if report.TotalEntries != 1 {
			t.Errorf("Expected 1 total entry, got %d", report.TotalEntries)
		}

		if report.VerifiedEntries != 1 {
			t.Errorf("Expected 1 verified entry, got %d", report.VerifiedEntries)
		}
	}
}

func TestKeyMonitor(t *testing.T) {
	// Create key monitor
	config := DefaultMonitorConfig()
	monitor := NewKeyMonitor(config)

	// Test event logging
	event := KeyEvent{
		Type:      "key_validated",
		KeyID:     "test-key",
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"test": "data",
		},
	}

	monitor.LogKeyEvent(event)

	// Test event retrieval
	events := monitor.GetRecentEvents(10)
	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}

	if events[0].Type != "key_validated" {
		t.Errorf("Expected event type 'key_validated', got '%s'", events[0].Type)
	}

	// Test usage stats
	stats, err := monitor.GetKeyUsageStats("test-key", time.Hour)
	if err != nil {
		t.Fatalf("Failed to get usage stats: %v", err)
	}

	if stats.KeyID != "test-key" {
		t.Errorf("Expected key ID 'test-key', got '%s'", stats.KeyID)
	}
}

func TestKeyRotator(t *testing.T) {
	// Create key rotator
	config := DefaultRotationConfig()
	rotator := NewKeyRotator(config)

	// Test rotation decision
	keyInfo := &KeyInfo{
		ID:         "test-key",
		CreatedAt:  time.Now().Add(-25 * time.Hour), // Older than max age
		UpdatedAt:  time.Now().Add(-25 * time.Hour),
		ExpiresAt:  time.Now().Add(24 * time.Hour),
		UsageCount: 50000, // Below usage threshold
	}

	shouldRotate := rotator.ShouldRotate(keyInfo)
	if !shouldRotate {
		t.Error("Expected key to need rotation due to age")
	}

	// Test rotation schedule
	schedule := rotator.GetRotationSchedule(keyInfo)
	if schedule.KeyID != "test-key" {
		t.Errorf("Expected key ID 'test-key', got '%s'", schedule.KeyID)
	}

	if schedule.Reason == "" {
		t.Error("Expected rotation reason to be set")
	}
}