package security

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// MemoryVaultStorage implements in-memory storage for the vault
type MemoryVaultStorage struct {
	data map[string][]byte
	mu   sync.RWMutex
}

// NewMemoryVaultStorage creates a new memory vault storage
func NewMemoryVaultStorage() *MemoryVaultStorage {
	return &MemoryVaultStorage{
		data: make(map[string][]byte),
	}
}

// Store stores data with the given key
func (m *MemoryVaultStorage) Store(key string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Make a copy of the data to avoid external modifications
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	
	m.data[key] = dataCopy
	return nil
}

// Retrieve retrieves data for the given key
func (m *MemoryVaultStorage) Retrieve(key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	data, exists := m.data[key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	
	// Return a copy to avoid external modifications
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	
	return dataCopy, nil
}

// Delete deletes data for the given key
func (m *MemoryVaultStorage) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.data, key)
	return nil
}

// List lists all keys with the given prefix
func (m *MemoryVaultStorage) List(prefix string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var keys []string
	for key := range m.data {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	
	return keys, nil
}

// Exists checks if a key exists
func (m *MemoryVaultStorage) Exists(key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	_, exists := m.data[key]
	return exists, nil
}

// FileVaultStorage implements file-based storage for the vault
type FileVaultStorage struct {
	basePath string
	mu       sync.RWMutex
}

// NewFileVaultStorage creates a new file vault storage
func NewFileVaultStorage(basePath string) (*FileVaultStorage, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	
	return &FileVaultStorage{
		basePath: basePath,
	}, nil
}

// Store stores data in a file
func (f *FileVaultStorage) Store(key string, data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	filePath := f.getFilePath(key)
	
	// Create directory if needed
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Write file with secure permissions
	if err := ioutil.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	return nil
}

// Retrieve retrieves data from a file
func (f *FileVaultStorage) Retrieve(key string) ([]byte, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	filePath := f.getFilePath(key)
	
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("key not found: %s", key)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	
	return data, nil
}

// Delete deletes a file
func (f *FileVaultStorage) Delete(key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	filePath := f.getFilePath(key)
	
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	
	return nil
}

// List lists all files with the given prefix
func (f *FileVaultStorage) List(prefix string) ([]string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	var keys []string
	
	err := filepath.Walk(f.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if info.IsDir() {
			return nil
		}
		
		// Convert file path back to key
		relPath, err := filepath.Rel(f.basePath, path)
		if err != nil {
			return err
		}
		
		key := f.pathToKey(relPath)
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}
	
	return keys, nil
}

// Exists checks if a file exists
func (f *FileVaultStorage) Exists(key string) (bool, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	filePath := f.getFilePath(key)
	
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check file existence: %w", err)
	}
	
	return true, nil
}

// getFilePath converts a key to a file path
func (f *FileVaultStorage) getFilePath(key string) string {
	// Replace colons and other special characters with path separators
	safePath := strings.ReplaceAll(key, ":", string(filepath.Separator))
	return filepath.Join(f.basePath, safePath+".vault")
}

// pathToKey converts a file path back to a key
func (f *FileVaultStorage) pathToKey(path string) string {
	// Remove .vault extension
	if strings.HasSuffix(path, ".vault") {
		path = path[:len(path)-6]
	}
	
	// Replace path separators with colons
	return strings.ReplaceAll(path, string(filepath.Separator), ":")
}

// DatabaseVaultStorage implements database-based storage for the vault
type DatabaseVaultStorage struct {
	// TODO: Implement database storage
	// This would use the existing database connection
	// to store encrypted key data in a dedicated table
}

// NewDatabaseVaultStorage creates a new database vault storage
func NewDatabaseVaultStorage(connectionString string) (*DatabaseVaultStorage, error) {
	// TODO: Implement database storage
	return nil, fmt.Errorf("database vault storage not implemented yet")
}

// Implement VaultStorage interface for DatabaseVaultStorage
func (d *DatabaseVaultStorage) Store(key string, data []byte) error {
	return fmt.Errorf("not implemented")
}

func (d *DatabaseVaultStorage) Retrieve(key string) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (d *DatabaseVaultStorage) Delete(key string) error {
	return fmt.Errorf("not implemented")
}

func (d *DatabaseVaultStorage) List(prefix string) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}

func (d *DatabaseVaultStorage) Exists(key string) (bool, error) {
	return false, fmt.Errorf("not implemented")
}