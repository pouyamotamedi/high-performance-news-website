package services

import (
	"fmt"
	"time"
)

// MockRSSService provides mock RSS functionality for development mode
type MockRSSService struct{}

// NewMockRSSService creates a new mock RSS service
func NewMockRSSService() *MockRSSService {
	return &MockRSSService{}
}

func (m *MockRSSService) GenerateMainRSSFeed(languageCode string, limit int) ([]byte, error) {
	now := time.Now().Format(time.RFC1123Z)
	rss := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
  <channel>
    <title>High Performance News Website - Main Feed</title>
    <link>http://localhost:8080</link>
    <description>Latest news and articles from our high performance news website</description>
    <language>%s</language>
    <lastBuildDate>%s</lastBuildDate>
    <atom:link href="http://localhost:8080/rss" rel="self" type="application/rss+xml"/>
    
    <item>
      <title>Sample Article 1 - Breaking News</title>
      <link>http://localhost:8080/article/sample-article-1</link>
      <description>This is a sample article for development mode. Lorem ipsum dolor sit amet, consectetur adipiscing elit.</description>
      <pubDate>%s</pubDate>
      <guid>http://localhost:8080/article/sample-article-1</guid>
    </item>
    
    <item>
      <title>Sample Article 2 - Technology Update</title>
      <link>http://localhost:8080/article/sample-article-2</link>
      <description>Another sample article showcasing the RSS feed functionality in development mode.</description>
      <pubDate>%s</pubDate>
      <guid>http://localhost:8080/article/sample-article-2</guid>
    </item>
    
    <item>
      <title>Sample Article 3 - Sports News</title>
      <link>http://localhost:8080/article/sample-article-3</link>
      <description>Sports news sample article for testing RSS feed generation.</description>
      <pubDate>%s</pubDate>
      <guid>http://localhost:8080/article/sample-article-3</guid>
    </item>
  </channel>
</rss>`, languageCode, now, now, now, now)

	return []byte(rss), nil
}

func (m *MockRSSService) GenerateCategoryRSSFeed(categorySlug, languageCode string, limit int) ([]byte, error) {
	now := time.Now().Format(time.RFC1123Z)
	// Use language code for URL prefix (SEO best practice)
	lang := languageCode
	if lang == "" {
		lang = "en"
	}
	rss := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
  <channel>
    <title>High Performance News Website - %s Category</title>
    <link>http://localhost:8080/%s/category/%s</link>
    <description>Latest articles from the %s category</description>
    <language>%s</language>
    <lastBuildDate>%s</lastBuildDate>
    <atom:link href="http://localhost:8080/rss/category/%s" rel="self" type="application/rss+xml"/>
    
    <item>
      <title>Sample %s Article 1</title>
      <link>http://localhost:8080/%s/article/sample-%s-article-1</link>
      <description>This is a sample article from the %s category for development mode.</description>
      <pubDate>%s</pubDate>
      <guid>http://localhost:8080/%s/article/sample-%s-article-1</guid>
    </item>
    
    <item>
      <title>Sample %s Article 2</title>
      <link>http://localhost:8080/%s/article/sample-%s-article-2</link>
      <description>Another sample article from the %s category for testing purposes.</description>
      <pubDate>%s</pubDate>
      <guid>http://localhost:8080/%s/article/sample-%s-article-2</guid>
    </item>
  </channel>
</rss>`, categorySlug, lang, categorySlug, categorySlug, languageCode, now, categorySlug, 
		categorySlug, lang, categorySlug, categorySlug, now, lang, categorySlug,
		categorySlug, lang, categorySlug, categorySlug, now, lang, categorySlug)

	return []byte(rss), nil
}

func (m *MockRSSService) GenerateTagRSSFeed(tagSlug, languageCode string, limit int) ([]byte, error) {
	now := time.Now().Format(time.RFC1123Z)
	// Use language code for URL prefix (SEO best practice)
	lang := languageCode
	if lang == "" {
		lang = "en"
	}
	rss := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
  <channel>
    <title>High Performance News Website - %s Tag</title>
    <link>http://localhost:8080/%s/tag/%s</link>
    <description>Latest articles tagged with %s</description>
    <language>%s</language>
    <lastBuildDate>%s</lastBuildDate>
    <atom:link href="http://localhost:8080/rss/tag/%s" rel="self" type="application/rss+xml"/>
    
    <item>
      <title>Sample Article Tagged with %s</title>
      <link>http://localhost:8080/%s/article/sample-tagged-article-1</link>
      <description>This is a sample article tagged with %s for development mode.</description>
      <pubDate>%s</pubDate>
      <guid>http://localhost:8080/%s/article/sample-tagged-article-1</guid>
    </item>
  </channel>
</rss>`, tagSlug, lang, tagSlug, tagSlug, languageCode, now, tagSlug, 
		tagSlug, lang, tagSlug, now, lang)

	return []byte(rss), nil
}

func (m *MockRSSService) GenerateGoogleNewsRSSFeed(languageCode string, limit int) ([]byte, error) {
	now := time.Now().Format(time.RFC1123Z)
	rss := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:news="http://www.google.com/schemas/sitemap-news/0.9" xmlns:atom="http://www.w3.org/2005/Atom">
  <channel>
    <title>High Performance News Website - Google News</title>
    <link>http://localhost:8080</link>
    <description>Latest news articles optimized for Google News</description>
    <language>%s</language>
    <lastBuildDate>%s</lastBuildDate>
    <atom:link href="http://localhost:8080/rss/googlenews" rel="self" type="application/rss+xml"/>
    
    <item>
      <title>Breaking: Sample Google News Article</title>
      <link>http://localhost:8080/article/google-news-sample-1</link>
      <description>This is a sample Google News article for development mode testing.</description>
      <pubDate>%s</pubDate>
      <guid>http://localhost:8080/article/google-news-sample-1</guid>
      <news:news>
        <news:publication>
          <news:name>High Performance News Website</news:name>
          <news:language>%s</news:language>
        </news:publication>
        <news:publication_date>%s</news:publication_date>
        <news:title>Breaking: Sample Google News Article</news:title>
        <news:keywords>sample, news, development, testing</news:keywords>
      </news:news>
    </item>
    
    <item>
      <title>Technology: Sample Tech News</title>
      <link>http://localhost:8080/article/google-news-sample-2</link>
      <description>Sample technology news article for Google News RSS feed testing.</description>
      <pubDate>%s</pubDate>
      <guid>http://localhost:8080/article/google-news-sample-2</guid>
      <news:news>
        <news:publication>
          <news:name>High Performance News Website</news:name>
          <news:language>%s</news:language>
        </news:publication>
        <news:publication_date>%s</news:publication_date>
        <news:title>Technology: Sample Tech News</news:title>
        <news:keywords>technology, innovation, development</news:keywords>
      </news:news>
    </item>
  </channel>
</rss>`, languageCode, now, now, languageCode, now, now, languageCode, now)

	return []byte(rss), nil
}

func (m *MockRSSService) ValidateRSSFeed(xmlData []byte) error {
	// Mock validation always passes
	return nil
}

func (m *MockRSSService) ValidateGoogleNewsRSSFeed(xmlData []byte) error {
	// Mock validation always passes
	return nil
}

func (m *MockRSSService) ForceRefreshFeed(feedType, identifier, languageCode string) error {
	// Mock refresh always succeeds
	return nil
}

func (m *MockRSSService) GetFeedStats(languageCode string) (map[string]interface{}, error) {
	// Return mock feed statistics
	stats := map[string]interface{}{
		"total_feeds": 4,
		"main_feed": map[string]interface{}{
			"articles": 3,
			"last_updated": "2024-01-01T12:00:00Z",
		},
		"category_feeds": map[string]interface{}{
			"count": 5,
			"total_articles": 15,
		},
		"tag_feeds": map[string]interface{}{
			"count": 10,
			"total_articles": 25,
		},
		"google_news_feed": map[string]interface{}{
			"articles": 2,
			"last_updated": "2024-01-01T12:00:00Z",
		},
	}
	return stats, nil
}