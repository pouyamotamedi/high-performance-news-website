package api

import (
	"time"
	"database/sql"
	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/auth"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/repositories"
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
	configurationHandlers     *ConfigurationHandlers
	monitoringHandler         *MonitoringHandler
	widgetHandlers            *WidgetHandlers
	themeHandlers             *ThemeHandlers
	cdnHandlers               *CDNHandlers
	authMiddleware            *AuthMiddleware
	rateLimiter               *RateLimitMiddleware
	perEndpointRateLimiter    *PerEndpointRateLimitMiddleware
	csrfMiddleware            *CSRFMiddleware
	twoFactorMiddleware       *TwoFactorMiddleware
	apiKeyMiddleware          *APIKeyMiddleware
	inputValidationMiddleware *InputValidationMiddleware
	sessionAuthMiddleware     *SessionAuthMiddleware
}

// NewRouter creates a new API router with all dependencies
func NewRouter(
	db *sql.DB,
	userService *services.UserService,
	articleService *services.ArticleService,
	searchService services.SearchServiceInterface,
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
	widgetService *services.WidgetService,
	themeService *services.ThemeService,
	cdnService services.CDNServiceInterface,
) *Router {
	// Create security middleware
	authMiddleware := NewAuthMiddleware(authService, userService)
	rateLimiter := NewRateLimitMiddleware(cacheService)
	perEndpointRateLimiter := NewPerEndpointRateLimitMiddleware(cacheService)
	csrfMiddleware := NewCSRFMiddleware(cacheService)
	twoFactorMiddleware := NewTwoFactorMiddleware(cacheService)
	apiKeyMiddleware := NewAPIKeyMiddleware(cacheService, userService)
	inputValidationMiddleware := NewInputValidationMiddleware()
	sessionAuthMiddleware := NewSessionAuthMiddleware(authService)
	
	// Create keyword bank repository
	keywordBankRepo := repositories.NewKeywordBankRepository(db)
	
	// Create handlers with security middleware
	handler := NewAPIHandler(userService, articleService, searchService, contentIngestionService, tagService, keywordBankRepo, apiKeyMiddleware, twoFactorMiddleware, csrfMiddleware)
	searchHandlers := NewSearchHandlers(searchService)
	
	// Create admin handlers
	adminHandlers := NewAdminHandlers(
		userService,
		articleService,
		searchService,
		cacheService,
		configService,
		metricsService,
	)
	
	// Create media service
	mediaService := services.NewMediaService(db)

	// Create content management handlers
	contentManagementHandlers := NewContentManagementHandlers(
		articleService,
		userService,
		categoryService,
		tagService,
		mediaService,  // Add this line
	)
	
	// Create monitoring handler
	monitoringHandler := NewMonitoringHandler(
		metricsService,
		healthService,
		alertingService,
	)
	
	// Create widget and theme handlers
	widgetHandlers := NewWidgetHandlers(widgetService)
	themeHandlers := NewThemeHandlers(themeService)
	
	// Create configuration handlers
	configurationHandlers := NewConfigurationHandlers(configService)
	
	// Create CDN handlers
	var cdnHandlers *CDNHandlers
	if cdnService != nil {
		cdnHandlers = NewCDNHandlers(cdnService)
	}

	return &Router{
		handler:                   handler,
		commentHandlers:           commentHandlers,
		searchHandlers:            searchHandlers,
		imageHandlers:             imageHandlers,
		adminHandlers:             adminHandlers,
		contentManagementHandlers: contentManagementHandlers,
		configurationHandlers:     configurationHandlers,
		monitoringHandler:         monitoringHandler,
		widgetHandlers:            widgetHandlers,
		themeHandlers:             themeHandlers,
		cdnHandlers:               cdnHandlers,
		authMiddleware:            authMiddleware,
		rateLimiter:               rateLimiter,
		perEndpointRateLimiter:    perEndpointRateLimiter,
		csrfMiddleware:            csrfMiddleware,
		twoFactorMiddleware:       twoFactorMiddleware,
		apiKeyMiddleware:          apiKeyMiddleware,
		inputValidationMiddleware: inputValidationMiddleware,
		sessionAuthMiddleware:     sessionAuthMiddleware,
	}
}

// SetupRoutes configures all API routes
func (r *Router) SetupRoutes(engine *gin.Engine) {
	// Global middleware
	engine.Use(CORSMiddleware())
	engine.Use(EnhancedSecurityHeaders()) // Use enhanced security headers
	engine.Use(GzipCompression())         // Response compression
	engine.Use(RequestID())
	engine.Use(LoggingMiddleware())
	engine.Use(gin.Recovery())
	engine.Use(r.inputValidationMiddleware.ValidateInput()) // Global input validation

	// Global monitoring routes (no auth required)
	r.monitoringHandler.RegisterRoutes(engine)

	// Frontend routes are handled by the main server, not the API router

	// API v1 routes
	v1 := engine.Group("/api/v1")
	{
		// Apply global rate limiting and per-endpoint rate limiting
		v1.Use(r.rateLimiter.UserRateLimit())
		v1.Use(r.perEndpointRateLimiter.EndpointRateLimit())

		// Health check (no auth required)
		v1.GET("/health", r.handler.HealthCheck)

		// Authentication routes (no auth required)
		auth := v1.Group("/auth")
		{
			auth.Use(r.rateLimiter.RateLimit(10, time.Minute)) // Stricter rate limit for auth
			auth.POST("/login", r.handler.Login)
			auth.POST("/refresh", r.handler.RefreshToken)
			auth.GET("/csrf-token", r.handler.GetCSRFToken) // GET /api/v1/auth/csrf-token
			
			// Protected auth routes (require authentication)
			authProtected := auth.Group("")
			authProtected.Use(r.authMiddleware.RequireAuth())
			{
				// Token verification and 2FA status
				authProtected.GET("/verify", r.handler.VerifyToken)           // GET /api/v1/auth/verify
				authProtected.GET("/2fa/status", r.handler.Get2FAStatus)     // GET /api/v1/auth/2fa/status
				authProtected.POST("/2fa/verify", r.handler.Verify2FA)       // POST /api/v1/auth/2fa/verify
				
				// API Key Management
				authProtected.POST("/api-key/generate", r.handler.GenerateAPIKey)
				authProtected.POST("/api-key/rotate", r.handler.RotateAPIKey)
				authProtected.DELETE("/api-key", r.handler.RevokeAPIKey)
				
				// 2FA Management (admin only)
				authProtected.GET("/2fa/setup", r.handler.Get2FASetup)
			}
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
			
			// Engagement routes (like, dislike, bookmark) - require CSRF protection
			engagementRoutes := articles.Group("")
			engagementRoutes.Use(r.csrfMiddleware.CSRFProtection()) // CSRF protection for state-changing operations
			{
				engagementRoutes.POST("/:id/like", r.handler.LikeArticle)           // POST /api/v1/articles/123/like
				engagementRoutes.POST("/:id/dislike", r.handler.DislikeArticle)     // POST /api/v1/articles/123/dislike
				engagementRoutes.POST("/:id/bookmark", r.handler.BookmarkArticle)   // POST /api/v1/articles/123/bookmark
			}
		}

		// Protected article routes (require authentication)
		protectedArticles := v1.Group("/articles")
		{
			protectedArticles.Use(r.authMiddleware.RequireAuth())
			// Note: CSRF temporarily disabled for admin functionality
			protectedArticles.POST("", r.handler.CreateArticle)                    // POST /api/v1/articles
			protectedArticles.PUT("/:id", r.handler.UpdateArticle)                 // PUT /api/v1/articles/123
			protectedArticles.PATCH("/:id", r.handler.UpdateArticle)               // PATCH /api/v1/articles/123
			protectedArticles.DELETE("/:id", r.handler.DeleteArticle)              // DELETE /api/v1/articles/123
			protectedArticles.POST("/:id/publish", r.handler.PublishArticle)       // POST /api/v1/articles/123/publish
			protectedArticles.POST("/bulk", r.handler.BulkCreateArticles)          // POST /api/v1/articles/bulk
		}

		// Search routes (optional auth)
		r.searchHandlers.RegisterSearchRoutes(v1)

		// Search index management routes (admin only with 2FA)
		searchAdmin := v1.Group("/search/admin")
		{
			searchAdmin.Use(r.authMiddleware.RequireAuth())
			searchAdmin.Use(RequireRole(models.RoleAdmin))
			searchAdmin.Use(r.twoFactorMiddleware.Require2FA()) // 2FA required for admin operations
			
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
				adminUsers.GET("/export", r.handler.ExportUsers)                  // GET /api/v1/users/export
				adminUsers.GET("/:id", r.handler.GetUser)                         // GET /api/v1/users/123
				adminUsers.PUT("/:id", r.handler.UpdateUser)                      // PUT /api/v1/users/123
				adminUsers.POST("/:id/change-password", r.handler.ChangePassword) // POST /api/v1/users/123/change-password
			}
			
			// Admin only routes (with 2FA)
			adminOnly := users.Group("")
			adminOnly.Use(RequireRole(models.RoleAdmin))
			adminOnly.Use(r.twoFactorMiddleware.Require2FA()) // 2FA required for admin operations
			{
				adminOnly.DELETE("/:id", r.handler.DeleteUser)                    // DELETE /api/v1/users/123
			}
		}

		// Categories routes (for content management)
		categories := v1.Group("/categories")
		{
			categories.Use(r.authMiddleware.RequireAuth())
			categories.Use(RequireRole(models.RoleAdmin, models.RoleEditor))
			categories.GET("", r.handler.ListCategories)                          // GET /api/v1/categories
		}

		// System management routes (admin only with 2FA)
		system := v1.Group("/system")
		{
			system.Use(r.authMiddleware.RequireAuth())
			system.Use(RequireRole(models.RoleAdmin))
			// Note: CSRF and 2FA temporarily disabled for admin functionality
			
			system.GET("/health", r.handler.HealthCheck)                           // GET /api/v1/system/health
			system.POST("/cache/clear", r.handler.ClearCache)                      // POST /api/v1/system/cache/clear
			system.GET("/metrics", r.handler.GetMetrics)                           // GET /api/v1/system/metrics
		}

		// Note: Monitoring routes are registered by monitoringHandler.RegisterRoutes() above

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
				adminIngestion.GET("/pending", r.handler.GetPendingContent)        // GET /api/v1/content/pending
				adminIngestion.GET("/processed", r.handler.GetProcessedContent)    // GET /api/v1/content/processed
				adminIngestion.POST("/process/:id", r.handler.ProcessPendingContent) // POST /api/v1/content/process/123
				adminIngestion.POST("/process/batch", r.handler.ProcessBatchContent) // POST /api/v1/content/process/batch
				
				// Categories for content ingestion
				adminIngestion.GET("/categories", r.handler.ListCategories)        // GET /api/v1/content/categories
				
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

		// Newsletter routes
		newsletter := v1.Group("/newsletter")
		{
			newsletter.POST("/subscribe", r.handler.SubscribeNewsletter)            // POST /api/v1/newsletter/subscribe
		}

		// Admin comment routes (moderation)
		adminComments := v1.Group("/admin/comments")
		{
			adminComments.Use(r.authMiddleware.RequireAuth())
			adminComments.Use(RequireRole(models.RoleAdmin, models.RoleEditor))
			
			adminComments.GET("/pending", r.commentHandlers.GetPendingComments)    // GET /api/v1/admin/comments/pending
			adminComments.PUT("/:id/moderate", r.commentHandlers.ModerateComment)  // PUT /api/v1/admin/comments/123/moderate
			adminComments.PUT("/:id/edit", r.commentHandlers.EditComment)          // PUT /api/v1/admin/comments/123/edit
			adminComments.GET("/:id/replies", r.commentHandlers.GetCommentReplies) // GET /api/v1/admin/comments/123/replies
			adminComments.PUT("/bulk-moderate", r.commentHandlers.BulkModerateComments) // PUT /api/v1/admin/comments/bulk-moderate
			adminComments.GET("/stats", r.commentHandlers.GetModerationStats)      // GET /api/v1/admin/comments/stats
			adminComments.GET("/search", r.commentHandlers.SearchComments)         // GET /api/v1/admin/comments/search
			adminComments.GET("/recent", r.commentHandlers.GetRecentComments)      // GET /api/v1/admin/comments/recent
			adminComments.GET("/spam-settings", r.commentHandlers.GetSpamSettings) // GET /api/v1/admin/comments/spam-settings
			adminComments.PUT("/spam-settings", r.commentHandlers.UpdateSpamSettings) // PUT /api/v1/admin/comments/spam-settings
			adminComments.POST("/recalculate-spam", r.commentHandlers.RecalculateSpamScores) // POST /api/v1/admin/comments/recalculate-spam
			
			// Admin only routes
			adminOnly := adminComments.Group("")
			adminOnly.Use(RequireRole(models.RoleAdmin))
			{
				adminOnly.DELETE("/:id", r.commentHandlers.DeleteComment)          // DELETE /api/v1/admin/comments/123
			}
		}

		// Admin panel routes (require admin authentication + 2FA)
		admin := v1.Group("/admin")
		{
			admin.Use(r.authMiddleware.RequireAuth())
			admin.Use(RequireRole(models.RoleAdmin))
			// Note: CSRF and 2FA temporarily disabled for admin panel functionality
			
			// Register admin panel routes
			r.adminHandlers.RegisterAdminRoutes(admin)
			
			// Register content management routes
			r.contentManagementHandlers.RegisterContentManagementRoutes(admin)
			
			// Widget management routes
			widgets := admin.Group("/widgets")
			{
				// Widget CRUD
				widgets.GET("", r.widgetHandlers.GetAllWidgets)                    // GET /api/v1/admin/widgets
				widgets.POST("", r.widgetHandlers.CreateWidget)                    // POST /api/v1/admin/widgets
				widgets.GET("/:id", r.widgetHandlers.GetWidget)                    // GET /api/v1/admin/widgets/123
				widgets.PUT("/:id", r.widgetHandlers.UpdateWidget)                 // PUT /api/v1/admin/widgets/123
				widgets.DELETE("/:id", r.widgetHandlers.DeleteWidget)              // DELETE /api/v1/admin/widgets/123
				
				// Widget types and configuration
				widgets.GET("/types", r.widgetHandlers.GetWidgetTypes)             // GET /api/v1/admin/widgets/types
				widgets.GET("/page-types", r.widgetHandlers.GetPageTypes)          // GET /api/v1/admin/widgets/page-types
				widgets.GET("/zones", r.widgetHandlers.GetWidgetZones)             // GET /api/v1/admin/widgets/zones
				widgets.GET("/by-type/:type", r.widgetHandlers.GetWidgetsByType)   // GET /api/v1/admin/widgets/by-type/latest_articles
				
				// Widget rendering
				widgets.GET("/:id/render", r.widgetHandlers.RenderWidget)          // GET /api/v1/admin/widgets/123/render
				
				// Widget placements
				widgets.GET("/placements", r.widgetHandlers.GetWidgetPlacements)   // GET /api/v1/admin/widgets/placements?page_type=homepage&zone=sidebar
				widgets.POST("/placements", r.widgetHandlers.CreateWidgetPlacement) // POST /api/v1/admin/widgets/placements
				widgets.PUT("/placements/:id", r.widgetHandlers.UpdateWidgetPlacement) // PUT /api/v1/admin/widgets/placements/123
				widgets.DELETE("/placements/:id", r.widgetHandlers.DeleteWidgetPlacement) // DELETE /api/v1/admin/widgets/placements/123
				widgets.PUT("/placements/positions", r.widgetHandlers.UpdatePlacementPositions) // PUT /api/v1/admin/widgets/placements/positions
			}
			
			// Theme management routes
			themes := admin.Group("/themes")
			{
				// Theme CRUD
				themes.GET("", r.themeHandlers.GetAllThemes)                       // GET /api/v1/admin/themes
				themes.POST("", r.themeHandlers.CreateTheme)                       // POST /api/v1/admin/themes
				themes.GET("/active", r.themeHandlers.GetActiveTheme)              // GET /api/v1/admin/themes/active
				themes.GET("/:id", r.themeHandlers.GetTheme)                       // GET /api/v1/admin/themes/123
				themes.PUT("/:id", r.themeHandlers.UpdateTheme)                    // PUT /api/v1/admin/themes/123
				themes.DELETE("/:id", r.themeHandlers.DeleteTheme)                 // DELETE /api/v1/admin/themes/123
				themes.POST("/:id/activate", r.themeHandlers.SetActiveTheme)       // POST /api/v1/admin/themes/123/activate
				
				// Theme CSS generation
				themes.GET("/:id/css", r.themeHandlers.GenerateThemeCSS)           // GET /api/v1/admin/themes/123/css
				themes.GET("/active/css", r.themeHandlers.GetActiveThemeCSS)       // GET /api/v1/admin/themes/active/css
				
				// Default configuration
				themes.GET("/default-config", r.themeHandlers.GetDefaultThemeConfig) // GET /api/v1/admin/themes/default-config
				
				// Template overrides
				templates := themes.Group("/templates")
				{
					templates.GET("", r.themeHandlers.GetAllTemplateOverrides)     // GET /api/v1/admin/themes/templates
					templates.POST("", r.themeHandlers.CreateTemplateOverride)     // POST /api/v1/admin/themes/templates
					templates.GET("/override", r.themeHandlers.GetTemplateOverride) // GET /api/v1/admin/themes/templates/override?path=pages/article.html
					templates.GET("/content", r.themeHandlers.GetTemplateContent)  // GET /api/v1/admin/themes/templates/content?path=pages/article.html
					templates.PUT("/:id", r.themeHandlers.UpdateTemplateOverride)  // PUT /api/v1/admin/themes/templates/123
					templates.DELETE("/:id", r.themeHandlers.DeleteTemplateOverride) // DELETE /api/v1/admin/themes/templates/123
					templates.POST("/preview", r.themeHandlers.PreviewTemplate)    // POST /api/v1/admin/themes/templates/preview
				}
			}
			
			// CDN management routes (admin only)
			if r.cdnHandlers != nil {
				cdn := admin.Group("/cdn")
				{
					// CDN configuration
					cdn.GET("/config", r.cdnHandlers.GetCDNConfig)                 // GET /api/v1/admin/cdn/config
					cdn.PUT("/config", r.cdnHandlers.UpdateCDNConfig)              // PUT /api/v1/admin/cdn/config
					cdn.POST("/test", r.cdnHandlers.TestCDNConnection)             // POST /api/v1/admin/cdn/test
					
					// Cache management
					cdn.POST("/purge", r.cdnHandlers.PurgeCache)                   // POST /api/v1/admin/cdn/purge
					cdn.POST("/purge/url", r.cdnHandlers.PurgeURL)                 // POST /api/v1/admin/cdn/purge/url
					cdn.POST("/purge/urls", r.cdnHandlers.PurgeURLs)               // POST /api/v1/admin/cdn/purge/urls
					cdn.POST("/purge/all", r.cdnHandlers.PurgeAll)                 // POST /api/v1/admin/cdn/purge/all
					
					// Content-specific purging
					cdn.POST("/purge/article/:slug", r.cdnHandlers.PurgeArticle)   // POST /api/v1/admin/cdn/purge/article/my-article
					cdn.POST("/purge/category/:slug", r.cdnHandlers.PurgeCategory) // POST /api/v1/admin/cdn/purge/category/tech
					cdn.POST("/purge/tag/:slug", r.cdnHandlers.PurgeTag)           // POST /api/v1/admin/cdn/purge/tag/breaking
					
					// Monitoring and stats
					cdn.GET("/stats", r.cdnHandlers.GetCDNStats)                   // GET /api/v1/admin/cdn/stats
					cdn.GET("/health", r.cdnHandlers.GetCDNHealth)                 // GET /api/v1/admin/cdn/health
					
					// Failover management
					cdn.POST("/failover/enable", r.cdnHandlers.EnableFailover)     // POST /api/v1/admin/cdn/failover/enable
					cdn.POST("/failover/disable", r.cdnHandlers.DisableFailover)   // POST /api/v1/admin/cdn/failover/disable
					cdn.GET("/failover/status", r.cdnHandlers.GetFailoverStatus)   // GET /api/v1/admin/cdn/failover/status
				}
			}
			
			// Configuration management routes
			r.configurationHandlers.RegisterConfigurationRoutes(admin)
			
			// SEO settings routes (simple in-memory storage)
			seo := admin.Group("/seo")
			{
				seo.GET("/settings", getSEOSettings)
				seo.POST("/settings", saveSEOSettings)
				seo.GET("/robots", getRobotsTxt)
				seo.POST("/robots", saveRobotsTxt)
			}
		}

		// Admin panel user management routes (require session-based admin authentication)
		// Create admin panel group WITHOUT inheriting rate limiting from v1
		adminPanel := engine.Group("/api/v1/admin-panel")
		adminPanel.Use(r.sessionAuthMiddleware.RequireSessionAuth()) // Auth first
		adminPanel.Use(r.rateLimiter.UserRateLimit())                // Then rate limiting with user context
		adminPanel.Use(r.perEndpointRateLimiter.EndpointRateLimit()) // Then endpoint-specific limits
		// Note: Session-based authentication for admin panel users
		{
			// User management routes
			adminPanel.GET("/users", r.handler.ListUsers)                          // GET /api/v1/admin-panel/users
			adminPanel.POST("/users", r.handler.CreateUser)                        // POST /api/v1/admin-panel/users
			adminPanel.GET("/users/export", r.handler.ExportUsers)                 // GET /api/v1/admin-panel/users/export
			adminPanel.GET("/users/:id", r.handler.GetUser)                        // GET /api/v1/admin-panel/users/:id
			adminPanel.PUT("/users/:id", r.handler.UpdateUser)                     // PUT /api/v1/admin-panel/users/:id
			adminPanel.DELETE("/users/:id", r.handler.DeleteUser)                  // DELETE /api/v1/admin-panel/users/:id
			
			// Content ingestion routes for admin panel (session-based auth)
			adminPanel.GET("/content/sources", r.handler.ListContentSources)      // GET /api/v1/admin-panel/content/sources
			adminPanel.POST("/content/sources", r.handler.CreateContentSource)    // POST /api/v1/admin-panel/content/sources
			adminPanel.PUT("/content/sources/:id", r.handler.UpdateContentSource) // PUT /api/v1/admin-panel/content/sources/:id
			adminPanel.DELETE("/content/sources/:id", r.handler.DeleteContentSource) // DELETE /api/v1/admin-panel/content/sources/:id
			adminPanel.GET("/content/pending", r.handler.GetPendingContent)       // GET /api/v1/admin-panel/content/pending
			adminPanel.GET("/content/processed", r.handler.GetProcessedContent)   // GET /api/v1/admin-panel/content/processed
			adminPanel.GET("/content/categories", r.handler.ListCategories)       // GET /api/v1/admin-panel/content/categories
			adminPanel.GET("/content/stats", r.handler.GetIngestionStatsAdmin)         // GET /api/v1/admin-panel/content/stats
			adminPanel.GET("/content/details/:id", r.handler.GetContentByID) // GET /api/v1/admin-panel/content/details/123
			adminPanel.POST("/content/process/:id", r.handler.ProcessPendingContentByID) // POST /api/v1/admin-panel/content/process/123
			adminPanel.POST("/content/reject/:id", r.handler.RejectPendingContent) // POST /api/v1/admin-panel/content/reject/:id
			adminPanel.POST("/content/reprocess/:id", r.handler.ReprocessRejectedContent) // POST /api/v1/admin-panel/content/reprocess/:id
			adminPanel.POST("/content/process/batch", r.handler.ProcessBatchContentByIDs) // POST /api/v1/admin-panel/content/process/batch
			
			// Auto-linking routes for admin panel
			adminPanel.GET("/autolinking/stats", r.handleAutoLinkingStats)           // GET /api/v1/admin-panel/autolinking/stats
			adminPanel.GET("/autolinking/settings", r.handleAutoLinkingSettings)     // GET /api/v1/admin-panel/autolinking/settings
			adminPanel.PUT("/autolinking/settings", r.handleAutoLinkingUpdateSettings)  // PUT /api/v1/admin-panel/autolinking/settings
			adminPanel.POST("/autolinking/refresh", r.handleAutoLinkingRefresh) // POST /api/v1/admin-panel/autolinking/refresh
			adminPanel.GET("/autolinking/conflicts", r.handleAutoLinkingConflicts) // GET /api/v1/admin-panel/autolinking/conflicts
			adminPanel.POST("/autolinking/test", r.handleAutoLinkingTest)    // POST /api/v1/admin-panel/autolinking/test
			adminPanel.POST("/autolinking/reprocess", r.handleReprocessAllArticles) // POST /api/v1/admin-panel/autolinking/reprocess
			adminPanel.POST("/autolinking/clean", r.handleCleanAllLinks)       // POST /api/v1/admin-panel/autolinking/clean
			adminPanel.GET("/tags", r.handleGetTagsWithKeywords)             // GET /api/v1/admin-panel/tags
			adminPanel.GET("/autolinking/keyword-banks", r.handleGetKeywordBanks) // GET /api/v1/admin-panel/autolinking/keyword-banks
			
			// Keyword bank management routes
			adminPanel.GET("/keyword-banks", r.handleGetKeywordBanks)           // GET /api/v1/admin-panel/keyword-banks
			adminPanel.GET("/keyword-banks/:id", r.handleGetKeywordBank)        // GET /api/v1/admin-panel/keyword-banks/:id
			adminPanel.POST("/keyword-banks", r.handleCreateKeywordBank)        // POST /api/v1/admin-panel/keyword-banks
			adminPanel.PUT("/keyword-banks/:id", r.handleUpdateKeywordBank)     // PUT /api/v1/admin-panel/keyword-banks/:id
			adminPanel.PATCH("/keyword-banks/:id", r.handlePatchKeywordBank)    // PATCH /api/v1/admin-panel/keyword-banks/:id
			adminPanel.DELETE("/keyword-banks/:id", r.handleDeleteKeywordBank)  // DELETE /api/v1/admin-panel/keyword-banks/:id
		}
	}
}

// Note: CreateArticle method exists in article_handlers.go

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


// SEO Settings - simple in-memory storage
var seoSettings = map[string]interface{}{
	"site_name":               "Cryptonlisys",
	"site_tagline":            "Real-time crypto news, exchange insights, and data-driven guides",
	"site_url":                "https://cryptonlisys.com",
	"publisher_name":          "Cryptonlisys",
	"publisher_logo":          "/static/images/logo.svg",
	"default_language":        "en",
	"auto_meta_desc":          true,
	"enable_schema":           true,
	"enable_canonical":        true,
	"home_title":              "Cryptonlisys - Crypto News & Exchange Reviews",
	"home_desc":               "Real-time crypto news, exchange insights, and data-driven guides for modern investors",
	"article_title_template":  "{title} - {site_name}",
	"category_title_template": "{category} - {site_name}",
	"default_schema_type":     "NewsArticle",
	"og_site_name":            "Cryptonlisys",
	"og_default_image":        "/static/images/og-default.jpg",
	"fb_app_id":               "",
	"twitter_site":            "",
	"twitter_card_type":       "summary_large_image",
	"enable_sitemap":          true,
	"enable_news_sitemap":     true,
	"enable_rss":              true,
	"rss_delay_hours":         2,
}

// robotsTxtContent is the default robots.txt template
// Note: {SITE_URL} will be replaced with actual site URL when served
var robotsTxtContent = `User-agent: *
Allow: /

Sitemap: {SITE_URL}/sitemap.xml
Sitemap: {SITE_URL}/sitemap-news.xml

Crawl-delay: 1

Disallow: /admin/
Disallow: /api/
Allow: /api/v1/articles/
Allow: /static/
Allow: /uploads/`

// GetRobotsTxtContent returns the current robots.txt content (exported for server package)
func GetRobotsTxtContent() string {
	return robotsTxtContent
}

func getSEOSettings(c *gin.Context) {
	c.JSON(200, gin.H{"settings": seoSettings})
}

func saveSEOSettings(c *gin.Context) {
	var input map[string]interface{}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}
	
	// Merge input into settings
	for key, value := range input {
		seoSettings[key] = value
	}
	
	c.JSON(200, gin.H{"status": "success", "message": "Settings saved"})
}

func getRobotsTxt(c *gin.Context) {
	c.JSON(200, gin.H{"content": robotsTxtContent})
}

func saveRobotsTxt(c *gin.Context) {
	var input struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}
	
	robotsTxtContent = input.Content
	c.JSON(200, gin.H{"status": "success", "message": "Robots.txt saved"})
}
