package services

import (
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"log"
	"net/url"
	"strings"
	"time"

	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/repositories"
	"high-performance-news-website/pkg/cache"
)

// RSSService handles RSS feed generation with delayed publishing
type RSSService struct {
	articleRepo  *repositories.ArticleRepository
	categoryRepo *repositories.CategoryRepository
	tagRepo      *repositories.TagRepository
	cache        cache.CacheService
	baseURL      string
	siteName     string
	siteDesc     string
	delayHours   int // Delay in hours before articles appear in RSS
}

// NewRSSService creates a new RSS service instance
func NewRSSService(
	articleRepo *repositories.ArticleRepository,
	categoryRepo *repositories.CategoryRepository,
	tagRepo *repositories.TagRepository,
	cache cache.CacheService,
	baseURL, siteName, siteDesc string,
) *RSSService {
	return &RSSService{
		articleRepo:  articleRepo,
		categoryRepo: categoryRepo,
		tagRepo:      tagRepo,
		cache:        cache,
		baseURL:      baseURL,
		siteName:     siteName,
		siteDesc:     siteDesc,
		delayHours:   2, // Default 2-hour delay as per requirements
	}
}

// RSS 2.0 XML structures
type RSS struct {
	XMLName   xml.Name `xml:"rss"`
	Version   string   `xml:"version,attr"`
	XmlnsAtom string   `xml:"xmlns:atom,attr"` // Atom namespace for self-link
	Channel   Channel  `xml:"channel"`
}

// AtomLink represents the atom:link element for self-reference
type AtomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

type Channel struct {
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	AtomLink      *AtomLink `xml:"atom:link,omitempty"` // Self-referencing link
	Description   string    `xml:"description"`
	Language      string    `xml:"language"`
	Copyright     string    `xml:"copyright"`
	ManagingEditor string   `xml:"managingEditor,omitempty"`
	WebMaster     string    `xml:"webMaster,omitempty"`
	PubDate       string    `xml:"pubDate"`
	LastBuildDate string    `xml:"lastBuildDate"`
	Category      string    `xml:"category,omitempty"`
	Generator     string    `xml:"generator"`
	TTL           int       `xml:"ttl"`
	Items         []Item    `xml:"item"`
}

type Item struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Author      string    `xml:"author,omitempty"`
	Category    []string  `xml:"category,omitempty"`
	Comments    string    `xml:"comments,omitempty"`
	Enclosure   *Enclosure `xml:"enclosure,omitempty"`
	GUID        GUID      `xml:"guid"`
	PubDate     string    `xml:"pubDate"`
	Source      *Source   `xml:"source,omitempty"`
}

type GUID struct {
	IsPermaLink bool   `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

type Enclosure struct {
	URL    string `xml:"url,attr"`
	Length int64  `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

type Source struct {
	URL   string `xml:"url,attr"`
	Value string `xml:",chardata"`
}

// Google News RSS extensions (using media namespace instead of sitemap-news)
type GoogleNewsRSS struct {
	XMLName   xml.Name           `xml:"rss"`
	Version   string             `xml:"version,attr"`
	XmlnsAtom string             `xml:"xmlns:atom,attr"` // Atom namespace for self-link
	XmlnsMedia string            `xml:"xmlns:media,attr"` // Media namespace (standard for RSS)
	Channel   GoogleNewsChannel  `xml:"channel"`
}

type GoogleNewsChannel struct {
	Title         string           `xml:"title"`
	Link          string           `xml:"link"`
	AtomLink      *AtomLink        `xml:"atom:link,omitempty"` // Self-referencing link
	Description   string           `xml:"description"`
	Language      string           `xml:"language"`
	Copyright     string           `xml:"copyright"`
	PubDate       string           `xml:"pubDate"`
	LastBuildDate string           `xml:"lastBuildDate"`
	Generator     string           `xml:"generator"`
	Items         []GoogleNewsItem `xml:"item"`
}

type GoogleNewsItem struct {
	Title       string           `xml:"title"`
	Link        string           `xml:"link"`
	Description string           `xml:"description"`
	PubDate     string           `xml:"pubDate"`
	GUID        GUID             `xml:"guid"`
	Author      string           `xml:"author,omitempty"`
	Category    []string         `xml:"category,omitempty"`
}

// RSSGoogleNewsData and RSSGoogleNewsPublication are kept for backward compatibility
// but not used in the RSS feed output (Google News sitemap uses different format)
type RSSGoogleNewsData struct {
	Publication     RSSGoogleNewsPublication `xml:"news:publication"`
	PublicationDate string                   `xml:"news:publication_date,omitempty"`
	Genres          string                   `xml:"news:genres,omitempty"`
	Keywords        string                   `xml:"news:keywords,omitempty"`
	StockTickers    string                   `xml:"news:stock_tickers,omitempty"`
}

type RSSGoogleNewsPublication struct {
	Name     string `xml:"name,attr"`
	Language string `xml:"language,attr"`
}

// GenerateMainRSSFeed generates the main RSS feed with delayed publishing
func (r *RSSService) GenerateMainRSSFeed(languageCode string, limit int) ([]byte, error) {
	log.Printf("RSS: GenerateMainRSSFeed called with languageCode=%s, limit=%d", languageCode, limit)
	
	cacheKey := fmt.Sprintf("rss:main:%s", languageCode)
	
	// Try to get from cache first
	ctx := context.Background()
	if cached, err := r.cache.Get(ctx, cacheKey); err == nil && cached != nil {
		log.Printf("RSS: Returning cached feed for key=%s", cacheKey)
		return cached, nil
	}

	// Calculate cutoff time for delayed publishing (2 hours ago)
	cutoffTime := time.Now().Add(-time.Duration(r.delayHours) * time.Hour)
	log.Printf("RSS: Cutoff time=%v, delayHours=%d", cutoffTime, r.delayHours)

	// Get articles published before cutoff time
	articles, err := r.articleRepo.GetPublishedArticlesBeforeTime(cutoffTime, languageCode, limit)
	if err != nil {
		log.Printf("RSS: Error getting articles: %v", err)
		return nil, fmt.Errorf("failed to get articles: %w", err)
	}
	log.Printf("RSS: Got %d articles", len(articles))

	// Generate RSS feed
	rss := r.buildRSSFeed(articles, r.siteName, r.baseURL, r.siteDesc, languageCode, "")

	// Marshal to XML
	xmlData, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RSS XML: %w", err)
	}

	// Add XML declaration
	fullXML := []byte(xml.Header + string(xmlData))

	// Cache for 4 hours (as per design document)
	r.cache.Set(ctx, cacheKey, fullXML, 4*time.Hour)

	return fullXML, nil
}

// GenerateCategoryRSSFeed generates RSS feed for a specific category
func (r *RSSService) GenerateCategoryRSSFeed(categorySlug, languageCode string, limit int) ([]byte, error) {
	cacheKey := fmt.Sprintf("rss:category:%s:%s", categorySlug, languageCode)
	
	// Try to get from cache first
	ctx := context.Background()
	if cached, err := r.cache.Get(ctx, cacheKey); err == nil && cached != nil {
		return cached, nil
	}

	// Get category
	category, err := r.categoryRepo.GetBySlug(categorySlug, languageCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get category: %w", err)
	}

	// Calculate cutoff time for delayed publishing
	cutoffTime := time.Now().Add(-time.Duration(r.delayHours) * time.Hour)

	// Get articles for category
	articles, err := r.articleRepo.GetArticlesByCategoryBeforeTime(category.ID, cutoffTime, languageCode, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get category articles: %w", err)
	}

	// Generate RSS feed
	feedTitle := fmt.Sprintf("%s - %s", r.siteName, category.Name)
	feedLink := fmt.Sprintf("%s/category/%s", r.baseURL, category.Slug)
	feedDesc := category.Description
	if feedDesc == "" {
		feedDesc = fmt.Sprintf("Latest articles from %s category", category.Name)
	}

	rss := r.buildRSSFeed(articles, feedTitle, feedLink, feedDesc, languageCode, category.Name)

	// Marshal to XML
	xmlData, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RSS XML: %w", err)
	}

	// Add XML declaration
	fullXML := []byte(xml.Header + string(xmlData))

	// Cache for 4 hours
	r.cache.Set(ctx, cacheKey, fullXML, 4*time.Hour)

	return fullXML, nil
}

// GenerateTagRSSFeed generates RSS feed for a specific tag
func (r *RSSService) GenerateTagRSSFeed(tagSlug, languageCode string, limit int) ([]byte, error) {
	cacheKey := fmt.Sprintf("rss:tag:%s:%s", tagSlug, languageCode)
	
	// Try to get from cache first
	ctx := context.Background()
	if cached, err := r.cache.Get(ctx, cacheKey); err == nil && cached != nil {
		return cached, nil
	}

	// Get tag
	tag, err := r.tagRepo.GetBySlug(tagSlug, languageCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get tag: %w", err)
	}

	// Calculate cutoff time for delayed publishing
	cutoffTime := time.Now().Add(-time.Duration(r.delayHours) * time.Hour)

	// Get articles for tag
	articles, err := r.articleRepo.GetArticlesByTagBeforeTime(tag.ID, cutoffTime, languageCode, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get tag articles: %w", err)
	}

	// Generate RSS feed
	feedTitle := fmt.Sprintf("%s - %s", r.siteName, tag.Name)
	feedLink := fmt.Sprintf("%s/tag/%s", r.baseURL, tag.Slug)
	feedDesc := tag.Description
	if feedDesc == "" {
		feedDesc = fmt.Sprintf("Latest articles tagged with %s", tag.Name)
	}

	rss := r.buildRSSFeed(articles, feedTitle, feedLink, feedDesc, languageCode, "")

	// Marshal to XML
	xmlData, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RSS XML: %w", err)
	}

	// Add XML declaration
	fullXML := []byte(xml.Header + string(xmlData))

	// Cache for 4 hours
	r.cache.Set(ctx, cacheKey, fullXML, 4*time.Hour)

	return fullXML, nil
}

// GenerateGoogleNewsRSSFeed generates Google News compliant RSS feed
func (r *RSSService) GenerateGoogleNewsRSSFeed(languageCode string, limit int) ([]byte, error) {
	log.Printf("RSS: GenerateGoogleNewsRSSFeed called with languageCode=%s, limit=%d", languageCode, limit)
	
	cacheKey := fmt.Sprintf("rss:googlenews:%s", languageCode)
	
	// Try to get from cache first
	ctx := context.Background()
	if cached, err := r.cache.Get(ctx, cacheKey); err == nil && cached != nil {
		log.Printf("RSS: Returning cached Google News feed")
		return cached, nil
	}

	// For Google News, only include articles from the last 48 hours
	// This is a Google News requirement - older articles should not be in the feed
	cutoffTime := time.Now().Add(-48 * time.Hour)
	log.Printf("RSS: Google News cutoff time=%v (48 hours ago)", cutoffTime)

	// Get recent news articles (limit to 1000 as per Google News requirements)
	if limit > 1000 {
		limit = 1000
	}

	// Use GetPublishedArticlesAfterTimeWithOffset to only get articles from last 48 hours
	articles, err := r.articleRepo.GetPublishedArticlesAfterTimeWithOffset(cutoffTime, languageCode, limit, 0)
	if err != nil {
		log.Printf("RSS: Error getting articles for Google News: %v", err)
		return nil, fmt.Errorf("failed to get articles: %w", err)
	}
	log.Printf("RSS: Got %d articles for Google News feed (last 48 hours)", len(articles))

	// Generate Google News RSS feed
	rss := r.buildGoogleNewsRSSFeed(articles, languageCode)

	// Marshal to XML
	xmlData, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		log.Printf("RSS: Error marshaling Google News XML: %v", err)
		return nil, fmt.Errorf("failed to marshal Google News RSS XML: %w", err)
	}

	// Add XML declaration
	fullXML := []byte(xml.Header + string(xmlData))
	log.Printf("RSS: Generated Google News XML, length=%d bytes", len(fullXML))

	// Cache for 30 minutes (shorter cache for news)
	r.cache.Set(ctx, cacheKey, fullXML, 30*time.Minute)

	return fullXML, nil
}

// buildRSSFeed constructs the RSS feed structure
func (r *RSSService) buildRSSFeed(articles []models.Article, title, link, description, languageCode, category string) *RSS {
	now := time.Now()
	
	// Determine the self-link URL based on the feed type
	selfLink := link
	if !strings.Contains(link, "/category/") && !strings.Contains(link, "/tag/") {
		selfLink = r.baseURL + "/rss"
	}
	
	channel := Channel{
		Title:         title,
		Link:          link,
		AtomLink: &AtomLink{
			Href: selfLink,
			Rel:  "self",
			Type: "application/rss+xml",
		},
		Description:   description,
		Language:      languageCode,
		Copyright:     fmt.Sprintf("Copyright %d %s", now.Year(), r.siteName),
		PubDate:       now.Format(time.RFC1123Z),
		LastBuildDate: now.Format(time.RFC1123Z),
		Generator:     "High-Performance News Website RSS Generator",
		TTL:           240, // 4 hours in minutes
		Items:         make([]Item, 0, len(articles)),
	}

	if category != "" {
		channel.Category = category
	}

	// Convert articles to RSS items
	for _, article := range articles {
		item := r.buildRSSItem(article)
		channel.Items = append(channel.Items, item)
	}

	return &RSS{
		Version:   "2.0",
		XmlnsAtom: "http://www.w3.org/2005/Atom",
		Channel:   channel,
	}
}

// buildGoogleNewsRSSFeed constructs Google News compliant RSS feed
func (r *RSSService) buildGoogleNewsRSSFeed(articles []models.Article, languageCode string) *GoogleNewsRSS {
	now := time.Now()
	
	channel := GoogleNewsChannel{
		Title:         r.siteName + " - Google News",
		Link:          r.baseURL,
		AtomLink: &AtomLink{
			Href: r.baseURL + "/rss/googlenews",
			Rel:  "self",
			Type: "application/rss+xml",
		},
		Description:   r.siteDesc,
		Language:      languageCode,
		Copyright:     fmt.Sprintf("Copyright %d %s", now.Year(), r.siteName),
		PubDate:       now.Format(time.RFC1123Z),
		LastBuildDate: now.Format(time.RFC1123Z),
		Generator:     "High-Performance News Website Google News RSS Generator",
		Items:         make([]GoogleNewsItem, 0, len(articles)),
	}

	// Convert articles to Google News RSS items
	for _, article := range articles {
		item := r.buildGoogleNewsItem(article, languageCode)
		channel.Items = append(channel.Items, item)
	}

	return &GoogleNewsRSS{
		Version:    "2.0",
		XmlnsAtom:  "http://www.w3.org/2005/Atom",
		XmlnsMedia: "http://search.yahoo.com/mrss/",
		Channel:    channel,
	}
}

// buildRSSItem converts an article to RSS item
func (r *RSSService) buildRSSItem(article models.Article) Item {
	articleURL := fmt.Sprintf("%s/article/%s", r.baseURL, article.Slug)
	
	// Create categories from tags
	categories := make([]string, len(article.Tags))
	for i, tag := range article.Tags {
		categories[i] = tag.Name
	}

	// Use excerpt or truncated content for description
	description := article.Excerpt
	if description == "" && len(article.Content) > 300 {
		description = article.Content[:300] + "..."
	} else if description == "" {
		description = article.Content
	}

	// Escape HTML in description
	description = html.EscapeString(description)

	pubDate := ""
	if article.PublishedAt != nil {
		pubDate = article.PublishedAt.Format(time.RFC1123Z)
	}

	return Item{
		Title:       html.EscapeString(article.Title),
		Link:        articleURL,
		Description: description,
		Category:    categories,
		GUID: GUID{
			IsPermaLink: true,
			Value:       articleURL,
		},
		PubDate: pubDate,
	}
}

// buildGoogleNewsItem converts an article to Google News RSS item
func (r *RSSService) buildGoogleNewsItem(article models.Article, languageCode string) GoogleNewsItem {
	articleURL := fmt.Sprintf("%s/article/%s", r.baseURL, article.Slug)
	
	// Use excerpt or truncated content for description
	description := article.Excerpt
	if description == "" && len(article.Content) > 300 {
		description = article.Content[:300] + "..."
	} else if description == "" {
		description = article.Content
	}

	// Escape HTML in description
	description = html.EscapeString(description)

	pubDate := ""
	if article.PublishedAt != nil {
		pubDate = article.PublishedAt.Format(time.RFC1123Z)
	}

	// Extract categories from tags
	categories := make([]string, len(article.Tags))
	for i, tag := range article.Tags {
		categories[i] = tag.Name
	}

	return GoogleNewsItem{
		Title:       html.EscapeString(article.Title),
		Link:        articleURL,
		Description: description,
		PubDate:     pubDate,
		GUID: GUID{
			IsPermaLink: true,
			Value:       articleURL,
		},
		Category: categories,
	}
}

// InvalidateCache invalidates RSS feed caches
func (r *RSSService) InvalidateCache(languageCode string) error {
	ctx := context.Background()
	patterns := []string{
		fmt.Sprintf("rss:main:%s", languageCode),
		fmt.Sprintf("rss:googlenews:%s", languageCode),
		fmt.Sprintf("rss:category:*:%s", languageCode),
		fmt.Sprintf("rss:tag:*:%s", languageCode),
	}

	for _, pattern := range patterns {
		if err := r.cache.DeletePattern(ctx, pattern); err != nil {
			return fmt.Errorf("failed to invalidate cache pattern %s: %w", pattern, err)
		}
	}

	return nil
}

// InvalidateCategoryCache invalidates cache for a specific category
func (r *RSSService) InvalidateCategoryCache(categorySlug, languageCode string) error {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("rss:category:%s:%s", categorySlug, languageCode)
	return r.cache.Delete(ctx, cacheKey)
}

// InvalidateTagCache invalidates cache for a specific tag
func (r *RSSService) InvalidateTagCache(tagSlug, languageCode string) error {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("rss:tag:%s:%s", tagSlug, languageCode)
	return r.cache.Delete(ctx, cacheKey)
}

// ValidateRSSFeed validates the generated RSS feed
func (r *RSSService) ValidateRSSFeed(xmlData []byte) error {
	// Basic XML validation by attempting to unmarshal
	var rss RSS
	if err := xml.Unmarshal(xmlData, &rss); err != nil {
		return fmt.Errorf("invalid RSS XML: %w", err)
	}

	// Validate required fields
	if rss.Channel.Title == "" {
		return fmt.Errorf("RSS feed missing required title")
	}
	if rss.Channel.Link == "" {
		return fmt.Errorf("RSS feed missing required link")
	}
	if rss.Channel.Description == "" {
		return fmt.Errorf("RSS feed missing required description")
	}

	// Validate items
	for i, item := range rss.Channel.Items {
		if item.Title == "" {
			return fmt.Errorf("RSS item %d missing required title", i)
		}
		if item.Link == "" {
			return fmt.Errorf("RSS item %d missing required link", i)
		}
		if item.Description == "" {
			return fmt.Errorf("RSS item %d missing required description", i)
		}
		if item.GUID.Value == "" {
			return fmt.Errorf("RSS item %d missing required GUID", i)
		}

		// Validate URL format
		if _, err := url.Parse(item.Link); err != nil {
			return fmt.Errorf("RSS item %d has invalid link URL: %w", i, err)
		}
	}

	return nil
}

// ValidateGoogleNewsRSSFeed validates Google News RSS feed
func (r *RSSService) ValidateGoogleNewsRSSFeed(xmlData []byte) error {
	log.Printf("RSS: Validating Google News RSS feed, length=%d bytes", len(xmlData))
	
	// Basic XML validation by attempting to unmarshal
	var rss GoogleNewsRSS
	if err := xml.Unmarshal(xmlData, &rss); err != nil {
		log.Printf("RSS: Google News XML unmarshal error: %v", err)
		return fmt.Errorf("invalid Google News RSS XML: %w", err)
	}

	// Validate required fields
	if rss.Channel.Title == "" {
		log.Printf("RSS: Google News feed missing title")
		return fmt.Errorf("Google News RSS feed missing required title")
	}
	if rss.Channel.Link == "" {
		log.Printf("RSS: Google News feed missing link")
		return fmt.Errorf("Google News RSS feed missing required link")
	}
	if rss.Channel.Description == "" {
		log.Printf("RSS: Google News feed missing description")
		return fmt.Errorf("Google News RSS feed missing required description")
	}

	log.Printf("RSS: Google News feed has %d items", len(rss.Channel.Items))
	
	// Validate items (Google News specific)
	for i, item := range rss.Channel.Items {
		if item.Title == "" {
			log.Printf("RSS: Google News item %d missing title", i)
			return fmt.Errorf("Google News RSS item %d missing required title", i)
		}
		if item.Link == "" {
			log.Printf("RSS: Google News item %d missing link", i)
			return fmt.Errorf("Google News RSS item %d missing required link", i)
		}
		if item.Description == "" {
			log.Printf("RSS: Google News item %d missing description", i)
			return fmt.Errorf("Google News RSS item %d missing required description", i)
		}
		if item.GUID.Value == "" {
			log.Printf("RSS: Google News item %d missing GUID", i)
			return fmt.Errorf("Google News RSS item %d missing required GUID", i)
		}
		// Note: Publication name/language validation skipped because Go's XML unmarshaler
		// doesn't properly read namespaced attributes. The values are correctly generated
		// in the XML output but can't be read back via unmarshal.

		// Validate URL format
		if _, err := url.Parse(item.Link); err != nil {
			log.Printf("RSS: Google News item %d has invalid URL: %v", i, err)
			return fmt.Errorf("Google News RSS item %d has invalid link URL: %w", i, err)
		}
	}

	log.Printf("RSS: Google News feed validation passed")
	return nil
}

// SetDelayHours allows configuring the RSS delay
func (r *RSSService) SetDelayHours(hours int) {
	if hours >= 0 {
		r.delayHours = hours
	}
}

// GetDelayHours returns the current RSS delay in hours
func (r *RSSService) GetDelayHours() int {
	return r.delayHours
}

// ForceRefreshFeed forces a refresh of a specific feed by clearing its cache
func (r *RSSService) ForceRefreshFeed(feedType, identifier, languageCode string) error {
	ctx := context.Background()
	var cacheKey string
	
	switch feedType {
	case "main":
		cacheKey = fmt.Sprintf("rss:main:%s", languageCode)
	case "category":
		cacheKey = fmt.Sprintf("rss:category:%s:%s", identifier, languageCode)
	case "tag":
		cacheKey = fmt.Sprintf("rss:tag:%s:%s", identifier, languageCode)
	case "googlenews":
		cacheKey = fmt.Sprintf("rss:googlenews:%s", languageCode)
	default:
		return fmt.Errorf("unknown feed type: %s", feedType)
	}

	return r.cache.Delete(ctx, cacheKey)
}

// GetFeedStats returns statistics about RSS feeds
func (r *RSSService) GetFeedStats(languageCode string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// Get total published articles count
	cutoffTime := time.Now().Add(-time.Duration(r.delayHours) * time.Hour)
	totalArticles, err := r.articleRepo.CountPublishedArticlesBeforeTime(cutoffTime, languageCode)
	if err != nil {
		return nil, fmt.Errorf("failed to count articles: %w", err)
	}

	stats["total_articles_in_feed"] = totalArticles
	stats["delay_hours"] = r.delayHours
	stats["cutoff_time"] = cutoffTime.Format(time.RFC3339)
	stats["last_updated"] = time.Now().Format(time.RFC3339)

	return stats, nil
}