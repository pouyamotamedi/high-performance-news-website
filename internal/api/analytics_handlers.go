package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// AnalyticsHandlers handles analytics-related HTTP requests
type AnalyticsHandlers struct {
	analyticsService *services.AnalyticsService
}

// NewAnalyticsHandlers creates new analytics handlers
func NewAnalyticsHandlers(analyticsService *services.AnalyticsService) *AnalyticsHandlers {
	return &AnalyticsHandlers{
		analyticsService: analyticsService,
	}
}

// TrackView handles article view tracking
func (h *AnalyticsHandlers) TrackView(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	articleIDStr := vars["articleId"]
	
	articleID, err := strconv.ParseUint(articleIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid article ID", http.StatusBadRequest)
		return
	}

	err = h.analyticsService.TrackArticleView(r.Context(), articleID, r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to track view: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TrackEngagement handles engagement tracking (likes, dislikes, shares, comments)
func (h *AnalyticsHandlers) TrackEngagement(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	articleIDStr := vars["articleId"]
	actionStr := vars["action"]
	
	articleID, err := strconv.ParseUint(articleIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid article ID", http.StatusBadRequest)
		return
	}

	var action models.EngagementAction
	switch actionStr {
	case "like":
		action = models.ActionLike
	case "dislike":
		action = models.ActionDislike
	case "share":
		action = models.ActionShare
	case "comment":
		action = models.ActionComment
	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	err = h.analyticsService.TrackEngagement(r.Context(), articleID, action, r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to track engagement: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TrackBehavior handles user behavior tracking
func (h *AnalyticsHandlers) TrackBehavior(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID    string  `json:"session_id"`
		UserID       *uint64 `json:"user_id,omitempty"`
		PageURL      string  `json:"page_url"`
		TimeOnPage   int     `json:"time_on_page"`
		ScrollDepth  float64 `json:"scroll_depth"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.analyticsService.TrackUserBehavior(r.Context(), req.SessionID, req.UserID, 
		req.PageURL, req.TimeOnPage, req.ScrollDepth, r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to track behavior: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetArticleAnalytics retrieves analytics for a specific article
func (h *AnalyticsHandlers) GetArticleAnalytics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	articleIDStr := vars["articleId"]
	
	articleID, err := strconv.ParseUint(articleIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid article ID", http.StatusBadRequest)
		return
	}

	// Parse date range parameters
	startDate, endDate, err := h.parseDateRange(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid date range: %v", err), http.StatusBadRequest)
		return
	}

	analytics, err := h.analyticsService.GetArticleAnalytics(r.Context(), articleID, startDate, endDate)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get analytics: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analytics)
}

// GetDashboard retrieves dashboard metrics
func (h *AnalyticsHandlers) GetDashboard(w http.ResponseWriter, r *http.Request) {
	// Parse date range parameters
	startDate, endDate, err := h.parseDateRange(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid date range: %v", err), http.StatusBadRequest)
		return
	}

	metrics, err := h.analyticsService.GetDashboardMetrics(r.Context(), startDate, endDate)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get dashboard metrics: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// GetTopArticles retrieves top performing articles
func (h *AnalyticsHandlers) GetTopArticles(w http.ResponseWriter, r *http.Request) {
	// Parse date range parameters
	startDate, endDate, err := h.parseDateRange(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid date range: %v", err), http.StatusBadRequest)
		return
	}

	// Parse limit parameter
	limitStr := r.URL.Query().Get("limit")
	limit := 10 // default
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	articles, err := h.analyticsService.GetTopArticles(r.Context(), startDate, endDate, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get top articles: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"articles": articles,
		"total":    len(articles),
	})
}

// GetTrafficSources retrieves traffic source analytics
func (h *AnalyticsHandlers) GetTrafficSources(w http.ResponseWriter, r *http.Request) {
	// Parse date range parameters
	startDate, endDate, err := h.parseDateRange(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid date range: %v", err), http.StatusBadRequest)
		return
	}

	sources, err := h.analyticsService.GetTrafficSources(r.Context(), startDate, endDate)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get traffic sources: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sources": sources,
		"total":   len(sources),
	})
}

// GenerateReport generates an analytics report
func (h *AnalyticsHandlers) GenerateReport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ReportType models.ReportType       `json:"report_type"`
		Parameters models.ReportParameters `json:"parameters"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get user ID from context (assuming it's set by auth middleware)
	userID := uint64(1) // TODO: Get from auth context

	report, err := h.analyticsService.GenerateReport(r.Context(), req.ReportType, req.Parameters, userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate report: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// GetReport retrieves a saved analytics report
func (h *AnalyticsHandlers) GetReport(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	reportIDStr := vars["reportId"]
	
	reportID, err := strconv.ParseUint(reportIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid report ID", http.StatusBadRequest)
		return
	}

	report, err := h.analyticsService.GetReport(r.Context(), reportID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get report: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// ExportData exports analytics data in various formats
func (h *AnalyticsHandlers) ExportData(w http.ResponseWriter, r *http.Request) {
	var req models.ExportRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	data, contentType, err := h.analyticsService.ExportData(r.Context(), req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to export data: %v", err), http.StatusInternalServerError)
		return
	}

	// Set appropriate headers
	w.Header().Set("Content-Type", contentType)
	
	// Set filename based on format
	var filename string
	switch req.Format {
	case models.ExportFormatCSV:
		filename = fmt.Sprintf("analytics_%s_%s.csv", req.ReportType, time.Now().Format("20060102"))
	case models.ExportFormatJSON:
		filename = fmt.Sprintf("analytics_%s_%s.json", req.ReportType, time.Now().Format("20060102"))
	default:
		filename = fmt.Sprintf("analytics_%s_%s", req.ReportType, time.Now().Format("20060102"))
	}
	
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Write(data)
}

// BulkTrackViews handles bulk view tracking
func (h *AnalyticsHandlers) BulkTrackViews(w http.ResponseWriter, r *http.Request) {
	var views []models.ArticleView

	if err := json.NewDecoder(r.Body).Decode(&views); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.analyticsService.BulkTrackViews(r.Context(), views)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to bulk track views: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// BulkTrackEngagements handles bulk engagement tracking
func (h *AnalyticsHandlers) BulkTrackEngagements(w http.ResponseWriter, r *http.Request) {
	var engagements []models.ArticleEngagement

	if err := json.NewDecoder(r.Body).Decode(&engagements); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.analyticsService.BulkTrackEngagements(r.Context(), engagements)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to bulk track engagements: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper methods

// parseDateRange parses start_date and end_date query parameters
func (h *AnalyticsHandlers) parseDateRange(r *http.Request) (time.Time, time.Time, error) {
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	// Default to last 30 days if not provided
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)

	if startDateStr != "" {
		parsed, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid start_date format, use YYYY-MM-DD")
		}
		startDate = parsed
	}

	if endDateStr != "" {
		parsed, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid end_date format, use YYYY-MM-DD")
		}
		endDate = parsed
	}

	if endDate.Before(startDate) {
		return time.Time{}, time.Time{}, fmt.Errorf("end_date must be after start_date")
	}

	return startDate, endDate, nil
}

// RegisterRoutes registers analytics routes
func (h *AnalyticsHandlers) RegisterRoutes(router *mux.Router) {
	// Tracking endpoints
	router.HandleFunc("/api/analytics/track/view/{articleId}", h.TrackView).Methods("POST")
	router.HandleFunc("/api/analytics/track/engagement/{articleId}/{action}", h.TrackEngagement).Methods("POST")
	router.HandleFunc("/api/analytics/track/behavior", h.TrackBehavior).Methods("POST")
	router.HandleFunc("/api/analytics/track/views/bulk", h.BulkTrackViews).Methods("POST")
	router.HandleFunc("/api/analytics/track/engagements/bulk", h.BulkTrackEngagements).Methods("POST")

	// Analytics endpoints
	router.HandleFunc("/api/analytics/articles/{articleId}", h.GetArticleAnalytics).Methods("GET")
	router.HandleFunc("/api/analytics/dashboard", h.GetDashboard).Methods("GET")
	router.HandleFunc("/api/analytics/articles/top", h.GetTopArticles).Methods("GET")
	router.HandleFunc("/api/analytics/traffic-sources", h.GetTrafficSources).Methods("GET")

	// Reporting endpoints
	router.HandleFunc("/api/analytics/reports", h.GenerateReport).Methods("POST")
	router.HandleFunc("/api/analytics/reports/{reportId}", h.GetReport).Methods("GET")
	router.HandleFunc("/api/analytics/export", h.ExportData).Methods("POST")
}