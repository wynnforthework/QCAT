package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Auditor manages audit logging and permission tracking
type Auditor struct {
	// Prometheus metrics
	auditEventsTotal  prometheus.Counter
	auditEventsByType *prometheus.CounterVec
	auditEventsByUser *prometheus.CounterVec
	auditLatency      prometheus.Histogram

	// Configuration
	config *AuditConfig

	// Audit storage
	storage AuditStorage

	// Permission cache
	permissions map[string]*UserPermissions
	mu          sync.RWMutex

	// Channels
	eventCh chan *AuditEvent
	stopCh  chan struct{}
}

// AuditConfig represents audit configuration
type AuditConfig struct {
	Enabled            bool
	LogLevel           string
	RetentionDays      int
	BatchSize          int
	BatchTimeout       time.Duration
	AsyncMode          bool
	CompressionEnabled bool
	EncryptionEnabled  bool
}

// AuditEvent represents an audit event
type AuditEvent struct {
	ID         string                 `json:"id"`
	Timestamp  time.Time              `json:"timestamp"`
	UserID     string                 `json:"user_id"`
	Username   string                 `json:"username"`
	Action     string                 `json:"action"`
	Resource   string                 `json:"resource"`
	ResourceID string                 `json:"resource_id"`
	IPAddress  string                 `json:"ip_address"`
	UserAgent  string                 `json:"user_agent"`
	Status     AuditStatus            `json:"status"`
	Details    map[string]interface{} `json:"details"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// AuditStatus represents audit event status
type AuditStatus string

const (
	AuditStatusSuccess AuditStatus = "success"
	AuditStatusFailure AuditStatus = "failure"
	AuditStatusDenied  AuditStatus = "denied"
)

// UserPermissions represents user permissions
type UserPermissions struct {
	UserID      string                 `json:"user_id"`
	Username    string                 `json:"username"`
	Roles       []string               `json:"roles"`
	Permissions []Permission           `json:"permissions"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// Permission represents a permission
type Permission struct {
	Resource   string                 `json:"resource"`
	Actions    []string               `json:"actions"`
	Conditions map[string]interface{} `json:"conditions"`
}

// AuditStorage represents audit storage interface
type AuditStorage interface {
	Store(ctx context.Context, event *AuditEvent) error
	Query(ctx context.Context, filter *AuditFilter) ([]*AuditEvent, error)
	GetUserPermissions(ctx context.Context, userID string) (*UserPermissions, error)
	SetUserPermissions(ctx context.Context, permissions *UserPermissions) error
	Cleanup(ctx context.Context, before time.Time) error
}

// AuditFilter represents audit query filter
type AuditFilter struct {
	UserID    string
	Action    string
	Resource  string
	Status    AuditStatus
	StartTime time.Time
	EndTime   time.Time
	Limit     int
	Offset    int
}

// NewAuditor creates a new auditor
func NewAuditor(config *AuditConfig, storage AuditStorage) *Auditor {
	if config == nil {
		config = &AuditConfig{
			Enabled:            true,
			LogLevel:           "info",
			RetentionDays:      90,
			BatchSize:          100,
			BatchTimeout:       5 * time.Second,
			AsyncMode:          true,
			CompressionEnabled: true,
			EncryptionEnabled:  false,
		}
	}

	auditor := &Auditor{
		config:      config,
		storage:     storage,
		permissions: make(map[string]*UserPermissions),
		eventCh:     make(chan *AuditEvent, 1000),
		stopCh:      make(chan struct{}),
	}

	// Initialize Prometheus metrics
	auditor.initializeMetrics()

	return auditor
}

// initializeMetrics initializes Prometheus metrics
func (auditor *Auditor) initializeMetrics() {
	auditor.auditEventsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "audit_events_total",
		Help: "Total number of audit events",
	})

	auditor.auditEventsByType = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "audit_events_by_type_total",
		Help: "Number of audit events by type",
	}, []string{"action", "resource", "status"})

	auditor.auditEventsByUser = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "audit_events_by_user_total",
		Help: "Number of audit events by user",
	}, []string{"user_id", "username"})

	auditor.auditLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "audit_event_duration_seconds",
		Help:    "Audit event processing duration in seconds",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
	})
}

// Start starts the auditor
func (auditor *Auditor) Start() {
	if !auditor.config.Enabled {
		return
	}

	// Start event processing worker
	go auditor.eventWorker()

	// Start cleanup worker
	go auditor.cleanupWorker()
}

// Stop stops the auditor
func (auditor *Auditor) Stop() {
	close(auditor.stopCh)
}

// LogEvent logs an audit event
func (auditor *Auditor) LogEvent(ctx context.Context, event *AuditEvent) error {
	if !auditor.config.Enabled {
		return nil
	}

	// Set event ID and timestamp if not set
	if event.ID == "" {
		event.ID = generateEventID()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Update metrics
	auditor.auditEventsTotal.Inc()
	auditor.auditEventsByType.WithLabelValues(event.Action, event.Resource, string(event.Status)).Inc()
	auditor.auditEventsByUser.WithLabelValues(event.UserID, event.Username).Inc()

	// Send event to processing channel
	select {
	case auditor.eventCh <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("audit event queue is full")
	}
}

// CheckPermission checks if a user has permission to perform an action
func (auditor *Auditor) CheckPermission(ctx context.Context, userID, action, resource string) (bool, error) {
	auditor.mu.RLock()
	permissions, exists := auditor.permissions[userID]
	auditor.mu.RUnlock()

	if !exists {
		// Load permissions from storage
		var err error
		permissions, err = auditor.storage.GetUserPermissions(ctx, userID)
		if err != nil {
			return false, err
		}

		// Cache permissions
		auditor.mu.Lock()
		auditor.permissions[userID] = permissions
		auditor.mu.Unlock()
	}

	// Check permissions
	for _, permission := range permissions.Permissions {
		if permission.Resource == resource || permission.Resource == "*" {
			for _, permAction := range permission.Actions {
				if permAction == action || permAction == "*" {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// SetUserPermissions sets user permissions
func (auditor *Auditor) SetUserPermissions(ctx context.Context, permissions *UserPermissions) error {
	// Update storage
	if err := auditor.storage.SetUserPermissions(ctx, permissions); err != nil {
		return err
	}

	// Update cache
	auditor.mu.Lock()
	auditor.permissions[permissions.UserID] = permissions
	auditor.mu.Unlock()

	return nil
}

// GetUserPermissions gets user permissions
func (auditor *Auditor) GetUserPermissions(ctx context.Context, userID string) (*UserPermissions, error) {
	auditor.mu.RLock()
	permissions, exists := auditor.permissions[userID]
	auditor.mu.RUnlock()

	if exists {
		return permissions, nil
	}

	// Load from storage
	permissions, err := auditor.storage.GetUserPermissions(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Cache permissions
	auditor.mu.Lock()
	auditor.permissions[userID] = permissions
	auditor.mu.Unlock()

	return permissions, nil
}

// QueryEvents queries audit events
func (auditor *Auditor) QueryEvents(ctx context.Context, filter *AuditFilter) ([]*AuditEvent, error) {
	return auditor.storage.Query(ctx, filter)
}

// eventWorker processes audit events
func (auditor *Auditor) eventWorker() {
	var events []*AuditEvent
	ticker := time.NewTicker(auditor.config.BatchTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-auditor.stopCh:
			// Process remaining events
			if len(events) > 0 {
				auditor.processEvents(events)
			}
			return
		case event := <-auditor.eventCh:
			events = append(events, event)
			if len(events) >= auditor.config.BatchSize {
				auditor.processEvents(events)
				events = events[:0]
			}
		case <-ticker.C:
			if len(events) > 0 {
				auditor.processEvents(events)
				events = events[:0]
			}
		}
	}
}

// processEvents processes a batch of audit events
func (auditor *Auditor) processEvents(events []*AuditEvent) {
	start := time.Now()

	for _, event := range events {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := auditor.storage.Store(ctx, event); err != nil {
			// Log error but don't fail the entire batch
			fmt.Printf("Failed to store audit event: %v\n", err)
		}
		cancel()
	}

	duration := time.Since(start)
	auditor.auditLatency.Observe(duration.Seconds())
}

// cleanupWorker cleans up old audit events
func (auditor *Auditor) cleanupWorker() {
	ticker := time.NewTicker(24 * time.Hour) // Run daily
	defer ticker.Stop()

	for {
		select {
		case <-auditor.stopCh:
			return
		case <-ticker.C:
			auditor.cleanupOldEvents()
		}
	}
}

// cleanupOldEvents removes old audit events
func (auditor *Auditor) cleanupOldEvents() {
	cutoff := time.Now().AddDate(0, 0, -auditor.config.RetentionDays)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if err := auditor.storage.Cleanup(ctx, cutoff); err != nil {
		fmt.Printf("Failed to cleanup old audit events: %v\n", err)
	}
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return fmt.Sprintf("audit_%d", time.Now().UnixNano())
}

// AuditMiddleware creates audit middleware for HTTP requests
func (auditor *Auditor) AuditMiddleware(action string) func(ctx context.Context, userID, username, resource, resourceID, ipAddress, userAgent string, status AuditStatus, details map[string]interface{}) {
	return func(ctx context.Context, userID, username, resource, resourceID, ipAddress, userAgent string, status AuditStatus, details map[string]interface{}) {
		event := &AuditEvent{
			UserID:     userID,
			Username:   username,
			Action:     action,
			Resource:   resource,
			ResourceID: resourceID,
			IPAddress:  ipAddress,
			UserAgent:  userAgent,
			Status:     status,
			Details:    details,
			Metadata: map[string]interface{}{
				"timestamp": time.Now(),
			},
		}

		auditor.LogEvent(ctx, event)
	}
}

// PermissionMiddleware creates permission checking middleware
func (auditor *Auditor) PermissionMiddleware(action, resource string) func(ctx context.Context, userID string) (bool, error) {
	return func(ctx context.Context, userID string) (bool, error) {
		return auditor.CheckPermission(ctx, userID, action, resource)
	}
}

// GetAuditSummary returns audit summary statistics
func (auditor *Auditor) GetAuditSummary(ctx context.Context, days int) (map[string]interface{}, error) {
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -days)

	filter := &AuditFilter{
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     10000, // Large limit to get all events
	}

	events, err := auditor.QueryEvents(ctx, filter)
	if err != nil {
		return nil, err
	}

	summary := map[string]interface{}{
		"total_events":     len(events),
		"period_days":      days,
		"start_time":       startTime,
		"end_time":         endTime,
		"events_by_action": make(map[string]int),
		"events_by_user":   make(map[string]int),
		"events_by_status": make(map[string]int),
	}

	for _, event := range events {
		// Count by action
		summary["events_by_action"].(map[string]int)[event.Action]++

		// Count by user
		summary["events_by_user"].(map[string]int)[event.Username]++

		// Count by status
		summary["events_by_status"].(map[string]int)[string(event.Status)]++
	}

	return summary, nil
}

// ExportAuditLog exports audit events to JSON
func (auditor *Auditor) ExportAuditLog(ctx context.Context, filter *AuditFilter) ([]byte, error) {
	events, err := auditor.QueryEvents(ctx, filter)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(events, "", "  ")
}

// ClearUserPermissions clears user permissions cache
func (auditor *Auditor) ClearUserPermissions(userID string) {
	auditor.mu.Lock()
	delete(auditor.permissions, userID)
	auditor.mu.Unlock()
}

// ClearAllPermissions clears all permissions cache
func (auditor *Auditor) ClearAllPermissions() {
	auditor.mu.Lock()
	auditor.permissions = make(map[string]*UserPermissions)
	auditor.mu.Unlock()
}
