package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// MockSearchService for testing
type MockSearchService struct {
	mock.Mock
}

func (m *MockSearchService) SearchArticles(ctx context.Context, filters services.SearchFilters, limit, offset int) ([]services.SearchResult, services.SearchFacets, int, float64, error) {
	args := m.Called(ctx, filters, limit, offset)
	return args.Get(0).([]services.SearchResult), args.Get(1).(services.SearchFacets), args.Int(2), args.Get(3).(float64), args.Error(4)
}

func (m *MockSearchService) GetSuggestions(ctx context.Context, query string, limit int) ([]string, error) {
	args := m.Called(ctx, query, limit)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockSearchService) GetPopularSearches(ctx context.Context, limit, days int) ([]services.PopularSearch, error) {
	args := m.Called(ctx, limit, days)
	return args.Get(0).([]services.PopularSearch), args.Error(1)
}

func (m *MockSearchService) IndexArticle(ctx context.Context, article *models.Article) error {
	args := m.Called(ctx, article)
	return args.Error(0)
}

func (m *MockSearchService) RemoveArticle(ctx context.Context, articleID uint64) error {
	args := m.Called(ctx, articleID)
	return args.Error(0)
}

func (m *MockSearchService) UpdateArticle(ctx context.Context, article *models.Article) error {
	args := m.Called(ctx, article)
	return args.Error(0)
}

func (m *MockSearchService) RebuildIndex(ctx context.Context) (*services.IndexStats, error) {
	args := m.Called(ctx)
	return args.Get(0).(*services.IndexStats), args.Error(1)
}

func (m *MockSearchService) GetIndexStats(ctx context.Context) (*services.IndexStats, error) {
	args := m.Called(ctx)
	return args.Get(0).(*services.IndexStats), args.Error(1)
}

func TestSearchHandlers_SearchArticles(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup mock service
	mockSearchService := &MockSearchService{}
	handler := &APIHandler{
		searchService: mockSearchService,
	}

	// Setup expected results
	expectedResults := []services.SearchResult{
		{
			ID:          1,
			Title:       "Test Article",
			Slug:        "test-article",
			Excerpt:     "Test excerpt",
			AuthorName:  "Test Author",
			Category:    "Test Category",
			PublishedAt: time.Now().Format(time.RFC3339),
			ViewCount:   100,
			Score:       0.95,
			Tags:        []string{"test", "article"},
			Highlights:  []string{"Test Article"},
		},
	}

	expectedFacets := services.SearchFacets{
		Categories: []services.FacetItem{
			{ID: 1, Name: "Test Category", Count: 1},
		},
		Tags: []services.FacetItem{
			{ID: 1, Name: "test", Count: 1},
		},
	}

	// Setup mock expectations
	mockSearchService.On("SearchArticles", 
		mock.Anything, 
		mock.MatchedBy(func(filters services.SearchFilters) bool {
			return filters.Query == "test query"
		}), 
		20, 0).Return(expectedResults, expectedFacets, 1, 15.5, nil)

	// Create test request
	router := gin.New()
	router.GET("/search", handler.SearchArticles)

	req, _ := http.NewRequest("GET", "/search?q=test+query", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response SearchResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Len(t, response.Results, 1)
	assert.Equal(t, "Test Article", response.Results[0].Title)
	assert.Equal(t, "test query", response.Query)
	assert.Equal(t, 15.5, response.TotalTime)
	assert.Equal(t, 1, response.Pagination.Total)
	assert.Len(t, response.Facets.Categories, 1)

	mockSearchService.AssertExpectations(t)
}

func TestSearchHandlers_SearchArticles_ValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &APIHandler{}

	// Create test request without query parameter
	router := gin.New()
	router.GET("/search", handler.SearchArticles)

	req, _ := http.NewRequest("GET", "/search", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert validation error
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "required")
}

func TestSearchHandlers_SearchArticles_WithFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSearchService := &MockSearchService{}
	handler := &APIHandler{
		searchService: mockSearchService,
	}

	// Setup mock expectations with filters
	mockSearchService.On("SearchArticles", 
		mock.Anything, 
		mock.MatchedBy(func(filters services.SearchFilters) bool {
			return filters.Query == "test" &&
				len(filters.Categories) == 2 &&
				filters.Categories[0] == 1 &&
				filters.Categories[1] == 2 &&
				filters.SortBy == "published_at" &&
				filters.SortOrder == "asc"
		}), 
		10, 20).Return([]services.SearchResult{}, services.SearchFacets{}, 0, 5.0, nil)

	// Create test request with filters
	router := gin.New()
	router.GET("/search", handler.SearchArticles)

	req, _ := http.NewRequest("GET", "/search?q=test&categories=1,2&sort_by=published_at&sort_order=asc&page=3&limit=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	mockSearchService.AssertExpectations(t)
}

func TestSearchHandlers_GetSearchSuggestions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSearchService := &MockSearchService{}
	handler := &APIHandler{
		searchService: mockSearchService,
	}

	expectedSuggestions := []string{
		"test article",
		"test content",
		"testing framework",
	}

	// Setup mock expectations
	mockSearchService.On("GetSuggestions", mock.Anything, "test", 10).Return(expectedSuggestions, nil)

	// Create test request
	router := gin.New()
	router.GET("/search/suggestions", handler.GetSearchSuggestions)

	req, _ := http.NewRequest("GET", "/search/suggestions?q=test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response SearchSuggestionResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Len(t, response.Suggestions, 3)
	assert.Equal(t, "test article", response.Suggestions[0])
	assert.Equal(t, "test", response.Query)

	mockSearchService.AssertExpectations(t)
}

func TestSearchHandlers_GetPopularSearches(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSearchService := &MockSearchService{}
	handler := &APIHandler{
		searchService: mockSearchService,
	}

	expectedSearches := []services.PopularSearch{
		{Query: "technology", Count: 100},
		{Query: "science", Count: 80},
	}

	// Setup mock expectations
	mockSearchService.On("GetPopularSearches", mock.Anything, 10, 7).Return(expectedSearches, nil)

	// Create test request
	router := gin.New()
	router.GET("/search/popular", handler.GetPopularSearches)

	req, _ := http.NewRequest("GET", "/search/popular", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Convert data back to expected type
	dataBytes, _ := json.Marshal(response.Data)
	var searches []services.PopularSearch
	json.Unmarshal(dataBytes, &searches)

	assert.Len(t, searches, 2)
	assert.Equal(t, "technology", searches[0].Query)
	assert.Equal(t, 100, searches[0].Count)

	mockSearchService.AssertExpectations(t)
}

func TestSearchHandlers_RebuildSearchIndex(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSearchService := &MockSearchService{}
	handler := &APIHandler{
		searchService: mockSearchService,
	}

	expectedStats := &services.IndexStats{
		TotalDocuments:   1000,
		LastIndexed:      time.Now(),
		IndexingTime:     5000.0,
		BatchesProcessed: 10,
		ErrorCount:       0,
	}

	// Setup mock expectations
	mockSearchService.On("RebuildIndex", mock.Anything).Return(expectedStats, nil)

	// Create test request with admin user
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", &models.User{Role: models.RoleAdmin})
		c.Next()
	})
	router.POST("/search/admin/rebuild", handler.RebuildSearchIndex)

	req, _ := http.NewRequest("POST", "/search/admin/rebuild", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "completed successfully")

	mockSearchService.AssertExpectations(t)
}

func TestSearchHandlers_RebuildSearchIndex_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &APIHandler{}

	// Create test request without user
	router := gin.New()
	router.POST("/search/admin/rebuild", handler.RebuildSearchIndex)

	req, _ := http.NewRequest("POST", "/search/admin/rebuild", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert unauthorized response
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSearchHandlers_RebuildSearchIndex_NonAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &APIHandler{}

	// Create test request with non-admin user
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", &models.User{Role: models.RoleReporter})
		c.Next()
	})
	router.POST("/search/admin/rebuild", handler.RebuildSearchIndex)

	req, _ := http.NewRequest("POST", "/search/admin/rebuild", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert unauthorized response
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSearchHandlers_GetSearchIndexStats(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSearchService := &MockSearchService{}
	handler := &APIHandler{
		searchService: mockSearchService,
	}

	expectedStats := &services.IndexStats{
		TotalDocuments: 1000,
		LastIndexed:    time.Now(),
	}

	// Setup mock expectations
	mockSearchService.On("GetIndexStats", mock.Anything).Return(expectedStats, nil)

	// Create test request with admin user
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", &models.User{Role: models.RoleAdmin})
		c.Next()
	})
	router.GET("/search/admin/stats", handler.GetSearchIndexStats)

	req, _ := http.NewRequest("GET", "/search/admin/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	mockSearchService.AssertExpectations(t)
}

// Helper function to parse uint64 array from query string (needed for testing)
func parseUint64ArrayQuery(c *gin.Context, param string) []uint64 {
	// This is a simplified version for testing
	// In the real implementation, this would parse comma-separated values
	return []uint64{}
}

// Benchmark tests

func BenchmarkSearchHandlers_SearchArticles(b *testing.B) {
	gin.SetMode(gin.TestMode)

	mockSearchService := &MockSearchService{}
	handler := &APIHandler{
		searchService: mockSearchService,
	}

	// Setup mock to return quickly
	mockSearchService.On("SearchArticles", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]services.SearchResult{}, services.SearchFacets{}, 0, 1.0, nil)

	router := gin.New()
	router.GET("/search", handler.SearchArticles)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/search?q=test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}