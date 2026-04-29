package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"high-performance-news-website/pkg/cache"
)

// MockCacheService for testing CSRF middleware
type MockCacheService struct {
	mock.Mock
}

func (m *MockCacheService) Get(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockCacheService) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	args := m.Called(ctx, key, value, expiration)
	return args.Error(0)
}

func (m *MockCacheService) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockCacheService) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockCacheService) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockCacheService) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestCSRFMiddleware_GenerateToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	mockCache := &MockCacheService{}
	middleware := NewCSRFMiddleware(mockCache)
	
	// Mock cache set operation
	mockCache.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8"), 24*time.Hour).Return(nil)
	
	// Create test context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.RemoteAddr = "127.0.0.1:12345"
	
	token := middleware.GenerateCSRFToken(c)
	
	assert.NotEmpty(t, token)
	assert.Len(t, token, 64) // 32 bytes hex encoded = 64 characters
	mockCache.AssertExpectations(t)
}

func TestCSRFMiddleware_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	mockCache := &MockCacheService{}
	middleware := NewCSRFMiddleware(mockCache)
	
	router := gin.New()
	router.Use(middleware.CSRFProtection())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	
	// Mock valid token in cache
	validToken := "valid-csrf-token"
	mockCache.On("Get", mock.Anything, mock.AnythingOfType("string")).Return([]byte(validToken), nil)
	
	// Create request with valid CSRF token
	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(`{"data": "test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", validToken)
	req.RemoteAddr = "127.0.0.1:12345"
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	mockCache.AssertExpectations(t)
}

func TestCSRFMiddleware_MissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	mockCache := &MockCacheService{}
	middleware := NewCSRFMiddleware(mockCache)
	
	router := gin.New()
	router.Use(middleware.CSRFProtection())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	
	// Create request without CSRF token
	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(`{"data": "test"}`))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "CSRF token missing")
}

func TestCSRFMiddleware_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	mockCache := &MockCacheService{}
	middleware := NewCSRFMiddleware(mockCache)
	
	router := gin.New()
	router.Use(middleware.CSRFProtection())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	
	// Mock different token in cache
	mockCache.On("Get", mock.Anything, mock.AnythingOfType("string")).Return([]byte("different-token"), nil)
	
	// Create request with invalid CSRF token
	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(`{"data": "test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", "invalid-token")
	req.RemoteAddr = "127.0.0.1:12345"
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid CSRF token")
	mockCache.AssertExpectations(t)
}

func TestCSRFMiddleware_ExpiredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	mockCache := &MockCacheService{}
	middleware := NewCSRFMiddleware(mockCache)
	
	router := gin.New()
	router.Use(middleware.CSRFProtection())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	
	// Mock cache miss (expired token)
	mockCache.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(nil, cache.ErrCacheMiss)
	
	// Create request with expired CSRF token
	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(`{"data": "test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", "expired-token")
	req.RemoteAddr = "127.0.0.1:12345"
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid CSRF token")
	mockCache.AssertExpectations(t)
}

func TestCSRFMiddleware_GetRequestsAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	mockCache := &MockCacheService{}
	middleware := NewCSRFMiddleware(mockCache)
	
	router := gin.New()
	router.Use(middleware.CSRFProtection())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	
	// GET requests should not require CSRF token
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	// No cache operations should be called for GET requests
	mockCache.AssertNotCalled(t, "Get")
}

func TestCSRFMiddleware_FormToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	mockCache := &MockCacheService{}
	middleware := NewCSRFMiddleware(mockCache)
	
	router := gin.New()
	router.Use(middleware.CSRFProtection())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	
	// Mock valid token in cache
	validToken := "form-csrf-token"
	mockCache.On("Get", mock.Anything, mock.AnythingOfType("string")).Return([]byte(validToken), nil)
	
	// Create form request with CSRF token in form data
	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString("_csrf_token="+validToken+"&data=test"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.RemoteAddr = "127.0.0.1:12345"
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	mockCache.AssertExpectations(t)
}

func TestCSRFMiddleware_OptionsRequestsAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	mockCache := &MockCacheService{}
	middleware := NewCSRFMiddleware(mockCache)
	
	router := gin.New()
	router.Use(middleware.CSRFProtection())
	router.OPTIONS("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	
	// OPTIONS requests should not require CSRF token
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	// No cache operations should be called for OPTIONS requests
	mockCache.AssertNotCalled(t, "Get")
}