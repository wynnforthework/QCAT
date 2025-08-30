package protector

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"
)

// MockNotificationService 模拟通知服务（用于测试）
type MockNotificationService struct {
	emailsSent    []EmailRecord
	smsSent       []SMSRecord
	webhooksSent  []WebhookRecord
	slackSent     []SlackRecord
	shouldFail    bool
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

// TestMockNotificationService 测试mock通知服务
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
