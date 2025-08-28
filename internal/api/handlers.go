package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/auth"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// APIHandler contains all API handlers
type APIHandler struct {
	userService             *services.UserService
	articleService          *services.ArticleService
	searchService           *services.SearchService
	contentIngestionService *services.ContentIngestionService
}

// NewAPIHandler creates a new API handler
func NewAPIHandler(userService *services.UserService, articleService *services.ArticleService, searchService *services.SearchService, contentIngestionService *services.ContentIngestionService) *APIHandler {
	return &APIHandler{
		userService:             userService,
		articleService:          articleService,
		searchService:           searchService,
		contentIngestionService: contentIngestionService,
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