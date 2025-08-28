package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"high-performance-news-website/internal/models"
)

// EmailProcessor handles email-related background jobs
type EmailProcessor struct {
	emailService EmailService
	jobQueue     JobQueue
}

// EmailJob represents different types of email jobs
type EmailJob struct {
	Type    EmailJobType               `json:"type"`
	Payload map[string]interface{}     `json:"payload"`
}

// EmailJobType represents the type of email job
type EmailJobType string

const (
	EmailJobTypeWelcome      EmailJobType = "welcome"
	EmailJobTypeConfirmation EmailJobType = "confirmation"
	EmailJobTypeCampaign     EmailJobType = "campaign"
	EmailJobTypeBulk         EmailJobType = "bulk"
	EmailJobTypeNewsletter   EmailJobType = "newsletter"
)

// EmailService interface to avoid import cycles
type EmailService interface {
	SendWelcomeEmail(ctx context.Context, subscriber *models.EmailSubscriber) error
	SendConfirmationEmail(ctx context.Context, subscriber *models.EmailSubscriber) error
	SendCampaign(ctx context.Context, campaignID uint64) error
	SendBulkEmails(ctx context.Context, emails []BulkEmail) error
	GetSubscribers(ctx context.Context, status models.SubscriberStatus, limit, offset int) ([]*models.EmailSubscriber, error)
}

// BulkEmail represents a bulk email to be sent
type BulkEmail struct {
	To          string            `json:"to"`
	Subject     string            `json:"subject"`
	HTMLContent string            `json:"html_content"`
	TextContent string            `json:"text_content"`
	Metadata    map[string]string `json:"metadata"`
}

// NewEmailProcessor creates a new email processor
func NewEmailProcessor(emailService EmailService, jobQueue JobQueue) *EmailProcessor {
	return &EmailProcessor{
		emailService: emailService,
		jobQueue:     jobQueue,
	}
}

// Start starts the email processor
func (p *EmailProcessor) Start(ctx context.Context) error {
	log.Println("Starting email processor...")

	// Register job handlers
	p.jobQueue.RegisterHandler("email", p.handleEmailJob)

	// Start processing jobs
	return p.jobQueue.Start(ctx)
}

// QueueWelcomeEmail queues a welcome email job
func (p *EmailProcessor) QueueWelcomeEmail(subscriberID uint64) error {
	job := &EmailJob{
		Type: EmailJobTypeWelcome,
		Payload: map[string]interface{}{
			"subscriber_id": subscriberID,
		},
	}

	return p.queueJob(job, JobPriorityHigh)
}

// QueueConfirmationEmail queues a confirmation email job
func (p *EmailProcessor) QueueConfirmationEmail(subscriberID uint64) error {
	job := &EmailJob{
		Type: EmailJobTypeConfirmation,
		Payload: map[string]interface{}{
			"subscriber_id": subscriberID,
		},
	}

	return p.queueJob(job, JobPriorityHigh)
}

// QueueCampaignSend queues a campaign send job
func (p *EmailProcessor) QueueCampaignSend(campaignID uint64) error {
	job := &EmailJob{
		Type: EmailJobTypeCampaign,
		Payload: map[string]interface{}{
			"campaign_id": campaignID,
		},
	}

	return p.queueJob(job, JobPriorityMedium)
}

// QueueBulkEmails queues bulk email sending
func (p *EmailProcessor) QueueBulkEmails(emails []BulkEmail) error {
	// Split into batches to respect rate limits
	batchSize := 1000 // Configurable batch size
	
	for i := 0; i < len(emails); i += batchSize {
		end := i + batchSize
		if end > len(emails) {
			end = len(emails)
		}

		batch := emails[i:end]
		job := &EmailJob{
			Type: EmailJobTypeBulk,
			Payload: map[string]interface{}{
				"emails": batch,
				"batch":  i/batchSize + 1,
			},
		}

		if err := p.queueJob(job, JobPriorityMedium); err != nil {
			return fmt.Errorf("failed to queue batch %d: %w", i/batchSize+1, err)
		}
	}

	return nil
}

// QueueNewsletterSend queues newsletter sending to all subscribers
func (p *EmailProcessor) QueueNewsletterSend(subject, content string, templateID *uint64) error {
	job := &EmailJob{
		Type: EmailJobTypeNewsletter,
		Payload: map[string]interface{}{
			"subject":     subject,
			"content":     content,
			"template_id": templateID,
		},
	}

	return p.queueJob(job, JobPriorityLow)
}

// handleEmailJob handles email jobs
func (p *EmailProcessor) handleEmailJob(ctx context.Context, jobData []byte) error {
	var job EmailJob
	if err := json.Unmarshal(jobData, &job); err != nil {
		return fmt.Errorf("failed to unmarshal email job: %w", err)
	}

	switch job.Type {
	case EmailJobTypeWelcome:
		return p.handleWelcomeEmail(ctx, job.Payload)
	case EmailJobTypeConfirmation:
		return p.handleConfirmationEmail(ctx, job.Payload)
	case EmailJobTypeCampaign:
		return p.handleCampaignSend(ctx, job.Payload)
	case EmailJobTypeBulk:
		return p.handleBulkEmails(ctx, job.Payload)
	case EmailJobTypeNewsletter:
		return p.handleNewsletterSend(ctx, job.Payload)
	default:
		return fmt.Errorf("unknown email job type: %s", job.Type)
	}
}

// handleWelcomeEmail handles welcome email sending
func (p *EmailProcessor) handleWelcomeEmail(ctx context.Context, payload map[string]interface{}) error {
	subscriberID, ok := payload["subscriber_id"].(float64)
	if !ok {
		return fmt.Errorf("invalid subscriber_id in payload")
	}

	// Get subscriber details
	subscriber, err := p.getSubscriberByID(ctx, uint64(subscriberID))
	if err != nil {
		return fmt.Errorf("failed to get subscriber: %w", err)
	}

	// Send welcome email
	if err := p.emailService.SendWelcomeEmail(ctx, subscriber); err != nil {
		return fmt.Errorf("failed to send welcome email: %w", err)
	}

	log.Printf("Welcome email sent to subscriber %d (%s)", uint64(subscriberID), subscriber.Email)
	return nil
}

// handleConfirmationEmail handles confirmation email sending
func (p *EmailProcessor) handleConfirmationEmail(ctx context.Context, payload map[string]interface{}) error {
	subscriberID, ok := payload["subscriber_id"].(float64)
	if !ok {
		return fmt.Errorf("invalid subscriber_id in payload")
	}

	// Get subscriber details
	subscriber, err := p.getSubscriberByID(ctx, uint64(subscriberID))
	if err != nil {
		return fmt.Errorf("failed to get subscriber: %w", err)
	}

	// Send confirmation email
	if err := p.emailService.SendConfirmationEmail(ctx, subscriber); err != nil {
		return fmt.Errorf("failed to send confirmation email: %w", err)
	}

	log.Printf("Confirmation email sent to subscriber %d (%s)", uint64(subscriberID), subscriber.Email)
	return nil
}

// handleCampaignSend handles campaign sending
func (p *EmailProcessor) handleCampaignSend(ctx context.Context, payload map[string]interface{}) error {
	campaignID, ok := payload["campaign_id"].(float64)
	if !ok {
		return fmt.Errorf("invalid campaign_id in payload")
	}

	// Send campaign
	if err := p.emailService.SendCampaign(ctx, uint64(campaignID)); err != nil {
		return fmt.Errorf("failed to send campaign: %w", err)
	}

	log.Printf("Campaign %d sent successfully", uint64(campaignID))
	return nil
}

// handleBulkEmails handles bulk email sending
func (p *EmailProcessor) handleBulkEmails(ctx context.Context, payload map[string]interface{}) error {
	emailsData, ok := payload["emails"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid emails in payload")
	}

	batch, ok := payload["batch"].(float64)
	if !ok {
		batch = 1
	}

	// Convert to BulkEmail slice
	var emails []BulkEmail
	for _, emailData := range emailsData {
		emailMap, ok := emailData.(map[string]interface{})
		if !ok {
			continue
		}

		email := BulkEmail{
			To:          getString(emailMap, "to"),
			Subject:     getString(emailMap, "subject"),
			HTMLContent: getString(emailMap, "html_content"),
			TextContent: getString(emailMap, "text_content"),
		}

		if metadataData, exists := emailMap["metadata"]; exists {
			if metadataMap, ok := metadataData.(map[string]interface{}); ok {
				email.Metadata = make(map[string]string)
				for k, v := range metadataMap {
					if str, ok := v.(string); ok {
						email.Metadata[k] = str
					}
				}
			}
		}

		emails = append(emails, email)
	}

	// Send bulk emails with rate limiting
	if err := p.emailService.SendBulkEmails(ctx, emails); err != nil {
		return fmt.Errorf("failed to send bulk emails batch %d: %w", int(batch), err)
	}

	log.Printf("Bulk email batch %d sent successfully (%d emails)", int(batch), len(emails))
	return nil
}

// handleNewsletterSend handles newsletter sending to all subscribers
func (p *EmailProcessor) handleNewsletterSend(ctx context.Context, payload map[string]interface{}) error {
	subject := getString(payload, "subject")
	content := getString(payload, "content")
	
	var templateID *uint64
	if tid, exists := payload["template_id"]; exists && tid != nil {
		if tidFloat, ok := tid.(float64); ok {
			id := uint64(tidFloat)
			templateID = &id
		}
	}

	// Get all confirmed subscribers
	subscribers, err := p.emailService.GetSubscribers(ctx, models.SubscriberStatusConfirmed, 10000, 0)
	if err != nil {
		return fmt.Errorf("failed to get subscribers: %w", err)
	}

	// Create bulk emails
	var emails []BulkEmail
	for _, subscriber := range subscribers {
		// Personalize content if needed
		personalizedSubject := subject
		personalizedContent := content

		// Replace placeholders
		personalizedSubject = replacePlaceholders(personalizedSubject, subscriber)
		personalizedContent = replacePlaceholders(personalizedContent, subscriber)

		email := BulkEmail{
			To:          subscriber.Email,
			Subject:     personalizedSubject,
			HTMLContent: personalizedContent,
			TextContent: stripHTML(personalizedContent),
			Metadata: map[string]string{
				"subscriber_id": fmt.Sprintf("%d", subscriber.ID),
				"campaign_type": "newsletter",
			},
		}

		emails = append(emails, email)
	}

	// Queue bulk email sending
	if err := p.QueueBulkEmails(emails); err != nil {
		return fmt.Errorf("failed to queue newsletter emails: %w", err)
	}

	log.Printf("Newsletter queued for %d subscribers", len(emails))
	return nil
}

// queueJob queues an email job
func (p *EmailProcessor) queueJob(job *EmailJob, priority JobPriority) error {
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	return p.jobQueue.Enqueue(&Job{
		Type:     "email",
		Data:     jobData,
		Priority: priority,
		Retry:    3,
		Delay:    0,
	})
}

// Helper functions

func (p *EmailProcessor) getSubscriberByID(ctx context.Context, id uint64) (*models.EmailSubscriber, error) {
	// This would need to be implemented to get subscriber by ID
	// For now, return a placeholder
	return &models.EmailSubscriber{
		ID:    id,
		Email: "placeholder@example.com",
	}, nil
}

func getString(m map[string]interface{}, key string) string {
	if val, exists := m[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func replacePlaceholders(content string, subscriber *models.EmailSubscriber) string {
	// Replace common placeholders
	content = fmt.Sprintf(content, subscriber.FirstName, subscriber.LastName, subscriber.Email)
	return content
}

func stripHTML(html string) string {
	// Simple HTML stripping - in production, use a proper HTML parser
	// This is a placeholder implementation
	return html
}

// RateLimiter for email sending
type EmailRateLimiter struct {
	maxPerHour int
	sent       int
	resetTime  time.Time
}

// NewEmailRateLimiter creates a new rate limiter
func NewEmailRateLimiter(maxPerHour int) *EmailRateLimiter {
	return &EmailRateLimiter{
		maxPerHour: maxPerHour,
		resetTime:  time.Now().Add(time.Hour),
	}
}

// Allow checks if sending is allowed
func (rl *EmailRateLimiter) Allow() bool {
	now := time.Now()
	if now.After(rl.resetTime) {
		rl.sent = 0
		rl.resetTime = now.Add(time.Hour)
	}

	if rl.sent >= rl.maxPerHour {
		return false
	}

	rl.sent++
	return true
}

// GetWaitTime returns how long to wait before next send
func (rl *EmailRateLimiter) GetWaitTime() time.Duration {
	if rl.sent < rl.maxPerHour {
		return 0
	}
	return time.Until(rl.resetTime)
}