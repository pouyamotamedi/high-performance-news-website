package services

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"strings"
	"time"

	"high-performance-news-website/internal/config"
	"high-performance-news-website/internal/models"
)

// EmailService interface defines email service operations
type EmailService interface {
	// Subscriber management
	Subscribe(ctx context.Context, req *models.SubscribeRequest) (*models.EmailSubscriber, error)
	ConfirmSubscription(ctx context.Context, token string) error
	Unsubscribe(ctx context.Context, token string) error
	GetSubscriber(ctx context.Context, email string) (*models.EmailSubscriber, error)
	GetSubscribers(ctx context.Context, status models.SubscriberStatus, limit, offset int) ([]*models.EmailSubscriber, error)
	UpdateSubscriberPreferences(ctx context.Context, email string, preferences map[string]interface{}) error

	// Campaign management
	CreateCampaign(ctx context.Context, req *models.CreateCampaignRequest, userID uint64) (*models.EmailCampaign, error)
	GetCampaign(ctx context.Context, id uint64) (*models.EmailCampaign, error)
	GetCampaigns(ctx context.Context, limit, offset int) ([]*models.EmailCampaign, error)
	UpdateCampaign(ctx context.Context, id uint64, updates map[string]interface{}) error
	DeleteCampaign(ctx context.Context, id uint64) error
	ScheduleCampaign(ctx context.Context, id uint64, scheduledAt time.Time) error
	SendCampaign(ctx context.Context, id uint64) error

	// Template management
	CreateTemplate(ctx context.Context, template *models.EmailTemplate) (*models.EmailTemplate, error)
	GetTemplate(ctx context.Context, id uint64) (*models.EmailTemplate, error)
	GetTemplates(ctx context.Context, templateType models.TemplateType) ([]*models.EmailTemplate, error)
	UpdateTemplate(ctx context.Context, id uint64, updates map[string]interface{}) error
	DeleteTemplate(ctx context.Context, id uint64) error

	// Email sending
	SendEmail(ctx context.Context, to, subject, htmlContent, textContent string) error
	SendBulkEmails(ctx context.Context, emails []BulkEmail) error
	SendWelcomeEmail(ctx context.Context, subscriber *models.EmailSubscriber) error
	SendConfirmationEmail(ctx context.Context, subscriber *models.EmailSubscriber) error

	// Webhook handling
	HandleWebhook(ctx context.Context, provider string, eventType string, payload []byte) error

	// Analytics
	GetEmailStats(ctx context.Context) (*models.EmailStats, error)
	GetCampaignStats(ctx context.Context, campaignID uint64) (*CampaignStats, error)

	// Bounce and complaint handling
	HandleBounce(ctx context.Context, email string, reason string) error
	HandleComplaint(ctx context.Context, email string) error
}

// BulkEmail represents a bulk email to be sent
type BulkEmail struct {
	To          string
	Subject     string
	HTMLContent string
	TextContent string
	Metadata    map[string]string
}

// CampaignStats represents campaign statistics
type CampaignStats struct {
	CampaignID      uint64    `json:"campaign_id"`
	SentCount       int       `json:"sent_count"`
	DeliveredCount  int       `json:"delivered_count"`
	OpenedCount     int       `json:"opened_count"`
	ClickedCount    int       `json:"clicked_count"`
	BouncedCount    int       `json:"bounced_count"`
	UnsubscribedCount int     `json:"unsubscribed_count"`
	OpenRate        float64   `json:"open_rate"`
	ClickRate       float64   `json:"click_rate"`
	BounceRate      float64   `json:"bounce_rate"`
	UnsubscribeRate float64   `json:"unsubscribe_rate"`
	LastUpdated     time.Time `json:"last_updated"`
}

// emailService implements EmailService interface
type emailService struct {
	db     *sql.DB
	config *config.EmailConfig
	client EmailProvider
}

// EmailProvider interface for different email providers
type EmailProvider interface {
	SendEmail(ctx context.Context, email *EmailMessage) (*SendResult, error)
	SendBulkEmails(ctx context.Context, emails []*EmailMessage) ([]*SendResult, error)
}

// EmailMessage represents an email message
type EmailMessage struct {
	To          string
	From        string
	FromName    string
	ReplyTo     string
	Subject     string
	HTMLContent string
	TextContent string
	Metadata    map[string]string
}

// SendResult represents the result of sending an email
type SendResult struct {
	MessageID string
	Status    string
	Error     error
}

// NewEmailService creates a new email service
func NewEmailService(db *sql.DB, config *config.EmailConfig) (EmailService, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid email config: %w", err)
	}

	var client EmailProvider
	var err error

	switch config.Provider {
	case "sendgrid":
		client, err = NewSendGridProvider(config)
	case "mailgun":
		client, err = NewMailgunProvider(config)
	default:
		return nil, fmt.Errorf("unsupported email provider: %s", config.Provider)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create email provider: %w", err)
	}

	return &emailService{
		db:     db,
		config: config,
		client: client,
	}, nil
}

// Subscribe creates a new email subscription with double opt-in
func (s *emailService) Subscribe(ctx context.Context, req *models.SubscribeRequest) (*models.EmailSubscriber, error) {
	// Check if subscriber already exists
	existing, err := s.GetSubscriber(ctx, req.Email)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check existing subscriber: %w", err)
	}

	if existing != nil {
		if existing.Status == models.SubscriberStatusConfirmed {
			return existing, nil // Already subscribed
		}
		if existing.Status == models.SubscriberStatusUnsubscribed {
			// Resubscribe
			return s.resubscribe(ctx, existing, req)
		}
	}

	// Generate tokens
	confirmationToken, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate confirmation token: %w", err)
	}

	unsubscribeToken, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate unsubscribe token: %w", err)
	}

	// Create subscriber
	subscriber := &models.EmailSubscriber{
		Email:             req.Email,
		Status:            models.SubscriberStatusPending,
		ConfirmationToken: confirmationToken,
		UnsubscribeToken:  unsubscribeToken,
		FirstName:         req.FirstName,
		LastName:          req.LastName,
		Preferences:       req.Preferences,
		Source:            req.Source,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Insert into database
	query := `
		INSERT INTO email_subscribers (email, status, confirmation_token, unsubscribe_token, 
			first_name, last_name, preferences, source, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`

	preferencesJSON, _ := json.Marshal(subscriber.Preferences)
	err = s.db.QueryRowContext(ctx, query,
		subscriber.Email, subscriber.Status, subscriber.ConfirmationToken,
		subscriber.UnsubscribeToken, subscriber.FirstName, subscriber.LastName,
		preferencesJSON, subscriber.Source, subscriber.CreatedAt, subscriber.UpdatedAt,
	).Scan(&subscriber.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to create subscriber: %w", err)
	}

	// Send confirmation email
	if err := s.SendConfirmationEmail(ctx, subscriber); err != nil {
		log.Printf("Failed to send confirmation email to %s: %v", subscriber.Email, err)
		// Don't fail the subscription if email sending fails
	}

	return subscriber, nil
}

// ConfirmSubscription confirms a pending subscription
func (s *emailService) ConfirmSubscription(ctx context.Context, token string) error {
	query := `
		UPDATE email_subscribers 
		SET status = $1, confirmed_at = $2, updated_at = $3
		WHERE confirmation_token = $4 AND status = $5`

	now := time.Now()
	result, err := s.db.ExecContext(ctx, query,
		models.SubscriberStatusConfirmed, now, now, token, models.SubscriberStatusPending)

	if err != nil {
		return fmt.Errorf("failed to confirm subscription: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("invalid or expired confirmation token")
	}

	// Get subscriber for welcome email
	subscriber, err := s.getSubscriberByToken(ctx, token, "confirmation_token")
	if err != nil {
		log.Printf("Failed to get subscriber for welcome email: %v", err)
		return nil // Don't fail confirmation if welcome email fails
	}

	// Send welcome email
	if err := s.SendWelcomeEmail(ctx, subscriber); err != nil {
		log.Printf("Failed to send welcome email to %s: %v", subscriber.Email, err)
	}

	return nil
}

// Unsubscribe unsubscribes a user
func (s *emailService) Unsubscribe(ctx context.Context, token string) error {
	query := `
		UPDATE email_subscribers 
		SET status = $1, unsubscribed_at = $2, updated_at = $3
		WHERE unsubscribe_token = $4 AND status = $5`

	now := time.Now()
	result, err := s.db.ExecContext(ctx, query,
		models.SubscriberStatusUnsubscribed, now, now, token, models.SubscriberStatusConfirmed)

	if err != nil {
		return fmt.Errorf("failed to unsubscribe: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("invalid unsubscribe token or user not subscribed")
	}

	return nil
}

// GetSubscriber gets a subscriber by email
func (s *emailService) GetSubscriber(ctx context.Context, email string) (*models.EmailSubscriber, error) {
	query := `
		SELECT id, email, status, confirmation_token, unsubscribe_token, 
			first_name, last_name, preferences, source, confirmed_at, 
			unsubscribed_at, created_at, updated_at
		FROM email_subscribers 
		WHERE email = $1`

	subscriber := &models.EmailSubscriber{}
	var preferencesJSON []byte

	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&subscriber.ID, &subscriber.Email, &subscriber.Status,
		&subscriber.ConfirmationToken, &subscriber.UnsubscribeToken,
		&subscriber.FirstName, &subscriber.LastName, &preferencesJSON,
		&subscriber.Source, &subscriber.ConfirmedAt, &subscriber.UnsubscribedAt,
		&subscriber.CreatedAt, &subscriber.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if len(preferencesJSON) > 0 {
		json.Unmarshal(preferencesJSON, &subscriber.Preferences)
	}

	return subscriber, nil
}

// GetSubscribers gets subscribers by status with pagination
func (s *emailService) GetSubscribers(ctx context.Context, status models.SubscriberStatus, limit, offset int) ([]*models.EmailSubscriber, error) {
	query := `
		SELECT id, email, status, confirmation_token, unsubscribe_token, 
			first_name, last_name, preferences, source, confirmed_at, 
			unsubscribed_at, created_at, updated_at
		FROM email_subscribers 
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := s.db.QueryContext(ctx, query, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscribers: %w", err)
	}
	defer rows.Close()

	var subscribers []*models.EmailSubscriber
	for rows.Next() {
		subscriber := &models.EmailSubscriber{}
		var preferencesJSON []byte

		err := rows.Scan(
			&subscriber.ID, &subscriber.Email, &subscriber.Status,
			&subscriber.ConfirmationToken, &subscriber.UnsubscribeToken,
			&subscriber.FirstName, &subscriber.LastName, &preferencesJSON,
			&subscriber.Source, &subscriber.ConfirmedAt, &subscriber.UnsubscribedAt,
			&subscriber.CreatedAt, &subscriber.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan subscriber: %w", err)
		}

		if len(preferencesJSON) > 0 {
			json.Unmarshal(preferencesJSON, &subscriber.Preferences)
		}

		subscribers = append(subscribers, subscriber)
	}

	return subscribers, nil
}

// SendConfirmationEmail sends a confirmation email to a subscriber
func (s *emailService) SendConfirmationEmail(ctx context.Context, subscriber *models.EmailSubscriber) error {
	template, err := s.getTemplateByType(ctx, models.TemplateTypeConfirmation)
	if err != nil {
		return fmt.Errorf("failed to get confirmation template: %w", err)
	}

	confirmationURL := s.config.GetConfirmationURL(subscriber.ConfirmationToken)
	
	data := map[string]interface{}{
		"first_name":       subscriber.FirstName,
		"last_name":        subscriber.LastName,
		"email":           subscriber.Email,
		"confirmation_url": confirmationURL,
		"site_name":       s.config.FromName,
	}

	subject, err := s.renderTemplate(template.Subject, data)
	if err != nil {
		return fmt.Errorf("failed to render subject: %w", err)
	}

	htmlContent, err := s.renderTemplate(template.HTMLContent, data)
	if err != nil {
		return fmt.Errorf("failed to render HTML content: %w", err)
	}

	textContent, err := s.renderTemplate(template.TextContent, data)
	if err != nil {
		return fmt.Errorf("failed to render text content: %w", err)
	}

	return s.SendEmail(ctx, subscriber.Email, subject, htmlContent, textContent)
}

// SendWelcomeEmail sends a welcome email to a confirmed subscriber
func (s *emailService) SendWelcomeEmail(ctx context.Context, subscriber *models.EmailSubscriber) error {
	template, err := s.getTemplateByType(ctx, models.TemplateTypeWelcome)
	if err != nil {
		return fmt.Errorf("failed to get welcome template: %w", err)
	}

	unsubscribeURL := s.config.GetUnsubscribeURL(subscriber.UnsubscribeToken)
	
	data := map[string]interface{}{
		"first_name":      subscriber.FirstName,
		"last_name":       subscriber.LastName,
		"email":          subscriber.Email,
		"unsubscribe_url": unsubscribeURL,
		"site_name":      s.config.FromName,
	}

	subject, err := s.renderTemplate(template.Subject, data)
	if err != nil {
		return fmt.Errorf("failed to render subject: %w", err)
	}

	htmlContent, err := s.renderTemplate(template.HTMLContent, data)
	if err != nil {
		return fmt.Errorf("failed to render HTML content: %w", err)
	}

	textContent, err := s.renderTemplate(template.TextContent, data)
	if err != nil {
		return fmt.Errorf("failed to render text content: %w", err)
	}

	return s.SendEmail(ctx, subscriber.Email, subject, htmlContent, textContent)
}

// SendEmail sends a single email
func (s *emailService) SendEmail(ctx context.Context, to, subject, htmlContent, textContent string) error {
	if !s.config.Enabled {
		log.Printf("Email service disabled, would send email to %s with subject: %s", to, subject)
		return nil
	}

	message := &EmailMessage{
		To:          to,
		From:        s.config.FromEmail,
		FromName:    s.config.FromName,
		ReplyTo:     s.config.ReplyTo,
		Subject:     subject,
		HTMLContent: htmlContent,
		TextContent: textContent,
	}

	result, err := s.client.SendEmail(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	if result.Error != nil {
		return fmt.Errorf("email sending failed: %w", result.Error)
	}

	log.Printf("Email sent successfully to %s, message ID: %s", to, result.MessageID)
	return nil
}

// Helper functions

func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *emailService) renderTemplate(templateStr string, data map[string]interface{}) (string, error) {
	tmpl, err := template.New("email").Parse(templateStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (s *emailService) getTemplateByType(ctx context.Context, templateType models.TemplateType) (*models.EmailTemplate, error) {
	query := `
		SELECT id, name, subject, html_content, text_content, template_type, is_active, created_at, updated_at
		FROM email_templates 
		WHERE template_type = $1 AND is_active = true
		ORDER BY created_at DESC
		LIMIT 1`

	template := &models.EmailTemplate{}
	err := s.db.QueryRowContext(ctx, query, templateType).Scan(
		&template.ID, &template.Name, &template.Subject,
		&template.HTMLContent, &template.TextContent, &template.TemplateType,
		&template.IsActive, &template.CreatedAt, &template.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return template, nil
}

func (s *emailService) getSubscriberByToken(ctx context.Context, token, tokenField string) (*models.EmailSubscriber, error) {
	query := fmt.Sprintf(`
		SELECT id, email, status, confirmation_token, unsubscribe_token, 
			first_name, last_name, preferences, source, confirmed_at, 
			unsubscribed_at, created_at, updated_at
		FROM email_subscribers 
		WHERE %s = $1`, tokenField)

	subscriber := &models.EmailSubscriber{}
	var preferencesJSON []byte

	err := s.db.QueryRowContext(ctx, query, token).Scan(
		&subscriber.ID, &subscriber.Email, &subscriber.Status,
		&subscriber.ConfirmationToken, &subscriber.UnsubscribeToken,
		&subscriber.FirstName, &subscriber.LastName, &preferencesJSON,
		&subscriber.Source, &subscriber.ConfirmedAt, &subscriber.UnsubscribedAt,
		&subscriber.CreatedAt, &subscriber.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if len(preferencesJSON) > 0 {
		json.Unmarshal(preferencesJSON, &subscriber.Preferences)
	}

	return subscriber, nil
}

func (s *emailService) resubscribe(ctx context.Context, existing *models.EmailSubscriber, req *models.SubscribeRequest) (*models.EmailSubscriber, error) {
	// Generate new confirmation token
	confirmationToken, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate confirmation token: %w", err)
	}

	// Update subscriber
	query := `
		UPDATE email_subscribers 
		SET status = $1, confirmation_token = $2, first_name = $3, last_name = $4, 
			preferences = $5, source = $6, updated_at = $7
		WHERE id = $8`

	preferencesJSON, _ := json.Marshal(req.Preferences)
	now := time.Now()

	_, err = s.db.ExecContext(ctx, query,
		models.SubscriberStatusPending, confirmationToken, req.FirstName,
		req.LastName, preferencesJSON, req.Source, now, existing.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to resubscribe: %w", err)
	}

	// Update the existing subscriber object
	existing.Status = models.SubscriberStatusPending
	existing.ConfirmationToken = confirmationToken
	existing.FirstName = req.FirstName
	existing.LastName = req.LastName
	existing.Preferences = req.Preferences
	existing.Source = req.Source
	existing.UpdatedAt = now

	// Send confirmation email
	if err := s.SendConfirmationEmail(ctx, existing); err != nil {
		log.Printf("Failed to send confirmation email to %s: %v", existing.Email, err)
	}

	return existing, nil
}

// UpdateSubscriberPreferences updates subscriber preferences
func (s *emailService) UpdateSubscriberPreferences(ctx context.Context, email string, preferences map[string]interface{}) error {
	preferencesJSON, err := json.Marshal(preferences)
	if err != nil {
		return fmt.Errorf("failed to marshal preferences: %w", err)
	}

	query := `
		UPDATE email_subscribers 
		SET preferences = $1, updated_at = $2
		WHERE email = $3 AND status = $4`

	result, err := s.db.ExecContext(ctx, query, preferencesJSON, time.Now(), email, models.SubscriberStatusConfirmed)
	if err != nil {
		return fmt.Errorf("failed to update preferences: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("subscriber not found or not confirmed")
	}

	return nil
}

// CreateCampaign creates a new email campaign
func (s *emailService) CreateCampaign(ctx context.Context, req *models.CreateCampaignRequest, userID uint64) (*models.EmailCampaign, error) {
	campaign := &models.EmailCampaign{
		Name:        req.Name,
		Subject:     req.Subject,
		TemplateID:  req.TemplateID,
		Content:     req.Content,
		Status:      models.CampaignStatusDraft,
		ScheduledAt: req.ScheduledAt,
		CreatedBy:   userID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	query := `
		INSERT INTO email_campaigns (name, subject, template_id, content, status, scheduled_at, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`

	err := s.db.QueryRowContext(ctx, query,
		campaign.Name, campaign.Subject, campaign.TemplateID, campaign.Content,
		campaign.Status, campaign.ScheduledAt, campaign.CreatedBy,
		campaign.CreatedAt, campaign.UpdatedAt,
	).Scan(&campaign.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to create campaign: %w", err)
	}

	return campaign, nil
}

// GetCampaign gets a campaign by ID
func (s *emailService) GetCampaign(ctx context.Context, id uint64) (*models.EmailCampaign, error) {
	query := `
		SELECT id, name, subject, template_id, content, status, scheduled_at, sent_at,
			recipient_count, sent_count, delivered_count, opened_count, clicked_count,
			bounced_count, unsubscribed_count, created_by, created_at, updated_at
		FROM email_campaigns 
		WHERE id = $1`

	campaign := &models.EmailCampaign{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&campaign.ID, &campaign.Name, &campaign.Subject, &campaign.TemplateID,
		&campaign.Content, &campaign.Status, &campaign.ScheduledAt, &campaign.SentAt,
		&campaign.RecipientCount, &campaign.SentCount, &campaign.DeliveredCount,
		&campaign.OpenedCount, &campaign.ClickedCount, &campaign.BouncedCount,
		&campaign.UnsubscribedCount, &campaign.CreatedBy, &campaign.CreatedAt, &campaign.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return campaign, nil
}

// GetCampaigns gets campaigns with pagination
func (s *emailService) GetCampaigns(ctx context.Context, limit, offset int) ([]*models.EmailCampaign, error) {
	query := `
		SELECT id, name, subject, template_id, content, status, scheduled_at, sent_at,
			recipient_count, sent_count, delivered_count, opened_count, clicked_count,
			bounced_count, unsubscribed_count, created_by, created_at, updated_at
		FROM email_campaigns 
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := s.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get campaigns: %w", err)
	}
	defer rows.Close()

	var campaigns []*models.EmailCampaign
	for rows.Next() {
		campaign := &models.EmailCampaign{}
		err := rows.Scan(
			&campaign.ID, &campaign.Name, &campaign.Subject, &campaign.TemplateID,
			&campaign.Content, &campaign.Status, &campaign.ScheduledAt, &campaign.SentAt,
			&campaign.RecipientCount, &campaign.SentCount, &campaign.DeliveredCount,
			&campaign.OpenedCount, &campaign.ClickedCount, &campaign.BouncedCount,
			&campaign.UnsubscribedCount, &campaign.CreatedBy, &campaign.CreatedAt, &campaign.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan campaign: %w", err)
		}
		campaigns = append(campaigns, campaign)
	}

	return campaigns, nil
}

// SendBulkEmails sends multiple emails with rate limiting
func (s *emailService) SendBulkEmails(ctx context.Context, emails []BulkEmail) error {
	if !s.config.Enabled {
		log.Printf("Email service disabled, would send %d bulk emails", len(emails))
		return nil
	}

	// Convert to EmailMessage slice
	var messages []*EmailMessage
	for _, email := range emails {
		message := &EmailMessage{
			To:          email.To,
			From:        s.config.FromEmail,
			FromName:    s.config.FromName,
			ReplyTo:     s.config.ReplyTo,
			Subject:     email.Subject,
			HTMLContent: email.HTMLContent,
			TextContent: email.TextContent,
			Metadata:    email.Metadata,
		}
		messages = append(messages, message)
	}

	// Send in batches to respect rate limits
	batchSize := s.config.BatchSize
	if batchSize <= 0 {
		batchSize = 1000
	}

	for i := 0; i < len(messages); i += batchSize {
		end := i + batchSize
		if end > len(messages) {
			end = len(messages)
		}

		batch := messages[i:end]
		results, err := s.client.SendBulkEmails(ctx, batch)
		if err != nil {
			return fmt.Errorf("failed to send batch %d-%d: %w", i, end-1, err)
		}

		// Log results
		for j, result := range results {
			if result.Error != nil {
				log.Printf("Failed to send email to %s: %v", batch[j].To, result.Error)
			} else {
				log.Printf("Email sent to %s, message ID: %s", batch[j].To, result.MessageID)
			}
		}

		// Rate limiting delay between batches
		if end < len(messages) {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Second): // 1 second delay between batches
			}
		}
	}

	return nil
}

// SendCampaign sends a campaign to all confirmed subscribers
func (s *emailService) SendCampaign(ctx context.Context, id uint64) error {
	// Get campaign
	campaign, err := s.GetCampaign(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get campaign: %w", err)
	}

	if campaign.Status != models.CampaignStatusDraft && campaign.Status != models.CampaignStatusScheduled {
		return fmt.Errorf("campaign cannot be sent in status: %s", campaign.Status)
	}

	// Update campaign status to sending
	_, err = s.db.ExecContext(ctx, 
		"UPDATE email_campaigns SET status = $1, updated_at = $2 WHERE id = $3",
		models.CampaignStatusSending, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update campaign status: %w", err)
	}

	// Get all confirmed subscribers
	subscribers, err := s.GetSubscribers(ctx, models.SubscriberStatusConfirmed, 100000, 0)
	if err != nil {
		return fmt.Errorf("failed to get subscribers: %w", err)
	}

	// Create bulk emails
	var emails []BulkEmail
	for _, subscriber := range subscribers {
		// Personalize content
		personalizedSubject := s.personalizeContent(campaign.Subject, subscriber)
		personalizedContent := s.personalizeContent(campaign.Content, subscriber)

		email := BulkEmail{
			To:          subscriber.Email,
			Subject:     personalizedSubject,
			HTMLContent: personalizedContent,
			TextContent: stripHTMLTags(personalizedContent),
			Metadata: map[string]string{
				"campaign_id":   fmt.Sprintf("%d", campaign.ID),
				"subscriber_id": fmt.Sprintf("%d", subscriber.ID),
			},
		}
		emails = append(emails, email)
	}

	// Send bulk emails
	if err := s.SendBulkEmails(ctx, emails); err != nil {
		// Update campaign status to failed
		s.db.ExecContext(ctx, 
			"UPDATE email_campaigns SET status = $1, updated_at = $2 WHERE id = $3",
			models.CampaignStatusDraft, time.Now(), id)
		return fmt.Errorf("failed to send campaign emails: %w", err)
	}

	// Update campaign status to sent
	now := time.Now()
	_, err = s.db.ExecContext(ctx, 
		"UPDATE email_campaigns SET status = $1, sent_at = $2, recipient_count = $3, updated_at = $4 WHERE id = $5",
		models.CampaignStatusSent, now, len(emails), now, id)
	if err != nil {
		return fmt.Errorf("failed to update campaign completion: %w", err)
	}

	log.Printf("Campaign %d sent to %d subscribers", id, len(emails))
	return nil
}

// GetEmailStats gets email service statistics
func (s *emailService) GetEmailStats(ctx context.Context) (*models.EmailStats, error) {
	stats := &models.EmailStats{}

	// Get subscriber counts
	err := s.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'confirmed') as active
		FROM email_subscribers
	`).Scan(&stats.TotalSubscribers, &stats.ActiveSubscribers)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscriber stats: %w", err)
	}

	// Get campaign count
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM email_campaigns
	`).Scan(&stats.TotalCampaigns)
	if err != nil {
		return nil, fmt.Errorf("failed to get campaign count: %w", err)
	}

	// Get emails sent today and this week
	today := time.Now().Truncate(24 * time.Hour)
	weekAgo := today.AddDate(0, 0, -7)

	err = s.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*) FILTER (WHERE sent_at >= $1) as today,
			COUNT(*) FILTER (WHERE sent_at >= $2) as week
		FROM email_sends 
		WHERE status = 'sent'
	`, today, weekAgo).Scan(&stats.EmailsSentToday, &stats.EmailsSentThisWeek)
	if err != nil {
		return nil, fmt.Errorf("failed to get send stats: %w", err)
	}

	// Calculate rates
	err = s.db.QueryRowContext(ctx, `
		SELECT 
			COALESCE(AVG(CASE WHEN sent_count > 0 THEN opened_count::float / sent_count END), 0) as open_rate,
			COALESCE(AVG(CASE WHEN sent_count > 0 THEN clicked_count::float / sent_count END), 0) as click_rate,
			COALESCE(AVG(CASE WHEN sent_count > 0 THEN bounced_count::float / sent_count END), 0) as bounce_rate
		FROM email_campaigns 
		WHERE status = 'sent' AND sent_count > 0
	`).Scan(&stats.AverageOpenRate, &stats.AverageClickRate, &stats.BounceRate)
	if err != nil {
		return nil, fmt.Errorf("failed to get rate stats: %w", err)
	}

	return stats, nil
}

// HandleBounce handles email bounces
func (s *emailService) HandleBounce(ctx context.Context, email string, reason string) error {
	// Update subscriber status to bounced
	query := `
		UPDATE email_subscribers 
		SET status = $1, updated_at = $2
		WHERE email = $3 AND status = $4`

	_, err := s.db.ExecContext(ctx, query,
		models.SubscriberStatusBounced, time.Now(), email, models.SubscriberStatusConfirmed)

	if err != nil {
		return fmt.Errorf("failed to handle bounce: %w", err)
	}

	log.Printf("Email %s marked as bounced: %s", email, reason)
	return nil
}

// HandleComplaint handles spam complaints
func (s *emailService) HandleComplaint(ctx context.Context, email string) error {
	// Unsubscribe user due to complaint
	query := `
		UPDATE email_subscribers 
		SET status = $1, unsubscribed_at = $2, updated_at = $3
		WHERE email = $4 AND status = $5`

	now := time.Now()
	_, err := s.db.ExecContext(ctx, query,
		models.SubscriberStatusUnsubscribed, now, now, email, models.SubscriberStatusConfirmed)

	if err != nil {
		return fmt.Errorf("failed to handle complaint: %w", err)
	}

	log.Printf("Email %s unsubscribed due to spam complaint", email)
	return nil
}

// Helper methods

func (s *emailService) personalizeContent(content string, subscriber *models.EmailSubscriber) string {
	// Replace common placeholders
	content = strings.ReplaceAll(content, "{{first_name}}", subscriber.FirstName)
	content = strings.ReplaceAll(content, "{{last_name}}", subscriber.LastName)
	content = strings.ReplaceAll(content, "{{email}}", subscriber.Email)
	content = strings.ReplaceAll(content, "{{unsubscribe_url}}", s.config.GetUnsubscribeURL(subscriber.UnsubscribeToken))
	content = strings.ReplaceAll(content, "{{site_name}}", s.config.FromName)
	return content
}

func stripHTMLTags(html string) string {
	// Simple HTML tag removal - in production, use a proper HTML parser
	// This is a basic implementation
	result := html
	result = strings.ReplaceAll(result, "<br>", "\n")
	result = strings.ReplaceAll(result, "<br/>", "\n")
	result = strings.ReplaceAll(result, "<p>", "")
	result = strings.ReplaceAll(result, "</p>", "\n\n")
	result = strings.ReplaceAll(result, "<h1>", "")
	result = strings.ReplaceAll(result, "</h1>", "\n\n")
	result = strings.ReplaceAll(result, "<h2>", "")
	result = strings.ReplaceAll(result, "</h2>", "\n\n")
	// Add more tag replacements as needed
	return result
}

// Placeholder implementations for remaining methods
func (s *emailService) UpdateCampaign(ctx context.Context, id uint64, updates map[string]interface{}) error {
	// Implementation would update campaign fields
	return fmt.Errorf("not implemented")
}

func (s *emailService) DeleteCampaign(ctx context.Context, id uint64) error {
	// Implementation would delete campaign
	return fmt.Errorf("not implemented")
}

func (s *emailService) ScheduleCampaign(ctx context.Context, id uint64, scheduledAt time.Time) error {
	// Implementation would schedule campaign
	return fmt.Errorf("not implemented")
}

func (s *emailService) CreateTemplate(ctx context.Context, template *models.EmailTemplate) (*models.EmailTemplate, error) {
	// Implementation would create email template
	return nil, fmt.Errorf("not implemented")
}

func (s *emailService) GetTemplate(ctx context.Context, id uint64) (*models.EmailTemplate, error) {
	// Implementation would get template by ID
	return nil, fmt.Errorf("not implemented")
}

func (s *emailService) GetTemplates(ctx context.Context, templateType models.TemplateType) ([]*models.EmailTemplate, error) {
	// Implementation would get templates by type
	return nil, fmt.Errorf("not implemented")
}

func (s *emailService) UpdateTemplate(ctx context.Context, id uint64, updates map[string]interface{}) error {
	// Implementation would update template
	return fmt.Errorf("not implemented")
}

func (s *emailService) DeleteTemplate(ctx context.Context, id uint64) error {
	// Implementation would delete template
	return fmt.Errorf("not implemented")
}

func (s *emailService) HandleWebhook(ctx context.Context, provider string, eventType string, payload []byte) error {
	// Implementation would handle webhooks from email providers
	return fmt.Errorf("not implemented")
}

func (s *emailService) GetCampaignStats(ctx context.Context, campaignID uint64) (*CampaignStats, error) {
	// Implementation would get campaign statistics
	return nil, fmt.Errorf("not implemented")
}