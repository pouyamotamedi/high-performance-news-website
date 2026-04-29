package services

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
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
	
	// Initialize metadata if needed
	if content.Metadata == nil {
		content.Metadata = make(map[string]interface{})
	}
	
	// Add featured image URL to metadata if provided
	fmt.Printf("DEBUG IngestContent: FeaturedImageURL from request: '%s'\n", request.FeaturedImageURL)
	if request.FeaturedImageURL != "" {
		content.Metadata["featured_image_url"] = request.FeaturedImageURL
		fmt.Printf("DEBUG IngestContent: Metadata after adding featured_image_url: %v\n", content.Metadata)
	} else {
		fmt.Printf("DEBUG IngestContent: FeaturedImageURL is empty, not adding to metadata\n")
	}
	
	// Add SEO fields to metadata if provided
	if request.MetaTitle != "" {
		content.Metadata["meta_title"] = request.MetaTitle
	}
	if request.MetaDescription != "" {
		content.Metadata["meta_description"] = request.MetaDescription
	}
	if request.CanonicalURL != "" {
		content.Metadata["canonical_url"] = request.CanonicalURL
	}
	if request.FocusKeyword != "" {
		content.Metadata["focus_keyword"] = request.FocusKeyword
	}
	
	// Add auto-linking flag to metadata
	content.Metadata["enable_auto_linking"] = request.EnableAutoLinking
	
	// Add language code to metadata if provided
	if request.LanguageCode != "" {
		content.Metadata["language_code"] = request.LanguageCode
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
	contents, err := s.ingestionRepo.GetPendingContent(ctx, 100, 0) // Get up to 100 items
	if err != nil {
		return fmt.Errorf("failed to get pending content: %w", err)
	}
	
	var content *models.IngestedContent
	for _, c := range contents {
		if c.ID == contentID {
			content = c // c is already a pointer from repository
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
	
	// Process tags if provided
	fmt.Printf("DEBUG: Processing article %d, tags count: %d, tags: %v\n", createdArticle.ID, len(content.Tags), content.Tags)
	if len(content.Tags) > 0 {
		fmt.Printf("DEBUG: Calling processArticleTags for article %d with tags: %v\n", createdArticle.ID, content.Tags)
		if err := s.processArticleTags(ctx, createdArticle.ID, content.Tags); err != nil {
			// Log error but don't fail the whole process
			fmt.Printf("ERROR: Failed to process tags for article %d: %v\n", createdArticle.ID, err)
		} else {
			fmt.Printf("SUCCESS: Tags processed for article %d\n", createdArticle.ID)
		}
	}
	
	// Process featured image if URL provided
	fmt.Printf("DEBUG: Checking featured image, metadata: %v\n", content.Metadata)
	if featuredImageURL, ok := content.Metadata["featured_image_url"].(string); ok && featuredImageURL != "" {
		fmt.Printf("DEBUG: Downloading featured image for article %d from URL: %s\n", createdArticle.ID, featuredImageURL)
		if err := s.downloadAndSetFeaturedImage(ctx, createdArticle.ID, featuredImageURL); err != nil {
			// Log error but don't fail the whole process
			fmt.Printf("ERROR: Failed to download featured image for article %d: %v\n", createdArticle.ID, err)
		} else {
			fmt.Printf("SUCCESS: Featured image downloaded for article %d\n", createdArticle.ID)
		}
	} else {
		fmt.Printf("DEBUG: No featured image URL found in metadata for article %d\n", createdArticle.ID)
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
	contents, err := s.ingestionRepo.GetPendingContent(ctx, limit, 0)
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
	// Check permissions - only users with manage_system permission can create content sources
	if currentUser == nil || !currentUser.HasPermission("manage_system") {
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
	// Check permissions - only users with manage_system permission can list content sources
	if currentUser == nil || !currentUser.HasPermission("manage_system") {
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
	
	// Set category - prioritize provided category over default
	if content.CategoryName != "" {
		// Try to find category by name
		categoryID, err := s.findCategoryByName(ctx, content.CategoryName)
		if err == nil && categoryID != 0 {
			article.CategoryID = categoryID
		} else if source.Config.DefaultCategoryID != 0 {
			// Fall back to default if category not found
			article.CategoryID = source.Config.DefaultCategoryID
		} else {
			// Use fallback category
			article.CategoryID = 1
		}
	} else if source.Config.DefaultCategoryID != 0 {
		// Use default category if no category provided
		article.CategoryID = source.Config.DefaultCategoryID
	} else {
		// Use fallback category
		article.CategoryID = 1
	}
	
	// Set published date - always ensure we have a date for partitioning
	if content.PublishedAt != nil {
		article.PublishedAt = content.PublishedAt
	} else {
		// Always set a published date for partitioning, even for drafts
		now := time.Now()
		article.PublishedAt = &now
	}
	
	// Auto-publish if configured
	if source.Config.AutoPublish {
		article.Status = "published"
	}
	
	// Set SEO fields directly on article (not in SEOData struct)
	// Use provided SEO fields from metadata if available, otherwise use defaults
	fmt.Printf("DEBUG convertToArticle: Processing metadata: %+v\n", content.Metadata)
	
	if mt, ok := content.Metadata["meta_title"].(string); ok && mt != "" {
		article.MetaTitle = mt
		fmt.Printf("DEBUG convertToArticle: Set meta_title to: %s\n", mt)
	} else {
		article.MetaTitle = content.Title
		fmt.Printf("DEBUG convertToArticle: Using title as meta_title: %s\n", content.Title)
	}
	
	if md, ok := content.Metadata["meta_description"].(string); ok && md != "" {
		article.MetaDescription = md
		fmt.Printf("DEBUG convertToArticle: Set meta_description to: %s\n", md)
	} else {
		article.MetaDescription = content.Excerpt
		fmt.Printf("DEBUG convertToArticle: Using excerpt as meta_description\n")
	}
	
	if cu, ok := content.Metadata["canonical_url"].(string); ok && cu != "" {
		article.CanonicalURL = cu
		fmt.Printf("DEBUG convertToArticle: Set canonical_url to: %s\n", cu)
	} else if content.SourceURL != "" {
		article.CanonicalURL = content.SourceURL
		fmt.Printf("DEBUG convertToArticle: Using source_url as canonical_url: %s\n", content.SourceURL)
	}
	
	if fk, ok := content.Metadata["focus_keyword"].(string); ok && fk != "" {
		article.FocusKeyword = fk
		fmt.Printf("DEBUG convertToArticle: Set focus_keyword to: %s\n", fk)
	} else {
		fmt.Printf("DEBUG convertToArticle: No focus_keyword in metadata\n")
	}
	
	// Set schema type
	article.SchemaType = "NewsArticle"
	
	// Set auto-linking flag from metadata
	if enableAutoLinking, ok := content.Metadata["enable_auto_linking"].(bool); ok {
		article.AutoLinking = enableAutoLinking
		fmt.Printf("DEBUG convertToArticle: Set auto_linking to: %v\n", enableAutoLinking)
	} else {
		fmt.Printf("DEBUG convertToArticle: No auto_linking in metadata, defaulting to false\n")
	}
	
	// Set language code from metadata or default to Persian
	if lc, ok := content.Metadata["language_code"].(string); ok && lc != "" {
		article.LanguageCode = lc
		fmt.Printf("DEBUG convertToArticle: Set language_code to: %s\n", lc)
	} else {
		article.LanguageCode = "fa"
		fmt.Printf("DEBUG convertToArticle: No language_code in metadata, defaulting to 'fa'\n")
	}
	
	// Set moderation status (default to approved for auto-published content)
	if source.Config.AutoPublish {
		article.ModerationStatus = "approved"
	} else {
		article.ModerationStatus = "pending"
	}
	
	// Tags will be processed after article creation in ProcessPendingContent
	
	return article, nil
}

func (s *ContentIngestionService) findCategoryByName(ctx context.Context, categoryName string) (uint64, error) {
	// Query to find category by name (case-insensitive)
	query := `SELECT id FROM categories WHERE LOWER(name) = LOWER($1) LIMIT 1`
	
	var categoryID uint64
	err := s.ingestionRepo.GetDB().QueryRowContext(ctx, query, categoryName).Scan(&categoryID)
	if err != nil {
		return 0, err
	}
	
	return categoryID, nil
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

// GetCategories returns all categories for content management
func (s *ContentIngestionService) GetCategories(ctx context.Context) ([]map[string]interface{}, error) {
	// Use the category repository to get real categories from database
	if s.categoryRepo == nil {
		// Fallback: return basic categories if repository not available
		return []map[string]interface{}{
			{"id": 1, "name": "Technology", "slug": "technology"},
			{"id": 2, "name": "Business", "slug": "business"},
			{"id": 3, "name": "Sports", "slug": "sports"},
			{"id": 4, "name": "Entertainment", "slug": "entertainment"},
			{"id": 5, "name": "Health", "slug": "health"},
			{"id": 6, "name": "Science", "slug": "science"},
		}, nil
	}
	
	// Get real categories from database
	categories, err := s.categoryRepo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}
	
	// Convert to map format for API response
	result := make([]map[string]interface{}, len(categories))
	for i, cat := range categories {
		result[i] = map[string]interface{}{
			"id":   cat.ID,
			"name": cat.Name,
			"slug": cat.Slug,
		}
	}
	
	return result, nil
}

// GetPendingContent returns pending content from ingested_content table
func (s *ContentIngestionService) GetPendingContent(ctx context.Context, limit, offset int) ([]map[string]interface{}, int, error) {
	if s.ingestionRepo == nil {
		return nil, 0, fmt.Errorf("ingestion repository not available")
	}
	
	// Get pending content from database
	pendingContent, err := s.ingestionRepo.GetPendingContent(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get pending content: %w", err)
	}
	
	// Convert to API response format
	result := make([]map[string]interface{}, len(pendingContent))
	for i, content := range pendingContent {
		result[i] = map[string]interface{}{
			"id":            content.ID,
			"title":         content.Title,
			"source_id":     content.SourceID,
			"author_name":   content.AuthorName,
			"category_name": content.CategoryName,
			"created_at":    content.CreatedAt,
			"excerpt":       content.Excerpt,
			"status":        content.Status,
		}
	}
	
	// Get total count for pagination
	total, err := s.ingestionRepo.GetPendingContentCount(ctx)
	if err != nil {
		total = len(result) // Fallback to current result count
	}
	
	return result, total, nil
}

// GetProcessedContent returns processed content history from database
func (s *ContentIngestionService) GetProcessedContent(ctx context.Context, limit, offset int, status string) ([]map[string]interface{}, int, error) {
	if s.ingestionRepo == nil {
		return nil, 0, fmt.Errorf("ingestion repository not available")
	}
	
	// Get processed content from database
	processedContent, err := s.ingestionRepo.GetProcessedContent(ctx, limit, offset, status)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get processed content: %w", err)
	}
	
	// Convert to API response format
	result := make([]map[string]interface{}, len(processedContent))
	for i, content := range processedContent {
		result[i] = map[string]interface{}{
			"id":           content.ID,
			"title":        content.Title,
			"source_id":    content.SourceID,
			"status":       content.Status,
			"processed_at": content.ProcessedAt,
			"article_id":   content.ArticleID,
		}
		
		// Add article slug if available
		if content.Metadata != nil {
			if slug, exists := content.Metadata["article_slug"]; exists {
				result[i]["article_slug"] = slug
			}
		}
	}
	
	// Get total count for pagination
	total, err := s.ingestionRepo.GetProcessedContentCount(ctx, status)
	if err != nil {
		total = len(result) // Fallback to current result count
	}
	
	return result, total, nil
}

// GetContentSources returns content sources from database
func (s *ContentIngestionService) GetContentSources(ctx context.Context, limit, offset int) ([]map[string]interface{}, int, error) {
	if s.ingestionRepo == nil {
		return nil, 0, fmt.Errorf("ingestion repository not available")
	}
	
	// Get content sources from database
	sources, total, err := s.ingestionRepo.GetContentSources(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get content sources: %w", err)
	}
	
	// Convert to API response format
	result := make([]map[string]interface{}, len(sources))
	for i, source := range sources {
		result[i] = map[string]interface{}{
			"id":         source.ID,
			"name":       source.Name,
			"type":       source.Type,
			"is_active":  source.IsActive,
			"api_key":    source.APIKey,
			"rate_limit": source.RateLimit,
			"priority":   source.Priority,
			"created_at": source.CreatedAt,
			"updated_at": source.UpdatedAt,
		}
	}
	
	return result, total, nil
}

// UpdateContentSource updates an existing content source
func (s *ContentIngestionService) UpdateContentSource(ctx context.Context, source *models.ContentSource, currentUser *models.User) (*models.ContentSource, error) {
	// Check permissions - only users with manage_system permission can update content sources
	if currentUser == nil || !currentUser.HasPermission("manage_system") {
		return nil, fmt.Errorf("insufficient permissions")
	}

	// If API key is empty, fetch the existing source and preserve its API key
	if source.APIKey == "" {
		existingSource, err := s.ingestionRepo.GetContentSourceByID(ctx, source.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch existing source: %w", err)
		}
		source.APIKey = existingSource.APIKey
	}

	// Update source through repository
	updatedSource, err := s.ingestionRepo.UpdateContentSource(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("failed to update content source: %w", err)
	}

	return updatedSource, nil
}

// GetIngestionStatsForAdmin returns content ingestion statistics for admin panel
func (s *ContentIngestionService) GetIngestionStatsForAdmin(ctx context.Context) (map[string]interface{}, error) {
	if s.ingestionRepo == nil {
		return nil, fmt.Errorf("ingestion repository not available")
	}

	// Get stats for today (last 24 hours) - for processed/rejected actions
	todayStats, err := s.ingestionRepo.GetIngestionStatsByAction(ctx, nil, 24)
	if err != nil {
		return nil, fmt.Errorf("failed to get today's stats: %w", err)
	}

	// Get total pending content (all time)
	allTimeStats, err := s.ingestionRepo.GetIngestionStats(ctx, nil, 24*365*10) // Last 10 years (effectively all time)
	if err != nil {
		return nil, fmt.Errorf("failed to get all-time stats: %w", err)
	}

	// Get total sources count
	_, totalSources, err := s.ingestionRepo.GetContentSources(ctx, 1000, 0) // Get up to 1000 sources
	if err != nil {
		return nil, fmt.Errorf("failed to get sources: %w", err)
	}

	// Format stats for frontend
	result := map[string]interface{}{
		"pending_content":  allTimeStats["pending"],     // Total pending content
		"processed_today":  todayStats["processed"],     // Processed in last 24h
		"rejected_today":   todayStats["rejected"],      // Rejected in last 24h
		"total_sources":    totalSources,                // Total active sources
	}

	return result, nil
}

// ProcessPendingContentByID processes a single content item by ID (can be pending or rejected)
func (s *ContentIngestionService) ProcessPendingContentByID(ctx context.Context, contentID uint64, currentUser *models.User) error {
	// Get the specific content item (any status)
	content, err := s.ingestionRepo.GetContentByID(ctx, contentID)
	if err != nil {
		return fmt.Errorf("failed to get content: %w", err)
	}

	// Reset status to pending before processing
	err = s.ingestionRepo.UpdateContentStatus(ctx, contentID, "pending")
	if err != nil {
		return fmt.Errorf("failed to reset content status: %w", err)
	}

	// Process the content item
	err = s.processIngestedContent(ctx, content)
	if err != nil {
		return fmt.Errorf("failed to process content: %w", err)
	}

	return nil
}

// RejectPendingContent rejects a pending content item
func (s *ContentIngestionService) RejectPendingContent(ctx context.Context, contentID uint64, reason string, currentUser *models.User) error {
	// Check permissions - allow admin, editor, and reporter roles
	if currentUser == nil || (currentUser.Role != "admin" && currentUser.Role != "editor" && currentUser.Role != "reporter") {
		return fmt.Errorf("insufficient permissions")
	}

	// Update the content status to rejected with reason using existing repository method
	err := s.ingestionRepo.UpdateIngestedContentStatus(ctx, contentID, "rejected", nil, reason)
	if err != nil {
		return fmt.Errorf("failed to reject content: %w", err)
	}

	return nil
}

// ProcessBatchContentByIDs processes multiple pending content items by their IDs
func (s *ContentIngestionService) ProcessBatchContentByIDs(ctx context.Context, contentIDs []uint64, currentUser *models.User) (int, error) {
	// Check permissions - allow admin, editor, and reporter roles
	if currentUser == nil || (currentUser.Role != "admin" && currentUser.Role != "editor" && currentUser.Role != "reporter") {
		return 0, fmt.Errorf("insufficient permissions")
	}

	processedCount := 0
	for _, contentID := range contentIDs {
		err := s.ProcessPendingContentByID(ctx, contentID, currentUser)
		if err != nil {
			// Log error but continue processing other items
			fmt.Printf("Failed to process content ID %d: %v\n", contentID, err)
			continue
		}
		processedCount++
	}

	return processedCount, nil
}

// processIngestedContent processes a single ingested content item into an article
func (s *ContentIngestionService) processIngestedContent(ctx context.Context, content *models.IngestedContent) error {
	// Get the content source to use for conversion
	source, err := s.ingestionRepo.GetContentSourceByID(ctx, content.SourceID)
	if err != nil {
		return fmt.Errorf("failed to get content source: %w", err)
	}

	// Convert to article using the proper conversion method
	article, err := s.convertToArticle(ctx, content, source)
	if err != nil {
		return fmt.Errorf("failed to convert to article: %w", err)
	}

	// Create the article
	createdArticle, err := s.articleRepo.Create(ctx, article)
	if err != nil {
		return fmt.Errorf("failed to create article: %w", err)
	}

	// Update ingested content status
	err = s.ingestionRepo.UpdateIngestedContentStatus(ctx, content.ID, "processed", &createdArticle.ID, "")
	if err != nil {
		return fmt.Errorf("failed to update ingested content status: %w", err)
	}

	return nil
}

// DeleteContentSource deletes a content source
func (s *ContentIngestionService) DeleteContentSource(ctx context.Context, sourceID uint64, currentUser *models.User) error {
	// Check permissions - only users with manage_system permission can delete content sources
	if currentUser == nil || !currentUser.HasPermission("manage_system") {
		return auth.ErrInsufficientPermissions
	}

	// Delete source
	err := s.ingestionRepo.DeleteContentSource(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("failed to delete content source: %w", err)
	}

	return nil
}

// GetContentByID retrieves a specific content item by ID
func (s *ContentIngestionService) GetContentByID(ctx context.Context, contentID uint64) (*models.IngestedContent, error) {
	// Get content from repository
	content, err := s.ingestionRepo.GetContentByID(ctx, contentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get content: %w", err)
	}

	return content, nil
}

// ReprocessRejectedContent moves rejected content back to pending status
func (s *ContentIngestionService) ReprocessRejectedContent(ctx context.Context, contentID uint64, currentUser *models.User) error {
	// Check permissions - allow admin, editor, and reporter roles
	if currentUser == nil || (currentUser.Role != "admin" && currentUser.Role != "editor" && currentUser.Role != "reporter") {
		return fmt.Errorf("insufficient permissions")
	}

	if s.ingestionRepo == nil {
		return fmt.Errorf("ingestion repository not available")
	}
	
	// Update the content status back to pending
	err := s.ingestionRepo.UpdateIngestedContentStatus(ctx, contentID, "pending", nil, "")
	if err != nil {
		return fmt.Errorf("failed to reprocess content: %w", err)
	}
	
	return nil
}

// processArticleTags finds or creates tags and associates them with the article
func (s *ContentIngestionService) processArticleTags(ctx context.Context, articleID uint64, tagNames []string) error {
	if len(tagNames) == 0 {
		return nil
	}
	
	db := s.ingestionRepo.GetDB()
	
	for _, tagName := range tagNames {
		tagName = strings.TrimSpace(tagName)
		if tagName == "" {
			continue
		}
		
		// Find or create tag (search in English language)
		var tagID uint64
		query := `SELECT id FROM tags WHERE LOWER(name) = LOWER($1) AND language_code = 'en' LIMIT 1`
		err := db.QueryRowContext(ctx, query, tagName).Scan(&tagID)
		
		if err != nil {
			// Tag doesn't exist, create it
			slug := strings.ToLower(strings.ReplaceAll(tagName, " ", "-"))
			fmt.Printf("DEBUG: Creating new tag '%s' with slug '%s'\n", tagName, slug)
			insertQuery := `INSERT INTO tags (name, slug, language_code, created_at, updated_at) VALUES ($1, $2, 'en', NOW(), NOW()) RETURNING id`
			err = db.QueryRowContext(ctx, insertQuery, tagName, slug).Scan(&tagID)
			if err != nil {
				fmt.Printf("ERROR: Failed to create tag %s: %v\n", tagName, err)
				continue
			}
			fmt.Printf("DEBUG: Created tag '%s' with ID %d\n", tagName, tagID)
		} else {
			fmt.Printf("DEBUG: Found existing tag '%s' with ID %d\n", tagName, tagID)
		}
		
		// Associate tag with article (table is partitioned by created_at)
		fmt.Printf("DEBUG: Associating tag %d (%s) with article %d\n", tagID, tagName, articleID)
		result, err := db.ExecContext(ctx, `INSERT INTO article_tags (article_id, tag_id, created_at) VALUES ($1, $2, NOW()) ON CONFLICT DO NOTHING`, articleID, tagID)
		if err != nil {
			fmt.Printf("ERROR: Failed to associate tag %d with article %d: %v\n", tagID, articleID, err)
		} else {
			rowsAffected, _ := result.RowsAffected()
			fmt.Printf("DEBUG: Tag association result - rows affected: %d\n", rowsAffected)
		}
	}
	
	return nil
}

// downloadAndSetFeaturedImage downloads an image from URL and sets it as the article's featured image
func (s *ContentIngestionService) downloadAndSetFeaturedImage(ctx context.Context, articleID uint64, imageURL string) error {
	// Download the image
	resp, err := http.Get(imageURL)
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download image: HTTP %d", resp.StatusCode)
	}
	
	// Read image data
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read image data: %w", err)
	}
	
	// Detect content type
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(imageData)
	}
	
	// Generate filename
	ext := ".jpg"
	switch contentType {
	case "image/png":
		ext = ".png"
	case "image/jpeg", "image/jpg":
		ext = ".jpg"
	case "image/gif":
		ext = ".gif"
	case "image/webp":
		ext = ".webp"
	}
	
	filename := fmt.Sprintf("article_%d_featured%s", articleID, ext)
	
	// Create media directory if it doesn't exist
	mediaDir := "./web/static/media/articles"
	if err := os.MkdirAll(mediaDir, 0755); err != nil {
		return fmt.Errorf("failed to create media directory: %w", err)
	}
	
	// Save image to disk
	filepath := fmt.Sprintf("%s/%s", mediaDir, filename)
	if err := os.WriteFile(filepath, imageData, 0644); err != nil {
		return fmt.Errorf("failed to save image: %w", err)
	}
	
	// Detect image dimensions
	img, _, err := image.DecodeConfig(bytes.NewReader(imageData))
	if err != nil {
		// If we can't decode, use default dimensions
		img.Width = 1200
		img.Height = 630
		fmt.Printf("Warning: Could not decode image dimensions, using defaults: %v\n", err)
	}
	
	// Create image record in database
	db := s.ingestionRepo.GetDB()
	var imageID uint64
	imageQuery := `
		INSERT INTO images (original_url, filename, width, height, file_size, mime_type, article_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id`
	
	err = db.QueryRowContext(ctx, imageQuery,
		imageURL,
		filename,
		img.Width,
		img.Height,
		len(imageData),
		contentType,
		articleID,
	).Scan(&imageID)
	
	if err != nil {
		return fmt.Errorf("failed to create image record: %w", err)
	}
	
	// Update article's featured_image_id
	updateQuery := `UPDATE articles SET featured_image_id = $1, updated_at = NOW() WHERE id = $2`
	_, err = db.ExecContext(ctx, updateQuery, imageID, articleID)
	if err != nil {
		return fmt.Errorf("failed to update article featured image: %w", err)
	}
	
	fmt.Printf("Successfully downloaded and set featured image for article %d: %s\n", articleID, filename)
	return nil
}
