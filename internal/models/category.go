package models

import (
	"fmt"
	"strings"
	"time"
)

// Category represents a content category with hierarchical support
type Category struct {
	ID                 uint64     `json:"id" db:"id"`
	Name               string     `json:"name" db:"name" validate:"required,max=100"`
	Slug               string     `json:"slug" db:"slug" validate:"required,max=100,slug"`
	Description        string     `json:"description" db:"description" validate:"max=500"`
	ParentID           *uint64    `json:"parent_id" db:"parent_id"`
	SortOrder          int        `json:"sort_order" db:"sort_order"`
	ImageURL           *string    `json:"image_url" db:"image_url" validate:"omitempty,max=500"`
	ImageAltText       *string    `json:"image_alt_text" db:"image_alt_text" validate:"omitempty,max=200"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`
	LanguageCode       string     `json:"language_code" db:"language_code" validate:"required,len=2"`
	TranslationGroupID *uint64    `json:"translation_group_id" db:"translation_group_id"`
	
	// Computed fields (not stored in DB)
	Parent      *Category   `json:"parent,omitempty"`
	Children    []Category  `json:"children,omitempty"`
	ArticleCount int        `json:"article_count,omitempty"`
}

// ValidateCategory validates a Category struct with comprehensive error checking
func ValidateCategory(category *Category) error {
	var errors []string

	// Name validation
	if strings.TrimSpace(category.Name) == "" {
		errors = append(errors, "name is required")
	}
	if len(category.Name) > 100 {
		errors = append(errors, "name must be less than 100 characters")
	}

	// Description validation
	if len(category.Description) > 500 {
		errors = append(errors, "description must be less than 500 characters")
	}

	// Image URL validation
	if category.ImageURL != nil && len(*category.ImageURL) > 500 {
		errors = append(errors, "image URL must be less than 500 characters")
	}

	// Image alt text validation
	if category.ImageAltText != nil && len(*category.ImageAltText) > 200 {
		errors = append(errors, "image alt text must be less than 200 characters")
	}

	// Slug validation and generation
	if category.Slug == "" {
		category.Slug = GenerateSlug(category.Name)
	}
	if !IsValidSlug(category.Slug) {
		errors = append(errors, "slug contains invalid characters")
	}

	// Parent ID validation (cannot be self-referencing)
	if category.ParentID != nil && *category.ParentID == category.ID {
		errors = append(errors, "category cannot be its own parent")
	}

	// Language code validation
	if strings.TrimSpace(category.LanguageCode) == "" {
		category.LanguageCode = "en" // Default to English
	}
	if len(category.LanguageCode) != 2 {
		errors = append(errors, "language_code must be exactly 2 characters")
	}

	if len(errors) > 0 {
		return &ValidationError{
			Message: "Category validation failed",
			Fields:  errors,
		}
	}

	return nil
}

// PrepareForDB prepares the category for database insertion
func (c *Category) PrepareForDB() {
	c.Name = strings.TrimSpace(c.Name)
	c.Description = strings.TrimSpace(c.Description)
	
	if c.Slug == "" {
		c.Slug = GenerateSlug(c.Name)
	}
	if c.LanguageCode == "" {
		c.LanguageCode = "en" // Default to English
	}
}

// IsRoot returns true if this is a root category (no parent)
func (c *Category) IsRoot() bool {
	return c.ParentID == nil
}

// GetPath returns the full path of the category (e.g., "Technology/Programming/Go")
func (c *Category) GetPath() string {
	if c.Parent == nil {
		return c.Name
	}
	return c.Parent.GetPath() + "/" + c.Name
}

// GetSlugPath returns the full slug path (e.g., "technology/programming/go")
func (c *Category) GetSlugPath() string {
	if c.Parent == nil {
		return c.Slug
	}
	return c.Parent.GetSlugPath() + "/" + c.Slug
}

// HasChildren returns true if the category has child categories
func (c *Category) HasChildren() bool {
	return len(c.Children) > 0
}

// GetDepth returns the depth level of the category (0 for root)
func (c *Category) GetDepth() int {
	if c.Parent == nil {
		return 0
	}
	return c.Parent.GetDepth() + 1
}

// HasImage returns true if the category has an image
func (c *Category) HasImage() bool {
	return c.ImageURL != nil && *c.ImageURL != ""
}

// GetImageURL returns the image URL or empty string if no image
func (c *Category) GetImageURL() string {
	if c.ImageURL == nil {
		return ""
	}
	return *c.ImageURL
}

// GetImageAltText returns the image alt text or category name as fallback
func (c *Category) GetImageAltText() string {
	if c.ImageAltText != nil && *c.ImageAltText != "" {
		return *c.ImageAltText
	}
	return c.Name + " category"
}

// SetImage sets the image URL and alt text for the category
func (c *Category) SetImage(url, altText string) {
	if url == "" {
		c.ImageURL = nil
		c.ImageAltText = nil
	} else {
		c.ImageURL = &url
		if altText == "" {
			altText = c.Name + " category"
		}
		c.ImageAltText = &altText
	}
}

// ValidateHierarchy validates that the category hierarchy is valid
func ValidateHierarchy(categories []Category) error {
	// Create a map for quick lookup
	categoryMap := make(map[uint64]*Category)
	for i := range categories {
		categoryMap[categories[i].ID] = &categories[i]
	}

	// Check for circular references and invalid parents
	for _, category := range categories {
		if category.ParentID == nil {
			continue
		}

		// Check if parent exists
		parent, exists := categoryMap[*category.ParentID]
		if !exists {
			return &ValidationError{
				Message: "Category hierarchy validation failed",
				Fields:  []string{fmt.Sprintf("parent category with ID %d does not exist", *category.ParentID)},
			}
		}

		// Check for circular reference by traversing up the hierarchy
		current := parent
		visited := make(map[uint64]bool)
		for current != nil {
			if visited[current.ID] {
				return &ValidationError{
					Message: "Category hierarchy validation failed",
					Fields:  []string{"circular reference detected in category hierarchy"},
				}
			}
			visited[current.ID] = true

			if current.ParentID == nil {
				break
			}

			current = categoryMap[*current.ParentID]
		}
	}

	return nil
}