package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// MonitoringHandlers handles monitoring-related HTTP requests
type MonitoringHandlers struct {
	metricsService  *services.MetricsService
	healthService   *services.HealthService
	alertingService *services.AlertingService
}

// NewMonitoringHandlers creates a new MonitoringHandlers instance
func NewMonitoringHandlers(metricsService *services.MetricsService, healthService *services.HealthService, alertingService *services.AlertingService) *MonitoringHandlers {
	return &MonitoringHandlers{
		metricsService:  metricsService,
		healthService:   healthService,
		alertingService: alertingService,
	}
}

// RegisterRoutes registers monitoring routes
func (h *MonitoringHandlers) RegisterRoutes(router *gin.Engine) {
	// Prometheus metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	
	// Health check endpoints
	health := router.Group("/health")
	{
		health.GET("/", h.GetHealth)
		health.GET("/ready", h.GetReadiness)
		health.GET("/live", h.GetLiveness)
		health.GET("/components", h.GetComponentHealth)
	}
	
	// Monitoring API endpoints
	monitoring := router.Group("/api/v1/monitoring")
	{
		monitoring.GET("/dashboard", h.GetMonitoringDashboard)
		monitoring.GET("/metrics/system", h.GetSystemMetrics)
		monitoring.GET("/metrics/database", h.GetDatabaseMetrics)
		monitoring.GET("/metrics/cache", h.GetCacheMetrics)
		monitoring.GET("/metrics/publishing", h.GetPublishingMetrics)
		monitoring.GET("/metrics/performance", h.GetPerformanceMetrics)
		monitoring.GET("/metrics/detailed", h.GetDetailedMetrics)
		
		// Alert management
		monitoring.GET("/alerts", h.GetActiveAlerts)
		monitoring.POST("/alerts/test", h.SendTestAlert)
		monitoring.DELETE("/alerts/:name", h.ResolveAlert)
		
		// Cache management
		monitoring.DELETE("/cache", h.ClearCache)
		
		// Health checks
		monitoring.POST("/health/check", h.PerformHealthCheck)
		monitoring.GET("/health/status", h.GetHealthStatus)
	}
}

// GetHealth returns the overall health status
func (h *MonitoringHandlers) GetHealth(c *gin.Context) {
	includeMetrics := c.Query("metrics") == "true"
	
	if h.healthService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Health service not available",
		})
		return
	}
	
	healthResponse := h.healthService.PerformHealthCheck(includeMetrics)
	
	statusCode := http.StatusOK
	switch healthResponse.Status {
	case "degraded":
		statusCode = http.StatusOK // Still OK, but with warnings
	case "unhealthy":
		statusCode = http.StatusServiceUnavailable
	}
	
	c.JSON(statusCode, healthResponse)
}

// GetReadiness returns the readiness status
func (h *MonitoringHandlers) GetReadiness(c *gin.Context) {
	if h.healthService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"reason": "Health service not available",
		})
		return
	}
	
	// Use the health service's readiness handler
	h.healthService.ReadinessHandler()(c.Writer, c.Request)
}

// GetLiveness returns the liveness status
func (h *MonitoringHandlers) GetLiveness(c *gin.Context) {
	if h.healthService == nil {
		c.JSON(http.StatusOK, gin.H{
			"status": "alive",
		})
		return
	}
	
	// Use the health service's liveness handler
	h.healthService.LivenessHandler()(c.Writer, c.Request)
}

// GetComponentHealth returns detailed component health information
func (h *MonitoringHandlers) GetComponentHealth(c *gin.Context) {
	if h.metricsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Metrics service not available",
		})
		return
	}
	
	healthChecks := h.metricsService.GetHealthChecks()
	overallHealth := h.metricsService.GetOverallHealth()
	
	c.JSON(http.StatusOK, gin.H{
		"overall_status": overallHealth,
		"components":     healthChecks,
		"timestamp":      time.Now(),
	})
}

// GetMonitoringDashboard returns comprehensive monitoring dashboard data
func (h *MonitoringHandlers) GetMonitoringDashboard(c *gin.Context) {
	if h.metricsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Metrics service not available",
		})
		return
	}
	
	dashboard, err := h.metricsService.GetMonitoringDashboard()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get monitoring dashboard",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, dashboard)
}

// GetSystemMetrics returns system resource metrics
func (h *MonitoringHandlers) GetSystemMetrics(c *gin.Context) {
	if h.metricsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Metrics service not available",
		})
		return
	}
	
	metrics, err := h.metricsService.GetSystemMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get system metrics",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, metrics)
}

// GetDatabaseMetrics returns database performance metrics
func (h *MonitoringHandlers) GetDatabaseMetrics(c *gin.Context) {
	if h.metricsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Metrics service not available",
		})
		return
	}
	
	metrics, err := h.metricsService.GetDatabaseMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get database metrics",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, metrics)
}

// GetCacheMetrics returns cache performance metrics
func (h *MonitoringHandlers) GetCacheMetrics(c *gin.Context) {
	if h.metricsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Metrics service not available",
		})
		return
	}
	
	metrics, err := h.metricsService.GetCacheMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get cache metrics",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, metrics)
}

// GetPublishingMetrics returns publishing performance metrics
func (h *MonitoringHandlers) GetPublishingMetrics(c *gin.Context) {
	if h.metricsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Metrics service not available",
		})
		return
	}
	
	metrics, err := h.metricsService.GetPublishingMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get publishing metrics",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, metrics)
}

// GetPerformanceMetrics returns general performance metrics
func (h *MonitoringHandlers) GetPerformanceMetrics(c *gin.Context) {
	if h.metricsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Metrics service not available",
		})
		return
	}
	
	metrics := h.metricsService.GetPerformanceMetrics()
	c.JSON(http.StatusOK, metrics)
}

// GetDetailedMetrics returns detailed performance metrics for a specific time range
func (h *MonitoringHandlers) GetDetailedMetrics(c *gin.Context) {
	if h.metricsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Metrics service not available",
		})
		return
	}
	
	timeRange := c.DefaultQuery("range", "1h")
	
	metrics, err := h.metricsService.GetDetailedPerformanceMetrics(timeRange)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to get detailed metrics",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, metrics)
}

// GetActiveAlerts returns all active alerts
func (h *MonitoringHandlers) GetActiveAlerts(c *gin.Context) {
	if h.metricsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Metrics service not available",
		})
		return
	}
	
	alerts := h.metricsService.GetActiveAlerts()
	c.JSON(http.StatusOK, gin.H{
		"alerts": alerts,
		"count":  len(alerts),
	})
}

// SendTestAlert sends a test alert
func (h *MonitoringHandlers) SendTestAlert(c *gin.Context) {
	if h.alertingService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Alerting service not available",
		})
		return
	}
	
	err := h.alertingService.TestAlert()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to send test alert",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Test alert sent successfully",
	})
}

// ResolveAlert resolves an active alert
func (h *MonitoringHandlers) ResolveAlert(c *gin.Context) {
	if h.metricsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Metrics service not available",
		})
		return
	}
	
	alertName := c.Param("name")
	if alertName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Alert name is required",
		})
		return
	}
	
	h.metricsService.ResolveAlert(alertName)
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Alert resolved successfully",
		"alert":   alertName,
	})
}

// ClearCache clears cache based on the specified type
func (h *MonitoringHandlers) ClearCache(c *gin.Context) {
	if h.metricsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Metrics service not available",
		})
		return
	}
	
	cacheType := c.DefaultQuery("type", "all")
	
	clearedCaches, err := h.metricsService.ClearCache(cacheType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to clear cache",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Cache cleared successfully",
		"cleared": clearedCaches,
	})
}

// PerformHealthCheck triggers a manual health check
func (h *MonitoringHandlers) PerformHealthCheck(c *gin.Context) {
	if h.metricsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Metrics service not available",
		})
		return
	}
	
	component := c.DefaultQuery("component", "all")
	
	if component == "all" {
		components := []string{"database", "cache", "disk", "memory", "cpu"}
		results := make(map[string]*models.HealthCheck)
		
		for _, comp := range components {
			results[comp] = h.metricsService.PerformHealthCheck(comp)
		}
		
		c.JSON(http.StatusOK, gin.H{
			"health_checks": results,
			"overall_status": h.metricsService.GetOverallHealth(),
		})
	} else {
		result := h.metricsService.PerformHealthCheck(component)
		c.JSON(http.StatusOK, result)
	}
}

// GetHealthStatus returns the current health status
func (h *MonitoringHandlers) GetHealthStatus(c *gin.Context) {
	if h.metricsService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Metrics service not available",
		})
		return
	}
	
	overallHealth := h.metricsService.GetOverallHealth()
	healthChecks := h.metricsService.GetHealthChecks()
	
	c.JSON(http.StatusOK, gin.H{
		"status": overallHealth,
		"components": healthChecks,
		"timestamp": time.Now(),
	})
}