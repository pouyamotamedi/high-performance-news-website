package integration

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// CanonicalIntegrationTestSuite tests canonical URL interactions with other systems
type CanonicalIntegrationTestSuite struct {
	canonicalService *services.CanonicalService
	seoService      *services.SEOService
	cacheService    *services.CacheService
	articleService  *services.ArticleService
}

func TestCanonicalURLChainValidation(t *testing.T) {
	t.Run("Canonical chain detection and resolution", func(t *testing.T) {
		canonicalService := &MockCanonicalServiceAdvanced{
			mappings: map[string]string{
				"/article/step1": "/article/step2",
				"/article/step2": "/article/step3", 
				"/article/step3": "/article/final",
				"/article/final": "/article/final",
			},
		}

		ctx := context.Background()

		// Test chain resolution
		finalURL, chainLength, err := canonicalService.ResolveCanonicalChain(ctx, "/article/step1")
		require.NoError(t, err)
		
		assert.Equal(t, "/article/final", finalURL)
		assert.Equal(t, 3, chainLength)

		// Test direct canonical
		finalURL, chainLength, err = canonicalService.ResolveCanonicalChain(ctx, "/article/final")
		require.NoError(t, err)
		
		assert.Equal(t, "/article/final", finalURL)
		assert.Equal(t, 0, chainLength)

		t.Logf("Chain resolution: /article/step1 -> %s (length: %d)", finalURL, chainLength)
	})

	t.Run("Canonical cycle detection and prevention", func(t *testing.T) {
		canonicalService := &MockCanonicalServiceAdvanced{
			mappings: map[string]string{
				"/article/cycle1": "/article/cycle2",
				"/article/cycle2": "/article/cycle3",
				"/article/cycle3": "/article/cycle1", // Creates cycle
			},
		}

		ctx := context.Background()

		// Should detect cycle and return error or original URL
		finalURL, chainLength, err := canonicalService.ResolveCanonicalChain(ctx, "/article/cycle1")
		
		if err != nil {
			assert.Contains(t, err.Error(), "cycle detected")
		} else {
			// If no error, should return original URL
			assert.Equal(t, "/article/cycle1", finalURL)
			assert.Equal(t, -1, chainLength) // Indicates cycle
		}

		t.Logf("Cycle detection result: %s, length: %d, error: %v", finalURL, chainLength, err)
	})

	t.Run("Canonical chain length optimization", func(t *testing.T) {
		canonicalService := &MockCanonicalServiceAdvanced{
			mappings: map[string]string{
				"/article/long1": "/article/long2",
				"/article/long2": "/article/long3",
				"/article/long3": "/article/long4",
				"/article/long4": "/article/long5",
				"/article/long5": "/article/final",
				"/article/final": "/article/final",
			},
		}

		ctx := context.Background()

		// Test long chain
		finalURL, chainLength, err := canonicalService.ResolveCanonicalChain(ctx, "/article/long1")
		require.NoError(t, err)
		
		assert.Equal(t, "/article/final", finalURL)
		assert.Equal(t, 5, chainLength)

		// Optimize chain by updating intermediate mappings
		err = canonicalService.OptimizeCanonicalChain(ctx, "/article/long1")
		require.NoError(t, err)

		// After optimization, all intermediate URLs should point directly to final
		for _, url := range []string{"/article/long1", "/article/long2", "/article/long3", "/article/long4"} {
			canonical, _, err := canonicalService.ResolveCanonicalChain(ctx, url)
			require.NoError(t, err)
			assert.Equal(t, "/article/final", canonical)
		}

		t.Logf("Chain optimization completed for %d URLs", chainLength)
	})

	t.Run("Multilingual canonical URL handling", func(t *testing.T) {
		canonicalService := &MockCanonicalServiceAdvanced{
			mappings: map[string]string{
				"/en/article/news":    "/en/article/news",    // English is canonical
				"/fa/article/news":    "/en/article/news",    // Persian points to English
				"/ar/article/news":    "/en/article/news",    // Arabic points to English
				"/fr/article/news":    "/en/article/news",    // French points to English
			},
		}

		ctx := context.Background()

		languages := []string{"en", "fa", "ar", "fr"}
		
		for _, lang := range languages {
			url := fmt.Sprintf("/%s/article/news", lang)
			canonical, _, err := canonicalService.ResolveCanonicalChain(ctx, url)
			require.NoError(t, err)
			
			assert.Equal(t, "/en/article/news", canonical, 
				"All language variants should point to English canonical")
		}

		// Test language-specific canonical resolution
		langCanonical, err := canonicalService.GetLanguageCanonical(ctx, "/fa/article/news", "fa")
		require.NoError(t, err)
		assert.Equal(t, "/fa/article/news", langCanonical, 
			"Language-specific canonical should return same-language URL when available")

		t.Logf("Multilingual canonical resolution completed for %d languages", len(languages))
	})
}

func TestCanonicalSEOIntegration(t *testing.T) {
	t.Run("SEO metadata consistency with canonical URLs", func(t *testing.T) {
		canonicalService := &MockCanonicalServiceAdvanced{
			mappings: map[string]string{
				"/article/duplicate": "/article/original",
				"/article/original": "/article/original",
			},
		}

		seoService := &MockSEOService{
			canonicalService: canonicalService,
		}

		ctx := context.Background()

		// Generate SEO metadata for duplicate URL
		metadata, err := seoService.GenerateMetadataWithCanonical(ctx, "/article/duplicate", &models.Article{
			ID:    1,
			Title: "Test Article",
			Content: "Test content",
		})
		require.NoError(t, err)

		// Canonical URL in metadata should point to original
		assert.Equal(t, "/article/original", metadata.CanonicalURL)
		assert.Contains(t, metadata.MetaTags, `<link rel="canonical" href="/article/original">`)

		// Generate metadata for original URL
		originalMetadata, err := seoService.GenerateMetadataWithCanonical(ctx, "/article/original", &models.Article{
			ID:    1,
			Title: "Test Article",
			Content: "Test content",
		})
		require.NoError(t, err)

		// Should be self-referencing
		assert.Equal(t, "/article/original", originalMetadata.CanonicalURL)

		t.Logf("SEO canonical metadata: duplicate=%s, original=%s", 
			metadata.CanonicalURL, originalMetadata.CanonicalURL)
	})

	t.Run("Sitemap generation with canonical URLs", func(t *testing.T) {
		canonicalService := &MockCanonicalServiceAdvanced{
			mappings: map[string]string{
				"/article/dup1": "/article/canonical1",
				"/article/dup2": "/article/canonical1",
				"/article/dup3": "/article/canonical2",
				"/article/canonical1": "/article/canonical1",
				"/article/canonical2": "/article/canonical2",
			},
		}

		seoService := &MockSEOService{
			canonicalService: canonicalService,
		}

		ctx := context.Background()

		allURLs := []string{
			"/article/dup1", "/article/dup2", "/article/dup3",
			"/article/canonical1", "/article/canonical2",
		}

		sitemap, err := seoService.GenerateSitemapWithCanonical(ctx, allURLs)
		require.NoError(t, err)

		// Sitemap should only contain canonical URLs
		assert.Contains(t, sitemap, "/article/canonical1")
		assert.Contains(t, sitemap, "/article/canonical2")
		
		// Should not contain duplicate URLs
		assert.NotContains(t, sitemap, "/article/dup1")
		assert.NotContains(t, sitemap, "/article/dup2")
		assert.NotContains(t, sitemap, "/article/dup3")

		// Count canonical URLs in sitemap
		canonicalCount := 0
		if contains(sitemap, "/article/canonical1") {
			canonicalCount++
		}
		if contains(sitemap, "/article/canonical2") {
			canonicalCount++
		}

		assert.Equal(t, 2, canonicalCount, "Sitemap should contain exactly 2 canonical URLs")

		t.Logf("Sitemap generated with %d canonical URLs", canonicalCount)
	})

	t.Run("Schema markup with canonical URLs", func(t *testing.T) {
		canonicalService := &MockCanonicalServiceAdvanced{
			mappings: map[string]string{
				"/article/schema-dup": "/article/schema-original",
				"/article/schema-original": "/article/schema-original",
			},
		}

		seoService := &MockSEOService{
			canonicalService: canonicalService,
		}

		ctx := context.Background()

		article := &models.Article{
			ID:      1,
			Title:   "Schema Test Article",
			Content: "Test content for schema",
			URL:     "/article/schema-dup",
		}

		schema, err := seoService.GenerateSchemaMarkupWithCanonical(ctx, article)
		require.NoError(t, err)

		// Schema should use canonical URL
		assert.Contains(t, schema, `"url": "/article/schema-original"`)
		assert.Contains(t, schema, `"mainEntityOfPage": "/article/schema-original"`)
		
		// Should not contain duplicate URL
		assert.NotContains(t, schema, `"url": "/article/schema-dup"`)

		t.Logf("Schema markup with canonical URL: %s", schema)
	})
}

func TestCanonicalCacheIntegration(t *testing.T) {
	t.Run("Canonical URL cache invalidation", func(t *testing.T) {
		canonicalService := &MockCanonicalServiceAdvanced{
			mappings: map[string]string{
				"/article/cached": "/article/original",
			},
			cache: make(map[string]string),
		}

		ctx := context.Background()

		// First resolution - should cache result
		canonical1, _, err := canonicalService.ResolveCanonicalChain(ctx, "/article/cached")
		require.NoError(t, err)
		assert.Equal(t, "/article/original", canonical1)

		// Verify cache was populated
		assert.Len(t, canonicalService.cache, 1)

		// Update canonical mapping
		canonicalService.mappings["/article/cached"] = "/article/new-canonical"

		// Invalidate cache
		err = canonicalService.InvalidateCache(ctx, "/article/cached")
		require.NoError(t, err)

		// Second resolution - should use new mapping
		canonical2, _, err := canonicalService.ResolveCanonicalChain(ctx, "/article/cached")
		require.NoError(t, err)
		assert.Equal(t, "/article/new-canonical", canonical2)

		t.Logf("Cache invalidation: %s -> %s", canonical1, canonical2)
	})

	t.Run("Bulk canonical cache warming", func(t *testing.T) {
		canonicalService := &MockCanonicalServiceAdvanced{
			mappings: make(map[string]string),
			cache:    make(map[string]string),
		}

		// Create many canonical mappings
		urls := make([]string, 1000)
		for i := 0; i < 1000; i++ {
			url := fmt.Sprintf("/article/bulk-%d", i)
			canonical := fmt.Sprintf("/article/canonical-%d", i/10) // 10 URLs per canonical
			urls[i] = url
			canonicalService.mappings[url] = canonical
			canonicalService.mappings[canonical] = canonical
		}

		ctx := context.Background()

		start := time.Now()
		err := canonicalService.WarmCache(ctx, urls)
		duration := time.Since(start)

		require.NoError(t, err)
		assert.Less(t, duration, 2*time.Second, "Bulk cache warming should be fast")

		// Verify cache was populated
		assert.Greater(t, len(canonicalService.cache), 900, 
			"Most URLs should be cached")

		t.Logf("Bulk cache warming completed in %v for %d URLs", duration, len(urls))
	})

	t.Run("Canonical cache consistency under concurrent access", func(t *testing.T) {
		canonicalService := &MockCanonicalServiceAdvanced{
			mappings: map[string]string{
				"/article/concurrent": "/article/canonical",
				"/article/canonical": "/article/canonical",
			},
			cache: make(map[string]string),
		}

		ctx := context.Background()

		const numConcurrent = 100
		var wg sync.WaitGroup
		results := make(chan string, numConcurrent)

		// Concurrent canonical resolutions
		for i := 0; i < numConcurrent; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				canonical, _, err := canonicalService.ResolveCanonicalChain(ctx, "/article/concurrent")
				if err != nil {
					results <- "ERROR"
				} else {
					results <- canonical
				}
			}()
		}

		wg.Wait()
		close(results)

		// All results should be consistent
		var canonicalResults []string
		for result := range results {
			canonicalResults = append(canonicalResults, result)
		}

		assert.Len(t, canonicalResults, numConcurrent)
		for _, result := range canonicalResults {
			assert.Equal(t, "/article/canonical", result, 
				"All concurrent resolutions should return same canonical URL")
		}

		t.Logf("Concurrent canonical resolution: %d consistent results", len(canonicalResults))
	})
}

// Mock implementations

type MockCanonicalServiceAdvanced struct {
	mappings map[string]string
	cache    map[string]string
	mu       sync.RWMutex
}

func (cs *MockCanonicalServiceAdvanced) ResolveCanonicalChain(ctx context.Context, url string) (string, int, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	// Check cache first
	if cs.cache != nil {
		if cached, exists := cs.cache[url]; exists {
			return cached, 0, nil // Cached results don't track chain length
		}
	}

	visited := make(map[string]bool)
	current := url
	chainLength := 0

	for {
		if visited[current] {
			// Cycle detected
			return current, -1, fmt.Errorf("canonical cycle detected starting from %s", url)
		}

		visited[current] = true
		canonical, exists := cs.mappings[current]

		if !exists || canonical == current {
			// Cache result
			if cs.cache != nil {
				cs.cache[url] = current
			}
			return current, chainLength, nil
		}

		current = canonical
		chainLength++

		// Prevent infinite chains
		if chainLength > 10 {
			return url, chainLength, fmt.Errorf("canonical chain too long for %s", url)
		}
	}
}

func (cs *MockCanonicalServiceAdvanced) OptimizeCanonicalChain(ctx context.Context, url string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Find final canonical URL
	finalCanonical, _, err := cs.ResolveCanonicalChain(ctx, url)
	if err != nil {
		return err
	}

	// Update all intermediate URLs to point directly to final canonical
	visited := make(map[string]bool)
	current := url

	for {
		if visited[current] || current == finalCanonical {
			break
		}

		visited[current] = true
		next, exists := cs.mappings[current]
		
		if !exists {
			break
		}

		// Update to point directly to final canonical
		cs.mappings[current] = finalCanonical
		current = next
	}

	// Clear cache for optimized URLs
	if cs.cache != nil {
		for optimizedURL := range visited {
			delete(cs.cache, optimizedURL)
		}
	}

	return nil
}

func (cs *MockCanonicalServiceAdvanced) GetLanguageCanonical(ctx context.Context, url, language string) (string, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	// For language-specific canonical, return same-language URL if it exists
	// This is a simplified implementation
	if strings.Contains(url, "/"+language+"/") {
		return url, nil
	}

	// Otherwise resolve to canonical
	canonical, _, err := cs.ResolveCanonicalChain(ctx, url)
	return canonical, err
}

func (cs *MockCanonicalServiceAdvanced) InvalidateCache(ctx context.Context, url string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.cache != nil {
		delete(cs.cache, url)
	}

	return nil
}

func (cs *MockCanonicalServiceAdvanced) WarmCache(ctx context.Context, urls []string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.cache == nil {
		cs.cache = make(map[string]string)
	}

	for _, url := range urls {
		canonical, _, err := cs.ResolveCanonicalChain(ctx, url)
		if err == nil {
			cs.cache[url] = canonical
		}
	}

	return nil
}

type MockSEOService struct {
	canonicalService *MockCanonicalServiceAdvanced
}

func (seo *MockSEOService) GenerateMetadataWithCanonical(ctx context.Context, url string, article *models.Article) (*models.SEOMetadata, error) {
	canonical, _, err := seo.canonicalService.ResolveCanonicalChain(ctx, url)
	if err != nil {
		canonical = url // Fallback to original URL
	}

	metadata := &models.SEOMetadata{
		Title:        article.Title,
		Description:  "Generated description",
		CanonicalURL: canonical,
		MetaTags:     fmt.Sprintf(`<link rel="canonical" href="%s">`, canonical),
	}

	return metadata, nil
}

func (seo *MockSEOService) GenerateSitemapWithCanonical(ctx context.Context, urls []string) (string, error) {
	canonicalURLs := make(map[string]bool)

	// Resolve all URLs to their canonical versions
	for _, url := range urls {
		canonical, _, err := seo.canonicalService.ResolveCanonicalChain(ctx, url)
		if err == nil {
			canonicalURLs[canonical] = true
		}
	}

	// Generate sitemap with only canonical URLs
	sitemap := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<urlset>\n"
	for canonical := range canonicalURLs {
		sitemap += fmt.Sprintf("  <url><loc>%s</loc></url>\n", canonical)
	}
	sitemap += "</urlset>"

	return sitemap, nil
}

func (seo *MockSEOService) GenerateSchemaMarkupWithCanonical(ctx context.Context, article *models.Article) (string, error) {
	canonical, _, err := seo.canonicalService.ResolveCanonicalChain(ctx, article.URL)
	if err != nil {
		canonical = article.URL
	}

	schema := fmt.Sprintf(`{
		"@context": "https://schema.org",
		"@type": "Article",
		"headline": "%s",
		"url": "%s",
		"mainEntityOfPage": "%s"
	}`, article.Title, canonical, canonical)

	return schema, nil
}

// Helper functions

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// Benchmark tests

func BenchmarkCanonicalChainResolution(b *testing.B) {
	canonicalService := &MockCanonicalServiceAdvanced{
		mappings: map[string]string{
			"/article/bench1": "/article/bench2",
			"/article/bench2": "/article/bench3",
			"/article/bench3": "/article/final",
			"/article/final": "/article/final",
		},
		cache: make(map[string]string),
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := canonicalService.ResolveCanonicalChain(ctx, "/article/bench1")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConcurrentCanonicalResolution(b *testing.B) {
	canonicalService := &MockCanonicalServiceAdvanced{
		mappings: map[string]string{
			"/article/concurrent": "/article/canonical",
			"/article/canonical": "/article/canonical",
		},
		cache: make(map[string]string),
	}

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, err := canonicalService.ResolveCanonicalChain(ctx, "/article/concurrent")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}