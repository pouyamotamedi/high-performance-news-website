package services

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"high-performance-news-website/internal/config"
	"high-performance-news-website/pkg/cache"
)

// HealthService handles health checks and system status monitoring
type HealthService struct {
	db           *sql.DB
	cache        cache.CacheService
	config       *config.MonitoringConfig
	metricsService *MetricsService
}

// NewHealthService creates a new HealthService instance
func NewHealthService(db *sql.DB, cacheService cache.CacheService, config *config.MonitoringConfig, metricsService *MetricsService) *HealthService {
	return &HealthService{
		db:             db,
		cache:          cacheService,
		config:         config,
		metricsService: metricsService,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status     string                        `json:"status"`
	Timestamp  time.Time                     `json:"timestamp"`
	Uptime     string                        `json:"uptime"`
	Version    string                        `json:"version"`
	Components map[string]ComponentHealth    `json:"components"`
	Metrics    map[string]interface{}        `json:"metrics,omitempty"`
}

// ComponentHealth represents the health status of a component
type ComponentHealth struct {
	Status      string                 `json:"status"`
	Message     string                 `json:"message"`
	LastChecked time.Time              `json:"last_checked"`
	ResponseTime string                `json:"response_time"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// PerformHealthCheck performs a comprehensive health check
func (hs *HealthService) PerformHealthCheck(includeMetrics bool) *HealthResponse {
	components := make(map[string]ComponentHealth)
	
	// Check database health
	components["database"] = hs.checkDatabaseHealth()
	
	// Check cache health
	components["cache"] = hs.checkCacheHealth()
	
	// Check system resources
	components["system"] = hs.checkSystemHealth()
	
	// Determine overall status
	overallStatus := hs.determineOverallStatus(components)
	
	response := &HealthResponse{
		Status:     overallStatus,
		Timestamp:  time.Now(),
		Uptime:     hs.getUptime(),
		Version:    "1.0.0", // This would come from build info
		Components: components,
	}
	
	// Include metrics if requested
	if includeMetrics && hs.metricsService != nil {
		response.Metrics = hs.getBasicMetrics()
	}
	
	return response
}

// checkDatabaseHealth checks database connectivity and performance
func (hs *HealthService) checkDatabaseHealth() ComponentHealth {
	start := time.Now()
	
	if hs.db == nil {
		return ComponentHealth{
			Status:       "unhealthy",
			Message:      "Database connection not available",
			LastChecked:  time.Now(),
			ResponseTime: time.Since(start).String(),
		}
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err := hs.db.PingContext(ctx)
	responseTime := time.Since(start)
	
	if err != nil {
		return ComponentHealth{
			Status:       "unhealthy",
			Message:      fmt.Sprintf("Database ping failed: %v", err),
			LastChecked:  time.Now(),
			ResponseTime: responseTime.String(),
		}
	}
	
	// Get database stats
	stats := hs.db.Stats()
	details := map[string]interface{}{
		"open_connections": stats.OpenConnections,
		"idle_connections": stats.Idle,
		"max_connections":  stats.MaxOpenConnections,
		"wait_count":       stats.WaitCount,
		"wait_duration":    stats.WaitDuration.String(),
	}
	
	status := "healthy"
	message := "Database is healthy"
	
	// Check if connection usage is high
	if hs.config != nil && stats.OpenConnections >= hs.config.DBConnectionThreshold {
		status = "degraded"
		message = fmt.Sprintf("High database connection usage: %d/%d", stats.OpenConnections, stats.MaxOpenConnections)
	}
	
	return ComponentHealth{
		Status:       status,
		Message:      message,
		LastChecked:  time.Now(),
		ResponseTime: responseTime.String(),
		Details:      details,
	}
}

// checkCacheHealth checks cache connectivity and performance
func (hs *HealthService) checkCacheHealth() ComponentHealth {
	start := time.Now()
	
	if hs.cache == nil {
		return ComponentHealth{
			Status:       "unhealthy",
			Message:      "Cache service not available",
			LastChecked:  time.Now(),
			ResponseTime: time.Since(start).String(),
		}
	}
	
	// Perform a simple cache test
	testKey := "health_check_test"
	testValue := []byte("test_value")
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	// Test write
	err := hs.cache.Set(ctx, testKey, testValue, time.Minute)
	if err != nil {
		return ComponentHealth{
			Status:       "unhealthy",
			Message:      fmt.Sprintf("Cache write failed: %v", err),
			LastChecked:  time.Now(),
			ResponseTime: time.Since(start).String(),
		}
	}
	
	// Test read
	_, err = hs.cache.Get(ctx, testKey)
	if err != nil {
		return ComponentHealth{
			Status:       "degraded",
			Message:      fmt.Sprintf("Cache read failed: %v", err),
			LastChecked:  time.Now(),
			ResponseTime: time.Since(start).String(),
		}
	}
	
	// Clean up test key
	hs.cache.Delete(ctx, testKey)
	
	responseTime := time.Since(start)
	
	return ComponentHealth{
		Status:       "healthy",
		Message:      "Cache is healthy",
		LastChecked:  time.Now(),
		ResponseTime: responseTime.String(),
	}
}

// checkSystemHealth checks system resource health
func (hs *HealthService) checkSystemHealth() ComponentHealth {
	start := time.Now()
	
	if hs.metricsService == nil {
		return ComponentHealth{
			Status:       "unknown",
			Message:      "Metrics service not available",
			LastChecked:  time.Now(),
			ResponseTime: time.Since(start).String(),
		}
	}
	
	systemMetrics, err := hs.metricsService.GetSystemMetrics()
	if err != nil {
		return ComponentHealth{
			Status:       "unhealthy",
			Message:      fmt.Sprintf("Failed to get system metrics: %v", err),
			LastChecked:  time.Now(),
			ResponseTime: time.Since(start).String(),
		}
	}
	
	details := map[string]interface{}{
		"cpu_usage":    fmt.Sprintf("%.2f%%", systemMetrics.CPUUsage),
		"memory_usage": fmt.Sprintf("%.2f%%", systemMetrics.MemoryUsage),
		"disk_usage":   fmt.Sprintf("%.2f%%", systemMetrics.DiskUsage),
		"load_avg_1":   systemMetrics.LoadAverage1,
		"load_avg_5":   systemMetrics.LoadAverage5,
		"load_avg_15":  systemMetrics.LoadAverage15,
	}
	
	status := "healthy"
	message := "System resources are healthy"
	
	// Check thresholds
	if hs.config != nil {
		if systemMetrics.CPUUsage >= hs.config.CPUThreshold ||
		   systemMetrics.MemoryUsage >= hs.config.MemoryThreshold ||
		   systemMetrics.DiskUsage >= hs.config.DiskThreshold {
			status = "degraded"
			message = "System resources are under high load"
		}
	}
	
	responseTime := time.Since(start)
	
	return ComponentHealth{
		Status:       status,
		Message:      message,
		LastChecked:  time.Now(),
		ResponseTime: responseTime.String(),
		Details:      details,
	}
}

// determineOverallStatus determines the overall health status based on component statuses
func (hs *HealthService) determineOverallStatus(components map[string]ComponentHealth) string {
	hasUnhealthy := false
	hasDegraded := false
	
	for _, component := range components {
		switch component.Status {
		case "unhealthy":
			hasUnhealthy = true
		case "degraded":
			hasDegraded = true
		}
	}
	
	if hasUnhealthy {
		return "unhealthy"
	} else if hasDegraded {
		return "degraded"
	}
	
	return "healthy"
}

// getUptime returns the system uptime
func (hs *HealthService) getUptime() string {
	if hs.metricsService != nil {
		uptime := time.Since(hs.metricsService.startTime)
		return formatDuration(uptime)
	}
	return "unknown"
}

// getBasicMetrics returns basic system metrics
func (hs *HealthService) getBasicMetrics() map[string]interface{} {
	if hs.metricsService == nil {
		return nil
	}
	
	metrics := make(map[string]interface{})
	
	// Get publishing rate
	if rate, err := hs.metricsService.GetPublishingRate(); err == nil {
		metrics["publishing_rate"] = rate
	}
	
	// Get articles published today
	if count, err := hs.metricsService.GetArticlesPublishedToday(); err == nil {
		metrics["articles_today"] = count
	}
	
	// Get total articles
	if count, err := hs.metricsService.GetTotalArticles(); err == nil {
		metrics["total_articles"] = count
	}
	
	// Get active users (last 24 hours)
	if count, err := hs.metricsService.GetActiveUsersCount(24 * time.Hour); err == nil {
		metrics["active_users_24h"] = count
	}
	
	return metrics
}

// HTTPHealthHandler returns an HTTP handler for health checks
func (hs *HealthService) HTTPHealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		includeMetrics := r.URL.Query().Get("metrics") == "true"
		
		healthResponse := hs.PerformHealthCheck(includeMetrics)
		
		// Set appropriate HTTP status code
		statusCode := http.StatusOK
		switch healthResponse.Status {
		case "degraded":
			statusCode = http.StatusOK // Still OK, but with warnings
		case "unhealthy":
			statusCode = http.StatusServiceUnavailable
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		
		// Write JSON response (simplified - would use proper JSON marshaling)
		fmt.Fprintf(w, `{
			"status": "%s",
			"timestamp": "%s",
			"uptime": "%s",
			"version": "%s"
		}`, healthResponse.Status, healthResponse.Timestamp.Format(time.RFC3339), 
		healthResponse.Uptime, healthResponse.Version)
	}
}

// ReadinessHandler returns an HTTP handler for readiness checks
func (hs *HealthService) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if essential services are ready
		ready := true
		
		// Check database
		if hs.db != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := hs.db.PingContext(ctx); err != nil {
				ready = false
			}
		}
		
		if ready {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"status": "ready"}`)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, `{"status": "not ready"}`)
		}
	}
}

// LivenessHandler returns an HTTP handler for liveness checks
func (hs *HealthService) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Simple liveness check - if we can respond, we're alive
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status": "alive"}`)
	}
}