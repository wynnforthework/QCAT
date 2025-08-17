package security

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// AuditLogger provides secure audit logging with integrity verification
type AuditLogger struct {
	storage    AuditStorage
	encryptor  *Encryptor
	config     *AuditConfig
	chain      *AuditChain
	mu         sync.RWMutex
}

// AuditStorage defines the interface for audit log storage
type AuditStorage interface {
	Store(entry *AuditEntry) error
	Retrieve(id string) (*AuditEntry, error)
	List(filter *AuditFilter) ([]*AuditEntry, error)
	Delete(id string) error
}

// AuditConfig represents audit configuration
type AuditConfig struct {
	EnableEncryption    bool          `json:"enable_encryption"`
	EnableIntegrityCheck bool         `json:"enable_integrity_check"`
	RetentionPeriod     time.Duration `json:"retention_period"`
	MaxEntries          int           `json:"max_entries"`
	StorageType         string        `json:"storage_type"`
	StoragePath         string        `json:"storage_path"`
	AlertOnTampering    bool          `json:"alert_on_tampering"`
}

// AuditEntry represents a single audit log entry
type AuditEntry struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	UserID      string                 `json:"user_id"`
	Action      string                 `json:"action"`
	Resource    string                 `json:"resource"`
	Details     map[string]interface{} `json:"details"`
	IPAddress   string                 `json:"ip_address"`
	UserAgent   string                 `json:"user_agent"`
	Success     bool                   `json:"success"`
	ErrorMsg    string                 `json:"error_msg,omitempty"`
	Hash        string                 `json:"hash"`
	PrevHash    string                 `json:"prev_hash"`
	Signature   string                 `json:"signature,omitempty"`
}

// AuditChain maintains the integrity chain of audit logs
type AuditChain struct {
	lastHash string
	mu       sync.RWMutex
}

// AuditFilter represents filtering options for audit logs
type AuditFilter struct {
	UserID      string    `json:"user_id,omitempty"`
	Action      string    `json:"action,omitempty"`
	Resource    string    `json:"resource,omitempty"`
	StartTime   time.Time `json:"start_time,omitempty"`
	EndTime     time.Time `json:"end_time,omitempty"`
	Success     *bool     `json:"success,omitempty"`
	IPAddress   string    `json:"ip_address,omitempty"`
	Limit       int       `json:"limit,omitempty"`
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(config *AuditConfig) (*AuditLogger, error) {
	if config == nil {
		config = DefaultAuditConfig()
	}

	// Create storage backend
	var storage AuditStorage
	var err error

	switch config.StorageType {
	case "memory":
		storage = NewMemoryAuditStorage()
	case "file":
		storage, err = NewFileAuditStorage(config.StoragePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create file audit storage: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported audit storage type: %s", config.StorageType)
	}

	// Create encryptor if encryption is enabled
	var encryptor *Encryptor
	if config.EnableEncryption {
		key, err := GenerateKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate encryption key: %w", err)
		}
		encryptor, err = NewEncryptor(key)
		if err != nil {
			return nil, fmt.Errorf("failed to create encryptor: %w", err)
		}
	}

	logger := &AuditLogger{
		storage:   storage,
		encryptor: encryptor,
		config:    config,
		chain:     &AuditChain{},
	}

	// Initialize chain with last hash
	logger.initializeChain()

	// Start cleanup routine
	go logger.startCleanupRoutine()

	return logger, nil
}

// Log logs an audit entry
func (al *AuditLogger) Log(entry *AuditEntry) error {
	al.mu.Lock()
	defer al.mu.Unlock()

	// Set timestamp and ID if not provided
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}
	if entry.ID == "" {
		entry.ID = generateAuditID()
	}

	// Calculate hash and maintain chain integrity
	if al.config.EnableIntegrityCheck {
		al.chain.mu.Lock()
		entry.PrevHash = al.chain.lastHash
		entry.Hash = al.calculateHash(entry)
		al.chain.lastHash = entry.Hash
		al.chain.mu.Unlock()
	}

	// Store the entry
	if err := al.storage.Store(entry); err != nil {
		return fmt.Errorf("failed to store audit entry: %w", err)
	}

	return nil
}

// LogUserAction logs a user action
func (al *AuditLogger) LogUserAction(userID, action, resource string, details map[string]interface{}, success bool, errorMsg string) error {
	entry := &AuditEntry{
		UserID:   userID,
		Action:   action,
		Resource: resource,
		Details:  details,
		Success:  success,
		ErrorMsg: errorMsg,
	}

	return al.Log(entry)
}

// LogSecurityEvent logs a security-related event
func (al *AuditLogger) LogSecurityEvent(eventType, description string, details map[string]interface{}) error {
	entry := &AuditEntry{
		UserID:   "system",
		Action:   "security_event",
		Resource: eventType,
		Details: map[string]interface{}{
			"event_type":  eventType,
			"description": description,
			"details":     details,
		},
		Success: true,
	}

	return al.Log(entry)
}

// LogAPIAccess logs API access
func (al *AuditLogger) LogAPIAccess(userID, method, endpoint, ipAddress, userAgent string, statusCode int, duration time.Duration) error {
	entry := &AuditEntry{
		UserID:    userID,
		Action:    "api_access",
		Resource:  fmt.Sprintf("%s %s", method, endpoint),
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Success:   statusCode < 400,
		Details: map[string]interface{}{
			"method":      method,
			"endpoint":    endpoint,
			"status_code": statusCode,
			"duration_ms": duration.Milliseconds(),
		},
	}

	if statusCode >= 400 {
		entry.ErrorMsg = fmt.Sprintf("HTTP %d", statusCode)
	}

	return al.Log(entry)
}

// GetEntries retrieves audit entries with optional filtering
func (al *AuditLogger) GetEntries(filter *AuditFilter) ([]*AuditEntry, error) {
	al.mu.RLock()
	defer al.mu.RUnlock()

	return al.storage.List(filter)
}

// VerifyIntegrity verifies the integrity of the audit log chain
func (al *AuditLogger) VerifyIntegrity() (*IntegrityReport, error) {
	if !al.config.EnableIntegrityCheck {
		return nil, fmt.Errorf("integrity checking is not enabled")
	}

	al.mu.RLock()
	defer al.mu.RUnlock()

	entries, err := al.storage.List(&AuditFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve entries for integrity check: %w", err)
	}

	report := &IntegrityReport{
		TotalEntries:    len(entries),
		VerifiedEntries: 0,
		TamperedEntries: make([]string, 0),
		MissingEntries:  make([]string, 0),
		CheckTime:       time.Now(),
	}

	var prevHash string
	for i, entry := range entries {
		// Verify previous hash link
		if entry.PrevHash != prevHash {
			if i > 0 { // Skip first entry
				report.TamperedEntries = append(report.TamperedEntries, entry.ID)
				continue
			}
		}

		// Verify entry hash
		expectedHash := al.calculateHash(entry)
		if entry.Hash != expectedHash {
			report.TamperedEntries = append(report.TamperedEntries, entry.ID)
			continue
		}

		report.VerifiedEntries++
		prevHash = entry.Hash
	}

	report.IntegrityValid = len(report.TamperedEntries) == 0
	return report, nil
}

// ExportLogs exports audit logs for external analysis
func (al *AuditLogger) ExportLogs(filter *AuditFilter, format string) ([]byte, error) {
	entries, err := al.GetEntries(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get entries: %w", err)
	}

	switch format {
	case "json":
		return json.MarshalIndent(entries, "", "  ")
	case "csv":
		return al.exportToCSV(entries)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// calculateHash calculates the hash of an audit entry
func (al *AuditLogger) calculateHash(entry *AuditEntry) string {
	// Create a copy without hash and signature for calculation
	hashEntry := *entry
	hashEntry.Hash = ""
	hashEntry.Signature = ""

	data, _ := json.Marshal(hashEntry)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// initializeChain initializes the audit chain with the last hash
func (al *AuditLogger) initializeChain() {
	entries, err := al.storage.List(&AuditFilter{Limit: 1})
	if err != nil || len(entries) == 0 {
		al.chain.lastHash = ""
		return
	}

	al.chain.lastHash = entries[0].Hash
}

// startCleanupRoutine starts the cleanup routine for old entries
func (al *AuditLogger) startCleanupRoutine() {
	if al.config.RetentionPeriod <= 0 {
		return
	}

	ticker := time.NewTicker(24 * time.Hour) // Run daily
	defer ticker.Stop()

	for range ticker.C {
		al.cleanup()
	}
}

// cleanup removes old audit entries
func (al *AuditLogger) cleanup() {
	cutoff := time.Now().Add(-al.config.RetentionPeriod)
	
	entries, err := al.storage.List(&AuditFilter{EndTime: cutoff})
	if err != nil {
		return
	}

	for _, entry := range entries {
		al.storage.Delete(entry.ID)
	}
}

// exportToCSV exports entries to CSV format
func (al *AuditLogger) exportToCSV(entries []*AuditEntry) ([]byte, error) {
	// This is a simplified CSV export
	// In production, you'd want a proper CSV library
	csv := "ID,Timestamp,UserID,Action,Resource,Success,IPAddress,ErrorMsg\n"
	
	for _, entry := range entries {
		csv += fmt.Sprintf("%s,%s,%s,%s,%s,%t,%s,%s\n",
			entry.ID,
			entry.Timestamp.Format(time.RFC3339),
			entry.UserID,
			entry.Action,
			entry.Resource,
			entry.Success,
			entry.IPAddress,
			entry.ErrorMsg,
		)
	}

	return []byte(csv), nil
}

// DefaultAuditConfig returns default audit configuration
func DefaultAuditConfig() *AuditConfig {
	return &AuditConfig{
		EnableEncryption:     true,
		EnableIntegrityCheck: true,
		RetentionPeriod:      365 * 24 * time.Hour, // 1 year
		MaxEntries:           1000000,              // 1 million entries
		StorageType:          "file",
		StoragePath:          "./audit_logs",
		AlertOnTampering:     true,
	}
}

// generateAuditID generates a unique audit entry ID
func generateAuditID() string {
	return fmt.Sprintf("audit-%d", time.Now().UnixNano())
}

// IntegrityReport represents the result of an integrity check
type IntegrityReport struct {
	TotalEntries    int       `json:"total_entries"`
	VerifiedEntries int       `json:"verified_entries"`
	TamperedEntries []string  `json:"tampered_entries"`
	MissingEntries  []string  `json:"missing_entries"`
	IntegrityValid  bool      `json:"integrity_valid"`
	CheckTime       time.Time `json:"check_time"`
}