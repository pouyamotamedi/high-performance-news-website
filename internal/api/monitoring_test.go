package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"high-performance-news-website/internal/config"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

func setupMonitoringHandler() (*MonitoringHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	
	config := &config.MonitoringConfig{
		EnablePrometheus:     true,
		EnableHealthChecks:   true,
		EnableAlerting:      true,
		CPUThreshold:        80.0,
		MemoryThreshold:     85.0,
		DiskThreshold:       90.0,
		AlertCooldownPeriod: 5 * time.Minute,
	}
	
	cache := services.NewMockCacheService()
	metricsService := services.NewMetricsService(nil, cache, config)
	healthService := services.NewHealthService(nil, cache, config, metricsService)
	emailService := services.NewMockEmailService()
	alertingService := services.NewAlertingService(config, emailService)
	
	handler := NewMonitoringHandler(metricsService, healthService, alertingService)
	
	router := gin.New()
	handler.RegisterRoutes(router)
	
	return handler, router
}

func TestMonitoringHandler_HealthCheck(t *testing.T) {
	_, router := setupMonitoringHandler()
	
	// Test basic health check
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
	
	// Test health check with metrics
	req = httptest.NewRequest("GET", "/health?metrics=true", nil)
	w = httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMonitoringHandler_LivenessCheck(t *testing.T) {
	_, router := setupMonitoringHandler()
	
	req := httptest.NewRequest("GET", "/health/live", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMonitoringHandler_ReadinessCheck(t *testing.T) {
	_, router := setupMonitoringHandler()
	
	req := httptest.NewRequest("GET", "/health/ready", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	// Should be 503 since no database is configured
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestMonitoringHandler_GetDashboard(t *testing.T) {
	_, router := setupMonitoringHandler()
	
	req := httptest.NewRequest("GET", "/api/v1/monitoring/dashboard", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response models.MonitoringDashboard
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
}

func TestMonitoringHandler_GetOverview(t *testing.T) {
	_, router := setupMonitoringHandler()
	
	req := httptest.NewRequest("GET", "/api/v1/monitoring/overview", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	
	assert.Contains(t, response, "system_health")
	assert.Contains(t, response, "uptime")
	assert.Contains(t, response, "timestamp")
}

func TestMonitoringHandler_GetSystemMetrics(t *testing.T) {
	_, router := setupMonitoringHandler()
	
	req := httptest.NewRequest("GET", "/api/v1/monitoring/metrics/system", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response models.SystemMetrics
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	
	assert.GreaterOrEqual(t, response.CPUUsage, 0.0)
	assert.LessOrEqual(t, response.CPUUsage, 100.0)
}

func TestMonitoringHandler_GetDatabaseMetrics(t *testing.T) {
	_, router := setupMonitoringHandler()
	
	req := httptest.NewRequest("GET", "/api/v1/monitoring/metrics/database", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response models.DatabaseMetrics
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
}

func TestMonitoringHandler_GetCacheMetrics(t *testing.T) {
	_, router := setupMonitoringHandler()
	
	req := httptest.NewRequest("GET", "/api/v1/monitoring/metrics/cache", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response models.CacheMetrics
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	
	assert.GreaterOrEqual(t, response.HitRate, 0.0)
	assert.LessOrEqual(t, response.HitRate, 1.0)
}

func TestMonitoringHandler_GetPublishingMetrics(t *testing.T) {
	_, router := setupMonitoringHandler()
	
	req := httptest.NewRequest("GET", "/api/v1/monitoring/metrics/publishing", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response models.PublishingMetrics
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
}

func TestMonitoringHandler_GetPerformanceMetrics(t *testing.T) {
	_, router := setupMonitoringHandler()
	
	req := httptest.NewRequest("GET", "/api/v1/monitoring/metrics/performance", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	
	assert.Contains(t, response, "uptime_seconds")
	assert.Contains(t, response, "avg_response_time")
}

func TestMonitoringHandler_GetComponentHealth(t *testing.T) {
	_, router := setupMonitoringHandler()
	
	req := httptest.NewRequest("GET", "/api/v1/monitoring/health/components", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	
	assert.Contains(t, response, "overall_health")
	assert.Contains(t, response, "components")
}

func TestMonitoringHandler_CheckComponent(t *testing.T) {
	_, router := setupMonitoringHandler()
	
	req := httptest.NewRequest("POST", "/api/v1/monitoring/health/check/cache", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response models.HealthCheck
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	
	assert.Equal(t, "cache", response.Component)
}

func TestMonitoringHandler_GetActiveAlerts(t *testing.T) {
	_, router := setupMonitoringHandler()
	
	req := httptest.NewRequest("GET", "/api/v1/monitoring/alerts/active", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	
	assert.Contains(t, response, "active_alerts")
	assert.Contains(t, response, "count")
}

func TestMonitoringHandler_SendTestAlert(t *testing.T) {
	_, router := setupMonitoringHandler()
	
	req := httptest.NewRequest("POST", "/api/v1/monitoring/alerts/test", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	
	assert.Contains(t, response, "message")
}

func TestMonitoringHandler_ResolveAlert(t *testing.T) {
	_, router := setupMonitoringHandler()
	
	req := httptest.NewRequest("POST", "/api/v1/monitoring/alerts/test_alert/resolve", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	
	assert.Contains(t, response, "message")
}

func TestMonitoringHandler_CreateAlertRule(t *testing.T) {
	_, router := setupMonitoringHandler()
	
	rule := models.AlertRule{
		Name:        "test_rule",
		Description: "Test alert rule",
		Component:   "system",
		Metric:      "cpu_usage",
		Operator:    ">",
		Threshold:   80.0,
		Severity:    models.AlertSeverityWarning,
		Enabled:     true,
	}
	
	jsonData, _ := json.Marshal(rule)
	req := httptest.NewRequest("POST", "/api/v1/monitoring/alert-rules", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	
	assert.Contains(t, response, "message")
	assert.Contains(t, response, "rule")
}

func TestMonitoringHandler_ClearCache(t *testing.T) {
	_, router := setupMonitoringHandler()
	
	request := map[string]string{
		"pattern": "articles",
	}
	
	jsonData, _ := json.Marshal(request)
	req := httptest.NewRequest("POST", "/api/v1/monitoring/cache/clear", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	
	assert.Contains(t, response, "message")
	assert.Contains(t, response, "patterns_cleared")
}

func TestMonitoringHandler_GetCacheStats(t *testing.T) {
	_, router := setupMonitoringHandler()
	
	req := httptest.NewRequest("GET", "/api/v1/monitoring/cache/stats", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response models.CacheMetrics
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
}

func TestMonitoringHandler_GetMonitoringConfig(t *testing.T) {
	_, router := setupMonitoringHandler()
	
	req := httptest.NewRequest("GET", "/api/v1/monitoring/config", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response config.MonitoringConfig
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
}

func TestMonitoringHandler_PrometheusMetrics(t *testing.T) {
	_, router := setupMonitoringHandler()
	
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "# HELP")
}

// Benchmark tests
func BenchmarkMonitoringHandler_HealthCheck(b *testing.B) {
	_, router := setupMonitoringHandler()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkMonitoringHandler_GetSystemMetrics(b *testing.B) {
	_, router := setupMonitoringHandler()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/monitoring/metrics/system", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkMonitoringHandler_GetDashboard(b *testing.B) {
	_, router := setupMonitoringHandler()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/monitoring/dashboard", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}