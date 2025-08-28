package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"high-performance-news-website/internal/config"
)

// MailgunProvider implements EmailProvider for Mailgun
type MailgunProvider struct {
	config     *config.EmailConfig
	httpClient *http.Client
	baseURL    string
}

// MailgunResponse represents Mailgun API response
type MailgunResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

// MailgunError represents Mailgun API error
type MailgunError struct {
	Message string `json:"message"`
}

// NewMailgunProvider creates a new Mailgun provider
func NewMailgunProvider(config *config.EmailConfig) (*MailgunProvider, error) {
	if config.Mailgun.APIKey == "" {
		return nil, fmt.Errorf("Mailgun API key is required")
	}
	if config.Mailgun.Domain == "" {
		return nil, fmt.Errorf("Mailgun domain is required")
	}

	baseURL := fmt.Sprintf("%s/%s", config.Mailgun.Endpoint, config.Mailgun.Domain)

	return &MailgunProvider{
		config:  config,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// SendEmail sends a single email via Mailgun
func (p *MailgunProvider) SendEmail(ctx context.Context, email *EmailMessage) (*SendResult, error) {
	endpoint := fmt.Sprintf("%s/messages", p.baseURL)

	// Prepare form data
	data := url.Values{}
	data.Set("from", p.formatFromAddress(email))
	data.Set("to", email.To)
	data.Set("subject", email.Subject)

	if email.ReplyTo != "" {
		data.Set("h:Reply-To", email.ReplyTo)
	}

	if email.HTMLContent != "" {
		data.Set("html", email.HTMLContent)
	}

	if email.TextContent != "" {
		data.Set("text", email.TextContent)
	}

	// Add custom variables/metadata
	for key, value := range email.Metadata {
		data.Set(fmt.Sprintf("v:%s", key), value)
	}

	// Add tracking
	data.Set("o:tracking", "yes")
	data.Set("o:tracking-clicks", "yes")
	data.Set("o:tracking-opens", "yes")

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return &SendResult{Error: fmt.Errorf("failed to create request: %w", err)}, nil
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("api", p.config.Mailgun.APIKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return &SendResult{Error: fmt.Errorf("failed to send request: %w", err)}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var mgError MailgunError
		if err := json.NewDecoder(resp.Body).Decode(&mgError); err == nil {
			return &SendResult{
				Error: fmt.Errorf("Mailgun error: %s", mgError.Message),
			}, nil
		}
		return &SendResult{
			Error: fmt.Errorf("Mailgun API error: status %d", resp.StatusCode),
		}, nil
	}

	var mgResp MailgunResponse
	if err := json.NewDecoder(resp.Body).Decode(&mgResp); err != nil {
		return &SendResult{Error: fmt.Errorf("failed to decode response: %w", err)}, nil
	}

	return &SendResult{
		MessageID: mgResp.ID,
		Status:    "sent",
	}, nil
}

// SendBulkEmails sends multiple emails via Mailgun
func (p *MailgunProvider) SendBulkEmails(ctx context.Context, emails []*EmailMessage) ([]*SendResult, error) {
	results := make([]*SendResult, len(emails))
	
	// Mailgun supports batch sending with recipient variables
	// For simplicity, we'll send individually, but in production you might want to batch
	for i, email := range emails {
		result, err := p.SendEmail(ctx, email)
		if err != nil {
			results[i] = &SendResult{Error: err}
		} else {
			results[i] = result
		}
		
		// Rate limiting - respect Mailgun's rate limits
		if i < len(emails)-1 {
			select {
			case <-ctx.Done():
				return results, ctx.Err()
			case <-time.After(time.Millisecond * 100): // 10 emails per second max for free tier
			}
		}
	}

	return results, nil
}

// formatFromAddress formats the from address with name
func (p *MailgunProvider) formatFromAddress(email *EmailMessage) string {
	if email.FromName != "" {
		return fmt.Sprintf("%s <%s>", email.FromName, email.From)
	}
	return email.From
}