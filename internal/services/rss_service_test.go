package services

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
	"testing"
	"time"

	"high-performance-news-website/internal/models"
)

// Mock repositories for testing
type mockArticleRepository struct {
	articles []models.Article
}

func (m *mockArticleRepository) GetPublishedArticlesBeforeTime(cutoffTime time.Time, languageCode string, limit int) ([]models.Article, error) {
	var result []models.Article
	for _, article := range m.articles {
		if article.PublishedAt != nil && article.PublishedAt.Before(cutoffTime) && article.LanguageCode == languageCode {
			result = append(result, article)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *mockArticleRepository) GetArticlesByCategoryBeforeTime(categoryID uint64, cutoffTime time.Time, languageCode string, limit int) ([]models.Article, error) {
	var result []models.Article
	for _, article := range m.articles {
		if article.CategoryID == categoryID && article.PublishedAt != nil && article.PublishedAt.Before(cutoffTime) && article.LanguageCode == languageCode {
			result = append(result, article)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *mockArticleRepository) GetArticlesByTagBeforeTime(tagID uint64, cutoffTime time.Time, languageCode string, limit int) ([]models.Article, error) {
	var result []models.Article
	for _, article := range m.articles {
		for _, tag := range article.Tags {
			if tag.ID == tagID && article.PublishedAt != nil && article.PublishedAt.Before(cutoffTime) && article.LanguageCode == languageCode {
				result = append(result, article)
				if len(result) >= limit {
					break
				}
			}
		}
	}
	return result, nil
}

func (m *mockArticleRepository) GetPublishedArticlesAfterTimeWithOffset(cutoffTime time.Time, languageCode string, limit, offset int) ([]models.Article, error) {
	var result []models.Article
	count := 0
	for _, article := range m.articles {
		if article.PublishedAt != nil && article.PublishedAt.After(cutoffTime) && article.LanguageCode == languageCode {
			if count >= offset {
				result = append(result, article)
				if len(result) >= limit {
					break
				}
			}
			count++
		}
	}
	return result, nil
}

func (m *mockArticleRepository) CountPublishedArticlesBeforeTime(cutoffTime time.Time, languageCode string) (int64, error) {
	count := int64(0)
	for _, article := range m.articles {
		if article.PublishedAt != nil && article.PublishedAt.Before(cutoffTime) && article.LanguageCode == languageCode {
			count++
		}
	}
	return count, nil
}

func (m *mockArticleRepository) CountPublishedArticlesAfterTime(cutoffTime time.Time, languageCode string) (int64, error) {
	count := int64(0)
	for _, article := range m.articles {
		if article.PublishedAt != nil && article.PublishedAt.After(cutoffTime) && article.LanguageCode == languageCode {
			count++
		}
	}
	return count, nil
}

type mockCategoryRepository struct {
	categories []models.Category
}

func (m *mockCategoryRepository) GetBySlug(slug, languageCode string) (*models.Category, error) {
	for _, category := range m.categories {
		if category.Slug == slug && category.LanguageCode == languageCode {
			return &category, nil
		}
	}
	return nil, fmt.Errorf("category not found")
}

type mockTagRepository struct {
	tags []models.Tag
}

func (m *mockTagRepository) GetBySlug(slug, languageCode string) (*models.Tag, error) {
	for _, tag := range m.tags {
		if tag.Slug == slug && tag.LanguageCode == languageCode {
			return &tag, nil
		}
	}
	return nil, fmt.Errorf("tag not found")
}

type mockRSSCache struct {
	data map[string][]byte
}

func (m *mockRSSCache) Get(ctx context.Context, key string) ([]byte, error) {
	if data, exists := m.data[key]; exists {
		return data, nil
	}
	return nil, fmt.Errorf("key not found")
}

func (m *mockRSSCache) Set(ctx context.Context, key string, value []byte, duration time.Duration) error {
	if m.data == nil {
		m.data = make(map[string][]byte)
	}
	m.data[key] = value
	return nil
}

func (m *mockRSSCache) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *mockRSSCache) DeletePattern(ctx context.Context, pattern string) error {
	// Simple pattern matching for tests
	for key := range m.data {
		if strings.Contains(pattern, "*") {
			prefix := strings.Split(pattern, "*")[0]
			if strings.HasPrefix(key, prefix) {
				delete(m.data, key)
			}
		} else if key == pattern {
			delete(m.data, key)
		}
	}
	return nil
}

func (m *mockRSSCache) Exists(ctx context.Context, key string) (bool, error) {
	_, exists := m.data[key]
	return exists, nil
}

func (m *mockRSSCache) Close() error {
	return nil
}

func (m *mockRSSCache) Health(ctx context.Context) error {
	return nil
}

func createTestRSSService() *RSSService {
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
				{ID: 2, Name: "News", Slug: "news"},
			},
			SEOData: models.SEOData{
				Keywords: []string{"tech", "innovation"},
			},
		},
		{
			ID:          2,
			Title:       "Test Article 2",
			Slug:        "test-article-2",
			Content:     "This is test content for article 2",
			Excerpt:     "Test excerpt 2",
			AuthorID:    1,
			CategoryID:  2,
			Status:      "published",
			PublishedAt: &publishedTime,
			LanguageCode: "fa",
			Tags: []models.Tag{
				{ID: 3, Name: "Business", Slug: "business"},
			},
		},
	}

	categories := []models.Category{
		{ID: 1, Name: "Technology", Slug: "technology", Description: "Tech news", LanguageCode: "fa"},
		{ID: 2, Name: "Business", Slug: "business", Description: "Business news", LanguageCode: "fa"},
	}

	tags := []models.Tag{
		{ID: 1, Name: "Technology", Slug: "technology", LanguageCode: "fa"},
		{ID: 2, Name: "News", Slug: "news", LanguageCode: "fa"},
		{ID: 3, Name: "Business", Slug: "business", LanguageCode: "fa"},
	}

	articleRepo := &mockArticleRepository{articles: articles}
	categoryRepo := &mockCategoryRepository{categories: categories}
	tagRepo := &mockTagRepository{tags: tags}
	cache := &mockCache{}

	return NewRSSService(
		articleRepo,
		categoryRepo,
		tagRepo,
		cache,
		"https://example.com",
		"Test News Site",
		"Test news site description",
	)
}

func TestRSSService_GenerateMainRSSFeed(t *testing.T) {
	service := createTestRSSService()

	xmlData, err := service.GenerateMainRSSFeed("fa", 10)
	if err != nil {
		t.Fatalf("Failed to generate main RSS feed: %v", err)
	}

	// Validate XML structure
	var rss RSS
	if err := xml.Unmarshal(xmlData, &rss); err != nil {
		t.Fatalf("Failed to unmarshal RSS XML: %v", err)
	}

	// Validate RSS structure
	if rss.Version != "2.0" {
		t.Errorf("Expected RSS version 2.0, got %s", rss.Version)
	}

	if rss.Channel.Title != "Test News Site" {
		t.Errorf("Expected title 'Test News Site', got %s", rss.Channel.Title)
	}

	if rss.Channel.Link != "https://example.com" {
		t.Errorf("Expected link 'https://example.com', got %s", rss.Channel.Link)
	}

	if len(rss.Channel.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(rss.Channel.Items))
	}

	// Validate first item
	item := rss.Channel.Items[0]
	if item.Title != "Test Article 1" {
		t.Errorf("Expected item title 'Test Article 1', got %s", item.Title)
	}

	if item.Link != "https://example.com/article/test-article-1" {
		t.Errorf("Expected item link 'https://example.com/article/test-article-1', got %s", item.Link)
	}

	if !item.GUID.IsPermaLink {
		t.Error("Expected GUID to be permalink")
	}
}

func TestRSSService_GenerateCategoryRSSFeed(t *testing.T) {
	service := createTestRSSService()

	xmlData, err := service.GenerateCategoryRSSFeed("technology", "fa", 10)
	if err != nil {
		t.Fatalf("Failed to generate category RSS feed: %v", err)
	}

	// Validate XML structure
	var rss RSS
	if err := xml.Unmarshal(xmlData, &rss); err != nil {
		t.Fatalf("Failed to unmarshal RSS XML: %v", err)
	}

	// Should contain only articles from technology category
	if len(rss.Channel.Items) != 1 {
		t.Errorf("Expected 1 item for technology category, got %d", len(rss.Channel.Items))
	}

	if rss.Channel.Title != "Test News Site - Technology" {
		t.Errorf("Expected title 'Test News Site - Technology', got %s", rss.Channel.Title)
	}

	if rss.Channel.Category != "Technology" {
		t.Errorf("Expected category 'Technology', got %s", rss.Channel.Category)
	}
}

func TestRSSService_GenerateTagRSSFeed(t *testing.T) {
	service := createTestRSSService()

	xmlData, err := service.GenerateTagRSSFeed("business", "fa", 10)
	if err != nil {
		t.Fatalf("Failed to generate tag RSS feed: %v", err)
	}

	// Validate XML structure
	var rss RSS
	if err := xml.Unmarshal(xmlData, &rss); err != nil {
		t.Fatalf("Failed to unmarshal RSS XML: %v", err)
	}

	// Should contain only articles tagged with business
	if len(rss.Channel.Items) != 1 {
		t.Errorf("Expected 1 item for business tag, got %d", len(rss.Channel.Items))
	}

	if rss.Channel.Title != "Test News Site - Business" {
		t.Errorf("Expected title 'Test News Site - Business', got %s", rss.Channel.Title)
	}
}

func TestRSSService_GenerateGoogleNewsRSSFeed(t *testing.T) {
	service := createTestRSSService()

	xmlData, err := service.GenerateGoogleNewsRSSFeed("fa", 100)
	if err != nil {
		t.Fatalf("Failed to generate Google News RSS feed: %v", err)
	}

	// Validate XML structure
	var rss GoogleNewsRSS
	if err := xml.Unmarshal(xmlData, &rss); err != nil {
		t.Fatalf("Failed to unmarshal Google News RSS XML: %v", err)
	}

	// Validate Google News structure
	if rss.Version != "2.0" {
		t.Errorf("Expected RSS version 2.0, got %s", rss.Version)
	}

	if rss.Xmlns != "http://www.google.com/schemas/sitemap-news/0.9" {
		t.Errorf("Expected Google News namespace, got %s", rss.Xmlns)
	}

	if len(rss.Channel.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(rss.Channel.Items))
	}

	// Validate Google News specific fields
	item := rss.Channel.Items[0]
	if item.News.Publication.Name != "Test News Site" {
		t.Errorf("Expected publication name 'Test News Site', got %s", item.News.Publication.Name)
	}

	if item.News.Publication.Language != "fa" {
		t.Errorf("Expected publication language 'fa', got %s", item.News.Publication.Language)
	}

	if item.News.Keywords == "" {
		t.Error("Expected keywords to be populated")
	}
}

func TestRSSService_ValidateRSSFeed(t *testing.T) {
	service := createTestRSSService()

	// Generate valid RSS feed
	xmlData, err := service.GenerateMainRSSFeed("fa", 10)
	if err != nil {
		t.Fatalf("Failed to generate RSS feed: %v", err)
	}

	// Validate should pass
	if err := service.ValidateRSSFeed(xmlData); err != nil {
		t.Errorf("Valid RSS feed failed validation: %v", err)
	}

	// Test invalid XML
	invalidXML := []byte("invalid xml")
	if err := service.ValidateRSSFeed(invalidXML); err == nil {
		t.Error("Expected validation to fail for invalid XML")
	}
}

func TestRSSService_ValidateGoogleNewsRSSFeed(t *testing.T) {
	service := createTestRSSService()

	// Generate valid Google News RSS feed
	xmlData, err := service.GenerateGoogleNewsRSSFeed("fa", 100)
	if err != nil {
		t.Fatalf("Failed to generate Google News RSS feed: %v", err)
	}

	// Validate should pass
	if err := service.ValidateGoogleNewsRSSFeed(xmlData); err != nil {
		t.Errorf("Valid Google News RSS feed failed validation: %v", err)
	}

	// Test invalid XML
	invalidXML := []byte("invalid xml")
	if err := service.ValidateGoogleNewsRSSFeed(invalidXML); err == nil {
		t.Error("Expected validation to fail for invalid XML")
	}
}

func TestRSSService_DelayLogic(t *testing.T) {
	service := createTestRSSService()

	// Test with default 2-hour delay
	if service.GetDelayHours() != 2 {
		t.Errorf("Expected default delay of 2 hours, got %d", service.GetDelayHours())
	}

	// Test setting custom delay
	service.SetDelayHours(4)
	if service.GetDelayHours() != 4 {
		t.Errorf("Expected delay of 4 hours, got %d", service.GetDelayHours())
	}

	// Test negative delay (should not change)
	service.SetDelayHours(-1)
	if service.GetDelayHours() != 4 {
		t.Errorf("Expected delay to remain 4 hours, got %d", service.GetDelayHours())
	}
}

func TestRSSService_CacheInvalidation(t *testing.T) {
	service := createTestRSSService()

	// Generate feed to populate cache
	_, err := service.GenerateMainRSSFeed("fa", 10)
	if err != nil {
		t.Fatalf("Failed to generate RSS feed: %v", err)
	}

	// Invalidate cache
	if err := service.InvalidateCache("fa"); err != nil {
		t.Errorf("Failed to invalidate cache: %v", err)
	}

	// Test specific cache invalidation
	if err := service.InvalidateCategoryCache("technology", "fa"); err != nil {
		t.Errorf("Failed to invalidate category cache: %v", err)
	}

	if err := service.InvalidateTagCache("business", "fa"); err != nil {
		t.Errorf("Failed to invalidate tag cache: %v", err)
	}
}

func TestRSSService_ForceRefreshFeed(t *testing.T) {
	service := createTestRSSService()

	// Test force refresh for different feed types
	testCases := []struct {
		feedType   string
		identifier string
		language   string
		expectErr  bool
	}{
		{"main", "", "fa", false},
		{"category", "technology", "fa", false},
		{"tag", "business", "fa", false},
		{"googlenews", "", "fa", false},
		{"invalid", "", "fa", true},
	}

	for _, tc := range testCases {
		err := service.ForceRefreshFeed(tc.feedType, tc.identifier, tc.language)
		if tc.expectErr && err == nil {
			t.Errorf("Expected error for feed type %s, got nil", tc.feedType)
		}
		if !tc.expectErr && err != nil {
			t.Errorf("Unexpected error for feed type %s: %v", tc.feedType, err)
		}
	}
}

func TestRSSService_GetFeedStats(t *testing.T) {
	service := createTestRSSService()

	stats, err := service.GetFeedStats("fa")
	if err != nil {
		t.Fatalf("Failed to get feed stats: %v", err)
	}

	// Validate stats structure
	if _, exists := stats["total_articles_in_feed"]; !exists {
		t.Error("Expected total_articles_in_feed in stats")
	}

	if _, exists := stats["delay_hours"]; !exists {
		t.Error("Expected delay_hours in stats")
	}

	if _, exists := stats["cutoff_time"]; !exists {
		t.Error("Expected cutoff_time in stats")
	}

	if _, exists := stats["last_updated"]; !exists {
		t.Error("Expected last_updated in stats")
	}
}