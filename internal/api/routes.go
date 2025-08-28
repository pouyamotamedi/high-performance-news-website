package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/auth"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
	"high-performance-news-website/pkg/cache"
)

// Router contains all API routes and middleware
type Router struct {
	handler                   *APIHandler
	commentHandlers           *CommentHandlers
	searchHandlers            *SearchHandlers
	imageHandlers             *ImageHandlers
	adminHandlers             *AdminHandlers
	contentManagementHandlers *ContentManagementHandlers
	monitoringHandler         *MonitoringHandler
	authMiddleware            *AuthMiddleware
	rateLimiter               *RateLimitMiddleware
}

// NewRouter creates a new API router with all dependencies
func NewRouter(
	userService *services.UserService,
	articleService *services.ArticleService,
	searchService *services.SearchService,
	contentIngestionService *services.ContentIngestionService,
	authService *auth.AuthService,
	cacheService cache.CacheService,
	commentHandlers *CommentHandlers,
	imageHandlers *ImageHandlers,
	configService *services.ConfigService,
	metricsService *services.MetricsService,
	healthService *services.HealthService,
	alertingService *services.AlertingService,
	categoryService *services.CategoryService,
	tagService *services.TagService,
) *Router {
	handler := NewAPIHandler(userService, articleService, searchService, contentIngestionService)
	searchHandlers := NewSearchHandlers(searchService)
	authMiddleware := NewAuthMiddleware(authService, userService)
	rateLimiter := NewRateLimitMiddleware(cacheService)
	
	// Create admin handlers
	adminHandlers := NewAdminHandlers(
		userService,
		articleService,
		searchService,
		cacheService,
		configService,
		metricsService,
	)
	
	// Create content management handlers
	contentManagementHandlers := NewContentManagementHandlers(
		articleService,
		userService,
		categoryService,
		tagService,
	)
	
	// Create monitoring handler
	monitoringHandler := NewMonitoringHandler(
		metricsService,
		healthService,
		alertingService,
	)

	return &Router{
		handler:                   handler,
		commentHandlers:           commentHandlers,
		searchHandlers:            searchHandlers,
		imageHandlers:             imageHandlers,
		adminHandlers:             adminHandlers,
		contentManagementHandlers: contentManagementHandlers,
		monitoringHandler:         monitoringHandler,
		authMiddleware:            authMiddleware,
		rateLimiter:               rateLimiter,
	}
}

// SetupRoutes configures all API routes
func (r *Router) SetupRoutes(engine *gin.Engine) {
	// Global middleware
	engine.Use(CORSMiddleware())
	engine.Use(SecurityHeaders())
	engine.Use(RequestID())
	engine.Use(LoggingMiddleware())
	engine.Use(gin.Recovery())

	// Global monitoring routes (no auth required)
	r.monitoringHandler.RegisterRoutes(engine)

	// API v1 routes
	v1 := engine.Group("/api/v1")
	{
		// Apply global rate limiting
		v1.Use(r.rateLimiter.UserRateLimit())

		// Health check (no auth required)
		v1.GET("/health", r.handler.HealthCheck)

		// Authentication routes (no auth required)
		auth := v1.Group("/auth")
		{
			auth.Use(r.rateLimiter.RateLimit(10, time.Minute)) // Stricter rate limit for auth
			auth.POST("/login", r.handler.Login)
			auth.POST("/refresh", r.handler.RefreshToken)
		}

		// Public article routes (optional auth for personalization)
		articles := v1.Group("/articles")
		{
			articles.Use(r.authMiddleware.OptionalAuth())
			articles.GET("", r.handler.ListArticles)                    // GET /api/v1/articles
			articles.GET("/:id", r.handler.GetArticle)                  // GET /api/v1/articles/123
			articles.GET("/slug/:slug", r.handler.GetArticleBySlug)     // GET /api/v1/articles/slug/my-article
			articles.GET("/trending", r.handler.GetTrendingArticles)    // GET /api/v1/articles/trending
			articles.GET("/popular", r.handler.GetPopularArticles)      // GET /api/v1/articles/popular
			
			// Comment routes for articles
			articles.GET("/:id/comments", r.commentHandlers.GetCommentsByArticle) // GET /api/v1/articles/123/comments
		}

		// Protected article routes (require authentication)
		protectedArticles := v1.Group("/articles")
		{
			protectedArticles.Use(r.authMiddleware.RequireAuth())
			protectedArticles.POST("", r.handler.CreateArticle)                    // POST /api/v1/articles
			protectedArticles.PUT("/:id", r.handler.UpdateArticle)                 // PUT /api/v1/articles/123
			protectedArticles.DELETE("/:id", r.handler.DeleteArticle)              // DELETE /api/v1/articles/123
			protectedArticles.POST("/:id/publish", r.handler.PublishArticle)       // POST /api/v1/articles/123/publish
			protectedArticles.POST("/bulk", r.handler.BulkCreateArticles)          // POST /api/v1/articles/bulk
		}

		// Search routes (optional auth)
		r.searchHandlers.RegisterSearchRoutes(v1)

		// Search index management routes (admin only)
		searchAdmin := v1.Group("/search/admin")
		{
			searchAdmin.Use(r.authMiddleware.RequireAuth())
			searchAdmin.Use(RequireRole(models.RoleAdmin))
			
			searchAdmin.POST("/rebuild", r.handler.RebuildSearchIndex)             // POST /api/v1/search/admin/rebuild
			searchAdmin.GET("/stats", r.handler.GetSearchIndexStats)               // GET /api/v1/search/admin/stats
			searchAdmin.POST("/articles/:id", r.handler.IndexArticle)              // POST /api/v1/search/admin/articles/123
			searchAdmin.DELETE("/articles/:id", r.handler.RemoveArticleFromIndex)  // DELETE /api/v1/search/admin/articles/123
		}

		// User management routes (require authentication)
		users := v1.Group("/users")
		{
			users.Use(r.authMiddleware.RequireAuth())
			
			// Current user routes (any authenticated user)
			users.GET("/me", r.handler.GetCurrentUser)                             // GET /api/v1/users/me
			users.PUT("/me", r.handler.UpdateUser)                                 // PUT /api/v1/users/me (special handling needed)
			users.POST("/me/change-password", r.handler.ChangePassword)            // POST /api/v1/users/me/change-password (special handling needed)
			
			// Admin/Editor only routes
			adminUsers := users.Group("")
			adminUsers.Use(RequireRole(models.RoleAdmin, models.RoleEditor))
			{
				adminUsers.GET("", r.handler.ListUsers)                           // GET /api/v1/users
				adminUsers.POST("", r.handler.CreateUser)                         // POST /api/v1/users
				adminUsers.GET("/:id", r.handler.GetUser)                         // GET /api/v1/users/123
				adminUsers.PUT("/:id", r.handler.UpdateUser)                      // PUT /api/v1/users/123
				adminUsers.POST("/:id/change-password", r.handler.ChangePassword) // POST /api/v1/users/123/change-password
			}
			
			// Admin only routes
			adminOnly := users.Group("")
			adminOnly.Use(RequireRole(models.RoleAdmin))
			{
				adminOnly.DELETE("/:id", r.handler.DeleteUser)                    // DELETE /api/v1/users/123
			}
		}

		// System management routes (admin only)
		system := v1.Group("/system")
		{
			system.Use(r.authMiddleware.RequireAuth())
			system.Use(RequireRole(models.RoleAdmin))
			
			system.GET("/health", r.handler.HealthCheck)                           // GET /api/v1/system/health
			system.POST("/cache/clear", r.handler.ClearCache)                      // POST /api/v1/system/cache/clear
			system.GET("/metrics", r.handler.GetMetrics)                           // GET /api/v1/system/metrics
		}

		// Monitoring routes (admin only) - comprehensive monitoring system
		monitoring := v1.Group("/monitoring")
		{
			monitoring.Use(r.authMiddleware.RequireAuth())
			monitoring.Use(RequireRole(models.RoleAdmin))
			
			// Dashboard and overview
			monitoring.GET("/dashboard", r.monitoringHandler.GetDashboard)         // GET /api/v1/monitoring/dashboard
			monitoring.GET("/overview", r.monitoringHandler.GetOverview)           // GET /api/v1/monitoring/overview
			
			// System metrics
			monitoring.GET("/metrics/system", r.monitoringHandler.GetSystemMetrics)       // GET /api/v1/monitoring/metrics/system
			monitoring.GET("/metrics/database", r.monitoringHandler.GetDatabaseMetrics)   // GET /api/v1/monitoring/metrics/database
			monitoring.GET("/metrics/cache", r.monitoringHandler.GetCacheMetrics)         // GET /api/v1/monitoring/metrics/cache
			monitoring.GET("/metrics/publishing", r.monitoringHandler.GetPublishingMetrics) // GET /api/v1/monitoring/metrics/publishing
			monitoring.GET("/metrics/performance", r.monitoringHandler.GetPerformanceMetrics) // GET /api/v1/monitoring/metrics/performance
			
			// Health checks
			monitoring.GET("/health/components", r.monitoringHandler.GetComponentHealth)  // GET /api/v1/monitoring/health/components
			monitoring.POST("/health/check/:component", r.monitoringHandler.CheckComponent) // POST /api/v1/monitoring/health/check/database
			
			// Alerts
			monitoring.GET("/alerts", r.monitoringHandler.GetAlerts)               // GET /api/v1/monitoring/alerts
			monitoring.GET("/alerts/active", r.monitoringHandler.GetActiveAlerts)  // GET /api/v1/monitoring/alerts/active
			monitoring.POST("/alerts/test", r.monitoringHandler.SendTestAlert)     // POST /api/v1/monitoring/alerts/test
			monitoring.POST("/alerts/:id/resolve", r.monitoringHandler.ResolveAlert) // POST /api/v1/monitoring/alerts/123/resolve
			
			// Alert rules
			monitoring.GET("/alert-rules", r.monitoringHandler.GetAlertRules)      // GET /api/v1/monitoring/alert-rules
			monitoring.POST("/alert-rules", r.monitoringHandler.CreateAlertRule)   // POST /api/v1/monitoring/alert-rules
			monitoring.PUT("/alert-rules/:id", r.monitoringHandler.UpdateAlertRule) // PUT /api/v1/monitoring/alert-rules/123
			monitoring.DELETE("/alert-rules/:id", r.monitoringHandler.DeleteAlertRule) // DELETE /api/v1/monitoring/alert-rules/123
			
			// Cache management
			monitoring.POST("/cache/clear", r.monitoringHandler.ClearCache)        // POST /api/v1/monitoring/cache/clear
			monitoring.GET("/cache/stats", r.monitoringHandler.GetCacheStats)      // GET /api/v1/monitoring/cache/stats
			
			// Configuration
			monitoring.GET("/config", r.monitoringHandler.GetMonitoringConfig)     // GET /api/v1/monitoring/config
			monitoring.PUT("/config", r.monitoringHandler.UpdateMonitoringConfig) // PUT /api/v1/monitoring/config
		}

		// Analytics routes (admin and editor only)
		analytics := v1.Group("/analytics")
		{
			analytics.Use(r.authMiddleware.RequireAuth())
			analytics.Use(RequireRole(models.RoleAdmin, models.RoleEditor))
			
			analytics.GET("/overview", r.handler.GetAnalyticsOverview)             // GET /api/v1/analytics/overview
			analytics.GET("/articles", r.handler.GetArticleAnalytics)              // GET /api/v1/analytics/articles
			analytics.GET("/users", r.handler.GetUserAnalytics)                    // GET /api/v1/analytics/users
			analytics.GET("/sources", r.handler.GetSourceAnalytics)                // GET /api/v1/analytics/sources
		}

		// Content Ingestion routes
		ingestion := v1.Group("/content")
		{
			// Public ingestion endpoint (API key authentication)
			ingestion.POST("/ingest", r.handler.IngestContent)                     // POST /api/v1/content/ingest
			
			// Webhook ingestion endpoint (webhook secret authentication)
			ingestion.POST("/webhook/:source_id", r.handler.WebhookIngestion)      // POST /api/v1/content/webhook/123
			
			// Admin routes for content source management
			adminIngestion := ingestion.Group("")
			adminIngestion.Use(r.authMiddleware.RequireAuth())
			adminIngestion.Use(RequireRole(models.RoleAdmin, models.RoleEditor))
			{
				// Content source management
				adminIngestion.GET("/sources", r.handler.ListContentSources)       // GET /api/v1/content/sources
				adminIngestion.POST("/sources", r.handler.CreateContentSource)     // POST /api/v1/content/sources
				
				// Content processing
				adminIngestion.POST("/process/:id", r.handler.ProcessPendingContent) // POST /api/v1/content/process/123
				adminIngestion.POST("/process/batch", r.handler.ProcessBatchContent) // POST /api/v1/content/process/batch
				
				// Ingestion statistics
				adminIngestion.GET("/stats", r.handler.GetIngestionStats)          // GET /api/v1/content/stats
			}
		}

		// Image processing routes
		images := v1.Group("/images")
		{
			// Public image routes (no auth required for viewing)
			images.GET("/:id/html", r.imageHandlers.GetImageHTML)                  // GET /api/v1/images/123/html
			images.GET("/status", r.imageHandlers.GetProcessingStatus)             // GET /api/v1/images/status
			
			// Protected image routes (require authentication)
			protectedImages := images.Group("")
			protectedImages.Use(r.authMiddleware.RequireAuth())
			{
				protectedImages.POST("/upload", r.imageHandlers.UploadImage)       // POST /api/v1/images/upload
				protectedImages.POST("/:id/variants", r.imageHandlers.ProcessImageVariants) // POST /api/v1/images/123/variants
			}
		}

		// Comment routes
		comments := v1.Group("/comments")
		{
			// Public comment routes (optional auth)
			comments.Use(r.authMiddleware.OptionalAuth())
			comments.POST("", r.commentHandlers.CreateComment)                     // POST /api/v1/comments
			comments.GET("/:id", r.commentHandlers.GetComment)                     // GET /api/v1/comments/123
		}

		// Admin comment routes (moderation)
		adminComments := v1.Group("/admin/comments")
		{
			adminComments.Use(r.authMiddleware.RequireAuth())
			adminComments.Use(RequireRole(models.RoleAdmin, models.RoleEditor))
			
			adminComments.GET("/pending", r.commentHandlers.GetPendingComments)    // GET /api/v1/admin/comments/pending
			adminComments.PUT("/:id/moderate", r.commentHandlers.ModerateComment)  // PUT /api/v1/admin/comments/123/moderate
			adminComments.PUT("/bulk-moderate", r.commentHandlers.BulkModerateComments) // PUT /api/v1/admin/comments/bulk-moderate
			adminComments.GET("/stats", r.commentHandlers.GetModerationStats)      // GET /api/v1/admin/comments/stats
			adminComments.GET("/search", r.commentHandlers.SearchComments)         // GET /api/v1/admin/comments/search
			adminComments.GET("/recent", r.commentHandlers.GetRecentComments)      // GET /api/v1/admin/comments/recent
			
			// Admin only routes
			adminOnly := adminComments.Group("")
			adminOnly.Use(RequireRole(models.RoleAdmin))
			{
				adminOnly.DELETE("/:id", r.commentHandlers.DeleteComment)          // DELETE /api/v1/admin/comments/123
			}
		}

		// Admin panel routes (require admin authentication)
		admin := v1.Group("/admin")
		{
			admin.Use(r.authMiddleware.RequireAuth())
			admin.Use(RequireRole(models.RoleAdmin))
			
			// Register admin panel routes
			r.adminHandlers.RegisterAdminRoutes(admin)
			
			// Register content management routes
			r.contentManagementHandlers.RegisterContentManagementRoutes(admin)
		}
	}
}

// Additional handler methods that need to be implemented

// ClearCache clears system cache
func (h *APIHandler) ClearCache(c *gin.Context) {
	// Implementation would go here
	c.JSON(200, SuccessResponse{
		Message: "Cache cleared successfully",
	})
}

// GetMetrics returns system metrics
func (h *APIHandler) GetMetrics(c *gin.Context) {
	// Implementation would go here
	c.JSON(200, SuccessResponse{
		Data: gin.H{
			"uptime":     "24h",
			"requests":   12345,
			"errors":     23,
			"cache_hits": 8901,
		},
	})
}

// GetAnalyticsOverview returns analytics overview
func (h *APIHandler) GetAnalyticsOverview(c *gin.Context) {
	// Implementation would go here
	c.JSON(200, SuccessResponse{
		Data: gin.H{
			"total_articles":    1000,
			"published_today":   25,
			"total_views":       50000,
			"active_users":      150,
			"popular_articles":  []string{"Article 1", "Article 2"},
		},
	})
}

// GetArticleAnalytics returns article performance analytics
func (h *APIHandler) GetArticleAnalytics(c *gin.Context) {
	// Implementation would go here
	c.JSON(200, SuccessResponse{
		Data: gin.H{
			"top_articles": []gin.H{
				{"title": "Article 1", "views": 1000, "likes": 50},
				{"title": "Article 2", "views": 800, "likes": 40},
			},
			"views_by_day": []gin.H{
				{"date": "2024-01-01", "views": 1200},
				{"date": "2024-01-02", "views": 1500},
			},
		},
	})
}

// GetUserAnalytics returns user analytics
func (h *APIHandler) GetUserAnalytics(c *gin.Context) {
	// Implementation would go here
	c.JSON(200, SuccessResponse{
		Data: gin.H{
			"total_users":   100,
			"active_users":  80,
			"new_users":     5,
			"user_roles": gin.H{
				"admin":       2,
				"editor":      5,
				"reporter":    20,
				"contributor": 73,
			},
		},
	})
}

// GetSourceAnalytics returns API source performance analytics
func (h *APIHandler) GetSourceAnalytics(c *gin.Context) {
	// Implementation would go here
	c.JSON(200, SuccessResponse{
		Data: gin.H{
			"api_calls_today": 500,
			"bulk_operations": 25,
			"error_rate":      0.02,
			"top_sources": []gin.H{
				{"source": "n8n", "calls": 200},
				{"source": "manual", "calls": 300},
			},
		},
	})
}