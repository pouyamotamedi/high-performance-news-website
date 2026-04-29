package config

import (
	"fmt"
	"os"
)

// EmailConfig holds email service configuration
type EmailConfig struct {
	Provider    string `yaml:"provider" env:"EMAIL_PROVIDER"`
	SendGrid    SendGridConfig `yaml:"sendgrid"`
	Mailgun     MailgunConfig `yaml:"mailgun"`
	FromEmail   string `yaml:"from_email" env:"EMAIL_FROM"`
	FromName    string `yaml:"from_name" env:"EMAIL_FROM_NAME"`
	ReplyTo     string `yaml:"reply_to" env:"EMAIL_REPLY_TO"`
	BaseURL     string `yaml:"base_url" env:"BASE_URL"`
	RateLimit   int    `yaml:"rate_limit" env:"EMAIL_RATE_LIMIT"`
	BatchSize   int    `yaml:"batch_size" env:"EMAIL_BATCH_SIZE"`
	MaxRetries  int    `yaml:"max_retries" env:"EMAIL_MAX_RETRIES"`
	Enabled     bool   `yaml:"enabled" env:"EMAIL_ENABLED"`
}

// SendGridConfig holds SendGrid specific configuration
type SendGridConfig struct {
	APIKey    string `yaml:"api_key" env:"SENDGRID_API_KEY"`
	Endpoint  string `yaml:"endpoint" env:"SENDGRID_ENDPOINT"`
}

// MailgunConfig holds Mailgun specific configuration
type MailgunConfig struct {
	APIKey    string `yaml:"api_key" env:"MAILGUN_API_KEY"`
	Domain    string `yaml:"domain" env:"MAILGUN_DOMAIN"`
	Endpoint  string `yaml:"endpoint" env:"MAILGUN_ENDPOINT"`
}

// LoadEmailConfig loads email configuration from environment variables
func LoadEmailConfig() *EmailConfig {
	config := &EmailConfig{
		Provider:   getEnv("EMAIL_PROVIDER", "sendgrid"),
		FromEmail:  getEnv("EMAIL_FROM", "noreply@example.com"),
		FromName:   getEnv("EMAIL_FROM_NAME", "News Website"),
		ReplyTo:    getEnv("EMAIL_REPLY_TO", "support@example.com"),
		BaseURL:    getEnv("BASE_URL", "http://localhost:8080"),
		RateLimit:  getEnvInt("EMAIL_RATE_LIMIT", 100000), // 100K emails per hour
		BatchSize:  getEnvInt("EMAIL_BATCH_SIZE", 1000),
		MaxRetries: getEnvInt("EMAIL_MAX_RETRIES", 3),
		Enabled:    getEnvBool("EMAIL_ENABLED", true),
		SendGrid: SendGridConfig{
			APIKey:   getEnv("SENDGRID_API_KEY", ""),
			Endpoint: getEnv("SENDGRID_ENDPOINT", "https://api.sendgrid.com/v3/mail/send"),
		},
		Mailgun: MailgunConfig{
			APIKey:   getEnv("MAILGUN_API_KEY", ""),
			Domain:   getEnv("MAILGUN_DOMAIN", ""),
			Endpoint: getEnv("MAILGUN_ENDPOINT", "https://api.mailgun.net/v3"),
		},
	}

	return config
}

// Validate validates the email configuration
func (c *EmailConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.FromEmail == "" {
		return fmt.Errorf("EMAIL_FROM is required")
	}

	if c.FromName == "" {
		return fmt.Errorf("EMAIL_FROM_NAME is required")
	}

	if c.BaseURL == "" {
		return fmt.Errorf("BASE_URL is required")
	}

	switch c.Provider {
	case "sendgrid":
		if c.SendGrid.APIKey == "" {
			return fmt.Errorf("SENDGRID_API_KEY is required when using SendGrid")
		}
	case "mailgun":
		if c.Mailgun.APIKey == "" {
			return fmt.Errorf("MAILGUN_API_KEY is required when using Mailgun")
		}
		if c.Mailgun.Domain == "" {
			return fmt.Errorf("MAILGUN_DOMAIN is required when using Mailgun")
		}
	default:
		return fmt.Errorf("unsupported email provider: %s", c.Provider)
	}

	return nil
}

// GetWebhookURL returns the webhook URL for the email provider
func (c *EmailConfig) GetWebhookURL(eventType string) string {
	return fmt.Sprintf("%s/api/v1/webhooks/email/%s/%s", c.BaseURL, c.Provider, eventType)
}

// GetUnsubscribeURL returns the unsubscribe URL
func (c *EmailConfig) GetUnsubscribeURL(token string) string {
	return fmt.Sprintf("%s/unsubscribe/%s", c.BaseURL, token)
}

// GetConfirmationURL returns the confirmation URL
func (c *EmailConfig) GetConfirmationURL(token string) string {
	return fmt.Sprintf("%s/confirm-subscription/%s", c.BaseURL, token)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

