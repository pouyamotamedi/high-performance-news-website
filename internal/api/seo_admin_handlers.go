package api

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

// SEOSettings represents the SEO configuration
type SEOSettings struct {
	// General Settings
	SiteName         string `json:"siteName"`
	SiteTagline      string `json:"siteTagline"`
	SiteURL          string `json:"siteUrl"`
	PublisherName    string `json:"publisherName"`
	PublisherLogo    string `json:"publisherLogo"`
	DefaultLanguage  string `json:"defaultLanguage"`
	AutoMetaDesc     bool   `json:"autoMetaDesc"`
	EnableSchema     bool   `json:"enableSchema"`
	EnableBreadcrumbs bool  `json:"enableBreadcrumbs"`
	EnableCanonical  bool   `json:"enableCanonical"`

	// Meta Templates
	HomeTitle             string `json:"homeTitle"`
	HomeDesc              string `json:"homeDesc"`
	HomeKeywords          string `json:"homeKeywords"`
	ArticleTitleTemplate  string `json:"articleTitleTemplate"`
	ArticleDescTemplate   string `json:"articleDescTemplate"`
	CategoryTitleTemplate string `json:"categoryTitleTemplate"`
	CategoryDescTemplate  string `json:"categoryDescTemplate"`
	DefaultSchemaType     string `json:"defaultSchemaType"`

	// Social Media
	OGSiteName          string `json:"ogSiteName"`
	OGDefaultImage      string `json:"ogDefaultImage"`
	FBAppID             string `json:"fbAppId"`
	OGType              string `json:"ogType"`
	TwitterSite         string `json:"twitterSite"`
	TwitterCardType     string `json:"twitterCardType"`
	TwitterDefaultImage string `json:"twitterDefaultImage"`

	// Sitemap & RSS
	EnableSitemap      bool `json:"enableSitemap"`
	EnableNewsSitemap  bool `json:"enableNewsSitemap"`
	SitemapMaxURLs     int  `json:"sitemapMaxUrls"`
	NewsSitemapLimit   int  `json:"newsSitemapLimit"`
	EnableRSS          bool `json:"enableRSS"`
	EnableGoogleNewsRSS bool `json:"enableGoogleNewsRSS"`
	RSSDelayHours      int  `json:"rssDelayHours"`
	RSSCacheTTL        int  `json:"rssCacheTTL"`
	RSSItemLimit       int  `json:"rssItemLimit"`

	// Robots.txt
	RobotsTxtContent string `json:"robotsTxtContent"`
}

// SEOAdminHandlers handles SEO admin panel API endpoints
type SEOAdminHandlers struct {
	settings *SEOSettings
	mu       sync.RWMutex
}

// NewSEOAdminHandlers creates a new SEO admin handlers instance
func NewSEOAdminHandlers() *SEOAdminHandlers {
	return &SEOAdminHandlers{
		settings: getDefaultSEOSettings(),
	}
}

// getDefaultSEOSettings returns default SEO settings
func getDefaultSEOSettings() *SEOSettings {
	return &SEOSettings{
		SiteName:              "High Performance News",
		SiteTagline:           "Breaking News & Latest Updates",
		SiteURL:               "https://a.10top.shop",
		PublisherName:         "High Performance News",
		PublisherLogo:         "/static/images/logo.png",
		DefaultLanguage:       "en",
		AutoMetaDesc:          true,
		EnableSchema:          true,
		EnableBreadcrumbs:     true,
		EnableCanonical:       true,
		HomeTitle:             "High Performance News - Breaking News & Updates",
		HomeDesc:              "Get the latest breaking news, in-depth analysis, and exclusive stories from around the world.",
		HomeKeywords:          "news, breaking news, latest updates, world news",
		ArticleTitleTemplate:  "{title} - {site_name}",
		ArticleDescTemplate:   "{excerpt}",
		CategoryTitleTemplate: "{category} News - {site_name}",
		CategoryDescTemplate:  "Latest {category} news and updates from {site_name}",
		DefaultSchemaType:     "NewsArticle",
		OGSiteName:            "High Performance News",
		OGDefaultImage:        "/static/images/og-default.jpg",
		OGType:                "article",
		TwitterCardType:       "summary_large_image",
		EnableSitemap:         true,
		EnableNewsSitemap:     true,
		SitemapMaxURLs:        50000,
		NewsSitemapLimit:      1000,
		EnableRSS:             true,
		EnableGoogleNewsRSS:   true,
		RSSDelayHours:         2,
		RSSCacheTTL:           4,
		RSSItemLimit:          50,
		RobotsTxtContent: `# Robots.txt - Optimized for SEO + AI Protection
# Use {SITE_URL} as placeholder for your domain

User-agent: *
Allow: /
Disallow: /admin/
Disallow: /api/
Allow: /api/v1/articles/
Allow: /static/
Allow: /uploads/
Disallow: /*?utm_
Disallow: /*?fbclid=
Disallow: /*?gclid=
Crawl-delay: 1

# AI Bots - Block Training
User-agent: GPTBot
Disallow: /

User-agent: Google-Extended
Disallow: /

User-agent: ClaudeBot
Disallow: /

User-agent: CCBot
Disallow: /

User-agent: Bytespider
Disallow: /

User-agent: PerplexityBot
Disallow: /

User-agent: Amazonbot
Disallow: /

Sitemap: {SITE_URL}/sitemap.xml
Sitemap: {SITE_URL}/sitemap-news.xml`,
	}
}

// RegisterSEOAdminRoutes registers SEO admin routes
func (h *SEOAdminHandlers) RegisterSEOAdminRoutes(router *gin.RouterGroup) {
	seo := router.Group("/seo")
	{
		seo.GET("/settings", h.GetSEOSettings)
		seo.POST("/settings", h.SaveSEOSettings)
		seo.GET("/robots", h.GetRobotsTxt)
		seo.POST("/robots", h.SaveRobotsTxt)
		seo.POST("/sitemap/regenerate", h.RegenerateSitemap)
		seo.POST("/rss/refresh", h.RefreshRSSCache)
		seo.GET("/health", h.GetSEOHealth)
	}
}

// GetSEOSettings returns current SEO settings
func (h *SEOAdminHandlers) GetSEOSettings(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	c.JSON(http.StatusOK, h.settings)
}

// SaveSEOSettings saves SEO settings
func (h *SEOAdminHandlers) SaveSEOSettings(c *gin.Context) {
	var request struct {
		Section  string          `json:"section"`
		Settings json.RawMessage `json:"settings"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Parse and merge settings based on section
	switch request.Section {
	case "general":
		var general struct {
			SiteName          string `json:"siteName"`
			SiteTagline       string `json:"siteTagline"`
			SiteURL           string `json:"siteUrl"`
			PublisherName     string `json:"publisherName"`
			PublisherLogo     string `json:"publisherLogo"`
			DefaultLanguage   string `json:"defaultLanguage"`
			AutoMetaDesc      bool   `json:"autoMetaDesc"`
			EnableSchema      bool   `json:"enableSchema"`
			EnableBreadcrumbs bool   `json:"enableBreadcrumbs"`
			EnableCanonical   bool   `json:"enableCanonical"`
		}
		if err := json.Unmarshal(request.Settings, &general); err == nil {
			if general.SiteName != "" {
				h.settings.SiteName = general.SiteName
			}
			if general.SiteTagline != "" {
				h.settings.SiteTagline = general.SiteTagline
			}
			if general.SiteURL != "" {
				h.settings.SiteURL = general.SiteURL
			}
			if general.PublisherName != "" {
				h.settings.PublisherName = general.PublisherName
			}
			if general.PublisherLogo != "" {
				h.settings.PublisherLogo = general.PublisherLogo
			}
			if general.DefaultLanguage != "" {
				h.settings.DefaultLanguage = general.DefaultLanguage
			}
			h.settings.AutoMetaDesc = general.AutoMetaDesc
			h.settings.EnableSchema = general.EnableSchema
			h.settings.EnableBreadcrumbs = general.EnableBreadcrumbs
			h.settings.EnableCanonical = general.EnableCanonical
		}

	case "meta":
		var meta struct {
			HomeTitle             string `json:"homeTitle"`
			HomeDesc              string `json:"homeDesc"`
			HomeKeywords          string `json:"homeKeywords"`
			ArticleTitleTemplate  string `json:"articleTitleTemplate"`
			ArticleDescTemplate   string `json:"articleDescTemplate"`
			CategoryTitleTemplate string `json:"categoryTitleTemplate"`
			CategoryDescTemplate  string `json:"categoryDescTemplate"`
			DefaultSchemaType     string `json:"defaultSchemaType"`
		}
		if err := json.Unmarshal(request.Settings, &meta); err == nil {
			if meta.HomeTitle != "" {
				h.settings.HomeTitle = meta.HomeTitle
			}
			if meta.HomeDesc != "" {
				h.settings.HomeDesc = meta.HomeDesc
			}
			if meta.HomeKeywords != "" {
				h.settings.HomeKeywords = meta.HomeKeywords
			}
			if meta.ArticleTitleTemplate != "" {
				h.settings.ArticleTitleTemplate = meta.ArticleTitleTemplate
			}
			if meta.ArticleDescTemplate != "" {
				h.settings.ArticleDescTemplate = meta.ArticleDescTemplate
			}
			if meta.CategoryTitleTemplate != "" {
				h.settings.CategoryTitleTemplate = meta.CategoryTitleTemplate
			}
			if meta.CategoryDescTemplate != "" {
				h.settings.CategoryDescTemplate = meta.CategoryDescTemplate
			}
			if meta.DefaultSchemaType != "" {
				h.settings.DefaultSchemaType = meta.DefaultSchemaType
			}
		}

	case "social":
		var social struct {
			OGSiteName          string `json:"ogSiteName"`
			OGDefaultImage      string `json:"ogDefaultImage"`
			FBAppID             string `json:"fbAppId"`
			OGType              string `json:"ogType"`
			TwitterSite         string `json:"twitterSite"`
			TwitterCardType     string `json:"twitterCardType"`
			TwitterDefaultImage string `json:"twitterDefaultImage"`
		}
		if err := json.Unmarshal(request.Settings, &social); err == nil {
			if social.OGSiteName != "" {
				h.settings.OGSiteName = social.OGSiteName
			}
			if social.OGDefaultImage != "" {
				h.settings.OGDefaultImage = social.OGDefaultImage
			}
			h.settings.FBAppID = social.FBAppID
			if social.OGType != "" {
				h.settings.OGType = social.OGType
			}
			h.settings.TwitterSite = social.TwitterSite
			if social.TwitterCardType != "" {
				h.settings.TwitterCardType = social.TwitterCardType
			}
			h.settings.TwitterDefaultImage = social.TwitterDefaultImage
		}

	case "sitemap":
		var sitemap struct {
			EnableSitemap       bool `json:"enableSitemap"`
			EnableNewsSitemap   bool `json:"enableNewsSitemap"`
			SitemapMaxURLs      int  `json:"sitemapMaxUrls"`
			NewsSitemapLimit    int  `json:"newsSitemapLimit"`
			EnableRSS           bool `json:"enableRSS"`
			EnableGoogleNewsRSS bool `json:"enableGoogleNewsRSS"`
			RSSDelayHours       int  `json:"rssDelayHours"`
			RSSCacheTTL         int  `json:"rssCacheTTL"`
			RSSItemLimit        int  `json:"rssItemLimit"`
		}
		if err := json.Unmarshal(request.Settings, &sitemap); err == nil {
			h.settings.EnableSitemap = sitemap.EnableSitemap
			h.settings.EnableNewsSitemap = sitemap.EnableNewsSitemap
			if sitemap.SitemapMaxURLs > 0 {
				h.settings.SitemapMaxURLs = sitemap.SitemapMaxURLs
			}
			if sitemap.NewsSitemapLimit > 0 {
				h.settings.NewsSitemapLimit = sitemap.NewsSitemapLimit
			}
			h.settings.EnableRSS = sitemap.EnableRSS
			h.settings.EnableGoogleNewsRSS = sitemap.EnableGoogleNewsRSS
			if sitemap.RSSDelayHours >= 0 {
				h.settings.RSSDelayHours = sitemap.RSSDelayHours
			}
			if sitemap.RSSCacheTTL > 0 {
				h.settings.RSSCacheTTL = sitemap.RSSCacheTTL
			}
			if sitemap.RSSItemLimit > 0 {
				h.settings.RSSItemLimit = sitemap.RSSItemLimit
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "SEO settings saved successfully",
	})
}

// GetRobotsTxt returns current robots.txt content
func (h *SEOAdminHandlers) GetRobotsTxt(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"content": h.settings.RobotsTxtContent,
	})
}

// SaveRobotsTxt saves robots.txt content
func (h *SEOAdminHandlers) SaveRobotsTxt(c *gin.Context) {
	var request struct {
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	h.mu.Lock()
	h.settings.RobotsTxtContent = request.Content
	h.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Robots.txt saved successfully",
	})
}

// RegenerateSitemap triggers sitemap regeneration
func (h *SEOAdminHandlers) RegenerateSitemap(c *gin.Context) {
	// In a real implementation, this would trigger sitemap regeneration
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Sitemap regeneration triggered",
	})
}

// RefreshRSSCache refreshes RSS feed cache
func (h *SEOAdminHandlers) RefreshRSSCache(c *gin.Context) {
	// In a real implementation, this would clear RSS cache
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "RSS cache refresh triggered",
	})
}

// GetSEOHealth returns SEO health check results
func (h *SEOAdminHandlers) GetSEOHealth(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	checks := []gin.H{
		{"name": "Sitemap enabled", "status": h.settings.EnableSitemap},
		{"name": "RSS enabled", "status": h.settings.EnableRSS},
		{"name": "Schema markup enabled", "status": h.settings.EnableSchema},
		{"name": "Canonical URLs enabled", "status": h.settings.EnableCanonical},
		{"name": "Breadcrumbs enabled", "status": h.settings.EnableBreadcrumbs},
		{"name": "Google News RSS enabled", "status": h.settings.EnableGoogleNewsRSS},
		{"name": "News sitemap enabled", "status": h.settings.EnableNewsSitemap},
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"checks": checks,
	})
}

// GetSettings returns the current settings (for use by other services)
func (h *SEOAdminHandlers) GetSettings() *SEOSettings {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.settings
}
