package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
	"high-performance-news-website/pkg/cache"
)

// Mock services for testing
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) GetTotalCount() (int64, error) {
	args := m.Called()
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockUserService) GetNewUsersToday() (int64, error) {
	args := m.Called()
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockUserService) GetNewUsersThisMonth() (int64, error) {
	args := m.Called()
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockUserService) GetCountByRole(role models.UserRole) (int64, error) {
	args := m.Called(role)
	return int64(args.Int(0)), args.Error(1)
}

type MockArticleService struct {
	mock.Mock
}

func (m *MockArticleService) GetTotalCount() (int64, error) {
	args := m.Called()
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockArticleService) GetPublishedTodayCount() (int64, error) {
	args := m.Called()
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockArticleService) GetPendingCount() (int64, error) {
	args := m.Called()
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockArticleService) GetDraftCount() (int64, error) {
	args := m.Called()
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockArticleService) GetPublishedCount() (int64, error) {
	args := m.Called()
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockArticleService) HealthCheck() error {
	args := m.Called()
	return args.Error(0)
}

type MockSearchService struct {
	mock.Mock
}

func (m *MockSearchService) HealthCheck() error {
	args := m.Called()
	return args.Error(0)
}

type MockCacheService struct {
	mock.Mock
}

func (m *MockCacheService) Get(key string) ([]byte, error) {
	args := m.Called(key)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockCacheService) Set(key string, value []byte, ttl time.Duration) error {
	args := m.Called(key, value, ttl)
	return args.Error(0)
}

func (m *MockCacheService) Delete(key string) error {
	args := m.Called(key)
	return args.Error(0)
}

func (m *MockCacheService) DeletePattern(pattern string) error {
	args := m.Called(pattern)
	return args.Error(0)
}

func (m *MockCacheService) Exists(key string) bool {
	args := m.Called(key)
	return args.Bool(0)
}

type MockConfigService struct {
	mock.Mock
}

func (m *MockConfigService) Get(key string) (string, error) {
	args := m.Called(key)
	return args.String(0), args.Error(1)
}

func (m *MockConfigService) Set(key string, value interface{}) error {
	args := m.Called(key, value)
	return args.Error(0)
}

type MockMetricsService struct {
	mock.Mock
}

func setupAdminHandlersTest() (*AdminHandlers, *MockUserService, *MockArticleService, *MockSearchService, *MockCacheService, *MockConfigService, *MockMetricsService) {
	userService := &MockUserService{}
	articleService := &MockArticleService{}
	searchService := &MockSearchService{}
	cacheService := &MockCacheService{}
	configService := &MockConfigService{}
	metricsService := &MockMetricsService{}

	adminHandlers := NewAdminHandlers(
		userService,
		articleService,
		searchService,
		cacheService,
		configService,
		metricsService,
	)

	return adminHandlers, userService, articleService, searchService, cacheService, configService, metricsService
}

func TestNewAdminHandlers(t *testing.T) {
	adminHandlers, _, _, _, _, _, _ := setupAdminHandlersTest()
	
	assert.NotNil(t, adminHandlers)
	assert.NotNil(t, adminHandlers.userService)
	assert.NotNil(t, adminHandlers.articleService)
	assert.NotNil(t, adminHandlers.searchService)
	assert.NotNil(t, adminHandlers.cacheService)
	assert.NotNil(t, adminHandlers.configService)
	assert.NotNil(t, adminHandlers.metricsService)
}

func TestGetDashboard(t *testing.T) {
	adminHandlers, userService, articleService, _, _, _, _ := setupAdminHandlersTest()

	// Setup mocks
	articleService.On("GetTotalCount").Return(1000, nil)
	articleService.On("GetPublishedTodayCount").Return(25, nil)
	articleService.On("GetPendingCount").Return(5, nil)
	articleService.On("GetDraftCount").Return(15, nil)
	userService.On("GetTotalCount").Return(150, nil)
	userService.On("GetNewUsersToday").Return(3, nil)

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/admin/dashboard", adminHandlers.GetDashboard)

	// Create request
	req, _ := http.NewRequest("GET", "/admin/dashboard", nil)
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Dashboard data retrieved successfully", response.Message)
	assert.NotNil(t, response.Data)

	// Verify mock calls
	articleService.AssertExpectations(t)
	userService.AssertExpectations(t)
}

func TestGetDashboardMetrics(t *testing.T) {
	adminHandlers, userService, articleService, _, _, _, _ := setupAdminHandlersTest()

	// Setup mocks
	articleService.On("GetTotalCount").Return(1000, nil)
	articleService.On("GetPublishedTodayCount").Return(25, nil)
	articleService.On("GetPendingCount").Return(5, nil)
	articleService.On("GetDraftCount").Return(15, nil)
	userService.On("GetTotalCount").Return(150, nil)

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/admin/dashboard/metrics", adminHandlers.GetDashboardMetrics)

	// Create request
	req, _ := http.NewRequest("GET", "/admin/dashboard/metrics", nil)
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Dashboard metrics retrieved successfully", response.Message)

	// Check that response contains expected metrics
	data := response.Data.(map[string]interface{})
	assert.Contains(t, data, "timestamp")
	assert.Contains(t, data, "articles")
	assert.Contains(t, data, "users")
	assert.Contains(t, data, "system")
	assert.Contains(t, data, "traffic")

	// Verify mock calls
	articleService.AssertExpectations(t)
	userService.AssertExpectations(t)
}

func TestGetSystemHealth(t *testing.T) {
	adminHandlers, _, articleService, searchService, cacheService, _, _ := setupAdminHandlersTest()

	// Setup mocks
	articleService.On("HealthCheck").Return(nil)
	searchService.On("HealthCheck").Return(nil)
	cacheService.On("Exists", "health_check").Return(true)

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/admin/system/health", adminHandlers.GetSystemHealth)

	// Create request
	req, _ := http.NewRequest("GET", "/admin/system/health", nil)
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "System health retrieved successfully", response.Message)

	// Check that response contains health data
	data := response.Data.(map[string]interface{})
	assert.Contains(t, data, "status")
	assert.Contains(t, data, "services")
	assert.Contains(t, data, "metrics")
	assert.Equal(t, "healthy", data["status"])

	// Verify mock calls
	articleService.AssertExpectations(t)
	searchService.AssertExpectations(t)
	cacheService.AssertExpectations(t)
}

func TestClearSystemCache(t *testing.T) {
	adminHandlers, _, _, _, cacheService, _, _ := setupAdminHandlersTest()

	// Setup mocks for clearing all caches
	patterns := []string{"article:*", "homepage:*", "category:*", "tag:*", "user:*", "search:*"}
	for _, pattern := range patterns {
		cacheService.On("DeletePattern", pattern).Return(nil)
	}

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/admin/system/cache/clear", adminHandlers.ClearSystemCache)

	// Create request
	req, _ := http.NewRequest("POST", "/admin/system/cache/clear?type=all", nil)
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Cache cleared successfully", response.Message)

	// Check response data
	data := response.Data.(map[string]interface{})
	assert.Equal(t, "all", data["cache_type"])
	assert.Contains(t, data, "cleared_at")

	// Verify mock calls
	cacheService.AssertExpectations(t)
}

func TestClearSystemCacheSpecificType(t *testing.T) {
	adminHandlers, _, _, _, cacheService, _, _ := setupAdminHandlersTest()

	// Setup mock for clearing article cache only
	cacheService.On("DeletePattern", "article:*").Return(nil)

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/admin/system/cache/clear", adminHandlers.ClearSystemCache)

	// Create request
	req, _ := http.NewRequest("POST", "/admin/system/cache/clear?type=articles", nil)
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Cache cleared successfully", response.Message)

	// Check response data
	data := response.Data.(map[string]interface{})
	assert.Equal(t, "articles", data["cache_type"])

	// Verify mock calls
	cacheService.AssertExpectations(t)
}

func TestGetConfiguration(t *testing.T) {
	adminHandlers, _, _, _, _, configService, _ := setupAdminHandlersTest()

	// Setup mocks for configuration values
	configService.On("Get", "site_name").Return("Test News Site", nil)
	configService.On("Get", "site_description").Return("Test Description", nil)
	configService.On("Get", "site_url").Return("https://test.com", nil)
	configService.On("Get", "site_logo").Return("", nil)
	configService.On("Get", "site_favicon").Return("", nil)
	configService.On("Get", "cache_ttl").Return("3600", nil)
	configService.On("Get", "static_generation").Return("true", nil)
	configService.On("Get", "compression_enabled").Return("true", nil)
	configService.On("Get", "comments_enabled").Return("true", nil)
	configService.On("Get", "registration_enabled").Return("false", nil)
	configService.On("Get", "search_enabled").Return("true", nil)
	configService.On("Get", "analytics_enabled").Return("false", nil)
	configService.On("Get", "social_sharing").Return("true", nil)
	configService.On("Get", "newsletter_enabled").Return("false", nil)

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/admin/config", adminHandlers.GetConfiguration)

	// Create request
	req, _ := http.NewRequest("GET", "/admin/config", nil)
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Configuration retrieved successfully", response.Message)

	// Check configuration structure
	data := response.Data.(map[string]interface{})
	assert.Contains(t, data, "site")
	assert.Contains(t, data, "performance")
	assert.Contains(t, data, "features")
	assert.Contains(t, data, "integrations")

	// Verify mock calls
	configService.AssertExpectations(t)
}

func TestUpdateConfiguration(t *testing.T) {
	adminHandlers, _, _, _, _, configService, _ := setupAdminHandlersTest()

	// Setup mocks for configuration updates
	configService.On("Set", "site_name", "Updated Site Name").Return(nil)
	configService.On("Set", "cache_ttl", "7200").Return(nil)

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.PUT("/admin/config", adminHandlers.UpdateConfiguration)

	// Create request body
	configUpdate := map[string]interface{}{
		"site_name": "Updated Site Name",
		"cache_ttl": "7200",
	}
	jsonBody, _ := json.Marshal(configUpdate)

	// Create request
	req, _ := http.NewRequest("PUT", "/admin/config", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Configuration updated successfully", response.Message)

	// Check response data
	data := response.Data.(map[string]interface{})
	assert.Contains(t, data, "updated_at")
	assert.Equal(t, float64(2), data["keys_updated"]) // JSON numbers are float64

	// Verify mock calls
	configService.AssertExpectations(t)
}

func TestGetAnalyticsOverview(t *testing.T) {
	adminHandlers, _, _, _, _, _, _ := setupAdminHandlersTest()

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/admin/analytics/overview", adminHandlers.GetAnalyticsOverview)

	// Create request
	req, _ := http.NewRequest("GET", "/admin/analytics/overview?range=7d", nil)
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Analytics overview retrieved successfully", response.Message)

	// Check analytics structure
	data := response.Data.(map[string]interface{})
	assert.Contains(t, data, "time_range")
	assert.Contains(t, data, "traffic")
	assert.Contains(t, data, "content")
	assert.Contains(t, data, "users")
	assert.Equal(t, "7d", data["time_range"])
}

func TestRegisterAdminRoutes(t *testing.T) {
	adminHandlers, _, _, _, _, _, _ := setupAdminHandlersTest()

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	adminGroup := router.Group("/admin")

	// Register routes
	adminHandlers.RegisterAdminRoutes(adminGroup)

	// Test that routes are registered by checking if they exist
	routes := router.Routes()
	
	// Check for some key routes
	routePaths := make(map[string]bool)
	for _, route := range routes {
		routePaths[route.Path] = true
	}

	expectedRoutes := []string{
		"/admin/dashboard",
		"/admin/dashboard/metrics",
		"/admin/system/health",
		"/admin/config",
		"/admin/analytics/overview",
	}

	for _, expectedRoute := range expectedRoutes {
		assert.True(t, routePaths[expectedRoute], "Route %s should be registered", expectedRoute)
	}
}

func TestInvalidCacheType(t *testing.T) {
	adminHandlers, _, _, _, _, _, _ := setupAdminHandlersTest()

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/admin/system/cache/clear", adminHandlers.ClearSystemCache)

	// Create request with invalid cache type
	req, _ := http.NewRequest("POST", "/admin/system/cache/clear?type=invalid", nil)
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid cache type", response.Error)
	assert.Equal(t, "INVALID_CACHE_TYPE", response.Code)
}

func TestGetSystemMetrics(t *testing.T) {
	adminHandlers, _, _, _, _, _, _ := setupAdminHandlersTest()

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/admin/system/metrics", adminHandlers.GetSystemMetrics)

	// Create request
	req, _ := http.NewRequest("GET", "/admin/system/metrics", nil)
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "System metrics retrieved successfully", response.Message)

	// Check metrics structure
	data := response.Data.(map[string]interface{})
	assert.Contains(t, data, "timestamp")
	assert.Contains(t, data, "performance")
	assert.Contains(t, data, "resources")
	assert.Contains(t, data, "database")
}

func TestExportAnalytics(t *testing.T) {
	adminHandlers, _, _, _, _, _, _ := setupAdminHandlersTest()

	// Setup Gin
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/admin/analytics/export", adminHandlers.ExportAnalytics)

	// Create request
	req, _ := http.NewRequest("GET", "/admin/analytics/export?format=json&type=overview&range=7d", nil)
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Analytics data exported successfully", response.Message)
	assert.NotNil(t, response.Data)
}