package services

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"high-performance-news-website/internal/config"
	"high-performance-news-website/internal/models"

	_ "github.com/lib/pq"
)

// MockCacheService implements CacheService for testing
type MockCacheService struct {
	data map[string][]byte
}

func NewMockCacheService() *MockCacheService {
	return &MockCacheService{
		data: make(map[string][]byte),
	}
}

func (m *MockCacheService) Get(ctx context.Context, key string) ([]byte, error) {
	if value, exists := m.data[key]; exists {
		return value, nil
	}
	return nil, fmt.Errorf("key not found")
}

func (m *MockCacheService) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.data[key] = value
	return nil
}

func (m *MockCacheService) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *MockCacheService) DeletePattern(ctx context.Context, pattern string) error {
	// Simple pattern matching for testing
	for key := range m.data {
		if key == pattern || (pattern == "*" || pattern == "test:*" && key[:5] == "test:") {
			delete(m.data, key)
		}
	}
	return nil
}

func (m *MockCacheService) Exists(ctx context.Context, key string) bool {
	_, exists := m.data[key]
	return exists
}

// MockEmailService implements EmailService for testing
type MockEmailService struct {
	sentEmails []EmailRecord
}

type EmailRecord struct {
	To      string
	Subject string
	Body    string
}

func NewMockEmailService() *MockEmailService {
	return &MockEmailService{
		sentEmails: make([]EmailRecord, 0),
	}
}

func (m *MockEmailService) SendEmail(ctx context.Context, to, subject, body string) error {
	m.sentEmails = append(m.sentEmails, EmailRecord{
		To:      to,
		Subject: subject,
		Body:    body,
	})
	return nil
}

func (m *MockEmailService) GetSentEmails() []EmailRecord {
	return m.sentEmails
}

func TestNewMetricsService(t *testing.T) {
	config := &config.MonitoringConfig{
		EnablePrometheus: true,
		CPUThreshold:     80.0,
		MemoryThreshold:  85.0,
		DiskThreshold:    90.0,
	}
	
	cache := NewMockCacheService()
	
	service := NewMetricsService(nil, cache, config)
	
	if service == nil {
		t.Fatal("Expected MetricsService to be created")
	}
	
	if service.config != config {
		t.Error("Config not properly set")
	}
	
	if service.cache != cache {
		t.Error("Cache not properly set")
	}
}

func TestRecordHTTPRequest(t *testing.T) {
	config := &config.MonitoringConfig{
		EnablePrometheus: true,
	}
	
	service := NewMetricsService(nil, NewMockCacheService(), config)
	
	// Record a request
	service.RecordHTTPRequest("GET", "/api/articles", "200", 150*time.Millisecond)
	
	// Check if metric was recorded
	value, exists := service.GetMetric("http_requests_total")
	if !exists {
		t.Error("HTTP request metric not recorded")
	}
	
	if count, ok := value.(int64); !ok || count != 1 {
		t.Errorf("Expected count 1, got %v", value)
	}
	
	// Check response time
	responseTime, exists := service.GetMetric("last_response_time")
	if !exists {
		t.Error("Response time metric not recorded")
	}
	
	if duration, ok := responseTime.(int64); !ok || duration != 150 {
		t.Errorf("Expected response time 150ms, got %v", responseTime)
	}
}

func TestRecordArticlePublished(t *testing.T) {
	config := &config.MonitoringConfig{
		EnablePrometheus: true,
	}
	
	service := NewMetricsService(nil, NewMockCacheService(), config)
	
	// Record article publications
	service.RecordArticlePublished()
	service.RecordArticlePublished()
	service.RecordArticlePublished()
	
	// Check if metric was recorded
	value, exists := service.GetMetric("articles_published_total")
	if !exists {
		t.Error("Articles published metric not recorded")
	}
	
	if count, ok := value.(int64); !ok || count != 3 {
		t.Errorf("Expected count 3, got %v", value)
	}
}

func TestUpdatePublishingRate(t *testing.T) {
	config := &config.MonitoringConfig{
		EnablePrometheus: true,
	}
	
	service := NewMetricsService(nil, NewMockCacheService(), config)
	
	// Update publishing rate
	service.UpdatePublishingRate(45.5)
	
	// Check if metric was recorded
	value, exists := service.GetMetric("publishing_rate")
	if !exists {
		t.Error("Publishing rate metric not recorded")
	}
	
	if rate, ok := value.(float64); !ok || rate != 45.5 {
		t.Errorf("Expected rate 45.5, got %v", value)
	}
}

func TestUpdateCacheHitRate(t *testing.T) {
	config := &config.MonitoringConfig{
		EnablePrometheus: true,
	}
	
	service := NewMetricsService(nil, NewMockCacheService(), config)
	
	// Update cache hit rate
	service.UpdateCacheHitRate(0.92)
	
	// Check if metric was recorded
	value, exists := service.GetMetric("cache_hit_rate")
	if !exists {
		t.Error("Cache hit rate metric not recorded")
	}
	
	if rate, ok := value.(float64); !ok || rate != 0.92 {
		t.Errorf("Expected rate 0.92, got %v", value)
	}
}

func TestGetSystemMetrics(t *testing.T) {
	config := &config.MonitoringConfig{
		EnablePrometheus: true,
	}
	
	service := NewMetricsService(nil, NewMockCacheService(), config)
	
	metrics, err := service.GetSystemMetrics()
	if err != nil {
		t.Fatalf("Failed to get system metrics: %v", err)
	}
	
	if metrics == nil {
		t.Fatal("System metrics should not be nil")
	}
	
	// Check that metrics have reasonable values
	if metrics.CPUUsage < 0 || metrics.CPUUsage > 100 {
		t.Errorf("Invalid CPU usage: %f", metrics.CPUUsage)
	}
	
	if metrics.MemoryUsage < 0 || metrics.MemoryUsage > 100 {
		t.Errorf("Invalid memory usage: %f", metrics.MemoryUsage)
	}
	
	if metrics.DiskUsage < 0 || metrics.DiskUsage > 100 {
		t.Errorf("Invalid disk usage: %f", metrics.DiskUsage)
	}
	
	if metrics.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

func TestGetCacheMetrics(t *testing.T) {
	config := &config.MonitoringConfig{
		EnablePrometheus: true,
	}
	
	service := NewMetricsService(nil, NewMockCacheService(), config)
	
	metrics, err := service.GetCacheMetrics()
	if err != nil {
		t.Fatalf("Failed to get cache metrics: %v", err)
	}
	
	if metrics == nil {
		t.Fatal("Cache metrics should not be nil")
	}
	
	// Check that metrics have reasonable values
	if metrics.HitRate < 0 || metrics.HitRate > 1 {
		t.Errorf("Invalid hit rate: %f", metrics.HitRate)
	}
	
	if metrics.HitCount < 0 {
		t.Errorf("Invalid hit count: %d", metrics.HitCount)
	}
	
	if metrics.MissCount < 0 {
		t.Errorf("Invalid miss count: %d", metrics.MissCount)
	}
	
	if metrics.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

func TestPerformHealthCheck(t *testing.T) {
	config := &config.MonitoringConfig{
		CPUThreshold:    80.0,
		MemoryThreshold: 85.0,
		DiskThreshold:   90.0,
	}
	
	cache := NewMockCacheService()
	service := NewMetricsService(nil, cache, config)
	
	// Test cache health check
	healthCheck := service.PerformHealthCheck("cache")
	
	if healthCheck == nil {
		t.Fatal("Health check should not be nil")
	}
	
	if healthCheck.Component != "cache" {
		t.Errorf("Expected component 'cache', got '%s'", healthCheck.Component)
	}
	
	if healthCheck.Status != models.HealthStatusHealthy {
		t.Errorf("Expected healthy status, got %s", healthCheck.Status)
	}
	
	if healthCheck.CheckedAt.IsZero() {
		t.Error("CheckedAt should be set")
	}
	
	// Test system health checks
	components := []string{"disk", "memory", "cpu"}
	for _, component := range components {
		healthCheck := service.PerformHealthCheck(component)
		if healthCheck.Component != component {
			t.Errorf("Expected component '%s', got '%s'", component, healthCheck.Component)
		}
	}
}

func TestGetOverallHealth(t *testing.T) {
	config := &config.MonitoringConfig{
		CPUThreshold:    80.0,
		MemoryThreshold: 85.0,
		DiskThreshold:   90.0,
	}
	
	service := NewMetricsService(nil, NewMockCacheService(), config)
	
	// Initially should be unhealthy (no health checks performed)
	status := service.GetOverallHealth()
	if status != models.HealthStatusUnhealthy {
		t.Errorf("Expected unhealthy status initially, got %s", status)
	}
	
	// Perform some health checks
	service.PerformHealthCheck("cache")
	service.PerformHealthCheck("disk")
	service.PerformHealthCheck("memory")
	
	// Should now have a status
	status = service.GetOverallHealth()
	if status == models.HealthStatusUnhealthy {
		t.Error("Status should not be unhealthy after performing health checks")
	}
}

func TestAlertTriggering(t *testing.T) {
	config := &config.MonitoringConfig{
		EnableAlerting:      true,
		CPUThreshold:        80.0,
		MemoryThreshold:     85.0,
		AlertCooldownPeriod: 5 * time.Minute,
	}
	
	service := NewMetricsService(nil, NewMockCacheService(), config)
	
	// Create mock system metrics that exceed thresholds
	systemMetrics := &models.SystemMetrics{
		CPUUsage:    85.0, // Above threshold
		MemoryUsage: 90.0, // Above threshold
		DiskUsage:   70.0, // Below threshold
		CreatedAt:   time.Now(),
	}
	
	// Check system alerts
	service.checkSystemAlerts(systemMetrics)
	
	// Check that alerts were triggered
	alerts := service.GetActiveAlerts()
	if len(alerts) == 0 {
		t.Error("Expected alerts to be triggered")
	}
	
	// Check for specific alerts
	foundCPUAlert := false
	foundMemoryAlert := false
	
	for _, alert := range alerts {
		switch alert.Name {
		case "high_cpu_usage":
			foundCPUAlert = true
			if alert.Severity != models.AlertSeverityCritical {
				t.Errorf("Expected critical severity for CPU alert, got %s", alert.Severity)
			}
		case "high_memory_usage":
			foundMemoryAlert = true
			if alert.Severity != models.AlertSeverityCritical {
				t.Errorf("Expected critical severity for memory alert, got %s", alert.Severity)
			}
		}
	}
	
	if !foundCPUAlert {
		t.Error("Expected CPU alert to be triggered")
	}
	
	if !foundMemoryAlert {
		t.Error("Expected memory alert to be triggered")
	}
}

func TestAlertResolution(t *testing.T) {
	config := &config.MonitoringConfig{
		EnableAlerting:      true,
		CPUThreshold:        80.0,
		AlertCooldownPeriod: 5 * time.Minute,
	}
	
	service := NewMetricsService(nil, NewMockCacheService(), config)
	
	// Trigger an alert
	service.triggerAlert("test_alert", "Test Alert", "Test description", 
		models.AlertSeverityWarning, "test", "test_metric", 80.0, 85.0)
	
	// Check that alert is active
	alerts := service.GetActiveAlerts()
	if len(alerts) != 1 {
		t.Fatalf("Expected 1 active alert, got %d", len(alerts))
	}
	
	// Resolve the alert
	service.ResolveAlert("test_alert")
	
	// Check that alert is no longer active
	alerts = service.GetActiveAlerts()
	if len(alerts) != 0 {
		t.Errorf("Expected 0 active alerts after resolution, got %d", len(alerts))
	}
}

func TestClearCache(t *testing.T) {
	config := &config.MonitoringConfig{}
	cache := NewMockCacheService()
	service := NewMetricsService(nil, cache, config)
	
	// Add some test data to cache
	ctx := context.Background()
	cache.Set(ctx, "article:1", []byte("test"), time.Hour)
	cache.Set(ctx, "article:2", []byte("test"), time.Hour)
	cache.Set(ctx, "homepage:en", []byte("test"), time.Hour)
	cache.Set(ctx, "category:tech", []byte("test"), time.Hour)
	
	// Clear articles cache
	cleared, err := service.ClearCache("articles")
	if err != nil {
		t.Fatalf("Failed to clear cache: %v", err)
	}
	
	if len(cleared) != 1 || cleared[0] != "article:*" {
		t.Errorf("Expected to clear article:*, got %v", cleared)
	}
	
	// Clear all cache
	cleared, err = service.ClearCache("all")
	if err != nil {
		t.Fatalf("Failed to clear all cache: %v", err)
	}
	
	if len(cleared) == 0 {
		t.Error("Expected to clear multiple cache patterns")
	}
}

func TestStartMonitoring(t *testing.T) {
	config := &config.MonitoringConfig{
		EnableHealthChecks:       true,
		EnableResourceMonitoring: true,
		EnableAlerting:          true,
		HealthCheckInterval:     100 * time.Millisecond,
		ResourceCheckInterval:   100 * time.Millisecond,
		AlertCheckInterval:      100 * time.Millisecond,
	}
	
	service := NewMetricsService(nil, NewMockCacheService(), config)
	
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	
	// Start monitoring
	service.StartMonitoring(ctx)
	
	// Wait for some monitoring cycles
	time.Sleep(300 * time.Millisecond)
	
	// Check that health checks were performed
	healthChecks := service.GetHealthChecks()
	if len(healthChecks) == 0 {
		t.Error("Expected health checks to be performed")
	}
	
	// Check that metrics were collected
	allMetrics := service.GetAllMetrics()
	if len(allMetrics) == 0 {
		t.Error("Expected metrics to be collected")
	}
}

// Benchmark tests
func BenchmarkRecordHTTPRequest(b *testing.B) {
	config := &config.MonitoringConfig{
		EnablePrometheus: true,
	}
	
	service := NewMetricsService(nil, NewMockCacheService(), config)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.RecordHTTPRequest("GET", "/api/articles", "200", 150*time.Millisecond)
	}
}

func BenchmarkGetSystemMetrics(b *testing.B) {
	config := &config.MonitoringConfig{
		EnablePrometheus: true,
	}
	
	service := NewMetricsService(nil, NewMockCacheService(), config)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.GetSystemMetrics()
	}
}

func BenchmarkPerformHealthCheck(b *testing.B) {
	config := &config.MonitoringConfig{
		CPUThreshold:    80.0,
		MemoryThreshold: 85.0,
		DiskThreshold:   90.0,
	}
	
	service := NewMetricsService(nil, NewMockCacheService(), config)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.PerformHealthCheck("cache")
	}
}