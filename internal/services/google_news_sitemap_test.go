package services

import (
	"encoding/xml"
	"fmt"
	"testing"
	"time"

	"high-performance-news-website/internal/models"
)

func createTestGoogleNewsSitemapService() *GoogleNewsSitemapService {
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
				{ID: 2, Name: "AAPL", Slug: "aapl"}, // Stock ticker
			},
			SEOData: models.SEOData{
				Keywords: []string{"tech", "innovation", "breaking"},
			},
		},
		{
			ID:          2,
			Title:       "Market Update: Business News",
			Slug:        "market-update-business-news",
			Content:     "Latest market updates and business news",
			Excerpt:     "Market update",
			AuthorID:    1,
			CategoryID:  2,
			Status:      "published",
			PublishedAt: &recentTime,
			LanguageCode: "fa",
			Tags: []models.Tag{
				{ID: 3, Name: "Business", Slug: "business"},
				{ID: 4, Name: "GOOGL", Slug: "googl"}, // Stock ticker
			},
		},
	}

	articleRepo := &mockArticleRepository{articles: articles}
	cache := &mockCache{}

	return NewGoogleNewsSitemapService(
		articleRepo,
		cache,
		"https://example.com",
		"Test News Site",
	)
}

func TestGoogleNewsSitemapService_GenerateGoogleNewsSitemap(t *testing.T) {
	service := createTestGoogleNewsSitemapService()

	xmlData, err := service.GenerateGoogleNewsSitemap("fa", 0)
	if err != nil {
		t.Fatalf("Failed to generate Google News sitemap: %v", err)
	}

	// Validate XML structure
	var sitemap NewsSitemap
	if err := xml.Unmarshal(xmlData, &sitemap); err != nil {
		t.Fatalf("Failed to unmarshal sitemap XML: %v", err)
	}

	// Validate sitemap structure
	if sitemap.Xmlns != "http://www.sitemaps.org/schemas/sitemap/0.9" {
		t.Errorf("Expected sitemap namespace, got %s", sitemap.Xmlns)
	}

	if sitemap.News != "http://www.google.com/schemas/sitemap-news/0.9" {
		t.Errorf("Expected news namespace, got %s", sitemap.News)
	}

	if len(sitemap.URLs) != 2 {
		t.Errorf("Expected 2 URLs, got %d", len(sitemap.URLs))
	}

	// Validate first URL
	url := sitemap.URLs[0]
	if url.Loc != "https://example.com/article/breaking-news-tech-innovation" {
		t.Errorf("Expected URL 'https://example.com/article/breaking-news-tech-innovation', got %s", url.Loc)
	}

	if url.News.Publication.Name != "Test News Site" {
		t.Errorf("Expected publication name 'Test News Site', got %s", url.News.Publication.Name)
	}

	if url.News.Publication.Language != "fa" {
		t.Errorf("Expected publication language 'fa', got %s", url.News.Publication.Language)
	}

	if url.News.Title != "Breaking News: Tech Innovation" {
		t.Errorf("Expected title 'Breaking News: Tech Innovation', got %s", url.News.Title)
	}

	// Validate keywords
	if url.News.Keywords == "" {
		t.Error("Expected keywords to be populated")
	}

	// Validate stock tickers (should extract AAPL from tags)
	if url.News.StockTickers != "AAPL" {
		t.Errorf("Expected stock ticker 'AAPL', got %s", url.News.StockTickers)
	}

	// Validate publication date format (should be RFC3339)
	if _, err := time.Parse(time.RFC3339, url.News.PubDate); err != nil {
		t.Errorf("Invalid publication date format: %v", err)
	}
}

func TestGoogleNewsSitemapService_GenerateGoogleNewsSitemapIndex(t *testing.T) {
	service := createTestGoogleNewsSitemapService()

	xmlData, err := service.GenerateGoogleNewsSitemapIndex("fa")
	if err != nil {
		t.Fatalf("Failed to generate Google News sitemap index: %v", err)
	}

	// Validate XML structure
	var index SitemapIndex
	if err := xml.Unmarshal(xmlData, &index); err != nil {
		t.Fatalf("Failed to unmarshal sitemap index XML: %v", err)
	}

	// Validate index structure
	if index.Xmlns != "http://www.sitemaps.org/schemas/sitemap/0.9" {
		t.Errorf("Expected sitemap namespace, got %s", index.Xmlns)
	}

	// Should have at least one sitemap reference
	if len(index.Sitemaps) == 0 {
		t.Error("Expected at least one sitemap reference")
	}

	// Validate first sitemap reference
	sitemapRef := index.Sitemaps[0]
	expectedURL := "https://example.com/sitemap/googlenews-fa-0.xml"
	if sitemapRef.Loc != expectedURL {
		t.Errorf("Expected sitemap URL '%s', got %s", expectedURL, sitemapRef.Loc)
	}

	// Validate lastmod format (should be RFC3339)
	if _, err := time.Parse(time.RFC3339, sitemapRef.LastMod); err != nil {
		t.Errorf("Invalid lastmod date format: %v", err)
	}
}

func TestGoogleNewsSitemapService_ValidateGoogleNewsSitemap(t *testing.T) {
	service := createTestGoogleNewsSitemapService()

	// Generate valid sitemap
	xmlData, err := service.GenerateGoogleNewsSitemap("fa", 0)
	if err != nil {
		t.Fatalf("Failed to generate sitemap: %v", err)
	}

	// Validate should pass
	if err := service.ValidateGoogleNewsSitemap(xmlData); err != nil {
		t.Errorf("Valid sitemap failed validation: %v", err)
	}

	// Test invalid XML
	invalidXML := []byte("invalid xml")
	if err := service.ValidateGoogleNewsSitemap(invalidXML); err == nil {
		t.Error("Expected validation to fail for invalid XML")
	}

	// Test sitemap with too many articles (over 1000)
	largeSitemap := &NewsSitemap{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		News:  "http://www.google.com/schemas/sitemap-news/0.9",
		URLs:  make([]NewsURL, 1001), // Over the limit
	}

	// Fill with dummy data
	for i := 0; i < 1001; i++ {
		largeSitemap.URLs[i] = NewsURL{
			Loc: "https://example.com/article/test",
			News: NewsData{
				Publication: NewsPublication{Name: "Test", Language: "fa"},
				PubDate:     time.Now().Format(time.RFC3339),
				Title:       "Test",
			},
		}
	}

	largeXML, _ := xml.Marshal(largeSitemap)
	if err := service.ValidateGoogleNewsSitemap(largeXML); err == nil {
		t.Error("Expected validation to fail for sitemap with over 1000 articles")
	}
}

func TestGoogleNewsSitemapService_ExtractStockTickers(t *testing.T) {
	service := createTestGoogleNewsSitemapService()

	testCases := []struct {
		name     string
		article  models.Article
		expected string
	}{
		{
			name: "Single stock ticker",
			article: models.Article{
				Tags: []models.Tag{
					{Name: "AAPL"},
					{Name: "Technology"},
				},
			},
			expected: "AAPL",
		},
		{
			name: "Multiple stock tickers",
			article: models.Article{
				Tags: []models.Tag{
					{Name: "AAPL"},
					{Name: "GOOGL"},
					{Name: "Technology"},
				},
			},
			expected: "AAPL,GOOGL",
		},
		{
			name: "No stock tickers",
			article: models.Article{
				Tags: []models.Tag{
					{Name: "Technology"},
					{Name: "News"},
				},
			},
			expected: "",
		},
		{
			name: "Mixed case (should not match)",
			article: models.Article{
				Tags: []models.Tag{
					{Name: "Apple"},
					{Name: "Google"},
				},
			},
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := service.extractStockTickers(tc.article)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestGoogleNewsSitemapService_InvalidateCache(t *testing.T) {
	service := createTestGoogleNewsSitemapService()

	// Generate sitemap to populate cache
	_, err := service.GenerateGoogleNewsSitemap("fa", 0)
	if err != nil {
		t.Fatalf("Failed to generate sitemap: %v", err)
	}

	// Invalidate cache
	if err := service.InvalidateCache("fa"); err != nil {
		t.Errorf("Failed to invalidate cache: %v", err)
	}
}

func TestGoogleNewsSitemapService_GetSitemapStats(t *testing.T) {
	service := createTestGoogleNewsSitemapService()

	stats, err := service.GetSitemapStats("fa")
	if err != nil {
		t.Fatalf("Failed to get sitemap stats: %v", err)
	}

	// Validate stats structure
	if _, exists := stats["total_articles"]; !exists {
		t.Error("Expected total_articles in stats")
	}

	if _, exists := stats["num_sitemap_files"]; !exists {
		t.Error("Expected num_sitemap_files in stats")
	}

	if _, exists := stats["articles_per_file"]; !exists {
		t.Error("Expected articles_per_file in stats")
	}

	if _, exists := stats["cutoff_time"]; !exists {
		t.Error("Expected cutoff_time in stats")
	}

	if _, exists := stats["last_updated"]; !exists {
		t.Error("Expected last_updated in stats")
	}

	// Validate articles_per_file value
	if stats["articles_per_file"] != 1000 {
		t.Errorf("Expected articles_per_file to be 1000, got %v", stats["articles_per_file"])
	}
}

func TestGoogleNewsSitemapService_EmptySitemap(t *testing.T) {
	// Create service with no articles
	articleRepo := &mockArticleRepository{articles: []models.Article{}}
	cache := &mockCache{}
	service := NewGoogleNewsSitemapService(articleRepo, cache, "https://example.com", "Test News Site")

	xmlData, err := service.GenerateGoogleNewsSitemap("fa", 0)
	if err != nil {
		t.Fatalf("Failed to generate empty sitemap: %v", err)
	}

	// Validate empty sitemap structure
	var sitemap NewsSitemap
	if err := xml.Unmarshal(xmlData, &sitemap); err != nil {
		t.Fatalf("Failed to unmarshal empty sitemap XML: %v", err)
	}

	if len(sitemap.URLs) != 0 {
		t.Errorf("Expected empty sitemap to have 0 URLs, got %d", len(sitemap.URLs))
	}
}

func TestGoogleNewsSitemapService_PaginationLogic(t *testing.T) {
	// Create service with many articles to test pagination
	articles := make([]models.Article, 2500) // More than 2 sitemap files worth
	now := time.Now()
	recentTime := now.Add(-1 * time.Hour)

	for i := 0; i < 2500; i++ {
		articles[i] = models.Article{
			ID:           uint64(i + 1),
			Title:        fmt.Sprintf("Article %d", i+1),
			Slug:         fmt.Sprintf("article-%d", i+1),
			Status:       "published",
			PublishedAt:  &recentTime,
			LanguageCode: "fa",
		}
	}

	articleRepo := &mockArticleRepository{articles: articles}
	cache := &mockCache{}
	service := NewGoogleNewsSitemapService(articleRepo, cache, "https://example.com", "Test News Site")

	// Test first sitemap file (index 0)
	xmlData0, err := service.GenerateGoogleNewsSitemap("fa", 0)
	if err != nil {
		t.Fatalf("Failed to generate sitemap file 0: %v", err)
	}

	var sitemap0 NewsSitemap
	if err := xml.Unmarshal(xmlData0, &sitemap0); err != nil {
		t.Fatalf("Failed to unmarshal sitemap 0 XML: %v", err)
	}

	// Should have 1000 articles (max per file)
	if len(sitemap0.URLs) != 1000 {
		t.Errorf("Expected sitemap 0 to have 1000 URLs, got %d", len(sitemap0.URLs))
	}

	// Test second sitemap file (index 1)
	xmlData1, err := service.GenerateGoogleNewsSitemap("fa", 1)
	if err != nil {
		t.Fatalf("Failed to generate sitemap file 1: %v", err)
	}

	var sitemap1 NewsSitemap
	if err := xml.Unmarshal(xmlData1, &sitemap1); err != nil {
		t.Fatalf("Failed to unmarshal sitemap 1 XML: %v", err)
	}

	// Should have 1000 articles (max per file)
	if len(sitemap1.URLs) != 1000 {
		t.Errorf("Expected sitemap 1 to have 1000 URLs, got %d", len(sitemap1.URLs))
	}

	// Test third sitemap file (index 2)
	xmlData2, err := service.GenerateGoogleNewsSitemap("fa", 2)
	if err != nil {
		t.Fatalf("Failed to generate sitemap file 2: %v", err)
	}

	var sitemap2 NewsSitemap
	if err := xml.Unmarshal(xmlData2, &sitemap2); err != nil {
		t.Fatalf("Failed to unmarshal sitemap 2 XML: %v", err)
	}

	// Should have 500 articles (remaining)
	if len(sitemap2.URLs) != 500 {
		t.Errorf("Expected sitemap 2 to have 500 URLs, got %d", len(sitemap2.URLs))
	}
}