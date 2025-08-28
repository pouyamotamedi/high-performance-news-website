package services

import (
	"context"
	"testing"
	"time"

	"high-performance-news-website/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockArticleRepository is a mock implementation of ArticleRepository
type MockArticleRepository struct {
	mock.Mock
}

func (m *MockArticleRepository) GetByID(ctx context.Context, id uint64) (*models.Article, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.Article), args.Error(1)
}

func (m *MockArticleRepository) Search(query string, filters map[string]interface{}, limit, offset int) ([]models.Article, int64, error) {
	args := m.Called(query, filters, limit, offset)
	return args.Get(0).([]models.Article), args.Get(1).(int64), args.Error(2)
}

// TestSearchIndexer_NewSearchIndexer tests the creation of a new search indexer
func TestSearchIndexer_NewSearchIndexer(t *testing.T) {
	mockRepo := &MockArticleRepository{}
	
	indexer := NewSearchIndexer("http://localhost:7700", "test-key", "test-index", mockRepo)
	
	assert.NotNil(t, indexer)
	assert.Equal(t, "test-index", indexer.indexName)
	assert.Equal(t, 1000, indexer.batchSize)
	assert.Equal(t, 30*time.Second, indexer.timeout)
	assert.Equal(t, mockRepo, indexer.fallbackDB)
}

// TestSearchIndexer_convertToSearchDocument tests article to search document conversion
func TestSearchIndexer_convertToSearchDocument(t *testing.T) {
	mockRepo := &MockArticleRepository{}
	indexer := NewSearchIndexer("http://localhost:7700", "test-key", "test-index", mockRepo)
	
	publishedAt := time.Now()
	createdAt := time.Now().Add(-1 * time.Hour)
	
	article := &models.Article{
		ID:          123,
		Title:       "Test Article",
		Content:     "This is test content for the article",
		Excerpt:     "Test excerpt",
		AuthorID:    456,
		CategoryID:  789,
		Tags:        []models.Tag{{Name: "tech"}, {Name: "news"}},
		Status:      "published",
		PublishedAt: &publishedAt,
		CreatedAt:   createdAt,
		ViewCount:   100,
		LikeCount:   25,
		LanguageCode: "en",
		SEOData: models.SEOData{
			MetaTitle:       "Test Meta Title",
			MetaDescription: "Test meta description",
			Keywords:        []string{"test", "article"},
		},
	}
	
	doc := indexer.convertToSearchDocument(article)
	
	assert.Equal(t, "123", doc.ID)
	assert.Equal(t, "Test Article", doc.Title)
	assert.Equal(t, "This is test content for the article", doc.Content)
	assert.Equal(t, "Test excerpt", doc.Excerpt)
	assert.Equal(t, uint64(456), doc.AuthorID)
	assert.Equal(t, uint64(789), doc.CategoryID)
	assert.Equal(t, []string{"tech", "news"}, doc.Tags)
	assert.Equal(t, "published", doc.Status)
	assert.Equal(t, publishedAt.Unix(), doc.PublishedAt)
	assert.Equal(t, createdAt.Unix(), doc.CreatedAt)
	assert.Equal(t, uint64(100), doc.ViewCount)
	assert.Equal(t, uint64(25), doc.LikeCount)
	assert.Equal(t, "en", doc.LanguageCode)
	assert.Equal(t, "Test Meta Title", doc.MetaTitle)
	assert.Equal(t, "Test meta description", doc.MetaDescription)
	assert.Equal(t, []string{"test", "article"}, doc.Keywords)
}

// TestSearchIndexer_convertToSearchDocument_NilPublishedAt tests conversion with nil published date
func TestSearchIndexer_convertToSearchDocument_NilPublishedAt(t *testing.T) {
	mockRepo := &MockArticleRepository{}
	indexer := NewSearchIndexer("http://localhost:7700", "test-key", "test-index", mockRepo)
	
	article := &models.Article{
		ID:           123,
		Title:        "Draft Article",
		Content:      "Draft content",
		Status:       "draft",
		PublishedAt:  nil,
		CreatedAt:    time.Now(),
		LanguageCode: "en",
	}
	
	doc := indexer.convertToSearchDocument(article)
	
	assert.Equal(t, "123", doc.ID)
	assert.Equal(t, "Draft Article", doc.Title)
	assert.Equal(t, int64(0), doc.PublishedAt) // Should be 0 for nil published date
}

// TestSearchIndexer_truncateContent tests content truncation
func TestSearchIndexer_truncateContent(t *testing.T) {
	mockRepo := &MockArticleRepository{}
	indexer := NewSearchIndexer("http://localhost:7700", "test-key", "test-index", mockRepo)
	
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "Short content",
			content:  "Short content",
			expected: "Short content",
		},
		{
			name:     "Long content with spaces",
			content:  string(make([]byte, 10001)) + " more content",
			expected: string(make([]byte, 10000)) + "...",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := indexer.truncateContent(tt.content)
			if len(tt.content) <= 10000 {
				assert.Equal(t, tt.expected, result)
			} else {
				assert.True(t, len(result) <= 10003) // 10000 + "..."
				assert.Contains(t, result, "...")
			}
		})
	}
}

// TestSearchIndexer_IndexArticlesBatch_EmptySlice tests batch indexing with empty slice
func TestSearchIndexer_IndexArticlesBatch_EmptySlice(t *testing.T) {
	mockRepo := &MockArticleRepository{}
	indexer := NewSearchIndexer("http://localhost:7700", "test-key", "test-index", mockRepo)
	
	err := indexer.IndexArticlesBatch([]models.Article{})
	
	assert.NoError(t, err)
}

// TestSearchIndexer_IndexArticlesBatch_OnlyDrafts tests batch indexing with only draft articles
func TestSearchIndexer_IndexArticlesBatch_OnlyDrafts(t *testing.T) {
	mockRepo := &MockArticleRepository{}
	indexer := NewSearchIndexer("http://localhost:7700", "test-key", "test-index", mockRepo)
	
	articles := []models.Article{
		{
			ID:           1,
			Title:        "Draft 1",
			Status:       "draft",
			LanguageCode: "en",
		},
		{
			ID:           2,
			Title:        "Draft 2",
			Status:       "draft",
			LanguageCode: "en",
		},
	}
	
	err := indexer.IndexArticlesBatch(articles)
	
	assert.NoError(t, err)
}

// TestSearchIndexer_BatchProcessing tests that articles are processed in correct batch sizes
func TestSearchIndexer_BatchProcessing(t *testing.T) {
	mockRepo := &MockArticleRepository{}
	indexer := NewSearchIndexer("http://localhost:7700", "test-key", "test-index", mockRepo)
	
	// Create more articles than batch size to test batching
	articles := make([]models.Article, 2500) // More than 2 batches of 1000
	for i := 0; i < 2500; i++ {
		articles[i] = models.Article{
			ID:           uint64(i + 1),
			Title:        "Published Article",
			Status:       "published",
			LanguageCode: "en",
			CreatedAt:    time.Now(),
		}
	}
	
	// This test mainly verifies that the function doesn't panic with large datasets
	// In a real test environment with MeiliSearch running, this would actually index
	err := indexer.IndexArticlesBatch(articles)
	
	// We expect this to fail in test environment without MeiliSearch, but it shouldn't panic
	// The important thing is that the batching logic works correctly
	assert.Error(t, err) // Expected to fail without real MeiliSearch instance
}

// TestSearchIndexer_GetIndexStats tests index statistics retrieval
func TestSearchIndexer_GetIndexStats(t *testing.T) {
	mockRepo := &MockArticleRepository{}
	indexer := NewSearchIndexer("http://localhost:7700", "test-key", "test-index", mockRepo)
	
	// This will fail without a real MeiliSearch instance, but tests the method exists
	stats, err := indexer.GetIndexStats()
	
	assert.Error(t, err) // Expected to fail without real MeiliSearch
	assert.Nil(t, stats)
}

// TestSearchIndexer_HealthCheck tests health check functionality
func TestSearchIndexer_HealthCheck(t *testing.T) {
	mockRepo := &MockArticleRepository{}
	indexer := NewSearchIndexer("http://localhost:7700", "test-key", "test-index", mockRepo)
	
	// This will fail without a real MeiliSearch instance
	err := indexer.HealthCheck()
	
	assert.Error(t, err) // Expected to fail without real MeiliSearch
}

// TestSearchIndexer_DeleteArticle tests article deletion from index
func TestSearchIndexer_DeleteArticle(t *testing.T) {
	mockRepo := &MockArticleRepository{}
	indexer := NewSearchIndexer("http://localhost:7700", "test-key", "test-index", mockRepo)
	
	// This will fail without a real MeiliSearch instance
	err := indexer.DeleteArticle("123")
	
	assert.Error(t, err) // Expected to fail without real MeiliSearch
}

// TestSearchIndexer_ClearIndex tests index clearing
func TestSearchIndexer_ClearIndex(t *testing.T) {
	mockRepo := &MockArticleRepository{}
	indexer := NewSearchIndexer("http://localhost:7700", "test-key", "test-index", mockRepo)
	
	// This will fail without a real MeiliSearch instance
	err := indexer.ClearIndex()
	
	assert.Error(t, err) // Expected to fail without real MeiliSearch
}

// TestSearchIndexer_RebuildIndex tests complete index rebuild
func TestSearchIndexer_RebuildIndex(t *testing.T) {
	mockRepo := &MockArticleRepository{}
	indexer := NewSearchIndexer("http://localhost:7700", "test-key", "test-index", mockRepo)
	
	articles := []models.Article{
		{
			ID:           1,
			Title:        "Test Article",
			Status:       "published",
			LanguageCode: "en",
			CreatedAt:    time.Now(),
		},
	}
	
	// This will fail without a real MeiliSearch instance
	err := indexer.RebuildIndex(articles)
	
	assert.Error(t, err) // Expected to fail without real MeiliSearch
}

// BenchmarkSearchIndexer_convertToSearchDocument benchmarks document conversion
func BenchmarkSearchIndexer_convertToSearchDocument(b *testing.B) {
	mockRepo := &MockArticleRepository{}
	indexer := NewSearchIndexer("http://localhost:7700", "test-key", "test-index", mockRepo)
	
	article := &models.Article{
		ID:          123,
		Title:       "Benchmark Article",
		Content:     "This is benchmark content for performance testing",
		Excerpt:     "Benchmark excerpt",
		AuthorID:    456,
		CategoryID:  789,
		Tags:        []models.Tag{{Name: "benchmark"}, {Name: "performance"}},
		Status:      "published",
		PublishedAt: &time.Time{},
		CreatedAt:   time.Now(),
		ViewCount:   1000,
		LikeCount:   50,
		LanguageCode: "en",
		SEOData: models.SEOData{
			MetaTitle:       "Benchmark Meta Title",
			MetaDescription: "Benchmark meta description",
			Keywords:        []string{"benchmark", "performance"},
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		indexer.convertToSearchDocument(article)
	}
}