package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/services"
)

// SEOHandlers handles SEO-related API endpoints
type SEOHandlers struct {
	seoService       *services.SEOService
	sitemapService   *services.SitemapService
	breadcrumbService *services.BreadcrumbService
	articleService   *services.ArticleService
	categoryService  *services.CategoryService
	tagService       *services.TagService
}

// NewSEOHandlers creates a new SEO handlers instance
func NewSEOHandlers(
	seoService *services.SEOService,
	sitemapService *services.SitemapService,
	breadcrumbService *services.BreadcrumbService,
	articleService *services.ArticleService,
	categoryService *services.CategoryService,
	tagService *services.TagService,
) *SEOHandlers {
	return &SEOHandlers{
		seoService:        seoService,
		sitemapService:    sitemapService,
		breadcrumbService: breadcrumbService,
		articleService:    articleService,
		categoryService:   categoryService,
		tagService:        tagService,
	}
}

// RegisterSEORoutes registers all SEO-related routes
func (h *SEOHandlers) RegisterSEORoutes(router *gin.Engine) {
	// Sitemap routes
	router.GET("/sitemap.xml", h.handleSitemapIndex)
	router.GET("/sitemap-main.xml", h.handleMainSitemap)
	router.GET("/sitemap-articles-:page.xml", h.handleArticleSitemap)
	router.GET("/sitemap-news-:page.xml", h.handleNewsSitemap)
	router.GET("/sitemap-categories.xml", h.handleCategorySitemap)
	router.GET("/sitemap-tags.xml", h.handleTagSitemap)

	// Robots.txt
	router.GET("/robots.txt", h.handleRobotsTxt)

	// SEO API endpoints (for admin/testing)
	seoAPI := router.Group("/api/v1/seo")
	{
		// Schema markup endpoints
		seoAPI.GET("/schema/article/:slug", h.handleArticleSchema)
		seoAPI.GET("/schema/category/:slug", h.handleCategorySchema)
		seoAPI.GET("/schema/tag/:slug", h.handleTagSchema)
		seoAPI.GET("/schema/homepage", h.handleHomepageSchema)
		
		// Meta tags endpoints
		seoAPI.GET("/meta/article/:slug", h.handleArticleMetaTags)
		seoAPI.GET("/meta/category/:slug", h.handleCategoryMetaTags)
		seoAPI.GET("/meta/tag/:slug", h.handleTagMetaTags)
		seoAPI.GET("/meta/homepage", h.handleHomepageMetaTags)
		
		// Breadcrumb endpoints
		seoAPI.GET("/breadcrumbs/article/:slug", h.handleArticleBreadcrumbs)
		seoAPI.GET("/breadcrumbs/category/:slug", h.handleCategoryBreadcrumbs)
		seoAPI.GET("/breadcrumbs/tag/:slug", h.handleTagBreadcrumbs)
		seoAPI.GET("/breadcrumbs/search", h.handleSearchBreadcrumbs)
		
		// SEO validation endpoints
		seoAPI.POST("/validate/schema", h.handleValidateSchema)
		seoAPI.POST("/validate/meta", h.handleValidateMeta)
		seoAPI.POST("/validate/breadcrumbs", h.handleValidateBreadcrumbs)
		
		// SEO optimization endpoints
		seoAPI.GET("/optimize/title/:slug", h.handleOptimizeTitle)
		seoAPI.GET("/optimize/description/:slug", h.handleOptimizeDescription)
		seoAPI.GET("/analyze/keywords/:slug", h.handleAnalyzeKeywords)
	}
}

// Sitemap handlers

func (h *SEOHandlers) handleSitemapIndex(c *gin.Context) {
	// Get all data needed for sitemap generation
	data, err := h.getSitemapData()
	if err != nil {
		c.String(http.StatusInternalServerError, "Error generating sitemap index")
		return
	}

	index, err := h.sitemapService.GenerateSitemapIndex(data)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error generating sitemap index")
		return
	}

	xml, err := h.sitemapService.RenderXML(index)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error rendering sitemap XML")
		return
	}

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.String(http.StatusOK, xml)
}

func (h *SEOHandlers) handleMainSitemap(c *gin.Context) {
	sitemap, err := h.sitemapService.GenerateMainSitemap()
	if err != nil {
		c.String(http.StatusInternalServerError, "Error generating main sitemap")
		return
	}

	xml, err := h.sitemapService.RenderXML(sitemap)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error rendering sitemap XML")
		return
	}

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.String(http.StatusOK, xml)
}

func (h *SEOHandlers) handleArticleSitemap(c *gin.Context) {
	pageStr := c.Param("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.String(http.StatusBadRequest, "Invalid page number")
		return
	}

	// Get published articles
	articles, err := h.getPublishedArticles()
	if err != nil {
		c.String(http.StatusInternalServerError, "Error fetching articles")
		return
	}

	sitemap, err := h.sitemapService.GenerateArticleSitemap(articles, page)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error generating article sitemap")
		return
	}

	xml, err := h.sitemapService.RenderXML(sitemap)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error rendering sitemap XML")
		return
	}

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.String(http.StatusOK, xml)
}

func (h *SEOHandlers) handleNewsSitemap(c *gin.Context) {
	pageStr := c.Param("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.String(http.StatusBadRequest, "Invalid page number")
		return
	}

	// Get recent published articles (last 48 hours for Google News)
	articles, err := h.getRecentArticles(48 * time.Hour)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error fetching recent articles")
		return
	}

	sitemap, err := h.sitemapService.GenerateNewsSitemap(articles, "High Performance News", "en", page)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error generating news sitemap")
		return
	}

	xml, err := h.sitemapService.RenderXML(sitemap)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error rendering sitemap XML")
		return
	}

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.String(http.StatusOK, xml)
}

func (h *SEOHandlers) handleCategorySitemap(c *gin.Context) {
	categories, err := h.getAllCategories()
	if err != nil {
		c.String(http.StatusInternalServerError, "Error fetching categories")
		return
	}

	sitemap, err := h.sitemapService.GenerateCategorySitemap(categories)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error generating category sitemap")
		return
	}

	xml, err := h.sitemapService.RenderXML(sitemap)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error rendering sitemap XML")
		return
	}

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.String(http.StatusOK, xml)
}

func (h *SEOHandlers) handleTagSitemap(c *gin.Context) {
	tags, err := h.getAllTags()
	if err != nil {
		c.String(http.StatusInternalServerError, "Error fetching tags")
		return
	}

	sitemap, err := h.sitemapService.GenerateTagSitemap(tags)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error rendering sitemap XML")
		return
	}

	xml, err := h.sitemapService.RenderXML(sitemap)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error rendering sitemap XML")
		return
	}

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.String(http.StatusOK, xml)
}

func (h *SEOHandlers) handleRobotsTxt(c *gin.Context) {
	robotsTxt := `User-agent: *
Allow: /

# Sitemaps
Sitemap: %s/sitemap.xml

# Crawl-delay for polite crawling
Crawl-delay: 1

# Disallow admin areas
Disallow: /admin/
Disallow: /api/

# Allow specific paths
Allow: /api/v1/articles/
Allow: /static/`

	baseURL := h.getBaseURL(c)
	content := fmt.Sprintf(robotsTxt, baseURL)

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, content)
}

// Schema markup handlers

func (h *SEOHandlers) handleArticleSchema(c *gin.Context) {
	slug := c.Param("slug")
	
	article, err := h.getArticleBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})
		return
	}

	// Get additional data (author, category) - for now using mock data
	author := &models.User{
		FirstName: "Demo",
		LastName:  "Author",
		Username:  "demo-author",
	}

	category := &models.Category{
		Name: "Technology",
		Slug: "technology",
	}

	schema, err := h.seoService.GenerateArticleSchema(article, author, category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating schema"})
		return
	}

	c.JSON(http.StatusOK, schema)
}

func (h *SEOHandlers) handleCategorySchema(c *gin.Context) {
	slug := c.Param("slug")
	
	category, err := h.getCategoryBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	schema, err := h.seoService.GenerateCategorySchema(category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating schema"})
		return
	}

	c.JSON(http.StatusOK, schema)
}

func (h *SEOHandlers) handleTagSchema(c *gin.Context) {
	slug := c.Param("slug")
	
	tag, err := h.getTagBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tag not found"})
		return
	}

	schema, err := h.seoService.GenerateTagSchema(tag)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating schema"})
		return
	}

	c.JSON(http.StatusOK, schema)
}

func (h *SEOHandlers) handleHomepageSchema(c *gin.Context) {
	schema, err := h.seoService.GenerateHomepageSchema()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating schema"})
		return
	}

	c.JSON(http.StatusOK, schema)
}

// Meta tags handlers

func (h *SEOHandlers) handleArticleMetaTags(c *gin.Context) {
	slug := c.Param("slug")
	
	article, err := h.getArticleBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})
		return
	}

	meta, err := h.seoService.GenerateMetaTags("article", article)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating meta tags"})
		return
	}

	c.JSON(http.StatusOK, meta)
}

func (h *SEOHandlers) handleCategoryMetaTags(c *gin.Context) {
	slug := c.Param("slug")
	
	category, err := h.getCategoryBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	meta, err := h.seoService.GenerateMetaTags("category", category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating meta tags"})
		return
	}

	c.JSON(http.StatusOK, meta)
}

func (h *SEOHandlers) handleTagMetaTags(c *gin.Context) {
	slug := c.Param("slug")
	
	tag, err := h.getTagBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tag not found"})
		return
	}

	meta, err := h.seoService.GenerateMetaTags("tag", tag)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating meta tags"})
		return
	}

	c.JSON(http.StatusOK, meta)
}

// Breadcrumb handlers

func (h *SEOHandlers) handleArticleBreadcrumbs(c *gin.Context) {
	slug := c.Param("slug")
	
	article, err := h.getArticleBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})
		return
	}

	// Get category data - for now using mock data
	category := &models.Category{
		Name: "Technology",
		Slug: "technology",
	}

	breadcrumbs, err := h.breadcrumbService.GenerateArticleBreadcrumbs(article, category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating breadcrumbs"})
		return
	}

	c.JSON(http.StatusOK, breadcrumbs)
}

func (h *SEOHandlers) handleCategoryBreadcrumbs(c *gin.Context) {
	slug := c.Param("slug")
	
	category, err := h.getCategoryBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	breadcrumbs, err := h.breadcrumbService.GenerateCategoryBreadcrumbs(category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating breadcrumbs"})
		return
	}

	c.JSON(http.StatusOK, breadcrumbs)
}

func (h *SEOHandlers) handleTagBreadcrumbs(c *gin.Context) {
	slug := c.Param("slug")
	
	tag, err := h.getTagBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tag not found"})
		return
	}

	breadcrumbs, err := h.breadcrumbService.GenerateTagBreadcrumbs(tag)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating breadcrumbs"})
		return
	}

	c.JSON(http.StatusOK, breadcrumbs)
}

// Helper methods

func (h *SEOHandlers) getSitemapData() (*services.SitemapData, error) {
	articles, err := h.getPublishedArticles()
	if err != nil {
		return nil, err
	}

	categories, err := h.getAllCategories()
	if err != nil {
		return nil, err
	}

	tags, err := h.getAllTags()
	if err != nil {
		return nil, err
	}

	return &services.SitemapData{
		Articles:   articles,
		Categories: categories,
		Tags:       tags,
		LastUpdate: time.Now(),
	}, nil
}

func (h *SEOHandlers) getPublishedArticles() ([]models.Article, error) {
	// This would use the actual article service
	// For now, return mock data
	return []models.Article{
		{
			ID:          1,
			Title:       "Sample Article 1",
			Slug:        "sample-article-1",
			Content:     "Sample content for article 1",
			Excerpt:     "Sample excerpt",
			Status:      "published",
			PublishedAt: &time.Time{},
			UpdatedAt:   time.Now(),
			LanguageCode: "en",
		},
	}, nil
}

func (h *SEOHandlers) getRecentArticles(duration time.Duration) ([]models.Article, error) {
	articles, err := h.getPublishedArticles()
	if err != nil {
		return nil, err
	}

	cutoff := time.Now().Add(-duration)
	var recent []models.Article

	for _, article := range articles {
		if article.PublishedAt != nil && article.PublishedAt.After(cutoff) {
			recent = append(recent, article)
		}
	}

	return recent, nil
}

func (h *SEOHandlers) getAllCategories() ([]models.Category, error) {
	// This would use the actual category service
	// For now, return mock data
	return []models.Category{
		{
			ID:           1,
			Name:         "Technology",
			Slug:         "technology",
			Description:  "Technology news and updates",
			UpdatedAt:    time.Now(),
			LanguageCode: "en",
		},
	}, nil
}

func (h *SEOHandlers) getAllTags() ([]models.Tag, error) {
	// This would use the actual tag service
	// For now, return mock data
	return []models.Tag{
		{
			ID:           1,
			Name:         "golang",
			Slug:         "golang",
			Description:  "Go programming language",
			Keywords:     []string{"go", "golang", "programming"},
			UpdatedAt:    time.Now(),
			LanguageCode: "en",
		},
	}, nil
}

func (h *SEOHandlers) getArticleBySlug(slug string) (*models.Article, error) {
	// This would use the actual article service
	// For now, return mock data
	now := time.Now()
	return &models.Article{
		ID:          1,
		Title:       "Sample Article: " + slug,
		Slug:        slug,
		Content:     "This is sample content for the article about " + slug,
		Excerpt:     "Sample excerpt for " + slug,
		Status:      "published",
		PublishedAt: &now,
		UpdatedAt:   now,
		LanguageCode: "en",
		SEOData: models.SEOData{
			MetaTitle:       "Sample Article: " + slug,
			MetaDescription: "Read about " + slug + " in this comprehensive article",
			Keywords:        []string{slug, "news", "article"},
			SchemaType:      "NewsArticle",
		},
	}, nil
}

func (h *SEOHandlers) getCategoryBySlug(slug string) (*models.Category, error) {
	// This would use the actual category service
	// For now, return mock data
	return &models.Category{
		ID:           1,
		Name:         strings.Title(slug),
		Slug:         slug,
		Description:  "Category for " + slug + " related content",
		UpdatedAt:    time.Now(),
		LanguageCode: "en",
	}, nil
}

func (h *SEOHandlers) getTagBySlug(slug string) (*models.Tag, error) {
	// This would use the actual tag service
	// For now, return mock data
	return &models.Tag{
		ID:           1,
		Name:         slug,
		Slug:         slug,
		Description:  "Tag for " + slug + " related content",
		Keywords:     []string{slug},
		UpdatedAt:    time.Now(),
		LanguageCode: "en",
	}, nil
}

func (h *SEOHandlers) getBaseURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, c.Request.Host)
}

// Additional SEO handlers

func (h *SEOHandlers) handleHomepageMetaTags(c *gin.Context) {
	meta, err := h.seoService.GenerateMetaTags("homepage", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating homepage meta tags"})
		return
	}

	c.JSON(http.StatusOK, meta)
}

func (h *SEOHandlers) handleSearchBreadcrumbs(c *gin.Context) {
	query := c.Query("q")
	
	breadcrumbs, err := h.breadcrumbService.GenerateSearchBreadcrumbs(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating search breadcrumbs"})
		return
	}

	c.JSON(http.StatusOK, breadcrumbs)
}

func (h *SEOHandlers) handleValidateSchema(c *gin.Context) {
	var schemaData map[string]interface{}
	if err := c.ShouldBindJSON(&schemaData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schema data"})
		return
	}

	// Basic schema validation
	if _, ok := schemaData["@context"]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing @context in schema"})
		return
	}

	if _, ok := schemaData["@type"]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing @type in schema"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true, "message": "Schema is valid"})
}

func (h *SEOHandlers) handleValidateMeta(c *gin.Context) {
	var metaData struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Keywords    string `json:"keywords"`
	}

	if err := c.ShouldBindJSON(&metaData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid meta data"})
		return
	}

	issues := []string{}

	// Validate title length
	if len(metaData.Title) == 0 {
		issues = append(issues, "Title is required")
	} else if len(metaData.Title) > 60 {
		issues = append(issues, "Title should be 60 characters or less")
	}

	// Validate description length
	if len(metaData.Description) == 0 {
		issues = append(issues, "Description is required")
	} else if len(metaData.Description) > 160 {
		issues = append(issues, "Description should be 160 characters or less")
	}

	// Validate keywords
	if metaData.Keywords != "" {
		keywords := strings.Split(metaData.Keywords, ",")
		if len(keywords) > 10 {
			issues = append(issues, "Too many keywords (max 10 recommended)")
		}
	}

	if len(issues) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"valid": false, "issues": issues})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true, "message": "Meta tags are valid"})
}

func (h *SEOHandlers) handleValidateBreadcrumbs(c *gin.Context) {
	var breadcrumbData services.BreadcrumbList
	if err := c.ShouldBindJSON(&breadcrumbData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid breadcrumb data"})
		return
	}

	if err := h.breadcrumbService.ValidateBreadcrumbs(&breadcrumbData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"valid": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true, "message": "Breadcrumbs are valid"})
}

func (h *SEOHandlers) handleOptimizeTitle(c *gin.Context) {
	slug := c.Param("slug")
	
	article, err := h.getArticleBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})
		return
	}

	// Generate optimized title suggestions
	suggestions := []string{}
	
	// Original title
	originalTitle := fmt.Sprintf("%s - %s", article.Title, "High Performance News")
	
	// Shortened version if too long
	if len(originalTitle) > 60 {
		if len(article.Title) <= 57 {
			suggestions = append(suggestions, article.Title)
		} else {
			truncated := h.truncateText(article.Title, 57)
			suggestions = append(suggestions, truncated)
		}
	} else {
		suggestions = append(suggestions, originalTitle)
	}

	// Alternative formats
	if len(article.Tags) > 0 {
		tagTitle := fmt.Sprintf("%s | %s", article.Title, article.Tags[0].Name)
		if len(tagTitle) <= 60 {
			suggestions = append(suggestions, tagTitle)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"original":    originalTitle,
		"suggestions": suggestions,
		"current_length": len(originalTitle),
		"optimal_length": "50-60 characters",
	})
}

func (h *SEOHandlers) handleOptimizeDescription(c *gin.Context) {
	slug := c.Param("slug")
	
	article, err := h.getArticleBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})
		return
	}

	suggestions := []string{}
	
	// Use existing excerpt if good length
	if article.Excerpt != "" {
		if len(article.Excerpt) <= 160 {
			suggestions = append(suggestions, article.Excerpt)
		} else {
			truncated := h.truncateText(article.Excerpt, 157)
			suggestions = append(suggestions, truncated)
		}
	}

	// Generate from content if no excerpt
	if article.Excerpt == "" {
		contentDesc := h.stripHTML(article.Content)
		truncated := h.truncateText(contentDesc, 157)
		suggestions = append(suggestions, truncated)
	}

	// Add call-to-action version
	if len(suggestions) > 0 {
		ctaVersion := suggestions[0]
		if len(ctaVersion) <= 145 { // Leave room for CTA
			ctaVersion += " Read more..."
			suggestions = append(suggestions, ctaVersion)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"original":       article.Excerpt,
		"suggestions":    suggestions,
		"current_length": len(article.Excerpt),
		"optimal_length": "150-160 characters",
	})
}

func (h *SEOHandlers) handleAnalyzeKeywords(c *gin.Context) {
	slug := c.Param("slug")
	
	article, err := h.getArticleBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})
		return
	}

	analysis := map[string]interface{}{
		"seo_keywords":     article.SEOData.Keywords,
		"tag_keywords":     []string{},
		"content_keywords": h.extractKeywordsFromContent(article.Content),
		"recommendations":  []string{},
	}

	// Collect tag keywords
	var tagKeywords []string
	for _, tag := range article.Tags {
		tagKeywords = append(tagKeywords, tag.Name)
		tagKeywords = append(tagKeywords, tag.Keywords...)
	}
	analysis["tag_keywords"] = tagKeywords

	// Generate recommendations
	recommendations := []string{}
	
	if len(article.SEOData.Keywords) == 0 {
		recommendations = append(recommendations, "Add SEO keywords to improve search visibility")
	}
	
	if len(article.SEOData.Keywords) > 10 {
		recommendations = append(recommendations, "Consider reducing keywords to 5-10 for better focus")
	}
	
	if len(article.Tags) == 0 {
		recommendations = append(recommendations, "Add relevant tags to improve categorization")
	}
	
	analysis["recommendations"] = recommendations

	c.JSON(http.StatusOK, analysis)
}

// Helper methods for new handlers

func (h *SEOHandlers) truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	
	// Find the last space before maxLength to avoid cutting words
	truncated := text[:maxLength]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > 0 {
		truncated = truncated[:lastSpace]
	}
	
	return truncated + "..."
}

func (h *SEOHandlers) stripHTML(html string) string {
	// Simple HTML tag removal - in production, use a proper HTML parser
	result := html
	for strings.Contains(result, "<") && strings.Contains(result, ">") {
		start := strings.Index(result, "<")
		end := strings.Index(result[start:], ">")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+1:]
	}
	return strings.TrimSpace(result)
}

func (h *SEOHandlers) extractKeywordsFromContent(content string) []string {
	// Simple keyword extraction - in production, use NLP libraries
	text := h.stripHTML(content)
	words := strings.Fields(strings.ToLower(text))
	
	// Filter common words and short words
	commonWords := map[string]bool{
		"the": true, "and": true, "or": true, "but": true, "in": true,
		"on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true,
		"were": true, "be": true, "been": true, "have": true, "has": true,
		"had": true, "do": true, "does": true, "did": true, "will": true,
		"would": true, "could": true, "should": true, "may": true, "might": true,
		"can": true, "this": true, "that": true, "these": true, "those": true,
		"a": true, "an": true, "as": true, "if": true, "then": true,
	}
	
	wordCount := make(map[string]int)
	for _, word := range words {
		if len(word) > 3 && !commonWords[word] {
			wordCount[word]++
		}
	}
	
	// Get top keywords
	var keywords []string
	for word, count := range wordCount {
		if count >= 2 { // Word appears at least twice
			keywords = append(keywords, word)
		}
		if len(keywords) >= 10 {
			break
		}
	}
	
	return keywords
}