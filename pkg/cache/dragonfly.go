package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"high-performance-news-website/internal/config"
)

// CacheService defines the interface for cache operations
type CacheService interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	DeletePattern(ctx context.Context, pattern string) error
	Exists(ctx context.Context, key string) (bool, error)
	Close() error
	Health(ctx context.Context) error
}

// Cache TTL constants as specified in requirements
const (
	CacheTTLHomepage = 15 * time.Minute  // Homepage: 15 minutes
	CacheTTLArticle  = 24 * time.Hour    // Articles: 24 hours
	CacheTTLCategory = 30 * time.Minute  // Categories: 30 minutes
	CacheTTLTag      = 30 * time.Minute  // Tags: 30 minutes
	CacheTTLRSS      = 2 * time.Hour     // RSS feeds: 2 hours
	CacheTTLSearch   = 10 * time.Minute  // Search results: 10 minutes
	CacheTTLSitemap  = 6 * time.Hour     // Sitemaps: 6 hours
	CacheTTLUser     = 24 * time.Hour    // User sessions: 24 hours
	CacheTTLConfig   = 1 * time.Hour     // Configuration: 1 hour
)

// Cache key patterns for consistent key naming
const (
	CacheKeyArticle     = "article:%d"
	CacheKeyArticleSlug = "article:slug:%s"
	CacheKeyHomepage    = "homepage:%s"        // language code
	CacheKeyCategoryPage = "category:%s:%d"    // slug, page
	CacheKeyTagPage     = "tag:%s:%d"          // slug, page
	CacheKeyRSSFeed     = "rss:%s"             // category/tag slug
	CacheKeySitemap     = "sitemap:%s"         // type (news, articles, etc.)
	CacheKeySearch      = "search:%s:%d"       // query, page
	CacheKeyUserSession = "session:%s"         // session ID
	CacheKeyConfig      = "config:%s"          // config key
	CacheKeyTrending    = "trending:%s"        // time period
	CacheKeyPopular     = "popular:%s"         // time period
)

// DragonflyCache implements CacheService interface
type DragonflyCache struct {
	client *redis.Client
}

// Ensure DragonflyCache implements CacheService
var _ CacheService = (*DragonflyCache)(nil)

func NewDragonflyClient(cfg *config.CacheConfig) (*DragonflyCache, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		
		// Optimized for high performance
		PoolSize:        100,
		MinIdleConns:    20,
		MaxIdleConns:    50,
		ConnMaxIdleTime: 30 * time.Minute,
		ConnMaxLifetime: time.Hour,
		
		// Timeouts
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to DragonflyDB: %w", err)
	}

	return &DragonflyCache{client: rdb}, nil
}

func (c *DragonflyCache) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil // Key not found
	}
	return val, err
}

func (c *DragonflyCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

// GetRedisClient returns the underlying Redis client for advanced operations
func (c *DragonflyCache) GetRedisClient() *redis.Client {
	return c.client
}

func (c *DragonflyCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

func (c *DragonflyCache) DeletePattern(ctx context.Context, pattern string) error {
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}
	
	if len(keys) > 0 {
		return c.client.Del(ctx, keys...).Err()
	}
	
	return nil
}

func (c *DragonflyCache) Exists(ctx context.Context, key string) (bool, error) {
	count, err := c.client.Exists(ctx, key).Result()
	return count > 0, err
}

func (c *DragonflyCache) Close() error {
	return c.client.Close()
}

func (c *DragonflyCache) Health(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// CacheInvalidator handles cache invalidation strategies
type CacheInvalidator struct {
	cache CacheService
}

// NewCacheInvalidator creates a new cache invalidator
func NewCacheInvalidator(cache CacheService) *CacheInvalidator {
	return &CacheInvalidator{cache: cache}
}

// InvalidateArticle invalidates all caches related to an article
func (ci *CacheInvalidator) InvalidateArticle(ctx context.Context, articleID uint64, slug string) error {
	// Clear specific article cache
	if err := ci.cache.Delete(ctx, fmt.Sprintf(CacheKeyArticle, articleID)); err != nil {
		return fmt.Errorf("failed to delete article cache: %w", err)
	}
	
	if err := ci.cache.Delete(ctx, fmt.Sprintf(CacheKeyArticleSlug, slug)); err != nil {
		return fmt.Errorf("failed to delete article slug cache: %w", err)
	}
	
	// Clear homepage cache (all languages)
	if err := ci.cache.DeletePattern(ctx, "homepage:*"); err != nil {
		return fmt.Errorf("failed to delete homepage cache: %w", err)
	}
	
	// Clear category and tag pages
	if err := ci.cache.DeletePattern(ctx, "category:*"); err != nil {
		return fmt.Errorf("failed to delete category cache: %w", err)
	}
	
	if err := ci.cache.DeletePattern(ctx, "tag:*"); err != nil {
		return fmt.Errorf("failed to delete tag cache: %w", err)
	}
	
	// Clear RSS feeds
	if err := ci.cache.DeletePattern(ctx, "rss:*"); err != nil {
		return fmt.Errorf("failed to delete RSS cache: %w", err)
	}
	
	// Clear sitemaps
	if err := ci.cache.DeletePattern(ctx, "sitemap:*"); err != nil {
		return fmt.Errorf("failed to delete sitemap cache: %w", err)
	}
	
	// Clear trending and popular caches
	if err := ci.cache.DeletePattern(ctx, "trending:*"); err != nil {
		return fmt.Errorf("failed to delete trending cache: %w", err)
	}
	
	if err := ci.cache.DeletePattern(ctx, "popular:*"); err != nil {
		return fmt.Errorf("failed to delete popular cache: %w", err)
	}
	
	return nil
}

// InvalidateCategory invalidates caches related to a category
func (ci *CacheInvalidator) InvalidateCategory(ctx context.Context, categorySlug string) error {
	// Clear category pages
	if err := ci.cache.DeletePattern(ctx, fmt.Sprintf("category:%s:*", categorySlug)); err != nil {
		return fmt.Errorf("failed to delete category cache: %w", err)
	}
	
	// Clear homepage cache
	if err := ci.cache.DeletePattern(ctx, "homepage:*"); err != nil {
		return fmt.Errorf("failed to delete homepage cache: %w", err)
	}
	
	// Clear RSS feeds
	if err := ci.cache.Delete(ctx, fmt.Sprintf(CacheKeyRSSFeed, categorySlug)); err != nil {
		return fmt.Errorf("failed to delete RSS cache: %w", err)
	}
	
	return nil
}

// InvalidateTag invalidates caches related to a tag
func (ci *CacheInvalidator) InvalidateTag(ctx context.Context, tagSlug string) error {
	// Clear tag pages
	if err := ci.cache.DeletePattern(ctx, fmt.Sprintf("tag:%s:*", tagSlug)); err != nil {
		return fmt.Errorf("failed to delete tag cache: %w", err)
	}
	
	// Clear RSS feeds
	if err := ci.cache.Delete(ctx, fmt.Sprintf(CacheKeyRSSFeed, tagSlug)); err != nil {
		return fmt.Errorf("failed to delete RSS cache: %w", err)
	}
	
	return nil
}

// InvalidateAll clears all caches (use with caution)
func (ci *CacheInvalidator) InvalidateAll(ctx context.Context) error {
	return ci.cache.DeletePattern(ctx, "*")
}

// CacheKeyBuilder provides helper functions for building cache keys
type CacheKeyBuilder struct{}

// NewCacheKeyBuilder creates a new cache key builder
func NewCacheKeyBuilder() *CacheKeyBuilder {
	return &CacheKeyBuilder{}
}

// ArticleKey builds cache key for an article by ID
func (ckb *CacheKeyBuilder) ArticleKey(id uint64) string {
	return fmt.Sprintf(CacheKeyArticle, id)
}

// ArticleSlugKey builds cache key for an article by slug
func (ckb *CacheKeyBuilder) ArticleSlugKey(slug string) string {
	return fmt.Sprintf(CacheKeyArticleSlug, slug)
}

// HomepageKey builds cache key for homepage
func (ckb *CacheKeyBuilder) HomepageKey(language string) string {
	return fmt.Sprintf(CacheKeyHomepage, language)
}

// CategoryPageKey builds cache key for category page
func (ckb *CacheKeyBuilder) CategoryPageKey(slug string, page int) string {
	return fmt.Sprintf(CacheKeyCategoryPage, slug, page)
}

// TagPageKey builds cache key for tag page
func (ckb *CacheKeyBuilder) TagPageKey(slug string, page int) string {
	return fmt.Sprintf(CacheKeyTagPage, slug, page)
}

// RSSFeedKey builds cache key for RSS feed
func (ckb *CacheKeyBuilder) RSSFeedKey(slug string) string {
	return fmt.Sprintf(CacheKeyRSSFeed, slug)
}

// SitemapKey builds cache key for sitemap
func (ckb *CacheKeyBuilder) SitemapKey(sitemapType string) string {
	return fmt.Sprintf(CacheKeySitemap, sitemapType)
}

// SearchKey builds cache key for search results
func (ckb *CacheKeyBuilder) SearchKey(query string, page int) string {
	return fmt.Sprintf(CacheKeySearch, query, page)
}

// UserSessionKey builds cache key for user session
func (ckb *CacheKeyBuilder) UserSessionKey(sessionID string) string {
	return fmt.Sprintf(CacheKeyUserSession, sessionID)
}

// ConfigKey builds cache key for configuration
func (ckb *CacheKeyBuilder) ConfigKey(key string) string {
	return fmt.Sprintf(CacheKeyConfig, key)
}

// TrendingKey builds cache key for trending content
func (ckb *CacheKeyBuilder) TrendingKey(period string) string {
	return fmt.Sprintf(CacheKeyTrending, period)
}

// PopularKey builds cache key for popular content
func (ckb *CacheKeyBuilder) PopularKey(period string) string {
	return fmt.Sprintf(CacheKeyPopular, period)
}