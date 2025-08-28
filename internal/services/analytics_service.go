package services

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/repositories"
)

// AnalyticsService handles analytics operations
type AnalyticsService struct {
	repo *repositories.AnalyticsRepository
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(repo *repositories.AnalyticsRepository) *AnalyticsService {
	return &AnalyticsService{
		repo: repo,
	}
}

// TrackArticleView records an article view with IP-based analytics
func (s *AnalyticsService) TrackArticleView(ctx context.Context, articleID uint64, r *http.Request) error {
	view := &models.ArticleView{
		ArticleID: articleID,
		IPAddress: s.getClientIP(r),
		UserAgent: r.UserAgent(),
		Referer:   r.Referer(),
		CreatedAt: time.Now(),
	}

	if err := models.ValidateArticleView(view); err != nil {
		return err
	}

	return s.repo.RecordArticleView(view)
}

// TrackEngagement records user engagement (likes, dislikes, shares, comments)
func (s *AnalyticsService) TrackEngagement(ctx context.Context, articleID uint64, action models.EngagementAction, r *http.Request) error {
	engagement := &models.ArticleEngagement{
		ArticleID: articleID,
		Action:    action,
		IPAddress: s.getClientIP(r),
		CreatedAt: time.Now(),
	}

	if err := models.ValidateArticleEngagement(engagement); err != nil {
		return err
	}

	return s.repo.RecordArticleEngagement(engagement)
}

// TrackUserBehavior records detailed user behavior analytics
func (s *AnalyticsService) TrackUserBehavior(ctx context.Context, sessionID string, userID *uint64, 
	pageURL string, timeOnPage int, scrollDepth float64, r *http.Request) error {
	
	behaviorData := s.extractBehaviorData(r)
	
	behavior := &models.UserBehavior{
		SessionID:    sessionID,
		UserID:       userID,
		IPAddress:    s.getClientIP(r),
		UserAgent:    r.UserAgent(),
		PageURL:      pageURL,
		Referer:      r.Referer(),
		TimeOnPage:   timeOnPage,
		ScrollDepth:  scrollDepth,
		BehaviorData: behaviorData,
		CreatedAt:    time.Now(),
	}

	return s.repo.RecordUserBehavior(behavior)
}

// TrackPerformanceMetric records system performance metrics
func (s *AnalyticsService) TrackPerformanceMetric(ctx context.Context, metricType models.MetricType, 
	name string, value float64, unit string, tags map[string]interface{}) error {
	
	metric := &models.PerformanceMetric{
		MetricType: metricType,
		Name:       name,
		Value:      value,
		Unit:       unit,
		Tags:       tags,
		CreatedAt:  time.Now(),
	}

	return s.repo.RecordPerformanceMetric(metric)
}

// GetArticleAnalytics retrieves comprehensive analytics for a specific article
func (s *AnalyticsService) GetArticleAnalytics(ctx context.Context, articleID uint64, startDate, endDate time.Time) (*models.ArticleAnalytics, error) {
	return s.repo.GetArticleAnalytics(articleID, startDate, endDate)
}

// GetDashboardMetrics retrieves key metrics for the analytics dashboard
func (s *AnalyticsService) GetDashboardMetrics(ctx context.Context, startDate, endDate time.Time) (*models.DashboardMetrics, error) {
	return s.repo.GetDashboardMetrics(startDate, endDate)
}

// GetTopArticles retrieves top performing articles
func (s *AnalyticsService) GetTopArticles(ctx context.Context, startDate, endDate time.Time, limit int) ([]models.ArticleAnalytics, error) {
	return s.repo.GetTopArticles(startDate, endDate, limit)
}

// GetTrafficSources retrieves traffic source analytics
func (s *AnalyticsService) GetTrafficSources(ctx context.Context, startDate, endDate time.Time) ([]models.TrafficSource, error) {
	return s.repo.GetTrafficSources(startDate, endDate)
}

// GenerateReport generates and saves an analytics report
func (s *AnalyticsService) GenerateReport(ctx context.Context, reportType models.ReportType, 
	params models.ReportParameters, generatedBy uint64) (*models.AnalyticsReport, error) {
	
	// Get data based on report type
	var data map[string]interface{}
	var err error

	switch reportType {
	case models.ReportTypeArticlePerformance:
		articles, err := s.repo.GetTopArticles(params.StartDate, params.EndDate, params.Limit)
		if err != nil {
			return nil, err
		}
		data = map[string]interface{}{
			"articles": articles,
			"total":    len(articles),
		}

	case models.ReportTypeUserBehavior:
		metrics, err := s.repo.GetDashboardMetrics(params.StartDate, params.EndDate)
		if err != nil {
			return nil, err
		}
		data = map[string]interface{}{
			"avg_time_on_site": metrics.AvgTimeOnSite,
			"bounce_rate":      metrics.BounceRate,
			"device_breakdown": metrics.DeviceBreakdown,
			"country_breakdown": metrics.CountryBreakdown,
		}

	case models.ReportTypeEngagement:
		data, err = s.repo.GetAnalyticsData(params)
		if err != nil {
			return nil, err
		}

	case models.ReportTypeTrafficSources:
		sources, err := s.repo.GetTrafficSources(params.StartDate, params.EndDate)
		if err != nil {
			return nil, err
		}
		data = map[string]interface{}{
			"sources": sources,
			"total":   len(sources),
		}

	default:
		return nil, fmt.Errorf("unsupported report type: %s", reportType)
	}

	// Create report
	report := &models.AnalyticsReport{
		Name:        fmt.Sprintf("%s Report - %s to %s", reportType, params.StartDate.Format("2006-01-02"), params.EndDate.Format("2006-01-02")),
		ReportType:  reportType,
		Parameters:  params,
		Data:        data,
		GeneratedBy: generatedBy,
		GeneratedAt: time.Now(),
	}

	// Set expiration (30 days from generation)
	expiresAt := time.Now().AddDate(0, 0, 30)
	report.ExpiresAt = &expiresAt

	// Save report
	err = s.repo.SaveReport(report)
	if err != nil {
		return nil, err
	}

	return report, nil
}

// ExportData exports analytics data in various formats
func (s *AnalyticsService) ExportData(ctx context.Context, req models.ExportRequest) ([]byte, string, error) {
	if err := models.ValidateExportRequest(&req); err != nil {
		return nil, "", err
	}

	// Get data
	data, err := s.repo.GetAnalyticsData(req.Parameters)
	if err != nil {
		return nil, "", err
	}

	// Export based on format
	switch req.Format {
	case models.ExportFormatJSON:
		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return nil, "", err
		}
		return jsonData, "application/json", nil

	case models.ExportFormatCSV:
		csvData, err := s.exportToCSV(data)
		if err != nil {
			return nil, "", err
		}
		return csvData, "text/csv", nil

	default:
		return nil, "", fmt.Errorf("unsupported export format: %s", req.Format)
	}
}

// BulkTrackViews efficiently records multiple article views
func (s *AnalyticsService) BulkTrackViews(ctx context.Context, views []models.ArticleView) error {
	// Validate all views
	for i, view := range views {
		if err := models.ValidateArticleView(&view); err != nil {
			return fmt.Errorf("validation failed for view %d: %w", i, err)
		}
	}

	return s.repo.BulkRecordViews(views)
}

// BulkTrackEngagements efficiently records multiple engagements
func (s *AnalyticsService) BulkTrackEngagements(ctx context.Context, engagements []models.ArticleEngagement) error {
	// Validate all engagements
	for i, engagement := range engagements {
		if err := models.ValidateArticleEngagement(&engagement); err != nil {
			return fmt.Errorf("validation failed for engagement %d: %w", i, err)
		}
	}

	return s.repo.BulkRecordEngagements(engagements)
}

// GetReport retrieves a saved analytics report
func (s *AnalyticsService) GetReport(ctx context.Context, reportID uint64) (*models.AnalyticsReport, error) {
	return s.repo.GetReport(reportID)
}

// Helper methods

// getClientIP extracts the real client IP address from the request
func (s *AnalyticsService) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for load balancers/proxies)
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xForwardedFor, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}

	// Check X-Real-IP header
	xRealIP := r.Header.Get("X-Real-IP")
	if xRealIP != "" {
		if net.ParseIP(xRealIP) != nil {
			return xRealIP
		}
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// extractBehaviorData extracts behavior data from the request
func (s *AnalyticsService) extractBehaviorData(r *http.Request) models.BehaviorData {
	data := models.BehaviorData{
		Device:      s.detectDevice(r.UserAgent()),
		Browser:     s.detectBrowser(r.UserAgent()),
		OS:          s.detectOS(r.UserAgent()),
		Language:    r.Header.Get("Accept-Language"),
		UTMSource:   r.URL.Query().Get("utm_source"),
		UTMMedium:   r.URL.Query().Get("utm_medium"),
		UTMCampaign: r.URL.Query().Get("utm_campaign"),
	}

	// Extract screen size if available
	if screenSize := r.Header.Get("X-Screen-Size"); screenSize != "" {
		data.ScreenSize = screenSize
	}

	return data
}

// detectDevice detects device type from user agent
func (s *AnalyticsService) detectDevice(userAgent string) string {
	ua := strings.ToLower(userAgent)
	
	if strings.Contains(ua, "mobile") || strings.Contains(ua, "android") || strings.Contains(ua, "iphone") {
		return "mobile"
	}
	if strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad") {
		return "tablet"
	}
	return "desktop"
}

// detectBrowser detects browser from user agent
func (s *AnalyticsService) detectBrowser(userAgent string) string {
	ua := strings.ToLower(userAgent)
	
	if strings.Contains(ua, "chrome") && !strings.Contains(ua, "edge") {
		return "chrome"
	}
	if strings.Contains(ua, "firefox") {
		return "firefox"
	}
	if strings.Contains(ua, "safari") && !strings.Contains(ua, "chrome") {
		return "safari"
	}
	if strings.Contains(ua, "edge") {
		return "edge"
	}
	if strings.Contains(ua, "opera") {
		return "opera"
	}
	return "unknown"
}

// detectOS detects operating system from user agent
func (s *AnalyticsService) detectOS(userAgent string) string {
	ua := strings.ToLower(userAgent)
	
	if strings.Contains(ua, "windows") {
		return "windows"
	}
	if strings.Contains(ua, "mac os") || strings.Contains(ua, "macos") {
		return "macos"
	}
	if strings.Contains(ua, "linux") {
		return "linux"
	}
	if strings.Contains(ua, "android") {
		return "android"
	}
	if strings.Contains(ua, "ios") || strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") {
		return "ios"
	}
	return "unknown"
}

// exportToCSV converts data to CSV format
func (s *AnalyticsService) exportToCSV(data map[string]interface{}) ([]byte, error) {
	var csvData strings.Builder
	writer := csv.NewWriter(&csvData)

	// Handle different data types
	for key, value := range data {
		switch v := value.(type) {
		case []models.ArticleAnalytics:
			// Write header
			header := []string{"Article ID", "Title", "Slug", "View Count", "Unique Views", 
				"Like Count", "Dislike Count", "Share Count", "Comment Count", "Engagement Rate"}
			writer.Write(header)

			// Write data
			for _, article := range v {
				record := []string{
					strconv.FormatUint(article.ArticleID, 10),
					article.Title,
					article.Slug,
					strconv.FormatInt(article.ViewCount, 10),
					strconv.FormatInt(article.UniqueViews, 10),
					strconv.FormatInt(article.LikeCount, 10),
					strconv.FormatInt(article.DislikeCount, 10),
					strconv.FormatInt(article.ShareCount, 10),
					strconv.FormatInt(article.CommentCount, 10),
					strconv.FormatFloat(article.EngagementRate, 'f', 4, 64),
				}
				writer.Write(record)
			}

		case map[string]int64:
			// Write header
			writer.Write([]string{key, "Count"})
			
			// Write data
			for k, count := range v {
				writer.Write([]string{k, strconv.FormatInt(count, 10)})
			}

		default:
			// Generic handling
			writer.Write([]string{key, fmt.Sprintf("%v", value)})
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return []byte(csvData.String()), nil
}