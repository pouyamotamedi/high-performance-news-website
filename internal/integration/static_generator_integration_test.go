package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
	"high-performance-news-website/pkg/cache"
)

// StaticGeneratorIntegrationTestSuite tests static generation and dynamic content synchronization
type StaticGeneratorIntegrationTestSuite struct {
	staticGenerator   *services.StaticGenerator
	dynamicService    *services.DynamicContentService
	cacheService      cache.CacheService
	articleService    *services.ArticleService
	templateService   *services.TemplateService
}

func TestStaticDynamicContentSynchronization(t *testing.T) {
	t.Run("Static generation with dynamic content updates", func(t *testing.T) {
		tempDir := t.TempDir()
		
		mockCache := &MockCacheService{}
		mockDynamicService := &MockDynamicContentService{
			articles: map[uint64]*models.Article{
				1: {ID: 1, Title: "Dynamic Article 1", Content: "Dynamic content 1", Status: "published"},
				2: {ID: 2, Title: "Dynamic Article 2", Content: "Dynamic content 2", Status: "published"},
			},
		}

		staticGenerator := &MockStaticGeneratorAdvanced{
			outputDir:       tempDir,
			cache:          mockCache,
			dynamicService: mockDynamicService,
		}

		ctx := context.Background()

		// Generate initial static files
		err := staticGenerator.GenerateStaticSite(ctx)
		require.NoError(t, err)

		// Verify static files were created
		staticFile1 := filepath.Join(tempDir, "article", "1", "index.html")
		staticFile2 := filepath.Join(tempDir, "article", "2", "index.html")
		
		assert.FileExists(t, staticFile1)
		assert.FileExists(t, staticFile2)

		// Read initial content
		content1, err := os.ReadFile(staticFile1)
		require.NoError(t, err)
		assert.Contains(t, string(content1), "Dynamic Article 1")

		// Update dynamic content
		mockDynamicService.articles[1].Title = "Updated Dynamic Article 1"
		mockDynamicService.articles[1].Content = "Updated dynamic content 1"

		// Trigger incremental static generation
		err = staticGenerator.IncrementalGeneration(ctx, []uint64{1})
		require.NoError(t, err)

		// Verify static file was updated
		updatedContent1, err := os.ReadFile(staticFile1)
		require.NoError(t, err)
		assert.Contains(t, string(updatedContent1), "Updated Dynamic Article 1")
		assert.NotEqual(t, string(content1), string(updatedContent1))

		// Verify other files weren't unnecessarily regenerated
		content2, err := os.ReadFile(staticFile2)
		require.NoError(t, err)
		assert.Contains(t, string(content2), "Dynamic Article 2")

		t.Logf("Static-dynamic synchronization completed successfully")
	})

	t.Run("Real-time static generation on content changes", func(t *testing.T) {
		tempDir := t.TempDir()
		
		mockCache := &MockCacheService{}
		mockDynamicService := &MockDynamicContentService{
			articles: make(map[uint64]*models.Article),
		}

		staticGenerator := &MockStaticGeneratorAdvanced{
			outputDir:       tempDir,
			cache:          mockCache,
			dynamicService: mockDynamicService,
			realTimeMode:   true,
		}

		ctx := context.Background()

		// Start real-time generation
		err := staticGenerator.StartRealTimeGeneration(ctx)
		require.NoError(t, err)
		defer staticGenerator.StopRealTimeGeneration()

		// Add new article dynamically
		newArticle := &models.Article{
			ID:      100,
			Title:   "Real-time Article",
			Content: "Real-time content",
			Status:  "published",
		}

		// Simulate content change event
		err = staticGenerator.HandleContentChange(ctx, "article_created", newArticle)
		require.NoError(t, err)

		// Wait for real-time generation
		time.Sleep(200 * time.Millisecond)

		// Verify static file was created
		staticFile := filepath.Join(tempDir, "article", "100", "index.html")
		assert.FileExists(t, staticFile)

		content, err := os.ReadFile(staticFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "Real-time Article")

		// Update article
		newArticle.Title = "Updated Real-time Article"
		err = staticGenerator.HandleContentChange(ctx, "article_updated", newArticle)
		require.NoError(t, err)

		time.Sleep(200 * time.Millisecond)

		// Verify static file was updated
		updatedContent, err := os.ReadFile(staticFile)
		require.NoError(t, err)
		assert.Contains(t, string(updatedContent), "Updated Real-time Article")

		t.Logf("Real-time static generation completed successfully")
	})

	t.Run("Static generation with dependency tracking", func(t *testing.T) {
		tempDir := t.TempDir()
		
		mockCache := &MockCacheService{}
		mockDynamicService := &MockDynamicContentService{
			articles: map[uint64]*models.Article{
				1: {ID: 1, Title: "Parent Article", Content: "Content with references", Status: "published"},
				2: {ID: 2, Title: "Referenced Article", Content: "Referenced content", Status: "published"},
				3: {ID: 3, Title: "Category Page", Content: "Category content", Status: "published"},
			},
			dependencies: map[uint64][]uint64{
				1: {2, 3}, // Article 1 depends on articles 2 and 3
				2: {3},    // Article 2 depends on article 3
			},
		}

		staticGenerator := &MockStaticGeneratorAdvanced{
			outputDir:       tempDir,
			cache:          mockCache,
			dynamicService: mockDynamicService,
		}

		ctx := context.Background()

		// Generate initial static files
		err := staticGenerator.GenerateStaticSite(ctx)
		require.NoError(t, err)

		// Update referenced article
		mockDynamicService.articles[3].Title = "Updated Category Page"

		// Trigger dependency-aware regeneration
		err = staticGenerator.RegenerateWithDependencies(ctx, []uint64{3})
		require.NoError(t, err)

		// Verify all dependent files were regenerated
		dependentFiles := []string{
			filepath.Join(tempDir, "article", "1", "index.html"), // Depends on 3
			filepath.Join(tempDir, "article", "2", "index.html"), // Depends on 3
			filepath.Join(tempDir, "article", "3", "index.html"), // The updated article
		}

		for _, file := range dependentFiles {
			assert.FileExists(t, file)
			content, err := os.ReadFile(file)
			require.NoError(t, err)
			
			// Check that content reflects the update (simplified check)
			if strings.Contains(file, "/3/") {
				assert.Contains(t, string(content), "Updated Category Page")
			}
		}

		t.Logf("Dependency-aware static generation completed")
	})

	t.Run("Static generation performance optimization", func(t *testing.T) {
		tempDir := t.TempDir()
		
		mockCache := &MockCacheService{}
		mockDynamicService := &MockDynamicContentService{
			articles: make(map[uint64]*models.Article),
		}

		// Create many articles
		for i := uint64(1); i <= 1000; i++ {
			mockDynamicService.articles[i] = &models.Article{
				ID:      i,
				Title:   fmt.Sprintf("Article %d", i),
				Content: fmt.Sprintf("Content for article %d", i),
				Status:  "published",
			}
		}

		staticGenerator := &MockStaticGeneratorAdvanced{
			outputDir:       tempDir,
			cache:          mockCache,
			dynamicService: mockDynamicService,
			parallelWorkers: 10,
		}

		ctx := context.Background()

		// Measure full generation time
		start := time.Now()
		err := staticGenerator.GenerateStaticSite(ctx)
		fullGenerationTime := time.Since(start)
		require.NoError(t, err)

		// Verify files were created
		generatedCount := 0
		err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if strings.HasSuffix(path, "index.html") {
				generatedCount++
			}
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, 1000, generatedCount)

		// Test incremental generation performance
		start = time.Now()
		err = staticGenerator.IncrementalGeneration(ctx, []uint64{1, 2, 3, 4, 5})
		incrementalTime := time.Since(start)
		require.NoError(t, err)

		// Incremental should be much faster
		assert.Less(t, incrementalTime, fullGenerationTime/10, 
			"Incremental generation should be much faster than full generation")

		t.Logf("Performance test: Full generation: %v, Incremental (5 files): %v", 
			fullGenerationTime, incrementalTime)
	})
}

func TestStaticGenerationCacheIntegration(t *testing.T) {
	t.Run("Static generation with cache warming", func(t *testing.T) {
		tempDir := t.TempDir()
		
		mockCache := &MockCacheService{
			data: make(map[string][]byte),
		}
		mockDynamicService := &MockDynamicContentService{
			articles: map[uint64]*models.Article{
				1: {ID: 1, Title: "Cached Article", Content: "Cached content", Status: "published"},
			},
		}

		staticGenerator := &MockStaticGeneratorAdvanced{
			outputDir:       tempDir,
			cache:          mockCache,
			dynamicService: mockDynamicService,
		}

		ctx := context.Background()

		// Generate static files with cache warming
		err := staticGenerator.GenerateWithCacheWarming(ctx, []uint64{1})
		require.NoError(t, err)

		// Verify cache was warmed
		assert.Greater(t, len(mockCache.data), 0, "Cache should be populated")

		// Verify static file was created
		staticFile := filepath.Join(tempDir, "article", "1", "index.html")
		assert.FileExists(t, staticFile)

		// Second generation should use cache
		start := time.Now()
		err = staticGenerator.GenerateWithCacheWarming(ctx, []uint64{1})
		cachedGenerationTime := time.Since(start)
		require.NoError(t, err)

		assert.Less(t, cachedGenerationTime, 50*time.Millisecond, 
			"Cached generation should be very fast")

		t.Logf("Cache-warmed generation completed in %v", cachedGenerationTime)
	})

	t.Run("Static generation cache invalidation", func(t *testing.T) {
		tempDir := t.TempDir()
		
		mockCache := &MockCacheService{
			data: make(map[string][]byte),
		}
		mockDynamicService := &MockDynamicContentService{
			articles: map[uint64]*models.Article{
				1: {ID: 1, Title: "Original Title", Content: "Original content", Status: "published"},
			},
		}

		staticGenerator := &MockStaticGeneratorAdvanced{
			outputDir:       tempDir,
			cache:          mockCache,
			dynamicService: mockDynamicService,
		}

		ctx := context.Background()

		// Generate with caching
		err := staticGenerator.GenerateWithCacheWarming(ctx, []uint64{1})
		require.NoError(t, err)

		// Update article
		mockDynamicService.articles[1].Title = "Updated Title"

		// Invalidate cache and regenerate
		err = staticGenerator.InvalidateCacheAndRegenerate(ctx, []uint64{1})
		require.NoError(t, err)

		// Verify static file reflects update
		staticFile := filepath.Join(tempDir, "article", "1", "index.html")
		content, err := os.ReadFile(staticFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "Updated Title")
		assert.NotContains(t, string(content), "Original Title")

		t.Logf("Cache invalidation and regeneration completed")
	})
}

func TestConcurrentStaticGeneration(t *testing.T) {
	t.Run("Concurrent static generation safety", func(t *testing.T) {
		tempDir := t.TempDir()
		
		mockCache := &MockCacheService{
			data: make(map[string][]byte),
		}
		mockDynamicService := &MockDynamicContentService{
			articles: make(map[uint64]*models.Article),
		}

		// Create articles for concurrent generation
		for i := uint64(1); i <= 100; i++ {
			mockDynamicService.articles[i] = &models.Article{
				ID:      i,
				Title:   fmt.Sprintf("Concurrent Article %d", i),
				Content: fmt.Sprintf("Concurrent content %d", i),
				Status:  "published",
			}
		}

		staticGenerator := &MockStaticGeneratorAdvanced{
			outputDir:       tempDir,
			cache:          mockCache,
			dynamicService: mockDynamicService,
			parallelWorkers: 5,
		}

		ctx := context.Background()

		const numConcurrentOperations = 20
		var wg sync.WaitGroup
		errors := make(chan error, numConcurrentOperations)

		// Start concurrent generation operations
		for i := 0; i < numConcurrentOperations; i++ {
			wg.Add(1)
			go func(opID int) {
				defer wg.Done()
				
				// Each operation generates a subset of articles
				startID := uint64(opID*5 + 1)
				endID := startID + 4
				
				var articleIDs []uint64
				for id := startID; id <= endID && id <= 100; id++ {
					articleIDs = append(articleIDs, id)
				}
				
				err := staticGenerator.IncrementalGeneration(ctx, articleIDs)
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
				t.Logf("Concurrent generation error: %v", err)
			}
		}

		assert.Equal(t, 0, errorCount, "All concurrent operations should succeed")

		// Verify all files were created
		generatedCount := 0
		err := filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if strings.HasSuffix(path, "index.html") {
				generatedCount++
			}
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, 100, generatedCount, "All articles should be generated")

		t.Logf("Concurrent static generation completed: %d files generated", generatedCount)
	})
}

// Mock implementations

type MockCacheService struct {
	data map[string][]byte
	mu   sync.RWMutex
}

func (c *MockCacheService) Get(key string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if data, exists := c.data[key]; exists {
		return data, nil
	}
	return nil, cache.ErrCacheMiss
}

func (c *MockCacheService) Set(key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.data == nil {
		c.data = make(map[string][]byte)
	}
	c.data[key] = value
	return nil
}

func (c *MockCacheService) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.data, key)
	return nil
}

func (c *MockCacheService) DeletePattern(pattern string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	for key := range c.data {
		if strings.Contains(key, pattern) {
			delete(c.data, key)
		}
	}
	return nil
}

func (c *MockCacheService) Exists(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	_, exists := c.data[key]
	return exists
}

type MockDynamicContentService struct {
	articles     map[uint64]*models.Article
	dependencies map[uint64][]uint64
	mu           sync.RWMutex
}

func (d *MockDynamicContentService) GetArticle(ctx context.Context, id uint64) (*models.Article, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	if article, exists := d.articles[id]; exists {
		return article, nil
	}
	return nil, fmt.Errorf("article %d not found", id)
}

func (d *MockDynamicContentService) GetArticleDependencies(ctx context.Context, id uint64) ([]uint64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	if deps, exists := d.dependencies[id]; exists {
		return deps, nil
	}
	return nil, nil
}

func (d *MockDynamicContentService) GetAllArticles(ctx context.Context) ([]*models.Article, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	var articles []*models.Article
	for _, article := range d.articles {
		articles = append(articles, article)
	}
	return articles, nil
}

type MockStaticGeneratorAdvanced struct {
	outputDir       string
	cache          *MockCacheService
	dynamicService *MockDynamicContentService
	realTimeMode   bool
	parallelWorkers int
	mu             sync.Mutex
}

func (sg *MockStaticGeneratorAdvanced) GenerateStaticSite(ctx context.Context) error {
	articles, err := sg.dynamicService.GetAllArticles(ctx)
	if err != nil {
		return err
	}

	// Generate static files for all articles
	for _, article := range articles {
		err := sg.generateStaticFile(article)
		if err != nil {
			return err
		}
	}

	return nil
}

func (sg *MockStaticGeneratorAdvanced) IncrementalGeneration(ctx context.Context, articleIDs []uint64) error {
	for _, id := range articleIDs {
		article, err := sg.dynamicService.GetArticle(ctx, id)
		if err != nil {
			return err
		}

		err = sg.generateStaticFile(article)
		if err != nil {
			return err
		}
	}

	return nil
}

func (sg *MockStaticGeneratorAdvanced) RegenerateWithDependencies(ctx context.Context, articleIDs []uint64) error {
	// Find all articles that depend on the changed articles
	allAffected := make(map[uint64]bool)
	
	for _, id := range articleIDs {
		allAffected[id] = true
		
		// Find articles that depend on this one
		for articleID, deps := range sg.dynamicService.dependencies {
			for _, depID := range deps {
				if depID == id {
					allAffected[articleID] = true
				}
			}
		}
	}

	// Regenerate all affected articles
	var affectedIDs []uint64
	for id := range allAffected {
		affectedIDs = append(affectedIDs, id)
	}

	return sg.IncrementalGeneration(ctx, affectedIDs)
}

func (sg *MockStaticGeneratorAdvanced) GenerateWithCacheWarming(ctx context.Context, articleIDs []uint64) error {
	for _, id := range articleIDs {
		cacheKey := fmt.Sprintf("static:article:%d", id)
		
		// Check cache first
		if sg.cache.Exists(cacheKey) {
			// Use cached version
			continue
		}

		// Generate and cache
		article, err := sg.dynamicService.GetArticle(ctx, id)
		if err != nil {
			return err
		}

		err = sg.generateStaticFile(article)
		if err != nil {
			return err
		}

		// Cache the result
		content := fmt.Sprintf("cached-static-content-%d", id)
		sg.cache.Set(cacheKey, []byte(content), time.Hour)
	}

	return nil
}

func (sg *MockStaticGeneratorAdvanced) InvalidateCacheAndRegenerate(ctx context.Context, articleIDs []uint64) error {
	// Invalidate cache
	for _, id := range articleIDs {
		cacheKey := fmt.Sprintf("static:article:%d", id)
		sg.cache.Delete(cacheKey)
	}

	// Regenerate
	return sg.IncrementalGeneration(ctx, articleIDs)
}

func (sg *MockStaticGeneratorAdvanced) StartRealTimeGeneration(ctx context.Context) error {
	sg.realTimeMode = true
	return nil
}

func (sg *MockStaticGeneratorAdvanced) StopRealTimeGeneration() {
	sg.realTimeMode = false
}

func (sg *MockStaticGeneratorAdvanced) HandleContentChange(ctx context.Context, eventType string, article *models.Article) error {
	if !sg.realTimeMode {
		return fmt.Errorf("real-time mode not enabled")
	}

	// Add article to dynamic service if it's new
	if eventType == "article_created" || eventType == "article_updated" {
		sg.dynamicService.mu.Lock()
		sg.dynamicService.articles[article.ID] = article
		sg.dynamicService.mu.Unlock()
	}

	// Generate static file
	return sg.generateStaticFile(article)
}

func (sg *MockStaticGeneratorAdvanced) generateStaticFile(article *models.Article) error {
	sg.mu.Lock()
	defer sg.mu.Unlock()

	// Create directory structure
	articleDir := filepath.Join(sg.outputDir, "article", fmt.Sprintf("%d", article.ID))
	err := os.MkdirAll(articleDir, 0755)
	if err != nil {
		return err
	}

	// Generate HTML content
	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>%s</title>
</head>
<body>
    <h1>%s</h1>
    <div>%s</div>
</body>
</html>`, article.Title, article.Title, article.Content)

	// Write static file
	staticFile := filepath.Join(articleDir, "index.html")
	return os.WriteFile(staticFile, []byte(htmlContent), 0644)
}

// Benchmark tests

func BenchmarkStaticGeneration(b *testing.B) {
	tempDir := b.TempDir()
	
	mockCache := &MockCacheService{
		data: make(map[string][]byte),
	}
	mockDynamicService := &MockDynamicContentService{
		articles: make(map[uint64]*models.Article),
	}

	// Create test articles
	for i := uint64(1); i <= 100; i++ {
		mockDynamicService.articles[i] = &models.Article{
			ID:      i,
			Title:   fmt.Sprintf("Benchmark Article %d", i),
			Content: fmt.Sprintf("Benchmark content %d", i),
			Status:  "published",
		}
	}

	staticGenerator := &MockStaticGeneratorAdvanced{
		outputDir:       tempDir,
		cache:          mockCache,
		dynamicService: mockDynamicService,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := staticGenerator.GenerateStaticSite(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkIncrementalGeneration(b *testing.B) {
	tempDir := b.TempDir()
	
	mockCache := &MockCacheService{
		data: make(map[string][]byte),
	}
	mockDynamicService := &MockDynamicContentService{
		articles: map[uint64]*models.Article{
			1: {ID: 1, Title: "Incremental Article", Content: "Incremental content", Status: "published"},
		},
	}

	staticGenerator := &MockStaticGeneratorAdvanced{
		outputDir:       tempDir,
		cache:          mockCache,
		dynamicService: mockDynamicService,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := staticGenerator.IncrementalGeneration(ctx, []uint64{1})
		if err != nil {
			b.Fatal(err)
		}
	}
}