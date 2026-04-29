package integration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/queue"
	"high-performance-news-website/internal/services"
	"high-performance-news-website/pkg/cache"
	"high-performance-news-website/pkg/database"
)

// DependencyFailureTestSuite tests system behavior when dependencies fail
type DependencyFailureTestSuite struct {
	db            *database.DB
	cache         cache.CacheService
	searchService *services.SearchService
	emailService  *services.EmailService
	cdnService    *services.CDNService
	jobQueue      *queue.InMemoryJobQueue
	workerPool    *queue.WorkerPool
	memMonitor    *queue.MemoryMonitor
}

func TestDatabaseFailureAndRecovery(t *testing.T) {
	t.Run("Database connection failure handling", func(t *testing.T) {
		// Create a mock database that fails initially
		mockDB := &FailingMockDB{
			failureCount: 3, // Fail first 3 attempts
			maxFailures:  3,
		}

		// Test repository operations with database failures
		articleRepo := &MockArticleRepository{db: mockDB}

		ctx := context.Background()

		// First few attempts should fail
		for i := 0; i < 3; i++ {
			_, err := articleRepo.GetByID(ctx, 1)
			assert.Error(t, err, "Expected database failure on attempt %d", i+1)
			assert.Contains(t, err.Error(), "database connection failed")
		}

		// After max failures, should succeed
		article, err := articleRepo.GetByID(ctx, 1)
		assert.NoError(t, err, "Database should recover after max failures")
		assert.NotNil(t, article)
		assert.Equal(t, uint64(1), article.ID)
	})

	t.Run("Database transaction rollback on failure", func(t *testing.T) {
		mockDB := &FailingMockDB{
			failureCount: 1, // Fail during transaction
			maxFailures:  1,
		}

		articleRepo := &MockArticleRepository{db: mockDB}
		ctx := context.Background()

		// Simulate transaction that fails midway
		article := &models.Article{
			ID:       1,
			Title:    "Test Article",
			Content:  "Test content",
			AuthorID: 1,
			Status:   "published",
		}

		err := articleRepo.Create(ctx, article)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "transaction failed")

		// Verify rollback occurred (no partial data)
		_, err = articleRepo.GetByID(ctx, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Database connection pool exhaustion", func(t *testing.T) {
		mockDB := &FailingMockDB{
			simulatePoolExhaustion: true,
		}

		articleRepo := &MockArticleRepository{db: mockDB}
		ctx := context.Background()

		const numConcurrentRequests = 20
		var wg sync.WaitGroup
		errors := make(chan error, numConcurrentRequests)

		// Simulate many concurrent requests
		for i := 0; i < numConcurrentRequests; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				_, err := articleRepo.GetByID(ctx, uint64(id))
				if err != nil {
					errors <- err
				}
			}(i + 1)
		}

		wg.Wait()
		close(errors)

		// Some requests should fail due to pool exhaustion
		var errorCount int
		for err := range errors {
			if err != nil {
				errorCount++
				assert.Contains(t, err.Error(), "connection pool exhausted")
			}
		}

		assert.Greater(t, errorCount, 0, "Expected some requests to fail due to pool exhaustion")
		t.Logf("Pool exhaustion caused %d/%d requests to fail", errorCount, numConcurrentRequests)
	})

	t.Run("Database partition failure handling", func(t *testing.T) {
		mockDB := &FailingMockDB{
			partitionFailures: map[string]bool{
				"articles_2024_01": true, // Simulate partition failure
			},
		}

		articleRepo := &MockArticleRepository{db: mockDB}
		ctx := context.Background()

		// Try to access article in failed partition
		_, err := articleRepo.GetByDateRange(ctx, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "partition unavailable")

		// Access to other partitions should work
		articles, err := articleRepo.GetByDateRange(ctx, time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 2, 28, 23, 59, 59, 0, time.UTC))
		assert.NoError(t, err)
		assert.NotNil(t, articles)
	})

	t.Run("Database recovery after extended downtime", func(t *testing.T) {
		mockDB := &FailingMockDB{
			extendedDowntime: 5 * time.Second,
		}

		articleRepo := &MockArticleRepository{db: mockDB}
		ctx := context.Background()

		// Requests during downtime should fail
		start := time.Now()
		_, err := articleRepo.GetByID(ctx, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database unavailable")

		// Wait for recovery
		time.Sleep(6 * time.Second)

		// Requests after recovery should succeed
		article, err := articleRepo.GetByID(ctx, 1)
		duration := time.Since(start)
		assert.NoError(t, err)
		assert.NotNil(t, article)
		assert.Greater(t, duration, 5*time.Second, "Should have waited for recovery")
	})
}

func TestCacheFailureAndGracefulDegradation(t *testing.T) {
	t.Run("Cache service unavailable fallback", func(t *testing.T) {
		// Create failing cache service
		failingCache := &FailingMockCache{
			failureRate: 1.0, // 100% failure rate
		}

		// Create service that uses cache with fallback
		searchService := &MockSearchService{
			cache: failingCache,
			db:    &MockDB{},
		}

		ctx := context.Background()
		filters := services.SearchFilters{Query: "test"}

		// Search should still work via database fallback
		results, facets, total, searchTime, err := searchService.SearchArticles(ctx, filters, 10, 0)
		assert.NoError(t, err, "Search should fallback to database when cache fails")
		assert.NotNil(t, results)
		assert.NotNil(t, facets)
		assert.GreaterOrEqual(t, total, 0)
		assert.Greater(t, searchTime, 0.0)

		// Should be slower than cached version but still functional
		assert.Greater(t, searchTime, 50.0, "Fallback should be slower than cache")
	})

	t.Run("Cache partial failure handling", func(t *testing.T) {
		// Cache that fails intermittently
		partialFailCache := &FailingMockCache{
			failureRate: 0.5, // 50% failure rate
		}

		searchService := &MockSearchService{
			cache: partialFailCache,
			db:    &MockDB{},
		}

		ctx := context.Background()
		filters := services.SearchFilters{Query: "test"}

		// Run multiple searches to test partial failure handling
		successCount := 0
		fallbackCount := 0

		for i := 0; i < 20; i++ {
			results, _, _, searchTime, err := searchService.SearchArticles(ctx, filters, 10, 0)
			assert.NoError(t, err, "Search should always work even with cache failures")
			assert.NotNil(t, results)

			if searchTime < 50.0 {
				successCount++ // Fast response indicates cache hit
			} else {
				fallbackCount++ // Slow response indicates database fallback
			}
		}

		assert.Greater(t, successCount, 0, "Some requests should succeed with cache")
		assert.Greater(t, fallbackCount, 0, "Some requests should fallback to database")
		t.Logf("Cache success: %d, Fallback: %d", successCount, fallbackCount)
	})

	t.Run("Cache memory pressure handling", func(t *testing.T) {
		// Cache with memory pressure
		memoryPressureCache := &FailingMockCache{
			memoryPressure: true,
		}

		searchService := &MockSearchService{
			cache: memoryPressureCache,
			db:    &MockDB{},
		}

		ctx := context.Background()

		// High-priority operations should still work
		filters := services.SearchFilters{
			Query:    "urgent",
			Priority: "high",
		}

		results, _, _, _, err := searchService.SearchArticles(ctx, filters, 10, 0)
		assert.NoError(t, err, "High priority operations should work under memory pressure")
		assert.NotNil(t, results)

		// Low-priority operations should fallback
		filters.Priority = "low"
		results, _, _, searchTime, err := searchService.SearchArticles(ctx, filters, 10, 0)
		assert.NoError(t, err, "Low priority operations should fallback gracefully")
		assert.NotNil(t, results)
		assert.Greater(t, searchTime, 50.0, "Should fallback to database under memory pressure")
	})

	t.Run("Cache cluster node failure", func(t *testing.T) {
		// Simulate cache cluster with node failures
		clusterCache := &FailingMockCache{
			nodeFailures: map[string]bool{
				"node1": true,  // Failed node
				"node2": false, // Healthy node
				"node3": false, // Healthy node
			},
		}

		searchService := &MockSearchService{
			cache: clusterCache,
			db:    &MockDB{},
		}

		ctx := context.Background()
		filters := services.SearchFilters{Query: "test"}

		// Should still work with remaining healthy nodes
		results, _, _, _, err := searchService.SearchArticles(ctx, filters, 10, 0)
		assert.NoError(t, err, "Should work with remaining healthy cache nodes")
		assert.NotNil(t, results)

		// Simulate all nodes failing
		clusterCache.nodeFailures["node2"] = true
		clusterCache.nodeFailures["node3"] = true

		// Should fallback to database
		results, _, _, searchTime, err := searchService.SearchArticles(ctx, filters, 10, 0)
		assert.NoError(t, err, "Should fallback when all cache nodes fail")
		assert.NotNil(t, results)
		assert.Greater(t, searchTime, 50.0, "Should use database fallback")
	})
}

func TestExternalAPIFailureAndFallback(t *testing.T) {
	t.Run("Search service MeiliSearch failure fallback", func(t *testing.T) {
		// Create failing MeiliSearch server
		failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "MeiliSearch unavailable"}`))
		}))
		defer failingServer.Close()

		// Create search service with failing MeiliSearch
		db := &MockDB{}
		cache := &MockCache{}
		searchService, err := services.NewSearchService(db, cache, failingServer.URL, "test-key")
		require.NoError(t, err)

		ctx := context.Background()
		filters := services.SearchFilters{Query: "test"}

		// Should fallback to PostgreSQL
		results, facets, total, searchTime, err := searchService.SearchArticles(ctx, filters, 10, 0)
		assert.NoError(t, err, "Should fallback to PostgreSQL when MeiliSearch fails")
		assert.NotNil(t, results)
		assert.NotNil(t, facets)
		assert.GreaterOrEqual(t, total, 0)
		assert.Greater(t, searchTime, 0.0)
	})

	t.Run("Email service provider failure fallback", func(t *testing.T) {
		// Create failing email server
		failingEmailServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error": "Email service unavailable"}`))
		}))
		defer failingEmailServer.Close()

		// Create backup email server
		backupEmailServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message": "Email sent via backup provider"}`))
		}))
		defer backupEmailServer.Close()

		// Test email service with primary failure
		emailService := &MockEmailService{
			primaryURL: failingEmailServer.URL,
			backupURL:  backupEmailServer.URL,
		}

		ctx := context.Background()
		email := &services.EmailMessage{
			To:      "test@example.com",
			Subject: "Test Email",
			Content: "Test content",
		}

		result, err := emailService.SendEmail(ctx, email)
		assert.NoError(t, err, "Should fallback to backup email provider")
		assert.NotNil(t, result)
		assert.Equal(t, "sent", result.Status)
		assert.Contains(t, result.Message, "backup provider")
	})

	t.Run("CDN service failure fallback", func(t *testing.T) {
		// Create failing CDN server
		failingCDNServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(`{"error": "CDN service unavailable"}`))
		}))
		defer failingCDNServer.Close()

		cdnService := &MockCDNService{
			apiURL:      failingCDNServer.URL,
			failoverMode: false,
		}

		// CDN operations should enable failover mode
		purgeRequest := &models.CDNPurgeRequest{
			URLs: []string{"https://example.com/test"},
		}

		response, err := cdnService.PurgeCache(purgeRequest)
		assert.NoError(t, err, "Should handle CDN failure gracefully")
		assert.NotNil(t, response)
		assert.False(t, response.Success)
		assert.Contains(t, response.Message, "failover mode")
		assert.True(t, cdnService.IsFailoverActive())
	})

	t.Run("Social media API rate limiting and failure", func(t *testing.T) {
		// Create rate-limited social media server
		var requestCount int
		var mu sync.Mutex

		rateLimitedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			requestCount++
			count := requestCount
			mu.Unlock()

			if count <= 5 {
				// Allow first 5 requests
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"success": true, "id": "post-` + fmt.Sprintf("%d", count) + `"}`))
			} else {
				// Rate limit subsequent requests
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Hour).Unix()))
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error": "Rate limit exceeded"}`))
			}
		}))
		defer rateLimitedServer.Close()

		socialService := &MockSocialMediaService{
			apiURL: rateLimitedServer.URL,
		}

		ctx := context.Background()

		// First few posts should succeed
		for i := 1; i <= 5; i++ {
			post := &models.SocialMediaPost{
				ID:       uint64(i),
				Platform: models.PlatformTwitter,
				Content: models.PostContent{
					Text: fmt.Sprintf("Test post %d", i),
				},
			}

			result, err := socialService.PublishPost(ctx, post)
			assert.NoError(t, err, "Post %d should succeed", i)
			assert.True(t, result.Success)
		}

		// Subsequent posts should be rate limited
		post := &models.SocialMediaPost{
			ID:       6,
			Platform: models.PlatformTwitter,
			Content: models.PostContent{
				Text: "Rate limited post",
			},
		}

		result, err := socialService.PublishPost(ctx, post)
		assert.NoError(t, err, "Should handle rate limiting gracefully")
		assert.False(t, result.Success)
		assert.True(t, result.RateLimited)
		assert.Greater(t, result.RetryAfter, time.Duration(0))
	})

	t.Run("External API circuit breaker pattern", func(t *testing.T) {
		// Simulate circuit breaker behavior
		circuitBreaker := &MockCircuitBreaker{
			failureThreshold: 5,
			timeout:         2 * time.Second,
		}

		externalService := &MockExternalService{
			circuitBreaker: circuitBreaker,
		}

		ctx := context.Background()

		// Generate failures to trip circuit breaker
		for i := 0; i < 6; i++ {
			_, err := externalService.MakeRequest(ctx, "failing-endpoint")
			if i < 5 {
				assert.Error(t, err, "Request %d should fail", i+1)
			} else {
				assert.Error(t, err, "Circuit breaker should be open")
				assert.Contains(t, err.Error(), "circuit breaker open")
			}
		}

		assert.True(t, circuitBreaker.IsOpen(), "Circuit breaker should be open")

		// Wait for circuit breaker timeout
		time.Sleep(3 * time.Second)

		// Circuit breaker should be half-open, allowing test requests
		assert.True(t, circuitBreaker.IsHalfOpen(), "Circuit breaker should be half-open")

		// Successful request should close circuit breaker
		response, err := externalService.MakeRequest(ctx, "healthy-endpoint")
		assert.NoError(t, err, "Request should succeed in half-open state")
		assert.NotNil(t, response)
		assert.False(t, circuitBreaker.IsOpen(), "Circuit breaker should be closed after success")
	})
}

func TestJobQueueMemoryPressureAndRecovery(t *testing.T) {
	t.Run("Job queue memory pressure handling", func(t *testing.T) {
		// Create memory monitor with low threshold
		memMonitor := queue.NewMemoryMonitor()
		memMonitor.SetMemoryThreshold(1024) // Very low threshold to simulate pressure

		workerPool := queue.NewWorkerPool(2)
		jobQueue := queue.NewInMemoryJobQueue(memMonitor, workerPool)

		handler := &TestJobHandler{}
		jobQueue.RegisterHandler(handler)

		ctx := context.Background()
		err := jobQueue.Start(ctx)
		require.NoError(t, err)
		defer jobQueue.Stop()

		// High priority jobs should be accepted even under memory pressure
		highPriorityJob := &queue.Job{
			ID:       "high-priority-job",
			Type:     queue.JobTypeStaticGeneration,
			Priority: queue.PriorityHigh,
		}

		err = jobQueue.Enqueue(highPriorityJob)
		assert.NoError(t, err, "High priority jobs should be accepted under memory pressure")

		// Low priority jobs should be rejected under memory pressure
		lowPriorityJob := &queue.Job{
			ID:       "low-priority-job",
			Type:     queue.JobTypeStaticGeneration,
			Priority: queue.PriorityLow,
		}

		err = jobQueue.Enqueue(lowPriorityJob)
		assert.Error(t, err, "Low priority jobs should be rejected under memory pressure")
		assert.Contains(t, err.Error(), "memory pressure")
	})

	t.Run("Worker pool recovery after memory pressure", func(t *testing.T) {
		// Create memory monitor that recovers after some time
		memMonitor := &RecoveringMemoryMonitor{
			initialPressure: true,
			recoveryTime:    2 * time.Second,
		}

		workerPool := queue.NewWorkerPool(2)
		jobQueue := queue.NewInMemoryJobQueue(memMonitor, workerPool)

		handler := &TestJobHandler{}
		jobQueue.RegisterHandler(handler)

		ctx := context.Background()
		err := jobQueue.Start(ctx)
		require.NoError(t, err)
		defer jobQueue.Stop()

		// Initially under memory pressure
		lowPriorityJob := &queue.Job{
			ID:       "low-priority-job-1",
			Type:     queue.JobTypeStaticGeneration,
			Priority: queue.PriorityLow,
		}

		err = jobQueue.Enqueue(lowPriorityJob)
		assert.Error(t, err, "Should reject low priority jobs initially")

		// Wait for memory pressure to recover
		time.Sleep(3 * time.Second)

		// Should now accept low priority jobs
		lowPriorityJob2 := &queue.Job{
			ID:       "low-priority-job-2",
			Type:     queue.JobTypeStaticGeneration,
			Priority: queue.PriorityLow,
		}

		err = jobQueue.Enqueue(lowPriorityJob2)
		assert.NoError(t, err, "Should accept low priority jobs after recovery")
	})

	t.Run("Job queue worker failure and recovery", func(t *testing.T) {
		memMonitor := queue.NewMemoryMonitor()
		workerPool := queue.NewWorkerPool(3)
		jobQueue := queue.NewInMemoryJobQueue(memMonitor, workerPool)

		// Handler that fails occasionally
		failingHandler := &FailingJobHandler{
			failureRate: 0.3, // 30% failure rate
		}
		jobQueue.RegisterHandler(failingHandler)

		ctx := context.Background()
		err := jobQueue.Start(ctx)
		require.NoError(t, err)
		defer jobQueue.Stop()

		// Submit multiple jobs
		const numJobs = 20
		var wg sync.WaitGroup
		successCount := 0
		failureCount := 0
		var mu sync.Mutex

		for i := 0; i < numJobs; i++ {
			wg.Add(1)
			go func(jobID int) {
				defer wg.Done()

				job := &queue.Job{
					ID:       fmt.Sprintf("job-%d", jobID),
					Type:     queue.JobTypeStaticGeneration,
					Priority: queue.PriorityMedium,
				}

				err := jobQueue.Enqueue(job)
				if err != nil {
					mu.Lock()
					failureCount++
					mu.Unlock()
				} else {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
			}(i)
		}

		wg.Wait()

		// Wait for jobs to be processed
		time.Sleep(2 * time.Second)

		mu.Lock()
		totalSubmitted := successCount + failureCount
		mu.Unlock()

		assert.Equal(t, numJobs, totalSubmitted, "All jobs should be submitted")
		assert.Greater(t, successCount, 0, "Some jobs should succeed")

		// Check processed jobs
		processedJobs := failingHandler.GetProcessedJobs()
		failedJobs := failingHandler.GetFailedJobs()

		t.Logf("Submitted: %d, Processed: %d, Failed: %d", 
			totalSubmitted, len(processedJobs), len(failedJobs))

		// Some jobs should be processed despite failures
		assert.Greater(t, len(processedJobs), 0, "Some jobs should be processed successfully")
	})

	t.Run("Job queue graceful shutdown under load", func(t *testing.T) {
		memMonitor := queue.NewMemoryMonitor()
		workerPool := queue.NewWorkerPool(2)
		jobQueue := queue.NewInMemoryJobQueue(memMonitor, workerPool)

		// Handler that takes time to process
		slowHandler := &SlowJobHandler{
			processingTime: 500 * time.Millisecond,
		}
		jobQueue.RegisterHandler(slowHandler)

		ctx := context.Background()
		err := jobQueue.Start(ctx)
		require.NoError(t, err)

		// Submit several slow jobs
		for i := 0; i < 5; i++ {
			job := &queue.Job{
				ID:       fmt.Sprintf("slow-job-%d", i),
				Type:     queue.JobTypeStaticGeneration,
				Priority: queue.PriorityMedium,
			}
			err := jobQueue.Enqueue(job)
			assert.NoError(t, err)
		}

		// Give jobs time to start processing
		time.Sleep(100 * time.Millisecond)

		// Shutdown should wait for jobs to complete
		start := time.Now()
		err = jobQueue.Stop()
		shutdownDuration := time.Since(start)

		assert.NoError(t, err, "Shutdown should succeed")
		assert.Greater(t, shutdownDuration, 400*time.Millisecond, "Should wait for jobs to complete")
		assert.Less(t, shutdownDuration, 2*time.Second, "Should not wait too long")

		t.Logf("Graceful shutdown took %v", shutdownDuration)
	})
}

// Mock implementations for testing

type FailingMockDB struct {
	failureCount            int
	maxFailures             int
	simulatePoolExhaustion  bool
	partitionFailures       map[string]bool
	extendedDowntime        time.Duration
	downtimeStart           time.Time
}

func (db *FailingMockDB) simulateFailure() error {
	if db.extendedDowntime > 0 {
		if db.downtimeStart.IsZero() {
			db.downtimeStart = time.Now()
		}
		if time.Since(db.downtimeStart) < db.extendedDowntime {
			return errors.New("database unavailable during downtime")
		}
	}

	if db.failureCount > 0 {
		db.failureCount--
		return errors.New("database connection failed")
	}

	if db.simulatePoolExhaustion {
		return errors.New("connection pool exhausted")
	}

	return nil
}

type MockArticleRepository struct {
	db *FailingMockDB
}

func (r *MockArticleRepository) GetByID(ctx context.Context, id uint64) (*models.Article, error) {
	if err := r.db.simulateFailure(); err != nil {
		return nil, err
	}

	return &models.Article{
		ID:    id,
		Title: fmt.Sprintf("Article %d", id),
	}, nil
}

func (r *MockArticleRepository) Create(ctx context.Context, article *models.Article) error {
	if err := r.db.simulateFailure(); err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}
	return nil
}

func (r *MockArticleRepository) GetByDateRange(ctx context.Context, start, end time.Time) ([]models.Article, error) {
	partition := fmt.Sprintf("articles_%d_%02d", start.Year(), start.Month())
	
	if r.db.partitionFailures != nil && r.db.partitionFailures[partition] {
		return nil, fmt.Errorf("partition unavailable: %s", partition)
	}

	if err := r.db.simulateFailure(); err != nil {
		return nil, err
	}

	return []models.Article{
		{ID: 1, Title: "Test Article"},
	}, nil
}

type FailingMockCache struct {
	failureRate    float64
	memoryPressure bool
	nodeFailures   map[string]bool
}

func (c *FailingMockCache) Get(key string) ([]byte, error) {
	if c.shouldFail() {
		return nil, errors.New("cache unavailable")
	}
	return nil, cache.ErrCacheMiss
}

func (c *FailingMockCache) Set(key string, value []byte, ttl time.Duration) error {
	if c.shouldFail() {
		return errors.New("cache unavailable")
	}
	return nil
}

func (c *FailingMockCache) Delete(key string) error {
	if c.shouldFail() {
		return errors.New("cache unavailable")
	}
	return nil
}

func (c *FailingMockCache) DeletePattern(pattern string) error {
	if c.shouldFail() {
		return errors.New("cache unavailable")
	}
	return nil
}

func (c *FailingMockCache) Exists(key string) bool {
	return !c.shouldFail()
}

func (c *FailingMockCache) shouldFail() bool {
	if c.memoryPressure {
		return true
	}
	
	if c.nodeFailures != nil {
		// Check if all nodes are failed
		allFailed := true
		for _, failed := range c.nodeFailures {
			if !failed {
				allFailed = false
				break
			}
		}
		if allFailed {
			return true
		}
	}
	
	// Random failure based on failure rate
	return c.failureRate > 0 && (c.failureRate >= 1.0 || time.Now().UnixNano()%100 < int64(c.failureRate*100))
}

type MockSearchService struct {
	cache cache.CacheService
	db    *MockDB
}

func (s *MockSearchService) SearchArticles(ctx context.Context, filters services.SearchFilters, limit, offset int) ([]models.Article, map[string]interface{}, int, float64, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("search:%s", filters.Query)
	_, err := s.cache.Get(cacheKey)
	
	if err == nil {
		// Cache hit - fast response
		return []models.Article{{ID: 1, Title: "Cached Result"}}, map[string]interface{}{}, 1, 10.0, nil
	}
	
	// Cache miss or failure - fallback to database
	results := []models.Article{{ID: 1, Title: "Database Result"}}
	facets := map[string]interface{}{"categories": []string{"tech"}}
	
	// Simulate database query time
	time.Sleep(50 * time.Millisecond)
	
	return results, facets, 1, 75.0, nil
}

type MockEmailService struct {
	primaryURL string
	backupURL  string
}

func (e *MockEmailService) SendEmail(ctx context.Context, email *services.EmailMessage) (*services.EmailResult, error) {
	// Try primary provider first
	resp, err := http.Get(e.primaryURL + "/send")
	if err == nil && resp.StatusCode == http.StatusOK {
		return &services.EmailResult{
			Status:  "sent",
			Message: "Email sent via primary provider",
		}, nil
	}
	
	// Fallback to backup provider
	resp, err = http.Get(e.backupURL + "/send")
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("both email providers failed")
	}
	
	return &services.EmailResult{
		Status:  "sent",
		Message: "Email sent via backup provider",
	}, nil
}

type MockCDNService struct {
	apiURL       string
	failoverMode bool
}

func (c *MockCDNService) PurgeCache(request *models.CDNPurgeRequest) (*models.CDNPurgeResponse, error) {
	if c.failoverMode {
		return &models.CDNPurgeResponse{
			Success: false,
			Message: "Operating in failover mode",
		}, nil
	}
	
	// Try CDN API
	resp, err := http.Get(c.apiURL + "/purge")
	if err != nil || resp.StatusCode != http.StatusOK {
		// Enable failover mode
		c.failoverMode = true
		return &models.CDNPurgeResponse{
			Success: false,
			Message: "CDN unavailable, enabled failover mode",
		}, nil
	}
	
	return &models.CDNPurgeResponse{
		Success:   true,
		RequestID: "test-purge-id",
	}, nil
}

func (c *MockCDNService) IsFailoverActive() bool {
	return c.failoverMode
}

type MockSocialMediaService struct {
	apiURL string
}

func (s *MockSocialMediaService) PublishPost(ctx context.Context, post *models.SocialMediaPost) (*models.PostResult, error) {
	resp, err := http.Get(s.apiURL + "/post")
	if err != nil {
		return &models.PostResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := resp.Header.Get("X-RateLimit-Reset")
		var retryDuration time.Duration
		if retryAfter != "" {
			retryDuration = time.Hour // Simplified
		}
		
		return &models.PostResult{
			Success:     false,
			RateLimited: true,
			RetryAfter:  retryDuration,
		}, nil
	}
	
	if resp.StatusCode != http.StatusOK {
		return &models.PostResult{
			Success: false,
			Error:   "API request failed",
		}, nil
	}
	
	return &models.PostResult{
		Success: true,
		PostID:  fmt.Sprintf("post-%d", post.ID),
	}, nil
}

type MockCircuitBreaker struct {
	failureThreshold int
	timeout          time.Duration
	failures         int
	lastFailureTime  time.Time
	state            string // "closed", "open", "half-open"
}

func (cb *MockCircuitBreaker) IsOpen() bool {
	if cb.state == "open" && time.Since(cb.lastFailureTime) > cb.timeout {
		cb.state = "half-open"
	}
	return cb.state == "open"
}

func (cb *MockCircuitBreaker) IsHalfOpen() bool {
	return cb.state == "half-open"
}

func (cb *MockCircuitBreaker) RecordFailure() {
	cb.failures++
	cb.lastFailureTime = time.Now()
	if cb.failures >= cb.failureThreshold {
		cb.state = "open"
	}
}

func (cb *MockCircuitBreaker) RecordSuccess() {
	cb.failures = 0
	cb.state = "closed"
}

type MockExternalService struct {
	circuitBreaker *MockCircuitBreaker
}

func (s *MockExternalService) MakeRequest(ctx context.Context, endpoint string) (interface{}, error) {
	if s.circuitBreaker.IsOpen() {
		return nil, errors.New("circuit breaker open")
	}
	
	// Simulate request
	if endpoint == "failing-endpoint" {
		s.circuitBreaker.RecordFailure()
		return nil, errors.New("request failed")
	}
	
	if endpoint == "healthy-endpoint" {
		s.circuitBreaker.RecordSuccess()
		return map[string]string{"status": "success"}, nil
	}
	
	return nil, errors.New("unknown endpoint")
}

type RecoveringMemoryMonitor struct {
	initialPressure bool
	recoveryTime    time.Duration
	startTime       time.Time
}

func (m *RecoveringMemoryMonitor) IsMemoryPressure() bool {
	if m.startTime.IsZero() {
		m.startTime = time.Now()
	}
	
	if m.initialPressure && time.Since(m.startTime) < m.recoveryTime {
		return true
	}
	
	return false
}

func (m *RecoveringMemoryMonitor) GetMemoryUsage() (uint64, error) {
	return 1024 * 1024 * 1024, nil // 1GB
}

func (m *RecoveringMemoryMonitor) GetMemoryThreshold() uint64 {
	return 2 * 1024 * 1024 * 1024 // 2GB
}

func (m *RecoveringMemoryMonitor) SetMemoryThreshold(threshold uint64) {
	// No-op for mock
}

type FailingJobHandler struct {
	failureRate   float64
	processedJobs []string
	failedJobs    []string
	mu            sync.Mutex
}

func (h *FailingJobHandler) Handle(ctx context.Context, job *queue.Job) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	// Simulate random failures
	if h.failureRate > 0 && time.Now().UnixNano()%100 < int64(h.failureRate*100) {
		h.failedJobs = append(h.failedJobs, job.ID)
		return fmt.Errorf("job %s failed", job.ID)
	}
	
	h.processedJobs = append(h.processedJobs, job.ID)
	return nil
}

func (h *FailingJobHandler) GetJobType() queue.JobType {
	return queue.JobTypeStaticGeneration
}

func (h *FailingJobHandler) GetProcessedJobs() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	result := make([]string, len(h.processedJobs))
	copy(result, h.processedJobs)
	return result
}

func (h *FailingJobHandler) GetFailedJobs() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	result := make([]string, len(h.failedJobs))
	copy(result, h.failedJobs)
	return result
}

type SlowJobHandler struct {
	processingTime time.Duration
	processedJobs  []string
	mu             sync.Mutex
}

func (h *SlowJobHandler) Handle(ctx context.Context, job *queue.Job) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(h.processingTime):
		h.mu.Lock()
		h.processedJobs = append(h.processedJobs, job.ID)
		h.mu.Unlock()
		return nil
	}
}

func (h *SlowJobHandler) GetJobType() queue.JobType {
	return queue.JobTypeStaticGeneration
}

type TestJobHandler struct {
	processedJobs []string
	mu            sync.Mutex
}

func (h *TestJobHandler) Handle(ctx context.Context, job *queue.Job) error {
	h.mu.Lock()
	h.processedJobs = append(h.processedJobs, job.ID)
	h.mu.Unlock()
	return nil
}

func (h *TestJobHandler) GetJobType() queue.JobType {
	return queue.JobTypeStaticGeneration
}

// Additional mock types for completeness

type MockDB struct{}

func (m *MockDB) Close() error { return nil }

type MockCache struct{}

func (m *MockCache) Get(key string) ([]byte, error) { return nil, cache.ErrCacheMiss }
func (m *MockCache) Set(key string, value []byte, ttl time.Duration) error { return nil }
func (m *MockCache) Delete(key string) error { return nil }
func (m *MockCache) DeletePattern(pattern string) error { return nil }
func (m *MockCache) Exists(key string) bool { return false }