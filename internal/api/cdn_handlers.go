package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// CDNHandlers handles CDN-related HTTP requests
type CDNHandlers struct {
	cdnService services.CDNServiceInterface
}

// NewCDNHandlers creates a new CDN handlers instance
func NewCDNHandlers(cdnService services.CDNServiceInterface) *CDNHandlers {
	return &CDNHandlers{
		cdnService: cdnService,
	}
}

// GetCDNConfig handles GET /api/v1/cdn/config
func (h *CDNHandlers) GetCDNConfig(c *gin.Context) {
	config, err := h.cdnService.GetConfig()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Configuration not found",
			"message": err.Error(),
		})
		return
	}

	// Don't expose sensitive information
	safeConfig := map[string]interface{}{
		"id":         config.ID,
		"provider":   config.Provider,
		"domain":     config.Domain,
		"enabled":    config.Enabled,
		"created_at": config.CreatedAt,
		"updated_at": config.UpdatedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    safeConfig,
	})
}

// UpdateCDNConfig handles PUT /api/v1/cdn/config
func (h *CDNHandlers) UpdateCDNConfig(c *gin.Context) {
	var request struct {
		Provider  string `json:"provider" binding:"required"`
		APIKey    string `json:"api_key" binding:"required"`
		APISecret string `json:"api_secret"`
		ZoneID    string `json:"zone_id" binding:"required"`
		Domain    string `json:"domain" binding:"required"`
		Enabled   bool   `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	config := &models.CDNConfig{
		Provider:  request.Provider,
		APIKey:    request.APIKey,
		APISecret: request.APISecret,
		ZoneID:    request.ZoneID,
		Domain:    request.Domain,
		Enabled:   request.Enabled,
	}

	if err := h.cdnService.UpdateConfig(config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to update configuration",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "CDN configuration updated successfully",
	})
}

// TestCDNConnection handles POST /api/v1/cdn/test
func (h *CDNHandlers) TestCDNConnection(c *gin.Context) {
	if err := h.cdnService.TestConnection(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "Connection test failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "CDN connection test successful",
	})
}

// PurgeCache handles POST /api/v1/cdn/purge
func (h *CDNHandlers) PurgeCache(c *gin.Context) {
	var request models.CDNPurgeRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	response, err := h.cdnService.PurgeCache(&request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to purge cache",
			"message": err.Error(),
		})
		return
	}

	statusCode := http.StatusOK
	if !response.Success {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, gin.H{
		"success":    response.Success,
		"request_id": response.RequestID,
		"message":    response.Message,
		"timestamp":  response.Timestamp,
	})
}

// PurgeURL handles POST /api/v1/cdn/purge/url
func (h *CDNHandlers) PurgeURL(c *gin.Context) {
	var request struct {
		URL string `json:"url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	if err := h.cdnService.PurgeURL(request.URL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to purge URL",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "URL purged successfully",
		"url":     request.URL,
	})
}

// PurgeURLs handles POST /api/v1/cdn/purge/urls
func (h *CDNHandlers) PurgeURLs(c *gin.Context) {
	var request struct {
		URLs []string `json:"urls" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	if len(request.URLs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "No URLs provided",
			"message": "At least one URL is required",
		})
		return
	}

	if err := h.cdnService.PurgeURLs(request.URLs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to purge URLs",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "URLs purged successfully",
		"count":   len(request.URLs),
	})
}

// PurgeAll handles POST /api/v1/cdn/purge/all
func (h *CDNHandlers) PurgeAll(c *gin.Context) {
	if err := h.cdnService.PurgeAll(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to purge all cache",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "All cache purged successfully",
	})
}

// GetCDNStats handles GET /api/v1/cdn/stats
func (h *CDNHandlers) GetCDNStats(c *gin.Context) {
	stats, err := h.cdnService.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get CDN stats",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetCDNHealth handles GET /api/v1/cdn/health
func (h *CDNHandlers) GetCDNHealth(c *gin.Context) {
	health, err := h.cdnService.GetHealthStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get CDN health status",
			"message": err.Error(),
		})
		return
	}

	statusCode := http.StatusOK
	if health.Status == "down" {
		statusCode = http.StatusServiceUnavailable
	} else if health.Status == "degraded" {
		statusCode = http.StatusPartialContent
	}

	c.JSON(statusCode, gin.H{
		"success": true,
		"data":    health,
	})
}

// EnableFailover handles POST /api/v1/cdn/failover/enable
func (h *CDNHandlers) EnableFailover(c *gin.Context) {
	if err := h.cdnService.EnableFailover(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to enable failover",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "CDN failover enabled",
	})
}

// DisableFailover handles POST /api/v1/cdn/failover/disable
func (h *CDNHandlers) DisableFailover(c *gin.Context) {
	if err := h.cdnService.DisableFailover(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to disable failover",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "CDN failover disabled",
	})
}

// GetFailoverStatus handles GET /api/v1/cdn/failover/status
func (h *CDNHandlers) GetFailoverStatus(c *gin.Context) {
	isActive := h.cdnService.IsFailoverActive()

	c.JSON(http.StatusOK, gin.H{
		"success":         true,
		"failover_active": isActive,
		"status":          map[string]interface{}{
			"active":    isActive,
			"timestamp": time.Now(),
		},
	})
}

// PurgeArticle handles POST /api/v1/cdn/purge/article/:slug
func (h *CDNHandlers) PurgeArticle(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": "Article slug is required",
		})
		return
	}

	// Cast to concrete type to access PurgeArticleCache method
	if cdnService, ok := h.cdnService.(*services.CloudflareCDNService); ok {
		if err := cdnService.PurgeArticleCache(slug); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to purge article cache",
				"message": err.Error(),
			})
			return
		}
	} else {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error":   "Method not supported",
			"message": "Article cache purging not supported by current CDN provider",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Article cache purged successfully",
		"slug":    slug,
	})
}

// PurgeCategory handles POST /api/v1/cdn/purge/category/:slug
func (h *CDNHandlers) PurgeCategory(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": "Category slug is required",
		})
		return
	}

	// Cast to concrete type to access PurgeCategoryCache method
	if cdnService, ok := h.cdnService.(*services.CloudflareCDNService); ok {
		if err := cdnService.PurgeCategoryCache(slug); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to purge category cache",
				"message": err.Error(),
			})
			return
		}
	} else {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error":   "Method not supported",
			"message": "Category cache purging not supported by current CDN provider",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Category cache purged successfully",
		"slug":    slug,
	})
}

// PurgeTag handles POST /api/v1/cdn/purge/tag/:slug
func (h *CDNHandlers) PurgeTag(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": "Tag slug is required",
		})
		return
	}

	// Cast to concrete type to access PurgeTagCache method
	if cdnService, ok := h.cdnService.(*services.CloudflareCDNService); ok {
		if err := cdnService.PurgeTagCache(slug); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to purge tag cache",
				"message": err.Error(),
			})
			return
		}
	} else {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error":   "Method not supported",
			"message": "Tag cache purging not supported by current CDN provider",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Tag cache purged successfully",
		"slug":    slug,
	})
}