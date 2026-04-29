package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"high-performance-news-website/internal/config"
)

// SendGridProvider implements EmailProvider for SendGrid
type SendGridProvider struct {
	config     *config.EmailConfig
	httpClient *http.Client
}

// SendGridMessage represents a SendGrid email message
type SendGridMessage struct {
	Personalizations []SendGridPersonalization `json:"personalizations"`
	From             SendGridEmail             `json:"from"`
	ReplyTo          *SendGridEmail            `json:"reply_to,omitempty"`
	Subject          string                    `json:"subject"`
	Content          []SendGridContent         `json:"content"`
	CustomArgs       map[string]string         `json:"custom_args,omitempty"`
}

// SendGridPersonalization represents SendGrid personalization
type SendGridPersonalization struct {
	To           []SendGridEmail   `json:"to"`
	CustomArgs   map[string]string `json:"custom_args,omitempty"`
}

// SendGridEmail represents a SendGrid email address
type SendGridEmail struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// SendGridContent represents SendGrid email content
type SendGridContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// SendGridResponse represents SendGrid API response
type SendGridResponse struct {
	MessageID string `json:"message_id,omitempty"`
	Errors    []struct {
		Message string `json:"message"`
		Field   string `json:"field,omitempty"`
		Help    string `json:"help,omitempty"`
	} `json:"errors,omitempty"`
}

// NewSendGridProvider creates a new SendGrid provider
func NewSendGridProvider(config *config.EmailConfig) (*SendGridProvider, error) {
	if config.SendGrid.APIKey == "" {
		return nil, fmt.Errorf("SendGrid API key is required")
	}

	return &SendGridProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// SendEmail sends a single email via SendGrid
func (p *SendGridProvider) SendEmail(ctx context.Context, email *EmailMessage) (*SendResult, error) {
	message := p.buildSendGridMessage(email)
	
	jsonData, err := json.Marshal(message)
	if err != nil {
		return &SendResult{Error: fmt.Errorf("failed to marshal message: %w", err)}, nil
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.SendGrid.Endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return &SendResult{Error: fmt.Errorf("failed to create request: %w", err)}, nil
	}

	req.Header.Set("Authorization", "Bearer "+p.config.SendGrid.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return &SendResult{Error: fmt.Errorf("failed to send request: %w", err)}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var sgResp SendGridResponse
		if err := json.NewDecoder(resp.Body).Decode(&sgResp); err == nil && len(sgResp.Errors) > 0 {
			return &SendResult{
				Error: fmt.Errorf("SendGrid error: %s", sgResp.Errors[0].Message),
			}, nil
		}
		return &SendResult{
			Error: fmt.Errorf("SendGrid API error: status %d", resp.StatusCode),
		}, nil
	}

	// Extract message ID from response headers
	messageID := resp.Header.Get("X-Message-Id")
	if messageID == "" {
		messageID = fmt.Sprintf("sg_%d", time.Now().Unix())
	}

	return &SendResult{
		MessageID: messageID,
		Status:    "sent",
	}, nil
}

// SendBulkEmails sends multiple emails via SendGrid
func (p *SendGridProvider) SendBulkEmails(ctx context.Context, emails []*EmailMessage) ([]*SendResult, error) {
	results := make([]*SendResult, len(emails))
	
	// SendGrid supports batch sending, but for simplicity, we'll send individually
	// In production, you might want to use SendGrid's batch API for better performance
	for i, email := range emails {
		result, err := p.SendEmail(ctx, email)
		if err != nil {
			results[i] = &SendResult{Error: err}
		} else {
			results[i] = result
		}
		
		// Rate limiting - respect SendGrid's rate limits
		if i < len(emails)-1 {
			select {
			case <-ctx.Done():
				return results, ctx.Err()
			case <-time.After(time.Millisecond * 10): // 100 emails per second max
			}
		}
	}

	return results, nil
}

// buildSendGridMessage converts EmailMessage to SendGrid format
func (p *SendGridProvider) buildSendGridMessage(email *EmailMessage) *SendGridMessage {
	message := &SendGridMessage{
		Personalizations: []SendGridPersonalization{
			{
				To: []SendGridEmail{
					{
						Email: email.To,
					},
				},
			},
		},
		From: SendGridEmail{
			Email: email.From,
			Name:  email.FromName,
		},
		Subject: email.Subject,
		Content: []SendGridContent{},
	}

	// Add reply-to if specified
	if email.ReplyTo != "" {
		message.ReplyTo = &SendGridEmail{
			Email: email.ReplyTo,
		}
	}

	// Add HTML content
	if email.HTMLContent != "" {
		message.Content = append(message.Content, SendGridContent{
			Type:  "text/html",
			Value: email.HTMLContent,
		})
	}

	// Add text content
	if email.TextContent != "" {
		message.Content = append(message.Content, SendGridContent{
			Type:  "text/plain",
			Value: email.TextContent,
		})
	}

	// Add custom args/metadata
	if len(email.Metadata) > 0 {
		message.CustomArgs = email.Metadata
		message.Personalizations[0].CustomArgs = email.Metadata
	}

	return message
}