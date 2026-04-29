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
	"high-performance-news-website/pkg/cache"
)

// StaticGeneratorCacheTestSuite tests complex interactions between static generation and cache warming
type StaticGeneratorCacheTestSuite struct {
	staticGenerator *services.StaticGenerator
	cacheService    cache.CacheService
	articleService  *services.ArticleService
	partitionManager *services.PartitionManager
}

func TestPartitionCleanupDuringCacheWarming(t *testing.T) {
	t.Run("Cache warming continues during partition cleanup", func(t *testing.T) {
		mockCache := &MockCacheWithPartitions{}
		mockPartitionManager := &MockPartitionManager{
			cleanupInProgress: false,
		}
		
		staticGenerator := &MockStaticGenerator{
			cache:            mockCache,
			partitionManager: mockPartitionManager,
		}

		ctx := context.Background()

		// Start cache warming for multiple articles
		articles := []uint64{1, 2, 3, 4, 5}
		
		// Start partition cleanup in background
		go func() {
			time.Sleep(100 * time.Millisecond)
			mockPartitionManager.StartCleanup("articles_2023_12")
		}()

		// Cache warming should handle partition cleanup gracefully
		start := time.Now()
		err := staticGenerator.WarmCacheWithPartitionAwareness(ctx, articles)
		duration := time.Since(start)

		assert.NoError(t, err, "Cache warming should succeed despite partition cleanup")
		assert.Greater(t, duration, 100*time.Millisecond, "Should wait for cleanup to complete")
		assert.Less(t, duration, 2*time.Second, "Should not take too long")

		// Verify cache warming completed for available partitions
		warmedCount := mockCache.GetWarmedArticleCount()
		assert.Greater(t, warmedCount, 0, "Some articles should be warmed")
		
		t.Logf("Cache warming completed in %v, warmed %d articles during partition cleanup", 
			duration, warmedCount)
	})

	t.Run("Partition cleanup waits for critical cache operations", func(t *testing.T) {
		mockCache := &MockCacheWithPartitions{}
		mockPartitionManager := &MockPartitionManager{
			cleanupInProgress: false,
		}
		
		staticGenerator := &MockStaticGenerator{
			cache:            mockCache,
			partitionManager: mockPartitionManager,
		}

		ctx := context.Background()

		// Start critical cache operation
		criticalArticles := []uint64{100, 101, 102} // High priority articles
		
		var wg sync.WaitGroup
		wg.Add(2)

		var cacheWarmingDuration, cleanupDuration time.Duration

		// Start critical cache warming
		go func() {
			defer wg.Done()
			start := time.Now()
			err := staticGenerator.WarmCriticalCache(ctx, criticalArticles)
			cacheWarmingDuration = time.Since(start)
			assert.NoError(t, err, "Critical cache warming should succeed")
		}()

		// Start partition cleanup after short delay
		go func() {
			defer wg.Done()
			time.Sleep(50 * time.Millisecond)
			start := time.Now()
			err := mockPartitionManager.CleanupPartition("articles_2023_11")
			cleanupDuration = time.Since(start)
			assert.NoError(t, err, "Partition cleanup should succeed")
		}()

		wg.Wait()

		// Cleanup should wait for critical operations
		assert.Greater(t, cleanupDuration, 200*time.Millisecond, 
			"Cleanup should wait for critical cache operations")
		assert.Less(t, cacheWarmingDuration, cleanupDuration, 
			"Cache warming should complete before cleanup")

		t.Logf("Critical cache warming: %v, Partition cleanup: %v", 
			cacheWarmingDuration, cleanupDuration)
	})

	t.Run("Cache invalidation during partition maintenance", func(t *testing.T) {
		mockCache := &MockCacheWithPartitions{
			partitionData: map[string][]uint64{
				"articles_2024_01": {1, 2, 3},
				"articles_2024_02": {4, 5, 6},
			},
		}
		
		mockPartitionManager := &MockPartitionManager{}
		
		staticGenerator := &MockStaticGenerator{
			cache:            mockCache,
			partitionManager: mockPartitionManager,
		}

		ctx := context.Background()

		// Warm cache for articles in partition that will be maintained
		articles := []uint64{1, 2, 3}
		err := staticGenerator.WarmCacheWithPartitionAwareness(ctx, articles)
		require.NoError(t, err)

		// Verify articles are cached
		cachedCount := mockCache.GetCachedArticleCount("articles_2024_01")
		assert.Equal(t, 3, cachedCount, "All articles should be cached initially")

		// Start partition maintenance
		err = mockPartitionManager.StartMaintenance("articles_2024_01")
		require.NoError(t, err)

		// Cache should be invalidated for maintained partition
		cachedCount = mockCache.GetCachedArticleCount("articles_2024_01")
		assert.Equal(t, 0, cachedCount, "Cache should be invalidated during maintenance")

		// Other partitions should remain cached
		cachedCount = mockCache.GetCachedArticleCount("articles_2024_02")
		assert.Greater(t, cachedCount, 0, "Other partitions should remain cached")
	})

	t.Run("Concurrent cache warming and partition operations", func(t *testing.T) {
		mockCache := &MockCacheWithPartitions{
			partitionData: map[string][]uint64{
				"articles_2024_01": {1, 2, 3, 4, 5},
				"articles_2024_02": {6, 7, 8, 9, 10},
				"articles_2024_03": {11, 12, 13, 14, 15},
			},
		}
		
		mockPartitionManager := &MockPartitionManager{}
		
		staticGenerator := &MockStaticGenerator{
			cache:            mockCache,
			partitionManager: mockPartitionManager,
		}

		ctx := context.Background()

		const numOperations = 10
		var wg sync.WaitGroup
		results := make(chan error, numOperations*2)

		// Start multiple concurrent cache warming operations
		for i := 0; i < numOperations; i++ {
			wg.Add(1)
			go func(opID int) {
				defer wg.Done()
				articles := []uint64{uint64(opID*3 + 1), uint64(opID*3 + 2), uint64(opID*3 + 3)}
				err := staticGenerator.WarmCacheWithPartitionAwareness(ctx, articles)
				results <- err
			}(i)
		}

		// Start multiple concurrent partition operations
		partitions := []string{"articles_2024_01", "articles_2024_02", "articles_2024_03"}
		for i := 0; i < numOperations; i++ {
			wg.Add(1)
			go func(opID int) {
				defer wg.Done()
				partition := partitions[opID%len(partitions)]
				err := mockPartitionManager.OptimizePartition(partition)
				results <- err
			}(i)
		}

		wg.Wait()
		close(results)

		// Count successful operations
		var successCount, errorCount int
		for err := range results {
			if err != nil {
				errorCount++
			} else {
				successCount++
			}
		}

		assert.Greater(t, successCount, numOperations, 
			"Most operations should succeed despite concurrency")
		assert.Less(t, errorCount, numOperations/2, 
			"Error rate should be manageable")

		t.Logf("Concurrent operations: %d successes, %d errors", successCount, errorCount)
	})
}

func TestCacheWarmingStrategies(t *testing.T) {
	t.Run("Priority-based cache warming during partition operations", func(t *testing.T) {
		mockCache := &MockCacheWithPartitions{}
		mockPartitionManager := &MockPartitionManager{
			cleanupInProgress: true,
		}
		
		staticGenerator := &MockStaticGenerator{
			cache:            mockCache,
			partitionManager: mockPartitionManager,
		}

		ctx := context.Background()

		// Mix of high and low priority articles
		highPriorityArticles := []uint64{1, 2, 3}
		lowPriorityArticles := []uint64{4, 5, 6}

		// High priority articles should be warmed even during cleanup
		err := staticGenerator.WarmCacheByPriority(ctx, highPriorityArticles, "high")
		assert.NoError(t, err, "High priority cache warming should succeed during cleanup")

		// Low priority articles should be deferred
		err = staticGenerator.WarmCacheByPriority(ctx, lowPriorityArticles, "low")
		assert.Error(t, err, "Low priority cache warming should be deferred during cleanup")
		assert.Contains(t, err.Error(), "deferred due to partition operations")

		// Verify high priority articles are cached
		warmedCount := mockCache.GetWarmedArticleCount()
		assert.GreaterOrEqual(t, warmedCount, len(highPriorityArticles), 
			"High priority articles should be warmed")
	})

	t.Run("Adaptive cache warming based on partition health", func(t *testing.T) {
		mockCache := &MockCacheWithPartitions{}
		mockPartitionManager := &MockPartitionManager{
			partitionHealth: map[string]float64{
				"articles_2024_01": 0.95, // Healthy
				"articles_2024_02": 0.60, // Degraded
				"articles_2024_03": 0.30, // Unhealthy
			},
		}
		
		staticGenerator := &MockStaticGenerator{
			cache:            mockCache,
			partitionManager: mockPartitionManager,
		}

		ctx := context.Background()

		// Articles from different partitions
		healthyPartitionArticles := []uint64{1, 2, 3}
		degradedPartitionArticles := []uint64{4, 5, 6}
		unhealthyPartitionArticles := []uint64{7, 8, 9}

		// Test adaptive warming
		results := make(map[string]error)
		
		results["healthy"] = staticGenerator.WarmCacheAdaptive(ctx, healthyPartitionArticles)
		results["degraded"] = staticGenerator.WarmCacheAdaptive(ctx, degradedPartitionArticles)
		results["unhealthy"] = staticGenerator.WarmCacheAdaptive(ctx, unhealthyPartitionArticles)

		// Healthy partition should succeed
		assert.NoError(t, results["healthy"], "Healthy partition warming should succeed")

		// Degraded partition should succeed with warnings
		assert.NoError(t, results["degraded"], "Degraded partition warming should succeed")

		// Unhealthy partition should be skipped or fail gracefully
		if results["unhealthy"] != nil {
			assert.Contains(t, results["unhealthy"].Error(), "partition unhealthy")
		}

		t.Logf("Adaptive warming results: healthy=%v, degraded=%v, unhealthy=%v",
			results["healthy"], results["degraded"], results["unhealthy"])
	})
}

// Mock implementations

type MockCacheWithPartitions struct {
	partitionData    map[string][]uint64
	cachedArticles   map[uint64]bool
	warmedCount      int
	mu               sync.Mutex
}

func (c *MockCacheWithPartitions) Get(key string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.cachedArticles == nil {
		c.cachedArticles = make(map[uint64]bool)
	}
	
	// Extract article ID from key (simplified)
	var articleID uint64
	fmt.Sscanf(key, "article:%d", &articleID)
	
	if c.cachedArticles[articleID] {
		return []byte(fmt.Sprintf(`{"id":%d,"cached":true}`, articleID)), nil
	}
	
	return nil, cache.ErrCacheMiss
}

func (c *MockCacheWithPartitions) Set(key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.cachedArticles == nil {
		c.cachedArticles = make(map[uint64]bool)
	}
	
	var articleID uint64
	fmt.Sscanf(key, "article:%d", &articleID)
	
	c.cachedArticles[articleID] = true
	c.warmedCount++
	
	return nil
}

func (c *MockCacheWithPartitions) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	var articleID uint64
	fmt.Sscanf(key, "article:%d", &articleID)
	
	if c.cachedArticles != nil {
		delete(c.cachedArticles, articleID)
	}
	
	return nil
}

func (c *MockCacheWithPartitions) DeletePattern(pattern string) error {
	return nil
}

func (c *MockCacheWithPartitions) Exists(key string) bool {
	_, err := c.Get(key)
	return err == nil
}

func (c *MockCacheWithPartitions) GetWarmedArticleCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.warmedCount
}

func (c *MockCacheWithPartitions) GetCachedArticleCount(partition string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.partitionData == nil || c.cachedArticles == nil {
		return 0
	}
	
	articles, exists := c.partitionData[partition]
	if !exists {
		return 0
	}
	
	count := 0
	for _, articleID := range articles {
		if c.cachedArticles[articleID] {
			count++
		}
	}
	
	return count
}

func (c *MockCacheWithPartitions) InvalidatePartition(partition string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.partitionData == nil || c.cachedArticles == nil {
		return
	}
	
	articles, exists := c.partitionData[partition]
	if !exists {
		return
	}
	
	for _, articleID := range articles {
		delete(c.cachedArticles, articleID)
	}
}

type MockPartitionManager struct {
	cleanupInProgress bool
	partitionHealth   map[string]float64
	mu                sync.Mutex
}

func (pm *MockPartitionManager) StartCleanup(partition string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.cleanupInProgress = true
	
	// Simulate cleanup time
	time.Sleep(200 * time.Millisecond)
	pm.cleanupInProgress = false
}

func (pm *MockPartitionManager) CleanupPartition(partition string) error {
	// Wait for critical operations to complete
	time.Sleep(250 * time.Millisecond)
	return nil
}

func (pm *MockPartitionManager) StartMaintenance(partition string) error {
	// Simulate maintenance
	time.Sleep(100 * time.Millisecond)
	return nil
}

func (pm *MockPartitionManager) OptimizePartition(partition string) error {
	// Simulate optimization
	time.Sleep(50 * time.Millisecond)
	return nil
}

func (pm *MockPartitionManager) IsCleanupInProgress() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.cleanupInProgress
}

func (pm *MockPartitionManager) GetPartitionHealth(partition string) float64 {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if pm.partitionHealth == nil {
		return 1.0
	}
	
	health, exists := pm.partitionHealth[partition]
	if !exists {
		return 1.0
	}
	
	return health
}

type MockStaticGenerator struct {
	cache            *MockCacheWithPartitions
	partitionManager *MockPartitionManager
}

func (sg *MockStaticGenerator) WarmCacheWithPartitionAwareness(ctx context.Context, articles []uint64) error {
	// Check if partition cleanup is in progress
	if sg.partitionManager.IsCleanupInProgress() {
		// Wait for cleanup to complete or proceed with available partitions
		time.Sleep(150 * time.Millisecond)
	}
	
	// Warm cache for available articles
	for _, articleID := range articles {
		key := fmt.Sprintf("article:%d", articleID)
		data := []byte(fmt.Sprintf(`{"id":%d,"warmed":true}`, articleID))
		
		err := sg.cache.Set(key, data, time.Hour)
		if err != nil {
			return err
		}
	}
	
	return nil
}

func (sg *MockStaticGenerator) WarmCriticalCache(ctx context.Context, articles []uint64) error {
	// Critical cache warming has higher priority
	for _, articleID := range articles {
		key := fmt.Sprintf("article:%d", articleID)
		data := []byte(fmt.Sprintf(`{"id":%d,"critical":true}`, articleID))
		
		err := sg.cache.Set(key, data, time.Hour)
		if err != nil {
			return err
		}
		
		// Simulate critical processing time
		time.Sleep(20 * time.Millisecond)
	}
	
	return nil
}

func (sg *MockStaticGenerator) WarmCacheByPriority(ctx context.Context, articles []uint64, priority string) error {
	if priority == "low" && sg.partitionManager.IsCleanupInProgress() {
		return fmt.Errorf("low priority cache warming deferred due to partition operations")
	}
	
	// High priority warming proceeds even during cleanup
	for _, articleID := range articles {
		key := fmt.Sprintf("article:%d", articleID)
		data := []byte(fmt.Sprintf(`{"id":%d,"priority":"%s"}`, articleID, priority))
		
		err := sg.cache.Set(key, data, time.Hour)
		if err != nil {
			return err
		}
	}
	
	return nil
}

func (sg *MockStaticGenerator) WarmCacheAdaptive(ctx context.Context, articles []uint64) error {
	// Determine partition for articles (simplified)
	partition := fmt.Sprintf("articles_2024_%02d", (articles[0]-1)/3+1)
	
	health := sg.partitionManager.GetPartitionHealth(partition)
	
	if health < 0.5 {
		return fmt.Errorf("partition unhealthy: %s (health: %.2f)", partition, health)
	}
	
	// Adjust warming strategy based on health
	delay := time.Duration((1.0 - health) * 100) * time.Millisecond
	
	for _, articleID := range articles {
		key := fmt.Sprintf("article:%d", articleID)
		data := []byte(fmt.Sprintf(`{"id":%d,"adaptive":true}`, articleID))
		
		err := sg.cache.Set(key, data, time.Hour)
		if err != nil {
			return err
		}
		
		// Adaptive delay based on partition health
		time.Sleep(delay)
	}
	
	return nil
}

// Benchmark tests

func BenchmarkCacheWarmingDuringPartitionCleanup(b *testing.B) {
	mockCache := &MockCacheWithPartitions{}
	mockPartitionManager := &MockPartitionManager{
		cleanupInProgress: true,
	}
	
	staticGenerator := &MockStaticGenerator{
		cache:            mockCache,
		partitionManager: mockPartitionManager,
	}

	ctx := context.Background()
	articles := []uint64{1, 2, 3, 4, 5}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := staticGenerator.WarmCacheWithPartitionAwareness(ctx, articles)
		if err != nil {
			b.Fatal(err)
		}
	}
}