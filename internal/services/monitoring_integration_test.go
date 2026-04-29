package services

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"high-performance-news-website/internal/config"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/pkg/cache"

	_ "github.com/lib/pq"
)

// TestMonitoringSystemIntegration tests the complete monitoring system
func TestMonitoringSystemIntegration(t *testing.T) {
	// Skip if no database connection available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database connection
	db, err := setupTestDatabase()
	if err != nil {
		t.Skipf("Skipping test due to database setup error: %v", err)
	}
	defer db.Close()

	// Setup test cache
	cacheService := &mockCacheService{}

	// Setup monitoring config
	config := &config.MonitoringConfig{
		EnablePrometheus:           true,
		EnableHealthChecks:         true,
		HealthCheckInterval:        5 * time.Second,
		EnableResourceMonitoring:   true,
		ResourceCheckInterval:      10 * time.Second,
		CPUThreshold:              80.0,
		MemoryThreshold:           85.0,
		DiskThreshold:             90.0,
		EnableDBMonitoring:        true,
		DBConnectionThreshold:     140,
		SlowQueryThreshold:        1000 * time.Millisecond,
		EnableCacheMonitoring:     true,
		CacheHitRateThreshold:     0.8,
		EnablePublishingMonitoring: true,
		PublishingRateThreshold:   35.0,
		EnableAlerting:            true,
		AlertCheckInterval:        30 * time.Second,
		AlertCooldownPeriod:       15 * time.Minute,
	}

	// Create services
	metricsService := NewMetricsService(db, cacheService, config)
	healthService := NewHealthService(db, cacheService, config, metricsService)
	alertingService := NewAlertingService(config, nil)

	// Test metrics collection
	t.Run("MetricsCollection", func(t *testing.T) {
		testMetricsCollection(t, metricsService)
	})

	// Test health checks
	t.Run("HealthChecks", func(t *testing.T) {
		testHealthChecks(t, healthService)
	})

	// Test alerting
	t.Run("Alerting", func(t *testing.T) {
		testAlerting(t, alertingService)
	})

	// Test monitoring dashboard
	t.Run("MonitoringDashboard", func(t *testing.T) {
		testMonitoringDashboard(t, metricsService)
	})

	// Test persistence
	t.Run("Persistence", func(t *testing.T) {
		testMonitoringPersistence(t, db)
	})
}

func testMetricsCollection(t *testing.T, metricsService *MetricsService) {
	// Test system metrics collection
	systemMetrics, err := metricsService.GetSystemMetrics()
	if err != nil {
		t.Errorf("Failed to get system metrics: %v", err)
	}

	if systemMetrics.CPUUsage < 0 || systemMetrics.CPUUsage > 100 {
		t.Errorf("Invalid CPU usage: %f", systemMetrics.CPUUsage)
	}

	if systemMetrics.MemoryUsage < 0 || systemMetrics.MemoryUsage > 100 {
		t.Errorf("Invalid memory usage: %f", systemMetrics.MemoryUsage)
	}

	// Test database metrics collection
	dbMetrics, err := metricsService.GetDatabaseMetrics()
	if err != nil {
		t.Errorf("Failed to get database metrics: %v", err)
	}

	if dbMetrics.ActiveConnections < 0 {
		t.Errorf("Invalid active connections: %d", dbMetrics.ActiveConnections)
	}

	// Test cache metrics collection
	cacheMetrics, err := metricsService.GetCacheMetrics()
	if err != nil {
		t.Errorf("Failed to get cache metrics: %v", err)
	}

	if cacheMetrics.HitRate < 0 || cacheMetrics.HitRate > 1 {
		t.Errorf("Invalid cache hit rate: %f", cacheMetrics.HitRate)
	}

	// Test publishing metrics collection
	publishingMetrics, err := metricsService.GetPublishingMetrics()
	if err != nil {
		t.Errorf("Failed to get publishing metrics: %v", err)
	}

	if publishingMetrics.PublishingRate < 0 {
		t.Errorf("Invalid publishing rate: %f", publishingMetrics.PublishingRate)
	}
}

func testHealthChecks(t *testing.T, healthService *HealthService) {
	// Test comprehensive health check
	healthResponse := healthService.PerformHealthCheck(true)

	if healthResponse.Status == "" {
		t.Error("Health check status is empty")
	}

	if len(healthResponse.Components) == 0 {
		t.Error("No health check components found")
	}

	// Verify required components are checked
	requiredComponents := []string{"database", "system"}
	for _, component := range requiredComponents {
		if _, exists := healthResponse.Components[component]; !exists {
			t.Errorf("Required health check component missing: %s", component)
		}
	}

	// Test individual component health checks
	for component, health := range healthResponse.Components {
		if health.Status == "" {
			t.Errorf("Component %s has empty status", component)
		}

		if health.LastChecked.IsZero() {
			t.Errorf("Component %s has zero last checked time", component)
		}
	}
}

func testAlerting(t *testing.T, alertingService *AlertingService) {
	// Create test alert
	testAlert := &models.Alert{
		Name:         "test_alert",
		Description:  "Test alert for monitoring system",
		Severity:     models.AlertSeverityInfo,
		Status:       models.AlertStatusActive,
		Component:    "test",
		Metric:       "test_metric",
		Threshold:    100.0,
		CurrentValue: 150.0,
		Metadata:     make(map[string]interface{}),
		TriggeredAt:  time.Now(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Test alert sending (without actual external services)
	err := alertingService.SendAlert(testAlert)
	if err != nil {
		t.Logf("Alert sending failed (expected without configured services): %v", err)
	}

	// Test alert rule creation
	testRule := &models.AlertRule{
		Name:        "test_rule",
		Description: "Test alert rule",
		Component:   "system",
		Metric:      "cpu_usage",
		Operator:    ">",
		Threshold:   80.0,
		Severity:    models.AlertSeverityWarning,
		Enabled:     true,
		Cooldown:    15 * time.Minute,
		Conditions:  make(map[string]interface{}),
		Actions:     models.AlertActions{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err = alertingService.CreateAlertRule(testRule)
	if err != nil {
		t.Errorf("Failed to create alert rule: %v", err)
	}

	// Test alert rule evaluation
	if !alertingService.EvaluateAlertRule(testRule, 85.0) {
		t.Error("Alert rule should trigger for value 85.0 with threshold 80.0")
	}

	if alertingService.EvaluateAlertRule(testRule, 75.0) {
		t.Error("Alert rule should not trigger for value 75.0 with threshold 80.0")
	}
}

func testMonitoringDashboard(t *testing.T, metricsService *MetricsService) {
	// Test dashboard data generation
	dashboard, err := metricsService.GetMonitoringDashboard()
	if err != nil {
		t.Errorf("Failed to get monitoring dashboard: %v", err)
	}

	if dashboard.LastUpdated.IsZero() {
		t.Error("Dashboard last updated time is zero")
	}

	// Test overall health calculation
	health := metricsService.GetOverallHealth()
	validHealthStatuses := []string{"healthy", "degraded", "unhealthy"}
	isValidHealth := false
	for _, validStatus := range validHealthStatuses {
		if health == validStatus {
			isValidHealth = true
			break
		}
	}
	if !isValidHealth {
		t.Errorf("Invalid overall health status: %s", health)
	}

	// Test active alerts retrieval
	alerts := metricsService.GetActiveAlerts()
	if alerts == nil {
		t.Error("Active alerts should not be nil")
	}

	// Test health checks retrieval
	healthChecks := metricsService.GetHealthChecks()
	if healthChecks == nil {
		t.Error("Health checks should not be nil")
	}
}

func testMonitoringPersistence(t *testing.T, db *sql.DB) {
	persistence := NewMonitoringPersistenceService(db)
	ctx := context.Background()

	// Test health check persistence
	healthCheck := &models.HealthCheck{
		Component:    "test_component",
		Status:       models.HealthStatusHealthy,
		Message:      "Test health check",
		ResponseTime: 100 * time.Millisecond,
		Metadata:     map[string]interface{}{"test": "data"},
		CheckedAt:    time.Now(),
	}

	err := persistence.SaveHealthCheck(ctx, healthCheck)
	if err != nil {
		t.Errorf("Failed to save health check: %v", err)
	}

	// Test system metrics persistence
	systemMetrics := &models.SystemMetrics{
		CPUUsage:      45.5,
		MemoryUsage:   67.8,
		MemoryTotal:   16 * 1024 * 1024 * 1024, // 16GB
		MemoryUsed:    11 * 1024 * 1024 * 1024, // 11GB
		DiskUsage:     23.4,
		DiskTotal:     1024 * 1024 * 1024 * 1024, // 1TB
		DiskUsed:      234 * 1024 * 1024 * 1024,  // 234GB
		LoadAverage1:  1.2,
		LoadAverage5:  1.5,
		LoadAverage15: 1.8,
		CreatedAt:     time.Now(),
	}

	err = persistence.SaveSystemMetrics(ctx, systemMetrics)
	if err != nil {
		t.Errorf("Failed to save system metrics: %v", err)
	}

	// Test alert persistence
	alert := &models.Alert{
		Name:         "test_persistence_alert",
		Description:  "Test alert for persistence",
		Severity:     models.AlertSeverityWarning,
		Status:       models.AlertStatusActive,
		Component:    "test",
		Metric:       "test_metric",
		Threshold:    100.0,
		CurrentValue: 120.0,
		Metadata:     map[string]interface{}{"test": "alert"},
		TriggeredAt:  time.Now(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err = persistence.SaveAlert(ctx, alert)
	if err != nil {
		t.Errorf("Failed to save alert: %v", err)
	}

	// Test alert retrieval
	alerts, err := persistence.GetActiveAlerts(ctx)
	if err != nil {
		t.Errorf("Failed to get active alerts: %v", err)
	}

	found := false
	for _, retrievedAlert := range alerts {
		if retrievedAlert.Name == alert.Name {
			found = true
			break
		}
	}
	if !found {
		t.Error("Saved alert not found in active alerts")
	}

	// Test alert rule persistence
	alertRule := &models.AlertRule{
		Name:        "test_persistence_rule",
		Description: "Test rule for persistence",
		Component:   "system",
		Metric:      "cpu_usage",
		Operator:    ">",
		Threshold:   80.0,
		Severity:    models.AlertSeverityWarning,
		Enabled:     true,
		Cooldown:    15 * time.Minute,
		Conditions:  map[string]interface{}{"test": "condition"},
		Actions:     models.AlertActions{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err = persistence.SaveAlertRule(ctx, alertRule)
	if err != nil {
		t.Errorf("Failed to save alert rule: %v", err)
	}

	// Test alert rule retrieval
	rules, err := persistence.GetAlertRules(ctx)
	if err != nil {
		t.Errorf("Failed to get alert rules: %v", err)
	}

	ruleFound := false
	for _, rule := range rules {
		if rule.Name == alertRule.Name {
			ruleFound = true
			break
		}
	}
	if !ruleFound {
		t.Error("Saved alert rule not found in alert rules")
	}
}

// Mock cache service for testing
type mockCacheService struct{}

func (m *mockCacheService) Get(ctx context.Context, key string) ([]byte, error) {
	return []byte("test_value"), nil
}

func (m *mockCacheService) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return nil
}

func (m *mockCacheService) Delete(ctx context.Context, key string) error {
	return nil
}

func (m *mockCacheService) DeletePattern(ctx context.Context, pattern string) error {
	return nil
}

func (m *mockCacheService) Exists(ctx context.Context, key string) (bool, error) {
	return true, nil
}

func (m *mockCacheService) Close() error {
	return nil
}

// setupTestDatabase sets up a test database connection
func setupTestDatabase() (*sql.DB, error) {
	// This would typically use a test database
	// For now, return nil to skip database-dependent tests
	return nil, nil
}

// BenchmarkMonitoringSystem benchmarks the monitoring system performance
func BenchmarkMonitoringSystem(b *testing.B) {
	config := &config.MonitoringConfig{
		EnablePrometheus:         true,
		EnableHealthChecks:       true,
		EnableResourceMonitoring: true,
		EnableDBMonitoring:       true,
		EnableCacheMonitoring:    true,
	}

	cacheService := &mockCacheService{}
	metricsService := NewMetricsService(nil, cacheService, config)

	b.ResetTimer()

	b.Run("SystemMetrics", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := metricsService.GetSystemMetrics()
			if err != nil {
				b.Errorf("Error getting system metrics: %v", err)
			}
		}
	})

	b.Run("CacheMetrics", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := metricsService.GetCacheMetrics()
			if err != nil {
				b.Errorf("Error getting cache metrics: %v", err)
			}
		}
	})

	b.Run("HealthCheck", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			healthCheck := metricsService.PerformHealthCheck("system")
			if healthCheck == nil {
				b.Error("Health check returned nil")
			}
		}
	})
}