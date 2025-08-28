package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
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
	config                   *config.Config
	router                   *gin.Engine
	cache                    cache.CacheService
	db                       *database.DB
	apiRouter                *api.Router
	templateEngine           *templates.TemplateEngine
	rssHandlers              *api.RSSHandlers
	googleNewsHandlers       *api.GoogleNewsHandlers
	metricsService           *services.MetricsService
	healthService            *services.HealthService
	alertingService          *services.AlertingService
	monitoringConfig         *config.MonitoringConfig
	analyticsService         *services.AnalyticsService
	advertisementService     *services.AdvertisementService
	pushNotificationService  *services.PushNotificationService
	widgetService            *services.WidgetService
	themeService             *services.ThemeService
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
	baseURL := fmt.Sprintf("http://localhost:%d", cfg.Server.Port) // In production, this would come from config
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
	var searchService *services.SearchService
	var commentHandlers *api.CommentHandlers
	var imageHandlers *api.ImageHandlers
	var analyticsService *services.AnalyticsService
	var advertisementService *services.AdvertisementService
	var pushNotificationService *services.PushNotificationService

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
		
		configService = services.NewConfigService()
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
		searchRepo := &services.ArticleRepositoryAdapter{Repository: articleRepo}

		redisClient := dragonflyCache.GetRedisClient()
		
		if cfg.Search.Enabled {
			searchIndexer := services.NewSearchIndexer(
				cfg.Search.MeiliSearchURL,
				cfg.Search.MeiliSearchAPIKey,
				cfg.Search.IndexName,
				searchRepo,
			)
			searchService = services.NewSearchService(searchIndexer, redisClient, searchRepo)
		} else {
			searchService = services.NewSearchService(nil, redisClient, searchRepo)
		}

		articleService = services.NewArticleService(db, articleRepo, nil)
		contentIngestionService = &services.ContentIngestionService{}
		commentHandlers = &api.CommentHandlers{}
		imageHandlers = &api.ImageHandlers{}

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
	
	if !useMockServices {
		finalAnalyticsService = analyticsService
		finalAdvertisementService = advertisementService
		finalPushNotificationService = pushNotificationService
		finalWidgetService = widgetService
		finalThemeService = themeService
	}

	return &Server{
		config:                   cfg,
		router:                   router,
		cache:                    cacheClient,
		db:                       db,
		apiRouter:                apiRouter,
		templateEngine:           templateEngine,
		rssHandlers:              rssHandlers,
		googleNewsHandlers:       googleNewsHandlers,
		metricsService:           finalMetricsService,
		healthService:            finalHealthService,
		alertingService:          finalAlertingService,
		monitoringConfig:         finalMonitoringConfig,
		analyticsService:         finalAnalyticsService,
		advertisementService:     finalAdvertisementService,
		pushNotificationService:  finalPushNotificationService,
		widgetService:            finalWidgetService,
		themeService:             finalThemeService,
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
	s.router.GET("/", s.handleHomepage)
	
	// Frontend website routes (missing from Task 22 implementation)
	if !s.config.App.DevMode {
		s.setupProductionFrontendRoutes()
		// Setup admin frontend routes for production mode
		s.setupAdminFrontendRoutes()
	}
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
	_, err := userService.GetByEmail("admin@example.com")
	if err == nil {
		return nil
	}
	
	createReq := &services.CreateUserRequest{
		Username:  "admin",
		Email:     "admin@example.com",
		Password:  "Admin123!",
		Role:      models.RoleAdmin,
		FirstName: "Demo",
		LastName:  "Admin",
		Bio:       "Demo administrator account",
	}
	
	_, err = userService.Create(createReq, nil)
	if err != nil {
		return fmt.Errorf("failed to create demo user: %w", err)
	}
	
	log.Println("Demo admin user created successfully (admin@example.com / Admin123!)")
	return nil
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
	data["Articles"] = []gin.H{
		{
			"Title":    "Welcome to our High Performance News Website",
			"Slug":     "welcome-news-website",
			"Excerpt":  "This is a high-performance news website with RSS feeds and Google News integration.",
			"Author":   "News Team",
			"TimeAgo":  "1 hour ago",
			"Views":    2345,
			"Category": "Announcement",
		},
		{
			"Title":    "Breaking: Latest Technology Innovation",
			"Slug":     "latest-tech-innovation",
			"Excerpt":  "A groundbreaking technology innovation has been announced today.",
			"Author":   "Tech Reporter",
			"TimeAgo":  "2 hours ago",
			"Views":    1876,
			"Category": "Technology",
		},
		{
			"Title":    "Sports Championship Finals Begin",
			"Slug":     "sports-championship-finals",
			"Excerpt":  "The highly anticipated championship finals are starting this weekend.",
			"Author":   "Sports Writer",
			"TimeAgo":  "3 hours ago",
			"Views":    1543,
			"Category": "Sports",
		},
	}
	data["Categories"] = []gin.H{
		{"Name": "Technology", "Slug": "technology", "Description": "Latest tech news", "Count": 25},
		{"Name": "Sports", "Slug": "sports", "Description": "Sports updates", "Count": 18},
		{"Name": "Politics", "Slug": "politics", "Description": "Political news", "Count": 32},
		{"Name": "Business", "Slug": "business", "Description": "Business news", "Count": 22},
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
	
	server := &Server{
		config:                   cfg,
		router:                   router,
		cache:                    nil, // No cache in dev mode
		db:                       nil, // No database in dev mode
		apiRouter:                nil, // No API router in dev mode
		templateEngine:           templateEngine,
		rssHandlers:              rssHandlers,
		googleNewsHandlers:       googleNewsHandlers,
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

// setupProductionFrontendRoutes sets up frontend routes for production mode
// This was missing from Task 22 implementation
func (s *Server) setupProductionFrontendRoutes() {
	// Article pages
	s.router.GET("/article/:slug", s.handleProductionArticle)
	
	// Category pages  
	s.router.GET("/category/:slug", s.handleProductionCategory)
	s.router.GET("/categories", s.handleProductionCategories)
	
	// Tag pages
	s.router.GET("/tag/:slug", s.handleProductionTag)
	s.router.GET("/tags", s.handleProductionTags)
	
	// Content listing pages
	s.router.GET("/latest", s.handleProductionLatest)
	s.router.GET("/trending", s.handleProductionTrending)
	
	// Static pages
	s.router.GET("/about", s.handleProductionAbout)
	s.router.GET("/contact", s.handleProductionContact)
}

// Production frontend handlers (missing from Task 22)
func (s *Server) handleProductionArticle(c *gin.Context) {
	slug := c.Param("slug")
	
	// Create proper template data structure
	data := s.createBaseTemplateData(c)
	data["Title"] = "Article: " + slug
	data["Article"] = gin.H{
		"Title":   "Sample Article: " + slug,
		"Slug":    slug,
		"Content": "<p>This is sample article content for: " + slug + "</p><p>Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.</p>",
		"Excerpt": "This is a sample article excerpt for " + slug,
		"Author": gin.H{
			"FirstName": "Demo",
			"LastName":  "Author",
			"Bio":       "Sample author bio",
		},
		"Category": gin.H{
			"Name": "Sample Category",
			"Slug": "sample-category",
		},
		"Tags": []gin.H{
			{"Name": "sample", "Slug": "sample"},
			{"Name": "demo", "Slug": "demo"},
		},
		"Views":    1234,
		"ReadTime": 5,
	}
	
	if s.templateEngine != nil {
		if html, err := s.templateEngine.Render("article", data); err == nil {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, html)
			return
		}
	}
	
	// Fallback to simple response
	c.String(http.StatusOK, "Article: "+slug+" (Template not available)")
}

func (s *Server) handleProductionCategory(c *gin.Context) {
	slug := c.Param("slug")
	
	data := s.createBaseTemplateData(c)
	data["Title"] = "Category: " + slug
	data["Category"] = gin.H{
		"Name":        slug,
		"Slug":        slug,
		"Description": "Articles in the " + slug + " category",
	}
	data["Articles"] = []gin.H{
		{
			"Title":   "Sample Article 1 in " + slug,
			"Slug":    "sample-article-1",
			"Excerpt": "This is a sample article excerpt",
			"Author":  "Demo Author",
			"TimeAgo": "2 hours ago",
		},
		{
			"Title":   "Sample Article 2 in " + slug,
			"Slug":    "sample-article-2", 
			"Excerpt": "Another sample article excerpt",
			"Author":  "Demo Author",
			"TimeAgo": "4 hours ago",
		},
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
	data := s.createBaseTemplateData(c)
	data["Title"] = "All Categories"
	data["Categories"] = []gin.H{
		{"Name": "Technology", "Slug": "technology", "Description": "Latest tech news", "Count": 25},
		{"Name": "Sports", "Slug": "sports", "Description": "Sports updates", "Count": 18},
		{"Name": "Politics", "Slug": "politics", "Description": "Political news", "Count": 32},
		{"Name": "Business", "Slug": "business", "Description": "Business news", "Count": 22},
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
	
	data := s.createBaseTemplateData(c)
	data["Title"] = "Tag: " + slug
	data["Tag"] = gin.H{
		"Name":        slug,
		"Slug":        slug,
		"Description": "Articles tagged with " + slug,
	}
	data["Articles"] = []gin.H{
		{
			"Title":   "Sample Article tagged with " + slug,
			"Slug":    "sample-tagged-article-1",
			"Excerpt": "This article is tagged with " + slug,
			"Author":  "Demo Author",
			"TimeAgo": "1 hour ago",
		},
		{
			"Title":   "Another " + slug + " tagged article",
			"Slug":    "sample-tagged-article-2",
			"Excerpt": "Another article with the " + slug + " tag",
			"Author":  "Demo Author",
			"TimeAgo": "3 hours ago",
		},
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
	data := s.createBaseTemplateData(c)
	data["Title"] = "All Tags"
	data["Tags"] = []gin.H{
		{"Name": "breaking", "Slug": "breaking", "Count": 15},
		{"Name": "news", "Slug": "news", "Count": 42},
		{"Name": "technology", "Slug": "technology", "Count": 28},
		{"Name": "sports", "Slug": "sports", "Count": 18},
		{"Name": "politics", "Slug": "politics", "Count": 22},
		{"Name": "business", "Slug": "business", "Count": 19},
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
			"Title":   "Latest Breaking News Story",
			"Slug":    "latest-breaking-news",
			"Excerpt": "This is the most recent breaking news story",
			"Author":  "News Reporter",
			"TimeAgo": "2 minutes ago",
			"Views":   1234,
			"Category": "Breaking News",
		},
		{
			"Title":   "New Technology Innovation Announced",
			"Slug":    "tech-innovation",
			"Excerpt": "A groundbreaking technology innovation",
			"Author":  "Tech Writer",
			"TimeAgo": "15 minutes ago",
			"Views":   856,
			"Category": "Technology",
		},
		{
			"Title":   "Major Sports Championship Update",
			"Slug":    "sports-update",
			"Excerpt": "Latest updates from the championship",
			"Author":  "Sports Reporter",
			"TimeAgo": "30 minutes ago",
			"Views":   642,
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
			"Title":   "🔥 Most Viral Story of the Week",
			"Slug":    "viral-story",
			"Excerpt": "This story has gone viral across social media",
			"Author":  "Viral Reporter",
			"TimeAgo": "2 hours ago",
			"Views":   15420,
			"Category": "Viral",
		},
		{
			"Title":   "📈 Trending Technology News",
			"Slug":    "trending-tech",
			"Excerpt": "This tech story is gaining massive attention",
			"Author":  "Tech Analyst",
			"TimeAgo": "4 hours ago",
			"Views":   12350,
			"Category": "Technology",
		},
		{
			"Title":   "⚡ Breaking Sports Sensation",
			"Slug":    "trending-sports",
			"Excerpt": "A sports story capturing everyone's attention",
			"Author":  "Sports Writer",
			"TimeAgo": "6 hours ago",
			"Views":   9870,
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

// setupAdminFrontendRoutes sets up admin panel frontend routes
func (s *Server) setupAdminFrontendRoutes() {
	// Admin login page (no auth required)
	s.router.GET("/admin/login", s.handleAdminLogin)
	
	// Admin dashboard and pages
	adminGroup := s.router.Group("/admin")
	{
		adminGroup.GET("/", s.handleAdminDashboard)
		adminGroup.GET("/dashboard", s.handleAdminDashboard)
		adminGroup.GET("/analytics", s.handleAdminAnalytics)
		adminGroup.GET("/users", s.handleAdminUsers)
		adminGroup.GET("/content", s.handleAdminContent)
		adminGroup.GET("/settings", s.handleAdminSettings)
		adminGroup.GET("/system", s.handleAdminSystem)
	}
}

// Admin frontend handlers
func (s *Server) handleAdminLogin(c *gin.Context) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Admin Login - High Performance News Website</title>
    <link rel="stylesheet" href="/static/css/admin.css">
</head>
<body>
    <div class="container">
        <div class="login-form">
            <div class="login-header">
                <h2 class="login-title">Admin Panel Login</h2>
                <p class="login-subtitle">Sign in to access the admin dashboard</p>
            </div>
            
            <div id="errorMessage" class="error-message hidden"></div>
            
            <form id="loginForm">
                <div class="form-group">
                    <label for="email" class="form-label">Email address</label>
                    <input 
                        id="email" 
                        name="email" 
                        type="email" 
                        autocomplete="email" 
                        required 
                        class="form-input" 
                        placeholder="Enter your email"
                    >
                </div>
                
                <div class="form-group">
                    <label for="password" class="form-label">Password</label>
                    <input 
                        id="password" 
                        name="password" 
                        type="password" 
                        autocomplete="current-password" 
                        required 
                        class="form-input" 
                        placeholder="Enter your password"
                    >
                </div>

                <button type="submit" id="loginButton" class="btn">
                    <span id="loadingSpan" class="loading hidden"></span>
                    <span id="buttonText">Sign in</span>
                </button>
            </form>

            <div class="demo-credentials">
                <p>Demo credentials:</p>
                <p><strong>Email:</strong> admin@example.com</p>
                <p><strong>Password:</strong> Admin123!</p>
            </div>
        </div>
    </div>

    <script src="/static/js/admin-login.js"></script>
</body>
</html>`
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (s *Server) handleAdminDashboard(c *gin.Context) {
	s.renderAdminDashboard(c, "Admin Dashboard", "")
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

func (s *Server) renderAdminDashboard(c *gin.Context, title, page string) {
	activeClass := func(currentPage, targetPage string) string {
		if currentPage == targetPage || (currentPage == "" && targetPage == "") {
			return "active"
		}
		return ""
	}
	
	pageContent := ""
	if page != "" {
		pageContent = fmt.Sprintf(`
            <div class="dashboard-card">
                <div class="card-title">%s Management</div>
                <div>
                    <p>This is the %s management page.</p>
                    <p>Features for %s management will be implemented here.</p>
                </div>
            </div>

            <div class="dashboard-card">
                <div class="card-title">%s Statistics</div>
                <div>
                    <p>Statistics and metrics for %s will be displayed here.</p>
                </div>
            </div>`, page, page, page, page, page)
	} else {
		pageContent = `
            <div class="dashboard-card">
                <div class="card-title">System Status</div>
                <div>
                    <p>Database: <span class="status-badge status-healthy">Healthy</span></p>
                    <p>Cache: <span class="status-badge status-healthy">Operational</span></p>
                    <p>Search: <span class="status-badge status-healthy">Available</span></p>
                </div>
            </div>

            <div class="dashboard-card">
                <div class="card-title">Quick Stats</div>
                <div>
                    <p>Total Articles: <span class="metric">0</span></p>
                    <p>Active Users: <span class="metric">1</span></p>
                    <p>Today's Views: <span class="metric">0</span></p>
                </div>
            </div>

            <div class="dashboard-card">
                <div class="card-title">Recent Activity</div>
                <div>
                    <p>• Admin user logged in</p>
                    <p>• System started successfully</p>
                    <p>• Database connection established</p>
                </div>
            </div>

            <div class="dashboard-card">
                <div class="card-title">Quick Actions</div>
                <div class="actions-container">
                    <a href="/admin/content" class="action-button">📝 Manage Content</a>
                    <a href="/admin/users" class="action-button">👥 Manage Users</a>
                    <a href="/admin/system" class="action-button">⚙️ System Monitor</a>
                </div>
            </div>`
	}
	
	welcomeMessage := ""
	if page == "" {
		welcomeMessage = `
        <div class="welcome-message">
            <h2>Welcome to the Admin Dashboard</h2>
            <p>Manage your high-performance news website from here</p>
        </div>`
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s - High Performance News Website</title>
    <link rel="stylesheet" href="/static/css/admin.css">
    <style>
        .dashboard-container { max-width: 1200px; margin: 0 auto; padding: 2rem; }
        .dashboard-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 2rem; padding-bottom: 1rem; border-bottom: 2px solid #e5e7eb; }
        .dashboard-nav { display: flex; gap: 1rem; margin-bottom: 2rem; flex-wrap: wrap; }
        .nav-item { padding: 0.5rem 1rem; background-color: #f3f4f6; border-radius: 6px; text-decoration: none; color: #374151; transition: background-color 0.2s; }
        .nav-item:hover, .nav-item.active { background-color: #3b82f6; color: white; }
        .dashboard-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 2rem; }
        .dashboard-card { background: white; padding: 1.5rem; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .card-title { font-size: 1.2rem; font-weight: 600; margin-bottom: 1rem; color: #1f2937; }
        .metric { font-size: 2rem; font-weight: bold; color: #3b82f6; }
        .logout-btn { background-color: #ef4444; color: white; padding: 0.5rem 1rem; border: none; border-radius: 6px; cursor: pointer; text-decoration: none; }
        .logout-btn:hover { background-color: #dc2626; }
        .status-badge { display: inline-block; padding: 0.25rem 0.5rem; border-radius: 4px; font-size: 0.75rem; font-weight: 500; }
        .status-healthy { background-color: #d1fae5; color: #065f46; }
        .welcome-message { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 2rem; border-radius: 8px; margin-bottom: 2rem; text-align: center; }
        .action-button { display: inline-block; padding: 0.75rem 1.5rem; background-color: #3b82f6; color: white; text-decoration: none; border-radius: 6px; margin: 0.5rem 0.5rem 0.5rem 0; transition: background-color 0.2s; font-size: 0.9rem; }
        .action-button:hover { background-color: #2563eb; color: white; }
        .actions-container { display: flex; flex-direction: column; gap: 0.5rem; }
    </style>
</head>
<body>
    <div class="dashboard-container">
        <div class="dashboard-header">
            <h1>%s</h1>
            <div>
                <span>Welcome, Admin</span>
                <button class="logout-btn" onclick="logout()">Logout</button>
            </div>
        </div>

        <div class="dashboard-nav">
            <a href="/admin/dashboard" class="nav-item %s">Dashboard</a>
            <a href="/admin/analytics" class="nav-item %s">Analytics</a>
            <a href="/admin/users" class="nav-item %s">Users</a>
            <a href="/admin/content" class="nav-item %s">Content</a>
            <a href="/admin/settings" class="nav-item %s">Settings</a>
            <a href="/admin/system" class="nav-item %s">System</a>
        </div>

        %s

        <div class="dashboard-grid">
            %s
        </div>
    </div>

    <script>
        function logout() {
            localStorage.removeItem('auth_token');
            localStorage.removeItem('user_role');
            window.location.href = '/admin/login';
        }
        document.addEventListener('DOMContentLoaded', function() {
            console.log('Admin dashboard loaded successfully');
        });
    </script>
</body>
</html>`, 
		title, title,
		activeClass(page, ""), 
		activeClass(page, "analytics"), 
		activeClass(page, "users"), 
		activeClass(page, "content"), 
		activeClass(page, "settings"), 
		activeClass(page, "system"),
		welcomeMessage,
		pageContent)
	
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
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
	return gin.H{
		"SiteName":         s.config.App.Name,
		"SiteDescription":  "High-performance multilingual news website",
		"LanguageCode":     "en",
		"LanguageDirection": "ltr",
		"ThemeMode":        "auto",
		"CurrentYear":      2024,
		"Navigation": []gin.H{
			{"Name": "Home", "URL": "/", "Active": c.Request.URL.Path == "/"},
			{"Name": "Latest", "URL": "/latest", "Active": c.Request.URL.Path == "/latest"},
			{"Name": "Trending", "URL": "/trending", "Active": c.Request.URL.Path == "/trending"},
			{"Name": "Categories", "URL": "/categories", "Active": c.Request.URL.Path == "/categories"},
			{"Name": "Tags", "URL": "/tags", "Active": c.Request.URL.Path == "/tags"},
			{"Name": "About", "URL": "/about", "Active": c.Request.URL.Path == "/about"},
			{"Name": "Contact", "URL": "/contact", "Active": c.Request.URL.Path == "/contact"},
		},
		"IsAuthenticated": false,
		"OGType":         "website",
		"TwitterCard":    "summary_large_image",
	}
}
//
