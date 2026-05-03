package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Tag represents a content tag with keyword bank for auto-linking
type Tag struct {
	ID                 uint64    `json:"id" db:"id"`
	Name               string    `json:"name" db:"name" validate:"required,max=100"`
	Slug               string    `json:"slug" db:"slug" validate:"required,max=100,slug"`
	Description        string    `json:"description" db:"description" validate:"max=500"`
	Keywords           []string  `json:"keywords" db:"keywords"`
	Color              string    `json:"color" db:"color" validate:"omitempty,hexcolor"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
	LanguageCode       string    `json:"language_code" db:"language_code" validate:"required,len=2"`
	TranslationGroupID *uint64   `json:"translation_group_id" db:"translation_group_id"`
	
	// Computed fields (not stored in DB)
	ArticleCount int `json:"article_count,omitempty"`
}

// Scan implements the sql.Scanner interface for Keywords slice
func (t *Tag) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, &t.Keywords)
	case string:
		return json.Unmarshal([]byte(v), &t.Keywords)
	default:
		return fmt.Errorf("cannot scan %T into Keywords", value)
	}
}

// Value implements the driver.Valuer interface for Keywords slice
func (t Tag) Value() (driver.Value, error) {
	return json.Marshal(t.Keywords)
}

// ValidateTag validates a Tag struct with comprehensive error checking
func ValidateTag(tag *Tag) error {
	var errors []string

	// Name validation
	if strings.TrimSpace(tag.Name) == "" {
		errors = append(errors, "name is required")
	}
	if len(tag.Name) > 100 {
		errors = append(errors, "name must be less than 100 characters")
	}

	// Description validation
	if len(tag.Description) > 500 {
		errors = append(errors, "description must be less than 500 characters")
	}

	// Slug validation and generation
	if tag.Slug == "" {
		tag.Slug = GenerateSlug(tag.Name)
	}
	if !IsValidSlug(tag.Slug) {
		errors = append(errors, "slug contains invalid characters")
	}

	// Color validation
	if tag.Color != "" && !IsValidHexColor(tag.Color) {
		errors = append(errors, "color must be a valid hex color (e.g., #FF0000)")
	}

	// Keywords validation
	if err := ValidateKeywords(tag.Keywords); err != nil {
		errors = append(errors, fmt.Sprintf("keywords: %s", err.Error()))
	}

	// Language code validation
	if strings.TrimSpace(tag.LanguageCode) == "" {
		tag.LanguageCode = "en" // Default to English
	}
	if len(tag.LanguageCode) != 2 {
		errors = append(errors, "language_code must be exactly 2 characters")
	}

	if len(errors) > 0 {
		return &ValidationError{
			Message: "Tag validation failed",
			Fields:  errors,
		}
	}

	return nil
}

// ValidateKeywords validates the keywords array
func ValidateKeywords(keywords []string) error {
	var errors []string

	// Check for duplicate keywords
	seen := make(map[string]bool)
	for _, keyword := range keywords {
		normalized := strings.ToLower(strings.TrimSpace(keyword))
		if normalized == "" {
			continue // Skip empty keywords
		}
		
		if seen[normalized] {
			errors = append(errors, fmt.Sprintf("duplicate keyword: %s", keyword))
		}
		seen[normalized] = true

		// Validate individual keyword
		if len(keyword) > 100 {
			errors = append(errors, fmt.Sprintf("keyword '%s' must be less than 100 characters", keyword))
		}
		
		if !IsValidKeyword(keyword) {
			errors = append(errors, fmt.Sprintf("keyword '%s' contains invalid characters", keyword))
		}
	}

	if len(errors) > 0 {
		return &ValidationError{
			Message: "Keywords validation failed",
			Fields:  errors,
		}
	}

	return nil
}

// IsValidHexColor validates hex color format
func IsValidHexColor(color string) bool {
	if len(color) != 7 || color[0] != '#' {
		return false
	}
	
	for i := 1; i < 7; i++ {
		c := color[i]
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	
	return true
}

// IsValidKeyword validates individual keyword format
func IsValidKeyword(keyword string) bool {
	// Keywords should not contain special characters that could break auto-linking
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return false
	}
	
	// Allow letters, numbers, spaces, hyphens, and apostrophes
	for _, r := range keyword {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || 
			 (r >= '0' && r <= '9') || r == ' ' || r == '-' || r == '\'') {
			return false
		}
	}
	
	return true
}

// PrepareForDB prepares the tag for database insertion
func (t *Tag) PrepareForDB() {
	t.Name = strings.TrimSpace(t.Name)
	t.Description = strings.TrimSpace(t.Description)
	
	if t.Slug == "" {
		t.Slug = GenerateSlug(t.Name)
	}
	
	if t.Color == "" {
		t.Color = "#000000" // Default color
	}
	
	if t.LanguageCode == "" {
		t.LanguageCode = "en" // Default to English
	}
	
	// Clean and normalize keywords
	t.Keywords = NormalizeKeywords(t.Keywords)
}

// NormalizeKeywords cleans and normalizes the keywords array
func NormalizeKeywords(keywords []string) []string {
	var normalized []string
	seen := make(map[string]bool)
	
	for _, keyword := range keywords {
		cleaned := strings.TrimSpace(keyword)
		if cleaned == "" {
			continue
		}
		
		// Convert to lowercase for deduplication
		lower := strings.ToLower(cleaned)
		if !seen[lower] {
			normalized = append(normalized, cleaned)
			seen[lower] = true
		}
	}
	
	return normalized
}

// AddKeyword adds a new keyword to the tag if it doesn't already exist
func (t *Tag) AddKeyword(keyword string) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return
	}
	
	// Check if keyword already exists (case-insensitive)
	lower := strings.ToLower(keyword)
	for _, existing := range t.Keywords {
		if strings.ToLower(existing) == lower {
			return // Already exists
		}
	}
	
	t.Keywords = append(t.Keywords, keyword)
}

// RemoveKeyword removes a keyword from the tag
func (t *Tag) RemoveKeyword(keyword string) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return
	}
	
	lower := strings.ToLower(keyword)
	var filtered []string
	
	for _, existing := range t.Keywords {
		if strings.ToLower(existing) != lower {
			filtered = append(filtered, existing)
		}
	}
	
	t.Keywords = filtered
}

// HasKeyword checks if the tag contains a specific keyword
func (t *Tag) HasKeyword(keyword string) bool {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return false
	}
	
	lower := strings.ToLower(keyword)
	for _, existing := range t.Keywords {
		if strings.ToLower(existing) == lower {
			return true
		}
	}
	
	return false
}

// GetLongestKeyword returns the longest keyword for priority matching
func (t *Tag) GetLongestKeyword() string {
	longest := ""
	for _, keyword := range t.Keywords {
		if len(keyword) > len(longest) {
			longest = keyword
		}
	}
	return longest
}