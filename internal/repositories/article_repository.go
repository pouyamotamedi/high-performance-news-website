package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
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
	
	// Ensure partition exists for the article's published date
	if err := r.ensurePartitionExists(ctx, article.PublishedAt); err != nil {
		log.Printf("Warning: Failed to ensure partition exists: %v", err)
		// Continue anyway - the insert might still work if partition exists
	}
	
	// Get prepared statement
	stmt, err := r.db.GetPreparedStatement(database.StmtInsertArticle)
	if err != nil {
		return nil, fmt.Errorf("failed to get prepared statement: %w", err)
	}
	
	// Execute insert with prepared statement
	var id uint64
	var createdAt time.Time
	log.Printf("Repository: Inserting article with AutoLinking=%t", article.AutoLinking)
	
	// Debug: Add logging to getByIDFromDB method as well
	
	err = stmt.QueryRowContext(ctx,
		article.Title,
		article.Slug,
		article.Content,
		article.Excerpt,
		article.AuthorID,
		article.CategoryID,
		article.Status,
		article.PublishedAt,
		article.MetaTitle,
		article.MetaDescription,
		article.CanonicalURL,
		article.SchemaType,
		article.FeaturedImageID,
		article.AutoLinking,
		article.LanguageCode,
		article.FocusKeyword,
		article.ModerationStatus,
		article.LastModeratedBy,
		article.TranslationGroupID,
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

// GetByTranslationGroup retrieves all articles in the same translation group
// This is used for generating hreflang tags with only existing translations
func (r *ArticleRepository) GetByTranslationGroup(ctx context.Context, translationGroupID uint64) ([]models.Article, error) {
	query := `
		SELECT id, title, slug, content, excerpt, author_id, category_id, status, 
		       published_at, created_at, updated_at, view_count, like_count, dislike_count,
		       meta_title, meta_description, focus_keyword, canonical_url, schema_type,
		       auto_linking, language_code, translation_group_id, moderation_status,
		       moderation_notes, last_moderated_at, last_moderated_by, featured_image_id
		FROM articles 
		WHERE (translation_group_id = $1 OR id = $1) AND status = 'published'
		ORDER BY language_code
	`
	
	rows, err := r.db.Query(query, translationGroupID)
	if err != nil {
		return nil, fmt.Errorf("failed to query articles by translation group: %w", err)
	}
	defer rows.Close()
	
	var articles []models.Article
	for rows.Next() {
		var article models.Article
		var publishedAt, lastModeratedAt sql.NullTime
		var translationGrpID, lastModeratedBy, featuredImageID sql.NullInt64
		var moderationNotes sql.NullString
		
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
			&article.CreatedAt,
			&article.UpdatedAt,
			&article.ViewCount,
			&article.LikeCount,
			&article.DislikeCount,
			&article.MetaTitle,
			&article.MetaDescription,
			&article.FocusKeyword,
			&article.CanonicalURL,
			&article.SchemaType,
			&article.AutoLinking,
			&article.LanguageCode,
			&translationGrpID,
			&article.ModerationStatus,
			&moderationNotes,
			&lastModeratedAt,
			&lastModeratedBy,
			&featuredImageID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}
		
		if publishedAt.Valid {
			article.PublishedAt = &publishedAt.Time
		}
		if translationGrpID.Valid {
			tgid := uint64(translationGrpID.Int64)
			article.TranslationGroupID = &tgid
		}
		if lastModeratedAt.Valid {
			article.LastModeratedAt = &lastModeratedAt.Time
		}
		if lastModeratedBy.Valid {
			lmb := uint64(lastModeratedBy.Int64)
			article.LastModeratedBy = &lmb
		}
		if featuredImageID.Valid {
			fid := uint64(featuredImageID.Int64)
			article.FeaturedImageID = &fid
		}
		if moderationNotes.Valid {
			article.ModerationNotes = moderationNotes.String
		}
		
		articles = append(articles, article)
	}
	
	return articles, nil
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
		var featuredImage sql.NullString
		err := rows.Scan(
			&article.ID,
			&article.Title,
			&article.Slug,
			&article.Excerpt,
			&article.AuthorID,
			&article.PublishedAt,
			&article.ViewCount,
			&featuredImage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}
		if featuredImage.Valid {
			article.FeaturedImage = featuredImage.String
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
			   CASE WHEN i.original_url LIKE '/uploads/%' THEN i.original_url ELSE NULL END,
			   (a.view_count * 0.7 + a.like_count * 0.2 + (EXTRACT(EPOCH FROM NOW() - a.published_at) / 3600)::int * -0.1) as trending_score
		FROM articles a
		LEFT JOIN images i ON a.featured_image_id = i.id
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
		var featuredImage sql.NullString
		var trendingScore float64
		err := rows.Scan(
			&article.ID,
			&article.Title,
			&article.Slug,
			&article.Excerpt,
			&article.AuthorID,
			&article.PublishedAt,
			&article.ViewCount,
			&featuredImage,
			&trendingScore,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trending article: %w", err)
		}
		if featuredImage.Valid {
			article.FeaturedImage = featuredImage.String
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
			article.MetaTitle,
			article.MetaDescription,
			article.CanonicalURL,
			article.SchemaType,
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
	
	// Use direct query instead of prepared statement to include focus_keyword
	query := `
		UPDATE articles SET 
			title = $2, slug = $3, content = $4, excerpt = $5, category_id = $6,
			status = $7, published_at = $8, meta_title = $9, meta_description = $10,
			focus_keyword = $11, canonical_url = $12, featured_image_id = $13, 
			auto_linking = $14, updated_at = NOW()
		WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query,
		article.ID,
		article.Title,
		article.Slug,
		article.Content,
		article.Excerpt,
		article.CategoryID,
		article.Status,
		article.PublishedAt,
		article.MetaTitle,
		article.MetaDescription,
		article.FocusKeyword,
		article.CanonicalURL,
		article.FeaturedImageID,
		article.AutoLinking,
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
		var featuredImage sql.NullString
		err := rows.Scan(
			&article.ID,
			&article.Title,
			&article.Slug,
			&article.Excerpt,
			&article.AuthorID,
			&article.CategoryID,
			&article.PublishedAt,
			&article.ViewCount,
			&featuredImage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}
		if featuredImage.Valid {
			article.FeaturedImage = featuredImage.String
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
	log.Printf("getByIDFromDB called for article ID: %d", id)
	if r.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}
	
	query := `
		SELECT a.id, a.title, a.slug, a.content, a.excerpt, a.author_id, a.category_id,
			   a.status, a.published_at, a.created_at, a.updated_at, a.view_count, 
			   a.like_count, a.dislike_count, a.meta_title, a.meta_description, 
			   a.focus_keyword, a.canonical_url, a.schema_type, a.featured_image_id, a.auto_linking,
			   CASE WHEN i.original_url LIKE '/uploads/%' THEN i.original_url ELSE NULL END as featured_image
		FROM articles a
		LEFT JOIN images i ON a.featured_image_id = i.id
		WHERE a.id = $1`
	
	var article models.Article
	var metaTitle, metaDescription, focusKeyword, canonicalURL, schemaType sql.NullString
	var featuredImageID sql.NullInt64
	var featuredImage sql.NullString
	
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
		&focusKeyword,
		&canonicalURL,
		&schemaType,
		&featuredImageID,
		&article.AutoLinking,
		&featuredImage,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("article not found")
		}
		return nil, fmt.Errorf("failed to query article: %w", err)
	}
	
	// Handle nullable SEO fields
	article.MetaTitle = metaTitle.String
	article.MetaDescription = metaDescription.String
	article.FocusKeyword = focusKeyword.String
	article.CanonicalURL = canonicalURL.String
	article.SchemaType = schemaType.String
	
	// Handle featured image fields
	if featuredImageID.Valid {
		imageID := uint64(featuredImageID.Int64)
		article.FeaturedImageID = &imageID
	}
	if featuredImage.Valid {
		article.FeaturedImage = featuredImage.String
	}
	
	// PROFESSIONAL FIX: Load tags for the article
	tags, err := r.getArticleTags(id)
	if err != nil {
		log.Printf("Warning: Failed to load tags for article %d: %v", id, err)
		article.Tags = []models.Tag{} // Empty array instead of null
	} else {
		article.Tags = tags
	}
	
	// Load categories for the article
	log.Printf("DEBUG: Loading categories for article %d (getBySlugFromDB)", article.ID)
	categories, err := r.getArticleCategories(article.ID)
	if err != nil {
		log.Printf("Warning: Failed to load categories for article %d: %v", article.ID, err)
		article.Categories = []models.Category{} // Empty array instead of null
	} else {
		log.Printf("DEBUG: Loaded %d categories for article %d (getBySlugFromDB)", len(categories), article.ID)
		article.Categories = categories
	}
	
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
	var featuredImageID sql.NullInt64
	var featuredImage sql.NullString
	
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
		&featuredImageID,
		&article.AutoLinking,
		&featuredImage,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("article not found")
		}
		return nil, fmt.Errorf("failed to query article by slug: %w", err)
	}
	
	// Handle nullable SEO fields
	article.MetaTitle = metaTitle.String
	article.MetaDescription = metaDescription.String
	article.CanonicalURL = canonicalURL.String
	article.SchemaType = schemaType.String
	article.Status = "published" // Since we only query published articles
	
	// Handle featured image fields
	if featuredImageID.Valid {
		imageID := uint64(featuredImageID.Int64)
		article.FeaturedImageID = &imageID
	}
	if featuredImage.Valid {
		article.FeaturedImage = featuredImage.String
	}
	
	// Load tags for the article
	tags, err := r.getArticleTags(article.ID)
	if err != nil {
		log.Printf("Warning: Failed to load tags for article %d: %v", article.ID, err)
		article.Tags = []models.Tag{} // Empty array instead of null
	} else {
		article.Tags = tags
	}
	
	// Load categories for the article
	categories, err := r.getArticleCategories(article.ID)
	if err != nil {
		log.Printf("Warning: Failed to load categories for article %d: %v", article.ID, err)
		article.Categories = []models.Category{} // Empty array instead of null
	} else {
		article.Categories = categories
	}
	
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
	// Build query - if languageCode is empty or "en", get all articles (most sites default to English)
	var query string
	var rows *sql.Rows
	var err error

	if languageCode == "" || languageCode == "en" {
		query = `
			SELECT a.id, a.title, a.slug, a.content, a.excerpt, a.author_id, a.category_id, 
				   a.status, a.published_at, a.created_at, a.updated_at, a.view_count, 
				   a.like_count, a.dislike_count, a.language_code,
				   COALESCE(a.meta_title, ''), COALESCE(a.meta_description, ''), 
				   COALESCE(a.canonical_url, ''), COALESCE(a.schema_type, 'NewsArticle'),
				   COALESCE(a.focus_keyword, '')
			FROM articles a
			WHERE a.status = 'published' 
			  AND a.published_at IS NOT NULL 
			  AND a.published_at <= $1
			ORDER BY a.published_at DESC
			LIMIT $2`
		rows, err = r.db.QueryContext(context.Background(), query, cutoffTime, limit)
	} else {
		query = `
			SELECT a.id, a.title, a.slug, a.content, a.excerpt, a.author_id, a.category_id, 
				   a.status, a.published_at, a.created_at, a.updated_at, a.view_count, 
				   a.like_count, a.dislike_count, a.language_code,
				   COALESCE(a.meta_title, ''), COALESCE(a.meta_description, ''), 
				   COALESCE(a.canonical_url, ''), COALESCE(a.schema_type, 'NewsArticle'),
				   COALESCE(a.focus_keyword, '')
			FROM articles a
			WHERE a.status = 'published' 
			  AND a.published_at IS NOT NULL 
			  AND a.published_at <= $1
			  AND a.language_code = $2
			ORDER BY a.published_at DESC
			LIMIT $3`
		rows, err = r.db.QueryContext(context.Background(), query, cutoffTime, languageCode, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query articles: %w", err)
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		var article models.Article
		var metaTitle, metaDesc, canonicalURL, schemaType, focusKeyword string

		err := rows.Scan(
			&article.ID, &article.Title, &article.Slug, &article.Content, &article.Excerpt,
			&article.AuthorID, &article.CategoryID, &article.Status, &article.PublishedAt,
			&article.CreatedAt, &article.UpdatedAt, &article.ViewCount, &article.LikeCount,
			&article.DislikeCount, &article.LanguageCode,
			&metaTitle, &metaDesc, &canonicalURL, &schemaType, &focusKeyword,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}

		// Populate SEOData from individual columns
		article.SEOData = models.SEOData{
			MetaTitle:       metaTitle,
			MetaDescription: metaDesc,
			CanonicalURL:    canonicalURL,
			SchemaType:      schemaType,
			FocusKeyword:    focusKeyword,
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
	// Build query - if languageCode is empty or "en", get all articles
	var query string
	var rows *sql.Rows
	var err error

	if languageCode == "" || languageCode == "en" {
		query = `
			SELECT a.id, a.title, a.slug, a.content, a.excerpt, a.author_id, a.category_id, 
				   a.status, a.published_at, a.created_at, a.updated_at, a.view_count, 
				   a.like_count, a.dislike_count, a.language_code,
				   COALESCE(a.meta_title, ''), COALESCE(a.meta_description, ''), 
				   COALESCE(a.canonical_url, ''), COALESCE(a.schema_type, 'NewsArticle'),
				   COALESCE(a.focus_keyword, '')
			FROM articles a
			WHERE a.status = 'published' 
			  AND a.published_at IS NOT NULL 
			  AND a.published_at <= $1
			  AND a.category_id = $2
			ORDER BY a.published_at DESC
			LIMIT $3`
		rows, err = r.db.QueryContext(context.Background(), query, cutoffTime, categoryID, limit)
	} else {
		query = `
			SELECT a.id, a.title, a.slug, a.content, a.excerpt, a.author_id, a.category_id, 
				   a.status, a.published_at, a.created_at, a.updated_at, a.view_count, 
				   a.like_count, a.dislike_count, a.language_code,
				   COALESCE(a.meta_title, ''), COALESCE(a.meta_description, ''), 
				   COALESCE(a.canonical_url, ''), COALESCE(a.schema_type, 'NewsArticle'),
				   COALESCE(a.focus_keyword, '')
			FROM articles a
			WHERE a.status = 'published' 
			  AND a.published_at IS NOT NULL 
			  AND a.published_at <= $1
			  AND a.category_id = $2
			  AND a.language_code = $3
			ORDER BY a.published_at DESC
			LIMIT $4`
		rows, err = r.db.QueryContext(context.Background(), query, cutoffTime, categoryID, languageCode, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query category articles: %w", err)
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		var article models.Article
		var metaTitle, metaDesc, canonicalURL, schemaType, focusKeyword string

		err := rows.Scan(
			&article.ID, &article.Title, &article.Slug, &article.Content, &article.Excerpt,
			&article.AuthorID, &article.CategoryID, &article.Status, &article.PublishedAt,
			&article.CreatedAt, &article.UpdatedAt, &article.ViewCount, &article.LikeCount,
			&article.DislikeCount, &article.LanguageCode,
			&metaTitle, &metaDesc, &canonicalURL, &schemaType, &focusKeyword,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}

		// Populate SEOData from individual columns
		article.SEOData = models.SEOData{
			MetaTitle:       metaTitle,
			MetaDescription: metaDesc,
			CanonicalURL:    canonicalURL,
			SchemaType:      schemaType,
			FocusKeyword:    focusKeyword,
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
	// Build query - if languageCode is empty or "en", get all articles
	var query string
	var rows *sql.Rows
	var err error

	if languageCode == "" || languageCode == "en" {
		query = `
			SELECT a.id, a.title, a.slug, a.content, a.excerpt, a.author_id, a.category_id, 
				   a.status, a.published_at, a.created_at, a.updated_at, a.view_count, 
				   a.like_count, a.dislike_count, a.language_code,
				   COALESCE(a.meta_title, ''), COALESCE(a.meta_description, ''), 
				   COALESCE(a.canonical_url, ''), COALESCE(a.schema_type, 'NewsArticle'),
				   COALESCE(a.focus_keyword, '')
			FROM articles a
			JOIN article_tags at ON a.id = at.article_id
			WHERE a.status = 'published' 
			  AND a.published_at IS NOT NULL 
			  AND a.published_at <= $1
			  AND at.tag_id = $2
			ORDER BY a.published_at DESC
			LIMIT $3`
		rows, err = r.db.QueryContext(context.Background(), query, cutoffTime, tagID, limit)
	} else {
		query = `
			SELECT a.id, a.title, a.slug, a.content, a.excerpt, a.author_id, a.category_id, 
				   a.status, a.published_at, a.created_at, a.updated_at, a.view_count, 
				   a.like_count, a.dislike_count, a.language_code,
				   COALESCE(a.meta_title, ''), COALESCE(a.meta_description, ''), 
				   COALESCE(a.canonical_url, ''), COALESCE(a.schema_type, 'NewsArticle'),
				   COALESCE(a.focus_keyword, '')
			FROM articles a
			JOIN article_tags at ON a.id = at.article_id
			WHERE a.status = 'published' 
			  AND a.published_at IS NOT NULL 
			  AND a.published_at <= $1
			  AND at.tag_id = $2
			  AND a.language_code = $3
			ORDER BY a.published_at DESC
			LIMIT $4`
		rows, err = r.db.QueryContext(context.Background(), query, cutoffTime, tagID, languageCode, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query tag articles: %w", err)
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		var article models.Article
		var metaTitle, metaDesc, canonicalURL, schemaType, focusKeyword string

		err := rows.Scan(
			&article.ID, &article.Title, &article.Slug, &article.Content, &article.Excerpt,
			&article.AuthorID, &article.CategoryID, &article.Status, &article.PublishedAt,
			&article.CreatedAt, &article.UpdatedAt, &article.ViewCount, &article.LikeCount,
			&article.DislikeCount, &article.LanguageCode,
			&metaTitle, &metaDesc, &canonicalURL, &schemaType, &focusKeyword,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}

		// Populate SEOData from individual columns
		article.SEOData = models.SEOData{
			MetaTitle:       metaTitle,
			MetaDescription: metaDesc,
			CanonicalURL:    canonicalURL,
			SchemaType:      schemaType,
			FocusKeyword:    focusKeyword,
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
	// Build query - if languageCode is empty or "en", get all articles (most sites default to English)
	var query string
	var rows *sql.Rows
	var err error
	
	if languageCode == "" || languageCode == "en" {
		query = `
			SELECT a.id, a.title, a.slug, a.content, a.excerpt, a.author_id, a.category_id, 
				   a.status, a.published_at, a.created_at, a.updated_at, a.view_count, 
				   a.like_count, a.dislike_count, 
				   COALESCE(a.meta_title, ''), COALESCE(a.meta_description, ''), 
				   COALESCE(a.focus_keyword, ''), COALESCE(a.language_code, 'en')
			FROM articles a
			WHERE a.status = 'published' 
			  AND a.published_at IS NOT NULL 
			  AND a.published_at >= $1
			ORDER BY a.published_at DESC
			LIMIT $2 OFFSET $3`
		rows, err = r.db.QueryContext(context.Background(), query, cutoffTime, limit, offset)
	} else {
		query = `
			SELECT a.id, a.title, a.slug, a.content, a.excerpt, a.author_id, a.category_id, 
				   a.status, a.published_at, a.created_at, a.updated_at, a.view_count, 
				   a.like_count, a.dislike_count, 
				   COALESCE(a.meta_title, ''), COALESCE(a.meta_description, ''), 
				   COALESCE(a.focus_keyword, ''), COALESCE(a.language_code, 'en')
			FROM articles a
			WHERE a.status = 'published' 
			  AND a.published_at IS NOT NULL 
			  AND a.published_at >= $1
			  AND a.language_code = $2
			ORDER BY a.published_at DESC
			LIMIT $3 OFFSET $4`
		rows, err = r.db.QueryContext(context.Background(), query, cutoffTime, languageCode, limit, offset)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to query articles: %w", err)
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		var article models.Article
		var metaTitle, metaDescription, focusKeyword string

		err := rows.Scan(
			&article.ID, &article.Title, &article.Slug, &article.Content, &article.Excerpt,
			&article.AuthorID, &article.CategoryID, &article.Status, &article.PublishedAt,
			&article.CreatedAt, &article.UpdatedAt, &article.ViewCount, &article.LikeCount,
			&article.DislikeCount, &metaTitle, &metaDescription, &focusKeyword, &article.LanguageCode,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}

		// Set SEO data from individual columns
		article.SEOData.MetaTitle = metaTitle
		article.SEOData.MetaDescription = metaDescription
		if focusKeyword != "" {
			article.SEOData.Keywords = []string{focusKeyword}
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
	var query string
	var count int64
	var err error
	
	// If languageCode is empty or "en", count all articles
	if languageCode == "" || languageCode == "en" {
		query = `
			SELECT COUNT(*)
			FROM articles
			WHERE status = 'published' 
			  AND published_at IS NOT NULL 
			  AND published_at >= $1`
		err = r.db.QueryRowContext(context.Background(), query, cutoffTime).Scan(&count)
	} else {
		query = `
			SELECT COUNT(*)
			FROM articles
			WHERE status = 'published' 
			  AND published_at IS NOT NULL 
			  AND published_at >= $1
			  AND language_code = $2`
		err = r.db.QueryRowContext(context.Background(), query, cutoffTime, languageCode).Scan(&count)
	}
	
	if err != nil {
		return 0, fmt.Errorf("failed to count articles: %w", err)
	}

	return count, nil
}

// getArticleTags retrieves tags for a specific article
func (r *ArticleRepository) getArticleTags(articleID uint64) ([]models.Tag, error) {
	query := `
		SELECT t.id, t.name, t.slug, t.description, t.keywords, t.color, 
			   t.created_at, t.language_code, t.translation_group_id
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
		var description sql.NullString
		var keywordsJSON sql.NullString
		var color sql.NullString
		var translationGroupID sql.NullInt64

		err := rows.Scan(
			&tag.ID, &tag.Name, &tag.Slug, &description, &keywordsJSON,
			&color, &tag.CreatedAt, &tag.LanguageCode, &translationGroupID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}

		// Handle nullable fields
		if description.Valid {
			tag.Description = description.String
		}
		if color.Valid {
			tag.Color = color.String
		} else {
			tag.Color = "#000000" // Default color
		}

		// Handle nullable translation group ID
		if translationGroupID.Valid {
			groupID := uint64(translationGroupID.Int64)
			tag.TranslationGroupID = &groupID
		}

		// Parse keywords JSON if present
		if keywordsJSON.Valid && keywordsJSON.String != "" {
			if err := json.Unmarshal([]byte(keywordsJSON.String), &tag.Keywords); err != nil {
				// If JSON parsing fails, treat as empty array
				tag.Keywords = []string{}
			}
		} else {
			tag.Keywords = []string{}
		}

		// Set UpdatedAt to CreatedAt since table doesn't have updated_at
		tag.UpdatedAt = tag.CreatedAt

		tags = append(tags, tag)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tag rows: %w", err)
	}

	return tags, nil
}

// PROFESSIONAL FIXES: Add tag management methods

// LoadArticleTags loads tags for an article
func (r *ArticleRepository) LoadArticleTags(ctx context.Context, articleID uint64) ([]models.Tag, error) {
	query := `
		SELECT t.id, t.name, t.slug, t.description, t.color, t.language_code, 
			   t.created_at, t.updated_at
		FROM tags t
		INNER JOIN article_tags at ON t.id = at.tag_id  
		WHERE at.article_id = $1
		ORDER BY t.name
	`
	
	rows, err := r.db.QueryContext(ctx, query, articleID)
	if err != nil {
		return nil, fmt.Errorf("failed to query article tags: %w", err)
	}
	defer rows.Close()
	
	var tags []models.Tag
	for rows.Next() {
		var tag models.Tag
		err := rows.Scan(
			&tag.ID,
			&tag.Name,
			&tag.Slug,
			&tag.Description,
			&tag.Color,
			&tag.LanguageCode,
			&tag.CreatedAt,
			&tag.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, tag)
	}
	
	return tags, nil
}

// AssociateTagWithArticle creates an association between an article and a tag
func (r *ArticleRepository) AssociateTagWithArticle(ctx context.Context, articleID, tagID uint64) error {
	query := `
		INSERT INTO article_tags (article_id, tag_id, created_at) 
		VALUES ($1, $2, NOW()) 
		ON CONFLICT (article_id, tag_id) DO NOTHING
	`
	
	_, err := r.db.ExecContext(ctx, query, articleID, tagID)
	if err != nil {
		return fmt.Errorf("failed to associate tag with article: %w", err)
	}
	
	return nil
}

// ClearArticleTags removes all tag associations for an article
func (r *ArticleRepository) ClearArticleTags(ctx context.Context, articleID uint64) error {
	query := `DELETE FROM article_tags WHERE article_id = $1`
	
	_, err := r.db.ExecContext(ctx, query, articleID)
	if err != nil {
		return fmt.Errorf("failed to clear article tags: %w", err)
	}
	
	return nil
}

// UpdateArticleTags updates all tag associations for an article
func (r *ArticleRepository) UpdateArticleTags(ctx context.Context, articleID uint64, tagIDs []uint64) error {
	// Start transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Clear existing associations
	_, err = tx.ExecContext(ctx, "DELETE FROM article_tags WHERE article_id = $1", articleID)
	if err != nil {
		return fmt.Errorf("failed to clear existing tags: %w", err)
	}
	
	// Add new associations
	if len(tagIDs) > 0 {
		for _, tagID := range tagIDs {
			_, err = tx.ExecContext(ctx, 
				"INSERT INTO article_tags (article_id, tag_id, created_at) VALUES ($1, $2, NOW())", 
				articleID, tagID)
			if err != nil {
				return fmt.Errorf("failed to associate tag %d: %w", tagID, err)
			}
		}
	}
	
	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

// UpdateArticleTagsByNames updates all tag associations for an article using tag names
func (r *ArticleRepository) UpdateArticleTagsByNames(ctx context.Context, articleID uint64, tagNames []string) error {
	if len(tagNames) == 0 {
		// Clear all tags if empty array provided
		return r.UpdateArticleTags(ctx, articleID, []uint64{})
	}
	
	// Get or create tag IDs from names
	tagIDs := make([]uint64, 0, len(tagNames))
	
	for _, tagName := range tagNames {
		if tagName == "" {
			continue // Skip empty tag names
		}
		
		// Try to find existing tag by name
		var tagID uint64
		err := r.db.QueryRowContext(ctx, 
			"SELECT id FROM tags WHERE name = $1 LIMIT 1", 
			tagName).Scan(&tagID)
		
		if err == sql.ErrNoRows {
			// Tag doesn't exist, create it
			slug := models.GenerateSlug(tagName)
			err = r.db.QueryRowContext(ctx,
				`INSERT INTO tags (name, slug, language_code, created_at) 
				 VALUES ($1, $2, 'fa', NOW()) 
				 RETURNING id`,
				tagName, slug).Scan(&tagID)
			if err != nil {
				return fmt.Errorf("failed to create tag '%s': %w", tagName, err)
			}
		} else if err != nil {
			return fmt.Errorf("failed to query tag '%s': %w", tagName, err)
		}
		
		tagIDs = append(tagIDs, tagID)
	}
	
	// Update article tags with the collected IDs
	return r.UpdateArticleTags(ctx, articleID, tagIDs)
}

// UpdateArticleCategories updates all category associations for an article
func (r *ArticleRepository) UpdateArticleCategories(ctx context.Context, articleID uint64, categoryIDs []uint64) error {
	// Start transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Clear existing category associations
	_, err = tx.ExecContext(ctx, "DELETE FROM article_categories WHERE article_id = $1", articleID)
	if err != nil {
		return fmt.Errorf("failed to clear existing categories: %w", err)
	}

	// Insert new category associations
	for _, categoryID := range categoryIDs {
		_, err = tx.ExecContext(ctx, 
			"INSERT INTO article_categories (article_id, category_id, created_at) VALUES ($1, $2, NOW())",
			articleID, categoryID)
		if err != nil {
			return fmt.Errorf("failed to associate category %d: %w", categoryID, err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// getArticleCategories retrieves categories for a specific article
func (r *ArticleRepository) getArticleCategories(articleID uint64) ([]models.Category, error) {
	query := `
		SELECT c.id, c.name, c.slug, c.description, c.language_code, 
			   c.created_at
		FROM categories c
		INNER JOIN article_categories ac ON c.id = ac.category_id
		WHERE ac.article_id = $1
		ORDER BY c.name`
	
	rows, err := r.db.Query(query, articleID)
	if err != nil {
		return nil, fmt.Errorf("failed to query article categories: %w", err)
	}
	defer rows.Close()
	
	var categories []models.Category
	for rows.Next() {
		var category models.Category
		err := rows.Scan(
			&category.ID,
			&category.Name,
			&category.Slug,
			&category.Description,
			&category.LanguageCode,
			&category.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, category)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating categories: %w", err)
	}
	
	return categories, nil
}

// LoadArticleCategories loads categories for a specific article (public method)
func (r *ArticleRepository) LoadArticleCategories(ctx context.Context, articleID uint64) ([]models.Category, error) {
	return r.getArticleCategories(articleID)
}

// ensurePartitionExists creates a partition for the given date if it doesn't exist
func (r *ArticleRepository) ensurePartitionExists(ctx context.Context, publishedAt *time.Time) error {
	var targetDate time.Time
	if publishedAt != nil {
		targetDate = *publishedAt
	} else {
		targetDate = time.Now()
	}
	
	// Format partition name
	partitionName := fmt.Sprintf("articles_%s", targetDate.Format("2006_01_02"))
	
	// Check if partition already exists
	var exists bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_class WHERE relname = $1
		)
	`, partitionName).Scan(&exists)
	
	if err != nil {
		return fmt.Errorf("failed to check partition existence: %w", err)
	}
	
	if exists {
		return nil // Partition already exists
	}
	
	// Create the partition
	startDate := targetDate.Format("2006-01-02")
	endDate := targetDate.AddDate(0, 0, 1).Format("2006-01-02")
	
	createSQL := fmt.Sprintf(`
		CREATE TABLE %s PARTITION OF articles 
		FOR VALUES FROM ('%s') TO ('%s')
	`, partitionName, startDate, endDate)
	
	_, err = r.db.ExecContext(ctx, createSQL)
	if err != nil {
		// Check if it's a "relation already exists" error (race condition)
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "42P07" {
			// Partition was created by another process, that's fine
			return nil
		}
		return fmt.Errorf("failed to create partition %s: %w", partitionName, err)
	}
	
	log.Printf("Created partition %s for date %s", partitionName, startDate)
	return nil
}