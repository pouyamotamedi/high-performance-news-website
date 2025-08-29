package services

import (
	"log"
	"high-performance-news-website/internal/models"
)

// CDNIntegrationService handles integration between CDN and content operations
type CDNIntegrationService struct {
	cdnService CDNServiceInterface
}

// NewCDNIntegrationService creates a new CDN integration service
func NewCDNIntegrationService(cdnService CDNServiceInterface) *CDNIntegrationService {
	return &CDNIntegrationService{
		cdnService: cdnService,
	}
}

// PurgeArticleCache purges CDN cache when an article is published/updated
func (c *CDNIntegrationService) PurgeArticleCache(article *models.Article) error {
	if c.cdnService == nil {
		return nil // CDN not configured, skip silently
	}
	
	// Cast to concrete type to access specific purge methods
	if cloudflareService, ok := c.cdnService.(*CloudflareCDNService); ok {
		if err := cloudflareService.PurgeArticleCache(article.Slug); err != nil {
			log.Printf("Warning: Failed to purge article cache for %s: %v", article.Slug, err)
			// Don't return error - cache purging failure shouldn't block article operations
		}
	}
	
	return nil
}

// PurgeCategoryCache purges CDN cache when a category is updated
func (c *CDNIntegrationService) PurgeCategoryCache(categorySlug string) error {
	if c.cdnService == nil {
		return nil
	}
	
	if cloudflareService, ok := c.cdnService.(*CloudflareCDNService); ok {
		if err := cloudflareService.PurgeCategoryCache(categorySlug); err != nil {
			log.Printf("Warning: Failed to purge category cache for %s: %v", categorySlug, err)
		}
	}
	
	return nil
}

// PurgeTagCache purges CDN cache when a tag is updated
func (c *CDNIntegrationService) PurgeTagCache(tagSlug string) error {
	if c.cdnService == nil {
		return nil
	}
	
	if cloudflareService, ok := c.cdnService.(*CloudflareCDNService); ok {
		if err := cloudflareService.PurgeTagCache(tagSlug); err != nil {
			log.Printf("Warning: Failed to purge tag cache for %s: %v", tagSlug, err)
		}
	}
	
	return nil
}

// PurgeHomepageCache purges homepage and main navigation cache
func (c *CDNIntegrationService) PurgeHomepageCache() error {
	if c.cdnService == nil {
		return nil
	}
	
	// Get CDN config to build URLs
	config, err := c.cdnService.GetConfig()
	if err != nil || config == nil {
		return nil
	}
	
	urls := []string{
		"https://" + config.Domain + "/",
		"https://" + config.Domain + "/rss.xml",
		"https://" + config.Domain + "/sitemap.xml",
	}
	
	if err := c.cdnService.PurgeURLs(urls); err != nil {
		log.Printf("Warning: Failed to purge homepage cache: %v", err)
	}
	
	return nil
}

// PurgeAllCache purges all CDN cache (use with caution)
func (c *CDNIntegrationService) PurgeAllCache() error {
	if c.cdnService == nil {
		return nil
	}
	
	if err := c.cdnService.PurgeAll(); err != nil {
		log.Printf("Warning: Failed to purge all cache: %v", err)
		return err
	}
	
	return nil
}

// GetCDNStats returns CDN performance statistics
func (c *CDNIntegrationService) GetCDNStats() (*models.CDNStats, error) {
	if c.cdnService == nil {
		return nil, nil
	}
	
	return c.cdnService.GetStats()
}

// GetCDNHealth returns CDN health status
func (c *CDNIntegrationService) GetCDNHealth() (*models.CDNHealthCheck, error) {
	if c.cdnService == nil {
		return nil, nil
	}
	
	return c.cdnService.GetHealthStatus()
}

// IsEnabled returns whether CDN is enabled and configured
func (c *CDNIntegrationService) IsEnabled() bool {
	if c.cdnService == nil {
		return false
	}
	
	config, err := c.cdnService.GetConfig()
	if err != nil || config == nil {
		return false
	}
	
	return config.Enabled
}