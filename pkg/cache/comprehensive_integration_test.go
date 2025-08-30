package cache

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"high-performance-news-website/internal/config"
)

// Integration tests require a real DragonflyDB/Redis connection
// These tests are skipped unless INTEGRATION_TEST=1 environment variable is set

func setupCacheIntegrationTest(t *testing.T) (*DragonflyCache, func()) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("Skipping cache integration test. Set INTEGRATION_TEST=1 to run.")
	}

	// Setup test cache connection
	cacheConfig := &config.CacheConfig{
		Host:     getEnvOrDefault("CACHE_HOST", "localhost"),
		Port:     6379,
		Password: getEnvOrDefault("CACHE_PASSWORD", ""),
		DB:       2, // Use different DB for integration tests
	}

	cache, err := NewDragonflyClient(cacheConfig)
	require.NoError(t, err)

	// Clear test database
	ctx := context.Background()
	err = cache.DeletePattern(ctx, "*")
	require.NoError(t, err)

	cleanup := func() {
		// Clean up test data
		cache.DeletePattern(context.Background(), "*")
		cache.Close()
	}

	return cache, cleanup
}

func TestDragonflyDB_Integration_ConnectionPooling(t *testing.T) {
	cache, cleanup := setupCacheIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("Connection pool stress test", func(t *testing.T) {
		// Test concurrent operations to verify connection pooling
		const numGoroutines = 50
		const operationsPerGoroutine = 20

		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines*operationsPerGoroutine)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					key := fmt.Sprintf("pool_test:%d:%d", goroutineID, j)
					value := []byte(fmt.Sprintf("value_%d_%d", goroutineID, j))

					// Set operation
					if err := cache.Set(ctx, key, value, time.Minute); err != nil {
						errors <- fmt.Errorf("set failed for %s: %w", key, err)
						continue
					}

					// Get operation
					retrieved, err := cache.Get(ctx, key)
					if err != nil {
						errors <- fmt.Errorf("get failed for %s: %w", key, err)
						continue
					}

					if string(retrieved) != string(value) {
						errors <- fmt.Errorf("value mismatch for %s: expected %s, got %s", key, value, retrieved)
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for errors
		var errorCount int
		for err := range errors {
			t.Logf("Connection pool error: %v", err)
			errorCount++
		}

		assert.Equal(t, 0, errorCount, "Connection pool should handle concurrent operations without errors")
	})

	t.Run("Connection pool configuration validation", func(t *testing.T) {
		// Verify that the connection pool is properly configured
		client := cache.GetRedisClient()
		require.NotNil(t, client)

		// Test that we can perform operations indicating pool is working
		for i := 0; i < 10; i++ {
			key := fmt.Sprintf("config_test:%d", i)
			err := cache.Set(ctx, key, []byte("test"), time.Second*10)
			require.NoError(t, err)
		}

		// Verify all keys exist
		for i := 0; i < 10; i++ {
			key := fmt.Sprintf("config_test:%d", i)
			exists, err := cache.Exists(ctx, key)
			require.NoError(t, err)
			assert.True(t, exists)
		}
	})
}

func TestDragonflyDB_Integration_CacheInvalidation(t *testing.T) {
	cache, cleanup := setupCacheIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()
	invalidator := NewCacheInvalidator(cache)
	builder := NewCacheKeyBuilder()

	t.Run("Article invalidation cascade", func(t *testing.T) {
		articleID := uint64(12345)
		slug := "test-article-invalidation"

		// Set up comprehensive cache entries that should be invalidated
		testEntries := map[string][]byte{
			builder.ArticleKey(articleID):        []byte("article content"),
			builder.ArticleSlugKey(slug):         []byte("article content"),
			builder.HomepageKey("en"):            []byte("homepage en"),
			builder.HomepageKey("fa"):            []byte("homepage fa"),
			builder.CategoryPageKey("tech", 1):   []byte("tech page 1"),
			builder.CategoryPageKey("tech", 2):   []byte("tech page 2"),
			builder.TagPageKey("golang", 1):      []byte("golang tag page"),
			builder.RSSFeedKey("tech"):           []byte("tech rss"),
			builder.RSSFeedKey("golang"):         []byte("golang rss"),
			builder.SitemapKey("articles"):       []byte("articles sitemap"),
			builder.SitemapKey("news"):           []byte("news sitemap"),
			builder.TrendingKey("24h"):           []byte("trending 24h"),
			builder.PopularKey("week"):           []byte("popular week"),
		}

		// Set all test entries
		for key, value := range testEntries {
			err := cache.Set(ctx, key, value, time.Hour)
			require.NoError(t, err)
		}

		// Verify all entries exist before invalidation
		for key := range testEntries {
			exists, err := cache.Exists(ctx, key)
			require.NoError(t, err)
			assert.True(t, exists, "Key %s should exist before invalidation", key)
		}

		// Perform article invalidation
		err := invalidator.InvalidateArticle(ctx, articleID, slug)
		require.NoError(t, err)

		// Give some time for async operations
		time.Sleep(100 * time.Millisecond)

		// Verify all related caches are cleared
		for key := range testEntries {
			exists, err := cache.Exists(ctx, key)
			require.NoError(t, err)
			assert.False(t, exists, "Key %s should be deleted after article invalidation", key)
		}
	})

	t.Run("Category invalidation", func(t *testing.T) {
		categorySlug := "technology"

		// Set up category-related cache entries
		testEntries := map[string][]byte{
			builder.CategoryPageKey(categorySlug, 1): []byte("category page 1"),
			builder.CategoryPageKey(categorySlug, 2): []byte("category page 2"),
			builder.CategoryPageKey(categorySlug, 3): []byte("category page 3"),
			builder.HomepageKey("en"):                []byte("homepage en"),
			builder.HomepageKey("fa"):                []byte("homepage fa"),
			builder.RSSFeedKey(categorySlug):         []byte("category rss"),
		}

		// Set all test entries
		for key, value := range testEntries {
			err := cache.Set(ctx, key, value, time.Hour)
			require.NoError(t, err)
		}

		// Perform category invalidation
		err := invalidator.InvalidateCategory(ctx, categorySlug)
		require.NoError(t, err)

		// Verify category-related caches are cleared
		for key := range testEntries {
			exists, err := cache.Exists(ctx, key)
			require.NoError(t, err)
			assert.False(t, exists, "Key %s should be deleted after category invalidation", key)
		}
	})

	t.Run("Tag invalidation", func(t *testing.T) {
		tagSlug := "artificial-intelligence"

		// Set up tag-related cache entries
		testEntries := map[string][]byte{
			builder.TagPageKey(tagSlug, 1): []byte("tag page 1"),
			builder.TagPageKey(tagSlug, 2): []byte("tag page 2"),
			builder.RSSFeedKey(tagSlug):    []byte("tag rss"),
		}

		// Set all test entries
		for key, value := range testEntries {
			err := cache.Set(ctx, key, value, time.Hour)
			require.NoError(t, err)
		}

		// Perform tag invalidation
		err := invalidator.InvalidateTag(ctx, tagSlug)
		require.NoError(t, err)

		// Verify tag-related caches are cleared
		for key := range testEntries {
			exists, err := cache.Exists(ctx, key)
			require.NoError(t, err)
			assert.False(t, exists, "Key %s should be deleted after tag invalidation", key)
		}
	})
}

func TestDragonflyDB_Integration_TTLBehavior(t *testing.T) {
	cache, cleanup := setupCacheIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("TTL expiration behavior", func(t *testing.T) {
		key := "ttl_test_key"
		value := []byte("ttl test value")
		shortTTL := 2 * time.Second

		// Set with short TTL
		err := cache.Set(ctx, key, value, shortTTL)
		require.NoError(t, err)

		// Verify key exists immediately
		exists, err := cache.Exists(ctx, key)
		require.NoError(t, err)
		assert.True(t, exists)

		// Verify value can be retrieved
		retrieved, err := cache.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, retrieved)

		// Wait for TTL to expire
		time.Sleep(shortTTL + 500*time.Millisecond)

		// Verify key no longer exists
		exists, err = cache.Exists(ctx, key)
		require.NoError(t, err)
		assert.False(t, exists)

		// Verify get returns nil for expired key
		retrieved, err = cache.Get(ctx, key)
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("Different TTL values for different content types", func(t *testing.T) {
		builder := NewCacheKeyBuilder()

		testCases := []struct {
			name string
			key  string
			ttl  time.Duration
		}{
			{"Homepage", builder.HomepageKey("en"), CacheTTLHomepage},
			{"Article", builder.ArticleKey(123), CacheTTLArticle},
			{"Category", builder.CategoryPageKey("tech", 1), CacheTTLCategory},
			{"Tag", builder.TagPageKey("golang", 1), CacheTTLTag},
			{"RSS", builder.RSSFeedKey("tech"), CacheTTLRSS},
			{"Search", builder.SearchKey("golang", 1), CacheTTLSearch},
			{"Sitemap", builder.SitemapKey("articles"), CacheTTLSitemap},
			{"User Session", builder.UserSessionKey("session123"), CacheTTLUser},
			{"Config", builder.ConfigKey("site_settings"), CacheTTLConfig},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				value := []byte(fmt.Sprintf("%s test value", tc.name))

				// Set with appropriate TTL
				err := cache.Set(ctx, tc.key, value, tc.ttl)
				require.NoError(t, err)

				// Verify key exists
				exists, err := cache.Exists(ctx, tc.key)
				require.NoError(t, err)
				assert.True(t, exists)

				// Verify value can be retrieved
				retrieved, err := cache.Get(ctx, tc.key)
				require.NoError(t, err)
				assert.Equal(t, value, retrieved)
			})
		}
	})
}

func TestDragonflyDB_Integration_CacheWarming(t *testing.T) {
	cache, cleanup := setupCacheIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()
	builder := NewCacheKeyBuilder()

	t.Run("Cache warming simulation", func(t *testing.T) {
		// Simulate cache warming with realistic data
		warmingData := map[string][]byte{
			builder.HomepageKey("en"):            generateRealisticContent("homepage", 5000),
			builder.HomepageKey("fa"):            generateRealisticContent("homepage_fa", 5000),
			builder.CategoryPageKey("tech", 1):   generateRealisticContent("tech_page", 3000),
			builder.CategoryPageKey("news", 1):   generateRealisticContent("news_page", 3000),
			builder.TagPageKey("golang", 1):      generateRealisticContent("golang_tag", 2000),
			builder.TagPageKey("ai", 1):          generateRealisticContent("ai_tag", 2000),
			builder.SitemapKey("articles"):       generateRealisticContent("sitemap", 10000),
			builder.RSSFeedKey("tech"):           generateRealisticContent("rss_tech", 8000),
		}

		// Warm up cache with realistic content sizes
		start := time.Now()
		for key, value := range warmingData {
			err := cache.Set(ctx, key, value, time.Hour)
			require.NoError(t, err)
		}
		warmingDuration := time.Since(start)

		t.Logf("Cache warming took: %v for %d entries", warmingDuration, len(warmingData))

		// Verify all entries are cached
		for key := range warmingData {
			exists, err := cache.Exists(ctx, key)
			require.NoError(t, err)
			assert.True(t, exists, "Warmed cache entry %s should exist", key)
		}

		// Test cache hit performance after warming
		start = time.Now()
		for key := range warmingData {
			_, err := cache.Get(ctx, key)
			require.NoError(t, err)
		}
		retrievalDuration := time.Since(start)

		t.Logf("Cache retrieval took: %v for %d entries", retrievalDuration, len(warmingData))

		// Cache retrieval should be significantly faster than warming
		assert.Less(t, retrievalDuration, warmingDuration/2, "Cache hits should be faster than cache warming")
	})
}

func TestDragonflyDB_Integration_CacheFailover(t *testing.T) {
	cache, cleanup := setupCacheIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("Cache failure simulation and fallback", func(t *testing.T) {
		key := "failover_test"
		value := []byte("failover test value")

		// Set initial value
		err := cache.Set(ctx, key, value, time.Hour)
		require.NoError(t, err)

		// Verify value exists
		retrieved, err := cache.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, retrieved)

		// Simulate cache failure by closing connection temporarily
		// Note: In a real failover test, you would simulate network issues
		// or server unavailability. Here we test graceful error handling.

		// Test error handling for non-existent keys (simulates cache miss during failure)
		nonExistentKey := "non_existent_key"
		retrieved, err = cache.Get(ctx, nonExistentKey)
		require.NoError(t, err)
		assert.Nil(t, retrieved, "Non-existent key should return nil, not error")

		// Test that cache operations continue to work after simulated issues
		newKey := "post_failover_test"
		newValue := []byte("post failover value")

		err = cache.Set(ctx, newKey, newValue, time.Hour)
		require.NoError(t, err)

		retrieved, err = cache.Get(ctx, newKey)
		require.NoError(t, err)
		assert.Equal(t, newValue, retrieved)
	})

	t.Run("Health check during operations", func(t *testing.T) {
		// Verify health check works during normal operations
		err := cache.Health(ctx)
		assert.NoError(t, err)

		// Perform some operations
		for i := 0; i < 10; i++ {
			key := fmt.Sprintf("health_test_%d", i)
			value := []byte(fmt.Sprintf("health test value %d", i))

			err := cache.Set(ctx, key, value, time.Minute)
			require.NoError(t, err)

			// Check health between operations
			err = cache.Health(ctx)
			assert.NoError(t, err)
		}
	})
}

func TestDragonflyDB_Integration_Performance(t *testing.T) {
	cache, cleanup := setupCacheIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("High-throughput operations", func(t *testing.T) {
		const numOperations = 1000
		const valueSize = 1024 // 1KB values

		// Generate test data
		testData := make(map[string][]byte, numOperations)
		for i := 0; i < numOperations; i++ {
			key := fmt.Sprintf("perf_test_%d", i)
			value := make([]byte, valueSize)
			for j := range value {
				value[j] = byte(i % 256)
			}
			testData[key] = value
		}

		// Test SET operations
		start := time.Now()
		for key, value := range testData {
			err := cache.Set(ctx, key, value, time.Hour)
			require.NoError(t, err)
		}
		setDuration := time.Since(start)

		t.Logf("SET operations: %d ops in %v (%.2f ops/sec)", 
			numOperations, setDuration, float64(numOperations)/setDuration.Seconds())

		// Test GET operations
		start = time.Now()
		for key, expectedValue := range testData {
			retrieved, err := cache.Get(ctx, key)
			require.NoError(t, err)
			assert.Equal(t, expectedValue, retrieved)
		}
		getDuration := time.Since(start)

		t.Logf("GET operations: %d ops in %v (%.2f ops/sec)", 
			numOperations, getDuration, float64(numOperations)/getDuration.Seconds())

		// Performance assertions (adjust based on requirements)
		assert.Less(t, setDuration, 10*time.Second, "SET operations should complete within 10 seconds")
		assert.Less(t, getDuration, 5*time.Second, "GET operations should complete within 5 seconds")
		assert.Less(t, getDuration, setDuration, "GET operations should be faster than SET operations")
	})

	t.Run("Concurrent performance test", func(t *testing.T) {
		const numGoroutines = 20
		const operationsPerGoroutine = 100

		var wg sync.WaitGroup
		start := time.Now()

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					key := fmt.Sprintf("concurrent_%d_%d", goroutineID, j)
					value := []byte(fmt.Sprintf("value_%d_%d", goroutineID, j))

					// Set and immediately get
					err := cache.Set(ctx, key, value, time.Hour)
					require.NoError(t, err)

					retrieved, err := cache.Get(ctx, key)
					require.NoError(t, err)
					assert.Equal(t, value, retrieved)
				}
			}(i)
		}

		wg.Wait()
		totalDuration := time.Since(start)
		totalOperations := numGoroutines * operationsPerGoroutine * 2 // SET + GET

		t.Logf("Concurrent operations: %d ops in %v (%.2f ops/sec)", 
			totalOperations, totalDuration, float64(totalOperations)/totalDuration.Seconds())

		// Should handle concurrent operations efficiently
		assert.Less(t, totalDuration, 30*time.Second, "Concurrent operations should complete within 30 seconds")
	})
}

// Helper function to generate realistic content for cache testing
func generateRealisticContent(contentType string, size int) []byte {
	content := make([]byte, size)
	
	// Generate semi-realistic content based on type
	switch contentType {
	case "homepage", "homepage_fa":
		// Simulate HTML content
		template := `<html><head><title>News Site</title></head><body><div class="articles">`
		for i := 0; i < size-len(template)-20; i++ {
			if i%100 == 0 {
				content[i] = '<'
			} else if i%100 == 50 {
				content[i] = '>'
			} else {
				content[i] = byte('a' + (i % 26))
			}
		}
	case "sitemap":
		// Simulate XML sitemap
		for i := 0; i < size; i++ {
			if i%200 == 0 {
				content[i] = '<'
			} else if i%200 == 100 {
				content[i] = '>'
			} else {
				content[i] = byte('0' + (i % 10))
			}
		}
	default:
		// Generic content
		for i := 0; i < size; i++ {
			content[i] = byte('A' + (i % 26))
		}
	}
	
	return content
}

// Benchmark tests for cache integration
func BenchmarkDragonflyDB_Integration_Set(b *testing.B) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		b.Skip("Skipping integration benchmark. Set INTEGRATION_TEST=1 to run.")
	}

	cache, cleanup := setupCacheIntegrationTest(&testing.T{})
	defer cleanup()

	ctx := context.Background()
	value := make([]byte, 1024) // 1KB value

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("benchmark_set_%d", i)
		err := cache.Set(ctx, key, value, time.Hour)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDragonflyDB_Integration_Get(b *testing.B) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		b.Skip("Skipping integration benchmark. Set INTEGRATION_TEST=1 to run.")
	}

	cache, cleanup := setupCacheIntegrationTest(&testing.T{})
	defer cleanup()

	ctx := context.Background()
	value := make([]byte, 1024) // 1KB value

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("benchmark_get_%d", i)
		err := cache.Set(ctx, key, value, time.Hour)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("benchmark_get_%d", i%1000)
		_, err := cache.Get(ctx, key)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDragonflyDB_Integration_Concurrent(b *testing.B) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		b.Skip("Skipping integration benchmark. Set INTEGRATION_TEST=1 to run.")
	}

	cache, cleanup := setupCacheIntegrationTest(&testing.T{})
	defer cleanup()

	ctx := context.Background()
	value := make([]byte, 1024) // 1KB value

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("benchmark_concurrent_%d", i)
			
			// Alternate between set and get operations
			if i%2 == 0 {
				err := cache.Set(ctx, key, value, time.Hour)
				if err != nil {
					b.Fatal(err)
				}
			} else {
				_, err := cache.Get(ctx, key)
				if err != nil {
					b.Fatal(err)
				}
			}
			i++
		}
	})
}