package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// Performance test constants
const (
	ConcurrentUsers    = 100
	RequestsPerUser    = 10
	BulkArticleCount   = 100
	SearchConcurrency  = 50
	MaxResponseTime    = 100 * time.Millisecond
)

// Performance benchmarks for API endpoints

func BenchmarkCreateArticle(b *testing.B) {
	handler, _, mockArticleService, _ := setupTestHandler()
	router := setupTestRouter(handler)
	router.POST("/articles", handler.CreateArticle)

	// Mock the service call
	expectedArticle := &models.Article{
		ID:         1,
		Title:      "Benchmark Article",
		Content:    "Benchmark content",
		AuthorID:   1,
		CategoryID: 1,
		Status:     "draft",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	mockArticleService.On("Create", mock.Anything, mock.AnythingOfType("*models.Article"), mock.AnythingOfType("*models.User")).Return(expectedArticle, nil)

	requestBody := CreateArticleRequest{
		Title:      "Benchmark Article",
		Content:    "Benchmark content",
		CategoryID: 1,
		Status:     "draft",
	}

	jsonBody, _ := json.Marshal(requestBody)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest("POST", "/articles", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusCreated {
				b.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
			}
		}
	})
}

// BenchmarkGetArticle tests article retrieval performance
func BenchmarkGetArticle(b *testing.B) {
	handler, _, mockArticleService, _ := setupTestHandler()
	router := setupTestRouter(handler)
	router.GET("/articles/:id", handler.GetArticle)

	// Mock article retrieval
	mockArticle := &models.Article{
		ID:         1,
		Title:      "Benchmark Article",
		Content:    "Benchmark content",
		AuthorID:   1,
		CategoryID: 1,
		Status:     "published",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	mockArticleService.On("GetByID", mock.Anything, uint64(1)).Return(mockArticle, nil)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest("GET", "/articles/1", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				b.Errorf("Expected status 200, got %d", w.Code)
			}
		}
	})
}

// BenchmarkBulkArticleCreation tests bulk article creation performance
func BenchmarkBulkArticleCreation(b *testing.B) {
	handler, _, mockArticleService, _ := setupTestHandler()
	router := setupTestRouter(handler)
	router.POST("/articles/bulk", handler.BulkCreateArticles)

	// Create bulk articles for testing
	articles := make([]CreateArticleRequest, BulkArticleCount)
	expectedArticles := make([]models.Article, BulkArticleCount)

	for i := 0; i < BulkArticleCount; i++ {
		articles[i] = CreateArticleRequest{
			Title:      fmt.Sprintf("Bulk Article %d", i),
			Content:    fmt.Sprintf("Content for bulk article %d", i),
			CategoryID: 1,
			Status:     "draft",
		}
		expectedArticles[i] = models.Article{
			ID:         uint64(i + 1),
			Title:      fmt.Sprintf("Bulk Article %d", i),
			AuthorID:   1,
			CategoryID: 1,
			Status:     "draft",
		}
	}

	mockArticleService.On("BulkCreate", mock.Anything, mock.AnythingOfType("[]models.Article"), mock.AnythingOfType("*models.User")).Return(expectedArticles, nil)

	requestBody := BulkCreateArticleRequest{Articles: articles}
	jsonBody, _ := json.Marshal(requestBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/articles/bulk", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			b.Errorf("Expected status 201, got %d", w.Code)
		}
	}
}

// BenchmarkSearch tests search performance
func BenchmarkSearch(b *testing.B) {
	handler, _, _, mockSearchService := setupTestHandler()
	router := setupTestRouter(handler)
	router.GET("/search", handler.SearchArticles)

	// Mock search results
	searchResults := make([]services.SearchResult, 20)
	for i := 0; i < 20; i++ {
		searchResults[i] = services.SearchResult{
			ID:          uint64(i + 1),
			Title:       fmt.Sprintf("Search Result %d", i),
			Slug:        fmt.Sprintf("search-result-%d", i),
			Excerpt:     fmt.Sprintf("Excerpt for search result %d", i),
			AuthorID:    1,
			AuthorName:  "Test Author",
			CategoryID:  1,
			Category:    "Test Category",
			Tags:        []string{"test", "search"},
			PublishedAt: time.Now().Format(time.RFC3339),
			ViewCount:   uint64(100 + i),
			Score:       0.9 - float64(i)*0.01,
		}
	}

	facets := services.SearchFacets{
		Categories: []services.FacetItem{{ID: 1, Name: "Test Category", Count: 20}},
		Tags:       []services.FacetItem{{ID: 1, Name: "test", Count: 20}},
		Authors:    []services.FacetItem{{ID: 1, Name: "Test Author", Count: 20}},
	}

	mockSearchService.On("SearchArticles", mock.Anything, mock.AnythingOfType("services.SearchFilters"), 20, 0).Return(searchResults, facets, 100, 15.5, nil)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest("GET", "/search?q=test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				b.Errorf("Expected status 200, got %d", w.Code)
			}
		}
	})
}

// TestConcurrentArticleCreation tests concurrent article creation
func TestConcurrentArticleCreation(t *testing.T) {
	handler, _, mockArticleService, _ := setupTestHandler()
	router := setupTestRouter(handler)
	router.POST("/articles", handler.CreateArticle)

	// Mock successful article creation
	mockArticle := &models.Article{
		ID:         1,
		Title:      "Concurrent Article",
		Content:    "Concurrent content",
		AuthorID:   1,
		CategoryID: 1,
		Status:     "draft",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	mockArticleService.On("Create", mock.Anything, mock.AnythingOfType("*models.Article"), mock.AnythingOfType("*models.User")).Return(mockArticle, nil)

	requestBody := CreateArticleRequest{
		Title:      "Concurrent Article",
		Content:    "Content for concurrent testing",
		CategoryID: 1,
		Status:     "draft",
	}

	jsonBody, _ := json.Marshal(requestBody)

	var wg sync.WaitGroup
	results := make(chan time.Duration, ConcurrentUsers*RequestsPerUser)
	errors := make(chan error, ConcurrentUsers*RequestsPerUser)

	// Start concurrent users
	for i := 0; i < ConcurrentUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			for j := 0; j < RequestsPerUser; j++ {
				start := time.Now()
				
				req, _ := http.NewRequest("POST", "/articles", bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				
				duration := time.Since(start)
				results <- duration

				if w.Code != http.StatusCreated {
					errors <- fmt.Errorf("user %d request %d: expected status 201, got %d", userID, j, w.Code)
				}
			}
		}(i)
	}

	wg.Wait()
	close(results)
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Error(err)
		errorCount++
	}

	// Analyze response times
	var totalDuration time.Duration
	var maxDuration time.Duration
	requestCount := 0

	for duration := range results {
		totalDuration += duration
		if duration > maxDuration {
			maxDuration = duration
		}
		requestCount++
	}

	avgDuration := totalDuration / time.Duration(requestCount)

	t.Logf("Concurrent test results:")
	t.Logf("- Total requests: %d", requestCount)
	t.Logf("- Errors: %d", errorCount)
	t.Logf("- Average response time: %v", avgDuration)
	t.Logf("- Max response time: %v", maxDuration)
	t.Logf("- Success rate: %.2f%%", float64(requestCount-errorCount)/float64(requestCount)*100)

	// Assert performance requirements
	assert.Equal(t, ConcurrentUsers*RequestsPerUser, requestCount, "All requests should complete")
	assert.Equal(t, 0, errorCount, "No errors should occur")
	assert.Less(t, avgDuration, MaxResponseTime, "Average response time should be under 100ms")
}

// TestSearchPerformance tests search performance under load
func TestSearchPerformance(t *testing.T) {
	handler, _, _, mockSearchService := setupTestHandler()
	router := setupTestRouter(handler)
	router.GET("/search", handler.SearchArticles)

	// Mock search results
	searchResults := make([]services.SearchResult, 20)
	for i := 0; i < 20; i++ {
		searchResults[i] = services.SearchResult{
			ID:          uint64(i + 1),
			Title:       fmt.Sprintf("Performance Test Result %d", i),
			Slug:        fmt.Sprintf("performance-test-result-%d", i),
			Excerpt:     fmt.Sprintf("Excerpt for performance test result %d", i),
			AuthorID:    1,
			AuthorName:  "Test Author",
			CategoryID:  1,
			Category:    "Test Category",
			Tags:        []string{"performance", "test"},
			PublishedAt: time.Now().Format(time.RFC3339),
			ViewCount:   uint64(100 + i),
			Score:       0.9 - float64(i)*0.01,
		}
	}

	facets := services.SearchFacets{
		Categories: []services.FacetItem{{ID: 1, Name: "Test Category", Count: 20}},
		Tags:       []services.FacetItem{{ID: 1, Name: "performance", Count: 20}},
		Authors:    []services.FacetItem{{ID: 1, Name: "Test Author", Count: 20}},
	}

	mockSearchService.On("SearchArticles", mock.Anything, mock.AnythingOfType("services.SearchFilters"), 20, 0).Return(searchResults, facets, 100, 15.5, nil)

	var wg sync.WaitGroup
	results := make(chan time.Duration, SearchConcurrency*RequestsPerUser)
	errors := make(chan error, SearchConcurrency*RequestsPerUser)

	// Test different search queries
	queries := []string{"test", "performance", "article", "news", "content"}

	// Start concurrent search requests
	for i := 0; i < SearchConcurrency; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			for j := 0; j < RequestsPerUser; j++ {
				query := queries[j%len(queries)]
				start := time.Now()
				
				req, _ := http.NewRequest("GET", fmt.Sprintf("/search?q=%s", query), nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				
				duration := time.Since(start)
				results <- duration

				if w.Code != http.StatusOK {
					errors <- fmt.Errorf("user %d search %d: expected status 200, got %d", userID, j, w.Code)
				}
			}
		}(i)
	}

	wg.Wait()
	close(results)
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Error(err)
		errorCount++
	}

	// Analyze response times
	var totalDuration time.Duration
	var maxDuration time.Duration
	requestCount := 0

	for duration := range results {
		totalDuration += duration
		if duration > maxDuration {
			maxDuration = duration
		}
		requestCount++
	}

	avgDuration := totalDuration / time.Duration(requestCount)

	t.Logf("Search performance test results:")
	t.Logf("- Total search requests: %d", requestCount)
	t.Logf("- Errors: %d", errorCount)
	t.Logf("- Average response time: %v", avgDuration)
	t.Logf("- Max response time: %v", maxDuration)
	t.Logf("- Success rate: %.2f%%", float64(requestCount-errorCount)/float64(requestCount)*100)

	// Assert performance requirements (search should be under 200ms)
	searchMaxTime := 200 * time.Millisecond
	assert.Equal(t, SearchConcurrency*RequestsPerUser, requestCount, "All search requests should complete")
	assert.Equal(t, 0, errorCount, "No search errors should occur")
	assert.Less(t, avgDuration, searchMaxTime, "Average search response time should be under 200ms")
}

// TestRateLimitingPerformance tests rate limiting under high load
func TestRateLimitingPerformance(t *testing.T) {
	// This test would require actual rate limiting middleware
	// For now, we'll test the concept with a simple counter
	
	handler, _, mockArticleService, _ := setupTestHandler()
	router := setupTestRouter(handler)
	router.GET("/articles/:id", handler.GetArticle)

	// Mock article retrieval
	mockArticle := &models.Article{
		ID:         1,
		Title:      "Rate Limit Test Article",
		Content:    "Content for rate limiting test",
		AuthorID:   1,
		CategoryID: 1,
		Status:     "published",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	mockArticleService.On("GetByID", mock.Anything, uint64(1)).Return(mockArticle, nil)

	// Test high request rate
	requestCount := 1000
	var wg sync.WaitGroup
	results := make(chan int, requestCount)

	start := time.Now()
	for i := 0; i < requestCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest("GET", "/articles/1", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			results <- w.Code
		}()
	}

	wg.Wait()
	close(results)
	duration := time.Since(start)

	// Count response codes
	statusCounts := make(map[int]int)
	for code := range results {
		statusCounts[code]++
	}

	t.Logf("Rate limiting test results:")
	t.Logf("- Total requests: %d", requestCount)
	t.Logf("- Duration: %v", duration)
	t.Logf("- Requests per second: %.2f", float64(requestCount)/duration.Seconds())
	t.Logf("- Status code distribution: %v", statusCounts)

	// Verify all requests were processed (in this mock scenario)
	assert.Equal(t, requestCount, statusCounts[http.StatusOK], "All requests should succeed in mock scenario")
}

// TestBulkOperationPerformance tests bulk operations performance
func TestBulkOperationPerformance(t *testing.T) {
	handler, _, mockArticleService, _ := setupTestHandler()
	router := setupTestRouter(handler)
	router.POST("/articles/bulk", handler.BulkCreateArticles)

	// Test different bulk sizes
	bulkSizes := []int{10, 50, 100, 500, 1000}

	for _, size := range bulkSizes {
		t.Run(fmt.Sprintf("BulkSize_%d", size), func(t *testing.T) {
			// Create bulk articles for testing
			articles := make([]CreateArticleRequest, size)
			expectedArticles := make([]models.Article, size)

			for i := 0; i < size; i++ {
				articles[i] = CreateArticleRequest{
					Title:      fmt.Sprintf("Bulk Article %d", i),
					Content:    fmt.Sprintf("Content for bulk article %d", i),
					CategoryID: 1,
					Status:     "draft",
				}
				expectedArticles[i] = models.Article{
					ID:         uint64(i + 1),
					Title:      fmt.Sprintf("Bulk Article %d", i),
					AuthorID:   1,
					CategoryID: 1,
					Status:     "draft",
				}
			}

			mockArticleService.On("BulkCreate", mock.Anything, mock.AnythingOfType("[]models.Article"), mock.AnythingOfType("*models.User")).Return(expectedArticles, nil).Once()

			requestBody := BulkCreateArticleRequest{Articles: articles}
			jsonBody, _ := json.Marshal(requestBody)

			start := time.Now()
			req, _ := http.NewRequest("POST", "/articles/bulk", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			duration := time.Since(start)

			assert.Equal(t, http.StatusCreated, w.Code, "Bulk creation should succeed")
			
			t.Logf("Bulk operation results for %d articles:", size)
			t.Logf("- Duration: %v", duration)
			t.Logf("- Articles per second: %.2f", float64(size)/duration.Seconds())
			
			// Performance assertion - should handle 1000 articles in under 5 seconds
			if size == 1000 {
				maxDuration := 5 * time.Second
				assert.Less(t, duration, maxDuration, "Should handle 1000 articles in under 5 seconds")
			}
		})
	}
}