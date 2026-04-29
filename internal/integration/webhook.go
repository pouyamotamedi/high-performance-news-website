package integration

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// WebhookManager manages webhook integrations
type WebhookManager struct {
	webhooks map[string]*Webhook
	client   *http.Client
	mu       sync.RWMutex
}

// Webhook represents a webhook configuration
type Webhook struct {
	Name      string            `json:"name"`
	URL       string            `json:"url"`
	Secret    string            `json:"secret"`
	Headers   map[string]string `json:"headers"`
	Enabled   bool              `json:"enabled"`
	Events    []EventType       `json:"events"`
	Retries   int               `json:"retries"`
	Timeout   time.Duration     `json:"timeout"`
}

// WebhookPayload represents the payload sent to webhooks
type WebhookPayload struct {
	Event     Event     `json:"event"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`
	Signature string    `json:"signature,omitempty"`
}

// NewWebhookManager creates a new webhook manager
func NewWebhookManager() *WebhookManager {
	return &WebhookManager{
		webhooks: make(map[string]*Webhook),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RegisterWebhook registers a new webhook
func (wm *WebhookManager) RegisterWebhook(webhook *Webhook) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if webhook.Timeout == 0 {
		webhook.Timeout = 10 * time.Second
	}
	if webhook.Retries == 0 {
		webhook.Retries = 3
	}

	wm.webhooks[webhook.Name] = webhook
	return nil
}

// SendWebhook sends an event to all matching webhooks
func (wm *WebhookManager) SendWebhook(ctx context.Context, event Event) error {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	var errors []error
	for name, webhook := range wm.webhooks {
		if !webhook.Enabled {
			continue
		}

		if !wm.shouldSendToWebhook(webhook, event) {
			continue
		}

		if err := wm.sendToWebhook(ctx, webhook, event); err != nil {
			errors = append(errors, fmt.Errorf("webhook %s failed: %w", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("webhook errors: %v", errors)
	}

	return nil
}

// shouldSendToWebhook checks if an event should be sent to a webhook
func (wm *WebhookManager) shouldSendToWebhook(webhook *Webhook, event Event) bool {
	if len(webhook.Events) == 0 {
		return true // Send all events if no filter specified
	}

	for _, eventType := range webhook.Events {
		if eventType == event.Type {
			return true
		}
	}

	return false
}

// sendToWebhook sends an event to a specific webhook with retries
func (wm *WebhookManager) sendToWebhook(ctx context.Context, webhook *Webhook, event Event) error {
	payload := WebhookPayload{
		Event:     event,
		Timestamp: time.Now(),
		Source:    "comprehensive-testing-qa",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Add signature if secret is provided
	if webhook.Secret != "" {
		signature := wm.generateSignature(jsonData, webhook.Secret)
		payload.Signature = signature
		
		// Re-marshal with signature
		jsonData, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload with signature: %w", err)
		}
	}

	var lastErr error
	for attempt := 0; attempt <= webhook.Retries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(attempt*attempt) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, "POST", webhook.URL, bytes.NewBuffer(jsonData))
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "comprehensive-testing-qa/1.0")

		// Add custom headers
		for key, value := range webhook.Headers {
			req.Header.Set(key, value)
		}

		// Add signature header if present
		if webhook.Secret != "" {
			req.Header.Set("X-Signature-SHA256", payload.Signature)
		}

		client := &http.Client{Timeout: webhook.Timeout}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			resp.Body.Close()
			return nil // Success
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		lastErr = fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	return fmt.Errorf("webhook failed after %d retries: %w", webhook.Retries, lastErr)
}

// generateSignature generates HMAC-SHA256 signature for webhook payload
func (wm *WebhookManager) generateSignature(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

// ValidateSignature validates webhook signature
func (wm *WebhookManager) ValidateSignature(payload []byte, signature, secret string) bool {
	expectedSignature := wm.generateSignature(payload, secret)
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// GetWebhooks returns all registered webhooks
func (wm *WebhookManager) GetWebhooks() map[string]*Webhook {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	webhooks := make(map[string]*Webhook)
	for name, webhook := range wm.webhooks {
		webhooks[name] = webhook
	}
	return webhooks
}

// RemoveWebhook removes a webhook
func (wm *WebhookManager) RemoveWebhook(name string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if _, exists := wm.webhooks[name]; !exists {
		return fmt.Errorf("webhook %s not found", name)
	}

	delete(wm.webhooks, name)
	return nil
}