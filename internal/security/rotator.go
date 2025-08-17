package security

import (
	"time"
)

// KeyRotator handles automatic key rotation
type KeyRotator struct {
	config *RotationConfig
}

// RotationConfig represents rotation configuration
type RotationConfig struct {
	AutoRotate              bool          `json:"auto_rotate"`
	CheckInterval           time.Duration `json:"check_interval"`
	MinRotationInterval     time.Duration `json:"min_rotation_interval"`
	MaxKeyAge               time.Duration `json:"max_key_age"`
	RotateBeforeExpiry      time.Duration `json:"rotate_before_expiry"`
	MaxUsageBeforeRotation  int64         `json:"max_usage_before_rotation"`
	RotateOnSuspiciousActivity bool       `json:"rotate_on_suspicious_activity"`
	NotifyBeforeRotation    time.Duration `json:"notify_before_rotation"`
	GracePeriodAfterRotation time.Duration `json:"grace_period_after_rotation"`
}

// NewKeyRotator creates a new key rotator
func NewKeyRotator(config *RotationConfig) *KeyRotator {
	if config == nil {
		config = DefaultRotationConfig()
	}

	return &KeyRotator{
		config: config,
	}
}

// ShouldRotate determines if a key should be rotated based on the configuration
func (kr *KeyRotator) ShouldRotate(keyInfo *KeyInfo) bool {
	now := time.Now()

	// Check minimum rotation interval
	if now.Sub(keyInfo.UpdatedAt) < kr.config.MinRotationInterval {
		return false
	}

	// Check if key is approaching expiration
	if !keyInfo.ExpiresAt.IsZero() {
		timeToExpiry := keyInfo.ExpiresAt.Sub(now)
		if timeToExpiry < kr.config.RotateBeforeExpiry {
			return true
		}
	}

	// Check if key has exceeded maximum age
	if now.Sub(keyInfo.CreatedAt) > kr.config.MaxKeyAge {
		return true
	}

	// Check if key has been used too much
	if keyInfo.UsageCount > kr.config.MaxUsageBeforeRotation {
		return true
	}

	return false
}

// GetRotationSchedule returns when a key should be rotated
func (kr *KeyRotator) GetRotationSchedule(keyInfo *KeyInfo) *RotationSchedule {
	now := time.Now()
	schedule := &RotationSchedule{
		KeyID:     keyInfo.ID,
		CurrentAge: now.Sub(keyInfo.CreatedAt),
		LastRotation: keyInfo.UpdatedAt,
	}

	// Calculate next rotation based on various factors
	var nextRotation time.Time

	// Based on expiration
	if !keyInfo.ExpiresAt.IsZero() {
		expiryBasedRotation := keyInfo.ExpiresAt.Add(-kr.config.RotateBeforeExpiry)
		if nextRotation.IsZero() || expiryBasedRotation.Before(nextRotation) {
			nextRotation = expiryBasedRotation
			schedule.Reason = "approaching_expiry"
		}
	}

	// Based on maximum age
	ageBasedRotation := keyInfo.CreatedAt.Add(kr.config.MaxKeyAge)
	if nextRotation.IsZero() || ageBasedRotation.Before(nextRotation) {
		nextRotation = ageBasedRotation
		schedule.Reason = "max_age_reached"
	}

	// Ensure minimum rotation interval
	minNextRotation := keyInfo.UpdatedAt.Add(kr.config.MinRotationInterval)
	if nextRotation.Before(minNextRotation) {
		nextRotation = minNextRotation
	}

	schedule.NextRotation = nextRotation
	schedule.TimeUntilRotation = nextRotation.Sub(now)

	// Check if notification should be sent
	if kr.config.NotifyBeforeRotation > 0 {
		schedule.NotificationTime = nextRotation.Add(-kr.config.NotifyBeforeRotation)
		schedule.ShouldNotify = now.After(schedule.NotificationTime) && now.Before(nextRotation)
	}

	return schedule
}

// DefaultRotationConfig returns default rotation configuration
func DefaultRotationConfig() *RotationConfig {
	return &RotationConfig{
		AutoRotate:                 true,
		CheckInterval:              time.Hour,
		MinRotationInterval:        24 * time.Hour,        // Don't rotate more than once per day
		MaxKeyAge:                  30 * 24 * time.Hour,   // Rotate after 30 days
		RotateBeforeExpiry:         7 * 24 * time.Hour,    // Rotate 7 days before expiry
		MaxUsageBeforeRotation:     100000,                // Rotate after 100k uses
		RotateOnSuspiciousActivity: true,
		NotifyBeforeRotation:       24 * time.Hour,        // Notify 24 hours before rotation
		GracePeriodAfterRotation:   time.Hour,             // 1 hour grace period for old key
	}
}

// RotationSchedule represents the rotation schedule for a key
type RotationSchedule struct {
	KeyID               string        `json:"key_id"`
	CurrentAge          time.Duration `json:"current_age"`
	LastRotation        time.Time     `json:"last_rotation"`
	NextRotation        time.Time     `json:"next_rotation"`
	TimeUntilRotation   time.Duration `json:"time_until_rotation"`
	Reason              string        `json:"reason"`
	NotificationTime    time.Time     `json:"notification_time,omitempty"`
	ShouldNotify        bool          `json:"should_notify"`
}