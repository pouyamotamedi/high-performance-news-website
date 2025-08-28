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
	templates    *template.Template
	outputPath   string
	cacheService cache.CacheService
	articleRepo  *repositories.ArticleRepository
	categoryRepo *repositories.CategoryRepository
	tagRepo      *repositories.TagRepository
	baseURL      string
}

// StaticGeneratorConfig holds configuration for static generation
type StaticGeneratorConfig struct {
	OutputPath   string
	TemplatesDir string
	BaseURL      string
}

// PageData represents common data for all pages
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
	tagRepo *repositories.TagRepository) (*StaticGenerator, error) {

	// Load templates
	templates, err := loadTemplates(config.TemplatesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(config.OutputPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	return &StaticGenerator{
		templates:    templates,
		outputPath:   config.OutputPath,
		cacheService: cacheService,
		articleRepo:  articleRepo,
		categoryRepo: categoryRepo,
		tagRepo:      tagRepo,
		baseURL:      config.BaseURL,
	}, nil
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

	// Get categories (simplified for now)
	var categories []models.Category

	// Get featured tags (simplified for now)
	var featuredTags []models.Tag

	// Prepare page data
	pageData := PageData{
		Title:        "High Performance News - Latest News and Updates",
		Description:  "Stay updated with the latest news and trending stories from around the world.",
		Keywords:     []string{"news", "latest", "trending", "updates"},
		CanonicalURL: sg.baseURL + "/",
		Language:     language,
		Direction:    getTextDirection(language),
		BaseURL:      sg.baseURL,
		SchemaMarkup: sg.generateHomepageSchema(latestArticles),
	}

	data := HomepageData{
		PageData:         pageData,
		LatestArticles:   latestArticles,
		TrendingArticles: trendingArticles,
		Categories:       categories,
		FeaturedTags:     featuredTags,
	}

	// Generate HTML
	var buf bytes.Buffer
	if err := sg.templates.ExecuteTemplate(&buf, "homepage.html", data); err != nil {
		return fmt.Errorf("failed to execute homepage template: %w", err)
	}

	// Write to file
	outputPath := filepath.Join(sg.outputPath, "index.html")
	if language != "fa" { // Default language
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

	log.Printf("Generated homepage for language: %s", language)
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

	// Get category (simplified for now)
	var category models.Category

	// Get tags
	tags := article.Tags

	// Get author (simplified for now)
	var author models.User

	// Prepare page data
	pageData := PageData{
		Title:        article.SEOData.MetaTitle,
		Description:  article.SEOData.MetaDescription,
		Keywords:     article.SEOData.Keywords,
		CanonicalURL: sg.baseURL + "/articles/" + article.Slug,
		Language:     article.LanguageCode,
		Direction:    getTextDirection(article.LanguageCode),
		BaseURL:      sg.baseURL,
		SchemaMarkup: sg.generateArticleSchema(article),
	}

	if pageData.Title == "" {
		pageData.Title = article.Title
	}
	if pageData.Description == "" {
		pageData.Description = article.Excerpt
	}

	data := ArticlePageData{
		PageData:        pageData,
		Article:         *article,
		RelatedArticles: relatedArticles,
		Category:        category,
		Tags:            tags,
		Author:          author,
	}

	// Generate HTML
	var buf bytes.Buffer
	if err := sg.templates.ExecuteTemplate(&buf, "article.html", data); err != nil {
		return fmt.Errorf("failed to execute article template: %w", err)
	}

	// Create directory structure
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

	log.Printf("Generated article page: %s", article.Slug)
	return nil
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

	// Prepare page data
	pageData := PageData{
		Title:        fmt.Sprintf("%s - Category", category.Name),
		Description:  category.Description,
		Keywords:     []string{category.Name, "category", "articles"},
		CanonicalURL: sg.baseURL + "/categories/" + category.Slug,
		Language:     "fa", // Default language, could be parameterized
		Direction:    "rtl",
		BaseURL:      sg.baseURL,
		SchemaMarkup: sg.generateCategorySchema(category, articles),
	}

	if page > 1 {
		pageData.Title = fmt.Sprintf("%s - Page %d - Category", category.Name, page)
		pageData.CanonicalURL = fmt.Sprintf("%s/categories/%s/page-%d", sg.baseURL, category.Slug, page)
	}

	data := CategoryPageData{
		PageData:   pageData,
		Category:   *category,
		Articles:   articles,
		Pagination: pagination,
	}

	// Generate HTML
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

	log.Printf("Generated category page: %s (page %d)", category.Slug, page)
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

	// Prepare page data
	pageData := PageData{
		Title:        fmt.Sprintf("%s - Tag", tag.Name),
		Description:  tag.Description,
		Keywords:     append([]string{tag.Name, "tag", "articles"}, tag.Keywords...),
		CanonicalURL: sg.baseURL + "/tags/" + tag.Slug,
		Language:     "fa", // Default language, could be parameterized
		Direction:    "rtl",
		BaseURL:      sg.baseURL,
		SchemaMarkup: sg.generateTagSchema(tag, articles),
	}

	if page > 1 {
		pageData.Title = fmt.Sprintf("%s - Page %d - Tag", tag.Name, page)
		pageData.CanonicalURL = fmt.Sprintf("%s/tags/%s/page-%d", sg.baseURL, tag.Slug, page)
	}

	data := TagPageData{
		PageData:   pageData,
		Tag:        *tag,
		Articles:   articles,
		Pagination: pagination,
	}

	// Generate HTML
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

	log.Printf("Generated tag page: %s (page %d)", tag.Slug, page)
	return nil
}

// RegenerateOnContentUpdate regenerates static files when content is updated
func (sg *StaticGenerator) RegenerateOnContentUpdate(ctx context.Context, article *models.Article) error {
	// Regenerate article page
	if err := sg.GenerateArticlePage(ctx, article); err != nil {
		return fmt.Errorf("failed to regenerate article page: %w", err)
	}

	// Regenerate homepage for all languages
	languages := []string{"fa", "en", "ar"} // Supported languages
	for _, lang := range languages {
		if err := sg.GenerateHomepage(ctx, lang); err != nil {
			log.Printf("Warning: failed to regenerate homepage for language %s: %v", lang, err)
		}
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

	return nil
}

// Helper functions

func loadTemplates(templatesDir string) (*template.Template, error) {
	// Define template functions
	funcMap := template.FuncMap{
		"formatDate": func(t time.Time) string {
			return t.Format("January 2, 2006")
		},
		"formatDateTime": func(t time.Time) string {
			return t.Format("January 2, 2006 at 3:04 PM")
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
		"join": func(slice []string, sep string) string {
			return strings.Join(slice, sep)
		},
	}

	// Create base template
	tmpl := template.New("").Funcs(funcMap)

	// Walk through templates directory and load all .html files
	err := filepath.Walk(templatesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() && strings.HasSuffix(path, ".html") {
			// Get relative path for template name
			relPath, err := filepath.Rel(templatesDir, path)
			if err != nil {
				return err
			}
			
			// Read template content
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			
			// Parse template with proper name
			templateName := filepath.Base(relPath)
			_, err = tmpl.New(templateName).Parse(string(content))
			if err != nil {
				return fmt.Errorf("failed to parse template %s: %w", templateName, err)
			}
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}

	return tmpl, nil
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

func (sg *StaticGenerator) generateHomepageSchema(articles []models.Article) string {
	schema := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    "WebSite",
		"name":     "High Performance News",
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
			articleSchema := map[string]interface{}{
				"@type":         "NewsArticle",
				"headline":      article.Title,
				"url":           sg.baseURL + "/articles/" + article.Slug,
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

func (sg *StaticGenerator) generateArticleSchema(article *models.Article) string {
	schema := map[string]interface{}{
		"@context":      "https://schema.org",
		"@type":         article.SEOData.SchemaType,
		"headline":      article.Title,
		"url":           sg.baseURL + "/articles/" + article.Slug,
		"datePublished": article.PublishedAt,
		"dateModified":  article.UpdatedAt,
		"description":   article.Excerpt,
		"articleBody":   article.Content,
		"wordCount":     len(strings.Fields(article.Content)),
	}

	if article.SEOData.Keywords != nil && len(article.SEOData.Keywords) > 0 {
		schema["keywords"] = article.SEOData.Keywords
	}

	schemaJSON, _ := json.Marshal(schema)
	return string(schemaJSON)
}

func (sg *StaticGenerator) generateCategorySchema(category *models.Category, articles []models.Article) string {
	schema := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    "CollectionPage",
		"name":     category.Name,
		"url":      sg.baseURL + "/categories/" + category.Slug,
		"description": category.Description,
	}

	if len(articles) > 0 {
		var articleSchemas []map[string]interface{}
		for _, article := range articles {
			articleSchema := map[string]interface{}{
				"@type":         "NewsArticle",
				"headline":      article.Title,
				"url":           sg.baseURL + "/articles/" + article.Slug,
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
	schema := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    "CollectionPage",
		"name":     tag.Name,
		"url":      sg.baseURL + "/tags/" + tag.Slug,
		"description": tag.Description,
	}

	if len(tag.Keywords) > 0 {
		schema["keywords"] = tag.Keywords
	}

	if len(articles) > 0 {
		var articleSchemas []map[string]interface{}
		for _, article := range articles {
			articleSchema := map[string]interface{}{
				"@type":         "NewsArticle",
				"headline":      article.Title,
				"url":           sg.baseURL + "/articles/" + article.Slug,
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