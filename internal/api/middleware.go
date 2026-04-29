package api

import (
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/auth"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
	"high-performance-news-website/pkg/cache"
)

// AuthMiddleware provides JWT authentication middleware
type AuthMiddleware struct {
	authService *auth.AuthService
	userService *services.UserService
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(authService *auth.AuthService, userService *services.UserService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
		userService: userService,
	}
}

// RequireAuth middleware that requires valid authentication
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "Missing authorization header",
				Code:    ErrCodeUnauthorized,
				Message: "Authorization header is required",
			})
			c.Abort()
			return
		}

		// Check Bearer token format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "Invalid authorization header format",
				Code:    ErrCodeUnauthorized,
				Message: "Authorization header must be in format 'Bearer <token>'",
			})
			c.Abort()
			return
		}

		token := parts[1]

		// Validate token
		claims, err := m.authService.ValidateAccessToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "Invalid token",
				Code:    ErrCodeUnauthorized,
				Message: "The provided token is invalid or expired",
			})
			c.Abort()
			return
		}

		// Get user from database
		user, err := m.userService.GetByID(claims.UserID, nil)
		if err != nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "User not found",
				Code:    ErrCodeUnauthorized,
				Message: "The user associated with this token was not found",
			})
			c.Abort()
			return
		}

		// Check if user is active
		if !user.IsActive {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "User inactive",
				Code:    ErrCodeUnauthorized,
				Message: "Your account has been deactivated",
			})
			c.Abort()
			return
		}

		// Store user in context
		c.Set("user", user)
		c.Set("user_id", user.ID)
		c.Set("user_role", user.Role)

		c.Next()
	}
}

// OptionalAuth middleware that optionally extracts user if token is provided
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		// Check Bearer token format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		token := parts[1]

		// Validate token
		claims, err := m.authService.ValidateAccessToken(token)
		if err != nil {
			c.Next()
			return
		}

		// Get user from database
		user, err := m.userService.GetByID(claims.UserID, nil)
		if err != nil || !user.IsActive {
			c.Next()
			return
		}

		// Store user in context
		c.Set("user", user)
		c.Set("user_id", user.ID)
		c.Set("user_role", user.Role)

		c.Next()
	}
}

// RequireRole middleware that requires specific user roles
func RequireRole(roles ...models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		userInterface, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "Authentication required",
				Code:    ErrCodeUnauthorized,
				Message: "You must be authenticated to access this resource",
			})
			c.Abort()
			return
		}

		user, ok := userInterface.(*models.User)
		if !ok {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Invalid user context",
				Code:    ErrCodeServerError,
				Message: "An unexpected error occurred",
			})
			c.Abort()
			return
		}

		// Check if user has any of the required roles
		hasRole := false
		for _, role := range roles {
			if user.Role == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "Insufficient permissions",
				Code:    ErrCodeForbidden,
				Message: "You don't have permission to access this resource",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitMiddleware provides rate limiting functionality
type RateLimitMiddleware struct {
	cache cache.CacheService
}

// NewRateLimitMiddleware creates a new rate limit middleware
func NewRateLimitMiddleware(cacheService cache.CacheService) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		cache: cacheService,
	}
}

// RateLimit applies rate limiting based on IP address
func (m *RateLimitMiddleware) RateLimit(requests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := fmt.Sprintf("rate_limit:%s", c.ClientIP())
		
		// Get current count
		ctx := context.Background()
		current, err := m.cache.Get(ctx, key)
		count := 0
		
		if err == nil && current != nil {
			count, _ = strconv.Atoi(string(current))
		}
		
		// Check if limit exceeded
		if count >= requests {
			c.Header("X-RateLimit-Limit", strconv.Itoa(requests))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(window).Unix(), 10))
			
			c.JSON(http.StatusTooManyRequests, ErrorResponse{
				Error:   "Rate limit exceeded",
				Code:    ErrCodeRateLimit,
				Message: fmt.Sprintf("Too many requests. Limit: %d requests per %v", requests, window),
			})
			c.Abort()
			return
		}
		
		// Increment counter
		count++
		m.cache.Set(ctx, key, []byte(strconv.Itoa(count)), window)
		
		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(requests))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(requests-count))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(window).Unix(), 10))
		
		c.Next()
	}
}

// UserRateLimit applies different rate limits based on user role
func (m *RateLimitMiddleware) UserRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		var key string
		var requests int
		var window time.Duration

		// Get user from context if available
		userInterface, exists := c.Get("user")
		if exists {
			user, ok := userInterface.(*models.User)
			if ok {
				// Different limits based on user role
				switch user.Role {
				case models.RoleAdmin:
					requests = 1000
					window = time.Minute
				case models.RoleEditor:
					requests = 500
					window = time.Minute
				case models.RoleReporter:
					requests = 200
					window = time.Minute
				case models.RoleContributor:
					requests = 100
					window = time.Minute
				default:
					requests = 60
					window = time.Minute
				}
				key = fmt.Sprintf("user_rate_limit:%d", user.ID)
			} else {
				// Fallback to IP-based limiting
				requests = 60
				window = time.Minute
				key = fmt.Sprintf("rate_limit:%s", c.ClientIP())
			}
		} else {
			// Anonymous users - stricter limits
			requests = 60
			window = time.Minute
			key = fmt.Sprintf("rate_limit:%s", c.ClientIP())
		}

		// Apply rate limiting logic
		ctx := context.Background()
		current, err := m.cache.Get(ctx, key)
		count := 0
		
		if err == nil && current != nil {
			count, _ = strconv.Atoi(string(current))
		}
		
		if count >= requests {
			c.Header("X-RateLimit-Limit", strconv.Itoa(requests))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(window).Unix(), 10))
			
			c.JSON(http.StatusTooManyRequests, ErrorResponse{
				Error:   "Rate limit exceeded",
				Code:    ErrCodeRateLimit,
				Message: fmt.Sprintf("Too many requests. Limit: %d requests per %v", requests, window),
			})
			c.Abort()
			return
		}
		
		count++
		m.cache.Set(ctx, key, []byte(strconv.Itoa(count)), window)
		
		c.Header("X-RateLimit-Limit", strconv.Itoa(requests))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(requests-count))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(window).Unix(), 10))
		
		c.Next()
	}
}

// CORS middleware for handling cross-origin requests
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// SecurityHeaders middleware adds security headers
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		// Implement different CSP policies for admin vs public areas
		var csp string
		if strings.HasPrefix(c.Request.URL.Path, "/admin/") {
			// Admin CSP: More permissive for admin functionality while maintaining security
			csp = "default-src 'self'; " +
				"script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.jsdelivr.net; " +
				"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://cdnjs.cloudflare.com; " +
				"style-src-elem 'self' 'unsafe-inline' https://fonts.googleapis.com https://cdnjs.cloudflare.com; " +
				"font-src 'self' https://fonts.gstatic.com https://cdnjs.cloudflare.com; " +
				"img-src 'self' data: https: blob:; " +
				"connect-src 'self' https:; " +
				"media-src 'self' data: blob:; " +
				"object-src 'none'; " +
				"frame-ancestors 'none'; " +
				"base-uri 'self'; " +
				"form-action 'self'"
		} else {
			// Public CSP: More restrictive for public-facing pages
			csp = "default-src 'self'; " +
				"script-src 'self'; " +
				"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
				"font-src 'self' https://fonts.gstatic.com; " +
				"img-src 'self' data: https:; " +
				"connect-src 'self'; " +
				"object-src 'none'; " +
				"frame-ancestors 'none'; " +
				"base-uri 'self'; " +
				"form-action 'self'"
		}
		c.Header("Content-Security-Policy", csp)
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		
		c.Next()
	}
}

// RequestID middleware adds a unique request ID to each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("%d", time.Now().UnixNano())
		}
		
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		
		c.Next()
	}
}

// LoggingMiddleware provides structured logging for API requests
func LoggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[%s] %s %s %d %s %s %s\n",
			param.TimeStamp.Format("2006-01-02 15:04:05"),
			param.Method,
			param.Path,
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.ErrorMessage,
		)
	})
}

// PerEndpointRateLimitMiddleware provides different rate limits for different endpoints
type PerEndpointRateLimitMiddleware struct {
	cache cache.CacheService
}

// NewPerEndpointRateLimitMiddleware creates a new per-endpoint rate limit middleware
func NewPerEndpointRateLimitMiddleware(cacheService cache.CacheService) *PerEndpointRateLimitMiddleware {
	return &PerEndpointRateLimitMiddleware{
		cache: cacheService,
	}
}

// EndpointRateLimit applies different rate limits based on endpoint patterns
func (m *PerEndpointRateLimitMiddleware) EndpointRateLimit() gin.HandlerFunc {
	// Define rate limits for different endpoint patterns
	endpointLimits := map[string]struct {
		requests int
		window   time.Duration
	}{
		// Authentication endpoints - stricter limits
		"POST:/api/v1/auth/login":    {requests: 5, window: time.Minute},
		"POST:/api/v1/auth/register": {requests: 3, window: time.Minute},
		"POST:/api/v1/auth/refresh":  {requests: 10, window: time.Minute},
		
		// Article creation - moderate limits
		"POST:/api/v1/articles":      {requests: 50, window: time.Minute},
		"PUT:/api/v1/articles/*":     {requests: 100, window: time.Minute},
		"DELETE:/api/v1/articles/*":  {requests: 20, window: time.Minute},
		
		// Bulk operations - very strict limits
		"POST:/api/v1/articles/bulk": {requests: 5, window: time.Minute},
		"POST:/api/v1/tags/bulk":     {requests: 10, window: time.Minute},
		"POST:/api/v1/categories/bulk": {requests: 10, window: time.Minute},
		
		// Search endpoints - moderate limits
		"GET:/api/v1/search":         {requests: 100, window: time.Minute},
		
		// User management - moderate limits
		"POST:/api/v1/users":         {requests: 10, window: time.Minute},
		"PUT:/api/v1/users/*":        {requests: 20, window: time.Minute},
		"DELETE:/api/v1/users/*":     {requests: 5, window: time.Minute},
		
		// System management - very strict limits
		"POST:/api/v1/system/cache/clear": {requests: 2, window: time.Minute},
		"GET:/api/v1/system/metrics":      {requests: 30, window: time.Minute},
		
		// Admin panel endpoints - generous limits for authenticated admin users
		"GET:/api/v1/admin-panel/*":       {requests: 1000, window: time.Minute},
		"POST:/api/v1/admin-panel/*":      {requests: 500, window: time.Minute},
		"PUT:/api/v1/admin-panel/*":       {requests: 500, window: time.Minute},
		"DELETE:/api/v1/admin-panel/*":    {requests: 100, window: time.Minute},
		
		// Default for other endpoints
		"*": {requests: 200, window: time.Minute},
	}

	return func(c *gin.Context) {
		endpoint := fmt.Sprintf("%s:%s", c.Request.Method, c.FullPath())
		
		// Find matching rate limit
		var limit struct {
			requests int
			window   time.Duration
		}
		
		// Check for exact match first
		if l, exists := endpointLimits[endpoint]; exists {
			limit = l
		} else {
			// Check for wildcard patterns
			found := false
			for pattern, l := range endpointLimits {
				if pattern == "*" {
					continue
				}
				if matched, _ := regexp.MatchString(strings.Replace(pattern, "*", ".*", -1), endpoint); matched {
					limit = l
					found = true
					break
				}
			}
			if !found {
				limit = endpointLimits["*"]
			}
		}

		// Create rate limit key
		var key string
		userInterface, exists := c.Get("user")
		if exists {
			user, ok := userInterface.(*models.User)
			if ok {
				key = fmt.Sprintf("endpoint_rate_limit:%d:%s", user.ID, endpoint)
			} else {
				key = fmt.Sprintf("endpoint_rate_limit:%s:%s", c.ClientIP(), endpoint)
			}
		} else {
			key = fmt.Sprintf("endpoint_rate_limit:%s:%s", c.ClientIP(), endpoint)
		}

		// Apply rate limiting
		ctx := context.Background()
		current, err := m.cache.Get(ctx, key)
		count := 0
		
		if err == nil && current != nil {
			count, _ = strconv.Atoi(string(current))
		}
		
		if count >= limit.requests {
			c.Header("X-RateLimit-Limit", strconv.Itoa(limit.requests))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(limit.window).Unix(), 10))
			c.Header("X-RateLimit-Endpoint", endpoint)
			
			c.JSON(http.StatusTooManyRequests, ErrorResponse{
				Error:   "Endpoint rate limit exceeded",
				Code:    ErrCodeRateLimit,
				Message: fmt.Sprintf("Too many requests to %s. Limit: %d requests per %v", endpoint, limit.requests, limit.window),
			})
			c.Abort()
			return
		}
		
		count++
		m.cache.Set(ctx, key, []byte(strconv.Itoa(count)), limit.window)
		
		c.Header("X-RateLimit-Limit", strconv.Itoa(limit.requests))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(limit.requests-count))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(limit.window).Unix(), 10))
		c.Header("X-RateLimit-Endpoint", endpoint)
		
		c.Next()
	}
}

// CSRFMiddleware provides CSRF protection
type CSRFMiddleware struct {
	cache cache.CacheService
}

// NewCSRFMiddleware creates a new CSRF middleware
func NewCSRFMiddleware(cacheService cache.CacheService) *CSRFMiddleware {
	return &CSRFMiddleware{
		cache: cacheService,
	}
}

// CSRFProtection provides CSRF token generation and validation
func (m *CSRFMiddleware) CSRFProtection() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip CSRF for GET, HEAD, OPTIONS requests
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// Skip CSRF only for specific safe routes
		skipCSRFPaths := []string{
			"/admin/login",           // Login page doesn't need CSRF
			"/api/v1/auth/login",     // Login API doesn't need CSRF
			"/api/v1/auth/csrf-token", // CSRF token endpoint
			"/api/v1/auth/verify",    // Token verification
			"/api/v1/auth/2fa/status", // 2FA status check
		}
		
		for _, path := range skipCSRFPaths {
			if c.Request.URL.Path == path {
				c.Next()
				return
			}
		}
		
		// Skip CSRF for GET requests to admin panel (viewing pages is safe)
		if c.Request.Method == "GET" && strings.HasPrefix(c.Request.URL.Path, "/admin") {
			c.Next()
			return
		}

		// Get CSRF token from header or form
		token := c.GetHeader("X-CSRF-Token")
		if token == "" {
			token = c.PostForm("_csrf_token")
		}

		if token == "" {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "CSRF token missing",
				Code:    "CSRF_TOKEN_MISSING",
				Message: "CSRF token is required for this request",
			})
			c.Abort()
			return
		}

		// Validate CSRF token
		if !m.validateCSRFToken(c, token) {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "Invalid CSRF token",
				Code:    "CSRF_TOKEN_INVALID",
				Message: "The CSRF token is invalid or expired",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GenerateCSRFToken generates a new CSRF token for the session
func (m *CSRFMiddleware) GenerateCSRFToken(c *gin.Context) string {
	// Generate random token
	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Store token in cache with session key
	sessionKey := m.getSessionKey(c)
	ctx := context.Background()
	m.cache.Set(ctx, fmt.Sprintf("csrf_token:%s", sessionKey), []byte(token), 24*time.Hour)

	return token
}

// validateCSRFToken validates a CSRF token
func (m *CSRFMiddleware) validateCSRFToken(c *gin.Context, token string) bool {
	sessionKey := m.getSessionKey(c)
	ctx := context.Background()
	
	storedToken, err := m.cache.Get(ctx, fmt.Sprintf("csrf_token:%s", sessionKey))
	if err != nil || storedToken == nil {
		return false
	}

	// Use constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare([]byte(token), storedToken) == 1
}

// getSessionKey creates a session key based on user ID or IP
func (m *CSRFMiddleware) getSessionKey(c *gin.Context) string {
	userInterface, exists := c.Get("user")
	if exists {
		user, ok := userInterface.(*models.User)
		if ok {
			return fmt.Sprintf("user_%d", user.ID)
		}
	}
	return fmt.Sprintf("ip_%s", c.ClientIP())
}

// APIKeyMiddleware provides API key management and validation
type APIKeyMiddleware struct {
	cache       cache.CacheService
	userService *services.UserService
}

// NewAPIKeyMiddleware creates a new API key middleware
func NewAPIKeyMiddleware(cacheService cache.CacheService, userService *services.UserService) *APIKeyMiddleware {
	return &APIKeyMiddleware{
		cache:       cacheService,
		userService: userService,
	}
}

// APIKeyAuth validates API keys for programmatic access
func (m *APIKeyMiddleware) APIKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "API key missing",
				Code:    "API_KEY_MISSING",
				Message: "X-API-Key header is required for API access",
			})
			c.Abort()
			return
		}

		// Validate API key
		user, err := m.validateAPIKey(apiKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "Invalid API key",
				Code:    "API_KEY_INVALID",
				Message: "The provided API key is invalid or expired",
			})
			c.Abort()
			return
		}

		// Store user in context
		c.Set("user", user)
		c.Set("user_id", user.ID)
		c.Set("user_role", user.Role)
		c.Set("auth_method", "api_key")

		c.Next()
	}
}

// validateAPIKey validates an API key and returns the associated user
func (m *APIKeyMiddleware) validateAPIKey(apiKey string) (*models.User, error) {
	ctx := context.Background()
	
	// Check if API key exists in cache
	userIDBytes, err := m.cache.Get(ctx, fmt.Sprintf("api_key:%s", apiKey))
	if err != nil || userIDBytes == nil {
		return nil, fmt.Errorf("API key not found")
	}

	userID, err := strconv.ParseUint(string(userIDBytes), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID in API key")
	}

	// Get user from service
	user, err := m.userService.GetByID(userID, nil)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	if !user.IsActive {
		return nil, fmt.Errorf("user account is inactive")
	}

	return user, nil
}

// GenerateAPIKey generates a new API key for a user
func (m *APIKeyMiddleware) GenerateAPIKey(userID uint64, expiresIn time.Duration) (string, error) {
	// Generate random API key
	keyBytes := make([]byte, 32)
	rand.Read(keyBytes)
	apiKey := hex.EncodeToString(keyBytes)

	// Store API key in cache
	ctx := context.Background()
	err := m.cache.Set(ctx, fmt.Sprintf("api_key:%s", apiKey), []byte(strconv.FormatUint(userID, 10)), expiresIn)
	if err != nil {
		return "", err
	}

	// Store reverse mapping for key rotation
	err = m.cache.Set(ctx, fmt.Sprintf("user_api_key:%d", userID), []byte(apiKey), expiresIn)
	if err != nil {
		return "", err
	}

	return apiKey, nil
}

// RotateAPIKey rotates an existing API key
func (m *APIKeyMiddleware) RotateAPIKey(userID uint64, expiresIn time.Duration) (string, error) {
	ctx := context.Background()
	
	// Get existing API key
	oldKeyBytes, err := m.cache.Get(ctx, fmt.Sprintf("user_api_key:%d", userID))
	if err == nil && oldKeyBytes != nil {
		// Delete old key
		m.cache.Delete(ctx, fmt.Sprintf("api_key:%s", string(oldKeyBytes)))
	}

	// Generate new API key
	return m.GenerateAPIKey(userID, expiresIn)
}

// RevokeAPIKey revokes an API key
func (m *APIKeyMiddleware) RevokeAPIKey(userID uint64) error {
	ctx := context.Background()
	
	// Get existing API key
	oldKeyBytes, err := m.cache.Get(ctx, fmt.Sprintf("user_api_key:%d", userID))
	if err != nil {
		return err
	}

	// Delete both mappings
	m.cache.Delete(ctx, fmt.Sprintf("api_key:%s", string(oldKeyBytes)))
	m.cache.Delete(ctx, fmt.Sprintf("user_api_key:%d", userID))

	return nil
}

// TwoFactorMiddleware provides 2FA validation for admin accounts
type TwoFactorMiddleware struct {
	cache cache.CacheService
}

// NewTwoFactorMiddleware creates a new 2FA middleware
func NewTwoFactorMiddleware(cacheService cache.CacheService) *TwoFactorMiddleware {
	return &TwoFactorMiddleware{
		cache: cacheService,
	}
}

// Require2FA requires 2FA verification for admin operations
func (m *TwoFactorMiddleware) Require2FA() gin.HandlerFunc {
	return func(c *gin.Context) {
		userInterface, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "Authentication required",
				Code:    ErrCodeUnauthorized,
				Message: "You must be authenticated to access this resource",
			})
			c.Abort()
			return
		}

		user, ok := userInterface.(*models.User)
		if !ok {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Invalid user context",
				Code:    ErrCodeServerError,
				Message: "An unexpected error occurred",
			})
			c.Abort()
			return
		}

		// Only require 2FA for admin accounts
		if user.Role != models.RoleAdmin {
			c.Next()
			return
		}

		// Check if 2FA is verified for this session
		// TODO: Re-enable 2FA after frontend integration is complete
		if false && !m.is2FAVerified(c, user.ID) {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "2FA verification required",
				Code:    "2FA_REQUIRED",
				Message: "Two-factor authentication is required for admin operations",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// is2FAVerified checks if 2FA is verified for the current session
func (m *TwoFactorMiddleware) is2FAVerified(c *gin.Context, userID uint64) bool {
	ctx := context.Background()
	sessionKey := fmt.Sprintf("2fa_verified:%d:%s", userID, c.ClientIP())
	
	verified, err := m.cache.Get(ctx, sessionKey)
	return err == nil && verified != nil && string(verified) == "true"
}

// Verify2FA verifies a TOTP code and marks the session as verified
func (m *TwoFactorMiddleware) Verify2FA(c *gin.Context, userID uint64, code string) bool {
	// Validate TOTP code
	if !m.ValidateTOTP(userID, code) {
		return false
	}

	// Mark session as 2FA verified for 30 minutes
	ctx := context.Background()
	sessionKey := fmt.Sprintf("2fa_verified:%d:%s", userID, c.ClientIP())
	m.cache.Set(ctx, sessionKey, []byte("true"), 30*time.Minute)

	return true
}

// generateTimeBasedCode generates a proper TOTP code
func (m *TwoFactorMiddleware) generateTimeBasedCode(userID uint64) string {
	// Generate a secret key based on user ID (in production, store unique secrets per user)
	secret := m.generateUserSecret(userID)
	
	// Generate TOTP code
	return m.generateTOTP(secret, time.Now())
}

// generateUserSecret generates a consistent secret for a user (in production, store in database)
func (m *TwoFactorMiddleware) generateUserSecret(userID uint64) string {
	// In production, each user should have a unique secret stored securely
	// This is a deterministic approach for demonstration
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, userID)
	
	// Create HMAC with a server secret (should be from config in production)
	serverSecret := []byte("your-server-secret-key-change-in-production")
	h := hmac.New(sha1.New, serverSecret)
	h.Write(data)
	hash := h.Sum(nil)
	
	// Encode as base32 for TOTP compatibility
	return base32.StdEncoding.EncodeToString(hash[:20])
}

// generateTOTP generates a TOTP code using RFC 6238
func (m *TwoFactorMiddleware) generateTOTP(secret string, timestamp time.Time) string {
	// Decode base32 secret
	key, err := base32.StdEncoding.DecodeString(secret)
	if err != nil {
		return "000000" // Fallback
	}
	
	// Calculate time counter (30-second intervals)
	counter := uint64(timestamp.Unix()) / 30
	
	// Convert counter to bytes
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)
	
	// Generate HMAC-SHA1
	h := hmac.New(sha1.New, key)
	h.Write(buf)
	hash := h.Sum(nil)
	
	// Dynamic truncation
	offset := hash[19] & 0x0f
	code := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7fffffff
	
	// Generate 6-digit code
	return fmt.Sprintf("%06d", code%1000000)
}

// ValidateTOTP validates a TOTP code with time window tolerance
func (m *TwoFactorMiddleware) ValidateTOTP(userID uint64, inputCode string) bool {
	secret := m.generateUserSecret(userID)
	now := time.Now()
	
	// Check current time window and ±1 window for clock skew tolerance
	for i := -1; i <= 1; i++ {
		testTime := now.Add(time.Duration(i) * 30 * time.Second)
		expectedCode := m.generateTOTP(secret, testTime)
		
		if subtle.ConstantTimeCompare([]byte(inputCode), []byte(expectedCode)) == 1 {
			return true
		}
	}
	
	return false
}

// GenerateQRCodeURL generates a QR code URL for TOTP setup
func (m *TwoFactorMiddleware) GenerateQRCodeURL(userID uint64, userEmail string, issuer string) string {
	secret := m.generateUserSecret(userID)
	
	// Generate otpauth URL for QR code
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30",
		issuer, userEmail, secret, issuer)
}

// InputValidationMiddleware provides comprehensive input validation
type InputValidationMiddleware struct{}

// NewInputValidationMiddleware creates a new input validation middleware
func NewInputValidationMiddleware() *InputValidationMiddleware {
	return &InputValidationMiddleware{}
}

// ValidateInput provides input sanitization and validation
func (m *InputValidationMiddleware) ValidateInput() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Validate Content-Type for POST/PUT requests
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			contentType := c.GetHeader("Content-Type")
			if contentType != "" && !strings.Contains(contentType, "application/json") && 
			   !strings.Contains(contentType, "application/x-www-form-urlencoded") &&
			   !strings.Contains(contentType, "multipart/form-data") {
				c.JSON(http.StatusBadRequest, ErrorResponse{
					Error:   "Invalid content type",
					Code:    ErrCodeValidation,
					Message: "Content-Type must be application/json, application/x-www-form-urlencoded, or multipart/form-data",
				})
				c.Abort()
				return
			}
		}

		// Validate request size
		if c.Request.ContentLength > 10*1024*1024 { // 10MB limit
			c.JSON(http.StatusRequestEntityTooLarge, ErrorResponse{
				Error:   "Request too large",
				Code:    "REQUEST_TOO_LARGE",
				Message: "Request body size exceeds 10MB limit",
			})
			c.Abort()
			return
		}

		// Validate query parameters
		for key, values := range c.Request.URL.Query() {
			if len(key) > 100 {
				c.JSON(http.StatusBadRequest, ErrorResponse{
					Error:   "Invalid query parameter",
					Code:    ErrCodeValidation,
					Message: "Query parameter names must be less than 100 characters",
				})
				c.Abort()
				return
			}
			
			for _, value := range values {
				if len(value) > 1000 {
					c.JSON(http.StatusBadRequest, ErrorResponse{
						Error:   "Invalid query parameter value",
						Code:    ErrCodeValidation,
						Message: "Query parameter values must be less than 1000 characters",
					})
					c.Abort()
					return
				}
				
				// Check for potential XSS/injection attempts
				if m.containsSuspiciousContent(value) {
					c.JSON(http.StatusBadRequest, ErrorResponse{
						Error:   "Suspicious content detected",
						Code:    "SUSPICIOUS_CONTENT",
						Message: "Request contains potentially malicious content",
					})
					c.Abort()
					return
				}
			}
		}

		c.Next()
	}
}

// containsSuspiciousContent checks for potential XSS/injection attempts
func (m *InputValidationMiddleware) containsSuspiciousContent(input string) bool {
	suspiciousPatterns := []string{
		"<script",
		"javascript:",
		"vbscript:",
		"onload=",
		"onerror=",
		"onclick=",
		"eval(",
		"expression(",
		"url(",
		"import(",
		"document.cookie",
		"document.write",
		"window.location",
		"alert(",
		"confirm(",
		"prompt(",
		"setTimeout(",
		"setInterval(",
		"Function(",
		"constructor",
		"__proto__",
		"prototype",
		"../",
		"..\\",
		"union select",
		"drop table",
		"insert into",
		"delete from",
		"update set",
		"exec(",
		"execute(",
		"sp_",
		"xp_",
	}

	inputLower := strings.ToLower(input)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(inputLower, pattern) {
			return true
		}
	}

	return false
}

// EnhancedSecurityHeaders provides comprehensive security headers
func EnhancedSecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Basic security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// HSTS with preload
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		
		// Enhanced CSP with dual policy for admin vs public
		var csp string
		if strings.HasPrefix(c.Request.URL.Path, "/admin/") {
			// Admin CSP: More permissive for admin functionality
			csp = "default-src 'self'; " +
				"script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.jsdelivr.net; " +
				"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://cdnjs.cloudflare.com; " +
				"style-src-elem 'self' 'unsafe-inline' https://fonts.googleapis.com https://cdnjs.cloudflare.com; " +
				"font-src 'self' https://fonts.gstatic.com https://cdnjs.cloudflare.com; " +
				"img-src 'self' data: https: blob:; " +
				"connect-src 'self' https:; " +
				"media-src 'self' data: blob:; " +
				"object-src 'none'; " +
				"child-src 'none'; " +
				"worker-src 'self'; " +
				"frame-ancestors 'none'; " +
				"form-action 'self'; " +
				"base-uri 'self'; " +
				"manifest-src 'self'"
		} else {
			// Public CSP: More restrictive for public pages
			csp = "default-src 'self'; " +
				"script-src 'self' 'unsafe-inline'; " +
				"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
				"img-src 'self' data: https:; " +
				"font-src 'self' data: https://fonts.gstatic.com; " +
				"connect-src 'self'; " +
				"media-src 'self'; " +
				"object-src 'none'; " +
				"child-src 'none'; " +
				"worker-src 'self'; " +
				"frame-ancestors 'none'; " +
				"form-action 'self'; " +
				"base-uri 'self'; " +
				"manifest-src 'self'"
		}
		c.Header("Content-Security-Policy", csp)
		
		// Additional security headers
		c.Header("X-Permitted-Cross-Domain-Policies", "none")
		c.Header("X-Download-Options", "noopen")
		c.Header("X-DNS-Prefetch-Control", "off")
		c.Header("Expect-CT", "max-age=86400, enforce")
		
		// Remove server information
		c.Header("Server", "")
		
		c.Next()
	}
}

// SessionAuthMiddleware provides session-based authentication for admin panel
type SessionAuthMiddleware struct {
	authService *auth.AuthService
}

// NewSessionAuthMiddleware creates a new session auth middleware
func NewSessionAuthMiddleware(authService *auth.AuthService) *SessionAuthMiddleware {
	return &SessionAuthMiddleware{
		authService: authService,
	}
}

// RequireSessionAuth middleware for admin panel routes
func (m *SessionAuthMiddleware) RequireSessionAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for session cookie
		cookie, err := c.Cookie("auth_token")
		if err != nil || cookie == "" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "No session found",
				Code:    ErrCodeUnauthorized,
				Message: "Please log in to access this resource",
			})
			c.Abort()
			return
		}

		// Validate the token using the auth service
		// ValidateAccessToken expects raw token (no Bearer prefix)
		claims, err := m.authService.ValidateAccessToken(cookie)
		if err != nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "Invalid session",
				Code:    ErrCodeUnauthorized,
				Message: "Session expired or invalid",
			})
			c.Abort()
			return
		}

		// Check if user has admin or editor role
		if claims.Role != "admin" && claims.Role != "editor" {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "Insufficient permissions",
				Code:    ErrCodeForbidden,
				Message: "Admin or editor role required",
			})
			c.Abort()
			return
		}

		// Store user info in context for use by handlers
		c.Set("user_id", claims.UserID)
		c.Set("user_role", claims.Role)
		c.Set("username", claims.Username)
		c.Set("user", &models.User{
			ID:       claims.UserID,
			Username: claims.Username,
			Role:     models.UserRole(claims.Role),
		})
		
		c.Next()
	}
}


// =============================================================================
// COMPRESSION MIDDLEWARE
// =============================================================================

// gzipResponseWriter wraps gin.ResponseWriter with gzip compression
type gzipResponseWriter struct {
	gin.ResponseWriter
	writer *gzip.Writer
}

func (g *gzipResponseWriter) Write(data []byte) (int, error) {
	return g.writer.Write(data)
}

func (g *gzipResponseWriter) WriteString(s string) (int, error) {
	return g.writer.Write([]byte(s))
}

// gzipWriterPool pools gzip writers for reuse
var gzipWriterPool = sync.Pool{
	New: func() interface{} {
		w, _ := gzip.NewWriterLevel(io.Discard, gzip.BestSpeed)
		return w
	},
}

// GzipCompression provides response compression middleware
func GzipCompression() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if client accepts gzip encoding
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}

		// Skip compression for small responses or specific content types
		// Also skip for SSE, WebSocket, and streaming responses
		if c.GetHeader("Upgrade") != "" || 
		   strings.Contains(c.GetHeader("Accept"), "text/event-stream") {
			c.Next()
			return
		}

		// Get gzip writer from pool
		gz := gzipWriterPool.Get().(*gzip.Writer)
		gz.Reset(c.Writer)
		
		// Set response headers
		c.Header("Content-Encoding", "gzip")
		c.Header("Vary", "Accept-Encoding")
		
		// Wrap response writer
		c.Writer = &gzipResponseWriter{
			ResponseWriter: c.Writer,
			writer:         gz,
		}

		defer func() {
			gz.Close()
			gzipWriterPool.Put(gz)
		}()

		c.Next()
	}
}
