package alerting

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// AlertManager manages alert channels and notifications
type AlertManager struct {
	// Prometheus metrics
	alertSent     prometheus.Counter
	alertFailed   prometheus.Counter
	alertDuration prometheus.Histogram

	// Configuration
	config *AlertConfig

	// Alert channels
	channels map[string]AlertChannel
	mu       sync.RWMutex

	// Channels
	alertCh chan *Alert
	stopCh  chan struct{}
}

// AlertConfig represents alert configuration
type AlertConfig struct {
	DefaultChannels []string
	RetryCount      int
	RetryInterval   time.Duration
	Timeout         time.Duration
	RateLimit       int
	RateLimitWindow time.Duration
}

// Alert represents an alert
type Alert struct {
	ID         string
	Level      AlertLevel
	Title      string
	Message    string
	Source     string
	Timestamp  time.Time
	Channels   []string
	Metadata   map[string]interface{}
	RetryCount int
}

// AlertLevel represents alert level
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelCritical AlertLevel = "critical"
	AlertLevelError    AlertLevel = "error"
)

// AlertChannel represents an alert channel
type AlertChannel interface {
	Send(ctx context.Context, alert *Alert) error
	GetName() string
	IsEnabled() bool
}

// EmailChannel represents email alert channel
type EmailChannel struct {
	config *EmailConfig
	client *http.Client
}

// EmailConfig represents email configuration
type EmailConfig struct {
	Enabled  bool
	SMTPHost string
	SMTPPort int
	Username string
	Password string
	From     string
	To       []string
	Subject  string
	Template string
	UseTLS   bool
	Timeout  time.Duration
}

// SMSChannel represents SMS alert channel
type SMSChannel struct {
	config *SMSConfig
	client *http.Client
}

// SMSConfig represents SMS configuration
type SMSConfig struct {
	Enabled   bool
	Provider  string
	APIKey    string
	APISecret string
	Endpoint  string
	From      string
	To        []string
	Template  string
	Timeout   time.Duration
}

// DingTalkChannel represents DingTalk alert channel
type DingTalkChannel struct {
	config *DingTalkConfig
	client *http.Client
}

// DingTalkConfig represents DingTalk configuration
type DingTalkConfig struct {
	Enabled    bool
	WebhookURL string
	Secret     string
	Template   string
	Timeout    time.Duration
}

// SlackChannel represents Slack alert channel
type SlackChannel struct {
	config *SlackConfig
	client *http.Client
}

// SlackConfig represents Slack configuration
type SlackConfig struct {
	Enabled    bool
	WebhookURL string
	Channel    string
	Username   string
	IconEmoji  string
	Template   string
	Timeout    time.Duration
}

// NewAlertManager creates a new alert manager
func NewAlertManager(config *AlertConfig) *AlertManager {
	if config == nil {
		config = &AlertConfig{
			DefaultChannels: []string{"email"},
			RetryCount:      3,
			RetryInterval:   30 * time.Second,
			Timeout:         10 * time.Second,
			RateLimit:       100,
			RateLimitWindow: 1 * time.Minute,
		}
	}

	am := &AlertManager{
		config:   config,
		channels: make(map[string]AlertChannel),
		alertCh:  make(chan *Alert, 100),
		stopCh:   make(chan struct{}),
	}

	// Initialize Prometheus metrics
	am.initializeMetrics()

	return am
}

// initializeMetrics initializes Prometheus metrics
func (am *AlertManager) initializeMetrics() {
	am.alertSent = promauto.NewCounter(prometheus.CounterOpts{
		Name: "alert_sent_total",
		Help: "Total number of alerts sent",
	})

	am.alertFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "alert_failed_total",
		Help: "Total number of alert failures",
	})

	am.alertDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "alert_duration_seconds",
		Help:    "Alert sending duration in seconds",
		Buckets: prometheus.ExponentialBuckets(0.1, 2, 10),
	})
}

// Start starts the alert manager
func (am *AlertManager) Start() {
	go am.alertWorker()
}

// Stop stops the alert manager
func (am *AlertManager) Stop() {
	close(am.stopCh)
}

// RegisterChannel registers an alert channel
func (am *AlertManager) RegisterChannel(channel AlertChannel) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.channels[channel.GetName()] = channel
}

// SendAlert sends an alert
func (am *AlertManager) SendAlert(ctx context.Context, level AlertLevel, title, message, source string, channels []string, metadata map[string]interface{}) error {
	if len(channels) == 0 {
		channels = am.config.DefaultChannels
	}

	alert := &Alert{
		ID:        generateAlertID(),
		Level:     level,
		Title:     title,
		Message:   message,
		Source:    source,
		Timestamp: time.Now(),
		Channels:  channels,
		Metadata:  metadata,
	}

	select {
	case am.alertCh <- alert:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("alert queue is full")
	}
}

// alertWorker processes alerts
func (am *AlertManager) alertWorker() {
	for {
		select {
		case <-am.stopCh:
			return
		case alert := <-am.alertCh:
			am.processAlert(alert)
		}
	}
}

// processAlert processes a single alert
func (am *AlertManager) processAlert(alert *Alert) {
	start := time.Now()

	for _, channelName := range alert.Channels {
		am.mu.RLock()
		channel, exists := am.channels[channelName]
		am.mu.RUnlock()

		if !exists {
			fmt.Printf("Alert channel %s not found\n", channelName)
			continue
		}

		if !channel.IsEnabled() {
			continue
		}

		// Send alert with retry
		for i := 0; i <= am.config.RetryCount; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), am.config.Timeout)
			err := channel.Send(ctx, alert)
			cancel()

			if err == nil {
				am.alertSent.Inc()
				break
			}

			if i < am.config.RetryCount {
				time.Sleep(am.config.RetryInterval)
			} else {
				am.alertFailed.Inc()
				fmt.Printf("Failed to send alert via %s after %d retries: %v\n", channelName, am.config.RetryCount, err)
			}
		}
	}

	duration := time.Since(start)
	am.alertDuration.Observe(duration.Seconds())
}

// NewEmailChannel creates a new email alert channel
func NewEmailChannel(config *EmailConfig) *EmailChannel {
	return &EmailChannel{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Send sends an alert via email
func (ec *EmailChannel) Send(ctx context.Context, alert *Alert) error {
	if !ec.config.Enabled {
		return fmt.Errorf("email channel is disabled")
	}

	// Build email content
	subject := fmt.Sprintf("[%s] %s", strings.ToUpper(string(alert.Level)), alert.Title)
	body := ec.buildEmailBody(alert)

	// Send email
	auth := smtp.PlainAuth("", ec.config.Username, ec.config.Password, ec.config.SMTPHost)
	addr := fmt.Sprintf("%s:%d", ec.config.SMTPHost, ec.config.SMTPPort)

	to := strings.Join(ec.config.To, ",")
	msg := fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", to, subject, body)

	if ec.config.UseTLS {
		return smtp.SendMail(addr, auth, ec.config.From, ec.config.To, []byte(msg))
	}

	return smtp.SendMail(addr, auth, ec.config.From, ec.config.To, []byte(msg))
}

// GetName returns the channel name
func (ec *EmailChannel) GetName() string {
	return "email"
}

// IsEnabled returns whether the channel is enabled
func (ec *EmailChannel) IsEnabled() bool {
	return ec.config.Enabled
}

// buildEmailBody builds the email body
func (ec *EmailChannel) buildEmailBody(alert *Alert) string {
	if ec.config.Template != "" {
		// Use custom template
		return fmt.Sprintf(ec.config.Template, alert.Title, alert.Message, alert.Source, alert.Timestamp)
	}

	// Default template
	return fmt.Sprintf(`
Alert Details:
==============

Level: %s
Title: %s
Message: %s
Source: %s
Time: %s

Please take appropriate action.
`, alert.Level, alert.Title, alert.Message, alert.Source, alert.Timestamp.Format(time.RFC3339))
}

// NewSMSChannel creates a new SMS alert channel
func NewSMSChannel(config *SMSConfig) *SMSChannel {
	return &SMSChannel{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Send sends an alert via SMS
func (sc *SMSChannel) Send(ctx context.Context, alert *Alert) error {
	if !sc.config.Enabled {
		return fmt.Errorf("SMS channel is disabled")
	}

	// Build SMS content
	content := sc.buildSMSContent(alert)

	// Send SMS via API
	payload := map[string]interface{}{
		"api_key":    sc.config.APIKey,
		"api_secret": sc.config.APISecret,
		"from":       sc.config.From,
		"to":         sc.config.To,
		"message":    content,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", sc.config.Endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := sc.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SMS API returned status %d", resp.StatusCode)
	}

	return nil
}

// GetName returns the channel name
func (sc *SMSChannel) GetName() string {
	return "sms"
}

// IsEnabled returns whether the channel is enabled
func (sc *SMSChannel) IsEnabled() bool {
	return sc.config.Enabled
}

// buildSMSContent builds the SMS content
func (sc *SMSChannel) buildSMSContent(alert *Alert) string {
	if sc.config.Template != "" {
		return fmt.Sprintf(sc.config.Template, alert.Title, alert.Message)
	}

	return fmt.Sprintf("[%s] %s: %s", strings.ToUpper(string(alert.Level)), alert.Title, alert.Message)
}

// NewDingTalkChannel creates a new DingTalk alert channel
func NewDingTalkChannel(config *DingTalkConfig) *DingTalkChannel {
	return &DingTalkChannel{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Send sends an alert via DingTalk
func (dtc *DingTalkChannel) Send(ctx context.Context, alert *Alert) error {
	if !dtc.config.Enabled {
		return fmt.Errorf("DingTalk channel is disabled")
	}

	// Build DingTalk message
	message := dtc.buildDingTalkMessage(alert)

	// Add signature if secret is provided
	webhookURL := dtc.config.WebhookURL
	if dtc.config.Secret != "" {
		timestamp := time.Now().UnixNano() / 1e6
		sign := dtc.generateSignature(timestamp, dtc.config.Secret)
		webhookURL = fmt.Sprintf("%s&timestamp=%d&sign=%s", dtc.config.WebhookURL, timestamp, sign)
	}

	// Send message
	jsonData, err := json.Marshal(message)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := dtc.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("DingTalk API returned status %d", resp.StatusCode)
	}

	return nil
}

// GetName returns the channel name
func (dtc *DingTalkChannel) GetName() string {
	return "dingtalk"
}

// IsEnabled returns whether the channel is enabled
func (dtc *DingTalkChannel) IsEnabled() bool {
	return dtc.config.Enabled
}

// buildDingTalkMessage builds the DingTalk message
func (dtc *DingTalkChannel) buildDingTalkMessage(alert *Alert) map[string]interface{} {
	if dtc.config.Template != "" {
		// Use custom template
		content := fmt.Sprintf(dtc.config.Template, alert.Title, alert.Message, alert.Source, alert.Timestamp)
		return map[string]interface{}{
			"msgtype": "text",
			"text": map[string]string{
				"content": content,
			},
		}
	}

	// Default template
	content := fmt.Sprintf("[%s] %s\n\n%s\n\nSource: %s\nTime: %s",
		strings.ToUpper(string(alert.Level)), alert.Title, alert.Message, alert.Source, alert.Timestamp.Format(time.RFC3339))

	return map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": content,
		},
	}
}

// generateSignature generates DingTalk signature
func (dtc *DingTalkChannel) generateSignature(timestamp int64, secret string) string {
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
	hash := hmac.New(sha256.New, []byte(secret))
	hash.Write([]byte(stringToSign))
	return url.QueryEscape(base64.StdEncoding.EncodeToString(hash.Sum(nil)))
}

// NewSlackChannel creates a new Slack alert channel
func NewSlackChannel(config *SlackConfig) *SlackChannel {
	return &SlackChannel{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Send sends an alert via Slack
func (slc *SlackChannel) Send(ctx context.Context, alert *Alert) error {
	if !slc.config.Enabled {
		return fmt.Errorf("Slack channel is disabled")
	}

	// Build Slack message
	message := slc.buildSlackMessage(alert)

	// Send message
	jsonData, err := json.Marshal(message)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", slc.config.WebhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := slc.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack API returned status %d", resp.StatusCode)
	}

	return nil
}

// GetName returns the channel name
func (slc *SlackChannel) GetName() string {
	return "slack"
}

// IsEnabled returns whether the channel is enabled
func (slc *SlackChannel) IsEnabled() bool {
	return slc.config.Enabled
}

// buildSlackMessage builds the Slack message
func (slc *SlackChannel) buildSlackMessage(alert *Alert) map[string]interface{} {
	color := "#36a64f" // Green for info
	switch alert.Level {
	case AlertLevelWarning:
		color = "#ff9500" // Orange
	case AlertLevelCritical, AlertLevelError:
		color = "#ff0000" // Red
	}

	message := map[string]interface{}{
		"channel":    slc.config.Channel,
		"username":   slc.config.Username,
		"icon_emoji": slc.config.IconEmoji,
		"attachments": []map[string]interface{}{
			{
				"color": color,
				"title": alert.Title,
				"text":  alert.Message,
				"fields": []map[string]interface{}{
					{
						"title": "Level",
						"value": strings.ToUpper(string(alert.Level)),
						"short": true,
					},
					{
						"title": "Source",
						"value": alert.Source,
						"short": true,
					},
					{
						"title": "Time",
						"value": alert.Timestamp.Format(time.RFC3339),
						"short": false,
					},
				},
			},
		},
	}

	return message
}

// generateAlertID generates a unique alert ID
func generateAlertID() string {
	return fmt.Sprintf("alert_%d", time.Now().UnixNano())
}
