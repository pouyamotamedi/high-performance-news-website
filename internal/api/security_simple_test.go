package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSecurityHeadersBasic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(EnhancedSecurityHeaders())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check key security headers
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	assert.Contains(t, w.Header().Get("Strict-Transport-Security"), "max-age=31536000")
	
	// Check CSP header exists
	csp := w.Header().Get("Content-Security-Policy")
	assert.NotEmpty(t, csp)
	assert.Contains(t, csp, "default-src 'self'")
}

func TestInputValidationBasic(t *testing.T) {
	middleware := NewInputValidationMiddleware()

	// Test suspicious content detection
	assert.True(t, middleware.containsSuspiciousContent("<script>alert('xss')</script>"))
	assert.True(t, middleware.containsSuspiciousContent("javascript:alert('xss')"))
	assert.True(t, middleware.containsSuspiciousContent("'; DROP TABLE users; --"))
	assert.False(t, middleware.containsSuspiciousContent("hello world"))
	assert.False(t, middleware.containsSuspiciousContent("https://example.com"))
}