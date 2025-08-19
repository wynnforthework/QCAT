package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/golang-migrate/migrate/v4"
)

// MigrationMonitor monitors database migration health and provides recovery mechanisms
type MigrationMonitor struct {
	migrator *Migrator
	config   *MigrationMonitorConfig
}

// MigrationMonitorConfig contains configuration for migration monitoring
type MigrationMonitorConfig struct {
	CheckInterval    time.Duration // How often to check migration status
	MaxRetries       int           // Maximum number of retry attempts
	RetryDelay       time.Duration // Delay between retry attempts
	AlertThreshold   int           // Number of failures before alerting
	AutoRecovery     bool          // Whether to attempt automatic recovery
	NotificationFunc func(string)  // Function to call for notifications
}

// MigrationStatus represents the current status of database migrations
type MigrationStatus struct {
	CurrentVersion uint
	IsDirty        bool
	LastChecked    time.Time
	ErrorCount     int
	LastError      error
}

// NewMigrationMonitor creates a new migration monitor
func NewMigrationMonitor(migrator *Migrator, config *MigrationMonitorConfig) *MigrationMonitor {
	if config == nil {
		config = &MigrationMonitorConfig{
			CheckInterval:  30 * time.Second,
			MaxRetries:     3,
			RetryDelay:     5 * time.Second,
			AlertThreshold: 2,
			AutoRecovery:   true,
		}
	}
	
	return &MigrationMonitor{
		migrator: migrator,
		config:   config,
	}
}

// Start begins monitoring database migration status
func (m *MigrationMonitor) Start(ctx context.Context) {
	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	log.Println("Migration monitor started")

	for {
		select {
		case <-ctx.Done():
			log.Println("Migration monitor stopped")
			return
		case <-ticker.C:
			m.checkMigrationStatus()
		}
	}
}

// checkMigrationStatus checks the current migration status and handles issues
func (m *MigrationMonitor) checkMigrationStatus() {
	status := m.getMigrationStatus()
	
	if status.IsDirty {
		log.Printf("⚠️ Detected dirty migration state at version %d", status.CurrentVersion)
		m.handleDirtyState(status)
	} else if status.LastError != nil {
		log.Printf("⚠️ Migration error detected: %v", status.LastError)
		m.handleMigrationError(status)
	} else {
		log.Printf("✅ Migration status healthy - version: %d", status.CurrentVersion)
	}
}

// getMigrationStatus retrieves the current migration status
func (m *MigrationMonitor) getMigrationStatus() *MigrationStatus {
	status := &MigrationStatus{
		LastChecked: time.Now(),
	}

	version, dirty, err := m.migrator.migrate.Version()
	if err != nil {
		status.LastError = err
		if err == migrate.ErrNilVersion {
			status.CurrentVersion = 0
		}
	} else {
		status.CurrentVersion = version
		status.IsDirty = dirty
	}

	return status
}

// handleDirtyState attempts to recover from a dirty migration state
func (m *MigrationMonitor) handleDirtyState(status *MigrationStatus) {
	if !m.config.AutoRecovery {
		m.notify(fmt.Sprintf("Dirty migration state detected at version %d. Manual intervention required.", status.CurrentVersion))
		return
	}

	log.Printf("Attempting automatic recovery from dirty state at version %d", status.CurrentVersion)

	// Strategy 1: Try to force the current version
	if err := m.migrator.Force(int(status.CurrentVersion)); err != nil {
		log.Printf("Failed to force version %d: %v", status.CurrentVersion, err)
		
		// Strategy 2: Try to force the previous version
		if status.CurrentVersion > 0 {
			prevVersion := int(status.CurrentVersion - 1)
			log.Printf("Attempting to force previous version %d", prevVersion)
			if err := m.migrator.Force(prevVersion); err != nil {
				log.Printf("Failed to force previous version %d: %v", prevVersion, err)
				m.notify(fmt.Sprintf("Automatic recovery failed for dirty state at version %d. Manual intervention required.", status.CurrentVersion))
				return
			}
		}
	}

	// Verify recovery
	if newStatus := m.getMigrationStatus(); !newStatus.IsDirty {
		log.Printf("✅ Successfully recovered from dirty state")
		m.notify(fmt.Sprintf("Successfully recovered from dirty migration state at version %d", status.CurrentVersion))
		
		// Try to run migrations again
		if err := m.migrator.Up(); err != nil && err != migrate.ErrNoChange {
			log.Printf("Failed to run migrations after recovery: %v", err)
		}
	} else {
		m.notify(fmt.Sprintf("Recovery attempt failed for dirty state at version %d", status.CurrentVersion))
	}
}

// handleMigrationError handles general migration errors
func (m *MigrationMonitor) handleMigrationError(status *MigrationStatus) {
	status.ErrorCount++
	
	if status.ErrorCount >= m.config.AlertThreshold {
		m.notify(fmt.Sprintf("Migration error threshold exceeded: %v", status.LastError))
	}

	if m.config.AutoRecovery && status.ErrorCount <= m.config.MaxRetries {
		log.Printf("Attempting retry %d/%d for migration error", status.ErrorCount, m.config.MaxRetries)
		time.Sleep(m.config.RetryDelay)
		
		if err := m.migrator.Up(); err != nil && err != migrate.ErrNoChange {
			log.Printf("Retry %d failed: %v", status.ErrorCount, err)
		} else {
			log.Printf("✅ Migration retry %d succeeded", status.ErrorCount)
			status.ErrorCount = 0
		}
	}
}

// notify sends a notification about migration issues
func (m *MigrationMonitor) notify(message string) {
	log.Printf("🚨 MIGRATION ALERT: %s", message)
	
	if m.config.NotificationFunc != nil {
		m.config.NotificationFunc(message)
	}
}

// GetStatus returns the current migration status
func (m *MigrationMonitor) GetStatus() (*MigrationStatus, error) {
	return m.getMigrationStatus(), nil
}

// ForceRecovery manually triggers recovery from dirty state
func (m *MigrationMonitor) ForceRecovery(targetVersion int) error {
	log.Printf("Manual recovery triggered for version %d", targetVersion)
	
	if err := m.migrator.Force(targetVersion); err != nil {
		return fmt.Errorf("failed to force version %d: %w", targetVersion, err)
	}
	
	// Verify recovery
	if status := m.getMigrationStatus(); status.IsDirty {
		return fmt.Errorf("recovery failed - database still in dirty state")
	}
	
	log.Printf("✅ Manual recovery successful")
	return nil
}

// ValidateMigrationIntegrity checks if all migration files are consistent
func (m *MigrationMonitor) ValidateMigrationIntegrity() error {
	// This could be extended to validate migration file checksums,
	// check for missing files, etc.
	status := m.getMigrationStatus()
	
	if status.LastError != nil {
		return fmt.Errorf("migration integrity check failed: %w", status.LastError)
	}
	
	if status.IsDirty {
		return fmt.Errorf("migration integrity check failed: database in dirty state at version %d", status.CurrentVersion)
	}
	
	return nil
}
