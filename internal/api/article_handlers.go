package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// Article API request/response types

// CreateArticleRequest represents a request to create an article
type CreateArticleRequest struct {
	Title      string             `json:"title" validate:"required,max=255"`
	Content    string             `json:"content" validate:"required"`
	Excerpt    string             `json:"excerpt" validate:"max=500"`
	CategoryID uint64             `json:"category_id" validate:"required"`
	Tags       []uint64           `json:"tags"`
	Status     string             `json:"status" validate:"required,oneof=draft published"`
	SEOData    models.SEOData     `json:"seo_data"`
}

// UpdateArticleRequest represents a request to update an article
type UpdateArticleRequest struct {
	Title      *string            `json:"title,omitempty" validate:"omitempty,max=255"`
	Content    *string            `json:"content,omitempty"`
	Excerpt    *string            `json:"excerpt,omitempty" validate:"omitempty,max=500"`
	CategoryID *uint64            `json:"category_id,omitempty"`
	Tags       []uint64           `json:"tags,omitempty"`
	Status     *string            `json:"status,omitempty" validate:"omitempty,oneof=draft published archived"`
	SEOData    *models.SEOData    `json:"seo_data,omitempty"`
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

	// Get current user
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// Create article model
	article := &models.Article{
		Title:      req.Title,
		Content:    req.Content,
		Excerpt:    req.Excerpt,
		AuthorID:   currentUser.ID,
		CategoryID: req.CategoryID,
		Status:     req.Status,
		SEOData:    req.SEOData,
	}

	// Create article through service
	createdArticle, err := h.articleService.Create(c.Request.Context(), article, currentUser)
	if err != nil {
		handleError(c, err)
		return
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

	// Get current user
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		handleError(c, err)
		return
	}

	// Update article through service
	updatedArticle, err := h.articleService.Update(c.Request.Context(), id, &req, currentUser)
	if err != nil {
		handleError(c, err)
		return
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

	// Delete article through service
	err = h.articleService.Delete(c.Request.Context(), id, currentUser)
	if err != nil {
		handleError(c, err)
		return
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
			Title:      articleReq.Title,
			Content:    articleReq.Content,
			Excerpt:    articleReq.Excerpt,
			AuthorID:   currentUser.ID,
			CategoryID: articleReq.CategoryID,
			Status:     articleReq.Status,
			SEOData:    articleReq.SEOData,
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