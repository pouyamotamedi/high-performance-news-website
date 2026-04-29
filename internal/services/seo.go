package services

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/url"
	"strings"
	"time"

	"high-performance-news-website/internal/models"
)

// SEOService handles all SEO-related functionality
type SEOService struct {
	baseURL    string
	siteName   string
	defaultLang string
}

// NewSEOService creates a new SEO service instance
func NewSEOService(baseURL, siteName, defaultLang string) *SEOService {
	return &SEOService{
		baseURL:     strings.TrimSuffix(baseURL, "/"),
		siteName:    siteName,
		defaultLang: defaultLang,
	}
}

// SchemaMarkup represents structured data for SEO
type SchemaMarkup struct {
	Context     string      `json:"@context"`
	Type        string      `json:"@type"`
	Headline    string      `json:"headline,omitempty"`
	Description string      `json:"description,omitempty"`
	Image       interface{} `json:"image,omitempty"`
	Author      interface{} `json:"author,omitempty"`
	Publisher   interface{} `json:"publisher,omitempty"`
	DatePublished string    `json:"datePublished,omitempty"`
	DateModified  string    `json:"dateModified,omitempty"`
	URL         string      `json:"url,omitempty"`
	MainEntityOfPage interface{} `json:"mainEntityOfPage,omitempty"`
	ArticleSection string    `json:"articleSection,omitempty"`
	Keywords    []string    `json:"keywords,omitempty"`
	WordCount   int         `json:"wordCount,omitempty"`
	TimeRequired string     `json:"timeRequired,omitempty"`
}

// MetaTags represents HTML meta tags for SEO
type MetaTags struct {
	Title              string
	Description        string
	Keywords           string
	CanonicalURL       string
	OGTitle            string
	OGDescription      string
	OGImage            string
	OGType             string
	OGURL              string
	OGSiteName         string
	OGPublishedTime    string
	OGModifiedTime     string
	OGAuthor           string
	TwitterCard        string
	TwitterTitle       string
	TwitterDescription string
	TwitterImage       string
	TwitterSite        string
	TwitterLabel1      string
	TwitterData1       string
	TwitterLabel2      string
	TwitterData2       string
	Language           string
	Robots             string
	Author             string
}

// BreadcrumbItemLD represents a single breadcrumb item for JSON-LD
type BreadcrumbItemLD struct {
	Name string `json:"name"`
	URL  string `json:"item"`
}

// BreadcrumbSchemaLD represents breadcrumb structured data for JSON-LD
type BreadcrumbSchemaLD struct {
	Context         string             `json:"@context"`
	Type            string             `json:"@type"`
	ItemListElement []BreadcrumbItemLD `json:"itemListElement"`
}

// GenerateArticleSchema creates NewsArticle schema markup for an article
func (s *SEOService) GenerateArticleSchema(article *models.Article, author *models.User, category *models.Category) (*SchemaMarkup, error) {
	if article == nil {
		return nil, fmt.Errorf("article cannot be nil")
	}

	schemaType := "NewsArticle"
	if article.SEOData.SchemaType != "" {
		schemaType = article.SEOData.SchemaType
	}

	schema := &SchemaMarkup{
		Context:     "https://schema.org",
		Type:        schemaType,
		Headline:    article.Title,
		Description: article.Excerpt,
		URL:         s.GetArticleURL(article.Slug),
		WordCount:   s.countWords(article.Content),
	}

	// Add publication dates
	if article.PublishedAt != nil {
		schema.DatePublished = article.PublishedAt.Format(time.RFC3339)
	}
	schema.DateModified = article.UpdatedAt.Format(time.RFC3339)

	// Add author information with enhanced data
	if author != nil {
		authorSchema := map[string]interface{}{
			"@type": "Person",
			"name":  fmt.Sprintf("%s %s", author.FirstName, author.LastName),
			"url":   s.GetAuthorURL(author.Username),
		}
		
		// Add author image if available
		if author.Avatar != "" {
			authorSchema["image"] = map[string]interface{}{
				"@type": "ImageObject",
				"url":   author.Avatar,
			}
		}
		
		// Add author bio if available
		if author.Bio != "" {
			authorSchema["description"] = author.Bio
		}
		
		schema.Author = authorSchema
	}

	// Enhanced publisher information
	publisherSchema := map[string]interface{}{
		"@type": "Organization",
		"name":  s.siteName,
		"url":   s.baseURL,
		"logo": map[string]interface{}{
			"@type":  "ImageObject",
			"url":    s.baseURL + "/static/images/logo.png",
			"width":  "600",
			"height": "60",
		},
	}
	
	// Add social media profiles if available
	socialProfiles := []string{
		s.baseURL + "/social/facebook",
		s.baseURL + "/social/twitter",
		s.baseURL + "/social/linkedin",
	}
	publisherSchema["sameAs"] = socialProfiles
	
	schema.Publisher = publisherSchema

	// Enhanced main entity
	schema.MainEntityOfPage = map[string]interface{}{
		"@type": "WebPage",
		"@id":   schema.URL,
		"name":  article.Title,
	}

	// Add category/section with breadcrumb
	if category != nil {
		schema.ArticleSection = category.Name
		
		// Add category as about property
		schema.MainEntityOfPage.(map[string]interface{})["about"] = map[string]interface{}{
			"@type": "Thing",
			"name":  category.Name,
			"url":   s.GetCategoryURL(category.Slug),
		}
	}

	// Enhanced keywords handling
	var allKeywords []string
	if len(article.SEOData.Keywords) > 0 {
		allKeywords = append(allKeywords, article.SEOData.Keywords...)
	}
	
	// Add tag names as keywords
	for _, tag := range article.Tags {
		allKeywords = append(allKeywords, tag.Name)
		// Also add tag keywords if available
		allKeywords = append(allKeywords, tag.Keywords...)
	}
	
	// Remove duplicates and limit to 10 keywords
	schema.Keywords = s.deduplicateKeywords(allKeywords, 10)

	// Calculate reading time
	readingTime := s.calculateReadingTime(article.Content)
	if readingTime > 0 {
		schema.TimeRequired = fmt.Sprintf("PT%dM", readingTime)
	}

	// Add article image if available
	articleImage := s.GetArticleImageURL(article.Slug)
	if articleImage != "" {
		schema.Image = map[string]interface{}{
			"@type":  "ImageObject",
			"url":    articleImage,
			"width":  "1200",
			"height": "630",
		}
	}

	// Add language
	if article.LanguageCode != "" {
		schema.MainEntityOfPage.(map[string]interface{})["inLanguage"] = article.LanguageCode
	}

	return schema, nil
}

// GenerateHomepageSchema creates WebSite schema markup for homepage
func (s *SEOService) GenerateHomepageSchema() (*SchemaMarkup, error) {
	schema := &SchemaMarkup{
		Context:     "https://schema.org",
		Type:        "WebSite",
		URL:         s.baseURL,
		Headline:    s.siteName,
		Description: fmt.Sprintf("%s - High Performance News Website", s.siteName),
	}

	// Add search action
	searchAction := map[string]interface{}{
		"@type":       "SearchAction",
		"target":      s.baseURL + "/search?q={search_term_string}",
		"query-input": "required name=search_term_string",
	}

	schema.MainEntityOfPage = map[string]interface{}{
		"@type":          "WebPage",
		"@id":            s.baseURL,
		"potentialAction": searchAction,
	}

	return schema, nil
}

// GenerateCategorySchema creates CollectionPage schema markup for category pages
func (s *SEOService) GenerateCategorySchema(category *models.Category) (*SchemaMarkup, error) {
	if category == nil {
		return nil, fmt.Errorf("category cannot be nil")
	}

	schema := &SchemaMarkup{
		Context:     "https://schema.org",
		Type:        "CollectionPage",
		Headline:    category.Name,
		Description: category.Description,
		URL:         s.GetCategoryURL(category.Slug),
	}

	schema.MainEntityOfPage = map[string]interface{}{
		"@type": "WebPage",
		"@id":   schema.URL,
	}

	return schema, nil
}

// GenerateTagSchema creates CollectionPage schema markup for tag pages
func (s *SEOService) GenerateTagSchema(tag *models.Tag) (*SchemaMarkup, error) {
	if tag == nil {
		return nil, fmt.Errorf("tag cannot be nil")
	}

	schema := &SchemaMarkup{
		Context:     "https://schema.org",
		Type:        "CollectionPage",
		Headline:    tag.Name,
		Description: tag.Description,
		URL:         s.GetTagURL(tag.Slug),
	}

	if len(tag.Keywords) > 0 {
		schema.Keywords = tag.Keywords
	}

	schema.MainEntityOfPage = map[string]interface{}{
		"@type": "WebPage",
		"@id":   schema.URL,
	}

	return schema, nil
}

// GenerateMetaTags creates comprehensive meta tags for SEO
func (s *SEOService) GenerateMetaTags(pageType string, data interface{}) (*MetaTags, error) {
	meta := &MetaTags{
		Language:    s.defaultLang,
		Robots:      "index, follow",
		TwitterCard: "summary_large_image",
		OGType:      "website",
	}

	switch pageType {
	case "article":
		article, ok := data.(*models.Article)
		if !ok {
			return nil, fmt.Errorf("invalid data type for article meta tags")
		}
		return s.generateArticleMetaTags(article, meta)

	case "category":
		category, ok := data.(*models.Category)
		if !ok {
			return nil, fmt.Errorf("invalid data type for category meta tags")
		}
		return s.generateCategoryMetaTags(category, meta)

	case "tag":
		tag, ok := data.(*models.Tag)
		if !ok {
			return nil, fmt.Errorf("invalid data type for tag meta tags")
		}
		return s.generateTagMetaTags(tag, meta)

	case "homepage":
		return s.generateHomepageMetaTags(meta)

	default:
		return nil, fmt.Errorf("unsupported page type: %s", pageType)
	}
}

// generateArticleMetaTags creates meta tags for article pages
func (s *SEOService) generateArticleMetaTags(article *models.Article, meta *MetaTags) (*MetaTags, error) {
	// Enhanced title optimization with length checking
	if article.SEOData.MetaTitle != "" {
		meta.Title = article.SEOData.MetaTitle
	} else {
		// Optimize title length for SEO (under 60 characters)
		baseTitle := fmt.Sprintf("%s - %s", article.Title, s.siteName)
		if len(baseTitle) > 60 {
			// Try without site name first
			if len(article.Title) <= 57 { // Leave room for " - "
				meta.Title = article.Title
			} else {
				// Truncate article title
				truncated := s.truncateText(article.Title, 57)
				meta.Title = truncated
			}
		} else {
			meta.Title = baseTitle
		}
	}

	// Enhanced description optimization
	if article.SEOData.MetaDescription != "" {
		meta.Description = article.SEOData.MetaDescription
	} else if article.Excerpt != "" {
		// Ensure excerpt is within optimal length (150-160 characters)
		if len(article.Excerpt) > 160 {
			meta.Description = s.truncateText(article.Excerpt, 157) // Leave room for "..."
		} else {
			meta.Description = article.Excerpt
		}
	} else {
		// Generate description from content
		stripped := s.stripHTML(article.Content)
		meta.Description = s.truncateText(stripped, 157)
	}

	// Enhanced keywords with deduplication
	var allKeywords []string
	if len(article.SEOData.Keywords) > 0 {
		allKeywords = append(allKeywords, article.SEOData.Keywords...)
	}
	
	// Add tag names and their keywords
	for _, tag := range article.Tags {
		allKeywords = append(allKeywords, tag.Name)
		allKeywords = append(allKeywords, tag.Keywords...)
	}
	
	// Deduplicate and limit keywords
	uniqueKeywords := s.deduplicateKeywords(allKeywords, 10)
	if len(uniqueKeywords) > 0 {
		meta.Keywords = strings.Join(uniqueKeywords, ", ")
	}

	// Canonical URL with fallback
	if article.SEOData.CanonicalURL != "" {
		meta.CanonicalURL = article.SEOData.CanonicalURL
	} else {
		meta.CanonicalURL = s.GetArticleURL(article.Slug)
	}

	// Enhanced Open Graph tags
	meta.OGTitle = meta.Title
	meta.OGDescription = meta.Description
	meta.OGType = "article"
	meta.OGURL = meta.CanonicalURL
	meta.OGImage = s.GetArticleImageURL(article.Slug)
	
	// Add additional Open Graph article properties
	meta.OGSiteName = s.siteName
	if article.PublishedAt != nil {
		meta.OGPublishedTime = article.PublishedAt.Format(time.RFC3339)
	}
	meta.OGModifiedTime = article.UpdatedAt.Format(time.RFC3339)
	
	// Add article author for Open Graph
	meta.OGAuthor = s.GetAuthorURL("author") // This would be dynamic in real implementation

	// Enhanced Twitter Card tags
	meta.TwitterCard = "summary_large_image"
	meta.TwitterTitle = meta.Title
	meta.TwitterDescription = meta.Description
	meta.TwitterImage = meta.OGImage
	meta.TwitterSite = "@" + s.siteName // This would be configurable
	
	// Add Twitter-specific properties
	if article.PublishedAt != nil {
		meta.TwitterLabel1 = "Published"
		meta.TwitterData1 = article.PublishedAt.Format("Jan 2, 2006")
	}
	
	readingTime := s.calculateReadingTime(article.Content)
	if readingTime > 0 {
		meta.TwitterLabel2 = "Reading time"
		meta.TwitterData2 = fmt.Sprintf("%d min", readingTime)
	}

	// Language and additional properties
	meta.Language = article.LanguageCode
	meta.Author = "Author Name" // This would be dynamic in real implementation
	
	// Add article-specific robots directive
	if article.Status == "published" {
		meta.Robots = "index, follow, max-snippet:-1, max-image-preview:large, max-video-preview:-1"
	} else {
		meta.Robots = "noindex, nofollow"
	}

	return meta, nil
}

// generateCategoryMetaTags creates meta tags for category pages
func (s *SEOService) generateCategoryMetaTags(category *models.Category, meta *MetaTags) (*MetaTags, error) {
	meta.Title = fmt.Sprintf("%s - %s", category.Name, s.siteName)
	
	if category.Description != "" {
		meta.Description = category.Description
	} else {
		meta.Description = fmt.Sprintf("Browse articles in the %s category on %s", category.Name, s.siteName)
	}

	meta.CanonicalURL = s.GetCategoryURL(category.Slug)
	meta.OGTitle = meta.Title
	meta.OGDescription = meta.Description
	meta.OGURL = meta.CanonicalURL
	meta.TwitterTitle = meta.Title
	meta.TwitterDescription = meta.Description
	meta.Language = category.LanguageCode

	return meta, nil
}

// generateTagMetaTags creates meta tags for tag pages
func (s *SEOService) generateTagMetaTags(tag *models.Tag, meta *MetaTags) (*MetaTags, error) {
	meta.Title = fmt.Sprintf("%s - %s", tag.Name, s.siteName)
	
	if tag.Description != "" {
		meta.Description = tag.Description
	} else {
		meta.Description = fmt.Sprintf("Articles tagged with %s on %s", tag.Name, s.siteName)
	}

	if len(tag.Keywords) > 0 {
		meta.Keywords = strings.Join(tag.Keywords, ", ")
	}

	meta.CanonicalURL = s.GetTagURL(tag.Slug)
	meta.OGTitle = meta.Title
	meta.OGDescription = meta.Description
	meta.OGURL = meta.CanonicalURL
	meta.TwitterTitle = meta.Title
	meta.TwitterDescription = meta.Description
	meta.Language = tag.LanguageCode

	return meta, nil
}

// generateHomepageMetaTags creates meta tags for homepage
func (s *SEOService) generateHomepageMetaTags(meta *MetaTags) (*MetaTags, error) {
	meta.Title = fmt.Sprintf("%s - High Performance News Website", s.siteName)
	meta.Description = fmt.Sprintf("Stay updated with the latest news on %s. Fast, reliable, and comprehensive news coverage.", s.siteName)
	meta.CanonicalURL = s.baseURL
	meta.OGTitle = meta.Title
	meta.OGDescription = meta.Description
	meta.OGURL = meta.CanonicalURL
	meta.TwitterTitle = meta.Title
	meta.TwitterDescription = meta.Description

	return meta, nil
}

// GenerateBreadcrumbs creates breadcrumb navigation with schema markup
func (s *SEOService) GenerateBreadcrumbs(pageType string, data interface{}) (*BreadcrumbSchemaLD, error) {
	breadcrumbs := &BreadcrumbSchemaLD{
		Context: "https://schema.org",
		Type:    "BreadcrumbList",
	}

	// Always start with home
	items := []BreadcrumbItemLD{
		{Name: "Home", URL: s.baseURL},
	}

	switch pageType {
	case "article":
		article, ok := data.(*models.Article)
		if !ok {
			return nil, fmt.Errorf("invalid data type for article breadcrumbs")
		}
		
		// Add category if available
		if len(article.Tags) > 0 && article.CategoryID > 0 {
			// This would need category data - for now, add a placeholder
			items = append(items, BreadcrumbItemLD{
				Name: "Category", // Would be actual category name
				URL:  s.GetCategoryURL("category-slug"),
			})
		}
		
		// Add article
		items = append(items, BreadcrumbItemLD{
			Name: article.Title,
			URL:  s.GetArticleURL(article.Slug),
		})

	case "category":
		category, ok := data.(*models.Category)
		if !ok {
			return nil, fmt.Errorf("invalid data type for category breadcrumbs")
		}
		
		items = append(items, BreadcrumbItemLD{Name: "Categories", URL: s.baseURL + "/categories"})
		items = append(items, BreadcrumbItemLD{
			Name: category.Name,
			URL:  s.GetCategoryURL(category.Slug),
		})

	case "tag":
		tag, ok := data.(*models.Tag)
		if !ok {
			return nil, fmt.Errorf("invalid data type for tag breadcrumbs")
		}
		
		items = append(items, BreadcrumbItemLD{Name: "Tags", URL: s.baseURL + "/tags"})
		items = append(items, BreadcrumbItemLD{
			Name: tag.Name,
			URL:  s.GetTagURL(tag.Slug),
		})
	}

	breadcrumbs.ItemListElement = items
	return breadcrumbs, nil
}

// URL generation helpers
func (s *SEOService) GetArticleURL(slug string) string {
	return fmt.Sprintf("%s/article/%s", s.baseURL, slug)
}

func (s *SEOService) GetCategoryURL(slug string) string {
	return fmt.Sprintf("%s/category/%s", s.baseURL, slug)
}

func (s *SEOService) GetTagURL(slug string) string {
	return fmt.Sprintf("%s/tag/%s", s.baseURL, slug)
}

func (s *SEOService) GetAuthorURL(username string) string {
	return fmt.Sprintf("%s/author/%s", s.baseURL, username)
}

func (s *SEOService) GetArticleImageURL(slug string) string {
	return fmt.Sprintf("%s/static/images/articles/%s.jpg", s.baseURL, slug)
}

// Utility functions
func (s *SEOService) countWords(text string) int {
	words := strings.Fields(s.stripHTML(text))
	return len(words)
}

func (s *SEOService) calculateReadingTime(content string) int {
	wordCount := s.countWords(content)
	// Average reading speed: 200 words per minute
	return (wordCount + 199) / 200 // Round up
}

func (s *SEOService) stripHTML(html string) string {
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

func (s *SEOService) truncateText(text string, maxLength int) string {
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

// RenderSchemaJSON converts schema markup to JSON-LD string
func (s *SEOService) RenderSchemaJSON(schema interface{}) (template.HTML, error) {
	jsonBytes, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal schema: %w", err)
	}
	
	jsonLD := fmt.Sprintf(`<script type="application/ld+json">
%s
</script>`, string(jsonBytes))
	
	return template.HTML(jsonLD), nil
}

// deduplicateKeywords removes duplicate keywords and limits the count
func (s *SEOService) deduplicateKeywords(keywords []string, maxCount int) []string {
	seen := make(map[string]bool)
	var result []string
	
	for _, keyword := range keywords {
		normalized := strings.ToLower(strings.TrimSpace(keyword))
		if normalized == "" || seen[normalized] {
			continue
		}
		
		seen[normalized] = true
		result = append(result, keyword)
		
		if len(result) >= maxCount {
			break
		}
	}
	
	return result
}

// ValidateURL checks if a URL is valid
func (s *SEOService) ValidateURL(rawURL string) error {
	_, err := url.Parse(rawURL)
	return err
}