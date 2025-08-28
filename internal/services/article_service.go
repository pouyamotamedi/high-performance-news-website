package services

import (
	"context"
	"fmt"
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
}

// NewArticleService creates a new article service
func NewArticleService(db *database.DB, repo *repositories.ArticleRepository, autoLinkService *AutoLinkingService) *ArticleService {
	return &ArticleService{
		repo:            repo,
		db:              db,
		autoLinkService: autoLinkService,
	}
}

// ArticleFilters represents filters for article listing
type ArticleFilters struct {
	Status     string    `json:"status,omitempty"`
	CategoryID *uint64   `json:"category_id,omitempty"`
	AuthorID   *uint64   `json:"author_id,omitempty"`
	Search     string    `json:"search,omitempty"`
	Tags       []uint64  `json:"tags,omitempty"`
	DateFrom   string    `json:"date_from,omitempty"`
	DateTo     string    `json:"date_to,omitempty"`
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
		// Regenerate slug if title changed
		article.Slug = models.GenerateSlug(*updateReq.Title)
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
		article.SEOData = *updateReq.SEOData
	}

	if updateReq.AutoLinking != nil {
		article.AutoLinking = *updateReq.AutoLinking
	}

	article.UpdatedAt = time.Now()

	// Update article through repository
	err = s.repo.Update(ctx, article)
	if err != nil {
		return nil, fmt.Errorf("failed to update article: %w", err)
	}

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

	return nil
}

// List retrieves articles with pagination, filtering, and sorting
func (s *ArticleService) List(ctx context.Context, limit, offset int, filters ArticleFilters, sortBy, sortOrder string) ([]models.Article, int, error) {
	// Build query based on filters
	query := `
		SELECT a.id, a.title, a.slug, a.excerpt, a.author_id, a.category_id,
			   a.status, a.published_at, a.created_at, a.updated_at, a.view_count,
			   a.like_count, a.dislike_count
		FROM articles a
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
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan article: %w", err)
		}
		articles = append(articles, article)
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
	Title       *string         `json:"title,omitempty"`
	Content     *string         `json:"content,omitempty"`
	Excerpt     *string         `json:"excerpt,omitempty"`
	CategoryID  *uint64         `json:"category_id,omitempty"`
	Status      *string         `json:"status,omitempty"`
	SEOData     *models.SEOData `json:"seo_data,omitempty"`
	AutoLinking *bool           `json:"auto_linking,omitempty"`
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