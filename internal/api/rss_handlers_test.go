package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

func createTestRSSHandlers() *RSSHandlers {
	now := time.Now()
	publishedTime := now.Add(-3 * time.Hour) // 3 hours ago (past the 2-hour delay)

	articles := []models.Article{
		{
			ID:          1,
			Title:       "Test Article 1",
			Slug:        "test-article-1",
			Content:     "This is test content for article 1",
			Excerpt:     "Test excerpt 1",
			AuthorID:    1,
			CategoryID:  1,
			Status:      "published",
			PublishedAt: &publishedTime,
			LanguageCode: "fa",
			Tags: []models.Tag{
				{ID: 1, Name: "Technology", Slug: "technology"},
			},
		},
	}

	categories := []models.Category{
		{ID: 1, Name: "Technology", Slug: "technology", Description: "Tech news", LanguageCode: "fa"},
	}

	tags := []models.Tag{
		{ID: 1, Name: "Technology", Slug: "technology", LanguageCode: "fa"},
	}

	articleRepo := &mockArticleRepository{articles: articles}
	categoryRepo := &mockCategoryRepository{categories: categories}
	tagRepo := &mockTagRepository{tags: tags}
	cache := &mockCache{}

	rssService := services.NewRSSService(
		articleRepo,
		categoryRepo,
		tagRepo,
		cache,
		"https://example.com",
		"Test News Site",
		"Test news site description",
	)

	return NewRSSHandlers(rssService)
}

func TestRSSHandlers_HandleMainRSSFeed(t *testing.T) {
	handlers := createTestRSSHandlers()

	// Test basic request
	req, err := http.NewRequest("GET", "/rss", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.HandleMainRSSFeed)
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Check content type
	expectedContentType := "application/rss+xml; charset=utf-8"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("Expected content type %s, got %s", expectedContentType, contentType)
	}

	// Check cache control header
	if cacheControl := rr.Header().Get("Cache-Control"); cacheControl != "public, max-age=14400" {
		t.Errorf("Expected cache control 'public, max-age=14400', got %s", cacheControl)
	}

	// Check that response contains XML
	body := rr.Body.String()
	if !strings.Contains(body, "<?xml") {
		t.Error("Expected XML response to contain XML declaration")
	}

	if !strings.Contains(body, "<rss") {
		t.Error("Expected XML response to contain RSS element")
	}
}

func TestRSSHandlers_HandleMainRSSFeedWithParams(t *testing.T) {
	handlers := createTestRSSHandlers()

	// Test with language and limit parameters
	req, err := http.NewRequest("GET", "/rss?lang=en&limit=5", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.HandleMainRSSFeed)
	handler.ServeHTTP(rr, req)

	// Should still return OK (even if no articles match the language)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}
}

func TestRSSHandlers_HandleCategoryRSSFeed(t *testing.T) {
	handlers := createTestRSSHandlers()

	// Create request with mux vars
	req, err := http.NewRequest("GET", "/rss/category/technology", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Set up mux vars
	vars := map[string]string{
		"slug": "technology",
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.HandleCategoryRSSFeed)
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Check content type
	expectedContentType := "application/rss+xml; charset=utf-8"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("Expected content type %s, got %s", expectedContentType, contentType)
	}

	// Check that response contains XML
	body := rr.Body.String()
	if !strings.Contains(body, "<rss") {
		t.Error("Expected XML response to contain RSS element")
	}
}

func TestRSSHandlers_HandleCategoryRSSFeedNotFound(t *testing.T) {
	handlers := createTestRSSHandlers()

	// Create request with non-existent category
	req, err := http.NewRequest("GET", "/rss/category/nonexistent", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Set up mux vars
	vars := map[string]string{
		"slug": "nonexistent",
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.HandleCategoryRSSFeed)
	handler.ServeHTTP(rr, req)

	// Should return 404
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, status)
	}
}

func TestRSSHandlers_HandleCategoryRSSFeedMissingSlug(t *testing.T) {
	handlers := createTestRSSHandlers()

	// Create request without slug
	req, err := http.NewRequest("GET", "/rss/category/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.HandleCategoryRSSFeed)
	handler.ServeHTTP(rr, req)

	// Should return 400
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, status)
	}
}

func TestRSSHandlers_HandleTagRSSFeed(t *testing.T) {
	handlers := createTestRSSHandlers()

	// Create request with mux vars
	req, err := http.NewRequest("GET", "/rss/tag/technology", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Set up mux vars
	vars := map[string]string{
		"slug": "technology",
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.HandleTagRSSFeed)
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Check content type
	expectedContentType := "application/rss+xml; charset=utf-8"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("Expected content type %s, got %s", expectedContentType, contentType)
	}
}

func TestRSSHandlers_HandleGoogleNewsRSSFeed(t *testing.T) {
	handlers := createTestRSSHandlers()

	req, err := http.NewRequest("GET", "/rss/googlenews", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.HandleGoogleNewsRSSFeed)
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Check content type
	expectedContentType := "application/rss+xml; charset=utf-8"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("Expected content type %s, got %s", expectedContentType, contentType)
	}

	// Check cache control header (shorter for news)
	if cacheControl := rr.Header().Get("Cache-Control"); cacheControl != "public, max-age=1800" {
		t.Errorf("Expected cache control 'public, max-age=1800', got %s", cacheControl)
	}

	// Check that response contains Google News XML
	body := rr.Body.String()
	if !strings.Contains(body, "xmlns:news") {
		t.Error("Expected Google News RSS to contain news namespace")
	}
}

func TestRSSHandlers_HandleRSSFeedRefresh(t *testing.T) {
	handlers := createTestRSSHandlers()

	// Test POST request
	form := url.Values{}
	form.Add("feed_type", "main")
	form.Add("language_code", "fa")

	req, err := http.NewRequest("POST", "/admin/rss/refresh", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.HandleRSSFeedRefresh)
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Check content type
	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected content type 'application/json', got %s", contentType)
	}

	// Check response body
	body := rr.Body.String()
	if !strings.Contains(body, "success") {
		t.Error("Expected success response")
	}
}

func TestRSSHandlers_HandleRSSFeedRefreshInvalidMethod(t *testing.T) {
	handlers := createTestRSSHandlers()

	// Test GET request (should fail)
	req, err := http.NewRequest("GET", "/admin/rss/refresh", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.HandleRSSFeedRefresh)
	handler.ServeHTTP(rr, req)

	// Should return 405 Method Not Allowed
	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("Expected status code %d, got %d", http.StatusMethodNotAllowed, status)
	}
}

func TestRSSHandlers_HandleRSSFeedStats(t *testing.T) {
	handlers := createTestRSSHandlers()

	req, err := http.NewRequest("GET", "/admin/rss/stats", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.HandleRSSFeedStats)
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Check content type
	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected content type 'application/json', got %s", contentType)
	}

	// Check response body contains expected fields
	body := rr.Body.String()
	expectedFields := []string{"total_articles_in_feed", "delay_hours", "cutoff_time", "last_updated"}
	for _, field := range expectedFields {
		if !strings.Contains(body, field) {
			t.Errorf("Expected response to contain field '%s'", field)
		}
	}
}

func TestRSSHandlers_HandleRSSFeedStatsWithLanguage(t *testing.T) {
	handlers := createTestRSSHandlers()

	req, err := http.NewRequest("GET", "/admin/rss/stats?lang=en", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.HandleRSSFeedStats)
	handler.ServeHTTP(rr, req)

	// Should still return OK
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}
}

func TestRSSHandlers_LimitValidation(t *testing.T) {
	handlers := createTestRSSHandlers()

	testCases := []struct {
		name          string
		limit         string
		expectedLimit int
	}{
		{"Valid limit", "25", 25},
		{"Limit too high", "200", 50}, // Should default to 50
		{"Invalid limit", "abc", 50},   // Should default to 50
		{"Negative limit", "-5", 50},   // Should default to 50
		{"Zero limit", "0", 50},        // Should default to 50
		{"Empty limit", "", 50},        // Should default to 50
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := "/rss"
			if tc.limit != "" {
				url += "?limit=" + tc.limit
			}

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(handlers.HandleMainRSSFeed)
			handler.ServeHTTP(rr, req)

			// Should always return OK regardless of limit validation
			if status := rr.Code; status != http.StatusOK {
				t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
			}
		})
	}
}