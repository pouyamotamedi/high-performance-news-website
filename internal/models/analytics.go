package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// ArticleView represents a view event for an article
type ArticleView struct {
	ID        uint64    `json:"id" db:"id"`
	ArticleID uint64    `json:"article_id" db:"article_id"`
	IPAddress string    `json:"ip_address" db:"ip_address"`
	UserAgent string    `json:"user_agent" db:"user_agent"`
	Referer   string    `json:"referer" db:"referer"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// ArticleEngagement represents engagement actions on articles
type ArticleEngagement struct {
	ID        uint64           `json:"id" db:"id"`
	ArticleID uint64           `json:"article_id" db:"article_id"`
	Action    EngagementAction `json:"action" db:"action"`
	IPAddress string           `json:"ip_address" db:"ip_address"`
	CreatedAt time.Time        `json:"created_at" db:"created_at"`
}

// EngagementAction represents different types of engagement
type EngagementAction string

const (
	ActionLike    EngagementAction = "like"
	ActionDislike EngagementAction = "dislike"
	ActionShare   EngagementAction = "share"
	ActionComment EngagementAction = "comment"
)

// UserBehavior represents user behavior analytics
type UserBehavior struct {
	ID            uint64                 `json:"id" db:"id"`
	SessionID     string                 `json:"session_id" db:"session_id"`
	UserID        *uint64                `json:"user_id" db:"user_id"`
	IPAddress     string                 `json:"ip_address" db:"ip_address"`
	UserAgent     string                 `json:"user_agent" db:"user_agent"`
	PageURL       string                 `json:"page_url" db:"page_url"`
	Referer       string                 `json:"referer" db:"referer"`
	TimeOnPage    int                    `json:"time_on_page" db:"time_on_page"` // seconds
	ScrollDepth   float64                `json:"scroll_depth" db:"scroll_depth"` // percentage
	BehaviorData  BehaviorData           `json:"behavior_data" db:"behavior_data"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
}

// BehaviorData contains additional behavior tracking data
type BehaviorData struct {
	Device       string            `json:"device"`        // mobile, tablet, desktop
	Browser      string            `json:"browser"`       // chrome, firefox, safari, etc.
	OS           string            `json:"os"`            // windows, macos, linux, android, ios
	Country      string            `json:"country"`       // from IP geolocation
	City         string            `json:"city"`          // from IP geolocation
	Language     string            `json:"language"`      // browser language
	ScreenSize   string            `json:"screen_size"`   // 1920x1080, etc.
	UTMSource    string            `json:"utm_source"`    // traffic source
	UTMMedium    string            `json:"utm_medium"`    // traffic medium
	UTMCampaign  string            `json:"utm_campaign"`  // campaign name
	CustomEvents []CustomEvent     `json:"custom_events"` // custom tracking events
}

// CustomEvent represents custom tracking events
type CustomEvent struct {
	Name      string                 `json:"name"`
	Value     string                 `json:"value"`
	Metadata  map[string]interface{} `json:"metadata"`
	Timestamp time.Time              `json:"timestamp"`
}

// Scan implements the sql.Scanner interface for BehaviorData
func (b *BehaviorData) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, b)
	case string:
		return json.Unmarshal([]byte(v), b)
	default:
		return fmt.Errorf("cannot scan %T into BehaviorData", value)
	}
}

// Value implements the driver.Valuer interface for BehaviorData
func (b BehaviorData) Value() (driver.Value, error) {
	return json.Marshal(b)
}

// PerformanceMetric represents system performance metrics
type PerformanceMetric struct {
	ID         uint64                 `json:"id" db:"id"`
	MetricType MetricType             `json:"metric_type" db:"metric_type"`
	Name       string                 `json:"name" db:"name"`
	Value      float64                `json:"value" db:"value"`
	Unit       string                 `json:"unit" db:"unit"`
	Tags       map[string]interface{} `json:"tags" db:"tags"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
}

// MetricType represents different types of performance metrics
type MetricType string

const (
	MetricTypeResponse    MetricType = "response_time"
	MetricTypeDatabase    MetricType = "database_query"
	MetricTypeCache       MetricType = "cache_operation"
	MetricTypeMemory      MetricType = "memory_usage"
	MetricTypeCPU         MetricType = "cpu_usage"
	MetricTypeDisk        MetricType = "disk_usage"
	MetricTypeNetwork     MetricType = "network_io"
	MetricTypeCustom      MetricType = "custom"
)

// AnalyticsReport represents generated analytics reports
type AnalyticsReport struct {
	ID          uint64                 `json:"id" db:"id"`
	Name        string                 `json:"name" db:"name"`
	ReportType  ReportType             `json:"report_type" db:"report_type"`
	Parameters  ReportParameters       `json:"parameters" db:"parameters"`
	Data        map[string]interface{} `json:"data" db:"data"`
	GeneratedBy uint64                 `json:"generated_by" db:"generated_by"`
	GeneratedAt time.Time              `json:"generated_at" db:"generated_at"`
	ExpiresAt   *time.Time             `json:"expires_at" db:"expires_at"`
}

// ReportType represents different types of analytics reports
type ReportType string

const (
	ReportTypeArticlePerformance ReportType = "article_performance"
	ReportTypeUserBehavior       ReportType = "user_behavior"
	ReportTypeEngagement         ReportType = "engagement"
	ReportTypeTrafficSources     ReportType = "traffic_sources"
	ReportTypePerformance        ReportType = "performance"
	ReportTypeCustom             ReportType = "custom"
)

// ReportParameters contains parameters for report generation
type ReportParameters struct {
	StartDate   time.Time              `json:"start_date"`
	EndDate     time.Time              `json:"end_date"`
	Filters     map[string]interface{} `json:"filters"`
	GroupBy     []string               `json:"group_by"`
	Metrics     []string               `json:"metrics"`
	Limit       int                    `json:"limit"`
	Offset      int                    `json:"offset"`
}

// Scan implements the sql.Scanner interface for ReportParameters
func (r *ReportParameters) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, r)
	case string:
		return json.Unmarshal([]byte(v), r)
	default:
		return fmt.Errorf("cannot scan %T into ReportParameters", value)
	}
}

// Value implements the driver.Valuer interface for ReportParameters
func (r ReportParameters) Value() (driver.Value, error) {
	return json.Marshal(r)
}

// ArticleAnalytics represents aggregated analytics for an article
type ArticleAnalytics struct {
	ArticleID       uint64    `json:"article_id"`
	Title           string    `json:"title"`
	Slug            string    `json:"slug"`
	PublishedAt     time.Time `json:"published_at"`
	ViewCount       int64     `json:"view_count"`
	UniqueViews     int64     `json:"unique_views"`
	LikeCount       int64     `json:"like_count"`
	DislikeCount    int64     `json:"dislike_count"`
	ShareCount      int64     `json:"share_count"`
	CommentCount    int64     `json:"comment_count"`
	AvgTimeOnPage   float64   `json:"avg_time_on_page"`
	AvgScrollDepth  float64   `json:"avg_scroll_depth"`
	BounceRate      float64   `json:"bounce_rate"`
	EngagementRate  float64   `json:"engagement_rate"`
	TopReferers     []string  `json:"top_referers"`
	TopCountries    []string  `json:"top_countries"`
	DeviceBreakdown map[string]int64 `json:"device_breakdown"`
}

// TrafficSource represents traffic source analytics
type TrafficSource struct {
	Source      string  `json:"source"`
	Medium      string  `json:"medium"`
	Campaign    string  `json:"campaign"`
	Sessions    int64   `json:"sessions"`
	Users       int64   `json:"users"`
	PageViews   int64   `json:"page_views"`
	BounceRate  float64 `json:"bounce_rate"`
	AvgDuration float64 `json:"avg_duration"`
}

// DashboardMetrics represents key metrics for the analytics dashboard
type DashboardMetrics struct {
	TotalViews        int64                    `json:"total_views"`
	UniqueVisitors    int64                    `json:"unique_visitors"`
	TotalEngagements  int64                    `json:"total_engagements"`
	AvgTimeOnSite     float64                  `json:"avg_time_on_site"`
	BounceRate        float64                  `json:"bounce_rate"`
	TopArticles       []ArticleAnalytics       `json:"top_articles"`
	TrafficSources    []TrafficSource          `json:"traffic_sources"`
	DeviceBreakdown   map[string]int64         `json:"device_breakdown"`
	CountryBreakdown  map[string]int64         `json:"country_breakdown"`
	HourlyTraffic     []HourlyTrafficData      `json:"hourly_traffic"`
	PerformanceMetrics map[string]float64      `json:"performance_metrics"`
}

// HourlyTrafficData represents traffic data by hour
type HourlyTrafficData struct {
	Hour      int   `json:"hour"`
	Views     int64 `json:"views"`
	Visitors  int64 `json:"visitors"`
}

// ExportFormat represents different export formats
type ExportFormat string

const (
	ExportFormatCSV  ExportFormat = "csv"
	ExportFormatJSON ExportFormat = "json"
	ExportFormatXLSX ExportFormat = "xlsx"
	ExportFormatPDF  ExportFormat = "pdf"
)

// ExportRequest represents a request to export analytics data
type ExportRequest struct {
	ReportType ReportType       `json:"report_type" validate:"required"`
	Parameters ReportParameters `json:"parameters" validate:"required"`
	Format     ExportFormat     `json:"format" validate:"required"`
	Email      string           `json:"email,omitempty" validate:"omitempty,email"`
}



// ValidateArticleView validates article view data
func ValidateArticleView(view *ArticleView) error {
	var errors []string

	if view.ArticleID == 0 {
		errors = append(errors, "article_id is required")
	}

	if view.IPAddress == "" {
		errors = append(errors, "ip_address is required")
	}

	if len(errors) > 0 {
		return &ValidationError{
			Message: "Article view validation failed",
			Fields:  errors,
		}
	}

	return nil
}

// ValidateArticleEngagement validates article engagement data
func ValidateArticleEngagement(engagement *ArticleEngagement) error {
	var errors []string

	if engagement.ArticleID == 0 {
		errors = append(errors, "article_id is required")
	}

	validActions := map[EngagementAction]bool{
		ActionLike:    true,
		ActionDislike: true,
		ActionShare:   true,
		ActionComment: true,
	}
	if !validActions[engagement.Action] {
		errors = append(errors, "invalid action")
	}

	if engagement.IPAddress == "" {
		errors = append(errors, "ip_address is required")
	}

	if len(errors) > 0 {
		return &ValidationError{
			Message: "Article engagement validation failed",
			Fields:  errors,
		}
	}

	return nil
}

// ValidateExportRequest validates export request data
func ValidateExportRequest(req *ExportRequest) error {
	var errors []string

	validReportTypes := map[ReportType]bool{
		ReportTypeArticlePerformance: true,
		ReportTypeUserBehavior:       true,
		ReportTypeEngagement:         true,
		ReportTypeTrafficSources:     true,
		ReportTypePerformance:        true,
		ReportTypeCustom:             true,
	}
	if !validReportTypes[req.ReportType] {
		errors = append(errors, "invalid report_type")
	}

	validFormats := map[ExportFormat]bool{
		ExportFormatCSV:  true,
		ExportFormatJSON: true,
		ExportFormatXLSX: true,
		ExportFormatPDF:  true,
	}
	if !validFormats[req.Format] {
		errors = append(errors, "invalid format")
	}

	if req.Parameters.StartDate.IsZero() {
		errors = append(errors, "start_date is required")
	}

	if req.Parameters.EndDate.IsZero() {
		errors = append(errors, "end_date is required")
	}

	if req.Parameters.EndDate.Before(req.Parameters.StartDate) {
		errors = append(errors, "end_date must be after start_date")
	}

	if len(errors) > 0 {
		return &ValidationError{
			Message: "Export request validation failed",
			Fields:  errors,
		}
	}

	return nil
}