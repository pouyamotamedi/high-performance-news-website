package services

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"high-performance-news-website/internal/models"
)

// Mock repositories for SEO validator testing
type mockSEOArticleRepository struct {
	articles map[uint64]*models.Article
	slugMap  map[string]*models.Article
}

func (m *mockSEOArticleRepository) GetByID(id uint64) (*models.Article, error) {
	if article, exists := m.articles[id]; exists {
		return article, nil
	}
	return nil, fmt.Errorf("article not found")
}

func (m *mockSEOArticleRepository) GetBySlug(slug string) (*models.Article, error) {
	if article, exists := m.slugMap[slug]; exists {
		return article, nil
	}
	return nil, fmt.Errorf("article not found")
}

func (m *mockSEOArticleRepository) GetPublishedArticlesAfterTimeWithOffset(cutoffTime time.Time, languageCode string, limit, offset int) ([]models.Article, error) {
	var result []models.Article
	count := 0
	for _, article := range m.articles {
		if article.PublishedAt != nil && article.PublishedAt.After(cutoffTime) && article.LanguageCode == languageCode {
			if count >= offset {
				result = append(result, *article)
				if len(result) >= limit {
					break
				}
			}
			count++
		}
	}
	return result, nil
}

type mockSEOCategoryRepository struct {
	categories map[string]*models.Category
}

func (m *mockSEOCategoryRepository) GetBySlug(slug, languageCode string) (*models.Category, error) {
	if category, exists := m.categories[slug]; exists {
		return category, nil
	}
	return nil, fmt.Errorf("category not found")
}

type mockSEOTagRepository struct {
	tags map[string]*models.Tag
}

func (m *mockSEOTagRepository) GetBySlug(slug, languageCode string) (*models.Tag, error) {
	if tag, exists := m.tags[slug]; exists {
		return tag, nil
	}
	return nil, fmt.Errorf("tag not found")
}

func createTestSEOValidator() *SEOValidator {
	articles := map[uint64]*models.Article{
		1: {
			ID:    1,
			Title: "Test Article 1",
			Slug:  "test-article-1",
			SEOData: models.SEOData{
				CanonicalURL: "",
			},
		},
		2: {
			ID:    2,
			Title: "Test Article 2",
			Slug:  "test-article-2",
			SEOData: models.SEOData{
				CanonicalURL: "https://example.com/article/canonical-target",
			},
		},
		3: {
			ID:    3,
			Title: "Test Article 3",
			Slug:  "test-article-3",
			SEOData: models.SEOData{
				CanonicalURL: "https://example.com/article/test-article-3", // Self-referencing
			},
		},
	}

	slugMap := map[string]*models.Article{
		"test-article-1": articles[1],
		"test-article-2": articles[2],
		"test-article-3": articles[3],
	}

	categories := map[string]*models.Category{
		"technology": {
			ID:           1,
			Name:         "Technology",
			Slug:         "technology",
			CanonicalURL: "",
		},
	}

	tags := map[string]*models.Tag{
		"golang": {
			ID:           1,
			Name:         "Golang",
			Slug:         "golang",
			CanonicalURL: "",
		},
	}

	articleRepo := &mockSEOArticleRepository{
		articles: articles,
		slugMap:  slugMap,
	}
	categoryRepo := &mockSEOCategoryRepository{categories: categories}
	tagRepo := &mockSEOTagRepository{tags: tags}

	return NewSEOValidator(articleRepo, categoryRepo, tagRepo, "https://example.com", "Test Site")
}

func TestSEOValidator_ValidateCanonicalChain_SelfReferencing(t *testing.T) {
	validator := createTestSEOValidator()

	// Test self-referencing canonical URL (optimal case)
	result, err := validator.ValidateCanonicalChain("https://example.com/article/test-article-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !result.IsValid {
		t.Errorf("Expected valid result for self-referencing canonical")
	}

	if result.ChainLength != 1 {
		t.Errorf("Expected chain length 1, got %d", result.ChainLength)
	}

	if result.HasCircularRef {
		t.Errorf("Expected no circular reference for self-referencing canonical")
	}

	if len(result.Issues) > 0 {
		t.Errorf("Expected no issues for self-referencing canonical, got %d", len(result.Issues))
	}
}

func TestSEOValidator_ValidateCanonicalChain_WithCanonicalURL(t *testing.T) {
	validator := createTestSEOValidator()

	// Test article with canonical URL pointing to another article
	result, err := validator.ValidateCanonicalChain("https://example.com/article/test-article-2")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.ChainLength != 2 {
		t.Errorf("Expected chain length 2, got %d", result.ChainLength)
	}

	expectedChain := []string{
		"https://example.com/article/test-article-2",
		"https://example.com/article/canonical-target",
	}

	if len(result.CanonicalChain) != len(expectedChain) {
		t.Errorf("Expected chain length %d, got %d", len(expectedChain), len(result.CanonicalChain))
	}

	for i, expected := range expectedChain {
		if i < len(result.CanonicalChain) && result.CanonicalChain[i] != expected {
			t.Errorf("Expected chain[%d] = %s, got %s", i, expected, result.CanonicalChain[i])
		}
	}
}

func TestSEOValidator_DetectCircularReferences(t *testing.T) {
	validator := createTestSEOValidator()

	// Create a mock result with circular reference
	result := &CanonicalValidationResult{
		CanonicalChain: []string{
			"https://example.com/article/a",
			"https://example.com/article/b",
			"https://example.com/article/c",
			"https://example.com/article/a", // Circular reference
		},
		Issues: []CanonicalIssue{},
	}

	validator.detectCircularReferences(result)

	if !result.HasCircularRef {
		t.Errorf("Expected circular reference to be detected")
	}

	if result.CircularRefPoint != "https://example.com/article/a" {
		t.Errorf("Expected circular ref point 'https://example.com/article/a', got %s", result.CircularRefPoint)
	}

	// Check if critical issue was added
	found := false
	for _, issue := range result.Issues {
		if issue.Type == "circular_canonical_reference" && issue.Severity == "critical" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected circular reference issue to be added")
	}
}

func TestSEOValidator_ValidateChainLength(t *testing.T) {
	validator := createTestSEOValidator()

	testCases := []struct {
		name           string
		chainLength    int
		expectedIssues int
		expectedSeverity string
	}{
		{"Optimal length", 1, 0, ""},
		{"Good length", 2, 0, ""},
		{"Acceptable length", 3, 0, ""},
		{"Long chain", 4, 1, "medium"},
		{"Very long chain", 6, 2, "high"}, // Should have both medium and high severity issues
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := &CanonicalValidationResult{
				ChainLength: tc.chainLength,
				Issues:      []CanonicalIssue{},
				Recommendations: []string{},
			}

			validator.validateChainLength(result)

			if len(result.Issues) != tc.expectedIssues {
				t.Errorf("Expected %d issues, got %d", tc.expectedIssues, len(result.Issues))
			}

			if tc.expectedIssues > 0 {
				found := false
				for _, issue := range result.Issues {
					if issue.Severity == tc.expectedSeverity {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected issue with severity %s", tc.expectedSeverity)
				}
			}
		})
	}
}

func TestSEOValidator_ValidateChainConsistency(t *testing.T) {
	validator := createTestSEOValidator()

	t.Run("Mixed protocols", func(t *testing.T) {
		result := &CanonicalValidationResult{
			CanonicalChain: []string{
				"http://example.com/article/a",
				"https://example.com/article/b",
			},
			Issues: []CanonicalIssue{},
			Recommendations: []string{},
		}

		validator.validateChainConsistency(result)

		found := false
		for _, issue := range result.Issues {
			if issue.Type == "mixed_protocol_canonical_chain" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected mixed protocol issue to be detected")
		}
	})

	t.Run("Cross domain", func(t *testing.T) {
		result := &CanonicalValidationResult{
			CanonicalChain: []string{
				"https://example.com/article/a",
				"https://other.com/article/b",
			},
			Issues: []CanonicalIssue{},
			Recommendations: []string{},
		}

		validator.validateChainConsistency(result)

		found := false
		for _, issue := range result.Issues {
			if issue.Type == "cross_domain_canonical_chain" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected cross domain issue to be detected")
		}
	})
}

func TestSEOValidator_NormalizeURL(t *testing.T) {
	validator := createTestSEOValidator()

	testCases := []struct {
		input    string
		expected string
	}{
		{"https://Example.com/Path/", "https://example.com/Path"},
		{"HTTP://EXAMPLE.COM/PATH", "http://example.com/PATH"},
		{"https://example.com/path#fragment", "https://example.com/path"},
		{"https://example.com/", "https://example.com/"},
		{"https://example.com", "https://example.com/"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result, err := validator.normalizeURL(tc.input)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestSEOValidator_IsInternalURL(t *testing.T) {
	validator := createTestSEOValidator()

	testCases := []struct {
		url      string
		expected bool
	}{
		{"https://example.com/article/test", true},
		{"http://example.com/category/tech", true},
		{"https://other.com/article/test", false},
		{"https://subdomain.example.com/test", false},
		{"invalid-url", false},
	}

	for _, tc := range testCases {
		t.Run(tc.url, func(t *testing.T) {
			result := validator.isInternalURL(tc.url)
			if result != tc.expected {
				t.Errorf("Expected %t, got %t", tc.expected, result)
			}
		})
	}
}

func TestSEOValidator_ValidateCanonicalURLsForArticles(t *testing.T) {
	validator := createTestSEOValidator()

	articleIDs := []uint64{1, 2, 3, 999} // 999 doesn't exist
	results, err := validator.ValidateCanonicalURLsForArticles(articleIDs)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should have results for existing articles only
	expectedCount := 3
	if len(results) != expectedCount {
		t.Errorf("Expected %d results, got %d", expectedCount, len(results))
	}

	// Check that we have results for the existing articles
	for _, id := range []uint64{1, 2, 3} {
		if _, exists := results[id]; !exists {
			t.Errorf("Expected result for article ID %d", id)
		}
	}

	// Check that we don't have result for non-existing article
	if _, exists := results[999]; exists {
		t.Errorf("Unexpected result for non-existing article ID 999")
	}
}

func TestSEOValidator_GetCanonicalValidationSummary(t *testing.T) {
	validator := createTestSEOValidator()

	// Create mock results
	results := map[uint64]*CanonicalValidationResult{
		1: {
			IsValid:      true,
			ChainLength:  1,
			HasCircularRef: false,
			ResponseTime: 100 * time.Millisecond,
			Issues:       []CanonicalIssue{},
		},
		2: {
			IsValid:      false,
			ChainLength:  4,
			HasCircularRef: true,
			ResponseTime: 200 * time.Millisecond,
			Issues: []CanonicalIssue{
				{Type: "long_canonical_chain", Severity: "medium"},
				{Type: "circular_canonical_reference", Severity: "critical"},
			},
		},
		3: {
			IsValid:      true,
			ChainLength:  2,
			HasCircularRef: false,
			ResponseTime: 150 * time.Millisecond,
			Issues:       []CanonicalIssue{},
		},
	}

	summary := validator.GetCanonicalValidationSummary(results)

	// Validate summary structure
	if summary["total_validated"] != 3 {
		t.Errorf("Expected total_validated 3, got %v", summary["total_validated"])
	}

	if summary["valid_count"] != 2 {
		t.Errorf("Expected valid_count 2, got %v", summary["valid_count"])
	}

	if summary["invalid_count"] != 1 {
		t.Errorf("Expected invalid_count 1, got %v", summary["invalid_count"])
	}

	if summary["circular_ref_count"] != 1 {
		t.Errorf("Expected circular_ref_count 1, got %v", summary["circular_ref_count"])
	}

	if summary["long_chain_count"] != 1 {
		t.Errorf("Expected long_chain_count 1, got %v", summary["long_chain_count"])
	}

	// Check average calculations
	expectedAvgChainLength := float64(1+4+2) / 3.0
	if summary["avg_chain_length"] != expectedAvgChainLength {
		t.Errorf("Expected avg_chain_length %f, got %v", expectedAvgChainLength, summary["avg_chain_length"])
	}

	expectedAvgResponseTime := (100 + 200 + 150) / 3.0 / 1000.0 // Convert to seconds
	if summary["avg_response_time"] != expectedAvgResponseTime {
		t.Errorf("Expected avg_response_time %f, got %v", expectedAvgResponseTime, summary["avg_response_time"])
	}

	// Check issue breakdown
	issueBreakdown := summary["issue_breakdown"].(map[string]int)
	if issueBreakdown["long_canonical_chain"] != 1 {
		t.Errorf("Expected 1 long_canonical_chain issue, got %d", issueBreakdown["long_canonical_chain"])
	}

	if issueBreakdown["circular_canonical_reference"] != 1 {
		t.Errorf("Expected 1 circular_canonical_reference issue, got %d", issueBreakdown["circular_canonical_reference"])
	}
}

func TestSEOValidator_HasOnlyMinorIssues(t *testing.T) {
	validator := createTestSEOValidator()

	testCases := []struct {
		name     string
		issues   []CanonicalIssue
		expected bool
	}{
		{
			name:     "No issues",
			issues:   []CanonicalIssue{},
			expected: true,
		},
		{
			name: "Only minor issues",
			issues: []CanonicalIssue{
				{Severity: "low"},
				{Severity: "medium"},
			},
			expected: true,
		},
		{
			name: "Has critical issue",
			issues: []CanonicalIssue{
				{Severity: "low"},
				{Severity: "critical"},
			},
			expected: false,
		},
		{
			name: "Has high severity issue",
			issues: []CanonicalIssue{
				{Severity: "medium"},
				{Severity: "high"},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.hasOnlyMinorIssues(tc.issues)
			if result != tc.expected {
				t.Errorf("Expected %t, got %t", tc.expected, result)
			}
		})
	}
}

func TestSEOValidator_GenerateCanonicalRecommendations(t *testing.T) {
	validator := createTestSEOValidator()

	testCases := []struct {
		name            string
		chainLength     int
		httpStatus      int
		responseTime    time.Duration
		expectedMinRecs int
	}{
		{"Optimal setup", 1, 200, 500 * time.Millisecond, 1},
		{"Good setup", 2, 200, 1 * time.Second, 1},
		{"Slow response", 1, 200, 3 * time.Second, 2},
		{"Bad status", 1, 404, 500 * time.Millisecond, 2},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := &CanonicalValidationResult{
				ChainLength:     tc.chainLength,
				HTTPStatus:      tc.httpStatus,
				ResponseTime:    tc.responseTime,
				Recommendations: []string{},
			}

			validator.generateCanonicalRecommendations(result)

			if len(result.Recommendations) < tc.expectedMinRecs {
				t.Errorf("Expected at least %d recommendations, got %d", tc.expectedMinRecs, len(result.Recommendations))
			}
		})
	}
}

// Integration test with HTTP server
func TestSEOValidator_IntegrationWithHTTPServer(t *testing.T) {
	// Create test server that simulates canonical URL responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/en/article/test":
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html><head><link rel="canonical" href="https://example.com/en/article/canonical-target"></head></html>`))
		case "/en/article/self-canonical":
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<html><head><link rel="canonical" href="` + server.URL + `/en/article/self-canonical"></head></html>`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Parse server URL to get host
	serverURL, _ := url.Parse(server.URL)
	
	validator := NewSEOValidator(nil, nil, nil, server.URL, "Test Site")

	t.Run("External URL validation", func(t *testing.T) {
		result, err := validator.ValidateCanonicalChain(server.URL + "/en/article/self-canonical")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if result.HTTPStatus != 200 {
			t.Errorf("Expected HTTP status 200, got %d", result.HTTPStatus)
		}

		if result.ResponseTime == 0 {
			t.Errorf("Expected response time to be measured")
		}
	})
}

// Google News validation tests
func TestSEOValidator_ValidateGoogleNewsCompliance(t *testing.T) {
	validator := createTestSEOValidator()

	// Create a test server to simulate RSS and sitemap endpoints
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rss/googlenews-fa.xml":
			w.Header().Set("Content-Type", "application/xml")
			w.Header().Set("Last-Modified", time.Now().Format(time.RFC1123))
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0"></rss>`))
		case "/sitemap/googlenews-fa-0.xml":
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><urlset></urlset>`))
		case "/sitemap/googlenews-index-fa.xml":
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><sitemapindex></sitemapindex>`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Update validator base URL to use test server
	validator.baseURL = server.URL

	result, err := validator.ValidateGoogleNewsCompliance("fa")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !result.RSSFeedValid {
		t.Errorf("Expected RSS feed to be valid")
	}

	if !result.SitemapValid {
		t.Errorf("Expected sitemap to be valid")
	}

	if !result.MetadataValid {
		t.Errorf("Expected metadata to be valid")
	}

	if !result.IsCompliant {
		t.Errorf("Expected Google News compliance to be true")
	}
}

func TestSEOValidator_ValidateGoogleNewsCompliance_MissingRSS(t *testing.T) {
	validator := createTestSEOValidator()

	// Create a test server that returns 404 for RSS
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	validator.baseURL = server.URL

	result, err := validator.ValidateGoogleNewsCompliance("fa")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.RSSFeedValid {
		t.Errorf("Expected RSS feed to be invalid")
	}

	if result.IsCompliant {
		t.Errorf("Expected Google News compliance to be false")
	}

	// Check for RSS-related issues
	found := false
	for _, issue := range result.Issues {
		if issue.Component == "rss" && issue.Severity == "critical" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected critical RSS issue to be reported")
	}
}

func TestSEOValidator_ValidateGoogleNewsCompliance_StaleRSS(t *testing.T) {
	validator := createTestSEOValidator()

	// Create a test server with stale RSS feed
	staleTime := time.Now().Add(-48 * time.Hour) // 48 hours ago
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rss/googlenews-fa.xml" {
			w.Header().Set("Content-Type", "application/xml")
			w.Header().Set("Last-Modified", staleTime.Format(time.RFC1123))
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0"></rss>`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	validator.baseURL = server.URL

	result, err := validator.ValidateGoogleNewsCompliance("fa")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check for stale RSS issue
	found := false
	for _, issue := range result.Issues {
		if issue.Type == "rss_feed_stale" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected stale RSS feed issue to be reported")
	}
}

func TestSEOValidator_ValidateGoogleNewsArticle(t *testing.T) {
	validator := createTestSEOValidator()

	// Test valid article
	result, err := validator.ValidateGoogleNewsArticle(1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !result.IsCompliant {
		t.Errorf("Expected article to be compliant")
	}

	// Test article with missing data
	// Create article with missing required fields
	validator.articleRepo.(*mockSEOArticleRepository).articles[4] = &models.Article{
		ID:          4,
		Title:       "", // Missing title
		Slug:        "test-article-4",
		Content:     "Short", // Too short
		AuthorID:    0,       // Missing author
		PublishedAt: nil,     // Missing publication date
		SEOData: models.SEOData{
			SchemaType: "", // Missing schema type
		},
	}

	result, err = validator.ValidateGoogleNewsArticle(4)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.IsCompliant {
		t.Errorf("Expected article to be non-compliant")
	}

	// Check for specific issues
	expectedIssues := []string{
		"missing_title",
		"missing_publication_date",
		"missing_author",
		"insufficient_content",
		"missing_schema_type",
	}

	for _, expectedType := range expectedIssues {
		found := false
		for _, issue := range result.Issues {
			if issue.Type == expectedType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected issue type %s to be reported", expectedType)
		}
	}
}

func TestSEOValidator_ValidateGoogleNewsArticle_OldArticle(t *testing.T) {
	validator := createTestSEOValidator()

	// Create old article
	oldTime := time.Now().Add(-72 * time.Hour) // 72 hours ago
	validator.articleRepo.(*mockSEOArticleRepository).articles[5] = &models.Article{
		ID:          5,
		Title:       "Old Article",
		Slug:        "old-article",
		Content:     "This is an old article with sufficient content for testing purposes.",
		AuthorID:    1,
		PublishedAt: &oldTime,
		SEOData: models.SEOData{
			SchemaType: "NewsArticle",
		},
	}

	result, err := validator.ValidateGoogleNewsArticle(5)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should still be compliant but with a low severity issue
	if !result.IsCompliant {
		t.Errorf("Expected old article to still be compliant")
	}

	// Check for age-related issue
	found := false
	for _, issue := range result.Issues {
		if issue.Type == "article_too_old" && issue.Severity == "low" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected article_too_old issue to be reported")
	}
}

func TestSEOValidator_IsValidLanguageCode(t *testing.T) {
	validator := createTestSEOValidator()

	testCases := []struct {
		code     string
		expected bool
	}{
		{"en", true},
		{"fa", true},
		{"ar", true},
		{"es", true},
		{"invalid", false},
		{"", false},
		{"eng", false}, // 3 characters
		{"zh", true},
		{"ja", true},
	}

	for _, tc := range testCases {
		t.Run(tc.code, func(t *testing.T) {
			result := validator.isValidLanguageCode(tc.code)
			if result != tc.expected {
				t.Errorf("Expected %t for language code %s, got %t", tc.expected, tc.code, result)
			}
		})
	}
}

func TestSEOValidator_HasOnlyMinorGoogleNewsIssues(t *testing.T) {
	validator := createTestSEOValidator()

	testCases := []struct {
		name     string
		issues   []GoogleNewsIssue
		expected bool
	}{
		{
			name:     "No issues",
			issues:   []GoogleNewsIssue{},
			expected: true,
		},
		{
			name: "Only minor issues",
			issues: []GoogleNewsIssue{
				{Severity: "low"},
				{Severity: "medium"},
			},
			expected: true,
		},
		{
			name: "Has critical issue",
			issues: []GoogleNewsIssue{
				{Severity: "low"},
				{Severity: "critical"},
			},
			expected: false,
		},
		{
			name: "Has high severity issue",
			issues: []GoogleNewsIssue{
				{Severity: "medium"},
				{Severity: "high"},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.hasOnlyMinorGoogleNewsIssues(tc.issues)
			if result != tc.expected {
				t.Errorf("Expected %t, got %t", tc.expected, result)
			}
		})
	}
}

func TestSEOValidator_GoogleNewsRecommendations(t *testing.T) {
	validator := createTestSEOValidator()

	testCases := []struct {
		name            string
		rssValid        bool
		sitemapValid    bool
		metadataValid   bool
		articleCount    int
		expectedMinRecs int
	}{
		{"All valid", true, true, true, 10, 4},
		{"RSS invalid", false, true, true, 10, 5},
		{"Low article count", true, true, true, 2, 5},
		{"Multiple issues", false, false, false, 1, 7},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := &GoogleNewsValidationResult{
				RSSFeedValid:  tc.rssValid,
				SitemapValid:  tc.sitemapValid,
				MetadataValid: tc.metadataValid,
				PublicationInfo: SEOGoogleNewsPublication{
					ArticleCount: tc.articleCount,
				},
				Recommendations: []string{},
			}

			validator.generateGoogleNewsRecommendations(result)

			if len(result.Recommendations) < tc.expectedMinRecs {
				t.Errorf("Expected at least %d recommendations, got %d", tc.expectedMinRecs, len(result.Recommendations))
			}
		})
	}
}

// Benchmark tests
func BenchmarkSEOValidator_ValidateCanonicalChain(b *testing.B) {
	validator := createTestSEOValidator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validator.ValidateCanonicalChain("https://example.com/article/test-article-1")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSEOValidator_DetectCircularReferences(b *testing.B) {
	validator := createTestSEOValidator()
	
	result := &CanonicalValidationResult{
		CanonicalChain: []string{
			"https://example.com/article/a",
			"https://example.com/article/b",
			"https://example.com/article/c",
			"https://example.com/article/d",
			"https://example.com/article/a", // Circular reference
		},
		Issues: []CanonicalIssue{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset issues for each iteration
		result.Issues = []CanonicalIssue{}
		result.HasCircularRef = false
		result.CircularRefPoint = ""
		
		validator.detectCircularReferences(result)
	}
}

func BenchmarkSEOValidator_ValidateGoogleNewsArticle(b *testing.B) {
	validator := createTestSEOValidator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validator.ValidateGoogleNewsArticle(1)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Schema validation tests
func TestSEOValidator_ValidateSchemaMarkup(t *testing.T) {
	validator := createTestSEOValidator()

	// Test valid article
	result, err := validator.ValidateSchemaMarkup(1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !result.IsValid {
		t.Errorf("Expected article to have valid schema markup")
	}

	if len(result.SchemaTypes) == 0 {
		t.Errorf("Expected schema types to be detected")
	}
}

func TestSEOValidator_ValidateSchemaMarkup_MissingRequiredProperties(t *testing.T) {
	validator := createTestSEOValidator()

	// Create article with missing required properties
	validator.articleRepo.(*mockSEOArticleRepository).articles[6] = &models.Article{
		ID:          6,
		Title:       "", // Missing title
		Slug:        "test-article-6",
		Content:     "", // Missing content
		Excerpt:     "", // Missing excerpt
		AuthorID:    0,  // Missing author
		PublishedAt: nil, // Missing publication date
		SEOData: models.SEOData{
			SchemaType: "NewsArticle",
		},
	}

	result, err := validator.ValidateSchemaMarkup(6)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.IsValid {
		t.Errorf("Expected article to have invalid schema markup")
	}

	// Check for specific issues
	expectedIssues := []string{
		"missing_headline",
		"missing_description",
		"missing_date_published",
		"missing_author",
	}

	for _, expectedType := range expectedIssues {
		found := false
		for _, issue := range result.Issues {
			if issue.Type == expectedType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected issue type %s to be reported", expectedType)
		}
	}
}

func TestSEOValidator_ValidateSchemaType(t *testing.T) {
	validator := createTestSEOValidator()

	testCases := []struct {
		name           string
		schemaType     string
		publishedDays  int
		expectedIssues []string
	}{
		{
			name:           "Valid NewsArticle",
			schemaType:     "NewsArticle",
			publishedDays:  1,
			expectedIssues: []string{},
		},
		{
			name:           "Stale NewsArticle",
			schemaType:     "NewsArticle",
			publishedDays:  10,
			expectedIssues: []string{"stale_news_article"},
		},
		{
			name:           "Invalid schema type",
			schemaType:     "InvalidType",
			publishedDays:  1,
			expectedIssues: []string{"invalid_schema_type"},
		},
		{
			name:           "Valid Article",
			schemaType:     "Article",
			publishedDays:  30,
			expectedIssues: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			publishedAt := time.Now().Add(-time.Duration(tc.publishedDays) * 24 * time.Hour)
			
			validator.articleRepo.(*mockSEOArticleRepository).articles[7] = &models.Article{
				ID:          7,
				Title:       "Test Article",
				Slug:        "test-article-7",
				Content:     "Test content with sufficient length for validation purposes.",
				Excerpt:     "Test excerpt",
				AuthorID:    1,
				PublishedAt: &publishedAt,
				SEOData: models.SEOData{
					SchemaType: tc.schemaType,
				},
			}

			result, err := validator.ValidateSchemaMarkup(7)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			for _, expectedType := range tc.expectedIssues {
				found := false
				for _, issue := range result.Issues {
					if issue.Type == expectedType {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected issue type %s to be reported", expectedType)
				}
			}

			// If no issues expected, check that none of the test issue types are present
			if len(tc.expectedIssues) == 0 {
				testIssueTypes := []string{"stale_news_article", "invalid_schema_type"}
				for _, testType := range testIssueTypes {
					for _, issue := range result.Issues {
						if issue.Type == testType {
							t.Errorf("Unexpected issue type %s reported", testType)
						}
					}
				}
			}
		})
	}
}

func TestSEOValidator_ValidateSchemaConsistency(t *testing.T) {
	validator := createTestSEOValidator()

	testCases := []struct {
		name           string
		title          string
		excerpt        string
		content        string
		expectedIssues []string
	}{
		{
			name:           "Optimal content",
			title:          "Good Title",
			excerpt:        "This is a good excerpt with optimal length for SEO purposes.",
			content:        "This is good content with sufficient length for validation purposes and provides value to readers.",
			expectedIssues: []string{},
		},
		{
			name:           "Long title",
			title:          "This is a very long title that exceeds the recommended 60 character limit for optimal SEO",
			excerpt:        "Good excerpt",
			content:        "Good content with sufficient length.",
			expectedIssues: []string{"long_headline"},
		},
		{
			name:           "Long description",
			title:          "Good Title",
			excerpt:        "This is a very long excerpt that exceeds the recommended 160 character limit for meta descriptions and will be truncated in search results which is not optimal for SEO.",
			content:        "Good content",
			expectedIssues: []string{"long_description"},
		},
		{
			name:           "Short description",
			title:          "Good Title",
			excerpt:        "Short",
			content:        "Good content with sufficient length.",
			expectedIssues: []string{"short_description"},
		},
		{
			name:           "Thin content",
			title:          "Good Title",
			excerpt:        "Good excerpt with optimal length.",
			content:        "Short content",
			expectedIssues: []string{"thin_content"},
		},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			articleID := uint64(10 + i)
			publishedAt := time.Now()
			
			validator.articleRepo.(*mockSEOArticleRepository).articles[articleID] = &models.Article{
				ID:          articleID,
				Title:       tc.title,
				Slug:        fmt.Sprintf("test-article-%d", articleID),
				Content:     tc.content,
				Excerpt:     tc.excerpt,
				AuthorID:    1,
				PublishedAt: &publishedAt,
				SEOData: models.SEOData{
					SchemaType: "Article",
				},
			}

			result, err := validator.ValidateSchemaMarkup(articleID)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			for _, expectedType := range tc.expectedIssues {
				found := false
				for _, issue := range result.Issues {
					if issue.Type == expectedType {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected issue type %s to be reported", expectedType)
				}
			}
		})
	}
}

func TestSEOValidator_ValidateGoogleStructuredDataGuidelines(t *testing.T) {
	validator := createTestSEOValidator()

	t.Run("Invalid dateModified", func(t *testing.T) {
		publishedAt := time.Now()
		updatedAt := publishedAt.Add(-1 * time.Hour) // Updated before published
		
		validator.articleRepo.(*mockSEOArticleRepository).articles[8] = &models.Article{
			ID:          8,
			Title:       "Test Article",
			Slug:        "test-article-8",
			Content:     "Test content with sufficient length.",
			Excerpt:     "Test excerpt",
			AuthorID:    1,
			PublishedAt: &publishedAt,
			UpdatedAt:   updatedAt,
			SEOData: models.SEOData{
				SchemaType: "NewsArticle",
			},
		}

		result, err := validator.ValidateSchemaMarkup(8)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		found := false
		for _, issue := range result.Issues {
			if issue.Type == "invalid_date_modified" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected invalid_date_modified issue to be reported")
		}
	})

	t.Run("Excessive keywords", func(t *testing.T) {
		publishedAt := time.Now()
		
		// Create article with many keywords
		manyKeywords := make([]string, 15)
		for i := 0; i < 15; i++ {
			manyKeywords[i] = fmt.Sprintf("keyword%d", i)
		}
		
		validator.articleRepo.(*mockSEOArticleRepository).articles[9] = &models.Article{
			ID:          9,
			Title:       "Test Article",
			Slug:        "test-article-9",
			Content:     "Test content with sufficient length.",
			Excerpt:     "Test excerpt",
			AuthorID:    1,
			PublishedAt: &publishedAt,
			SEOData: models.SEOData{
				SchemaType: "Article",
				Keywords:   manyKeywords,
			},
		}

		result, err := validator.ValidateSchemaMarkup(9)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		found := false
		for _, issue := range result.Issues {
			if issue.Type == "excessive_keywords" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected excessive_keywords issue to be reported")
		}
	})
}

func TestSEOValidator_ValidateSchemaMarkupForArticles(t *testing.T) {
	validator := createTestSEOValidator()

	articleIDs := []uint64{1, 2, 3, 999} // 999 doesn't exist
	results, err := validator.ValidateSchemaMarkupForArticles(articleIDs)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should have results for existing articles only
	expectedCount := 3
	if len(results) != expectedCount {
		t.Errorf("Expected %d results, got %d", expectedCount, len(results))
	}

	// Check that we have results for the existing articles
	for _, id := range []uint64{1, 2, 3} {
		if _, exists := results[id]; !exists {
			t.Errorf("Expected result for article ID %d", id)
		}
	}

	// Check that we don't have result for non-existing article
	if _, exists := results[999]; exists {
		t.Errorf("Unexpected result for non-existing article ID 999")
	}
}

func TestSEOValidator_GetSchemaValidationSummary(t *testing.T) {
	validator := createTestSEOValidator()

	// Create mock results
	results := map[uint64]*SchemaValidationResult{
		1: {
			IsValid:     true,
			SchemaTypes: []string{"NewsArticle"},
			Issues:      []SchemaIssue{},
		},
		2: {
			IsValid:     false,
			SchemaTypes: []string{"Article"},
			Issues: []SchemaIssue{
				{Type: "missing_headline", Severity: "critical"},
				{Type: "long_description", Severity: "medium"},
			},
		},
		3: {
			IsValid:     true,
			SchemaTypes: []string{"BlogPosting"},
			Issues:      []SchemaIssue{},
		},
	}

	summary := validator.GetSchemaValidationSummary(results)

	// Validate summary structure
	if summary["total_validated"] != 3 {
		t.Errorf("Expected total_validated 3, got %v", summary["total_validated"])
	}

	if summary["valid_count"] != 2 {
		t.Errorf("Expected valid_count 2, got %v", summary["valid_count"])
	}

	if summary["invalid_count"] != 1 {
		t.Errorf("Expected invalid_count 1, got %v", summary["invalid_count"])
	}

	// Check schema type breakdown
	schemaBreakdown := summary["schema_type_breakdown"].(map[string]int)
	if schemaBreakdown["NewsArticle"] != 1 {
		t.Errorf("Expected 1 NewsArticle, got %d", schemaBreakdown["NewsArticle"])
	}

	if schemaBreakdown["Article"] != 1 {
		t.Errorf("Expected 1 Article, got %d", schemaBreakdown["Article"])
	}

	if schemaBreakdown["BlogPosting"] != 1 {
		t.Errorf("Expected 1 BlogPosting, got %d", schemaBreakdown["BlogPosting"])
	}

	// Check issue breakdown
	issueBreakdown := summary["issue_breakdown"].(map[string]int)
	if issueBreakdown["missing_headline"] != 1 {
		t.Errorf("Expected 1 missing_headline issue, got %d", issueBreakdown["missing_headline"])
	}

	if issueBreakdown["long_description"] != 1 {
		t.Errorf("Expected 1 long_description issue, got %d", issueBreakdown["long_description"])
	}

	// Check severity breakdown
	severityBreakdown := summary["severity_breakdown"].(map[string]int)
	if severityBreakdown["critical"] != 1 {
		t.Errorf("Expected 1 critical issue, got %d", severityBreakdown["critical"])
	}

	if severityBreakdown["medium"] != 1 {
		t.Errorf("Expected 1 medium issue, got %d", severityBreakdown["medium"])
	}
}

func BenchmarkSEOValidator_ValidateSchemaMarkup(b *testing.B) {
	validator := createTestSEOValidator()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validator.ValidateSchemaMarkup(1)
		if err != nil {
			b.Fatal(err)
		}
	}
}