package api

import (
	"net/http"
	"strconv"
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
}

// NewContentManagementHandlers creates a new content management handlers instance
func NewContentManagementHandlers(
	articleService *services.ArticleService,
	userService *services.UserService,
	categoryService *services.CategoryService,
	tagService *services.TagService,
) *ContentManagementHandlers {
	return &ContentManagementHandlers{
		articleService:  articleService,
		userService:     userService,
		categoryService: categoryService,
		tagService:      tagService,
	}
}

// ArticleManagementResponse represents article management data
type ArticleManagementResponse struct {
	ID              uint64    `json:"id"`
	Title           string    `json:"title"`
	Slug            string    `json:"slug"`
	Status          string    `json:"status"`
	AuthorID        uint64    `json:"author_id"`
	AuthorName      string    `json:"author_name"`
	CategoryID      uint64    `json:"category_id"`
	CategoryName    string    `json:"category_name"`
	Tags            []string  `json:"tags"`
	Views           int64     `json:"views"`
	Likes           int64     `json:"likes"`
	Comments        int64     `json:"comments"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	PublishedAt     *time.Time `json:"published_at"`
	ScheduledAt     *time.Time `json:"scheduled_at"`
	WordCount       int       `json:"word_count"`
	ReadingTime     int       `json:"reading_time_minutes"`
	FeaturedImage   string    `json:"featured_image"`
	MetaDescription string    `json:"meta_description"`
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
	ID            uint64    `json:"id"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	Description   string    `json:"description"`
	ParentID      *uint64   `json:"parent_id"`
	ParentName    string    `json:"parent_name,omitempty"`
	ArticlesCount int64     `json:"articles_count"`
	IsActive      bool      `json:"is_active"`
	SortOrder     int       `json:"sort_order"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TagManagementResponse represents tag management data
type TagManagementResponse struct {
	ID            uint64    `json:"id"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	Description   string    `json:"description"`
	ArticlesCount int64     `json:"articles_count"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
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
	// This would typically query the database with joins to get all the management info
	// For now, returning mock data
	articles := []ArticleManagementResponse{
		{
			ID:              1,
			Title:           "Sample Article 1",
			Slug:            "sample-article-1",
			Status:          "published",
			AuthorID:        1,
			AuthorName:      "John Doe",
			CategoryID:      1,
			CategoryName:    "Technology",
			Tags:            []string{"tech", "news"},
			Views:           1500,
			Likes:           45,
			Comments:        12,
			CreatedAt:       time.Now().Add(-24 * time.Hour),
			UpdatedAt:       time.Now().Add(-2 * time.Hour),
			WordCount:       850,
			ReadingTime:     4,
			FeaturedImage:   "/images/sample1.jpg",
			MetaDescription: "A sample article about technology",
		},
	}
	
	return articles, 1, nil
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
	// Mock implementation
	categories := []CategoryManagementResponse{
		{
			ID:            1,
			Name:          "Technology",
			Slug:          "technology",
			Description:   "Technology related articles",
			ArticlesCount: 150,
			IsActive:      true,
			SortOrder:     1,
			CreatedAt:     time.Now().Add(-60 * 24 * time.Hour),
			UpdatedAt:     time.Now().Add(-5 * 24 * time.Hour),
		},
	}
	
	return categories, nil
}

func (cmh *ContentManagementHandlers) getTagsWithManagementInfo(filters map[string]interface{}) ([]TagManagementResponse, int64, error) {
	// Mock implementation
	tags := []TagManagementResponse{
		{
			ID:            1,
			Name:          "Technology",
			Slug:          "technology",
			Description:   "Technology related content",
			ArticlesCount: 75,
			IsActive:      true,
			CreatedAt:     time.Now().Add(-45 * 24 * time.Hour),
			UpdatedAt:     time.Now().Add(-3 * 24 * time.Hour),
		},
	}
	
	return tags, 1, nil
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
		
		// Tag management
		content.GET("/tags", cmh.GetTagsForManagement)
	}
}