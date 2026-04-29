package services

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
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
// CONFIGURATION
// =============================================================================

// SearchConfig holds configurable search parameters
type SearchConfig struct {
	CacheTTL           time.Duration
	MeiliSearchTimeout time.Duration
	MaxQueryLength     int
	MaxLimit           int
	MaxOffset          int
	CircuitBreakerThreshold int
	CircuitBreakerTimeout   time.Duration
}

// DefaultSearchConfig returns production-safe defaults
func DefaultSearchConfig() SearchConfig {
	return SearchConfig{
		CacheTTL:                5 * time.Minute,
		MeiliSearchTimeout:      10 * time.Second,
		MaxQueryLength:          500,
		MaxLimit:                50,
		MaxOffset:               10000,
		CircuitBreakerThreshold: 5,
		CircuitBreakerTimeout:   30 * time.Second,
	}
}

// =============================================================================
// METRICS & MONITORING
// =============================================================================

// SearchMetrics tracks search operation metrics
type SearchMetrics struct {
	TotalSearches      int64
	MeiliSearchHits    int64
	PostgreSQLFallbacks int64
	CacheHits          int64
	CacheMisses        int64
	Errors             int64
	mu                 sync.RWMutex
}

// Global metrics instance
var searchMetrics = &SearchMetrics{}

// GetSearchMetrics returns current search metrics
func GetSearchMetrics() map[string]int64 {
	return map[string]int64{
		"total_searches":       atomic.LoadInt64(&searchMetrics.TotalSearches),
		"meilisearch_hits":     atomic.LoadInt64(&searchMetrics.MeiliSearchHits),
		"postgresql_fallbacks": atomic.LoadInt64(&searchMetrics.PostgreSQLFallbacks),
		"cache_hits":           atomic.LoadInt64(&searchMetrics.CacheHits),
		"cache_misses":         atomic.LoadInt64(&searchMetrics.CacheMisses),
		"errors":               atomic.LoadInt64(&searchMetrics.Errors),
	}
}

// =============================================================================
// CIRCUIT BREAKER
// =============================================================================

// CircuitBreaker implements circuit breaker pattern for MeiliSearch
type CircuitBreaker struct {
	failures    int32
	lastFailure time.Time
	threshold   int32
	timeout     time.Duration
	mu          sync.RWMutex
	isOpen      bool
}

func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		threshold: int32(threshold),
		timeout:   timeout,
	}
}

func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	if !cb.isOpen {
		return false
	}
	
	// Check if timeout has passed
	if time.Since(cb.lastFailure) > cb.timeout {
		return false // Allow retry (half-open state)
	}
	return true
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	cb.isOpen = false
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	cb.lastFailure = time.Now()
	if cb.failures >= cb.threshold {
		cb.isOpen = true
		log.Printf("[CIRCUIT_BREAKER] MeiliSearch circuit breaker OPENED after %d failures", cb.failures)
	}
}

// =============================================================================
// DATA TYPES
// =============================================================================

// SearchFilters represents search filtering options
type SearchFilters struct {
	Query        string    `json:"query"`
	AuthorID     *uint64   `json:"author_id,omitempty"`
	CategoryID   *uint64   `json:"category_id,omitempty"`
	Categories   []uint64  `json:"categories,omitempty"`
	Tags         []string  `json:"tags,omitempty"`
	LanguageCode string    `json:"language_code,omitempty"`
	DateFrom     *int64    `json:"date_from,omitempty"`
	DateTo       *int64    `json:"date_to,omitempty"`
	Status       string    `json:"status,omitempty"`
	SortBy       string    `json:"sort_by,omitempty"`
	SortOrder    string    `json:"sort_order,omitempty"`
}

// SearchSort represents sorting options
type SearchSort struct {
	Field string `json:"field"`
	Order string `json:"order"`
}

// SearchRequest represents a complete search request
type SearchRequest struct {
	Query   string         `json:"query"`
	Filters *SearchFilters `json:"filters,omitempty"`
	Sort    *SearchSort    `json:"sort,omitempty"`
	Limit   int            `json:"limit"`
	Offset  int            `json:"offset"`
}

// SearchResponse represents search results (normalized schema)
type SearchResponse struct {
	Articles       []models.Article `json:"articles"`
	Total          int64            `json:"total"`
	Limit          int              `json:"limit"`
	Offset         int              `json:"offset"`
	ProcessingTime int64            `json:"processing_time_ms"`
	Source         string           `json:"source"`
}

// SearchResult represents a single search result for API responses
type SearchResult struct {
	ID          uint64   `json:"id"`
	Title       string   `json:"title"`
	Slug        string   `json:"slug"`
	Excerpt     string   `json:"excerpt"`
	AuthorName  string   `json:"author_name"`
	Category    string   `json:"category"`
	PublishedAt string   `json:"published_at"`
	ViewCount   uint64   `json:"view_count"`
	Score       float64  `json:"score"`
	Tags        []string `json:"tags"`
	Highlights  []string `json:"highlights"`
}

// SearchFacets represents search facets for filtering
type SearchFacets struct {
	Categories []FacetItem `json:"categories"`
	Tags       []FacetItem `json:"tags"`
	Authors    []FacetItem `json:"authors"`
}

// FacetItem represents a single facet item
type FacetItem struct {
	ID    uint64 `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// PopularSearch represents a popular search query
type PopularSearch struct {
	Query string `json:"query"`
	Count int    `json:"count"`
}

// IndexStats represents search index statistics
type IndexStats struct {
	TotalDocuments   int64     `json:"total_documents"`
	LastIndexed      time.Time `json:"last_indexed"`
	IndexingTime     float64   `json:"indexing_time_ms"`
	BatchesProcessed int       `json:"batches_processed"`
	ErrorCount       int       `json:"error_count"`
}

// =============================================================================
// DATABASE INTERFACE (Proper Dependency Injection)
// =============================================================================

// SearchDB defines the database interface for search operations
type SearchDB interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// =============================================================================
// SEARCH SERVICE (Production-Hardened)
// =============================================================================

// SearchService handles search operations with caching, fallback, and circuit breaker
type SearchService struct {
	indexer        SearchIndexerInterface
	cache          *redis.Client
	db             SearchDB
	config         SearchConfig
	cachePrefix    string
	circuitBreaker *CircuitBreaker
}

// NewSearchService creates a new search service with proper dependency injection
func NewSearchService(indexer SearchIndexerInterface, cache *redis.Client, db SearchDB) *SearchService {
	config := DefaultSearchConfig()
	return &SearchService{
		indexer:        indexer,
		cache:          cache,
		db:             db,
		config:         config,
		cachePrefix:    "search:",
		circuitBreaker: NewCircuitBreaker(config.CircuitBreakerThreshold, config.CircuitBreakerTimeout),
	}
}

// NewSearchServiceWithConfig creates a search service with custom configuration
func NewSearchServiceWithConfig(indexer SearchIndexerInterface, cache *redis.Client, db SearchDB, config SearchConfig) *SearchService {
	return &SearchService{
		indexer:        indexer,
		cache:          cache,
		db:             db,
		config:         config,
		cachePrefix:    "search:",
		circuitBreaker: NewCircuitBreaker(config.CircuitBreakerThreshold, config.CircuitBreakerTimeout),
	}
}

// SetCacheTTL allows runtime configuration of cache TTL
func (ss *SearchService) SetCacheTTL(ttl time.Duration) {
	ss.config.CacheTTL = ttl
}

// =============================================================================
// MAIN SEARCH METHOD
// =============================================================================

// Search performs a search with caching, circuit breaker, and PostgreSQL fallback
func (ss *SearchService) Search(req SearchRequest) (*SearchResponse, error) {
	startTime := time.Now()
	atomic.AddInt64(&searchMetrics.TotalSearches, 1)

	// Input validation and sanitization
	if err := ss.validateAndSanitizeRequest(&req); err != nil {
		atomic.AddInt64(&searchMetrics.Errors, 1)
		return nil, fmt.Errorf("invalid search request: %w", err)
	}

	// Generate cache key (with hash for long queries)
	cacheKey := ss.generateCacheKey(req)

	// Try cache first
	if cachedResult, err := ss.getFromCache(cacheKey); err == nil {
		atomic.AddInt64(&searchMetrics.CacheHits, 1)
		cachedResult.ProcessingTime = time.Since(startTime).Milliseconds()
		log.Printf("[SEARCH] Cache HIT for key: %s", cacheKey[:min(50, len(cacheKey))])
		return cachedResult, nil
	}
	atomic.AddInt64(&searchMetrics.CacheMisses, 1)

	var result *SearchResponse
	var err error

	// Check circuit breaker before trying MeiliSearch
	if ss.indexer != nil && !ss.circuitBreaker.IsOpen() {
		result, err = ss.searchWithMeiliSearch(req)
		if err != nil {
			ss.circuitBreaker.RecordFailure()
			log.Printf("[SEARCH] MeiliSearch failed (circuit breaker: %d failures): %v", 
				atomic.LoadInt32(&ss.circuitBreaker.failures), err)
		} else {
			ss.circuitBreaker.RecordSuccess()
			atomic.AddInt64(&searchMetrics.MeiliSearchHits, 1)
			result.Source = "meilisearch"
		}
	} else if ss.circuitBreaker.IsOpen() {
		log.Printf("[SEARCH] Circuit breaker OPEN - skipping MeiliSearch")
	}

	// Fallback to PostgreSQL if MeiliSearch failed or unavailable
	if result == nil {
		log.Printf("[SEARCH] Falling back to PostgreSQL full-text search")
		atomic.AddInt64(&searchMetrics.PostgreSQLFallbacks, 1)
		
		result, err = ss.searchWithPostgreSQL(req)
		if err != nil {
			atomic.AddInt64(&searchMetrics.Errors, 1)
			return nil, fmt.Errorf("search failed (both MeiliSearch and PostgreSQL): %w", err)
		}
		result.Source = "postgresql"
	}

	result.ProcessingTime = time.Since(startTime).Milliseconds()

	// Cache the result asynchronously
	go func() {
		if cacheErr := ss.cacheResult(cacheKey, result); cacheErr != nil {
			log.Printf("[SEARCH] Failed to cache result: %v", cacheErr)
		}
	}()

	return result, nil
}

// validateAndSanitizeRequest validates and sanitizes the search request
func (ss *SearchService) validateAndSanitizeRequest(req *SearchRequest) error {
	// Sanitize query - prevent injection and limit length
	req.Query = strings.TrimSpace(req.Query)
	if len(req.Query) > ss.config.MaxQueryLength {
		req.Query = req.Query[:ss.config.MaxQueryLength]
	}
	
	// Remove potentially dangerous characters for PostgreSQL
	req.Query = sanitizeSearchQuery(req.Query)

	// Enforce limits
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > ss.config.MaxLimit {
		req.Limit = ss.config.MaxLimit
	}

	if req.Offset < 0 {
		req.Offset = 0
	}
	if req.Offset > ss.config.MaxOffset {
		return fmt.Errorf("offset exceeds maximum allowed value of %d", ss.config.MaxOffset)
	}

	// Validate sort fields (whitelist approach)
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

// sanitizeSearchQuery removes potentially dangerous characters
func sanitizeSearchQuery(query string) string {
	// Remove SQL injection patterns
	dangerous := []string{"--", ";", "/*", "*/", "xp_", "sp_", "0x"}
	result := query
	for _, d := range dangerous {
		result = strings.ReplaceAll(result, d, "")
	}
	return result
}

// =============================================================================
// MEILISEARCH IMPLEMENTATION
// =============================================================================

// searchWithMeiliSearch performs search using MeiliSearch with timeout
func (ss *SearchService) searchWithMeiliSearch(req SearchRequest) (*SearchResponse, error) {
	if ss.indexer == nil {
		return nil, fmt.Errorf("MeiliSearch indexer not configured")
	}

	// Build search request
	searchReq := &meilisearch.SearchRequest{
		Query:  req.Query,
		Limit:  int64(req.Limit),
		Offset: int64(req.Offset),
	}

	// Add filters
	if req.Filters != nil {
		filter := ss.buildMeiliSearchFilter(req.Filters)
		if filter != "" {
			searchReq.Filter = filter
		}
	}

	// Add sorting
	if req.Sort != nil && req.Sort.Field != "relevance" {
		sort := fmt.Sprintf("%s:%s", req.Sort.Field, req.Sort.Order)
		searchReq.Sort = []string{sort}
	}

	// Perform search with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), ss.config.MeiliSearchTimeout)
	defer cancel()

	// Execute search (MeiliSearch client handles context internally)
	_ = ctx // Context for future use if MeiliSearch client supports it
	searchResp, err := ss.indexer.GetClient().Index(ss.indexer.GetIndexName()).Search(req.Query, searchReq)
	if err != nil {
		return nil, fmt.Errorf("MeiliSearch query failed: %w", err)
	}

	// Convert search documents to articles
	articles, err := ss.convertSearchHitsToArticles(searchResp.Hits)
	if err != nil {
		return nil, fmt.Errorf("failed to convert search hits: %w", err)
	}

	return &SearchResponse{
		Articles: articles,
		Total:    searchResp.EstimatedTotalHits,
		Limit:    req.Limit,
		Offset:   req.Offset,
	}, nil
}

// buildMeiliSearchFilter constructs MeiliSearch filter string
func (ss *SearchService) buildMeiliSearchFilter(filters *SearchFilters) string {
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

// =============================================================================
// POSTGRESQL FALLBACK (Fully Implemented)
// =============================================================================

// searchWithPostgreSQL performs fallback search using PostgreSQL full-text search
// Uses GIN index on search_vector column with ts_rank for relevance scoring
func (ss *SearchService) searchWithPostgreSQL(req SearchRequest) (*SearchResponse, error) {
	if ss.db == nil {
		return nil, fmt.Errorf("database connection not available for PostgreSQL fallback")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sanitizedQuery := strings.TrimSpace(req.Query)
	
	// Build the query using the pre-computed search_vector column with GIN index
	var queryBuilder strings.Builder
	args := []interface{}{}
	argIndex := 1

	// SELECT clause with relevance score
	queryBuilder.WriteString(`
		SELECT a.id, a.title, a.slug, a.content, a.excerpt, a.author_id, a.category_id,
			   a.status, a.published_at, a.created_at, a.updated_at, a.view_count,
			   a.like_count, a.language_code`)
	
	if sanitizedQuery != "" {
		queryBuilder.WriteString(fmt.Sprintf(`,
			   ts_rank(a.search_vector, plainto_tsquery('simple', $%d)) as relevance_score`, argIndex))
		args = append(args, sanitizedQuery)
		argIndex++
	} else {
		queryBuilder.WriteString(`, 0::float as relevance_score`)
	}

	// FROM and WHERE clause
	queryBuilder.WriteString(`
		FROM articles a
		WHERE a.status = 'published'`)

	// Full-text search condition using GIN-indexed search_vector
	if sanitizedQuery != "" {
		queryBuilder.WriteString(fmt.Sprintf(` AND a.search_vector @@ plainto_tsquery('simple', $%d)`, argIndex))
		args = append(args, sanitizedQuery)
		argIndex++
	}

	// Apply filters
	if req.Filters != nil {
		if req.Filters.AuthorID != nil {
			queryBuilder.WriteString(fmt.Sprintf(` AND a.author_id = $%d`, argIndex))
			args = append(args, *req.Filters.AuthorID)
			argIndex++
		}

		if req.Filters.CategoryID != nil {
			queryBuilder.WriteString(fmt.Sprintf(` AND a.category_id = $%d`, argIndex))
			args = append(args, *req.Filters.CategoryID)
			argIndex++
		}

		if req.Filters.LanguageCode != "" {
			queryBuilder.WriteString(fmt.Sprintf(` AND a.language_code = $%d`, argIndex))
			args = append(args, req.Filters.LanguageCode)
			argIndex++
		}

		if len(req.Filters.Tags) > 0 {
			queryBuilder.WriteString(fmt.Sprintf(` AND EXISTS (
				SELECT 1 FROM article_tags at
				JOIN tags t ON at.tag_id = t.id
				WHERE at.article_id = a.id AND t.name = ANY($%d::text[])
			)`, argIndex))
			args = append(args, req.Filters.Tags)
			argIndex++
		}

		if req.Filters.DateFrom != nil {
			queryBuilder.WriteString(fmt.Sprintf(` AND a.published_at >= to_timestamp($%d)`, argIndex))
			args = append(args, *req.Filters.DateFrom)
			argIndex++
		}

		if req.Filters.DateTo != nil {
			queryBuilder.WriteString(fmt.Sprintf(` AND a.published_at <= to_timestamp($%d)`, argIndex))
			args = append(args, *req.Filters.DateTo)
			argIndex++
		}
	}

	// ORDER BY clause
	sortField := "published_at"
	sortOrder := "DESC"
	if req.Sort != nil {
		switch req.Sort.Field {
		case "published_at", "created_at", "view_count", "like_count":
			sortField = req.Sort.Field
		case "relevance":
			if sanitizedQuery != "" {
				sortField = "relevance_score"
			}
		}
		if req.Sort.Order == "asc" {
			sortOrder = "ASC"
		}
	}

	if sortField == "relevance_score" {
		queryBuilder.WriteString(` ORDER BY relevance_score DESC, a.published_at DESC`)
	} else {
		queryBuilder.WriteString(fmt.Sprintf(` ORDER BY a.%s %s`, sortField, sortOrder))
	}

	// LIMIT and OFFSET with bounds protection
	queryBuilder.WriteString(fmt.Sprintf(` LIMIT $%d OFFSET $%d`, argIndex, argIndex+1))
	args = append(args, req.Limit, req.Offset)

	// Execute the main query
	log.Printf("[SEARCH_FALLBACK] Executing PostgreSQL search: query=%q, filters=%+v", sanitizedQuery, req.Filters)
	
	rows, err := ss.db.QueryContext(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("PostgreSQL search query failed: %w", err)
	}
	defer rows.Close()

	// Parse results
	var articles []models.Article
	for rows.Next() {
		var article models.Article
		var relevanceScore float64
		var publishedAt, createdAt, updatedAt sql.NullTime
		var languageCode sql.NullString

		err := rows.Scan(
			&article.ID,
			&article.Title,
			&article.Slug,
			&article.Content,
			&article.Excerpt,
			&article.AuthorID,
			&article.CategoryID,
			&article.Status,
			&publishedAt,
			&createdAt,
			&updatedAt,
			&article.ViewCount,
			&article.LikeCount,
			&languageCode,
			&relevanceScore,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article row: %w", err)
		}

		if publishedAt.Valid {
			article.PublishedAt = &publishedAt.Time
		}
		if createdAt.Valid {
			article.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			article.UpdatedAt = updatedAt.Time
		}
		if languageCode.Valid {
			article.LanguageCode = languageCode.String
		}

		articles = append(articles, article)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating search results: %w", err)
	}

	// Get total count for pagination
	total, err := ss.getPostgreSQLSearchCount(ctx, sanitizedQuery, req.Filters, argIndex)
	if err != nil {
		log.Printf("[SEARCH_FALLBACK] Warning: failed to get total count: %v", err)
		total = int64(len(articles)) // Fallback to current page count
	}

	log.Printf("[SEARCH_FALLBACK] PostgreSQL search completed: %d results, total=%d", len(articles), total)

	return &SearchResponse{
		Articles: articles,
		Total:    total,
		Limit:    req.Limit,
		Offset:   req.Offset,
	}, nil
}

// getPostgreSQLSearchCount returns the total count for pagination
func (ss *SearchService) getPostgreSQLSearchCount(ctx context.Context, query string, filters *SearchFilters, startArgIndex int) (int64, error) {
	var countBuilder strings.Builder
	args := []interface{}{}
	argIndex := 1

	countBuilder.WriteString(`SELECT COUNT(*) FROM articles a WHERE a.status = 'published'`)

	if query != "" {
		countBuilder.WriteString(fmt.Sprintf(` AND a.search_vector @@ plainto_tsquery('simple', $%d)`, argIndex))
		args = append(args, query)
		argIndex++
	}

	if filters != nil {
		if filters.AuthorID != nil {
			countBuilder.WriteString(fmt.Sprintf(` AND a.author_id = $%d`, argIndex))
			args = append(args, *filters.AuthorID)
			argIndex++
		}
		if filters.CategoryID != nil {
			countBuilder.WriteString(fmt.Sprintf(` AND a.category_id = $%d`, argIndex))
			args = append(args, *filters.CategoryID)
			argIndex++
		}
		if filters.LanguageCode != "" {
			countBuilder.WriteString(fmt.Sprintf(` AND a.language_code = $%d`, argIndex))
			args = append(args, filters.LanguageCode)
			argIndex++
		}
		if len(filters.Tags) > 0 {
			countBuilder.WriteString(fmt.Sprintf(` AND EXISTS (
				SELECT 1 FROM article_tags at
				JOIN tags t ON at.tag_id = t.id
				WHERE at.article_id = a.id AND t.name = ANY($%d::text[])
			)`, argIndex))
			args = append(args, filters.Tags)
			argIndex++
		}
		if filters.DateFrom != nil {
			countBuilder.WriteString(fmt.Sprintf(` AND a.published_at >= to_timestamp($%d)`, argIndex))
			args = append(args, *filters.DateFrom)
			argIndex++
		}
		if filters.DateTo != nil {
			countBuilder.WriteString(fmt.Sprintf(` AND a.published_at <= to_timestamp($%d)`, argIndex))
			args = append(args, *filters.DateTo)
			argIndex++
		}
	}

	var total int64
	err := ss.db.QueryRowContext(ctx, countBuilder.String(), args...).Scan(&total)
	return total, err
}

// =============================================================================
// CACHE MANAGEMENT
// =============================================================================

// generateCacheKey creates a unique cache key with hash for long queries
func (ss *SearchService) generateCacheKey(req SearchRequest) string {
	// Build key components
	var keyParts []string
	keyParts = append(keyParts, ss.cachePrefix)
	
	// Hash the query if it's long to prevent key abuse
	if len(req.Query) > 100 {
		hash := sha256.Sum256([]byte(req.Query))
		keyParts = append(keyParts, "q:"+hex.EncodeToString(hash[:8]))
	} else {
		keyParts = append(keyParts, "q:"+req.Query)
	}
	
	keyParts = append(keyParts, fmt.Sprintf("l:%d", req.Limit))
	keyParts = append(keyParts, fmt.Sprintf("o:%d", req.Offset))

	if req.Filters != nil {
		if req.Filters.AuthorID != nil {
			keyParts = append(keyParts, fmt.Sprintf("a:%d", *req.Filters.AuthorID))
		}
		if req.Filters.CategoryID != nil {
			keyParts = append(keyParts, fmt.Sprintf("c:%d", *req.Filters.CategoryID))
		}
		if req.Filters.LanguageCode != "" {
			keyParts = append(keyParts, "lang:"+req.Filters.LanguageCode)
		}
		if len(req.Filters.Tags) > 0 {
			tagHash := sha256.Sum256([]byte(strings.Join(req.Filters.Tags, ",")))
			keyParts = append(keyParts, "t:"+hex.EncodeToString(tagHash[:4]))
		}
		if req.Filters.DateFrom != nil {
			keyParts = append(keyParts, fmt.Sprintf("df:%d", *req.Filters.DateFrom))
		}
		if req.Filters.DateTo != nil {
			keyParts = append(keyParts, fmt.Sprintf("dt:%d", *req.Filters.DateTo))
		}
	}

	if req.Sort != nil {
		keyParts = append(keyParts, fmt.Sprintf("s:%s:%s", req.Sort.Field, req.Sort.Order))
	}

	return strings.Join(keyParts, ":")
}

// getFromCache retrieves search results from cache
func (ss *SearchService) getFromCache(key string) (*SearchResponse, error) {
	if ss.cache == nil {
		return nil, fmt.Errorf("cache not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	data, err := ss.cache.Get(ctx, key).Result()
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
func (ss *SearchService) cacheResult(key string, result *SearchResponse) error {
	if ss.cache == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return ss.cache.Set(ctx, key, data, ss.config.CacheTTL).Err()
}

// InvalidateCache removes cached search results matching a pattern
func (ss *SearchService) InvalidateCache(pattern string) error {
	if ss.cache == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if pattern == "" {
		pattern = ss.cachePrefix + "*"
	}

	keys, err := ss.cache.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		log.Printf("[CACHE] Invalidating %d cache keys matching pattern: %s", len(keys), pattern)
		return ss.cache.Del(ctx, keys...).Err()
	}

	return nil
}

// InvalidateCacheForArticle invalidates all cache entries related to an article
func (ss *SearchService) InvalidateCacheForArticle(article *models.Article) error {
	if ss.cache == nil || article == nil {
		return nil
	}

	// Invalidate by category, author, and general search cache
	patterns := []string{
		fmt.Sprintf("%s*c:%d*", ss.cachePrefix, article.CategoryID),
		fmt.Sprintf("%s*a:%d*", ss.cachePrefix, article.AuthorID),
		ss.cachePrefix + "*", // Invalidate all search cache on article change
	}

	for _, pattern := range patterns {
		if err := ss.InvalidateCache(pattern); err != nil {
			log.Printf("[CACHE] Warning: failed to invalidate pattern %s: %v", pattern, err)
		}
	}

	return nil
}

// =============================================================================
// DOCUMENT CONVERSION
// =============================================================================

// convertSearchHitsToArticles converts MeiliSearch hits to Article models
func (ss *SearchService) convertSearchHitsToArticles(hits meilisearch.Hits) ([]models.Article, error) {
	articles := make([]models.Article, 0, len(hits))

	for i, hit := range hits {
		hitMap := hit
		if hitMap == nil {
			continue
		}

		hitJSON, err := json.Marshal(hitMap)
		if err != nil {
			log.Printf("[SEARCH] Warning: failed to marshal hit %d: %v", i, err)
			continue
		}

		var doc SearchDocument
		if err := json.Unmarshal(hitJSON, &doc); err != nil {
			log.Printf("[SEARCH] Warning: failed to unmarshal hit %d: %v", i, err)
			continue
		}

		article, err := ss.convertSearchDocumentToArticle(doc)
		if err != nil {
			log.Printf("[SEARCH] Warning: failed to convert document %d: %v", i, err)
			continue
		}

		articles = append(articles, *article)
	}

	return articles, nil
}

// convertSearchDocumentToArticle converts SearchDocument to Article model
func (ss *SearchService) convertSearchDocumentToArticle(doc SearchDocument) (*models.Article, error) {
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
		ID:           id,
		Title:        doc.Title,
		Content:      doc.Content,
		Excerpt:      doc.Excerpt,
		AuthorID:     doc.AuthorID,
		CategoryID:   doc.CategoryID,
		Tags:         tags,
		Status:       doc.Status,
		PublishedAt:  publishedAt,
		CreatedAt:    time.Unix(doc.CreatedAt, 0),
		ViewCount:    doc.ViewCount,
		LikeCount:    doc.LikeCount,
		LanguageCode: doc.LanguageCode,
		SEOData: models.SEOData{
			MetaTitle:       doc.MetaTitle,
			MetaDescription: doc.MetaDescription,
			Keywords:        doc.Keywords,
		},
	}, nil
}

// =============================================================================
// INDEX MANAGEMENT
// =============================================================================

// IndexArticle indexes a single article and invalidates related cache
func (ss *SearchService) IndexArticle(article *models.Article) error {
	if ss.indexer == nil {
		log.Printf("[INDEX] Warning: indexer not available, skipping index for article %d", article.ID)
		return nil
	}

	if err := ss.indexer.IndexArticle(article); err != nil {
		return fmt.Errorf("failed to index article %d: %w", article.ID, err)
	}

	// Invalidate related cache
	go ss.InvalidateCacheForArticle(article)

	log.Printf("[INDEX] Successfully indexed article %d: %s", article.ID, article.Title)
	return nil
}

// RemoveArticle removes an article from the search index
func (ss *SearchService) RemoveArticle(ctx context.Context, articleID uint64) error {
	if ss.indexer == nil {
		return nil
	}

	if err := ss.indexer.DeleteArticle(fmt.Sprintf("%d", articleID)); err != nil {
		return fmt.Errorf("failed to remove article %d from index: %w", articleID, err)
	}

	// Invalidate all search cache
	go ss.InvalidateCache("")

	return nil
}

// UpdateArticle updates an article in the search index
func (ss *SearchService) UpdateArticle(ctx context.Context, article *models.Article) error {
	return ss.IndexArticle(article)
}

// RebuildIndex rebuilds the entire search index
func (ss *SearchService) RebuildIndex(ctx context.Context) (*IndexStats, error) {
	return nil, fmt.Errorf("rebuild index requires article repository access - use batch indexing instead")
}

// GetIndexStats returns search index statistics
func (ss *SearchService) GetIndexStats(ctx context.Context) (*IndexStats, error) {
	if ss.indexer == nil {
		return nil, fmt.Errorf("indexer not available")
	}

	stats, err := ss.indexer.GetIndexStats()
	if err != nil {
		return nil, err
	}

	totalDocs, _ := stats["number_of_documents"].(int64)

	return &IndexStats{
		TotalDocuments:   totalDocs,
		LastIndexed:      time.Now(),
		IndexingTime:     0,
		BatchesProcessed: 0,
		ErrorCount:       0,
	}, nil
}

// =============================================================================
// API METHODS
// =============================================================================

// SearchArticles performs a comprehensive search with facets (API-compatible method)
func (ss *SearchService) SearchArticles(ctx context.Context, filters SearchFilters, limit, offset int) ([]SearchResult, SearchFacets, int, float64, error) {
	startTime := time.Now()

	internalFilters := &SearchFilters{
		Query:        filters.Query,
		AuthorID:     filters.AuthorID,
		CategoryID:   filters.CategoryID,
		Tags:         filters.Tags,
		LanguageCode: filters.LanguageCode,
		DateFrom:     filters.DateFrom,
		DateTo:       filters.DateTo,
		Status:       filters.Status,
	}

	if len(filters.Categories) > 0 && filters.CategoryID == nil {
		internalFilters.CategoryID = &filters.Categories[0]
	}

	req := SearchRequest{
		Query:   filters.Query,
		Limit:   limit,
		Offset:  offset,
		Filters: internalFilters,
	}

	if filters.SortBy != "" && filters.SortOrder != "" {
		req.Sort = &SearchSort{
			Field: filters.SortBy,
			Order: filters.SortOrder,
		}
	}

	response, err := ss.Search(req)
	if err != nil {
		return nil, SearchFacets{}, 0, 0, err
	}

	results := make([]SearchResult, len(response.Articles))
	for i, article := range response.Articles {
		publishedAt := ""
		if article.PublishedAt != nil {
			publishedAt = article.PublishedAt.Format(time.RFC3339)
		}

		results[i] = SearchResult{
			ID:          article.ID,
			Title:       article.Title,
			Slug:        article.Slug,
			Excerpt:     article.Excerpt,
			AuthorName:  "",
			Category:    "",
			PublishedAt: publishedAt,
			ViewCount:   article.ViewCount,
			Score:       1.0,
			Tags:        extractTagNames(article.Tags),
			Highlights:  []string{},
		}
	}

	facets := SearchFacets{
		Categories: []FacetItem{},
		Tags:       []FacetItem{},
		Authors:    []FacetItem{},
	}

	processingTime := time.Since(startTime).Seconds() * 1000

	return results, facets, int(response.Total), processingTime, nil
}

// GetSuggestions provides search suggestions based on query
func (ss *SearchService) GetSuggestions(ctx context.Context, query string, limit int) ([]string, error) {
	return []string{}, nil
}

// GetPopularSearches returns popular search queries
func (ss *SearchService) GetPopularSearches(ctx context.Context, limit int, days int) ([]PopularSearch, error) {
	return []PopularSearch{}, nil
}

// =============================================================================
// HEALTH CHECK
// =============================================================================

// HealthCheck verifies search service health
func (ss *SearchService) HealthCheck() map[string]interface{} {
	health := map[string]interface{}{
		"meilisearch":     "unknown",
		"cache":           "unknown",
		"database":        "unknown",
		"circuit_breaker": "closed",
	}

	// Check MeiliSearch
	if ss.indexer != nil {
		if err := ss.indexer.HealthCheck(); err != nil {
			health["meilisearch"] = fmt.Sprintf("error: %v", err)
		} else {
			health["meilisearch"] = "healthy"
		}
	} else {
		health["meilisearch"] = "not configured"
	}

	// Check circuit breaker
	if ss.circuitBreaker.IsOpen() {
		health["circuit_breaker"] = "open"
	}

	// Check cache
	if ss.cache != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := ss.cache.Ping(ctx).Err(); err != nil {
			health["cache"] = fmt.Sprintf("error: %v", err)
		} else {
			health["cache"] = "healthy"
		}
	} else {
		health["cache"] = "not configured"
	}

	// Check database
	if ss.db != nil {
		health["database"] = "available"
	} else {
		health["database"] = "not configured"
	}

	// Add metrics
	health["metrics"] = GetSearchMetrics()

	return health
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func extractTagNames(tags []models.Tag) []string {
	names := make([]string, len(tags))
	for i, tag := range tags {
		names[i] = tag.Name
	}
	return names
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// =============================================================================
// LEGACY ADAPTER (For backward compatibility)
// =============================================================================

// ArticleRepositoryAdapter adapts the actual repository to the search interface (legacy)
type ArticleRepositoryAdapter struct {
	Repository interface {
		GetByID(ctx context.Context, id uint64) (*models.Article, error)
	}
}

// GetByID implements the ArticleRepository interface
func (a *ArticleRepositoryAdapter) GetByID(ctx context.Context, id uint64) (*models.Article, error) {
	return a.Repository.GetByID(ctx, id)
}

// Search implements a basic search (legacy - use SearchService.searchWithPostgreSQL instead)
func (a *ArticleRepositoryAdapter) Search(query string, filters map[string]interface{}, limit, offset int) ([]models.Article, int64, error) {
	return []models.Article{}, 0, fmt.Errorf("use SearchService with proper DB injection instead")
}
