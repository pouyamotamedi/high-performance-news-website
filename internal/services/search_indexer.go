package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"high-performance-news-website/internal/models"

	meilisearch "github.com/meilisearch/meilisearch-go"
)

// SearchDocument represents an article document for search indexing
type SearchDocument struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Slug            string   `json:"slug"`
	Content         string   `json:"content"`
	Excerpt         string   `json:"excerpt"`
	FeaturedImage   string   `json:"featured_image"`
	AuthorID        uint64   `json:"author_id"`
	CategoryID      uint64   `json:"category_id"`
	Tags            []string `json:"tags"`
	Status          string   `json:"status"`
	PublishedAt     int64    `json:"published_at"`
	CreatedAt       int64    `json:"created_at"`
	ViewCount       uint64   `json:"view_count"`
	LikeCount       uint64   `json:"like_count"`
	LanguageCode    string   `json:"language_code"`
	MetaTitle       string   `json:"meta_title"`
	MetaDescription string   `json:"meta_description"`
	Keywords        []string `json:"keywords"`
}

// SearchIndexer handles MeiliSearch indexing operations
type SearchIndexer struct {
	client     meilisearch.ServiceManager
	indexName  string
	batchSize  int
	timeout    time.Duration
	fallbackDB ArticleRepository
}

// ArticleRepository interface for fallback operations
type ArticleRepository interface {
	GetByID(ctx context.Context, id uint64) (*models.Article, error)
	Search(query string, filters map[string]interface{}, limit, offset int) ([]models.Article, int64, error)
}

// SearchIndexerInterface defines the interface for search indexing operations
type SearchIndexerInterface interface {
	IndexArticle(article *models.Article) error
	IndexArticlesBatch(articles []models.Article) error
	DeleteArticle(articleID string) error
	RebuildIndex(articles []models.Article) error
	ClearIndex() error
	GetIndexStats() (map[string]interface{}, error)
	HealthCheck() error
	GetClient() meilisearch.ServiceManager
	GetIndexName() string
}

// NewSearchIndexer creates a new search indexer instance
func NewSearchIndexer(host, apiKey, indexName string, fallbackDB ArticleRepository) *SearchIndexer {
	client := meilisearch.New(host, meilisearch.WithAPIKey(apiKey))

	indexer := &SearchIndexer{
		client:     client,
		indexName:  indexName,
		batchSize:  1000, // As specified in requirements
		timeout:    30 * time.Second,
		fallbackDB: fallbackDB,
	}

	// Initialize index settings
	if err := indexer.initializeIndex(); err != nil {
		log.Printf("Warning: Failed to initialize search index: %v", err)
	}

	return indexer
}

// initializeIndex sets up the MeiliSearch index with proper configuration
func (si *SearchIndexer) initializeIndex() error {

	// Configure searchable attributes
	searchableAttributes := []string{
		"title",
		"content",
		"excerpt",
		"meta_title",
		"meta_description",
		"keywords",
		"tags",
	}

	if _, err := si.client.Index(si.indexName).UpdateSearchableAttributes(&searchableAttributes); err != nil {
		return fmt.Errorf("failed to update searchable attributes: %w", err)
	}

	// Configure filterable attributes
	filterableAttributes := []string{
		"author_id",
		"category_id",
		"status",
		"published_at",
		"language_code",
		"tags",
	}

	filterableAttrsInterface := make([]interface{}, len(filterableAttributes))
	for i, attr := range filterableAttributes {
		filterableAttrsInterface[i] = attr
	}

	if _, err := si.client.Index(si.indexName).UpdateFilterableAttributes(&filterableAttrsInterface); err != nil {
		return fmt.Errorf("failed to update filterable attributes: %w", err)
	}

	// Configure sortable attributes
	sortableAttributes := []string{
		"published_at",
		"created_at",
		"view_count",
		"like_count",
	}

	if _, err := si.client.Index(si.indexName).UpdateSortableAttributes(&sortableAttributes); err != nil {
		return fmt.Errorf("failed to update sortable attributes: %w", err)
	}

	// Configure ranking rules for news relevance
	rankingRules := []string{
		"words",
		"typo",
		"proximity",
		"attribute",
		"sort",      // Enable dynamic sorting at search time
		"exactness",
	}

	if _, err := si.client.Index(si.indexName).UpdateRankingRules(&rankingRules); err != nil {
		return fmt.Errorf("failed to update ranking rules: %w", err)
	}

	// Wait for settings to be applied
	time.Sleep(1 * time.Second)

	return nil
}

// IndexArticle indexes a single article in real-time
func (si *SearchIndexer) IndexArticle(article *models.Article) error {
	if article.Status != "published" {
		// Only index published articles
		return si.DeleteArticle(fmt.Sprintf("%d", article.ID))
	}

	doc := si.convertToSearchDocument(article)
	
	primaryKey := "id"
	_, err := si.client.Index(si.indexName).AddDocuments([]SearchDocument{doc}, &primaryKey)
	if err != nil {
		return fmt.Errorf("failed to index article %d: %w", article.ID, err)
	}

	log.Printf("Successfully indexed article %d: %s", article.ID, article.Title)
	return nil
}

// IndexArticlesBatch indexes multiple articles in batches for optimal performance
func (si *SearchIndexer) IndexArticlesBatch(articles []models.Article) error {
	if len(articles) == 0 {
		return nil
	}

	// Filter only published articles
	var publishedArticles []models.Article
	for _, article := range articles {
		if article.Status == "published" {
			publishedArticles = append(publishedArticles, article)
		}
	}

	if len(publishedArticles) == 0 {
		return nil
	}

	// Process in batches of 1000 articles
	for i := 0; i < len(publishedArticles); i += si.batchSize {
		end := i + si.batchSize
		if end > len(publishedArticles) {
			end = len(publishedArticles)
		}

		batch := publishedArticles[i:end]
		if err := si.processBatch(batch); err != nil {
			return fmt.Errorf("failed to process batch %d-%d: %w", i, end-1, err)
		}

		log.Printf("Successfully indexed batch %d-%d (%d articles)", i, end-1, len(batch))
		
		// Small delay between batches to prevent overwhelming MeiliSearch
		if end < len(publishedArticles) {
			time.Sleep(100 * time.Millisecond)
		}
	}

	log.Printf("Successfully indexed %d articles in batches", len(publishedArticles))
	return nil
}

// processBatch processes a single batch of articles
func (si *SearchIndexer) processBatch(articles []models.Article) error {
	docs := make([]SearchDocument, len(articles))
	for i, article := range articles {
		docs[i] = si.convertToSearchDocument(&article)
	}

	primaryKey := "id"
	_, err := si.client.Index(si.indexName).AddDocuments(docs, &primaryKey)
	if err != nil {
		return fmt.Errorf("failed to add documents to index: %w", err)
	}

	return nil
}

// DeleteArticle removes an article from the search index
func (si *SearchIndexer) DeleteArticle(articleID string) error {
	_, err := si.client.Index(si.indexName).DeleteDocument(articleID)
	if err != nil {
		return fmt.Errorf("failed to delete article %s from index: %w", articleID, err)
	}

	log.Printf("Successfully deleted article %s from search index", articleID)
	return nil
}

// RebuildIndex completely rebuilds the search index
func (si *SearchIndexer) RebuildIndex(articles []models.Article) error {
	log.Printf("Starting search index rebuild with %d articles", len(articles))

	// Clear existing index
	if err := si.ClearIndex(); err != nil {
		return fmt.Errorf("failed to clear index: %w", err)
	}

	// Rebuild with all articles
	if err := si.IndexArticlesBatch(articles); err != nil {
		return fmt.Errorf("failed to rebuild index: %w", err)
	}

	log.Printf("Successfully rebuilt search index with %d articles", len(articles))
	return nil
}

// ClearIndex removes all documents from the search index
func (si *SearchIndexer) ClearIndex() error {
	_, err := si.client.Index(si.indexName).DeleteAllDocuments()
	if err != nil {
		return fmt.Errorf("failed to clear search index: %w", err)
	}

	log.Printf("Successfully cleared search index")
	return nil
}

// GetIndexStats returns statistics about the search index
func (si *SearchIndexer) GetIndexStats() (map[string]interface{}, error) {
	stats, err := si.client.Index(si.indexName).GetStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get index stats: %w", err)
	}

	return map[string]interface{}{
		"number_of_documents": stats.NumberOfDocuments,
		"is_indexing":        stats.IsIndexing,
		"field_distribution": stats.FieldDistribution,
	}, nil
}

// HealthCheck verifies that MeiliSearch is accessible
func (si *SearchIndexer) HealthCheck() error {
	_, err := si.client.Health()
	if err != nil {
		return fmt.Errorf("MeiliSearch health check failed: %w", err)
	}

	return nil
}

// convertToSearchDocument converts an Article model to SearchDocument
func (si *SearchIndexer) convertToSearchDocument(article *models.Article) SearchDocument {
	var publishedAt int64
	if article.PublishedAt != nil {
		publishedAt = article.PublishedAt.Unix()
	}

	// Extract tag names
	tagNames := make([]string, len(article.Tags))
	for i, tag := range article.Tags {
		tagNames[i] = tag.Name
	}

	return SearchDocument{
		ID:              fmt.Sprintf("%d", article.ID),
		Title:           article.Title,
		Slug:            article.Slug,
		Content:         si.truncateContent(article.Content),
		Excerpt:         article.Excerpt,
		FeaturedImage:   article.FeaturedImage,
		AuthorID:        article.AuthorID,
		CategoryID:      article.CategoryID,
		Tags:            tagNames,
		Status:          article.Status,
		PublishedAt:     publishedAt,
		CreatedAt:       article.CreatedAt.Unix(),
		ViewCount:       article.ViewCount,
		LikeCount:       article.LikeCount,
		LanguageCode:    article.LanguageCode,
		MetaTitle:       article.SEOData.MetaTitle,
		MetaDescription: article.SEOData.MetaDescription,
		Keywords:        article.SEOData.Keywords,
	}
}

// GetClient returns the MeiliSearch client
func (si *SearchIndexer) GetClient() meilisearch.ServiceManager {
	return si.client
}

// GetIndexName returns the index name
func (si *SearchIndexer) GetIndexName() string {
	return si.indexName
}

// truncateContent limits content length for search indexing
func (si *SearchIndexer) truncateContent(content string) string {
	// Limit content to 10,000 characters for search indexing
	const maxLength = 10000
	
	if len(content) <= maxLength {
		return content
	}
	
	// Truncate at word boundary
	truncated := content[:maxLength]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > 0 {
		truncated = truncated[:lastSpace]
	}
	
	return truncated + "..."
}