package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"high-performance-news-website/internal/models"

	meilisearch "github.com/meilisearch/meilisearch-go"
	"github.com/redis/go-redis/v9"
)

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
	Order string `json:"order"` // "asc" or "desc"
}

// SearchRequest represents a complete search request
type SearchRequest struct {
	Query   string         `json:"query"`
	Filters *SearchFilters `json:"filters,omitempty"`
	Sort    *SearchSort    `json:"sort,omitempty"`
	Limit   int            `json:"limit"`
	Offset  int            `json:"offset"`
}

// SearchResponse represents search results
type SearchResponse struct {
	Articles      []models.Article `json:"articles"`
	Total         int64            `json:"total"`
	Limit         int              `json:"limit"`
	Offset        int              `json:"offset"`
	ProcessingTime int64           `json:"processing_time_ms"`
	Source        string           `json:"source"` // "meilisearch" or "postgresql"
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

// SearchService handles search operations with caching and fallback
type SearchService struct {
	indexer    SearchIndexerInterface
	cache      *redis.Client
	fallbackDB ArticleRepository
	cachePrefix string
	cacheTTL   time.Duration
	timeout    time.Duration
}

// NewSearchService creates a new search service instance
func NewSearchService(indexer SearchIndexerInterface, cache *redis.Client, fallbackDB ArticleRepository) *SearchService {
	return &SearchService{
		indexer:     indexer,
		cache:       cache,
		fallbackDB:  fallbackDB,
		cachePrefix: "search:",
		cacheTTL:    5 * time.Minute, // Cache search results for 5 minutes
		timeout:     10 * time.Second,
	}
}

// Search performs a search with caching and fallback to PostgreSQL
func (ss *SearchService) Search(req SearchRequest) (*SearchResponse, error) {
	startTime := time.Now()

	// Generate cache key
	cacheKey := ss.generateCacheKey(req)

	// Try to get from cache first
	if cachedResult, err := ss.getFromCache(cacheKey); err == nil {
		cachedResult.ProcessingTime = time.Since(startTime).Milliseconds()
		return cachedResult, nil
	}

	// Try MeiliSearch first
	result, err := ss.searchWithMeiliSearch(req)
	if err != nil {
		log.Printf("MeiliSearch failed, falling back to PostgreSQL: %v", err)
		// Fallback to PostgreSQL
		result, err = ss.searchWithPostgreSQL(req)
		if err != nil {
			return nil, fmt.Errorf("both MeiliSearch and PostgreSQL search failed: %w", err)
		}
		result.Source = "postgresql"
	} else {
		result.Source = "meilisearch"
	}

	result.ProcessingTime = time.Since(startTime).Milliseconds()

	// Cache the result
	if err := ss.cacheResult(cacheKey, result); err != nil {
		log.Printf("Failed to cache search result: %v", err)
	}

	return result, nil
}

// searchWithMeiliSearch performs search using MeiliSearch
func (ss *SearchService) searchWithMeiliSearch(req SearchRequest) (*SearchResponse, error) {
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
	if req.Sort != nil {
		sort := fmt.Sprintf("%s:%s", req.Sort.Field, req.Sort.Order)
		searchReq.Sort = []string{sort}
	}

	// Perform search
	searchResp, err := ss.indexer.GetClient().Index(ss.indexer.GetIndexName()).Search(req.Query, searchReq)
	if err != nil {
		return nil, fmt.Errorf("MeiliSearch query failed: %w", err)
	}

	// Convert search documents back to articles
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

// searchWithPostgreSQL performs fallback search using PostgreSQL
func (ss *SearchService) searchWithPostgreSQL(req SearchRequest) (*SearchResponse, error) {
	// Convert filters to map for repository
	filters := make(map[string]interface{})
	
	if req.Filters != nil {
		if req.Filters.AuthorID != nil {
			filters["author_id"] = *req.Filters.AuthorID
		}
		if req.Filters.CategoryID != nil {
			filters["category_id"] = *req.Filters.CategoryID
		}
		if req.Filters.LanguageCode != "" {
			filters["language_code"] = req.Filters.LanguageCode
		}
		if req.Filters.Status != "" {
			filters["status"] = req.Filters.Status
		}
		if len(req.Filters.Tags) > 0 {
			filters["tags"] = req.Filters.Tags
		}
		if req.Filters.DateFrom != nil {
			filters["date_from"] = *req.Filters.DateFrom
		}
		if req.Filters.DateTo != nil {
			filters["date_to"] = *req.Filters.DateTo
		}
	}

	// Add sorting to filters
	if req.Sort != nil {
		filters["sort_field"] = req.Sort.Field
		filters["sort_order"] = req.Sort.Order
	}

	articles, total, err := ss.fallbackDB.Search(req.Query, filters, req.Limit, req.Offset)
	if err != nil {
		return nil, fmt.Errorf("PostgreSQL search failed: %w", err)
	}

	return &SearchResponse{
		Articles: articles,
		Total:    total,
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

// convertSearchHitsToArticles converts MeiliSearch hits to Article models
func (ss *SearchService) convertSearchHitsToArticles(hits meilisearch.Hits) ([]models.Article, error) {
	articles := make([]models.Article, len(hits))

	for i, hit := range hits {
		// MeiliSearch Hit is already a map[string]interface{}
		hitMap := hit
		if hitMap == nil {
			return nil, fmt.Errorf("invalid hit format at index %d", i)
		}

		// Convert hit to SearchDocument first
		hitJSON, err := json.Marshal(hitMap)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal hit %d: %w", i, err)
		}

		var doc SearchDocument
		if err := json.Unmarshal(hitJSON, &doc); err != nil {
			return nil, fmt.Errorf("failed to unmarshal hit %d: %w", i, err)
		}

		// Convert SearchDocument to Article
		article, err := ss.convertSearchDocumentToArticle(doc)
		if err != nil {
			return nil, fmt.Errorf("failed to convert document %d: %w", i, err)
		}

		articles[i] = *article
	}

	return articles, nil
}

// convertSearchDocumentToArticle converts SearchDocument back to Article model
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

	// Convert tag names to Tag models
	tags := make([]models.Tag, len(doc.Tags))
	for i, tagName := range doc.Tags {
		tags[i] = models.Tag{Name: tagName}
	}

	return &models.Article{
		ID:          id,
		Title:       doc.Title,
		Content:     doc.Content,
		Excerpt:     doc.Excerpt,
		AuthorID:    doc.AuthorID,
		CategoryID:  doc.CategoryID,
		Tags:        tags,
		Status:      doc.Status,
		PublishedAt: publishedAt,
		CreatedAt:   time.Unix(doc.CreatedAt, 0),
		ViewCount:   doc.ViewCount,
		LikeCount:   doc.LikeCount,
		LanguageCode: doc.LanguageCode,
		SEOData: models.SEOData{
			MetaTitle:       doc.MetaTitle,
			MetaDescription: doc.MetaDescription,
			Keywords:        doc.Keywords,
		},
	}, nil
}

// generateCacheKey creates a unique cache key for the search request
func (ss *SearchService) generateCacheKey(req SearchRequest) string {
	// Create a deterministic key based on search parameters
	key := fmt.Sprintf("%s%s:%d:%d", ss.cachePrefix, req.Query, req.Limit, req.Offset)

	if req.Filters != nil {
		if req.Filters.AuthorID != nil {
			key += fmt.Sprintf(":author_%d", *req.Filters.AuthorID)
		}
		if req.Filters.CategoryID != nil {
			key += fmt.Sprintf(":cat_%d", *req.Filters.CategoryID)
		}
		if req.Filters.LanguageCode != "" {
			key += fmt.Sprintf(":lang_%s", req.Filters.LanguageCode)
		}
		if len(req.Filters.Tags) > 0 {
			key += fmt.Sprintf(":tags_%s", strings.Join(req.Filters.Tags, ","))
		}
		if req.Filters.DateFrom != nil {
			key += fmt.Sprintf(":from_%d", *req.Filters.DateFrom)
		}
		if req.Filters.DateTo != nil {
			key += fmt.Sprintf(":to_%d", *req.Filters.DateTo)
		}
	}

	if req.Sort != nil {
		key += fmt.Sprintf(":sort_%s_%s", req.Sort.Field, req.Sort.Order)
	}

	return key
}

// getFromCache retrieves search results from cache
func (ss *SearchService) getFromCache(key string) (*SearchResponse, error) {
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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return ss.cache.Set(ctx, key, data, ss.cacheTTL).Err()
}

// InvalidateCache removes cached search results
func (ss *SearchService) InvalidateCache(pattern string) error {
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
		return ss.cache.Del(ctx, keys...).Err()
	}

	return nil
}

// SearchArticles performs a comprehensive search with facets (API-compatible method)
func (ss *SearchService) SearchArticles(ctx context.Context, filters SearchFilters, limit, offset int) ([]SearchResult, SearchFacets, int, float64, error) {
	startTime := time.Now()

	// Convert to internal SearchRequest format
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

	// Handle multiple categories
	if len(filters.Categories) > 0 && filters.CategoryID == nil {
		// Use the first category for now (could be enhanced to support multiple)
		internalFilters.CategoryID = &filters.Categories[0]
	}

	req := SearchRequest{
		Query:   filters.Query,
		Limit:   limit,
		Offset:  offset,
		Filters: internalFilters,
	}

	// Add sorting
	if filters.SortBy != "" && filters.SortOrder != "" {
		req.Sort = &SearchSort{
			Field: filters.SortBy,
			Order: filters.SortOrder,
		}
	}

	// Perform search using existing method
	response, err := ss.Search(req)
	if err != nil {
		return nil, SearchFacets{}, 0, 0, err
	}

	// Convert articles to SearchResult format
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
			AuthorName:  "", // TODO: Populate from user lookup
			Category:    "", // TODO: Populate from category lookup
			PublishedAt: publishedAt,
			ViewCount:   article.ViewCount,
			Score:       1.0, // TODO: Get actual relevance score from MeiliSearch
			Tags:        extractTagNames(article.Tags),
			Highlights:  []string{}, // TODO: Get highlights from MeiliSearch
		}
	}

	// Generate facets (simplified implementation)
	facets := SearchFacets{
		Categories: []FacetItem{},
		Tags:       []FacetItem{},
		Authors:    []FacetItem{},
	}

	processingTime := time.Since(startTime).Seconds() * 1000 // Convert to milliseconds

	return results, facets, int(response.Total), processingTime, nil
}

// GetSuggestions provides search suggestions based on query
func (ss *SearchService) GetSuggestions(ctx context.Context, query string, limit int) ([]string, error) {
	// For now, return empty suggestions - this could be enhanced with
	// a dedicated suggestions index or autocomplete functionality
	return []string{}, nil
}

// GetPopularSearches returns popular search queries
func (ss *SearchService) GetPopularSearches(ctx context.Context, limit int, days int) ([]PopularSearch, error) {
	// For now, return empty popular searches - this could be enhanced with
	// search analytics tracking
	return []PopularSearch{}, nil
}

// RemoveArticle removes an article from the search index
func (ss *SearchService) RemoveArticle(ctx context.Context, articleID uint64) error {
	return ss.indexer.DeleteArticle(fmt.Sprintf("%d", articleID))
}

// UpdateArticle updates an article in the search index
func (ss *SearchService) UpdateArticle(ctx context.Context, article *models.Article) error {
	return ss.indexer.IndexArticle(article)
}

// RebuildIndex rebuilds the entire search index
func (ss *SearchService) RebuildIndex(ctx context.Context) (*IndexStats, error) {
	// This would need to be implemented with access to the article repository
	// For now, return an error indicating this needs to be implemented
	return nil, fmt.Errorf("rebuild index not implemented - requires article repository access")
}

// GetIndexStats returns search index statistics
func (ss *SearchService) GetIndexStats(ctx context.Context) (*IndexStats, error) {
	stats, err := ss.indexer.GetIndexStats()
	if err != nil {
		return nil, err
	}

	// Convert to IndexStats format
	totalDocs, ok := stats["number_of_documents"].(int64)
	if !ok {
		totalDocs = 0
	}

	return &IndexStats{
		TotalDocuments:   totalDocs,
		LastIndexed:      time.Now(), // TODO: Track actual last indexed time
		IndexingTime:     0,          // TODO: Track indexing time
		BatchesProcessed: 0,          // TODO: Track batches processed
		ErrorCount:       0,          // TODO: Track errors
	}, nil
}

// extractTagNames extracts tag names from Tag models
func extractTagNames(tags []models.Tag) []string {
	names := make([]string, len(tags))
	for i, tag := range tags {
		names[i] = tag.Name
	}
	return names
}

// ArticleRepositoryAdapter adapts the actual repository to the search interface
type ArticleRepositoryAdapter struct {
	Repository interface {
		GetByID(ctx context.Context, id uint64) (*models.Article, error)
	}
}

// GetByID implements the ArticleRepository interface
func (a *ArticleRepositoryAdapter) GetByID(ctx context.Context, id uint64) (*models.Article, error) {
	return a.Repository.GetByID(ctx, id)
}

// Search implements a basic search using PostgreSQL (fallback implementation)
func (a *ArticleRepositoryAdapter) Search(query string, filters map[string]interface{}, limit, offset int) ([]models.Article, int64, error) {
	// This is a simplified fallback implementation
	// In a real implementation, this would use PostgreSQL full-text search
	return []models.Article{}, 0, fmt.Errorf("PostgreSQL search fallback not fully implemented")
}

// IndexArticle indexes a single article and invalidates related cache
func (ss *SearchService) IndexArticle(article *models.Article) error {
	if err := ss.indexer.IndexArticle(article); err != nil {
		return err
	}

	// Invalidate cache for this article's category and tags
	patterns := []string{
		fmt.Sprintf("%s*:cat_%d*", ss.cachePrefix, article.CategoryID),
		fmt.Sprintf("%s*:author_%d*", ss.cachePrefix, article.AuthorID),
	}

	for _, pattern := range patterns {
		if err := ss.InvalidateCache(pattern); err != nil {
			log.Printf("Failed to invalidate cache pattern %s: %v", pattern, err)
		}
	}

	return nil
}

// HealthCheck verifies search service health
func (ss *SearchService) HealthCheck() map[string]interface{} {
	health := map[string]interface{}{
		"meilisearch": "unknown",
		"cache":       "unknown",
		"fallback_db": "unknown",
	}

	// Check MeiliSearch
	if err := ss.indexer.HealthCheck(); err != nil {
		health["meilisearch"] = fmt.Sprintf("error: %v", err)
	} else {
		health["meilisearch"] = "healthy"
	}

	// Check cache
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	if err := ss.cache.Ping(ctx).Err(); err != nil {
		health["cache"] = fmt.Sprintf("error: %v", err)
	} else {
		health["cache"] = "healthy"
	}

	// Check fallback DB (basic check)
	if ss.fallbackDB != nil {
		health["fallback_db"] = "available"
	} else {
		health["fallback_db"] = "not configured"
	}

	return health
}