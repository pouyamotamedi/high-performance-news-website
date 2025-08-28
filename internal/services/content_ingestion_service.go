package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"high-performance-news-website/internal/auth"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/repositories"
)

// ContentIngestionService handles content ingestion business logic
type ContentIngestionService struct {
	ingestionRepo *repositories.ContentIngestionRepository
	articleRepo   *repositories.ArticleRepository
	userRepo      *repositories.UserRepository
	categoryRepo  *repositories.CategoryRepository
	tagRepo       *repositories.TagRepository
}

// NewContentIngestionService creates a new content ingestion service
func NewContentIngestionService(
	ingestionRepo *repositories.ContentIngestionRepository,
	articleRepo *repositories.ArticleRepository,
	userRepo *repositories.UserRepository,
	categoryRepo *repositories.CategoryRepository,
	tagRepo *repositories.TagRepository,
) *ContentIngestionService {
	return &ContentIngestionService{
		ingestionRepo: ingestionRepo,
		articleRepo:   articleRepo,
		userRepo:      userRepo,
		categoryRepo:  categoryRepo,
		tagRepo:       tagRepo,
	}
}

// IngestContent processes content from external sources
func (s *ContentIngestionService) IngestContent(ctx context.Context, sourceAPIKey string, request *models.ContentIngestionRequest) (*models.IngestedContent, error) {
	// 1. Validate and get content source
	source, err := s.ingestionRepo.GetContentSourceByAPIKey(ctx, sourceAPIKey)
	if err != nil {
		return nil, fmt.Errorf("invalid or inactive content source: %w", err)
	}
	
	// 2. Check rate limiting (basic implementation)
	if err := s.checkRateLimit(ctx, source); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}
	
	// 3. Create ingested content record
	content := &models.IngestedContent{
		SourceID:     source.ID,
		ExternalID:   request.ExternalID,
		Title:        request.Title,
		Content:      request.Content,
		Excerpt:      request.Excerpt,
		AuthorName:   request.AuthorName,
		AuthorEmail:  request.AuthorEmail,
		CategoryName: request.CategoryName,
		Tags:         request.Tags,
		PublishedAt:  request.PublishedAt,
		SourceURL:    request.SourceURL,
		Metadata:     request.Metadata,
	}
	
	// 4. Sanitize and validate content
	models.SanitizeIngestedContent(content)
	validationResult := models.ValidateIngestedContent(content)
	if !validationResult.IsValid {
		return nil, &models.ValidationError{
			Message: "Content validation failed",
			Fields:  validationResult.Errors,
		}
	}
	
	// 5. Check for duplicates
	duplicateResult, err := s.ingestionRepo.CheckDuplicateContent(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("failed to check for duplicates: %w", err)
	}
	
	if duplicateResult.IsDuplicate {
		content.Status = "duplicate"
		content.RejectionReason = fmt.Sprintf("Duplicate content detected (match: %s, similarity: %.2f)", 
			duplicateResult.MatchType, duplicateResult.Similarity)
	}
	
	// 6. Prepare for database insertion
	content.PrepareForProcessing()
	
	// 7. Save to database
	createdContent, err := s.ingestionRepo.CreateIngestedContent(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("failed to save ingested content: %w", err)
	}
	
	// 8. Auto-process if configured and not duplicate
	if source.Config.AutoPublish && createdContent.Status == "pending" {
		go func() {
			// Process asynchronously to avoid blocking the API response
			if err := s.ProcessPendingContent(context.Background(), createdContent.ID); err != nil {
				// Log error but don't fail the ingestion
				fmt.Printf("Failed to auto-process content %d: %v\n", createdContent.ID, err)
			}
		}()
	}
	
	return createdContent, nil
}

// ProcessPendingContent processes pending ingested content into articles
func (s *ContentIngestionService) ProcessPendingContent(ctx context.Context, contentID uint64) error {
	// Get pending content
	contents, err := s.ingestionRepo.GetPendingContent(ctx, 1)
	if err != nil {
		return fmt.Errorf("failed to get pending content: %w", err)
	}
	
	var content *models.IngestedContent
	for _, c := range contents {
		if c.ID == contentID {
			content = &c
			break
		}
	}
	
	if content == nil {
		return fmt.Errorf("content not found or not pending")
	}
	
	// Get source configuration
	source, err := s.ingestionRepo.GetContentSourceByID(ctx, content.SourceID)
	if err != nil {
		return fmt.Errorf("failed to get content source: %w", err)
	}
	
	// Create article from ingested content
	article, err := s.convertToArticle(ctx, content, source)
	if err != nil {
		// Mark as rejected
		s.ingestionRepo.UpdateIngestedContentStatus(ctx, content.ID, "rejected", nil, err.Error())
		return fmt.Errorf("failed to convert to article: %w", err)
	}
	
	// Create the article
	createdArticle, err := s.articleRepo.Create(ctx, article)
	if err != nil {
		// Mark as rejected
		s.ingestionRepo.UpdateIngestedContentStatus(ctx, content.ID, "rejected", nil, err.Error())
		return fmt.Errorf("failed to create article: %w", err)
	}
	
	// Mark as processed
	err = s.ingestionRepo.UpdateIngestedContentStatus(ctx, content.ID, "processed", &createdArticle.ID, "")
	if err != nil {
		return fmt.Errorf("failed to update content status: %w", err)
	}
	
	return nil
}

// ProcessBatchContent processes multiple pending content items
func (s *ContentIngestionService) ProcessBatchContent(ctx context.Context, limit int) (int, error) {
	contents, err := s.ingestionRepo.GetPendingContent(ctx, limit)
	if err != nil {
		return 0, fmt.Errorf("failed to get pending content: %w", err)
	}
	
	processed := 0
	for _, content := range contents {
		if err := s.ProcessPendingContent(ctx, content.ID); err != nil {
			// Log error but continue processing other items
			fmt.Printf("Failed to process content %d: %v\n", content.ID, err)
			continue
		}
		processed++
	}
	
	return processed, nil
}

// GetIngestionStats retrieves ingestion statistics
func (s *ContentIngestionService) GetIngestionStats(ctx context.Context, sourceID *uint64, hours int) (map[string]int, error) {
	stats, err := s.ingestionRepo.GetIngestionStats(ctx, sourceID, hours)
	if err != nil {
		return nil, fmt.Errorf("failed to get ingestion stats: %w", err)
	}
	
	return stats, nil
}

// CreateContentSource creates a new content source
func (s *ContentIngestionService) CreateContentSource(ctx context.Context, source *models.ContentSource, currentUser *models.User) (*models.ContentSource, error) {
	// Check permissions - only admins can create content sources
	if currentUser == nil || !currentUser.HasPermission("admin") {
		return nil, auth.ErrInsufficientPermissions
	}
	
	// Validate source
	if err := s.validateContentSource(source); err != nil {
		return nil, err
	}
	
	// Generate API key if not provided
	if source.APIKey == "" {
		source.APIKey = s.generateAPIKey()
	}
	
	// Set defaults
	if source.RateLimit == 0 {
		source.RateLimit = 100 // 100 requests per minute default
	}
	
	if source.Priority == 0 {
		source.Priority = 5 // Medium priority default
	}
	
	// Create source
	createdSource, err := s.ingestionRepo.CreateContentSource(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("failed to create content source: %w", err)
	}
	
	return createdSource, nil
}

// ListContentSources retrieves content sources with pagination
func (s *ContentIngestionService) ListContentSources(ctx context.Context, limit, offset int, currentUser *models.User) ([]models.ContentSource, int, error) {
	// Check permissions - only admins can list content sources
	if currentUser == nil || !currentUser.HasPermission("admin") {
		return nil, 0, auth.ErrInsufficientPermissions
	}
	
	sources, total, err := s.ingestionRepo.ListContentSources(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list content sources: %w", err)
	}
	
	return sources, total, nil
}

// Private helper methods

func (s *ContentIngestionService) checkRateLimit(ctx context.Context, source *models.ContentSource) error {
	// Basic rate limiting implementation
	// In production, use Redis or similar for distributed rate limiting
	
	// For now, just check if source is active
	if !source.IsActive {
		return fmt.Errorf("content source is inactive")
	}
	
	// TODO: Implement proper rate limiting with Redis/cache
	return nil
}

func (s *ContentIngestionService) convertToArticle(ctx context.Context, content *models.IngestedContent, source *models.ContentSource) (*models.Article, error) {
	article := &models.Article{
		Title:   content.Title,
		Content: content.Content,
		Excerpt: content.Excerpt,
		Status:  "draft", // Default to draft
	}
	
	// Set author
	if source.Config.DefaultAuthorID != 0 {
		article.AuthorID = source.Config.DefaultAuthorID
	} else {
		// Try to find author by name or email
		if content.AuthorName != "" || content.AuthorEmail != "" {
			// TODO: Implement author lookup/creation
			// For now, use default author
			article.AuthorID = 1 // System user
		} else {
			article.AuthorID = 1 // System user
		}
	}
	
	// Set category
	if source.Config.DefaultCategoryID != 0 {
		article.CategoryID = source.Config.DefaultCategoryID
	} else if content.CategoryName != "" {
		// Try to find or create category
		categoryID, err := s.findOrCreateCategory(ctx, content.CategoryName)
		if err != nil {
			return nil, fmt.Errorf("failed to handle category: %w", err)
		}
		article.CategoryID = categoryID
	} else {
		// Use default category (assuming ID 1 exists)
		article.CategoryID = 1
	}
	
	// Set published date
	if content.PublishedAt != nil {
		article.PublishedAt = content.PublishedAt
	}
	
	// Auto-publish if configured
	if source.Config.AutoPublish {
		article.Status = "published"
		if article.PublishedAt == nil {
			now := time.Now()
			article.PublishedAt = &now
		}
	}
	
	// Generate SEO data
	article.SEOData = models.SEOData{
		MetaTitle:       content.Title,
		MetaDescription: content.Excerpt,
		SchemaType:      "NewsArticle",
	}
	
	if content.SourceURL != "" {
		article.SEOData.CanonicalURL = content.SourceURL
	}
	
	// TODO: Handle tags
	// For now, skip tag processing
	
	return article, nil
}

func (s *ContentIngestionService) findOrCreateCategory(ctx context.Context, categoryName string) (uint64, error) {
	// TODO: Implement category lookup/creation
	// For now, return default category
	return 1, nil
}

func (s *ContentIngestionService) validateContentSource(source *models.ContentSource) error {
	var errors []string
	
	if strings.TrimSpace(source.Name) == "" {
		errors = append(errors, "name is required")
	}
	
	if len(source.Name) > 100 {
		errors = append(errors, "name must be less than 100 characters")
	}
	
	validTypes := map[string]bool{
		"api":     true,
		"webhook": true,
		"manual":  true,
	}
	
	if !validTypes[source.Type] {
		errors = append(errors, "type must be one of: api, webhook, manual")
	}
	
	if source.RateLimit < 0 || source.RateLimit > 10000 {
		errors = append(errors, "rate_limit must be between 0 and 10000")
	}
	
	if source.Priority < 1 || source.Priority > 10 {
		errors = append(errors, "priority must be between 1 and 10")
	}
	
	if len(errors) > 0 {
		return &models.ValidationError{
			Message: "Content source validation failed",
			Fields:  errors,
		}
	}
	
	return nil
}

func (s *ContentIngestionService) generateAPIKey() string {
	// Generate a random API key - using timestamp and nanoseconds for uniqueness
	timestamp := time.Now().Unix()
	nanoseconds := time.Now().UnixNano()
	
	// Create a more unique key by combining timestamp, nanoseconds, and a counter
	return fmt.Sprintf("ci_%d_%d_%d", timestamp, nanoseconds, nanoseconds%1000000)
}

// ValidateContentRequest validates a content ingestion request
func (s *ContentIngestionService) ValidateContentRequest(request *models.ContentIngestionRequest) error {
	var errors []string
	
	if strings.TrimSpace(request.ExternalID) == "" {
		errors = append(errors, "external_id is required")
	}
	
	if len(request.ExternalID) > 255 {
		errors = append(errors, "external_id must be less than 255 characters")
	}
	
	if strings.TrimSpace(request.Title) == "" {
		errors = append(errors, "title is required")
	}
	
	if len(request.Title) > 255 {
		errors = append(errors, "title must be less than 255 characters")
	}
	
	if strings.TrimSpace(request.Content) == "" {
		errors = append(errors, "content is required")
	}
	
	if len(request.Excerpt) > 500 {
		errors = append(errors, "excerpt must be less than 500 characters")
	}
	
	if len(request.AuthorName) > 100 {
		errors = append(errors, "author_name must be less than 100 characters")
	}
	
	if request.AuthorEmail != "" && !models.IsValidEmail(request.AuthorEmail) {
		errors = append(errors, "author_email must be a valid email address")
	}
	
	if len(request.AuthorEmail) > 255 {
		errors = append(errors, "author_email must be less than 255 characters")
	}
	
	if len(request.CategoryName) > 100 {
		errors = append(errors, "category_name must be less than 100 characters")
	}
	
	if request.SourceURL != "" && !models.IsValidURL(request.SourceURL) {
		errors = append(errors, "source_url must be a valid URL")
	}
	
	if len(request.SourceURL) > 500 {
		errors = append(errors, "source_url must be less than 500 characters")
	}
	
	if len(errors) > 0 {
		return &models.ValidationError{
			Message: "Content ingestion request validation failed",
			Fields:  errors,
		}
	}
	
	return nil
}