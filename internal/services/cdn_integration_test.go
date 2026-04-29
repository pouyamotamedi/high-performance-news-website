package services

import (
	"testing"
	"time"

	"high-performance-news-website/internal/models"
)

// MockCDNServiceForIntegration implements CDNServiceInterface for integration testing
type MockCDNServiceForIntegration struct {
	config       *models.CDNConfig
	failoverMode bool
	shouldError  bool
	purgedURLs   []string
	purgedAll    bool
}

func NewMockCDNServiceForIntegration() *MockCDNServiceForIntegration {
	return &MockCDNServiceForIntegration{
		config: &models.CDNConfig{
			ID:        1,
			Provider:  "cloudflare",
			Domain:    "example.com",
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		failoverMode: false,
		shouldError:  false,
		purgedURLs:   make([]string, 0),
		purgedAll:    false,
	}
}

func (m *MockCDNServiceForIntegration) GetConfig() (*models.CDNConfig, error) {
	if m.shouldError {
		return nil, &models.ValidationError{Field: "config", Message: "Configuration not found"}
	}
	return m.config, nil
}

func (m *MockCDNServiceForIntegration) UpdateConfig(config *models.CDNConfig) error {
	if m.shouldError {
		return &models.ValidationError{Field: "config", Message: "Invalid configuration"}
	}
	m.config = config
	return nil
}

func (m *MockCDNServiceForIntegration) TestConnection() error {
	if m.shouldError {
		return &models.ValidationError{Field: "connection", Message: "Connection failed"}
	}
	return nil
}

func (m *MockCDNServiceForIntegration) PurgeCache(request *models.CDNPurgeRequest) (*models.CDNPurgeResponse, error) {
	if m.shouldError {
		return nil, &models.ValidationError{Field: "purge", Message: "Purge failed"}
	}
	
	if request.PurgeAll {
		m.purgedAll = true
	} else {
		m.purgedURLs = append(m.purgedURLs, request.URLs...)
	}
	
	return &models.CDNPurgeResponse{
		Success:   true,
		RequestID: "test-request-id",
		Message:   "Cache purged successfully",
		Timestamp: time.Now(),
	}, nil
}

func (m *MockCDNServiceForIntegration) PurgeURL(url string) error {
	if m.shouldError {
		return &models.ValidationError{Field: "purge", Message: "Purge URL failed"}
	}
	m.purgedURLs = append(m.purgedURLs, url)
	return nil
}

func (m *MockCDNServiceForIntegration) PurgeURLs(urls []string) error {
	if m.shouldError {
		return &models.ValidationError{Field: "purge", Message: "Purge URLs failed"}
	}
	m.purgedURLs = append(m.purgedURLs, urls...)
	return nil
}

func (m *MockCDNServiceForIntegration) PurgeAll() error {
	if m.shouldError {
		return &models.ValidationError{Field: "purge", Message: "Purge all failed"}
	}
	m.purgedAll = true
	return nil
}

func (m *MockCDNServiceForIntegration) GetStats() (*models.CDNStats, error) {
	if m.shouldError {
		return nil, &models.ValidationError{Field: "stats", Message: "Get stats failed"}
	}
	return &models.CDNStats{
		CacheHitRatio:  85.5,
		BandwidthSaved: 1024000,
		RequestsServed: 50000,
		OriginRequests: 7500,
		ResponseTime:   150,
		LastUpdated:    time.Now(),
	}, nil
}

func (m *MockCDNServiceForIntegration) GetHealthStatus() (*models.CDNHealthCheck, error) {
	if m.shouldError {
		return nil, &models.ValidationError{Field: "health", Message: "Get health failed"}
	}
	
	status := "healthy"
	if m.failoverMode {
		status = "degraded"
	}
	
	return &models.CDNHealthCheck{
		Provider:     "cloudflare",
		Status:       status,
		ResponseTime: 150,
		LastCheck:    time.Now(),
		ErrorCount:   0,
		Message:      "CDN is operating normally",
	}, nil
}

func (m *MockCDNServiceForIntegration) EnableFailover() error {
	if m.shouldError {
		return &models.ValidationError{Field: "failover", Message: "Enable failover failed"}
	}
	m.failoverMode = true
	return nil
}

func (m *MockCDNServiceForIntegration) DisableFailover() error {
	if m.shouldError {
		return &models.ValidationError{Field: "failover", Message: "Disable failover failed"}
	}
	m.failoverMode = false
	return nil
}

func (m *MockCDNServiceForIntegration) IsFailoverActive() bool {
	return m.failoverMode
}

func TestNewCDNIntegrationService(t *testing.T) {
	mockCDN := NewMockCDNServiceForIntegration()
	service := NewCDNIntegrationService(mockCDN)
	
	if service == nil {
		t.Fatal("Expected service to be created")
	}
	
	if service.cdnService != mockCDN {
		t.Error("Expected CDN service to be set")
	}
}

func TestPurgeHomepageCache(t *testing.T) {
	mockCDN := NewMockCDNServiceForIntegration()
	service := NewCDNIntegrationService(mockCDN)
	
	err := service.PurgeHomepageCache()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// Check that homepage URLs were purged
	expectedURLs := []string{
		"https://example.com/",
		"https://example.com/rss.xml",
		"https://example.com/sitemap.xml",
	}
	
	if len(mockCDN.purgedURLs) != len(expectedURLs) {
		t.Errorf("Expected %d URLs to be purged, got %d", len(expectedURLs), len(mockCDN.purgedURLs))
	}
	
	for _, expectedURL := range expectedURLs {
		found := false
		for _, purgedURL := range mockCDN.purgedURLs {
			if purgedURL == expectedURL {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected URL %s to be purged", expectedURL)
		}
	}
}

func TestPurgeAllCache(t *testing.T) {
	mockCDN := NewMockCDNServiceForIntegration()
	service := NewCDNIntegrationService(mockCDN)
	
	err := service.PurgeAllCache()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if !mockCDN.purgedAll {
		t.Error("Expected all cache to be purged")
	}
}

func TestGetCDNStats(t *testing.T) {
	mockCDN := NewMockCDNServiceForIntegration()
	service := NewCDNIntegrationService(mockCDN)
	
	stats, err := service.GetCDNStats()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if stats == nil {
		t.Fatal("Expected stats to be returned")
	}
	
	if stats.CacheHitRatio != 85.5 {
		t.Errorf("Expected cache hit ratio 85.5, got %f", stats.CacheHitRatio)
	}
	
	if stats.RequestsServed != 50000 {
		t.Errorf("Expected requests served 50000, got %d", stats.RequestsServed)
	}
}

func TestGetCDNHealth(t *testing.T) {
	mockCDN := NewMockCDNServiceForIntegration()
	service := NewCDNIntegrationService(mockCDN)
	
	health, err := service.GetCDNHealth()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if health == nil {
		t.Fatal("Expected health to be returned")
	}
	
	if health.Status != "healthy" {
		t.Errorf("Expected status healthy, got %s", health.Status)
	}
	
	if health.Provider != "cloudflare" {
		t.Errorf("Expected provider cloudflare, got %s", health.Provider)
	}
}

func TestIsEnabled(t *testing.T) {
	mockCDN := NewMockCDNServiceForIntegration()
	service := NewCDNIntegrationService(mockCDN)
	
	if !service.IsEnabled() {
		t.Error("Expected CDN to be enabled")
	}
	
	// Test with disabled CDN
	mockCDN.config.Enabled = false
	if service.IsEnabled() {
		t.Error("Expected CDN to be disabled")
	}
}

func TestNilCDNService(t *testing.T) {
	service := NewCDNIntegrationService(nil)
	
	// All operations should work without error when CDN is nil
	article := &models.Article{Slug: "test-article"}
	
	err := service.PurgeArticleCache(article)
	if err != nil {
		t.Errorf("Expected no error with nil CDN service, got %v", err)
	}
	
	err = service.PurgeCategoryCache("test-category")
	if err != nil {
		t.Errorf("Expected no error with nil CDN service, got %v", err)
	}
	
	err = service.PurgeTagCache("test-tag")
	if err != nil {
		t.Errorf("Expected no error with nil CDN service, got %v", err)
	}
	
	err = service.PurgeHomepageCache()
	if err != nil {
		t.Errorf("Expected no error with nil CDN service, got %v", err)
	}
	
	stats, err := service.GetCDNStats()
	if err != nil {
		t.Errorf("Expected no error with nil CDN service, got %v", err)
	}
	if stats != nil {
		t.Error("Expected nil stats with nil CDN service")
	}
	
	health, err := service.GetCDNHealth()
	if err != nil {
		t.Errorf("Expected no error with nil CDN service, got %v", err)
	}
	if health != nil {
		t.Error("Expected nil health with nil CDN service")
	}
	
	if service.IsEnabled() {
		t.Error("Expected CDN to be disabled with nil service")
	}
}

func TestCDNServiceErrors(t *testing.T) {
	mockCDN := NewMockCDNServiceForIntegration()
	mockCDN.shouldError = true
	service := NewCDNIntegrationService(mockCDN)
	
	// Test that errors are handled gracefully (logged but not returned)
	article := &models.Article{Slug: "test-article"}
	
	err := service.PurgeArticleCache(article)
	if err != nil {
		t.Errorf("Expected no error (should be logged), got %v", err)
	}
	
	err = service.PurgeHomepageCache()
	if err != nil {
		t.Errorf("Expected no error (should be logged), got %v", err)
	}
	
	// Test that PurgeAllCache returns errors (more critical operation)
	err = service.PurgeAllCache()
	if err == nil {
		t.Error("Expected error for PurgeAllCache when CDN service fails")
	}
	
	// Test stats and health return errors
	_, err = service.GetCDNStats()
	if err == nil {
		t.Error("Expected error for GetCDNStats when CDN service fails")
	}
	
	_, err = service.GetCDNHealth()
	if err == nil {
		t.Error("Expected error for GetCDNHealth when CDN service fails")
	}
}

func TestCDNIntegrationFailoverMode(t *testing.T) {
	mockCDN := NewMockCDNServiceForIntegration()
	service := NewCDNIntegrationService(mockCDN)
	
	// Enable failover mode
	err := mockCDN.EnableFailover()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// Check health status reflects failover mode
	health, err := service.GetCDNHealth()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if health.Status != "degraded" {
		t.Errorf("Expected status degraded in failover mode, got %s", health.Status)
	}
	
	// Disable failover mode
	err = mockCDN.DisableFailover()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// Check health status is back to normal
	health, err = service.GetCDNHealth()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if health.Status != "healthy" {
		t.Errorf("Expected status healthy after disabling failover, got %s", health.Status)
	}
}