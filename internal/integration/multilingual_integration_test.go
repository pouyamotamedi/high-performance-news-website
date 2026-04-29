package integration

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// MultilingualIntegrationTestSuite tests multilingual content and SEO integration
type MultilingualIntegrationTestSuite struct {
	multilingualService *services.MultilingualService
	seoService         *services.SEOService
	canonicalService   *services.CanonicalService
	articleService     *services.ArticleService
	templateService    *services.TemplateService
}

func TestMultilingualSEOIntegration(t *testing.T) {
	t.Run("Multilingual SEO metadata generation", func(t *testing.T) {
		multilingualService := &MockMultilingualService{
			translations: map[string]map[string]*models.Article{
				"en": {
					"tech-news": {ID: 1, Title: "Tech News", Content: "English tech content", Language: "en", URL: "/en/article/tech-news"},
				},
				"fa": {
					"tech-news": {ID: 2, Title: "اخبار فناوری", Content: "محتوای فناوری فارسی", Language: "fa", URL: "/fa/article/tech-news"},
				},
				"ar": {
					"tech-news": {ID: 3, Title: "أخبار التكنولوجيا", Content: "محتوى التكنولوجيا العربي", Language: "ar", URL: "/ar/article/tech-news"},
				},
			},
		}

		seoService := &MockSEOServiceMultilingual{
			multilingualService: multilingualService,
		}

		ctx := context.Background()

		// Generate SEO metadata for each language
		languages := []string{"en", "fa", "ar"}
		
		for _, lang := range languages {
			article := multilingualService.translations[lang]["tech-news"]
			metadata, err := seoService.GenerateMultilingualSEO(ctx, article)
			require.NoError(t, err)

			// Verify language-specific metadata
			assert.Equal(t, lang, metadata.Language)
			assert.Contains(t, metadata.MetaTags, fmt.Sprintf(`lang="%s"`, lang))
			
			// Verify hreflang tags for all languages
			for _, targetLang := range languages {
				expectedHreflang := fmt.Sprintf(`<link rel="alternate" hreflang="%s" href="/%s/article/tech-news">`, targetLang, targetLang)
				assert.Contains(t, metadata.MetaTags, expectedHreflang)
			}

			// Verify canonical URL points to primary language (English)
			assert.Equal(t, "/en/article/tech-news", metadata.CanonicalURL)

			t.Logf("SEO metadata for %s: canonical=%s, hreflang_count=%d", 
				lang, metadata.CanonicalURL, strings.Count(metadata.MetaTags, "hreflang"))
		}
	})

	t.Run("Multilingual schema markup consistency", func(t *testing.T) {
		multilingualService := &MockMultilingualService{
			translations: map[string]map[string]*models.Article{
				"en": {
					"news": {ID: 1, Title: "Breaking News", Content: "English news content", Language: "en", URL: "/en/article/news"},
				},
				"fa": {
					"news": {ID: 2, Title: "اخبار فوری", Content: "محتوای خبری فارسی", Language: "fa", URL: "/fa/article/news"},
				},
			},
		}

		seoService := &MockSEOServiceMultilingual{
			multilingualService: multilingualService,
		}

		ctx := context.Background()

		// Generate schema markup for both languages
		enArticle := multilingualService.translations["en"]["news"]
		faArticle := multilingualService.translations["fa"]["news"]

		enSchema, err := seoService.GenerateSchemaMarkup(ctx, enArticle)
		require.NoError(t, err)

		faSchema, err := seoService.GenerateSchemaMarkup(ctx, faArticle)
		require.NoError(t, err)

		// Verify schema consistency
		assert.Contains(t, enSchema, `"@type": "NewsArticle"`)
		assert.Contains(t, faSchema, `"@type": "NewsArticle"`)

		// Verify language-specific properties
		assert.Contains(t, enSchema, `"inLanguage": "en"`)
		assert.Contains(t, faSchema, `"inLanguage": "fa"`)

		// Verify canonical URL consistency
		assert.Contains(t, enSchema, `"url": "/en/article/news"`)
		assert.Contains(t, faSchema, `"mainEntityOfPage": "/en/article/news"`) // Points to canonical

		// Verify content in correct language
		assert.Contains(t, enSchema, "Breaking News")
		assert.Contains(t, faSchema, "اخبار فوری")

		t.Logf("Schema markup generated for English and Persian with proper language attributes")
	})

	t.Run("Multilingual sitemap generation", func(t *testing.T) {
		multilingualService := &MockMultilingualService{
			translations: map[string]map[string]*models.Article{
				"en": {
					"article1": {ID: 1, Title: "Article 1", Language: "en", URL: "/en/article/article1"},
					"article2": {ID: 2, Title: "Article 2", Language: "en", URL: "/en/article/article2"},
				},
				"fa": {
					"article1": {ID: 3, Title: "مقاله ۱", Language: "fa", URL: "/fa/article/article1"},
					"article2": {ID: 4, Title: "مقاله ۲", Language: "fa", URL: "/fa/article/article2"},
				},
				"ar": {
					"article1": {ID: 5, Title: "مقال ١", Language: "ar", URL: "/ar/article/article1"},
				},
			},
		}

		seoService := &MockSEOServiceMultilingual{
			multilingualService: multilingualService,
		}

		ctx := context.Background()

		// Generate multilingual sitemap
		sitemap, err := seoService.GenerateMultilingualSitemap(ctx)
		require.NoError(t, err)

		// Verify sitemap structure
		assert.Contains(t, sitemap, `<?xml version="1.0" encoding="UTF-8"?>`)
		assert.Contains(t, sitemap, `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"`)
		assert.Contains(t, sitemap, `xmlns:xhtml="http://www.w3.org/1999/xhtml">`)

		// Verify all URLs are included
		expectedURLs := []string{
			"/en/article/article1", "/en/article/article2",
			"/fa/article/article1", "/fa/article/article2",
			"/ar/article/article1",
		}

		for _, url := range expectedURLs {
			assert.Contains(t, sitemap, fmt.Sprintf("<loc>%s</loc>", url))
		}

		// Verify hreflang annotations
		assert.Contains(t, sitemap, `<xhtml:link rel="alternate" hreflang="en" href="/en/article/article1"/>`)
		assert.Contains(t, sitemap, `<xhtml:link rel="alternate" hreflang="fa" href="/fa/article/article1"/>`)
		assert.Contains(t, sitemap, `<xhtml:link rel="alternate" hreflang="ar" href="/ar/article/article1"/>`)

		t.Logf("Multilingual sitemap generated with %d URLs and hreflang annotations", len(expectedURLs))
	})

	t.Run("RTL content handling in SEO", func(t *testing.T) {
		multilingualService := &MockMultilingualService{
			translations: map[string]map[string]*models.Article{
				"fa": {
					"rtl-article": {
						ID: 1, 
						Title: "مقاله راست به چپ", 
						Content: "این محتوای فارسی است که باید از راست به چپ نمایش داده شود", 
						Language: "fa", 
						URL: "/fa/article/rtl-article",
					},
				},
				"ar": {
					"rtl-article": {
						ID: 2, 
						Title: "مقال من اليمين إلى اليسار", 
						Content: "هذا محتوى عربي يجب عرضه من اليمين إلى اليسار", 
						Language: "ar", 
						URL: "/ar/article/rtl-article",
					},
				},
			},
		}

		seoService := &MockSEOServiceMultilingual{
			multilingualService: multilingualService,
		}

		ctx := context.Background()

		// Test Persian RTL handling
		faArticle := multilingualService.translations["fa"]["rtl-article"]
		faMetadata, err := seoService.GenerateMultilingualSEO(ctx, faArticle)
		require.NoError(t, err)

		assert.Contains(t, faMetadata.MetaTags, `dir="rtl"`)
		assert.Contains(t, faMetadata.MetaTags, `lang="fa"`)

		// Test Arabic RTL handling
		arArticle := multilingualService.translations["ar"]["rtl-article"]
		arMetadata, err := seoService.GenerateMultilingualSEO(ctx, arArticle)
		require.NoError(t, err)

		assert.Contains(t, arMetadata.MetaTags, `dir="rtl"`)
		assert.Contains(t, arMetadata.MetaTags, `lang="ar"`)

		// Verify RTL-specific meta tags
		assert.Contains(t, faMetadata.MetaTags, `<meta name="text-direction" content="rtl">`)
		assert.Contains(t, arMetadata.MetaTags, `<meta name="text-direction" content="rtl">`)

		t.Logf("RTL content handling verified for Persian and Arabic")
	})
}

func TestMultilingualContentConsistency(t *testing.T) {
	t.Run("Content synchronization across languages", func(t *testing.T) {
		multilingualService := &MockMultilingualService{
			translations: map[string]map[string]*models.Article{
				"en": {
					"sync-test": {ID: 1, Title: "Original Title", Content: "Original content", Language: "en", URL: "/en/article/sync-test"},
				},
			},
		}

		ctx := context.Background()

		// Add translation
		err := multilingualService.AddTranslation(ctx, "sync-test", "fa", &models.Article{
			ID: 2, Title: "عنوان اصلی", Content: "محتوای اصلی", Language: "fa", URL: "/fa/article/sync-test",
		})
		require.NoError(t, err)

		// Update original content
		err = multilingualService.UpdateContent(ctx, "sync-test", "en", &models.Article{
			ID: 1, Title: "Updated Title", Content: "Updated content", Language: "en", URL: "/en/article/sync-test",
		})
		require.NoError(t, err)

		// Verify translation status
		status, err := multilingualService.GetTranslationStatus(ctx, "sync-test")
		require.NoError(t, err)

		assert.Equal(t, "en", status.PrimaryLanguage)
		assert.Contains(t, status.OutdatedTranslations, "fa")
		assert.True(t, status.RequiresUpdate)

		// Update translation
		err = multilingualService.UpdateContent(ctx, "sync-test", "fa", &models.Article{
			ID: 2, Title: "عنوان به‌روزشده", Content: "محتوای به‌روزشده", Language: "fa", URL: "/fa/article/sync-test",
		})
		require.NoError(t, err)

		// Verify synchronization
		updatedStatus, err := multilingualService.GetTranslationStatus(ctx, "sync-test")
		require.NoError(t, err)

		assert.False(t, updatedStatus.RequiresUpdate)
		assert.Empty(t, updatedStatus.OutdatedTranslations)

		t.Logf("Content synchronization completed: %d languages in sync", len(updatedStatus.AvailableLanguages))
	})

	t.Run("Multilingual URL structure validation", func(t *testing.T) {
		multilingualService := &MockMultilingualService{
			translations: make(map[string]map[string]*models.Article),
		}

		ctx := context.Background()

		// Test URL structure consistency
		baseSlug := "url-structure-test"
		languages := []string{"en", "fa", "ar", "fr"}

		for i, lang := range languages {
			article := &models.Article{
				ID:       uint64(i + 1),
				Title:    fmt.Sprintf("Title in %s", lang),
				Content:  fmt.Sprintf("Content in %s", lang),
				Language: lang,
				URL:      fmt.Sprintf("/%s/article/%s", lang, baseSlug),
			}

			err := multilingualService.AddTranslation(ctx, baseSlug, lang, article)
			require.NoError(t, err)
		}

		// Validate URL structure
		for _, lang := range languages {
			article, err := multilingualService.GetTranslation(ctx, baseSlug, lang)
			require.NoError(t, err)

			expectedURL := fmt.Sprintf("/%s/article/%s", lang, baseSlug)
			assert.Equal(t, expectedURL, article.URL)

			// Verify URL follows pattern
			assert.True(t, strings.HasPrefix(article.URL, "/"+lang+"/"))
			assert.Contains(t, article.URL, baseSlug)
		}

		t.Logf("URL structure validation completed for %d languages", len(languages))
	})

	t.Run("Multilingual metadata consistency", func(t *testing.T) {
		multilingualService := &MockMultilingualService{
			translations: map[string]map[string]*models.Article{
				"en": {
					"metadata-test": {
						ID: 1, Title: "Metadata Test", Content: "English content", Language: "en", 
						URL: "/en/article/metadata-test", PublishedAt: time.Now(),
					},
				},
				"fa": {
					"metadata-test": {
						ID: 2, Title: "تست متادیتا", Content: "محتوای فارسی", Language: "fa", 
						URL: "/fa/article/metadata-test", PublishedAt: time.Now(),
					},
				},
			},
		}

		seoService := &MockSEOServiceMultilingual{
			multilingualService: multilingualService,
		}

		ctx := context.Background()

		// Generate metadata for both languages
		enArticle := multilingualService.translations["en"]["metadata-test"]
		faArticle := multilingualService.translations["fa"]["metadata-test"]

		enMetadata, err := seoService.GenerateMultilingualSEO(ctx, enArticle)
		require.NoError(t, err)

		faMetadata, err := seoService.GenerateMultilingualSEO(ctx, faArticle)
		require.NoError(t, err)

		// Verify metadata consistency
		assert.Equal(t, enMetadata.CanonicalURL, faMetadata.CanonicalURL, 
			"Both languages should have same canonical URL")

		// Verify language-specific attributes
		assert.Contains(t, enMetadata.MetaTags, `lang="en"`)
		assert.Contains(t, faMetadata.MetaTags, `lang="fa"`)

		// Verify Open Graph consistency
		assert.Contains(t, enMetadata.MetaTags, `property="og:locale" content="en_US"`)
		assert.Contains(t, faMetadata.MetaTags, `property="og:locale" content="fa_IR"`)

		// Verify alternate language tags
		assert.Contains(t, enMetadata.MetaTags, `hreflang="fa"`)
		assert.Contains(t, faMetadata.MetaTags, `hreflang="en"`)

		t.Logf("Metadata consistency verified across languages")
	})
}

func TestMultilingualPerformance(t *testing.T) {
	t.Run("Bulk multilingual SEO generation", func(t *testing.T) {
		multilingualService := &MockMultilingualService{
			translations: make(map[string]map[string]*models.Article),
		}

		// Create many multilingual articles
		const numArticles = 1000
		languages := []string{"en", "fa", "ar"}

		for i := 0; i < numArticles; i++ {
			slug := fmt.Sprintf("article-%d", i)
			
			for j, lang := range languages {
				article := &models.Article{
					ID:       uint64(i*len(languages) + j + 1),
					Title:    fmt.Sprintf("Article %d in %s", i, lang),
					Content:  fmt.Sprintf("Content %d in %s", i, lang),
					Language: lang,
					URL:      fmt.Sprintf("/%s/article/%s", lang, slug),
				}

				multilingualService.AddTranslation(context.Background(), slug, lang, article)
			}
		}

		seoService := &MockSEOServiceMultilingual{
			multilingualService: multilingualService,
		}

		ctx := context.Background()

		// Measure bulk SEO generation
		start := time.Now()
		sitemap, err := seoService.GenerateMultilingualSitemap(ctx)
		duration := time.Since(start)

		require.NoError(t, err)
		assert.NotEmpty(t, sitemap)
		assert.Less(t, duration, 5*time.Second, "Bulk generation should be reasonably fast")

		// Verify all articles are included
		totalExpectedURLs := numArticles * len(languages)
		urlCount := strings.Count(sitemap, "<loc>")
		assert.Equal(t, totalExpectedURLs, urlCount)

		t.Logf("Bulk multilingual SEO generation: %d URLs in %v", totalExpectedURLs, duration)
	})

	t.Run("Concurrent multilingual operations", func(t *testing.T) {
		multilingualService := &MockMultilingualService{
			translations: make(map[string]map[string]*models.Article),
		}

		ctx := context.Background()

		const numConcurrent = 50
		var wg sync.WaitGroup
		errors := make(chan error, numConcurrent)

		// Concurrent translation additions
		for i := 0; i < numConcurrent; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				
				slug := fmt.Sprintf("concurrent-%d", id)
				article := &models.Article{
					ID:       uint64(id + 1),
					Title:    fmt.Sprintf("Concurrent Article %d", id),
					Content:  fmt.Sprintf("Concurrent content %d", id),
					Language: "en",
					URL:      fmt.Sprintf("/en/article/%s", slug),
				}

				err := multilingualService.AddTranslation(ctx, slug, "en", article)
				errors <- err
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check results
		var errorCount int
		for err := range errors {
			if err != nil {
				errorCount++
			}
		}

		assert.Equal(t, 0, errorCount, "All concurrent operations should succeed")

		// Verify all translations were added
		totalTranslations := 0
		for _, langMap := range multilingualService.translations {
			totalTranslations += len(langMap)
		}
		assert.Equal(t, numConcurrent, totalTranslations)

		t.Logf("Concurrent multilingual operations completed: %d translations added", totalTranslations)
	})
}

// Mock implementations

type MockMultilingualService struct {
	translations map[string]map[string]*models.Article // [language][slug] -> Article
	mu           sync.RWMutex
}

func (ms *MockMultilingualService) AddTranslation(ctx context.Context, slug, language string, article *models.Article) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.translations == nil {
		ms.translations = make(map[string]map[string]*models.Article)
	}

	if ms.translations[language] == nil {
		ms.translations[language] = make(map[string]*models.Article)
	}

	ms.translations[language][slug] = article
	return nil
}

func (ms *MockMultilingualService) GetTranslation(ctx context.Context, slug, language string) (*models.Article, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if langMap, exists := ms.translations[language]; exists {
		if article, exists := langMap[slug]; exists {
			return article, nil
		}
	}

	return nil, fmt.Errorf("translation not found: %s/%s", language, slug)
}

func (ms *MockMultilingualService) UpdateContent(ctx context.Context, slug, language string, article *models.Article) error {
	return ms.AddTranslation(ctx, slug, language, article)
}

func (ms *MockMultilingualService) GetTranslationStatus(ctx context.Context, slug string) (*models.TranslationStatus, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	status := &models.TranslationStatus{
		Slug:                 slug,
		PrimaryLanguage:      "en",
		AvailableLanguages:   []string{},
		OutdatedTranslations: []string{},
		RequiresUpdate:       false,
	}

	// Simplified logic - assume English is primary and others need updates if content differs
	for lang := range ms.translations {
		if _, exists := ms.translations[lang][slug]; exists {
			status.AvailableLanguages = append(status.AvailableLanguages, lang)
			
			if lang != "en" {
				// Simplified: assume non-English translations are outdated
				status.OutdatedTranslations = append(status.OutdatedTranslations, lang)
			}
		}
	}

	status.RequiresUpdate = len(status.OutdatedTranslations) > 0

	return status, nil
}

func (ms *MockMultilingualService) GetAllTranslations(ctx context.Context) (map[string]map[string]*models.Article, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	// Return copy to avoid race conditions
	result := make(map[string]map[string]*models.Article)
	for lang, langMap := range ms.translations {
		result[lang] = make(map[string]*models.Article)
		for slug, article := range langMap {
			result[lang][slug] = article
		}
	}

	return result, nil
}

type MockSEOServiceMultilingual struct {
	multilingualService *MockMultilingualService
}

func (seo *MockSEOServiceMultilingual) GenerateMultilingualSEO(ctx context.Context, article *models.Article) (*models.SEOMetadata, error) {
	// Get all translations for this article
	slug := strings.TrimPrefix(strings.TrimPrefix(article.URL, "/"+article.Language), "/article/")
	allTranslations, err := seo.multilingualService.GetAllTranslations(ctx)
	if err != nil {
		return nil, err
	}

	metadata := &models.SEOMetadata{
		Title:        article.Title,
		Description:  fmt.Sprintf("Description for %s", article.Title),
		Language:     article.Language,
		CanonicalURL: "/en/article/" + slug, // Always point to English as canonical
	}

	// Build meta tags
	var metaTags []string
	
	// Language and direction
	metaTags = append(metaTags, fmt.Sprintf(`<html lang="%s"`, article.Language))
	if article.Language == "fa" || article.Language == "ar" {
		metaTags = append(metaTags, `dir="rtl"`)
		metaTags = append(metaTags, `<meta name="text-direction" content="rtl">`)
	}
	metaTags = append(metaTags, `>`)

	// Canonical link
	metaTags = append(metaTags, fmt.Sprintf(`<link rel="canonical" href="%s">`, metadata.CanonicalURL))

	// Hreflang tags
	for lang := range allTranslations {
		if _, exists := allTranslations[lang][slug]; exists {
			hrefLangURL := fmt.Sprintf("/%s/article/%s", lang, slug)
			metaTags = append(metaTags, fmt.Sprintf(`<link rel="alternate" hreflang="%s" href="%s">`, lang, hrefLangURL))
		}
	}

	// Open Graph locale
	localeMap := map[string]string{
		"en": "en_US",
		"fa": "fa_IR",
		"ar": "ar_SA",
		"fr": "fr_FR",
	}
	if locale, exists := localeMap[article.Language]; exists {
		metaTags = append(metaTags, fmt.Sprintf(`<meta property="og:locale" content="%s">`, locale))
	}

	metadata.MetaTags = strings.Join(metaTags, "\n")

	return metadata, nil
}

func (seo *MockSEOServiceMultilingual) GenerateSchemaMarkup(ctx context.Context, article *models.Article) (string, error) {
	// Determine canonical URL
	slug := strings.TrimPrefix(strings.TrimPrefix(article.URL, "/"+article.Language), "/article/")
	canonicalURL := "/en/article/" + slug

	schema := fmt.Sprintf(`{
		"@context": "https://schema.org",
		"@type": "NewsArticle",
		"headline": "%s",
		"url": "%s",
		"mainEntityOfPage": "%s",
		"inLanguage": "%s",
		"author": {
			"@type": "Organization",
			"name": "News Website"
		}
	}`, article.Title, article.URL, canonicalURL, article.Language)

	return schema, nil
}

func (seo *MockSEOServiceMultilingual) GenerateMultilingualSitemap(ctx context.Context) (string, error) {
	allTranslations, err := seo.multilingualService.GetAllTranslations(ctx)
	if err != nil {
		return "", err
	}

	sitemap := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"
        xmlns:xhtml="http://www.w3.org/1999/xhtml">
`

	// Group by slug to create hreflang annotations
	slugGroups := make(map[string]map[string]string) // [slug][language] -> URL
	
	for lang, langMap := range allTranslations {
		for slug, article := range langMap {
			if slugGroups[slug] == nil {
				slugGroups[slug] = make(map[string]string)
			}
			slugGroups[slug][lang] = article.URL
		}
	}

	// Generate sitemap entries
	for slug, langURLs := range slugGroups {
		for lang, url := range langURLs {
			sitemap += fmt.Sprintf("  <url>\n    <loc>%s</loc>\n", url)
			
			// Add hreflang annotations
			for hrefLang, hrefURL := range langURLs {
				sitemap += fmt.Sprintf(`    <xhtml:link rel="alternate" hreflang="%s" href="%s"/>`, hrefLang, hrefURL) + "\n"
			}
			
			sitemap += "  </url>\n"
		}
	}

	sitemap += "</urlset>"

	return sitemap, nil
}

// Benchmark tests

func BenchmarkMultilingualSEOGeneration(b *testing.B) {
	multilingualService := &MockMultilingualService{
		translations: map[string]map[string]*models.Article{
			"en": {
				"benchmark": {ID: 1, Title: "Benchmark Article", Language: "en", URL: "/en/article/benchmark"},
			},
			"fa": {
				"benchmark": {ID: 2, Title: "مقاله بنچمارک", Language: "fa", URL: "/fa/article/benchmark"},
			},
		},
	}

	seoService := &MockSEOServiceMultilingual{
		multilingualService: multilingualService,
	}

	ctx := context.Background()
	article := multilingualService.translations["en"]["benchmark"]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := seoService.GenerateMultilingualSEO(ctx, article)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMultilingualSitemapGeneration(b *testing.B) {
	multilingualService := &MockMultilingualService{
		translations: make(map[string]map[string]*models.Article),
	}

	// Create test data
	languages := []string{"en", "fa", "ar"}
	for i := 0; i < 100; i++ {
		slug := fmt.Sprintf("article-%d", i)
		for j, lang := range languages {
			article := &models.Article{
				ID:       uint64(i*len(languages) + j + 1),
				Title:    fmt.Sprintf("Article %d", i),
				Language: lang,
				URL:      fmt.Sprintf("/%s/article/%s", lang, slug),
			}
			multilingualService.AddTranslation(context.Background(), slug, lang, article)
		}
	}

	seoService := &MockSEOServiceMultilingual{
		multilingualService: multilingualService,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := seoService.GenerateMultilingualSitemap(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}