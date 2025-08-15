package disaster_recovery

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// BackupManager manages data backup and disaster recovery
type BackupManager struct {
	// Prometheus metrics
	backupDuration  prometheus.Histogram
	backupSize      prometheus.Gauge
	backupStatus    prometheus.Gauge
	backupErrors    prometheus.Counter
	restoreDuration prometheus.Histogram
	restoreStatus   prometheus.Gauge
	restoreErrors   prometheus.Counter

	// Configuration
	config *BackupConfig

	// State
	backups map[string]*BackupInfo
	mu      sync.RWMutex

	// Channels
	backupCh chan *BackupRequest
	stopCh   chan struct{}
}

// BackupConfig represents backup configuration
type BackupConfig struct {
	BackupDir            string
	RetentionDays        int
	CompressionEnabled   bool
	EncryptionEnabled    bool
	EncryptionKey        string
	MaxConcurrentBackups int
	BackupInterval       time.Duration
	VerifyBackup         bool
	AutoBackup           bool
}

// BackupInfo represents backup information
type BackupInfo struct {
	ID        string
	Name      string
	Type      BackupType
	Path      string
	Size      int64
	Checksum  string
	CreatedAt time.Time
	Status    BackupStatus
	Error     string
	Metadata  map[string]interface{}
}

// BackupType represents the type of backup
type BackupType string

const (
	BackupTypeFull         BackupType = "full"
	BackupTypeIncremental  BackupType = "incremental"
	BackupTypeDifferential BackupType = "differential"
)

// BackupStatus represents backup status
type BackupStatus string

const (
	BackupStatusPending   BackupStatus = "pending"
	BackupStatusRunning   BackupStatus = "running"
	BackupStatusCompleted BackupStatus = "completed"
	BackupStatusFailed    BackupStatus = "failed"
	BackupStatusVerified  BackupStatus = "verified"
)

// BackupRequest represents a backup request
type BackupRequest struct {
	ID       string
	Name     string
	Type     BackupType
	Data     []byte
	Metadata map[string]interface{}
	Callback func(*BackupInfo, error)
}

// RestoreRequest represents a restore request
type RestoreRequest struct {
	BackupID string
	Target   string
	Callback func(error)
}

// NewBackupManager creates a new backup manager
func NewBackupManager(config *BackupConfig) *BackupManager {
	if config == nil {
		config = &BackupConfig{
			BackupDir:            "backups",
			RetentionDays:        30,
			CompressionEnabled:   true,
			EncryptionEnabled:    false,
			MaxConcurrentBackups: 3,
			BackupInterval:       24 * time.Hour,
			VerifyBackup:         true,
			AutoBackup:           true,
		}
	}

	bm := &BackupManager{
		config:   config,
		backups:  make(map[string]*BackupInfo),
		backupCh: make(chan *BackupRequest, 100),
		stopCh:   make(chan struct{}),
	}

	// Initialize Prometheus metrics
	bm.initializeMetrics()

	return bm
}

// initializeMetrics initializes Prometheus metrics
func (bm *BackupManager) initializeMetrics() {
	bm.backupDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "backup_duration_seconds",
		Help:    "Backup duration in seconds",
		Buckets: prometheus.ExponentialBuckets(1, 2, 10),
	})

	bm.backupSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "backup_size_bytes",
		Help: "Backup size in bytes",
	})

	bm.backupStatus = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "backup_status",
		Help: "Backup status (0=pending, 1=running, 2=completed, 3=failed, 4=verified)",
	})

	bm.backupErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "backup_errors_total",
		Help: "Total number of backup errors",
	})

	bm.restoreDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "restore_duration_seconds",
		Help:    "Restore duration in seconds",
		Buckets: prometheus.ExponentialBuckets(1, 2, 10),
	})

	bm.restoreStatus = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "restore_status",
		Help: "Restore status (0=pending, 1=running, 2=completed, 3=failed)",
	})

	bm.restoreErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "restore_errors_total",
		Help: "Total number of restore errors",
	})
}

// Start starts the backup manager
func (bm *BackupManager) Start() error {
	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(bm.config.BackupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Start backup worker
	go bm.backupWorker()

	// Start automatic backup if enabled
	if bm.config.AutoBackup {
		go bm.autoBackupWorker()
	}

	// Start cleanup worker
	go bm.cleanupWorker()

	return nil
}

// Stop stops the backup manager
func (bm *BackupManager) Stop() {
	close(bm.stopCh)
}

// CreateBackup creates a new backup
func (bm *BackupManager) CreateBackup(ctx context.Context, name string, backupType BackupType, data []byte, metadata map[string]interface{}) (*BackupInfo, error) {
	backupID := generateBackupID()
	backupInfo := &BackupInfo{
		ID:        backupID,
		Name:      name,
		Type:      backupType,
		Status:    BackupStatusPending,
		CreatedAt: time.Now(),
		Metadata:  metadata,
	}

	// Add to backups map
	bm.mu.Lock()
	bm.backups[backupID] = backupInfo
	bm.mu.Unlock()

	// Send backup request
	request := &BackupRequest{
		ID:       backupID,
		Name:     name,
		Type:     backupType,
		Data:     data,
		Metadata: metadata,
	}

	select {
	case bm.backupCh <- request:
		return backupInfo, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return nil, fmt.Errorf("backup queue is full")
	}
}

// RestoreBackup restores data from a backup
func (bm *BackupManager) RestoreBackup(ctx context.Context, backupID string, target string) error {
	bm.mu.RLock()
	backupInfo, exists := bm.backups[backupID]
	bm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("backup %s not found", backupID)
	}

	if backupInfo.Status != BackupStatusCompleted && backupInfo.Status != BackupStatusVerified {
		return fmt.Errorf("backup %s is not ready for restore (status: %s)", backupID, backupInfo.Status)
	}

	start := time.Now()
	bm.restoreStatus.Set(1.0) // Running

	// Perform restore
	if err := bm.performRestore(backupInfo, target); err != nil {
		bm.restoreStatus.Set(3.0) // Failed
		bm.restoreErrors.Inc()
		return fmt.Errorf("restore failed: %w", err)
	}

	duration := time.Since(start)
	bm.restoreDuration.Observe(duration.Seconds())
	bm.restoreStatus.Set(2.0) // Completed

	return nil
}

// GetBackup gets backup information
func (bm *BackupManager) GetBackup(backupID string) *BackupInfo {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.backups[backupID]
}

// ListBackups lists all backups
func (bm *BackupManager) ListBackups() []*BackupInfo {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	backups := make([]*BackupInfo, 0, len(bm.backups))
	for _, backup := range bm.backups {
		backups = append(backups, backup)
	}

	return backups
}

// DeleteBackup deletes a backup
func (bm *BackupManager) DeleteBackup(backupID string) error {
	bm.mu.Lock()
	backupInfo, exists := bm.backups[backupID]
	if !exists {
		bm.mu.Unlock()
		return fmt.Errorf("backup %s not found", backupID)
	}
	delete(bm.backups, backupID)
	bm.mu.Unlock()

	// Delete backup file
	if backupInfo.Path != "" {
		if err := os.Remove(backupInfo.Path); err != nil {
			return fmt.Errorf("failed to delete backup file: %w", err)
		}
	}

	return nil
}

// backupWorker processes backup requests
func (bm *BackupManager) backupWorker() {
	semaphore := make(chan struct{}, bm.config.MaxConcurrentBackups)

	for {
		select {
		case <-bm.stopCh:
			return
		case request := <-bm.backupCh:
			semaphore <- struct{}{} // Acquire semaphore
			go func(req *BackupRequest) {
				defer func() { <-semaphore }() // Release semaphore
				bm.processBackup(req)
			}(request)
		}
	}
}

// processBackup processes a single backup request
func (bm *BackupManager) processBackup(request *BackupRequest) {
	start := time.Now()

	// Update status to running
	bm.mu.Lock()
	backupInfo := bm.backups[request.ID]
	if backupInfo != nil {
		backupInfo.Status = BackupStatusRunning
	}
	bm.mu.Unlock()

	bm.backupStatus.Set(1.0) // Running

	// Create backup file
	backupPath := filepath.Join(bm.config.BackupDir, fmt.Sprintf("%s_%s.backup", request.ID, request.Name))

	// Write data to file
	if err := bm.writeBackupFile(backupPath, request.Data); err != nil {
		bm.handleBackupError(request.ID, err)
		return
	}

	// Get file info
	fileInfo, err := os.Stat(backupPath)
	if err != nil {
		bm.handleBackupError(request.ID, err)
		return
	}

	// Calculate checksum
	checksum, err := bm.calculateChecksum(backupPath)
	if err != nil {
		bm.handleBackupError(request.ID, err)
		return
	}

	// Update backup info
	bm.mu.Lock()
	if backupInfo := bm.backups[request.ID]; backupInfo != nil {
		backupInfo.Path = backupPath
		backupInfo.Size = fileInfo.Size()
		backupInfo.Checksum = checksum
		backupInfo.Status = BackupStatusCompleted
	}
	bm.mu.Unlock()

	// Update metrics
	duration := time.Since(start)
	bm.backupDuration.Observe(duration.Seconds())
	bm.backupSize.Set(float64(fileInfo.Size()))
	bm.backupStatus.Set(2.0) // Completed

	// Verify backup if enabled
	if bm.config.VerifyBackup {
		if err := bm.verifyBackup(request.ID); err != nil {
			bm.backupErrors.Inc()
			// Don't fail the backup, just log the verification error
			fmt.Printf("Backup verification failed for %s: %v\n", request.ID, err)
		} else {
			bm.mu.Lock()
			if backupInfo := bm.backups[request.ID]; backupInfo != nil {
				backupInfo.Status = BackupStatusVerified
			}
			bm.mu.Unlock()
			bm.backupStatus.Set(4.0) // Verified
		}
	}

	// Call callback if provided
	if request.Callback != nil {
		backupInfo := bm.GetBackup(request.ID)
		request.Callback(backupInfo, nil)
	}
}

// handleBackupError handles backup errors
func (bm *BackupManager) handleBackupError(backupID string, err error) {
	bm.mu.Lock()
	if backupInfo := bm.backups[backupID]; backupInfo != nil {
		backupInfo.Status = BackupStatusFailed
		backupInfo.Error = err.Error()
	}
	bm.mu.Unlock()

	bm.backupStatus.Set(3.0) // Failed
	bm.backupErrors.Inc()
}

// writeBackupFile writes backup data to file
func (bm *BackupManager) writeBackupFile(path string, data []byte) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write data
	if _, err := file.Write(data); err != nil {
		return err
	}

	return file.Sync()
}

// calculateChecksum calculates MD5 checksum of a file
func (bm *BackupManager) calculateChecksum(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// verifyBackup verifies a backup
func (bm *BackupManager) verifyBackup(backupID string) error {
	backupInfo := bm.GetBackup(backupID)
	if backupInfo == nil {
		return fmt.Errorf("backup not found")
	}

	// Calculate current checksum
	currentChecksum, err := bm.calculateChecksum(backupInfo.Path)
	if err != nil {
		return err
	}

	// Compare checksums
	if currentChecksum != backupInfo.Checksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", backupInfo.Checksum, currentChecksum)
	}

	return nil
}

// performRestore performs the actual restore operation
func (bm *BackupManager) performRestore(backupInfo *BackupInfo, target string) error {
	// Read backup file
	data, err := os.ReadFile(backupInfo.Path)
	if err != nil {
		return err
	}

	// Verify checksum
	checksum, err := bm.calculateChecksum(backupInfo.Path)
	if err != nil {
		return err
	}

	if checksum != backupInfo.Checksum {
		return fmt.Errorf("backup file corrupted: checksum mismatch")
	}

	// Write to target
	if err := os.WriteFile(target, data, 0644); err != nil {
		return err
	}

	return nil
}

// autoBackupWorker performs automatic backups
func (bm *BackupManager) autoBackupWorker() {
	ticker := time.NewTicker(bm.config.BackupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-bm.stopCh:
			return
		case <-ticker.C:
			// Perform automatic backup
			bm.performAutoBackup()
		}
	}
}

// performAutoBackup performs an automatic backup
func (bm *BackupManager) performAutoBackup() {
	// This is a simplified implementation
	// In a real system, you would collect data from various sources
	data := []byte("automatic backup data")

	_, err := bm.CreateBackup(context.Background(), "auto_backup", BackupTypeFull, data, map[string]interface{}{
		"auto": true,
		"time": time.Now(),
	})

	if err != nil {
		fmt.Printf("Automatic backup failed: %v\n", err)
	}
}

// cleanupWorker cleans up old backups
func (bm *BackupManager) cleanupWorker() {
	ticker := time.NewTicker(24 * time.Hour) // Run daily
	defer ticker.Stop()

	for {
		select {
		case <-bm.stopCh:
			return
		case <-ticker.C:
			bm.cleanupOldBackups()
		}
	}
}

// cleanupOldBackups removes backups older than retention period
func (bm *BackupManager) cleanupOldBackups() {
	cutoff := time.Now().AddDate(0, 0, -bm.config.RetentionDays)

	bm.mu.Lock()
	defer bm.mu.Unlock()

	for backupID, backupInfo := range bm.backups {
		if backupInfo.CreatedAt.Before(cutoff) {
			// Delete backup file
			if backupInfo.Path != "" {
				os.Remove(backupInfo.Path)
			}
			delete(bm.backups, backupID)
		}
	}
}

// generateBackupID generates a unique backup ID
func generateBackupID() string {
	return fmt.Sprintf("backup_%d", time.Now().UnixNano())
}
