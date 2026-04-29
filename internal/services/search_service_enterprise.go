package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"high-performance-news-website/internal/models"

	meilisearch "github.com/meilisearch/meilisearch-go"
	"github.com/redis/go-redis/v9"
)

// =============================================================================
// ENTERPRISE SEARCH SERVICE
// =============================================================================

// EnterpriseSearchService is the production-grade search service
type EnterpriseSearchService struct {
	// Core dependencies
	indexer SearchIndexerInterface
	cache   *redis.Client
	db      SearchDB

	// Configuration
	config      EnterpriseSearchConfig
	cachePrefix string

	// Resilience components
	circuitBreaker     *EnhancedCircuitBreaker
	concurrencyLimiter *ConcurrencyLimiter
	rateLimiter        *TokenBucketRateLimiter

	// Monitoring
	metrics *EnhancedSearchMetrics
	logger  *StructuredLogger
	audit   *AuditLogger

	// Dead letter queue
	deadLetterQueue *DeadLetterQueue

	// Background workers
	stopChan     chan struct{}
	wg           sync.WaitGroup
	isRunning    int32
}

// NewEnterpriseSearchService creates a new enterprise search service
func NewEnterpriseSearchService(
	indexer SearchIndexerInterface,
	cache *redis.Client,
	db SearchDB,
) *EnterpriseSearchService {
	config := LoadEnterpriseConfigFromEnv()

	ess := &EnterpriseSearchService{
		indexer:     indexer,
		cache:       cache,
		db:          db,
		config:      config,
		cachePrefix: "search:",
		circuitBreaker: NewEnhancedCircuitBreaker(
			config.CircuitBreakerThreshold,
			config.CircuitBreakerBackoffBase,
			config.CircuitBreakerBackoffMax,
		),
		concurrencyLimiter: NewConcurrencyLimiter(config.MaxConcurrentSearches),
		rateLimiter:        NewTokenBucketRateLimiter(config.RateLimitPerSecond, config.RateLimitBurst),
		metrics:            NewEnhancedSearchMetrics(),
		logger:             NewStructuredLogger("search"),
		audit:              NewAuditLogger(10000),
		deadLetterQueue:    NewDeadLetterQueue(config.DeadLetterMaxRetries),
		stopChan:           make(chan struct{}),
	}

	return ess
}

// NewEnterpriseSearchServiceWithConfig creates a service with custom config
func NewEnterpriseSearchServiceWithConfig(
	indexer SearchIndexerInterface,
	cache *redis.Client,
	db SearchDB,
	config EnterpriseSearchConfig,
) *EnterpriseSearchService {
	ess := &EnterpriseSearchService{
		indexer:     indexer,
		cache:       cache,
		db:          db,
		config:      config,
		cachePrefix: "search:",
		circuitBreaker: NewEnhancedCircuitBreaker(
			config.CircuitBreakerThreshold,
			config.CircuitBreakerBackoffBase,
			config.CircuitBreakerBackoffMax,
		),
		concurrencyLimiter: NewConcurrencyLimiter(config.MaxConcurrentSearches),
		rateLimiter:        NewTokenBucketRateLimiter(config.RateLimitPerSecond, config.RateLimitBurst),
		metrics:            NewEnhancedSearchMetrics(),
		logger:             NewStructuredLogger("search"),
		audit:              NewAuditLogger(10000),
		deadLetterQueue:    NewDeadLetterQueue(config.DeadLetterMaxRetries),
		stopChan:           make(chan struct{}),
	}

	return ess
}

// Start starts background workers
func (ess *EnterpriseSearchService) Start() error {
	if !atomic.CompareAndSwapInt32(&ess.isRunning, 0, 1) {
		return fmt.Errorf("service already running")
	}

	ess.logger.Info("service_start", map[string]interface{}{
		"engine_mode":           string(ess.config.EngineMode),
		"max_concurrent":        ess.config.MaxConcurrentSearches,
		"rate_limit_per_second": ess.config.RateLimitPerSecond,
	})

	// Start dead letter queue processor
	ess.wg.Add(1)
	go ess.deadLetterProcessor()

	// Start metrics updater
	ess.wg.Add(1)
	go ess.metricsUpdater()

	// Start reconciliation worker (if MeiliSearch is configured)
	if ess.indexer != nil && ess.config.ReconciliationInterval > 0 {
		ess.wg.Add(1)
		go ess.reconciliationWorker()
	}

	return nil
}

// Stop gracefully stops the service
func (ess *EnterpriseSearchService) Stop() error {
	if !atomic.CompareAndSwapInt32(&ess.isRunning, 1, 0) {
		return fmt.Errorf("service not running")
	}

	ess.logger.Info("service_stop", nil)
	close(ess.stopChan)
	ess.wg.Wait()

	return nil
}

// deadLetterProcessor processes failed indexing operations
func (ess *EnterpriseSearchService) deadLetterProcessor() {
	defer ess.wg.Done()

	ticker := time.NewTicker(ess.config.DeadLetterRetryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ess.stopChan:
			return
		case <-ticker.C:
			ess.processDeadLetterQueue()
		}
	}
}

// processDeadLetterQueue retries failed indexing operations
func (ess *EnterpriseSearchService) processDeadLetterQueue() {
	items := ess.deadLetterQueue.GetRetryable()
	if len(items) == 0 {
		return
	}

	ess.logger.Info("dlq_processing", map[string]interface{}{
		"items_count": len(items),
	})

	for _, item := range items {
		var err error

		if item.Operation == "index" && len(item.ArticleData) > 0 {
			var article models.Article
			if jsonErr := json.Unmarshal(item.ArticleData, &article); jsonErr == nil {
				err = ess.IndexArticle(&article)
			} else {
				err = jsonErr
			}
		} else if item.Operation == "delete" {
			err = ess.RemoveArticle(context.Background(), item.ArticleID)
		}

		if err == nil {
			ess.deadLetterQueue.Remove(item.ArticleID, item.Operation)
			ess.logger.Info("dlq_retry_success", map[string]interface{}{
				"article_id": item.ArticleID,
				"operation":  item.Operation,
			})
		} else {
			ess.logger.Warn("dlq_retry_failed", map[string]interface{}{
				"article_id":  item.ArticleID,
				"operation":   item.Operation,
				"retry_count": item.RetryCount + 1,
				"error":       err.Error(),
			})
		}
	}
}

// metricsUpdater updates Prometheus metrics periodically
func (ess *EnterpriseSearchService) metricsUpdater() {
	defer ess.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ess.stopChan:
			return
		case <-ticker.C:
			cacheHitRatioGauge.Set(ess.metrics.GetCacheHitRatio())
			fallbackRateGauge.Set(ess.metrics.GetFallbackRate())
		}
	}
}

// reconciliationWorker periodically reconciles DB and index
func (ess *EnterpriseSearchService) reconciliationWorker() {
	defer ess.wg.Done()

	ticker := time.NewTicker(ess.config.ReconciliationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ess.stopChan:
			return
		case <-ticker.C:
			result, err := ess.ReconcileIndex(context.Background())
			if err != nil {
				ess.logger.Error("reconciliation_failed", err, nil)
			} else if result.ReconcileNeeded {
				ess.logger.Warn("reconciliation_needed", map[string]interface{}{
					"db_count":    result.DBCount,
					"index_count": result.IndexCount,
					"difference":  result.Difference,
				})
			}
		}
	}
}

// =============================================================================
// MAIN SEARCH METHOD (Enterprise Grade)
// =============================================================================

// Search performs a search with all enterprise features
func (ess *EnterpriseSearchService) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	startTime := time.Now()

	// Rate limiting check
	if !ess.rateLimiter.Allow() {
		atomic.AddInt64(&ess.metrics.RateLimited, 1)
		searchRequestsTotal.WithLabelValues("rate_limited", "rejected").Inc()
		return nil, fmt.Errorf("rate limit exceeded")
	}

	// Concurrency limiting
	if err := ess.concurrencyLimiter.Acquire(ctx); err != nil {
		return nil, fmt.Errorf("concurrency limit: %w", err)
	}
	defer ess.concurrencyLimiter.Release()

	atomic.AddInt64(&ess.metrics.TotalSearches, 1)

	// Input validation and sanitization
	if err := ess.validateAndSanitizeRequest(&req); err != nil {
		atomic.AddInt64(&ess.metrics.Errors, 1)
		searchRequestsTotal.WithLabelValues("validation", "error").Inc()
		return nil, fmt.Errorf("invalid search request: %w", err)
	}

	// Generate cache key
	cacheKey := ess.generateCacheKey(req)

	// Try cache first
	if cachedResult, err := ess.getFromCache(ctx, cacheKey); err == nil {
		atomic.AddInt64(&ess.metrics.CacheHits, 1)
		latency := time.Since(startTime)
		cachedResult.ProcessingTime = latency.Milliseconds()

		ess.recordSearchMetrics(latency, "cache", true)
		ess.logger.SearchComplete(req.Query, "cache", float64(latency.Milliseconds()), len(cachedResult.Articles), true)

		return cachedResult, nil
	}
	atomic.AddInt64(&ess.metrics.CacheMisses, 1)

	var result *SearchResponse
	var err error
	var source string

	// Execute search based on engine mode
	switch ess.config.EngineMode {
	case SearchModePostgres:
		result, err = ess.searchWithPostgreSQL(ctx, req)
		source = "postgresql"

	case SearchModeMeili:
		result, err = ess.searchWithMeiliSearch(ctx, req)
		source = "meilisearch"
		if err != nil {
			// No fallback in meili-only mode
			atomic.AddInt64(&ess.metrics.Errors, 1)
			searchRequestsTotal.WithLabelValues("meilisearch", "error").Inc()
			return nil, err
		}

	case SearchModeHybrid:
		fallthrough
	default:
		// Try MeiliSearch first, fallback to PostgreSQL
		if ess.indexer != nil && !ess.circuitBreaker.IsOpen() {
			result, err = ess.searchWithMeiliSearch(ctx, req)
			if err == nil {
				ess.circuitBreaker.RecordSuccess()
				atomic.AddInt64(&ess.metrics.MeiliSearchHits, 1)
				source = "meilisearch"
			} else {
				ess.circuitBreaker.RecordFailure()
				ess.logger.Warn("meilisearch_failed", map[string]interface{}{
					"error":          err.Error(),
					"circuit_state":  ess.circuitBreaker.GetState(),
					"backoff_duration": ess.circuitBreaker.GetBackoffDuration().String(),
				})
			}
		}

		// Fallback to PostgreSQL
		if result == nil {
			atomic.AddInt64(&ess.metrics.PostgreSQLFallbacks, 1)
			result, err = ess.searchWithPostgreSQL(ctx, req)
			source = "postgresql"
			if err != nil {
				atomic.AddInt64(&ess.metrics.Errors, 1)
				searchRequestsTotal.WithLabelValues("postgresql", "error").Inc()
				return nil, fmt.Errorf("search failed: %w", err)
			}
		}
	}

	latency := time.Since(startTime)
	result.ProcessingTime = latency.Milliseconds()
	result.Source = source

	// Record metrics
	ess.recordSearchMetrics(latency, source, false)

	// Check for slow query
	if latency > ess.config.SlowQueryThreshold {
		atomic.AddInt64(&ess.metrics.SlowQueries, 1)
		ess.logger.SlowQuery(req.Query, float64(latency.Milliseconds()), source, float64(ess.config.SlowQueryThreshold.Milliseconds()))
	}

	// Cache result asynchronously
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), ess.config.CacheTimeout)
		defer cancel()
		if cacheErr := ess.cacheResult(cacheCtx, cacheKey, result); cacheErr != nil {
			ess.logger.Warn("cache_store_failed", map[string]interface{}{
				"error": cacheErr.Error(),
			})
		}
	}()

	ess.logger.SearchComplete(req.Query, source, float64(latency.Milliseconds()), len(result.Articles), false)

	return result, nil
}

// recordSearchMetrics records search metrics
func (ess *EnterpriseSearchService) recordSearchMetrics(latency time.Duration, source string, cacheHit bool) {
	ess.metrics.RecordLatency(latency.Nanoseconds())

	status := "success"
	searchLatencyHistogram.WithLabelValues(source, status).Observe(latency.Seconds())
	searchRequestsTotal.WithLabelValues(source, status).Inc()
}

// validateAndSanitizeRequest validates and sanitizes the search request
func (ess *EnterpriseSearchService) validateAndSanitizeRequest(req *SearchRequest) error {
	// Sanitize query
	req.Query = sanitizeSearchQuery(req.Query)
	if len(req.Query) > ess.config.MaxQueryLength {
		req.Query = req.Query[:ess.config.MaxQueryLength]
	}

	// Enforce limits
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > ess.config.MaxLimit {
		req.Limit = ess.config.MaxLimit
	}

	if req.Offset < 0 {
		req.Offset = 0
	}
	if req.Offset > ess.config.MaxOffset {
		return fmt.Errorf("offset exceeds maximum allowed value of %d", ess.config.MaxOffset)
	}

	// Validate sort fields
	if req.Sort != nil {
		validSortFields := map[string]bool{
			"published_at": true,
			"created_at":   true,
			"view_count":   true,
			"like_count":   true,
			"relevance":    true,
		}
		if !validSortFields[req.Sort.Field] {
			req.Sort.Field = "published_at"
		}
		if req.Sort.Order != "asc" && req.Sort.Order != "desc" {
			req.Sort.Order = "desc"
		}
	}

	return nil
}

// =============================================================================
// MEILISEARCH IMPLEMENTATION
// =============================================================================

// searchWithMeiliSearch performs search using MeiliSearch with proper timeout
func (ess *EnterpriseSearchService) searchWithMeiliSearch(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	if ess.indexer == nil {
		return nil, fmt.Errorf("MeiliSearch indexer not configured")
	}

	// Create timeout context
	searchCtx, cancel := context.WithTimeout(ctx, ess.config.MeiliSearchTimeout)
	defer cancel()

	// Build search request
	searchReq := buildMeiliSearchRequest(req)

	// Execute search with context awareness
	resultChan := make(chan *SearchResponse, 1)
	errChan := make(chan error, 1)

	go func() {
		searchResp, err := ess.indexer.GetClient().Index(ess.indexer.GetIndexName()).Search(req.Query, searchReq)
		if err != nil {
			errChan <- fmt.Errorf("MeiliSearch query failed: %w", err)
			return
		}

		articles, err := convertMeiliHitsToArticles(searchResp.Hits)
		if err != nil {
			errChan <- fmt.Errorf("failed to convert search hits: %w", err)
			return
		}

		resultChan <- &SearchResponse{
			Articles: articles,
			Total:    searchResp.EstimatedTotalHits,
			Limit:    req.Limit,
			Offset:   req.Offset,
		}
	}()

	select {
	case <-searchCtx.Done():
		return nil, fmt.Errorf("MeiliSearch timeout: %w", searchCtx.Err())
	case err := <-errChan:
		return nil, err
	case result := <-resultChan:
		return result, nil
	}
}

// =============================================================================
// POSTGRESQL IMPLEMENTATION
// =============================================================================

// searchWithPostgreSQL performs fallback search using PostgreSQL
func (ess *EnterpriseSearchService) searchWithPostgreSQL(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	if ess.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	// Create timeout context for PostgreSQL query
	pgCtx, cancel := context.WithTimeout(ctx, ess.config.PostgreSQLTimeout)
	defer cancel()

	// Use the existing PostgreSQL search implementation
	ss := &SearchService{
		db:          ess.db,
		config:      ess.config.SearchConfig,
		cachePrefix: ess.cachePrefix,
	}

	// Execute search with timeout context
	resultChan := make(chan *SearchResponse, 1)
	errChan := make(chan error, 1)

	go func() {
		result, err := ss.searchWithPostgreSQL(SearchRequest{
			Query:   req.Query,
			Filters: req.Filters,
			Sort:    req.Sort,
			Limit:   req.Limit,
			Offset:  req.Offset,
		})
		if err != nil {
			errChan <- err
			return
		}
		resultChan <- result
	}()

	select {
	case <-pgCtx.Done():
		return nil, fmt.Errorf("PostgreSQL search timeout: %w", pgCtx.Err())
	case err := <-errChan:
		return nil, err
	case result := <-resultChan:
		return result, nil
	}
}

// =============================================================================
// CACHE OPERATIONS
// =============================================================================

// generateCacheKey creates a unique cache key
func (ess *EnterpriseSearchService) generateCacheKey(req SearchRequest) string {
	ss := &SearchService{cachePrefix: ess.cachePrefix}
	return ss.generateCacheKey(req)
}

// getFromCache retrieves search results from cache
func (ess *EnterpriseSearchService) getFromCache(ctx context.Context, key string) (*SearchResponse, error) {
	if ess.cache == nil {
		return nil, fmt.Errorf("cache not available")
	}

	cacheCtx, cancel := context.WithTimeout(ctx, ess.config.CacheTimeout)
	defer cancel()

	data, err := ess.cache.Get(cacheCtx, key).Result()
	if err != nil {
		return nil, err
	}

	var result SearchResponse
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// cacheResult stores search results in cache
func (ess *EnterpriseSearchService) cacheResult(ctx context.Context, key string, result *SearchResponse) error {
	if ess.cache == nil {
		return nil
	}

	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return ess.cache.Set(ctx, key, data, ess.config.CacheTTL).Err()
}

// InvalidateCache removes cached search results
func (ess *EnterpriseSearchService) InvalidateCache(ctx context.Context, pattern string) error {
	if ess.cache == nil {
		return nil
	}

	if pattern == "" {
		pattern = ess.cachePrefix + "*"
	}

	keys, err := ess.cache.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		ess.logger.Info("cache_invalidate", map[string]interface{}{
			"pattern":    pattern,
			"keys_count": len(keys),
		})
		return ess.cache.Del(ctx, keys...).Err()
	}

	return nil
}

// =============================================================================
// INDEX MANAGEMENT
// =============================================================================

// IndexArticle indexes a single article with dead letter queue support
func (ess *EnterpriseSearchService) IndexArticle(article *models.Article) error {
	startTime := time.Now()

	if ess.indexer == nil {
		ess.logger.Warn("indexer_not_available", map[string]interface{}{
			"article_id": article.ID,
		})
		return nil
	}

	err := ess.indexer.IndexArticle(article)
	if err != nil {
		// Add to dead letter queue
		articleData, _ := json.Marshal(article)
		ess.deadLetterQueue.Add(FailedIndexItem{
			ArticleID:   article.ID,
			Operation:   "index",
			FailedAt:    time.Now(),
			LastError:   err.Error(),
			ArticleData: articleData,
		})
		return fmt.Errorf("failed to index article %d: %w", article.ID, err)
	}

	// Record indexing duration
	indexingDurationHistogram.Observe(time.Since(startTime).Seconds())

	// Invalidate related cache
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		ess.InvalidateCache(ctx, "")
	}()

	ess.logger.Info("article_indexed", map[string]interface{}{
		"article_id": article.ID,
		"title":      article.Title,
		"duration_ms": time.Since(startTime).Milliseconds(),
	})

	return nil
}

// RemoveArticle removes an article from the search index
func (ess *EnterpriseSearchService) RemoveArticle(ctx context.Context, articleID uint64) error {
	if ess.indexer == nil {
		return nil
	}

	err := ess.indexer.DeleteArticle(fmt.Sprintf("%d", articleID))
	if err != nil {
		ess.deadLetterQueue.Add(FailedIndexItem{
			ArticleID: articleID,
			Operation: "delete",
			FailedAt:  time.Now(),
			LastError: err.Error(),
		})
		return fmt.Errorf("failed to remove article %d: %w", articleID, err)
	}

	// Invalidate cache
	go func() {
		invalidateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		ess.InvalidateCache(invalidateCtx, "")
	}()

	return nil
}

// RebuildIndex rebuilds the entire search index with audit logging
func (ess *EnterpriseSearchService) RebuildIndex(ctx context.Context, articles []models.Article, userID, userIP string) error {
	ess.audit.Log(AuditLogEntry{
		Operation: "rebuild_index",
		UserID:    userID,
		UserIP:    userIP,
		Details: map[string]interface{}{
			"article_count": len(articles),
		},
		Success: false,
	})

	if ess.indexer == nil {
		return fmt.Errorf("indexer not available")
	}

	startTime := time.Now()

	err := ess.indexer.RebuildIndex(articles)
	if err != nil {
		ess.audit.Log(AuditLogEntry{
			Operation: "rebuild_index",
			UserID:    userID,
			UserIP:    userIP,
			Details: map[string]interface{}{
				"article_count": len(articles),
				"duration_ms":   time.Since(startTime).Milliseconds(),
			},
			Success:  false,
			ErrorMsg: err.Error(),
		})
		return err
	}

	ess.audit.Log(AuditLogEntry{
		Operation: "rebuild_index",
		UserID:    userID,
		UserIP:    userIP,
		Details: map[string]interface{}{
			"article_count": len(articles),
			"duration_ms":   time.Since(startTime).Milliseconds(),
		},
		Success: true,
	})

	// Invalidate all cache
	go func() {
		invalidateCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		ess.InvalidateCache(invalidateCtx, "")
	}()

	return nil
}

// ReconcileIndex compares DB and index counts
func (ess *EnterpriseSearchService) ReconcileIndex(ctx context.Context) (*IndexReconciliationResult, error) {
	result := &IndexReconciliationResult{
		Timestamp: time.Now(),
	}

	// Get DB count
	if ess.db != nil {
		row := ess.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM articles WHERE status = 'published'")
		if err := row.Scan(&result.DBCount); err != nil {
			return nil, fmt.Errorf("failed to get DB count: %w", err)
		}
	}

	// Get index count
	if ess.indexer != nil {
		stats, err := ess.indexer.GetIndexStats()
		if err != nil {
			return nil, fmt.Errorf("failed to get index stats: %w", err)
		}
		if count, ok := stats["number_of_documents"].(int64); ok {
			result.IndexCount = count
		}
	}

	result.Difference = result.DBCount - result.IndexCount
	result.ReconcileNeeded = result.Difference != 0

	return result, nil
}

// =============================================================================
// HEALTH CHECK (Enterprise Grade)
// =============================================================================

// HealthCheck returns comprehensive health status
func (ess *EnterpriseSearchService) HealthCheck(ctx context.Context) *EnhancedHealthStatus {
	status := &EnhancedHealthStatus{
		Status:     "healthy",
		Timestamp:  time.Now(),
		Components: make(map[string]SearchComponentHealth),
		Metrics:    ess.metrics.ToMap(),
		SystemResources: GetSystemResources(),
	}

	// Check MeiliSearch
	if ess.indexer != nil {
		start := time.Now()
		err := ess.indexer.HealthCheck()
		latency := time.Since(start)

		if err != nil {
			status.Components["meilisearch"] = SearchComponentHealth{
				Status:      "unhealthy",
				Latency:     latency,
				Message:     err.Error(),
				LastChecked: time.Now(),
			}
			status.Status = "degraded"
		} else {
			status.Components["meilisearch"] = SearchComponentHealth{
				Status:      "healthy",
				Latency:     latency,
				LastChecked: time.Now(),
			}
		}
	} else {
		status.Components["meilisearch"] = SearchComponentHealth{
			Status:      "not_configured",
			LastChecked: time.Now(),
		}
	}

	// Check cache
	if ess.cache != nil {
		start := time.Now()
		err := ess.cache.Ping(ctx).Err()
		latency := time.Since(start)

		if err != nil {
			status.Components["cache"] = SearchComponentHealth{
				Status:      "unhealthy",
				Latency:     latency,
				Message:     err.Error(),
				LastChecked: time.Now(),
			}
			status.Status = "degraded"
		} else {
			status.Components["cache"] = SearchComponentHealth{
				Status:      "healthy",
				Latency:     latency,
				LastChecked: time.Now(),
			}
		}
	} else {
		status.Components["cache"] = SearchComponentHealth{
			Status:      "not_configured",
			LastChecked: time.Now(),
		}
	}

	// Check database
	if ess.db != nil {
		start := time.Now()
		row := ess.db.QueryRowContext(ctx, "SELECT 1")
		var result int
		err := row.Scan(&result)
		latency := time.Since(start)

		if err != nil {
			status.Components["database"] = SearchComponentHealth{
				Status:      "unhealthy",
				Latency:     latency,
				Message:     err.Error(),
				LastChecked: time.Now(),
			}
			status.Status = "unhealthy"
		} else {
			status.Components["database"] = SearchComponentHealth{
				Status:      "healthy",
				Latency:     latency,
				LastChecked: time.Now(),
			}
		}
	} else {
		status.Components["database"] = SearchComponentHealth{
			Status:      "not_configured",
			LastChecked: time.Now(),
		}
		status.Status = "unhealthy"
	}

	// Check circuit breaker
	cbState := ess.circuitBreaker.GetState()
	status.Components["circuit_breaker"] = SearchComponentHealth{
		Status:      cbState,
		LastChecked: time.Now(),
		Message:     fmt.Sprintf("backoff: %s", ess.circuitBreaker.GetBackoffDuration()),
	}

	// Add dead letter queue status
	dlqSize := ess.deadLetterQueue.Size()
	status.Metrics["dead_letter_queue_size"] = dlqSize
	if dlqSize > 100 {
		status.Status = "degraded"
	}

	return status
}

// GetMetrics returns current metrics
func (ess *EnterpriseSearchService) GetMetrics() map[string]interface{} {
	return ess.metrics.ToMap()
}

// GetAuditLog returns recent audit entries
func (ess *EnterpriseSearchService) GetAuditLog(count int) []AuditLogEntry {
	return ess.audit.GetRecent(count)
}

// GetDeadLetterQueue returns dead letter queue items
func (ess *EnterpriseSearchService) GetDeadLetterQueue() []FailedIndexItem {
	return ess.deadLetterQueue.GetAll()
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// buildMeiliSearchRequest builds a MeiliSearch request from SearchRequest
func buildMeiliSearchRequest(req SearchRequest) *meilisearch.SearchRequest {
	searchReq := &meilisearch.SearchRequest{
		Query:  req.Query,
		Limit:  int64(req.Limit),
		Offset: int64(req.Offset),
	}

	// Build filter string
	if req.Filters != nil {
		filter := buildMeiliSearchFilter(req.Filters)
		if filter != "" {
			searchReq.Filter = filter
		}
	}

	// Add sorting
	if req.Sort != nil && req.Sort.Field != "relevance" {
		sort := fmt.Sprintf("%s:%s", req.Sort.Field, req.Sort.Order)
		searchReq.Sort = []string{sort}
	}

	return searchReq
}

// buildMeiliSearchFilter constructs MeiliSearch filter string
func buildMeiliSearchFilter(filters *SearchFilters) string {
	var conditions []string

	// Always filter for published articles only
	conditions = append(conditions, "status = published")

	if filters.AuthorID != nil {
		conditions = append(conditions, fmt.Sprintf("author_id = %d", *filters.AuthorID))
	}

	if filters.CategoryID != nil {
		conditions = append(conditions, fmt.Sprintf("category_id = %d", *filters.CategoryID))
	}

	if filters.LanguageCode != "" {
		conditions = append(conditions, fmt.Sprintf("language_code = %s", filters.LanguageCode))
	}

	if len(filters.Tags) > 0 {
		tagConditions := make([]string, len(filters.Tags))
		for i, tag := range filters.Tags {
			tagConditions[i] = fmt.Sprintf("tags = %s", tag)
		}
		conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(tagConditions, " OR ")))
	}

	if filters.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("published_at >= %d", *filters.DateFrom))
	}

	if filters.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("published_at <= %d", *filters.DateTo))
	}

	return strings.Join(conditions, " AND ")
}

// convertMeiliHitsToArticles converts MeiliSearch hits to Article models
func convertMeiliHitsToArticles(hits meilisearch.Hits) ([]models.Article, error) {
	articles := make([]models.Article, 0, len(hits))

	for i, hit := range hits {
		if hit == nil {
			continue
		}

		hitJSON, err := json.Marshal(hit)
		if err != nil {
			log.Printf("[SEARCH] Warning: failed to marshal hit %d: %v", i, err)
			continue
		}

		var doc SearchDocument
		if err := json.Unmarshal(hitJSON, &doc); err != nil {
			log.Printf("[SEARCH] Warning: failed to unmarshal hit %d: %v", i, err)
			continue
		}

		article, err := convertSearchDocumentToArticle(doc)
		if err != nil {
			log.Printf("[SEARCH] Warning: failed to convert document %d: %v", i, err)
			continue
		}

		articles = append(articles, *article)
	}

	return articles, nil
}

// convertSearchDocumentToArticle converts SearchDocument to Article model
func convertSearchDocumentToArticle(doc SearchDocument) (*models.Article, error) {
	id, err := strconv.ParseUint(doc.ID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid article ID: %s", doc.ID)
	}

	var publishedAt *time.Time
	if doc.PublishedAt > 0 {
		t := time.Unix(doc.PublishedAt, 0)
		publishedAt = &t
	}

	tags := make([]models.Tag, len(doc.Tags))
	for i, tagName := range doc.Tags {
		tags[i] = models.Tag{Name: tagName}
	}

	return &models.Article{
		ID:            id,
		Title:         doc.Title,
		Slug:          doc.Slug,
		Content:       doc.Content,
		Excerpt:       doc.Excerpt,
		FeaturedImage: doc.FeaturedImage,
		AuthorID:      doc.AuthorID,
		CategoryID:    doc.CategoryID,
		Tags:          tags,
		Status:        doc.Status,
		PublishedAt:   publishedAt,
		CreatedAt:     time.Unix(doc.CreatedAt, 0),
		ViewCount:     doc.ViewCount,
		LikeCount:     doc.LikeCount,
		LanguageCode:  doc.LanguageCode,
		SEOData: models.SEOData{
			MetaTitle:       doc.MetaTitle,
			MetaDescription: doc.MetaDescription,
			Keywords:        doc.Keywords,
		},
	}, nil
}


// =============================================================================
// INTERFACE COMPATIBILITY METHODS
// =============================================================================

// SearchWithoutContext implements SearchServiceInterface.Search
// This is a wrapper that creates a background context for the enterprise Search method
func (ess *EnterpriseSearchService) SearchWithoutContext(req SearchRequest) (*SearchResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ess.config.MeiliSearchTimeout+ess.config.PostgreSQLTimeout)
	defer cancel()
	return ess.Search(ctx, req)
}

// GetSuggestions returns search suggestions based on query prefix
func (ess *EnterpriseSearchService) GetSuggestions(ctx context.Context, query string, limit int) ([]string, error) {
	// Use MeiliSearch for suggestions if available
	if ess.indexer != nil && !ess.circuitBreaker.IsOpen() {
		searchReq := &meilisearch.SearchRequest{
			Query:  query,
			Limit:  int64(limit),
		}

		searchResp, err := ess.indexer.GetClient().Index(ess.indexer.GetIndexName()).Search(query, searchReq)
		if err == nil {
			suggestions := make([]string, 0, len(searchResp.Hits))
			for _, hit := range searchResp.Hits {
				// Marshal and unmarshal to get the title
				hitJSON, err := json.Marshal(hit)
				if err != nil {
					continue
				}
				var doc map[string]interface{}
				if err := json.Unmarshal(hitJSON, &doc); err != nil {
					continue
				}
				if title, ok := doc["title"].(string); ok {
					suggestions = append(suggestions, title)
				}
			}
			return suggestions, nil
		}
		ess.logger.Warn("suggestions_meilisearch_failed", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Fallback to PostgreSQL
	if ess.db == nil {
		return []string{}, nil
	}

	rows, err := ess.db.QueryContext(ctx, `
		SELECT DISTINCT title 
		FROM articles 
		WHERE status = 'published' 
		AND title ILIKE $1 
		ORDER BY published_at DESC 
		LIMIT $2
	`, query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suggestions []string
	for rows.Next() {
		var title string
		if err := rows.Scan(&title); err == nil {
			suggestions = append(suggestions, title)
		}
	}

	return suggestions, nil
}

// InvalidateCacheForArticle invalidates cache entries related to a specific article
func (ess *EnterpriseSearchService) InvalidateCacheForArticle(article *models.Article) error {
	if ess.cache == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Invalidate all search cache since article content affects search results
	return ess.InvalidateCache(ctx, "")
}

// UpdateArticle updates an article in the search index
func (ess *EnterpriseSearchService) UpdateArticle(ctx context.Context, article *models.Article) error {
	// For MeiliSearch, update is the same as index (upsert behavior)
	return ess.IndexArticle(article)
}
