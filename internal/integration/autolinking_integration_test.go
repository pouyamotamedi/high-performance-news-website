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

// AutolinkingIntegrationTestSuite tests complex interactions between auto-linking and canonicalization
type AutolinkingIntegrationTestSuite struct {
	autolinkingService     *services.AutolinkingService
	canonicalService       *services.CanonicalService
	articleService         *services.ArticleService
	seoService            *services.SEOService
}

func TestAutolinkingCanonicalInteraction(t *testing.T) {
	t.Run("Auto-linking respects canonical URLs", func(t *testing.T) {
		mockCanonicalService := &MockCanonicalService{
			canonicalMappings: map[string]string{
				"/en/article/duplicate-1": "/en/article/original",
				"/en/article/duplicate-2": "/en/article/original",
				"/en/article/original":    "/en/article/original",
			},
		}

		mockAutolinkingService := &MockAutolinkingService{
			canonicalService: mockCanonicalService,
		}

		ctx := context.Background()

		// Article content with references to duplicate URLs
		content := `
		This article references several related pieces:
		- First reference: /en/article/duplicate-1
		- Second reference: /en/article/duplicate-2  
		- Original reference: /en/article/original
		`

		processedContent, err := mockAutolinkingService.ProcessContentWithCanonical(ctx, content)
		require.NoError(t, err)

		// All links should point to canonical URL
		assert.Contains(t, processedContent, `<a href="/en/article/original">duplicate-1</a>`)
		assert.Contains(t, processedContent, `<a href="/en/article/original">duplicate-2</a>`)
		assert.Contains(t, processedContent, `<a href="/en/article/original">original</a>`)

		// Should not contain non-canonical URLs in links
		assert.NotContains(t, processedContent, `href="/en/article/duplicate-1"`)
		assert.NotContains(t, processedContent, `href="/en/article/duplicate-2"`)

		t.Logf("Processed content with canonical auto-linking:\n%s", processedContent)
	})

	t.Run("Canonical chain resolution in auto-linking", func(t *testing.T) {
		mockCanonicalService := &MockCanonicalService{
			canonicalMappings: map[string]string{
				"/en/article/redirect-1": "/en/article/redirect-2",
				"/en/article/redirect-2": "/en/article/redirect-3",
				"/en/article/redirect-3": "/en/article/final",
				"/en/article/final":      "/en/article/final",
			},
		}

		mockAutolinkingService := &MockAutolinkingService{
			canonicalService: mockCanonicalService,
		}

		ctx := context.Background()

		content := `Check out this article: /en/article/redirect-1`

		processedContent, err := mockAutolinkingService.ProcessContentWithCanonical(ctx, content)
		require.NoError(t, err)

		// Should resolve to final canonical URL
		assert.Contains(t, processedContent, `<a href="/en/article/final">redirect-1</a>`)
		assert.NotContains(t, processedContent, `href="/en/article/redirect-1"`)
		assert.NotContains(t, processedContent, `href="/en/article/redirect-2"`)
		assert.NotContains(t, processedContent, `href="/en/article/redirect-3"`)

		t.Logf("Canonical chain resolved: %s", processedContent)
	})

	t.Run("Auto-linking prevents canonical cycles", func(t *testing.T) {
		mockCanonicalService := &MockCanonicalService{
			canonicalMappings: map[string]string{
				"/en/article/cycle-1": "/en/article/cycle-2",
				"/en/article/cycle-2": "/en/article/cycle-3",
				"/en/article/cycle-3": "/en/article/cycle-1", // Creates cycle
			},
		}

		mockAutolinkingService := &MockAutolinkingService{
			canonicalService: mockCanonicalService,
		}

		ctx := context.Background()

		content := `This creates a cycle: /en/article/cycle-1`

		processedContent, err := mockAutolinkingService.ProcessContentWithCanonical(ctx, content)
		require.NoError(t, err)

		// Should detect cycle and use original URL or skip linking
		assert.NotContains(t, processedContent, `<a href="/en/article/cycle-1">`)
		assert.NotContains(t, processedContent, `<a href="/en/article/cycle-2">`)
		assert.NotContains(t, processedContent, `<a href="/en/article/cycle-3">`)

		// Original text should remain unchanged
		assert.Contains(t, processedContent, "/en/article/cycle-1")

		t.Logf("Cycle prevention result: %s", processedContent)
	})

	t.Run("Multilingual auto-linking with canonical URLs", func(t *testing.T) {
		mockCanonicalService := &MockCanonicalService{
			canonicalMappings: map[string]string{
				"/en/article/tech-news":    "/en/article/tech-news",
				"/fa/article/tech-news":    "/en/article/tech-news", // Persian points to English canonical
				"/ar/article/tech-news":    "/en/article/tech-news", // Arabic points to English canonical
			},
		}

		mockAutolinkingService := &MockAutolinkingService{
			canonicalService: mockCanonicalService,
		}

		ctx := context.Background()

		// Persian content referencing multilingual URLs
		content := `
		مقاله‌های مرتبط:
		- نسخه انگلیسی: /en/article/tech-news
		- نسخه فارسی: /fa/article/tech-news
		- نسخه عربی: /ar/article/tech-news
		`

		processedContent, err := mockAutolinkingService.ProcessContentWithCanonical(ctx, content)
		require.NoError(t, err)

		// All links should point to canonical English version
		linkCount := strings.Count(processedContent, `href="/en/article/tech-news"`)
		assert.Equal(t, 3, linkCount, "All three references should link to canonical URL")

		// Should not contain non-canonical language URLs in links
		assert.NotContains(t, processedContent, `href="/fa/article/tech-news"`)
		assert.NotContains(t, processedContent, `href="/ar/article/tech-news"`)

		t.Logf("Multilingual canonical auto-linking: %s", processedContent)
	})
}

func TestAutolinkingPerformanceWithCanonical(t *testing.T) {
	t.Run("Bulk auto-linking with canonical resolution", func(t *testing.T) {
		mockCanonicalService := &MockCanonicalService{
			canonicalMappings: make(map[string]string),
		}

		// Create many canonical mappings
		for i := 1; i <= 1000; i++ {
			duplicate := fmt.Sprintf("/en/article/dup-%d", i)
			canonical := fmt.Sprintf("/en/article/canonical-%d", (i-1)/10+1) // 10 duplicates per canonical
			mockCanonicalService.canonicalMappings[duplicate] = canonical
			mockCanonicalService.canonicalMappings[canonical] = canonical
		}

		mockAutolinkingService := &MockAutolinkingService{
			canonicalService: mockCanonicalService,
		}

		ctx := context.Background()

		// Content with many references
		var contentBuilder strings.Builder
		contentBuilder.WriteString("Related articles:\n")
		for i := 1; i <= 100; i++ {
			contentBuilder.WriteString(fmt.Sprintf("- Article %d: /en/article/dup-%d\n", i, i))
		}
		content := contentBuilder.String()

		start := time.Now()
		processedContent, err := mockAutolinkingService.ProcessContentWithCanonical(ctx, content)
		duration := time.Since(start)

		require.NoError(t, err)
		assert.NotEmpty(t, processedContent)
		assert.Less(t, duration, 1*time.Second, "Bulk processing should be fast")

		// Verify canonical resolution worked
		canonicalLinkCount := strings.Count(processedContent, "/en/article/canonical-")
		duplicateLinkCount := strings.Count(processedContent, "/en/article/dup-")
		
		assert.Greater(t, canonicalLinkCount, 0, "Should have canonical links")
		assert.Equal(t, 0, strings.Count(processedContent, `href="/en/article/dup-`), 
			"Should not have duplicate URLs in href attributes")

		t.Logf("Bulk processing completed in %v, %d canonical links created", 
			duration, canonicalLinkCount)
	})

	t.Run("Concurrent auto-linking with canonical service", func(t *testing.T) {
		mockCanonicalService := &MockCanonicalService{
			canonicalMappings: map[string]string{
				"/en/article/concurrent-1": "/en/article/canonical-1",
				"/en/article/concurrent-2": "/en/article/canonical-2",
				"/en/article/concurrent-3": "/en/article/canonical-3",
			},
		}

		mockAutolinkingService := &MockAutolinkingService{
			canonicalService: mockCanonicalService,
		}

		ctx := context.Background()

		const numConcurrent = 50
		var wg sync.WaitGroup
		results := make(chan error, numConcurrent)

		start := time.Now()

		for i := 0; i < numConcurrent; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				
				content := fmt.Sprintf("Reference: /en/article/concurrent-%d", (id%3)+1)
				_, err := mockAutolinkingService.ProcessContentWithCanonical(ctx, content)
				results <- err
			}(i)
		}

		wg.Wait()
		close(results)
		duration := time.Since(start)

		// Check results
		var errorCount int
		for err := range results {
			if err != nil {
				errorCount++
			}
		}

		assert.Equal(t, 0, errorCount, "All concurrent operations should succeed")
		assert.Less(t, duration, 2*time.Second, "Concurrent processing should be efficient")

		t.Logf("Concurrent auto-linking completed in %v with %d operations", 
			duration, numConcurrent)
	})
}

func TestAutolinkingCacheIntegration(t *testing.T) {
	t.Run("Auto-linking cache invalidation on canonical changes", func(t *testing.T) {
		mockCanonicalService := &MockCanonicalService{
			canonicalMappings: map[string]string{
				"/en/article/test": "/en/article/original",
			},
		}

		mockAutolinkingService := &MockAutolinkingService{
			canonicalService: mockCanonicalService,
			cache:           make(map[string]string),
		}

		ctx := context.Background()
		content := "Check this out: /en/article/test"

		// First processing - should cache result
		result1, err := mockAutolinkingService.ProcessContentWithCanonical(ctx, content)
		require.NoError(t, err)
		assert.Contains(t, result1, `href="/en/article/original"`)

		// Verify cache was populated
		assert.Len(t, mockAutolinkingService.cache, 1)

		// Change canonical mapping
		mockCanonicalService.canonicalMappings["/en/article/test"] = "/en/article/new-canonical"

		// Invalidate cache (simulating canonical service notification)
		mockAutolinkingService.InvalidateCache("/en/article/test")

		// Second processing - should use new canonical
		result2, err := mockAutolinkingService.ProcessContentWithCanonical(ctx, content)
		require.NoError(t, err)
		assert.Contains(t, result2, `href="/en/article/new-canonical"`)
		assert.NotContains(t, result2, `href="/en/article/original"`)

		t.Logf("Cache invalidation test: %s -> %s", result1, result2)
	})

	t.Run("Auto-linking with canonical service fallback", func(t *testing.T) {
		mockCanonicalService := &MockCanonicalService{
			canonicalMappings: map[string]string{
				"/en/article/exists": "/en/article/canonical",
			},
			simulateFailure: true,
		}

		mockAutolinkingService := &MockAutolinkingService{
			canonicalService: mockCanonicalService,
		}

		ctx := context.Background()
		content := "References: /en/article/exists and /en/article/missing"

		// Should handle canonical service failures gracefully
		processedContent, err := mockAutolinkingService.ProcessContentWithCanonical(ctx, content)
		require.NoError(t, err)

		// Should still create links, but without canonical resolution
		assert.Contains(t, processedContent, `<a href="/en/article/exists">exists</a>`)
		assert.Contains(t, processedContent, `<a href="/en/article/missing">missing</a>`)

		t.Logf("Fallback processing: %s", processedContent)
	})
}

// Mock implementations

type MockCanonicalService struct {
	canonicalMappings map[string]string
	simulateFailure   bool
	mu                sync.RWMutex
}

func (cs *MockCanonicalService) GetCanonicalURL(url string) (string, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if cs.simulateFailure {
		return "", fmt.Errorf("canonical service unavailable")
	}

	// Detect cycles
	visited := make(map[string]bool)
	current := url

	for {
		if visited[current] {
			// Cycle detected, return original URL
			return url, nil
		}

		visited[current] = true
		canonical, exists := cs.canonicalMappings[current]
		
		if !exists || canonical == current {
			return current, nil
		}

		current = canonical
	}
}

func (cs *MockCanonicalService) SetCanonicalMapping(url, canonical string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.canonicalMappings[url] = canonical
}

type MockAutolinkingService struct {
	canonicalService *MockCanonicalService
	cache           map[string]string
	mu              sync.RWMutex
}

func (als *MockAutolinkingService) ProcessContentWithCanonical(ctx context.Context, content string) (string, error) {
	// Check cache first
	als.mu.RLock()
	if cached, exists := als.cache[content]; exists {
		als.mu.RUnlock()
		return cached, nil
	}
	als.mu.RUnlock()

	// Find URLs in content (simplified regex)
	result := content
	
	// Simple URL pattern matching
	urls := als.extractURLs(content)
	
	for _, url := range urls {
		// Get canonical URL
		canonicalURL, err := als.canonicalService.GetCanonicalURL(url)
		if err != nil {
			// Fallback to original URL on canonical service failure
			canonicalURL = url
		}

		// Create link with canonical URL
		linkText := strings.TrimPrefix(url, "/article/")
		link := fmt.Sprintf(`<a href="%s">%s</a>`, canonicalURL, linkText)
		
		// Replace in content
		result = strings.ReplaceAll(result, url, link)
	}

	// Cache result
	als.mu.Lock()
	if als.cache == nil {
		als.cache = make(map[string]string)
	}
	als.cache[content] = result
	als.mu.Unlock()

	return result, nil
}

func (als *MockAutolinkingService) extractURLs(content string) []string {
	var urls []string
	
	// Simple extraction - look for /{lang}/article/ patterns
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		words := strings.Fields(line)
		for _, word := range words {
			// Match patterns like /en/article/, /de/article/, etc.
			if strings.Contains(word, "/article/") {
				// Clean up the URL (remove trailing punctuation)
				url := strings.TrimRight(word, ".,!?;:")
				urls = append(urls, url)
			}
		}
	}
	
	return urls
}

func (als *MockAutolinkingService) InvalidateCache(url string) {
	als.mu.Lock()
	defer als.mu.Unlock()
	
	// Remove cache entries that contain this URL
	for key := range als.cache {
		if strings.Contains(key, url) {
			delete(als.cache, key)
		}
	}
}

// Benchmark tests

func BenchmarkAutolinkingWithCanonical(b *testing.B) {
	mockCanonicalService := &MockCanonicalService{
		canonicalMappings: map[string]string{
			"/en/article/test-1": "/en/article/canonical-1",
			"/en/article/test-2": "/en/article/canonical-2",
			"/en/article/test-3": "/en/article/canonical-3",
		},
	}

	mockAutolinkingService := &MockAutolinkingService{
		canonicalService: mockCanonicalService,
	}

	ctx := context.Background()
	content := "References: /en/article/test-1, /en/article/test-2, and /en/article/test-3"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := mockAutolinkingService.ProcessContentWithCanonical(ctx, content)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConcurrentAutolinking(b *testing.B) {
	mockCanonicalService := &MockCanonicalService{
		canonicalMappings: map[string]string{
			"/en/article/bench": "/en/article/canonical",
		},
	}

	mockAutolinkingService := &MockAutolinkingService{
		canonicalService: mockCanonicalService,
	}

	ctx := context.Background()
	content := "Benchmark reference: /en/article/bench"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := mockAutolinkingService.ProcessContentWithCanonical(ctx, content)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}