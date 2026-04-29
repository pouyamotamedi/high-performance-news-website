package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"high-performance-news-website/internal/models"
	testinghelpers "high-performance-news-website/internal/testing"
)

// TestSEOServiceComprehensive provides comprehensive SEO service testing
func TestSEOServiceComprehensive(t *testing.T) {
	env := testinghelpers.NewTestEnvironment(t)
	seoService := NewSEOService("https://example.com", "Test Site", "en")

	suite := testinghelpers.NewTestSuite("SEOService")

	// Test schema generation
	suite.AddTest(testinghelpers.TestCase{
		Name: "generate_article_schema_success",
		Test: func(t *testing.T, env *testinghelpers.TestEnvironment) {
			article := env.DataGen.GenerateTestArticle()
			author := env.DataGen.GenerateTestUser()
			category := env.DataGen.GenerateTestCategory()

			schema, err := seoService.GenerateArticleSchema(article, author, category)
			require.NoError(t, err)
			require.NotNil(t, schema)

			assert.Equal(t, "https://schema.org", schema.Context)
			assert.Equal(t, "NewsArticle", schema.Type)
			assert.Equal(t, article.Title, schema.Headline)
			assert.Equal(t, article.Excerpt, schema.Description)
			assert.Contains(t, schema.URL, article.Slug)
		},
	})

	suite.AddTest(testinghelpers.TestCase{
		Name: "generate_meta_tags_article",
		Test: func(t *testing.T, env *testinghelpers.TestEnvironment) {
			article := env.DataGen.GenerateTestArticle()
			
			meta, err := seoService.GenerateMetaTags("article", article)
			require.NoError(t, err)
			require.NotNil(t, meta)

			assert.NotEmpty(t, meta.Title)
			assert.NotEmpty(t, meta.Description)
			assert.Equal(t, "en", meta.Language)
			assert.Equal(t, "article", meta.OGType)
			assert.Contains(t, meta.CanonicalURL, article.Slug)
		},
	})

	suite.Run(t)
}

// TestSEOServiceEdgeCases tests edge cases and error conditions
func TestSEOServiceEdgeCases(t *testing.T) {
	env := testinghelpers.NewTestEnvironment(t)
	seoService := NewSEOService("https://example.com", "Test Site", "en")

	t.Run("nil_article_error", func(t *testing.T) {
		_, err := seoService.GenerateArticleSchema(nil, nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "article cannot be nil")
	})

	t.Run("invalid_page_type_error", func(t *testing.T) {
		_, err := seoService.GenerateMetaTags("invalid_type", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported page type")
	})

	t.Run("empty_title_handling", func(t *testing.T) {
		article := env.DataGen.GenerateTestArticle()
		article.Title = ""
		
		meta, err := seoService.GenerateMetaTags("article", article)
		require.NoError(t, err)
		
		// Should have fallback title
		assert.NotEmpty(t, meta.Title)
		assert.Contains(t, meta.Title, "Test Site")
	})

	t.Run("long_title_truncation", func(t *testing.T) {
		article := env.DataGen.GenerateTestArticle()
		article.Title = "This is a very long title that exceeds the optimal SEO length and should be truncated appropriately"
		
		meta, err := seoService.GenerateMetaTags("article", article)
		require.NoError(t, err)
		
		// Title should be optimized for SEO
		assert.LessOrEqual(t, len(meta.Title), 60)
	})

	t.Run("multilingual_support", func(t *testing.T) {
		languages := []string{"en", "fa", "ar"}
		
		for _, lang := range languages {
			article := env.DataGen.GenerateTestArticle()
			article.LanguageCode = lang
			
			meta, err := seoService.GenerateMetaTags("article", article)
			require.NoError(t, err)
			assert.Equal(t, lang, meta.Language)
		}
	})
}

// TestSEOServicePerformance tests SEO service performance
func TestSEOServicePerformance(t *testing.T) {
	env := testinghelpers.NewTestEnvironment(t)
	seoService := NewSEOService("https://example.com", "Test Site", "en")

	t.Run("concurrent_schema_generation", func(t *testing.T) {
		const numGoroutines = 50
		done := make(chan bool, numGoroutines)
		errors := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer func() { done <- true }()
				
				article := env.DataGen.GenerateTestArticle()
				author := env.DataGen.GenerateTestUser()
				category := env.DataGen.GenerateTestCategory()
				
				_, err := seoService.GenerateArticleSchema(article, author, category)
				if err != nil {
					errors <- err
				}
			}()
		}

		// Wait for all goroutines
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
		close(errors)

		// Check for errors
		for err := range errors {
			t.Errorf("Concurrent schema generation failed: %v", err)
		}
	})

	t.Run("bulk_meta_tag_generation", func(t *testing.T) {
		articles := make([]*models.Article, 100)
		for i := range articles {
			articles[i] = env.DataGen.GenerateTestArticle()
		}

		start := time.Now()
		for _, article := range articles {
			_, err := seoService.GenerateMetaTags("article", article)
			require.NoError(t, err)
		}
		duration := time.Since(start)

		assert.Less(t, duration, 1*time.Second, "Bulk meta tag generation should complete within 1 second")
	})
}

// BenchmarkSEOService benchmarks SEO service operations
func BenchmarkSEOService(b *testing.B) {
	env := testinghelpers.NewTestEnvironment(&testing.T{})
	seoService := NewSEOService("https://example.com", "Test Site", "en")

	article := env.DataGen.GenerateTestArticle()
	author := env.DataGen.GenerateTestUser()
	category := env.DataGen.GenerateTestCategory()

	b.Run("GenerateArticleSchema", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := seoService.GenerateArticleSchema(article, author, category)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("GenerateMetaTags", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := seoService.GenerateMetaTags("article", article)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("RenderSchemaJSON", func(b *testing.B) {
		schema := &SchemaMarkup{
			Context:     "https://schema.org",
			Type:        "Article",
			Headline:    article.Title,
			Description: article.Excerpt,
			URL:         "https://example.com/test",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := seoService.RenderSchemaJSON(schema)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// TestSEOServiceIntegration tests integration with other services
func TestSEOServiceIntegration(t *testing.T) {
	env := testinghelpers.NewTestEnvironment(t)
	seoService := NewSEOService("https://example.com", "Test Site", "en")

	t.Run("integration_with_cache", func(t *testing.T) {
		article := env.DataGen.GenerateTestArticle()
		
		// Generate meta tags multiple times
		meta1, err := seoService.GenerateMetaTags("article", article)
		require.NoError(t, err)
		
		meta2, err := seoService.GenerateMetaTags("article", article)
		require.NoError(t, err)
		
		// Results should be consistent
		assert.Equal(t, meta1.Title, meta2.Title)
		assert.Equal(t, meta1.Description, meta2.Description)
		assert.Equal(t, meta1.CanonicalURL, meta2.CanonicalURL)
	})

	t.Run("schema_consistency_across_languages", func(t *testing.T) {
		article := env.DataGen.GenerateTestArticle()
		author := env.DataGen.GenerateTestUser()
		category := env.DataGen.GenerateTestCategory()

		// Test different language codes
		languages := []string{"en", "fa", "ar"}
		schemas := make([]*SchemaMarkup, len(languages))

		for i, lang := range languages {
			article.LanguageCode = lang
			schema, err := seoService.GenerateArticleSchema(article, author, category)
			require.NoError(t, err)
			schemas[i] = schema
		}

		// Core schema properties should be consistent
		for i := 1; i < len(schemas); i++ {
			assert.Equal(t, schemas[0].Context, schemas[i].Context)
			assert.Equal(t, schemas[0].Type, schemas[i].Type)
			assert.Equal(t, schemas[0].Headline, schemas[i].Headline)
		}
	})
}