package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

func createTestGoogleNewsHandlers() *GoogleNewsHandlers {
	now := time.Now()
	recentTime := now.Add(-1 * time.Hour) // 1 hour ago (within 48-hour window)

	articles := []models.Article{
		{
			ID:          1,
			Title:       "Breaking News: Tech Innovation",
			Slug:        "breaking-news-tech-innovation",
			Content:     "This is breaking news about tech innovation",
			Excerpt:     "Tech innovation news",
			AuthorID:    1,
			CategoryID:  1,
			Status:      "published",
			PublishedAt: &recentTime,
			LanguageCode: "fa",
			Tags: []models.Tag{
				{ID: 1, Name: "Technology", Slug: "technology"},
			},
		},
	}

	articleRepo := &mockArticleRepository{articles: articles}
	cache := &mockCache{}

	sitemapService := services.NewGoogleNewsSitemapService(
		articleRepo,
		cache,
		"https://example.com",
		"Test News Site",
	)

	return NewGoogleNewsHandlers(sitemapService)
}

func TestGoogleNewsHandlers_HandleGoogleNewsSitemap(t *testing.T) {
	handlers := createTestGoogleNewsHandlers()

	// Create request with mux vars
	req, err := http.NewRequest("GET", "/sitemap/googlenews-fa-0.xml", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Set up mux vars
	vars := map[string]string{
		"filename": "googlenews-fa-0.xml",
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.HandleGoogleNewsSitemap)
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Check content type
	expectedContentType := "application/xml; charset=utf-8"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("Expected content type %s, got %s", expectedContentType, contentType)
	}

	// Check cache control header
	if cacheControl := rr.Header().Get("Cache-Control"); cacheControl != "public, max-age=3600" {
		t.Errorf("Expected cache control 'public, max-age=3600', got %s", cacheControl)
	}

	// Check that response contains XML
	body := rr.Body.String()
	if !strings.Contains(body, "<?xml") {
		t.Error("Expected XML response to contain XML declaration")
	}

	if !strings.Contains(body, "<urlset") {
		t.Error("Expected XML response to contain urlset element")
	}

	if !strings.Contains(body, "xmlns:news") {
		t.Error("Expected XML response to contain news namespace")
	}
}

func TestGoogleNewsHandlers_HandleGoogleNewsSitemapInvalidFilename(t *testing.T) {
	handlers := createTestGoogleNewsHandlers()

	testCases := []struct {
		name     string
		filename string
	}{
		{"Missing extension", "googlenews-fa-0"},
		{"Wrong extension", "googlenews-fa-0.txt"},
		{"Invalid format", "invalid-filename.xml"},
		{"Missing language", "googlenews--0.xml"},
		{"Missing index", "googlenews-fa-.xml"},
		{"Invalid index", "googlenews-fa-abc.xml"},
		{"Negative index", "googlenews-fa--1.xml"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/sitemap/"+tc.filename, nil)
			if err != nil {
				t.Fatal(err)
			}

			// Set up mux vars
			vars := map[string]string{
				"filename": tc.filename,
			}
			req = mux.SetURLVars(req, vars)

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(handlers.HandleGoogleNewsSitemap)
			handler.ServeHTTP(rr, req)

			// Should return 400 Bad Request
			if status := rr.Code; status != http.StatusBadRequest {
				t.Errorf("Expected status code %d for filename %s, got %d", http.StatusBadRequest, tc.filename, status)
			}
		})
	}
}

func TestGoogleNewsHandlers_HandleGoogleNewsSitemapIndex(t *testing.T) {
	handlers := createTestGoogleNewsHandlers()

	req, err := http.NewRequest("GET", "/sitemap/googlenews-index.xml", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.HandleGoogleNewsSitemapIndex)
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	// Check content type
	expectedContentType := "application/xml; charset=utf-8"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("Expected content type %s, got %s", expectedContentType, contentType)
	}

	// Check cache control header
	if cacheControl := rr.Header().Get("Cache-Control"); cacheControl != "public, max-age=3600" {
		t.Errorf("Expected cache control 'public, max-age=3600', got %s", cacheControl)
	}

	// Check that response contains XML
	body := rr.Body.String()
	if !strings.Contains(body, "<?xml") {
		t.Error("Expected XML response to contain XML declaration")
	}

	if !strings.Contains(body, "<sitemapindex") {
		t.Error("Expected XML response to contain sitemapindex element")
	}
}

func TestGoogleNewsHandlers_HandleGoogleNewsSitemapIndexWithLanguage(t *testing.T) {
	handlers := createTestGoogleNewsHandlers()

	req, err := http.NewRequest("GET", "/sitemap/googlenews-index.xml?lang=en", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.HandleGoogleNewsSitemapIndex)
	handler.ServeHTTP(rr, req)

	// Should still return OK
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}
}

func TestGoogleNewsHandlers_HandleGoogleNewsSitemapStats(t *testing.T) {
	handlers := createTestGoogleNewsHandlers()

	req, err := http.NewRequest("GET", "/admin/sitemap/googlenews/stats", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.HandleGoogleNewsSitemapStats)
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
	expectedFields := []string{"total_articles", "num_sitemap_files", "articles_per_file", "cutoff_time", "last_updated"}
	for _, field := range expectedFields {
		if !strings.Contains(body, field) {
			t.Errorf("Expected response to contain field '%s'", field)
		}
	}
}

func TestGoogleNewsHandlers_ParseGoogleNewsSitemapFilename(t *testing.T) {
	handlers := createTestGoogleNewsHandlers()

	testCases := []struct {
		name         string
		filename     string
		expectedLang string
		expectedIdx  int
		expectError  bool
	}{
		{"Valid filename", "googlenews-fa-0.xml", "fa", 0, false},
		{"Valid filename with higher index", "googlenews-en-5.xml", "en", 5, false},
		{"Missing extension", "googlenews-fa-0", "", 0, true},
		{"Wrong extension", "googlenews-fa-0.txt", "", 0, true},
		{"Invalid format", "invalid-filename.xml", "", 0, true},
		{"Missing language", "googlenews--0.xml", "", 0, true},
		{"Missing index", "googlenews-fa-.xml", "", 0, true},
		{"Invalid index", "googlenews-fa-abc.xml", "", 0, true},
		{"Negative index", "googlenews-fa--1.xml", "", 0, true},
		{"Wrong prefix", "sitemap-fa-0.xml", "", 0, true},
		{"Too many parts", "googlenews-fa-0-extra.xml", "", 0, true},
		{"Too few parts", "googlenews-fa.xml", "", 0, true},
		{"Invalid language length", "googlenews-eng-0.xml", "", 0, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			lang, idx, err := handlers.parseGoogleNewsSitemapFilename(tc.filename)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for filename %s, got nil", tc.filename)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for filename %s: %v", tc.filename, err)
				}

				if lang != tc.expectedLang {
					t.Errorf("Expected language %s, got %s", tc.expectedLang, lang)
				}

				if idx != tc.expectedIdx {
					t.Errorf("Expected index %d, got %d", tc.expectedIdx, idx)
				}
			}
		})
	}
}

func TestGoogleNewsHandlers_MultipleFileIndexes(t *testing.T) {
	handlers := createTestGoogleNewsHandlers()

	// Test multiple file indexes
	for i := 0; i < 3; i++ {
		filename := fmt.Sprintf("googlenews-fa-%d.xml", i)
		
		req, err := http.NewRequest("GET", "/sitemap/"+filename, nil)
		if err != nil {
			t.Fatal(err)
		}

		// Set up mux vars
		vars := map[string]string{
			"filename": filename,
		}
		req = mux.SetURLVars(req, vars)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(handlers.HandleGoogleNewsSitemap)
		handler.ServeHTTP(rr, req)

		// Should return OK for all valid indexes
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Expected status code %d for index %d, got %d", http.StatusOK, i, status)
		}

		// Check that response contains XML
		body := rr.Body.String()
		if !strings.Contains(body, "<urlset") {
			t.Errorf("Expected XML response for index %d to contain urlset element", i)
		}
	}
}

func TestGoogleNewsHandlers_DifferentLanguages(t *testing.T) {
	handlers := createTestGoogleNewsHandlers()

	languages := []string{"fa", "en", "ar", "fr"}

	for _, lang := range languages {
		filename := fmt.Sprintf("googlenews-%s-0.xml", lang)
		
		req, err := http.NewRequest("GET", "/sitemap/"+filename, nil)
		if err != nil {
			t.Fatal(err)
		}

		// Set up mux vars
		vars := map[string]string{
			"filename": filename,
		}
		req = mux.SetURLVars(req, vars)

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(handlers.HandleGoogleNewsSitemap)
		handler.ServeHTTP(rr, req)

		// Should return OK for all valid languages
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Expected status code %d for language %s, got %d", http.StatusOK, lang, status)
		}
	}
}