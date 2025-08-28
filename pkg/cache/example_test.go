package cache_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"high-performance-news-website/internal/config"
	"high-performance-news-website/pkg/cache"
)

// ExampleDragonflyCache demonstrates basic cache usage
func ExampleDragonflyCache() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	// Create cache client
	cacheClient, err := cache.NewDragonflyClient(&cfg.Cache)
	if err != nil {
		log.Fatal(err)
	}
	defer cacheClient.Close()

	ctx := context.Background()

	// Set an article in cache
	articleData := []byte(`{"id": 123, "title": "Sample Article", "content": "Article content..."}`)
	err = cacheClient.Set(ctx, "article:123", articleData, cache.CacheTTLArticle)
	if err != nil {
		log.Fatal(err)
	}

	// Get the article from cache
	cachedData, err := cacheClient.Get(ctx, "article:123")
	if err != nil {
		log.Fatal(err)
	}

	if cachedData != nil {
		fmt.Println("Article found in cache")
	}

	// Check if article exists
	exists, err := cacheClient.Exists(ctx, "article:123")
	if err != nil {
		log.Fatal(err)
	}

	if exists {
		fmt.Println("Article exists in cache")
	}

	// Output:
	// Article found in cache
	// Article exists in cache
}

// ExampleCacheKeyBuilder demonstrates cache key building
func ExampleCacheKeyBuilder() {
	builder := cache.NewCacheKeyBuilder()

	// Build various cache keys
	articleKey := builder.ArticleKey(123)
	homepageKey := builder.HomepageKey("en")
	categoryKey := builder.CategoryPageKey("news", 1)

	fmt.Println("Article key:", articleKey)
	fmt.Println("Homepage key:", homepageKey)
	fmt.Println("Category key:", categoryKey)

	// Output:
	// Article key: article:123
	// Homepage key: homepage:en
	// Category key: category:news:1
}

// ExampleCacheInvalidator demonstrates cache invalidation
func ExampleCacheInvalidator() {
	// This example shows how to use cache invalidation
	// (actual implementation would require a real cache connection)

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	cacheClient, err := cache.NewDragonflyClient(&cfg.Cache)
	if err != nil {
		log.Fatal(err)
	}
	defer cacheClient.Close()

	ctx := context.Background()
	invalidator := cache.NewCacheInvalidator(cacheClient)

	// Set up some test data
	builder := cache.NewCacheKeyBuilder()
	testKeys := []string{
		builder.ArticleKey(123),
		builder.HomepageKey("en"),
		builder.CategoryPageKey("news", 1),
	}

	// Set test data
	for _, key := range testKeys {
		cacheClient.Set(ctx, key, []byte("test data"), time.Minute)
	}

	// Invalidate all caches related to article 123
	err = invalidator.InvalidateArticle(ctx, 123, "sample-article")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Article caches invalidated")

	// Output:
	// Article caches invalidated
}

// ExampleCacheTTL demonstrates TTL usage
func ExampleCacheTTL() {
	// Show different TTL constants
	fmt.Printf("Homepage TTL: %v\n", cache.CacheTTLHomepage)
	fmt.Printf("Article TTL: %v\n", cache.CacheTTLArticle)
	fmt.Printf("Category TTL: %v\n", cache.CacheTTLCategory)
	fmt.Printf("RSS TTL: %v\n", cache.CacheTTLRSS)

	// Output:
	// Homepage TTL: 15m0s
	// Article TTL: 24h0m0s
	// Category TTL: 30m0s
	// RSS TTL: 2h0m0s
}