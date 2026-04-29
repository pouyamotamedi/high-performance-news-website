package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/pkg/cache"
)

// TestSecurityIntegration tests the complete security middleware integration
func TestSecurityIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup mock cache
	mockCache := &MockCacheService{}
	
	// Create all security middleware
	rateLimiter := NewRateLimitMiddleware(mockCache)
	perEndpointRateLimiter := NewPerEndpointRateLimitMiddleware(mockCache)
	csrfMiddleware := NewCSRFMiddleware(mockCache)
	twoFactorMiddleware := NewTwoFactorMiddleware(mockCache)
	inputValidationMiddleware := NewInputValidationMiddleware()

	router := gin.New()
	
	// Apply security middleware
	router.Use(EnhancedSecurityHeaders())
	router.Use(inputValidationMiddleware.ValidateInput())
	router.Use(rateLimiter.UserRateLimit())
	router.Use(perEndpointRateLimiter.EndpointRateLimit())

	// Test routes
	router.POST("/api/v1/articles", csrfMiddleware.CSRFProtection(), func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"message": "article created"})
	})

	router.POST("/api/v1/admin/system", twoFactorMiddleware.Require2FA(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "admin operation"})
	})

	t.Run("Security Headers Applied", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		w := httptest.NewRecorder()
		
		router.GET("/api/v1/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "test"})
		})
		
		router.ServeHTTP(w, req)

		// Check security headers
		assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
		assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
		assert.Contains(t, w.Header().Get("Content-Security-Policy"), "default-src 'self'")
		assert.Contains(t, w.Header().Get("Strict-Transport-Security"), "max-age=31536000")
	})

	t.Run("Rate Limiting Applied", func(t *testing.T) {
		// Mock rate limiting
		mockCache.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(nil, cache.ErrCacheMiss).Times(1)
		mockCache.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8"), mock.AnythingOfType("time.Duration")).Return(nil).Times(1)

		req := httptest.NewRequest("POST", "/api/v1/articles", bytes.NewBufferString(`{"title": "test"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Should have rate limit headers
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Limit"))
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Remaining"))
	})

	t.Run("CSRF Protection Applied", func(t *testing.T) {
		// Request without CSRF token should be rejected
		req := httptest.NewRequest("POST", "/api/v1/articles", bytes.NewBufferString(`{"title": "test"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "CSRF token missing")
	})

	t.Run("Input Validation Applied", func(t *testing.T) {
		// Request with suspicious content should be rejected
		req := httptest.NewRequest("GET", "/api/v1/test?q=<script>alert('xss')</script>", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Suspicious content detected")
	})

	t.Run("2FA Protection Applied", func(t *testing.T) {
		// Mock user context
		req := httptest.NewRequest("POST", "/api/v1/admin/system", bytes.NewBufferString(`{}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req

		// Set admin user in context
		adminUser := &models.User{
			ID:   1,
			Role: models.RoleAdmin,
		}
		c.Set("user", adminUser)

		// Mock 2FA not verified
		mockCache.On("Get", mock.Anything, mock.AnythingOfType("string")).Return(nil, cache.ErrCacheMiss)

		// Should require 2FA
		twoFactorMiddleware.Require2FA()(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "2FA verification required")
	})

	mockCache.AssertExpectations(t)
}

// TestTOTPImplementation tests the TOTP implementation
func TestTOTPImplementation(t *testing.T) {
	mockCache := &MockCacheService{}
	middleware := NewTwoFactorMiddleware(mockCache)

	t.Run("TOTP Code Generation", func(t *testing.T) {
		userID := uint64(123)
		
		// Generate code
		code1 := middleware.generateTimeBasedCode(userID)
		assert.Len(t, code1, 6)
		assert.Regexp(t, `^\d{6}$`, code1)

		// Code should be consistent within time window
		code2 := middleware.generateTimeBasedCode(userID)
		assert.Equal(t, code1, code2)
	})

	t.Run("TOTP Validation", func(t *testing.T) {
		userID := uint64(123)
		
		// Generate valid code
		validCode := middleware.generateTimeBasedCode(userID)
		
		// Should validate correctly
		assert.True(t, middleware.ValidateTOTP(userID, validCode))
		
		// Invalid code should fail
		assert.False(t, middleware.ValidateTOTP(userID, "000000"))
		assert.False(t, middleware.ValidateTOTP(userID, "invalid"))
	})

	t.Run("QR Code URL Generation", func(t *testing.T) {
		userID := uint64(123)
		email := "admin@example.com"
		issuer := "Test App"
		
		qrURL := middleware.GenerateQRCodeURL(userID, email, issuer)
		
		assert.Contains(t, qrURL, "otpauth://totp/")
		assert.Contains(t, qrURL, email)
		assert.Contains(t, qrURL, issuer)
		assert.Contains(t, qrURL, "secret=")
		assert.Contains(t, qrURL, "algorithm=SHA1")
		assert.Contains(t, qrURL, "digits=6")
		assert.Contains(t, qrURL, "period=30")
	})
}

// TestAPIKeyManagement tests API key management functionality
func TestAPIKeyManagement(t *testing.T) {
	mockCache := &MockCacheService{}
	mockUserService := &MockUserService{}
	middleware := NewAPIKeyMiddleware(mockCache, mockUserService)

	t.Run("Generate API Key", func(t *testing.T) {
		userID := uint64(123)
		expiresIn := 24 * time.Hour

		// Mock cache operations
		mockCache.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8"), expiresIn).Return(nil).Times(2)

		apiKey, err := middleware.GenerateAPIKey(userID, expiresIn)
		
		assert.NoError(t, err)
		assert.Len(t, apiKey, 64) // 32 bytes hex encoded
		assert.Regexp(t, `^[a-f0-9]{64}$`, apiKey)
	})

	t.Run("Rotate API Key", func(t *testing.T) {
		userID := uint64(123)
		expiresIn := 24 * time.Hour
		oldKey := "old-api-key"

		// Mock getting old key
		mockCache.On("Get", mock.Anything, fmt.Sprintf("user_api_key:%d", userID)).Return([]byte(oldKey), nil)
		// Mock deleting old key
		mockCache.On("Delete", mock.Anything, fmt.Sprintf("api_key:%s", oldKey)).Return(nil)
		// Mock setting new key
		mockCache.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8"), expiresIn).Return(nil).Times(2)

		newKey, err := middleware.RotateAPIKey(userID, expiresIn)
		
		assert.NoError(t, err)
		assert.NotEqual(t, oldKey, newKey)
		assert.Len(t, newKey, 64)
	})

	t.Run("Validate API Key", func(t *testing.T) {
		userID := uint64(123)
		apiKey := "valid-api-key"
		
		user := &models.User{
			ID:   userID,
			Role: models.RoleAdmin,
		}

		// Mock cache get
		mockCache.On("Get", mock.Anything, fmt.Sprintf("api_key:%s", apiKey)).Return([]byte("123"), nil)
		// Mock user service
		mockUserService.On("GetByID", userID, (*models.User)(nil)).Return(user, nil)

		validatedUser, err := middleware.validateAPIKey(apiKey)
		
		assert.NoError(t, err)
		assert.Equal(t, userID, validatedUser.ID)
	})

	mockCache.AssertExpectations(t)
	mockUserService.AssertExpectations(t)
}

// TestPerEndpointRateLimiting tests per-endpoint rate limiting
func TestPerEndpointRateLimiting(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	mockCache := &MockCacheService{}
	middleware := NewPerEndpointRateLimitMiddleware(mockCache)

	router := gin.New()
	router.Use(middleware.EndpointRateLimit())
	
	// Different endpoints with different limits
	router.POST("/api/v1/auth/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "login"})
	})
	
	router.POST("/api/v1/articles", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"message": "article created"})
	})

	t.Run("Different Limits for Different Endpoints", func(t *testing.T) {
		// Mock cache for login endpoint (stricter limit: 5/min)
		mockCache.On("Get", mock.Anything, mock.MatchedBy(func(key string) bool {
			return key == "endpoint_rate_limit:127.0.0.1:POST:/api/v1/auth/login"
		})).Return(nil, cache.ErrCacheMiss)
		mockCache.On("Set", mock.Anything, mock.MatchedBy(func(key string) bool {
			return key == "endpoint_rate_limit:127.0.0.1:POST:/api/v1/auth/login"
		}), mock.AnythingOfType("[]uint8"), mock.AnythingOfType("time.Duration")).Return(nil)

		// Mock cache for articles endpoint (higher limit: 50/min)
		mockCache.On("Get", mock.Anything, mock.MatchedBy(func(key string) bool {
			return key == "endpoint_rate_limit:127.0.0.1:POST:/api/v1/articles"
		})).Return(nil, cache.ErrCacheMiss)
		mockCache.On("Set", mock.Anything, mock.MatchedBy(func(key string) bool {
			return key == "endpoint_rate_limit:127.0.0.1:POST:/api/v1/articles"
		}), mock.AnythingOfType("[]uint8"), mock.AnythingOfType("time.Duration")).Return(nil)

		// Test login endpoint
		req1 := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBufferString(`{}`))
		req1.Header.Set("Content-Type", "application/json")
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)

		assert.Equal(t, http.StatusOK, w1.Code)
		assert.Equal(t, "5", w1.Header().Get("X-RateLimit-Limit")) // Login limit

		// Test articles endpoint
		req2 := httptest.NewRequest("POST", "/api/v1/articles", bytes.NewBufferString(`{}`))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusCreated, w2.Code)
		assert.Equal(t, "50", w2.Header().Get("X-RateLimit-Limit")) // Articles limit
	})

	mockCache.AssertExpectations(t)
}