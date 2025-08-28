package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// SocialMediaPlatform represents different social media platforms
type SocialMediaPlatform string

const (
	PlatformFacebook  SocialMediaPlatform = "facebook"
	PlatformTelegram  SocialMediaPlatform = "telegram"
	PlatformTwitter   SocialMediaPlatform = "twitter"
	PlatformInstagram SocialMediaPlatform = "instagram"
	PlatformLinkedIn  SocialMediaPlatform = "linkedin"
)

// SocialMediaCredentials stores encrypted credentials for social media platforms
type SocialMediaCredentials struct {
	ID           uint64              `json:"id" db:"id"`
	Platform     SocialMediaPlatform `json:"platform" db:"platform" validate:"required"`
	Name         string              `json:"name" db:"name" validate:"required,max=100"`
	Credentials  EncryptedData       `json:"credentials" db:"credentials"`
	IsActive     bool                `json:"is_active" db:"is_active"`
	LastRotated  *time.Time          `json:"last_rotated" db:"last_rotated"`
	CreatedAt    time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at" db:"updated_at"`
}

// EncryptedData represents encrypted credential data
type EncryptedData struct {
	Data      string `json:"data"`      // Encrypted credential data
	Algorithm string `json:"algorithm"` // Encryption algorithm used
	KeyID     string `json:"key_id"`    // Key identifier for rotation
}

// Scan implements the sql.Scanner interface for EncryptedData
func (e *EncryptedData) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, e)
	case string:
		return json.Unmarshal([]byte(v), e)
	default:
		return fmt.Errorf("cannot scan %T into EncryptedData", value)
	}
}

// Value implements the driver.Valuer interface for EncryptedData
func (e EncryptedData) Value() (driver.Value, error) {
	return json.Marshal(e)
}

// SocialMediaPost represents a post made to social media platforms
type SocialMediaPost struct {
	ID           uint64              `json:"id" db:"id"`
	ArticleID    uint64              `json:"article_id" db:"article_id" validate:"required"`
	Platform     SocialMediaPlatform `json:"platform" db:"platform" validate:"required"`
	CredentialID uint64              `json:"credential_id" db:"credential_id" validate:"required"`
	PostID       string              `json:"post_id" db:"post_id"`        // Platform-specific post ID
	Status       PostStatus          `json:"status" db:"status"`
	Content      PostContent         `json:"content" db:"content"`
	ScheduledAt  *time.Time          `json:"scheduled_at" db:"scheduled_at"`
	PostedAt     *time.Time          `json:"posted_at" db:"posted_at"`
	Attempts     int                 `json:"attempts" db:"attempts"`
	MaxAttempts  int                 `json:"max_attempts" db:"max_attempts"`
	LastError    string              `json:"last_error" db:"last_error"`
	CreatedAt    time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at" db:"updated_at"`
}

// PostStatus represents the status of a social media post
type PostStatus string

const (
	PostStatusPending   PostStatus = "pending"
	PostStatusScheduled PostStatus = "scheduled"
	PostStatusPosted    PostStatus = "posted"
	PostStatusFailed    PostStatus = "failed"
	PostStatusRetrying  PostStatus = "retrying"
)

// PostContent contains platform-specific content for social media posts
type PostContent struct {
	Text        string            `json:"text"`
	ImageURL    string            `json:"image_url,omitempty"`
	VideoURL    string            `json:"video_url,omitempty"`
	LinkURL     string            `json:"link_url"`
	Hashtags    []string          `json:"hashtags,omitempty"`
	Mentions    []string          `json:"mentions,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"` // Platform-specific metadata
}

// Scan implements the sql.Scanner interface for PostContent
func (p *PostContent) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, p)
	case string:
		return json.Unmarshal([]byte(v), p)
	default:
		return fmt.Errorf("cannot scan %T into PostContent", value)
	}
}

// Value implements the driver.Valuer interface for PostContent
func (p PostContent) Value() (driver.Value, error) {
	return json.Marshal(p)
}

// SocialMediaWebhook represents webhook events from social media platforms
type SocialMediaWebhook struct {
	ID           uint64              `json:"id" db:"id"`
	Platform     SocialMediaPlatform `json:"platform" db:"platform" validate:"required"`
	EventType    string              `json:"event_type" db:"event_type" validate:"required"`
	PostID       string              `json:"post_id" db:"post_id"`
	Payload      WebhookPayload      `json:"payload" db:"payload"`
	Signature    string              `json:"signature" db:"signature"`
	Verified     bool                `json:"verified" db:"verified"`
	Processed    bool                `json:"processed" db:"processed"`
	ProcessedAt  *time.Time          `json:"processed_at" db:"processed_at"`
	CreatedAt    time.Time           `json:"created_at" db:"created_at"`
}

// WebhookPayload contains the raw webhook payload data
type WebhookPayload struct {
	Data      map[string]interface{} `json:"data"`
	Headers   map[string]string      `json:"headers"`
	Timestamp time.Time              `json:"timestamp"`
}

// Scan implements the sql.Scanner interface for WebhookPayload
func (w *WebhookPayload) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, w)
	case string:
		return json.Unmarshal([]byte(v), w)
	default:
		return fmt.Errorf("cannot scan %T into WebhookPayload", value)
	}
}

// Value implements the driver.Valuer interface for WebhookPayload
func (w WebhookPayload) Value() (driver.Value, error) {
	return json.Marshal(w)
}

// FacebookInstantArticle represents Facebook Instant Article specific data
type FacebookInstantArticle struct {
	ArticleID     uint64    `json:"article_id" db:"article_id"`
	InstantID     string    `json:"instant_id" db:"instant_id"`
	Status        string    `json:"status" db:"status"`
	HTML          string    `json:"html" db:"html"`
	PublishedURL  string    `json:"published_url" db:"published_url"`
	LastSynced    time.Time `json:"last_synced" db:"last_synced"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// TelegramChannel represents Telegram channel configuration
type TelegramChannel struct {
	ID          uint64    `json:"id" db:"id"`
	ChannelID   string    `json:"channel_id" db:"channel_id" validate:"required"`
	ChannelName string    `json:"channel_name" db:"channel_name" validate:"required"`
	BotToken    string    `json:"bot_token" db:"bot_token" validate:"required"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// ValidateSocialMediaCredentials validates social media credentials
func ValidateSocialMediaCredentials(creds *SocialMediaCredentials) error {
	var errors []string

	// Platform validation
	validPlatforms := map[SocialMediaPlatform]bool{
		PlatformFacebook:  true,
		PlatformTelegram:  true,
		PlatformTwitter:   true,
		PlatformInstagram: true,
		PlatformLinkedIn:  true,
	}
	if !validPlatforms[creds.Platform] {
		errors = append(errors, "invalid platform")
	}

	// Name validation
	if len(creds.Name) == 0 {
		errors = append(errors, "name is required")
	}
	if len(creds.Name) > 100 {
		errors = append(errors, "name must be less than 100 characters")
	}

	// Credentials validation
	if creds.Credentials.Data == "" {
		errors = append(errors, "credentials data is required")
	}

	if len(errors) > 0 {
		return &ValidationError{
			Message: "Social media credentials validation failed",
			Fields:  errors,
		}
	}

	return nil
}

// ValidateSocialMediaPost validates social media post data
func ValidateSocialMediaPost(post *SocialMediaPost) error {
	var errors []string

	// Article ID validation
	if post.ArticleID == 0 {
		errors = append(errors, "article_id is required")
	}

	// Platform validation
	validPlatforms := map[SocialMediaPlatform]bool{
		PlatformFacebook:  true,
		PlatformTelegram:  true,
		PlatformTwitter:   true,
		PlatformInstagram: true,
		PlatformLinkedIn:  true,
	}
	if !validPlatforms[post.Platform] {
		errors = append(errors, "invalid platform")
	}

	// Credential ID validation
	if post.CredentialID == 0 {
		errors = append(errors, "credential_id is required")
	}

	// Status validation
	validStatuses := map[PostStatus]bool{
		PostStatusPending:   true,
		PostStatusScheduled: true,
		PostStatusPosted:    true,
		PostStatusFailed:    true,
		PostStatusRetrying:  true,
	}
	if post.Status != "" && !validStatuses[post.Status] {
		errors = append(errors, "invalid status")
	}

	// Content validation
	if post.Content.Text == "" && post.Content.ImageURL == "" && post.Content.VideoURL == "" {
		errors = append(errors, "post must have text, image, or video content")
	}

	// Max attempts validation
	if post.MaxAttempts <= 0 {
		post.MaxAttempts = 3 // Default to 3 attempts
	}

	if len(errors) > 0 {
		return &ValidationError{
			Message: "Social media post validation failed",
			Fields:  errors,
		}
	}

	return nil
}