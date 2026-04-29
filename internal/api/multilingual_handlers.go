package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// MultilingualHandlers handles multilingual API endpoints
type MultilingualHandlers struct {
	multilingualService *services.MultilingualService
}

// NewMultilingualHandlers creates new multilingual handlers
func NewMultilingualHandlers(multilingualService *services.MultilingualService) *MultilingualHandlers {
	return &MultilingualHandlers{
		multilingualService: multilingualService,
	}
}

// GetLanguages returns all supported languages
// GET /api/v1/languages
func (h *MultilingualHandlers) GetLanguages(c *gin.Context) {
	languages, err := h.multilingualService.GetLanguages()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve languages",
			"message": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"languages": languages,
	})
}

// GetActiveLanguages returns only active languages
// GET /api/v1/languages/active
func (h *MultilingualHandlers) GetActiveLanguages(c *gin.Context) {
	languages, err := h.multilingualService.GetActiveLanguages()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve active languages",
			"message": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"languages": languages,
	})
}

// GetLanguageConfig returns the current language configuration
// GET /api/v1/languages/config
func (h *MultilingualHandlers) GetLanguageConfig(c *gin.Context) {
	config, err := h.multilingualService.GetLanguageConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve language configuration",
			"message": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"config": config,
	})
}

// CreateTranslationGroup creates a new translation group
// POST /api/v1/translations/groups
func (h *MultilingualHandlers) CreateTranslationGroup(c *gin.Context) {
	var request models.TranslationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"message": err.Error(),
		})
		return
	}
	
	// Validate request
	if request.GroupType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"message": "group_type is required",
		})
		return
	}
	
	if len(request.ContentIDs) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"message": "at least 2 content IDs are required",
		})
		return
	}
	
	groupID, err := h.multilingualService.CreateTranslationGroup(request.GroupType, request.ContentIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create translation group",
			"message": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"group_id": groupID,
		"message":  "Translation group created successfully",
	})
}

// GetArticleTranslations returns an article with its translations
// GET /api/v1/articles/:id/translations
func (h *MultilingualHandlers) GetArticleTranslations(c *gin.Context) {
	idStr := c.Param("id")
	articleID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid article ID",
			"message": "Article ID must be a valid number",
		})
		return
	}
	
	article, err := h.multilingualService.GetArticleTranslations(articleID)
	if err != nil {
		if err.Error() == "article not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Article not found",
				"message": "The requested article does not exist",
			})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve article translations",
			"message": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"article": article,
	})
}

// GetArticlesByLanguage returns articles in a specific language
// GET /api/v1/articles/language/:lang
func (h *MultilingualHandlers) GetArticlesByLanguage(c *gin.Context) {
	languageCode := c.Param("lang")
	
	// Validate language code
	if err := h.multilingualService.ValidateLanguageCode(languageCode); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid language code",
			"message": err.Error(),
		})
		return
	}
	
	// Parse pagination parameters
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")
	
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}
	
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}
	
	// Get fallback language
	fallbackLanguage := c.DefaultQuery("fallback", h.multilingualService.GetDefaultLanguage())
	
	articles, err := h.multilingualService.GetArticlesByLanguage(languageCode, fallbackLanguage, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve articles",
			"message": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"articles":        articles,
		"language_code":   languageCode,
		"fallback_language": fallbackLanguage,
		"limit":          limit,
		"offset":         offset,
		"count":          len(articles),
	})
}

// GetCategoriesByLanguage returns categories in a specific language
// GET /api/v1/categories/language/:lang
func (h *MultilingualHandlers) GetCategoriesByLanguage(c *gin.Context) {
	languageCode := c.Param("lang")
	
	// Validate language code
	if err := h.multilingualService.ValidateLanguageCode(languageCode); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid language code",
			"message": err.Error(),
		})
		return
	}
	
	categories, err := h.multilingualService.GetCategoriesByLanguage(languageCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve categories",
			"message": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"categories":    categories,
		"language_code": languageCode,
		"count":        len(categories),
	})
}

// GetTagsByLanguage returns tags in a specific language
// GET /api/v1/tags/language/:lang
func (h *MultilingualHandlers) GetTagsByLanguage(c *gin.Context) {
	languageCode := c.Param("lang")
	
	// Validate language code
	if err := h.multilingualService.ValidateLanguageCode(languageCode); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid language code",
			"message": err.Error(),
		})
		return
	}
	
	tags, err := h.multilingualService.GetTagsByLanguage(languageCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve tags",
			"message": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"tags":          tags,
		"language_code": languageCode,
		"count":        len(tags),
	})
}

// GetLanguageRouteInfo generates routing information for content
// GET /api/v1/routes/:type/:slug/language/:lang
func (h *MultilingualHandlers) GetLanguageRouteInfo(c *gin.Context) {
	contentType := c.Param("type")
	slug := c.Param("slug")
	languageCode := c.Param("lang")
	
	// Validate content type
	validTypes := map[string]bool{
		"articles":   true,
		"categories": true,
		"tags":       true,
	}
	if !validTypes[contentType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid content type",
			"message": "Content type must be one of: articles, categories, tags",
		})
		return
	}
	
	// Validate language code
	if err := h.multilingualService.ValidateLanguageCode(languageCode); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid language code",
			"message": err.Error(),
		})
		return
	}
	
	routeInfo, err := h.multilingualService.GenerateLanguageRouteInfo(contentType, slug, languageCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate route information",
			"message": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"route_info": routeInfo,
	})
}

// ValidateLanguageCode validates a language code
// GET /api/v1/languages/:code/validate
func (h *MultilingualHandlers) ValidateLanguageCode(c *gin.Context) {
	languageCode := c.Param("code")
	
	err := h.multilingualService.ValidateLanguageCode(languageCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"valid":   false,
			"error":   err.Error(),
		})
		return
	}
	
	// Get additional language info
	isRTL, _ := h.multilingualService.IsRTLLanguage(languageCode)
	isDefault := languageCode == h.multilingualService.GetDefaultLanguage()
	
	c.JSON(http.StatusOK, gin.H{
		"valid":      true,
		"code":       languageCode,
		"is_rtl":     isRTL,
		"is_default": isDefault,
	})
}

// RegisterMultilingualRoutes registers all multilingual routes
func RegisterMultilingualRoutes(router *gin.RouterGroup, handlers *MultilingualHandlers) {
	// Language management
	router.GET("/languages", handlers.GetLanguages)
	router.GET("/languages/active", handlers.GetActiveLanguages)
	router.GET("/languages/config", handlers.GetLanguageConfig)
	router.GET("/languages/:code/validate", handlers.ValidateLanguageCode)
	
	// Translation groups
	router.POST("/translations/groups", handlers.CreateTranslationGroup)
	
	// Content by language
	router.GET("/articles/:id/translations", handlers.GetArticleTranslations)
	router.GET("/articles/language/:lang", handlers.GetArticlesByLanguage)
	router.GET("/categories/language/:lang", handlers.GetCategoriesByLanguage)
	router.GET("/tags/language/:lang", handlers.GetTagsByLanguage)
	
	// Route information
	router.GET("/routes/:type/:slug/language/:lang", handlers.GetLanguageRouteInfo)
}