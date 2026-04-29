package services

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"high-performance-news-website/internal/config"
	"high-performance-news-website/internal/models"

	_ "github.com/lib/pq"
)

// MockDB implements basic database operations for testing
type MockDB struct {
	shouldFail bool
	stats      sql.DBStats
}

func NewMockDB() *MockDB {
	return &MockDB{
		shouldFail: false,
		stats: sql.DBStats{
			OpenConnections: 10,
			Idle:           5,
			MaxOpenConns:   150,
		},
	}
}

func (m *MockDB) PingContext(ctx context.Context) error {
	if m.shouldFail {
		return fmt.Errorf("database connection failed")
	}
	return nil
}

func (m *MockDB) Stats() sql.DBStats {
	return m.stats
}

func (m *MockDB) SetShouldFail(fail bool) {
	m.shouldFail = fail
}

func (m *MockDB) SetStats(stats sql.DBStats) {
	m.stats = stats
}

func TestNewHealthService(t *testing.T) {
	config := &config.MonitoringConfig{
		CPUThreshold:    80.0,
		MemoryThreshold: 85.0,
		DiskThreshold:   90.0,
	}
	
	cache := NewMockCacheService()
	metricsService := NewMetricsService(nil, cache, config)
	
	service := NewHealthService(nil, cache, config, metricsService)
	
	if service == nil {
		t.Fatal("Expected HealthService to be created")
	}
	
	if service.config != config {
		t.Error("Config not properly set")
	}
	
	if service.cache != cache {
		t.Error("Cache not properly set")
	}
	
	if service.metricsService != metricsService {
		t.Error("MetricsService not properly set")
	}
}

func TestPerformHealthCheck(t *testing.T) {
	config := &config.MonitoringConfig{
		CPUThreshold:          80.0,
		MemoryThreshold:       85.0,
		DiskThreshold:         90.0,
		DBConnectionThreshold: 140,
	}
	
	cache := NewMockCacheService()
	metricsService := NewMetricsService(nil, cache, config)
	
	service := NewHealthService(nil, cache, config, metricsService)
	
	// Test health check without metrics
	response := service.PerformHealthCheck(false)
	
	if response == nil {
		t.Fatal("Health response should not be nil")
	}
	
	if response.Status == "" {
		t.Error("Health status should be set")
	}
	
	if response.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
	
	if len(response.Components) == 0 {
		t.Error("Components should be checked")
	}
	
	// Test health check with metrics
	response = service.PerformHealthCheck(true)
	
	if response.Metrics == nil {
		t.Error("Metrics should be included when requested")
	}
}

func TestCheckDatabaseHealth(t *testing.T) {
	config := &config.MonitoringConfig{
		DBConnectionThreshold: 140,
	}
	
	cache := NewMockCacheService()
	metricsService := NewMetricsService(nil, cache, config)
	
	// Test with no database
	service := NewHealthService(nil, cache, config, metricsService)
	component := service.checkDatabaseHealth()
	
	if component.Status != "unhealthy" {
		t.Errorf("Expected unhealthy status with no database, got %s", component.Status)
	}
	
	if component.Message != "Database connection not available" {
		t.Errorf("Expected specific message, got %s", component.Message)
	}
	
	// Test with healthy database
	mockDB := NewMockDB()
	service = NewHealthService(mockDB, cache, config, metricsService)
	component = service.checkDatabaseHealth()
	
	if component.Status != "healthy" {
		t.Errorf("Expected healthy status with good database, got %s", component.Status)
	}
	
	if component.Details == nil {
		t.Error("Database details should be included")
	}
	
	// Test with high connection usage
	mockDB.SetStats(sql.DBStats{
		OpenConnections: 145, // Above threshold
		Idle:           5,
		MaxOpenConns:   150,
	})
	
	component = service.checkDatabaseHealth()
	
	if component.Status != "degraded" {
		t.Errorf("Expected degraded status with high connections, got %s", component.Status)
	}
	
	// Test with failing database
	mockDB.SetShouldFail(true)
	component = service.checkDatabaseHealth()
	
	if component.Status != "unhealthy" {
		t.Errorf("Expected unhealthy status with failing database, got %s", component.Status)
	}
}

func TestCheckCacheHealth(t *testing.T) {
	config := &config.MonitoringConfig{}
	metricsService := NewMetricsService(nil, nil, config)
	
	// Test with no cache
	service := NewHealthService(nil, nil, config, metricsService)
	component := service.checkCacheHealth()
	
	if component.Status != "unhealthy" {
		t.Errorf("Expected unhealthy status with no cache, got %s", component.Status)
	}
	
	// Test with healthy cache
	cache := NewMockCacheService()
	service = NewHealthService(nil, cache, config, metricsService)
	component = service.checkCacheHealth()
	
	if component.Status != "healthy" {
		t.Errorf("Expected healthy status with good cache, got %s", component.Status)
	}
}

func TestCheckSystemHealth(t *testing.T) {
	config := &config.MonitoringConfig{
		CPUThreshold:    80.0,
		MemoryThreshold: 85.0,
		DiskThreshold:   90.0,
	}
	
	cache := NewMockCacheService()
	metricsService := NewMetricsService(nil, cache, config)
	
	service := NewHealthService(nil, cache, config, metricsService)
	
	component := service.checkSystemHealth()
	
	if component.Status == "" {
		t.Error("System health status should be set")
	}
	
	if component.Details == nil {
		t.Error("System health details should be included")
	}
	
	// Check that details contain expected fields
	expectedFields := []string{"cpu_usage", "memory_usage", "disk_usage", "load_avg_1", "load_avg_5", "load_avg_15"}
	for _, field := range expectedFields {
		if _, exists := component.Details[field]; !exists {
			t.Errorf("Expected field %s in system health details", field)
		}
	}
}

func TestDetermineOverallStatus(t *testing.T) {
	config := &config.MonitoringConfig{}
	cache := NewMockCacheService()
	metricsService := NewMetricsService(nil, cache, config)
	
	service := NewHealthService(nil, cache, config, metricsService)
	
	// Test all healthy
	components := map[string]ComponentHealth{
		"database": {Status: "healthy"},
		"cache":    {Status: "healthy"},
		"system":   {Status: "healthy"},
	}
	
	status := service.determineOverallStatus(components)
	if status != "healthy" {
		t.Errorf("Expected healthy status, got %s", status)
	}
	
	// Test with degraded component
	components["database"] = ComponentHealth{Status: "degraded"}
	status = service.determineOverallStatus(components)
	if status != "degraded" {
		t.Errorf("Expected degraded status, got %s", status)
	}
	
	// Test with unhealthy component
	components["cache"] = ComponentHealth{Status: "unhealthy"}
	status = service.determineOverallStatus(components)
	if status != "unhealthy" {
		t.Errorf("Expected unhealthy status, got %s", status)
	}
}

func TestHTTPHealthHandler(t *testing.T) {
	config := &config.MonitoringConfig{
		CPUThreshold:    80.0,
		MemoryThreshold: 85.0,
		DiskThreshold:   90.0,
	}
	
	cache := NewMockCacheService()
	metricsService := NewMetricsService(nil, cache, config)
	
	service := NewHealthService(nil, cache, config, metricsService)
	
	handler := service.HTTPHealthHandler()
	
	// Test basic health check
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	
	handler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected JSON content type, got %s", contentType)
	}
	
	// Test health check with metrics
	req = httptest.NewRequest("GET", "/health?metrics=true", nil)
	w = httptest.NewRecorder()
	
	handler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestReadinessHandler(t *testing.T) {
	config := &config.MonitoringConfig{}
	cache := NewMockCacheService()
	metricsService := NewMetricsService(nil, cache, config)
	
	// Test with no database (not ready)
	service := NewHealthService(nil, cache, config, metricsService)
	handler := service.ReadinessHandler()
	
	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()
	
	handler(w, req)
	
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}
	
	// Test with healthy database (ready)
	mockDB := NewMockDB()
	service = NewHealthService(mockDB, cache, config, metricsService)
	handler = service.ReadinessHandler()
	
	req = httptest.NewRequest("GET", "/ready", nil)
	w = httptest.NewRecorder()
	
	handler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestLivenessHandler(t *testing.T) {
	config := &config.MonitoringConfig{}
	cache := NewMockCacheService()
	metricsService := NewMetricsService(nil, cache, config)
	
	service := NewHealthService(nil, cache, config, metricsService)
	handler := service.LivenessHandler()
	
	req := httptest.NewRequest("GET", "/live", nil)
	w := httptest.NewRecorder()
	
	handler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetUptime(t *testing.T) {
	config := &config.MonitoringConfig{}
	cache := NewMockCacheService()
	metricsService := NewMetricsService(nil, cache, config)
	
	service := NewHealthService(nil, cache, config, metricsService)
	
	uptime := service.getUptime()
	
	if uptime == "unknown" {
		t.Error("Uptime should not be unknown with metrics service")
	}
	
	// Test without metrics service
	service = NewHealthService(nil, cache, config, nil)
	uptime = service.getUptime()
	
	if uptime != "unknown" {
		t.Error("Uptime should be unknown without metrics service")
	}
}

func TestGetBasicMetrics(t *testing.T) {
	config := &config.MonitoringConfig{}
	cache := NewMockCacheService()
	metricsService := NewMetricsService(nil, cache, config)
	
	service := NewHealthService(nil, cache, config, metricsService)
	
	metrics := service.getBasicMetrics()
	
	if metrics == nil {
		t.Error("Basic metrics should not be nil with metrics service")
	}
	
	// Test without metrics service
	service = NewHealthService(nil, cache, config, nil)
	metrics = service.getBasicMetrics()
	
	if metrics != nil {
		t.Error("Basic metrics should be nil without metrics service")
	}
}

// Benchmark tests
func BenchmarkPerformHealthCheck(b *testing.B) {
	config := &config.MonitoringConfig{
		CPUThreshold:    80.0,
		MemoryThreshold: 85.0,
		DiskThreshold:   90.0,
	}
	
	cache := NewMockCacheService()
	metricsService := NewMetricsService(nil, cache, config)
	
	service := NewHealthService(nil, cache, config, metricsService)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.PerformHealthCheck(false)
	}
}

func BenchmarkCheckDatabaseHealth(b *testing.B) {
	config := &config.MonitoringConfig{
		DBConnectionThreshold: 140,
	}
	
	cache := NewMockCacheService()
	metricsService := NewMetricsService(nil, cache, config)
	mockDB := NewMockDB()
	
	service := NewHealthService(mockDB, cache, config, metricsService)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.checkDatabaseHealth()
	}
}

func BenchmarkCheckCacheHealth(b *testing.B) {
	config := &config.MonitoringConfig{}
	cache := NewMockCacheService()
	metricsService := NewMetricsService(nil, cache, config)
	
	service := NewHealthService(nil, cache, config, metricsService)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.checkCacheHealth()
	}
}