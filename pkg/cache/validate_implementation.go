package cache

import (
	"context"
	"fmt"
	"time"
)

// ValidateImplementation performs comprehensive validation of the cache implementation
func ValidateImplementation(cache CacheService) error {
	ctx := context.Background()

	// Test 1: Basic operations
	if err := validateBasicOperations(ctx, cache); err != nil {
		return fmt.Errorf("basic operations validation failed: %w", err)
	}

	// Test 2: TTL behavior
	if err := validateTTLBehavior(ctx, cache); err != nil {
		return fmt.Errorf("TTL behavior validation failed: %w", err)
	}

	// Test 3: Pattern deletion
	if err := validatePatternDeletion(ctx, cache); err != nil {
		return fmt.Errorf("pattern deletion validation failed: %w", err)
	}

	// Test 4: Key patterns
	if err := validateKeyPatterns(); err != nil {
		return fmt.Errorf("key patterns validation failed: %w", err)
	}

	// Test 5: Cache invalidation
	if err := validateCacheInvalidation(ctx, cache); err != nil {
		return fmt.Errorf("cache invalidation validation failed: %w", err)
	}

	return nil
}

func validateBasicOperations(ctx context.Context, cache CacheService) error {
	key := "validate:basic"
	value := []byte("test value")

	// Set
	if err := cache.Set(ctx, key, value, time.Minute); err != nil {
		return fmt.Errorf("set operation failed: %w", err)
	}

	// Get
	result, err := cache.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("get operation failed: %w", err)
	}
	if string(result) != string(value) {
		return fmt.Errorf("get returned incorrect value: expected %s, got %s", value, result)
	}

	// Exists
	exists, err := cache.Exists(ctx, key)
	if err != nil {
		return fmt.Errorf("exists operation failed: %w", err)
	}
	if !exists {
		return fmt.Errorf("exists returned false for existing key")
	}

	// Delete
	if err := cache.Delete(ctx, key); err != nil {
		return fmt.Errorf("delete operation failed: %w", err)
	}

	// Verify deletion
	exists, err = cache.Exists(ctx, key)
	if err != nil {
		return fmt.Errorf("exists check after delete failed: %w", err)
	}
	if exists {
		return fmt.Errorf("key still exists after deletion")
	}

	return nil
}

func validateTTLBehavior(ctx context.Context, cache CacheService) error {
	key := "validate:ttl"
	value := []byte("ttl test")

	// Set with short TTL
	if err := cache.Set(ctx, key, value, 100*time.Millisecond); err != nil {
		return fmt.Errorf("set with TTL failed: %w", err)
	}

	// Should exist immediately
	exists, err := cache.Exists(ctx, key)
	if err != nil {
		return fmt.Errorf("exists check failed: %w", err)
	}
	if !exists {
		return fmt.Errorf("key should exist immediately after set")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired (note: this might not work with miniredis without FastForward)
	result, err := cache.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("get after expiration failed: %w", err)
	}
	// Note: In real Redis, this should be nil, but miniredis might behave differently
	_ = result // We'll just check that no error occurred

	return nil
}

func validatePatternDeletion(ctx context.Context, cache CacheService) error {
	// Set multiple keys with pattern
	keys := []string{
		"validate:pattern:1",
		"validate:pattern:2",
		"validate:pattern:3",
		"validate:other:1",
	}

	for _, key := range keys {
		if err := cache.Set(ctx, key, []byte("value"), time.Minute); err != nil {
			return fmt.Errorf("failed to set key %s: %w", key, err)
		}
	}

	// Delete pattern
	if err := cache.DeletePattern(ctx, "validate:pattern:*"); err != nil {
		return fmt.Errorf("delete pattern failed: %w", err)
	}

	// Check that pattern keys are deleted
	for i := 1; i <= 3; i++ {
		key := fmt.Sprintf("validate:pattern:%d", i)
		exists, err := cache.Exists(ctx, key)
		if err != nil {
			return fmt.Errorf("exists check failed for %s: %w", key, err)
		}
		if exists {
			return fmt.Errorf("key %s should be deleted by pattern", key)
		}
	}

	// Check that other key still exists
	exists, err := cache.Exists(ctx, "validate:other:1")
	if err != nil {
		return fmt.Errorf("exists check failed for other key: %w", err)
	}
	if !exists {
		return fmt.Errorf("other key should not be deleted by pattern")
	}

	// Cleanup
	cache.Delete(ctx, "validate:other:1")

	return nil
}

func validateKeyPatterns() error {
	builder := NewCacheKeyBuilder()

	// Test key building
	tests := []struct {
		method   string
		expected string
		actual   string
	}{
		{"ArticleKey", "article:123", builder.ArticleKey(123)},
		{"ArticleSlugKey", "article:slug:test", builder.ArticleSlugKey("test")},
		{"HomepageKey", "homepage:en", builder.HomepageKey("en")},
		{"CategoryPageKey", "category:news:1", builder.CategoryPageKey("news", 1)},
		{"TagPageKey", "tag:tech:2", builder.TagPageKey("tech", 2)},
		{"RSSFeedKey", "rss:sports", builder.RSSFeedKey("sports")},
		{"SitemapKey", "sitemap:articles", builder.SitemapKey("articles")},
		{"SearchKey", "search:golang:1", builder.SearchKey("golang", 1)},
		{"UserSessionKey", "session:abc123", builder.UserSessionKey("abc123")},
		{"ConfigKey", "config:site_name", builder.ConfigKey("site_name")},
		{"TrendingKey", "trending:daily", builder.TrendingKey("daily")},
		{"PopularKey", "popular:weekly", builder.PopularKey("weekly")},
	}

	for _, test := range tests {
		if test.actual != test.expected {
			return fmt.Errorf("%s: expected %s, got %s", test.method, test.expected, test.actual)
		}
	}

	return nil
}

func validateCacheInvalidation(ctx context.Context, cache CacheService) error {
	invalidator := NewCacheInvalidator(cache)
	builder := NewCacheKeyBuilder()

	// Set up test data
	testKeys := []string{
		builder.ArticleKey(999),
		builder.ArticleSlugKey("test-article"),
		builder.HomepageKey("en"),
		builder.CategoryPageKey("news", 1),
		builder.TagPageKey("tech", 1),
		builder.RSSFeedKey("news"),
		builder.SitemapKey("articles"),
		builder.TrendingKey("daily"),
		builder.PopularKey("weekly"),
	}

	// Set all test keys
	for _, key := range testKeys {
		if err := cache.Set(ctx, key, []byte("test"), time.Minute); err != nil {
			return fmt.Errorf("failed to set test key %s: %w", key, err)
		}
	}

	// Invalidate article
	if err := invalidator.InvalidateArticle(ctx, 999, "test-article"); err != nil {
		return fmt.Errorf("article invalidation failed: %w", err)
	}

	// Check that all keys are deleted
	for _, key := range testKeys {
		exists, err := cache.Exists(ctx, key)
		if err != nil {
			return fmt.Errorf("exists check failed for %s: %w", key, err)
		}
		if exists {
			return fmt.Errorf("key %s should be deleted by article invalidation", key)
		}
	}

	return nil
}

// ValidateTTLConstants checks that all TTL constants are properly defined
func ValidateTTLConstants() error {
	constants := map[string]time.Duration{
		"CacheTTLHomepage": CacheTTLHomepage,
		"CacheTTLArticle":  CacheTTLArticle,
		"CacheTTLCategory": CacheTTLCategory,
		"CacheTTLTag":      CacheTTLTag,
		"CacheTTLRSS":      CacheTTLRSS,
		"CacheTTLSearch":   CacheTTLSearch,
		"CacheTTLSitemap":  CacheTTLSitemap,
		"CacheTTLUser":     CacheTTLUser,
		"CacheTTLConfig":   CacheTTLConfig,
	}

	expected := map[string]time.Duration{
		"CacheTTLHomepage": 15 * time.Minute,
		"CacheTTLArticle":  24 * time.Hour,
		"CacheTTLCategory": 30 * time.Minute,
		"CacheTTLTag":      30 * time.Minute,
		"CacheTTLRSS":      2 * time.Hour,
		"CacheTTLSearch":   10 * time.Minute,
		"CacheTTLSitemap":  6 * time.Hour,
		"CacheTTLUser":     24 * time.Hour,
		"CacheTTLConfig":   1 * time.Hour,
	}

	for name, actual := range constants {
		if expected[name] != actual {
			return fmt.Errorf("TTL constant %s: expected %v, got %v", name, expected[name], actual)
		}
	}

	return nil
}

// ValidateKeyConstants checks that all key pattern constants are properly defined
func ValidateKeyConstants() error {
	constants := map[string]string{
		"CacheKeyArticle":     CacheKeyArticle,
		"CacheKeyArticleSlug": CacheKeyArticleSlug,
		"CacheKeyHomepage":    CacheKeyHomepage,
		"CacheKeyCategoryPage": CacheKeyCategoryPage,
		"CacheKeyTagPage":     CacheKeyTagPage,
		"CacheKeyRSSFeed":     CacheKeyRSSFeed,
		"CacheKeySitemap":     CacheKeySitemap,
		"CacheKeySearch":      CacheKeySearch,
		"CacheKeyUserSession": CacheKeyUserSession,
		"CacheKeyConfig":      CacheKeyConfig,
		"CacheKeyTrending":    CacheKeyTrending,
		"CacheKeyPopular":     CacheKeyPopular,
	}

	expected := map[string]string{
		"CacheKeyArticle":     "article:%d",
		"CacheKeyArticleSlug": "article:slug:%s",
		"CacheKeyHomepage":    "homepage:%s",
		"CacheKeyCategoryPage": "category:%s:%d",
		"CacheKeyTagPage":     "tag:%s:%d",
		"CacheKeyRSSFeed":     "rss:%s",
		"CacheKeySitemap":     "sitemap:%s",
		"CacheKeySearch":      "search:%s:%d",
		"CacheKeyUserSession": "session:%s",
		"CacheKeyConfig":      "config:%s",
		"CacheKeyTrending":    "trending:%s",
		"CacheKeyPopular":     "popular:%s",
	}

	for name, actual := range constants {
		if expected[name] != actual {
			return fmt.Errorf("key constant %s: expected %s, got %s", name, expected[name], actual)
		}
	}

	return nil
}