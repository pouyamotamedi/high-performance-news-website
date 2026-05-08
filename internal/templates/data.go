package templates

import (
	"time"
)

// BaseTemplateData contains common data for all templates
type BaseTemplateData struct {
	// Site information
	SiteName        string `json:"site_name"`
	SiteDescription string `json:"site_description"`
	SiteURL         string `json:"site_url"`
	
	// Page metadata
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Keywords        []string `json:"keywords"`
	CanonicalURL    string   `json:"canonical_url"`
	
	// SEO metadata
	SEOTitle       string   `json:"seo_title"`
	SEODescription string   `json:"seo_description"`
	SEOKeywords    []string `json:"seo_keywords"`
	
	// Open Graph metadata
	OGTitle       string `json:"og_title"`
	OGDescription string `json:"og_description"`
	OGType        string `json:"og_type"`
	OGImage       string `json:"og_image"`
	OGURL         string `json:"og_url"`
	
	// Twitter Card metadata
	TwitterCard        string `json:"twitter_card"`
	TwitterTitle       string `json:"twitter_title"`
	TwitterDescription string `json:"twitter_description"`
	TwitterImage       string `json:"twitter_image"`
	
	// Language and localization
	LanguageCode      string         `json:"language_code"`
	LanguageDirection string         `json:"language_direction"`
	AlternateURLs     []AlternateURL `json:"alternate_urls"`
	
	// Navigation
	Navigation []NavigationItem `json:"navigation"`
	Breadcrumbs []BreadcrumbItem `json:"breadcrumbs"`
	
	// Theme and appearance
	ThemeMode string `json:"theme_mode"` // light, dark, auto
	
	// Structured data
	StructuredData string `json:"structured_data"`
	
	// Current time
	CurrentYear int       `json:"current_year"`
	CurrentTime time.Time `json:"current_time"`
	
	// User context
	IsAuthenticated bool        `json:"is_authenticated"`
	User            interface{} `json:"user,omitempty"`
	
	// Performance hints
	PreloadResources []PreloadResource `json:"preload_resources"`
	
	// PWA data
	PWAManifest string `json:"pwa_manifest"`
	
	// Analytics
	GoogleAnalyticsID string `json:"google_analytics_id,omitempty"`
	
	// Additional custom data
	CustomData map[string]interface{} `json:"custom_data,omitempty"`
}

// NavigationItem represents a navigation menu item
type NavigationItem struct {
	Name     string           `json:"name"`
	URL      string           `json:"url"`
	Icon     string           `json:"icon,omitempty"`
	Children []NavigationItem `json:"children,omitempty"`
	Active   bool             `json:"active"`
}

// BreadcrumbItem represents a breadcrumb navigation item
type BreadcrumbItem struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// AlternateURL represents an alternate language URL for hreflang tags
type AlternateURL struct {
	Lang string `json:"lang"`
	URL  string `json:"url"`
}

// PreloadResource represents a resource to preload
type PreloadResource struct {
	URL  string `json:"url"`
	Type string `json:"type"` // script, style, image, font
	As   string `json:"as,omitempty"`
}

// HomepageData contains data specific to the homepage
type HomepageData struct {
	BaseTemplateData
	
	// Featured content
	FeaturedArticles []Article `json:"featured_articles"`
	TrendingArticles []Article `json:"trending_articles"`
	LatestArticles   []Article `json:"latest_articles"`
	
	// Categories and tags
	Categories []Category `json:"categories"`
	PopularTags []Tag     `json:"popular_tags"`
	
	// Statistics
	TotalArticles int `json:"total_articles"`
	TotalViews    int `json:"total_views"`
	
	// Hero section
	HeroTitle    string `json:"hero_title"`
	HeroSubtitle string `json:"hero_subtitle"`
	HeroImage    string `json:"hero_image,omitempty"`
}

// ArticlePageData contains data specific to article pages
type ArticlePageData struct {
	BaseTemplateData
	
	// Article content
	Article Article `json:"article"`
	
	// Related content
	RelatedArticles []Article `json:"related_articles"`
	Category        Category  `json:"category"`
	Tags            []Tag     `json:"tags"`
	
	// Author information
	Author Author `json:"author"`
	
	// Comments
	Comments     []Comment `json:"comments"`
	CommentCount int       `json:"comment_count"`
	
	// Social sharing
	ShareURLs map[string]string `json:"share_urls"`
	
	// Reading time
	ReadingTime int `json:"reading_time"` // in minutes
}

// CategoryPageData contains data specific to category pages
type CategoryPageData struct {
	BaseTemplateData
	
	// Category information
	Category Category `json:"category"`
	
	// Articles in category
	Articles []Article `json:"articles"`
	
	// Pagination
	Pagination PaginationData `json:"pagination"`
	
	// Subcategories
	Subcategories []Category `json:"subcategories"`
	
	// Related categories
	RelatedCategories []Category `json:"related_categories"`
}

// TagPageData contains data specific to tag pages
type TagPageData struct {
	BaseTemplateData
	
	// Tag information
	Tag Tag `json:"tag"`
	
	// Articles with tag
	Articles []Article `json:"articles"`
	
	// Pagination
	Pagination PaginationData `json:"pagination"`
	
	// Related tags
	RelatedTags []Tag `json:"related_tags"`
}

// SearchPageData contains data specific to search results pages
type SearchPageData struct {
	BaseTemplateData
	
	// Search query
	Query string `json:"query"`
	
	// Search results
	Articles []Article `json:"articles"`
	
	// Search statistics
	TotalResults int           `json:"total_results"`
	SearchTime   time.Duration `json:"search_time"`
	
	// Pagination
	Pagination PaginationData `json:"pagination"`
	
	// Search suggestions
	Suggestions []string `json:"suggestions"`
	
	// Filters
	Filters SearchFilters `json:"filters"`
}

// PaginationData contains pagination information
type PaginationData struct {
	CurrentPage  int  `json:"current_page"`
	TotalPages   int  `json:"total_pages"`
	TotalItems   int  `json:"total_items"`
	ItemsPerPage int  `json:"items_per_page"`
	HasPrevious  bool `json:"has_previous"`
	HasNext      bool `json:"has_next"`
	PreviousURL  string `json:"previous_url,omitempty"`
	NextURL      string `json:"next_url,omitempty"`
}

// SearchFilters contains search filter options
type SearchFilters struct {
	Categories []Category `json:"categories"`
	Tags       []Tag      `json:"tags"`
	DateRange  DateRange  `json:"date_range"`
	SortBy     string     `json:"sort_by"`
	SortOrder  string     `json:"sort_order"`
}

// DateRange represents a date range filter
type DateRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// Article represents an article
type Article struct {
	ID           uint64    `json:"id"`
	Title        string    `json:"title"`
	Slug         string    `json:"slug"`
	Content      string    `json:"content"`
	Excerpt      string    `json:"excerpt"`
	AuthorID     uint64    `json:"author_id"`
	CategoryID   uint64    `json:"category_id"`
	Status       string    `json:"status"`
	PublishedAt  time.Time `json:"published_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	ViewCount    uint64    `json:"view_count"`
	LikeCount    uint64    `json:"like_count"`
	DislikeCount uint64    `json:"dislike_count"`
	
	// SEO data
	SEOTitle       string   `json:"seo_title,omitempty"`
	SEODescription string   `json:"seo_description,omitempty"`
	SEOKeywords    []string `json:"seo_keywords,omitempty"`
	CanonicalURL   string   `json:"canonical_url,omitempty"`
	
	// Featured image
	FeaturedImage string `json:"featured_image,omitempty"`
	ImageAlt      string `json:"image_alt,omitempty"`
	
	// Tags and category (populated via joins)
	Tags     []Tag    `json:"tags,omitempty"`
	Category Category `json:"category,omitempty"`
	Author   Author   `json:"author,omitempty"`
}

// Category represents a content category
type Category struct {
	ID          uint64    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	ParentID    *uint64   `json:"parent_id,omitempty"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	
	// SEO data
	SEOTitle       string   `json:"seo_title,omitempty"`
	SEODescription string   `json:"seo_description,omitempty"`
	SEOKeywords    []string `json:"seo_keywords,omitempty"`
	
	// Statistics
	ArticleCount int `json:"article_count"`
	
	// Hierarchy
	Parent   *Category  `json:"parent,omitempty"`
	Children []Category `json:"children,omitempty"`
}

// Tag represents a content tag
type Tag struct {
	ID          uint64    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Keywords    []string  `json:"keywords,omitempty"`
	Color       string    `json:"color"`
	CreatedAt   time.Time `json:"created_at"`
	
	// SEO data
	SEOTitle       string   `json:"seo_title,omitempty"`
	SEODescription string   `json:"seo_description,omitempty"`
	SEOKeywords    []string `json:"seo_keywords,omitempty"`
	
	// Statistics
	ArticleCount int `json:"article_count"`
}

// Author represents an article author
type Author struct {
	ID        uint64    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Bio       string    `json:"bio"`
	Avatar    string    `json:"avatar"`
	CreatedAt time.Time `json:"created_at"`
	
	// Statistics
	ArticleCount int `json:"article_count"`
	
	// Social links
	SocialLinks map[string]string `json:"social_links,omitempty"`
}

// Comment represents a user comment
type Comment struct {
	ID        uint64    `json:"id"`
	ArticleID uint64    `json:"article_id"`
	ParentID  *uint64   `json:"parent_id,omitempty"`
	AuthorID  *uint64   `json:"author_id,omitempty"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Content   string    `json:"content"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	
	// Nested comments
	Replies []Comment `json:"replies,omitempty"`
	
	// Author information (if registered user)
	Author *Author `json:"author,omitempty"`
}

// NewBaseTemplateData creates a new BaseTemplateData with default values
func NewBaseTemplateData() *BaseTemplateData {
	return &BaseTemplateData{
		CurrentYear:       time.Now().Year(),
		CurrentTime:       time.Now(),
		LanguageCode:      "en",
		LanguageDirection: "ltr",
		ThemeMode:         "auto",
		OGType:            "website",
		TwitterCard:       "summary_large_image",
		CustomData:        make(map[string]interface{}),
	}
}