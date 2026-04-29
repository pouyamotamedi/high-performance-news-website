package cdn_test

import (
	"testing"
	"time"

	"high-performance-news-website/internal/config"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

func TestCDNIntegrationComplete(t *testing.T) {
	// Test CDN configuration loading
	t.Run("CDN Configuration", func(t *testing.T) {
		cdnConfig := config.LoadCDNConfig()
		if cdnConfig == nil {
			t.Fatal("CDN configuration should not be nil")
		}
		
		// Test validation with disabled CDN (should not error)
		if !cdnConfig.Enabled {
			err := cdnConfig.Validate()
			if err != nil {
				t.Errorf("Disabled CDN validation should not error: %v", err)
			}
		}
		
		// Test model conversion
		modelConfig := cdnConfig.ToModel()
		if modelConfig == nil {
			t.Fatal("Model config should not be nil")
		}
		
		if modelConfig.Provider != cdnConfig.Provider {
			t.Errorf("Expected provider %s, got %s", cdnConfig.Provider, modelConfig.Provider)
		}
	})
	
	// Test CDN service creation and basic operations
	t.Run("CDN Service Operations", func(t *testing.T) {
		config := &models.CDNConfig{
			Provider:  "cloudflare",
			APIKey:    "test-api-key",
			ZoneID:    "test-zone-id",
			Domain:    "example.com",
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		cdnService := services.NewCloudflareCDNService(config)
		if cdnService == nil {
			t.Fatal("CDN service should not be nil")
		}
		
		// Test config retrieval
		retrievedConfig, err := cdnService.GetConfig()
		if err != nil {
			t.Fatalf("Failed to get config: %v", err)
		}
		
		if retrievedConfig.Provider != "cloudflare" {
			t.Errorf("Expected provider cloudflare, got %s", retrievedConfig.Provider)
		}
		
		// Test failover functionality
		if cdnService.IsFailoverActive() {
			t.Error("Failover should not be active initially")
		}
		
		err = cdnService.EnableFailover()
		if err != nil {
			t.Fatalf("Failed to enable failover: %v", err)
		}
		
		if !cdnService.IsFailoverActive() {
			t.Error("Failover should be active after enabling")
		}
		
		err = cdnService.DisableFailover()
		if err != nil {
			t.Fatalf("Failed to disable failover: %v", err)
		}
		
		if cdnService.IsFailoverActive() {
			t.Error("Failover should not be active after disabling")
		}
	})
	
	// Test CDN integration service
	t.Run("CDN Integration Service", func(t *testing.T) {
		config := &models.CDNConfig{
			Provider:  "cloudflare",
			APIKey:    "test-api-key",
			ZoneID:    "test-zone-id",
			Domain:    "example.com",
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		cdnService := services.NewCloudflareCDNService(config)
		integrationService := services.NewCDNIntegrationService(cdnService)
		
		if integrationService == nil {
			t.Fatal("CDN integration service should not be nil")
		}
		
		if !integrationService.IsEnabled() {
			t.Error("CDN should be enabled")
		}
		
		// Test article cache purging (should not error even if CDN is not actually configured)
		article := &models.Article{
			Slug: "test-article",
		}
		
		err := integrationService.PurgeArticleCache(article)
		if err != nil {
			t.Errorf("Article cache purging should not error: %v", err)
		}
		
		// Test category cache purging
		err = integrationService.PurgeCategoryCache("test-category")
		if err != nil {
			t.Errorf("Category cache purging should not error: %v", err)
		}
		
		// Test tag cache purging
		err = integrationService.PurgeTagCache("test-tag")
		if err != nil {
			t.Errorf("Tag cache purging should not error: %v", err)
		}
		
		// Test homepage cache purging
		err = integrationService.PurgeHomepageCache()
		if err != nil {
			t.Errorf("Homepage cache purging should not error: %v", err)
		}
	})
	
	// Test CDN with nil service (graceful degradation)
	t.Run("CDN Graceful Degradation", func(t *testing.T) {
		integrationService := services.NewCDNIntegrationService(nil)
		
		if integrationService.IsEnabled() {
			t.Error("CDN should not be enabled with nil service")
		}
		
		// All operations should work without error
		article := &models.Article{Slug: "test-article"}
		
		err := integrationService.PurgeArticleCache(article)
		if err != nil {
			t.Errorf("Should not error with nil CDN service: %v", err)
		}
		
		err = integrationService.PurgeHomepageCache()
		if err != nil {
			t.Errorf("Should not error with nil CDN service: %v", err)
		}
		
		stats, err := integrationService.GetCDNStats()
		if err != nil {
			t.Errorf("Should not error with nil CDN service: %v", err)
		}
		if stats != nil {
			t.Error("Stats should be nil with nil CDN service")
		}
	})
}

func TestCDNConfigurationValidation(t *testing.T) {
	// Test valid configuration
	t.Run("Valid Configuration", func(t *testing.T) {
		config := &config.CDNConfig{
			Enabled:   true,
			Provider:  "cloudflare",
			APIKey:    "test-api-key",
			ZoneID:    "test-zone-id",
			Domain:    "example.com",
		}
		
		err := config.Validate()
		if err != nil {
			t.Errorf("Valid configuration should not error: %v", err)
		}
	})
	
	// Test disabled configuration (should skip validation)
	t.Run("Disabled Configuration", func(t *testing.T) {
		config := &config.CDNConfig{
			Enabled: false,
		}
		
		err := config.Validate()
		if err != nil {
			t.Errorf("Disabled configuration should not error: %v", err)
		}
	})
}

func TestCDNFailoverPerformance(t *testing.T) {
	// Test failover performance
	t.Run("Failover Performance", func(t *testing.T) {
		config := &models.CDNConfig{
			Provider:  "cloudflare",
			APIKey:    "test-api-key",
			ZoneID:    "test-zone-id",
			Domain:    "example.com",
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		cdnService := services.NewCloudflareCDNService(config)
		
		// Enable failover
		err := cdnService.EnableFailover()
		if err != nil {
			t.Fatalf("Failed to enable failover: %v", err)
		}
		
		// Operations in failover mode should be fast and not make external calls
		start := time.Now()
		
		purgeRequest := &models.CDNPurgeRequest{
			URLs: []string{"https://example.com/test"},
		}
		
		response, err := cdnService.PurgeCache(purgeRequest)
		duration := time.Since(start)
		
		if err != nil {
			t.Fatalf("Purge cache should not error in failover mode: %v", err)
		}
		
		if response.Success {
			t.Error("Purge should not succeed in failover mode")
		}
		
		if response.Message != "CDN is in failover mode" {
			t.Errorf("Expected failover message, got: %s", response.Message)
		}
		
		// Should be very fast since no external call is made
		if duration > 100*time.Millisecond {
			t.Errorf("Failover operation took too long: %v", duration)
		}
	})
}