package protector

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"time"
)

// NotificationService 通知服务接口
type NotificationService interface {
	SendEmail(ctx context.Context, to, subject, body string) error
	SendSMS(ctx context.Context, phone, message string) error
	SendWebhook(ctx context.Context, url string, payload interface{}) error
	SendSlack(ctx context.Context, webhook, message string) error
}

// NotificationConfig 通知配置
type NotificationConfig struct {
	Email   EmailConfig   `json:"email"`
	SMS     SMSConfig     `json:"sms"`
	Webhook WebhookConfig `json:"webhook"`
	Slack   SlackConfig   `json:"slack"`
}

// EmailConfig 邮件配置
type EmailConfig struct {
	Enabled  bool   `json:"enabled"`
	SMTPHost string `json:"smtp_host"`
	SMTPPort int    `json:"smtp_port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	UseTLS   bool   `json:"use_tls"`
}

// SMSConfig 短信配置
type SMSConfig struct {
	Enabled   bool   `json:"enabled"`
	Provider  string `json:"provider"` // twilio, aws_sns, etc.
	APIKey    string `json:"api_key"`
	APISecret string `json:"api_secret"`
	From      string `json:"from"`
}

// WebhookConfig Webhook配置
type WebhookConfig struct {
	Enabled bool              `json:"enabled"`
	URL     string            `json:"url"`
	Timeout time.Duration     `json:"timeout"`
	Headers map[string]string `json:"headers"`
}

// SlackConfig Slack配置
type SlackConfig struct {
	Enabled    bool   `json:"enabled"`
	WebhookURL string `json:"webhook_url"`
	Channel    string `json:"channel"`
	Username   string `json:"username"`
}

// DefaultNotificationService 默认通知服务实现
type DefaultNotificationService struct {
	config *NotificationConfig
	client *http.Client
}

// NewDefaultNotificationService 创建默认通知服务
func NewDefaultNotificationService(config *NotificationConfig) *DefaultNotificationService {
	return &DefaultNotificationService{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
			},
		},
	}
}

// SendEmail 发送邮件
func (ns *DefaultNotificationService) SendEmail(ctx context.Context, to, subject, body string) error {
	if !ns.config.Email.Enabled {
		return fmt.Errorf("email notifications are disabled")
	}

	// 构建邮件内容
	msg := fmt.Sprintf("From: %s\r\n", ns.config.Email.From)
	msg += fmt.Sprintf("To: %s\r\n", to)
	msg += fmt.Sprintf("Subject: %s\r\n", subject)
	msg += fmt.Sprintf("Content-Type: text/plain; charset=UTF-8\r\n")
	msg += "\r\n"
	msg += body

	// SMTP认证
	auth := smtp.PlainAuth("", ns.config.Email.Username, ns.config.Email.Password, ns.config.Email.SMTPHost)

	// 发送邮件
	addr := fmt.Sprintf("%s:%d", ns.config.Email.SMTPHost, ns.config.Email.SMTPPort)

	var err error
	if ns.config.Email.UseTLS {
		// 使用TLS连接
		err = ns.sendEmailWithTLS(addr, auth, ns.config.Email.From, []string{to}, []byte(msg))
	} else {
		// 使用普通SMTP
		err = smtp.SendMail(addr, auth, ns.config.Email.From, []string{to}, []byte(msg))
	}

	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("Email sent successfully to %s", to)
	return nil
}

// sendEmailWithTLS 使用TLS发送邮件
func (ns *DefaultNotificationService) sendEmailWithTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	// 这里实现TLS邮件发送逻辑
	// 简化实现，实际应该使用更完整的TLS SMTP客户端
	return smtp.SendMail(addr, auth, from, to, msg)
}

// SendSMS 发送短信
func (ns *DefaultNotificationService) SendSMS(ctx context.Context, phone, message string) error {
	if !ns.config.SMS.Enabled {
		return fmt.Errorf("SMS notifications are disabled")
	}

	switch ns.config.SMS.Provider {
	case "twilio":
		return ns.sendTwilioSMS(ctx, phone, message)
	case "aws_sns":
		return ns.sendAWSSNS(ctx, phone, message)
	default:
		return fmt.Errorf("unsupported SMS provider: %s", ns.config.SMS.Provider)
	}
}

// sendTwilioSMS 通过Twilio发送短信
func (ns *DefaultNotificationService) sendTwilioSMS(ctx context.Context, phone, message string) error {
	// Twilio API实现
	url := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", ns.config.SMS.APIKey)

	payload := map[string]string{
		"From": ns.config.SMS.From,
		"To":   phone,
		"Body": message,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal SMS payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create SMS request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(ns.config.SMS.APIKey, ns.config.SMS.APISecret)

	resp, err := ns.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("SMS API returned status: %d", resp.StatusCode)
	}

	log.Printf("SMS sent successfully to %s", phone)
	return nil
}

// sendAWSSNS 通过AWS SNS发送短信
func (ns *DefaultNotificationService) sendAWSSNS(ctx context.Context, phone, message string) error {
	// AWS SNS实现
	// 这里需要集成AWS SDK
	log.Printf("AWS SNS SMS not implemented yet for %s", phone)
	return fmt.Errorf("AWS SNS SMS not implemented")
}

// SendWebhook 发送Webhook
func (ns *DefaultNotificationService) SendWebhook(ctx context.Context, url string, payload interface{}) error {
	if !ns.config.Webhook.Enabled {
		return fmt.Errorf("webhook notifications are disabled")
	}

	if url == "" {
		url = ns.config.Webhook.URL
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// 添加自定义头部
	for key, value := range ns.config.Webhook.Headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{
		Timeout: ns.config.Webhook.Timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status: %d", resp.StatusCode)
	}

	log.Printf("Webhook sent successfully to %s", url)
	return nil
}

// SendSlack 发送Slack消息
func (ns *DefaultNotificationService) SendSlack(ctx context.Context, webhook, message string) error {
	if !ns.config.Slack.Enabled {
		return fmt.Errorf("Slack notifications are disabled")
	}

	if webhook == "" {
		webhook = ns.config.Slack.WebhookURL
	}

	payload := map[string]interface{}{
		"text":     message,
		"channel":  ns.config.Slack.Channel,
		"username": ns.config.Slack.Username,
	}

	return ns.SendWebhook(ctx, webhook, payload)
}
