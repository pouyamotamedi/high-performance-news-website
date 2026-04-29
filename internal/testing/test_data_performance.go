package testing

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"
	"math"
	"sort"
)

// TestDataPerformanceOptimizer optimizes test data operations for large datasets
type TestDataPerformanceOptimizer struct {
	db                *sql.DB
	connectionPool    *ConnectionPool
	batchProcessor    *BatchProcessor
	memoryManager     *MemoryManager
	performanceCache  *PerformanceCache
	metrics          *PerformanceMetrics
	config           PerformanceConfig
}

// PerformanceConfig holds configuration for performance optimization
type PerformanceConfig struct {
	MaxBatchSize        int           `json:"max_batch_size"`
	MaxConcurrency      int           `json:"max_concurrency"`
	MemoryLimitMB       int64         `json:"memory_limit_mb"`
	CacheSize           int           `json:"cache_size"`
	ConnectionPoolSize  int           `json:"connection_pool_size"`
	QueryTimeout        time.Duration `json:"query_timeout"`
	EnableProfiling     bool          `json:"enable_profiling"`
	OptimizationLevel   int           `json:"optimization_level"` // 1-3
}

// ConnectionPool manages database connections for optimal performance
type ConnectionPool struct {
	connections chan *sql.DB
	maxSize     int
	mutex       sync.RWMutex
	stats       ConnectionPoolStats
}

// ConnectionPoolStats tracks connection pool statistics
type ConnectionPoolStats struct {
	TotalConnections   int           `json:"total_connections"`
	ActiveConnections  int           `json:"active_connections"`
	IdleConnections    int           `json:"idle_connections"`
	AverageWaitTime    time.Duration `json:"average_wait_time"`
	TotalRequests      int64         `json:"total_requests"`
	FailedRequests     int64         `json:"failed_requests"`
}

// BatchProcessor handles batch operations for optimal throughput
type BatchProcessor struct {
	batchSize     int
	maxWorkers    int
	workQueue     chan BatchJob
	resultQueue   chan BatchResult
	workerPool    sync.WaitGroup
	metrics       BatchMetrics
	mutex         sync.RWMutex
}

// BatchJob represents a batch operation job
type BatchJob struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Data      []interface{}          `json:"data"`
	Processor func([]interface{}) error `json:"-"`
	Context   context.Context        `json:"-"`
}

// BatchResult represents the result of a batch operation
type BatchResult struct {
	JobID     string        `json:"job_id"`
	Success   bool          `json:"success"`
	Error     error         `json:"error"`
	Duration  time.Duration `json:"duration"`
	ItemCount int           `json:"item_count"`
}

// BatchMetrics tracks batch processing metrics
type BatchMetrics struct {
	TotalJobs       int64         `json:"total_jobs"`
	CompletedJobs   int64         `json:"completed_jobs"`
	FailedJobs      int64         `json:"failed_jobs"`
	AverageJobTime  time.Duration `json:"average_job_time"`
	ThroughputPerSec float64      `json:"throughput_per_sec"`
	TotalItemsProcessed int64     `json:"total_items_processed"`
}

// MemoryManager manages memory usage during large data operations
type MemoryManager struct {
	limitMB       int64
	currentUsageMB int64
	mutex         sync.RWMutex
	gcThreshold   float64 // Trigger GC when usage exceeds this percentage
	stats         MemoryStats
}

// MemoryStats tracks memory usage statistics
type MemoryStats struct {
	MaxUsageMB     int64   `json:"max_usage_mb"`
	CurrentUsageMB int64   `json:"current_usage_mb"`
	LimitMB        int64   `json:"limit_mb"`
	GCCount        int64   `json:"gc_count"`
	UsagePercent   float64 `json:"usage_percent"`
}

// PerformanceCache provides caching for frequently accessed data
type PerformanceCache struct {
	cache     map[string]CacheEntry
	maxSize   int
	mutex     sync.RWMutex
	stats     CacheStats
	eviction  EvictionPolicy
}

// CacheEntry represents a cached item
type CacheEntry struct {
	Key        string      `json:"key"`
	Value      interface{} `json:"value"`
	Size       int64       `json:"size"`
	CreatedAt  time.Time   `json:"created_at"`
	AccessedAt time.Time   `json:"accessed_at"`
	AccessCount int64      `json:"access_count"`
}

// CacheStats tracks cache performance statistics
type CacheStats struct {
	TotalEntries int64   `json:"total_entries"`
	HitCount     int64   `json:"hit_count"`
	MissCount    int64   `json:"miss_count"`
	HitRatio     float64 `json:"hit_ratio"`
	TotalSize    int64   `json:"total_size_bytes"`
	Evictions    int64   `json:"evictions"`
}

// EvictionPolicy defines cache eviction strategies
type EvictionPolicy string

const (
	EvictionLRU  EvictionPolicy = "lru"
	EvictionLFU  EvictionPolicy = "lfu"
	EvictionFIFO EvictionPolicy = "fifo"
)

// PerformanceMetrics tracks overall performance metrics
type PerformanceMetrics struct {
	StartTime           time.Time     `json:"start_time"`
	TotalOperations     int64         `json:"total_operations"`
	SuccessfulOps       int64         `json:"successful_operations"`
	FailedOps           int64         `json:"failed_operations"`
	AverageOpTime       time.Duration `json:"average_operation_time"`
	ThroughputPerSec    float64       `json:"throughput_per_second"`
	MemoryEfficiency    float64       `json:"memory_efficiency"`
	CacheEfficiency     float64       `json:"cache_efficiency"`
	DatabaseEfficiency  float64       `json:"database_efficiency"`
	mutex               sync.RWMutex
}

// NewTestDataPerformanceOptimizer creates a new performance optimizer
func NewTestDataPerformanceOptimizer(db *sql.DB, config PerformanceConfig) *TestDataPerformanceOptimizer {
	optimizer := &TestDataPerformanceOptimizer{
		db:     db,
		config: config,
	}
	
	optimizer.connectionPool = NewConnectionPool(config.ConnectionPoolSize)
	optimizer.batchProcessor = NewBatchProcessor(config.MaxBatchSize, config.MaxConcurrency)
	optimizer.memoryManager = NewMemoryManager(config.MemoryLimitMB)
	optimizer.performanceCache = NewPerformanceCache(config.CacheSize, EvictionLRU)
	optimizer.metrics = NewPerformanceMetrics()
	
	return optimizer
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(maxSize int) *ConnectionPool {
	return &ConnectionPool{
		connections: make(chan *sql.DB, maxSize),
		maxSize:     maxSize,
		stats:       ConnectionPoolStats{},
	}
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(batchSize, maxWorkers int) *BatchProcessor {
	processor := &BatchProcessor{
		batchSize:   batchSize,
		maxWorkers:  maxWorkers,
		workQueue:   make(chan BatchJob, maxWorkers*2),
		resultQueue: make(chan BatchResult, maxWorkers*2),
		metrics:     BatchMetrics{},
	}
	
	processor.startWorkers()
	return processor
}

// NewMemoryManager creates a new memory manager
func NewMemoryManager(limitMB int64) *MemoryManager {
	return &MemoryManager{
		limitMB:     limitMB,
		gcThreshold: 0.8, // Trigger GC at 80% usage
		stats: MemoryStats{
			LimitMB: limitMB,
		},
	}
}

// NewPerformanceCache creates a new performance cache
func NewPerformanceCache(maxSize int, eviction EvictionPolicy) *PerformanceCache {
	return &PerformanceCache{
		cache:    make(map[string]CacheEntry),
		maxSize:  maxSize,
		eviction: eviction,
		stats:    CacheStats{},
	}
}

// NewPerformanceMetrics creates a new performance metrics tracker
func NewPerformanceMetrics() *PerformanceMetrics {
	return &PerformanceMetrics{
		StartTime: time.Now(),
	}
}

// OptimizeDataGeneration optimizes large-scale data generation
func (opt *TestDataPerformanceOptimizer) OptimizeDataGeneration(generator *TestDataGenerator, count int) ([]TestArticle, error) {
	log.Printf("Optimizing data generation for %d articles", count)
	
	startTime := time.Now()
	
	// Calculate optimal batch size and worker count
	batchSize := opt.calculateOptimalBatchSize(count)
	workerCount := opt.calculateOptimalWorkerCount()
	
	log.Printf("Using batch size: %d, workers: %d", batchSize, workerCount)
	
	// Pre-allocate memory
	articles := make([]TestArticle, 0, count)
	
	// Generate data in optimized batches
	batches := (count + batchSize - 1) / batchSize
	articleChan := make(chan []TestArticle, batches)
	errorChan := make(chan error, batches)
	
	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			opt.dataGenerationWorker(generator, batchSize, articleChan, errorChan)
		}(i)
	}
	
	// Send work to workers
	go func() {
		for i := 0; i < batches; i++ {
			remainingCount := count - (i * batchSize)
			currentBatchSize := batchSize
			if remainingCount < batchSize {
				currentBatchSize = remainingCount
			}
			
			// Check memory usage before generating more data
			if err := opt.memoryManager.CheckMemoryUsage(); err != nil {
				errorChan <- err
				return
			}
			
			// Generate batch
			batchArticles, err := generator.GenerateMultilingualTestDataParallel(currentBatchSize, 2)
			if err != nil {
				errorChan <- err
				return
			}
			
			articleChan <- batchArticles
		}
		close(articleChan)
	}()
	
	// Wait for workers to complete
	go func() {
		wg.Wait()
		close(errorChan)
	}()
	
	// Collect results
	for batchArticles := range articleChan {
		articles = append(articles, batchArticles...)
		
		// Update progress
		if len(articles)%10000 == 0 {
			log.Printf("Generated %d/%d articles", len(articles), count)
		}
	}
	
	// Check for errors
	for err := range errorChan {
		if err != nil {
			return nil, fmt.Errorf("data generation failed: %w", err)
		}
	}
	
	duration := time.Since(startTime)
	throughput := float64(len(articles)) / duration.Seconds()
	
	log.Printf("Data generation completed: %d articles in %v (%.2f articles/sec)", 
		len(articles), duration, throughput)
	
	// Update metrics
	opt.metrics.RecordOperation(duration, len(articles), true)
	
	return articles, nil
}

// dataGenerationWorker processes data generation jobs
func (opt *TestDataPerformanceOptimizer) dataGenerationWorker(generator *TestDataGenerator, batchSize int, 
	articleChan chan<- []TestArticle, errorChan chan<- error) {
	
	// Worker implementation would go here
	// This is a placeholder for the actual worker logic
}

// OptimizeBulkInsert optimizes bulk database insertions
func (opt *TestDataPerformanceOptimizer) OptimizeBulkInsert(articles []TestArticle) error {
	log.Printf("Optimizing bulk insert for %d articles", len(articles))
	
	startTime := time.Now()
	
	// Calculate optimal batch size for database operations
	dbBatchSize := opt.calculateOptimalDBBatchSize(len(articles))
	
	// Use transaction for better performance
	tx, err := opt.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Prepare optimized insert statement
	stmt, err := tx.Prepare(`
		INSERT INTO articles (
			id, title, slug, content, excerpt, author_id, category_id, 
			status, published_at, created_at, updated_at, view_count, 
			like_count, dislike_count, language_code, translation_group_id,
			meta_title, meta_description, canonical_url, schema_type
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()
	
	// Insert in optimized batches
	inserted := 0
	for i := 0; i < len(articles); i += dbBatchSize {
		end := i + dbBatchSize
		if end > len(articles) {
			end = len(articles)
		}
		
		// Check memory usage
		if err := opt.memoryManager.CheckMemoryUsage(); err != nil {
			return err
		}
		
		// Insert batch
		for j := i; j < end; j++ {
			article := articles[j]
			_, err = stmt.Exec(
				article.ID, article.Title, article.Slug, article.Content, article.Excerpt,
				article.AuthorID, article.CategoryID, article.Status, article.PublishedAt,
				article.CreatedAt, article.UpdatedAt, article.ViewCount, article.LikeCount,
				article.DislikeCount, article.LanguageCode, article.TranslationGroupID,
				article.SEOMetadata.MetaTitle, article.SEOMetadata.MetaDescription,
				article.SEOMetadata.CanonicalURL, article.SEOMetadata.SchemaType,
			)
			if err != nil {
				return fmt.Errorf("failed to insert article %d: %w", article.ID, err)
			}
			inserted++
		}
		
		// Log progress for large datasets
		if len(articles) > 10000 && inserted%10000 == 0 {
			log.Printf("Inserted %d/%d articles", inserted, len(articles))
		}
	}
	
	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	duration := time.Since(startTime)
	throughput := float64(inserted) / duration.Seconds()
	
	log.Printf("Bulk insert completed: %d articles in %v (%.2f articles/sec)", 
		inserted, duration, throughput)
	
	// Update metrics
	opt.metrics.RecordOperation(duration, inserted, true)
	
	return nil
}

// calculateOptimalBatchSize calculates the optimal batch size based on available resources
func (opt *TestDataPerformanceOptimizer) calculateOptimalBatchSize(totalCount int) int {
	// Base batch size on memory limits and total count
	memoryBasedSize := int(opt.config.MemoryLimitMB * 1024 * 1024 / 10000) // Assume ~10KB per article
	
	// Consider total count
	countBasedSize := totalCount / 100 // Aim for ~100 batches
	if countBasedSize < 100 {
		countBasedSize = 100
	}
	
	// Use the smaller of the two, but not less than minimum
	batchSize := int(math.Min(float64(memoryBasedSize), float64(countBasedSize)))
	if batchSize < 100 {
		batchSize = 100
	}
	if batchSize > opt.config.MaxBatchSize {
		batchSize = opt.config.MaxBatchSize
	}
	
	return batchSize
}

// calculateOptimalWorkerCount calculates the optimal number of workers
func (opt *TestDataPerformanceOptimizer) calculateOptimalWorkerCount() int {
	// Base on CPU count and configuration
	cpuCount := runtime.NumCPU()
	
	// Use 2x CPU count for I/O bound operations, but respect limits
	workerCount := cpuCount * 2
	if workerCount > opt.config.MaxConcurrency {
		workerCount = opt.config.MaxConcurrency
	}
	if workerCount < 1 {
		workerCount = 1
	}
	
	return workerCount
}

// calculateOptimalDBBatchSize calculates optimal database batch size
func (opt *TestDataPerformanceOptimizer) calculateOptimalDBBatchSize(totalCount int) int {
	// Database operations typically work better with smaller batches
	batchSize := 1000
	
	// Adjust based on total count
	if totalCount < 10000 {
		batchSize = 500
	} else if totalCount > 100000 {
		batchSize = 2000
	}
	
	return batchSize
}

// CheckMemoryUsage checks current memory usage and triggers GC if needed
func (mm *MemoryManager) CheckMemoryUsage() error {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	currentUsageMB := int64(m.Alloc / 1024 / 1024)
	mm.currentUsageMB = currentUsageMB
	mm.stats.CurrentUsageMB = currentUsageMB
	
	if currentUsageMB > mm.stats.MaxUsageMB {
		mm.stats.MaxUsageMB = currentUsageMB
	}
	
	usagePercent := float64(currentUsageMB) / float64(mm.limitMB)
	mm.stats.UsagePercent = usagePercent
	
	// Check if we're approaching the limit
	if usagePercent > mm.gcThreshold {
		log.Printf("Memory usage at %.1f%%, triggering GC", usagePercent*100)
		runtime.GC()
		mm.stats.GCCount++
		
		// Re-check after GC
		runtime.ReadMemStats(&m)
		newUsageMB := int64(m.Alloc / 1024 / 1024)
		mm.currentUsageMB = newUsageMB
		mm.stats.CurrentUsageMB = newUsageMB
		
		log.Printf("Memory usage after GC: %d MB", newUsageMB)
	}
	
	// Check if we're still over the limit
	if mm.currentUsageMB > mm.limitMB {
		return fmt.Errorf("memory usage (%d MB) exceeds limit (%d MB)", mm.currentUsageMB, mm.limitMB)
	}
	
	return nil
}

// startWorkers starts the batch processing workers
func (bp *BatchProcessor) startWorkers() {
	for i := 0; i < bp.maxWorkers; i++ {
		bp.workerPool.Add(1)
		go bp.worker(i)
	}
}

// worker processes batch jobs
func (bp *BatchProcessor) worker(workerID int) {
	defer bp.workerPool.Done()
	
	for job := range bp.workQueue {
		startTime := time.Now()
		
		err := job.Processor(job.Data)
		
		duration := time.Since(startTime)
		
		result := BatchResult{
			JobID:     job.ID,
			Success:   err == nil,
			Error:     err,
			Duration:  duration,
			ItemCount: len(job.Data),
		}
		
		bp.resultQueue <- result
		
		// Update metrics
		bp.mutex.Lock()
		bp.metrics.TotalJobs++
		if err == nil {
			bp.metrics.CompletedJobs++
		} else {
			bp.metrics.FailedJobs++
		}
		bp.metrics.TotalItemsProcessed += int64(len(job.Data))
		bp.mutex.Unlock()
	}
}

// RecordOperation records a performance operation
func (pm *PerformanceMetrics) RecordOperation(duration time.Duration, itemCount int, success bool) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	
	pm.TotalOperations++
	if success {
		pm.SuccessfulOps++
	} else {
		pm.FailedOps++
	}
	
	// Update average operation time
	totalTime := time.Duration(pm.TotalOperations-1) * pm.AverageOpTime + duration
	pm.AverageOpTime = totalTime / time.Duration(pm.TotalOperations)
	
	// Update throughput
	elapsed := time.Since(pm.StartTime)
	if elapsed > 0 {
		pm.ThroughputPerSec = float64(pm.SuccessfulOps) / elapsed.Seconds()
	}
}

// Get returns a cached value
func (pc *PerformanceCache) Get(key string) (interface{}, bool) {
	pc.mutex.RLock()
	defer pc.mutex.RUnlock()
	
	entry, exists := pc.cache[key]
	if !exists {
		pc.stats.MissCount++
		pc.updateHitRatio()
		return nil, false
	}
	
	// Update access information
	entry.AccessedAt = time.Now()
	entry.AccessCount++
	pc.cache[key] = entry
	
	pc.stats.HitCount++
	pc.updateHitRatio()
	
	return entry.Value, true
}

// Set stores a value in the cache
func (pc *PerformanceCache) Set(key string, value interface{}, size int64) {
	pc.mutex.Lock()
	defer pc.mutex.Unlock()
	
	// Check if we need to evict entries
	if len(pc.cache) >= pc.maxSize {
		pc.evict()
	}
	
	entry := CacheEntry{
		Key:        key,
		Value:      value,
		Size:       size,
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
		AccessCount: 1,
	}
	
	pc.cache[key] = entry
	pc.stats.TotalEntries++
	pc.stats.TotalSize += size
}

// evict removes entries based on the eviction policy
func (pc *PerformanceCache) evict() {
	if len(pc.cache) == 0 {
		return
	}
	
	var keyToEvict string
	
	switch pc.eviction {
	case EvictionLRU:
		keyToEvict = pc.findLRUKey()
	case EvictionLFU:
		keyToEvict = pc.findLFUKey()
	case EvictionFIFO:
		keyToEvict = pc.findFIFOKey()
	default:
		keyToEvict = pc.findLRUKey()
	}
	
	if keyToEvict != "" {
		entry := pc.cache[keyToEvict]
		delete(pc.cache, keyToEvict)
		pc.stats.TotalSize -= entry.Size
		pc.stats.Evictions++
	}
}

// findLRUKey finds the least recently used key
func (pc *PerformanceCache) findLRUKey() string {
	var oldestKey string
	var oldestTime time.Time
	
	for key, entry := range pc.cache {
		if oldestKey == "" || entry.AccessedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.AccessedAt
		}
	}
	
	return oldestKey
}

// findLFUKey finds the least frequently used key
func (pc *PerformanceCache) findLFUKey() string {
	var leastUsedKey string
	var leastCount int64 = -1
	
	for key, entry := range pc.cache {
		if leastCount == -1 || entry.AccessCount < leastCount {
			leastUsedKey = key
			leastCount = entry.AccessCount
		}
	}
	
	return leastUsedKey
}

// findFIFOKey finds the first in, first out key
func (pc *PerformanceCache) findFIFOKey() string {
	var oldestKey string
	var oldestTime time.Time
	
	for key, entry := range pc.cache {
		if oldestKey == "" || entry.CreatedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.CreatedAt
		}
	}
	
	return oldestKey
}

// updateHitRatio updates the cache hit ratio
func (pc *PerformanceCache) updateHitRatio() {
	total := pc.stats.HitCount + pc.stats.MissCount
	if total > 0 {
		pc.stats.HitRatio = float64(pc.stats.HitCount) / float64(total)
	}
}

// GetPerformanceReport generates a comprehensive performance report
func (opt *TestDataPerformanceOptimizer) GetPerformanceReport() map[string]interface{} {
	report := map[string]interface{}{
		"memory_stats":     opt.memoryManager.GetStats(),
		"cache_stats":      opt.performanceCache.GetStats(),
		"batch_stats":      opt.batchProcessor.GetStats(),
		"connection_stats": opt.connectionPool.GetStats(),
		"overall_metrics":  opt.metrics.GetMetrics(),
		"recommendations":  opt.generateRecommendations(),
	}
	
	return report
}

// GetStats returns memory statistics
func (mm *MemoryManager) GetStats() MemoryStats {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()
	return mm.stats
}

// GetStats returns cache statistics
func (pc *PerformanceCache) GetStats() CacheStats {
	pc.mutex.RLock()
	defer pc.mutex.RUnlock()
	return pc.stats
}

// GetStats returns batch processing statistics
func (bp *BatchProcessor) GetStats() BatchMetrics {
	bp.mutex.RLock()
	defer bp.mutex.RUnlock()
	return bp.metrics
}

// GetStats returns connection pool statistics
func (cp *ConnectionPool) GetStats() ConnectionPoolStats {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	return cp.stats
}

// GetMetrics returns performance metrics
func (pm *PerformanceMetrics) GetMetrics() map[string]interface{} {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	
	return map[string]interface{}{
		"start_time":           pm.StartTime,
		"total_operations":     pm.TotalOperations,
		"successful_ops":       pm.SuccessfulOps,
		"failed_ops":           pm.FailedOps,
		"average_op_time":      pm.AverageOpTime,
		"throughput_per_sec":   pm.ThroughputPerSec,
		"memory_efficiency":    pm.MemoryEfficiency,
		"cache_efficiency":     pm.CacheEfficiency,
		"database_efficiency":  pm.DatabaseEfficiency,
	}
}

// generateRecommendations generates performance optimization recommendations
func (opt *TestDataPerformanceOptimizer) generateRecommendations() []string {
	var recommendations []string
	
	// Memory recommendations
	memStats := opt.memoryManager.GetStats()
	if memStats.UsagePercent > 0.9 {
		recommendations = append(recommendations, "Consider increasing memory limit or reducing batch sizes")
	}
	
	// Cache recommendations
	cacheStats := opt.performanceCache.GetStats()
	if cacheStats.HitRatio < 0.5 {
		recommendations = append(recommendations, "Cache hit ratio is low, consider increasing cache size or reviewing caching strategy")
	}
	
	// Batch processing recommendations
	batchStats := opt.batchProcessor.GetStats()
	if batchStats.FailedJobs > batchStats.CompletedJobs*10/100 { // > 10% failure rate
		recommendations = append(recommendations, "High batch failure rate detected, consider reducing batch sizes or investigating errors")
	}
	
	// Throughput recommendations
	if opt.metrics.ThroughputPerSec < 100 { // Less than 100 ops/sec
		recommendations = append(recommendations, "Low throughput detected, consider increasing concurrency or optimizing operations")
	}
	
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Performance is within acceptable parameters")
	}
	
	return recommendations
}