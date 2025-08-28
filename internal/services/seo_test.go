package services

import (
	"strings"
	"testing"
	"time"

	"high-performance-news-website/internal/models"
)

func TestSEOService_GenerateArticleSchema(t *testing.T) {
	seoService := NewSEOService("https://example.com", "Test Site", "en")

	// Create test article
	publishedAt := time.Now()
	article := &models.Article{
		ID:          1,
		Title:       "Test Article Title",
		Slug:        "test-article-title",
		Content:     "This is a test article content with multiple words to test word counting functionality.",
		Excerpt:     "Test article excerpt",
		Status:      "published",
		PublishedAt: &publishedAt,
		UpdatedAt:   time.Now(),
		LanguageCode: "en",
		SEOData: models.SEOData{
			MetaTitle:       "Custom Meta Title",
			MetaDescription: "Custom meta description",
			Keywords:        []string{"test", "article", "seo"},
			SchemaType:      "NewsArticle",
		},
		Tags: []models.Tag{
			{Name: "Technology", Slug: "technology"},
			{Name: "News", Slug: "news"},
		},
	}

	author := &models.User{
		FirstName: "John",
		LastName:  "Doe",
		Username:  "johndoe",
	}

	category := &models.Category{
		Name: "Technology",
		Slug: "technology",
	}

	schema, err := seoService.GenerateArticleSchema(article, author, category)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test schema properties
	if schema.Context != "https://schema.org" {
		t.Errorf("Expected context 'https://schema.org', got '%s'", schema.Context)
	}

	if schema.Type != "NewsArticle" {
		t.Errorf("Expected type 'NewsArticle', got '%s'", schema.Type)
	}

	if schema.Headline != article.Title {
		t.Errorf("Expected headline '%s', got '%s'", article.Title, schema.Headline)
	}

	if schema.Description != article.Excerpt {
		t.Errorf("Expected description '%s', got '%s'", article.Excerpt, schema.Description)
	}

	expectedURL := "https://example.com/article/test-article-title"
	if schema.URL != expectedURL {
		t.Errorf("Expected URL '%s', got '%s'", expectedURL, schema.URL)
	}

	// Test word count
	expectedWordCount := 15 // Approximate word count of the content
	if schema.WordCount < 10 || schema.WordCount > 20 {
		t.Errorf("Expected word count around %d, got %d", expectedWordCount, schema.WordCount)
	}

	// Test keywords
	if len(schema.Keywords) != 3 {
		t.Errorf("Expected 3 keywords, got %d", len(schema.Keywords))
	}

	// Test author information
	authorData, ok := schema.Author.(map[string]interface{})
	if !ok {
		t.Error("Expected author to be a map")
	} else {
		if authorData["name"] != "John Doe" {
			t.Errorf("Expected author name 'John Doe', got '%v'", authorData["name"])
		}
	}

	// Test article section
	if schema.ArticleSection != category.Name {
		t.Errorf("Expected article section '%s', got '%s'", category.Name, schema.ArticleSection)
	}
}

func TestSEOService_GenerateHomepageSchema(t *testing.T) {
	seoService := NewSEOService("https://example.com", "Test Site", "en")

	schema, err := seoService.GenerateHomepageSchema()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if schema.Context != "https://schema.org" {
		t.Errorf("Expected context 'https://schema.org', got '%s'", schema.Context)
	}

	if schema.Type != "WebSite" {
		t.Errorf("Expected type 'WebSite', got '%s'", schema.Type)
	}

	if schema.URL != "https://example.com" {
		t.Errorf("Expected URL 'https://example.com', got '%s'", schema.URL)
	}

	if schema.Headline != "Test Site" {
		t.Errorf("Expected headline 'Test Site', got '%s'", schema.Headline)
	}
}

func TestSEOService_GenerateMetaTags(t *testing.T) {
	seoService := NewSEOService("https://example.com", "Test Site", "en")

	// Test article meta tags
	publishedAt := time.Now()
	article := &models.Article{
		Title:       "Test Article",
		Slug:        "test-article",
		Excerpt:     "Test excerpt",
		Status:      "published",
		PublishedAt: &publishedAt,
		UpdatedAt:   time.Now(),
		LanguageCode: "en",
		SEOData: models.SEOData{
			MetaTitle:       "Custom Meta Title",
			MetaDescription: "Custom meta description",
			Keywords:        []string{"test", "article"},
		},
		Tags: []models.Tag{
			{Name: "Technology", Keywords: []string{"tech", "innovation"}},
		},
	}

	meta, err := seoService.GenerateMetaTags("article", article)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if meta.Title != "Custom Meta Title" {
		t.Errorf("Expected title 'Custom Meta Title', got '%s'", meta.Title)
	}

	if meta.Description != "Custom meta description" {
		t.Errorf("Expected description 'Custom meta description', got '%s'", meta.Description)
	}

	// Test enhanced keywords (should include both SEO keywords and tag keywords)
	expectedKeywords := "test, article, Technology, tech, innovation"
	if meta.Keywords != expectedKeywords {
		t.Errorf("Expected keywords '%s', got '%s'", expectedKeywords, meta.Keywords)
	}

	if meta.Language != "en" {
		t.Errorf("Expected language 'en', got '%s'", meta.Language)
	}

	if meta.OGType != "article" {
		t.Errorf("Expected OG type 'article', got '%s'", meta.OGType)
	}

	expectedCanonical := "https://example.com/article/test-article"
	if meta.CanonicalURL != expectedCanonical {
		t.Errorf("Expected canonical URL '%s', got '%s'", expectedCanonical, meta.CanonicalURL)
	}

	// Test enhanced Open Graph properties
	if meta.OGSiteName != "Test Site" {
		t.Errorf("Expected OG site name 'Test Site', got '%s'", meta.OGSiteName)
	}

	if meta.OGPublishedTime == "" {
		t.Error("Expected OG published time to be set")
	}

	if meta.OGModifiedTime == "" {
		t.Error("Expected OG modified time to be set")
	}

	// Test enhanced Twitter Card properties
	if meta.TwitterCard != "summary_large_image" {
		t.Errorf("Expected Twitter card 'summary_large_image', got '%s'", meta.TwitterCard)
	}

	if meta.TwitterLabel1 != "Published" {
		t.Errorf("Expected Twitter label1 'Published', got '%s'", meta.TwitterLabel1)
	}

	if meta.TwitterData1 == "" {
		t.Error("Expected Twitter data1 to be set")
	}

	// Test robots directive for published article
	expectedRobots := "index, follow, max-snippet:-1, max-image-preview:large, max-video-preview:-1"
	if meta.Robots != expectedRobots {
		t.Errorf("Expected robots '%s', got '%s'", expectedRobots, meta.Robots)
	}
}

func TestSEOService_GenerateCategoryMetaTags(t *testing.T) {
	seoService := NewSEOService("https://example.com", "Test Site", "en")

	category := &models.Category{
		Name:         "Technology",
		Slug:         "technology",
		Description:  "Technology news and updates",
		LanguageCode: "en",
	}

	meta, err := seoService.GenerateMetaTags("category", category)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expectedTitle := "Technology - Test Site"
	if meta.Title != expectedTitle {
		t.Errorf("Expected title '%s', got '%s'", expectedTitle, meta.Title)
	}

	if meta.Description != category.Description {
		t.Errorf("Expected description '%s', got '%s'", category.Description, meta.Description)
	}

	expectedCanonical := "https://example.com/category/technology"
	if meta.CanonicalURL != expectedCanonical {
		t.Errorf("Expected canonical URL '%s', got '%s'", expectedCanonical, meta.CanonicalURL)
	}
}

func TestSEOService_GenerateTagMetaTags(t *testing.T) {
	seoService := NewSEOService("https://example.com", "Test Site", "en")

	tag := &models.Tag{
		Name:         "golang",
		Slug:         "golang",
		Description:  "Go programming language",
		Keywords:     []string{"go", "golang", "programming"},
		LanguageCode: "en",
	}

	meta, err := seoService.GenerateMetaTags("tag", tag)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expectedTitle := "golang - Test Site"
	if meta.Title != expectedTitle {
		t.Errorf("Expected title '%s', got '%s'", expectedTitle, meta.Title)
	}

	if meta.Description != tag.Description {
		t.Errorf("Expected description '%s', got '%s'", tag.Description, meta.Description)
	}

	expectedKeywords := "go, golang, programming"
	if meta.Keywords != expectedKeywords {
		t.Errorf("Expected keywords '%s', got '%s'", expectedKeywords, meta.Keywords)
	}

	expectedCanonical := "https://example.com/tag/golang"
	if meta.CanonicalURL != expectedCanonical {
		t.Errorf("Expected canonical URL '%s', got '%s'", expectedCanonical, meta.CanonicalURL)
	}
}

func TestSEOService_RenderSchemaJSON(t *testing.T) {
	seoService := NewSEOService("https://example.com", "Test Site", "en")

	schema := &SchemaMarkup{
		Context:     "https://schema.org",
		Type:        "Article",
		Headline:    "Test Article",
		Description: "Test description",
		URL:         "https://example.com/test",
	}

	jsonLD, err := seoService.RenderSchemaJSON(schema)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	jsonStr := string(jsonLD)

	// Check that it contains script tags
	if !strings.Contains(jsonStr, `<script type="application/ld+json">`) {
		t.Error("Expected JSON-LD to contain script tag")
	}

	if !strings.Contains(jsonStr, `</script>`) {
		t.Error("Expected JSON-LD to contain closing script tag")
	}

	// Check that it contains the schema data
	if !strings.Contains(jsonStr, `"@context": "https://schema.org"`) {
		t.Error("Expected JSON-LD to contain schema context")
	}

	if !strings.Contains(jsonStr, `"@type": "Article"`) {
		t.Error("Expected JSON-LD to contain schema type")
	}

	if !strings.Contains(jsonStr, `"headline": "Test Article"`) {
		t.Error("Expected JSON-LD to contain headline")
	}
}

func TestSEOService_URLGeneration(t *testing.T) {
	seoService := NewSEOService("https://example.com", "Test Site", "en")

	tests := []struct {
		method   func(string) string
		slug     string
		expected string
	}{
		{seoService.GetArticleURL, "test-article", "https://example.com/article/test-article"},
		{seoService.GetCategoryURL, "technology", "https://example.com/category/technology"},
		{seoService.GetTagURL, "golang", "https://example.com/tag/golang"},
		{seoService.GetAuthorURL, "johndoe", "https://example.com/author/johndoe"},
	}

	for _, test := range tests {
		result := test.method(test.slug)
		if result != test.expected {
			t.Errorf("Expected URL '%s', got '%s'", test.expected, result)
		}
	}
}

func TestSEOService_UtilityFunctions(t *testing.T) {
	seoService := NewSEOService("https://example.com", "Test Site", "en")

	// Test word counting
	content := "This is a test content with exactly ten words here."
	wordCount := seoService.countWords(content)
	if wordCount != 10 {
		t.Errorf("Expected word count 10, got %d", wordCount)
	}

	// Test HTML stripping
	htmlContent := "<p>This is <strong>HTML</strong> content with <a href='#'>links</a>.</p>"
	stripped := seoService.stripHTML(htmlContent)
	expected := "This is HTML content with links."
	if stripped != expected {
		t.Errorf("Expected stripped content '%s', got '%s'", expected, stripped)
	}

	// Test text truncation
	longText := "This is a very long text that should be truncated at some point to fit within the specified length limit."
	truncated := seoService.truncateText(longText, 50)
	if len(truncated) > 53 { // 50 + "..." = 53
		t.Errorf("Expected truncated text to be at most 53 characters, got %d", len(truncated))
	}
	if !strings.HasSuffix(truncated, "...") {
		t.Error("Expected truncated text to end with '...'")
	}

	// Test reading time calculation
	longContent := strings.Repeat("word ", 400) // 400 words
	readingTime := seoService.calculateReadingTime(longContent)
	if readingTime != 2 { // 400 words / 200 words per minute = 2 minutes
		t.Errorf("Expected reading time 2 minutes, got %d", readingTime)
	}
}

func TestSEOService_ErrorHandling(t *testing.T) {
	seoService := NewSEOService("https://example.com", "Test Site", "en")

	// Test with nil article
	_, err := seoService.GenerateArticleSchema(nil, nil, nil)
	if err == nil {
		t.Error("Expected error for nil article, got none")
	}

	// Test with invalid page type
	_, err = seoService.GenerateMetaTags("invalid", nil)
	if err == nil {
		t.Error("Expected error for invalid page type, got none")
	}

	// Test URL validation
	err = seoService.ValidateURL("invalid-url")
	if err == nil {
		t.Error("Expected error for invalid URL, got none")
	}

	err = seoService.ValidateURL("https://example.com/valid")
	if err != nil {
		t.Errorf("Expected no error for valid URL, got %v", err)
	}
}

func TestSEOService_SchemaTypes(t *testing.T) {
	seoService := NewSEOService("https://example.com", "Test Site", "en")

	// Test different schema types
	schemaTypes := []string{"NewsArticle", "Article", "BlogPosting"}

	for _, schemaType := range schemaTypes {
		article := &models.Article{
			Title:       "Test Article",
			Slug:        "test-article",
			Content:     "Test content",
			Excerpt:     "Test excerpt",
			Status:      "published",
			PublishedAt: &time.Time{},
			UpdatedAt:   time.Now(),
			SEOData: models.SEOData{
				SchemaType: schemaType,
			},
		}

		schema, err := seoService.GenerateArticleSchema(article, nil, nil)
		if err != nil {
			t.Fatalf("Expected no error for schema type %s, got %v", schemaType, err)
		}

		if schema.Type != schemaType {
			t.Errorf("Expected schema type '%s', got '%s'", schemaType, schema.Type)
		}
	}
}

func TestSEOService_MultilingualSupport(t *testing.T) {
	seoService := NewSEOService("https://example.com", "Test Site", "en")

	languages := []string{"en", "fa", "ar"}

	for _, lang := range languages {
		article := &models.Article{
			Title:        "Test Article",
			Slug:         "test-article",
			Content:      "Test content",
			Excerpt:      "Test excerpt",
			LanguageCode: lang,
		}

		meta, err := seoService.GenerateMetaTags("article", article)
		if err != nil {
			t.Fatalf("Expected no error for language %s, got %v", lang, err)
		}

		if meta.Language != lang {
			t.Errorf("Expected language '%s', got '%s'", lang, meta.Language)
		}
	}
}

// Benchmark tests
func BenchmarkSEOService_GenerateArticleSchema(b *testing.B) {
	seoService := NewSEOService("https://example.com", "Test Site", "en")
	
	publishedAt := time.Now()
	article := &models.Article{
		ID:          1,
		Title:       "Benchmark Test Article",
		Slug:        "benchmark-test-article",
		Content:     "This is benchmark test content for performance testing.",
		Excerpt:     "Benchmark test excerpt",
		Status:      "published",
		PublishedAt: &publishedAt,
		UpdatedAt:   time.Now(),
		LanguageCode: "en",
	}

	author := &models.User{
		FirstName: "Benchmark",
		LastName:  "User",
		Username:  "benchmarkuser",
	}

	category := &models.Category{
		Name: "Benchmark",
		Slug: "benchmark",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := seoService.GenerateArticleSchema(article, author, category)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSEOService_GenerateMetaTags(b *testing.B) {
	seoService := NewSEOService("https://example.com", "Test Site", "en")
	
	article := &models.Article{
		Title:       "Benchmark Test Article",
		Slug:        "benchmark-test-article",
		Excerpt:     "Benchmark test excerpt",
		LanguageCode: "en",
		SEOData: models.SEOData{
			MetaTitle:       "Benchmark Meta Title",
			MetaDescription: "Benchmark meta description",
			Keywords:        []string{"benchmark", "test", "seo"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := seoService.GenerateMetaTags("article", article)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Test enhanced schema generation
func TestSEOService_EnhancedArticleSchema(t *testing.T) {
	seoService := NewSEOService("https://example.com", "Test Site", "en")

	publishedAt := time.Now()
	article := &models.Article{
		ID:          1,
		Title:       "Enhanced Test Article",
		Slug:        "enhanced-test-article",
		Content:     "This is enhanced test content with multiple words to test enhanced word counting and schema generation functionality.",
		Excerpt:     "Enhanced test article excerpt",
		Status:      "published",
		PublishedAt: &publishedAt,
		UpdatedAt:   time.Now(),
		LanguageCode: "en",
		SEOData: models.SEOData{
			MetaTitle:       "Enhanced Meta Title",
			MetaDescription: "Enhanced meta description",
			Keywords:        []string{"enhanced", "test", "seo"},
			SchemaType:      "NewsArticle",
		},
		Tags: []models.Tag{
			{Name: "Technology", Slug: "technology", Keywords: []string{"tech", "innovation"}},
			{Name: "News", Slug: "news", Keywords: []string{"breaking", "updates"}},
		},
	}

	author := &models.User{
		FirstName: "John",
		LastName:  "Doe",
		Username:  "johndoe",
		Bio:       "Technology journalist and writer",
		Avatar:    "https://example.com/avatars/johndoe.jpg",
	}

	category := &models.Category{
		Name: "Technology",
		Slug: "technology",
	}

	schema, err := seoService.GenerateArticleSchema(article, author, category)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test enhanced author information
	authorData, ok := schema.Author.(map[string]interface{})
	if !ok {
		t.Error("Expected author to be a map")
	} else {
		if authorData["name"] != "John Doe" {
			t.Errorf("Expected author name 'John Doe', got '%v'", authorData["name"])
		}
		
		if authorData["description"] != author.Bio {
			t.Errorf("Expected author bio '%s', got '%v'", author.Bio, authorData["description"])
		}
		
		authorImage, ok := authorData["image"].(map[string]interface{})
		if !ok {
			t.Error("Expected author image to be a map")
		} else if authorImage["url"] != author.Avatar {
			t.Errorf("Expected author image URL '%s', got '%v'", author.Avatar, authorImage["url"])
		}
	}

	// Test enhanced publisher information
	publisherData, ok := schema.Publisher.(map[string]interface{})
	if !ok {
		t.Error("Expected publisher to be a map")
	} else {
		if publisherData["name"] != "Test Site" {
			t.Errorf("Expected publisher name 'Test Site', got '%v'", publisherData["name"])
		}
		
		sameAs, ok := publisherData["sameAs"].([]string)
		if !ok {
			t.Error("Expected sameAs to be a string slice")
		} else if len(sameAs) != 3 {
			t.Errorf("Expected 3 social profiles, got %d", len(sameAs))
		}
	}

	// Test enhanced keywords (should include all unique keywords)
	expectedKeywordCount := 7 // enhanced, test, seo, Technology, News, tech, innovation, breaking, updates (deduplicated)
	if len(schema.Keywords) > expectedKeywordCount {
		t.Errorf("Expected at most %d keywords, got %d", expectedKeywordCount, len(schema.Keywords))
	}

	// Test image information
	imageData, ok := schema.Image.(map[string]interface{})
	if !ok {
		t.Error("Expected image to be a map")
	} else {
		if imageData["@type"] != "ImageObject" {
			t.Errorf("Expected image type 'ImageObject', got '%v'", imageData["@type"])
		}
	}

	// Test main entity page enhancement
	mainEntity, ok := schema.MainEntityOfPage.(map[string]interface{})
	if !ok {
		t.Error("Expected mainEntityOfPage to be a map")
	} else {
		if mainEntity["name"] != article.Title {
			t.Errorf("Expected main entity name '%s', got '%v'", article.Title, mainEntity["name"])
		}
		
		if mainEntity["inLanguage"] != article.LanguageCode {
			t.Errorf("Expected language '%s', got '%v'", article.LanguageCode, mainEntity["inLanguage"])
		}
		
		about, ok := mainEntity["about"].(map[string]interface{})
		if !ok {
			t.Error("Expected about to be a map")
		} else if about["name"] != category.Name {
			t.Errorf("Expected about name '%s', got '%v'", category.Name, about["name"])
		}
	}
}

// Test keyword deduplication
func TestSEOService_KeywordDeduplication(t *testing.T) {
	seoService := NewSEOService("https://example.com", "Test Site", "en")

	keywords := []string{"test", "Test", "TEST", "article", "Article", "seo", "SEO", "duplicate", "duplicate"}
	result := seoService.deduplicateKeywords(keywords, 5)

	if len(result) > 5 {
		t.Errorf("Expected at most 5 keywords, got %d", len(result))
	}

	// Check for duplicates (case-insensitive)
	seen := make(map[string]bool)
	for _, keyword := range result {
		lower := strings.ToLower(keyword)
		if seen[lower] {
			t.Errorf("Found duplicate keyword: %s", keyword)
		}
		seen[lower] = true
	}
}

// Test meta tag optimization for long titles
func TestSEOService_MetaTagOptimization(t *testing.T) {
	seoService := NewSEOService("https://example.com", "Test Site", "en")

	// Test with very long title
	longTitle := "This is a very long article title that exceeds the optimal SEO length of 60 characters and should be optimized"
	article := &models.Article{
		Title:        longTitle,
		Slug:         "long-title-article",
		Excerpt:      "Test excerpt",
		Status:       "published",
		LanguageCode: "en",
	}

	meta, err := seoService.GenerateMetaTags("article", article)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Title should be optimized (truncated or without site name)
	if len(meta.Title) > 60 {
		t.Errorf("Expected title to be optimized to ≤60 characters, got %d: %s", len(meta.Title), meta.Title)
	}

	// Test with very long description
	longExcerpt := "This is a very long excerpt that exceeds the optimal meta description length of 160 characters and should be truncated to fit within the recommended limits for better SEO performance and user experience in search results."
	article.Excerpt = longExcerpt

	meta, err = seoService.GenerateMetaTags("article", article)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(meta.Description) > 160 {
		t.Errorf("Expected description to be ≤160 characters, got %d: %s", len(meta.Description), meta.Description)
	}
}

// Test robots directive for different article statuses
func TestSEOService_RobotsDirective(t *testing.T) {
	seoService := NewSEOService("https://example.com", "Test Site", "en")

	testCases := []struct {
		status         string
		expectedRobots string
	}{
		{"published", "index, follow, max-snippet:-1, max-image-preview:large, max-video-preview:-1"},
		{"draft", "noindex, nofollow"},
		{"archived", "noindex, nofollow"},
	}

	for _, tc := range testCases {
		article := &models.Article{
			Title:        "Test Article",
			Slug:         "test-article",
			Status:       tc.status,
			LanguageCode: "en",
		}

		meta, err := seoService.GenerateMetaTags("article", article)
		if err != nil {
			t.Fatalf("Expected no error for status %s, got %v", tc.status, err)
		}

		if meta.Robots != tc.expectedRobots {
			t.Errorf("Expected robots '%s' for status %s, got '%s'", tc.expectedRobots, tc.status, meta.Robots)
		}
	}
}

// Test sitemap manager instant updates
func TestSitemapManager_InstantUpdates(t *testing.T) {
	// Mock cache service
	mockCache := &MockCacheService{
		data: make(map[string][]byte),
	}

	sitemapService := NewSitemapService("https://example.com")
	manager := NewSitemapManager(sitemapService, mockCache)

	// Start the update processor
	manager.StartUpdateProcessor()
	defer manager.StopUpdateProcessor()

	// Test article update notification
	manager.NotifyUpdate("article", "create", 1, nil)
	manager.NotifyUpdate("article", "update", 2, nil)
	manager.NotifyUpdate("category", "create", 1, nil)

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	// Verify that the manager is running
	if !manager.isRunning {
		t.Error("Expected manager to be running")
	}

	// Test cache operations
	testKey := "test-key"
	testValue := []byte("test-value")
	
	err := mockCache.Set(testKey, testValue, time.Hour)
	if err != nil {
		t.Fatalf("Expected no error setting cache, got %v", err)
	}

	retrieved, err := mockCache.Get(testKey)
	if err != nil {
		t.Fatalf("Expected no error getting cache, got %v", err)
	}

	if string(retrieved) != string(testValue) {
		t.Errorf("Expected cached value '%s', got '%s'", string(testValue), string(retrieved))
	}
}

// Test breadcrumb HTML generation with microdata
func TestBreadcrumbService_EnhancedHTML(t *testing.T) {
	breadcrumbService := NewBreadcrumbService("https://example.com", "Test Site")

	article := &models.Article{
		Title: "Test Article",
		Slug:  "test-article",
	}

	category := &models.Category{
		Name: "Technology",
		Slug: "technology",
	}

	breadcrumbs, err := breadcrumbService.GenerateArticleBreadcrumbs(article, category)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	html := breadcrumbService.RenderBreadcrumbHTML(breadcrumbs, "custom-breadcrumb")
	htmlStr := string(html)

	// Test microdata presence
	if !strings.Contains(htmlStr, `itemscope itemtype="https://schema.org/BreadcrumbList"`) {
		t.Error("Expected breadcrumb list microdata")
	}

	if !strings.Contains(htmlStr, `itemscope itemtype="https://schema.org/ListItem"`) {
		t.Error("Expected list item microdata")
	}

	if !strings.Contains(htmlStr, `itemprop="name"`) {
		t.Error("Expected name property")
	}

	if !strings.Contains(htmlStr, `itemprop="position"`) {
		t.Error("Expected position property")
	}

	// Test accessibility attributes
	if !strings.Contains(htmlStr, `aria-label="Breadcrumb navigation"`) {
		t.Error("Expected aria-label for navigation")
	}

	if !strings.Contains(htmlStr, `role="navigation"`) {
		t.Error("Expected navigation role")
	}

	if !strings.Contains(htmlStr, `aria-current="page"`) {
		t.Error("Expected aria-current for active item")
	}
}

// Test breadcrumb JSON-LD generation
func TestBreadcrumbService_JSONLD(t *testing.T) {
	breadcrumbService := NewBreadcrumbService("https://example.com", "Test Site")

	tag := &models.Tag{
		Name: "golang",
		Slug: "golang",
	}

	breadcrumbs, err := breadcrumbService.GenerateTagBreadcrumbs(tag)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	jsonLD, err := breadcrumbService.RenderBreadcrumbJSON(breadcrumbs)
	if err != nil {
		t.Fatalf("Expected no error generating JSON-LD, got %v", err)
	}

	jsonStr := string(jsonLD)

	// Test JSON-LD structure
	if !strings.Contains(jsonStr, `"@context": "https://schema.org"`) {
		t.Error("Expected schema.org context")
	}

	if !strings.Contains(jsonStr, `"@type": "BreadcrumbList"`) {
		t.Error("Expected BreadcrumbList type")
	}

	if !strings.Contains(jsonStr, `"itemListElement"`) {
		t.Error("Expected itemListElement property")
	}

	if !strings.Contains(jsonStr, `<script type="application/ld+json">`) {
		t.Error("Expected script tag wrapper")
	}
}

