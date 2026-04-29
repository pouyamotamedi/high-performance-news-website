package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestEnhancedSecurityHeaders(t *testing.T) {
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

	// Check security headers
	expectedHeaders := map[string]string{
		"X-Content-Type-Options":           "nosniff",
		"X-Frame-Options":                  "DENY",
		"X-XSS-Protection":                 "1; mode=block",
		"Referrer-Policy":                  "strict-origin-when-cross-origin",
		"Strict-Transport-Security":        "max-age=31536000; includeSubDomains; preload",
		"X-Permitted-Cross-Domain-Policies": "none",
		"X-Download-Options":               "noopen",
		"X-DNS-Prefetch-Control":           "off",
		"Expect-CT":                        "max-age=86400, enforce",
		"Server":                           "",
	}

	for header, expectedValue := range expectedHeaders {
		assert.Equal(t, expectedValue, w.Header().Get(header), "Header %s should be %s", header, expectedValue)
	}

	// Check CSP header exists and contains expected directives
	csp := w.Header().Get("Content-Security-Policy")
	assert.NotEmpty(t, csp)
	assert.Contains(t, csp, "default-src 'self'")
	assert.Contains(t, csp, "object-src 'none'")
	assert.Contains(t, csp, "frame-ancestors 'none'")
}

func TestInputValidationSuspiciousContent(t *testing.T) {
	middleware := NewInputValidationMiddleware()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Clean input", "hello world", false},
		{"XSS script tag", "<script>alert('xss')</script>", true},
		{"JavaScript protocol", "javascript:alert('xss')", true},
		{"SQL injection", "'; DROP TABLE users; --", true},
		{"Path traversal", "../../../etc/passwd", true},
		{"Normal URL", "https://example.com", false},
		{"Eval function", "eval('malicious code')", true},
		{"Document cookie", "document.cookie", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := middleware.containsSuspiciousContent(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}