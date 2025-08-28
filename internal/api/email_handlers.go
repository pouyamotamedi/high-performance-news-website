package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// EmailHandlers handles email-related HTTP requests
type EmailHandlers struct {
	emailService services.EmailService
}

// NewEmailHandlers creates new email handlers
func NewEmailHandlers(emailService services.EmailService) *EmailHandlers {
	return &EmailHandlers{
		emailService: emailService,
	}
}

// RegisterRoutes registers email routes
func (h *EmailHandlers) RegisterRoutes(router *gin.RouterGroup) {
	// Public routes
	public := router.Group("/email")
	{
		public.POST("/subscribe", h.Subscribe)
		public.GET("/confirm/:token", h.ConfirmSubscription)
		public.GET("/unsubscribe/:token", h.Unsubscribe)
		public.POST("/webhooks/:provider/:event", h.HandleWebhook)
	}

	// Admin routes (require authentication)
	admin := router.Group("/admin/email")
	admin.Use(AuthMiddleware(), RoleMiddleware("admin", "editor"))
	{
		// Subscriber management
		admin.GET("/subscribers", h.GetSubscribers)
		admin.GET("/subscribers/:email", h.GetSubscriber)
		admin.PUT("/subscribers/:email/preferences", h.UpdateSubscriberPreferences)

		// Campaign management
		admin.POST("/campaigns", h.CreateCampaign)
		admin.GET("/campaigns", h.GetCampaigns)
		admin.GET("/campaigns/:id", h.GetCampaign)
		admin.PUT("/campaigns/:id", h.UpdateCampaign)
		admin.DELETE("/campaigns/:id", h.DeleteCampaign)
		admin.POST("/campaigns/:id/schedule", h.ScheduleCampaign)
		admin.POST("/campaigns/:id/send", h.SendCampaign)
		admin.GET("/campaigns/:id/stats", h.GetCampaignStats)

		// Template management
		admin.POST("/templates", h.CreateTemplate)
		admin.GET("/templates", h.GetTemplates)
		admin.GET("/templates/:id", h.GetTemplate)
		admin.PUT("/templates/:id", h.UpdateTemplate)
		admin.DELETE("/templates/:id", h.DeleteTemplate)

		// Analytics
		admin.GET("/stats", h.GetEmailStats)
	}
}

// Subscribe handles email subscription requests
func (h *EmailHandlers) Subscribe(c *gin.Context) {
	var req models.SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Validate email format
	if !isValidEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid email format",
		})
		return
	}

	// Set source if not provided
	if req.Source == "" {
		req.Source = "website"
	}

	subscriber, err := h.emailService.Subscribe(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to subscribe",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Subscription successful. Please check your email to confirm.",
		"subscriber": gin.H{
			"id":     subscriber.ID,
			"email":  subscriber.Email,
			"status": subscriber.Status,
		},
	})
}

// ConfirmSubscription handles subscription confirmation
func (h *EmailHandlers) ConfirmSubscription(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Confirmation token is required",
		})
		return
	}

	err := h.emailService.ConfirmSubscription(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to confirm subscription",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Subscription confirmed successfully!",
	})
}

// Unsubscribe handles unsubscription requests
func (h *EmailHandlers) Unsubscribe(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Unsubscribe token is required",
		})
		return
	}

	err := h.emailService.Unsubscribe(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to unsubscribe",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Successfully unsubscribed from newsletter",
	})
}

// GetSubscribers handles getting subscribers with pagination
func (h *EmailHandlers) GetSubscribers(c *gin.Context) {
	status := models.SubscriberStatus(c.DefaultQuery("status", "confirmed"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100 // Max limit
	}

	subscribers, err := h.emailService.GetSubscribers(c.Request.Context(), status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get subscribers",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"subscribers": subscribers,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
			"count":  len(subscribers),
		},
	})
}

// GetSubscriber handles getting a specific subscriber
func (h *EmailHandlers) GetSubscriber(c *gin.Context) {
	email := c.Param("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Email is required",
		})
		return
	}

	subscriber, err := h.emailService.GetSubscriber(c.Request.Context(), email)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Subscriber not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"subscriber": subscriber,
	})
}

// UpdateSubscriberPreferences handles updating subscriber preferences
func (h *EmailHandlers) UpdateSubscriberPreferences(c *gin.Context) {
	email := c.Param("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Email is required",
		})
		return
	}

	var req struct {
		Preferences map[string]interface{} `json:"preferences"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	err := h.emailService.UpdateSubscriberPreferences(c.Request.Context(), email, req.Preferences)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update preferences",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Preferences updated successfully",
	})
}

// CreateCampaign handles campaign creation
func (h *EmailHandlers) CreateCampaign(c *gin.Context) {
	var req models.CreateCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	campaign, err := h.emailService.CreateCampaign(c.Request.Context(), &req, userID.(uint64))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create campaign",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"campaign": campaign,
	})
}

// GetCampaigns handles getting campaigns with pagination
func (h *EmailHandlers) GetCampaigns(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}

	campaigns, err := h.emailService.GetCampaigns(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get campaigns",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"campaigns": campaigns,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
			"count":  len(campaigns),
		},
	})
}

// GetCampaign handles getting a specific campaign
func (h *EmailHandlers) GetCampaign(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid campaign ID",
		})
		return
	}

	campaign, err := h.emailService.GetCampaign(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Campaign not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"campaign": campaign,
	})
}

// HandleWebhook handles webhooks from email providers
func (h *EmailHandlers) HandleWebhook(c *gin.Context) {
	provider := c.Param("provider")
	eventType := c.Param("event")

	if provider == "" || eventType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Provider and event type are required",
		})
		return
	}

	// Read raw body
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to read request body",
		})
		return
	}

	err = h.emailService.HandleWebhook(c.Request.Context(), provider, eventType, body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process webhook",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Webhook processed successfully",
	})
}

// GetEmailStats handles getting email statistics
func (h *EmailHandlers) GetEmailStats(c *gin.Context) {
	stats, err := h.emailService.GetEmailStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get email stats",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stats": stats,
	})
}

// GetCampaignStats handles getting campaign statistics
func (h *EmailHandlers) GetCampaignStats(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid campaign ID",
		})
		return
	}

	stats, err := h.emailService.GetCampaignStats(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get campaign stats",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stats": stats,
	})
}

// Placeholder implementations for remaining handlers
func (h *EmailHandlers) UpdateCampaign(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented"})
}

func (h *EmailHandlers) DeleteCampaign(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented"})
}

func (h *EmailHandlers) ScheduleCampaign(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented"})
}

func (h *EmailHandlers) SendCampaign(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented"})
}

func (h *EmailHandlers) CreateTemplate(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented"})
}

func (h *EmailHandlers) GetTemplates(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented"})
}

func (h *EmailHandlers) GetTemplate(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented"})
}

func (h *EmailHandlers) UpdateTemplate(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented"})
}

func (h *EmailHandlers) DeleteTemplate(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented"})
}

// Helper functions

func isValidEmail(email string) bool {
	// Basic email validation
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

// Placeholder middleware functions - these should be implemented based on your auth system
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Implement JWT token validation
		// Set user_id in context
		c.Set("user_id", uint64(1)) // Placeholder
		c.Next()
	}
}

func RoleMiddleware(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Implement role-based access control
		c.Next()
	}
}