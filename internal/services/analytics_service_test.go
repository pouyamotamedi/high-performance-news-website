package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"high-performance-news-website/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAnalyticsRepository is a mock implementation of AnalyticsRepository
type MockAnalyticsRepository struct {
	mock.Mock
}

func (m *MockAnalyticsRepository) RecordArticleView(view *models.ArticleView) error {
	args := m.Called(view)
	return args.Error(0)
}

func (m *MockAnalyticsRepository) RecordArticleEngagement(engagement *models.ArticleEngagement) error {
	args := m.Called(engagement)
	return args.Error(0)
}

func (m *MockAnalyticsRepository) RecordUserBehavior(behavior *models.UserBehavior) error {
	args := m.Called(behavior)
	return args.Error(0)
}

func (m *MockAnalyticsRepository) RecordPerformanceMetric(metric *models.PerformanceMetric) error {
	args := m.Called(metric)
	return args.Error(0)
}

func (m *MockAnalyticsRepository) GetArticleAnalytics(articleID uint64, startDate, endDate time.Time) (*models.ArticleAnalytics, error) {
	args := m.Called(articleID, startDate, endDate)
	return args.Get(0).(*models.ArticleAnalytics), args.Error(1)
}

func (m *MockAnalyticsRepository) GetTopArticles(startDate, endDate time.Time, limit int) ([]models.ArticleAnalytics, error) {
	args := m.Called(startDate, endDate, limit)
	return args.Get(0).([]models.ArticleAnalytics), args.Error(1)
}

func (m *MockAnalyticsRepository) GetTrafficSources(startDate, endDate time.Time) ([]models.TrafficSource, error) {
	args := m.Called(startDate, endDate)
	return args.Get(0).([]models.TrafficSource), args.Error(1)
}

func (m *MockAnalyticsRepository) GetDashboardMetrics(startDate, endDate time.Time) (*models.DashboardMetrics, error) {
	args := m.Called(startDate, endDate)
	return args.Get(0).(*models.DashboardMetrics), args.Error(1)
}

func (m *MockAnalyticsRepository) SaveReport(report *models.AnalyticsReport) error {
	args := m.Called(report)
	return args.Error(0)
}

func (m *MockAnalyticsRepository) GetReport(reportID uint64) (*models.AnalyticsReport, error) {
	args := m.Called(reportID)
	return args.Get(0).(*models.AnalyticsReport), args.Error(1)
}

func (m *MockAnalyticsRepository) BulkRecordViews(views []models.ArticleView) error {
	args := m.Called(views)
	return args.Error(0)
}

func (m *MockAnalyticsRepository) BulkRecordEngagements(engagements []models.ArticleEngagement) error {
	args := m.Called(engagements)
	return args.Error(0)
}

func (m *MockAnalyticsRepository) GetAnalyticsData(params models.ReportParameters) (map[string]interface{}, error) {
	args := m.Called(params)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func TestAnalyticsService_TrackArticleView(t *testing.T) {
	mockRepo := new(MockAnalyticsRepository)
	service := NewAnalyticsService(mockRepo)

	// Create a test request
	req := httptest.NewRequest("GET", "/article/123", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("X-Forwarded-For", "192.168.1.1")

	// Set up mock expectation
	mockRepo.On("RecordArticleView", mock.AnythingOfType("*models.ArticleView")).Return(nil)

	// Test
	err := service.TrackArticleView(context.Background(), 123, req)

	// Assertions
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAnalyticsService_TrackEngagement(t *testing.T) {
	mockRepo := new(MockAnalyticsRepository)
	service := NewAnalyticsService(mockRepo)

	// Create a test request
	req := httptest.NewRequest("POST", "/engagement", nil)
	req.Header.Set("X-Real-IP", "10.0.0.1")

	// Set up mock expectation
	mockRepo.On("RecordArticleEngagement", mock.AnythingOfType("*models.ArticleEngagement")).Return(nil)

	// Test
	err := service.TrackEngagement(context.Background(), 123, models.ActionLike, req)

	// Assertions
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAnalyticsService_TrackUserBehavior(t *testing.T) {
	mockRepo := new(MockAnalyticsRepository)
	service := NewAnalyticsService(mockRepo)

	// Create a test request
	req := httptest.NewRequest("POST", "/behavior", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X)")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	// Set up mock expectation
	mockRepo.On("RecordUserBehavior", mock.AnythingOfType("*models.UserBehavior")).Return(nil)

	// Test
	userID := uint64(456)
	err := service.TrackUserBehavior(context.Background(), "session123", &userID, 
		"/article/test", 120, 85.5, req)

	// Assertions
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAnalyticsService_TrackPerformanceMetric(t *testing.T) {
	mockRepo := new(MockAnalyticsRepository)
	service := NewAnalyticsService(mockRepo)

	// Set up mock expectation
	mockRepo.On("RecordPerformanceMetric", mock.AnythingOfType("*models.PerformanceMetric")).Return(nil)

	// Test
	tags := map[string]interface{}{
		"endpoint": "/api/articles",
		"method":   "GET",
	}
	err := service.TrackPerformanceMetric(context.Background(), models.MetricTypeResponse, 
		"api_response_time", 150.5, "ms", tags)

	// Assertions
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAnalyticsService_GetArticleAnalytics(t *testing.T) {
	mockRepo := new(MockAnalyticsRepository)
	service := NewAnalyticsService(mockRepo)

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

	startDate := time.Now().AddDate(0, 0, -7)
	endDate := time.Now()

	// Set up mock expectation
	mockRepo.On("GetArticleAnalytics", uint64(123), startDate, endDate).Return(expectedAnalytics, nil)

	// Test
	result, err := service.GetArticleAnalytics(context.Background(), 123, startDate, endDate)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, expectedAnalytics, result)
	mockRepo.AssertExpectations(t)
}

func TestAnalyticsService_GetDashboardMetrics(t *testing.T) {
	mockRepo := new(MockAnalyticsRepository)
	service := NewAnalyticsService(mockRepo)

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

	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now()

	// Set up mock expectation
	mockRepo.On("GetDashboardMetrics", startDate, endDate).Return(expectedMetrics, nil)

	// Test
	result, err := service.GetDashboardMetrics(context.Background(), startDate, endDate)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, expectedMetrics, result)
	mockRepo.AssertExpectations(t)
}

func TestAnalyticsService_GenerateReport(t *testing.T) {
	mockRepo := new(MockAnalyticsRepository)
	service := NewAnalyticsService(mockRepo)

	// Test parameters
	params := models.ReportParameters{
		StartDate: time.Now().AddDate(0, 0, -7),
		EndDate:   time.Now(),
		Limit:     10,
	}

	// Expected articles data
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

	// Set up mock expectations
	mockRepo.On("GetTopArticles", params.StartDate, params.EndDate, params.Limit).Return(expectedArticles, nil)
	mockRepo.On("SaveReport", mock.AnythingOfType("*models.AnalyticsReport")).Return(nil)

	// Test
	report, err := service.GenerateReport(context.Background(), models.ReportTypeArticlePerformance, params, 1)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, models.ReportTypeArticlePerformance, report.ReportType)
	assert.Equal(t, uint64(1), report.GeneratedBy)
	assert.NotNil(t, report.ExpiresAt)
	mockRepo.AssertExpectations(t)
}

func TestAnalyticsService_ExportData(t *testing.T) {
	mockRepo := new(MockAnalyticsRepository)
	service := NewAnalyticsService(mockRepo)

	// Test parameters
	params := models.ReportParameters{
		StartDate: time.Now().AddDate(0, 0, -7),
		EndDate:   time.Now(),
		Metrics:   []string{"views", "engagements"},
	}

	req := models.ExportRequest{
		ReportType: models.ReportTypeEngagement,
		Parameters: params,
		Format:     models.ExportFormatJSON,
	}

	// Expected data
	expectedData := map[string]interface{}{
		"total_views": int64(1000),
		"engagements": map[string]int64{
			"like":    50,
			"dislike": 5,
			"share":   25,
		},
	}

	// Set up mock expectation
	mockRepo.On("GetAnalyticsData", params).Return(expectedData, nil)

	// Test
	data, contentType, err := service.ExportData(context.Background(), req)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, "application/json", contentType)
	assert.NotEmpty(t, data)
	mockRepo.AssertExpectations(t)
}

func TestAnalyticsService_BulkTrackViews(t *testing.T) {
	mockRepo := new(MockAnalyticsRepository)
	service := NewAnalyticsService(mockRepo)

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
	mockRepo.On("BulkRecordViews", views).Return(nil)

	// Test
	err := service.BulkTrackViews(context.Background(), views)

	// Assertions
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAnalyticsService_BulkTrackEngagements(t *testing.T) {
	mockRepo := new(MockAnalyticsRepository)
	service := NewAnalyticsService(mockRepo)

	// Test data
	engagements := []models.ArticleEngagement{
		{
			ArticleID: 1,
			Action:    models.ActionLike,
			IPAddress: "192.168.1.1",
			CreatedAt: time.Now(),
		},
		{
			ArticleID: 2,
			Action:    models.ActionShare,
			IPAddress: "192.168.1.2",
			CreatedAt: time.Now(),
		},
	}

	// Set up mock expectation
	mockRepo.On("BulkRecordEngagements", engagements).Return(nil)

	// Test
	err := service.BulkTrackEngagements(context.Background(), engagements)

	// Assertions
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAnalyticsService_GetClientIP(t *testing.T) {
	service := &AnalyticsService{}

	tests := []struct {
		name           string
		headers        map[string]string
		remoteAddr     string
		expectedIP     string
	}{
		{
			name: "X-Forwarded-For header",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.1, 198.51.100.1",
			},
			remoteAddr: "192.168.1.1:12345",
			expectedIP: "203.0.113.1",
		},
		{
			name: "X-Real-IP header",
			headers: map[string]string{
				"X-Real-IP": "203.0.113.2",
			},
			remoteAddr: "192.168.1.1:12345",
			expectedIP: "203.0.113.2",
		},
		{
			name:       "RemoteAddr fallback",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.1:12345",
			expectedIP: "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			ip := service.getClientIP(req)
			assert.Equal(t, tt.expectedIP, ip)
		})
	}
}

func TestAnalyticsService_DetectDevice(t *testing.T) {
	service := &AnalyticsService{}

	tests := []struct {
		userAgent      string
		expectedDevice string
	}{
		{
			userAgent:      "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X)",
			expectedDevice: "mobile",
		},
		{
			userAgent:      "Mozilla/5.0 (iPad; CPU OS 14_0 like Mac OS X)",
			expectedDevice: "tablet",
		},
		{
			userAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			expectedDevice: "desktop",
		},
		{
			userAgent:      "Mozilla/5.0 (Android 10; Mobile; rv:81.0)",
			expectedDevice: "mobile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expectedDevice, func(t *testing.T) {
			device := service.detectDevice(tt.userAgent)
			assert.Equal(t, tt.expectedDevice, device)
		})
	}
}

func TestAnalyticsService_DetectBrowser(t *testing.T) {
	service := &AnalyticsService{}

	tests := []struct {
		userAgent       string
		expectedBrowser string
	}{
		{
			userAgent:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			expectedBrowser: "chrome",
		},
		{
			userAgent:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0",
			expectedBrowser: "firefox",
		},
		{
			userAgent:       "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
			expectedBrowser: "safari",
		},
		{
			userAgent:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 Edg/91.0.864.59",
			expectedBrowser: "edge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expectedBrowser, func(t *testing.T) {
			browser := service.detectBrowser(tt.userAgent)
			assert.Equal(t, tt.expectedBrowser, browser)
		})
	}
}

func TestAnalyticsService_DetectOS(t *testing.T) {
	service := &AnalyticsService{}

	tests := []struct {
		userAgent    string
		expectedOS   string
	}{
		{
			userAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
			expectedOS: "windows",
		},
		{
			userAgent:  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
			expectedOS: "macos",
		},
		{
			userAgent:  "Mozilla/5.0 (X11; Linux x86_64)",
			expectedOS: "linux",
		},
		{
			userAgent:  "Mozilla/5.0 (Android 10; Mobile; rv:81.0)",
			expectedOS: "android",
		},
		{
			userAgent:  "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X)",
			expectedOS: "ios",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expectedOS, func(t *testing.T) {
			os := service.detectOS(tt.userAgent)
			assert.Equal(t, tt.expectedOS, os)
		})
	}
}