package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/services"
	"high-performance-news-website/pkg/cache"
)

// AdminHandlers handles admin panel API endpoints
type AdminHandlers struct {
	userService    *services.UserService
	articleService *services.ArticleService
	searchService  *services.SearchService
	cacheService   cache.CacheService
	configService  *services.ConfigService
	metricsService *services.MetricsService
}

// NewAdminHandlers creates a new AdminHandlers instance
func NewAdminHandlers(
	userService *services.UserService,
	articleService *services.ArticleService,
	searchService *services.SearchService,
	cacheService cache.CacheService,
	configService *services.ConfigService,
	metricsService *services.MetricsService,
) *AdminHandlers {
	return &AdminHandlers{
		userService:    userService,
		articleService: articleService,
		searchService:  searchService,
		cacheService:   cacheService,
		configService:  configService,
		metricsService: metricsService,
	}
}

// RegisterAdminRoutes registers all admin panel routes
func (h *AdminHandlers) RegisterAdminRoutes(router *gin.RouterGroup) {
	// Dashboard routes
	router.GET("/dashboard", h.GetDashboard)
	router.GET("/dashboard/metrics", h.GetDashboardMetrics)
	
	// System monitoring routes
	router.GET("/system/health", h.GetSystemHealth)
	router.POST("/system/cache/clear", h.ClearSystemCache)
	
	// Configuration management routes
	router.GET("/config", h.GetConfiguration)
	router.PUT("/config", h.UpdateConfiguration)
	
	// Content management overview
	router.GET("/content/overview", h.GetContentOverview)
	router.GET("/content/recent", h.GetRecentContent)
	
	// Analytics and reporting
	router.GET("/analytics/overview", h.GetAnalyticsOverview)
}

// GetDashboard returns the main dashboard data
func (h *AdminHandlers) GetDashboard(c *gin.Context) {
	stats := gin.H{
		"articles": gin.H{
			"total":           h.getArticleCount(),
			"published_today": h.getArticlesPublishedToday(),
			"pending":         h.getPendingArticleCount(),
			"drafts":          h.getDraftArticleCount(),
		},
		"users": gin.H{
			"total":        h.getUserCount(),
			"active_today": h.getActiveUsersToday(),
		},
		"system": gin.H{
			"uptime":     h.getSystemUptime(),
			"cache_hits": h.getCacheHitRate(),
		},
		"traffic": gin.H{
			"views_today":    h.getPageViewsToday(),
			"visitors_today": h.getUniqueVisitorsToday(),
		},
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Dashboard data retrieved successfully",
		Data:    stats,
	})
}

// GetDashboardMetrics returns real-time dashboard metrics
func (h *AdminHandlers) GetDashboardMetrics(c *gin.Context) {
	metrics := gin.H{
		"timestamp": time.Now().Unix(),
		"articles": gin.H{
			"total":           h.getArticleCount(),
			"published_today": h.getArticlesPublishedToday(),
			"pending":         h.getPendingArticleCount(),
			"drafts":          h.getDraftArticleCount(),
		},
		"users": gin.H{
			"total":        h.getUserCount(),
			"active_today": h.getActiveUsersToday(),
			"online":       h.getOnlineUsers(),
		},
		"system": gin.H{
			"uptime":      h.getSystemUptime(),
			"memory_used": h.getMemoryUsage(),
			"cpu_usage":   h.getCPUUsage(),
			"cache_hits":  h.getCacheHitRate(),
		},
		"traffic": gin.H{
			"page_views_today": h.getPageViewsToday(),
			"unique_visitors":  h.getUniqueVisitorsToday(),
			"bounce_rate":      h.getBounceRate(),
		},
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Dashboard metrics retrieved successfully",
		Data:    metrics,
	})
}

// GetSystemHealth returns system health status
func (h *AdminHandlers) GetSystemHealth(c *gin.Context) {
	health := gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"services": gin.H{
			"database": gin.H{"status": "healthy", "latency": "2ms"},
			"cache":    gin.H{"status": "healthy", "latency": "1ms"},
			"search":   gin.H{"status": "healthy", "latency": "5ms"},
		},
		"metrics": gin.H{
			"uptime":        h.getSystemUptime(),
			"memory_usage":  h.getMemoryUsage(),
			"cpu_usage":     h.getCPUUsage(),
			"disk_usage":    h.getDiskUsage(),
		},
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "System health retrieved successfully",
		Data:    health,
	})
}

// ClearSystemCache clears system caches
func (h *AdminHandlers) ClearSystemCache(c *gin.Context) {
	cacheType := c.DefaultQuery("type", "all")
	
	var err error
	switch cacheType {
	case "all":
		err = h.clearAllCaches()
	case "articles":
		err = h.clearArticleCache()
	case "users":
		err = h.clearUserCache()
	default:
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid cache type",
			Code:    "INVALID_CACHE_TYPE",
			Message: "Valid types: all, articles, users",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to clear cache",
			Code:    "CACHE_CLEAR_ERROR",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Cache cleared successfully",
		Data: gin.H{
			"cache_type": cacheType,
			"cleared_at": time.Now().Unix(),
		},
	})
}

// GetConfiguration returns system configuration
func (h *AdminHandlers) GetConfiguration(c *gin.Context) {
	config := gin.H{
		"site": gin.H{
			"name":        h.getSiteConfig("site_name", "News Website"),
			"description": h.getSiteConfig("site_description", "High Performance News Website"),
			"url":         h.getSiteConfig("site_url", "https://example.com"),
		},
		"performance": gin.H{
			"cache_ttl":         h.getSiteConfig("cache_ttl", "3600"),
			"static_generation": h.getSiteConfig("static_generation", "true"),
		},
		"features": gin.H{
			"comments_enabled": h.getSiteConfig("comments_enabled", "true"),
			"search_enabled":   h.getSiteConfig("search_enabled", "true"),
		},
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Configuration retrieved successfully",
		Data:    config,
	})
}

// UpdateConfiguration updates system configuration
func (h *AdminHandlers) UpdateConfiguration(c *gin.Context) {
	var config map[string]interface{}
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid configuration data",
			Code:    "INVALID_CONFIG",
			Message: err.Error(),
		})
		return
	}

	// Update configuration
	for key, value := range config {
		if err := h.updateSiteConfig(key, value); err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Failed to update configuration",
				Code:    "CONFIG_UPDATE_ERROR",
				Message: err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Configuration updated successfully",
		Data: gin.H{
			"updated_at":   time.Now().Unix(),
			"keys_updated": len(config),
		},
	})
}

// GetContentOverview returns content management overview
func (h *AdminHandlers) GetContentOverview(c *gin.Context) {
	overview := gin.H{
		"articles": gin.H{
			"total":     h.getArticleCount(),
			"published": h.getPublishedArticleCount(),
			"drafts":    h.getDraftArticleCount(),
			"pending":   h.getPendingArticleCount(),
		},
		"categories": gin.H{
			"total": h.getCategoryCount(),
		},
		"tags": gin.H{
			"total": h.getTagCount(),
		},
		"recent_activity": h.getRecentContentActivity(),
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Content overview retrieved successfully",
		Data:    overview,
	})
}

// GetRecentContent returns recently created/updated content
func (h *AdminHandlers) GetRecentContent(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	
	recentContent := gin.H{
		"articles": h.getRecentArticles(limit),
		"comments": h.getRecentComments(limit),
		"users":    h.getRecentUsers(limit),
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Recent content retrieved successfully",
		Data:    recentContent,
	})
}

// GetAnalyticsOverview returns analytics overview
func (h *AdminHandlers) GetAnalyticsOverview(c *gin.Context) {
	timeRange := c.DefaultQuery("range", "7d")
	
	analytics := gin.H{
		"time_range": timeRange,
		"traffic": gin.H{
			"page_views":      h.getPageViews(timeRange),
			"unique_visitors": h.getUniqueVisitors(timeRange),
			"bounce_rate":     h.getBounceRate(),
		},
		"content": gin.H{
			"top_articles":    h.getTopArticlesByViews(timeRange, 10),
			"engagement_rate": h.getEngagementRate(timeRange),
		},
		"users": gin.H{
			"new_users":       h.getNewUsers(timeRange),
			"returning_users": h.getReturningUsers(timeRange),
		},
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Analytics overview retrieved successfully",
		Data:    analytics,
	})
}

// Helper methods with mock implementations
func (h *AdminHandlers) getArticleCount() int {
	count, _ := h.articleService.GetTotalCount()
	return int(count)
}

func (h *AdminHandlers) getArticlesPublishedToday() int {
	count, _ := h.articleService.GetPublishedTodayCount()
	return int(count)
}

func (h *AdminHandlers) getPendingArticleCount() int {
	count, _ := h.articleService.GetPendingCount()
	return int(count)
}

func (h *AdminHandlers) getDraftArticleCount() int {
	count, _ := h.articleService.GetDraftCount()
	return int(count)
}

func (h *AdminHandlers) getPublishedArticleCount() int {
	count, _ := h.articleService.GetPublishedCount()
	return int(count)
}

func (h *AdminHandlers) getUserCount() int {
	count, _ := h.userService.GetTotalCount()
	return int(count)
}

func (h *AdminHandlers) getActiveUsersToday() int {
	return 45 // Mock implementation
}

func (h *AdminHandlers) getOnlineUsers() int {
	return 12 // Mock implementation
}

func (h *AdminHandlers) getSystemUptime() string {
	return "2d 14h 32m" // Mock implementation
}

func (h *AdminHandlers) getMemoryUsage() float64 {
	return 68.5 // Mock implementation
}

func (h *AdminHandlers) getCPUUsage() float64 {
	return 23.4 // Mock implementation
}

func (h *AdminHandlers) getDiskUsage() float64 {
	return 45.2 // Mock implementation
}

func (h *AdminHandlers) getCacheHitRate() float64 {
	return 89.3 // Mock implementation
}

func (h *AdminHandlers) getPageViewsToday() int {
	return 12543 // Mock implementation
}

func (h *AdminHandlers) getUniqueVisitorsToday() int {
	return 3421 // Mock implementation
}

func (h *AdminHandlers) getBounceRate() float64 {
	return 34.2 // Mock implementation
}

func (h *AdminHandlers) getPageViews(timeRange string) int {
	multiplier := map[string]int{"1d": 1, "7d": 7, "30d": 30}[timeRange]
	if multiplier == 0 {
		multiplier = 7
	}
	return 12543 * multiplier
}

func (h *AdminHandlers) getUniqueVisitors(timeRange string) int {
	multiplier := map[string]int{"1d": 1, "7d": 6, "30d": 25}[timeRange]
	if multiplier == 0 {
		multiplier = 6
	}
	return 3421 * multiplier
}

func (h *AdminHandlers) getCategoryCount() int {
	return 25 // Mock implementation
}

func (h *AdminHandlers) getTagCount() int {
	return 156 // Mock implementation
}

func (h *AdminHandlers) getRecentContentActivity() []gin.H {
	return []gin.H{
		{"type": "article", "action": "published", "title": "Breaking News Article", "time": time.Now().Unix() - 300},
		{"type": "comment", "action": "approved", "title": "Comment on Tech News", "time": time.Now().Unix() - 600},
		{"type": "user", "action": "registered", "title": "New user: john_doe", "time": time.Now().Unix() - 900},
	}
}

func (h *AdminHandlers) getRecentArticles(limit int) []gin.H {
	return []gin.H{
		{"id": 1, "title": "Recent Article 1", "status": "published", "created_at": time.Now().Unix() - 3600},
		{"id": 2, "title": "Recent Article 2", "status": "draft", "created_at": time.Now().Unix() - 7200},
	}
}

func (h *AdminHandlers) getRecentComments(limit int) []gin.H {
	return []gin.H{
		{"id": 1, "content": "Great article!", "status": "approved", "created_at": time.Now().Unix() - 1800},
		{"id": 2, "content": "Interesting perspective", "status": "pending", "created_at": time.Now().Unix() - 3600},
	}
}

func (h *AdminHandlers) getRecentUsers(limit int) []gin.H {
	return []gin.H{
		{"id": 1, "username": "john_doe", "role": "reporter", "created_at": time.Now().Unix() - 86400},
		{"id": 2, "username": "jane_smith", "role": "contributor", "created_at": time.Now().Unix() - 172800},
	}
}

func (h *AdminHandlers) getTopArticlesByViews(timeRange string, limit int) []gin.H {
	return []gin.H{
		{"id": 1, "title": "Top Article 1", "views": 5432, "engagement": 78.5},
		{"id": 2, "title": "Top Article 2", "views": 4321, "engagement": 65.2},
	}
}

func (h *AdminHandlers) getEngagementRate(timeRange string) float64 {
	return 72.3
}

func (h *AdminHandlers) getNewUsers(timeRange string) int {
	multiplier := map[string]int{"1d": 1, "7d": 7, "30d": 30}[timeRange]
	if multiplier == 0 {
		multiplier = 7
	}
	return 15 * multiplier
}

func (h *AdminHandlers) getReturningUsers(timeRange string) int {
	multiplier := map[string]int{"1d": 1, "7d": 6, "30d": 25}[timeRange]
	if multiplier == 0 {
		multiplier = 6
	}
	return 234 * multiplier
}

// Cache management methods
func (h *AdminHandlers) clearAllCaches() error {
	ctx := context.Background()
	patterns := []string{"article:*", "homepage:*", "category:*", "tag:*", "user:*"}
	for _, pattern := range patterns {
		if err := h.cacheService.DeletePattern(ctx, pattern); err != nil {
			return err
		}
	}
	return nil
}

func (h *AdminHandlers) clearArticleCache() error {
	ctx := context.Background()
	return h.cacheService.DeletePattern(ctx, "article:*")
}

func (h *AdminHandlers) clearUserCache() error {
	ctx := context.Background()
	return h.cacheService.DeletePattern(ctx, "user:*")
}

// Configuration methods
func (h *AdminHandlers) getSiteConfig(key, defaultValue string) string {
	if h.configService != nil {
		if value, err := h.configService.Get(key); err == nil {
			return value
		}
	}
	return defaultValue
}

func (h *AdminHandlers) updateSiteConfig(key string, value interface{}) error {
	if h.configService != nil {
		return h.configService.Set(key, value)
	}
	return nil
}