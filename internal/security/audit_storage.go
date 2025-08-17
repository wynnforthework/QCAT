package security

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// MemoryAuditStorage implements in-memory storage for audit logs
type MemoryAuditStorage struct {
	entries map[string]*AuditEntry
	mu      sync.RWMutex
}

// NewMemoryAuditStorage creates a new memory audit storage
func NewMemoryAuditStorage() *MemoryAuditStorage {
	return &MemoryAuditStorage{
		entries: make(map[string]*AuditEntry),
	}
}

// Store stores an audit entry
func (m *MemoryAuditStorage) Store(entry *AuditEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Make a copy to avoid external modifications
	entryCopy := *entry
	m.entries[entry.ID] = &entryCopy

	return nil
}

// Retrieve retrieves an audit entry by ID
func (m *MemoryAuditStorage) Retrieve(id string) (*AuditEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, exists := m.entries[id]
	if !exists {
		return nil, fmt.Errorf("audit entry not found: %s", id)
	}

	// Return a copy to avoid external modifications
	entryCopy := *entry
	return &entryCopy, nil
}

// List lists audit entries with optional filtering
func (m *MemoryAuditStorage) List(filter *AuditFilter) ([]*AuditEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var entries []*AuditEntry
	for _, entry := range m.entries {
		if m.matchesFilter(entry, filter) {
			entryCopy := *entry
			entries = append(entries, &entryCopy)
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})

	// Apply limit
	if filter != nil && filter.Limit > 0 && len(entries) > filter.Limit {
		entries = entries[:filter.Limit]
	}

	return entries, nil
}

// Delete deletes an audit entry
func (m *MemoryAuditStorage) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.entries, id)
	return nil
}

// matchesFilter checks if an entry matches the filter criteria
func (m *MemoryAuditStorage) matchesFilter(entry *AuditEntry, filter *AuditFilter) bool {
	if filter == nil {
		return true
	}

	if filter.UserID != "" && entry.UserID != filter.UserID {
		return false
	}

	if filter.Action != "" && entry.Action != filter.Action {
		return false
	}

	if filter.Resource != "" && entry.Resource != filter.Resource {
		return false
	}

	if !filter.StartTime.IsZero() && entry.Timestamp.Before(filter.StartTime) {
		return false
	}

	if !filter.EndTime.IsZero() && entry.Timestamp.After(filter.EndTime) {
		return false
	}

	if filter.Success != nil && entry.Success != *filter.Success {
		return false
	}

	if filter.IPAddress != "" && entry.IPAddress != filter.IPAddress {
		return false
	}

	return true
}

// FileAuditStorage implements file-based storage for audit logs
type FileAuditStorage struct {
	basePath string
	mu       sync.RWMutex
}

// NewFileAuditStorage creates a new file audit storage
func NewFileAuditStorage(basePath string) (*FileAuditStorage, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create audit storage directory: %w", err)
	}

	return &FileAuditStorage{
		basePath: basePath,
	}, nil
}

// Store stores an audit entry in a file
func (f *FileAuditStorage) Store(entry *AuditEntry) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Organize files by date for better performance
	dateDir := entry.Timestamp.Format("2006-01-02")
	dirPath := filepath.Join(f.basePath, dateDir)

	// Create date directory if it doesn't exist
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return fmt.Errorf("failed to create date directory: %w", err)
	}

	// Store entry in JSON file
	filePath := filepath.Join(dirPath, entry.ID+".json")
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal audit entry: %w", err)
	}

	if err := ioutil.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write audit entry file: %w", err)
	}

	return nil
}

// Retrieve retrieves an audit entry from file
func (f *FileAuditStorage) Retrieve(id string) (*AuditEntry, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Search for the file across all date directories
	filePath, err := f.findEntryFile(id)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read audit entry file: %w", err)
	}

	var entry AuditEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal audit entry: %w", err)
	}

	return &entry, nil
}

// List lists audit entries from files with optional filtering
func (f *FileAuditStorage) List(filter *AuditFilter) ([]*AuditEntry, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var entries []*AuditEntry

	// Walk through all date directories
	err := filepath.Walk(f.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}

		// Read and parse the entry
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		var entry AuditEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			return err
		}

		// Apply filter
		if f.matchesFilter(&entry, filter) {
			entries = append(entries, &entry)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk audit storage directory: %w", err)
	}

	// Sort by timestamp (newest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})

	// Apply limit
	if filter != nil && filter.Limit > 0 && len(entries) > filter.Limit {
		entries = entries[:filter.Limit]
	}

	return entries, nil
}

// Delete deletes an audit entry file
func (f *FileAuditStorage) Delete(id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	filePath, err := f.findEntryFile(id)
	if err != nil {
		return err
	}

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete audit entry file: %w", err)
	}

	return nil
}

// findEntryFile finds the file path for an audit entry ID
func (f *FileAuditStorage) findEntryFile(id string) (string, error) {
	var foundPath string

	err := filepath.Walk(f.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, id+".json") {
			foundPath = path
			return filepath.SkipDir // Stop walking once found
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to search for audit entry: %w", err)
	}

	if foundPath == "" {
		return "", fmt.Errorf("audit entry not found: %s", id)
	}

	return foundPath, nil
}

// matchesFilter checks if an entry matches the filter criteria
func (f *FileAuditStorage) matchesFilter(entry *AuditEntry, filter *AuditFilter) bool {
	if filter == nil {
		return true
	}

	if filter.UserID != "" && entry.UserID != filter.UserID {
		return false
	}

	if filter.Action != "" && entry.Action != filter.Action {
		return false
	}

	if filter.Resource != "" && entry.Resource != filter.Resource {
		return false
	}

	if !filter.StartTime.IsZero() && entry.Timestamp.Before(filter.StartTime) {
		return false
	}

	if !filter.EndTime.IsZero() && entry.Timestamp.After(filter.EndTime) {
		return false
	}

	if filter.Success != nil && entry.Success != *filter.Success {
		return false
	}

	if filter.IPAddress != "" && entry.IPAddress != filter.IPAddress {
		return false
	}

	return true
}