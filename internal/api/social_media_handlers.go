package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// SocialMediaHandlers handles social media related HTTP requests
type SocialMediaHandlers struct {
	socialService *services.SocialMediaService
}

// NewSocialMediaHandlers creates new social media handlers
func NewSocialMediaHandlers(socialService *services.SocialMediaService) *SocialMediaHandlers {
	return &SocialMediaHandlers{
		socialService: socialService,
	}
}

// CreateCredentials creates new social media credentials
// POST /api/admin/social-media/credentials
func (h *SocialMediaHandlers) CreateCredentials(c *gin.Context) {
	var req struct {
		Platform    models.SocialMediaPlatform `json:"platform" binding:"required"`
		Name        string                     `json:"name" binding:"required"`
		Credentials string                     `json:"credentials" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	creds := &models.SocialMediaCredentials{
		Platform: req.Platform,
		Name:     req.Name,
		Credentials: models.EncryptedData{
			Data: req.Credentials, // Will be encrypted by service
		},
		IsActive: true,
	}

	if err := h.socialService.CreateCredentials(creds); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Don't return the actual credentials in response
	response := map[string]interface{}{
		"id":       creds.ID,
		"platform": creds.Platform,
		"name":     creds.Name,
		"is_active": creds.IsActive,
		"created_at": creds.CreatedAt,
	}

	c.JSON(http.StatusCreated, response)
}

// GetCredentials retrieves social media credentials
// GET /api/admin/social-media/credentials/:id
func (h *SocialMediaHandlers) GetCredentials(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid credential ID"})
		return
	}

	creds, err := h.socialService.GetCredentials(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Credentials not found"})
		return
	}

	// Don't return the actual credentials in response
	response := map[string]interface{}{
		"id":           creds.ID,
		"platform":     creds.Platform,
		"name":         creds.Name,
		"is_active":    creds.IsActive,
		"last_rotated": creds.LastRotated,
		"created_at":   creds.CreatedAt,
		"updated_at":   creds.UpdatedAt,
	}

	c.JSON(http.StatusOK, response)
}

// RotateCredentials rotates social media credentials
// PUT /api/admin/social-media/credentials/:id/rotate
func (h *SocialMediaHandlers) RotateCredentials(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid credential ID"})
		return
	}

	var req struct {
		Credentials string `json:"credentials" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.socialService.RotateCredentials(id, req.Credentials); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Credentials rotated successfully"})
}

// PublishToSocialMedia publishes an article to social media platforms
// POST /api/admin/articles/:id/publish-social
func (h *SocialMediaHandlers) PublishToSocialMedia(c *gin.Context) {
	articleIDStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})
		return
	}

	var req struct {
		Platforms []models.SocialMediaPlatform `json:"platforms" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.socialService.PublishToSocialMedia(articleID, req.Platforms); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Article scheduled for social media publishing"})
}

// SchedulePost schedules a post for future publishing
// POST /api/admin/articles/:id/schedule-social
func (h *SocialMediaHandlers) SchedulePost(c *gin.Context) {
	articleIDStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})
		return
	}

	var req struct {
		Platforms   []models.SocialMediaPlatform `json:"platforms" binding:"required"`
		ScheduledAt time.Time                    `json:"scheduled_at" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.socialService.SchedulePost(articleID, req.Platforms, req.ScheduledAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Posts scheduled successfully"})
}

// GetPostStatus retrieves the status of a social media post
// GET /api/admin/social-media/posts/:id
func (h *SocialMediaHandlers) GetPostStatus(c *gin.Context) {
	postIDStr := c.Param("id")
	postID, err := strconv.ParseUint(postIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	post, err := h.socialService.GetPostStatus(postID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	c.JSON(http.StatusOK, post)
}

// CreateFacebookInstantArticle creates a Facebook Instant Article
// POST /api/admin/articles/:id/facebook-instant
func (h *SocialMediaHandlers) CreateFacebookInstantArticle(c *gin.Context) {
	articleIDStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})
		return
	}

	if err := h.socialService.CreateFacebookInstantArticle(articleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Facebook Instant Article created successfully"})
}

// HandleWebhook handles incoming webhooks from social media platforms
// POST /api/webhooks/social-media/:platform
func (h *SocialMediaHandlers) HandleWebhook(c *gin.Context) {
	platform := models.SocialMediaPlatform(c.Param("platform"))
	
	// Get raw body
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Get signature from headers (platform-specific)
	var signature string
	switch platform {
	case models.PlatformFacebook:
		signature = c.GetHeader("X-Hub-Signature-256")
	case models.PlatformTelegram:
		signature = c.GetHeader("X-Telegram-Bot-Api-Secret-Token")
	case models.PlatformTwitter:
		signature = c.GetHeader("X-Twitter-Webhooks-Signature")
	}

	if err := h.socialService.HandleWebhook(platform, body, signature); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Webhook processed successfully"})
}

// PublishToAllPlatforms publishes an article to all configured platforms
// POST /api/admin/articles/:id/publish-all-social
func (h *SocialMediaHandlers) PublishToAllPlatforms(c *gin.Context) {
	articleIDStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid article ID"})
		return
	}

	if err := h.socialService.PublishArticleToAllPlatforms(articleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Article scheduled for publishing to all platforms"})
}

// RegisterSocialMediaRoutes registers all social media routes
func RegisterSocialMediaRoutes(router *gin.RouterGroup, handlers *SocialMediaHandlers, authMiddleware gin.HandlerFunc) {
	// Admin routes (require authentication)
	admin := router.Group("/admin", authMiddleware)
	{
		// Credentials management
		admin.POST("/social-media/credentials", handlers.CreateCredentials)
		admin.GET("/social-media/credentials/:id", handlers.GetCredentials)
		admin.PUT("/social-media/credentials/:id/rotate", handlers.RotateCredentials)

		// Post management
		admin.POST("/articles/:id/publish-social", handlers.PublishToSocialMedia)
		admin.POST("/articles/:id/schedule-social", handlers.SchedulePost)
		admin.POST("/articles/:id/publish-all-social", handlers.PublishToAllPlatforms)
		admin.POST("/articles/:id/facebook-instant", handlers.CreateFacebookInstantArticle)
		admin.GET("/social-media/posts/:id", handlers.GetPostStatus)
	}

	// Public webhook routes (no authentication required)
	router.POST("/webhooks/social-media/:platform", handlers.HandleWebhook)
}