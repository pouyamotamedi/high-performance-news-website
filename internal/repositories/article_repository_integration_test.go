package repositories

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"high-performance-news-website/internal/config"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/pkg/cache"
	"high-performance-news-website/pkg/database"
)

// Integration tests require a real database connection
// These tests are skipped unless INTEGRATION_TEST=1 environment variable is set

func setupIntegrationTest(t *testing.T) (*ArticleRepository, func()) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to run.")
	}
	
	// Setup test database connection
	cfg := &config.DatabaseConfig{
		Host:     getEnvOrDefault("DB_HOST", "localhost"),
		Port:     5432,
		User:     getEnvOrDefault("DB_USER", "postgres"),
		Password: getEnvOrDefault("DB_PASSWORD", "password"),
		DBName:   getEnvOrDefault("DB_NAME", "news_test"),
		SSLMode:  "disable",
		MaxConns: 10,
		MinConns: 2,
	}
	
	db, err := database.NewConnection(cfg)
	require.NoError(t, err)
	
	// Setup test cache
	cacheConfig := &config.CacheConfig{
		Host:     getEnvOrDefault("CACHE_HOST", "localhost"),
		Port:     6379,
		Password: "",
		DB:       1, // Use different DB for tests
	}
	
	cacheService, err := cache.NewDragonflyClient(cacheConfig)
	require.NoError(t, err)
	
	// Create temporary directory for static files
	tempDir, err := os.MkdirTemp("", "article_integration_test")
	require.NoError(t, err)
	
	repo := NewArticleRepository(db, cacheService, tempDir)
	
	// Setup test data
	setupTestSchema(t, db)
	
	cleanup := func() {
		cleanupTestData(t, db)
		db.Close()
		cacheService.Close()
		os.RemoveAll(tempDir)
	}
	
	return repo, cleanup
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func setupTestSchema(t *testing.T, db *database.DB) {
	// Create test tables if they don't exist
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username VARCHAR(50) UNIQUE NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			role VARCHAR(20) NOT NULL DEFAULT 'reporter',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		
		`CREATE TABLE IF NOT EXISTS categories (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			slug VARCHAR(100) UNIQUE NOT NULL,
			description TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		
		`CREATE TABLE IF NOT EXISTS articles (
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
			PRIMARY KEY (id, created_at)
		)`,
		
		`CREATE TABLE IF NOT EXISTS article_views (
			id BIGSERIAL,
			article_id BIGINT NOT NULL,
			ip_address INET,
			user_agent TEXT,
			referer TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			PRIMARY KEY (id, created_at)
		)`,
		
		// Insert test user
		`INSERT INTO users (id, username, email, password_hash, role) 
		 VALUES (1, 'testuser', 'test@example.com', 'hashed_password', 'admin')
		 ON CONFLICT (id) DO NOTHING`,
		
		// Insert test category
		`INSERT INTO categories (id, name, slug, description) 
		 VALUES (1, 'Technology', 'technology', 'Technology news and articles')
		 ON CONFLICT (id) DO NOTHING`,
	}
	
	for _, query := range queries {
		_, err := db.Exec(query)
		require.NoError(t, err, "Failed to execute setup query: %s", query)
	}
}

func cleanupTestData(t *testing.T, db *database.DB) {
	// Clean up test data
	queries := []string{
		"DELETE FROM article_views WHERE article_id IN (SELECT id FROM articles WHERE title LIKE 'Test%')",
		"DELETE FROM articles WHERE title LIKE 'Test%'",
	}
	
	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			t.Logf("Cleanup query failed (this may be expected): %s - %v", query, err)
		}
	}
}

func TestArticleRepository_Integration_Create(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	
	article := createTestArticle()
	article.Title = "Test Integration Article"
	article.Slug = "test-integration-article"
	
	ctx := context.Background()
	
	// Test successful creation
	result, err := repo.Create(ctx, article)
	require.NoError(t, err)
	assert.NotZero(t, result.ID)
	assert.NotZero(t, result.CreatedAt)
	assert.Equal(t, article.Title, result.Title)
	assert.Equal(t, article.Slug, result.Slug)
}

func TestArticleRepository_Integration_GetBySlug(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Create test article
	article := createTestArticle()
	article.Title = "Test Get By Slug Article"
	article.Slug = "test-get-by-slug-article"
	
	created, err := repo.Create(ctx, article)
	require.NoError(t, err)
	
	// Test retrieval by slug
	retrieved, err := repo.GetBySlug(ctx, created.Slug)
	require.NoError(t, err)
	assert.Equal(t, created.ID, retrieved.ID)
	assert.Equal(t, created.Title, retrieved.Title)
	assert.Equal(t, created.Slug, retrieved.Slug)
	assert.Equal(t, created.Content, retrieved.Content)
}

func TestArticleRepository_Integration_GetByID(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Create test article
	article := createTestArticle()
	article.Title = "Test Get By ID Article"
	article.Slug = "test-get-by-id-article"
	
	created, err := repo.Create(ctx, article)
	require.NoError(t, err)
	
	// Test retrieval by ID
	retrieved, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, retrieved.ID)
	assert.Equal(t, created.Title, retrieved.Title)
	assert.Equal(t, created.Slug, retrieved.Slug)
}

func TestArticleRepository_Integration_Update(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Create test article
	article := createTestArticle()
	article.Title = "Test Update Article"
	article.Slug = "test-update-article"
	
	created, err := repo.Create(ctx, article)
	require.NoError(t, err)
	
	// Update the article
	created.Title = "Updated Test Article"
	created.Content = "Updated content"
	
	err = repo.Update(ctx, created)
	require.NoError(t, err)
	
	// Verify the update
	retrieved, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Test Article", retrieved.Title)
	assert.Equal(t, "Updated content", retrieved.Content)
}

func TestArticleRepository_Integration_Delete(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Create test article
	article := createTestArticle()
	article.Title = "Test Delete Article"
	article.Slug = "test-delete-article"
	
	created, err := repo.Create(ctx, article)
	require.NoError(t, err)
	
	// Delete the article (soft delete)
	err = repo.Delete(ctx, created.ID)
	require.NoError(t, err)
	
	// Verify the article is archived (not deleted)
	retrieved, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "archived", retrieved.Status)
}

func TestArticleRepository_Integration_GetByCategory(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Create multiple test articles in the same category
	for i := 0; i < 5; i++ {
		article := createTestArticle()
		article.Title = fmt.Sprintf("Test Category Article %d", i)
		article.Slug = fmt.Sprintf("test-category-article-%d", i)
		article.CategoryID = 1
		
		_, err := repo.Create(ctx, article)
		require.NoError(t, err)
	}
	
	// Test retrieval by category
	articles, err := repo.GetByCategory(ctx, 1, 3, 0)
	require.NoError(t, err)
	assert.Len(t, articles, 3)
	
	// Test pagination
	moreArticles, err := repo.GetByCategory(ctx, 1, 3, 3)
	require.NoError(t, err)
	assert.Len(t, moreArticles, 2)
}

func TestArticleRepository_Integration_GetTrendingArticles(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Create test articles with different view counts
	for i := 0; i < 3; i++ {
		article := createTestArticle()
		article.Title = fmt.Sprintf("Test Trending Article %d", i)
		article.Slug = fmt.Sprintf("test-trending-article-%d", i)
		
		created, err := repo.Create(ctx, article)
		require.NoError(t, err)
		
		// Simulate different view counts
		for j := 0; j < (i+1)*10; j++ {
			err = repo.RecordView(ctx, created.ID, "192.168.1.1", "Mozilla/5.0", "")
			require.NoError(t, err)
		}
	}
	
	// Test trending articles retrieval
	trending, err := repo.GetTrendingArticles(ctx, 5, 24)
	require.NoError(t, err)
	assert.True(t, len(trending) > 0)
	
	// Verify articles are ordered by trending score (highest first)
	if len(trending) > 1 {
		assert.True(t, trending[0].ViewCount >= trending[1].ViewCount)
	}
}

func TestArticleRepository_Integration_GetLatestArticles(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Create test articles with different publish times
	for i := 0; i < 3; i++ {
		article := createTestArticle()
		article.Title = fmt.Sprintf("Test Latest Article %d", i)
		article.Slug = fmt.Sprintf("test-latest-article-%d", i)
		
		// Set different publish times
		publishTime := time.Now().Add(time.Duration(-i) * time.Hour)
		article.PublishedAt = &publishTime
		
		_, err := repo.Create(ctx, article)
		require.NoError(t, err)
	}
	
	// Test latest articles retrieval
	latest, err := repo.GetLatestArticles(ctx, 5)
	require.NoError(t, err)
	assert.True(t, len(latest) > 0)
	
	// Verify articles are ordered by publish date (newest first)
	if len(latest) > 1 {
		assert.True(t, latest[0].PublishedAt.After(*latest[1].PublishedAt) || 
					latest[0].PublishedAt.Equal(*latest[1].PublishedAt))
	}
}

func TestArticleRepository_Integration_BulkCreate(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Create multiple articles for bulk insert
	articles := make([]models.Article, 10)
	for i := 0; i < 10; i++ {
		article := createTestArticle()
		article.Title = fmt.Sprintf("Test Bulk Article %d", i)
		article.Slug = fmt.Sprintf("test-bulk-article-%d", i)
		articles[i] = *article
	}
	
	// Test bulk creation
	err := repo.BulkCreate(ctx, articles)
	require.NoError(t, err)
	
	// Verify articles were created
	for i := 0; i < 10; i++ {
		retrieved, err := repo.GetBySlug(ctx, fmt.Sprintf("test-bulk-article-%d", i))
		require.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("Test Bulk Article %d", i), retrieved.Title)
	}
}

func TestArticleRepository_Integration_RecordView(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Create test article
	article := createTestArticle()
	article.Title = "Test Record View Article"
	article.Slug = "test-record-view-article"
	
	created, err := repo.Create(ctx, article)
	require.NoError(t, err)
	
	// Record multiple views
	for i := 0; i < 5; i++ {
		err = repo.RecordView(ctx, created.ID, "192.168.1.1", "Mozilla/5.0", "https://example.com")
		require.NoError(t, err)
	}
	
	// Give some time for async view count update
	time.Sleep(100 * time.Millisecond)
	
	// Verify view count was updated
	retrieved, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.True(t, retrieved.ViewCount > 0)
}

func TestArticleRepository_Integration_CacheInvalidation(t *testing.T) {
	repo, cleanup := setupIntegrationTest(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Create test article
	article := createTestArticle()
	article.Title = "Test Cache Article"
	article.Slug = "test-cache-article"
	
	created, err := repo.Create(ctx, article)
	require.NoError(t, err)
	
	// First retrieval should cache the article
	retrieved1, err := repo.GetBySlug(ctx, created.Slug)
	require.NoError(t, err)
	assert.Equal(t, created.Title, retrieved1.Title)
	
	// Update the article (should invalidate cache)
	created.Title = "Updated Cache Article"
	err = repo.Update(ctx, created)
	require.NoError(t, err)
	
	// Give some time for async cache invalidation
	time.Sleep(100 * time.Millisecond)
	
	// Second retrieval should get updated data
	retrieved2, err := repo.GetBySlug(ctx, created.Slug)
	require.NoError(t, err)
	assert.Equal(t, "Updated Cache Article", retrieved2.Title)
}

// Performance benchmarks for integration testing
func BenchmarkArticleRepository_Integration_Create(b *testing.B) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		b.Skip("Skipping integration benchmark. Set INTEGRATION_TEST=1 to run.")
	}
	
	repo, cleanup := setupIntegrationTest(&testing.T{})
	defer cleanup()
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		article := createTestArticle()
		article.Title = fmt.Sprintf("Benchmark Article %d", i)
		article.Slug = fmt.Sprintf("benchmark-article-%d", i)
		
		_, err := repo.Create(ctx, article)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkArticleRepository_Integration_GetBySlug(b *testing.B) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		b.Skip("Skipping integration benchmark. Set INTEGRATION_TEST=1 to run.")
	}
	
	repo, cleanup := setupIntegrationTest(&testing.T{})
	defer cleanup()
	
	ctx := context.Background()
	
	// Create test article
	article := createTestArticle()
	article.Title = "Benchmark Get Article"
	article.Slug = "benchmark-get-article"
	
	created, err := repo.Create(ctx, article)
	if err != nil {
		b.Fatal(err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := repo.GetBySlug(ctx, created.Slug)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkArticleRepository_Integration_BulkCreate(b *testing.B) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		b.Skip("Skipping integration benchmark. Set INTEGRATION_TEST=1 to run.")
	}
	
	repo, cleanup := setupIntegrationTest(&testing.T{})
	defer cleanup()
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create batch of 100 articles
		articles := make([]models.Article, 100)
		for j := 0; j < 100; j++ {
			article := createTestArticle()
			article.Title = fmt.Sprintf("Bulk Benchmark Article %d-%d", i, j)
			article.Slug = fmt.Sprintf("bulk-benchmark-article-%d-%d", i, j)
			articles[j] = *article
		}
		
		err := repo.BulkCreate(ctx, articles)
		if err != nil {
			b.Fatal(err)
		}
	}
}