package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"high-performance-news-website/internal/config"
)

func TestCacheIntegration(t *testing.T) {
	// Skip integration tests if not running with integration flag
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		t.Skip("Skipping integration test: config not available")
	}

	// Try to connect to actual DragonflyDB/Redis instance
	cache, err := NewDragonflyClient(&cfg.Cache)
	if err != nil {
		t.Skip("Skipping integration test: DragonflyDB not available")
	}
	defer cache.Close()

	ctx := context.Background()

	t.Run("Real cache operations", func(t *testing.T) {
		key := "integration:test"
		value := []byte("integration test value")

		// Set value
		err := cache.Set(ctx, key, value, CacheTTLArticle)
		require.NoError(t, err)

		// Get value
		result, err := cache.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, result)

		// Check exists
		exists, err := cache.Exists(ctx, key)
		require.NoError(t, err)
		assert.True(t, exists)

		// Delete
		err = cache.Delete(ctx, key)
		require.NoError(t, err)

		// Verify deletion
		exists, err = cache.Exists(ctx, key)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("Cache invalidation integration", func(t *testing.T) {
		invalidator := NewCacheInvalidator(cache)
		builder := NewCacheKeyBuilder()

		// Set up test data
		articleID := uint64(999)
		slug := "integration-test-article"

		// Create test cache entries
		testData := map[string][]byte{
			builder.ArticleKey(articleID):        []byte("article data"),
			builder.ArticleSlugKey(slug):         []byte("article data"),
			builder.HomepageKey("en"):            []byte("homepage data"),
			builder.CategoryPageKey("tech", 1):   []byte("category data"),
			builder.TagPageKey("golang", 1):      []byte("tag data"),
			builder.RSSFeedKey("tech"):           []byte("rss data"),
			builder.SitemapKey("articles"):       []byte("sitemap data"),
		}

		// Set all test data
		for key, value := range testData {
			err := cache.Set(ctx, key, value, time.Minute)
			require.NoError(t, err)
		}

		// Invalidate article
		err = invalidator.InvalidateArticle(ctx, articleID, slug)
		require.NoError(t, err)

		// Verify all related caches are cleared
		for key := range testData {
			exists, err := cache.Exists(ctx, key)
			require.NoError(t, err)
			assert.False(t, exists, "Key %s should be deleted", key)
		}
	})

	t.Run("Performance test", func(t *testing.T) {
		// Test cache performance with realistic data sizes
		key := "performance:test"
		largeValue := make([]byte, 1024*10) // 10KB value
		for i := range largeValue {
			largeValue[i] = byte(i % 256)
		}

		start := time.Now()

		// Perform multiple operations
		for i := 0; i < 100; i++ {
			testKey := key + string(rune(i))
			
			// Set
			err := cache.Set(ctx, testKey, largeValue, time.Minute)
			require.NoError(t, err)
			
			// Get
			_, err = cache.Get(ctx, testKey)
			require.NoError(t, err)
		}

		duration := time.Since(start)
		t.Logf("100 set+get operations took: %v", duration)
		
		// Should complete within reasonable time (adjust based on requirements)
		assert.Less(t, duration, 5*time.Second, "Cache operations should be fast")

		// Cleanup
		err := cache.DeletePattern(ctx, "performance:*")
		require.NoError(t, err)
	})
}

func TestCacheHealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Skip("Skipping integration test: config not available")
	}

	cache, err := NewDragonflyClient(&cfg.Cache)
	if err != nil {
		t.Skip("Skipping integration test: DragonflyDB not available")
	}
	defer cache.Close()

	ctx := context.Background()

	t.Run("Health check", func(t *testing.T) {
		err := cache.Health(ctx)
		assert.NoError(t, err)
	})
}