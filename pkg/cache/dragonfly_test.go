package cache

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"high-performance-news-website/internal/config"
)

func setupTestCache(t *testing.T) (*DragonflyCache, *miniredis.Miniredis) {
	// Start miniredis server for testing
	mr, err := miniredis.Run()
	require.NoError(t, err)

	// Create test config
	cfg := &config.CacheConfig{
		Host:     mr.Host(),
		Port:     mr.Port(),
		Password: "",
		DB:       0,
	}

	// Create cache client
	cache, err := NewDragonflyClient(cfg)
	require.NoError(t, err)

	return cache, mr
}

func TestNewDragonflyClient(t *testing.T) {
	cache, mr := setupTestCache(t)
	defer mr.Close()
	defer cache.Close()

	// Test connection
	ctx := context.Background()
	err := cache.Health(ctx)
	assert.NoError(t, err)
}

func TestCacheOperations(t *testing.T) {
	cache, mr := setupTestCache(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()
	key := "test:key"
	value := []byte("test value")

	t.Run("Set and Get", func(t *testing.T) {
		// Set value
		err := cache.Set(ctx, key, value, time.Minute)
		assert.NoError(t, err)

		// Get value
		result, err := cache.Get(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, value, result)
	})

	t.Run("Get non-existent key", func(t *testing.T) {
		result, err := cache.Get(ctx, "non:existent")
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("Exists", func(t *testing.T) {
		// Key should exist
		exists, err := cache.Exists(ctx, key)
		assert.NoError(t, err)
		assert.True(t, exists)

		// Non-existent key
		exists, err = cache.Exists(ctx, "non:existent")
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("Delete", func(t *testing.T) {
		// Delete key
		err := cache.Delete(ctx, key)
		assert.NoError(t, err)

		// Verify deletion
		exists, err := cache.Exists(ctx, key)
		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestCacheTTL(t *testing.T) {
	cache, mr := setupTestCache(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()
	key := "test:ttl"
	value := []byte("test value")

	t.Run("TTL expiration", func(t *testing.T) {
		// Set with short TTL
		err := cache.Set(ctx, key, value, 100*time.Millisecond)
		assert.NoError(t, err)

		// Value should exist immediately
		result, err := cache.Get(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, value, result)

		// Fast forward time in miniredis
		mr.FastForward(200 * time.Millisecond)

		// Value should be expired
		result, err = cache.Get(ctx, key)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("TTL constants validation", func(t *testing.T) {
		// Verify TTL constants are set correctly
		assert.Equal(t, 15*time.Minute, CacheTTLHomepage)
		assert.Equal(t, 24*time.Hour, CacheTTLArticle)
		assert.Equal(t, 30*time.Minute, CacheTTLCategory)
		assert.Equal(t, 30*time.Minute, CacheTTLTag)
		assert.Equal(t, 2*time.Hour, CacheTTLRSS)
		assert.Equal(t, 10*time.Minute, CacheTTLSearch)
		assert.Equal(t, 6*time.Hour, CacheTTLSitemap)
		assert.Equal(t, 24*time.Hour, CacheTTLUser)
		assert.Equal(t, 1*time.Hour, CacheTTLConfig)
	})
}

func TestDeletePattern(t *testing.T) {
	cache, mr := setupTestCache(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()

	// Set multiple keys with pattern
	keys := []string{
		"article:1",
		"article:2",
		"article:3",
		"category:news",
		"category:sports",
	}

	for _, key := range keys {
		err := cache.Set(ctx, key, []byte("value"), time.Minute)
		assert.NoError(t, err)
	}

	t.Run("Delete pattern", func(t *testing.T) {
		// Delete all article keys
		err := cache.DeletePattern(ctx, "article:*")
		assert.NoError(t, err)

		// Verify article keys are deleted
		for i := 1; i <= 3; i++ {
			exists, err := cache.Exists(ctx, fmt.Sprintf("article:%d", i))
			assert.NoError(t, err)
			assert.False(t, exists)
		}

		// Verify category keys still exist
		exists, err := cache.Exists(ctx, "category:news")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("Delete non-matching pattern", func(t *testing.T) {
		// Delete pattern that doesn't match anything
		err := cache.DeletePattern(ctx, "nonexistent:*")
		assert.NoError(t, err) // Should not error even if no keys match
	})
}

func TestCacheInvalidator(t *testing.T) {
	cache, mr := setupTestCache(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()
	invalidator := NewCacheInvalidator(cache)

	// Set up test data
	testKeys := []string{
		"article:1",
		"article:slug:test-article",
		"homepage:en",
		"homepage:fa",
		"category:news:1",
		"tag:tech:1",
		"rss:news",
		"sitemap:articles",
		"trending:daily",
		"popular:weekly",
	}

	for _, key := range testKeys {
		err := cache.Set(ctx, key, []byte("value"), time.Minute)
		assert.NoError(t, err)
	}

	t.Run("InvalidateArticle", func(t *testing.T) {
		err := invalidator.InvalidateArticle(ctx, 1, "test-article")
		assert.NoError(t, err)

		// Verify all related caches are cleared
		expectedDeletedKeys := []string{
			"article:1",
			"article:slug:test-article",
			"homepage:en",
			"homepage:fa",
			"category:news:1",
			"tag:tech:1",
			"rss:news",
			"sitemap:articles",
			"trending:daily",
			"popular:weekly",
		}

		for _, key := range expectedDeletedKeys {
			exists, err := cache.Exists(ctx, key)
			assert.NoError(t, err)
			assert.False(t, exists, "Key %s should be deleted", key)
		}
	})
}

func TestCacheInvalidatorCategory(t *testing.T) {
	cache, mr := setupTestCache(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()
	invalidator := NewCacheInvalidator(cache)

	// Set up test data
	testKeys := []string{
		"category:news:1",
		"category:news:2",
		"category:sports:1",
		"homepage:en",
		"rss:news",
		"rss:sports",
	}

	for _, key := range testKeys {
		err := cache.Set(ctx, key, []byte("value"), time.Minute)
		assert.NoError(t, err)
	}

	t.Run("InvalidateCategory", func(t *testing.T) {
		err := invalidator.InvalidateCategory(ctx, "news")
		assert.NoError(t, err)

		// Verify news category caches are cleared
		exists, err := cache.Exists(ctx, "category:news:1")
		assert.NoError(t, err)
		assert.False(t, exists)

		exists, err = cache.Exists(ctx, "category:news:2")
		assert.NoError(t, err)
		assert.False(t, exists)

		exists, err = cache.Exists(ctx, "homepage:en")
		assert.NoError(t, err)
		assert.False(t, exists)

		exists, err = cache.Exists(ctx, "rss:news")
		assert.NoError(t, err)
		assert.False(t, exists)

		// Verify sports category cache still exists
		exists, err = cache.Exists(ctx, "category:sports:1")
		assert.NoError(t, err)
		assert.True(t, exists)

		exists, err = cache.Exists(ctx, "rss:sports")
		assert.NoError(t, err)
		assert.True(t, exists)
	})
}

func TestCacheKeyBuilder(t *testing.T) {
	builder := NewCacheKeyBuilder()

	t.Run("ArticleKey", func(t *testing.T) {
		key := builder.ArticleKey(123)
		assert.Equal(t, "article:123", key)
	})

	t.Run("ArticleSlugKey", func(t *testing.T) {
		key := builder.ArticleSlugKey("test-article")
		assert.Equal(t, "article:slug:test-article", key)
	})

	t.Run("HomepageKey", func(t *testing.T) {
		key := builder.HomepageKey("en")
		assert.Equal(t, "homepage:en", key)
	})

	t.Run("CategoryPageKey", func(t *testing.T) {
		key := builder.CategoryPageKey("news", 2)
		assert.Equal(t, "category:news:2", key)
	})

	t.Run("TagPageKey", func(t *testing.T) {
		key := builder.TagPageKey("tech", 1)
		assert.Equal(t, "tag:tech:1", key)
	})

	t.Run("RSSFeedKey", func(t *testing.T) {
		key := builder.RSSFeedKey("news")
		assert.Equal(t, "rss:news", key)
	})

	t.Run("SitemapKey", func(t *testing.T) {
		key := builder.SitemapKey("articles")
		assert.Equal(t, "sitemap:articles", key)
	})

	t.Run("SearchKey", func(t *testing.T) {
		key := builder.SearchKey("golang", 1)
		assert.Equal(t, "search:golang:1", key)
	})

	t.Run("UserSessionKey", func(t *testing.T) {
		key := builder.UserSessionKey("session123")
		assert.Equal(t, "session:session123", key)
	})

	t.Run("ConfigKey", func(t *testing.T) {
		key := builder.ConfigKey("site_name")
		assert.Equal(t, "config:site_name", key)
	})

	t.Run("TrendingKey", func(t *testing.T) {
		key := builder.TrendingKey("daily")
		assert.Equal(t, "trending:daily", key)
	})

	t.Run("PopularKey", func(t *testing.T) {
		key := builder.PopularKey("weekly")
		assert.Equal(t, "popular:weekly", key)
	})
}

func TestCacheKeyPatterns(t *testing.T) {
	t.Run("Key pattern constants", func(t *testing.T) {
		// Verify key patterns are correctly defined
		assert.Equal(t, "article:%d", CacheKeyArticle)
		assert.Equal(t, "article:slug:%s", CacheKeyArticleSlug)
		assert.Equal(t, "homepage:%s", CacheKeyHomepage)
		assert.Equal(t, "category:%s:%d", CacheKeyCategoryPage)
		assert.Equal(t, "tag:%s:%d", CacheKeyTagPage)
		assert.Equal(t, "rss:%s", CacheKeyRSSFeed)
		assert.Equal(t, "sitemap:%s", CacheKeySitemap)
		assert.Equal(t, "search:%s:%d", CacheKeySearch)
		assert.Equal(t, "session:%s", CacheKeyUserSession)
		assert.Equal(t, "config:%s", CacheKeyConfig)
		assert.Equal(t, "trending:%s", CacheKeyTrending)
		assert.Equal(t, "popular:%s", CacheKeyPopular)
	})
}

func TestCacheConnectionPooling(t *testing.T) {
	cache, mr := setupTestCache(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()

	t.Run("Multiple concurrent operations", func(t *testing.T) {
		// Test concurrent operations to verify connection pooling
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func(id int) {
				key := fmt.Sprintf("concurrent:test:%d", id)
				value := []byte(fmt.Sprintf("value-%d", id))

				// Set value
				err := cache.Set(ctx, key, value, time.Minute)
				assert.NoError(t, err)

				// Get value
				result, err := cache.Get(ctx, key)
				assert.NoError(t, err)
				assert.Equal(t, value, result)

				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

func TestCacheErrorHandling(t *testing.T) {
	// Test with invalid configuration
	t.Run("Invalid connection", func(t *testing.T) {
		cfg := &config.CacheConfig{
			Host:     "invalid-host",
			Port:     9999,
			Password: "",
			DB:       0,
		}

		_, err := NewDragonflyClient(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to connect to DragonflyDB")
	})
}

// Benchmark tests for performance validation
func BenchmarkCacheSet(b *testing.B) {
	cache, mr := setupTestCache(b.(*testing.T))
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()
	value := []byte("benchmark value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("benchmark:set:%d", i)
		cache.Set(ctx, key, value, time.Minute)
	}
}

func BenchmarkCacheGet(b *testing.B) {
	cache, mr := setupTestCache(b.(*testing.T))
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()
	key := "benchmark:get"
	value := []byte("benchmark value")

	// Pre-populate cache
	cache.Set(ctx, key, value, time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(ctx, key)
	}
}

func BenchmarkCacheExists(b *testing.B) {
	cache, mr := setupTestCache(b.(*testing.T))
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()
	key := "benchmark:exists"
	value := []byte("benchmark value")

	// Pre-populate cache
	cache.Set(ctx, key, value, time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Exists(ctx, key)
	}
}