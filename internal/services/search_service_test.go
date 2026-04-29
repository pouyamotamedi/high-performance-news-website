package services

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"high-performance-news-website/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	meilisearch "github.com/meilisearch/meilisearch-go"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// =============================================================================
// MOCK IMPLEMENTATIONS
// =============================================================================

// MockSearchIndexer is a mock implementation of SearchIndexerInterface
type MockSearchIndexer struct {
	mock.Mock
}

func (m *MockSearchIndexer) IndexArticle(article *models.Article) error {
	args := m.Called(article)
	return args.Error(0)
}

func (m *MockSearchIndexer) IndexArticlesBatch(articles []models.Article) error {
	args := m.Called(articles)
	return args.Error(0)
}

func (m *MockSearchIndexer) DeleteArticle(articleID string) error {
	args := m.Called(articleID)
	return args.Error(0)
}

func (m *MockSearchIndexer) RebuildIndex(articles []models.Article) error {
	args := m.Called(articles)
	return args.Error(0)
}

func (m *MockSearchIndexer) ClearIndex() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockSearchIndexer) GetIndexStats() (map[string]interface{}, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockSearchIndexer) HealthCheck() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockSearchIndexer) GetClient() meilisearch.ServiceManager {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(meilisearch.ServiceManager)
}

func (m *MockSearchIndexer) GetIndexName() string {
	args := m.Called()
	return args.String(0)
}

// MockSearchDB wraps sqlmock for SearchDB interface
type MockSearchDB struct {
	db *sql.DB
}

func (m *MockSearchDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return m.db.QueryContext(ctx, query, args...)
}

func (m *MockSearchDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return m.db.QueryRowContext(ctx, query, args...)
}

// =============================================================================
// TEST HELPERS
// =============================================================================

// setupTestRedis creates a test Redis instance using miniredis
func setupTestRedis(t *testing.T) *redis.Client {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	t.Cleanup(func() {
		client.Close()
		mr.Close()
	})

	return client
}

// setupTestDB creates a mock database for testing
func setupTestDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db, mock
}

// =============================================================================
// UNIT TESTS
// =============================================================================

// TestNewSearchService tests the creation of a new search service
func TestNewSearchService(t *testing.T) {
	mockIndexer := &MockSearchIndexer{}
	cache := setupTestRedis(t)
	db, _ := setupTestDB(t)
	mockDB := &MockSearchDB{db: db}

	service := NewSearchService(mockIndexer, cache, mockDB)

	assert.NotNil(t, service)
	assert.Equal(t, mockIndexer, service.indexer)
	assert.Equal(t, cache, service.cache)
	assert.Equal(t, mockDB, service.db)
	assert.Equal(t, "search:", service.cachePrefix)
	assert.NotNil(t, service.circuitBreaker)
}

// TestNewSearchServiceWithConfig tests creation with custom config
func TestNewSearchServiceWithConfig(t *testing.T) {
	mockIndexer := &MockSearchIndexer{}
	cache := setupTestRedis(t)
	db, _ := setupTestDB(t)
	mockDB := &MockSearchDB{db: db}

	config := SearchConfig{
		CacheTTL:                10 * time.Minute,
		MeiliSearchTimeout:      20 * time.Second,
		MaxQueryLength:          1000,
		MaxLimit:                100,
		MaxOffset:               50000,
		CircuitBreakerThreshold: 10,
		CircuitBreakerTimeout:   60 * time.Second,
	}

	service := NewSearchServiceWithConfig(mockIndexer, cache, mockDB, config)

	assert.NotNil(t, service)
	assert.Equal(t, 10*time.Minute, service.config.CacheTTL)
	assert.Equal(t, 20*time.Second, service.config.MeiliSearchTimeout)
	assert.Equal(t, 1000, service.config.MaxQueryLength)
	assert.Equal(t, 100, service.config.MaxLimit)
}

// TestGenerateCacheKey tests cache key generation
func TestGenerateCacheKey(t *testing.T) {
	cache := setupTestRedis(t)
	db, _ := setupTestDB(t)
	mockDB := &MockSearchDB{db: db}
	service := NewSearchService(nil, cache, mockDB)

	tests := []struct {
		name    string
		request SearchRequest
		check   func(t *testing.T, key string)
	}{
		{
			name: "Basic query",
			request: SearchRequest{
				Query:  "test",
				Limit:  20,
				Offset: 0,
			},
			check: func(t *testing.T, key string) {
				assert.Contains(t, key, "search:")
				assert.Contains(t, key, "q:test")
				assert.Contains(t, key, "l:20")
				assert.Contains(t, key, "o:0")
			},
		},
		{
			name: "Long query gets hashed",
			request: SearchRequest{
				Query:  string(make([]byte, 150)), // 150 chars
				Limit:  20,
				Offset: 0,
			},
			check: func(t *testing.T, key string) {
				assert.Contains(t, key, "search:")
				assert.Contains(t, key, "q:") // Should have hash prefix
				assert.Less(t, len(key), 200) // Key should be shorter than query
			},
		},
		{
			name: "Query with filters",
			request: SearchRequest{
				Query:  "news",
				Limit:  10,
				Offset: 20,
				Filters: &SearchFilters{
					AuthorID:   func() *uint64 { id := uint64(123); return &id }(),
					CategoryID: func() *uint64 { id := uint64(456); return &id }(),
				},
			},
			check: func(t *testing.T, key string) {
				assert.Contains(t, key, "a:123")
				assert.Contains(t, key, "c:456")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := service.generateCacheKey(tt.request)
			tt.check(t, key)
		})
	}
}

// TestValidateAndSanitizeRequest tests input validation
func TestValidateAndSanitizeRequest(t *testing.T) {
	cache := setupTestRedis(t)
	db, _ := setupTestDB(t)
	mockDB := &MockSearchDB{db: db}
	service := NewSearchService(nil, cache, mockDB)

	tests := []struct {
		name        string
		request     SearchRequest
		expectError bool
		check       func(t *testing.T, req *SearchRequest)
	}{
		{
			name: "Valid request",
			request: SearchRequest{
				Query:  "test query",
				Limit:  20,
				Offset: 0,
			},
			expectError: false,
			check: func(t *testing.T, req *SearchRequest) {
				assert.Equal(t, "test query", req.Query)
				assert.Equal(t, 20, req.Limit)
			},
		},
		{
			name: "Query too long gets truncated",
			request: SearchRequest{
				Query:  string(make([]byte, 1000)),
				Limit:  20,
				Offset: 0,
			},
			expectError: false,
			check: func(t *testing.T, req *SearchRequest) {
				assert.LessOrEqual(t, len(req.Query), 500)
			},
		},
		{
			name: "Negative limit gets default",
			request: SearchRequest{
				Query:  "test",
				Limit:  -1,
				Offset: 0,
			},
			expectError: false,
			check: func(t *testing.T, req *SearchRequest) {
				assert.Equal(t, 20, req.Limit)
			},
		},
		{
			name: "Limit exceeds max gets capped",
			request: SearchRequest{
				Query:  "test",
				Limit:  1000,
				Offset: 0,
			},
			expectError: false,
			check: func(t *testing.T, req *SearchRequest) {
				assert.Equal(t, 50, req.Limit)
			},
		},
		{
			name: "Offset exceeds max returns error",
			request: SearchRequest{
				Query:  "test",
				Limit:  20,
				Offset: 20000,
			},
			expectError: true,
		},
		{
			name: "SQL injection patterns removed",
			request: SearchRequest{
				Query:  "test; DROP TABLE articles;--",
				Limit:  20,
				Offset: 0,
			},
			expectError: false,
			check: func(t *testing.T, req *SearchRequest) {
				assert.NotContains(t, req.Query, ";")
				assert.NotContains(t, req.Query, "--")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateAndSanitizeRequest(&tt.request)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.check != nil {
					tt.check(t, &tt.request)
				}
			}
		})
	}
}

// TestCircuitBreaker tests circuit breaker behavior
func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(3, 5*time.Second)

	// Initially closed
	assert.False(t, cb.IsOpen())

	// Record failures
	cb.RecordFailure()
	assert.False(t, cb.IsOpen())
	cb.RecordFailure()
	assert.False(t, cb.IsOpen())
	cb.RecordFailure()

	// Should be open after threshold
	assert.True(t, cb.IsOpen())

	// Success should reset
	cb.RecordSuccess()
	assert.False(t, cb.IsOpen())
}

// TestCacheOperations tests cache get/set operations
func TestCacheOperations(t *testing.T) {
	cache := setupTestRedis(t)
	db, _ := setupTestDB(t)
	mockDB := &MockSearchDB{db: db}
	service := NewSearchService(nil, cache, mockDB)

	response := &SearchResponse{
		Articles: []models.Article{
			{ID: 1, Title: "Test Article"},
		},
		Total:  1,
		Limit:  20,
		Offset: 0,
		Source: "postgresql",
	}

	cacheKey := "test:cache:key"

	// Test caching
	err := service.cacheResult(cacheKey, response)
	assert.NoError(t, err)

	// Test retrieval
	cached, err := service.getFromCache(cacheKey)
	assert.NoError(t, err)
	assert.Equal(t, response.Total, cached.Total)
	assert.Equal(t, response.Source, cached.Source)
	assert.Len(t, cached.Articles, 1)
}

// TestCacheMiss tests cache miss scenario
func TestCacheMiss(t *testing.T) {
	cache := setupTestRedis(t)
	db, _ := setupTestDB(t)
	mockDB := &MockSearchDB{db: db}
	service := NewSearchService(nil, cache, mockDB)

	result, err := service.getFromCache("nonexistent:key")
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestInvalidateCache tests cache invalidation
func TestInvalidateCache(t *testing.T) {
	cache := setupTestRedis(t)
	db, _ := setupTestDB(t)
	mockDB := &MockSearchDB{db: db}
	service := NewSearchService(nil, cache, mockDB)

	ctx := context.Background()

	// Set test entries
	cache.Set(ctx, "search:test1", "value1", time.Minute)
	cache.Set(ctx, "search:test2", "value2", time.Minute)
	cache.Set(ctx, "other:test3", "value3", time.Minute)

	// Invalidate search cache
	err := service.InvalidateCache("search:*")
	assert.NoError(t, err)

	// Verify search keys are gone
	_, err1 := cache.Get(ctx, "search:test1").Result()
	_, err2 := cache.Get(ctx, "search:test2").Result()
	result3, err3 := cache.Get(ctx, "other:test3").Result()

	assert.Error(t, err1)
	assert.Error(t, err2)
	assert.NoError(t, err3)
	assert.Equal(t, "value3", result3)
}

// TestSearchWithPostgreSQL tests PostgreSQL fallback search
func TestSearchWithPostgreSQL(t *testing.T) {
	cache := setupTestRedis(t)
	db, mock := setupTestDB(t)
	mockDB := &MockSearchDB{db: db}
	service := NewSearchService(nil, cache, mockDB)

	// Setup mock expectations for search query
	rows := sqlmock.NewRows([]string{
		"id", "title", "slug", "content", "excerpt", "author_id", "category_id",
		"status", "published_at", "created_at", "updated_at", "view_count",
		"like_count", "language_code", "relevance_score",
	}).AddRow(
		1, "Test Article", "test-article", "Content", "Excerpt", 1, 1,
		"published", time.Now(), time.Now(), time.Now(), 100,
		10, "en", 0.5,
	)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	// Setup count query
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(countRows)

	req := SearchRequest{
		Query:  "test",
		Limit:  20,
		Offset: 0,
	}

	result, err := service.searchWithPostgreSQL(req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(1), result.Total)
	assert.Len(t, result.Articles, 1)
	assert.Equal(t, "Test Article", result.Articles[0].Title)
}

// TestSearchWithPostgreSQLFilters tests PostgreSQL search with filters
func TestSearchWithPostgreSQLFilters(t *testing.T) {
	cache := setupTestRedis(t)
	db, mock := setupTestDB(t)
	mockDB := &MockSearchDB{db: db}
	service := NewSearchService(nil, cache, mockDB)

	rows := sqlmock.NewRows([]string{
		"id", "title", "slug", "content", "excerpt", "author_id", "category_id",
		"status", "published_at", "created_at", "updated_at", "view_count",
		"like_count", "language_code", "relevance_score",
	}).AddRow(
		1, "Filtered Article", "filtered-article", "Content", "Excerpt", 123, 456,
		"published", time.Now(), time.Now(), time.Now(), 50,
		5, "en", 0.8,
	)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(countRows)

	authorID := uint64(123)
	categoryID := uint64(456)
	req := SearchRequest{
		Query:  "filtered",
		Limit:  20,
		Offset: 0,
		Filters: &SearchFilters{
			AuthorID:     &authorID,
			CategoryID:   &categoryID,
			LanguageCode: "en",
		},
	}

	result, err := service.searchWithPostgreSQL(req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Articles, 1)
}

// TestSearchFallbackOnMeiliSearchFailure tests fallback behavior
func TestSearchFallbackOnMeiliSearchFailure(t *testing.T) {
	// Test with nil indexer - should use PostgreSQL fallback directly
	cache := setupTestRedis(t)
	db, mock := setupTestDB(t)
	mockDB := &MockSearchDB{db: db}
	service := NewSearchService(nil, cache, mockDB) // nil indexer forces fallback

	// PostgreSQL fallback should work
	rows := sqlmock.NewRows([]string{
		"id", "title", "slug", "content", "excerpt", "author_id", "category_id",
		"status", "published_at", "created_at", "updated_at", "view_count",
		"like_count", "language_code", "relevance_score",
	}).AddRow(
		1, "Fallback Article", "fallback-article", "Content", "Excerpt", 1, 1,
		"published", time.Now(), time.Now(), time.Now(), 100,
		10, "en", 0.5,
	)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(countRows)

	req := SearchRequest{
		Query:  "test",
		Limit:  20,
		Offset: 0,
	}

	result, err := service.Search(req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "postgresql", result.Source)
}

// TestHealthCheck tests health check functionality
func TestHealthCheck(t *testing.T) {
	mockIndexer := &MockSearchIndexer{}
	cache := setupTestRedis(t)
	db, _ := setupTestDB(t)
	mockDB := &MockSearchDB{db: db}
	service := NewSearchService(mockIndexer, cache, mockDB)

	mockIndexer.On("HealthCheck").Return(nil)

	health := service.HealthCheck()

	assert.NotNil(t, health)
	assert.Contains(t, health, "meilisearch")
	assert.Contains(t, health, "cache")
	assert.Contains(t, health, "database")
	assert.Contains(t, health, "circuit_breaker")
	assert.Contains(t, health, "metrics")
}

// TestHealthCheckMeiliSearchDown tests health check when MeiliSearch is down
func TestHealthCheckMeiliSearchDown(t *testing.T) {
	mockIndexer := &MockSearchIndexer{}
	cache := setupTestRedis(t)
	db, _ := setupTestDB(t)
	mockDB := &MockSearchDB{db: db}
	service := NewSearchService(mockIndexer, cache, mockDB)

	mockIndexer.On("HealthCheck").Return(assert.AnError)

	health := service.HealthCheck()

	assert.Contains(t, health["meilisearch"].(string), "error")
	assert.Equal(t, "healthy", health["cache"])
	assert.Equal(t, "available", health["database"])
}

// TestConvertSearchDocumentToArticle tests document conversion
func TestConvertSearchDocumentToArticle(t *testing.T) {
	cache := setupTestRedis(t)
	db, _ := setupTestDB(t)
	mockDB := &MockSearchDB{db: db}
	service := NewSearchService(nil, cache, mockDB)

	publishedAt := time.Now().Unix()
	createdAt := time.Now().Add(-1 * time.Hour).Unix()

	doc := SearchDocument{
		ID:              "123",
		Title:           "Test Article",
		Content:         "Test content",
		Excerpt:         "Test excerpt",
		AuthorID:        456,
		CategoryID:      789,
		Tags:            []string{"tech", "news"},
		Status:          "published",
		PublishedAt:     publishedAt,
		CreatedAt:       createdAt,
		ViewCount:       100,
		LikeCount:       25,
		LanguageCode:    "en",
		MetaTitle:       "Test Meta Title",
		MetaDescription: "Test meta description",
	}

	article, err := service.convertSearchDocumentToArticle(doc)

	assert.NoError(t, err)
	assert.Equal(t, uint64(123), article.ID)
	assert.Equal(t, "Test Article", article.Title)
	assert.Equal(t, uint64(456), article.AuthorID)
	assert.Equal(t, uint64(789), article.CategoryID)
	assert.Len(t, article.Tags, 2)
}

// TestConvertSearchDocumentToArticleInvalidID tests conversion with invalid ID
func TestConvertSearchDocumentToArticleInvalidID(t *testing.T) {
	cache := setupTestRedis(t)
	db, _ := setupTestDB(t)
	mockDB := &MockSearchDB{db: db}
	service := NewSearchService(nil, cache, mockDB)

	doc := SearchDocument{
		ID:    "invalid-id",
		Title: "Test Article",
	}

	article, err := service.convertSearchDocumentToArticle(doc)

	assert.Error(t, err)
	assert.Nil(t, article)
	assert.Contains(t, err.Error(), "invalid article ID")
}

// TestGetSearchMetrics tests metrics retrieval
func TestGetSearchMetrics(t *testing.T) {
	metrics := GetSearchMetrics()

	assert.Contains(t, metrics, "total_searches")
	assert.Contains(t, metrics, "meilisearch_hits")
	assert.Contains(t, metrics, "postgresql_fallbacks")
	assert.Contains(t, metrics, "cache_hits")
	assert.Contains(t, metrics, "cache_misses")
	assert.Contains(t, metrics, "errors")
}

// TestIndexArticle tests article indexing
func TestIndexArticle(t *testing.T) {
	mockIndexer := &MockSearchIndexer{}
	cache := setupTestRedis(t)
	db, _ := setupTestDB(t)
	mockDB := &MockSearchDB{db: db}
	service := NewSearchService(mockIndexer, cache, mockDB)

	article := &models.Article{
		ID:         1,
		Title:      "Test Article",
		AuthorID:   456,
		CategoryID: 123,
	}

	mockIndexer.On("IndexArticle", article).Return(nil)

	err := service.IndexArticle(article)

	assert.NoError(t, err)
	mockIndexer.AssertExpectations(t)
}

// TestIndexArticleNoIndexer tests indexing when indexer is nil
func TestIndexArticleNoIndexer(t *testing.T) {
	cache := setupTestRedis(t)
	db, _ := setupTestDB(t)
	mockDB := &MockSearchDB{db: db}
	service := NewSearchService(nil, cache, mockDB)

	article := &models.Article{
		ID:    1,
		Title: "Test Article",
	}

	err := service.IndexArticle(article)

	assert.NoError(t, err) // Should not error, just skip
}

// TestRemoveArticle tests article removal from index
func TestRemoveArticle(t *testing.T) {
	mockIndexer := &MockSearchIndexer{}
	cache := setupTestRedis(t)
	db, _ := setupTestDB(t)
	mockDB := &MockSearchDB{db: db}
	service := NewSearchService(mockIndexer, cache, mockDB)

	mockIndexer.On("DeleteArticle", "123").Return(nil)

	err := service.RemoveArticle(context.Background(), 123)

	assert.NoError(t, err)
	mockIndexer.AssertExpectations(t)
}

// TestBuildMeiliSearchFilter tests MeiliSearch filter building
func TestBuildMeiliSearchFilter(t *testing.T) {
	cache := setupTestRedis(t)
	db, _ := setupTestDB(t)
	mockDB := &MockSearchDB{db: db}
	service := NewSearchService(nil, cache, mockDB)

	tests := []struct {
		name     string
		filters  *SearchFilters
		contains []string
	}{
		{
			name:     "Empty filters",
			filters:  &SearchFilters{},
			contains: []string{"status = published"},
		},
		{
			name: "Author filter",
			filters: &SearchFilters{
				AuthorID: func() *uint64 { id := uint64(123); return &id }(),
			},
			contains: []string{"status = published", "author_id = 123"},
		},
		{
			name: "Multiple filters",
			filters: &SearchFilters{
				AuthorID:     func() *uint64 { id := uint64(123); return &id }(),
				CategoryID:   func() *uint64 { id := uint64(456); return &id }(),
				LanguageCode: "en",
			},
			contains: []string{"author_id = 123", "category_id = 456", "language_code = en"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.buildMeiliSearchFilter(tt.filters)
			for _, expected := range tt.contains {
				assert.Contains(t, result, expected)
			}
		})
	}
}

// =============================================================================
// BENCHMARKS
// =============================================================================

func setupTestRedisForBenchmark(b *testing.B) *redis.Client {
	mr, err := miniredis.Run()
	if err != nil {
		b.Fatalf("Failed to start miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	b.Cleanup(func() {
		client.Close()
		mr.Close()
	})

	return client
}

func BenchmarkGenerateCacheKey(b *testing.B) {
	cache := setupTestRedisForBenchmark(b)
	service := NewSearchService(nil, cache, nil)

	req := SearchRequest{
		Query:  "benchmark query",
		Limit:  20,
		Offset: 0,
		Filters: &SearchFilters{
			AuthorID:     func() *uint64 { id := uint64(123); return &id }(),
			CategoryID:   func() *uint64 { id := uint64(456); return &id }(),
			LanguageCode: "en",
			Tags:         []string{"tech", "news"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.generateCacheKey(req)
	}
}

func BenchmarkConvertSearchDocument(b *testing.B) {
	cache := setupTestRedisForBenchmark(b)
	service := NewSearchService(nil, cache, nil)

	doc := SearchDocument{
		ID:           "123",
		Title:        "Benchmark Article",
		Content:      "Benchmark content",
		Excerpt:      "Benchmark excerpt",
		AuthorID:     456,
		CategoryID:   789,
		Tags:         []string{"benchmark", "performance"},
		Status:       "published",
		PublishedAt:  time.Now().Unix(),
		CreatedAt:    time.Now().Unix(),
		ViewCount:    1000,
		LikeCount:    50,
		LanguageCode: "en",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.convertSearchDocumentToArticle(doc)
	}
}
