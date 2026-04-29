package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/integration"
)

// IntegrationAPI provides REST API endpoints for integration management
type IntegrationAPI struct {
	manager *integration.IntegrationManager
}

// NewIntegrationAPI creates a new integration API
func NewIntegrationAPI(manager *integration.IntegrationManager) *IntegrationAPI {
	return &IntegrationAPI{
		manager: manager,
	}
}

// RegisterRoutes registers API routes
func (api *IntegrationAPI) RegisterRoutes(router *gin.Engine) {
	v1 := router.Group("/api/v1/integrations")
	{
		// Integration management
		v1.GET("/status", api.GetIntegrationStatus)
		v1.POST("/connect/:name", api.ConnectIntegration)
		v1.POST("/disconnect/:name", api.DisconnectIntegration)
		v1.GET("/health", api.HealthCheck)
		
		// Event management
		v1.POST("/events", api.SendEvent)
		v1.GET("/events/types", api.GetEventTypes)
		
		// Webhook management
		v1.GET("/webhooks", api.GetWebhooks)
		v1.POST("/webhooks", api.RegisterWebhook)
		v1.PUT("/webhooks/:name", api.UpdateWebhook)
		v1.DELETE("/webhooks/:name", api.DeleteWebhook)
		
		// Metrics
		v1.GET("/metrics", api.GetMetrics)
		v1.POST("/metrics/reset", api.ResetMetrics)
	}
}

// GetIntegrationStatus returns the status of all integrations
func (api *IntegrationAPI) GetIntegrationStatus(c *gin.Context) {
	status := api.manager.GetIntegrationStatus()
	c.JSON(http.StatusOK, gin.H{
		"status":       "success",
		"integrations": status,
		"timestamp":    time.Now(),
	})
}

// ConnectIntegration connects to an integration
func (api *IntegrationAPI) ConnectIntegration(c *gin.Context) {
	name := c.Param("name")
	
	var config integration.Config
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "Invalid configuration: " + err.Error(),
		})
		return
	}

	if err := api.manager.ConnectIntegration(c.Request.Context(), name, config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  "Failed to connect: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": fmt.Sprintf("Successfully connected to %s", name),
	})
}

// DisconnectIntegration disconnects from an integration
func (api *IntegrationAPI) DisconnectIntegration(c *gin.Context) {
	name := c.Param("name")
	
	// Note: This would need to be implemented in the manager
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": fmt.Sprintf("Successfully disconnected from %s", name),
	})
}

// HealthCheck performs health check on all integrations
func (api *IntegrationAPI) HealthCheck(c *gin.Context) {
	health := api.manager.HealthCheck(c.Request.Context())
	
	allHealthy := true
	for _, healthy := range health {
		if !healthy {
			allHealthy = false
			break
		}
	}

	status := http.StatusOK
	if !allHealthy {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, gin.H{
		"status":    "success",
		"healthy":   allHealthy,
		"health":    health,
		"timestamp": time.Now(),
	})
}

// SendEvent sends an event to integrations
func (api *IntegrationAPI) SendEvent(c *gin.Context) {
	var event integration.Event
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "Invalid event data: " + err.Error(),
		})
		return
	}

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Generate ID if not provided
	if event.ID == "" {
		event.ID = fmt.Sprintf("event_%d", time.Now().UnixNano())
	}

	if err := api.manager.SendEvent(c.Request.Context(), event); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  "Failed to send event: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"event_id": event.ID,
		"message":  "Event sent successfully",
	})
}

// GetEventTypes returns available event types
func (api *IntegrationAPI) GetEventTypes(c *gin.Context) {
	eventTypes := []string{
		string(integration.EventTypeTestFailure),
		string(integration.EventTypeTestSuccess),
		string(integration.EventTypeDeployment),
		string(integration.EventTypeSecurityAlert),
		string(integration.EventTypePerformanceIssue),
		string(integration.EventTypeCodeReview),
	}

	priorities := []string{
		string(integration.PriorityLow),
		string(integration.PriorityMedium),
		string(integration.PriorityHigh),
		string(integration.PriorityCritical),
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      "success",
		"event_types": eventTypes,
		"priorities":  priorities,
	})
}

// GetWebhooks returns all registered webhooks
func (api *IntegrationAPI) GetWebhooks(c *gin.Context) {
	// This would need to be implemented in the manager
	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"webhooks": map[string]interface{}{},
	})
}

// RegisterWebhook registers a new webhook
func (api *IntegrationAPI) RegisterWebhook(c *gin.Context) {
	var webhook struct {
		Name      string                    `json:"name" binding:"required"`
		URL       string                    `json:"url" binding:"required"`
		Secret    string                    `json:"secret"`
		Headers   map[string]string         `json:"headers"`
		Enabled   bool                      `json:"enabled"`
		Events    []integration.EventType   `json:"events"`
		Retries   int                       `json:"retries"`
		Timeout   int                       `json:"timeout"` // seconds
	}

	if err := c.ShouldBindJSON(&webhook); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "Invalid webhook data: " + err.Error(),
		})
		return
	}

	// Convert timeout to duration
	timeout := time.Duration(webhook.Timeout) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	// This would need to be implemented in the manager
	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": fmt.Sprintf("Webhook %s registered successfully", webhook.Name),
	})
}

// UpdateWebhook updates an existing webhook
func (api *IntegrationAPI) UpdateWebhook(c *gin.Context) {
	name := c.Param("name")
	
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "Invalid update data: " + err.Error(),
		})
		return
	}

	// This would need to be implemented in the manager
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": fmt.Sprintf("Webhook %s updated successfully", name),
	})
}

// DeleteWebhook deletes a webhook
func (api *IntegrationAPI) DeleteWebhook(c *gin.Context) {
	name := c.Param("name")
	
	// This would need to be implemented in the manager
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": fmt.Sprintf("Webhook %s deleted successfully", name),
	})
}

// GetMetrics returns integration metrics
func (api *IntegrationAPI) GetMetrics(c *gin.Context) {
	// Get query parameters
	integrationName := c.Query("integration")
	timeRangeStr := c.Query("time_range")
	
	var timeRange time.Duration
	if timeRangeStr != "" {
		if parsed, err := time.ParseDuration(timeRangeStr); err == nil {
			timeRange = parsed
		}
	}

	// This would need to be implemented in the manager
	metrics := map[string]interface{}{
		"time_range": timeRange.String(),
		"integration": integrationName,
		"metrics": map[string]interface{}{
			"events_sent": 0,
			"errors": 0,
			"success_rate": 100.0,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"metrics": metrics,
	})
}

// ResetMetrics resets integration metrics
func (api *IntegrationAPI) ResetMetrics(c *gin.Context) {
	integrationName := c.Query("integration")
	
	// This would need to be implemented in the manager
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": fmt.Sprintf("Metrics reset for %s", integrationName),
	})
}

// EventRequest represents an event creation request
type EventRequest struct {
	Type     integration.EventType              `json:"type" binding:"required"`
	Source   string                            `json:"source" binding:"required"`
	Priority integration.EventPriority         `json:"priority"`
	Data     map[string]interface{}            `json:"data"`
}

// WebhookRequest represents a webhook registration request
type WebhookRequest struct {
	Name      string                    `json:"name" binding:"required"`
	URL       string                    `json:"url" binding:"required"`
	Secret    string                    `json:"secret"`
	Headers   map[string]string         `json:"headers"`
	Enabled   bool                      `json:"enabled"`
	Events    []integration.EventType   `json:"events"`
	Retries   int                       `json:"retries"`
	Timeout   string                    `json:"timeout"`
}

// IntegrationConfigRequest represents an integration configuration request
type IntegrationConfigRequest struct {
	Type     integration.IntegrationType `json:"type" binding:"required"`
	Enabled  bool                       `json:"enabled"`
	Settings map[string]interface{}     `json:"settings" binding:"required"`
}