package services

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"high-performance-news-website/internal/models"
)

// SitemapService handles sitemap generation and management
type SitemapService struct {
	baseURL     string
	maxURLs     int // Maximum URLs per sitemap file (50,000 for regular, 1,000 for news)
	maxNewsURLs int
}

// NewSitemapService creates a new sitemap service instance
func NewSitemapService(baseURL string) *SitemapService {
	return &SitemapService{
		baseURL:     strings.TrimSuffix(baseURL, "/"),
		maxURLs:     50000,
		maxNewsURLs: 1000,
	}
}

// SitemapIndex represents the main sitemap index
type SitemapIndex struct {
	XMLName xml.Name `xml:"sitemapindex"`
	Xmlns   string   `xml:"xmlns,attr"`
	Sitemaps []SitemapReference `xml:"sitemap"`
}

// SitemapReference represents a reference to a sitemap file
type SitemapReference struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

// URLSet represents a sitemap with URLs
type URLSet struct {
	XMLName xml.Name `xml:"urlset"`
	Xmlns   string   `xml:"xmlns,attr"`
	URLs    []URL    `xml:"url"`
}

// NewsURLSet represents a news sitemap
type NewsURLSet struct {
	XMLName xml.Name `xml:"urlset"`
	Xmlns   string   `xml:"xmlns,attr"`
	News    string   `xml:"xmlns:news,attr"`
	URLs    []NewsURL `xml:"url"`
}

// URL represents a single URL in a sitemap
type URL struct {
	Loc        string  `xml:"loc"`
	LastMod    string  `xml:"lastmod,omitempty"`
	ChangeFreq string  `xml:"changefreq,omitempty"`
	Priority   float64 `xml:"priority,omitempty"`
}

// NewsURL represents a URL in a news sitemap
type NewsURL struct {
	Loc  string `xml:"loc"`
	News News   `xml:"news:news"`
}

// News represents news-specific information
type News struct {
	Publication Publication `xml:"news:publication"`
	PublicationDate string `xml:"news:publication_date"`
	Title       string     `xml:"news:title"`
	Keywords    string     `xml:"news:keywords,omitempty"`
	StockTickers string    `xml:"news:stock_tickers,omitempty"`
}

// Publication represents the news publication information
type Publication struct {
	Name     string `xml:"news:name"`
	Language string `xml:"news:language"`
}

// SitemapData contains all data needed for sitemap generation
type SitemapData struct {
	Articles   []models.Article
	Categories []models.Category
	Tags       []models.Tag
	LastUpdate time.Time
}

// GenerateSitemapIndex creates the main sitemap index
func (s *SitemapService) GenerateSitemapIndex(data *SitemapData) (*SitemapIndex, error) {
	index := &SitemapIndex{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
	}

	lastMod := data.LastUpdate.Format(time.RFC3339)

	// Add main sitemap
	index.Sitemaps = append(index.Sitemaps, SitemapReference{
		Loc:     s.baseURL + "/sitemap-main.xml",
		LastMod: lastMod,
	})

	// Add article sitemaps (split if needed)
	articleCount := len(data.Articles)
	if articleCount > 0 {
		sitemapCount := (articleCount + s.maxURLs - 1) / s.maxURLs
		for i := 0; i < sitemapCount; i++ {
			index.Sitemaps = append(index.Sitemaps, SitemapReference{
				Loc:     fmt.Sprintf("%s/sitemap-articles-%d.xml", s.baseURL, i+1),
				LastMod: lastMod,
			})
		}
	}

	// Add news sitemap
	recentArticles := s.getRecentArticles(data.Articles, 48*time.Hour)
	if len(recentArticles) > 0 {
		newsCount := (len(recentArticles) + s.maxNewsURLs - 1) / s.maxNewsURLs
		for i := 0; i < newsCount; i++ {
			index.Sitemaps = append(index.Sitemaps, SitemapReference{
				Loc:     fmt.Sprintf("%s/sitemap-news-%d.xml", s.baseURL, i+1),
				LastMod: lastMod,
			})
		}
	}

	// Add category sitemap
	if len(data.Categories) > 0 {
		index.Sitemaps = append(index.Sitemaps, SitemapReference{
			Loc:     s.baseURL + "/sitemap-categories.xml",
			LastMod: lastMod,
		})
	}

	// Add tag sitemap
	if len(data.Tags) > 0 {
		index.Sitemaps = append(index.Sitemaps, SitemapReference{
			Loc:     s.baseURL + "/sitemap-tags.xml",
			LastMod: lastMod,
		})
	}

	return index, nil
}

// GenerateMainSitemap creates the main sitemap with static pages
func (s *SitemapService) GenerateMainSitemap() (*URLSet, error) {
	sitemap := &URLSet{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
	}

	now := time.Now().Format(time.RFC3339)

	// Static pages
	staticPages := []struct {
		path       string
		changeFreq string
		priority   float64
	}{
		{"/", "daily", 1.0},
		{"/categories", "weekly", 0.8},
		{"/tags", "weekly", 0.8},
		{"/trending", "hourly", 0.9},
		{"/latest", "hourly", 0.9},
		{"/popular", "daily", 0.8},
		{"/about", "monthly", 0.5},
		{"/contact", "monthly", 0.5},
	}

	for _, page := range staticPages {
		sitemap.URLs = append(sitemap.URLs, URL{
			Loc:        s.baseURL + page.path,
			LastMod:    now,
			ChangeFreq: page.changeFreq,
			Priority:   page.priority,
		})
	}

	return sitemap, nil
}

// GenerateArticleSitemap creates sitemap for articles
func (s *SitemapService) GenerateArticleSitemap(articles []models.Article, pageNum int) (*URLSet, error) {
	sitemap := &URLSet{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
	}

	start := (pageNum - 1) * s.maxURLs
	end := start + s.maxURLs
	if end > len(articles) {
		end = len(articles)
	}

	if start >= len(articles) {
		return sitemap, nil // Empty sitemap
	}

	for _, article := range articles[start:end] {
		if article.Status != "published" || article.PublishedAt == nil {
			continue
		}

		url := URL{
			Loc:        s.GetArticleURL(article.Slug),
			LastMod:    article.UpdatedAt.Format(time.RFC3339),
			ChangeFreq: s.getArticleChangeFreq(article),
			Priority:   s.getArticlePriority(article),
		}

		sitemap.URLs = append(sitemap.URLs, url)
	}

	return sitemap, nil
}

// GenerateNewsSitemap creates Google News compliant sitemap
func (s *SitemapService) GenerateNewsSitemap(articles []models.Article, siteName, defaultLang string, pageNum int) (*NewsURLSet, error) {
	sitemap := &NewsURLSet{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		News:  "http://www.google.com/schemas/sitemap-news/0.9",
	}

	// Filter recent articles (last 2 days for Google News)
	recentArticles := s.getRecentArticles(articles, 48*time.Hour)

	start := (pageNum - 1) * s.maxNewsURLs
	end := start + s.maxNewsURLs
	if end > len(recentArticles) {
		end = len(recentArticles)
	}

	if start >= len(recentArticles) {
		return sitemap, nil // Empty sitemap
	}

	for _, article := range recentArticles[start:end] {
		if article.Status != "published" || article.PublishedAt == nil {
			continue
		}

		// Generate keywords from tags and SEO data
		keywords := s.generateNewsKeywords(article)

		newsURL := NewsURL{
			Loc: s.GetArticleURL(article.Slug),
			News: News{
				Publication: Publication{
					Name:     siteName,
					Language: s.getLanguageCode(article.LanguageCode, defaultLang),
				},
				PublicationDate: article.PublishedAt.Format(time.RFC3339),
				Title:           article.Title,
				Keywords:        keywords,
			},
		}

		sitemap.URLs = append(sitemap.URLs, newsURL)
	}

	return sitemap, nil
}

// GenerateCategorySitemap creates sitemap for categories
func (s *SitemapService) GenerateCategorySitemap(categories []models.Category) (*URLSet, error) {
	sitemap := &URLSet{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
	}

	for _, category := range categories {
		url := URL{
			Loc:        s.GetCategoryURL(category.Slug),
			LastMod:    category.UpdatedAt.Format(time.RFC3339),
			ChangeFreq: "weekly",
			Priority:   0.7,
		}

		sitemap.URLs = append(sitemap.URLs, url)
	}

	return sitemap, nil
}

// GenerateTagSitemap creates sitemap for tags
func (s *SitemapService) GenerateTagSitemap(tags []models.Tag) (*URLSet, error) {
	sitemap := &URLSet{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
	}

	for _, tag := range tags {
		url := URL{
			Loc:        s.GetTagURL(tag.Slug),
			LastMod:    tag.UpdatedAt.Format(time.RFC3339),
			ChangeFreq: "weekly",
			Priority:   0.6,
		}

		sitemap.URLs = append(sitemap.URLs, url)
	}

	return sitemap, nil
}

// RenderXML converts sitemap to XML string
func (s *SitemapService) RenderXML(sitemap interface{}) (string, error) {
	xmlBytes, err := xml.MarshalIndent(sitemap, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal sitemap: %w", err)
	}

	return xml.Header + string(xmlBytes), nil
}

// Helper functions

func (s *SitemapService) getRecentArticles(articles []models.Article, duration time.Duration) []models.Article {
	cutoff := time.Now().Add(-duration)
	var recent []models.Article

	for _, article := range articles {
		if article.Status == "published" && 
		   article.PublishedAt != nil && 
		   article.PublishedAt.After(cutoff) {
			recent = append(recent, article)
		}
	}

	return recent
}

func (s *SitemapService) getArticleChangeFreq(article models.Article) string {
	if article.PublishedAt == nil {
		return "monthly"
	}

	age := time.Since(*article.PublishedAt)
	
	if age < 24*time.Hour {
		return "hourly"
	} else if age < 7*24*time.Hour {
		return "daily"
	} else if age < 30*24*time.Hour {
		return "weekly"
	}
	
	return "monthly"
}

func (s *SitemapService) getArticlePriority(article models.Article) float64 {
	if article.PublishedAt == nil {
		return 0.5
	}

	age := time.Since(*article.PublishedAt)
	
	// Higher priority for newer articles
	if age < 24*time.Hour {
		return 0.9
	} else if age < 7*24*time.Hour {
		return 0.8
	} else if age < 30*24*time.Hour {
		return 0.7
	}
	
	// Consider view count for older articles
	if article.ViewCount > 1000 {
		return 0.7
	} else if article.ViewCount > 100 {
		return 0.6
	}
	
	return 0.5
}

func (s *SitemapService) generateNewsKeywords(article models.Article) string {
	var keywords []string

	// Add keywords from SEO data
	if len(article.SEOData.Keywords) > 0 {
		keywords = append(keywords, article.SEOData.Keywords...)
	}

	// Add tag names as keywords
	for _, tag := range article.Tags {
		keywords = append(keywords, tag.Name)
	}

	// Limit to 10 keywords and join with commas
	if len(keywords) > 10 {
		keywords = keywords[:10]
	}

	return strings.Join(keywords, ", ")
}

func (s *SitemapService) getLanguageCode(articleLang, defaultLang string) string {
	if articleLang != "" {
		return articleLang
	}
	return defaultLang
}

// URL generation helpers (matching SEOService)
func (s *SitemapService) GetArticleURL(slug string) string {
	return fmt.Sprintf("%s/article/%s", s.baseURL, slug)
}

func (s *SitemapService) GetCategoryURL(slug string) string {
	return fmt.Sprintf("%s/category/%s", s.baseURL, slug)
}

func (s *SitemapService) GetTagURL(slug string) string {
	return fmt.Sprintf("%s/tag/%s", s.baseURL, slug)
}

// SitemapManager handles sitemap updates and caching with instant updates
type SitemapManager struct {
	sitemapService *SitemapService
	cacheService   CacheService // Cache interface for storing generated sitemaps
	lastUpdate     time.Time
	updateChannel  chan SitemapUpdateEvent
	isRunning      bool
}

// CacheService interface for sitemap caching
type CacheService interface {
	Set(key string, value []byte, ttl time.Duration) error
	Get(key string) ([]byte, error)
	Delete(key string) error
	DeletePattern(pattern string) error
}

// SitemapUpdateEvent represents a sitemap update event
type SitemapUpdateEvent struct {
	Type      string      // "article", "category", "tag"
	Action    string      // "create", "update", "delete"
	EntityID  uint64      // ID of the entity
	Data      interface{} // Entity data
	Timestamp time.Time
}

// NewSitemapManager creates a new sitemap manager with instant updates
func NewSitemapManager(sitemapService *SitemapService, cacheService CacheService) *SitemapManager {
	return &SitemapManager{
		sitemapService: sitemapService,
		cacheService:   cacheService,
		lastUpdate:     time.Now(),
		updateChannel:  make(chan SitemapUpdateEvent, 1000), // Buffer for high-volume updates
		isRunning:      false,
	}
}

// StartUpdateProcessor starts the background sitemap update processor
func (sm *SitemapManager) StartUpdateProcessor() {
	if sm.isRunning {
		return
	}
	
	sm.isRunning = true
	go sm.processUpdates()
}

// StopUpdateProcessor stops the background sitemap update processor
func (sm *SitemapManager) StopUpdateProcessor() {
	if !sm.isRunning {
		return
	}
	
	sm.isRunning = false
	close(sm.updateChannel)
}

// processUpdates processes sitemap update events in the background
func (sm *SitemapManager) processUpdates() {
	ticker := time.NewTicker(30 * time.Second) // Batch updates every 30 seconds
	defer ticker.Stop()
	
	var pendingUpdates []SitemapUpdateEvent
	
	for sm.isRunning {
		select {
		case event, ok := <-sm.updateChannel:
			if !ok {
				// Channel closed, process remaining updates and exit
				if len(pendingUpdates) > 0 {
					sm.processBatchUpdates(pendingUpdates)
				}
				return
			}
			pendingUpdates = append(pendingUpdates, event)
			
		case <-ticker.C:
			if len(pendingUpdates) > 0 {
				sm.processBatchUpdates(pendingUpdates)
				pendingUpdates = nil
			}
		}
	}
}

// processBatchUpdates processes a batch of sitemap updates
func (sm *SitemapManager) processBatchUpdates(updates []SitemapUpdateEvent) {
	// Group updates by type for efficient processing
	articleUpdates := make(map[string]bool)
	categoryUpdates := make(map[string]bool)
	tagUpdates := make(map[string]bool)
	
	for _, update := range updates {
		switch update.Type {
		case "article":
			articleUpdates[update.Action] = true
		case "category":
			categoryUpdates[update.Action] = true
		case "tag":
			tagUpdates[update.Action] = true
		}
	}
	
	// Update affected sitemaps
	if len(articleUpdates) > 0 {
		sm.updateArticleSitemaps()
		sm.updateNewsSitemaps()
	}
	
	if len(categoryUpdates) > 0 {
		sm.updateCategorySitemap()
	}
	
	if len(tagUpdates) > 0 {
		sm.updateTagSitemap()
	}
	
	// Always update the main sitemap index
	sm.updateSitemapIndex()
	
	sm.lastUpdate = time.Now()
}

// NotifyUpdate sends an update notification to the sitemap manager
func (sm *SitemapManager) NotifyUpdate(updateType, action string, entityID uint64, data interface{}) {
	if !sm.isRunning {
		return
	}
	
	event := SitemapUpdateEvent{
		Type:      updateType,
		Action:    action,
		EntityID:  entityID,
		Data:      data,
		Timestamp: time.Now(),
	}
	
	// Non-blocking send
	select {
	case sm.updateChannel <- event:
		// Update sent successfully
	default:
		// Channel full, skip this update (could log this)
	}
}

// updateArticleSitemaps updates all article-related sitemaps
func (sm *SitemapManager) updateArticleSitemaps() {
	// Clear article sitemap cache
	sm.cacheService.DeletePattern("sitemap-articles-*")
	
	// The actual sitemap will be regenerated on next request
	// This approach is more efficient for high-volume updates
}

// updateNewsSitemaps updates news sitemaps
func (sm *SitemapManager) updateNewsSitemaps() {
	// Clear news sitemap cache
	sm.cacheService.DeletePattern("sitemap-news-*")
}

// updateCategorySitemap updates category sitemap
func (sm *SitemapManager) updateCategorySitemap() {
	sm.cacheService.Delete("sitemap-categories")
}

// updateTagSitemap updates tag sitemap
func (sm *SitemapManager) updateTagSitemap() {
	sm.cacheService.Delete("sitemap-tags")
}

// updateSitemapIndex updates the main sitemap index
func (sm *SitemapManager) updateSitemapIndex() {
	sm.cacheService.Delete("sitemap-index")
}

// UpdateSitemaps regenerates all sitemaps when content changes (legacy method)
func (sm *SitemapManager) UpdateSitemaps(data *SitemapData) error {
	// Clear all sitemap caches for immediate regeneration
	sm.cacheService.DeletePattern("sitemap-*")
	sm.lastUpdate = time.Now()
	return nil
}

// GetSitemapIndex returns the main sitemap index with caching
func (sm *SitemapManager) GetSitemapIndex(data *SitemapData) (*SitemapIndex, error) {
	// Try to get from cache first
	cacheKey := "sitemap-index"
	if cached, err := sm.cacheService.Get(cacheKey); err == nil && len(cached) > 0 {
		var index SitemapIndex
		if err := xml.Unmarshal(cached, &index); err == nil {
			return &index, nil
		}
	}
	
	// Generate new sitemap index
	index, err := sm.sitemapService.GenerateSitemapIndex(data)
	if err != nil {
		return nil, err
	}
	
	// Cache the result
	if xmlData, err := xml.Marshal(index); err == nil {
		sm.cacheService.Set(cacheKey, xmlData, 1*time.Hour)
	}
	
	return index, nil
}

// GetCachedSitemap returns a cached sitemap or generates a new one
func (sm *SitemapManager) GetCachedSitemap(sitemapType string, generator func() (interface{}, error)) (string, error) {
	cacheKey := fmt.Sprintf("sitemap-%s", sitemapType)
	
	// Try cache first
	if cached, err := sm.cacheService.Get(cacheKey); err == nil && len(cached) > 0 {
		return string(cached), nil
	}
	
	// Generate new sitemap
	sitemap, err := generator()
	if err != nil {
		return "", err
	}
	
	// Render to XML
	xml, err := sm.sitemapService.RenderXML(sitemap)
	if err != nil {
		return "", err
	}
	
	// Cache the result
	sm.cacheService.Set(cacheKey, []byte(xml), 1*time.Hour)
	
	return xml, nil
}