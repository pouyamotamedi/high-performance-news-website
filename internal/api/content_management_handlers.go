package api

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// ContentManagementHandlers handles admin content management operations
type ContentManagementHandlers struct {
	articleService  *services.ArticleService
	userService     *services.UserService
	categoryService *services.CategoryService
	tagService      *services.TagService
	mediaService    *services.MediaService
}

// NewContentManagementHandlers creates a new content management handlers instance
func NewContentManagementHandlers(
	articleService *services.ArticleService,
	userService *services.UserService,
	categoryService *services.CategoryService,
	tagService *services.TagService,
	mediaService *services.MediaService,
) *ContentManagementHandlers {
	return &ContentManagementHandlers{
		articleService:  articleService,
		userService:     userService,
		categoryService: categoryService,
		tagService:      tagService,
		mediaService:    mediaService,
	}
}

// ArticleManagementResponse represents article management data
type ArticleManagementResponse struct {
	ID                 uint64    `json:"id"`
	Title              string    `json:"title"`
	Slug               string    `json:"slug"`
	Status             string    `json:"status"`
	AuthorID           uint64    `json:"author_id"`
	AuthorName         string    `json:"author_name"`
	CategoryID         uint64    `json:"category_id"`
	CategoryName       string    `json:"category_name"`
	Tags               []string  `json:"tags"`
	Views              int64     `json:"views"`
	Likes              int64     `json:"likes"`
	Comments           int64     `json:"comments"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	PublishedAt        *time.Time `json:"published_at"`
	ScheduledAt        *time.Time `json:"scheduled_at"`
	WordCount          int       `json:"word_count"`
	ReadingTime        int       `json:"reading_time_minutes"`
	FeaturedImage      string    `json:"featured_image"`
	MetaDescription    string    `json:"meta_description"`
	LanguageCode       string    `json:"language_code"`
	TranslationGroupID *uint64   `json:"translation_group_id"`
}

// UserManagementResponse represents user management data
type UserManagementResponse struct {
	ID              uint64    `json:"id"`
	Username        string    `json:"username"`
	Email           string    `json:"email"`
	Role            string    `json:"role"`
	FirstName       string    `json:"first_name"`
	LastName        string    `json:"last_name"`
	IsActive        bool      `json:"is_active"`
	ArticlesCount   int64     `json:"articles_count"`
	LastLoginAt     *time.Time `json:"last_login_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CategoryManagementResponse represents category management data
type CategoryManagementResponse struct {
	ID                 uint64    `json:"id"`
	Name               string    `json:"name"`
	Slug               string    `json:"slug"`
	Description        string    `json:"description"`
	ParentID           *uint64   `json:"parent_id"`
	ParentName         string    `json:"parent_name,omitempty"`
	LanguageCode       string    `json:"language_code"`  
	TranslationGroupID *uint64   `json:"translation_group_id"`
	ImageURL           *string   `json:"image_url"`
	ImageAltText       *string   `json:"image_alt_text"`
	ArticlesCount      int64     `json:"articles_count"`
	IsActive           bool      `json:"is_active"`
	SortOrder          int       `json:"sort_order"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// TagManagementResponse represents tag management data
type TagManagementResponse struct {
    ID                 uint64    `json:"id"`
    Name               string    `json:"name"`
    Slug               string    `json:"slug"`
    Description        string    `json:"description"`
    Keywords           []string  `json:"keywords"`      
    Color              string    `json:"color"`
    LanguageCode       string    `json:"language_code"`
    TranslationGroupID *uint64   `json:"translation_group_id"`
    ArticlesCount      int64     `json:"articles_count"`
    IsActive           bool      `json:"is_active"`
    CreatedAt          time.Time `json:"created_at"`
    UpdatedAt          time.Time `json:"updated_at"`
}

// MediaManagementResponse represents media file management data
type MediaManagementResponse struct {
	ID          string                `json:"id"`
	Filename    string                `json:"filename"`
	OriginalURL string                `json:"original_url"`
	AltText     string                `json:"alt_text"`
	Caption     string                `json:"caption"`
	Width       int                   `json:"width"`
	Height      int                   `json:"height"`
	FileSize    int64                 `json:"file_size"`
	MimeType    string                `json:"mime_type"`
	Variants    []models.ImageVariant `json:"variants"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
}

// GetArticlesForManagement returns articles with management information
// GET /api/v1/admin/content/articles
func (cmh *ContentManagementHandlers) GetArticlesForManagement(c *gin.Context) {
	// Permission check
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Authentication required",
			Code:    "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	currentUser := user.(*models.User)
	if !currentUser.HasPermission("manage_system") && !currentUser.HasPermission("moderate") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Insufficient permissions",
			Code:    "FORBIDDEN",
			Message: "Content management access required",
		})
		return
	}

	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	status := c.Query("status")
	authorID := c.Query("author_id")
	categoryID := c.Query("category_id")
	search := c.Query("search")
	sortBy := c.DefaultQuery("sort_by", "created_at")
	sortOrder := c.DefaultQuery("sort_order", "desc")

	// Build filter options
	filters := map[string]interface{}{
		"page":       page,
		"limit":      limit,
		"status":     status,
		"author_id":  authorID,
		"category_id": categoryID,
		"search":     search,
		"sort_by":    sortBy,
		"sort_order": sortOrder,
	}

	articles, total, err := cmh.getArticlesWithManagementInfo(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get articles",
			Code:    "ARTICLES_FETCH_ERROR",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Data: gin.H{
			"articles": articles,
			"pagination": gin.H{
				"page":        page,
				"limit":       limit,
				"total":       total,
				"total_pages": (total + int64(limit) - 1) / int64(limit),
			},
		},
	})
}

// BulkUpdateArticles performs bulk operations on articles
// POST /api/v1/admin/content/articles/bulk
func (cmh *ContentManagementHandlers) BulkUpdateArticles(c *gin.Context) {
	// Permission check
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Authentication required",
			Code:    "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	currentUser := user.(*models.User)
	if !currentUser.HasPermission("manage_system") && !currentUser.HasPermission("moderate") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Insufficient permissions",
			Code:    "FORBIDDEN",
			Message: "Content management access required",
		})
		return
	}

	var request struct {
		ArticleIDs []uint64 `json:"article_ids" binding:"required"`
		Action     string   `json:"action" binding:"required"` // publish, unpublish, delete, archive
		CategoryID *uint64  `json:"category_id,omitempty"`
		TagIDs     []uint64 `json:"tag_ids,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Code:    "INVALID_JSON",
			Message: err.Error(),
		})
		return
	}

	results, err := cmh.performBulkArticleOperation(request.ArticleIDs, request.Action, currentUser.ID, request.CategoryID, request.TagIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Bulk operation failed",
			Code:    "BULK_OPERATION_ERROR",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Bulk operation completed",
		Data:    results,
	})
}

// GetUsersForManagement returns users with management information
// GET /api/v1/admin/content/users
func (cmh *ContentManagementHandlers) GetUsersForManagement(c *gin.Context) {
	// Permission check
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Authentication required",
			Code:    "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	currentUser := user.(*models.User)
	if !currentUser.HasPermission("manage_users") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Insufficient permissions",
			Code:    "FORBIDDEN",
			Message: "User management access required",
		})
		return
	}

	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	role := c.Query("role")
	status := c.Query("status") // active, inactive
	search := c.Query("search")
	sortBy := c.DefaultQuery("sort_by", "created_at")
	sortOrder := c.DefaultQuery("sort_order", "desc")

	filters := map[string]interface{}{
		"page":       page,
		"limit":      limit,
		"role":       role,
		"status":     status,
		"search":     search,
		"sort_by":    sortBy,
		"sort_order": sortOrder,
	}

	users, total, err := cmh.getUsersWithManagementInfo(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get users",
			Code:    "USERS_FETCH_ERROR",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Data: gin.H{
			"users": users,
			"pagination": gin.H{
				"page":        page,
				"limit":       limit,
				"total":       total,
				"total_pages": (total + int64(limit) - 1) / int64(limit),
			},
		},
	})
}

// GetCategoriesForManagement returns categories with management information
// GET /api/v1/admin/content/categories
func (cmh *ContentManagementHandlers) GetCategoriesForManagement(c *gin.Context) {
	// Permission check
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Authentication required",
			Code:    "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	currentUser := user.(*models.User)
	if !currentUser.HasPermission("manage_system") && !currentUser.HasPermission("moderate") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Insufficient permissions",
			Code:    "FORBIDDEN",
			Message: "Content management access required",
		})
		return
	}

	categories, err := cmh.getCategoriesWithManagementInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get categories",
			Code:    "CATEGORIES_FETCH_ERROR",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Data: gin.H{
			"categories": categories,
		},
	})
}

// GetTagsForManagement returns tags with management information
// GET /api/v1/admin/content/tags
func (cmh *ContentManagementHandlers) GetTagsForManagement(c *gin.Context) {
	// Permission check
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Authentication required",
			Code:    "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	currentUser := user.(*models.User)
	if !currentUser.HasPermission("manage_system") && !currentUser.HasPermission("moderate") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Insufficient permissions",
			Code:    "FORBIDDEN",
			Message: "Content management access required",
		})
		return
	}

	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	search := c.Query("search")
	sortBy := c.DefaultQuery("sort_by", "name")
	sortOrder := c.DefaultQuery("sort_order", "asc")

	filters := map[string]interface{}{
		"page":       page,
		"limit":      limit,
		"search":     search,
		"sort_by":    sortBy,
		"sort_order": sortOrder,
	}

	tags, total, err := cmh.getTagsWithManagementInfo(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get tags",
			Code:    "TAGS_FETCH_ERROR",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Data: gin.H{
			"tags": tags,
			"pagination": gin.H{
				"page":        page,
				"limit":       limit,
				"total":       total,
				"total_pages": (total + int64(limit) - 1) / int64(limit),
			},
		},
	})
}

// Helper methods (simplified implementations)

func (cmh *ContentManagementHandlers) getArticlesWithManagementInfo(filters map[string]interface{}) ([]ArticleManagementResponse, int64, error) {
	// Get real articles from ArticleService using List method
	limit := 20
	offset := 0
	
	if l, ok := filters["limit"].(int); ok && l > 0 {
		limit = l
	}
	if p, ok := filters["page"].(int); ok && p > 1 {
		offset = (p - 1) * limit
	}
	
	// Build article filters
	articleFilters := services.ArticleFilters{}
	
	// Get articles from service using List method
	ctx := context.Background()
	articles, total, err := cmh.articleService.List(ctx, limit, offset, articleFilters, "created_at", "desc")
	if err != nil {
		return nil, 0, err
	}
	
	// Convert to management response format
	var result []ArticleManagementResponse
	for _, article := range articles {
		authorName := "Unknown"
		if article.Author != nil {
			authorName = article.Author.FirstName + " " + article.Author.LastName
		}
		
		categoryName := ""
		var categoryID uint64
		if article.Category != nil {
			categoryName = article.Category.Name
			categoryID = article.Category.ID
		}
		
		// Extract tag names
		var tagNames []string
		for _, tag := range article.Tags {
			tagNames = append(tagNames, tag.Name)
		}
		
		result = append(result, ArticleManagementResponse{
			ID:                   article.ID,
			Title:                article.Title,
			Slug:                 article.Slug,
			Status:               article.Status,
			AuthorID:             article.AuthorID,
			AuthorName:           authorName,
			CategoryID:           categoryID,
			CategoryName:         categoryName,
			Tags:                 tagNames,
			Views:                int64(article.ViewCount),
			Likes:                0,
			Comments:             int64(article.CommentCount),
			CreatedAt:            article.CreatedAt,
			UpdatedAt:            article.UpdatedAt,
			WordCount:            article.WordCount,
			ReadingTime:          article.ReadTime,
			FeaturedImage:        article.FeaturedImage,
			MetaDescription:      article.MetaDescription,
			LanguageCode:         article.LanguageCode,
			TranslationGroupID:   article.TranslationGroupID,
		})
	}
	
	return result, int64(total), nil
}

func (cmh *ContentManagementHandlers) performBulkArticleOperation(articleIDs []uint64, action string, userID uint64, categoryID *uint64, tagIDs []uint64) (map[string]interface{}, error) {
	// This would perform the actual bulk operations
	// For now, returning mock results
	return map[string]interface{}{
		"processed":   len(articleIDs),
		"successful":  len(articleIDs),
		"failed":      0,
		"action":      action,
		"processed_at": time.Now(),
	}, nil
}

func (cmh *ContentManagementHandlers) getUsersWithManagementInfo(filters map[string]interface{}) ([]UserManagementResponse, int64, error) {
	// Mock implementation
	users := []UserManagementResponse{
		{
			ID:            1,
			Username:      "admin",
			Email:         "admin@example.com",
			Role:          "admin",
			FirstName:     "Admin",
			LastName:      "User",
			IsActive:      true,
			ArticlesCount: 25,
			CreatedAt:     time.Now().Add(-30 * 24 * time.Hour),
			UpdatedAt:     time.Now().Add(-1 * time.Hour),
		},
	}
	
	return users, 1, nil
}

func (cmh *ContentManagementHandlers) getCategoriesWithManagementInfo() ([]CategoryManagementResponse, error) {
    // Get real categories from CategoryService
    categories, err := cmh.categoryService.GetAll()
    if err != nil {
        return nil, err
    }
    
    // Convert to management response format
    var result []CategoryManagementResponse
    for _, cat := range categories {
		result = append(result, CategoryManagementResponse{
			ID:                 cat.ID,
			Name:               cat.Name,
			Slug:               cat.Slug,
			Description:        cat.Description,
			ParentID:           cat.ParentID,        
			LanguageCode:       cat.LanguageCode,
			TranslationGroupID: cat.TranslationGroupID,
			ImageURL:           cat.ImageURL,
			ImageAltText:       cat.ImageAltText,
			ArticlesCount:      0,
			IsActive:           true,
			SortOrder:          cat.SortOrder,
			CreatedAt:          cat.CreatedAt,
			UpdatedAt:          cat.UpdatedAt,
		})
    }
    
    return result, nil
}

func (cmh *ContentManagementHandlers) getTagsWithManagementInfo(filters map[string]interface{}) ([]TagManagementResponse, int64, error) {
	// Get real tags from TagService
	tags, err := cmh.tagService.GetAll()
	if err != nil {
		return nil, 0, err
	}
	
	// Convert to management response format
	var result []TagManagementResponse
	for _, tag := range tags {
		result = append(result, TagManagementResponse{
			ID:                 tag.ID,
			Name:               tag.Name,
			Slug:               tag.Slug,
			Description:        tag.Description,
			Keywords:           tag.Keywords,
			Color:              tag.Color,
			LanguageCode:       tag.LanguageCode,
			TranslationGroupID: tag.TranslationGroupID,
			ArticlesCount:      0,
			IsActive:           true,
			CreatedAt:          tag.CreatedAt,
			UpdatedAt:          tag.UpdatedAt,
		})
	}
	
	return result, int64(len(result)), nil
}

// RegisterContentManagementRoutes registers content management routes
func (cmh *ContentManagementHandlers) RegisterContentManagementRoutes(router *gin.RouterGroup) {
	content := router.Group("/content")
	{
		// Article management
		content.GET("/articles", cmh.GetArticlesForManagement)
		content.POST("/articles/bulk", cmh.BulkUpdateArticles)
		
		// User management
		content.GET("/users", cmh.GetUsersForManagement)
		
		// Category management
		content.GET("/categories", cmh.GetCategoriesForManagement)
		content.POST("/categories", cmh.CreateCategory)
		content.PUT("/categories/:id", cmh.UpdateCategory)
		content.DELETE("/categories/:id", cmh.DeleteCategory)
		content.DELETE("/categories/bulk", cmh.BulkDeleteCategories)
		content.POST("/categories/import", cmh.ImportCategoriesCSV)

		// Tag management
		content.GET("/tags", cmh.GetTagsForManagement)
		content.POST("/tags", cmh.CreateTag)
		content.PUT("/tags/:id", cmh.UpdateTag)
		content.DELETE("/tags/:id", cmh.DeleteTag)
		content.DELETE("/tags/bulk", cmh.BulkDeleteTags)
		content.POST("/tags/import", cmh.ImportTagsCSV)

		// Media management (add these lines)
		content.GET("/media", cmh.GetMediaForManagement)
		content.DELETE("/media/:id", cmh.DeleteMedia)

	}
}
// CreateCategory creates a new category
// POST /api/v1/admin/content/categories
func (cmh *ContentManagementHandlers) CreateCategory(c *gin.Context) {
	// Permission check
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Authentication required",
			Code:    "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	currentUser := user.(*models.User)
	if !currentUser.HasPermission("manage_system") && !currentUser.HasPermission("moderate") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Insufficient permissions",
			Code:    "FORBIDDEN",
			Message: "Content management access required",
		})
		return
	}

	var req struct {
		Name               string  `json:"name" binding:"required"`
		Slug               string  `json:"slug" binding:"required"`
		Description        string  `json:"description"`
		ParentID           *uint64 `json:"parent_id"`
		SortOrder          int     `json:"sort_order"`
		LanguageCode       string  `json:"language_code"`
		TranslationGroupID *uint64 `json:"translation_group_id"`
		ImageURL           *string `json:"image_url"`
		ImageAltText       *string `json:"image_alt_text"`
	}


	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}

	category := &models.Category{
		Name:               req.Name,
		Slug:               req.Slug,
		Description:        req.Description,
		ParentID:           req.ParentID,
		SortOrder:          req.SortOrder,
		LanguageCode:       req.LanguageCode,
		TranslationGroupID: req.TranslationGroupID,
		ImageURL:           req.ImageURL,
		ImageAltText:       req.ImageAltText,
	}


	if err := cmh.categoryService.Create(category); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to create category",
			Code:    "CATEGORY_CREATE_ERROR",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Data: gin.H{
			"category": category,
		},
	})
}

// UpdateCategory updates an existing category
// PUT /api/v1/admin/content/categories/:id
func (cmh *ContentManagementHandlers) UpdateCategory(c *gin.Context) {
	// Permission check
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Authentication required",
			Code:    "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	currentUser := user.(*models.User)
	if !currentUser.HasPermission("manage_system") && !currentUser.HasPermission("moderate") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Insufficient permissions",
			Code:    "FORBIDDEN",
			Message: "Content management access required",
		})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid category ID",
			Code:    "INVALID_ID",
			Message: "Category ID must be a valid number",
		})
		return
	}

	var req struct {
		Name               string  `json:"name" binding:"required"`
		Slug               string  `json:"slug" binding:"required"`
		Description        string  `json:"description"`
		ParentID           *uint64 `json:"parent_id"`
		SortOrder          int     `json:"sort_order"`
		LanguageCode       string  `json:"language_code"`
		TranslationGroupID *uint64 `json:"translation_group_id"`
		ImageURL           *string `json:"image_url"`
		ImageAltText       *string `json:"image_alt_text"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}

	category := &models.Category{
		ID:                 id,  // Add this line!
		Name:               req.Name,
		Slug:               req.Slug,
		Description:        req.Description,
		ParentID:           req.ParentID,
		SortOrder:          req.SortOrder,
		TranslationGroupID: req.TranslationGroupID,
		ImageURL:           req.ImageURL,
		ImageAltText:       req.ImageAltText,
		LanguageCode:       req.LanguageCode,
	}

	if err := cmh.categoryService.Update(category); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to update category",
			Code:    "CATEGORY_UPDATE_ERROR",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Data: gin.H{
			"category": category,
		},
	})
}

// DeleteCategory deletes a category
// DELETE /api/v1/admin/content/categories/:id
func (cmh *ContentManagementHandlers) DeleteCategory(c *gin.Context) {
	// Permission check
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Authentication required",
			Code:    "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	currentUser := user.(*models.User)
	if !currentUser.HasPermission("manage_system") && !currentUser.HasPermission("moderate") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Insufficient permissions",
			Code:    "FORBIDDEN",
			Message: "Content management access required",
		})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid category ID",
			Code:    "INVALID_ID",
			Message: "Category ID must be a valid number",
		})
		return
	}

	if err := cmh.categoryService.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to delete category",
			Code:    "CATEGORY_DELETE_ERROR",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Data: gin.H{
			"message": "Category deleted successfully",
		},
	})
}

// CreateTag creates a new tag
func (cmh *ContentManagementHandlers) CreateTag(c *gin.Context) {
	// Permission check
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Authentication required",
			Code:    "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	currentUser := user.(*models.User)
	if !currentUser.HasPermission("manage_system") && !currentUser.HasPermission("moderate") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Insufficient permissions",
			Code:    "FORBIDDEN",
			Message: "Content management access required",
		})
		return
	}

	var req struct {
		Name               string   `json:"name" binding:"required"`
		Slug               string   `json:"slug" binding:"required"`
		Description        string   `json:"description"`
		Keywords           []string `json:"keywords"`
		Color              string   `json:"color"`
		LanguageCode       string   `json:"language_code"`
		TranslationGroupID *uint64  `json:"translation_group_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}

	tag := &models.Tag{
		Name:               req.Name,
		Slug:               req.Slug,
		Description:        req.Description,
		Keywords:           req.Keywords,
		Color:              req.Color,
		LanguageCode:       req.LanguageCode,
		TranslationGroupID: req.TranslationGroupID,
	}

	if err := cmh.tagService.Create(tag); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to create tag",
			Code:    "TAG_CREATE_ERROR",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Data: gin.H{
			"tag": tag,
		},
	})
}

// UpdateTag updates an existing tag
func (cmh *ContentManagementHandlers) UpdateTag(c *gin.Context) {
	// Permission check
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Authentication required",
			Code:    "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	currentUser := user.(*models.User)
	if !currentUser.HasPermission("manage_system") && !currentUser.HasPermission("moderate") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Insufficient permissions",
			Code:    "FORBIDDEN",
			Message: "Content management access required",
		})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid tag ID",
			Code:    "INVALID_ID",
			Message: "Tag ID must be a valid number",
		})
		return
	}

	var req struct {
		Name               string   `json:"name" binding:"required"`
		Slug               string   `json:"slug" binding:"required"`
		Description        string   `json:"description"`
		Keywords           []string `json:"keywords"`
		Color              string   `json:"color"`
		LanguageCode       string   `json:"language_code"`
		TranslationGroupID *uint64  `json:"translation_group_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request data",
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}

	tag := &models.Tag{
		ID:                 id,
		Name:               req.Name,
		Slug:               req.Slug,
		Description:        req.Description,
		Keywords:           req.Keywords,
		Color:              req.Color,
		LanguageCode:       req.LanguageCode,
		TranslationGroupID: req.TranslationGroupID,
	}

	if err := cmh.tagService.Update(tag); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to update tag",
			Code:    "TAG_UPDATE_ERROR",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Data: gin.H{
			"tag": tag,
		},
	})
}

// DeleteTag deletes a tag
func (cmh *ContentManagementHandlers) DeleteTag(c *gin.Context) {
	// Permission check
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Authentication required",
			Code:    "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	currentUser := user.(*models.User)
	if !currentUser.HasPermission("manage_system") && !currentUser.HasPermission("moderate") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Insufficient permissions",
			Code:    "FORBIDDEN",
			Message: "Content management access required",
		})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid tag ID",
			Code:    "INVALID_ID",
			Message: "Tag ID must be a valid number",
		})
		return
	}

	if err := cmh.tagService.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to delete tag",
			Code:    "TAG_DELETE_ERROR",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Data: gin.H{
			"message": "Tag deleted successfully",
		},
	})
}


// GetMediaForManagement returns media files with management information
// GET /api/v1/admin/content/media
func (cmh *ContentManagementHandlers) GetMediaForManagement(c *gin.Context) {
	// Permission check
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Authentication required",
			Code:    "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	currentUser := user.(*models.User)
	if !currentUser.HasPermission("manage_system") && !currentUser.HasPermission("moderate") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Insufficient permissions",
			Code:    "FORBIDDEN",
			Message: "Content management access required",
		})
		return
	}

	// Check if mediaService is nil
	if cmh.mediaService == nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Media service not available",
			Code:    "SERVICE_ERROR",
			Message: "Media service is not initialized",
		})
		return
	}

	media, err := cmh.getMediaWithManagementInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get media",
			Code:    "MEDIA_FETCH_ERROR",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Data: gin.H{
			"media": media,
		},
	})
}

// DeleteMedia deletes a media file and its variants
// DELETE /api/v1/admin/content/media/:id
func (cmh *ContentManagementHandlers) DeleteMedia(c *gin.Context) {
	// Permission check
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Authentication required",
			Code:    "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	currentUser := user.(*models.User)
	if !currentUser.HasPermission("manage_system") && !currentUser.HasPermission("moderate") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Insufficient permissions",
			Code:    "FORBIDDEN",
			Message: "Content management access required",
		})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid media ID",
			Code:    "INVALID_ID",
			Message: "Media ID must be a valid number",
		})
		return
	}

	// Use MediaService to delete media
	if err := cmh.mediaService.DeleteMedia(id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to delete media",
			Code:    "MEDIA_DELETE_ERROR",
			Message: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse{
		Data: gin.H{
			"message": "Media deleted successfully",
		},
	})
}

// Helper function to get media with management info
func (cmh *ContentManagementHandlers) getMediaWithManagementInfo() ([]MediaManagementResponse, error) {
	// Check if mediaService is nil
	if cmh.mediaService == nil {
		return nil, fmt.Errorf("media service is nil")
	}
	
	// Get all media from MediaService
	images, err := cmh.mediaService.GetAllMedia()
	if err != nil {
		return nil, fmt.Errorf("failed to get all media: %w", err)
	}
	
	// Convert to management response format
	var result []MediaManagementResponse
	for _, img := range images {
		// Get variants for this image
		variants, err := cmh.mediaService.GetImageVariants(img.ID)
		if err != nil {
			// Log error but continue with other images
			variants = []models.ImageVariant{}
		}
		
		result = append(result, MediaManagementResponse{
			ID:          fmt.Sprintf("%d", img.ID),
			Filename:    img.Filename,
			OriginalURL: img.OriginalURL,
			AltText:     img.AltText,
			Caption:     img.Caption,
			Width:       img.Width,
			Height:      img.Height,
			FileSize:    img.FileSize,
			MimeType:    img.MimeType,
			Variants:    variants,
			CreatedAt:   img.CreatedAt,
			UpdatedAt:   img.UpdatedAt,
		})
	}
	
	return result, nil
}



// GetMedia handles GET /api/v1/admin/content/media
// BulkDeleteCategories deletes multiple categories
// DELETE /api/v1/admin/content/categories/bulk
func (cmh *ContentManagementHandlers) BulkDeleteCategories(c *gin.Context) {
	// Permission check
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Authentication required",
			Code:    "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	currentUser := user.(*models.User)
	if !currentUser.HasPermission("manage_system") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Insufficient permissions",
			Code:    "FORBIDDEN",
			Message: "System management access required",
		})
		return
	}

	var request struct {
		CategoryIDs []uint64 `json:"category_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Code:    "INVALID_JSON",
			Message: err.Error(),
		})
		return
	}

	if len(request.CategoryIDs) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "No categories specified",
			Code:    "INVALID_REQUEST",
			Message: "At least one category ID must be provided",
		})
		return
	}

	// Implement actual bulk delete logic
	deletedCount := 0
	var errors []string

	for _, categoryID := range request.CategoryIDs {
		// Check if category exists and can be deleted
		category, err := cmh.categoryService.GetByID(categoryID)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Category ID %d not found", categoryID))
			continue
		}

		// TODO: Check if category has child categories
		// This would require implementing GetChildren in CategoryService
		// For now, we'll proceed with deletion
		
		// TODO: Check if category has articles
		// This would require implementing GetArticleCount in CategoryService
		// For now, we'll proceed with deletion
		
		// Delete the category
		err = cmh.categoryService.Delete(categoryID)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to delete category '%s': %s", category.Name, err.Error()))
			continue
		}

		deletedCount++
	}

	// Return results
	response := SuccessResponse{
		Message: fmt.Sprintf("Bulk delete completed: %d categories deleted", deletedCount),
		Data: gin.H{
			"deleted_count":    deletedCount,
			"requested_count":  len(request.CategoryIDs),
			"error_count":      len(errors),
		},
	}

	if len(errors) > 0 {
		response.Data.(gin.H)["errors"] = errors
	}

	c.JSON(http.StatusOK, response)
}

// BulkDeleteTags deletes multiple tags
// DELETE /api/v1/admin/content/tags/bulk
func (cmh *ContentManagementHandlers) BulkDeleteTags(c *gin.Context) {
	// Permission check
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Authentication required",
			Code:    "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	currentUser := user.(*models.User)
	if !currentUser.HasPermission("manage_system") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Insufficient permissions",
			Code:    "FORBIDDEN",
			Message: "System management access required",
		})
		return
	}

	var request struct {
		TagIDs []uint64 `json:"tag_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Code:    "INVALID_JSON",
			Message: err.Error(),
		})
		return
	}

	if len(request.TagIDs) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "No tags specified",
			Code:    "INVALID_REQUEST",
			Message: "At least one tag ID must be provided",
		})
		return
	}

	// Implement actual bulk delete logic
	deletedCount := 0
	var errors []string

	for _, tagID := range request.TagIDs {
		// Check if tag exists
		tag, err := cmh.tagService.GetByID(tagID)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Tag ID %d not found", tagID))
			continue
		}

		// Delete the tag (this will also remove article-tag associations)
		err = cmh.tagService.Delete(tagID)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to delete tag '%s': %s", tag.Name, err.Error()))
			continue
		}

		deletedCount++
	}

	// Return results
	response := SuccessResponse{
		Message: fmt.Sprintf("Bulk delete completed: %d tags deleted", deletedCount),
		Data: gin.H{
			"deleted_count":    deletedCount,
			"requested_count":  len(request.TagIDs),
			"error_count":      len(errors),
		},
	}

	if len(errors) > 0 {
		response.Data.(gin.H)["errors"] = errors
	}

	c.JSON(http.StatusOK, response)
}

// ImportCategoriesCSV imports categories from CSV file
// POST /api/v1/admin/content/categories/import
func (cmh *ContentManagementHandlers) ImportCategoriesCSV(c *gin.Context) {
	// Permission check
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Authentication required",
			Code:    "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	currentUser := user.(*models.User)
	if !currentUser.HasPermission("manage_system") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Insufficient permissions",
			Code:    "FORBIDDEN",
			Message: "System management access required",
		})
		return
	}

	// Get uploaded file
	file, header, err := c.Request.FormFile("csv_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "No file uploaded",
			Code:    "FILE_REQUIRED",
			Message: "Please upload a CSV file",
		})
		return
	}
	defer file.Close()

	// Validate file type
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".csv") {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid file type",
			Code:    "INVALID_FILE_TYPE",
			Message: "Only CSV files are allowed",
		})
		return
	}

	// Parse CSV file
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	// Read header row
	headers, err := reader.Read()
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid CSV format",
			Code:    "CSV_PARSE_ERROR",
			Message: "Could not read CSV headers: " + err.Error(),
		})
		return
	}

	// Validate required headers
	requiredHeaders := []string{"name", "slug", "description"}
	headerMap := make(map[string]int)
	for i, header := range headers {
		headerMap[strings.ToLower(strings.TrimSpace(header))] = i
	}

	for _, required := range requiredHeaders {
		if _, exists := headerMap[required]; !exists {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Missing required column",
				Code:    "MISSING_COLUMN",
				Message: fmt.Sprintf("CSV must contain '%s' column", required),
			})
			return
		}
	}

	// Process CSV rows
	importedCount := 0
	skippedCount := 0
	errorCount := 0
	var errors []string

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			errorCount++
			errors = append(errors, fmt.Sprintf("Row parse error: %s", err.Error()))
			continue
		}

		// Skip empty rows
		if len(record) == 0 || strings.TrimSpace(record[0]) == "" {
			skippedCount++
			continue
		}

		// Create category from CSV row
		category := &models.Category{
			Name:         strings.TrimSpace(record[headerMap["name"]]),
			Slug:         strings.TrimSpace(record[headerMap["slug"]]),
			Description:  strings.TrimSpace(record[headerMap["description"]]),
			LanguageCode: "fa", // Default language
			SortOrder:    0,    // Default sort order
		}

		// Handle optional fields
		if parentIDIdx, exists := headerMap["parent_id"]; exists && len(record) > parentIDIdx {
			if parentIDStr := strings.TrimSpace(record[parentIDIdx]); parentIDStr != "" {
				if parentID, err := strconv.ParseUint(parentIDStr, 10, 64); err == nil {
					category.ParentID = &parentID
				}
			}
		}

		if sortOrderIdx, exists := headerMap["sort_order"]; exists && len(record) > sortOrderIdx {
			if sortOrderStr := strings.TrimSpace(record[sortOrderIdx]); sortOrderStr != "" {
				if sortOrder, err := strconv.Atoi(sortOrderStr); err == nil {
					category.SortOrder = sortOrder
				}
			}
		}

		if imageURLIdx, exists := headerMap["image_url"]; exists && len(record) > imageURLIdx {
			if imageURL := strings.TrimSpace(record[imageURLIdx]); imageURL != "" {
				category.ImageURL = &imageURL
			}
		}

		if imageAltIdx, exists := headerMap["image_alt_text"]; exists && len(record) > imageAltIdx {
			if imageAlt := strings.TrimSpace(record[imageAltIdx]); imageAlt != "" {
				category.ImageAltText = &imageAlt
			}
		}

		// Validate and create category
		if category.Name == "" {
			errorCount++
			errors = append(errors, "Category name cannot be empty")
			continue
		}

		// Try to create the category
		err = cmh.categoryService.Create(category)
		if err != nil {
			errorCount++
			errors = append(errors, fmt.Sprintf("Failed to create category '%s': %s", category.Name, err.Error()))
			continue
		}

		importedCount++
	}

	// Return results
	response := SuccessResponse{
		Message: fmt.Sprintf("CSV import completed: %d categories imported", importedCount),
		Data: gin.H{
			"filename":        header.Filename,
			"imported_count":  importedCount,
			"skipped_count":   skippedCount,
			"error_count":     errorCount,
		},
	}

	if len(errors) > 0 {
		response.Data.(gin.H)["errors"] = errors
	}

	c.JSON(http.StatusOK, response)
}

// ImportTagsCSV imports tags from CSV file
// POST /api/v1/admin/content/tags/import
func (cmh *ContentManagementHandlers) ImportTagsCSV(c *gin.Context) {
	// Permission check
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Authentication required",
			Code:    "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	currentUser := user.(*models.User)
	if !currentUser.HasPermission("manage_system") {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Insufficient permissions",
			Code:    "FORBIDDEN",
			Message: "System management access required",
		})
		return
	}

	// Get uploaded file
	file, header, err := c.Request.FormFile("csv_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "No file uploaded",
			Code:    "FILE_REQUIRED",
			Message: "Please upload a CSV file",
		})
		return
	}
	defer file.Close()

	// Validate file type
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".csv") {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid file type",
			Code:    "INVALID_FILE_TYPE",
			Message: "Only CSV files are allowed",
		})
		return
	}

	// Parse CSV file
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	// Read header row
	headers, err := reader.Read()
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid CSV format",
			Code:    "CSV_PARSE_ERROR",
			Message: "Could not read CSV headers: " + err.Error(),
		})
		return
	}

	// Validate required headers
	requiredHeaders := []string{"name", "slug"}
	headerMap := make(map[string]int)
	for i, header := range headers {
		headerMap[strings.ToLower(strings.TrimSpace(header))] = i
	}

	for _, required := range requiredHeaders {
		if _, exists := headerMap[required]; !exists {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Missing required column",
				Code:    "MISSING_COLUMN",
				Message: fmt.Sprintf("CSV must contain '%s' column", required),
			})
			return
		}
	}

	// Process CSV rows
	importedCount := 0
	skippedCount := 0
	errorCount := 0
	var errors []string

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			errorCount++
			errors = append(errors, fmt.Sprintf("Row parse error: %s", err.Error()))
			continue
		}

		// Skip empty rows
		if len(record) == 0 || strings.TrimSpace(record[0]) == "" {
			skippedCount++
			continue
		}

		// Create tag from CSV row
		tag := &models.Tag{
			Name:         strings.TrimSpace(record[headerMap["name"]]),
			Slug:         strings.TrimSpace(record[headerMap["slug"]]),
			LanguageCode: "fa", // Default language
			Color:        "#000000", // Default color
		}

		// Handle optional fields
		if descIdx, exists := headerMap["description"]; exists && len(record) > descIdx {
			tag.Description = strings.TrimSpace(record[descIdx])
		}

		if colorIdx, exists := headerMap["color"]; exists && len(record) > colorIdx {
			if color := strings.TrimSpace(record[colorIdx]); color != "" {
				tag.Color = color
			}
		}

		// Handle keywords (comma-separated or semicolon-separated)
		if keywordsIdx, exists := headerMap["keywords"]; exists && len(record) > keywordsIdx {
			if keywordsStr := strings.TrimSpace(record[keywordsIdx]); keywordsStr != "" {
				// Split by comma or semicolon
				var keywords []string
				if strings.Contains(keywordsStr, ";") {
					keywords = strings.Split(keywordsStr, ";")
				} else {
					keywords = strings.Split(keywordsStr, ",")
				}
				
				// Clean up keywords
				var cleanKeywords []string
				for _, keyword := range keywords {
					if cleaned := strings.TrimSpace(keyword); cleaned != "" {
						cleanKeywords = append(cleanKeywords, cleaned)
					}
				}
				tag.Keywords = cleanKeywords
			}
		}

		// Validate and create tag
		if tag.Name == "" {
			errorCount++
			errors = append(errors, "Tag name cannot be empty")
			continue
		}

		// Try to create the tag
		err = cmh.tagService.Create(tag)
		if err != nil {
			errorCount++
			errors = append(errors, fmt.Sprintf("Failed to create tag '%s': %s", tag.Name, err.Error()))
			continue
		}

		importedCount++
	}

	// Return results
	response := SuccessResponse{
		Message: fmt.Sprintf("CSV import completed: %d tags imported", importedCount),
		Data: gin.H{
			"filename":        header.Filename,
			"imported_count":  importedCount,
			"skipped_count":   skippedCount,
			"error_count":     errorCount,
		},
	}

	if len(errors) > 0 {
		response.Data.(gin.H)["errors"] = errors
	}

	c.JSON(http.StatusOK, response)
}