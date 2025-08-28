package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"high-performance-news-website/internal/services"

	"github.com/gin-gonic/gin"
)

// SearchHandlers contains search-related HTTP handlers
type SearchHandlers struct {
	searchService *services.SearchService
}

// NewSearchHandlers creates a new search handlers instance
func NewSearchHandlers(searchService *services.SearchService) *SearchHandlers {
	return &SearchHandlers{
		searchService: searchService,
	}
}

// SearchRequest represents the HTTP search request
type SearchRequestHTTP struct {
	Query        string   `form:"q" json:"q"`
	AuthorID     *uint64  `form:"author_id" json:"author_id"`
	CategoryID   *uint64  `form:"category_id" json:"category_id"`
	Tags         []string `form:"tags" json:"tags"`
	LanguageCode string   `form:"language" json:"language"`
	DateFrom     string   `form:"date_from" json:"date_from"` // ISO 8601 format
	DateTo       string   `form:"date_to" json:"date_to"`     // ISO 8601 format
	SortBy       string   `form:"sort_by" json:"sort_by"`     // published_at, view_count, like_count
	SortOrder    string   `form:"sort_order" json:"sort_order"` // asc, desc
	Page         int      `form:"page" json:"page"`
	Limit        int      `form:"limit" json:"limit"`
}

// SearchResponse represents the HTTP search response
type SearchResponseHTTP struct {
	Success        bool                    `json:"success"`
	Data           *services.SearchResponse `json:"data,omitempty"`
	Error          string                  `json:"error,omitempty"`
	Pagination     *PaginationInfo         `json:"pagination,omitempty"`
}

// PaginationInfo contains pagination metadata
type PaginationInfo struct {
	CurrentPage int   `json:"current_page"`
	PerPage     int   `json:"per_page"`
	Total       int64 `json:"total"`
	TotalPages  int   `json:"total_pages"`
	HasNext     bool  `json:"has_next"`
	HasPrev     bool  `json:"has_prev"`
}

// Search handles search requests with filtering, sorting, and pagination
func (sh *SearchHandlers) Search(c *gin.Context) {
	var req SearchRequestHTTP

	// Bind query parameters
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, SearchResponseHTTP{
			Success: false,
			Error:   "Invalid query parameters: " + err.Error(),
		})
		return
	}

	// Set defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 20 // Default limit
	}
	if req.SortBy == "" {
		req.SortBy = "published_at"
	}
	if req.SortOrder == "" {
		req.SortOrder = "desc"
	}

	// Validate sort parameters
	validSortFields := map[string]bool{
		"published_at": true,
		"created_at":   true,
		"view_count":   true,
		"like_count":   true,
	}
	if !validSortFields[req.SortBy] {
		c.JSON(http.StatusBadRequest, SearchResponseHTTP{
			Success: false,
			Error:   "Invalid sort field. Allowed: published_at, created_at, view_count, like_count",
		})
		return
	}

	if req.SortOrder != "asc" && req.SortOrder != "desc" {
		c.JSON(http.StatusBadRequest, SearchResponseHTTP{
			Success: false,
			Error:   "Invalid sort order. Allowed: asc, desc",
		})
		return
	}

	// Convert to service request
	serviceReq, err := sh.convertToServiceRequest(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, SearchResponseHTTP{
			Success: false,
			Error:   "Invalid request parameters: " + err.Error(),
		})
		return
	}

	// Perform search
	result, err := sh.searchService.Search(*serviceReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, SearchResponseHTTP{
			Success: false,
			Error:   "Search failed: " + err.Error(),
		})
		return
	}

	// Calculate pagination info
	totalPages := int((result.Total + int64(req.Limit) - 1) / int64(req.Limit))
	pagination := &PaginationInfo{
		CurrentPage: req.Page,
		PerPage:     req.Limit,
		Total:       result.Total,
		TotalPages:  totalPages,
		HasNext:     req.Page < totalPages,
		HasPrev:     req.Page > 1,
	}

	c.JSON(http.StatusOK, SearchResponseHTTP{
		Success:    true,
		Data:       result,
		Pagination: pagination,
	})
}

// Suggestions handles search suggestion requests
func (sh *SearchHandlers) Suggestions(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Query parameter 'q' is required",
		})
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 20 {
		limit = 10
	}

	suggestions, err := sh.searchService.GetSuggestions(c.Request.Context(), query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get suggestions: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"suggestions": suggestions,
	})
}

// SearchHealth checks the health of search components
func (sh *SearchHandlers) SearchHealth(c *gin.Context) {
	health := sh.searchService.HealthCheck()
	
	// Determine overall status
	allHealthy := true
	for _, status := range health {
		if statusStr, ok := status.(string); ok && strings.Contains(statusStr, "error") {
			allHealthy = false
			break
		}
	}

	statusCode := http.StatusOK
	if !allHealthy {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, gin.H{
		"success": allHealthy,
		"health":  health,
	})
}

// InvalidateSearchCache invalidates search cache
func (sh *SearchHandlers) InvalidateSearchCache(c *gin.Context) {
	pattern := c.Query("pattern")
	
	if err := sh.searchService.InvalidateCache(pattern); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to invalidate cache: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cache invalidated successfully",
	})
}

// convertToServiceRequest converts HTTP request to service request
func (sh *SearchHandlers) convertToServiceRequest(req SearchRequestHTTP) (*services.SearchRequest, error) {
	// Calculate offset from page
	offset := (req.Page - 1) * req.Limit

	serviceReq := &services.SearchRequest{
		Query:  strings.TrimSpace(req.Query),
		Limit:  req.Limit,
		Offset: offset,
	}

	// Add filters if provided
	if req.AuthorID != nil || req.CategoryID != nil || len(req.Tags) > 0 || 
	   req.LanguageCode != "" || req.DateFrom != "" || req.DateTo != "" {
		
		filters := &services.SearchFilters{
			AuthorID:     req.AuthorID,
			CategoryID:   req.CategoryID,
			Tags:         req.Tags,
			LanguageCode: req.LanguageCode,
			Status:       "published", // Always filter for published articles
		}

		// Parse date filters
		if req.DateFrom != "" {
			if dateFrom, err := time.Parse(time.RFC3339, req.DateFrom); err == nil {
				timestamp := dateFrom.Unix()
				filters.DateFrom = &timestamp
			} else {
				return nil, err
			}
		}

		if req.DateTo != "" {
			if dateTo, err := time.Parse(time.RFC3339, req.DateTo); err == nil {
				timestamp := dateTo.Unix()
				filters.DateTo = &timestamp
			} else {
				return nil, err
			}
		}

		serviceReq.Filters = filters
	}

	// Add sorting
	if req.SortBy != "" && req.SortOrder != "" {
		serviceReq.Sort = &services.SearchSort{
			Field: req.SortBy,
			Order: req.SortOrder,
		}
	}

	return serviceReq, nil
}

// RegisterSearchRoutes registers search-related routes
func (sh *SearchHandlers) RegisterSearchRoutes(router *gin.RouterGroup) {
	search := router.Group("/search")
	{
		search.GET("", sh.Search)
		search.GET("/suggestions", sh.Suggestions)
		search.GET("/health", sh.SearchHealth)
		search.DELETE("/cache", sh.InvalidateSearchCache) // Admin endpoint
	}
}