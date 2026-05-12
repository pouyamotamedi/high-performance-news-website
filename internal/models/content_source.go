package models

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"
)

// ContentSource represents an external content source
type ContentSource struct {
	ID          uint64    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name" validate:"required,max=100"`
	Type        string    `json:"type" db:"type" validate:"required,oneof=api webhook manual"`
	APIKey      string    `json:"api_key,omitempty" db:"api_key"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	RateLimit   int       `json:"rate_limit" db:"rate_limit"` // requests per minute
	Priority    int       `json:"priority" db:"priority"`     // 1-10, higher is more important
	Config      SourceConfig `json:"config" db:"config"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// SourceConfig contains source-specific configuration
type SourceConfig struct {
	AutoPublish      bool     `json:"auto_publish"`
	DefaultCategoryID uint64  `json:"default_category_id"`
	DefaultAuthorID   uint64  `json:"default_author_id"`
	AllowedDomains   []string `json:"allowed_domains,omitempty"`
	RequiredFields   []string `json:"required_fields,omitempty"`
	TagMappings      map[string]uint64 `json:"tag_mappings,omitempty"`
}

// IngestedContent represents content received from external sources
type IngestedContent struct {
	ID              uint64    `json:"id" db:"id"`
	SourceID        uint64    `json:"source_id" db:"source_id"`
	ExternalID      string    `json:"external_id" db:"external_id"`
	Title           string    `json:"title" db:"title"`
	Content         string    `json:"content" db:"content"`
	Excerpt         string    `json:"excerpt" db:"excerpt"`
	AuthorName      string    `json:"author_name" db:"author_name"`
	AuthorEmail     string    `json:"author_email" db:"author_email"`
	CategoryName    string    `json:"category_name" db:"category_name"`
	Tags            []string  `json:"tags" db:"tags"`
	PublishedAt     *time.Time `json:"published_at" db:"published_at"`
	SourceURL       string    `json:"source_url" db:"source_url"`
	ContentHash     string    `json:"content_hash" db:"content_hash"`
	Status          string    `json:"status" db:"status"` // pending, processed, rejected, duplicate
	ProcessedAt     *time.Time `json:"processed_at" db:"processed_at"`
	ArticleID       *uint64   `json:"article_id" db:"article_id"`
	RejectionReason string    `json:"rejection_reason" db:"rejection_reason"`
	Metadata        map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// ContentIngestionRequest represents a request to ingest content
type ContentIngestionRequest struct {
	ExternalID        string                 `json:"external_id" validate:"required,max=255"`
	Title             string                 `json:"title" validate:"required,max=255"`
	Content           string                 `json:"content" validate:"required"`
	Excerpt           string                 `json:"excerpt,omitempty" validate:"max=500"`
	AuthorName        string                 `json:"author_name,omitempty" validate:"max=100"`
	AuthorEmail       string                 `json:"author_email,omitempty" validate:"omitempty,email,max=255"`
	CategoryName      string                 `json:"category_name,omitempty" validate:"max=100"`
	Tags              []string               `json:"tags,omitempty"`
	PublishedAt       *time.Time             `json:"published_at,omitempty"`
	SourceURL         string                 `json:"source_url,omitempty" validate:"omitempty,url,max=500"`
	FeaturedImageURL  string                 `json:"featured_image_url,omitempty" validate:"omitempty,url,max=500"`
	// SEO fields
	MetaTitle         string                 `json:"meta_title,omitempty" validate:"max=255"`
	MetaDescription   string                 `json:"meta_description,omitempty" validate:"max=500"`
	CanonicalURL      string                 `json:"canonical_url,omitempty" validate:"omitempty,url,max=500"`
	FocusKeyword      string                 `json:"focus_keyword,omitempty" validate:"max=100"`
	// Auto-linking
	EnableAutoLinking bool                   `json:"enable_auto_linking,omitempty"`
	// Language and Translation
	LanguageCode        string               `json:"language_code,omitempty" validate:"omitempty,max=5"`
	TranslationGroupID  *uint64              `json:"translation_group_id,omitempty"`  // Link to existing translation group
	TranslateOfArticleID *uint64             `json:"translate_of_article_id,omitempty"` // ID of the original article this is a translation of
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// DuplicateCheckResult represents the result of duplicate content detection
type DuplicateCheckResult struct {
	IsDuplicate   bool    `json:"is_duplicate"`
	ExistingID    *uint64 `json:"existing_id,omitempty"`
	Similarity    float64 `json:"similarity"`
	MatchType     string  `json:"match_type"` // exact, hash, title, content
	MatchedField  string  `json:"matched_field,omitempty"`
}

// ValidationResult represents content validation results
type ValidationResult struct {
	IsValid bool     `json:"is_valid"`
	Errors  []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// GenerateContentHash creates a hash of the content for duplicate detection
func (ic *IngestedContent) GenerateContentHash() {
	// Normalize content for hashing
	normalizedContent := strings.ToLower(strings.TrimSpace(ic.Content))
	normalizedTitle := strings.ToLower(strings.TrimSpace(ic.Title))
	
	// Create hash from title + content
	hashInput := normalizedTitle + "|" + normalizedContent
	hash := sha256.Sum256([]byte(hashInput))
	ic.ContentHash = fmt.Sprintf("%x", hash)
}

// ValidateIngestedContent validates ingested content
func ValidateIngestedContent(content *IngestedContent) *ValidationResult {
	result := &ValidationResult{IsValid: true}
	
	// Required field validation
	if strings.TrimSpace(content.Title) == "" {
		result.Errors = append(result.Errors, "title is required")
		result.IsValid = false
	}
	
	if strings.TrimSpace(content.Content) == "" {
		result.Errors = append(result.Errors, "content is required")
		result.IsValid = false
	}
	
	if content.SourceID == 0 {
		result.Errors = append(result.Errors, "source_id is required")
		result.IsValid = false
	}
	
	if strings.TrimSpace(content.ExternalID) == "" {
		result.Errors = append(result.Errors, "external_id is required")
		result.IsValid = false
	}
	
	// Length validation
	if len(content.Title) > 255 {
		result.Errors = append(result.Errors, "title must be less than 255 characters")
		result.IsValid = false
	}
	
	if len(content.Excerpt) > 500 {
		result.Errors = append(result.Errors, "excerpt must be less than 500 characters")
		result.IsValid = false
	}
	
	if len(content.AuthorName) > 100 {
		result.Errors = append(result.Errors, "author_name must be less than 100 characters")
		result.IsValid = false
	}
	
	if len(content.AuthorEmail) > 255 {
		result.Errors = append(result.Errors, "author_email must be less than 255 characters")
		result.IsValid = false
	}
	
	if len(content.SourceURL) > 500 {
		result.Errors = append(result.Errors, "source_url must be less than 500 characters")
		result.IsValid = false
	}
	
	// Email validation
	if content.AuthorEmail != "" && !IsValidEmail(content.AuthorEmail) {
		result.Errors = append(result.Errors, "author_email must be a valid email address")
		result.IsValid = false
	}
	
	// URL validation
	if content.SourceURL != "" && !IsValidURL(content.SourceURL) {
		result.Errors = append(result.Errors, "source_url must be a valid URL")
		result.IsValid = false
	}
	
	// Warnings for missing optional but recommended fields
	if content.AuthorName == "" {
		result.Warnings = append(result.Warnings, "author_name is recommended for better attribution")
	}
	
	if content.Excerpt == "" {
		result.Warnings = append(result.Warnings, "excerpt is recommended for better SEO")
	}
	
	if content.PublishedAt == nil {
		result.Warnings = append(result.Warnings, "published_at is recommended for proper chronological ordering")
	}
	
	return result
}

// SanitizeIngestedContent sanitizes content from external sources
func SanitizeIngestedContent(content *IngestedContent) {
	// Sanitize title
	content.Title = SanitizeTitle(content.Title)
	
	// Sanitize content (basic HTML sanitization)
	content.Content = SanitizeHTML(content.Content)
	
	// Sanitize excerpt
	if content.Excerpt != "" {
		content.Excerpt = SanitizeTitle(content.Excerpt) // Use same sanitization as title
	}
	
	// Sanitize author name
	if content.AuthorName != "" {
		content.AuthorName = SanitizeTitle(content.AuthorName)
	}
	
	// Sanitize author email
	if content.AuthorEmail != "" {
		content.AuthorEmail = strings.ToLower(strings.TrimSpace(content.AuthorEmail))
	}
	
	// Sanitize category name
	if content.CategoryName != "" {
		content.CategoryName = SanitizeTitle(content.CategoryName)
	}
	
	// Sanitize tags
	for i, tag := range content.Tags {
		content.Tags[i] = strings.TrimSpace(tag)
	}
	
	// Generate content hash after sanitization
	content.GenerateContentHash()
}

// PrepareForProcessing prepares ingested content for processing
func (ic *IngestedContent) PrepareForProcessing() {
	if ic.Status == "" {
		ic.Status = "pending"
	}
	
	// Generate content hash if not already set
	if ic.ContentHash == "" {
		ic.GenerateContentHash()
	}
	
	// Set timestamps
	now := time.Now()
	if ic.CreatedAt.IsZero() {
		ic.CreatedAt = now
	}
	ic.UpdatedAt = now
}



// SanitizeHTML performs basic HTML sanitization (placeholder - use a proper library in production)
func SanitizeHTML(content string) string {
	// This is a basic implementation - in production, use a library like bluemonday
	// Remove script tags and other dangerous elements
	content = strings.ReplaceAll(content, "<script", "&lt;script")
	content = strings.ReplaceAll(content, "</script>", "&lt;/script&gt;")
	content = strings.ReplaceAll(content, "<iframe", "&lt;iframe")
	content = strings.ReplaceAll(content, "</iframe>", "&lt;/iframe&gt;")
	
	return content
}