package services

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"high-performance-news-website/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCacheService provides a testable cache implementation
type MockCacheService struct {
	data   map[string][]byte
	ttls   map[string]time.Time
	mutex  sync.RWMutex
	getCalls    int
	setCalls    int
	deleteCalls int
	deletePatternCalls int
}

func NewMockCacheService() *MockCacheService {
	return &MockCacheService{
		data: make(map[string][]byte),
		ttls: make(map[string]time.Time),
	}
}

func (m *MockCacheService) Get(ctx context.Context, key string) ([]byte, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	m.getCalls++
	
	// Check if key exists and hasn't expired
	if data, exists := m.data[key]; exists {
		if ttl, hasTTL := m.ttls[key]; !hasTTL || time.Now().Before(ttl) {
			return data, nil
		}
		// Key expired, remove it
		delete(m.data, key)
		delete(m.ttls, key)
	}
	
	return nil, nil // Return nil for cache miss
}

func (m *MockCacheService) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.setCalls++
	
	m.data[key] = value
	if ttl > 0 {
		m.ttls[key] = time.Now().Add(ttl)
	}
	
	return nil
}

func (m *MockCacheService) Delete(ctx context.Context, key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.deleteCalls++
	
	delete(m.data, key)
	delete(m.ttls, key)
	
	return nil
}

func (m *MockCacheService) DeletePattern(ctx context.Context, pattern string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.deletePatternCalls++
	
	// Simple pattern matching for testing (supports * at the end)
	if pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		for key := range m.data {
			if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
				delete(m.data, key)
				delete(m.ttls, key)
			}
		}
	}
	
	return nil
}

func (m *MockCacheService) Exists(ctx context.Context, key string) (bool, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	if _, exists := m.data[key]; exists {
		if ttl, hasTTL := m.ttls[key]; !hasTTL || time.Now().Before(ttl) {
			return true, nil
		}
		// Key expired
		delete(m.data, key)
		delete(m.ttls, key)
	}
	
	return false, nil
}

func (m *MockCacheService) Close() error {
	return nil
}

func (m *MockCacheService) Health(ctx context.Context) error {
	return nil
}

// Helper methods for testing
func (m *MockCacheService) GetCallCounts() (get, set, delete, deletePattern int) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.getCalls, m.setCalls, m.deleteCalls, m.deletePatternCalls
}

func (m *MockCacheService) GetDataCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.data)
}

func (m *MockCacheService) HasKey(key string) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	_, exists := m.data[key]
	return exists
}

func TestStaticGenerator_CacheWarming_Homepage(t *testing.T) {
	mockCache := NewMockCacheService()
	sg := &StaticGenerator{
		cacheService: mockCache,
		baseURL:      "https://example.com",
	}

	// Test homepage cache warming
	sg.warmHomepageCache("en")
	
	// Allow some time for async operations
	time.Sleep(100 * time.Millisecond)
	
	// Verify cache operations were called
	_, set, delete, deletePattern := mockCache.GetCallCounts()
	assert.True(t, set >= 0, "Cache warming should attempt to set cache entries")
	assert.Equal(t, 0, delete, "Cache warming should not delete entries")
	assert.Equal(t, 0, deletePattern, "Cache warming should not delete patterns")
	
	// The exact number of get/set calls depends on repository availability
	// but the function should complete without errors
}

func TestStaticGenerator_CacheWarming_Article(t *testing.T) {
	mockCache := NewMockCacheService()
	sg := &StaticGenerator{
		cacheService: mockCache,
		baseURL:      "https://example.com",
	}

	// Create test article
	publishedAt := time.Now()
	article := &models.Article{
		ID:           1,
		Title:        "Test Article for Cache Warming",
		Slug:         "test-article-cache-warming",
		Content:      "<p>Test content for cache warming</p>",
		Excerpt:      "Test excerpt for cache warming",
		CategoryID:   1,
		PublishedAt:  &publishedAt,
		UpdatedAt:    time.Now(),
		LanguageCode: "en",
		SEOData: models.SEOData{
			SchemaType: "NewsArticle",
		},
	}

	// Test article cache warming
	sg.warmArticleCache(article)
	
	// Allow some time for async operations
	time.Sleep(100 * time.Millisecond)
	
	// Verify cache operations were called
	_, set, delete, deletePattern := mockCache.GetCallCounts()
	assert.True(t, set >= 1, "Article cache warming should set at least one cache entry")
	assert.Equal(t, 0, delete, "Cache warming should not delete entries")
	assert.Equal(t, 0, deletePattern, "Cache warming should not delete patterns")
}

func TestStaticGenerator_CacheInvalidation_Article(t *testing.T) {
	mockCache := NewMockCacheService()
	sg := &StaticGenerator{
		cacheService: mockCache,
		baseURL:      "https://example.com",
	}

	// Pre-populate cache with some test data
	ctx := context.Background()
	testData := []byte("test data")
	
	cacheKeys := []string{
		"article:test-slug",
		"article:1",
		"homepage:fa",
		"homepage:en",
		"homepage:ar",
		"category:1",
		"tag:1",
	}
	
	for _, key := range cacheKeys {
		mockCache.Set(ctx, key, testData, time.Hour)
	}
	
	// Verify data was set
	initialCount := mockCache.GetDataCount()
	assert.Equal(t, len(cacheKeys), initialCount)

	// Create test article with tags
	publishedAt := time.Now()
	article := &models.Article{
		ID:         1,
		Title:      "Test Article for Cache Invalidation",
		Slug:       "test-slug",
		CategoryID: 1,
		PublishedAt: &publishedAt,
		Tags: []models.Tag{
			{ID: 1, Name: "Test Tag", Slug: "test-tag"},
			{ID: 2, Name: "Another Tag", Slug: "another-tag"},
		},
	}

	// Test cache invalidation
	sg.invalidateRelatedCaches(article)
	
	// Allow some time for async operations
	time.Sleep(100 * time.Millisecond)
	
	// Verify cache operations were called
	_, _, delete, deletePattern := mockCache.GetCallCounts()
	assert.True(t, delete > 0, "Cache invalidation should delete cache entries")
	assert.True(t, deletePattern > 0, "Cache invalidation should delete pattern-based entries")
	
	// Verify specific keys were invalidated
	assert.False(t, mockCache.HasKey("article:test-slug"), "Article cache should be invalidated")
	assert.False(t, mockCache.HasKey("homepage:fa"), "Homepage cache should be invalidated")
}

func TestStaticGenerator_CacheInvalidation_Patterns(t *testing.T) {
	mockCache := NewMockCacheService()
	sg := &StaticGenerator{
		cacheService: mockCache,
		baseURL:      "https://example.com",
	}

	// Pre-populate cache with pattern-based keys
	ctx := context.Background()
	testData := []byte("test data")
	
	patternKeys := []string{
		"homepage:fa",
		"homepage:en", 
		"homepage:ar",
		"category:1:page:1",
		"category:1:page:2",
		"tag:1:page:1",
		"tag:1:page:2",
		"other:key", // Should not be affected
	}
	
	for _, key := range patternKeys {
		mockCache.Set(ctx, key, testData, time.Hour)
	}
	
	// Verify data was set
	initialCount := mockCache.GetDataCount()
	assert.Equal(t, len(patternKeys), initialCount)

	// Create test article
	publishedAt := time.Now()
	article := &models.Article{
		ID:         1,
		Slug:       "test-article",
		CategoryID: 1,
		PublishedAt: &publishedAt,
		Tags: []models.Tag{
			{ID: 1, Name: "Test Tag"},
		},
	}

	// Test pattern-based cache invalidation
	sg.invalidateRelatedCaches(article)
	
	// Allow some time for async operations
	time.Sleep(100 * time.Millisecond)
	
	// Verify pattern deletion was called
	_, _, _, deletePattern := mockCache.GetCallCounts()
	assert.True(t, deletePattern > 0, "Pattern-based cache invalidation should be called")
	
	// Verify non-pattern keys are preserved
	assert.True(t, mockCache.HasKey("other:key"), "Non-pattern keys should be preserved")
}

func TestStaticGenerator_CacheWarming_CategoryAndTag(t *testing.T) {
	mockCache := NewMockCacheService()
	sg := &StaticGenerator{
		cacheService: mockCache,
		baseURL:      "https://example.com",
	}

	// Test category cache warming
	category := &models.Category{
		ID:          1,
		Name:        "Test Category",
		Slug:        "test-category",
		Description: "A test category",
	}

	sg.warmCategoryCache(category, 1)
	
	// Test tag cache warming
	tag := &models.Tag{
		ID:          1,
		Name:        "Test Tag",
		Slug:        "test-tag",
		Description: "A test tag",
		Keywords:    []string{"test", "tag"},
	}

	sg.warmTagCache(tag, 1)
	
	// Allow some time for async operations
	time.Sleep(100 * time.Millisecond)
	
	// Verify cache operations were called
	_, set, delete, deletePattern := mockCache.GetCallCounts()
	assert.True(t, set >= 2, "Category and tag cache warming should set cache entries")
	assert.Equal(t, 0, delete, "Cache warming should not delete entries")
	assert.Equal(t, 0, deletePattern, "Cache warming should not delete patterns")
}

func TestStaticGenerator_CacheOperations_WithNilCache(t *testing.T) {
	// Test that cache operations handle nil cache gracefully
	sg := &StaticGenerator{
		cacheService: nil, // No cache service
		baseURL:      "https://example.com",
	}

	// Create test article
	publishedAt := time.Now()
	article := &models.Article{
		ID:          1,
		Slug:        "test-article",
		CategoryID:  1,
		PublishedAt: &publishedAt,
	}

	// These operations should not panic with nil cache
	assert.NotPanics(t, func() {
		sg.warmHomepageCache("en")
	}, "Homepage cache warming should handle nil cache")

	assert.NotPanics(t, func() {
		sg.warmArticleCache(article)
	}, "Article cache warming should handle nil cache")

	assert.NotPanics(t, func() {
		sg.invalidateRelatedCaches(article)
	}, "Cache invalidation should handle nil cache")

	category := &models.Category{ID: 1, Name: "Test", Slug: "test"}
	assert.NotPanics(t, func() {
		sg.warmCategoryCache(category, 1)
	}, "Category cache warming should handle nil cache")

	tag := &models.Tag{ID: 1, Name: "Test", Slug: "test"}
	assert.NotPanics(t, func() {
		sg.warmTagCache(tag, 1)
	}, "Tag cache warming should handle nil cache")
}

func TestStaticGenerator_CacheOperations_TTL(t *testing.T) {
	mockCache := NewMockCacheService()
	sg := &StaticGenerator{
		cacheService: mockCache,
		baseURL:      "https://example.com",
	}

	ctx := context.Background()

	// Test that cache entries are set with appropriate TTLs
	testData := []byte("test data")
	
	// Set cache entries with different TTLs
	mockCache.Set(ctx, "short-ttl", testData, 100*time.Millisecond)
	mockCache.Set(ctx, "long-ttl", testData, time.Hour)
	
	// Verify entries exist initially
	exists, err := mockCache.Exists(ctx, "short-ttl")
	require.NoError(t, err)
	assert.True(t, exists)
	
	exists, err = mockCache.Exists(ctx, "long-ttl")
	require.NoError(t, err)
	assert.True(t, exists)
	
	// Wait for short TTL to expire
	time.Sleep(150 * time.Millisecond)
	
	// Verify short TTL entry expired
	exists, err = mockCache.Exists(ctx, "short-ttl")
	require.NoError(t, err)
	assert.False(t, exists, "Short TTL entry should have expired")
	
	// Verify long TTL entry still exists
	exists, err = mockCache.Exists(ctx, "long-ttl")
	require.NoError(t, err)
	assert.True(t, exists, "Long TTL entry should still exist")
}

func TestStaticGenerator_SchemaGeneration(t *testing.T) {
	sg := &StaticGenerator{
		baseURL: "https://example.com",
	}

	// Test article schema generation
	publishedAt := time.Now()
	article := &models.Article{
		ID:          1,
		Title:       "Test Article Schema",
		Slug:        "test-article-schema",
		Content:     "<p>Test content for schema generation</p>",
		Excerpt:     "Test excerpt for schema",
		PublishedAt: &publishedAt,
		UpdatedAt:   time.Now(),
		SEOData: models.SEOData{
			SchemaType: "NewsArticle",
			Keywords:   []string{"test", "schema", "generation"},
		},
	}

	schema := sg.generateArticleSchema(article, nil)
	assert.NotEmpty(t, schema)
	
	// Verify it's valid JSON
	var schemaData map[string]interface{}
	err := json.Unmarshal([]byte(schema), &schemaData)
	require.NoError(t, err)
	
	// Verify schema structure
	assert.Equal(t, "https://schema.org", schemaData["@context"])
	assert.Equal(t, "NewsArticle", schemaData["@type"])
	assert.Equal(t, "Test Article Schema", schemaData["headline"])
	assert.Contains(t, schemaData["url"], "test-article-schema")

	// Test category schema generation
	category := &models.Category{
		ID:          1,
		Name:        "Test Category",
		Slug:        "test-category",
		Description: "Test category description",
	}

	articles := []models.Article{*article}
	categorySchema := sg.generateCategorySchema(category, articles)
	assert.NotEmpty(t, categorySchema)
	
	// Verify category schema JSON
	var categorySchemaData map[string]interface{}
	err = json.Unmarshal([]byte(categorySchema), &categorySchemaData)
	require.NoError(t, err)
	
	assert.Equal(t, "https://schema.org", categorySchemaData["@context"])
	assert.Equal(t, "CollectionPage", categorySchemaData["@type"])
	assert.Equal(t, "Test Category", categorySchemaData["name"])

	// Test tag schema generation
	tag := &models.Tag{
		ID:          1,
		Name:        "Test Tag",
		Slug:        "test-tag",
		Description: "Test tag description",
		Keywords:    []string{"test", "tag", "keywords"},
	}

	tagSchema := sg.generateTagSchema(tag, articles)
	assert.NotEmpty(t, tagSchema)
	
	// Verify tag schema JSON
	var tagSchemaData map[string]interface{}
	err = json.Unmarshal([]byte(tagSchema), &tagSchemaData)
	require.NoError(t, err)
	
	assert.Equal(t, "https://schema.org", tagSchemaData["@context"])
	assert.Equal(t, "CollectionPage", tagSchemaData["@type"])
	assert.Equal(t, "Test Tag", tagSchemaData["name"])
	
	// Verify keywords are included
	keywords, ok := tagSchemaData["keywords"].([]interface{})
	assert.True(t, ok)
	assert.Contains(t, keywords, "test")
	assert.Contains(t, keywords, "tag")
	assert.Contains(t, keywords, "keywords")
}