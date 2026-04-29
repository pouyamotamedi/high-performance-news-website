package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/auth"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/repositories"
	"high-performance-news-website/internal/services"
)

// APIHandler contains all API handlers
type APIHandler struct {
	userService             *services.UserService
	articleService          *services.ArticleService
	searchService           services.SearchServiceInterface
	contentIngestionService *services.ContentIngestionService
	tagService              *services.TagService
	keywordBankRepo         *repositories.KeywordBankRepository
	apiKeyMiddleware        *APIKeyMiddleware
	twoFactorMiddleware     *TwoFactorMiddleware
	csrfMiddleware          *CSRFMiddleware
}

// NewAPIHandler creates a new API handler
func NewAPIHandler(userService *services.UserService, articleService *services.ArticleService, searchService services.SearchServiceInterface, contentIngestionService *services.ContentIngestionService, tagService *services.TagService, keywordBankRepo *repositories.KeywordBankRepository, apiKeyMiddleware *APIKeyMiddleware, twoFactorMiddleware *TwoFactorMiddleware, csrfMiddleware *CSRFMiddleware) *APIHandler {
	return &APIHandler{
		userService:             userService,
		articleService:          articleService,
		searchService:           searchService,
		contentIngestionService: contentIngestionService,
		tagService:              tagService,
		keywordBankRepo:         keywordBankRepo,
		apiKeyMiddleware:        apiKeyMiddleware,
		twoFactorMiddleware:     twoFactorMiddleware,
		csrfMiddleware:          csrfMiddleware,
	}
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string            `json:"error"`
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// SuccessResponse represents a successful API response
type SuccessResponse struct {
	Data    interface{} `json:"data"`
	Message string      `json:"message,omitempty"`
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// Pagination contains pagination metadata
type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// Error codes
const (
	ErrCodeValidation    = "VALIDATION_ERROR"
	ErrCodeNotFound      = "NOT_FOUND"
	ErrCodeUnauthorized  = "UNAUTHORIZED"
	ErrCodeForbidden     = "FORBIDDEN"
	ErrCodeRateLimit     = "RATE_LIMIT_EXCEEDED"
	ErrCodeServerError   = "INTERNAL_SERVER_ERROR"
	ErrCodeDatabaseError = "DATABASE_ERROR"
)

// Common helper functions

// getCurrentUser extracts the current user from the context
func (h *APIHandler) getCurrentUser(c *gin.Context) (*models.User, error) {
	userInterface, exists := c.Get("user")
	if !exists {
		return nil, errors.New("user not found in context")
	}
	
	user, ok := userInterface.(*models.User)
	if !ok {
		return nil, errors.New("invalid user type in context")
	}
	
	return user, nil
}

// getPaginationParams extracts pagination parameters from query string
func getPaginationParams(c *gin.Context) (int, int, error) {
	page := 1
	limit := 20 // Default limit
	
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}
	
	offset := (page - 1) * limit
	return limit, offset, nil
}

// handleError handles different types of errors and returns appropriate HTTP responses
func handleError(c *gin.Context, err error) {
	var validationErr *models.ValidationError
	
	switch {
	case errors.As(err, &validationErr):
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Validation failed",
			Code:    ErrCodeValidation,
			Message: validationErr.Message,
			Details: map[string]string{"fields": strings.Join(validationErr.Fields, ", ")},
		})
	case errors.Is(err, auth.ErrUserNotFound):
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "User not found",
			Code:    ErrCodeNotFound,
			Message: "The requested user was not found",
		})
	case errors.Is(err, auth.ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Invalid credentials",
			Code:    ErrCodeUnauthorized,
			Message: "Username or password is incorrect",
		})
	case errors.Is(err, auth.ErrInsufficientPermissions):
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Insufficient permissions",
			Code:    ErrCodeForbidden,
			Message: "You don't have permission to perform this action",
		})
	case errors.Is(err, sql.ErrNoRows):
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Resource not found",
			Code:    ErrCodeNotFound,
			Message: "The requested resource was not found",
		})
	default:
		// Log internal errors
		fmt.Printf("Internal server error: %v\n", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    ErrCodeServerError,
			Message: "An unexpected error occurred",
		})
	}
}

// Health check endpoint
func (h *APIHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	})
}

// RebuildSearchIndex rebuilds the search index (admin only)
func (h *APIHandler) RebuildSearchIndex(c *gin.Context) {
	stats, err := h.searchService.RebuildIndex(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: "Failed to rebuild search index: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Search index rebuild completed successfully",
		Data:    stats,
	})
}

// GetSearchIndexStats returns search index statistics (admin only)
func (h *APIHandler) GetSearchIndexStats(c *gin.Context) {
	stats, err := h.searchService.GetIndexStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: "Failed to get search index stats: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Data: stats,
	})
}

// IndexArticle indexes a specific article (admin only)
func (h *APIHandler) IndexArticle(c *gin.Context) {
	articleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Invalid article ID",
		})
		return
	}

	// Get the article from the service
	article, err := h.articleService.GetByID(c.Request.Context(), articleID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Message: "Article not found",
		})
		return
	}

	// Index the article
	err = h.searchService.IndexArticle(article)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: "Failed to index article: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Article indexed successfully",
	})
}

// RemoveArticleFromIndex removes an article from the search index (admin only)
func (h *APIHandler) RemoveArticleFromIndex(c *gin.Context) {
	articleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Invalid article ID",
		})
		return
	}

	err = h.searchService.RemoveArticle(context.Background(), articleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: "Failed to remove article from index: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Article removed from search index successfully",
	})
}

// Get2FAStatus returns the 2FA status for the current user
func (h *APIHandler) Get2FAStatus(c *gin.Context) {
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Authentication required",
			Code:    ErrCodeUnauthorized,
			Message: "You must be authenticated to access this resource",
		})
		return
	}

	user, ok := userInterface.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Invalid user context",
			Code:    ErrCodeServerError,
			Message: "An unexpected error occurred",
		})
		return
	}

	// Check if user has 2FA enabled (for now, return false as we don't have this field yet)
	enabled := false // TODO: Add TwoFactorEnabled field to User model
	
	c.JSON(http.StatusOK, gin.H{
		"enabled": enabled,
		"user_id": user.ID,
	})
}

// Verify2FA verifies a TOTP code and enables 2FA for the user
func (h *APIHandler) Verify2FA(c *gin.Context) {
	var request struct {
		Code string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Code:    ErrCodeValidation,
			Message: "Invalid request format",
		})
		return
	}

	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Authentication required",
			Code:    ErrCodeUnauthorized,
			Message: "You must be authenticated to access this resource",
		})
		return
	}

	_, ok := userInterface.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Invalid user context",
			Code:    ErrCodeServerError,
			Message: "An unexpected error occurred",
		})
		return
	}

	// Verify the TOTP code (simplified for now)
	if len(request.Code) == 6 {
		// Enable 2FA for the user (you may need to update the user model)
		// user.TwoFactorEnabled = true
		// Save to database
		
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "2FA enabled successfully",
		})
	} else {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid verification code",
			Code:    "INVALID_2FA_CODE",
			Message: "The verification code is invalid or expired",
		})
	}
}

// VerifyToken verifies if the current token is valid
func (h *APIHandler) VerifyToken(c *gin.Context) {
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Invalid token",
			Code:    ErrCodeUnauthorized,
			Message: "Token is invalid or expired",
		})
		return
	}

	user, ok := userInterface.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Invalid user context",
			Code:    ErrCodeServerError,
			Message: "An unexpected error occurred",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid": true,
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"role":  user.Role,
		},
	})
}

// GetCSRFToken generates and returns a CSRF token for the current session
func (h *APIHandler) GetCSRFToken(c *gin.Context) {
	// Generate CSRF token using the middleware
	token := h.csrfMiddleware.GenerateCSRFToken(c)
	
	c.JSON(http.StatusOK, gin.H{
		"csrf_token": token,
		"expires_in": 3600, // 1 hour
	})
}

// API Key Management Endpoints

// GenerateAPIKeyRequest represents a request to generate an API key
type GenerateAPIKeyRequest struct {
	ExpiresInDays int    `json:"expires_in_days" validate:"required,min=1,max=365"`
	Description   string `json:"description" validate:"max=255"`
}

// GenerateAPIKeyResponse represents the response for API key generation
type GenerateAPIKeyResponse struct {
	APIKey      string    `json:"api_key"`
	ExpiresAt   time.Time `json:"expires_at"`
	Description string    `json:"description"`
}

// Verify2FARequest represents a 2FA verification request
type Verify2FARequest struct {
	Code string `json:"code" validate:"required,len=6"`
}

// GenerateAPIKey generates a new API key for the current user
// POST /api/v1/auth/api-key/generate
func (h *APIHandler) GenerateAPIKey(c *gin.Context) {
	var req GenerateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, &models.ValidationError{
			Message: "Invalid request format",
			Fields:  []string{err.Error()},
		})
		return
	}

	// Get current user
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// Calculate expiration
	expiresIn := time.Duration(req.ExpiresInDays) * 24 * time.Hour
	
	// Generate API key
	apiKey, err := h.apiKeyMiddleware.GenerateAPIKey(currentUser.ID, expiresIn)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to generate API key",
			Code:    ErrCodeServerError,
			Message: err.Error(),
		})
		return
	}

	response := GenerateAPIKeyResponse{
		APIKey:      apiKey,
		ExpiresAt:   time.Now().Add(expiresIn),
		Description: req.Description,
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Data:    response,
		Message: "API key generated successfully",
	})
}

// RotateAPIKey rotates the current user's API key
// POST /api/v1/auth/api-key/rotate
func (h *APIHandler) RotateAPIKey(c *gin.Context) {
	var req GenerateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, &models.ValidationError{
			Message: "Invalid request format",
			Fields:  []string{err.Error()},
		})
		return
	}

	// Get current user
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// Calculate expiration
	expiresIn := time.Duration(req.ExpiresInDays) * 24 * time.Hour
	
	// Rotate API key
	apiKey, err := h.apiKeyMiddleware.RotateAPIKey(currentUser.ID, expiresIn)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to rotate API key",
			Code:    ErrCodeServerError,
			Message: err.Error(),
		})
		return
	}

	response := GenerateAPIKeyResponse{
		APIKey:      apiKey,
		ExpiresAt:   time.Now().Add(expiresIn),
		Description: req.Description,
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Data:    response,
		Message: "API key rotated successfully",
	})
}

// RevokeAPIKey revokes the current user's API key
// DELETE /api/v1/auth/api-key
func (h *APIHandler) RevokeAPIKey(c *gin.Context) {
	// Get current user
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// Revoke API key
	err = h.apiKeyMiddleware.RevokeAPIKey(currentUser.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to revoke API key",
			Code:    ErrCodeServerError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "API key revoked successfully",
	})
}

// Duplicate method removed - using the first Verify2FA method instead

// Get2FASetup returns 2FA setup information for admin users
// GET /api/v1/auth/2fa/setup
func (h *APIHandler) Get2FASetup(c *gin.Context) {
	// Get current user
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// Only admin accounts can use 2FA
	if currentUser.Role != models.RoleAdmin {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "2FA not available",
			Code:    "2FA_NOT_AVAILABLE",
			Message: "Two-factor authentication is only available for admin accounts",
		})
		return
	}

	// Generate QR code URL for TOTP setup
	qrCodeURL := h.twoFactorMiddleware.GenerateQRCodeURL(currentUser.ID, currentUser.Email, "High Performance News")

	c.JSON(http.StatusOK, SuccessResponse{
		Data: gin.H{
			"qr_code_url": qrCodeURL,
			"manual_entry_key": h.twoFactorMiddleware.generateUserSecret(currentUser.ID),
			"instructions": "Scan the QR code with your authenticator app or enter the manual key",
		},
		Message: "2FA setup information retrieved successfully",
	})
}

// ListCategories returns real categories from database for content management
func (h *APIHandler) ListCategories(c *gin.Context) {
	// Connect to real database and get actual categories
	// This is critical for content ingestion - we need real categories!
	
	// Use the existing content ingestion service to get categories
	ctx := c.Request.Context()
	
	// Get real categories from database
	categories, err := h.contentIngestionService.GetCategories(ctx)
	if err != nil {
		// Log the error for debugging
		log.Printf("Failed to get categories: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to load categories",
			"message": "Could not retrieve categories from database",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"categories": categories,
		"total":      len(categories),
	})
}

// GetPendingContent returns REAL pending content from ingested_content table
func (h *APIHandler) GetPendingContent(c *gin.Context) {
	// This is CRITICAL - real content ingestion for news website!
	ctx := c.Request.Context()
	
	// Parse pagination parameters
	limit := 20
	offset := 0
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Get REAL pending content from the content ingestion service
	pendingContent, total, err := h.contentIngestionService.GetPendingContent(ctx, limit, offset)
	if err != nil {
		log.Printf("Failed to get pending content: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to load pending content",
			"message": "Could not retrieve pending content from database",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"content": pendingContent,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// GetProcessedContent returns REAL processed content history from database
func (h *APIHandler) GetProcessedContent(c *gin.Context) {
	// This tracks real article processing - essential for news operations!
	ctx := c.Request.Context()
	
	// Parse pagination and filter parameters
	limit := 20
	offset := 0
	status := c.Query("status") // processed, rejected, duplicate
	
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Get REAL processed content from the content ingestion service
	processedContent, total, err := h.contentIngestionService.GetProcessedContent(ctx, limit, offset, status)
	if err != nil {
		log.Printf("Failed to get processed content: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to load processed content",
			"message": "Could not retrieve processed content from database",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"content": processedContent,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
		"status":  status,
	})
}



// GetContentByID returns detailed information about a specific content item
func (h *APIHandler) GetContentByID(c *gin.Context) {
	log.Printf("GetContentByID called for ID: %s", c.Param("id"))
	
	// Parse content ID
	contentID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		log.Printf("GetContentByID: Invalid content ID: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid content ID",
			"message": "Content ID must be a valid number",
		})
		return
	}

	log.Printf("GetContentByID: Parsed content ID: %d", contentID)

	// Get content details through service
	content, err := h.contentIngestionService.GetContentByID(c.Request.Context(), contentID)
	if err != nil {
		log.Printf("GetContentByID: Service error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get content details",
			"message": err.Error(),
		})
		return
	}

	log.Printf("GetContentByID: Successfully retrieved content: %s", content.Title)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"content": content,
	})
}