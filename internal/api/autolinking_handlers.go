package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// AutoLinkingHandler handles auto-linking admin API requests
type AutoLinkingHandler struct {
	autoLinkingService *services.AutoLinkingService
	settingsRepo       AutoLinkingSettingsRepository
}

// AutoLinkingSettingsRepository interface for settings storage
type AutoLinkingSettingsRepository interface {
	GetSettings(ctx context.Context) (*AutoLinkingSettings, error)
	UpdateSettings(ctx context.Context, settings *AutoLinkingSettings) error
}

// AutoLinkingSettings represents auto-linking configuration
type AutoLinkingSettings struct {
	GlobalEnabled            bool `json:"global_enabled"`
	ContentIngestionEnabled  bool `json:"content_ingestion_enabled"`
}

// NewAutoLinkingHandler creates a new auto-linking handler
func NewAutoLinkingHandler(autoLinkingService *services.AutoLinkingService, settingsRepo AutoLinkingSettingsRepository) *AutoLinkingHandler {
	return &AutoLinkingHandler{
		autoLinkingService: autoLinkingService,
		settingsRepo:       settingsRepo,
	}
}

// GetStats returns auto-linking system statistics
func (h *AutoLinkingHandler) GetStats(c *gin.Context) {
	ctx := c.Request.Context()
	
	// Load keywords to ensure stats are up-to-date
	if err := h.autoLinkingService.LoadKeywords(ctx); err != nil {
		// Log the error for debugging
		c.Error(err)
		// If loading fails, return empty stats
		c.JSON(http.StatusOK, gin.H{
			"total_keywords": 0,
			"active_tags":    0,
			"status":         "Inactive",
			"error":          err.Error(),
		})
		return
	}
	
	stats := h.autoLinkingService.GetTrieStats()
	
	status := "Active"
	if stats["total_keywords"] == 0 {
		status = "Inactive"
	}
	
	c.JSON(http.StatusOK, gin.H{
		"total_keywords":    stats["total_keywords"],
		"active_tags":       stats["total_tags"],
		"active_banks":      stats["total_banks"],
		"status":            status,
	})
}

// GetSettings returns current auto-linking settings
func (h *AutoLinkingHandler) GetSettings(c *gin.Context) {
	ctx := c.Request.Context()
	
	settings, err := h.settingsRepo.GetSettings(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to load settings",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, settings)
}

// UpdateSettings updates auto-linking settings
func (h *AutoLinkingHandler) UpdateSettings(c *gin.Context) {
	ctx := c.Request.Context()
	
	var settings AutoLinkingSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}
	
	if err := h.settingsRepo.UpdateSettings(ctx, &settings); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to update settings",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Settings updated successfully",
		Data:    settings,
	})
}

// RefreshKeywords refreshes the keyword Trie from database
func (h *AutoLinkingHandler) RefreshKeywords(c *gin.Context) {
	ctx := c.Request.Context()
	
	if err := h.autoLinkingService.RefreshKeywords(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to refresh keywords",
			Message: err.Error(),
		})
		return
	}
	
	stats := h.autoLinkingService.GetTrieStats()
	
	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Keywords refreshed successfully",
		Data: gin.H{
			"total_keywords": stats["total_keywords"],
			"active_tags":    stats["total_tags"],
		},
	})
}

// CheckConflicts checks for keyword conflicts across tags
func (h *AutoLinkingHandler) CheckConflicts(c *gin.Context) {
	ctx := c.Request.Context()
	
	conflicts, err := h.autoLinkingService.ValidateKeywordConflicts(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to check conflicts",
			Message: err.Error(),
		})
		return
	}
	
	// Parse conflicts into structured format
	conflictList := []gin.H{}
	
	for _, conflict := range conflicts {
		// Parse conflict string (format: "keyword: tag1, tag2")
		// This is a simplified parser - adjust based on actual format
		conflictList = append(conflictList, gin.H{
			"keyword": conflict,
			"tags":    []string{}, // TODO: Parse actual tags from conflict string
		})
	}
	
	c.JSON(http.StatusOK, conflictList)
}

// TestAutoLinking tests auto-linking on sample text
func (h *AutoLinkingHandler) TestAutoLinking(c *gin.Context) {
	ctx := c.Request.Context()
	
	var request struct {
		Text string `json:"text" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "Text field is required",
		})
		return
	}
	
	// Create a temporary article for testing
	testArticle := &models.Article{
		Content:     request.Text,
		AutoLinking: true,
	}
	
	// Get keyword matches
	matches, err := h.autoLinkingService.GetKeywordMatches(ctx, request.Text)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to process text",
			Message: err.Error(),
		})
		return
	}
	
	// Process the text
	processedText, err := h.autoLinkingService.ProcessArticleLinks(ctx, testArticle)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to process text",
			Message: err.Error(),
		})
		return
	}
	
	// Format matches for response
	matchList := []gin.H{}
	for _, match := range matches {
		matchList = append(matchList, gin.H{
			"keyword": match.Keyword,
			"tag":     match.Tag.Name,
		})
	}
	
	c.JSON(http.StatusOK, gin.H{
		"processed_text": processedText,
		"matches":        matchList,
	})
}
