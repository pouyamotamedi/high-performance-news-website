package services

import (
	"context"
	"fmt"
	"html/template"
	"time"

	"high-performance-news-website/internal/models"
	"high-performance-news-website/internal/repositories"
)

// WidgetService handles widget business logic
type WidgetService struct {
	widgetRepo    *repositories.WidgetRepository
	articleRepo   *repositories.ArticleRepository
	categoryRepo  *repositories.CategoryRepository
	tagRepo       *repositories.TagRepository
	cacheService  CacheService
}

// NewWidgetService creates a new widget service
func NewWidgetService(
	widgetRepo *repositories.WidgetRepository,
	articleRepo *repositories.ArticleRepository,
	categoryRepo *repositories.CategoryRepository,
	tagRepo *repositories.TagRepository,
	cacheService CacheService,
) *WidgetService {
	return &WidgetService{
		widgetRepo:   widgetRepo,
		articleRepo:  articleRepo,
		categoryRepo: categoryRepo,
		tagRepo:      tagRepo,
		cacheService: cacheService,
	}
}

// CreateWidget creates a new widget
func (s *WidgetService) CreateWidget(widget *models.Widget) (*models.Widget, error) {
	if err := widget.Validate(); err != nil {
		return nil, err
	}

	// Set default sort order if not provided
	if widget.SortOrder == 0 {
		widget.SortOrder = 100
	}

	createdWidget, err := s.widgetRepo.Create(widget)
	if err != nil {
		return nil, fmt.Errorf("failed to create widget: %w", err)
	}

	// Clear widget cache
	s.clearWidgetCache()

	return createdWidget, nil
}

// GetWidget retrieves a widget by ID
func (s *WidgetService) GetWidget(id uint64) (*models.Widget, error) {
	return s.widgetRepo.GetByID(id)
}

// GetAllWidgets retrieves all widgets
func (s *WidgetService) GetAllWidgets() ([]*models.Widget, error) {
	return s.widgetRepo.GetAll()
}

// GetWidgetsByType retrieves widgets by type
func (s *WidgetService) GetWidgetsByType(widgetType models.WidgetType) ([]*models.Widget, error) {
	return s.widgetRepo.GetByType(widgetType)
}

// UpdateWidget updates a widget
func (s *WidgetService) UpdateWidget(widget *models.Widget) error {
	if err := widget.Validate(); err != nil {
		return err
	}

	if err := s.widgetRepo.Update(widget); err != nil {
		return fmt.Errorf("failed to update widget: %w", err)
	}

	// Clear widget cache
	s.clearWidgetCache()

	return nil
}

// DeleteWidget deletes a widget
func (s *WidgetService) DeleteWidget(id uint64) error {
	if err := s.widgetRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete widget: %w", err)
	}

	// Clear widget cache
	s.clearWidgetCache()

	return nil
}

// CreateWidgetPlacement creates a new widget placement
func (s *WidgetService) CreateWidgetPlacement(placement *models.WidgetPlacement) (*models.WidgetPlacement, error) {
	if err := placement.Validate(); err != nil {
		return nil, err
	}

	createdPlacement, err := s.widgetRepo.CreatePlacement(placement)
	if err != nil {
		return nil, fmt.Errorf("failed to create widget placement: %w", err)
	}

	// Clear placement cache
	s.clearPlacementCache(placement.PageType, placement.Zone)

	return createdPlacement, nil
}

// GetWidgetPlacements retrieves widget placements for a page and zone
func (s *WidgetService) GetWidgetPlacements(pageType models.PageType, zone models.WidgetZone) ([]*models.WidgetPlacement, error) {
	cacheKey := fmt.Sprintf("widget_placements:%s:%s", pageType, zone)
	
	// Try to get from cache first
	if cached, err := s.getCachedPlacements(cacheKey); err == nil && cached != nil {
		return cached, nil
	}

	placements, err := s.widgetRepo.GetPlacementsByPage(pageType, zone)
	if err != nil {
		return nil, fmt.Errorf("failed to get widget placements: %w", err)
	}

	// Cache the result
	s.cachePlacements(cacheKey, placements, 15*time.Minute)

	return placements, nil
}

// UpdateWidgetPlacement updates a widget placement
func (s *WidgetService) UpdateWidgetPlacement(placement *models.WidgetPlacement) error {
	if err := placement.Validate(); err != nil {
		return err
	}

	if err := s.widgetRepo.UpdatePlacement(placement); err != nil {
		return fmt.Errorf("failed to update widget placement: %w", err)
	}

	// Clear placement cache
	s.clearPlacementCache(placement.PageType, placement.Zone)

	return nil
}

// DeleteWidgetPlacement deletes a widget placement
func (s *WidgetService) DeleteWidgetPlacement(id uint64) error {
	if err := s.widgetRepo.DeletePlacement(id); err != nil {
		return fmt.Errorf("failed to delete widget placement: %w", err)
	}

	// Clear all placement cache (we don't know which page/zone it was)
	s.clearAllPlacementCache()

	return nil
}

// UpdatePlacementPositions updates the positions of multiple widget placements
func (s *WidgetService) UpdatePlacementPositions(placements []*models.WidgetPlacement) error {
	for _, placement := range placements {
		if err := placement.Validate(); err != nil {
			return err
		}
	}

	if err := s.widgetRepo.UpdatePlacementPositions(placements); err != nil {
		return fmt.Errorf("failed to update placement positions: %w", err)
	}

	// Clear all placement cache
	s.clearAllPlacementCache()

	return nil
}

// RenderWidget renders a widget's content
func (s *WidgetService) RenderWidget(widget *models.Widget) (template.HTML, error) {
	config, err := widget.GetConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get widget config: %w", err)
	}

	switch widget.Type {
	case models.WidgetTypeLatestArticles:
		return s.renderLatestArticlesWidget(widget, config)
	case models.WidgetTypePopularArticles:
		return s.renderPopularArticlesWidget(widget, config)
	case models.WidgetTypeTrendingArticles:
		return s.renderTrendingArticlesWidget(widget, config)
	case models.WidgetTypeCategories:
		return s.renderCategoriesWidget(widget, config)
	case models.WidgetTypeTags:
		return s.renderTagsWidget(widget, config)
	case models.WidgetTypeSearch:
		return s.renderSearchWidget(widget, config)
	case models.WidgetTypeNewsletter:
		return s.renderNewsletterWidget(widget, config)
	case models.WidgetTypeCustomHTML:
		return s.renderCustomHTMLWidget(widget, config)
	case models.WidgetTypeAdvertisement:
		return s.renderAdvertisementWidget(widget, config)
	case models.WidgetTypeSocialMedia:
		return s.renderSocialMediaWidget(widget, config)
	default:
		return "", fmt.Errorf("unsupported widget type: %s", widget.Type)
	}
}

// renderLatestArticlesWidget renders the latest articles widget
func (s *WidgetService) renderLatestArticlesWidget(widget *models.Widget, config *models.WidgetConfig) (template.HTML, error) {
	count := config.ArticleCount
	if count == 0 {
		count = 5
	}

	articles, err := s.articleRepo.GetLatestArticles(context.Background(), count)
	if err != nil {
		return "", fmt.Errorf("failed to get latest articles: %w", err)
	}

	html := fmt.Sprintf(`<div class="widget widget-latest-articles" id="widget-%d">`, widget.ID)
	if widget.Title != "" {
		html += fmt.Sprintf(`<h3 class="widget-title">%s</h3>`, template.HTMLEscapeString(widget.Title))
	}
	html += `<div class="widget-content"><ul class="article-list">`

	for _, article := range articles {
		html += fmt.Sprintf(`<li class="article-item">`)
		// Use article language, default to English
		articleLang := article.LanguageCode
		if articleLang == "" {
			articleLang = "en"
		}
		html += fmt.Sprintf(`<a href="/%s/article/%s" class="article-link">`, articleLang, article.Slug)
		

		
		html += fmt.Sprintf(`<h4 class="article-title">%s</h4>`, template.HTMLEscapeString(article.Title))
		
		if config.ShowExcerpt && article.Excerpt != "" {
			html += fmt.Sprintf(`<p class="article-excerpt">%s</p>`, template.HTMLEscapeString(article.Excerpt))
		}
		
		if config.ShowDate {
			html += fmt.Sprintf(`<time class="article-date">%s</time>`, article.PublishedAt.Format("Jan 2, 2006"))
		}
		
		html += `</a></li>`
	}

	html += `</ul></div></div>`
	return template.HTML(html), nil
}

// renderPopularArticlesWidget renders the popular articles widget
func (s *WidgetService) renderPopularArticlesWidget(widget *models.Widget, config *models.WidgetConfig) (template.HTML, error) {
	count := config.ArticleCount
	if count == 0 {
		count = 5
	}

	articles, err := s.articleRepo.GetTrendingArticles(context.Background(), count, 24)
	if err != nil {
		return "", fmt.Errorf("failed to get popular articles: %w", err)
	}

	html := fmt.Sprintf(`<div class="widget widget-popular-articles" id="widget-%d">`, widget.ID)
	if widget.Title != "" {
		html += fmt.Sprintf(`<h3 class="widget-title">%s</h3>`, template.HTMLEscapeString(widget.Title))
	}
	html += `<div class="widget-content"><ol class="article-list">`

	for _, article := range articles {
		html += fmt.Sprintf(`<li class="article-item">`)
		// Use article language, default to English
		articleLang := article.LanguageCode
		if articleLang == "" {
			articleLang = "en"
		}
		html += fmt.Sprintf(`<a href="/%s/article/%s" class="article-link">`, articleLang, article.Slug)
		html += fmt.Sprintf(`<h4 class="article-title">%s</h4>`, template.HTMLEscapeString(article.Title))
		html += fmt.Sprintf(`<span class="article-views">%d views</span>`, article.ViewCount)
		html += `</a></li>`
	}

	html += `</ol></div></div>`
	return template.HTML(html), nil
}

// renderTrendingArticlesWidget renders the trending articles widget
func (s *WidgetService) renderTrendingArticlesWidget(widget *models.Widget, config *models.WidgetConfig) (template.HTML, error) {
	count := config.ArticleCount
	if count == 0 {
		count = 5
	}

	articles, err := s.articleRepo.GetTrendingArticles(context.Background(), count, 24)
	if err != nil {
		return "", fmt.Errorf("failed to get trending articles: %w", err)
	}

	html := fmt.Sprintf(`<div class="widget widget-trending-articles" id="widget-%d">`, widget.ID)
	if widget.Title != "" {
		html += fmt.Sprintf(`<h3 class="widget-title">%s</h3>`, template.HTMLEscapeString(widget.Title))
	}
	html += `<div class="widget-content"><ul class="article-list">`

	for _, article := range articles {
		html += fmt.Sprintf(`<li class="article-item">`)
		// Use article language, default to English
		articleLang := article.LanguageCode
		if articleLang == "" {
			articleLang = "en"
		}
		html += fmt.Sprintf(`<a href="/%s/article/%s" class="article-link">`, articleLang, article.Slug)
		html += fmt.Sprintf(`<h4 class="article-title">%s</h4>`, template.HTMLEscapeString(article.Title))
		html += `</a></li>`
	}

	html += `</ul></div></div>`
	return template.HTML(html), nil
}

// renderCategoriesWidget renders the categories widget
func (s *WidgetService) renderCategoriesWidget(widget *models.Widget, config *models.WidgetConfig) (template.HTML, error) {
	categories, err := s.categoryRepo.GetAll()
	if err != nil {
		return "", fmt.Errorf("failed to get categories: %w", err)
	}

	html := fmt.Sprintf(`<div class="widget widget-categories" id="widget-%d">`, widget.ID)
	if widget.Title != "" {
		html += fmt.Sprintf(`<h3 class="widget-title">%s</h3>`, template.HTMLEscapeString(widget.Title))
	}
	html += `<div class="widget-content"><ul class="category-list">`

	for _, category := range categories {
		html += fmt.Sprintf(`<li class="category-item">`)
		html += fmt.Sprintf(`<a href="/categories/%s" class="category-link">%s</a>`, 
			category.Slug, template.HTMLEscapeString(category.Name))
		html += `</li>`
	}

	html += `</ul></div></div>`
	return template.HTML(html), nil
}

// renderTagsWidget renders the tags widget
func (s *WidgetService) renderTagsWidget(widget *models.Widget, config *models.WidgetConfig) (template.HTML, error) {
	tags, err := s.tagRepo.GetAll()
	if err != nil {
		return "", fmt.Errorf("failed to get tags: %w", err)
	}

	html := fmt.Sprintf(`<div class="widget widget-tags" id="widget-%d">`, widget.ID)
	if widget.Title != "" {
		html += fmt.Sprintf(`<h3 class="widget-title">%s</h3>`, template.HTMLEscapeString(widget.Title))
	}
	html += `<div class="widget-content"><div class="tag-cloud">`

	for _, tag := range tags {
		html += fmt.Sprintf(`<a href="/tags/%s" class="tag-link">%s</a>`, 
			tag.Slug, template.HTMLEscapeString(tag.Name))
	}

	html += `</div></div></div>`
	return template.HTML(html), nil
}

// renderSearchWidget renders the search widget
func (s *WidgetService) renderSearchWidget(widget *models.Widget, config *models.WidgetConfig) (template.HTML, error) {
	html := fmt.Sprintf(`<div class="widget widget-search" id="widget-%d">`, widget.ID)
	if widget.Title != "" {
		html += fmt.Sprintf(`<h3 class="widget-title">%s</h3>`, template.HTMLEscapeString(widget.Title))
	}
	html += `<div class="widget-content">
		<form action="/search" method="GET" class="search-form">
			<input type="text" name="q" placeholder="Search articles..." class="search-input" required>
			<button type="submit" class="search-button">Search</button>
		</form>
	</div></div>`
	return template.HTML(html), nil
}

// renderNewsletterWidget renders the newsletter widget
func (s *WidgetService) renderNewsletterWidget(widget *models.Widget, config *models.WidgetConfig) (template.HTML, error) {
	html := fmt.Sprintf(`<div class="widget widget-newsletter" id="widget-%d">`, widget.ID)
	if widget.Title != "" {
		html += fmt.Sprintf(`<h3 class="widget-title">%s</h3>`, template.HTMLEscapeString(widget.Title))
	}
	html += `<div class="widget-content">
		<form action="/newsletter/subscribe" method="POST" class="newsletter-form">
			<input type="email" name="email" placeholder="Enter your email" class="newsletter-input" required>
			<button type="submit" class="newsletter-button">Subscribe</button>
		</form>
	</div></div>`
	return template.HTML(html), nil
}

// renderCustomHTMLWidget renders the custom HTML widget
func (s *WidgetService) renderCustomHTMLWidget(widget *models.Widget, config *models.WidgetConfig) (template.HTML, error) {
	html := fmt.Sprintf(`<div class="widget widget-custom-html" id="widget-%d">`, widget.ID)
	if widget.Title != "" {
		html += fmt.Sprintf(`<h3 class="widget-title">%s</h3>`, template.HTMLEscapeString(widget.Title))
	}
	html += `<div class="widget-content">`
	html += config.HTMLContent // Note: This should be sanitized before storage
	html += `</div></div>`
	return template.HTML(html), nil
}

// renderAdvertisementWidget renders the advertisement widget
func (s *WidgetService) renderAdvertisementWidget(widget *models.Widget, config *models.WidgetConfig) (template.HTML, error) {
	html := fmt.Sprintf(`<div class="widget widget-advertisement" id="widget-%d">`, widget.ID)
	if widget.Title != "" {
		html += fmt.Sprintf(`<h3 class="widget-title">%s</h3>`, template.HTMLEscapeString(widget.Title))
	}
	html += fmt.Sprintf(`<div class="widget-content">
		<div class="ad-slot" data-ad-slot="%s" data-ad-size="%s">
			<!-- Advertisement will be loaded here -->
		</div>
	</div></div>`, config.AdSlotID, config.AdSize)
	return template.HTML(html), nil
}

// renderSocialMediaWidget renders the social media widget
func (s *WidgetService) renderSocialMediaWidget(widget *models.Widget, config *models.WidgetConfig) (template.HTML, error) {
	html := fmt.Sprintf(`<div class="widget widget-social-media" id="widget-%d">`, widget.ID)
	if widget.Title != "" {
		html += fmt.Sprintf(`<h3 class="widget-title">%s</h3>`, template.HTMLEscapeString(widget.Title))
	}
	html += `<div class="widget-content"><div class="social-links">`

	for _, platform := range config.Platforms {
		html += fmt.Sprintf(`<a href="#" class="social-link social-%s" target="_blank" rel="noopener">%s</a>`, 
			platform, template.HTMLEscapeString(platform))
	}

	html += `</div></div></div>`
	return template.HTML(html), nil
}

// Helper methods for caching
func (s *WidgetService) clearWidgetCache() {
	if s.cacheService != nil {
		s.cacheService.DeletePattern("widgets:*")
	}
}

func (s *WidgetService) clearPlacementCache(pageType models.PageType, zone models.WidgetZone) {
	if s.cacheService != nil {
		cacheKey := fmt.Sprintf("widget_placements:%s:%s", pageType, zone)
		s.cacheService.Delete(cacheKey)
	}
}

func (s *WidgetService) clearAllPlacementCache() {
	if s.cacheService != nil {
		s.cacheService.DeletePattern("widget_placements:*")
	}
}

func (s *WidgetService) getCachedPlacements(cacheKey string) ([]*models.WidgetPlacement, error) {
	// Implementation would depend on your cache service
	// This is a placeholder
	return nil, fmt.Errorf("not implemented")
}

func (s *WidgetService) cachePlacements(cacheKey string, placements []*models.WidgetPlacement, ttl time.Duration) {
	// Implementation would depend on your cache service
	// This is a placeholder
}