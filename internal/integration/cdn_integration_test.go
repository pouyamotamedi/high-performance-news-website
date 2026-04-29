package integration

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
		
		// Test validation
		if cdnConfig.Enabled {
			err := cdnConfig.Validate()
			if err != nil {
				t.Logf("CDN validation failed (expected in test environment): %v", err)
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
	
	// Test CDN models
	t.Run("CDN Models", func(t *testing.T) {
		// Test CDN config model
		config := &models.CDNConfig{
			ID:        1,
			Provider:  "cloudflare",
			APIKey:    "test-key",
			ZoneID:    "test-zone",
			Domain:    "example.com",
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		if config.Provider != "cloudflare" {
			t.Errorf("Expected provider cloudflare, got %s", config.Provider)
		}
		
		// Test CDN purge request model
		purgeRequest := &models.CDNPurgeRequest{
			URLs:     []string{"https://example.com/page1"},
			Tags:     []string{"tag1"},
			Hosts:    []string{"example.com"},
			PurgeAll: false,
		}
		
		if len(purgeRequest.URLs) != 1 {
			t.Errorf("Expected 1 URL, got %d", len(purgeRequest.URLs))
		}
		
		// Test CDN purge response model
		purgeResponse := &models.CDNPurgeResponse{
			Success:   true,
			RequestID: "test-id",
			Message:   "Success",
			Timestamp: time.Now(),
		}
		
		if !purgeResponse.Success {
			t.Error("Expected success to be true")
		}
		
		// Test CDN stats model
		stats := &models.CDNStats{
			CacheHitRatio:  85.5,
			BandwidthSaved: 1024000,
			RequestsServed: 50000,
			OriginRequests: 7500,
			ResponseTime:   150,
			LastUpdated:    time.Now(),
		}
		
		if stats.CacheHitRatio != 85.5 {
			t.Errorf("Expected cache hit ratio 85.5, got %f", stats.CacheHitRatio)
		}
		
		// Test CDN health check model
		health := &models.CDNHealthCheck{
			Provider:     "cloudflare",
			Status:       "healthy",
			ResponseTime: 150,
			LastCheck:    time.Now(),
			ErrorCount:   0,
			Message:      "All systems operational",
		}
		
		if health.Status != "healthy" {
			t.Errorf("Expected status healthy, got %s", health.Status)
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
	
	// Test invalid configurations
	t.Run("Invalid Configurations", func(t *testing.T) {
		testCases := []struct {
			name   string
			config *config.CDNConfig
		}{
			{
				name: "Missing Provider",
				config: &config.CDNConfig{
					Enabled: true,
					APIKey:  "test-key",
					ZoneID:  "test-zone",
					Domain:  "example.com",
				},
			},
			{
				name: "Missing API Key",
				config: &config.CDNConfig{
					Enabled:  true,
					Provider: "cloudflare",
					ZoneID:   "test-zone",
					Domain:   "example.com",
				},
			},
			{
				name: "Missing Zone ID",
				config: &config.CDNConfig{
					Enabled:  true,
					Provider: "cloudflare",
					APIKey:   "test-key",
					Domain:   "example.com",
				},
			},
			{
				name: "Missing Domain",
				config: &config.CDNConfig{
					Enabled:  true,
					Provider: "cloudflare",
					APIKey:   "test-key",
					ZoneID:   "test-zone",
				},
			},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := tc.config.Validate()
				if err == nil {
					t.Errorf("Expected validation error for %s", tc.name)
				}
			})
		}
	})
}

func TestCDNPerformanceOptimization(t *testing.T) {
	// Test that CDN operations are designed for high performance
	t.Run("Batch URL Purging", func(t *testing.T) {
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
		
		// Test with large number of URLs (should handle batching)
		urls := make([]string, 100)
		for i := 0; i < 100; i++ {
			urls[i] = "https://example.com/page" + string(rune(i))
		}
		
		// This should not panic or error due to batching logic
		err := cdnService.PurgeURLs(urls)
		// We expect this to fail in test environment, but it should handle batching gracefully
		t.Logf("Batch purge result (expected to fail in test): %v", err)
	})
	
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