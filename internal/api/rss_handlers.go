package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/services"
)

// RSSHandlers handles RSS feed HTTP requests
type RSSHandlers struct {
	rssService services.RSSServiceInterface
}

// NewRSSHandlers creates a new RSS handlers instance
func NewRSSHandlers(rssService services.RSSServiceInterface) *RSSHandlers {
	return &RSSHandlers{
		rssService: rssService,
	}
}

// HandleMainRSSFeed serves the main RSS feed
func (h *RSSHandlers) HandleMainRSSFeed(c *gin.Context) {
	// Get language from query parameter or default to "en"
	languageCode := c.Query("lang")
	if languageCode == "" {
		languageCode = "en"
	}

	// Get limit from query parameter or default to 50
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	// Generate RSS feed
	xmlData, err := h.rssService.GenerateMainRSSFeed(languageCode, limit)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to generate RSS feed")
		return
	}

	// Validate the feed
	if err := h.rssService.ValidateRSSFeed(xmlData); err != nil {
		c.String(http.StatusInternalServerError, "Invalid RSS feed generated")
		return
	}

	// Set appropriate headers and return RSS feed
	c.Header("Cache-Control", "public, max-age=14400") // 4 hours
	c.Data(http.StatusOK, "application/rss+xml; charset=utf-8", xmlData)
}

// HandleCategoryRSSFeed serves RSS feed for a specific category
func (h *RSSHandlers) HandleCategoryRSSFeed(c *gin.Context) {
	categorySlug := c.Param("slug")
	
	if categorySlug == "" {
		c.String(http.StatusBadRequest, "Category slug is required")
		return
	}

	// Get language from query parameter or default to "en"
	languageCode := c.Query("lang")
	if languageCode == "" {
		languageCode = "en"
	}

	// Get limit from query parameter or default to 50
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	// Generate category RSS feed
	xmlData, err := h.rssService.GenerateCategoryRSSFeed(categorySlug, languageCode, limit)
	if err != nil {
		if err.Error() == "category not found" {
			c.String(http.StatusNotFound, "Category not found")
			return
		}
		c.String(http.StatusInternalServerError, "Failed to generate category RSS feed")
		return
	}

	// Validate the feed
	if err := h.rssService.ValidateRSSFeed(xmlData); err != nil {
		c.String(http.StatusInternalServerError, "Invalid RSS feed generated")
		return
	}

	// Set appropriate headers and return RSS feed
	c.Header("Cache-Control", "public, max-age=14400") // 4 hours
	c.Data(http.StatusOK, "application/rss+xml; charset=utf-8", xmlData)
}

// HandleTagRSSFeed serves RSS feed for a specific tag
func (h *RSSHandlers) HandleTagRSSFeed(c *gin.Context) {
	tagSlug := c.Param("slug")
	
	if tagSlug == "" {
		c.String(http.StatusBadRequest, "Tag slug is required")
		return
	}

	// Get language from query parameter or default to "en"
	languageCode := c.Query("lang")
	if languageCode == "" {
		languageCode = "en"
	}

	// Get limit from query parameter or default to 50
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	// Generate tag RSS feed
	xmlData, err := h.rssService.GenerateTagRSSFeed(tagSlug, languageCode, limit)
	if err != nil {
		if err.Error() == "tag not found" {
			c.String(http.StatusNotFound, "Tag not found")
			return
		}
		c.String(http.StatusInternalServerError, "Failed to generate tag RSS feed")
		return
	}

	// Validate the feed
	if err := h.rssService.ValidateRSSFeed(xmlData); err != nil {
		c.String(http.StatusInternalServerError, "Invalid RSS feed generated")
		return
	}

	// Set appropriate headers and return RSS feed
	c.Header("Cache-Control", "public, max-age=14400") // 4 hours
	c.Data(http.StatusOK, "application/rss+xml; charset=utf-8", xmlData)
}

// HandleGoogleNewsRSSFeed serves Google News compliant RSS feed
func (h *RSSHandlers) HandleGoogleNewsRSSFeed(c *gin.Context) {
	// Get language from query parameter or default to "en"
	languageCode := c.Query("lang")
	if languageCode == "" {
		languageCode = "en"
	}

	// Get limit from query parameter or default to 100, max 1000 for Google News
	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 1000 {
			limit = parsedLimit
		}
	}

	// Generate Google News RSS feed
	xmlData, err := h.rssService.GenerateGoogleNewsRSSFeed(languageCode, limit)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to generate Google News RSS feed")
		return
	}

	// Validate the feed
	if err := h.rssService.ValidateGoogleNewsRSSFeed(xmlData); err != nil {
		c.String(http.StatusInternalServerError, "Invalid Google News RSS feed generated")
		return
	}

	// Set appropriate headers and return RSS feed
	c.Header("Cache-Control", "public, max-age=1800") // 30 minutes for news
	c.Data(http.StatusOK, "application/rss+xml; charset=utf-8", xmlData)
}

// HandleRSSFeedRefresh handles manual RSS feed refresh (admin only)
func (h *RSSHandlers) HandleRSSFeedRefresh(c *gin.Context) {
	// This should be protected by admin authentication middleware
	if c.Request.Method != http.MethodPost {
		c.String(http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	feedType := c.PostForm("feed_type")
	identifier := c.PostForm("identifier") // category slug or tag slug
	languageCode := c.PostForm("language_code")
	
	if languageCode == "" {
		languageCode = "fa"
	}

	// Force refresh the specified feed
	if err := h.rssService.ForceRefreshFeed(feedType, identifier, languageCode); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"message": fmt.Sprintf("Failed to refresh feed: %v", err),
		})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"message": "Feed refreshed successfully",
	})
}

// HandleRSSFeedStats returns RSS feed statistics (admin only)
func (h *RSSHandlers) HandleRSSFeedStats(c *gin.Context) {
	// This should be protected by admin authentication middleware
	languageCode := c.Query("lang")
	if languageCode == "" {
		languageCode = "fa"
	}

	stats, err := h.rssService.GetFeedStats(languageCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"message": "Failed to get feed stats",
		})
		return
	}

	// Return stats as JSON
	c.JSON(http.StatusOK, stats)
}