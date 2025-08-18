package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	
	_ "github.com/lib/pq" // PostgreSQL driver
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
	db     *sql.DB
	config *DatabaseStorageConfig
	mu     sync.RWMutex
}

// DatabaseStorageConfig represents database storage configuration
type DatabaseStorageConfig struct {
	TableName     string `json:"table_name"`
	KeyColumn     string `json:"key_column"`
	DataColumn    string `json:"data_column"`
	CreatedColumn string `json:"created_column"`
	UpdatedColumn string `json:"updated_column"`
	EncryptionKey string `json:"encryption_key"`
}

// NewDatabaseVaultStorage creates a new database vault storage
func NewDatabaseVaultStorage(connectionString string) (*DatabaseVaultStorage, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}
	
	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	
	config := &DatabaseStorageConfig{
		TableName:     "vault_storage",
		KeyColumn:     "key_name",
		DataColumn:    "encrypted_data",
		CreatedColumn: "created_at",
		UpdatedColumn: "updated_at",
		EncryptionKey: generateEncryptionKey(),
	}
	
	storage := &DatabaseVaultStorage{
		db:     db,
		config: config,
	}
	
	// Initialize database schema
	if err := storage.initializeSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize database schema: %w", err)
	}
	
	return storage, nil
}

// Store implements VaultStorage interface for DatabaseVaultStorage
func (d *DatabaseVaultStorage) Store(key string, data []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	// Encrypt data before storing
	encryptedData, err := d.encryptData(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt data: %w", err)
	}
	
	// Use UPSERT to handle both insert and update
	upsertSQL := fmt.Sprintf(`
		INSERT INTO %s (%s, %s, %s, %s) 
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (%s) 
		DO UPDATE SET 
			%s = EXCLUDED.%s,
			%s = NOW()`,
		d.config.TableName,
		d.config.KeyColumn, d.config.DataColumn, d.config.CreatedColumn, d.config.UpdatedColumn,
		d.config.KeyColumn,
		d.config.DataColumn, d.config.DataColumn,
		d.config.UpdatedColumn,
	)
	
	_, err = d.db.Exec(upsertSQL, key, encryptedData)
	if err != nil {
		return fmt.Errorf("failed to store data in database: %w", err)
	}
	
	return nil
}

func (d *DatabaseVaultStorage) Retrieve(key string) ([]byte, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	selectSQL := fmt.Sprintf(`
		SELECT %s FROM %s WHERE %s = $1`,
		d.config.DataColumn, d.config.TableName, d.config.KeyColumn,
	)
	
	var encryptedData []byte
	err := d.db.QueryRow(selectSQL, key).Scan(&encryptedData)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("key not found: %s", key)
		}
		return nil, fmt.Errorf("failed to retrieve data from database: %w", err)
	}
	
	// Decrypt data before returning
	data, err := d.decryptData(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}
	
	return data, nil
}

func (d *DatabaseVaultStorage) Delete(key string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	deleteSQL := fmt.Sprintf(`
		DELETE FROM %s WHERE %s = $1`,
		d.config.TableName, d.config.KeyColumn,
	)
	
	result, err := d.db.Exec(deleteSQL, key)
	if err != nil {
		return fmt.Errorf("failed to delete data from database: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("key not found: %s", key)
	}
	
	return nil
}

func (d *DatabaseVaultStorage) List(prefix string) ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	var selectSQL string
	var args []interface{}
	
	if prefix == "" {
		selectSQL = fmt.Sprintf(`
			SELECT %s FROM %s ORDER BY %s`,
			d.config.KeyColumn, d.config.TableName, d.config.KeyColumn,
		)
	} else {
		selectSQL = fmt.Sprintf(`
			SELECT %s FROM %s WHERE %s LIKE $1 ORDER BY %s`,
			d.config.KeyColumn, d.config.TableName, d.config.KeyColumn, d.config.KeyColumn,
		)
		args = append(args, prefix+"%")
	}
	
	rows, err := d.db.Query(selectSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list keys from database: %w", err)
	}
	defer rows.Close()
	
	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("failed to scan key: %w", err)
		}
		keys = append(keys, key)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}
	
	return keys, nil
}

func (d *DatabaseVaultStorage) Exists(key string) (bool, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	selectSQL := fmt.Sprintf(`
		SELECT 1 FROM %s WHERE %s = $1 LIMIT 1`,
		d.config.TableName, d.config.KeyColumn,
	)
	
	var exists int
	err := d.db.QueryRow(selectSQL, key).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to check existence in database: %w", err)
	}
	
	return true, nil
}

// initializeSchema creates the necessary database tables
func (d *DatabaseVaultStorage) initializeSchema() error {
	createTableSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			%s VARCHAR(255) PRIMARY KEY,
			%s BYTEA NOT NULL,
			%s TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			%s TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		d.config.TableName,
		d.config.KeyColumn,
		d.config.DataColumn,
		d.config.CreatedColumn,
		d.config.UpdatedColumn,
	)
	
	_, err := d.db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create vault storage table: %w", err)
	}
	
	// Create index on key column for faster lookups
	indexSQL := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS idx_%s_%s ON %s (%s)`,
		d.config.TableName, d.config.KeyColumn, d.config.TableName, d.config.KeyColumn,
	)
	
	_, err = d.db.Exec(indexSQL)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	
	return nil
}

// encryptData encrypts data using AES-GCM
func (d *DatabaseVaultStorage) encryptData(data []byte) ([]byte, error) {
	// Use AES-256-GCM for encryption
	key := []byte(d.config.EncryptionKey)[:32] // Use first 32 bytes as key
	
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// decryptData decrypts data using AES-GCM
func (d *DatabaseVaultStorage) decryptData(encryptedData []byte) ([]byte, error) {
	key := []byte(d.config.EncryptionKey)[:32] // Use first 32 bytes as key
	
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	
	nonceSize := gcm.NonceSize()
	if len(encryptedData) < nonceSize {
		return nil, fmt.Errorf("encrypted data too short")
	}
	
	nonce, ciphertext := encryptedData[:nonceSize], encryptedData[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}
	
	return plaintext, nil
}

// generateEncryptionKey generates a random encryption key
func generateEncryptionKey() string {
	key := make([]byte, 32) // 256-bit key
	if _, err := rand.Read(key); err != nil {
		// Fallback to a deterministic key (not recommended for production)
		return "default-encryption-key-not-secure-replace-immediately-with-proper-key"
	}
	return base64.StdEncoding.EncodeToString(key)
}

func (d *DatabaseVaultStorage) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}