package models

import (
	"time"
)

// PushSubscription represents a user's push notification subscription
type PushSubscription struct {
	ID         uint64    `json:"id" db:"id"`
	UserID     *uint64   `json:"user_id,omitempty" db:"user_id"` // Optional for anonymous users
	Endpoint   string    `json:"endpoint" db:"endpoint"`
	P256DH     string    `json:"p256dh" db:"p256dh"`
	Auth       string    `json:"auth" db:"auth"`
	UserAgent  string    `json:"user_agent" db:"user_agent"`
	IPAddress  string    `json:"ip_address" db:"ip_address"`
	IsActive   bool      `json:"is_active" db:"is_active"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// PushNotification represents a push notification to be sent
type PushNotification struct {
	ID            uint64                 `json:"id" db:"id"`
	Title         string                 `json:"title" db:"title"`
	Body          string                 `json:"body" db:"body"`
	Icon          string                 `json:"icon" db:"icon"`
	Badge         string                 `json:"badge" db:"badge"`
	Image         string                 `json:"image" db:"image"`
	URL           string                 `json:"url" db:"url"`
	Data          map[string]interface{} `json:"data" db:"data"`
	TargetType    string                 `json:"target_type" db:"target_type"` // all, category, tag, user
	TargetValue   string                 `json:"target_value" db:"target_value"`
	ScheduledAt   *time.Time             `json:"scheduled_at" db:"scheduled_at"`
	SentAt        *time.Time             `json:"sent_at" db:"sent_at"`
	Status        string                 `json:"status" db:"status"` // pending, sending, sent, failed
	TotalSent     int                    `json:"total_sent" db:"total_sent"`
	TotalDelivered int                   `json:"total_delivered" db:"total_delivered"`
	TotalClicked  int                    `json:"total_clicked" db:"total_clicked"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at" db:"updated_at"`
}

// PushDelivery tracks individual notification deliveries
type PushDelivery struct {
	ID               uint64    `json:"id" db:"id"`
	NotificationID   uint64    `json:"notification_id" db:"notification_id"`
	SubscriptionID   uint64    `json:"subscription_id" db:"subscription_id"`
	Status           string    `json:"status" db:"status"` // sent, delivered, failed, clicked
	ErrorMessage     string    `json:"error_message" db:"error_message"`
	DeliveredAt      *time.Time `json:"delivered_at" db:"delivered_at"`
	ClickedAt        *time.Time `json:"clicked_at" db:"clicked_at"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// PushTemplate represents a notification template
type PushTemplate struct {
	ID          uint64    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Title       string    `json:"title" db:"title"`
	Body        string    `json:"body" db:"body"`
	Icon        string    `json:"icon" db:"icon"`
	Badge       string    `json:"badge" db:"badge"`
	Image       string    `json:"image" db:"image"`
	URL         string    `json:"url" db:"url"`
	Variables   []string  `json:"variables" db:"variables"` // Template variables like {{title}}, {{author}}
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// NotificationPreference represents user notification preferences
type NotificationPreference struct {
	ID                uint64    `json:"id" db:"id"`
	UserID            *uint64   `json:"user_id,omitempty" db:"user_id"`
	SubscriptionID    uint64    `json:"subscription_id" db:"subscription_id"`
	BreakingNews      bool      `json:"breaking_news" db:"breaking_news"`
	CategoryUpdates   bool      `json:"category_updates" db:"category_updates"`
	TagUpdates        bool      `json:"tag_updates" db:"tag_updates"`
	AuthorUpdates     bool      `json:"author_updates" db:"author_updates"`
	PreferredCategories []uint64 `json:"preferred_categories" db:"preferred_categories"`
	PreferredTags     []uint64  `json:"preferred_tags" db:"preferred_tags"`
	PreferredAuthors  []uint64  `json:"preferred_authors" db:"preferred_authors"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// Validation constants
const (
	NotificationStatusPending = "pending"
	NotificationStatusSending = "sending"
	NotificationStatusSent    = "sent"
	NotificationStatusFailed  = "failed"

	DeliveryStatusSent      = "sent"
	DeliveryStatusDelivered = "delivered"
	DeliveryStatusFailed    = "failed"
	DeliveryStatusClicked   = "clicked"

	TargetTypeAll      = "all"
	TargetTypeCategory = "category"
	TargetTypeTag      = "tag"
	TargetTypeUser     = "user"
)

// Validate validates the push notification data
func (pn *PushNotification) Validate() error {
	if pn.Title == "" {
		return NewValidationError("title is required")
	}
	if pn.Body == "" {
		return NewValidationError("body is required")
	}
	if pn.TargetType == "" {
		pn.TargetType = TargetTypeAll
	}
	if pn.Status == "" {
		pn.Status = NotificationStatusPending
	}
	return nil
}

// Validate validates the push subscription data
func (ps *PushSubscription) Validate() error {
	if ps.Endpoint == "" {
		return NewValidationError("endpoint is required")
	}
	if ps.P256DH == "" {
		return NewValidationError("p256dh is required")
	}
	if ps.Auth == "" {
		return NewValidationError("auth is required")
	}
	return nil
}