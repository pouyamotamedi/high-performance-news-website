package services

// RSSServiceInterface defines the interface for RSS services
type RSSServiceInterface interface {
	GenerateMainRSSFeed(languageCode string, limit int) ([]byte, error)
	GenerateCategoryRSSFeed(categorySlug, languageCode string, limit int) ([]byte, error)
	GenerateTagRSSFeed(tagSlug, languageCode string, limit int) ([]byte, error)
	GenerateGoogleNewsRSSFeed(languageCode string, limit int) ([]byte, error)
	ValidateRSSFeed(xmlData []byte) error
	ValidateGoogleNewsRSSFeed(xmlData []byte) error
	ForceRefreshFeed(feedType, identifier, languageCode string) error
	GetFeedStats(languageCode string) (map[string]interface{}, error)
}

// GoogleNewsSitemapServiceInterface defines the interface for Google News sitemap services
type GoogleNewsSitemapServiceInterface interface {
	GenerateGoogleNewsSitemap(languageCode string, page int) ([]byte, error)
	GenerateGoogleNewsSitemapIndex(languageCode string) ([]byte, error)
	ValidateGoogleNewsSitemap(xmlData []byte) error
	GetSitemapStats(languageCode string) (map[string]interface{}, error)
}