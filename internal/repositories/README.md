# Article Repository Implementation

This directory contains the article repository implementation for the high-performance news website, providing comprehensive data access layer with caching, prepared statements, and graceful degradation.

## Features Implemented

### Core Repository Operations
- **Create**: Insert new articles with validation and prepared statements
- **GetByID**: Retrieve articles by ID with multi-layer caching
- **GetBySlug**: Retrieve articles by slug with cache → database → static file fallback
- **Update**: Modify existing articles with cache invalidation
- **Delete**: Soft delete articles (archive status)
- **BulkCreate**: High-performance bulk insert using PostgreSQL COPY

### Advanced Query Operations
- **GetByCategory**: Paginated category-based article retrieval
- **GetTrendingArticles**: Trending articles based on views, likes, and recency
- **GetLatestArticles**: Most recent published articles
- **RecordView**: Analytics view tracking with async view count updates

### Performance Features
- **Prepared Statements**: All queries use prepared statements for optimal performance
- **Multi-Layer Caching**: Cache → Database → Static File fallback strategy
- **Async Operations**: Non-blocking cache invalidation and view count updates
- **Bulk Operations**: PostgreSQL COPY for high-volume article processing
- **Connection Pooling**: Optimized database connection management

### Graceful Degradation
1. **Cache First**: Try to serve from DragonflyDB cache
2. **Database Fallback**: Query PostgreSQL if cache miss
3. **Static File Fallback**: Serve pre-generated HTML as last resort

## Architecture

```
ArticleRepository
├── Database Layer (PostgreSQL with prepared statements)
├── Cache Layer (DragonflyDB with TTL management)
├── Static File Layer (Pre-generated HTML fallback)
└── Cache Invalidation (Async pattern-based invalidation)
```

## Usage Example

```go
// Initialize repository
repo := NewArticleRepository(db, cacheService, "/path/to/static/files")

// Create article
article := &models.Article{
    Title:      "Breaking News",
    Content:    "Article content...",
    AuthorID:   1,
    CategoryID: 1,
    Status:     "published",
}

created, err := repo.Create(ctx, article)
if err != nil {
    log.Fatal(err)
}

// Retrieve by slug (with caching)
retrieved, err := repo.GetBySlug(ctx, "breaking-news")
if err != nil {
    log.Fatal(err)
}

// Bulk create for high volume
articles := []models.Article{...} // 1000+ articles
err = repo.BulkCreate(ctx, articles)
if err != nil {
    log.Fatal(err)
}
```

## Performance Characteristics

### Target Performance (Requirements 1, 1.5, 22)
- **Article Creation**: < 1 second per article
- **Article Retrieval**: < 100ms (cached), < 500ms (database)
- **Bulk Operations**: 1000+ articles per minute using COPY
- **Database Queries**: < 10ms for indexed queries
- **Cache Hit Ratio**: > 80% for frequently accessed content

### Caching Strategy
- **Articles**: 24 hours TTL
- **Homepage**: 15 minutes TTL
- **Categories**: 30 minutes TTL
- **Trending**: 30 minutes TTL
- **Search Results**: 10 minutes TTL

## Testing

### Unit Tests
- Model validation testing
- Cache key generation testing
- Error handling testing
- Slug generation and validation
- SEO data validation

### Integration Tests
- Full database operations
- Cache integration testing
- Bulk operation performance
- Graceful degradation testing
- Performance benchmarking

Run tests with:
```bash
# Unit tests only
go test ./internal/repositories -v

# Integration tests (requires database)
INTEGRATION_TEST=1 go test ./internal/repositories -v

# Performance benchmarks
go test ./internal/repositories -bench=. -benchmem
```

## Database Schema Requirements

The repository expects the following database schema:

```sql
-- Articles table (partitioned by published_at)
CREATE TABLE articles (
    id BIGSERIAL,
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    excerpt TEXT,
    author_id BIGINT NOT NULL,
    category_id BIGINT NOT NULL,
    status VARCHAR(20) DEFAULT 'draft',
    published_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    view_count BIGINT DEFAULT 0,
    like_count BIGINT DEFAULT 0,
    dislike_count BIGINT DEFAULT 0,
    meta_title VARCHAR(60),
    meta_description VARCHAR(160),
    canonical_url VARCHAR(500),
    schema_type VARCHAR(50) DEFAULT 'NewsArticle',
    PRIMARY KEY (id, published_at)
) PARTITION BY RANGE (published_at);

-- Analytics table (partitioned by created_at)
CREATE TABLE article_views (
    id BIGSERIAL,
    article_id BIGINT NOT NULL,
    ip_address INET,
    user_agent TEXT,
    referer TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);
```

## Cache Invalidation Strategy

The repository implements intelligent cache invalidation:

### Article Operations
- **Create/Update/Delete**: Invalidates article cache, homepage, categories, tags, RSS feeds, sitemaps
- **Bulk Operations**: Full cache invalidation for performance
- **View Recording**: No cache invalidation (analytics only)

### Pattern-Based Invalidation
- `article:*` - Specific article caches
- `homepage:*` - Homepage in all languages
- `category:*` - All category pages
- `tag:*` - All tag pages
- `rss:*` - All RSS feeds
- `sitemap:*` - All sitemaps

## Error Handling

The repository provides comprehensive error handling:

- **Validation Errors**: Input validation with detailed field errors
- **Database Errors**: Connection, query, and constraint errors
- **Cache Errors**: Non-blocking cache failures with fallback
- **Not Found Errors**: Proper 404 handling with fallback strategies
- **Timeout Errors**: Context-aware timeout handling

## Dependencies

- `database/sql` - Database operations
- `github.com/lib/pq` - PostgreSQL driver and COPY support
- `high-performance-news-website/pkg/cache` - Cache service interface
- `high-performance-news-website/pkg/database` - Database connection management
- `high-performance-news-website/internal/models` - Data models and validation

## Configuration

The repository requires:
- Database connection with prepared statement support
- Cache service (DragonflyDB recommended)
- Static file path for fallback serving
- Proper database schema with partitioning

## Monitoring and Metrics

The repository supports monitoring through:
- Database connection pool metrics
- Cache hit/miss ratios
- Query performance timing
- Error rate tracking
- Async operation success rates

## Security Considerations

- **SQL Injection Prevention**: All queries use prepared statements
- **Input Validation**: Comprehensive validation before database operations
- **Access Control**: Repository-level access patterns
- **Data Sanitization**: HTML content sanitization
- **Error Information**: No sensitive data in error messages