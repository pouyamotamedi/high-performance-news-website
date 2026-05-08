package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/repositories"
	"high-performance-news-website/pkg/cache"
)

// StaticGenerator handles generation of static HTML files for maximum performance
type StaticGenerator struct {
	templates    *templateSet
	outputPath   string
	cacheService cache.CacheService
	articleRepo  *repositories.ArticleRepository
	categoryRepo *repositories.CategoryRepository
	tagRepo      *repositories.TagRepository
	mediaService *MediaService
	baseURL      string
	siteName     string
}

// StaticGeneratorConfig holds configuration for static generation
type StaticGeneratorConfig struct {
	OutputPath   string
	TemplatesDir string
	BaseURL      string
	SiteName     string
}

// PageData represents common data for all pages (internal use)
type PageData struct {
	Title        string
	Description  string
	Keywords     []string
	CanonicalURL string
	SchemaMarkup string
	Language     string
	Direction    string // "ltr" or "rtl"
	BaseURL      string
}

// ArticleForTemplate wraps models.Article with additional fields needed by templates
type ArticleForTemplate struct {
	*models.Article
	Category     *models.Category     // Single category for template compatibility
	Author       *AuthorData          // Author data for template
	CommentCount int                  // Number of comments
	ReadTime     int                  // Estimated read time in minutes
	ImageData    *ResponsiveImageData // Responsive image data for featured image
}

// AuthorData represents author information for templates
type AuthorData struct {
	ID        uint64
	Name      string
	FirstName string
	LastName  string
	Avatar    string
	Bio       string
	Username  string
}

// TemplateData represents the data structure expected by existing templates
// This matches the structure returned by createBaseTemplateData() in server.go
type TemplateData struct {
	// Base template fields (from base.html)
	SiteName          string
	SiteDescription   string
	LanguageCode      string
	LanguageDirection string
	ThemeMode         string
	CurrentYear       int
	Navigation        []NavigationItem
	IsAuthenticated   bool
	OGType            string
	TwitterCard       string

	// Header/branding fields (required by header.html)
	HeaderStyle  string
	LogoURL      string
	ShowSiteName bool
	ThemeConfig  map[string]interface{}

	// SEO fields
	Title          string
	SEOTitle       string
	SEODescription string
	SEOKeywords    []string
	Description    string
	CanonicalURL   string
	AlternateURLs  map[string]string

	// Open Graph fields
	OGTitle       string
	OGDescription string
	OGURL         string
	OGImage       string

	// Twitter fields
	TwitterTitle       string
	TwitterDescription string
	TwitterImage       string

	// Structured data
	StructuredData   string
	PreloadResources []PreloadResource

	// Hero image for preload tag
	HeroImage string

	// Analytics
	GoogleAnalyticsID string

	// Page-specific fields
	PageType string
	Article  *ArticleForTemplate
	Articles []ArticleTemplateData
	Category *models.Category
	Tag      *models.Tag

	// Related content
	RelatedArticles  []ArticleTemplateData
	TrendingArticles []ArticleTemplateData
	PopularArticles  []ArticleTemplateData
	CategoryArticles []ArticleTemplateData
	Categories       []CategoryTemplateData
	PopularTags      []TagTemplateData

	// Pagination
	Pagination PaginationData

	// Breadcrumbs
	Breadcrumbs      *BreadcrumbData
	BreadcrumbSchema string // JSON-LD schema for breadcrumbs

	// Static generation marker
	IsStaticPage bool
	GeneratedAt  time.Time

	// Breaking news
	HasBreakingNews bool
	BreakingNews    []ArticleTemplateData

	// Article navigation
	NextArticle *ArticleTemplateData
}

// NavigationItem represents a navigation menu item
type NavigationItem struct {
	Name     string
	URL      string
	Active   bool
	Icon     string
	Children []NavigationItem
}

// PreloadResource represents a resource to preload
type PreloadResource struct {
	URL  string
	As   string
	Type string
}

// BreadcrumbData represents breadcrumb navigation
type BreadcrumbData struct {
	HTML string
}

// ResponsiveImageData holds all data needed for responsive image rendering
type ResponsiveImageData struct {
	// Primary URLs for each size (WebP preferred, JPEG fallback)
	ThumbnailWebP string
	ThumbnailJPEG string
	SmallWebP     string
	SmallJPEG     string
	MediumWebP    string
	MediumJPEG    string
	LargeWebP     string
	LargeJPEG     string
	// Dimensions for the large size (for width/height attributes)
	Width  int
	Height int
	// LQIP placeholder (base64 data URI)
	LQIP string
	// Alt text
	AltText string
	// Whether this image has variants
	HasVariants bool
}

// ArticleTemplateData represents article data for templates
type ArticleTemplateData struct {
	ID            uint64
	Title         string
	Slug          string
	Excerpt       string
	Author        string
	TimeAgo       string
	ViewCount     uint64
	Views         uint64
	Category      string
	FeaturedImage string
	// Responsive image data with all variants
	ImageData   *ResponsiveImageData
	PublishedAt *time.Time
	ReadTime    int
}

// CategoryTemplateData represents category data for templates
type CategoryTemplateData struct {
	ID           uint64
	Name         string
	Slug         string
	Description  string
	ImageURL     string
	ImageAltText string
	Count        int
}

// TagTemplateData represents tag data for templates
type TagTemplateData struct {
	Name  string
	Slug  string
	Count int
}

// HomepageData represents data for homepage generation
type HomepageData struct {
	PageData
	LatestArticles   []models.Article
	TrendingArticles []models.Article
	Categories       []models.Category
	FeaturedTags     []models.Tag
}

// ArticlePageData represents data for article page generation
type ArticlePageData struct {
	PageData
	Article         models.Article
	RelatedArticles []models.Article
	Category        models.Category
	Tags            []models.Tag
	Author          models.User
}

// CategoryPageData represents data for category page generation
type CategoryPageData struct {
	PageData
	Category   models.Category
	Articles   []models.Article
	Pagination PaginationData
}

// TagPageData represents data for tag page generation
type TagPageData struct {
	PageData
	Tag        models.Tag
	Articles   []models.Article
	Pagination PaginationData
}

// PaginationData represents pagination information
type PaginationData struct {
	CurrentPage  int
	TotalPages   int
	HasPrevious  bool
	HasNext      bool
	PreviousPage int
	NextPage     int
	TotalItems   int
}

// NewStaticGenerator creates a new static generator instance
func NewStaticGenerator(config StaticGeneratorConfig, cacheService cache.CacheService,
	articleRepo *repositories.ArticleRepository, categoryRepo *repositories.CategoryRepository,
	tagRepo *repositories.TagRepository, mediaService *MediaService) (*StaticGenerator, error) {

	// Load templates from the main templates directory (same as dynamic pages)
	templates, err := loadTemplates(config.TemplatesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(config.OutputPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	siteName := config.SiteName
	if siteName == "" {
		siteName = "High Performance News"
	}

	return &StaticGenerator{
		templates:    templates,
		outputPath:   config.OutputPath,
		cacheService: cacheService,
		articleRepo:  articleRepo,
		categoryRepo: categoryRepo,
		tagRepo:      tagRepo,
		mediaService: mediaService,
		baseURL:      config.BaseURL,
		siteName:     siteName,
	}, nil
}

// buildResponsiveImageData builds ResponsiveImageData from an image ID
func (sg *StaticGenerator) buildResponsiveImageData(imageID uint64, altText string) *ResponsiveImageData {
	if sg.mediaService == nil || imageID == 0 {
		return nil
	}

	variants, err := sg.mediaService.GetImageVariants(imageID)
	if err != nil || len(variants) == 0 {
		return nil
	}

	data := &ResponsiveImageData{
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

// createBaseTemplateData creates the base template data structure that matches dynamic templates
func (sg *StaticGenerator) createBaseTemplateData(language string, currentPath string) TemplateData {
	direction := getTextDirection(language)

	// Get breaking news articles
	breakingNews := sg.getBreakingNewsArticles()

	// Get theme config from database if available
	themeConfig := sg.getActiveThemeConfig()

	// Use theme branding or fallback to defaults
	siteName := sg.siteName
	siteDescription := "High-performance multilingual news website"
	logoURL := "/static/images/logo.svg"
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

	return TemplateData{
		SiteName:          siteName,
		SiteDescription:   siteDescription,
		LogoURL:           logoURL,
		ShowSiteName:      showSiteName,
		HeaderStyle:       headerStyle,
		ThemeConfig:       themeConfig,
		LanguageCode:      language,
		LanguageDirection: direction,
		ThemeMode:         "auto",
		CurrentYear:       time.Now().Year(),
		Navigation: []NavigationItem{
			{Name: "Home", URL: "/", Active: currentPath == "/"},
			{Name: "Latest", URL: "/latest", Active: currentPath == "/latest"},
			{Name: "Trending", URL: "/trending", Active: currentPath == "/trending"},
			{Name: "Categories", URL: "/categories", Active: currentPath == "/categories"},
			{Name: "Tags", URL: "/tags", Active: currentPath == "/tags"},
			{Name: "About", URL: "/about", Active: currentPath == "/about"},
			{Name: "Contact", URL: "/contact", Active: currentPath == "/contact"},
		},
		IsAuthenticated:  false,
		OGType:           "website",
		TwitterCard:      "summary_large_image",
		IsStaticPage:     true,
		GeneratedAt:      time.Now(),
		AlternateURLs:    make(map[string]string),
		PreloadResources: []PreloadResource{},
		HasBreakingNews:  len(breakingNews) > 0,
		BreakingNews:     breakingNews,
	}
}

// getActiveThemeConfig returns nil for static generation (uses defaults)
// Theme config is primarily used for dynamic pages; static pages use sensible defaults
func (sg *StaticGenerator) getActiveThemeConfig() map[string]interface{} {
	// For static generation, we use default values
	// This avoids needing a database connection in the static generator
	return nil
}

// getBreakingNewsArticles fetches articles tagged with "breaking-news"
func (sg *StaticGenerator) getBreakingNewsArticles() []ArticleTemplateData {
	if sg.tagRepo == nil || sg.articleRepo == nil {
		return nil
	}

	ctx := context.Background()

	// Get the "breaking-news" tag
	tag, err := sg.tagRepo.GetBySlug("breaking-news", "en")
	if err != nil || tag == nil {
		return nil
	}

	// Get articles with this tag (limit to 10 most recent)
	articles, err := sg.articleRepo.GetByTag(ctx, tag.ID, 10, 0)
	if err != nil || len(articles) == 0 {
		return nil
	}

	// Convert to template data
	result := make([]ArticleTemplateData, len(articles))
	for i, article := range articles {
		result[i] = sg.convertArticleToTemplateData(article)
	}

	return result
}

// convertArticleToTemplateData converts a models.Article to ArticleTemplateData
func (sg *StaticGenerator) convertArticleToTemplateData(article models.Article) ArticleTemplateData {
	// Author name would need to be fetched separately or passed in
	// For now, use a placeholder
	authorName := "Author"

	// Build responsive image data if article has a featured image
	var imageData *ResponsiveImageData
	if article.FeaturedImageID != nil && *article.FeaturedImageID > 0 {
		imageData = sg.buildResponsiveImageData(*article.FeaturedImageID, article.Title)
	}

	// Calculate read time (roughly 200 words per minute)
	wordCount := len(strings.Fields(article.Content))
	readTime := wordCount / 200
	if readTime < 1 {
		readTime = 1
	}

	return ArticleTemplateData{
		ID:            article.ID,
		Title:         article.Title,
		Slug:          article.Slug,
		Excerpt:       article.Excerpt,
		Author:        authorName,
		TimeAgo:       formatTimeAgoStatic(article.PublishedAt),
		ViewCount:     article.ViewCount,
		Views:         article.ViewCount,
		Category:      getCategoryName(article),
		FeaturedImage: article.FeaturedImage,
		ImageData:     imageData,
		PublishedAt:   article.PublishedAt,
		ReadTime:      readTime,
	}
}

// formatTimeAgoStatic formats time for static pages
func formatTimeAgoStatic(t *time.Time) string {
	if t == nil {
		return "Recently"
	}

	now := time.Now()
	diff := now.Sub(*t)

	switch {
	case diff < time.Minute:
		return "Just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("January 2, 2006")
	}
}

// getCategoryName extracts category name from article
func getCategoryName(article models.Article) string {
	if len(article.Categories) > 0 {
		return article.Categories[0].Name
	}
	return "General"
}

// GenerateHomepage generates static HTML for the homepage
func (sg *StaticGenerator) GenerateHomepage(ctx context.Context, language string) error {
	// Get latest articles (handle nil repository)
	var latestArticles []models.Article
	if sg.articleRepo != nil {
		articles, err := sg.articleRepo.GetLatestArticles(ctx, 20)
		if err != nil {
			return fmt.Errorf("failed to get latest articles: %w", err)
		}
		latestArticles = articles
	}

	// Get trending articles (handle nil repository)
	var trendingArticles []models.Article
	if sg.articleRepo != nil {
		articles, err := sg.articleRepo.GetTrendingArticles(ctx, 10, 24)
		if err != nil {
			return fmt.Errorf("failed to get trending articles: %w", err)
		}
		trendingArticles = articles
	}

	// Get categories
	var categories []models.Category
	if sg.categoryRepo != nil {
		cats, err := sg.categoryRepo.GetAll()
		if err == nil {
			categories = cats
		}
	}

	// Create base template data matching dynamic templates
	data := sg.createBaseTemplateData(language, "/")
	data.Title = sg.siteName
	data.PageType = "homepage"
	data.SEOTitle = sg.siteName + " - Latest News and Updates"
	data.SEODescription = "Stay updated with the latest news and trending stories from around the world."
	data.SEOKeywords = []string{"news", "latest", "trending", "updates"}
	data.CanonicalURL = sg.baseURL + "/"
	data.OGType = "website"

	// Convert articles to template format
	articleData := make([]ArticleTemplateData, len(latestArticles))
	for i, article := range latestArticles {
		articleData[i] = sg.convertArticleToTemplateData(article)
	}
	data.Articles = articleData

	// Convert trending articles
	trendingData := make([]ArticleTemplateData, len(trendingArticles))
	for i, article := range trendingArticles {
		trendingData[i] = sg.convertArticleToTemplateData(article)
	}
	data.TrendingArticles = trendingData

	// Convert categories
	categoryData := make([]CategoryTemplateData, len(categories))
	for i, cat := range categories {
		categoryData[i] = CategoryTemplateData{
			ID:           cat.ID,
			Name:         cat.Name,
			Slug:         cat.Slug,
			Description:  cat.Description,
			ImageURL:     cat.GetImageURL(),
			ImageAltText: cat.GetImageAltText(),
		}
	}
	data.Categories = categoryData

	// Add structured data
	data.StructuredData = sg.generateHomepageSchema(latestArticles)

	// Generate HTML using the existing homepage template
	var buf bytes.Buffer
	if err := sg.templates.ExecuteTemplate(&buf, "homepage.html", data); err != nil {
		return fmt.Errorf("failed to execute homepage template: %w", err)
	}

	// Write to file
	outputPath := filepath.Join(sg.outputPath, "index.html")
	if language != "en" { // English is default
		outputPath = filepath.Join(sg.outputPath, language, "index.html")
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("failed to create language directory: %w", err)
		}
	}

	if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write homepage file: %w", err)
	}

	// Warm cache
	go sg.warmHomepageCache(language)

	log.Printf("Generated static homepage for language: %s at %s", language, outputPath)
	return nil
}

// GenerateArticlePage generates static HTML for an article page
func (sg *StaticGenerator) GenerateArticlePage(ctx context.Context, article *models.Article) error {
	// Get related articles
	relatedArticles, err := sg.getRelatedArticles(ctx, article)
	if err != nil {
		log.Printf("Warning: failed to get related articles for article %d: %v", article.ID, err)
		relatedArticles = []models.Article{} // Continue with empty related articles
	}

	// Use article's language for URL (SEO best practice)
	language := article.LanguageCode
	if language == "" {
		language = "en"
	}

	// Get the article's category - resolve to correct language version for SEO
	var articleCategory *models.Category
	if sg.categoryRepo != nil && article.CategoryID > 0 {
		// First get the original category to find its translation group
		originalCat, err := sg.categoryRepo.GetByID(ctx, article.CategoryID)
		if err == nil && originalCat != nil {
			// Try to find the category in the article's language
			translationGroupID := originalCat.ID
			if originalCat.TranslationGroupID != nil {
				translationGroupID = *originalCat.TranslationGroupID
			}
			
			// Query for category in the correct language
			langCat, err := sg.getCategoryByTranslationGroupAndLanguage(ctx, translationGroupID, language)
			if err == nil && langCat != nil {
				articleCategory = langCat
			} else {
				// Fallback to original category
				articleCategory = originalCat
			}
		}
	}
	
	// If still no category, check article.Categories
	if articleCategory == nil && len(article.Categories) > 0 {
		articleCategory = &article.Categories[0]
	}
	
	// Fallback to a default category if none found
	if articleCategory == nil {
		articleCategory = &models.Category{Name: "General", Slug: "general"}
	}

	// Create base template data matching dynamic templates
	articlePath := "/" + language + "/article/" + article.Slug
	data := sg.createBaseTemplateData(language, articlePath)

	// Set article-specific SEO data
	data.Title = article.Title
	data.PageType = "article"

	if article.SEOData.MetaTitle != "" {
		data.SEOTitle = article.SEOData.MetaTitle
	} else {
		data.SEOTitle = article.Title + " - " + sg.siteName
	}

	if article.SEOData.MetaDescription != "" {
		data.SEODescription = article.SEOData.MetaDescription
	} else {
		data.SEODescription = article.Excerpt
	}

	data.SEOKeywords = article.SEOData.Keywords
	data.CanonicalURL = fmt.Sprintf("%s/%s/article/%s", sg.baseURL, language, article.Slug)
	data.OGType = "article"

	// Open Graph data
	data.OGTitle = data.SEOTitle
	data.OGDescription = data.SEODescription
	data.OGURL = data.CanonicalURL
	if article.FeaturedImage != "" {
		data.OGImage = article.FeaturedImage
		data.TwitterImage = article.FeaturedImage
		// Set HeroImage for preload tag in base template
		data.HeroImage = article.FeaturedImage
	}

	// Twitter data
	data.TwitterTitle = data.SEOTitle
	data.TwitterDescription = data.SEODescription

	// Calculate read time (roughly 200 words per minute)
	wordCount := len(strings.Fields(article.Content))
	readTime := wordCount / 200
	if readTime < 1 {
		readTime = 1
	}

	// Build responsive image data for the article's featured image
	var articleImageData *ResponsiveImageData
	if article.FeaturedImageID != nil && *article.FeaturedImageID > 0 {
		articleImageData = sg.buildResponsiveImageData(*article.FeaturedImageID, article.Title)

		// If FeaturedImage URL is not set, try to get it from the image record
		if article.FeaturedImage == "" && sg.mediaService != nil {
			if img, err := sg.mediaService.GetImageByID(*article.FeaturedImageID); err == nil && img != nil {
				article.FeaturedImage = img.OriginalURL
			}
		}
	}

	// Set the article with category wrapper
	data.Article = &ArticleForTemplate{
		Article:  article,
		Category: articleCategory,
		Author: &AuthorData{
			ID:        article.AuthorID,
			Name:      "Author",
			FirstName: "Author",
			LastName:  "",
			Avatar:    "",
			Bio:       "",
			Username:  "",
		},
		CommentCount: 0, // Would need to fetch from comment repo
		ReadTime:     readTime,
		ImageData:    articleImageData,
	}

	// Convert related articles
	relatedData := make([]ArticleTemplateData, len(relatedArticles))
	for i, rel := range relatedArticles {
		relatedData[i] = sg.convertArticleToTemplateData(rel)
	}
	data.RelatedArticles = relatedData

	// Get popular articles for sidebar
	var popularArticles []models.Article
	if sg.articleRepo != nil {
		articles, err := sg.articleRepo.GetTrendingArticles(ctx, 5, 168) // Last 7 days
		if err == nil {
			popularArticles = articles
		}
	}
	popularData := make([]ArticleTemplateData, len(popularArticles))
	for i, pop := range popularArticles {
		popularData[i] = sg.convertArticleToTemplateData(pop)
	}
	data.PopularArticles = popularData

	// Get category articles for sidebar (articles from same category)
	var categoryArticles []models.Article
	if sg.articleRepo != nil && article.CategoryID > 0 {
		articles, err := sg.articleRepo.GetByCategory(ctx, article.CategoryID, 5, 0)
		if err == nil {
			// Filter out current article
			for _, a := range articles {
				if a.ID != article.ID {
					categoryArticles = append(categoryArticles, a)
				}
			}
		}
	}
	categoryData := make([]ArticleTemplateData, len(categoryArticles))
	for i, cat := range categoryArticles {
		categoryData[i] = sg.convertArticleToTemplateData(cat)
	}
	data.CategoryArticles = categoryData

	// Add structured data - pass the language-resolved category
	data.StructuredData = sg.generateArticleSchema(article, articleCategory)

	// Generate breadcrumbs HTML
	breadcrumbHTML := sg.generateArticleBreadcrumbs(article)
	data.Breadcrumbs = &BreadcrumbData{HTML: breadcrumbHTML}

	// Generate BreadcrumbList JSON-LD schema
	data.BreadcrumbSchema = sg.generateBreadcrumbSchema(article)

	// Get next article for navigation
	if sg.articleRepo != nil {
		nextArticles, err := sg.articleRepo.GetLatestArticles(ctx, 2)
		if err == nil && len(nextArticles) > 0 {
			// Find an article that's not the current one
			for _, nextArt := range nextArticles {
				if nextArt.ID != article.ID {
					nextData := sg.convertArticleToTemplateData(nextArt)
					data.NextArticle = &nextData
					break
				}
			}
		}
	}

	// Generate HTML using the existing article template
	var buf bytes.Buffer
	if err := sg.templates.ExecuteTemplate(&buf, "article.html", data); err != nil {
		return fmt.Errorf("failed to execute article template: %w", err)
	}

	// Create directory structure: static-html/articles/{slug}/index.html
	articleDir := filepath.Join(sg.outputPath, "articles", article.Slug)
	if err := os.MkdirAll(articleDir, 0755); err != nil {
		return fmt.Errorf("failed to create article directory: %w", err)
	}

	// Write to file
	outputPath := filepath.Join(articleDir, "index.html")
	if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write article file: %w", err)
	}

	// Warm cache
	go sg.warmArticleCache(article)

	log.Printf("Generated static article page: %s at %s", article.Slug, outputPath)
	return nil
}

// generateArticleBreadcrumbs generates breadcrumb HTML for an article
func (sg *StaticGenerator) generateArticleBreadcrumbs(article *models.Article) string {
	var breadcrumbs []string
	breadcrumbs = append(breadcrumbs, fmt.Sprintf(`<a href="/">Home</a>`))

	if len(article.Categories) > 0 {
		cat := article.Categories[0]
		// Use category's language for URL (SEO best practice)
		catLang := cat.LanguageCode
		if catLang == "" {
			catLang = "en"
		}
		breadcrumbs = append(breadcrumbs, fmt.Sprintf(`<a href="/%s/category/%s">%s</a>`, catLang, cat.Slug, cat.Name))
	}

	breadcrumbs = append(breadcrumbs, fmt.Sprintf(`<span>%s</span>`, article.Title))

	return `<nav class="breadcrumb" aria-label="Breadcrumb">` + strings.Join(breadcrumbs, ` <span class="separator">/</span> `) + `</nav>`
}

// generateBreadcrumbSchema generates JSON-LD BreadcrumbList schema for an article
func (sg *StaticGenerator) generateBreadcrumbSchema(article *models.Article) string {
	type ListItem struct {
		Type     string `json:"@type"`
		Position int    `json:"position"`
		Name     string `json:"name"`
		Item     string `json:"item"`
	}

	type BreadcrumbList struct {
		Context         string     `json:"@context"`
		Type            string     `json:"@type"`
		ItemListElement []ListItem `json:"itemListElement"`
	}

	items := []ListItem{
		{Type: "ListItem", Position: 1, Name: "Home", Item: sg.baseURL},
	}
	position := 2

	// Add category if available
	if len(article.Categories) > 0 {
		cat := article.Categories[0]
		// Use category's language for URL
		catLang := cat.LanguageCode
		if catLang == "" {
			catLang = "en"
		}
		items = append(items, ListItem{
			Type:     "ListItem",
			Position: position,
			Name:     cat.Name,
			Item:     fmt.Sprintf("%s/%s/category/%s", sg.baseURL, catLang, cat.Slug),
		})
		position++
	}

	// Use article's language for URL (SEO best practice)
	articleLang := article.LanguageCode
	if articleLang == "" {
		articleLang = "en"
	}

	// Add current article
	items = append(items, ListItem{
		Type:     "ListItem",
		Position: position,
		Name:     article.Title,
		Item:     fmt.Sprintf("%s/%s/article/%s", sg.baseURL, articleLang, article.Slug),
	})

	schema := BreadcrumbList{
		Context:         "https://schema.org",
		Type:            "BreadcrumbList",
		ItemListElement: items,
	}

	jsonBytes, err := json.Marshal(schema)
	if err != nil {
		log.Printf("Warning: failed to marshal breadcrumb schema: %v", err)
		return ""
	}

	return string(jsonBytes)
}

// GenerateCategoryPage generates static HTML for a category page
func (sg *StaticGenerator) GenerateCategoryPage(ctx context.Context, category *models.Category, page int) error {
	// Get articles for this category with pagination
	limit := 20
	offset := (page - 1) * limit
	articles, err := sg.articleRepo.GetByCategory(ctx, category.ID, limit, offset)
	if err != nil {
		return fmt.Errorf("failed to get articles for category: %w", err)
	}

	// Get total count for pagination
	totalCount, err := sg.articleRepo.CountByCategory(ctx, category.ID)
	if err != nil {
		log.Printf("Warning: failed to get total count for category %d: %v", category.ID, err)
		totalCount = int64(len(articles)) // Fallback to current articles count
	}

	// Calculate pagination
	totalPages := int((totalCount + int64(limit) - 1) / int64(limit))
	pagination := PaginationData{
		CurrentPage:  page,
		TotalPages:   totalPages,
		HasPrevious:  page > 1,
		HasNext:      page < totalPages,
		PreviousPage: page - 1,
		NextPage:     page + 1,
		TotalItems:   int(totalCount),
	}

	// Create base template data
	// Use category's language for URL (SEO best practice)
	catLang := category.LanguageCode
	if catLang == "" {
		catLang = "en"
	}
	
	categoryPath := fmt.Sprintf("/%s/category/%s", catLang, category.Slug)
	data := sg.createBaseTemplateData(catLang, categoryPath)

	// Set category-specific data
	data.Title = category.Name
	data.PageType = "category"
	data.SEOTitle = fmt.Sprintf("%s - Category - %s", category.Name, sg.siteName)
	data.SEODescription = category.Description
	data.SEOKeywords = []string{category.Name, "category", "articles"}
	data.CanonicalURL = fmt.Sprintf("%s/%s/category/%s", sg.baseURL, catLang, category.Slug)

	if page > 1 {
		data.SEOTitle = fmt.Sprintf("%s - Page %d - Category - %s", category.Name, page, sg.siteName)
		data.CanonicalURL = fmt.Sprintf("%s/%s/category/%s/page-%d", sg.baseURL, catLang, category.Slug, page)
	}

	data.Category = category
	data.Pagination = pagination

	// Convert articles to template format
	articleData := make([]ArticleTemplateData, len(articles))
	for i, article := range articles {
		articleData[i] = sg.convertArticleToTemplateData(article)
	}
	data.Articles = articleData

	// Add structured data
	data.StructuredData = sg.generateCategorySchema(category, articles)

	// Generate HTML using the existing category template
	var buf bytes.Buffer
	if err := sg.templates.ExecuteTemplate(&buf, "category.html", data); err != nil {
		return fmt.Errorf("failed to execute category template: %w", err)
	}

	// Create directory structure
	categoryDir := filepath.Join(sg.outputPath, "categories", category.Slug)
	if err := os.MkdirAll(categoryDir, 0755); err != nil {
		return fmt.Errorf("failed to create category directory: %w", err)
	}

	// Write to file
	var outputPath string
	if page == 1 {
		outputPath = filepath.Join(categoryDir, "index.html")
	} else {
		outputPath = filepath.Join(categoryDir, fmt.Sprintf("page-%d.html", page))
	}

	if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write category file: %w", err)
	}

	// Warm cache
	go sg.warmCategoryCache(category, page)

	log.Printf("Generated static category page: %s (page %d) at %s", category.Slug, page, outputPath)
	return nil
}

// GenerateTagPage generates static HTML for a tag page
func (sg *StaticGenerator) GenerateTagPage(ctx context.Context, tag *models.Tag, page int) error {
	// Get articles for this tag with pagination
	limit := 20
	offset := (page - 1) * limit
	articles, err := sg.articleRepo.GetByTag(ctx, tag.ID, limit, offset)
	if err != nil {
		return fmt.Errorf("failed to get articles for tag: %w", err)
	}

	// Get total count for pagination
	totalCount, err := sg.articleRepo.CountByTag(ctx, tag.ID)
	if err != nil {
		log.Printf("Warning: failed to get total count for tag %d: %v", tag.ID, err)
		totalCount = int64(len(articles)) // Fallback to current articles count
	}

	// Calculate pagination
	totalPages := int((totalCount + int64(limit) - 1) / int64(limit))
	pagination := PaginationData{
		CurrentPage:  page,
		TotalPages:   totalPages,
		HasPrevious:  page > 1,
		HasNext:      page < totalPages,
		PreviousPage: page - 1,
		NextPage:     page + 1,
		TotalItems:   int(totalCount),
	}

	// Create base template data
	// Use tag's language for URL (SEO best practice)
	tagLang := tag.LanguageCode
	if tagLang == "" {
		tagLang = "en"
	}
	tagPath := "/" + tagLang + "/tag/" + tag.Slug
	language := tagLang
	data := sg.createBaseTemplateData(language, tagPath)

	// Set tag-specific data
	data.Title = tag.Name
	data.PageType = "tag"
	data.SEOTitle = fmt.Sprintf("%s - Tag - %s", tag.Name, sg.siteName)
	data.SEODescription = tag.Description
	data.SEOKeywords = append([]string{tag.Name, "tag", "articles"}, tag.Keywords...)
	data.CanonicalURL = fmt.Sprintf("%s/%s/tag/%s", sg.baseURL, tagLang, tag.Slug)

	if page > 1 {
		data.SEOTitle = fmt.Sprintf("%s - Page %d - Tag - %s", tag.Name, page, sg.siteName)
		data.CanonicalURL = fmt.Sprintf("%s/%s/tag/%s/page-%d", sg.baseURL, tagLang, tag.Slug, page)
	}

	data.Tag = tag
	data.Pagination = pagination

	// Convert articles to template format
	articleData := make([]ArticleTemplateData, len(articles))
	for i, article := range articles {
		articleData[i] = sg.convertArticleToTemplateData(article)
	}
	data.Articles = articleData

	// Add structured data
	data.StructuredData = sg.generateTagSchema(tag, articles)

	// Generate HTML using the existing tag template
	var buf bytes.Buffer
	if err := sg.templates.ExecuteTemplate(&buf, "tag.html", data); err != nil {
		return fmt.Errorf("failed to execute tag template: %w", err)
	}

	// Create directory structure
	tagDir := filepath.Join(sg.outputPath, "tags", tag.Slug)
	if err := os.MkdirAll(tagDir, 0755); err != nil {
		return fmt.Errorf("failed to create tag directory: %w", err)
	}

	// Write to file
	var outputPath string
	if page == 1 {
		outputPath = filepath.Join(tagDir, "index.html")
	} else {
		outputPath = filepath.Join(tagDir, fmt.Sprintf("page-%d.html", page))
	}

	if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write tag file: %w", err)
	}

	// Warm cache
	go sg.warmTagCache(tag, page)

	log.Printf("Generated static tag page: %s (page %d) at %s", tag.Slug, page, outputPath)
	return nil
}

// RegenerateOnContentUpdate regenerates static files when content is updated
func (sg *StaticGenerator) RegenerateOnContentUpdate(ctx context.Context, article *models.Article) error {
	// Regenerate article page
	if err := sg.GenerateArticlePage(ctx, article); err != nil {
		return fmt.Errorf("failed to regenerate article page: %w", err)
	}

	// Regenerate homepage for default language (English)
	if err := sg.GenerateHomepage(ctx, "en"); err != nil {
		log.Printf("Warning: failed to regenerate homepage: %v", err)
	}

	// Regenerate category pages (first page only for performance)
	if sg.categoryRepo != nil {
		category, err := sg.categoryRepo.GetByID(ctx, article.CategoryID)
		if err == nil {
			if err := sg.GenerateCategoryPage(ctx, category, 1); err != nil {
				log.Printf("Warning: failed to regenerate category page: %v", err)
			}
		}
	}

	// Regenerate tag pages (first page only for performance)
	if sg.tagRepo != nil && len(article.Tags) > 0 {
		for _, tag := range article.Tags {
			if err := sg.GenerateTagPage(ctx, &tag, 1); err != nil {
				log.Printf("Warning: failed to regenerate tag page %s: %v", tag.Slug, err)
			}
		}
	}

	// Invalidate related caches
	sg.invalidateRelatedCaches(article)

	log.Printf("Static regeneration completed for article: %s", article.Slug)
	return nil
}

// Helper functions

// templateSet holds multiple named templates for proper template inheritance
type templateSet struct {
	templates map[string]*template.Template
	funcMap   template.FuncMap
}

func (ts *templateSet) ExecuteTemplate(buf *bytes.Buffer, name string, data interface{}) error {
	// Remove .html suffix if present to get the template name
	templateName := strings.TrimSuffix(name, ".html")

	tmpl, exists := ts.templates[templateName]
	if !exists {
		return fmt.Errorf("template %s not found", templateName)
	}

	// Try to execute with .html suffix first (matches {{define "article.html"}})
	if err := tmpl.ExecuteTemplate(buf, name, data); err != nil {
		// If that fails, try without suffix
		if err2 := tmpl.ExecuteTemplate(buf, templateName, data); err2 != nil {
			return fmt.Errorf("failed to execute template %s: %w (also tried: %v)", name, err, err2)
		}
	}
	return nil
}

func loadTemplates(templatesDir string) (*templateSet, error) {
	// Define template functions
	funcMap := template.FuncMap{
		"formatDate": func(t interface{}) string {
			switch v := t.(type) {
			case time.Time:
				return v.Format("January 2, 2006")
			case *time.Time:
				if v == nil {
					return "Unknown"
				}
				return v.Format("January 2, 2006")
			default:
				return "Unknown"
			}
		},
		"formatDateTime": func(t interface{}) string {
			switch v := t.(type) {
			case time.Time:
				return v.Format("January 2, 2006 at 3:04 PM")
			case *time.Time:
				if v == nil {
					return "Unknown"
				}
				return v.Format("January 2, 2006 at 3:04 PM")
			default:
				return "Unknown"
			}
		},
		"timeAgo": func(t interface{}) string {
			var tm *time.Time
			switch v := t.(type) {
			case time.Time:
				tm = &v
			case *time.Time:
				tm = v
			default:
				return "Unknown"
			}
			if tm == nil {
				return "Unknown"
			}
			now := time.Now()
			diff := now.Sub(*tm)
			switch {
			case diff < time.Minute:
				return "Just now"
			case diff < time.Hour:
				mins := int(diff.Minutes())
				if mins == 1 {
					return "1 minute ago"
				}
				return fmt.Sprintf("%d minutes ago", mins)
			case diff < 24*time.Hour:
				hours := int(diff.Hours())
				if hours == 1 {
					return "1 hour ago"
				}
				return fmt.Sprintf("%d hours ago", hours)
			case diff < 7*24*time.Hour:
				days := int(diff.Hours() / 24)
				if days == 1 {
					return "1 day ago"
				}
				return fmt.Sprintf("%d days ago", days)
			case diff < 30*24*time.Hour:
				weeks := int(diff.Hours() / 24 / 7)
				if weeks == 1 {
					return "1 week ago"
				}
				return fmt.Sprintf("%d weeks ago", weeks)
			default:
				return tm.Format("January 2, 2006")
			}
		},
		"truncate": func(s string, length int) string {
			if len(s) <= length {
				return s
			}
			return s[:length] + "..."
		},
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"safeJS": func(s string) template.JS {
			return template.JS(s)
		},
		"join": func(slice []string, sep string) string {
			return strings.Join(slice, sep)
		},
		// Math functions
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"subtract": func(a, b int) int {
			return a - b
		},
		"mul": func(a, b int) int {
			return a * b
		},
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"mod": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a % b
		},
		// Comparison functions
		"eq": func(a, b interface{}) bool {
			return a == b
		},
		"ne": func(a, b interface{}) bool {
			return a != b
		},
		"lt": func(a, b int) bool {
			return a < b
		},
		"le": func(a, b int) bool {
			return a <= b
		},
		"gt": func(a, b int) bool {
			return a > b
		},
		"ge": func(a, b int) bool {
			return a >= b
		},
		// String functions
		"lower": func(s string) string {
			return strings.ToLower(s)
		},
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"title": func(s string) string {
			return strings.Title(s)
		},
		"contains": func(s, substr string) bool {
			return strings.Contains(s, substr)
		},
		"hasPrefix": func(s, prefix string) bool {
			return strings.HasPrefix(s, prefix)
		},
		"hasSuffix": func(s, suffix string) bool {
			return strings.HasSuffix(s, suffix)
		},
		"replace": func(s, old, new string) string {
			return strings.ReplaceAll(s, old, new)
		},
		// Default value function
		"default": func(defaultVal, val interface{}) interface{} {
			if val == nil || val == "" || val == 0 {
				return defaultVal
			}
			return val
		},
		// Printf function for formatting
		"printf": fmt.Sprintf,
		// Sequence function for pagination
		"seq": func(start, end int) []int {
			if start > end {
				return []int{}
			}
			result := make([]int, end-start+1)
			for i := range result {
				result[i] = start + i
			}
			return result
		},
		// hasResponsiveImage checks if the imageData has valid responsive variants
		"hasResponsiveImage": func(imageData interface{}) bool {
			if imageData == nil {
				return false
			}
			switch v := imageData.(type) {
			case *ResponsiveImageData:
				return v != nil && v.HasVariants
			default:
				return false
			}
		},
		// Responsive image helper - generates <picture> element with WebP and JPEG sources
		"responsiveImage": func(imageData *ResponsiveImageData, lazyLoad bool, cssClass string) template.HTML {
			if imageData == nil || !imageData.HasVariants {
				return template.HTML("")
			}

			var html strings.Builder

			// Wrapper div for LQIP effect (only for lazy loaded images)
			if lazyLoad && imageData.LQIP != "" {
				html.WriteString(`<div class="responsive-image-wrapper" style="position: relative; overflow: hidden;">`)
				html.WriteString(fmt.Sprintf(`<div class="lqip-placeholder" style="position: absolute; inset: 0; background-image: url('%s'); background-size: cover; filter: blur(20px); transform: scale(1.1); transition: opacity 0.3s;"></div>`, imageData.LQIP))
			}

			html.WriteString("<picture>")

			// WebP sources (modern browsers)
			webpSrcset := []string{}
			if imageData.SmallWebP != "" {
				webpSrcset = append(webpSrcset, fmt.Sprintf("%s 300w", imageData.SmallWebP))
			}
			if imageData.MediumWebP != "" {
				webpSrcset = append(webpSrcset, fmt.Sprintf("%s 600w", imageData.MediumWebP))
			}
			if imageData.LargeWebP != "" {
				webpSrcset = append(webpSrcset, fmt.Sprintf("%s 1200w", imageData.LargeWebP))
			}
			if len(webpSrcset) > 0 {
				html.WriteString(fmt.Sprintf(`<source type="image/webp" srcset="%s" sizes="(max-width: 300px) 300px, (max-width: 600px) 600px, (max-width: 1200px) 1200px, 100vw">`, strings.Join(webpSrcset, ", ")))
			}

			// JPEG sources (fallback)
			jpegSrcset := []string{}
			if imageData.SmallJPEG != "" {
				jpegSrcset = append(jpegSrcset, fmt.Sprintf("%s 300w", imageData.SmallJPEG))
			}
			if imageData.MediumJPEG != "" {
				jpegSrcset = append(jpegSrcset, fmt.Sprintf("%s 600w", imageData.MediumJPEG))
			}
			if imageData.LargeJPEG != "" {
				jpegSrcset = append(jpegSrcset, fmt.Sprintf("%s 1200w", imageData.LargeJPEG))
			}
			if len(jpegSrcset) > 0 {
				html.WriteString(fmt.Sprintf(`<source type="image/jpeg" srcset="%s" sizes="(max-width: 300px) 300px, (max-width: 600px) 600px, (max-width: 1200px) 1200px, 100vw">`, strings.Join(jpegSrcset, ", ")))
			}

			// Fallback img element
			fallbackSrc := imageData.MediumJPEG
			if fallbackSrc == "" {
				fallbackSrc = imageData.LargeJPEG
			}

			imgAttrs := []string{
				fmt.Sprintf(`src="%s"`, fallbackSrc),
				fmt.Sprintf(`alt="%s"`, imageData.AltText),
				`decoding="async"`,
				`sizes="(max-width: 300px) 300px, (max-width: 600px) 600px, (max-width: 1200px) 1200px, 100vw"`,
			}

			if imageData.Width > 0 {
				imgAttrs = append(imgAttrs, fmt.Sprintf(`width="%d"`, imageData.Width))
			}
			if imageData.Height > 0 {
				imgAttrs = append(imgAttrs, fmt.Sprintf(`height="%d"`, imageData.Height))
			}
			if cssClass != "" {
				imgAttrs = append(imgAttrs, fmt.Sprintf(`class="%s"`, cssClass))
			}

			if lazyLoad {
				imgAttrs = append(imgAttrs, `loading="lazy"`)
				if imageData.LQIP != "" {
					imgAttrs = append(imgAttrs, `onload="this.parentElement.parentElement.querySelector('.lqip-placeholder')?.remove()"`)
					imgAttrs = append(imgAttrs, `style="position: relative; z-index: 1;"`)
				}
			} else {
				imgAttrs = append(imgAttrs, `loading="eager"`, `fetchpriority="high"`)
			}

			html.WriteString(fmt.Sprintf("<img %s>", strings.Join(imgAttrs, " ")))
			html.WriteString("</picture>")

			// Close wrapper div
			if lazyLoad && imageData.LQIP != "" {
				html.WriteString("</div>")
			}

			return template.HTML(html.String())
		},
		// Simple responsive image without LQIP wrapper (for thumbnails/cards)
		"responsiveImageSimple": func(imageData *ResponsiveImageData, lazyLoad bool, cssClass string) template.HTML {
			if imageData == nil || !imageData.HasVariants {
				return template.HTML("")
			}

			var html strings.Builder
			html.WriteString("<picture>")

			// WebP sources
			webpSrcset := []string{}
			if imageData.ThumbnailWebP != "" {
				webpSrcset = append(webpSrcset, fmt.Sprintf("%s 150w", imageData.ThumbnailWebP))
			}
			if imageData.SmallWebP != "" {
				webpSrcset = append(webpSrcset, fmt.Sprintf("%s 300w", imageData.SmallWebP))
			}
			if imageData.MediumWebP != "" {
				webpSrcset = append(webpSrcset, fmt.Sprintf("%s 600w", imageData.MediumWebP))
			}
			if len(webpSrcset) > 0 {
				html.WriteString(fmt.Sprintf(`<source type="image/webp" srcset="%s" sizes="(max-width: 150px) 150px, (max-width: 300px) 300px, 600px">`, strings.Join(webpSrcset, ", ")))
			}

			// JPEG sources
			jpegSrcset := []string{}
			if imageData.ThumbnailJPEG != "" {
				jpegSrcset = append(jpegSrcset, fmt.Sprintf("%s 150w", imageData.ThumbnailJPEG))
			}
			if imageData.SmallJPEG != "" {
				jpegSrcset = append(jpegSrcset, fmt.Sprintf("%s 300w", imageData.SmallJPEG))
			}
			if imageData.MediumJPEG != "" {
				jpegSrcset = append(jpegSrcset, fmt.Sprintf("%s 600w", imageData.MediumJPEG))
			}
			if len(jpegSrcset) > 0 {
				html.WriteString(fmt.Sprintf(`<source type="image/jpeg" srcset="%s" sizes="(max-width: 150px) 150px, (max-width: 300px) 300px, 600px">`, strings.Join(jpegSrcset, ", ")))
			}

			// Fallback img
			fallbackSrc := imageData.SmallJPEG
			if fallbackSrc == "" {
				fallbackSrc = imageData.MediumJPEG
			}

			loadingAttr := "lazy"
			if !lazyLoad {
				loadingAttr = "eager"
			}

			html.WriteString(fmt.Sprintf(`<img src="%s" alt="%s" class="%s" loading="%s" decoding="async">`,
				fallbackSrc, imageData.AltText, cssClass, loadingAttr))
			html.WriteString("</picture>")

			return template.HTML(html.String())
		},
		// containsSlice checks if a slice contains a specific item
		"containsSlice": func(slice []string, item string) bool {
			for _, s := range slice {
				if s == item {
					return true
				}
			}
			return false
		},
		// iterate creates a slice of integers for looping
		"iterate": func(count int) []int {
			result := make([]int, count)
			for i := 0; i < count; i++ {
				result[i] = i
			}
			return result
		},
		// imageVariant generates a URL for a specific image size/format variant
		// Usage: {{imageVariant .ImageURL "640" "webp"}} or {{imageVariant .ImageURL "1024" ""}}
		"imageVariant": func(src string, width string, format string) string {
			if src == "" {
				return ""
			}
			// If the image is already a variant URL or external, return as-is
			if strings.Contains(src, "?") || strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
				return src
			}
			// Extract file extension
			ext := filepath.Ext(src)
			basePath := strings.TrimSuffix(src, ext)

			// Determine output format
			outputExt := ext
			if format != "" {
				outputExt = "." + format
			}

			// Generate variant URL: /uploads/images/article-w640.webp
			return fmt.Sprintf("%s-w%s%s", basePath, width, outputExt)
		},
		// dict creates a map from key-value pairs for template use
		// Usage: {{template "responsive-image" (dict "src" .ImageURL "alt" .Title "class" "hero-img")}}
		"dict": func(values ...interface{}) map[string]interface{} {
			if len(values)%2 != 0 {
				return nil
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					continue
				}
				dict[key] = values[i+1]
			}
			return dict
		},
		// Translation function for multilingual support
		"t": func(lang string, key string) string {
			return translateText(lang, key)
		},
	}

	// Create templateSet to hold separate templates for each page
	ts := &templateSet{
		templates: make(map[string]*template.Template),
		funcMap:   funcMap,
	}

	// Load layout templates
	layoutsPattern := filepath.Join(templatesDir, "layouts", "*.html")
	layouts, err := filepath.Glob(layoutsPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to load layouts: %w", err)
	}

	// Load component templates
	componentsPattern := filepath.Join(templatesDir, "components", "*.html")
	components, err := filepath.Glob(componentsPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to load components: %w", err)
	}

	// Load page templates
	pagesPattern := filepath.Join(templatesDir, "pages", "*.html")
	pages, err := filepath.Glob(pagesPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to load pages: %w", err)
	}

	// Create a separate template for each page with layouts and components
	for _, page := range pages {
		pageName := strings.TrimSuffix(filepath.Base(page), ".html")

		// Skip admin templates
		if strings.Contains(page, "admin") {
			continue
		}

		// IMPORTANT: Use the filename WITH .html extension as the template name
		// because that's what {{define "article.html"}} uses in the template files
		pageFileName := filepath.Base(page) // e.g., "article.html"

		// Combine all template files: page first, then layouts, then components
		allFiles := append([]string{page}, layouts...)
		allFiles = append(allFiles, components...)

		// Create template with the EXACT name used in {{define "..."}}
		tmpl, err := template.New(pageFileName).Funcs(funcMap).ParseFiles(allFiles...)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template %s: %w", pageName, err)
		}

		// Store with the name without extension for lookup
		ts.templates[pageName] = tmpl
	}

	return ts, nil
}

func getTextDirection(language string) string {
	rtlLanguages := map[string]bool{
		"fa": true, // Persian
		"ar": true, // Arabic
		"he": true, // Hebrew
		"ur": true, // Urdu
	}

	if rtlLanguages[language] {
		return "rtl"
	}
	return "ltr"
}

// translateText returns the translation for a key in the specified language
// Falls back to English if translation not found
func translateText(lang, key string) string {
	translations := map[string]map[string]string{
		"en": {
			"home": "Home", "latest": "Latest", "trending": "Trending", "categories": "Categories",
			"tags": "Tags", "about": "About", "contact": "Contact", "search": "Search",
			"login": "Login", "logout": "Logout", "menu": "Menu", "latest_news": "Latest News",
			"view_all": "View All", "explore_topics": "Explore Topics", "trending_now": "Trending Now",
			"popular_tags": "Popular Tags", "stay_updated": "Stay Updated",
			"newsletter_desc": "Get the latest news delivered to your inbox", "subscribe": "Subscribe",
			"privacy_note": "We respect your privacy. Unsubscribe anytime.", "follow_us": "Follow Us",
			"read_more": "Read More", "share": "Share", "comments": "Comments", "related": "Related Articles",
			"views": "views", "min_read": "min read", "published": "Published", "updated": "Updated", "by": "By",
			"search_placeholder": "Type to search...", "search_results": "Search Results", "no_results": "No results found",
			"all_rights": "All rights reserved", "privacy_policy": "Privacy Policy", "terms": "Terms of Service",
			"loading": "Loading...", "error": "Error", "back": "Back", "next": "Next", "previous": "Previous", "articles": "articles",
		},
		"de": {
			"home": "Startseite", "latest": "Neueste", "trending": "Beliebt", "categories": "Kategorien",
			"tags": "Schlagwörter", "about": "Über uns", "contact": "Kontakt", "search": "Suche",
			"login": "Anmelden", "logout": "Abmelden", "menu": "Menü", "latest_news": "Neueste Nachrichten",
			"view_all": "Alle anzeigen", "explore_topics": "Themen erkunden", "trending_now": "Jetzt im Trend",
			"popular_tags": "Beliebte Tags", "stay_updated": "Bleiben Sie informiert",
			"newsletter_desc": "Erhalten Sie die neuesten Nachrichten in Ihrem Posteingang", "subscribe": "Abonnieren",
			"privacy_note": "Wir respektieren Ihre Privatsphäre. Jederzeit abmelden.", "follow_us": "Folgen Sie uns",
			"read_more": "Weiterlesen", "share": "Teilen", "comments": "Kommentare", "related": "Ähnliche Artikel",
			"views": "Aufrufe", "min_read": "Min. Lesezeit", "published": "Veröffentlicht", "updated": "Aktualisiert", "by": "Von",
			"search_placeholder": "Suchen...", "search_results": "Suchergebnisse", "no_results": "Keine Ergebnisse gefunden",
			"all_rights": "Alle Rechte vorbehalten", "privacy_policy": "Datenschutz", "terms": "Nutzungsbedingungen",
			"loading": "Laden...", "error": "Fehler", "back": "Zurück", "next": "Weiter", "previous": "Zurück", "articles": "Artikel",
		},
		"fr": {
			"home": "Accueil", "latest": "Récent", "trending": "Tendances", "categories": "Catégories",
			"tags": "Étiquettes", "about": "À propos", "contact": "Contact", "search": "Recherche",
			"login": "Connexion", "logout": "Déconnexion", "menu": "Menu", "latest_news": "Dernières actualités",
			"view_all": "Voir tout", "explore_topics": "Explorer les sujets", "trending_now": "Tendances actuelles",
			"popular_tags": "Tags populaires", "stay_updated": "Restez informé",
			"newsletter_desc": "Recevez les dernières nouvelles dans votre boîte mail", "subscribe": "S'abonner",
			"privacy_note": "Nous respectons votre vie privée. Désabonnez-vous à tout moment.", "follow_us": "Suivez-nous",
			"read_more": "Lire la suite", "share": "Partager", "comments": "Commentaires", "related": "Articles similaires",
			"views": "vues", "min_read": "min de lecture", "published": "Publié", "updated": "Mis à jour", "by": "Par",
			"search_placeholder": "Rechercher...", "search_results": "Résultats de recherche", "no_results": "Aucun résultat trouvé",
			"all_rights": "Tous droits réservés", "privacy_policy": "Politique de confidentialité", "terms": "Conditions d'utilisation",
			"loading": "Chargement...", "error": "Erreur", "back": "Retour", "next": "Suivant", "previous": "Précédent", "articles": "articles",
		},
		"es": {
			"home": "Inicio", "latest": "Reciente", "trending": "Tendencias", "categories": "Categorías",
			"tags": "Etiquetas", "about": "Acerca de", "contact": "Contacto", "search": "Buscar",
			"login": "Iniciar sesión", "logout": "Cerrar sesión", "menu": "Menú", "latest_news": "Últimas noticias",
			"view_all": "Ver todo", "explore_topics": "Explorar temas", "trending_now": "Tendencias ahora",
			"popular_tags": "Tags populares", "stay_updated": "Mantente informado",
			"newsletter_desc": "Recibe las últimas noticias en tu correo", "subscribe": "Suscribirse",
			"privacy_note": "Respetamos tu privacidad. Cancela cuando quieras.", "follow_us": "Síguenos",
			"read_more": "Leer más", "share": "Compartir", "comments": "Comentarios", "related": "Artículos relacionados",
			"views": "vistas", "min_read": "min de lectura", "published": "Publicado", "updated": "Actualizado", "by": "Por",
			"search_placeholder": "Buscar...", "search_results": "Resultados de búsqueda", "no_results": "No se encontraron resultados",
			"all_rights": "Todos los derechos reservados", "privacy_policy": "Política de privacidad", "terms": "Términos de servicio",
			"loading": "Cargando...", "error": "Error", "back": "Atrás", "next": "Siguiente", "previous": "Anterior", "articles": "artículos",
		},
		"ar": {
			"home": "الرئيسية", "latest": "الأحدث", "trending": "الأكثر رواجاً", "categories": "التصنيفات",
			"tags": "الوسوم", "about": "من نحن", "contact": "اتصل بنا", "search": "بحث",
			"login": "تسجيل الدخول", "logout": "تسجيل الخروج", "menu": "القائمة", "latest_news": "آخر الأخبار",
			"view_all": "عرض الكل", "explore_topics": "استكشف المواضيع", "trending_now": "الأكثر رواجاً الآن",
			"popular_tags": "الوسوم الشائعة", "stay_updated": "ابق على اطلاع",
			"newsletter_desc": "احصل على آخر الأخبار في بريدك الإلكتروني", "subscribe": "اشترك",
			"privacy_note": "نحترم خصوصيتك. يمكنك إلغاء الاشتراك في أي وقت.", "follow_us": "تابعنا",
			"read_more": "اقرأ المزيد", "share": "مشاركة", "comments": "التعليقات", "related": "مقالات ذات صلة",
			"views": "مشاهدة", "min_read": "دقيقة قراءة", "published": "نُشر في", "updated": "تم التحديث", "by": "بواسطة",
			"search_placeholder": "ابحث هنا...", "search_results": "نتائج البحث", "no_results": "لم يتم العثور على نتائج",
			"all_rights": "جميع الحقوق محفوظة", "privacy_policy": "سياسة الخصوصية", "terms": "شروط الخدمة",
			"loading": "جاري التحميل...", "error": "خطأ", "back": "رجوع", "next": "التالي", "previous": "السابق", "articles": "مقالات",
		},
	}

	if langTranslations, ok := translations[lang]; ok {
		if text, ok := langTranslations[key]; ok {
			return text
		}
	}
	// Fallback to English
	if engTranslations, ok := translations["en"]; ok {
		if text, ok := engTranslations[key]; ok {
			return text
		}
	}
	return key
}

func (sg *StaticGenerator) getRelatedArticles(ctx context.Context, article *models.Article) ([]models.Article, error) {
	// Return empty slice if no repository is available
	if sg.articleRepo == nil {
		return []models.Article{}, nil
	}

	// Get articles from the same category
	articles, err := sg.articleRepo.GetByCategory(ctx, article.CategoryID, 5, 0)
	if err != nil {
		return nil, err
	}

	// Filter out the current article
	var related []models.Article
	for _, a := range articles {
		if a.ID != article.ID {
			related = append(related, a)
		}
	}

	return related, nil
}

// getCategoryByTranslationGroupAndLanguage finds a category by its translation group and language
// This is used to get the correct localized version of a category for SEO purposes
func (sg *StaticGenerator) getCategoryByTranslationGroupAndLanguage(ctx context.Context, translationGroupID uint64, languageCode string) (*models.Category, error) {
	if sg.categoryRepo == nil {
		return nil, fmt.Errorf("category repository not available")
	}

	return sg.categoryRepo.GetByTranslationGroupAndLanguage(ctx, translationGroupID, languageCode)
}

func (sg *StaticGenerator) generateHomepageSchema(articles []models.Article) string {
	schema := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    "WebSite",
		"name":     sg.siteName,
		"url":      sg.baseURL,
		"potentialAction": map[string]interface{}{
			"@type":       "SearchAction",
			"target":      sg.baseURL + "/search?q={search_term_string}",
			"query-input": "required name=search_term_string",
		},
	}

	if len(articles) > 0 {
		var articleSchemas []map[string]interface{}
		for _, article := range articles {
			// Use article's language for URL (SEO best practice)
			articleLang := article.LanguageCode
			if articleLang == "" {
				articleLang = "en"
			}
			articleSchema := map[string]interface{}{
				"@type":         "NewsArticle",
				"headline":      article.Title,
				"url":           fmt.Sprintf("%s/%s/article/%s", sg.baseURL, articleLang, article.Slug),
				"datePublished": article.PublishedAt,
				"dateModified":  article.UpdatedAt,
				"description":   article.Excerpt,
			}
			articleSchemas = append(articleSchemas, articleSchema)
		}
		schema["mainEntity"] = articleSchemas
	}

	schemaJSON, _ := json.Marshal(schema)
	return string(schemaJSON)
}

func (sg *StaticGenerator) generateArticleSchema(article *models.Article, category *models.Category) string {
	schemaType := "NewsArticle"
	if article.SEOData.SchemaType != "" {
		schemaType = article.SEOData.SchemaType
	}

	// Use article's language for URL (SEO best practice)
	articleLang := article.LanguageCode
	if articleLang == "" {
		articleLang = "en"
	}
	articleURL := fmt.Sprintf("%s/%s/article/%s", sg.baseURL, articleLang, article.Slug)

	schema := map[string]interface{}{
		"@context":            "https://schema.org",
		"@type":               schemaType,
		"headline":            article.Title,
		"url":                 articleURL,
		"datePublished":       article.PublishedAt,
		"dateModified":        article.UpdatedAt,
		"description":         article.Excerpt,
		"articleBody":         article.Content,
		"wordCount":           len(strings.Fields(article.Content)),
		"isAccessibleForFree": true,
		"inLanguage":          articleLang,
	}

	// Add mainEntityOfPage (required for Google News)
	schema["mainEntityOfPage"] = map[string]interface{}{
		"@type": "WebPage",
		"@id":   articleURL,
	}

	// Add author information (required for Google News)
	schema["author"] = map[string]interface{}{
		"@type": "Person",
		"name":  "Editorial Team",
		"url":   sg.baseURL + "/about",
	}

	// Collect keywords from SEO data and tags
	var allKeywords []string
	if article.SEOData.Keywords != nil && len(article.SEOData.Keywords) > 0 {
		allKeywords = append(allKeywords, article.SEOData.Keywords...)
	}
	for _, tag := range article.Tags {
		allKeywords = append(allKeywords, tag.Name)
	}
	if len(allKeywords) > 0 {
		// Deduplicate and limit to 10 keywords
		seen := make(map[string]bool)
		unique := []string{}
		for _, kw := range allKeywords {
			if !seen[kw] && kw != "" {
				seen[kw] = true
				unique = append(unique, kw)
				if len(unique) >= 10 {
					break
				}
			}
		}
		schema["keywords"] = strings.Join(unique, ", ")
	}

	// Add articleSection (category) - use the language-resolved category passed as parameter
	if category != nil {
		schema["articleSection"] = category.Name
	} else if len(article.Categories) > 0 {
		// Fallback to article.Categories if no category passed
		schema["articleSection"] = article.Categories[0].Name
	}

	// Add publisher information with logo (required for Google News)
	schema["publisher"] = map[string]interface{}{
		"@type": "Organization",
		"name":  sg.siteName,
		"url":   sg.baseURL,
		"logo": map[string]interface{}{
			"@type":  "ImageObject",
			"url":    sg.baseURL + "/static/images/logo.svg",
			"width":  600,
			"height": 60,
		},
	}

	// Add featured image with full URL (required for Google News)
	if article.FeaturedImage != "" {
		imageURL := article.FeaturedImage
		// Ensure absolute URL
		if !strings.HasPrefix(imageURL, "http") {
			imageURL = sg.baseURL + imageURL
		}
		schema["image"] = map[string]interface{}{
			"@type":  "ImageObject",
			"url":    imageURL,
			"width":  1200,
			"height": 630,
		}
	}

	schemaJSON, _ := json.Marshal(schema)
	return string(schemaJSON)
}

func (sg *StaticGenerator) generateCategorySchema(category *models.Category, articles []models.Article) string {
	// Use category's language for URL (SEO best practice)
	catLang := category.LanguageCode
	if catLang == "" {
		catLang = "en"
	}
	
	schema := map[string]interface{}{
		"@context":    "https://schema.org",
		"@type":       "CollectionPage",
		"name":        category.Name,
		"url":         fmt.Sprintf("%s/%s/category/%s", sg.baseURL, catLang, category.Slug),
		"description": category.Description,
	}

	if len(articles) > 0 {
		var articleSchemas []map[string]interface{}
		for _, article := range articles {
			// Use article's language for URL (SEO best practice)
			articleLang := article.LanguageCode
			if articleLang == "" {
				articleLang = "en"
			}
			articleSchema := map[string]interface{}{
				"@type":         "NewsArticle",
				"headline":      article.Title,
				"url":           fmt.Sprintf("%s/%s/article/%s", sg.baseURL, articleLang, article.Slug),
				"datePublished": article.PublishedAt,
				"dateModified":  article.UpdatedAt,
				"description":   article.Excerpt,
			}
			articleSchemas = append(articleSchemas, articleSchema)
		}
		schema["mainEntity"] = articleSchemas
	}

	schemaJSON, _ := json.Marshal(schema)
	return string(schemaJSON)
}

func (sg *StaticGenerator) generateTagSchema(tag *models.Tag, articles []models.Article) string {
	// Use tag's language for URL (SEO best practice)
	tagLang := tag.LanguageCode
	if tagLang == "" {
		tagLang = "en"
	}
	
	schema := map[string]interface{}{
		"@context":    "https://schema.org",
		"@type":       "CollectionPage",
		"name":        tag.Name,
		"url":         fmt.Sprintf("%s/%s/tag/%s", sg.baseURL, tagLang, tag.Slug),
		"description": tag.Description,
	}

	if len(tag.Keywords) > 0 {
		schema["keywords"] = tag.Keywords
	}

	if len(articles) > 0 {
		var articleSchemas []map[string]interface{}
		for _, article := range articles {
			// Use article's language for URL (SEO best practice)
			articleLang := article.LanguageCode
			if articleLang == "" {
				articleLang = "en"
			}
			articleSchema := map[string]interface{}{
				"@type":         "NewsArticle",
				"headline":      article.Title,
				"url":           fmt.Sprintf("%s/%s/article/%s", sg.baseURL, articleLang, article.Slug),
				"datePublished": article.PublishedAt,
				"dateModified":  article.UpdatedAt,
				"description":   article.Excerpt,
			}
			articleSchemas = append(articleSchemas, articleSchema)
		}
		schema["mainEntity"] = articleSchemas
	}

	schemaJSON, _ := json.Marshal(schema)
	return string(schemaJSON)
}

func (sg *StaticGenerator) warmHomepageCache(language string) {
	if sg.cacheService == nil {
		return
	}

	ctx := context.Background()
	cacheKey := fmt.Sprintf("homepage:%s", language)

	// Pre-load homepage data into cache
	if sg.articleRepo != nil {
		// Cache latest articles
		if articles, err := sg.articleRepo.GetLatestArticles(ctx, 20); err == nil {
			if data, err := json.Marshal(articles); err == nil {
				sg.cacheService.Set(ctx, fmt.Sprintf("latest_articles:%s", language), data, 15*time.Minute)
			}
		}

		// Cache trending articles
		if trending, err := sg.articleRepo.GetTrendingArticles(ctx, 10, 24); err == nil {
			if data, err := json.Marshal(trending); err == nil {
				sg.cacheService.Set(ctx, fmt.Sprintf("trending_articles:%s", language), data, 15*time.Minute)
			}
		}
	}

	log.Printf("Warmed cache for homepage: %s", cacheKey)
}

func (sg *StaticGenerator) warmArticleCache(article *models.Article) {
	if sg.cacheService == nil {
		return
	}

	ctx := context.Background()
	cacheKey := fmt.Sprintf("article:%s", article.Slug)

	// Cache article data
	if data, err := json.Marshal(article); err == nil {
		sg.cacheService.Set(ctx, cacheKey, data, 24*time.Hour)
	}

	// Cache related articles
	if related, err := sg.getRelatedArticles(ctx, article); err == nil {
		if data, err := json.Marshal(related); err == nil {
			sg.cacheService.Set(ctx, fmt.Sprintf("related_articles:%d", article.ID), data, 1*time.Hour)
		}
	}

	log.Printf("Warmed cache for article: %s", cacheKey)
}

func (sg *StaticGenerator) warmCategoryCache(category *models.Category, page int) {
	if sg.cacheService == nil {
		return
	}

	ctx := context.Background()
	cacheKey := fmt.Sprintf("category:%s:page:%d", category.Slug, page)

	// Cache category data
	if data, err := json.Marshal(category); err == nil {
		sg.cacheService.Set(ctx, fmt.Sprintf("category:%d", category.ID), data, 30*time.Minute)
	}

	log.Printf("Warmed cache for category: %s (page %d)", cacheKey, page)
}

func (sg *StaticGenerator) warmTagCache(tag *models.Tag, page int) {
	if sg.cacheService == nil {
		return
	}

	ctx := context.Background()
	cacheKey := fmt.Sprintf("tag:%s:page:%d", tag.Slug, page)

	// Cache tag data
	if data, err := json.Marshal(tag); err == nil {
		sg.cacheService.Set(ctx, fmt.Sprintf("tag:%d", tag.ID), data, 30*time.Minute)
	}

	log.Printf("Warmed cache for tag: %s (page %d)", cacheKey, page)
}

func (sg *StaticGenerator) invalidateRelatedCaches(article *models.Article) {
	if sg.cacheService == nil {
		return
	}

	ctx := context.Background()

	// Invalidate article-specific caches
	cacheKeys := []string{
		fmt.Sprintf("article:%s", article.Slug),
		fmt.Sprintf("article:%d", article.ID),
		fmt.Sprintf("related_articles:%d", article.ID),
	}

	// Invalidate homepage caches for all languages
	languages := []string{"fa", "en", "ar"}
	for _, lang := range languages {
		cacheKeys = append(cacheKeys,
			fmt.Sprintf("homepage:%s", lang),
			fmt.Sprintf("latest_articles:%s", lang),
			fmt.Sprintf("trending_articles:%s", lang),
		)
	}

	// Invalidate category caches
	cacheKeys = append(cacheKeys,
		fmt.Sprintf("category:%d", article.CategoryID),
		fmt.Sprintf("category_articles:%d", article.CategoryID),
	)

	// Invalidate tag caches
	for _, tag := range article.Tags {
		cacheKeys = append(cacheKeys,
			fmt.Sprintf("tag:%d", tag.ID),
			fmt.Sprintf("tag_articles:%d", tag.ID),
		)
	}

	// Perform cache invalidation
	for _, key := range cacheKeys {
		if err := sg.cacheService.Delete(ctx, key); err != nil {
			log.Printf("Warning: failed to invalidate cache key %s: %v", key, err)
		}
	}

	// Invalidate pattern-based caches
	patterns := []string{
		"homepage:*",
		fmt.Sprintf("category:%d:*", article.CategoryID),
	}

	for _, tag := range article.Tags {
		patterns = append(patterns, fmt.Sprintf("tag:%d:*", tag.ID))
	}

	for _, pattern := range patterns {
		if err := sg.cacheService.DeletePattern(ctx, pattern); err != nil {
			log.Printf("Warning: failed to invalidate cache pattern %s: %v", pattern, err)
		}
	}

	log.Printf("Invalidated caches for article: %s", article.Slug)
}
