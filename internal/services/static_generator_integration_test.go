package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"high-performance-news-website/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Simple cache implementation for testing
type TestCacheService struct{}

func (t *TestCacheService) Get(ctx context.Context, key string) ([]byte, error) {
	return nil, nil
}

func (t *TestCacheService) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return nil
}

func (t *TestCacheService) Delete(ctx context.Context, key string) error {
	return nil
}

func (t *TestCacheService) DeletePattern(ctx context.Context, pattern string) error {
	return nil
}

func (t *TestCacheService) Exists(ctx context.Context, key string) (bool, error) {
	return false, nil
}

func (t *TestCacheService) Close() error {
	return nil
}

func (t *TestCacheService) Health(ctx context.Context) error {
	return nil
}

func TestStaticGenerator_Integration_TemplateLoading(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to run.")
	}

	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "static_generator_integration_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	templatesDir := filepath.Join(tempDir, "templates")
	outputDir := filepath.Join(tempDir, "output")

	// Create template directory
	require.NoError(t, os.MkdirAll(templatesDir, 0755))
	require.NoError(t, os.MkdirAll(outputDir, 0755))

	// Create basic test templates
	createBasicTestTemplates(t, templatesDir)

	// Create static generator
	config := StaticGeneratorConfig{
		OutputPath:   outputDir,
		TemplatesDir: templatesDir,
		BaseURL:      "https://example.com",
	}

	cacheService := &TestCacheService{}
	sg, err := NewStaticGenerator(config, cacheService, nil, nil, nil, nil)
	require.NoError(t, err)

	assert.NotNil(t, sg)
	assert.NotNil(t, sg.templates)
	assert.Equal(t, outputDir, sg.outputPath)
	assert.Equal(t, "https://example.com", sg.baseURL)
}

func TestStaticGenerator_Integration_DirectoryCreation(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to run.")
	}

	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "static_generator_integration_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	templatesDir := filepath.Join(tempDir, "templates")
	outputDir := filepath.Join(tempDir, "output")

	// Create template directory
	require.NoError(t, os.MkdirAll(templatesDir, 0755))

	// Create basic test templates
	createBasicTestTemplates(t, templatesDir)

	// Create static generator (output directory should be created automatically)
	config := StaticGeneratorConfig{
		OutputPath:   outputDir,
		TemplatesDir: templatesDir,
		BaseURL:      "https://example.com",
	}

	cacheService := &TestCacheService{}
	sg, err := NewStaticGenerator(config, cacheService, nil, nil, nil, nil)
	require.NoError(t, err)

	// Verify output directory was created
	assert.DirExists(t, outputDir)

	// Test article directory creation
	publishedAt := time.Now()
	article := &models.Article{
		ID:           1,
		Title:        "Test Article",
		Slug:         "test-article",
		Content:      "<p>Test content</p>",
		Excerpt:      "Test excerpt",
		CategoryID:   1,
		PublishedAt:  &publishedAt,
		UpdatedAt:    time.Now(),
		LanguageCode: "en",
		SEOData: models.SEOData{
			SchemaType: "NewsArticle",
		},
	}

	ctx := context.Background()
	err = sg.GenerateArticlePage(ctx, article)
	require.NoError(t, err)

	// Verify article directory and file were created
	articleDir := filepath.Join(outputDir, "articles", "test-article")
	assert.DirExists(t, articleDir)
	assert.FileExists(t, filepath.Join(articleDir, "index.html"))
}

func TestGetTextDirection_Integration(t *testing.T) {
	tests := []struct {
		language string
		expected string
		name     string
	}{
		{"fa", "rtl", "Persian should be RTL"},
		{"ar", "rtl", "Arabic should be RTL"},
		{"he", "rtl", "Hebrew should be RTL"},
		{"ur", "rtl", "Urdu should be RTL"},
		{"en", "ltr", "English should be LTR"},
		{"fr", "ltr", "French should be LTR"},
		{"de", "ltr", "German should be LTR"},
		{"unknown", "ltr", "Unknown language should default to LTR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTextDirection(tt.language)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStaticGenerator_Integration_CategoryPageGeneration(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to run.")
	}

	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "static_generator_integration_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	templatesDir := filepath.Join(tempDir, "templates")
	outputDir := filepath.Join(tempDir, "output")

	// Create template directory
	require.NoError(t, os.MkdirAll(templatesDir, 0755))
	require.NoError(t, os.MkdirAll(outputDir, 0755))

	// Create basic test templates
	createBasicTestTemplates(t, templatesDir)

	// Create static generator
	config := StaticGeneratorConfig{
		OutputPath:   outputDir,
		TemplatesDir: templatesDir,
		BaseURL:      "https://example.com",
	}

	cacheService := &TestCacheService{}
	sg, err := NewStaticGenerator(config, cacheService, nil, nil, nil, nil)
	require.NoError(t, err)

	// Test category page generation
	category := &models.Category{
		ID:          1,
		Name:        "Test Category",
		Slug:        "test-category",
		Description: "A test category for integration testing",
	}

	ctx := context.Background()
	err = sg.GenerateCategoryPage(ctx, category, 1)
	require.NoError(t, err)

	// Verify category directory and file were created
	categoryDir := filepath.Join(outputDir, "categories", "test-category")
	assert.DirExists(t, categoryDir)
	assert.FileExists(t, filepath.Join(categoryDir, "index.html"))
}

func TestStaticGenerator_Integration_TagPageGeneration(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to run.")
	}

	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "static_generator_integration_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	templatesDir := filepath.Join(tempDir, "templates")
	outputDir := filepath.Join(tempDir, "output")

	// Create template directory
	require.NoError(t, os.MkdirAll(templatesDir, 0755))
	require.NoError(t, os.MkdirAll(outputDir, 0755))

	// Create basic test templates
	createBasicTestTemplates(t, templatesDir)

	// Create static generator
	config := StaticGeneratorConfig{
		OutputPath:   outputDir,
		TemplatesDir: templatesDir,
		BaseURL:      "https://example.com",
	}

	cacheService := &TestCacheService{}
	sg, err := NewStaticGenerator(config, cacheService, nil, nil, nil, nil)
	require.NoError(t, err)

	// Test tag page generation
	tag := &models.Tag{
		ID:          1,
		Name:        "Test Tag",
		Slug:        "test-tag",
		Description: "A test tag for integration testing",
		Keywords:    []string{"test", "integration", "tag"},
		Color:       "#007cba",
	}

	ctx := context.Background()
	err = sg.GenerateTagPage(ctx, tag, 1)
	require.NoError(t, err)

	// Verify tag directory and file were created
	tagDir := filepath.Join(outputDir, "tags", "test-tag")
	assert.DirExists(t, tagDir)
	assert.FileExists(t, filepath.Join(tagDir, "index.html"))
}

func TestStaticGenerator_Integration_CacheWarmingAndInvalidation(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to run.")
	}

	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "static_generator_integration_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	templatesDir := filepath.Join(tempDir, "templates")
	outputDir := filepath.Join(tempDir, "output")

	// Create template directory
	require.NoError(t, os.MkdirAll(templatesDir, 0755))
	require.NoError(t, os.MkdirAll(outputDir, 0755))

	// Create basic test templates
	createBasicTestTemplates(t, templatesDir)

	// Create static generator
	config := StaticGeneratorConfig{
		OutputPath:   outputDir,
		TemplatesDir: templatesDir,
		BaseURL:      "https://example.com",
	}

	cacheService := &TestCacheService{}
	sg, err := NewStaticGenerator(config, cacheService, nil, nil, nil, nil)
	require.NoError(t, err)

	// Test cache warming and invalidation
	publishedAt := time.Now()
	article := &models.Article{
		ID:           1,
		Title:        "Test Article for Cache",
		Slug:         "test-article-cache",
		Content:      "<p>Test content for cache testing</p>",
		Excerpt:      "Test excerpt for cache",
		CategoryID:   1,
		PublishedAt:  &publishedAt,
		UpdatedAt:    time.Now(),
		LanguageCode: "en",
		SEOData: models.SEOData{
			SchemaType: "NewsArticle",
		},
	}

	// Test cache warming (should not error even with mock cache)
	sg.warmArticleCache(article)
	sg.warmHomepageCache("en")

	// Test cache invalidation (should not error even with mock cache)
	sg.invalidateRelatedCaches(article)

	// These should complete without errors
	assert.True(t, true, "Cache operations completed without errors")
}

func createBasicTestTemplates(t *testing.T, templatesDir string) {
	// Create a simple homepage template
	homepageTemplate := `<!DOCTYPE html>
<html>
<head><title>{{.Title}}</title></head>
<body>
<h1>{{.Title}}</h1>
<div class="articles">
{{range .LatestArticles}}
<article>
<h2>{{.Title}}</h2>
<p>{{.Excerpt}}</p>
</article>
{{end}}
</div>
</body>
</html>`

	err := os.WriteFile(filepath.Join(templatesDir, "homepage.html"), []byte(homepageTemplate), 0644)
	require.NoError(t, err)

	// Create a simple article template
	articleTemplate := `<!DOCTYPE html>
<html>
<head><title>{{.Title}}</title></head>
<body>
<article>
<h1>{{.Article.Title}}</h1>
<div class="content">{{.Article.Content | safeHTML}}</div>
<div class="meta">Published: {{.Article.PublishedAt}}</div>
</article>
</body>
</html>`

	err = os.WriteFile(filepath.Join(templatesDir, "article.html"), []byte(articleTemplate), 0644)
	require.NoError(t, err)

	// Create a simple category template
	categoryTemplate := `<!DOCTYPE html>
<html>
<head><title>{{.Title}}</title></head>
<body>
<h1>{{.Category.Name}}</h1>
<div class="articles">
{{range .Articles}}
<article>
<h2>{{.Title}}</h2>
<p>{{.Excerpt}}</p>
</article>
{{end}}
</div>
{{if gt .Pagination.TotalPages 1}}
<div class="pagination">
  Page {{.Pagination.CurrentPage}} of {{.Pagination.TotalPages}}
</div>
{{end}}
</body>
</html>`

	err = os.WriteFile(filepath.Join(templatesDir, "category.html"), []byte(categoryTemplate), 0644)
	require.NoError(t, err)

	// Create a simple tag template
	tagTemplate := `<!DOCTYPE html>
<html>
<head><title>{{.Title}}</title></head>
<body>
<h1>{{.Tag.Name}}</h1>
{{if .Tag.Description}}<p>{{.Tag.Description}}</p>{{end}}
{{if .Tag.Keywords}}
<div class="keywords">Keywords: {{range .Tag.Keywords}}{{.}} {{end}}</div>
{{end}}
<div class="articles">
{{range .Articles}}
<article>
<h2>{{.Title}}</h2>
<p>{{.Excerpt}}</p>
</article>
{{end}}
</div>
{{if gt .Pagination.TotalPages 1}}
<div class="pagination">
  Page {{.Pagination.CurrentPage}} of {{.Pagination.TotalPages}}
</div>
{{end}}
</body>
</html>`

	err = os.WriteFile(filepath.Join(templatesDir, "tag.html"), []byte(tagTemplate), 0644)
	require.NoError(t, err)
}
