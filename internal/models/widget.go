package models

import (
	"encoding/json"
	"time"
)

// Widget represents a reusable UI component that can be placed on pages
type Widget struct {
	ID          uint64                 `json:"id" db:"id"`
	Name        string                 `json:"name" db:"name"`
	Type        WidgetType             `json:"type" db:"type"`
	Title       string                 `json:"title" db:"title"`
	Description string                 `json:"description" db:"description"`
	Config      map[string]interface{} `json:"config" db:"config"`
	IsActive    bool                   `json:"is_active" db:"is_active"`
	SortOrder   int                    `json:"sort_order" db:"sort_order"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
}

// WidgetType defines the type of widget
type WidgetType string

const (
	WidgetTypeLatestArticles   WidgetType = "latest_articles"
	WidgetTypePopularArticles  WidgetType = "popular_articles"
	WidgetTypeTrendingArticles WidgetType = "trending_articles"
	WidgetTypeCategories       WidgetType = "categories"
	WidgetTypeTags             WidgetType = "tags"
	WidgetTypeSearch           WidgetType = "search"
	WidgetTypeNewsletter       WidgetType = "newsletter"
	WidgetTypeCustomHTML       WidgetType = "custom_html"
	WidgetTypeAdvertisement    WidgetType = "advertisement"
	WidgetTypeSocialMedia      WidgetType = "social_media"
)

// WidgetPlacement represents where a widget is placed on a page
type WidgetPlacement struct {
	ID         uint64     `json:"id" db:"id"`
	WidgetID   uint64     `json:"widget_id" db:"widget_id"`
	PageType   PageType   `json:"page_type" db:"page_type"`
	Zone       WidgetZone `json:"zone" db:"zone"`
	Position   int        `json:"position" db:"position"`
	IsActive   bool       `json:"is_active" db:"is_active"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
	Widget     *Widget    `json:"widget,omitempty"`
}

// PageType defines where widgets can be placed
type PageType string

const (
	PageTypeHomepage PageType = "homepage"
	PageTypeArticle  PageType = "article"
	PageTypeCategory PageType = "category"
	PageTypeTag      PageType = "tag"
	PageTypeSearch   PageType = "search"
	PageTypeAuthor   PageType = "author"
	PageTypeGlobal   PageType = "global" // Appears on all pages
)

// WidgetZone defines the zones where widgets can be placed
type WidgetZone string

const (
	WidgetZoneHeader     WidgetZone = "header"
	WidgetZoneSidebar    WidgetZone = "sidebar"
	WidgetZoneFooter     WidgetZone = "footer"
	WidgetZoneContent    WidgetZone = "content"
	WidgetZoneAfterTitle WidgetZone = "after_title"
	WidgetZoneBeforeContent WidgetZone = "before_content"
	WidgetZoneAfterContent  WidgetZone = "after_content"
)

// WidgetConfig holds configuration for different widget types
type WidgetConfig struct {
	// Common config
	CacheEnabled bool          `json:"cache_enabled"`
	CacheTTL     time.Duration `json:"cache_ttl"`
	
	// Latest/Popular Articles config
	ArticleCount int      `json:"article_count,omitempty"`
	CategoryIDs  []uint64 `json:"category_ids,omitempty"`
	TagIDs       []uint64 `json:"tag_ids,omitempty"`
	ShowExcerpt  bool     `json:"show_excerpt,omitempty"`
	ShowDate     bool     `json:"show_date,omitempty"`
	ShowAuthor   bool     `json:"show_author,omitempty"`
	ShowImage    bool     `json:"show_image,omitempty"`
	
	// Categories config
	ShowHierarchy bool `json:"show_hierarchy,omitempty"`
	MaxDepth      int  `json:"max_depth,omitempty"`
	ShowCount     bool `json:"show_count,omitempty"`
	
	// Custom HTML config
	HTMLContent string `json:"html_content,omitempty"`
	
	// Advertisement config
	AdSlotID string `json:"ad_slot_id,omitempty"`
	AdSize   string `json:"ad_size,omitempty"`
	
	// Social Media config
	Platforms []string `json:"platforms,omitempty"`
	ShowIcons bool     `json:"show_icons,omitempty"`
}

// GetConfig returns the widget configuration as a typed struct
func (w *Widget) GetConfig() (*WidgetConfig, error) {
	configBytes, err := json.Marshal(w.Config)
	if err != nil {
		return nil, err
	}
	
	var config WidgetConfig
	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		return nil, err
	}
	
	return &config, nil
}

// SetConfig sets the widget configuration from a typed struct
func (w *Widget) SetConfig(config *WidgetConfig) error {
	configMap := make(map[string]interface{})
	configBytes, err := json.Marshal(config)
	if err != nil {
		return err
	}
	
	err = json.Unmarshal(configBytes, &configMap)
	if err != nil {
		return err
	}
	
	w.Config = configMap
	return nil
}

// Validate validates the widget data
func (w *Widget) Validate() error {
	if w.Name == "" {
		return NewValidationError("name", "Widget name is required")
	}
	
	if w.Type == "" {
		return NewValidationError("type", "Widget type is required")
	}
	
	// Validate widget type
	validTypes := []WidgetType{
		WidgetTypeLatestArticles,
		WidgetTypePopularArticles,
		WidgetTypeTrendingArticles,
		WidgetTypeCategories,
		WidgetTypeTags,
		WidgetTypeSearch,
		WidgetTypeNewsletter,
		WidgetTypeCustomHTML,
		WidgetTypeAdvertisement,
		WidgetTypeSocialMedia,
	}
	
	isValidType := false
	for _, validType := range validTypes {
		if w.Type == validType {
			isValidType = true
			break
		}
	}
	
	if !isValidType {
		return NewValidationError("type", "Invalid widget type")
	}
	
	return nil
}

// Validate validates the widget placement data
func (wp *WidgetPlacement) Validate() error {
	if wp.WidgetID == 0 {
		return NewValidationError("widget_id", "Widget ID is required")
	}
	
	if wp.PageType == "" {
		return NewValidationError("page_type", "Page type is required")
	}
	
	if wp.Zone == "" {
		return NewValidationError("zone", "Widget zone is required")
	}
	
	// Validate page type
	validPageTypes := []PageType{
		PageTypeHomepage,
		PageTypeArticle,
		PageTypeCategory,
		PageTypeTag,
		PageTypeSearch,
		PageTypeAuthor,
		PageTypeGlobal,
	}
	
	isValidPageType := false
	for _, validPageType := range validPageTypes {
		if wp.PageType == validPageType {
			isValidPageType = true
			break
		}
	}
	
	if !isValidPageType {
		return NewValidationError("page_type", "Invalid page type")
	}
	
	// Validate zone
	validZones := []WidgetZone{
		WidgetZoneHeader,
		WidgetZoneSidebar,
		WidgetZoneFooter,
		WidgetZoneContent,
		WidgetZoneAfterTitle,
		WidgetZoneBeforeContent,
		WidgetZoneAfterContent,
	}
	
	isValidZone := false
	for _, validZone := range validZones {
		if wp.Zone == validZone {
			isValidZone = true
			break
		}
	}
	
	if !isValidZone {
		return NewValidationError("zone", "Invalid widget zone")
	}
	
	return nil
}