package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/api"
	"high-performance-news-website/internal/auth"
	"high-performance-news-website/internal/config"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/repositories"
	"high-performance-news-website/internal/services"
	"high-performance-news-website/internal/templates"
	"high-performance-news-website/pkg/cache"
	"high-performance-news-website/pkg/database"
)

// CacheServiceAdapter adapts cache.CacheService to services.CacheService
type CacheServiceAdapter struct {
	cache cache.CacheService
}

func (c *CacheServiceAdapter) Set(key string, value []byte, ttl time.Duration) error {
	return c.cache.Set(context.Background(), key, value, ttl)
}

func (c *CacheServiceAdapter) Get(key string) ([]byte, error) {
	return c.cache.Get(context.Background(), key)
}

func (c *CacheServiceAdapter) Delete(key string) error {
	return c.cache.Delete(context.Background(), key)
}

func (c *CacheServiceAdapter) DeletePattern(pattern string) error {
	// This would need to be implemented based on the cache implementation
	// For now, return nil
	return nil
}

type Server struct {
	config                  *config.Config
	router                  *gin.Engine
	cache                   cache.CacheService
	db                      *database.DB
	apiRouter               *api.Router
	templateEngine          *templates.TemplateEngine
	rssHandlers             *api.RSSHandlers
	googleNewsHandlers      *api.GoogleNewsHandlers
	metricsService          *services.MetricsService
	healthService           *services.HealthService
	alertingService         *services.AlertingService
	monitoringConfig        *config.MonitoringConfig
	analyticsService        *services.AnalyticsService
	advertisementService    *services.AdvertisementService
	pushNotificationService *services.PushNotificationService
	widgetService           *services.WidgetService
	themeService            *services.ThemeService
	cdnService              services.CDNServiceInterface
	articleService          *services.ArticleService
	categoryService         *services.CategoryService
	tagService              *services.TagService
	authService             *auth.AuthService
	mediaService            *services.MediaService
	staticGenerator         *services.StaticGenerator
	enterpriseSearchService *services.EnterpriseSearchService
}

// requireAuth middleware checks authentication for admin routes
func (s *Server) requireAuth(c *gin.Context) {
	// Check for auth token in cookie or header
	token := c.GetHeader("Authorization")
	if token == "" {
		// Check for session cookie
		cookie, err := c.Cookie("auth_token")
		if err != nil || cookie == "" {
			s.redirectToLogin(c, "No authentication token found")
			return
		}
		token = "Bearer " + cookie
	}

	// Remove "Bearer " prefix if present
	if strings.HasPrefix(token, "Bearer ") {
		token = strings.TrimPrefix(token, "Bearer ")
	}

	// Validate the token using the auth service
	if s.authService == nil {
		s.redirectToLogin(c, "Authentication service not available")
		return
	}

	claims, err := s.authService.ValidateAccessToken(token)
	if err != nil {
		s.redirectToLogin(c, "Invalid or expired token")
		return
	}

	// Check if user has admin or editor role
	if claims.Role != "admin" && claims.Role != "editor" {
		s.redirectToLogin(c, "Insufficient permissions")
		return
	}

	// Store user info in context for use by handlers
	c.Set("user_id", claims.UserID)
	c.Set("user_role", claims.Role)
	c.Set("username", claims.Username)

	c.Next()
}

// Helper function to handle redirects consistently
func (s *Server) redirectToLogin(c *gin.Context, reason string) {
	// Log the reason for debugging
	log.Printf("Authentication failed: %s", reason)

	// Check if it's an AJAX request
	if c.GetHeader("X-Requested-With") == "XMLHttpRequest" ||
		c.GetHeader("Accept") == "application/json" ||
		strings.Contains(c.GetHeader("Accept"), "application/json") {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":    "Authentication required",
			"redirect": "/admin/login",
			"reason":   reason,
		})
	} else {
		// For browser requests, redirect immediately
		c.Redirect(http.StatusFound, "/admin/login")
	}
	c.Abort()
}

func New(cfg *config.Config) (*Server, error) {
	gin.SetMode(cfg.Server.Mode)
	router := gin.New()

	// Check if we're in development mode
	if cfg.App.DevMode {
		log.Println("Starting in development mode with mock services...")
		return newDevelopmentServer(cfg, router)
	}

	// Try to initialize cache and database
	var cacheClient cache.CacheService
	var db *database.DB
	var useMockServices bool

	dragonflyCache, err := cache.NewDragonflyClient(&cfg.Cache)
	if err != nil {
		log.Printf("Failed to initialize cache: %v", err)
		log.Println("RSS feeds will use mock data")
		useMockServices = true
	} else {
		cacheClient = cache.CacheService(dragonflyCache)
	}

	if !useMockServices {
		db, err = database.NewConnection(&cfg.Database)
		if err != nil {
			log.Printf("Failed to initialize database: %v", err)
			log.Println("RSS feeds will use mock data")
			useMockServices = true
		}
	}

	// Initialize common variables
	baseURL := cfg.App.BaseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://localhost:%d", cfg.Server.Port) // Fallback for development
	}
	var seoService *services.SEOService
	var breadcrumbService *services.BreadcrumbService
	var rssHandlers *api.RSSHandlers
	var googleNewsHandlers *api.GoogleNewsHandlers
	var apiRouter *api.Router

	// Initialize RSS service - use mock if database not available
	if useMockServices {
		log.Println("Using mock RSS and Google News services")
		mockRSSService := services.NewMockRSSService()
		rssHandlers = api.NewRSSHandlers(mockRSSService)

		mockGoogleNewsSitemapService := services.NewMockGoogleNewsSitemapService()
		googleNewsHandlers = api.NewGoogleNewsHandlers(mockGoogleNewsSitemapService)
	}

	// Declare services outside the block
	var authService *auth.AuthService
	var userService *services.UserService
	var configService *services.ConfigService
	var metricsService *services.MetricsService
	var healthService *services.HealthService
	var alertingService *services.AlertingService
	var monitoringConfig *config.MonitoringConfig
	var categoryService *services.CategoryService
	var tagService *services.TagService
	var widgetService *services.WidgetService
	var themeService *services.ThemeService
	var articleService *services.ArticleService
	var contentIngestionService *services.ContentIngestionService
	var searchService services.SearchServiceInterface
	var enterpriseSearchService *services.EnterpriseSearchService
	var commentHandlers *api.CommentHandlers
	var imageHandlers *api.ImageHandlers
	var analyticsService *services.AnalyticsService
	var advertisementService *services.AdvertisementService
	var pushNotificationService *services.PushNotificationService
	var cdnService services.CDNServiceInterface
	var mediaService *services.MediaService
	var staticGenerator *services.StaticGenerator

	// Initialize services - skip database-dependent services if using mock
	if !useMockServices {
		seoService = services.NewSEOService(baseURL, cfg.App.Name, "en")
		breadcrumbService = services.NewBreadcrumbService(baseURL, cfg.App.Name)
		// Initialize full services when database is available
		authService = auth.NewAuthService(cfg.JWT.Secret, cfg.JWT.Secret)
		userService = services.NewUserService(db.DB, authService)

		if err := createDemoUser(userService); err != nil {
			log.Printf("Warning: Failed to create demo user: %v", err)
		}

		configService = services.NewConfigService(db.DB, cacheClient)
		configService.LoadDefaults()

		// Initialize monitoring configuration
		monitoringConfig = config.LoadMonitoringConfig()
		if err := monitoringConfig.Validate(); err != nil {
			log.Printf("Warning: Invalid monitoring config: %v", err)
		}

		// Initialize metrics service with proper parameters
		metricsService = services.NewMetricsService(db.DB, cacheClient, monitoringConfig)

		// Initialize health service
		healthService = services.NewHealthService(db.DB, cacheClient, monitoringConfig, metricsService)

		// Initialize alerting service
		var emailService services.EmailService // This would be initialized based on config
		alertingService = services.NewAlertingService(monitoringConfig, emailService)

		categoryService = services.NewCategoryService(db.DB)
		tagService = services.NewTagService(db.DB)

		articleRepo := repositories.NewArticleRepository(db, cacheClient, cfg.Server.StaticPath)

		// Initialize widget and theme services
		widgetRepo := repositories.NewWidgetRepository(db.DB)
		themeRepo := repositories.NewThemeRepository(db.DB)
		categoryRepo := repositories.NewCategoryRepository(db.DB)
		tagRepo := repositories.NewTagRepository(db.DB)

		// Create cache adapter for services that expect the old interface
		cacheAdapter := &CacheServiceAdapter{cache: cacheClient}
		widgetService = services.NewWidgetService(widgetRepo, articleRepo, categoryRepo, tagRepo, cacheAdapter)
		themeService = services.NewThemeService(themeRepo, cacheAdapter, "web/templates")

		// Initialize additional admin services
		analyticsRepo := repositories.NewAnalyticsRepository(db.DB)
		analyticsService = services.NewAnalyticsService(analyticsRepo)

		advertisementRepo := repositories.NewAdvertisementRepository(db.DB)
		advertisementService = services.NewAdvertisementService(advertisementRepo, cacheAdapter, baseURL)

		pushNotificationRepo := repositories.NewPushNotificationRepository(db.DB)
		pushNotificationService = services.NewPushNotificationService(pushNotificationRepo, "", "", "")

		// Initialize CDN service
		cdnConfig := config.LoadCDNConfig()
		if cdnConfig.Enabled {
			cdnService = services.NewCloudflareCDNService(cdnConfig.ToModel())
		}

		redisClient := dragonflyCache.GetRedisClient()

		if cfg.Search.Enabled {
			searchIndexer := services.NewSearchIndexer(
				cfg.Search.MeiliSearchURL,
				cfg.Search.MeiliSearchAPIKey,
				cfg.Search.IndexName,
				nil, // fallbackDB not needed - EnterpriseSearchService handles fallback directly
			)
			// Create EnterpriseSearchService with all enterprise features
			enterpriseSearchService = services.NewEnterpriseSearchService(searchIndexer, redisClient, db.DB)
			// Start background workers (dead letter queue processor, metrics updater, reconciliation)
			if err := enterpriseSearchService.Start(); err != nil {
				log.Printf("Warning: Failed to start enterprise search service workers: %v", err)
			}
			// Wrap for interface compatibility
			searchService = services.NewEnterpriseSearchServiceWrapper(enterpriseSearchService)
			log.Println("Enterprise search service initialized with MeiliSearch + PostgreSQL fallback")
		} else {
			// Even without MeiliSearch, use EnterpriseSearchService for PostgreSQL-only mode
			enterpriseSearchService = services.NewEnterpriseSearchService(nil, redisClient, db.DB)
			if err := enterpriseSearchService.Start(); err != nil {
				log.Printf("Warning: Failed to start enterprise search service workers: %v", err)
			}
			searchService = services.NewEnterpriseSearchServiceWrapper(enterpriseSearchService)
			log.Println("Enterprise search service initialized with PostgreSQL fallback only")
		}

		articleService = services.NewArticleService(db, articleRepo, nil)

		// Initialize comment handlers with proper dependencies
		commentRepo := repositories.NewCommentRepository(db.DB)
		userRepo := repositories.NewUserRepository(db.DB)

		// Initialize content ingestion service with proper dependencies (after userRepo is defined)
		ingestionRepo := repositories.NewContentIngestionRepository(db)
		contentIngestionService = services.NewContentIngestionService(
			ingestionRepo,
			articleRepo,
			userRepo,
			categoryRepo,
			tagRepo,
		)
		rateLimiter := api.NewRateLimiter()
		commentHandlers = api.NewCommentHandlers(commentRepo, userRepo, rateLimiter)
		//imageHandlers = &api.ImageHandlers{}
		// Create image processor for handling uploads and variants
		imageProcessorConfig := services.ImageProcessorConfig{
			StorageBasePath: "uploads",
			MaxWorkers:      4,
			QueueSize:       100,
		}
		imageProcessor := services.NewImageProcessor(imageProcessorConfig)

		// Create media service for database operations
		fmt.Printf("DEBUG: Creating MediaService with database connection\n")
		mediaService = services.NewMediaService(db.DB)
		fmt.Printf("DEBUG: MediaService created successfully, is nil: %v\n", mediaService == nil)

		// Initialize static generator for automatic static file generation
		// Uses the same templates as dynamic pages with adapted data structure
		// NOTE: mediaService must be created before this to enable responsive image generation
		staticGeneratorConfig := services.StaticGeneratorConfig{
			OutputPath:   "static-html",
			TemplatesDir: "web/templates",
			BaseURL:      baseURL,
			SiteName:     cfg.App.Name,
		}
		var sgErr error
		staticGenerator, sgErr = services.NewStaticGenerator(staticGeneratorConfig, cacheClient, articleRepo, categoryRepo, tagRepo, mediaService)
		if sgErr != nil {
			log.Printf("Warning: Failed to initialize static generator: %v (static file generation disabled)", sgErr)
		} else {
			articleService.SetStaticGenerator(staticGenerator)
			log.Println("Static generator initialized with MediaService - responsive image generation enabled")
		}

		// Wire MediaService to ArticleService for image cleanup on article deletion
		articleService.SetMediaService(mediaService)
		log.Println("MediaService wired to ArticleService - image cleanup on article deletion enabled")

		// Wire SearchService to ArticleService for real-time search indexing
		if searchService != nil {
			articleService.SetSearchService(searchService)
			log.Println("SearchService wired to ArticleService - real-time search indexing enabled")
		}

		// Create image handlers with proper configuration
		fmt.Printf("DEBUG: Creating ImageHandlers with MediaService\n")
		imageHandlers = api.NewImageHandlers(
			imageProcessor,
			mediaService,
			"uploads",
			10*1024*1024, // 10MB max file size
		)
		fmt.Printf("DEBUG: ImageHandlers created successfully\n")

		// Initialize real RSS services now that articleRepo is available
		if rssHandlers == nil {
			categoryRepo := repositories.NewCategoryRepository(db.DB)
			tagRepo := repositories.NewTagRepository(db.DB)
			rssService := services.NewRSSService(
				articleRepo,
				categoryRepo,
				tagRepo,
				cacheClient,
				baseURL,
				cfg.App.Name,
				"High-performance multilingual news website",
			)
			rssHandlers = api.NewRSSHandlers(rssService)

			googleNewsSitemapService := services.NewGoogleNewsSitemapService(
				articleRepo,
				cacheClient,
				baseURL,
				cfg.App.Name,
			)
			googleNewsHandlers = api.NewGoogleNewsHandlers(googleNewsSitemapService)
		}

		apiRouter = api.NewRouter(
			db.DB,
			userService,
			articleService,
			searchService,
			contentIngestionService,
			authService,
			cacheClient,
			commentHandlers,
			imageHandlers,
			configService,
			metricsService,
			healthService,
			alertingService,
			categoryService,
			tagService,
			widgetService,
			themeService,
			cdnService,
		)
	} else {
		// Skip API router when using mock services
		log.Println("Skipping API router initialization - using mock services only")
		apiRouter = nil
	}

	if !useMockServices {
		seoService = services.NewSEOService(baseURL, cfg.App.Name, "en")
		breadcrumbService = services.NewBreadcrumbService(baseURL, cfg.App.Name)
	}

	// Initialize RSS service - use mock if database not available
	if useMockServices {
		log.Println("Using mock RSS and Google News services")
		mockRSSService := services.NewMockRSSService()
		rssHandlers = api.NewRSSHandlers(mockRSSService)

		mockGoogleNewsSitemapService := services.NewMockGoogleNewsSitemapService()
		googleNewsHandlers = api.NewGoogleNewsHandlers(mockGoogleNewsSitemapService)
	}

	templateEngine := templates.NewTemplateEngine(cfg.Server.Mode == "debug")
	if !useMockServices {
		templateEngine.SetSEOServices(seoService, breadcrumbService)
	}
	// Always try to load templates, even with mock services
	if err := templateEngine.LoadTemplates("web/templates"); err != nil {
		log.Printf("Warning: Failed to load templates, using fallback rendering: %v", err)
	}

	// Initialize monitoring services variables for all cases
	var finalMetricsService *services.MetricsService
	var finalHealthService *services.HealthService
	var finalAlertingService *services.AlertingService
	var finalMonitoringConfig *config.MonitoringConfig

	if !useMockServices {
		finalMetricsService = metricsService
		finalHealthService = healthService
		finalAlertingService = alertingService
		finalMonitoringConfig = monitoringConfig
	}

	// Initialize final service variables for all cases
	var finalAnalyticsService *services.AnalyticsService
	var finalAdvertisementService *services.AdvertisementService
	var finalPushNotificationService *services.PushNotificationService
	var finalWidgetService *services.WidgetService
	var finalThemeService *services.ThemeService
	var finalCDNService services.CDNServiceInterface
	var finalMediaService *services.MediaService

	if !useMockServices {
		finalAnalyticsService = analyticsService
		finalAdvertisementService = advertisementService
		finalPushNotificationService = pushNotificationService
		finalWidgetService = widgetService
		finalThemeService = themeService
		finalCDNService = cdnService
		finalMediaService = mediaService
	}

	return &Server{
		config:                  cfg,
		router:                  router,
		cache:                   cacheClient,
		db:                      db,
		apiRouter:               apiRouter,
		templateEngine:          templateEngine,
		rssHandlers:             rssHandlers,
		googleNewsHandlers:      googleNewsHandlers,
		metricsService:          finalMetricsService,
		healthService:           finalHealthService,
		alertingService:         finalAlertingService,
		monitoringConfig:        finalMonitoringConfig,
		analyticsService:        finalAnalyticsService,
		advertisementService:    finalAdvertisementService,
		pushNotificationService: finalPushNotificationService,
		widgetService:           finalWidgetService,
		themeService:            finalThemeService,
		cdnService:              finalCDNService,
		articleService:          articleService,
		categoryService:         categoryService,
		tagService:              tagService,
		authService:             authService,
		mediaService:            finalMediaService,
		staticGenerator:         staticGenerator,
		enterpriseSearchService: enterpriseSearchService,
	}, nil
}

func (s *Server) setupRoutes() {
	// Only setup API routes if not in development mode
	if s.apiRouter != nil {
		s.apiRouter.SetupRoutes(s.router)
	}

	s.setupRSSRoutes()
	s.setupGoogleNewsRoutes()

	// Health route is handled by the monitoring handler

	s.router.Static("/static", "./web/static")
	s.router.Static("/uploads", "./uploads")

	// Root redirect to default language
	s.router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/en/")
	})

	s.router.GET("/offline", s.handleOfflinePage)

	// Setup admin frontend routes FIRST (before catch-all routes)
	s.setupAdminFrontendRoutes()

	// Setup multilingual frontend routes
	s.setupMultilingualRoutes()

	// Frontend website routes - set up AFTER admin routes
	// This is important because setupProductionFrontendRoutes has a catch-all /:slug route
	s.setupProductionFrontendRoutes()
}

func (s *Server) Start() error {
	s.setupRoutes()

	// Start monitoring system if available
	if s.metricsService != nil && s.monitoringConfig != nil {
		log.Println("Starting monitoring system...")
		ctx := context.Background()
		go s.metricsService.StartMonitoring(ctx)
		log.Println("Monitoring system started successfully")
	} else {
		log.Println("Monitoring system not available (development mode)")
	}

	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)

	srv := &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	go func() {
		log.Printf("Server starting on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	// Stop monitoring system
	if s.metricsService != nil {
		log.Println("Stopping monitoring system...")
		s.metricsService.StopMonitoring()
	}

	// Stop enterprise search service background workers
	if s.enterpriseSearchService != nil {
		log.Println("Stopping enterprise search service...")
		if err := s.enterpriseSearchService.Stop(); err != nil {
			log.Printf("Error stopping enterprise search service: %v", err)
		}
	}

	if s.cache != nil {
		if err := s.cache.Close(); err != nil {
			log.Printf("Error closing cache connection: %v", err)
		}
	}

	if s.db != nil {
		if err := s.db.Close(); err != nil {
			log.Printf("Error closing database connection: %v", err)
		}
	}

	log.Println("Server exited")
	return nil
}

func createDemoUser(userService *services.UserService) error {
	// Get admin credentials from environment variables, with fallback to demo values
	adminEmail := os.Getenv("NEWS_ADMIN_EMAIL")
	if adminEmail == "" {
		adminEmail = "admin@example.com"
	}

	adminPassword := os.Getenv("NEWS_ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "Admin123!"
	}

	_, err := userService.GetByEmail(adminEmail)
	if err == nil {
		return nil
	}

	createReq := &services.CreateUserRequest{
		Username:  "admin",
		Email:     adminEmail,
		Password:  adminPassword,
		Role:      models.RoleAdmin,
		FirstName: "Admin",
		LastName:  "User",
		Bio:       "Administrator account",
	}

	_, err = userService.Create(createReq, nil)
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	log.Printf("Admin user created successfully (%s)", adminEmail)
	return nil
}

func (s *Server) handleOfflinePage(c *gin.Context) {
	data := s.createBaseTemplateData(c)
	data["Title"] = "Offline"
	data["PageType"] = "offline"

	if s.templateEngine != nil {
		if html, err := s.templateEngine.Render("offline", data); err == nil {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, html)
			return
		}
	}

	// Fallback to simple HTML
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, `<!DOCTYPE html><html><head><title>Offline</title></head><body><h1>You're Offline</h1><p>Please check your internet connection.</p></body></html>`)
}

func (s *Server) handleHomepage(c *gin.Context) {
	// Use development homepage if in dev mode
	if s.config.App.DevMode {
		s.handleDevHomepage(c)
		return
	}

	// Create proper template data for homepage
	data := s.createBaseTemplateData(c)
	data["Title"] = s.config.App.Name
	data["PageType"] = "homepage"

	// Get real articles from database
	if s.articleService != nil {
		// Get latest published articles for homepage
		filters := services.ArticleFilters{Status: "published"}
		articles, _, err := s.articleService.List(c.Request.Context(), 10, 0, filters, "published_at", "DESC")
		if err == nil {
			// Convert to template format
			articleData := make([]gin.H, len(articles))
			for i, article := range articles {
				// Get category name for this article
				categoryName := "General"
				if article.CategoryID > 0 {
					if categoryData, err := s.getCategoryByID(article.CategoryID); err == nil {
						if name, ok := categoryData["Name"].(string); ok {
							categoryName = name
						}
						log.Printf("Category data for article %d (category_id %d): %+v, name: %s", article.ID, article.CategoryID, categoryData, categoryName)
					} else {
						log.Printf("Error getting category for article %d (category_id %d): %v", article.ID, article.CategoryID, err)
						// Fallback category mapping based on database data
						switch article.CategoryID {
						case 1:
							categoryName = "News"
						case 2:
							categoryName = "Technology"
						case 3:
							categoryName = "Sports"
						case 5:
							categoryName = "Health"
						case 22:
							categoryName = "Test1"
						case 23:
							categoryName = "Test 2"
						case 24:
							categoryName = "Pooya"
						case 25:
							categoryName = "Test3"
						default:
							categoryName = "General"
						}
					}
				}

				// Handle featured image
				featuredImage := ""
				if article.FeaturedImage != "" {
					featuredImage = article.FeaturedImage
				}

				// Build responsive image data if article has a featured image ID
				var imageData *services.ResponsiveImageData
				if article.FeaturedImageID != nil && *article.FeaturedImageID > 0 && s.mediaService != nil {
					imageData = s.buildResponsiveImageData(*article.FeaturedImageID, article.Title)
				}

				articleData[i] = gin.H{
					"ID":            article.ID,
					"Title":         article.Title,
					"Slug":          article.Slug,
					"Excerpt":       article.Excerpt,
					"Author":        "Author", // Will be populated by repository joins
					"TimeAgo":       formatTimeAgo(article.PublishedAt),
					"ViewCount":     article.ViewCount,
					"Views":         article.ViewCount,
					"Category":      categoryName,
					"FeaturedImage": featuredImage,
					"ImageData":     imageData,
				}
			}
			data["Articles"] = articleData
		} else {
			log.Printf("Error fetching articles for homepage: %v", err)
			data["Articles"] = []gin.H{}
		}
	} else {
		data["Articles"] = []gin.H{}
	}

	// Get real categories from database
	if s.categoryService != nil {
		categories, err := s.categoryService.GetAll()
		if err == nil {
			// Convert to template format
			categoryData := make([]gin.H, len(categories))
			for i, cat := range categories {
				categoryData[i] = gin.H{
					"ID":           cat.ID,
					"Name":         cat.Name,
					"Slug":         cat.Slug,
					"Description":  cat.Description,
					"ImageURL":     cat.GetImageURL(),
					"ImageAltText": cat.GetImageAltText(),
					"Count":        s.getCategoryArticleCount(cat.ID),
				}
			}
			data["Categories"] = categoryData
		} else {
			log.Printf("Error fetching categories for homepage: %v", err)
			data["Categories"] = []gin.H{}
		}
	} else {
		data["Categories"] = []gin.H{}
	}

	// Get real trending articles from database (most viewed in last 24 hours)
	if s.articleService != nil {
		trendingArticles, err := s.articleService.GetTrending(c.Request.Context(), 5, 24)
		if err == nil && len(trendingArticles) > 0 {
			trendingData := make([]gin.H, len(trendingArticles))
			for i, article := range trendingArticles {
				trendingData[i] = gin.H{
					"ID":        article.ID,
					"Title":     article.Title,
					"Slug":      article.Slug,
					"ViewCount": article.ViewCount,
					"TimeAgo":   formatTimeAgo(article.PublishedAt),
				}
			}
			data["TrendingArticles"] = trendingData
		} else {
			// Fallback: use most viewed articles if no trending data
			filters := services.ArticleFilters{Status: "published"}
			popularArticles, _, err := s.articleService.List(c.Request.Context(), 5, 0, filters, "view_count", "DESC")
			if err == nil && len(popularArticles) > 0 {
				trendingData := make([]gin.H, len(popularArticles))
				for i, article := range popularArticles {
					trendingData[i] = gin.H{
						"ID":        article.ID,
						"Title":     article.Title,
						"Slug":      article.Slug,
						"ViewCount": article.ViewCount,
						"TimeAgo":   formatTimeAgo(article.PublishedAt),
					}
				}
				data["TrendingArticles"] = trendingData
			} else {
				data["TrendingArticles"] = []gin.H{}
			}
		}
	} else {
		data["TrendingArticles"] = []gin.H{}
	}

	// Video articles - filter by video tag or category if available
	data["VideoArticles"] = []gin.H{}

	// Opinion/editorial articles - would filter by opinion category
	data["OpinionArticles"] = []gin.H{}

	// Get real popular tags from database
	if s.tagService != nil {
		tags, err := s.tagService.GetAll()
		if err == nil {
			// Convert to template format with consistent theme color
			type TagWithCount struct {
				Name  string
				Slug  string
				Count int
			}

			tagList := make([]TagWithCount, 0)
			for _, tag := range tags {
				// Get article count for this tag
				articleCount := s.getTagArticleCount(tag.ID)

				// Only include tags that have articles
				if articleCount > 0 {
					tagList = append(tagList, TagWithCount{
						Name:  tag.Name,
						Slug:  tag.Slug,
						Count: articleCount,
					})
				}
			}

			// Sort by article count (most popular first)
			for i := 0; i < len(tagList)-1; i++ {
				for j := i + 1; j < len(tagList); j++ {
					if tagList[j].Count > tagList[i].Count {
						tagList[i], tagList[j] = tagList[j], tagList[i]
					}
				}
			}

			// Limit to 8 most popular tags
			if len(tagList) > 8 {
				tagList = tagList[:8]
			}

			// Convert to gin.H format
			tagData := make([]gin.H, len(tagList))
			for i, tag := range tagList {
				tagData[i] = gin.H{
					"Name":  tag.Name,
					"Slug":  tag.Slug,
					"Count": tag.Count,
				}
			}

			data["PopularTags"] = tagData
		} else {
			log.Printf("Error fetching tags for homepage: %v", err)
			data["PopularTags"] = []gin.H{}
		}
	} else {
		data["PopularTags"] = []gin.H{}
	}

	if s.templateEngine != nil {
		if html, err := s.templateEngine.Render("homepage", data); err == nil {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, html)
			return
		}
	}

	// Fallback to simple response
	c.String(http.StatusOK, "Homepage (Template not available)")
}

// setupRSSRoutes sets up RSS feed routes
func (s *Server) setupRSSRoutes() {
	// Main RSS feed
	s.router.GET("/rss", s.rssHandlers.HandleMainRSSFeed)
	s.router.GET("/rss.xml", s.rssHandlers.HandleMainRSSFeed)
	s.router.GET("/feed", s.rssHandlers.HandleMainRSSFeed)
	s.router.GET("/feed.xml", s.rssHandlers.HandleMainRSSFeed)

	// Category RSS feeds
	s.router.GET("/rss/category/:slug", s.rssHandlers.HandleCategoryRSSFeed)

	// Tag RSS feeds
	s.router.GET("/rss/tag/:slug", s.rssHandlers.HandleTagRSSFeed)

	// Google News RSS feed
	s.router.GET("/rss/googlenews", s.rssHandlers.HandleGoogleNewsRSSFeed)
	s.router.GET("/rss/googlenews.xml", s.rssHandlers.HandleGoogleNewsRSSFeed)
}

// setupGoogleNewsRoutes sets up Google News sitemap routes
func (s *Server) setupGoogleNewsRoutes() {
	// Google News sitemap
	s.router.GET("/sitemap-news.xml", s.googleNewsHandlers.HandleGoogleNewsSitemap)
	s.router.GET("/sitemap-news-:page.xml", s.googleNewsHandlers.HandleGoogleNewsSitemap)

	// Google News sitemap index
	s.router.GET("/sitemap-news-index.xml", s.googleNewsHandlers.HandleGoogleNewsSitemapIndex)

	// Main sitemap routes
	s.router.GET("/sitemap.xml", s.handleSitemapIndex)
	s.router.GET("/sitemap-main.xml", s.handleMainSitemap)
	s.router.GET("/sitemap-articles.xml", s.handleArticlesSitemap)
	s.router.GET("/sitemap-categories.xml", s.handleCategoriesSitemap)
	s.router.GET("/sitemap-tags.xml", s.handleTagsSitemap)
	s.router.GET("/robots.txt", s.handleRobotsTxt)
}

// handleSitemapIndex generates the main sitemap index
func (s *Server) handleSitemapIndex(c *gin.Context) {
	baseURL := s.config.App.BaseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://%s", c.Request.Host)
	}

	now := time.Now().Format(time.RFC3339)

	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <sitemap>
    <loc>%s/sitemap-main.xml</loc>
    <lastmod>%s</lastmod>
  </sitemap>
  <sitemap>
    <loc>%s/sitemap-articles.xml</loc>
    <lastmod>%s</lastmod>
  </sitemap>
  <sitemap>
    <loc>%s/sitemap-categories.xml</loc>
    <lastmod>%s</lastmod>
  </sitemap>
  <sitemap>
    <loc>%s/sitemap-tags.xml</loc>
    <lastmod>%s</lastmod>
  </sitemap>
  <sitemap>
    <loc>%s/sitemap-news.xml</loc>
    <lastmod>%s</lastmod>
  </sitemap>
</sitemapindex>`, baseURL, now, baseURL, now, baseURL, now, baseURL, now, baseURL, now)

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.String(http.StatusOK, xml)
}

// handleMainSitemap generates sitemap for static pages
func (s *Server) handleMainSitemap(c *gin.Context) {
	baseURL := s.config.App.BaseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://%s", c.Request.Host)
	}

	now := time.Now().Format(time.RFC3339)

	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>%s/</loc>
    <lastmod>%s</lastmod>
    <changefreq>daily</changefreq>
    <priority>1.0</priority>
  </url>
  <url>
    <loc>%s/latest</loc>
    <lastmod>%s</lastmod>
    <changefreq>hourly</changefreq>
    <priority>0.9</priority>
  </url>
  <url>
    <loc>%s/trending</loc>
    <lastmod>%s</lastmod>
    <changefreq>hourly</changefreq>
    <priority>0.9</priority>
  </url>
</urlset>`, baseURL, now, baseURL, now, baseURL, now)

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.String(http.StatusOK, xml)
}

// handleArticlesSitemap generates sitemap for all published articles
func (s *Server) handleArticlesSitemap(c *gin.Context) {
	baseURL := s.config.App.BaseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://%s", c.Request.Host)
	}

	var urls []string

	// Get all published articles from database
	if s.db != nil {
		query := `
			SELECT slug, updated_at 
			FROM articles 
			WHERE status = 'published' AND published_at IS NOT NULL
			ORDER BY published_at DESC
			LIMIT 50000`

		rows, err := s.db.DB.Query(query)
		if err != nil {
			log.Printf("Error fetching articles for sitemap: %v", err)
		} else {
			defer rows.Close()
			for rows.Next() {
				var slug string
				var updatedAt time.Time
				if err := rows.Scan(&slug, &updatedAt); err == nil {
					urls = append(urls, fmt.Sprintf(`  <url>
    <loc>%s/article/%s</loc>
    <lastmod>%s</lastmod>
    <changefreq>weekly</changefreq>
    <priority>0.8</priority>
  </url>`, baseURL, slug, updatedAt.Format(time.RFC3339)))
				}
			}
		}
	}

	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
%s
</urlset>`, strings.Join(urls, "\n"))

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.String(http.StatusOK, xml)
}

// handleCategoriesSitemap generates sitemap for all categories
func (s *Server) handleCategoriesSitemap(c *gin.Context) {
	baseURL := s.config.App.BaseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://%s", c.Request.Host)
	}

	var urls []string

	// Get all categories from database
	if s.db != nil {
		query := `SELECT slug, updated_at FROM categories ORDER BY name`

		rows, err := s.db.DB.Query(query)
		if err != nil {
			log.Printf("Error fetching categories for sitemap: %v", err)
		} else {
			defer rows.Close()
			for rows.Next() {
				var slug string
				var updatedAt time.Time
				if err := rows.Scan(&slug, &updatedAt); err == nil {
					urls = append(urls, fmt.Sprintf(`  <url>
    <loc>%s/category/%s</loc>
    <lastmod>%s</lastmod>
    <changefreq>weekly</changefreq>
    <priority>0.7</priority>
  </url>`, baseURL, slug, updatedAt.Format(time.RFC3339)))
				}
			}
		}
	}

	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
%s
</urlset>`, strings.Join(urls, "\n"))

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.String(http.StatusOK, xml)
}

// handleTagsSitemap generates sitemap for all tags
func (s *Server) handleTagsSitemap(c *gin.Context) {
	baseURL := s.config.App.BaseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://%s", c.Request.Host)
	}

	var urls []string

	// Get all tags from database
	if s.db != nil {
		query := `SELECT slug, updated_at FROM tags ORDER BY name`

		rows, err := s.db.DB.Query(query)
		if err != nil {
			log.Printf("Error fetching tags for sitemap: %v", err)
		} else {
			defer rows.Close()
			for rows.Next() {
				var slug string
				var updatedAt time.Time
				if err := rows.Scan(&slug, &updatedAt); err == nil {
					urls = append(urls, fmt.Sprintf(`  <url>
    <loc>%s/tag/%s</loc>
    <lastmod>%s</lastmod>
    <changefreq>weekly</changefreq>
    <priority>0.6</priority>
  </url>`, baseURL, slug, updatedAt.Format(time.RFC3339)))
				}
			}
		}
	}

	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
%s
</urlset>`, strings.Join(urls, "\n"))

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.String(http.StatusOK, xml)
}

// handleLanguageSitemap generates sitemap for a specific language with hreflang support
// IMPORTANT: hreflang tags are ONLY generated for languages that have actual translations
// This follows SEO best practices - missing translation ≠ SEO penalty, but broken hreflang = problem
func (s *Server) handleLanguageSitemap(c *gin.Context) {
	// Extract language from URL (e.g., /sitemap-en.xml -> en)
	path := c.Request.URL.Path
	lang := strings.TrimPrefix(path, "/sitemap-")
	lang = strings.TrimSuffix(lang, ".xml")

	baseURL := s.config.App.BaseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://%s", c.Request.Host)
	}

	allLanguages := []string{"en", "de", "fr", "es", "ar"}

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"
        xmlns:xhtml="http://www.w3.org/1999/xhtml">`

	// Add homepage - homepage exists in all languages
	xml += fmt.Sprintf(`
  <url>
    <loc>%s/%s/</loc>
    <lastmod>%s</lastmod>
    <changefreq>daily</changefreq>
    <priority>1.0</priority>`, baseURL, lang, time.Now().Format("2006-01-02"))

	// Add hreflang for all languages (homepage always exists in all languages)
	for _, l := range allLanguages {
		xml += fmt.Sprintf(`
    <xhtml:link rel="alternate" hreflang="%s" href="%s/%s/"/>`, l, baseURL, l)
	}
	xml += fmt.Sprintf(`
    <xhtml:link rel="alternate" hreflang="x-default" href="%s/en/"/>
  </url>`, baseURL)

	// Add articles - only include hreflang for existing translations
	if s.articleService != nil {
		filters := services.ArticleFilters{Status: "published"}
		articles, _, _ := s.articleService.List(c.Request.Context(), 1000, 0, filters, "published_at", "DESC")

		// Group articles by translation_group_id to avoid duplicates
		processedGroups := make(map[uint64]bool)

		for _, article := range articles {
			// Skip if we've already processed this translation group
			groupID := article.ID
			if article.TranslationGroupID != nil {
				groupID = *article.TranslationGroupID
			}
			if processedGroups[groupID] {
				continue
			}
			processedGroups[groupID] = true

			lastMod := time.Now().Format("2006-01-02")
			if !article.UpdatedAt.IsZero() {
				lastMod = article.UpdatedAt.Format("2006-01-02")
			}

			// Get available translations for this article
			availableTranslations, err := s.articleService.GetAvailableTranslations(c.Request.Context(), article.ID)
			var availableLangs []string
			if err == nil && len(availableTranslations) > 0 {
				for _, trans := range availableTranslations {
					availableLangs = append(availableLangs, trans.LanguageCode)
				}
			} else {
				availableLangs = []string{article.LanguageCode}
			}

			// Only add URL if this language has a translation
			hasCurrentLang := false
			for _, l := range availableLangs {
				if l == lang {
					hasCurrentLang = true
					break
				}
			}
			if !hasCurrentLang {
				continue
			}

			xml += fmt.Sprintf(`
  <url>
    <loc>%s/%s/article/%s</loc>
    <lastmod>%s</lastmod>
    <changefreq>weekly</changefreq>
    <priority>0.8</priority>`, baseURL, lang, article.Slug, lastMod)

			// Add hreflang ONLY for existing translations
			for _, l := range availableLangs {
				xml += fmt.Sprintf(`
    <xhtml:link rel="alternate" hreflang="%s" href="%s/%s/article/%s"/>`, l, baseURL, l, article.Slug)
			}
			
			// x-default points to English if available, otherwise first available language
			xDefaultLang := "en"
			hasEnglish := false
			for _, l := range availableLangs {
				if l == "en" {
					hasEnglish = true
					break
				}
			}
			if !hasEnglish && len(availableLangs) > 0 {
				xDefaultLang = availableLangs[0]
			}
			xml += fmt.Sprintf(`
    <xhtml:link rel="alternate" hreflang="x-default" href="%s/%s/article/%s"/>
  </url>`, baseURL, xDefaultLang, article.Slug)
		}
	}

	// Add categories - only include hreflang for existing translations
	if s.categoryService != nil {
		categories, _ := s.categoryService.GetAll()
		
		// Group categories by translation_group_id to avoid duplicates
		processedGroups := make(map[uint64]bool)
		
		for _, cat := range categories {
			// Skip if we've already processed this translation group
			groupID := cat.ID
			if cat.TranslationGroupID != nil {
				groupID = *cat.TranslationGroupID
			}
			if processedGroups[groupID] {
				continue
			}
			processedGroups[groupID] = true

			// Get available translations for this category
			availableTranslations, err := s.categoryService.GetAllTranslations(groupID)
			var availableLangs []string
			if err == nil && len(availableTranslations) > 0 {
				for _, trans := range availableTranslations {
					availableLangs = append(availableLangs, trans.LanguageCode)
				}
			} else {
				availableLangs = []string{cat.LanguageCode}
			}

			// Only add URL if this language has a translation
			hasCurrentLang := false
			for _, l := range availableLangs {
				if l == lang {
					hasCurrentLang = true
					break
				}
			}
			if !hasCurrentLang {
				continue
			}

			xml += fmt.Sprintf(`
  <url>
    <loc>%s/%s/category/%s</loc>
    <changefreq>weekly</changefreq>
    <priority>0.7</priority>`, baseURL, lang, cat.Slug)

			// Add hreflang ONLY for existing translations
			for _, l := range availableLangs {
				xml += fmt.Sprintf(`
    <xhtml:link rel="alternate" hreflang="%s" href="%s/%s/category/%s"/>`, l, baseURL, l, cat.Slug)
			}
			
			// x-default
			xDefaultLang := "en"
			hasEnglish := false
			for _, l := range availableLangs {
				if l == "en" {
					hasEnglish = true
					break
				}
			}
			if !hasEnglish && len(availableLangs) > 0 {
				xDefaultLang = availableLangs[0]
			}
			xml += fmt.Sprintf(`
    <xhtml:link rel="alternate" hreflang="x-default" href="%s/%s/category/%s"/>
  </url>`, baseURL, xDefaultLang, cat.Slug)
		}
	}

	// Add tags - only include hreflang for existing translations
	if s.tagService != nil {
		tags, _ := s.tagService.GetAll()
		
		// Group tags by translation_group_id to avoid duplicates
		processedGroups := make(map[uint64]bool)
		
		for _, tag := range tags {
			// Skip if we've already processed this translation group
			groupID := tag.ID
			if tag.TranslationGroupID != nil {
				groupID = *tag.TranslationGroupID
			}
			if processedGroups[groupID] {
				continue
			}
			processedGroups[groupID] = true

			// Get available translations for this tag
			availableTranslations, err := s.tagService.GetAllTranslations(groupID)
			var availableLangs []string
			if err == nil && len(availableTranslations) > 0 {
				for _, trans := range availableTranslations {
					availableLangs = append(availableLangs, trans.LanguageCode)
				}
			} else {
				availableLangs = []string{tag.LanguageCode}
			}

			// Only add URL if this language has a translation
			hasCurrentLang := false
			for _, l := range availableLangs {
				if l == lang {
					hasCurrentLang = true
					break
				}
			}
			if !hasCurrentLang {
				continue
			}

			xml += fmt.Sprintf(`
  <url>
    <loc>%s/%s/tag/%s</loc>
    <changefreq>weekly</changefreq>
    <priority>0.6</priority>`, baseURL, lang, tag.Slug)

			// Add hreflang ONLY for existing translations
			for _, l := range availableLangs {
				xml += fmt.Sprintf(`
    <xhtml:link rel="alternate" hreflang="%s" href="%s/%s/tag/%s"/>`, l, baseURL, l, tag.Slug)
			}
			
			// x-default
			xDefaultLang := "en"
			hasEnglish := false
			for _, l := range availableLangs {
				if l == "en" {
					hasEnglish = true
					break
				}
			}
			if !hasEnglish && len(availableLangs) > 0 {
				xDefaultLang = availableLangs[0]
			}
			xml += fmt.Sprintf(`
    <xhtml:link rel="alternate" hreflang="x-default" href="%s/%s/tag/%s"/>
  </url>`, baseURL, xDefaultLang, tag.Slug)
		}
	}

	// Add static pages - these exist in all languages
	staticPages := []string{"latest", "trending", "categories", "tags", "about", "contact"}
	for _, page := range staticPages {
		xml += fmt.Sprintf(`
  <url>
    <loc>%s/%s/%s</loc>
    <changefreq>weekly</changefreq>
    <priority>0.5</priority>`, baseURL, lang, page)

		for _, l := range allLanguages {
			xml += fmt.Sprintf(`
    <xhtml:link rel="alternate" hreflang="%s" href="%s/%s/%s"/>`, l, baseURL, l, page)
		}
		xml += fmt.Sprintf(`
    <xhtml:link rel="alternate" hreflang="x-default" href="%s/en/%s"/>
  </url>`, baseURL, page)
	}

	xml += `
</urlset>`

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.String(http.StatusOK, xml)
}

// handleRobotsTxt generates robots.txt
func (s *Server) handleRobotsTxt(c *gin.Context) {
	// Try to get custom robots.txt from admin settings via API
	// For now, use the shared variable from api package
	robotsTxt := api.GetRobotsTxtContent()

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, robotsTxt)
}

// newDevelopmentServer creates a server with mock services for development mode
func newDevelopmentServer(cfg *config.Config, router *gin.Engine) (*Server, error) {
	log.Println("Initializing mock services for development mode...")

	// Create mock RSS service
	mockRSSService := services.NewMockRSSService()
	rssHandlers := api.NewRSSHandlers(mockRSSService)

	// Create mock Google News sitemap service
	mockGoogleNewsSitemapService := services.NewMockGoogleNewsSitemapService()
	googleNewsHandlers := api.NewGoogleNewsHandlers(mockGoogleNewsSitemapService)

	// Create a minimal template engine for development
	templateEngine := templates.NewTemplateEngine(true)

	// Create a basic auth service for development mode
	devAuthService := auth.NewAuthService(cfg.JWT.Secret, cfg.JWT.Secret)

	server := &Server{
		config:             cfg,
		router:             router,
		cache:              nil, // No cache in dev mode
		db:                 nil, // No database in dev mode
		apiRouter:          nil, // No API router in dev mode
		templateEngine:     templateEngine,
		rssHandlers:        rssHandlers,
		googleNewsHandlers: googleNewsHandlers,
		authService:        devAuthService,
	}

	// Setup development routes
	server.setupDevRoutes()

	// Setup admin routes for development mode too
	server.setupAdminFrontendRoutes()

	log.Println("Development server initialized with mock services")
	return server, nil
}

// setupDevRoutes sets up additional routes for development mode
func (s *Server) setupDevRoutes() {
	// Website pages with mock content
	s.router.GET("/category/:slug", s.handleDevCategory)
	s.router.GET("/tag/:slug", s.handleDevTag)
	s.router.GET("/latest", s.handleDevLatest)
	s.router.GET("/trending", s.handleDevTrending)
	s.router.GET("/article/:slug", s.handleDevArticle)
}

// setupMultilingualRoutes sets up language-prefixed routes for multilingual support
func (s *Server) setupMultilingualRoutes() {
	log.Println("Setting up multilingual routes...")

	// Supported languages
	supportedLanguages := []string{"en", "de", "fr", "es", "ar"}

	for _, lang := range supportedLanguages {
		langGroup := s.router.Group("/" + lang)
		{
			// Homepage for each language
			langGroup.GET("/", s.handleMultilingualHomepage)
			langGroup.GET("", s.handleMultilingualHomepage)

			// Article pages
			langGroup.GET("/article/:slug", s.handleMultilingualArticle)

			// Category pages
			langGroup.GET("/category/:slug", s.handleMultilingualCategory)
			langGroup.GET("/categories", s.handleMultilingualCategories)

			// Tag pages
			langGroup.GET("/tag/:slug", s.handleMultilingualTag)
			langGroup.GET("/tags", s.handleMultilingualTags)

			// Content listing pages
			langGroup.GET("/latest", s.handleMultilingualLatest)
			langGroup.GET("/trending", s.handleMultilingualTrending)
			langGroup.GET("/search", s.handleMultilingualSearch)

			// Static pages
			langGroup.GET("/about", s.handleMultilingualAbout)
			langGroup.GET("/contact", s.handleMultilingualContact)
		}
		log.Printf("Route group registered: /%s/*", lang)
	}

	// Multilingual sitemaps (language-specific only, main sitemap.xml is in setupGoogleNewsRoutes)
	for _, lang := range supportedLanguages {
		s.router.GET("/sitemap-"+lang+".xml", s.handleLanguageSitemap)
	}

	log.Println("Multilingual routes setup completed")
}

// validateLanguageMiddleware validates the language code in the URL
func (s *Server) validateLanguageMiddleware(c *gin.Context) {
	lang := c.Param("lang")
	supportedLanguages := map[string]bool{
		"en": true,
		"de": true,
		"fr": true,
		"es": true,
		"ar": true,
	}

	if !supportedLanguages[lang] {
		c.Redirect(http.StatusMovedPermanently, "/en"+c.Request.URL.Path)
		c.Abort()
		return
	}

	// Set language info in context
	c.Set("language_code", lang)
	c.Set("language_direction", getLanguageDirection(lang))
	c.Next()
}

// getLanguageDirection returns the text direction for a language
func getLanguageDirection(lang string) string {
	if lang == "ar" {
		return "rtl"
	}
	return "ltr"
}

// getLanguageNativeName returns the native name for a language
func getLanguageNativeName(lang string) string {
	names := map[string]string{
		"en": "English",
		"de": "Deutsch",
		"fr": "Français",
		"es": "Español",
		"ar": "العربية",
	}
	if name, ok := names[lang]; ok {
		return name
	}
	return lang
}

// getLanguageName returns the English name for a language
func getLanguageName(lang string) string {
	names := map[string]string{
		"en": "English",
		"de": "German",
		"fr": "French",
		"es": "Spanish",
		"ar": "Arabic",
	}
	if name, ok := names[lang]; ok {
		return name
	}
	return lang
}

// generateAlternateURLs generates alternate language URLs for hreflang tags
// This is the default implementation for static pages (homepage, about, etc.)
// For articles, use generateArticleAlternateURLs instead
func (s *Server) generateAlternateURLs(currentPath string) []map[string]string {
	languages := []string{"en", "de", "fr", "es", "ar"}
	baseURL := s.config.App.BaseURL
	if baseURL == "" {
		baseURL = "https://a.10top.shop"
	}

	// Remove any existing language prefix from the path
	cleanPath := currentPath
	for _, lang := range languages {
		prefix := "/" + lang + "/"
		if strings.HasPrefix(cleanPath, prefix) {
			cleanPath = cleanPath[len(prefix)-1:]
			break
		}
		if cleanPath == "/"+lang {
			cleanPath = "/"
			break
		}
	}

	var alternates []map[string]string
	for _, lang := range languages {
		url := baseURL + "/" + lang + cleanPath
		// Clean up double slashes
		url = strings.ReplaceAll(url, "//", "/")
		url = strings.Replace(url, ":/", "://", 1)

		alternates = append(alternates, map[string]string{
			"lang": lang,
			"url":  url,
		})
	}

	// Add x-default pointing to English
	alternates = append(alternates, map[string]string{
		"lang": "x-default",
		"url":  baseURL + "/en" + cleanPath,
	})

	return alternates
}

// generateAlternateURLsForTranslations generates hreflang URLs only for existing translations
// This should be used for articles, categories, and tags that have translation groups
func (s *Server) generateAlternateURLsForTranslations(availableLanguages []string, pathTemplate string) []map[string]string {
	baseURL := s.config.App.BaseURL
	if baseURL == "" {
		baseURL = "https://a.10top.shop"
	}

	var alternates []map[string]string
	var hasEnglish bool
	
	for _, lang := range availableLanguages {
		url := baseURL + "/" + lang + pathTemplate
		// Clean up double slashes
		url = strings.ReplaceAll(url, "//", "/")
		url = strings.Replace(url, ":/", "://", 1)

		alternates = append(alternates, map[string]string{
			"lang": lang,
			"url":  url,
		})
		
		if lang == "en" {
			hasEnglish = true
		}
	}

	// Add x-default pointing to English if available, otherwise first language
	xDefaultLang := "en"
	if !hasEnglish && len(availableLanguages) > 0 {
		xDefaultLang = availableLanguages[0]
	}
	
	xDefaultURL := baseURL + "/" + xDefaultLang + pathTemplate
	xDefaultURL = strings.ReplaceAll(xDefaultURL, "//", "/")
	xDefaultURL = strings.Replace(xDefaultURL, ":/", "://", 1)
	
	alternates = append(alternates, map[string]string{
		"lang": "x-default",
		"url":  xDefaultURL,
	})

	return alternates
}

// generateCanonicalURL generates the canonical URL for a page
func (s *Server) generateCanonicalURL(lang, path string) string {
	baseURL := s.config.App.BaseURL
	if baseURL == "" {
		baseURL = "https://a.10top.shop"
	}

	url := baseURL + "/" + lang + path
	// Clean up double slashes
	url = strings.ReplaceAll(url, "//", "/")
	url = strings.Replace(url, ":/", "://", 1)
	return url
}

// getAvailableLanguages returns language info for the language switcher
func (s *Server) getAvailableLanguages(currentPath string) []map[string]interface{} {
	languages := []struct {
		Code       string
		Name       string
		NativeName string
		Direction  string
	}{
		{"en", "English", "English", "ltr"},
		{"de", "German", "Deutsch", "ltr"},
		{"fr", "French", "Français", "ltr"},
		{"es", "Spanish", "Español", "ltr"},
		{"ar", "Arabic", "العربية", "rtl"},
	}

	// Remove any existing language prefix from the path
	cleanPath := currentPath
	for _, lang := range languages {
		prefix := "/" + lang.Code + "/"
		if strings.HasPrefix(cleanPath, prefix) {
			cleanPath = cleanPath[len(prefix)-1:]
			break
		}
		if cleanPath == "/"+lang.Code {
			cleanPath = "/"
			break
		}
	}

	var result []map[string]interface{}
	for _, lang := range languages {
		url := "/" + lang.Code + cleanPath
		// Clean up double slashes
		url = strings.ReplaceAll(url, "//", "/")

		result = append(result, map[string]interface{}{
			"Code":       lang.Code,
			"Name":       lang.Name,
			"NativeName": lang.NativeName,
			"Direction":  lang.Direction,
			"URL":        url,
		})
	}

	return result
}

// addMultilingualData adds multilingual data to template context
func (s *Server) addMultilingualData(data map[string]interface{}, lang, currentPath string) {
	data["LanguageCode"] = lang
	data["LanguageDirection"] = getLanguageDirection(lang)
	data["LanguageName"] = getLanguageName(lang)
	data["LanguageNativeName"] = getLanguageNativeName(lang)
	data["AlternateURLs"] = s.generateAlternateURLs(currentPath)
	data["AvailableLanguages"] = s.getAvailableLanguages(currentPath)
	data["CanonicalURL"] = s.generateCanonicalURL(lang, currentPath)
	data["IsRTL"] = lang == "ar"
	data["BaseURL"] = s.config.App.BaseURL
	// Update navigation with translated items for this language
	data["Navigation"] = s.getNavigationForLanguage(lang)
}

// setupProductionFrontendRoutes sets up frontend routes for production mode
// This was missing from Task 22 implementation
func (s *Server) setupProductionFrontendRoutes() {
	log.Println("Setting up production frontend routes...")

	// Article pages
	s.router.GET("/article/:slug", s.handleProductionArticle)
	log.Println("Route registered: /article/:slug")

	// Category pages
	s.router.GET("/category/:slug", s.handleProductionCategory)
	log.Println("Route registered: /category/:slug")
	s.router.GET("/categories", s.handleProductionCategories)
	log.Println("Route registered: /categories")

	// Tag pages
	s.router.GET("/tag/:slug", s.handleProductionTag)
	log.Println("Route registered: /tag/:slug")
	s.router.GET("/tags", s.handleProductionTags)
	log.Println("Route registered: /tags")

	// Content listing pages
	s.router.GET("/latest", s.handleProductionLatest)
	s.router.GET("/trending", s.handleProductionTrending)
	s.router.GET("/search", s.handleProductionSearch)
	log.Println("Route registered: /search")

	// Static pages
	s.router.GET("/about", s.handleProductionAbout)
	s.router.GET("/contact", s.handleProductionContact)

	// API endpoints for article engagement are handled by the API router at /api/v1/articles/:id/like etc.

	// NOTE: Removed catch-all /:slug route to prevent duplicate URLs for articles
	// All articles should be accessed via /article/:slug for SEO consistency
	// The catch-all route was causing duplicate content issues (same article at /slug and /article/slug)

	log.Println("Production frontend routes setup completed")
}

// Production frontend handlers (missing from Task 22)
func (s *Server) handleProductionArticle(c *gin.Context) {
	slug := c.Param("slug")

	// Get the real article from the database
	if s.articleService != nil {
		// Get article by slug from the service
		article, err := s.articleService.GetBySlug(c.Request.Context(), slug)
		if err != nil {
			// Article not found, show 404
			data := s.createBaseTemplateData(c)
			data["Title"] = "Article Not Found"
			data["Content"] = "<div class='error-message'><h2>Article Not Found</h2><p>The article you're looking for doesn't exist or may have been moved.</p><a href='/'>← Back to Home</a></div>"

			if s.templateEngine != nil {
				if html, err := s.templateEngine.Render("homepage", data); err == nil {
					c.Header("Content-Type", "text/html; charset=utf-8")
					c.String(http.StatusNotFound, html)
					return
				}
			}
			c.String(http.StatusNotFound, "Article not found: "+slug)
			return
		}

		// Debug: Log the article data
		log.Printf("Article data: ID=%d, ViewCount=%d, LikeCount=%d, DislikeCount=%d",
			article.ID, article.ViewCount, article.LikeCount, article.DislikeCount)

		// Convert article to template data using actual model fields
		articleData := gin.H{
			"ID":           article.ID,
			"Title":        article.Title,
			"Slug":         article.Slug,
			"Content":      article.Content,
			"Excerpt":      article.Excerpt,
			"PublishedAt":  article.PublishedAt,
			"ViewCount":    article.ViewCount,
			"LikeCount":    article.LikeCount,
			"DislikeCount": article.DislikeCount,
			"CommentCount": 0, // TODO: Implement comment count
			"ReadTime":     calculateReadTime(article.Content),
			// Add SEO fields from individual columns
			"MetaTitle":       article.MetaTitle,
			"MetaDescription": article.MetaDescription,
			"CanonicalURL":    article.CanonicalURL,
			"SchemaType":      article.SchemaType,
		}

		// Get featured image if available (only local paths starting with /uploads/)
		if article.FeaturedImageID != nil && *article.FeaturedImageID > 0 {
			log.Printf("DEBUG: FeaturedImageID is %d", *article.FeaturedImageID)
			if s.db != nil {
				var imageURL sql.NullString
				query := "SELECT CASE WHEN original_url LIKE '/uploads/%' THEN original_url ELSE NULL END FROM images WHERE id = $1"
				err := s.db.DB.QueryRow(query, *article.FeaturedImageID).Scan(&imageURL)
				log.Printf("DEBUG: Image query result - err: %v, valid: %v, value: %s", err, imageURL.Valid, imageURL.String)
				if err == nil && imageURL.Valid && imageURL.String != "" {
					articleData["FeaturedImage"] = imageURL.String
					log.Printf("DEBUG: Set FeaturedImage to %s", imageURL.String)
				}
			}
			// Build responsive image data for the article
			if s.mediaService != nil {
				imageData := s.buildResponsiveImageData(*article.FeaturedImageID, article.Title)
				if imageData != nil {
					articleData["ImageData"] = imageData
					log.Printf("DEBUG: Set ImageData with HasVariants=%v", imageData.HasVariants)

					// Trigger static regeneration in background if static file doesn't exist
					// This ensures articles with responsive images get proper static files
					if s.staticGenerator != nil && imageData.HasVariants {
						go func() {
							staticPath := fmt.Sprintf("static-html/articles/%s/index.html", article.Slug)
							if _, err := os.Stat(staticPath); os.IsNotExist(err) {
								log.Printf("Static file missing for article %s with responsive images, regenerating...", article.Slug)
								ctx := context.Background()
								if err := s.staticGenerator.GenerateArticlePage(ctx, article); err != nil {
									log.Printf("Warning: Failed to regenerate static file for article %s: %v", article.Slug, err)
								} else {
									log.Printf("Successfully regenerated static file for article %s", article.Slug)
								}
							}
						}()
					}
				}
			}
		} else {
			log.Printf("DEBUG: FeaturedImageID is nil or 0")
		}

		// Increment view count in background
		if s.db != nil {
			go func() {
				updateQuery := "UPDATE articles SET view_count = view_count + 1 WHERE id = $1"
				_, err := s.db.DB.Exec(updateQuery, article.ID)
				if err != nil {
					log.Printf("Error incrementing view count for article %d: %v", article.ID, err)
				}
			}()
		}

		// Add tags if available from article model
		if len(article.Tags) > 0 {
			tags := make([]gin.H, len(article.Tags))
			for i, tag := range article.Tags {
				tags[i] = gin.H{
					"Name": tag.Name,
					"Slug": tag.Slug,
				}
			}
			articleData["Tags"] = tags
		}

		// Add author info
		articleData["Author"] = gin.H{
			"FirstName": "Article",
			"LastName":  "Author",
			"Bio":       "Content Creator",
		}

		// Get category info and add it to the article data
		// Check if article has multiple categories loaded
		if len(article.Categories) > 0 {
			categories := make([]gin.H, len(article.Categories))
			for i, category := range article.Categories {
				categories[i] = gin.H{
					"ID":          category.ID,
					"Name":        category.Name,
					"Slug":        category.Slug,
					"Description": category.Description,
				}
			}
			articleData["Categories"] = categories
			articleData["Category"] = categories[0] // First category for backward compatibility
		} else if article.CategoryID > 0 {
			// Fallback to single category lookup
			categoryData, err := s.getCategoryByID(article.CategoryID)
			if err == nil {
				articleData["Category"] = categoryData
				articleData["Categories"] = []gin.H{categoryData}
			} else {
				fallbackCategory := gin.H{"Name": "General", "Slug": "general"}
				articleData["Category"] = fallbackCategory
				articleData["Categories"] = []gin.H{fallbackCategory}
			}
		} else {
			fallbackCategory := gin.H{"Name": "General", "Slug": "general"}
			articleData["Category"] = fallbackCategory
			articleData["Categories"] = []gin.H{fallbackCategory}
		}

		// Get tags for the article and add them to article data (if not already loaded)
		if articleData["Tags"] == nil {
			tags, err := s.getArticleTags(article.ID)
			if err == nil && len(tags) > 0 {
				articleData["Tags"] = tags
			} else {
				// Provide empty tags array if none found
				articleData["Tags"] = []gin.H{}
			}
		}

		// Create template data
		data := s.createBaseTemplateData(c)
		data["Title"] = article.Title
		data["Article"] = articleData

		// Add HeroImage to root level for base template preload tag
		if featuredImage, ok := articleData["FeaturedImage"].(string); ok && featuredImage != "" {
			data["HeroImage"] = featuredImage
			log.Printf("DEBUG: HeroImage set to: %s", featuredImage)
		} else {
			log.Printf("DEBUG: FeaturedImage not found or empty in articleData")
		}

		// Add SEO fields to root level for base template
		// Use meta fields if they exist, otherwise use article title/excerpt
		if article.MetaTitle != "" {
			data["SEOTitle"] = article.MetaTitle
		} else {
			data["SEOTitle"] = article.Title + " - " + s.config.App.Name
		}

		if article.MetaDescription != "" {
			data["SEODescription"] = article.MetaDescription
		} else {
			data["SEODescription"] = article.Excerpt
		}

		if article.CanonicalURL != "" {
			data["CanonicalURL"] = article.CanonicalURL
		} else {
			data["CanonicalURL"] = fmt.Sprintf("%s/article/%s", s.config.App.BaseURL, article.Slug)
		}

		// Set OG and Twitter image URLs (must be absolute URLs)
		baseURL := s.config.App.BaseURL
		if baseURL == "" {
			baseURL = fmt.Sprintf("https://%s", c.Request.Host)
		}
		if featuredImage, ok := articleData["FeaturedImage"].(string); ok && featuredImage != "" {
			// Convert relative URL to absolute URL
			if strings.HasPrefix(featuredImage, "/") {
				data["OGImage"] = baseURL + featuredImage
				data["TwitterImage"] = baseURL + featuredImage
			} else {
				data["OGImage"] = featuredImage
				data["TwitterImage"] = featuredImage
			}
		}

		// Set OG type for articles
		data["OGType"] = "article"

		// Build breadcrumb items
		breadcrumbItems := []gin.H{
			{"Name": "Home", "URL": baseURL, "Position": 1},
		}
		position := 2

		// Add category to breadcrumbs if available
		if categoryData, ok := articleData["Category"].(gin.H); ok {
			if catName, ok := categoryData["Name"].(string); ok && catName != "" {
				if catSlug, ok := categoryData["Slug"].(string); ok && catSlug != "" {
					breadcrumbItems = append(breadcrumbItems, gin.H{
						"Name":     catName,
						"URL":      fmt.Sprintf("%s/category/%s", baseURL, catSlug),
						"Position": position,
					})
					position++
				}
			}
		}

		// Add current article (active item)
		breadcrumbItems = append(breadcrumbItems, gin.H{
			"Name":     article.Title,
			"URL":      fmt.Sprintf("%s/article/%s", baseURL, article.Slug),
			"Position": position,
			"Active":   true,
		})

		// Generate BreadcrumbList JSON-LD schema
		breadcrumbListElements := make([]gin.H, len(breadcrumbItems))
		for i, item := range breadcrumbItems {
			breadcrumbListElements[i] = gin.H{
				"@type":    "ListItem",
				"position": item["Position"],
				"name":     item["Name"],
				"item":     item["URL"],
			}
		}

		breadcrumbSchema := gin.H{
			"@context":        "https://schema.org",
			"@type":           "BreadcrumbList",
			"itemListElement": breadcrumbListElements,
		}

		// Convert breadcrumb schema to JSON string
		breadcrumbJSON, err := json.Marshal(breadcrumbSchema)
		if err == nil {
			data["BreadcrumbSchema"] = string(breadcrumbJSON)
			log.Printf("DEBUG: BreadcrumbSchema set: %s", string(breadcrumbJSON)[:100])
		} else {
			log.Printf("ERROR: Failed to marshal breadcrumb schema: %v", err)
		}

		// Also pass breadcrumb items for HTML rendering
		data["BreadcrumbItems"] = breadcrumbItems

		// Generate NewsArticle structured data
		publishedTime := ""
		modifiedTime := ""
		if article.PublishedAt != nil {
			publishedTime = article.PublishedAt.Format(time.RFC3339)
		}
		if !article.UpdatedAt.IsZero() {
			modifiedTime = article.UpdatedAt.Format(time.RFC3339)
		}

		// Build keywords from tags
		var keywords []string
		if tags, ok := articleData["Tags"].([]gin.H); ok {
			for _, tag := range tags {
				if name, ok := tag["Name"].(string); ok {
					keywords = append(keywords, name)
				}
			}
		}

		articleSchema := gin.H{
			"@context":      "https://schema.org",
			"@type":         "NewsArticle",
			"headline":      article.Title,
			"description":   article.Excerpt,
			"datePublished": publishedTime,
			"dateModified":  modifiedTime,
			"mainEntityOfPage": gin.H{
				"@type": "WebPage",
				"@id":   fmt.Sprintf("%s/article/%s", baseURL, article.Slug),
			},
			"author": gin.H{
				"@type": "Person",
				"name":  "Article Author",
			},
			"publisher": gin.H{
				"@type": "Organization",
				"name":  s.config.App.Name,
				"logo": gin.H{
					"@type": "ImageObject",
					"url":   fmt.Sprintf("%s/static/images/logo.png", baseURL),
				},
			},
		}

		// Add image if available
		if featuredImage, ok := articleData["FeaturedImage"].(string); ok && featuredImage != "" {
			articleSchema["image"] = fmt.Sprintf("%s%s", baseURL, featuredImage)
		}

		// Add keywords if available
		if len(keywords) > 0 {
			articleSchema["keywords"] = strings.Join(keywords, ", ")
		}

		// Convert article schema to JSON string
		articleSchemaJSON, err := json.Marshal(articleSchema)
		if err == nil {
			data["StructuredData"] = string(articleSchemaJSON)
		}

		// Add related articles (could be enhanced to get real related articles)
		data["RelatedArticles"] = []gin.H{
			{"Title": "Related Article 1", "Slug": "related-1", "Excerpt": "Related content"},
			{"Title": "Related Article 2", "Slug": "related-2", "Excerpt": "More related content"},
		}

		// Debug: Log the article data being passed to template
		log.Printf("Article data for template: Title=%s, MetaTitle=%s, Tags=%v, Category=%v",
			articleData["Title"], articleData["MetaTitle"], articleData["Tags"], articleData["Category"])

		if s.templateEngine != nil {
			if html, err := s.templateEngine.Render("article", data); err == nil {
				c.Header("Content-Type", "text/html; charset=utf-8")
				c.String(http.StatusOK, html)
				return
			} else {
				log.Printf("Template render error for 'article': %v", err)
			}
		} else {
			log.Printf("Template engine is nil")
		}
	}

	// Fallback to simple response
	c.String(http.StatusOK, "Article: "+slug+" (Template not available)")
}

func (s *Server) handleProductionCategory(c *gin.Context) {
	slug := c.Param("slug")
	log.Printf("handleProductionCategory called with slug: %s", slug)

	data := s.createBaseTemplateData(c)

	// Get real category from database
	if s.categoryService != nil {
		category, err := s.categoryService.GetBySlug(slug)
		if err != nil {
			log.Printf("Category not found: %s, error: %v", slug, err)
			c.String(http.StatusNotFound, "Category not found")
			return
		}

		data["Title"] = "Category: " + category.Name
		data["Category"] = gin.H{
			"ID":          category.ID,
			"Name":        category.Name,
			"Slug":        category.Slug,
			"Description": category.Description,
		}

		// Get real articles for this category (including from junction table)
		articles := []gin.H{}
		if s.db != nil {
			query := `
				SELECT DISTINCT a.id, a.title, a.slug, a.excerpt, a.published_at, a.view_count,
				       CASE WHEN i.original_url LIKE '/uploads/%' THEN i.original_url ELSE NULL END as featured_image,
				       u.first_name, u.last_name
				FROM articles a
				LEFT JOIN users u ON a.author_id = u.id
				LEFT JOIN images i ON a.featured_image_id = i.id
				LEFT JOIN article_categories ac ON a.id = ac.article_id
				WHERE (a.category_id = $1 OR ac.category_id = $1) AND a.status = 'published'
				ORDER BY a.published_at DESC
				LIMIT 20
			`

			rows, err := s.db.DB.Query(query, category.ID)
			if err != nil {
				log.Printf("Error fetching articles for category %s: %v", slug, err)
			} else {
				defer rows.Close()

				for rows.Next() {
					var id uint64
					var title, articleSlug, excerpt string
					var publishedAt time.Time
					var viewCount int
					var featuredImage sql.NullString
					var firstName, lastName sql.NullString

					err := rows.Scan(&id, &title, &articleSlug, &excerpt, &publishedAt, &viewCount, &featuredImage, &firstName, &lastName)
					if err != nil {
						log.Printf("Error scanning article row: %v", err)
						continue
					}

					// Format author name
					author := "Unknown Author"
					if firstName.Valid && lastName.Valid {
						author = firstName.String + " " + lastName.String
					} else if firstName.Valid {
						author = firstName.String
					}

					// Prepare article data
					articleData := gin.H{
						"ID":      id,
						"Title":   title,
						"Slug":    articleSlug,
						"Excerpt": excerpt,
						"Author":  author,
						"TimeAgo": formatTimeAgo(&publishedAt),
						"Views":   viewCount,
					}

					// Add featured image if available
					if featuredImage.Valid && featuredImage.String != "" {
						articleData["FeaturedImage"] = featuredImage.String
					}

					articles = append(articles, articleData)
				}
			}
		}

		data["Articles"] = articles
	} else {
		c.String(http.StatusNotFound, "Category service not available")
		return
	}

	if s.templateEngine != nil {
		if html, err := s.templateEngine.Render("category", data); err == nil {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, html)
			return
		}
	}

	c.String(http.StatusOK, "Category: "+slug+" (Template not available)")
}

func (s *Server) handleProductionCategories(c *gin.Context) {
	log.Println("handleProductionCategories called")
	data := s.createBaseTemplateData(c)
	data["Title"] = "All Categories"

	// Get real categories from database
	if s.categoryService != nil {
		categories, err := s.categoryService.GetAll()
		if err == nil {
			// Convert to template format
			categoryData := make([]gin.H, len(categories))
			for i, cat := range categories {
				categoryData[i] = gin.H{
					"ID":           cat.ID,
					"Name":         cat.Name,
					"Slug":         cat.Slug,
					"Description":  cat.Description,
					"ImageURL":     cat.GetImageURL(),
					"ImageAltText": cat.GetImageAltText(),
					"Count":        s.getCategoryArticleCount(cat.ID),
				}
			}
			data["Categories"] = categoryData
		} else {
			log.Printf("Error fetching categories: %v", err)
			// Fallback to empty array
			data["Categories"] = []gin.H{}
		}
	} else {
		// Fallback if service not available
		data["Categories"] = []gin.H{}
	}

	if s.templateEngine != nil {
		if html, err := s.templateEngine.Render("categories", data); err == nil {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, html)
			return
		}
	}

	c.String(http.StatusOK, "All Categories (Template not available)")
}

func (s *Server) handleProductionTag(c *gin.Context) {
	slug := c.Param("slug")
	log.Printf("handleProductionTag called with slug: %s", slug)

	data := s.createBaseTemplateData(c)

	// Get real tag from database
	if s.tagService != nil {
		tag, err := s.tagService.GetBySlug(slug)
		if err != nil {
			log.Printf("Tag not found: %s, error: %v", slug, err)
			c.String(http.StatusNotFound, "Tag not found")
			return
		}

		data["Title"] = "Tag: " + tag.Name
		data["Tag"] = gin.H{
			"ID":          tag.ID,
			"Name":        tag.Name,
			"Slug":        tag.Slug,
			"Description": tag.Description,
			"Color":       tag.Color,
		}

		// Get real articles for this tag
		articles := []gin.H{}
		if s.db != nil {
			query := `
				SELECT DISTINCT a.id, a.title, a.slug, a.excerpt, a.published_at, a.view_count,
				       CASE WHEN i.original_url LIKE '/uploads/%' THEN i.original_url ELSE NULL END as featured_image,
				       u.first_name, u.last_name
				FROM articles a
				LEFT JOIN users u ON a.author_id = u.id
				LEFT JOIN images i ON a.featured_image_id = i.id
				JOIN article_tags at ON a.id = at.article_id
				WHERE at.tag_id = $1 AND a.status = 'published'
				ORDER BY a.published_at DESC
				LIMIT 20
			`

			rows, err := s.db.DB.Query(query, tag.ID)
			if err != nil {
				log.Printf("Error fetching articles for tag %s: %v", slug, err)
			} else {
				defer rows.Close()

				for rows.Next() {
					var id uint64
					var title, articleSlug, excerpt string
					var publishedAt time.Time
					var viewCount int
					var featuredImage sql.NullString
					var firstName, lastName sql.NullString

					err := rows.Scan(&id, &title, &articleSlug, &excerpt, &publishedAt, &viewCount, &featuredImage, &firstName, &lastName)
					if err != nil {
						log.Printf("Error scanning article row: %v", err)
						continue
					}

					// Format author name
					author := "Unknown Author"
					if firstName.Valid && lastName.Valid {
						author = firstName.String + " " + lastName.String
					} else if firstName.Valid {
						author = firstName.String
					}

					// Prepare article data
					articleData := gin.H{
						"ID":      id,
						"Title":   title,
						"Slug":    articleSlug,
						"Excerpt": excerpt,
						"Author":  author,
						"TimeAgo": formatTimeAgo(&publishedAt),
						"Views":   viewCount,
					}

					// Add featured image if available
					if featuredImage.Valid && featuredImage.String != "" {
						articleData["FeaturedImage"] = featuredImage.String
					}

					articles = append(articles, articleData)
				}
			}
		}

		data["Articles"] = articles
	} else {
		c.String(http.StatusNotFound, "Tag service not available")
		return
	}

	if s.templateEngine != nil {
		if html, err := s.templateEngine.Render("tag", data); err == nil {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, html)
			return
		}
	}

	c.String(http.StatusOK, "Tag: "+slug+" (Template not available)")
}

func (s *Server) handleProductionTags(c *gin.Context) {
	log.Println("handleProductionTags called")
	data := s.createBaseTemplateData(c)
	data["Title"] = "All Tags"

	// Get real tags from database
	if s.tagService != nil {
		tags, err := s.tagService.GetAll()
		if err == nil {
			// Convert to template format with actual article counts
			tagData := make([]gin.H, 0)
			for _, tag := range tags {
				// Get article count for this tag
				articleCount := s.getTagArticleCount(tag.ID)

				// Include all tags (even with 0 count) for the tags page
				tagData = append(tagData, gin.H{
					"ID":    tag.ID,
					"Name":  tag.Name,
					"Slug":  tag.Slug,
					"Count": articleCount,
				})
			}

			// Sort by article count (most used first)
			for i := 0; i < len(tagData)-1; i++ {
				for j := i + 1; j < len(tagData); j++ {
					if tagData[j]["Count"].(int) > tagData[i]["Count"].(int) {
						tagData[i], tagData[j] = tagData[j], tagData[i]
					}
				}
			}

			data["AllTags"] = tagData
		} else {
			log.Printf("Error fetching tags: %v", err)
			// Fallback to empty array
			data["AllTags"] = []gin.H{}
		}
	} else {
		// Fallback if service not available
		data["AllTags"] = []gin.H{}
	}

	if s.templateEngine != nil {
		if html, err := s.templateEngine.Render("tags", data); err == nil {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, html)
			return
		}
	}

	c.String(http.StatusOK, "All Tags (Template not available)")
}

func (s *Server) handleProductionLatest(c *gin.Context) {
	data := s.createBaseTemplateData(c)
	data["Title"] = "Latest Articles"
	data["Articles"] = []gin.H{
		{
			"Title":    "Latest Breaking News Story",
			"Slug":     "latest-breaking-news",
			"Excerpt":  "This is the most recent breaking news story",
			"Author":   "News Reporter",
			"TimeAgo":  "2 minutes ago",
			"Views":    1234,
			"Category": "Breaking News",
		},
		{
			"Title":    "New Technology Innovation Announced",
			"Slug":     "tech-innovation",
			"Excerpt":  "A groundbreaking technology innovation",
			"Author":   "Tech Writer",
			"TimeAgo":  "15 minutes ago",
			"Views":    856,
			"Category": "Technology",
		},
		{
			"Title":    "Major Sports Championship Update",
			"Slug":     "sports-update",
			"Excerpt":  "Latest updates from the championship",
			"Author":   "Sports Reporter",
			"TimeAgo":  "30 minutes ago",
			"Views":    642,
			"Category": "Sports",
		},
	}

	if s.templateEngine != nil {
		if html, err := s.templateEngine.Render("latest", data); err == nil {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, html)
			return
		}
	}

	c.String(http.StatusOK, "Latest Articles (Template not available)")
}

func (s *Server) handleProductionTrending(c *gin.Context) {
	data := s.createBaseTemplateData(c)
	data["Title"] = "Trending Articles"
	data["Articles"] = []gin.H{
		{
			"Title":    "🔥 Most Viral Story of the Week",
			"Slug":     "viral-story",
			"Excerpt":  "This story has gone viral across social media",
			"Author":   "Viral Reporter",
			"TimeAgo":  "2 hours ago",
			"Views":    15420,
			"Category": "Viral",
		},
		{
			"Title":    "📈 Trending Technology News",
			"Slug":     "trending-tech",
			"Excerpt":  "This tech story is gaining massive attention",
			"Author":   "Tech Analyst",
			"TimeAgo":  "4 hours ago",
			"Views":    12350,
			"Category": "Technology",
		},
		{
			"Title":    "⚡ Breaking Sports Sensation",
			"Slug":     "trending-sports",
			"Excerpt":  "A sports story capturing everyone's attention",
			"Author":   "Sports Writer",
			"TimeAgo":  "6 hours ago",
			"Views":    9870,
			"Category": "Sports",
		},
	}

	if s.templateEngine != nil {
		if html, err := s.templateEngine.Render("trending", data); err == nil {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, html)
			return
		}
	}

	c.String(http.StatusOK, "Trending Articles (Template not available)")
}

func (s *Server) handleProductionAbout(c *gin.Context) {
	data := s.createBaseTemplateData(c)
	data["Title"] = "About Us"

	if s.templateEngine != nil {
		if html, err := s.templateEngine.Render("about", data); err == nil {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, html)
			return
		}
	}

	c.String(http.StatusOK, "About Us (Template not available)")
}

func (s *Server) handleProductionSearch(c *gin.Context) {
	data := s.createBaseTemplateData(c)

	// Get query parameters
	searchQuery := c.Query("q")
	categoryID := c.Query("category_id")
	sortBy := c.DefaultQuery("sort_by", "relevance")
	page := c.DefaultQuery("page", "1")

	// Build canonical URL
	baseURL := s.config.App.BaseURL
	if baseURL == "" {
		baseURL = "https://" + c.Request.Host
	}
	canonicalURL := baseURL + "/search"
	if searchQuery != "" {
		canonicalURL += "?q=" + searchQuery
	}

	// Parse page number
	pageNum := 1
	if p, err := strconv.Atoi(page); err == nil && p > 0 {
		pageNum = p
	}

	// Set page data
	data["Title"] = "Search"
	if searchQuery != "" {
		data["Title"] = "Search: " + searchQuery
	}
	data["SEOTitle"] = data["Title"]
	data["SEODescription"] = "Search articles on " + s.config.App.Name
	data["CanonicalURL"] = canonicalURL
	data["SearchQuery"] = searchQuery
	data["SelectedCategory"] = categoryID
	data["SortBy"] = sortBy

	// Perform server-side search for SEO and initial load
	var articles []gin.H
	var totalResults int64
	var searchTime int64

	if searchQuery != "" && s.enterpriseSearchService != nil {
		// Build search request
		searchReq := services.SearchRequest{
			Query:  searchQuery,
			Limit:  20,
			Offset: (pageNum - 1) * 20,
		}

		// Add sort
		if sortBy != "" && sortBy != "relevance" {
			searchReq.Sort = &services.SearchSort{
				Field: sortBy,
				Order: "desc",
			}
		}

		// Perform search using enterprise service wrapper
		wrapper := services.NewEnterpriseSearchServiceWrapper(s.enterpriseSearchService)
		if result, err := wrapper.Search(searchReq); err == nil {
			totalResults = result.Total
			searchTime = result.ProcessingTime

			for _, article := range result.Articles {
				// Get category info
				categoryName := ""
				categorySlug := ""
				if s.categoryService != nil {
					if cat, err := s.categoryService.GetByID(article.CategoryID); err == nil && cat != nil {
						categoryName = cat.Name
						categorySlug = cat.Slug
					}
				}

				articles = append(articles, gin.H{
					"ID":            article.ID,
					"Title":         article.Title,
					"Slug":          article.Slug,
					"Excerpt":       article.Excerpt,
					"Author":        "", // Would need to join with users table
					"Category":      categoryName,
					"CategorySlug":  categorySlug,
					"FeaturedImage": article.FeaturedImage,
					"PublishedAt":   article.PublishedAt,
					"TimeAgo":       formatTimeAgo(article.PublishedAt),
					"ViewCount":     article.ViewCount,
				})
			}
		}
	}

	data["Articles"] = articles
	data["TotalResults"] = totalResults
	data["SearchTime"] = searchTime
	data["CurrentPage"] = pageNum

	// Pagination URLs
	if pageNum > 1 {
		data["PrevPageURL"] = canonicalURL + "&page=" + strconv.Itoa(pageNum-1)
	}

	// Get categories for filter
	categories := []gin.H{}
	if s.categoryService != nil {
		if cats, err := s.categoryService.GetAll(); err == nil {
			for _, cat := range cats {
				categories = append(categories, gin.H{
					"ID":    cat.ID,
					"Name":  cat.Name,
					"Slug":  cat.Slug,
					"Count": s.getCategoryArticleCount(cat.ID),
				})
			}
		}
	}
	data["Categories"] = categories

	// Get popular tags for filter
	popularTags := []gin.H{}
	if s.tagService != nil {
		if tags, err := s.tagService.GetPopular(20); err == nil {
			for _, tag := range tags {
				popularTags = append(popularTags, gin.H{
					"ID":    tag.ID,
					"Name":  tag.Name,
					"Slug":  tag.Slug,
					"Count": s.getTagArticleCount(tag.ID),
				})
			}
		}
	}
	data["PopularTags"] = popularTags
	data["SelectedTags"] = []string{}
	data["InfiniteScroll"] = false
	data["ActiveFiltersCount"] = 0

	// Build pagination
	if totalResults > 0 {
		totalPages := int((totalResults + 19) / 20)
		pages := []int{}
		for i := 1; i <= totalPages && i <= 10; i++ {
			pages = append(pages, i)
		}
		data["Pagination"] = gin.H{
			"CurrentPage": pageNum,
			"TotalPages":  totalPages,
			"Pages":       pages,
			"HasPrev":     pageNum > 1,
			"HasNext":     pageNum < totalPages,
		}
		if pageNum < totalPages {
			data["NextPageURL"] = canonicalURL + "&page=" + strconv.Itoa(pageNum+1)
		}
	}

	// Render template
	if s.templateEngine != nil {
		if html, err := s.templateEngine.Render("search", data); err == nil {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, html)
			return
		} else {
			log.Printf("Search template render error: %v", err)
		}
	}

	c.String(http.StatusOK, "Search (Template not available)")
}

func (s *Server) handleProductionContact(c *gin.Context) {
	data := s.createBaseTemplateData(c)
	data["Title"] = "Contact Us"

	if s.templateEngine != nil {
		if html, err := s.templateEngine.Render("contact", data); err == nil {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, html)
			return
		}
	}

	c.String(http.StatusOK, "Contact Us (Template not available)")
}

// handleProductionArticleBySlug handles article pages with direct slug URLs (e.g., /my-article-slug)
func (s *Server) handleProductionArticleBySlug(c *gin.Context) {
	slug := c.Param("slug")

	// Skip known routes that shouldn't be treated as article slugs
	knownRoutes := []string{"admin", "api", "static", "assets", "rss", "sitemap", "robots.txt", "favicon.ico"}
	for _, route := range knownRoutes {
		if slug == route {
			c.Next()
			return
		}
	}

	// Try to get the real article from the database
	var articleData gin.H
	var articleTitle string

	// Get the real article from the database directly
	if s.db != nil {
		// Get article by slug directly from database
		var article models.Article
		query := `
			SELECT id, title, slug, content, excerpt, author_id, category_id, status, 
			       published_at, created_at, updated_at, view_count, like_count, dislike_count,
			       meta_title, meta_description, canonical_url, schema_type, featured_image_id
			FROM articles 
			WHERE slug = $1 AND status = 'published'
		`

		var publishedAt, createdAt, updatedAt time.Time
		var featuredImageID sql.NullInt64

		err := s.db.DB.QueryRow(query, slug).Scan(
			&article.ID, &article.Title, &article.Slug, &article.Content, &article.Excerpt,
			&article.AuthorID, &article.CategoryID, &article.Status,
			&publishedAt, &createdAt, &updatedAt,
			&article.ViewCount, &article.LikeCount, &article.DislikeCount,
			&article.MetaTitle, &article.MetaDescription, &article.CanonicalURL, &article.SchemaType,
			&featuredImageID,
		)

		// Set time fields
		if err == nil {
			article.PublishedAt = &publishedAt
			article.CreatedAt = createdAt
			article.UpdatedAt = updatedAt

			// Set featured image ID if available
			if featuredImageID.Valid {
				val := uint64(featuredImageID.Int64)
				article.FeaturedImageID = &val
			}
		}

		if err != nil {
			// Article not found, show 404
			data := s.createBaseTemplateData(c)
			data["Title"] = "Article Not Found"
			data["Content"] = "<div class='error-message'><h2>Article Not Found</h2><p>The article you're looking for doesn't exist or may have been moved.</p><a href='/'>← Back to Home</a></div>"

			if s.templateEngine != nil {
				if html, err := s.templateEngine.Render("homepage", data); err == nil {
					c.Header("Content-Type", "text/html; charset=utf-8")
					c.String(http.StatusNotFound, html)
					return
				}
			}
			c.String(http.StatusNotFound, "Article not found: "+slug)
			return
		}

		// Debug: Log the article data
		log.Printf("Article data: ID=%d, ViewCount=%d, LikeCount=%d, DislikeCount=%d",
			article.ID, article.ViewCount, article.LikeCount, article.DislikeCount)

		// Convert article to template data using actual model fields
		articleData = gin.H{
			"ID":           article.ID,
			"Title":        article.Title,
			"Slug":         article.Slug,
			"Content":      article.Content,
			"Excerpt":      article.Excerpt,
			"PublishedAt":  article.PublishedAt,
			"ViewCount":    article.ViewCount,
			"LikeCount":    article.LikeCount,
			"DislikeCount": article.DislikeCount,
			"CommentCount": 0, // TODO: Implement comment system
			"ReadTime":     calculateReadTime(article.Content),
			// Add SEO fields from individual columns
			"MetaTitle":       article.MetaTitle,
			"MetaDescription": article.MetaDescription,
			"CanonicalURL":    article.CanonicalURL,
			"SchemaType":      article.SchemaType,
		}

		// Get featured image if available (only local paths starting with /uploads/)
		if article.FeaturedImageID != nil && *article.FeaturedImageID > 0 {
			if s.db != nil {
				var imageURL sql.NullString
				query := "SELECT CASE WHEN original_url LIKE '/uploads/%' THEN original_url ELSE NULL END FROM images WHERE id = $1"
				err := s.db.DB.QueryRow(query, *article.FeaturedImageID).Scan(&imageURL)
				if err == nil && imageURL.Valid && imageURL.String != "" {
					articleData["FeaturedImage"] = imageURL.String
				}
			}
		}

		// Increment view count
		if s.db != nil {
			go func() {
				updateQuery := "UPDATE articles SET view_count = view_count + 1 WHERE id = $1"
				_, err := s.db.DB.Exec(updateQuery, article.ID)
				if err != nil {
					log.Printf("Error incrementing view count for article %d: %v", article.ID, err)
				}
			}()
		}

		// Add tags if available
		if len(article.Tags) > 0 {
			tags := make([]gin.H, len(article.Tags))
			for i, tag := range article.Tags {
				tags[i] = gin.H{
					"Name": tag.Name,
					"Slug": tag.Slug,
				}
			}
			articleData["Tags"] = tags
		}

		// TODO: Get author info from user service
		articleData["Author"] = gin.H{
			"FirstName": "Article",
			"LastName":  "Author",
			"Bio":       "Content Creator",
		}

		// Get category info and add it to the article data
		// Check if article has multiple categories loaded
		if len(article.Categories) > 0 {
			categories := make([]gin.H, len(article.Categories))
			for i, category := range article.Categories {
				categories[i] = gin.H{
					"ID":          category.ID,
					"Name":        category.Name,
					"Slug":        category.Slug,
					"Description": category.Description,
				}
			}
			articleData["Categories"] = categories
			articleData["Category"] = categories[0] // First category for backward compatibility
		} else if article.CategoryID > 0 {
			// Fallback to single category lookup
			categoryData, err := s.getCategoryByID(article.CategoryID)
			if err == nil {
				articleData["Category"] = categoryData
				articleData["Categories"] = []gin.H{categoryData}
			} else {
				fallbackCategory := gin.H{"Name": "General", "Slug": "general"}
				articleData["Category"] = fallbackCategory
				articleData["Categories"] = []gin.H{fallbackCategory}
			}
		} else {
			fallbackCategory := gin.H{"Name": "General", "Slug": "general"}
			articleData["Category"] = fallbackCategory
			articleData["Categories"] = []gin.H{fallbackCategory}
		}

		// Get tags for the article and add them to article data
		tags, err := s.getArticleTags(article.ID)
		if err == nil && len(tags) > 0 {
			articleData["Tags"] = tags
		} else {
			// Provide empty tags array if none found
			articleData["Tags"] = []gin.H{}
		}

		articleTitle = article.Title
	} else {
		// Fallback to sample data if no article service
		articleData = gin.H{
			"Title":   "Sample Article: " + slug,
			"Slug":    slug,
			"Content": "<p>This is sample article content for: " + slug + "</p><p>Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.</p>",
			"Excerpt": "This is a sample article excerpt for " + slug,
			"Author": gin.H{
				"FirstName": "John",
				"LastName":  "Doe",
				"Bio":       "Senior Writer",
			},
			"Category": gin.H{
				"Name": "Technology",
				"Slug": "technology",
			},
			"Tags": []gin.H{
				{"Name": "Sample", "Slug": "sample"},
				{"Name": "Test", "Slug": "test"},
			},
			"PublishedAt": time.Now().AddDate(0, 0, -1),
			"Views":       250,
			"ReadTime":    3,
		}
		articleTitle = "Sample Article: " + slug
	}

	// Create template data
	data := s.createBaseTemplateData(c)
	data["Title"] = articleTitle
	data["Article"] = articleData

	// Add HeroImage to root level for base template preload tag
	if featuredImage, ok := articleData["FeaturedImage"].(string); ok && featuredImage != "" {
		data["HeroImage"] = featuredImage
	}

	// Add SEO fields to root level for base template (catch-all route)
	if metaTitle, ok := articleData["MetaTitle"].(string); ok && metaTitle != "" {
		data["SEOTitle"] = metaTitle
	} else {
		data["SEOTitle"] = articleTitle
	}

	if metaDesc, ok := articleData["MetaDescription"].(string); ok && metaDesc != "" {
		data["SEODescription"] = metaDesc
	} else if excerpt, ok := articleData["Excerpt"].(string); ok {
		data["SEODescription"] = excerpt
	}

	if canonicalURL, ok := articleData["CanonicalURL"].(string); ok && canonicalURL != "" {
		data["CanonicalURL"] = canonicalURL
	}

	// Add related articles (could be enhanced to get real related articles)
	data["RelatedArticles"] = []gin.H{
		{"Title": "Related Article 1", "Slug": "related-1", "Excerpt": "Related content"},
		{"Title": "Related Article 2", "Slug": "related-2", "Excerpt": "More related content"},
	}

	// Debug: Log the article data being passed to template
	log.Printf("Article data for template: Title=%s, MetaTitle=%s, Tags=%v, Category=%v",
		articleData["Title"], articleData["MetaTitle"], articleData["Tags"], articleData["Category"])
	log.Printf("Root template data: Title=%s, SEOTitle=%s, SEODescription=%s",
		data["Title"], data["SEOTitle"], data["SEODescription"])

	if s.templateEngine != nil {
		if html, err := s.templateEngine.Render("article", data); err == nil {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, html)
			return
		} else {
			log.Printf("Template render error for 'article': %v", err)
		}
	} else {
		log.Printf("Template engine is nil")
	}

	// Fallback to simple response
	c.String(http.StatusOK, "Article: "+slug+" (Template not available)")
}

// setupAdminFrontendRoutes sets up admin panel frontend routes
func (s *Server) setupAdminFrontendRoutes() {
	// Admin login page (no auth required)
	s.router.GET("/admin/login", s.handleAdminLogin)

	// Admin dashboard and pages (require authentication)
	adminGroup := s.router.Group("/admin")
	adminGroup.Use(s.requireAuth) // Add authentication middleware
	{
		adminGroup.GET("/", s.handleAdminDashboard)
		adminGroup.GET("/dashboard", s.handleAdminDashboard)
		adminGroup.GET("/analytics", s.handleAdminAnalytics)
		adminGroup.GET("/users", s.handleAdminUsers)
		adminGroup.GET("/users/create", s.renderCreateUser)
		adminGroup.GET("/users/list", s.renderUserList)
		adminGroup.GET("/users/edit/:id", s.renderEditUser)
		adminGroup.GET("/users/roles", s.renderManageRoles)
		adminGroup.GET("/users/export", s.renderExportUsers)
		adminGroup.GET("/content", s.handleAdminContent)
		adminGroup.GET("/content/create", s.renderCreateArticle)
		adminGroup.GET("/content/edit/:id", s.renderEditArticle)
		adminGroup.GET("/content/articles", s.renderManageArticles)
		adminGroup.GET("/content/trash", s.renderRecycleBin)
		adminGroup.GET("/content/categories", s.renderManageCategories)
		adminGroup.GET("/content/tags", s.renderManageTags)
		adminGroup.GET("/content/media", s.renderMediaLibrary)
		adminGroup.GET("/content-ingestion", s.handleAdminContentIngestion)
		adminGroup.GET("/autolinking", s.handleAdminAutoLinking)
		adminGroup.GET("/keyword-banks", s.handleAdminKeywordBanks)
		adminGroup.GET("/comments", s.handleAdminComments)
		adminGroup.GET("/comments/analytics", s.handleAdminCommentsAnalytics)
		adminGroup.GET("/comments/settings", s.handleAdminCommentsSettings)
		adminGroup.GET("/settings", s.handleAdminSettings)
		adminGroup.GET("/settings/general", s.handleAdminSettingsGeneral)
		adminGroup.GET("/settings/performance", s.handleAdminSettingsPerformance)
		adminGroup.GET("/settings/security", s.handleAdminSettingsSecurity)
		adminGroup.GET("/settings/backup", s.handleAdminSettingsBackup)
		adminGroup.GET("/settings/email", s.handleAdminSettingsEmail)
		adminGroup.GET("/settings/api", s.handleAdminSettingsAPI)
		adminGroup.GET("/system", s.handleAdminSystem)
		adminGroup.GET("/logs", s.handleAdminLogs)

		// Marketing & Engagement
		adminGroup.GET("/ads/campaigns", s.handleAdminAdsCampaigns)
		adminGroup.GET("/ads/slots", s.handleAdminAdsSlots)
		adminGroup.GET("/ads/creatives", s.handleAdminAdsCreatives)
		adminGroup.GET("/ads/targeting", s.handleAdminAdsTargeting)
		adminGroup.GET("/ads/analytics", s.handleAdminAdsAnalytics)
		adminGroup.GET("/push/send", s.handleAdminPushSend)
		adminGroup.GET("/push/templates", s.handleAdminPushTemplates)
		adminGroup.GET("/push/subscribers", s.handleAdminPushSubscribers)
		adminGroup.GET("/push/analytics", s.handleAdminPushAnalytics)
		adminGroup.GET("/social/accounts", s.handleAdminSocialAccounts)
		adminGroup.GET("/social/auto-publish", s.handleAdminSocialAutoPublish)
		adminGroup.GET("/social/scheduled", s.handleAdminSocialScheduled)
		adminGroup.GET("/social/analytics", s.handleAdminSocialAnalytics)
		adminGroup.GET("/newsletter", s.handleAdminNewsletter)

		// Analytics & SEO
		adminGroup.GET("/analytics/traffic", s.handleAdminAnalyticsTraffic)
		adminGroup.GET("/analytics/content", s.handleAdminAnalyticsContent)
		adminGroup.GET("/analytics/audience", s.handleAdminAnalyticsAudience)
		adminGroup.GET("/analytics/realtime", s.handleAdminAnalyticsRealtime)
		adminGroup.GET("/seo", s.handleAdminSEOSettings)
		adminGroup.GET("/seo/overview", s.handleAdminSEOOverview)
		adminGroup.GET("/seo/sitemap", s.handleAdminSEOSitemap)
		adminGroup.GET("/seo/google-news", s.handleAdminSEOGoogleNews)
		adminGroup.GET("/seo/schema", s.handleAdminSEOSchema)
		adminGroup.GET("/seo/redirects", s.handleAdminSEORedirects)

		// Appearance
		adminGroup.GET("/themes", s.handleAdminThemes)
		adminGroup.GET("/widgets", s.handleAdminWidgets)
		adminGroup.GET("/menus", s.handleAdminMenus)

		// Distribution
		adminGroup.GET("/rss", s.handleAdminRSS)
		adminGroup.GET("/cdn/config", s.handleAdminCDNConfig)
		adminGroup.GET("/cdn/purge", s.handleAdminCDNPurge)
		adminGroup.GET("/cdn/stats", s.handleAdminCDNStats)

		// Backup & Restore
		adminGroup.GET("/backup/create", s.handleAdminBackupCreate)
		adminGroup.GET("/backup/list", s.handleAdminBackupList)
		adminGroup.GET("/backup/restore", s.handleAdminBackupRestore)
		adminGroup.GET("/backup/schedule", s.handleAdminBackupSchedule)

		// Integrations - Third Party
		adminGroup.GET("/integrations/google-analytics", s.handleAdminGoogleAnalytics)
		adminGroup.GET("/integrations/tag-manager", s.handleAdminTagManager)
		adminGroup.GET("/integrations/search-console", s.handleAdminSearchConsole)
		adminGroup.GET("/integrations/adsense", s.handleAdminAdSense)
		adminGroup.GET("/integrations/facebook-pixel", s.handleAdminFacebookPixel)

		// Code Injection
		adminGroup.GET("/code/header", s.handleAdminHeaderScripts)
		adminGroup.GET("/code/footer", s.handleAdminFooterScripts)
		adminGroup.GET("/code/custom-css", s.handleAdminCustomCSS)
		adminGroup.GET("/code/custom-js", s.handleAdminCustomJS)
	}
}

// Admin frontend handlers
func (s *Server) handleAdminLogin(c *gin.Context) {
	// Serve the enhanced login template file
	c.File("web/templates/admin/login-enhanced.html")
}

func (s *Server) handleAdminDashboard(c *gin.Context) {
	s.renderAdminDashboard(c)
}

func (s *Server) handleAdminAnalytics(c *gin.Context) {
	s.renderAdminAnalytics(c)
}

func (s *Server) handleAdminUsers(c *gin.Context) {
	s.renderAdminUsers(c)
}

func (s *Server) handleAdminContent(c *gin.Context) {
	s.renderAdminContent(c)
}

func (s *Server) handleAdminSettings(c *gin.Context) {
	s.renderAdminSettings(c)
}

func (s *Server) handleAdminSystem(c *gin.Context) {
	s.renderAdminSystem(c)
}

func (s *Server) getCurrentAdminUser(c *gin.Context) gin.H {
	// Mock admin user data for now
	return gin.H{
		"id":       1,
		"username": "admin",
		"email":    "admin@example.com",
		"role":     "admin",
	}
}

// createBaseTemplateData creates the base template data structure that all templates expect
func (s *Server) createBaseTemplateData(c *gin.Context) gin.H {
	// Get active theme settings
	themeConfig := s.getActiveThemeConfig()

	// Use theme branding or fallback to config
	siteName := s.config.App.Name
	siteDescription := "High-performance multilingual news website"
	logoURL := ""
	faviconURL := "/static/favicon.ico"
	showSiteName := true
	headerStyle := "sticky"

	if themeConfig != nil {
		if branding, ok := themeConfig["branding"].(map[string]interface{}); ok {
			if name, ok := branding["site_name"].(string); ok && name != "" {
				siteName = name
			}
			if desc, ok := branding["site_description"].(string); ok && desc != "" {
				siteDescription = desc
			}
			if logo, ok := branding["logo_url"].(string); ok && logo != "" {
				logoURL = logo
			}
			if favicon, ok := branding["favicon_url"].(string); ok && favicon != "" {
				faviconURL = favicon
			}
			if show, ok := branding["show_site_name"].(bool); ok {
				showSiteName = show
			}
		}
		if layout, ok := themeConfig["layout"].(map[string]interface{}); ok {
			if style, ok := layout["header_style"].(string); ok && style != "" {
				headerStyle = style
			}
		}
	}

	data := gin.H{
		"SiteName":          siteName,
		"SiteDescription":   siteDescription,
		"LogoURL":           logoURL,
		"FaviconURL":        faviconURL,
		"ShowSiteName":      showSiteName,
		"HeaderStyle":       headerStyle,
		"ThemeConfig":       themeConfig,
		"LanguageCode":      "en",
		"LanguageDirection": "ltr",
		"ThemeMode":         "auto",
		"CurrentYear":       2024,
		"Navigation":        s.getNavigationForLanguage("en"),
		"IsAuthenticated":   false,
		"OGType":            "website",
		"TwitterCard":       "summary_large_image",
	}

	// Fetch breaking news articles (articles with "breaking" tag)
	breakingNews := s.getBreakingNewsArticles()
	data["BreakingNews"] = breakingNews
	data["HasBreakingNews"] = len(breakingNews) > 0

	return data
}

// getNavigationForLanguage returns navigation items translated for the given language
func (s *Server) getNavigationForLanguage(lang string) []gin.H {
	// Translation maps for navigation items
	navTranslations := map[string]map[string]string{
		"en": {"home": "Home", "latest": "Latest", "trending": "Trending", "categories": "Categories", "tags": "Tags", "about": "About", "contact": "Contact"},
		"de": {"home": "Startseite", "latest": "Neueste", "trending": "Beliebt", "categories": "Kategorien", "tags": "Schlagwörter", "about": "Über uns", "contact": "Kontakt"},
		"fr": {"home": "Accueil", "latest": "Récent", "trending": "Tendances", "categories": "Catégories", "tags": "Étiquettes", "about": "À propos", "contact": "Contact"},
		"es": {"home": "Inicio", "latest": "Reciente", "trending": "Tendencias", "categories": "Categorías", "tags": "Etiquetas", "about": "Acerca de", "contact": "Contacto"},
		"ar": {"home": "الرئيسية", "latest": "الأحدث", "trending": "الأكثر رواجاً", "categories": "التصنيفات", "tags": "الوسوم", "about": "من نحن", "contact": "اتصل بنا"},
	}

	trans := navTranslations[lang]
	if trans == nil {
		trans = navTranslations["en"]
	}

	prefix := "/" + lang

	return []gin.H{
		{"Name": trans["home"], "URL": prefix + "/", "Active": false},
		{"Name": trans["latest"], "URL": prefix + "/latest", "Active": false},
		{"Name": trans["trending"], "URL": prefix + "/trending", "Active": false},
		{"Name": trans["categories"], "URL": prefix + "/categories", "Active": false},
		{"Name": trans["tags"], "URL": prefix + "/tags", "Active": false},
		{"Name": trans["about"], "URL": prefix + "/about", "Active": false},
		{"Name": trans["contact"], "URL": prefix + "/contact", "Active": false},
	}
}

// getActiveThemeConfig fetches the active theme configuration from the database
func (s *Server) getActiveThemeConfig() map[string]interface{} {
	if s.db == nil {
		log.Println("DEBUG getActiveThemeConfig: db is nil")
		return nil
	}

	var configJSON []byte
	err := s.db.QueryRow("SELECT config FROM themes WHERE is_active = true LIMIT 1").Scan(&configJSON)
	if err != nil {
		log.Printf("DEBUG getActiveThemeConfig: query error: %v", err)
		return nil
	}

	var config map[string]interface{}
	if err := json.Unmarshal(configJSON, &config); err != nil {
		log.Printf("DEBUG getActiveThemeConfig: unmarshal error: %v", err)
		return nil
	}

	// Log branding info
	if branding, ok := config["branding"].(map[string]interface{}); ok {
		log.Printf("DEBUG getActiveThemeConfig: branding found - site_name=%v, logo_url=%v", branding["site_name"], branding["logo_url"])
	}

	return config
}

// getBreakingNewsArticles fetches articles tagged with "breaking-news" for the ticker
func (s *Server) getBreakingNewsArticles() []gin.H {
	var breakingNews []gin.H

	// Try to get the "breaking-news" tag
	if s.tagService == nil || s.db == nil {
		return breakingNews
	}

	// Get the breaking news tag by slug
	tag, err := s.tagService.GetBySlug("breaking-news")
	if err != nil || tag == nil {
		// Tag doesn't exist yet, return empty
		return breakingNews
	}

	// Query articles with the breaking-news tag (limit to 10 most recent)
	query := `
		SELECT a.id, a.title, a.slug, a.excerpt
		FROM articles a
		INNER JOIN article_tags at ON a.id = at.article_id
		WHERE at.tag_id = $1 AND a.status = 'published'
		ORDER BY a.published_at DESC
		LIMIT 10`

	rows, err := s.db.Query(query, tag.ID)
	if err != nil {
		log.Printf("Error fetching breaking news: %v", err)
		return breakingNews
	}
	defer rows.Close()

	for rows.Next() {
		var id uint64
		var title, slug, excerpt string
		if err := rows.Scan(&id, &title, &slug, &excerpt); err != nil {
			continue
		}
		breakingNews = append(breakingNews, gin.H{
			"ID":      id,
			"Title":   title,
			"Slug":    slug,
			"Excerpt": excerpt,
		})
	}

	return breakingNews
}

//

func (s *Server) handleAdminContentCreate(c *gin.Context) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Create Article - Admin Panel</title>
    <link rel="stylesheet" href="/static/css/admin.css">
</head>
<body>
    <div class="dashboard-container">
        <h1>Create New Article</h1>
        <p>Article creation functionality coming soon...</p>
        <a href="/admin/content" class="action-button">← Back to Content</a>
    </div>
</body>
</html>`
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (s *Server) handleAdminContentCategories(c *gin.Context) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Manage Categories - Admin Panel</title>
    <link rel="stylesheet" href="/static/css/admin.css">
</head>
<body>
    <div class="dashboard-container">
        <h1>Manage Categories</h1>
        <p>Category management functionality coming soon...</p>
        <a href="/admin/content" class="action-button">← Back to Content</a>
    </div>
</body>
</html>`
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (s *Server) handleAdminContentTags(c *gin.Context) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Manage Tags - Admin Panel</title>
    <link rel="stylesheet" href="/static/css/admin.css">
</head>
<body>
    <div class="dashboard-container">
        <h1>Manage Tags</h1>
        <p>Tag management functionality coming soon...</p>
        <a href="/admin/content" class="action-button">← Back to Content</a>
    </div>
</body>
</html>`
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (s *Server) handleAdminContentMedia(c *gin.Context) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Media Library - Admin Panel</title>
    <link rel="stylesheet" href="/static/css/admin.css">
</head>
<body>
    <div class="dashboard-container">
        <h1>Media Library</h1>
        <p>Media management functionality coming soon...</p>
        <a href="/admin/content" class="action-button">← Back to Content</a>
    </div>
</body>
</html>`
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// Users management handlers
func (s *Server) handleAdminUsersCreate(c *gin.Context) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Add New User - Admin Panel</title>
    <link rel="stylesheet" href="/static/css/admin.css">
</head>
<body>
    <div class="dashboard-container">
        <h1>Add New User</h1>
        <p>User creation functionality coming soon...</p>
        <a href="/admin/users" class="action-button">← Back to Users</a>
    </div>
</body>
</html>`
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (s *Server) handleAdminUsersRoles(c *gin.Context) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Manage Roles - Admin Panel</title>
    <link rel="stylesheet" href="/static/css/admin.css">
</head>
<body>
    <div class="dashboard-container">
        <h1>Manage User Roles</h1>
        <p>Role management functionality coming soon...</p>
        <a href="/admin/users" class="action-button">← Back to Users</a>
    </div>
</body>
</html>`
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (s *Server) handleAdminUsersPermissions(c *gin.Context) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>User Permissions - Admin Panel</title>
    <link rel="stylesheet" href="/static/css/admin.css">
</head>
<body>
    <div class="dashboard-container">
        <h1>User Permissions</h1>
        <p>Permission management functionality coming soon...</p>
        <a href="/admin/users" class="action-button">← Back to Users</a>
    </div>
</body>
</html>`
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (s *Server) handleAdminUsersExport(c *gin.Context) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Export Users - Admin Panel</title>
    <link rel="stylesheet" href="/static/css/admin.css">
</head>
<body>
    <div class="dashboard-container">
        <h1>Export Users</h1>
        <p>User export functionality coming soon...</p>
        <a href="/admin/users" class="action-button">← Back to Users</a>
    </div>
</body>
</html>`
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// Settings management handlers
func (s *Server) handleAdminSettingsGeneral(c *gin.Context) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>General Settings - Admin Panel</title>
    <link rel="stylesheet" href="/static/css/admin.css">
</head>
<body>
    <div class="dashboard-container">
        <h1>General Settings</h1>
        <p>General settings functionality coming soon...</p>
        <a href="/admin/settings" class="action-button">← Back to Settings</a>
    </div>
</body>
</html>`
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (s *Server) handleAdminSettingsPerformance(c *gin.Context) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Performance Settings - Admin Panel</title>
    <link rel="stylesheet" href="/static/css/admin.css">
</head>
<body>
    <div class="dashboard-container">
        <h1>Performance Settings</h1>
        <p>Performance configuration functionality coming soon...</p>
        <a href="/admin/settings" class="action-button">← Back to Settings</a>
    </div>
</body>
</html>`
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (s *Server) handleAdminSettingsSecurity(c *gin.Context) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Security Settings - Admin Panel</title>
    <link rel="stylesheet" href="/static/css/admin.css">
</head>
<body>
    <div class="dashboard-container">
        <h1>Security Settings</h1>
        <p>Security configuration functionality coming soon...</p>
        <a href="/admin/settings" class="action-button">← Back to Settings</a>
    </div>
</body>
</html>`
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (s *Server) handleAdminSettingsBackup(c *gin.Context) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Backup Settings - Admin Panel</title>
    <link rel="stylesheet" href="/static/css/admin.css">
</head>
<body>
    <div class="dashboard-container">
        <h1>Backup Settings</h1>
        <p>Backup configuration functionality coming soon...</p>
        <a href="/admin/settings" class="action-button">← Back to Settings</a>
    </div>
</body>
</html>`
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// calculateReadTime estimates reading time based on content length
func calculateReadTime(content string) int {
	// Average reading speed is about 200 words per minute
	wordCount := len(strings.Fields(content))
	readTime := wordCount / 200
	if readTime < 1 {
		readTime = 1
	}
	return readTime
}

// getCategoryByID fetches category information by ID
func (s *Server) getCategoryByID(categoryID uint64) (gin.H, error) {
	if s.db == nil {
		return gin.H{"Name": "General", "Slug": "general"}, nil
	}

	var name, slug string
	query := "SELECT name, slug FROM categories WHERE id = $1"
	err := s.db.DB.QueryRow(query, categoryID).Scan(&name, &slug)
	if err != nil {
		return gin.H{"Name": "General", "Slug": "general"}, err
	}

	return gin.H{
		"Name": name,
		"Slug": slug,
	}, nil
}

// getCategoryByIDForLanguage fetches category information by ID, resolving to the correct language version
// If the category has a translation_group_id, it finds the version in the requested language
func (s *Server) getCategoryByIDForLanguage(categoryID uint64, lang string) (gin.H, error) {
	if s.db == nil {
		return gin.H{"Name": "General", "Slug": "general"}, nil
	}

	// First, get the translation_group_id for this category
	var translationGroupID uint64
	var originalName, originalSlug string
	query := `SELECT COALESCE(translation_group_id, id), name, slug FROM categories WHERE id = $1`
	err := s.db.DB.QueryRow(query, categoryID).Scan(&translationGroupID, &originalName, &originalSlug)
	if err != nil {
		return gin.H{"Name": "General", "Slug": "general"}, err
	}

	// Now find the category in the requested language
	var name, slug string
	langQuery := `
		SELECT name, slug FROM categories 
		WHERE (translation_group_id = $1 OR id = $1) AND language_code = $2
		LIMIT 1
	`
	err = s.db.DB.QueryRow(langQuery, translationGroupID, lang).Scan(&name, &slug)
	if err != nil {
		// If no translation exists, return the original
		return gin.H{
			"Name": originalName,
			"Slug": originalSlug,
		}, nil
	}

	return gin.H{
		"Name": name,
		"Slug": slug,
	}, nil
}

// getArticleTagsForLanguage fetches tags for an article, resolving to the correct language versions
func (s *Server) getArticleTagsForLanguage(articleID uint64, lang string) ([]gin.H, error) {
	if s.db == nil {
		return nil, nil
	}

	// Get tag IDs for this article
	query := `SELECT tag_id FROM article_tags WHERE article_id = $1`
	rows, err := s.db.DB.Query(query, articleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tagIDs []uint64
	for rows.Next() {
		var tagID uint64
		if err := rows.Scan(&tagID); err != nil {
			continue
		}
		tagIDs = append(tagIDs, tagID)
	}

	if len(tagIDs) == 0 {
		return nil, nil
	}

	// For each tag, find the correct language version
	var tags []gin.H
	for _, tagID := range tagIDs {
		// Get translation_group_id
		var translationGroupID uint64
		var originalName, originalSlug, originalColor string
		groupQuery := `SELECT COALESCE(translation_group_id, id), name, slug, COALESCE(color, '#3b82f6') FROM tags WHERE id = $1`
		err := s.db.DB.QueryRow(groupQuery, tagID).Scan(&translationGroupID, &originalName, &originalSlug, &originalColor)
		if err != nil {
			continue
		}

		// Find the tag in the requested language
		var name, slug, color string
		langQuery := `
			SELECT name, slug, COALESCE(color, '#3b82f6') FROM tags 
			WHERE (translation_group_id = $1 OR id = $1) AND language_code = $2
			LIMIT 1
		`
		err = s.db.DB.QueryRow(langQuery, translationGroupID, lang).Scan(&name, &slug, &color)
		if err != nil {
			// Use original if no translation
			name, slug, color = originalName, originalSlug, originalColor
		}

		tags = append(tags, gin.H{
			"Name":  name,
			"Slug":  slug,
			"Color": color,
		})
	}

	return tags, nil
}

// getAvailableLanguagesForArticle returns language info for the language switcher, only for available translations
func (s *Server) getAvailableLanguagesForArticle(translations []models.Article, slug string) []map[string]interface{} {
	languageInfo := map[string]struct {
		Name       string
		NativeName string
		Direction  string
	}{
		"en": {"English", "English", "ltr"},
		"de": {"German", "Deutsch", "ltr"},
		"fr": {"French", "Français", "ltr"},
		"es": {"Spanish", "Español", "ltr"},
		"ar": {"Arabic", "العربية", "rtl"},
	}

	var result []map[string]interface{}
	for _, trans := range translations {
		info, ok := languageInfo[trans.LanguageCode]
		if !ok {
			continue
		}

		url := "/" + trans.LanguageCode + "/article/" + slug
		result = append(result, map[string]interface{}{
			"Code":       trans.LanguageCode,
			"Name":       info.Name,
			"NativeName": info.NativeName,
			"Direction":  info.Direction,
			"URL":        url,
		})
	}

	return result
}

// buildResponsiveImageData builds ResponsiveImageData from an image ID for dynamic rendering
func (s *Server) buildResponsiveImageData(imageID uint64, altText string) *services.ResponsiveImageData {
	if s.mediaService == nil || imageID == 0 {
		return nil
	}

	variants, err := s.mediaService.GetImageVariants(imageID)
	if err != nil || len(variants) == 0 {
		return nil
	}

	data := &services.ResponsiveImageData{
		AltText:     altText,
		HasVariants: true,
	}

	// Organize variants by size and format
	for _, v := range variants {
		switch v.Size {
		case models.ImageSizeThumbnail:
			if v.Format == models.ImageFormatWebP {
				data.ThumbnailWebP = v.URL
			} else if v.Format == models.ImageFormatJPEG {
				data.ThumbnailJPEG = v.URL
			}
		case models.ImageSizeSmall:
			if v.Format == models.ImageFormatWebP {
				data.SmallWebP = v.URL
			} else if v.Format == models.ImageFormatJPEG {
				data.SmallJPEG = v.URL
			}
		case models.ImageSizeMedium:
			if v.Format == models.ImageFormatWebP {
				data.MediumWebP = v.URL
			} else if v.Format == models.ImageFormatJPEG {
				data.MediumJPEG = v.URL
			}
		case models.ImageSizeLarge:
			if v.Format == models.ImageFormatWebP {
				data.LargeWebP = v.URL
				data.Width = v.Width
				data.Height = v.Height
			} else if v.Format == models.ImageFormatJPEG {
				data.LargeJPEG = v.URL
				if data.Width == 0 {
					data.Width = v.Width
					data.Height = v.Height
				}
			}
		}
	}

	// Check if we have at least some variants
	if data.SmallWebP == "" && data.SmallJPEG == "" && data.MediumWebP == "" && data.MediumJPEG == "" {
		data.HasVariants = false
	}

	return data
}

// getArticleTags fetches tags for an article
func (s *Server) getArticleTags(articleID uint64) ([]gin.H, error) {
	if s.db == nil {
		return []gin.H{}, nil
	}

	query := `
		SELECT t.name, t.slug 
		FROM tags t 
		JOIN article_tags at ON t.id = at.tag_id 
		WHERE at.article_id = $1
	`

	rows, err := s.db.DB.Query(query, articleID)
	if err != nil {
		return []gin.H{}, err
	}
	defer rows.Close()

	var tags []gin.H
	for rows.Next() {
		var name, slug string
		if err := rows.Scan(&name, &slug); err != nil {
			continue
		}
		tags = append(tags, gin.H{
			"Name": name,
			"Slug": slug,
		})
	}

	return tags, nil
}

// getCategoryArticleCount gets the number of published articles in a category
func (s *Server) getCategoryArticleCount(categoryID uint64) int {
	if s.db == nil {
		return 0
	}

	var count int
	query := "SELECT COUNT(*) FROM articles WHERE category_id = $1 AND status = 'published'"
	err := s.db.DB.QueryRow(query, categoryID).Scan(&count)
	if err != nil {
		log.Printf("Error getting article count for category %d: %v", categoryID, err)
		return 0
	}

	return count
}

// getTagArticleCount gets the number of published articles with a specific tag
func (s *Server) getTagArticleCount(tagID uint64) int {
	if s.db == nil {
		return 0
	}

	var count int
	query := `
		SELECT COUNT(DISTINCT a.id) 
		FROM articles a 
		JOIN article_tags at ON a.id = at.article_id 
		WHERE at.tag_id = $1 AND a.status = 'published'
	`
	err := s.db.DB.QueryRow(query, tagID).Scan(&count)
	if err != nil {
		log.Printf("Error getting article count for tag %d: %v", tagID, err)
		return 0
	}

	return count
}

// Helper function to format time ago
func formatTimeAgo(t *time.Time) string {
	if t == nil {
		return "Unknown"
	}

	now := time.Now()
	diff := now.Sub(*t)

	if diff < time.Minute {
		return "Just now"
	} else if diff < time.Hour {
		minutes := int(diff.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	} else {
		return t.Format("Jan 2, 2006")
	}
}

// Article engagement handlers are now handled by the API router in article_handlers.go

// ============================================================================
// MULTILINGUAL HANDLERS
// ============================================================================

// handleMultilingualHomepage handles the homepage for each language
func (s *Server) handleMultilingualHomepage(c *gin.Context) {
	// Extract language from URL path
	lang := c.Request.URL.Path[1:3] // Get "en", "de", etc.
	if len(lang) != 2 {
		lang = "en"
	}

	// Create template data with multilingual support
	data := s.createBaseTemplateData(c)
	
	// Get site name and description from theme (already set in createBaseTemplateData)
	siteName := data["SiteName"].(string)
	siteDescription := data["SiteDescription"].(string)
	
	// Set page title based on language
	homeTitles := map[string]string{
		"en": "Home",
		"de": "Startseite",
		"fr": "Accueil",
		"es": "Inicio",
		"ar": "الرئيسية",
	}
	homeTitle := homeTitles[lang]
	if homeTitle == "" {
		homeTitle = "Home"
	}
	data["Title"] = homeTitle + " | " + siteName
	data["SEOTitle"] = homeTitle + " | " + siteName
	data["Description"] = siteDescription
	data["SEODescription"] = siteDescription
	data["PageType"] = "homepage"

	// Add multilingual data
	s.addMultilingualData(data, lang, "/")

	// Get articles for this language
	if s.articleService != nil {
		filters := services.ArticleFilters{
			Status:       "published",
			LanguageCode: lang,
		}
		articles, _, err := s.articleService.List(c.Request.Context(), 10, 0, filters, "published_at", "DESC")
		if err == nil {
			articleData := make([]gin.H, len(articles))
			for i, article := range articles {
				categoryName := "General"
				if article.CategoryID > 0 {
					if categoryData, err := s.getCategoryByID(article.CategoryID); err == nil {
						if name, ok := categoryData["Name"].(string); ok {
							categoryName = name
						}
					}
				}

				var imageData *services.ResponsiveImageData
				if article.FeaturedImageID != nil && *article.FeaturedImageID > 0 && s.mediaService != nil {
					imageData = s.buildResponsiveImageData(*article.FeaturedImageID, article.Title)
				}

				articleData[i] = gin.H{
					"ID":            article.ID,
					"Title":         article.Title,
					"Slug":          article.Slug,
					"Excerpt":       article.Excerpt,
					"Author":        "Author",
					"TimeAgo":       formatTimeAgo(article.PublishedAt),
					"ViewCount":     article.ViewCount,
					"Category":      categoryName,
					"FeaturedImage": article.FeaturedImage,
					"ImageData":     imageData,
					"URL":           fmt.Sprintf("/%s/article/%s", lang, article.Slug),
				}
			}
			data["Articles"] = articleData
		} else {
			data["Articles"] = []gin.H{}
		}
	} else {
		data["Articles"] = []gin.H{}
	}

	// Get categories
	if s.categoryService != nil {
		categories, err := s.categoryService.GetAll()
		if err == nil {
			categoryData := make([]gin.H, len(categories))
			for i, cat := range categories {
				categoryData[i] = gin.H{
					"ID":          cat.ID,
					"Name":        cat.Name,
					"Slug":        cat.Slug,
					"Description": cat.Description,
					"URL":         fmt.Sprintf("/%s/category/%s", lang, cat.Slug),
					"Count":       s.getCategoryArticleCount(cat.ID),
				}
			}
			data["Categories"] = categoryData
		}
	}

	// Get trending articles
	if s.articleService != nil {
		trendingArticles, err := s.articleService.GetTrending(c.Request.Context(), 5, 24)
		if err == nil && len(trendingArticles) > 0 {
			trendingData := make([]gin.H, len(trendingArticles))
			for i, article := range trendingArticles {
				trendingData[i] = gin.H{
					"ID":        article.ID,
					"Title":     article.Title,
					"Slug":      article.Slug,
					"ViewCount": article.ViewCount,
					"TimeAgo":   formatTimeAgo(article.PublishedAt),
					"URL":       fmt.Sprintf("/%s/article/%s", lang, article.Slug),
				}
			}
			data["TrendingArticles"] = trendingData
		} else {
			// Fallback: use most viewed articles
			filters := services.ArticleFilters{Status: "published"}
			popularArticles, _, err := s.articleService.List(c.Request.Context(), 5, 0, filters, "view_count", "DESC")
			if err == nil && len(popularArticles) > 0 {
				trendingData := make([]gin.H, len(popularArticles))
				for i, article := range popularArticles {
					trendingData[i] = gin.H{
						"ID":        article.ID,
						"Title":     article.Title,
						"Slug":      article.Slug,
						"ViewCount": article.ViewCount,
						"TimeAgo":   formatTimeAgo(article.PublishedAt),
						"URL":       fmt.Sprintf("/%s/article/%s", lang, article.Slug),
					}
				}
				data["TrendingArticles"] = trendingData
			} else {
				data["TrendingArticles"] = []gin.H{}
			}
		}
	} else {
		data["TrendingArticles"] = []gin.H{}
	}

	// Get popular tags
	if s.tagService != nil {
		tags, err := s.tagService.GetAll()
		if err == nil {
			type TagWithCount struct {
				Name  string
				Slug  string
				Count int
			}
			tagList := make([]TagWithCount, 0)
			for _, tag := range tags {
				articleCount := s.getTagArticleCount(tag.ID)
				if articleCount > 0 {
					tagList = append(tagList, TagWithCount{
						Name:  tag.Name,
						Slug:  tag.Slug,
						Count: articleCount,
					})
				}
			}
			// Sort by count
			for i := 0; i < len(tagList)-1; i++ {
				for j := i + 1; j < len(tagList); j++ {
					if tagList[j].Count > tagList[i].Count {
						tagList[i], tagList[j] = tagList[j], tagList[i]
					}
				}
			}
			// Limit to 8
			if len(tagList) > 8 {
				tagList = tagList[:8]
			}
			tagData := make([]gin.H, len(tagList))
			for i, tag := range tagList {
				tagData[i] = gin.H{
					"Name":  tag.Name,
					"Slug":  tag.Slug,
					"Count": tag.Count,
					"URL":   fmt.Sprintf("/%s/tag/%s", lang, tag.Slug),
				}
			}
			data["PopularTags"] = tagData
		} else {
			data["PopularTags"] = []gin.H{}
		}
	} else {
		data["PopularTags"] = []gin.H{}
	}

	// Render template
	if s.templateEngine != nil {
		html, err := s.templateEngine.Render("homepage", data)
		if err != nil {
			log.Printf("ERROR: Template render failed for homepage: %v", err)
		} else {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, html)
			return
		}
	} else {
		log.Printf("ERROR: templateEngine is nil")
	}

	c.String(http.StatusOK, "Homepage - "+lang)
}

// handleMultilingualArticle handles article pages for each language
func (s *Server) handleMultilingualArticle(c *gin.Context) {
	lang := c.Request.URL.Path[1:3]
	slug := c.Param("slug")

	if s.articleService == nil {
		c.String(http.StatusNotFound, "Article not found")
		return
	}

	article, err := s.articleService.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		data := s.createBaseTemplateData(c)
		data["Title"] = "Article Not Found"
		s.addMultilingualData(data, lang, "/article/"+slug)

		if s.templateEngine != nil {
			if html, err := s.templateEngine.Render("404", data); err == nil {
				c.Header("Content-Type", "text/html; charset=utf-8")
				c.String(http.StatusNotFound, html)
				return
			}
		}
		c.String(http.StatusNotFound, "Article not found: "+slug)
		return
	}

	// Build article data
	articleData := gin.H{
		"ID":              article.ID,
		"Title":           article.Title,
		"Slug":            article.Slug,
		"Content":         article.Content,
		"Excerpt":         article.Excerpt,
		"PublishedAt":     article.PublishedAt,
		"ViewCount":       article.ViewCount,
		"LikeCount":       article.LikeCount,
		"DislikeCount":    article.DislikeCount,
		"ReadTime":        calculateReadTime(article.Content),
		"MetaTitle":       article.MetaTitle,
		"MetaDescription": article.MetaDescription,
	}

	// Get featured image
	if article.FeaturedImageID != nil && *article.FeaturedImageID > 0 {
		if s.mediaService != nil {
			imageData := s.buildResponsiveImageData(*article.FeaturedImageID, article.Title)
			if imageData != nil {
				articleData["ImageData"] = imageData
			}
		}
	}

	// Get category - resolve to correct language version
	if article.CategoryID > 0 {
		categoryData, err := s.getCategoryByIDForLanguage(article.CategoryID, lang)
		if err == nil {
			articleData["Category"] = categoryData
		}
	}

	// Get tags - resolve to correct language versions
	tags, err := s.getArticleTagsForLanguage(article.ID, lang)
	if err == nil {
		articleData["Tags"] = tags
	}

	// Increment view count
	if s.db != nil {
		go func() {
			s.db.DB.Exec("UPDATE articles SET view_count = view_count + 1 WHERE id = $1", article.ID)
		}()
	}

	data := s.createBaseTemplateData(c)
	data["Title"] = article.Title
	data["PageType"] = "article"
	data["Article"] = articleData
	
	// Get available translations for correct hreflang tags
	availableTranslations, err := s.articleService.GetAvailableTranslations(c.Request.Context(), article.ID)
	if err == nil && len(availableTranslations) > 0 {
		// Build list of available languages
		var availableLangs []string
		for _, trans := range availableTranslations {
			availableLangs = append(availableLangs, trans.LanguageCode)
		}
		// Use the new method that only includes existing translations
		data["AlternateURLs"] = s.generateAlternateURLsForTranslations(availableLangs, "/article/"+slug)
	} else {
		// Fallback to current article's language only
		data["AlternateURLs"] = s.generateAlternateURLsForTranslations([]string{lang}, "/article/"+slug)
	}
	
	// Add other multilingual data
	data["LanguageCode"] = lang
	data["LanguageDirection"] = getLanguageDirection(lang)
	data["LanguageName"] = getLanguageName(lang)
	data["LanguageNativeName"] = getLanguageNativeName(lang)
	data["AvailableLanguages"] = s.getAvailableLanguagesForArticle(availableTranslations, slug)
	data["CanonicalURL"] = s.generateCanonicalURL(lang, "/article/"+slug)
	data["IsRTL"] = lang == "ar"
	data["BaseURL"] = s.config.App.BaseURL
	data["Navigation"] = s.getNavigationForLanguage(lang)

	if s.templateEngine != nil {
		if html, err := s.templateEngine.Render("article", data); err == nil {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, html)
			return
		}
	}

	c.String(http.StatusOK, article.Title)
}

// handleMultilingualCategory handles category pages for each language
func (s *Server) handleMultilingualCategory(c *gin.Context) {
	lang := c.Request.URL.Path[1:3]
	slug := c.Param("slug")

	data := s.createBaseTemplateData(c)
	data["PageType"] = "category"
	s.addMultilingualData(data, lang, "/category/"+slug)

	if s.categoryService != nil {
		category, err := s.categoryService.GetBySlug(slug)
		if err == nil {
			data["Title"] = category.Name
			data["Category"] = gin.H{
				"ID":          category.ID,
				"Name":        category.Name,
				"Slug":        category.Slug,
				"Description": category.Description,
			}

			// Get articles in this category
			if s.articleService != nil {
				categoryID := category.ID
				filters := services.ArticleFilters{
					Status:     "published",
					CategoryID: &categoryID,
				}
				articles, _, _ := s.articleService.List(c.Request.Context(), 20, 0, filters, "published_at", "DESC")

				articleData := make([]gin.H, len(articles))
				for i, article := range articles {
					articleData[i] = gin.H{
						"ID":        article.ID,
						"Title":     article.Title,
						"Slug":      article.Slug,
						"Excerpt":   article.Excerpt,
						"TimeAgo":   formatTimeAgo(article.PublishedAt),
						"ViewCount": article.ViewCount,
						"URL":       fmt.Sprintf("/%s/article/%s", lang, article.Slug),
					}
				}
				data["Articles"] = articleData
			}
		}
	}

	if s.templateEngine != nil {
		if html, err := s.templateEngine.Render("category", data); err == nil {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, html)
			return
		}
	}

	c.String(http.StatusOK, "Category: "+slug)
}

// handleMultilingualCategories handles the categories listing page
func (s *Server) handleMultilingualCategories(c *gin.Context) {
	lang := c.Request.URL.Path[1:3]

	data := s.createBaseTemplateData(c)
	data["Title"] = "Categories"
	data["PageType"] = "categories"
	s.addMultilingualData(data, lang, "/categories")

	if s.categoryService != nil {
		categories, err := s.categoryService.GetAll()
		if err == nil {
			categoryData := make([]gin.H, len(categories))
			for i, cat := range categories {
				categoryData[i] = gin.H{
					"ID":          cat.ID,
					"Name":        cat.Name,
					"Slug":        cat.Slug,
					"Description": cat.Description,
					"URL":         fmt.Sprintf("/%s/category/%s", lang, cat.Slug),
					"Count":       s.getCategoryArticleCount(cat.ID),
				}
			}
			data["Categories"] = categoryData
		}
	}

	if s.templateEngine != nil {
		if html, err := s.templateEngine.Render("categories", data); err == nil {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, html)
			return
		}
	}

	c.String(http.StatusOK, "Categories")
}

// handleMultilingualTag handles tag pages for each language
func (s *Server) handleMultilingualTag(c *gin.Context) {
	lang := c.Request.URL.Path[1:3]
	slug := c.Param("slug")

	data := s.createBaseTemplateData(c)
	data["PageType"] = "tag"
	s.addMultilingualData(data, lang, "/tag/"+slug)

	if s.tagService != nil {
		tag, err := s.tagService.GetBySlug(slug)
		if err == nil {
			data["Title"] = tag.Name
			data["Tag"] = gin.H{
				"ID":          tag.ID,
				"Name":        tag.Name,
				"Slug":        tag.Slug,
				"Description": tag.Description,
			}

			// Get articles with this tag
			if s.articleService != nil {
				tagID := tag.ID
				filters := services.ArticleFilters{
					Status: "published",
					TagID:  &tagID,
				}
				articles, _, _ := s.articleService.List(c.Request.Context(), 20, 0, filters, "published_at", "DESC")

				articleData := make([]gin.H, len(articles))
				for i, article := range articles {
					articleData[i] = gin.H{
						"ID":        article.ID,
						"Title":     article.Title,
						"Slug":      article.Slug,
						"Excerpt":   article.Excerpt,
						"TimeAgo":   formatTimeAgo(article.PublishedAt),
						"ViewCount": article.ViewCount,
						"URL":       fmt.Sprintf("/%s/article/%s", lang, article.Slug),
					}
				}
				data["Articles"] = articleData
			}
		}
	}

	if s.templateEngine != nil {
		if html, err := s.templateEngine.Render("tag", data); err == nil {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, html)
			return
		}
	}

	c.String(http.StatusOK, "Tag: "+slug)
}

// handleMultilingualTags handles the tags listing page
func (s *Server) handleMultilingualTags(c *gin.Context) {
	lang := c.Request.URL.Path[1:3]

	data := s.createBaseTemplateData(c)
	data["Title"] = "Tags"
	data["PageType"] = "tags"
	s.addMultilingualData(data, lang, "/tags")

	if s.tagService != nil {
		tags, err := s.tagService.GetAll()
		if err == nil {
			tagData := make([]gin.H, len(tags))
			for i, tag := range tags {
				tagData[i] = gin.H{
					"ID":          tag.ID,
					"Name":        tag.Name,
					"Slug":        tag.Slug,
					"Description": tag.Description,
					"URL":         fmt.Sprintf("/%s/tag/%s", lang, tag.Slug),
					"Count":       s.getTagArticleCount(tag.ID),
				}
			}
			data["Tags"] = tagData
		}
	}

	if s.templateEngine != nil {
		if html, err := s.templateEngine.Render("tags", data); err == nil {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, html)
			return
		}
	}

	c.String(http.StatusOK, "Tags")
}

// handleMultilingualLatest handles the latest articles page
func (s *Server) handleMultilingualLatest(c *gin.Context) {
	lang := c.Request.URL.Path[1:3]

	data := s.createBaseTemplateData(c)
	data["Title"] = "Latest Articles"
	data["PageType"] = "latest"
	s.addMultilingualData(data, lang, "/latest")

	if s.articleService != nil {
		filters := services.ArticleFilters{Status: "published"}
		articles, _, _ := s.articleService.List(c.Request.Context(), 20, 0, filters, "published_at", "DESC")

		articleData := make([]gin.H, len(articles))
		for i, article := range articles {
			articleData[i] = gin.H{
				"ID":        article.ID,
				"Title":     article.Title,
				"Slug":      article.Slug,
				"Excerpt":   article.Excerpt,
				"TimeAgo":   formatTimeAgo(article.PublishedAt),
				"ViewCount": article.ViewCount,
				"URL":       fmt.Sprintf("/%s/article/%s", lang, article.Slug),
			}
		}
		data["Articles"] = articleData
	}

	if s.templateEngine != nil {
		if html, err := s.templateEngine.Render("latest", data); err == nil {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, html)
			return
		}
	}

	c.String(http.StatusOK, "Latest Articles")
}

// handleMultilingualTrending handles the trending articles page
func (s *Server) handleMultilingualTrending(c *gin.Context) {
	lang := c.Request.URL.Path[1:3]

	data := s.createBaseTemplateData(c)
	data["Title"] = "Trending Articles"
	data["PageType"] = "trending"
	s.addMultilingualData(data, lang, "/trending")

	if s.articleService != nil {
		articles, _ := s.articleService.GetTrending(c.Request.Context(), 20, 24)

		articleData := make([]gin.H, len(articles))
		for i, article := range articles {
			articleData[i] = gin.H{
				"ID":        article.ID,
				"Title":     article.Title,
				"Slug":      article.Slug,
				"Excerpt":   article.Excerpt,
				"TimeAgo":   formatTimeAgo(article.PublishedAt),
				"ViewCount": article.ViewCount,
				"URL":       fmt.Sprintf("/%s/article/%s", lang, article.Slug),
			}
		}
		data["Articles"] = articleData
	}

	if s.templateEngine != nil {
		if html, err := s.templateEngine.Render("trending", data); err == nil {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, html)
			return
		}
	}

	c.String(http.StatusOK, "Trending Articles")
}

// handleMultilingualSearch handles the search page
func (s *Server) handleMultilingualSearch(c *gin.Context) {
	lang := c.Request.URL.Path[1:3]
	query := c.Query("q")

	data := s.createBaseTemplateData(c)
	data["Title"] = "Search"
	data["PageType"] = "search"
	data["SearchQuery"] = query
	s.addMultilingualData(data, lang, "/search")

	if query != "" && s.enterpriseSearchService != nil {
		searchReq := services.SearchRequest{
			Query:  query,
			Limit:  20,
			Offset: 0,
		}
		results, _ := s.enterpriseSearchService.Search(c.Request.Context(), searchReq)
		data["SearchResults"] = results
	}

	if s.templateEngine != nil {
		if html, err := s.templateEngine.Render("search", data); err == nil {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, html)
			return
		}
	}

	c.String(http.StatusOK, "Search: "+query)
}

// handleMultilingualAbout handles the about page
func (s *Server) handleMultilingualAbout(c *gin.Context) {
	lang := c.Request.URL.Path[1:3]

	data := s.createBaseTemplateData(c)
	data["Title"] = "About Us"
	data["PageType"] = "about"
	s.addMultilingualData(data, lang, "/about")

	if s.templateEngine != nil {
		if html, err := s.templateEngine.Render("about", data); err == nil {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, html)
			return
		}
	}

	c.String(http.StatusOK, "About Us")
}

// handleMultilingualContact handles the contact page
func (s *Server) handleMultilingualContact(c *gin.Context) {
	lang := c.Request.URL.Path[1:3]

	data := s.createBaseTemplateData(c)
	data["Title"] = "Contact Us"
	data["PageType"] = "contact"
	s.addMultilingualData(data, lang, "/contact")

	if s.templateEngine != nil {
		if html, err := s.templateEngine.Render("contact", data); err == nil {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, html)
			return
		}
	}

	c.String(http.StatusOK, "Contact Us")
}
