package repositories

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/pkg/cache"
)

// MockCacheService implements cache.CacheService for testing
type MockCacheService struct {
	mock.Mock
}

func (m *MockCacheService) Get(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockCacheService) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockCacheService) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockCacheService) DeletePattern(ctx context.Context, pattern string) error {
	args := m.Called(ctx, pattern)
	return args.Error(0)
}

func (m *MockCacheService) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockCacheService) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockCacheService) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Test helper functions
func createTestArticle() *models.Article {
	now := time.Now()
	return &models.Article{
		Title:      "Test Article",
		Slug:       "test-article",
		Content:    "This is test content for the article.",
		Excerpt:    "Test excerpt",
		AuthorID:   1,
		CategoryID: 1,
		Status:     "published",
		PublishedAt: &now,
		SEOData: models.SEOData{
			MetaTitle:       "Test Article - Meta Title",
			MetaDescription: "Test meta description",
			SchemaType:      "NewsArticle",
		},
	}
}

func setupTestRepository(t *testing.T) (*ArticleRepository, *MockCacheService, func()) {
	// Create temporary directory for static files
	tempDir, err := os.MkdirTemp("", "article_repo_test")
	require.NoError(t, err)
	
	mockCache := &MockCacheService{}
	
	// For unit tests, we'll create a repository without database
	// This means some tests will be limited to validation only
	repo := NewArticleRepository(nil, mockCache, tempDir)
	
	cleanup := func() {
		os.RemoveAll(tempDir)
	}
	
	return repo, mockCache, cleanup
}

func TestArticleRepository_Create(t *testing.T) {
	_, mockCache, cleanup := setupTestRepository(t)
	defer cleanup()
	
	article := createTestArticle()
	
	// Mock cache invalidation calls
	mockCache.On("DeletePattern", mock.Anything, "homepage:*").Return(nil)
	mockCache.On("DeletePattern", mock.Anything, "category:*").Return(nil)
	mockCache.On("DeletePattern", mock.Anything, "tag:*").Return(nil)
	mockCache.On("DeletePattern", mock.Anything, "rss:*").Return(nil)
	mockCache.On("DeletePattern", mock.Anything, "sitemap:*").Return(nil)
	mockCache.On("DeletePattern", mock.Anything, "trending:*").Return(nil)
	mockCache.On("DeletePattern", mock.Anything, "popular:*").Return(nil)
	mockCache.On("Delete", mock.Anything, mock.AnythingOfType("string")).Return(nil)
	
	// Test validation
	t.Run("ValidArticle", func(t *testing.T) {
		// Since we don't have a real DB connection in unit tests,
		// we'll test the validation logic
		err := models.ValidateArticle(article)
		assert.NoError(t, err)
		
		// Test that PrepareForDB works correctly
		article.PrepareForDB()
		assert.Equal(t, "test-article", article.Slug)
		assert.Equal(t, "published", article.Status)
		assert.Equal(t, "NewsArticle", article.SEOData.SchemaType)
	})
	
	t.Run("InvalidArticle", func(t *testing.T) {
		invalidArticle := &models.Article{
			Title: "", // Empty title should fail validation
		}
		
		err := models.ValidateArticle(invalidArticle)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "title is required")
	})
}

func TestArticleRepository_GetBySlug_CacheHit(t *testing.T) {
	repo, mockCache, cleanup := setupTestRepository(t)
	defer cleanup()
	
	article := createTestArticle()
	article.ID = 1
	
	// Mock cache hit
	cachedData, _ := json.Marshal(article)
	mockCache.On("Get", mock.Anything, "article:slug:test-article").Return(cachedData, nil)
	
	result, err := repo.GetBySlug(context.Background(), "test-article")
	
	assert.NoError(t, err)
	assert.Equal(t, article.Title, result.Title)
	assert.Equal(t, article.Slug, result.Slug)
	mockCache.AssertExpectations(t)
}

func TestArticleRepository_GetBySlug_CacheMiss(t *testing.T) {
	repo, mockCache, cleanup := setupTestRepository(t)
	defer cleanup()
	
	// Mock cache miss
	mockCache.On("Get", mock.Anything, "article:slug:test-article").Return([]byte(nil), nil)
	
	// Since we don't have a real DB, this will fail at the DB level
	// This is expected behavior for unit tests without database
	_, err := repo.GetBySlug(context.Background(), "test-article")
	
	assert.Error(t, err) // Expected since we don't have a real DB connection
	assert.Contains(t, err.Error(), "database connection not available")
	mockCache.AssertExpectations(t)
}

func TestArticleRepository_GetBySlug_StaticFallback(t *testing.T) {
	repo, mockCache, cleanup := setupTestRepository(t)
	defer cleanup()
	
	// Create static file structure
	articleDir := filepath.Join(repo.staticPath, "articles", "test-article")
	err := os.MkdirAll(articleDir, 0755)
	require.NoError(t, err)
	
	staticFile := filepath.Join(articleDir, "index.html")
	err = os.WriteFile(staticFile, []byte("<html><body>Test Article</body></html>"), 0644)
	require.NoError(t, err)
	
	// Mock cache miss
	mockCache.On("Get", mock.Anything, "article:slug:test-article").Return([]byte(nil), nil)
	
	result, err := repo.getFromStaticFile("test-article")
	
	assert.NoError(t, err)
	assert.Equal(t, "test-article", result.Slug)
	assert.Equal(t, "published", result.Status)
	assert.Equal(t, "NewsArticle", result.SEOData.SchemaType)
}

func TestArticleRepository_BulkCreate_Validation(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()
	
	// Test empty slice
	err := repo.BulkCreate(context.Background(), []models.Article{})
	assert.NoError(t, err)
	
	// Test validation failure
	invalidArticles := []models.Article{
		{Title: "Valid Article", Content: "Content", AuthorID: 1, CategoryID: 1, Status: "published"},
		{Title: "", Content: "Content", AuthorID: 1, CategoryID: 1, Status: "published"}, // Invalid
	}
	
	err = repo.BulkCreate(context.Background(), invalidArticles)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestArticleRepository_RecordView(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()
	
	// Test that RecordView handles missing database gracefully
	err := repo.RecordView(context.Background(), 1, "192.168.1.1", "Mozilla/5.0", "https://example.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection not available")
}

func TestCacheKeyBuilder(t *testing.T) {
	builder := cache.NewCacheKeyBuilder()
	
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
		key := builder.CategoryPageKey("technology", 2)
		assert.Equal(t, "category:technology:2", key)
	})
	
	t.Run("TagPageKey", func(t *testing.T) {
		key := builder.TagPageKey("golang", 1)
		assert.Equal(t, "tag:golang:1", key)
	})
	
	t.Run("TrendingKey", func(t *testing.T) {
		key := builder.TrendingKey("24h")
		assert.Equal(t, "trending:24h", key)
	})
}

func TestCacheInvalidator(t *testing.T) {
	mockCache := &MockCacheService{}
	invalidator := cache.NewCacheInvalidator(mockCache)
	
	t.Run("InvalidateArticle", func(t *testing.T) {
		// Mock all the expected cache deletion calls
		mockCache.On("Delete", mock.Anything, "article:123").Return(nil)
		mockCache.On("Delete", mock.Anything, "article:slug:test-article").Return(nil)
		mockCache.On("DeletePattern", mock.Anything, "homepage:*").Return(nil)
		mockCache.On("DeletePattern", mock.Anything, "category:*").Return(nil)
		mockCache.On("DeletePattern", mock.Anything, "tag:*").Return(nil)
		mockCache.On("DeletePattern", mock.Anything, "rss:*").Return(nil)
		mockCache.On("DeletePattern", mock.Anything, "sitemap:*").Return(nil)
		mockCache.On("DeletePattern", mock.Anything, "trending:*").Return(nil)
		mockCache.On("DeletePattern", mock.Anything, "popular:*").Return(nil)
		
		err := invalidator.InvalidateArticle(context.Background(), 123, "test-article")
		assert.NoError(t, err)
		mockCache.AssertExpectations(t)
	})
	
	t.Run("InvalidateCategory", func(t *testing.T) {
		mockCache.On("DeletePattern", mock.Anything, "category:technology:*").Return(nil)
		mockCache.On("DeletePattern", mock.Anything, "homepage:*").Return(nil)
		mockCache.On("Delete", mock.Anything, "rss:technology").Return(nil)
		
		err := invalidator.InvalidateCategory(context.Background(), "technology")
		assert.NoError(t, err)
		mockCache.AssertExpectations(t)
	})
	
	t.Run("InvalidateTag", func(t *testing.T) {
		mockCache.On("DeletePattern", mock.Anything, "tag:golang:*").Return(nil)
		mockCache.On("Delete", mock.Anything, "rss:golang").Return(nil)
		
		err := invalidator.InvalidateTag(context.Background(), "golang")
		assert.NoError(t, err)
		mockCache.AssertExpectations(t)
	})
}

func TestArticleValidation(t *testing.T) {
	t.Run("ValidArticle", func(t *testing.T) {
		article := createTestArticle()
		err := models.ValidateArticle(article)
		assert.NoError(t, err)
	})
	
	t.Run("EmptyTitle", func(t *testing.T) {
		article := createTestArticle()
		article.Title = ""
		err := models.ValidateArticle(article)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "title is required")
	})
	
	t.Run("LongTitle", func(t *testing.T) {
		article := createTestArticle()
		article.Title = string(make([]byte, 300)) // Too long
		err := models.ValidateArticle(article)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "title must be less than 255 characters")
	})
	
	t.Run("EmptyContent", func(t *testing.T) {
		article := createTestArticle()
		article.Content = ""
		err := models.ValidateArticle(article)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "content is required")
	})
	
	t.Run("InvalidStatus", func(t *testing.T) {
		article := createTestArticle()
		article.Status = "invalid"
		err := models.ValidateArticle(article)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "status must be one of")
	})
	
	t.Run("MissingAuthorID", func(t *testing.T) {
		article := createTestArticle()
		article.AuthorID = 0
		err := models.ValidateArticle(article)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "author_id is required")
	})
	
	t.Run("MissingCategoryID", func(t *testing.T) {
		article := createTestArticle()
		article.CategoryID = 0
		err := models.ValidateArticle(article)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "category_id is required")
	})
}

func TestSEODataValidation(t *testing.T) {
	t.Run("ValidSEOData", func(t *testing.T) {
		seo := &models.SEOData{
			MetaTitle:       "Test Title",
			MetaDescription: "Test description",
			CanonicalURL:    "https://example.com/test",
			SchemaType:      "NewsArticle",
		}
		err := models.ValidateSEOData(seo)
		assert.NoError(t, err)
	})
	
	t.Run("LongMetaTitle", func(t *testing.T) {
		seo := &models.SEOData{
			MetaTitle: string(make([]byte, 70)), // Too long
		}
		err := models.ValidateSEOData(seo)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "meta_title must be less than 60 characters")
	})
	
	t.Run("LongMetaDescription", func(t *testing.T) {
		seo := &models.SEOData{
			MetaDescription: string(make([]byte, 200)), // Too long
		}
		err := models.ValidateSEOData(seo)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "meta_description must be less than 160 characters")
	})
	
	t.Run("InvalidCanonicalURL", func(t *testing.T) {
		seo := &models.SEOData{
			CanonicalURL: "not-a-url",
		}
		err := models.ValidateSEOData(seo)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "canonical_url must be a valid URL")
	})
	
	t.Run("InvalidSchemaType", func(t *testing.T) {
		seo := &models.SEOData{
			SchemaType: "InvalidType",
		}
		err := models.ValidateSEOData(seo)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "schema_type must be one of")
	})
	
	t.Run("DefaultSchemaType", func(t *testing.T) {
		seo := &models.SEOData{}
		err := models.ValidateSEOData(seo)
		assert.NoError(t, err)
		assert.Equal(t, "NewsArticle", seo.SchemaType)
	})
}

func TestSlugGeneration(t *testing.T) {
	t.Run("BasicSlug", func(t *testing.T) {
		slug := models.GenerateSlug("Test Article Title")
		assert.Equal(t, "test-article-title", slug)
	})
	
	t.Run("SpecialCharacters", func(t *testing.T) {
		slug := models.GenerateSlug("Test & Article: With Special Characters!")
		assert.Equal(t, "test-article-with-special-characters", slug)
	})
	
	t.Run("MultipleSpaces", func(t *testing.T) {
		slug := models.GenerateSlug("Test    Article   With    Spaces")
		assert.Equal(t, "test-article-with-spaces", slug)
	})
	
	t.Run("LeadingTrailingSpaces", func(t *testing.T) {
		slug := models.GenerateSlug("  Test Article  ")
		assert.Equal(t, "test-article", slug)
	})
}

func TestSlugValidation(t *testing.T) {
	t.Run("ValidSlugs", func(t *testing.T) {
		validSlugs := []string{
			"test-article",
			"test123",
			"article-with-numbers-123",
			"a",
		}
		
		for _, slug := range validSlugs {
			assert.True(t, models.IsValidSlug(slug), "Slug should be valid: %s", slug)
		}
	})
	
	t.Run("InvalidSlugs", func(t *testing.T) {
		invalidSlugs := []string{
			"",
			"-test",
			"test-",
			"test--article",
			"Test-Article", // Uppercase
			"test_article", // Underscore
			"test article", // Space
		}
		
		for _, slug := range invalidSlugs {
			assert.False(t, models.IsValidSlug(slug), "Slug should be invalid: %s", slug)
		}
	})
}

func TestURLValidation(t *testing.T) {
	t.Run("ValidURLs", func(t *testing.T) {
		validURLs := []string{
			"https://example.com",
			"http://example.com/path",
			"https://subdomain.example.com/path?query=value",
		}
		
		for _, url := range validURLs {
			assert.True(t, models.IsValidURL(url), "URL should be valid: %s", url)
		}
	})
	
	t.Run("InvalidURLs", func(t *testing.T) {
		invalidURLs := []string{
			"",
			"not-a-url",
			"ftp://example.com", // Not http/https
			"example.com",       // Missing protocol
		}
		
		for _, url := range invalidURLs {
			assert.False(t, models.IsValidURL(url), "URL should be invalid: %s", url)
		}
	})
}

// Benchmark tests for performance validation
func BenchmarkArticleValidation(b *testing.B) {
	article := createTestArticle()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		models.ValidateArticle(article)
	}
}

func BenchmarkSlugGeneration(b *testing.B) {
	title := "This is a Test Article Title with Special Characters & Numbers 123"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		models.GenerateSlug(title)
	}
}

func BenchmarkCacheKeyGeneration(b *testing.B) {
	builder := cache.NewCacheKeyBuilder()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder.ArticleKey(uint64(i))
		builder.ArticleSlugKey("test-article")
		builder.HomepageKey("en")
	}
}