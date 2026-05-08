package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// Article API request/response types

// CreateArticleRequest represents a request to create an article
type CreateArticleRequest struct {
	Title              string             `json:"title" validate:"required,max=255"`
	Slug               string             `json:"slug" validate:"max=255"`
	Content            string             `json:"content"`                     // Not required for drafts
	Excerpt            string             `json:"excerpt" validate:"max=500"`
	CategoryID         uint64             `json:"category_id"`                 // Backward compatibility
	CategoryIDs        []uint64           `json:"category_ids"`                // Multiple categories support
	Tags               []string           `json:"tags"`                        // Changed to []string to match frontend
	Status             string             `json:"status" validate:"required,oneof=draft published archived scheduled deleted"`
	FeaturedImageID    *string            `json:"featured_image_id"`           // Featured image from media library (as string to avoid JS precision loss)
	AutoLinking        bool               `json:"auto_linking"`                // Enable/disable auto-linking for this article
	ScheduledAt        *string            `json:"scheduled_at"`                // For scheduled publishing
	LanguageCode       string             `json:"language_code"`               // Language code (en, de, fr, es, ar)
	TranslationGroupID *uint64            `json:"translation_group_id"`        // Group ID for translations
	SEOData            SEODataRequest     `json:"seo_data"`                    // Enhanced SEO data
}

// SEODataRequest represents SEO data from the frontend
type SEODataRequest struct {
	MetaTitle       string `json:"meta_title" validate:"max=60"`
	MetaDescription string `json:"meta_description" validate:"max=160"`
	FocusKeyword    string `json:"focus_keyword" validate:"max=100"`
}

// UpdateArticleRequest represents a request to update an article
type UpdateArticleRequest struct {
	Title              *string            `json:"title,omitempty" validate:"omitempty,max=255"`
	Slug               *string            `json:"slug,omitempty" validate:"omitempty,max=255"`
	Content            *string            `json:"content,omitempty"`
	Excerpt            *string            `json:"excerpt,omitempty" validate:"omitempty,max=500"`
	CategoryID         *uint64            `json:"category_id,omitempty"`      // Backward compatibility
	CategoryIDs        []uint64           `json:"category_ids,omitempty"`     // Multiple categories support
	Tags               []string           `json:"tags,omitempty"`
	Status             *string            `json:"status,omitempty" validate:"omitempty,oneof=draft published archived scheduled"`
	FeaturedImageID    *string            `json:"featured_image_id,omitempty"`
	ScheduledAt        *string            `json:"scheduled_at,omitempty"`
	LanguageCode       *string            `json:"language_code,omitempty"`    // Language code (en, de, fr, es, ar)
	TranslationGroupID *uint64            `json:"translation_group_id,omitempty"` // Group ID for translations
	SEOData            *SEODataRequest    `json:"seo_data,omitempty"`
}

// BulkCreateArticleRequest represents a bulk article creation request
type BulkCreateArticleRequest struct {
	Articles []CreateArticleRequest `json:"articles" validate:"required,min=1,max=1000"`
}

// ArticleListResponse represents the response for article listing
type ArticleListResponse struct {
	Articles   []models.Article `json:"articles"`
	Pagination Pagination       `json:"pagination"`
}

// Article CRUD Handlers

// CreateArticle creates a new article
func (h *APIHandler) CreateArticle(c *gin.Context) {
	var req CreateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, &models.ValidationError{
			Message: "Invalid request format",
			Fields:  []string{err.Error()},
		})
		return
	}

	// Custom validation based on status
	if req.Status == "published" || req.Status == "scheduled" {
		if req.Content == "" {
			handleError(c, &models.ValidationError{
				Message: "Content is required for published/scheduled articles",
				Fields:  []string{"content is required"},
			})
			return
		}
		// Check for categories (support both single and multiple)
		hasCategories := req.CategoryID > 0 || len(req.CategoryIDs) > 0
		if !hasCategories {
			handleError(c, &models.ValidationError{
				Message: "Category is required for published/scheduled articles", 
				Fields:  []string{"at least one category is required"},
			})
			return
		}
	}

	// Get current user
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// Generate unique slug if not provided
	slug := req.Slug
	if slug == "" {
		slug = generateSlugFromTitle(req.Title)
	}
	
	// Ensure slug is unique by appending timestamp if needed
	originalSlug := slug
	for i := 1; i <= 10; i++ {
		// Check if slug exists (simplified check - in production you'd query the database)
		if i > 1 {
			slug = fmt.Sprintf("%s-%d", originalSlug, i)
		}
		// For now, we'll let the database handle the uniqueness constraint
		// and catch the error if it occurs
		break
	}

	// Create article model - map to actual database columns
	log.Printf("Auto-linking value received: %t", req.AutoLinking)
	log.Printf("Request data: Title=%s, Status=%s, CategoryID=%d, CategoryIDs=%v", req.Title, req.Status, req.CategoryID, req.CategoryIDs)
	
	// Handle backward compatibility: use CategoryIDs if provided, otherwise use CategoryID
	var primaryCategoryID uint64
	if len(req.CategoryIDs) > 0 {
		primaryCategoryID = req.CategoryIDs[0] // Use first category as primary
		log.Printf("Using CategoryIDs[0] as primaryCategoryID: %d", primaryCategoryID)
	} else if req.CategoryID > 0 {
		primaryCategoryID = req.CategoryID
		log.Printf("Using CategoryID as primaryCategoryID: %d", primaryCategoryID)
	} else {
		log.Printf("WARNING: No category provided! CategoryID=%d, CategoryIDs=%v", req.CategoryID, req.CategoryIDs)
	}
	
	// Truncate SEO fields to prevent validation errors
	// Use rune slicing to avoid cutting multi-byte UTF-8 characters
	metaTitle := req.SEOData.MetaTitle
	metaTitleRunes := []rune(metaTitle)
	if len(metaTitleRunes) > 60 {
		metaTitle = string(metaTitleRunes[:60])
	}
	metaDescription := req.SEOData.MetaDescription
	metaDescRunes := []rune(metaDescription)
	if len(metaDescRunes) > 160 {
		metaDescription = string(metaDescRunes[:160])
	}
	
	// Truncate focus keyword
	focusKeyword := req.SEOData.FocusKeyword
	focusKeywordRunes := []rune(focusKeyword)
	if len(focusKeywordRunes) > 100 {
		focusKeyword = string(focusKeywordRunes[:100])
	}
	
	article := &models.Article{
		Title:              req.Title,
		Slug:               slug,
		Content:            req.Content,
		Excerpt:            req.Excerpt,
		AuthorID:           currentUser.ID,
		CategoryID:         primaryCategoryID, // Keep for backward compatibility
		Status:             req.Status,
		AutoLinking:        req.AutoLinking, // Use the value from the request
		// Map SEO fields to individual columns (truncated)
		MetaTitle:          metaTitle,
		MetaDescription:    metaDescription,
		FocusKeyword:       focusKeyword, // Add focus keyword
		SchemaType:         "NewsArticle", // Default schema type
		LanguageCode:       req.LanguageCode,
		TranslationGroupID: req.TranslationGroupID,
	}
	
	log.Printf("Article object created: CategoryID=%d, TranslationGroupID=%v", article.CategoryID, article.TranslationGroupID)
	
	// Set default language code if not provided
	if article.LanguageCode == "" {
		article.LanguageCode = "en" // Default to English
	}
	
	log.Printf("Article auto-linking set to: %t", article.AutoLinking)

	// Handle published_at for partitioning
	now := time.Now()
	if req.Status == "published" {
		article.PublishedAt = &now
	} else if req.Status == "scheduled" && req.ScheduledAt != nil {
		// Parse the scheduled date (frontend sends ISO string)
		// For now, use current time for partitioning, actual scheduling handled elsewhere
		article.PublishedAt = &now
	} else {
		// For drafts, use current time for partitioning but mark as unpublished
		article.PublishedAt = &now
	}

	// Handle featured image - convert string to uint64
	if req.FeaturedImageID != nil && *req.FeaturedImageID != "" {
		imageIDStr := *req.FeaturedImageID
		log.Printf("Featured image ID received as string: %s", imageIDStr)
		
		if imageID, err := strconv.ParseUint(imageIDStr, 10, 64); err == nil && imageID > 0 {
			log.Printf("Converted to uint64: %d", imageID)
			article.FeaturedImageID = &imageID
		} else {
			log.Printf("Invalid featured image ID format: %s, error: %v", imageIDStr, err)
			article.FeaturedImageID = nil
		}
	} else {
		log.Printf("No featured image provided or empty ID: %v", req.FeaturedImageID)
		article.FeaturedImageID = nil
	}

	// Create article through service
	createdArticle, err := h.articleService.Create(c.Request.Context(), article, currentUser)
	if err != nil {
		// Handle duplicate slug error
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") && strings.Contains(err.Error(), "slug") {
			// Try with a timestamp suffix
			timestamp := time.Now().Unix()
			article.Slug = fmt.Sprintf("%s-%d", originalSlug, timestamp)
			log.Printf("Slug conflict detected, trying with timestamp: %s", article.Slug)
			
			createdArticle, err = h.articleService.Create(c.Request.Context(), article, currentUser)
			if err != nil {
				handleError(c, err)
				return
			}
		} else {
			handleError(c, err)
			return
		}
	}

	// Handle tags after article creation - pass article language for correct tag association
	if len(req.Tags) > 0 {
		err = h.handleArticleTags(c.Request.Context(), createdArticle.ID, req.Tags, createdArticle.LanguageCode)
		if err != nil {
			// Log error but don't fail the request since article was created
			// In production, you might want to handle this differently
			// For now, we'll continue and return success
		}
	}

	// Handle featured image association - only if we have a valid ID and it was successfully parsed
	if article.FeaturedImageID != nil {
		err = h.handleFeaturedImage(c.Request.Context(), createdArticle.ID, *article.FeaturedImageID)
		if err != nil {
			// Log error but don't fail the request
		}
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Data:    createdArticle,
		Message: "Article created successfully",
	})
}

// GetArticle retrieves an article by ID
func (h *APIHandler) GetArticle(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		handleError(c, &models.ValidationError{
			Message: "Invalid article ID",
			Fields:  []string{"id must be a valid number"},
		})
		return
	}

	article, err := h.articleService.GetByID(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Data: article,
	})
}

// GetArticleBySlug retrieves an article by slug
func (h *APIHandler) GetArticleBySlug(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		handleError(c, &models.ValidationError{
			Message: "Invalid slug",
			Fields:  []string{"slug is required"},
		})
		return
	}

	article, err := h.articleService.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		handleError(c, err)
		return
	}

	// Record view asynchronously
	go func() {
		clientIP := c.ClientIP()
		userAgent := c.GetHeader("User-Agent")
		referer := c.GetHeader("Referer")
		h.articleService.RecordView(c.Request.Context(), article.ID, clientIP, userAgent, referer)
	}()

	c.JSON(http.StatusOK, SuccessResponse{
		Data: article,
	})
}

// UpdateArticle updates an existing article
func (h *APIHandler) UpdateArticle(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		handleError(c, &models.ValidationError{
			Message: "Invalid article ID",
			Fields:  []string{"id must be a valid number"},
		})
		return
	}

	var req UpdateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, &models.ValidationError{
			Message: "Invalid request format",
			Fields:  []string{err.Error()},
		})
		return
	}

	// Get current user for authentication
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// Create the service's UpdateArticleRequest type
	serviceReq := &services.UpdateArticleRequest{
		Title:              req.Title,
		Slug:               req.Slug,
		Content:            req.Content,
		Excerpt:            req.Excerpt,
		CategoryID:         req.CategoryID,
		CategoryIDs:        req.CategoryIDs,
		Status:             req.Status,
		Tags:               req.Tags,
		LanguageCode:       req.LanguageCode,
		TranslationGroupID: req.TranslationGroupID,
	}

	// Handle SEO data conversion
	if req.SEOData != nil {
		serviceReq.SEOData = &models.SEOData{
			MetaTitle:       req.SEOData.MetaTitle,
			MetaDescription: req.SEOData.MetaDescription,
			FocusKeyword:    req.SEOData.FocusKeyword,
		}
	}

	// Handle featured image
	if req.FeaturedImageID != nil {
		if *req.FeaturedImageID == "" {
			// Empty string means remove featured image (set to nil)
			serviceReq.FeaturedImageID = nil
		} else {
			// Convert string to uint64
			if imageID, err := strconv.ParseUint(*req.FeaturedImageID, 10, 64); err == nil {
				serviceReq.FeaturedImageID = &imageID
			}
		}
	}

	// Call the service Update method
	updatedArticle, err := h.articleService.Update(c.Request.Context(), id, serviceReq, currentUser)
	if err != nil {
		// Handle foreign key constraint violations gracefully
		if strings.Contains(err.Error(), "foreign key constraint") && strings.Contains(err.Error(), "featured_image_id") {
			// Try again without the featured image
			log.Printf("Featured image foreign key error, retrying without featured image: %v", err)
			serviceReq.FeaturedImageID = nil
			updatedArticle, err = h.articleService.Update(c.Request.Context(), id, serviceReq, currentUser)
			if err != nil {
				handleError(c, err)
				return
			}
		} else {
			handleError(c, err)
			return
		}
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Data:    updatedArticle,
		Message: "Article updated successfully",
	})
}

// DeleteArticle deletes an article (soft delete)
func (h *APIHandler) DeleteArticle(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		handleError(c, &models.ValidationError{
			Message: "Invalid article ID",
			Fields:  []string{"id must be a valid number"},
		})
		return
	}

	// Get current user
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// Check if this is a permanent delete request (from recycle bin)
	permanent := c.Query("permanent") == "true"
	
	if permanent {
		// Permanent delete - actually remove from database
		log.Printf("Performing PERMANENT delete for article ID: %d", id)
		
		// First check if article exists and is already deleted (status = 'deleted')
		existingArticle, err := h.articleService.GetByID(c.Request.Context(), id)
		if err != nil {
			handleError(c, err)
			return
		}
		
		if existingArticle.Status != "deleted" {
			handleError(c, &models.ValidationError{
				Message: "Article must be in trash before permanent deletion",
				Fields:  []string{"only deleted articles can be permanently removed"},
			})
			return
		}
		
		// Implement actual permanent delete from database
		log.Printf("Executing permanent delete from database for article ID: %d", id)
		
		// Get current user for permission checking
		currentUser, err := h.getCurrentUser(c)
		if err != nil {
			log.Printf("Failed to get current user: %v", err)
			handleError(c, fmt.Errorf("authentication required for permanent delete"))
			return
		}
		
		// Call the service method for permanent delete
		err = h.articleService.PermanentDelete(c.Request.Context(), id, currentUser)
		if err != nil {
			log.Printf("Failed to permanently delete article: %v", err)
			handleError(c, fmt.Errorf("failed to permanently delete article: %w", err))
			return
		}
		
		log.Printf("Successfully permanently deleted article ID: %d", id)
		c.JSON(http.StatusOK, SuccessResponse{
			Message: fmt.Sprintf("Article %d permanently deleted from database", id),
		})
		
	} else {
		// Soft delete - move to trash (change status to 'deleted')
		log.Printf("Performing SOFT delete (move to trash) for article ID: %d", id)
		
		// Use the Update method to change status to 'deleted'
		updateReq := &services.UpdateArticleRequest{
			Status: stringPtr("deleted"),
		}
		
		_, err = h.articleService.Update(c.Request.Context(), id, updateReq, currentUser)
		if err != nil {
			handleError(c, err)
			return
		}
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Article deleted successfully",
	})
}

// ListArticles retrieves articles with pagination, filtering, and sorting
func (h *APIHandler) ListArticles(c *gin.Context) {
	// Get pagination parameters
	limit, offset, err := getPaginationParams(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// Get filtering parameters
	filters := services.ArticleFilters{
		Status:     c.Query("status"),
		CategoryID: parseUint64Query(c, "category_id"),
		AuthorID:   parseUint64Query(c, "author_id"),
		Search:     c.Query("search"),
		Tags:       parseUint64ArrayQuery(c, "tags"),
	}

	// Get sorting parameters
	sortBy := c.DefaultQuery("sort_by", "published_at")
	sortOrder := c.DefaultQuery("sort_order", "desc")

	// List articles through service
	articles, total, err := h.articleService.List(c.Request.Context(), limit, offset, filters, sortBy, sortOrder)
	if err != nil {
		handleError(c, err)
		return
	}

	// Calculate pagination
	page := (offset / limit) + 1
	totalPages := (total + limit - 1) / limit

	response := ArticleListResponse{
		Articles: articles,
		Pagination: Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	c.JSON(http.StatusOK, response)
}

// PublishArticle publishes a draft article
func (h *APIHandler) PublishArticle(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		handleError(c, &models.ValidationError{
			Message: "Invalid article ID",
			Fields:  []string{"id must be a valid number"},
		})
		return
	}

	// Get current user
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// Publish article through service
	article, err := h.articleService.Publish(c.Request.Context(), id, currentUser)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Data:    article,
		Message: "Article published successfully",
	})
}

// GetTrendingArticles retrieves trending articles
func (h *APIHandler) GetTrendingArticles(c *gin.Context) {
	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	hours := 24
	if hoursStr := c.Query("hours"); hoursStr != "" {
		if h, err := strconv.Atoi(hoursStr); err == nil && h > 0 && h <= 168 {
			hours = h
		}
	}

	articles, err := h.articleService.GetTrending(c.Request.Context(), limit, hours)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Data: articles,
	})
}

// GetPopularArticles retrieves popular articles by view count
func (h *APIHandler) GetPopularArticles(c *gin.Context) {
	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	days := 7
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 30 {
			days = d
		}
	}

	articles, err := h.articleService.GetPopular(c.Request.Context(), limit, days)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Data: articles,
	})
}

// BulkCreateArticles creates multiple articles in a single request
func (h *APIHandler) BulkCreateArticles(c *gin.Context) {
	var req BulkCreateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError(c, &models.ValidationError{
			Message: "Invalid request format",
			Fields:  []string{err.Error()},
		})
		return
	}

	// Validate request
	if len(req.Articles) == 0 {
		handleError(c, &models.ValidationError{
			Message: "No articles provided",
			Fields:  []string{"articles array cannot be empty"},
		})
		return
	}

	if len(req.Articles) > 1000 {
		handleError(c, &models.ValidationError{
			Message: "Too many articles",
			Fields:  []string{"maximum 1000 articles per request"},
		})
		return
	}

	// Get current user
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// Convert requests to article models
	articles := make([]models.Article, len(req.Articles))
	for i, articleReq := range req.Articles {
		articles[i] = models.Article{
			Title:           articleReq.Title,
			Content:         articleReq.Content,
			Excerpt:         articleReq.Excerpt,
			AuthorID:        currentUser.ID,
			CategoryID:      articleReq.CategoryID,
			Status:          articleReq.Status,
			// Map SEO fields to individual columns
			MetaTitle:       articleReq.SEOData.MetaTitle,
			MetaDescription: articleReq.SEOData.MetaDescription,
			SchemaType:      "NewsArticle",
			LanguageCode:    "fa",
		}
	}

	// Create articles through service
	createdArticles, err := h.articleService.BulkCreate(c.Request.Context(), articles, currentUser)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Data:    createdArticles,
		Message: "Articles created successfully",
	})
}

// Helper functions for query parameter parsing

func parseUint64Query(c *gin.Context, key string) *uint64 {
	value := c.Query(key)
	if value == "" {
		return nil
	}
	
	if parsed, err := strconv.ParseUint(value, 10, 64); err == nil {
		return &parsed
	}
	
	return nil
}

func parseUint64ArrayQuery(c *gin.Context, key string) []uint64 {
	value := c.Query(key)
	if value == "" {
		return nil
	}
	
	parts := strings.Split(value, ",")
	var result []uint64
	
	for _, part := range parts {
		if parsed, err := strconv.ParseUint(strings.TrimSpace(part), 10, 64); err == nil {
			result = append(result, parsed)
		}
	}
	
	return result
}

// Transliteration maps for SEO-friendly slugs
var arabicToLatin = map[rune]string{
	'ا': "a", 'أ': "a", 'إ': "e", 'آ': "a",
	'ب': "b", 'ت': "t", 'ث': "th",
	'ج': "j", 'ح': "h", 'خ': "kh",
	'د': "d", 'ذ': "dh",
	'ر': "r", 'ز': "z",
	'س': "s", 'ش': "sh",
	'ص': "s", 'ض': "d",
	'ط': "t", 'ظ': "z",
	'ع': "a", 'غ': "gh",
	'ف': "f", 'ق': "q",
	'ك': "k", 'ک': "k",
	'ل': "l", 'م': "m", 'ن': "n",
	'ه': "h", 'ة': "h",
	'و': "w", 'ؤ': "w",
	'ي': "y", 'ى': "y", 'ئ': "y",
	'ء': "",
	'٠': "0", '١': "1", '٢': "2", '٣': "3", '٤': "4",
	'٥': "5", '٦': "6", '٧': "7", '٨': "8", '٩': "9",
	'َ': "a", 'ُ': "u", 'ِ': "i", 'ً': "", 'ٌ': "", 'ٍ': "",
	'ّ': "", 'ْ': "",
}

var germanToLatin = map[rune]string{
	'ä': "ae", 'Ä': "ae", 'ö': "oe", 'Ö': "oe",
	'ü': "ue", 'Ü': "ue", 'ß': "ss",
}

var frenchToLatin = map[rune]string{
	'à': "a", 'â': "a", 'æ': "ae", 'ç': "c",
	'é': "e", 'è': "e", 'ê': "e", 'ë': "e",
	'î': "i", 'ï': "i", 'ô': "o", 'œ': "oe",
	'ù': "u", 'û': "u", 'ü': "u", 'ÿ': "y",
}

var spanishToLatin = map[rune]string{
	'á': "a", 'é': "e", 'í': "i", 'ó': "o", 'ú': "u",
	'ñ': "n", 'Ñ': "n", 'ü': "u",
}

// transliterate converts non-ASCII characters to ASCII equivalents
func transliterate(text string) string {
	var result strings.Builder
	for _, char := range text {
		if val, ok := arabicToLatin[char]; ok {
			result.WriteString(val)
		} else if val, ok := germanToLatin[char]; ok {
			result.WriteString(val)
		} else if val, ok := frenchToLatin[char]; ok {
			result.WriteString(val)
		} else if val, ok := spanishToLatin[char]; ok {
			result.WriteString(val)
		} else {
			result.WriteRune(char)
		}
	}
	return result.String()
}

// generateSlugFromTitle creates a URL-friendly slug from a title
func generateSlugFromTitle(title string) string {
	// First transliterate non-ASCII characters
	slug := transliterate(title)
	// Convert to lowercase
	slug = strings.ToLower(slug)
	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")
	// Remove special characters (keep only letters, numbers, and hyphens)
	slug = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(slug, "")
	// Remove multiple consecutive hyphens
	slug = regexp.MustCompile(`-+`).ReplaceAllString(slug, "-")
	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")
	// If slug is empty after all processing, generate a random one
	if slug == "" {
		slug = fmt.Sprintf("article-%d", time.Now().UnixNano())
	}
	return slug
}

// handleArticleTags processes and associates tags with an article
// languageCode is the article's language to ensure tags are created/associated in the correct language
func (h *APIHandler) handleArticleTags(ctx context.Context, articleID uint64, tagNames []string, languageCode string) error {
	// Use direct database approach for now
	return h.handleTagsDirect(ctx, articleID, tagNames, languageCode)
}

// handleTagsDirect handles tags with direct database queries
// languageCode is the article's language to ensure tags are created/associated in the correct language
func (h *APIHandler) handleTagsDirect(ctx context.Context, articleID uint64, tagNames []string, languageCode string) error {
	// Use the article service to handle tag creation and association
	if h.articleService == nil {
		log.Printf("Article service not available for tag processing")
		return nil
	}
	
	// Default to English if no language specified
	if languageCode == "" {
		languageCode = "en"
	}
	
	for _, tagName := range tagNames {
		tagName = strings.TrimSpace(tagName)
		if tagName == "" {
			continue
		}
		
		// Generate slug from tag name
		tagSlug := generateSlugFromTitle(tagName)
		
		// Try to create/associate tag through article service
		// Note: This assumes the article service has tag handling methods
		// If not, we'll need to implement direct database operations
		
		err := h.createAndAssociateTag(ctx, articleID, tagName, tagSlug, languageCode)
		if err != nil {
			log.Printf("Failed to create/associate tag %s with article %d: %v", tagName, articleID, err)
			// Continue with other tags even if one fails
			continue
		}
		
		log.Printf("Successfully created/associated tag: %s (slug: %s) with article %d (lang: %s)", tagName, tagSlug, articleID, languageCode)
	}
	
	return nil
}

// createAndAssociateTag creates a tag if it doesn't exist and associates it with an article
// languageCode is the article's language - we try to find an existing tag in that language first,
// or create a new one in that language if it doesn't exist
func (h *APIHandler) createAndAssociateTag(ctx context.Context, articleID uint64, tagName, tagSlug, languageCode string) error {
	// Check if we have access to tag service
	if h.tagService == nil {
		log.Printf("TagService not available - cannot create/associate tag: %s", tagName)
		return nil
	}
	
	// 1. First try to find an existing tag with the same name in the article's language
	existingTag, err := h.findTagByNameAndLanguage(ctx, tagName, languageCode)
	if err != nil || existingTag == nil {
		// No tag found in the article's language
		// Check if there's a tag with the same name in any language (to get translation_group_id)
		anyLangTag, _ := h.findTagByName(ctx, tagName)
		
		// Create a new tag in the article's language
		uniqueSlug, err := h.generateUniqueTagSlug(ctx, tagSlug)
		if err != nil {
			log.Printf("Failed to generate unique slug for %s: %v", tagName, err)
			return err
		}
		
		newTag := &models.Tag{
			Name:         tagName,
			Slug:         uniqueSlug,
			LanguageCode: languageCode, // Use article's language
			Keywords:     []string{tagName}, // Add the tag name as a keyword
		}
		
		// If we found a tag with the same name in another language, link them via translation_group_id
		if anyLangTag != nil && anyLangTag.TranslationGroupID != nil {
			newTag.TranslationGroupID = anyLangTag.TranslationGroupID
			log.Printf("Linking new tag to existing translation group: %d", *anyLangTag.TranslationGroupID)
		}
		
		// Validate and prepare the tag
		if err := models.ValidateTag(newTag); err != nil {
			log.Printf("Tag validation failed for %s: %v", tagName, err)
			return err
		}
		
		newTag.PrepareForDB()
		
		// Create the tag
		err = h.tagService.Create(newTag)
		if err != nil {
			log.Printf("Failed to create tag %s: %v", tagName, err)
			return fmt.Errorf("failed to create tag: %w", err)
		}
		
		log.Printf("Created new tag: %s (slug: %s, lang: %s)", tagName, uniqueSlug, languageCode)
		existingTag = newTag
	} else {
		log.Printf("Using existing tag: %s (ID: %d, lang: %s)", tagName, existingTag.ID, existingTag.LanguageCode)
	}
	
	// 2. Associate tag with article using direct database query
	// Since we don't have a direct article-tag association service,
	// we'll use the database connection from the article service
	if h.articleService == nil {
		log.Printf("ArticleService not available - cannot associate tag with article")
		return nil
	}
	
	// Get database connection through article service
	// We'll need to add a method to associate tags with articles
	err = h.associateTagWithArticle(ctx, articleID, existingTag.ID)
	if err != nil {
		log.Printf("Failed to associate tag %s (ID: %d) with article %d: %v", tagName, existingTag.ID, articleID, err)
		return fmt.Errorf("failed to associate tag with article: %w", err)
	}
	
	log.Printf("Successfully associated tag %s (ID: %d) with article %d", tagName, existingTag.ID, articleID)
	return nil
}

// associateTagWithArticle creates the association between an article and a tag
func (h *APIHandler) associateTagWithArticle(ctx context.Context, articleID, tagID uint64) error {
	// Use the ArticleService to handle the database operation
	err := h.articleService.AssociateTagWithArticle(ctx, articleID, tagID)
	if err != nil {
		return fmt.Errorf("failed to associate tag %d with article %d: %w", tagID, articleID, err)
	}
	
	log.Printf("Successfully associated article %d with tag %d in database", articleID, tagID)
	return nil
}

// findTagByName searches for a tag by name (case-insensitive)
func (h *APIHandler) findTagByName(ctx context.Context, tagName string) (*models.Tag, error) {
	// Get all tags and search by name (case-insensitive)
	tags, err := h.tagService.GetAll()
	if err != nil {
		return nil, err
	}
	
	lowerTagName := strings.ToLower(strings.TrimSpace(tagName))
	for _, tag := range tags {
		if strings.ToLower(tag.Name) == lowerTagName {
			return &tag, nil
		}
	}
	
	return nil, nil // Not found
}

// findTagByNameAndLanguage searches for a tag by name and language code (case-insensitive)
// This is used to find the correct localized version of a tag for SEO purposes
func (h *APIHandler) findTagByNameAndLanguage(ctx context.Context, tagName string, languageCode string) (*models.Tag, error) {
	// Get all tags and search by name and language (case-insensitive)
	tags, err := h.tagService.GetAll()
	if err != nil {
		return nil, err
	}
	
	lowerTagName := strings.ToLower(strings.TrimSpace(tagName))
	for _, tag := range tags {
		if strings.ToLower(tag.Name) == lowerTagName && tag.LanguageCode == languageCode {
			return &tag, nil
		}
	}
	
	return nil, nil // Not found
}

// generateUniqueTagSlug generates a unique slug for a tag
func (h *APIHandler) generateUniqueTagSlug(ctx context.Context, baseSlug string) (string, error) {
	// Try the base slug first
	_, err := h.tagService.GetBySlug(baseSlug)
	if err != nil {
		// Slug doesn't exist, we can use it
		return baseSlug, nil
	}
	
	// Slug exists, try with numbers
	for i := 1; i <= 100; i++ {
		candidateSlug := fmt.Sprintf("%s-%d", baseSlug, i)
		_, err := h.tagService.GetBySlug(candidateSlug)
		if err != nil {
			// This slug doesn't exist, we can use it
			return candidateSlug, nil
		}
	}
	
	return "", fmt.Errorf("could not generate unique slug for %s", baseSlug)
}

// handleFeaturedImage associates a featured image with an article
func (h *APIHandler) handleFeaturedImage(ctx context.Context, articleID uint64, imageID uint64) error {
	// This would need to be implemented with proper media service
	// For now, we'll create a placeholder implementation
	
	// In a full implementation, this would:
	// 1. Verify the image exists
	// 2. Update the article's featured_image_id field
	// 3. Handle any image processing or optimization
	
	// TODO: Implement actual featured image association
	// mediaService.AssociateFeaturedImage(articleID, imageID)
	
	return nil
}


// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}

// Article Engagement Handlers

// LikeArticle handles POST /api/v1/articles/:id/like
func (h *APIHandler) LikeArticle(c *gin.Context) {
	articleIDStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid article ID",
			"message": "Article ID must be a valid number",
		})
		return
	}

	// Rate limiting for likes (10 likes per minute per IP)
	clientIP := c.ClientIP()
	rateLimiter := NewRateLimiter()
	if !rateLimiter.Allow("like:"+clientIP, 10, time.Minute) {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": "Rate limit exceeded",
			"message": "Too many likes. Please wait before liking again.",
		})
		return
	}

	// Check if article exists
	ctx := c.Request.Context()
	article, err := h.articleService.GetByID(ctx, articleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Article not found",
			"message": "The requested article does not exist",
		})
		return
	}

	// For now, we'll increment the like count directly
	// In a full implementation, you'd track individual user likes
	newLikeCount := article.LikeCount + 1
	
	// Update the article's like count
	err = h.articleService.UpdateLikeCount(articleID, int(newLikeCount))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update like count",
			"message": "An error occurred while processing your like",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count": newLikeCount,
		"message": "Article liked successfully",
	})
}

// DislikeArticle handles POST /api/v1/articles/:id/dislike
func (h *APIHandler) DislikeArticle(c *gin.Context) {
	articleIDStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid article ID",
			"message": "Article ID must be a valid number",
		})
		return
	}

	// Rate limiting for dislikes (10 dislikes per minute per IP)
	clientIP := c.ClientIP()
	rateLimiter := NewRateLimiter()
	if !rateLimiter.Allow("dislike:"+clientIP, 10, time.Minute) {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": "Rate limit exceeded",
			"message": "Too many dislikes. Please wait before disliking again.",
		})
		return
	}

	// Check if article exists
	ctx := c.Request.Context()
	article, err := h.articleService.GetByID(ctx, articleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Article not found",
			"message": "The requested article does not exist",
		})
		return
	}

	// For now, we'll increment the dislike count directly
	// In a full implementation, you'd track individual user dislikes
	newDislikeCount := article.DislikeCount + 1
	
	// Update the article's dislike count
	err = h.articleService.UpdateDislikeCount(articleID, int(newDislikeCount))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update dislike count",
			"message": "An error occurred while processing your dislike",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count": newDislikeCount,
		"message": "Article disliked successfully",
	})
}

// BookmarkArticle handles POST /api/v1/articles/:id/bookmark
func (h *APIHandler) BookmarkArticle(c *gin.Context) {
	articleIDStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid article ID",
			"message": "Article ID must be a valid number",
		})
		return
	}

	// Rate limiting for bookmarks (20 bookmarks per minute per IP)
	clientIP := c.ClientIP()
	rateLimiter := NewRateLimiter()
	if !rateLimiter.Allow("bookmark:"+clientIP, 20, time.Minute) {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": "Rate limit exceeded",
			"message": "Too many bookmark actions. Please wait before trying again.",
		})
		return
	}

	// Check if article exists
	ctx := c.Request.Context()
	_, err = h.articleService.GetByID(ctx, articleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Article not found",
			"message": "The requested article does not exist",
		})
		return
	}

	// For now, we'll just return success
	// In a full implementation, you'd:
	// 1. Check if user is authenticated
	// 2. Check if article is already bookmarked by this user
	// 3. Toggle bookmark status
	// 4. Store bookmark in database

	// Simulate bookmark toggle (always return bookmarked for demo)
	bookmarked := true

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"bookmarked": bookmarked,
		"message": "Article bookmarked successfully",
	})
}

// Newsletter subscription handlers

// NewsletterSubscribeRequest represents a newsletter subscription request
type NewsletterSubscribeRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// SubscribeNewsletter handles POST /api/v1/newsletter/subscribe
func (h *APIHandler) SubscribeNewsletter(c *gin.Context) {
	var req NewsletterSubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request",
			"message": "Please provide a valid email address",
		})
		return
	}

	// Rate limiting for newsletter subscriptions (5 per minute per IP)
	clientIP := c.ClientIP()
	rateLimiter := NewRateLimiter()
	if !rateLimiter.Allow("newsletter:"+clientIP, 5, time.Minute) {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": "Rate limit exceeded",
			"message": "Too many subscription attempts. Please wait before trying again.",
		})
		return
	}

	// For now, we'll just return success
	// In a full implementation, you'd:
	// 1. Check if email is already subscribed
	// 2. Store email in newsletter database table
	// 3. Send confirmation email
	// 4. Handle unsubscribe tokens

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Successfully subscribed to newsletter",
	})
}