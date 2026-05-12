package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"high-performance-news-website/internal/auth"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/repositories"
	"high-performance-news-website/pkg/database"
)

// ArticleService handles article-related business logic
type ArticleService struct {
	repo            *repositories.ArticleRepository
	db              *database.DB
	autoLinkService *AutoLinkingService
	staticGenerator *StaticGenerator
	mediaService    *MediaService
	searchService   SearchServiceInterface
}

// NewArticleService creates a new article service
func NewArticleService(db *database.DB, repo *repositories.ArticleRepository, autoLinkService *AutoLinkingService) *ArticleService {
	return &ArticleService{
		repo:            repo,
		db:              db,
		autoLinkService: autoLinkService,
	}
}

// SetMediaService sets the media service for image cleanup operations
func (s *ArticleService) SetMediaService(ms *MediaService) {
	s.mediaService = ms
}

// SetStaticGenerator sets the static generator for automatic static file regeneration
func (s *ArticleService) SetStaticGenerator(sg *StaticGenerator) {
	log.Printf("DEBUG: SetStaticGenerator called, sg is nil: %v", sg == nil)
	s.staticGenerator = sg
	log.Printf("DEBUG: staticGenerator set, s.staticGenerator is nil: %v", s.staticGenerator == nil)
}

// SetSearchService sets the search service for real-time search indexing
func (s *ArticleService) SetSearchService(ss SearchServiceInterface) {
	s.searchService = ss
}

// triggerSearchIndexing indexes an article in the search engine (MeiliSearch)
// This is called when articles are created, updated, or published
// Search indexing failure does NOT affect the article operation (fire-and-forget with retry)
func (s *ArticleService) triggerSearchIndexing(article *models.Article) {
	if s.searchService == nil || article == nil {
		return
	}

	// Only index published articles
	if article.Status != "published" {
		// If article was unpublished, remove from index
		go func() {
			ctx := context.Background()
			if err := s.searchService.RemoveArticle(ctx, article.ID); err != nil {
				log.Printf("[SEARCH_INDEX] Warning: Failed to remove unpublished article %d from search index: %v", article.ID, err)
			}
		}()
		return
	}

	// Run in background with retry mechanism to not block the main operation
	go func() {
		maxRetries := 3
		retryDelay := 1 * time.Second

		for attempt := 1; attempt <= maxRetries; attempt++ {
			if err := s.searchService.IndexArticle(article); err != nil {
				log.Printf("[SEARCH_INDEX] Attempt %d/%d failed for article %d: %v", attempt, maxRetries, article.ID, err)

				if attempt < maxRetries {
					time.Sleep(retryDelay)
					retryDelay *= 2 // Exponential backoff
				} else {
					log.Printf("[SEARCH_INDEX] FAILED after %d attempts for article %d: %s - marking for manual reindex",
						maxRetries, article.ID, article.Title)
					// In production, this could write to a failed_indexing table for later retry
				}
			} else {
				log.Printf("[SEARCH_INDEX] Successfully indexed article %d: %s (attempt %d)", article.ID, article.Title, attempt)
				return
			}
		}
	}()
}

// triggerStaticRegeneration triggers static file regeneration in background
func (s *ArticleService) triggerStaticRegeneration(article *models.Article) {
	log.Printf("DEBUG: triggerStaticRegeneration called for article: %v", article)
	log.Printf("DEBUG: staticGenerator is nil: %v", s.staticGenerator == nil)

	if s.staticGenerator == nil || article == nil {
		log.Printf("DEBUG: Skipping static regeneration - staticGenerator nil: %v, article nil: %v", s.staticGenerator == nil, article == nil)
		return
	}

	// Only regenerate for published articles
	if article.Status != "published" {
		log.Printf("DEBUG: Skipping static regeneration - article status is: %s", article.Status)
		return
	}

	log.Printf("DEBUG: Starting static regeneration for article: %s (ID: %d)", article.Slug, article.ID)

	// Run in background to not block the main operation
	go func() {
		ctx := context.Background()
		if err := s.staticGenerator.RegenerateOnContentUpdate(ctx, article); err != nil {
			log.Printf("Warning: Failed to regenerate static files for article %d: %v", article.ID, err)
		} else {
			log.Printf("Static files regenerated for article: %s", article.Slug)
		}
	}()
}

// ArticleFilters represents filters for article listing
type ArticleFilters struct {
	Status       string   `json:"status,omitempty"`
	CategoryID   *uint64  `json:"category_id,omitempty"`
	AuthorID     *uint64  `json:"author_id,omitempty"`
	Search       string   `json:"search,omitempty"`
	Tags         []uint64 `json:"tags,omitempty"`
	TagID        *uint64  `json:"tag_id,omitempty"`
	DateFrom     string   `json:"date_from,omitempty"`
	DateTo       string   `json:"date_to,omitempty"`
	LanguageCode string   `json:"language_code,omitempty"`
}

// Create creates a new article with permission checking
func (s *ArticleService) Create(ctx context.Context, article *models.Article, currentUser *models.User) (*models.Article, error) {
	// Check permissions - users need create permission
	if currentUser != nil && !currentUser.HasPermission("create") {
		return nil, auth.ErrInsufficientPermissions
	}

	// Set author to current user
	article.AuthorID = currentUser.ID

	// Set timestamps
	now := time.Now()
	article.CreatedAt = now
	article.UpdatedAt = now

	// Set published_at if status is published
	if article.Status == "published" {
		article.PublishedAt = &now
	}

	// Process auto-linking if enabled and service is available
	if s.autoLinkService != nil && article.AutoLinking {
		processedContent, err := s.autoLinkService.ProcessHTMLContent(ctx, article)
		if err != nil {
			// Log error but don't fail the creation
			fmt.Printf("Warning: Auto-linking failed: %v\n", err)
		} else {
			article.Content = processedContent
		}
	}

	// Create article through repository
	createdArticle, err := s.repo.Create(ctx, article)
	if err != nil {
		return nil, fmt.Errorf("failed to create article: %w", err)
	}

	// Trigger static file regeneration for published articles
	s.triggerStaticRegeneration(createdArticle)

	// Trigger search indexing for published articles
	s.triggerSearchIndexing(createdArticle)

	return createdArticle, nil
}

// GetByID retrieves an article by ID
func (s *ArticleService) GetByID(ctx context.Context, id uint64) (*models.Article, error) {
	article, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get article: %w", err)
	}

	return article, nil
}

// GetBySlug retrieves an article by slug
func (s *ArticleService) GetBySlug(ctx context.Context, slug string) (*models.Article, error) {
	article, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to get article by slug: %w", err)
	}

	return article, nil
}

// GetAvailableTranslations returns all available translations for an article
// This is used for generating correct hreflang tags
func (s *ArticleService) GetAvailableTranslations(ctx context.Context, articleID uint64) ([]models.Article, error) {
	// First get the article to find its translation_group_id
	article, err := s.repo.GetByID(ctx, articleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get article: %w", err)
	}
	
	// Use translation_group_id if available, otherwise use article's own ID
	// This handles the case where the article is the "master" (no translation_group_id)
	// but other articles reference it via their translation_group_id
	groupID := articleID
	if article.TranslationGroupID != nil {
		groupID = *article.TranslationGroupID
	}
	
	// Get all articles in the same translation group
	return s.repo.GetByTranslationGroup(ctx, groupID)
}

// GetAvailableTranslationsBySlug returns all available translations for an article by slug
func (s *ArticleService) GetAvailableTranslationsBySlug(ctx context.Context, slug string) ([]models.Article, error) {
	article, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to get article by slug: %w", err)
	}
	
	return s.GetAvailableTranslations(ctx, article.ID)
}

// Update updates an existing article with permission checking
func (s *ArticleService) Update(ctx context.Context, id uint64, req interface{}, currentUser *models.User) (*models.Article, error) {
	// Get existing article
	article, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get article: %w", err)
	}

	// Check permissions - users can update their own articles, editors can update any
	if currentUser != nil {
		if article.AuthorID != currentUser.ID && !currentUser.HasPermission("update") {
			return nil, auth.ErrInsufficientPermissions
		}
	}

	// Type assertion for update request (this would be properly typed in real implementation)
	updateReq, ok := req.(*UpdateArticleRequest)
	if !ok {
		return nil, fmt.Errorf("invalid update request type")
	}

	// Update fields if provided
	if updateReq.Title != nil {
		article.Title = *updateReq.Title
		// Only regenerate slug if no custom slug is provided
		if updateReq.Slug == nil || *updateReq.Slug == "" {
			article.Slug = models.GenerateSlug(*updateReq.Title)
		}
	}

	if updateReq.Slug != nil && *updateReq.Slug != "" {
		article.Slug = *updateReq.Slug
	}

	if updateReq.Content != nil {
		article.Content = *updateReq.Content

		// Process auto-linking if enabled and service is available
		if s.autoLinkService != nil && article.AutoLinking {
			processedContent, err := s.autoLinkService.ProcessHTMLContent(ctx, article)
			if err != nil {
				// Log error but don't fail the update
				fmt.Printf("Warning: Auto-linking failed: %v\n", err)
			} else {
				article.Content = processedContent
			}
		}
	}

	if updateReq.Excerpt != nil {
		article.Excerpt = *updateReq.Excerpt
	}

	if updateReq.CategoryID != nil {
		article.CategoryID = *updateReq.CategoryID
	}

	// Handle multiple categories (CategoryIDs takes precedence over CategoryID)
	if len(updateReq.CategoryIDs) > 0 {
		// Set primary category to first one for backward compatibility
		article.CategoryID = updateReq.CategoryIDs[0]

		// Update article categories in junction table
		err = s.repo.UpdateArticleCategories(ctx, id, updateReq.CategoryIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to update article categories: %w", err)
		}
	}

	if updateReq.Status != nil {
		oldStatus := article.Status
		article.Status = *updateReq.Status

		// Set published_at when changing from draft to published
		if oldStatus != "published" && *updateReq.Status == "published" {
			now := time.Now()
			article.PublishedAt = &now
		}
	}

	if updateReq.SEOData != nil {
		// Update individual SEO fields from SEOData
		article.MetaTitle = updateReq.SEOData.MetaTitle
		article.MetaDescription = updateReq.SEOData.MetaDescription
		article.FocusKeyword = updateReq.SEOData.FocusKeyword
		article.CanonicalURL = updateReq.SEOData.CanonicalURL
		article.SchemaType = updateReq.SEOData.SchemaType
	}

	if updateReq.AutoLinking != nil {
		article.AutoLinking = *updateReq.AutoLinking
	}

	// Handle language code update
	if updateReq.LanguageCode != nil && *updateReq.LanguageCode != "" {
		article.LanguageCode = *updateReq.LanguageCode
	}

	// Handle translation group ID update
	if updateReq.TranslationGroupID != nil {
		article.TranslationGroupID = updateReq.TranslationGroupID
	}

	// Handle featured image update
	if updateReq.FeaturedImageID != nil {
		if *updateReq.FeaturedImageID == 0 {
			// If 0 is passed, remove the featured image
			article.FeaturedImageID = nil
		} else {
			// Set the new featured image ID
			article.FeaturedImageID = updateReq.FeaturedImageID
		}
	}

	article.UpdatedAt = time.Now()

	// Update article through repository
	err = s.repo.Update(ctx, article)
	if err != nil {
		return nil, fmt.Errorf("failed to update article: %w", err)
	}

	// Handle tags update if provided
	if updateReq.Tags != nil {
		err = s.repo.UpdateArticleTagsByNames(ctx, article.ID, updateReq.Tags)
		if err != nil {
			return nil, fmt.Errorf("failed to update article tags: %w", err)
		}

		// Reload article with updated tags
		updatedArticle, err := s.repo.GetByID(ctx, article.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to reload article with tags: %w", err)
		}

		// Trigger static file regeneration
		s.triggerStaticRegeneration(updatedArticle)

		// Trigger search indexing
		s.triggerSearchIndexing(updatedArticle)

		return updatedArticle, nil
	}

	// Trigger static file regeneration
	s.triggerStaticRegeneration(article)

	// Trigger search indexing
	s.triggerSearchIndexing(article)

	return article, nil
}

// Delete deletes an article with permission checking
func (s *ArticleService) Delete(ctx context.Context, id uint64, currentUser *models.User) error {
	// Get existing article
	article, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get article: %w", err)
	}

	// Check permissions - users can delete their own articles, editors can delete any
	if currentUser != nil {
		if article.AuthorID != currentUser.ID && !currentUser.HasPermission("delete") {
			return auth.ErrInsufficientPermissions
		}
	}

	// Delete article through repository (soft delete)
	err = s.repo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete article: %w", err)
	}

	// Remove from search index
	if s.searchService != nil {
		go func() {
			ctx := context.Background()
			if err := s.searchService.RemoveArticle(ctx, id); err != nil {
				log.Printf("Warning: Failed to remove deleted article %d from search index: %v", id, err)
			}
		}()
	}

	return nil
}

// PermanentDelete permanently deletes an article from the database (hard delete)
func (s *ArticleService) PermanentDelete(ctx context.Context, id uint64, currentUser *models.User) error {
	// Get existing article first to check permissions
	article, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get article: %w", err)
	}

	// Check permissions - only editors/admins can permanently delete
	if currentUser != nil && !currentUser.HasPermission("delete") {
		return auth.ErrInsufficientPermissions
	}

	// Log the permanent delete operation
	log.Printf("Performing permanent delete for article ID: %d, Title: %s", id, article.Title)

	// Clean up associated featured image and all its variants
	if article.FeaturedImageID != nil && s.mediaService != nil {
		log.Printf("Cleaning up featured image ID: %d for article ID: %d", *article.FeaturedImageID, id)
		if err := s.mediaService.DeleteMedia(*article.FeaturedImageID); err != nil {
			log.Printf("Warning: Failed to delete featured image %d: %v", *article.FeaturedImageID, err)
			// Continue with article deletion even if image cleanup fails
		} else {
			log.Printf("Successfully deleted featured image %d and all variants", *article.FeaturedImageID)
		}
	}

	// Clean up any other images associated with this article
	if s.mediaService != nil {
		articleImages, err := s.getArticleImages(ctx, id)
		if err != nil {
			log.Printf("Warning: Failed to get article images: %v", err)
		} else {
			for _, imgID := range articleImages {
				if err := s.mediaService.DeleteMedia(imgID); err != nil {
					log.Printf("Warning: Failed to delete article image %d: %v", imgID, err)
				}
			}
		}
	}

	// Execute hard delete using raw SQL
	query := `DELETE FROM articles WHERE id = $1`
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to permanently delete article: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("article with ID %d not found or already deleted", id)
	}

	log.Printf("Successfully permanently deleted article ID: %d", id)
	return nil
}

// getArticleImages retrieves all image IDs associated with an article
func (s *ArticleService) getArticleImages(ctx context.Context, articleID uint64) ([]uint64, error) {
	query := `SELECT id FROM images WHERE article_id = $1`
	rows, err := s.db.QueryContext(ctx, query, articleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var imageIDs []uint64
	for rows.Next() {
		var id uint64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		imageIDs = append(imageIDs, id)
	}

	return imageIDs, nil
}

// List retrieves articles with pagination, filtering, and sorting
func (s *ArticleService) List(ctx context.Context, limit, offset int, filters ArticleFilters, sortBy, sortOrder string) ([]models.Article, int, error) {
	// Build query based on filters
	query := `
		SELECT a.id, a.title, a.slug, a.excerpt, a.author_id, a.category_id,
			   a.status, a.published_at, a.created_at, a.updated_at, a.view_count,
			   a.like_count, a.dislike_count, a.featured_image_id,
			   a.language_code, a.translation_group_id,
			   CASE 
			       WHEN i.original_url LIKE '/uploads/%' THEN i.original_url
			       WHEN i.filename IS NOT NULL AND i.filename != '' THEN '/static/media/articles/' || i.filename
			       ELSE NULL 
			   END as featured_image
		FROM articles a
		LEFT JOIN images i ON a.featured_image_id = i.id
		WHERE 1=1`

	args := []interface{}{}
	argIndex := 1

	// Apply filters
	if filters.Status != "" {
		query += fmt.Sprintf(" AND a.status = $%d", argIndex)
		args = append(args, filters.Status)
		argIndex++
	}

	if filters.CategoryID != nil {
		query += fmt.Sprintf(" AND a.category_id = $%d", argIndex)
		args = append(args, *filters.CategoryID)
		argIndex++
	}

	if filters.AuthorID != nil {
		query += fmt.Sprintf(" AND a.author_id = $%d", argIndex)
		args = append(args, *filters.AuthorID)
		argIndex++
	}

	if filters.Search != "" {
		query += fmt.Sprintf(" AND (a.title ILIKE $%d OR a.content ILIKE $%d)", argIndex, argIndex+1)
		searchTerm := "%" + filters.Search + "%"
		args = append(args, searchTerm, searchTerm)
		argIndex += 2
	}

	if filters.DateFrom != "" {
		query += fmt.Sprintf(" AND a.published_at >= $%d", argIndex)
		args = append(args, filters.DateFrom)
		argIndex++
	}

	if filters.DateTo != "" {
		query += fmt.Sprintf(" AND a.published_at <= $%d", argIndex)
		args = append(args, filters.DateTo)
		argIndex++
	}

	// Filter by language code
	if filters.LanguageCode != "" {
		query += fmt.Sprintf(" AND a.language_code = $%d", argIndex)
		args = append(args, filters.LanguageCode)
		argIndex++
	}

	// Filter by tag ID (using article_tags junction table)
	if filters.TagID != nil {
		query += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM article_tags at WHERE at.article_id = a.id AND at.tag_id = $%d)", argIndex)
		args = append(args, *filters.TagID)
		argIndex++
	}

	// Apply sorting
	validSortFields := map[string]bool{
		"id":           true,
		"title":        true,
		"published_at": true,
		"created_at":   true,
		"updated_at":   true,
		"view_count":   true,
		"like_count":   true,
	}

	if !validSortFields[sortBy] {
		sortBy = "published_at"
	}

	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	query += fmt.Sprintf(" ORDER BY a.%s %s", sortBy, sortOrder)

	// Add pagination
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	// Execute query
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query articles: %w", err)
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		var article models.Article
		var featuredImageURL sql.NullString
		var translationGroupID sql.NullInt64
		err := rows.Scan(
			&article.ID,
			&article.Title,
			&article.Slug,
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
			&article.FeaturedImageID,
			&article.LanguageCode,
			&translationGroupID,
			&featuredImageURL,
		)
		if err == nil && featuredImageURL.Valid {
			article.FeaturedImage = featuredImageURL.String
		}
		if err == nil && translationGroupID.Valid {
			tgid := uint64(translationGroupID.Int64)
			article.TranslationGroupID = &tgid
		}
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan article: %w", err)
		}
		articles = append(articles, article)
	}

	// Load categories for each article
	for i := range articles {
		categories, err := s.repo.LoadArticleCategories(ctx, articles[i].ID)
		if err != nil {
			log.Printf("Warning: Failed to load categories for article %d: %v", articles[i].ID, err)
			articles[i].Categories = []models.Category{} // Empty array instead of null
		} else {
			articles[i].Categories = categories
		}
	}

	// Get total count
	countQuery := `SELECT COUNT(*) FROM articles a WHERE 1=1`
	countArgs := args[:len(args)-2] // Remove LIMIT and OFFSET args

	// Rebuild count query with same filters
	argIndex = 1
	if filters.Status != "" {
		countQuery += fmt.Sprintf(" AND a.status = $%d", argIndex)
		argIndex++
	}
	if filters.CategoryID != nil {
		countQuery += fmt.Sprintf(" AND a.category_id = $%d", argIndex)
		argIndex++
	}
	if filters.AuthorID != nil {
		countQuery += fmt.Sprintf(" AND a.author_id = $%d", argIndex)
		argIndex++
	}
	if filters.Search != "" {
		countQuery += fmt.Sprintf(" AND (a.title ILIKE $%d OR a.content ILIKE $%d)", argIndex, argIndex+1)
		argIndex += 2
	}
	if filters.DateFrom != "" {
		countQuery += fmt.Sprintf(" AND a.published_at >= $%d", argIndex)
		argIndex++
	}
	if filters.DateTo != "" {
		countQuery += fmt.Sprintf(" AND a.published_at <= $%d", argIndex)
		argIndex++
	}
	if filters.LanguageCode != "" {
		countQuery += fmt.Sprintf(" AND a.language_code = $%d", argIndex)
		argIndex++
	}
	if filters.TagID != nil {
		countQuery += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM article_tags at WHERE at.article_id = a.id AND at.tag_id = $%d)", argIndex)
		argIndex++
	}

	var total int
	err = s.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count articles: %w", err)
	}

	return articles, total, nil
}

// Publish publishes a draft article
func (s *ArticleService) Publish(ctx context.Context, id uint64, currentUser *models.User) (*models.Article, error) {
	// Get existing article
	article, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get article: %w", err)
	}

	// Check permissions - users can publish their own articles, editors can publish any
	if currentUser != nil {
		if article.AuthorID != currentUser.ID && !currentUser.HasPermission("publish") {
			return nil, auth.ErrInsufficientPermissions
		}
	}

	// Check if article is already published
	if article.Status == "published" {
		return article, nil
	}

	// Update status and published_at
	article.Status = "published"
	now := time.Now()
	article.PublishedAt = &now
	article.UpdatedAt = now

	// Update article through repository
	err = s.repo.Update(ctx, article)
	if err != nil {
		return nil, fmt.Errorf("failed to publish article: %w", err)
	}

	// Trigger static file regeneration for newly published article
	s.triggerStaticRegeneration(article)

	// Trigger search indexing for newly published article
	s.triggerSearchIndexing(article)

	return article, nil
}

// GetTrending retrieves trending articles
func (s *ArticleService) GetTrending(ctx context.Context, limit, hours int) ([]models.Article, error) {
	articles, err := s.repo.GetTrendingArticles(ctx, limit, hours)
	if err != nil {
		return nil, fmt.Errorf("failed to get trending articles: %w", err)
	}

	return articles, nil
}

// GetPopular retrieves popular articles by view count
func (s *ArticleService) GetPopular(ctx context.Context, limit, days int) ([]models.Article, error) {
	// Calculate date threshold
	dateThreshold := time.Now().AddDate(0, 0, -days)

	query := `
		SELECT a.id, a.title, a.slug, a.excerpt, a.author_id, a.category_id,
			   a.published_at, a.view_count, a.like_count, a.dislike_count
		FROM articles a
		WHERE a.status = 'published' 
		  AND a.published_at IS NOT NULL
		  AND a.published_at > $1
		ORDER BY a.view_count DESC, a.like_count DESC
		LIMIT $2`

	rows, err := s.db.QueryContext(ctx, query, dateThreshold, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query popular articles: %w", err)
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
			&article.LikeCount,
			&article.DislikeCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan popular article: %w", err)
		}
		articles = append(articles, article)
	}

	return articles, nil
}

// BulkCreate creates multiple articles with validation and permission checking
func (s *ArticleService) BulkCreate(ctx context.Context, articles []models.Article, currentUser *models.User) ([]models.Article, error) {
	// Check permissions - users need create permission
	if currentUser != nil && !currentUser.HasPermission("create") {
		return nil, auth.ErrInsufficientPermissions
	}

	// Validate article count
	if len(articles) == 0 {
		return nil, &models.ValidationError{
			Message: "No articles provided",
			Fields:  []string{"articles array cannot be empty"},
		}
	}

	if len(articles) > 1000 {
		return nil, &models.ValidationError{
			Message: "Too many articles",
			Fields:  []string{"maximum 1000 articles per request"},
		}
	}

	// Set common fields for all articles
	now := time.Now()
	for i := range articles {
		articles[i].AuthorID = currentUser.ID
		articles[i].CreatedAt = now
		articles[i].UpdatedAt = now

		// Set published_at if status is published
		if articles[i].Status == "published" {
			articles[i].PublishedAt = &now
		}
	}

	// Create articles through repository
	err := s.repo.BulkCreate(ctx, articles)
	if err != nil {
		return nil, fmt.Errorf("failed to bulk create articles: %w", err)
	}

	return articles, nil
}

// RecordView records an article view for analytics
func (s *ArticleService) RecordView(ctx context.Context, articleID uint64, ipAddress, userAgent, referer string) error {
	err := s.repo.RecordView(ctx, articleID, ipAddress, userAgent, referer)
	if err != nil {
		return fmt.Errorf("failed to record view: %w", err)
	}

	return nil
}

// UpdateArticleRequest represents an update request (temporary type for compilation)
type UpdateArticleRequest struct {
	Title              *string         `json:"title,omitempty"`
	Slug               *string         `json:"slug,omitempty"`
	Content            *string         `json:"content,omitempty"`
	Excerpt            *string         `json:"excerpt,omitempty"`
	CategoryID         *uint64         `json:"category_id,omitempty"`  // Backward compatibility
	CategoryIDs        []uint64        `json:"category_ids,omitempty"` // Multiple categories support
	Status             *string         `json:"status,omitempty"`
	Tags               []string        `json:"tags,omitempty"`
	FeaturedImageID    *uint64         `json:"featured_image_id,omitempty"`
	SEOData            *models.SEOData `json:"seo_data,omitempty"`
	AutoLinking        *bool           `json:"auto_linking,omitempty"`
	LanguageCode       *string         `json:"language_code,omitempty"`
	TranslationGroupID *uint64         `json:"translation_group_id,omitempty"`
}

// GetTotalCount returns the total number of articles
func (s *ArticleService) GetTotalCount() (int64, error) {
	return s.repo.GetTotalCount()
}

// GetPublishedTodayCount returns the count of articles published today
func (s *ArticleService) GetPublishedTodayCount() (int64, error) {
	return s.repo.GetPublishedTodayCount()
}

// GetPendingCount returns the count of pending articles
func (s *ArticleService) GetPendingCount() (int64, error) {
	return s.repo.GetPendingCount()
}

// GetDraftCount returns the count of draft articles
func (s *ArticleService) GetDraftCount() (int64, error) {
	return s.repo.GetDraftCount()
}

// GetPublishedCount returns the count of published articles
func (s *ArticleService) GetPublishedCount() (int64, error) {
	return s.repo.GetPublishedCount()
}

// HealthCheck checks if the article service is healthy
func (s *ArticleService) HealthCheck() error {
	// Simple health check - try to count articles
	_, err := s.GetTotalCount()
	return err
}

// AssociateTagWithArticle creates an association between an article and a tag
func (s *ArticleService) AssociateTagWithArticle(ctx context.Context, articleID, tagID uint64) error {
	// First check if association already exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM article_tags WHERE article_id = $1 AND tag_id = $2)`
	err := s.db.QueryRowContext(ctx, checkQuery, articleID, tagID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check existing association: %w", err)
	}

	if exists {
		log.Printf("Association between article %d and tag %d already exists", articleID, tagID)
		return nil
	}

	// Insert into article_tags table
	query := `INSERT INTO article_tags (article_id, tag_id, created_at) VALUES ($1, $2, NOW())`

	_, err = s.db.ExecContext(ctx, query, articleID, tagID)
	if err != nil {
		return fmt.Errorf("failed to associate tag %d with article %d: %w", tagID, articleID, err)
	}

	return nil
}

// GetDB returns the database connection for direct access (used for permanent delete)
func (s *ArticleService) GetDB() *database.DB {
	return s.db
}

// GetByCategory retrieves articles by category
func (s *ArticleService) GetByCategory(ctx context.Context, categoryID uint64, limit, offset int) ([]models.Article, error) {
	return s.repo.GetByCategory(ctx, categoryID, limit, offset)
}

// GetByTag retrieves articles by tag
func (s *ArticleService) GetByTag(ctx context.Context, tagID uint64, limit, offset int) ([]models.Article, error) {
	return s.repo.GetByTag(ctx, tagID, limit, offset)
}

// UpdateLikeCount updates the like count for an article
func (s *ArticleService) UpdateLikeCount(articleID uint64, newCount int) error {
	query := `UPDATE articles SET like_count = $1, updated_at = NOW() WHERE id = $2`

	_, err := s.db.Exec(query, newCount, articleID)
	if err != nil {
		return fmt.Errorf("failed to update like count for article %d: %w", articleID, err)
	}

	return nil
}

// UpdateDislikeCount updates the dislike count for an article
func (s *ArticleService) UpdateDislikeCount(articleID uint64, newCount int) error {
	query := `UPDATE articles SET dislike_count = $1, updated_at = NOW() WHERE id = $2`

	_, err := s.db.Exec(query, newCount, articleID)
	if err != nil {
		return fmt.Errorf("failed to update dislike count for article %d: %w", articleID, err)
	}

	return nil
}
