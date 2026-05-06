package services

import (
	"fmt"
	"strings"
	"time"

	"high-performance-news-website/internal/models"
)

// LLMsTxtService generates llms.txt files for AI-ready structured data
type LLMsTxtService struct {
	articleRepo  ArticleRepositoryInterface
	categoryRepo CategoryRepositoryInterface
	tagRepo      TagRepositoryInterface
	userRepo     UserRepositoryInterface
	siteConfig   *SiteConfig
}

// SiteConfig holds site configuration for llms.txt generation
type SiteConfig struct {
	SiteName        string
	SiteURL         string
	Description     string
	Language        string
	ContactEmail    string
	LastUpdated     time.Time
}

// NewLLMsTxtService creates a new llms.txt service
func NewLLMsTxtService(
	articleRepo ArticleRepositoryInterface,
	categoryRepo CategoryRepositoryInterface,
	tagRepo TagRepositoryInterface,
	userRepo UserRepositoryInterface,
	siteConfig *SiteConfig,
) *LLMsTxtService {
	return &LLMsTxtService{
		articleRepo:  articleRepo,
		categoryRepo: categoryRepo,
		tagRepo:      tagRepo,
		userRepo:     userRepo,
		siteConfig:   siteConfig,
	}
}

// LLMsTxtContent represents the structured content for llms.txt
type LLMsTxtContent struct {
	SiteInfo     SiteInfo              `json:"site_info"`
	Content      ContentSummary        `json:"content"`
	Categories   []CategoryInfo        `json:"categories"`
	Tags         []TagInfo             `json:"tags"`
	Authors      []AuthorInfo          `json:"authors"`
	RecentNews   []ArticleSummary      `json:"recent_news"`
	PopularNews  []ArticleSummary      `json:"popular_news"`
	APIEndpoints []APIEndpoint         `json:"api_endpoints"`
	GeneratedAt  time.Time             `json:"generated_at"`
}

// SiteInfo contains basic site information
type SiteInfo struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Language    string `json:"language"`
	Contact     string `json:"contact"`
	Type        string `json:"type"`
	LastUpdated string `json:"last_updated"`
}

// ContentSummary provides an overview of site content
type ContentSummary struct {
	TotalArticles    int      `json:"total_articles"`
	TotalCategories  int      `json:"total_categories"`
	TotalTags        int      `json:"total_tags"`
	TotalAuthors     int      `json:"total_authors"`
	PublishingRate   string   `json:"publishing_rate"`
	MainTopics       []string `json:"main_topics"`
	ContentLanguages []string `json:"content_languages"`
}

// CategoryInfo represents category information for AI
type CategoryInfo struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	ArticleCount int   `json:"article_count"`
	URL         string `json:"url"`
}

// TagInfo represents tag information for AI
type TagInfo struct {
	ID          uint64   `json:"id"`
	Name        string   `json:"name"`
	Slug        string   `json:"slug"`
	Description string   `json:"description"`
	Keywords    []string `json:"keywords"`
	ArticleCount int     `json:"article_count"`
	URL         string   `json:"url"`
}

// AuthorInfo represents author information for AI
type AuthorInfo struct {
	ID           uint64 `json:"id"`
	Name         string `json:"name"`
	Bio          string `json:"bio"`
	ArticleCount int    `json:"article_count"`
	URL          string `json:"url"`
}

// ArticleSummary represents article summary for AI
type ArticleSummary struct {
	ID          uint64    `json:"id"`
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	Excerpt     string    `json:"excerpt"`
	Author      string    `json:"author"`
	Category    string    `json:"category"`
	Tags        []string  `json:"tags"`
	PublishedAt time.Time `json:"published_at"`
	URL         string    `json:"url"`
	ViewCount   uint64    `json:"view_count"`
}

// APIEndpoint represents available API endpoints
type APIEndpoint struct {
	Path        string `json:"path"`
	Method      string `json:"method"`
	Description string `json:"description"`
	Parameters  string `json:"parameters,omitempty"`
}

// GenerateLLMsTxt generates the complete llms.txt content
func (l *LLMsTxtService) GenerateLLMsTxt() (string, error) {
	content, err := l.GenerateLLMsTxtContent()
	if err != nil {
		return "", fmt.Errorf("failed to generate llms.txt content: %w", err)
	}
	
	return l.formatLLMsTxt(content), nil
}

// GenerateLLMsTxtContent generates structured content for llms.txt
func (l *LLMsTxtService) GenerateLLMsTxtContent() (*LLMsTxtContent, error) {
	content := &LLMsTxtContent{
		GeneratedAt: time.Now(),
	}
	
	// Site information
	content.SiteInfo = SiteInfo{
		Name:        l.siteConfig.SiteName,
		URL:         l.siteConfig.SiteURL,
		Description: l.siteConfig.Description,
		Language:    l.siteConfig.Language,
		Contact:     l.siteConfig.ContactEmail,
		Type:        "news_website",
		LastUpdated: l.siteConfig.LastUpdated.Format("2006-01-02"),
	}
	
	// Content summary
	contentSummary, err := l.generateContentSummary()
	if err != nil {
		return nil, fmt.Errorf("failed to generate content summary: %w", err)
	}
	content.Content = *contentSummary
	
	// Categories
	categories, err := l.generateCategoryInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to generate category info: %w", err)
	}
	content.Categories = categories
	
	// Tags
	tags, err := l.generateTagInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to generate tag info: %w", err)
	}
	content.Tags = tags
	
	// Authors
	authors, err := l.generateAuthorInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to generate author info: %w", err)
	}
	content.Authors = authors
	
	// Recent news
	recentNews, err := l.generateRecentNews(20)
	if err != nil {
		return nil, fmt.Errorf("failed to generate recent news: %w", err)
	}
	content.RecentNews = recentNews
	
	// Popular news
	popularNews, err := l.generatePopularNews(20)
	if err != nil {
		return nil, fmt.Errorf("failed to generate popular news: %w", err)
	}
	content.PopularNews = popularNews
	
	// API endpoints
	content.APIEndpoints = l.generateAPIEndpoints()
	
	return content, nil
}

// generateContentSummary creates a summary of site content
func (l *LLMsTxtService) generateContentSummary() (*ContentSummary, error) {
	// Get total counts
	totalArticles, err := l.articleRepo.GetTotalCount()
	if err != nil {
		return nil, err
	}
	
	totalCategories, err := l.categoryRepo.GetTotalCount()
	if err != nil {
		return nil, err
	}
	
	totalTags, err := l.tagRepo.GetTotalCount()
	if err != nil {
		return nil, err
	}
	
	totalAuthors, err := l.userRepo.GetTotalCount()
	if err != nil {
		return nil, err
	}
	
	// Calculate publishing rate (articles per day over last 30 days)
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	recentArticles, err := l.articleRepo.GetCountSince(thirtyDaysAgo)
	if err != nil {
		return nil, err
	}
	
	publishingRate := fmt.Sprintf("%.1f articles/day", float64(recentArticles)/30.0)
	
	// Get main topics (top categories)
	topCategories, err := l.categoryRepo.GetTopCategories(10)
	if err != nil {
		return nil, err
	}
	
	mainTopics := make([]string, len(topCategories))
	for i, cat := range topCategories {
		mainTopics[i] = cat.Name
	}
	
	return &ContentSummary{
		TotalArticles:    totalArticles,
		TotalCategories:  totalCategories,
		TotalTags:        totalTags,
		TotalAuthors:     totalAuthors,
		PublishingRate:   publishingRate,
		MainTopics:       mainTopics,
		ContentLanguages: []string{l.siteConfig.Language},
	}, nil
}

// generateCategoryInfo creates category information for AI
func (l *LLMsTxtService) generateCategoryInfo() ([]CategoryInfo, error) {
	categories, err := l.categoryRepo.GetAll()
	if err != nil {
		return nil, err
	}
	
	categoryInfos := make([]CategoryInfo, len(categories))
	for i, cat := range categories {
		articleCount, err := l.articleRepo.GetCountByCategory(cat.ID)
		if err != nil {
			articleCount = 0 // Continue with 0 if error
		}
		
		// Use category's language for URL (SEO best practice)
		catLang := cat.LanguageCode
		if catLang == "" {
			catLang = "en"
		}
		
		categoryInfos[i] = CategoryInfo{
			ID:           cat.ID,
			Name:         cat.Name,
			Slug:         cat.Slug,
			Description:  cat.Description,
			ArticleCount: articleCount,
			URL:          fmt.Sprintf("%s/%s/category/%s", l.siteConfig.SiteURL, catLang, cat.Slug),
		}
	}
	
	return categoryInfos, nil
}

// generateTagInfo creates tag information for AI
func (l *LLMsTxtService) generateTagInfo() ([]TagInfo, error) {
	tags, err := l.tagRepo.GetAll()
	if err != nil {
		return nil, err
	}
	
	tagInfos := make([]TagInfo, len(tags))
	for i, tag := range tags {
		articleCount, err := l.articleRepo.GetCountByTag(tag.ID)
		if err != nil {
			articleCount = 0 // Continue with 0 if error
		}
		
		tagInfos[i] = TagInfo{
			ID:           tag.ID,
			Name:         tag.Name,
			Slug:         tag.Slug,
			Description:  tag.Description,
			Keywords:     tag.Keywords,
			ArticleCount: articleCount,
			URL:          fmt.Sprintf("%s/en/tag/%s", l.siteConfig.SiteURL, tag.Slug),
		}
	}
	
	return tagInfos, nil
}

// generateAuthorInfo creates author information for AI
func (l *LLMsTxtService) generateAuthorInfo() ([]AuthorInfo, error) {
	users, err := l.userRepo.GetAll()
	if err != nil {
		return nil, err
	}
	
	authorInfos := make([]AuthorInfo, len(users))
	for i, user := range users {
		articleCount, err := l.articleRepo.GetCountByAuthor(user.ID)
		if err != nil {
			articleCount = 0 // Continue with 0 if error
		}
		
		authorInfos[i] = AuthorInfo{
			ID:           user.ID,
			Name:         fmt.Sprintf("%s %s", user.FirstName, user.LastName),
			Bio:          user.Bio,
			ArticleCount: articleCount,
			URL:          fmt.Sprintf("%s/author/%s", l.siteConfig.SiteURL, user.Username),
		}
	}
	
	return authorInfos, nil
}

// generateRecentNews creates recent news summaries
func (l *LLMsTxtService) generateRecentNews(limit int) ([]ArticleSummary, error) {
	articles, err := l.articleRepo.GetRecent(limit)
	if err != nil {
		return nil, err
	}
	
	return l.convertToArticleSummaries(articles)
}

// generatePopularNews creates popular news summaries
func (l *LLMsTxtService) generatePopularNews(limit int) ([]ArticleSummary, error) {
	articles, err := l.articleRepo.GetPopular(limit)
	if err != nil {
		return nil, err
	}
	
	return l.convertToArticleSummaries(articles)
}

// convertToArticleSummaries converts articles to summaries
func (l *LLMsTxtService) convertToArticleSummaries(articles []*models.Article) ([]ArticleSummary, error) {
	summaries := make([]ArticleSummary, len(articles))
	
	for i, article := range articles {
		// Get author name
		author, err := l.userRepo.GetByID(article.AuthorID)
		authorName := "Unknown"
		if err == nil {
			authorName = fmt.Sprintf("%s %s", author.FirstName, author.LastName)
		}
		
		// Get category name
		category, err := l.categoryRepo.GetByID(article.CategoryID)
		categoryName := "Uncategorized"
		if err == nil {
			categoryName = category.Name
		}
		
		// Get tag names
		tags, err := l.tagRepo.GetByArticleID(article.ID)
		tagNames := make([]string, len(tags))
		if err == nil {
			for j, tag := range tags {
				tagNames[j] = tag.Name
			}
		}
		
		summaries[i] = ArticleSummary{
			ID:          article.ID,
			Title:       article.Title,
			Slug:        article.Slug,
			Excerpt:     article.Excerpt,
			Author:      authorName,
			Category:    categoryName,
			Tags:        tagNames,
			PublishedAt: func() time.Time {
				if article.PublishedAt != nil {
					return *article.PublishedAt
				}
				return time.Time{}
			}(),
			URL: func() string {
				// Use article's language for URL (SEO best practice)
				lang := article.LanguageCode
				if lang == "" {
					lang = "en"
				}
				return fmt.Sprintf("%s/%s/article/%s", l.siteConfig.SiteURL, lang, article.Slug)
			}(),
			ViewCount:   article.ViewCount,
		}
	}
	
	return summaries, nil
}

// generateAPIEndpoints creates API endpoint information
func (l *LLMsTxtService) generateAPIEndpoints() []APIEndpoint {
	return []APIEndpoint{
		{
			Path:        "/api/v1/articles",
			Method:      "GET",
			Description: "Get list of articles with pagination and filtering",
			Parameters:  "page, limit, category, tag, author, search",
		},
		{
			Path:        "/api/v1/articles/{id}",
			Method:      "GET",
			Description: "Get specific article by ID",
		},
		{
			Path:        "/api/v1/search",
			Method:      "GET",
			Description: "Search articles by query",
			Parameters:  "q, category, tag, limit",
		},
		{
			Path:        "/api/v1/categories",
			Method:      "GET",
			Description: "Get list of all categories",
		},
		{
			Path:        "/api/v1/tags",
			Method:      "GET",
			Description: "Get list of all tags",
		},
		{
			Path:        "/api/v1/authors",
			Method:      "GET",
			Description: "Get list of all authors",
		},
		{
			Path:        "/rss",
			Method:      "GET",
			Description: "RSS feed for all articles",
		},
		{
			Path:        "/rss/category/{slug}",
			Method:      "GET",
			Description: "RSS feed for specific category",
		},
		{
			Path:        "/rss/tag/{slug}",
			Method:      "GET",
			Description: "RSS feed for specific tag",
		},
		{
			Path:        "/sitemap.xml",
			Method:      "GET",
			Description: "XML sitemap for search engines",
		},
	}
}

// formatLLMsTxt formats the content as llms.txt
func (l *LLMsTxtService) formatLLMsTxt(content *LLMsTxtContent) string {
	var builder strings.Builder
	
	// Header
	builder.WriteString("# llms.txt - AI-Ready News Website Data\n\n")
	builder.WriteString(fmt.Sprintf("Generated: %s\n\n", content.GeneratedAt.Format("2006-01-02 15:04:05 UTC")))
	
	// Site Information
	builder.WriteString("## Site Information\n\n")
	builder.WriteString(fmt.Sprintf("Name: %s\n", content.SiteInfo.Name))
	builder.WriteString(fmt.Sprintf("URL: %s\n", content.SiteInfo.URL))
	builder.WriteString(fmt.Sprintf("Type: %s\n", content.SiteInfo.Type))
	builder.WriteString(fmt.Sprintf("Language: %s\n", content.SiteInfo.Language))
	builder.WriteString(fmt.Sprintf("Description: %s\n", content.SiteInfo.Description))
	builder.WriteString(fmt.Sprintf("Contact: %s\n", content.SiteInfo.Contact))
	builder.WriteString(fmt.Sprintf("Last Updated: %s\n\n", content.SiteInfo.LastUpdated))
	
	// Content Summary
	builder.WriteString("## Content Overview\n\n")
	builder.WriteString(fmt.Sprintf("Total Articles: %d\n", content.Content.TotalArticles))
	builder.WriteString(fmt.Sprintf("Total Categories: %d\n", content.Content.TotalCategories))
	builder.WriteString(fmt.Sprintf("Total Tags: %d\n", content.Content.TotalTags))
	builder.WriteString(fmt.Sprintf("Total Authors: %d\n", content.Content.TotalAuthors))
	builder.WriteString(fmt.Sprintf("Publishing Rate: %s\n", content.Content.PublishingRate))
	builder.WriteString(fmt.Sprintf("Main Topics: %s\n", strings.Join(content.Content.MainTopics, ", ")))
	builder.WriteString(fmt.Sprintf("Languages: %s\n\n", strings.Join(content.Content.ContentLanguages, ", ")))
	
	// Categories
	builder.WriteString("## Categories\n\n")
	for _, cat := range content.Categories {
		builder.WriteString(fmt.Sprintf("- %s (%d articles): %s\n", cat.Name, cat.ArticleCount, cat.Description))
		builder.WriteString(fmt.Sprintf("  URL: %s\n", cat.URL))
	}
	builder.WriteString("\n")
	
	// Tags
	builder.WriteString("## Tags\n\n")
	for _, tag := range content.Tags {
		builder.WriteString(fmt.Sprintf("- %s (%d articles): %s\n", tag.Name, tag.ArticleCount, tag.Description))
		if len(tag.Keywords) > 0 {
			builder.WriteString(fmt.Sprintf("  Keywords: %s\n", strings.Join(tag.Keywords, ", ")))
		}
		builder.WriteString(fmt.Sprintf("  URL: %s\n", tag.URL))
	}
	builder.WriteString("\n")
	
	// Authors
	builder.WriteString("## Authors\n\n")
	for _, author := range content.Authors {
		builder.WriteString(fmt.Sprintf("- %s (%d articles): %s\n", author.Name, author.ArticleCount, author.Bio))
		builder.WriteString(fmt.Sprintf("  URL: %s\n", author.URL))
	}
	builder.WriteString("\n")
	
	// Recent News
	builder.WriteString("## Recent News\n\n")
	for _, article := range content.RecentNews {
		builder.WriteString(fmt.Sprintf("- %s\n", article.Title))
		builder.WriteString(fmt.Sprintf("  Author: %s | Category: %s | Published: %s\n", 
			article.Author, article.Category, article.PublishedAt.Format("2006-01-02")))
		if len(article.Tags) > 0 {
			builder.WriteString(fmt.Sprintf("  Tags: %s\n", strings.Join(article.Tags, ", ")))
		}
		builder.WriteString(fmt.Sprintf("  URL: %s\n", article.URL))
		if article.Excerpt != "" {
			builder.WriteString(fmt.Sprintf("  Summary: %s\n", article.Excerpt))
		}
		builder.WriteString("\n")
	}
	
	// Popular News
	builder.WriteString("## Popular News\n\n")
	for _, article := range content.PopularNews {
		builder.WriteString(fmt.Sprintf("- %s (Views: %d)\n", article.Title, article.ViewCount))
		builder.WriteString(fmt.Sprintf("  Author: %s | Category: %s | Published: %s\n", 
			article.Author, article.Category, article.PublishedAt.Format("2006-01-02")))
		if len(article.Tags) > 0 {
			builder.WriteString(fmt.Sprintf("  Tags: %s\n", strings.Join(article.Tags, ", ")))
		}
		builder.WriteString(fmt.Sprintf("  URL: %s\n", article.URL))
		if article.Excerpt != "" {
			builder.WriteString(fmt.Sprintf("  Summary: %s\n", article.Excerpt))
		}
		builder.WriteString("\n")
	}
	
	// API Endpoints
	builder.WriteString("## API Endpoints\n\n")
	for _, endpoint := range content.APIEndpoints {
		builder.WriteString(fmt.Sprintf("- %s %s: %s\n", endpoint.Method, endpoint.Path, endpoint.Description))
		if endpoint.Parameters != "" {
			builder.WriteString(fmt.Sprintf("  Parameters: %s\n", endpoint.Parameters))
		}
	}
	builder.WriteString("\n")
	
	// Footer
	builder.WriteString("## Usage Guidelines\n\n")
	builder.WriteString("This data is provided for AI systems to understand the structure and content of this news website.\n")
	builder.WriteString("Please respect our terms of service and rate limits when accessing our APIs.\n")
	builder.WriteString("For bulk data access or special requirements, please contact us.\n\n")
	
	builder.WriteString("## Data Freshness\n\n")
	builder.WriteString("This llms.txt file is regenerated daily to ensure data freshness.\n")
	builder.WriteString("For real-time data, please use our API endpoints listed above.\n")
	
	return builder.String()
}