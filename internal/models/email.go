package models

import (
	"time"
)

// EmailSubscriber represents a newsletter subscriber
type EmailSubscriber struct {
	ID                uint64                 `json:"id" db:"id"`
	Email             string                 `json:"email" db:"email"`
	Status            SubscriberStatus       `json:"status" db:"status"`
	ConfirmationToken string                 `json:"-" db:"confirmation_token"`
	UnsubscribeToken  string                 `json:"-" db:"unsubscribe_token"`
	FirstName         string                 `json:"first_name" db:"first_name"`
	LastName          string                 `json:"last_name" db:"last_name"`
	Preferences       map[string]interface{} `json:"preferences" db:"preferences"`
	Source            string                 `json:"source" db:"source"`
	ConfirmedAt       *time.Time             `json:"confirmed_at" db:"confirmed_at"`
	UnsubscribedAt    *time.Time             `json:"unsubscribed_at" db:"unsubscribed_at"`
	CreatedAt         time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at" db:"updated_at"`
}

// SubscriberStatus represents the status of an email subscriber
type SubscriberStatus string

const (
	SubscriberStatusPending      SubscriberStatus = "pending"
	SubscriberStatusConfirmed    SubscriberStatus = "confirmed"
	SubscriberStatusUnsubscribed SubscriberStatus = "unsubscribed"
	SubscriberStatusBounced      SubscriberStatus = "bounced"
)

// EmailCampaign represents an email campaign
type EmailCampaign struct {
	ID                uint64         `json:"id" db:"id"`
	Name              string         `json:"name" db:"name"`
	Subject           string         `json:"subject" db:"subject"`
	TemplateID        *uint64        `json:"template_id" db:"template_id"`
	Content           string         `json:"content" db:"content"`
	Status            CampaignStatus `json:"status" db:"status"`
	ScheduledAt       *time.Time     `json:"scheduled_at" db:"scheduled_at"`
	SentAt            *time.Time     `json:"sent_at" db:"sent_at"`
	RecipientCount    int            `json:"recipient_count" db:"recipient_count"`
	SentCount         int            `json:"sent_count" db:"sent_count"`
	DeliveredCount    int            `json:"delivered_count" db:"delivered_count"`
	OpenedCount       int            `json:"opened_count" db:"opened_count"`
	ClickedCount      int            `json:"clicked_count" db:"clicked_count"`
	BouncedCount      int            `json:"bounced_count" db:"bounced_count"`
	UnsubscribedCount int            `json:"unsubscribed_count" db:"unsubscribed_count"`
	CreatedBy         uint64         `json:"created_by" db:"created_by"`
	CreatedAt         time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at" db:"updated_at"`
}

// CampaignStatus represents the status of an email campaign
type CampaignStatus string

const (
	CampaignStatusDraft     CampaignStatus = "draft"
	CampaignStatusScheduled CampaignStatus = "scheduled"
	CampaignStatusSending   CampaignStatus = "sending"
	CampaignStatusSent      CampaignStatus = "sent"
	CampaignStatusCancelled CampaignStatus = "cancelled"
)

// EmailTemplate represents an email template
type EmailTemplate struct {
	ID           uint64       `json:"id" db:"id"`
	Name         string       `json:"name" db:"name"`
	Subject      string       `json:"subject" db:"subject"`
	HTMLContent  string       `json:"html_content" db:"html_content"`
	TextContent  string       `json:"text_content" db:"text_content"`
	TemplateType TemplateType `json:"template_type" db:"template_type"`
	IsActive     bool         `json:"is_active" db:"is_active"`
	CreatedAt    time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at" db:"updated_at"`
}

// TemplateType represents the type of email template
type TemplateType string

const (
	TemplateTypeNewsletter    TemplateType = "newsletter"
	TemplateTypeWelcome       TemplateType = "welcome"
	TemplateTypeConfirmation  TemplateType = "confirmation"
	TemplateTypeNotification  TemplateType = "notification"
)

// EmailSend represents an individual email send record
type EmailSend struct {
	ID             uint64     `json:"id" db:"id"`
	CampaignID     uint64     `json:"campaign_id" db:"campaign_id"`
	SubscriberID   uint64     `json:"subscriber_id" db:"subscriber_id"`
	Email          string     `json:"email" db:"email"`
	Status         SendStatus `json:"status" db:"status"`
	ExternalID     string     `json:"external_id" db:"external_id"`
	SentAt         *time.Time `json:"sent_at" db:"sent_at"`
	DeliveredAt    *time.Time `json:"delivered_at" db:"delivered_at"`
	OpenedAt       *time.Time `json:"opened_at" db:"opened_at"`
	ClickedAt      *time.Time `json:"clicked_at" db:"clicked_at"`
	BouncedAt      *time.Time `json:"bounced_at" db:"bounced_at"`
	BounceReason   string     `json:"bounce_reason" db:"bounce_reason"`
	UnsubscribedAt *time.Time `json:"unsubscribed_at" db:"unsubscribed_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
}

// SendStatus represents the status of an email send
type SendStatus string

const (
	SendStatusPending   SendStatus = "pending"
	SendStatusSent      SendStatus = "sent"
	SendStatusDelivered SendStatus = "delivered"
	SendStatusBounced   SendStatus = "bounced"
	SendStatusFailed    SendStatus = "failed"
)

// SubscribeRequest represents a subscription request
type SubscribeRequest struct {
	Email       string                 `json:"email" validate:"required,email"`
	FirstName   string                 `json:"first_name" validate:"max=100"`
	LastName    string                 `json:"last_name" validate:"max=100"`
	Preferences map[string]interface{} `json:"preferences"`
	Source      string                 `json:"source"`
}

// CreateCampaignRequest represents a campaign creation request
type CreateCampaignRequest struct {
	Name        string     `json:"name" validate:"required,max=255"`
	Subject     string     `json:"subject" validate:"required,max=255"`
	TemplateID  *uint64    `json:"template_id"`
	Content     string     `json:"content" validate:"required"`
	ScheduledAt *time.Time `json:"scheduled_at"`
}

// EmailStats represents email campaign statistics
type EmailStats struct {
	TotalSubscribers   int     `json:"total_subscribers"`
	ActiveSubscribers  int     `json:"active_subscribers"`
	TotalCampaigns     int     `json:"total_campaigns"`
	EmailsSentToday    int     `json:"emails_sent_today"`
	EmailsSentThisWeek int     `json:"emails_sent_this_week"`
	AverageOpenRate    float64 `json:"average_open_rate"`
	AverageClickRate   float64 `json:"average_click_rate"`
	BounceRate         float64 `json:"bounce_rate"`
}