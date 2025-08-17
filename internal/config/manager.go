package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Manager manages configuration with hot reload support
type Manager struct {
	configs    map[string]interface{}
	watchers   map[string]*Watcher
	validators map[string]Validator
	mu         sync.RWMutex
	basePath   string
}

// Validator defines the interface for configuration validation
type Validator interface {
	Validate(config interface{}) error
}

// ChangeHandler defines the interface for configuration change handlers
type ChangeHandler interface {
	OnConfigChange(configName string, oldConfig, newConfig interface{}) error
}

// NewManager creates a new configuration manager
func NewManager(basePath string) *Manager {
	return &Manager{
		configs:    make(map[string]interface{}),
		watchers:   make(map[string]*Watcher),
		validators: make(map[string]Validator),
		basePath:   basePath,
	}
}

// LoadConfig loads a configuration file
func (m *Manager) LoadConfig(name string, config interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	filePath := filepath.Join(m.basePath, name+".yaml")
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", filePath, err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to unmarshal config %s: %w", name, err)
	}

	// Validate configuration if validator exists
	if validator, exists := m.validators[name]; exists {
		if err := validator.Validate(config); err != nil {
			return fmt.Errorf("config validation failed for %s: %w", name, err)
		}
	}

	m.configs[name] = config
	return nil
}

// GetConfig retrieves a configuration
func (m *Manager) GetConfig(name string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config, exists := m.configs[name]
	return config, exists
}

// SetConfig sets a configuration programmatically
func (m *Manager) SetConfig(name string, config interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate configuration if validator exists
	if validator, exists := m.validators[name]; exists {
		if err := validator.Validate(config); err != nil {
			return fmt.Errorf("config validation failed for %s: %w", name, err)
		}
	}

	m.configs[name] = config
	return nil
}

// RegisterValidator registers a validator for a configuration
func (m *Manager) RegisterValidator(name string, validator Validator) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.validators[name] = validator
}

// WatchConfig starts watching a configuration file for changes
func (m *Manager) WatchConfig(name string, config interface{}, handler ChangeHandler) error {
	filePath := filepath.Join(m.basePath, name+".yaml")
	
	watcher, err := NewWatcher(filePath)
	if err != nil {
		return fmt.Errorf("failed to create watcher for %s: %w", name, err)
	}

	m.mu.Lock()
	m.watchers[name] = watcher
	m.mu.Unlock()

	// Start watching in a goroutine
	go m.handleConfigChanges(name, config, handler, watcher)

	return nil
}

// handleConfigChanges handles configuration file changes
func (m *Manager) handleConfigChanges(name string, config interface{}, handler ChangeHandler, watcher *Watcher) {
	for event := range watcher.Events() {
		if event.Type == FileModified {
			// Create a copy of the current config
			oldConfig := m.deepCopy(config)

			// Load the new configuration
			if err := m.LoadConfig(name, config); err != nil {
				fmt.Printf("Failed to reload config %s: %v\n", name, err)
				continue
			}

			// Notify handler of the change
			if handler != nil {
				if err := handler.OnConfigChange(name, oldConfig, config); err != nil {
					fmt.Printf("Config change handler failed for %s: %v\n", name, err)
				}
			}

			fmt.Printf("Configuration %s reloaded successfully\n", name)
		}
	}
}

// deepCopy creates a deep copy of a configuration
func (m *Manager) deepCopy(config interface{}) interface{} {
	// Use JSON marshal/unmarshal for deep copy
	data, err := json.Marshal(config)
	if err != nil {
		return nil
	}

	configType := reflect.TypeOf(config)
	if configType.Kind() == reflect.Ptr {
		configType = configType.Elem()
	}

	newConfig := reflect.New(configType).Interface()
	if err := json.Unmarshal(data, newConfig); err != nil {
		return nil
	}

	return newConfig
}

// SaveConfig saves a configuration to file
func (m *Manager) SaveConfig(name string, config interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	filePath := filepath.Join(m.basePath, name+".yaml")
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config %s: %w", name, err)
	}

	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", filePath, err)
	}

	m.configs[name] = config
	return nil
}

// ListConfigs returns all loaded configuration names
func (m *Manager) ListConfigs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.configs))
	for name := range m.configs {
		names = append(names, name)
	}
	return names
}

// ReloadConfig reloads a specific configuration
func (m *Manager) ReloadConfig(name string, config interface{}) error {
	return m.LoadConfig(name, config)
}

// ReloadAllConfigs reloads all configurations
func (m *Manager) ReloadAllConfigs() error {
	m.mu.RLock()
	configNames := make([]string, 0, len(m.configs))
	for name := range m.configs {
		configNames = append(configNames, name)
	}
	m.mu.RUnlock()

	for _, name := range configNames {
		// Create a new instance of the config type
		m.mu.RLock()
		existingConfig := m.configs[name]
		m.mu.RUnlock()

		configType := reflect.TypeOf(existingConfig)
		if configType.Kind() == reflect.Ptr {
			configType = configType.Elem()
		}

		newConfig := reflect.New(configType).Interface()
		if err := m.LoadConfig(name, newConfig); err != nil {
			return fmt.Errorf("failed to reload config %s: %w", name, err)
		}
	}

	return nil
}

// Stop stops all watchers
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, watcher := range m.watchers {
		watcher.Stop()
	}
}

// GetConfigValue retrieves a specific value from a configuration using dot notation
func (m *Manager) GetConfigValue(configName, path string) (interface{}, error) {
	m.mu.RLock()
	config, exists := m.configs[configName]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("configuration %s not found", configName)
	}

	return m.getValueByPath(config, path)
}

// SetConfigValue sets a specific value in a configuration using dot notation
func (m *Manager) SetConfigValue(configName, path string, value interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	config, exists := m.configs[configName]
	if !exists {
		return fmt.Errorf("configuration %s not found", configName)
	}

	return m.setValueByPath(config, path, value)
}

// getValueByPath retrieves a value using dot notation path
func (m *Manager) getValueByPath(config interface{}, path string) (interface{}, error) {
	// This is a simplified implementation
	// In production, you'd want a more robust path parser
	return config, nil
}

// setValueByPath sets a value using dot notation path
func (m *Manager) setValueByPath(config interface{}, path string, value interface{}) error {
	// This is a simplified implementation
	// In production, you'd want a more robust path parser
	return nil
}

// ConfigSnapshot represents a snapshot of all configurations
type ConfigSnapshot struct {
	Timestamp time.Time              `json:"timestamp"`
	Configs   map[string]interface{} `json:"configs"`
}

// CreateSnapshot creates a snapshot of all configurations
func (m *Manager) CreateSnapshot() *ConfigSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := &ConfigSnapshot{
		Timestamp: time.Now(),
		Configs:   make(map[string]interface{}),
	}

	for name, config := range m.configs {
		snapshot.Configs[name] = m.deepCopy(config)
	}

	return snapshot
}

// RestoreSnapshot restores configurations from a snapshot
func (m *Manager) RestoreSnapshot(snapshot *ConfigSnapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, config := range snapshot.Configs {
		// Validate configuration if validator exists
		if validator, exists := m.validators[name]; exists {
			if err := validator.Validate(config); err != nil {
				return fmt.Errorf("config validation failed for %s during restore: %w", name, err)
			}
		}

		m.configs[name] = config
	}

	return nil
}