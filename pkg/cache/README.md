# Cache Package

This package provides a high-performance caching layer using DragonflyDB (Redis-compatible) for the news website platform.

## Features

- **CacheService Interface**: Clean abstraction for cache operations
- **DragonflyDB Client**: Optimized Redis client with connection pooling
- **TTL Constants**: Pre-configured cache durations for different content types
- **Cache Key Patterns**: Consistent key naming conventions
- **Cache Invalidation**: Intelligent cache invalidation strategies
- **Connection Pooling**: Optimized for high-performance scenarios

## Usage

### Basic Cache Operations

```go
package main

import (
    "context"
    "time"
    "high-performance-news-website/internal/config"
    "high-performance-news-website/pkg/cache"
)

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        panic(err)
    }

    // Create cache client
    cacheClient, err := cache.NewDragonflyClient(&cfg.Cache)
    if err != nil {
        panic(err)
    }
    defer cacheClient.Close()

    ctx := context.Background()

    // Set a value
    err = cacheClient.Set(ctx, "article:123", []byte("article data"), cache.CacheTTLArticle)
    if err != nil {
        panic(err)
    }

    // Get a value
    data, err := cacheClient.Get(ctx, "article:123")
    if err != nil {
        panic(err)
    }

    // Check if key exists
    exists, err := cacheClient.Exists(ctx, "article:123")
    if err != nil {
        panic(err)
    }

    // Delete a key
    err = cacheClient.Delete(ctx, "article:123")
    if err != nil {
        panic(err)
    }

    // Delete keys by pattern
    err = cacheClient.DeletePattern(ctx, "article:*")
    if err != nil {
        panic(err)
    }
}
```

### Using Cache Key Builder

```go
// Create key builder
builder := cache.NewCacheKeyBuilder()

// Build cache keys
articleKey := builder.ArticleKey(123)                    // "article:123"
homepageKey := builder.HomepageKey("en")                 // "homepage:en"
categoryKey := builder.CategoryPageKey("news", 1)       // "category:news:1"
tagKey := builder.TagPageKey("tech", 2)                 // "tag:tech:2"
rssKey := builder.RSSFeedKey("sports")                  // "rss:sports"
searchKey := builder.SearchKey("golang", 1)            // "search:golang:1"
```

### Cache Invalidation

```go
// Create invalidator
invalidator := cache.NewCacheInvalidator(cacheClient)

// Invalidate all caches related to an article
err := invalidator.InvalidateArticle(ctx, 123, "article-slug")
if err != nil {
    panic(err)
}

// Invalidate category-related caches
err = invalidator.InvalidateCategory(ctx, "news")
if err != nil {
    panic(err)
}

// Invalidate tag-related caches
err = invalidator.InvalidateTag(ctx, "tech")
if err != nil {
    panic(err)
}
```

## Cache TTL Constants

The package provides pre-configured TTL constants optimized for different content types:

- `CacheTTLHomepage`: 15 minutes - Homepage content
- `CacheTTLArticle`: 24 hours - Individual articles
- `CacheTTLCategory`: 30 minutes - Category pages
- `CacheTTLTag`: 30 minutes - Tag pages
- `CacheTTLRSS`: 2 hours - RSS feeds
- `CacheTTLSearch`: 10 minutes - Search results
- `CacheTTLSitemap`: 6 hours - Sitemaps
- `CacheTTLUser`: 24 hours - User sessions
- `CacheTTLConfig`: 1 hour - Configuration data

## Cache Key Patterns

Consistent key naming patterns are provided:

- `CacheKeyArticle`: "article:%d" - Article by ID
- `CacheKeyArticleSlug`: "article:slug:%s" - Article by slug
- `CacheKeyHomepage`: "homepage:%s" - Homepage by language
- `CacheKeyCategoryPage`: "category:%s:%d" - Category page
- `CacheKeyTagPage`: "tag:%s:%d" - Tag page
- `CacheKeyRSSFeed`: "rss:%s" - RSS feed
- `CacheKeySitemap`: "sitemap:%s" - Sitemap
- `CacheKeySearch`: "search:%s:%d" - Search results
- `CacheKeyUserSession`: "session:%s" - User session
- `CacheKeyConfig`: "config:%s" - Configuration
- `CacheKeyTrending`: "trending:%s" - Trending content
- `CacheKeyPopular`: "popular:%s" - Popular content

## Configuration

The cache client uses the following configuration from `internal/config`:

```yaml
cache:
  host: localhost
  port: 6379
  password: ""
  db: 0
```

Environment variables can also be used:
- `NEWS_CACHE_HOST`
- `NEWS_CACHE_PORT`
- `NEWS_CACHE_PASSWORD`
- `NEWS_CACHE_DB`

## Connection Pooling

The DragonflyDB client is optimized for high performance with:

- Pool Size: 100 connections
- Min Idle Connections: 20
- Max Idle Connections: 50
- Connection Max Idle Time: 30 minutes
- Connection Max Lifetime: 1 hour
- Dial Timeout: 5 seconds
- Read Timeout: 3 seconds
- Write Timeout: 3 seconds

## Testing

### Unit Tests

Run unit tests with:
```bash
go test ./pkg/cache -v
```

### Integration Tests

Run integration tests (requires DragonflyDB/Redis running):
```bash
go test ./pkg/cache -v -tags=integration
```

### Benchmarks

Run performance benchmarks:
```bash
go test ./pkg/cache -bench=. -benchmem
```

## Error Handling

The cache client provides proper error handling:

- Connection errors are returned immediately
- Redis `Nil` errors (key not found) are handled gracefully
- All operations include context support for timeouts
- Connection pooling handles reconnection automatically

## Performance Considerations

- Use appropriate TTL values for different content types
- Batch operations when possible using `DeletePattern`
- Monitor cache hit rates and adjust TTL values accordingly
- Use cache invalidation strategically to maintain data consistency
- Consider cache warming for frequently accessed content

## Requirements Satisfied

This implementation satisfies the following requirements:

- **Requirement 5**: Performance and Caching Strategy
- **Requirement 5.5**: Static Generation Strategy (cache support)
- **Requirement 17**: Technology Stack Requirements (DragonflyDB)