package models

import (
	"time"
)

// AdvertisementCampaign represents an advertising campaign
type AdvertisementCampaign struct {
	ID             uint64     `json:"id" db:"id"`
	Name           string     `json:"name" db:"name" validate:"required,max=255"`
	Description    string     `json:"description" db:"description"`
	AdvertiserName string     `json:"advertiser_name" db:"advertiser_name" validate:"required,max=255"`
	AdvertiserEmail string    `json:"advertiser_email" db:"advertiser_email" validate:"email,max=255"`
	StartDate      time.Time  `json:"start_date" db:"start_date" validate:"required"`
	EndDate        *time.Time `json:"end_date" db:"end_date"`
	BudgetTotal    *float64   `json:"budget_total" db:"budget_total" validate:"omitempty,min=0"`
	BudgetDaily    *float64   `json:"budget_daily" db:"budget_daily" validate:"omitempty,min=0"`
	Status         string     `json:"status" db:"status" validate:"required,oneof=active paused completed draft"`
	Priority       int        `json:"priority" db:"priority" validate:"min=1,max=10"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
	
	// Related data (not stored in DB)
	Creatives   []AdvertisementCreative   `json:"creatives,omitempty"`
	Targeting   []AdvertisementTargeting  `json:"targeting,omitempty"`
	Placements  []AdvertisementPlacement  `json:"placements,omitempty"`
	Stats       *AdvertisementStats       `json:"stats,omitempty"`
}

// AdvertisementSlot represents an advertisement placement slot
type AdvertisementSlot struct {
	ID          uint64    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name" validate:"required,max=255"`
	Slug        string    `json:"slug" db:"slug" validate:"required,max=255"`
	Description string    `json:"description" db:"description"`
	PageType    string    `json:"page_type" db:"page_type" validate:"required,oneof=homepage article category tag search all"`
	Position    string    `json:"position" db:"position" validate:"required,oneof=header sidebar content-top content-middle content-bottom footer floating"`
	Width       *int      `json:"width" db:"width" validate:"omitempty,min=1"`
	Height      *int      `json:"height" db:"height" validate:"omitempty,min=1"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	LazyLoad    bool      `json:"lazy_load" db:"lazy_load"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// AdvertisementCreative represents an advertisement creative (image, HTML, script)
type AdvertisementCreative struct {
	ID         uint64    `json:"id" db:"id"`
	CampaignID uint64    `json:"campaign_id" db:"campaign_id" validate:"required"`
	Name       string    `json:"name" db:"name" validate:"required,max=255"`
	Type       string    `json:"type" db:"type" validate:"required,oneof=image html script video"`
	Content    string    `json:"content" db:"content" validate:"required"`
	AltText    string    `json:"alt_text" db:"alt_text" validate:"max=255"`
	ClickURL   string    `json:"click_url" db:"click_url" validate:"omitempty,url,max=1000"`
	Width      *int      `json:"width" db:"width" validate:"omitempty,min=1"`
	Height     *int      `json:"height" db:"height" validate:"omitempty,min=1"`
	FileSize   *int      `json:"file_size" db:"file_size" validate:"omitempty,min=0"` // in bytes
	IsActive   bool      `json:"is_active" db:"is_active"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// AdvertisementTargeting represents targeting rules for campaigns
type AdvertisementTargeting struct {
	ID          uint64    `json:"id" db:"id"`
	CampaignID  uint64    `json:"campaign_id" db:"campaign_id" validate:"required"`
	TargetType  string    `json:"target_type" db:"target_type" validate:"required,oneof=category tag page_type device time"`
	TargetValue string    `json:"target_value" db:"target_value" validate:"required,max=255"`
	IsInclude   bool      `json:"is_include" db:"is_include"` // true for include, false for exclude
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// AdvertisementPlacement represents the assignment of campaigns to slots
type AdvertisementPlacement struct {
	ID         uint64    `json:"id" db:"id"`
	CampaignID uint64    `json:"campaign_id" db:"campaign_id" validate:"required"`
	SlotID     uint64    `json:"slot_id" db:"slot_id" validate:"required"`
	CreativeID uint64    `json:"creative_id" db:"creative_id" validate:"required"`
	Weight     int       `json:"weight" db:"weight" validate:"min=1"` // For A/B testing and rotation
	IsActive   bool      `json:"is_active" db:"is_active"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	
	// Related data (not stored in DB)
	Campaign *AdvertisementCampaign `json:"campaign,omitempty"`
	Slot     *AdvertisementSlot     `json:"slot,omitempty"`
	Creative *AdvertisementCreative `json:"creative,omitempty"`
}

// AdvertisementImpression represents an ad impression
type AdvertisementImpression struct {
	ID          uint64    `json:"id" db:"id"`
	PlacementID uint64    `json:"placement_id" db:"placement_id" validate:"required"`
	CampaignID  uint64    `json:"campaign_id" db:"campaign_id" validate:"required"`
	SlotID      uint64    `json:"slot_id" db:"slot_id" validate:"required"`
	CreativeID  uint64    `json:"creative_id" db:"creative_id" validate:"required"`
	IPAddress   string    `json:"ip_address" db:"ip_address"`
	UserAgent   string    `json:"user_agent" db:"user_agent"`
	Referer     string    `json:"referer" db:"referer"`
	PageURL     string    `json:"page_url" db:"page_url"`
	DeviceType  string    `json:"device_type" db:"device_type" validate:"omitempty,oneof=mobile tablet desktop"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// AdvertisementClick represents an ad click
type AdvertisementClick struct {
	ID           uint64    `json:"id" db:"id"`
	PlacementID  uint64    `json:"placement_id" db:"placement_id" validate:"required"`
	CampaignID   uint64    `json:"campaign_id" db:"campaign_id" validate:"required"`
	SlotID       uint64    `json:"slot_id" db:"slot_id" validate:"required"`
	CreativeID   uint64    `json:"creative_id" db:"creative_id" validate:"required"`
	ImpressionID *uint64   `json:"impression_id" db:"impression_id"`
	IPAddress    string    `json:"ip_address" db:"ip_address"`
	UserAgent    string    `json:"user_agent" db:"user_agent"`
	Referer      string    `json:"referer" db:"referer"`
	PageURL      string    `json:"page_url" db:"page_url"`
	DeviceType   string    `json:"device_type" db:"device_type" validate:"omitempty,oneof=mobile tablet desktop"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// AdvertisementStats represents aggregated advertisement statistics
type AdvertisementStats struct {
	CampaignID   uint64  `json:"campaign_id"`
	Impressions  uint64  `json:"impressions"`
	Clicks       uint64  `json:"clicks"`
	CTR          float64 `json:"ctr"` // Click-through rate
	Spend        float64 `json:"spend"`
	CPM          float64 `json:"cpm"` // Cost per mille (thousand impressions)
	CPC          float64 `json:"cpc"` // Cost per click
	Period       string  `json:"period"` // daily, weekly, monthly
	StartDate    time.Time `json:"start_date"`
	EndDate      time.Time `json:"end_date"`
}

// AdvertisementRequest represents a request for advertisements for a specific context
type AdvertisementRequest struct {
	PageType     string            `json:"page_type" validate:"required,oneof=homepage article category tag search"`
	Position     string            `json:"position" validate:"required"`
	CategoryID   *uint64           `json:"category_id,omitempty"`
	TagIDs       []uint64          `json:"tag_ids,omitempty"`
	DeviceType   string            `json:"device_type" validate:"omitempty,oneof=mobile tablet desktop"`
	UserAgent    string            `json:"user_agent"`
	IPAddress    string            `json:"ip_address"`
	PageURL      string            `json:"page_url"`
	Referer      string            `json:"referer"`
	MaxAds       int               `json:"max_ads" validate:"min=1,max=10"`
	ExcludeAds   []uint64          `json:"exclude_ads,omitempty"` // Exclude specific campaign IDs
	TestMode     bool              `json:"test_mode"` // For A/B testing
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// AdvertisementResponse represents the response containing ads to display
type AdvertisementResponse struct {
	Ads           []AdvertisementAd `json:"ads"`
	TotalAds      int               `json:"total_ads"`
	RequestID     string            `json:"request_id"`
	CacheTTL      int               `json:"cache_ttl"` // in seconds
	PerformanceMs int               `json:"performance_ms"`
}

// AdvertisementAd represents a single ad to be displayed
type AdvertisementAd struct {
	ID           string `json:"id"` // Unique identifier for this ad serving
	PlacementID  uint64 `json:"placement_id"`
	CampaignID   uint64 `json:"campaign_id"`
	SlotID       uint64 `json:"slot_id"`
	CreativeID   uint64 `json:"creative_id"`
	Type         string `json:"type"`
	Content      string `json:"content"`
	AltText      string `json:"alt_text"`
	ClickURL     string `json:"click_url"`
	Width        *int   `json:"width"`
	Height       *int   `json:"height"`
	LazyLoad     bool   `json:"lazy_load"`
	TrackingURL  string `json:"tracking_url"` // URL for impression tracking
	ClickTrackURL string `json:"click_track_url"` // URL for click tracking
	Weight       int    `json:"weight"`
	Priority     int    `json:"priority"`
}

// AdvertisementPerformanceReport represents performance metrics for reporting
type AdvertisementPerformanceReport struct {
	CampaignID      uint64    `json:"campaign_id"`
	CampaignName    string    `json:"campaign_name"`
	SlotID          uint64    `json:"slot_id"`
	SlotName        string    `json:"slot_name"`
	CreativeID      uint64    `json:"creative_id"`
	CreativeName    string    `json:"creative_name"`
	Impressions     uint64    `json:"impressions"`
	Clicks          uint64    `json:"clicks"`
	CTR             float64   `json:"ctr"`
	UniqueViews     uint64    `json:"unique_views"`
	Spend           float64   `json:"spend"`
	Revenue         float64   `json:"revenue"`
	CPM             float64   `json:"cpm"`
	CPC             float64   `json:"cpc"`
	ConversionRate  float64   `json:"conversion_rate"`
	ViewabilityRate float64   `json:"viewability_rate"`
	LoadTime        float64   `json:"load_time_ms"`
	ErrorRate       float64   `json:"error_rate"`
	Period          string    `json:"period"`
	Date            time.Time `json:"date"`
}

// Validation methods

// IsValidCampaign validates campaign data
func (c *AdvertisementCampaign) IsValidCampaign() error {
	if c.Name == "" {
		return NewValidationError("name", "Campaign name is required")
	}
	if c.AdvertiserName == "" {
		return NewValidationError("advertiser_name", "Advertiser name is required")
	}
	if c.StartDate.IsZero() {
		return NewValidationError("start_date", "Start date is required")
	}
	if c.EndDate != nil && c.EndDate.Before(c.StartDate) {
		return NewValidationError("end_date", "End date must be after start date")
	}
	if c.Priority < 1 || c.Priority > 10 {
		return NewValidationError("priority", "Priority must be between 1 and 10")
	}
	return nil
}

// IsValidSlot validates slot data
func (s *AdvertisementSlot) IsValidSlot() error {
	if s.Name == "" {
		return NewValidationError("name", "Slot name is required")
	}
	if s.Slug == "" {
		return NewValidationError("slug", "Slot slug is required")
	}
	validPageTypes := []string{"homepage", "article", "category", "tag", "search", "all"}
	if !containsString(validPageTypes, s.PageType) {
		return NewValidationError("page_type", "Invalid page type")
	}
	validPositions := []string{"header", "sidebar", "content-top", "content-middle", "content-bottom", "footer", "floating"}
	if !containsString(validPositions, s.Position) {
		return NewValidationError("position", "Invalid position")
	}
	return nil
}

// IsValidCreative validates creative data
func (c *AdvertisementCreative) IsValidCreative() error {
	if c.Name == "" {
		return NewValidationError("name", "Creative name is required")
	}
	if c.Content == "" {
		return NewValidationError("content", "Creative content is required")
	}
	validTypes := []string{"image", "html", "script", "video"}
	if !containsString(validTypes, c.Type) {
		return NewValidationError("type", "Invalid creative type")
	}
	return nil
}

// Helper function to check if slice contains string
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}