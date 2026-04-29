package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// Content Ingestion API request/response types

// CreateContentSourceRequest represents a request to create a content source
type CreateContentSourceRequest struct {
	Name      string                  `json:"name" validate:"required,max=100"`
	Type      string                  `json:"type" validate:"required,oneof=api webhook manual"`
	APIKey    string                  `json:"api_key,omitempty"`
	IsActive  bool                    `json:"is_active"`
	RateLimit int                     `json:"rate_limit" validate:"min=0,max=10000"`
	Priority  int                     `json:"priority" validate:"min=1,max=10"`
	Config    models.SourceConfig     `json:"config"`
}

// ContentSourceResponse represents a content source response
type ContentSourceResponse struct {
	ID        uint64              `json:"id"`
	Name      string              `json:"name"`
	Type      string              `json:"type"`
	APIKey    string              `json:"api_key"`
	IsActive  bool                `json:"is_active"`
	RateLimit int                 `json:"rate_limit"`
	Priority  int                 `json:"priority"`
	Config    models.SourceConfig `json:"config"`
	CreatedAt string              `json:"created_at"`
	UpdatedAt string              `json:"updated_at"`
}

// ContentSourceListResponse represents the response for content source listing
type ContentSourceListResponse struct {
	Sources    []ContentSourceResponse `json:"sources"`
	Pagination Pagination              `json:"pagination"`
}

// IngestionStatsResponse represents ingestion statistics
type IngestionStatsResponse struct {
	SourceID *uint64        `json:"source_id,omitempty"`
	Hours    int            `json:"hours"`
	Stats    map[string]int `json:"stats"`
}

// ProcessBatchResponse represents batch processing results
type ProcessBatchResponse struct {
	Processed int `json:"processed"`
	Limit     int `json:"limit"`
}

// Content Ingestion Handlers

// IngestContent handles external content ingestion
func (h *APIHandler) IngestContent(c *gin.Context) {
	// Get API key from header
	apiKey := c.GetHeader("X-API-Key")
	if apiKey == "" {
		handleError(c, &models.ValidationError{
			Message: "API key required",
			Fields:  []string{"X-API-Key header is required"},
		})
		return
	}

	var req models.ContentIngestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, &models.ValidationError{
			Message: "Invalid request format",
			Fields:  []string{err.Error()},
		})
		return
	}

	// Validate request
	if err := h.contentIngestionService.ValidateContentRequest(&req); err != nil {
		handleError(c, err)
		return
	}

	// Ingest content
	ingestedContent, err := h.contentIngestionService.IngestContent(c.Request.Context(), apiKey, &req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Data:    ingestedContent,
		Message: "Content ingested successfully",
	})
}

// CreateContentSource creates a new content source
func (h *APIHandler) CreateContentSource(c *gin.Context) {
	var req CreateContentSourceRequest
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

	// Create content source model
	source := &models.ContentSource{
		Name:      req.Name,
		Type:      req.Type,
		APIKey:    req.APIKey,
		IsActive:  req.IsActive,
		RateLimit: req.RateLimit,
		Priority:  req.Priority,
		Config:    req.Config,
	}

	// Create content source through service
	createdSource, err := h.contentIngestionService.CreateContentSource(c.Request.Context(), source, currentUser)
	if err != nil {
		handleError(c, err)
		return
	}

	// Convert to response format
	response := ContentSourceResponse{
		ID:        createdSource.ID,
		Name:      createdSource.Name,
		Type:      createdSource.Type,
		APIKey:    createdSource.APIKey,
		IsActive:  createdSource.IsActive,
		RateLimit: createdSource.RateLimit,
		Priority:  createdSource.Priority,
		Config:    createdSource.Config,
		CreatedAt: createdSource.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: createdSource.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Data:    response,
		Message: "Content source created successfully",
	})
}

// ListContentSources retrieves content sources with pagination
func (h *APIHandler) ListContentSources(c *gin.Context) {
	// Get pagination parameters
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// Get current user
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// List content sources through service
	sources, total, err := h.contentIngestionService.ListContentSources(c.Request.Context(), limit, offset, currentUser)
	if err != nil {
		handleError(c, err)
		return
	}

	// Convert to response format
	var responseData []ContentSourceResponse
	for _, source := range sources {
		responseData = append(responseData, ContentSourceResponse{
			ID:        source.ID,
			Name:      source.Name,
			Type:      source.Type,
			APIKey:    source.APIKey,
			IsActive:  source.IsActive,
			RateLimit: source.RateLimit,
			Priority:  source.Priority,
			Config:    source.Config,
			CreatedAt: source.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt: source.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	// Calculate pagination
	page := (offset / limit) + 1
	totalPages := (total + limit - 1) / limit

	response := ContentSourceListResponse{
		Sources: responseData,
		Pagination: Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	c.JSON(http.StatusOK, response)
}

// ProcessPendingContent processes a specific pending content item
func (h *APIHandler) ProcessPendingContent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		handleError(c, &models.ValidationError{
			Message: "Invalid content ID",
			Fields:  []string{"id must be a valid number"},
		})
		return
	}

	// Get current user
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// Check permissions - only admins and editors can process content
	if currentUser == nil || (!currentUser.HasPermission("admin") && !currentUser.HasPermission("edit")) {
		handleError(c, &models.ValidationError{
			Message: "Insufficient permissions",
			Fields:  []string{"admin or editor permissions required"},
		})
		return
	}

	// Process content through service
	err = h.contentIngestionService.ProcessPendingContent(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Content processed successfully",
	})
}

// ProcessBatchContent processes multiple pending content items
func (h *APIHandler) ProcessBatchContent(c *gin.Context) {
	// Get limit parameter
	limit := 10 // Default limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Get current user
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// Check permissions - only admins and editors can process content
	if currentUser == nil || (!currentUser.HasPermission("admin") && !currentUser.HasPermission("edit")) {
		handleError(c, &models.ValidationError{
			Message: "Insufficient permissions",
			Fields:  []string{"admin or editor permissions required"},
		})
		return
	}

	// Process batch through service
	processed, err := h.contentIngestionService.ProcessBatchContent(c.Request.Context(), limit)
	if err != nil {
		handleError(c, err)
		return
	}

	response := ProcessBatchResponse{
		Processed: processed,
		Limit:     limit,
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Data:    response,
		Message: "Batch processing completed",
	})
}

// GetIngestionStats retrieves ingestion statistics
func (h *APIHandler) GetIngestionStats(c *gin.Context) {
	// Get parameters
	var sourceID *uint64
	if sourceIDStr := c.Query("source_id"); sourceIDStr != "" {
		if id, err := strconv.ParseUint(sourceIDStr, 10, 64); err == nil {
			sourceID = &id
		}
	}

	hours := 24 // Default to last 24 hours
	if hoursStr := c.Query("hours"); hoursStr != "" {
		if h, err := strconv.Atoi(hoursStr); err == nil && h > 0 && h <= 168 { // Max 1 week
			hours = h
		}
	}

	// Get current user
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// Check permissions - only admins and editors can view stats
	if currentUser == nil || (!currentUser.HasPermission("admin") && !currentUser.HasPermission("edit")) {
		handleError(c, &models.ValidationError{
			Message: "Insufficient permissions",
			Fields:  []string{"admin or editor permissions required"},
		})
		return
	}

	// Get stats through service
	stats, err := h.contentIngestionService.GetIngestionStats(c.Request.Context(), sourceID, hours)
	if err != nil {
		handleError(c, err)
		return
	}

	response := IngestionStatsResponse{
		SourceID: sourceID,
		Hours:    hours,
		Stats:    stats,
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Data: response,
	})
}

// WebhookIngestion handles webhook-based content ingestion
func (h *APIHandler) WebhookIngestion(c *gin.Context) {
	// Get source ID from URL parameter
	sourceIDStr := c.Param("source_id")
	_, err := strconv.ParseUint(sourceIDStr, 10, 64)
	if err != nil {
		handleError(c, &models.ValidationError{
			Message: "Invalid source ID",
			Fields:  []string{"source_id must be a valid number"},
		})
		return
	}

	// Get webhook secret from header for verification
	webhookSecret := c.GetHeader("X-Webhook-Secret")
	if webhookSecret == "" {
		handleError(c, &models.ValidationError{
			Message: "Webhook secret required",
			Fields:  []string{"X-Webhook-Secret header is required"},
		})
		return
	}

	// TODO: Verify webhook secret against stored source configuration

	var req models.ContentIngestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, &models.ValidationError{
			Message: "Invalid request format",
			Fields:  []string{err.Error()},
		})
		return
	}

	// Validate request
	if err := h.contentIngestionService.ValidateContentRequest(&req); err != nil {
		handleError(c, err)
		return
	}

	// For webhook ingestion, we need to get the source by ID and use its API key
	// This is a simplified implementation - in production, you'd have proper webhook verification
	
	// TODO: Implement proper webhook ingestion with source lookup by ID
	// For now, return a placeholder response
	
	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Webhook received successfully",
	})
}

// Helper function to add content ingestion service to APIHandler
// This would be added to the APIHandler struct and constructor
type ContentIngestionHandler struct {
	contentIngestionService *services.ContentIngestionService
}

// Note: In the actual implementation, you would add contentIngestionService to the main APIHandler struct
// and initialize it in NewAPIHandler. This is shown here for clarity of what needs to be added.

// UpdateContentSource updates an existing content source
func (h *APIHandler) UpdateContentSource(c *gin.Context) {
	// Parse source ID
	sourceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid source ID",
		})
		return
	}

	// Parse request body
	var req CreateContentSourceRequest
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

	// Create content source model
	source := &models.ContentSource{
		ID:        sourceID,
		Name:      req.Name,
		Type:      req.Type,
		APIKey:    req.APIKey,
		IsActive:  req.IsActive,
		RateLimit: req.RateLimit,
		Priority:  req.Priority,
		Config:    req.Config,
	}

	// Update content source through service
	updatedSource, err := h.contentIngestionService.UpdateContentSource(c.Request.Context(), source, currentUser)
	if err != nil {
		handleError(c, err)
		return
	}

	// Convert to response format
	response := ContentSourceResponse{
		ID:        updatedSource.ID,
		Name:      updatedSource.Name,
		Type:      updatedSource.Type,
		APIKey:    updatedSource.APIKey,
		IsActive:  updatedSource.IsActive,
		RateLimit: updatedSource.RateLimit,
		Priority:  updatedSource.Priority,
		Config:    updatedSource.Config,
		CreatedAt: updatedSource.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: updatedSource.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Data:    response,
		Message: "Content source updated successfully",
	})
}

// RejectPendingContent rejects a pending content item
func (h *APIHandler) RejectPendingContent(c *gin.Context) {
	// Parse content ID
	contentID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid content ID",
		})
		return
	}

	// Parse request body for rejection reason
	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// If no body provided, use default reason
		req.Reason = "Rejected by admin"
	}

	// Get current user
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// Reject content through service
	err = h.contentIngestionService.RejectPendingContent(c.Request.Context(), contentID, req.Reason, currentUser)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Content rejected successfully",
	})
}

// GetIngestionStatsAdmin returns content ingestion statistics for admin panel
func (h *APIHandler) GetIngestionStatsAdmin(c *gin.Context) {
	// Get ingestion stats from service
	stats, err := h.contentIngestionService.GetIngestionStatsForAdmin(c.Request.Context())
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Data: stats,
	})
}

// ProcessPendingContentByID processes a single pending content item by ID
func (h *APIHandler) ProcessPendingContentByID(c *gin.Context) {
	// Parse content ID
	contentID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid content ID",
		})
		return
	}

	// Get current user
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// Process content through service
	err = h.contentIngestionService.ProcessPendingContentByID(c.Request.Context(), contentID, currentUser)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Content processed successfully",
	})
}

// ProcessBatchContentByIDs processes multiple pending content items by their IDs
func (h *APIHandler) ProcessBatchContentByIDs(c *gin.Context) {
	// Parse request body
	var req struct {
		ContentIDs []uint64 `json:"content_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, &models.ValidationError{
			Message: "Invalid request format",
			Fields:  []string{err.Error()},
		})
		return
	}

	if len(req.ContentIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No content IDs provided",
		})
		return
	}

	// Get current user
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// Process batch through service
	processedCount, err := h.contentIngestionService.ProcessBatchContentByIDs(c.Request.Context(), req.ContentIDs, currentUser)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Data: gin.H{
			"processed_count": processedCount,
			"total_requested": len(req.ContentIDs),
		},
		Message: fmt.Sprintf("Processed %d out of %d content items", processedCount, len(req.ContentIDs)),
	})
}

// DeleteContentSource deletes a content source
func (h *APIHandler) DeleteContentSource(c *gin.Context) {
	// Parse source ID
	sourceID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid source ID",
		})
		return
	}

	// Get current user
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// Delete content source through service
	err = h.contentIngestionService.DeleteContentSource(c.Request.Context(), sourceID, currentUser)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Content source deleted successfully",
	})
}

// ReprocessRejectedContent moves rejected content back to pending status
func (h *APIHandler) ReprocessRejectedContent(c *gin.Context) {
	// Parse content ID
	contentID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid content ID",
		})
		return
	}

	// Get current user
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// Reprocess content through service
	err = h.contentIngestionService.ReprocessRejectedContent(c.Request.Context(), contentID, currentUser)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Content moved back to pending for reprocessing",
	})
}