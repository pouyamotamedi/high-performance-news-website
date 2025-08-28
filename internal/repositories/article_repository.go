package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lib/pq"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/pkg/cache"
	"high-performance-news-website/pkg/database"
)

// ArticleRepository handles article data operations with caching and fallback strategies
type ArticleRepository struct {
	db           *database.DB
	cache        cache.CacheService
	invalidator  *cache.CacheInvalidator
	keyBuilder   *cache.CacheKeyBuilder
	staticPath   string // Path to static HTML files for fallback
}

// NewArticleRepository creates a new article repository with dependencies
func NewArticleRepository(db *database.DB, cacheService cache.CacheService, staticPath string) *ArticleRepository {
	return &ArticleRepository{
		db:          db,
		cache:       cacheService,
		invalidator: cache.NewCacheInvalidator(cacheService),
		keyBuilder:  cache.NewCacheKeyBuilder(),
		staticPath:  staticPath,
	}
}

// Create inserts a new article using prepared statements
func (r *ArticleRepository) Create(ctx context.Context, article *models.Article) (*models.Article, error) {
	// Validate and prepare article for database
	if err := models.ValidateArticle(article); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	
	article.PrepareForDB()
	
	// Get prepared statement
	stmt, err := r.db.GetPreparedStatement(database.StmtInsertArticle)
	if err != nil {
		return nil, fmt.Errorf("failed to get prepared statement: %w", err)
	}
	
	// Execute insert with prepared statement
	var id uint64
	var createdAt time.Time
	err = stmt.QueryRowContext(ctx,
		article.Title,
		article.Slug,
		article.Content,
		article.Excerpt,
		article.AuthorID,
		article.CategoryID,
		article.Status,
		article.PublishedAt,
		article.SEOData.MetaTitle,
		article.SEOData.MetaDescription,
		article.SEOData.CanonicalURL,
		article.SEOData.SchemaType,
	).Scan(&id, &createdAt)
	
	if err != nil {
		return nil, fmt.Errorf("failed to insert article: %w", err)
	}
	
	article.ID = id
	article.CreatedAt = createdAt
	article.UpdatedAt = createdAt
	
	// Async cache invalidation (don't block the response)
	go func() {
		if err := r.invalidator.InvalidateArticle(context.Background(), article.ID, article.Slug); err != nil {
			// Log error but don't fail the operation
			fmt.Printf("Cache invalidation failed for article %d: %v\n", article.ID, err)
		}
	}()
	
	return article, nil
}

// GetByID retrieves an article by ID with cache → database → static file fallback
func (r *ArticleRepository) GetByID(ctx context.Context, id uint64) (*models.Article, error) {
	cacheKey := r.keyBuilder.ArticleKey(id)
	
	// Try cache first
	if cached, err := r.cache.Get(ctx, cacheKey); err == nil && cached != nil {
		var article models.Article
		if err := json.Unmarshal(cached, &article); err == nil {
			return &article, nil
		}
	}
	
	// Fallback to database
	article, err := r.getByIDFromDB(ctx, id)
	if err != nil {
		return nil, err
	}
	
	// Cache the result asynchronously
	go func() {
		if data, err := json.Marshal(article); err == nil {
			r.cache.Set(context.Background(), cacheKey, data, cache.CacheTTLArticle)
		}
	}()
	
	return article, nil
}

// GetBySlug retrieves an article by slug with graceful degradation
func (r *ArticleRepository) GetBySlug(ctx context.Context, slug string) (*models.Article, error) {
	cacheKey := r.keyBuilder.ArticleSlugKey(slug)
	
	// Try cache first
	if cached, err := r.cache.Get(ctx, cacheKey); err == nil && cached != nil {
		var article models.Article
		if err := json.Unmarshal(cached, &article); err == nil {
			return &article, nil
		}
	}
	
	// Fallback to database
	article, err := r.getBySlugFromDB(ctx, slug)
	if err != nil {
		// Final fallback: try static file
		if staticArticle, staticErr := r.getFromStaticFile(slug); staticErr == nil {
			return staticArticle, nil
		}
		return nil, err
	}
	
	// Cache the result asynchronously
	go func() {
		if data, err := json.Marshal(article); err == nil {
			r.cache.Set(context.Background(), cacheKey, data, cache.CacheTTLArticle)
		}
	}()
	
	return article, nil
}

// GetByCategory retrieves articles by category with pagination
func (r *ArticleRepository) GetByCategory(ctx context.Context, categoryID uint64, limit, offset int) ([]models.Article, error) {
	stmt, err := r.db.GetPreparedStatement(database.StmtGetCategory)
	if err != nil {
		return nil, fmt.Errorf("failed to get prepared statement: %w", err)
	}
	
	rows, err := stmt.QueryContext(ctx, categoryID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query articles by category: %w", err)
	}
	defer rows.Close()
	
	var articles []models.Article
	for rows.Next() {
		var article models.Article
		err := rows.Scan(
			&article.ID,
			&article.Title,
			&article.Slug,
			&article.Excerpt,
			&article.AuthorID,
			&article.PublishedAt,
			&article.ViewCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}
		articles = append(articles, article)
	}
	
	return articles, nil
}

// GetTrendingArticles retrieves trending articles based on view count and recency
func (r *ArticleRepository) GetTrendingArticles(ctx context.Context, limit int, hours int) ([]models.Article, error) {
	cacheKey := r.keyBuilder.TrendingKey(fmt.Sprintf("%dh", hours))
	
	// Try cache first
	if cached, err := r.cache.Get(ctx, cacheKey); err == nil && cached != nil {
		var articles []models.Article
		if err := json.Unmarshal(cached, &articles); err == nil {
			return articles, nil
		}
	}
	
	// Query from database with trending algorithm
	query := `
		SELECT a.id, a.title, a.slug, a.excerpt, a.author_id, a.published_at, a.view_count,
			   (a.view_count * 0.7 + a.like_count * 0.2 + (EXTRACT(EPOCH FROM NOW() - a.published_at) / 3600)::int * -0.1) as trending_score
		FROM articles a
		WHERE a.status = 'published' 
		  AND a.published_at IS NOT NULL
		  AND a.published_at > NOW() - INTERVAL '%d hours'
		ORDER BY trending_score DESC
		LIMIT $1`
	
	formattedQuery := fmt.Sprintf(query, hours)
	rows, err := r.db.QueryContext(ctx, formattedQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query trending articles: %w", err)
	}
	defer rows.Close()
	
	var articles []models.Article
	for rows.Next() {
		var article models.Article
		var trendingScore float64
		err := rows.Scan(
			&article.ID,
			&article.Title,
			&article.Slug,
			&article.Excerpt,
			&article.AuthorID,
			&article.PublishedAt,
			&article.ViewCount,
			&trendingScore,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trending article: %w", err)
		}
		articles = append(articles, article)
	}
	
	// Cache the result asynchronously
	go func() {
		if data, err := json.Marshal(articles); err == nil {
			r.cache.Set(context.Background(), cacheKey, data, cache.CacheTTLCategory)
		}
	}()
	
	return articles, nil
}

// BulkCreate performs bulk insert using PostgreSQL COPY for high-volume processing
func (r *ArticleRepository) BulkCreate(ctx context.Context, articles []models.Article) error {
	if len(articles) == 0 {
		return nil
	}
	
	// Validate all articles first
	for i := range articles {
		if err := models.ValidateArticle(&articles[i]); err != nil {
			return fmt.Errorf("validation failed for article %d: %w", i, err)
		}
		articles[i].PrepareForDB()
	}
	
	// Use PostgreSQL COPY for maximum performance
	stmt, err := r.db.Prepare(pq.CopyIn("articles",
		"title", "slug", "content", "excerpt", "author_id", "category_id",
		"status", "published_at", "meta_title", "meta_description",
		"canonical_url", "schema_type", "created_at", "updated_at"))
	if err != nil {
		return fmt.Errorf("failed to prepare COPY statement: %w", err)
	}
	defer stmt.Close()
	
	now := time.Now()
	for _, article := range articles {
		_, err = stmt.ExecContext(ctx,
			article.Title,
			article.Slug,
			article.Content,
			article.Excerpt,
			article.AuthorID,
			article.CategoryID,
			article.Status,
			article.PublishedAt,
			article.SEOData.MetaTitle,
			article.SEOData.MetaDescription,
			article.SEOData.CanonicalURL,
			article.SEOData.SchemaType,
			now,
			now,
		)
		if err != nil {
			return fmt.Errorf("failed to execute COPY for article %s: %w", article.Title, err)
		}
	}
	
	// Execute the COPY
	if err := stmt.Close(); err != nil {
		return fmt.Errorf("failed to complete COPY operation: %w", err)
	}
	
	// Async cache invalidation for bulk operations
	go func() {
		if err := r.invalidator.InvalidateAll(context.Background()); err != nil {
			fmt.Printf("Cache invalidation failed for bulk operation: %v\n", err)
		}
	}()
	
	return nil
}

// Update modifies an existing article using prepared statements
func (r *ArticleRepository) Update(ctx context.Context, article *models.Article) error {
	if err := models.ValidateArticle(article); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	
	article.PrepareForDB()
	
	stmt, err := r.db.GetPreparedStatement(database.StmtUpdateArticle)
	if err != nil {
		return fmt.Errorf("failed to get prepared statement: %w", err)
	}
	
	result, err := stmt.ExecContext(ctx,
		article.ID,
		article.Title,
		article.Slug,
		article.Content,
		article.Excerpt,
		article.CategoryID,
		article.Status,
		article.PublishedAt,
		article.SEOData.MetaTitle,
		article.SEOData.MetaDescription,
		article.SEOData.CanonicalURL,
	)
	if err != nil {
		return fmt.Errorf("failed to update article: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("article not found or no changes made")
	}
	
	// Async cache invalidation
	go func() {
		if err := r.invalidator.InvalidateArticle(context.Background(), article.ID, article.Slug); err != nil {
			fmt.Printf("Cache invalidation failed for article %d: %v\n", article.ID, err)
		}
	}()
	
	return nil
}

// Delete removes an article (soft delete by changing status)
func (r *ArticleRepository) Delete(ctx context.Context, id uint64) error {
	query := `UPDATE articles SET status = 'archived', updated_at = NOW() WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete article: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("article not found")
	}
	
	// Async cache invalidation
	go func() {
		// Get article slug for cache invalidation
		if article, err := r.getByIDFromDB(context.Background(), id); err == nil {
			r.invalidator.InvalidateArticle(context.Background(), id, article.Slug)
		}
	}()
	
	return nil
}

// RecordView records an article view for analytics
func (r *ArticleRepository) RecordView(ctx context.Context, articleID uint64, ipAddress, userAgent, referer string) error {
	if r.db == nil {
		return fmt.Errorf("database connection not available")
	}
	
	stmt, err := r.db.GetPreparedStatement(database.StmtInsertView)
	if err != nil {
		return fmt.Errorf("failed to get prepared statement: %w", err)
	}
	
	_, err = stmt.ExecContext(ctx, articleID, ipAddress, userAgent, referer)
	if err != nil {
		return fmt.Errorf("failed to record view: %w", err)
	}
	
	// Update view count asynchronously
	go func() {
		updateQuery := `UPDATE articles SET view_count = view_count + 1 WHERE id = $1`
		r.db.ExecContext(context.Background(), updateQuery, articleID)
	}()
	
	return nil
}

// GetLatestArticles retrieves the most recent published articles
func (r *ArticleRepository) GetLatestArticles(ctx context.Context, limit int) ([]models.Article, error) {
	stmt, err := r.db.GetPreparedStatement(database.StmtGetHomepage)
	if err != nil {
		return nil, fmt.Errorf("failed to get prepared statement: %w", err)
	}
	
	rows, err := stmt.QueryContext(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest articles: %w", err)
	}
	defer rows.Close()
	
	var articles []models.Article
	for rows.Next() {
		var article models.Article
		err := rows.Scan(
			&article.ID,
			&article.Title,
			&article.Slug,
			&article.Excerpt,
			&article.AuthorID,
			&article.CategoryID,
			&article.PublishedAt,
			&article.ViewCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}
		articles = append(articles, article)
	}
	
	return articles, nil
}

// CountByCategory returns the total number of articles in a category
func (r *ArticleRepository) CountByCategory(ctx context.Context, categoryID uint64) (int64, error) {
	query := `SELECT COUNT(*) FROM articles WHERE category_id = $1 AND status = 'published'`
	
	var count int64
	err := r.db.QueryRowContext(ctx, query, categoryID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count articles by category: %w", err)
	}
	
	return count, nil
}

// GetByTag retrieves articles by tag with pagination
func (r *ArticleRepository) GetByTag(ctx context.Context, tagID uint64, limit, offset int) ([]models.Article, error) {
	query := `
		SELECT a.id, a.title, a.slug, a.excerpt, a.author_id, a.category_id, a.published_at, a.view_count
		FROM articles a
		INNER JOIN article_tags at ON a.id = at.article_id
		WHERE at.tag_id = $1 AND a.status = 'published'
		ORDER BY a.published_at DESC
		LIMIT $2 OFFSET $3`
	
	rows, err := r.db.QueryContext(ctx, query, tagID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query articles by tag: %w", err)
	}
	defer rows.Close()
	
	var articles []models.Article
	for rows.Next() {
		var article models.Article
		err := rows.Scan(
			&article.ID,
			&article.Title,
			&article.Slug,
			&article.Excerpt,
			&article.AuthorID,
			&article.CategoryID,
			&article.PublishedAt,
			&article.ViewCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}
		articles = append(articles, article)
	}
	
	return articles, nil
}

// CountByTag returns the total number of articles with a specific tag
func (r *ArticleRepository) CountByTag(ctx context.Context, tagID uint64) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM articles a
		INNER JOIN article_tags at ON a.id = at.article_id
		WHERE at.tag_id = $1 AND a.status = 'published'`
	
	var count int64
	err := r.db.QueryRowContext(ctx, query, tagID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count articles by tag: %w", err)
	}
	
	return count, nil
}

// Private helper methods

func (r *ArticleRepository) getByIDFromDB(ctx context.Context, id uint64) (*models.Article, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}
	
	query := `
		SELECT id, title, slug, content, excerpt, author_id, category_id,
			   status, published_at, created_at, updated_at, view_count, 
			   like_count, dislike_count, meta_title, meta_description, 
			   canonical_url, schema_type
		FROM articles 
		WHERE id = $1`
	
	var article models.Article
	var metaTitle, metaDescription, canonicalURL, schemaType sql.NullString
	
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&article.ID,
		&article.Title,
		&article.Slug,
		&article.Content,
		&article.Excerpt,
		&article.AuthorID,
		&article.CategoryID,
		&article.Status,
		&article.PublishedAt,
		&article.CreatedAt,
		&article.UpdatedAt,
		&article.ViewCount,
		&article.LikeCount,
		&article.DislikeCount,
		&metaTitle,
		&metaDescription,
		&canonicalURL,
		&schemaType,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("article not found")
		}
		return nil, fmt.Errorf("failed to query article: %w", err)
	}
	
	// Handle nullable SEO fields
	article.SEOData.MetaTitle = metaTitle.String
	article.SEOData.MetaDescription = metaDescription.String
	article.SEOData.CanonicalURL = canonicalURL.String
	article.SEOData.SchemaType = schemaType.String
	
	return &article, nil
}

func (r *ArticleRepository) getBySlugFromDB(ctx context.Context, slug string) (*models.Article, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}
	
	stmt, err := r.db.GetPreparedStatement(database.StmtGetArticle)
	if err != nil {
		return nil, fmt.Errorf("failed to get prepared statement: %w", err)
	}
	
	var article models.Article
	var metaTitle, metaDescription, canonicalURL, schemaType sql.NullString
	
	err = stmt.QueryRowContext(ctx, slug).Scan(
		&article.ID,
		&article.Title,
		&article.Slug,
		&article.Content,
		&article.Excerpt,
		&article.AuthorID,
		&article.CategoryID,
		&article.PublishedAt,
		&article.ViewCount,
		&article.LikeCount,
		&article.DislikeCount,
		&metaTitle,
		&metaDescription,
		&canonicalURL,
		&schemaType,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("article not found")
		}
		return nil, fmt.Errorf("failed to query article by slug: %w", err)
	}
	
	// Handle nullable SEO fields
	article.SEOData.MetaTitle = metaTitle.String
	article.SEOData.MetaDescription = metaDescription.String
	article.SEOData.CanonicalURL = canonicalURL.String
	article.SEOData.SchemaType = schemaType.String
	article.Status = "published" // Since we only query published articles
	
	return &article, nil
}

// getFromStaticFile attempts to load article from static HTML file as final fallback
func (r *ArticleRepository) getFromStaticFile(slug string) (*models.Article, error) {
	if r.staticPath == "" {
		return nil, fmt.Errorf("static path not configured")
	}
	
	staticFile := filepath.Join(r.staticPath, "articles", slug, "index.html")
	if _, err := os.Stat(staticFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("static file not found")
	}
	
	// For now, return a basic article structure
	// In a full implementation, you would parse the HTML to extract metadata
	return &models.Article{
		Slug:   slug,
		Status: "published",
		SEOData: models.SEOData{
			SchemaType: "NewsArticle",
		},
	}, nil
}

// GetTotalCount returns the total number of articles
func (r *ArticleRepository) GetTotalCount() (int64, error) {
	var count int64
	query := "SELECT COUNT(*) FROM articles"
	err := r.db.DB.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get total article count: %w", err)
	}
	return count, nil
}

// GetPublishedTodayCount returns the count of articles published today
func (r *ArticleRepository) GetPublishedTodayCount() (int64, error) {
	var count int64
	query := `
		SELECT COUNT(*) FROM articles 
		WHERE DATE(published_at) = CURRENT_DATE 
		AND status = 'published'`
	err := r.db.DB.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get published today count: %w", err)
	}
	return count, nil
}

// GetPendingCount returns the count of pending articles
func (r *ArticleRepository) GetPendingCount() (int64, error) {
	var count int64
	query := "SELECT COUNT(*) FROM articles WHERE status = 'pending'"
	err := r.db.DB.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get pending article count: %w", err)
	}
	return count, nil
}

// GetDraftCount returns the count of draft articles
func (r *ArticleRepository) GetDraftCount() (int64, error) {
	var count int64
	query := "SELECT COUNT(*) FROM articles WHERE status = 'draft'"
	err := r.db.DB.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get draft article count: %w", err)
	}
	return count, nil
}

// GetPublishedCount returns the count of published articles
func (r *ArticleRepository) GetPublishedCount() (int64, error) {
	var count int64
	query := "SELECT COUNT(*) FROM articles WHERE status = 'published'"
	err := r.db.DB.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get published article count: %w", err)
	}
	return count, nil
}

// RSS-specific methods for delayed publishing and feed generation

// GetPublishedArticlesBeforeTime retrieves published articles before a specific time (for RSS delay)
func (r *ArticleRepository) GetPublishedArticlesBeforeTime(cutoffTime time.Time, languageCode string, limit int) ([]models.Article, error) {
	query := `
		SELECT a.id, a.title, a.slug, a.content, a.excerpt, a.author_id, a.category_id, 
			   a.status, a.published_at, a.created_at, a.updated_at, a.view_count, 
			   a.like_count, a.dislike_count, a.seo_data, a.language_code
		FROM articles a
		WHERE a.status = 'published' 
		  AND a.published_at IS NOT NULL 
		  AND a.published_at <= $1
		  AND a.language_code = $2
		ORDER BY a.published_at DESC
		LIMIT $3`

	rows, err := r.db.QueryContext(context.Background(), query, cutoffTime, languageCode, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query articles: %w", err)
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		var article models.Article
		var seoDataJSON []byte

		err := rows.Scan(
			&article.ID, &article.Title, &article.Slug, &article.Content, &article.Excerpt,
			&article.AuthorID, &article.CategoryID, &article.Status, &article.PublishedAt,
			&article.CreatedAt, &article.UpdatedAt, &article.ViewCount, &article.LikeCount,
			&article.DislikeCount, &seoDataJSON, &article.LanguageCode,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}

		// Parse SEO data JSON
		if len(seoDataJSON) > 0 {
			if err := json.Unmarshal(seoDataJSON, &article.SEOData); err != nil {
				return nil, fmt.Errorf("failed to unmarshal SEO data: %w", err)
			}
		}

		// Load tags for the article
		tags, err := r.getArticleTags(article.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load tags for article %d: %w", article.ID, err)
		}
		article.Tags = tags

		articles = append(articles, article)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return articles, nil
}

// GetArticlesByCategoryBeforeTime retrieves articles by category before a specific time
func (r *ArticleRepository) GetArticlesByCategoryBeforeTime(categoryID uint64, cutoffTime time.Time, languageCode string, limit int) ([]models.Article, error) {
	query := `
		SELECT a.id, a.title, a.slug, a.content, a.excerpt, a.author_id, a.category_id, 
			   a.status, a.published_at, a.created_at, a.updated_at, a.view_count, 
			   a.like_count, a.dislike_count, a.seo_data, a.language_code
		FROM articles a
		WHERE a.status = 'published' 
		  AND a.published_at IS NOT NULL 
		  AND a.published_at <= $1
		  AND a.category_id = $2
		  AND a.language_code = $3
		ORDER BY a.published_at DESC
		LIMIT $4`

	rows, err := r.db.QueryContext(context.Background(), query, cutoffTime, categoryID, languageCode, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query category articles: %w", err)
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		var article models.Article
		var seoDataJSON []byte

		err := rows.Scan(
			&article.ID, &article.Title, &article.Slug, &article.Content, &article.Excerpt,
			&article.AuthorID, &article.CategoryID, &article.Status, &article.PublishedAt,
			&article.CreatedAt, &article.UpdatedAt, &article.ViewCount, &article.LikeCount,
			&article.DislikeCount, &seoDataJSON, &article.LanguageCode,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}

		// Parse SEO data JSON
		if len(seoDataJSON) > 0 {
			if err := json.Unmarshal(seoDataJSON, &article.SEOData); err != nil {
				return nil, fmt.Errorf("failed to unmarshal SEO data: %w", err)
			}
		}

		// Load tags for the article
		tags, err := r.getArticleTags(article.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load tags for article %d: %w", article.ID, err)
		}
		article.Tags = tags

		articles = append(articles, article)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return articles, nil
}

// GetArticlesByTagBeforeTime retrieves articles by tag before a specific time
func (r *ArticleRepository) GetArticlesByTagBeforeTime(tagID uint64, cutoffTime time.Time, languageCode string, limit int) ([]models.Article, error) {
	query := `
		SELECT a.id, a.title, a.slug, a.content, a.excerpt, a.author_id, a.category_id, 
			   a.status, a.published_at, a.created_at, a.updated_at, a.view_count, 
			   a.like_count, a.dislike_count, a.seo_data, a.language_code
		FROM articles a
		JOIN article_tags at ON a.id = at.article_id
		WHERE a.status = 'published' 
		  AND a.published_at IS NOT NULL 
		  AND a.published_at <= $1
		  AND at.tag_id = $2
		  AND a.language_code = $3
		ORDER BY a.published_at DESC
		LIMIT $4`

	rows, err := r.db.QueryContext(context.Background(), query, cutoffTime, tagID, languageCode, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query tag articles: %w", err)
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		var article models.Article
		var seoDataJSON []byte

		err := rows.Scan(
			&article.ID, &article.Title, &article.Slug, &article.Content, &article.Excerpt,
			&article.AuthorID, &article.CategoryID, &article.Status, &article.PublishedAt,
			&article.CreatedAt, &article.UpdatedAt, &article.ViewCount, &article.LikeCount,
			&article.DislikeCount, &seoDataJSON, &article.LanguageCode,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}

		// Parse SEO data JSON
		if len(seoDataJSON) > 0 {
			if err := json.Unmarshal(seoDataJSON, &article.SEOData); err != nil {
				return nil, fmt.Errorf("failed to unmarshal SEO data: %w", err)
			}
		}

		// Load tags for the article
		tags, err := r.getArticleTags(article.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load tags for article %d: %w", article.ID, err)
		}
		article.Tags = tags

		articles = append(articles, article)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return articles, nil
}

// GetPublishedArticlesAfterTimeWithOffset retrieves articles after a specific time with offset (for Google News sitemap)
func (r *ArticleRepository) GetPublishedArticlesAfterTimeWithOffset(cutoffTime time.Time, languageCode string, limit, offset int) ([]models.Article, error) {
	query := `
		SELECT a.id, a.title, a.slug, a.content, a.excerpt, a.author_id, a.category_id, 
			   a.status, a.published_at, a.created_at, a.updated_at, a.view_count, 
			   a.like_count, a.dislike_count, a.seo_data, a.language_code
		FROM articles a
		WHERE a.status = 'published' 
		  AND a.published_at IS NOT NULL 
		  AND a.published_at >= $1
		  AND a.language_code = $2
		ORDER BY a.published_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.db.QueryContext(context.Background(), query, cutoffTime, languageCode, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query articles: %w", err)
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		var article models.Article
		var seoDataJSON []byte

		err := rows.Scan(
			&article.ID, &article.Title, &article.Slug, &article.Content, &article.Excerpt,
			&article.AuthorID, &article.CategoryID, &article.Status, &article.PublishedAt,
			&article.CreatedAt, &article.UpdatedAt, &article.ViewCount, &article.LikeCount,
			&article.DislikeCount, &seoDataJSON, &article.LanguageCode,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}

		// Parse SEO data JSON
		if len(seoDataJSON) > 0 {
			if err := json.Unmarshal(seoDataJSON, &article.SEOData); err != nil {
				return nil, fmt.Errorf("failed to unmarshal SEO data: %w", err)
			}
		}

		// Load tags for the article
		tags, err := r.getArticleTags(article.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load tags for article %d: %w", article.ID, err)
		}
		article.Tags = tags

		articles = append(articles, article)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return articles, nil
}

// CountPublishedArticlesBeforeTime counts articles published before a specific time
func (r *ArticleRepository) CountPublishedArticlesBeforeTime(cutoffTime time.Time, languageCode string) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM articles
		WHERE status = 'published' 
		  AND published_at IS NOT NULL 
		  AND published_at <= $1
		  AND language_code = $2`

	var count int64
	err := r.db.QueryRowContext(context.Background(), query, cutoffTime, languageCode).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count articles: %w", err)
	}

	return count, nil
}

// CountPublishedArticlesAfterTime counts articles published after a specific time
func (r *ArticleRepository) CountPublishedArticlesAfterTime(cutoffTime time.Time, languageCode string) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM articles
		WHERE status = 'published' 
		  AND published_at IS NOT NULL 
		  AND published_at >= $1
		  AND language_code = $2`

	var count int64
	err := r.db.QueryRowContext(context.Background(), query, cutoffTime, languageCode).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count articles: %w", err)
	}

	return count, nil
}

// getArticleTags retrieves tags for a specific article
func (r *ArticleRepository) getArticleTags(articleID uint64) ([]models.Tag, error) {
	query := `
		SELECT t.id, t.name, t.slug, t.description, t.keywords, t.color, 
			   t.created_at, t.updated_at, t.language_code, t.translation_group_id
		FROM tags t
		JOIN article_tags at ON t.id = at.tag_id
		WHERE at.article_id = $1
		ORDER BY t.name`

	rows, err := r.db.QueryContext(context.Background(), query, articleID)
	if err != nil {
		return nil, fmt.Errorf("failed to query article tags: %w", err)
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var tag models.Tag
		var keywordsJSON []byte

		err := rows.Scan(
			&tag.ID, &tag.Name, &tag.Slug, &tag.Description, &keywordsJSON,
			&tag.Color, &tag.CreatedAt, &tag.UpdatedAt, &tag.LanguageCode,
			&tag.TranslationGroupID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}

		// Parse keywords JSON
		if len(keywordsJSON) > 0 {
			if err := json.Unmarshal(keywordsJSON, &tag.Keywords); err != nil {
				return nil, fmt.Errorf("failed to unmarshal keywords: %w", err)
			}
		}

		tags = append(tags, tag)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tag rows: %w", err)
	}

	return tags, nil
}