package services

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/repositories"
	"high-performance-news-website/pkg/cache"
)

// GoogleNewsSitemapService handles Google News sitemap generation
type GoogleNewsSitemapService struct {
	articleRepo *repositories.ArticleRepository
	cache       cache.CacheService
	baseURL     string
	siteName    string
}

// NewGoogleNewsSitemapService creates a new Google News sitemap service
func NewGoogleNewsSitemapService(
	articleRepo *repositories.ArticleRepository,
	cache cache.CacheService,
	baseURL, siteName string,
) *GoogleNewsSitemapService {
	return &GoogleNewsSitemapService{
		articleRepo: articleRepo,
		cache:       cache,
		baseURL:     baseURL,
		siteName:    siteName,
	}
}

// Google News Sitemap XML structures
type GoogleNewsSitemap struct {
	XMLName xml.Name        `xml:"urlset"`
	Xmlns   string          `xml:"xmlns,attr"`
	News    string          `xml:"xmlns:news,attr"`
	URLs    []GoogleNewsURL `xml:"url"`
}

type GoogleNewsURL struct {
	Loc  string         `xml:"loc"`
	News GoogleNewsData `xml:"news:news"`
}

type GoogleNewsData struct {
	Publication GoogleNewsPublication `xml:"news:publication"`
	PubDate     string                `xml:"news:publication_date"`
	Title       string                `xml:"news:title"`
	Keywords    string                `xml:"news:keywords,omitempty"`
	StockTickers string               `xml:"news:stock_tickers,omitempty"`
	Genres      string                `xml:"news:genres,omitempty"`
}

type GoogleNewsPublication struct {
	Name     string `xml:"name,attr"`
	Language string `xml:"language,attr"`
}

// GenerateGoogleNewsSitemap generates Google News sitemap with 1000 article limit
func (g *GoogleNewsSitemapService) GenerateGoogleNewsSitemap(languageCode string, fileIndex int) ([]byte, error) {
	cacheKey := fmt.Sprintf("sitemap:googlenews:%s:%d", languageCode, fileIndex)
	
	// Try to get from cache first
	ctx := context.Background()
	if cached, err := g.cache.Get(ctx, cacheKey); err == nil && cached != nil {
		return cached, nil
	}

	// Calculate offset based on file index (1000 articles per file)
	limit := 1000
	offset := fileIndex * limit

	// Get recent articles (last 2 days for Google News standard)
	cutoffTime := time.Now().Add(-48 * time.Hour)
	articles, err := g.articleRepo.GetPublishedArticlesAfterTimeWithOffset(cutoffTime, languageCode, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get articles: %w", err)
	}

	// If no articles found in last 48 hours, try last 7 days as fallback
	// This ensures the sitemap is never completely empty for Google Search Console
	if len(articles) == 0 && fileIndex == 0 {
		cutoffTime = time.Now().Add(-7 * 24 * time.Hour) // 7 days
		articles, err = g.articleRepo.GetPublishedArticlesAfterTimeWithOffset(cutoffTime, languageCode, limit, offset)
		if err != nil {
			return nil, fmt.Errorf("failed to get articles (7-day fallback): %w", err)
		}
	}

	// If still no articles found, return empty sitemap
	if len(articles) == 0 {
		return g.generateEmptySitemap(), nil
	}

	// Generate sitemap
	sitemap := g.buildGoogleNewsSitemap(articles, languageCode)

	// Marshal to XML
	xmlData, err := xml.MarshalIndent(sitemap, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sitemap XML: %w", err)
	}

	// Add XML declaration
	fullXML := []byte(xml.Header + string(xmlData))

	// Cache for 1 hour
	g.cache.Set(ctx, cacheKey, fullXML, time.Hour)

	return fullXML, nil
}

// GenerateGoogleNewsSitemapIndex generates sitemap index for multiple news sitemaps
func (g *GoogleNewsSitemapService) GenerateGoogleNewsSitemapIndex(languageCode string) ([]byte, error) {
	cacheKey := fmt.Sprintf("sitemap:googlenews:index:%s", languageCode)
	
	// Try to get from cache first
	ctx := context.Background()
	if cached, err := g.cache.Get(ctx, cacheKey); err == nil && cached != nil {
		return cached, nil
	}

	// Count total articles in last 2 days
	cutoffTime := time.Now().Add(-48 * time.Hour)
	totalArticles, err := g.articleRepo.CountPublishedArticlesAfterTime(cutoffTime, languageCode)
	if err != nil {
		return nil, fmt.Errorf("failed to count articles: %w", err)
	}

	// If no articles in 48 hours, try 7 days as fallback
	if totalArticles == 0 {
		cutoffTime = time.Now().Add(-7 * 24 * time.Hour)
		totalArticles, err = g.articleRepo.CountPublishedArticlesAfterTime(cutoffTime, languageCode)
		if err != nil {
			return nil, fmt.Errorf("failed to count articles (7-day fallback): %w", err)
		}
	}

	// Calculate number of sitemap files needed (1000 articles per file)
	numFiles := int((totalArticles + 999) / 1000) // Ceiling division

	if numFiles == 0 {
		numFiles = 1 // Always have at least one sitemap file
	}

	// Generate sitemap index
	index := g.buildSitemapIndex(languageCode, numFiles)

	// Marshal to XML
	xmlData, err := xml.MarshalIndent(index, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sitemap index XML: %w", err)
	}

	// Add XML declaration
	fullXML := []byte(xml.Header + string(xmlData))

	// Cache for 1 hour
	g.cache.Set(ctx, cacheKey, fullXML, time.Hour)

	return fullXML, nil
}

// buildGoogleNewsSitemap constructs the Google News sitemap structure
func (g *GoogleNewsSitemapService) buildGoogleNewsSitemap(articles []models.Article, languageCode string) *GoogleNewsSitemap {
	sitemap := &GoogleNewsSitemap{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		News:  "http://www.google.com/schemas/sitemap-news/0.9",
		URLs:  make([]GoogleNewsURL, 0, len(articles)),
	}

	for _, article := range articles {
		newsURL := g.buildNewsURL(article, languageCode)
		sitemap.URLs = append(sitemap.URLs, newsURL)
	}

	return sitemap
}

// buildNewsURL converts an article to Google News sitemap URL entry
func (g *GoogleNewsSitemapService) buildNewsURL(article models.Article, languageCode string) GoogleNewsURL {
	// Use article's language for URL (SEO best practice)
	articleLang := article.LanguageCode
	if articleLang == "" {
		articleLang = languageCode
	}
	if articleLang == "" {
		articleLang = "en"
	}
	articleURL := fmt.Sprintf("%s/%s/article/%s", g.baseURL, articleLang, article.Slug)
	
	// Extract keywords from tags and SEO data
	keywords := make([]string, 0)
	for _, tag := range article.Tags {
		keywords = append(keywords, tag.Name)
	}
	if len(article.SEOData.Keywords) > 0 {
		keywords = append(keywords, article.SEOData.Keywords...)
	}

	// Format publication date for Google News (RFC3339 format)
	pubDate := ""
	if article.PublishedAt != nil {
		pubDate = article.PublishedAt.Format(time.RFC3339)
	}

	// Determine genres based on article content or category
	genres := "PressRelease" // Default genre
	// You could implement logic to determine genres based on category or content analysis

	return GoogleNewsURL{
		Loc: articleURL,
		News: GoogleNewsData{
			Publication: GoogleNewsPublication{
				Name:     g.siteName,
				Language: languageCode,
			},
			PubDate:      pubDate,
			Title:        article.Title,
			Keywords:     strings.Join(keywords, ","),
			StockTickers: g.extractStockTickers(article),
			Genres:       genres,
		},
	}
}

// extractStockTickers extracts stock ticker symbols from article content
func (g *GoogleNewsSitemapService) extractStockTickers(article models.Article) string {
	// This is a simplified implementation
	// In production, you might want to use NLP or regex to extract stock symbols
	// For now, we'll check if any tags contain stock-like patterns
	
	stockTickers := make([]string, 0)
	
	for _, tag := range article.Tags {
		// Simple check for stock ticker patterns (3-5 uppercase letters)
		if len(tag.Name) >= 3 && len(tag.Name) <= 5 && strings.ToUpper(tag.Name) == tag.Name {
			stockTickers = append(stockTickers, tag.Name)
		}
	}
	
	return strings.Join(stockTickers, ",")
}

// Google News Sitemap Index structures
type GoogleNewsSitemapIndex struct {
	XMLName  xml.Name                    `xml:"sitemapindex"`
	Xmlns    string                      `xml:"xmlns,attr"`
	Sitemaps []GoogleNewsSitemapRef      `xml:"sitemap"`
}

type GoogleNewsSitemapRef struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod"`
}

// buildSitemapIndex constructs the sitemap index structure
func (g *GoogleNewsSitemapService) buildSitemapIndex(languageCode string, numFiles int) *GoogleNewsSitemapIndex {
	now := time.Now().Format(time.RFC3339)
	
	index := &GoogleNewsSitemapIndex{
		Xmlns:    "http://www.sitemaps.org/schemas/sitemap/0.9",
		Sitemaps: make([]GoogleNewsSitemapRef, 0, numFiles),
	}

	for i := 0; i < numFiles; i++ {
		sitemapURL := fmt.Sprintf("%s/sitemap/googlenews-%s-%d.xml", g.baseURL, languageCode, i)
		
		index.Sitemaps = append(index.Sitemaps, GoogleNewsSitemapRef{
			Loc:     sitemapURL,
			LastMod: now,
		})
	}

	return index
}

// generateEmptySitemap generates an empty sitemap when no articles are found
func (g *GoogleNewsSitemapService) generateEmptySitemap() []byte {
	sitemap := &GoogleNewsSitemap{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		News:  "http://www.google.com/schemas/sitemap-news/0.9",
		URLs:  []GoogleNewsURL{},
	}

	xmlData, _ := xml.MarshalIndent(sitemap, "", "  ")
	return []byte(xml.Header + string(xmlData))
}

// InvalidateCache invalidates Google News sitemap caches
func (g *GoogleNewsSitemapService) InvalidateCache(languageCode string) error {
	ctx := context.Background()
	patterns := []string{
		fmt.Sprintf("sitemap:googlenews:%s:*", languageCode),
		fmt.Sprintf("sitemap:googlenews:index:%s", languageCode),
	}

	for _, pattern := range patterns {
		if err := g.cache.DeletePattern(ctx, pattern); err != nil {
			return fmt.Errorf("failed to invalidate cache pattern %s: %w", pattern, err)
		}
	}

	return nil
}

// ValidateGoogleNewsSitemap validates the generated Google News sitemap
func (g *GoogleNewsSitemapService) ValidateGoogleNewsSitemap(xmlData []byte) error {
	// Basic XML validation by attempting to unmarshal
	var sitemap GoogleNewsSitemap
	if err := xml.Unmarshal(xmlData, &sitemap); err != nil {
		return fmt.Errorf("invalid Google News sitemap XML: %w", err)
	}

	// Validate required namespaces
	if sitemap.Xmlns == "" {
		return fmt.Errorf("missing required xmlns namespace")
	}
	if sitemap.News == "" {
		return fmt.Errorf("missing required news namespace")
	}

	// Validate URLs
	for i, url := range sitemap.URLs {
		if url.Loc == "" {
			return fmt.Errorf("sitemap URL %d missing required loc", i)
		}
		if url.News.Publication.Name == "" {
			return fmt.Errorf("sitemap URL %d missing publication name", i)
		}
		if url.News.Publication.Language == "" {
			return fmt.Errorf("sitemap URL %d missing publication language", i)
		}
		if url.News.PubDate == "" {
			return fmt.Errorf("sitemap URL %d missing publication date", i)
		}
		if url.News.Title == "" {
			return fmt.Errorf("sitemap URL %d missing title", i)
		}

		// Validate date format
		if _, err := time.Parse(time.RFC3339, url.News.PubDate); err != nil {
			return fmt.Errorf("sitemap URL %d has invalid publication date format: %w", i, err)
		}
	}

	// Check article limit (Google News allows max 1000 articles per sitemap)
	if len(sitemap.URLs) > 1000 {
		return fmt.Errorf("sitemap contains %d articles, maximum allowed is 1000", len(sitemap.URLs))
	}

	return nil
}

// GetSitemapStats returns statistics about Google News sitemaps
func (g *GoogleNewsSitemapService) GetSitemapStats(languageCode string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// Count articles in last 2 days (Google News timeframe)
	cutoffTime := time.Now().Add(-48 * time.Hour)
	totalArticles, err := g.articleRepo.CountPublishedArticlesAfterTime(cutoffTime, languageCode)
	if err != nil {
		return nil, fmt.Errorf("failed to count articles: %w", err)
	}

	// Also count 7-day fallback
	fallbackUsed := false
	if totalArticles == 0 {
		cutoffTime = time.Now().Add(-7 * 24 * time.Hour)
		totalArticles, err = g.articleRepo.CountPublishedArticlesAfterTime(cutoffTime, languageCode)
		if err != nil {
			return nil, fmt.Errorf("failed to count articles (7-day fallback): %w", err)
		}
		fallbackUsed = true
	}

	numFiles := (totalArticles + 999) / 1000 // Ceiling division
	if numFiles == 0 {
		numFiles = 1
	}

	stats["total_articles"] = totalArticles
	stats["num_sitemap_files"] = numFiles
	stats["articles_per_file"] = 1000
	stats["cutoff_time"] = cutoffTime.Format(time.RFC3339)
	stats["last_updated"] = time.Now().Format(time.RFC3339)
	stats["fallback_used"] = fallbackUsed
	if fallbackUsed {
		stats["note"] = "No articles in last 48 hours, using 7-day fallback"
	}

	return stats, nil
}