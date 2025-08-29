package services

import (
	"context"
	"time"
	"high-performance-news-website/internal/models"
)

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

// CategoryRepositoryInterface defines the interface for category repository operations
type CategoryRepositoryInterface interface {
	GetAll() ([]*models.Category, error)
	GetByID(id uint64) (*models.Category, error)
	GetTotalCount() (int, error)
	GetTopCategories(limit int) ([]*models.Category, error)
}

// UserRepositoryInterface defines the interface for user repository operations  
type UserRepositoryInterface interface {
	GetAll() ([]*models.User, error)
	GetByID(id uint64) (*models.User, error)
	GetTotalCount() (int, error)
}

// TagRepositoryInterface defines the interface for tag repository operations
type TagRepositoryInterface interface {
	GetAll() ([]*models.Tag, error)
	GetByID(id uint64) (*models.Tag, error)
	GetByArticleID(articleID uint64) ([]*models.Tag, error)
	GetTotalCount() (int, error)
	GetAllWithKeywords(ctx context.Context) ([]models.Tag, error)
}

// ArticleRepositoryInterface defines the interface for article repository operations
type ArticleRepositoryInterface interface {
	GetByID(id uint64) (*models.Article, error)
	Update(article *models.Article) error
	GetByFilter(filter *models.ArticleFilter, offset, limit int) ([]*models.Article, error)
	GetTotalCount() (int, error)
	GetCountSince(since time.Time) (int, error)
	GetCountByCategory(categoryID uint64) (int, error)
	GetCountByTag(tagID uint64) (int, error)
	GetCountByAuthor(authorID uint64) (int, error)
	GetRecent(limit int) ([]*models.Article, error)
	GetPopular(limit int) ([]*models.Article, error)
}

// CDNServiceInterface defines the interface for CDN operations
type CDNServiceInterface interface {
	// Configuration management
	GetConfig() (*models.CDNConfig, error)
	UpdateConfig(config *models.CDNConfig) error
	TestConnection() error
	
	// Cache management
	PurgeCache(request *models.CDNPurgeRequest) (*models.CDNPurgeResponse, error)
	PurgeURL(url string) error
	PurgeURLs(urls []string) error
	PurgeAll() error
	
	// Performance monitoring
	GetStats() (*models.CDNStats, error)
	GetHealthStatus() (*models.CDNHealthCheck, error)
	
	// Failover management
	EnableFailover() error
	DisableFailover() error
	IsFailoverActive() bool
}