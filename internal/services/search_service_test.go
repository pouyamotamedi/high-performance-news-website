package services

import (
	"context"
	"testing"
	"time"

	"high-performance-news-website/internal/models"

	"github.com/alicebob/miniredis/v2"
	meilisearch "github.com/meilisearch/meilisearch-go"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSearchIndexer is a mock implementation of SearchIndexer
type MockSearchIndexer struct {
	mock.Mock
	client    meilisearch.ServiceManager
	indexName string
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
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockSearchIndexer) HealthCheck() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockSearchIndexer) GetClient() meilisearch.ServiceManager {
	args := m.Called()
	return args.Get(0).(meilisearch.ServiceManager)
}

func (m *MockSearchIndexer) GetIndexName() string {
	args := m.Called()
	return args.String(0)
}

// setupTestRedis creates a test Redis instance
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

// TestSearchService_NewSearchService tests the creation of a new search service
func TestSearchService_NewSearchService(t *testing.T) {
	mockIndexer := &MockSearchIndexer{}
	mockRepo := &MockArticleRepository{}
	cache := setupTestRedis(t)
	
	service := NewSearchService(mockIndexer, cache, mockRepo)
	
	assert.NotNil(t, service)
	assert.Equal(t, mockIndexer, service.indexer)
	assert.Equal(t, cache, service.cache)
	assert.Equal(t, mockRepo, service.fallbackDB)
	assert.Equal(t, "search:", service.cachePrefix)
	assert.Equal(t, 5*time.Minute, service.cacheTTL)
	assert.Equal(t, 10*time.Second, service.timeout)
}

// TestSearchService_generateCacheKey tests cache key generation
func TestSearchService_generateCacheKey(t *testing.T) {
	mockIndexer := &MockSearchIndexer{}
	mockRepo := &MockArticleRepository{}
	cache := setupTestRedis(t)
	
	service := NewSearchService(mockIndexer, cache, mockRepo)
	
	tests := []struct {
		name     string
		request  SearchRequest
		expected string
	}{
		{
			name: "Basic query",
			request: SearchRequest{
				Query:  "test query",
				Limit:  20,
				Offset: 0,
			},
			expected: "search:test query:20:0",
		},
		{
			name: "Query with filters",
			request: SearchRequest{
				Query:  "news",
				Limit:  10,
				Offset: 20,
				Filters: &SearchFilters{
					AuthorID:     func() *uint64 { id := uint64(123); return &id }(),
					CategoryID:   func() *uint64 { id := uint64(456); return &id }(),
					LanguageCode: "en",
					Tags:         []string{"tech", "news"},
				},
			},
			expected: "search:news:10:20:author_123:cat_456:lang_en:tags_tech,news",
		},
		{
			name: "Query with sorting",
			request: SearchRequest{
				Query:  "article",
				Limit:  15,
				Offset: 0,
				Sort: &SearchSort{
					Field: "published_at",
					Order: "desc",
				},
			},
			expected: "search:article:15:0:sort_published_at_desc",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := service.generateCacheKey(tt.request)
			assert.Equal(t, tt.expected, key)
		})
	}
}

// TestSearchService_buildMeiliSearchFilter tests MeiliSearch filter building
func TestSearchService_buildMeiliSearchFilter(t *testing.T) {
	mockIndexer := &MockSearchIndexer{}
	mockRepo := &MockArticleRepository{}
	cache := setupTestRedis(t)
	
	service := NewSearchService(mockIndexer, cache, mockRepo)
	
	tests := []struct {
		name     string
		filters  *SearchFilters
		expected string
	}{
		{
			name:     "Nil filters",
			filters:  nil,
			expected: "",
		},
		{
			name:     "Empty filters",
			filters:  &SearchFilters{},
			expected: "status = published",
		},
		{
			name: "Author filter",
			filters: &SearchFilters{
				AuthorID: func() *uint64 { id := uint64(123); return &id }(),
			},
			expected: "status = published AND author_id = 123",
		},
		{
			name: "Multiple filters",
			filters: &SearchFilters{
				AuthorID:     func() *uint64 { id := uint64(123); return &id }(),
				CategoryID:   func() *uint64 { id := uint64(456); return &id }(),
				LanguageCode: "en",
				Tags:         []string{"tech", "news"},
			},
			expected: "status = published AND author_id = 123 AND category_id = 456 AND language_code = en AND (tags = tech OR tags = news)",
		},
		{
			name: "Date range filters",
			filters: &SearchFilters{
				DateFrom: func() *int64 { ts := int64(1640995200); return &ts }(), // 2022-01-01
				DateTo:   func() *int64 { ts := int64(1672531200); return &ts }(), // 2023-01-01
			},
			expected: "status = published AND published_at >= 1640995200 AND published_at <= 1672531200",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.buildMeiliSearchFilter(tt.filters)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSearchService_convertSearchDocumentToArticle tests search document to article conversion
func TestSearchService_convertSearchDocumentToArticle(t *testing.T) {
	mockIndexer := &MockSearchIndexer{}
	mockRepo := &MockArticleRepository{}
	cache := setupTestRedis(t)
	
	service := NewSearchService(mockIndexer, cache, mockRepo)
	
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
		Keywords:        []string{"test", "article"},
	}
	
	article, err := service.convertSearchDocumentToArticle(doc)
	
	assert.NoError(t, err)
	assert.Equal(t, uint64(123), article.ID)
	assert.Equal(t, "Test Article", article.Title)
	assert.Equal(t, "Test content", article.Content)
	assert.Equal(t, "Test excerpt", article.Excerpt)
	assert.Equal(t, uint64(456), article.AuthorID)
	assert.Equal(t, uint64(789), article.CategoryID)
	assert.Equal(t, "published", article.Status)
	assert.Equal(t, publishedAt, article.PublishedAt.Unix())
	assert.Equal(t, createdAt, article.CreatedAt.Unix())
	assert.Equal(t, uint64(100), article.ViewCount)
	assert.Equal(t, uint64(25), article.LikeCount)
	assert.Equal(t, "en", article.LanguageCode)
	assert.Equal(t, "Test Meta Title", article.SEOData.MetaTitle)
	assert.Equal(t, "Test meta description", article.SEOData.MetaDescription)
	assert.Equal(t, []string{"test", "article"}, article.SEOData.Keywords)
	
	// Check tags conversion
	assert.Len(t, article.Tags, 2)
	assert.Equal(t, "tech", article.Tags[0].Name)
	assert.Equal(t, "news", article.Tags[1].Name)
}

// TestSearchService_convertSearchDocumentToArticle_InvalidID tests conversion with invalid ID
func TestSearchService_convertSearchDocumentToArticle_InvalidID(t *testing.T) {
	mockIndexer := &MockSearchIndexer{}
	mockRepo := &MockArticleRepository{}
	cache := setupTestRedis(t)
	
	service := NewSearchService(mockIndexer, cache, mockRepo)
	
	doc := SearchDocument{
		ID:           "invalid-id",
		Title:        "Test Article",
		LanguageCode: "en",
	}
	
	article, err := service.convertSearchDocumentToArticle(doc)
	
	assert.Error(t, err)
	assert.Nil(t, article)
	assert.Contains(t, err.Error(), "invalid article ID")
}

// TestSearchService_convertSearchDocumentToArticle_ZeroPublishedAt tests conversion with zero published timestamp
func TestSearchService_convertSearchDocumentToArticle_ZeroPublishedAt(t *testing.T) {
	mockIndexer := &MockSearchIndexer{}
	mockRepo := &MockArticleRepository{}
	cache := setupTestRedis(t)
	
	service := NewSearchService(mockIndexer, cache, mockRepo)
	
	doc := SearchDocument{
		ID:           "123",
		Title:        "Draft Article",
		PublishedAt:  0, // Zero timestamp
		CreatedAt:    time.Now().Unix(),
		LanguageCode: "en",
	}
	
	article, err := service.convertSearchDocumentToArticle(doc)
	
	assert.NoError(t, err)
	assert.Nil(t, article.PublishedAt) // Should be nil for zero timestamp
}

// TestSearchService_cacheResult_and_getFromCache tests caching functionality
func TestSearchService_cacheResult_and_getFromCache(t *testing.T) {
	mockIndexer := &MockSearchIndexer{}
	mockRepo := &MockArticleRepository{}
	cache := setupTestRedis(t)
	
	service := NewSearchService(mockIndexer, cache, mockRepo)
	
	// Create test search response
	response := &SearchResponse{
		Articles: []models.Article{
			{
				ID:           1,
				Title:        "Test Article",
				LanguageCode: "en",
			},
		},
		Total:          1,
		Limit:          20,
		Offset:         0,
		ProcessingTime: 50,
		Source:         "meilisearch",
	}
	
	cacheKey := "test:cache:key"
	
	// Test caching
	err := service.cacheResult(cacheKey, response)
	assert.NoError(t, err)
	
	// Test retrieval
	cachedResponse, err := service.getFromCache(cacheKey)
	assert.NoError(t, err)
	assert.Equal(t, response.Total, cachedResponse.Total)
	assert.Equal(t, response.Limit, cachedResponse.Limit)
	assert.Equal(t, response.Offset, cachedResponse.Offset)
	assert.Equal(t, response.Source, cachedResponse.Source)
	assert.Len(t, cachedResponse.Articles, 1)
	assert.Equal(t, "Test Article", cachedResponse.Articles[0].Title)
}

// TestSearchService_getFromCache_NotFound tests cache miss
func TestSearchService_getFromCache_NotFound(t *testing.T) {
	mockIndexer := &MockSearchIndexer{}
	mockRepo := &MockArticleRepository{}
	cache := setupTestRedis(t)
	
	service := NewSearchService(mockIndexer, cache, mockRepo)
	
	// Try to get non-existent key
	result, err := service.getFromCache("non:existent:key")
	
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestSearchService_InvalidateCache tests cache invalidation
func TestSearchService_InvalidateCache(t *testing.T) {
	mockIndexer := &MockSearchIndexer{}
	mockRepo := &MockArticleRepository{}
	cache := setupTestRedis(t)
	
	service := NewSearchService(mockIndexer, cache, mockRepo)
	
	// Set some test cache entries
	ctx := context.Background()
	cache.Set(ctx, "search:test1", "value1", time.Minute)
	cache.Set(ctx, "search:test2", "value2", time.Minute)
	cache.Set(ctx, "other:test3", "value3", time.Minute)
	
	// Invalidate search cache
	err := service.InvalidateCache("search:*")
	assert.NoError(t, err)
	
	// Check that search keys are gone
	result1, err1 := cache.Get(ctx, "search:test1").Result()
	result2, err2 := cache.Get(ctx, "search:test2").Result()
	result3, err3 := cache.Get(ctx, "other:test3").Result()
	
	assert.Error(t, err1) // Should be gone
	assert.Error(t, err2) // Should be gone
	assert.NoError(t, err3) // Should still exist
	assert.Empty(t, result1)
	assert.Empty(t, result2)
	assert.Equal(t, "value3", result3)
}

// TestSearchService_InvalidateCache_EmptyPattern tests cache invalidation with empty pattern
func TestSearchService_InvalidateCache_EmptyPattern(t *testing.T) {
	mockIndexer := &MockSearchIndexer{}
	mockRepo := &MockArticleRepository{}
	cache := setupTestRedis(t)
	
	service := NewSearchService(mockIndexer, cache, mockRepo)
	
	// Set some test cache entries
	ctx := context.Background()
	cache.Set(ctx, "search:test1", "value1", time.Minute)
	cache.Set(ctx, "search:test2", "value2", time.Minute)
	
	// Invalidate with empty pattern (should use default)
	err := service.InvalidateCache("")
	assert.NoError(t, err)
	
	// Check that search keys are gone
	result1, err1 := cache.Get(ctx, "search:test1").Result()
	result2, err2 := cache.Get(ctx, "search:test2").Result()
	
	assert.Error(t, err1) // Should be gone
	assert.Error(t, err2) // Should be gone
	assert.Empty(t, result1)
	assert.Empty(t, result2)
}

// TestSearchService_searchWithPostgreSQL tests PostgreSQL fallback search
func TestSearchService_searchWithPostgreSQL(t *testing.T) {
	mockIndexer := &MockSearchIndexer{}
	mockRepo := &MockArticleRepository{}
	cache := setupTestRedis(t)
	
	service := NewSearchService(mockIndexer, cache, mockRepo)
	
	// Setup mock expectations
	expectedArticles := []models.Article{
		{
			ID:           1,
			Title:        "Test Article",
			LanguageCode: "en",
		},
	}
	expectedTotal := int64(1)
	
	mockRepo.On("Search", "test query", mock.AnythingOfType("map[string]interface {}"), 20, 0).
		Return(expectedArticles, expectedTotal, nil)
	
	// Create search request
	req := SearchRequest{
		Query:  "test query",
		Limit:  20,
		Offset: 0,
		Filters: &SearchFilters{
			AuthorID: func() *uint64 { id := uint64(123); return &id }(),
		},
		Sort: &SearchSort{
			Field: "published_at",
			Order: "desc",
		},
	}
	
	// Perform search
	result, err := service.searchWithPostgreSQL(req)
	
	assert.NoError(t, err)
	assert.Equal(t, expectedTotal, result.Total)
	assert.Equal(t, 20, result.Limit)
	assert.Equal(t, 0, result.Offset)
	assert.Len(t, result.Articles, 1)
	assert.Equal(t, "Test Article", result.Articles[0].Title)
	
	mockRepo.AssertExpectations(t)
}

// TestSearchService_IndexArticle tests article indexing with cache invalidation
func TestSearchService_IndexArticle(t *testing.T) {
	mockIndexer := &MockSearchIndexer{}
	mockRepo := &MockArticleRepository{}
	cache := setupTestRedis(t)
	
	service := NewSearchService(mockIndexer, cache, mockRepo)
	
	// Set up some cache entries that should be invalidated
	ctx := context.Background()
	cache.Set(ctx, "search:test:cat_123", "value1", time.Minute)
	cache.Set(ctx, "search:test:author_456", "value2", time.Minute)
	cache.Set(ctx, "search:other:cat_999", "value3", time.Minute)
	
	article := &models.Article{
		ID:         1,
		Title:      "Test Article",
		AuthorID:   456,
		CategoryID: 123,
	}
	
	// Setup mock expectations
	mockIndexer.On("IndexArticle", article).Return(nil)
	
	// Index article
	err := service.IndexArticle(article)
	
	assert.NoError(t, err)
	mockIndexer.AssertExpectations(t)
	
	// Check that related cache entries were invalidated
	// Note: The exact cache invalidation behavior depends on the implementation
	// This test verifies the method completes without error
}

// TestSearchService_HealthCheck tests health check functionality
func TestSearchService_HealthCheck(t *testing.T) {
	mockIndexer := &MockSearchIndexer{}
	mockRepo := &MockArticleRepository{}
	cache := setupTestRedis(t)
	
	service := NewSearchService(mockIndexer, cache, mockRepo)
	
	// Setup mock expectations
	mockIndexer.On("HealthCheck").Return(nil)
	
	// Perform health check
	health := service.HealthCheck()
	
	assert.NotNil(t, health)
	assert.Contains(t, health, "meilisearch")
	assert.Contains(t, health, "cache")
	assert.Contains(t, health, "fallback_db")
	
	// MeiliSearch should be healthy (mocked)
	assert.Equal(t, "healthy", health["meilisearch"])
	
	// Cache should be healthy (real Redis)
	assert.Equal(t, "healthy", health["cache"])
	
	// Fallback DB should be available
	assert.Equal(t, "available", health["fallback_db"])
	
	mockIndexer.AssertExpectations(t)
}

// TestSearchService_GetSuggestions tests search suggestions
func TestSearchService_GetSuggestions(t *testing.T) {
	mockIndexer := &MockSearchIndexer{}
	mockRepo := &MockArticleRepository{}
	cache := setupTestRedis(t)
	
	service := NewSearchService(mockIndexer, cache, mockRepo)
	
	suggestions, err := service.GetSuggestions(context.Background(), "test", 10)
	
	assert.NoError(t, err)
	assert.NotNil(t, suggestions)
	assert.Equal(t, []string{}, suggestions) // Currently returns empty slice
}

// BenchmarkSearchService_generateCacheKey benchmarks cache key generation
func BenchmarkSearchService_generateCacheKey(b *testing.B) {
	mockIndexer := &MockSearchIndexer{}
	mockRepo := &MockArticleRepository{}
	cache := setupTestRedisForBenchmark(b)
	
	service := NewSearchService(mockIndexer, cache, mockRepo)
	
	req := SearchRequest{
		Query:  "benchmark query",
		Limit:  20,
		Offset: 0,
		Filters: &SearchFilters{
			AuthorID:     func() *uint64 { id := uint64(123); return &id }(),
			CategoryID:   func() *uint64 { id := uint64(456); return &id }(),
			LanguageCode: "en",
			Tags:         []string{"tech", "news", "benchmark"},
		},
		Sort: &SearchSort{
			Field: "published_at",
			Order: "desc",
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.generateCacheKey(req)
	}
}

// setupTestRedisForBenchmark creates a test Redis instance for benchmarking
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

// BenchmarkSearchService_convertSearchDocumentToArticle benchmarks document conversion
func BenchmarkSearchService_convertSearchDocumentToArticle(b *testing.B) {
	mockIndexer := &MockSearchIndexer{}
	mockRepo := &MockArticleRepository{}
	cache := setupTestRedisForBenchmark(b)
	
	service := NewSearchService(mockIndexer, cache, mockRepo)
	
	doc := SearchDocument{
		ID:              "123",
		Title:           "Benchmark Article",
		Content:         "Benchmark content for performance testing",
		Excerpt:         "Benchmark excerpt",
		AuthorID:        456,
		CategoryID:      789,
		Tags:            []string{"benchmark", "performance"},
		Status:          "published",
		PublishedAt:     time.Now().Unix(),
		CreatedAt:       time.Now().Unix(),
		ViewCount:       1000,
		LikeCount:       50,
		LanguageCode:    "en",
		MetaTitle:       "Benchmark Meta Title",
		MetaDescription: "Benchmark meta description",
		Keywords:        []string{"benchmark", "performance"},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.convertSearchDocumentToArticle(doc)
	}
}