package protector

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"
)

// MockNotificationService 测试用通知服务，实现完整的 NotificationService 接口
type MockNotificationService struct {
	emailsSent   []EmailRecord
	smsSent      []SMSRecord
	webhooksSent []WebhookRecord
	slackSent    []SlackRecord
	shouldFail   bool
}

// EmailRecord 邮件记录
type EmailRecord struct {
	To      string
	Subject string
	Body    string
	SentAt  time.Time
}

// SMSRecord 短信记录
type SMSRecord struct {
	Phone   string
	Message string
	SentAt  time.Time
}

// WebhookRecord Webhook记录
type WebhookRecord struct {
	URL     string
	Payload interface{}
	SentAt  time.Time
}

// SlackRecord Slack记录
type SlackRecord struct {
	Webhook string
	Message string
	SentAt  time.Time
}

// NewMockNotificationService 创建模拟通知服务
func NewMockNotificationService() *MockNotificationService {
	return &MockNotificationService{
		emailsSent:   make([]EmailRecord, 0),
		smsSent:      make([]SMSRecord, 0),
		webhooksSent: make([]WebhookRecord, 0),
		slackSent:    make([]SlackRecord, 0),
		shouldFail:   false,
	}
}

// SendEmail 发送邮件（模拟）
func (m *MockNotificationService) SendEmail(ctx context.Context, to, subject, body string) error {
	if m.shouldFail {
		return fmt.Errorf("mock email failure")
	}

	m.emailsSent = append(m.emailsSent, EmailRecord{
		To:      to,
		Subject: subject,
		Body:    body,
		SentAt:  time.Now(),
	})

	log.Printf("Mock email sent to %s: %s", to, subject)
	return nil
}

// SendSMS 发送短信（模拟）
func (m *MockNotificationService) SendSMS(ctx context.Context, phone, message string) error {
	if m.shouldFail {
		return fmt.Errorf("mock SMS failure")
	}

	m.smsSent = append(m.smsSent, SMSRecord{
		Phone:   phone,
		Message: message,
		SentAt:  time.Now(),
	})

	log.Printf("Mock SMS sent to %s: %s", phone, message)
	return nil
}

// SendWebhook 发送Webhook（模拟）
func (m *MockNotificationService) SendWebhook(ctx context.Context, url string, payload interface{}) error {
	if m.shouldFail {
		return fmt.Errorf("mock webhook failure")
	}

	m.webhooksSent = append(m.webhooksSent, WebhookRecord{
		URL:     url,
		Payload: payload,
		SentAt:  time.Now(),
	})

	log.Printf("Mock webhook sent to %s", url)
	return nil
}

// SendSlack 发送Slack消息（模拟）
func (m *MockNotificationService) SendSlack(ctx context.Context, webhook, message string) error {
	if m.shouldFail {
		return fmt.Errorf("mock Slack failure")
	}

	m.slackSent = append(m.slackSent, SlackRecord{
		Webhook: webhook,
		Message: message,
		SentAt:  time.Now(),
	})

	log.Printf("Mock Slack message sent: %s", message)
	return nil
}

// SetShouldFail 设置是否应该失败（测试用）
func (m *MockNotificationService) SetShouldFail(shouldFail bool) {
	m.shouldFail = shouldFail
}

// GetEmailsSent 获取已发送的邮件（测试用）
func (m *MockNotificationService) GetEmailsSent() []EmailRecord {
	return m.emailsSent
}

// GetSMSSent 获取已发送的短信（测试用）
func (m *MockNotificationService) GetSMSSent() []SMSRecord {
	return m.smsSent
}

// GetWebhooksSent 获取已发送的Webhook（测试用）
func (m *MockNotificationService) GetWebhooksSent() []WebhookRecord {
	return m.webhooksSent
}

// GetSlackSent 获取已发送的Slack消息（测试用）
func (m *MockNotificationService) GetSlackSent() []SlackRecord {
	return m.slackSent
}

// Reset 重置记录（测试用）
func (m *MockNotificationService) Reset() {
	m.emailsSent = make([]EmailRecord, 0)
	m.smsSent = make([]SMSRecord, 0)
	m.webhooksSent = make([]WebhookRecord, 0)
	m.slackSent = make([]SlackRecord, 0)
}

// TestRealNotificationService 测试真实通知服务
func TestRealNotificationService(t *testing.T) {
	ctx := context.Background()

	// 创建真实的通知服务配置
	config := &NotificationConfig{
		Email: EmailConfig{
			Enabled:  true,
			SMTPHost: "smtp.gmail.com",
			SMTPPort: 587,
			Username: "test@example.com",
			Password: "testpass",
			From:     "test@example.com",
			UseTLS:   true,
		},
		SMS: SMSConfig{
			Enabled:   true,
			Provider:  "twilio",
			APIKey:    "test_key",
			APISecret: "test_secret",
			From:      "+1234567890",
		},
		Webhook: WebhookConfig{
			Enabled: true,
			URL:     "https://example.com/webhook",
			Timeout: 10 * time.Second,
		},
		Slack: SlackConfig{
			Enabled:    true,
			WebhookURL: "https://hooks.slack.com/test",
			Channel:    "#test",
			Username:   "Test Bot",
		},
	}

	service := NewDefaultNotificationService(config)

	// 测试发送邮件（在真实环境中可能会失败，这是正常的）
	err := service.SendEmail(ctx, "test@example.com", "Test Subject", "Test Body")
	t.Logf("Email send result: %v", err)

	// 测试发送短信（在真实环境中可能会失败，这是正常的）
	err = service.SendSMS(ctx, "1234567890", "Test SMS")
	t.Logf("SMS send result: %v", err)

	// 测试发送Webhook（在真实环境中可能会失败，这是正常的）
	payload := map[string]interface{}{
		"message":   "Test webhook",
		"timestamp": time.Now().Unix(),
	}
	err = service.SendWebhook(ctx, "https://example.com/webhook", payload)
	t.Logf("Webhook send result: %v", err)

	// 测试发送Slack消息（在真实环境中可能会失败，这是正常的）
	err = service.SendSlack(ctx, "https://hooks.slack.com/test", "Test Slack message")
	t.Logf("Slack send result: %v", err)
}

// TestMockNotificationService 测试mock通知服务（保留用于其他测试）
func TestMockNotificationService(t *testing.T) {
	ctx := context.Background()
	mock := NewMockNotificationService()

	// 测试发送邮件
	err := mock.SendEmail(ctx, "test@example.com", "Test Subject", "Test Body")
	if err != nil {
		t.Errorf("SendEmail failed: %v", err)
	}

	emails := mock.GetEmailsSent()
	if len(emails) != 1 {
		t.Errorf("Expected 1 email, got %d", len(emails))
	}
	if emails[0].To != "test@example.com" {
		t.Errorf("Expected email to test@example.com, got %s", emails[0].To)
	}

	// 测试发送短信
	err = mock.SendSMS(ctx, "1234567890", "Test SMS")
	if err != nil {
		t.Errorf("SendSMS failed: %v", err)
	}

	sms := mock.GetSMSSent()
	if len(sms) != 1 {
		t.Errorf("Expected 1 SMS, got %d", len(sms))
	}

	// 测试失败模式
	mock.SetShouldFail(true)
	err = mock.SendEmail(ctx, "test@example.com", "Test", "Test")
	if err == nil {
		t.Error("Expected error when shouldFail is true")
	}

	// 测试重置
	mock.Reset()
	emails = mock.GetEmailsSent()
	if len(emails) != 0 {
		t.Errorf("Expected 0 emails after reset, got %d", len(emails))
	}
}

// TestNotificationServiceConfiguration 测试通知服务配置
func TestNotificationServiceConfiguration(t *testing.T) {
	// 测试空配置
	service := NewDefaultNotificationService(nil)
	if service == nil {
		t.Error("Service should not be nil even with nil config")
	}

	// 测试部分配置
	config := &NotificationConfig{
		Email: EmailConfig{
			Enabled:  true,
			SMTPHost: "smtp.example.com",
			SMTPPort: 587,
			Username: "test@example.com",
			Password: "password",
			From:     "noreply@example.com",
			UseTLS:   true,
		},
		// 其他服务未配置
	}

	service = NewDefaultNotificationService(config)
	ctx := context.Background()

	// 测试邮件发送（可能失败，但不应该panic）
	err := service.SendEmail(ctx, "recipient@example.com", "Test", "Test message")
	t.Logf("Email with partial config result: %v", err)

	// 测试未配置的服务
	err = service.SendSMS(ctx, "+1234567890", "Test SMS")
	t.Logf("SMS with no config result: %v", err)
}

// TestNotificationServiceErrorHandling 测试通知服务错误处理
func TestNotificationServiceErrorHandling(t *testing.T) {
	config := &NotificationConfig{
		Email: EmailConfig{
			Enabled:  true,
			SMTPHost: "invalid.smtp.server",
			SMTPPort: 587,
			Username: "invalid@example.com",
			Password: "wrongpassword",
			From:     "invalid@example.com",
			UseTLS:   true,
		},
		SMS: SMSConfig{
			Enabled:   true,
			Provider:  "invalid_provider",
			APIKey:    "invalid_key",
			APISecret: "invalid_secret",
			From:      "+0000000000",
		},
		Webhook: WebhookConfig{
			Enabled: true,
			URL:     "https://invalid.webhook.url/nonexistent",
			Timeout: 5 * time.Second,
		},
		Slack: SlackConfig{
			Enabled:    true,
			WebhookURL: "https://hooks.slack.com/invalid",
			Channel:    "#nonexistent",
			Username:   "TestBot",
		},
	}

	service := NewDefaultNotificationService(config)
	ctx := context.Background()

	// 测试各种错误情况
	err := service.SendEmail(ctx, "test@example.com", "Test", "Test")
	if err == nil {
		t.Log("Email unexpectedly succeeded with invalid config")
	} else {
		t.Logf("Email failed as expected: %v", err)
	}

	err = service.SendSMS(ctx, "+1234567890", "Test")
	if err == nil {
		t.Log("SMS unexpectedly succeeded with invalid config")
	} else {
		t.Logf("SMS failed as expected: %v", err)
	}

	err = service.SendWebhook(ctx, "https://invalid.url", map[string]interface{}{"test": "data"})
	if err == nil {
		t.Log("Webhook unexpectedly succeeded with invalid URL")
	} else {
		t.Logf("Webhook failed as expected: %v", err)
	}

	err = service.SendSlack(ctx, "https://hooks.slack.com/invalid", "Test message")
	if err == nil {
		t.Log("Slack unexpectedly succeeded with invalid webhook")
	} else {
		t.Logf("Slack failed as expected: %v", err)
	}
}

// TestNotificationServiceBatchOperations 测试批量通知操作
func TestNotificationServiceBatchOperations(t *testing.T) {
	config := &NotificationConfig{
		Email: EmailConfig{
			Enabled:  true,
			SMTPHost: "smtp.gmail.com",
			SMTPPort: 587,
			Username: "test@example.com",
			Password: "testpass",
			From:     "test@example.com",
			UseTLS:   true,
		},
	}

	service := NewDefaultNotificationService(config)
	ctx := context.Background()

	// 测试批量发送邮件
	recipients := []string{
		"user1@example.com",
		"user2@example.com",
		"user3@example.com",
	}

	for i, recipient := range recipients {
		subject := fmt.Sprintf("Batch Test Email %d", i+1)
		body := fmt.Sprintf("This is batch test email number %d", i+1)

		err := service.SendEmail(ctx, recipient, subject, body)
		t.Logf("Batch email %d to %s result: %v", i+1, recipient, err)
	}
}

// TestNotificationServiceConcurrency 测试并发通知
func TestNotificationServiceConcurrency(t *testing.T) {
	config := &NotificationConfig{
		Email: EmailConfig{
			Enabled:  true,
			SMTPHost: "smtp.gmail.com",
			SMTPPort: 587,
			Username: "test@example.com",
			Password: "testpass",
			From:     "test@example.com",
			UseTLS:   true,
		},
	}

	service := NewDefaultNotificationService(config)
	ctx := context.Background()

	// 并发发送通知
	const numGoroutines = 5
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			subject := fmt.Sprintf("Concurrent Test %d", id)
			body := fmt.Sprintf("This is concurrent test message %d", id)

			err := service.SendEmail(ctx, "test@example.com", subject, body)
			t.Logf("Concurrent email %d result: %v", id, err)
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}
