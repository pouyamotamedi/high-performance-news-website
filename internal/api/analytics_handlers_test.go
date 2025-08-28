package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"high-performance-news-website/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAnalyticsService is a mock implementation of AnalyticsService
type MockAnalyticsService struct {
	mock.Mock
}

func (m *MockAnalyticsService) TrackArticleView(ctx context.Context, articleID uint64, r *http.Request) error {
	args := m.Called(ctx, articleID, r)
	return args.Error(0)
}

func (m *MockAnalyticsService) TrackEngagement(ctx context.Context, articleID uint64, action models.EngagementAction, r *http.Request) error {
	args := m.Called(ctx, articleID, action, r)
	return args.Error(0)
}

func (m *MockAnalyticsService) TrackUserBehavior(ctx context.Context, sessionID string, userID *uint64, pageURL string, timeOnPage int, scrollDepth float64, r *http.Request) error {
	args := m.Called(ctx, sessionID, userID, pageURL, timeOnPage, scrollDepth, r)
	return args.Error(0)
}

func (m *MockAnalyticsService) GetArticleAnalytics(ctx context.Context, articleID uint64, startDate, endDate time.Time) (*models.ArticleAnalytics, error) {
	args := m.Called(ctx, articleID, startDate, endDate)
	return args.Get(0).(*models.ArticleAnalytics), args.Error(1)
}

func (m *MockAnalyticsService) GetDashboardMetrics(ctx context.Context, startDate, endDate time.Time) (*models.DashboardMetrics, error) {
	args := m.Called(ctx, startDate, endDate)
	return args.Get(0).(*models.DashboardMetrics), args.Error(1)
}

func (m *MockAnalyticsService) GetTopArticles(ctx context.Context, startDate, endDate time.Time, limit int) ([]models.ArticleAnalytics, error) {
	args := m.Called(ctx, startDate, endDate, limit)
	return args.Get(0).([]models.ArticleAnalytics), args.Error(1)
}

func (m *MockAnalyticsService) GetTrafficSources(ctx context.Context, startDate, endDate time.Time) ([]models.TrafficSource, error) {
	args := m.Called(ctx, startDate, endDate)
	return args.Get(0).([]models.TrafficSource), args.Error(1)
}

func (m *MockAnalyticsService) GenerateReport(ctx context.Context, reportType models.ReportType, params models.ReportParameters, generatedBy uint64) (*models.AnalyticsReport, error) {
	args := m.Called(ctx, reportType, params, generatedBy)
	return args.Get(0).(*models.AnalyticsReport), args.Error(1)
}

func (m *MockAnalyticsService) GetReport(ctx context.Context, reportID uint64) (*models.AnalyticsReport, error) {
	args := m.Called(ctx, reportID)
	return args.Get(0).(*models.AnalyticsReport), args.Error(1)
}

func (m *MockAnalyticsService) ExportData(ctx context.Context, req models.ExportRequest) ([]byte, string, error) {
	args := m.Called(ctx, req)
	return args.Get(0).([]byte), args.String(1), args.Error(2)
}

func (m *MockAnalyticsService) BulkTrackViews(ctx context.Context, views []models.ArticleView) error {
	args := m.Called(ctx, views)
	return args.Error(0)
}

func (m *MockAnalyticsService) BulkTrackEngagements(ctx context.Context, engagements []models.ArticleEngagement) error {
	args := m.Called(ctx, engagements)
	return args.Error(0)
}

func TestAnalyticsHandlers_TrackView(t *testing.T) {
	mockService := new(MockAnalyticsService)
	handlers := NewAnalyticsHandlers(mockService)

	// Set up mock expectation
	mockService.On("TrackArticleView", mock.Anything, uint64(123), mock.AnythingOfType("*http.Request")).Return(nil)

	// Create request
	req := httptest.NewRequest("POST", "/api/analytics/track/view/123", nil)
	w := httptest.NewRecorder()

	// Set up router
	router := mux.NewRouter()
	router.HandleFunc("/api/analytics/track/view/{articleId}", handlers.TrackView).Methods("POST")

	// Execute request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusNoContent, w.Code)
	mockService.AssertExpectations(t)
}

func TestAnalyticsHandlers_TrackView_InvalidArticleID(t *testing.T) {
	mockService := new(MockAnalyticsService)
	handlers := NewAnalyticsHandlers(mockService)

	// Create request with invalid article ID
	req := httptest.NewRequest("POST", "/api/analytics/track/view/invalid", nil)
	w := httptest.NewRecorder()

	// Set up router
	router := mux.NewRouter()
	router.HandleFunc("/api/analytics/track/view/{articleId}", handlers.TrackView).Methods("POST")

	// Execute request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid article ID")
}

func TestAnalyticsHandlers_TrackEngagement(t *testing.T) {
	mockService := new(MockAnalyticsService)
	handlers := NewAnalyticsHandlers(mockService)

	// Set up mock expectation
	mockService.On("TrackEngagement", mock.Anything, uint64(123), models.ActionLike, mock.AnythingOfType("*http.Request")).Return(nil)

	// Create request
	req := httptest.NewRequest("POST", "/api/analytics/track/engagement/123/like", nil)
	w := httptest.NewRecorder()

	// Set up router
	router := mux.NewRouter()
	router.HandleFunc("/api/analytics/track/engagement/{articleId}/{action}", handlers.TrackEngagement).Methods("POST")

	// Execute request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusNoContent, w.Code)
	mockService.AssertExpectations(t)
}

func TestAnalyticsHandlers_TrackEngagement_InvalidAction(t *testing.T) {
	mockService := new(MockAnalyticsService)
	handlers := NewAnalyticsHandlers(mockService)

	// Create request with invalid action
	req := httptest.NewRequest("POST", "/api/analytics/track/engagement/123/invalid", nil)
	w := httptest.NewRecorder()

	// Set up router
	router := mux.NewRouter()
	router.HandleFunc("/api/analytics/track/engagement/{articleId}/{action}", handlers.TrackEngagement).Methods("POST")

	// Execute request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid action")
}

func TestAnalyticsHandlers_TrackBehavior(t *testing.T) {
	mockService := new(MockAnalyticsService)
	handlers := NewAnalyticsHandlers(mockService)

	// Test data
	userID := uint64(456)
	behaviorReq := struct {
		SessionID   string  `json:"session_id"`
		UserID      *uint64 `json:"user_id,omitempty"`
		PageURL     string  `json:"page_url"`
		TimeOnPage  int     `json:"time_on_page"`
		ScrollDepth float64 `json:"scroll_depth"`
	}{
		SessionID:   "session123",
		UserID:      &userID,
		PageURL:     "/article/test",
		TimeOnPage:  120,
		ScrollDepth: 85.5,
	}

	// Set up mock expectation
	mockService.On("TrackUserBehavior", mock.Anything, "session123", &userID, "/article/test", 120, 85.5, mock.AnythingOfType("*http.Request")).Return(nil)

	// Create request
	body, _ := json.Marshal(behaviorReq)
	req := httptest.NewRequest("POST", "/api/analytics/track/behavior", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute request
	handlers.TrackBehavior(w, req)

	// Assertions
	assert.Equal(t, http.StatusNoContent, w.Code)
	mockService.AssertExpectations(t)
}

func TestAnalyticsHandlers_GetArticleAnalytics(t *testing.T) {
	mockService := new(MockAnalyticsService)
	handlers := NewAnalyticsHandlers(mockService)

	// Expected result
	expectedAnalytics := &models.ArticleAnalytics{
		ArticleID:      123,
		Title:          "Test Article",
		Slug:           "test-article",
		ViewCount:      1000,
		UniqueViews:    750,
		LikeCount:      50,
		DislikeCount:   5,
		ShareCount:     25,
		CommentCount:   15,
		EngagementRate: 0.095,
	}

	// Set up mock expectation
	mockService.On("GetArticleAnalytics", mock.Anything, uint64(123), mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(expectedAnalytics, nil)

	// Create request
	req := httptest.NewRequest("GET", "/api/analytics/articles/123?start_date=2023-01-01&end_date=2023-01-31", nil)
	w := httptest.NewRecorder()

	// Set up router
	router := mux.NewRouter()
	router.HandleFunc("/api/analytics/articles/{articleId}", handlers.GetArticleAnalytics).Methods("GET")

	// Execute request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var result models.ArticleAnalytics
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Equal(t, expectedAnalytics.ArticleID, result.ArticleID)
	assert.Equal(t, expectedAnalytics.Title, result.Title)
	assert.Equal(t, expectedAnalytics.ViewCount, result.ViewCount)

	mockService.AssertExpectations(t)
}

func TestAnalyticsHandlers_GetDashboard(t *testing.T) {
	mockService := new(MockAnalyticsService)
	handlers := NewAnalyticsHandlers(mockService)

	// Expected result
	expectedMetrics := &models.DashboardMetrics{
		TotalViews:       10000,
		UniqueVisitors:   7500,
		TotalEngagements: 500,
		AvgTimeOnSite:    180.5,
		BounceRate:       0.35,
		DeviceBreakdown: map[string]int64{
			"desktop": 6000,
			"mobile":  3500,
			"tablet":  500,
		},
	}

	// Set up mock expectation
	mockService.On("GetDashboardMetrics", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(expectedMetrics, nil)

	// Create request
	req := httptest.NewRequest("GET", "/api/analytics/dashboard", nil)
	w := httptest.NewRecorder()

	// Execute request
	handlers.GetDashboard(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var result models.DashboardMetrics
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Equal(t, expectedMetrics.TotalViews, result.TotalViews)
	assert.Equal(t, expectedMetrics.UniqueVisitors, result.UniqueVisitors)

	mockService.AssertExpectations(t)
}

func TestAnalyticsHandlers_GetTopArticles(t *testing.T) {
	mockService := new(MockAnalyticsService)
	handlers := NewAnalyticsHandlers(mockService)

	// Expected result
	expectedArticles := []models.ArticleAnalytics{
		{
			ArticleID:   1,
			Title:       "Article 1",
			ViewCount:   1000,
			LikeCount:   50,
		},
		{
			ArticleID:   2,
			Title:       "Article 2",
			ViewCount:   800,
			LikeCount:   40,
		},
	}

	// Set up mock expectation
	mockService.On("GetTopArticles", mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time"), 10).Return(expectedArticles, nil)

	// Create request
	req := httptest.NewRequest("GET", "/api/analytics/articles/top?limit=10", nil)
	w := httptest.NewRecorder()

	// Execute request
	handlers.GetTopArticles(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Equal(t, float64(2), result["total"])

	mockService.AssertExpectations(t)
}

func TestAnalyticsHandlers_GenerateReport(t *testing.T) {
	mockService := new(MockAnalyticsService)
	handlers := NewAnalyticsHandlers(mockService)

	// Test data
	reportReq := struct {
		ReportType models.ReportType       `json:"report_type"`
		Parameters models.ReportParameters `json:"parameters"`
	}{
		ReportType: models.ReportTypeArticlePerformance,
		Parameters: models.ReportParameters{
			StartDate: time.Now().AddDate(0, 0, -7),
			EndDate:   time.Now(),
			Limit:     10,
		},
	}

	// Expected result
	expectedReport := &models.AnalyticsReport{
		ID:          1,
		Name:        "Article Performance Report",
		ReportType:  models.ReportTypeArticlePerformance,
		GeneratedBy: 1,
		GeneratedAt: time.Now(),
	}

	// Set up mock expectation
	mockService.On("GenerateReport", mock.Anything, models.ReportTypeArticlePerformance, mock.AnythingOfType("models.ReportParameters"), uint64(1)).Return(expectedReport, nil)

	// Create request
	body, _ := json.Marshal(reportReq)
	req := httptest.NewRequest("POST", "/api/analytics/reports", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute request
	handlers.GenerateReport(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var result models.AnalyticsReport
	err := json.Unmarshal(w.Body.Bytes(), &result)
	assert.NoError(t, err)
	assert.Equal(t, expectedReport.ID, result.ID)
	assert.Equal(t, expectedReport.ReportType, result.ReportType)

	mockService.AssertExpectations(t)
}

func TestAnalyticsHandlers_ExportData(t *testing.T) {
	mockService := new(MockAnalyticsService)
	handlers := NewAnalyticsHandlers(mockService)

	// Test data
	exportReq := models.ExportRequest{
		ReportType: models.ReportTypeEngagement,
		Parameters: models.ReportParameters{
			StartDate: time.Now().AddDate(0, 0, -7),
			EndDate:   time.Now(),
		},
		Format: models.ExportFormatCSV,
	}

	// Expected result
	expectedData := []byte("article_id,views,likes\n1,1000,50\n2,800,40\n")
	expectedContentType := "text/csv"

	// Set up mock expectation
	mockService.On("ExportData", mock.Anything, exportReq).Return(expectedData, expectedContentType, nil)

	// Create request
	body, _ := json.Marshal(exportReq)
	req := httptest.NewRequest("POST", "/api/analytics/export", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute request
	handlers.ExportData(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, expectedContentType, w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
	assert.Equal(t, expectedData, w.Body.Bytes())

	mockService.AssertExpectations(t)
}

func TestAnalyticsHandlers_BulkTrackViews(t *testing.T) {
	mockService := new(MockAnalyticsService)
	handlers := NewAnalyticsHandlers(mockService)

	// Test data
	views := []models.ArticleView{
		{
			ArticleID: 1,
			IPAddress: "192.168.1.1",
			UserAgent: "Mozilla/5.0",
			CreatedAt: time.Now(),
		},
		{
			ArticleID: 2,
			IPAddress: "192.168.1.2",
			UserAgent: "Mozilla/5.0",
			CreatedAt: time.Now(),
		},
	}

	// Set up mock expectation
	mockService.On("BulkTrackViews", mock.Anything, views).Return(nil)

	// Create request
	body, _ := json.Marshal(views)
	req := httptest.NewRequest("POST", "/api/analytics/track/views/bulk", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute request
	handlers.BulkTrackViews(w, req)

	// Assertions
	assert.Equal(t, http.StatusNoContent, w.Code)
	mockService.AssertExpectations(t)
}

func TestAnalyticsHandlers_ParseDateRange(t *testing.T) {
	handlers := &AnalyticsHandlers{}

	tests := []struct {
		name        string
		startDate   string
		endDate     string
		expectError bool
	}{
		{
			name:        "Valid date range",
			startDate:   "2023-01-01",
			endDate:     "2023-01-31",
			expectError: false,
		},
		{
			name:        "Invalid start date format",
			startDate:   "2023/01/01",
			endDate:     "2023-01-31",
			expectError: true,
		},
		{
			name:        "Invalid end date format",
			startDate:   "2023-01-01",
			endDate:     "2023/01/31",
			expectError: true,
		},
		{
			name:        "End date before start date",
			startDate:   "2023-01-31",
			endDate:     "2023-01-01",
			expectError: true,
		},
		{
			name:        "No dates provided (should use defaults)",
			startDate:   "",
			endDate:     "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			q := req.URL.Query()
			if tt.startDate != "" {
				q.Add("start_date", tt.startDate)
			}
			if tt.endDate != "" {
				q.Add("end_date", tt.endDate)
			}
			req.URL.RawQuery = q.Encode()

			startDate, endDate, err := handlers.parseDateRange(req)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.True(t, endDate.After(startDate) || endDate.Equal(startDate))
			}
		})
	}
}