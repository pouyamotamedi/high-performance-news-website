package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"your-project/internal/models"
	"your-project/internal/services"
)

type PushNotificationHandlers struct {
	service *services.PushNotificationService
}

func NewPushNotificationHandlers(service *services.PushNotificationService) *PushNotificationHandlers {
	return &PushNotificationHandlers{
		service: service,
	}
}

// RegisterRoutes registers all push notification routes
func (h *PushNotificationHandlers) RegisterRoutes(router *gin.RouterGroup) {
	push := router.Group("/push")
	{
		// Public endpoints
		push.POST("/subscribe", h.Subscribe)
		push.POST("/unsubscribe", h.Unsubscribe)
		push.POST("/preferences", h.UpdatePreferences)
		push.GET("/preferences/:subscription_id", h.GetPreferences)
		
		// Tracking endpoints
		push.POST("/track/delivery/:delivery_id", h.TrackDelivery)
		push.POST("/track/click/:delivery_id", h.TrackClick)
		
		// Admin endpoints (require authentication)
		admin := push.Group("/admin")
		admin.Use(RequireAuth(), RequireRole("admin", "editor"))
		{
			// Notifications
			admin.POST("/notifications", h.CreateNotification)
			admin.GET("/notifications/:id", h.GetNotification)
			admin.POST("/notifications/:id/send", h.SendNotification)
			admin.GET("/notifications", h.ListNotifications)
			
			// Templates
			admin.POST("/templates", h.CreateTemplate)
			admin.GET("/templates", h.ListTemplates)
			admin.GET("/templates/:name", h.GetTemplate)
			
			// Analytics
			admin.GET("/analytics/overview", h.GetAnalytics)
			admin.GET("/subscriptions", h.ListSubscriptions)
		}
	}
}

// Public endpoints

// Subscribe handles push notification subscription
func (h *PushNotificationHandlers) Subscribe(c *gin.Context) {
	var req struct {
		Endpoint  string  `json:"endpoint" binding:"required"`
		P256DH    string  `json:"p256dh" binding:"required"`
		Auth      string  `json:"auth" binding:"required"`
		UserID    *uint64 `json:"user_id,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	subscription := &models.PushSubscription{
		UserID:    req.UserID,
		Endpoint:  req.Endpoint,
		P256DH:    req.P256DH,
		Auth:      req.Auth,
		UserAgent: c.GetHeader("User-Agent"),
		IPAddress: c.ClientIP(),
		IsActive:  true,
	}

	err := h.service.Subscribe(subscription)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create subscription"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":        "Subscription created successfully",
		"subscription_id": subscription.ID,
	})
}

// Unsubscribe handles push notification unsubscription
func (h *PushNotificationHandlers) Unsubscribe(c *gin.Context) {
	var req struct {
		Endpoint string `json:"endpoint" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.Unsubscribe(req.Endpoint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unsubscribe"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Unsubscribed successfully"})
}

// UpdatePreferences handles notification preference updates
func (h *PushNotificationHandlers) UpdatePreferences(c *gin.Context) {
	var req struct {
		SubscriptionID      uint64   `json:"subscription_id" binding:"required"`
		UserID              *uint64  `json:"user_id,omitempty"`
		BreakingNews        bool     `json:"breaking_news"`
		CategoryUpdates     bool     `json:"category_updates"`
		TagUpdates          bool     `json:"tag_updates"`
		AuthorUpdates       bool     `json:"author_updates"`
		PreferredCategories []uint64 `json:"preferred_categories"`
		PreferredTags       []uint64 `json:"preferred_tags"`
		PreferredAuthors    []uint64 `json:"preferred_authors"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	prefs := &models.NotificationPreference{
		UserID:              req.UserID,
		SubscriptionID:      req.SubscriptionID,
		BreakingNews:        req.BreakingNews,
		CategoryUpdates:     req.CategoryUpdates,
		TagUpdates:          req.TagUpdates,
		AuthorUpdates:       req.AuthorUpdates,
		PreferredCategories: req.PreferredCategories,
		PreferredTags:       req.PreferredTags,
		PreferredAuthors:    req.PreferredAuthors,
	}

	err := h.service.UpdatePreferences(prefs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update preferences"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Preferences updated successfully"})
}

// GetPreferences retrieves notification preferences
func (h *PushNotificationHandlers) GetPreferences(c *gin.Context) {
	subscriptionID, err := strconv.ParseUint(c.Param("subscription_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscription ID"})
		return
	}

	prefs, err := h.service.GetPreferences(subscriptionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Preferences not found"})
		return
	}

	c.JSON(http.StatusOK, prefs)
}

// Tracking endpoints

// TrackDelivery tracks notification delivery
func (h *PushNotificationHandlers) TrackDelivery(c *gin.Context) {
	deliveryID, err := strconv.ParseUint(c.Param("delivery_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid delivery ID"})
		return
	}

	err = h.service.TrackDelivery(deliveryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to track delivery"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Delivery tracked successfully"})
}

// TrackClick tracks notification click
func (h *PushNotificationHandlers) TrackClick(c *gin.Context) {
	deliveryID, err := strconv.ParseUint(c.Param("delivery_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid delivery ID"})
		return
	}

	err = h.service.TrackClick(deliveryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to track click"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Click tracked successfully"})
}

// Admin endpoints

// CreateNotification creates a new push notification
func (h *PushNotificationHandlers) CreateNotification(c *gin.Context) {
	var req struct {
		Title       string                 `json:"title" binding:"required"`
		Body        string                 `json:"body" binding:"required"`
		Icon        string                 `json:"icon"`
		Badge       string                 `json:"badge"`
		Image       string                 `json:"image"`
		URL         string                 `json:"url"`
		Data        map[string]interface{} `json:"data"`
		TargetType  string                 `json:"target_type"`
		TargetValue string                 `json:"target_value"`
		ScheduledAt *time.Time             `json:"scheduled_at"`
		SendNow     bool                   `json:"send_now"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	notification := &models.PushNotification{
		Title:       req.Title,
		Body:        req.Body,
		Icon:        req.Icon,
		Badge:       req.Badge,
		Image:       req.Image,
		URL:         req.URL,
		Data:        req.Data,
		TargetType:  req.TargetType,
		TargetValue: req.TargetValue,
		ScheduledAt: req.ScheduledAt,
		Status:      models.NotificationStatusPending,
	}

	err := h.service.CreateNotification(notification)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create notification"})
		return
	}

	// Send immediately if requested
	if req.SendNow {
		err = h.service.SendNotification(notification.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Notification created but failed to send",
				"notification_id": notification.ID,
			})
			return
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":         "Notification created successfully",
		"notification_id": notification.ID,
		"sent":           req.SendNow,
	})
}

// GetNotification retrieves a notification by ID
func (h *PushNotificationHandlers) GetNotification(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	notification, err := h.service.GetNotification(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}

	c.JSON(http.StatusOK, notification)
}

// SendNotification sends a pending notification
func (h *PushNotificationHandlers) SendNotification(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	err = h.service.SendNotification(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send notification"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification sent successfully"})
}

// ListNotifications lists notifications with pagination
func (h *PushNotificationHandlers) ListNotifications(c *gin.Context) {
	// This would typically include pagination parameters
	// For now, we'll return a simple response
	c.JSON(http.StatusOK, gin.H{
		"message": "List notifications endpoint - implementation needed",
		"todo":    "Add pagination and filtering",
	})
}

// CreateTemplate creates a new notification template
func (h *PushNotificationHandlers) CreateTemplate(c *gin.Context) {
	var req struct {
		Name      string   `json:"name" binding:"required"`
		Title     string   `json:"title" binding:"required"`
		Body      string   `json:"body" binding:"required"`
		Icon      string   `json:"icon"`
		Badge     string   `json:"badge"`
		Image     string   `json:"image"`
		URL       string   `json:"url"`
		Variables []string `json:"variables"`
		IsActive  bool     `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	template := &models.PushTemplate{
		Name:      req.Name,
		Title:     req.Title,
		Body:      req.Body,
		Icon:      req.Icon,
		Badge:     req.Badge,
		Image:     req.Image,
		URL:       req.URL,
		Variables: req.Variables,
		IsActive:  req.IsActive,
	}

	err := h.service.CreateTemplate(template)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create template"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Template created successfully",
		"template_id": template.ID,
	})
}

// ListTemplates lists all active templates
func (h *PushNotificationHandlers) ListTemplates(c *gin.Context) {
	templates, err := h.service.GetActiveTemplates()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve templates"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"templates": templates,
		"count":     len(templates),
	})
}

// GetTemplate retrieves a template by name
func (h *PushNotificationHandlers) GetTemplate(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Template name is required"})
		return
	}

	template, err := h.service.GetTemplate(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}

	c.JSON(http.StatusOK, template)
}

// GetAnalytics provides push notification analytics
func (h *PushNotificationHandlers) GetAnalytics(c *gin.Context) {
	// This would typically query analytics data
	// For now, we'll return a placeholder response
	c.JSON(http.StatusOK, gin.H{
		"message": "Analytics endpoint - implementation needed",
		"todo":    "Add analytics queries and metrics",
		"metrics": gin.H{
			"total_subscriptions": 0,
			"active_subscriptions": 0,
			"notifications_sent_today": 0,
			"delivery_rate": 0.0,
			"click_rate": 0.0,
		},
	})
}

// ListSubscriptions lists push subscriptions for admin
func (h *PushNotificationHandlers) ListSubscriptions(c *gin.Context) {
	// This would typically include pagination and filtering
	// For now, we'll return a placeholder response
	c.JSON(http.StatusOK, gin.H{
		"message": "List subscriptions endpoint - implementation needed",
		"todo":    "Add pagination, filtering, and subscription details",
	})
}

// Helper method to get notification by ID (used internally)
func (h *PushNotificationHandlers) getNotificationByID(id uint64) (*models.PushNotification, error) {
	// This would be implemented in the service layer
	// For now, we'll use a placeholder
	return nil, fmt.Errorf("not implemented")
}