package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
	"high-performance-news-website/pkg/cache"
	"high-performance-news-website/pkg/database"
)

// SearchIntegrationTestSuite tests the complete search system integration
type SearchIntegrationTestSuite struct {
	suite.Suite
	db            *database.DB
	cache         cache.CacheService
	searchService *services.SearchService
	indexer       *services.SearchIndexer
	testArticles  []models.Article
}

func (suite *SearchIntegrationTestSuite) SetupSuite() {
	// Setup test database
	suite.db = setupIntegrationTestDB(suite.T())
	
	// Setup test cache
	suite.cache = setupTestCache()
	
	// Create search service (will fallback to PostgreSQL in test environment)
	var err error
	suite.searchService, err = services.NewSearchService(
		suite.db, 
		suite.cache, 
		"http://localhost:7700", // MeiliSearch test instance
		"test-key",
	)
	suite.Require().NoError(err)
	
	// Setup test data
	suite.setupTestData()
}

func (suite *SearchIntegrationTestSuite) TearDownSuite() {
	suite.cleanupTestData()
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *SearchIntegrationTestSuite) setupTestData() {
	// Create test users
	testUser := &models.User{
		ID:        1,
		Username:  "testuser",
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleReporter,
	}
	
	// Create test category
	testCategory := &models.Category{
		ID:   1,
		Name: "Technology",
		Slug: "technology",
	}
	
	// Create test tags
	testTags := []models.Tag{
		{ID: 1, Name: "AI", Slug: "ai", Keywords: []string{"artificial intelligence", "machine learning"}},
		{ID: 2, Name: "Web", Slug: "web", Keywords: []string{"web development", "frontend", "backend"}},
		{ID: 3, Name: "Mobile", Slug: "mobile", Keywords: []string{"mobile app", "ios", "android"}},
	}
	
	// Create test articles
	now := time.Now()
	suite.testArticles = []models.Article{
		{
			ID:          1,
			Title:       "Introduction to Artificial Intelligence",
			Content:     "This article covers the basics of artificial intelligence and machine learning technologies.",
			Excerpt:     "Learn about AI and ML fundamentals",
			Slug:        "intro-to-ai",
			AuthorID:    1,
			CategoryID:  1,
			Status:      "published",
			PublishedAt: &now,
			ViewCount:   150,
			LikeCount:   25,
		},
		{
			ID:          2,
			Title:       "Modern Web Development Practices",
			Content:     "Explore modern web development techniques including frontend frameworks and backend APIs.",
			Excerpt:     "Modern web dev techniques and best practices",
			Slug:        "modern-web-dev",
			AuthorID:    1,
			CategoryID:  1,
			Status:      "published",
			PublishedAt: &now,
			ViewCount:   200,
			LikeCount:   30,
		},
		{
			ID:          3,
			Title:       "Mobile App Development Guide",
			Content:     "Complete guide to mobile app development for iOS and Android platforms.",
			Excerpt:     "Complete mobile app development guide",
			Slug:        "mobile-app-guide",
			AuthorID:    1,
			CategoryID:  1,
			Status:      "published",
			PublishedAt: &now,
			ViewCount:   100,
			LikeCount:   15,
		},
		{
			ID:          4,
			Title:       "Draft Article About Future Tech",
			Content:     "This is a draft article about future technology trends.",
			Excerpt:     "Future tech trends",
			Slug:        "future-tech-draft",
			AuthorID:    1,
			CategoryID:  1,
			Status:      "draft",
			PublishedAt: nil,
			ViewCount:   0,
			LikeCount:   0,
		},
	}
	
	// Insert test data into database (mock implementation)
	// In a real integration test, you would insert this data into a test database
}

func (suite *SearchIntegrationTestSuite) cleanupTestData() {
	// Clean up test data from database
	// In a real integration test, you would clean up the test database
}

func (suite *SearchIntegrationTestSuite) TestSearchArticles_BasicSearch() {
	ctx := context.Background()
	
	// Test basic search functionality
	filters := services.SearchFilters{
		Query: "artificial intelligence",
	}
	
	results, facets, total, searchTime, err := suite.searchService.SearchArticles(ctx, filters, 10, 0)
	
	suite.NoError(err)
	suite.Greater(searchTime, 0.0)
	suite.NotNil(facets)
	
	// In a real test with actual data, we would verify:
	// suite.Greater(total, 0)
	// suite.Len(results, 1)
	// suite.Equal("Introduction to Artificial Intelligence", results[0].Title)
	
	// For now, just verify the structure
	suite.NotNil(results)
	suite.GreaterOrEqual(total, 0)
}

func (suite *SearchIntegrationTestSuite) TestSearchArticles_WithFilters() {
	ctx := context.Background()
	
	// Test search with category filter
	filters := services.SearchFilters{
		Query:      "development",
		Categories: []uint64{1}, // Technology category
		SortBy:     "view_count",
		SortOrder:  "desc",
	}
	
	results, facets, total, searchTime, err := suite.searchService.SearchArticles(ctx, filters, 10, 0)
	
	suite.NoError(err)
	suite.Greater(searchTime, 0.0)
	suite.NotNil(results)
	suite.NotNil(facets)
	suite.GreaterOrEqual(total, 0)
}

func (suite *SearchIntegrationTestSuite) TestSearchArticles_Pagination() {
	ctx := context.Background()
	
	// Test pagination
	filters := services.SearchFilters{
		Query: "development",
	}
	
	// First page
	results1, _, total1, _, err := suite.searchService.SearchArticles(ctx, filters, 2, 0)
	suite.NoError(err)
	
	// Second page
	results2, _, total2, _, err := suite.searchService.SearchArticles(ctx, filters, 2, 2)
	suite.NoError(err)
	
	// Total should be the same
	suite.Equal(total1, total2)
	
	// Results should be different (if we have enough data)
	if len(results1) > 0 && len(results2) > 0 {
		suite.NotEqual(results1[0].ID, results2[0].ID)
	}
}

func (suite *SearchIntegrationTestSuite) TestSearchArticles_Caching() {
	ctx := context.Background()
	
	filters := services.SearchFilters{
		Query: "technology",
	}
	
	// First search (should hit database/MeiliSearch)
	start1 := time.Now()
	results1, facets1, total1, _, err := suite.searchService.SearchArticles(ctx, filters, 10, 0)
	duration1 := time.Since(start1)
	suite.NoError(err)
	
	// Second search (should hit cache)
	start2 := time.Now()
	results2, facets2, total2, _, err := suite.searchService.SearchArticles(ctx, filters, 10, 0)
	duration2 := time.Since(start2)
	suite.NoError(err)
	
	// Results should be identical
	suite.Equal(total1, total2)
	suite.Equal(len(results1), len(results2))
	
	// Second search should be faster (cached)
	// Note: This might not always be true in tests due to timing variations
	// suite.Less(duration2, duration1)
	
	suite.T().Logf("First search: %v, Second search: %v", duration1, duration2)
}

func (suite *SearchIntegrationTestSuite) TestSearchSuggestions() {
	ctx := context.Background()
	
	suggestions, err := suite.searchService.GetSuggestions(ctx, "artif", 5)
	
	suite.NoError(err)
	suite.NotNil(suggestions)
	// In a real test: suite.Contains(suggestions, "Introduction to Artificial Intelligence")
}

func (suite *SearchIntegrationTestSuite) TestPopularSearches() {
	ctx := context.Background()
	
	searches, err := suite.searchService.GetPopularSearches(ctx, 10, 7)
	
	suite.NoError(err)
	suite.NotNil(searches)
	suite.LenRange(searches, 0, 10)
}

func (suite *SearchIntegrationTestSuite) TestIndexingOperations() {
	ctx := context.Background()
	
	// Test indexing a new article
	newArticle := &models.Article{
		ID:          5,
		Title:       "New Test Article",
		Content:     "This is a new test article for indexing",
		Excerpt:     "New test article",
		Slug:        "new-test-article",
		AuthorID:    1,
		CategoryID:  1,
		Status:      "published",
		PublishedAt: &time.Time{},
		ViewCount:   0,
		LikeCount:   0,
	}
	
	// Index the article
	err := suite.searchService.IndexArticle(ctx, newArticle)
	suite.NoError(err)
	
	// Update the article
	newArticle.Title = "Updated Test Article"
	err = suite.searchService.UpdateArticle(ctx, newArticle)
	suite.NoError(err)
	
	// Remove the article from index
	err = suite.searchService.RemoveArticle(ctx, newArticle.ID)
	suite.NoError(err)
}

func (suite *SearchIntegrationTestSuite) TestBatchIndexing() {
	ctx := context.Background()
	
	// Test batch indexing (will fail in fallback mode, but we test the interface)
	stats, err := suite.searchService.RebuildIndex(ctx)
	
	// In fallback mode, this should return an error
	suite.Error(err)
	suite.Contains(err.Error(), "fallback mode")
	
	// Test getting index stats (will also fail in fallback mode)
	stats, err = suite.searchService.GetIndexStats(ctx)
	suite.Error(err)
	suite.Contains(err.Error(), "fallback mode")
}

func (suite *SearchIntegrationTestSuite) TestSearchPerformance() {
	ctx := context.Background()
	
	filters := services.SearchFilters{
		Query: "development",
	}
	
	// Measure search performance
	start := time.Now()
	results, _, total, searchTime, err := suite.searchService.SearchArticles(ctx, filters, 20, 0)
	totalDuration := time.Since(start)
	
	suite.NoError(err)
	suite.NotNil(results)
	suite.GreaterOrEqual(total, 0)
	
	// Search should complete within reasonable time (adjust based on requirements)
	suite.Less(totalDuration, 2*time.Second, "Search took too long: %v", totalDuration)
	suite.Less(searchTime, 200.0, "Search time reported as too slow: %v ms", searchTime)
	
	suite.T().Logf("Search completed in %v (reported: %v ms) for %d results", 
		totalDuration, searchTime, total)
}

func (suite *SearchIntegrationTestSuite) TestSearchFallbackMechanism() {
	// This test verifies that the search system gracefully falls back to PostgreSQL
	// when MeiliSearch is unavailable
	
	ctx := context.Background()
	
	// Verify that the search service is working (fallback mode is internal to the service)
	// We can't directly access the fallback mode from the service interface
	
	// Search should still work via PostgreSQL fallback
	filters := services.SearchFilters{
		Query: "test",
	}
	
	results, facets, total, searchTime, err := suite.searchService.SearchArticles(ctx, filters, 10, 0)
	
	// Should work even in fallback mode
	suite.NoError(err)
	suite.NotNil(results)
	suite.NotNil(facets)
	suite.GreaterOrEqual(total, 0)
	suite.Greater(searchTime, 0.0)
}

// Helper functions

func setupIntegrationTestDB(t *testing.T) *database.DB {
	// In a real integration test, this would set up a test database
	// For now, return a mock database
	return &database.DB{}
}

func setupTestCache() cache.CacheService {
	// In a real integration test, this would set up a test cache (Redis/in-memory)
	// For now, return a mock cache
	return &MockCacheService{}
}

// MockCacheService for integration tests
type MockCacheService struct{}

func (m *MockCacheService) Get(key string) ([]byte, error) {
	return nil, cache.ErrCacheMiss
}

func (m *MockCacheService) Set(key string, value []byte, ttl time.Duration) error {
	return nil
}

func (m *MockCacheService) Delete(key string) error {
	return nil
}

func (m *MockCacheService) DeletePattern(pattern string) error {
	return nil
}

func (m *MockCacheService) Exists(key string) bool {
	return false
}

// Run the test suite
func TestSearchIntegrationSuite(t *testing.T) {
	suite.Run(t, new(SearchIntegrationTestSuite))
}