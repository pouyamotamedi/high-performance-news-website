package models

import (
	"regexp"
	"strings"
	"time"
	"unicode"
)

// Article represents a news article with comprehensive metadata
type Article struct {
	ID                 uint64     `json:"id" db:"id"`
	Title              string     `json:"title" db:"title" validate:"required,max=255"`
	Slug               string     `json:"slug" db:"slug" validate:"required,max=255,slug"`
	Content            string     `json:"content" db:"content" validate:"required"`
	Excerpt            string     `json:"excerpt" db:"excerpt" validate:"max=500"`
	AuthorID           uint64     `json:"author_id" db:"author_id" validate:"required"`
	CategoryID         uint64     `json:"category_id" db:"category_id" validate:"required"`
	Categories         []Category `json:"categories"`
	Tags               []Tag      `json:"tags"`
	Status             string     `json:"status" db:"status" validate:"required,oneof=draft published archived scheduled deleted"`
	PublishedAt        *time.Time `json:"published_at" db:"published_at"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`
	ViewCount          uint64     `json:"view_count" db:"view_count"`
	LikeCount          uint64     `json:"like_count" db:"like_count"`
	DislikeCount         uint64     `json:"dislike_count" db:"dislike_count"`
	// Individual SEO fields (matching database columns)
	MetaTitle            string     `json:"meta_title" db:"meta_title" validate:"max=60"`
	MetaDescription      string     `json:"meta_description" db:"meta_description" validate:"max=160"`
	FocusKeyword         string     `json:"focus_keyword" db:"focus_keyword" validate:"max=100"`
	CanonicalURL         string     `json:"canonical_url" db:"canonical_url" validate:"omitempty,url,max=500"`
	SchemaType           string     `json:"schema_type" db:"schema_type" validate:"oneof=NewsArticle Article BlogPosting"`
	AutoLinking          bool       `json:"auto_linking" db:"auto_linking"` // Override for per-article auto-linking
	LanguageCode         string     `json:"language_code" db:"language_code" validate:"required,len=5"`
	TranslationGroupID   *uint64    `json:"translation_group_id" db:"translation_group_id"`
	ModerationStatus     string     `json:"moderation_status" db:"moderation_status"`
	ModerationNotes      string     `json:"moderation_notes" db:"moderation_notes"`
	LastModeratedAt      *time.Time `json:"last_moderated_at" db:"last_moderated_at"`
	LastModeratedBy      *uint64    `json:"last_moderated_by" db:"last_moderated_by"`
	
	// Featured image fields
	FeaturedImageID      *uint64    `json:"featured_image_id" db:"featured_image_id"`
	FeaturedImage        string     `json:"featured_image,omitempty"` // URL populated from join
	
	// Temporary compatibility field for repository - TODO: remove when repository is fixed
	SEOData              SEOData `json:"-"` // Don't serialize this field
}

// SEOData - temporary compatibility struct for services layer
type SEOData struct {
	MetaTitle       string   `json:"meta_title"`
	MetaDescription string   `json:"meta_description"`
	FocusKeyword    string   `json:"focus_keyword"`
	Keywords        []string `json:"keywords"`
	CanonicalURL    string   `json:"canonical_url"`
	SchemaType      string   `json:"schema_type"`
}

// ArticleFilter represents filters for article queries
type ArticleFilter struct {
	Status          string     `json:"status,omitempty"`
	CategoryID      *uint64    `json:"category_id,omitempty"`
	TagID           *uint64    `json:"tag_id,omitempty"`
	AuthorID        *uint64    `json:"author_id,omitempty"`
	PublishedAfter  *time.Time `json:"published_after,omitempty"`
	PublishedBefore *time.Time `json:"published_before,omitempty"`
	LanguageCode    string     `json:"language_code,omitempty"`
}

// ValidateArticle validates an Article struct with comprehensive error checking
func ValidateArticle(article *Article) error {
	var errors []string

	// Title validation
	if strings.TrimSpace(article.Title) == "" {
		errors = append(errors, "title is required")
	}
	if len(article.Title) > 255 {
		errors = append(errors, "title must be less than 255 characters")
	}

	// Content validation
	if strings.TrimSpace(article.Content) == "" {
		errors = append(errors, "content is required")
	}

	// Excerpt validation
	if len(article.Excerpt) > 500 {
		errors = append(errors, "excerpt must be less than 500 characters")
	}

	// Author ID validation
	if article.AuthorID == 0 {
		errors = append(errors, "author_id is required")
	}

	// Category ID validation
	if article.CategoryID == 0 {
		errors = append(errors, "category_id is required")
	}

	// Language code validation
	if strings.TrimSpace(article.LanguageCode) == "" {
		article.LanguageCode = "fa" // Default to Persian
	}
	if len(article.LanguageCode) != 2 {
		errors = append(errors, "language_code must be exactly 2 characters")
	}

	// Status validation
	validStatuses := map[string]bool{
		"draft":     true,
		"published": true,
		"archived":  true,
		"scheduled": true,
		"deleted":   true,
	}
	if !validStatuses[article.Status] {
		errors = append(errors, "status must be one of: draft, published, archived")
	}

	// Slug validation and generation
	if article.Slug == "" {
		article.Slug = GenerateSlug(article.Title)
	}
	if !IsValidSlug(article.Slug) {
		errors = append(errors, "slug contains invalid characters")
	}

	// SEO data validation - now using individual fields
	if len(article.MetaTitle) > 60 {
		errors = append(errors, "meta_title: must be 60 characters or less")
	}
	if len(article.MetaDescription) > 160 {
		errors = append(errors, "meta_description: must be 160 characters or less")
	}

	if len(errors) > 0 {
		return &ValidationError{
			Message: "Article validation failed",
			Fields:  errors,
		}
	}

	return nil
}

// ValidateSEOData validates SEO data with comprehensive error checking
func ValidateSEOData(seo *SEOData) error {
	var errors []string

	// Meta title validation
	if len(seo.MetaTitle) > 60 {
		errors = append(errors, "meta_title must be less than 60 characters")
	}

	// Meta description validation
	if len(seo.MetaDescription) > 160 {
		errors = append(errors, "meta_description must be less than 160 characters")
	}

	// Focus keyword validation
	if len(seo.FocusKeyword) > 100 {
		errors = append(errors, "focus_keyword must be less than 100 characters")
	}

	// Canonical URL validation
	if seo.CanonicalURL != "" {
		if !IsValidURL(seo.CanonicalURL) {
			errors = append(errors, "canonical_url must be a valid URL")
		}
		if len(seo.CanonicalURL) > 500 {
			errors = append(errors, "canonical_url must be less than 500 characters")
		}
	}

	// Schema type validation
	if seo.SchemaType != "" {
		validSchemaTypes := []string{"NewsArticle", "Article", "BlogPosting"}
		if !containsString(validSchemaTypes, seo.SchemaType) {
			errors = append(errors, "schema_type must be one of: NewsArticle, Article, BlogPosting")
		}
	}

	// Keywords validation
	if len(seo.Keywords) > 10 {
		errors = append(errors, "keywords array cannot have more than 10 items")
	}
	for _, keyword := range seo.Keywords {
		if len(keyword) > 50 {
			errors = append(errors, "each keyword must be less than 50 characters")
		}
	}

	if len(errors) > 0 {
		return &ValidationError{
			Message: "SEO data validation failed",
			Fields:  errors,
		}
	}
	return nil
}

// GenerateSlug creates a URL-friendly slug from a title
func GenerateSlug(title string) string {
	// Convert to lowercase
	slug := strings.ToLower(title)
	
	// Replace spaces and special characters with hyphens
	slug = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(slug, "-")
	
	// Remove leading and trailing hyphens
	slug = strings.Trim(slug, "-")
	
	// Replace multiple consecutive hyphens with single hyphen
	slug = regexp.MustCompile(`-+`).ReplaceAllString(slug, "-")
	
	return slug
}

// IsValidSlug checks if a slug contains only valid characters
func IsValidSlug(s string) bool {
	// Slug should contain only lowercase letters, numbers, and hyphens
	// Should not start or end with hyphen
	if s == "" {
		return false
	}
	
	slugRegex := regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	return slugRegex.MatchString(s)
}

// IsValidURL validates URL format
func IsValidURL(url string) bool {
	urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	return urlRegex.MatchString(url)
}

// SanitizeTitle removes invalid characters from title
func SanitizeTitle(title string) string {
	// Remove control characters and normalize spaces
	var result strings.Builder
	for _, r := range title {
		if unicode.IsControl(r) {
			continue
		}
		if unicode.IsSpace(r) {
			result.WriteRune(' ')
		} else {
			result.WriteRune(r)
		}
	}
	
	// Normalize multiple spaces to single space
	normalized := regexp.MustCompile(`\s+`).ReplaceAllString(result.String(), " ")
	return strings.TrimSpace(normalized)
}

// PrepareForDB prepares the article for database insertion
func (a *Article) PrepareForDB() {
	a.Title = SanitizeTitle(a.Title)
	if a.Slug == "" {
		a.Slug = GenerateSlug(a.Title)
	}
	if a.Status == "" {
		a.Status = "draft"
	}
	if a.SchemaType == "" {
		a.SchemaType = "NewsArticle"
	}
	if a.LanguageCode == "" {
		a.LanguageCode = "fa" // Default to Persian
	}
	// Auto-linking is enabled by default (true)
	// This field allows per-article override to disable auto-linking
}