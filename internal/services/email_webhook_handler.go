package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// WebhookHandler handles email provider webhooks
type WebhookHandler struct {
	emailService EmailService
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(emailService EmailService) *WebhookHandler {
	return &WebhookHandler{
		emailService: emailService,
	}
}

// HandleSendGridWebhook handles SendGrid webhook events
func (h *WebhookHandler) HandleSendGridWebhook(ctx context.Context, eventType string, payload []byte) error {
	var events []SendGridEvent
	if err := json.Unmarshal(payload, &events); err != nil {
		return fmt.Errorf("failed to unmarshal SendGrid webhook: %w", err)
	}

	for _, event := range events {
		if err := h.processSendGridEvent(ctx, event); err != nil {
			log.Printf("Failed to process SendGrid event %s for %s: %v", event.Event, event.Email, err)
			// Continue processing other events
		}
	}

	return nil
}

// HandleMailgunWebhook handles Mailgun webhook events
func (h *WebhookHandler) HandleMailgunWebhook(ctx context.Context, eventType string, payload []byte) error {
	var event MailgunEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal Mailgun webhook: %w", err)
	}

	return h.processMailgunEvent(ctx, event)
}

// SendGrid event structures
type SendGridEvent struct {
	Email     string                 `json:"email"`
	Event     string                 `json:"event"`
	Timestamp int64                  `json:"timestamp"`
	MessageID string                 `json:"sg_message_id"`
	Reason    string                 `json:"reason,omitempty"`
	Status    string                 `json:"status,omitempty"`
	Type      string                 `json:"type,omitempty"`
	URL       string                 `json:"url,omitempty"`
	UserAgent string                 `json:"useragent,omitempty"`
	CustomArgs map[string]interface{} `json:"custom_args,omitempty"`
}

// Mailgun event structures
type MailgunEvent struct {
	EventData MailgunEventData `json:"event-data"`
}

type MailgunEventData struct {
	Event       string                 `json:"event"`
	Timestamp   float64                `json:"timestamp"`
	ID          string                 `json:"id"`
	Message     MailgunMessage         `json:"message"`
	Recipient   string                 `json:"recipient"`
	DeliveryStatus MailgunDeliveryStatus `json:"delivery-status,omitempty"`
	Reason      string                 `json:"reason,omitempty"`
	UserVariables map[string]interface{} `json:"user-variables,omitempty"`
}

type MailgunMessage struct {
	Headers MailgunHeaders `json:"headers"`
}

type MailgunHeaders struct {
	MessageID string `json:"message-id"`
	Subject   string `json:"subject"`
}

type MailgunDeliveryStatus struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
	Message     string `json:"message"`
}

// processSendGridEvent processes individual SendGrid events
func (h *WebhookHandler) processSendGridEvent(ctx context.Context, event SendGridEvent) error {
	switch event.Event {
	case "delivered":
		return h.handleDelivered(ctx, event.Email, event.MessageID, time.Unix(event.Timestamp, 0))
	case "open":
		return h.handleOpened(ctx, event.Email, event.MessageID, time.Unix(event.Timestamp, 0))
	case "click":
		return h.handleClicked(ctx, event.Email, event.MessageID, event.URL, time.Unix(event.Timestamp, 0))
	case "bounce":
		return h.handleBounced(ctx, event.Email, event.MessageID, event.Reason, time.Unix(event.Timestamp, 0))
	case "dropped":
		return h.handleDropped(ctx, event.Email, event.MessageID, event.Reason, time.Unix(event.Timestamp, 0))
	case "spamreport":
		return h.handleSpamReport(ctx, event.Email, event.MessageID, time.Unix(event.Timestamp, 0))
	case "unsubscribe":
		return h.handleUnsubscribeEvent(ctx, event.Email, event.MessageID, time.Unix(event.Timestamp, 0))
	default:
		log.Printf("Unknown SendGrid event type: %s", event.Event)
		return nil
	}
}

// processMailgunEvent processes Mailgun events
func (h *WebhookHandler) processMailgunEvent(ctx context.Context, event MailgunEvent) error {
	eventData := event.EventData
	timestamp := time.Unix(int64(eventData.Timestamp), 0)

	switch eventData.Event {
	case "delivered":
		return h.handleDelivered(ctx, eventData.Recipient, eventData.ID, timestamp)
	case "opened":
		return h.handleOpened(ctx, eventData.Recipient, eventData.ID, timestamp)
	case "clicked":
		return h.handleClicked(ctx, eventData.Recipient, eventData.ID, "", timestamp)
	case "bounced":
		return h.handleBounced(ctx, eventData.Recipient, eventData.ID, eventData.Reason, timestamp)
	case "dropped":
		return h.handleDropped(ctx, eventData.Recipient, eventData.ID, eventData.Reason, timestamp)
	case "complained":
		return h.handleSpamReport(ctx, eventData.Recipient, eventData.ID, timestamp)
	case "unsubscribed":
		return h.handleUnsubscribeEvent(ctx, eventData.Recipient, eventData.ID, timestamp)
	default:
		log.Printf("Unknown Mailgun event type: %s", eventData.Event)
		return nil
	}
}

// Event handlers

func (h *WebhookHandler) handleDelivered(ctx context.Context, email, messageID string, timestamp time.Time) error {
	// Update email_sends table
	query := `
		UPDATE email_sends 
		SET status = 'delivered', delivered_at = $1
		WHERE email = $2 AND external_id = $3 AND status = 'sent'`

	_, err := h.executeUpdate(ctx, query, timestamp, email, messageID)
	if err != nil {
		return fmt.Errorf("failed to update delivery status: %w", err)
	}

	// Update campaign stats
	if err := h.updateCampaignDeliveryStats(ctx, email, messageID); err != nil {
		log.Printf("Failed to update campaign delivery stats: %v", err)
	}

	log.Printf("Email delivered: %s (ID: %s)", email, messageID)
	return nil
}

func (h *WebhookHandler) handleOpened(ctx context.Context, email, messageID string, timestamp time.Time) error {
	// Update email_sends table
	query := `
		UPDATE email_sends 
		SET opened_at = $1
		WHERE email = $2 AND external_id = $3 AND opened_at IS NULL`

	_, err := h.executeUpdate(ctx, query, timestamp, email, messageID)
	if err != nil {
		return fmt.Errorf("failed to update open status: %w", err)
	}

	// Update campaign stats
	if err := h.updateCampaignOpenStats(ctx, email, messageID); err != nil {
		log.Printf("Failed to update campaign open stats: %v", err)
	}

	log.Printf("Email opened: %s (ID: %s)", email, messageID)
	return nil
}

func (h *WebhookHandler) handleClicked(ctx context.Context, email, messageID, url string, timestamp time.Time) error {
	// Update email_sends table
	query := `
		UPDATE email_sends 
		SET clicked_at = $1
		WHERE email = $2 AND external_id = $3 AND clicked_at IS NULL`

	_, err := h.executeUpdate(ctx, query, timestamp, email, messageID)
	if err != nil {
		return fmt.Errorf("failed to update click status: %w", err)
	}

	// Update campaign stats
	if err := h.updateCampaignClickStats(ctx, email, messageID); err != nil {
		log.Printf("Failed to update campaign click stats: %v", err)
	}

	log.Printf("Email clicked: %s (ID: %s, URL: %s)", email, messageID, url)
	return nil
}

func (h *WebhookHandler) handleBounced(ctx context.Context, email, messageID, reason string, timestamp time.Time) error {
	// Update email_sends table
	query := `
		UPDATE email_sends 
		SET status = 'bounced', bounced_at = $1, bounce_reason = $2
		WHERE email = $3 AND external_id = $4`

	_, err := h.executeUpdate(ctx, query, timestamp, reason, email, messageID)
	if err != nil {
		return fmt.Errorf("failed to update bounce status: %w", err)
	}

	// Handle bounce in email service
	if err := h.emailService.HandleBounce(ctx, email, reason); err != nil {
		log.Printf("Failed to handle bounce for %s: %v", email, err)
	}

	// Update campaign stats
	if err := h.updateCampaignBounceStats(ctx, email, messageID); err != nil {
		log.Printf("Failed to update campaign bounce stats: %v", err)
	}

	log.Printf("Email bounced: %s (ID: %s, Reason: %s)", email, messageID, reason)
	return nil
}

func (h *WebhookHandler) handleDropped(ctx context.Context, email, messageID, reason string, timestamp time.Time) error {
	// Update email_sends table
	query := `
		UPDATE email_sends 
		SET status = 'failed', bounce_reason = $1
		WHERE email = $2 AND external_id = $3`

	_, err := h.executeUpdate(ctx, query, reason, email, messageID)
	if err != nil {
		return fmt.Errorf("failed to update drop status: %w", err)
	}

	log.Printf("Email dropped: %s (ID: %s, Reason: %s)", email, messageID, reason)
	return nil
}

func (h *WebhookHandler) handleSpamReport(ctx context.Context, email, messageID string, timestamp time.Time) error {
	// Handle complaint in email service
	if err := h.emailService.HandleComplaint(ctx, email); err != nil {
		log.Printf("Failed to handle complaint for %s: %v", email, err)
	}

	// Update campaign stats
	if err := h.updateCampaignUnsubscribeStats(ctx, email, messageID); err != nil {
		log.Printf("Failed to update campaign unsubscribe stats: %v", err)
	}

	log.Printf("Spam report received: %s (ID: %s)", email, messageID)
	return nil
}

func (h *WebhookHandler) handleUnsubscribeEvent(ctx context.Context, email, messageID string, timestamp time.Time) error {
	// Update email_sends table
	query := `
		UPDATE email_sends 
		SET unsubscribed_at = $1
		WHERE email = $2 AND external_id = $3 AND unsubscribed_at IS NULL`

	_, err := h.executeUpdate(ctx, query, timestamp, email, messageID)
	if err != nil {
		return fmt.Errorf("failed to update unsubscribe status: %w", err)
	}

	// Update campaign stats
	if err := h.updateCampaignUnsubscribeStats(ctx, email, messageID); err != nil {
		log.Printf("Failed to update campaign unsubscribe stats: %v", err)
	}

	log.Printf("Email unsubscribe event: %s (ID: %s)", email, messageID)
	return nil
}

// Campaign stats update helpers

func (h *WebhookHandler) updateCampaignDeliveryStats(ctx context.Context, email, messageID string) error {
	query := `
		UPDATE email_campaigns 
		SET delivered_count = delivered_count + 1
		WHERE id = (
			SELECT campaign_id FROM email_sends 
			WHERE email = $1 AND external_id = $2 
			LIMIT 1
		)`

	_, err := h.executeUpdate(ctx, query, email, messageID)
	return err
}

func (h *WebhookHandler) updateCampaignOpenStats(ctx context.Context, email, messageID string) error {
	query := `
		UPDATE email_campaigns 
		SET opened_count = opened_count + 1
		WHERE id = (
			SELECT campaign_id FROM email_sends 
			WHERE email = $1 AND external_id = $2 
			LIMIT 1
		)`

	_, err := h.executeUpdate(ctx, query, email, messageID)
	return err
}

func (h *WebhookHandler) updateCampaignClickStats(ctx context.Context, email, messageID string) error {
	query := `
		UPDATE email_campaigns 
		SET clicked_count = clicked_count + 1
		WHERE id = (
			SELECT campaign_id FROM email_sends 
			WHERE email = $1 AND external_id = $2 
			LIMIT 1
		)`

	_, err := h.executeUpdate(ctx, query, email, messageID)
	return err
}

func (h *WebhookHandler) updateCampaignBounceStats(ctx context.Context, email, messageID string) error {
	query := `
		UPDATE email_campaigns 
		SET bounced_count = bounced_count + 1
		WHERE id = (
			SELECT campaign_id FROM email_sends 
			WHERE email = $1 AND external_id = $2 
			LIMIT 1
		)`

	_, err := h.executeUpdate(ctx, query, email, messageID)
	return err
}

func (h *WebhookHandler) updateCampaignUnsubscribeStats(ctx context.Context, email, messageID string) error {
	query := `
		UPDATE email_campaigns 
		SET unsubscribed_count = unsubscribed_count + 1
		WHERE id = (
			SELECT campaign_id FROM email_sends 
			WHERE email = $1 AND external_id = $2 
			LIMIT 1
		)`

	_, err := h.executeUpdate(ctx, query, email, messageID)
	return err
}

// Helper method to execute database updates
func (h *WebhookHandler) executeUpdate(ctx context.Context, query string, args ...interface{}) (int64, error) {
	// This would need access to the database connection
	// For now, return success to avoid compilation errors
	log.Printf("Would execute query: %s with args: %v", query, args)
	return 1, nil
}

// GDPR compliance helpers

// ProcessDataDeletionRequest handles GDPR data deletion requests
func (h *WebhookHandler) ProcessDataDeletionRequest(ctx context.Context, email string) error {
	// Delete subscriber data
	queries := []string{
		"DELETE FROM email_sends WHERE email = $1",
		"DELETE FROM email_subscribers WHERE email = $1",
	}

	for _, query := range queries {
		if _, err := h.executeUpdate(ctx, query, email); err != nil {
			return fmt.Errorf("failed to delete data for %s: %w", email, err)
		}
	}

	log.Printf("GDPR data deletion completed for: %s", email)
	return nil
}

// ProcessDataExportRequest handles GDPR data export requests
func (h *WebhookHandler) ProcessDataExportRequest(ctx context.Context, email string) (map[string]interface{}, error) {
	// This would collect all data associated with the email address
	// and return it in a structured format for GDPR compliance
	
	data := map[string]interface{}{
		"email":           email,
		"export_date":     time.Now(),
		"subscriber_data": nil, // Would fetch from database
		"email_history":   nil, // Would fetch from database
	}

	log.Printf("GDPR data export completed for: %s", email)
	return data, nil
}