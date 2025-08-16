package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// ConfigWatcher watches for configuration file changes and reloads automatically
type ConfigWatcher struct {
	configPath     string
	checkInterval  time.Duration
	lastModTime    time.Time
	callbacks      []ConfigUpdateCallback
	mu             sync.RWMutex
	running        bool
}

// ConfigUpdateCallback represents a callback function for configuration updates
type ConfigUpdateCallback func(*AlgorithmConfig) error

// NewConfigWatcher creates a new configuration watcher
func NewConfigWatcher(configPath string, checkInterval time.Duration) *ConfigWatcher {
	return &ConfigWatcher{
		configPath:    configPath,
		checkInterval: checkInterval,
		callbacks:     make([]ConfigUpdateCallback, 0),
	}
}

// AddCallback adds a callback for configuration updates
func (w *ConfigWatcher) AddCallback(callback ConfigUpdateCallback) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.callbacks = append(w.callbacks, callback)
}

// Start starts watching for configuration changes
func (w *ConfigWatcher) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return nil
	}
	w.running = true
	w.mu.Unlock()

	log.Printf("Starting configuration watcher for %s", w.configPath)

	ticker := time.NewTicker(w.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.mu.Lock()
			w.running = false
			w.mu.Unlock()
			log.Println("Configuration watcher stopped")
			return ctx.Err()

		case <-ticker.C:
			if err := w.checkAndReload(); err != nil {
				log.Printf("Error checking configuration: %v", err)
			}
		}
	}
}

// checkAndReload checks if configuration file has changed and reloads if necessary
func (w *ConfigWatcher) checkAndReload() error {
	// Check if file has been modified
	stat, err := os.Stat(w.configPath)
	if err != nil {
		return fmt.Errorf("failed to stat config file: %w", err)
	}

	modTime := stat.ModTime()
	if !modTime.After(w.lastModTime) {
		return nil // No changes
	}

	log.Printf("Configuration file changed, reloading...")

	// Reload configuration
	newConfig, err := LoadAlgorithmConfig(w.configPath)
	if err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}

	// Validate new configuration
	if err := newConfig.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Update last modification time
	w.lastModTime = modTime

	// Notify callbacks
	w.mu.RLock()
	callbacks := make([]ConfigUpdateCallback, len(w.callbacks))
	copy(callbacks, w.callbacks)
	w.mu.RUnlock()

	for _, callback := range callbacks {
		if err := callback(newConfig); err != nil {
			log.Printf("Configuration update callback error: %v", err)
		}
	}

	log.Printf("Configuration reloaded successfully")
	return nil
}

// Stop stops the configuration watcher
func (w *ConfigWatcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.running = false
}

// IsRunning returns whether the watcher is currently running
func (w *ConfigWatcher) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.running
}