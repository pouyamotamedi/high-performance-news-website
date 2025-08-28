package services

import (
	"fmt"
	"time"
)

// MockGoogleNewsSitemapService provides mock Google News sitemap functionality for development mode
type MockGoogleNewsSitemapService struct{}

// NewMockGoogleNewsSitemapService creates a new mock Google News sitemap service
func NewMockGoogleNewsSitemapService() *MockGoogleNewsSitemapService {
	return &MockGoogleNewsSitemapService{}
}

func (m *MockGoogleNewsSitemapService) GenerateGoogleNewsSitemap(languageCode string, page int) ([]byte, error) {
	now := time.Now().Format("2006-01-02T15:04:05Z07:00")
	sitemap := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"
        xmlns:news="http://www.google.com/schemas/sitemap-news/0.9">
  <url>
    <loc>http://localhost:8080/article/sample-news-article-1</loc>
    <news:news>
      <news:publication>
        <news:name>High Performance News Website</news:name>
        <news:language>%s</news:language>
      </news:publication>
      <news:publication_date>%s</news:publication_date>
      <news:title>Sample Breaking News Article</news:title>
      <news:keywords>breaking news, sample, development</news:keywords>
      <news:stock_tickers>GOOGL, MSFT</news:stock_tickers>
    </news:news>
  </url>
  
  <url>
    <loc>http://localhost:8080/article/sample-news-article-2</loc>
    <news:news>
      <news:publication>
        <news:name>High Performance News Website</news:name>
        <news:language>%s</news:language>
      </news:publication>
      <news:publication_date>%s</news:publication_date>
      <news:title>Technology Update: Sample Tech News</news:title>
      <news:keywords>technology, innovation, sample</news:keywords>
    </news:news>
  </url>
  
  <url>
    <loc>http://localhost:8080/article/sample-news-article-3</loc>
    <news:news>
      <news:publication>
        <news:name>High Performance News Website</news:name>
        <news:language>%s</news:language>
      </news:publication>
      <news:publication_date>%s</news:publication_date>
      <news:title>Sports Update: Sample Sports News</news:title>
      <news:keywords>sports, update, sample</news:keywords>
    </news:news>
  </url>
</urlset>`, languageCode, now, languageCode, now, languageCode, now)

	return []byte(sitemap), nil
}

func (m *MockGoogleNewsSitemapService) GenerateGoogleNewsSitemapIndex(languageCode string) ([]byte, error) {
	now := time.Now().Format("2006-01-02T15:04:05Z07:00")
	sitemapIndex := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <sitemap>
    <loc>http://localhost:8080/sitemap-news.xml</loc>
    <lastmod>%s</lastmod>
  </sitemap>
  <sitemap>
    <loc>http://localhost:8080/sitemap-news-1.xml</loc>
    <lastmod>%s</lastmod>
  </sitemap>
  <sitemap>
    <loc>http://localhost:8080/sitemap-news-2.xml</loc>
    <lastmod>%s</lastmod>
  </sitemap>
</sitemapindex>`, now, now, now)

	return []byte(sitemapIndex), nil
}

func (m *MockGoogleNewsSitemapService) ValidateGoogleNewsSitemap(xmlData []byte) error {
	// Mock validation always passes
	return nil
}

func (m *MockGoogleNewsSitemapService) GetSitemapStats(languageCode string) (map[string]interface{}, error) {
	// Return mock sitemap statistics
	stats := map[string]interface{}{
		"total_sitemaps": 3,
		"total_articles": 3,
		"last_updated": "2024-01-01T12:00:00Z",
		"sitemaps": []map[string]interface{}{
			{
				"name": "sitemap-news.xml",
				"articles": 3,
				"last_updated": "2024-01-01T12:00:00Z",
			},
			{
				"name": "sitemap-news-1.xml", 
				"articles": 0,
				"last_updated": "2024-01-01T12:00:00Z",
			},
			{
				"name": "sitemap-news-2.xml",
				"articles": 0,
				"last_updated": "2024-01-01T12:00:00Z",
			},
		},
	}
	return stats, nil
}