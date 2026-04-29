package services

import (
	"context"
	"fmt"
	"time"

	"high-performance-news-website/internal/models"
)

// SearchServiceInterface defines the interface for search services
// Both SearchService and EnterpriseSearchService implement this interface
type SearchServiceInterface interface {
	// Core search method
	Search(req SearchRequest) (*SearchResponse, error)

	// Suggestions
	GetSuggestions(ctx context.Context, query string, limit int) ([]string, error)

	// Health check
	HealthCheck() map[string]interface{}

	// Cache management
	InvalidateCache(pattern string) error
	InvalidateCacheForArticle(article *models.Article) error

	// Indexing operations
	IndexArticle(article *models.Article) error
	RemoveArticle(ctx context.Context, articleID uint64) error
	UpdateArticle(ctx context.Context, article *models.Article) error
	RebuildIndex(ctx context.Context) (*IndexStats, error)
	GetIndexStats(ctx context.Context) (*IndexStats, error)
}

// EnterpriseSearchServiceWrapper wraps EnterpriseSearchService to implement SearchServiceInterface
type EnterpriseSearchServiceWrapper struct {
	*EnterpriseSearchService
}

// NewEnterpriseSearchServiceWrapper creates a wrapper that implements SearchServiceInterface
func NewEnterpriseSearchServiceWrapper(ess *EnterpriseSearchService) *EnterpriseSearchServiceWrapper {
	return &EnterpriseSearchServiceWrapper{EnterpriseSearchService: ess}
}

// Search implements SearchServiceInterface.Search without context parameter
func (w *EnterpriseSearchServiceWrapper) Search(req SearchRequest) (*SearchResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 
		w.config.MeiliSearchTimeout+w.config.PostgreSQLTimeout)
	defer cancel()
	return w.EnterpriseSearchService.Search(ctx, req)
}

// HealthCheck implements SearchServiceInterface.HealthCheck
func (w *EnterpriseSearchServiceWrapper) HealthCheck() map[string]interface{} {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	status := w.EnterpriseSearchService.HealthCheck(ctx)
	
	// Convert EnhancedHealthStatus to map[string]interface{}
	return map[string]interface{}{
		"status":           status.Status,
		"timestamp":        status.Timestamp,
		"meilisearch":      status.Components["meilisearch"].Status,
		"cache":            status.Components["cache"].Status,
		"database":         status.Components["database"].Status,
		"circuit_breaker":  status.Components["circuit_breaker"].Status,
		"meilisearch_hits": status.Metrics["meilisearch_hits"],
		"cache_hits":       status.Metrics["cache_hits"],
		"cache_misses":     status.Metrics["cache_misses"],
		"p95_latency_ms":   status.Metrics["p95_latency_ms"],
		"p99_latency_ms":   status.Metrics["p99_latency_ms"],
	}
}

// InvalidateCache implements SearchServiceInterface.InvalidateCache
func (w *EnterpriseSearchServiceWrapper) InvalidateCache(pattern string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return w.EnterpriseSearchService.InvalidateCache(ctx, pattern)
}

// InvalidateCacheForArticle implements SearchServiceInterface.InvalidateCacheForArticle
func (w *EnterpriseSearchServiceWrapper) InvalidateCacheForArticle(article *models.Article) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return w.EnterpriseSearchService.InvalidateCache(ctx, "")
}

// RebuildIndex implements SearchServiceInterface.RebuildIndex
func (w *EnterpriseSearchServiceWrapper) RebuildIndex(ctx context.Context) (*IndexStats, error) {
	// Get all published articles from database
	if w.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	rows, err := w.db.QueryContext(ctx, `
		SELECT id, title, content, excerpt, author_id, category_id, status, 
		       published_at, created_at, view_count, like_count, language_code
		FROM articles 
		WHERE status = 'published'
		ORDER BY published_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch articles: %w", err)
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		var a models.Article
		var publishedAt, createdAt *time.Time
		err := rows.Scan(&a.ID, &a.Title, &a.Content, &a.Excerpt, &a.AuthorID, 
			&a.CategoryID, &a.Status, &publishedAt, &createdAt, &a.ViewCount, 
			&a.LikeCount, &a.LanguageCode)
		if err != nil {
			continue
		}
		a.PublishedAt = publishedAt
		if createdAt != nil {
			a.CreatedAt = *createdAt
		}
		articles = append(articles, a)
	}

	// Rebuild index using enterprise service
	err = w.EnterpriseSearchService.RebuildIndex(ctx, articles, "system", "127.0.0.1")
	if err != nil {
		return nil, err
	}

	return &IndexStats{
		TotalDocuments: int64(len(articles)),
		LastIndexed:    time.Now(),
	}, nil
}

// GetIndexStats implements SearchServiceInterface.GetIndexStats
func (w *EnterpriseSearchServiceWrapper) GetIndexStats(ctx context.Context) (*IndexStats, error) {
	if w.indexer == nil {
		return &IndexStats{
			TotalDocuments: 0,
			LastIndexed:    time.Now(),
		}, nil
	}

	stats, err := w.indexer.GetIndexStats()
	if err != nil {
		return nil, err
	}

	var totalDocs int64
	if count, ok := stats["number_of_documents"].(int64); ok {
		totalDocs = count
	}

	return &IndexStats{
		TotalDocuments: totalDocs,
		LastIndexed:    time.Now(),
	}, nil
}
