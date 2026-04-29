package services

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"high-performance-news-website/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStaticGenerator_FileServing_ArticleGeneration(t *testing.T) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "static_generator_file_serving_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	templatesDir := filepath.Join(tempDir, "templates")
	outputDir := filepath.Join(tempDir, "output")

	// Create template directory
	require.NoError(t, os.MkdirAll(templatesDir, 0755))
	require.NoError(t, os.MkdirAll(outputDir, 0755))

	// Create test templates
	createTestTemplatesForFileServing(t, templatesDir)

	// Create static generator
	config := StaticGeneratorConfig{
		OutputPath:   outputDir,
		TemplatesDir: templatesDir,
		BaseURL:      "https://example.com",
	}

	cacheService := &TestCacheService{}
	sg, err := NewStaticGenerator(config, cacheService, nil, nil, nil, nil)
	require.NoError(t, err)

	// Create test article
	publishedAt := time.Now()
	article := &models.Article{
		ID:           1,
		Title:        "Test Article for File Serving",
		Slug:         "test-article-file-serving",
		Content:      "<p>This is test content for file serving validation.</p><p>It includes multiple paragraphs.</p>",
		Excerpt:      "Test excerpt for file serving validation",
		CategoryID:   1,
		AuthorID:     1,
		PublishedAt:  &publishedAt,
		UpdatedAt:    time.Now(),
		LanguageCode: "en",
		ViewCount:    100,
		LikeCount:    10,
		DislikeCount: 2,
		SEOData: models.SEOData{
			MetaTitle:       "Test Article - File Serving",
			MetaDescription: "This is a test article for validating file serving functionality",
			Keywords:        []string{"test", "file", "serving", "static"},
			SchemaType:      "NewsArticle",
		},
		Tags: []models.Tag{
			{
				ID:    1,
				Name:  "Test Tag",
				Slug:  "test-tag",
				Color: "#007cba",
			},
		},
	}

	ctx := context.Background()

	// Generate article page
	err = sg.GenerateArticlePage(ctx, article)
	require.NoError(t, err)

	// Verify file was created
	articleFile := filepath.Join(outputDir, "articles", "test-article-file-serving", "index.html")
	assert.FileExists(t, articleFile)

	// Read and validate file content
	content, err := os.ReadFile(articleFile)
	require.NoError(t, err)

	contentStr := string(content)
	
	// Validate HTML structure
	assert.Contains(t, contentStr, "<!DOCTYPE html>")
	assert.Contains(t, contentStr, "<html")
	assert.Contains(t, contentStr, "</html>")
	
	// Validate article content
	assert.Contains(t, contentStr, "Test Article for File Serving")
	assert.Contains(t, contentStr, "This is test content for file serving validation")
	assert.Contains(t, contentStr, "Test excerpt for file serving validation")
	
	// Validate SEO metadata
	assert.Contains(t, contentStr, "Test Article - File Serving")
	assert.Contains(t, contentStr, "This is a test article for validating file serving functionality")
	assert.Contains(t, contentStr, "https://example.com/articles/test-article-file-serving")
	
	// Validate structured data
	assert.Contains(t, contentStr, "application/ld+json")
	assert.Contains(t, contentStr, "NewsArticle")
	
	// Validate language and direction
	assert.Contains(t, contentStr, `lang="en"`)
	assert.Contains(t, contentStr, `dir="ltr"`)
}

func TestStaticGenerator_FileServing_HomepageGeneration(t *testing.T) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "static_generator_homepage_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	templatesDir := filepath.Join(tempDir, "templates")
	outputDir := filepath.Join(tempDir, "output")

	// Create template directory
	require.NoError(t, os.MkdirAll(templatesDir, 0755))
	require.NoError(t, os.MkdirAll(outputDir, 0755))

	// Create test templates
	createTestTemplatesForFileServing(t, templatesDir)

	// Create static generator
	config := StaticGeneratorConfig{
		OutputPath:   outputDir,
		TemplatesDir: templatesDir,
		BaseURL:      "https://example.com",
	}

	cacheService := &TestCacheService{}
	sg, err := NewStaticGenerator(config, cacheService, nil, nil, nil, nil)
	require.NoError(t, err)

	ctx := context.Background()

	// Generate homepage for different languages
	languages := []string{"fa", "en", "ar"}
	for _, lang := range languages {
		err = sg.GenerateHomepage(ctx, lang)
		require.NoError(t, err)

		// Verify file was created
		var homepageFile string
		if lang == "fa" {
			homepageFile = filepath.Join(outputDir, "index.html")
		} else {
			homepageFile = filepath.Join(outputDir, lang, "index.html")
		}
		
		assert.FileExists(t, homepageFile)

		// Read and validate file content
		content, err := os.ReadFile(homepageFile)
		require.NoError(t, err)

		contentStr := string(content)
		
		// Validate HTML structure
		assert.Contains(t, contentStr, "<!DOCTYPE html>")
		assert.Contains(t, contentStr, "<html")
		assert.Contains(t, contentStr, "</html>")
		
		// Validate language-specific attributes
		assert.Contains(t, contentStr, `lang="`+lang+`"`)
		
		// Validate direction based on language
		if lang == "fa" || lang == "ar" {
			assert.Contains(t, contentStr, `dir="rtl"`)
		} else {
			assert.Contains(t, contentStr, `dir="ltr"`)
		}
		
		// Validate canonical URL
		expectedCanonical := "https://example.com/"
		if lang != "fa" {
			expectedCanonical = "https://example.com/" + lang + "/"
		}
		assert.Contains(t, contentStr, expectedCanonical)
	}
}

func TestStaticGenerator_FileServing_DirectoryStructure(t *testing.T) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "static_generator_structure_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	templatesDir := filepath.Join(tempDir, "templates")
	outputDir := filepath.Join(tempDir, "output")

	// Create template directory
	require.NoError(t, os.MkdirAll(templatesDir, 0755))
	require.NoError(t, os.MkdirAll(outputDir, 0755))

	// Create test templates
	createTestTemplatesForFileServing(t, templatesDir)

	// Create static generator
	config := StaticGeneratorConfig{
		OutputPath:   outputDir,
		TemplatesDir: templatesDir,
		BaseURL:      "https://example.com",
	}

	cacheService := &TestCacheService{}
	sg, err := NewStaticGenerator(config, cacheService, nil, nil, nil, nil)
	require.NoError(t, err)

	ctx := context.Background()

	// Generate multiple articles with different slugs
	articles := []struct {
		slug     string
		title    string
		language string
	}{
		{"article-one", "Article One", "en"},
		{"article-two", "Article Two", "fa"},
		{"article-three", "Article Three", "ar"},
		{"very-long-article-slug-with-many-words", "Very Long Article", "en"},
		{"article-with-numbers-123", "Article with Numbers", "fa"},
	}

	for _, articleData := range articles {
		publishedAt := time.Now()
		article := &models.Article{
			ID:           uint64(len(articles)),
			Title:        articleData.title,
			Slug:         articleData.slug,
			Content:      "<p>Test content for " + articleData.title + "</p>",
			Excerpt:      "Test excerpt for " + articleData.title,
			CategoryID:   1,
			PublishedAt:  &publishedAt,
			UpdatedAt:    time.Now(),
			LanguageCode: articleData.language,
			SEOData: models.SEOData{
				SchemaType: "NewsArticle",
			},
		}

		err = sg.GenerateArticlePage(ctx, article)
		require.NoError(t, err)

		// Verify directory structure
		articleDir := filepath.Join(outputDir, "articles", articleData.slug)
		assert.DirExists(t, articleDir)
		assert.FileExists(t, filepath.Join(articleDir, "index.html"))
	}

	// Verify overall directory structure
	assert.DirExists(t, filepath.Join(outputDir, "articles"))
	
	// Count generated articles
	articleDirs := 0
	err = filepath.WalkDir(filepath.Join(outputDir, "articles"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && path != filepath.Join(outputDir, "articles") {
			articleDirs++
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, len(articles), articleDirs)
}

func TestStaticGenerator_FileServing_CategoryAndTagPages(t *testing.T) {
	// Create temporary directories
	tempDir, err := os.MkdirTemp("", "static_generator_category_tag_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	templatesDir := filepath.Join(tempDir, "templates")
	outputDir := filepath.Join(tempDir, "output")

	// Create template directory
	require.NoError(t, os.MkdirAll(templatesDir, 0755))
	require.NoError(t, os.MkdirAll(outputDir, 0755))

	// Create test templates
	createTestTemplatesForFileServing(t, templatesDir)

	// Create static generator
	config := StaticGeneratorConfig{
		OutputPath:   outputDir,
		TemplatesDir: templatesDir,
		BaseURL:      "https://example.com",
	}

	cacheService := &TestCacheService{}
	sg, err := NewStaticGenerator(config, cacheService, nil, nil, nil, nil)
	require.NoError(t, err)

	ctx := context.Background()

	// Test category page generation
	category := &models.Category{
		ID:          1,
		Name:        "Technology",
		Slug:        "technology",
		Description: "Technology news and updates",
	}

	err = sg.GenerateCategoryPage(ctx, category, 1)
	require.NoError(t, err)

	// Verify category file structure
	categoryDir := filepath.Join(outputDir, "categories", "technology")
	assert.DirExists(t, categoryDir)
	assert.FileExists(t, filepath.Join(categoryDir, "index.html"))

	// Test pagination
	err = sg.GenerateCategoryPage(ctx, category, 2)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(categoryDir, "page-2.html"))

	// Test tag page generation
	tag := &models.Tag{
		ID:          1,
		Name:        "AI",
		Slug:        "ai",
		Description: "Artificial Intelligence news",
		Keywords:    []string{"artificial", "intelligence", "machine learning"},
		Color:       "#ff6b6b",
	}

	err = sg.GenerateTagPage(ctx, tag, 1)
	require.NoError(t, err)

	// Verify tag file structure
	tagDir := filepath.Join(outputDir, "tags", "ai")
	assert.DirExists(t, tagDir)
	assert.FileExists(t, filepath.Join(tagDir, "index.html"))

	// Test tag pagination
	err = sg.GenerateTagPage(ctx, tag, 2)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(tagDir, "page-2.html"))

	// Validate tag page content
	tagContent, err := os.ReadFile(filepath.Join(tagDir, "index.html"))
	require.NoError(t, err)
	
	tagContentStr := string(tagContent)
	assert.Contains(t, tagContentStr, "AI")
	assert.Contains(t, tagContentStr, "Artificial Intelligence news")
	assert.Contains(t, tagContentStr, "artificial")
	assert.Contains(t, tagContentStr, "intelligence")
	assert.Contains(t, tagContentStr, "machine learning")
}

func createTestTemplatesForFileServing(t *testing.T, templatesDir string) {
	// Create comprehensive homepage template
	homepageTemplate := `<!DOCTYPE html>
<html lang="{{.Language}}" dir="{{.Direction}}">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <meta name="description" content="{{.Description}}">
    <meta name="keywords" content="{{join .Keywords ", "}}">
    <link rel="canonical" href="{{.CanonicalURL}}">
    <script type="application/ld+json">{{.SchemaMarkup | safeHTML}}</script>
</head>
<body>
    <header>
        <h1>{{.Title}}</h1>
    </header>
    <main>
        {{if .LatestArticles}}
        <section class="latest-articles">
            <h2>Latest Articles</h2>
            {{range .LatestArticles}}
            <article>
                <h3>{{.Title}}</h3>
                <p>{{.Excerpt}}</p>
                <time>{{formatDate .PublishedAt}}</time>
            </article>
            {{end}}
        </section>
        {{end}}
        {{if .TrendingArticles}}
        <section class="trending-articles">
            <h2>Trending Articles</h2>
            {{range .TrendingArticles}}
            <article>
                <h3>{{.Title}}</h3>
                <p>{{.Excerpt}}</p>
                <span>{{.ViewCount}} views</span>
            </article>
            {{end}}
        </section>
        {{end}}
    </main>
</body>
</html>`

	err := os.WriteFile(filepath.Join(templatesDir, "homepage.html"), []byte(homepageTemplate), 0644)
	require.NoError(t, err)

	// Create comprehensive article template
	articleTemplate := `<!DOCTYPE html>
<html lang="{{.Language}}" dir="{{.Direction}}">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <meta name="description" content="{{.Description}}">
    <meta name="keywords" content="{{join .Keywords ", "}}">
    <link rel="canonical" href="{{.CanonicalURL}}">
    <script type="application/ld+json">{{.SchemaMarkup | safeHTML}}</script>
</head>
<body>
    <article>
        <header>
            <h1>{{.Article.Title}}</h1>
            <div class="meta">
                <time datetime="{{.Article.PublishedAt}}">{{formatDateTime .Article.PublishedAt}}</time>
                <span>Author: {{.Article.AuthorID}}</span>
                <span>{{.Article.ViewCount}} views</span>
                <span>{{.Article.LikeCount}} likes</span>
            </div>
            {{if .Article.Tags}}
            <div class="tags">
                {{range .Article.Tags}}
                <span class="tag" style="background-color: {{.Color}}">{{.Name}}</span>
                {{end}}
            </div>
            {{end}}
            {{if .Article.Excerpt}}
            <div class="excerpt">{{.Article.Excerpt}}</div>
            {{end}}
        </header>
        <div class="content">{{.Article.Content | safeHTML}}</div>
    </article>
    {{if .RelatedArticles}}
    <section class="related">
        <h2>Related Articles</h2>
        {{range .RelatedArticles}}
        <article>
            <h3>{{.Title}}</h3>
            <p>{{.Excerpt}}</p>
        </article>
        {{end}}
    </section>
    {{end}}
</body>
</html>`

	err = os.WriteFile(filepath.Join(templatesDir, "article.html"), []byte(articleTemplate), 0644)
	require.NoError(t, err)

	// Create comprehensive category template
	categoryTemplate := `<!DOCTYPE html>
<html lang="{{.Language}}" dir="{{.Direction}}">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <meta name="description" content="{{.Description}}">
    <link rel="canonical" href="{{.CanonicalURL}}">
    <script type="application/ld+json">{{.SchemaMarkup | safeHTML}}</script>
</head>
<body>
    <header>
        <h1>{{.Category.Name}}</h1>
        {{if .Category.Description}}<p>{{.Category.Description}}</p>{{end}}
    </header>
    <main>
        {{if .Articles}}
        <section class="articles">
            {{range .Articles}}
            <article>
                <h2>{{.Title}}</h2>
                <p>{{.Excerpt}}</p>
                <time>{{formatDate .PublishedAt}}</time>
            </article>
            {{end}}
        </section>
        {{end}}
        {{if gt .Pagination.TotalPages 1}}
        <nav class="pagination">
            <span>Page {{.Pagination.CurrentPage}} of {{.Pagination.TotalPages}}</span>
            {{if .Pagination.HasPrevious}}<a href="page-{{.Pagination.PreviousPage}}.html">Previous</a>{{end}}
            {{if .Pagination.HasNext}}<a href="page-{{.Pagination.NextPage}}.html">Next</a>{{end}}
        </nav>
        {{end}}
    </main>
</body>
</html>`

	err = os.WriteFile(filepath.Join(templatesDir, "category.html"), []byte(categoryTemplate), 0644)
	require.NoError(t, err)

	// Create comprehensive tag template
	tagTemplate := `<!DOCTYPE html>
<html lang="{{.Language}}" dir="{{.Direction}}">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <meta name="description" content="{{.Description}}">
    <link rel="canonical" href="{{.CanonicalURL}}">
    <script type="application/ld+json">{{.SchemaMarkup | safeHTML}}</script>
</head>
<body>
    <header>
        <h1 style="color: {{.Tag.Color}}">{{.Tag.Name}}</h1>
        {{if .Tag.Description}}<p>{{.Tag.Description}}</p>{{end}}
        {{if .Tag.Keywords}}
        <div class="keywords">
            <strong>Keywords:</strong>
            {{range .Tag.Keywords}}<span class="keyword">{{.}}</span> {{end}}
        </div>
        {{end}}
    </header>
    <main>
        {{if .Articles}}
        <section class="articles">
            {{range .Articles}}
            <article>
                <h2>{{.Title}}</h2>
                <p>{{.Excerpt}}</p>
                <time>{{formatDate .PublishedAt}}</time>
            </article>
            {{end}}
        </section>
        {{end}}
        {{if gt .Pagination.TotalPages 1}}
        <nav class="pagination">
            <span>Page {{.Pagination.CurrentPage}} of {{.Pagination.TotalPages}}</span>
            {{if .Pagination.HasPrevious}}<a href="page-{{.Pagination.PreviousPage}}.html">Previous</a>{{end}}
            {{if .Pagination.HasNext}}<a href="page-{{.Pagination.NextPage}}.html">Next</a>{{end}}
        </nav>
        {{end}}
    </main>
</body>
</html>`

	err = os.WriteFile(filepath.Join(templatesDir, "tag.html"), []byte(tagTemplate), 0644)
	require.NoError(t, err)
}
