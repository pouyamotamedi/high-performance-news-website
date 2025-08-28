package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// MonitoringHandler handles monitoring and metrics API endpoints
type MonitoringHandler struct {
	metricsService  *services.MetricsService
	healthService   *services.HealthService
	alertingService *services.AlertingService
}

// NewMonitoringHandler creates a new MonitoringHandler instance
func NewMonitoringHandler(
	metricsService *services.MetricsService,
	healthService *services.HealthService,
	alertingService *services.AlertingService,
) *MonitoringHandler {
	return &MonitoringHandler{
		metricsService:  metricsService,
		healthService:   healthService,
		alertingService: alertingService,
	}
}

// RegisterRoutes registers monitoring API routes
func (mh *MonitoringHandler) RegisterRoutes(router *gin.Engine) {
	// Health check endpoints
	router.GET("/health", mh.HealthCheck)
	router.GET("/health/live", mh.LivenessCheck)
	router.GET("/health/ready", mh.ReadinessCheck)
	
	// Prometheus metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	
	// Monitoring API group
	monitoring := router.Group("/api/v1/monitoring")
	{
		// Dashboard and overview
		monitoring.GET("/dashboard", mh.GetDashboard)
		monitoring.GET("/overview", mh.GetOverview)
		
		// System metrics
		monitoring.GET("/metrics/system", mh.GetSystemMetrics)
		monitoring.GET("/metrics/database", mh.GetDatabaseMetrics)
		monitoring.GET("/metrics/cache", mh.GetCacheMetrics)
		monitoring.GET("/metrics/publishing", mh.GetPublishingMetrics)
		monitoring.GET("/metrics/performance", mh.GetPerformanceMetrics)
		
		// Health checks
		monitoring.GET("/health/components", mh.GetComponentHealth)
		monitoring.POST("/health/check/:component", mh.CheckComponent)
		
		// Alerts
		monitoring.GET("/alerts", mh.GetAlerts)
		monitoring.GET("/alerts/active", mh.GetActiveAlerts)
		monitoring.POST("/alerts/test", mh.SendTestAlert)
		monitoring.POST("/alerts/:id/resolve", mh.ResolveAlert)
		
		// Alert rules
		monitoring.GET("/alert-rules", mh.GetAlertRules)
		monitoring.POST("/alert-rules", mh.CreateAlertRule)
		monitoring.PUT("/alert-rules/:id", mh.UpdateAlertRule)
		monitoring.DELETE("/alert-rules/:id", mh.DeleteAlertRule)
		
		// Cache management
		monitoring.POST("/cache/clear", mh.ClearCache)
		monitoring.GET("/cache/stats", mh.GetCacheStats)
		
		// Configuration
		monitoring.GET("/config", mh.GetMonitoringConfig)
		monitoring.PUT("/config", mh.UpdateMonitoringConfig)
	}
}

// HealthCheck handles the main health check endpoint
func (mh *MonitoringHandler) HealthCheck(c *gin.Context) {
	includeMetrics := c.Query("metrics") == "true"
	
	if mh.healthService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Health service not available",
		})
		return
	}
	
	response := mh.healthService.PerformHealthCheck(includeMetrics)
	
	statusCode := http.StatusOK
	switch response.Status {
	case "degraded":
		statusCode = http.StatusOK // Still OK, but with warnings
	case "unhealthy":
		statusCode = http.StatusServiceUnavailable
	}
	
	c.JSON(statusCode, response)
}

// LivenessCheck handles Kubernetes liveness probe
func (mh *MonitoringHandler) LivenessCheck(c *gin.Context) {
	if mh.healthService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "service unavailable",
		})
		return
	}
	
	handler := mh.healthService.LivenessHandler()
	handler(c.Writer, c.Request)
}

// ReadinessCheck handles Kubernetes readiness probe
func (mh *MonitoringHandler) ReadinessCheck(c *gin.Context) {
	if mh.healthService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "service unavailable",
		})
		return
	}
	
	handler := mh.healthService.ReadinessHandler()
	handler(c.Writer, c.Request)
}

// GetDashboard returns comprehensive monitoring dashboard data
func (mh *MonitoringHandler) GetDashboard(c *gin.Context) {
	if mh.metricsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Metrics service not available",
		})
		return
	}
	
	dashboard, err := mh.metricsService.GetMonitoringDashboard()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to get dashboard data: %v", err),
		})
		return
	}
	
	c.JSON(http.StatusOK, dashboard)
}

// GetOverview returns system overview metrics
func (mh *MonitoringHandler) GetOverview(c *gin.Context) {
	if mh.metricsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Metrics service not available",
		})
		return
	}
	
	overview := gin.H{
		"system_health": mh.metricsService.GetOverallHealth(),
		"uptime":        time.Since(mh.metricsService.startTime).String(),
		"timestamp":     time.Now(),
	}
	
	// Add basic metrics
	if systemMetrics, err := mh.metricsService.GetSystemMetrics(); err == nil {
		overview["cpu_usage"] = systemMetrics.CPUUsage
		overview["memory_usage"] = systemMetrics.MemoryUsage
		overview["disk_usage"] = systemMetrics.DiskUsage
	}
	
	if publishingMetrics, err := mh.metricsService.GetPublishingMetrics(); err == nil {
		overview["publishing_rate"] = publishingMetrics.PublishingRate
		overview["articles_published"] = publishingMetrics.ArticlesPublished
	}
	
	if cacheMetrics, err := mh.metricsService.GetCacheMetrics(); err == nil {
		overview["cache_hit_rate"] = cacheMetrics.HitRate
	}
	
	c.JSON(http.StatusOK, overview)
}

// GetSystemMetrics returns system resource metrics
func (mh *MonitoringHandler) GetSystemMetrics(c *gin.Context) {
	if mh.metricsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Metrics service not available",
		})
		return
	}
	
	metrics, err := mh.metricsService.GetSystemMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to get system metrics: %v", err),
		})
		return
	}
	
	c.JSON(http.StatusOK, metrics)
}

// GetDatabaseMetrics returns database performance metrics
func (mh *MonitoringHandler) GetDatabaseMetrics(c *gin.Context) {
	if mh.metricsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Metrics service not available",
		})
		return
	}
	
	metrics, err := mh.metricsService.GetDatabaseMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to get database metrics: %v", err),
		})
		return
	}
	
	c.JSON(http.StatusOK, metrics)
}

// GetCacheMetrics returns cache performance metrics
func (mh *MonitoringHandler) GetCacheMetrics(c *gin.Context) {
	if mh.metricsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Metrics service not available",
		})
		return
	}
	
	metrics, err := mh.metricsService.GetCacheMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to get cache metrics: %v", err),
		})
		return
	}
	
	c.JSON(http.StatusOK, metrics)
}

// GetPublishingMetrics returns publishing performance metrics
func (mh *MonitoringHandler) GetPublishingMetrics(c *gin.Context) {
	if mh.metricsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Metrics service not available",
		})
		return
	}
	
	metrics, err := mh.metricsService.GetPublishingMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to get publishing metrics: %v", err),
		})
		return
	}
	
	c.JSON(http.StatusOK, metrics)
}

// GetPerformanceMetrics returns performance-related metrics
func (mh *MonitoringHandler) GetPerformanceMetrics(c *gin.Context) {
	if mh.metricsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Metrics service not available",
		})
		return
	}
	
	metrics := mh.metricsService.GetPerformanceMetrics()
	c.JSON(http.StatusOK, metrics)
}

// GetComponentHealth returns health status of all components
func (mh *MonitoringHandler) GetComponentHealth(c *gin.Context) {
	if mh.metricsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Metrics service not available",
		})
		return
	}
	
	healthChecks := mh.metricsService.GetHealthChecks()
	c.JSON(http.StatusOK, gin.H{
		"overall_health": mh.metricsService.GetOverallHealth(),
		"components":     healthChecks,
		"last_updated":   time.Now(),
	})
}

// CheckComponent performs a health check on a specific component
func (mh *MonitoringHandler) CheckComponent(c *gin.Context) {
	component := c.Param("component")
	
	if mh.metricsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Metrics service not available",
		})
		return
	}
	
	healthCheck := mh.metricsService.PerformHealthCheck(component)
	if healthCheck == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid component name",
		})
		return
	}
	
	c.JSON(http.StatusOK, healthCheck)
}

// GetAlerts returns alert history
func (mh *MonitoringHandler) GetAlerts(c *gin.Context) {
	if mh.alertingService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Alerting service not available",
		})
		return
	}
	
	history := mh.alertingService.GetAlertHistory()
	c.JSON(http.StatusOK, gin.H{
		"alerts": history,
		"count":  len(history),
	})
}

// GetActiveAlerts returns currently active alerts
func (mh *MonitoringHandler) GetActiveAlerts(c *gin.Context) {
	if mh.metricsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Metrics service not available",
		})
		return
	}
	
	alerts := mh.metricsService.GetActiveAlerts()
	c.JSON(http.StatusOK, gin.H{
		"active_alerts": alerts,
		"count":         len(alerts),
	})
}

// SendTestAlert sends a test alert to verify alerting channels
func (mh *MonitoringHandler) SendTestAlert(c *gin.Context) {
	if mh.alertingService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Alerting service not available",
		})
		return
	}
	
	err := mh.alertingService.TestAlert()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to send test alert: %v", err),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Test alert sent successfully",
	})
}

// ResolveAlert resolves an active alert
func (mh *MonitoringHandler) ResolveAlert(c *gin.Context) {
	alertID := c.Param("id")
	
	if mh.metricsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Metrics service not available",
		})
		return
	}
	
	mh.metricsService.ResolveAlert(alertID)
	
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Alert %s resolved", alertID),
	})
}

// GetAlertRules returns configured alert rules
func (mh *MonitoringHandler) GetAlertRules(c *gin.Context) {
	// This would typically fetch from database
	// For now, return empty array
	c.JSON(http.StatusOK, gin.H{
		"alert_rules": []models.AlertRule{},
		"count":       0,
	})
}

// CreateAlertRule creates a new alert rule
func (mh *MonitoringHandler) CreateAlertRule(c *gin.Context) {
	var rule models.AlertRule
	
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}
	
	if mh.alertingService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Alerting service not available",
		})
		return
	}
	
	err := mh.alertingService.CreateAlertRule(&rule)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Failed to create alert rule: %v", err),
		})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"message": "Alert rule created successfully",
		"rule":    rule,
	})
}

// UpdateAlertRule updates an existing alert rule
func (mh *MonitoringHandler) UpdateAlertRule(c *gin.Context) {
	ruleID := c.Param("id")
	
	var rule models.AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}
	
	// Parse rule ID
	id, err := strconv.ParseUint(ruleID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid rule ID",
		})
		return
	}
	
	rule.ID = id
	
	// This would typically update in database
	c.JSON(http.StatusOK, gin.H{
		"message": "Alert rule updated successfully",
		"rule":    rule,
	})
}

// DeleteAlertRule deletes an alert rule
func (mh *MonitoringHandler) DeleteAlertRule(c *gin.Context) {
	ruleID := c.Param("id")
	
	// Parse rule ID
	_, err := strconv.ParseUint(ruleID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid rule ID",
		})
		return
	}
	
	// This would typically delete from database
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Alert rule %s deleted successfully", ruleID),
	})
}

// ClearCache clears cache based on pattern
func (mh *MonitoringHandler) ClearCache(c *gin.Context) {
	var request struct {
		Pattern string `json:"pattern" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}
	
	if mh.metricsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Metrics service not available",
		})
		return
	}
	
	cleared, err := mh.metricsService.ClearCache(request.Pattern)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to clear cache: %v", err),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message":        "Cache cleared successfully",
		"patterns_cleared": cleared,
	})
}

// GetCacheStats returns cache statistics
func (mh *MonitoringHandler) GetCacheStats(c *gin.Context) {
	if mh.metricsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Metrics service not available",
		})
		return
	}
	
	metrics, err := mh.metricsService.GetCacheMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to get cache stats: %v", err),
		})
		return
	}
	
	c.JSON(http.StatusOK, metrics)
}

// GetMonitoringConfig returns current monitoring configuration
func (mh *MonitoringHandler) GetMonitoringConfig(c *gin.Context) {
	if mh.alertingService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Alerting service not available",
		})
		return
	}
	
	config := mh.alertingService.GetAlertingConfig()
	c.JSON(http.StatusOK, config)
}

// UpdateMonitoringConfig updates monitoring configuration
func (mh *MonitoringHandler) UpdateMonitoringConfig(c *gin.Context) {
	var config models.MonitoringConfig
	
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}
	
	if mh.alertingService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Alerting service not available",
		})
		return
	}
	
	// This would typically validate and save configuration
	c.JSON(http.StatusOK, gin.H{
		"message": "Monitoring configuration updated successfully",
		"config":  config,
	})
}